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

type AnthropicProvider struct {
	config     *config.LLMProviderConfig
	httpClient *httpclient.Client
}

type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Stream      bool               `json:"stream"`
	System      string             `json:"system,omitempty"`
	Tools       []AnthropicTool    `json:"tools,omitempty"`
}

type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type AnthropicResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []AnthropicContent `json:"content"`
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	Usage      AnthropicUsage     `json:"usage"`
	Error      *AnthropicError    `json:"error,omitempty"`
}

type AnthropicStreamResponse struct {
	Type         string             `json:"type"`
	Index        int                `json:"index,omitempty"`
	Delta        *AnthropicDelta    `json:"delta,omitempty"`
	ContentBlock *AnthropicContent  `json:"content_block,omitempty"`
	Message      *AnthropicResponse `json:"message,omitempty"`
	Usage        *AnthropicUsage    `json:"usage,omitempty"`
}

type AnthropicContent struct {
	Type      string                  `json:"type"`
	Text      string                  `json:"text,omitempty"`
	ID        string                  `json:"id,omitempty"`
	Name      string                  `json:"name,omitempty"`
	Input     *map[string]interface{} `json:"input,omitempty"`
	ToolUseID string                  `json:"tool_use_id,omitempty"`
	Content   string                  `json:"content,omitempty"`
}

type AnthropicDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type AnthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func NewAnthropicProvider(apiKey string, model string) *AnthropicProvider {
	config := &config.LLMProviderConfig{
		Type:        "anthropic",
		Model:       model,
		APIKey:      apiKey,
		Host:        "https://api.anthropic.com",
		Temperature: func() *float64 { t := 1.0; return &t }(),
		MaxTokens:   4096,
		Timeout:     120,
	}

	provider, _ := NewAnthropicProviderFromConfig(config)
	return provider
}

func NewAnthropicProviderFromConfig(cfg *config.LLMProviderConfig) (*AnthropicProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Anthropic")
	}

	if cfg.Host == "" {
		cfg.Host = "https://api.anthropic.com"
	}

	return &AnthropicProvider{
		config: cfg,
		httpClient: httpclient.New(
			httpclient.WithHTTPClient(&http.Client{
				Timeout: time.Duration(cfg.Timeout) * time.Second,
			}),
			httpclient.WithMaxRetries(cfg.MaxRetries),
			httpclient.WithBaseDelay(time.Duration(cfg.RetryDelay)*time.Second),
			httpclient.WithHeaderParser(httpclient.ParseAnthropicRateLimitHeaders),
		),
	}, nil
}

func (p *AnthropicProvider) GetModelName() string {
	return p.config.Model
}

func (p *AnthropicProvider) GetMaxTokens() int {
	return p.config.MaxTokens
}

func (p *AnthropicProvider) GetTemperature() float64 {
	if p.config.Temperature == nil {
		return 0.7 // Default
	}
	return *p.config.Temperature
}

func (p *AnthropicProvider) Close() error {
	return nil
}

func (p *AnthropicProvider) Generate(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (string, []*protocol.ToolCall, int, error) {
	startTime := time.Now()

	// Create span for LLM request
	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "anthropic"),
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
		apiErr := fmt.Errorf("anthropic API error: %s", response.Error.Message)
		span.RecordError(apiErr)
		span.SetStatus(codes.Error, response.Error.Message)

		// Record metrics for API error
		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, apiErr)
		}

		return "", nil, 0, apiErr
	}

	tokensUsed := response.Usage.InputTokens + response.Usage.OutputTokens

	var text string
	var toolCalls []*protocol.ToolCall

	for _, content := range response.Content {
		if content.Type == "text" {
			text += content.Text
		} else if content.Type == "tool_use" {

			var args map[string]interface{}
			if content.Input != nil {
				args = *content.Input
			}
			toolCalls = append(toolCalls, &protocol.ToolCall{
				ID:   content.ID,
				Name: content.Name,
				Args: args,
			})
		}
	}

	// Record successful metrics
	span.SetAttributes(
		attribute.Int(observability.AttrLLMTokensInput, response.Usage.InputTokens),
		attribute.Int(observability.AttrLLMTokensOutput, response.Usage.OutputTokens),
		attribute.Int("llm.tool_calls", len(toolCalls)),
	)
	span.SetStatus(codes.Ok, "success")

	metrics := observability.GetGlobalMetrics()
	if metrics != nil {
		metrics.RecordLLMCall(ctx, p.config.Model, duration, response.Usage.InputTokens, response.Usage.OutputTokens, nil)
	}

	return text, toolCalls, tokensUsed, nil
}

func (p *AnthropicProvider) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
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

