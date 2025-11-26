package llms

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
			fmt.Printf("Warning: TLS certificate verification disabled for LLM provider %s (insecure_skip_verify=true)\n", cfg.Type)
		}
	}

	opts := []httpclient.Option{
		httpclient.WithHTTPClient(&http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		}),
		httpclient.WithMaxRetries(cfg.MaxRetries),
		httpclient.WithBaseDelay(time.Duration(cfg.RetryDelay) * time.Second),
	}

	if tlsConfig != nil {
		opts = append(opts, httpclient.WithTLSConfig(tlsConfig))
	}

	return httpclient.New(opts...)
}

type OpenAIProvider struct {
	config     *config.LLMProviderConfig
	httpClient *httpclient.Client
}

type OpenAIRequest struct {
	Model               string                `json:"model"`
	Messages            []OpenAIMessage       `json:"messages"`
	MaxTokens           *int                  `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int                  `json:"max_completion_tokens,omitempty"`
	Temperature         float64               `json:"temperature"`
	Stream              bool                  `json:"stream"`
	Tools               []OpenAITool          `json:"tools,omitempty"`
	ToolChoice          string                `json:"tool_choice,omitempty"`
	ResponseFormat      *OpenAIResponseFormat `json:"response_format,omitempty"`
	ReasoningEffort     string                `json:"reasoning_effort,omitempty"` // For o-series models: "minimal", "low", "medium", "high"
}

type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
	Error   *Error   `json:"error,omitempty"`
}

type OpenAIStreamResponse struct {
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
	Error   *Error         `json:"error,omitempty"`
}

type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content"` // string or []OpenAIContentPart
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type OpenAIContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *OpenAIImageURL `json:"image_url,omitempty"`
}

type OpenAIImageURL struct {
	URL string `json:"url"`
}

type Choice struct {
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type StreamChoice struct {
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type Delta struct {
	Content   string           `json:"content,omitempty"`
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
	Reasoning string           `json:"reasoning,omitempty"` // Thinking/reasoning content (if exposed by OpenAI)
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

type OpenAIResponseFormat struct {
	Type       string            `json:"type"`
	JSONSchema *OpenAIJSONSchema `json:"json_schema,omitempty"`
}

type OpenAIJSONSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Schema      map[string]interface{} `json:"schema"`
	Strict      bool                   `json:"strict,omitempty"`
}

type OpenAITool struct {
	Type     string             `json:"type"`
	Function OpenAIToolFunction `json:"function"`
}

type OpenAIToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func NewOpenAIProvider(apiKey string, model string) *OpenAIProvider {
	cfg := &config.LLMProviderConfig{
		Type:        "openai",
		Model:       model,
		APIKey:      apiKey,
		Host:        "https://api.openai.com/v1",
		Temperature: func() *float64 { t := 0.7; return &t }(),
		MaxTokens:   1000,
		Timeout:     60,
	}

	provider, _ := NewOpenAIProviderFromConfig(cfg)
	return provider
}

func NewOpenAIProviderFromConfig(cfg *config.LLMProviderConfig) (*OpenAIProvider, error) {

	httpClient := httpclient.New(
		httpclient.WithHTTPClient(&http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		}),
		httpclient.WithMaxRetries(cfg.MaxRetries),
		httpclient.WithBaseDelay(time.Duration(cfg.RetryDelay)*time.Second),
		httpclient.WithHeaderParser(httpclient.ParseOpenAIRateLimitHeaders),
	)

	return &OpenAIProvider{
		config:     cfg,
		httpClient: httpClient,
	}, nil
}

func (p *OpenAIProvider) Generate(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (string, []*protocol.ToolCall, int, error) {
	startTime := time.Now()

	// Create span for LLM request
	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "openai"),
			attribute.Bool("streaming", false),
		),
	)
	defer span.End()

	request := p.buildRequest(messages, false, tools)

	response, err := p.makeRequest(ctx, request)
	duration := time.Since(startTime)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		// Record metrics for failed request
		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, err)
		}

		return "", nil, 0, err
	}

	if response.Error != nil {
		apiErr := fmt.Errorf("OpenAI API error: %s", response.Error.Message)
		span.RecordError(apiErr)
		span.SetStatus(codes.Error, response.Error.Message)

		// Record metrics for API error
		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, apiErr)
		}

		return "", nil, 0, apiErr
	}

	if len(response.Choices) == 0 {
		noChoiceErr := fmt.Errorf("no response choices returned")
		span.RecordError(noChoiceErr)
		span.SetStatus(codes.Error, "no choices")

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, noChoiceErr)
		}

		return "", nil, 0, noChoiceErr
	}

	choice := response.Choices[0]
	tokensUsed := response.Usage.TotalTokens

	text := ""
	if choice.Message.Content != nil {
		if str, ok := choice.Message.Content.(string); ok {
			text = str
		}
	}

	var toolCalls []*protocol.ToolCall
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls, err = parseToolCalls(choice.Message.ToolCalls)
		if err != nil {
			span.RecordError(err)
			return text, nil, tokensUsed, err
		}
	}

	// Record successful metrics
	span.SetAttributes(
		attribute.Int(observability.AttrLLMTokensInput, response.Usage.PromptTokens),
		attribute.Int(observability.AttrLLMTokensOutput, response.Usage.CompletionTokens),
		attribute.Int("llm.tool_calls", len(toolCalls)),
	)
	span.SetStatus(codes.Ok, "success")

	metrics := observability.GetGlobalMetrics()
	if metrics != nil {
		metrics.RecordLLMCall(ctx, p.config.Model, duration, response.Usage.PromptTokens, response.Usage.CompletionTokens, nil)
	}

	return text, toolCalls, tokensUsed, nil
}

