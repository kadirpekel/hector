package component

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/plugins"
	plugingrpc "github.com/kadirpekel/hector/pkg/plugins/grpc"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/tools"
	yaml "gopkg.in/yaml.v3"
)

type ComponentManager struct {
	globalConfig *config.Config

	llmRegistry      *llms.LLMRegistry
	dbRegistry       *databases.DatabaseRegistry
	embedderRegistry *embedders.EmbedderRegistry
	toolRegistry     *tools.ToolRegistry

	pluginRegistry *plugins.PluginRegistry

	sessionStoreDBs map[string]interface{}
}

func NewComponentManager(globalConfig *config.Config) (*ComponentManager, error) {
	return NewComponentManagerWithAgentRegistry(globalConfig, nil)
}

func NewComponentManagerWithAgentRegistry(globalConfig *config.Config, agentRegistry interface{}) (*ComponentManager, error) {
	ctx := context.Background()

	toolRegistry, err := tools.NewToolRegistryWithConfigAndAgentRegistry(globalConfig.Tools, agentRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool registry: %w", err)
	}

	pluginRegistry := plugins.NewPluginRegistry(nil)

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
		sessionStoreDBs:  make(map[string]interface{}),
	}

	if err := cm.loadPlugins(ctx); err != nil {
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	for name, llmConfig := range cm.globalConfig.LLMs {
		_, err := cm.llmRegistry.CreateLLMFromConfig(name, llmConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize LLM '%s': %w", name, err)
		}
	}

	usedEmbedders := make(map[string]bool)

	// Collect vector stores/embedders from agents
	usedVectorStores := make(map[string]bool)
	for _, agentConfig := range cm.globalConfig.Agents {
		if agentConfig.VectorStore != "" {
			usedVectorStores[agentConfig.VectorStore] = true
		}
		if agentConfig.Embedder != "" {
			usedEmbedders[agentConfig.Embedder] = true
		}
	}

	// Also collect vector stores/embedders from document stores
	for _, storeConfig := range cm.globalConfig.DocumentStores {
		if storeConfig.VectorStore != "" {
			usedVectorStores[storeConfig.VectorStore] = true
		}
		if storeConfig.Embedder != "" {
			usedEmbedders[storeConfig.Embedder] = true
		}
	}

	// Initialize vector stores
	for name, vsConfig := range cm.globalConfig.VectorStores {
		if usedVectorStores[name] {
			_, err := cm.dbRegistry.CreateDatabaseFromConfig(name, vsConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize vector store '%s': %w", name, err)
			}
		}
	}

	for name, embedderConfig := range cm.globalConfig.Embedders {
		if usedEmbedders[name] {
			_, err := cm.embedderRegistry.CreateEmbedderFromConfig(name, embedderConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize embedder '%s': %w", name, err)
			}
		}
	}

	return cm, nil
}

func (cm *ComponentManager) GetGlobalConfig() *config.Config {
	return cm.globalConfig
}

func (cm *ComponentManager) GetLLMRegistry() *llms.LLMRegistry {
	return cm.llmRegistry
}

func (cm *ComponentManager) GetDatabaseRegistry() *databases.DatabaseRegistry {
	return cm.dbRegistry
}

func (cm *ComponentManager) GetEmbedderRegistry() *embedders.EmbedderRegistry {
	return cm.embedderRegistry
}

func (cm *ComponentManager) GetToolRegistry() *tools.ToolRegistry {
	return cm.toolRegistry
}

func (cm *ComponentManager) GetPluginRegistry() *plugins.PluginRegistry {
	return cm.pluginRegistry
}

func (cm *ComponentManager) GetLLM(name string) (llms.LLMProvider, error) {
	return cm.llmRegistry.GetLLM(name)
}

func (cm *ComponentManager) GetDatabase(name string) (databases.DatabaseProvider, error) {
	return cm.dbRegistry.GetDatabase(name)
}

// GetSQLDatabase returns a SQL database connection and driver name
func (cm *ComponentManager) GetSQLDatabase(name string) (*sql.DB, string, error) {
	dbConfig, exists := cm.globalConfig.Databases[name]
	if !exists {
		return nil, "", fmt.Errorf("SQL database '%s' not found", name)
	}

	db, err := cm.getOrCreateSQLDatabase(name, dbConfig)
	if err != nil {
		return nil, "", err
	}

	return db, dbConfig.Driver, nil
}

func (cm *ComponentManager) GetEmbedder(name string) (embedders.EmbedderProvider, error) {
	return cm.embedderRegistry.GetEmbedder(name)
}

