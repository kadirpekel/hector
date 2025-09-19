package embedders

// ============================================================================
// EMBEDDING PROVIDER INTERFACE
// ============================================================================

// EmbeddingProvider interface for embedding generation
type EmbeddingProvider interface {
	Embed(text string) ([]float32, error)
	GetDimension() int
	GetModelName() string
}

// ============================================================================
// CONVENIENT REEXPORTS
// ============================================================================

// This file provides convenient reexports for all embedder implementations.
// All embedder types and functions are available directly from the embedders package.
