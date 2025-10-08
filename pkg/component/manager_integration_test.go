//go:build integration

package component

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
)

// ============================================================================
// COMPONENT MANAGER TESTS - Critical System Initialization
// Tests focus on verifying registries are initialized, not full integration
// ============================================================================

func createMinimalConfig() *config.Config {
	return &config.Config{
		LLMs:      make(map[string]config.LLMProviderConfig),
		Databases: make(map[string]config.DatabaseProviderConfig),
		Embedders: make(map[string]config.EmbedderProviderConfig),
		Agents:    make(map[string]config.AgentConfig),
		Tools:     config.ToolConfigs{Tools: make(map[string]config.ToolConfig)},
		Plugins: config.PluginConfigs{
			Discovery:           config.PluginDiscoveryConfig{Enabled: false},
			LLMProviders:        make(map[string]config.PluginConfig),
			DatabaseProviders:   make(map[string]config.PluginConfig),
			EmbedderProviders:   make(map[string]config.PluginConfig),
			ToolProviders:       make(map[string]config.PluginConfig),
			ReasoningStrategies: make(map[string]config.PluginConfig),
		},
	}
}

func TestNewComponentManager_MinimalConfig(t *testing.T) {
	cfg := createMinimalConfig()

	cm, err := NewComponentManager(cfg)
	if err != nil {
		t.Fatalf("NewComponentManager failed with minimal config: %v", err)
	}

	// Verify all registries are initialized
	if cm.GetLLMRegistry() == nil {
		t.Error("LLM registry is nil")
	}
	if cm.GetDatabaseRegistry() == nil {
		t.Error("Database registry is nil")
	}
	if cm.GetEmbedderRegistry() == nil {
		t.Error("Embedder registry is nil")
	}
	if cm.GetToolRegistry() == nil {
		t.Error("Tool registry is nil")
	}
	if cm.GetPluginRegistry() == nil {
		t.Error("Plugin registry is nil")
	}
	if cm.GetGlobalConfig() != cfg {
		t.Error("Global config not stored correctly")
	}
}

func TestComponentManager_AllGettersReturnNonNil(t *testing.T) {
	cfg := createMinimalConfig()
	cm, err := NewComponentManager(cfg)
	if err != nil {
		t.Fatalf("NewComponentManager failed: %v", err)
	}

	tests := []struct {
		name   string
		getter func() interface{}
	}{
		{"GetLLMRegistry", func() interface{} { return cm.GetLLMRegistry() }},
		{"GetDatabaseRegistry", func() interface{} { return cm.GetDatabaseRegistry() }},
		{"GetEmbedderRegistry", func() interface{} { return cm.GetEmbedderRegistry() }},
		{"GetToolRegistry", func() interface{} { return cm.GetToolRegistry() }},
		{"GetPluginRegistry", func() interface{} { return cm.GetPluginRegistry() }},
		{"GetGlobalConfig", func() interface{} { return cm.GetGlobalConfig() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter()
			if result == nil {
				t.Errorf("%s returned nil", tt.name)
			}
		})
	}
}

func TestComponentManager_GetLLM_NotFound(t *testing.T) {
	cfg := createMinimalConfig()
	cm, err := NewComponentManager(cfg)
	if err != nil {
		t.Fatalf("NewComponentManager failed: %v", err)
	}

	_, err = cm.GetLLM("non-existent-llm")
	if err == nil {
		t.Error("Expected error when getting non-existent LLM, got nil")
	}
}

func TestComponentManager_GetDatabase_NotFound(t *testing.T) {
	cfg := createMinimalConfig()
	cm, err := NewComponentManager(cfg)
	if err != nil {
		t.Fatalf("NewComponentManager failed: %v", err)
	}

	_, err = cm.GetDatabase("non-existent-db")
	if err == nil {
		t.Error("Expected error when getting non-existent database, got nil")
	}
}

func TestComponentManager_GetEmbedder_NotFound(t *testing.T) {
	cfg := createMinimalConfig()
	cm, err := NewComponentManager(cfg)
	if err != nil {
		t.Fatalf("NewComponentManager failed: %v", err)
	}

	_, err = cm.GetEmbedder("non-existent-embedder")
	if err == nil {
		t.Error("Expected error when getting non-existent embedder, got nil")
	}
}

func TestComponentManager_ShutdownPlugins(t *testing.T) {
	cfg := createMinimalConfig()
	cm, err := NewComponentManager(cfg)
	if err != nil {
		t.Fatalf("NewComponentManager failed: %v", err)
	}

	// Should not panic even with no plugins loaded
	err = cm.ShutdownPlugins(nil)
	if err != nil {
		t.Errorf("ShutdownPlugins returned error: %v", err)
	}
}

func TestIsPluginConfigured(t *testing.T) {
	cfg := createMinimalConfig()
	cfg.Plugins.LLMProviders = map[string]config.PluginConfig{
		"llm-plugin": {Name: "llm-plugin", Enabled: true},
	}
	cfg.Plugins.DatabaseProviders = map[string]config.PluginConfig{
		"db-plugin": {Name: "db-plugin", Enabled: true},
	}
	cfg.Plugins.EmbedderProviders = map[string]config.PluginConfig{
		"embedder-plugin": {Name: "embedder-plugin", Enabled: true},
	}
	cfg.Plugins.ToolProviders = map[string]config.PluginConfig{
		"tool-plugin": {Name: "tool-plugin", Enabled: true},
	}
	cfg.Plugins.ReasoningStrategies = map[string]config.PluginConfig{
		"reasoning-plugin": {Name: "reasoning-plugin", Enabled: true},
	}

	cm, err := NewComponentManager(cfg)
	if err != nil {
		t.Fatalf("NewComponentManager failed: %v", err)
	}

	tests := []struct {
		name       string
		pluginName string
		want       bool
	}{
		{"LLM plugin configured", "llm-plugin", true},
		{"Database plugin configured", "db-plugin", true},
		{"Embedder plugin configured", "embedder-plugin", true},
		{"Tool plugin configured", "tool-plugin", true},
		{"Reasoning plugin configured", "reasoning-plugin", true},
		{"Non-existent plugin", "non-existent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cm.isPluginConfigured(tt.pluginName, &cfg.Plugins)
			if got != tt.want {
				t.Errorf("isPluginConfigured(%q) = %v, want %v", tt.pluginName, got, tt.want)
			}
		})
	}
}

// ============================================================================
// COVERAGE NOTES:
// These tests cover the critical initialization paths:
// - Registry creation (LLM, Database, Embedder, Tool, Plugin)
// - Config storage
// - Error handling for missing components
// - Plugin configuration checks
// - Graceful shutdown
//
// NOT tested (would require real services):
// - Actual LLM initialization with API keys
// - Actual database connections (Qdrant)
// - Actual embedder connections (Ollama)
// - Plugin loading from disk
//
// These tests verify the WIRING works, which is the critical part.
// Full integration tests would test actual service connections.
// ============================================================================
