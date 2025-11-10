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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/agui"
	aguipb "github.com/kadirpekel/hector/pkg/agui/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed static/index.html
var webUIHTML []byte

// Context key type for auth claims
type contextKey string

const authClaimsKey contextKey = "auth_claims"

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

type RESTGatewayConfig struct {
	HTTPAddress string
	GRPCAddress string
	BaseURL     string // Server's base URL for agent card URLs
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
		config.GRPCAddress = "localhost:50051"
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

	// Root path: Web UI only
	r.Get("/", g.handleWebUI)
	log.Printf("   → Web UI: /")

	// Health check (operational endpoint, not agent-specific)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Metrics (operational endpoint, not agent-specific)
	r.Get("/metrics", MetricsHandler().ServeHTTP)
	log.Printf("   → Prometheus metrics: /metrics")

	// REST API: Discovery (the only v1/ root endpoint)
	if g.discovery != nil {
		r.Get("/v1/agents", g.discovery.ServeHTTP)
		log.Printf("   → Discovery endpoint: /v1/agents")
	}

	// REST API: All agent-specific routes under /v1/agents/{agent}/
	// This follows A2A protocol best practice: each agent gets its own URL space
	r.Route("/v1/agents/{agent}", func(r chi.Router) {
		// Agent card (A2A spec required endpoint)
		// Per A2A spec Section 5.3: MUST be at /.well-known/agent-card.json
		r.Get("/.well-known/agent-card.json", g.handlePerAgentCard)

		// Convenient shortcut: GET /v1/agents/{agent}/ also returns agent card
		r.Get("/", g.handlePerAgentCard)

		// Messages (A2A core operations)
		r.Post("/message:send", g.handleSendMessage)
		r.Post("/message:stream", g.handleStreamingMessageSSE)

		// JSON-RPC endpoint (agent-scoped)
		// Handles JSON-RPC 2.0 requests for this specific agent
		// Note: GET returns agent card, POST handles JSON-RPC
		r.Post("/", g.handleJSONRPCAgentScoped)

		// JSON-RPC streaming (agent-scoped)
		r.Post("/stream", g.handleJSONRPCStreamAgentScoped)

		// Task operations (A2A spec core methods - HTTP+JSON/REST transport)
		// Per A2A spec Section 3.5.3 and 7.3-7.5
		r.Get("/tasks", g.handleListTasks)                           // Optional: tasks/list
		r.Get("/tasks/{taskID}", g.handleGetTask)                    // Core: tasks/get
		r.Post("/tasks/{taskID}:cancel", g.handleCancelTask)         // Core: tasks/cancel
		r.Get("/tasks/{taskID}:subscribe", g.handleTaskSubscription) // Optional: tasks/subscribe
	})

	log.Printf("   → Agent-scoped endpoints: /v1/agents/{agent}/*")
	log.Printf("     • Agent root: / (GET: agent card, POST: JSON-RPC)")
	log.Printf("     • Agent card: /.well-known/agent-card.json (A2A spec compliant)")
	log.Printf("     • Messages: /message:send, /message:stream")
	log.Printf("     • JSON-RPC streaming: /stream (POST, agent-scoped)")
	log.Printf("     • Tasks: /tasks (GET: list), /tasks/{id} (GET: retrieve), /tasks/{id}:cancel (POST: cancel), /tasks/{id}:subscribe (GET: subscribe)")
	log.Printf("   → A2A Protocol Compliance: Core methods (message/send, tasks/get, tasks/cancel) + Optional (tasks/list, tasks/subscribe)")

	return r
}

func (g *RESTGateway) SetDiscovery(discovery *AgentDiscovery) {
	g.discovery = discovery
}

func (g *RESTGateway) SetAuth(authConfig *AuthConfig) {
	g.authConfig = authConfig
}

