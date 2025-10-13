// Package config provides configuration types and utilities for the AI agent framework.
// This file contains the main unified configuration entry point.
package config

import (
	"fmt"
	"os"
)

// ============================================================================
// MAIN UNIFIED CONFIGURATION
// ============================================================================

// Config represents the complete configuration
// Similar to docker-compose.yml, this is the single entry point for all configuration
type Config struct {
	// Version and metadata
	Version     string            `yaml:"version,omitempty"`
	Name        string            `yaml:"name,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`

	// Global settings
	Global GlobalSettings `yaml:"global,omitempty"`

	// Service configurations (direct access, no providers wrapper)
	LLMs      map[string]LLMProviderConfig      `yaml:"llms,omitempty"`
	Databases map[string]DatabaseProviderConfig `yaml:"databases,omitempty"`
	Embedders map[string]EmbedderProviderConfig `yaml:"embedders,omitempty"`

	// Agent definitions
	Agents map[string]AgentConfig `yaml:"agents,omitempty"`

	// Tool configurations
	Tools ToolConfigs `yaml:"tools,omitempty"`

	// Document store configurations
	DocumentStores map[string]DocumentStoreConfig `yaml:"document_stores,omitempty"`

	// Session store configurations (shared across agents) - REMOVED for now
	// SessionStores map[string]SessionStoreConfig `yaml:"session_stores,omitempty"`

	// Plugin configurations
	Plugins PluginConfigs `yaml:"plugins,omitempty"`
}

// Validate implements Config.Validate for Config
func (c *Config) Validate() error {
	// Validate global settings
	if err := c.Global.Validate(); err != nil {
		return fmt.Errorf("global settings validation failed: %w", err)
	}

	// Validate LLMs
	for name, llm := range c.LLMs {
		if err := llm.Validate(); err != nil {
			return fmt.Errorf("LLM '%s' validation failed: %w", name, err)
		}
	}

	// Validate databases
	for name, db := range c.Databases {
		if err := db.Validate(); err != nil {
			return fmt.Errorf("database '%s' validation failed: %w", name, err)
		}
	}

	// Validate embedders
	for name, embedder := range c.Embedders {
		if err := embedder.Validate(); err != nil {
			return fmt.Errorf("embedder '%s' validation failed: %w", name, err)
		}
	}

	// Validate agents
	for name, agent := range c.Agents {
		if err := agent.Validate(); err != nil {
			return fmt.Errorf("agent '%s' validation failed: %w", name, err)
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

	// Validate plugins
	if err := c.Plugins.Validate(); err != nil {
		return fmt.Errorf("plugins validation failed: %w", err)
	}

	return nil
}

// SetDefaults implements Config.SetDefaults for Config
func (c *Config) SetDefaults() {
	// Set global defaults
	c.Global.SetDefaults()

	// Initialize maps if nil
	if c.LLMs == nil {
		c.LLMs = make(map[string]LLMProviderConfig)
	}
	if c.Databases == nil {
		c.Databases = make(map[string]DatabaseProviderConfig)
	}
	if c.Embedders == nil {
		c.Embedders = make(map[string]EmbedderProviderConfig)
	}
	if c.Agents == nil {
		c.Agents = make(map[string]AgentConfig)
	}
	if c.DocumentStores == nil {
		c.DocumentStores = make(map[string]DocumentStoreConfig)
	}

	// Zero-config: Create default services if none exist
	if len(c.LLMs) == 0 {
		c.LLMs["default-llm"] = LLMProviderConfig{}
	}
	if len(c.Databases) == 0 {
		c.Databases["default-database"] = DatabaseProviderConfig{}
	}
	if len(c.Embedders) == 0 {
		c.Embedders["default-embedder"] = EmbedderProviderConfig{}
	}
	// Document stores must be explicitly configured - no defaults

	// Zero-config: Create default agent if none exist
	if len(c.Agents) == 0 {
		c.Agents["default-agent"] = AgentConfig{}
	}

	// Set LLM defaults (now handles zero-config)
	for name := range c.LLMs {
		llm := c.LLMs[name]
		llm.SetDefaults()
		c.LLMs[name] = llm
	}

	// Set database defaults (now handles zero-config)
	for name := range c.Databases {
		db := c.Databases[name]
		db.SetDefaults()
		c.Databases[name] = db
	}

	// Set embedder defaults (now handles zero-config)
	for name := range c.Embedders {
		embedder := c.Embedders[name]
		embedder.SetDefaults()
		c.Embedders[name] = embedder
	}

	// Set agent defaults (now handles zero-config)
	for name := range c.Agents {
		agent := c.Agents[name]
		agent.SetDefaults()
		c.Agents[name] = agent
	}

	// Set tool defaults
	c.Tools.SetDefaults()

	// Set document store defaults (now handles zero-config)
	for name := range c.DocumentStores {
		store := c.DocumentStores[name]
		store.SetDefaults()
		c.DocumentStores[name] = store
	}

	// Set plugin defaults
	c.Plugins.SetDefaults()
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

	// A2A Server configuration
	A2AServer A2AServerConfig `yaml:"a2a_server,omitempty"`

	// Authentication configuration
	Auth AuthConfig `yaml:"auth,omitempty"`
}

// Validate implements Config.Validate for GlobalSettings
func (c *GlobalSettings) Validate() error {
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config validation failed: %w", err)
	}
	if err := c.Performance.Validate(); err != nil {
		return fmt.Errorf("performance config validation failed: %w", err)
	}
	if err := c.A2AServer.Validate(); err != nil {
		return fmt.Errorf("A2A server config validation failed: %w", err)
	}
	if err := c.Auth.Validate(); err != nil {
		return fmt.Errorf("auth config validation failed: %w", err)
	}
	return nil
}

// SetDefaults implements Config.SetDefaults for GlobalSettings
func (c *GlobalSettings) SetDefaults() {
	c.Logging.SetDefaults()
	c.Performance.SetDefaults()
	c.A2AServer.SetDefaults()
	c.Auth.SetDefaults()
}

// ============================================================================
// A2A SERVER CONFIGURATION
// ============================================================================

// A2AServerConfig contains configuration for the A2A protocol server
// Presence of configuration implies enabled (no explicit enabled field needed)
type A2AServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	BaseURL string `yaml:"base_url,omitempty"` // Public URL for agent cards
}

