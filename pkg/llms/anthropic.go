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

// Image size limits for different providers (in bytes)
const (
	MaxAnthropicImageSize = 5 * 1024 * 1024  // 5MB - Anthropic's limit
	MaxGeminiImageSize    = 20 * 1024 * 1024 // 20MB - Gemini's limit
	MaxOpenAIImageSize    = 20 * 1024 * 1024 // 20MB - OpenAI's limit
	MaxOllamaImageSize    = 20 * 1024 * 1024 // 20MB - Ollama's limit (typical)
)

// Anthropic API constants
const (
	// Default API endpoint
	anthropicDefaultHost = "https://api.anthropic.com"
	anthropicAPIVersion  = "2023-06-01"

	// Beta features - interleaved thinking allows Claude to think between tool calls
	anthropicBetaInterleavedThinking = "interleaved-thinking-2025-05-14"

	// SSE Event Types
	anthropicEventContentBlockStart = "content_block_start"
	anthropicEventContentBlockDelta = "content_block_delta"
	anthropicEventContentBlockStop  = "content_block_stop"
	anthropicEventMessageDelta      = "message_delta"
	anthropicEventMessageStop       = "message_stop"

	// Content Types
	anthropicContentTypeText     = "text"
	anthropicContentTypeThinking = "thinking"
	anthropicContentTypeToolUse  = "tool_use"

	// Delta Types
	anthropicDeltaTypeText      = "text_delta"
	anthropicDeltaTypeThinking  = "thinking_delta"
	anthropicDeltaTypeSignature = "signature_delta"
	anthropicDeltaTypeInputJSON = "input_json_delta"

	// Channel buffer sizes
	anthropicStreamChannelBufferSize    = 100
	anthropicStructuredStreamBufferSize = 10

	// Thinking defaults
	anthropicDefaultThinkingBudget = 1024
	anthropicThinkingTemperature   = 1.0

	// Logging preview limits
	anthropicMaxPayloadPreviewLength = 200
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

// anthropicStreamingState encapsulates all state variables used during SSE streaming.
type anthropicStreamingState struct {
	toolCalls          map[int]*protocol.ToolCall
	toolJSONBuffers    map[int]string
	thinkingBuffers    map[int]string
	thinkingSignatures map[int]string
	totalTokens        int
}

// newAnthropicStreamingState creates a new streaming state
func newAnthropicStreamingState() *anthropicStreamingState {
	return &anthropicStreamingState{
		toolCalls:          make(map[int]*protocol.ToolCall),
		toolJSONBuffers:    make(map[int]string),
		thinkingBuffers:    make(map[int]string),
		thinkingSignatures: make(map[int]string),
	}
}

// cleanupThinkingBlock removes thinking block data for a given index
func (s *anthropicStreamingState) cleanupThinkingBlock(index int) {
	delete(s.thinkingBuffers, index)
	delete(s.thinkingSignatures, index)
}

// NewAnthropicProvider creates a new Anthropic provider with default configuration.
// This is a convenience constructor for simple use cases.
// For production use, prefer NewAnthropicProviderFromConfig with explicit configuration.
func NewAnthropicProvider(apiKey string, model string) *AnthropicProvider {
	cfg := &config.LLMProviderConfig{
		Type:        "anthropic",
		Model:       model,
		APIKey:      apiKey,
		Host:        anthropicDefaultHost,
		Temperature: func() *float64 { t := 1.0; return &t }(),
		MaxTokens:   4096,
		Timeout:     120,
	}

	provider, err := NewAnthropicProviderFromConfig(cfg)
	if err != nil {
		slog.Error("Failed to create Anthropic provider", "error", err)
		return nil
	}
	return provider
}

func NewAnthropicProviderFromConfig(cfg *config.LLMProviderConfig) (*AnthropicProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Anthropic")
	}

	if cfg.Host == "" {
		cfg.Host = anthropicDefaultHost
	}

	httpClient := createAnthropicHTTPClient(cfg)

	return &AnthropicProvider{
		config:     cfg,
		httpClient: httpClient,
	}, nil
}

