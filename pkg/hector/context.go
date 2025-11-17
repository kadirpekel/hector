package hector

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/context/extraction"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/tools"
)

// toolRegistryAdapter bridges tools.ToolRegistry to extraction.ToolCaller
// This avoids import cycles by keeping the adapter in the hector package
type toolRegistryAdapter struct {
	registry *tools.ToolRegistry
}

func (a *toolRegistryAdapter) GetTool(name string) (extraction.Tool, error) {
	tool, err := a.registry.GetTool(name)
	if err != nil {
		return nil, err
	}
	return &toolAdapter{tool: tool}, nil
}

// toolAdapter bridges tools.Tool to extraction.Tool
type toolAdapter struct {
	tool tools.Tool
}

func (a *toolAdapter) GetInfo() extraction.ToolInfo {
	info := a.tool.GetInfo()
	params := make([]extraction.ToolParameter, len(info.Parameters))
	for i, p := range info.Parameters {
		params[i] = extraction.ToolParameter{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
			Required:    p.Required,
		}
	}
	return extraction.ToolInfo{
		Name:        info.Name,
		Description: info.Description,
		Parameters:  params,
	}
}

func (a *toolAdapter) Execute(ctx context.Context, args map[string]interface{}) (extraction.ToolResult, error) {
	result, err := a.tool.Execute(ctx, args)
	if err != nil {
		return extraction.ToolResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}
	return extraction.ToolResult{
		Success:  result.Success,
		Content:  result.Content,
		Error:    result.Error,
		Metadata: result.Metadata,
	}, nil
}

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
	accessAllStores  bool                        // If true, agent has access to all stores (DocumentStores was nil)
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
// accessAllStores indicates if agent has access to all stores (DocumentStores was nil)
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

// WithAccessAllStores marks that the agent has access to all stores (DocumentStores was nil)
func (b *ContextServiceBuilder) WithAccessAllStores(accessAll bool) *ContextServiceBuilder {
	b.accessAllStores = accessAll
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
	// Pass LLM registry if component manager is available (for reranking)
	var llmRegistry hectorcontext.LLMRegistryForReranking
	if b.componentManager != nil {
		llmRegistry = b.componentManager.GetLLMRegistry()
	}
	defaultSearchEngine, err := hectorcontext.NewSearchEngine(b.database, b.embedder, b.searchConfig, llmRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create default search engine: %w", err)
	}

	// Initialize document stores - create separate search engines for stores with their own database
	if err := b.initializeDocumentStoresWithDatabases(defaultSearchEngine); err != nil {
		return nil, fmt.Errorf("failed to initialize document stores: %w\n"+
			"  💡 This usually indicates a connection issue with required services.\n"+
			"     Check the error details above for specific service connection problems.", err)
	}

	// Extract store names for scoping the context service
	storeNames := make([]string, 0, len(b.documentStores))
	for _, entry := range b.documentStores {
		storeNames = append(storeNames, entry.name)
	}

	// Create context service scoped to assigned document stores
	// If accessAllStores is true, pass nil (means all stores)
	// Otherwise, pass store names (scoped access)
	// Note: Empty slice [] means no stores, nil means all stores
	var storesForService []string
	if len(storeNames) == 0 {
		// This shouldn't happen if Build() is called, but handle gracefully
		return agent.NewNoOpContextService(), nil
	}
	if b.accessAllStores {
		// Pass nil to indicate "all stores" access
		storesForService = nil
	} else {
		// Pass store names for scoped access
		storesForService = storeNames
	}

	return agent.NewContextServiceWithStores(defaultSearchEngine, storesForService), nil
}

// initializeDocumentStoresWithDatabases initializes document stores, creating separate search engines
// for stores that specify their own database/embedder
func (b *ContextServiceBuilder) initializeDocumentStoresWithDatabases(defaultSearchEngine *hectorcontext.SearchEngine) error {
	for _, entry := range b.documentStores {
		storeName := entry.name
		storeConfig := entry.config
		var searchEngine *hectorcontext.SearchEngine

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
			// Pass LLM registry if component manager is available (for reranking)
			var llmRegistry hectorcontext.LLMRegistryForReranking
			if b.componentManager != nil {
				llmRegistry = b.componentManager.GetLLMRegistry()
			}
			var err error
			searchEngine, err = hectorcontext.NewSearchEngine(db, embedder, b.searchConfig, llmRegistry)
			if err != nil {
				return fmt.Errorf("failed to create search engine for document store '%s': %w", storeName, err)
			}
		} else {
			// Use default search engine
			searchEngine = defaultSearchEngine
		}

		// Get tool registry if available (for MCP parsers)
		var toolCaller extraction.ToolCaller
		if b.componentManager != nil {
			toolReg := b.componentManager.GetToolRegistry()
			if toolReg != nil {
				// Create adapter to bridge tools.ToolRegistry to extraction.ToolCaller
				toolCaller = &toolRegistryAdapter{registry: toolReg}
			}
		}

		// Create and register document store
		store, err := hectorcontext.NewDocumentStoreWithToolRegistry(storeName, storeConfig, searchEngine, toolCaller)
		if err != nil {
			return fmt.Errorf("failed to create document store '%s' (source: %s): %w", storeName, storeConfig.Source, err)
		}

		hectorcontext.RegisterDocumentStore(store)

		// Start indexing (will skip for collection-only stores)
		if err := store.StartIndexing(); err != nil {
			return fmt.Errorf("failed to index document store '%s': %w", storeName, err)
		}

		// Start watching if enabled
		if storeConfig.EnableWatchChanges != nil && *storeConfig.EnableWatchChanges {
			go func(s *hectorcontext.DocumentStore, name string) {
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
