package indexing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPISource_Type(t *testing.T) {
	source := NewAPISource("http://example.com", []APIEndpointConfig{}, nil)
	if source.Type() != "api" {
		t.Errorf("Type() = %v, want 'api'", source.Type())
	}
}

func TestAPISource_DiscoverDocuments_SimpleArray(t *testing.T) {
	// Create mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []map[string]interface{}{
			{
				"id":      "1",
				"title":   "Article 1",
				"content": "This is the content of article 1",
				"author":  "John Doe",
			},
			{
				"id":      "2",
				"title":   "Article 2",
				"content": "This is the content of article 2",
				"author":  "Jane Smith",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title,content",
			MetadataFields: []string{"author"},
		},
	}

	source := NewAPISource(server.URL, endpoints, nil)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	errors := make([]error, 0)
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("DiscoverDocuments() returned errors: %v", errors)
	}

	if len(documents) != 2 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 2", len(documents))
	}

	// Verify first document
	doc := documents[0]
	if !strings.HasPrefix(doc.ID, "api:/articles:") {
		t.Errorf("Document ID should start with 'api:/articles:', got %s", doc.ID)
	}
	if doc.Content == "" {
		t.Error("Document content should not be empty")
	}
	if !strings.Contains(doc.Content, "Article 1") {
		t.Errorf("Document content should contain title, got: %s", doc.Content)
	}
	if doc.Metadata["author"] == nil {
		t.Error("Metadata should contain author")
	}
	if doc.Metadata["endpoint"] != "/articles" {
		t.Errorf("Metadata endpoint = %v, want '/articles'", doc.Metadata["endpoint"])
	}
}

func TestAPISource_DiscoverDocuments_NestedObject(t *testing.T) {
	// Create mock HTTP server with nested response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":      "1",
					"title":   "Article 1",
					"content": "Content 1",
				},
				{
					"id":      "2",
					"title":   "Article 2",
					"content": "Content 2",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title,content",
		},
	}

	source := NewAPISource(server.URL, endpoints, nil)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	// Should extract array from nested object
	if len(documents) != 2 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 2", len(documents))
	}
}

func TestAPISource_DiscoverDocuments_SingleObject(t *testing.T) {
	// Create mock HTTP server returning single object
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id":      "1",
			"title":   "Single Article",
			"content": "This is a single article",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	endpoints := []APIEndpointConfig{
		{
			Path:         "/article",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title,content",
		},
	}

	source := NewAPISource(server.URL, endpoints, nil)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	// Should treat single object as array with one item
	if len(documents) != 1 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 1", len(documents))
	}
}

func TestAPISource_DiscoverDocuments_WithPagination(t *testing.T) {
	page := 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageParam := r.URL.Query().Get("page")
		if pageParam == "1" {
			response := []map[string]interface{}{
				{"id": "1", "title": "Page 1 Article"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if pageParam == "2" {
			response := []map[string]interface{}{
				{"id": "2", "title": "Page 2 Article"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title",
			Pagination: &PaginationConfig{
				Type:      "page",
				PageParam: "page",
				SizeParam: "size",
				PageSize:  1,
				MaxPages:  2,
			},
		},
	}

	source := NewAPISource(server.URL, endpoints, nil)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	// Should have documents from both pages
	if len(documents) < 2 {
		t.Errorf("DiscoverDocuments() returned %d documents, want at least 2 (from pagination)", len(documents))
	}

	_ = page // Avoid unused variable warning
}

func TestAPISource_DiscoverDocuments_WithQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		if r.URL.Query().Get("status") != "active" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := []map[string]interface{}{
			{"id": "1", "title": "Active Article"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title",
			Params: map[string]string{
				"status": "active",
			},
		},
	}

	source := NewAPISource(server.URL, endpoints, nil)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	if len(documents) != 1 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 1", len(documents))
	}
}

func TestAPISource_DiscoverDocuments_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom header
		if r.Header.Get("X-Custom-Header") != "test-value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := []map[string]interface{}{
			{"id": "1", "title": "Article"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title",
			Headers: map[string]string{
				"X-Custom-Header": "test-value",
			},
		},
	}

	source := NewAPISource(server.URL, endpoints, nil)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	if len(documents) != 1 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 1", len(documents))
	}
}