func (cm *ComponentManager) GetSessionService(storeName string, agentID string) (reasoning.SessionService, error) {
	if storeName == "" {

		return memory.NewInMemorySessionService(), nil
	}

	if agentID == "" {
		return nil, fmt.Errorf("agent ID is required for session service")
	}

	storeConfig, ok := cm.globalConfig.SessionStores[storeName]
	if !ok {
		return nil, fmt.Errorf("session store '%s' not found in configuration", storeName)
	}

	if storeConfig.Backend == "" || storeConfig.Backend == "memory" {
		return memory.NewInMemorySessionService(), nil
	}

	if storeConfig.Backend == "sql" {
		var db *sql.DB
		var driver string
		var err error

		if storeConfig.SQLDatabase == "" {
			return nil, fmt.Errorf("session store '%s': sql_database reference is required for SQL backend", storeName)
		}
		dbConfig, exists := cm.globalConfig.Databases[storeConfig.SQLDatabase]
		if !exists {
			return nil, fmt.Errorf("session store '%s': sql_database '%s' not found", storeName, storeConfig.SQLDatabase)
		}
		db, err = cm.getOrCreateSQLDatabase(storeConfig.SQLDatabase, dbConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get SQL database '%s': %w", storeConfig.SQLDatabase, err)
		}
		driver = dbConfig.Driver

		return memory.NewSQLSessionService(db, driver, agentID)
	}

	return nil, fmt.Errorf("unsupported session store backend '%s' for store '%s'", storeConfig.Backend, storeName)
}

// getOrCreateSQLDatabase creates or retrieves a cached SQL database connection
func (cm *ComponentManager) getOrCreateSQLDatabase(dbName string, cfg *config.DatabaseConfig) (*sql.DB, error) {
	// Check cache first
	if cached, ok := cm.sessionStoreDBs[dbName]; ok {
		if db, ok := cached.(*sql.DB); ok {
			return db, nil
		}
	}

	driverName := cfg.Driver
	if driverName == "sqlite" {
		driverName = "sqlite3"
	}

	db, err := sql.Open(driverName, cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxIdle)

	// Parse connection lifetime
	if cfg.ConnMaxLifetime != "" {
		if lifetime, err := time.ParseDuration(cfg.ConnMaxLifetime); err == nil {
			db.SetConnMaxLifetime(lifetime)
		} else {
			db.SetConnMaxLifetime(time.Hour) // Default
		}
	} else {
		db.SetConnMaxLifetime(time.Hour)
	}

	// Parse idle timeout
	if cfg.ConnMaxIdleTime != "" {
		if idleTime, err := time.ParseDuration(cfg.ConnMaxIdleTime); err == nil {
			db.SetConnMaxIdleTime(idleTime)
		} else {
			db.SetConnMaxIdleTime(30 * time.Minute) // Default
		}
	} else {
		db.SetConnMaxIdleTime(30 * time.Minute)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	cm.sessionStoreDBs[dbName] = db

	fmt.Printf("✅ SQL database '%s' connected (driver: %s, database: %s)\n", dbName, cfg.Driver, cfg.Database)

	return db, nil
}

func (cm *ComponentManager) loadPlugins(ctx context.Context) error {
	pluginConfig := &cm.globalConfig.Plugins

	discoveryConfig := &plugins.DiscoveryConfig{
		Enabled:            config.BoolValue(pluginConfig.Discovery.Enabled, false),
		Paths:              pluginConfig.Discovery.Paths,
		ScanSubdirectories: config.BoolValue(pluginConfig.Discovery.ScanSubdirectories, false),
	}

	discovery := plugins.NewPluginDiscovery(discoveryConfig)
	discoveredPlugins, err := discovery.DiscoverPlugins(ctx)
	if err != nil {
		return fmt.Errorf("plugin discovery failed: %w", err)
	}

	if err := cm.loadConfiguredPlugins(ctx, pluginConfig); err != nil {
		return fmt.Errorf("failed to load configured plugins: %w", err)
	}

	if err := cm.loadDiscoveredPlugins(ctx, discoveredPlugins, pluginConfig); err != nil {
		return fmt.Errorf("failed to load discovered plugins: %w", err)
	}

	return nil
}

func (cm *ComponentManager) loadConfiguredPlugins(ctx context.Context, pluginConfig *config.PluginConfigs) error {

	for name, cfg := range pluginConfig.LLMProviders {
		if cfg != nil && cfg.Enabled != nil && !*cfg.Enabled {
			continue
		}
		if cfg != nil {
			if err := cm.loadAndRegisterPlugin(ctx, name, cfg, plugins.PluginTypeLLM); err != nil {
				fmt.Printf("Warning: Failed to load LLM plugin '%s': %v\n", name, err)
			}
		}
	}

	for name, cfg := range pluginConfig.DatabaseProviders {
		if cfg != nil && cfg.Enabled != nil && !*cfg.Enabled {
			continue
		}
		if cfg != nil {
			if err := cm.loadAndRegisterPlugin(ctx, name, cfg, plugins.PluginTypeDatabase); err != nil {
				fmt.Printf("Warning: Failed to load Database plugin '%s': %v\n", name, err)
			}
		}
	}

	for name, cfg := range pluginConfig.EmbedderProviders {
		if cfg != nil && cfg.Enabled != nil && !*cfg.Enabled {
			continue
		}
		if cfg != nil {
			if err := cm.loadAndRegisterPlugin(ctx, name, cfg, plugins.PluginTypeEmbedder); err != nil {
				fmt.Printf("Warning: Failed to load Embedder plugin '%s': %v\n", name, err)
			}
		}
	}

	return nil
}

func (cm *ComponentManager) loadDiscoveredPlugins(ctx context.Context, discovered []*plugins.DiscoveredPlugin, pluginConfig *config.PluginConfigs) error {

	for _, dp := range discovered {

		if cm.isPluginConfigured(dp.Name, pluginConfig) {
			continue
		}

		cfg := &config.PluginConfig{
			Name:    dp.Name,
			Type:    string(dp.Manifest.Protocol),
			Path:    dp.Path,
			Enabled: config.BoolPtr(true),
			Config:  make(map[string]interface{}),
		}

		if err := cm.loadAndRegisterPlugin(ctx, dp.Name, cfg, dp.Manifest.Type); err != nil {
			fmt.Printf("Warning: Failed to load discovered plugin '%s': %v\n", dp.Name, err)
		}
	}

	return nil
}

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

func (cm *ComponentManager) loadAndRegisterPlugin(ctx context.Context, name string, cfg *config.PluginConfig, pluginType plugins.PluginType) error {

	manifestPath := cfg.Path + ".plugin.yaml"
	manifest, err := loadPluginManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest from %s: %w", manifestPath, err)
	}

	pluginCfg := &plugins.PluginConfig{
		Name:     name,
		Type:     plugins.PluginProtocol(cfg.Type),
		Path:     cfg.Path,
		Enabled:  config.BoolValue(cfg.Enabled, true),
		Config:   cfg.Config,
		Manifest: manifest,
	}

	if pluginErr := cm.pluginRegistry.LoadPlugin(ctx, pluginCfg); pluginErr != nil {
		return err
	}

	plugin, err := cm.pluginRegistry.GetPlugin(name)
	if err != nil {
		return err
	}

	switch pluginType {
	case plugins.PluginTypeLLM:
		llmAdapter, ok := plugin.(*plugingrpc.LLMPluginAdapter)
		if !ok {
			return fmt.Errorf("plugin is not an LLM provider")
		}

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

		embedderBridge := &embedderPluginBridge{adapter: embedderAdapter}
		if err := cm.embedderRegistry.RegisterEmbedder(name, embedderBridge); err != nil {
			return fmt.Errorf("failed to register Embedder plugin: %w", err)
		}
		fmt.Printf("✓ Registered Embedder plugin: %s\n", name)
	}

	return nil
}

