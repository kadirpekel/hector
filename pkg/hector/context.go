package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// ContextServiceBuilder provides a fluent API for building context services (RAG)
type ContextServiceBuilder struct {
	database        databases.DatabaseProvider
	embedder        embedders.EmbedderProvider
	searchConfig    config.SearchConfig
	documentStores  []*config.DocumentStoreConfig
	includeContext  *bool
}

// NewContextService creates a new context service builder
func NewContextService() *ContextServiceBuilder {
	return &ContextServiceBuilder{
		documentStores: make([]*config.DocumentStoreConfig, 0),
		includeContext: boolPtr(true),
	}
}

// WithDatabase sets the database provider
func (b *ContextServiceBuilder) WithDatabase(db databases.DatabaseProvider) *ContextServiceBuilder {
	b.database = db
	return b
}

// WithEmbedder sets the embedder provider
func (b *ContextServiceBuilder) WithEmbedder(embedder embedders.EmbedderProvider) *ContextServiceBuilder {
	b.embedder = embedder
	return b
}

// WithSearchConfig sets the search configuration
func (b *ContextServiceBuilder) WithSearchConfig(cfg config.SearchConfig) *ContextServiceBuilder {
	b.searchConfig = cfg
	return b
}

// WithSearchModel adds a search model
func (b *ContextServiceBuilder) WithSearchModel(model config.SearchModel) *ContextServiceBuilder {
	if len(b.searchConfig.Models) == 0 {
		b.searchConfig.Models = make([]config.SearchModel, 0)
	}
	b.searchConfig.Models = append(b.searchConfig.Models, model)
	return b
}

// TopK sets the default top K results
func (b *ContextServiceBuilder) TopK(k int) *ContextServiceBuilder {
	b.searchConfig.TopK = k
	return b
}

// Threshold sets the similarity threshold
func (b *ContextServiceBuilder) Threshold(threshold float32) *ContextServiceBuilder {
	b.searchConfig.Threshold = threshold
	return b
}

// PreserveCase sets whether to preserve case in queries
func (b *ContextServiceBuilder) PreserveCase(preserve bool) *ContextServiceBuilder {
	b.searchConfig.PreserveCase = &preserve
	return b
}

// WithDocumentStore adds a document store configuration
func (b *ContextServiceBuilder) WithDocumentStore(store *config.DocumentStoreConfig) *ContextServiceBuilder {
	if store == nil {
		panic("document store cannot be nil")
	}
	b.documentStores = append(b.documentStores, store)
	return b
}

// WithDocumentStores adds multiple document store configurations
func (b *ContextServiceBuilder) WithDocumentStores(stores []*config.DocumentStoreConfig) *ContextServiceBuilder {
	for _, store := range stores {
		if store == nil {
			panic("document store cannot be nil")
		}
		b.documentStores = append(b.documentStores, store)
	}
	return b
}

// IncludeContext enables or disables context inclusion in prompts
func (b *ContextServiceBuilder) IncludeContext(include bool) *ContextServiceBuilder {
	b.includeContext = &include
	return b
}

// Build creates the context service
func (b *ContextServiceBuilder) Build() (reasoning.ContextService, error) {
	// If no document stores, return no-op service
	if len(b.documentStores) == 0 {
		return agent.NewNoOpContextService(), nil
	}

	// Validate required components
	if b.database == nil {
		return nil, fmt.Errorf("database provider is required for context service")
	}
	if b.embedder == nil {
		return nil, fmt.Errorf("embedder provider is required for context service")
	}

	// Set defaults for search config
	b.searchConfig.SetDefaults()

	// Create search engine
	searchEngine, err := context.NewSearchEngine(b.database, b.embedder, b.searchConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create search engine: %w", err)
	}

	// Initialize document stores
	if err := context.InitializeDocumentStoresFromConfig(b.documentStores, searchEngine); err != nil {
		return nil, fmt.Errorf("failed to initialize document stores: %w", err)
	}

	// Create context service
	return agent.NewContextService(searchEngine), nil
}

// GetIncludeContext returns whether to include context in prompts
func (b *ContextServiceBuilder) GetIncludeContext() *bool {
	return b.includeContext
}

