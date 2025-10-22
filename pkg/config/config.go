// Package config provides configuration types and utilities for the AI agent framework.
// This file contains the main unified configuration entry point.
package config

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// Session store configurations (shared across agents)
	SessionStores map[string]SessionStoreConfig `yaml:"session_stores,omitempty"`

	// Plugin configurations
	Plugins PluginConfigs `yaml:"plugins,omitempty"`
}

// ValidateAgent checks if an agent exists in the configuration
// Returns error with list of available agents if not found
func (c *Config) ValidateAgent(agentID string) error {
	// For zero-config mode, we only have one default agent
	if len(c.Agents) == 0 {
		// Zero-config will create default agent, skip validation
		return nil
	}

	// Check if agent exists
	if _, exists := c.Agents[agentID]; !exists {
		// Build list of available agents
		availableAgents := make([]string, 0, len(c.Agents))
		for name := range c.Agents {
			availableAgents = append(availableAgents, name)
		}

		if len(availableAgents) == 0 {
			return fmt.Errorf("agent '%s' not found: no agents defined in configuration", agentID)
		}

		return fmt.Errorf("agent '%s' not found\n\nAvailable agents:\n  - %s",
			agentID, strings.Join(availableAgents, "\n  - "))
	}

	return nil
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

	// Validate session stores
	for name, store := range c.SessionStores {
		if err := store.Validate(); err != nil {
			return fmt.Errorf("session store '%s' validation failed: %w", name, err)
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
	if c.SessionStores == nil {
		c.SessionStores = make(map[string]SessionStoreConfig)
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

	// Expand agent shortcuts BEFORE setting agent defaults
	// This allows shortcuts to auto-create document stores, databases, embedders, and tools
	// NOTE: Validation of mutual exclusivity happens in AgentConfig.Validate() AFTER this expansion
	for agentID, agent := range c.Agents {
		// Shortcut 1: docs_folder -> auto-create document store (only if document_stores not explicitly set)
		if agent.DocsFolder != "" && len(agent.DocumentStores) == 0 {
			// Generate unique store name from path
			storeName := generateStoreNameFromPath(agent.DocsFolder)

			// Create document store config
			docStoreConfig := DocumentStoreConfig{
				Name:                storeName,
				Source:              "directory",
				Path:                agent.DocsFolder,
				WatchChanges:        true,
				MaxFileSize:         50 * 1024 * 1024, // 50MB
				IncrementalIndexing: true,
				// Use default include/exclude patterns (set in DocumentStoreConfig.SetDefaults)
			}
			c.DocumentStores[storeName] = docStoreConfig

			// Auto-assign document store to agent
			agent.DocumentStores = []string{storeName}

			// Clear the shortcut field after successful expansion (so validation doesn't see both)
			agent.DocsFolder = ""

			// Auto-create database and embedder if not already set
			if agent.Database == "" {
				if _, exists := c.Databases["default-database"]; !exists {
					c.Databases["default-database"] = DatabaseProviderConfig{}
				}
				agent.Database = "default-database"
			}
			if agent.Embedder == "" {
				if _, exists := c.Embedders["default-embedder"]; !exists {
					c.Embedders["default-embedder"] = EmbedderProviderConfig{}
				}
				agent.Embedder = "default-embedder"
			}

			// Auto-enable search tool for this agent if not already configured
			if c.Tools.Tools == nil {
				c.Tools.Tools = make(map[string]ToolConfig)
			}
			if _, exists := c.Tools.Tools["search"]; !exists {
				c.Tools.Tools["search"] = ToolConfig{
					Type:           "search",
					DocumentStores: []string{storeName},
				}
			}
		}

		// Shortcut 2: enable_tools -> auto-enable all local tools (only if tools not explicitly set)
		if agent.EnableTools && len(agent.Tools) == 0 {
			// nil means all tools available
			agent.Tools = nil

			// Clear the shortcut field after successful expansion (so validation doesn't see both)
			agent.EnableTools = false

			// Initialize tools config with default local tools if not already set
			if c.Tools.Tools == nil {
				c.Tools.Tools = make(map[string]ToolConfig)
			}

			// Ensure core local tools are configured
			toolDefaults := map[string]ToolConfig{
				"execute_command": {
					Type:             "command",
					Enabled:          true,
					WorkingDirectory: "./",
					MaxExecutionTime: "30s",
					EnableSandboxing: true, // Sandboxing enabled by default (allows all commands safely)
					AllowedCommands:  nil,  // nil = allow all (when sandboxing enabled)
				},
				"write_file": {
					Type:             "write_file",
					MaxFileSize:      1048576, // 1MB
					WorkingDirectory: "./",
				},
				"search_replace": {
					Type:             "search_replace",
					MaxReplacements:  100,
					WorkingDirectory: "./",
					BackupEnabled:    true,
				},
				"todo_write": {
					Type: "todo",
				},
			}

			// Add default tools if not already configured
			for toolName, toolConfig := range toolDefaults {
				if _, exists := c.Tools.Tools[toolName]; !exists {
					c.Tools.Tools[toolName] = toolConfig
				}
			}
		}

		// Update agent in map
		c.Agents[agentID] = agent
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

	// Set session store defaults
	for name := range c.SessionStores {
		store := c.SessionStores[name]
		store.SetDefaults()
		c.SessionStores[name] = store
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
// If the file doesn't exist, attempts to create zero-config from environment
func LoadConfig(filePath string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist - return zero-config default (may fail if no env vars set)
		return createZeroConfig()
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

// Default agent name for zero-config mode
const DefaultAgentName = "assistant"

// ZeroConfigOptions holds configuration options for zero-config mode
type ZeroConfigOptions struct {
	Provider    string // LLM provider: "openai", "anthropic", "gemini"
	APIKey      string // API key for the selected provider
	BaseURL     string // API base URL
	Model       string // Model name
	EnableTools bool   // Enable all local tools
	MCPURL      string // MCP server URL for tool integration
	DocsFolder  string // Document store folder path (RAG support)
	AgentName   string // Agent name/ID (defaults to DefaultAgentName)
}

// CreateZeroConfig creates a configuration for zero-config mode
// Supports openai, anthropic, and gemini providers
func CreateZeroConfig(opts ZeroConfigOptions) *Config {
	// Resolve API key from environment if not provided via flags
	if opts.APIKey == "" {
		if opts.Provider != "" {
			// Provider specified: look for matching API key
			switch opts.Provider {
			case "openai":
				opts.APIKey = os.Getenv("OPENAI_API_KEY")
			case "anthropic":
				opts.APIKey = os.Getenv("ANTHROPIC_API_KEY")
			case "gemini":
				opts.APIKey = os.Getenv("GEMINI_API_KEY")
			}
		} else {
			// Auto-detect provider from available API keys (priority: OpenAI → Anthropic → Gemini)
			if key := os.Getenv("OPENAI_API_KEY"); key != "" {
				opts.APIKey = key
				opts.Provider = "openai"
			} else if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
				opts.APIKey = key
				opts.Provider = "anthropic"
			} else if key := os.Getenv("GEMINI_API_KEY"); key != "" {
				opts.APIKey = key
				opts.Provider = "gemini"
			}
		}
	}

	// Resolve MCP URL from environment if not provided
	if opts.MCPURL == "" {
		opts.MCPURL = os.Getenv("MCP_URL")
	}

	// Default to OpenAI if provider not specified
	if opts.Provider == "" {
		opts.Provider = "openai"
	}

	// Default to DefaultAgentName if agent name not specified
	if opts.AgentName == "" {
		opts.AgentName = DefaultAgentName
	}

	// Note: API key validation happens in Config.Validate() via LLMProviderConfig.Validate()
	// This allows proper error handling through the runtime initialization flow

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
			// Use Claude Sonnet 4.5 (latest as of Feb 2025)
			// See: https://docs.anthropic.com/en/docs/about-claude/models/overview
			opts.Model = "claude-sonnet-4-5-20250929"
		}
	case "gemini":
		if opts.BaseURL == "" {
			// Note: Don't include /v1beta here, it's added by the Gemini provider
			opts.BaseURL = "https://generativelanguage.googleapis.com"
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
				APIKey:      opts.APIKey,
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
		// Generate a unique store name based on the source path to avoid conflicts
		storeName := generateStoreNameFromPath(opts.DocsFolder)

		// Create a default document store configuration for zero-config mode
		docStoreConfig := DocumentStoreConfig{
			Name:                storeName,
			Source:              "directory",
			Path:                opts.DocsFolder,
			IncludePatterns:     []string{"*.md", "*.txt", "*.go", "*.py", "*.js", "*.ts", "*.yaml", "*.yml", "*.json", "*.xml", "*.html", "*.csv", "*.pdf", "*.docx", "*.xlsx"},
			ExcludePatterns:     []string{"**/node_modules/**", "**/.git/**", "**/vendor/**", "**/__pycache__/**", "**/.DS_Store", "**/.hector/**", "**/index_state_*.json"},
			WatchChanges:        true,
			MaxFileSize:         50 * 1024 * 1024, // 50MB
			IncrementalIndexing: true,
		}

		// Add to document stores
		if cfg.DocumentStores == nil {
			cfg.DocumentStores = make(map[string]DocumentStoreConfig)
		}
		cfg.DocumentStores[storeName] = docStoreConfig

		// Enable search tool for the agent
		if cfg.Tools.Tools == nil {
			cfg.Tools.Tools = make(map[string]ToolConfig)
		}
		cfg.Tools.Tools["search"] = ToolConfig{
			Type:           "search",
			DocumentStores: []string{storeName},
		}
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
		// Don't override system prompt - let strategy use its optimized defaults
		// Don't auto-enable context inclusion - let users choose
		Prompt: PromptConfig{},
		Reasoning: ReasoningConfig{
			Engine:        "chain-of-thought",
			MaxIterations: 100,
		},
	}

	// Add document store to agent if docs folder provided
	if opts.DocsFolder != "" {
		storeName := generateStoreNameFromPath(opts.DocsFolder)
		agentConfig.DocumentStores = []string{storeName}
		agentConfig.Database = "default-database"
		agentConfig.Embedder = "default-embedder"

		// Create default database and embedder configurations
		cfg.Databases["default-database"] = DatabaseProviderConfig{
			Type:    "qdrant",
			Host:    "localhost",
			Port:    6334,
			Timeout: 30,
		}
		cfg.Embedders["default-embedder"] = EmbedderProviderConfig{
			Type:      "ollama",
			Model:     "nomic-embed-text",
			Host:      "http://localhost:11434",
			Dimension: 768,
			Timeout:   30,
		}
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
			Type:             "file_writer",
			Enabled:          true,
			MaxFileSize:      1048576, // 1MB
			WorkingDirectory: "./",
			// Note: AllowedExtensions intentionally not set = allow ALL file types (default permissive)
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
		// Enable MCP tools - nil means all tools available (including discovered MCP tools)
		agentConfig.Tools = nil
	} else {
		// No tools
		agentConfig.Tools = []string{}
	}

	cfg.Agents = map[string]AgentConfig{
		opts.AgentName: agentConfig,
	}

	// Configure MCP tool if URL provided
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
func createZeroConfig() (*Config, error) {
	cfg := CreateZeroConfig(ZeroConfigOptions{})
	return cfg, nil
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

// generateStoreNameFromPath creates a unique store name based on the source path
// This ensures different directories get different collections and index states
func generateStoreNameFromPath(sourcePath string) string {
	// Normalize the path to handle different path separators and trailing slashes
	normalizedPath := filepath.Clean(sourcePath)

	// Get the absolute path to ensure uniqueness
	absPath, err := filepath.Abs(normalizedPath)
	if err != nil {
		// Fallback to normalized path if absolute path fails
		absPath = normalizedPath
	}

	// Create a hash of the FULL path for uniqueness
	hash := md5.Sum([]byte(absPath))
	hashStr := hex.EncodeToString(hash[:])[:8] // Use first 8 characters for brevity

	// Get the directory name for readability
	dirName := filepath.Base(absPath)
	if dirName == "" || dirName == "." {
		dirName = "root"
	}

	// Clean the directory name to be safe for use as a collection name
	dirName = strings.ReplaceAll(dirName, " ", "_")
	dirName = strings.ReplaceAll(dirName, "-", "_")

	// Return a combination of directory name and hash for uniqueness
	// The hash is based on the FULL absolute path, so 8 chars is sufficient
	return fmt.Sprintf("docs_%s_%s", dirName, hashStr)
}
