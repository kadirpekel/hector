package embedders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// OLLAMA EMBEDDER CONFIGURATION
// ============================================================================

// OllamaEmbedderConfig holds configuration for the Ollama embedder provider
type OllamaEmbedderConfig struct {
	Provider   string `yaml:"provider"`    // Always "ollama"
	Model      string `yaml:"model"`       // Model name
	Host       string `yaml:"host"`        // Host for ollama
	Dimension  int    `yaml:"dimension"`   // Embedding dimension
	Timeout    int    `yaml:"timeout"`     // Request timeout in seconds
	MaxRetries int    `yaml:"max_retries"` // Maximum retry attempts
}

// GetProviderType implements ProviderConfig.GetProviderType
func (c *OllamaEmbedderConfig) GetProviderType() interfaces.ProviderType {
	return interfaces.ProviderTypeEmbedder
}

// GetProviderName implements ProviderConfig.GetProviderName
func (c *OllamaEmbedderConfig) GetProviderName() string {
	return "ollama"
}

// CreateProvider implements ProviderConfig.CreateProvider
func (c *OllamaEmbedderConfig) CreateProvider() (interface{}, error) {
	provider := &OllamaEmbedder{
		baseURL:   c.Host,
		model:     c.Model,
		dimension: c.Dimension,
		httpClient: &http.Client{
			Timeout: time.Duration(c.Timeout) * time.Second,
		},
		maxRetries: c.MaxRetries,
	}

	// Set defaults if not specified
	if provider.baseURL == "" {
		provider.baseURL = "http://localhost:11434"
	}
	if provider.model == "" {
		provider.model = "nomic-embed-text"
	}
	if provider.dimension == 0 {
		provider.dimension = 768
	}
	if provider.httpClient.Timeout == 0 {
		provider.httpClient.Timeout = 30 * time.Second
	}
	if provider.maxRetries == 0 {
		provider.maxRetries = 3
	}

	return provider, nil
}

// Validate implements ProviderConfig.Validate
func (c *OllamaEmbedderConfig) Validate() error {
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.Dimension < 0 {
		return fmt.Errorf("dimension must be positive")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be positive")
	}
	return nil
}

// ============================================================================
// OLLAMA EMBEDDER IMPLEMENTATION
// ============================================================================

// OllamaEmbedder implements the EmbeddingProvider interface using Ollama
type OllamaEmbedder struct {
	baseURL    string
	model      string
	dimension  int
	httpClient *http.Client
	maxRetries int
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
type OllamaConfig struct {
	BaseURL    string
	Model      string
	Dimension  int
	Timeout    time.Duration
	MaxRetries int
}

// Default Ollama configuration
var (
	DefaultOllamaConfig = OllamaConfig{
		BaseURL:    "http://localhost:11434",
		Model:      "nomic-embed-text", // Good default embedding model
		Dimension:  768,                // Default dimension for nomic-embed-text
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}
)

// NewOllamaEmbedder creates a new Ollama embedder with default configuration (legacy method)
func NewOllamaEmbedder() *OllamaEmbedder {
	config := &OllamaEmbedderConfig{
		Provider:   "ollama",
		Model:      "nomic-embed-text",
		Host:       "http://localhost:11434",
		Dimension:  768,
		Timeout:    30,
		MaxRetries: 3,
	}

	provider, _ := config.CreateProvider()
	return provider.(*OllamaEmbedder)
}

// NewOllamaEmbedderFromConfig creates a new Ollama embedder from config
func NewOllamaEmbedderFromConfig(config *OllamaEmbedderConfig) (*OllamaEmbedder, error) {
	provider, err := config.CreateProvider()
	if err != nil {
		return nil, err
	}
	return provider.(*OllamaEmbedder), nil
}

// NewOllamaEmbedderWithConfig creates a new Ollama embedder with custom configuration
func NewOllamaEmbedderWithConfig(config OllamaConfig) *OllamaEmbedder {
	return &OllamaEmbedder{
		baseURL:   config.BaseURL,
		model:     config.Model,
		dimension: config.Dimension,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// NewOllamaEmbedderWithModel creates a new Ollama embedder with a specific model
func NewOllamaEmbedderWithModel(model string) *OllamaEmbedder {
	config := DefaultOllamaConfig
	config.Model = model
	return NewOllamaEmbedderWithConfig(config)
}

// Embed generates embeddings for the given text using Ollama
func (e *OllamaEmbedder) Embed(text string) ([]float32, error) {
	return e.EmbedWithContext(context.Background(), text)
}

// EmbedWithContext generates embeddings for the given text using Ollama with context
func (e *OllamaEmbedder) EmbedWithContext(ctx context.Context, text string) ([]float32, error) {
	// Prepare the request
	request := OllamaEmbedRequest{
		Model:  e.model,
		Prompt: text,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/embeddings", e.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request with retries
	var resp *http.Response
	for attempt := 0; attempt < e.maxRetries; attempt++ {
		resp, err = e.httpClient.Do(req)
		if err == nil {
			break
		}

		if attempt < e.maxRetries-1 {
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
	return e.model
}

// SetModel changes the model being used
func (e *OllamaEmbedder) SetModel(model string) {
	e.model = model
}

// GetBaseURL returns the current base URL
func (e *OllamaEmbedder) GetBaseURL() string {
	return e.baseURL
}

// SetBaseURL changes the base URL
func (e *OllamaEmbedder) SetBaseURL(baseURL string) {
	e.baseURL = baseURL
}

// GetDimension returns the embedding dimension
func (e *OllamaEmbedder) GetDimension() int {
	return e.dimension
}

// GetModelName returns the model name
func (e *OllamaEmbedder) GetModelName() string {
	return e.model
}

// HealthCheck verifies that the Ollama service is available
func (e *OllamaEmbedder) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/tags", e.baseURL)
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

// NewOllamaEmbedderWithPreset creates an Ollama embedder with a preset model
func NewOllamaEmbedderWithPreset(preset string) *OllamaEmbedder {
	return NewOllamaEmbedderWithModel(preset)
}
