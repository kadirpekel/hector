package databases

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/qdrant/go-client/qdrant"
)

func NewQdrantDatabaseProvider() (DatabaseProvider, error) {
	config := &config.VectorStoreConfig{
		Type:      "qdrant",
		Host:      "localhost",
		Port:      6334,
		EnableTLS: config.BoolPtr(false),
	}

	return NewQdrantDatabaseProviderFromConfig(config)
}

func NewQdrantDatabaseProviderFromConfig(config *config.VectorStoreConfig) (DatabaseProvider, error) {
	useTLS := false
	if config.EnableTLS != nil {
		useTLS = *config.EnableTLS
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   config.Host,
		Port:   config.Port,
		APIKey: config.APIKey,
		UseTLS: useTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client for %s:%d: %w\n"+
			"  💡 Troubleshooting:\n"+
			"     - Ensure Qdrant is running\n"+
			"     - Verify host and port configuration\n"+
			"     - For Docker: start Qdrant container (docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant)",
			config.Host, config.Port, err)
	}

	return &qdrantDatabaseProvider{
		client: client,
		config: config,
	}, nil
}

type qdrantDatabaseProvider struct {
	client *qdrant.Client
	config *config.VectorStoreConfig
}

func (db *qdrantDatabaseProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {

	exists, err := db.client.CollectionExists(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to connect to Qdrant at %s:%d (collection: %s): %w\n"+
			"  💡 Troubleshooting:\n"+
			"     - Ensure Qdrant is running (check: curl http://%s:%d/)\n"+
			"     - Verify the host and port are correct\n"+
			"     - Check network connectivity\n"+
			"     - For Docker: ensure container is running (docker ps | grep qdrant)\n"+
			"     - Check Qdrant logs for errors",
			db.config.Host, db.config.Port, collection, err,
			db.config.Host, db.config.Port)
	}

	if !exists {

		err = db.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: collection,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     uint64(len(vector)),
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {

			if strings.Contains(err.Error(), "already exists") {

			} else {
				return fmt.Errorf("failed to create collection: %w", err)
			}
		}
	}

	payload := make(map[string]*qdrant.Value)
	for key, value := range metadata {
		val, err := qdrant.NewValue(value)
		if err != nil {
			return fmt.Errorf("failed to convert metadata value for key %s: %w", key, err)
		}
		payload[key] = val
	}

	point := &qdrant.PointStruct{
		Id:      qdrant.NewID(id),
		Vectors: qdrant.NewVectors(vector...),
		Payload: payload,
	}

	_, err = db.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Points:         []*qdrant.PointStruct{point},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert point: %w", err)
	}

	return nil
}

func (db *qdrantDatabaseProvider) Search(ctx context.Context, collection string, queryVector []float32, topK int) ([]SearchResult, error) {
	return db.SearchWithFilter(ctx, collection, queryVector, topK, nil)
}

func (db *qdrantDatabaseProvider) SearchWithFilter(ctx context.Context, collection string, queryVector []float32, topK int, filter map[string]interface{}) ([]SearchResult, error) {

	searchRequest := &qdrant.SearchPoints{
		CollectionName: collection,
		Vector:         queryVector,
		Limit:          uint64(topK),
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(true),
	}

	if len(filter) > 0 {
		searchRequest.Filter = buildQdrantFilter(filter)
	}

	pointsClient := db.client.GetPointsClient()
	searchResult, err := pointsClient.Search(ctx, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search points: %w", err)
	}

	return convertQdrantResults(searchResult.Result), nil
}

func buildQdrantFilter(filter map[string]interface{}) *qdrant.Filter {
	conditions := make([]*qdrant.Condition, 0, len(filter))

	for key, value := range filter {

		val, err := qdrant.NewValue(value)
		if err != nil {
			continue
		}

		condition := &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key: key,
					Match: &qdrant.Match{
						MatchValue: &qdrant.Match_Keyword{
							Keyword: val.GetStringValue(),
						},
					},
				},
			},
		}
		conditions = append(conditions, condition)
	}

	return &qdrant.Filter{
		Must: conditions,
	}
}