// validateAgentID checks if the given agent ID exists in the registry.
// Returns true if valid, false otherwise. On false, writes an appropriate error response.
func (g *RESTGateway) validateAgentID(agentID string, w http.ResponseWriter, sendSSE bool) bool {
	if g.discovery == nil {
		return true // If discovery is not available, skip validation
	}

	validAgents := g.discovery.service.ListAgents()
	for _, validID := range validAgents {
		if validID == agentID {
			return true
		}
	}

	// Agent not found - send appropriate error
	errorMsg := fmt.Sprintf("Agent '%s' not found. Available agents: %v", agentID, validAgents)
	if sendSSE {
		g.sendSSEError(w, errorMsg)
	} else {
		http.Error(w, errorMsg, http.StatusNotFound)
	}
	return false
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

func (g *RESTGateway) applyAuthMiddleware(next http.Handler) http.Handler {
	if g.authConfig == nil || g.authConfig.Validator == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public endpoints that don't require authentication
		publicPaths := []string{
			"/",        // Web UI
			"/health",  // Health checks
			"/metrics", // Prometheus metrics
		}

		// Check if this is a public endpoint
		for _, path := range publicPaths {
			if r.URL.Path == path {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Agent discovery endpoint - handles auth internally for visibility filtering
		// (public agents visible without auth, internal/private need auth)
		if r.URL.Path == "/v1/agents" {
			next.ServeHTTP(w, r)
			return
		}

		// All other endpoints require authentication when global auth is enabled
		// Extract Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Unauthorized","message":"Missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Extract token (handle both "Bearer token" and "token" formats)
		tokenString := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		// Validate token using JWT validator
		claims, err := g.authConfig.Validator.ValidateToken(r.Context(), tokenString)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, fmt.Sprintf(`{"error":"Unauthorized","message":"Invalid token: %s"}`, err.Error()), http.StatusUnauthorized)
			return
		}

		// Add claims to context for downstream handlers
		ctx := context.WithValue(r.Context(), authClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (g *RESTGateway) handlePerAgentCard(w http.ResponseWriter, r *http.Request) {
	// A2A spec: agent cards must be retrieved via HTTP GET only
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get agent ID from URL parameter (chi router extracts this)
	agentID := chi.URLParam(r, "agent")
	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	// Validate agent exists before routing (fail fast)
	if !g.validateAgentID(agentID, w, false) {
		return
	}

	client := pb.NewA2AServiceClient(g.conn)
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentID)

	// Forward Authorization header to gRPC metadata (for auth interceptor)
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)
	}

	card, err := client.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get agent card: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Build the agent URL - all transports now use agent-scoped pattern
	// The URL is simply the agent's base path: /v1/agents/{agent}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	agentURL := fmt.Sprintf("%s://%s/v1/agents/%s", scheme, r.Host, agentID)

	response := map[string]interface{}{
		"name":         card.Name,
		"description":  card.Description,
		"version":      card.Version,
		"url":          agentURL, // Agent-scoped URL (works for all transports)
		"capabilities": card.Capabilities,
	}

	if card.PreferredTransport != "" {
		response["preferred_transport"] = card.PreferredTransport
	}
	if len(card.SecuritySchemes) > 0 {
		response["security_schemes"] = card.SecuritySchemes
	}
	if len(card.Security) > 0 {
		response["security"] = card.Security
	}

	_ = json.NewEncoder(w).Encode(response)
}

//nolint:unused // Reserved for future agent routing middleware
func (g *RESTGateway) createAgentRoutingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This handler is now mounted under /v1/agents/{agent}/
		// The chi router extracts the {agent} param, and we receive the remainder
		// e.g., incoming /v1/agents/my-agent/tasks/123 becomes /tasks/123 here

		// Get agent ID from chi router
		agentName := chi.URLParam(r, "agent")

		// Forward Authorization header to gRPC metadata (for auth interceptor)
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			r.Header.Set("grpc-metadata-authorization", authHeader)
		}

		// If we have an agent name from the URL, use it
		if agentName != "" {
			// Rewrite path to gRPC gateway format: /v1/{resource}
			// e.g., /tasks/123 -> /v1/tasks/123
			if !strings.HasPrefix(r.URL.Path, "/v1/") {
				r.URL.Path = "/v1" + r.URL.Path
			}

			// Set agent in gRPC metadata
			r.Header.Set("grpc-metadata-agent-name", agentName)
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

	// Get agent ID from URL parameter (chi router extracts this)
	agentID := chi.URLParam(r, "agent")
	if agentID == "" {
		g.sendSSEError(w, "Agent ID required")
		return
	}

	// Validate agent exists before routing (fail fast)
	if !g.validateAgentID(agentID, w, true) {
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

	log.Printf("REST SSE: agent=%s", agentID)

	// Create incoming metadata for the gRPC service (server-side)
	md := metadata.Pairs("agent-name", agentID)

	// Forward Authorization header to gRPC metadata (for auth interceptor)
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		md = metadata.Join(md, metadata.Pairs("authorization", authHeader))
	}

	ctx := metadata.NewIncomingContext(r.Context(), md)

	// Detect AG-UI format preference
	// Check Accept header for "application/x-agui-events" or query parameter "format=agui"
	useAGUI := false
	acceptHeader := r.Header.Get("Accept")
	if strings.Contains(acceptHeader, "application/x-agui-events") ||
		strings.Contains(acceptHeader, "application/agui+json") {
		useAGUI = true
	}

	// Also check query parameter
	if r.URL.Query().Get("format") == "agui" {
		useAGUI = true
	}

	// Generate a message ID for AG-UI events
	messageID := ""
	if useAGUI {
		messageID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	}

	streamWrapper := &restStreamWrapper{
		writer:    w,
		flusher:   w.(http.Flusher),
		context:   ctx,
		useAGUI:   useAGUI,
		messageID: messageID,
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

	// Get agent ID from URL parameter (chi router extracts this)
	agentID := chi.URLParam(r, "agent")
	if agentID == "" {
		http.Error(w, "Agent ID required", http.StatusBadRequest)
		return
	}

	// Validate agent exists before routing (fail fast)
	if !g.validateAgentID(agentID, w, false) {
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

	log.Printf("REST: agent=%s", agentID)

	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentID)

	// Forward Authorization header to gRPC metadata (for auth interceptor)
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)
	}

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
	writer    http.ResponseWriter
	flusher   http.Flusher
	context   context.Context
	useAGUI   bool
	converter *agui.Converter
	messageID string
	contextID string
	taskID    string
	inMessage bool
}

