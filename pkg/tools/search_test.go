package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
)

func TestNewSearchToolForTesting(t *testing.T) {
	tool := NewSearchToolForTesting()
	if tool == nil {
		t.Fatal("NewSearchToolForTesting() returned nil")
	}

	// Test that the tool has the expected name
	if tool.GetName() != "search" {
		t.Errorf("GetName() = %v, want 'search'", tool.GetName())
	}

	// Test that the tool has a description
	description := tool.GetDescription()
	if description == "" {
		t.Error("GetDescription() should not return empty string")
	}
}

func TestSearchTool_GetInfo(t *testing.T) {
	tool := NewSearchToolForTesting()
	info := tool.GetInfo()

	if info.Name == "" {
		t.Fatal("GetInfo() returned empty name")
	}

	// Verify info structure
	if info.Description == "" {
		t.Error("Expected non-empty description")
	}
	if len(info.Parameters) == 0 {
		t.Error("Expected at least one parameter")
	}

	// Check for required parameters
	hasQueryParam := false
	for _, param := range info.Parameters {
		if param.Name == "query" && param.Required {
			hasQueryParam = true
		}
	}
	if !hasQueryParam {
		t.Error("Expected 'query' parameter to be required")
	}
}

func TestSearchTool_Execute_ValidationOnly(t *testing.T) {
	tool := NewSearchToolForTesting()

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing query parameter",
			args: map[string]interface{}{
				"type": "content",
			},
			wantErr: true,
			errMsg:  "query parameter is required",
		},
		{
			name: "empty query",
			args: map[string]interface{}{
				"query": "",
				"type":  "content",
			},
			wantErr: true,
			errMsg:  "query parameter is required",
		},
		{
			name: "invalid query type",
			args: map[string]interface{}{
				"query": 123,
				"type":  "content",
			},
			wantErr: true,
			errMsg:  "query parameter is required",
		},
		{
			name: "valid query with default type",
			args: map[string]interface{}{
				"query": "test search",
			},
			wantErr: false, // Will fail at document store level, but validation should pass
		},
		{
			name: "valid query with explicit type",
			args: map[string]interface{}{
				"query": "test search",
				"type":  "file",
			},
			wantErr: false,
		},
		{
			name: "valid query with limit",
			args: map[string]interface{}{
				"query": "test search",
				"limit": 5,
			},
			wantErr: false,
		},
		{
			name: "valid query with language filter",
			args: map[string]interface{}{
				"query":    "test search",
				"language": "go",
			},
			wantErr: false,
		},
		{
			name: "valid query with stores array",
			args: map[string]interface{}{
				"query":  "test search",
				"stores": []interface{}{"store1", "store2"},
			},
			wantErr: false,
		},
		{
			name: "valid query with single store string",
			args: map[string]interface{}{
				"query":  "test search",
				"stores": "store1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := tool.Execute(ctx, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errMsg, err)
				}
			} else {
				// For valid parameters, we expect document store errors, not validation errors
				if err != nil && strings.Contains(err.Error(), "query parameter is required") {
					t.Errorf("Expected document store error, got validation error: %v", err)
				}
			}
		})
	}
}

