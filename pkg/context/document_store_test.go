package context

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
)

type mockSearchEngine struct {
	ingestFunc func(ctx context.Context, docID, content string, metadata map[string]interface{}) error
	searchFunc func(ctx context.Context, query string, limit int) ([]databases.SearchResult, error)
}

func (m *mockSearchEngine) IngestDocument(ctx context.Context, docID, content string, metadata map[string]interface{}) error {
	if m.ingestFunc != nil {
		return m.ingestFunc(ctx, docID, content, metadata)
	}
	return nil
}

func (m *mockSearchEngine) Search(ctx context.Context, query string, limit int) ([]databases.SearchResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, query, limit)
	}
	return []databases.SearchResult{}, nil
}

func TestNewDocumentStore(t *testing.T) {

	tempDir, err := os.MkdirTemp("", "test-doc-store")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	_ = &mockSearchEngine{}

	tests := []struct {
		name         string
		storeConfig  *config.DocumentStoreConfig
		searchEngine *SearchEngine
		wantError    bool
	}{
		{
			name: "valid_configuration",
			storeConfig: &config.DocumentStoreConfig{
				Name:            "test-store",
				Path:            tempDir,
				Source:          "directory",
				MaxFileSize:     1024 * 1024,
				IncludePatterns: []string{"*.txt", "*.md"},
				ExcludePatterns: []string{"*.tmp"},
				WatchChanges:    false,
			},
			searchEngine: &SearchEngine{},
			wantError:    false,
		},
		{
			name:         "nil_store_config",
			storeConfig:  nil,
			searchEngine: &SearchEngine{},
			wantError:    true,
		},
		{
			name: "nil_search_engine",
			storeConfig: &config.DocumentStoreConfig{
				Name: "test-store",
				Path: tempDir,
			},
			searchEngine: nil,
			wantError:    true,
		},
		{
			name: "empty_store_name",
			storeConfig: &config.DocumentStoreConfig{
				Name: "",
				Path: tempDir,
			},
			searchEngine: &SearchEngine{},
			wantError:    false,
		},
		{
			name: "empty_source_path",
			storeConfig: &config.DocumentStoreConfig{
				Name: "test-store",
				Path: "",
			},
			searchEngine: &SearchEngine{},
			wantError:    false, // SetDefaults will set it to "./" which is valid
		},
		{
			name: "nonexistent_source_path",
			storeConfig: &config.DocumentStoreConfig{
				Name: "test-store",
				Path: "/nonexistent/path",
			},
			searchEngine: &SearchEngine{},
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewDocumentStore(tt.storeConfig, tt.searchEngine)

			if tt.wantError {
				if err == nil {
					t.Error("NewDocumentStore() expected error, got nil")
				}
				if store != nil {
					t.Error("NewDocumentStore() expected nil store on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewDocumentStore() error = %v, want nil", err)
				}
				if store == nil {
					t.Error("NewDocumentStore() returned nil store")
				}
				if store != nil {
					if store.name != tt.storeConfig.Name {
						t.Errorf("NewDocumentStore() name = %v, want %v", store.name, tt.storeConfig.Name)
					}
					if store.sourcePath != tt.storeConfig.Path {
						t.Errorf("NewDocumentStore() sourcePath = %v, want %v", store.sourcePath, tt.storeConfig.Path)
					}
				}
			}
		})
	}
}

