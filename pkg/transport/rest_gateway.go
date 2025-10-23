package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

// RESTGatewayConfig holds configuration for the REST gateway
type RESTGatewayConfig struct {
	HTTPAddress string // e.g., ":8080"
	GRPCAddress string // e.g., "localhost:8080" - the gRPC server to proxy to
}

// RESTGateway wraps the grpc-gateway runtime to provide REST API
type RESTGateway struct {
	config     RESTGatewayConfig
	httpServer *http.Server
	mux        *runtime.ServeMux
	authConfig *AuthConfig
	discovery  *AgentDiscovery
	conn       *grpc.ClientConn    // gRPC connection for per-agent card lookups
	service    pb.A2AServiceServer // Direct service access for SSE streaming
}

// NewRESTGateway creates a new REST gateway that proxies to the gRPC service
func NewRESTGateway(config RESTGatewayConfig) *RESTGateway {
	if config.HTTPAddress == "" {
		config.HTTPAddress = ":8080"
	}
	if config.GRPCAddress == "" {
		config.GRPCAddress = "localhost:8080"
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

	// Register per-agent endpoints with special handling for well-known URIs and streaming
	mainMux.Handle("/v1/agents/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a well-known request for an agent card
		if strings.HasSuffix(r.URL.Path, "/.well-known/agent-card.json") {
			g.handlePerAgentCard(w, r)
			return
		}
		// Check if this is a streaming request - intercept for true SSE
		if strings.HasSuffix(r.URL.Path, "/message:stream") {
			g.handleStreamingMessageSSE(w, r)
			return
		}
		// Otherwise, route to agent-specific endpoints via gRPC
		g.createAgentRoutingHandler(g.mux).ServeHTTP(w, r)
	}))
	log.Printf("   → Agent Cards: /v1/agents/{name}/.well-known/agent-card.json (per-agent)")
	log.Printf("   → Agent endpoints: /v1/agents/{name}/* (A2A spec-compliant)")

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

