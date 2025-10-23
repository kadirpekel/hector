package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

// JSONRPCConfig holds configuration for the JSON-RPC server
type JSONRPCConfig struct {
	HTTPAddress string // e.g., ":8081"
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// JSONRPCHandler provides JSON-RPC 2.0 interface to A2A service
type JSONRPCHandler struct {
	config      JSONRPCConfig
	service     pb.A2AServiceServer
	httpServer  *http.Server
	marshaler   protojson.MarshalOptions
	unmarshaler protojson.UnmarshalOptions
	authConfig  *AuthConfig // Auth configuration
}

// NewJSONRPCHandler creates a new JSON-RPC handler
func NewJSONRPCHandler(config JSONRPCConfig, service pb.A2AServiceServer) *JSONRPCHandler {
	if config.HTTPAddress == "" {
		config.HTTPAddress = ":8081"
	}

	return &JSONRPCHandler{
		config:  config,
		service: service,
		marshaler: protojson.MarshalOptions{
			UseProtoNames:   false, // Use JSON names (camelCase) not proto names (snake_case)
			EmitUnpopulated: true,  // Include zero values (e.g., false for bool) - required for A2A "final" field
		},
		unmarshaler: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}
}

// SetAuth sets authentication configuration
func (h *JSONRPCHandler) SetAuth(authConfig *AuthConfig) {
	h.authConfig = authConfig
}

// Start starts the JSON-RPC server (blocking call)
func (h *JSONRPCHandler) Start() error {
	mux := http.NewServeMux()

	// Root endpoint - routes to streaming or non-streaming based on method
	mux.HandleFunc("/", h.handleRootJSONRPC)

	// JSON-RPC endpoint (non-streaming)
	mux.HandleFunc("/rpc", h.handleJSONRPC)

	// SSE streaming endpoint for message/stream
	mux.HandleFunc("/rpc/stream", h.handleStreamingMessage)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Add middleware (auth, CORS, logging)
	var handler http.Handler = mux
	if h.authConfig != nil && h.authConfig.Enabled && h.authConfig.Validator != nil {
		handler = h.authConfig.Validator.HTTPMiddleware(handler)
	}
	handler = corsMiddleware(loggingMiddleware(handler))

	h.httpServer = &http.Server{
		Addr:    h.config.HTTPAddress,
		Handler: handler,
	}

	log.Printf("🔌 JSON-RPC API starting on %s", h.config.HTTPAddress)
	log.Printf("   → Endpoint: POST %s/rpc", h.config.HTTPAddress)
	log.Printf("   → Streaming: POST %s/rpc/stream (SSE)", h.config.HTTPAddress)

	if err := h.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("JSON-RPC server failed: %w", err)
	}

	return nil
}

// Stop gracefully stops the JSON-RPC server
func (h *JSONRPCHandler) Stop(ctx context.Context) error {
	if h.httpServer == nil {
		return nil
	}

	log.Printf("🛑 Shutting down JSON-RPC server...")

	if err := h.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown JSON-RPC server: %w", err)
	}

	log.Printf("✅ JSON-RPC server stopped")
	return nil
}

// handleRootJSONRPC handles JSON-RPC requests at the root path
// This is what a2a-inspector and other JSON-RPC clients use
// Routes to streaming or non-streaming based on the method
func (h *JSONRPCHandler) handleRootJSONRPC(w http.ResponseWriter, r *http.Request) {
	// Handle root path and only root path
	if r.URL.Path != "/" && r.URL.Path != "" {
		// Let other handlers handle non-root paths
		http.NotFound(w, r)
		return
	}

	log.Printf("JSON-RPC Root: %s %s", r.Method, r.URL.Path)

	// Only accept POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Peek at the request to determine if it's a streaming request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}

	// Parse to check the method
	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("JSON-RPC Root: method=%s, routing to %s", rpcReq.Method,
		map[bool]string{true: "streaming", false: "non-streaming"}[rpcReq.Method == "message/stream"])

	// Recreate request body for handlers
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	// Route based on method
	if rpcReq.Method == "message/stream" {
		// Streaming request - use SSE
		h.handleStreamingMessage(w, r)
	} else {
		// Non-streaming request - use regular JSON-RPC
		h.handleJSONRPC(w, r)
	}
}

