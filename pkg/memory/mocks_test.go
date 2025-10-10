package memory

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/llms"
)

// ============================================================================
// MOCK DATABASE PROVIDER
// ============================================================================

type MockDatabaseProvider struct {
	storage    map[string]map[string]*databases.SearchResult // collection -> id -> result
	searchFunc func(ctx context.Context, collection string, vector []float32, topK int, filter map[string]interface{}) ([]databases.SearchResult, error)
}

func NewMockDatabaseProvider() *MockDatabaseProvider {
	return &MockDatabaseProvider{
		storage: make(map[string]map[string]*databases.SearchResult),
	}
}

func (m *MockDatabaseProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	if m.storage[collection] == nil {
		m.storage[collection] = make(map[string]*databases.SearchResult)
	}

	// Store with content from metadata
	content := ""
	if c, ok := metadata["content"].(string); ok {
		content = c
	}

	m.storage[collection][id] = &databases.SearchResult{
		ID:       id,
		Vector:   vector,
		Metadata: metadata,
		Content:  content,
		Score:    1.0,
	}

	return nil
}

func (m *MockDatabaseProvider) Search(ctx context.Context, collection string, vector []float32, topK int) ([]databases.SearchResult, error) {
	return m.SearchWithFilter(ctx, collection, vector, topK, nil)
}

func (m *MockDatabaseProvider) SearchWithFilter(ctx context.Context, collection string, vector []float32, topK int, filter map[string]interface{}) ([]databases.SearchResult, error) {
	// Use custom search function if provided
	if m.searchFunc != nil {
		return m.searchFunc(ctx, collection, vector, topK, filter)
	}

	// Default: return all results in collection that match filter
	results := []databases.SearchResult{}

	items, exists := m.storage[collection]
	if !exists {
		return results, nil
	}

	for _, item := range items {
		// Apply filter if provided
		if len(filter) > 0 {
			match := true
			for filterKey, filterValue := range filter {
				if metadataValue, ok := item.Metadata[filterKey]; !ok || fmt.Sprintf("%v", metadataValue) != fmt.Sprintf("%v", filterValue) {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		results = append(results, *item)

		if len(results) >= topK {
			break
		}
	}

	return results, nil
}

func (m *MockDatabaseProvider) Delete(ctx context.Context, collection string, id string) error {
	if m.storage[collection] != nil {
		delete(m.storage[collection], id)
	}
	return nil
}

func (m *MockDatabaseProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]interface{}) error {
	items, exists := m.storage[collection]
	if !exists {
		return nil
	}

	// Find and delete matching items
	for id, item := range items {
		match := true
		for filterKey, filterValue := range filter {
			if metadataValue, ok := item.Metadata[filterKey]; !ok || fmt.Sprintf("%v", metadataValue) != fmt.Sprintf("%v", filterValue) {
				match = false
				break
			}
		}
		if match {
			delete(items, id)
		}
	}

	return nil
}

func (m *MockDatabaseProvider) CreateCollection(ctx context.Context, collection string, vectorSize uint64) error {
	if m.storage[collection] == nil {
		m.storage[collection] = make(map[string]*databases.SearchResult)
	}
	return nil
}

func (m *MockDatabaseProvider) DeleteCollection(ctx context.Context, collection string) error {
	delete(m.storage, collection)
	return nil
}

func (m *MockDatabaseProvider) Close() error {
	return nil
}

// Helper methods for testing
func (m *MockDatabaseProvider) GetStoredCount(collection string) int {
	if m.storage[collection] == nil {
		return 0
	}
	return len(m.storage[collection])
}

func (m *MockDatabaseProvider) SetSearchFunc(fn func(ctx context.Context, collection string, vector []float32, topK int, filter map[string]interface{}) ([]databases.SearchResult, error)) {
	m.searchFunc = fn
}

// ============================================================================
// MOCK EMBEDDER PROVIDER
// ============================================================================

type MockEmbedderProvider struct {
	embedFunc func(text string) ([]float32, error)
}

func NewMockEmbedderProvider() *MockEmbedderProvider {
	return &MockEmbedderProvider{
		embedFunc: func(text string) ([]float32, error) {
			// Default: simple hash-based embedding for testing
			hash := 0
			for _, c := range text {
				hash = hash*31 + int(c)
			}
			// Generate a simple 3-dimensional vector
			return []float32{float32(hash % 100), float32((hash / 100) % 100), float32((hash / 10000) % 100)}, nil
		},
	}
}

func (m *MockEmbedderProvider) Embed(text string) ([]float32, error) {
	return m.embedFunc(text)
}

func (m *MockEmbedderProvider) GetModelName() string {
	return "mock-embedder"
}

func (m *MockEmbedderProvider) GetDimension() int {
	return 3
}

func (m *MockEmbedderProvider) Close() error {
	return nil
}

// Helper methods for testing
func (m *MockEmbedderProvider) SetEmbedFunc(fn func(text string) ([]float32, error)) {
	m.embedFunc = fn
}

// ============================================================================
// MOCK SUMMARIZER (for working memory tests)
// ============================================================================

type MockSummarizer struct {
	summaries []string
}

func NewMockSummarizer() *MockSummarizer {
	return &MockSummarizer{
		summaries: []string{},
	}
}

func (m *MockSummarizer) SummarizeConversation(ctx context.Context, messages []llms.Message) (string, error) {
	summary := fmt.Sprintf("Summary of %d messages", len(messages))
	m.summaries = append(m.summaries, summary)
	return summary, nil
}

func (m *MockSummarizer) GetSummaryCount() int {
	return len(m.summaries)
}
