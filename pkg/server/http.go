package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/invopop/jsonschema"

	"github.com/verikod/hector/pkg/app"
	"github.com/verikod/hector/pkg/auth"
	"github.com/verikod/hector/pkg/config"
	"github.com/verikod/hector/pkg/execution"
	"github.com/verikod/hector/pkg/observability"
	"github.com/verikod/hector/pkg/rag"
	"github.com/verikod/hector/pkg/ratelimit"
	"github.com/verikod/hector/pkg/session"
	"github.com/verikod/hector/pkg/task"
	"github.com/verikod/hector/pkg/trigger"
)

// webUIHTML removed (headless server)

// prepareAgentCard clones an agent card and sets its URL dynamically based on the request.
// This ensures the URL is correct regardless of proxies, tunnels, or port changes.
func prepareAgentCard(card *a2a.AgentCard, r *http.Request) *a2a.AgentCard {
	scheme := "http"
	// Trust X-Forwarded-Proto from trusted proxies (like Fly.io)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}

	c := *card
	if strings.HasPrefix(c.URL, "/") {
		// Relative URL (from app_manager, e.g. /agents/pr-reviewer) - make absolute
		c.URL = fmt.Sprintf("%s://%s%s", scheme, r.Host, c.URL)
	}
	// If c.URL is absolute (e.g. valid remote agent) or empty, leave it as is
	return &c
}

// HTTPServer is the Hector HTTP server.
// Uses a2a-go native handlers for A2A protocol compliance.
type HTTPServer struct {
	serverCfg *config.ServerConfig
	appCfg    *config.AppConfig
	server    *http.Server

	// gRPC server (only when Transport == TransportGRPC)
	// grpcServer *grpc.Server
	//nolint:unused // Reserved for future use
	grpcListener net.Listener

	// TaskStore for persistent task storage (nil = in-memory)
	taskStore a2asrv.TaskStore

	// Execution provider for async execution
	execProvider execution.Provider

	// Auth: JWT validator and a2a-go interceptor
	authValidator   auth.TokenValidator
	authInterceptor *auth.Interceptor

	// Observability: tracing and metrics
	observability *observability.Manager

	// Rate limiting
	rateLimiter ratelimit.RateLimiter

	// Task service for task-scoped cancellation
	taskService task.Service

	// Session service for session events API
	sessionService session.Service

	// Document store provider for RAG indexing status
	documentStoreProvider DocumentStoreProvider

	// UI handler for serving the web UI at /
	uiHandler http.Handler

	// Multi-app manager and dynamic handlers
	appManager          *AppManager
	dynamicAgentHandler *DynamicAgentHandler
	adminHandler        *AdminHandler

	mu sync.RWMutex
}

// HTTPServerOption configures the HTTP server.
type HTTPServerOption func(*HTTPServer)

// WithTaskStore sets the task store for persistent task storage.
// If not set, a2a-go uses its internal in-memory store.
func WithTaskStore(store a2asrv.TaskStore) HTTPServerOption {
	return func(s *HTTPServer) {
		s.taskStore = store
	}
}

// WithAuthValidator sets the JWT validator for authentication.
// When set, HTTP requests will be validated and claims passed to agents.
func WithAuthValidator(validator auth.TokenValidator) HTTPServerOption {
	return func(s *HTTPServer) {
		s.authValidator = validator
	}
}

// WithObservability sets the observability manager for tracing and metrics.
func WithObservability(obs *observability.Manager) HTTPServerOption {
	return func(s *HTTPServer) {
		s.observability = obs
	}
}

// WithRateLimiter sets the rate limiter for request rate limiting.
func WithRateLimiter(limiter ratelimit.RateLimiter) HTTPServerOption {
	return func(s *HTTPServer) {
		s.rateLimiter = limiter
	}
}

// WithTaskService sets the task service for task-scoped cancellation.
func WithTaskService(ts task.Service) HTTPServerOption {
	return func(s *HTTPServer) {
		s.taskService = ts
	}
}

