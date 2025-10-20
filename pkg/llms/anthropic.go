// Package llms provides LLM provider implementations.
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

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// ============================================================================
// ANTHROPIC PROVIDER IMPLEMENTATION
// ============================================================================

// AnthropicProvider implements LLMProvider for Anthropic Claude API
type AnthropicProvider struct {
	config     *config.LLMProviderConfig
	httpClient *httpclient.Client
}

// AnthropicRequest represents the request payload for Anthropic API

// AnthropicTool represents a tool definition in Anthropic format
type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"` // JSON Schema
}

type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Stream      bool               `json:"stream"`
	System      string             `json:"system,omitempty"`
	Tools       []AnthropicTool    `json:"tools,omitempty"` // Tool use support
}

// AnthropicMessage represents a message in the conversation
type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // Can be string or []AnthropicContent
}

// AnthropicResponse represents the response from Anthropic API
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

// AnthropicStreamResponse represents streaming response chunks
type AnthropicStreamResponse struct {
	Type         string             `json:"type"`
	Index        int                `json:"index,omitempty"`
	Delta        *AnthropicDelta    `json:"delta,omitempty"`
	ContentBlock *AnthropicContent  `json:"content_block,omitempty"`
	Message      *AnthropicResponse `json:"message,omitempty"`
	Usage        *AnthropicUsage    `json:"usage,omitempty"`
}

// AnthropicContent represents content blocks in requests and responses
type AnthropicContent struct {
	Type      string                  `json:"type"`                  // "text", "tool_use", "tool_result"
	Text      string                  `json:"text,omitempty"`        // For text content
	ID        string                  `json:"id,omitempty"`          // Tool call ID (for tool_use)
	Name      string                  `json:"name,omitempty"`        // Tool name (for tool_use)
	Input     *map[string]interface{} `json:"input,omitempty"`       // Tool arguments (pointer ensures field presence as {} for tool_use)
	ToolUseID string                  `json:"tool_use_id,omitempty"` // Tool call ID reference (for tool_result)
	Content   string                  `json:"content,omitempty"`     // Tool result content (for tool_result)
}

// AnthropicDelta represents incremental content in streaming
type AnthropicDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"` // For tool use parameter streaming
}

// AnthropicUsage represents token usage information
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicError represents an API error
type AnthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey string, model string) *AnthropicProvider {
	config := &config.LLMProviderConfig{
		Type:        "anthropic",
		Model:       model,
		APIKey:      apiKey,
		Host:        "https://api.anthropic.com",
		Temperature: 1.0, // Claude default
		MaxTokens:   4096,
		Timeout:     120,
	}

	provider, _ := NewAnthropicProviderFromConfig(config)
	return provider
}

// NewAnthropicProviderFromConfig creates a new Anthropic provider from config
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

// GetModelName returns the model name
func (p *AnthropicProvider) GetModelName() string {
	return p.config.Model
}

// GetMaxTokens returns the maximum tokens
func (p *AnthropicProvider) GetMaxTokens() int {
	return p.config.MaxTokens
}

// GetTemperature returns the temperature
func (p *AnthropicProvider) GetTemperature() float64 {
	return p.config.Temperature
}

// Close closes the provider
func (p *AnthropicProvider) Close() error {
	return nil
}

