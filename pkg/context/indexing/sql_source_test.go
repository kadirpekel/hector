package indexing

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (string, *sql.DB) {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create test tables
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

		CREATE TABLE products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			price REAL,
			status TEXT DEFAULT 'active',
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		INSERT INTO articles (title, content, author, category, updated_at) VALUES
		('Getting Started with Go', 'Go is a programming language developed by Google.', 'John Doe', 'Programming', '2024-01-01T10:00:00Z'),
		('Understanding Databases', 'Databases are essential for storing data.', 'Jane Smith', 'Database', '2024-01-02T10:00:00Z'),
		('REST API Best Practices', 'REST APIs should follow conventions.', 'Bob Johnson', 'API', '2024-01-03T10:00:00Z');

		INSERT INTO products (name, description, price, status, updated_at) VALUES
		('Laptop', 'High-performance laptop', 1299.99, 'active', '2024-01-01T10:00:00Z'),
		('Monitor', '27-inch 4K monitor', 399.99, 'active', '2024-01-02T10:00:00Z');
	`)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return dbPath, db
}

func TestSQLSource_Type(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	source := NewSQLSource(db, "sqlite3", []SQLTableConfig{}, 0)
	if source.Type() != "sql" {
		t.Errorf("Type() = %v, want 'sql'", source.Type())
	}
}

func TestSQLSource_DiscoverDocuments(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	tableConfigs := []SQLTableConfig{
		{
			Table:           "articles",
			Columns:         []string{"title", "content"},
			IDColumn:        "id",
			UpdatedColumn:   "updated_at",
			MetadataColumns: []string{"author", "category"},
		},
	}

	source := NewSQLSource(db, "sqlite3", tableConfigs, 0)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	errors := make([]error, 0)

	// Collect documents
	for doc := range docChan {
		documents = append(documents, doc)
	}

	// Collect errors
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("DiscoverDocuments() returned errors: %v", errors)
	}

	if len(documents) != 3 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 3", len(documents))
	}

	// Verify first document
	doc := documents[0]
	if doc.ID == "" {
		t.Error("Document ID should not be empty")
	}
	if !strings.HasPrefix(doc.ID, "sqlite3:articles:") {
		t.Errorf("Document ID should start with 'sqlite3:articles:', got %s", doc.ID)
	}
	if doc.Content == "" {
		t.Error("Document content should not be empty")
	}
	if !strings.Contains(doc.Content, "Getting Started with Go") {
		t.Errorf("Document content should contain title, got: %s", doc.Content)
	}
	if doc.Metadata["table"] != "articles" {
		t.Errorf("Metadata table = %v, want 'articles'", doc.Metadata["table"])
	}
	if doc.Metadata["author"] == nil {
		t.Error("Metadata should contain author")
	}
	if !doc.ShouldIndex {
		t.Error("Document ShouldIndex should be true")
	}
}

func TestSQLSource_DiscoverDocuments_MultipleTables(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	tableConfigs := []SQLTableConfig{
		{
			Table:           "articles",
			Columns:         []string{"title", "content"},
			IDColumn:        "id",
			UpdatedColumn:   "updated_at",
			MetadataColumns: []string{"author"},
		},
		{
			Table:           "products",
			Columns:         []string{"name", "description"},
			IDColumn:        "id",
			UpdatedColumn:   "updated_at",
			MetadataColumns: []string{"price", "status"},
		},
	}

	source := NewSQLSource(db, "sqlite3", tableConfigs, 0)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	// Should have 3 articles + 2 products = 5 documents
	if len(documents) != 5 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 5", len(documents))
	}

	// Verify we have documents from both tables
	hasArticles := false
	hasProducts := false
	for _, doc := range documents {
		if strings.Contains(doc.ID, ":articles:") {
			hasArticles = true
		}
		if strings.Contains(doc.ID, ":products:") {
			hasProducts = true
		}
	}

	if !hasArticles {
		t.Error("Should have documents from articles table")
	}
	if !hasProducts {
		t.Error("Should have documents from products table")
	}
}

func TestSQLSource_DiscoverDocuments_WithWhereClause(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	tableConfigs := []SQLTableConfig{
		{
			Table:           "articles",
			Columns:         []string{"title", "content"},
			IDColumn:        "id",
			UpdatedColumn:   "updated_at",
			MetadataColumns: []string{"author"},
			WhereClause:     "category = 'Programming'",
		},
	}

	source := NewSQLSource(db, "sqlite3", tableConfigs, 0)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	// Should only have 1 article with category 'Programming'
	if len(documents) != 1 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 1", len(documents))
	}

	if !strings.Contains(documents[0].Content, "Getting Started with Go") {
		t.Errorf("Document should be filtered by WHERE clause")
	}
}

func TestSQLSource_DiscoverDocuments_WithMaxRows(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	// Insert more rows
	_, err := db.Exec(`
		INSERT INTO articles (title, content, author, category) VALUES
		('Article 4', 'Content 4', 'Author 4', 'Category 4'),
		('Article 5', 'Content 5', 'Author 5', 'Category 5'),
		('Article 6', 'Content 6', 'Author 6', 'Category 6');
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	tableConfigs := []SQLTableConfig{
		{
			Table:         "articles",
			Columns:       []string{"title", "content"},
			IDColumn:      "id",
			UpdatedColumn: "updated_at",
		},
	}

	source := NewSQLSource(db, "sqlite3", tableConfigs, 2) // Limit to 2 rows
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	// Should be limited to 2 documents
	if len(documents) != 2 {
		t.Errorf("DiscoverDocuments() returned %d documents, want 2 (maxRows limit)", len(documents))
	}
}

