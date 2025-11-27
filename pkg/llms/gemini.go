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

// Gemini API constants
const (
	// Default API endpoint
	geminiDefaultHost = "https://generativelanguage.googleapis.com"
	geminiAPIVersion  = "v1beta"

	// API Methods
	geminiMethodGenerate       = "generateContent"
	geminiMethodStreamGenerate = "streamGenerateContent"

	// SSE Query Parameters
	geminiStreamFormat = "sse"

	// Channel buffer sizes
	geminiStreamChannelBufferSize = 100

	// Defaults
	geminiDefaultTemperature = 0.7
	geminiDefaultMediaType   = "image/jpeg"

	// Headers
	geminiHeaderAPIKey      = "X-goog-api-key"
	geminiHeaderContentType = "application/json"

	// Roles
	geminiRoleUser  = "user"
	geminiRoleModel = "model"

	// Response MIME Types
	geminiMimeTypeJSON = "application/json"
	geminiMimeTypeEnum = "text/x.enum"

	// Part Types
	geminiPartText             = "text"
	geminiPartThought          = "thought"
	geminiPartFunctionCall     = "functionCall"
	geminiPartFunctionResponse = "functionResponse"
	geminiPartFileData         = "file_data"
	geminiPartInlineData       = "inline_data"
	geminiPartSignature        = "signature"

	// Logging preview limits
	geminiMaxPayloadPreviewLength = 200
)

type GeminiProvider struct {
	config     *config.LLMProviderConfig
	httpClient *httpclient.Client
}

type GeminiRequest struct {
	Contents          []GeminiContent         `json:"contents"`
	SystemInstruction *GeminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	Tools             []GeminiToolSet         `json:"tools,omitempty"`
}

type GeminiGenerationConfig struct {
	Temperature      *float64               `json:"temperature,omitempty"`
	MaxOutputTokens  int                    `json:"maxOutputTokens,omitempty"`
	ResponseMimeType string                 `json:"responseMimeType,omitempty"`
	ResponseSchema   map[string]interface{} `json:"responseSchema,omitempty"`
	ThinkingConfig   *GeminiThinkingConfig  `json:"thinkingConfig,omitempty"`
}

// GeminiThinkingConfig configures thinking/reasoning for Gemini models
// See: https://ai.google.dev/gemini-api/docs/thinking
type GeminiThinkingConfig struct {
	IncludeThoughts bool `json:"includeThoughts,omitempty"` // Include thought summaries in response
	ThinkingBudget  *int `json:"thinkingBudget,omitempty"`  // Token budget: 0=off, -1=dynamic, >0=limit
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart map[string]interface{}

type GeminiToolSet struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type GeminiResponse struct {
	Candidates    []GeminiCandidate    `json:"candidates"`
	UsageMetadata *GeminiUsageMetadata `json:"usageMetadata,omitempty"`
	Error         *GeminiError         `json:"error,omitempty"`
}

type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type GeminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type GeminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// geminiStreamingState encapsulates all state variables used during SSE streaming.
type geminiStreamingState struct {
	accumulatedText        strings.Builder
	accumulatedThinking    strings.Builder
	thinkingState          ThinkingState
	totalTokens            int
	chunkCount             int
	mismarkedThinkingCount int
}

// newGeminiStreamingState creates a new streaming state
func newGeminiStreamingState() *geminiStreamingState {
	return &geminiStreamingState{
		thinkingState: ThinkingStateNone,
	}
}

// closeThinkingBlock marks the thinking block as closed
func (s *geminiStreamingState) closeThinkingBlock() {
	s.accumulatedThinking.Reset()
	s.thinkingState = ThinkingStateClosed
}

// hasActiveThinking returns true if thinking block is active
func (s *geminiStreamingState) hasActiveThinking() bool {
	return s.thinkingState == ThinkingStateActive
}

// NewGeminiProvider creates a new Gemini provider with default configuration.
// This is a convenience constructor for simple use cases.
// For production use, prefer NewGeminiProviderFromConfig with explicit configuration.
func NewGeminiProvider(apiKey string, model string) *GeminiProvider {
	cfg := &config.LLMProviderConfig{
		Type:        "gemini",
		Model:       model,
		APIKey:      apiKey,
		Host:        geminiDefaultHost,
		Temperature: func() *float64 { t := geminiDefaultTemperature; return &t }(),
		MaxTokens:   1000,
		Timeout:     120,
	}

	provider, err := NewGeminiProviderFromConfig(cfg)
	if err != nil {
		slog.Error("Failed to create Gemini provider", "error", err)
		return nil
	}
	return provider
}

func NewGeminiProviderFromConfig(cfg *config.LLMProviderConfig) (*GeminiProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}

	// Set default host if not provided
	if cfg.Host == "" {
		cfg.Host = geminiDefaultHost
	}

	return &GeminiProvider{
		config:     cfg,
		httpClient: createHTTPClient(cfg),
	}, nil
}

