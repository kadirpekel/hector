package component

import (
	"fmt"

	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/databases"
	"github.com/kadirpekel/hector/embedders"
	"github.com/kadirpekel/hector/llms"
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
}

// NewComponentManager creates a new component manager and initializes all components
func NewComponentManager(globalConfig *config.Config) (*ComponentManager, error) {
	// Initialize tool registry with configuration
	toolRegistry, err := tools.NewToolRegistryWithConfig(&globalConfig.Tools)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool registry: %w", err)
	}

	cm := &ComponentManager{
		globalConfig:     globalConfig,
		llmRegistry:      llms.NewLLMRegistry(),
		dbRegistry:       databases.NewDatabaseRegistry(),
		embedderRegistry: embedders.NewEmbedderRegistry(),
		toolRegistry:     toolRegistry,
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
// AGENT COMPONENT CREATION
// ============================================================================