// WithSessionService sets the session service.
func WithSessionService(ss session.Service) HTTPServerOption {
	return func(s *HTTPServer) {
		s.sessionService = ss
	}
}

// WithExecutionProvider sets the execution provider for async execution.
func WithExecutionProvider(p execution.Provider) HTTPServerOption {
	return func(s *HTTPServer) {
		s.execProvider = p
	}
}

// WithAdminHandler sets the admin handler for app management.
func WithAdminHandler(handler *AdminHandler) HTTPServerOption {
	return func(s *HTTPServer) {
		s.adminHandler = handler
	}
}

// DocumentStoreProvider provides access to document stores for status reporting.
// This interface avoids importing the runtime package directly.
type DocumentStoreProvider interface {
	DocumentStores() map[string]*rag.DocumentStore
}

// WithDocumentStoreProvider sets the provider for document store status.
func WithDocumentStoreProvider(p DocumentStoreProvider) HTTPServerOption {
	return func(s *HTTPServer) {
		s.documentStoreProvider = p
	}
}

// WithUIHandler sets the handler for serving the web UI.
func WithUIHandler(handler http.Handler) HTTPServerOption {
	return func(s *HTTPServer) {
		s.uiHandler = handler
	}
}

// NewHTTPServer creates a new HTTP server from config.
// appManager handles loading apps and creating executors.
func NewHTTPServer(serverCfg *config.ServerConfig, appCfg *config.AppConfig, appManager *AppManager, opts ...HTTPServerOption) *HTTPServer {
	s := &HTTPServer{
		serverCfg:  serverCfg,
		appCfg:     appCfg,
		appManager: appManager,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Create auth interceptor if validator is configured.
	// IMPORTANT: RequireAuth is set to FALSE here because:
	// 1. /agents/ paths are excluded from HTTP auth middleware to support per-agent visibility
	// 2. Visibility-based auth (public/internal/private) is handled in handleAgentRoutes/DynamicHandler
	var authInterceptor *auth.Interceptor
	if s.authValidator != nil {
		authInterceptor = auth.NewInterceptor(false)
		s.authInterceptor = authInterceptor
	}

	// Initialize dynamic handler
	s.dynamicAgentHandler = NewDynamicAgentHandler(appManager, serverCfg, authInterceptor, s.authValidator, s.taskStore, s.execProvider)

	return s
}

// GetAgentA2AInvoker returns an AgentInvoker that uses A2A handlers for invocation.
// This enables webhooks to invoke agents through A2A protocol with automatic TaskStore registration.
func (s *HTTPServer) GetAgentA2AInvoker() trigger.AgentInvoker {
	return func(ctx context.Context, agentName, input string) (string, error) {
		// Get Agent Runtime dynamically
		appID := app.IDFromContext(ctx)
		runtime, err := s.appManager.GetRuntime(ctx, appID, agentName)
		if err != nil {
			return "", fmt.Errorf("agent %q not found: %w", agentName, err)
		}

		// Build A2A message with metadata to signal non-streaming mode.
		// Webhooks don't benefit from streaming - we just need the final result.
		// Note: RequestContext.Metadata comes from Message.Metadata, not params.Metadata!
		msg := &a2a.Message{
			Role:  a2a.MessageRoleUser,
			Parts: []a2a.Part{a2a.TextPart{Text: input}},
			Metadata: map[string]any{
				"hector:source": "webhook", // Signal to Executor to disable streaming
			},
		}

		// Session continuity: Use ContextID if provided via trigger context
		if contextID := trigger.ContextIDFromContext(ctx); contextID != "" {
			msg.ContextID = contextID
		}

		blocking := false
		params := &a2a.MessageSendParams{
			Message: msg,
			Config: &a2a.MessageSendConfig{
				Blocking: &blocking,
			},
		}

		// Create a temporary handler for this invocation
		// We could cache this but it's cheap to create
		var handlerOpts []a2asrv.RequestHandlerOption
		if s.taskStore != nil {
			handlerOpts = append(handlerOpts, a2asrv.WithTaskStore(s.taskStore))
		}

		requestHandler := a2asrv.NewHandler(runtime.Executor, handlerOpts...)

		// Call A2A handler - this auto-registers task in TaskStore
		result, err := requestHandler.OnSendMessage(ctx, params)
		if err != nil {
			return "", fmt.Errorf("A2A invocation failed: %w", err)
		}

		// Extract task ID from result
		if task, ok := result.(*a2a.Task); ok {
			return string(task.ID), nil
		}

		// If result is not a task, return empty ID (invocation still succeeded)
		return "", nil
	}
}

// Start starts the HTTP server.
func (s *HTTPServer) Start(ctx context.Context) error {
	// Build the initial handler (Mux + middleware)
	handler := s.buildHandler()

	s.server = &http.Server{
		Addr:         s.serverCfg.Address(),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	slog.Info("HTTP server starting", "address", s.serverCfg.Address())

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	}
}

// getClientIP extracts the real client IP address from the request.
// Handles proxies/load balancers by checking configurable headers.
// This is critical for accurate rate limiting when behind proxies/load balancers.
//
// Priority order (configurable via ipHeaders):
//  1. First header in ipHeaders list (checked in order)
//  2. For X-Forwarded-For: uses leftmost IP (original client)
//  3. RemoteAddr (direct connection or last proxy)
//
// ipHeaders: List of HTTP header names to check (in priority order).
//
//	Defaults to ["X-Forwarded-For", "X-Real-IP"] if nil/empty.
func getClientIP(r *http.Request, ipHeaders []string) string {
	// Use default headers if not specified
	headers := ipHeaders
	if len(headers) == 0 {
		headers = []string{"X-Forwarded-For", "X-Real-IP"}
	}

	// Check each header in priority order
	for _, headerName := range headers {
		headerValue := r.Header.Get(headerName)
		if headerValue == "" {
			continue
		}

		// Special handling for X-Forwarded-For (comma-separated list)
		// The leftmost IP is the original client
		if headerName == "X-Forwarded-For" {
			ips := strings.Split(headerValue, ",")
			if len(ips) > 0 {
				ip := strings.TrimSpace(ips[0])
				if ip != "" {
					return ip
				}
			}
		} else {
			// For other headers, use the value directly (single IP expected)
			ip := strings.TrimSpace(headerValue)
			if ip != "" {
				return ip
			}
		}
	}

	// Fallback to RemoteAddr (may be proxy IP if behind proxy)
	// Remove port if present (e.g., "192.168.1.1:12345" -> "192.168.1.1")
	addr := r.RemoteAddr
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// createRateLimitIdentifier creates an identifier function that works with auth.
func (s *HTTPServer) createRateLimitIdentifier(cfg *config.RateLimitConfig) ratelimit.IdentifierFunc {
	scope := ratelimit.ScopeFromConfig(cfg)
	ipHeaders := cfg.IPHeaders // Use configured headers, defaults handled in getClientIP

	return func(r *http.Request) (string, ratelimit.Scope) {
		// Try to get user ID from auth claims (if authenticated)
		if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
			// User scope: always use subject, regardless of session ID
			if scope == ratelimit.ScopeUser && claims.Subject != "" {
				return claims.Subject, ratelimit.ScopeUser
			}
			// Session scope with auth: use subject + session ID if available
			if scope == ratelimit.ScopeSession {
				if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
					return fmt.Sprintf("%s:%s", claims.Subject, sessionID), ratelimit.ScopeSession
				}
				// No session ID, use subject alone for session scope
				if claims.Subject != "" {
					return claims.Subject, ratelimit.ScopeSession
				}
			}
		}

		// Try session ID header (no auth claims)
		if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
			return sessionID, ratelimit.ScopeSession
		}

		// Fallback to IP address (extracts real client IP from configurable headers)
		return getClientIP(r, ipHeaders), ratelimit.ScopeSession
	}
}

