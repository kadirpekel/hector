package component

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/plugins"
	plugingrpc "github.com/kadirpekel/hector/pkg/plugins/grpc"
	"github.com/kadirpekel/hector/pkg/tools"
	"gopkg.in/yaml.v3"
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
	// Load manifest from .plugin.yaml file
	manifestPath := cfg.Path + ".plugin.yaml"
	manifest, err := loadPluginManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest from %s: %w", manifestPath, err)
	}

	// Convert config.PluginConfig to plugins.PluginConfig
	pluginCfg := &plugins.PluginConfig{
		Name:     name,
		Type:     plugins.PluginProtocol(cfg.Type),
		Path:     cfg.Path,
		Enabled:  cfg.Enabled,
		Config:   cfg.Config,
		Manifest: manifest,
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
		llmAdapter, ok := plugin.(*plugingrpc.LLMPluginAdapter)
		if !ok {
			return fmt.Errorf("plugin is not an LLM provider")
		}

		// Create bridge and register
		llmBridge := &llmPluginBridge{adapter: llmAdapter}
		if err := cm.llmRegistry.RegisterLLM(name, llmBridge); err != nil {
			return fmt.Errorf("failed to register LLM plugin: %w", err)
		}
		fmt.Printf("✓ Registered LLM plugin: %s\n", name)

	case plugins.PluginTypeDatabase:
		dbAdapter, ok := plugin.(*plugingrpc.DatabasePluginAdapter)
		if !ok {
			return fmt.Errorf("plugin is not a Database provider")
		}

		// Create bridge and register
		dbBridge := &databasePluginBridge{adapter: dbAdapter}
		if err := cm.dbRegistry.RegisterDatabase(name, dbBridge); err != nil {
			return fmt.Errorf("failed to register Database plugin: %w", err)
		}
		fmt.Printf("✓ Registered Database plugin: %s\n", name)

	case plugins.PluginTypeEmbedder:
		embedderAdapter, ok := plugin.(*plugingrpc.EmbedderPluginAdapter)
		if !ok {
			return fmt.Errorf("plugin is not an Embedder provider")
		}

		// Create bridge and register
		embedderBridge := &embedderPluginBridge{adapter: embedderAdapter}
		if err := cm.embedderRegistry.RegisterEmbedder(name, embedderBridge); err != nil {
			return fmt.Errorf("failed to register Embedder plugin: %w", err)
		}
		fmt.Printf("✓ Registered Embedder plugin: %s\n", name)
	}

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
// PLUGIN HELPER FUNCTIONS
// ============================================================================

// loadPluginManifest loads a plugin manifest from a .plugin.yaml file
func loadPluginManifest(manifestPath string) (*plugins.PluginManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifestWrapper struct {
		Plugin plugins.PluginManifest `yaml:"plugin"`
	}
	if err := yaml.Unmarshal(data, &manifestWrapper); err != nil {
		return nil, fmt.Errorf("failed to parse manifest YAML: %w", err)
	}

	return &manifestWrapper.Plugin, nil
}

// ============================================================================
// PLUGIN BRIDGE IMPLEMENTATIONS
// ============================================================================
// These bridge types adapt plugin adapters to component registry interfaces
// with zero overhead - just simple method forwarding and type conversion.

// llmPluginBridge adapts LLMPluginAdapter to llms.LLMProvider interface
type llmPluginBridge struct {
	adapter *plugingrpc.LLMPluginAdapter
}

func (b *llmPluginBridge) Generate(messages []llms.Message, tools []llms.ToolDefinition) (text string, toolCalls []llms.ToolCall, tokens int, err error) {
	// Convert llms.Message to pb.Message
	pbMessages := make([]*plugingrpc.Message, len(messages))
	for i, msg := range messages {
		pbMessages[i] = &plugingrpc.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Convert llms.ToolDefinition to pb.ToolDefinition
	pbTools := make([]*plugingrpc.ToolDefinition, len(tools))
	for i, tool := range tools {
		// Serialize parameters to JSON
		paramsJSON, _ := json.Marshal(tool.Parameters)
		pbTools[i] = &plugingrpc.ToolDefinition{
			Name:           tool.Name,
			Description:    tool.Description,
			ParametersJson: string(paramsJSON),
		}
	}

	// Call plugin
	response, err := b.adapter.GetPlugin().Generate(context.Background(), pbMessages, pbTools)
	if err != nil {
		return "", nil, 0, err
	}

	// Convert pb.ToolCall back to llms.ToolCall
	llmToolCalls := make([]llms.ToolCall, len(response.ToolCalls))
	for i, tc := range response.ToolCalls {
		// Deserialize arguments from JSON
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.ArgumentsJson), &args); err != nil {
			args = make(map[string]interface{})
		}

		llmToolCalls[i] = llms.ToolCall{
			ID:        tc.Id,
			Name:      tc.Name,
			Arguments: args,
			RawArgs:   tc.ArgumentsJson,
		}
	}

	return response.Text, llmToolCalls, int(response.TokensUsed), nil
}

