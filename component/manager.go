package component

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/databases"
	"github.com/kadirpekel/hector/embedders"
	"github.com/kadirpekel/hector/llms"
	"github.com/kadirpekel/hector/plugins"
	plugingrpc "github.com/kadirpekel/hector/plugins/grpc"
	"github.com/kadirpekel/hector/tools"
)

// ============================================================================
// COMPONENT MANAGER
// ============================================================================

// ComponentManager manages all component registries and global configuration
type ComponentManager struct {
	// Global configuration
	globalConfig *config.Config

	// Component registries
	llmRegistry      *llms.LLMRegistry
	dbRegistry       *databases.DatabaseRegistry
	embedderRegistry *embedders.EmbedderRegistry
	toolRegistry     *tools.ToolRegistry

	// Plugin registry
	pluginRegistry *plugins.PluginRegistry
}

// NewComponentManager creates a new component manager and initializes all components
func NewComponentManager(globalConfig *config.Config) (*ComponentManager, error) {
	ctx := context.Background()

	// Initialize tool registry with configuration
	toolRegistry, err := tools.NewToolRegistryWithConfig(&globalConfig.Tools)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool registry: %w", err)
	}

	// Initialize plugin registry
	pluginRegistry := plugins.NewPluginRegistry(nil)

	// Register gRPC plugin loader
	grpcLoader := plugingrpc.NewGRPCLoader()
	if err := pluginRegistry.RegisterLoader(grpcLoader); err != nil {
		return nil, fmt.Errorf("failed to register gRPC loader: %w", err)
	}

	cm := &ComponentManager{
		globalConfig:     globalConfig,
		llmRegistry:      llms.NewLLMRegistry(),
		dbRegistry:       databases.NewDatabaseRegistry(),
		embedderRegistry: embedders.NewEmbedderRegistry(),
		toolRegistry:     toolRegistry,
		pluginRegistry:   pluginRegistry,
	}

	// Discover and load plugins
	if err := cm.loadPlugins(ctx); err != nil {
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	// Initialize LLM providers
	for name, llmConfig := range cm.globalConfig.LLMs {
		_, err := cm.llmRegistry.CreateLLMFromConfig(name, &llmConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize LLM '%s': %w", name, err)
		}
	}

	// Initialize only databases that are actually used by agents
	usedDatabases := make(map[string]bool)
	usedEmbedders := make(map[string]bool)

	// Collect used services from all agents
	for _, agentConfig := range cm.globalConfig.Agents {
		if agentConfig.Database != "" {
			usedDatabases[agentConfig.Database] = true
		}
		if agentConfig.Embedder != "" {
			usedEmbedders[agentConfig.Embedder] = true
		}
	}

	// Initialize only used Database providers
	for name, dbConfig := range cm.globalConfig.Databases {
		if usedDatabases[name] {
			_, err := cm.dbRegistry.CreateDatabaseFromConfig(name, &dbConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize database '%s': %w", name, err)
			}
		}
	}

	// Initialize only used Embedder providers
	for name, embedderConfig := range cm.globalConfig.Embedders {
		if usedEmbedders[name] {
			_, err := cm.embedderRegistry.CreateEmbedderFromConfig(name, &embedderConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize embedder '%s': %w", name, err)
			}
		}
	}

	// Tool registry is already initialized with configuration in constructor

	// Document stores must be explicitly configured by user
	// No automatic initialization of document stores or search engines

	return cm, nil
}

// ============================================================================
// GETTERS
// ============================================================================

// GetGlobalConfig returns the global configuration
func (cm *ComponentManager) GetGlobalConfig() *config.Config {
	return cm.globalConfig
}

// GetLLMRegistry returns the LLM registry
func (cm *ComponentManager) GetLLMRegistry() *llms.LLMRegistry {
	return cm.llmRegistry
}

// GetDatabaseRegistry returns the database registry
func (cm *ComponentManager) GetDatabaseRegistry() *databases.DatabaseRegistry {
	return cm.dbRegistry
}

// GetEmbedderRegistry returns the embedder registry
func (cm *ComponentManager) GetEmbedderRegistry() *embedders.EmbedderRegistry {
	return cm.embedderRegistry
}

// GetToolRegistry returns the tool registry
func (cm *ComponentManager) GetToolRegistry() *tools.ToolRegistry {
	return cm.toolRegistry
}

// GetPluginRegistry returns the plugin registry
func (cm *ComponentManager) GetPluginRegistry() *plugins.PluginRegistry {
	return cm.pluginRegistry
}

// ============================================================================
// COMPONENT CREATION HELPERS
// ============================================================================

// GetLLM returns an LLM provider by name
func (cm *ComponentManager) GetLLM(name string) (llms.LLMProvider, error) {
	return cm.llmRegistry.GetLLM(name)
}

// GetDatabase returns a database provider by name
func (cm *ComponentManager) GetDatabase(name string) (databases.DatabaseProvider, error) {
	return cm.dbRegistry.GetDatabase(name)
}

// GetEmbedder returns an embedder provider by name
func (cm *ComponentManager) GetEmbedder(name string) (embedders.EmbedderProvider, error) {
	return cm.embedderRegistry.GetEmbedder(name)
}

// ============================================================================
// PLUGIN MANAGEMENT
// ============================================================================

