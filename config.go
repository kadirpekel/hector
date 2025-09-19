package hector

import (
	"context"
	"fmt"
	"os"

	"github.com/kadirpekel/hector/databases"
	"github.com/kadirpekel/hector/embedders"
	"github.com/kadirpekel/hector/llms"
	"github.com/kadirpekel/hector/providers"
	"gopkg.in/yaml.v3"
)

// ============================================================================
// YAML CONFIGURATION
// ============================================================================

// Default provider names
const (
	DefaultLLMProvider      = "ollama"
	DefaultDatabaseProvider = "qdrant"
	DefaultEmbedderProvider = "ollama"
)

// AgentConfig represents the complete agent configuration
type AgentConfig struct {
	Agent      AgentInfo          `yaml:"agent"`
	LLM        YAMLProviderConfig `yaml:"llm"`
	Memory     YAMLProviderConfig `yaml:"memory"`
	Embedder   YAMLProviderConfig `yaml:"embedder"`
	Search     SearchConfig       `yaml:"search"`
	Models     []ModelConfig      `yaml:"models"`
	MCPServers []MCPServerConfig  `yaml:"mcp_servers"`
	Reasoning  ReasoningConfig    `yaml:"reasoning"`
}

// AgentInfo contains basic agent information
type AgentInfo struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// YAMLProviderConfig holds provider configuration with dynamic provider lookup
type YAMLProviderConfig struct {
	Name   string                 `yaml:"name"` // Provider name (e.g., "ollama", "openai", "qdrant")
	Config map[string]interface{} `yaml:"-"`    // Dynamic configuration for the provider (populated via custom unmarshaling)
}

// UnmarshalYAML implements custom YAML unmarshaling to flatten the config structure
func (y *YAMLProviderConfig) UnmarshalYAML(value *yaml.Node) error {
	// Initialize the config map
	y.Config = make(map[string]interface{})

	// Parse the YAML node
	var raw map[string]interface{}
	if err := value.Decode(&raw); err != nil {
		return err
	}

	// Extract the provider name
	if name, exists := raw["name"]; exists {
		if nameStr, ok := name.(string); ok {
			y.Name = nameStr
		}
	}

	// Move all other fields to the config map
	for key, value := range raw {
		if key != "name" {
			y.Config[key] = value
		}
	}

	return nil
}

// MarshalYAML implements custom YAML marshaling to flatten the config structure
func (y YAMLProviderConfig) MarshalYAML() (interface{}, error) {
	result := make(map[string]interface{})
	result["name"] = y.Name

	// Add all config fields at the top level
	for key, value := range y.Config {
		result[key] = value
	}

	return result, nil
}

// SearchConfig holds search and context configuration
type SearchConfig struct {
	MaxContextLength int    `yaml:"max_context_length"`
	ContextStrategy  string `yaml:"context_strategy"`
	EnableReranking  bool   `yaml:"enable_reranking"`
}

// SetDefaults sets default values for SearchConfig
func (s *SearchConfig) SetDefaults() {
	if s.MaxContextLength == 0 {
		s.MaxContextLength = 2000
	}
	if s.ContextStrategy == "" {
		s.ContextStrategy = "relevance"
	}
}

// ReasoningConfig holds reasoning and execution configuration
type ReasoningConfig struct {
	Strategy       string              `yaml:"strategy"`
	MaxSteps       int                 `yaml:"max_steps"`
	MaxRetries     int                 `yaml:"max_retries"`
	EnableRetry    bool                `yaml:"enable_retry"`
	EnableFeedback bool                `yaml:"enable_feedback"`
	ToolExecution  ToolExecutionConfig `yaml:"tool_execution"`
	Steps          []ReasoningStep     `yaml:"steps"`
	ErrorHandling  ErrorHandlingConfig `yaml:"error_handling"`
	Context        ContextConfig       `yaml:"context"`
}

// SetDefaults sets default values for ReasoningConfig
func (r *ReasoningConfig) SetDefaults() {
	if r.Strategy == "" {
		r.Strategy = "single_shot"
	}
	if r.MaxSteps == 0 {
		r.MaxSteps = 1
	}
	if r.MaxRetries == 0 {
		r.MaxRetries = 2
	}
	r.ToolExecution.SetDefaults()
	r.ErrorHandling.SetDefaults()
	r.Context.SetDefaults()
}