func (p *OpenAIProvider) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
	request := p.buildRequest(messages, true, tools)

	outputCh := make(chan StreamChunk, 100)

	go func() {
		defer close(outputCh)

		if err := p.makeStreamingRequest(ctx, request, outputCh); err != nil {
			outputCh <- StreamChunk{
				Type:  "error",
				Error: err,
			}
		}
	}()

	return outputCh, nil
}

func (p *OpenAIProvider) GetModelName() string {
	return p.config.Model
}

func (p *OpenAIProvider) GetMaxTokens() int {
	return p.config.MaxTokens
}

func (p *OpenAIProvider) GetTemperature() float64 {
	if p.config.Temperature == nil {
		return 0.7 // Default
	}
	return *p.config.Temperature
}

// GetSupportedInputModes returns the MIME types supported by OpenAI.
// OpenAI GPT-4o and GPT-4o-mini support images via base64 data URIs or HTTP/HTTPS URLs.
// Supported formats: JPEG, PNG (WebP support may vary by model).
// Note: Video and audio are not supported by OpenAI vision models.
func (p *OpenAIProvider) GetSupportedInputModes() []string {
	return []string{
		"text/plain",
		"application/json",
		"image/jpeg",
		"image/png",
		"image/webp", // Supported by GPT-4o, may vary by model
	}
}

func (p *OpenAIProvider) Close() error {
	return nil
}