// estimateTokenCount estimates tokens for a request.
// This is a simple heuristic - you may want to improve this.
func (s *HTTPServer) estimateTokenCount(r *http.Request) int64 {
	// Estimate based on content length
	// Rough heuristic: ~4 characters per token
	if r.ContentLength > 0 {
		return r.ContentLength / 4
	}

	// For JSON-RPC requests, try to parse and estimate
	// For JSON-RPC requests, try to parse and estimate
	if r.Header.Get("Content-Type") == "application/json" {
		// Could parse JSON and estimate more accurately
		// For now, use content length heuristic
		return r.ContentLength / 4
	}

	return 0 // Default: no token estimation
}

// buildHandler creates the full HTTP handler stack (Mux + middleware).
func (s *HTTPServer) buildHandler() http.Handler {
	mux := s.setupRoutes()

	// Apply middleware chain (order: observability -> logging -> cors -> rate limiting -> app context -> auth -> routes)
	// Observability wraps everything so all requests are traced/measured
	var handler http.Handler = mux

	// App Context Middleware REMOVED from global chain.
	// It depends on Auth, which is per-route.
	// App context resolution is now handled in specific handlers or per-route middleware.

	// Auth middleware is now applied per-route in setupRoutes using s.withAuth()
	// This avoids "all protected except X" logic which caused issues with webhooks.

	handler = s.corsMiddleware(handler)
	handler = s.loggingMiddleware(handler)

	// Rate Limiting Middleware
	// Execution order: runs after observability (outermost), before logging/cors
	// This ensures rate-limited requests are still traced/measured by observability
	if s.rateLimiter != nil && s.serverCfg.RateLimit != nil {
		identifierFunc := s.createRateLimitIdentifier(s.serverCfg.RateLimit)
		tokenEstimator := s.estimateTokenCount

		// Exclude health checks and metrics
		excludedPaths := []string{
			"/health",
			"/metrics",
			"/schema", // Public schema endpoint
		}

		rateLimitMiddleware := ratelimit.Middleware(ratelimit.MiddlewareConfig{
			Limiter:        s.rateLimiter,
			IdentifierFunc: identifierFunc,
			TokenEstimator: tokenEstimator,
			ExcludedPaths:  excludedPaths,
		})

		handler = rateLimitMiddleware(handler)
		slog.Info("Rate limiting middleware enabled",
			"excluded_paths", excludedPaths)
	}

	// Observability middleware (outermost for complete request coverage)
	// Applied last so all requests (including rate-limited ones) are traced/measured
	if s.observability != nil {
		handler = observability.HTTPMiddleware(s.observability.Tracer(), s.observability.Metrics())(handler)
	}

	return handler
}

