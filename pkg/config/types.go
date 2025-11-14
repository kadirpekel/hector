package config

import (
	"fmt"
	"time"
)

const (
	DefaultOpenAIModel    = "gpt-4o-mini"
	DefaultAnthropicModel = "claude-3-7-sonnet-latest"
	DefaultGeminiModel    = "gemini-2.0-flash-exp"
)

// BoolPtr returns a pointer to the given bool value
func BoolPtr(b bool) *bool {
	return &b
}

// BoolValue returns the value of the bool pointer, or the default if nil
func BoolValue(b *bool, defaultValue bool) bool {
	if b == nil {
		return defaultValue
	}
	return *b
}

// SetBoolDefault sets the bool pointer to default if nil
func SetBoolDefault(b **bool, defaultValue bool) {
	if *b == nil {
		*b = BoolPtr(defaultValue)
	}
}

type PluginDiscoveryConfig struct {
	Enabled            *bool    `yaml:"enabled" json:"enabled"`
	Paths              []string `yaml:"paths" json:"paths"`
	ScanSubdirectories *bool    `yaml:"scan_subdirectories" json:"scan_subdirectories"`
}

func (c *PluginDiscoveryConfig) SetDefaults() {
	if len(c.Paths) == 0 {
		c.Paths = []string{"./plugins", "~/.hector/plugins"}
	}
	if c.Enabled == nil {
		c.Enabled = BoolPtr(false)
	}
	if c.ScanSubdirectories == nil {
		c.ScanSubdirectories = BoolPtr(false)
	}
}

func (c *PluginDiscoveryConfig) Validate() error {
	return nil
}

type PluginConfig struct {
	Name    string                 `yaml:"name" json:"name"`
	Type    string                 `yaml:"type" json:"type"`
	Path    string                 `yaml:"path" json:"path"`
	Enabled *bool                  `yaml:"enabled" json:"enabled"`
	Config  map[string]interface{} `yaml:"config" json:"config"`
}

func (c *PluginConfig) SetDefaults() {
	if c.Type == "" {
		c.Type = "grpc"
	}
	if c.Enabled == nil {
		c.Enabled = BoolPtr(true)
	}
}

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

type PluginConfigs struct {
	Discovery PluginDiscoveryConfig `yaml:"plugin_discovery,omitempty" json:"plugin_discovery,omitempty"`

	LLMProviders        map[string]*PluginConfig `yaml:"llm_providers,omitempty" json:"llm_providers,omitempty"`
	DatabaseProviders   map[string]*PluginConfig `yaml:"database_providers,omitempty" json:"database_providers,omitempty"`
	EmbedderProviders   map[string]*PluginConfig `yaml:"embedder_providers,omitempty" json:"embedder_providers,omitempty"`
	ToolProviders       map[string]*PluginConfig `yaml:"tool_providers,omitempty" json:"tool_providers,omitempty"`
	ReasoningStrategies map[string]*PluginConfig `yaml:"reasoning_strategies,omitempty" json:"reasoning_strategies,omitempty"`
}

func (c *PluginConfigs) SetDefaults() {
	c.Discovery.SetDefaults()

	if c.LLMProviders == nil {
		c.LLMProviders = make(map[string]*PluginConfig)
	}
	if c.DatabaseProviders == nil {
		c.DatabaseProviders = make(map[string]*PluginConfig)
	}
	if c.EmbedderProviders == nil {
		c.EmbedderProviders = make(map[string]*PluginConfig)
	}
	if c.ToolProviders == nil {
		c.ToolProviders = make(map[string]*PluginConfig)
	}
	if c.ReasoningStrategies == nil {
		c.ReasoningStrategies = make(map[string]*PluginConfig)
	}

	for name := range c.LLMProviders {
		if c.LLMProviders[name] != nil {
			c.LLMProviders[name].SetDefaults()
		}
	}
	for name := range c.DatabaseProviders {
		if c.DatabaseProviders[name] != nil {
			c.DatabaseProviders[name].SetDefaults()
		}
	}
	for name := range c.EmbedderProviders {
		if c.EmbedderProviders[name] != nil {
			c.EmbedderProviders[name].SetDefaults()
		}
	}
	for name := range c.ToolProviders {
		if c.ToolProviders[name] != nil {
			c.ToolProviders[name].SetDefaults()
		}
	}
	for name := range c.ReasoningStrategies {
		if c.ReasoningStrategies[name] != nil {
			c.ReasoningStrategies[name].SetDefaults()
		}
	}
}

func (c *PluginConfigs) Validate() error {
	if err := c.Discovery.Validate(); err != nil {
		return fmt.Errorf("plugin discovery validation failed: %w", err)
	}
	for name, cfg := range c.LLMProviders {
		if cfg != nil {
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("LLM provider plugin '%s' validation failed: %w", name, err)
			}
		}
	}
	for name, cfg := range c.DatabaseProviders {
		if cfg != nil {
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("database provider plugin '%s' validation failed: %w", name, err)
			}
		}
	}
	for name, cfg := range c.EmbedderProviders {
		if cfg != nil {
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("embedder provider plugin '%s' validation failed: %w", name, err)
			}
		}
	}
	for name, cfg := range c.ToolProviders {
		if cfg != nil {
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("tool provider plugin '%s' validation failed: %w", name, err)
			}
		}
	}
	for name, cfg := range c.ReasoningStrategies {
		if cfg != nil {
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("reasoning strategy plugin '%s' validation failed: %w", name, err)
			}
		}
	}
	return nil
}

