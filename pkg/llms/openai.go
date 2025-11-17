package llms

import (
	"bufio"
	"bytes"
	"context"
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
	MaxTokens           int                   `json:"max_tokens,omitempty"`
	MaxCompletionTokens int                   `json:"max_completion_tokens,omitempty"`
	Temperature         float64               `json:"temperature"`
	Stream              bool                  `json:"stream"`
	Tools               []OpenAITool          `json:"tools,omitempty"`
	ToolChoice          string                `json:"tool_choice,omitempty"`
	ResponseFormat      *OpenAIResponseFormat `json:"response_format,omitempty"`
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
	Content    *string          `json:"content"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
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
		text = *choice.Message.Content
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
		text = *choice.Message.Content
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

		content := protocol.ExtractTextFromMessage(msg)

		openaiMsg := OpenAIMessage{
			Role:    roleToOpenAI(msg.Role),
			Content: &content,
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

	request := OpenAIRequest{
		Model:    p.config.Model,
		Messages: openaiMessages,
		Temperature: func() float64 {
			if p.config.Temperature == nil {
				return 0.7 // Default
			}
			return *p.config.Temperature
		}(),
		Stream: stream,
	}

	if strings.HasPrefix(p.config.Model, "o1-") || strings.HasPrefix(p.config.Model, "o3-") {
		request.MaxCompletionTokens = p.config.MaxTokens
	} else {
		request.MaxTokens = p.config.MaxTokens
	}

	if len(tools) > 0 {
		request.Tools = convertToOpenAITools(tools)
		request.ToolChoice = "auto"
	}

	return request
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

		if choice.Delta.Content != "" {
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

	outputCh <- StreamChunk{
		Type:   "done",
		Tokens: totalTokens,
	}

	return nil
}
