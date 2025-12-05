// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rag

import (
	"context"
	"fmt"
	"time"
)

// CollectionSource implements DataSource for collection-only stores.
// It's a no-op source that doesn't index anything - used when document store
// points to an existing collection that's already populated.
//
// Direct port from legacy pkg/context/indexing/collection_source.go
type CollectionSource struct {
	collectionName string
}

// NewCollectionSource creates a new collection-only data source.
func NewCollectionSource(collectionName string) *CollectionSource {
	return &CollectionSource{
		collectionName: collectionName,
	}
}

// Type returns the data source type.
func (cs *CollectionSource) Type() string {
	return "collection"
}

// DiscoverDocuments returns empty channels - no documents to index.
func (cs *CollectionSource) DiscoverDocuments(ctx context.Context) (<-chan Document, <-chan error) {
	docChan := make(chan Document)
	errChan := make(chan error)

	go func() {
		defer close(docChan)
		defer close(errChan)
	}()

	return docChan, errChan
}

// ReadDocument returns an error - not supported for collection sources.
func (cs *CollectionSource) ReadDocument(ctx context.Context, id string) (*Document, error) {
	// Not used for collection-only stores (collection is pre-populated)
	return nil, fmt.Errorf("reading documents not supported for collection source")
}

// SupportsIncrementalIndexing returns false.
func (cs *CollectionSource) SupportsIncrementalIndexing() bool {
	return false
}

// GetLastModified returns zero time - not supported for collection sources.
func (cs *CollectionSource) GetLastModified(ctx context.Context, id string) (time.Time, error) {
	return time.Time{}, nil
}

// Close closes the collection source.
func (cs *CollectionSource) Close() error {
	return nil
}

// CollectionName returns the collection name.
func (cs *CollectionSource) CollectionName() string {
	return cs.collectionName
}

// Ensure CollectionSource implements DataSource.
var _ DataSource = (*CollectionSource)(nil)