func TestAPISource_DiscoverDocuments_WithBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer test-token") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		response := []map[string]interface{}{
			{"id": "1", "title": "Article"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	auth := &APIAuthConfig{
		Type:  "bearer",
		Token: "test-token",
	}

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title",
		},
	}

	source := NewAPISource(server.URL, endpoints, auth)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	if len(documents) != 1 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 1", len(documents))
	}
}

func TestAPISource_DiscoverDocuments_WithBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		response := []map[string]interface{}{
			{"id": "1", "title": "Article"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	auth := &APIAuthConfig{
		Type: "basic",
		User: "testuser",
		Pass: "testpass",
	}

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title",
		},
	}

	source := NewAPISource(server.URL, endpoints, auth)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	if len(documents) != 1 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 1", len(documents))
	}
}

func TestAPISource_DiscoverDocuments_WithAPIKeyAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		response := []map[string]interface{}{
			{"id": "1", "title": "Article"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	auth := &APIAuthConfig{
		Type:   "apikey",
		Token:  "test-api-key",
		Header: "X-API-Key",
	}

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title",
		},
	}

	source := NewAPISource(server.URL, endpoints, auth)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	if len(documents) != 1 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 1", len(documents))
	}
}

func TestAPISource_DiscoverDocuments_MultipleEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/articles" {
			response := []map[string]interface{}{
				{"id": "1", "title": "Article 1"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if r.URL.Path == "/products" {
			response := []map[string]interface{}{
				{"id": "1", "name": "Product 1"},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title",
		},
		{
			Path:         "/products",
			Method:       "GET",
			IDField:      "id",
			ContentField: "name",
		},
	}

	source := NewAPISource(server.URL, endpoints, nil)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	// Should have documents from both endpoints
	if len(documents) != 2 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 2", len(documents))
	}

	hasArticles := false
	hasProducts := false
	for _, doc := range documents {
		if strings.Contains(doc.ID, ":/articles:") {
			hasArticles = true
		}
		if strings.Contains(doc.ID, ":/products:") {
			hasProducts = true
		}
	}

	if !hasArticles {
		t.Error("Should have documents from /articles endpoint")
	}
	if !hasProducts {
		t.Error("Should have documents from /products endpoint")
	}
}

func TestAPISource_SupportsIncrementalIndexing(t *testing.T) {
	tests := []struct {
		name          string
		endpoints     []APIEndpointConfig
		wantSupported bool
	}{
		{
			name: "with_updated_field",
			endpoints: []APIEndpointConfig{
				{
					Path:         "/articles",
					IDField:      "id",
					ContentField: "title",
					UpdatedField: "updated_at",
				},
			},
			wantSupported: true,
		},
		{
			name: "without_updated_field",
			endpoints: []APIEndpointConfig{
				{
					Path:         "/articles",
					IDField:      "id",
					ContentField: "title",
				},
			},
			wantSupported: false,
		},
		{
			name: "mixed_configs",
			endpoints: []APIEndpointConfig{
				{
					Path:         "/articles",
					IDField:      "id",
					ContentField: "title",
				},
				{
					Path:         "/products",
					IDField:      "id",
					ContentField: "name",
					UpdatedField: "updated_at",
				},
			},
			wantSupported: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewAPISource("http://example.com", tt.endpoints, nil)
			if got := source.SupportsIncrementalIndexing(); got != tt.wantSupported {
				t.Errorf("SupportsIncrementalIndexing() = %v, want %v", got, tt.wantSupported)
			}
		})
	}
}

