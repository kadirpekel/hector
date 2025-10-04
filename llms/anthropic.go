package llms

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// ANTHROPIC PROVIDER IMPLEMENTATION
// ============================================================================

// AnthropicProvider implements LLMProvider for Anthropic Claude API
type AnthropicProvider struct {
	config *config.LLMProviderConfig
	client *http.Client
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

// AnthropicContent represents content blocks in response
type AnthropicContent struct {
	Type      string                 `json:"type"`                  // "text", "tool_use", "tool_result"
	Text      string                 `json:"text,omitempty"`        // For text
	ID        string                 `json:"id,omitempty"`          // For tool_use
	Name      string                 `json:"name,omitempty"`        // For tool_use
	Input     map[string]interface{} `json:"input,omitempty"`       // For tool_use
	ToolUseID string                 `json:"tool_use_id,omitempty"` // For tool_result
	Content   string                 `json:"content,omitempty"`     // For tool_result
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

	return &AnthropicProvider{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
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
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
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
func (p *AnthropicProvider) Generate(messages []Message, tools []ToolDefinition) (string, []ToolCall, int, error) {
	request := p.buildRequest(messages, false, tools)

	response, err := p.makeRequest(request)
	if err != nil {
		return "", nil, 0, err
	}

	if response.Error != nil {
		return "", nil, 0, fmt.Errorf("Anthropic API error: %s", response.Error.Message)
	}

	tokensUsed := response.Usage.InputTokens + response.Usage.OutputTokens

	// Extract text and tool calls from content
	var text string
	var toolCalls []ToolCall

	for _, content := range response.Content {
		if content.Type == "text" {
			text += content.Text
		} else if content.Type == "tool_use" {
			// Convert to ToolCall
			rawArgs, _ := json.Marshal(content.Input)
			toolCalls = append(toolCalls, ToolCall{
				ID:        content.ID,
				Name:      content.Name,
				Arguments: content.Input,
				RawArgs:   string(rawArgs),
			})
		}
	}

	return text, toolCalls, tokensUsed, nil
}

// GenerateStreaming generates a streaming response given conversation messages
func (p *AnthropicProvider) GenerateStreaming(messages []Message, tools []ToolDefinition) (<-chan StreamChunk, error) {
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
func (p *AnthropicProvider) buildRequest(messages []Message, stream bool, tools []ToolDefinition) AnthropicRequest {
	// Extract system prompt (Anthropic requires it in a separate field)
	var systemPrompt string
	anthropicMessages := make([]AnthropicMessage, 0, len(messages))

	for _, msg := range messages {
		// Extract system messages for the system field
		if msg.Role == "system" {
			if systemPrompt != "" {
				systemPrompt += "\n\n" // Concatenate multiple system messages
			}
			systemPrompt += msg.Content
			continue
		}

		// For tool results, Anthropic expects them as user messages with specific format
		if msg.Role == "tool" {
			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role: "user",
				Content: []AnthropicContent{
					{
						Type:      "tool_result",
						ToolUseID: msg.ToolCallID,
						Content:   msg.Content,
					},
				},
			})
		} else if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			// Assistant message with tool calls
			// Anthropic expects content to be an array with text + tool_use blocks
			contents := []AnthropicContent{}

			// Add text content if present
			if msg.Content != "" {
				contents = append(contents, AnthropicContent{
					Type: "text",
					Text: msg.Content,
				})
			}

			// Add tool use blocks
			for _, toolCall := range msg.ToolCalls {
				contents = append(contents, AnthropicContent{
					Type:  "tool_use",
					ID:    toolCall.ID,
					Name:  toolCall.Name,
					Input: toolCall.Arguments,
				})
			}

			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role:    "assistant",
				Content: contents,
			})
		} else {
			// Regular user/assistant messages
			anthropicMessages = append(anthropicMessages, AnthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
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

// RetryStrategy represents the retry approach for different error types
type RetryStrategy int

const (
	NoRetry           RetryStrategy = iota
	ConservativeRetry               // Quick retry for server errors (max 2 attempts)
	SmartRetry                      // Header-driven retry for rate limits
)

// getRetryStrategy determines the retry strategy based on HTTP status code
func getRetryStrategy(statusCode int) RetryStrategy {
	switch statusCode {
	case http.StatusTooManyRequests, // 429 - Rate limit with headers
		http.StatusServiceUnavailable: // 503 - May have Retry-After
		return SmartRetry
	case http.StatusRequestTimeout, // 408 - Network timeout
		http.StatusInternalServerError, // 500 - Server error
		http.StatusBadGateway,          // 502 - Gateway issue
		http.StatusGatewayTimeout:      // 504 - Gateway timeout
		return ConservativeRetry
	default:
		return NoRetry
	}
}

// RateLimitInfo contains rate limit information from response headers
type RateLimitInfo struct {
	RetryAfter            time.Duration
	ResetTime             int64
	RequestsRemaining     int
	InputTokensRemaining  int
	OutputTokensRemaining int
	TokensRemaining       int // For OpenAI (combined tokens)
}

// makeRequest makes a non-streaming request to Anthropic API with smart retry logic
func (p *AnthropicProvider) makeRequest(request AnthropicRequest) (*AnthropicResponse, error) {
	maxRetries := p.config.MaxRetries
	baseDelay := time.Duration(p.config.RetryDelay) * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		response, strategy, err, retryInfo := p.attemptRequestWithHeaders(request)

		// No retry - fail immediately
		if strategy == NoRetry {
			return response, err
		}

		// Success or error after exhausting retries
		if err == nil || attempt >= maxRetries {
			return response, err
		}

		// Determine retry delay based on strategy
		var delay time.Duration
		var maxAttempts int

		switch strategy {
		case SmartRetry:
			// Header-driven retry for rate limits (use full retry budget)
			maxAttempts = maxRetries

			if retryInfo.RetryAfter > 0 {
				delay = retryInfo.RetryAfter
				fmt.Printf("⏳ Rate limited. Provider suggests retry in %v (attempt %d/%d)\n",
					delay, attempt+1, maxAttempts)
			} else if retryInfo.ResetTime > 0 {
				delay = time.Until(time.Unix(retryInfo.ResetTime, 0))
				if delay < 0 {
					delay = baseDelay
				}
				fmt.Printf("⏳ Rate limited. Retry at reset time in %v (attempt %d/%d)\n",
					delay, attempt+1, maxAttempts)
			} else {
				// Fallback to exponential backoff with jitter
				exponentialDelay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
				jitter := time.Duration(float64(exponentialDelay) * 0.1)
				delay = exponentialDelay + jitter
				fmt.Printf("⏳ Rate limited. Retrying in %v (attempt %d/%d)\n",
					delay, attempt+1, maxAttempts)
			}

		case ConservativeRetry:
			// Conservative retry for server errors (max 2 attempts, short delays)
			maxAttempts = 2
			if attempt >= maxAttempts {
				return response, err
			}

			delay = time.Duration(2+attempt) * time.Second // 2s, 3s
			fmt.Printf("⚠️  Server error. Quick retry in %v (attempt %d/%d)\n",
				delay, attempt+1, maxAttempts)
		}

		time.Sleep(delay)
	}

	return nil, fmt.Errorf("max retries exceeded after %d attempts", maxRetries)
}

// attemptRequestWithHeaders makes a request and extracts retry-related headers
func (p *AnthropicProvider) attemptRequestWithHeaders(request AnthropicRequest) (*AnthropicResponse, RetryStrategy, error, RateLimitInfo) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, NoRetry, fmt.Errorf("failed to marshal request: %w", err), RateLimitInfo{}
	}

	req, err := http.NewRequest("POST", p.config.Host+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, NoRetry, fmt.Errorf("failed to create request: %w", err), RateLimitInfo{}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NoRetry, fmt.Errorf("failed to make request: %w", err), RateLimitInfo{}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Extract rate limit headers
	retryInfo := extractAnthropicRateLimitHeaders(resp.Header)

	// Determine retry strategy
	strategy := getRetryStrategy(resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, strategy, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body)), retryInfo
	}

	var response AnthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, NoRetry, fmt.Errorf("failed to decode response: %w", err), RateLimitInfo{}
	}

	return &response, NoRetry, nil, retryInfo
}

