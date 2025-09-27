// Package config provides configuration types and utilities for the Hector AI agent framework.
// This file contains the main unified configuration entry point.
package config

import (
	"fmt"
)

// ============================================================================
// MAIN UNIFIED CONFIGURATION
// ============================================================================

// HectorConfig represents the complete Hector configuration
// Similar to docker-compose.yml, this is the single entry point for all configuration
type HectorConfig struct {
	// Version and metadata
	Version     string            `yaml:"version,omitempty"`
	Name        string            `yaml:"name,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`

	// Global settings
	Global GlobalSettings `yaml:"global,omitempty"`

	// Provider configurations (shared across agents)
	Providers ProviderConfigs `yaml:"providers,omitempty"`

	// Agent definitions
	Agents map[string]AgentConfig `yaml:"agents,omitempty"`

	// Workflow definitions
	Workflows map[string]WorkflowConfig `yaml:"workflows,omitempty"`

	// Tool configurations
	Tools ToolConfigs `yaml:"tools,omitempty"`

	// Document store configurations
	DocumentStores map[string]DocumentStoreConfig `yaml:"document_stores,omitempty"`
}

// Validate implements Config.Validate for HectorConfig
func (c *HectorConfig) Validate() error {
	// Validate global settings
	if err := c.Global.Validate(); err != nil {
		return fmt.Errorf("global settings validation failed: %w", err)
	}

	// Validate providers
	if err := c.Providers.Validate(); err != nil {
		return fmt.Errorf("providers validation failed: %w", err)
	}

	// Validate agents
	for name, agent := range c.Agents {
		if err := agent.Validate(); err != nil {
			return fmt.Errorf("agent '%s' validation failed: %w", name, err)
		}
	}

	// Validate workflows
	for name, workflow := range c.Workflows {
		if err := workflow.Validate(); err != nil {
			return fmt.Errorf("workflow '%s' validation failed: %w", name, err)
		}
	}

	// Validate tools
	if err := c.Tools.Validate(); err != nil {
		return fmt.Errorf("tools validation failed: %w", err)
	}

	// Validate document stores
	for name, store := range c.DocumentStores {
		if err := store.Validate(); err != nil {
			return fmt.Errorf("document store '%s' validation failed: %w", name, err)
		}
	}

	return nil
}

// SetDefaults implements Config.SetDefaults for HectorConfig
func (c *HectorConfig) SetDefaults() {
	// Set global defaults
	c.Global.SetDefaults()

	// Set provider defaults
	c.Providers.SetDefaults()

	// Set agent defaults
	for name := range c.Agents {
		agent := c.Agents[name]
		agent.SetDefaults()
		c.Agents[name] = agent
	}

	// Set workflow defaults
	for name := range c.Workflows {
		workflow := c.Workflows[name]
		workflow.SetDefaults()
		c.Workflows[name] = workflow
	}

	// Set tool defaults
	c.Tools.SetDefaults()

	// Set document store defaults
	for name := range c.DocumentStores {
		store := c.DocumentStores[name]
		store.SetDefaults()
		c.DocumentStores[name] = store
	}
}

// ============================================================================
// GLOBAL SETTINGS
// ============================================================================

// GlobalSettings contains global configuration settings
type GlobalSettings struct {
	// Logging configuration
	Logging LoggingConfig `yaml:"logging,omitempty"`

	// Performance settings
	Performance PerformanceConfig `yaml:"performance,omitempty"`
}

// Validate implements Config.Validate for GlobalSettings
func (c *GlobalSettings) Validate() error {
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config validation failed: %w", err)
	}
	if err := c.Performance.Validate(); err != nil {
		return fmt.Errorf("performance config validation failed: %w", err)
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for GlobalSettings
func (c *GlobalSettings) SetDefaults() {
	c.Logging.SetDefaults()
	c.Performance.SetDefaults()
}

// ============================================================================
// CONFIGURATION LOADING
// ============================================================================

// LoadHectorConfig loads the complete Hector configuration from a YAML file
// This is the main entry point for configuration loading
func LoadHectorConfig(filePath string) (*HectorConfig, error) {
	var config HectorConfig
	if err := LoadConfig(filePath, &config); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &config, nil
}

// LoadHectorConfigFromString loads configuration from a YAML string
func LoadHectorConfigFromString(yamlContent string) (*HectorConfig, error) {
	var config HectorConfig
	if err := loadConfigFromString(yamlContent, &config); err != nil {
		return nil, fmt.Errorf("failed to load config from string: %w", err)
	}
	return &config, nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// GetAgent returns an agent configuration by name
func (c *HectorConfig) GetAgent(name string) (*AgentConfig, bool) {
	agent, exists := c.Agents[name]
	return &agent, exists
}

// GetWorkflow returns a workflow configuration by name
func (c *HectorConfig) GetWorkflow(name string) (*WorkflowConfig, bool) {
	workflow, exists := c.Workflows[name]
	return &workflow, exists
}

// GetDocumentStore returns a document store configuration by name
func (c *HectorConfig) GetDocumentStore(name string) (*DocumentStoreConfig, bool) {
	store, exists := c.DocumentStores[name]
	return &store, exists
}

// ListAgents returns a list of all agent names
func (c *HectorConfig) ListAgents() []string {
	agents := make([]string, 0, len(c.Agents))
	for name := range c.Agents {
		agents = append(agents, name)
	}
	return agents
}

// ListWorkflows returns a list of all workflow names
func (c *HectorConfig) ListWorkflows() []string {
	workflows := make([]string, 0, len(c.Workflows))
	for name := range c.Workflows {
		workflows = append(workflows, name)
	}
	return workflows
}

// ListDocumentStores returns a list of all document store names
func (c *HectorConfig) ListDocumentStores() []string {
	stores := make([]string, 0, len(c.DocumentStores))
	for name := range c.DocumentStores {
		stores = append(stores, name)
	}
	return stores
}
