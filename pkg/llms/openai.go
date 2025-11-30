package llms

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
	"github.com/kadirpekel/hector/pkg/observability"
	"github.com/kadirpekel/hector/pkg/protocol"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func createHTTPClient(cfg *config.LLMProviderConfig) *httpclient.Client {
	// Configure TLS if needed
	var tlsConfig *httpclient.TLSConfig
	if cfg.InsecureSkipVerify != nil && *cfg.InsecureSkipVerify || cfg.CACertificate != "" {
		tlsConfig = &httpclient.TLSConfig{
			InsecureSkipVerify: cfg.InsecureSkipVerify != nil && *cfg.InsecureSkipVerify,
			CACertificate:      cfg.CACertificate,
		}
		if tlsConfig.InsecureSkipVerify {
			slog.Warn("TLS certificate verification disabled for LLM provider",
				"provider_type", cfg.Type,
				"insecure_skip_verify", true)
		}
	}

	opts := []httpclient.Option{
		httpclient.WithHTTPClient(&http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		}),
		httpclient.WithMaxRetries(cfg.MaxRetries),
		httpclient.WithBaseDelay(time.Duration(cfg.RetryDelay) * time.Second),
		httpclient.WithHeaderParser(httpclient.ParseOpenAIRateLimitHeaders),
	}

	if tlsConfig != nil {
		opts = append(opts, httpclient.WithTLSConfig(tlsConfig))
	}

	return httpclient.New(opts...)
}

// Constants for OpenAI Responses API
const (
	// Default OpenAI API base URL
	openAIDefaultHost = "https://api.openai.com/v1"

	// SSE Event Types
	eventResponseCreated           = "response.created"
	eventOutputItemAdded           = "response.output_item.added"
	eventOutputItemDone            = "response.output_item.done"
	eventOutputTextDelta           = "response.output_text.delta"
	eventOutputTextDone            = "response.output_text.done"
	eventFunctionCallArgsDelta     = "response.function_call_arguments.delta"
	eventFunctionCallArgsDone      = "response.function_call_arguments.done"
	eventReasoningSummaryTextDelta = "response.reasoning_summary_text.delta"
	eventReasoningSummaryTextDone  = "response.reasoning_summary_text.done"
	eventReasoningSummaryPartDone  = "response.reasoning_summary_part.done"
	eventContentPartAdded          = "response.content_part.added"
	eventContentPartDone           = "response.content_part.done"
	eventInProgress                = "response.in_progress"
	eventResponseCompleted         = "response.completed"

	// Logging preview limits
	maxPayloadPreviewLength = 200
	maxDataPreviewLength    = 300

	// Stream channel buffer size
	streamChannelBufferSize = 100

	// Reasoning effort thresholds based on OpenAI docs
	reasoningEffortLowThreshold    = 1024
	reasoningEffortMediumThreshold = 8192
)

type OpenAIProvider struct {
	config     *config.LLMProviderConfig
	httpClient *httpclient.Client
}

// streamingState encapsulates all state variables used during SSE streaming.
// This improves code organization and makes state management explicit.
type streamingState struct {
	thinkingBlockID   string
	thinkingSignature string
	thinkingStreamed  bool
	functionCallID    string
	functionCallName  string
	functionCallArgs  strings.Builder
	totalTokens       int
	emittedCallIDs    map[string]bool // Track emitted tool call IDs to prevent duplicates
}

// resetThinking clears thinking-related state
func (s *streamingState) resetThinking() {
	s.thinkingBlockID = ""
	s.thinkingSignature = ""
}

// resetFunctionCall clears function call state
func (s *streamingState) resetFunctionCall() {
	s.functionCallID = ""
	s.functionCallName = ""
	s.functionCallArgs.Reset()
}

// hasOpenThinkingBlock returns true if there's an uncompleted thinking block
func (s *streamingState) hasOpenThinkingBlock() bool {
	return s.thinkingBlockID != "" && !s.thinkingStreamed
}

// getMapKeys returns the keys of a map for debugging purposes
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Responses API Types - OpenAI Responses API is the only supported API
// See: https://platform.openai.com/docs/api-reference/responses

// OpenAIResponsesRequest represents a request to the OpenAI Responses API
type OpenAIResponsesRequest struct {
	Model              string                 `json:"model"`
	Input              interface{}            `json:"input,omitempty"`             // string or []OpenAIInputItem
	Instructions       string                 `json:"instructions,omitempty"`      // System message
	MaxOutputTokens    *int                   `json:"max_output_tokens,omitempty"` // NOT max_tokens
	Temperature        *float64               `json:"temperature,omitempty"`
	Tools              []OpenAIResponsesTool  `json:"tools,omitempty"`
	ToolChoice         interface{}            `json:"tool_choice,omitempty"`          // string or object
	Reasoning          *OpenAIReasoningConfig `json:"reasoning,omitempty"`            // Nested object, NOT top-level
	Include            []string               `json:"include,omitempty"`              // For encrypted content
	PreviousResponseID string                 `json:"previous_response_id,omitempty"` // Multi-turn support
	Store              *bool                  `json:"store,omitempty"`                // Stateless mode
	Stream             bool                   `json:"stream,omitempty"`               // Enable streaming
	Text               *OpenAITextFormat      `json:"text,omitempty"`                 // For structured outputs (text.format)
}

// OpenAITextFormat represents the text.format field for structured outputs
// In Responses API, structured outputs moved from response_format to text.format
type OpenAITextFormat struct {
	Format *OpenAIJSONSchemaFormat `json:"format,omitempty"`
}

// OpenAIJSONSchemaFormat represents the JSON schema format for structured outputs
type OpenAIJSONSchemaFormat struct {
	Type   string                 `json:"type"`   // "json_schema"
	Name   string                 `json:"name"`   // Schema name
	Strict bool                   `json:"strict"` // Strict mode
	Schema map[string]interface{} `json:"schema"` // JSON schema
}

// OpenAIReasoningConfig represents the reasoning configuration (nested object)
type OpenAIReasoningConfig struct {
	Effort  string `json:"effort,omitempty"`  // "low", "medium", "high"
	Summary string `json:"summary,omitempty"` // "auto", "concise", "detailed"
}

// OpenAIResponsesTool represents a tool in the Responses API
// Responses API format is flat: type, name, description, parameters, strict at top level
// See: https://platform.openai.com/docs/api-reference/responses/create#responses-create-tools
type OpenAIResponsesTool struct {
	Type        string                 `json:"type"`                  // "function"
	Name        string                 `json:"name"`                  // Function name
	Description string                 `json:"description,omitempty"` // Function description
	Parameters  map[string]interface{} `json:"parameters,omitempty"`  // JSON Schema for parameters
	Strict      bool                   `json:"strict,omitempty"`      // Strict mode for schema validation
}

