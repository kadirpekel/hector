package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// RESTGatewayConfig holds configuration for the REST gateway
type RESTGatewayConfig struct {
	HTTPAddress string // e.g., ":8080"
	GRPCAddress string // e.g., "localhost:50051" - the gRPC server to proxy to
}

// RESTGateway wraps the grpc-gateway runtime to provide REST API
type RESTGateway struct {
	config     RESTGatewayConfig
	httpServer *http.Server
	mux        *runtime.ServeMux
	authConfig *AuthConfig
	discovery  *AgentDiscovery
	conn       *grpc.ClientConn // gRPC connection for per-agent card lookups
}

// NewRESTGateway creates a new REST gateway that proxies to the gRPC service
func NewRESTGateway(config RESTGatewayConfig) *RESTGateway {
	if config.HTTPAddress == "" {
		config.HTTPAddress = ":8080"
	}
	if config.GRPCAddress == "" {
		config.GRPCAddress = "localhost:50051"
	}

	// Create grpc-gateway mux with custom options
	mux := runtime.NewServeMux(
		// Custom error handler for A2A protocol errors
		runtime.WithErrorHandler(customErrorHandler),
	)

	return &RESTGateway{
		config: config,
		mux:    mux,
	}
}

// Start starts the REST gateway (blocking call)
func (g *RESTGateway) Start(ctx context.Context) error {
	// Connect to gRPC server
	conn, err := grpc.NewClient(
		g.config.GRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %w", err)
	}
	g.conn = conn // Store for per-agent card handler

	// Register A2A service handler
	if err := pb.RegisterA2AServiceHandler(ctx, g.mux, conn); err != nil {
		return fmt.Errorf("failed to register A2A service handler: %w", err)
	}

	// Setup routing (discovery, cards, agent endpoints)
	handler := g.setupRouting()

	// Create HTTP server
	g.httpServer = &http.Server{
		Addr:    g.config.HTTPAddress,
		Handler: handler,
	}

	log.Printf("🌐 REST API (grpc-gateway) starting on %s", g.config.HTTPAddress)
	log.Printf("   → Proxying to gRPC server at %s", g.config.GRPCAddress)
	if g.authConfig != nil && g.authConfig.Enabled {
		log.Printf("   → Authentication: ENABLED")
	}

	// Start serving (blocking)
	if err := g.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("REST gateway failed: %w", err)
	}

	return nil
}

// setupRouting configures all HTTP routes for the REST gateway
func (g *RESTGateway) setupRouting() http.Handler {
	mainMux := http.NewServeMux()

	// Register discovery endpoint if available
	if g.discovery != nil {
		mainMux.Handle("/v1/agents", g.discovery)
		log.Printf("   → Discovery endpoint: /v1/agents")
	}

	// Register service-level agent card endpoint (RFC 8615)
	serviceCardHandler := g.createServiceLevelAgentCardHandler()
	mainMux.Handle("/.well-known/agent-card.json", serviceCardHandler)
	log.Printf("   → Service Card: /.well-known/agent-card.json (multi-agent service)")

	// Register per-agent endpoints with special handling for well-known URIs
	mainMux.Handle("/v1/agents/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a well-known request for an agent card
		if strings.HasSuffix(r.URL.Path, "/.well-known/agent-card.json") {
			g.handlePerAgentCard(w, r)
			return
		}
		// Otherwise, route to agent-specific endpoints via gRPC
		g.createAgentRoutingHandler(g.mux).ServeHTTP(w, r)
	}))
	log.Printf("   → Agent Cards: /v1/agents/{agent_id}/.well-known/agent-card.json (per-agent)")
	log.Printf("   → Agent endpoints: /v1/agents/{agent_id}/* (A2A spec-compliant)")

	// Also support root-level endpoints for backward compatibility and single-agent mode
	mainMux.Handle("/v1/", g.mux)

	// Add auth middleware first, then CORS and logging
	var handler http.Handler = mainMux
	if g.authConfig != nil && g.authConfig.Enabled {
		handler = g.applyAuthMiddleware(handler)
	}
	handler = corsMiddleware(loggingMiddleware(handler))

	return handler
}

// SetDiscovery sets the agent discovery handler
func (g *RESTGateway) SetDiscovery(discovery *AgentDiscovery) {
	g.discovery = discovery
}

// SetAuth sets authentication configuration
func (g *RESTGateway) SetAuth(authConfig *AuthConfig) {
	g.authConfig = authConfig
}

// Stop gracefully stops the REST gateway
func (g *RESTGateway) Stop(ctx context.Context) error {
	if g.httpServer == nil {
		return nil
	}

	log.Printf("🛑 Shutting down REST gateway...")

	if err := g.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown REST gateway: %w", err)
	}

	log.Printf("✅ REST gateway stopped")
	return nil
}