// Shutdown gracefully shuts down the server(s).
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var errs []error

	// Shutdown HTTP server
	if s.server != nil {
		slog.Info("HTTP server shutting down")
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("HTTP shutdown error: %w", err))
		}
	}

	// Shutdown gRPC server
	// Shutdown gRPC server
	// if s.grpcServer != nil {
	// 	slog.Info("gRPC server shutting down")
	// 	stopped := make(chan struct{})
	// 	go func() {
	// 		s.grpcServer.GracefulStop()
	// 		close(stopped)
	// 	}()
	//
	// 	select {
	// 	case <-stopped:
	// 		slog.Info("gRPC server stopped gracefully")
	// 	case <-shutdownCtx.Done():
	// 		slog.Warn("gRPC graceful stop timeout, forcing shutdown")
	// 		s.grpcServer.Stop()
	// 	}
	// }

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

// Address returns the HTTP server address.
func (s *HTTPServer) Address() string {
	return s.serverCfg.Address()
}

// GRPCAddress returns the gRPC server address (if enabled).
// Transport and gRPC features temporarily disabled during config refactoring.
func (s *HTTPServer) GRPCAddress() string {
	// gRPC temporarily disabled during config refactoring
	return ""
}

// initGRPCServer initializes the gRPC server with interceptors.
// Note: This is currently unused until gRPC is re-enabled, but provides the foundation.
// func (s *HTTPServer) initGRPCServer() {
// 	opts := []grpc.ServerOption{
// 		grpc.StatsHandler(otelgrpc.NewServerHandler()),
// 	}
// 	s.grpcServer = grpc.NewServer(opts...)
// }

