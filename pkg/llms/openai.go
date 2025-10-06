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

	"github.com/kadirpekel/hector/pkg/httpclient"
	"github.com/kadirpekel/hector/pkg/config"
)

// ============================================================================
// OPENAI PROVIDER - CONSOLIDATED (Function Calling Only)
// ============================================================================

// OpenAIProvider implements LLMProvider for OpenAI API with native function calling
type OpenAIProvider struct {
	config     *config.LLMProviderConfig
	httpClient *httpclient.Client
}

// ============================================================================
// REQUEST/RESPONSE TYPES
// ============================================================================

// OpenAIRequest represents the request payload for OpenAI API
type OpenAIRequest struct {
	Model               string          `json:"model"`
	Messages            []OpenAIMessage `json:"messages"`
	MaxTokens           int             `json:"max_tokens,omitempty"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	Temperature         float64         `json:"temperature"`
	Stream              bool            `json:"stream"`
	Tools               []OpenAITool    `json:"tools,omitempty"`       // Function calling
	ToolChoice          string          `json:"tool_choice,omitempty"` // "auto", "required", "none"
}

// OpenAIResponse represents the response from OpenAI API
type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
	Error   *Error   `json:"error,omitempty"`
}

// OpenAIStreamResponse represents streaming response chunks
type OpenAIStreamResponse struct {
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"` // Token usage (may be included in final chunks)
	Error   *Error         `json:"error,omitempty"`
}

// OpenAIMessage represents a message in OpenAI's format
type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`   // Tool calls from assistant
	ToolCallID string           `json:"tool_call_id,omitempty"` // Tool result reference
}

// Choice represents a response choice
type Choice struct {
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// StreamChoice represents a streaming response choice
type StreamChoice struct {
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

// Delta represents incremental content in streaming (including tool calls)
type Delta struct {
	Content   string           `json:"content,omitempty"`
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Error represents an API error
type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// ============================================================================
// FUNCTION CALLING TYPES
// ============================================================================

// OpenAITool represents a tool definition in OpenAI format
type OpenAITool struct {
	Type     string             `json:"type"` // Always "function"
	Function OpenAIToolFunction `json:"function"`
}

// OpenAIToolFunction represents the function details
type OpenAIToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// OpenAIToolCall represents a tool call in the response
type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"` // Always "function"
	Function OpenAIFunctionCall `json:"function"`
}

// OpenAIFunctionCall represents the function being called
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ============================================================================
// CONSTRUCTORS
// ============================================================================

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string, model string) *OpenAIProvider {
	cfg := &config.LLMProviderConfig{
		Type:        "openai",
		Model:       model,
		APIKey:      apiKey,
		Host:        "https://api.openai.com/v1",
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     60,
	}

	provider, _ := NewOpenAIProviderFromConfig(cfg)
	return provider
}

// NewOpenAIProviderFromConfig creates a new OpenAI provider from config
func NewOpenAIProviderFromConfig(cfg *config.LLMProviderConfig) (*OpenAIProvider, error) {
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &OpenAIProvider{
		config: cfg,
		httpClient: httpclient.New(
			httpclient.WithHTTPClient(&http.Client{
				Timeout: time.Duration(cfg.Timeout) * time.Second,
			}),
			httpclient.WithMaxRetries(cfg.MaxRetries),
			httpclient.WithBaseDelay(time.Duration(cfg.RetryDelay)*time.Second),
			httpclient.WithHeaderParser(httpclient.ParseOpenAIRateLimitHeaders),
		),
	}, nil
}

// ============================================================================
// INTERFACE IMPLEMENTATION
// ============================================================================

// Generate generates a response with native function calling
func (p *OpenAIProvider) Generate(messages []Message, tools []ToolDefinition) (string, []ToolCall, int, error) {
	request := p.buildRequest(messages, false, tools)

	response, err := p.makeRequest(request)
	if err != nil {
		return "", nil, 0, err
	}

	if response.Error != nil {
		return "", nil, 0, fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", nil, 0, fmt.Errorf("no response choices returned")
	}

	choice := response.Choices[0]
	tokensUsed := response.Usage.TotalTokens

	// Extract text content (may be empty if only tool calls)
	text := choice.Message.Content

	// Check if model wants to call tools
	var toolCalls []ToolCall
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls, err = parseToolCalls(choice.Message.ToolCalls)
		if err != nil {
			return text, nil, tokensUsed, err
		}
	}

	// Return both text and tool calls (both can be present)
	return text, toolCalls, tokensUsed, nil
}

