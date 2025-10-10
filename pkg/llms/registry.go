package llms

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/registry"
)

// ============================================================================
// LLM REGISTRY
// ============================================================================

// LLMProvider interface for language model generation with native function calling
// All providers must support native function calling
type LLMProvider interface {
	// Generate generates a response with native function calling support
	// Takes a conversation history as an array of messages (proper multi-turn support)
	// Returns: text content, tool calls, tokens used, error
	// - text: The LLM's text response (may be empty if only tool calls)
	// - toolCalls: Structured tool calls from the LLM (may be empty if only text)
	// - tokens: Total tokens used
	Generate(messages []Message, tools []ToolDefinition) (text string, toolCalls []ToolCall, tokens int, err error)

	// GenerateStreaming generates a streaming response with function calling
	// Takes a conversation history as an array of messages (proper multi-turn support)
	// Returns a channel that streams text chunks and eventually tool calls
	GenerateStreaming(messages []Message, tools []ToolDefinition) (<-chan StreamChunk, error)

	// GetModelName returns the model name
	GetModelName() string

	// GetMaxTokens returns the maximum tokens for generation
	GetMaxTokens() int

	// GetTemperature returns the temperature setting
	GetTemperature() float64

	// Close closes the provider and releases resources
	Close() error
}

// StructuredOutputProvider is an optional interface for providers that support structured output
// Providers can implement this to enable structured output features (JSON, XML, enum)
type StructuredOutputProvider interface {
	LLMProvider

	// GenerateStructured generates a response with structured output
	// config specifies the desired output format (JSON schema, enum, etc.)
	GenerateStructured(messages []Message, tools []ToolDefinition, config *StructuredOutputConfig) (text string, toolCalls []ToolCall, tokens int, err error)

	// GenerateStructuredStreaming generates a streaming response with structured output
	GenerateStructuredStreaming(messages []Message, tools []ToolDefinition, config *StructuredOutputConfig) (<-chan StreamChunk, error)

	// SupportsStructuredOutput returns true if the provider supports structured output
	SupportsStructuredOutput() bool
}

// LLMRegistry manages LLM provider instances
type LLMRegistry struct {
	*registry.BaseRegistry[LLMProvider]
	// mu sync.RWMutex // Reserved for future use
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
	case "openai":
		provider, err = NewOpenAIProviderFromConfig(config)
	case "ollama":
		// Ollama uses OpenAI-compatible API
		provider, err = NewOpenAIProviderFromConfig(config)
	case "anthropic":
		provider, err = NewAnthropicProviderFromConfig(config)
	case "gemini":
		provider, err = NewGeminiProviderFromConfig(config)
	default:
		return nil, fmt.Errorf("unsupported LLM type: %s (supported: openai, ollama, anthropic, gemini)", config.Type)
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
