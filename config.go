package hector

import (
	"fmt"
	"os"
	"time"

	"github.com/kadirpekel/hector/providers"
	"gopkg.in/yaml.v3"
)

// ============================================================================
// TYPED ENUMS
// ============================================================================

// ContextStrategy defines how search context is built
type ContextStrategy string

const (
	ContextStrategyRelevant      ContextStrategy = "relevant"
	ContextStrategySummary       ContextStrategy = "summary"
	ContextStrategyChronological ContextStrategy = "chronological"
)

// ============================================================================
// CORE CONFIGURATION TYPES
// ============================================================================

// LLMConfig represents provider configuration from YAML
type LLMConfig struct {
	Name string `yaml:"name"`
	// Specific typed fields for common providers
	Model       string  `yaml:"model,omitempty"`
	APIKey      string  `yaml:"api_key,omitempty"`
	BaseURL     string  `yaml:"base_url,omitempty"`
	Temperature float64 `yaml:"temperature,omitempty"`
	MaxTokens   int     `yaml:"max_tokens,omitempty"`
	// Fallback for provider-specific config
	Extra map[string]interface{} `yaml:",inline"`
}

// SearchConfig represents search configuration with models
type SearchConfig struct {
	// Model definitions - moved here from agent level
	Models []ModelConfig `yaml:"models,omitempty"`

	// Search parameters
	TopK             int             `yaml:"top_k,omitempty"`
	Threshold        float64         `yaml:"threshold,omitempty"`
	MaxContextLength int             `yaml:"max_context_length,omitempty"`
	ContextStrategy  ContextStrategy `yaml:"context_strategy,omitempty"`
	EnableReranking  bool            `yaml:"enable_reranking,omitempty"`
}

// ============================================================================
// AGENT CONFIGURATION
// ============================================================================

// PromptConfig represents prompt configuration
type PromptConfig struct {
	// Full template mode - user provides complete prompt template
	FullTemplate string `yaml:"full_template,omitempty"`

	// Component mode - system builds prompt from components
	SystemPrompt string            `yaml:"system_prompt,omitempty"`
	Template     string            `yaml:"template,omitempty"`
	Instructions string            `yaml:"instructions,omitempty"`
	Variables    map[string]string `yaml:"variables,omitempty"`

	// Context inclusion controls
	IncludeContext   bool `yaml:"include_context"`
	IncludeHistory   bool `yaml:"include_history"`
	IncludeTools     bool `yaml:"include_tools"`
	MaxContextLength int  `yaml:"max_context_length,omitempty"`
}

// CommandToolsConfig represents command-line tools configuration with security
type CommandToolsConfig struct {
	AllowedCommands  []string      `yaml:"allowed_commands"`
	WorkingDirectory string        `yaml:"working_directory"`
	MaxExecutionTime time.Duration `yaml:"max_execution_time"`
	EnableSandboxing bool          `yaml:"enable_sandboxing"`
}

// AgentConfig represents the main agent configuration
type AgentConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`

	// Core AI Configuration
	LLM      LLMConfig `yaml:"llm"`
	Database LLMConfig `yaml:"database,omitempty"` // Renamed from Memory - for vector storage
	Embedder LLMConfig `yaml:"embedder,omitempty"`

	// Prompt Configuration
	Prompt PromptConfig `yaml:"prompt,omitempty"`

	// Search Configuration (now includes models)
	Search SearchConfig `yaml:"search,omitempty"`

	// Tool Integration
	MCPServers   []MCPServerConfig   `yaml:"mcp_servers,omitempty"`
	CommandTools *CommandToolsConfig `yaml:"command_tools,omitempty"`
}

// ============================================================================
// CONFIGURATION UTILITIES
// ============================================================================

// Default provider names
const (
	DefaultLLMProvider      = "ollama"
	DefaultDatabaseProvider = "qdrant"
	DefaultEmbedderProvider = "ollama"
)

// ============================================================================
// AGENT LOADING
// ============================================================================

// LoadAgentConfig loads an agent configuration from a YAML file
func LoadAgentConfig(filePath string) (*AgentConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent config: %w", err)
	}

	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse agent config: %w", err)
	}

	// Set defaults
	if config.LLM.Name == "" {
		config.LLM.Name = DefaultLLMProvider
	}

	return &config, nil
}

// LoadAgentFromFile loads an Agent from a YAML file
func LoadAgentFromFile(filename string) (*Agent, error) {
	// Register default providers
	if err := providers.RegisterDefaultProviders(); err != nil {
		return nil, fmt.Errorf("failed to register providers: %w", err)
	}

	// Load agent config
	agentConfig, err := LoadAgentConfig(filename)
	if err != nil {
		return nil, err
	}

	// Create Agent instance from config
	return NewAgent(agentConfig)
}
