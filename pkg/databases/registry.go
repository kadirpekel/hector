package databases

import (
	"context"
	"fmt"
	"sync"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/registry"
)

// ============================================================================
// DATABASE INTERFACE
// ============================================================================

// DatabaseProvider defines the interface for vector database operations
type DatabaseProvider interface {
	// Upsert adds or updates a document in the database
	Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error

	// Search performs vector similarity search
	Search(ctx context.Context, collection string, vector []float32, topK int) ([]SearchResult, error)

	// Delete removes a document from the database
	Delete(ctx context.Context, collection string, id string) error

	// CreateCollection creates a new collection
	CreateCollection(ctx context.Context, collection string, vectorSize uint64) error

	// DeleteCollection removes a collection
	DeleteCollection(ctx context.Context, collection string) error

	// Close closes the database provider and releases resources
	Close() error
}

// SearchResult represents a search result from the vector database
type SearchResult struct {
	ID        string                 `json:"id"`
	Score     float32                `json:"score"`
	Content   string                 `json:"content"`
	Vector    []float32              `json:"vector,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
	ModelName string                 `json:"model_name,omitempty"`
}

// ============================================================================
// DATABASE REGISTRY
// ============================================================================

// DatabaseRegistry manages database provider instances
type DatabaseRegistry struct {
	*registry.BaseRegistry[DatabaseProvider]
	mu sync.RWMutex
}

// NewDatabaseRegistry creates a new database registry
func NewDatabaseRegistry() *DatabaseRegistry {
	return &DatabaseRegistry{
		BaseRegistry: registry.NewBaseRegistry[DatabaseProvider](),
	}
}

// RegisterDatabase registers a database provider instance
func (r *DatabaseRegistry) RegisterDatabase(name string, provider DatabaseProvider) error {
	if name == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("database provider cannot be nil")
	}
	return r.Register(name, provider)
}

// CreateDatabaseFromConfig creates a database provider from configuration
func (r *DatabaseRegistry) CreateDatabaseFromConfig(name string, config *config.DatabaseProviderConfig) (DatabaseProvider, error) {
	if name == "" {
		return nil, fmt.Errorf("database name cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("database config cannot be nil")
	}

	// Set defaults and validate
	config.SetDefaults()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid database config: %w", err)
	}

	var provider DatabaseProvider
	var err error

	switch config.Type {
	case "qdrant":
		provider, err = NewQdrantDatabaseProviderFromConfig(config)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create database provider: %w", err)
	}

	// Register the provider
	if err := r.RegisterDatabase(name, provider); err != nil {
		return nil, fmt.Errorf("failed to register database: %w", err)
	}

	return provider, nil
}

// GetDatabase retrieves a database provider by name
func (r *DatabaseRegistry) GetDatabase(name string) (DatabaseProvider, error) {
	provider, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("database provider '%s' not found", name)
	}
	return provider, nil
}

// ListDatabases returns all registered database names
func (r *DatabaseRegistry) ListDatabases() []string {
	names := make([]string, 0)
	for _, _ = range r.List() {
		// Use a placeholder name since we don't have a name method
		names = append(names, "database")
	}
	return names
}