// OpenAIInputItem represents an input item in the Responses API
// Different item types have different required fields at top level
type OpenAIInputItem struct {
	Type    string      `json:"type"`              // "message", "function_call", "function_call_output", etc.
	ID      string      `json:"id,omitempty"`      // Item ID
	Role    string      `json:"role,omitempty"`    // "user", "assistant", etc. (for message type)
	Content interface{} `json:"content,omitempty"` // Content array or string (for message type)
	// Function call fields (for type="function_call")
	CallID    string `json:"call_id,omitempty"`   // Required for function_call and function_call_output
	Name      string `json:"name,omitempty"`      // Function name (for function_call)
	Arguments string `json:"arguments,omitempty"` // JSON arguments (for function_call)
	// Function call output fields (for type="function_call_output")
	Output *string `json:"output,omitempty"` // Output string (required for function_call_output, nil for other types)
}

// OpenAIResponsesResponse represents a response from the Responses API
type OpenAIResponsesResponse struct {
	ID                 string                   `json:"id"`
	Object             string                   `json:"object"`
	CreatedAt          int64                    `json:"created_at"`
	Status             string                   `json:"status"` // "completed", "failed", "in_progress", etc.
	Error              *OpenAIError             `json:"error,omitempty"`
	IncompleteDetails  *OpenAIIncompleteDetails `json:"incomplete_details,omitempty"`
	Model              string                   `json:"model"`
	Output             []OpenAIOutputItem       `json:"output"`              // Array, NOT choices
	Reasoning          *OpenAIReasoningResponse `json:"reasoning,omitempty"` // Contains summary
	Usage              OpenAIUsage              `json:"usage"`
	PreviousResponseID string                   `json:"previous_response_id,omitempty"`
}

// OpenAIOutputItem represents an item in the output array
// For function_call type: id is the output item id, call_id is the function call id used for results
// See: https://platform.openai.com/docs/api-reference/responses/object#responses/object-output
type OpenAIOutputItem struct {
	Type             string                       `json:"type"` // "message", "reasoning", "function_call", etc.
	ID               string                       `json:"id,omitempty"`
	Status           string                       `json:"status,omitempty"`            // "completed", "failed", etc.
	Role             string                       `json:"role,omitempty"`              // For message type
	Content          interface{}                  `json:"content,omitempty"`           // Content array (for message type)
	Summary          []OpenAIReasoningSummaryItem `json:"summary,omitempty"`           // For reasoning type
	EncryptedContent *OpenAIEncryptedContent      `json:"encrypted_content,omitempty"` // For reasoning items with encryption
	// Function call fields (for type="function_call") - these are top-level, not nested
	CallID    string `json:"call_id,omitempty"`   // The call_id to reference in function_call_output
	Name      string `json:"name,omitempty"`      // Function name
	Arguments string `json:"arguments,omitempty"` // JSON string of arguments
}

// OpenAIEncryptedContent represents encrypted content in reasoning items
type OpenAIEncryptedContent struct {
	Type  string `json:"type"`   // "aes-256-gcm"
	Data  string `json:"data"`   // Base64-encoded encrypted data
	IV    string `json:"iv"`     // Base64-encoded initialization vector
	Tag   string `json:"tag"`    // Base64-encoded authentication tag
	KeyID string `json:"key_id"` // Key identifier
}

// OpenAIReasoningSummaryItem represents an item in the reasoning summary array
type OpenAIReasoningSummaryItem struct {
	Type string `json:"type"` // "summary_text"
	Text string `json:"text"`
}

// OpenAIReasoningResponse represents the reasoning object in the response
type OpenAIReasoningResponse struct {
	Effort  *string `json:"effort,omitempty"`
	Summary *string `json:"summary,omitempty"`
}

// OpenAIIncompleteDetails represents details about why a response is incomplete
type OpenAIIncompleteDetails struct {
	Reason string `json:"reason,omitempty"`
}

// OpenAIError represents an error in the Responses API
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Code    string `json:"code,omitempty"`
}

// OpenAIUsage represents token usage in the Responses API
type OpenAIUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// NewOpenAIProvider creates a new OpenAI provider with default configuration.
// This is a convenience constructor for simple use cases.
// For production use, prefer NewOpenAIProviderFromConfig with explicit configuration.
func NewOpenAIProvider(apiKey string, model string) *OpenAIProvider {
	cfg := &config.LLMProviderConfig{
		Type:        "openai",
		Model:       model,
		APIKey:      apiKey,
		Host:        openAIDefaultHost,
		Temperature: func() *float64 { t := 0.7; return &t }(),
		MaxTokens:   1000,
		Timeout:     60,
	}

	provider, err := NewOpenAIProviderFromConfig(cfg)
	if err != nil {
		// This should never happen with valid config, but log for debugging
		slog.Error("Failed to create OpenAI provider", "error", err)
		return nil
	}
	return provider
}

func NewOpenAIProviderFromConfig(cfg *config.LLMProviderConfig) (*OpenAIProvider, error) {
	// Use createHTTPClient to properly configure TLS settings
	httpClient := createHTTPClient(cfg)

	return &OpenAIProvider{
		config:     cfg,
		httpClient: httpClient,
	}, nil
}

func (p *OpenAIProvider) Generate(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (string, []*protocol.ToolCall, int, *ThinkingBlock, error) {
	startTime := time.Now()

	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "openai"),
			attribute.String("api", "responses"),
			attribute.Bool("streaming", false),
		),
	)
	defer span.End()

	effort := p.getReasoningEffort()
	if effort != "" {
		slog.Debug("Using Responses API with reasoning enabled",
			"model", p.config.Model,
			"effort", effort)
	} else {
		slog.Debug("Using Responses API (thinking disabled)",
			"model", p.config.Model)
	}

	text, toolCalls, tokens, thinkingBlock, _, err := p.GenerateWithReasoning(ctx, messages, tools, effort, "")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		duration := time.Since(startTime)
		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, err)
		}
		return "", nil, 0, nil, err
	}

	duration := time.Since(startTime)
	if thinkingBlock != nil {
		slog.Debug("Received thinking block from Responses API",
			"content_length", len(thinkingBlock.Content),
			"has_signature", thinkingBlock.Signature != "")
	}

	span.SetStatus(codes.Ok, "success")
	metrics := observability.GetGlobalMetrics()
	if metrics != nil {
		metrics.RecordLLMCall(ctx, p.config.Model, duration, tokens, tokens, nil)
	}

	return text, toolCalls, tokens, thinkingBlock, nil
}