func convertQdrantResults(points []*qdrant.ScoredPoint) []SearchResult {
	var results []SearchResult
	for _, point := range points {

		var id string
		if point.Id != nil {
			if point.Id.PointIdOptions != nil {
				switch idType := point.Id.PointIdOptions.(type) {
				case *qdrant.PointId_Uuid:
					id = idType.Uuid
				case *qdrant.PointId_Num:
					id = fmt.Sprintf("%d", idType.Num)
				}
			}
		}

		var vector []float32
		if point.Vectors != nil {
			if vectorData := point.Vectors.GetVector(); vectorData != nil {
				switch v := vectorData.Vector.(type) {
				case *qdrant.VectorOutput_Dense:
					if v.Dense != nil {
						vector = v.Dense.Data
					}
				default:

					vector = []float32{}
				}
			}
		}

		metadata := make(map[string]interface{})
		if point.Payload != nil {
			for key, value := range point.Payload {

				switch v := value.Kind.(type) {
				case *qdrant.Value_StringValue:
					metadata[key] = v.StringValue
				case *qdrant.Value_IntegerValue:
					metadata[key] = v.IntegerValue
				case *qdrant.Value_DoubleValue:
					metadata[key] = v.DoubleValue
				case *qdrant.Value_BoolValue:
					metadata[key] = v.BoolValue
				case *qdrant.Value_ListValue:

					if v.ListValue != nil {
						list := make([]interface{}, len(v.ListValue.Values))
						for i, item := range v.ListValue.Values {
							switch itemVal := item.Kind.(type) {
							case *qdrant.Value_StringValue:
								list[i] = itemVal.StringValue
							case *qdrant.Value_IntegerValue:
								list[i] = itemVal.IntegerValue
							case *qdrant.Value_DoubleValue:
								list[i] = itemVal.DoubleValue
							case *qdrant.Value_BoolValue:
								list[i] = itemVal.BoolValue
							default:
								list[i] = item
							}
						}
						metadata[key] = list
					}
				default:
					metadata[key] = value
				}
			}
		}

		score := point.Score

		content := ""
		if contentValue, exists := metadata["content"]; exists {
			if contentStr, ok := contentValue.(string); ok {
				content = contentStr
			}
		}

		results = append(results, SearchResult{
			ID:       id,
			Content:  content,
			Vector:   vector,
			Metadata: metadata,
			Score:    score,
		})
	}

	return results
}

func (db *qdrantDatabaseProvider) CreateCollection(ctx context.Context, collection string, vectorSize uint64) error {

	exists, err := db.client.CollectionExists(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to connect to Qdrant at %s:%d (collection: %s): %w\n"+
			"  💡 Troubleshooting:\n"+
			"     - Ensure Qdrant is running (check: curl http://%s:%d/)\n"+
			"     - Verify the host and port are correct\n"+
			"     - Check network connectivity\n"+
			"     - For Docker: ensure container is running (docker ps | grep qdrant)\n"+
			"     - Check Qdrant logs for errors",
			db.config.Host, db.config.Port, collection, err,
			db.config.Host, db.config.Port)
	}

	if exists {
		return nil
	}

	err = db.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

func (db *qdrantDatabaseProvider) Delete(ctx context.Context, collection string, id string) error {

	deletePoints := &qdrant.DeletePoints{
		CollectionName: collection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{
						{PointIdOptions: &qdrant.PointId_Uuid{Uuid: id}},
					},
				},
			},
		},
	}
	_, err := db.client.Delete(ctx, deletePoints)
	if err != nil {
		return fmt.Errorf("failed to delete point %s from collection %s: %w", id, collection, err)
	}
	return nil
}

func (db *qdrantDatabaseProvider) DeleteByFilter(ctx context.Context, collection string, filter map[string]interface{}) error {

	qdrantFilter := buildQdrantFilter(filter)

	deletePoints := &qdrant.DeletePoints{
		CollectionName: collection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: qdrantFilter,
			},
		},
	}

	_, err := db.client.Delete(ctx, deletePoints)
	if err != nil {
		return fmt.Errorf("failed to delete points by filter from collection %s: %w", collection, err)
	}
	return nil
}

