package interfaces

// ============================================================================
// PROVIDER INTERFACES
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

	// CreateProvider creates a new provider instance from this config
	CreateProvider() (interface{}, error)

	// Validate validates the configuration
	Validate() error
}
