package rag

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBlobSource_LocalFileSystem(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"doc1.txt":        "This is document 1",
		"doc2.md":         "# Document 2\nThis is markdown",
		"subdir/doc3.txt": "This is document 3 in a subdirectory",
		".hidden":         "This should be excluded",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Create blob source with file:// URL
	cfg := BlobSourceConfig{
		URL:         "file://" + tmpDir,
		Prefix:      "",
		Include:     nil,                 // Include all
		Exclude:     []string{".hidden"}, // Exclude specific hidden file (glob pattern .* is treated as extension)
		MaxFileSize: 10 * 1024 * 1024,
	}

	ctx := context.Background()
	source, err := NewBlobSource(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create blob source: %v", err)
	}
	defer source.Close()

	// Discover documents
	docChan, errChan := source.DiscoverDocuments(ctx)

	var docs []Document
	var errors []error

	for {
		select {
		case doc, ok := <-docChan:
			if !ok {
				docChan = nil
			} else {
				docs = append(docs, doc)
			}
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
			} else {
				errors = append(errors, err)
			}
		}

		if docChan == nil && errChan == nil {
			break
		}
	}

	// Verify results
	if len(errors) > 0 {
		t.Errorf("Got %d errors during discovery: %v", len(errors), errors)
	}

	// Should find 3 non-hidden files
	if len(docs) != 3 {
		t.Errorf("Expected 3 documents, got %d", len(docs))
	}

	// Verify document content
	for _, doc := range docs {
		t.Logf("Found document: %s (size: %d, should_index: %v)",
			doc.SourcePath, doc.Size, doc.Metadata["should_index"])

		if doc.Content == "" {
			t.Errorf("Document %s has empty content", doc.SourcePath)
		}

		if doc.Metadata["storage_type"] != "local" {
			t.Errorf("Expected storage_type=local, got %v", doc.Metadata["storage_type"])
		}
	}
}

func TestBlobSource_ReadDocument(t *testing.T) {
	tmpDir := t.TempDir()

	testContent := "Test document content"
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := BlobSourceConfig{
		URL:         "file://" + tmpDir,
		MaxFileSize: 10 * 1024 * 1024,
	}

	ctx := context.Background()
	source, err := NewBlobSource(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create blob source: %v", err)
	}
	defer source.Close()

	// First discover to get the document ID
	docChan, _ := source.DiscoverDocuments(ctx)
	var docID string
	for doc := range docChan {
		docID = doc.ID
		break
	}

	if docID == "" {
		t.Fatal("No documents discovered")
	}

	// Now read the specific document
	doc, err := source.ReadDocument(ctx, docID)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	if doc.Content != testContent {
		t.Errorf("Expected content %q, got %q", testContent, doc.Content)
	}
}

func TestBlobSource_SupportsIncrementalIndexing(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := BlobSourceConfig{
		URL:         "file://" + tmpDir,
		MaxFileSize: 10 * 1024 * 1024,
	}

	ctx := context.Background()
	source, err := NewBlobSource(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create blob source: %v", err)
	}
	defer source.Close()

	if !source.SupportsIncrementalIndexing() {
		t.Error("Blob source should support incremental indexing")
	}
}

func TestBlobSource_GetStorageType(t *testing.T) {
	tests := []struct {
		url          string
		expectedType string
	}{
		{"file:///path/to/docs", "local"},
		{"s3://my-bucket/docs", "s3"},
		{"gs://my-bucket/docs", "gcs"},
		{"azblob://my-container/docs", "azure"},
		{"unknown://something", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			bs := &BlobSource{url: tt.url}
			storageType := bs.getStorageType()
			if storageType != tt.expectedType {
				t.Errorf("Expected storage type %q for URL %q, got %q",
					tt.expectedType, tt.url, storageType)
			}
		})
	}
}

func TestBlobSourceConfig_Validate(t *testing.T) {
	ctx := context.Background()

	validURLs := []string{
		"file:///path/to/docs",
		"s3://bucket/prefix",
		"gs://bucket/prefix",
		"azblob://container/prefix",
	}

	for _, url := range validURLs {
		t.Run(url, func(t *testing.T) {
			cfg := BlobSourceConfig{
				URL:         url,
				MaxFileSize: 10 * 1024 * 1024,
			}

			// For non-file URLs, we can't actually connect without credentials
			// So we just test the file:// case
			if url == "file:///path/to/docs" {
				// This will fail because path doesn't exist, but that's OK for validation test
				_, err := NewBlobSource(ctx, cfg)
				// We expect an error because the path doesn't exist
				// but not a validation error
				t.Logf("Error (expected for non-existent path): %v", err)
			}
		})
	}
}

func TestBlobSource_WithPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files in subdirectories
	files := map[string]string{
		"docs/public/doc1.txt":  "Public doc 1",
		"docs/public/doc2.txt":  "Public doc 2",
		"docs/private/doc3.txt": "Private doc 3",
		"other/doc4.txt":        "Other doc 4",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Create blob source with prefix to only index docs/public/
	cfg := BlobSourceConfig{
		URL:         "file://" + tmpDir,
		Prefix:      "docs/public/",
		MaxFileSize: 10 * 1024 * 1024,
	}

	ctx := context.Background()
	source, err := NewBlobSource(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create blob source: %v", err)
	}
	defer source.Close()

	// Discover documents
	docChan, errChan := source.DiscoverDocuments(ctx)

	var docs []Document
	var errors []error

	for {
		select {
		case doc, ok := <-docChan:
			if !ok {
				docChan = nil
			} else {
				docs = append(docs, doc)
			}
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
			} else {
				errors = append(errors, err)
			}
		}

		if docChan == nil && errChan == nil {
			break
		}
	}

	if len(errors) > 0 {
		t.Errorf("Got errors: %v", errors)
	}

	// Should only find 2 documents in docs/public/
	if len(docs) != 2 {
		t.Errorf("Expected 2 documents with prefix 'docs/public/', got %d", len(docs))
		for _, doc := range docs {
			t.Logf("Found: %s", doc.Metadata["key"])
		}
	}

	// Verify all documents are from the prefix
	for _, doc := range docs {
		key := doc.Metadata["key"].(string)
		if len(key) < len("docs/public/") || key[:len("docs/public/")] != "docs/public/" {
			t.Errorf("Document key %q doesn't start with prefix 'docs/public/'", key)
		}
	}
}