// extractAnthropicRateLimitHeaders extracts Anthropic rate limit information
func extractAnthropicRateLimitHeaders(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	// Retry-After (seconds)
	if retryAfter := headers.Get("retry-after"); retryAfter != "" {
		if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
			info.RetryAfter = seconds
		}
	}

	// Parse reset time (RFC 3339 format)
	if resetStr := headers.Get("anthropic-ratelimit-requests-reset"); resetStr != "" {
		if resetTime, err := time.Parse(time.RFC3339, resetStr); err == nil {
			info.ResetTime = resetTime.Unix()
		}
	}

	// Parse remaining counts
	if remaining := headers.Get("anthropic-ratelimit-requests-remaining"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &info.RequestsRemaining)
	}
	if remaining := headers.Get("anthropic-ratelimit-input-tokens-remaining"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &info.InputTokensRemaining)
	}
	if remaining := headers.Get("anthropic-ratelimit-output-tokens-remaining"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &info.OutputTokensRemaining)
	}

	return info
}

// attemptRequest makes a single HTTP request attempt
func (p *AnthropicProvider) attemptRequest(request AnthropicRequest) (*AnthropicResponse, bool, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.config.Host+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Determine if error is retryable
	shouldRetry := isRetryableError(resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, shouldRetry, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response AnthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, false, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, false, nil
}

// isRetryableError determines if an HTTP status code is retryable
func isRetryableError(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, // 408
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// makeStreamingRequest makes a streaming request to Anthropic API
func (p *AnthropicProvider) makeStreamingRequest(request AnthropicRequest, outputCh chan<- StreamChunk) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.config.Host+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	// Enable fine-grained tool streaming for better performance
	if len(request.Tools) > 0 {
		req.Header.Set("anthropic-beta", "fine-grained-tool-streaming-2025-05-14")
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Track tool calls being accumulated across deltas
	toolCalls := make(map[int]*ToolCall) // index -> accumulated ToolCall
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
			// New content block started (could be text or tool_use)
			if streamResp.ContentBlock != nil && streamResp.ContentBlock.Type == "tool_use" {
				// Initialize tool call accumulator
				toolCalls[streamResp.Index] = &ToolCall{
					ID:        streamResp.ContentBlock.ID,
					Name:      streamResp.ContentBlock.Name,
					Arguments: make(map[string]interface{}),
					RawArgs:   "",
				}
			}

		case "content_block_delta":
			if streamResp.Delta != nil {
				// Text delta
				if streamResp.Delta.Text != "" {
					outputCh <- StreamChunk{Type: "text", Text: streamResp.Delta.Text}
				}

				// Tool use parameter delta (partial JSON)
				if streamResp.Delta.PartialJSON != "" {
					if tc, exists := toolCalls[streamResp.Index]; exists {
						// Accumulate partial JSON
						tc.RawArgs += streamResp.Delta.PartialJSON
					}
				}
			}

		case "content_block_stop":
			// Content block finished
			if tc, exists := toolCalls[streamResp.Index]; exists {
				// Parse accumulated JSON for tool arguments
				if tc.RawArgs != "" {
					if err := json.Unmarshal([]byte(tc.RawArgs), &tc.Arguments); err != nil {
						// If JSON is invalid, continue anyway (partial streaming may result in invalid JSON)
						tc.Arguments = map[string]interface{}{"_raw": tc.RawArgs}
					}
				}

				// Send complete tool call
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
