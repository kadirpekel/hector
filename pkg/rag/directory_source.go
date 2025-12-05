// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rag

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// DirectorySource implements DataSource for local filesystem directories.
//
// Direct port from legacy pkg/context/indexing/directory_source.go
type DirectorySource struct {
	basePath    string
	filter      FileFilter
	maxFileSize int64
}

// NewDirectorySource creates a new directory-based data source.
//
// Direct port from legacy pkg/context/indexing/directory_source.go
func NewDirectorySource(basePath string, filter FileFilter, maxFileSize int64) *DirectorySource {
	return &DirectorySource{
		basePath:    basePath,
		filter:      filter,
		maxFileSize: maxFileSize,
	}
}

// NewDirectorySourceFromConfig creates a directory source from config.
func NewDirectorySourceFromConfig(cfg DirectorySourceConfig) (DataSource, error) {
	filter, err := NewPatternFilter(cfg.Path, cfg.Include, cfg.Exclude)
	if err != nil {
		return nil, err
	}
	return NewDirectorySource(cfg.Path, filter, cfg.MaxFileSize), nil
}

// Type returns the data source type.
func (ds *DirectorySource) Type() string {
	return "directory"
}

// DiscoverDocuments returns channels of discovered documents and errors.
// Documents are discovered asynchronously and sent through the channel.
//
// Direct port from legacy pkg/context/indexing/directory_source.go
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
				ID:         path,
				Content:    string(content),
				SourcePath: relPath,
				MimeType:   detectMimeType(path),
				Size:       info.Size(),
				Metadata: map[string]any{
					"path":          path,
					"rel_path":      relPath,
					"name":          info.Name(),
					"absolute_path": path,
					"last_modified": info.ModTime().Unix(),
					"should_index":  shouldIndex,
				},
			}

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

// ReadDocument retrieves a specific document by its ID (file path).
//
// Direct port from legacy pkg/context/indexing/directory_source.go
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

	doc := &Document{
		ID:         id,
		Content:    string(content),
		SourcePath: relPath,
		MimeType:   detectMimeType(id),
		Size:       info.Size(),
		Metadata: map[string]any{
			"path":          id,
			"rel_path":      relPath,
			"name":          info.Name(),
			"absolute_path": id,
			"last_modified": info.ModTime().Unix(),
			"should_index":  true,
		},
	}

	return doc, nil
}

// SupportsIncrementalIndexing returns true as directory sources support incremental indexing.
func (ds *DirectorySource) SupportsIncrementalIndexing() bool {
	return true
}

// GetLastModified returns the last modification time for a document.
func (ds *DirectorySource) GetLastModified(ctx context.Context, id string) (time.Time, error) {
	info, err := os.Stat(id)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// Close releases any resources held by the data source.
func (ds *DirectorySource) Close() error {
	// No resources to close for directory source
	return nil
}

// GetBasePath returns the base directory path (helper method).
func (ds *DirectorySource) GetBasePath() string {
	return ds.basePath
}

// GetFilter returns the file filter (helper method).
func (ds *DirectorySource) GetFilter() FileFilter {
	return ds.filter
}

// Ensure DirectorySource implements DataSource.
var _ DataSource = (*DirectorySource)(nil)
