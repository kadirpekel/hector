package databases

import (
	"context"
)

// ============================================================================
// VECTOR DATABASE INTERFACE
// ============================================================================

// VectorDB defines the interface for vector database operations
type VectorDB interface {
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
// CONVENIENT REEXPORTS
// ============================================================================

// This file provides convenient reexports for all database implementations.
// All database types and functions are available directly from the databases package.

// Reexport Qdrant types and functions (legacy)
type QdrantOption = qdrantOption

// Reexport Qdrant functions (legacy)
var (
	WithHost        = withHost
	WithPort        = withPort
	WithAPIKey      = withAPIKey
	WithTimeout     = withTimeout
	WithTLS         = withTLS
	WithInsecureTLS = withInsecureTLS
)