type LLMProviderConfig struct {
	Type        string  `yaml:"type"`
	Model       string  `yaml:"model"`
	APIKey      string  `yaml:"api_key"`
	Host        string  `yaml:"host"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
	Timeout     int     `yaml:"timeout"`
	MaxRetries  int     `yaml:"max_retries"`
	RetryDelay  int     `yaml:"retry_delay"`

	StructuredOutput *StructuredOutputConfig `yaml:"structured_output,omitempty"`
}

type StructuredOutputConfig struct {
	Format string `yaml:"format,omitempty"`

	Schema map[string]interface{} `yaml:"schema,omitempty"`

	Enum []string `yaml:"enum,omitempty"`

	Prefill string `yaml:"prefill,omitempty"`

	PropertyOrdering []string `yaml:"property_ordering,omitempty"`
}

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

	if c.APIKey == "" {
		switch c.Type {
		case "openai":
			return fmt.Errorf("api_key is required for OpenAI")
		case "anthropic":
			return fmt.Errorf("api_key is required for Anthropic")
		case "gemini":
			return fmt.Errorf("api_key is required for Gemini")
		case "ollama":
			// Ollama doesn't require API key for local deployments
		}
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

func (c *LLMProviderConfig) SetDefaults() {

	if c.Type == "" {
		c.Type = "openai"
	}
	if c.Model == "" {
		switch c.Type {
		case "openai":
			c.Model = DefaultOpenAIModel
		case "anthropic":
			c.Model = DefaultAnthropicModel
		case "gemini":
			c.Model = DefaultGeminiModel
		case "ollama":
			c.Model = "llama3.2" // Default Ollama model
		default:
			c.Model = DefaultOpenAIModel
		}
	}
	if c.Host == "" {

		switch c.Type {
		case "openai":
			c.Host = "https://api.openai.com/v1"
		case "anthropic":
			c.Host = "https://api.anthropic.com"
		case "gemini":
			c.Host = "https://generativelanguage.googleapis.com"
		case "ollama":
			c.Host = "http://localhost:11434"
		default:
			c.Host = "https://api.openai.com/v1"
		}
	}
	// Only set defaults if temperature is 0 AND it wasn't explicitly set to 0
	// We use -0.000001 as a sentinel value to indicate "explicitly set to 0" from zero-config
	if c.Temperature == -0.000001 {
		c.Temperature = 0.0
	} else if c.Temperature == 0 {
		c.Temperature = 0.7
	}
	// Only set defaults if maxTokens is 0 AND it wasn't explicitly set to 0
	// We use -1 as a sentinel value to indicate "explicitly set to 0" from zero-config
	if c.MaxTokens == -1 {
		c.MaxTokens = 0
	} else if c.MaxTokens == 0 {
		c.MaxTokens = 8000
	}
	if c.Timeout == 0 {
		c.Timeout = 600 // Default: 10 minutes
	}
	if c.MaxRetries == 0 {

		c.MaxRetries = 5
	}
	if c.RetryDelay == 0 {

		c.RetryDelay = 2
	}

	if c.APIKey == "" {
		if key := GetProviderAPIKey(c.Type); key != "" {
			c.APIKey = key
		}
	}
}

// DatabaseProviderConfig holds configuration for database connections.
// Note: For operation timeouts, use database client-specific timeout settings.
type DatabaseProviderConfig struct {
	Type   string `yaml:"type"`
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	APIKey string `yaml:"api_key"`
	UseTLS *bool  `yaml:"use_tls"`
}

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
	return nil
}

func (c *DatabaseProviderConfig) SetDefaults() {

	if c.Type == "" {
		c.Type = "qdrant"
	}
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 6334
	}
	if c.UseTLS == nil {
		c.UseTLS = BoolPtr(false)
	}
}

type EmbedderProviderConfig struct {
	Type       string `yaml:"type"`
	Model      string `yaml:"model"`
	Host       string `yaml:"host"`
	APIKey     string `yaml:"api_key,omitempty"`
	Dimension  int    `yaml:"dimension"`
	Timeout    int    `yaml:"timeout"`
	MaxRetries int    `yaml:"max_retries"`
	BatchSize  int    `yaml:"batch_size,omitempty"`
}

func (c *EmbedderProviderConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("type is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}
	// Host is required for all embedder types (defaults are set in SetDefaults)
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

func (c *EmbedderProviderConfig) SetDefaults() {
	// Set type default if not specified
	if c.Type == "" {
		c.Type = "ollama"
	}

	// Set model defaults based on type if not specified
	if c.Model == "" {
		switch c.Type {
		case "ollama":
			c.Model = "nomic-embed-text"
		case "openai":
			c.Model = "text-embedding-3-small"
		case "cohere":
			c.Model = "embed-english-v3.0"
		default:
			c.Model = "nomic-embed-text" // Fallback to Ollama default
		}
	}

	// Set Host defaults based on type (all providers have explicit defaults)
	if c.Host == "" {
		switch c.Type {
		case "ollama":
			c.Host = "http://localhost:11434"
		case "openai":
			c.Host = "https://api.openai.com/v1"
		case "cohere":
			c.Host = "https://api.cohere.ai/v1"
		default:
			// Default to Ollama host for unknown types
			c.Host = "http://localhost:11434"
		}
	}

	// Set dimension defaults based on type and model if not specified
	if c.Dimension == 0 {
		switch c.Type {
		case "openai":
			switch c.Model {
			case "text-embedding-3-small":
				c.Dimension = 1536
			case "text-embedding-3-large":
				c.Dimension = 3072
			case "text-embedding-ada-002":
				c.Dimension = 1536
			default:
				c.Dimension = 1536 // Default for OpenAI
			}
		case "cohere":
			switch c.Model {
			case "embed-english-v3.0":
				c.Dimension = 1024
			case "embed-multilingual-v3.0":
				c.Dimension = 1024
			case "embed-english-light-v3.0":
				c.Dimension = 384
			case "embed-multilingual-light-v3.0":
				c.Dimension = 384
			default:
				c.Dimension = 1024 // Default for Cohere
			}
		case "ollama":
			c.Dimension = 768 // Default for Ollama
		default:
			c.Dimension = 768 // Fallback default
		}
	}

	// Set timeout default if not specified
	if c.Timeout == 0 {
		c.Timeout = 30
	}

	// Set max retries default if not specified
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}
}

type AgentConfig struct {
	Type string `yaml:"type,omitempty"`

	Name        string `yaml:"name"`
	Description string `yaml:"description"`

	Visibility string `yaml:"visibility,omitempty"`

	// External A2A agent configuration
	URL           string            `yaml:"url,omitempty"`             // Agent card URL or service base URL
	TargetAgentID string            `yaml:"target_agent_id,omitempty"` // Remote agent ID (if different from local config key)
	Credentials   *AgentCredentials `yaml:"credentials,omitempty"`

	LLM              string                  `yaml:"llm,omitempty"`
	Database         string                  `yaml:"database,omitempty"`
	Embedder         string                  `yaml:"embedder,omitempty"`
	DocumentStores   []string                `yaml:"document_stores,omitempty"`
	Prompt           PromptConfig            `yaml:"prompt,omitempty"`
	Memory           MemoryConfig            `yaml:"memory,omitempty"`
	Reasoning        ReasoningConfig         `yaml:"reasoning,omitempty"`
	Search           SearchConfig            `yaml:"search,omitempty"`
	Task             *TaskConfig             `yaml:"task,omitempty"`
	SessionStore     string                  `yaml:"session_store,omitempty"`
	Tools            []string                `yaml:"tools,omitempty"`
	SubAgents        []string                `yaml:"sub_agents,omitempty"`
	Security         *SecurityConfig         `yaml:"security,omitempty"`
	StructuredOutput *StructuredOutputConfig `yaml:"structured_output,omitempty"`

	DocsFolder  string `yaml:"docs_folder,omitempty"`
	EnableTools *bool  `yaml:"enable_tools,omitempty"`

	A2A *A2ACardConfig `yaml:"a2a,omitempty"`
}

func (c *AgentConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	if c.Type == "" {
		c.Type = "native"
	}

	if c.Visibility == "" {
		c.Visibility = "public"
	}

	switch c.Visibility {
	case "public", "internal", "private":

	default:
		return fmt.Errorf("invalid visibility '%s' (must be 'public', 'internal', or 'private')", c.Visibility)
	}

	if c.A2A != nil {
		if err := c.A2A.Validate(); err != nil {
			return fmt.Errorf("a2a configuration: %w", err)
		}
	}

	switch c.Type {
	case "a2a":

		if c.URL == "" {
			return fmt.Errorf("url is required for external A2A agents (type=a2a)")
		}

		if c.LLM != "" {
			return fmt.Errorf("llm should not be specified for external A2A agents (agent has its own LLM)")
		}

		if c.Credentials != nil {
			if err := c.Credentials.Validate(); err != nil {
				return fmt.Errorf("invalid credentials for external agent: %w", err)
			}
		}

	case "native":

		if c.LLM == "" {
			return fmt.Errorf("llm provider reference is required for native agents")
		}

		if c.DocsFolder != "" && len(c.DocumentStores) > 0 {
			return fmt.Errorf("docs_folder shortcut and document_stores are mutually exclusive (use one or the other)")
		}
		if c.EnableTools != nil && *c.EnableTools && len(c.Tools) > 0 {
			return fmt.Errorf("enable_tools shortcut and explicit tools list are mutually exclusive (use one or the other)")
		}

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
		if c.Task != nil {
			if err := c.Task.Validate(); err != nil {
				return fmt.Errorf("task configuration validation failed: %w", err)
			}
		}

		if c.Security != nil {
			if err := c.Security.Validate(); err != nil {
				return fmt.Errorf("security configuration validation failed: %w", err)
			}
		}

	default:
		return fmt.Errorf("invalid agent type '%s' (must be 'native' or 'a2a')", c.Type)
	}

	return nil
}

func (c *AgentConfig) SetDefaults() {

	if c.Type == "" {
		c.Type = "native"
	}

	if c.Visibility == "" {
		c.Visibility = "public"
	}

	switch c.Type {
	case "native":

		// NOTE: Name default is handled in Config.Validate() to use the agent ID
		// if not explicitly provided, ensuring URL-safe names
		if c.Description == "" {
			c.Description = "AI assistant with local tools and knowledge"
		}
		if c.LLM == "" {
			c.LLM = "default-llm"
		}

		c.Prompt.SetDefaults()
		c.Memory.SetDefaults()
		c.Reasoning.SetDefaults()
		c.Search.SetDefaults()
		if c.Task != nil {
			c.Task.SetDefaults()
		}

		if c.Security != nil {
			c.Security.SetDefaults()
		}

	case "a2a":

		if c.Name == "" {
			c.Name = "External Agent"
		}
		if c.Description == "" {
			c.Description = "External A2A-compliant agent"
		}

		if c.Credentials != nil {
			c.Credentials.SetDefaults()
		}
	}
}

type CommandToolsConfig struct {
	AllowedCommands  []string      `yaml:"allowed_commands"`
	WorkingDirectory string        `yaml:"working_directory"`
	MaxExecutionTime time.Duration `yaml:"max_execution_time"`
	EnableSandboxing *bool         `yaml:"enable_sandboxing"`
}

func (c *CommandToolsConfig) Validate() error {
	if c.EnableSandboxing == nil {
		c.EnableSandboxing = BoolPtr(true)
	}
	if !*c.EnableSandboxing && len(c.AllowedCommands) == 0 {
		return fmt.Errorf("allowed_commands is required when enable_sandboxing is false (security requirement)")
	}

	return nil
}

func (c *CommandToolsConfig) SetDefaults() {

	if c.WorkingDirectory == "" {
		c.WorkingDirectory = "./"
	}
	if c.MaxExecutionTime == 0 {
		c.MaxExecutionTime = 30 * time.Second
	}
	if c.EnableSandboxing == nil {
		c.EnableSandboxing = BoolPtr(true)
	}
}

type SearchToolConfig struct {
	DocumentStores     []string `yaml:"document_stores"`
	DefaultLimit       int      `yaml:"default_limit"`
	MaxLimit           int      `yaml:"max_limit"`
	EnabledSearchTypes []string `yaml:"enabled_search_types"`
}

func (c *SearchToolConfig) Validate() error {
	if c.DefaultLimit <= 0 {
		return fmt.Errorf("default_limit must be positive")
	}
	return nil
}

func (c *SearchToolConfig) SetDefaults() {
	if c.DefaultLimit == 0 {
		c.DefaultLimit = 10
	}
	if c.MaxLimit == 0 {
		c.MaxLimit = 50
	}
	if len(c.EnabledSearchTypes) == 0 {
		c.EnabledSearchTypes = []string{"content", "file", "function", "struct"}
	}
}

type FileWriterConfig struct {
	MaxFileSize       int      `yaml:"max_file_size"`
	AllowedExtensions []string `yaml:"allowed_extensions"`
	DeniedExtensions  []string `yaml:"denied_extensions"`
	BackupOnOverwrite *bool    `yaml:"backup_on_overwrite"`
	WorkingDirectory  string   `yaml:"working_directory"`
}

func (c *FileWriterConfig) Validate() error {
	if c.MaxFileSize < 0 {
		return fmt.Errorf("max_file_size must be non-negative")
	}
	return nil
}

func (c *FileWriterConfig) SetDefaults() {
	if c.MaxFileSize == 0 {
		c.MaxFileSize = 1048576
	}

	if c.WorkingDirectory == "" {
		c.WorkingDirectory = "./"
	}
	if c.BackupOnOverwrite == nil {
		c.BackupOnOverwrite = BoolPtr(false)
	}
}

type SearchReplaceConfig struct {
	MaxReplacements  int    `yaml:"max_replacements"`
	ShowDiff         *bool  `yaml:"show_diff"`
	CreateBackup     *bool  `yaml:"create_backup"`
	WorkingDirectory string `yaml:"working_directory"`
}

func (c *SearchReplaceConfig) Validate() error {
	if c.MaxReplacements < 0 {
		return fmt.Errorf("max_replacements must be non-negative")
	}
	return nil
}

func (c *SearchReplaceConfig) SetDefaults() {
	if c.MaxReplacements == 0 {
		c.MaxReplacements = 100
	}
	if c.WorkingDirectory == "" {
		c.WorkingDirectory = "./"
	}
	if c.ShowDiff == nil {
		c.ShowDiff = BoolPtr(false)
	}
	if c.CreateBackup == nil {
		c.CreateBackup = BoolPtr(false)
	}
}

type ReadFileConfig struct {
	MaxFileSize      int    `yaml:"max_file_size"`
	WorkingDirectory string `yaml:"working_directory"`
	ShowLineNumbers  *bool  `yaml:"show_line_numbers"`
}

func (c *ReadFileConfig) Validate() error {
	if c.MaxFileSize < 0 {
		return fmt.Errorf("max_file_size must be non-negative")
	}
	return nil
}

func (c *ReadFileConfig) SetDefaults() {
	if c.MaxFileSize == 0 {
		c.MaxFileSize = 10485760 // 10MB
	}
	if c.WorkingDirectory == "" {
		c.WorkingDirectory = "./"
	}
	if c.ShowLineNumbers == nil {
		c.ShowLineNumbers = BoolPtr(true)
	}
}

type ApplyPatchConfig struct {
	MaxFileSize      int    `yaml:"max_file_size"`
	CreateBackup     *bool  `yaml:"create_backup"`
	ContextLines     int    `yaml:"context_lines"`
	WorkingDirectory string `yaml:"working_directory"`
}

func (c *ApplyPatchConfig) Validate() error {
	if c.MaxFileSize < 0 {
		return fmt.Errorf("max_file_size must be non-negative")
	}
	if c.ContextLines < 0 {
		return fmt.Errorf("context_lines must be non-negative")
	}
	return nil
}

func (c *ApplyPatchConfig) SetDefaults() {
	if c.MaxFileSize == 0 {
		c.MaxFileSize = 10485760 // 10MB
	}
	if c.ContextLines == 0 {
		c.ContextLines = 3
	}
	if c.WorkingDirectory == "" {
		c.WorkingDirectory = "./"
	}
	if c.CreateBackup == nil {
		c.CreateBackup = BoolPtr(true)
	}
}

type GrepSearchConfig struct {
	MaxResults       int    `yaml:"max_results"`
	MaxFileSize      int    `yaml:"max_file_size"`
	WorkingDirectory string `yaml:"working_directory"`
	ContextLines     int    `yaml:"context_lines"`
}

func (c *GrepSearchConfig) Validate() error {
	if c.MaxResults < 0 {
		return fmt.Errorf("max_results must be non-negative")
	}
	if c.MaxFileSize < 0 {
		return fmt.Errorf("max_file_size must be non-negative")
	}
	if c.ContextLines < 0 {
		return fmt.Errorf("context_lines must be non-negative")
	}
	return nil
}

func (c *GrepSearchConfig) SetDefaults() {
	if c.MaxResults == 0 {
		c.MaxResults = 1000
	}
	if c.MaxFileSize == 0 {
		c.MaxFileSize = 10485760 // 10MB
	}
	if c.WorkingDirectory == "" {
		c.WorkingDirectory = "./"
	}
	if c.ContextLines == 0 {
		c.ContextLines = 2
	}
}

// ToolConfigs is kept for backwards compatibility but is no longer used directly in Config.
// Config.Tools is now a map[string]*ToolConfig directly.
type ToolConfigs struct {
	Tools map[string]*ToolConfig `yaml:"tools,omitempty"`
}

func GetDefaultToolConfigs() map[string]*ToolConfig {
	return map[string]*ToolConfig{
		"execute_command": {
			Type:             "command",
			WorkingDirectory: "./",
			MaxExecutionTime: "30s",
			EnableSandboxing: BoolPtr(true),
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
		},
		"read_file": {
			Type:             "read_file",
			MaxFileSize:      10485760, // 10MB
			WorkingDirectory: "./",
		},
		"apply_patch": {
			Type:             "apply_patch",
			MaxFileSize:      10485760, // 10MB
			WorkingDirectory: "./",
			ContextLines:     3,
		},
		"grep_search": {
			Type:             "grep_search",
			MaxResults:       1000,
			MaxFileSize:      10485760, // 10MB
			WorkingDirectory: "./",
			ContextLines:     2,
		},
		"web_request": {
			Type:            "web_request",
			Timeout:         "30s",
			MaxRetries:      3,
			MaxRequestSize:  10485760, // 10MB
			MaxResponseSize: 52428800, // 50MB
			AllowRedirects:  BoolPtr(true),
			MaxRedirects:    10,
			UserAgent:       "Hector-Agent/1.0",
		},
		"todo_write": {
			Type: "todo",
		},
		"agent_call": {
			Type: "agent_call",
		},
	}
}

func (c *ToolConfigs) Validate() error {
	for name, tool := range c.Tools {
		if tool != nil {
			if err := tool.Validate(); err != nil {
				return fmt.Errorf("tool '%s' validation failed: %w", name, err)
			}
		}
	}
	return nil
}

func (c *ToolConfigs) SetDefaults() {
	if c.Tools == nil {
		c.Tools = make(map[string]*ToolConfig)
	}

	if len(c.Tools) == 0 {
		c.Tools = GetDefaultToolConfigs()
	}

	for name := range c.Tools {
		if c.Tools[name] != nil {
			c.Tools[name].SetDefaults()
		}
	}
}

type ToolConfig struct {
	Type        string `yaml:"type"`
	Enabled     *bool  `yaml:"enabled,omitempty"`
	Description string `yaml:"description,omitempty"`

	AllowedCommands  []string `yaml:"allowed_commands,omitempty"`
	WorkingDirectory string   `yaml:"working_directory,omitempty"`
	MaxExecutionTime string   `yaml:"max_execution_time,omitempty"`
	EnableSandboxing *bool    `yaml:"enable_sandboxing,omitempty"`

	MaxFileSize       int64    `yaml:"max_file_size,omitempty"`
	AllowedExtensions []string `yaml:"allowed_extensions,omitempty"`
	DeniedExtensions  []string `yaml:"denied_extensions,omitempty"`

	MaxReplacements int `yaml:"max_replacements,omitempty"`

	// Human-in-the-loop: Tool approval (A2A Protocol Section 6.3 - INPUT_REQUIRED)
	RequiresApproval *bool  `yaml:"requires_approval,omitempty"` // If true, agent pauses for user approval
	ApprovalPrompt   string `yaml:"approval_prompt,omitempty"`   // Custom prompt for approval request

	// read_file, apply_patch, grep_search settings
	ContextLines int `yaml:"context_lines,omitempty"`
	MaxResults   int `yaml:"max_results,omitempty"`

	DocumentStores     []string `yaml:"document_stores,omitempty"`
	DefaultLimit       int      `yaml:"default_limit,omitempty"`
	MaxLimit           int      `yaml:"max_limit,omitempty"`
	EnabledSearchTypes []string `yaml:"enabled_search_types,omitempty"`

	// web_request tool settings
	Timeout            string   `yaml:"timeout,omitempty"`
	MaxRetries         int      `yaml:"max_retries,omitempty"`
	MaxRequestSize     int64    `yaml:"max_request_size,omitempty"`
	MaxResponseSize    int64    `yaml:"max_response_size,omitempty"`
	AllowedDomains     []string `yaml:"allowed_domains,omitempty"`
	DeniedDomains      []string `yaml:"denied_domains,omitempty"`
	AllowedMethods     []string `yaml:"allowed_methods,omitempty"`
	AllowRedirects     *bool    `yaml:"allow_redirects,omitempty"`
	MaxRedirects       int      `yaml:"max_redirects,omitempty"`
	UserAgent          string   `yaml:"user_agent,omitempty"`
	FollowMetaRefresh  *bool    `yaml:"follow_meta_refresh,omitempty"`
	JavaScriptRendered *bool    `yaml:"javascript_rendered,omitempty"`

	ServerURL string `yaml:"server_url,omitempty"`

	Config map[string]interface{} `yaml:"config,omitempty"`
}

func (c *ToolConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("type is required")
	}

	switch c.Type {
	case "command":
		if c.EnableSandboxing == nil {
			c.EnableSandboxing = BoolPtr(true)
		}
		if !*c.EnableSandboxing && len(c.AllowedCommands) == 0 {
			return fmt.Errorf("allowed_commands is required when enable_sandboxing is false (security requirement)")
		}

	case "write_file":

	case "search_replace":

	case "search":
		if len(c.DocumentStores) == 0 {
			return fmt.Errorf("document_stores is required for search tool")
		}
	case "todo":

	case "web_request":
		// Liberal defaults - no validation for allowed/denied domains or methods
		// Users can opt-in to restrictions

	default:

	}

	return nil
}

func (c *ToolConfig) SetDefaults() {
	if c.Enabled == nil {
		c.Enabled = BoolPtr(true)
	}

	switch c.Type {
	case "command":
		if c.EnableSandboxing == nil {
			c.EnableSandboxing = BoolPtr(true)
		}
		if c.WorkingDirectory == "" {
			c.WorkingDirectory = "./"
		}
		if c.MaxExecutionTime == "" {
			c.MaxExecutionTime = "30s"
		}
	case "write_file":
		if c.MaxFileSize == 0 {
			c.MaxFileSize = 1048576
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
	case "web_request":
		// Liberal defaults - allow everything unless explicitly restricted
		if c.Timeout == "" {
			c.Timeout = "30s"
		}
		if c.MaxRetries == 0 {
			c.MaxRetries = 3
		}
		if c.MaxRequestSize == 0 {
			c.MaxRequestSize = 10485760 // 10MB
		}
		if c.MaxResponseSize == 0 {
			c.MaxResponseSize = 52428800 // 50MB
		}
		if c.AllowRedirects == nil {
			c.AllowRedirects = BoolPtr(true)
		}
		if c.MaxRedirects == 0 {
			c.MaxRedirects = 10
		}
		if c.UserAgent == "" {
			c.UserAgent = "Hector-Agent/1.0"
		}
		if c.FollowMetaRefresh == nil {
			c.FollowMetaRefresh = BoolPtr(false)
		}
		if c.JavaScriptRendered == nil {
			c.JavaScriptRendered = BoolPtr(false)
		}
		// No defaults for AllowedDomains, DeniedDomains, AllowedMethods
		// Omitted = allow all (liberal default)
	case "mcp":

	}
}

type DocumentStoreConfig struct {
	Name                string   `yaml:"name"`
	Source              string   `yaml:"source"` // "directory", "sql", "api"
	Path                string   `yaml:"path"`   // Required for directory source
	IncludePatterns     []string `yaml:"include_patterns"`
	ExcludePatterns     []string `yaml:"exclude_patterns"`            // If set, replaces defaults entirely
	AdditionalExcludes  []string `yaml:"additional_exclude_patterns"` // Extends default exclusions
	WatchChanges        *bool    `yaml:"watch_changes"`               // Only for directory source
	MaxFileSize         int64    `yaml:"max_file_size"`               // Only for directory source
	IncrementalIndexing *bool    `yaml:"incremental_indexing"`

	// SQL source configuration
	SQL        *DocumentStoreSQLConfig       `yaml:"sql,omitempty"`
	SQLTables  []DocumentStoreSQLTableConfig `yaml:"sql_tables,omitempty"`
	SQLMaxRows int                           `yaml:"sql_max_rows,omitempty"` // Max rows to index per table

	// API source configuration
	API *DocumentStoreAPIConfig `yaml:"api,omitempty"`

	// Chunking configuration
	ChunkSize     int    `yaml:"chunk_size"`     // Default: 800 characters
	ChunkOverlap  int    `yaml:"chunk_overlap"`  // Default: 0 characters
	ChunkStrategy string `yaml:"chunk_strategy"` // "simple", "overlapping", "semantic"

	// Metadata extraction
	ExtractMetadata   *bool    `yaml:"extract_metadata"`   // Default: false
	MetadataLanguages []string `yaml:"metadata_languages"` // Languages to extract metadata from

	// Performance
	MaxConcurrentFiles int `yaml:"max_concurrent_files"` // Default: 10 (renamed from MaxConcurrentFiles for clarity)

	// Progress tracking
	ShowProgress      *bool `yaml:"show_progress"`      // Default: true - show progress bar
	VerboseProgress   *bool `yaml:"verbose_progress"`   // Default: false - show current file
	EnableCheckpoints *bool `yaml:"enable_checkpoints"` // Default: true - enable resume capability
	QuietMode         *bool `yaml:"quiet_mode"`         // Default: true - suppress per-file warnings
}

// DocumentStoreSQLConfig defines SQL database connection for document store
type DocumentStoreSQLConfig struct {
	Driver   string `yaml:"driver"` // "postgres", "mysql", "sqlite3"
	Host     string `yaml:"host,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Database string `yaml:"database"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	SSLMode  string `yaml:"ssl_mode,omitempty"`
}

