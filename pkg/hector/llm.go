package hector

import (
	"fmt"
	"os"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/llms"
)

// LLMProviderBuilder provides a fluent API for building LLM providers
type LLMProviderBuilder struct {
	providerType     string
	model            string
	apiKey           string
	host             string
	temperature      float64
	maxTokens        int
	timeout          int
	maxRetries       int
	retryDelay       int
	structuredOutput *config.StructuredOutputConfig
}

// NewLLMProvider creates a new LLM provider builder
func NewLLMProvider(providerType string) *LLMProviderBuilder {
	builder := &LLMProviderBuilder{
		providerType: providerType,
		temperature:  0.7,
		maxTokens:    8000,
		timeout:      600, // Default: 10 minutes
		maxRetries:   5,
		retryDelay:   2,
	}

	// Set defaults based on provider type
	switch providerType {
	case "openai":
		builder.host = "https://api.openai.com/v1"
		builder.model = config.DefaultOpenAIModel
	case "anthropic":
		builder.host = "https://api.anthropic.com"
		builder.model = config.DefaultAnthropicModel
	case "gemini":
		builder.host = "https://generativelanguage.googleapis.com"
		builder.model = config.DefaultGeminiModel
	case "ollama":
		builder.host = "http://localhost:11434"
		builder.model = "llama3.2"
	default:
		builder.host = "https://api.openai.com/v1"
		builder.model = config.DefaultOpenAIModel
	}

	return builder
}

// Model sets the model name
func (b *LLMProviderBuilder) Model(model string) *LLMProviderBuilder {
	b.model = model
	return b
}

// APIKey sets the API key (can also use environment variable)
func (b *LLMProviderBuilder) APIKey(key string) *LLMProviderBuilder {
	b.apiKey = key
	return b
}

// APIKeyFromEnv sets the API key from an environment variable
func (b *LLMProviderBuilder) APIKeyFromEnv(envVar string) *LLMProviderBuilder {
	b.apiKey = os.Getenv(envVar)
	return b
}

// Host sets the API host
func (b *LLMProviderBuilder) Host(host string) *LLMProviderBuilder {
	b.host = host
	return b
}

// Temperature sets the temperature
func (b *LLMProviderBuilder) Temperature(temp float64) *LLMProviderBuilder {
	if temp < 0 || temp > 2 {
		panic("temperature must be between 0 and 2")
	}
	b.temperature = temp
	return b
}

// MaxTokens sets the maximum tokens
func (b *LLMProviderBuilder) MaxTokens(max int) *LLMProviderBuilder {
	if max < 0 {
		panic("max tokens must be non-negative")
	}
	b.maxTokens = max
	return b
}

// Timeout sets the timeout in seconds
func (b *LLMProviderBuilder) Timeout(seconds int) *LLMProviderBuilder {
	if seconds < 0 {
		panic("timeout must be non-negative")
	}
	b.timeout = seconds
	return b
}

// MaxRetries sets the maximum retries
func (b *LLMProviderBuilder) MaxRetries(max int) *LLMProviderBuilder {
	if max < 0 {
		panic("max retries must be non-negative")
	}
	b.maxRetries = max
	return b
}

// RetryDelay sets the retry delay in seconds
func (b *LLMProviderBuilder) RetryDelay(seconds int) *LLMProviderBuilder {
	if seconds < 0 {
		panic("retry delay must be non-negative")
	}
	b.retryDelay = seconds
	return b
}

// StructuredOutput sets structured output configuration
func (b *LLMProviderBuilder) StructuredOutput(cfg *config.StructuredOutputConfig) *LLMProviderBuilder {
	b.structuredOutput = cfg
	return b
}

// WithStructuredOutput sets structured output from a builder
func (b *LLMProviderBuilder) WithStructuredOutput(builder *StructuredOutputBuilder) *LLMProviderBuilder {
	if builder != nil {
		b.structuredOutput = builder.Build()
	}
	return b
}

// Build creates the LLM provider
func (b *LLMProviderBuilder) Build() (llms.LLMProvider, error) {
	if b.model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if b.host == "" {
		return nil, fmt.Errorf("host is required")
	}

	// Try to get API key from environment if not set
	if b.apiKey == "" {
		switch b.providerType {
		case "openai":
			b.apiKey = os.Getenv("OPENAI_API_KEY")
		case "anthropic":
			b.apiKey = os.Getenv("ANTHROPIC_API_KEY")
		case "gemini":
			b.apiKey = os.Getenv("GEMINI_API_KEY")
		case "ollama":
			// Ollama doesn't require API key
		}
	}

	llmConfig := &config.LLMProviderConfig{
		Type:        b.providerType,
		Model:       b.model,
		APIKey:      b.apiKey,
		Host:        b.host,
		Temperature: &b.temperature,
		MaxTokens:   b.maxTokens,
		Timeout:     b.timeout,
		MaxRetries:  b.maxRetries,
		RetryDelay:  b.retryDelay,
	}

	if b.structuredOutput != nil {
		llmConfig.StructuredOutput = b.structuredOutput
	}

	switch b.providerType {
	case "openai":
		return llms.NewOpenAIProviderFromConfig(llmConfig)
	case "anthropic":
		return llms.NewAnthropicProviderFromConfig(llmConfig)
	case "gemini":
		return llms.NewGeminiProviderFromConfig(llmConfig)
	case "ollama":
		return llms.NewOllamaProviderFromConfig(llmConfig)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", b.providerType)
	}
}
