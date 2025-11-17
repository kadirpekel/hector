package databases

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/pinecone-io/go-pinecone/pinecone"
	"google.golang.org/protobuf/types/known/structpb"
)

func NewPineconeDatabaseProviderFromConfig(config *config.VectorStoreConfig) (DatabaseProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Pinecone")
	}

	// Create Pinecone client
	client, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: config.APIKey,
		Host:   config.Host, // Optional: Pinecone API host (defaults to https://api.pinecone.io)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Pinecone client: %w", err)
	}

	// Index name - use Host field from config as index name, or default
	indexName := config.Host
	if indexName == "" {
		indexName = "hector-index" // Default index name
	}

	return &pineconeDatabaseProvider{
		client:    client,
		config:    config,
		indexName: indexName,
	}, nil
}

type pineconeDatabaseProvider struct {
	client    *pinecone.Client
	config    *config.VectorStoreConfig
	indexName string
}

// getIndexConnection gets or creates an IndexConnection for the index
func (db *pineconeDatabaseProvider) getIndexConnection(ctx context.Context) (*pinecone.IndexConnection, error) {
	// First, describe the index to get its host URL
	index, err := db.client.DescribeIndex(ctx, db.indexName)
	if err != nil {
		return nil, fmt.Errorf("failed to describe index %s: %w", db.indexName, err)
	}

	// Create index connection using the host from the index
	indexConn, err := db.client.Index(pinecone.NewIndexConnParams{
		Host:      index.Host,
		Namespace: "", // Use default namespace
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create index connection: %w", err)
	}

	return indexConn, nil
}

func (db *pineconeDatabaseProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	// Pinecone uses index name, not collection name
	// Use collection if provided, otherwise use default index name
	actualIndexName := db.indexName
	if collection != "" && collection != db.indexName {
		actualIndexName = collection
	}

	// Temporarily override index name for this operation
	originalIndexName := db.indexName
	db.indexName = actualIndexName
	defer func() { db.indexName = originalIndexName }()

	// Get index connection
	indexConn, err := db.getIndexConnection(ctx)
	if err != nil {
		return err
	}
	defer indexConn.Close()

	// Convert metadata to structpb.Struct
	var pineconeMetadata *pinecone.Metadata
	if len(metadata) > 0 {
		pineconeMetadata, err = structpb.NewStruct(metadata)
		if err != nil {
			return fmt.Errorf("failed to convert metadata: %w", err)
		}
	}

	// Create vector (Pinecone uses float32, not float64)
	pineconeVector := &pinecone.Vector{
		Id:       id,
		Values:   vector, // Already float32
		Metadata: pineconeMetadata,
	}

	// Upsert vector
	_, err = indexConn.UpsertVectors(ctx, []*pinecone.Vector{pineconeVector})
	if err != nil {
		return fmt.Errorf("failed to upsert vector: %w", err)
	}

	return nil
}

func (db *pineconeDatabaseProvider) Search(ctx context.Context, collection string, queryVector []float32, topK int) ([]SearchResult, error) {
	return db.SearchWithFilter(ctx, collection, queryVector, topK, nil)
}

func (db *pineconeDatabaseProvider) SearchWithFilter(ctx context.Context, collection string, queryVector []float32, topK int, filter map[string]interface{}) ([]SearchResult, error) {
	// Use collection if provided, otherwise use default index name
	actualIndexName := db.indexName
	if collection != "" && collection != db.indexName {
		actualIndexName = collection
	}

	// Temporarily override index name for this operation
	originalIndexName := db.indexName
	db.indexName = actualIndexName
	defer func() { db.indexName = originalIndexName }()

	// Get index connection
	indexConn, err := db.getIndexConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer indexConn.Close()

	// Convert filter to MetadataFilter
	var metadataFilter *pinecone.MetadataFilter
	if len(filter) > 0 {
		metadataFilter, err = structpb.NewStruct(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to convert filter: %w", err)
		}
	}

	// Create query request
	queryRequest := &pinecone.QueryByVectorValuesRequest{
		Vector:          queryVector, // Already float32
		TopK:            uint32(topK),
		MetadataFilter:  metadataFilter,
		IncludeMetadata: true,
		IncludeValues:   true,
	}

	// Query vectors
	queryResponse, err := indexConn.QueryByVectorValues(ctx, queryRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to query Pinecone: %w", err)
	}

	return convertPineconeResults(queryResponse.Matches), nil
}

func convertPineconeResults(matches []*pinecone.ScoredVector) []SearchResult {
	results := make([]SearchResult, 0, len(matches))
	for _, scoredVector := range matches {
		if scoredVector.Vector == nil {
			continue
		}

		vector := scoredVector.Vector
		id := vector.Id
		score := scoredVector.Score

		// Extract vector values
		var vectorValues []float32
		if vector.Values != nil {
			vectorValues = vector.Values
		}

		// Extract metadata
		metadata := make(map[string]interface{})
		if vector.Metadata != nil {
			// Convert structpb.Struct to map[string]interface{}
			metadata = vector.Metadata.AsMap()
		}

		// Extract content from metadata if present
		content := ""
		if contentVal, exists := metadata["content"]; exists {
			if str, ok := contentVal.(string); ok {
				content = str
			}
		}

		results = append(results, SearchResult{
			ID:       id,
			Content:  content,
			Vector:   vectorValues,
			Metadata: metadata,
			Score:    score,
		})
	}

	return results
}

func (db *pineconeDatabaseProvider) Delete(ctx context.Context, collection string, id string) error {
	// Use collection if provided, otherwise use default index name
	actualIndexName := db.indexName
	if collection != "" && collection != db.indexName {
		actualIndexName = collection
	}

	// Temporarily override index name for this operation
	originalIndexName := db.indexName
	db.indexName = actualIndexName
	defer func() { db.indexName = originalIndexName }()

	// Get index connection
	indexConn, err := db.getIndexConnection(ctx)
	if err != nil {
		return err
	}
	defer indexConn.Close()

	// Delete vector by ID
	err = indexConn.DeleteVectorsById(ctx, []string{id})
	if err != nil {
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	return nil
}

func (db *pineconeDatabaseProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]interface{}) error {
	// Use collection if provided, otherwise use default index name
	actualIndexName := db.indexName
	if collection != "" && collection != db.indexName {
		actualIndexName = collection
	}

	// Temporarily override index name for this operation
	originalIndexName := db.indexName
	db.indexName = actualIndexName
	defer func() { db.indexName = originalIndexName }()

	// Get index connection
	indexConn, err := db.getIndexConnection(ctx)
	if err != nil {
		return err
	}
	defer indexConn.Close()

	// Convert filter to MetadataFilter
	var metadataFilter *pinecone.MetadataFilter
	if len(filter) > 0 {
		metadataFilter, err = structpb.NewStruct(filter)
		if err != nil {
			return fmt.Errorf("failed to convert filter: %w", err)
		}
	}

	// Delete vectors by filter
	err = indexConn.DeleteVectorsByFilter(ctx, metadataFilter)
	if err != nil {
		return fmt.Errorf("failed to delete by filter: %w", err)
	}

	return nil
}

func (db *pineconeDatabaseProvider) CreateCollection(ctx context.Context, collection string, vectorSize uint64) error {
	// Pinecone indexes are created via the Pinecone console or API separately
	// This method can check if index exists
	indexName := collection
	if collection == "" {
		indexName = db.indexName
	}

	// Check if index exists
	indexes, err := db.client.ListIndexes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list indexes: %w", err)
	}

	for _, idx := range indexes {
		if idx.Name == indexName {
			// Index exists
			return nil
		}
	}

	// Index doesn't exist - would need to create via Pinecone API
	// For now, return error indicating index must be created manually
	return fmt.Errorf("index %s does not exist. Please create it via Pinecone console or API", indexName)
}