// createAnthropicHTTPClient creates an HTTP client with TLS and rate limit support
func createAnthropicHTTPClient(cfg *config.LLMProviderConfig) *httpclient.Client {
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
		httpclient.WithHeaderParser(httpclient.ParseAnthropicRateLimitHeaders),
	}

	if tlsConfig != nil {
		opts = append(opts, httpclient.WithTLSConfig(tlsConfig))
	}

	return httpclient.New(opts...)
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

// getMessagesURL returns the URL for the Anthropic Messages API
func (p *AnthropicProvider) getMessagesURL() string {
	host := p.config.Host
	if host == "" {
		host = anthropicDefaultHost
	}
	return fmt.Sprintf("%s/v1/messages", strings.TrimSuffix(host, "/"))
}

// createAPIRequest creates an HTTP request for the Anthropic API
func (p *AnthropicProvider) createAPIRequest(ctx context.Context, request AnthropicRequest) (*http.Request, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.getMessagesURL(), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	// Enable interleaved thinking - allows Claude to think between tool calls
	// This provides better reasoning after receiving tool results
	if p.config.Thinking != nil && p.config.Thinking.Enabled {
		req.Header.Set("anthropic-beta", anthropicBetaInterleavedThinking)
	}

	return req, nil
}

// parseErrorResponse parses an error response from the Anthropic API
func (p *AnthropicProvider) parseErrorResponse(resp *http.Response) error {
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("anthropic API error (HTTP %d): failed to read body: %w", resp.StatusCode, readErr)
	}

	var errorResp struct {
		Error *AnthropicError `json:"error"`
	}
	if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != nil {
		slog.Error("Anthropic API request failed",
			"status_code", resp.StatusCode,
			"error_type", errorResp.Error.Type,
			"error_message", errorResp.Error.Message)
		return fmt.Errorf("anthropic API error (HTTP %d): %s (type: %s)",
			resp.StatusCode, errorResp.Error.Message, errorResp.Error.Type)
	}

	slog.Error("Anthropic API request failed",
		"status_code", resp.StatusCode,
		"response_body", string(body))
	return fmt.Errorf("anthropic API error (HTTP %d): %s", resp.StatusCode, string(body))
}

// logRequestDebug logs debug information about an Anthropic API request
func (p *AnthropicProvider) logRequestDebug(request AnthropicRequest) {
	payloadPreview := ""
	if reqBody, err := json.Marshal(request); err == nil {
		payloadPreview = string(reqBody)
		if len(payloadPreview) > anthropicMaxPayloadPreviewLength {
			payloadPreview = payloadPreview[:anthropicMaxPayloadPreviewLength] + "..."
		}
	}

	hasThinking := request.Thinking != nil
	budgetTokens := 0
	if hasThinking {
		budgetTokens = request.Thinking.BudgetTokens
	}

	slog.Debug("Anthropic API request",
		"model", request.Model,
		"message_count", len(request.Messages),
		"has_system", request.System != "",
		"max_tokens", request.MaxTokens,
		"stream", request.Stream,
		"thinking_enabled", hasThinking,
		"thinking_budget", budgetTokens,
		"tools_count", len(request.Tools),
		"payload_preview", payloadPreview)
}

// getThinkingConfig returns whether thinking is enabled and the budget tokens
func (p *AnthropicProvider) getThinkingConfig() (bool, int) {
	if p.config.Thinking != nil && p.config.Thinking.Enabled {
		budgetTokens := p.config.Thinking.BudgetTokens
		if budgetTokens <= 0 {
			budgetTokens = anthropicDefaultThinkingBudget
		}
		return true, budgetTokens
	}
	return false, 0
}

// getTemperature returns the appropriate temperature based on thinking mode
func (p *AnthropicProvider) getTemperatureForRequest(thinkingEnabled bool) float64 {
	if thinkingEnabled {
		// When thinking is enabled, Anthropic requires temperature=1.0
		return anthropicThinkingTemperature
	}
	if p.config.Temperature == nil {
		return 0.7 // Default
	}
	return *p.config.Temperature
}

