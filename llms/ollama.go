package llms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/ollama"
	"github.com/kadirpekel/hector/utils"
)

// ============================================================================
// OLLAMA LLM PROVIDER CONFIGURATION
// ============================================================================

// OllamaProvider uses the new LLMProviderConfig from config/types.go

// ============================================================================
// OLLAMA LLM PROVIDER IMPLEMENTATION
// ============================================================================

// OllamaProvider implements LLMProvider for Ollama
type OllamaProvider struct {
	config *config.LLMProviderConfig // Hold the config object
	client *ollama.Client            // Shared Ollama client
}

// NewOllamaProvider creates a new Ollama LLM provider
func NewOllamaProvider(model string) *OllamaProvider {
	config := &config.LLMProviderConfig{
		Type:        "ollama",
		Model:       model,
		Host:        "http://localhost:11434",
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     60,
	}

	provider, _ := NewOllamaProviderFromConfig(config)
	return provider
}

// NewOllamaProviderFromConfig creates a new Ollama provider from config
func NewOllamaProviderFromConfig(config *config.LLMProviderConfig) (*OllamaProvider, error) {
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &OllamaProvider{
		config: config,
		client: ollama.NewClientWithTimeout(config.Host, time.Duration(config.Timeout)*time.Second),
	}, nil
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
	tokensUsed := utils.EstimateTokens(response)

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

	// Make the HTTP request using shared client
	resp, err := o.client.MakeRequest(context.Background(), "/api/generate", payload)
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

	// Make the streaming HTTP request using shared client
	resp, err := o.client.MakeStreamingRequest(context.Background(), "/api/generate", payload)
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
