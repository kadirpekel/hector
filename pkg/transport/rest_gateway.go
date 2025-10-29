package transport

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed static/index.html
var webUIHTML []byte

// JSON-RPC 2.0 types and constants
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

type jsonrpcStreamWrapper struct {
	writer    http.ResponseWriter
	flusher   http.Flusher
	marshaler protojson.MarshalOptions
	id        interface{}
	context   context.Context
}

func (w *jsonrpcStreamWrapper) Send(resp *pb.StreamResponse) error {
	data, err := w.marshaler.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	jsonrpcResp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      w.id,
		Result:  json.RawMessage(data),
	}

	respData, err := json.Marshal(jsonrpcResp)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w.writer, "data: %s\n\n", respData)
	if err != nil {
		return err
	}
	w.flusher.Flush()
	return nil
}

func (w *jsonrpcStreamWrapper) Context() context.Context {
	return w.context
}

func (w *jsonrpcStreamWrapper) SendMsg(m interface{}) error {
	return nil
}

func (w *jsonrpcStreamWrapper) RecvMsg(m interface{}) error {
	return nil
}

func (w *jsonrpcStreamWrapper) SetHeader(metadata.MD) error {
	return nil
}

func (w *jsonrpcStreamWrapper) SendHeader(metadata.MD) error {
	return nil
}

func (w *jsonrpcStreamWrapper) SetTrailer(metadata.MD) {
}

type RESTGatewayConfig struct {
	HTTPAddress string
	GRPCAddress string
}

type RESTGateway struct {
	config      RESTGatewayConfig
	httpServer  *http.Server
	mux         *runtime.ServeMux
	authConfig  *AuthConfig
	discovery   *AgentDiscovery
	conn        *grpc.ClientConn
	service     pb.A2AServiceServer
	marshaler   protojson.MarshalOptions
	unmarshaler protojson.UnmarshalOptions
}

func NewRESTGateway(config RESTGatewayConfig) *RESTGateway {
	if config.HTTPAddress == "" {
		config.HTTPAddress = ":8080"
	}
	if config.GRPCAddress == "" {
		config.GRPCAddress = "localhost:9090"
	}

	mux := runtime.NewServeMux(
		runtime.WithErrorHandler(customErrorHandler),
	)

	return &RESTGateway{
		config: config,
		mux:    mux,
		marshaler: protojson.MarshalOptions{
			UseProtoNames:   false,
			EmitUnpopulated: true,
		},
		unmarshaler: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}
}

func (g *RESTGateway) Start(ctx context.Context) error {

	conn, err := grpc.NewClient(
		g.config.GRPCAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %w", err)
	}
	g.conn = conn

	if err := pb.RegisterA2AServiceHandler(ctx, g.mux, conn); err != nil {
		return fmt.Errorf("failed to register A2A service handler: %w", err)
	}

	handler := g.setupRouting()

	g.httpServer = &http.Server{
		Addr:    g.config.HTTPAddress,
		Handler: handler,
	}

	log.Printf("🌐 HTTP Server starting on %s", g.config.HTTPAddress)
	if g.authConfig != nil && g.authConfig.Enabled {
		log.Printf("   → Authentication: ENABLED")
	}

	if err := g.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("REST gateway failed: %w", err)
	}

	return nil
}