func (p *OpenAIProvider) GenerateStructured(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (string, []*protocol.ToolCall, int, error) {
	startTime := time.Now()

	// Create span for structured LLM request
	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "openai"),
			attribute.Bool("streaming", false),
			attribute.Bool("structured", true),
		),
	)
	defer span.End()

	req := p.buildRequest(messages, false, nil)

	if structConfig != nil && structConfig.Format == "json" {
		if structConfig.Schema != nil {
			schema, ok := structConfig.Schema.(map[string]interface{})
			if !ok {
				schemaErr := fmt.Errorf("schema must be a map")
				span.RecordError(schemaErr)
				span.SetStatus(codes.Error, "invalid schema")
				return "", nil, 0, schemaErr
			}

			req.ResponseFormat = &OpenAIResponseFormat{
				Type: "json_schema",
				JSONSchema: &OpenAIJSONSchema{
					Name:   "response",
					Schema: schema,
					Strict: true,
				},
			}
		} else {

			req.ResponseFormat = &OpenAIResponseFormat{
				Type: "json_object",
			}
		}
	}

	response, err := p.makeRequest(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, err)
		}

		return "", nil, 0, err
	}

	if response.Error != nil {
		apiErr := fmt.Errorf("OpenAI API error: %s", response.Error.Message)
		span.RecordError(apiErr)
		span.SetStatus(codes.Error, response.Error.Message)

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, apiErr)
		}

		return "", nil, 0, apiErr
	}

	if len(response.Choices) == 0 {
		noChoiceErr := fmt.Errorf("no response choices returned")
		span.RecordError(noChoiceErr)
		span.SetStatus(codes.Error, "no choices")

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, noChoiceErr)
		}

		return "", nil, 0, noChoiceErr
	}

	choice := response.Choices[0]
	tokensUsed := response.Usage.TotalTokens

	text := ""
	if choice.Message.Content != nil {
		if str, ok := choice.Message.Content.(string); ok {
			text = str
		}
	}

	var toolCalls []*protocol.ToolCall
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls, err = parseToolCalls(choice.Message.ToolCalls)
		if err != nil {
			span.RecordError(err)
			return text, nil, tokensUsed, err
		}
	}

	// Record successful metrics
	span.SetAttributes(
		attribute.Int(observability.AttrLLMTokensInput, response.Usage.PromptTokens),
		attribute.Int(observability.AttrLLMTokensOutput, response.Usage.CompletionTokens),
		attribute.Int("llm.tool_calls", len(toolCalls)),
	)
	span.SetStatus(codes.Ok, "success")

	metrics := observability.GetGlobalMetrics()
	if metrics != nil {
		metrics.RecordLLMCall(ctx, p.config.Model, duration, response.Usage.PromptTokens, response.Usage.CompletionTokens, nil)
	}

	return text, toolCalls, tokensUsed, nil
}

func (p *OpenAIProvider) GenerateStructuredStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (<-chan StreamChunk, error) {
	req := p.buildRequest(messages, true, tools)

	if structConfig != nil && structConfig.Format == "json" && structConfig.Schema != nil {
		schema, ok := structConfig.Schema.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("schema must be a map")
		}

		req.ResponseFormat = &OpenAIResponseFormat{
			Type: "json_schema",
			JSONSchema: &OpenAIJSONSchema{
				Name:   "response",
				Schema: schema,
				Strict: true,
			},
		}
	}

	outputCh := make(chan StreamChunk, 100)

	go func() {
		defer close(outputCh)

		if err := p.makeStreamingRequest(ctx, req, outputCh); err != nil {
			outputCh <- StreamChunk{
				Type:  "error",
				Error: err,
			}
		}
	}()

	return outputCh, nil
}

func (p *OpenAIProvider) SupportsStructuredOutput() bool {
	return true
}

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

