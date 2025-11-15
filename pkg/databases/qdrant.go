package databases

import (
	"context"
	"fmt"
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
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
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
		return fmt.Errorf("failed to check if collection exists: %w", err)
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
		return fmt.Errorf("failed to check if collection exists: %w", err)
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