// handleJSONRPC processes JSON-RPC 2.0 requests
func (h *JSONRPCHandler) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Only accept POST
	if r.Method != http.MethodPost {
		h.sendError(w, nil, MethodNotFound, "Method not allowed")
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendError(w, nil, ParseError, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Parse JSON-RPC request
	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		h.sendError(w, nil, ParseError, "Invalid JSON")
		return
	}

	// Validate JSON-RPC version
	if rpcReq.JSONRPC != "2.0" {
		h.sendError(w, rpcReq.ID, InvalidRequest, "Invalid JSON-RPC version")
		return
	}

	log.Printf("JSON-RPC: method=%s id=%v", rpcReq.Method, rpcReq.ID)

	// Apply A2A field mapping before processing
	mappedParams := h.applyA2AFieldMapping(rpcReq.Params)

	// Route to appropriate handler
	result, err := h.handleMethod(r.Context(), rpcReq.Method, mappedParams, r)
	if err != nil {
		h.sendError(w, rpcReq.ID, InternalError, err.Error())
		return
	}

	// Apply A2A transformations to result
	result = h.transformResultForA2A(result)

	// Send success response
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      rpcReq.ID,
		Result:  result,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// handleMethod routes JSON-RPC methods to gRPC service calls
func (h *JSONRPCHandler) handleMethod(ctx context.Context, method string, params json.RawMessage, r *http.Request) (interface{}, error) {
	// Extract agent name from URL query parameter for routing
	var md metadata.MD
	if agentName := r.URL.Query().Get("agent"); agentName != "" {
		md = metadata.Pairs("agent-name", agentName)
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	switch method {
	case "message/send":
		return h.handleSendMessage(ctx, params)
	case "message/stream":
		// Streaming not supported via regular JSON-RPC response
		// Client should use the /rpc/stream endpoint with SSE
		return nil, fmt.Errorf("use /rpc/stream endpoint for streaming messages")
	case "tasks/get":
		return h.handleGetTask(ctx, params)
	case "tasks/cancel":
		return h.handleCancelTask(ctx, params)
	case "card/get":
		return h.handleGetAgentCard(ctx, params)
	default:
		return nil, fmt.Errorf("method not found: %s", method)
	}
}

// applyA2AFieldMapping transforms A2A JSON fields to protobuf format
func (h *JSONRPCHandler) applyA2AFieldMapping(params json.RawMessage) json.RawMessage {
	var paramsMap map[string]interface{}
	if err := json.Unmarshal(params, &paramsMap); err != nil {
		return params // Return unchanged if parsing fails
	}

	// Look for message object and apply transformations
	if message, ok := paramsMap["message"].(map[string]interface{}); ok {
		// Map "parts" → "content" for A2A compatibility
		if parts, ok := message["parts"]; ok {
			message["content"] = parts
			delete(message, "parts")
		}
		// Map lowercase "role" values to protobuf enum format
		if role, ok := message["role"].(string); ok {
			switch role {
			case "user":
				message["role"] = "ROLE_USER"
			case "agent", "assistant":
				message["role"] = "ROLE_AGENT"
			case "system":
				message["role"] = "ROLE_SYSTEM"
			}
		}
	}

	// Re-marshal the modified params
	result, _ := json.Marshal(paramsMap)
	return result
}

// transformResultForA2A transforms protobuf response to A2A format
func (h *JSONRPCHandler) transformResultForA2A(result interface{}) interface{} {
	// Convert to JSON and back to apply transformations
	jsonData, err := json.Marshal(result)
	if err != nil {
		return result
	}

	var resultMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &resultMap); err != nil {
		return result
	}

	// Check if this is a SendMessageResponse with task wrapper
	// The protobuf SendMessageResponse has: oneof response { Task task = 1; }
	// We need to unwrap it
	if taskObj, ok := resultMap["task"].(map[string]interface{}); ok {
		// Unwrap the task from the response wrapper
		resultMap = taskObj
	}

	// Apply field name transformations recursively
	h.transformMapForA2A(resultMap)

	// Add kind discriminator based on object structure
	if _, hasId := resultMap["id"]; hasId {
		if _, hasStatus := resultMap["status"]; hasStatus {
			resultMap["kind"] = "task"
		}
	} else if _, hasMessageId := resultMap["messageId"]; hasMessageId {
		resultMap["kind"] = "message"
	}

	return resultMap
}

// transformMapForA2A recursively transforms field names
func (h *JSONRPCHandler) transformMapForA2A(obj map[string]interface{}) {
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

	// Apply transformations at THIS level first
	for oldKey, newKey := range fieldMappings {
		if val, ok := obj[oldKey]; ok {
			obj[newKey] = val
			delete(obj, oldKey)
		}
	}

	// Transform role enum values
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

	// Transform state enum values
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

	// Recursively transform nested objects and arrays
	for _, val := range obj {
		switch v := val.(type) {
		case map[string]interface{}:
			h.transformMapForA2A(v)
		case []interface{}:
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					h.transformMapForA2A(m)
				}
			}
		}
	}
}

