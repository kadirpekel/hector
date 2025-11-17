package context

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/llms"
)

// searchWithHyDE performs HyDE (Hypothetical Document Embeddings) search
func (se *SearchEngine) searchWithHyDE(ctx context.Context, embedCtx context.Context, query string, collection string, limit int, filter map[string]interface{}) ([]databases.SearchResult, error) {
	// HyDE requires an LLM to generate hypothetical documents
	if se.config.HyDE == nil || se.config.HyDE.LLM == "" {
		return nil, fmt.Errorf("HyDE requires llm to be configured")
	}

	// Get LLM provider from registry
	if se.llmRegistry == nil {
		slog.Warn("HyDE LLM registry not available, falling back to vector search", "query", query)
		// Fallback to regular vector search
		vector, err := se.embedder.Embed(query)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for fallback: %w", err)
		}
		return se.db.SearchWithFilter(embedCtx, collection, vector, limit, filter)
	}

	llmProvider, err := se.llmRegistry.GetLLM(se.config.HyDE.LLM)
	if err != nil {
		slog.Warn("Failed to get LLM for HyDE, falling back to vector search", "llm", se.config.HyDE.LLM, "error", err)
		// Fallback to regular vector search
		vector, embedErr := se.embedder.Embed(query)
		if embedErr != nil {
			return nil, fmt.Errorf("failed to generate embedding for fallback: %w", embedErr)
		}
		return se.db.SearchWithFilter(embedCtx, collection, vector, limit, filter)
	}

	// Generate hypothetical document
	hypotheticalDoc, err := generateHypotheticalDocument(ctx, llmProvider, query)
	if err != nil {
		slog.Error("Failed to generate hypothetical document for HyDE", "query", query, "error", err)
		// Fallback to regular vector search
		vector, embedErr := se.embedder.Embed(query)
		if embedErr != nil {
			return nil, fmt.Errorf("failed to generate embedding for fallback: %w", embedErr)
		}
		return se.db.SearchWithFilter(embedCtx, collection, vector, limit, filter)
	}

	// Embed the hypothetical document (not the query)
	vector, err := se.embedder.Embed(hypotheticalDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding for hypothetical document: %w", err)
	}

	// Perform vector search with the hypothetical document's embedding
	results, err := se.db.SearchWithFilter(embedCtx, collection, vector, limit, filter)
	if err != nil {
		return nil, fmt.Errorf("database search with hypothetical document failed: %w", err)
	}

	return results, nil
}

// generateHypotheticalDocument generates a hypothetical document that would answer the query
// This is the core of HyDE: instead of searching with the query, we search with a hypothetical answer
func generateHypotheticalDocument(ctx context.Context, llmProvider llms.LLMProvider, query string) (string, error) {
	prompt := fmt.Sprintf(`Write a concise, hypothetical document that would be highly relevant to answer the following query: "%s"

The document should be brief and directly address the core of the query.`, query)

	messages := []*pb.Message{
		{
			Role: pb.Role_ROLE_AGENT,
			Parts: []*pb.Part{
				{
					Part: &pb.Part_Text{
						Text: "You are an expert document writer. Your task is to generate a hypothetical document that directly answers a given query.",
					},
				},
			},
		},
		{
			Role: pb.Role_ROLE_USER,
			Parts: []*pb.Part{
				{
					Part: &pb.Part_Text{
						Text: prompt,
					},
				},
			},
		},
	}

	response, _, _, err := llmProvider.Generate(ctx, messages, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate hypothetical document: %w", err)
	}

	return response, nil
}

