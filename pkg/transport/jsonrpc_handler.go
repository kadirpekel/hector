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

type JSONRPCConfig struct {
	HTTPAddress string
}

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

type JSONRPCHandler struct {
	config      JSONRPCConfig
	service     pb.A2AServiceServer
	httpServer  *http.Server
	marshaler   protojson.MarshalOptions
	unmarshaler protojson.UnmarshalOptions
	authConfig  *AuthConfig
}

func NewJSONRPCHandler(config JSONRPCConfig, service pb.A2AServiceServer) *JSONRPCHandler {
	if config.HTTPAddress == "" {
		config.HTTPAddress = ":8081"
	}

	return &JSONRPCHandler{
		config:  config,
		service: service,
		marshaler: protojson.MarshalOptions{
			UseProtoNames:   false,
			EmitUnpopulated: true,
		},
		unmarshaler: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}
}

func (h *JSONRPCHandler) SetAuth(authConfig *AuthConfig) {
	h.authConfig = authConfig
}

func (h *JSONRPCHandler) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", h.handleRootJSONRPC)

	mux.HandleFunc("/rpc", h.handleJSONRPC)

	mux.HandleFunc("/rpc/stream", h.handleStreamingMessage)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	var handler http.Handler = mux
	if h.authConfig != nil && h.authConfig.Enabled && h.authConfig.Validator != nil {
		handler = h.authConfig.Validator.HTTPMiddleware(handler)
	}
	handler = corsMiddleware(loggingMiddleware(handler))

	h.httpServer = &http.Server{
		Addr:    h.config.HTTPAddress,
		Handler: handler,
	}

	log.Printf("JSON-RPC API starting on %s", h.config.HTTPAddress)
	log.Printf("   → Endpoint: POST %s/rpc", h.config.HTTPAddress)
	log.Printf("   → Streaming: POST %s/rpc/stream (SSE)", h.config.HTTPAddress)

	if err := h.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("JSON-RPC server failed: %w", err)
	}

	return nil
}

func (h *JSONRPCHandler) Stop(ctx context.Context) error {
	if h.httpServer == nil {
		return nil
	}

	log.Printf("Shutting down JSON-RPC server...")

	if err := h.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown JSON-RPC server: %w", err)
	}

	log.Printf("JSON-RPC server stopped")
	return nil
}

func (h *JSONRPCHandler) handleRootJSONRPC(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" && r.URL.Path != "" {

		http.NotFound(w, r)
		return
	}

	log.Printf("JSON-RPC Root: %s %s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}

	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("JSON-RPC Root: method=%s, routing to %s", rpcReq.Method,
		map[bool]string{true: "streaming", false: "non-streaming"}[rpcReq.Method == "message/stream"])

	r.Body = io.NopCloser(strings.NewReader(string(body)))

	if rpcReq.Method == "message/stream" {

		h.handleStreamingMessage(w, r)
	} else {

		h.handleJSONRPC(w, r)
	}
}