func (w *restStreamWrapper) Send(resp *pb.StreamResponse) error {
	// If AG-UI mode is enabled, convert A2A to AG-UI events
	if w.useAGUI {
		return w.sendAsAGUI(resp)
	}

	// A2A native format (default)
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

// sendAsAGUI converts A2A StreamResponse to AG-UI events and sends them
func (w *restStreamWrapper) sendAsAGUI(resp *pb.StreamResponse) error {
	events := w.convertToAGUIEvents(resp)

	marshaler := protojson.MarshalOptions{
		UseProtoNames:   false,
		EmitUnpopulated: false,
	}

	for _, event := range events {
		data, err := marshaler.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal AG-UI event: %w", err)
		}

		// Determine SSE event type
		eventType := w.getAGUIEventType(event.Type)

		_, err = fmt.Fprintf(w.writer, "event: %s\ndata: %s\n\n", eventType, string(data))
		if err != nil {
			return err
		}

		w.flusher.Flush()
	}

	return nil
}

// convertToAGUIEvents converts an A2A StreamResponse to AG-UI events
func (w *restStreamWrapper) convertToAGUIEvents(resp *pb.StreamResponse) []*aguipb.AGUIEvent {
	var events []*aguipb.AGUIEvent

	switch payload := resp.Payload.(type) {
	case *pb.StreamResponse_Task:
		task := payload.Task
		w.taskID = task.Id
		w.contextID = task.ContextId

		if task.Status != nil {
			switch task.Status.State {
			case pb.TaskState_TASK_STATE_SUBMITTED:
				events = append(events, agui.NewTaskStartEvent(task.Id, task.ContextId, ""))
				events = append(events, agui.NewMessageStartEvent(w.messageID, w.contextID, w.taskID, "agent"))
				w.inMessage = true
				w.converter = agui.NewConverter(w.messageID, w.contextID, w.taskID)

			case pb.TaskState_TASK_STATE_WORKING:
				if !w.inMessage {
					events = append(events, agui.NewMessageStartEvent(w.messageID, w.contextID, w.taskID, "agent"))
					w.inMessage = true
					w.converter = agui.NewConverter(w.messageID, w.contextID, w.taskID)
				}
				events = append(events, agui.NewTaskUpdateEvent(task.Id, "working", nil))

			case pb.TaskState_TASK_STATE_COMPLETED:
				if w.converter != nil {
					closeEvents := w.converter.CloseCurrentBlock()
					events = append(events, closeEvents...)
				}
				if w.inMessage {
					events = append(events, agui.NewMessageStopEvent(w.messageID))
					w.inMessage = false
				}
				events = append(events, agui.NewTaskCompleteEvent(task.Id, nil))

			case pb.TaskState_TASK_STATE_FAILED:
				if w.converter != nil {
					closeEvents := w.converter.CloseCurrentBlock()
					events = append(events, closeEvents...)
				}
				if w.inMessage {
					events = append(events, agui.NewMessageStopEvent(w.messageID))
					w.inMessage = false
				}
				errorMsg := "Task failed"
				if task.Status.Update != nil && len(task.Status.Update.Parts) > 0 {
					if text := task.Status.Update.Parts[0].GetText(); text != "" {
						errorMsg = text
					}
				}
				events = append(events, agui.NewTaskErrorEvent(task.Id, errorMsg, "TASK_FAILED", nil))
			}
		}

	case *pb.StreamResponse_Msg:
		msg := payload.Msg

		if msg.ContextId != "" {
			w.contextID = msg.ContextId
		}
		if msg.TaskId != "" {
			w.taskID = msg.TaskId
		}

		if !w.inMessage {
			role := "agent"
			if msg.Role == pb.Role_ROLE_USER {
				role = "user"
			}
			events = append(events, agui.NewMessageStartEvent(w.messageID, w.contextID, w.taskID, role))
			w.inMessage = true
			w.converter = agui.NewConverter(w.messageID, w.contextID, w.taskID)
		}

		for _, part := range msg.Parts {
			partEvents := w.converter.ConvertPart(part)
			events = append(events, partEvents...)
		}

	case *pb.StreamResponse_StatusUpdate:
		update := payload.StatusUpdate
		w.taskID = update.TaskId
		w.contextID = update.ContextId

		status := strings.ToLower(update.Status.State.String())
		status = strings.TrimPrefix(status, "task_state_")

		if update.Final {
			if w.converter != nil {
				closeEvents := w.converter.CloseCurrentBlock()
				events = append(events, closeEvents...)
			}
			if w.inMessage {
				events = append(events, agui.NewMessageStopEvent(w.messageID))
				w.inMessage = false
			}
		}

		events = append(events, agui.NewTaskUpdateEvent(update.TaskId, status, nil))

	case *pb.StreamResponse_ArtifactUpdate:
		artifact := payload.ArtifactUpdate.Artifact
		if artifact != nil {
			if !w.inMessage {
				events = append(events, agui.NewMessageStartEvent(w.messageID, w.contextID, w.taskID, "agent"))
				w.inMessage = true
				w.converter = agui.NewConverter(w.messageID, w.contextID, w.taskID)
			}

			for _, part := range artifact.Parts {
				partEvents := w.converter.ConvertPart(part)
				events = append(events, partEvents...)
			}
		}
	}

	return events
}