// messageAnalysisResult holds the results of analyzing messages for thinking block requirements
type messageAnalysisResult struct {
	skipMessageIndices  map[int]bool
	includedToolCallIDs map[string]bool
}

// analyzeMessagesForThinking analyzes messages to determine which should be skipped
// when thinking is enabled. Returns skip indices and included tool call IDs.
// This implements Anthropic's requirement that assistant messages with tool_use must have thinking blocks.
func (p *AnthropicProvider) analyzeMessagesForThinking(messages []*pb.Message, thinkingEnabled bool) (*messageAnalysisResult, error) {
	result := &messageAnalysisResult{
		skipMessageIndices:  make(map[int]bool),
		includedToolCallIDs: make(map[string]bool),
	}

	if !thinkingEnabled {
		return result, nil
	}

	// First pass: find all assistant messages with tool_use
	assistantMessagesWithToolUse := make([]int, 0)
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == pb.Role_ROLE_AGENT {
			toolCalls := protocol.GetToolCallsFromMessage(messages[i])
			if len(toolCalls) > 0 {
				assistantMessagesWithToolUse = append(assistantMessagesWithToolUse, i)
			}
		}
	}

	if len(assistantMessagesWithToolUse) == 0 {
		return result, nil
	}

	// Second pass: identify which assistant messages with tool_use are missing thinking blocks
	for _, idx := range assistantMessagesWithToolUse {
		thinkingContent, _ := protocol.ExtractThinkingBlockFromMessage(messages[idx])
		if thinkingContent == "" {
			result.skipMessageIndices[idx] = true
		}
	}

	// Validate conversation continuity
	if len(result.skipMessageIndices) > 0 {
		// Check if we're skipping ALL assistant messages with tool_use
		if len(result.skipMessageIndices) == len(assistantMessagesWithToolUse) {
			return nil, fmt.Errorf(
				"thinking enabled but no assistant messages with thinking blocks found - "+
					"cannot build valid request (found %d assistant messages with tool_use, all missing thinking blocks)",
				len(assistantMessagesWithToolUse))
		}

		slog.Debug("Skipping assistant messages without thinking blocks",
			"skipped_count", len(result.skipMessageIndices),
			"total_with_tool_use", len(assistantMessagesWithToolUse))
	}

	return result, nil
}

// parseResponseContent extracts text, tool calls, and thinking content from response content blocks
func (p *AnthropicProvider) parseResponseContent(content []AnthropicContent) (text string, toolCalls []*protocol.ToolCall, thinkingContent string, thinkingSignature string) {
	for _, c := range content {
		switch c.Type {
		case anthropicContentTypeText:
			text += c.Text
		case anthropicContentTypeThinking:
			// Extract thinking content and signature from non-streaming responses
			// Thinking blocks must be preserved for multi-turn conversations
			if c.Thinking != "" {
				thinkingContent += c.Thinking
			}
			if c.Signature != "" {
				thinkingSignature = c.Signature // Last signature is the complete one
			}
		case anthropicContentTypeToolUse:
			var args map[string]interface{}
			if c.Input != nil {
				args = *c.Input
			}
			toolCalls = append(toolCalls, &protocol.ToolCall{
				ID:   c.ID,
				Name: c.Name,
				Args: args,
			})
		}
	}
	return
}

