// Package embedder provides text embedding services for semantic search.
//
// Package embedder provides text embedding generation for vector search.
package embedder

import (
	"context"
)

// Embedder produces vector embeddings from text.
//
// Embeddings are used by IndexService for semantic similarity search.
// Different providers (OpenAI, Ollama) implement this interface.
type Embedder interface {
	// Embed converts text to a vector embedding.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch converts multiple texts to vector embeddings.
	// More efficient than calling Embed multiple times.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension returns the embedding vector dimension.
	Dimension() int

	// Model returns the model name being used.
	Model() string

	// Close releases any resources held by the embedder.
	Close() error
}