// getAGUIEventType returns the SSE event type name for an AG-UI event
// Converts: AGUI_EVENT_TYPE_MESSAGE_START -> message_start
func (w *restStreamWrapper) getAGUIEventType(eventType aguipb.AGUIEventType) string {
	enumStr := eventType.String()

	// Handle UNSPECIFIED case before trimming
	if enumStr == "AGUI_EVENT_TYPE_UNSPECIFIED" {
		return "agui_event"
	}

	// Remove AGUI_EVENT_TYPE_ prefix
	if strings.HasPrefix(enumStr, "AGUI_EVENT_TYPE_") {
		enumStr = strings.TrimPrefix(enumStr, "AGUI_EVENT_TYPE_")
	}

	// Handle empty string case
	if enumStr == "" {
		return "agui_event"
	}

	// Convert to lowercase
	return strings.ToLower(enumStr)
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

// handleJSONRPCAgentScoped handles JSON-RPC requests for a specific agent
// This is the agent-scoped version at /v1/agents/{agent}/
func (g *RESTGateway) handleJSONRPCAgentScoped(w http.ResponseWriter, r *http.Request) {
	// Get agent ID from URL parameter
	agentID := chi.URLParam(r, "agent")
	if agentID == "" {
		g.sendJSONRPCError(w, nil, InvalidRequest, "Agent ID required")
		return
	}

	// Validate agent exists
	if !g.validateAgentID(agentID, w, false) {
		g.sendJSONRPCError(w, nil, InvalidRequest, fmt.Sprintf("Agent not found: %s", agentID))
		return
	}

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

	log.Printf("JSON-RPC (agent-scoped): agent=%s method=%s id=%v", agentID, rpcReq.Method, rpcReq.ID)

	// Create context with agent metadata
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentID)

	// Forward Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)
	}

	// Handle JSON-RPC methods
	mappedParams := applyA2AFieldMapping(rpcReq.Params)

	var result interface{}

	switch rpcReq.Method {
	case "message/send":
		// Non-streaming message send
		var req pb.SendMessageRequest
		if err := g.unmarshaler.Unmarshal(mappedParams, &req); err != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InvalidParams, fmt.Sprintf("Invalid params: %v", err))
			return
		}

		resp, svcErr := g.service.SendMessage(ctx, &req)
		if svcErr != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Service error: %v", svcErr))
			return
		}

		jsonData, marshalErr := g.marshaler.Marshal(resp)
		if marshalErr != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Failed to marshal response: %v", marshalErr))
			return
		}

		if unmarshalErr := json.Unmarshal(jsonData, &result); unmarshalErr != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Failed to unmarshal response: %v", unmarshalErr))
			return
		}

	case "tasks/get":
		// Get task by ID
		var req pb.GetTaskRequest
		if err := g.unmarshaler.Unmarshal(mappedParams, &req); err != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InvalidParams, fmt.Sprintf("Invalid params: %v", err))
			return
		}

		task, svcErr := g.service.GetTask(ctx, &req)
		if svcErr != nil {
			// Convert gRPC error to JSON-RPC error
			if strings.Contains(svcErr.Error(), "not found") {
				g.sendJSONRPCError(w, rpcReq.ID, InvalidParams, fmt.Sprintf("Task not found: %v", svcErr))
			} else if strings.Contains(svcErr.Error(), "Unimplemented") {
				g.sendJSONRPCError(w, rpcReq.ID, InvalidRequest, "Task tracking not enabled")
			} else {
				g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Service error: %v", svcErr))
			}
			return
		}

		jsonData, marshalErr := g.marshaler.Marshal(task)
		if marshalErr != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Failed to marshal response: %v", marshalErr))
			return
		}

		if unmarshalErr := json.Unmarshal(jsonData, &result); unmarshalErr != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Failed to unmarshal response: %v", unmarshalErr))
			return
		}

	case "tasks/cancel":
		// Cancel a task
		var req pb.CancelTaskRequest
		if err := g.unmarshaler.Unmarshal(mappedParams, &req); err != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InvalidParams, fmt.Sprintf("Invalid params: %v", err))
			return
		}

		task, svcErr := g.service.CancelTask(ctx, &req)
		if svcErr != nil {
			// Convert gRPC error to JSON-RPC error
			if strings.Contains(svcErr.Error(), "not found") {
				g.sendJSONRPCError(w, rpcReq.ID, InvalidParams, fmt.Sprintf("Task not found: %v", svcErr))
			} else if strings.Contains(svcErr.Error(), "Unimplemented") {
				g.sendJSONRPCError(w, rpcReq.ID, InvalidRequest, "Task tracking not enabled")
			} else {
				g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Service error: %v", svcErr))
			}
			return
		}

		jsonData, marshalErr := g.marshaler.Marshal(task)
		if marshalErr != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Failed to marshal response: %v", marshalErr))
			return
		}

		if unmarshalErr := json.Unmarshal(jsonData, &result); unmarshalErr != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Failed to unmarshal response: %v", unmarshalErr))
			return
		}

	case "tasks/list":
		// List tasks with filtering
		var req pb.ListTasksRequest
		if err := g.unmarshaler.Unmarshal(mappedParams, &req); err != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InvalidParams, fmt.Sprintf("Invalid params: %v", err))
			return
		}

		resp, svcErr := g.service.ListTasks(ctx, &req)
		if svcErr != nil {
			if strings.Contains(svcErr.Error(), "Unimplemented") {
				g.sendJSONRPCError(w, rpcReq.ID, InvalidRequest, "Task tracking not enabled")
			} else {
				g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Service error: %v", svcErr))
			}
			return
		}

		jsonData, marshalErr := g.marshaler.Marshal(resp)
		if marshalErr != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Failed to marshal response: %v", marshalErr))
			return
		}

		if unmarshalErr := json.Unmarshal(jsonData, &result); unmarshalErr != nil {
			g.sendJSONRPCError(w, rpcReq.ID, InternalError, fmt.Sprintf("Failed to unmarshal response: %v", unmarshalErr))
			return
		}

	case "message/stream":
		g.sendJSONRPCError(w, rpcReq.ID, InvalidRequest, "Use /stream endpoint for streaming messages")
		return

	case "tasks/resubscribe":
		g.sendJSONRPCError(w, rpcReq.ID, InvalidRequest, "Use /stream endpoint for task resubscription")
		return

	default:
		g.sendJSONRPCError(w, rpcReq.ID, MethodNotFound, fmt.Sprintf("Method not found: %s", rpcReq.Method))
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

