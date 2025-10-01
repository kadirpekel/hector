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
// OPENAI PROVIDER CONFIGURATION
// ============================================================================

// OpenAIProvider uses the new LLMProviderConfig from config/types.go

// ============================================================================
// OPENAI PROVIDER IMPLEMENTATION
// ============================================================================

// OpenAIProvider implements LLMProvider for OpenAI API
type OpenAIProvider struct {
	config *config.LLMProviderConfig // Hold the config object
	client *http.Client              // HTTP client for requests
}

// OpenAIRequest represents the request payload for OpenAI API
type OpenAIRequest struct {
	Model               string    `json:"model"`
	Messages            []Message `json:"messages"`
	MaxTokens           int       `json:"max_tokens,omitempty"`            // Legacy parameter
	MaxCompletionTokens int       `json:"max_completion_tokens,omitempty"` // New parameter
	Temperature         float64   `json:"temperature"`
	Stream              bool      `json:"stream"`
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
	Error   *Error         `json:"error,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Choice represents a response choice
type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// StreamChoice represents a streaming response choice
type StreamChoice struct {
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

// Delta represents incremental content in streaming
type Delta struct {
	Content string `json:"content"`
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

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string, model string) *OpenAIProvider {
	config := &config.LLMProviderConfig{
		Type:        "openai",
		Model:       model,
		APIKey:      apiKey,
		Host:        "https://api.openai.com/v1",
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     60,
	}

	provider, _ := NewOpenAIProviderFromConfig(config)
	return provider
}

// NewOpenAIProviderFromConfig creates a new OpenAI provider from config
func NewOpenAIProviderFromConfig(config *config.LLMProviderConfig) (*OpenAIProvider, error) {
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &OpenAIProvider{
		config: config,
		client: &http.Client{Timeout: time.Duration(config.Timeout) * time.Second},
	}, nil
}

// WithBaseURL sets a custom base URL (useful for proxies or local servers)
func (p *OpenAIProvider) WithBaseURL(baseURL string) *OpenAIProvider {
	p.config.Host = strings.TrimSuffix(baseURL, "/")
	return p
}

// WithMaxTokens sets the maximum tokens for generation
func (p *OpenAIProvider) WithMaxTokens(maxTokens int) *OpenAIProvider {
	p.config.MaxTokens = maxTokens
	return p
}

// WithTemperature sets the temperature for generation
func (p *OpenAIProvider) WithTemperature(temperature float64) *OpenAIProvider {
	p.config.Temperature = temperature
	return p
}

// Generate generates a response given a pre-built prompt
func (p *OpenAIProvider) Generate(prompt string) (string, int, error) {
	request := p.buildRequest(prompt, false)

	response, err := p.makeRequest(request)
	if err != nil {
		return "", 0, err
	}

	if response.Error != nil {
		return "", 0, fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", 0, fmt.Errorf("no response choices returned")
	}

	content := response.Choices[0].Message.Content
	tokensUsed := response.Usage.TotalTokens

	return content, tokensUsed, nil
}

// GenerateStreaming generates a streaming response given a pre-built prompt
func (p *OpenAIProvider) GenerateStreaming(prompt string) (<-chan string, error) {
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

// GetModelName returns the model name
func (p *OpenAIProvider) GetModelName() string {
	return p.config.Model
}

// buildRequest builds an OpenAI request with appropriate parameters based on model
func (p *OpenAIProvider) buildRequest(prompt string, stream bool) OpenAIRequest {
	request := OpenAIRequest{
		Model:       p.config.Model,
		Messages:    []Message{{Role: "user", Content: prompt}},
		Temperature: p.config.Temperature,
		Stream:      stream,
	}

	// Use appropriate token parameter based on model
	if p.isNewerModel() {
		request.MaxCompletionTokens = p.config.MaxTokens
	} else {
		request.MaxTokens = p.config.MaxTokens
	}

	// Handle temperature restrictions for certain models
	if p.hasTemperatureRestrictions() {
		request.Temperature = 1.0 // Default temperature for restricted models
	}

	return request
}

// hasTemperatureRestrictions checks if the model has temperature restrictions
func (p *OpenAIProvider) hasTemperatureRestrictions() bool {
	// Models that only support default temperature (1.0)
	restrictedModels := []string{
		"gpt-5-nano",
	}

	for _, model := range restrictedModels {
		if p.config.Model == model {
			return true
		}
	}

	return false
}

// isNewerModel checks if the model requires max_completion_tokens instead of max_tokens
func (p *OpenAIProvider) isNewerModel() bool {
	// Models that require max_completion_tokens
	newerModels := []string{
		"gpt-5-nano",
		"gpt-5",
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4",
	}

	for _, model := range newerModels {
		if p.config.Model == model {
			return true
		}
	}

	return false
}

// GetMaxTokens returns the maximum tokens for generation
func (p *OpenAIProvider) GetMaxTokens() int {
	return p.config.MaxTokens
}

// GetTemperature returns the temperature setting
func (p *OpenAIProvider) GetTemperature() float64 {
	return p.config.Temperature
}

// Close closes the provider
func (p *OpenAIProvider) Close() error {
	// OpenAI provider doesn't need explicit cleanup
	return nil
}

// makeRequest makes a non-streaming request to OpenAI API
func (p *OpenAIProvider) makeRequest(request OpenAIRequest) (*OpenAIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.config.Host+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// makeStreamingRequest makes a streaming request to OpenAI API
func (p *OpenAIProvider) makeStreamingRequest(request OpenAIRequest, responseChan chan<- string) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.config.Host+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

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

		// Skip empty lines and non-data lines
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		// Extract JSON data after "data: "
		jsonData := strings.TrimPrefix(line, "data: ")

		// Skip the final "[DONE]" message
		if jsonData == "[DONE]" {
			break
		}

		// Parse JSON response
		var streamResp OpenAIStreamResponse
		if err := json.Unmarshal([]byte(jsonData), &streamResp); err != nil {
			return fmt.Errorf("failed to decode streaming response: %w, data: %s", err, jsonData)
		}

		if streamResp.Error != nil {
			return fmt.Errorf("OpenAI API error: %s", streamResp.Error.Message)
		}

		if len(streamResp.Choices) > 0 {
			choice := streamResp.Choices[0]
			if choice.Delta.Content != "" {
				responseChan <- choice.Delta.Content
			}
			// Handle finish_reason to properly terminate stream
			if choice.FinishReason != "" {
				// Stream finished
				break
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read streaming response: %w", err)
	}

	return nil
}