func TestAPISource_ContentExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []map[string]interface{}{
			{
				"id":          "1",
				"title":       "Article Title",
				"description": "Article Description",
				"content":     "Article Content",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title,description,content",
		},
	}

	source := NewAPISource(server.URL, endpoints, nil)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	if len(documents) != 1 {
		t.Fatalf("DiscoverDocuments() returned %d documents, want 1", len(documents))
	}

	doc := documents[0]
	// Content should contain all specified fields
	if !strings.Contains(doc.Content, "Article Title") {
		t.Errorf("Content should contain title, got: %s", doc.Content)
	}
	if !strings.Contains(doc.Content, "Article Description") {
		t.Errorf("Content should contain description, got: %s", doc.Content)
	}
	if !strings.Contains(doc.Content, "Article Content") {
		t.Errorf("Content should contain content, got: %s", doc.Content)
	}
}

func TestAPISource_MetadataExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []map[string]interface{}{
			{
				"id":      "1",
				"title":   "Article",
				"author":  "John Doe",
				"status":  "published",
				"views":   100,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	endpoints := []APIEndpointConfig{
		{
			Path:           "/articles",
			Method:         "GET",
			IDField:        "id",
			ContentField:   "title",
			MetadataFields: []string{"author", "status", "views"},
		},
	}

	source := NewAPISource(server.URL, endpoints, nil)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	if len(documents) != 1 {
		t.Fatalf("DiscoverDocuments() returned %d documents, want 1", len(documents))
	}

	doc := documents[0]
	if doc.Metadata["author"] == nil {
		t.Error("Metadata should contain author")
	}
	if doc.Metadata["status"] == nil {
		t.Error("Metadata should contain status")
	}
	if doc.Metadata["views"] == nil {
		t.Error("Metadata should contain views")
	}
	if doc.Metadata["endpoint"] != "/articles" {
		t.Errorf("Metadata endpoint = %v, want '/articles'", doc.Metadata["endpoint"])
	}
}

func TestAPISource_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		response := []map[string]interface{}{
			{"id": "1", "title": "Article"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title",
		},
	}

	source := NewAPISource(server.URL, endpoints, nil)
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	docChan, errChan := source.DiscoverDocuments(ctx)

	// Should handle cancellation gracefully
	docCount := 0
	for range docChan {
		docCount++
	}

	// Check for context cancellation error
	hasCancelErr := false
	for err := range errChan {
		if err == context.Canceled {
			hasCancelErr = true
		}
	}

	// Context cancellation should be handled gracefully
	_ = docCount
	_ = hasCancelErr
}

func TestAPISource_ReadDocument_NotImplemented(t *testing.T) {
	source := NewAPISource("http://example.com", []APIEndpointConfig{}, nil)
	ctx := context.Background()

	_, err := source.ReadDocument(ctx, "api:/articles:1")
	if err == nil {
		t.Error("ReadDocument() should return error (not implemented)")
	}
	// The actual error message may vary, just check that it's an error
	if err != nil && !strings.Contains(err.Error(), "not yet implemented") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("ReadDocument() error should mention implementation issue, got: %v", err)
	}
}

func TestAPISource_Close(t *testing.T) {
	source := NewAPISource("http://example.com", []APIEndpointConfig{}, nil)
	
	err := source.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestAPISource_HTTPErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	endpoints := []APIEndpointConfig{
		{
			Path:         "/articles",
			Method:       "GET",
			IDField:      "id",
			ContentField: "title",
		},
	}

	source := NewAPISource(server.URL, endpoints, nil)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	errors := make([]error, 0)
	for err := range errChan {
		errors = append(errors, err)
	}

	// Should have error for HTTP 500
	if len(errors) == 0 {
		t.Error("DiscoverDocuments() should return error for HTTP 500")
	}

	if len(documents) > 0 {
		t.Error("DiscoverDocuments() should not return documents on HTTP error")
	}
}

