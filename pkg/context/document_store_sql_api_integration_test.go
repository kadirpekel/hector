package context

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/context/indexing"
	_ "github.com/mattn/go-sqlite3"
)

func setupSQLTestDB(t *testing.T) string {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			author TEXT,
			category TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		INSERT INTO articles (title, content, author, category, updated_at) VALUES
		('Getting Started with Go', 'Go is a programming language.', 'John Doe', 'Programming', '2024-01-01T10:00:00Z'),
		('Understanding Databases', 'Databases are essential.', 'Jane Smith', 'Database', '2024-01-02T10:00:00Z');
	`)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return dbPath
}

func setupMockAPIServer(t *testing.T) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/articles" {
			response := []map[string]interface{}{
				{
					"id":      "1",
					"title":   "API Article 1",
					"content": "This is content from API",
					"author":  "API Author",
				},
				{
					"id":      "2",
					"title":   "API Article 2",
					"content": "More API content",
					"author":  "Another Author",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server
}

func createMockSearchEngine() *SearchEngine {
	return &SearchEngine{
		// Mock implementation - in real tests, you'd use a test vector database
	}
}

func TestDocumentStore_SQLSource_Integration(t *testing.T) {
	dbPath := setupSQLTestDB(t)
	searchEngine := createMockSearchEngine()

	storeConfig := &config.DocumentStoreConfig{
		Name:   "sql-test-store",
		Source: "sql",
		SQL: &config.DocumentStoreSQLConfig{
			Driver:   "sqlite3",
			Database: dbPath,
		},
		SQLTables: []config.DocumentStoreSQLTableConfig{
			{
				Table:           "articles",
				Columns:         []string{"title", "content"},
				IDColumn:        "id",
				UpdatedColumn:   "updated_at",
				MetadataColumns: []string{"author", "category"},
			},
		},
		ChunkSize:     500,
		ChunkOverlap:  50,
		ChunkStrategy: "simple",
	}

	store, err := NewDocumentStore(storeConfig, searchEngine)
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}
	defer store.Close()

	// Test that data source is created correctly
	if store.dataSource == nil {
		t.Fatal("DocumentStore dataSource should not be nil")
	}

	if store.dataSource.Type() != "sql" {
		t.Errorf("DataSource type = %v, want 'sql'", store.dataSource.Type())
	}

	// Test document discovery
	ctx := context.Background()
	docChan, errChan := store.dataSource.DiscoverDocuments(ctx)

	documents := make([]indexing.Document, 0)
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

	// Verify document structure
	if len(documents) > 0 {
		doc := documents[0]
		if doc.ID == "" {
			t.Error("Document ID should not be empty")
		}
		if doc.Content == "" {
			t.Error("Document content should not be empty")
		}
		if doc.Metadata["table"] != "articles" {
			t.Errorf("Document metadata table = %v, want 'articles'", doc.Metadata["table"])
		}
		if !doc.ShouldIndex {
			t.Error("Document ShouldIndex should be true")
		}
	}
}

func TestDocumentStore_APISource_Integration(t *testing.T) {
	server := setupMockAPIServer(t)
	defer server.Close()

	searchEngine := createMockSearchEngine()

	storeConfig := &config.DocumentStoreConfig{
		Name:   "api-test-store",
		Source: "api",
		API: &config.DocumentStoreAPIConfig{
			BaseURL: server.URL,
			Endpoints: []config.DocumentStoreAPIEndpointConfig{
				{
					Path:           "/articles",
					Method:         "GET",
					IDField:        "id",
					ContentField:   "title,content",
					MetadataFields: []string{"author"},
				},
			},
		},
		ChunkSize:     500,
		ChunkOverlap:  50,
		ChunkStrategy: "simple",
	}

	store, err := NewDocumentStore(storeConfig, searchEngine)
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}
	defer store.Close()

	// Test that data source is created correctly
	if store.dataSource == nil {
		t.Fatal("DocumentStore dataSource should not be nil")
	}

	if store.dataSource.Type() != "api" {
		t.Errorf("DataSource type = %v, want 'api'", store.dataSource.Type())
	}

	// Test document discovery
	ctx := context.Background()
	docChan, errChan := store.dataSource.DiscoverDocuments(ctx)

	documents := make([]indexing.Document, 0)
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

	// Verify document structure
	if len(documents) > 0 {
		doc := documents[0]
		if doc.ID == "" {
			t.Error("Document ID should not be empty")
		}
		if doc.Content == "" {
			t.Error("Document content should not be empty")
		}
		if doc.Metadata["endpoint"] != "/articles" {
			t.Errorf("Document metadata endpoint = %v, want '/articles'", doc.Metadata["endpoint"])
		}
		if !doc.ShouldIndex {
			t.Error("Document ShouldIndex should be true")
		}
	}
}

func TestDocumentStore_SQLSource_ReadDocument(t *testing.T) {
	dbPath := setupSQLTestDB(t)
	searchEngine := createMockSearchEngine()

	storeConfig := &config.DocumentStoreConfig{
		Name:   "sql-test-store",
		Source: "sql",
		SQL: &config.DocumentStoreSQLConfig{
			Driver:   "sqlite3",
			Database: dbPath,
		},
		SQLTables: []config.DocumentStoreSQLTableConfig{
			{
				Table:           "articles",
				Columns:         []string{"title", "content"},
				IDColumn:        "id",
				UpdatedColumn:   "updated_at",
				MetadataColumns: []string{"author", "category"},
			},
		},
	}

	store, err := NewDocumentStore(storeConfig, searchEngine)
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Read a specific document
	doc, err := store.dataSource.ReadDocument(ctx, "sqlite3:articles:1")
	if err != nil {
		t.Fatalf("ReadDocument() error = %v", err)
	}

	if doc == nil {
		t.Fatal("ReadDocument() returned nil document")
	}

	if doc.ID != "sqlite3:articles:1" {
		t.Errorf("Document ID = %v, want 'sqlite3:articles:1'", doc.ID)
	}

	if doc.Content == "" {
		t.Error("Document content should not be empty")
	}

	if doc.Metadata["author"] == nil {
		t.Error("Document metadata should contain author")
	}

	if doc.Metadata["category"] == nil {
		t.Error("Document metadata should contain category")
	}
}

func TestDocumentStore_SQLSource_IncrementalIndexing(t *testing.T) {
	dbPath := setupSQLTestDB(t)
	searchEngine := createMockSearchEngine()

	storeConfig := &config.DocumentStoreConfig{
		Name:   "sql-test-store",
		Source: "sql",
		SQL: &config.DocumentStoreSQLConfig{
			Driver:   "sqlite3",
			Database: dbPath,
		},
		SQLTables: []config.DocumentStoreSQLTableConfig{
			{
				Table:         "articles",
				Columns:       []string{"title", "content"},
				IDColumn:      "id",
				UpdatedColumn: "updated_at",
			},
		},
		IncrementalIndexing: config.BoolPtr(true),
	}

	store, err := NewDocumentStore(storeConfig, searchEngine)
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}
	defer store.Close()

	// Test incremental indexing support
	if !store.dataSource.SupportsIncrementalIndexing() {
		t.Error("SQL source with UpdatedColumn should support incremental indexing")
	}
}

func TestDocumentStore_APISource_IncrementalIndexing(t *testing.T) {
	server := setupMockAPIServer(t)
	defer server.Close()

	searchEngine := createMockSearchEngine()

	storeConfig := &config.DocumentStoreConfig{
		Name:   "api-test-store",
		Source: "api",
		API: &config.DocumentStoreAPIConfig{
			BaseURL: server.URL,
			Endpoints: []config.DocumentStoreAPIEndpointConfig{
				{
					Path:         "/articles",
					Method:       "GET",
					IDField:      "id",
					ContentField: "title,content",
					UpdatedField: "updated_at",
				},
			},
		},
		IncrementalIndexing: config.BoolPtr(true),
	}

	store, err := NewDocumentStore(storeConfig, searchEngine)
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}
	defer store.Close()

	// Test incremental indexing support
	if !store.dataSource.SupportsIncrementalIndexing() {
		t.Error("API source with UpdatedField should support incremental indexing")
	}
}

func TestDocumentStore_SQLSource_GetLastModified(t *testing.T) {
	dbPath := setupSQLTestDB(t)
	searchEngine := createMockSearchEngine()

	storeConfig := &config.DocumentStoreConfig{
		Name:   "sql-test-store",
		Source: "sql",
		SQL: &config.DocumentStoreSQLConfig{
			Driver:   "sqlite3",
			Database: dbPath,
		},
		SQLTables: []config.DocumentStoreSQLTableConfig{
			{
				Table:         "articles",
				Columns:       []string{"title", "content"},
				IDColumn:      "id",
				UpdatedColumn: "updated_at",
			},
		},
	}

	store, err := NewDocumentStore(storeConfig, searchEngine)
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	lastModified, err := store.dataSource.GetLastModified(ctx, "sqlite3:articles:1")
	if err != nil {
		t.Fatalf("GetLastModified() error = %v", err)
	}

	if lastModified.IsZero() {
		t.Error("GetLastModified() should return non-zero time")
	}
}

func TestDocumentStore_DataSourceFactory_SQL(t *testing.T) {
	dbPath := setupSQLTestDB(t)

	factory := indexing.NewDataSourceFactory()

	storeConfig := &config.DocumentStoreConfig{
		Name:   "sql-test-store",
		Source: "sql",
		SQL: &config.DocumentStoreSQLConfig{
			Driver:   "sqlite3",
			Database: dbPath,
		},
		SQLTables: []config.DocumentStoreSQLTableConfig{
			{
				Table:    "articles",
				Columns:  []string{"title", "content"},
				IDColumn: "id",
			},
		},
	}

	dataSource, err := factory.CreateDataSource(storeConfig)
	if err != nil {
		t.Fatalf("CreateDataSource() error = %v", err)
	}
	defer dataSource.Close()

	if dataSource.Type() != "sql" {
		t.Errorf("DataSource type = %v, want 'sql'", dataSource.Type())
	}
}

func TestDocumentStore_DataSourceFactory_API(t *testing.T) {
	server := setupMockAPIServer(t)
	defer server.Close()

	factory := indexing.NewDataSourceFactory()

	storeConfig := &config.DocumentStoreConfig{
		Name:   "api-test-store",
		Source: "api",
		API: &config.DocumentStoreAPIConfig{
			BaseURL: server.URL,
			Endpoints: []config.DocumentStoreAPIEndpointConfig{
				{
					Path:         "/articles",
					Method:       "GET",
					IDField:      "id",
					ContentField: "title,content",
				},
			},
		},
	}

	dataSource, err := factory.CreateDataSource(storeConfig)
	if err != nil {
		t.Fatalf("CreateDataSource() error = %v", err)
	}
	defer dataSource.Close()

	if dataSource.Type() != "api" {
		t.Errorf("DataSource type = %v, want 'api'", dataSource.Type())
	}
}

func TestDocumentStore_DataSourceFactory_InvalidConfig(t *testing.T) {
	factory := indexing.NewDataSourceFactory()

	tests := []struct {
		name        string
		config      *config.DocumentStoreConfig
		wantError   bool
		errorSubstr string
	}{
		{
			name: "sql_without_config",
			config: &config.DocumentStoreConfig{
				Name:   "test",
				Source: "sql",
			},
			wantError:   true,
			errorSubstr: "SQL configuration is required",
		},
		{
			name: "api_without_config",
			config: &config.DocumentStoreConfig{
				Name:   "test",
				Source: "api",
			},
			wantError:   true,
			errorSubstr: "API configuration is required",
		},
		{
			name: "sql_without_driver",
			config: &config.DocumentStoreConfig{
				Name:   "test",
				Source: "sql",
				SQL: &config.DocumentStoreSQLConfig{
					Database: "test.db",
				},
			},
			wantError:   true,
			errorSubstr: "SQL driver is required",
		},
		{
			name: "api_without_baseurl",
			config: &config.DocumentStoreConfig{
				Name:   "test",
				Source: "api",
				API: &config.DocumentStoreAPIConfig{
					Endpoints: []config.DocumentStoreAPIEndpointConfig{
						{Path: "/test"},
					},
				},
			},
			wantError:   true,
			errorSubstr: "API base URL is required",
		},
		{
			name: "api_without_endpoints",
			config: &config.DocumentStoreConfig{
				Name:   "test",
				Source: "api",
				API: &config.DocumentStoreAPIConfig{
					BaseURL: "http://example.com",
				},
			},
			wantError:   true,
			errorSubstr: "at least one API endpoint is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := factory.CreateDataSource(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("CreateDataSource() expected error, got nil")
				} else if tt.errorSubstr != "" && !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("CreateDataSource() error = %v, want error containing '%s'", err, tt.errorSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("CreateDataSource() error = %v, want nil", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