func (g *RESTGateway) setupRouting() http.Handler {
	r := chi.NewRouter()

	// Apply global middleware
	// Order: logging -> metrics -> cors -> auth
	r.Use(loggingMiddleware)
	r.Use(metricsMiddleware)
	r.Use(corsMiddleware)

	if g.authConfig != nil && g.authConfig.Enabled {
		r.Use(func(next http.Handler) http.Handler {
			return g.applyAuthMiddleware(next)
		})
	}

	// Root path: Web UI (GET) and JSON-RPC (POST)
	r.Get("/", g.handleWebUI)
	r.Post("/", g.handleJSONRPC)
	log.Printf("   → Root (GET: Web UI, POST: JSON-RPC): /")

	// JSON-RPC streaming endpoint
	r.Post("/stream", g.handleJSONRPCStream)
	log.Printf("   → JSON-RPC Streaming: /stream")

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Metrics
	r.Get("/metrics", MetricsHandler().ServeHTTP)
	log.Printf("   → Prometheus metrics: /metrics")

	// REST API: Discovery
	if g.discovery != nil {
		r.Get("/v1/agents", g.discovery.ServeHTTP)
		log.Printf("   → Discovery endpoint: /v1/agents")
	}

	// REST API: Service card
	serviceCardHandler := g.createServiceLevelAgentCardHandler()
	r.Get("/.well-known/agent-card.json", serviceCardHandler.ServeHTTP)
	log.Printf("   → Service Card: /.well-known/agent-card.json")

	// REST API: Agent-specific routes
	r.Get("/v1/agents/{agent}/.well-known/agent-card.json", g.handlePerAgentCard)
	log.Printf("   → Agent Cards: /v1/agents/{agent}/.well-known/agent-card.json")

	r.Post("/v1/agents/{agent}/message:stream", g.handleStreamingMessageSSE)
	r.Post("/v1/agents/{agent}/message:send", g.handleSendMessage)
	log.Printf("   → Agent endpoints: /v1/agents/{agent}/*")

	// Mount grpc-gateway mux for other /v1/* routes
	r.Mount("/v1/", g.createAgentRoutingHandler(g.mux))

	return r
}

func (g *RESTGateway) SetDiscovery(discovery *AgentDiscovery) {
	g.discovery = discovery
}

func (g *RESTGateway) SetAuth(authConfig *AuthConfig) {
	g.authConfig = authConfig
}

func (g *RESTGateway) SetService(service pb.A2AServiceServer) {
	g.service = service
}

func (g *RESTGateway) Stop(ctx context.Context) error {
	if g.httpServer == nil {
		return nil
	}

	log.Printf("Shutting down REST gateway...")

	if err := g.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown REST gateway: %w", err)
	}

	log.Printf("REST gateway stopped")
	return nil
}

func (g *RESTGateway) createServiceLevelAgentCardHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// A2A spec: agent cards must be retrieved via HTTP GET only
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		agentName := r.URL.Query().Get("agent")

		if agentName != "" {

			g.handleAgentCardByName(w, r, agentName)
			return
		}

		if g.discovery != nil {
			agentNames := g.discovery.service.ListAgents()
			if len(agentNames) > 0 {

				g.handleAgentCardByName(w, r, agentNames[0])
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

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

func (g *RESTGateway) handleAgentCardByName(w http.ResponseWriter, r *http.Request, agentName string) {

	client := pb.NewA2AServiceClient(g.conn)
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentName)

	card, err := client.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get agent card for '%s': %v", agentName, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Use the current host and add agent parameter to root path
	// JSON-RPC is now served at POST /
	baseURL := fmt.Sprintf("http://%s/?agent=%s", r.Host, agentName)

	response := map[string]interface{}{
		"name":         card.Name,
		"description":  card.Description,
		"version":      card.Version,
		"url":          baseURL,
		"capabilities": card.Capabilities,

		"defaultInputModes":  card.DefaultInputModes,
		"defaultOutputModes": card.DefaultOutputModes,
		"skills":             card.Skills,
	}

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

func (g *RESTGateway) applyAuthMiddleware(next http.Handler) http.Handler {
	if g.authConfig == nil || g.authConfig.Validator == nil {
		return next
	}

	return next
}

func (g *RESTGateway) handlePerAgentCard(w http.ResponseWriter, r *http.Request) {
	// A2A spec: agent cards must be retrieved via HTTP GET only
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get agent name from URL parameter (chi router extracts this)
	agentName := chi.URLParam(r, "agent")
	if agentName == "" {
		http.Error(w, "Agent name required", http.StatusBadRequest)
		return
	}

	client := pb.NewA2AServiceClient(g.conn)
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentName)

	card, err := client.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get agent card: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"name":         card.Name,
		"description":  card.Description,
		"version":      card.Version,
		"capabilities": card.Capabilities,
		"endpoint":     fmt.Sprintf("/v1/agents/%s", agentName),
	}

	if len(card.SecuritySchemes) > 0 {
		response["security_schemes"] = card.SecuritySchemes
	}
	if len(card.Security) > 0 {
		response["security"] = card.Security
	}

	_ = json.NewEncoder(w).Encode(response)
}

