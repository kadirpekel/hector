package indexing

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// DirectorySource implements DataSource for local filesystem directories
type DirectorySource struct {
	basePath    string
	filter      FileFilter
	maxFileSize int64
}

// NewDirectorySource creates a new directory-based data source
func NewDirectorySource(basePath string, filter FileFilter, maxFileSize int64) *DirectorySource {
	return &DirectorySource{
		basePath:    basePath,
		filter:      filter,
		maxFileSize: maxFileSize,
	}
}

func (ds *DirectorySource) Type() string {
	return "directory"
}

func (ds *DirectorySource) DiscoverDocuments(ctx context.Context) (<-chan Document, <-chan error) {
	docChan := make(chan Document, 100)
	errChan := make(chan error, 10)

	go func() {
		defer close(docChan)
		defer close(errChan)

		err := filepath.Walk(ds.basePath, func(path string, info os.FileInfo, err error) error {
			// Check context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err != nil {
				// Non-fatal error, log and continue
				select {
				case errChan <- err:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}

			// Handle directories
			if info.IsDir() {
				// Early pruning: skip excluded directories
				if ds.filter != nil && ds.filter.ShouldExclude(path) {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip empty files
			if info.Size() == 0 {
				return nil
			}

			// Skip files exceeding max size
			if ds.maxFileSize > 0 && info.Size() > ds.maxFileSize {
				return nil
			}

			relPath, _ := filepath.Rel(ds.basePath, path)

			// Apply filters
			shouldIndex := true
			if ds.filter != nil {
				if ds.filter.ShouldExclude(path) || !ds.filter.ShouldInclude(path) {
					shouldIndex = false
				}
			}

			// Read file content
			content, err := os.ReadFile(path)
			if err != nil {
				select {
				case errChan <- err:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}

			// Create document with content
			doc := Document{
				ID:           path,
				Content:      string(content),
				Metadata:     make(map[string]interface{}),
				LastModified: info.ModTime(),
				Size:         info.Size(),
				ShouldIndex:  shouldIndex,
				SourcePath:   relPath,
			}
			doc.Metadata["path"] = path
			doc.Metadata["rel_path"] = relPath
			doc.Metadata["name"] = info.Name()
			doc.Metadata["absolute_path"] = path

			select {
			case docChan <- doc:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})

		if err != nil && err != context.Canceled {
			select {
			case errChan <- err:
			case <-ctx.Done():
			}
		}
	}()

	return docChan, errChan
}

func (ds *DirectorySource) ReadDocument(ctx context.Context, id string) (*Document, error) {
	// ID is the file path
	info, err := os.Stat(id)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(id)
	if err != nil {
		return nil, err
	}

	relPath, _ := filepath.Rel(ds.basePath, id)

	doc := Document{
		ID:           id,
		Content:      string(content),
		Metadata:     make(map[string]interface{}),
		LastModified: info.ModTime(),
		Size:         info.Size(),
		ShouldIndex:  true,
		SourcePath:   relPath,
	}
	doc.Metadata["path"] = id
	doc.Metadata["rel_path"] = relPath
	doc.Metadata["name"] = info.Name()
	doc.Metadata["absolute_path"] = id

	return &doc, nil
}

func (ds *DirectorySource) SupportsIncrementalIndexing() bool {
	return true
}

func (ds *DirectorySource) GetLastModified(ctx context.Context, id string) (time.Time, error) {
	info, err := os.Stat(id)
			if err != nil {
		return time.Time{}, err
				}
	return info.ModTime(), nil
			}

func (ds *DirectorySource) Close() error {
	// No resources to close for directory source
				return nil
			}

// GetBasePath returns the base directory path (helper method)
func (ds *DirectorySource) GetBasePath() string {
	return ds.basePath
}

// GetFilter returns the file filter (helper method)
func (ds *DirectorySource) GetFilter() FileFilter {
	return ds.filter
}