func (db *pineconeDatabaseProvider) DeleteCollection(ctx context.Context, collection string) error {
	indexName := collection
	if collection == "" {
		indexName = db.indexName
	}

	// Pinecone index deletion requires API call
	// This would need to be implemented via Pinecone management API
	return fmt.Errorf("index deletion not implemented. Please delete index %s via Pinecone console or API", indexName)
}

func (db *pineconeDatabaseProvider) HybridSearch(ctx context.Context, collection string, query string, vector []float32, topK int, filter map[string]interface{}, alpha float32) ([]SearchResult, error) {
	// Pinecone doesn't natively support hybrid search, so we implement a fallback:
	// Perform vector search and filter by keywords, then fuse results
	
	if alpha >= 1.0 {
		// Pure vector search
		return db.SearchWithFilter(ctx, collection, vector, topK, filter)
	}
	
	// Get vector results
	vectorResults, err := db.SearchWithFilter(ctx, collection, vector, topK*2, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}
	
	// Filter by keywords (simple text matching)
	keywordResults := filterByKeywordsPinecone(vectorResults, query, topK*2)
	
	// Fuse results using Reciprocal Rank Fusion
	fusedResults := reciprocalRankFusionPinecone(vectorResults, keywordResults, alpha, topK)
	
	return fusedResults, nil
}

