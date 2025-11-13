package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/auth"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/observability"
	"github.com/kadirpekel/hector/pkg/runtime"
	"github.com/kadirpekel/hector/pkg/transport"
	"google.golang.org/grpc"
)

type Server struct {
	config       *config.Config
	configLoader *config.Loader
	opts         Options

	runtime       *runtime.Runtime
	observability *observability.Manager

	grpcServer  *transport.Server
	restGateway *transport.RESTGateway

	stopChan   chan struct{}
	reloadChan chan *config.Config // Pass config through channel to avoid races
	doneChan   chan struct{}

	mu sync.RWMutex // Protect config updates during reload
}

type Options struct {
	Config *config.Config

	ConfigLoader *config.Loader

	Host    string
	Port    int
	BaseURL string
	Debug   bool
}

func New(opts Options) (*Server, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}

	s := &Server{
		config:       opts.Config,
		configLoader: opts.ConfigLoader,
		opts:         opts,
		stopChan:     make(chan struct{}),
		reloadChan:   make(chan *config.Config, 1), // Buffered to hold latest config
		doneChan:     make(chan struct{}),
	}

	if s.configLoader != nil {
		s.configLoader.SetOnChange(func(newCfg *config.Config) error {
			log.Printf("📝 Configuration change detected, triggering server reload...")
			// Send config through channel (non-blocking, will overwrite if reload in progress)
			select {
			case s.reloadChan <- newCfg:
				// Config queued for reload
			default:
				// Reload already in progress, replace with latest config
				select {
				case <-s.reloadChan: // Remove old config
					s.reloadChan <- newCfg // Queue latest config
				default:
					// Channel was already empty, just send
					s.reloadChan <- newCfg
				}
			}
			return nil
		})
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.initialize(); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	if err := s.startTransport(); err != nil {
		s.cleanup(context.Background())
		return fmt.Errorf("failed to start transport: %w", err)
	}

	s.logStartup()

	go s.runLifecycle()

	return nil
}

func (s *Server) Wait() {
	<-s.doneChan

	if s.configLoader != nil {
		s.configLoader.Stop()
	}
}

func (s *Server) Stop(ctx context.Context) error {
	close(s.stopChan)

	select {
	case <-s.doneChan:

		if s.configLoader != nil {
			s.configLoader.Stop()
		}
		return nil
	case <-ctx.Done():

		if s.configLoader != nil {
			s.configLoader.Stop()
		}
		return ctx.Err()
	}
}

func (s *Server) initialize() error {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	if s.opts.Debug {
		log.Println("Initializing runtime and agents...")
	}

	// Resolve base URL before creating runtime so agents can use it
	addresses := s.resolveAddresses()
	cfg.Global.A2AServer.BaseURL = addresses.BaseURL

	rt, err := runtime.NewWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("runtime initialization failed: %w", err)
	}
	s.runtime = rt

	if config.BoolValue(cfg.Global.Observability.MetricsEnabled, false) || config.BoolValue(cfg.Global.Observability.Tracing.Enabled, false) {
		log.Println("🔭 Initializing observability...")

		obsConfig := observability.Config{
			Tracing: observability.TracerConfig{
				Enabled:      config.BoolValue(cfg.Global.Observability.Tracing.Enabled, false),
				ExporterType: cfg.Global.Observability.Tracing.ExporterType,
				EndpointURL:  cfg.Global.Observability.Tracing.EndpointURL,
				SamplingRate: cfg.Global.Observability.Tracing.SamplingRate,
				ServiceName:  cfg.Global.Observability.Tracing.ServiceName,
			},
			MetricsEnabled: config.BoolValue(cfg.Global.Observability.MetricsEnabled, false),
		}

		obsMgr := observability.NewManager(obsConfig)
		if err := obsMgr.Initialize(context.Background()); err != nil {
			log.Printf("⚠️  Failed to initialize observability: %v", err)
		} else {
			s.observability = obsMgr
			if obsConfig.Tracing.Enabled {
				log.Printf("  ✅ Tracing enabled (endpoint: %s, sampling: %.2f)",
					obsConfig.Tracing.EndpointURL, obsConfig.Tracing.SamplingRate)
			}
			if obsConfig.MetricsEnabled {
				log.Printf("  ✅ Metrics enabled")
			}
		}
	}

	agentRegistry := rt.Registry()
	for _, agentID := range agentRegistry.ListAgents() {
		entry, _ := agentRegistry.Get(agentID)

		if entry.Config.Type == "a2a" {
			log.Printf("External agent '%s' connected to %s", agentID, entry.Config.URL)
		} else {
			log.Printf("Native agent '%s' created", agentID)

			// Recover pending tasks with checkpoints (if enabled)
			if nativeAgent, ok := entry.Agent.(*agent.Agent); ok {
				if err := nativeAgent.RecoverPendingTasks(context.Background()); err != nil {
					log.Printf("⚠️  Failed to recover pending tasks for agent '%s': %v", agentID, err)
					// Don't fail initialization - recovery is best-effort
				}
			}
		}
	}

	if agentRegistry.ListAgents() == nil || len(agentRegistry.ListAgents()) == 0 {
		return fmt.Errorf("no agents successfully registered")
	}

	return nil
}