func (db *qdrantDatabaseProvider) HybridSearch(ctx context.Context, collection string, query string, vector []float32, topK int, filter map[string]interface{}, alpha float32) ([]SearchResult, error) {
	// Qdrant supports hybrid search via QueryPoints API with sparse + dense vectors
	// For now, we'll use a fallback approach: parallel vector + keyword search with RRF fusion
	// This works even if Qdrant's native hybrid search isn't available in the Go client

	// If alpha is 0.0, pure keyword search (not supported yet, fallback to vector)
	// If alpha is 1.0, pure vector search
	// If alpha is 0.5, balanced hybrid

	queryPreview := query
	if len(query) > 50 {
		queryPreview = query[:50] + "..."
	}
	slog.Debug("Qdrant hybrid search", "collection", collection, "alpha", alpha, "query", queryPreview)

	if alpha >= 1.0 {
		// Pure vector search
		slog.Debug("Hybrid search: using pure vector (alpha >= 1.0)")
		return db.SearchWithFilter(ctx, collection, vector, topK, filter)
	}

	// For hybrid search, we'll do parallel searches and fuse results
	// This is a simplified implementation - full hybrid would use Qdrant's QueryPoints with sparse vectors

	// Get vector results
	vectorResults, err := db.SearchWithFilter(ctx, collection, vector, topK*2, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}

	// For keyword search, we'll use a simple text matching approach
	// In a full implementation, this would use BM25 or Qdrant's sparse vector search
	// For now, we'll filter vector results by keyword presence as a fallback
	keywordResults := filterByKeywords(vectorResults, query, topK*2)

	// Fuse results using Reciprocal Rank Fusion (RRF)
	fusedResults := reciprocalRankFusion(vectorResults, keywordResults, alpha, topK)

	return fusedResults, nil
}

// filterByKeywords filters results that contain query keywords
func filterByKeywords(results []SearchResult, query string, limit int) []SearchResult {
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
		// If at least one keyword matches, include the result
		if matches > 0 {
			// Score based on keyword match ratio
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

// reciprocalRankFusion combines results from vector and keyword searches using RRF
func reciprocalRankFusion(vectorResults, keywordResults []SearchResult, alpha float32, topK int) []SearchResult {
	// Create maps for quick lookup
	vectorRankMap := make(map[string]int)
	keywordRankMap := make(map[string]int)

	for i, result := range vectorResults {
		vectorRankMap[result.ID] = i + 1 // RRF uses 1-based ranking
	}
	for i, result := range keywordResults {
		keywordRankMap[result.ID] = i + 1
	}

	// Collect all unique document IDs
	allIDs := make(map[string]bool)
	for _, result := range vectorResults {
		allIDs[result.ID] = true
	}
	for _, result := range keywordResults {
		allIDs[result.ID] = true
	}

	// Calculate RRF scores
	type scoredDoc struct {
		result SearchResult
		score  float32
	}
	scoredDocs := make([]scoredDoc, 0, len(allIDs))

	const rrfK = 60 // RRF constant (standard value)

	for id := range allIDs {
		var result SearchResult
		var vectorScore float32

		// Find the result from either list
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

		// Calculate RRF scores
		vectorRRF := float32(0)
		if rank, exists := vectorRankMap[id]; exists {
			vectorRRF = 1.0 / float32(rrfK+rank)
		}

		keywordRRF := float32(0)
		if rank, exists := keywordRankMap[id]; exists {
			keywordRRF = 1.0 / float32(rrfK+rank)
		}

		// Blend scores: alpha * vector + (1-alpha) * keyword
		// For RRF, we blend the RRF scores, then optionally weight by original scores
		blendedRRF := alpha*vectorRRF + (1-alpha)*keywordRRF

		// Also consider original similarity scores
		blendedScore := alpha*vectorScore + (1-alpha)*result.Score

		// Final score: weighted combination of RRF and original scores
		finalScore := 0.7*blendedRRF + 0.3*blendedScore

		result.Score = finalScore
		scoredDocs = append(scoredDocs, scoredDoc{result: result, score: finalScore})
	}

	// Sort by final score
	sort.Slice(scoredDocs, func(i, j int) bool {
		return scoredDocs[i].score > scoredDocs[j].score
	})

	// Return top K
	results := make([]SearchResult, 0, topK)
	for i, sd := range scoredDocs {
		if i >= topK {
			break
		}
		results = append(results, sd.result)
	}

	return results
}

func (db *qdrantDatabaseProvider) DeleteCollection(ctx context.Context, collection string) error {
	err := db.client.DeleteCollection(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	return nil
}

func (db *qdrantDatabaseProvider) Close() error {
	return db.client.Close()
}
