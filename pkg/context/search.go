package context

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
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
	mu       sync.RWMutex
	db       databases.DatabaseProvider
	embedder embedders.EmbedderProvider
	config   config.SearchConfig
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

	// Launch parallel searches for all targets
	for _, target := range targets {
		wg.Add(1)
		go func(t T) {
			defer wg.Done()
			defer func() {
				// Recover from any panics to prevent hanging
				if r := recover(); r != nil {
					fmt.Printf("Error: Panic in parallel search for target %s: %v\n", t.GetID(), r)
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
	}()

	// Collect results from all targets
	var allResults []ParallelSearchResult[R]
	for {
		select {
		case <-ctx.Done():
			// Context cancelled or timed out, return partial results
			return allResults, ctx.Err()
		case result, ok := <-resultsChan:
			if !ok {
				// Channel closed, all results collected
				return allResults, nil
			}
			allResults = append(allResults, result)
		}
	}
}

func NewSearchEngine(db databases.DatabaseProvider, embedder embedders.EmbedderProvider, searchConfig config.SearchConfig) (*SearchEngine, error) {
	if db == nil {
		return nil, NewSearchError("SearchEngine", "NewSearchEngine", "database provider is required", "", nil)
	}
	if embedder == nil {
		return nil, NewSearchError("SearchEngine", "NewSearchEngine", "embedder provider is required", "", nil)
	}

	engine := &SearchEngine{
		db:       db,
		embedder: embedder,
		config:   searchConfig,
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

	// Lowercase query only if explicitly disabled (default: preserve for code search)
	// Preserving case is important for code identifiers like HTTP, API, etc.
	if se.config.PreserveCase == nil || !*se.config.PreserveCase {
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

	vector, err := se.embedder.Embed(processedQuery)
	if err != nil {
		return nil, NewSearchError("SearchEngine", "SearchWithFilter", "failed to generate embedding", processedQuery, err)
	}

	// Get collection from filter (required)
	collection := se.getCollectionFromMap(filter)
	if collection == "" {
		return nil, NewSearchError("SearchEngine", "SearchWithFilter", "collection must be specified in filter", processedQuery, nil)
	}

	results, err := se.db.SearchWithFilter(embedCtx, collection, vector, limit, filter)
	if err != nil {
		return nil, NewSearchError("SearchEngine", "SearchWithFilter", "database search failed", processedQuery, err)
	}

	// Filter results by similarity threshold from config
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