// handleJSONRPCStreamAgentScoped handles streaming JSON-RPC for a specific agent
func (g *RESTGateway) handleJSONRPCStreamAgentScoped(w http.ResponseWriter, r *http.Request) {
	// Get agent ID from URL parameter
	agentID := chi.URLParam(r, "agent")
	if agentID == "" {
		g.sendSSEError(w, "Agent ID required")
		return
	}

	// Validate agent exists
	if !g.validateAgentID(agentID, w, true) {
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		g.sendSSEError(w, "Streaming not supported")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		g.sendSSEError(w, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		g.sendSSEError(w, "Invalid JSON")
		return
	}

	log.Printf("JSON-RPC Stream (agent-scoped): agent=%s method=%s", agentID, rpcReq.Method)

	// Create context with agent metadata
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentID)

	// Forward Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)
	}

	// Handle streaming methods
	if rpcReq.Method == "message/stream" || rpcReq.Method == "tasks/resubscribe" {
		// Reuse the existing streaming logic with the context that has agent metadata
		var req pb.SendMessageRequest
		mappedParams := applyA2AFieldMapping(rpcReq.Params)
		log.Printf("[DEBUG] JSON-RPC params after mapping: %s", string(mappedParams))
		if err := g.unmarshaler.Unmarshal(mappedParams, &req); err != nil {
			g.sendSSEError(w, fmt.Sprintf("Invalid params: %v", err))
			return
		}
		log.Printf("[DEBUG] Unmarshaled request - TaskId: %s", req.Request.GetTaskId())

		// Set contextId as session ID for memory persistence
		if req.Request != nil && req.Request.ContextId != "" {
			ctx = context.WithValue(ctx, agent.SessionIDKey, req.Request.ContextId)
		}

		// Detect AG-UI format preference
		useAGUI := false
		acceptHeader := r.Header.Get("Accept")
		if strings.Contains(acceptHeader, "application/x-agui-events") ||
			strings.Contains(acceptHeader, "application/agui+json") {
			useAGUI = true
		}

		// Also check query parameter
		if r.URL.Query().Get("format") == "agui" {
			useAGUI = true
		}

		// Generate a message ID for AG-UI events
		messageID := ""
		if useAGUI {
			messageID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
		}

		// Create stream wrapper for SSE
		streamWrapper := &restStreamWrapper{
			writer:    w,
			flusher:   flusher,
			context:   ctx,
			useAGUI:   useAGUI,
			messageID: messageID,
		}

		// Call the streaming service
		if err := g.service.SendStreamingMessage(&req, streamWrapper); err != nil {
			g.sendSSEError(w, fmt.Sprintf("Service error: %v", err))
			return
		}
		return
	}

	g.sendSSEError(w, fmt.Sprintf("Method %s does not support streaming", rpcReq.Method))
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

