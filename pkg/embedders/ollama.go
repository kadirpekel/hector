package embedders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/ollama"
)

// ============================================================================
// OLLAMA EMBEDDER CONFIGURATION
// ============================================================================

// ============================================================================
// OLLAMA EMBEDDER IMPLEMENTATION
// ============================================================================

// OllamaEmbedder implements the EmbedderProvider interface using Ollama
type OllamaEmbedder struct {
	config *config.EmbedderProviderConfig
	client *ollama.Client
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
		config: config,
		client: ollama.NewClientWithTimeout(config.Host, time.Duration(config.Timeout)*time.Second),
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

	// Send request with retries using shared client
	var resp *http.Response
	var err error
	for attempt := 0; attempt < e.config.MaxRetries; attempt++ {
		resp, err = e.client.MakeRequest(ctx, "/api/embeddings", request)
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

// Close closes the Ollama embedder and releases resources
func (oe *OllamaEmbedder) Close() error {
	// Ollama embedder doesn't need explicit cleanup
	return nil
}