func (s *Server) startTransport() error {
	agentRegistry := s.runtime.Registry()
	agentRouter := agent.NewAgentRouter(agentRegistry)

	addresses := s.resolveAddresses()

	authConfig, err := s.setupAuth()
	if err != nil {
		return fmt.Errorf("auth setup failed: %w", err)
	}

	grpcConfig := transport.Config{
		Address: addresses.GRPC,
	}

	// Chain observability and auth interceptors
	// Observability runs first to capture all requests, then auth
	var unaryInterceptors []grpc.UnaryServerInterceptor
	var streamInterceptors []grpc.StreamServerInterceptor

	// Always add observability interceptors
	unaryInterceptors = append(unaryInterceptors, transport.UnaryServerInterceptor())
	streamInterceptors = append(streamInterceptors, transport.StreamServerInterceptor())

	// Add auth interceptors if configured
	if authConfig != nil && authConfig.Validator != nil {
		unaryInterceptors = append(unaryInterceptors, authConfig.Validator.UnaryServerInterceptor())
		streamInterceptors = append(streamInterceptors, authConfig.Validator.StreamServerInterceptor())
	}

	// Chain the interceptors
	grpcConfig.UnaryInterceptor = transport.ChainUnaryInterceptors(unaryInterceptors...)
	grpcConfig.StreamInterceptor = transport.ChainStreamInterceptors(streamInterceptors...)

	s.grpcServer = transport.NewRegistryServer(agentRegistry, grpcConfig)

	// Create HTTP server (REST Gateway with JSON-RPC and Web UI)
	s.restGateway = transport.NewRESTGateway(transport.RESTGatewayConfig{
		HTTPAddress: addresses.HTTP,
		GRPCAddress: addresses.GRPC,
		BaseURL:     addresses.BaseURL,
	})
	if authConfig != nil {
		s.restGateway.SetAuth(authConfig)
	}

	discovery := transport.NewAgentDiscovery(agentRouter, authConfig)
	s.restGateway.SetDiscovery(discovery)
	s.restGateway.SetService(agentRouter)

	errChan := make(chan error, 2)

	go func() {
		if err := s.grpcServer.Start(); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	go func() {
		if err := s.restGateway.Start(context.Background()); err != nil {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-time.After(500 * time.Millisecond):

		return nil
	}
}

func (s *Server) runLifecycle() {
	defer close(s.doneChan)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sigCh:
			log.Println("Shutting down...")
			s.cleanup(context.Background())
			return

		case <-s.stopChan:
			log.Println("Stop requested...")
			s.cleanup(context.Background())
			return

		case newCfg := <-s.reloadChan:
			log.Println("🔄 Configuration reload requested...")

			// Store old config for rollback (can't store runtime/servers as they'll be shut down)
			s.mu.RLock()
			oldConfig := s.config
			s.mu.RUnlock()

			// Update config atomically
			s.mu.Lock()
			s.config = newCfg
			s.mu.Unlock()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			s.cleanup(shutdownCtx)
			cancel()

			// Try to initialize with new config
			if err := s.initialize(); err != nil {
				log.Printf("❌ Failed to reinitialize after reload: %v", err)
				log.Printf("⚠️  Attempting to restore previous configuration...")

				// Rollback: restore old config and recreate everything
				s.mu.Lock()
				s.config = oldConfig
				s.mu.Unlock()

				// Recreate runtime and servers with old config
				if err := s.initialize(); err != nil {
					log.Printf("❌ Failed to restore previous configuration: %v", err)
					log.Printf("❌ Server is in an inconsistent state. Manual restart required.")
					return
				}

				if err := s.startTransport(); err != nil {
					log.Printf("❌ Failed to restore transport with previous configuration: %v", err)
					log.Printf("❌ Server is in an inconsistent state. Manual restart required.")
					return
				}
				log.Printf("✅ Previous configuration restored successfully")
				s.logStartup()
				continue // Continue lifecycle loop
			}

			if err := s.startTransport(); err != nil {
				log.Printf("❌ Failed to start transport after reload: %v", err)
				log.Printf("⚠️  Attempting to restore previous configuration...")

				// Rollback: restore old config and recreate everything
				s.mu.Lock()
				s.config = oldConfig
				s.mu.Unlock()

				// Recreate runtime and servers with old config
				if err := s.initialize(); err != nil {
					log.Printf("❌ Failed to restore previous configuration: %v", err)
					log.Printf("❌ Server is in an inconsistent state. Manual restart required.")
					return
				}

				if err := s.startTransport(); err != nil {
					log.Printf("❌ Failed to restore transport with previous configuration: %v", err)
					log.Printf("❌ Server is in an inconsistent state. Manual restart required.")
					return
				}
				log.Printf("✅ Previous configuration restored successfully")
				s.logStartup()
				continue // Continue lifecycle loop
			}

			s.logStartup()
			log.Println("✅ Server reloaded successfully")
		}
	}
}

