package bootstrap

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/verikod/hector/pkg/app"
	"github.com/verikod/hector/pkg/auth"
	"github.com/verikod/hector/pkg/checkpoint"
	"github.com/verikod/hector/pkg/config"
	"github.com/verikod/hector/pkg/execution"
	"github.com/verikod/hector/pkg/execution/native"
	"github.com/verikod/hector/pkg/observability"
	"github.com/verikod/hector/pkg/rag"
	"github.com/verikod/hector/pkg/ratelimit"
	"github.com/verikod/hector/pkg/runtime"
	"github.com/verikod/hector/pkg/server"
	"github.com/verikod/hector/pkg/session"
	sigsvc "github.com/verikod/hector/pkg/signal"
	"github.com/verikod/hector/pkg/task"
	"github.com/verikod/hector/pkg/utils"
	"github.com/verikod/hector/pkg/vector"
	"github.com/verikod/hector/ui"
)

// BootstrapDependencies holds the dependencies available to the runtime factory.
type BootstrapDependencies struct {
	DBPool         *config.DBPool
	SessionService session.Service
	TaskService    task.Service
	VectorProvider vector.Provider
}

// RuntimeFactory creates a runtime for an app.
type RuntimeFactory func(ctx context.Context, deps BootstrapDependencies, appID string, appCfg *config.AppConfig) (server.Runtime, error)

// ServeOptions configuration for the Serve function.
type ServeOptions struct {
	// ServerConfig is the operational configuration (port, db, etc).
	ServerConfig *config.ServerConfig
	// ConfigPath is the path to the app config file (yaml).
	ConfigPath string
	// Watch enables config file watching for hot-reload.
	Watch bool
	// Sync forces the config file to overwrite the database config on startup.
	// Without this, the file only seeds the DB when the app doesn't exist yet.
	Sync bool
	// RuntimeFactory is the factory for creating runtimes.
	// If nil, defaults to standard runtime.NewBuilder().WithConfig(cfg).Build().
	RuntimeFactory RuntimeFactory
	// AutoSecret is set when the auth secret was auto-generated (not user-provided).
	// It is displayed prominently in the startup banner.
	AutoSecret string
}

// ServeOption is a functional option for Serve.
type ServeOption func(*ServeOptions)

// WithServerConfig sets the server configuration.
func WithServerConfig(cfg *config.ServerConfig) ServeOption {
	return func(o *ServeOptions) {
		o.ServerConfig = cfg
	}
}

// WithConfigPath sets the path to the app config file.
func WithConfigPath(path string) ServeOption {
	return func(o *ServeOptions) {
		o.ConfigPath = path
	}
}

// WithWatch enables config file watching.
func WithWatch(watch bool) ServeOption {
	return func(o *ServeOptions) {
		o.Watch = watch
	}
}

// WithSync forces the config file to overwrite the database config on startup.
func WithSync(sync bool) ServeOption {
	return func(o *ServeOptions) {
		o.Sync = sync
	}
}

// WithAutoSecret sets an auto-generated secret to display prominently in the startup banner.
func WithAutoSecret(secret string) ServeOption {
	return func(o *ServeOptions) {
		o.AutoSecret = secret
	}
}

// WithRuntimeFactory sets a custom runtime factory.
func WithRuntimeFactory(f RuntimeFactory) ServeOption {
	return func(o *ServeOptions) {
		o.RuntimeFactory = f
	}
}

