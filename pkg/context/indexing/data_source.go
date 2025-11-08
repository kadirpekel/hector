package indexing

import (
	"context"
	"time"
)

// DataSource represents a generic source of documents to be indexed.
// It abstracts over filesystem, SQL databases, REST APIs, and cloud storage.
type DataSource interface {
	// Type returns the type of data source (e.g., "directory", "sql", "api", "s3")
	Type() string

	// DiscoverDocuments returns a channel of discovered documents and a channel of errors.
	// Documents are discovered asynchronously and sent through the channel.
	// For file sources, content should be read from files.
	// For SQL/API sources, content should already be populated.
	DiscoverDocuments(ctx context.Context) (<-chan Document, <-chan error)

	// ReadDocument retrieves a specific document by its ID.
	// The ID format depends on the source type (file path, SQL row ID, API endpoint, etc.)
	ReadDocument(ctx context.Context, id string) (*Document, error)

	// SupportsIncrementalIndexing indicates if this source supports incremental updates
	// based on modification timestamps or change tracking.
	SupportsIncrementalIndexing() bool

	// GetLastModified returns the last modification time for a document, if available.
	// Returns zero time if not supported or document doesn't exist.
	GetLastModified(ctx context.Context, id string) (time.Time, error)

	// Close releases any resources held by the data source.
	Close() error
}

// Document represents a document from any source (file, SQL row, API response, etc.)
type Document struct {
	// ID is a unique identifier for the document (format depends on source type)
	ID string

	// Content is the text content to be indexed.
	// For file sources, this should be populated by reading the file.
	// For SQL/API sources, this is populated during discovery.
	Content string

	// Metadata contains source-specific metadata (file path, table name, API endpoint, etc.)
	Metadata map[string]interface{}

	// LastModified is the last modification time, if available
	LastModified time.Time

	// Size is the size of the document in bytes (approximate for non-file sources)
	Size int64

	// ShouldIndex indicates whether this document should be indexed (after filtering)
	ShouldIndex bool

	// SourcePath is the original source path (file path, table name, API endpoint, etc.)
	// This is used for relative path calculations and display purposes
	SourcePath string
}

// FileFilter determines if a file should be indexed
type FileFilter interface {
	ShouldInclude(path string) bool
	ShouldExclude(path string) bool
}
