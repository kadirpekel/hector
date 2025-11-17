package context

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/context/reranking"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/llms"
)

const (
	DefaultTopK = 10

	MaxTopK = 100

	DefaultSearchTimeout = 30 * time.Second

	MinQueryLength = 1

	MaxQueryLength = 10000
)

type SearchError struct {
	Component string
	Operation string
	Message   string
	Query     string
	Err       error
}

func (e *SearchError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s (query: %q): %v", e.Component, e.Operation, e.Message, e.Query, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s (query: %q)", e.Component, e.Operation, e.Message, e.Query)
}

func (e *SearchError) Unwrap() error {
	return e.Err
}

func NewSearchError(component, operation, message, query string, err error) *SearchError {
	return &SearchError{
		Component: component,
		Operation: operation,
		Message:   message,
		Query:     query,
		Err:       err,
	}
}

type SearchEngine struct {
	mu            sync.RWMutex
	db            databases.DatabaseProvider
	embedder      embedders.EmbedderProvider
	config        config.SearchConfig
	reranker      reranking.Reranker      // Optional reranker for re-ranking results
	queryExpander QueryExpander           // Optional query expander for multi-query
	llmRegistry   LLMRegistryForReranking // Optional LLM registry for HyDE and other LLM-based features
}

// ParallelSearchTarget represents a single target to search in parallel
type ParallelSearchTarget interface {
	// GetID returns a unique identifier for this search target (for error reporting)
	GetID() string
}

// ParallelSearchFunc is a function that performs a search for a single target
// It should return results and an error. If error is non-nil, results are ignored.
type ParallelSearchFunc[T ParallelSearchTarget, R any] func(ctx context.Context, target T) (R, error)

// ParallelSearchResult holds results from a single parallel search operation
type ParallelSearchResult[T any] struct {
	TargetID string
	Results  T
	Error    error
}

// ParallelSearch executes searches across multiple targets in parallel
// It handles goroutines, error recovery, context cancellation, and result collection
func ParallelSearch[T ParallelSearchTarget, R any](
	ctx context.Context,
	targets []T,
	searchFunc ParallelSearchFunc[T, R],
) ([]ParallelSearchResult[R], error) {
	if len(targets) == 0 {
		return []ParallelSearchResult[R]{}, nil
	}

	var wg sync.WaitGroup
	resultsChan := make(chan ParallelSearchResult[R], len(targets))

	// Debug: Log parallel execution start
	if len(targets) > 1 {
		targetIDs := make([]string, len(targets))
		for i, t := range targets {
			targetIDs[i] = t.GetID()
		}
		slog.Debug("Launching parallel searches", "targets", targetIDs, "count", len(targets))
	}

	// Launch parallel searches for all targets
	for _, target := range targets {
		wg.Add(1)
		go func(t T) {
			defer wg.Done()
			defer func() {
				// Recover from any panics to prevent hanging
				if r := recover(); r != nil {
					slog.Error("Panic in parallel search", "target", t.GetID(), "panic", r)
					resultsChan <- ParallelSearchResult[R]{
						TargetID: t.GetID(),
						Error:    fmt.Errorf("panic: %v", r),
					}
				}
			}()

			// Check for context cancellation
			select {
			case <-ctx.Done():
				resultsChan <- ParallelSearchResult[R]{
					TargetID: t.GetID(),
					Error:    ctx.Err(),
				}
				return
			default:
			}

			// Perform the search
			results, err := searchFunc(ctx, t)
			if err != nil {
				resultsChan <- ParallelSearchResult[R]{
					TargetID: t.GetID(),
					Error:    err,
				}
				return
			}

			// Check for context cancellation before sending results
			select {
			case <-ctx.Done():
				resultsChan <- ParallelSearchResult[R]{
					TargetID: t.GetID(),
					Error:    ctx.Err(),
				}
				return
			case resultsChan <- ParallelSearchResult[R]{
				TargetID: t.GetID(),
				Results:  results,
			}:
			}
		}(target)
	}

	// Wait for all searches to complete and close channel
	go func() {
		wg.Wait()
		close(resultsChan)
		if len(targets) > 1 {
			slog.Debug("All parallel searches completed, channel closed")
		}
	}()

	// Collect results from all targets
	var allResults []ParallelSearchResult[R]
	for {
		select {
		case <-ctx.Done():
			// Context cancelled or timed out, return partial results
			if len(targets) > 1 {
				slog.Debug("Parallel search cancelled or timed out", "results_collected", len(allResults), "error", ctx.Err())
			}
			return allResults, ctx.Err()
		case result, ok := <-resultsChan:
			if !ok {
				// Channel closed, all results collected
				if len(targets) > 1 {
					slog.Debug("All parallel search results collected", "total_results", len(allResults))
				}
				return allResults, nil
			}
			allResults = append(allResults, result)
		}
	}
}

