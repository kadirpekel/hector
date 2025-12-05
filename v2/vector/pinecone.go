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

package vector

import (
	"context"
	"fmt"

	"github.com/pinecone-io/go-pinecone/pinecone"
	"google.golang.org/protobuf/types/known/structpb"
)

// PineconeConfig configures the Pinecone vector provider.
//
// Direct port from legacy pkg/databases/pinecone.go
type PineconeConfig struct {
	// APIKey is required for Pinecone authentication.
	APIKey string `yaml:"api_key"`

	// Host is the Pinecone API host (optional, defaults to https://api.pinecone.io).
	Host string `yaml:"host,omitempty"`

	// IndexName is the default index to use.
	IndexName string `yaml:"index_name"`

	// Environment is the Pinecone environment (e.g., "us-west1-gcp").
	Environment string `yaml:"environment,omitempty"`
}

// PineconeProvider implements Provider using Pinecone vector database.
//
// Direct port from legacy pkg/databases/pinecone.go
type PineconeProvider struct {
	client    *pinecone.Client
	config    PineconeConfig
	indexName string
}

// NewPineconeProvider creates a new Pinecone provider.
func NewPineconeProvider(cfg PineconeConfig) (*PineconeProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Pinecone")
	}

	clientParams := pinecone.NewClientParams{
		ApiKey: cfg.APIKey,
	}
	if cfg.Host != "" {
		clientParams.Host = cfg.Host
	}

	client, err := pinecone.NewClient(clientParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create Pinecone client: %w", err)
	}

	indexName := cfg.IndexName
	if indexName == "" {
		indexName = "hector-index"
	}

	return &PineconeProvider{
		client:    client,
		config:    cfg,
		indexName: indexName,
	}, nil
}

// Name returns the provider name.
func (p *PineconeProvider) Name() string {
	return "pinecone"
}

// getIndexConnection gets an IndexConnection for the index.
func (p *PineconeProvider) getIndexConnection(ctx context.Context, indexName string) (*pinecone.IndexConnection, error) {
	index, err := p.client.DescribeIndex(ctx, indexName)
	if err != nil {
		return nil, fmt.Errorf("failed to describe index %s: %w", indexName, err)
	}

	indexConn, err := p.client.Index(pinecone.NewIndexConnParams{
		Host:      index.Host,
		Namespace: "",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create index connection: %w", err)
	}

	return indexConn, nil
}

// Upsert adds or updates a document with its vector.
func (p *PineconeProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]any) error {
	indexName := collection
	if indexName == "" {
		indexName = p.indexName
	}

	indexConn, err := p.getIndexConnection(ctx, indexName)
	if err != nil {
		return err
	}
	defer indexConn.Close()

	// Convert metadata to structpb.Struct
	var pineconeMetadata *pinecone.Metadata
	if len(metadata) > 0 {
		// Convert map[string]any to map[string]interface{}
		metadataInterface := make(map[string]interface{}, len(metadata))
		for k, v := range metadata {
			metadataInterface[k] = v
		}
		pineconeMetadata, err = structpb.NewStruct(metadataInterface)
		if err != nil {
			return fmt.Errorf("failed to convert metadata: %w", err)
		}
	}

	pineconeVector := &pinecone.Vector{
		Id:       id,
		Values:   vector,
		Metadata: pineconeMetadata,
	}

	_, err = indexConn.UpsertVectors(ctx, []*pinecone.Vector{pineconeVector})
	if err != nil {
		return fmt.Errorf("failed to upsert vector: %w", err)
	}

	return nil
}

// Search finds the most similar vectors.
func (p *PineconeProvider) Search(ctx context.Context, collection string, vector []float32, topK int) ([]Result, error) {
	return p.SearchWithFilter(ctx, collection, vector, topK, nil)
}

