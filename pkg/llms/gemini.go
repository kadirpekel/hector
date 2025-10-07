package llms

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
)

// ============================================================================
// GEMINI PROVIDER IMPLEMENTATION
// Based on: https://ai.google.dev/gemini-api/docs/structured-output
// ============================================================================

// GeminiProvider implements LLMProvider for Google Gemini API
type GeminiProvider struct {
	config     *config.LLMProviderConfig
	httpClient *httpclient.Client
}

// ============================================================================
// REQUEST/RESPONSE TYPES
// ============================================================================

// GeminiRequest represents the request payload for Gemini API
type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	Tools            []GeminiToolSet         `json:"tools,omitempty"`
}

// GeminiGenerationConfig configures generation parameters
type GeminiGenerationConfig struct {
	Temperature      *float64               `json:"temperature,omitempty"`
	MaxOutputTokens  int                    `json:"maxOutputTokens,omitempty"`
	ResponseMimeType string                 `json:"responseMimeType,omitempty"` // "application/json" or "text/x.enum"
	ResponseSchema   map[string]interface{} `json:"responseSchema,omitempty"`   // JSON Schema
}

// GeminiContent represents content in a message
type GeminiContent struct {
	Role  string       `json:"role"` // "user" or "model"
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part of content (text or function call/result)
type GeminiPart map[string]interface{}

// GeminiToolSet represents a set of tools
type GeminiToolSet struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

// GeminiFunctionDeclaration represents a function that can be called
type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"` // JSON Schema
}

// GeminiResponse represents the response from Gemini API
type GeminiResponse struct {
	Candidates    []GeminiCandidate    `json:"candidates"`
	UsageMetadata *GeminiUsageMetadata `json:"usageMetadata,omitempty"`
	Error         *GeminiError         `json:"error,omitempty"`
}

// GeminiCandidate represents a candidate response
type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

// GeminiUsageMetadata represents token usage information
type GeminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// GeminiError represents an API error
type GeminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// ============================================================================
// PROVIDER IMPLEMENTATION
// ============================================================================

// NewGeminiProviderFromConfig creates a new Gemini provider from configuration
func NewGeminiProviderFromConfig(cfg *config.LLMProviderConfig) (*GeminiProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	httpClient := httpclient.New(
		httpclient.WithHTTPClient(&http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second}),
		httpclient.WithMaxRetries(cfg.MaxRetries),
		httpclient.WithBaseDelay(time.Duration(cfg.RetryDelay)*time.Second),
	)

	return &GeminiProvider{
		config:     cfg,
		httpClient: httpClient,
	}, nil
}

// Generate generates a response with function calling support
func (p *GeminiProvider) Generate(messages []Message, tools []ToolDefinition) (string, []ToolCall, int, error) {
	req := p.buildRequest(messages, tools, nil)

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s",
		p.config.Host, p.config.Model, p.config.APIKey)

	reqBody, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", nil, 0, fmt.Errorf("Gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, 0, fmt.Errorf("failed to read response: %w", err)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", nil, 0, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if geminiResp.Error != nil {
		return "", nil, 0, fmt.Errorf("Gemini API error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 {
		return "", nil, 0, fmt.Errorf("no candidates in response")
	}

	return p.parseResponse(&geminiResp)
}

// GenerateStreaming generates a streaming response
func (p *GeminiProvider) GenerateStreaming(messages []Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
	req := p.buildRequest(messages, tools, nil)

	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?key=%s&alt=sse",
		p.config.Host, p.config.Model, p.config.APIKey)

	chunks := make(chan StreamChunk, 10)

	go func() {
		defer close(chunks)

		reqBody, _ := json.Marshal(req)
		httpReq, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}
		defer resp.Body.Close()

		p.parseStreamingResponse(resp.Body, chunks)
	}()

	return chunks, nil
}

// GenerateStructured generates a response with structured output
func (p *GeminiProvider) GenerateStructured(messages []Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (string, []ToolCall, int, error) {
	req := p.buildRequest(messages, tools, structConfig)

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s",
		p.config.Host, p.config.Model, p.config.APIKey)

	reqBody, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", nil, 0, fmt.Errorf("Gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, 0, fmt.Errorf("failed to read response: %w", err)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return "", nil, 0, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if geminiResp.Error != nil {
		return "", nil, 0, fmt.Errorf("Gemini API error: %s", geminiResp.Error.Message)
	}

	return p.parseResponse(&geminiResp)
}

// GenerateStructuredStreaming generates a streaming response with structured output
func (p *GeminiProvider) GenerateStructuredStreaming(messages []Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (<-chan StreamChunk, error) {
	req := p.buildRequest(messages, tools, structConfig)

	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?key=%s&alt=sse",
		p.config.Host, p.config.Model, p.config.APIKey)

	chunks := make(chan StreamChunk, 10)

	go func() {
		defer close(chunks)

		reqBody, _ := json.Marshal(req)
		httpReq, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}
		defer resp.Body.Close()

		p.parseStreamingResponse(resp.Body, chunks)
	}()

	return chunks, nil
}

// SupportsStructuredOutput returns true (Gemini supports structured output)
func (p *GeminiProvider) SupportsStructuredOutput() bool {
	return true
}

// GetModelName returns the model name
func (p *GeminiProvider) GetModelName() string {
	return p.config.Model
}

// GetMaxTokens returns the maximum tokens for generation
func (p *GeminiProvider) GetMaxTokens() int {
	return p.config.MaxTokens
}

// GetTemperature returns the temperature setting
func (p *GeminiProvider) GetTemperature() float64 {
	return p.config.Temperature
}

