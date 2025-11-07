package llms

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/registry"
)

type LLMProvider interface {
	Generate(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (text string, toolCalls []*protocol.ToolCall, tokens int, err error)

	GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition) (<-chan StreamChunk, error)

	GetModelName() string

	GetMaxTokens() int

	GetTemperature() float64

	Close() error
}

type StructuredOutputProvider interface {
	LLMProvider

	GenerateStructured(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, config *StructuredOutputConfig) (text string, toolCalls []*protocol.ToolCall, tokens int, err error)

	GenerateStructuredStreaming(ctx context.Context, messages []*pb.Message, tools []ToolDefinition, config *StructuredOutputConfig) (<-chan StreamChunk, error)

	SupportsStructuredOutput() bool
}

type LLMRegistry struct {
	*registry.BaseRegistry[LLMProvider]
}

func NewLLMRegistry() *LLMRegistry {
	return &LLMRegistry{
		BaseRegistry: registry.NewBaseRegistry[LLMProvider](),
	}
}

func (r *LLMRegistry) RegisterLLM(name string, provider LLMProvider) error {
	if name == "" {
		return fmt.Errorf("LLM name cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("LLM provider cannot be nil")
	}
	return r.Register(name, provider)
}

func (r *LLMRegistry) CreateLLMFromConfig(name string, config *config.LLMProviderConfig) (LLMProvider, error) {
	if name == "" {
		return nil, fmt.Errorf("LLM name cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("LLM config cannot be nil")
	}

	var provider LLMProvider
	var err error

	switch config.Type {
	case "openai":
		provider, err = NewOpenAIProviderFromConfig(config)
	case "anthropic":
		provider, err = NewAnthropicProviderFromConfig(config)
	case "gemini":
		provider, err = NewGeminiProviderFromConfig(config)
	default:
		return nil, fmt.Errorf("unsupported LLM type: %s (supported: openai, anthropic, gemini)", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	if err := r.RegisterLLM(name, provider); err != nil {
		return nil, fmt.Errorf("failed to register LLM: %w", err)
	}

	return provider, nil
}

func (r *LLMRegistry) GetLLM(name string) (LLMProvider, error) {
	provider, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("LLM provider '%s' not found", name)
	}
	return provider, nil
}

func (r *LLMRegistry) ListLLMs() []string {
	names := make([]string, 0)
	for _, provider := range r.List() {
		names = append(names, provider.GetModelName())
	}
	return names
}
