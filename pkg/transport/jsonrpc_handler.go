package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
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
			UseProtoNames:   true,
			EmitUnpopulated: false,
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

	// JSON-RPC endpoint
	mux.HandleFunc("/rpc", h.handleJSONRPC)

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

	// Route to appropriate handler
	result, err := h.handleMethod(r.Context(), rpcReq.Method, rpcReq.Params)
	if err != nil {
		h.sendError(w, rpcReq.ID, InternalError, err.Error())
		return
	}

	// Send success response
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      rpcReq.ID,
		Result:  result,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// handleMethod routes JSON-RPC methods to gRPC service calls
func (h *JSONRPCHandler) handleMethod(ctx context.Context, method string, params json.RawMessage) (interface{}, error) {
	switch method {
	case "message/send":
		return h.handleSendMessage(ctx, params)
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
