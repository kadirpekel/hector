package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/embedders"
)

// EmbedderBuilder provides a fluent API for building embedder providers
type EmbedderBuilder struct {
	embedderType string
	model        string
	host         string
	apiKey       string
	dimension    int
	timeout      int
	maxRetries   int
	batchSize    int
}

// NewEmbedder creates a new embedder builder
func NewEmbedder(embedderType string) *EmbedderBuilder {
	builder := &EmbedderBuilder{
		embedderType: embedderType,
		timeout:      30,
		maxRetries:   3,
	}

	// Set defaults based on type
	switch embedderType {
	case "ollama":
		builder.host = "http://localhost:11434"
		builder.model = "nomic-embed-text"
		builder.dimension = 768
	case "openai":
		builder.host = "https://api.openai.com/v1"
		builder.model = "text-embedding-3-small"
		builder.dimension = 1536
	case "cohere":
		builder.host = "https://api.cohere.ai/v1"
		builder.model = "embed-english-v3.0"
		builder.dimension = 1024
	default:
		builder.host = "http://localhost:11434"
		builder.model = "nomic-embed-text"
		builder.dimension = 768
	}

	return builder
}

// Model sets the model name
func (b *EmbedderBuilder) Model(model string) *EmbedderBuilder {
	b.model = model
	return b
}

// Host sets the API host
func (b *EmbedderBuilder) Host(host string) *EmbedderBuilder {
	b.host = host
	return b
}

// APIKey sets the API key
func (b *EmbedderBuilder) APIKey(key string) *EmbedderBuilder {
	b.apiKey = key
	return b
}

// Dimension sets the embedding dimension
func (b *EmbedderBuilder) Dimension(dim int) *EmbedderBuilder {
	if dim <= 0 {
		panic("dimension must be positive")
	}
	b.dimension = dim
	return b
}

// Timeout sets the timeout in seconds
func (b *EmbedderBuilder) Timeout(seconds int) *EmbedderBuilder {
	if seconds < 0 {
		panic("timeout must be non-negative")
	}
	b.timeout = seconds
	return b
}

// MaxRetries sets the maximum retries
func (b *EmbedderBuilder) MaxRetries(max int) *EmbedderBuilder {
	if max < 0 {
		panic("max retries must be non-negative")
	}
	b.maxRetries = max
	return b
}

// BatchSize sets the batch size
func (b *EmbedderBuilder) BatchSize(size int) *EmbedderBuilder {
	b.batchSize = size
	return b
}

// Build creates the embedder provider
func (b *EmbedderBuilder) Build() (embedders.EmbedderProvider, error) {
	if b.model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if b.host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if b.dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}

	cfg := &config.EmbedderProviderConfig{
		Type:       b.embedderType,
		Model:      b.model,
		Host:       b.host,
		APIKey:     b.apiKey,
		Dimension:  b.dimension,
		Timeout:    b.timeout,
		MaxRetries: b.maxRetries,
		BatchSize:  b.batchSize,
	}

	switch b.embedderType {
	case "ollama":
		return embedders.NewOllamaEmbedderFromConfig(cfg)
	case "openai":
		return embedders.NewOpenAIEmbedderFromConfig(cfg)
	case "cohere":
		return embedders.NewCohereEmbedderFromConfig(cfg)
	default:
		return nil, fmt.Errorf("unknown embedder type: %s (supported: 'ollama', 'openai', 'cohere')", b.embedderType)
	}
}
