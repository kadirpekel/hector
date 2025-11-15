package databases

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/registry"
)

type DatabaseProvider interface {
	Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error

	Search(ctx context.Context, collection string, vector []float32, topK int) ([]SearchResult, error)

	SearchWithFilter(ctx context.Context, collection string, vector []float32, topK int, filter map[string]interface{}) ([]SearchResult, error)

	Delete(ctx context.Context, collection string, id string) error

	DeleteByFilter(ctx context.Context, collection string, filter map[string]interface{}) error

	CreateCollection(ctx context.Context, collection string, vectorSize uint64) error

	DeleteCollection(ctx context.Context, collection string) error

	Close() error
}

type SearchResult struct {
	ID        string                 `json:"id"`
	Score     float32                `json:"score"`
	Content   string                 `json:"content"`
	Vector    []float32              `json:"vector,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
	ModelName string                 `json:"model_name,omitempty"`
}

type DatabaseRegistry struct {
	*registry.BaseRegistry[DatabaseProvider]
}

func NewDatabaseRegistry() *DatabaseRegistry {
	return &DatabaseRegistry{
		BaseRegistry: registry.NewBaseRegistry[DatabaseProvider](),
	}
}

func (r *DatabaseRegistry) RegisterDatabase(name string, provider DatabaseProvider) error {
	if name == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	if provider == nil {
		return fmt.Errorf("database provider cannot be nil")
	}
	return r.Register(name, provider)
}

func (r *DatabaseRegistry) CreateDatabaseFromConfig(name string, config *config.VectorStoreConfig) (DatabaseProvider, error) {
	if name == "" {
		return nil, fmt.Errorf("database name cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("database config cannot be nil")
	}

	var provider DatabaseProvider
	var err error

	switch config.Type {
	case "qdrant":
		provider, err = NewQdrantDatabaseProviderFromConfig(config)
	case "pinecone":
		provider, err = NewPineconeDatabaseProviderFromConfig(config)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create database provider: %w", err)
	}

	if err := r.RegisterDatabase(name, provider); err != nil {
		return nil, fmt.Errorf("failed to register database: %w", err)
	}

	return provider, nil
}

func (r *DatabaseRegistry) GetDatabase(name string) (DatabaseProvider, error) {
	provider, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("database provider '%s' not found", name)
	}
	return provider, nil
}

func (r *DatabaseRegistry) ListDatabases() []string {
	names := make([]string, 0)
	for range r.List() {

		names = append(names, "database")
	}
	return names
}
