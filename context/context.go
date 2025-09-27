package context

import (
	"context"
	"fmt"
	"sync"
	"time"

	hectorconfig "github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/providers"
)

// ============================================================================
// CONTEXT CONSTANTS AND CONFIGURATION
// ============================================================================

const (
	// DefaultMaxMessages is the default maximum number of messages to keep
	DefaultMaxMessages = 100

	// DefaultSearchLimit is the default search result limit
	DefaultSearchLimit = 10

	// DefaultContextTimeout is the default context timeout
	DefaultContextTimeout = 30 * time.Second

	// DefaultIndexingTimeout is the default indexing timeout
	DefaultIndexingTimeout = 5 * time.Minute

	// DefaultHealthCheckTimeout is the default timeout for health checks
	DefaultHealthCheckTimeout = 10 * time.Second

	// MaxConcurrentOperations is the maximum number of concurrent operations
	MaxConcurrentOperations = 100
)

// ============================================================================
// CONTEXT ERRORS - ENHANCED ERROR TYPES
// ============================================================================

// ContextError represents errors in the context package
type ContextError struct {
	Component string
	Operation string
	Message   string
	Err       error
	Timestamp time.Time
}

func (e *ContextError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Component, e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Component, e.Operation, e.Message)
}

func (e *ContextError) Unwrap() error {
	return e.Err
}

