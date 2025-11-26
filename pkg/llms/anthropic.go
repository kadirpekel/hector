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

type AnthropicProvider struct {
	config     *config.LLMProviderConfig
	httpClient *httpclient.Client
}

type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type AnthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Stream      bool               `json:"stream"`
	System      string             `json:"system,omitempty"`
	Tools       []AnthropicTool    `json:"tools,omitempty"`
	Thinking    *AnthropicThinking `json:"thinking,omitempty"`
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
	Thinking  string                  `json:"thinking,omitempty"`  // Extended thinking content (string)
	Signature string                  `json:"signature,omitempty"` // Signature for thinking block (required when sending thinking back)
	ID        string                  `json:"id,omitempty"`
	Name      string                  `json:"name,omitempty"`
	Input     *map[string]interface{} `json:"input,omitempty"`
	ToolUseID string                  `json:"tool_use_id,omitempty"`
	Content   string                  `json:"content,omitempty"`
	Source    *AnthropicImageSource   `json:"source,omitempty"`
}

type AnthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type AnthropicDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	Thinking    string `json:"thinking,omitempty"` // Thinking content for thinking_delta events
	PartialJSON string `json:"partial_json,omitempty"`
	Signature   string `json:"signature,omitempty"` // Signature for thinking blocks (from signature_delta events)
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

	// Create HTTP client with TLS support and Anthropic-specific rate limit header parser
	opts := []httpclient.Option{
		httpclient.WithHTTPClient(&http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		}),
		httpclient.WithMaxRetries(cfg.MaxRetries),
		httpclient.WithBaseDelay(time.Duration(cfg.RetryDelay) * time.Second),
		httpclient.WithHeaderParser(httpclient.ParseAnthropicRateLimitHeaders),
	}

	// Add TLS config if needed
	if cfg.InsecureSkipVerify != nil && *cfg.InsecureSkipVerify || cfg.CACertificate != "" {
		tlsConfig := &httpclient.TLSConfig{
			InsecureSkipVerify: cfg.InsecureSkipVerify != nil && *cfg.InsecureSkipVerify,
			CACertificate:      cfg.CACertificate,
		}
		if tlsConfig.InsecureSkipVerify {
			fmt.Printf("Warning: TLS certificate verification disabled for Anthropic (insecure_skip_verify=true)\n")
		}
		opts = append(opts, httpclient.WithTLSConfig(tlsConfig))
	}

	return &AnthropicProvider{
		config:     cfg,
		httpClient: httpclient.New(opts...),
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

// GetSupportedInputModes returns the MIME types supported by Anthropic.
// Anthropic supports images (JPEG, PNG, GIF, WebP) via base64 data only (no URLs).
func (p *AnthropicProvider) GetSupportedInputModes() []string {
	return []string{
		"text/plain",
		"application/json",
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}
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

	request, err := p.buildRequest(ctx, messages, false, tools)
	if err != nil {
		return "", nil, 0, err
	}

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
		apiErr := fmt.Errorf("anthropic API error: %s (type: %s)", response.Error.Message, response.Error.Type)
		span.RecordError(apiErr)
		span.SetStatus(codes.Error, response.Error.Message)

		slog.Error("Anthropic API returned error",
			"model", p.config.Model,
			"duration", duration,
			"error_type", response.Error.Type,
			"error_message", response.Error.Message)

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
		} else if content.Type == "thinking" {
			// Thinking blocks are included in the response but we don't need to process them
			// They're already used by Claude internally for reasoning
			// The thinking content is available in content.Text if needed for debugging
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
	request, err := p.buildRequest(ctx, messages, true, tools)
	if err != nil {
		return nil, err
	}

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

func (p *AnthropicProvider) buildRequest(ctx context.Context, messages []*pb.Message, stream bool, tools []ToolDefinition) (AnthropicRequest, error) {

	// Check if thinking is enabled from provider config
	// Each LLM provider reads its own config - no context passing needed
	shouldEnableThinking := false
	budgetTokens := 0

	if p.config.Thinking != nil && p.config.Thinking.Enabled {
		shouldEnableThinking = true
		if p.config.Thinking.BudgetTokens > 0 {
			budgetTokens = p.config.Thinking.BudgetTokens
		}
	}

	var systemParts []string
	anthropicMessages := make([]AnthropicMessage, 0, len(messages))

	// Find the last assistant message with tool_use (for thinking block requirement)
	// According to Anthropic docs: "When thinking is enabled, a final assistant message must start
	// with a thinking block (preceeding the lastmost set of tool_use and tool_result blocks)."
	// The "final assistant message" is the LAST assistant message that has tool_use AND will be included
	// in the request (i.e., not skipped).
	//
	// Strategy: First pass - identify all assistant messages with tool_use
	// Then determine which is the last one that will actually be included (has thinking block if thinking is enabled)
	assistantMessagesWithToolUse := make([]int, 0)
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == pb.Role_ROLE_AGENT {
			toolCalls := protocol.GetToolCallsFromMessage(messages[i])
			if len(toolCalls) > 0 {
				assistantMessagesWithToolUse = append(assistantMessagesWithToolUse, i)
			}
		}
	}

	// Track which tool call IDs are actually included (not skipped)
	// This is needed to filter out tool results that reference skipped tool calls
	includedToolCallIDs := make(map[string]bool)

	// When thinking is enabled, we need to ensure the FINAL assistant message with tool_use
	// in the request has a thinking block. The "final" one is the LAST one in the array.
	//
	// Key insight: Messages we're processing are from HISTORY (previous turns).
	// NEW messages being generated will automatically have thinking blocks (because thinking is enabled).
	// The "final assistant message" in the ACTUAL REQUEST will be the NEW one being generated,
	// which will have a thinking block. So we can include ALL history messages, even without
	// thinking blocks (docs allow omitting from prior turns).
	// When thinking is enabled, we need to identify which assistant messages with tool_use
	// are missing thinking blocks. These messages (and their tool_results) must be SKIPPED
	// to avoid the API error. We keep thinking enabled so NEW responses have thinking blocks.
	//
	// Track which messages should be skipped (assistant with tool_use but no thinking block)
	skipMessageIndices := make(map[int]bool)
	if shouldEnableThinking {
		for _, idx := range assistantMessagesWithToolUse {
			msg := messages[idx]
			thinkingContent, _ := protocol.ExtractThinkingBlockFromMessage(msg)
			if thinkingContent == "" {
				// This assistant message with tool_use has no thinking block - mark for skipping
				skipMessageIndices[idx] = true
			}
		}
	}

	for i, msg := range messages {

		if msg.Role == pb.Role_ROLE_UNSPECIFIED {

			textContent := protocol.ExtractTextFromMessage(msg)
			if textContent != "" {
				systemParts = append(systemParts, textContent)
			}
			continue
		}

		if msg.Role == pb.Role_ROLE_USER {
			var contents []AnthropicContent

			for _, part := range msg.Parts {
				if text := part.GetText(); text != "" {
					contents = append(contents, AnthropicContent{
						Type: "text",
						Text: text,
					})
				} else if file := part.GetFile(); file != nil {
					mediaType := file.GetMediaType()

					// Anthropic only supports base64 image data, not URLs
					if uri := file.GetFileWithUri(); uri != "" {
						return AnthropicRequest{}, fmt.Errorf("anthropic provider does not support image URLs directly (found URI: %s). Please download the image and send as bytes (base64)", uri)
					}

					if bytes := file.GetFileWithBytes(); len(bytes) > 0 {
						if mediaType == "" {
							mediaType = detectImageMediaType(bytes)
						}

						if !strings.HasPrefix(mediaType, "image/") {
							continue
						}

						const maxImageSize = 5 * 1024 * 1024
						if len(bytes) > maxImageSize {
							continue
						}

						base64Data := base64.StdEncoding.EncodeToString(bytes)
						contents = append(contents, AnthropicContent{
							Type: "image",
							Source: &AnthropicImageSource{
								Type:      "base64",
								MediaType: mediaType,
								Data:      base64Data,
							},
						})
					}
				}
			}

			if len(contents) == 0 {
				contents = append(contents, AnthropicContent{Type: "text", Text: ""})
			}

			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role:    "user",
				Content: contents,
			})
			continue
		}

		toolResults := protocol.GetToolResultsFromMessage(msg)
		if len(toolResults) > 0 {
			// Filter tool results to only include those that reference tool calls we actually included
			// If we skipped an assistant message with tool_use, we must also skip its corresponding tool results
			for _, toolResult := range toolResults {
				if !includedToolCallIDs[toolResult.ToolCallID] {
					// This tool result references a tool call that was skipped (no thinking block)
					slog.Debug("Skipping tool result for skipped tool call",
						"tool_call_id", toolResult.ToolCallID,
						"message_index", i)
					continue
				}
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

			// Check if this message should be skipped (no thinking block when thinking is enabled)
			if skipMessageIndices[i] {
				slog.Debug("Skipping assistant message with tool_use (no thinking block)",
					"message_index", i,
					"tool_calls_count", len(toolCalls))
				// Don't add tool call IDs - this will cause corresponding tool_results to be skipped too
				continue
			}

			contents := []AnthropicContent{}

			// When thinking is enabled, assistant messages with tool_use MUST start with a thinking block
			// Extract thinking content and signature from the message (if it exists from previous turns)
			thinkingContent, thinkingSignature := protocol.ExtractThinkingBlockFromMessage(msg)

			slog.Debug("Processing assistant message with tool_use",
				"message_index", i,
				"has_thinking_content", thinkingContent != "",
				"has_signature", thinkingSignature != "",
				"tool_calls_count", len(toolCalls))

			if shouldEnableThinking {
				// Include thinking block if content exists from previous turn
				// Anthropic format: {"type": "thinking", "thinking": "...", "signature": "..."}
				// Both content and signature are required by Anthropic API
				if thinkingContent != "" && thinkingSignature != "" {
					contents = append(contents, AnthropicContent{
						Type:      "thinking",
						Thinking:  thinkingContent,
						Signature: thinkingSignature,
					})
				}
				// Note: If we have content but no signature, we can't send it back (signature is required)
			}
			// When thinking is disabled, skip thinking parts entirely.
			// Thinking content represents internal reasoning and should not be included as regular text
			// to prevent polluting conversation history with internal thoughts.

			textContent := protocol.ExtractTextFromMessage(msg)
			if textContent != "" {
				contents = append(contents, AnthropicContent{
					Type: "text",
					Text: textContent,
				})
			}

			for _, tc := range toolCalls {
				// Track this tool call ID as included
				includedToolCallIDs[tc.ID] = true

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

			// Skip adding message if it has no content
			// Anthropic requires all messages (except optional final assistant) to have non-empty content
			if len(contents) == 0 {
				slog.Debug("Skipping empty assistant message (no thinking block and no text content)")
				continue
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

	// Determine temperature: when thinking is enabled, Anthropic requires temperature=1.0
	// According to Anthropic docs: "Thinking isn't compatible with temperature or top_k modifications"
	var temperature float64
	if shouldEnableThinking {
		// When thinking is enabled, temperature must be 1.0
		temperature = 1.0
	} else {
		// Normal temperature handling
		if p.config.Temperature == nil {
			temperature = 0.7 // Default
		} else {
			temperature = *p.config.Temperature
		}
	}

	request := AnthropicRequest{
		Model:       p.config.Model,
		Messages:    anthropicMessages,
		MaxTokens:   p.config.MaxTokens,
		Temperature: temperature,
		Stream:      stream,
		System:      systemPrompt,
	}

	if shouldEnableThinking {
		if budgetTokens <= 0 {
			// Default to 1024 tokens minimum as per Anthropic docs
			budgetTokens = 1024
		}
		// Ensure budget_tokens is less than max_tokens
		if budgetTokens >= p.config.MaxTokens {
			budgetTokens = p.config.MaxTokens - 1
		}
		request.Thinking = &AnthropicThinking{
			Type:         "enabled",
			BudgetTokens: budgetTokens,
		}
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
	return request, nil
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
	// httpclient returns both response and error for non-2xx status codes
	// We need to check the response body even if there's an error
	if resp != nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			var errorResp struct {
				Error *AnthropicError `json:"error"`
			}
			if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != nil {
				errMsg := fmt.Sprintf("Anthropic API error (HTTP %d): %s (type: %s)",
					resp.StatusCode, errorResp.Error.Message, errorResp.Error.Type)
				slog.Error("Anthropic API request failed", "status_code", resp.StatusCode, "error_type", errorResp.Error.Type, "error_message", errorResp.Error.Message)
				return nil, fmt.Errorf("%s", errMsg)
			}
			errMsg := fmt.Sprintf("Anthropic API error (HTTP %d): %s", resp.StatusCode, string(body))
			slog.Error("Anthropic API request failed", "status_code", resp.StatusCode, "response_body", string(body))
			return nil, fmt.Errorf("%s", errMsg)
		}
		var response AnthropicResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &response, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	return nil, fmt.Errorf("failed to make request: no response received")

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
	// httpclient returns both response and error for non-2xx status codes
	// We need to check the response body even if there's an error
	if resp != nil {
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// For errors, read the body to get error details
			body, _ := io.ReadAll(resp.Body)
			// Try to parse error response
			var errorResp struct {
				Error *AnthropicError `json:"error"`
			}
			if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != nil {
				errMsg := fmt.Sprintf("Anthropic API error (HTTP %d): %s (type: %s)",
					resp.StatusCode, errorResp.Error.Message, errorResp.Error.Type)
				slog.Error("Anthropic API request failed", "status_code", resp.StatusCode, "error_type", errorResp.Error.Type, "error_message", errorResp.Error.Message)
				return fmt.Errorf("%s", errMsg)
			}
			// Fallback to raw body if error structure not found
			errMsg := fmt.Sprintf("Anthropic API error (HTTP %d): %s", resp.StatusCode, string(body))
			slog.Error("Anthropic API request failed", "status_code", resp.StatusCode, "response_body", string(body))
			return fmt.Errorf("%s", errMsg)
		}
		// Success case - continue with streaming (don't read body yet!)
	} else {
		// No response received - this is a real network/connection error
		if err != nil {
			return fmt.Errorf("failed to make request: %w", err)
		}
		return fmt.Errorf("failed to make request: no response received")
	}

	toolCalls := make(map[int]*protocol.ToolCall)
	toolJSONBuffers := make(map[int]string)
	thinkingBuffers := make(map[int]string)    // Track thinking content by index
	thinkingSignatures := make(map[int]string) // Track thinking signatures by index
	var totalTokens int

	// Stream directly from resp.Body (not consumed yet for successful responses)
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
				// Handle signature_delta events for thinking blocks
				if streamResp.Delta.Type == "signature_delta" || streamResp.Delta.Signature != "" {
					if streamResp.Delta.Signature != "" {
						thinkingSignatures[streamResp.Index] = streamResp.Delta.Signature
						continue
					}
				}

				// Handle thinking content deltas
				// Anthropic API structure: thinking blocks are identified by content_block_start with type "thinking"
				// and subsequent deltas have type "thinking_delta" with content in the "thinking" field.
				isThinkingDelta := streamResp.Delta.Type == "thinking_delta"
				_, isInThinkingBlock := thinkingBuffers[streamResp.Index]

				if isInThinkingBlock || isThinkingDelta {
					// Primary path: thinking content is in the "thinking" field
					thinkingText := streamResp.Delta.Thinking
					if thinkingText != "" {
						if !isInThinkingBlock {
							// Initialize buffer if not already tracking this index
							thinkingBuffers[streamResp.Index] = ""
						}
						thinkingBuffers[streamResp.Index] += thinkingText
						outputCh <- StreamChunk{Type: "thinking", Text: thinkingText}
					} else if streamResp.Delta.Text != "" && isInThinkingBlock {
						// Fallback: if we're already in a thinking block (tracked by index) but delta
						// uses "text" field instead of "thinking" field, treat it as thinking content.
						// This handles edge cases where API versions might differ.
						thinkingBuffers[streamResp.Index] += streamResp.Delta.Text
						outputCh <- StreamChunk{Type: "thinking", Text: streamResp.Delta.Text}
					}
				} else if streamResp.Delta.Text != "" {
					// Regular text content (not in a thinking block)
					outputCh <- StreamChunk{Type: "text", Text: streamResp.Delta.Text}
				}

				if streamResp.Delta.Type == "input_json_delta" && streamResp.Delta.PartialJSON != "" {
					toolJSONBuffers[streamResp.Index] += streamResp.Delta.PartialJSON
				}
			}

		case "content_block_stop":
			// When thinking block stops, emit signature if we have it
			if thinkingContent, hasThinking := thinkingBuffers[streamResp.Index]; hasThinking {
				signature, hasSig := thinkingSignatures[streamResp.Index]
				if hasSig && signature != "" {
					outputCh <- StreamChunk{
						Type:      "thinking_complete",
						Text:      thinkingContent,
						Signature: signature,
					}
				}
			}
			// Clean up thinking buffers when block stops
			delete(thinkingBuffers, streamResp.Index)
			delete(thinkingSignatures, streamResp.Index)

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

	req, err := p.buildRequest(ctx, messages, false, tools)
	if err != nil {
		return "", nil, 0, err
	}
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

	req, err := p.buildRequest(ctx, messages, true, tools)
	if err != nil {
		return nil, err
	}
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