func (p *GeminiProvider) Generate(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (string, []*protocol.ToolCall, int, *ThinkingBlock, error) {
	startTime := time.Now()

	// Create span for LLM request
	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "gemini"),
			attribute.Bool("streaming", false),
		),
	)
	defer span.End()

	req := p.buildRequest(messages, tools, nil)

	reqBody, err := json.Marshal(req)
	if err != nil {
		marshalErr := fmt.Errorf("failed to marshal request: %w", err)
		span.RecordError(marshalErr)
		span.SetStatus(codes.Error, "failed to marshal request")
		return "", nil, 0, nil, marshalErr
	}

	p.logRequestDebug(req, reqBody)

	httpReq, err := p.createAPIRequest(ctx, p.getGenerateURL(), reqBody)
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

	resp, err := p.httpClient.Do(httpReq)
	_, geminiResp, err := p.handleGeminiResponse(resp, err)
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

	if len(geminiResp.Candidates) == 0 {
		noCandErr := fmt.Errorf("no candidates in response")
		span.RecordError(noCandErr)
		span.SetStatus(codes.Error, "no candidates")

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, noCandErr)
		}

		return "", nil, 0, nil, noCandErr
	}

	text, toolCalls, tokens, thinkingBlock, parseErr := p.parseResponse(geminiResp)
	if parseErr != nil {
		span.RecordError(parseErr)
		span.SetStatus(codes.Error, "parse error")

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, parseErr)
		}

		return text, toolCalls, tokens, thinkingBlock, parseErr
	}

	// Record successful metrics
	// Gemini doesn't provide separate input/output token counts in all cases
	inputTokens := tokens / 2 // Rough estimate
	outputTokens := tokens / 2

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

func (p *GeminiProvider) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
	req := p.buildRequest(messages, tools, nil)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	p.logRequestDebug(req, reqBody)

	chunks := make(chan StreamChunk, geminiStreamChannelBufferSize)

	go func() {
		defer close(chunks)

		httpReq, err := p.createAPIRequest(ctx, p.getStreamGenerateURL(), reqBody)
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}

		resp, err := p.httpClient.Do(httpReq)
		if resp != nil {
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				chunks <- StreamChunk{Type: "error", Error: p.parseErrorResponse(resp)}
				return
			}
			// Success case - continue with streaming
			p.parseStreamingResponse(resp.Body, chunks)
			return
		}

		// No response received - this is a real network/connection error
		if err != nil {
			slog.Error("Gemini API request failed", "error", err)
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}
		slog.Error("Gemini API request failed: no response received")
		chunks <- StreamChunk{Type: "error", Error: fmt.Errorf("no response received")}
	}()

	return chunks, nil
}