// ===== A2A Field Transformation =====

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

// ===== A2A Task Handlers (HTTP+JSON/REST Transport) =====
// These handlers implement the A2A protocol Section 3.5.3 HTTP+JSON/REST transport
// and Section 7.3-7.5 task methods

// handleGetTask implements GET /v1/agents/{agent}/tasks/{taskID}
// A2A spec: Section 7.3 - tasks/get (CORE METHOD - MUST implement)
func (g *RESTGateway) handleGetTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get agent ID from URL parameter
	agentID := chi.URLParam(r, "agent")
	if agentID == "" {
		http.Error(w, `{"error":"Agent ID required"}`, http.StatusBadRequest)
		return
	}

	// Get task ID from URL parameter
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, `{"error":"Task ID required"}`, http.StatusBadRequest)
		return
	}

	// Validate agent exists
	if !g.validateAgentID(agentID, w, false) {
		return
	}

	log.Printf("REST: GET /v1/agents/%s/tasks/%s", agentID, taskID)

	// Create context with agent metadata
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentID)

	// Forward Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)
	}

	// Parse optional query parameters
	historyLength := int32(0)
	if histLenStr := r.URL.Query().Get("history_length"); histLenStr != "" {
		if histLen, err := fmt.Sscanf(histLenStr, "%d", &historyLength); err == nil && histLen == 1 {
			// Successfully parsed - historyLength is now set
			_ = histLen // Acknowledge successful parse
		}
	}

	// Call gRPC service
	client := pb.NewA2AServiceClient(g.conn)
	task, err := client.GetTask(ctx, &pb.GetTaskRequest{
		Name:          fmt.Sprintf("tasks/%s", taskID),
		HistoryLength: historyLength,
	})
	if err != nil {
		// Convert gRPC error to appropriate HTTP status
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, fmt.Sprintf(`{"error":"Task not found","task_id":"%s"}`, taskID), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf(`{"error":"Service error: %v"}`, err), http.StatusInternalServerError)
		}
		return
	}

	// Marshal response using protojson with A2A field mappings
	data, err := g.marshaler.Marshal(task)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to marshal response: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Note: We do NOT apply A2A field transformations here because:
	// 1. The HTTP client (CLI) expects native protobuf JSON format
	// 2. A2A transformation should only be applied for JSON-RPC and external A2A clients
	// 3. The REST endpoints serve both internal clients and A2A clients

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// handleCancelTask implements POST /v1/agents/{agent}/tasks/{taskID}:cancel
// A2A spec: Section 7.5 - tasks/cancel (CORE METHOD - MUST implement)
func (g *RESTGateway) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Get agent ID from URL parameter
	agentID := chi.URLParam(r, "agent")
	if agentID == "" {
		http.Error(w, `{"error":"Agent ID required"}`, http.StatusBadRequest)
		return
	}

	// Get task ID from URL parameter
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		http.Error(w, `{"error":"Task ID required"}`, http.StatusBadRequest)
		return
	}

	// Validate agent exists
	if !g.validateAgentID(agentID, w, false) {
		return
	}

	log.Printf("REST: POST /v1/agents/%s/tasks/%s:cancel", agentID, taskID)

	// Create context with agent metadata
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentID)

	// Forward Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)
	}

	// Call gRPC service
	client := pb.NewA2AServiceClient(g.conn)
	task, err := client.CancelTask(ctx, &pb.CancelTaskRequest{
		Name: fmt.Sprintf("tasks/%s", taskID),
	})
	if err != nil {
		// Convert gRPC error to appropriate HTTP status
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, fmt.Sprintf(`{"error":"Task not found","task_id":"%s"}`, taskID), http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf(`{"error":"Service error: %v"}`, err), http.StatusInternalServerError)
		}
		return
	}

	// Marshal response using protojson with A2A field mappings
	data, err := g.marshaler.Marshal(task)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to marshal response: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Note: We do NOT apply A2A field transformations here because:
	// 1. The HTTP client (CLI) expects native protobuf JSON format
	// 2. A2A transformation should only be applied for JSON-RPC and external A2A clients
	// 3. The REST endpoints serve both internal clients and A2A clients

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// handleListTasks implements GET /v1/agents/{agent}/tasks
// A2A spec: Section 7.4 - tasks/list (OPTIONAL METHOD - MAY implement)
func (g *RESTGateway) handleListTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get agent ID from URL parameter
	agentID := chi.URLParam(r, "agent")
	if agentID == "" {
		http.Error(w, `{"error":"Agent ID required"}`, http.StatusBadRequest)
		return
	}

	// Validate agent exists
	if !g.validateAgentID(agentID, w, false) {
		return
	}

	log.Printf("REST: GET /v1/agents/%s/tasks", agentID)

	// Create context with agent metadata
	ctx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentID)

	// Forward Authorization header
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authHeader)
	}

	// Parse query parameters
	req := &pb.ListTasksRequest{}

	if contextID := r.URL.Query().Get("context_id"); contextID != "" {
		req.ContextId = contextID
	}

	if pageSize := r.URL.Query().Get("page_size"); pageSize != "" {
		var ps int32
		if _, err := fmt.Sscanf(pageSize, "%d", &ps); err == nil {
			req.PageSize = ps
		}
	}

	if pageToken := r.URL.Query().Get("page_token"); pageToken != "" {
		req.PageToken = pageToken
	}

	if historyLength := r.URL.Query().Get("history_length"); historyLength != "" {
		var hl int32
		if _, err := fmt.Sscanf(historyLength, "%d", &hl); err == nil {
			req.HistoryLength = hl
		}
	}

	if includeArtifacts := r.URL.Query().Get("include_artifacts"); includeArtifacts == "true" {
		req.IncludeArtifacts = true
	}

	// Call gRPC service
	client := pb.NewA2AServiceClient(g.conn)
	resp, err := client.ListTasks(ctx, req)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Service error: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Marshal response using protojson with A2A field mappings
	data, err := g.marshaler.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to marshal response: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Note: We do NOT apply A2A field transformations here because:
	// 1. The HTTP client (CLI) expects native protobuf JSON format
	// 2. A2A transformation should only be applied for JSON-RPC and external A2A clients
	// 3. The REST endpoints serve both internal clients and A2A clients

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// handleTaskSubscription implements GET /v1/agents/{agent}/tasks/{taskID}:subscribe
// A2A spec: Section 7.6 - tasks/subscribe (OPTIONAL METHOD - MAY implement)
func (g *RESTGateway) handleTaskSubscription(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers for streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		g.sendSSEError(w, "Streaming not supported")
		return
	}

	// Get agent ID from URL parameter
	agentID := chi.URLParam(r, "agent")
	if agentID == "" {
		g.sendSSEError(w, "Agent ID required")
		return
	}

	// Get task ID from URL parameter
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		g.sendSSEError(w, "Task ID required")
		return
	}

	// Validate agent exists
	if !g.validateAgentID(agentID, w, true) {
		return
	}

	log.Printf("REST SSE Task Subscription: agent=%s task=%s", agentID, taskID)

	// Create incoming metadata for the gRPC service (server-side)
	md := metadata.Pairs("agent-name", agentID)

	// Forward Authorization header to gRPC metadata (for auth interceptor)
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		md = metadata.Join(md, metadata.Pairs("authorization", authHeader))
	}

	ctx := metadata.NewIncomingContext(r.Context(), md)

	// Detect AG-UI format preference
	useAGUI := false
	acceptHeader := r.Header.Get("Accept")
	if strings.Contains(acceptHeader, "application/x-agui-events") ||
		strings.Contains(acceptHeader, "application/agui+json") {
		useAGUI = true
	}
	if r.URL.Query().Get("format") == "agui" {
		useAGUI = true
	}

	req := &pb.TaskSubscriptionRequest{
		Name: fmt.Sprintf("tasks/%s", taskID),
	}

	// Try to call service directly if available (same pattern as handleStreamingMessageSSE)
	if g.service != nil {
		// Create stream wrapper with incoming metadata context (for service call)
		streamWrapper := &restStreamWrapper{
			writer:  w,
			flusher: flusher,
			context: ctx, // Use incoming metadata context for service call
			useAGUI: useAGUI,
		}

		err := g.service.TaskSubscription(req, streamWrapper)
		if err != nil {
			g.sendSSEError(w, fmt.Sprintf("Service error: %v", err))
			return
		}

		streamWrapper.sendCompletionEvent()
		return
	} else {
		// Fallback to gRPC client call
		// Create context with agent metadata for gRPC client call
		clientCtx := metadata.AppendToOutgoingContext(r.Context(), "agent-name", agentID)

		// Forward Authorization header
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			clientCtx = metadata.AppendToOutgoingContext(clientCtx, "authorization", authHeader)
		}

		// Call gRPC TaskSubscription service
		client := pb.NewA2AServiceClient(g.conn)
		stream, err := client.TaskSubscription(clientCtx, req)
		if err != nil {
			// Convert gRPC error to appropriate SSE error
			if strings.Contains(err.Error(), "not found") {
				g.sendSSEError(w, fmt.Sprintf("Task not found: %s", taskID))
			} else if strings.Contains(err.Error(), "Unimplemented") {
				g.sendSSEError(w, "Task subscription not supported")
			} else {
				g.sendSSEError(w, fmt.Sprintf("Failed to subscribe to task: %v", err))
			}
			return
		}

		// Create stream wrapper with request context (for cancellation checks)
		streamWrapper := &restStreamWrapper{
			writer:  w,
			flusher: flusher,
			context: r.Context(), // Use original request context for cancellation
			useAGUI: useAGUI,
		}

		// Stream task updates
		// The stream will be cancelled when the HTTP request context is cancelled
		for {
			// Check if context is cancelled (client disconnected)
			select {
			case <-r.Context().Done():
				log.Printf("Client disconnected from task subscription: agent=%s task=%s", agentID, taskID)
				return
			default:
			}

			resp, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					// Stream ended normally (task completed or channel closed)
					break
				}
				// Check if error is due to context cancellation
				if r.Context().Err() != nil {
					log.Printf("Stream cancelled due to context: agent=%s task=%s", agentID, taskID)
					return
				}
				// Other errors - send error event and close
				g.sendSSEError(w, fmt.Sprintf("Stream error: %v", err))
				return
			}

			if err := streamWrapper.Send(resp); err != nil {
				log.Printf("Failed to send SSE event: %v", err)
				return
			}
		}

		streamWrapper.sendCompletionEvent()
		return
	}
}
