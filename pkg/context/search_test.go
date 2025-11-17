package context

import (
	"context"
	"fmt"
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
)

type mockDatabaseProvider struct {
	upsertFunc func(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error
	searchFunc func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error)
}

func (m *mockDatabaseProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, collection, id, vector, metadata)
	}
	return nil
}

func (m *mockDatabaseProvider) Search(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, collection, vector, limit)
	}
	return []databases.SearchResult{}, nil
}

func (m *mockDatabaseProvider) Delete(ctx context.Context, collection string, id string) error {
	return nil
}

func (m *mockDatabaseProvider) SearchWithFilter(ctx context.Context, collection string, vector []float32, topK int, filter map[string]interface{}) ([]databases.SearchResult, error) {

	return m.Search(ctx, collection, vector, topK)
}

func (m *mockDatabaseProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]interface{}) error {
	return nil
}

func (m *mockDatabaseProvider) CreateCollection(ctx context.Context, collection string, vectorSize uint64) error {
	return nil
}

func (m *mockDatabaseProvider) DeleteCollection(ctx context.Context, collection string) error {
	return nil
}

func (m *mockDatabaseProvider) HybridSearch(ctx context.Context, collection string, query string, vector []float32, topK int, filter map[string]interface{}, alpha float32) ([]databases.SearchResult, error) {
	// Mock hybrid search - just use regular search
	return m.SearchWithFilter(ctx, collection, vector, topK, filter)
}

func (m *mockDatabaseProvider) Close() error {
	return nil
}

type mockEmbedderProvider struct {
	embedFunc func(text string) ([]float32, error)
}

