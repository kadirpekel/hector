package llms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// OPENAI PROVIDER CONFIGURATION
// ============================================================================

// OpenAIConfig holds configuration for the OpenAI LLM provider
type OpenAIConfig struct {
	Provider    string  `yaml:"provider"`    // Always "openai"
	Model       string  `yaml:"model"`       // Model name
	APIKey      string  `yaml:"api_key"`     // API key
	Host        string  `yaml:"host"`        // Custom host (optional)
	Temperature float64 `yaml:"temperature"` // Temperature setting
	MaxTokens   int     `yaml:"max_tokens"`  // Max tokens
	Timeout     int     `yaml:"timeout"`     // Request timeout in seconds
}

// GetProviderType implements ProviderConfig.GetProviderType
func (c *OpenAIConfig) GetProviderType() interfaces.ProviderType {
	return interfaces.ProviderTypeLLM
}

// GetProviderName implements ProviderConfig.GetProviderName
func (c *OpenAIConfig) GetProviderName() string {
	return "openai"
}

// CreateProvider implements ProviderConfig.CreateProvider
func (c *OpenAIConfig) CreateProvider() (interface{}, error) {
	// Set defaults before creating provider
	c.SetDefaults()

	provider := &OpenAIProvider{
		config: c,
		client: &http.Client{
			Timeout: time.Duration(c.Timeout) * time.Second,
		},
	}

	return provider, nil
}

// SetDefaults sets default values for OpenAIConfig
func (c *OpenAIConfig) SetDefaults() {
	if c.Model == "" {
		c.Model = "gpt-3.5-turbo"
	}
	if c.Host == "" {
		c.Host = "https://api.openai.com/v1"
	}
	if c.Temperature == 0 {
		c.Temperature = 0.7
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = 1000
	}
	if c.Timeout == 0 {
		c.Timeout = 60
	}
}

// Validate implements ProviderConfig.Validate
func (c *OpenAIConfig) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.Temperature < 0 || c.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if c.MaxTokens < 0 {
		return fmt.Errorf("max_tokens must be positive")
	}
	return nil
}

// ============================================================================
// OPENAI PROVIDER IMPLEMENTATION
// ============================================================================

// OpenAIProvider implements LLMProvider for OpenAI API
type OpenAIProvider struct {
	config *OpenAIConfig // Hold the config object
	client *http.Client  // HTTP client for requests
}

// OpenAIRequest represents the request payload for OpenAI API
type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	Stream      bool      `json:"stream"`
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

// NewOpenAIProvider creates a new OpenAI provider (legacy method)
func NewOpenAIProvider(apiKey string, model string) *OpenAIProvider {
	config := &OpenAIConfig{
		Provider:    "openai",
		Model:       model,
		APIKey:      apiKey,
		Host:        "https://api.openai.com/v1",
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     60,
	}

	provider, _ := config.CreateProvider()
	return provider.(*OpenAIProvider)
}

// NewOpenAIProviderFromConfig creates a new OpenAI provider from config
func NewOpenAIProviderFromConfig(config *OpenAIConfig) (*OpenAIProvider, error) {
	provider, err := config.CreateProvider()
	if err != nil {
		return nil, err
	}
	return provider.(*OpenAIProvider), nil
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
	request := OpenAIRequest{
		Model:       p.config.Model,
		Messages:    []Message{{Role: "user", Content: prompt}},
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
		Stream:      false,
	}

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
	request := OpenAIRequest{
		Model:       p.config.Model,
		Messages:    []Message{{Role: "user", Content: prompt}},
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
		Stream:      true,
	}

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

	decoder := json.NewDecoder(resp.Body)
	for {
		var streamResp OpenAIStreamResponse
		if err := decoder.Decode(&streamResp); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode streaming response: %w", err)
		}

		if streamResp.Error != nil {
			return fmt.Errorf("OpenAI API error: %s", streamResp.Error.Message)
		}

		if len(streamResp.Choices) > 0 {
			choice := streamResp.Choices[0]
			if choice.Delta.Content != "" {
				responseChan <- choice.Delta.Content
			}
			if choice.FinishReason != "" {
				break
			}
		}
	}

	return nil
}
