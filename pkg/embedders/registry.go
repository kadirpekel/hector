package embedders

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/registry"
)

type EmbedderProvider interface {
	Embed(text string) ([]float32, error)

	GetDimension() int

	GetModelName() string

	Close() error
}

type EmbedderRegistry struct {
	*registry.BaseRegistry[EmbedderProvider]
}

func NewEmbedderRegistry() *EmbedderRegistry {
	return &EmbedderRegistry{
		BaseRegistry: registry.NewBaseRegistry[EmbedderProvider](),
	}
}

func (r *EmbedderRegistry) RegisterEmbedder(name string, provider EmbedderProvider) error {
	if name == "" {
		return fmt.Errorf("embedder name cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("embedder provider cannot be nil")
	}
	return r.Register(name, provider)
}

func (r *EmbedderRegistry) CreateEmbedderFromConfig(name string, config *config.EmbedderProviderConfig) (EmbedderProvider, error) {
	if name == "" {
		return nil, fmt.Errorf("embedder name cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("embedder config cannot be nil")
	}

	var provider EmbedderProvider
	var err error

	switch config.Type {
	case "ollama":
		provider, err = NewOllamaEmbedderFromConfig(config)
	case "openai":
		provider, err = NewOpenAIEmbedderFromConfig(config)
	case "cohere":
		provider, err = NewCohereEmbedderFromConfig(config)
	default:
		return nil, fmt.Errorf("unsupported embedder type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create embedder provider: %w", err)
	}

	if err := r.RegisterEmbedder(name, provider); err != nil {
		return nil, fmt.Errorf("failed to register embedder: %w", err)
	}

	return provider, nil
}

func (r *EmbedderRegistry) GetEmbedder(name string) (EmbedderProvider, error) {
	provider, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("embedder provider '%s' not found", name)
	}
	return provider, nil
}

func (r *EmbedderRegistry) ListEmbedders() []string {
	names := make([]string, 0)
	for _, provider := range r.List() {
		names = append(names, provider.GetModelName())
	}
	return names
}