// GenerateStreaming generates a streaming response with function calling
func (p *OpenAIProvider) GenerateStreaming(messages []Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
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

// GetModelName returns the model name
func (p *OpenAIProvider) GetModelName() string {
	return p.config.Model
}

// GetMaxTokens returns the maximum tokens for generation
func (p *OpenAIProvider) GetMaxTokens() int {
	return p.config.MaxTokens
}

// GetTemperature returns the temperature setting
func (p *OpenAIProvider) GetTemperature() float64 {
	return p.config.Temperature
}

// Close closes the provider and releases resources
func (p *OpenAIProvider) Close() error {
	return nil
}

// ============================================================================
// INTERNAL HELPERS
// ============================================================================

// buildRequest builds an OpenAI request
func (p *OpenAIProvider) buildRequest(messages []Message, stream bool, tools []ToolDefinition) OpenAIRequest {
	// Convert universal Message to OpenAI-specific message format
	openaiMessages := make([]OpenAIMessage, len(messages))
	for i, msg := range messages {
		openaiMsg := OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// Handle tool calls (from assistant)
		if len(msg.ToolCalls) > 0 {
			openaiMsg.ToolCalls = make([]OpenAIToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				openaiMsg.ToolCalls[j] = OpenAIToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: OpenAIFunctionCall{
						Name:      tc.Name,
						Arguments: tc.RawArgs,
					},
				}
			}
		}

		// Handle tool results (role: "tool")
		if msg.ToolCallID != "" {
			openaiMsg.ToolCallID = msg.ToolCallID
		}

		openaiMessages[i] = openaiMsg
	}

	request := OpenAIRequest{
		Model:       p.config.Model,
		Messages:    openaiMessages,
		Temperature: p.config.Temperature,
		Stream:      stream,
	}

	// Set max tokens based on model (o1 models use max_completion_tokens)
	if strings.HasPrefix(p.config.Model, "o1-") || strings.HasPrefix(p.config.Model, "o3-") {
		request.MaxCompletionTokens = p.config.MaxTokens
	} else {
		request.MaxTokens = p.config.MaxTokens
	}

	// Add tools if provided
	if len(tools) > 0 {
		request.Tools = convertToOpenAITools(tools)
		request.ToolChoice = "auto" // Let model decide
	}

	return request
}

// convertToOpenAITools converts common ToolDefinition to OpenAI format
func convertToOpenAITools(tools []ToolDefinition) []OpenAITool {
	result := make([]OpenAITool, len(tools))
	for i, tool := range tools {
		result[i] = OpenAITool{
			Type: "function",
			Function: OpenAIToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}
	return result
}

// parseToolCalls extracts tool calls from OpenAI response
func parseToolCalls(openaiToolCalls []OpenAIToolCall) ([]ToolCall, error) {
	result := make([]ToolCall, len(openaiToolCalls))

	for i, tc := range openaiToolCalls {
		// Parse arguments JSON string into map
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
		}

		result[i] = ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
			RawArgs:   tc.Function.Arguments,
		}
	}

	return result, nil
}

// makeRequest makes a non-streaming request to OpenAI using the generic HTTP client
func (p *OpenAIProvider) makeRequest(request OpenAIRequest) (*OpenAIResponse, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.config.Host+"/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	// Use generic HTTP client with smart retry
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// makeStreamingRequest handles streaming responses with function calling
func (p *OpenAIProvider) makeStreamingRequest(request OpenAIRequest, outputCh chan<- StreamChunk) error {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.config.Host+"/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	// For streaming, we need the underlying HTTP client (not the retry wrapper)
	httpClient := &http.Client{Timeout: time.Duration(p.config.Timeout) * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	// Map to accumulate tool calls by index (OpenAI uses index for streaming)
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

		// Skip "data: " prefix
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		line = line[6:]

		// Check for stream end
		if bytes.Equal(line, []byte("[DONE]")) {
			break
		}

		var streamResp OpenAIStreamResponse
		if err := json.Unmarshal(line, &streamResp); err != nil {
			continue // Skip malformed chunks
		}

		if streamResp.Error != nil {
			return fmt.Errorf("API error: %s", streamResp.Error.Message)
		}

		// Extract token usage if available (some models include it)
		if streamResp.Usage != nil {
			totalTokens = streamResp.Usage.TotalTokens
		}

		if len(streamResp.Choices) == 0 {
			continue
		}

		choice := streamResp.Choices[0]

		// Handle text content
		if choice.Delta.Content != "" {
			outputCh <- StreamChunk{
				Type: "text",
				Text: choice.Delta.Content,
			}
		}

		// Handle tool calls (accumulated by index across chunks)
		for _, deltaCall := range choice.Delta.ToolCalls {
			// OpenAI uses index to identify which tool call this chunk belongs to
			// We need to get the index from the streaming tool call
			// For now, accumulate by merging into existing tool calls
			if deltaCall.ID != "" {
				// First chunk with full tool call structure
				toolCallsMap[len(toolCallsMap)] = &OpenAIToolCall{
					ID:       deltaCall.ID,
					Type:     deltaCall.Type,
					Function: deltaCall.Function,
				}
			} else {
				// Subsequent chunks with incremental arguments
				// Append to the last tool call
				if len(toolCallsMap) > 0 {
					lastIdx := len(toolCallsMap) - 1
					if toolCall, exists := toolCallsMap[lastIdx]; exists {
						toolCall.Function.Arguments += deltaCall.Function.Arguments
					}
				}
			}
		}

		// Check for completion
		if choice.FinishReason == "stop" || choice.FinishReason == "tool_calls" {
			// Convert map to slice for final processing
			var accumulatedToolCalls []OpenAIToolCall
			for i := 0; i < len(toolCallsMap); i++ {
				if toolCall, exists := toolCallsMap[i]; exists {
					accumulatedToolCalls = append(accumulatedToolCalls, *toolCall)
				}
			}

			// Send accumulated tool calls if any
			if len(accumulatedToolCalls) > 0 {
				toolCalls, err := parseToolCalls(accumulatedToolCalls)
				if err == nil {
					for _, tc := range toolCalls {
						outputCh <- StreamChunk{
							Type:     "tool_call",
							ToolCall: &tc,
						}
					}
				}
			}
			break
		}
	}

	// Send completion signal with token count
	outputCh <- StreamChunk{
		Type:   "done",
		Tokens: totalTokens,
	}

	return nil
}
