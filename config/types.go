// Package config provides configuration types and utilities for the AI agent framework.
// This file contains all configuration types in a unified structure.
package config

import (
	"fmt"
	"time"
)

// ============================================================================
// PROVIDER CONFIGURATIONS
// ============================================================================

// ProviderConfigs contains all provider configurations
type ProviderConfigs struct {
	// LLM providers
	LLMs map[string]LLMProviderConfig `yaml:"llms,omitempty"`

	// Database providers
	Databases map[string]DatabaseProviderConfig `yaml:"databases,omitempty"`

	// Embedder providers
	Embedders map[string]EmbedderProviderConfig `yaml:"embedders,omitempty"`
}

// Validate implements Config.Validate for ProviderConfigs
func (c *ProviderConfigs) Validate() error {
	for name, llm := range c.LLMs {
		if err := llm.Validate(); err != nil {
			return fmt.Errorf("LLM provider '%s' validation failed: %w", name, err)
		}
	}
	for name, db := range c.Databases {
		if err := db.Validate(); err != nil {
			return fmt.Errorf("database provider '%s' validation failed: %w", name, err)
		}
	}
	for name, embedder := range c.Embedders {
		if err := embedder.Validate(); err != nil {
			return fmt.Errorf("embedder provider '%s' validation failed: %w", name, err)
		}
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for ProviderConfigs
func (c *ProviderConfigs) SetDefaults() {
	for name := range c.LLMs {
		llm := c.LLMs[name]
		llm.SetDefaults()
		c.LLMs[name] = llm
	}
	for name := range c.Databases {
		db := c.Databases[name]
		db.SetDefaults()
		c.Databases[name] = db
	}
	for name := range c.Embedders {
		embedder := c.Embedders[name]
		embedder.SetDefaults()
		c.Embedders[name] = embedder
	}
}

// LLMProviderConfig represents LLM provider configuration
type LLMProviderConfig struct {
	Type        string  `yaml:"type"`        // "ollama", "openai"
	Model       string  `yaml:"model"`       // Model name
	APIKey      string  `yaml:"api_key"`     // API key (for OpenAI)
	Host        string  `yaml:"host"`        // Host for ollama or custom OpenAI endpoint
	Temperature float64 `yaml:"temperature"` // Temperature setting
	MaxTokens   int     `yaml:"max_tokens"`  // Max tokens
	Timeout     int     `yaml:"timeout"`     // Request timeout in seconds
}

// Validate implements Config.Validate for LLMProviderConfig
func (c *LLMProviderConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("type is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Type == "openai" && c.APIKey == "" {
		return fmt.Errorf("api_key is required for OpenAI")
	}
	if c.Temperature < 0 || c.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if c.MaxTokens < 0 {
		return fmt.Errorf("max_tokens must be non-negative")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for LLMProviderConfig
func (c *LLMProviderConfig) SetDefaults() {
	// Zero-config: Set default type and model if not specified
	if c.Type == "" {
		c.Type = "ollama"
	}
	if c.Model == "" {
		c.Model = "llama3.2" // Popular, well-supported model
	}
	if c.Host == "" {
		// Set default host based on provider type
		switch c.Type {
		case "openai":
			c.Host = "https://api.openai.com/v1"
		case "anthropic":
			c.Host = "https://api.anthropic.com"
		case "ollama":
			c.Host = "http://localhost:11434"
		default:
			c.Host = "http://localhost:11434" // Fallback to Ollama
		}
	}
	if c.Temperature == 0 {
		c.Temperature = 0.7
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = 2000
	}
	if c.Timeout == 0 {
		c.Timeout = 60
	}
}

// DatabaseProviderConfig represents database provider configuration
type DatabaseProviderConfig struct {
	Type     string `yaml:"type"`     // "qdrant"
	Host     string `yaml:"host"`     // Database host
	Port     int    `yaml:"port"`     // Database port
	APIKey   string `yaml:"api_key"`  // API key (optional)
	Timeout  int    `yaml:"timeout"`  // Connection timeout in seconds
	UseTLS   bool   `yaml:"use_tls"`  // Use TLS connection
	Insecure bool   `yaml:"insecure"` // Skip TLS verification
}

// Validate implements Config.Validate for DatabaseProviderConfig
func (c *DatabaseProviderConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("type is required")
	}
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for DatabaseProviderConfig
func (c *DatabaseProviderConfig) SetDefaults() {
	// Zero-config: Set default type and host if not specified
	if c.Type == "" {
		c.Type = "qdrant"
	}
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 6333
	}
	if c.Timeout == 0 {
		c.Timeout = 30
	}
}

// EmbedderProviderConfig represents embedder provider configuration
type EmbedderProviderConfig struct {
	Type       string `yaml:"type"`        // "ollama"
	Model      string `yaml:"model"`       // Model name
	Host       string `yaml:"host"`        // Host for ollama
	Dimension  int    `yaml:"dimension"`   // Embedding dimension
	Timeout    int    `yaml:"timeout"`     // Request timeout in seconds
	MaxRetries int    `yaml:"max_retries"` // Max retry attempts
}

// Validate implements Config.Validate for EmbedderProviderConfig
func (c *EmbedderProviderConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("type is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Dimension <= 0 {
		return fmt.Errorf("dimension must be positive")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for EmbedderProviderConfig
func (c *EmbedderProviderConfig) SetDefaults() {
	// Zero-config: Set default type, model, and host if not specified
	if c.Type == "" {
		c.Type = "ollama"
	}
	if c.Model == "" {
		c.Model = "nomic-embed-text" // Good general-purpose embedder
	}
	if c.Host == "" {
		c.Host = "http://localhost:11434"
	}
	if c.Dimension == 0 {
		c.Dimension = 768
	}
	if c.Timeout == 0 {
		c.Timeout = 30
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
}

// ============================================================================
// AGENT CONFIGURATIONS
// ============================================================================

// AgentConfig represents agent configuration
type AgentConfig struct {
	Name           string          `yaml:"name"`            // Agent name
	Description    string          `yaml:"description"`     // Agent description
	LLM            string          `yaml:"llm"`             // LLM provider reference
	Database       string          `yaml:"database"`        // Database provider reference
	Embedder       string          `yaml:"embedder"`        // Embedder provider reference
	DocumentStores []string        `yaml:"document_stores"` // Document store references
	Prompt         PromptConfig    `yaml:"prompt"`          // Prompt configuration
	Reasoning      ReasoningConfig `yaml:"reasoning"`       // Reasoning configuration
	Search         SearchConfig    `yaml:"search"`          // Search configuration
	Tools          ToolConfigs     `yaml:"tools"`           // Tool configuration
}

// Validate implements Config.Validate for AgentConfig
func (c *AgentConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.LLM == "" {
		return fmt.Errorf("llm provider reference is required")
	}
	// Database and embedder are only required if document stores are configured
	if len(c.DocumentStores) > 0 {
		if c.Database == "" {
			return fmt.Errorf("database provider reference is required when document stores are configured")
		}
		if c.Embedder == "" {
			return fmt.Errorf("embedder provider reference is required when document stores are configured")
		}
	}
	if err := c.Prompt.Validate(); err != nil {
		return fmt.Errorf("prompt configuration validation failed: %w", err)
	}
	if err := c.Reasoning.Validate(); err != nil {
		return fmt.Errorf("reasoning configuration validation failed: %w", err)
	}
	if err := c.Search.Validate(); err != nil {
		return fmt.Errorf("search configuration validation failed: %w", err)
	}
	if err := c.Tools.Validate(); err != nil {
		return fmt.Errorf("tools configuration validation failed: %w", err)
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for AgentConfig
func (c *AgentConfig) SetDefaults() {
	// Zero-config: Set default name and references if not specified
	if c.Name == "" {
		c.Name = "Assistant"
	}
	if c.Description == "" {
		c.Description = "AI assistant with local tools and knowledge"
	}
	if c.LLM == "" {
		c.LLM = "default-llm"
	}
	// Database, embedder, and document stores must be explicitly configured - no defaults

	c.Prompt.SetDefaults()
	c.Reasoning.SetDefaults()
	c.Search.SetDefaults()
	c.Tools.SetDefaults()
}

// ============================================================================
// WORKFLOW CONFIGURATIONS
// ============================================================================

// WorkflowConfig represents workflow configuration
type WorkflowConfig struct {
	Name        string               `yaml:"name"`        // Workflow name
	Description string               `yaml:"description"` // Workflow description
	Mode        ExecutionMode        `yaml:"mode"`        // Execution mode
	Agents      []string             `yaml:"agents"`      // Agent references
	Shared      SharedInfrastructure `yaml:"shared"`      // Shared infrastructure
	Execution   ExecutionConfig      `yaml:"execution"`   // Execution configuration
	Settings    WorkflowSettings     `yaml:"settings"`    // Workflow settings
}

// Validate implements Config.Validate for WorkflowConfig
func (c *WorkflowConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(c.Agents) == 0 {
		return fmt.Errorf("at least one agent is required")
	}
	if err := c.Shared.Validate(); err != nil {
		return fmt.Errorf("shared infrastructure validation failed: %w", err)
	}
	if err := c.Execution.Validate(); err != nil {
		return fmt.Errorf("execution configuration validation failed: %w", err)
	}
	if err := c.Settings.Validate(); err != nil {
		return fmt.Errorf("workflow settings validation failed: %w", err)
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for WorkflowConfig
func (c *WorkflowConfig) SetDefaults() {
	c.Shared.SetDefaults()
	c.Execution.SetDefaults()
	c.Settings.SetDefaults()
}

// ExecutionMode represents workflow execution mode
type ExecutionMode string

const (
	ExecutionModeDAG        ExecutionMode = "dag"
	ExecutionModeAutonomous ExecutionMode = "autonomous"
)

// SharedInfrastructure represents shared infrastructure configuration
type SharedInfrastructure struct {
	// Currently no shared infrastructure configurations are used
	// This struct is kept for future extensibility
}

// Validate implements Config.Validate for SharedInfrastructure
func (c *SharedInfrastructure) Validate() error {
	return nil
}

// SetDefaults implements Config.SetDefaults for SharedInfrastructure
func (c *SharedInfrastructure) SetDefaults() {
	// No defaults to set
}

// ExecutionConfig represents execution configuration
type ExecutionConfig struct {
	DAG        *DAGExecution        `yaml:"dag,omitempty"`        // DAG execution config
	Autonomous *AutonomousExecution `yaml:"autonomous,omitempty"` // Autonomous execution config
}

// Validate implements Config.Validate for ExecutionConfig
func (c *ExecutionConfig) Validate() error {
	if c.DAG != nil {
		if err := c.DAG.Validate(); err != nil {
			return fmt.Errorf("DAG execution validation failed: %w", err)
		}
	}
	if c.Autonomous != nil {
		if err := c.Autonomous.Validate(); err != nil {
			return fmt.Errorf("autonomous execution validation failed: %w", err)
		}
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for ExecutionConfig
func (c *ExecutionConfig) SetDefaults() {
	if c.DAG != nil {
		c.DAG.SetDefaults()
	}
	if c.Autonomous != nil {
		c.Autonomous.SetDefaults()
	}
}

// DAGExecution represents DAG execution configuration
type DAGExecution struct {
	Steps []WorkflowStep `yaml:"steps"` // Workflow steps
}

// Validate implements Config.Validate for DAGExecution
func (c *DAGExecution) Validate() error {
	if len(c.Steps) == 0 {
		return fmt.Errorf("at least one step is required for DAG execution")
	}
	for i, step := range c.Steps {
		if err := step.Validate(); err != nil {
			return fmt.Errorf("step %d validation failed: %w", i, err)
		}
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for DAGExecution
func (c *DAGExecution) SetDefaults() {
	for i := range c.Steps {
		c.Steps[i].SetDefaults()
	}
}

// WorkflowStep represents a workflow step
type WorkflowStep struct {
	Name      string        `yaml:"name"`       // Step name
	Agent     string        `yaml:"agent"`      // Agent reference
	Input     string        `yaml:"input"`      // Input expression
	Output    string        `yaml:"output"`     // Output variable
	DependsOn []string      `yaml:"depends_on"` // Dependencies
	Timeout   time.Duration `yaml:"timeout"`    // Step timeout
}

// Validate implements Config.Validate for WorkflowStep
func (c *WorkflowStep) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Agent == "" {
		return fmt.Errorf("agent reference is required")
	}
	if c.Input == "" {
		return fmt.Errorf("input is required")
	}
	if c.Output == "" {
		return fmt.Errorf("output is required")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for WorkflowStep
func (c *WorkflowStep) SetDefaults() {
	// No defaults to set
}

// AutonomousExecution represents autonomous execution configuration
type AutonomousExecution struct {
	Goal                  string                `yaml:"goal"`                   // Goal description
	Strategy              string                `yaml:"strategy"`               // Strategy type
	MaxIterations         int                   `yaml:"max_iterations"`         // Max iterations
	CoordinatorLLM        string                `yaml:"coordinator_llm"`        // Coordinator LLM reference
	TerminationConditions TerminationConditions `yaml:"termination_conditions"` // Termination conditions
}

// Validate implements Config.Validate for AutonomousExecution
func (c *AutonomousExecution) Validate() error {
	if c.Goal == "" {
		return fmt.Errorf("goal is required")
	}
	if c.Strategy == "" {
		return fmt.Errorf("strategy is required")
	}
	if c.MaxIterations <= 0 {
		return fmt.Errorf("max_iterations must be positive")
	}
	if c.CoordinatorLLM == "" {
		return fmt.Errorf("coordinator_llm is required")
	}
	if err := c.TerminationConditions.Validate(); err != nil {
		return fmt.Errorf("termination conditions validation failed: %w", err)
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for AutonomousExecution
func (c *AutonomousExecution) SetDefaults() {
	if c.Strategy == "" {
		c.Strategy = "dynamic"
	}
	if c.MaxIterations == 0 {
		c.MaxIterations = 10
	}
	c.TerminationConditions.SetDefaults()
}

// TerminationConditions represents termination conditions
type TerminationConditions struct {
	MaxDuration      time.Duration `yaml:"max_duration"`      // Max execution duration
	QualityThreshold float64       `yaml:"quality_threshold"` // Quality threshold
	MaxIterations    int           `yaml:"max_iterations"`    // Max iterations
}

// Validate implements Config.Validate for TerminationConditions
func (c *TerminationConditions) Validate() error {
	if c.MaxDuration < 0 {
		return fmt.Errorf("max_duration must be non-negative")
	}
	if c.QualityThreshold < 0 || c.QualityThreshold > 1 {
		return fmt.Errorf("quality_threshold must be between 0 and 1")
	}
	if c.MaxIterations < 0 {
		return fmt.Errorf("max_iterations must be non-negative")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for TerminationConditions
func (c *TerminationConditions) SetDefaults() {
	if c.MaxDuration == 0 {
		c.MaxDuration = 30 * time.Minute
	}
	if c.QualityThreshold == 0 {
		c.QualityThreshold = 0.8
	}
	if c.MaxIterations == 0 {
		c.MaxIterations = 10
	}
}

// WorkflowSettings represents workflow settings
type WorkflowSettings struct {
	MaxConcurrency int           `yaml:"max_concurrency"` // Max concurrent executions
	Timeout        time.Duration `yaml:"timeout"`         // Workflow timeout
	RetryPolicy    RetryPolicy   `yaml:"retry_policy"`    // Retry policy
	ErrorPolicy    string        `yaml:"error_policy"`    // Error handling policy
}

// Validate implements Config.Validate for WorkflowSettings
func (c *WorkflowSettings) Validate() error {
	if c.MaxConcurrency <= 0 {
		return fmt.Errorf("max_concurrency must be positive")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if err := c.RetryPolicy.Validate(); err != nil {
		return fmt.Errorf("retry policy validation failed: %w", err)
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for WorkflowSettings
func (c *WorkflowSettings) SetDefaults() {
	if c.MaxConcurrency == 0 {
		c.MaxConcurrency = 4
	}
	if c.Timeout == 0 {
		c.Timeout = 15 * time.Minute
	}
	if c.ErrorPolicy == "" {
		c.ErrorPolicy = "fail"
	}
	c.RetryPolicy.SetDefaults()
}

// RetryPolicy represents retry policy
type RetryPolicy struct {
	MaxRetries int           `yaml:"max_retries"` // Max retry attempts
	Backoff    time.Duration `yaml:"backoff"`     // Backoff duration
}

// Validate implements Config.Validate for RetryPolicy
func (c *RetryPolicy) Validate() error {
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}
	if c.Backoff < 0 {
		return fmt.Errorf("backoff must be non-negative")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for RetryPolicy
func (c *RetryPolicy) SetDefaults() {
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
	if c.Backoff == 0 {
		c.Backoff = 5 * time.Second
	}
}

// ============================================================================
// TOOL CONFIGURATIONS
// ============================================================================

// CommandToolsConfig represents command tool configuration
type CommandToolsConfig struct {
	AllowedCommands  []string      `yaml:"allowed_commands"`
	WorkingDirectory string        `yaml:"working_directory"`
	MaxExecutionTime time.Duration `yaml:"max_execution_time"`
	EnableSandboxing bool          `yaml:"enable_sandboxing"`
}

// Validate implements Config.Validate for CommandToolsConfig
func (c *CommandToolsConfig) Validate() error {
	if len(c.AllowedCommands) == 0 {
		return fmt.Errorf("at least one allowed command is required")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for CommandToolsConfig
func (c *CommandToolsConfig) SetDefaults() {
	if len(c.AllowedCommands) == 0 {
		c.AllowedCommands = []string{
			"cat", "head", "tail", "ls", "find", "grep", "wc", "pwd",
			"git", "npm", "go", "curl", "wget", "echo", "date",
		}
	}
	if c.WorkingDirectory == "" {
		c.WorkingDirectory = "./"
	}
	if c.MaxExecutionTime == 0 {
		c.MaxExecutionTime = 30 * time.Second
	}
}

// SearchToolConfig represents search tool configuration
type SearchToolConfig struct {
	DocumentStores     []string `yaml:"document_stores"`
	DefaultLimit       int      `yaml:"default_limit"`
	MaxLimit           int      `yaml:"max_limit"`
	MaxResults         int      `yaml:"max_results"`
	EnabledSearchTypes []string `yaml:"enabled_search_types"`
}

// Validate implements Config.Validate for SearchToolConfig
func (c *SearchToolConfig) Validate() error {
	if c.DefaultLimit <= 0 {
		return fmt.Errorf("default_limit must be positive")
	}
	if c.MaxResults <= 0 {
		return fmt.Errorf("max_results must be positive")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for SearchToolConfig
func (c *SearchToolConfig) SetDefaults() {
	if c.DefaultLimit == 0 {
		c.DefaultLimit = 10
	}
	if c.MaxLimit == 0 {
		c.MaxLimit = 50
	}
	if c.MaxResults == 0 {
		c.MaxResults = 100
	}
	if len(c.EnabledSearchTypes) == 0 {
		c.EnabledSearchTypes = []string{"content", "file", "function", "struct"}
	}
}

// ToolConfigs represents tool configurations
type ToolConfigs struct {
	DefaultRepo  string           `yaml:"default_repo,omitempty"` // Default repository
	Repositories []ToolRepository `yaml:"repositories,omitempty"` // Tool repositories
}

// Validate implements Config.Validate for ToolConfigs
func (c *ToolConfigs) Validate() error {
	repoNames := make(map[string]bool)
	for i, repo := range c.Repositories {
		if err := repo.Validate(); err != nil {
			return fmt.Errorf("repository %d validation failed: %w", i, err)
		}

		// Check for duplicate repository names
		if repoNames[repo.Name] {
			return fmt.Errorf("duplicate repository name: %s", repo.Name)
		}
		repoNames[repo.Name] = true
	}

	// Validate default repository
	if c.DefaultRepo != "" && !repoNames[c.DefaultRepo] {
		return fmt.Errorf("default_repo %s not found in repositories", c.DefaultRepo)
	}

	return nil
}

// SetDefaults implements Config.SetDefaults for ToolConfigs
func (c *ToolConfigs) SetDefaults() {
	// Zero-config: Create default local tool repository if none exist
	if len(c.Repositories) == 0 {
		c.DefaultRepo = "local"
		c.Repositories = []ToolRepository{
			{
				Name:        "local",
				Type:        "local",
				Description: "Built-in local tools",
				Tools: []ToolDefinition{
					{
						Name:    "execute_command",
						Type:    "command",
						Enabled: true,
						Config: map[string]interface{}{
							"command_config": map[string]interface{}{
								"allowed_commands":   []string{"ls", "cat", "head", "tail", "pwd", "find", "grep", "git", "curl", "wget", "echo", "date", "wc"},
								"working_directory":  "./",
								"max_execution_time": "30s",
								"enable_sandboxing":  true,
							},
						},
					},
					{
						Name:    "search",
						Type:    "search",
						Enabled: true,
						Config: map[string]interface{}{
							"search_config": map[string]interface{}{
								"document_stores":      []string{"default-docs"},
								"default_limit":        10,
								"max_limit":            50,
								"max_results":          100,
								"enabled_search_types": []string{"content", "file", "function"},
							},
						},
					},
				},
			},
		}
	}

	for i := range c.Repositories {
		c.Repositories[i].SetDefaults()
	}
}

// ToolRepository represents a tool repository
type ToolRepository struct {
	Name        string                 `yaml:"name"`        // Repository name
	Type        string                 `yaml:"type"`        // Repository type
	Description string                 `yaml:"description"` // Repository description
	Config      map[string]interface{} `yaml:"config"`      // Repository-specific config
	URL         string                 `yaml:"url"`         // Repository URL (for MCP)
	PluginPath  string                 `yaml:"plugin_path"` // Plugin path (for plugins)
	Tools       []ToolDefinition       `yaml:"tools"`       // Tool definitions
}

// Validate implements Config.Validate for ToolRepository
func (c *ToolRepository) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Type == "" {
		return fmt.Errorf("type is required")
	}

	// Validate repository type
	switch c.Type {
	case "local":
		return c.validateLocalRepository()
	case "mcp":
		return c.validateMCPRepository()
	case "plugin":
		return c.validatePluginRepository()
	default:
		return fmt.Errorf("unknown repository type: %s", c.Type)
	}
}

// SetDefaults implements Config.SetDefaults for ToolRepository
func (c *ToolRepository) SetDefaults() {
	for i := range c.Tools {
		c.Tools[i].SetDefaults()
	}
}

// validateLocalRepository validates local repository configuration
func (c *ToolRepository) validateLocalRepository() error {
	toolNames := make(map[string]bool)
	for i, tool := range c.Tools {
		if err := tool.Validate(); err != nil {
			return fmt.Errorf("tool %d validation failed: %w", i, err)
		}

		// Check for duplicate tool names
		if toolNames[tool.Name] {
			return fmt.Errorf("duplicate tool name: %s", tool.Name)
		}
		toolNames[tool.Name] = true
	}
	return nil
}

// validateMCPRepository validates MCP repository configuration
func (c *ToolRepository) validateMCPRepository() error {
	if c.URL == "" {
		return fmt.Errorf("url is required for MCP repository")
	}
	return nil
}

// validatePluginRepository validates plugin repository configuration
func (c *ToolRepository) validatePluginRepository() error {
	if c.PluginPath == "" {
		return fmt.Errorf("plugin_path is required for plugin repository")
	}
	return nil
}

// ToolDefinition represents a tool definition
type ToolDefinition struct {
	Name    string                 `yaml:"name"`    // Tool name
	Type    string                 `yaml:"type"`    // Tool type
	Enabled bool                   `yaml:"enabled"` // Tool enabled
	Config  map[string]interface{} `yaml:"config"`  // Tool-specific config
}

// Validate implements Config.Validate for ToolDefinition
func (c *ToolDefinition) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Type == "" {
		return fmt.Errorf("type is required")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for ToolDefinition
func (c *ToolDefinition) SetDefaults() {
	if !c.Enabled {
		c.Enabled = true
	}
}

// ============================================================================
// DOCUMENT STORE CONFIGURATIONS
// ============================================================================

// DocumentStoreConfig represents document store configuration
type DocumentStoreConfig struct {
	Name            string   `yaml:"name"`             // Store name
	Source          string   `yaml:"source"`           // Source type
	Path            string   `yaml:"path"`             // Source path
	IncludePatterns []string `yaml:"include_patterns"` // Include patterns
	ExcludePatterns []string `yaml:"exclude_patterns"` // Exclude patterns
	WatchChanges    bool     `yaml:"watch_changes"`    // Watch for changes
	MaxFileSize     int64    `yaml:"max_file_size"`    // Max file size in bytes
}

// Validate implements Config.Validate for DocumentStoreConfig
func (c *DocumentStoreConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Source == "" {
		return fmt.Errorf("source is required")
	}
	if c.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for DocumentStoreConfig
func (c *DocumentStoreConfig) SetDefaults() {
	// Zero-config: Set default name and source if not specified
	if c.Name == "" {
		c.Name = "default-docs"
	}
	if c.Source == "" {
		c.Source = "directory"
	}
	if c.Path == "" {
		c.Path = "./"
	}
	if len(c.IncludePatterns) == 0 {
		c.IncludePatterns = []string{"*.md", "*.txt", "*.go", "*.py", "*.js", "*.ts", "*.yaml", "*.yml"}
	}
	if len(c.ExcludePatterns) == 0 {
		c.ExcludePatterns = []string{"**/node_modules/**", "**/.git/**", "**/vendor/**", "**/__pycache__/**"}
	}
	if c.MaxFileSize == 0 {
		c.MaxFileSize = 10 * 1024 * 1024 // 10MB default
	}
	// Zero-config: Enable watching by default
	if !c.WatchChanges {
		c.WatchChanges = true
	}
}

// ============================================================================
// PROMPT CONFIGURATIONS
// ============================================================================

// PromptConfig represents prompt configuration
type PromptConfig struct {
	SystemPrompt     string            `yaml:"system_prompt"`      // System prompt
	Instructions     string            `yaml:"instructions"`       // Instructions
	FullTemplate     string            `yaml:"full_template"`      // Full template
	Template         string            `yaml:"template"`           // Template
	Variables        map[string]string `yaml:"variables"`          // Template variables
	IncludeContext   bool              `yaml:"include_context"`    // Include context
	IncludeHistory   bool              `yaml:"include_history"`    // Include history
	IncludeTools     bool              `yaml:"include_tools"`      // Include tools
	MaxContextLength int               `yaml:"max_context_length"` // Max context length
}

// Validate implements Config.Validate for PromptConfig
func (c *PromptConfig) Validate() error {
	if c.MaxContextLength < 0 {
		return fmt.Errorf("max_context_length must be non-negative")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for PromptConfig
func (c *PromptConfig) SetDefaults() {
	if c.SystemPrompt == "" {
		c.SystemPrompt = "You are a helpful AI assistant. Use available tools and context to provide accurate, helpful responses."
	}
	if c.MaxContextLength == 0 {
		c.MaxContextLength = 4000
	}
	// Zero-config: Enable useful features by default
	if !c.IncludeContext {
		c.IncludeContext = true
	}
	if !c.IncludeHistory {
		c.IncludeHistory = true
	}
	if !c.IncludeTools {
		c.IncludeTools = true
	}
}

// ============================================================================
// REASONING CONFIGURATIONS
// ============================================================================

// ReasoningConfig represents reasoning configuration
type ReasoningConfig struct {
	Engine               string  `yaml:"engine"`                 // Reasoning engine
	MaxIterations        int     `yaml:"max_iterations"`         // Max iterations
	EnableSelfReflection bool    `yaml:"enable_self_reflection"` // Enable self-reflection
	EnableMetaReasoning  bool    `yaml:"enable_meta_reasoning"`  // Enable meta-reasoning
	EnableGoalEvolution  bool    `yaml:"enable_goal_evolution"`  // Enable goal evolution
	EnableDynamicTools   bool    `yaml:"enable_dynamic_tools"`   // Enable dynamic tools
	ShowDebugInfo        bool    `yaml:"show_debug_info"`        // Show debug info
	ShowThinking         bool    `yaml:"show_thinking"`          // Show internal reasoning in grayed-out format (Claude-style)
	EnableStreaming      bool    `yaml:"enable_streaming"`       // Enable streaming
	QualityThreshold     float64 `yaml:"quality_threshold"`      // Quality threshold
}

// Validate implements Config.Validate for ReasoningConfig
func (c *ReasoningConfig) Validate() error {
	if c.Engine == "" {
		return fmt.Errorf("engine is required")
	}
	if c.MaxIterations <= 0 {
		return fmt.Errorf("max_iterations must be positive")
	}
	if c.QualityThreshold < 0 || c.QualityThreshold > 1 {
		return fmt.Errorf("quality_threshold must be between 0 and 1")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for ReasoningConfig
func (c *ReasoningConfig) SetDefaults() {
	if c.Engine == "" {
		c.Engine = "default" // Simple, fast reasoning for zero-config
	}
	if c.MaxIterations == 0 {
		c.MaxIterations = 3
	}
	if c.QualityThreshold == 0 {
		c.QualityThreshold = 0.7
	}
	// Note: EnableStreaming defaults to false, only set to true if not explicitly configured
	// This allows the YAML config to control the streaming behavior
}

// ============================================================================
// SEARCH CONFIGURATIONS
// ============================================================================

// SearchConfig represents search configuration
type SearchConfig struct {
	Models           []SearchModel `yaml:"models"`             // Search models
	TopK             int           `yaml:"top_k"`              // Top K results
	Threshold        float64       `yaml:"threshold"`          // Similarity threshold
	MaxContextLength int           `yaml:"max_context_length"` // Max context length
}

// Validate implements Config.Validate for SearchConfig
func (c *SearchConfig) Validate() error {
	if len(c.Models) == 0 {
		return fmt.Errorf("at least one search model is required")
	}
	for i, model := range c.Models {
		if err := model.Validate(); err != nil {
			return fmt.Errorf("search model %d validation failed: %w", i, err)
		}
	}
	if c.TopK <= 0 {
		return fmt.Errorf("top_k must be positive")
	}
	if c.Threshold < 0 || c.Threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1")
	}
	if c.MaxContextLength < 0 {
		return fmt.Errorf("max_context_length must be non-negative")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for SearchConfig
func (c *SearchConfig) SetDefaults() {
	// Zero-config: Create default search model if none exist
	if len(c.Models) == 0 {
		c.Models = []SearchModel{
			{
				Name:        "documents",
				Collection:  "docs",
				DefaultTopK: 5,
				MaxTopK:     20,
			},
		}
	}
	if c.TopK == 0 {
		c.TopK = 5
	}
	if c.Threshold == 0 {
		c.Threshold = 0.7
	}
	if c.MaxContextLength == 0 {
		c.MaxContextLength = 4000
	}
	for i := range c.Models {
		c.Models[i].SetDefaults()
	}
}

// SearchModel represents a search model configuration
type SearchModel struct {
	Name        string `yaml:"name"`          // Model name
	Collection  string `yaml:"collection"`    // Collection name for vector storage
	DefaultTopK int    `yaml:"default_top_k"` // Default top K results
	MaxTopK     int    `yaml:"max_top_k"`     // Maximum top K results
}

// Validate implements Config.Validate for SearchModel
func (c *SearchModel) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Collection == "" {
		return fmt.Errorf("collection is required")
	}
	if c.DefaultTopK <= 0 {
		return fmt.Errorf("default_top_k must be positive")
	}
	if c.MaxTopK <= 0 {
		return fmt.Errorf("max_top_k must be positive")
	}
	if c.DefaultTopK > c.MaxTopK {
		return fmt.Errorf("default_top_k cannot be greater than max_top_k")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for SearchModel
func (c *SearchModel) SetDefaults() {
	if c.DefaultTopK == 0 {
		c.DefaultTopK = 10
	}
	if c.MaxTopK == 0 {
		c.MaxTopK = 100
	}
}

// ============================================================================
// GLOBAL CONFIGURATIONS
// ============================================================================

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`  // Log level
	Format string `yaml:"format"` // Log format
	Output string `yaml:"output"` // Output destination
}

// Validate implements Config.Validate for LoggingConfig
func (c *LoggingConfig) Validate() error {
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLevels[c.Level] {
		return fmt.Errorf("invalid log level: %s", c.Level)
	}
	validFormats := map[string]bool{
		"text": true, "json": true,
	}
	if !validFormats[c.Format] {
		return fmt.Errorf("invalid log format: %s", c.Format)
	}
	validOutputs := map[string]bool{
		"stdout": true, "stderr": true, "file": true,
	}
	if !validOutputs[c.Output] {
		return fmt.Errorf("invalid output destination: %s", c.Output)
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for LoggingConfig
func (c *LoggingConfig) SetDefaults() {
	if c.Level == "" {
		c.Level = "info"
	}
	if c.Format == "" {
		c.Format = "text"
	}
	if c.Output == "" {
		c.Output = "stdout"
	}
}

// PerformanceConfig represents performance configuration
type PerformanceConfig struct {
	MaxConcurrency int           `yaml:"max_concurrency"` // Max concurrency
	Timeout        time.Duration `yaml:"timeout"`         // Global timeout
}

// Validate implements Config.Validate for PerformanceConfig
func (c *PerformanceConfig) Validate() error {
	if c.MaxConcurrency <= 0 {
		return fmt.Errorf("max_concurrency must be positive")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for PerformanceConfig
func (c *PerformanceConfig) SetDefaults() {
	if c.MaxConcurrency == 0 {
		c.MaxConcurrency = 4
	}
	if c.Timeout == 0 {
		c.Timeout = 15 * time.Minute
	}
}
