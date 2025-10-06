package embedders

import (
	"fmt"
	"sync"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/registry"
)

// ============================================================================
// EMBEDDER INTERFACE
// ============================================================================

// EmbedderProvider interface for embedding generation
type EmbedderProvider interface {
	// Embed generates embeddings for the given text
	Embed(text string) ([]float32, error)

	// GetDimension returns the dimension of the embedding vectors
	GetDimension() int

	// GetModelName returns the model name used for embeddings
	GetModelName() string

	// Close closes the embedder provider and releases resources
	Close() error
}

// ============================================================================
// EMBEDDER REGISTRY
// ============================================================================

// EmbedderRegistry manages embedder provider instances
type EmbedderRegistry struct {
	*registry.BaseRegistry[EmbedderProvider]
	mu sync.RWMutex
}

// NewEmbedderRegistry creates a new embedder registry
func NewEmbedderRegistry() *EmbedderRegistry {
	return &EmbedderRegistry{
		BaseRegistry: registry.NewBaseRegistry[EmbedderProvider](),
	}
}

// RegisterEmbedder registers an embedder provider instance
func (r *EmbedderRegistry) RegisterEmbedder(name string, provider EmbedderProvider) error {
	if name == "" {
		return fmt.Errorf("embedder name cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("embedder provider cannot be nil")
	}
	return r.Register(name, provider)
}

// CreateEmbedderFromConfig creates an embedder provider from configuration
func (r *EmbedderRegistry) CreateEmbedderFromConfig(name string, config *config.EmbedderProviderConfig) (EmbedderProvider, error) {
	if name == "" {
		return nil, fmt.Errorf("embedder name cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("embedder config cannot be nil")
	}

	// Set defaults and validate
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid embedder config: %w", err)
	}

	var provider EmbedderProvider
	var err error

	switch config.Type {
	case "ollama":
		provider, err = NewOllamaEmbedderFromConfig(config)
	default:
		return nil, fmt.Errorf("unsupported embedder type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create embedder provider: %w", err)
	}

	// Register the provider
	if err := r.RegisterEmbedder(name, provider); err != nil {
		return nil, fmt.Errorf("failed to register embedder: %w", err)
	}

	return provider, nil
}

// GetEmbedder retrieves an embedder provider by name
func (r *EmbedderRegistry) GetEmbedder(name string) (EmbedderProvider, error) {
	provider, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("embedder provider '%s' not found", name)
	}
	return provider, nil
}

// ListEmbedders returns all registered embedder names
func (r *EmbedderRegistry) ListEmbedders() []string {
	names := make([]string, 0)
	for _, provider := range r.List() {
		names = append(names, provider.GetModelName())
	}
	return names
}