func (p *GeminiProvider) GenerateStructured(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (string, []*protocol.ToolCall, int, *ThinkingBlock, error) {
	startTime := time.Now()

	// Create span for structured LLM request
	tracer := observability.GetTracer("hector.llm")
	ctx, span := tracer.Start(ctx, observability.SpanLLMRequest,
		trace.WithAttributes(
			attribute.String(observability.AttrLLMModel, p.config.Model),
			attribute.String("provider", "gemini"),
			attribute.Bool("streaming", false),
			attribute.Bool("structured", true),
		),
	)
	defer span.End()

	req := p.buildRequest(messages, tools, structConfig)

	reqBody, err := json.Marshal(req)
	if err != nil {
		marshalErr := fmt.Errorf("failed to marshal request: %w", err)
		span.RecordError(marshalErr)
		span.SetStatus(codes.Error, "failed to marshal request")
		return "", nil, 0, nil, marshalErr
	}

	p.logRequestDebug(req, reqBody)

	httpReq, err := p.createAPIRequest(ctx, p.getGenerateURL(), reqBody)
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

	resp, err := p.httpClient.Do(httpReq)
	_, geminiResp, err := p.handleGeminiResponse(resp, err)
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

	text, toolCalls, tokens, thinkingBlock, parseErr := p.parseResponse(geminiResp)
	if parseErr != nil {
		span.RecordError(parseErr)
		span.SetStatus(codes.Error, "parse error")

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, parseErr)
		}

		return text, toolCalls, tokens, thinkingBlock, parseErr
	}

	// Record successful metrics
	inputTokens := tokens / 2 // Rough estimate
	outputTokens := tokens / 2

	span.SetAttributes(
		attribute.Int(observability.AttrLLMTokensInput, inputTokens),
		attribute.Int(observability.AttrLLMTokensOutput, outputTokens),
		attribute.Int("llm.tool_calls", len(toolCalls)),
	)
	if thinkingBlock != nil {
		span.SetAttributes(
			attribute.Int(observability.AttrLLMThinkingBlocks, 1),
			attribute.Int(observability.AttrLLMThinkingLength, len(thinkingBlock.Content)),
			attribute.Bool("llm.thinking.has_signature", thinkingBlock.Signature != ""),
		)
	}
	span.SetStatus(codes.Ok, "success")

	metrics := observability.GetGlobalMetrics()
	if metrics != nil {
		metrics.RecordLLMCall(ctx, p.config.Model, duration, inputTokens, outputTokens, nil)
	}

	return text, toolCalls, tokens, thinkingBlock, nil
}

func (p *GeminiProvider) GenerateStructuredStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (<-chan StreamChunk, error) {
	req := p.buildRequest(messages, tools, structConfig)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	p.logRequestDebug(req, reqBody)

	chunks := make(chan StreamChunk, geminiStreamChannelBufferSize)

	go func() {
		defer close(chunks)

		httpReq, err := p.createAPIRequest(ctx, p.getStreamGenerateURL(), reqBody)
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}

		resp, err := p.httpClient.Do(httpReq)
		if resp != nil {
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				chunks <- StreamChunk{Type: "error", Error: p.parseErrorResponse(resp)}
				return
			}
			// Success case - continue with streaming
			p.parseStreamingResponse(resp.Body, chunks)
			return
		}

		// No response received - this is a real network/connection error
		if err != nil {
			slog.Error("Gemini API request failed", "error", err)
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}
		slog.Error("Gemini API request failed: no response received")
		chunks <- StreamChunk{Type: "error", Error: fmt.Errorf("no response received")}
	}()

	return chunks, nil
}

func (p *GeminiProvider) SupportsStructuredOutput() bool {
	return true
}

func (p *GeminiProvider) GetModelName() string {
	return p.config.Model
}

func (p *GeminiProvider) GetMaxTokens() int {
	return p.config.MaxTokens
}

func (p *GeminiProvider) GetTemperature() float64 {
	if p.config.Temperature == nil {
		return geminiDefaultTemperature
	}
	return *p.config.Temperature
}

// GetSupportedInputModes returns the MIME types supported by Gemini.
// Gemini 1.5 Pro and Flash support comprehensive multimodal inputs including images, video, and audio.
// Images: JPEG, PNG, WebP
// Video: MP4, WebM, Matroska (MKV), QuickTime (MOV)
// Audio: MP3, WAV, WebM, M4A, Opus, AAC, FLAC
// Files can be provided via base64 data or URIs (GCS preferred for video/audio).
func (p *GeminiProvider) GetSupportedInputModes() []string {
	return []string{
		"text/plain",
		"application/json",
		// Images
		"image/jpeg",
		"image/png",
		"image/webp",
		// Video
		"video/mp4",
		"video/webm",
		"video/x-matroska", // MKV
		"video/quicktime",  // MOV
		// Audio
		"audio/mpeg", // MP3
		"audio/wav",
		"audio/webm",
		"audio/mp4", // M4A
		"audio/opus",
		"audio/aac",
		"audio/flac",
	}
}

func (p *GeminiProvider) Close() error {
	return nil
}

// getGenerateURL returns the URL for non-streaming generate requests
func (p *GeminiProvider) getGenerateURL() string {
	host := p.config.Host
	if host == "" {
		host = geminiDefaultHost
	}
	return fmt.Sprintf("%s/%s/models/%s:%s",
		strings.TrimSuffix(host, "/"),
		geminiAPIVersion,
		p.config.Model,
		geminiMethodGenerate)
}