func (h *JSONRPCHandler) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		h.sendError(w, nil, MethodNotFound, "Method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendError(w, nil, ParseError, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		h.sendError(w, nil, ParseError, "Invalid JSON")
		return
	}

	if rpcReq.JSONRPC != "2.0" {
		h.sendError(w, rpcReq.ID, InvalidRequest, "Invalid JSON-RPC version")
		return
	}

	log.Printf("JSON-RPC: method=%s id=%v", rpcReq.Method, rpcReq.ID)

	mappedParams := applyA2AFieldMapping(rpcReq.Params)

	result, err := h.handleMethod(r.Context(), rpcReq.Method, mappedParams, r)
	if err != nil {
		h.sendError(w, rpcReq.ID, InternalError, err.Error())
		return
	}

	result = h.transformResultForA2A(result)

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      rpcReq.ID,
		Result:  result,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *JSONRPCHandler) handleMethod(ctx context.Context, method string, params json.RawMessage, r *http.Request) (interface{}, error) {

	var md metadata.MD
	if agentName := r.URL.Query().Get("agent"); agentName != "" {
		md = metadata.Pairs("agent-name", agentName)
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	switch method {
	case "message/send":
		return h.handleSendMessage(ctx, params)
	case "message/stream":

		return nil, fmt.Errorf("use /rpc/stream endpoint for streaming messages")
	case "tasks/get":
		return h.handleGetTask(ctx, params)
	case "tasks/list":
		return h.handleListTasks(ctx, params)
	case "tasks/cancel":
		return h.handleCancelTask(ctx, params)
	case "tasks/resubscribe":
		return nil, fmt.Errorf("tasks/resubscribe is only available via streaming endpoint")
	case "agent/getAuthenticatedExtendedCard":
		return h.handleGetAuthenticatedExtendedCard(ctx, params)
	default:
		return nil, fmt.Errorf("method not found: %s", method)
	}
}

func (h *JSONRPCHandler) transformResultForA2A(result interface{}) interface{} {

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

	h.transformMapForA2A(resultMap)

	if _, hasId := resultMap["id"]; hasId {
		if _, hasStatus := resultMap["status"]; hasStatus {
			resultMap["kind"] = "task"
		}
	} else if _, hasMessageId := resultMap["messageId"]; hasMessageId {
		resultMap["kind"] = "message"
	}

	return resultMap
}

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

func (h *JSONRPCHandler) handleSendMessage(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.SendMessageRequest
	if err := h.unmarshaler.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	resp, err := h.service.SendMessage(ctx, &req)
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

func (h *JSONRPCHandler) handleGetTask(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.GetTaskRequest
	if err := h.unmarshaler.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

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

func (h *JSONRPCHandler) handleListTasks(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.ListTasksRequest
	if len(params) > 0 && string(params) != "null" {
		if err := h.unmarshaler.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	resp, err := h.service.ListTasks(ctx, &req)
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

func (h *JSONRPCHandler) handleGetAuthenticatedExtendedCard(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var req pb.GetAgentCardRequest

	// Skip unmarshaling if params is null or empty
	if len(params) > 0 && string(params) != "null" {
		if err := h.unmarshaler.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	// Call GetAgentCard which returns the full card
	// For authenticated users, this can include additional information
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

func (h *JSONRPCHandler) handleStreamingMessage(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		h.sendSSEError(w, "Method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendSSEError(w, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		h.sendSSEError(w, "Invalid JSON")
		return
	}

	if rpcReq.JSONRPC != "2.0" {
		h.sendSSEError(w, "Invalid JSON-RPC version")
		return
	}

	mappedParams := applyA2AFieldMapping(rpcReq.Params)

	var req pb.SendMessageRequest
	if err := h.unmarshaler.Unmarshal(mappedParams, &req); err != nil {
		h.sendSSEError(w, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	log.Printf("JSON-RPC SSE: method=%s id=%v", rpcReq.Method, rpcReq.ID)

	ctx := r.Context()
	var agentName string
	var md metadata.MD

	agentName = r.URL.Query().Get("agent")
	if agentName != "" {
		log.Printf("JSON-RPC SSE: routing to agent '%s' from URL query param", agentName)
		md = metadata.Pairs("agent-name", agentName)
	} else if req.Request != nil && req.Request.ContextId != "" {

		parts := strings.Split(req.Request.ContextId, ":")
		if len(parts) >= 2 && parts[0] != "" {
			agentName = parts[0]
			log.Printf("JSON-RPC SSE: routing to agent '%s' from context_id", agentName)
			md = metadata.Pairs("agent-name", agentName)
		} else {

			log.Printf("JSON-RPC SSE: no agent specified, using single-agent fallback")
		}
	}

	if md != nil {
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	streamWrapper := &jsonrpcStreamWrapper{
		writer:    w,
		flusher:   w.(http.Flusher),
		marshaler: h.marshaler,
		id:        rpcReq.ID,
		context:   ctx,
	}

	err = h.service.SendStreamingMessage(&req, streamWrapper)
	if err != nil {
		h.sendSSEError(w, fmt.Sprintf("Service error: %v", err))
		return
	}

}

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

type jsonrpcStreamWrapper struct {
	writer    http.ResponseWriter
	flusher   http.Flusher
	marshaler protojson.MarshalOptions
	id        interface{}
	context   context.Context
}

func (w *jsonrpcStreamWrapper) Send(resp *pb.StreamResponse) error {

	jsonData, err := w.marshaler.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return fmt.Errorf("failed to unmarshal to interface: %w", err)
	}

	w.transformToA2AFieldNames(result)

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

		unwrappedResult = result
	}

	rpcResp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      w.id,
		Result:  unwrappedResult,
	}

	data, err := json.Marshal(rpcResp)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE data: %w", err)
	}

	_, err = fmt.Fprintf(w.writer, "event: message\ndata: %s\n\n", data)
	if err != nil {
		return err
	}

	w.flusher.Flush()

	return nil
}

func (w *jsonrpcStreamWrapper) transformToA2AFieldNames(obj map[string]interface{}) {

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
