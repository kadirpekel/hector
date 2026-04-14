package server

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/verikod/hector/pkg/app"
	"github.com/verikod/hector/pkg/auth"
	"github.com/verikod/hector/pkg/config"
	"github.com/verikod/hector/pkg/execution"
)

// DynamicAgentHandler routes requests to agent executors dynamically based on
// the current app context (tenant_id) and the agent name in the URL path.
type DynamicAgentHandler struct {
	appManager      *AppManager
	serverConfig    *config.ServerConfig
	authInterceptor *auth.Interceptor
	authValidator   auth.TokenValidator
	taskStore       a2asrv.TaskStore
	execProvider    execution.Provider
}

// NewDynamicAgentHandler creates a new dynamic agent handler.
func NewDynamicAgentHandler(
	appManager *AppManager,
	serverConfig *config.ServerConfig,
	authInterceptor *auth.Interceptor,
	authValidator auth.TokenValidator,
	taskStore a2asrv.TaskStore,
	execProvider execution.Provider,
) *DynamicAgentHandler {
	return &DynamicAgentHandler{
		appManager:      appManager,
		serverConfig:    serverConfig,
		authInterceptor: authInterceptor,
		authValidator:   authValidator,
		taskStore:       taskStore,
		execProvider:    execProvider,
	}
}

// ServeHTTP routes requests dynamically.
// Configured at path "/agents/".
func (h *DynamicAgentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Determine Identity & App Context
	// Since this handler is excluded from global auth middleware (to allow public agents),
	// we must perform our own auth validation to determine the caller's identity and app context.

	claims := resolveAuthClaims(r, h.authValidator)
	isAuthenticated := claims != nil

	// Resolve AppID:
	// 1. From authenticated claims (highest priority for multi-tenancy)
	// 2. From existing context (set by AppContextMiddleware, likely "default" if no auth)
	appID := app.IDFromContext(ctx)
	if isAuthenticated && claims.TenantID != "" {
		appID = claims.TenantID
	}

	// Admin Key/Token Convenience: Fall back to "default" app if authenticated but no tenant
	if isAuthenticated && appID == "" {
		appID = "default"
	}

	// 2. Parse Agent Name
	path := r.URL.Path
	if !strings.HasPrefix(path, "/agents/") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	subPath := strings.TrimPrefix(path, "/agents/")
	if subPath == "" {
		http.Error(w, "Agent name required", http.StatusNotFound)
		return
	}

	parts := strings.SplitN(subPath, "/", 2)
	agentName := parts[0]

	// 3. Get Agent Runtime (Executor + Card)
	// Now using the correctly resolved appID
	runtime, err := h.appManager.GetRuntime(ctx, appID, agentName)
	if err != nil {
		slog.Warn("Agent not found", "app", appID, "agent", agentName, "error", err)
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// 4. Visibility Check
	if runtime.Visibility == "private" {
		// Return 404 to hide existence
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	if runtime.Visibility == "internal" && !isAuthenticated {
		// Return 401 for internal agents if not authenticated
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 4. Determine Request Type and Delegate
	// Request path relative to agent root
	agentRelPath := "/"
	if len(parts) > 1 {
		agentRelPath += parts[1]
	}

	// Handle Agent Card request
	if r.Method == http.MethodGet && (agentRelPath == "/" || agentRelPath == "/.well-known/agent-card.json") {
		// Check Accept header for root path preference (JSON vs HTML/Text?)
		// a2a-go handles card serving.

		// Update Card URL to match current request context
		// This ensures correct self-reference
		card := prepareAgentCard(runtime.Card, r)

		handler := a2asrv.NewStaticAgentCardHandler(card)
		handler.ServeHTTP(w, r)
		return
	}

	// Handle A2A JSON-RPC Request
	// Usually POST to agent root
	if r.Method == http.MethodPost {
		// Create a2a-go handler stack
		// We could cache this in Runtime, but it's cheap to create if Executor is cached.

		var handlerOpts []a2asrv.RequestHandlerOption
		if h.taskStore != nil {
			handlerOpts = append(handlerOpts, a2asrv.WithTaskStore(h.taskStore))
		}
		if h.authInterceptor != nil {
			handlerOpts = append(handlerOpts, a2asrv.WithCallInterceptor(h.authInterceptor))
		}

		// 1. Create base handler from Executor
		var requestHandler a2asrv.RequestHandler = a2asrv.NewHandler(runtime.Executor, handlerOpts...)

		// 2. Wrap with QueuedRequestHandler if provider is available
		// This intercepts async requests (blocking=false) at the protocol level
		if h.execProvider != nil {
			requestHandler = NewQueuedRequestHandler(requestHandler, h.execProvider, appID, agentName)
		}

		// 3. Create transport handler (JSON-RPC)
		// Now using the queued handler which implements the RequestHandler interface
		jsonHandler := a2asrv.NewJSONRPCHandler(requestHandler)

		// Forward request
		jsonHandler.ServeHTTP(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
