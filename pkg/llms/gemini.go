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

func NewGeminiProviderFromConfig(cfg *config.LLMProviderConfig) (*GeminiProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}

	return &GeminiProvider{
		config:     cfg,
		httpClient: createHTTPClient(cfg),
	}, nil
}

func (p *GeminiProvider) Generate(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (string, []*protocol.ToolCall, int, error) {
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

	// Gemini API supports both query parameter and header for API key
	// Use header method (X-goog-api-key) as it's more standard and matches curl examples
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent",
		p.config.Host, p.config.Model)

	reqBody, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		reqErr := fmt.Errorf("failed to create request: %w", err)
		span.RecordError(reqErr)
		span.SetStatus(codes.Error, "failed to create request")

		duration := time.Since(startTime)
		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, reqErr)
		}

		return "", nil, 0, reqErr
	}
	httpReq.Header.Set("Content-Type", "application/json")

	apiKey := strings.TrimSpace(p.config.APIKey)
	if apiKey == "" {
		return "", nil, 0, fmt.Errorf("gemini API key is empty")
	}
	httpReq.Header.Set("X-goog-api-key", apiKey)

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

		return "", nil, 0, err
	}

	if len(geminiResp.Candidates) == 0 {
		noCandErr := fmt.Errorf("no candidates in response")
		span.RecordError(noCandErr)
		span.SetStatus(codes.Error, "no candidates")

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, noCandErr)
		}

		return "", nil, 0, noCandErr
	}

	text, toolCalls, tokens, parseErr := p.parseResponse(geminiResp)
	if parseErr != nil {
		span.RecordError(parseErr)
		span.SetStatus(codes.Error, "parse error")

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, parseErr)
		}

		return text, toolCalls, tokens, parseErr
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

	return text, toolCalls, tokens, nil
}

func (p *GeminiProvider) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
	req := p.buildRequest(messages, tools, nil)

	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse",
		p.config.Host, p.config.Model)

	chunks := make(chan StreamChunk, 10)

	go func() {
		defer close(chunks)

		reqBody, _ := json.Marshal(req)
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-goog-api-key", strings.TrimSpace(p.config.APIKey))

		resp, err := p.httpClient.Do(httpReq)
		// httpclient returns both response and error for non-2xx status codes
		// We need to check the response body even if there's an error
		if resp != nil {
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				var errorResp GeminiResponse
				if json.Unmarshal(bodyBytes, &errorResp) == nil && errorResp.Error != nil {
					err := fmt.Errorf("Gemini API error (HTTP %d): %s (code: %d, status: %s)",
						resp.StatusCode, errorResp.Error.Message, errorResp.Error.Code, errorResp.Error.Status)
					slog.Error("Gemini API request failed", "status_code", resp.StatusCode, "error_message", errorResp.Error.Message, "error_code", errorResp.Error.Code, "error_status", errorResp.Error.Status)
					chunks <- StreamChunk{Type: "error", Error: err}
					return
				} else {
					err := fmt.Errorf("Gemini API error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
					slog.Error("Gemini API request failed", "status_code", resp.StatusCode, "response_body", string(bodyBytes))
					chunks <- StreamChunk{Type: "error", Error: err}
					return
				}
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

func (p *GeminiProvider) GenerateStructured(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (string, []*protocol.ToolCall, int, error) {
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

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent",
		p.config.Host, p.config.Model)

	reqBody, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		reqErr := fmt.Errorf("failed to create request: %w", err)
		span.RecordError(reqErr)
		span.SetStatus(codes.Error, "failed to create request")

		duration := time.Since(startTime)
		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, reqErr)
		}

		return "", nil, 0, reqErr
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-goog-api-key", strings.TrimSpace(p.config.APIKey))

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

		return "", nil, 0, err
	}

	text, toolCalls, tokens, parseErr := p.parseResponse(geminiResp)
	if parseErr != nil {
		span.RecordError(parseErr)
		span.SetStatus(codes.Error, "parse error")

		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordLLMCall(ctx, p.config.Model, duration, 0, 0, parseErr)
		}

		return text, toolCalls, tokens, parseErr
	}

	// Record successful metrics
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

	return text, toolCalls, tokens, nil
}

