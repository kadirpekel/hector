package context

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/kadirpekel/hector/pkg/databases"
)

// querySearchTarget implements ParallelSearchTarget for multi-query parallel searches
type querySearchTarget struct {
	ID    string
	Query string
}

// GetID implements ParallelSearchTarget interface
func (t *querySearchTarget) GetID() string {
	return t.ID
}

// searchWithMultiQuery performs multi-query expansion search
func (se *SearchEngine) searchWithMultiQuery(ctx context.Context, embedCtx context.Context, query string, collection string, limit int, filter map[string]interface{}) ([]databases.SearchResult, error) {
	if se.queryExpander == nil {
		return nil, fmt.Errorf("query expander not available for multi-query search")
	}

	// Get number of variations from config
	numVariations := 3 // Default
	if se.config.MultiQuery != nil && se.config.MultiQuery.NumVariations > 0 {
		numVariations = se.config.MultiQuery.NumVariations
	}

	// Generate query variations
	queries, err := se.queryExpander.Expand(ctx, query, numVariations)
	if err != nil {
		// Fallback to original query if expansion fails
		slog.Warn("Query expansion failed, using original query", "error", err)
		queries = []string{query}
	}

	// Include original query in the list
	allQueries := append([]string{query}, queries...)

	// Use a higher limit per query to get more results for merging
	perQueryLimit := limit * 2
	if perQueryLimit > MaxTopK {
		perQueryLimit = MaxTopK
	}

	// Prepare targets for parallel search
	targets := make([]*querySearchTarget, len(allQueries))
	for i, q := range allQueries {
		targets[i] = &querySearchTarget{ID: fmt.Sprintf("query_%d", i), Query: q}
	}

	// Define the search function for each query
	searchFunc := func(ctx context.Context, target *querySearchTarget) ([]databases.SearchResult, error) {
		// Generate embedding for this query
		vector, err := se.embedder.Embed(target.Query)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for query variation: %w", err)
		}

		// Perform vector search
		queryResults, err := se.db.SearchWithFilter(embedCtx, collection, vector, perQueryLimit, filter)
		if err != nil {
			return nil, fmt.Errorf("search failed for query variation: %w", err)
		}

		return queryResults, nil
	}

	// Execute parallel searches
	rawResults, err := ParallelSearch(ctx, targets, searchFunc)
	if err != nil {
		return nil, fmt.Errorf("parallel search failed: %w", err)
	}

	// Collect all results and merge with score boosting for duplicates
	allResults := make([]databases.SearchResult, 0)
	seenIDs := make(map[string]int) // Map from ID to index in allResults

	for _, res := range rawResults {
		if res.Error != nil {
			slog.Warn("Parallel search for query variation failed", "query_id", res.TargetID, "error", res.Error)
			continue
		}

		// Merge results, avoiding duplicates and using max score
		for _, result := range res.Results {
			if idx, exists := seenIDs[result.ID]; exists {
				// Result already seen - use max score (order-independent)
				// If a document appears in multiple query results, keep the highest score
				if result.Score > allResults[idx].Score {
					allResults[idx].Score = result.Score
				}
			} else {
				// New result - add it
				allResults = append(allResults, result)
				seenIDs[result.ID] = len(allResults) - 1
			}
		}
	}

	// Sort by score (highest first)
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	// Limit to requested number
	if len(allResults) > limit {
		allResults = allResults[:limit]
	}

	return allResults, nil
}