// SearchWithFilter combines vector similarity with metadata filtering.
func (p *PineconeProvider) SearchWithFilter(ctx context.Context, collection string, vector []float32, topK int, filter map[string]any) ([]Result, error) {
	indexName := collection
	if indexName == "" {
		indexName = p.indexName
	}

	indexConn, err := p.getIndexConnection(ctx, indexName)
	if err != nil {
		return nil, err
	}
	defer indexConn.Close()

	// Convert filter to MetadataFilter
	var metadataFilter *pinecone.MetadataFilter
	if len(filter) > 0 {
		filterInterface := make(map[string]interface{}, len(filter))
		for k, v := range filter {
			filterInterface[k] = v
		}
		metadataFilter, err = structpb.NewStruct(filterInterface)
		if err != nil {
			return nil, fmt.Errorf("failed to convert filter: %w", err)
		}
	}

	queryRequest := &pinecone.QueryByVectorValuesRequest{
		Vector:          vector,
		TopK:            uint32(topK),
		MetadataFilter:  metadataFilter,
		IncludeMetadata: true,
		IncludeValues:   true,
	}

	queryResponse, err := indexConn.QueryByVectorValues(ctx, queryRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to query Pinecone: %w", err)
	}

	return convertPineconeResults(queryResponse.Matches), nil
}

// Delete removes a document by ID.
func (p *PineconeProvider) Delete(ctx context.Context, collection string, id string) error {
	indexName := collection
	if indexName == "" {
		indexName = p.indexName
	}

	indexConn, err := p.getIndexConnection(ctx, indexName)
	if err != nil {
		return err
	}
	defer indexConn.Close()

	err = indexConn.DeleteVectorsById(ctx, []string{id})
	if err != nil {
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	return nil
}

// DeleteByFilter removes all documents matching the filter.
func (p *PineconeProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]any) error {
	indexName := collection
	if indexName == "" {
		indexName = p.indexName
	}

	indexConn, err := p.getIndexConnection(ctx, indexName)
	if err != nil {
		return err
	}
	defer indexConn.Close()

	// Convert filter to MetadataFilter
	var metadataFilter *pinecone.MetadataFilter
	if len(filter) > 0 {
		filterInterface := make(map[string]interface{}, len(filter))
		for k, v := range filter {
			filterInterface[k] = v
		}
		metadataFilter, err = structpb.NewStruct(filterInterface)
		if err != nil {
			return fmt.Errorf("failed to convert filter: %w", err)
		}
	}

	err = indexConn.DeleteVectorsByFilter(ctx, metadataFilter)
	if err != nil {
		return fmt.Errorf("failed to delete by filter: %w", err)
	}

	return nil
}

// CreateCollection checks if the index exists (Pinecone indexes must be created separately).
func (p *PineconeProvider) CreateCollection(ctx context.Context, collection string, vectorDimension int) error {
	indexName := collection
	if indexName == "" {
		indexName = p.indexName
	}

	indexes, err := p.client.ListIndexes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list indexes: %w", err)
	}

	for _, idx := range indexes {
		if idx.Name == indexName {
			return nil // Index exists
		}
	}

	return fmt.Errorf("index %s does not exist. Please create it via Pinecone console or API", indexName)
}

// DeleteCollection returns an error (Pinecone index deletion requires API).
func (p *PineconeProvider) DeleteCollection(ctx context.Context, collection string) error {
	indexName := collection
	if indexName == "" {
		indexName = p.indexName
	}
	return fmt.Errorf("index deletion not implemented. Please delete index %s via Pinecone console or API", indexName)
}

// Close closes the Pinecone client.
func (p *PineconeProvider) Close() error {
	// Pinecone client doesn't have explicit close method
	return nil
}

// convertPineconeResults converts Pinecone results to our Result type.
func convertPineconeResults(matches []*pinecone.ScoredVector) []Result {
	results := make([]Result, 0, len(matches))

	for _, scoredVector := range matches {
		if scoredVector.Vector == nil {
			continue
		}

		vector := scoredVector.Vector
		id := vector.Id
		score := scoredVector.Score

		var vectorValues []float32
		if vector.Values != nil {
			vectorValues = vector.Values
		}

		metadata := make(map[string]any)
		if vector.Metadata != nil {
			for k, v := range vector.Metadata.AsMap() {
				metadata[k] = v
			}
		}

		content := ""
		if contentVal, exists := metadata["content"]; exists {
			if str, ok := contentVal.(string); ok {
				content = str
			}
		}

		results = append(results, Result{
			ID:       id,
			Content:  content,
			Vector:   vectorValues,
			Metadata: metadata,
			Score:    score,
		})
	}

	return results
}

// Ensure PineconeProvider implements Provider.
var _ Provider = (*PineconeProvider)(nil)