// LLMRegistryForReranking is an interface for getting LLM providers
type LLMRegistryForReranking interface {
	GetLLM(name string) (llms.LLMProvider, error)
}

// NewSearchEngine creates a new search engine
// llmRegistry is required if reranking is enabled
func NewSearchEngine(db databases.DatabaseProvider, embedder embedders.EmbedderProvider, searchConfig config.SearchConfig, llmRegistry LLMRegistryForReranking) (*SearchEngine, error) {
	if db == nil {
		return nil, NewSearchError("SearchEngine", "NewSearchEngine", "database provider is required", "", nil)
	}
	if embedder == nil {
		return nil, NewSearchError("SearchEngine", "NewSearchEngine", "embedder provider is required", "", nil)
	}

	engine := &SearchEngine{
		db:          db,
		embedder:    embedder,
		config:      searchConfig,
		llmRegistry: llmRegistry,
	}

	// Initialize reranker if configured
	if searchConfig.Rerank != nil && searchConfig.Rerank.Enabled != nil && *searchConfig.Rerank.Enabled {
		if searchConfig.Rerank.LLM == "" {
			return nil, NewSearchError("SearchEngine", "NewSearchEngine", "rerank.llm is required when reranking is enabled", "", nil)
		}

		// Get LLM provider from registry - this is required, not optional
		if llmRegistry == nil {
			return nil, NewSearchError("SearchEngine", "NewSearchEngine", "LLM registry is required when reranking is enabled", "", nil)
		}

		llmProvider, err := llmRegistry.GetLLM(searchConfig.Rerank.LLM)
		if err != nil {
			return nil, NewSearchError("SearchEngine", "NewSearchEngine", fmt.Sprintf("failed to get reranker LLM '%s' from registry: %v", searchConfig.Rerank.LLM, err), "", err)
		}

		// Create reranker with LLM provider
		maxResults := searchConfig.Rerank.MaxResults
		if maxResults == 0 {
			maxResults = 20
		}
		engine.reranker = reranking.NewLLMReranker(llmProvider, maxResults)
		slog.Debug("Reranker initialized successfully", "llm", searchConfig.Rerank.LLM, "max_results", maxResults)
	}

	// Initialize query expander if multi-query is enabled
	if searchConfig.MultiQuery != nil && searchConfig.MultiQuery.Enabled != nil && *searchConfig.MultiQuery.Enabled {
		if searchConfig.MultiQuery.LLM == "" {
			return nil, NewSearchError("SearchEngine", "NewSearchEngine", "multi_query.llm is required when multi-query is enabled", "", nil)
		}

		// Get LLM provider from registry if available
		if llmRegistry != nil {
			llmProvider, err := llmRegistry.GetLLM(searchConfig.MultiQuery.LLM)
			if err != nil {
				return nil, NewSearchError("SearchEngine", "NewSearchEngine", fmt.Sprintf("failed to get LLM '%s' for multi-query: %v", searchConfig.MultiQuery.LLM, err), "", err)
			}
			engine.queryExpander = NewLLMQueryExpander(llmProvider)
		}
	}

	return engine, nil
}

func (se *SearchEngine) validateQuery(query string) error {
	if query == "" {
		return NewSearchError("SearchEngine", "validateQuery", "query cannot be empty", "", nil)
	}
	if len(query) < MinQueryLength {
		return NewSearchError("SearchEngine", "validateQuery", "query too short", query, nil)
	}
	if len(query) > MaxQueryLength {
		return NewSearchError("SearchEngine", "validateQuery", "query too long", query, nil)
	}
	return nil
}

