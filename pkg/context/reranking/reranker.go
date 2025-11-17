package reranking

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// Reranker interface for re-ranking search results
type Reranker interface {
	// Rerank re-ranks search results based on query relevance
	// Returns topK re-ranked results sorted by relevance score
	Rerank(ctx context.Context, query string, results []databases.SearchResult, topK int) ([]databases.SearchResult, error)
}

// LLMProviderForReranking is a simplified interface for reranking
type LLMProviderForReranking interface {
	Generate(ctx context.Context, messages []*pb.Message, tools []llms.ToolDefinition) (string, []*protocol.ToolCall, int, error)
}

// LLMReranker uses an LLM to re-rank search results
type LLMReranker struct {
	llmProvider LLMProviderForReranking
	maxResults  int // Maximum number of results to send to LLM for reranking
}

// NewLLMReranker creates a new LLM-based reranker
func NewLLMReranker(llmProvider LLMProviderForReranking, maxResults int) *LLMReranker {
	if maxResults <= 0 {
		maxResults = 20 // Default: rerank up to 20 results
	}
	return &LLMReranker{
		llmProvider: llmProvider,
		maxResults:  maxResults,
	}
}

// Rerank implements the Reranker interface
func (r *LLMReranker) Rerank(ctx context.Context, query string, results []databases.SearchResult, topK int) ([]databases.SearchResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	// Limit the number of results to rerank (for efficiency and token limits)
	resultsToRerank := results
	if len(resultsToRerank) > r.maxResults {
		resultsToRerank = resultsToRerank[:r.maxResults]
	}

	// Build prompt for LLM reranking
	prompt := r.buildRerankingPrompt(query, resultsToRerank)

	// Create messages for LLM
	messages := []*pb.Message{
		{
			Role: pb.Role_ROLE_AGENT, // System message uses AGENT role
			Parts: []*pb.Part{
				{
					Part: &pb.Part_Text{
						Text: "You are a search result reranking system. Your task is to score and rank search results based on their relevance to a query. Return a JSON array of result IDs sorted by relevance (most relevant first).",
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

	// Call LLM to get reranked order
	response, _, _, err := r.llmProvider.Generate(ctx, messages, nil)
	if err != nil {
		// If LLM call fails, return original results
		return results[:min(topK, len(results))], fmt.Errorf("failed to rerank results: %w", err)
	}

	// Parse LLM response to get reranked order
	rerankedIDs, err := r.parseRerankingResponse(response)
	if err != nil {
		// If parsing fails, return original results
		return results[:min(topK, len(results))], nil
	}

	// Create map for quick lookup
	resultMap := make(map[string]databases.SearchResult)
	for _, result := range resultsToRerank {
		resultMap[result.ID] = result
	}

	// Reorder results based on LLM ranking
	reranked := make([]databases.SearchResult, 0, len(rerankedIDs))
	seen := make(map[string]bool)

	// Add reranked results in LLM order
	for _, id := range rerankedIDs {
		if result, exists := resultMap[id]; exists && !seen[id] {
			// Boost score slightly for reranked results (they're more relevant)
			result.Score = result.Score * 1.1
			reranked = append(reranked, result)
			seen[id] = true
		}
	}

	// Add any remaining results that weren't in LLM response
	for _, result := range resultsToRerank {
		if !seen[result.ID] {
			reranked = append(reranked, result)
		}
	}

	// Sort by score (highest first) and limit to topK
	sort.Slice(reranked, func(i, j int) bool {
		return reranked[i].Score > reranked[j].Score
	})

	if len(reranked) > topK {
		reranked = reranked[:topK]
	}

	return reranked, nil
}

// buildRerankingPrompt creates a prompt for LLM reranking
func (r *LLMReranker) buildRerankingPrompt(query string, results []databases.SearchResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Query: %s\n\n", query))
	sb.WriteString("Search Results:\n\n")

	for i, result := range results {
		// Truncate content if too long
		content := result.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}

		sb.WriteString(fmt.Sprintf("Result %d (ID: %s):\n", i+1, result.ID))
		sb.WriteString(fmt.Sprintf("Content: %s\n", content))
		if len(result.Metadata) > 0 {
			metadataStr := ""
			for k, v := range result.Metadata {
				if k != "content" {
					metadataStr += fmt.Sprintf("%s: %v, ", k, v)
				}
			}
			if metadataStr != "" {
				sb.WriteString(fmt.Sprintf("Metadata: %s\n", strings.TrimSuffix(metadataStr, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Please return a JSON array of result IDs sorted by relevance to the query (most relevant first).\n")
	sb.WriteString("Format: [\"id1\", \"id2\", \"id3\", ...]\n")
	sb.WriteString("Only include IDs that are relevant. Exclude irrelevant results.\n")

	return sb.String()
}

// parseRerankingResponse parses LLM response to extract reranked IDs
func (r *LLMReranker) parseRerankingResponse(response string) ([]string, error) {
	// Try to extract JSON array from response
	response = strings.TrimSpace(response)

	// Find JSON array in response (might have markdown code blocks)
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")
	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	jsonStr := response[startIdx : endIdx+1]

	var ids []string
	if err := json.Unmarshal([]byte(jsonStr), &ids); err != nil {
		// Try parsing as array of strings with different formats
		// Sometimes LLMs return with quotes or other formatting
		jsonStr = strings.ReplaceAll(jsonStr, "'", "\"")
		if err := json.Unmarshal([]byte(jsonStr), &ids); err != nil {
			// Last resort: try to extract IDs manually
			return r.extractIDsManually(response), nil
		}
	}

	return ids, nil
}

// extractIDsManually tries to extract IDs from response text
func (r *LLMReranker) extractIDsManually(response string) []string {
	var ids []string
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for patterns like "id1", 'id1', or id1
		if strings.Contains(line, "\"") {
			parts := strings.Split(line, "\"")
			for i := 1; i < len(parts); i += 2 {
				if len(parts[i]) > 0 {
					ids = append(ids, parts[i])
				}
			}
		} else if strings.Contains(line, "'") {
			parts := strings.Split(line, "'")
			for i := 1; i < len(parts); i += 2 {
				if len(parts[i]) > 0 {
					ids = append(ids, parts[i])
				}
			}
		} else {
			// Try to find ID-like strings
			words := strings.Fields(line)
			for _, word := range words {
				word = strings.Trim(word, "[]\",'")
				if len(word) > 0 && (strings.HasPrefix(word, "result-") || len(word) > 10) {
					ids = append(ids, word)
				}
			}
		}
	}
	return ids
}

// NoOpReranker is a reranker that doesn't modify results (for when reranking is disabled)
type NoOpReranker struct{}

// NewNoOpReranker creates a no-op reranker
func NewNoOpReranker() *NoOpReranker {
	return &NoOpReranker{}
}

// Rerank returns results unchanged
func (r *NoOpReranker) Rerank(ctx context.Context, query string, results []databases.SearchResult, topK int) ([]databases.SearchResult, error) {
	if len(results) > topK {
		return results[:topK], nil
	}
	return results, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