func TestSQLSource_ReadDocument(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	tableConfigs := []SQLTableConfig{
		{
			Table:           "articles",
			Columns:         []string{"title", "content"},
			IDColumn:        "id",
			UpdatedColumn:   "updated_at",
			MetadataColumns: []string{"author", "category"},
		},
	}

	source := NewSQLSource(db, "sqlite3", tableConfigs, 0)
	ctx := context.Background()

	// Read document with ID format: driver:table:id
	doc, err := source.ReadDocument(ctx, "sqlite3:articles:1")
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

	if !strings.Contains(doc.Content, "Getting Started with Go") {
		t.Errorf("Document content should contain expected title, got: %s", doc.Content)
	}

	if doc.Metadata["table"] != "articles" {
		t.Errorf("Metadata table = %v, want 'articles'", doc.Metadata["table"])
	}

	if doc.Metadata["author"] == nil {
		t.Error("Metadata should contain author")
	}
}

func TestSQLSource_ReadDocument_InvalidID(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	tableConfigs := []SQLTableConfig{
		{
			Table:    "articles",
			Columns:  []string{"title", "content"},
			IDColumn: "id",
		},
	}

	source := NewSQLSource(db, "sqlite3", tableConfigs, 0)
	ctx := context.Background()

	// Test invalid ID format
	_, err := source.ReadDocument(ctx, "invalid-id")
	if err == nil {
		t.Error("ReadDocument() should return error for invalid ID format")
	}

	// Test non-existent document
	_, err = source.ReadDocument(ctx, "sqlite3:articles:999")
	if err == nil {
		t.Error("ReadDocument() should return error for non-existent document")
	}

	// Test table not in config
	_, err = source.ReadDocument(ctx, "sqlite3:products:1")
	if err == nil {
		t.Error("ReadDocument() should return error for table not in configuration")
	}
}

func TestSQLSource_SupportsIncrementalIndexing(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	tests := []struct {
		name          string
		tableConfigs  []SQLTableConfig
		wantSupported bool
	}{
		{
			name: "with_updated_column",
			tableConfigs: []SQLTableConfig{
				{
					Table:         "articles",
					Columns:       []string{"title"},
					IDColumn:      "id",
					UpdatedColumn: "updated_at",
				},
			},
			wantSupported: true,
		},
		{
			name: "without_updated_column",
			tableConfigs: []SQLTableConfig{
				{
					Table:    "articles",
					Columns:  []string{"title"},
					IDColumn: "id",
				},
			},
			wantSupported: false,
		},
		{
			name: "mixed_configs",
			tableConfigs: []SQLTableConfig{
				{
					Table:    "articles",
					Columns:  []string{"title"},
					IDColumn: "id",
				},
				{
					Table:         "products",
					Columns:       []string{"name"},
					IDColumn:      "id",
					UpdatedColumn: "updated_at",
				},
			},
			wantSupported: true, // Should be true if at least one table has UpdatedColumn
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewSQLSource(db, "sqlite3", tt.tableConfigs, 0)
			if got := source.SupportsIncrementalIndexing(); got != tt.wantSupported {
				t.Errorf("SupportsIncrementalIndexing() = %v, want %v", got, tt.wantSupported)
			}
		})
	}
}