// IsEnabled returns true if A2A server configuration is present and valid
func (c *A2AServerConfig) IsEnabled() bool {
	return c.Port > 0 || c.Host != ""
}

// Validate validates the A2A server configuration
func (c *A2AServerConfig) Validate() error {
	if c.IsEnabled() {
		if c.Port <= 0 || c.Port > 65535 {
			return fmt.Errorf("invalid port: %d", c.Port)
		}
	}
	return nil
}

// SetDefaults sets default values for A2A server configuration
func (c *A2AServerConfig) SetDefaults() {
	if c.Host == "" {
		c.Host = "0.0.0.0"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
}

// ============================================================================
// AUTHENTICATION CONFIGURATION
// ============================================================================

// AuthConfig contains authentication configuration
// Hector is a JWT consumer - it validates tokens from external auth providers
// Presence of configuration implies enabled (no explicit enabled field needed)
type AuthConfig struct {
	JWKSURL  string `yaml:"jwks_url"` // JWKS URL from auth provider (e.g., https://auth0.com/.well-known/jwks.json)
	Issuer   string `yaml:"issuer"`   // Expected token issuer (e.g., https://auth0.com/)
	Audience string `yaml:"audience"` // Expected token audience (e.g., "hector-api")
}

// IsEnabled returns true if authentication configuration is present and valid
func (c *AuthConfig) IsEnabled() bool {
	return c.JWKSURL != "" && c.Issuer != "" && c.Audience != ""
}

// Validate validates the authentication configuration
func (c *AuthConfig) Validate() error {
	if c.IsEnabled() {
		if c.JWKSURL == "" {
			return fmt.Errorf("jwks_url is required for authentication")
		}
		if c.Issuer == "" {
			return fmt.Errorf("issuer is required for authentication")
		}
		if c.Audience == "" {
			return fmt.Errorf("audience is required for authentication")
		}
	}
	return nil
}

// SetDefaults sets default values for auth configuration
func (c *AuthConfig) SetDefaults() {
	// No defaults - auth is opt-in
}

// ============================================================================
// CONFIGURATION LOADING
// ============================================================================

// LoadConfig loads the complete configuration from a YAML file
// This is the main entry point for configuration loading
// If the file doesn't exist, returns a zero-config default
func LoadConfig(filePath string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist - return zero-config default
		return createZeroConfig(), nil
	}

	var config Config
	if err := loadConfig(filePath, &config); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &config, nil
}

