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
	"time"
)

// DataSource represents a generic source of documents to be indexed.
// It abstracts over filesystem, SQL databases, REST APIs, and cloud storage.
//
// Direct port from legacy pkg/context/indexing/data_source.go
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

// SourceDocument represents a document from any source (file, SQL row, API response, etc.)
//
// Direct port from legacy pkg/context/indexing/data_source.go:Document
// Renamed to SourceDocument to avoid conflict with v2/rag Document type
type SourceDocument struct {
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

// FileFilter determines if a file should be indexed.
//
// Direct port from legacy pkg/context/indexing/data_source.go:FileFilter
type FileFilter interface {
	ShouldInclude(path string) bool
	ShouldExclude(path string) bool
}

// NilDataSource is a no-op data source that returns no documents.
type NilDataSource struct{}

func (NilDataSource) Type() string { return "nil" }

func (NilDataSource) DiscoverDocuments(ctx context.Context) (<-chan Document, <-chan error) {
	docChan := make(chan Document)
	errChan := make(chan error)
	close(docChan)
	close(errChan)
	return docChan, errChan
}

func (NilDataSource) ReadDocument(ctx context.Context, id string) (*Document, error) {
	return nil, nil
}

func (NilDataSource) SupportsIncrementalIndexing() bool {
	return false
}

func (NilDataSource) GetLastModified(ctx context.Context, id string) (time.Time, error) {
	return time.Time{}, nil
}

func (NilDataSource) Close() error {
	return nil
}

// Ensure NilDataSource implements DataSource.
var _ DataSource = NilDataSource{}
