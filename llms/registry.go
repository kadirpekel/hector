package llms

import (
	"fmt"
	"sync"

	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/registry"
)

// ============================================================================
// LLM REGISTRY
// ============================================================================

// LLMProvider interface for language model generation
type LLMProvider interface {
	// Generate generates a response given a pre-built prompt
	Generate(prompt string) (string, int, error)

	// GenerateStreaming generates a streaming response given a pre-built prompt
	GenerateStreaming(prompt string) (<-chan string, error)

	// GetModelName returns the model name
	GetModelName() string

	// GetMaxTokens returns the maximum tokens for generation
	GetMaxTokens() int

	// GetTemperature returns the temperature setting
	GetTemperature() float64

	// Close closes the provider and releases resources
	Close() error
}

// LLMRegistry manages LLM provider instances
type LLMRegistry struct {
	*registry.BaseRegistry[LLMProvider]
	mu sync.RWMutex
}

// NewLLMRegistry creates a new LLM registry
func NewLLMRegistry() *LLMRegistry {
	return &LLMRegistry{
		BaseRegistry: registry.NewBaseRegistry[LLMProvider](),
	}
}

// RegisterLLM registers an LLM provider instance
func (r *LLMRegistry) RegisterLLM(name string, provider LLMProvider) error {
	if name == "" {
		return fmt.Errorf("LLM name cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("LLM provider cannot be nil")
	}
	return r.Register(name, provider)
}

// CreateLLMFromConfig creates an LLM provider from configuration
func (r *LLMRegistry) CreateLLMFromConfig(name string, config *config.LLMProviderConfig) (LLMProvider, error) {
	if name == "" {
		return nil, fmt.Errorf("LLM name cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("LLM config cannot be nil")
	}

	// Set defaults and validate
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid LLM config: %w", err)
	}

	var provider LLMProvider
	var err error

	switch config.Type {
	case "ollama":
		provider, err = NewOllamaProviderFromConfig(config)
	case "openai":
		provider, err = NewOpenAIProviderFromConfig(config)
	default:
		return nil, fmt.Errorf("unsupported LLM type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Register the provider
	if err := r.RegisterLLM(name, provider); err != nil {
		return nil, fmt.Errorf("failed to register LLM: %w", err)
	}

	return provider, nil
}

// GetLLM retrieves an LLM provider by name
func (r *LLMRegistry) GetLLM(name string) (LLMProvider, error) {
	provider, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("LLM provider '%s' not found", name)
	}
	return provider, nil
}

// ListLLMs returns all registered LLM names
func (r *LLMRegistry) ListLLMs() []string {
	names := make([]string, 0)
	for _, provider := range r.List() {
		names = append(names, provider.GetModelName())
	}
	return names
}
