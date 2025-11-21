package context

import (
	"context"
	"fmt"
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// mockLLMProviderForMultiQuery for testing query expansion
type mockLLMProviderForMultiQuery struct {
	response string
	err      error
}

func (m *mockLLMProviderForMultiQuery) Generate(ctx context.Context, messages []*pb.Message, tools []llms.ToolDefinition) (string, []*protocol.ToolCall, int, error) {
	if m.err != nil {
		return "", nil, 0, m.err
	}
	return m.response, nil, 0, nil
}

func (m *mockLLMProviderForMultiQuery) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []llms.ToolDefinition) (<-chan llms.StreamChunk, error) {
	ch := make(chan llms.StreamChunk, 1)
	if m.err != nil {
		ch <- llms.StreamChunk{Type: "error", Error: m.err}
		close(ch)
		return ch, nil
	}
	ch <- llms.StreamChunk{Type: "content", Text: m.response}
	close(ch)
	return ch, nil
}

func (m *mockLLMProviderForMultiQuery) GetModelName() string {
	return "test-model"
}

func (m *mockLLMProviderForMultiQuery) GetMaxTokens() int {
	return 4096
}

func (m *mockLLMProviderForMultiQuery) GetTemperature() float64 {
	return 0.7
}

func (m *mockLLMProviderForMultiQuery) GetSupportedInputModes() []string {
	return []string{"text/plain", "application/json"}
}

func (m *mockLLMProviderForMultiQuery) Close() error {
	return nil
}

func TestSearchWithMultiQuery_ScoreBoosting(t *testing.T) {
	// Test that max score is used for duplicates (order-independent)

	var searchCount int
	mockDB := &mockDatabaseProvider{
		searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
			searchCount++
			// Simulate different scores for the same document across query variations
			switch searchCount {
			case 1: // Original query
				return []databases.SearchResult{
					{ID: "doc1", Score: 0.8, Content: "content1"},
					{ID: "doc2", Score: 0.7, Content: "content2"},
				}, nil
			case 2: // Variation 1 - doc1 scores lower
				return []databases.SearchResult{
					{ID: "doc1", Score: 0.6, Content: "content1"},
					{ID: "doc3", Score: 0.75, Content: "content3"},
				}, nil
			case 3: // Variation 2 - doc2 scores higher
				return []databases.SearchResult{
					{ID: "doc2", Score: 0.9, Content: "content2"},
					{ID: "doc3", Score: 0.5, Content: "content3"},
				}, nil
			default:
				return []databases.SearchResult{}, nil
			}
		},
	}

	mockEmbedder := &mockEmbedderProvider{
		embedFunc: func(text string) ([]float32, error) {
			return []float32{0.1, 0.2, 0.3}, nil
		},
	}

	// Mock LLM for query expansion
	mockLLM := &mockLLMProviderForMultiQuery{
		response: `["variation 1", "variation 2"]`,
	}

	searchConfig := config.SearchConfig{
		TopK: 10,
		MultiQuery: &config.MultiQueryConfig{
			NumVariations: 2,
		},
	}

	engine, err := NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)
	if err != nil {
		t.Fatalf("NewSearchEngine() error = %v", err)
	}

	// Set query expander
	engine.queryExpander = NewLLMQueryExpander(mockLLM)

	ctx := context.Background()
	filter := map[string]interface{}{"collection": "test"}

	// Execute multi-query search
	results, err := engine.searchWithMultiQuery(ctx, ctx, "test query", "test", 10, filter)
	if err != nil {
		t.Fatalf("searchWithMultiQuery() error = %v", err)
	}

	// Verify results
	if len(results) != 3 {
		t.Errorf("Expected 3 unique documents, got %d", len(results))
	}

	// Verify max score was used for duplicates
	// doc1: max(0.8, 0.6) = 0.8
	// doc2: max(0.7, 0.9) = 0.9
	// doc3: max(0.75, 0.5) = 0.75
	expectedScores := map[string]float32{
		"doc1": 0.8,
		"doc2": 0.9,
		"doc3": 0.75,
	}

	for _, result := range results {
		expectedScore, exists := expectedScores[result.ID]
		if !exists {
			t.Errorf("Unexpected document ID: %s", result.ID)
			continue
		}
		if result.Score != expectedScore {
			t.Errorf("Document %s: expected score %.2f, got %.2f", result.ID, expectedScore, result.Score)
		}
	}

	// Verify results are sorted by score (highest first)
	if len(results) >= 2 {
		if results[0].Score < results[1].Score {
			t.Error("Results should be sorted by score descending")
		}
	}
}

