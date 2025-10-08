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

// ============================================================================
// SEARCH CONSTANTS AND CONFIGURATION
// ============================================================================

const (
	// DefaultTopK is the default number of results per model
	DefaultTopK = 10

	// MaxTopK is the maximum number of results per model
	MaxTopK = 100

	// DefaultSearchTimeout is the default timeout for search operations
	DefaultSearchTimeout = 30 * time.Second

	// MinQueryLength is the minimum query length for search
	MinQueryLength = 1

	// MaxQueryLength is the maximum query length for search
	MaxQueryLength = 10000
)

// ============================================================================
// SEARCH ERRORS - STANDARDIZED ERROR TYPES
// ============================================================================

// SearchError represents errors in search operations
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

// NewSearchError creates a new search error
func NewSearchError(component, operation, message, query string, err error) *SearchError {
	return &SearchError{
		Component: component,
		Operation: operation,
		Message:   message,
		Query:     query,
		Err:       err,
	}
}

// ============================================================================
// SEARCH ENGINE - ENHANCED WITH PROPER STRUCTURE
// ============================================================================

// SearchEngine handles search operations across multiple models with enhanced error handling and validation
type SearchEngine struct {
	mu       sync.RWMutex
	db       databases.DatabaseProvider
	embedder embedders.EmbedderProvider
	config   config.SearchConfig
	models   map[string]config.SearchModel
}

// NewSearchEngine creates a new search engine with proper validation and configuration
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
		models:   make(map[string]config.SearchModel),
	}

	// Initialize and validate models
	if err := engine.initializeModels(); err != nil {
		return nil, err
	}

	return engine, nil
}

// initializeModels initializes and validates search models
func (se *SearchEngine) initializeModels() error {
	if len(se.config.Models) == 0 {
		return NewSearchError("SearchEngine", "initializeModels", "no search models configured", "", nil)
	}

	for _, model := range se.config.Models {
		if err := se.validateModel(model); err != nil {
			return err
		}

		// Set defaults if not specified
		modelWithDefaults := model
		if modelWithDefaults.DefaultTopK == 0 {
			modelWithDefaults.DefaultTopK = DefaultTopK
		}
		if modelWithDefaults.MaxTopK == 0 {
			modelWithDefaults.MaxTopK = MaxTopK
		}

		se.models[model.Name] = modelWithDefaults
	}

	return nil
}

// validateModel validates a single search model
func (se *SearchEngine) validateModel(model config.SearchModel) error {
	if model.Name == "" {
		return NewSearchError("SearchEngine", "validateModel", "model name is required", "", nil)
	}
	if model.Collection == "" {
		return NewSearchError("SearchEngine", "validateModel", "model collection is required", model.Name, nil)
	}
	if model.DefaultTopK < 0 || model.DefaultTopK > MaxTopK {
		return NewSearchError("SearchEngine", "validateModel", "invalid default top K", model.Name, nil)
	}
	if model.MaxTopK < model.DefaultTopK {
		return NewSearchError("SearchEngine", "validateModel", "max top K must be >= default top K", model.Name, nil)
	}
	return nil
}

// ============================================================================
// QUERY VALIDATION AND PROCESSING
// ============================================================================

// validateQuery validates a search query
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

// processQuery processes and normalizes a search query
func (se *SearchEngine) processQuery(query string) string {
	// Trim whitespace and normalize
	processed := strings.TrimSpace(query)

	// Enhanced query processing
	processed = strings.ToLower(processed)

	// Remove common stop words
	stopWords := []string{"the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by"}
	for _, stopWord := range stopWords {
		processed = strings.ReplaceAll(processed, " "+stopWord+" ", " ")
	}

	// Clean up multiple spaces
	processed = strings.Join(strings.Fields(processed), " ")

	return processed
}

// ============================================================================
// CORE SEARCH OPERATIONS - ENHANCED
// ============================================================================

// IngestDocument ingests a document into the vector database with enhanced validation
func (se *SearchEngine) IngestDocument(ctx context.Context, docID, content string, metadata map[string]interface{}) error {
	se.mu.RLock()
	defer se.mu.RUnlock()

	// Validate inputs
	if docID == "" {
		return NewSearchError("SearchEngine", "IngestDocument", "document ID cannot be empty", "", nil)
	}
	if content == "" {
		return NewSearchError("SearchEngine", "IngestDocument", "content cannot be empty", docID, nil)
	}

	// Get the first available model's collection
	collection, err := se.getFirstCollection()
	if err != nil {
		return err
	}

	// Generate embedding with timeout
	embedCtx, cancel := context.WithTimeout(ctx, DefaultSearchTimeout)
	defer cancel()

	vector, err := se.embedder.Embed(content)
	if err != nil {
		return NewSearchError("SearchEngine", "IngestDocument", "failed to generate embedding", docID, err)
	}

	// Prepare metadata
	preparedMetadata := se.prepareMetadata(content, metadata)

	// Upsert into database
	if err := se.db.Upsert(embedCtx, collection, docID, vector, preparedMetadata); err != nil {
		return NewSearchError("SearchEngine", "IngestDocument", "failed to upsert document", docID, err)
	}

	return nil
}