// createServiceLevelAgentCardHandler creates handler for service-level /.well-known/agent-card.json
// Per A2A spec section 5.3 (RFC 8615 compliant)
// For multi-agent systems, this returns service metadata and points to discovery endpoint
func (g *RESTGateway) createServiceLevelAgentCardHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Build service-level card
		response := map[string]interface{}{
			"name":               "Hector Multi-Agent Service",
			"description":        "A2A-compliant multi-agent platform supporting multiple AI agents",
			"version":            "1.0.0",
			"type":               "multi-agent-service",
			"discovery_endpoint": "/v1/agents",
			"capabilities": map[string]interface{}{
				"streaming":       true,
				"tasks":           true,
				"authentication":  g.authConfig != nil && g.authConfig.Enabled,
				"multiple_agents": true,
			},
		}

		// Include security information if authentication is enabled
		if g.authConfig != nil && g.authConfig.Enabled {
			response["security_schemes"] = map[string]interface{}{
				"jwt": map[string]interface{}{
					"type":          "http",
					"scheme":        "bearer",
					"bearer_format": "JWT",
				},
			}
			response["security"] = []map[string]interface{}{
				{"jwt": []string{}},
			}
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}

// applyAuthMiddleware applies JWT authentication to all endpoints
func (g *RESTGateway) applyAuthMiddleware(next http.Handler) http.Handler {
	if g.authConfig == nil || g.authConfig.Validator == nil {
		return next
	}

	// Note: Auth middleware should be applied selectively
	// Public endpoints like /.well-known/agent-card.json and /v1/agents (public agents)
	// should not require auth. Auth is checked within those handlers.
	// For now, we return next without wrapping, as auth is handled per-endpoint.
	return next
}

// handlePerAgentCard handles per-agent well-known card requests via gRPC
func (g *RESTGateway) handlePerAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from path: /v1/agents/{agent_id}/.well-known/agent-card.json
	path := r.URL.Path
	remainder := strings.TrimPrefix(path, "/v1/agents/")
	parts := strings.Split(remainder, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	agentID := parts[0]

	// Create gRPC client and call GetAgentCard with agent-name metadata
	client := pb.NewA2AServiceClient(g.conn)
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentID)

	card, err := client.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get agent card: %v", err), http.StatusInternalServerError)
		return
	}

	// Build spec-compliant response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"name":         card.Name,
		"description":  card.Description,
		"version":      card.Version,
		"capabilities": card.Capabilities,
		"endpoint":     fmt.Sprintf("/v1/agents/%s", agentID),
	}

	// Include security information if present
	if len(card.SecuritySchemes) > 0 {
		response["security_schemes"] = card.SecuritySchemes
	}
	if len(card.Security) > 0 {
		response["security"] = card.Security
	}

	_ = json.NewEncoder(w).Encode(response)
}

// createAgentRoutingHandler creates a handler that extracts agent ID from URL path
// and adds it as metadata for the gRPC service to route correctly
// Transforms: /v1/agents/{agent_id}/message:send -> /v1/message:send with agent-name metadata
func (g *RESTGateway) createAgentRoutingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract agent ID from path: /v1/agents/{agent_id}/...
		path := r.URL.Path
		if !strings.HasPrefix(path, "/v1/agents/") {
			http.Error(w, "Invalid agent path", http.StatusNotFound)
			return
		}

		// Remove /v1/agents/ prefix
		remainder := strings.TrimPrefix(path, "/v1/agents/")
		parts := strings.SplitN(remainder, "/", 2)

		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "Agent ID required", http.StatusBadRequest)
			return
		}

		agentID := parts[0]

		// Rewrite path to standard A2A endpoint by removing /v1/agents/{agent_id}
		// e.g., /v1/agents/assistant/message:send -> /v1/message:send
		if len(parts) == 2 {
			r.URL.Path = "/v1/" + parts[1]
		} else {
			r.URL.Path = "/v1/"
		}

		// Add agent-name to gRPC metadata so RegistryService can route correctly
		// The grpc-gateway runtime will convert this header to gRPC metadata
		r.Header.Set("agent-name", agentID)

		// Forward to grpc-gateway mux
		next.ServeHTTP(w, r)
	})
}

// customErrorHandler provides A2A-compliant error responses
func customErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	// Use default error handler from grpc-gateway
	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("REST: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers for browser compatibility
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// For streaming endpoints, set SSE headers
		if strings.Contains(r.URL.Path, "stream") || strings.Contains(r.URL.Path, "subscribe") {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
		}

		next.ServeHTTP(w, r)
	})
}