// Close closes the provider and releases resources
func (p *GeminiProvider) Close() error {
	return nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// buildRequest builds a Gemini API request
func (p *GeminiProvider) buildRequest(messages []Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) *GeminiRequest {
	req := &GeminiRequest{
		Contents:         p.convertMessages(messages),
		GenerationConfig: p.buildGenerationConfig(structConfig),
	}

	if len(tools) > 0 {
		req.Tools = []GeminiToolSet{
			{FunctionDeclarations: p.convertTools(tools)},
		}
	}

	return req
}

// buildGenerationConfig builds generation configuration
func (p *GeminiProvider) buildGenerationConfig(structConfig *StructuredOutputConfig) *GeminiGenerationConfig {
	config := &GeminiGenerationConfig{
		MaxOutputTokens: p.config.MaxTokens,
	}

	// Only set temperature if not zero (Gemini uses default if omitted)
	if p.config.Temperature > 0 {
		temp := p.config.Temperature
		config.Temperature = &temp
	}

	// Add structured output configuration
	if structConfig != nil {
		switch structConfig.Format {
		case "json":
			config.ResponseMimeType = "application/json"
			if structConfig.Schema != nil {
				config.ResponseSchema = p.convertSchemaToGemini(structConfig.Schema, structConfig.PropertyOrdering)
			}
		case "enum":
			config.ResponseMimeType = "text/x.enum"
			// Enum handling would be done in schema
		}
	}

	return config
}

// convertSchemaToGemini converts schema to Gemini format with property ordering
func (p *GeminiProvider) convertSchemaToGemini(schema interface{}, propertyOrdering []string) map[string]interface{} {
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return nil
	}

	// Add propertyOrdering if provided (Gemini-specific optimization)
	if len(propertyOrdering) > 0 {
		schemaMap["propertyOrdering"] = propertyOrdering
	}

	return schemaMap
}

// convertMessages converts our Message format to Gemini format
func (p *GeminiProvider) convertMessages(messages []Message) []GeminiContent {
	var contents []GeminiContent

	for _, msg := range messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		if role == "system" {
			// Gemini doesn't have system role, convert to user message
			role = "user"
		}

		var parts []GeminiPart

		// Text content
		if msg.Content != "" {
			parts = append(parts, GeminiPart{"text": msg.Content})
		}

		// Tool calls (function calls)
		for _, tc := range msg.ToolCalls {
			parts = append(parts, GeminiPart{
				"functionCall": map[string]interface{}{
					"name": tc.Name,
					"args": tc.Arguments,
				},
			})
		}

		// Tool results
		if msg.Role == "tool" {
			parts = append(parts, GeminiPart{
				"functionResponse": map[string]interface{}{
					"name": msg.Name,
					"response": map[string]interface{}{
						"content": msg.Content,
					},
				},
			})
		}

		if len(parts) > 0 {
			contents = append(contents, GeminiContent{
				Role:  role,
				Parts: parts,
			})
		}
	}

	return contents
}

// convertTools converts our ToolDefinition format to Gemini format
func (p *GeminiProvider) convertTools(tools []ToolDefinition) []GeminiFunctionDeclaration {
	var funcs []GeminiFunctionDeclaration

	for _, tool := range tools {
		funcs = append(funcs, GeminiFunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		})
	}

	return funcs
}

// parseResponse parses a Gemini response and extracts text and tool calls
func (p *GeminiProvider) parseResponse(resp *GeminiResponse) (string, []ToolCall, int, error) {
	if len(resp.Candidates) == 0 {
		return "", nil, 0, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	var textParts []string
	var toolCalls []ToolCall

	for _, part := range candidate.Content.Parts {
		// Extract text
		if text, ok := part["text"].(string); ok {
			textParts = append(textParts, text)
		}

		// Extract function calls
		if fc, ok := part["functionCall"].(map[string]interface{}); ok {
			name, _ := fc["name"].(string)
			args, _ := fc["args"].(map[string]interface{})

			toolCalls = append(toolCalls, ToolCall{
				ID:        fmt.Sprintf("call_%d", len(toolCalls)),
				Name:      name,
				Arguments: args,
			})
		}
	}

	tokens := 0
	if resp.UsageMetadata != nil {
		tokens = resp.UsageMetadata.TotalTokenCount
	}

	return strings.Join(textParts, ""), toolCalls, tokens, nil
}

// parseStreamingResponse parses streaming response chunks
func (p *GeminiProvider) parseStreamingResponse(body io.Reader, chunks chan<- StreamChunk) {
	scanner := bufio.NewScanner(body)
	var accumulatedText strings.Builder
	totalTokens := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and non-data lines
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

			for _, part := range candidate.Content.Parts {
				// Stream text
				if text, ok := part["text"].(string); ok {
					accumulatedText.WriteString(text)
					chunks <- StreamChunk{Type: "text", Text: text}
				}

				// Stream function calls
				if fc, ok := part["functionCall"].(map[string]interface{}); ok {
					name, _ := fc["name"].(string)
					args, _ := fc["args"].(map[string]interface{})

					chunks <- StreamChunk{
						Type: "tool_call",
						ToolCall: &ToolCall{
							ID:        fmt.Sprintf("call_%d", time.Now().UnixNano()),
							Name:      name,
							Arguments: args,
						},
					}
				}
			}
		}

		if resp.UsageMetadata != nil {
			totalTokens = resp.UsageMetadata.TotalTokenCount
		}
	}

	// Send done chunk
	chunks <- StreamChunk{Type: "done", Tokens: totalTokens}
}