func (m *mockEmbedderProvider) Embed(text string) ([]float32, error) {
	if m.embedFunc != nil {
		return m.embedFunc(text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *mockEmbedderProvider) GetDimension() int {
	return 3
}

func (m *mockEmbedderProvider) GetModelName() string {
	return "test-model"
}

func (m *mockEmbedderProvider) Close() error {
	return nil
}

func TestNewSearchEngine(t *testing.T) {
	tests := []struct {
		name         string
		db           databases.DatabaseProvider
		embedder     embedders.EmbedderProvider
		searchConfig config.SearchConfig
		wantError    bool
	}{
		{
			name:         "valid_configuration",
			db:           &mockDatabaseProvider{},
			embedder:     &mockEmbedderProvider{},
			searchConfig: config.SearchConfig{},
			wantError:    false,
		},
		{
			name:         "nil_database_provider",
			db:           nil,
			embedder:     &mockEmbedderProvider{},
			searchConfig: config.SearchConfig{},
			wantError:    true,
		},
		{
			name:         "nil_embedder_provider",
			db:           &mockDatabaseProvider{},
			embedder:     nil,
			searchConfig: config.SearchConfig{},
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewSearchEngine(tt.db, tt.embedder, tt.searchConfig, nil)

			if tt.wantError {
				if err == nil {
					t.Error("NewSearchEngine() expected error, got nil")
				}
				if engine != nil {
					t.Error("NewSearchEngine() expected nil engine on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewSearchEngine() error = %v, want nil", err)
				}
				if engine == nil {
					t.Error("NewSearchEngine() returned nil engine")
				}
			}
		})
	}
}

func TestSearchEngine_IngestDocument(t *testing.T) {

	var upsertCalled bool
	var upsertCollection, upsertID string
	var upsertVector []float32
	var upsertMetadata map[string]interface{}

	mockDB := &mockDatabaseProvider{
		upsertFunc: func(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
			upsertCalled = true
			upsertCollection = collection
			upsertID = id
			upsertVector = vector
			upsertMetadata = metadata
			return nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{
		embedFunc: func(text string) ([]float32, error) {
			return []float32{0.1, 0.2, 0.3}, nil
		},
	}

	searchConfig := config.SearchConfig{}

	engine, err := NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	tests := []struct {
		name      string
		docID     string
		content   string
		metadata  map[string]interface{}
		wantError bool
	}{
		{
			name:      "valid_document",
			docID:     "doc-123",
			content:   "This is test content",
			metadata:  map[string]interface{}{"source": "test", "store_name": "test-store"},
			wantError: false,
		},
		{
			name:      "empty_document_id",
			docID:     "",
			content:   "This is test content",
			metadata:  nil,
			wantError: true,
		},
		{
			name:      "empty_content",
			docID:     "doc-123",
			content:   "",
			metadata:  nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			upsertCalled = false
			upsertCollection = ""
			upsertID = ""
			upsertVector = nil
			upsertMetadata = nil

			ctx := context.Background()
			err := engine.IngestDocument(ctx, tt.docID, tt.content, tt.metadata)

			if tt.wantError {
				if err == nil {
					t.Error("IngestDocument() expected error, got nil")
				}
				if upsertCalled {
					t.Error("IngestDocument() should not call upsert on error")
				}
			} else {
				if err != nil {
					t.Errorf("IngestDocument() error = %v, want nil", err)
				}
				if !upsertCalled {
					t.Error("IngestDocument() should call upsert on success")
				}
				// Collection comes from metadata store_name or collection field
				if upsertCollection == "" {
					t.Error("IngestDocument() collection should not be empty")
				}
				if upsertID != tt.docID {
					t.Errorf("IngestDocument() ID = %v, want %v", upsertID, tt.docID)
				}
				if len(upsertVector) != 3 {
					t.Errorf("IngestDocument() vector length = %v, want 3", len(upsertVector))
				}
				if upsertMetadata["content"] != tt.content {
					t.Errorf("IngestDocument() metadata content = %v, want %v", upsertMetadata["content"], tt.content)
				}
			}
		})
	}
}

func TestSearchEngine_Search(t *testing.T) {

	mockDB := &mockDatabaseProvider{
		searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
			results := make([]databases.SearchResult, limit)
			for i := 0; i < limit; i++ {
				results[i] = databases.SearchResult{
					ID:      "result-" + string(rune('0'+i)),
					Score:   0.9 - float32(i)*0.1,
					Content: "Test content " + string(rune('0'+i)),
					Vector:  []float32{0.1, 0.2, 0.3},
					Metadata: map[string]interface{}{
						"source": "test",
					},
				}
			}
			return results, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{
		embedFunc: func(text string) ([]float32, error) {
			return []float32{0.1, 0.2, 0.3}, nil
		},
	}

	searchConfig := config.SearchConfig{}

	engine, err := NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	tests := []struct {
		name          string
		query         string
		limit         int
		wantError     bool
		expectedCount int
	}{
		{
			name:          "valid_search",
			query:         "test query",
			limit:         5,
			wantError:     false,
			expectedCount: 5,
		},
		{
			name:      "empty_query",
			query:     "",
			limit:     5,
			wantError: true,
		},
		{
			name:      "query_too_short",
			query:     "",
			limit:     5,
			wantError: true,
		},
		{
			name:      "query_too_long",
			query:     string(make([]byte, MaxQueryLength+1)),
			limit:     5,
			wantError: true,
		},
		{
			name:          "zero_limit",
			query:         "test query",
			limit:         0,
			wantError:     false,
			expectedCount: DefaultTopK,
		},
		{
			name:          "negative_limit",
			query:         "test query",
			limit:         -1,
			wantError:     false,
			expectedCount: DefaultTopK,
		},
		{
			name:          "limit_too_high",
			query:         "test query",
			limit:         200,
			wantError:     false,
			expectedCount: MaxTopK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// Search requires a collection in filter
			filter := map[string]interface{}{
				"collection": "test-collection",
			}
			results, err := engine.SearchWithFilter(ctx, tt.query, tt.limit, filter)

			if tt.wantError {
				if err == nil {
					t.Error("SearchWithFilter() expected error, got nil")
				}
				if results != nil {
					t.Error("SearchWithFilter() expected nil results on error")
				}
			} else {
				if err != nil {
					t.Errorf("SearchWithFilter() error = %v, want nil", err)
				}
				if results == nil {
					t.Error("SearchWithFilter() returned nil results")
				}
				if len(results) != tt.expectedCount {
					t.Errorf("SearchWithFilter() results length = %v, want %v", len(results), tt.expectedCount)
				}
			}
		})
	}
}

func TestSearchEngine_GetStatus(t *testing.T) {
	searchConfig := config.SearchConfig{
		TopK:      10,
		Threshold: 0.7,
	}

	engine, err := NewSearchEngine(&mockDatabaseProvider{}, &mockEmbedderProvider{}, searchConfig, nil)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	status := engine.GetStatus()

	if status == nil {
		t.Fatal("GetStatus() returned nil status")
	}

	config, ok := status["config"].(config.SearchConfig)
	if !ok {
		t.Error("GetStatus() config should be config.SearchConfig")
	}
	if config.TopK != 10 {
		t.Errorf("GetStatus() config TopK = %v, want 10", config.TopK)
	}
}

func TestSearchEngine_QueryValidation(t *testing.T) {
	searchConfig := config.SearchConfig{}

	engine, err := NewSearchEngine(&mockDatabaseProvider{}, &mockEmbedderProvider{}, searchConfig, nil)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	tests := []struct {
		name      string
		query     string
		wantError bool
	}{
		{
			name:      "valid_query",
			query:     "This is a valid query",
			wantError: false,
		},
		{
			name:      "empty_query",
			query:     "",
			wantError: true,
		},
		{
			name:      "query_too_short",
			query:     "",
			wantError: true,
		},
		{
			name:      "query_too_long",
			query:     string(make([]byte, MaxQueryLength+1)),
			wantError: true,
		},
		{
			name:      "query_with_whitespace",
			query:     "  query with spaces  ",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// Search requires a collection in filter
			filter := map[string]interface{}{
				"collection": "test-collection",
			}
			_, err := engine.SearchWithFilter(ctx, tt.query, 5, filter)

			if tt.wantError {
				if err == nil {
					t.Error("SearchWithFilter() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SearchWithFilter() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestSearchEngine_QueryProcessing(t *testing.T) {
	searchConfig := config.SearchConfig{}

	engine, err := NewSearchEngine(&mockDatabaseProvider{}, &mockEmbedderProvider{}, searchConfig, nil)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "query_with_whitespace",
			query:    "  Hello World  ",
			expected: "hello world",
		},
		{
			name:     "query_with_stop_words",
			query:    "the quick brown fox",
			expected: "quick brown fox",
		},
		{
			name:     "query_with_multiple_spaces",
			query:    "hello    world",
			expected: "hello world",
		},
		{
			name:     "query_with_mixed_case",
			query:    "Hello WORLD",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// Search requires a collection in filter
			filter := map[string]interface{}{
				"collection": "test-collection",
			}
			_, err := engine.SearchWithFilter(ctx, tt.query, 5, filter)
			if err != nil {
				t.Errorf("SearchWithFilter() error = %v, want nil", err)
			}
		})
	}
}

func TestSearchEngine_Concurrency(t *testing.T) {
	searchConfig := config.SearchConfig{}

	engine, err := NewSearchEngine(&mockDatabaseProvider{}, &mockEmbedderProvider{}, searchConfig, nil)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	done := make(chan bool, 3)

	go func() {
		for i := 0; i < 5; i++ {
			ctx := context.Background()
			filter := map[string]interface{}{
				"collection": "test-collection",
			}
			_, err := engine.SearchWithFilter(ctx, "test query", 5, filter)
			if err != nil {
				t.Errorf("SearchWithFilter() error = %v", err)
			}
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 5; i++ {
			ctx := context.Background()
			metadata := map[string]interface{}{
				"store_name": "test-store",
			}
			err := engine.IngestDocument(ctx, "doc-"+string(rune('0'+i)), "content", metadata)
			if err != nil {
				t.Errorf("IngestDocument() error = %v", err)
			}
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 5; i++ {
			_ = engine.GetStatus()
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}

func TestSearchError_Error(t *testing.T) {
	tests := []struct {
		name      string
		component string
		operation string
		message   string
		query     string
		err       error
		expected  string
	}{
		{
			name:      "error_with_wrapped_error",
			component: "SearchEngine",
			operation: "Search",
			message:   "search failed",
			query:     "test query",
			err:       fmt.Errorf("database error"),
			expected:  "[SearchEngine:Search] search failed (query: \"test query\"): database error",
		},
		{
			name:      "error_without_wrapped_error",
			component: "SearchEngine",
			operation: "Search",
			message:   "search failed",
			query:     "test query",
			err:       nil,
			expected:  "[SearchEngine:Search] search failed (query: \"test query\")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchErr := NewSearchError(tt.component, tt.operation, tt.message, tt.query, tt.err)
			errorStr := searchErr.Error()

			if errorStr != tt.expected {
				t.Errorf("SearchError.Error() = %v, want %v", errorStr, tt.expected)
			}
		})
	}
}

func TestSearchError_Unwrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	searchErr := NewSearchError("SearchEngine", "Search", "failed", "query", originalErr)

	unwrapped := searchErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("SearchError.Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestNewSearchError(t *testing.T) {
	component := "SearchEngine"
	operation := "Search"
	message := "test error"
	query := "test query"
	err := fmt.Errorf("wrapped error")

	searchErr := NewSearchError(component, operation, message, query, err)

	if searchErr.Component != component {
		t.Errorf("NewSearchError() Component = %v, want %v", searchErr.Component, component)
	}
	if searchErr.Operation != operation {
		t.Errorf("NewSearchError() Operation = %v, want %v", searchErr.Operation, operation)
	}
	if searchErr.Message != message {
		t.Errorf("NewSearchError() Message = %v, want %v", searchErr.Message, message)
	}
	if searchErr.Query != query {
		t.Errorf("NewSearchError() Query = %v, want %v", searchErr.Query, query)
	}
	if searchErr.Err != err {
		t.Errorf("NewSearchError() Err = %v, want %v", searchErr.Err, err)
	}
}