// getStreamGenerateURL returns the URL for streaming generate requests
func (p *GeminiProvider) getStreamGenerateURL() string {
	host := p.config.Host
	if host == "" {
		host = geminiDefaultHost
	}
	return fmt.Sprintf("%s/%s/models/%s:%s?alt=%s",
		strings.TrimSuffix(host, "/"),
		geminiAPIVersion,
		p.config.Model,
		geminiMethodStreamGenerate,
		geminiStreamFormat)
}

// createAPIRequest creates an HTTP request for the Gemini API
func (p *GeminiProvider) createAPIRequest(ctx context.Context, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", geminiHeaderContentType)

	apiKey := strings.TrimSpace(p.config.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key is empty")
	}
	req.Header.Set(geminiHeaderAPIKey, apiKey)

	return req, nil
}

// parseErrorResponse parses an error response from the Gemini API
func (p *GeminiProvider) parseErrorResponse(resp *http.Response) error {
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("gemini API error (HTTP %d): failed to read body: %w", resp.StatusCode, readErr)
	}

	var errorResp GeminiResponse
	if json.Unmarshal(bodyBytes, &errorResp) == nil && errorResp.Error != nil {
		slog.Error("Gemini API request failed",
			"status_code", resp.StatusCode,
			"error_code", errorResp.Error.Code,
			"error_status", errorResp.Error.Status,
			"error_message", errorResp.Error.Message)
		return fmt.Errorf("gemini API error (HTTP %d): %s (code: %d, status: %s)",
			resp.StatusCode, errorResp.Error.Message, errorResp.Error.Code, errorResp.Error.Status)
	}

	slog.Error("Gemini API request failed",
		"status_code", resp.StatusCode,
		"response_body", string(bodyBytes))
	return fmt.Errorf("gemini API error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
}

// logRequestDebug logs debug information about a Gemini API request
func (p *GeminiProvider) logRequestDebug(req *GeminiRequest, reqBody []byte) {
	payloadPreview := string(reqBody)
	if len(payloadPreview) > geminiMaxPayloadPreviewLength {
		payloadPreview = payloadPreview[:geminiMaxPayloadPreviewLength] + "..."
	}

	hasThinking := req.GenerationConfig != nil && req.GenerationConfig.ThinkingConfig != nil
	thinkingBudget := 0
	if hasThinking && req.GenerationConfig.ThinkingConfig.ThinkingBudget != nil {
		thinkingBudget = *req.GenerationConfig.ThinkingConfig.ThinkingBudget
	}

	maxTokens := 0
	if req.GenerationConfig != nil {
		maxTokens = req.GenerationConfig.MaxOutputTokens
	}

	slog.Debug("Gemini API request",
		"model", p.config.Model,
		"content_count", len(req.Contents),
		"has_system", req.SystemInstruction != nil,
		"max_tokens", maxTokens,
		"thinking_enabled", hasThinking,
		"thinking_budget", thinkingBudget,
		"tools_count", len(req.Tools),
		"payload_preview", payloadPreview)
}

func (p *GeminiProvider) handleGeminiResponse(resp *http.Response, err error) ([]byte, *GeminiResponse, error) {
	// httpclient returns both response and error for non-2xx status codes
	// Check response body even when err != nil to extract API error details
	if resp != nil {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			if readErr == nil && len(body) > 0 {
				var errorResp GeminiResponse
				if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != nil {
					errMsg := fmt.Sprintf("Gemini API error (HTTP %d): %s (code: %d, status: %s)",
						resp.StatusCode, errorResp.Error.Message, errorResp.Error.Code, errorResp.Error.Status)
					slog.Error("Gemini API request failed", "status_code", resp.StatusCode, "error_message", errorResp.Error.Message, "error_code", errorResp.Error.Code, "error_status", errorResp.Error.Status)
					return nil, nil, fmt.Errorf("%s", errMsg)
				}
				errMsg := fmt.Sprintf("Gemini API error (HTTP %d): %s", resp.StatusCode, string(body))
				slog.Error("Gemini API request failed", "status_code", resp.StatusCode, "response_body", string(body))
				return nil, nil, fmt.Errorf("%s", errMsg)
			}
			errMsg := fmt.Sprintf("Gemini API error (HTTP %d): no response body", resp.StatusCode)
			slog.Error("Gemini API request failed", "status_code", resp.StatusCode, "error", "no response body")
			return nil, nil, fmt.Errorf("%s", errMsg)
		}

		if readErr != nil {
			return nil, nil, fmt.Errorf("failed to read response: %w", readErr)
		}

		var geminiResp GeminiResponse
		if err := json.Unmarshal(body, &geminiResp); err != nil {
			return body, nil, fmt.Errorf("failed to parse gemini response: %w", err)
		}

		if geminiResp.Error != nil {
			errMsg := fmt.Sprintf("Gemini API returned error (code: %d, status: %s): %s",
				geminiResp.Error.Code, geminiResp.Error.Status, geminiResp.Error.Message)
			slog.Error("Gemini API returned error", "error_code", geminiResp.Error.Code, "error_status", geminiResp.Error.Status, "error_message", geminiResp.Error.Message)
			return body, &geminiResp, fmt.Errorf("%s", errMsg)
		}

		return body, &geminiResp, nil
	}

	if err != nil {
		return nil, nil, fmt.Errorf("gemini API request failed: %w", err)
	}
	return nil, nil, fmt.Errorf("gemini API request failed: no response received")
}

