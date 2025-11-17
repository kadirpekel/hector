package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/llms"
)

// QueryExpander expands a single query into multiple query variations
type QueryExpander interface {
	// Expand generates multiple query variations from the original query
	Expand(ctx context.Context, query string, numVariations int) ([]string, error)
}

// LLMQueryExpander uses an LLM to generate query variations
type LLMQueryExpander struct {
	llmProvider llms.LLMProvider
}

// NewLLMQueryExpander creates a new LLM-based query expander
func NewLLMQueryExpander(llmProvider llms.LLMProvider) *LLMQueryExpander {
	return &LLMQueryExpander{
		llmProvider: llmProvider,
	}
}

// Expand implements the QueryExpander interface
func (e *LLMQueryExpander) Expand(ctx context.Context, query string, numVariations int) ([]string, error) {
	if numVariations <= 0 {
		numVariations = 3 // Default: generate 3 variations
	}
	if numVariations > 5 {
		numVariations = 5 // Cap at 5 variations to avoid too many API calls
	}

	prompt := fmt.Sprintf(`Generate %d different query variations for the following search query. Each variation should:
1. Use different wording or phrasing
2. Focus on different aspects or perspectives
3. Be semantically similar but not identical
4. Be suitable for document retrieval

Original query: %s

Return only a JSON array of query strings, one per line, without any additional text or explanation.
Example format: ["query 1", "query 2", "query 3"]`, numVariations, query)

	messages := []*pb.Message{
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

	response, _, _, err := e.llmProvider.Generate(ctx, messages, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query variations: %w", err)
	}

	// Parse JSON array from response
	queries, err := parseQueryArray(response)
	if err != nil {
		// Fallback: try to extract queries manually
		queries = extractQueriesFromText(response)
	}

	// Ensure we have at least the original query
	if len(queries) == 0 {
		queries = []string{query}
	}

	// Limit to requested number
	if len(queries) > numVariations {
		queries = queries[:numVariations]
	}

	return queries, nil
}

// parseQueryArray parses a JSON array of query strings
func parseQueryArray(response string) ([]string, error) {
	// Find JSON array in response
	startIdx := -1
	endIdx := -1
	depth := 0

	for i, char := range response {
		if char == '[' {
			if startIdx == -1 {
				startIdx = i
			}
			depth++
		} else if char == ']' {
			depth--
			if depth == 0 && startIdx != -1 {
				endIdx = i + 1
				break
			}
		}
	}

	if startIdx == -1 || endIdx == -1 {
		return nil, fmt.Errorf("no JSON array found")
	}

	jsonStr := response[startIdx:endIdx]

	// Simple JSON parsing for string array
	// Remove brackets and quotes
	jsonStr = jsonStr[1 : len(jsonStr)-1] // Remove [ and ]
	
	var queries []string
	var current strings.Builder
	inQuotes := false
	escape := false

	for _, char := range jsonStr {
		if escape {
			current.WriteRune(char)
			escape = false
			continue
		}

		if char == '\\' {
			escape = true
			continue
		}

		if char == '"' {
			if inQuotes {
				// End of string
				queries = append(queries, current.String())
				current.Reset()
			}
			inQuotes = !inQuotes
			continue
		}

		if inQuotes {
			current.WriteRune(char)
		}
	}

	if len(queries) == 0 {
		return nil, fmt.Errorf("failed to parse queries")
	}

	return queries, nil
}

// extractQueriesFromText tries to extract queries from unstructured text
func extractQueriesFromText(response string) []string {
	var queries []string
	lines := strings.Split(response, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for quoted strings
		if strings.HasPrefix(line, `"`) && strings.HasSuffix(line, `"`) {
			query := line[1 : len(line)-1] // Remove quotes
			if len(query) > 0 {
				queries = append(queries, query)
			}
		} else if strings.HasPrefix(line, `'`) && strings.HasSuffix(line, `'`) {
			query := line[1 : len(line)-1] // Remove quotes
			if len(query) > 0 {
				queries = append(queries, query)
			}
		} else if len(line) > 10 && !strings.Contains(line, ":") {
			// Might be a query without quotes
			queries = append(queries, line)
		}
	}

	return queries
}