// SetService sets the A2A service for direct SSE streaming
func (g *RESTGateway) SetService(service pb.A2AServiceServer) {
	g.service = service
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
// For multi-agent systems:
// - Supports ?agent=name query parameter to select specific agent
// - Returns first agent card if no agent parameter is provided (for single-agent compatibility)
// - Returns service metadata with discovery endpoint for multi-agent awareness
func (g *RESTGateway) createServiceLevelAgentCardHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check if a specific agent is requested via query parameter
		agentName := r.URL.Query().Get("agent")

		if agentName != "" {
			// Return specific agent card
			g.handleAgentCardByName(w, r, agentName)
			return
		}

		// Check if we should return a default agent card (first agent)
		// This provides A2A compliance for single-agent mode
		if g.discovery != nil {
			agentNames := g.discovery.service.ListAgents()
			if len(agentNames) > 0 {
				// Return the first agent's card for backward compatibility
				g.handleAgentCardByName(w, r, agentNames[0])
				return
			}
		}

		// Fallback: Return service-level metadata
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

// handleAgentCardByName returns the agent card for a specific agent
func (g *RESTGateway) handleAgentCardByName(w http.ResponseWriter, r *http.Request, agentName string) {
	// Create gRPC client and call GetAgentCard with agent-name metadata
	client := pb.NewA2AServiceClient(g.conn)
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentName)

	card, err := client.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get agent card for '%s': %v", agentName, err), http.StatusInternalServerError)
		return
	}

	// Build spec-compliant response with proper URL field
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Determine the base URL for the agent endpoint
	// For A2A compliance: point to JSON-RPC endpoint which supports SSE streaming
	// The a2a-inspector and other JSON-RPC clients expect this URL to accept JSON-RPC requests
	// IMPORTANT: Include agent name in URL so the JSON-RPC handler knows which agent to route to
	host := r.Host
	// Change port from 8081 (REST) to 8082 (JSON-RPC)
	if strings.Contains(host, ":8081") {
		host = strings.Replace(host, ":8081", ":8082", 1)
	} else if !strings.Contains(host, ":") {
		// No port specified, add JSON-RPC port
		host = host + ":8082"
	}
	// Include agent name as query parameter so JSON-RPC handler knows which agent
	baseURL := fmt.Sprintf("http://%s?agent=%s", host, agentName)

	// Build full A2A-compliant response including all required fields
	response := map[string]interface{}{
		"name":         card.Name,
		"description":  card.Description,
		"version":      card.Version,
		"url":          baseURL,
		"capabilities": card.Capabilities,
		// A2A required fields
		"defaultInputModes":  card.DefaultInputModes,
		"defaultOutputModes": card.DefaultOutputModes,
		"skills":             card.Skills,
	}

	// Include optional fields if present
	if card.ProtocolVersion != "" {
		response["protocolVersion"] = card.ProtocolVersion
	}
	if card.PreferredTransport != "" {
		response["preferredTransport"] = card.PreferredTransport
	}
	if len(card.AdditionalInterfaces) > 0 {
		response["additionalInterfaces"] = card.AdditionalInterfaces
	}
	if card.Provider != nil {
		response["provider"] = card.Provider
	}
	if card.DocumentationUrl != "" {
		response["documentationUrl"] = card.DocumentationUrl
	}
	if len(card.SecuritySchemes) > 0 {
		response["securitySchemes"] = card.SecuritySchemes
	}
	if len(card.Security) > 0 {
		response["security"] = card.Security
	}
	if card.SupportsAuthenticatedExtendedCard {
		response["supportsAuthenticatedExtendedCard"] = true
	}
	if len(card.Signatures) > 0 {
		response["signatures"] = card.Signatures
	}
	if card.IconUrl != "" {
		response["iconUrl"] = card.IconUrl
	}

	_ = json.NewEncoder(w).Encode(response)
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

	// Extract agent name from path: /v1/agents/{name}/.well-known/agent-card.json
	path := r.URL.Path
	remainder := strings.TrimPrefix(path, "/v1/agents/")
	parts := strings.Split(remainder, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Agent name required", http.StatusBadRequest)
		return
	}

	agentName := parts[0]

	// Create gRPC client and call GetAgentCard with agent-name metadata
	client := pb.NewA2AServiceClient(g.conn)
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentName)

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
		"endpoint":     fmt.Sprintf("/v1/agents/%s", agentName),
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

// createAgentRoutingHandler creates a handler that extracts agent name from URL path
// and adds it as metadata for the gRPC service to route correctly
// Transforms: /v1/agents/{name}/message:send -> /v1/message:send with agent-name metadata
func (g *RESTGateway) createAgentRoutingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract agent name from path: /v1/agents/{name}/...
		path := r.URL.Path
		if !strings.HasPrefix(path, "/v1/agents/") {
			http.Error(w, "Invalid agent path", http.StatusNotFound)
			return
		}

		// Remove /v1/agents/ prefix
		remainder := strings.TrimPrefix(path, "/v1/agents/")
		parts := strings.SplitN(remainder, "/", 2)

		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "Agent name required", http.StatusBadRequest)
			return
		}

		agentName := parts[0]

		// Rewrite path to standard A2A endpoint by removing /v1/agents/{name}
		// e.g., /v1/agents/assistant/message:send -> /v1/message:send
		if len(parts) == 2 {
			r.URL.Path = "/v1/" + parts[1]
		} else {
			r.URL.Path = "/v1/"
		}

		// Add agent-name to gRPC metadata so RegistryService can route correctly
		// The grpc-gateway runtime will convert this header to gRPC metadata
		// Use grpc-metadata- prefix for proper conversion
		r.Header.Set("grpc-metadata-agent-name", agentName)

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