func (p *OpenAIProvider) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
	effort := p.getReasoningEffort()
	if effort != "" {
		slog.Debug("Using Responses API streaming with reasoning enabled",
			"model", p.config.Model,
			"effort", effort)
	} else {
		slog.Debug("Using Responses API streaming (thinking disabled)",
			"model", p.config.Model)
	}

	return p.GenerateWithReasoningStreaming(ctx, messages, tools, effort, "")
}

func (p *OpenAIProvider) GenerateStructured(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (string, []*protocol.ToolCall, int, *ThinkingBlock, error) {
	startTime := time.Now()

	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "openai"),
			attribute.String("api", "responses"),
			attribute.Bool("streaming", false),
			attribute.Bool("structured", true),
		),
	)
	defer span.End()

	effort := p.getReasoningEffort()
	requestSummary := p.config.Thinking != nil && p.config.Thinking.Enabled

	// Build Responses API request
	// Only request summaries when thinking is explicitly enabled
	req := p.buildResponsesRequest(messages, tools, effort, "", requestSummary)

	// Add structured output format using text.format (Responses API format)
	if structConfig != nil && structConfig.Format == "json" {
		if structConfig.Schema != nil {
			schema, ok := structConfig.Schema.(map[string]interface{})
			if !ok {
				schemaErr := fmt.Errorf("schema must be a map")
				span.RecordError(schemaErr)
				span.SetStatus(codes.Error, "invalid schema")
				return "", nil, 0, nil, schemaErr
			}

			req.Text = &OpenAITextFormat{
				Format: &OpenAIJSONSchemaFormat{
					Type:   "json_schema",
					Name:   "response",
					Strict: true,
					Schema: schema,
				},
			}
		} else {
			// Simple JSON object format
			req.Text = &OpenAITextFormat{
				Format: &OpenAIJSONSchemaFormat{
					Type:   "json_schema",
					Name:   "response",
					Strict: true,
					Schema: map[string]interface{}{
						"type": "object",
					},
				},
			}
		}
	}

	// Use Responses API
	text, toolCalls, tokens, thinkingBlock, _, err := p.makeResponsesRequest(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, err)
		}
		return "", nil, 0, nil, err
	}

	span.SetAttributes(
		attribute.Int(observability.AttrLLMTokensInput, tokens),
		attribute.Int(observability.AttrLLMTokensOutput, tokens),
		attribute.Int("llm.tool_calls", len(toolCalls)),
	)
	span.SetStatus(codes.Ok, "success")

	metrics := observability.GetGlobalMetrics()
	if metrics != nil {
		metrics.RecordLLMCall(ctx, p.config.Model, duration, tokens, tokens, nil)
	}

	return text, toolCalls, tokens, thinkingBlock, nil
}

func (p *OpenAIProvider) GenerateStructuredStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (<-chan StreamChunk, error) {
	effort := p.getReasoningEffort()
	requestSummary := p.config.Thinking != nil && p.config.Thinking.Enabled

	// Build Responses API request with streaming
	// Only request summaries when thinking is explicitly enabled
	req := p.buildResponsesRequest(messages, tools, effort, "", requestSummary)
	req.Stream = true

	// Add structured output format using text.format (Responses API format)
	if structConfig != nil && structConfig.Format == "json" && structConfig.Schema != nil {
		schema, ok := structConfig.Schema.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("schema must be a map")
		}

		req.Text = &OpenAITextFormat{
			Format: &OpenAIJSONSchemaFormat{
				Type:   "json_schema",
				Name:   "response",
				Strict: true,
				Schema: schema,
			},
		}
	}

	// Use Responses API streaming
	return p.GenerateWithReasoningStreaming(ctx, messages, tools, effort, "")
}

func (p *OpenAIProvider) GetModelName() string {
	return p.config.Model
}

func (p *OpenAIProvider) GetMaxTokens() int {
	return p.config.MaxTokens
}

func (p *OpenAIProvider) GetTemperature() float64 {
	if p.config.Temperature == nil {
		return 0.7
	}
	return *p.config.Temperature
}

func (p *OpenAIProvider) GetSupportedInputModes() []string {
	return []string{
		"text/plain",
		"application/json",
		"image/jpeg",
		"image/png",
		"image/webp",
	}
}

func (p *OpenAIProvider) Close() error {
	return nil
}

func (p *OpenAIProvider) SupportsStructuredOutput() bool {
	return true
}

// getResponsesURL returns the URL for the OpenAI Responses API
func (p *OpenAIProvider) getResponsesURL() string {
	if p.config.Host == "" {
		return openAIDefaultHost + "/responses"
	}

	host := strings.TrimSuffix(p.config.Host, "/")
	if strings.HasSuffix(host, "/v1") {
		return fmt.Sprintf("%s/responses", host)
	}
	return fmt.Sprintf("%s/v1/responses", host)
}

// getReasoningEffort returns the reasoning effort level based on thinking config
func (p *OpenAIProvider) getReasoningEffort() string {
	if p.config.Thinking != nil && p.config.Thinking.Enabled {
		return p.mapBudgetToReasoningEffort(p.config.Thinking.BudgetTokens)
	}
	return ""
}

// shouldRetryWithoutSummary checks if an error indicates org verification is needed for summaries
func (p *OpenAIProvider) shouldRetryWithoutSummary(errorResp *OpenAIResponsesResponse) bool {
	if errorResp == nil || errorResp.Error == nil {
		return false
	}
	return errorResp.Error.Code == "unsupported_value" &&
		strings.Contains(errorResp.Error.Message, "reasoning summaries")
}

// logRequestDebug logs debug information about a Responses API request
func (p *OpenAIProvider) logRequestDebug(req *OpenAIResponsesRequest, reqBody []byte) {
	payloadPreview := string(reqBody)
	if len(payloadPreview) > maxPayloadPreviewLength {
		payloadPreview = payloadPreview[:maxPayloadPreviewLength] + "..."
	}

	// Safe extraction of reasoning effort (may be nil for non-reasoning models)
	reasoningEffort := ""
	if req.Reasoning != nil {
		reasoningEffort = req.Reasoning.Effort
	}

	// Safe extraction of input items count
	inputItemsCount := 0
	if items, ok := req.Input.([]OpenAIInputItem); ok {
		inputItemsCount = len(items)
	}

	slog.Debug("OpenAI Responses API request",
		"model", req.Model,
		"input_items", inputItemsCount,
		"has_instructions", req.Instructions != "",
		"max_output_tokens", req.MaxOutputTokens,
		"effort", reasoningEffort,
		"payload_preview", payloadPreview)
}

