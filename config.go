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
	Agent        AgentInfo          `yaml:"agent"`
	LLM          YAMLProviderConfig `yaml:"llm"`
	Memory       YAMLProviderConfig `yaml:"memory"`
	Embedder     YAMLProviderConfig `yaml:"embedder"`
	Search       SearchConfig       `yaml:"search"`
	Models       []ModelConfig      `yaml:"models"`
	MCPServers   []MCPServerConfig  `yaml:"mcp_servers"`
	CommandTools *SecurityConfig    `yaml:"command_tools"`
	Reasoning    ReasoningConfig    `yaml:"reasoning"`

	// New global configurations
	Sources map[string]SourceConfig `yaml:"sources"`
}

// SetDefaults sets default values for AgentConfig
func (a *AgentConfig) SetDefaults() {
	// Set default agent info if not specified
	if a.Agent.Name == "" {
		a.Agent.Name = "hector-agent"
	}
	if a.Agent.Description == "" {
		a.Agent.Description = "AI agent powered by Hector"
	}

	// Set default LLM model if not specified
	if a.LLM.Name != "" && len(a.LLM.Config) > 0 {
		if _, hasModel := a.LLM.Config["model"]; !hasModel {
			// Set default model based on provider
			switch a.LLM.Name {
			case "openai":
				a.LLM.Config["model"] = "gpt-4o-mini" // Cheapest OpenAI model
			case "ollama":
				a.LLM.Config["model"] = "llama3.2" // Common Ollama model
			case "tgi":
				a.LLM.Config["model"] = "microsoft/DialoGPT-medium" // Common TGI model
			}
		}
	}

	// Set defaults for reasoning configuration
	a.Reasoning.SetDefaults()

	// Set defaults for search configuration
	a.Search.SetDefaults()

	// Set default command tools config
	if a.CommandTools == nil {
		a.CommandTools = &SecurityConfig{
			AllowedCommands: []string{
				// File operations
				"cat", "head", "tail", "less", "more",
				"ls", "dir", "find", "locate", "which", "whereis",
				"cp", "mv", "rm", "mkdir", "rmdir", "touch",
				"chmod", "chown", "stat", "file", "du", "df",
				// Text processing
				"grep", "awk", "sed", "sort", "uniq", "cut", "paste",
				"wc", "tr", "diff", "patch",
				// System info
				"pwd", "whoami", "id", "uname", "uptime", "ps", "top",
				"free", "df", "mount", "env", "printenv",
				// Development tools
				"git", "npm", "node", "python", "go", "gcc", "make",
				"curl", "wget", "ssh", "scp", "rsync",
			},
			WorkingDirectory: "./",
			MaxExecutionTime: 30,
			EnableSandboxing: true,
		}
	}
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

// ============================================================================
// SOURCE AND INGESTION CONFIGURATION
// ============================================================================

// SourceConfig represents a global source configuration
type SourceConfig struct {
	Type            string                 `yaml:"type"`              // "local", "s3", "minio", "gdrive"
	Path            string                 `yaml:"path"`              // Directory path, bucket, endpoint
	Region          string                 `yaml:"region"`            // AWS region for S3
	AccessKeyID     string                 `yaml:"access_key_id"`     // AWS access key
	SecretAccessKey string                 `yaml:"secret_access_key"` // AWS secret key
	Credentials     map[string]string      `yaml:"credentials"`       // Google Drive credentials
	Options         map[string]interface{} `yaml:"options"`
}

// ModelIngestionSource represents a source reference for model ingestion
type ModelIngestionSource struct {
	Source          string   `yaml:"source"`           // Reference to global source
	Pattern         string   `yaml:"pattern"`          // Wildcard pattern
	ExcludePatterns []string `yaml:"exclude_patterns"` // Patterns to exclude

	// Inline source (alternative to reference)
	InlineSource *SourceConfig `yaml:"inline_source"`
}

// ModelIngestionConfig represents ingestion configuration for a model
type ModelIngestionConfig struct {
	AutoSync     bool                   `yaml:"auto_sync"`
	SyncInterval string                 `yaml:"sync_interval"`
	Sources      []ModelIngestionSource `yaml:"sources"`
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
	Strategy        string                  `yaml:"strategy"`
	MaxSteps        int                     `yaml:"max_steps"`
	MaxRetries      int                     `yaml:"max_retries"`
	EnableRetry     bool                    `yaml:"enable_retry"`
	EnableFeedback  bool                    `yaml:"enable_feedback"`
	StreamingMode   string                  `yaml:"streaming_mode"`
	Verbose         bool                    `yaml:"verbose"`
	VerboseTemplate string                  `yaml:"verbose_template"` // Template for verbose output formatting
	ToolExecution   ToolExecutionConfig     `yaml:"tool_execution"`
	Steps           []ReasoningStep         `yaml:"steps"`
	ErrorHandling   ErrorHandlingConfig     `yaml:"error_handling"`
	Context         ContextConfig           `yaml:"context"`
	Dynamic         *DynamicReasoningConfig `yaml:"dynamic,omitempty"` // Dynamic reasoning configuration
}

// SetDefaults sets default values for ReasoningConfig
func (r *ReasoningConfig) SetDefaults() {
	if r.Strategy == "" {
		r.Strategy = "auto" // Auto mode intelligently selects optimal strategy
	}
	if r.MaxSteps == 0 {
		r.MaxSteps = 1
	}
	if r.MaxRetries == 0 {
		r.MaxRetries = 2
	}
	if r.StreamingMode == "" {
		// Set default streaming mode based on strategy
		if r.Strategy == "simple" {
			r.StreamingMode = "all_steps" // Simple mode can stream normally
		} else {
			r.StreamingMode = "final_only" // Multi-step modes default to final only
		}
	}
	// Verbose defaults to false (clean output by default)
	// VerboseTemplate defaults to terminal format when verbose is enabled
	if r.VerboseTemplate == "" {
		r.VerboseTemplate = "\033[90m{{.Message}}\033[0m" // Default terminal gray
	}

	// If no steps are specified, create a default step
	if len(r.Steps) == 0 {
		r.Steps = []ReasoningStep{
			{
				Name:    "main",
				Type:    "execute",
				Enabled: true,
			},
		}
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

// SetDefaults sets default values for ReasoningStep
func (r *ReasoningStep) SetDefaults() {
	if !r.Enabled {
		r.Enabled = true // Enable steps by default
	}
	if r.Type == "" {
		r.Type = "execute" // Default to execute type
	}
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

// DynamicReasoningConfig holds configuration for AI-driven dynamic reasoning
type DynamicReasoningConfig struct {
	MaxIterations        int     `yaml:"max_iterations"`         // Maximum reasoning iterations
	GoalThreshold        float64 `yaml:"goal_threshold"`         // AI-determined goal achievement threshold
	AdaptationThreshold  float64 `yaml:"adaptation_threshold"`   // When to adapt approach
	QualityThreshold     float64 `yaml:"quality_threshold"`      // Minimum quality to continue
	EnableSelfReflection bool    `yaml:"enable_self_reflection"` // AI evaluates its own performance
	EnableMetaReasoning  bool    `yaml:"enable_meta_reasoning"`  // AI reasons about reasoning
	EnableDynamicTools   bool    `yaml:"enable_dynamic_tools"`   // AI selects tools dynamically
	EnableGoalEvolution  bool    `yaml:"enable_goal_evolution"`  // Goals can evolve during execution
	Verbose              bool    `yaml:"verbose"`
	StreamingMode        string  `yaml:"streaming_mode"`
}

// SetDefaults sets default values for DynamicReasoningConfig
func (d *DynamicReasoningConfig) SetDefaults() {
	if d.MaxIterations == 0 {
		d.MaxIterations = 10
	}
	if d.GoalThreshold == 0 {
		d.GoalThreshold = 0.85
	}
	if d.AdaptationThreshold == 0 {
		d.AdaptationThreshold = 0.3
	}
	if d.QualityThreshold == 0 {
		d.QualityThreshold = 0.6
	}
	if d.StreamingMode == "" {
		d.StreamingMode = "all_steps"
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

	// Set defaults for configuration
	config.SetDefaults()

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

	// Configure LLM (required)
	if err := configureProvider(agent, &config.LLM, "llm"); err != nil {
		return nil, fmt.Errorf("failed to configure LLM: %w", err)
	}

	// Configure Database (use defaults if not specified)
	if config.Memory.Name != "" {
		if err := configureProvider(agent, &config.Memory, "database"); err != nil {
			return nil, fmt.Errorf("failed to configure memory database: %w", err)
		}
	} else {
		// Use default database configuration
		fmt.Println("No memory configuration specified, using default Qdrant setup")
		dbConfig := YAMLProviderConfig{
			Name: DefaultDatabaseProvider,
			Config: map[string]interface{}{
				"host":     "localhost",
				"port":     6334,
				"timeout":  30,
				"use_tls":  false,
				"insecure": false,
			},
		}
		if err := configureProvider(agent, &dbConfig, "database"); err != nil {
			return nil, fmt.Errorf("failed to configure default memory database: %w", err)
		}
	}

	// Configure Embedder (use defaults if not specified)
	if config.Embedder.Name != "" {
		if err := configureProvider(agent, &config.Embedder, "embedder"); err != nil {
			return nil, fmt.Errorf("failed to configure embedder: %w", err)
		}
	} else {
		// Use default embedder configuration
		fmt.Println("No embedder configuration specified, using default Ollama setup")
		embedderConfig := map[string]interface{}{
			"provider":    DefaultEmbedderProvider,
			"model":       "nomic-embed-text",
			"host":        "http://localhost:11434",
			"dimension":   768,
			"timeout":     30,
			"max_retries": 3,
		}
		embedder, err := providers.CreateEmbedderProvider(embedderConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create default embedder provider: %w", err)
		}
		agent.WithEmbedder(embedder)
	}

	// Configure MCP Servers and discover tools
	if err := configureMCPServers(agent, config.MCPServers); err != nil {
		return nil, fmt.Errorf("failed to configure MCP servers: %w", err)
	}

	// Configure command-line tools
	if config.CommandTools != nil {
		agent.WithCommandToolsConfig(config.CommandTools)
		fmt.Printf("Configured command-line tools with %d allowed commands\n", len(config.CommandTools.AllowedCommands))
	}

	// Configure models (always configure since we now always have memory database)
	if err := configureModels(agent, config.Models); err != nil {
		return nil, fmt.Errorf("failed to configure models: %w", err)
	}

	// Initialize ModelManager with sources and models
	if config.Sources != nil {
		sourceManager := NewSourceManager(config.Sources)
		agent.modelManager = NewModelManager(config.Models, sourceManager, agent)
		fmt.Printf("Initialized ModelManager with %d sources and %d models\n",
			len(config.Sources), len(config.Models))
	}

	// Configure reasoning
	agent.ReasoningConfig = config.Reasoning

	// Set defaults for reasoning configuration
	agent.ReasoningConfig.SetDefaults()

	// Set defaults for reasoning steps
	for i := range agent.ReasoningConfig.Steps {
		agent.ReasoningConfig.Steps[i].SetDefaults()
	}

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
		Name:        "document",
		Collection:  "documents",
		DefaultTopK: 10,
		MaxTopK:     100,
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
