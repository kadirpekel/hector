package memorytool

import (
	"encoding/json"
	"fmt"

	"github.com/verikod/hector/pkg/tool"
)

// SearchTool allows agents to search their long-term memory.
// This is injected when auto-recall is disabled.
type SearchTool struct {
	defaultLimit int
	maxLimit     int
}

// New creates a new memory search tool.
func New() *SearchTool {
	return &SearchTool{
		defaultLimit: 5,
		maxLimit:     20,
	}
}

func (t *SearchTool) Name() string {
	return "memory_search"
}

func (t *SearchTool) Description() string {
	return "Search your long-term memory for past conversations, facts, and user details. Memory storage is automatic; do not attempt to write to memory. Use this tool only when you truly need to recall information to answer the user."

}

func (t *SearchTool) IsLongRunning() bool {
	return false
}

func (t *SearchTool) RequiresApproval() bool {
	return false
}

func (t *SearchTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query to find relevant memories",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("Maximum number of memories to return (default: %d)", t.defaultLimit),
			},
		},
		"required": []string{"query"},
	}
}

func (t *SearchTool) Call(ctx tool.Context, args map[string]any) (map[string]any, error) {
	// Parse arguments
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	limit := t.defaultLimit
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	} else if l, ok := args["limit"].(int); ok {
		limit = l
	}

	if limit <= 0 {
		limit = t.defaultLimit
	}
	if limit > t.maxLimit {
		limit = t.maxLimit
	}

	// Search Memory via tool context
	// Note: ctx.SearchMemory is the correct way to access memory from a tool
	response, err := ctx.SearchMemory(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("memory search failed: %w", err)
	}

	// Slice and sort results (if not already sorted by score)
	// Usually vector search returns sorted results.
	results := response.Results
	if len(results) > limit {
		results = results[:limit]
	}

	// Return as structured data
	// Convert to map for tool output
	data, err := json.Marshal(map[string]any{
		"results": results,
		"count":   len(results),
		"query":   query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// Ensure interface implementation
var _ tool.CallableTool = (*SearchTool)(nil)