// roleToOpenAI converts pb.Role to OpenAI role string
// Used internally in convertMessagesToInputItems
func roleToOpenAI(role pb.Role) string {
	switch role {
	case pb.Role_ROLE_USER:
		return "user"
	case pb.Role_ROLE_AGENT:
		return "assistant"
	case pb.Role_ROLE_UNSPECIFIED:
		return "system"
	default:
		return "system"
	}
}

// GenerateWithReasoning uses the OpenAI Responses API to get reasoning items
func (p *OpenAIProvider) GenerateWithReasoning(
	ctx context.Context,
	messages []*pb.Message,
	tools []ToolDefinition,
	effort string,
	previousResponseID string,
) (string, []*protocol.ToolCall, int, *ThinkingBlock, string, error) {
	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "openai"),
			attribute.String("api", "responses"),
			attribute.Bool("streaming", false),
			attribute.String("reasoning_effort", effort),
		),
	)
	defer span.End()

	// Determine if summaries should be requested based on thinking config
	requestSummary := p.config.Thinking != nil && p.config.Thinking.Enabled
	req := p.buildResponsesRequest(messages, tools, effort, previousResponseID, requestSummary)
	return p.makeResponsesRequest(ctx, req)
}

// makeResponsesRequest makes a non-streaming request to the Responses API
func (p *OpenAIProvider) makeResponsesRequest(ctx context.Context, req *OpenAIResponsesRequest) (string, []*protocol.ToolCall, int, *ThinkingBlock, string, error) {
	url := p.getResponsesURL()

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", nil, 0, nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	p.logRequestDebug(req, reqBody)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", nil, 0, nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", strings.TrimSpace(p.config.APIKey)))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", nil, 0, nil, "", fmt.Errorf("openai responses API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-OK responses
	if resp.StatusCode != http.StatusOK {
		return p.handleErrorResponse(ctx, resp, req, url)
	}

	// Decode successful response
	var responsesResp OpenAIResponsesResponse
	if err := json.NewDecoder(resp.Body).Decode(&responsesResp); err != nil {
		return "", nil, 0, nil, "", fmt.Errorf("failed to decode response: %w", err)
	}

	return p.processResponsesResponse(&responsesResp)
}

// handleErrorResponse handles non-OK HTTP responses from the Responses API
func (p *OpenAIProvider) handleErrorResponse(ctx context.Context, resp *http.Response, req *OpenAIResponsesRequest, url string) (string, []*protocol.ToolCall, int, *ThinkingBlock, string, error) {
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", nil, 0, nil, "", fmt.Errorf("openai responses API error (HTTP %d): failed to read body: %w", resp.StatusCode, readErr)
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", nil, 0, nil, "", fmt.Errorf("openai responses API endpoint not found (404): %s", string(bodyBytes))
	}

	// Try to parse error response
	var errorResp OpenAIResponsesResponse
	if err := json.Unmarshal(bodyBytes, &errorResp); err != nil || errorResp.Error == nil {
		return "", nil, 0, nil, "", fmt.Errorf("openai responses API error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Check if we should retry without reasoning summary
	if resp.StatusCode == http.StatusBadRequest && p.shouldRetryWithoutSummary(&errorResp) {
		return p.retryWithoutSummary(ctx, req, url)
	}

	return "", nil, 0, nil, "", fmt.Errorf("openai responses API error (HTTP %d): %s", resp.StatusCode, errorResp.Error.Message)
}

// retryWithoutSummary retries the request without the reasoning summary parameter
func (p *OpenAIProvider) retryWithoutSummary(ctx context.Context, originalReq *OpenAIResponsesRequest, url string) (string, []*protocol.ToolCall, int, *ThinkingBlock, string, error) {
	slog.Debug("Organization not verified for reasoning summaries, retrying without summary parameter")

	// Create a copy of the request to avoid mutating the original
	retryReq := *originalReq
	if retryReq.Reasoning != nil {
		reasoningCopy := *retryReq.Reasoning
		reasoningCopy.Summary = ""
		retryReq.Reasoning = &reasoningCopy
	}

	reqBody, err := json.Marshal(&retryReq)
	if err != nil {
		return "", nil, 0, nil, "", fmt.Errorf("failed to marshal retry request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", nil, 0, nil, "", fmt.Errorf("failed to create retry request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", strings.TrimSpace(p.config.APIKey)))

	retryResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", nil, 0, nil, "", fmt.Errorf("openai responses API retry request failed: %w", err)
	}
	defer retryResp.Body.Close()

	if retryResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(retryResp.Body)
		return "", nil, 0, nil, "", fmt.Errorf("openai responses API retry error (HTTP %d): %s", retryResp.StatusCode, string(bodyBytes))
	}

	var responsesResp OpenAIResponsesResponse
	if err := json.NewDecoder(retryResp.Body).Decode(&responsesResp); err != nil {
		return "", nil, 0, nil, "", fmt.Errorf("failed to decode retry response: %w", err)
	}

	return p.processResponsesResponse(&responsesResp)
}

// processResponsesResponse processes a successful response from the Responses API
func (p *OpenAIProvider) processResponsesResponse(responsesResp *OpenAIResponsesResponse) (string, []*protocol.ToolCall, int, *ThinkingBlock, string, error) {

	if responsesResp.Error != nil {
		return "", nil, 0, nil, "", fmt.Errorf("openai responses API error: %s", responsesResp.Error.Message)
	}

	if responsesResp.Status != "completed" {
		err := fmt.Errorf("openai responses API response incomplete: status=%s", responsesResp.Status)
		if responsesResp.IncompleteDetails != nil {
			err = fmt.Errorf("openai responses API response incomplete: status=%s, reason=%s", responsesResp.Status, responsesResp.IncompleteDetails.Reason)
		}
		return "", nil, 0, nil, "", err
	}

	if len(responsesResp.Output) == 0 {
		return "", nil, 0, nil, "", fmt.Errorf("no output items in response")
	}

	responseID := responsesResp.ID

	var text string
	var toolCalls []*protocol.ToolCall
	var thinkingBlock *ThinkingBlock

	if responsesResp.Reasoning != nil && responsesResp.Reasoning.Summary != nil {
		thinkingContent := *responsesResp.Reasoning.Summary
		if thinkingContent != "" {
			thinkingBlock = &ThinkingBlock{
				Content:   thinkingContent,
				Signature: "",
			}
		}
	}

	for _, outputItem := range responsesResp.Output {
		switch outputItem.Type {
		case "message":
			text = p.extractTextFromMessageOutput(outputItem)
		case "function_call":
			// Function call fields are at top level in Responses API (name, arguments, call_id)
			toolCall, err := p.parseFunctionCallOutput(outputItem)
			if err != nil {
				slog.Warn("Failed to parse function call", "error", err, "id", outputItem.ID)
				continue
			}
			if toolCall != nil {
				toolCalls = append(toolCalls, toolCall)
			}
		case "reasoning":
			thinkingContent := p.extractReasoningFromOutput(outputItem)
			if thinkingContent != "" {
				encryptedSig := ""
				if outputItem.EncryptedContent != nil {
					encryptedSig = outputItem.EncryptedContent.Data
				}
				thinkingBlock = &ThinkingBlock{
					Content:   thinkingContent,
					Signature: encryptedSig,
				}
			}
		}
	}

	tokensUsed := responsesResp.Usage.TotalTokens
	return text, toolCalls, tokensUsed, thinkingBlock, responseID, nil
}

// GenerateWithReasoningStreaming uses the OpenAI Responses API with streaming
func (p *OpenAIProvider) GenerateWithReasoningStreaming(
	ctx context.Context,
	messages []*pb.Message,
	tools []ToolDefinition,
	effort string,
	previousResponseID string,
) (<-chan StreamChunk, error) {
	startTime := time.Now()

	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "openai"),
			attribute.String("api", "responses"),
			attribute.Bool("streaming", true),
			attribute.String("reasoning_effort", effort),
		),
	)

	outputCh := make(chan StreamChunk, streamChannelBufferSize)

	go func() {
		defer span.End()
		defer close(outputCh)

		// Determine if summaries should be requested based on thinking config
		requestSummary := p.config.Thinking != nil && p.config.Thinking.Enabled
		req := p.buildResponsesRequest(messages, tools, effort, previousResponseID, requestSummary)
		req.Stream = true

		url := p.getResponsesURL()

		reqBody, err := json.Marshal(req)
		if err != nil {
			outputCh <- StreamChunk{
				Type:  "error",
				Error: fmt.Errorf("failed to marshal request: %w", err),
			}
			return
		}

		p.logRequestDebug(req, reqBody)
		slog.Debug("OpenAI Responses API streaming request", "effort", effort)

		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
		if err != nil {
			outputCh <- StreamChunk{
				Type:  "error",
				Error: fmt.Errorf("failed to create request: %w", err),
			}
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", strings.TrimSpace(p.config.APIKey)))

		resp, err := p.httpClient.Do(httpReq)
		if resp != nil {
			defer resp.Body.Close()

			slog.Debug("OpenAI Responses API response", "status", resp.StatusCode, "status_text", resp.Status)

			if resp.StatusCode == http.StatusNotFound {
				bodyBytes, _ := io.ReadAll(resp.Body)
				err := fmt.Errorf("openai responses API endpoint not found (404): %s", string(bodyBytes))
				span.RecordError(err)
				span.SetStatus(codes.Error, "Responses API not available")
				outputCh <- StreamChunk{
					Type:  "error",
					Error: err,
				}
				return
			}

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				var errorResp OpenAIResponsesResponse
				if json.Unmarshal(bodyBytes, &errorResp) == nil && errorResp.Error != nil {
					errMsg := fmt.Sprintf("openai responses API error (HTTP %d): %s", resp.StatusCode, errorResp.Error.Message)
					if errorResp.Error.Code != "" {
						errMsg += fmt.Sprintf(" - code: %s", errorResp.Error.Code)
					}
					err := fmt.Errorf("%s", errMsg)
					slog.Error("OpenAI Responses API streaming error",
						"status", resp.StatusCode,
						"message", errorResp.Error.Message,
						"code", errorResp.Error.Code)
					span.RecordError(err)
					span.SetStatus(codes.Error, errorResp.Error.Message)
					outputCh <- StreamChunk{
						Type:  "error",
						Error: err,
					}
					return
				}
				err := fmt.Errorf("openai responses API error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
				slog.Error("OpenAI Responses API streaming error", "status", resp.StatusCode, "response_body", string(bodyBytes))
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				outputCh <- StreamChunk{
					Type:  "error",
					Error: err,
				}
				return
			}
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			slog.Error("OpenAI Responses API request failed", "error", err)
			outputCh <- StreamChunk{
				Type:  "error",
				Error: fmt.Errorf("openai responses API request failed: %w", err),
			}
			return
		}

		if resp == nil {
			err := fmt.Errorf("openai responses API request failed: no response received")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			outputCh <- StreamChunk{
				Type:  "error",
				Error: err,
			}
			return
		}

		// Use bufio.NewReader with ReadBytes instead of Scanner for better handling of large lines
		// ReadBytes reads until delimiter (no fixed buffer limit), making it more suitable for
		// large tool results (web search, document parsing, etc.) compared to Scanner's default 64KB limit
		reader := bufio.NewReader(resp.Body)
		state := &streamingState{
			emittedCallIDs: make(map[string]bool),
		}
		var currentEventType string
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				outputCh <- StreamChunk{
					Type:  "error",
					Error: fmt.Errorf("failed to read stream: %w", err),
				}
				return
			}

			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			if bytes.HasPrefix(line, []byte("event: ")) {
				currentEventType = string(bytes.TrimSpace(line[7:]))
				continue
			}

			if !bytes.HasPrefix(line, []byte("data: ")) {
				continue
			}
			dataLine := line[6:]

			var streamEvent map[string]interface{}
			if err := json.Unmarshal(dataLine, &streamEvent); err != nil {
				slog.Debug("Failed to parse streaming event", "error", err, "line", string(dataLine))
				currentEventType = ""
				continue
			}

			eventType := currentEventType
			if eventType == "" {
				eventType, _ = streamEvent["type"].(string)
			}

			// Debug: Log every event we receive to diagnose streaming issues
			keys := make([]string, 0, len(streamEvent))
			for k := range streamEvent {
				keys = append(keys, k)
			}
			dataPreview := string(dataLine)
			if len(dataPreview) > maxDataPreviewLength {
				dataPreview = dataPreview[:maxDataPreviewLength] + "..."
			}
			slog.Debug("SSE event received",
				"event_type_from_header", currentEventType,
				"event_type_from_data", streamEvent["type"],
				"final_event_type", eventType,
				"data_keys", keys,
				"raw_data_preview", dataPreview)

			currentEventType = ""

			switch eventType {
			case eventResponseCreated:
				if response, ok := streamEvent["response"].(map[string]interface{}); ok {
					if id, ok := response["id"].(string); ok {
						slog.Debug("Responses API streaming started", "response_id", id)
					}
				}
			case eventOutputItemAdded:
				item, ok := streamEvent["item"].(map[string]interface{})
				if !ok {
					slog.Debug("output_item.added event missing item field")
					continue
				}
				itemType, ok := item["type"].(string)
				if !ok {
					slog.Debug("output_item.added item missing type field", "item_keys", getMapKeys(item))
					continue
				}
				switch itemType {
				case "reasoning":
					// Track the reasoning block - content will arrive via delta events
					if id, ok := item["id"].(string); ok {
						state.thinkingBlockID = id
					}
					// Reset the streamed flag for this new reasoning block
					state.thinkingStreamed = false
					slog.Debug("Reasoning block started", "id", state.thinkingBlockID)
				case "function_call":
					// Start tracking a new function call
					if callID, ok := item["call_id"].(string); ok {
						state.functionCallID = callID
					} else if id, ok := item["id"].(string); ok {
						state.functionCallID = id
					}
					if name, ok := item["name"].(string); ok {
						state.functionCallName = name
					}
					state.functionCallArgs.Reset()
					slog.Debug("Function call started",
						"call_id", state.functionCallID,
						"name", state.functionCallName)
				}
			case eventOutputItemDone:
				item, ok := streamEvent["item"].(map[string]interface{})
				if !ok {
					slog.Debug("output_item.done event missing item field")
					continue
				}
				itemType, ok := item["type"].(string)
				if !ok {
					slog.Debug("output_item.done item missing type field", "item_keys", getMapKeys(item))
					continue
				}
				switch itemType {
				case "reasoning":
					// Extract encrypted content signature if available
					if encryptedContentObj, ok := item["encrypted_content"].(map[string]interface{}); ok {
						if data, ok := encryptedContentObj["data"].(string); ok {
							state.thinkingSignature = data
						}
					}
					// Only emit thinking content if we haven't already streamed it via delta events
					// This prevents duplication when both delta streaming and done events have content
					if !state.thinkingStreamed {
						if summary, ok := item["summary"].([]interface{}); ok {
							for _, summaryItem := range summary {
								if itemMap, ok := summaryItem.(map[string]interface{}); ok {
									textType, _ := itemMap["type"].(string)
									if textType == "summary_text" {
										if text, ok := itemMap["text"].(string); ok && text != "" {
											outputCh <- StreamChunk{
												Type: "thinking",
												Text: text,
											}
										}
									}
								}
							}
						}
						outputCh <- StreamChunk{
							Type:      "thinking_complete",
							Signature: state.thinkingSignature,
						}
					}
					state.resetThinking()
				case "function_call":
					// Alternative path: function call completed via output_item.done
					// Extract call_id, name, arguments from the completed item
					callID := ""
					if cid, ok := item["call_id"].(string); ok {
						callID = cid
					} else if id, ok := item["id"].(string); ok {
						callID = id
					}
					name, _ := item["name"].(string)
					argsStr, _ := item["arguments"].(string)

					if callID != "" && name != "" {
						// Skip if already emitted (prevent duplicates from multiple events)
						if state.emittedCallIDs[callID] {
							slog.Debug("Skipping duplicate tool call from output_item.done",
								"call_id", callID, "name", name)
							state.resetFunctionCall()
							continue
						}

						var args map[string]interface{}
						if argsStr != "" {
							if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
								slog.Warn("Failed to parse function call arguments from output_item.done",
									"error", err, "call_id", callID)
								args = make(map[string]interface{})
							}
						} else {
							args = make(map[string]interface{})
						}

						state.emittedCallIDs[callID] = true
						outputCh <- StreamChunk{
							Type: "tool_call",
							ToolCall: &protocol.ToolCall{
								ID:   callID,
								Name: name,
								Args: args,
							},
						}
						slog.Debug("Function call completed via output_item.done",
							"call_id", callID, "name", name)
					}

					state.resetFunctionCall()
				}
			case eventOutputTextDelta:
				// Responses API streaming: delta can be in different formats
				// Try multiple possible structures
				var deltaText string
				if delta, ok := streamEvent["delta"].(string); ok && delta != "" {
					deltaText = delta
				} else if deltaObj, ok := streamEvent["delta"].(map[string]interface{}); ok {
					// Delta might be an object with text field
					if text, ok := deltaObj["text"].(string); ok {
						deltaText = text
					}
				} else if text, ok := streamEvent["text"].(string); ok && text != "" {
					// Some events might have text directly
					deltaText = text
				}

				if deltaText != "" {
					slog.Debug("Sending text delta to channel", "delta_length", len(deltaText), "delta_preview", deltaText[:min(len(deltaText), 50)])
					outputCh <- StreamChunk{
						Type: "text",
						Text: deltaText,
					}
				} else {
					slog.Debug("No delta text found in output_text.delta event", "event_keys", keys, "delta_type", fmt.Sprintf("%T", streamEvent["delta"]))
				}
			case eventFunctionCallArgsDelta:
				// Streaming function call arguments
				if delta, ok := streamEvent["delta"].(string); ok && delta != "" {
					state.functionCallArgs.WriteString(delta)
				}
			case eventFunctionCallArgsDone:
				// Function call arguments complete - emit tool call
				if state.functionCallID != "" && state.functionCallName != "" {
					// Skip if already emitted (prevent duplicates from multiple events)
					if state.emittedCallIDs[state.functionCallID] {
						slog.Debug("Skipping duplicate tool call from function_call_arguments.done",
							"call_id", state.functionCallID, "name", state.functionCallName)
						state.resetFunctionCall()
						continue
					}

					var args map[string]interface{}
					argsStr := state.functionCallArgs.String()
					if argsStr != "" {
						if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
							slog.Warn("Failed to parse streaming function call arguments",
								"error", err,
								"call_id", state.functionCallID,
								"args", argsStr)
							args = make(map[string]interface{})
						}
					} else {
						args = make(map[string]interface{})
					}

					state.emittedCallIDs[state.functionCallID] = true
					outputCh <- StreamChunk{
						Type: "tool_call",
						ToolCall: &protocol.ToolCall{
							ID:   state.functionCallID,
							Name: state.functionCallName,
							Args: args,
						},
					}
					slog.Debug("Function call completed",
						"call_id", state.functionCallID,
						"name", state.functionCallName)

					state.resetFunctionCall()
				}
			case eventReasoningSummaryTextDelta:
				// Stream reasoning/thinking content as it arrives
				if delta, ok := streamEvent["delta"].(string); ok && delta != "" {
					state.thinkingStreamed = true
					outputCh <- StreamChunk{
						Type: "thinking",
						Text: delta,
					}
				}
			case eventReasoningSummaryTextDone:
				// Reasoning summary text is complete - mark thinking as complete
				// The full text is in streamEvent["text"] but we've already streamed via deltas
				state.thinkingStreamed = true
				outputCh <- StreamChunk{
					Type:      "thinking_complete",
					Signature: state.thinkingSignature,
				}
				state.resetThinking()
			case eventReasoningSummaryPartDone:
				// Alternative completion event - mark thinking complete if not already done
				if state.thinkingBlockID != "" {
					state.thinkingStreamed = true
					outputCh <- StreamChunk{
						Type:      "thinking_complete",
						Signature: state.thinkingSignature,
					}
					state.resetThinking()
				}
			case eventContentPartAdded, eventContentPartDone, eventInProgress, eventOutputTextDone:
				// No action needed
			case eventResponseCompleted:
				if response, ok := streamEvent["response"].(map[string]interface{}); ok {
					if usage, ok := response["usage"].(map[string]interface{}); ok {
						if total, ok := usage["total_tokens"].(float64); ok {
							state.totalTokens = int(total)
						}
					}
					// Note: We don't emit thinking content from response.completed
					// because it should have been streamed via delta events already
				}
			default:
				// Log unhandled event types for debugging
				if eventType != "" {
					slog.Debug("Unhandled SSE event type", "event_type", eventType, "event_keys", keys)
				}
			}
		}

		// Safety: emit thinking_complete if we have an open thinking block that wasn't completed
		if state.hasOpenThinkingBlock() {
			outputCh <- StreamChunk{
				Type:      "thinking_complete",
				Signature: state.thinkingSignature,
			}
		}
		outputCh <- StreamChunk{
			Type:   "done",
			Tokens: state.totalTokens,
		}

		duration := time.Since(startTime)
		span.SetStatus(codes.Ok, "success")

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, state.totalTokens, nil)
		}
	}()

	return outputCh, nil
}