// SearchModels performs vector similarity search across specified models with enhanced error handling
func (se *SearchEngine) SearchModels(ctx context.Context, query string, topKPerModel int, modelNames ...string) (map[string][]databases.SearchResult, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	// Validate and process query
	if err := se.validateQuery(query); err != nil {
		return nil, err
	}
	processedQuery := se.processQuery(query)

	// Validate topK parameter
	if topKPerModel <= 0 {
		topKPerModel = DefaultTopK
	}
	if topKPerModel > MaxTopK {
		topKPerModel = MaxTopK
	}

	// Determine which models to search
	modelsToSearch, err := se.determineModelsToSearch(modelNames)
	if err != nil {
		return nil, err
	}

	// Generate embedding once for all models
	embedCtx, cancel := context.WithTimeout(ctx, DefaultSearchTimeout)
	defer cancel()

	vector, err := se.embedder.Embed(processedQuery)
	if err != nil {
		return nil, NewSearchError("SearchEngine", "SearchModels", "failed to generate embedding", processedQuery, err)
	}

	// Search each model
	results := make(map[string][]databases.SearchResult, len(modelsToSearch))
	for _, modelName := range modelsToSearch {
		modelResults, err := se.searchModel(embedCtx, modelName, vector, topKPerModel)
		if err != nil {
			// Log error but continue with other models
			continue
		}
		if len(modelResults) > 0 {
			results[modelName] = modelResults
		}
	}

	return results, nil
}

// Search performs vector similarity search and returns flattened results with enhanced processing
func (se *SearchEngine) Search(ctx context.Context, query string, limit int) ([]databases.SearchResult, error) {
	// Validate limit
	if limit <= 0 {
		limit = DefaultTopK
	}
	if limit > MaxTopK {
		limit = MaxTopK
	}

	// Use SearchModels with proper context
	results, err := se.SearchModels(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	// Flatten and enhance results
	flatResults := se.flattenResults(results, limit)

	return flatResults, nil
}

// ============================================================================
// HELPER METHODS - PRIVATE AND OPTIMIZED
// ============================================================================

// getFirstCollection returns the first available model's collection
func (se *SearchEngine) getFirstCollection() (string, error) {
	if len(se.models) == 0 {
		return "", NewSearchError("SearchEngine", "getFirstCollection", "no models configured", "", nil)
	}

	for _, model := range se.models {
		return model.Collection, nil
	}

	return "", NewSearchError("SearchEngine", "getFirstCollection", "no valid collections found", "", nil)
}

// determineModelsToSearch determines which models to search based on input
func (se *SearchEngine) determineModelsToSearch(modelNames []string) ([]string, error) {
	if len(modelNames) == 0 {
		// Search all models
		allModels := make([]string, 0, len(se.models))
		for modelName := range se.models {
			allModels = append(allModels, modelName)
		}
		return allModels, nil
	}

	// Validate specified models
	validModels := make([]string, 0, len(modelNames))
	for _, modelName := range modelNames {
		if _, exists := se.models[modelName]; exists {
			validModels = append(validModels, modelName)
		}
		// Note: We silently skip invalid models rather than failing
	}

	if len(validModels) == 0 {
		return nil, NewSearchError("SearchEngine", "determineModelsToSearch", "no valid models specified", "", nil)
	}

	return validModels, nil
}

// searchModel searches a single model
func (se *SearchEngine) searchModel(ctx context.Context, modelName string, vector []float32, topK int) ([]databases.SearchResult, error) {
	model, exists := se.models[modelName]
	if !exists {
		return nil, NewSearchError("SearchEngine", "searchModel", "model not found", modelName, nil)
	}

	// Search the model
	modelResults, err := se.db.Search(ctx, model.Collection, vector, topK)
	if err != nil {
		return nil, NewSearchError("SearchEngine", "searchModel", "database search failed", modelName, err)
	}

	// Convert and enhance results
	convertedResults := make([]databases.SearchResult, len(modelResults))
	for i, result := range modelResults {
		convertedResults[i] = databases.SearchResult{
			ID:        result.ID,
			Score:     result.Score,
			Content:   result.Content,
			Vector:    result.Vector,
			Metadata:  result.Metadata,
			ModelName: result.ModelName,
		}
	}

	return convertedResults, nil
}

// flattenResults flattens model results and adds source information
func (se *SearchEngine) flattenResults(results map[string][]databases.SearchResult, limit int) []databases.SearchResult {
	flatResults := make([]databases.SearchResult, 0, limit)

	for modelName, modelResults := range results {
		for _, result := range modelResults {
			// Add model name to metadata for source tracking
			if result.Metadata == nil {
				result.Metadata = make(map[string]interface{})
			}
			result.Metadata["search_source"] = modelName
			flatResults = append(flatResults, result)

			// Respect limit
			if len(flatResults) >= limit {
				return flatResults
			}
		}
	}

	return flatResults
}

// prepareMetadata prepares metadata for document ingestion
func (se *SearchEngine) prepareMetadata(content string, metadata map[string]interface{}) map[string]interface{} {
	prepared := make(map[string]interface{})

	// Add content
	prepared["content"] = content

	// Add existing metadata
	{ // Always iterate
		for k, v := range metadata {
			prepared[k] = v
		}
	}

	// Add ingestion timestamp
	prepared["ingested_at"] = time.Now().Unix()

	return prepared
}

// ============================================================================
// HEALTH AND STATUS METHODS
// ============================================================================

// GetStatus returns detailed status information
func (se *SearchEngine) GetStatus() map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()

	modelNames := make([]string, 0, len(se.models))
	for name := range se.models {
		modelNames = append(modelNames, name)
	}

	return map[string]interface{}{
		"models":      modelNames,
		"model_count": len(se.models),
		"config":      se.config,
	}
}
