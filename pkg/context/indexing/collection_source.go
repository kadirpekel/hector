package indexing

import (
	"context"
	"fmt"
	"time"
)

// CollectionSource implements DataSource for collection-only stores
// It's a no-op source that doesn't index anything - used when document store
// points to an existing collection that's already populated
type CollectionSource struct {
	collectionName string
}

// NewCollectionSource creates a new collection-only data source
func NewCollectionSource(collectionName string) *CollectionSource {
	return &CollectionSource{
		collectionName: collectionName,
	}
}

func (cs *CollectionSource) Type() string {
	return "collection"
}

func (cs *CollectionSource) DiscoverDocuments(ctx context.Context) (<-chan Document, <-chan error) {
	// Return empty channels - no documents to index
	docChan := make(chan Document)
	errChan := make(chan error)

	go func() {
		defer close(docChan)
		defer close(errChan)
	}()

	return docChan, errChan
}

func (cs *CollectionSource) ReadDocument(ctx context.Context, id string) (*Document, error) {
	// Not used for collection-only stores (collection is pre-populated)
	return nil, fmt.Errorf("reading documents not supported for collection source")
}

func (cs *CollectionSource) SupportsIncrementalIndexing() bool {
	return false
}

func (cs *CollectionSource) GetLastModified(ctx context.Context, id string) (time.Time, error) {
	return time.Time{}, nil
}

func (cs *CollectionSource) Close() error {
	return nil
}