// LoadConfigFromString loads configuration from a YAML string
func LoadConfigFromString(yamlContent string) (*Config, error) {
	var config Config
	if err := loadConfigFromString(yamlContent, &config); err != nil {
		return nil, fmt.Errorf("failed to load config from string: %w", err)
	}
	return &config, nil
}

// ZeroConfigOptions holds configuration options for zero-config mode
type ZeroConfigOptions struct {
	Provider    string // LLM provider: "openai" (default), "anthropic", "gemini"
	APIKey      string // API key for the selected provider (required)
	BaseURL     string // API base URL (provider-specific defaults)
	Model       string // Model name (provider-specific defaults)
	EnableTools bool   // Enable all local tools
	MCPURL      string // MCP server URL for tool integration
	DocsFolder  string // Document store folder path (RAG support)
}

// CreateZeroConfig creates a configuration for zero-config mode with custom options
// Supports openai (default), anthropic, and gemini providers
// API key is REQUIRED - will be validated by caller
func CreateZeroConfig(opts ZeroConfigOptions) *Config {
	// Default to OpenAI if provider not specified
	if opts.Provider == "" {
		opts.Provider = "openai"
	}

	// Set provider-specific defaults
	switch opts.Provider {
	case "openai":
		if opts.BaseURL == "" {
			opts.BaseURL = "https://api.openai.com/v1"
		}
		if opts.Model == "" {
			opts.Model = "gpt-4o-mini"
		}
	case "anthropic":
		if opts.BaseURL == "" {
			opts.BaseURL = "https://api.anthropic.com"
		}
		if opts.Model == "" {
			opts.Model = "claude-3-5-sonnet-20241022"
		}
	case "gemini":
		if opts.BaseURL == "" {
			opts.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
		}
		if opts.Model == "" {
			opts.Model = "gemini-2.0-flash-exp"
		}
	default:
		// Unknown provider, use as-is with OpenAI defaults
		if opts.BaseURL == "" {
			opts.BaseURL = "https://api.openai.com/v1"
		}
		if opts.Model == "" {
			opts.Model = "gpt-4o-mini"
		}
	}

	cfg := &Config{
		Name: "Zero Config Mode",
		LLMs: map[string]LLMProviderConfig{
			opts.Provider: {
				Type:        opts.Provider,
				Model:       opts.Model,
				APIKey:      opts.APIKey, // Required - validated before this function is called
				Host:        opts.BaseURL,
				Temperature: 0.7,
				MaxTokens:   4096,
				Timeout:     120,
			},
		},
		Databases:      make(map[string]DatabaseProviderConfig),
		Embedders:      make(map[string]EmbedderProviderConfig),
		DocumentStores: make(map[string]DocumentStoreConfig),
	}

	// Configure document store if folder provided
	if opts.DocsFolder != "" {
		// Note: Document stores require additional setup
		// For now, we'll keep this simple and not auto-configure
		// Users should use hector.yaml for advanced RAG setups
		fmt.Println("⚠️  Warning: Document stores (--docs) require additional configuration.")
		fmt.Println("   For RAG support, please create a hector.yaml configuration file.")
	}

	// Configure agent
	providerName := opts.Provider
	if providerName == "" {
		providerName = "OpenAI"
	}

	agentConfig := AgentConfig{
		Name:        "assistant",
		Description: fmt.Sprintf("AI assistant powered by %s (%s)", providerName, opts.Model),
		Type:        "native",
		LLM:         opts.Provider,
		Visibility:  "public",
		Prompt: PromptConfig{
			SystemPrompt: "You are a helpful AI assistant. Be concise and accurate in your responses.",
		},
		Reasoning: ReasoningConfig{
			Engine:        "chain-of-thought",
			MaxIterations: 10,
		},
	}

	// Configure tools
	if opts.EnableTools {
		// Enable all local tools
		agentConfig.Tools = nil // nil means all tools available

		// Initialize tools config with local tool sources
		if cfg.Tools.Tools == nil {
			cfg.Tools.Tools = make(map[string]ToolConfig)
		}

		// Add default local tools with proper configurations
		cfg.Tools.Tools["command"] = ToolConfig{
			Type:    "command",
			Enabled: true,
			AllowedCommands: []string{
				"ls", "cat", "head", "tail", "pwd", "find", "grep", "wc",
				"date", "echo", "tree", "du", "df", "git", "npm", "go",
				"curl", "wget", "chmod", "mkdir", "rm", "mv", "cp",
			},
			WorkingDirectory: "./",
			MaxExecutionTime: "30s",
			EnableSandboxing: false, // Disable for easier development
		}

		cfg.Tools.Tools["file_writer"] = ToolConfig{
			Type:              "file_writer",
			Enabled:           true,
			MaxFileSize:       1048576, // 1MB
			AllowedExtensions: []string{".go", ".yaml", ".yml", ".md", ".json", ".txt", ".sh", ".py", ".js", ".ts"},
			WorkingDirectory:  "./",
		}

		cfg.Tools.Tools["search_replace"] = ToolConfig{
			Type:             "search_replace",
			Enabled:          true,
			MaxReplacements:  100,
			WorkingDirectory: "./",
			BackupEnabled:    true,
		}

		cfg.Tools.Tools["todo"] = ToolConfig{
			Type:    "todo",
			Enabled: true,
		}
	} else if opts.MCPURL != "" {
		// Enable only MCP tools
		agentConfig.Tools = []string{"mcp"}
	} else {
		// No tools
		agentConfig.Tools = []string{}
	}

	cfg.Agents = map[string]AgentConfig{
		"assistant": agentConfig,
	}

	// Configure MCP if URL provided
	if opts.MCPURL != "" {
		if cfg.Tools.Tools == nil {
			cfg.Tools.Tools = make(map[string]ToolConfig)
		}
		cfg.Tools.Tools["mcp"] = ToolConfig{
			Type:      "mcp",
			Enabled:   true,
			ServerURL: opts.MCPURL,
		}
	}

	// Set defaults
	cfg.SetDefaults()

	return cfg
}

// createZeroConfig creates a minimal default configuration (for backward compatibility)
func createZeroConfig() *Config {
	return CreateZeroConfig(ZeroConfigOptions{})
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// GetAgent returns an agent configuration by name
func (c *Config) GetAgent(name string) (*AgentConfig, bool) {
	agent, exists := c.Agents[name]
	return &agent, exists
}

// GetDocumentStore returns a document store configuration by name
func (c *Config) GetDocumentStore(name string) (*DocumentStoreConfig, bool) {
	store, exists := c.DocumentStores[name]
	return &store, exists
}

// ListAgents returns a list of all agent names
func (c *Config) ListAgents() []string {
	agents := make([]string, 0, len(c.Agents))
	for name := range c.Agents {
		agents = append(agents, name)
	}
	return agents
}

// ListDocumentStores returns a list of all document store names
func (c *Config) ListDocumentStores() []string {
	stores := make([]string, 0, len(c.DocumentStores))
	for name := range c.DocumentStores {
		stores = append(stores, name)
	}
	return stores
}