func (g *RESTGateway) createAgentRoutingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Support two path formats per A2A spec:
		// 1. /v1/agents/{agent}/resource (agent in path - Section 3.5.3 preferred)
		// 2. /v1/resource (agent in header - proto annotations compatibility)

		path := r.URL.Path

		if strings.HasPrefix(path, "/v1/agents/") {
			// Format: /v1/agents/{agent}/resource
			remainder := strings.TrimPrefix(path, "/v1/agents/")
			parts := strings.SplitN(remainder, "/", 2)

			if len(parts) == 0 || parts[0] == "" {
				http.Error(w, "Agent name required", http.StatusBadRequest)
				return
			}

			agentName := parts[0]

			// Rewrite URL to remove /agents/{agent} prefix
			if len(parts) == 2 {
				r.URL.Path = "/v1/" + parts[1]
			} else {
				r.URL.Path = "/v1/"
			}

			r.Header.Set("grpc-metadata-agent-name", agentName)
		} else {
			// Format: /v1/resource (e.g., /v1/tasks, /v1/card)
			// Extract agent from header or query parameter
			agentName := r.Header.Get("agent-name")
			if agentName == "" {
				agentName = r.URL.Query().Get("agent")
			}

			// Set agent in gRPC metadata if specified
			// For single-agent mode, this is optional - AgentRouter handles fallback
			if agentName != "" {
				r.Header.Set("grpc-metadata-agent-name", agentName)
			}
		}

		next.ServeHTTP(w, r)
	})
}

func customErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {

	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method != "PRI" {
			log.Printf("REST: %s %s", r.Method, r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}

func (g *RESTGateway) handleStreamingMessageSSE(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		g.sendSSEError(w, "Method not allowed")
		return
	}

	// Get agent name from URL parameter (chi router extracts this)
	agentName := chi.URLParam(r, "agent")
	if agentName == "" {
		g.sendSSEError(w, "Agent name required")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		g.sendSSEError(w, fmt.Sprintf("Failed to read request body: %v", err))
		return
	}
	defer r.Body.Close()

	bodyBytes = applyA2AFieldMapping(bodyBytes)

	var req pb.SendMessageRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(bodyBytes, &req); err != nil {
		g.sendSSEError(w, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	log.Printf("REST SSE: agent=%s", agentName)

	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentName)

	streamWrapper := &restStreamWrapper{
		writer:  w,
		flusher: w.(http.Flusher),
	}

	if g.service != nil {
		err := g.service.SendStreamingMessage(&req, streamWrapper)
		if err != nil {
			g.sendSSEError(w, fmt.Sprintf("Service error: %v", err))
			return
		}
	} else {

		client := pb.NewA2AServiceClient(g.conn)
		stream, err := client.SendStreamingMessage(ctx, &req)
		if err != nil {
			g.sendSSEError(w, fmt.Sprintf("Failed to start stream: %v", err))
			return
		}

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

	streamWrapper.sendCompletionEvent()
}

func (g *RESTGateway) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get agent name from URL parameter (chi router extracts this)
	agentName := chi.URLParam(r, "agent")
	if agentName == "" {
		http.Error(w, "Agent name required", http.StatusBadRequest)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	bodyBytes = applyA2AFieldMapping(bodyBytes)

	var req pb.SendMessageRequest
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("REST: agent=%s", agentName)

	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentName)

	client := pb.NewA2AServiceClient(g.conn)
	resp, err := client.SendMessage(ctx, &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Service error: %v", err), http.StatusInternalServerError)
		return
	}

	marshaler := protojson.MarshalOptions{
		UseProtoNames:   false,
		EmitUnpopulated: false,
	}
	data, err := marshaler.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

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

type restStreamWrapper struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	context context.Context
}

func (w *restStreamWrapper) Send(resp *pb.StreamResponse) error {

	marshaler := protojson.MarshalOptions{
		UseProtoNames:   false,
		EmitUnpopulated: false,
	}
	data, err := marshaler.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	_, err = fmt.Fprintf(w.writer, "event: message\ndata: %s\n\n", data)
	if err != nil {
		return err
	}

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

// ===== Web UI Handler =====

func (g *RESTGateway) handleWebUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(webUIHTML)
}

// ===== JSON-RPC Handlers =====

func (g *RESTGateway) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		g.sendJSONRPCError(w, nil, ParseError, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		g.sendJSONRPCError(w, nil, ParseError, "Invalid JSON")
		return
	}

	if rpcReq.JSONRPC != "2.0" {
		g.sendJSONRPCError(w, rpcReq.ID, InvalidRequest, "Invalid JSON-RPC version")
		return
	}

	log.Printf("JSON-RPC: method=%s id=%v", rpcReq.Method, rpcReq.ID)

	mappedParams := applyA2AFieldMapping(rpcReq.Params)
	result, err := g.handleJSONRPCMethod(r.Context(), rpcReq.Method, mappedParams, r)
	if err != nil {
		g.sendJSONRPCError(w, rpcReq.ID, InternalError, err.Error())
		return
	}

	result = g.transformResultForA2A(result)

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      rpcReq.ID,
		Result:  result,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (g *RESTGateway) handleJSONRPCMethod(ctx context.Context, method string, params json.RawMessage, r *http.Request) (interface{}, error) {
	var md metadata.MD
	if agentName := r.URL.Query().Get("agent"); agentName != "" {
		md = metadata.Pairs("agent-name", agentName)
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	switch method {
	case "message/send":
		return g.handleSendMessageRPC(ctx, params)
	case "message/stream":
		return nil, fmt.Errorf("use /rpc/stream endpoint for streaming messages")
	case "tasks/get":
		return g.handleGetTask(ctx, params)
	case "tasks/list":
		return g.handleListTasks(ctx, params)
	case "tasks/cancel":
		return g.handleCancelTask(ctx, params)
	case "tasks/resubscribe":
		return nil, fmt.Errorf("tasks/resubscribe is only available via streaming endpoint")
	case "agent/getAuthenticatedExtendedCard":
		return g.handleGetAuthenticatedExtendedCard(ctx, params)
	default:
		return nil, fmt.Errorf("method not found: %s", method)
	}
}

func (g *RESTGateway) handleJSONRPCStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		g.sendJSONRPCSSEError(w, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		g.sendJSONRPCSSEError(w, "Invalid JSON")
		return
	}

	if rpcReq.JSONRPC != "2.0" {
		g.sendJSONRPCSSEError(w, "Invalid JSON-RPC version")
		return
	}

	mappedParams := applyA2AFieldMapping(rpcReq.Params)

	var req pb.SendMessageRequest
	if err := g.unmarshaler.Unmarshal(mappedParams, &req); err != nil {
		g.sendJSONRPCSSEError(w, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	log.Printf("JSON-RPC Stream: method=%s id=%v", rpcReq.Method, rpcReq.ID)

	ctx := r.Context()
	var agentName string
	var md metadata.MD

	agentName = r.URL.Query().Get("agent")
	if agentName != "" {
		log.Printf("JSON-RPC Stream: routing to agent '%s'", agentName)
		md = metadata.Pairs("agent-name", agentName)
	} else if req.Request != nil && req.Request.ContextId != "" {
		parts := strings.Split(req.Request.ContextId, ":")
		if len(parts) >= 2 && parts[0] != "" {
			agentName = parts[0]
			log.Printf("JSON-RPC Stream: routing to agent '%s' from context_id", agentName)
			md = metadata.Pairs("agent-name", agentName)
		}
	}

	if md != nil {
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	streamWrapper := &jsonrpcStreamWrapper{
		writer:    w,
		flusher:   w.(http.Flusher),
		marshaler: g.marshaler,
		id:        rpcReq.ID,
		context:   ctx,
	}

	if err := g.service.SendStreamingMessage(&req, streamWrapper); err != nil {
		g.sendJSONRPCSSEError(w, fmt.Sprintf("Service error: %v", err))
		return
	}
}

func (g *RESTGateway) sendJSONRPCSSEError(w http.ResponseWriter, message string) {
	sseData := map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    InternalError,
			"message": message,
		},
	}

	data, _ := json.Marshal(sseData)
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *RESTGateway) sendJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// ===== JSON-RPC Method Implementations =====

func (g *RESTGateway) handleSendMessageRPC(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.SendMessageRequest
	if err := g.unmarshaler.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	resp, err := g.service.SendMessage(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("service error: %w", err)
	}

	jsonData, err := g.marshaler.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to interface: %w", err)
	}

	return result, nil
}

func (g *RESTGateway) handleGetTask(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.GetTaskRequest
	if err := g.unmarshaler.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	resp, err := g.service.GetTask(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("service error: %w", err)
	}

	jsonData, err := g.marshaler.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to interface: %w", err)
	}

	return result, nil
}

func (g *RESTGateway) handleListTasks(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.ListTasksRequest
	if len(params) > 0 && string(params) != "null" {
		if err := g.unmarshaler.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	resp, err := g.service.ListTasks(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("service error: %w", err)
	}

	jsonData, err := g.marshaler.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to interface: %w", err)
	}

	return result, nil
}

func (g *RESTGateway) handleCancelTask(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.CancelTaskRequest
	if err := g.unmarshaler.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	resp, err := g.service.CancelTask(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("service error: %w", err)
	}

	jsonData, err := g.marshaler.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to interface: %w", err)
	}

	return result, nil
}

