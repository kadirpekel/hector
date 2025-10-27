package indexing

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
)

// DirectorySource implements FileSource for directory-based file discovery
type DirectorySource struct {
	basePath    string
	filter      FileFilter
	maxFileSize int64
}

// NewDirectorySource creates a new directory-based file source
func NewDirectorySource(basePath string, filter FileFilter, maxFileSize int64) *DirectorySource {
	return &DirectorySource{
		basePath:    basePath,
		filter:      filter,
		maxFileSize: maxFileSize,
	}
}

// DiscoverFiles walks the directory tree and returns discovered files
func (ds *DirectorySource) DiscoverFiles(ctx context.Context) (<-chan FileInfo, <-chan error) {
	filesChan := make(chan FileInfo, 100)
	errorsChan := make(chan error, 10)

	go func() {
		defer close(filesChan)
		defer close(errorsChan)

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
				case errorsChan <- err:
				default:
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

			fileInfo := FileInfo{
				Path:        path,
				RelPath:     relPath,
				Info:        info,
				ShouldIndex: shouldIndex,
			}

			select {
			case filesChan <- fileInfo:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})

		if err != nil && err != context.Canceled {
			select {
			case errorsChan <- err:
			default:
			}
		}
	}()

	return filesChan, errorsChan
}

// GetBasePath returns the base directory path
func (ds *DirectorySource) GetBasePath() string {
	return ds.basePath
}

// GetFilter returns the file filter
func (ds *DirectorySource) GetFilter() FileFilter {
	return ds.filter
}

// SupportsIncrementalIndexing returns true for directory sources
func (ds *DirectorySource) SupportsIncrementalIndexing() bool {
	return true
}

// DirectorySourceStats contains statistics about file discovery
type DirectorySourceStats struct {
	TotalFiles    int64
	SkippedFiles  int64
	ExcludedFiles int64
	ErrorCount    int64
}

// DiscoverFilesWithStats is like DiscoverFiles but also returns statistics
func (ds *DirectorySource) DiscoverFilesWithStats(ctx context.Context) (<-chan FileInfo, <-chan error, *DirectorySourceStats) {
	stats := &DirectorySourceStats{}
	filesChan := make(chan FileInfo, 100)
	errorsChan := make(chan error, 10)

	go func() {
		defer close(filesChan)
		defer close(errorsChan)

		err := filepath.Walk(ds.basePath, func(path string, info os.FileInfo, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err != nil {
				atomic.AddInt64(&stats.ErrorCount, 1)
				select {
				case errorsChan <- err:
				default:
				}
				return nil
			}

			if info.IsDir() {
				if ds.filter != nil && ds.filter.ShouldExclude(path) {
					return filepath.SkipDir
				}
				return nil
			}

			atomic.AddInt64(&stats.TotalFiles, 1)

			if info.Size() == 0 {
				atomic.AddInt64(&stats.SkippedFiles, 1)
				return nil
			}

			if ds.maxFileSize > 0 && info.Size() > ds.maxFileSize {
				atomic.AddInt64(&stats.SkippedFiles, 1)
				return nil
			}

			relPath, _ := filepath.Rel(ds.basePath, path)
			shouldIndex := true

			if ds.filter != nil {
				if ds.filter.ShouldExclude(path) || !ds.filter.ShouldInclude(path) {
					shouldIndex = false
					atomic.AddInt64(&stats.ExcludedFiles, 1)
				}
			}

			fileInfo := FileInfo{
				Path:        path,
				RelPath:     relPath,
				Info:        info,
				ShouldIndex: shouldIndex,
			}

			select {
			case filesChan <- fileInfo:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})

		if err != nil && err != context.Canceled {
			atomic.AddInt64(&stats.ErrorCount, 1)
			select {
			case errorsChan <- err:
			default:
			}
		}
	}()

	return filesChan, errorsChan, stats
}
