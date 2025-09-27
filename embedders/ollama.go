package embedders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// OLLAMA EMBEDDER CONFIGURATION
// ============================================================================

// OllamaEmbedderConfig is defined in config/providers.go

// Methods GetProviderType and GetProviderName are defined in config/providers.go

// SetDefaults method is defined in config/providers.go

// Validate and SetDefaults methods are defined in config/providers.go

// Validate method is defined in config/providers.go

// ============================================================================
// OLLAMA EMBEDDER IMPLEMENTATION
// ============================================================================

// OllamaEmbedder implements the EmbedderProvider interface using Ollama
type OllamaEmbedder struct {
	config     *config.EmbedderProviderConfig
	httpClient *http.Client
}

// OllamaEmbedRequest represents the request structure for Ollama embeddings
type OllamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbedResponse represents the response structure from Ollama embeddings
type OllamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// OllamaConfig holds configuration for the Ollama embedder

// NewOllamaEmbedder creates a new Ollama embedder with default configuration
func NewOllamaEmbedder() *OllamaEmbedder {
	config := &config.EmbedderProviderConfig{
		Type:       "ollama",
		Model:      "nomic-embed-text",
		Host:       "http://localhost:11434",
		Dimension:  768,
		Timeout:    30,
		MaxRetries: 3,
	}

	embedder, _ := NewOllamaEmbedderFromConfig(config)
	return embedder
}

// NewOllamaEmbedderFromConfig creates a new Ollama embedder from config
func NewOllamaEmbedderFromConfig(config *config.EmbedderProviderConfig) (*OllamaEmbedder, error) {
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &OllamaEmbedder{
		config:     config,
		httpClient: &http.Client{Timeout: time.Duration(config.Timeout) * time.Second},
	}, nil
}

// Embed generates embeddings for the given text using Ollama
func (e *OllamaEmbedder) Embed(text string) ([]float32, error) {
	return e.EmbedWithContext(context.Background(), text)
}

// EmbedWithContext generates embeddings for the given text using Ollama with context
func (e *OllamaEmbedder) EmbedWithContext(ctx context.Context, text string) ([]float32, error) {
	// Prepare the request
	request := OllamaEmbedRequest{
		Model:  e.config.Model,
		Prompt: text,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/embeddings", e.config.Host)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request with retries
	var resp *http.Response
	for attempt := 0; attempt < e.config.MaxRetries; attempt++ {
		resp, err = e.httpClient.Do(req)
		if err == nil {
			break
		}

		if attempt < e.config.MaxRetries-1 {
			time.Sleep(time.Duration(attempt+1) * time.Second) // Exponential backoff
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response OllamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Embedding) == 0 {
		return nil, fmt.Errorf("received empty embedding from Ollama")
	}

	return response.Embedding, nil
}

// GetModel returns the current model being used
func (e *OllamaEmbedder) GetModel() string {
	return e.config.Model
}

// SetModel changes the model being used
func (e *OllamaEmbedder) SetModel(model string) {
	e.config.Model = model
}

// GetBaseURL returns the current base URL
func (e *OllamaEmbedder) GetBaseURL() string {
	return e.config.Host
}

// SetBaseURL changes the base URL
func (e *OllamaEmbedder) SetBaseURL(baseURL string) {
	e.config.Host = baseURL
}

// GetDimension returns the embedding dimension
func (e *OllamaEmbedder) GetDimension() int {
	return e.config.Dimension
}

// GetModelName returns the model name
func (e *OllamaEmbedder) GetModelName() string {
	return e.config.Model
}

// HealthCheck verifies that the Ollama service is available
func (e *OllamaEmbedder) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/tags", e.config.Host)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama service is not available: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama service returned status %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// OLLAMA MODEL PRESETS
// ============================================================================

// Popular Ollama embedding models
var (
	// Nomic AI models (recommended for embeddings)
	OllamaNomicEmbedText   = "nomic-embed-text"
	OllamaNomicEmbedTextV2 = "nomic-embed-text-v2"

	// Sentence Transformers models
	OllamaAllMiniLML6V2  = "all-minilm:l6-v2"
	OllamaAllMpnetBaseV2 = "all-mpnet-base-v2"

	// BGE models
	OllamaBGESmallEnV15 = "bge-small-en-v1.5"
	OllamaBGELargeEnV15 = "bge-large-en-v1.5"

	// E5 models
	OllamaE5SmallV2 = "e5-small-v2"
	OllamaE5BaseV2  = "e5-base-v2"
	OllamaE5LargeV2 = "e5-large-v2"
)

// IsHealthy checks if the Ollama embedder is healthy and ready to use
func (oe *OllamaEmbedder) IsHealthy(ctx context.Context) bool {
	// Simple health check - try to embed a test string
	_, err := oe.Embed("test")
	return err == nil
}

// Close closes the Ollama embedder and releases resources
func (oe *OllamaEmbedder) Close() error {
	// Ollama embedder doesn't need explicit cleanup
	return nil
}