// loadPlugins discovers and loads plugins from configuration
func (cm *ComponentManager) loadPlugins(ctx context.Context) error {
	pluginConfig := &cm.globalConfig.Plugins

	// Convert config.PluginDiscoveryConfig to plugins.DiscoveryConfig
	discoveryConfig := &plugins.DiscoveryConfig{
		Enabled:            pluginConfig.Discovery.Enabled,
		Paths:              pluginConfig.Discovery.Paths,
		ScanSubdirectories: pluginConfig.Discovery.ScanSubdirectories,
	}

	// Discover plugins from configured paths
	discovery := plugins.NewPluginDiscovery(discoveryConfig)
	discoveredPlugins, err := discovery.DiscoverPlugins(ctx)
	if err != nil {
		return fmt.Errorf("plugin discovery failed: %w", err)
	}

	// Load plugins from explicit configuration
	if err := cm.loadConfiguredPlugins(ctx, pluginConfig); err != nil {
		return fmt.Errorf("failed to load configured plugins: %w", err)
	}

	// Load auto-discovered plugins that are enabled
	if err := cm.loadDiscoveredPlugins(ctx, discoveredPlugins, pluginConfig); err != nil {
		return fmt.Errorf("failed to load discovered plugins: %w", err)
	}

	return nil
}

// loadConfiguredPlugins loads plugins from explicit configuration
func (cm *ComponentManager) loadConfiguredPlugins(ctx context.Context, pluginConfig *config.PluginConfigs) error {
	// Load LLM provider plugins
	for name, cfg := range pluginConfig.LLMProviders {
		if !cfg.Enabled {
			continue
		}
		if err := cm.loadAndRegisterPlugin(ctx, name, &cfg, plugins.PluginTypeLLM); err != nil {
			fmt.Printf("Warning: Failed to load LLM plugin '%s': %v\n", name, err)
		}
	}

	// Load Database provider plugins
	for name, cfg := range pluginConfig.DatabaseProviders {
		if !cfg.Enabled {
			continue
		}
		if err := cm.loadAndRegisterPlugin(ctx, name, &cfg, plugins.PluginTypeDatabase); err != nil {
			fmt.Printf("Warning: Failed to load Database plugin '%s': %v\n", name, err)
		}
	}

	// Load Embedder provider plugins
	for name, cfg := range pluginConfig.EmbedderProviders {
		if !cfg.Enabled {
			continue
		}
		if err := cm.loadAndRegisterPlugin(ctx, name, &cfg, plugins.PluginTypeEmbedder); err != nil {
			fmt.Printf("Warning: Failed to load Embedder plugin '%s': %v\n", name, err)
		}
	}

	return nil
}

// loadDiscoveredPlugins loads auto-discovered plugins
func (cm *ComponentManager) loadDiscoveredPlugins(ctx context.Context, discovered []*plugins.DiscoveredPlugin, pluginConfig *config.PluginConfigs) error {
	// Only load discovered plugins that aren't already explicitly configured
	for _, dp := range discovered {
		// Check if already configured
		if cm.isPluginConfigured(dp.Name, pluginConfig) {
			continue // Skip, will be loaded from explicit config
		}

		// Create plugin config from discovery
		cfg := &config.PluginConfig{
			Name:    dp.Name,
			Type:    string(dp.Manifest.Protocol),
			Path:    dp.Path,
			Enabled: true,
			Config:  make(map[string]interface{}),
		}

		if err := cm.loadAndRegisterPlugin(ctx, dp.Name, cfg, dp.Manifest.Type); err != nil {
			fmt.Printf("Warning: Failed to load discovered plugin '%s': %v\n", dp.Name, err)
		}
	}

	return nil
}

// isPluginConfigured checks if a plugin is explicitly configured
func (cm *ComponentManager) isPluginConfigured(name string, pluginConfig *config.PluginConfigs) bool {
	if _, ok := pluginConfig.LLMProviders[name]; ok {
		return true
	}
	if _, ok := pluginConfig.DatabaseProviders[name]; ok {
		return true
	}
	if _, ok := pluginConfig.EmbedderProviders[name]; ok {
		return true
	}
	if _, ok := pluginConfig.ToolProviders[name]; ok {
		return true
	}
	if _, ok := pluginConfig.ReasoningStrategies[name]; ok {
		return true
	}
	return false
}

// loadAndRegisterPlugin loads a plugin and registers it with appropriate registry
func (cm *ComponentManager) loadAndRegisterPlugin(ctx context.Context, name string, cfg *config.PluginConfig, pluginType plugins.PluginType) error {
	// Convert config.PluginConfig to plugins.PluginConfig
	pluginCfg := &plugins.PluginConfig{
		Name:    name,
		Type:    plugins.PluginProtocol(cfg.Type),
		Path:    cfg.Path,
		Enabled: cfg.Enabled,
		Config:  cfg.Config,
	}

	// Load the plugin
	if err := cm.pluginRegistry.LoadPlugin(ctx, pluginCfg); err != nil {
		return err
	}

	// Get the loaded plugin
	plugin, err := cm.pluginRegistry.GetPlugin(name)
	if err != nil {
		return err
	}

	// Register with appropriate component registry based on type
	switch pluginType {
	case plugins.PluginTypeLLM:
		// TODO: Create LLM provider adapter and register with llmRegistry
		fmt.Printf("✓ Loaded LLM plugin: %s\n", name)

	case plugins.PluginTypeDatabase:
		// TODO: Create Database provider adapter and register with dbRegistry
		fmt.Printf("✓ Loaded Database plugin: %s\n", name)

	case plugins.PluginTypeEmbedder:
		// TODO: Create Embedder provider adapter and register with embedderRegistry
		fmt.Printf("✓ Loaded Embedder plugin: %s\n", name)
	}

	_ = plugin // Suppress unused variable warning for now

	return nil
}

// ShutdownPlugins gracefully shuts down all plugins
func (cm *ComponentManager) ShutdownPlugins(ctx context.Context) error {
	if cm.pluginRegistry != nil {
		return cm.pluginRegistry.Shutdown(ctx)
	}
	return nil
}

// ============================================================================
// AGENT COMPONENT CREATION
// ============================================================================