func (p *OpenAIProvider) buildRequest(messages []*pb.Message, stream bool, tools []ToolDefinition) OpenAIRequest {

	openaiMessages := make([]OpenAIMessage, 0, len(messages))
	for _, msg := range messages {

		toolResults := protocol.GetToolResultsFromMessage(msg)
		if len(toolResults) > 0 {

			for _, tr := range toolResults {
				content := tr.Content
				openaiMsg := OpenAIMessage{
					Role:       "tool",
					Content:    &content,
					ToolCallID: tr.ToolCallID,
				}
				openaiMessages = append(openaiMessages, openaiMsg)
			}
			continue
		}

		// Handle multi-modal content
		var contentParts []OpenAIContentPart

		for _, part := range msg.Parts {
			if text := part.GetText(); text != "" {
				contentParts = append(contentParts, OpenAIContentPart{
					Type: "text",
					Text: text,
				})
			} else if file := part.GetFile(); file != nil {
				// Handle file parts (images)
				mediaType := file.GetMediaType()
				url := ""

				if uri := file.GetFileWithUri(); uri != "" {
					url = uri
				} else if bytes := file.GetFileWithBytes(); len(bytes) > 0 {
					if mediaType == "" {
						mediaType = detectImageMediaType(bytes)
					}

					if !strings.HasPrefix(mediaType, "image/") {
						continue
					}

					const maxImageSize = 20 * 1024 * 1024
					if len(bytes) > maxImageSize {
						continue
					}

					base64Data := base64.StdEncoding.EncodeToString(bytes)
					url = fmt.Sprintf("data:%s;base64,%s", mediaType, base64Data)
				}

				if url != "" {
					contentParts = append(contentParts, OpenAIContentPart{
						Type: "image_url",
						ImageURL: &OpenAIImageURL{
							URL: url,
						},
					})
				}
			}
		}

		// Always use array format for content (OpenAI API accepts both string and array)
		var finalContent interface{}
		if len(contentParts) > 0 {
			finalContent = contentParts
		} else {
			// Empty message - use empty array
			finalContent = []OpenAIContentPart{}
		}

		openaiMsg := OpenAIMessage{
			Role:    roleToOpenAI(msg.Role),
			Content: finalContent,
		}

		toolCalls := protocol.GetToolCallsFromMessage(msg)
		if len(toolCalls) > 0 {
			openaiMsg.ToolCalls = make([]OpenAIToolCall, len(toolCalls))
			for j, tc := range toolCalls {
				argsJSON, _ := json.Marshal(tc.Args)
				openaiMsg.ToolCalls[j] = OpenAIToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: OpenAIFunctionCall{
						Name:      tc.Name,
						Arguments: string(argsJSON),
					},
				}
			}
		}

		openaiMessages = append(openaiMessages, openaiMsg)
	}

	// Determine if this is a reasoning model that requires max_completion_tokens and temperature=1.0
	modelName := p.config.Model
	isReasoningModel := p.isReasoningModel(modelName)
	
	// Determine temperature: reasoning models (o-series, gpt-5) require temperature=1.0
	// See: https://platform.openai.com/docs/guides/reasoning
	var temperature float64
	if isReasoningModel {
		// Reasoning models only support temperature=1.0 (default)
		temperature = 1.0
	} else {
		// Normal temperature handling for other models
		if p.config.Temperature == nil {
			temperature = 0.7 // Default
		} else {
			temperature = *p.config.Temperature
		}
	}
	
	// Build request with appropriate max tokens field based on model type
	request := OpenAIRequest{
		Model:       modelName,
		Messages:    openaiMessages,
		Temperature: temperature,
		Stream:      stream,
		// Initialize both to nil - we'll set only the appropriate one below
		MaxTokens:           nil,
		MaxCompletionTokens: nil,
	}

	// o-series models (o1, o3, o4) require max_completion_tokens instead of max_tokens
	// See: https://platform.openai.com/docs/guides/reasoning
	if isReasoningModel {
		// For o-series models, use max_completion_tokens
		if p.config.MaxTokens > 0 {
			maxCompletionTokens := p.config.MaxTokens
			request.MaxCompletionTokens = &maxCompletionTokens
		}
		// MaxTokens remains nil and will be omitted from JSON
	} else {
		// For other models, use max_tokens
		if p.config.MaxTokens > 0 {
			maxTokens := p.config.MaxTokens
			request.MaxTokens = &maxTokens
		}
		// MaxCompletionTokens remains nil and will be omitted from JSON
	}

	// Add thinking/reasoning support for reasoning models
	// See: https://platform.openai.com/docs/guides/reasoning
	// Reasoning models (o1, o3, o4, gpt-5) automatically perform reasoning internally.
	// The reasoning_effort parameter controls the depth of reasoning:
	// - "minimal": Least reasoning, fastest responses
	// - "low": Light reasoning
	// - "medium": Moderate reasoning (default when thinking enabled)
	// - "high": Maximum reasoning, best quality but slower
	// Note: OpenAI reasoning models do NOT expose raw thinking tokens in streaming responses
	// like Gemini/Anthropic do. The reasoning happens internally and is not visible.
	if isReasoningModel && p.config.Thinking != nil && p.config.Thinking.Enabled {
		request.ReasoningEffort = p.mapBudgetToReasoningEffort(p.config.Thinking.BudgetTokens)
	}

	if len(tools) > 0 {
		request.Tools = convertToOpenAITools(tools)
		request.ToolChoice = "auto"
	}

	return request
}