func (s *Server) cleanup(ctx context.Context) {
	var shutdownErrors []error

	// Shutdown all agents gracefully first
	if s.runtime != nil {
		agentRegistry := s.runtime.Registry()
		if agentRegistry != nil {
			agentIDs := agentRegistry.ListAgents()
			for _, agentID := range agentIDs {
				agent, err := agentRegistry.GetAgent(agentID)
				if err != nil {
					log.Printf("Warning: Failed to get agent %s for shutdown: %v", agentID, err)
					continue
				}
				// Check if agent implements Shutdown method
				if shutdownable, ok := agent.(interface{ Shutdown(context.Context) error }); ok {
					if err := shutdownable.Shutdown(ctx); err != nil {
						shutdownErrors = append(shutdownErrors, fmt.Errorf("agent %s: %w", agentID, err))
					}
				}
			}
		}
	}

	if s.grpcServer != nil {
		if err := s.grpcServer.Stop(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("gRPC: %w", err))
		}
	}
	if s.restGateway != nil {
		if err := s.restGateway.Stop(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("HTTP: %w", err))
		}
	}

	if s.observability != nil {
		if err := s.observability.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("observability: %w", err))
		}
	}

	if s.runtime != nil {
		if err := s.runtime.Close(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("runtime: %w", err))
		}
	}

	if len(shutdownErrors) > 0 {
		for _, err := range shutdownErrors {
			log.Printf("Shutdown error: %v", err)
		}
	}
}

func (s *Server) resolveAddresses() ServerAddresses {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	httpPort := 8080
	grpcPort := 50051
	serverHost := "0.0.0.0"
	baseURL := ""

	if cfg.Global.A2AServer.IsEnabled() {
		if cfg.Global.A2AServer.Port > 0 {
			httpPort = cfg.Global.A2AServer.Port
		}
		if cfg.Global.A2AServer.GRPCPort > 0 {
			grpcPort = cfg.Global.A2AServer.GRPCPort
		}
		if cfg.Global.A2AServer.Host != "" {
			serverHost = cfg.Global.A2AServer.Host
		}
		if cfg.Global.A2AServer.BaseURL != "" {
			baseURL = cfg.Global.A2AServer.BaseURL
		}
	}

	if s.opts.Port != 0 && s.opts.Port != 8080 {
		httpPort = s.opts.Port
		// gRPC port stays at 50051 (canonical gRPC port) regardless of HTTP port
	}
	if s.opts.Host != "" {
		serverHost = s.opts.Host
	}
	if s.opts.BaseURL != "" {
		baseURL = s.opts.BaseURL
	}

	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s:%d", serverHost, httpPort)
	}

	return ServerAddresses{
		HTTP:    fmt.Sprintf("%s:%d", serverHost, httpPort),
		GRPC:    fmt.Sprintf("%s:%d", serverHost, grpcPort),
		BaseURL: baseURL,
		Host:    serverHost,
		Port:    httpPort,
	}
}

func (s *Server) setupAuth() (*transport.AuthConfig, error) {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	if !cfg.Global.Auth.IsEnabled() {
		return nil, nil
	}

	jwtValidator, err := auth.NewJWTValidator(
		cfg.Global.Auth.JWKSURL,
		cfg.Global.Auth.Issuer,
		cfg.Global.Auth.Audience,
	)
	if err != nil {
		return nil, fmt.Errorf("JWT validator creation failed: %w", err)
	}

	log.Printf("✅ Authentication configured (JWT)")

	return &transport.AuthConfig{
		Enabled:   true,
		Validator: jwtValidator,
	}, nil
}

func (s *Server) logStartup() {
	addresses := s.resolveAddresses()
	agentCount := len(s.runtime.Registry().ListAgents())

	log.Printf("✅ Server started with %d agents", agentCount)
	log.Printf("🌐 HTTP: http://%s", addresses.HTTP)
	log.Printf("📡 gRPC: %s", addresses.GRPC)

	if s.opts.Debug {
		log.Printf("   → Web UI (GET): http://%s/", addresses.HTTP)
		log.Printf("   → JSON-RPC (POST): http://%s/", addresses.HTTP)
		log.Printf("   → REST API: http://%s/v1/", addresses.HTTP)
	}

	log.Printf("Press Ctrl+C to stop")
}

type ServerAddresses struct {
	HTTP    string
	GRPC    string
	BaseURL string
	Host    string
	Port    int
}
