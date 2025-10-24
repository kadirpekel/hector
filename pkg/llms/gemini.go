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
}

type GeminiContent struct {
	Role  string       `json:"role"`
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
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}

	return &GeminiProvider{
		config:     cfg,
		httpClient: createHTTPClient(cfg),
	}, nil
}

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
	_, geminiResp, err := p.handleGeminiResponse(resp, err)
	if err != nil {
		return "", nil, 0, err
	}

	if len(geminiResp.Candidates) == 0 {
		return "", nil, 0, fmt.Errorf("no candidates in response")
	}

	return p.parseResponse(geminiResp)
}

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

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {

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
	_, geminiResp, err := p.handleGeminiResponse(resp, err)
	if err != nil {
		return "", nil, 0, err
	}

	return p.parseResponse(geminiResp)
}

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

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			chunks <- StreamChunk{Type: "error", Error: err}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {

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
	return p.config.Temperature
}

func (p *GeminiProvider) Close() error {
	return nil
}

func (p *GeminiProvider) handleGeminiResponse(resp *http.Response, err error) ([]byte, *GeminiResponse, error) {
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		if resp != nil && resp.Body != nil {
			body, readErr := io.ReadAll(resp.Body)
			if readErr == nil && len(body) > 0 {
				var errorResp GeminiResponse
				if json.Unmarshal(body, &errorResp) == nil && errorResp.Error != nil {
					return nil, nil, fmt.Errorf("Gemini API error: %s (code: %d)",
						errorResp.Error.Message, errorResp.Error.Code)
				}
				return nil, nil, fmt.Errorf("gemini API request failed: %w - Response: %s", err, string(body))
			}
		}
		return nil, nil, fmt.Errorf("gemini API request failed: %w", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return respBody, nil, fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	if geminiResp.Error != nil {
		return respBody, &geminiResp, fmt.Errorf("Gemini API error: %s (code: %d, status: %s)",
			geminiResp.Error.Message, geminiResp.Error.Code, geminiResp.Error.Status)
	}

	return respBody, &geminiResp, nil
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

	if p.config.Temperature > 0 {
		temp := p.config.Temperature
		config.Temperature = &temp
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
		funcs = append(funcs, (GeminiFunctionDeclaration)(tool))
	}

	return funcs
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
	totalTokens := 0
	lineCount := 0
	chunkCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

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

				if text, ok := part["text"].(string); ok {
					accumulatedText.WriteString(text)
					chunks <- StreamChunk{Type: "text", Text: text}
					chunkCount++
				}

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

	chunks <- StreamChunk{Type: "done", Tokens: totalTokens}
}