func (p *GeminiProvider) buildRequest(messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) *GeminiRequest {
	contents, systemInstruction := p.convertMessages(messages)
	req := &GeminiRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
		GenerationConfig:  p.buildGenerationConfig(structConfig),
	}

	// Gemini only supports combining function calling with structured JSON output
	// on gemini-3-pro-preview model (preview feature). For all other models,
	// skip tools when structured JSON output is configured.
	// See: https://ai.google.dev/gemini-api/docs/structured-output#structured_outputs_with_tools
	canCombineToolsWithStructuredOutput := strings.Contains(strings.ToLower(p.config.Model), "gemini-3")
	useStructuredJSON := structConfig != nil && structConfig.Format == "json"

	if len(tools) > 0 && (!useStructuredJSON || canCombineToolsWithStructuredOutput) {
		req.Tools = []GeminiToolSet{
			{FunctionDeclarations: p.convertTools(tools)},
		}
	}

	return req
}

func (p *GeminiProvider) buildGenerationConfig(structConfig *StructuredOutputConfig) *GeminiGenerationConfig {
	config := &GeminiGenerationConfig{
		MaxOutputTokens: p.config.MaxTokens,
	}

	// Always include temperature if it's valid (>= 0 and <= 2)
	// Temperature 0 means deterministic output, which is a valid use case
	if p.config.Temperature != nil && *p.config.Temperature >= 0 && *p.config.Temperature <= 2 {
		config.Temperature = p.config.Temperature
	}

	// Add thinking config if enabled (same pattern as Anthropic/Ollama)
	// See: https://ai.google.dev/gemini-api/docs/thinking
	if p.config.Thinking != nil && p.config.Thinking.Enabled {
		config.ThinkingConfig = &GeminiThinkingConfig{
			IncludeThoughts: true,
		}
		if p.config.Thinking.BudgetTokens > 0 {
			budget := p.config.Thinking.BudgetTokens
			config.ThinkingConfig.ThinkingBudget = &budget
		}
	}

	if structConfig != nil {
		switch structConfig.Format {
		case "json":
			config.ResponseMimeType = geminiMimeTypeJSON
			if structConfig.Schema != nil {
				config.ResponseSchema = p.convertSchemaToGemini(structConfig.Schema, structConfig.PropertyOrdering)
			}
		case "enum":
			config.ResponseMimeType = geminiMimeTypeEnum
		}
	}

	return config
}

func (p *GeminiProvider) convertSchemaToGemini(schema interface{}, propertyOrdering []string) map[string]interface{} {
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return nil
	}

	cleaned := p.cleanSchemaForGemini(schemaMap)

	if len(propertyOrdering) > 0 {
		cleaned["propertyOrdering"] = propertyOrdering
	}

	return cleaned
}

func (p *GeminiProvider) cleanSchemaForGemini(schema map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range schema {

		if key == "additionalProperties" {
			continue
		}

		switch v := value.(type) {
		case map[string]interface{}:
			result[key] = p.cleanSchemaForGemini(v)
		case []interface{}:

			cleanedArray := make([]interface{}, len(v))
			for i, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					cleanedArray[i] = p.cleanSchemaForGemini(itemMap)
				} else {
					cleanedArray[i] = item
				}
			}
			result[key] = cleanedArray
		default:
			result[key] = value
		}
	}

	return result
}