// buildResponsesRequest builds a request for the Responses API
// requestSummary should only be true when thinking is explicitly enabled AND organization is verified
func (p *OpenAIProvider) buildResponsesRequest(messages []*pb.Message, tools []ToolDefinition, effort string, previousResponseID string, requestSummary bool) *OpenAIResponsesRequest {
	inputItems, instructions := p.convertMessagesToInputItems(messages)

	if len(inputItems) == 0 && previousResponseID == "" {
		slog.Warn("No input items and no previous_response_id - API requires at least one")
		inputItems = []OpenAIInputItem{
			{
				Type:    "message",
				Role:    "user",
				Content: []map[string]interface{}{{"type": "input_text", "text": ""}},
			},
		}
	}

	var maxOutputTokens *int
	if p.config.MaxTokens > 0 {
		maxOutputTokens = &p.config.MaxTokens
	}

	req := &OpenAIResponsesRequest{
		Model:           p.config.Model,
		Input:           inputItems,
		MaxOutputTokens: maxOutputTokens,
	}

	// Only set reasoning config for reasoning models (o1, o3, o4, gpt-5, etc.)
	// Non-reasoning models don't support the reasoning parameter
	if effort != "" && p.isReasoningModel(p.config.Model) {
		reasoningConfig := &OpenAIReasoningConfig{
			Effort: effort,
		}
		// Only request summaries when thinking is explicitly enabled
		// Summary requires organization verification, so don't set it by default
		if requestSummary {
			reasoningConfig.Summary = "auto"
		}
		req.Reasoning = reasoningConfig
	}

	if instructions != "" {
		req.Instructions = instructions
	}

	if len(tools) > 0 {
		req.Tools = p.convertToResponsesAPITools(tools)
		req.ToolChoice = "auto"
	}

	// Only request encrypted content for reasoning models (needed for multi-turn)
	// Non-reasoning models (like gpt-4o) don't support encrypted content
	if effort != "" && p.isReasoningModel(p.config.Model) {
		req.Include = []string{"reasoning.encrypted_content"}
	}

	if previousResponseID != "" {
		req.PreviousResponseID = previousResponseID
	}

	// Set temperature only for non-reasoning models
	// Reasoning models don't support temperature parameter
	if !p.isReasoningModel(p.config.Model) && p.config.Temperature != nil {
		req.Temperature = p.config.Temperature
	}

	return req
}