func (p *AnthropicProvider) buildRequest(messages []*pb.Message, stream bool, tools []ToolDefinition) AnthropicRequest {

	var systemParts []string
	anthropicMessages := make([]AnthropicMessage, 0, len(messages))

	for _, msg := range messages {

		if msg.Role == pb.Role_ROLE_UNSPECIFIED {

			textContent := protocol.ExtractTextFromMessage(msg)
			if textContent != "" {
				systemParts = append(systemParts, textContent)
			}
			continue
		}

		if msg.Role == pb.Role_ROLE_USER {

			textContent := protocol.ExtractTextFromMessage(msg)
			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role: "user",
				Content: []AnthropicContent{
					{Type: "text", Text: textContent},
				},
			})
			continue
		}

		toolResults := protocol.GetToolResultsFromMessage(msg)
		if len(toolResults) > 0 {

			for _, toolResult := range toolResults {
				anthropicMessages = append(anthropicMessages, AnthropicMessage{
					Role: "user",
					Content: []AnthropicContent{
						{
							Type:      "tool_result",
							ToolUseID: toolResult.ToolCallID,
							Content:   toolResult.Content,
						},
					},
				})
			}
			continue
		}

		toolCalls := protocol.GetToolCallsFromMessage(msg)
		if msg.Role == pb.Role_ROLE_AGENT && len(toolCalls) > 0 {

			contents := []AnthropicContent{}

			textContent := protocol.ExtractTextFromMessage(msg)
			if textContent != "" {
				contents = append(contents, AnthropicContent{
					Type: "text",
					Text: textContent,
				})
			}

			for _, tc := range toolCalls {

				input := tc.Args
				if input == nil {
					input = make(map[string]interface{})
				}
				contents = append(contents, AnthropicContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: &input,
				})
			}

			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role:    "assistant",
				Content: contents,
			})
		} else if msg.Role == pb.Role_ROLE_AGENT {

			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role: "assistant",
				Content: []AnthropicContent{
					{
						Type: "text",
						Text: protocol.ExtractTextFromMessage(msg),
					},
				},
			})
		}
	}

	var systemPrompt string
	if len(systemParts) > 0 {
		systemPrompt = strings.Join(systemParts, "\n\n")
	}

	request := AnthropicRequest{
		Model:     p.config.Model,
		Messages:  anthropicMessages,
		MaxTokens: p.config.MaxTokens,
		Temperature: func() float64 {
			if p.config.Temperature == nil {
				return 0.7 // Default
			}
			return *p.config.Temperature
		}(),
		Stream: stream,
		System: systemPrompt,
	}

	if len(tools) > 0 {
		anthropicTools := make([]AnthropicTool, len(tools))
		for i, tool := range tools {
			anthropicTools[i] = AnthropicTool{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.Parameters,
			}
		}
		request.Tools = anthropicTools
	}
	return request
}

func (p *AnthropicProvider) makeRequest(ctx context.Context, request AnthropicRequest) (*AnthropicResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.Host+"/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response AnthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func (p *AnthropicProvider) makeStreamingRequest(ctx context.Context, request AnthropicRequest, outputCh chan<- StreamChunk) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", p.config.Host+"/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	toolCalls := make(map[int]*protocol.ToolCall)
	toolJSONBuffers := make(map[int]string)
	thinkingBuffers := make(map[int]string) // Track thinking content by index
	var totalTokens int

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data: ")

		var streamResp AnthropicStreamResponse
		if err := json.Unmarshal([]byte(jsonData), &streamResp); err != nil {
			return fmt.Errorf("failed to decode streaming response: %w, data: %s", err, jsonData)
		}

		switch streamResp.Type {
		case "content_block_start":
			// Handle thinking blocks (Claude extended thinking)
			if streamResp.ContentBlock != nil && streamResp.ContentBlock.Type == "thinking" {
				thinkingBuffers[streamResp.Index] = ""
			}

			if streamResp.ContentBlock != nil && streamResp.ContentBlock.Type == "tool_use" {

				toolCalls[streamResp.Index] = &protocol.ToolCall{
					ID:   streamResp.ContentBlock.ID,
					Name: streamResp.ContentBlock.Name,
					Args: make(map[string]interface{}),
				}
				toolJSONBuffers[streamResp.Index] = ""
			}

		case "content_block_delta":
			if streamResp.Delta != nil {
				// Handle thinking content deltas
				if _, isThinking := thinkingBuffers[streamResp.Index]; isThinking {
					if streamResp.Delta.Text != "" {
						thinkingBuffers[streamResp.Index] += streamResp.Delta.Text
						outputCh <- StreamChunk{Type: "thinking", Text: streamResp.Delta.Text}
					}
				} else if streamResp.Delta.Text != "" {
					// Regular text content
					outputCh <- StreamChunk{Type: "text", Text: streamResp.Delta.Text}
				}

				if streamResp.Delta.Type == "input_json_delta" && streamResp.Delta.PartialJSON != "" {
					toolJSONBuffers[streamResp.Index] += streamResp.Delta.PartialJSON
				}
			}

		case "content_block_stop":
			// Clean up thinking buffer when block stops
			delete(thinkingBuffers, streamResp.Index)

			if tc, exists := toolCalls[streamResp.Index]; exists {

				if jsonStr, hasJSON := toolJSONBuffers[streamResp.Index]; hasJSON && jsonStr != "" {
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(jsonStr), &args); err == nil {
						tc.Args = args
					}
				}

				outputCh <- StreamChunk{
					Type:     "tool_call",
					ToolCall: tc,
				}
			}

		case "message_delta":

			if streamResp.Usage != nil {
				totalTokens = streamResp.Usage.OutputTokens
			}

		case "message_stop":

			outputCh <- StreamChunk{
				Type:   "done",
				Tokens: totalTokens,
			}
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read streaming response: %w", err)
	}

	return nil
}