// ToolExecutionConfig holds tool execution specific settings
type ToolExecutionConfig struct {
	ParallelExecution bool `yaml:"parallel_execution"`
	TimeoutSeconds    int  `yaml:"timeout_seconds"`
	RetryDelayMs      int  `yaml:"retry_delay_ms"`
	MaxConcurrent     int  `yaml:"max_concurrent"`
}

// SetDefaults sets default values for ToolExecutionConfig
func (t *ToolExecutionConfig) SetDefaults() {
	if t.TimeoutSeconds == 0 {
		t.TimeoutSeconds = 30
	}
	if t.RetryDelayMs == 0 {
		t.RetryDelayMs = 1000
	}
	if t.MaxConcurrent == 0 {
		t.MaxConcurrent = 3
	}
}

// ReasoningStep defines a custom reasoning step
type ReasoningStep struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Type        string                 `yaml:"type"`
	Enabled     bool                   `yaml:"enabled"`
	AgentConfig *AgentConfig           `yaml:"agent_config,omitempty"`
	Config      map[string]interface{} `yaml:"config,omitempty"`
}

// ErrorHandlingConfig holds error handling settings
type ErrorHandlingConfig struct {
	Strategy         string   `yaml:"strategy"`
	MaxErrorAnalysis int      `yaml:"max_error_analysis"`
	ErrorThreshold   float64  `yaml:"error_threshold"`
	EnableLearning   bool     `yaml:"enable_learning"`
	ErrorCategories  []string `yaml:"error_categories"`
}

// SetDefaults sets default values for ErrorHandlingConfig
func (e *ErrorHandlingConfig) SetDefaults() {
	if e.Strategy == "" {
		e.Strategy = "retry"
	}
	if e.MaxErrorAnalysis == 0 {
		e.MaxErrorAnalysis = 1
	}
	if e.ErrorThreshold == 0 {
		e.ErrorThreshold = 0.5
	}
}

// ContextConfig holds context management settings
type ContextConfig struct {
	PreserveHistory    bool `yaml:"preserve_history"`
	MaxHistorySteps    int  `yaml:"max_history_steps"`
	EnableContextShare bool `yaml:"enable_context_share"`
	ContextWindow      int  `yaml:"context_window"`
}

// SetDefaults sets default values for ContextConfig
func (c *ContextConfig) SetDefaults() {
	if c.MaxHistorySteps == 0 {
		c.MaxHistorySteps = 10
	}
	if c.ContextWindow == 0 {
		c.ContextWindow = 3
	}
}

// ============================================================================
// YAML CONFIGURATION LOADER
// ============================================================================

// LoadAgentFromFile loads Agent configuration from a YAML file
func LoadAgentFromFile(filename string) (*Agent, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Register default providers
	if err := providers.RegisterDefaultProviders(); err != nil {
		return nil, fmt.Errorf("failed to register providers: %w", err)
	}

	// Create Agent instance from config
	return createAgentFromConfig(&config)
}

// createAgentFromConfig creates an Agent instance from configuration
func createAgentFromConfig(config *AgentConfig) (*Agent, error) {
	agent := NewAgent()

	// Configure LLM
	if err := configureProvider(agent, &config.LLM, "llm"); err != nil {
		return nil, fmt.Errorf("failed to configure LLM: %w", err)
	}

	// Configure Database
	if err := configureProvider(agent, &config.Memory, "database"); err != nil {
		return nil, fmt.Errorf("failed to configure memory database: %w", err)
	}

	// Configure Embedder
	if err := configureProvider(agent, &config.Embedder, "embedder"); err != nil {
		return nil, fmt.Errorf("failed to configure embedder: %w", err)
	}

	// Configure MCP Servers and discover tools
	if err := configureMCPServers(agent, config.MCPServers); err != nil {
		return nil, fmt.Errorf("failed to configure MCP servers: %w", err)
	}

	// Configure models
	if err := configureModels(agent, config.Models); err != nil {
		return nil, fmt.Errorf("failed to configure models: %w", err)
	}

	// Configure reasoning
	agent.ReasoningConfig = config.Reasoning
	fmt.Printf("Loaded reasoning config: strategy=%s, max_steps=%d, steps_count=%d\n",
		config.Reasoning.Strategy, config.Reasoning.MaxSteps, len(config.Reasoning.Steps))

	return agent, nil
}

