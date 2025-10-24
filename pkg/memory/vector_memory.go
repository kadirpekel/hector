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

type VectorMemoryStrategy struct {
	db         databases.DatabaseProvider
	embedder   embedders.EmbedderProvider
	collection string
}

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

func (v *VectorMemoryStrategy) Name() string {
	return "vector_memory"
}

func (v *VectorMemoryStrategy) Store(agentID string, sessionID string, messages []*pb.Message) error {
	if len(messages) == 0 {
		return nil
	}

	for i, msg := range messages {

		textContent := protocol.ExtractTextFromMessage(msg)
		if textContent == "" {
			continue
		}

		vector, err := v.embedder.Embed(textContent)
		if err != nil {
			return fmt.Errorf("failed to embed message %d: %w", i, err)
		}

		docID := uuid.New().String()

		roleStr := "unknown"
		switch msg.Role {
		case pb.Role_ROLE_USER:
			roleStr = "user"
		case pb.Role_ROLE_AGENT:
			roleStr = "agent"
		case pb.Role_ROLE_UNSPECIFIED:
			roleStr = "system"
		}

		metadata := map[string]interface{}{
			"agent_id":      agentID,
			"session_id":    sessionID,
			"role":          roleStr,
			"content":       textContent,
			"message_index": i,
		}

		ctx := context.Background()
		if err := v.db.Upsert(ctx, v.collection, docID, vector, metadata); err != nil {
			return fmt.Errorf("failed to store message %d: %w", i, err)
		}
	}

	return nil
}

func (v *VectorMemoryStrategy) Recall(agentID string, sessionID string, query string, limit int) ([]*pb.Message, error) {
	if query == "" {
		return []*pb.Message{}, nil
	}

	queryVector, err := v.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	ctx := context.Background()
	results, err := v.db.SearchWithFilter(ctx, v.collection, queryVector, limit, map[string]interface{}{
		"agent_id":   agentID,
		"session_id": sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("recall failed: %w", err)
	}

	messages := make([]*pb.Message, 0, len(results))
	for _, result := range results {

		roleStr, ok := result.Metadata["role"].(string)
		if !ok {
			roleStr = "assistant"
		}

		contentStr, ok := result.Metadata["content"].(string)
		if !ok {
			contentStr = result.Content
		}

		var pbRole pb.Role
		switch roleStr {
		case "user":
			pbRole = pb.Role_ROLE_USER
		case "assistant", "agent":
			pbRole = pb.Role_ROLE_AGENT
		default:
			pbRole = pb.Role_ROLE_UNSPECIFIED
		}

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

func (v *VectorMemoryStrategy) Clear(agentID string, sessionID string) error {

	ctx := context.Background()
	return v.db.DeleteByFilter(ctx, v.collection, map[string]interface{}{
		"agent_id":   agentID,
		"session_id": sessionID,
	})
}