func (p *AnthropicProvider) Generate(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (string, []*protocol.ToolCall, int, *ThinkingBlock, error) {
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
		return "", nil, 0, nil, err
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

		return "", nil, 0, nil, err
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

		return "", nil, 0, nil, apiErr
	}

	tokensUsed := response.Usage.InputTokens + response.Usage.OutputTokens

	// Parse response content using helper method
	text, toolCalls, thinkingContent, thinkingSignature := p.parseResponseContent(response.Content)

	// Build thinking block if present
	var thinkingBlock *ThinkingBlock
	if thinkingContent != "" {
		thinkingBlock = &ThinkingBlock{
			Content:   thinkingContent,
			Signature: thinkingSignature,
		}
		// Store thinking in span attributes for observability
		span.SetAttributes(
			attribute.String("llm.thinking.content", thinkingContent),
			attribute.Bool("llm.thinking.present", true),
			attribute.Bool("llm.thinking.has_signature", thinkingSignature != ""),
		)
		slog.Debug("Anthropic non-streaming response contains thinking",
			"model", p.config.Model,
			"thinking_length", len(thinkingContent),
			"has_signature", thinkingSignature != "")
	}

	// Record successful metrics
	span.SetAttributes(
		attribute.Int(observability.AttrLLMTokensInput, response.Usage.InputTokens),
		attribute.Int(observability.AttrLLMTokensOutput, response.Usage.OutputTokens),
		attribute.Int("llm.tool_calls", len(toolCalls)),
	)
	if thinkingBlock != nil {
		span.SetAttributes(
			attribute.Int(observability.AttrLLMThinkingBlocks, 1),
			attribute.Int(observability.AttrLLMThinkingLength, len(thinkingBlock.Content)),
		)
	}
	span.SetStatus(codes.Ok, "success")

	metrics := observability.GetGlobalMetrics()
	if metrics != nil {
		metrics.RecordLLMCall(ctx, p.config.Model, duration, response.Usage.InputTokens, response.Usage.OutputTokens, nil)
	}

	return text, toolCalls, tokensUsed, thinkingBlock, nil
}

func (p *AnthropicProvider) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
	request, err := p.buildRequest(ctx, messages, true, tools)
	if err != nil {
		return nil, err
	}

	outputCh := make(chan StreamChunk, anthropicStreamChannelBufferSize)

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
	// Get thinking configuration from provider config
	shouldEnableThinking, budgetTokens := p.getThinkingConfig()

	// Analyze messages for thinking block requirements
	// This determines which messages to skip and which tool call IDs are included
	analysis, err := p.analyzeMessagesForThinking(messages, shouldEnableThinking)
	if err != nil {
		return AnthropicRequest{}, err
	}

	var systemParts []string
	anthropicMessages := make([]AnthropicMessage, 0, len(messages))
	skipMessageIndices := analysis.skipMessageIndices
	includedToolCallIDs := analysis.includedToolCallIDs

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

						const maxImageSize = MaxAnthropicImageSize
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
	temperature := p.getTemperatureForRequest(shouldEnableThinking)

	request := AnthropicRequest{
		Model:       p.config.Model,
		Messages:    anthropicMessages,
		MaxTokens:   p.config.MaxTokens,
		Temperature: temperature,
		Stream:      stream,
		System:      systemPrompt,
	}

	if shouldEnableThinking {
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
	p.logRequestDebug(request)

	req, err := p.createAPIRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, p.parseErrorResponse(resp)
		}

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read response body: %w", readErr)
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
	p.logRequestDebug(request)

	req, err := p.createAPIRequest(ctx, request)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return p.parseErrorResponse(resp)
		}
		// Success case - continue with streaming (don't read body yet!)
	} else {
		if err != nil {
			return fmt.Errorf("failed to make request: %w", err)
		}
		return fmt.Errorf("failed to make request: no response received")
	}

	state := newAnthropicStreamingState()

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
		case anthropicEventContentBlockStart:
			p.handleContentBlockStart(streamResp, state)

		case anthropicEventContentBlockDelta:
			p.handleContentBlockDelta(streamResp, state, outputCh)

		case anthropicEventContentBlockStop:
			p.handleContentBlockStop(streamResp, state, outputCh)

		case anthropicEventMessageDelta:
			if streamResp.Usage != nil {
				state.totalTokens = streamResp.Usage.OutputTokens
			}

		case anthropicEventMessageStop:
			outputCh <- StreamChunk{
				Type:   "done",
				Tokens: state.totalTokens,
			}
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read streaming response: %w", err)
	}

	return nil
}