func (cm *ComponentManager) ShutdownPlugins(ctx context.Context) error {
	if cm.pluginRegistry != nil {
		return cm.pluginRegistry.Shutdown(ctx)
	}
	return nil
}

func (cm *ComponentManager) Close() error {
	var errors []error

	for storeName, dbInterface := range cm.sessionStoreDBs {
		if db, ok := dbInterface.(*sql.DB); ok {
			if err := db.Close(); err != nil {
				errors = append(errors, fmt.Errorf("failed to close session store DB '%s': %w", storeName, err))
			}
		}
	}

	ctx := context.Background()
	if err := cm.ShutdownPlugins(ctx); err != nil {
		errors = append(errors, fmt.Errorf("plugin shutdown: %w", err))
	}

	if len(errors) > 0 {
		return errors[0]
	}
	return nil
}

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

type llmPluginBridge struct {
	adapter *plugingrpc.LLMPluginAdapter
}

func (b *llmPluginBridge) Generate(ctx context.Context, messages []*pb.Message, tools []llms.ToolDefinition) (text string, toolCalls []*protocol.ToolCall, tokens int, err error) {

	pbMessages := make([]*plugingrpc.Message, len(messages))
	for i, msg := range messages {
		textContent := protocol.ExtractTextFromMessage(msg)
		pbMessages[i] = &plugingrpc.Message{
			Role:    msg.Role.String(),
			Content: textContent,
		}
	}

	pbTools := make([]*plugingrpc.ToolDefinition, len(tools))
	for i, tool := range tools {

		paramsJSON, _ := json.Marshal(tool.Parameters)
		pbTools[i] = &plugingrpc.ToolDefinition{
			Name:           tool.Name,
			Description:    tool.Description,
			ParametersJson: string(paramsJSON),
		}
	}

	response, err := b.adapter.GetPlugin().Generate(ctx, pbMessages, pbTools)
	if err != nil {
		return "", nil, 0, err
	}

	llmToolCalls := make([]*protocol.ToolCall, len(response.ToolCalls))
	for i, tc := range response.ToolCalls {

		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.ArgumentsJson), &args); err != nil {
			args = make(map[string]interface{})
		}

		llmToolCalls[i] = &protocol.ToolCall{
			ID:   tc.Id,
			Name: tc.Name,
			Args: args,
		}
	}

	return response.Text, llmToolCalls, int(response.TokensUsed), nil
}

