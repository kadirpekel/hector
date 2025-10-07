// Package config provides configuration types and utilities for the AI agent framework.
// This file contains all configuration types in a unified structure.
package config

import (
	"fmt"
	"os"
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

// ============================================================================
// PLUGIN CONFIGURATIONS
// ============================================================================

// PluginDiscoveryConfig contains configuration for plugin discovery
type PluginDiscoveryConfig struct {
	Enabled            bool     `yaml:"enabled" json:"enabled"`
	Paths              []string `yaml:"paths" json:"paths"`
	ScanSubdirectories bool     `yaml:"scan_subdirectories" json:"scan_subdirectories"`
}

// SetDefaults sets default values for plugin discovery config
func (c *PluginDiscoveryConfig) SetDefaults() {
	if c.Paths == nil || len(c.Paths) == 0 {
		c.Paths = []string{"./plugins", "~/.hector/plugins"}
	}
	// Enabled defaults to true if not explicitly set (checked elsewhere)
}

// Validate validates the plugin discovery configuration
func (c *PluginDiscoveryConfig) Validate() error {
	return nil // No strict validation needed
}

// PluginConfig represents the configuration for a single plugin
type PluginConfig struct {
	Name    string                 `yaml:"name" json:"name"`
	Type    string                 `yaml:"type" json:"type"`       // Must be "grpc"
	Path    string                 `yaml:"path" json:"path"`       // Path to plugin executable
	Enabled bool                   `yaml:"enabled" json:"enabled"` // Whether plugin is enabled
	Config  map[string]interface{} `yaml:"config" json:"config"`   // Plugin-specific configuration
}

// SetDefaults sets default values for plugin config
func (c *PluginConfig) SetDefaults() {
	if c.Type == "" {
		c.Type = "grpc"
	}
}

// Validate validates the plugin configuration
func (c *PluginConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if c.Path == "" {
		return fmt.Errorf("plugin path is required")
	}
	if c.Type != "" && c.Type != "grpc" {
		return fmt.Errorf("invalid plugin type: %s (only 'grpc' is supported)", c.Type)
	}
	return nil
}

// PluginConfigs contains all plugin configurations
type PluginConfigs struct {
	// Plugin discovery configuration
	Discovery PluginDiscoveryConfig `yaml:"plugin_discovery,omitempty" json:"plugin_discovery,omitempty"`

	// Plugin definitions by category
	LLMProviders        map[string]PluginConfig `yaml:"llm_providers,omitempty" json:"llm_providers,omitempty"`
	DatabaseProviders   map[string]PluginConfig `yaml:"database_providers,omitempty" json:"database_providers,omitempty"`
	EmbedderProviders   map[string]PluginConfig `yaml:"embedder_providers,omitempty" json:"embedder_providers,omitempty"`
	ToolProviders       map[string]PluginConfig `yaml:"tool_providers,omitempty" json:"tool_providers,omitempty"`
	ReasoningStrategies map[string]PluginConfig `yaml:"reasoning_strategies,omitempty" json:"reasoning_strategies,omitempty"`
}

// SetDefaults sets default values for plugin configs
func (c *PluginConfigs) SetDefaults() {
	c.Discovery.SetDefaults()
	for name := range c.LLMProviders {
		cfg := c.LLMProviders[name]
		cfg.SetDefaults()
		c.LLMProviders[name] = cfg
	}
	for name := range c.DatabaseProviders {
		cfg := c.DatabaseProviders[name]
		cfg.SetDefaults()
		c.DatabaseProviders[name] = cfg
	}
	for name := range c.EmbedderProviders {
		cfg := c.EmbedderProviders[name]
		cfg.SetDefaults()
		c.EmbedderProviders[name] = cfg
	}
	for name := range c.ToolProviders {
		cfg := c.ToolProviders[name]
		cfg.SetDefaults()
		c.ToolProviders[name] = cfg
	}
	for name := range c.ReasoningStrategies {
		cfg := c.ReasoningStrategies[name]
		cfg.SetDefaults()
		c.ReasoningStrategies[name] = cfg
	}
}

// Validate validates all plugin configurations
func (c *PluginConfigs) Validate() error {
	if err := c.Discovery.Validate(); err != nil {
		return fmt.Errorf("plugin discovery validation failed: %w", err)
	}
	for name, cfg := range c.LLMProviders {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("LLM provider plugin '%s' validation failed: %w", name, err)
		}
	}
	for name, cfg := range c.DatabaseProviders {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("database provider plugin '%s' validation failed: %w", name, err)
		}
	}
	for name, cfg := range c.EmbedderProviders {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("embedder provider plugin '%s' validation failed: %w", name, err)
		}
	}
	for name, cfg := range c.ToolProviders {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("tool provider plugin '%s' validation failed: %w", name, err)
		}
	}
	for name, cfg := range c.ReasoningStrategies {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("reasoning strategy plugin '%s' validation failed: %w", name, err)
		}
	}
	return nil
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
	Type        string  `yaml:"type"`        // "ollama", "openai", "anthropic", "gemini"
	Model       string  `yaml:"model"`       // Model name
	APIKey      string  `yaml:"api_key"`     // API key (for OpenAI, Anthropic, Gemini)
	Host        string  `yaml:"host"`        // Host for ollama or custom OpenAI endpoint
	Temperature float64 `yaml:"temperature"` // Temperature setting
	MaxTokens   int     `yaml:"max_tokens"`  // Max tokens
	Timeout     int     `yaml:"timeout"`     // Request timeout in seconds
	MaxRetries  int     `yaml:"max_retries"` // Max retry attempts for rate limits (default: 5)
	RetryDelay  int     `yaml:"retry_delay"` // Base retry delay in seconds (default: 2, exponential backoff)

	// Structured output configuration (optional)
	StructuredOutput *StructuredOutputConfig `yaml:"structured_output,omitempty"`
}