// handleContentBlockStart handles content_block_start SSE events
func (p *AnthropicProvider) handleContentBlockStart(streamResp AnthropicStreamResponse, state *anthropicStreamingState) {
	if streamResp.ContentBlock == nil {
		return
	}

	switch streamResp.ContentBlock.Type {
	case anthropicContentTypeThinking:
		state.thinkingBuffers[streamResp.Index] = ""
	case anthropicContentTypeToolUse:
		state.toolCalls[streamResp.Index] = &protocol.ToolCall{
			ID:   streamResp.ContentBlock.ID,
			Name: streamResp.ContentBlock.Name,
			Args: make(map[string]interface{}),
		}
		state.toolJSONBuffers[streamResp.Index] = ""
	}
}

// handleContentBlockDelta handles content_block_delta SSE events
func (p *AnthropicProvider) handleContentBlockDelta(streamResp AnthropicStreamResponse, state *anthropicStreamingState, outputCh chan<- StreamChunk) {
	if streamResp.Delta == nil {
		return
	}

	// Handle signature_delta events for thinking blocks
	if streamResp.Delta.Type == anthropicDeltaTypeSignature || streamResp.Delta.Signature != "" {
		if streamResp.Delta.Signature != "" {
			state.thinkingSignatures[streamResp.Index] = streamResp.Delta.Signature
			return
		}
	}

	// Handle thinking content deltas
	isThinkingDelta := streamResp.Delta.Type == anthropicDeltaTypeThinking
	_, isInThinkingBlock := state.thinkingBuffers[streamResp.Index]

	if isInThinkingBlock || isThinkingDelta {
		thinkingText := streamResp.Delta.Thinking
		if thinkingText != "" {
			if !isInThinkingBlock {
				state.thinkingBuffers[streamResp.Index] = ""
			}
			state.thinkingBuffers[streamResp.Index] += thinkingText
			outputCh <- StreamChunk{Type: "thinking", Text: thinkingText}
		} else if streamResp.Delta.Text != "" && isInThinkingBlock {
			// Fallback for edge cases where API uses "text" field in thinking block
			state.thinkingBuffers[streamResp.Index] += streamResp.Delta.Text
			outputCh <- StreamChunk{Type: "thinking", Text: streamResp.Delta.Text}
		}
	} else if streamResp.Delta.Text != "" {
		// Regular text content (not in a thinking block)
		outputCh <- StreamChunk{Type: "text", Text: streamResp.Delta.Text}
	}

	// Handle tool call argument deltas
	if streamResp.Delta.Type == anthropicDeltaTypeInputJSON && streamResp.Delta.PartialJSON != "" {
		state.toolJSONBuffers[streamResp.Index] += streamResp.Delta.PartialJSON
	}
}

// handleContentBlockStop handles content_block_stop SSE events
func (p *AnthropicProvider) handleContentBlockStop(streamResp AnthropicStreamResponse, state *anthropicStreamingState, outputCh chan<- StreamChunk) {
	// Emit thinking_complete if we have a thinking block
	if thinkingContent, hasThinking := state.thinkingBuffers[streamResp.Index]; hasThinking {
		signature := state.thinkingSignatures[streamResp.Index]
		// Always emit thinking_complete so agent knows thinking is done
		// Signature may be empty for some providers/models
		outputCh <- StreamChunk{
			Type:      "thinking_complete",
			Text:      thinkingContent,
			Signature: signature,
		}
	}
	state.cleanupThinkingBlock(streamResp.Index)

	// Emit tool_call if we have a completed tool call
	if tc, exists := state.toolCalls[streamResp.Index]; exists {
		if jsonStr, hasJSON := state.toolJSONBuffers[streamResp.Index]; hasJSON && jsonStr != "" {
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
}

func (p *AnthropicProvider) GenerateStructured(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (string, []*protocol.ToolCall, int, *ThinkingBlock, error) {
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
		return "", nil, 0, nil, err
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

		return "", nil, 0, nil, err
	}

	// Calculate input/output tokens from total
	inputTokens := tokens / 2 // Rough estimate
	outputTokens := tokens / 2

	// Note: Structured output requests typically don't include thinking blocks
	// but we return nil for consistency with interface
	var thinkingBlock *ThinkingBlock

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

	return text, toolCalls, tokens, thinkingBlock, nil
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

	chunks := make(chan StreamChunk, anthropicStructuredStreamBufferSize)
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