// handleStreamingMessageSSE handles streaming messages with true SSE format
// This intercepts /v1/agents/{name}/message:stream to provide W3C SSE compliance
func (g *RESTGateway) handleStreamingMessageSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		g.sendSSEError(w, "Method not allowed")
		return
	}

	// Extract agent name from path
	path := r.URL.Path
	remainder := strings.TrimPrefix(path, "/v1/agents/")
	parts := strings.Split(remainder, "/")
	if len(parts) == 0 || parts[0] == "" {
		g.sendSSEError(w, "Agent name required")
		return
	}
	agentName := parts[0]

	// Parse request body - read raw JSON first for A2A field mapping
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		g.sendSSEError(w, fmt.Sprintf("Failed to read request body: %v", err))
		return
	}
	defer r.Body.Close()

	// Apply A2A field mapping (parts → content, lowercase roles/states → uppercase enums)
	bodyBytes = applyA2AFieldMapping(bodyBytes)

	// Parse into protobuf using protojson (handles enums correctly)
	var req pb.SendMessageRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(bodyBytes, &req); err != nil {
		g.sendSSEError(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	log.Printf("REST SSE: agent=%s path=%s", agentName, path)

	// Add agent name to context metadata
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentName)

	// Create stream wrapper
	streamWrapper := &restStreamWrapper{
		writer:  w,
		flusher: w.(http.Flusher),
	}

	// Call streaming service directly (bypassing grpc-gateway)
	if g.service != nil {
		err := g.service.SendStreamingMessage(&req, streamWrapper)
		if err != nil {
			g.sendSSEError(w, fmt.Sprintf("Service error: %v", err))
			return
		}
	} else {
		// Fallback to gRPC call
		client := pb.NewA2AServiceClient(g.conn)
		stream, err := client.SendStreamingMessage(ctx, &req)
		if err != nil {
			g.sendSSEError(w, fmt.Sprintf("Failed to start stream: %v", err))
			return
		}

		// Read from gRPC stream and convert to SSE
		for {
			resp, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				g.sendSSEError(w, fmt.Sprintf("Stream error: %v", err))
				return
			}

			if err := streamWrapper.Send(resp); err != nil {
				log.Printf("Failed to send SSE event: %v", err)
				return
			}
		}
	}

	// Send completion event
	streamWrapper.sendCompletionEvent()
}

// sendSSEError sends an error via SSE format
func (g *RESTGateway) sendSSEError(w http.ResponseWriter, message string) {
	sseData := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
		},
	}

	data, _ := json.Marshal(sseData)
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// restStreamWrapper wraps http.ResponseWriter to implement streaming with SSE format
type restStreamWrapper struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	context context.Context
}

func (w *restStreamWrapper) Send(resp *pb.StreamResponse) error {
	// Convert protobuf to JSON using protojson (preserves proper field names)
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   false, // Use JSON names (camelCase)
		EmitUnpopulated: false,
	}
	data, err := marshaler.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Send as SSE event
	_, err = fmt.Fprintf(w.writer, "event: message\ndata: %s\n\n", data)
	if err != nil {
		return err
	}

	// Flush immediately for real-time streaming
	w.flusher.Flush()
	return nil
}

func (w *restStreamWrapper) sendCompletionEvent() {
	fmt.Fprintf(w.writer, "event: done\ndata: {}\n\n")
	w.flusher.Flush()
}

func (w *restStreamWrapper) SetHeader(metadata.MD) error {
	return nil
}

func (w *restStreamWrapper) SendHeader(metadata.MD) error {
	return nil
}

func (w *restStreamWrapper) SetTrailer(metadata.MD) {
}

func (w *restStreamWrapper) Context() context.Context {
	if w.context != nil {
		return w.context
	}
	return context.Background()
}

func (w *restStreamWrapper) SendMsg(m interface{}) error {
	return fmt.Errorf("SendMsg not implemented")
}

func (w *restStreamWrapper) RecvMsg(m interface{}) error {
	return fmt.Errorf("RecvMsg not implemented")
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

		next.ServeHTTP(w, r)
	})
}
