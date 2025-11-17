// Package reranking provides LLM-based re-ranking of search results to improve relevance.
//
// # Overview
//
// Re-ranking is a post-processing step that uses an LLM to assess semantic relevance
// of search results beyond simple vector similarity. This improves search quality by:
//   - Understanding deeper semantic meaning
//   - Evaluating actual usefulness rather than just similarity
//   - Re-ordering results based on query intent
//
// # When to Use Reranking
//
// Use reranking when:
//   - Initial vector search returns low similarity scores but may include relevant results
//   - Query requires deep semantic understanding
//   - Results need ordering by actual usefulness, not just vector distance
//   - You have budget for LLM API calls (adds 100-500ms latency per search)
//
// Skip reranking when:
//   - Vector search already returns high-quality results
//   - Latency is critical
//   - Query is simple keyword match
//   - Cost constraints are strict
//
// # Score Semantics (IMPORTANT)
//
// Before reranking:
//   - Scores represent vector similarity (cosine distance, 0.0-1.0)
//   - Score 0.8 = "80% similar to query embedding"
//
// After reranking:
//   - Scores represent LLM-determined ranking position
//   - 1st result: 1.0, 2nd: 0.95, 3rd: 0.90, ... (decreasing by 0.05)
//   - Score 1.0 = "ranked first by LLM" (NOT "100% similar")
//   - Minimum score: 0.1 (for results beyond position 20)
//
// NOTE: Original vector scores are REPLACED, not preserved. This means:
//   - Threshold filtering should happen AFTER reranking
//   - Threshold semantics change (e.g., 0.5 = "top 11 results or better")
//   - Comparing scores across reranked/non-reranked results is meaningless
//
// # Configuration Example
//
//	search:
//	  top_k: 10
//	  threshold: 0.5  # Applied AFTER reranking (keeps top 11 results)
//	  search_mode: "hybrid"
//	  rerank:
//	    enabled: true
//	    llm: "reranker"  # Must reference an LLM in llms config
//	    max_results: 20  # Only rerank top 20 results (for efficiency)
//
// # Performance Considerations
//
//   - Latency: Adds 100-500ms per search (LLM API call)
//   - Cost: Incurs LLM API costs for every search
//   - Token usage: ~500 chars per result * maxResults = ~10KB per request
//   - Concurrency: LLM calls are sequential, not batched
//
// # Best Practices
//
//  1. Use hybrid or multi-query search first to get good candidates
//  2. Set maxResults to 10-20 (balance between quality and cost)
//  3. Apply threshold AFTER reranking, not before
//  4. Consider caching reranked results for identical queries
//  5. Use a fast LLM model (e.g., gpt-4o-mini) for reranking
//
// # Example Usage
//
//	// In your config:
//	llms:
//	  reranker:
//	    type: "openai"
//	    model: "gpt-4o-mini"  // Fast and cheap
//	    temperature: 0.0       // Deterministic ranking
//
//	search:
//	  rerank:
//	    enabled: true
//	    llm: "reranker"
//	    max_results: 20
//
// The reranker will automatically be initialized when SearchEngine is created.
package reranking

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

	slog.Debug("LLM reranking started", "query", query, "total_results", len(results), "max_results", r.maxResults)

	// Limit the number of results to rerank (for efficiency and token limits)
	resultsToRerank := results
	if len(resultsToRerank) > r.maxResults {
		resultsToRerank = resultsToRerank[:r.maxResults]
		slog.Debug("Limiting results for reranking", "original", len(results), "limited", len(resultsToRerank))
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

	// Add reranked results in LLM order with new relevance-based scores
	// The LLM has already ranked them, so we assign scores based on position
	for i, id := range rerankedIDs {
		if result, exists := resultMap[id]; exists && !seen[id] {
			// Assign score based on ranking position (1.0 for first, decreasing by 0.05 per position)
			// This ensures highly ranked results get high scores that pass reasonable thresholds
			newScore := 1.0 - (float32(i) * 0.05)
			if newScore < 0.1 {
				newScore = 0.1 // Minimum score for reranked results
			}

			// Create new result with all fields from original but updated score
			rerankedResult := databases.SearchResult{
				ID:       result.ID,
				Score:    newScore,
				Content:  result.Content,
				Metadata: result.Metadata,
			}
			reranked = append(reranked, rerankedResult)
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

	slog.Debug("LLM reranking completed", "original_count", len(results), "reranked_count", len(reranked))

	return reranked, nil
}

// buildRerankingPrompt creates a prompt for LLM reranking
func (r *LLMReranker) buildRerankingPrompt(query string, results []databases.SearchResult) string {
	var sb strings.Builder

	// Sanitize query to prevent prompt injection
	sanitizedQuery := sanitizeInput(query)
	sb.WriteString(fmt.Sprintf("Query: %s\n\n", sanitizedQuery))
	sb.WriteString("Search Results:\n\n")

	for i, result := range results {
		// Truncate content if too long
		content := result.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		// Sanitize content to prevent prompt injection
		content = sanitizeInput(content)

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

// sanitizeInput removes or escapes potential prompt injection patterns from user input
// This prevents malicious queries from manipulating the LLM's behavior
func sanitizeInput(input string) string {
	// Remove common prompt injection patterns
	sanitized := input

	// Remove system role indicators that could confuse the LLM
	sanitized = strings.ReplaceAll(sanitized, "SYSTEM:", "")
	sanitized = strings.ReplaceAll(sanitized, "System:", "")
	sanitized = strings.ReplaceAll(sanitized, "system:", "")
	sanitized = strings.ReplaceAll(sanitized, "ASSISTANT:", "")
	sanitized = strings.ReplaceAll(sanitized, "Assistant:", "")
	sanitized = strings.ReplaceAll(sanitized, "assistant:", "")
	sanitized = strings.ReplaceAll(sanitized, "USER:", "")
	sanitized = strings.ReplaceAll(sanitized, "User:", "")
	sanitized = strings.ReplaceAll(sanitized, "user:", "")

	// Remove instruction override attempts
	sanitized = strings.ReplaceAll(sanitized, "Ignore previous instructions", "")
	sanitized = strings.ReplaceAll(sanitized, "ignore previous instructions", "")
	sanitized = strings.ReplaceAll(sanitized, "Ignore all previous", "")
	sanitized = strings.ReplaceAll(sanitized, "ignore all previous", "")
	sanitized = strings.ReplaceAll(sanitized, "Disregard previous", "")
	sanitized = strings.ReplaceAll(sanitized, "disregard previous", "")

	// Remove common delimiter attacks (trying to break out of the prompt structure)
	sanitized = strings.ReplaceAll(sanitized, "---", "")
	sanitized = strings.ReplaceAll(sanitized, "===", "")
	sanitized = strings.ReplaceAll(sanitized, "***", "")

	// Escape backticks that could be used for code injection or markdown manipulation
	sanitized = strings.ReplaceAll(sanitized, "```", "")

	// Remove excessive whitespace that could be used for obfuscation
	sanitized = strings.TrimSpace(sanitized)

	return sanitized
}