func (p *GeminiProvider) convertMessages(messages []*pb.Message) ([]GeminiContent, *GeminiContent) {
	var contents []GeminiContent
	var systemParts []GeminiPart

	for _, msg := range messages {

		if msg.Role == pb.Role_ROLE_UNSPECIFIED {
			textContent := protocol.ExtractTextFromMessage(msg)
			if textContent != "" {
				systemParts = append(systemParts, GeminiPart{"text": textContent})
			}
			continue
		}

		var role string
		if msg.Role == pb.Role_ROLE_AGENT {
			role = geminiRoleModel
		} else {
			role = geminiRoleUser
		}

		var parts []GeminiPart

		// CRITICAL FIX: Extract and inject thinking from history for multi-turn conversations
		// Gemini requires thought signatures to be passed back for function calling continuity
		// See: https://ai.google.dev/gemini-api/docs/thought-signatures
		if msg.Role == pb.Role_ROLE_AGENT {
			thinkingContent := protocol.ExtractThinkingFromMessage(msg)
			if thinkingContent != "" && p.config.Thinking != nil && p.config.Thinking.Enabled {
				// Add thinking as first part with thought:true marker
				// This preserves reasoning context across multi-turn conversations
				parts = append(parts, GeminiPart{
					"text":    thinkingContent,
					"thought": true,
				})
			}
		}

		textContent := protocol.ExtractTextFromMessage(msg)
		if textContent != "" {
			parts = append(parts, GeminiPart{"text": textContent})
		}

		// Handle multi-modal parts
		for _, part := range msg.Parts {
			if file := part.GetFile(); file != nil {
				mediaType := file.GetMediaType()

				if uri := file.GetFileWithUri(); uri != "" {
					// Use fileData for URIs (assuming Google Cloud Storage URIs or similar supported by Gemini)
					// For URIs, media type must be provided (can't detect from bytes)
					if mediaType == "" {
						mediaType = geminiDefaultMediaType
					}

					// Validate media type for images
					if !strings.HasPrefix(mediaType, "image/") && !strings.HasPrefix(mediaType, "video/") && !strings.HasPrefix(mediaType, "audio/") {
						continue // Skip unsupported media types
					}

					parts = append(parts, GeminiPart{
						"file_data": map[string]interface{}{
							"mime_type": mediaType,
							"file_uri":  uri,
						},
					})
				} else if bytes := file.GetFileWithBytes(); len(bytes) > 0 {
					if mediaType == "" {
						mediaType = detectImageMediaType(bytes)
					}

					if !strings.HasPrefix(mediaType, "image/") && !strings.HasPrefix(mediaType, "video/") && !strings.HasPrefix(mediaType, "audio/") {
						continue
					}

					if len(bytes) > MaxGeminiImageSize {
						continue
					}

					base64Data := base64.StdEncoding.EncodeToString(bytes)
					parts = append(parts, GeminiPart{
						"inline_data": map[string]interface{}{
							"mime_type": mediaType,
							"data":      base64Data,
						},
					})
				}
			}
		}

		for _, tc := range protocol.GetToolCallsFromMessage(msg) {
			parts = append(parts, GeminiPart{
				"functionCall": map[string]interface{}{
					"name": tc.Name,
					"args": tc.Args,
				},
			})
		}

		toolResults := protocol.GetToolResultsFromMessage(msg)
		for _, toolResult := range toolResults {
			parts = append(parts, GeminiPart{
				"functionResponse": map[string]interface{}{
					"name": toolResult.ToolCallID,
					"response": map[string]interface{}{
						"content": toolResult.Content,
					},
				},
			})
		}

		if len(parts) > 0 {
			contents = append(contents, GeminiContent{
				Role:  string(role),
				Parts: parts,
			})
		}
	}

	var systemInstruction *GeminiContent
	if len(systemParts) > 0 {
		systemInstruction = &GeminiContent{
			Parts: systemParts,
		}
	}

	return contents, systemInstruction
}

func (p *GeminiProvider) convertTools(tools []ToolDefinition) []GeminiFunctionDeclaration {
	var funcs []GeminiFunctionDeclaration

	for _, tool := range tools {
		cleanedParams := p.cleanToolParametersForGemini(tool.Parameters)

		cleanedTool := GeminiFunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  cleanedParams,
		}
		funcs = append(funcs, cleanedTool)
	}

	return funcs
}