func (g *RESTGateway) handleGetAuthenticatedExtendedCard(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.GetAgentCardRequest

	if len(params) > 0 && string(params) != "null" {
		if err := g.unmarshaler.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	resp, err := g.service.GetAgentCard(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("service error: %w", err)
	}

	jsonData, err := g.marshaler.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to interface: %w", err)
	}

	return result, nil
}

func (g *RESTGateway) transformResultForA2A(result interface{}) interface{} {
	jsonData, err := json.Marshal(result)
	if err != nil {
		return result
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &resultMap); err != nil {
		return result
	}

	if taskObj, ok := resultMap["task"].(map[string]interface{}); ok {
		resultMap = taskObj
	}

	g.transformMapForA2A(resultMap)

	if _, hasId := resultMap["id"]; hasId {
		if _, hasStatus := resultMap["status"]; hasStatus {
			resultMap["kind"] = "task"
		}
	} else if _, hasMessageId := resultMap["messageId"]; hasMessageId {
		resultMap["kind"] = "message"
	}

	return resultMap
}

func (g *RESTGateway) transformMapForA2A(obj map[string]interface{}) {
	fieldMappings := map[string]string{
		"content":         "parts",
		"message_id":      "messageId",
		"context_id":      "contextId",
		"task_id":         "taskId",
		"artifact_id":     "artifactId",
		"msg":             "message",
		"status_update":   "statusUpdate",
		"artifact_update": "artifactUpdate",
	}

	for oldKey, newKey := range fieldMappings {
		if val, ok := obj[oldKey]; ok {
			obj[newKey] = val
			delete(obj, oldKey)
		}
	}

	if role, ok := obj["role"].(string); ok {
		switch role {
		case "ROLE_USER":
			obj["role"] = "user"
		case "ROLE_AGENT":
			obj["role"] = "agent"
		case "ROLE_SYSTEM":
			obj["role"] = "system"
		}
	}

	if state, ok := obj["state"].(string); ok {
		switch state {
		case "TASK_STATE_SUBMITTED":
			obj["state"] = "submitted"
		case "TASK_STATE_WORKING":
			obj["state"] = "working"
		case "TASK_STATE_INPUT_REQUIRED":
			obj["state"] = "input-required"
		case "TASK_STATE_COMPLETED":
			obj["state"] = "completed"
		case "TASK_STATE_CANCELED":
			obj["state"] = "canceled"
		case "TASK_STATE_FAILED":
			obj["state"] = "failed"
		case "TASK_STATE_REJECTED":
			obj["state"] = "rejected"
		case "TASK_STATE_AUTH_REQUIRED":
			obj["state"] = "auth-required"
		case "TASK_STATE_UNKNOWN":
			obj["state"] = "unknown"
		}
	}

	for _, val := range obj {
		switch v := val.(type) {
		case map[string]interface{}:
			g.transformMapForA2A(v)
		case []interface{}:
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					g.transformMapForA2A(m)
				}
			}
		}
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
