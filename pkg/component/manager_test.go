package component

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/plugins"
	"github.com/kadirpekel/hector/pkg/tools"
)

func TestIsPluginConfigured(t *testing.T) {

	cm := &ComponentManager{
		globalConfig: &config.Config{},
	}

	tests := []struct {
		name         string
		pluginName   string
		pluginConfig *config.PluginConfigs
		want         bool
	}{
		{
			name:       "LLM provider configured",
			pluginName: "custom-llm",
			pluginConfig: &config.PluginConfigs{
				LLMProviders: map[string]*config.PluginConfig{
					"custom-llm": {Enabled: config.BoolPtr(true)},
				},
			},
			want: true,
		},
		{
			name:       "Database provider configured",
			pluginName: "custom-db",
			pluginConfig: &config.PluginConfigs{
				DatabaseProviders: map[string]*config.PluginConfig{
					"custom-db": {Enabled: config.BoolPtr(true)},
				},
			},
			want: true,
		},
		{
			name:       "Embedder provider configured",
			pluginName: "custom-embedder",
			pluginConfig: &config.PluginConfigs{
				EmbedderProviders: map[string]*config.PluginConfig{
					"custom-embedder": {Enabled: config.BoolPtr(true)},
				},
			},
			want: true,
		},
		{
			name:       "Tool provider configured",
			pluginName: "custom-tool",
			pluginConfig: &config.PluginConfigs{
				ToolProviders: map[string]*config.PluginConfig{
					"custom-tool": {Enabled: config.BoolPtr(true)},
				},
			},
			want: true,
		},
		{
			name:       "Reasoning strategy configured",
			pluginName: "custom-strategy",
			pluginConfig: &config.PluginConfigs{
				ReasoningStrategies: map[string]*config.PluginConfig{
					"custom-strategy": {Enabled: config.BoolPtr(true)},
				},
			},
			want: true,
		},
		{
			name:         "Plugin not configured",
			pluginName:   "nonexistent",
			pluginConfig: &config.PluginConfigs{},
			want:         false,
		},
		{
			name:       "Plugin configured in multiple types",
			pluginName: "multi-plugin",
			pluginConfig: &config.PluginConfigs{
				LLMProviders: map[string]*config.PluginConfig{
					"multi-plugin": {Enabled: config.BoolPtr(true)},
				},
				DatabaseProviders: map[string]*config.PluginConfig{
					"multi-plugin": {Enabled: config.BoolPtr(true)},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cm.isPluginConfigured(tt.pluginName, tt.pluginConfig)
			if got != tt.want {
				t.Errorf("isPluginConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComponentManager_GettersNotNil(t *testing.T) {

	cm := &ComponentManager{
		globalConfig: &config.Config{
			Agents: map[string]*config.AgentConfig{},
		},
	}

	if cm.GetGlobalConfig() == nil {
		t.Error("GetGlobalConfig() returned nil")
	}

	if cm.GetGlobalConfig() != cm.globalConfig {
		t.Error("GetGlobalConfig() doesn't return the correct config")
	}
}

func TestComponentManager_GettersReturnSetValues(t *testing.T) {

	testConfig := &config.Config{
		Agents: map[string]*config.AgentConfig{
			"test-agent": {},
		},
	}

	cm := &ComponentManager{
		globalConfig: testConfig,
	}

	if got := cm.GetGlobalConfig(); got != testConfig {
		t.Error("GetGlobalConfig() didn't return the same instance")
	}

	if len(cm.GetGlobalConfig().Agents) != 1 {
		t.Errorf("Expected 1 agent in config, got %d", len(cm.GetGlobalConfig().Agents))
	}
}

func TestComponentManager_GetLLM_NotFound(t *testing.T) {

	cm := &ComponentManager{
		llmRegistry: llms.NewLLMRegistry(),
	}

	_, err := cm.GetLLM("nonexistent-llm")
	if err == nil {
		t.Error("Expected error when getting nonexistent LLM, got nil")
	}

	if err != nil && err.Error() == "" {
		t.Error("Error message is empty")
	}
}

func TestComponentManager_GetDatabase_NotFound(t *testing.T) {

	cm := &ComponentManager{
		dbRegistry: databases.NewDatabaseRegistry(),
	}

	_, err := cm.GetDatabase("nonexistent-db")
	if err == nil {
		t.Error("Expected error when getting nonexistent database, got nil")
	}
}

func TestComponentManager_GetEmbedder_NotFound(t *testing.T) {

	cm := &ComponentManager{
		embedderRegistry: embedders.NewEmbedderRegistry(),
	}

	_, err := cm.GetEmbedder("nonexistent-embedder")
	if err == nil {
		t.Error("Expected error when getting nonexistent embedder, got nil")
	}
}

func TestComponentManager_RegistryGetters(t *testing.T) {

	llmReg := llms.NewLLMRegistry()
	dbReg := databases.NewDatabaseRegistry()
	embedderReg := embedders.NewEmbedderRegistry()
	toolReg := tools.NewToolRegistry()
	pluginReg := plugins.NewPluginRegistry(nil)

	cm := &ComponentManager{
		llmRegistry:      llmReg,
		dbRegistry:       dbReg,
		embedderRegistry: embedderReg,
		toolRegistry:     toolReg,
		pluginRegistry:   pluginReg,
	}

	if cm.GetLLMRegistry() != llmReg {
		t.Error("GetLLMRegistry doesn't return the set instance")
	}

	if cm.GetDatabaseRegistry() != dbReg {
		t.Error("GetDatabaseRegistry doesn't return the set instance")
	}

	if cm.GetEmbedderRegistry() != embedderReg {
		t.Error("GetEmbedderRegistry doesn't return the set instance")
	}

	if cm.GetToolRegistry() != toolReg {
		t.Error("GetToolRegistry doesn't return the set instance")
	}

	if cm.GetPluginRegistry() != pluginReg {
		t.Error("GetPluginRegistry doesn't return the set instance")
	}
}

func TestIsPluginConfigured_EdgeCases(t *testing.T) {
	cm := &ComponentManager{
		globalConfig: &config.Config{},
	}

	tests := []struct {
		name         string
		pluginName   string
		pluginConfig *config.PluginConfigs
		want         bool
	}{
		{
			name:         "empty plugin name",
			pluginName:   "",
			pluginConfig: &config.PluginConfigs{},
			want:         false,
		},
		{
			name:         "nil plugin config",
			pluginName:   "test",
			pluginConfig: nil,
			want:         false,
		},
		{
			name:       "plugin disabled but configured",
			pluginName: "disabled-plugin",
			pluginConfig: &config.PluginConfigs{
				LLMProviders: map[string]*config.PluginConfig{
					"disabled-plugin": {Enabled: config.BoolPtr(false)},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pluginConfig == nil {

				return
			}

			got := cm.isPluginConfigured(tt.pluginName, tt.pluginConfig)
			if got != tt.want {
				t.Errorf("isPluginConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComponentManager_MultipleRegistryAccess(t *testing.T) {

	cm := &ComponentManager{
		llmRegistry:      llms.NewLLMRegistry(),
		dbRegistry:       databases.NewDatabaseRegistry(),
		embedderRegistry: embedders.NewEmbedderRegistry(),
	}

	llmReg1 := cm.GetLLMRegistry()
	llmReg2 := cm.GetLLMRegistry()

	if llmReg1 != llmReg2 {
		t.Error("GetLLMRegistry returns different instances on multiple calls")
	}

	dbReg1 := cm.GetDatabaseRegistry()
	dbReg2 := cm.GetDatabaseRegistry()

	if dbReg1 != dbReg2 {
		t.Error("GetDatabaseRegistry returns different instances on multiple calls")
	}
}