// Serve starts the Hector server with the given options.
// It handles the entire lifecycle: config loading, app initialization, server start, signal handling.
func Serve(ctx context.Context, opts ...ServeOption) error {
	options := &ServeOptions{
		// Defaults
		ServerConfig: &config.ServerConfig{
			Host:      "0.0.0.0",
			Port:      8080,
			LogLevel:  "info",
			Database:  "sqlite://.hector/hector.db",
			LogFormat: "text",
		},
		ConfigPath: ".hector/config.yaml",
	}

	for _, opt := range opts {
		opt(options)
	}

	// Validate server config
	options.ServerConfig.SetDefaults()
	if err := options.ServerConfig.Validate(); err != nil {
		return fmt.Errorf("invalid server configuration: %w", err)
	}

	// Initialize logger
	if err := initLoggerFromConfig(options.ServerConfig); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	slog.Info("Starting Hector server",
		"database", options.ServerConfig.Database,
		"address", options.ServerConfig.Address())

	// Create signal manager for lifecycle management
	sigMgr := sigsvc.New(ctx)
	serverCtx := sigMgr.Start()

	// Load env vars FIRST — generator needs them for provider detection
	if err := config.LoadDotEnvForConfig(options.ConfigPath); err != nil {
		slog.Warn("Failed to load .env file (continuing without it)", "error", err)
	}

	// Ensure config exists (auto-create)
	result, err := config.EnsureConfigExists(config.CLIOptions{}, options.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to ensure config exists: %w", err)
	}
	if result.CreatedNew {
		slog.Info("Created minimal config file", "path", options.ConfigPath)
	}

	// Load app config (full with defaults + lean JSON for DB)
	loadResult, err := config.LoadAppConfigWithLean(options.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	appCfg := loadResult.Config

	// Initialize Observability
	obsCfg := &observability.Config{
		Metrics: observability.MetricsConfig{
			Enabled:  options.ServerConfig.MetricsEnabled,
			Endpoint: "/metrics",
		},
		Tracing: observability.TracingConfig{
			Enabled:  options.ServerConfig.TracingEndpoint != "",
			Exporter: "otlp",
			Endpoint: options.ServerConfig.TracingEndpoint,
		},
	}
	obsMgr, err := observability.NewManager(serverCtx, obsCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize observability: %w", err)
	}
	defer func() {
		if err := obsMgr.Shutdown(context.Background()); err != nil {
			slog.Warn("Failed to shutdown observability manager", "error", err)
		}
	}()

	// Initialize app state
	state, err := loadAppState(serverCtx, options.ServerConfig, appCfg, loadResult.LeanJSON, options.ConfigPath, options.RuntimeFactory, obsMgr, options.Sync)
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	defer state.Close()

	// Start HTTP server
	srv, err := startServer(serverCtx, options.ServerConfig, appCfg, state, sigMgr)
	if err != nil {
		return err
	}

	// Register reload callback
	sigMgr.OnReload(func() {
		// Reload .env
		if _, err := config.ReloadDotEnvForConfig(options.ConfigPath); err != nil {
			slog.Warn("Failed to reload .env file", "error", err)
		} else {
			slog.Info("Reloaded .env file")
		}

		// Update log level
		newLevel := os.Getenv("HECTOR_LOG_LEVEL")
		if newLevel != "" {
			options.ServerConfig.LogLevel = newLevel
			if err := initLoggerFromConfig(options.ServerConfig); err != nil {
				slog.Warn("Failed to update logger", "error", err)
			} else {
				slog.Info("Updated log level", "level", newLevel)
			}
		}

		// Reload app config
		newCfg, err := reloadDefaultApp(context.Background(), options.ConfigPath, state)
		if err != nil {
			slog.Error("Failed to reload app configuration", "error", err)
		} else {
			// Update server state with new config
			// This invokes the clean Reload API which handles route regeneration.
			srv.Reload(newCfg, state.appManager)
			slog.Info("Updated server routes with new configuration")
		}
	})

	// Setup hot-reload
	if options.Watch {
		go watchConfigFile(serverCtx, options.ConfigPath, sigMgr)
	}

	// Print startup info
	autoSecret := options.AutoSecret
	printServerInfo(options.ServerConfig, appCfg, options.ConfigPath, autoSecret)

	// Block until shutdown
	return srv.Start(serverCtx)
}

// appState holds the runtime state.
type appState struct {
	dbPool         *config.DBPool
	sessionSvc     session.Service
	taskStore      a2asrv.TaskStore
	taskService    task.Service
	taskQueue      native.Queue
	execProvider   execution.Provider
	appStore       app.Store
	appManager     *server.AppManager
	vectorProvider vector.Provider
	authValidator  auth.TokenValidator
	tokenIssuer    *auth.TokenIssuer
	rateLimiter    ratelimit.RateLimiter
	obsMgr         *observability.Manager
}

func (s *appState) Close() {
	// Stop execution provider first
	if s.execProvider != nil {
		if err := s.execProvider.Stop(); err != nil {
			slog.Warn("Failed to stop execution provider gracefully", "error", err)
		}
	}
	// Queue is closed by provider.Stop() usually, but strict check?
	// native.Provider.Stop() closes the queue.

	if s.appManager != nil {
		s.appManager.Close()
	}
	if closer, ok := s.sessionSvc.(io.Closer); ok {
		closer.Close()
	}
	if closer, ok := s.authValidator.(io.Closer); ok {
		closer.Close()
	}
	// Rate limiter cleanup:
	// - SQL store: uses shared dbPool connection, closed when dbPool closes (no explicit cleanup needed)
	// - Memory store: no cleanup needed (in-memory)
	// Note: SQLStore.Close() is a no-op as it doesn't own the DB connection
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if closer, ok := s.vectorProvider.(io.Closer); ok {
		closer.Close()
	}
	if s.tokenIssuer != nil {
		s.tokenIssuer.Close()
	}
	// Note: We don't shutdown obsMgr here during reload because it's shared/lifecycled by Serve()
	// Only distinct app state components are closed.
}

// loadAppState initializes the application state.
func loadAppState(ctx context.Context, serverCfg *config.ServerConfig, appCfg *config.AppConfig, leanJSON []byte, configPath string, customFactory RuntimeFactory, obsMgr *observability.Manager, sync bool) (*appState, error) {
	state := &appState{
		obsMgr: obsMgr,
	}

	// DB Pool
	state.dbPool = config.NewDBPool()
	dbCfg, err := serverCfg.GetDatabaseConfig()
	if err != nil {
		state.Close()
		return nil, fmt.Errorf("failed to get database config: %w", err)
	}
	db, err := state.dbPool.Get(dbCfg)
	if err != nil {
		state.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Services
	state.sessionSvc, err = session.NewSessionService(state.dbPool, serverCfg.Database)
	if err != nil {
		state.Close()
		return nil, fmt.Errorf("failed to create session service: %w", err)
	}

	state.taskStore, err = task.NewTaskStore(state.dbPool, serverCfg.Database)
	if err != nil {
		state.Close()
		return nil, fmt.Errorf("failed to create task store: %w", err)
	}

	state.taskService, err = task.NewService(state.taskStore)
	if err != nil {
		state.Close()
		return nil, fmt.Errorf("failed to create task service: %w", err)
	}

	// Initialize task queue with retry policy from config
	retryPolicy := &native.RetryPolicy{
		MaxRetries:    serverCfg.Queue.MaxRetries,
		InitialDelay:  serverCfg.Queue.InitialDelay,
		MaxDelay:      serverCfg.Queue.MaxDelay,
		BackoffFactor: serverCfg.Queue.BackoffFactor,
	}

	state.appStore, err = app.NewSQLStore(db, dbCfg.DriverName())
	if err != nil {
		state.Close()
		return nil, fmt.Errorf("failed to create app store: %w", err)
	}

	state.vectorProvider, err = createDefaultVectorProvider()
	if err != nil {
		slog.Warn("Failed to create vector provider", "error", err)
	}

	// Prepare dependencies for factory
	deps := BootstrapDependencies{
		DBPool:         state.dbPool,
		SessionService: state.sessionSvc,
		TaskService:    state.taskService,
		VectorProvider: state.vectorProvider,
	}

	// Configure optional git source allowlist policy for document stores.
	rag.SetGitAllowedRepos(serverCfg.GitAllowedRepos)

	// Runtime Factory (Adapter)
	// We adapt the BootstrapRuntimeFactory to the server.RuntimeFactory expected by AppManager
	adapterFactory := func(ctx context.Context, appID string, appCfg *config.AppConfig) (server.Runtime, error) {
		if customFactory != nil {
			return customFactory(ctx, deps, appID, appCfg)
		}

		// Default behavior
		rt, err := runtime.NewBuilder().
			WithAppID(appID).
			WithConfig(appCfg).
			WithDBPool(state.dbPool).
			WithDatabaseDSN(serverCfg.Database).
			WithSessionService(state.sessionSvc).
			Build()
		if err != nil {
			return nil, fmt.Errorf("failed to create runtime: %w", err)
		}

		// Background services
		if len(appCfg.DocumentStores) > 0 {
			go func() {
				if err := rt.IndexDocumentStores(context.Background()); err != nil {
					slog.Warn("Failed to index document stores", "app", appID, "error", err)
				}
				if err := rt.StartDocumentStoreWatching(context.Background()); err != nil {
					slog.Warn("Failed to start document store watching", "app", appID, "error", err)
				}
			}()
		}

		if cpMgr := rt.CheckpointManager(); cpMgr != nil && cpMgr.IsEnabled() {
			cpMgr.SetResumeCallback(makeResumeCallback(state, appID))
			go func() {
				slog.Info("Checking for pending checkpoints", "app", appID)
				if err := cpMgr.RecoverOnStartup(context.Background(), appID); err != nil {
					slog.Warn("Checkpoint recovery failed", "app", appID, "error", err)
				}
			}()
		}

		rt.StartScheduler()
		return rt, nil
	}

	state.appManager = server.NewAppManager(state.appStore, adapterFactory, state.taskService, obsMgr.Metrics())

	// Sync config to DB ("default" app)
	// Store lean config (without defaults) so Studio shows only user-specified fields
	defaultApp := &app.App{
		ID:         "default",
		Name:       appNameFromPath(configPath),
		ConfigJSON: string(leanJSON),
	}

	exists, err := state.appStore.Exists(ctx, "default")
	if err != nil {
		state.Close()
		return nil, fmt.Errorf("failed to check app existence: %w", err)
	}

	if !exists {
		if _, err := state.appStore.Create(ctx, defaultApp); err != nil {
			state.Close()
			return nil, fmt.Errorf("failed to create default app: %w", err)
		}
		slog.Info("Created default app in database from config file", "id", "default")
	} else if sync {
		if err := state.appStore.Update(ctx, defaultApp); err != nil {
			state.Close()
			return nil, fmt.Errorf("failed to update default app: %w", err)
		}
		slog.Info("Synced default app config from file to database (--sync)", "id", "default")
	} else {
		slog.Info("Default app already exists in database, skipping file sync (use --sync to overwrite)", "id", "default")
	}

	// Preload default app
	if err := state.appManager.Preload(ctx, "default"); err != nil {
		state.Close()
		return nil, fmt.Errorf("failed to preload default app: %w", err)
	}

	// Auth Validator & Token Issuer
	if serverCfg.Auth != nil && serverCfg.Auth.IsEnabled() {
		// 1. Load/Generate Private Key for Token Signing
		// We use the foundational .hector directory to store certificates.
		// We try to derive the workspace root from the config path.
		workspaceRoot := filepath.Dir(configPath)
		if filepath.Base(workspaceRoot) == utils.DefaultHectorDir {
			workspaceRoot = filepath.Dir(workspaceRoot)
		}

		hectorDir, err := utils.EnsureHectorDir(workspaceRoot)
		if err != nil {
			state.Close()
			return nil, fmt.Errorf("failed to ensure hector directory: %w", err)
		}

		var privateKeyPEM []byte

		if serverCfg.Auth.SigningSeed != "" {
			slog.Info("Using deterministic Ed25519 signing key from seed")
			privateKeyPEM, err = auth.GenerateDeterministicEd25519KeyPEM(serverCfg.Auth.SigningSeed)
			if err != nil {
				state.Close()
				return nil, fmt.Errorf("failed to generate deterministic signing key: %w", err)
			}
		} else {
			slog.Info("Using file-based signing key (no seed provided)")
			keyPath := filepath.Join(hectorDir, "certs", "private.pem")
			privateKeyPEM, err = auth.LoadOrGenerateRSAPrivateKey(keyPath)
			if err != nil {
				state.Close()
				return nil, fmt.Errorf("failed to load private key: %w", err)
			}
		}

		// 2. Create Token Issuer (for App Tokens)
		// We need this for both issuing tokens AND validating them (RS256)
		tokenIssuer, err := auth.NewTokenIssuer(auth.TokenIssuerConfig{
			Issuer:        fmt.Sprintf("http://%s:%d", serverCfg.Host, serverCfg.Port),
			Audience:      "hector-api",
			PrivateKeyPEM: privateKeyPEM,
		})
		if err != nil {
			state.Close()
			return nil, fmt.Errorf("failed to create token issuer: %w", err)
		}
		state.tokenIssuer = tokenIssuer

		// 3. Create Config Validator (Admin Secret / OIDC)
		configValidator, err := auth.NewValidatorFromConfig(serverCfg.Auth)
		if err != nil {
			state.Close()
			return nil, fmt.Errorf("failed to create auth validator: %w", err)
		}

		// 3. Combine them
		// This ensures the server accepts BOTH Admin Tokens (HS256) AND App Tokens (RS256)
		state.authValidator = auth.NewCompositeValidator(configValidator, tokenIssuer)

		slog.Info("Authentication enabled",
			"jwks", serverCfg.Auth.JWKSURL != "",
			"secret", serverCfg.Auth.Secret != "",
			"internal_issuer", true)
	}

	// Rate Limiter
	if serverCfg.RateLimit != nil && serverCfg.RateLimit.IsEnabled() {
		limiter, err := ratelimit.NewRateLimiterFromConfig(
			serverCfg.RateLimit,
			state.dbPool,
			serverCfg.Database,
		)
		if err != nil {
			state.Close()
			return nil, fmt.Errorf("failed to create rate limiter: %w", err)
		}
		state.rateLimiter = limiter
		slog.Info("Rate limiting enabled",
			"scope", serverCfg.RateLimit.Scope,
			"limits", len(serverCfg.RateLimit.Limits))
	}

	// Initialize Execution Provider (Native)
	executorProvider := &appManagerExecutorProvider{appManager: state.appManager}
	nativeProvider, err := native.NewProvider(
		state.dbPool,
		serverCfg.Database,
		retryPolicy,
		executorProvider,
		state.taskService,
		obsMgr.Metrics(),
		native.WorkerPoolOptions{
			NumWorkers:     serverCfg.Queue.Workers,
			StaleThreshold: serverCfg.Queue.StaleThreshold,
		},
	)
	if err != nil {
		state.Close()
		return nil, fmt.Errorf("failed to create native execution provider: %w", err)
	}

	state.execProvider = nativeProvider
	state.taskQueue = nativeProvider.Queue() // We use the native queue for admin stats

	// Start Queue Metrics Exporter
	if obsMgr.Metrics() != nil {
		exporter := native.NewMetricsExporter(state.taskQueue, obsMgr.Metrics(), "default")
		go exporter.Start(ctx, 15*time.Second) // Export every 15s
		slog.Info("Queue metrics exporter started")
	}

	if err := state.execProvider.Start(ctx); err != nil {
		state.Close()
		return nil, fmt.Errorf("failed to start execution provider: %w", err)
	}
	slog.Info("Execution provider started", "type", "native", "workers", serverCfg.Queue.Workers)

	// Recover stale queue items on startup
	go func() {
		if err := state.taskQueue.RecoverStale(context.Background(), serverCfg.Queue.StaleThreshold); err != nil {
			slog.Warn("Failed to recover stale queue items on startup", "error", err)
		}
	}()

	return state, nil
}

func makeResumeCallback(state *appState, appName string) checkpoint.ResumeCallback {
	return func(ctx context.Context, cpState *checkpoint.State) error {
		targetApp := appName
		if targetApp == "" {
			targetApp = "default"
		}

		runtime, err := state.appManager.GetRuntime(ctx, targetApp, cpState.AgentName)
		if err != nil {
			return fmt.Errorf("no runtime found for checkpoint recovery: %w", err)
		}

		slog.Info("Resuming from checkpoint",
			"task_id", cpState.TaskID,
			"session_id", cpState.SessionID,
			"agent", cpState.AgentName)

		return runtime.Executor.ResumeFromCheckpoint(ctx, cpState)
	}
}

// appManagerExecutorProvider adapts AppManager to task.ExecutorProvider interface.
type appManagerExecutorProvider struct {
	appManager *server.AppManager
}

// GetExecutor returns an executor for the given app/agent combination.
func (p *appManagerExecutorProvider) GetExecutor(ctx context.Context, appID, agentName string) (a2asrv.AgentExecutor, error) {
	runtime, err := p.appManager.GetRuntime(ctx, appID, agentName)
	if err != nil {
		return nil, err
	}
	return runtime.Executor, nil
}

func startServer(ctx context.Context, serverCfg *config.ServerConfig, appCfg *config.AppConfig, state *appState, sigMgr *sigsvc.Manager) (*server.HTTPServer, error) {
	var serverOpts []server.HTTPServerOption

	if state.taskStore != nil {
		serverOpts = append(serverOpts, server.WithTaskStore(state.taskStore))
	}

	serverOpts = append(serverOpts, server.WithTaskService(state.taskService))
	serverOpts = append(serverOpts, server.WithSessionService(state.sessionSvc))

	// Add execution provider for async execution
	if state.execProvider != nil {
		serverOpts = append(serverOpts, server.WithExecutionProvider(state.execProvider))
	}

	if state.authValidator != nil {
		serverOpts = append(serverOpts, server.WithAuthValidator(state.authValidator))
	}

	if state.rateLimiter != nil {
		serverOpts = append(serverOpts, server.WithRateLimiter(state.rateLimiter))
	}

	serverOpts = append(serverOpts, server.WithDocumentStoreProvider(state.appManager))

	if state.obsMgr != nil {
		serverOpts = append(serverOpts, server.WithObservability(state.obsMgr))
	}

	// Admin API
	if serverCfg.Auth != nil && serverCfg.Auth.Secret != "" {
		// Ensure issuer was created in loadAppState
		if state.tokenIssuer == nil {
			return nil, fmt.Errorf("token issuer not initialized (check auth config)")
		}

		adminHandlerCfg := server.AdminHandlerConfig{
			AppStore:       state.appStore,
			TokenIssuer:    state.tokenIssuer,
			AppManager:     state.appManager,
			SessionService: state.sessionSvc,
			VectorProvider: state.vectorProvider,
			TaskQueue:      state.taskQueue,
			AdminKey:       serverCfg.Auth.Secret,
			RootDir:        ".",
			ReloadFunc:     sigMgr.TriggerReload,
		}

		if deleter, ok := state.taskStore.(server.TaskDeleter); ok {
			adminHandlerCfg.TaskDeleter = deleter
		}

		adminHandler, err := server.NewAdminHandler(adminHandlerCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create admin handler: %w", err)
		}

		serverOpts = append(serverOpts, server.WithAdminHandler(adminHandler))
		slog.Info("Admin API enabled", "path", "/admin/apps")
	}

	// UI handler (embedded or fallback)
	uiHandler := ui.Handler()
	serverOpts = append(serverOpts, server.WithUIHandler(uiHandler))

	srv := server.NewHTTPServer(serverCfg, appCfg, state.appManager, serverOpts...)
	return srv, nil
}

func watchConfigFile(ctx context.Context, configPath string, sigMgr *sigsvc.Manager) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Failed to create file watcher", "error", err)
		return
	}
	defer watcher.Close()

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		slog.Error("Failed to get absolute path", "error", err)
		return
	}

	if err := watcher.Add(filepath.Dir(absPath)); err != nil {
		slog.Error("Failed to watch config directory", "error", err)
		return
	}

	slog.Info("Watching config file for changes", "path", configPath)

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-watcher.Events:
			if event.Name != absPath {
				continue
			}

			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Rename == fsnotify.Rename {

				slog.Info("Config file changed, triggering reload...", "op", event.Op.String())
				// Debounce slightly
				time.Sleep(200 * time.Millisecond)

				// Trigger unified reload logic
				sigMgr.TriggerReload()
			}
		case err := <-watcher.Errors:
			slog.Error("File watcher error", "error", err)
		}
	}
}

