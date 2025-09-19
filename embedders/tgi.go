package embedders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kadirpekel/hector/interfaces"
)

// TGIEmbedderConfig holds configuration for TGI embedding requests (legacy)
type TGIEmbedderConfig struct {
	// Server configuration
	BaseURL string        `json:"base_url"`
	Timeout time.Duration `json:"timeout"`

	// Model configuration
	ModelID   string `json:"model_id"`
	Dimension int    `json:"dimension"`

	// Request options
	WaitForModel bool `json:"wait_for_model"`
	Normalize    bool `json:"normalize"`

	// Advanced options
	Truncate  bool `json:"truncate"`
	MaxLength int  `json:"max_length"`
}

// ============================================================================
// TGI EMBEDDER PROVIDER CONFIGURATION
// ============================================================================

// TGIEmbedderProviderConfig holds configuration for the TGI embedder provider
type TGIEmbedderProviderConfig struct {
	Provider     string `yaml:"provider"`       // Always "tgi"
	Model        string `yaml:"model"`          // Model name
	Host         string `yaml:"host"`           // Host for TGI
	Dimension    int    `yaml:"dimension"`      // Embedding dimension
	Timeout      int    `yaml:"timeout"`        // Request timeout in seconds
	WaitForModel bool   `yaml:"wait_for_model"` // Wait for model to be ready
	Normalize    bool   `yaml:"normalize"`      // Normalize vectors
	Truncate     bool   `yaml:"truncate"`       // Truncate text
	MaxLength    int    `yaml:"max_length"`     // Maximum text length
}

// GetProviderType implements ProviderConfig.GetProviderType
func (c *TGIEmbedderProviderConfig) GetProviderType() interfaces.ProviderType {
	return interfaces.ProviderTypeEmbedder
}

// GetProviderName implements ProviderConfig.GetProviderName
func (c *TGIEmbedderProviderConfig) GetProviderName() string {
	return "tgi"
}

// CreateProvider implements ProviderConfig.CreateProvider
func (c *TGIEmbedderProviderConfig) CreateProvider() (interface{}, error) {
	config := TGIEmbedderConfig{
		BaseURL:      c.Host,
		Timeout:      time.Duration(c.Timeout) * time.Second,
		ModelID:      c.Model,
		Dimension:    c.Dimension,
		WaitForModel: c.WaitForModel,
		Normalize:    c.Normalize,
		Truncate:     c.Truncate,
		MaxLength:    c.MaxLength,
	}

	// Set defaults if not specified
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:8080"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.ModelID == "" {
		config.ModelID = "BAAI/bge-small-en-v1.5"
	}
	if config.Dimension == 0 {
		config.Dimension = 384
	}
	if !config.WaitForModel {
		config.WaitForModel = true
	}
	if !config.Normalize {
		config.Normalize = true
	}
	if !config.Truncate {
		config.Truncate = true
	}
	if config.MaxLength == 0 {
		config.MaxLength = 512
	}

	return NewTGIEmbedderWithConfig(config), nil
}