func TestSearchTool_GetStringWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]interface{}
		key          string
		defaultValue string
		expected     string
	}{
		{
			name: "string value exists",
			args: map[string]interface{}{
				"type": "file",
			},
			key:          "type",
			defaultValue: "content",
			expected:     "file",
		},
		{
			name: "string value missing",
			args: map[string]interface{}{
				"other": "value",
			},
			key:          "type",
			defaultValue: "content",
			expected:     "content",
		},
		{
			name: "non-string value",
			args: map[string]interface{}{
				"type": 123,
			},
			key:          "type",
			defaultValue: "content",
			expected:     "content",
		},
		{
			name: "empty string value",
			args: map[string]interface{}{
				"type": "",
			},
			key:          "type",
			defaultValue: "content",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringWithDefault(tt.args, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getStringWithDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSearchTool_GetIntWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]interface{}
		key          string
		defaultValue int
		expected     int
	}{
		{
			name: "int value exists",
			args: map[string]interface{}{
				"limit": 5,
			},
			key:          "limit",
			defaultValue: 10,
			expected:     5,
		},
		{
			name: "float64 value exists",
			args: map[string]interface{}{
				"limit": 5.0,
			},
			key:          "limit",
			defaultValue: 10,
			expected:     5,
		},
		{
			name: "int value missing",
			args: map[string]interface{}{
				"other": "value",
			},
			key:          "limit",
			defaultValue: 10,
			expected:     10,
		},
		{
			name: "non-numeric value",
			args: map[string]interface{}{
				"limit": "not a number",
			},
			key:          "limit",
			defaultValue: 10,
			expected:     10,
		},
		{
			name: "zero int value",
			args: map[string]interface{}{
				"limit": 0,
			},
			key:          "limit",
			defaultValue: 10,
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIntWithDefault(tt.args, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getIntWithDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSearchTool_CreateErrorResponse(t *testing.T) {
	tool := NewSearchToolForTesting()

	message := "Test error message"
	response, err := tool.createErrorResponse(message)

	if err != nil {
		t.Fatalf("createErrorResponse() error = %v, want nil", err)
	}

	// Parse the JSON response
	var searchResponse SearchResponse
	if err := json.Unmarshal([]byte(response), &searchResponse); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify response structure
	if searchResponse.Total != 0 {
		t.Errorf("Expected Total = 0, got %d", searchResponse.Total)
	}
	if len(searchResponse.Results) != 0 {
		t.Errorf("Expected empty Results, got %d items", len(searchResponse.Results))
	}
	if len(searchResponse.StoresUsed) != 0 {
		t.Errorf("Expected empty StoresUsed, got %d items", len(searchResponse.StoresUsed))
	}
	if len(searchResponse.Suggestions) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(searchResponse.Suggestions))
	}
	if searchResponse.Suggestions[0] != message {
		t.Errorf("Expected suggestion '%s', got '%s'", message, searchResponse.Suggestions[0])
	}
}

func TestSearchTool_IsSearchTypeEnabled(t *testing.T) {
	tool := NewSearchToolForTesting()

	tests := []struct {
		name       string
		searchType string
		expected   bool
	}{
		{
			name:       "enabled search type - content",
			searchType: "content",
			expected:   true,
		},
		{
			name:       "enabled search type - file",
			searchType: "file",
			expected:   true,
		},
		{
			name:       "enabled search type - function",
			searchType: "function",
			expected:   true,
		},
		{
			name:       "enabled search type - struct",
			searchType: "struct",
			expected:   true,
		},
		{
			name:       "disabled search type",
			searchType: "disabled",
			expected:   false,
		},
		{
			name:       "empty search type",
			searchType: "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.isSearchTypeEnabled(tt.searchType)
			if result != tt.expected {
				t.Errorf("isSearchTypeEnabled(%q) = %v, want %v", tt.searchType, result, tt.expected)
			}
		})
	}
}

func TestSearchTool_GetStoresToSearch(t *testing.T) {
	tool := NewSearchToolForTesting()

	// Create mock document stores
	availableStores := map[string]*hectorcontext.DocumentStore{
		"store1": {},
		"store2": {},
		"store3": {},
	}

	tests := []struct {
		name            string
		requestedStores []string
		expected        []string
	}{
		{
			name:            "empty requested stores - should return all",
			requestedStores: []string{},
			expected:        []string{"store1", "store2", "store3"}, // Order may vary
		},
		{
			name:            "single matching store",
			requestedStores: []string{"store1"},
			expected:        []string{"store1"},
		},
		{
			name:            "multiple matching stores",
			requestedStores: []string{"store1", "store3"},
			expected:        []string{"store1", "store3"},
		},
		{
			name:            "non-matching store",
			requestedStores: []string{"nonexistent"},
			expected:        []string{},
		},
		{
			name:            "mixed matching and non-matching",
			requestedStores: []string{"store1", "nonexistent", "store3"},
			expected:        []string{"store1", "store3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.getStoresToSearch(tt.requestedStores, availableStores)
			if len(result) != len(tt.expected) {
				t.Errorf("getStoresToSearch() returned %d stores, want %d", len(result), len(tt.expected))
			}

			// For the empty requested stores case, we need to check that all stores are returned
			if len(tt.requestedStores) == 0 {
				if len(result) != 3 {
					t.Errorf("Expected 3 stores for empty request, got %d", len(result))
				}
				// Check that all expected stores are present
				storeMap := make(map[string]bool)
				for _, store := range result {
					storeMap[store] = true
				}
				for _, expected := range []string{"store1", "store2", "store3"} {
					if !storeMap[expected] {
						t.Errorf("Expected store %s to be in results", expected)
					}
				}
			} else {
				// For specific requests, check exact matches
				for i, expected := range tt.expected {
					if i >= len(result) || result[i] != expected {
						t.Errorf("getStoresToSearch()[%d] = %v, want %v", i, result[i], expected)
					}
				}
			}
		})
	}
}

func TestSearchTool_SortResultsByScore(t *testing.T) {
	tool := NewSearchToolForTesting()

	results := []DocumentSearchResult{
		{DocumentID: "doc1", Score: 0.5, Content: "content1"},
		{DocumentID: "doc2", Score: 0.9, Content: "content2"},
		{DocumentID: "doc3", Score: 0.3, Content: "content3"},
		{DocumentID: "doc4", Score: 0.8, Content: "content4"},
	}

	tool.sortResultsByScore(results)

	// Verify results are sorted by score (highest first)
	expectedOrder := []string{"doc2", "doc4", "doc1", "doc3"}
	for i, expected := range expectedOrder {
		if results[i].DocumentID != expected {
			t.Errorf("sortResultsByScore()[%d] = %s, want %s", i, results[i].DocumentID, expected)
		}
	}

	// Verify scores are in descending order
	for i := 1; i < len(results); i++ {
		if results[i-1].Score < results[i].Score {
			t.Errorf("Results not sorted by score: %f < %f", results[i-1].Score, results[i].Score)
		}
	}
}

func TestSearchTool_GenerateSuggestions(t *testing.T) {
	tool := NewSearchToolForTesting()

	req := SearchRequest{
		Query: "test query",
		Type:  "content",
	}

	suggestions := tool.generateSuggestions(req)

	if len(suggestions) == 0 {
		t.Error("Expected at least one suggestion")
	}

	// Verify suggestions contain helpful text
	allSuggestions := strings.Join(suggestions, " ")
	if !strings.Contains(strings.ToLower(allSuggestions), "search") {
		t.Error("Expected suggestions to contain 'search'")
	}
}

func TestSearchTool_MinInt(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{
			name:     "a is smaller",
			a:        5,
			b:        10,
			expected: 5,
		},
		{
			name:     "b is smaller",
			a:        10,
			b:        5,
			expected: 5,
		},
		{
			name:     "equal values",
			a:        5,
			b:        5,
			expected: 5,
		},
		{
			name:     "negative values",
			a:        -5,
			b:        -10,
			expected: -10,
		},
		{
			name:     "zero and positive",
			a:        0,
			b:        5,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := minInt(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("minInt(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestSearchTool_WithCustomConfig(t *testing.T) {
	// Test with custom configuration
	config := &config.SearchToolConfig{
		DocumentStores:     []string{"custom-store"},
		DefaultLimit:       3,
		MaxLimit:           5,
		EnabledSearchTypes: []string{"content", "file"},
	}

	tool := NewSearchTool(config)

	if tool == nil {
		t.Fatal("NewSearchTool() returned nil")
	}

	// Test that custom config is applied
	if tool.GetName() != "search" {
		t.Errorf("GetName() = %v, want 'search'", tool.GetName())
	}

	// Test with nil config (should use defaults)
	toolWithDefaults := NewSearchTool(nil)
	if toolWithDefaults == nil {
		t.Fatal("NewSearchTool(nil) returned nil")
	}

	if toolWithDefaults.GetName() != "search" {
		t.Errorf("GetName() = %v, want 'search'", toolWithDefaults.GetName())
	}
}

func TestSearchTool_WithConfig(t *testing.T) {
	// Test NewSearchToolWithConfig
	toolConfig := config.ToolConfig{
		DocumentStores:     []string{"config-store"},
		DefaultLimit:       2,
		MaxLimit:           4,
		EnabledSearchTypes: []string{"content"},
	}

	tool, err := NewSearchToolWithConfig("test-search", toolConfig)
	if err != nil {
		t.Fatalf("NewSearchToolWithConfig() error = %v", err)
	}

	if tool == nil {
		t.Fatal("NewSearchToolWithConfig() returned nil")
	}

	if tool.GetName() != "search" {
		t.Errorf("GetName() = %v, want 'search'", tool.GetName())
	}
}