// handleSendMessage handles the message/send method
func (h *JSONRPCHandler) handleSendMessage(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.SendMessageRequest
	if err := h.unmarshaler.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	resp, err := h.service.SendMessage(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("service error: %w", err)
	}

	// Convert protobuf response to JSON
	jsonData, err := h.marshaler.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to interface: %w", err)
	}

	return result, nil
}

// handleGetTask handles the tasks/get method
func (h *JSONRPCHandler) handleGetTask(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.GetTaskRequest
	if err := h.unmarshaler.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// Extract task name to get task ID: "tasks/{task_id}"
	// Use task ID to route to correct agent via task lookup
	resp, err := h.service.GetTask(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("service error: %w", err)
	}

	jsonData, err := h.marshaler.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to interface: %w", err)
	}

	return result, nil
}

// handleCancelTask handles the tasks/cancel method
func (h *JSONRPCHandler) handleCancelTask(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.CancelTaskRequest
	if err := h.unmarshaler.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	resp, err := h.service.CancelTask(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("service error: %w", err)
	}

	jsonData, err := h.marshaler.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to interface: %w", err)
	}

	return result, nil
}

// handleGetAgentCard handles the card/get method
func (h *JSONRPCHandler) handleGetAgentCard(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.GetAgentCardRequest
	// GetAgentCard typically has no params, but parse anyway for consistency
	if len(params) > 0 {
		if err := h.unmarshaler.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	resp, err := h.service.GetAgentCard(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("service error: %w", err)
	}

	jsonData, err := h.marshaler.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to interface: %w", err)
	}

	return result, nil
}