func (se *SearchEngine) processQuery(query string) string {
	// Trim whitespace
	processed := strings.TrimSpace(query)

	// Determine if case should be preserved (default: true for code search)
	// Preserving case is important for code identifiers like HTTP, API, etc.
	preserveCase := se.config.PreserveCase == nil || *se.config.PreserveCase
	if !preserveCase {
		processed = strings.ToLower(processed)
	}

	// Always normalize whitespace for query consistency
	// This collapses multiple spaces and ensures clean queries
	processed = strings.Join(strings.Fields(processed), " ")

	return processed
}

func (se *SearchEngine) IngestDocument(ctx context.Context, docID, content string, metadata map[string]interface{}) error {
	se.mu.RLock()
	defer se.mu.RUnlock()

	if docID == "" {
		return NewSearchError("SearchEngine", "IngestDocument", "document ID cannot be empty", "", nil)
	}
	if content == "" {
		return NewSearchError("SearchEngine", "IngestDocument", "content cannot be empty", docID, nil)
	}

	// Get collection name from metadata
	// DocumentStore always sets "collection" in metadata when indexing
	collection := se.getCollectionFromMap(metadata)
	if collection == "" {
		return NewSearchError("SearchEngine", "IngestDocument", "collection must be specified in metadata", docID, nil)
	}

	embedCtx, cancel := context.WithTimeout(ctx, DefaultSearchTimeout)
	defer cancel()

	vector, err := se.embedder.Embed(content)
	if err != nil {
		return NewSearchError("SearchEngine", "IngestDocument", "failed to generate embedding", docID, err)
	}

	preparedMetadata := se.prepareMetadata(content, metadata)

	if err := se.db.Upsert(embedCtx, collection, docID, vector, preparedMetadata); err != nil {
		return NewSearchError("SearchEngine", "IngestDocument", "failed to upsert document", docID, err)
	}

	return nil
}

func (se *SearchEngine) Search(ctx context.Context, query string, limit int) ([]databases.SearchResult, error) {
	return se.SearchWithFilter(ctx, query, limit, nil)
}

