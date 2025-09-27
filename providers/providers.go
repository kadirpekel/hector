package providers

import (
	"context"
	"fmt"
	"time"
)

// ============================================================================
// CONSTANTS AND ERROR TYPES
// ============================================================================

const (
	// DefaultProviderTimeout is the default timeout for provider operations
	DefaultProviderTimeout = 30 * time.Second

	// DefaultMaxRetries is the default number of retries for provider operations
	DefaultMaxRetries = 3

	// DefaultRetryDelay is the default delay between retries
	DefaultRetryDelay = 1 * time.Second
)

// ProviderError represents errors in the provider system
type ProviderError struct {
	Type      ProviderType
	Name      string
	Operation string
	Message   string
	Err       error
	Timestamp time.Time
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s:%s] %s: %v", e.Type, e.Name, e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s:%s] %s", e.Type, e.Name, e.Operation, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new provider error
func NewProviderError(providerType ProviderType, name, operation, message string, err error) *ProviderError {
	return &ProviderError{
		Type:      providerType,
		Name:      name,
		Operation: operation,
		Message:   message,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// ============================================================================
// CORE PROVIDER INTERFACES
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

	// IsHealthy checks if the provider is healthy and ready to use
	IsHealthy(ctx context.Context) bool

	// Close closes the provider and releases resources
	Close() error
}

// DatabaseProvider defines the interface for vector database operations
type DatabaseProvider interface {
	// Upsert adds or updates a document in the database
	Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error

	// Search performs vector similarity search
	Search(ctx context.Context, collection string, vector []float32, topK int) ([]SearchResult, error)

	// Delete removes a document from the database
	Delete(ctx context.Context, collection string, id string) error

	// CreateCollection creates a new collection
	CreateCollection(ctx context.Context, collection string, vectorSize uint64) error

	// DeleteCollection removes a collection
	DeleteCollection(ctx context.Context, collection string) error

	// IsHealthy checks if the database provider is healthy and ready to use
	IsHealthy(ctx context.Context) bool

	// Close closes the database provider and releases resources
	Close() error
}

// EmbedderProvider interface for embedding generation
type EmbedderProvider interface {
	// Embed generates embeddings for the given text
	Embed(text string) ([]float32, error)

	// GetDimension returns the dimension of the embedding vectors
	GetDimension() int

	// GetModelName returns the model name used for embeddings
	GetModelName() string

	// IsHealthy checks if the embedder provider is healthy and ready to use
	IsHealthy(ctx context.Context) bool

	// Close closes the embedder provider and releases resources
	Close() error
}

// SearchResult represents a search result from the vector database
type SearchResult struct {
	ID        string                 `json:"id"`
	Score     float32                `json:"score"`
	Content   string                 `json:"content"`
	Vector    []float32              `json:"vector,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
	ModelName string                 `json:"model_name,omitempty"`
}

// ============================================================================
// PROVIDER CONFIGURATION INTERFACES
// ============================================================================

// ProviderType represents the type of provider
type ProviderType string

const (
	ProviderTypeLLM      ProviderType = "llm"
	ProviderTypeDatabase ProviderType = "database"
	ProviderTypeEmbedder ProviderType = "embedder"
)

// ProviderConfig represents a configuration that can create a provider
type ProviderConfig interface {
	// GetProviderType returns the type of provider this config creates
	GetProviderType() ProviderType

	// GetProviderName returns the unique name/identifier for this provider
	GetProviderName() string

	// Validate validates the configuration
	Validate() error

	// SetDefaults sets default values for any unset fields
	SetDefaults()
}

// ProviderFactory defines the interface for creating provider instances
type ProviderFactory interface {
	// CreateProvider creates a new provider instance from the configuration
	CreateProvider(config ProviderConfig) (interface{}, error)

	// GetSupportedType returns the provider type this factory supports
	GetSupportedType() ProviderType

	// GetSupportedNames returns the list of provider names this factory supports
	GetSupportedNames() []string
}

// ============================================================================
// PROVIDER PACKAGE OVERVIEW
// ============================================================================

// The providers package provides core interfaces and types for the provider system.
// The registry functionality is implemented in registry.go following the same
// pattern as tools/registry.go and executors/registry.go.
//
// Core Components:
// - Provider interfaces (LLMProvider, DatabaseProvider, EmbedderProvider)
// - Configuration interfaces (ProviderConfig, ProviderFactory)
// - Error types (ProviderError)
// - Registry implementation (in registry.go)
//
// Example usage:
//   config := &MyProviderConfig{...}
//   providers.RegisterProvider(config)
//   provider, err := providers.CreateProvider(providers.ProviderTypeLLM, "my-provider")
//
// This approach provides clean separation of concerns and better maintainability.