func TestSearchWithMultiQuery_OrderIndependence(t *testing.T) {
	// Test that score boosting is order-independent
	testCases := []struct {
		name    string
		scores1 []float32 // Scores for doc1 across queries
		scores2 []float32 // Scores for doc2 across queries
		want1   float32   // Expected max score for doc1
		want2   float32   // Expected max score for doc2
	}{
		{
			name:    "Ascending order",
			scores1: []float32{0.5, 0.7, 0.9},
			scores2: []float32{0.6, 0.8, 0.85},
			want1:   0.9,
			want2:   0.85,
		},
		{
			name:    "Descending order",
			scores1: []float32{0.9, 0.7, 0.5},
			scores2: []float32{0.85, 0.8, 0.6},
			want1:   0.9,
			want2:   0.85,
		},
		{
			name:    "Random order",
			scores1: []float32{0.7, 0.9, 0.5},
			scores2: []float32{0.8, 0.6, 0.85},
			want1:   0.9,
			want2:   0.85,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var searchCount int
			mockDB := &mockDatabaseProvider{
				searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
					if searchCount >= len(tc.scores1) {
						return []databases.SearchResult{}, nil
					}
					results := []databases.SearchResult{
						{ID: "doc1", Score: tc.scores1[searchCount], Content: "content1"},
						{ID: "doc2", Score: tc.scores2[searchCount], Content: "content2"},
					}
					searchCount++
					return results, nil
				},
			}

			mockEmbedder := &mockEmbedderProvider{}
			mockLLM := &mockLLMProviderForMultiQuery{
				response: `["variation 1", "variation 2"]`,
			}

			searchConfig := config.SearchConfig{TopK: 10}
			engine, _ := NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)
			engine.queryExpander = NewLLMQueryExpander(mockLLM)

			results, err := engine.searchWithMultiQuery(context.Background(), context.Background(), "test", "test", 10, map[string]interface{}{"collection": "test"})
			if err != nil {
				t.Fatalf("searchWithMultiQuery() error = %v", err)
			}

			// Find doc1 and doc2 in results
			var doc1Score, doc2Score float32
			for _, result := range results {
				if result.ID == "doc1" {
					doc1Score = result.Score
				} else if result.ID == "doc2" {
					doc2Score = result.Score
				}
			}

			if doc1Score != tc.want1 {
				t.Errorf("doc1 score = %.2f, want %.2f", doc1Score, tc.want1)
			}
			if doc2Score != tc.want2 {
				t.Errorf("doc2 score = %.2f, want %.2f", doc2Score, tc.want2)
			}
		})
	}
}

func TestSearchWithMultiQuery_QueryExpansionFailure(t *testing.T) {
	// Test fallback when query expansion fails
	mockDB := &mockDatabaseProvider{
		searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
			return []databases.SearchResult{
				{ID: "doc1", Score: 0.8, Content: "content1"},
			}, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{}
	mockLLM := &mockLLMProviderForMultiQuery{
		err: fmt.Errorf("LLM API error"),
	}

	searchConfig := config.SearchConfig{TopK: 10}
	engine, _ := NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)
	engine.queryExpander = NewLLMQueryExpander(mockLLM)

	results, err := engine.searchWithMultiQuery(context.Background(), context.Background(), "test", "test", 10, map[string]interface{}{"collection": "test"})

	// Should not error - falls back to original query
	if err != nil {
		t.Fatalf("searchWithMultiQuery() should not error on expansion failure, got: %v", err)
	}

	// Should still return results from original query
	if len(results) == 0 {
		t.Error("Expected results from fallback query")
	}
}

func TestSearchWithMultiQuery_TopKLimit(t *testing.T) {
	// Test that topK limit is respected
	mockDB := &mockDatabaseProvider{
		searchFunc: func(ctx context.Context, collection string, vector []float32, limit int) ([]databases.SearchResult, error) {
			// Return 10 results
			results := make([]databases.SearchResult, 10)
			for i := 0; i < 10; i++ {
				results[i] = databases.SearchResult{
					ID:      fmt.Sprintf("doc%d", i),
					Score:   0.9 - float32(i)*0.05,
					Content: fmt.Sprintf("content%d", i),
				}
			}
			return results, nil
		},
	}

	mockEmbedder := &mockEmbedderProvider{}
	mockLLM := &mockLLMProviderForMultiQuery{
		response: `["variation 1"]`,
	}

	searchConfig := config.SearchConfig{TopK: 10}
	engine, _ := NewSearchEngine(mockDB, mockEmbedder, searchConfig, nil)
	engine.queryExpander = NewLLMQueryExpander(mockLLM)

	// Request only top 5
	results, err := engine.searchWithMultiQuery(context.Background(), context.Background(), "test", "test", 5, map[string]interface{}{"collection": "test"})

	if err != nil {
		t.Fatalf("searchWithMultiQuery() error = %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results (topK=5), got %d", len(results))
	}

	// Verify top 5 are highest scoring
	for i := 0; i < len(results)-1; i++ {
		if results[i].Score < results[i+1].Score {
			t.Error("Results not properly sorted by score")
		}
	}
}
