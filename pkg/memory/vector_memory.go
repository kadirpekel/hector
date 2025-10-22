package memory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/protocol"
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
// Isolation: agentID + sessionID (prevents cross-agent memory leaks)
func (v *VectorMemoryStrategy) Store(agentID string, sessionID string, messages []*pb.Message) error {
	if len(messages) == 0 {
		return nil
	}

	for i, msg := range messages {
		// Skip messages with no content
		textContent := protocol.ExtractTextFromMessage(msg)
		if textContent == "" {
			continue
		}

		// Generate embedding
		vector, err := v.embedder.Embed(textContent)
		if err != nil {
			return fmt.Errorf("failed to embed message %d: %w", i, err)
		}

		// Create unique document ID using UUID
		docID := uuid.New().String()

		// Convert pb.Role enum to string
		roleStr := "unknown"
		switch msg.Role {
		case pb.Role_ROLE_USER:
			roleStr = "user"
		case pb.Role_ROLE_AGENT:
			roleStr = "agent"
		case pb.Role_ROLE_UNSPECIFIED:
			roleStr = "system"
		}

		// Prepare metadata with BOTH agent_id AND session_id for proper isolation
		metadata := map[string]interface{}{
			"agent_id":      agentID,   // ✅ Agent isolation
			"session_id":    sessionID, // ✅ Session isolation
			"role":          roleStr,
			"content":       textContent, // Store content for retrieval
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
// Filters by BOTH agentID and sessionID to ensure proper isolation
func (v *VectorMemoryStrategy) Recall(agentID string, sessionID string, query string, limit int) ([]*pb.Message, error) {
	if query == "" {
		return []*pb.Message{}, nil
	}

	// Generate query embedding
	queryVector, err := v.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Search with BOTH agent and session filters for proper isolation
	ctx := context.Background()
	results, err := v.db.SearchWithFilter(ctx, v.collection, queryVector, limit, map[string]interface{}{
		"agent_id":   agentID,   // ✅ Agent isolation (prevents cross-agent leaks)
		"session_id": sessionID, // ✅ Session isolation
	})
	if err != nil {
		return nil, fmt.Errorf("recall failed: %w", err)
	}

	// Convert search results to messages
	messages := make([]*pb.Message, 0, len(results))
	for _, result := range results {
		// Extract role from metadata
		roleStr, ok := result.Metadata["role"].(string)
		if !ok {
			roleStr = "assistant" // Default fallback
		}

		// Extract content from metadata
		contentStr, ok := result.Metadata["content"].(string)
		if !ok {
			contentStr = result.Content // Fallback to result content
		}

		// Convert role string to pb.Role enum
		var pbRole pb.Role
		switch roleStr {
		case "user":
			pbRole = pb.Role_ROLE_USER
		case "assistant", "agent":
			pbRole = pb.Role_ROLE_AGENT
		default:
			pbRole = pb.Role_ROLE_UNSPECIFIED
		}

		// Create pb.Message
		msg := &pb.Message{
			Role: pbRole,
			Content: []*pb.Part{
				{Part: &pb.Part_Text{Text: contentStr}},
			},
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// Clear removes all long-term memory for a session
// Deletes all vectors associated with the agent+session from Qdrant
func (v *VectorMemoryStrategy) Clear(agentID string, sessionID string) error {
	// Delete all points where agent_id = agentID AND session_id = sessionID
	ctx := context.Background()
	return v.db.DeleteByFilter(ctx, v.collection, map[string]interface{}{
		"agent_id":   agentID,   // ✅ Agent isolation
		"session_id": sessionID, // ✅ Session isolation
	})
}
