// Package trigger provides scheduled and event-driven agent invocation.
package trigger

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/verikod/hector/pkg/config"
	"github.com/verikod/hector/pkg/httpclient"
	"github.com/verikod/hector/pkg/utils"
)

// WebhookHandler handles incoming webhook requests and invokes agents.
type WebhookHandler struct {
	agentName         string
	appName           string
	config            *config.TriggerConfig
	invoker           AgentInvoker
	enqueuer          AgentEnqueuer // Optional: for durable async/callback execution
	resultProvider    func(ctx context.Context, taskID string) (string, error)
	template          *template.Template
	sessionIDTemplate *template.Template // For deriving session ID from payload
}

// Option configures WebhookHandler.
type Option func(*WebhookHandler)

// WithResultProvider sets the result provider for fetching task results.
func WithResultProvider(rp func(ctx context.Context, taskID string) (string, error)) Option {
	return func(h *WebhookHandler) {
		h.resultProvider = rp
	}
}

// WithEnqueuer sets the enqueuer for durable async/callback execution.
// When set, async and callback modes will use the queue instead of direct invocation.
func WithEnqueuer(enqueuer AgentEnqueuer) Option {
	return func(h *WebhookHandler) {
		h.enqueuer = enqueuer
	}
}

// WithAppName sets the app name for the handler.
func WithAppName(appName string) Option {
	return func(h *WebhookHandler) {
		h.appName = appName
	}
}