// cleanToolParametersForGemini cleans and validates tool parameter schemas for Gemini
// Gemini requires that all properties in 'required' arrays must exist in 'properties'
func (p *GeminiProvider) cleanToolParametersForGemini(params map[string]interface{}) map[string]interface{} {
	if params == nil {
		return nil
	}

	// Clean the schema first (removes additionalProperties, etc.)
	cleaned := p.cleanSchemaForGemini(params)

	// Validate required properties (ensures all required props exist)
	// This must happen after cleaning to catch any properties that were removed
	p.validateRequiredProperties(cleaned)

	return cleaned
}

// validateRequiredProperties ensures all properties in 'required' arrays exist in 'properties'
// This is critical for Gemini which strictly validates JSON schemas
func (p *GeminiProvider) validateRequiredProperties(schema map[string]interface{}) {
	if schema == nil {
		return
	}

	// Validate top-level required properties
	if required, ok := schema["required"].([]interface{}); ok {
		if properties, ok := schema["properties"].(map[string]interface{}); ok && len(properties) > 0 {
			validRequired := make([]interface{}, 0)
			for _, req := range required {
				if reqStr, ok := req.(string); ok {
					if _, exists := properties[reqStr]; exists {
						validRequired = append(validRequired, reqStr)
					}
				}
			}
			if len(validRequired) > 0 {
				schema["required"] = validRequired
			} else {
				delete(schema, "required")
			}
		} else {
			// No properties defined but required array exists - remove required
			delete(schema, "required")
		}
	}

	// Recursively validate nested schemas (e.g., items for arrays)
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for _, propValue := range properties {
			if propMap, ok := propValue.(map[string]interface{}); ok {
				// Check if this property has items (array type)
				if items, ok := propMap["items"].(map[string]interface{}); ok {
					p.validateRequiredProperties(items)
				}
				// Recursively validate nested objects
				if propType, _ := propMap["type"].(string); propType == "object" {
					p.validateRequiredProperties(propMap)
				}
			}
		}
	}

	// Also check items directly (for array schemas)
	if items, ok := schema["items"].(map[string]interface{}); ok {
		p.validateRequiredProperties(items)
	}
}

func (p *GeminiProvider) parseResponse(resp *GeminiResponse) (string, []*protocol.ToolCall, int, *ThinkingBlock, error) {
	if len(resp.Candidates) == 0 {
		return "", nil, 0, nil, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	var textParts []string
	var toolCalls []*protocol.ToolCall
	var thinkingParts []string
	var thoughtSignature string

	for _, part := range candidate.Content.Parts {
		// CRITICAL FIX: Extract thinking content (marked with thought: true) from non-streaming responses
		// Thinking must be preserved for multi-turn conversations and function calling continuity
		// See: https://ai.google.dev/gemini-api/docs/thought-signatures
		if text, ok := part["text"].(string); ok && text != "" {
			thought, hasThought := part["thought"].(bool)
			isThinking := hasThought && thought

			if isThinking {
				// Thinking content - extract for history preservation
				thinkingParts = append(thinkingParts, text)
			} else {
				// Regular text content (not thinking)
				textParts = append(textParts, text)
			}
		}

		// Extract thought signature if present (required for function calling)
		if sig, hasSig := part["signature"].(string); hasSig && sig != "" {
			thoughtSignature = sig
		}

		if fc, ok := part["functionCall"].(map[string]interface{}); ok {
			name, _ := fc["name"].(string)
			args, _ := fc["args"].(map[string]interface{})

			toolCalls = append(toolCalls, &protocol.ToolCall{
				ID:   fmt.Sprintf("call_%d", len(toolCalls)),
				Name: name,
				Args: args,
			})
		}
	}

	tokens := 0
	if resp.UsageMetadata != nil {
		tokens = resp.UsageMetadata.TotalTokenCount
	}

	finalText := strings.Join(textParts, "")

	// CRITICAL FIX: Return thinking block from non-streaming response
	var thinkingBlock *ThinkingBlock
	if len(thinkingParts) > 0 {
		thinkingContent := strings.Join(thinkingParts, "")
		thinkingBlock = &ThinkingBlock{
			Content:   thinkingContent,
			Signature: thoughtSignature, // Gemini thought signature
		}
		slog.Debug("Gemini non-streaming response contains thinking",
			"model", p.config.Model,
			"thinking_length", len(thinkingContent),
			"has_signature", thoughtSignature != "",
			"thinking_parts_count", len(thinkingParts))
	}

	return finalText, toolCalls, tokens, thinkingBlock, nil
}

// ThinkingState represents the state machine for Gemini thinking blocks
type ThinkingState int

const (
	ThinkingStateNone   ThinkingState = iota // No thinking block active
	ThinkingStateActive                      // Thinking block is active (internal reasoning)
	ThinkingStateClosed                      // Thinking block closed (transitioned to response/tools)
)

func (p *GeminiProvider) parseStreamingResponse(body io.Reader, chunks chan<- StreamChunk) {
	scanner := bufio.NewScanner(body)
	state := newGeminiStreamingState()

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var resp GeminiResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			continue
		}

		if resp.Error != nil {
			chunks <- StreamChunk{Type: "error", Error: fmt.Errorf("%s", resp.Error.Message)}
			return
		}

		if len(resp.Candidates) > 0 {
			candidate := resp.Candidates[0]
			p.processStreamingCandidate(candidate, state, chunks)
		}

		if resp.UsageMetadata != nil {
			state.totalTokens = resp.UsageMetadata.TotalTokenCount
		}
	}

	// Close any open thinking block at end of stream
	if state.hasActiveThinking() && state.accumulatedThinking.Len() > 0 {
		chunks <- StreamChunk{
			Type: "thinking_complete",
			Text: state.accumulatedThinking.String(),
		}
		state.closeThinkingBlock()
	}

	// Log metrics for thinking state transitions (if mismarked count > 0)
	if state.mismarkedThinkingCount > 0 {
		slog.Debug("Gemini thinking state machine completed",
			"model", p.config.Model,
			"mismarked_thinking_count", state.mismarkedThinkingCount,
			"final_state", state.thinkingState)
	}

	chunks <- StreamChunk{Type: "done", Tokens: state.totalTokens}
}