// DocumentStoreSQLTableConfig defines which SQL table to index
type DocumentStoreSQLTableConfig struct {
	Table           string   `yaml:"table"`
	Columns         []string `yaml:"columns"`          // Columns to concatenate for content
	IDColumn        string   `yaml:"id_column"`        // Primary key or unique identifier
	UpdatedColumn   string   `yaml:"updated_column"`   // Column for tracking updates (e.g., updated_at)
	WhereClause     string   `yaml:"where_clause"`     // Optional WHERE clause for filtering
	MetadataColumns []string `yaml:"metadata_columns"` // Columns to include as metadata
}

// DocumentStoreAPIConfig defines REST API configuration for document store
type DocumentStoreAPIConfig struct {
	BaseURL   string                           `yaml:"base_url"`
	Auth      *DocumentStoreAPIAuthConfig      `yaml:"auth,omitempty"`
	Endpoints []DocumentStoreAPIEndpointConfig `yaml:"endpoints"`
}

// DocumentStoreAPIAuthConfig defines authentication for API requests
type DocumentStoreAPIAuthConfig struct {
	Type   string            `yaml:"type"` // "bearer", "basic", "apikey"
	Token  string            `yaml:"token,omitempty"`
	User   string            `yaml:"user,omitempty"`
	Pass   string            `yaml:"pass,omitempty"`
	Header string            `yaml:"header,omitempty"` // Header name for apikey type
	Extra  map[string]string `yaml:"extra,omitempty"`
}