// GetTaskResultProvider returns a function that fetches task result from TaskStore.
func (s *HTTPServer) GetTaskResultProvider() func(ctx context.Context, taskID string) (string, error) {
	return func(ctx context.Context, taskID string) (string, error) {
		t, err := s.taskStore.Get(ctx, a2a.TaskID(taskID))
		if err != nil {
			return "", err
		}

		if t.Status.State != a2a.TaskStateCompleted {
			if t.Status.State == a2a.TaskStateFailed {
				return "", fmt.Errorf("task failed")
			}
			return "", nil // Still running
		}

		// Return the text content of the first artifact
		if len(t.Artifacts) > 0 {
			var sb strings.Builder
			for _, p := range t.Artifacts[0].Parts {
				if txt, ok := p.(a2a.TextPart); ok {
					sb.WriteString(txt.Text)
				}
			}
			return sb.String(), nil
		}

		return "No result artifact produced", nil
	}
}

// withAuth wraps a handler with authentication middleware if authentication is enabled.
func (s *HTTPServer) withAuth(h http.Handler) http.Handler {
	if s.authValidator != nil {
		return auth.OptionalMiddleware(s.authValidator)(h)
	}
	return h
}

// withAppCtx wraps a handler with both Auth and AppContext middleware.
// It ensures that Auth runs first to populate claims, then AppContext runs to resolve the App ID.
func (s *HTTPServer) withAppCtx(h http.Handler) http.Handler {
	handler := h

	// Inner: Resolve App Context (depends on claims)
	if s.appManager != nil {
		handler = AppContextMiddleware(s.appManager.appStore)(handler)
	}

	// Outer: Authenticate (populates claims)
	return s.withAuth(handler)
}

// setupRoutes configures the HTTP routes.
func (s *HTTPServer) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Schema endpoint (public - no auth required)
	mux.HandleFunc("/schema", s.handleGetSchema)

	// Admin API (if handler configured)
	// Includes: /admin/apps, /admin/jwks
	if s.adminHandler != nil {
		mux.Handle("/admin/", http.StripPrefix("/admin", s.adminHandler))
	}

	// Prometheus metrics endpoint (if enabled)
	if s.observability != nil && s.observability.MetricsEnabled() {
		metricsEndpoint := s.observability.MetricsEndpoint()
		mux.Handle(metricsEndpoint, s.observability.MetricsHandler())
		slog.Info("Metrics endpoint enabled", "path", metricsEndpoint)
	}

	// Dynamic agent routes (JSON-RPC and Agent Card)
	// Handles: /agents/{name}, /agents/{name}/.well-known/agent-card.json, etc.
	// Note: We don't strip prefix because the handler needs to parse the name from URL
	mux.Handle("/agents/", s.withAppCtx(s.dynamicAgentHandler))

	// A2A spec: server-level well-known agent card (returns default/first agent of current app)
	mux.Handle(a2asrv.WellKnownAgentCardPath, s.withAppCtx(http.HandlerFunc(s.handleDefaultAgentCard)))
	// Agent discovery
	mux.Handle("/agents", s.withAppCtx(http.HandlerFunc(s.handleDiscovery)))

	// Tool cancellation endpoint (Hector extension)
	// Pattern: /tasks/{taskId}/toolCalls/{callId}/cancel
	// Protected by auth and requires app context
	mux.Handle("/tasks/", s.withAppCtx(http.HandlerFunc(s.handleTasksAPI)))

	// Web UI at root (catch-all; all explicit API routes above take priority)
	if s.uiHandler != nil {
		mux.Handle("/", s.uiHandler)
	}

	// Dynamic Webhook Handler
	// Handles: /webhooks/{agentName}
	// Resolved dynamically at runtime from current config.
	mux.HandleFunc("/webhooks/", s.handleWebhook)

	return mux
}

