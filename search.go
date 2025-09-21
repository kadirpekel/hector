package hector

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/databases"
	"github.com/kadirpekel/hector/embedders"
)

// ============================================================================
// SEARCH OPERATIONS
// ============================================================================

// SearchResult represents a search result from the vector database
type SearchResult = databases.SearchResult

// SearchEngine handles search operations across multiple models
type SearchEngine struct {
	db       databases.VectorDB
	embedder embedders.EmbeddingProvider
	models   map[string]ModelConfig
	config   SearchConfig // Component's own config
	parent   *Agent       // Reference to parent for fallback
}

// NewSearchEngine creates a new search engine
func NewSearchEngine(db databases.VectorDB, embedder embedders.EmbeddingProvider, models map[string]ModelConfig) *SearchEngine {
	return &SearchEngine{
		db:       db,
		embedder: embedder,
		models:   models,
		config:   SearchConfig{}, // Default config
		parent:   nil,            // Will be set by Agent
	}
}

// NewSearchEngineWithConfig creates a new search engine with specific config
func NewSearchEngineWithConfig(db databases.VectorDB, embedder embedders.EmbeddingProvider, models map[string]ModelConfig, config SearchConfig) *SearchEngine {
	return &SearchEngine{
		db:       db,
		embedder: embedder,
		models:   models,
		config:   config,
		parent:   nil, // Will be set by Agent
	}
}

// SetParent sets the parent Agent for fallback access
func (s *SearchEngine) SetParent(parent *Agent) {
	s.parent = parent
}

// GetMaxContextLength gets max context length with class defaults
func (s *SearchEngine) GetMaxContextLength() int {
	if s.config.MaxContextLength > 0 {
		return s.config.MaxContextLength
	}
	return 2000 // Class default
}

// GetContextStrategy gets context strategy with class defaults
func (s *SearchEngine) GetContextStrategy() string {
	if s.config.ContextStrategy != "" {
		return s.config.ContextStrategy
	}
	return "relevance" // Class default
}

// GetEnableReranking gets reranking setting with class defaults
func (s *SearchEngine) GetEnableReranking() bool {
	return s.config.EnableReranking // Use config value or false (class default)
}

// ExtractContextWithFallback builds context using component's fallback config
func (se *SearchEngine) ExtractContextWithFallback(allResults map[string][]SearchResult) []string {
	return se.ExtractContext(allResults, se.GetMaxContextLength())
}

// validateEmbedder checks if embedder is configured for SearchEngine
func (se *SearchEngine) validateEmbedder() error {
	if se.embedder == nil {
		return fmt.Errorf("no embedder configured")
	}
	return nil
}

// SearchModels performs vector similarity search across specified models
// If no modelNames provided, searches all models
func (se *SearchEngine) SearchModels(query string, topKPerModel int, modelNames ...string) (map[string][]SearchResult, error) {
	if err := se.validateEmbedder(); err != nil {
		return nil, err
	}

	// Generate embedding for query once
	vector, err := se.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	results := make(map[string][]SearchResult)

	// Determine which models to search
	var modelsToSearch []string
	if len(modelNames) == 0 {
		// No models specified, search all models
		for modelName := range se.models {
			modelsToSearch = append(modelsToSearch, modelName)
		}
	} else {
		// Search only specified models
		modelsToSearch = modelNames
	}

	// Search each model
	for _, modelName := range modelsToSearch {
		config, exists := se.models[modelName]
		if !exists {
			fmt.Printf("Warning: Model %s not found, skipping\n", modelName)
			continue
		}

		modelResults, err := se.db.Search(context.Background(), config.Collection, vector, topKPerModel)
		if err != nil {
			// Log error but continue with other models
			fmt.Printf("Warning: Failed to search model %s: %v\n", modelName, err)
			continue
		}

		if len(modelResults) > 0 {
			results[modelName] = modelResults
		}
	}

	return results, nil
}

// SearchDocuments performs vector similarity search on a single model
func (se *SearchEngine) SearchDocuments(query string, modelName string, topK int) ([]SearchResult, error) {
	if err := se.validateEmbedder(); err != nil {
		return nil, err
	}

	config, exists := se.models[modelName]
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelName)
	}

	// Generate embedding for query
	vector, err := se.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Search in vector database
	results, err := se.db.Search(context.Background(), config.Collection, vector, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search database: %w", err)
	}

	return results, nil
}

// ExtractContext builds context from search results
// Processes multi-model results with section headers
func (se *SearchEngine) ExtractContext(allResults map[string][]SearchResult, maxContextLength int) []string {
	var context []string
	totalLength := 0

	// Process all models
	for modelName, modelResults := range allResults {
		if len(modelResults) == 0 {
			continue
		}

		// Add model section header (only for multi-model results)
		if len(allResults) > 1 {
			sectionHeader := fmt.Sprintf("%s:", strings.Title(modelName))
			context = append(context, sectionHeader)
			totalLength += len(sectionHeader)
		}

		// Add results for this model
		for _, result := range modelResults {
			contextStr := se.buildContextString(result)

			if totalLength+len(contextStr) > maxContextLength {
				break // Stop if we exceed max context length
			}
			context = append(context, contextStr)
			totalLength += len(contextStr)
		}
	}

	return context
}

// buildContextString builds a context string from search result metadata
func (se *SearchEngine) buildContextString(result SearchResult) string {
	var parts []string

	// Iterate over all metadata and include everything
	for key, value := range result.Metadata {
		if valueStr, ok := value.(string); ok && valueStr != "" {
			// Use title case for the key to make it readable
			formattedKey := strings.Title(key)
			parts = append(parts, fmt.Sprintf("%s: %s", formattedKey, valueStr))
		}
	}

	return strings.Join(parts, "\n")
}

// ExtractSources extracts source information from search results
func (se *SearchEngine) ExtractSources(allResults map[string][]SearchResult) []string {
	sources := make(map[string]bool)

	// Process all models
	for _, modelResults := range allResults {
		for _, result := range modelResults {
			// Look for common source field names
			sourceFields := []string{"source", "url", "reference", "link", "origin"}
			for _, fieldName := range sourceFields {
				if source, exists := result.Metadata[fieldName]; exists {
					if sourceStr, ok := source.(string); ok && sourceStr != "" {
						sources[sourceStr] = true
						break // Found a source, no need to check other fields
					}
				}
			}
		}
	}

	var sourceList []string
	for source := range sources {
		sourceList = append(sourceList, source)
	}
	return sourceList
}

// CalculateConfidence calculates confidence based on search scores
func (se *SearchEngine) CalculateConfidence(allResults map[string][]SearchResult) float64 {
	// Process all models
	if len(allResults) == 0 {
		return 0.0
	}

	totalConfidence := 0.0
	modelCount := 0

	for _, modelResults := range allResults {
		if len(modelResults) > 0 {
			totalConfidence += float64(modelResults[0].Score)
			modelCount++
		}
	}

	if modelCount == 0 {
		return 0.0
	}

	return totalConfidence / float64(modelCount)
}

// FlattenMultiModelResults flattens multi-model results into a single slice
func (se *SearchEngine) FlattenMultiModelResults(allResults map[string][]SearchResult) []SearchResult {
	var flattened []SearchResult
	for _, results := range allResults {
		flattened = append(flattened, results...)
	}
	return flattened
}