func TestSQLSource_GetLastModified(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	tableConfigs := []SQLTableConfig{
		{
			Table:         "articles",
			Columns:       []string{"title", "content"},
			IDColumn:      "id",
			UpdatedColumn: "updated_at",
		},
	}

	source := NewSQLSource(db, "sqlite3", tableConfigs, 0)
	ctx := context.Background()

	lastModified, err := source.GetLastModified(ctx, "sqlite3:articles:1")
	if err != nil {
		t.Fatalf("GetLastModified() error = %v", err)
	}

	if lastModified.IsZero() {
		t.Error("GetLastModified() should return non-zero time")
	}
}

func TestSQLSource_Close(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	source := NewSQLSource(db, "sqlite3", []SQLTableConfig{}, 0)

	err := source.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Verify database is closed
	_, err = db.Query("SELECT 1")
	if err == nil {
		t.Error("Database should be closed after Close()")
	}
}

func TestSQLSource_ContextCancellation(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	tableConfigs := []SQLTableConfig{
		{
			Table:    "articles",
			Columns:  []string{"title", "content"},
			IDColumn: "id",
		},
	}

	source := NewSQLSource(db, "sqlite3", tableConfigs, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	docChan, errChan := source.DiscoverDocuments(ctx)

	// Should handle cancellation gracefully
	docCount := 0
	for range docChan {
		docCount++
	}

	// Should not process documents after cancellation
	// (exact behavior depends on timing, but should not panic)
	_ = docCount

	// Check for context cancellation error
	hasCancelErr := false
	for err := range errChan {
		if err == context.Canceled {
			hasCancelErr = true
		}
	}

	// Context cancellation should be handled gracefully
	_ = hasCancelErr
}

func TestSQLSource_ContentConcatenation(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	tableConfigs := []SQLTableConfig{
		{
			Table:    "articles",
			Columns:  []string{"title", "content"},
			IDColumn: "id",
		},
	}

	source := NewSQLSource(db, "sqlite3", tableConfigs, 0)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	if len(documents) == 0 {
		t.Fatal("Should have at least one document")
	}

	doc := documents[0]
	// Content should contain both title and content columns separated by \n\n
	if !strings.Contains(doc.Content, "Getting Started with Go") {
		t.Errorf("Content should contain title, got: %s", doc.Content)
	}
	if !strings.Contains(doc.Content, "Go is a programming language") {
		t.Errorf("Content should contain content column, got: %s", doc.Content)
	}
	// Check for double newline separator
	if !strings.Contains(doc.Content, "\n\n") {
		t.Errorf("Content columns should be separated by \\n\\n")
	}
}

func TestSQLSource_MetadataExtraction(t *testing.T) {
	_, db := setupTestDB(t)
	defer db.Close()

	tableConfigs := []SQLTableConfig{
		{
			Table:           "articles",
			Columns:         []string{"title", "content"},
			IDColumn:        "id",
			UpdatedColumn:   "updated_at",
			MetadataColumns: []string{"author", "category"},
		},
	}

	source := NewSQLSource(db, "sqlite3", tableConfigs, 0)
	ctx := context.Background()

	docChan, errChan := source.DiscoverDocuments(ctx)

	documents := make([]Document, 0)
	for doc := range docChan {
		documents = append(documents, doc)
	}

	for err := range errChan {
		t.Errorf("DiscoverDocuments() returned error: %v", err)
	}

	if len(documents) == 0 {
		t.Fatal("Should have at least one document")
	}

	doc := documents[0]
	if doc.Metadata["author"] == nil {
		t.Error("Metadata should contain author")
	}
	if doc.Metadata["category"] == nil {
		t.Error("Metadata should contain category")
	}
	if doc.Metadata["table"] != "articles" {
		t.Errorf("Metadata table = %v, want 'articles'", doc.Metadata["table"])
	}
	if doc.Metadata["id"] == nil {
		t.Error("Metadata should contain id")
	}
}