func (b *llmPluginBridge) GenerateStreaming(messages []llms.Message, tools []llms.ToolDefinition) (<-chan llms.StreamChunk, error) {
	// Convert messages and tools
	pbMessages := make([]*plugingrpc.Message, len(messages))
	for i, msg := range messages {
		pbMessages[i] = &plugingrpc.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	pbTools := make([]*plugingrpc.ToolDefinition, len(tools))
	for i, tool := range tools {
		// Serialize parameters to JSON
		paramsJSON, _ := json.Marshal(tool.Parameters)
		pbTools[i] = &plugingrpc.ToolDefinition{
			Name:           tool.Name,
			Description:    tool.Description,
			ParametersJson: string(paramsJSON),
		}
	}

	// Get plugin streaming channel
	pbChunks, err := b.adapter.GetPlugin().GenerateStreaming(context.Background(), pbMessages, pbTools)
	if err != nil {
		return nil, err
	}

	// Create output channel and convert chunks
	llmChunks := make(chan llms.StreamChunk, 10)
	go func() {
		defer close(llmChunks)
		for pbChunk := range pbChunks {
			llmChunk := llms.StreamChunk{
				Text:   pbChunk.Text,
				Tokens: int(pbChunk.TokensUsed),
			}

			// Convert chunk type
			switch pbChunk.Type {
			case plugingrpc.ChunkTypeText:
				llmChunk.Type = "text"
			case plugingrpc.ChunkTypeDone:
				llmChunk.Type = "done"
			case plugingrpc.ChunkTypeError:
				llmChunk.Type = "error"
				llmChunk.Error = fmt.Errorf("%s", pbChunk.Error)
			}

			llmChunks <- llmChunk
		}
	}()

	return llmChunks, nil
}

func (b *llmPluginBridge) GetModelName() string {
	info, err := b.adapter.GetPlugin().GetModelInfo(context.Background())
	if err != nil {
		return "unknown"
	}
	return info.ModelName
}

func (b *llmPluginBridge) GetMaxTokens() int {
	info, err := b.adapter.GetPlugin().GetModelInfo(context.Background())
	if err != nil {
		return 0
	}
	return int(info.MaxTokens)
}

func (b *llmPluginBridge) GetTemperature() float64 {
	info, err := b.adapter.GetPlugin().GetModelInfo(context.Background())
	if err != nil {
		return 0.0
	}
	return info.Temperature
}

func (b *llmPluginBridge) Close() error {
	return b.adapter.Shutdown(context.Background())
}

// databasePluginBridge adapts DatabasePluginAdapter to databases.DatabaseProvider interface
type databasePluginBridge struct {
	adapter *plugingrpc.DatabasePluginAdapter
}

func (b *databasePluginBridge) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {
	// Convert metadata from interface{} to string
	stringMetadata := make(map[string]string)
	for k, v := range metadata {
		stringMetadata[k] = fmt.Sprintf("%v", v)
	}

	return b.adapter.GetPlugin().Upsert(ctx, collection, id, vector, stringMetadata)
}

func (b *databasePluginBridge) Search(ctx context.Context, collection string, vector []float32, topK int) ([]databases.SearchResult, error) {
	pbResults, err := b.adapter.GetPlugin().Search(ctx, collection, vector, int32(topK))
	if err != nil {
		return nil, err
	}

	// Convert pb.SearchResult to databases.SearchResult
	results := make([]databases.SearchResult, len(pbResults))
	for i, pbResult := range pbResults {
		metadata := make(map[string]interface{})
		for k, v := range pbResult.Metadata {
			metadata[k] = v
		}

		results[i] = databases.SearchResult{
			ID:       pbResult.Id,
			Score:    pbResult.Score,
			Content:  pbResult.Content,
			Metadata: metadata,
		}
	}

	return results, nil
}

func (b *databasePluginBridge) Delete(ctx context.Context, collection string, id string) error {
	return b.adapter.GetPlugin().Delete(ctx, collection, id)
}

func (b *databasePluginBridge) CreateCollection(ctx context.Context, collection string, vectorSize uint64) error {
	return b.adapter.GetPlugin().CreateCollection(ctx, collection, vectorSize)
}

func (b *databasePluginBridge) DeleteCollection(ctx context.Context, collection string) error {
	return b.adapter.GetPlugin().DeleteCollection(ctx, collection)
}

func (b *databasePluginBridge) Close() error {
	return b.adapter.Shutdown(context.Background())
}

// embedderPluginBridge adapts EmbedderPluginAdapter to embedders.EmbedderProvider interface
type embedderPluginBridge struct {
	adapter *plugingrpc.EmbedderPluginAdapter
}

func (b *embedderPluginBridge) Embed(text string) ([]float32, error) {
	return b.adapter.GetPlugin().Embed(context.Background(), text)
}

func (b *embedderPluginBridge) GetDimension() int {
	info, err := b.adapter.GetPlugin().GetEmbedderInfo(context.Background())
	if err != nil {
		return 0
	}
	return int(info.Dimension)
}

func (b *embedderPluginBridge) GetModelName() string {
	info, err := b.adapter.GetPlugin().GetEmbedderInfo(context.Background())
	if err != nil {
		return "unknown"
	}
	return info.ModelName
}

func (b *embedderPluginBridge) Close() error {
	return b.adapter.Shutdown(context.Background())
}

// ============================================================================
// AGENT COMPONENT CREATION
// ============================================================================
