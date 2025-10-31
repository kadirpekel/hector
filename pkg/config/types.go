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

type PluginDiscoveryConfig struct {
	Enabled            bool     `yaml:"enabled" json:"enabled"`
	Paths              []string `yaml:"paths" json:"paths"`
	ScanSubdirectories bool     `yaml:"scan_subdirectories" json:"scan_subdirectories"`
}

func (c *PluginDiscoveryConfig) SetDefaults() {
	if len(c.Paths) == 0 {
		c.Paths = []string{"./plugins", "~/.hector/plugins"}
	}

}

func (c *PluginDiscoveryConfig) Validate() error {
	return nil
}

type PluginConfig struct {
	Name    string                 `yaml:"name" json:"name"`
	Type    string                 `yaml:"type" json:"type"`
	Path    string                 `yaml:"path" json:"path"`
	Enabled bool                   `yaml:"enabled" json:"enabled"`
	Config  map[string]interface{} `yaml:"config" json:"config"`
}

func (c *PluginConfig) SetDefaults() {
	if c.Type == "" {
		c.Type = "grpc"
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

type DatabaseProviderConfig struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	APIKey   string `yaml:"api_key"`
	Timeout  int    `yaml:"timeout"`
	UseTLS   bool   `yaml:"use_tls"`
	Insecure bool   `yaml:"insecure"`
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
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
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
	if c.Timeout == 0 {
		c.Timeout = 30
	}
}

type EmbedderProviderConfig struct {
	Type       string `yaml:"type"`
	Model      string `yaml:"model"`
	Host       string `yaml:"host"`
	Dimension  int    `yaml:"dimension"`
	Timeout    int    `yaml:"timeout"`
	MaxRetries int    `yaml:"max_retries"`
}

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

func (c *EmbedderProviderConfig) SetDefaults() {

	if c.Type == "" {
		c.Type = "ollama"
	}
	if c.Model == "" {
		c.Model = "nomic-embed-text"
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

type AgentConfig struct {
	Type string `yaml:"type,omitempty"`

	Name        string `yaml:"name"`
	Description string `yaml:"description"`

	Visibility string `yaml:"visibility,omitempty"`

	URL         string            `yaml:"url,omitempty"`
	Credentials *AgentCredentials `yaml:"credentials,omitempty"`

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
	EnableTools bool   `yaml:"enable_tools,omitempty"`

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
		if c.EnableTools && len(c.Tools) > 0 {
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
	EnableSandboxing bool          `yaml:"enable_sandboxing"`
}

func (c *CommandToolsConfig) Validate() error {

	if !c.EnableSandboxing && len(c.AllowedCommands) == 0 {
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

}

type SearchToolConfig struct {
	DocumentStores     []string `yaml:"document_stores"`
	DefaultLimit       int      `yaml:"default_limit"`
	MaxLimit           int      `yaml:"max_limit"`
	MaxResults         int      `yaml:"max_results"`
	EnabledSearchTypes []string `yaml:"enabled_search_types"`
}

func (c *SearchToolConfig) Validate() error {
	if c.DefaultLimit <= 0 {
		return fmt.Errorf("default_limit must be positive")
	}
	if c.MaxResults <= 0 {
		return fmt.Errorf("max_results must be positive")
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
	if c.MaxResults == 0 {
		c.MaxResults = 100
	}
	if len(c.EnabledSearchTypes) == 0 {
		c.EnabledSearchTypes = []string{"content", "file", "function", "struct"}
	}
}

type FileWriterConfig struct {
	MaxFileSize       int      `yaml:"max_file_size"`
	AllowedExtensions []string `yaml:"allowed_extensions"`
	DeniedExtensions  []string `yaml:"denied_extensions"`
	BackupOnOverwrite bool     `yaml:"backup_on_overwrite"`
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
}

type SearchReplaceConfig struct {
	MaxReplacements  int    `yaml:"max_replacements"`
	ShowDiff         bool   `yaml:"show_diff"`
	CreateBackup     bool   `yaml:"create_backup"`
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
			EnableSandboxing: true,
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
	Enabled     bool   `yaml:"enabled,omitempty"`
	Description string `yaml:"description,omitempty"`

	AllowedCommands  []string `yaml:"allowed_commands,omitempty"`
	WorkingDirectory string   `yaml:"working_directory,omitempty"`
	MaxExecutionTime string   `yaml:"max_execution_time,omitempty"`
	EnableSandboxing bool     `yaml:"enable_sandboxing,omitempty"`

	MaxFileSize       int64    `yaml:"max_file_size,omitempty"`
	AllowedExtensions []string `yaml:"allowed_extensions,omitempty"`
	DeniedExtensions  []string `yaml:"denied_extensions,omitempty"`
	ForbiddenPaths    []string `yaml:"forbidden_paths,omitempty"`

	MaxReplacements int  `yaml:"max_replacements,omitempty"`
	BackupEnabled   bool `yaml:"backup_enabled,omitempty"`

	DocumentStores     []string `yaml:"document_stores,omitempty"`
	DefaultLimit       int      `yaml:"default_limit,omitempty"`
	MaxLimit           int      `yaml:"max_limit,omitempty"`
	MaxResults         int      `yaml:"max_results,omitempty"`
	EnabledSearchTypes []string `yaml:"enabled_search_types,omitempty"`

	ServerURL string `yaml:"server_url,omitempty"`

	Config map[string]interface{} `yaml:"config,omitempty"`
}

func (c *ToolConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("type is required")
	}

	switch c.Type {
	case "command":

		if !c.EnableSandboxing && len(c.AllowedCommands) == 0 {
			return fmt.Errorf("allowed_commands is required when enable_sandboxing is false (security requirement)")
		}

	case "write_file":

	case "search_replace":

	case "search":
		if len(c.DocumentStores) == 0 {
			return fmt.Errorf("document_stores is required for search tool")
		}
	case "todo":

	default:

	}

	return nil
}

func (c *ToolConfig) SetDefaults() {

	c.Enabled = true

	switch c.Type {
	case "command":
		if !c.EnableSandboxing {
			c.EnableSandboxing = true
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
		if c.MaxResults == 0 {
			c.MaxResults = 100
		}
	case "mcp":

	}
}

type DocumentStoreConfig struct {
	Name                string   `yaml:"name"`
	Source              string   `yaml:"source"` // Only "directory" supported
	Path                string   `yaml:"path"`
	IncludePatterns     []string `yaml:"include_patterns"`
	ExcludePatterns     []string `yaml:"exclude_patterns"`            // If set, replaces defaults entirely
	AdditionalExcludes  []string `yaml:"additional_exclude_patterns"` // Extends default exclusions
	WatchChanges        bool     `yaml:"watch_changes"`
	MaxFileSize         int64    `yaml:"max_file_size"`
	IncrementalIndexing bool     `yaml:"incremental_indexing"`

	// Chunking configuration
	ChunkSize     int    `yaml:"chunk_size"`     // Default: 800 characters
	ChunkOverlap  int    `yaml:"chunk_overlap"`  // Default: 0 characters
	ChunkStrategy string `yaml:"chunk_strategy"` // "simple", "overlapping", "semantic"

	// Metadata extraction
	ExtractMetadata   bool     `yaml:"extract_metadata"`   // Default: true
	MetadataLanguages []string `yaml:"metadata_languages"` // Languages to extract metadata from

	// Performance
	MaxConcurrentFiles int `yaml:"max_concurrent_files"` // Default: 10

	// Progress tracking
	ShowProgress      *bool `yaml:"show_progress"`      // Default: true - show progress bar
	VerboseProgress   *bool `yaml:"verbose_progress"`   // Default: false - show current file
	EnableCheckpoints *bool `yaml:"enable_checkpoints"` // Default: true - enable resume capability
	QuietMode         *bool `yaml:"quiet_mode"`         // Default: true - suppress per-file warnings
}

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

func (c *DocumentStoreConfig) SetDefaults() {

	if c.Name == "" {
		c.Name = "default-docs"
	}
	if c.Source == "" {
		c.Source = "directory"
	}
	if c.Path == "" {
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

	if !c.WatchChanges {
		c.WatchChanges = true
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
	// extract_metadata defaults to false initially

	// Progress tracking defaults (using pointers for proper default detection)
	if c.ShowProgress == nil {
		trueVal := true
		c.ShowProgress = &trueVal
	}
	if c.VerboseProgress == nil {
		falseVal := false
		c.VerboseProgress = &falseVal
	}
	if c.EnableCheckpoints == nil {
		trueVal := true
		c.EnableCheckpoints = &trueVal
	}
	if c.QuietMode == nil {
		trueVal := true // Default to true - suppress warnings
		c.QuietMode = &trueVal
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
	Backend    string         `yaml:"backend,omitempty"`
	WorkerPool int            `yaml:"worker_pool,omitempty"`
	SQL        *TaskSQLConfig `yaml:"sql,omitempty"`
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
	if c.SQL != nil {
		c.SQL.SetDefaults()
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
	Backend string            `yaml:"backend,omitempty"`
	SQL     *SessionSQLConfig `yaml:"sql,omitempty"`
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

	Summarization          bool    `yaml:"summarization,omitempty"`
	SummarizationThreshold float64 `yaml:"summarization_threshold,omitempty"`
}

type LongTermMemoryConfig struct {
	StorageScope string `yaml:"storage_scope,omitempty"`
	BatchSize    int    `yaml:"batch_size,omitempty"`
	AutoRecall   bool   `yaml:"auto_recall,omitempty"`
	RecallLimit  int    `yaml:"recall_limit,omitempty"`
	Collection   string `yaml:"collection,omitempty"`
}

func (c *LongTermMemoryConfig) IsEnabled() bool {
	return c.StorageScope != "" || c.BatchSize > 0 || c.RecallLimit > 0 || c.Collection != ""
}

type PromptConfig struct {
	PromptSlots *PromptSlotsConfig `yaml:"prompt_slots,omitempty"`

	SystemPrompt   string            `yaml:"system_prompt"`
	Instructions   string            `yaml:"instructions"`
	FullTemplate   string            `yaml:"full_template"`
	Template       string            `yaml:"template"`
	Variables      map[string]string `yaml:"variables"`
	IncludeContext bool              `yaml:"include_context"`
	IncludeHistory bool              `yaml:"include_history"`
	IncludeTools   bool              `yaml:"include_tools"`

	MaxHistoryMessages  int     `yaml:"max_history_messages,omitempty"`
	MaxContextLength    int     `yaml:"max_context_length,omitempty"`
	EnableSummarization bool    `yaml:"enable_summarization,omitempty"`
	SummarizeThreshold  float64 `yaml:"summarize_threshold,omitempty"`
	SmartMemory         bool    `yaml:"smart_memory,omitempty"`
	MemoryBudget        int     `yaml:"memory_budget,omitempty"`
}

// PromptSlotsConfig defines typed prompt slots for composable prompt engineering
type PromptSlotsConfig struct {
	SystemRole            string `yaml:"system_role,omitempty"`
	ReasoningInstructions string `yaml:"reasoning_instructions,omitempty"`
	ToolUsage             string `yaml:"tool_usage,omitempty"`
	OutputFormat          string `yaml:"output_format,omitempty"`
	CommunicationStyle    string `yaml:"communication_style,omitempty"`
	Additional            string `yaml:"additional,omitempty"`
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

	if c.Summarization && c.Strategy == "summary_buffer" {

		if c.SummarizationThreshold > 0 {
			c.Threshold = c.SummarizationThreshold
		}
	}

	switch c.Strategy {
	case "buffer_window":
		if c.WindowSize <= 0 {
			c.WindowSize = 20
		}

	case "summary_buffer":
		if c.Budget <= 0 {
			c.Budget = 2000
		}
		if c.Threshold <= 0 {
			c.Threshold = 0.8
		}
		if c.Target <= 0 {
			c.Target = 0.6
		}
	}
}

func (c *PromptConfig) Validate() error {
	if c.MaxContextLength < 0 {
		return fmt.Errorf("max_context_length must be non-negative")
	}
	return nil
}

func (c *PromptConfig) SetDefaults() {

	if c.MaxContextLength == 0 {
		c.MaxContextLength = 4000
	}

	if c.MaxHistoryMessages == 0 {
		c.MaxHistoryMessages = 10
	}
	if c.SummarizeThreshold == 0 {
		c.SummarizeThreshold = 0.8
	}
	if c.MemoryBudget == 0 && c.SmartMemory {
		c.MemoryBudget = 2000
	}
}

type ReasoningConfig struct {
	Engine                     string  `yaml:"engine"`
	MaxIterations              int     `yaml:"max_iterations"`
	EnableSelfReflection       bool    `yaml:"enable_self_reflection"`
	EnableStructuredReflection *bool   `yaml:"enable_structured_reflection"`
	EnableGoalExtraction       bool    `yaml:"enable_goal_extraction"`
	ShowDebugInfo              bool    `yaml:"show_debug_info"`
	ShowToolExecution          *bool   `yaml:"show_tool_execution"`
	ShowThinking               bool    `yaml:"show_thinking"`
	EnableStreaming            *bool   `yaml:"enable_streaming"`
	QualityThreshold           float64 `yaml:"quality_threshold"`

	// Tool display configuration
	ToolDisplayMode string `yaml:"tool_display_mode"` // inline, detailed, hidden, thinking
	ShowToolArgs    bool   `yaml:"show_tool_args"`    // Show tool arguments
	ShowToolResults bool   `yaml:"show_tool_results"` // Show tool results
}

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

func (c *ReasoningConfig) SetDefaults() {
	if c.Engine == "" {
		c.Engine = "default"
	}
	if c.MaxIterations == 0 {

		c.MaxIterations = 100
	}
	if c.QualityThreshold == 0 {
		c.QualityThreshold = 0.7
	}

	if c.EnableStreaming == nil {
		trueVal := true
		c.EnableStreaming = &trueVal
	}
	if c.ShowToolExecution == nil {
		trueVal := true
		c.ShowToolExecution = &trueVal
	}
	if c.EnableStructuredReflection == nil {
		trueVal := true
		c.EnableStructuredReflection = &trueVal
	}

	// Set tool display defaults
	if c.ToolDisplayMode == "" {
		c.ToolDisplayMode = "inline" // Default to clean inline display
	}
	// ShowToolArgs and ShowToolResults default to false for clean output
}

type SearchConfig struct {
	Models              []SearchModel `yaml:"models"`
	TopK                int           `yaml:"top_k"`
	Threshold           float64       `yaml:"threshold"`
	MaxContextLength    int           `yaml:"max_context_length"`
	PreserveCase        bool          `yaml:"preserve_case"`        // Don't lowercase queries (default: true for code search)
	NormalizeWhitespace bool          `yaml:"normalize_whitespace"` // Normalize whitespace (default: true)
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
		c.Threshold = 0.7
	}
	if c.MaxContextLength == 0 {
		c.MaxContextLength = 4000
	}

	// Query processing defaults - optimized for code search
	// PreserveCase defaults to true (important for code identifiers like HTTP, API, etc.)
	// NormalizeWhitespace defaults to true (always safe for query consistency)

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

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

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

type PerformanceConfig struct {
	MaxConcurrency int           `yaml:"max_concurrency"`
	Timeout        time.Duration `yaml:"timeout"`
}

func (c *PerformanceConfig) Validate() error {
	if c.MaxConcurrency <= 0 {
		return fmt.Errorf("max_concurrency must be positive")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

func (c *PerformanceConfig) SetDefaults() {
	if c.MaxConcurrency == 0 {
		c.MaxConcurrency = 4
	}
	if c.Timeout == 0 {
		c.Timeout = 15 * time.Minute
	}
}

type A2AServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	BaseURL string `yaml:"base_url,omitempty"`
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
	MetricsEnabled bool          `yaml:"metrics_enabled,omitempty"`
}

type TracingConfig struct {
	Enabled      bool    `yaml:"enabled"`
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
}

func (c *TracingConfig) Validate() error {
	if c.Enabled {
		if c.EndpointURL == "" {
			return fmt.Errorf("endpoint_url is required when tracing is enabled")
		}
		if c.SamplingRate < 0 || c.SamplingRate > 1 {
			return fmt.Errorf("sampling_rate must be between 0 and 1")
		}
	}
	return nil
}

func (c *TracingConfig) SetDefaults() {
	if c.ServiceName == "" {
		c.ServiceName = "hector"
	}
	if c.SamplingRate == 0 && c.Enabled {
		c.SamplingRate = 1.0
	}
	if c.ExporterType == "" && c.Enabled {
		c.ExporterType = "otlp"
	}
	if c.EndpointURL == "" && c.Enabled {
		c.EndpointURL = "localhost:4317"
	}
}

type A2ACardConfig struct {
	Version string `yaml:"version"`

	InputModes []string `yaml:"input_modes"`

	OutputModes []string `yaml:"output_modes"`

	Skills []A2ASkillConfig `yaml:"skills"`

	Provider *A2AProviderConfig `yaml:"provider,omitempty"`

	DocumentationURL string `yaml:"documentation_url,omitempty"`
}

func (c *A2ACardConfig) Validate() error {
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

	URL string `yaml:"url,omitempty"`

	ContactEmail string `yaml:"contact_email,omitempty"`
}