func (se *SearchEngine) SearchWithFilter(ctx context.Context, query string, limit int, filter map[string]interface{}) ([]databases.SearchResult, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	if err := se.validateQuery(query); err != nil {
		return nil, err
	}
	processedQuery := se.processQuery(query)

	// Use config TopK as default if limit not specified
	if limit <= 0 {
		limit = se.config.TopK
		if limit <= 0 {
			limit = DefaultTopK
		}
	}
	if limit > MaxTopK {
		limit = MaxTopK
	}

	embedCtx, cancel := context.WithTimeout(ctx, DefaultSearchTimeout)
	defer cancel()

	// Get collection from filter (required)
	collection := se.getCollectionFromMap(filter)
	if collection == "" {
		return nil, NewSearchError("SearchEngine", "SearchWithFilter", "collection must be specified in filter", processedQuery, nil)
	}

	// Check search mode and route to appropriate search method
	searchMode := se.config.SearchMode
	if searchMode == "" {
		searchMode = "vector" // Default
	}

	slog.Debug("Search mode selected", "mode", searchMode, "query", processedQuery, "collection", collection)

	var results []databases.SearchResult
	var err error

	switch searchMode {
	case "multi_query":
		// Multi-query expansion: generate multiple query variations and merge results
		slog.Debug("Using multi-query expansion", "query", processedQuery)
		results, err = se.searchWithMultiQuery(ctx, embedCtx, processedQuery, collection, limit, filter)
		if err != nil {
			return nil, NewSearchError("SearchEngine", "SearchWithFilter", "multi-query search failed", processedQuery, err)
		}

	case "hyde":
		// HyDE: generate hypothetical document and search with its embedding
		slog.Debug("Using HyDE search", "query", processedQuery)
		results, err = se.searchWithHyDE(ctx, embedCtx, processedQuery, collection, limit, filter)
		if err != nil {
			return nil, NewSearchError("SearchEngine", "SearchWithFilter", "HyDE search failed", processedQuery, err)
		}

	case "hybrid":
		// Generate embedding for vector search
		slog.Debug("Using hybrid search", "query", processedQuery, "alpha", se.config.HybridAlpha)
		vector, embedErr := se.embedder.Embed(processedQuery)
		if embedErr != nil {
			return nil, NewSearchError("SearchEngine", "SearchWithFilter", "failed to generate embedding", processedQuery, embedErr)
		}

		// Use hybrid alpha from config (default 0.5 if not set)
		alpha := se.config.HybridAlpha
		if alpha == 0 {
			alpha = 0.5 // Default balanced hybrid
		}

		slog.Debug("Executing hybrid search", "alpha", alpha, "vector_dim", len(vector))
		results, err = se.db.HybridSearch(embedCtx, collection, processedQuery, vector, limit, filter, alpha)
		if err != nil {
			return nil, NewSearchError("SearchEngine", "SearchWithFilter", "hybrid search failed", processedQuery, err)
		}
		slog.Debug("Hybrid search completed", "results", len(results))

	case "keyword":
		// Pure keyword search (not fully implemented yet, fallback to hybrid with alpha=0)
		// For now, use hybrid with very low alpha to favor keywords
		vector, embedErr := se.embedder.Embed(processedQuery)
		if embedErr != nil {
			return nil, NewSearchError("SearchEngine", "SearchWithFilter", "failed to generate embedding", processedQuery, embedErr)
		}

		results, err = se.db.HybridSearch(embedCtx, collection, processedQuery, vector, limit, filter, 0.1)
		if err != nil {
			return nil, NewSearchError("SearchEngine", "SearchWithFilter", "keyword search failed", processedQuery, err)
		}

	default: // "vector" or empty
		// Standard vector search
		slog.Debug("Using vector search", "query", processedQuery)
		vector, embedErr := se.embedder.Embed(processedQuery)
		if embedErr != nil {
			return nil, NewSearchError("SearchEngine", "SearchWithFilter", "failed to generate embedding", processedQuery, embedErr)
		}

		results, err = se.db.SearchWithFilter(embedCtx, collection, vector, limit, filter)
		if err != nil {
			return nil, NewSearchError("SearchEngine", "SearchWithFilter", "database search failed", processedQuery, err)
		}
		slog.Debug("Vector search completed", "results", len(results))
	}

	// Apply reranking BEFORE threshold filtering
	// This is important because:
	// 1. Initial vector search scores might be low
	// 2. LLM reranking can identify truly relevant results
	// 3. Threshold should be applied to reranked scores, not raw vector scores
	if se.reranker != nil && len(results) > 0 {
		slog.Debug("Applying LLM-based reranking", "results_before", len(results), "max_results", se.config.Rerank.MaxResults)
		reranked, err := se.reranker.Rerank(ctx, processedQuery, results, limit)
		if err != nil {
			// Log error but continue with original results
			slog.Warn("Reranking failed, continuing with original results", "error", err)
		} else {
			slog.Debug("Reranking completed", "results_after", len(reranked))
			results = reranked
		}
	}

	// Filter results by similarity threshold from config (after reranking)
	if se.config.Threshold > 0 {
		filtered := make([]databases.SearchResult, 0, len(results))
		for _, result := range results {
			if result.Score >= se.config.Threshold {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	return results, nil
}

func (se *SearchEngine) DeleteByFilter(ctx context.Context, filter map[string]interface{}) error {
	se.mu.RLock()
	defer se.mu.RUnlock()

	// Get collection from filter (required)
	collection := se.getCollectionFromMap(filter)
	if collection == "" {
		return NewSearchError("SearchEngine", "DeleteByFilter", "collection must be specified in filter", "", nil)
	}

	return se.db.DeleteByFilter(ctx, collection, filter)
}

// getCollectionFromMap extracts collection name from a map (metadata or filter)
// Checks "collection" first, then falls back to "store_name" for backward compatibility
func (se *SearchEngine) getCollectionFromMap(m map[string]interface{}) string {
	if m == nil {
		return ""
	}

	// Prefer explicit collection name
	if collection, ok := m["collection"].(string); ok && collection != "" {
		return collection
	}

	// Fallback to store_name for backward compatibility
	if storeName, ok := m["store_name"].(string); ok && storeName != "" {
		return storeName
	}

	return ""
}

func (se *SearchEngine) prepareMetadata(content string, metadata map[string]interface{}) map[string]interface{} {
	prepared := make(map[string]interface{})

	prepared["content"] = content

	// Copy all metadata fields
	for k, v := range metadata {
		prepared[k] = v
	}

	prepared["ingested_at"] = time.Now().Unix()

	return prepared
}

func (se *SearchEngine) GetStatus() map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()

	return map[string]interface{}{
		"config": se.config,
	}
}