// NewContextError creates a new context error
func NewContextError(component, operation, message string, err error) *ContextError {
	return &ContextError{
		Component: component,
		Operation: operation,
		Message:   message,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// ============================================================================
// CONTEXT MANAGER - ENHANCED UNIFIED MANAGEMENT
// ============================================================================

// ContextManager manages all context-related components with enhanced features
type ContextManager struct {
	mu       sync.RWMutex
	factory  *ContextFactory
	status   *ContextManagerStatus
	
	// Core components
	searchEngine   *SearchEngine
	documentStores map[string]*DocumentStore
	conversations  map[string]*ConversationHistory
	
	// Operation control
	operationSemaphore chan struct{}
}

// ContextManagerStatus represents the status of the context manager
type ContextManagerStatus struct {
	Initialized     bool      `json:"initialized"`
	SearchEngine    bool      `json:"search_engine"`
	DocumentStores  int       `json:"document_stores"`
	Conversations   int       `json:"conversations"`
	LastUpdated     time.Time `json:"last_updated"`
	Healthy         bool      `json:"healthy"`
}

// NewContextManager creates a new context manager with enhanced initialization
func NewContextManager(
	llmProvider providers.LLMProvider,
	embedderProvider providers.EmbedderProvider,
	databaseProvider providers.DatabaseProvider,
) (*ContextManager, error) {
	if llmProvider == nil {
		return nil, NewContextError("ContextManager", "NewContextManager", "LLM provider is required", nil)
	}
	if embedderProvider == nil {
		return nil, NewContextError("ContextManager", "NewContextManager", "embedder provider is required", nil)
	}
	if databaseProvider == nil {
		return nil, NewContextError("ContextManager", "NewContextManager", "database provider is required", nil)
	}

	factory := NewContextFactory(llmProvider, embedderProvider, databaseProvider)
	
	return &ContextManager{
		factory:  factory,
		status: &ContextManagerStatus{
			Initialized:    false,
			SearchEngine:   false,
			DocumentStores: 0,
			Conversations:  0,
			LastUpdated:    time.Now(),
			Healthy:        true,
		},
		documentStores: make(map[string]*DocumentStore),
		conversations:  make(map[string]*ConversationHistory),
		operationSemaphore: make(chan struct{}, MaxConcurrentOperations),
	}, nil
}

// ============================================================================
// INITIALIZATION AND CONFIGURATION
// ============================================================================

// Initialize initializes the context manager with configuration
func (cm *ContextManager) Initialize(ctx context.Context, config *hectorconfig.HectorConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.status.Initialized {
		return NewContextError("ContextManager", "Initialize", "context manager already initialized", nil)
	}

	// Validate configuration
	if config == nil {
		return NewContextError("ContextManager", "Initialize", "configuration is required", nil)
	}

	// Create search engine with default config
	searchConfig := hectorconfig.SearchConfig{}
	searchEngine, err := cm.factory.CreateSearchEngine(searchConfig)
	if err != nil {
		return NewContextError("ContextManager", "Initialize", "failed to create search engine", err)
	}
	cm.searchEngine = searchEngine
	cm.status.SearchEngine = true

	// Initialize document stores
	if len(config.DocumentStores) > 0 {
		// Convert map to slice
		storeConfigs := make([]hectorconfig.DocumentStoreConfig, 0, len(config.DocumentStores))
		for _, storeConfig := range config.DocumentStores {
			storeConfigs = append(storeConfigs, storeConfig)
		}

		err := InitializeDocumentStoresFromConfig(storeConfigs, cm.searchEngine)
		if err != nil {
			return NewContextError("ContextManager", "Initialize", "failed to initialize document stores", err)
		}
		cm.status.DocumentStores = len(storeConfigs)
	}

	cm.status.Initialized = true
	cm.status.LastUpdated = time.Now()

	return nil
}

// ============================================================================
// SEARCH ENGINE MANAGEMENT
// ============================================================================

// GetSearchEngine returns the search engine
func (cm *ContextManager) GetSearchEngine() *SearchEngine {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.searchEngine
}

// CreateSearchEngine creates a new search engine with custom configuration
func (cm *ContextManager) CreateSearchEngine(config hectorconfig.SearchConfig) (*SearchEngine, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.status.Initialized {
		return nil, NewContextError("ContextManager", "CreateSearchEngine", "context manager not initialized", nil)
	}

	searchEngine, err := cm.factory.CreateSearchEngine(config)
	if err != nil {
		return nil, NewContextError("ContextManager", "CreateSearchEngine", "failed to create search engine", err)
	}

	// Replace the current search engine
	cm.searchEngine = searchEngine
	cm.status.SearchEngine = true
	cm.status.LastUpdated = time.Now()

	return searchEngine, nil
}

// ============================================================================
// DOCUMENT STORE MANAGEMENT
// ============================================================================

// GetDocumentStore returns a document store by name
func (cm *ContextManager) GetDocumentStore(name string) (*DocumentStore, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	store, exists := cm.documentStores[name]
	return store, exists
}

// CreateDocumentStore creates a new document store
func (cm *ContextManager) CreateDocumentStore(config *hectorconfig.DocumentStoreConfig) (*DocumentStore, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.status.Initialized {
		return nil, NewContextError("ContextManager", "CreateDocumentStore", "context manager not initialized", nil)
	}

	store, err := cm.factory.CreateDocumentStore(config)
	if err != nil {
		return nil, NewContextError("ContextManager", "CreateDocumentStore", "failed to create document store", err)
	}

	// Register in local map
	cm.documentStores[config.Name] = store
	cm.status.DocumentStores = len(cm.documentStores)
	cm.status.LastUpdated = time.Now()

	return store, nil
}

// ListDocumentStores returns all document store names
func (cm *ContextManager) ListDocumentStores() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var names []string
	for name := range cm.documentStores {
		names = append(names, name)
	}
	return names
}

// RemoveDocumentStore removes a document store
func (cm *ContextManager) RemoveDocumentStore(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	store, exists := cm.documentStores[name]
	if !exists {
		return NewContextError("ContextManager", "RemoveDocumentStore", "document store not found", nil)
	}

	// Close the store
	if err := store.Close(); err != nil {
		return NewContextError("ContextManager", "RemoveDocumentStore", "failed to close document store", err)
	}

	delete(cm.documentStores, name)
	cm.status.DocumentStores = len(cm.documentStores)
	cm.status.LastUpdated = time.Now()

	return nil
}

// ============================================================================
// CONVERSATION MANAGEMENT
// ============================================================================

// CreateConversation creates a new conversation
func (cm *ContextManager) CreateConversation(sessionID string) (*ConversationHistory, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.status.Initialized {
		return nil, NewContextError("ContextManager", "CreateConversation", "context manager not initialized", nil)
	}

	conversation, err := cm.factory.CreateConversationHistory(sessionID)
	if err != nil {
		return nil, NewContextError("ContextManager", "CreateConversation", "failed to create conversation", err)
	}

	cm.conversations[sessionID] = conversation
	cm.status.Conversations = len(cm.conversations)
	cm.status.LastUpdated = time.Now()

	return conversation, nil
}

// GetConversation returns a conversation by session ID
func (cm *ContextManager) GetConversation(sessionID string) (*ConversationHistory, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conversation, exists := cm.conversations[sessionID]
	return conversation, exists
}

// ListConversations returns all conversation session IDs
func (cm *ContextManager) ListConversations() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var sessionIDs []string
	for sessionID := range cm.conversations {
		sessionIDs = append(sessionIDs, sessionID)
	}
	return sessionIDs
}

// RemoveConversation removes a conversation
func (cm *ContextManager) RemoveConversation(sessionID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	_, exists := cm.conversations[sessionID]
	if !exists {
		return NewContextError("ContextManager", "RemoveConversation", "conversation not found", nil)
	}

	delete(cm.conversations, sessionID)
	cm.status.Conversations = len(cm.conversations)
	cm.status.LastUpdated = time.Now()

	return nil
}

// ============================================================================
// STATUS AND HEALTH MANAGEMENT
// ============================================================================

// IsInitialized returns whether the context manager is initialized
func (cm *ContextManager) IsInitialized() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.status.Initialized
}