// DocumentStoreAPIEndpointConfig defines an API endpoint to index
type DocumentStoreAPIEndpointConfig struct {
	Path           string                            `yaml:"path"`
	Method         string                            `yaml:"method,omitempty"` // Default: GET
	Params         map[string]string                 `yaml:"params,omitempty"`
	Headers        map[string]string                 `yaml:"headers,omitempty"`
	Body           string                            `yaml:"body,omitempty"`
	Auth           *DocumentStoreAPIAuthConfig       `yaml:"auth,omitempty"` // Endpoint-specific auth
	IDField        string                            `yaml:"id_field,omitempty"`
	ContentField   string                            `yaml:"content_field,omitempty"` // Comma-separated or JSONPath
	MetadataFields []string                          `yaml:"metadata_fields,omitempty"`
	UpdatedField   string                            `yaml:"updated_field,omitempty"`
	Pagination     *DocumentStoreAPIPaginationConfig `yaml:"pagination,omitempty"`
}

// DocumentStoreAPIPaginationConfig defines pagination for API endpoints
type DocumentStoreAPIPaginationConfig struct {
	Type      string `yaml:"type"`       // "offset", "cursor", "page", "link"
	PageParam string `yaml:"page_param"` // Query parameter name for page/offset
	SizeParam string `yaml:"size_param"` // Query parameter name for page size
	MaxPages  int    `yaml:"max_pages"`  // Maximum pages to fetch (0 = unlimited)
	PageSize  int    `yaml:"page_size"`  // Items per page
	NextField string `yaml:"next_field"` // JSON field containing next page URL/cursor
	DataField string `yaml:"data_field"` // JSON field containing array of items (if nested)
}