// handleWebhook dynamically handles webhook requests by resolving the agent from the URL.
// Logic:
// 1. Check for exact match on configured Trigger.Path (allows custom paths like /webhooks/secret)
// 2. Fallback to convention /webhooks/{agentName}
func (s *HTTPServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Resolve App Context
	appID := app.IDFromContext(ctx)
	var targetAgentName string

	// Foundational Resolution: If no App Context (unauthenticated), resolve via global lookup
	if appID == "" && s.appManager != nil {
		resolvedAppID, resolvedAgentName, err := s.appManager.ResolveWebhook(ctx, r.URL.Path)
		if err == nil {
			appID = resolvedAppID
			targetAgentName = resolvedAgentName
			// Inject resolved App ID into context for downstream usage
			ctx = app.WithAppID(ctx, appID)
			r = r.WithContext(ctx)
			slog.Debug("Resolved webhook globally", "app", appID, "agent", targetAgentName, "path", r.URL.Path)
		}
	}

	// 1. Lookup Config from AppManager (DB/Runtime source of truth)
	var cfg *config.AppConfig
	if s.appManager != nil {
		dbCfg, err := s.appManager.GetAppConfig(ctx, appID)
		if err == nil && dbCfg != nil {
			cfg = dbCfg
		}
	}

	// Fallback to file config if AppManager missing or failed
	if cfg == nil {
		s.mu.RLock()
		cfg = s.appCfg
		s.mu.RUnlock()
	}

	if cfg == nil || len(cfg.Agents) == 0 {
		http.Error(w, "No agents configured", http.StatusNotFound)
		return
	}

	var targetTriggerCfg *config.TriggerConfig

	if targetAgentName != "" {
		// Already resolved via global lookup
		if agent, ok := cfg.Agents[targetAgentName]; ok {
			targetTriggerCfg = agent.Trigger
		}
	} else if s.appManager != nil {
		// Fallback to manual lookup using shared logic if not resolved globally (e.g. file config matching)
		if name, ok := s.appManager.LookupWebhookInConfig(r.URL.Path, cfg); ok {
			targetAgentName = name
			targetTriggerCfg = cfg.Agents[name].Trigger
		}
	}

	if targetAgentName == "" || targetTriggerCfg == nil {
		slog.Debug("Webhook target not found", "path", r.URL.Path)
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	// 3. Create Handler on the fly
	invoker := s.GetAgentA2AInvoker()
	h, err := trigger.NewWebhookHandler(targetAgentName, targetTriggerCfg, invoker)
	if err != nil {
		slog.Error("Failed to create webhook handler", "agent", targetAgentName, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 4. Serve
	h.ServeHTTP(w, r)
}

// Reload updates the server configuration and state.
// Routes are dynamic, so this simply updates the internal config reference.
func (s *HTTPServer) Reload(newCfg *config.AppConfig, newAppManager *AppManager) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.appCfg = newCfg
	s.appManager = newAppManager

	// Update dynamic handler with new AppManager
	var authInterceptor *auth.Interceptor
	if s.authValidator != nil {
		authInterceptor = auth.NewInterceptor(false)
		s.authInterceptor = authInterceptor
	}
	// ServerConfig is now separate - use existing serverCfg instead of newCfg.Server
	s.dynamicAgentHandler = NewDynamicAgentHandler(newAppManager, s.serverCfg, authInterceptor, s.authValidator, s.taskStore, s.execProvider)

	// No route regeneration needed! Webhooks are resolved dynamically.
}

// resolveAuthClaims attempts to resolve auth claims from context or header.
func resolveAuthClaims(r *http.Request, validator auth.TokenValidator) *auth.Claims {
	if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
		return claims
	}
	if validator != nil {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != "" {
				if parsed, err := validator.ValidateToken(r.Context(), token); err == nil {
					return parsed
				}
			}
		}
	}
	return nil
}