// processStreamingCandidate processes a single streaming candidate and updates state
func (p *GeminiProvider) processStreamingCandidate(candidate GeminiCandidate, state *geminiStreamingState, chunks chan<- StreamChunk) {
	for _, part := range candidate.Content.Parts {
		// Extract thought signatures for function calling continuity
		if signature, hasSig := part[geminiPartSignature].(string); hasSig && signature != "" {
			chunks <- StreamChunk{
				Type: "thinking_complete",
				Text: state.accumulatedThinking.String(),
				Metadata: map[string]interface{}{
					"thought_signature": signature,
				},
			}
		}

		// Handle text content (may be marked as thinking)
		if text, ok := part[geminiPartText].(string); ok && text != "" {
			thought, hasThought := part[geminiPartThought].(bool)
			isMarkedAsThinking := hasThought && thought

			// State machine logic
			shouldTreatAsThinking := isMarkedAsThinking && state.thinkingState != ThinkingStateClosed

			if shouldTreatAsThinking {
				if state.thinkingState == ThinkingStateNone {
					state.thinkingState = ThinkingStateActive
				}
				state.accumulatedThinking.WriteString(text)
				chunks <- StreamChunk{Type: "thinking", Text: text}
			} else {
				// Regular text content - transition from ACTIVE to CLOSED
				if state.hasActiveThinking() {
					chunks <- StreamChunk{
						Type: "thinking_complete",
						Text: state.accumulatedThinking.String(),
					}
					state.closeThinkingBlock()
				}

				// Log if Gemini incorrectly marked post-closure text as thinking
				if isMarkedAsThinking && state.thinkingState == ThinkingStateClosed {
					state.mismarkedThinkingCount++
					if state.mismarkedThinkingCount == 1 {
						slog.Warn("Gemini marked post-tool-call text as thinking, treating as regular text",
							"model", p.config.Model,
							"chunk_index", state.chunkCount)
					}
				}

				state.accumulatedText.WriteString(text)
				chunks <- StreamChunk{Type: "text", Text: text}
				state.chunkCount++
			}
		}

		// Handle tool calls
		if fc, ok := part[geminiPartFunctionCall].(map[string]interface{}); ok {
			// Transition from ACTIVE to CLOSED before emitting tool call
			if state.hasActiveThinking() {
				chunks <- StreamChunk{
					Type: "thinking_complete",
					Text: state.accumulatedThinking.String(),
				}
				state.closeThinkingBlock()
			}

			name, _ := fc["name"].(string)
			args, _ := fc["args"].(map[string]interface{})

			chunks <- StreamChunk{
				Type: "tool_call",
				ToolCall: &protocol.ToolCall{
					ID:   fmt.Sprintf("call_%d", time.Now().UnixNano()),
					Name: name,
					Args: args,
				},
			}
			state.chunkCount++
		}
	}
}