func TestDocumentStore_StartIndexing(t *testing.T) {

	tempDir, err := os.MkdirTemp("", "test-indexing")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFiles := []struct {
		name    string
		content string
	}{
		{"test1.txt", "This is test content 1"},
		{"test2.md", "# Test Document\n\nThis is a test markdown file."},
		{"test3.go", "package main\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}"},
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file.name)
		err := os.WriteFile(filePath, []byte(file.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file.name, err)
		}
	}

	storeConfig := &config.DocumentStoreConfig{
		Name:            "test-store",
		Path:            tempDir,
		Source:          "directory",
		MaxFileSize:     1024 * 1024,
		IncludePatterns: []string{"*"},
		ExcludePatterns: []string{},
		WatchChanges:    false,
	}

	store, err := NewDocumentStore(storeConfig, &SearchEngine{})
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}

	tests := []struct {
		name      string
		source    string
		wantError bool
	}{
		{
			name:      "index_directory",
			source:    "directory",
			wantError: false,
		},
		{
			name:      "index_git_repository",
			source:    "git",
			wantError: true,
		},
		{
			name:      "unsupported_source",
			source:    "unsupported",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			store.config.Source = tt.source

			err := store.StartIndexing()

			if tt.wantError {
				if err == nil {
					t.Error("StartIndexing() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("StartIndexing() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestDocumentStore_Search(t *testing.T) {

	t.Skip("Skipping Search test - requires properly initialized SearchEngine")
}

func TestDocumentStore_GetDocument(t *testing.T) {

	t.Skip("Skipping GetDocument test - requires properly initialized SearchEngine")
}

func TestDocumentStore_GetStatus(t *testing.T) {
	storeConfig := &config.DocumentStoreConfig{
		Name: "test-store",
		Path: "/tmp",
	}

	store, err := NewDocumentStore(storeConfig, &SearchEngine{})
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}

	status := store.GetStatus()

	if status == nil {
		t.Fatal("GetStatus() returned nil status")
	}

	if status.Name != "test-store" {
		t.Errorf("GetStatus() Name = %v, want test-store", status.Name)
	}
	if status.SourcePath != "/tmp" {
		t.Errorf("GetStatus() SourcePath = %v, want /tmp", status.SourcePath)
	}
	if status.Storage != "vector_database" {
		t.Errorf("GetStatus() Storage = %v, want vector_database", status.Storage)
	}
	if status.IsIndexing {
		t.Error("GetStatus() IsIndexing should be false initially")
	}
	if status.IsWatching {
		t.Error("GetStatus() IsWatching should be false initially")
	}
}

func TestDocumentStore_FileFiltering(t *testing.T) {
	storeConfig := &config.DocumentStoreConfig{
		Name:            "test-store",
		Path:            "/tmp",
		IncludePatterns: []string{"*.txt", "*.md"},
		ExcludePatterns: []string{".tmp", "temp"},
	}

	store, err := NewDocumentStore(storeConfig, &SearchEngine{})
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}

	tests := []struct {
		name          string
		path          string
		shouldExclude bool
		shouldInclude bool
	}{
		{
			name:          "included_txt_file",
			path:          "test.txt",
			shouldExclude: false,
			shouldInclude: true,
		},
		{
			name:          "included_md_file",
			path:          "test.md",
			shouldExclude: false,
			shouldInclude: true,
		},
		{
			name:          "excluded_tmp_file",
			path:          "test.tmp",
			shouldExclude: true,
			shouldInclude: false,
		},
		{
			name:          "excluded_temp_directory",
			path:          "temp/file.txt",
			shouldExclude: true,
			shouldInclude: true,
		},
		{
			name:          "not_included_go_file",
			path:          "test.go",
			shouldExclude: false,
			shouldInclude: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluded := store.shouldExclude(tt.path)
			if excluded != tt.shouldExclude {
				t.Errorf("shouldExclude() = %v, want %v", excluded, tt.shouldExclude)
			}

			included := store.shouldInclude(tt.path)
			if included != tt.shouldInclude {
				t.Errorf("shouldInclude() = %v, want %v", included, tt.shouldInclude)
			}
		})
	}
}

func TestDocumentStore_TypeDetection(t *testing.T) {
	storeConfig := &config.DocumentStoreConfig{
		Name: "test-store",
		Path: "/tmp",
	}

	store, err := NewDocumentStore(storeConfig, &SearchEngine{})
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}

	tests := []struct {
		name         string
		path         string
		expectedType string
		expectedLang string
	}{
		{
			name:         "go_file",
			path:         "test.go",
			expectedType: DocumentTypeCode,
			expectedLang: "go",
		},
		{
			name:         "python_file",
			path:         "test.py",
			expectedType: DocumentTypeCode,
			expectedLang: "python",
		},
		{
			name:         "javascript_file",
			path:         "test.js",
			expectedType: DocumentTypeCode,
			expectedLang: "javascript",
		},
		{
			name:         "yaml_file",
			path:         "test.yaml",
			expectedType: DocumentTypeConfig,
			expectedLang: "yaml",
		},
		{
			name:         "json_file",
			path:         "test.json",
			expectedType: DocumentTypeConfig,
			expectedLang: "json",
		},
		{
			name:         "markdown_file",
			path:         "test.md",
			expectedType: DocumentTypeMarkdown,
			expectedLang: "markdown",
		},
		{
			name:         "text_file",
			path:         "test.txt",
			expectedType: DocumentTypeText,
			expectedLang: "text",
		},
		{
			name:         "shell_script",
			path:         "test.sh",
			expectedType: DocumentTypeScript,
			expectedLang: "shell",
		},
		{
			name:         "unknown_file",
			path:         "test.unknown",
			expectedType: DocumentTypeUnknown,
			expectedLang: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docType, language := store.detectTypeAndLanguage(tt.path)

			if docType != tt.expectedType {
				t.Errorf("detectTypeAndLanguage() type = %v, want %v", docType, tt.expectedType)
			}
			if language != tt.expectedLang {
				t.Errorf("detectTypeAndLanguage() language = %v, want %v", language, tt.expectedLang)
			}
		})
	}
}