// configureProvider is a generic function that configures any provider type
func configureProvider(agent *Agent, config *YAMLProviderConfig, providerType string) error {
	if config.Name == "" {
		// Set default provider based on type
		switch providerType {
		case "llm":
			config.Name = DefaultLLMProvider
		case "database":
			config.Name = DefaultDatabaseProvider
		case "embedder":
			config.Name = DefaultEmbedderProvider
		default:
			return fmt.Errorf("unknown provider type: %s", providerType)
		}
	}

	if config.Config == nil {
		config.Config = make(map[string]interface{})
	}

	// Add the provider name to the config map
	configMap := make(map[string]interface{})
	configMap["provider"] = config.Name

	// Merge the dynamic config
	for key, value := range config.Config {
		configMap[key] = value
	}

	// Create provider dynamically based on type
	var provider interface{}
	var err error

	switch providerType {
	case "llm":
		provider, err = providers.CreateLLMProvider(configMap)
		if err != nil {
			return fmt.Errorf("failed to create LLM provider '%s': %w", config.Name, err)
		}
		agent.WithLLM(provider.(llms.LLMProvider))

	case "database":
		provider, err = providers.CreateDatabaseProvider(configMap)
		if err != nil {
			return fmt.Errorf("failed to create database provider '%s': %w", config.Name, err)
		}
		agent.WithDatabase(provider.(databases.VectorDB))

	case "embedder":
		provider, err = providers.CreateEmbedderProvider(configMap)
		if err != nil {
			return fmt.Errorf("failed to create embedder provider '%s': %w", config.Name, err)
		}
		agent.WithEmbedder(provider.(embedders.EmbeddingProvider))

	default:
		return fmt.Errorf("unknown provider type: %s", providerType)
	}

	return nil
}

// createDefaultModel creates a default document model configuration
func createDefaultModel() ModelConfig {
	return ModelConfig{
		Name:            "document",
		Collection:      "documents",
		EmbeddingFields: []string{"content"},
		MetadataFields:  []string{"title", "source", "author"},
		DefaultTopK:     10,
		MaxTopK:         100,
		Fields: []FieldInfo{
			{Name: "title", Type: "string", Purpose: "meta", Required: false},
			{Name: "content", Type: "string", Purpose: "embed", Required: true},
			{Name: "source", Type: "string", Purpose: "meta", Required: false},
			{Name: "author", Type: "string", Purpose: "meta", Required: false},
		},
	}
}

// configureModels configures document models from YAML config
func configureModels(agent *Agent, models []ModelConfig) error {
	if len(models) == 0 {
		// Set default document model if none specified
		models = []ModelConfig{createDefaultModel()}
	}

	// Validate and create model map
	for _, model := range models {
		if err := ValidateModelConfig(model); err != nil {
			return fmt.Errorf("invalid model config '%s': %w", model.Name, err)
		}
	}

	// Create model map and configure agent
	modelMap := CreateModelMap(models)
	agent.WithModelsFromConfig(modelMap)

	fmt.Printf("Configured %d models: %v\n", len(models), GetAllModelNames(modelMap))
	return nil
}

// configureMCPServers configures MCP servers and discovers their tools
func configureMCPServers(agent *Agent, servers []MCPServerConfig) error {
	// Add servers to agent
	agent.WithMCPServers(servers...)

	// Discover tools from all servers
	return agent.GetMCP().DiscoverAllTools(context.Background())
}

// ============================================================================
// CONVENIENCE FUNCTIONS
// ============================================================================

// NewAgentFromYAML creates a new Agent instance from a YAML file
// This is an alias for LoadAgentFromFile for backward compatibility
func NewAgentFromYAML(filename string) (*Agent, error) {
	return LoadAgentFromFile(filename)
}
