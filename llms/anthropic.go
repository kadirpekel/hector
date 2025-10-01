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
type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Stream      bool               `json:"stream"`
	System      string             `json:"system,omitempty"`
}

// AnthropicMessage represents a message in the conversation
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicDelta represents incremental content in streaming
type AnthropicDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
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

// Generate generates a response given a pre-built prompt
func (p *AnthropicProvider) Generate(prompt string) (string, int, error) {
	request := p.buildRequest(prompt, false)

	response, err := p.makeRequest(request)
	if err != nil {
		return "", 0, err
	}

	if response.Error != nil {
		return "", 0, fmt.Errorf("Anthropic API error: %s", response.Error.Message)
	}

	if len(response.Content) == 0 {
		return "", 0, fmt.Errorf("no content in response")
	}

	content := response.Content[0].Text
	tokensUsed := response.Usage.InputTokens + response.Usage.OutputTokens

	return content, tokensUsed, nil
}

// GenerateStreaming generates a streaming response given a pre-built prompt
func (p *AnthropicProvider) GenerateStreaming(prompt string) (<-chan string, error) {
	request := p.buildRequest(prompt, true)

	responseChan := make(chan string, 100)

	go func() {
		defer close(responseChan)

		err := p.makeStreamingRequest(request, responseChan)
		if err != nil {
			responseChan <- fmt.Sprintf("Error: %v", err)
		}
	}()

	return responseChan, nil
}

// buildRequest builds an Anthropic request
func (p *AnthropicProvider) buildRequest(prompt string, stream bool) AnthropicRequest {
	return AnthropicRequest{
		Model: p.config.Model,
		Messages: []AnthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
		Stream:      stream,
	}
}

// makeRequest makes a non-streaming request to Anthropic API
func (p *AnthropicProvider) makeRequest(request AnthropicRequest) (*AnthropicResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.config.Host+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// makeStreamingRequest makes a streaming request to Anthropic API
func (p *AnthropicProvider) makeStreamingRequest(request AnthropicRequest, responseChan chan<- string) error {
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

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

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
		case "content_block_delta":
			if streamResp.Delta != nil && streamResp.Delta.Text != "" {
				responseChan <- streamResp.Delta.Text
			}
		case "message_stop":
			// Stream finished
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read streaming response: %w", err)
	}

	return nil
}