func (c *DocumentStoreConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Source == "" {
		return fmt.Errorf("source is required")
	}

	switch c.Source {
	case "directory":
		if c.Path == "" {
			return fmt.Errorf("path is required for directory source")
		}
	case "sql":
		if c.SQL == nil {
			return fmt.Errorf("SQL configuration is required for SQL source")
		}
		if c.SQL.Driver == "" {
			return fmt.Errorf("SQL driver is required")
		}
		if c.SQL.Database == "" {
			return fmt.Errorf("SQL database name is required")
		}
		if len(c.SQLTables) == 0 {
			return fmt.Errorf("at least one SQL table configuration is required")
		}
	case "api":
		if c.API == nil {
			return fmt.Errorf("API configuration is required for API source")
		}
		if c.API.BaseURL == "" {
			return fmt.Errorf("API base URL is required")
		}
		if len(c.API.Endpoints) == 0 {
			return fmt.Errorf("at least one API endpoint is required")
		}
	default:
		return fmt.Errorf("unsupported source type: %s (supported: directory, sql, api)", c.Source)
	}

	return nil
}

func (c *DocumentStoreConfig) SetDefaults() {

	if c.Name == "" {
		c.Name = "default-docs"
	}
	if c.Source == "" {
		c.Source = "directory"
	}
	if c.Source == "directory" && c.Path == "" {
		c.Path = "./"
	}

	// Build default exclusion patterns
	defaultExcludes := []string{
		// Version control
		"**/.git/**", "**/.svn/**", "**/.hg/**", "**/.bzr/**",

		// Python dependencies and caches
		"**/site-packages/**", "**/dist-packages/**",
		"**/venv/**", "**/.venv/**", "**/virtualenv/**", "**/env/**",
		"**/*-env/**", "**/*_env/**", "**/__pycache__/**", "**/*.pyc", "**/*.pyo", "**/*.pyd",

		// Node.js dependencies
		"**/node_modules/**", "**/.npm/**", "**/.yarn/**", "**/.pnp/**",

		// Other language dependencies
		"**/vendor/**", "**/.bundle/**", "**/gems/**",

		// Build artifacts
		"**/dist/**", "**/build/**", "**/out/**", "**/output/**",
		"**/target/**", "**/.next/**", "**/.nuxt/**", "**/.output/**",
		"**/bin/**", "**/obj/**", "**/.gradle/**", "**/.m2/**",
		"**/.cache/**", "**/.parcel-cache/**",

		// IDE files
		"**/.vscode/**", "**/.idea/**", "**/.eclipse/**",
		"**/.settings/**", "**/*.swp", "**/*.swo", "**/*~",
		"**/.DS_Store", "**/Thumbs.db", "**/.directory",

		// Binary files
		"*.exe", "*.dll", "*.so", "*.dylib", "*.bin", "*.o", "*.a",
		"*.obj", "*.lib", "*.class",

		// Media files
		"*.png", "*.jpg", "*.jpeg", "*.gif", "*.bmp", "*.ico", "*.webp", "*.svg",
		"*.mp4", "*.avi", "*.mov", "*.mkv", "*.flv", "*.wmv",
		"*.mp3", "*.wav", "*.flac", "*.aac", "*.ogg", "*.wma",

		// Archives
		"*.zip", "*.tar", "*.gz", "*.bz2", "*.7z", "*.rar", "*.xz", "*.tgz",

		// Fonts
		"*.ttf", "*.otf", "*.woff", "*.woff2", "*.eot",

		// Databases
		"*.db", "*.sqlite", "*.sqlite3", "*.mdb",

		// Logs and temp files
		"*.log", "*.tmp", "*.temp", "*.bak", "*.cache",
		"**/logs/**", "**/tmp/**", "**/temp/**",

		// Lock files
		"**/package-lock.json", "**/yarn.lock", "**/pnpm-lock.yaml",
		"**/Gemfile.lock", "**/Cargo.lock", "**/poetry.lock",

		// Hector internal
		"**/.hector/**", "**/index_state_*.json",

		// Test artifacts
		"**/coverage/**", "**/.nyc_output/**", "**/test-results/**",
		"**/public/assets/**", "**/static/media/**",
	}

	// If ExcludePatterns is set, use it exclusively (override mode)
	// Otherwise use defaults + additional excludes (extend mode)
	if len(c.ExcludePatterns) == 0 {
		c.ExcludePatterns = defaultExcludes
		if len(c.AdditionalExcludes) > 0 {
			c.ExcludePatterns = append(c.ExcludePatterns, c.AdditionalExcludes...)
		}
	}
	if c.MaxFileSize == 0 {
		c.MaxFileSize = 10 * 1024 * 1024
	}

	if c.WatchChanges == nil {
		c.WatchChanges = BoolPtr(true)
	}

	if c.IncrementalIndexing == nil {
		c.IncrementalIndexing = BoolPtr(true)
	}

	// Chunking defaults
	if c.ChunkSize == 0 {
		c.ChunkSize = 800
	}
	if c.ChunkStrategy == "" {
		c.ChunkStrategy = "simple"
	}
	// chunk_overlap defaults to 0 (no overlap)

	// Metadata defaults
	if len(c.MetadataLanguages) == 0 {
		c.MetadataLanguages = []string{"go"} // Only Go by default
	}
	if c.ExtractMetadata == nil {
		c.ExtractMetadata = BoolPtr(false)
	}

	// Progress tracking defaults (using pointers for proper default detection)
	if c.ShowProgress == nil {
		c.ShowProgress = BoolPtr(true)
	}
	if c.VerboseProgress == nil {
		c.VerboseProgress = BoolPtr(false)
	}
	if c.EnableCheckpoints == nil {
		c.EnableCheckpoints = BoolPtr(true)
	}
	if c.QuietMode == nil {
		c.QuietMode = BoolPtr(true) // Default to true - suppress warnings
	}

	// Performance defaults
	if c.MaxConcurrentFiles == 0 {
		c.MaxConcurrentFiles = 10
	}

	// Progress tracking defaults (use field presence to determine if unset)
	// ShowProgress defaults to true (enable by default)
	// VerboseProgress defaults to false
	// EnableCheckpoints defaults to true (enable by default)
}

type TaskConfig struct {
	Backend      string            `yaml:"backend,omitempty"`
	WorkerPool   int               `yaml:"worker_pool,omitempty"`
	SQL          *TaskSQLConfig    `yaml:"sql,omitempty"`
	InputTimeout int               `yaml:"input_timeout,omitempty"` // Timeout in seconds for INPUT_REQUIRED state (default: 600)
	Timeout      int               `yaml:"timeout,omitempty"`       // Timeout in seconds for async task execution (default: 3600 = 1 hour)
	HITL         *HITLConfig       `yaml:"hitl,omitempty"`          // Human-in-the-loop configuration
	Checkpoint   *CheckpointConfig `yaml:"checkpoint,omitempty"`    // Generic checkpoint/resume configuration
}

type HITLConfig struct {
	Mode string `yaml:"mode,omitempty"` // "auto" (default), "blocking", or "async"
}

type CheckpointConfig struct {
	Enabled  bool                      `yaml:"enabled,omitempty"`  // Enable generic checkpointing (default: false)
	Strategy string                    `yaml:"strategy,omitempty"` // "event", "interval", or "hybrid" (default: "event")
	Interval *CheckpointIntervalConfig `yaml:"interval,omitempty"`
	Recovery *CheckpointRecoveryConfig `yaml:"recovery,omitempty"`
}

type CheckpointIntervalConfig struct {
	EveryNIterations int  `yaml:"every_n_iterations,omitempty"` // Checkpoint every N iterations (0 = disabled)
	AfterToolCalls   bool `yaml:"after_tool_calls,omitempty"`   // Always checkpoint after tool calls (default: false)
	BeforeLLMCalls   bool `yaml:"before_llm_calls,omitempty"`   // Checkpoint before LLM calls (default: false)
}

type CheckpointRecoveryConfig struct {
	AutoResume     bool `yaml:"auto_resume,omitempty"`      // Auto-resume on startup (default: false)
	AutoResumeHITL bool `yaml:"auto_resume_hitl,omitempty"` // Auto-resume INPUT_REQUIRED tasks (default: false)
	ResumeTimeout  int  `yaml:"resume_timeout,omitempty"`   // Max time to resume after restart (seconds, default: 3600)
}

