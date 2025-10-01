package databases

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/config"
	"github.com/qdrant/go-client/qdrant"
)

// ============================================================================
// QDRANT PROVIDER CONFIGURATION
// ============================================================================

// QdrantConfig is defined in config/providers.go

// Methods SetDefaults, GetProviderType, and GetProviderName are defined in config/providers.go

// Methods Validate and SetDefaults are defined in config/providers.go

// ============================================================================
// QDRANT PROVIDER IMPLEMENTATION
// ============================================================================

// NewQdrantDatabaseProvider creates a new Qdrant vector database with default configuration
func NewQdrantDatabaseProvider() (DatabaseProvider, error) {
	config := &config.DatabaseProviderConfig{
		Type:    "qdrant",
		Host:    "localhost",
		Port:    6334,
		Timeout: 30,
		UseTLS:  false,
	}

	return NewQdrantDatabaseProviderFromConfig(config)
}

// NewQdrantDatabaseProviderFromConfig creates a new Qdrant vector database from config
func NewQdrantDatabaseProviderFromConfig(config *config.DatabaseProviderConfig) (DatabaseProvider, error) {
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create Qdrant client
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   config.Host,
		Port:   config.Port,
		APIKey: config.APIKey,
		UseTLS: config.UseTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	return &qdrantDatabaseProvider{
		client: client,
		config: config,
	}, nil
}

// ============================================================================
// QDRANT DATABASE IMPLEMENTATION
// ============================================================================

// newQdrantDatabaseProvider creates a new Qdrant vector database

// qdrantDatabaseProvider is a Qdrant vector database implementation
type qdrantDatabaseProvider struct {
	client *qdrant.Client
	config *config.DatabaseProviderConfig
}

// Upsert adds or updates a vector in the database
func (db *qdrantDatabaseProvider) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	// Check if collection exists, create if it doesn't
	exists, err := db.client.CollectionExists(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to check if collection exists: %w", err)
	}

	if !exists {
		// Create collection with vector size based on the provided vector
		err = db.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: collection,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     uint64(len(vector)),
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			// Handle the case where collection was created by another concurrent operation
			if strings.Contains(err.Error(), "already exists") {
				// Collection was created by another process, continue with upsert
			} else {
				return fmt.Errorf("failed to create collection: %w", err)
			}
		}
	}

	// Convert metadata to Qdrant format
	payload := make(map[string]*qdrant.Value)
	for key, value := range metadata {
		val, err := qdrant.NewValue(value)
		if err != nil {
			return fmt.Errorf("failed to convert metadata value for key %s: %w", key, err)
		}
		payload[key] = val
	}

	// Create point
	point := &qdrant.PointStruct{
		Id:      qdrant.NewID(id),
		Vectors: qdrant.NewVectors(vector...),
		Payload: payload,
	}

	// Upsert point
	_, err = db.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collection,
		Points:         []*qdrant.PointStruct{point},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert point: %w", err)
	}

	return nil
}

// Search performs vector similarity search
func (db *qdrantDatabaseProvider) Search(ctx context.Context, collection string, queryVector []float32, topK int) ([]SearchResult, error) {
	// Create search request
	searchRequest := &qdrant.SearchPoints{
		CollectionName: collection,
		Vector:         queryVector,
		Limit:          uint64(topK),
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectors(true),
	}

	// Perform search using the Points client
	pointsClient := db.client.GetPointsClient()
	searchResult, err := pointsClient.Search(ctx, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search points: %w", err)
	}

	// Convert results
	var results []SearchResult
	for _, point := range searchResult.Result {
		// Extract ID
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

		// Extract vector
		var vector []float32
		if point.Vectors != nil {
			if vectorData := point.Vectors.GetVector(); vectorData != nil {
				switch v := vectorData.Vector.(type) {
				case *qdrant.VectorOutput_Dense:
					if v.Dense != nil {
						vector = v.Dense.Data
					}
				default:
					// Handle other vector types or nil case
					vector = []float32{}
				}
			}
		}

		// Extract metadata
		metadata := make(map[string]interface{})
		if point.Payload != nil {
			for key, value := range point.Payload {
				// Convert Qdrant Value back to interface{}
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
					// Convert list value to Go slice
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

		// Extract score
		score := point.Score

		// Extract content from metadata if available
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

	return results, nil
}

// CreateCollection creates a collection if it doesn't exist
func (db *qdrantDatabaseProvider) CreateCollection(ctx context.Context, collection string, vectorSize uint64) error {
	// Check if collection exists
	exists, err := db.client.CollectionExists(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to check if collection exists: %w", err)
	}

	if exists {
		return nil // Collection already exists
	}

	// Create collection
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

// Delete removes a document from the database
func (db *qdrantDatabaseProvider) Delete(ctx context.Context, collection string, id string) error {
	// Delete a specific point by ID
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

// DeleteCollection removes a collection
func (db *qdrantDatabaseProvider) DeleteCollection(ctx context.Context, collection string) error {
	err := db.client.DeleteCollection(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	return nil
}

// Close closes the Qdrant client
func (db *qdrantDatabaseProvider) Close() error {
	return db.client.Close()
}