// isReasoningModel checks if a model supports reasoning and requires max_completion_tokens
// Supports models like: o1, o1-preview, o1-mini, o3, o3-mini, o4, o4-mini, gpt-5, gpt-5-mini, etc.
func (p *OpenAIProvider) isReasoningModel(modelName string) bool {
	modelLower := strings.ToLower(modelName)
	// Check for exact matches first (e.g., "o1", "o3", "o4", "gpt-5")
	if modelLower == "o1" || modelLower == "o3" || modelLower == "o4" || modelLower == "gpt-5" {
		return true
	}
	// Check for prefix matches (e.g., "o1-", "o3-", "o4-", "gpt-5-")
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
// OpenAI supports: "minimal", "low", "medium", "high"
// Mapping based on token budget similar to Gemini's thinking_budget:
// - minimal: <= 512 tokens (very low effort)
// - low: <= 1024 tokens
// - medium: <= 8192 tokens
// - high: > 8192 tokens
func (p *OpenAIProvider) mapBudgetToReasoningEffort(budgetTokens int) string {
	if budgetTokens <= 0 {
		// Default to "low" if not specified
		return "low"
	}
	if budgetTokens <= 512 {
		return "minimal"
	}
	if budgetTokens <= 1024 {
		return "low"
	}
	if budgetTokens <= 8192 {
		return "medium"
	}
	return "high"
}

func convertToOpenAITools(tools []ToolDefinition) []OpenAITool {
	result := make([]OpenAITool, len(tools))
	for i, tool := range tools {
		result[i] = OpenAITool{
			Type:     "function",
			Function: (OpenAIToolFunction)(tool),
		}
	}
	return result
}

func parseToolCalls(openaiToolCalls []OpenAIToolCall) ([]*protocol.ToolCall, error) {
	result := make([]*protocol.ToolCall, len(openaiToolCalls))

	for i, tc := range openaiToolCalls {

		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
		}

		result[i] = &protocol.ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: args,
		}
	}

	return result, nil
}

// parseErrorResponse extracts error information from OpenAI API error responses
func parseErrorResponse(body []byte) *Error {
	if len(body) == 0 {
		return nil
	}
	var errorResp struct {
		Error Error `json:"error"`
	}
	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
		return &errorResp.Error
	}
	return nil
}

