package databases

import (
	"context"
	"fmt"
	"time"

	"github.com/kadirpekel/hector/interfaces"
	"github.com/qdrant/go-client/qdrant"
)

// ============================================================================
// QDRANT PROVIDER CONFIGURATION
// ============================================================================

// QdrantConfig holds configuration for the Qdrant database provider
type QdrantConfig struct {
	Provider string `yaml:"provider"` // Always "qdrant"
	Host     string `yaml:"host"`     // Database host
	Port     int    `yaml:"port"`     // Database port
	APIKey   string `yaml:"api_key"`  // API key (optional)
	Timeout  int    `yaml:"timeout"`  // Connection timeout in seconds
	UseTLS   bool   `yaml:"use_tls"`  // Use TLS connection
	Insecure bool   `yaml:"insecure"` // Allow insecure TLS
}

// SetDefaults sets default values for QdrantConfig
func (c *QdrantConfig) SetDefaults() {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 6334 // Use gRPC port (required by qdrant go-client)
	}
	if c.Timeout == 0 {
		c.Timeout = 30 // Standard timeout
	}
	// UseTLS defaults to false (Go zero value)
	// Insecure defaults to false (Go zero value)
}

// GetProviderType implements ProviderConfig.GetProviderType
func (c *QdrantConfig) GetProviderType() interfaces.ProviderType {
	return interfaces.ProviderTypeDatabase
}

// GetProviderName implements ProviderConfig.GetProviderName
func (c *QdrantConfig) GetProviderName() string {
	return "qdrant"
}

// CreateProvider implements ProviderConfig.CreateProvider
func (c *QdrantConfig) CreateProvider() (interface{}, error) {
	// Set defaults before creating provider
	c.SetDefaults()

	config := &qdrantConfig{
		Host:     c.Host,
		Port:     c.Port,
		APIKey:   c.APIKey,
		Timeout:  time.Duration(c.Timeout) * time.Second,
		UseTLS:   c.UseTLS,
		Insecure: c.Insecure,
	}

	return newQdrantVectorDBFromConfig(config)
}

// Validate implements ProviderConfig.Validate
func (c *QdrantConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

// ============================================================================
// QDRANT INTERNAL CONFIGURATION
// ============================================================================

// qdrantConfig holds configuration for Qdrant database
type qdrantConfig struct {
	Host     string
	Port     int
	APIKey   string
	Timeout  time.Duration
	UseTLS   bool
	Insecure bool
}

// DefaultQdrantConfig returns a default Qdrant configuration
func DefaultQdrantConfig() *qdrantConfig {
	return &qdrantConfig{
		Host:     "localhost",
		Port:     6334, // Use gRPC port (required by qdrant go-client)
		Timeout:  30 * time.Second,
		UseTLS:   false,
		Insecure: false,
	}
}

// qdrantOption is a function that configures qdrantConfig
type qdrantOption func(*qdrantConfig)

// withHost sets the Qdrant host
func withHost(host string) qdrantOption {
	return func(c *qdrantConfig) {
		c.Host = host
	}
}

// withPort sets the Qdrant port
func withPort(port int) qdrantOption {
	return func(c *qdrantConfig) {
		c.Port = port
	}
}

// withAPIKey sets the Qdrant API key
func withAPIKey(apiKey string) qdrantOption {
	return func(c *qdrantConfig) {
		c.APIKey = apiKey
	}
}

// withTimeout sets the connection timeout
func withTimeout(timeout time.Duration) qdrantOption {
	return func(c *qdrantConfig) {
		c.Timeout = timeout
	}
}

// withTLS enables TLS connection
func withTLS() qdrantOption {
	return func(c *qdrantConfig) {
		c.UseTLS = true
	}
}

// withInsecureTLS allows insecure TLS connections
func withInsecureTLS() qdrantOption {
	return func(c *qdrantConfig) {
		c.UseTLS = true
		c.Insecure = true
	}
}

// newQdrantVectorDBFromConfig creates a new Qdrant vector database from config
func newQdrantVectorDBFromConfig(config *qdrantConfig) (VectorDB, error) {
	// Create Qdrant client with simple configuration
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   config.Host,
		Port:   config.Port,
		APIKey: config.APIKey,
		UseTLS: config.UseTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	return &qdrantVectorDB{
		client: client,
		config: config,
	}, nil
}

// NewQdrantVectorDB creates a new Qdrant vector database (legacy method)
func NewQdrantVectorDB() (VectorDB, error) {
	config := &QdrantConfig{
		Provider: "qdrant",
		Host:     "localhost",
		Port:     6334,
		Timeout:  30,
		UseTLS:   false,
		Insecure: false,
	}

	provider, err := config.CreateProvider()
	if err != nil {
		return nil, err
	}
	return provider.(VectorDB), nil
}

// NewQdrantVectorDBFromConfig creates a new Qdrant vector database from config
func NewQdrantVectorDBFromConfig(config *QdrantConfig) (VectorDB, error) {
	provider, err := config.CreateProvider()
	if err != nil {
		return nil, err
	}
	return provider.(VectorDB), nil
}

// ============================================================================
// QDRANT DATABASE IMPLEMENTATION
// ============================================================================

// newQdrantVectorDB creates a new Qdrant vector database
func newQdrantVectorDB(opts ...qdrantOption) (VectorDB, error) {
	config := DefaultQdrantConfig()
	for _, opt := range opts {
		opt(config)
	}

	// Create Qdrant client with simple configuration
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   config.Host,
		Port:   config.Port,
		APIKey: config.APIKey,
		UseTLS: config.UseTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	return &qdrantVectorDB{
		client: client,
		config: config,
	}, nil
}

// qdrantVectorDB is a Qdrant vector database implementation
type qdrantVectorDB struct {
	client *qdrant.Client
	config *qdrantConfig
}

// Upsert adds or updates a vector in the database
func (db *qdrantVectorDB) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
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
			return fmt.Errorf("failed to create collection: %w", err)
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
func (db *qdrantVectorDB) Search(ctx context.Context, collection string, queryVector []float32, topK int) ([]SearchResult, error) {
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
				if uuid := point.Id.PointIdOptions.(*qdrant.PointId_Uuid); uuid != nil {
					id = uuid.Uuid
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

		results = append(results, SearchResult{
			ID:       id,
			Vector:   vector,
			Metadata: metadata,
			Score:    score,
		})
	}

	return results, nil
}

// CreateCollection creates a collection if it doesn't exist
func (db *qdrantVectorDB) CreateCollection(ctx context.Context, collection string, vectorSize uint64) error {
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
func (db *qdrantVectorDB) Delete(ctx context.Context, collection string, id string) error {
	// For now, we'll implement a simple delete by ID
	// This is a simplified implementation - in production you'd want proper error handling
	return fmt.Errorf("delete operation not yet implemented for Qdrant")
}

// DeleteCollection removes a collection
func (db *qdrantVectorDB) DeleteCollection(ctx context.Context, collection string) error {
	err := db.client.DeleteCollection(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	return nil
}

// Close closes the Qdrant client
func (db *qdrantVectorDB) Close() error {
	return db.client.Close()
}