// Generate generates a response given conversation messages
func (p *AnthropicProvider) Generate(messages []*pb.Message, tools []ToolDefinition) (string, []*protocol.ToolCall, int, error) {
	request := p.buildRequest(messages, false, tools)

	response, err := p.makeRequest(request)
	if err != nil {
		return "", nil, 0, err
	}

	if response.Error != nil {
		return "", nil, 0, fmt.Errorf("anthropic API error: %s", response.Error.Message)
	}

	tokensUsed := response.Usage.InputTokens + response.Usage.OutputTokens

	// Extract text and tool calls from content
	var text string
	var toolCalls []*protocol.ToolCall

	for _, content := range response.Content {
		if content.Type == "text" {
			text += content.Text
		} else if content.Type == "tool_use" {
			// Convert to ToolCall
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

	return text, toolCalls, tokensUsed, nil
}

// GenerateStreaming generates a streaming response given conversation messages
func (p *AnthropicProvider) GenerateStreaming(messages []*pb.Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
	request := p.buildRequest(messages, true, tools)

	outputCh := make(chan StreamChunk, 100)

	go func() {
		defer close(outputCh)

		if err := p.makeStreamingRequest(request, outputCh); err != nil {
			outputCh <- StreamChunk{
				Type:  "error",
				Error: err,
			}
		}
	}()

	return outputCh, nil
}

// buildRequest builds an Anthropic request with tool support
func (p *AnthropicProvider) buildRequest(messages []*pb.Message, stream bool, tools []ToolDefinition) AnthropicRequest {
	// Extract system prompt (Anthropic requires it in a separate field)
	var systemParts []string
	anthropicMessages := make([]AnthropicMessage, 0, len(messages))

	for _, msg := range messages {
		// Extract system messages for the system field
		if msg.Role == pb.Role_ROLE_UNSPECIFIED {
			// ROLE_UNSPECIFIED is used for system messages - add to system field
			textContent := protocol.ExtractTextFromMessage(msg)
			if textContent != "" {
				systemParts = append(systemParts, textContent)
			}
			continue
		}

		if msg.Role == pb.Role_ROLE_USER {
			// Regular user message
			textContent := protocol.ExtractTextFromMessage(msg)
			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role: "user",
				Content: []AnthropicContent{
					{Type: "text", Text: textContent},
				},
			})
			continue
		}

		// For tool results (check parts for tool result)
		toolResults := protocol.GetToolResultsFromMessage(msg)
		if len(toolResults) > 0 {
			// Convert each tool result to Anthropic format
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

		// Assistant messages with tool calls
		toolCalls := protocol.GetToolCallsFromMessage(msg)
		if msg.Role == pb.Role_ROLE_AGENT && len(toolCalls) > 0 {
			// Anthropic expects content to be an array with text + tool_use blocks
			contents := []AnthropicContent{}

			// Add text content if present
			textContent := protocol.ExtractTextFromMessage(msg)
			if textContent != "" {
				contents = append(contents, AnthropicContent{
					Type: "text",
					Text: textContent,
				})
			}

			// Add tool use blocks
			for _, tc := range toolCalls {
				// Ensure Input is never nil (Anthropic requires this field always present for tool_use)
				// Use pointer to distinguish between "omitted" (nil) and "empty object" (pointer to empty map)
				input := tc.Args
				if input == nil {
					input = make(map[string]interface{})
				}
				contents = append(contents, AnthropicContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: &input, // Pointer ensures field is always present as {} or {data}
				})
			}

			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role:    "assistant",
				Content: contents,
			})
		} else if msg.Role == pb.Role_ROLE_AGENT {
			// Regular assistant message without tool calls
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

	// Combine system parts into single system prompt
	var systemPrompt string
	if len(systemParts) > 0 {
		systemPrompt = strings.Join(systemParts, "\n\n")
	}

	request := AnthropicRequest{
		Model:       p.config.Model,
		Messages:    anthropicMessages,
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
		Stream:      stream,
		System:      systemPrompt,
	}

	// Add tools if provided (Anthropic calls them "tools")
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

// makeRequest makes a non-streaming request to Anthropic API using the generic HTTP client
func (p *AnthropicProvider) makeRequest(request AnthropicRequest) (*AnthropicResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", p.config.Host+"/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Enable request body reuse for retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Use generic HTTP client with smart retry
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

// makeStreamingRequest makes a streaming request to Anthropic API
func (p *AnthropicProvider) makeStreamingRequest(request AnthropicRequest, outputCh chan<- StreamChunk) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", p.config.Host+"/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Enable request body reuse for retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(jsonData)), nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Use the smart retry client for streaming initial request
	// The retry logic applies to establishing the connection, not streaming
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Track tool calls and their arguments during streaming
	// Tool arguments arrive as fragmented JSON strings that must be concatenated
	toolCalls := make(map[int]*protocol.ToolCall) // index -> tool call metadata
	toolJSONBuffers := make(map[int]string)       // index -> concatenated JSON fragments
	var totalTokens int

	// Read streaming response line by line (SSE format)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comment lines
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Extract JSON data after "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data: ")

		// Parse JSON response
		var streamResp AnthropicStreamResponse
		if err := json.Unmarshal([]byte(jsonData), &streamResp); err != nil {
			return fmt.Errorf("failed to decode streaming response: %w, data: %s", err, jsonData)
		}

		// Handle different event types
		switch streamResp.Type {
		case "content_block_start":
			// New content block started (text or tool_use)
			if streamResp.ContentBlock != nil && streamResp.ContentBlock.Type == "tool_use" {
				// Initialize tool call - arguments will be accumulated from JSON deltas
				toolCalls[streamResp.Index] = &protocol.ToolCall{
					ID:   streamResp.ContentBlock.ID,
					Name: streamResp.ContentBlock.Name,
					Args: make(map[string]interface{}),
				}
				toolJSONBuffers[streamResp.Index] = ""
			}

		case "content_block_delta":
			if streamResp.Delta != nil {
				// Text delta - stream to output as it arrives
				if streamResp.Delta.Text != "" {
					outputCh <- StreamChunk{Type: "text", Text: streamResp.Delta.Text}
				}

				// Tool parameter delta - accumulate JSON fragments
				// Anthropic streams tool arguments as partial JSON strings that must be
				// concatenated before parsing (e.g., "{\"lo", "cation\"", ": ", "\"Berlin\"}")
				if streamResp.Delta.Type == "input_json_delta" && streamResp.Delta.PartialJSON != "" {
					toolJSONBuffers[streamResp.Index] += streamResp.Delta.PartialJSON
				}
			}

		case "content_block_stop":
			// Content block finished - parse and send complete tool call
			if tc, exists := toolCalls[streamResp.Index]; exists {
				// Parse accumulated JSON fragments into tool arguments
				if jsonStr, hasJSON := toolJSONBuffers[streamResp.Index]; hasJSON && jsonStr != "" {
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(jsonStr), &args); err == nil {
						tc.Args = args
					}
				}

				// Send complete tool call to agent
				outputCh <- StreamChunk{
					Type:     "tool_call",
					ToolCall: tc,
				}
			}

		case "message_delta":
			// Usage information in delta
			if streamResp.Usage != nil {
				totalTokens = streamResp.Usage.OutputTokens
			}

		case "message_stop":
			// Stream finished - send done chunk with total tokens
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