// handleDefaultAgentCard serves the card for the default (first) agent of the current app.
func (s *HTTPServer) handleDefaultAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Resolve Identity & App Context
	claims := resolveAuthClaims(r, s.authValidator)
	isAuthenticated := claims != nil

	appID := app.IDFromContext(ctx)
	if isAuthenticated && claims.TenantID != "" {
		appID = claims.TenantID
	}

	agents, err := s.appManager.ListAgents(ctx, appID)
	if err != nil || len(agents) == 0 {
		http.Error(w, "No agents configured", http.StatusNotFound)
		return
	}

	// Check first agent visibility
	first := agents[0]
	if first.Visibility == "private" {
		http.Error(w, "No agents configured", http.StatusNotFound)
		return
	}
	if first.Visibility == "internal" && !isAuthenticated {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Return the first agent's card
	card := *first.Card
	handler := a2asrv.NewStaticAgentCardHandler(&card)
	handler.ServeHTTP(w, r)
}

// handleDiscovery returns a list of agent cards for the current app.
func (s *HTTPServer) handleDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Resolve Identity & App Context
	claims := resolveAuthClaims(r, s.authValidator)
	isAuthenticated := claims != nil

	// App ID is resolved by AppContext middleware (now per-route)
	appID := app.IDFromContext(ctx)

	// Admin Key Convenience: If auth is enabled but no App Context (Admin Token without tenant_id),
	// fall back to "default" app. This is BY DESIGN for power CLI users who just want
	// to deploy with file config and go instantly. App Tokens provide strict multi-app isolation.
	if isAuthenticated && appID == "" {
		appID = "default"
	}

	agents, err := s.appManager.ListAgents(ctx, appID)
	if err != nil {
		slog.Error("Failed to list agents", "error", err, "app", appID)
		http.Error(w, "Failed to list agents", http.StatusInternalServerError)
		return
	}

	var cards []*a2a.AgentCard
	for _, rt := range agents {
		// Visibility Check
		// Private: Never shown in discovery
		if rt.Visibility == "private" {
			continue
		}
		// Internal: Only shown to authenticated users
		if rt.Visibility == "internal" && !isAuthenticated {
			continue
		}

		cards = append(cards, prepareAgentCard(rt.Card, r))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"agents": cards,
		"total":  len(cards),
	}); err != nil {
		slog.Error("Failed to encode agent list", "error", err)
	}
}

