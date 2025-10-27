package indexing

import (
	"context"
	"os"
)

// FileSource represents a source of files to be indexed
type FileSource interface {
	// DiscoverFiles returns a channel of discovered files and a channel of errors
	DiscoverFiles(ctx context.Context) (<-chan FileInfo, <-chan error)

	// GetBasePath returns the base path for relative path calculations
	GetBasePath() string

	// SupportsIncrementalIndexing indicates if this source supports incremental updates
	SupportsIncrementalIndexing() bool
}

// FileInfo contains information about a discovered file
type FileInfo struct {
	Path        string      // Absolute path to the file
	RelPath     string      // Relative path from base
	Info        os.FileInfo // File metadata
	ShouldIndex bool        // Whether this file should be indexed (after filtering)
}

// FileFilter determines if a file should be indexed
type FileFilter interface {
	ShouldInclude(path string) bool
	ShouldExclude(path string) bool
}