// filterByKeywordsPinecone filters results that contain query keywords
func filterByKeywordsPinecone(results []SearchResult, query string, limit int) []SearchResult {
	queryLower := strings.ToLower(query)
	keywords := strings.Fields(queryLower)
	
	filtered := make([]SearchResult, 0, len(results))
	for _, result := range results {
		contentLower := strings.ToLower(result.Content)
		matches := 0
		for _, keyword := range keywords {
			if strings.Contains(contentLower, keyword) {
				matches++
			}
		}
		if matches > 0 {
			keywordScore := float32(matches) / float32(len(keywords))
			result.Score = keywordScore
			filtered = append(filtered, result)
		}
	}
	
	// Sort by keyword score
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Score > filtered[j].Score
	})
	
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	
	return filtered
}

// reciprocalRankFusionPinecone combines results using RRF
func reciprocalRankFusionPinecone(vectorResults, keywordResults []SearchResult, alpha float32, topK int) []SearchResult {
	vectorRankMap := make(map[string]int)
	keywordRankMap := make(map[string]int)
	
	for i, result := range vectorResults {
		vectorRankMap[result.ID] = i + 1
	}
	for i, result := range keywordResults {
		keywordRankMap[result.ID] = i + 1
	}
	
	allIDs := make(map[string]bool)
	for _, result := range vectorResults {
		allIDs[result.ID] = true
	}
	for _, result := range keywordResults {
		allIDs[result.ID] = true
	}
	
	type scoredDoc struct {
		result SearchResult
		score  float32
	}
	scoredDocs := make([]scoredDoc, 0, len(allIDs))
	
	const rrfK = 60
	
	for id := range allIDs {
		var result SearchResult
		var vectorScore float32
		
		found := false
		for _, r := range vectorResults {
			if r.ID == id {
				result = r
				found = true
				vectorScore = r.Score
				break
			}
		}
		if !found {
			for _, r := range keywordResults {
				if r.ID == id {
					result = r
					break
				}
			}
		}
		
		vectorRRF := float32(0)
		if rank, exists := vectorRankMap[id]; exists {
			vectorRRF = 1.0 / float32(rrfK+rank)
		}
		
		keywordRRF := float32(0)
		if rank, exists := keywordRankMap[id]; exists {
			keywordRRF = 1.0 / float32(rrfK+rank)
		}
		
		blendedRRF := alpha*vectorRRF + (1-alpha)*keywordRRF
		blendedScore := alpha*vectorScore + (1-alpha)*result.Score
		finalScore := 0.7*blendedRRF + 0.3*blendedScore
		
		result.Score = finalScore
		scoredDocs = append(scoredDocs, scoredDoc{result: result, score: finalScore})
	}
	
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})
	
	results := make([]SearchResult, 0, topK)
	for i, sd := range scoredDocs {
		if i >= topK {
			break
		}
		results = append(results, sd.result)
	}
	
	return results
}

func (db *pineconeDatabaseProvider) Close() error {
	// Pinecone client doesn't have explicit close method
	return nil
}