// convertToResponsesAPITools converts ToolDefinition to Responses API tool format
// Responses API uses flat structure: type, name, description, parameters at top level
func (p *OpenAIProvider) convertToResponsesAPITools(tools []ToolDefinition) []OpenAIResponsesTool {
	result := make([]OpenAIResponsesTool, len(tools))
	for i, tool := range tools {
		result[i] = OpenAIResponsesTool{
			Type:        "function",
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
			Strict:      false,
		}
	}
	return result
}

// convertMessagesToInputItems converts pb.Message to OpenAI Responses API input items
// Handles: user messages, assistant messages (with optional tool calls), and tool results
// See: https://platform.openai.com/docs/api-reference/responses/create#responses-create-input
func (p *OpenAIProvider) convertMessagesToInputItems(messages []*pb.Message) ([]OpenAIInputItem, string) {
	inputItems := make([]OpenAIInputItem, 0, len(messages))
	var instructions strings.Builder

	for _, msg := range messages {
		if msg.Role == pb.Role_ROLE_UNSPECIFIED {
			// System messages go to instructions field
			for _, part := range msg.Parts {
				if text := part.GetText(); text != "" {
					if instructions.Len() > 0 {
						instructions.WriteString("\n")
					}
					instructions.WriteString(text)
				}
			}
			continue
		}

		// Check for tool results first (these are in user-role messages typically)
		toolResults := protocol.GetToolResultsFromMessage(msg)
		if len(toolResults) > 0 {
			// Tool results must be sent as function_call_output items
			// Fields at top level: type, call_id, output (output is required by OpenAI API)
			for _, result := range toolResults {
				output := result.Content // Output field is required, even if empty string
				inputItems = append(inputItems, OpenAIInputItem{
					Type:   "function_call_output",
					CallID: result.ToolCallID,
					Output: &output, // Use pointer so field is included even if empty string
				})
			}
			continue
		}

		// Check for tool calls in assistant messages
		toolCalls := protocol.GetToolCallsFromMessage(msg)
		if msg.Role == pb.Role_ROLE_AGENT && len(toolCalls) > 0 {
			// First, add text content if any (as a message)
			textContent := protocol.ExtractTextFromMessage(msg)
			if textContent != "" {
				inputItems = append(inputItems, OpenAIInputItem{
					Type:    "message",
					Role:    "assistant",
					Content: []map[string]interface{}{{"type": "output_text", "text": textContent}},
				})
			}

			// Add each tool call as a separate function_call item
			// Fields at top level: type, call_id, name, arguments
			for _, tc := range toolCalls {
				argsJSON, _ := json.Marshal(tc.Args)
				inputItems = append(inputItems, OpenAIInputItem{
					Type:      "function_call",
					CallID:    tc.ID,
					Name:      tc.Name,
					Arguments: string(argsJSON),
				})
			}
			continue
		}

		role := roleToOpenAI(msg.Role)

		content := p.extractContentFromMessage(msg, role)
		if len(content) == 0 {
			continue
		}

		inputItem := OpenAIInputItem{
			Type:    "message",
			Role:    role,
			Content: content,
		}

		inputItems = append(inputItems, inputItem)
	}

	return inputItems, instructions.String()
}