// WebhookResult represents the result of a webhook invocation.
type WebhookResult struct {
	TaskID    string `json:"task_id,omitempty"`
	Status    string `json:"status"`
	Result    string `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
	AgentName string `json:"agent_name"`
}

// WebhookPayloadContext provides data for template execution.
type WebhookPayloadContext struct {
	Body    map[string]any    `json:"body"`
	Headers map[string]string `json:"headers"`
	Query   map[string]string `json:"query"`
	Fields  map[string]any    `json:"fields"` // Extracted fields
}

// NewWebhookHandler creates a new webhook handler for an agent.
func NewWebhookHandler(agentName string, cfg *config.TriggerConfig, invoker AgentInvoker, opts ...Option) (*WebhookHandler, error) {
	h := &WebhookHandler{
		agentName: agentName,
		config:    cfg,
		invoker:   invoker,
	}

	for _, opt := range opts {
		opt(h)
	}

	// Pre-compile templates if configured
	if cfg.WebhookInput != nil {
		if cfg.WebhookInput.Template != "" {
			tmpl, err := template.New("webhook").Funcs(utils.TemplateFuncs()).Parse(cfg.WebhookInput.Template)
			if err != nil {
				return nil, fmt.Errorf("invalid webhook input template: %w", err)
			}
			h.template = tmpl
		}

		// Session ID template for conversation continuity
		if cfg.WebhookInput.SessionID != "" {
			tmpl, err := template.New("session_id").Funcs(utils.TemplateFuncs()).Parse(cfg.WebhookInput.SessionID)
			if err != nil {
				return nil, fmt.Errorf("invalid session_id template: %w", err)
			}
			h.sessionIDTemplate = tmpl
		}
	}

	return h, nil
}

// ServeHTTP handles incoming webhook requests.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Validate HTTP method
	if !h.isAllowedMethod(r.Method) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read webhook body", "agent", h.agentName, "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Validate signature if secret is configured
	if h.config.Secret != "" {
		if err := h.validateSignature(r, body); err != nil {
			slog.Warn("Webhook signature validation failed", "agent", h.agentName, "error", err)
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Transform payload to agent input
	input, payloadCtx, err := h.transformPayload(body, r)
	if err != nil {
		slog.Error("Failed to transform webhook payload", "agent", h.agentName, "error", err)
		http.Error(w, "Failed to process payload", http.StatusBadRequest)
		return
	}

	// Filter: Empty template output means "skip this event"
	// This allows users to use conditional templates to filter events.
	input = strings.TrimSpace(input)
	if input == "" {
		slog.Debug("Webhook filtered - empty template output", "agent", h.agentName, "path", r.URL.Path)
		w.WriteHeader(http.StatusOK) // Acknowledge but don't process
		return
	}

	// Derive session ID for conversation continuity
	ctx := r.Context()
	if h.sessionIDTemplate != nil {
		var buf bytes.Buffer
		if err := h.sessionIDTemplate.Execute(&buf, payloadCtx); err != nil {
			slog.Warn("Failed to evaluate session_id template", "agent", h.agentName, "error", err)
			// Continue without session ID - will create new session
		} else if sessionID := strings.TrimSpace(buf.String()); sessionID != "" {
			ctx = WithContextID(ctx, sessionID)
			slog.Debug("Webhook using session ID", "agent", h.agentName, "session_id", sessionID)
		}
	}

	slog.Info("Webhook received", "agent", h.agentName, "method", r.Method, "input_length", len(input))

	// Handle based on response mode
	// Pass enriched context with session ID to handlers
	mode := h.config.Response.Mode
	switch mode {
	case config.WebhookResponseSync:
		h.handleSyncWithContext(w, ctx, input)
	case config.WebhookResponseAsync:
		h.handleAsyncWithContext(w, ctx, input)
	case config.WebhookResponseCallback:
		h.handleCallbackWithContext(w, ctx, input, payloadCtx)
	default:
		h.handleSyncWithContext(w, ctx, input)
	}
}

// handleSyncWithContext waits for agent completion before responding.
func (h *WebhookHandler) handleSyncWithContext(w http.ResponseWriter, ctx context.Context, input string) {
	ctx, cancel := context.WithTimeout(ctx, h.config.Response.Timeout)
	defer cancel()

	taskID, err := h.invoker(ctx, h.agentName, input)

	result := WebhookResult{
		TaskID:    taskID,
		AgentName: h.agentName,
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Status = "timeout"
			result.Error = "Agent execution timed out"
			w.WriteHeader(http.StatusGatewayTimeout)
		} else {
			result.Status = "failed"
			result.Error = err.Error()
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		result.Status = "completed"
		result.Result = "Agent invocation completed successfully"
	}

	h.writeJSONResponse(w, result)
}

// handleAsyncWithContext returns immediately with a task ID for polling.
// The task is tracked via A2A internally - use tasks/get to poll status.
// If enqueuer is configured, task is enqueued for durable execution.
func (h *WebhookHandler) handleAsyncWithContext(w http.ResponseWriter, ctx context.Context, input string) {
	var taskID string
	var err error

	// Use enqueuer for durable execution if available
	if h.enqueuer != nil {
		// Generate unique task/context ID
		contextID := ContextIDFromContext(ctx)
		if contextID == "" {
			contextID = fmt.Sprintf("webhook-%s-%d", h.agentName, time.Now().UnixNano())
		}
		taskID = contextID // Use context ID as task ID for tracking

		err = h.enqueuer(ctx, h.appName, h.agentName, input, contextID)
		if err != nil {
			slog.Warn("Async webhook enqueue failed",
				"agent", h.agentName,
				"app", h.appName,
				"error", err)
		} else {
			slog.Info("Async webhook task enqueued",
				"agent", h.agentName,
				"app", h.appName,
				"task_id", taskID)
		}
	} else {
		// Fall back to direct invocation (legacy)
		taskID, err = h.invoker(ctx, h.agentName, input)
		if err != nil {
			slog.Warn("Async webhook invocation failed to start",
				"agent", h.agentName,
				"error", err)
			if taskID == "" {
				taskID = fmt.Sprintf("webhook-%s-%d", h.agentName, time.Now().UnixNano())
			}
		}
	}

	result := WebhookResult{
		TaskID:    taskID,
		Status:    "accepted",
		AgentName: h.agentName,
	}

	w.WriteHeader(http.StatusAccepted)
	h.writeJSONResponse(w, result)
}

// handleCallbackWithContext returns immediately and POSTs result to callback URL when done.
// If enqueuer is configured, task is enqueued for durable execution.
func (h *WebhookHandler) handleCallbackWithContext(w http.ResponseWriter, ctx context.Context, input string, payloadCtx *WebhookPayloadContext) {
	// Use static callback URL from config
	callbackURL := h.config.Response.CallbackURL

	if callbackURL == "" {
		http.Error(w, "Callback URL not configured (set response.callback_url in trigger config)", http.StatusBadRequest)
		return
	}

	var taskID string
	var startErr error

	// Use enqueuer for durable execution if available
	if h.enqueuer != nil {
		// Generate unique task/context ID
		contextID := ContextIDFromContext(ctx)
		if contextID == "" {
			contextID = fmt.Sprintf("webhook-%s-%d", h.agentName, time.Now().UnixNano())
		}
		taskID = contextID // Use context ID as task ID for tracking

		startErr = h.enqueuer(ctx, h.appName, h.agentName, input, contextID)
		if startErr != nil {
			slog.Warn("Callback webhook enqueue failed",
				"agent", h.agentName,
				"app", h.appName,
				"error", startErr)
		} else {
			slog.Info("Callback webhook task enqueued",
				"agent", h.agentName,
				"app", h.appName,
				"task_id", taskID)
		}
	} else {
		// Fall back to direct invocation (legacy)
		taskID, startErr = h.invoker(ctx, h.agentName, input)
		if startErr != nil {
			slog.Warn("Callback webhook invocation failed to start",
				"agent", h.agentName,
				"error", startErr)
			if taskID == "" {
				taskID = fmt.Sprintf("webhook-%s-%d", h.agentName, time.Now().UnixNano())
			}
		}
	}

	// Send callback when agent completes (agent runs async, callback sent by invoker or here)
	go func() {
		// Wait a bit for agent to complete (in real use, invoker should support callbacks)
		result := WebhookResult{
			TaskID:    taskID,
			AgentName: h.agentName,
		}

		if startErr != nil {
			result.Status = "failed"
			result.Error = startErr.Error()
		} else {
			// If we have a ResultProvider, try to get the actual result
			if h.resultProvider != nil {
				// Poll for result (simple polling for now, could be improved)
				// Wait up to 5 minutes
				timeout := 5 * time.Minute
				deadline := time.Now().Add(timeout)
				ticker := time.NewTicker(1 * time.Second)
				defer ticker.Stop()

				var content string
				var fetchErr error

				for time.Now().Before(deadline) {
					content, fetchErr = h.resultProvider(context.Background(), taskID)
					if fetchErr == nil && content != "" {
						break
					}
					// If error indicates not found or still running, keep polling
					// Ideally ResultProvider should return specific error types
					<-ticker.C
				}

				if content != "" {
					result.Result = content
					result.Status = "completed"
				} else {
					// Timed out or failed
					result.Status = "timeout"
					if fetchErr != nil {
						result.Error = fmt.Sprintf("Failed to fetch result: %v", fetchErr)
					}
				}
			} else {
				// Legacy behavior (no result provider)
				result.Status = "completed"
				result.Result = "Agent invocation completed successfully (result fetcher not configured)"
			}
		}

		// Send callback
		if err := h.sendCallback(callbackURL, result); err != nil {
			slog.Error("Failed to send webhook callback",
				"agent", h.agentName,
				"callback_url", callbackURL,
				"error", err)
		}
	}()

	// Return immediately with accepted status
	result := WebhookResult{
		TaskID:    taskID,
		Status:    "accepted",
		AgentName: h.agentName,
	}

	w.WriteHeader(http.StatusAccepted)
	h.writeJSONResponse(w, result)
}

// validateSignature validates the HMAC signature of the request.
func (h *WebhookHandler) validateSignature(r *http.Request, body []byte) error {
	signature := r.Header.Get(h.config.SignatureHeader)
	if signature == "" {
		return fmt.Errorf("missing signature header: %s", h.config.SignatureHeader)
	}

	// Handle different signature formats
	// GitHub: sha256=<hex>
	// Shopify: <base64>
	// Generic: <hex>
	signature = strings.TrimPrefix(signature, "sha256=")
	signature = strings.TrimPrefix(signature, "sha1=")

	// Compute expected HMAC
	mac := hmac.New(sha256.New, []byte(h.config.Secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// transformPayload transforms the webhook payload into agent input.
func (h *WebhookHandler) transformPayload(body []byte, r *http.Request) (string, *WebhookPayloadContext, error) {
	// Build context
	ctx := &WebhookPayloadContext{
		Body:    make(map[string]any),
		Headers: make(map[string]string),
		Query:   make(map[string]string),
		Fields:  make(map[string]any),
	}

	// Parse JSON body
	if len(body) > 0 {
		if err := json.Unmarshal(body, &ctx.Body); err != nil {
			// Not JSON, store as raw string
			ctx.Body["raw"] = string(body)
		}
	}

	// Extract headers
	for key := range r.Header {
		ctx.Headers[key] = r.Header.Get(key)
	}

	// Extract query params
	for key := range r.URL.Query() {
		ctx.Query[key] = r.URL.Query().Get(key)
	}

	// Extract configured fields
	if h.config.WebhookInput != nil {
		for _, field := range h.config.WebhookInput.ExtractFields {
			value := extractField(ctx.Body, field.Path)
			if value != nil {
				ctx.Fields[field.As] = value
			}
		}
	}

	// Generate input using template or fallback
	var input string

	if h.template != nil {
		var buf bytes.Buffer
		if err := h.template.Execute(&buf, ctx); err != nil {
			return "", ctx, fmt.Errorf("template execution failed: %w", err)
		}
		input = buf.String()
	} else if h.config.Input != "" {
		// Use static input
		input = h.config.Input
	} else {
		// Default: JSON-encode the body
		jsonBody, _ := json.Marshal(ctx.Body)
		input = string(jsonBody)
	}

	return input, ctx, nil
}

// isAllowedMethod checks if the HTTP method is allowed.
func (h *WebhookHandler) isAllowedMethod(method string) bool {
	for _, m := range h.config.Methods {
		if strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}

// writeJSONResponse writes a JSON response.
func (h *WebhookHandler) writeJSONResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

// sendCallback sends the result to the callback URL using httpclient with retries.
func (h *WebhookHandler) sendCallback(callbackURL string, result WebhookResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}

	// Use httpclient with retries for reliable callback delivery
	client := httpclient.New(
		httpclient.WithMaxRetries(3),
		httpclient.WithBaseDelay(1*time.Second),
		httpclient.WithMaxDelay(10*time.Second),
		httpclient.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	)

	req, err := http.NewRequest(http.MethodPost, callbackURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Hector-Webhook-Callback/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("callback returned status %d", resp.StatusCode)
	}

	slog.Info("Webhook callback sent",
		"agent", h.agentName,
		"callback_url", callbackURL,
		"status", resp.StatusCode)

	return nil
}

// extractField extracts a value from a nested map using dot notation.
// Path format: ".body.order.id" or "order.id"
func extractField(data map[string]any, path string) any {
	path = strings.TrimPrefix(path, ".")
	path = strings.TrimPrefix(path, "body.")

	parts := strings.Split(path, ".")
	current := any(data)

	for _, part := range parts {
		if m, ok := current.(map[string]any); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}