// StructuredOutputConfig represents configuration for structured output
// Works across all providers (OpenAI, Anthropic, Gemini)
type StructuredOutputConfig struct {
	// Format: "json", "xml", "enum"
	Format string `yaml:"format,omitempty"`

	// Schema: JSON schema as YAML/JSON (for format="json")
	Schema map[string]interface{} `yaml:"schema,omitempty"`

	// Enum: List of allowed values (for format="enum")
	Enum []string `yaml:"enum,omitempty"`

	// Prefill: Prefill string for Anthropic (optional, provider-specific)
	Prefill string `yaml:"prefill,omitempty"`

	// PropertyOrdering: Property order for Gemini (optional, provider-specific)
	PropertyOrdering []string `yaml:"property_ordering,omitempty"`
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
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}
	if c.RetryDelay < 0 {
		return fmt.Errorf("retry_delay must be non-negative")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for LLMProviderConfig
func (c *LLMProviderConfig) SetDefaults() {
	// Zero-config: Set default type and model if not specified
	// Default to OpenAI (requires OPENAI_API_KEY environment variable)
	if c.Type == "" {
		c.Type = "openai"
	}
	if c.Model == "" {
		switch c.Type {
		case "openai":
			c.Model = "gpt-4o"
		case "anthropic":
			c.Model = "claude-3-7-sonnet-latest"
		case "gemini":
			c.Model = "gemini-2.0-flash-exp"
		default:
			c.Model = "gpt-4o"
		}
	}
	if c.Host == "" {
		// Set default host based on provider type
		switch c.Type {
		case "openai":
			c.Host = "https://api.openai.com/v1"
		case "anthropic":
			c.Host = "https://api.anthropic.com"
		case "gemini":
			c.Host = "https://generativelanguage.googleapis.com"
		default:
			c.Host = "https://api.openai.com/v1"
		}
	}
	if c.Temperature == 0 {
		c.Temperature = 0.7
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = 8000
	}
	if c.Timeout == 0 {
		c.Timeout = 60
	}
	if c.MaxRetries == 0 {
		// Aggressive retry strategy to support "trust the LLM" philosophy
		// With 5 retries and exponential backoff (2s, 4s, 8s, 16s, 32s):
		// - Total max wait: ~62 seconds
		// - Supports up to 100 iterations without premature failure
		c.MaxRetries = 5
	}
	if c.RetryDelay == 0 {
		// Base delay for exponential backoff (2^attempt * RetryDelay)
		c.RetryDelay = 2
	}
	if c.APIKey == "" {
		// Try to get API key from environment based on provider type
		// Note: Don't use "${VAR}" syntax here because SetDefaults runs AFTER env expansion
		switch c.Type {
		case "openai":
			if key := os.Getenv("OPENAI_API_KEY"); key != "" {
				c.APIKey = key
			}
		case "anthropic":
			if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
				c.APIKey = key
			}
		case "gemini":
			if key := os.Getenv("GEMINI_API_KEY"); key != "" {
				c.APIKey = key
			}
		}
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
	// Note: Embedders are optional and only needed for semantic search
	if c.Type == "" {
		c.Type = "ollama" // Ollama is fine for embedders (no function calling needed)
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
	// Agent type: "native" (default) or "a2a" (external)
	Type string `yaml:"type,omitempty"` // Agent type: "native" (default) or "a2a"

	// Common fields (both native and external)
	Name        string `yaml:"name"`        // Agent name
	Description string `yaml:"description"` // Agent description

	// Visibility control
	// - "public" (default): Discoverable via /agents and callable
	// - "internal": Not discoverable, but callable if you know the agent ID
	// - "private": Only callable by local orchestrators, not via external API
	Visibility string `yaml:"visibility,omitempty"` // Agent visibility: "public" (default), "internal", or "private"

	// External A2A agent fields (when type="a2a")
	URL string `yaml:"url,omitempty"` // A2A agent URL (e.g., "https://server.com/agents/specialist")

	// Native agent fields (when type="native" or omitted)
	LLM            string          `yaml:"llm,omitempty"`             // LLM provider reference
	Database       string          `yaml:"database,omitempty"`        // Database provider reference
	Embedder       string          `yaml:"embedder,omitempty"`        // Embedder provider reference
	DocumentStores []string        `yaml:"document_stores,omitempty"` // Document store references
	Prompt         PromptConfig    `yaml:"prompt,omitempty"`          // Prompt configuration
	Reasoning      ReasoningConfig `yaml:"reasoning,omitempty"`       // Reasoning configuration
	Search         SearchConfig    `yaml:"search,omitempty"`          // Search configuration
	Tools          []string        `yaml:"tools,omitempty"`           // Tool references (defined globally in tools: section)
}

// Validate implements Config.Validate for AgentConfig
func (c *AgentConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Normalize type
	if c.Type == "" {
		c.Type = "native"
	}

	// Normalize visibility
	if c.Visibility == "" {
		c.Visibility = "public"
	}

	// Validate visibility
	switch c.Visibility {
	case "public", "internal", "private":
		// Valid
	default:
		return fmt.Errorf("invalid visibility '%s' (must be 'public', 'internal', or 'private')", c.Visibility)
	}

	// Validate based on agent type
	switch c.Type {
	case "a2a":
		// External A2A agent - only URL is required
		if c.URL == "" {
			return fmt.Errorf("url is required for external A2A agents (type=a2a)")
		}
		// LLM, Database, etc. should not be specified for external agents
		if c.LLM != "" {
			return fmt.Errorf("llm should not be specified for external A2A agents (agent has its own LLM)")
		}

	case "native":
		// Native agent - LLM is required
		if c.LLM == "" {
			return fmt.Errorf("llm provider reference is required for native agents")
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
		// Validate native agent configs
		if err := c.Prompt.Validate(); err != nil {
			return fmt.Errorf("prompt configuration validation failed: %w", err)
		}
		if err := c.Reasoning.Validate(); err != nil {
			return fmt.Errorf("reasoning configuration validation failed: %w", err)
		}
		if err := c.Search.Validate(); err != nil {
			return fmt.Errorf("search configuration validation failed: %w", err)
		}

	default:
		return fmt.Errorf("invalid agent type '%s' (must be 'native' or 'a2a')", c.Type)
	}

	return nil
}

// SetDefaults implements Config.SetDefaults for AgentConfig
func (c *AgentConfig) SetDefaults() {
	// Default to native agent if not specified
	if c.Type == "" {
		c.Type = "native"
	}

	// Default visibility
	if c.Visibility == "" {
		c.Visibility = "public"
	}

	// Set defaults based on agent type
	switch c.Type {
	case "native":
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

	case "a2a":
		// External A2A agent - minimal defaults
		if c.Name == "" {
			c.Name = "External Agent"
		}
		if c.Description == "" {
			c.Description = "External A2A-compliant agent"
		}
		// No other defaults needed - external agent has its own config
	}
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

// FileWriterConfig represents file writer tool configuration
type FileWriterConfig struct {
	MaxFileSize       int      `yaml:"max_file_size"`
	AllowedExtensions []string `yaml:"allowed_extensions"`
	BackupOnOverwrite bool     `yaml:"backup_on_overwrite"`
	WorkingDirectory  string   `yaml:"working_directory"`
}

// Validate implements Config.Validate for FileWriterConfig
func (c *FileWriterConfig) Validate() error {
	if c.MaxFileSize < 0 {
		return fmt.Errorf("max_file_size must be non-negative")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for FileWriterConfig
func (c *FileWriterConfig) SetDefaults() {
	if c.MaxFileSize == 0 {
		c.MaxFileSize = 1048576 // 1MB default
	}
	if len(c.AllowedExtensions) == 0 {
		c.AllowedExtensions = []string{".go", ".yaml", ".md", ".json", ".txt", ".sh"}
	}
	if c.WorkingDirectory == "" {
		c.WorkingDirectory = "./"
	}
}

// SearchReplaceConfig represents search/replace tool configuration
type SearchReplaceConfig struct {
	MaxReplacements  int    `yaml:"max_replacements"`
	ShowDiff         bool   `yaml:"show_diff"`
	CreateBackup     bool   `yaml:"create_backup"`
	WorkingDirectory string `yaml:"working_directory"`
}

// Validate implements Config.Validate for SearchReplaceConfig
func (c *SearchReplaceConfig) Validate() error {
	if c.MaxReplacements < 0 {
		return fmt.Errorf("max_replacements must be non-negative")
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for SearchReplaceConfig
func (c *SearchReplaceConfig) SetDefaults() {
	if c.MaxReplacements == 0 {
		c.MaxReplacements = 100
	}
	if c.WorkingDirectory == "" {
		c.WorkingDirectory = "./"
	}
}

// ToolConfigs represents tool configuration
// The Tools field is marked with `,inline` to flatten the YAML structure
type ToolConfigs struct {
	Tools map[string]ToolConfig `yaml:",inline"` // Tool configurations (map for easy override, inline to avoid double-nesting)
}

// Validate implements Config.Validate for ToolConfigs
func (c *ToolConfigs) Validate() error {
	for name, tool := range c.Tools {
		if err := tool.Validate(); err != nil {
			return fmt.Errorf("tool '%s' validation failed: %w", name, err)
		}
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for ToolConfigs
func (c *ToolConfigs) SetDefaults() {
	// Initialize the tools map if it's nil
	if c.Tools == nil {
		c.Tools = make(map[string]ToolConfig)
	}

	// Zero-config: Create default safe tools (Tier 1) ONLY if no tools are configured
	// For file editing tools (file_writer, search_replace), users must explicitly enable them
	if len(c.Tools) == 0 {
		c.Tools = map[string]ToolConfig{
			"execute_command": {
				Type:             "command",
				AllowedCommands:  []string{"ls", "cat", "head", "tail", "pwd", "find", "grep", "wc", "date", "echo", "tree", "du", "df"},
				WorkingDirectory: "./",
				MaxExecutionTime: "30s",
				EnableSandboxing: true,
			},
			"todo_write": {
				Type: "todo",
			},
			// NOTE: file_writer, search_replace are NOT included in safe defaults
			// Users must explicitly enable them via configuration for security
		}
	}

	// Set defaults for each tool
	for name, tool := range c.Tools {
		tool.SetDefaults()
		c.Tools[name] = tool
	}
}

// ToolConfig represents a single tool configuration
type ToolConfig struct {
	Type        string `yaml:"type"`                  // Tool type: "command", "file_writer", "search_replace", "todo", etc.
	Enabled     bool   `yaml:"enabled,omitempty"`     // Tool enabled (default: true)
	Description string `yaml:"description,omitempty"` // Tool description

	// Command tool fields
	AllowedCommands  []string `yaml:"allowed_commands,omitempty"`   // Allowed commands
	WorkingDirectory string   `yaml:"working_directory,omitempty"`  // Working directory
	MaxExecutionTime string   `yaml:"max_execution_time,omitempty"` // Max execution time
	EnableSandboxing bool     `yaml:"enable_sandboxing,omitempty"`  // Enable sandboxing

	// File writer tool fields
	MaxFileSize       int64    `yaml:"max_file_size,omitempty"`      // Max file size in bytes
	AllowedExtensions []string `yaml:"allowed_extensions,omitempty"` // Allowed file extensions
	ForbiddenPaths    []string `yaml:"forbidden_paths,omitempty"`    // Forbidden paths

	// Search replace tool fields
	MaxReplacements int  `yaml:"max_replacements,omitempty"` // Max replacements per operation
	BackupEnabled   bool `yaml:"backup_enabled,omitempty"`   // Enable file backups

	// Search tool fields
	DocumentStores     []string `yaml:"document_stores,omitempty"`      // Document stores to search
	DefaultLimit       int      `yaml:"default_limit,omitempty"`        // Default result limit
	MaxLimit           int      `yaml:"max_limit,omitempty"`            // Maximum result limit
	MaxResults         int      `yaml:"max_results,omitempty"`          // Maximum total results
	EnabledSearchTypes []string `yaml:"enabled_search_types,omitempty"` // Enabled search types

	// MCP tool fields
	ServerURL string `yaml:"server_url,omitempty"` // MCP server URL

	// Generic config for extensibility (for custom/future tools)
	Config map[string]interface{} `yaml:"config,omitempty"` // Additional tool-specific config
}

// Validate implements Config.Validate for ToolConfig
func (c *ToolConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("type is required")
	}

	// Type-specific validation
	switch c.Type {
	case "command":
		if len(c.AllowedCommands) == 0 {
			return fmt.Errorf("allowed_commands is required for command tool")
		}
	case "file_writer":
		// Optional validation
	case "search_replace":
		// Optional validation
	case "search":
		if len(c.DocumentStores) == 0 {
			return fmt.Errorf("document_stores is required for search tool")
		}
	case "todo":
		// No specific validation needed
	default:
		// Allow unknown types for extensibility
	}

	return nil
}

// SetDefaults implements Config.SetDefaults for ToolConfig
func (c *ToolConfig) SetDefaults() {
	// Enabled is explicitly false, so check for zero value and set to true
	// Note: Go's zero value for bool is false, so we can't distinguish between
	// unset (default) and explicitly set to false in the YAML
	// We'll assume if it's false, it was unset (unless explicitly "enabled: false")
	// For now, we'll keep it simple: tools are enabled by default
	c.Enabled = true

	// Type-specific defaults
	switch c.Type {
	case "command":
		if c.WorkingDirectory == "" {
			c.WorkingDirectory = "./"
		}
		if c.MaxExecutionTime == "" {
			c.MaxExecutionTime = "30s"
		}
	case "file_writer":
		if c.MaxFileSize == 0 {
			c.MaxFileSize = 1048576 // 1MB
		}
	case "search_replace":
		if c.MaxReplacements == 0 {
			c.MaxReplacements = 100
		}
		if c.WorkingDirectory == "" {
			c.WorkingDirectory = "./"
		}
	case "search":
		if c.DefaultLimit == 0 {
			c.DefaultLimit = 10
		}
		if c.MaxLimit == 0 {
			c.MaxLimit = 50
		}
		if c.MaxResults == 0 {
			c.MaxResults = 100
		}
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
	// Slot-based customization (preferred)
	// Note: We use map[string]string for YAML compatibility
	// Agent will convert to reasoning.PromptSlots
	PromptSlots map[string]string `yaml:"prompt_slots"` // Override strategy's prompt slots

	// Alternative: Full prompt override
	SystemPrompt        string            `yaml:"system_prompt"`        // Full system prompt override (bypasses slots)
	Instructions        string            `yaml:"instructions"`         // Instructions
	FullTemplate        string            `yaml:"full_template"`        // Full template
	Template            string            `yaml:"template"`             // Template
	Variables           map[string]string `yaml:"variables"`            // Template variables
	IncludeContext      bool              `yaml:"include_context"`      // Include context
	IncludeHistory      bool              `yaml:"include_history"`      // Include history
	MaxHistoryMessages  int               `yaml:"max_history_messages"` // Max history messages to include (default: 10)
	EnableSummarization bool              `yaml:"enable_summarization"` // Enable LLM-based history summarization
	SummarizeThreshold  float64           `yaml:"summarize_threshold"`  // Percentage (0.0-1.0) to trigger summarization (default: 0.8)
	IncludeTools        bool              `yaml:"include_tools"`        // Include tools
	MaxContextLength    int               `yaml:"max_context_length"`   // Max context length
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
	// DO NOT set a default SystemPrompt - leave it empty to allow slot-based prompts
	// If both SystemPrompt and prompt_slots are empty, strategies will provide default slots

	if c.MaxContextLength == 0 {
		c.MaxContextLength = 4000
	}
	// NOTE: Boolean flags (IncludeContext, IncludeHistory, IncludeTools) default to false
	// Users must explicitly enable them in YAML if desired
	// This allows explicit false values to work correctly
}

// ============================================================================
// REASONING CONFIGURATIONS
// ============================================================================

// ReasoningConfig represents reasoning configuration
type ReasoningConfig struct {
	Engine                       string  `yaml:"engine"`                         // Reasoning engine
	MaxIterations                int     `yaml:"max_iterations"`                 // Max iterations (safety valve, default: 100)
	EnableSelfReflection         bool    `yaml:"enable_self_reflection"`         // Enable self-reflection
	EnableMetaReasoning          bool    `yaml:"enable_meta_reasoning"`          // Enable meta-reasoning
	EnableGoalEvolution          bool    `yaml:"enable_goal_evolution"`          // Enable goal evolution
	EnableDynamicTools           bool    `yaml:"enable_dynamic_tools"`           // Enable dynamic tools
	EnableStructuredReflection   bool    `yaml:"enable_structured_reflection"`   // Enable LLM-based structured reflection (vs heuristics)
	EnableCompletionVerification bool    `yaml:"enable_completion_verification"` // Enable LLM-based task completion verification
	EnableGoalExtraction         bool    `yaml:"enable_goal_extraction"`         // Enable LLM-based goal extraction (supervisor strategy)
	ShowDebugInfo                bool    `yaml:"show_debug_info"`                // Show debug info (iteration counts, tokens, etc.)
	ShowToolExecution            bool    `yaml:"show_tool_execution"`            // Show tool execution labels (enabled by default for better UX)
	ShowThinking                 bool    `yaml:"show_thinking"`                  // Show internal reasoning in grayed-out format (Claude-style)
	EnableStreaming              bool    `yaml:"enable_streaming"`               // Enable streaming
	QualityThreshold             float64 `yaml:"quality_threshold"`              // Quality threshold
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
		// High limit as safety valve only - trust the LLM to naturally terminate
		// (matches Cursor's philosophy: loop until no more tool calls)
		c.MaxIterations = 100
	}
	if c.QualityThreshold == 0 {
		c.QualityThreshold = 0.7
	}
	// EnableStreaming defaults to true for better UX in zero-config mode
	// ShowToolExecution defaults to true - tool execution should be visible, not debug info
	// EnableStructuredReflection defaults to true for better quality (+13% quality, +20% cost)
	// Note: Go's zero value for bool is false, so we need to explicitly set it
	// In YAML configs, users can explicitly set these to false if needed
	c.EnableStreaming = true
	c.ShowToolExecution = true
	c.EnableStructuredReflection = true
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