func (p *OpenAIProvider) makeRequest(ctx context.Context, request OpenAIRequest) (*OpenAIResponse, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.Host+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(requestBody)), nil
	}

	req.Header.Set("Content-Type", "application/json")

	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	// HTTP client may return both response and error for non-2xx status codes
	// We need to check the response body even if there's an error
	if resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, readErr := io.ReadAll(resp.Body)
			errorBody := string(body)
			if readErr != nil {
				errorBody = fmt.Sprintf("(failed to read error body: %v)", readErr)
			}
			if apiErr := parseErrorResponse(body); apiErr != nil {
				return nil, fmt.Errorf("API request failed with status %d: %s (type: %s, code: %s)",
					resp.StatusCode, apiErr.Message, apiErr.Type, apiErr.Code)
			}
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, errorBody)
		}
	}

	if err != nil {
		// If we have a response body, try to extract error details
		if resp != nil && resp.Body != nil {
			body, readErr := io.ReadAll(resp.Body)
			if readErr == nil && len(body) > 0 {
				if apiErr := parseErrorResponse(body); apiErr != nil {
					return nil, fmt.Errorf("HTTP request failed: %s (type: %s, code: %s)",
						apiErr.Message, apiErr.Type, apiErr.Code)
				}
				return nil, fmt.Errorf("HTTP request failed: %w - Response: %s", err, string(body))
			}
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("HTTP request failed: no response received")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func (p *OpenAIProvider) makeStreamingRequest(ctx context.Context, request OpenAIRequest, outputCh chan<- StreamChunk) error {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.Host+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(requestBody)), nil
	}

	req.Header.Set("Content-Type", "application/json")

	if p.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	// HTTP client may return both response and error for non-2xx status codes
	// We need to check the response body even if there's an error
	if resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, readErr := io.ReadAll(resp.Body)
			errorBody := string(body)
			if readErr != nil {
				errorBody = fmt.Sprintf("(failed to read error body: %v)", readErr)
			}
			if apiErr := parseErrorResponse(body); apiErr != nil {
				return fmt.Errorf("API request failed with status %d: %s (type: %s, code: %s)",
					resp.StatusCode, apiErr.Message, apiErr.Type, apiErr.Code)
			}
			return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, errorBody)
		}
	}

	if err != nil {
		// If we have a response body, try to extract error details
		if resp != nil && resp.Body != nil {
			body, readErr := io.ReadAll(resp.Body)
			if readErr == nil && len(body) > 0 {
				if apiErr := parseErrorResponse(body); apiErr != nil {
					return fmt.Errorf("HTTP request failed: %s (type: %s, code: %s)",
						apiErr.Message, apiErr.Type, apiErr.Code)
				}
				return fmt.Errorf("HTTP request failed: %w - Response: %s", err, string(body))
			}
		}
		return fmt.Errorf("HTTP request failed: %w", err)
	}

	if resp == nil {
		return fmt.Errorf("HTTP request failed: no response received")
	}

	reader := bufio.NewReader(resp.Body)

	toolCallsMap := make(map[int]*OpenAIToolCall)
	totalTokens := 0
	var accumulatedThinking strings.Builder
	var inThinkingBlock bool

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read stream: %w", err)
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		line = line[6:]

		if bytes.Equal(line, []byte("[DONE]")) {
			break
		}

		var streamResp OpenAIStreamResponse
		if err := json.Unmarshal(line, &streamResp); err != nil {
			continue
		}

		if streamResp.Error != nil {
			return fmt.Errorf("API error: %s", streamResp.Error.Message)
		}

		if streamResp.Usage != nil {
			totalTokens = streamResp.Usage.TotalTokens
		}

		if len(streamResp.Choices) == 0 {
			continue
		}

		choice := streamResp.Choices[0]

		// Handle thinking/reasoning content (if OpenAI exposes it)
		// Note: OpenAI reasoning models may not expose raw thinking tokens in streaming responses
		// The reasoning_effort parameter controls internal reasoning, but the thinking process
		// is typically not exposed. If OpenAI adds support for exposing reasoning content,
		// it would likely appear in delta.reasoning field
		if choice.Delta.Reasoning != "" {
			if !inThinkingBlock {
				inThinkingBlock = true
			}
			accumulatedThinking.WriteString(choice.Delta.Reasoning)
			outputCh <- StreamChunk{
				Type: "thinking",
				Text: choice.Delta.Reasoning,
			}
		}

		// Handle regular text content
		if choice.Delta.Content != "" {
			// Close thinking block if we were in one
			if inThinkingBlock {
				outputCh <- StreamChunk{
					Type: "thinking_complete",
					Text: accumulatedThinking.String(),
				}
				accumulatedThinking.Reset()
				inThinkingBlock = false
			}
			outputCh <- StreamChunk{
				Type: "text",
				Text: choice.Delta.Content,
			}
		}

		for _, deltaCall := range choice.Delta.ToolCalls {

			if deltaCall.ID != "" {

				toolCallsMap[len(toolCallsMap)] = &OpenAIToolCall{
					ID:       deltaCall.ID,
					Type:     deltaCall.Type,
					Function: deltaCall.Function,
				}
			} else {

				if len(toolCallsMap) > 0 {
					lastIdx := len(toolCallsMap) - 1
					if toolCall, exists := toolCallsMap[lastIdx]; exists {
						toolCall.Function.Arguments += deltaCall.Function.Arguments
					}
				}
			}
		}

		if choice.FinishReason == "stop" || choice.FinishReason == "tool_calls" {
			// Close any open thinking block before tool calls
			if inThinkingBlock {
				outputCh <- StreamChunk{
					Type: "thinking_complete",
					Text: accumulatedThinking.String(),
				}
				accumulatedThinking.Reset()
				inThinkingBlock = false
			}

			var accumulatedToolCalls []OpenAIToolCall
			for i := 0; i < len(toolCallsMap); i++ {
				if toolCall, exists := toolCallsMap[i]; exists {
					accumulatedToolCalls = append(accumulatedToolCalls, *toolCall)
				}
			}

			if len(accumulatedToolCalls) > 0 {
				toolCalls, err := parseToolCalls(accumulatedToolCalls)
				if err == nil {
					for _, tc := range toolCalls {
						outputCh <- StreamChunk{
							Type:     "tool_call",
							ToolCall: tc,
						}
					}
				}
			}
			break
		}
	}

	// Close any remaining thinking block at end of stream
	if inThinkingBlock && accumulatedThinking.Len() > 0 {
		outputCh <- StreamChunk{
			Type: "thinking_complete",
			Text: accumulatedThinking.String(),
		}
	}

	outputCh <- StreamChunk{
		Type:   "done",
		Tokens: totalTokens,
	}

	return nil
}