// Validate implements ProviderConfig.Validate
func (c *TGIEmbedderProviderConfig) Validate() error {
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.Dimension < 0 {
		return fmt.Errorf("dimension must be positive")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.MaxLength < 0 {
		return fmt.Errorf("max_length must be positive")
	}
	return nil
}

// ============================================================================
// TGI EMBEDDER IMPLEMENTATION
// ============================================================================

// TGIEmbedder uses Hugging Face Text Embeddings Inference (TGI) for embeddings
type TGIEmbedder struct {
	config TGIEmbedderConfig
	client *http.Client
}

// NewTGIEmbedderFromProviderConfig creates a new TGI embedder from provider config
func NewTGIEmbedderFromProviderConfig(config *TGIEmbedderProviderConfig) (*TGIEmbedder, error) {
	provider, err := config.CreateProvider()
	if err != nil {
		return nil, err
	}
	return provider.(*TGIEmbedder), nil
}

// DefaultTGIConfig returns a default TGI configuration
func DefaultTGIConfig() TGIEmbedderConfig {
	return TGIEmbedderConfig{
		BaseURL:      "http://localhost:8080",
		Timeout:      30 * time.Second,
		ModelID:      "BAAI/bge-small-en-v1.5",
		Dimension:    384,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    512,
	}
}

// NewTGIEmbedder creates a new TGI embedder with default configuration
func NewTGIEmbedder() *TGIEmbedder {
	config := DefaultTGIConfig()
	return &TGIEmbedder{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// NewTGIEmbedderWithConfig creates a new TGI embedder with custom configuration
func NewTGIEmbedderWithConfig(config TGIEmbedderConfig) *TGIEmbedder {
	// Set defaults for missing values
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:8080"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.ModelID == "" {
		config.ModelID = "BAAI/bge-small-en-v1.5"
	}
	if config.Dimension == 0 {
		config.Dimension = 384
	}

	return &TGIEmbedder{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// NewTGIEmbedderWithModel creates a new TGI embedder with a specific model
func NewTGIEmbedderWithModel(modelID string, dimension int) *TGIEmbedder {
	config := DefaultTGIConfig()
	config.ModelID = modelID
	config.Dimension = dimension

	return &TGIEmbedder{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Embed generates embeddings for the given text using TGI
func (e *TGIEmbedder) Embed(text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("input text cannot be empty")
	}

	// Prepare the request payload
	payload := map[string]interface{}{
		"inputs": text,
		"parameters": map[string]interface{}{
			"wait_for_model": e.config.WaitForModel,
			"normalize":      e.config.Normalize,
			"truncate":       e.config.Truncate,
			"max_length":     e.config.MaxLength,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make the HTTP request
	url := fmt.Sprintf("%s/embed", e.config.BaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TGI server returned status %d", resp.StatusCode)
	}

	// Parse the response
	var response struct {
		Embeddings [][]float32 `json:"embeddings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return response.Embeddings[0], nil
}

// GetDimension returns the embedding dimension
func (e *TGIEmbedder) GetDimension() int {
	return e.config.Dimension
}

// GetModelName returns the model name
func (e *TGIEmbedder) GetModelName() string {
	return e.config.ModelID
}

// HealthCheck verifies that the TGI service is available
func (e *TGIEmbedder) HealthCheck(ctx context.Context) error {
	// Try to make a simple request to check if TGI is running
	url := fmt.Sprintf("%s/health", e.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("TGI service is not available: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("TGI service returned status %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// TGI BUILDER WITH CHAINED OPTIONS
// ============================================================================

// TGIBuilder provides a fluent interface for building TGI embedders
type TGIBuilder struct {
	config TGIEmbedderConfig
}

// NewTGIBuilder creates a new TGI builder with sane defaults
func NewTGIBuilder() *TGIBuilder {
	return &TGIBuilder{
		config: DefaultTGIConfig(),
	}
}

// WithBaseURL sets the TGI server base URL
func (b *TGIBuilder) WithBaseURL(baseURL string) *TGIBuilder {
	b.config.BaseURL = baseURL
	return b
}

// WithTimeout sets the HTTP timeout for TGI requests
func (b *TGIBuilder) WithTimeout(timeout time.Duration) *TGIBuilder {
	b.config.Timeout = timeout
	return b
}

// WithModel sets the model ID and dimension
func (b *TGIBuilder) WithModel(modelID string, dimension int) *TGIBuilder {
	b.config.ModelID = modelID
	b.config.Dimension = dimension
	return b
}

// WithPresetModel sets a predefined model configuration
func (b *TGIBuilder) WithPresetModel(preset TGIEmbedderConfig) *TGIBuilder {
	b.config = preset
	return b
}

// WithWaitForModel enables/disables waiting for model to be ready
func (b *TGIBuilder) WithWaitForModel(wait bool) *TGIBuilder {
	b.config.WaitForModel = wait
	return b
}

// WithNormalize enables/disables vector normalization
func (b *TGIBuilder) WithNormalize(normalize bool) *TGIBuilder {
	b.config.Normalize = normalize
	return b
}

// WithTruncate enables/disables text truncation
func (b *TGIBuilder) WithTruncate(truncate bool) *TGIBuilder {
	b.config.Truncate = truncate
	return b
}

// WithMaxLength sets the maximum text length
func (b *TGIBuilder) WithMaxLength(maxLength int) *TGIBuilder {
	b.config.MaxLength = maxLength
	return b
}

// Build creates the TGI embedder with the configured options
func (b *TGIBuilder) Build() *TGIEmbedder {
	return &TGIEmbedder{
		config: b.config,
		client: &http.Client{
			Timeout: b.config.Timeout,
		},
	}
}

// ============================================================================
// TGI MODEL CONFIGURATIONS
// ============================================================================

// Predefined TGI model configurations
var (
	// BGE Models (Most Popular - Beijing Academy of AI)
	BGESmallEnV15 = TGIEmbedderConfig{
		ModelID:      "BAAI/bge-small-en-v1.5",
		Dimension:    384,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    512,
	}

	BGELargeEnV15 = TGIEmbedderConfig{
		ModelID:      "BAAI/bge-large-en-v1.5",
		Dimension:    1024,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    512,
	}

	BGEBaseEnV15 = TGIEmbedderConfig{
		ModelID:      "BAAI/bge-base-en-v1.5",
		Dimension:    768,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    512,
	}

	BGEM3 = TGIEmbedderConfig{
		ModelID:      "BAAI/bge-m3",
		Dimension:    1024,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    8192,
	}

	// Sentence Transformers Models (Very Popular)
	AllMiniLML6V2 = TGIEmbedderConfig{
		ModelID:      "sentence-transformers/all-MiniLM-L6-v2",
		Dimension:    384,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    256,
	}

	AllMiniLML12V2 = TGIEmbedderConfig{
		ModelID:      "sentence-transformers/all-MiniLM-L12-v2",
		Dimension:    384,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    256,
	}

	AllMpnetBaseV2 = TGIEmbedderConfig{
		ModelID:      "sentence-transformers/all-mpnet-base-v2",
		Dimension:    768,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    384,
	}

	// E5 Models (Microsoft - Excellent for Retrieval)
	E5SmallV2 = TGIEmbedderConfig{
		ModelID:      "intfloat/e5-small-v2",
		Dimension:    384,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    512,
	}

	E5BaseV2 = TGIEmbedderConfig{
		ModelID:      "intfloat/e5-base-v2",
		Dimension:    768,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    512,
	}

	E5LargeV2 = TGIEmbedderConfig{
		ModelID:      "intfloat/e5-large-v2",
		Dimension:    1024,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    512,
	}

	// Other Popular Models
	CohereEmbedV3 = TGIEmbedderConfig{
		ModelID:      "Cohere/embed-english-v3.0",
		Dimension:    1024,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    512,
	}

	AlibabaGTE = TGIEmbedderConfig{
		ModelID:      "Alibaba-NLP/gte-base-en-v1.5",
		Dimension:    768,
		WaitForModel: true,
		Normalize:    true,
		Truncate:     true,
		MaxLength:    512,
	}
)