// extractContentFromMessage extracts content from pb.Message for Responses API
// The role parameter determines the content type:
// - "user" messages use "input_text" and "input_image"
// - "assistant" messages use "output_text"
func (p *OpenAIProvider) extractContentFromMessage(msg *pb.Message, role string) []map[string]interface{} {
	contentParts := make([]map[string]interface{}, 0)

	// Determine the text content type based on role
	// Per OpenAI Responses API: user messages use input_text, assistant uses output_text
	textType := "input_text"
	if role == "assistant" {
		textType = "output_text"
	}

	for _, part := range msg.Parts {
		if text := part.GetText(); text != "" {
			contentParts = append(contentParts, map[string]interface{}{
				"type": textType,
				"text": text,
			})
		} else if file := part.GetFile(); file != nil {
			url := ""
			if uri := file.GetFileWithUri(); uri != "" {
				url = uri
			} else if bytes := file.GetFileWithBytes(); len(bytes) > 0 {
				mediaType := file.GetMediaType()
				if mediaType == "" {
					mediaType = detectImageMediaType(bytes)
				}

				// Only process image types
				if strings.HasPrefix(mediaType, "image/") {
					// Check image size limit
					if len(bytes) > MaxOpenAIImageSize {
						slog.Warn("Image exceeds size limit, skipping",
							"size", len(bytes),
							"limit", MaxOpenAIImageSize)
						continue
					}

					// Encode bytes to base64 and format as data URL
					base64Data := base64.StdEncoding.EncodeToString(bytes)
					url = fmt.Sprintf("data:%s;base64,%s", mediaType, base64Data)
				}
			}
			if url != "" {
				contentParts = append(contentParts, map[string]interface{}{
					"type":      "input_image",
					"image_url": url,
				})
			}
		}
	}

	return contentParts
}

