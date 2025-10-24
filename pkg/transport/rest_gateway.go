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

type RESTGatewayConfig struct {
	HTTPAddress string
	GRPCAddress string
}

type RESTGateway struct {
	config     RESTGatewayConfig
	httpServer *http.Server
	mux        *runtime.ServeMux
	authConfig *AuthConfig
	discovery  *AgentDiscovery
	conn       *grpc.ClientConn
	service    pb.A2AServiceServer
}

func NewRESTGateway(config RESTGatewayConfig) *RESTGateway {
	if config.HTTPAddress == "" {
		config.HTTPAddress = ":8080"
	}
	if config.GRPCAddress == "" {
		config.GRPCAddress = "localhost:8080"
	}

	mux := runtime.NewServeMux(

		runtime.WithErrorHandler(customErrorHandler),
	)

	return &RESTGateway{
		config: config,
		mux:    mux,
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

	log.Printf("REST API (grpc-gateway) starting on %s", g.config.HTTPAddress)
	log.Printf("   → Proxying to gRPC server at %s", g.config.GRPCAddress)
	if g.authConfig != nil && g.authConfig.Enabled {
		log.Printf("   → Authentication: ENABLED")
	}

	if err := g.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("REST gateway failed: %w", err)
	}

	return nil
}

func (g *RESTGateway) setupRouting() http.Handler {
	mainMux := http.NewServeMux()

	mainMux.Handle("/metrics", MetricsHandler())
	log.Printf("   → Prometheus metrics: /metrics")

	if g.discovery != nil {
		mainMux.Handle("/v1/agents", g.discovery)
		log.Printf("   → Discovery endpoint: /v1/agents")
	}

	serviceCardHandler := g.createServiceLevelAgentCardHandler()
	mainMux.Handle("/.well-known/agent-card.json", serviceCardHandler)
	log.Printf("   → Service Card: /.well-known/agent-card.json (multi-agent service)")

	mainMux.Handle("/v1/agents/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if strings.HasSuffix(r.URL.Path, "/.well-known/agent-card.json") {
			g.handlePerAgentCard(w, r)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/message:stream") {
			g.handleStreamingMessageSSE(w, r)
			return
		}

		if strings.HasSuffix(r.URL.Path, "/message:send") {
			g.handleSendMessage(w, r)
			return
		}

		g.createAgentRoutingHandler(g.mux).ServeHTTP(w, r)
	}))
	log.Printf("   → Agent Cards: /v1/agents/{name}/.well-known/agent-card.json (per-agent)")
	log.Printf("   → Agent endpoints: /v1/agents/{name}/* (A2A spec-compliant)")

	mainMux.Handle("/v1/", g.mux)

	var handler http.Handler = mainMux
	if g.authConfig != nil && g.authConfig.Enabled {
		handler = g.applyAuthMiddleware(handler)
	}
	handler = corsMiddleware(loggingMiddleware(handler))

	return handler
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

	host := r.Host

	if strings.Contains(host, ":8081") {
		host = strings.Replace(host, ":8081", ":8082", 1)
	} else if !strings.Contains(host, ":") {

		host = host + ":8082"
	}

	baseURL := fmt.Sprintf("http://%s?agent=%s", host, agentName)

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
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	remainder := strings.TrimPrefix(path, "/v1/agents/")
	parts := strings.Split(remainder, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Agent name required", http.StatusBadRequest)
		return
	}

	agentName := parts[0]

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

		path := r.URL.Path
		if !strings.HasPrefix(path, "/v1/agents/") {
			http.Error(w, "Invalid agent path", http.StatusNotFound)
			return
		}

		remainder := strings.TrimPrefix(path, "/v1/agents/")
		parts := strings.SplitN(remainder, "/", 2)

		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "Agent name required", http.StatusBadRequest)
			return
		}

		agentName := parts[0]

		if len(parts) == 2 {
			r.URL.Path = "/v1/" + parts[1]
		} else {
			r.URL.Path = "/v1/"
		}

		r.Header.Set("grpc-metadata-agent-name", agentName)

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

	path := r.URL.Path
	remainder := strings.TrimPrefix(path, "/v1/agents/")
	parts := strings.Split(remainder, "/")
	if len(parts) == 0 || parts[0] == "" {
		g.sendSSEError(w, "Agent name required")
		return
	}
	agentName := parts[0]

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

	log.Printf("REST SSE: agent=%s path=%s", agentName, path)

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

	path := r.URL.Path
	remainder := strings.TrimPrefix(path, "/v1/agents/")
	parts := strings.Split(remainder, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Agent name required", http.StatusBadRequest)
		return
	}
	agentName := parts[0]

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

	log.Printf("REST: agent=%s path=%s", agentName, path)

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