func (p *AnthropicProvider) GenerateStructured(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (string, []*protocol.ToolCall, int, error) {
	startTime := time.Now()

	// Create span for structured LLM request
	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "anthropic"),
			attribute.Bool("streaming", false),
			attribute.Bool("structured", true),
		),
	)
	defer span.End()

	systemPrompt := p.buildSystemPromptWithSchema(structConfig)

	req := p.buildRequest(messages, false, tools)
	if systemPrompt != "" {
		if req.System != "" {
			req.System = req.System + "\n\n" + systemPrompt
		} else {
			req.System = systemPrompt
		}
	}

	if structConfig != nil && structConfig.Format == "json" {
		prefill := "{"
		if structConfig.Prefill != "" {
			prefill = structConfig.Prefill
		}

		req.Messages = append(req.Messages, AnthropicMessage{
			Role:    "assistant",
			Content: prefill,
		})
	}

	text, toolCalls, tokens, err := p.makeStructuredRequest(ctx, req)
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

	// Calculate input/output tokens from total
	inputTokens := tokens / 2 // Rough estimate
	outputTokens := tokens / 2

	// Record successful metrics
	span.SetAttributes(
		attribute.Int(observability.AttrLLMTokensInput, inputTokens),
		attribute.Int(observability.AttrLLMTokensOutput, outputTokens),
		attribute.Int("llm.tool_calls", len(toolCalls)),
	)
	span.SetStatus(codes.Ok, "success")

	metrics := observability.GetGlobalMetrics()
	if metrics != nil {
		metrics.RecordLLMCall(ctx, p.config.Model, duration, inputTokens, outputTokens, nil)
	}

	return text, toolCalls, tokens, nil
}

func (p *AnthropicProvider) GenerateStructuredStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (<-chan StreamChunk, error) {

	systemPrompt := p.buildSystemPromptWithSchema(structConfig)

	req := p.buildRequest(messages, true, tools)
	if systemPrompt != "" {
		if req.System != "" {
			req.System = req.System + "\n\n" + systemPrompt
		} else {
			req.System = systemPrompt
		}
	}

	prefill := ""
	if structConfig != nil && structConfig.Format == "json" {
		prefill = "{"
		if structConfig.Prefill != "" {
			prefill = structConfig.Prefill
		}

		req.Messages = append(req.Messages, AnthropicMessage{
			Role:    "assistant",
			Content: prefill,
		})
	}

	chunks := make(chan StreamChunk, 10)
	go func() {
		defer close(chunks)

		if prefill != "" {
			chunks <- StreamChunk{
				Type: "text",
				Text: prefill,
			}
		}

		if err := p.makeStreamingRequest(ctx, req, chunks); err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
		}
	}()

	return chunks, nil
}

func (p *AnthropicProvider) SupportsStructuredOutput() bool {
	return true
}

func (p *AnthropicProvider) makeStructuredRequest(ctx context.Context, req AnthropicRequest) (string, []*protocol.ToolCall, int, error) {

	prefill := ""
	if len(req.Messages) > 0 && req.Messages[len(req.Messages)-1].Role == "assistant" {
		if content, ok := req.Messages[len(req.Messages)-1].Content.(string); ok {
			prefill = content
		}
	}

	response, err := p.makeRequest(ctx, req)
	if err != nil {
		return "", nil, 0, err
	}

	if response.Error != nil {
		return "", nil, 0, fmt.Errorf("anthropic API error: %s (type: %s)", response.Error.Message, response.Error.Type)
	}

	tokensUsed := response.Usage.InputTokens + response.Usage.OutputTokens

	var text string
	var toolCalls []*protocol.ToolCall

	for _, content := range response.Content {
		if content.Type == "text" {
			text += content.Text
		} else if content.Type == "tool_use" {
			var args map[string]interface{}
			if content.Input != nil {
				args = *content.Input
			}
			toolCalls = append(toolCalls, &protocol.ToolCall{
				ID:   content.ID,
				Name: content.Name,
				Args: args,
			})
		}
	}

	if prefill != "" {
		text = prefill + text
	}

	return text, toolCalls, tokensUsed, nil
}

func (p *AnthropicProvider) buildSystemPromptWithSchema(structConfig *StructuredOutputConfig) string {
	if structConfig == nil || structConfig.Schema == nil {
		return ""
	}

	schemaJSON, err := json.MarshalIndent(structConfig.Schema, "", "  ")
	if err != nil {
		return ""
	}

	return fmt.Sprintf(`You must respond with valid JSON matching this exact schema:

%s

Important:
- Output ONLY valid JSON, no other text
- All required fields must be present
- Follow the exact structure specified
- Use correct data types for each field`, string(schemaJSON))
}