func TestDocumentStore_ContentChunking(t *testing.T) {
	storeConfig := &config.DocumentStoreConfig{
		Name:          "test-store",
		Path:          "/tmp",
		ChunkSize:     100,
		ChunkStrategy: "simple",
	}

	store, err := NewDocumentStore(storeConfig, &SearchEngine{})
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}

	tests := []struct {
		name           string
		content        string
		targetSize     int
		expectedChunks int
	}{
		{
			name:           "small_content",
			content:        "This is a short text.",
			targetSize:     100,
			expectedChunks: 1,
		},
		{
			name:           "large_content",
			content:        strings.Repeat("This is a line of text.\n", 50),
			targetSize:     100,
			expectedChunks: 13,
		},
		{
			name:           "empty_content",
			content:        "",
			targetSize:     100,
			expectedChunks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the store's chunker directly
			chunks, err := store.chunker.Chunk(tt.content, nil)
			if err != nil {
				t.Fatalf("chunker.Chunk() error = %v", err)
			}

			if len(chunks) != tt.expectedChunks {
				t.Errorf("chunker.Chunk() chunks length = %v, want %v", len(chunks), tt.expectedChunks)
			}

			for i, chunk := range chunks {
				if chunk.StartLine <= 0 {
					t.Errorf("chunker.Chunk() chunk %d StartLine = %v, want > 0", i, chunk.StartLine)
				}
				if chunk.EndLine < chunk.StartLine {
					t.Errorf("chunker.Chunk() chunk %d EndLine = %v, want >= StartLine", i, chunk.EndLine)
				}
				if chunk.Content == "" && tt.content != "" {
					t.Errorf("chunker.Chunk() chunk %d Content should not be empty", i)
				}
			}
		})
	}
}

func TestDocumentStore_Close(t *testing.T) {
	storeConfig := &config.DocumentStoreConfig{
		Name: "test-store",
		Path: "/tmp",
	}

	store, err := NewDocumentStore(storeConfig, &SearchEngine{})
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestDocumentStoreRegistry(t *testing.T) {

	storeConfig := &config.DocumentStoreConfig{
		Name: "test-store",
		Path: "/tmp",
	}

	store, err := NewDocumentStore(storeConfig, &SearchEngine{})
	if err != nil {
		t.Fatalf("NewDocumentStore() error = %v", err)
	}

	RegisterDocumentStore(store)

	retrievedStore, exists := GetDocumentStoreFromRegistry("test-store")
	if !exists {
		t.Error("GetDocumentStoreFromRegistry() store should exist after registration")
	}
	if retrievedStore != store {
		t.Error("GetDocumentStoreFromRegistry() should return the same store instance")
	}

	storeNames := ListDocumentStoresFromRegistry()
	if len(storeNames) == 0 {
		t.Error("ListDocumentStoresFromRegistry() should return at least one store")
	}
	found := false
	for _, name := range storeNames {
		if name == "test-store" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ListDocumentStoresFromRegistry() should include test-store")
	}

	stats := GetDocumentStoreStats()
	if len(stats) == 0 {
		t.Error("GetDocumentStoreStats() should return stats for registered stores")
	}

	UnregisterDocumentStore("test-store")
	_, exists = GetDocumentStoreFromRegistry("test-store")
	if exists {
		t.Error("GetDocumentStoreFromRegistry() store should not exist after unregistration")
	}
}

func TestDocumentStoreError_Error(t *testing.T) {
	tests := []struct {
		name      string
		storeName string
		operation string
		message   string
		filePath  string
		err       error
		expected  string
	}{
		{
			name:      "error_with_wrapped_error",
			storeName: "test-store",
			operation: "IndexDocument",
			message:   "failed to index",
			filePath:  "/path/to/file.txt",
			err:       fmt.Errorf("file not found"),
			expected:  "[test-store:IndexDocument] failed to index (file: /path/to/file.txt): file not found",
		},
		{
			name:      "error_without_wrapped_error",
			storeName: "test-store",
			operation: "IndexDocument",
			message:   "failed to index",
			filePath:  "/path/to/file.txt",
			err:       nil,
			expected:  "[test-store:IndexDocument] failed to index (file: /path/to/file.txt)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docErr := NewDocumentStoreError(tt.storeName, tt.operation, tt.message, tt.filePath, tt.err)
			errorStr := docErr.Error()

			if errorStr != tt.expected {
				t.Errorf("DocumentStoreError.Error() = %v, want %v", errorStr, tt.expected)
			}
		})
	}
}

func TestDocumentStoreError_Unwrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	docErr := NewDocumentStoreError("test-store", "IndexDocument", "failed", "/path/to/file.txt", originalErr)

	unwrapped := docErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("DocumentStoreError.Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestNewDocumentStoreError(t *testing.T) {
	storeName := "test-store"
	operation := "IndexDocument"
	message := "test error"
	filePath := "/path/to/file.txt"
	err := fmt.Errorf("wrapped error")

	docErr := NewDocumentStoreError(storeName, operation, message, filePath, err)

	if docErr.StoreName != storeName {
		t.Errorf("NewDocumentStoreError() StoreName = %v, want %v", docErr.StoreName, storeName)
	}
	if docErr.Operation != operation {
		t.Errorf("NewDocumentStoreError() Operation = %v, want %v", docErr.Operation, operation)
	}
	if docErr.Message != message {
		t.Errorf("NewDocumentStoreError() Message = %v, want %v", docErr.Message, message)
	}
	if docErr.FilePath != filePath {
		t.Errorf("NewDocumentStoreError() FilePath = %v, want %v", docErr.FilePath, filePath)
	}
	if docErr.Err != err {
		t.Errorf("NewDocumentStoreError() Err = %v, want %v", docErr.Err, err)
	}
	if docErr.Timestamp.IsZero() {
		t.Error("NewDocumentStoreError() Timestamp should not be zero")
	}
}