// handleStreamingMessage handles streaming messages via Server-Sent Events (SSE)
// This is what web clients like a2a-inspector expect for streaming
func (h *JSONRPCHandler) handleStreamingMessage(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Only accept POST
	if r.Method != http.MethodPost {
		h.sendSSEError(w, "Method not allowed")
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendSSEError(w, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Parse JSON-RPC request
	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		h.sendSSEError(w, "Invalid JSON")
		return
	}

	// Validate JSON-RPC version
	if rpcReq.JSONRPC != "2.0" {
		h.sendSSEError(w, "Invalid JSON-RPC version")
		return
	}

	// Apply A2A field mapping (parts → content, lowercase roles → UPPERCASE)
	mappedParams := h.applyA2AFieldMapping(rpcReq.Params)

	var req pb.SendMessageRequest
	if err := h.unmarshaler.Unmarshal(mappedParams, &req); err != nil {
		h.sendSSEError(w, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	log.Printf("JSON-RPC SSE: method=%s id=%v", rpcReq.Method, rpcReq.ID)

	// Extract agent name for routing
	// Priority: 1) URL query param, 2) context_id format, 3) message metadata
	ctx := r.Context()
	var agentName string
	var md metadata.MD

	// First check URL query parameter (set by agent card)
	agentName = r.URL.Query().Get("agent")
	if agentName != "" {
		log.Printf("JSON-RPC SSE: routing to agent '%s' from URL query param", agentName)
		md = metadata.Pairs("agent-name", agentName)
	} else if req.Request != nil && req.Request.ContextId != "" {
		// Try context_id format: "agent_name:session_id"
		parts := strings.Split(req.Request.ContextId, ":")
		if len(parts) >= 2 && parts[0] != "" {
			agentName = parts[0]
			log.Printf("JSON-RPC SSE: routing to agent '%s' from context_id", agentName)
			md = metadata.Pairs("agent-name", agentName)
		} else {
			// No agent specified, will use single-agent fallback
			log.Printf("JSON-RPC SSE: no agent specified, using single-agent fallback")
		}
	}

	// Create incoming context with metadata for the service call
	// This is needed because we're calling the service directly (not via gRPC)
	if md != nil {
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	// Create a stream server wrapper with the context that has agent-name metadata
	streamWrapper := &jsonrpcStreamWrapper{
		writer:    w,
		flusher:   w.(http.Flusher),
		marshaler: h.marshaler,
		id:        rpcReq.ID,
		context:   ctx, // This context has agent-name in metadata
	}

	// Call streaming service - it will use streamWrapper.Context() which returns our ctx with metadata
	err = h.service.SendStreamingMessage(&req, streamWrapper)
	if err != nil {
		h.sendSSEError(w, fmt.Sprintf("Service error: %v", err))
		return
	}

	// Don't send a completion event - A2A SDK doesn't expect it and will fail validation
	// Just close the stream by returning
}

// sendSSEError sends an error via SSE
func (h *JSONRPCHandler) sendSSEError(w http.ResponseWriter, message string) {
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

// jsonrpcStreamWrapper wraps http.ResponseWriter to implement pb.A2AService_SendStreamingMessageServer
type jsonrpcStreamWrapper struct {
	writer    http.ResponseWriter
	flusher   http.Flusher
	marshaler protojson.MarshalOptions
	id        interface{}
	context   context.Context
}

func (w *jsonrpcStreamWrapper) Send(resp *pb.StreamResponse) error {
	// Marshal protobuf to JSON
	jsonData, err := w.marshaler.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Parse to interface for A2A field name transformation
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return fmt.Errorf("failed to unmarshal to interface: %w", err)
	}

	// Transform protobuf field names to A2A field names
	// A2A spec uses camelCase: parts, messageId, contextId, taskId, artifactId, role: "user"/"agent"
	// Protobuf uses snake_case: content, message_id, context_id, task_id, artifact_id, role: "ROLE_USER"/"ROLE_AGENT"
	w.transformToA2AFieldNames(result)

	// Unwrap the protobuf oneof wrapper
	// StreamResponse has oneof payload {task, message, statusUpdate, artifactUpdate}
	// So result is like {"task": {...}} but A2A SDK expects the inner object directly
	// CRITICAL: Also add "kind" discriminator field for Pydantic union validation
	var unwrappedResult map[string]interface{}
	if taskObj, ok := result["task"].(map[string]interface{}); ok {
		taskObj["kind"] = "task"
		unwrappedResult = taskObj
	} else if msgObj, ok := result["message"].(map[string]interface{}); ok {
		msgObj["kind"] = "message"
		unwrappedResult = msgObj
	} else if statusObj, ok := result["statusUpdate"].(map[string]interface{}); ok {
		statusObj["kind"] = "status-update"
		unwrappedResult = statusObj
	} else if artifactObj, ok := result["artifactUpdate"].(map[string]interface{}); ok {
		artifactObj["kind"] = "artifact-update"
		unwrappedResult = artifactObj
	} else {
		// Fallback: use the result as-is (result is already map[string]interface{})
		unwrappedResult = result
	}

	// For SSE streaming with JSON-RPC, wrap in JSON-RPC success response structure
	// Per A2A TypeScript types: SendStreamingMessageSuccessResponse extends JSONRPCSuccessResponse
	// which has: { jsonrpc: "2.0", id: number|string|null, result: Task|Message|... }
	rpcResp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      w.id,
		Result:  unwrappedResult,
	}

	// Send as SSE event
	data, err := json.Marshal(rpcResp)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE data: %w", err)
	}

	// Write SSE format: event: message\ndata: {...}\n\n
	_, err = fmt.Fprintf(w.writer, "event: message\ndata: %s\n\n", data)
	if err != nil {
		return err
	}

	// Flush to ensure data is sent immediately
	w.flusher.Flush()

	return nil
}

// transformToA2AFieldNames recursively transforms protobuf field names to A2A field names
func (w *jsonrpcStreamWrapper) transformToA2AFieldNames(obj map[string]interface{}) {
	// Field name mappings: protobuf → A2A (snake_case → camelCase)
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

	// Apply transformations at THIS level first
	// 1. Transform field names
	for oldKey, newKey := range fieldMappings {
		if val, ok := obj[oldKey]; ok {
			obj[newKey] = val
			delete(obj, oldKey)
		}
	}

	// 2. Transform role enum values: ROLE_USER → user, ROLE_AGENT → agent
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

	// 3. Transform state enum values: TASK_STATE_SUBMITTED → submitted, etc.
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

	// THEN recursively transform all nested objects and arrays
	for _, val := range obj {
		switch v := val.(type) {
		case map[string]interface{}:
			w.transformToA2AFieldNames(v)
		case []interface{}:
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					w.transformToA2AFieldNames(m)
				}
			}
		}
	}
}

// sendCompletionEvent is no longer used - A2A SDK doesn't expect a done event
// and will fail validation if we send one. The stream closure is sufficient.
// func (w *jsonrpcStreamWrapper) sendCompletionEvent() {
// 	fmt.Fprintf(w.writer, "event: done\ndata: {}\n\n")
// 	w.flusher.Flush()
// }

func (w *jsonrpcStreamWrapper) SetHeader(metadata.MD) error {
	return nil
}

func (w *jsonrpcStreamWrapper) SendHeader(metadata.MD) error {
	return nil
}

func (w *jsonrpcStreamWrapper) SetTrailer(metadata.MD) {
}

func (w *jsonrpcStreamWrapper) Context() context.Context {
	if w.context != nil {
		return w.context
	}
	return context.Background()
}

func (w *jsonrpcStreamWrapper) SendMsg(m interface{}) error {
	return fmt.Errorf("SendMsg not implemented")
}

func (w *jsonrpcStreamWrapper) RecvMsg(m interface{}) error {
	return fmt.Errorf("RecvMsg not implemented")
}

// sendError sends a JSON-RPC error response
func (h *JSONRPCHandler) sendError(w http.ResponseWriter, id interface{}, code int, message string) {
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
