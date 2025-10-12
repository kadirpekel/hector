// Package llms provides LLM provider implementations.
package llms

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
	"github.com/kadirpekel/hector/pkg/protocol"
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
	Contents          []GeminiContent         `json:"contents"`
	SystemInstruction *GeminiContent          `json:"systemInstruction,omitempty"` // System instructions (Gemini 1.5+)
	GenerationConfig  *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	Tools             []GeminiToolSet         `json:"tools,omitempty"`
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
		return nil, fmt.Errorf("gemini API key is required")
	}

	return &GeminiProvider{
		config:     cfg,
		httpClient: createHTTPClient(cfg),
	}, nil
}

// Generate generates a response with function calling support
func (p *GeminiProvider) Generate(messages []*pb.Message, tools []ToolDefinition) (string, []*protocol.ToolCall, int, error) {

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
		return "", nil, 0, fmt.Errorf("gemini API request failed: %w", err)
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
		return "", nil, 0, fmt.Errorf("gemini API error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 {
		return "", nil, 0, fmt.Errorf("no candidates in response")
	}

	return p.parseResponse(&geminiResp)
}

// GenerateStreaming generates a streaming response
func (p *GeminiProvider) GenerateStreaming(messages []*pb.Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
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

		// Use p.httpClient which has retry logic and backoff configured
		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}
		defer resp.Body.Close()

		// Check for HTTP errors (rate limits, auth failures, etc.)
		if resp.StatusCode != http.StatusOK {
			// Read error response body
			bodyBytes, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("gemini API error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
			log.Printf("[GEMINI ERROR] %v\n", err)
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}

		p.parseStreamingResponse(resp.Body, chunks)
	}()

	return chunks, nil
}

// GenerateStructured generates a response with structured output
func (p *GeminiProvider) GenerateStructured(messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (string, []*protocol.ToolCall, int, error) {
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
		return "", nil, 0, fmt.Errorf("gemini API request failed: %w", err)
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
		return "", nil, 0, fmt.Errorf("gemini API error: %s", geminiResp.Error.Message)
	}

	return p.parseResponse(&geminiResp)
}

// GenerateStructuredStreaming generates a streaming response with structured output
func (p *GeminiProvider) GenerateStructuredStreaming(messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (<-chan StreamChunk, error) {
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

		// Use p.httpClient which has retry logic and backoff configured
		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}
		defer resp.Body.Close()

		// Check for HTTP errors (rate limits, auth failures, etc.)
		if resp.StatusCode != http.StatusOK {
			// Read error response body
			bodyBytes, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("gemini API error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
			log.Printf("[GEMINI ERROR] %v\n", err)
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}

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
// Returns (contents, systemInstruction)
func (p *GeminiProvider) convertMessages(messages []*pb.Message) ([]GeminiContent, *GeminiContent) {
	var contents []GeminiContent
	var systemParts []GeminiPart

	for _, msg := range messages {
		// Extract system messages for systemInstruction field
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
			// ROLE_USER
			role = "user"
		}

		var parts []GeminiPart

		// Text content
		textContent := protocol.ExtractTextFromMessage(msg)
		if textContent != "" {
			parts = append(parts, GeminiPart{"text": textContent})
		}

		// Tool calls (function calls)
		for _, tc := range protocol.GetToolCallsFromMessage(msg) {
			parts = append(parts, GeminiPart{
				"functionCall": map[string]interface{}{
					"name": tc.Name,
					"args": tc.Args,
				},
			})
		}

		// Tool results (check parts for tool results)
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

	// Create system instruction if we have system parts
	var systemInstruction *GeminiContent
	if len(systemParts) > 0 {
		systemInstruction = &GeminiContent{
			Parts: systemParts,
		}
	}

	return contents, systemInstruction
}

// convertTools converts our ToolDefinition format to Gemini format
func (p *GeminiProvider) convertTools(tools []ToolDefinition) []GeminiFunctionDeclaration {
	var funcs []GeminiFunctionDeclaration

	for _, tool := range tools {
		funcs = append(funcs, (GeminiFunctionDeclaration)(tool))
	}

	return funcs
}

// parseResponse parses a Gemini response and extracts text and tool calls
func (p *GeminiProvider) parseResponse(resp *GeminiResponse) (string, []*protocol.ToolCall, int, error) {
	if len(resp.Candidates) == 0 {
		return "", nil, 0, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	var textParts []string
	var toolCalls []*protocol.ToolCall

	for _, part := range candidate.Content.Parts {
		// Extract text
		if text, ok := part["text"].(string); ok {
			textParts = append(textParts, text)
		}

		// Extract function calls
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

// parseStreamingResponse parses streaming response chunks
func (p *GeminiProvider) parseStreamingResponse(body io.Reader, chunks chan<- StreamChunk) {
	scanner := bufio.NewScanner(body)
	var accumulatedText strings.Builder
	totalTokens := 0
	lineCount := 0
	chunkCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

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
					chunkCount++
				}

				// Stream function calls
				if fc, ok := part["functionCall"].(map[string]interface{}); ok {
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

	// Send done chunk
	chunks <- StreamChunk{Type: "done", Tokens: totalTokens}
}