func (b *llmPluginBridge) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []llms.ToolDefinition) (<-chan llms.StreamChunk, error) {

	pbMessages := make([]*plugingrpc.Message, len(messages))
	for i, msg := range messages {
		textContent := protocol.ExtractTextFromMessage(msg)
		pbMessages[i] = &plugingrpc.Message{
			Role:    msg.Role.String(),
			Content: textContent,
		}
	}

	pbTools := make([]*plugingrpc.ToolDefinition, len(tools))
	for i, tool := range tools {

		paramsJSON, _ := json.Marshal(tool.Parameters)
		pbTools[i] = &plugingrpc.ToolDefinition{
			Name:           tool.Name,
			Description:    tool.Description,
			ParametersJson: string(paramsJSON),
		}
	}

	pbChunks, err := b.adapter.GetPlugin().GenerateStreaming(ctx, pbMessages, pbTools)
	if err != nil {
		return nil, err
	}

	llmChunks := make(chan llms.StreamChunk, 10)
	go func() {
		defer close(llmChunks)
		for pbChunk := range pbChunks {
			llmChunk := llms.StreamChunk{
				Text:   pbChunk.Text,
				Tokens: int(pbChunk.TokensUsed),
			}

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
	// Plugin ModelInfo.Temperature is float64, not *float64
	// Return the value as-is - temperature 0.0 is a valid value for deterministic output
	// If the plugin needs to indicate "not set", it should use a sentinel value or
	// the protobuf definition should be updated to support optional fields
	return info.Temperature
}

func (b *llmPluginBridge) Close() error {
	return b.adapter.Shutdown(context.Background())
}

type databasePluginBridge struct {
	adapter *plugingrpc.DatabasePluginAdapter
}

func (b *databasePluginBridge) Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error {

	stringMetadata := make(map[string]string)
	for k, v := range metadata {
		stringMetadata[k] = fmt.Sprintf("%v", v)
	}

	return b.adapter.GetPlugin().Upsert(ctx, collection, id, vector, stringMetadata)
}

func (b *databasePluginBridge) Search(ctx context.Context, collection string, vector []float32, topK int) ([]databases.SearchResult, error) {
	return b.SearchWithFilter(ctx, collection, vector, topK, nil)
}

func (b *databasePluginBridge) SearchWithFilter(ctx context.Context, collection string, vector []float32, topK int, filter map[string]interface{}) ([]databases.SearchResult, error) {

	pbResults, err := b.adapter.GetPlugin().Search(ctx, collection, vector, int32(topK*2))
	if err != nil {
		return nil, err
	}

	results := make([]databases.SearchResult, 0, len(pbResults))
	for _, pbResult := range pbResults {
		metadata := make(map[string]interface{})
		for k, v := range pbResult.Metadata {
			metadata[k] = v
		}

		result := databases.SearchResult{
			ID:       pbResult.Id,
			Score:    pbResult.Score,
			Content:  pbResult.Content,
			Metadata: metadata,
		}

		if len(filter) > 0 {
			match := true
			for filterKey, filterValue := range filter {
				if metadataValue, ok := result.Metadata[filterKey]; !ok || fmt.Sprintf("%v", metadataValue) != fmt.Sprintf("%v", filterValue) {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		results = append(results, result)

		if len(results) >= topK {
			break
		}
	}

	return results, nil
}

func (b *databasePluginBridge) Delete(ctx context.Context, collection string, id string) error {
	return b.adapter.GetPlugin().Delete(ctx, collection, id)
}

func (b *databasePluginBridge) DeleteByFilter(ctx context.Context, collection string, filter map[string]interface{}) error {

	return fmt.Errorf("DeleteByFilter not supported for database plugins (collection: %s)", collection)
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
