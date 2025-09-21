package llms

import (
	"fmt"
	"time"

	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// TGI PROVIDER CONFIGURATION
// ============================================================================

// TGIConfig holds configuration for the TGI LLM provider
type TGIConfig struct {
	Provider    string  `yaml:"provider"`    // Always "tgi"
	Model       string  `yaml:"model"`       // Model name
	Host        string  `yaml:"host"`        // Host for TGI
	APIKey      string  `yaml:"api_key"`     // API key (optional)
	Temperature float64 `yaml:"temperature"` // Temperature setting
	MaxTokens   int     `yaml:"max_tokens"`  // Max tokens
	Timeout     int     `yaml:"timeout"`     // Request timeout in seconds
}

// GetProviderType implements ProviderConfig.GetProviderType
func (c *TGIConfig) GetProviderType() interfaces.ProviderType {
	return interfaces.ProviderTypeLLM
}

// GetProviderName implements ProviderConfig.GetProviderName
func (c *TGIConfig) GetProviderName() string {
	return "tgi"
}

// CreateProvider implements ProviderConfig.CreateProvider
func (c *TGIConfig) CreateProvider() (interface{}, error) {
	provider := &TGIProvider{
		baseURL:     c.Host,
		model:       c.Model,
		temperature: c.Temperature,
		maxTokens:   c.MaxTokens,
		apiKey:      c.APIKey,
		timeout:     time.Duration(c.Timeout) * time.Second,
	}

	// Set defaults if not specified
	if provider.baseURL == "" {
		provider.baseURL = "http://localhost:8080"
	}
	if provider.temperature == 0 {
		provider.temperature = 0.7
	}
	if provider.maxTokens == 0 {
		provider.maxTokens = 1000
	}
	if provider.timeout == 0 {
		provider.timeout = 60 * time.Second
	}

	return provider, nil
}

// Validate implements ProviderConfig.Validate
func (c *TGIConfig) Validate() error {
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
// TGI LLM PROVIDER IMPLEMENTATION
// ============================================================================

// TGIProvider implements LLMProvider for Hugging Face Text Generation Inference
type TGIProvider struct {
	baseURL     string
	model       string
	temperature float64
	maxTokens   int
	apiKey      string
	timeout     time.Duration
}

// NewTGIProvider creates a new TGI LLM provider (legacy method)
func NewTGIProvider(model string) *TGIProvider {
	config := &TGIConfig{
		Provider:    "tgi",
		Model:       model,
		Host:        "http://localhost:8080",
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     60,
	}

	provider, _ := config.CreateProvider()
	return provider.(*TGIProvider)
}

// NewTGIProviderFromConfig creates a new TGI provider from config
func NewTGIProviderFromConfig(config *TGIConfig) (*TGIProvider, error) {
	provider, err := config.CreateProvider()
	if err != nil {
		return nil, err
	}
	return provider.(*TGIProvider), nil
}

// WithBaseURL sets the TGI base URL
func (t *TGIProvider) WithBaseURL(url string) *TGIProvider {
	t.baseURL = url
	return t
}

// WithAPIKey sets the API key
func (t *TGIProvider) WithAPIKey(apiKey string) *TGIProvider {
	t.apiKey = apiKey
	return t
}

// WithTemperature sets the temperature
func (t *TGIProvider) WithTemperature(temp float64) *TGIProvider {
	t.temperature = temp
	return t
}

// WithMaxTokens sets the maximum tokens
func (t *TGIProvider) WithMaxTokens(tokens int) *TGIProvider {
	t.maxTokens = tokens
	return t
}

// Generate implements LLMProvider.Generate
func (t *TGIProvider) Generate(prompt string) (string, int, error) {
	// Call TGI API with the pre-built prompt
	response, err := t.callTGIAPI(prompt)
	if err != nil {
		return "", 0, err
	}

	// Estimate token usage
	tokensUsed := EstimateTokens(response)

	return response, tokensUsed, nil
}

// GenerateStreaming implements LLMProvider.GenerateStreaming
func (t *TGIProvider) GenerateStreaming(prompt string) (<-chan string, error) {
	ch := make(chan string)

	go func() {
		defer close(ch)

		// Call TGI streaming API with the pre-built prompt
		err := t.callTGIStreamingAPI(prompt, ch)
		if err != nil {
			ch <- "Error: " + err.Error()
		}
	}()

	return ch, nil
}

// GetModelName implements LLMProvider.GetModelName
func (t *TGIProvider) GetModelName() string {
	return t.model
}

// GetMaxTokens implements LLMProvider.GetMaxTokens
func (t *TGIProvider) GetMaxTokens() int {
	return t.maxTokens
}

// GetTemperature implements LLMProvider.GetTemperature
func (t *TGIProvider) GetTemperature() float64 {
	return t.temperature
}

// Close implements LLMProvider.Close
func (t *TGIProvider) Close() error {
	// TGI doesn't require explicit closing
	return nil
}

// callTGIAPI calls the TGI API for generation
func (t *TGIProvider) callTGIAPI(_ string) (string, error) {
	// TODO: Implement actual TGI API call
	// For now, return a mock response
	return "This is a mock response from TGI. The actual implementation would call the TGI API.", nil
}

// callTGIStreamingAPI calls the TGI streaming API
func (t *TGIProvider) callTGIStreamingAPI(_ string, ch chan<- string) error {
	// TODO: Implement actual TGI streaming API call
	// For now, send mock streaming response
	ch <- "This is a mock streaming response from TGI."
	return nil
}