// GetStatus returns detailed status information
func (cm *ContextManager) GetStatus() *ContextManagerStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy to prevent external modification
	statusCopy := *cm.status
	return &statusCopy
}

// IsHealthy returns the health status of the context manager
func (cm *ContextManager) IsHealthy() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.status.Healthy
}

// SetHealthy sets the health status
func (cm *ContextManager) SetHealthy(healthy bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.status.Healthy = healthy
	cm.status.LastUpdated = time.Now()
}

// ============================================================================
// OPERATION CONTROL AND SEMAPHORE
// ============================================================================

// acquireOperation acquires an operation semaphore
func (cm *ContextManager) acquireOperation() {
	cm.operationSemaphore <- struct{}{}
}

// releaseOperation releases an operation semaphore
func (cm *ContextManager) releaseOperation() {
	<-cm.operationSemaphore
}

// ============================================================================
// CONTEXT FACTORY - SIMPLIFIED AND ENHANCED
// ============================================================================

// ContextFactory provides unified creation of context components
type ContextFactory struct {
	llmProvider      providers.LLMProvider
	embedderProvider providers.EmbedderProvider
	databaseProvider providers.DatabaseProvider
}

// NewContextFactory creates a new context factory
func NewContextFactory(
	llmProvider providers.LLMProvider,
	embedderProvider providers.EmbedderProvider,
	databaseProvider providers.DatabaseProvider,
) *ContextFactory {
	return &ContextFactory{
		llmProvider:      llmProvider,
		embedderProvider: embedderProvider,
		databaseProvider: databaseProvider,
	}
}

// CreateSearchEngine creates a search engine with the factory's providers
func (cf *ContextFactory) CreateSearchEngine(searchConfig hectorconfig.SearchConfig) (*SearchEngine, error) {
	if cf.embedderProvider == nil {
		return nil, NewContextError("ContextFactory", "CreateSearchEngine", "embedder provider is required", nil)
	}
	if cf.databaseProvider == nil {
		return nil, NewContextError("ContextFactory", "CreateSearchEngine", "database provider is required", nil)
	}

	return NewSearchEngine(cf.databaseProvider, cf.embedderProvider, searchConfig)
}

// CreateDocumentStore creates a document store with the factory's search engine
func (cf *ContextFactory) CreateDocumentStore(storeConfig *hectorconfig.DocumentStoreConfig) (*DocumentStore, error) {
	if storeConfig == nil {
		return nil, NewContextError("ContextFactory", "CreateDocumentStore", "store config is required", nil)
	}

	// Create search engine for the document store
	searchEngine, err := cf.CreateSearchEngine(hectorconfig.SearchConfig{})
	if err != nil {
		return nil, NewContextError("ContextFactory", "CreateDocumentStore", "failed to create search engine", err)
	}

	return NewDocumentStore(storeConfig, searchEngine)
}

// CreateConversationHistory creates a new conversation history
func (cf *ContextFactory) CreateConversationHistory(sessionID string) (*ConversationHistory, error) {
	return NewConversationHistory(sessionID)
}

// ============================================================================
// CONTEXT UTILITIES - ENHANCED
// ============================================================================

// ValidateContext validates a context for common issues
func ValidateContext(ctx context.Context) error {
	if ctx == nil {
		return NewContextError("ContextUtils", "ValidateContext", "context cannot be nil", nil)
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return NewContextError("ContextUtils", "ValidateContext", "context is already cancelled", ctx.Err())
	default:
		return nil
	}
}

// CreateTimeoutContext creates a context with timeout
func CreateTimeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// CreateIndexingContext creates a context suitable for indexing operations
func CreateIndexingContext() (context.Context, context.CancelFunc) {
	return CreateTimeoutContext(DefaultIndexingTimeout)
}

// CreateSearchContext creates a context suitable for search operations
func CreateSearchContext() (context.Context, context.CancelFunc) {
	return CreateTimeoutContext(DefaultContextTimeout)
}

// CreateHealthCheckContext creates a context suitable for health check operations
func CreateHealthCheckContext() (context.Context, context.CancelFunc) {
	return CreateTimeoutContext(DefaultHealthCheckTimeout)
}

// ============================================================================
// CLEANUP AND RESOURCE MANAGEMENT
// ============================================================================

// Close closes the context manager and cleans up resources
func (cm *ContextManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Close all document stores
	for name, store := range cm.documentStores {
		if err := store.Close(); err != nil {
			// Log error but continue with other stores
			fmt.Printf("Warning: Failed to close document store %s: %v\n", name, err)
		}
	}

	// Clear all maps
	cm.documentStores = make(map[string]*DocumentStore)
	cm.conversations = make(map[string]*ConversationHistory)

	// Mark as unhealthy and not initialized
	cm.status.Healthy = false
	cm.status.Initialized = false
	cm.status.LastUpdated = time.Now()

	return nil
}
