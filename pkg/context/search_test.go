package context

import (
	"context"
	"fmt"
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/protocol"
)

type mockDatabaseProvider struct {
	upsertFunc       func(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error
	searchFunc       func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error)
	hybridSearchFunc func(ctx context.Context, collection string, query string, vector []float32, topK int, filter map[string]interface{}, alpha float32) ([]databases.SearchResult, error)
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
	if m.hybridSearchFunc != nil {
		return m.hybridSearchFunc(ctx, collection, query, vector, topK, filter, alpha)
	}
	// Default: just use regular search
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

func TestSearchWithFilter_HybridMode(t *testing.T) {
	var usedHybridSearch bool
	var capturedAlpha float32

	mockDB := &mockDatabaseProvider{
		hybridSearchFunc: func(ctx context.Context, collection string, query string, vector []float32, topK int, filter map[string]interface{}, alpha float32) ([]databases.SearchResult, error) {
			usedHybridSearch = true
			capturedAlpha = alpha
			return []databases.SearchResult{
				{ID: "doc1", Score: 0.9, Content: "hybrid result"},
			}, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{}

	searchConfig := config.SearchConfig{
		TopK:        10,
		SearchMode:  "hybrid",
		HybridAlpha: 0.6,
	}

	engine, err := NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	ctx := context.Background()
	filter := map[string]interface{}{"collection": "test"}

	results, err := engine.SearchWithFilter(ctx, "test query", 10, filter)

	if err != nil {
		t.Fatalf("SearchWithFilter() error = %v", err)
	}

	if !usedHybridSearch {
		t.Error("Expected hybrid search to be used when search_mode=hybrid")
	}

	if capturedAlpha != 0.6 {
		t.Errorf("Expected alpha=0.6, got %v", capturedAlpha)
	}

	if len(results) == 0 {
		t.Error("Expected results from hybrid search")
	}
}

func TestSearchWithFilter_VectorMode(t *testing.T) {
	var usedVectorSearch bool

	mockDB := &mockDatabaseProvider{
		searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
			usedVectorSearch = true
			return []databases.SearchResult{
				{ID: "doc1", Score: 0.8, Content: "vector result"},
			}, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{}

	searchConfig := config.SearchConfig{
		TopK:       10,
		SearchMode: "vector",
	}

	engine, _ := NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)

	ctx := context.Background()
	filter := map[string]interface{}{"collection": "test"}

	_, err := engine.SearchWithFilter(ctx, "test query", 10, filter)

	if err != nil {
		t.Fatalf("SearchWithFilter() error = %v", err)
	}

	if !usedVectorSearch {
		t.Error("Expected vector search to be used when search_mode=vector")
	}
}

func TestSearchWithFilter_DefaultMode(t *testing.T) {
	var usedVectorSearch bool

	mockDB := &mockDatabaseProvider{
		searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
			usedVectorSearch = true
			return []databases.SearchResult{{ID: "doc1", Score: 0.8}}, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{}

	searchConfig := config.SearchConfig{
		TopK:       10,
		SearchMode: "", // Empty - should default to vector
	}

	engine, _ := NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)

	ctx := context.Background()
	filter := map[string]interface{}{"collection": "test"}

	_, err := engine.SearchWithFilter(ctx, "test query", 10, filter)

	if err != nil {
		t.Fatalf("SearchWithFilter() error = %v", err)
	}

	if !usedVectorSearch {
		t.Error("Expected vector search to be used as default")
	}
}

func TestSearchWithFilter_HybridDefaultAlpha(t *testing.T) {
	var capturedAlpha float32

	mockDB := &mockDatabaseProvider{
		hybridSearchFunc: func(ctx context.Context, collection string, query string, vector []float32, topK int, filter map[string]interface{}, alpha float32) ([]databases.SearchResult, error) {
			capturedAlpha = alpha
			return []databases.SearchResult{{ID: "doc1", Score: 0.9}}, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{}

	searchConfig := config.SearchConfig{
		TopK:        10,
		SearchMode:  "hybrid",
		HybridAlpha: 0, // Not set - should default to 0.5
	}

	engine, _ := NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)

	ctx := context.Background()
	filter := map[string]interface{}{"collection": "test"}

	_, _ = engine.SearchWithFilter(ctx, "test query", 10, filter)

	if capturedAlpha != 0.5 {
		t.Errorf("Expected default alpha=0.5, got %v", capturedAlpha)
	}
}

// Mock reranker for testing
type mockReranker struct {
	rerankFunc func(ctx context.Context, query string, results []databases.SearchResult, limit int) ([]databases.SearchResult, error)
}

func (m *mockReranker) Rerank(ctx context.Context, query string, results []databases.SearchResult, limit int) ([]databases.SearchResult, error) {
	if m.rerankFunc != nil {
		return m.rerankFunc(ctx, query, results, limit)
	}
	// Default: return results in reverse order
	reranked := make([]databases.SearchResult, len(results))
	for i := range results {
		reranked[len(results)-1-i] = results[i]
	}
	return reranked, nil
}

// Mock LLM registry for testing
type mockLLMRegistry struct {
	getLLMFunc func(name string) (llms.LLMProvider, error)
}

func (m *mockLLMRegistry) GetLLM(name string) (llms.LLMProvider, error) {
	if m.getLLMFunc != nil {
		return m.getLLMFunc(name)
	}
	return nil, fmt.Errorf("LLM '%s' not found", name)
}

// Mock LLM provider for reranking tests
type mockLLMProviderForReranking struct{}

func (m *mockLLMProviderForReranking) Generate(ctx context.Context, messages []*pb.Message, tools []llms.ToolDefinition) (string, []*protocol.ToolCall, int, *llms.ThinkingBlock, error) {
	return "", nil, 0, nil, nil
}

func (m *mockLLMProviderForReranking) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []llms.ToolDefinition) (<-chan llms.StreamChunk, error) {
	ch := make(chan llms.StreamChunk, 1)
	close(ch)
	return ch, nil
}

func (m *mockLLMProviderForReranking) GetModelName() string {
	return "test-model"
}

func (m *mockLLMProviderForReranking) GetMaxTokens() int {
	return 4096
}

func (m *mockLLMProviderForReranking) GetTemperature() float64 {
	return 0.7
}

func (m *mockLLMProviderForReranking) GetSupportedInputModes() []string {
	return []string{"text/plain", "application/json"}
}

func (m *mockLLMProviderForReranking) Close() error {
	return nil
}

func TestSearchWithFilter_Reranking(t *testing.T) {
	var rerankerCalled bool
	var rerankerQuery string

	mockReranker := &mockReranker{
		rerankFunc: func(ctx context.Context, query string, results []databases.SearchResult, limit int) ([]databases.SearchResult, error) {
			rerankerCalled = true
			rerankerQuery = query
			// Return reranked results (reverse order for testing)
			reranked := make([]databases.SearchResult, len(results))
			for i := range results {
				reranked[len(results)-1-i] = results[i]
				reranked[len(results)-1-i].Score = float32(len(results)-i) * 0.1 // Update scores
			}
			return reranked, nil
		},
	}

	mockDB := &mockDatabaseProvider{
		searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
			return []databases.SearchResult{
				{ID: "doc1", Score: 0.5, Content: "result 1"},
				{ID: "doc2", Score: 0.4, Content: "result 2"},
				{ID: "doc3", Score: 0.3, Content: "result 3"},
			}, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{}

	// Create a mock LLM registry that returns a mock LLM provider
	mockLLMProvider := &mockLLMProviderForReranking{}
	mockRegistry := &mockLLMRegistry{
		getLLMFunc: func(name string) (llms.LLMProvider, error) {
			if name == "test-llm" {
				return mockLLMProvider, nil
			}
			return nil, fmt.Errorf("LLM '%s' not found", name)
		},
	}

	searchConfig := config.SearchConfig{
		TopK: 10,
		Rerank: &config.RerankConfig{
			Enabled:    boolPtr(true),
			LLM:        "test-llm",
			MaxResults: 20,
		},
	}

	engine, err := NewSearchEngine(mockDB, mockEmbedder, searchConfig, mockRegistry)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	// Manually set the reranker to use our mock for testing
	engine.reranker = mockReranker

	ctx := context.Background()
	filter := map[string]interface{}{"collection": "test"}

	results, err := engine.SearchWithFilter(ctx, "test query", 10, filter)

	if err != nil {
		t.Fatalf("SearchWithFilter() error = %v", err)
	}

	if !rerankerCalled {
		t.Error("Expected reranker to be called")
	}

	if rerankerQuery != "test query" {
		t.Errorf("Expected reranker query='test query', got '%s'", rerankerQuery)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Verify results are reranked (should be in reverse order)
	if results[0].ID != "doc3" {
		t.Errorf("Expected first result to be doc3 (reranked), got %s", results[0].ID)
	}
}

func TestSearchWithFilter_RerankingBeforeThreshold(t *testing.T) {
	// Test that reranking happens BEFORE threshold filtering
	// This is critical: low vector scores should be reranked first, then filtered

	var rerankerCalled bool
	var rerankerReceivedResults int

	mockReranker := &mockReranker{
		rerankFunc: func(ctx context.Context, query string, results []databases.SearchResult, limit int) ([]databases.SearchResult, error) {
			rerankerCalled = true
			rerankerReceivedResults = len(results)
			// Reranker assigns high scores to all results (position-based)
			reranked := make([]databases.SearchResult, len(results))
			for i := range results {
				reranked[i] = results[i]
				// Assign high scores (1.0, 0.95, 0.90, ...) so they pass threshold
				reranked[i].Score = 1.0 - (float32(i) * 0.05)
			}
			return reranked, nil
		},
	}

	mockDB := &mockDatabaseProvider{
		searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
			// Return results with LOW scores (below threshold)
			// These should still be reranked before filtering
			return []databases.SearchResult{
				{ID: "doc1", Score: 0.2, Content: "result 1"}, // Below 0.5 threshold
				{ID: "doc2", Score: 0.3, Content: "result 2"}, // Below 0.5 threshold
				{ID: "doc3", Score: 0.4, Content: "result 3"}, // Below 0.5 threshold
			}, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{}
	mockLLMProvider := &mockLLMProviderForReranking{}
	mockRegistry := &mockLLMRegistry{
		getLLMFunc: func(name string) (llms.LLMProvider, error) {
			if name == "test-llm" {
				return mockLLMProvider, nil
			}
			return nil, fmt.Errorf("LLM '%s' not found", name)
		},
	}

	searchConfig := config.SearchConfig{
		TopK:      10,
		Threshold: 0.5, // Threshold that would filter out low scores
		Rerank: &config.RerankConfig{
			Enabled:    boolPtr(true),
			LLM:        "test-llm",
			MaxResults: 20,
		},
	}

	engine, err := NewSearchEngine(mockDB, mockEmbedder, searchConfig, mockRegistry)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	// Use mock reranker
	engine.reranker = mockReranker

	ctx := context.Background()
	filter := map[string]interface{}{"collection": "test"}

	results, err := engine.SearchWithFilter(ctx, "test query", 10, filter)

	if err != nil {
		t.Fatalf("SearchWithFilter() error = %v", err)
	}

	// Verify reranker was called
	if !rerankerCalled {
		t.Error("Expected reranker to be called before threshold filtering")
	}

	// Verify reranker received all 3 results (not filtered before reranking)
	if rerankerReceivedResults != 3 {
		t.Errorf("Expected reranker to receive 3 results, got %d", rerankerReceivedResults)
	}

	// Verify results passed threshold after reranking (all should have scores >= 0.5)
	if len(results) == 0 {
		t.Error("Expected results after reranking (should pass threshold with high reranked scores)")
	}

	for _, result := range results {
		if result.Score < 0.5 {
			t.Errorf("Result %s has score %.2f below threshold 0.5 (should have been filtered)", result.ID, result.Score)
		}
	}
}

func TestSearchWithFilter_RerankerScoreAssignment(t *testing.T) {
	// Test that reranker assigns position-based scores correctly

	mockReranker := &mockReranker{
		rerankFunc: func(ctx context.Context, query string, results []databases.SearchResult, limit int) ([]databases.SearchResult, error) {
			// Assign position-based scores: 1.0, 0.95, 0.90, ...
			reranked := make([]databases.SearchResult, len(results))
			for i := range results {
				reranked[i] = results[i]
				reranked[i].Score = 1.0 - (float32(i) * 0.05)
				if reranked[i].Score < 0.1 {
					reranked[i].Score = 0.1
				}
			}
			return reranked, nil
		},
	}

	mockDB := &mockDatabaseProvider{
		searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
			return []databases.SearchResult{
				{ID: "doc1", Score: 0.5, Content: "result 1"},
				{ID: "doc2", Score: 0.4, Content: "result 2"},
				{ID: "doc3", Score: 0.3, Content: "result 3"},
			}, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{}
	mockLLMProvider := &mockLLMProviderForReranking{}
	mockRegistry := &mockLLMRegistry{
		getLLMFunc: func(name string) (llms.LLMProvider, error) {
			if name == "test-llm" {
				return mockLLMProvider, nil
			}
			return nil, fmt.Errorf("LLM '%s' not found", name)
		},
	}

	searchConfig := config.SearchConfig{
		TopK: 10,
		Rerank: &config.RerankConfig{
			Enabled:    boolPtr(true),
			LLM:        "test-llm",
			MaxResults: 20,
		},
	}

	engine, err := NewSearchEngine(mockDB, mockEmbedder, searchConfig, mockRegistry)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	engine.reranker = mockReranker

	ctx := context.Background()
	filter := map[string]interface{}{"collection": "test"}

	results, err := engine.SearchWithFilter(ctx, "test query", 10, filter)

	if err != nil {
		t.Fatalf("SearchWithFilter() error = %v", err)
	}

	// Verify scores are position-based
	expectedScores := []float32{1.0, 0.95, 0.90}
	if len(results) != len(expectedScores) {
		t.Fatalf("Expected %d results, got %d", len(expectedScores), len(results))
	}

	for i, result := range results {
		if result.Score != expectedScores[i] {
			t.Errorf("Result %d: expected score %.2f, got %.2f", i, expectedScores[i], result.Score)
		}
	}
}

func TestSearchWithFilter_RerankerErrorHandling(t *testing.T) {
	// Test that reranker errors don't break search - should continue with original results

	mockReranker := &mockReranker{
		rerankFunc: func(ctx context.Context, query string, results []databases.SearchResult, limit int) ([]databases.SearchResult, error) {
			return nil, fmt.Errorf("reranker error")
		},
	}

	mockDB := &mockDatabaseProvider{
		searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
			return []databases.SearchResult{
				{ID: "doc1", Score: 0.8, Content: "result 1"},
				{ID: "doc2", Score: 0.7, Content: "result 2"},
			}, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{}
	mockLLMProvider := &mockLLMProviderForReranking{}
	mockRegistry := &mockLLMRegistry{
		getLLMFunc: func(name string) (llms.LLMProvider, error) {
			if name == "test-llm" {
				return mockLLMProvider, nil
			}
			return nil, fmt.Errorf("LLM '%s' not found", name)
		},
	}

	searchConfig := config.SearchConfig{
		TopK: 10,
		Rerank: &config.RerankConfig{
			Enabled:    boolPtr(true),
			LLM:        "test-llm",
			MaxResults: 20,
		},
	}

	engine, err := NewSearchEngine(mockDB, mockEmbedder, searchConfig, mockRegistry)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	engine.reranker = mockReranker

	ctx := context.Background()
	filter := map[string]interface{}{"collection": "test"}

	// Should not error - should continue with original results
	results, err := engine.SearchWithFilter(ctx, "test query", 10, filter)

	if err != nil {
		t.Fatalf("SearchWithFilter() should not error on reranker failure, got: %v", err)
	}

	// Should still return original results
	if len(results) != 2 {
		t.Errorf("Expected 2 results (original), got %d", len(results))
	}

	// Verify original scores are preserved
	if results[0].Score != 0.8 {
		t.Errorf("Expected first result score 0.8, got %.2f", results[0].Score)
	}
}

func TestSearchWithFilter_ThresholdAfterReranking(t *testing.T) {
	// Test that threshold filtering happens AFTER reranking

	mockReranker := &mockReranker{
		rerankFunc: func(ctx context.Context, query string, results []databases.SearchResult, limit int) ([]databases.SearchResult, error) {
			// Assign scores: some above threshold, some below
			reranked := make([]databases.SearchResult, len(results))
			for i := range results {
				reranked[i] = results[i]
				// Assign scores: 0.9, 0.6, 0.4, 0.3
				reranked[i].Score = 0.9 - (float32(i) * 0.3)
				if reranked[i].Score < 0.1 {
					reranked[i].Score = 0.1
				}
			}
			return reranked, nil
		},
	}

	mockDB := &mockDatabaseProvider{
		searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
			return []databases.SearchResult{
				{ID: "doc1", Score: 0.2, Content: "result 1"},
				{ID: "doc2", Score: 0.2, Content: "result 2"},
				{ID: "doc3", Score: 0.2, Content: "result 3"},
				{ID: "doc4", Score: 0.2, Content: "result 4"},
			}, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{}
	mockLLMProvider := &mockLLMProviderForReranking{}
	mockRegistry := &mockLLMRegistry{
		getLLMFunc: func(name string) (llms.LLMProvider, error) {
			if name == "test-llm" {
				return mockLLMProvider, nil
			}
			return nil, fmt.Errorf("LLM '%s' not found", name)
		},
	}

	searchConfig := config.SearchConfig{
		TopK:      10,
		Threshold: 0.5, // Should filter out results with score < 0.5
		Rerank: &config.RerankConfig{
			Enabled:    boolPtr(true),
			LLM:        "test-llm",
			MaxResults: 20,
		},
	}

	engine, err := NewSearchEngine(mockDB, mockEmbedder, searchConfig, mockRegistry)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	engine.reranker = mockReranker

	ctx := context.Background()
	filter := map[string]interface{}{"collection": "test"}

	results, err := engine.SearchWithFilter(ctx, "test query", 10, filter)

	if err != nil {
		t.Fatalf("SearchWithFilter() error = %v", err)
	}

	// After reranking: scores are 0.9, 0.6, 0.4, 0.3
	// After threshold 0.5: should keep only 0.9 and 0.6
	if len(results) != 2 {
		t.Errorf("Expected 2 results after threshold filtering (scores 0.9 and 0.6), got %d", len(results))
	}

	// Verify all results pass threshold
	for _, result := range results {
		if result.Score < 0.5 {
			t.Errorf("Result %s has score %.2f below threshold 0.5", result.ID, result.Score)
		}
	}

	// Verify correct results are kept (use approximate comparison for floating point)
	if results[0].Score < 0.89 || results[0].Score > 0.91 {
		t.Errorf("Expected first result score ~0.9, got %.2f", results[0].Score)
	}
	if results[1].Score < 0.59 || results[1].Score > 0.61 {
		t.Errorf("Expected second result score ~0.6, got %.2f", results[1].Score)
	}
}

func TestNewSearchEngine_RerankingRequired(t *testing.T) {
	mockDB := &mockDatabaseProvider{}
	mockEmbedder := &mockEmbedderProvider{}

	searchConfig := config.SearchConfig{
		Rerank: &config.RerankConfig{
			Enabled: boolPtr(true),
			LLM:     "test-llm",
		},
	}

	// Test: reranking enabled but no LLM registry provided
	_, err := NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)
	if err == nil {
		t.Error("Expected error when reranking enabled but no LLM registry provided")
	}

	// Test: reranking enabled but rerank.llm is empty
	searchConfig.Rerank.LLM = ""
	_, err = NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)
	if err == nil {
		t.Error("Expected error when reranking enabled but rerank.llm is empty")
	}

	// Test: reranking enabled but LLM not found in registry
	searchConfig.Rerank.LLM = "nonexistent-llm"
	mockRegistry := &mockLLMRegistry{
		getLLMFunc: func(name string) (llms.LLMProvider, error) {
			return nil, fmt.Errorf("LLM '%s' not found", name)
		},
	}
	_, err = NewSearchEngine(mockDB, mockEmbedder, searchConfig, mockRegistry)
	if err == nil {
		t.Error("Expected error when reranking enabled but LLM not found in registry")
	}
}

func boolPtr(b bool) *bool {
	return &b
}