func initLoggerFromConfig(cfg *config.ServerConfig) error {
	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetLogLoggerLevel(level)
	return nil
}

func createDefaultVectorProvider() (vector.Provider, error) {
	pCfg := &vector.ProviderConfig{
		Type: vector.ProviderChromem,
		Chromem: &vector.ChromemConfig{
			PersistPath: ".hector/chromem",
			Compress:    true,
		},
	}
	return vector.NewProvider(pCfg)
}

// appNameFromPath derives a human-readable app name from the config file path.
func appNameFromPath(configPath string) string {
	dir := filepath.Dir(configPath)
	if filepath.Base(dir) == ".hector" {
		dir = filepath.Dir(dir)
	}
	return filepath.Base(dir)
}

func reloadDefaultApp(ctx context.Context, configPath string, state *appState) (*config.AppConfig, error) {
	result, err := config.LoadAppConfigWithLean(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	defaultApp := &app.App{
		ID:         "default",
		Name:       appNameFromPath(configPath),
		ConfigJSON: string(result.LeanJSON),
	}

	if err := state.appStore.Update(ctx, defaultApp); err != nil {
		return nil, fmt.Errorf("failed to update app in database: %w", err)
	}

	if err := state.appManager.Preload(ctx, "default"); err != nil {
		return nil, fmt.Errorf("failed to reload runtime: %w", err)
	}

	return result.Config, nil
}

func printServerInfo(serverCfg *config.ServerConfig, appCfg *config.AppConfig, configPath string, autoSecret string) {
	greenColor := "\033[38;2;16;185;129m"
	resetColor := "\033[0m"

	fmt.Printf("\n%s🚀 Hector Server Ready!%s\n", greenColor, resetColor)

	fmt.Printf("\n%sServer Configuration:%s\n", greenColor, resetColor)
	fmt.Printf("   Address:      http://%s\n", serverCfg.Address())
	fmt.Printf("   Database:     %s\n", serverCfg.Database)
	fmt.Printf("   Log Level:    %s\n", serverCfg.LogLevel)
	if serverCfg.Auth != nil && serverCfg.Auth.IsEnabled() {
		fmt.Printf("   Auth:         enabled\n")
		if serverCfg.Auth.Secret != "" {
			fmt.Printf("   Admin API:    enabled\n")
		}
	}
	if autoSecret != "" {
		yellowColor := "\033[33m"
		fmt.Printf("\n%s⚠  Admin Secret (auto-generated — use --auth-secret to set your own):%s\n", yellowColor, resetColor)
		fmt.Printf("   %s%s%s\n", yellowColor, autoSecret, resetColor)
	}
	if serverCfg.MetricsEnabled {
		fmt.Printf("   Metrics:      enabled\n")
	}

	fmt.Printf("\n%sApp Configuration:%s\n", greenColor, resetColor)
	fmt.Printf("   Config File:  %s\n", configPath)
	fmt.Printf("   Agents:       %d\n", len(appCfg.Agents))
	fmt.Printf("   LLMs:         %d\n", len(appCfg.LLMs))
	fmt.Printf("   Tools:        %d\n", len(appCfg.Tools))

	fmt.Printf("\n%sEndpoints:%s\n", greenColor, resetColor)
	fmt.Printf("   Discovery:    http://%s/agents\n", serverCfg.Address())
	fmt.Printf("   Health:       http://%s/health\n", serverCfg.Address())
	if serverCfg.MetricsEnabled {
		fmt.Printf("   Metrics:      http://%s/metrics\n", serverCfg.Address())
	}

	if len(appCfg.Agents) > 0 {
		fmt.Printf("\n%sAgents:%s\n", greenColor, resetColor)
		for name := range appCfg.Agents {
			fmt.Printf("   - http://%s/agents/%s\n", serverCfg.Address(), name)
		}
	}

	fmt.Println("\nPress Ctrl+C to stop")
}