// ============================================================================
// STRUCTURED OUTPUT METHODS
// ============================================================================

// GenerateStructured generates a response with structured output
func (p *AnthropicProvider) GenerateStructured(messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (string, []*protocol.ToolCall, int, error) {
	// TODO: Apply prefill if configured (Anthropic-specific optimization)
	// Currently unused but will be implemented when needed
	_ = structConfig

	// Build system prompt with schema instructions
	systemPrompt := p.buildSystemPromptWithSchema(structConfig)

	req := p.buildRequest(messages, false, tools)
	if systemPrompt != "" {
		if req.System != "" {
			req.System = req.System + "\n\n" + systemPrompt
		} else {
			req.System = systemPrompt
		}
	}

	// Make the actual API call using the existing Generate method
	return p.Generate(messages, tools)
}

// GenerateStructuredStreaming generates a streaming response with structured output
func (p *AnthropicProvider) GenerateStructuredStreaming(messages []*pb.Message, tools []ToolDefinition, structConfig *StructuredOutputConfig) (<-chan StreamChunk, error) {
	// Apply prefill if configured
	// Note: Prefill would require creating a pb.Message with ROLE_AGENT
	// For now, skip prefill support in migration.

	// Build system prompt with schema instructions
	systemPrompt := p.buildSystemPromptWithSchema(structConfig)

	req := p.buildRequest(messages, true, tools)
	if systemPrompt != "" {
		if req.System != "" {
			req.System = req.System + "\n\n" + systemPrompt
		} else {
			req.System = systemPrompt
		}
	}

	// Make the actual API call using the existing GenerateStreaming method
	return p.GenerateStreaming(messages, tools)
}

// SupportsStructuredOutput returns true (Anthropic supports structured output via prefill)
func (p *AnthropicProvider) SupportsStructuredOutput() bool {
	return true
}

// buildSystemPromptWithSchema builds system prompt with schema instructions
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