func (p *GeminiProvider) GenerateStructuredStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (<-chan StreamChunk, error) {
	req := p.buildRequest(messages, tools, structConfig)

	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse",
		p.config.Host, p.config.Model)

	chunks := make(chan StreamChunk, 10)

	go func() {
		defer close(chunks)

		reqBody, _ := json.Marshal(req)
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-goog-api-key", strings.TrimSpace(p.config.APIKey))

		resp, err := p.httpClient.Do(httpReq)
		// httpclient returns both response and error for non-2xx status codes
		// We need to check the response body even if there's an error
		if resp != nil {
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				var errorResp GeminiResponse
				if json.Unmarshal(bodyBytes, &errorResp) == nil && errorResp.Error != nil {
					err := fmt.Errorf("Gemini API error (HTTP %d): %s (code: %d, status: %s)",
						resp.StatusCode, errorResp.Error.Message, errorResp.Error.Code, errorResp.Error.Status)
					slog.Error("Gemini API request failed", "status_code", resp.StatusCode, "error_message", errorResp.Error.Message, "error_code", errorResp.Error.Code, "error_status", errorResp.Error.Status)
					chunks <- StreamChunk{Type: "error", Error: err}
					return
				} else {
					err := fmt.Errorf("Gemini API error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
					slog.Error("Gemini API request failed", "status_code", resp.StatusCode, "response_body", string(bodyBytes))
					chunks <- StreamChunk{Type: "error", Error: err}
					return
				}
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
		return 0.7 // Default
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
			return body, nil, fmt.Errorf("failed to parse Gemini response: %w", err)
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
		return nil, nil, fmt.Errorf("Gemini API request failed: %w", err)
	}
	return nil, nil, fmt.Errorf("Gemini API request failed: no response received")
}

func (p *GeminiProvider) buildRequest(messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) *GeminiRequest {
	contents, systemInstruction := p.convertMessages(messages)
	req := &GeminiRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
		GenerationConfig:  p.buildGenerationConfig(structConfig),
	}

	if len(tools) > 0 {
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
			config.ResponseMimeType = "application/json"
			if structConfig.Schema != nil {
				config.ResponseSchema = p.convertSchemaToGemini(structConfig.Schema, structConfig.PropertyOrdering)
			}
		case "enum":
			config.ResponseMimeType = "text/x.enum"

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
			role = "model"
		} else {

			role = "user"
		}

		var parts []GeminiPart

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
						mediaType = "image/jpeg"
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

					const maxInlineDataSize = 20 * 1024 * 1024
					if len(bytes) > maxInlineDataSize {
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

func (p *GeminiProvider) parseResponse(resp *GeminiResponse) (string, []*protocol.ToolCall, int, error) {
	if len(resp.Candidates) == 0 {
		return "", nil, 0, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	var textParts []string
	var toolCalls []*protocol.ToolCall

	for _, part := range candidate.Content.Parts {

		if text, ok := part["text"].(string); ok {
			textParts = append(textParts, text)
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

	return finalText, toolCalls, tokens, nil
}

func (p *GeminiProvider) parseStreamingResponse(body io.Reader, chunks chan<- StreamChunk) {
	scanner := bufio.NewScanner(body)
	var accumulatedText strings.Builder
	var accumulatedThinking strings.Builder
	var inThinkingBlock bool
	// Track if the thinking block has been closed in this stream.
	// Semantically, thinking represents internal reasoning before taking action.
	// Once closed (by tool call or regular text), any subsequent text marked as thinking
	// should be treated as regular response text, as Gemini sometimes incorrectly marks
	// post-tool-call responses as thinking.
	var thinkingBlockClosed bool
	totalTokens := 0
	chunkCount := 0

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

			// Process parts in order, maintaining correct thinking block state
			for _, part := range candidate.Content.Parts {
				// Handle text content (may be marked as thinking)
				if text, ok := part["text"].(string); ok && text != "" {
					// Gemini marks thinking parts with thought: true boolean
					// See: https://ai.google.dev/gemini-api/docs/thinking
					thought, hasThought := part["thought"].(bool)
					isMarkedAsThinking := hasThought && thought

					// Only treat as thinking if:
					// 1. It's marked as thinking by Gemini, AND
					// 2. We haven't closed the thinking block yet
					// Once closed, any text (even if marked as thinking) is regular response text
					shouldTreatAsThinking := isMarkedAsThinking && !thinkingBlockClosed

					if shouldTreatAsThinking {
						// Valid thinking block (internal reasoning before actions)
						if !inThinkingBlock {
							inThinkingBlock = true
						}
						accumulatedThinking.WriteString(text)
						chunks <- StreamChunk{Type: "thinking", Text: text}
					} else {
						// Regular text content - close any open thinking block first
						if inThinkingBlock {
							chunks <- StreamChunk{
								Type: "thinking_complete",
								Text: accumulatedThinking.String(),
							}
							accumulatedThinking.Reset()
							inThinkingBlock = false
							thinkingBlockClosed = true
						}
						accumulatedText.WriteString(text)
						chunks <- StreamChunk{Type: "text", Text: text}
						chunkCount++
					}
				}

				// Handle tool calls
				if fc, ok := part["functionCall"].(map[string]interface{}); ok {
					// Close any open thinking block before emitting tool call
					if inThinkingBlock {
						chunks <- StreamChunk{
							Type: "thinking_complete",
							Text: accumulatedThinking.String(),
						}
						accumulatedThinking.Reset()
						inThinkingBlock = false
						thinkingBlockClosed = true
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
					chunkCount++
				}
			}
		}

		if resp.UsageMetadata != nil {
			totalTokens = resp.UsageMetadata.TotalTokenCount
		}
	}

	// Close any open thinking block at end of stream
	if inThinkingBlock && accumulatedThinking.Len() > 0 {
		chunks <- StreamChunk{
			Type: "thinking_complete",
			Text: accumulatedThinking.String(),
		}
	}

	chunks <- StreamChunk{Type: "done", Tokens: totalTokens}
}
