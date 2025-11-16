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
	models   map[string]config.SearchModel
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
		models:   make(map[string]config.SearchModel),
	}

	if err := engine.initializeModels(); err != nil {
		return nil, err
	}

	return engine, nil
}

func (se *SearchEngine) initializeModels() error {
	if len(se.config.Models) == 0 {
		return NewSearchError("SearchEngine", "initializeModels", "no search models configured", "", nil)
	}

	for _, model := range se.config.Models {
		if err := se.validateModel(model); err != nil {
			return err
		}

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

	// Get collection name from metadata (store_name field)
	// This should always be set by DocumentStore when indexing
	collection := se.getCollectionFromMetadata(metadata)
	if collection == "" {
		var err error
		collection, err = se.getFirstCollection()
		if err != nil {
			return err
		}
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

func (se *SearchEngine) SearchModels(ctx context.Context, query string, topKPerModel int, modelNames ...string) (map[string][]databases.SearchResult, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	if err := se.validateQuery(query); err != nil {
		return nil, err
	}
	processedQuery := se.processQuery(query)

	// Use config TopK as default if not specified
	if topKPerModel <= 0 {
		topKPerModel = se.config.TopK
		if topKPerModel <= 0 {
			topKPerModel = DefaultTopK
		}
	}
	if topKPerModel > MaxTopK {
		topKPerModel = MaxTopK
	}

	modelsToSearch, err := se.determineModelsToSearch(modelNames)
	if err != nil {
		return nil, err
	}

	embedCtx, cancel := context.WithTimeout(ctx, DefaultSearchTimeout)
	defer cancel()

	vector, err := se.embedder.Embed(processedQuery)
	if err != nil {
		return nil, NewSearchError("SearchEngine", "SearchModels", "failed to generate embedding", processedQuery, err)
	}

	results := make(map[string][]databases.SearchResult, len(modelsToSearch))
	for _, modelName := range modelsToSearch {
		modelResults, err := se.searchModel(embedCtx, modelName, vector, topKPerModel)
		if err != nil {

			continue
		}

		// Filter by threshold
		if se.config.Threshold > 0 {
			filtered := make([]databases.SearchResult, 0, len(modelResults))
			for _, result := range modelResults {
				if result.Score >= se.config.Threshold {
					filtered = append(filtered, result)
				}
			}
			modelResults = filtered
		}

		if len(modelResults) > 0 {
			results[modelName] = modelResults
		}
	}

	return results, nil
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

	collection := se.getCollectionFromFilter(filter)
	if collection == "" {
		collection, err = se.getFirstCollection()
		if err != nil {
			return nil, err
		}
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

	// Get collection from filter (e.g., store_name), fallback to first collection
	collection := se.getCollectionFromFilter(filter)
	if collection == "" {
		var err error
		collection, err = se.getFirstCollection()
		if err != nil {
			return err
		}
	}

	return se.db.DeleteByFilter(ctx, collection, filter)
}

func (se *SearchEngine) getFirstCollection() (string, error) {
	if len(se.models) == 0 {
		return "", NewSearchError("SearchEngine", "getFirstCollection", "no models configured", "", nil)
	}

	for _, model := range se.models {
		return model.Collection, nil
	}

	return "", NewSearchError("SearchEngine", "getFirstCollection", "no valid collections found", "", nil)
}

func (se *SearchEngine) getCollectionFromMetadata(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}

	// Prefer explicit collection name in metadata
	if collection, ok := metadata["collection"].(string); ok && collection != "" {
		return collection
	}

	// Fallback to store_name for backward compatibility
	if storeName, ok := metadata["store_name"].(string); ok && storeName != "" {
		return storeName
	}

	return ""
}

func (se *SearchEngine) getCollectionFromFilter(filter map[string]interface{}) string {
	if filter == nil {
		return ""
	}

	// Check for explicit collection name first
	if collection, ok := filter["collection"].(string); ok && collection != "" {
		return collection
	}

	// Fall back to store_name (for backward compatibility)
	if storeName, ok := filter["store_name"].(string); ok && storeName != "" {
		return storeName
	}

	return ""
}

func (se *SearchEngine) determineModelsToSearch(modelNames []string) ([]string, error) {
	if len(modelNames) == 0 {

		allModels := make([]string, 0, len(se.models))
		for modelName := range se.models {
			allModels = append(allModels, modelName)
		}
		return allModels, nil
	}

	validModels := make([]string, 0, len(modelNames))
	for _, modelName := range modelNames {
		if _, exists := se.models[modelName]; exists {
			validModels = append(validModels, modelName)
		}

	}

	if len(validModels) == 0 {
		return nil, NewSearchError("SearchEngine", "determineModelsToSearch", "no valid models specified", "", nil)
	}

	return validModels, nil
}

func (se *SearchEngine) searchModel(ctx context.Context, modelName string, vector []float32, topK int) ([]databases.SearchResult, error) {
	model, exists := se.models[modelName]
	if !exists {
		return nil, NewSearchError("SearchEngine", "searchModel", "model not found", modelName, nil)
	}

	modelResults, err := se.db.Search(ctx, model.Collection, vector, topK)
	if err != nil {
		return nil, NewSearchError("SearchEngine", "searchModel", "database search failed", modelName, err)
	}

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

func (se *SearchEngine) prepareMetadata(content string, metadata map[string]interface{}) map[string]interface{} {
	prepared := make(map[string]interface{})

	prepared["content"] = content

	{
		for k, v := range metadata {
			prepared[k] = v
		}
	}

	prepared["ingested_at"] = time.Now().Unix()

	return prepared
}

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
