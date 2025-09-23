package hector

// ============================================================================
// MODEL CONFIGURATION
// ============================================================================

// ModelConfig holds configuration for a simple document model
type ModelConfig struct {
	Name        string `yaml:"name"`          // Model name (e.g., "document", "article")
	Collection  string `yaml:"collection"`    // Vector collection name
	DefaultTopK int    `yaml:"default_top_k"` // Default search results
	MaxTopK     int    `yaml:"max_top_k"`     // Maximum search results
}