// handleHealth returns server health status.
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	docStoreProvider := s.documentStoreProvider
	s.mu.RUnlock()

	response := map[string]any{
		"status": "ok",
	}

	// Auth discovery for clients
	if s.serverCfg.Auth != nil && s.serverCfg.Auth.IsEnabled() {
		response["auth"] = map[string]any{
			"enabled":   true,
			"type":      "jwt",
			"issuer":    s.serverCfg.Auth.Issuer,
			"audience":  s.serverCfg.Auth.Audience,
			"client_id": s.serverCfg.Auth.ClientID,
		}
	}

	// Admin / Studio availability
	if s.serverCfg.Auth != nil && s.serverCfg.Auth.Secret != "" {
		if response["auth"] == nil {
			response["auth"] = map[string]any{}
		}
		if authMap, ok := response["auth"].(map[string]any); ok {
			authMap["admin"] = true
		}

		response["admin"] = map[string]any{
			"enabled": true,
		}
	}

	// RAG document store indexing status
	if docStoreProvider != nil {
		stores := docStoreProvider.DocumentStores()
		if len(stores) > 0 {
			ragStatus := make(map[string]any)
			isIndexing := false
			for name, store := range stores {
				stats := store.Stats()
				storeStatus := map[string]any{
					"indexed": stats.IndexedCount,
					"total":   stats.TotalDocs,
					"skipped": stats.SkippedDocs,
				}
				// Check if indexing is still in progress
				processedCount := int64(stats.IndexedCount) + stats.SkippedDocs + stats.ErrorDocs
				if stats.TotalDocs > processedCount {
					storeStatus["indexing"] = true
					isIndexing = true
				}
				ragStatus[name] = storeStatus
			}
			response["rag"] = map[string]any{
				"stores":   ragStatus,
				"indexing": isIndexing,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// corsMiddleware adds CORS headers.
// Uses permissive defaults for development (allows all origins).
func (s *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs requests.
func (s *HTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Debug("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
		)
	})
}

// handleTasksAPI handles task-related API endpoints.
// POST /api/tasks/{taskId}/toolCalls/{callId}/cancel - Cancel a specific tool execution
func (s *HTTPServer) handleTasksAPI(w http.ResponseWriter, r *http.Request) {
	// Parse path: /tasks/{taskId}/toolCalls/{callId}/cancel
	path := strings.TrimPrefix(r.URL.Path, "/tasks/")
	parts := strings.Split(path, "/")

	// Expect: {taskId}/toolCalls/{callId}/cancel
	if len(parts) >= 4 && parts[1] == "toolCalls" && parts[3] == "cancel" {
		s.handleCancelToolCall(w, r, parts[0], parts[2])
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// handleCancelToolCall cancels a specific tool execution within a task.
// POST /api/tasks/{taskId}/toolCalls/{callId}/cancel
func (s *HTTPServer) handleCancelToolCall(w http.ResponseWriter, r *http.Request, taskID, callID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if taskID == "" || callID == "" {
		http.Error(w, "taskId and callId are required", http.StatusBadRequest)
		return
	}

	// Check if task service is configured
	if s.taskService == nil {
		http.Error(w, "Task cancellation not available", http.StatusServiceUnavailable)
		return
	}

	// Get task from service
	taskObj, err := s.taskService.Get(r.Context(), taskID)
	if err != nil {
		if err == task.ErrTaskNotFound {
			http.Error(w, "Task not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Cancel the specific tool execution
	// UI widget IDs are prefixed with "tool_" but backend uses raw callID
	actualCallID := callID
	if strings.HasPrefix(callID, "tool_") {
		actualCallID = strings.TrimPrefix(callID, "tool_")
	}
	cancelled := taskObj.CancelExecution(actualCallID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"cancelled": cancelled,
		"task_id":   taskID,
		"call_id":   callID,
	})

	if cancelled {
		slog.Info("Tool execution cancelled", "task_id", taskID, "call_id", callID)
	} else {
		slog.Debug("Tool cancellation failed - not found or already completed", "task_id", taskID, "call_id", callID)
	}
}

// handleGetSchema generates and returns JSON Schema for the config builder UI.
// This is a public endpoint (no authentication required) - schema is read-only metadata.
func (s *HTTPServer) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reflector := &jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
		FieldNameTag:              "yaml",
	}

	schema := reflector.Reflect(&config.AppConfig{})

	// Add metadata
	schema.ID = "https://hector.dev/schemas/config.json"
	schema.Title = "Hector Configuration Schema"
	schema.Description = "Complete configuration schema for Hector v2 agent framework"
	schema.Version = "http://json-schema.org/draft-07/schema#"

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour since it rarely changes
	w.Header().Set("Access-Control-Allow-Origin", "*")      // Allow CORS for public schema

	// Write response
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(schema); err != nil {
		slog.Error("Failed to encode schema", "error", err)
		http.Error(w, "Failed to generate schema", http.StatusInternalServerError)
	}
}
