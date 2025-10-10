package memory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/llms"
)

// VectorMemoryStrategy stores conversation messages in vector database for semantic recall
// Uses direct database + embedder access (not SearchEngine which is for RAG)
type VectorMemoryStrategy struct {
	db         databases.DatabaseProvider
	embedder   embedders.EmbedderProvider
	collection string
}

// NewVectorMemoryStrategy creates a new vector memory strategy
func NewVectorMemoryStrategy(
	db databases.DatabaseProvider,
	embedder embedders.EmbedderProvider,
	collection string,
) (*VectorMemoryStrategy, error) {
	if db == nil {
		return nil, fmt.Errorf("database provider is required for vector memory")
	}
	if embedder == nil {
		return nil, fmt.Errorf("embedder provider is required for vector memory")
	}
	if collection == "" {
		collection = "hector_session_memory"
	}

	return &VectorMemoryStrategy{
		db:         db,
		embedder:   embedder,
		collection: collection,
	}, nil
}

// Name returns the strategy identifier
func (v *VectorMemoryStrategy) Name() string {
	return "vector_memory"
}

// Store adds messages to long-term memory (batch operation)
// Each message is stored as a separate vector document in Qdrant
func (v *VectorMemoryStrategy) Store(sessionID string, messages []llms.Message) error {
	if len(messages) == 0 {
		return nil
	}

	for i, msg := range messages {
		// Skip messages with no content
		if msg.Content == "" {
			continue
		}

		// Generate embedding
		vector, err := v.embedder.Embed(msg.Content)
		if err != nil {
			return fmt.Errorf("failed to embed message %d: %w", i, err)
		}

		// Create unique document ID using UUID
		docID := uuid.New().String()

		// Prepare metadata
		metadata := map[string]interface{}{
			"session_id":    sessionID,
			"role":          msg.Role,
			"content":       msg.Content, // Store content for retrieval
			"message_index": i,
		}

		// Store directly to database
		ctx := context.Background()
		if err := v.db.Upsert(ctx, v.collection, docID, vector, metadata); err != nil {
			return fmt.Errorf("failed to store message %d: %w", i, err)
		}
	}

	return nil
}

// Recall retrieves relevant context from long-term memory using semantic search
// Filters by sessionID to ensure session isolation
func (v *VectorMemoryStrategy) Recall(sessionID string, query string, limit int) ([]llms.Message, error) {
	if query == "" {
		return []llms.Message{}, nil
	}

	// Generate query embedding
	queryVector, err := v.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Search with session filter
	ctx := context.Background()
	results, err := v.db.SearchWithFilter(ctx, v.collection, queryVector, limit, map[string]interface{}{
		"session_id": sessionID, // Session isolation
	})
	if err != nil {
		return nil, fmt.Errorf("recall failed: %w", err)
	}

	// Convert search results to messages
	messages := make([]llms.Message, 0, len(results))
	for _, result := range results {
		// Extract role from metadata
		role, ok := result.Metadata["role"].(string)
		if !ok {
			role = "assistant" // Default fallback
		}

		// Extract content from metadata
		content, ok := result.Metadata["content"].(string)
		if !ok {
			content = result.Content // Fallback to result content
		}

		messages = append(messages, llms.Message{
			Role:    role,
			Content: content,
		})
	}

	return messages, nil
}

// Clear removes all long-term memory for a session
// Deletes all vectors associated with the session from Qdrant
func (v *VectorMemoryStrategy) Clear(sessionID string) error {
	// Delete all points where session_id = sessionID
	ctx := context.Background()
	return v.db.DeleteByFilter(ctx, v.collection, map[string]interface{}{
		"session_id": sessionID,
	})
}