func (c *TaskConfig) IsEnabled() bool {
	return c.Backend != "" || c.WorkerPool > 0 || c.SQL != nil
}

type TaskSQLConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Database string `yaml:"database"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	SSLMode  string `yaml:"ssl_mode,omitempty"`
	MaxConns int    `yaml:"max_conns,omitempty"`
	MaxIdle  int    `yaml:"max_idle,omitempty"`
}

func (c *TaskConfig) SetDefaults() {
	if c.Backend == "" {
		c.Backend = "memory"
	}
	if c.WorkerPool == 0 {
		c.WorkerPool = 100
	}
	if c.Timeout == 0 {
		c.Timeout = 3600 // Default: 1 hour
	}
	if c.SQL != nil {
		c.SQL.SetDefaults()
	}
	if c.Checkpoint != nil {
		c.Checkpoint.SetDefaults()
	}
}

func (c *CheckpointConfig) SetDefaults() {
	if c.Strategy == "" {
		c.Strategy = "event" // Default: event-driven (async HITL)
	}
	if c.Interval != nil {
		c.Interval.SetDefaults()
	}
	if c.Recovery != nil {
		c.Recovery.SetDefaults()
	}
}

func (c *CheckpointIntervalConfig) SetDefaults() {
	// No defaults - all fields are optional
}

func (c *CheckpointRecoveryConfig) SetDefaults() {
	if c.ResumeTimeout == 0 {
		c.ResumeTimeout = 3600 // Default: 1 hour
	}
}

func (c *TaskSQLConfig) SetDefaults() {
	if c.Driver == "" {
		c.Driver = "postgres"
	}
	if c.Host == "" && c.Driver != "sqlite" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		switch c.Driver {
		case "postgres":
			c.Port = 5432
		case "mysql":
			c.Port = 3306
		}
	}
	if c.SSLMode == "" && c.Driver == "postgres" {
		c.SSLMode = "disable"
	}
	if c.MaxConns == 0 {
		c.MaxConns = 25
	}
	if c.MaxIdle == 0 {
		c.MaxIdle = 5
	}
}

type AgentCredentials struct {
	Type         string `yaml:"type"`
	Token        string `yaml:"token,omitempty"`
	APIKey       string `yaml:"api_key,omitempty"`
	APIKeyHeader string `yaml:"api_key_header,omitempty"`
	Username     string `yaml:"username,omitempty"`
	Password     string `yaml:"password,omitempty"`
}

func (c *AgentCredentials) SetDefaults() {
	if c.Type == "" {
		c.Type = "bearer"
	}
	if c.Type == "api_key" && c.APIKeyHeader == "" {
		c.APIKeyHeader = "X-API-Key"
	}
}

func (c *AgentCredentials) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("credential type is required")
	}

	switch c.Type {
	case "bearer":
		if c.Token == "" {
			return fmt.Errorf("token is required for bearer authentication")
		}
	case "api_key":
		if c.APIKey == "" {
			return fmt.Errorf("api_key is required for api_key authentication")
		}
	case "basic":
		if c.Username == "" || c.Password == "" {
			return fmt.Errorf("username and password are required for basic authentication")
		}
	default:
		return fmt.Errorf("unsupported credential type '%s' (supported: bearer, api_key, basic)", c.Type)
	}
	return nil
}

type SecurityConfig struct {
	Schemes  map[string]*SecurityScheme `yaml:"schemes,omitempty"`
	Require  []map[string][]string      `yaml:"require,omitempty"`
	JWKSURL  string                     `yaml:"jwks_url,omitempty"`
	Issuer   string                     `yaml:"issuer,omitempty"`
	Audience string                     `yaml:"audience,omitempty"`
}

func (c *SecurityConfig) IsEnabled() bool {
	return len(c.Schemes) > 0 || len(c.Require) > 0 || c.JWKSURL != "" || c.Issuer != "" || c.Audience != ""
}

type SecurityScheme struct {
	Type         string `yaml:"type"`
	Scheme       string `yaml:"scheme,omitempty"`
	BearerFormat string `yaml:"bearer_format,omitempty"`
	Description  string `yaml:"description,omitempty"`

	In   string `yaml:"in,omitempty"`
	Name string `yaml:"name,omitempty"`
}

func (c *SecurityConfig) SetDefaults() {

}

func (c *SecurityConfig) Validate() error {
	if !c.IsEnabled() {
		return nil
	}
	for name, scheme := range c.Schemes {
		if scheme != nil {
			if scheme.Type == "" {
				return fmt.Errorf("security scheme '%s' must have a type", name)
			}

			switch scheme.Type {
			case "http":
				if scheme.Scheme != "bearer" && scheme.Scheme != "basic" {
					return fmt.Errorf("http security scheme '%s' must have scheme 'bearer' or 'basic'", name)
				}
			case "apiKey":
				if scheme.In == "" || scheme.Name == "" {
					return fmt.Errorf("apiKey security scheme '%s' must have 'in' and 'name' fields", name)
				}
			case "oauth2", "openIdConnect", "mutualTLS":

			default:
				return fmt.Errorf("unsupported security scheme type '%s' for '%s'", scheme.Type, name)
			}
		}
	}
	return nil
}

func (c *TaskConfig) Validate() error {
	if c.Backend != "" && c.Backend != "memory" && c.Backend != "sql" {
		return fmt.Errorf("invalid task backend '%s', must be 'memory' or 'sql'", c.Backend)
	}
	if c.WorkerPool < 0 {
		return fmt.Errorf("worker_pool must be non-negative")
	}
	if c.Backend == "sql" && c.SQL == nil {
		return fmt.Errorf("sql configuration is required when backend is 'sql'")
	}
	if c.SQL != nil {
		if err := c.SQL.Validate(); err != nil {
			return fmt.Errorf("sql config validation failed: %w", err)
		}
	}
	if c.HITL != nil {
		if err := c.HITL.Validate(); err != nil {
			return fmt.Errorf("hitl config validation failed: %w", err)
		}
	}
	return nil
}

func (c *HITLConfig) Validate() error {
	if c.Mode != "" && c.Mode != "auto" && c.Mode != "blocking" && c.Mode != "async" {
		return fmt.Errorf("invalid hitl.mode '%s', must be 'auto', 'blocking', or 'async'", c.Mode)
	}
	return nil
}

func (c *TaskSQLConfig) Validate() error {
	if c.Driver == "" {
		return fmt.Errorf("driver is required")
	}
	if c.Driver != "postgres" && c.Driver != "mysql" && c.Driver != "sqlite" {
		return fmt.Errorf("invalid driver '%s', must be 'postgres', 'mysql', or 'sqlite'", c.Driver)
	}
	if c.Database == "" {
		return fmt.Errorf("database is required")
	}
	if c.Driver != "sqlite" {
		if c.Host == "" {
			return fmt.Errorf("host is required for %s", c.Driver)
		}
		if c.Port <= 0 {
			return fmt.Errorf("port must be positive for %s", c.Driver)
		}
	}
	if c.MaxConns <= 0 {
		return fmt.Errorf("max_conns must be positive")
	}
	if c.MaxIdle < 0 {
		return fmt.Errorf("max_idle must be non-negative")
	}
	return nil
}

func (c *TaskSQLConfig) ConnectionString() string {
	switch c.Driver {
	case "postgres":
		sslMode := c.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}

		if c.Password != "" {
			return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
				c.Host, c.Port, c.Username, c.Password, c.Database, sslMode)
		}
		return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.Username, c.Database, sslMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			c.Username, c.Password, c.Host, c.Port, c.Database)
	case "sqlite":
		return c.Database
	default:
		return ""
	}
}

type SessionStoreConfig struct {
	Backend   string            `yaml:"backend,omitempty"`
	SQL       *SessionSQLConfig `yaml:"sql,omitempty"`
	RateLimit *RateLimitConfig  `yaml:"rate_limit,omitempty"`
}

func (c *SessionStoreConfig) IsEnabled() bool {
	return c.Backend != "" || c.SQL != nil
}

type SessionSQLConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Database string `yaml:"database"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	SSLMode  string `yaml:"ssl_mode,omitempty"`
	MaxConns int    `yaml:"max_conns,omitempty"`
	MaxIdle  int    `yaml:"max_idle,omitempty"`
}

func (c *SessionStoreConfig) SetDefaults() {
	if c.Backend == "" {
		c.Backend = "memory"
	}
	if c.SQL != nil {
		c.SQL.SetDefaults()
	}
	if c.RateLimit != nil {
		c.RateLimit.SetDefaults()
	}
}

