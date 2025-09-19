package llms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// OLLAMA LLM PROVIDER CONFIGURATION
// ============================================================================

// OllamaConfig holds configuration for the Ollama LLM provider
type OllamaConfig struct {
	Provider    string  `yaml:"provider"`    // Always "ollama"
	Model       string  `yaml:"model"`       // Model name
	Host        string  `yaml:"host"`        // Host for ollama
	Temperature float64 `yaml:"temperature"` // Temperature setting
	MaxTokens   int     `yaml:"max_tokens"`  // Max tokens
	Timeout     int     `yaml:"timeout"`     // Request timeout in seconds
}

// GetProviderType implements ProviderConfig.GetProviderType
func (c *OllamaConfig) GetProviderType() interfaces.ProviderType {
	return interfaces.ProviderTypeLLM
}

// GetProviderName implements ProviderConfig.GetProviderName
func (c *OllamaConfig) GetProviderName() string {
	return "ollama"
}

// CreateProvider implements ProviderConfig.CreateProvider
func (c *OllamaConfig) CreateProvider() (interface{}, error) {
	// Set defaults before creating provider
	c.SetDefaults()

	provider := &OllamaProvider{
		config: c,
		client: &http.Client{
			Timeout: time.Duration(c.Timeout) * time.Second,
		},
	}

	return provider, nil
}

// SetDefaults sets default values for OllamaConfig
func (c *OllamaConfig) SetDefaults() {
	if c.Model == "" {
		c.Model = "llama3.2"
	}
	if c.Host == "" {
		c.Host = "http://localhost:11434"
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
func (c *OllamaConfig) Validate() error {
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
// OLLAMA LLM PROVIDER IMPLEMENTATION
// ============================================================================

// OllamaProvider implements LLMProvider for Ollama
type OllamaProvider struct {
	config *OllamaConfig // Hold the config object
	client *http.Client  // HTTP client for requests
}

// NewOllamaProvider creates a new Ollama LLM provider (legacy method)
func NewOllamaProvider(model string) *OllamaProvider {
	config := &OllamaConfig{
		Provider:    "ollama",
		Model:       model,
		Host:        "http://localhost:11434",
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     60,
	}

	provider, _ := config.CreateProvider()
	return provider.(*OllamaProvider)
}

// NewOllamaProviderFromConfig creates a new Ollama provider from config
func NewOllamaProviderFromConfig(config *OllamaConfig) (*OllamaProvider, error) {
	provider, err := config.CreateProvider()
	if err != nil {
		return nil, err
	}
	return provider.(*OllamaProvider), nil
}

// WithBaseURL sets the Ollama base URL
func (o *OllamaProvider) WithBaseURL(url string) *OllamaProvider {
	o.config.Host = url
	return o
}

// WithTemperature sets the temperature
func (o *OllamaProvider) WithTemperature(temp float64) *OllamaProvider {
	o.config.Temperature = temp
	return o
}

// WithMaxTokens sets the maximum tokens
func (o *OllamaProvider) WithMaxTokens(tokens int) *OllamaProvider {
	o.config.MaxTokens = tokens
	return o
}

// Generate implements LLMProvider.Generate
func (o *OllamaProvider) Generate(prompt string) (string, int, error) {
	// Call Ollama API with the pre-built prompt
	response, err := o.callOllamaAPI(prompt)
	if err != nil {
		return "", 0, err
	}

	// Estimate token usage
	tokensUsed := EstimateTokens(response)

	return response, tokensUsed, nil
}

// GenerateStreaming implements LLMProvider.GenerateStreaming
func (o *OllamaProvider) GenerateStreaming(prompt string) (<-chan string, error) {
	ch := make(chan string)

	go func() {
		defer close(ch)

		// Call Ollama streaming API with the pre-built prompt
		err := o.callOllamaStreamingAPI(prompt, ch)
		if err != nil {
			ch <- "Error: " + err.Error()
		}
	}()

	return ch, nil
}

// GetModelName implements LLMProvider.GetModelName
func (o *OllamaProvider) GetModelName() string {
	return o.config.Model
}

// GetMaxTokens implements LLMProvider.GetMaxTokens
func (o *OllamaProvider) GetMaxTokens() int {
	return o.config.MaxTokens
}

// GetTemperature implements LLMProvider.GetTemperature
func (o *OllamaProvider) GetTemperature() float64 {
	return o.config.Temperature
}

// Close implements LLMProvider.Close
func (o *OllamaProvider) Close() error {
	// Ollama doesn't require explicit closing
	return nil
}

// callOllamaAPI calls the Ollama API for generation
func (o *OllamaProvider) callOllamaAPI(prompt string) (string, error) {
	// Prepare the request payload
	payload := map[string]interface{}{
		"model":  o.config.Model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": o.config.Temperature,
			"num_predict": o.config.MaxTokens,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make the HTTP request
	url := fmt.Sprintf("%s/api/generate", o.config.Host)
	resp, err := o.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var response struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Response, nil
}

// callOllamaStreamingAPI calls the Ollama streaming API
func (o *OllamaProvider) callOllamaStreamingAPI(prompt string, ch chan<- string) error {
	// Prepare the request payload
	payload := map[string]interface{}{
		"model":  o.config.Model,
		"prompt": prompt,
		"stream": true,
		"options": map[string]interface{}{
			"temperature": o.config.Temperature,
			"num_predict": o.config.MaxTokens,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make the HTTP request
	url := fmt.Sprintf("%s/api/generate", o.config.Host)
	resp, err := o.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Stream the response
	decoder := json.NewDecoder(resp.Body)
	for {
		var response struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}

		if err := decoder.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode streaming response: %w", err)
		}

		if response.Response != "" {
			ch <- response.Response
		}

		if response.Done {
			break
		}
	}

	return nil
}
