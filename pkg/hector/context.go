package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// documentStoreEntry holds a store name and its config
type documentStoreEntry struct {
	name   string
	config *config.DocumentStoreConfig
}

// ContextServiceBuilder provides a fluent API for building context services (RAG)
type ContextServiceBuilder struct {
	database         databases.DatabaseProvider
	embedder         embedders.EmbedderProvider
	searchConfig     config.SearchConfig
	documentStores   []documentStoreEntry
	includeContext   *bool
	componentManager *component.ComponentManager // For creating search engines with different databases
}

// NewContextService creates a new context service builder
func NewContextService() *ContextServiceBuilder {
	return &ContextServiceBuilder{
		documentStores: make([]documentStoreEntry, 0),
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

// WithComponentManager sets the component manager (for creating search engines with different databases)
func (b *ContextServiceBuilder) WithComponentManager(cm *component.ComponentManager) *ContextServiceBuilder {
	b.componentManager = cm
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
// storeName is the name from the config map key (used as collection name unless overridden)
func (b *ContextServiceBuilder) WithDocumentStore(storeName string, store *config.DocumentStoreConfig) *ContextServiceBuilder {
	if storeName == "" {
		panic("document store name cannot be empty")
	}
	if store == nil {
		panic("document store cannot be nil")
	}
	b.documentStores = append(b.documentStores, documentStoreEntry{name: storeName, config: store})
	return b
}

// WithDocumentStores adds multiple document store configurations
// storeNames is a slice of store names (map keys) corresponding to stores
func (b *ContextServiceBuilder) WithDocumentStores(storeNames []string, stores []*config.DocumentStoreConfig) *ContextServiceBuilder {
	if len(storeNames) != len(stores) {
		panic("storeNames and stores must have the same length")
	}
	for i, store := range stores {
		if storeNames[i] == "" {
			panic("document store name cannot be empty")
		}
		if store == nil {
			panic("document store cannot be nil")
		}
		b.documentStores = append(b.documentStores, documentStoreEntry{name: storeNames[i], config: store})
	}
	return b
}

// WithDocumentStoreBuilder adds a document store using a builder
// storeName is the name from the config map key (used as collection name unless overridden)
func (b *ContextServiceBuilder) WithDocumentStoreBuilder(storeName string, storeBuilder *DocumentStoreBuilder) *ContextServiceBuilder {
	if storeName == "" {
		panic("document store name cannot be empty")
	}
	if storeBuilder == nil {
		panic("document store builder cannot be nil")
	}
	store, err := storeBuilder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build document store: %v", err))
	}
	return b.WithDocumentStore(storeName, store)
}

// WithDocumentStoreBuilders adds multiple document stores using builders
// storeNames is a slice of store names (map keys) corresponding to builders
func (b *ContextServiceBuilder) WithDocumentStoreBuilders(storeNames []string, storeBuilders []*DocumentStoreBuilder) *ContextServiceBuilder {
	if len(storeNames) != len(storeBuilders) {
		panic("storeNames and storeBuilders must have the same length")
	}
	for i, storeBuilder := range storeBuilders {
		if storeNames[i] == "" {
			panic("document store name cannot be empty")
		}
		if storeBuilder == nil {
			panic("document store builder cannot be nil")
		}
		b.WithDocumentStoreBuilder(storeNames[i], storeBuilder)
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

	// Create default search engine (for stores that don't specify their own database)
	defaultSearchEngine, err := context.NewSearchEngine(b.database, b.embedder, b.searchConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create default search engine: %w", err)
	}

	// Initialize document stores - create separate search engines for stores with their own database
	if err := b.initializeDocumentStoresWithDatabases(defaultSearchEngine); err != nil {
		return nil, fmt.Errorf("failed to initialize document stores: %w", err)
	}

	// Create context service (use default search engine for context service)
	return agent.NewContextService(defaultSearchEngine), nil
}

// initializeDocumentStoresWithDatabases initializes document stores, creating separate search engines
// for stores that specify their own database/embedder
func (b *ContextServiceBuilder) initializeDocumentStoresWithDatabases(defaultSearchEngine *context.SearchEngine) error {
	for _, entry := range b.documentStores {
		storeName := entry.name
		storeConfig := entry.config
		var searchEngine *context.SearchEngine

		// If store specifies its own vector_store/embedder, create a separate search engine
		if storeConfig.VectorStore != "" || storeConfig.Embedder != "" {
			if b.componentManager == nil {
				return fmt.Errorf("component manager required for document store '%s' with custom vector_store/embedder", storeName)
			}

			// Get vector store (use store's vector_store or default)
			db := b.database
			if storeConfig.VectorStore != "" {
				var err error
				db, err = b.componentManager.GetDatabase(storeConfig.VectorStore)
				if err != nil {
					return fmt.Errorf("failed to get vector store '%s' for document store '%s': %w", storeConfig.VectorStore, storeName, err)
				}
			}

			// Get embedder (use store's embedder or default)
			embedder := b.embedder
			if storeConfig.Embedder != "" {
				var err error
				embedder, err = b.componentManager.GetEmbedder(storeConfig.Embedder)
				if err != nil {
					return fmt.Errorf("failed to get embedder '%s' for document store '%s': %w", storeConfig.Embedder, storeName, err)
				}
			}

			// Create search engine for this store
			var err error
			searchEngine, err = context.NewSearchEngine(db, embedder, b.searchConfig)
			if err != nil {
				return fmt.Errorf("failed to create search engine for document store '%s': %w", storeName, err)
			}
		} else {
			// Use default search engine
			searchEngine = defaultSearchEngine
		}

		// Create and register document store
		store, err := context.NewDocumentStore(storeName, storeConfig, searchEngine)
		if err != nil {
			return fmt.Errorf("failed to create document store '%s': %w", storeName, err)
		}

		context.RegisterDocumentStore(store)

		// Start indexing (will skip for collection-only stores)
		if err := store.StartIndexing(); err != nil {
			return fmt.Errorf("failed to index document store '%s': %w", storeName, err)
		}

		// Start watching if enabled
		if storeConfig.EnableWatchChanges != nil && *storeConfig.EnableWatchChanges {
			go func(s *context.DocumentStore, name string) {
				if err := s.StartWatching(); err != nil {
					fmt.Printf("Warning: Failed to start file watching for %s: %v\n", name, err)
				}
			}(store, storeName)
		}
	}

	return nil
}

// GetIncludeContext returns whether to include context in prompts
func (b *ContextServiceBuilder) GetIncludeContext() *bool {
	return b.includeContext
}