func (c *SessionSQLConfig) SetDefaults() {
	if c.Driver == "" {
		c.Driver = "sqlite"
	}

	if c.Driver == "sqlite" && c.Database == "" {
		c.Database = "./sessions.db"
	}
	if c.Host == "" && c.Driver != "sqlite" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		switch c.Driver {
		case "postgres":
			c.Port = 5432
		case "mysql":
			c.Port = 3306
		}
	}
	if c.SSLMode == "" && c.Driver == "postgres" {
		c.SSLMode = "disable"
	}
	if c.MaxConns == 0 {
		c.MaxConns = 25
	}
	if c.MaxIdle == 0 {
		c.MaxIdle = 5
	}
}

func (c *SessionStoreConfig) Validate() error {
	if c.Backend != "" && c.Backend != "memory" && c.Backend != "sql" {
		return fmt.Errorf("invalid session store backend '%s', must be 'memory' or 'sql'", c.Backend)
	}
	if c.Backend == "sql" && c.SQL == nil {
		return fmt.Errorf("sql configuration is required when backend is 'sql'")
	}
	if c.SQL != nil {
		if err := c.SQL.Validate(); err != nil {
			return fmt.Errorf("sql config validation failed: %w", err)
		}
	}
	if c.RateLimit != nil {
		if err := c.RateLimit.Validate(); err != nil {
			return fmt.Errorf("rate limit config validation failed: %w", err)
		}
	}
	return nil
}

func (c *SessionSQLConfig) Validate() error {
	if c.Driver == "" {
		return fmt.Errorf("driver is required")
	}
	if c.Driver != "postgres" && c.Driver != "mysql" && c.Driver != "sqlite" {
		return fmt.Errorf("invalid driver '%s', must be 'postgres', 'mysql', or 'sqlite'", c.Driver)
	}
	if c.Database == "" {
		return fmt.Errorf("database is required")
	}
	if c.Driver != "sqlite" {
		if c.Host == "" {
			return fmt.Errorf("host is required for %s", c.Driver)
		}
		if c.Port <= 0 {
			return fmt.Errorf("port must be positive for %s", c.Driver)
		}
	}
	if c.MaxConns <= 0 {
		return fmt.Errorf("max_conns must be positive")
	}
	if c.MaxIdle < 0 {
		return fmt.Errorf("max_idle must be non-negative")
	}
	return nil
}

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	Enabled *bool           `yaml:"enabled" json:"enabled"`
	Scope   string          `yaml:"scope,omitempty" json:"scope,omitempty"`     // "session" or "user"
	Backend string          `yaml:"backend,omitempty" json:"backend,omitempty"` // "memory" or "sql"
	Limits  []RateLimitRule `yaml:"limits" json:"limits"`
}

// RateLimitRule defines a single rate limit rule
type RateLimitRule struct {
	Type   string `yaml:"type" json:"type"`     // "token" or "count"
	Window string `yaml:"window" json:"window"` // "minute", "hour", "day", "week", "month"
	Limit  int64  `yaml:"limit" json:"limit"`   // Maximum allowed in window
}

func (c *RateLimitConfig) SetDefaults() {
	if c.Enabled == nil {
		c.Enabled = BoolPtr(false)
	}
	if BoolValue(c.Enabled, false) && len(c.Limits) == 0 {
		// Default: 100k tokens per day, 60 requests per minute
		c.Limits = []RateLimitRule{
			{Type: "token", Window: "day", Limit: 100000},
			{Type: "count", Window: "minute", Limit: 60},
		}
	}
	if c.Scope == "" {
		c.Scope = "session" // Default to per-session limiting
	}
	if c.Backend == "" {
		c.Backend = "memory" // Default to memory backend
	}
}

func (c *RateLimitConfig) Validate() error {
	if c.Enabled == nil || !*c.Enabled {
		return nil
	}
	if len(c.Limits) == 0 {
		return fmt.Errorf("at least one limit must be defined when rate limiting is enabled")
	}
	if c.Scope != "" && c.Scope != "session" && c.Scope != "user" {
		return fmt.Errorf("invalid scope '%s', must be 'session' or 'user'", c.Scope)
	}
	if c.Backend != "" && c.Backend != "memory" && c.Backend != "sql" {
		return fmt.Errorf("invalid backend '%s', must be 'memory' or 'sql'", c.Backend)
	}
	for i, limit := range c.Limits {
		if err := limit.Validate(); err != nil {
			return fmt.Errorf("limit %d is invalid: %w", i, err)
		}
	}
	return nil
}

func (r *RateLimitRule) Validate() error {
	if r.Type != "token" && r.Type != "count" {
		return fmt.Errorf("invalid type '%s', must be 'token' or 'count'", r.Type)
	}
	if r.Window != "minute" && r.Window != "hour" && r.Window != "day" && r.Window != "week" && r.Window != "month" {
		return fmt.Errorf("invalid window '%s', must be 'minute', 'hour', 'day', 'week', or 'month'", r.Window)
	}
	if r.Limit <= 0 {
		return fmt.Errorf("limit must be positive, got %d", r.Limit)
	}
	return nil
}

func (c *SessionSQLConfig) ConnectionString() string {
	switch c.Driver {
	case "postgres":
		sslMode := c.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}

		if c.Password != "" {
			return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
				c.Host, c.Port, c.Username, c.Password, c.Database, sslMode)
		}
		return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.Username, c.Database, sslMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			c.Username, c.Password, c.Host, c.Port, c.Database)
	case "sqlite":
		return c.Database
	default:
		return ""
	}
}

type MemoryConfig struct {
	Strategy string `yaml:"strategy,omitempty"`

	Budget int `yaml:"budget,omitempty"`

	WindowSize int `yaml:"window_size,omitempty"`

	Threshold float64 `yaml:"threshold,omitempty"`
	Target    float64 `yaml:"target,omitempty"`

	LongTerm LongTermMemoryConfig `yaml:"long_term,omitempty"`
}

type LongTermMemoryConfig struct {
	StorageScope string `yaml:"storage_scope,omitempty"`
	BatchSize    int    `yaml:"batch_size,omitempty"`
	AutoRecall   *bool  `yaml:"auto_recall,omitempty"`
	RecallLimit  int    `yaml:"recall_limit,omitempty"`
	Collection   string `yaml:"collection,omitempty"`
}

func (c *LongTermMemoryConfig) IsEnabled() bool {
	return c.StorageScope != "" || c.BatchSize > 0 || c.RecallLimit > 0 || c.Collection != ""
}

func (c *LongTermMemoryConfig) SetDefaults() {
	if c.AutoRecall == nil {
		c.AutoRecall = BoolPtr(false)
	}
}

type PromptConfig struct {
	// Composable prompt slots for flexible prompt engineering
	PromptSlots *PromptSlotsConfig `yaml:"prompt_slots,omitempty"`

	// Simple system prompt override (use this OR prompt_slots, not both)
	SystemPrompt string `yaml:"system_prompt"`

	// Enable RAG context injection from document stores
	IncludeContext *bool `yaml:"include_context"`
}

// PromptSlotsConfig defines typed prompt slots for composable prompt engineering
type PromptSlotsConfig struct {
	SystemRole   string `yaml:"system_role,omitempty"`
	Instructions string `yaml:"instructions,omitempty"`
	UserGuidance string `yaml:"user_guidance,omitempty"`
}

func (c *MemoryConfig) Validate() error {
	if c.Budget < 0 {
		return fmt.Errorf("budget must be non-negative")
	}
	if c.WindowSize < 0 {
		return fmt.Errorf("window_size must be non-negative")
	}
	if c.Threshold < 0 || c.Threshold > 1 {
		return fmt.Errorf("threshold must be between 0.0 and 1.0")
	}
	if c.Target < 0 || c.Target > 1 {
		return fmt.Errorf("target must be between 0.0 and 1.0")
	}

	if c.Strategy != "" && c.Strategy != "buffer_window" && c.Strategy != "summary_buffer" {
		return fmt.Errorf("invalid strategy '%s', must be 'buffer_window' or 'summary_buffer'", c.Strategy)
	}

	return nil
}

func (c *MemoryConfig) SetDefaults() {

	if c.Strategy == "" {
		c.Strategy = "summary_buffer"
	}

	switch c.Strategy {
	case "buffer_window":
		if c.WindowSize <= 0 {
			c.WindowSize = 20
		}

	case "summary_buffer":
		if c.Budget <= 0 {
			c.Budget = 8000
		}
		if c.Threshold <= 0 {
			c.Threshold = 0.85
		}
		if c.Target <= 0 {
			c.Target = 0.7
		}
	}
}

func (c *PromptConfig) Validate() error {
	// All fields are optional, no validation needed
	return nil
}

func (c *PromptConfig) SetDefaults() {
	if c.IncludeContext == nil {
		c.IncludeContext = BoolPtr(false)
	}
}