// extractTextFromMessageOutput extracts text from a message output item
func (p *OpenAIProvider) extractTextFromMessageOutput(outputItem OpenAIOutputItem) string {
	if outputItem.Content == nil {
		return ""
	}

	contentArray, ok := outputItem.Content.([]interface{})
	if !ok {
		return ""
	}

	var textBuilder strings.Builder
	for _, part := range contentArray {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}

		partType, _ := partMap["type"].(string)
		if partType == "output_text" {
			if text, ok := partMap["text"].(string); ok {
				textBuilder.WriteString(text)
			}
		}
	}

	return textBuilder.String()
}

// parseFunctionCallOutput parses a function_call output item into a ToolCall
// Responses API has function call data at top level: call_id, name, arguments
func (p *OpenAIProvider) parseFunctionCallOutput(outputItem OpenAIOutputItem) (*protocol.ToolCall, error) {
	if outputItem.Name == "" {
		return nil, fmt.Errorf("function_call name is empty")
	}

	var args map[string]interface{}
	if outputItem.Arguments != "" {
		if err := json.Unmarshal([]byte(outputItem.Arguments), &args); err != nil {
			return nil, fmt.Errorf("failed to parse function arguments: %w", err)
		}
	} else {
		args = make(map[string]interface{})
	}

	// Use call_id as the ID (this is what we need to reference in function_call_output)
	// Fall back to output item ID if call_id is not present
	toolCallID := outputItem.CallID
	if toolCallID == "" {
		toolCallID = outputItem.ID
	}

	return &protocol.ToolCall{
		ID:   toolCallID,
		Name: outputItem.Name,
		Args: args,
	}, nil
}

// extractReasoningFromOutput extracts reasoning content from a reasoning output item
func (p *OpenAIProvider) extractReasoningFromOutput(outputItem OpenAIOutputItem) string {
	if len(outputItem.Summary) == 0 {
		return ""
	}

	var thinkingBuilder strings.Builder
	for _, summaryItem := range outputItem.Summary {
		if summaryItem.Type == "summary_text" && summaryItem.Text != "" {
			thinkingBuilder.WriteString(summaryItem.Text)
			thinkingBuilder.WriteString("\n")
		}
	}

	return strings.TrimSpace(thinkingBuilder.String())
}

// isReasoningModel checks if a model supports reasoning
func (p *OpenAIProvider) isReasoningModel(modelName string) bool {
	return IsOpenAIReasoningModel(modelName)
}

// IsOpenAIReasoningModel checks if an OpenAI model name is a reasoning model
func IsOpenAIReasoningModel(modelName string) bool {
	modelLower := strings.ToLower(modelName)
	if modelLower == "o1" || modelLower == "o3" || modelLower == "o4" || modelLower == "gpt-5" {
		return true
	}
	reasoningPrefixes := []string{
		"o1-",
		"o3-",
		"o4-",
		"gpt-5-",
	}
	for _, prefix := range reasoningPrefixes {
		if strings.HasPrefix(modelLower, prefix) {
			return true
		}
	}
	return false
}

// mapBudgetToReasoningEffort maps thinking budget tokens to OpenAI reasoning_effort levels
// Valid values are: "low", "medium", "high"
// See: https://platform.openai.com/docs/guides/reasoning
func (p *OpenAIProvider) mapBudgetToReasoningEffort(budgetTokens int) string {
	if budgetTokens <= reasoningEffortLowThreshold {
		return "low"
	}
	if budgetTokens <= reasoningEffortMediumThreshold {
		return "medium"
	}
	return "high"
}