type ReasoningConfig struct {
	Engine          string `yaml:"engine"`
	MaxIterations   int    `yaml:"max_iterations"`
	EnableStreaming *bool  `yaml:"enable_streaming"`

	// Display flags (enabled by default)
	ShowTools    *bool `yaml:"show_tools"`    // Show all tool-related events (calls, results, execution)
	ShowThinking *bool `yaml:"show_thinking"` // Show all thinking-related content (todos, goals, internal reasoning)
}

func (c *ReasoningConfig) Validate() error {
	if c.Engine == "" {
		return fmt.Errorf("engine is required")
	}
	if c.MaxIterations <= 0 {
		return fmt.Errorf("max_iterations must be positive")
	}
	return nil
}

func (c *ReasoningConfig) SetDefaults() {
	if c.Engine == "" {
		c.Engine = "default"
	}
	if c.MaxIterations == 0 {
		c.MaxIterations = 100
	}

	if c.EnableStreaming == nil {
		c.EnableStreaming = BoolPtr(true)
	}

	// Both flags enabled by default
	if c.ShowTools == nil {
		c.ShowTools = BoolPtr(true)
	}
	if c.ShowThinking == nil {
		c.ShowThinking = BoolPtr(true)
	}
}

type SearchConfig struct {
	Models       []SearchModel `yaml:"models"`
	TopK         int           `yaml:"top_k"`         // Default number of results to return
	Threshold    float32       `yaml:"threshold"`     // Minimum similarity score (0.0-1.0)
	PreserveCase *bool         `yaml:"preserve_case"` // Don't lowercase queries (default: true for code search)
}

func (c *SearchConfig) Validate() error {
	if len(c.Models) == 0 {
		return fmt.Errorf("at least one search model is required")
	}
	for i, model := range c.Models {
		if err := model.Validate(); err != nil {
			return fmt.Errorf("search model %d validation failed: %w", i, err)
		}
	}
	if c.TopK < 0 {
		return fmt.Errorf("top_k must be non-negative")
	}
	if c.Threshold < 0 || c.Threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1")
	}
	return nil
}

func (c *SearchConfig) SetDefaults() {

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
		c.Threshold = 0.5 // Default 50% similarity threshold (balanced precision/recall for RAG)
	}

	if c.PreserveCase == nil {
		c.PreserveCase = BoolPtr(true) // Default to true for code search
	}

	// Query processing defaults - optimized for code search
	// PreserveCase defaults to true (important for code identifiers like HTTP, API, etc.)
	// Whitespace is always normalized for query consistency

	for i := range c.Models {
		c.Models[i].SetDefaults()
	}
}

type SearchModel struct {
	Name        string `yaml:"name"`
	Collection  string `yaml:"collection"`
	DefaultTopK int    `yaml:"default_top_k"`
	MaxTopK     int    `yaml:"max_top_k"`
}

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

func (c *SearchModel) SetDefaults() {
	if c.DefaultTopK == 0 {
		c.DefaultTopK = 10
	}
	if c.MaxTopK == 0 {
		c.MaxTopK = 100
	}
}

// PerformanceConfig controls performance-related settings.
// Note: For operation timeouts, use per-tool max_execution_time instead.
type PerformanceConfig struct {
	MaxConcurrency int `yaml:"max_concurrency"`
}

func (c *PerformanceConfig) Validate() error {
	if c.MaxConcurrency <= 0 {
		return fmt.Errorf("max_concurrency must be positive")
	}
	return nil
}

func (c *PerformanceConfig) SetDefaults() {
	if c.MaxConcurrency == 0 {
		c.MaxConcurrency = 4
	}
}

type A2AServerConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	GRPCPort           int    `yaml:"grpc_port,omitempty"` // Optional separate gRPC port (default: 50051)
	BaseURL            string `yaml:"base_url,omitempty"`
	PreferredTransport string `yaml:"preferred_transport,omitempty"` // "grpc", "json-rpc", or "rest" (default: "json-rpc")
}

func (c *A2AServerConfig) IsEnabled() bool {
	return c.Port > 0 || c.Host != ""
}

func (c *A2AServerConfig) Validate() error {
	if c.IsEnabled() {
		if c.Port <= 0 || c.Port > 65535 {
			return fmt.Errorf("invalid port: %d", c.Port)
		}
	}
	return nil
}

func (c *A2AServerConfig) SetDefaults() {
	if c.Host == "" {
		c.Host = "0.0.0.0"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.PreferredTransport == "" {
		c.PreferredTransport = "json-rpc"
	}
}

type AuthConfig struct {
	JWKSURL  string `yaml:"jwks_url"`
	Issuer   string `yaml:"issuer"`
	Audience string `yaml:"audience"`
}

func (c *AuthConfig) IsEnabled() bool {
	return c.JWKSURL != "" && c.Issuer != "" && c.Audience != ""
}

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

func (c *AuthConfig) SetDefaults() {
}

type ObservabilityConfig struct {
	Tracing        TracingConfig `yaml:"tracing,omitempty"`
	MetricsEnabled *bool         `yaml:"metrics_enabled,omitempty"`
}

type TracingConfig struct {
	Enabled      *bool   `yaml:"enabled"`
	ExporterType string  `yaml:"exporter_type"`
	EndpointURL  string  `yaml:"endpoint_url"`
	SamplingRate float64 `yaml:"sampling_rate"`
	ServiceName  string  `yaml:"service_name"`
}

func (c *ObservabilityConfig) Validate() error {
	if err := c.Tracing.Validate(); err != nil {
		return fmt.Errorf("tracing config validation failed: %w", err)
	}
	return nil
}

func (c *ObservabilityConfig) SetDefaults() {
	c.Tracing.SetDefaults()
	if c.MetricsEnabled == nil {
		c.MetricsEnabled = BoolPtr(false)
	}
}

func (c *TracingConfig) Validate() error {
	if c.Enabled == nil || !*c.Enabled {
		return nil
	}
	if c.EndpointURL == "" {
		return fmt.Errorf("endpoint_url is required when tracing is enabled")
	}
	if c.SamplingRate < 0 || c.SamplingRate > 1 {
		return fmt.Errorf("sampling_rate must be between 0 and 1")
	}
	return nil
}

func (c *TracingConfig) SetDefaults() {
	if c.ServiceName == "" {
		c.ServiceName = "hector"
	}
	if c.Enabled == nil {
		c.Enabled = BoolPtr(false)
	}
	if BoolValue(c.Enabled, false) {
		if c.SamplingRate == 0 {
			c.SamplingRate = 1.0
		}
		if c.ExporterType == "" {
			c.ExporterType = "otlp"
		}
		if c.EndpointURL == "" {
			c.EndpointURL = "localhost:4317"
		}
	}
}

type A2ACardConfig struct {
	Version            string             `yaml:"version"`
	InputModes         []string           `yaml:"input_modes"`
	OutputModes        []string           `yaml:"output_modes"`
	Skills             []A2ASkillConfig   `yaml:"skills"`
	Provider           *A2AProviderConfig `yaml:"provider,omitempty"`
	PreferredTransport string             `yaml:"preferred_transport,omitempty"` // Override global preferred_transport for this agent

	DocumentationURL string `yaml:"documentation_url,omitempty"`
}

func (c *A2ACardConfig) Validate() error {
	// If only preferred_transport is set, it's valid (lightweight config)
	if c.PreferredTransport != "" && c.Version == "" && len(c.InputModes) == 0 && len(c.OutputModes) == 0 && len(c.Skills) == 0 {
		return nil
	}

	// Otherwise, validate full A2A config
	if c.Version == "" {
		return fmt.Errorf("a2a.version is required")
	}
	if len(c.InputModes) == 0 {
		return fmt.Errorf("a2a.input_modes is required and must not be empty")
	}
	if len(c.OutputModes) == 0 {
		return fmt.Errorf("a2a.output_modes is required and must not be empty")
	}
	if len(c.Skills) == 0 {
		return fmt.Errorf("a2a.skills is required and must not be empty")
	}

	for i, skill := range c.Skills {
		if err := skill.Validate(); err != nil {
			return fmt.Errorf("a2a.skills[%d]: %w", i, err)
		}
	}

	return nil
}

type A2ASkillConfig struct {
	ID string `yaml:"id"`

	Name string `yaml:"name"`

	Description string `yaml:"description"`

	Tags []string `yaml:"tags,omitempty"`

	Examples []string `yaml:"examples,omitempty"`
}

func (c *A2ASkillConfig) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("skill.id is required")
	}
	if c.Name == "" {
		return fmt.Errorf("skill.name is required")
	}
	if c.Description == "" {
		return fmt.Errorf("skill.description is required")
	}
	return nil
}

type A2AProviderConfig struct {
	Name string `yaml:"name,omitempty"`
	URL  string `yaml:"url,omitempty"`
}
