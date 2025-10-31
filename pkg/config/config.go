package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	DefaultMaxDocumentStoreSize = 50 * 1024 * 1024
)

func ProcessConfigPipeline(cfg *Config) (*Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("ProcessConfigPipeline: config cannot be nil")
	}

	cfg.PreProcess()

	cfg.SetDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("ProcessConfigPipeline: validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) PreProcess() {
	c.initializeMaps()

	for _, agent := range c.Agents {
		if agent.DocsFolder != "" && len(agent.DocumentStores) == 0 {
			c.expandDocsFolder(agent)
		}

		if agent.EnableTools && len(agent.Tools) == 0 {
			c.expandEnableTools()
		}
	}
}

func (c *Config) initializeMaps() {
	if c.LLMs == nil {
		c.LLMs = make(map[string]*LLMProviderConfig)
	}
	if c.Databases == nil {
		c.Databases = make(map[string]*DatabaseProviderConfig)
	}
	if c.Embedders == nil {
		c.Embedders = make(map[string]*EmbedderProviderConfig)
	}
	if c.Agents == nil {
		c.Agents = make(map[string]*AgentConfig)
	}
	if c.Tools == nil {
		c.Tools = make(map[string]*ToolConfig)
	}
	if c.DocumentStores == nil {
		c.DocumentStores = make(map[string]*DocumentStoreConfig)
	}
	if c.SessionStores == nil {
		c.SessionStores = make(map[string]*SessionStoreConfig)
	}
}

func (c *Config) expandDocsFolder(agent *AgentConfig) {
	storeName := generateStoreNameFromPath(agent.DocsFolder)

	docStoreConfig := &DocumentStoreConfig{
		Name:                storeName,
		Source:              "directory",
		Path:                agent.DocsFolder,
		WatchChanges:        true,
		MaxFileSize:         DefaultMaxDocumentStoreSize,
		IncrementalIndexing: true,
	}
	c.DocumentStores[storeName] = docStoreConfig

	agent.DocumentStores = []string{storeName}
	agent.DocsFolder = ""

	if agent.Database == "" {
		if _, exists := c.Databases["default-database"]; !exists {
			c.Databases["default-database"] = &DatabaseProviderConfig{}
		}
		agent.Database = "default-database"
	}

	if agent.Embedder == "" {
		if _, exists := c.Embedders["default-embedder"]; !exists {
			c.Embedders["default-embedder"] = &EmbedderProviderConfig{}
		}
		agent.Embedder = "default-embedder"
	}

	if c.Tools == nil {
		c.Tools = make(map[string]*ToolConfig)
	}
	if _, exists := c.Tools["search"]; !exists {
		c.Tools["search"] = &ToolConfig{
			Type:           "search",
			DocumentStores: []string{storeName},
		}
	}
}

func (c *Config) expandEnableTools() {
	if c.Tools == nil {
		c.Tools = make(map[string]*ToolConfig)
	}

	for toolName, toolConfig := range GetDefaultToolConfigs() {
		if _, exists := c.Tools[toolName]; !exists {
			c.Tools[toolName] = toolConfig
		}
	}
}

func (c *Config) SetDefaults() {
	c.Global.SetDefaults()

	if c.LLMs == nil {
		c.LLMs = make(map[string]*LLMProviderConfig)
	}
	if c.Databases == nil {
		c.Databases = make(map[string]*DatabaseProviderConfig)
	}
	if c.Embedders == nil {
		c.Embedders = make(map[string]*EmbedderProviderConfig)
	}
	if c.Agents == nil {
		c.Agents = make(map[string]*AgentConfig)
	}
	if c.DocumentStores == nil {
		c.DocumentStores = make(map[string]*DocumentStoreConfig)
	}
	if c.Tools == nil {
		c.Tools = make(map[string]*ToolConfig)
	}
	if c.SessionStores == nil {
		c.SessionStores = make(map[string]*SessionStoreConfig)
	}

	if len(c.LLMs) == 0 {
		c.LLMs["default-llm"] = &LLMProviderConfig{}
	}
	if len(c.Databases) == 0 {
		c.Databases["default-database"] = &DatabaseProviderConfig{}
	}
	if len(c.Embedders) == 0 {
		c.Embedders["default-embedder"] = &EmbedderProviderConfig{}
	}
	if len(c.Agents) == 0 {
		c.Agents["default-agent"] = &AgentConfig{}
	}

	for name := range c.LLMs {
		if c.LLMs[name] != nil {
			c.LLMs[name].SetDefaults()
		}
	}

	for name := range c.Databases {
		if c.Databases[name] != nil {
			c.Databases[name].SetDefaults()
		}
	}

	for name := range c.Embedders {
		if c.Embedders[name] != nil {
			c.Embedders[name].SetDefaults()
		}
	}

	for name := range c.Agents {
		if c.Agents[name] != nil {
			c.Agents[name].SetDefaults()
		}
	}

	if len(c.Tools) == 0 {
		c.Tools = GetDefaultToolConfigs()
	}

	for name := range c.Tools {
		if c.Tools[name] != nil {
			c.Tools[name].SetDefaults()
		}
	}

	for name := range c.DocumentStores {
		if c.DocumentStores[name] != nil {
			c.DocumentStores[name].SetDefaults()
		}
	}

	for name := range c.SessionStores {
		if c.SessionStores[name] != nil {
			c.SessionStores[name].SetDefaults()
		}
	}

	c.Plugins.SetDefaults()
}

type Config struct {
	Version     string            `yaml:"version,omitempty"`
	Name        string            `yaml:"name,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`

	Global GlobalSettings `yaml:"global,omitempty"`

	LLMs      map[string]*LLMProviderConfig      `yaml:"llms,omitempty"`
	Databases map[string]*DatabaseProviderConfig `yaml:"databases,omitempty"`
	Embedders map[string]*EmbedderProviderConfig `yaml:"embedders,omitempty"`

	Agents map[string]*AgentConfig `yaml:"agents,omitempty"`

	Tools map[string]*ToolConfig `yaml:"tools,omitempty"`

	DocumentStores map[string]*DocumentStoreConfig `yaml:"document_stores,omitempty"`

	SessionStores map[string]*SessionStoreConfig `yaml:"session_stores,omitempty"`

	Plugins PluginConfigs `yaml:"plugins,omitempty"`
}

func (c *Config) ValidateAgent(agentID string) error {

	if len(c.Agents) == 0 {

		return nil
	}

	if _, exists := c.Agents[agentID]; !exists {

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

func (c *Config) Validate() error {

	if err := c.Global.Validate(); err != nil {
		return fmt.Errorf("global settings validation failed: %w", err)
	}

	for name, llm := range c.LLMs {
		if llm != nil {
			if err := llm.Validate(); err != nil {
				return fmt.Errorf("LLM '%s' validation failed: %w", name, err)
			}
		}
	}

	for name, db := range c.Databases {
		if db != nil {
			if err := db.Validate(); err != nil {
				return fmt.Errorf("database '%s' validation failed: %w", name, err)
			}
		}
	}

	for name, embedder := range c.Embedders {
		if embedder != nil {
			if err := embedder.Validate(); err != nil {
				return fmt.Errorf("embedder '%s' validation failed: %w", name, err)
			}
		}
	}

	for agentID, agent := range c.Agents {
		if agent != nil {
			// Set agent name default to agent ID if not specified
			if agent.Name == "" {
				agent.Name = agentID
			}
			if err := agent.Validate(); err != nil {
				return fmt.Errorf("agent '%s' validation failed: %w", agentID, err)
			}
		}
	}

	for name, tool := range c.Tools {
		if tool != nil {
			if err := tool.Validate(); err != nil {
				return fmt.Errorf("tool '%s' validation failed: %w", name, err)
			}
		}
	}

	for name, store := range c.DocumentStores {
		if store != nil {
			if err := store.Validate(); err != nil {
				return fmt.Errorf("document store '%s' validation failed: %w", name, err)
			}
		}
	}

	for name, store := range c.SessionStores {
		if store != nil {
			if err := store.Validate(); err != nil {
				return fmt.Errorf("session store '%s' validation failed: %w", name, err)
			}
		}
	}

	if err := c.Plugins.Validate(); err != nil {
		return fmt.Errorf("plugins validation failed: %w", err)
	}

	if err := c.validateReferences(); err != nil {
		return fmt.Errorf("reference validation failed: %w", err)
	}

	return nil
}

func (c *Config) validateReferences() error {
	for agentName, agent := range c.Agents {
		if agent == nil {
			continue
		}

		if agent.Type != "native" {
			continue
		}

		if agent.LLM != "" {
			if _, exists := c.LLMs[agent.LLM]; !exists {
				return fmt.Errorf("agent '%s': llm '%s' not found (available: %v)",
					agentName, agent.LLM, mapKeys(c.LLMs))
			}
		}

		if agent.Database != "" {
			if _, exists := c.Databases[agent.Database]; !exists {
				return fmt.Errorf("agent '%s': database '%s' not found (available: %v)",
					agentName, agent.Database, mapKeys(c.Databases))
			}
		}

		if agent.Embedder != "" {
			if _, exists := c.Embedders[agent.Embedder]; !exists {
				return fmt.Errorf("agent '%s': embedder '%s' not found (available: %v)",
					agentName, agent.Embedder, mapKeys(c.Embedders))
			}
		}

		for _, storeName := range agent.DocumentStores {
			if _, exists := c.DocumentStores[storeName]; !exists {
				return fmt.Errorf("agent '%s': document store '%s' not found (available: %v)",
					agentName, storeName, mapKeys(c.DocumentStores))
			}
		}

		if agent.SessionStore != "" {
			if _, exists := c.SessionStores[agent.SessionStore]; !exists {
				return fmt.Errorf("agent '%s': session store '%s' not found (available: %v)",
					agentName, agent.SessionStore, mapKeys(c.SessionStores))
			}
		}
	}

	for toolName, tool := range c.Tools {
		if tool == nil {
			continue
		}

		if tool.Type == "search" {
			for _, storeName := range tool.DocumentStores {
				if _, exists := c.DocumentStores[storeName]; !exists {
					return fmt.Errorf("tool '%s': document store '%s' not found (available: %v)",
						toolName, storeName, mapKeys(c.DocumentStores))
				}
			}
		}
	}

	return nil
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

type GlobalSettings struct {
	Logging LoggingConfig `yaml:"logging,omitempty"`

	Performance PerformanceConfig `yaml:"performance,omitempty"`

	A2AServer A2AServerConfig `yaml:"a2a_server,omitempty"`

	Auth AuthConfig `yaml:"auth,omitempty"`

	Observability ObservabilityConfig `yaml:"observability,omitempty"`
}

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
	if err := c.Observability.Validate(); err != nil {
		return fmt.Errorf("observability config validation failed: %w", err)
	}
	return nil
}

func (c *GlobalSettings) SetDefaults() {
	c.Logging.SetDefaults()
	c.Performance.SetDefaults()
	c.A2AServer.SetDefaults()
	c.Auth.SetDefaults()
	c.Observability.SetDefaults()
}

const DefaultAgentName = "assistant"

func CreateZeroConfig(source interface{}) *Config {
	provider := extractStringField(source, "Provider")
	apiKey := extractStringField(source, "APIKey")
	baseURL := extractStringField(source, "BaseURL")
	model := extractStringField(source, "Model")
	enableTools := extractBoolField(source, "Tools")
	mcpURL := extractStringField(source, "MCPURL")
	docsFolder := extractStringField(source, "DocsFolder")
	embedderModel := extractStringField(source, "EmbedderModel")
	agentName := extractStringField(source, "AgentName")
	observe := extractBoolField(source, "Observe")

	if apiKey == "" {
		if provider != "" {
			apiKey = GetProviderAPIKey(provider)
		} else {
			if key := GetProviderAPIKey("openai"); key != "" {
				apiKey = key
				provider = "openai"
			} else if key := GetProviderAPIKey("anthropic"); key != "" {
				apiKey = key
				provider = "anthropic"
			} else if key := GetProviderAPIKey("gemini"); key != "" {
				apiKey = key
				provider = "gemini"
			}
		}
	}

	if mcpURL == "" {
		mcpURL = os.Getenv("MCP_URL")
	}

	if provider == "" {
		provider = "openai"
	}

	if agentName == "" {
		agentName = DefaultAgentName
	}

	llmConfig := &LLMProviderConfig{
		Type: provider,
	}
	if apiKey != "" {
		llmConfig.APIKey = apiKey
	}
	if baseURL != "" {
		llmConfig.Host = baseURL
	}
	if model != "" {
		llmConfig.Model = model
	}

	cfg := &Config{
		Name: "Zero Config Mode",
		LLMs: map[string]*LLMProviderConfig{
			provider: llmConfig,
		},
		Databases:      make(map[string]*DatabaseProviderConfig),
		Embedders:      make(map[string]*EmbedderProviderConfig),
		DocumentStores: make(map[string]*DocumentStoreConfig),
		SessionStores:  make(map[string]*SessionStoreConfig),
	}

	if observe {
		cfg.Global.Observability = ObservabilityConfig{
			MetricsEnabled: true,
			Tracing: TracingConfig{
				Enabled: true,
			},
		}
	}

	agentConfig := AgentConfig{
		Name: agentName,
		Type: "native",
		LLM:  provider,
	}

	if docsFolder != "" {
		agentConfig.DocsFolder = docsFolder
		if embedderModel != "" {
			cfg.Embedders["default-embedder"] = &EmbedderProviderConfig{
				Model: embedderModel,
			}
		}
	}

	if enableTools {
		agentConfig.EnableTools = true
	} else if mcpURL != "" {
		agentConfig.Tools = nil
	} else {
		agentConfig.Tools = []string{}
	}

	cfg.Agents = map[string]*AgentConfig{
		agentName: &agentConfig,
	}

	if mcpURL != "" {
		if cfg.Tools == nil {
			cfg.Tools = make(map[string]*ToolConfig)
		}
		cfg.Tools["mcp"] = &ToolConfig{
			Type:      "mcp",
			Enabled:   true,
			ServerURL: mcpURL,
		}
	}

	return cfg
}

func (c *Config) GetAgent(name string) (*AgentConfig, bool) {
	agent, exists := c.Agents[name]
	return agent, exists
}

func (c *Config) GetDocumentStore(name string) (*DocumentStoreConfig, bool) {
	store, exists := c.DocumentStores[name]
	return store, exists
}

func (c *Config) ListAgents() []string {
	agents := make([]string, 0, len(c.Agents))
	for name := range c.Agents {
		agents = append(agents, name)
	}
	return agents
}

func (c *Config) ListDocumentStores() []string {
	stores := make([]string, 0, len(c.DocumentStores))
	for name := range c.DocumentStores {
		stores = append(stores, name)
	}
	return stores
}

func extractStringField(source any, fieldName string) string {
	v := reflect.ValueOf(source)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return ""
	}

	if field.Kind() != reflect.String {
		return ""
	}

	return field.String()
}

func extractBoolField(source any, fieldName string) bool {
	v := reflect.ValueOf(source)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return false
	}

	if field.Kind() != reflect.Bool {
		return false
	}

	return field.Bool()
}

func extractIntField(source any, fieldName string) int {
	v := reflect.ValueOf(source)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return 0
	}

	if field.Kind() != reflect.Int {
		return 0
	}

	return int(field.Int())
}

func generateStoreNameFromPath(sourcePath string) string {
	normalizedPath := filepath.Clean(sourcePath)

	absPath, err := filepath.Abs(normalizedPath)
	if err != nil {
		absPath = normalizedPath
	}

	hash := sha256.Sum256([]byte(absPath))
	hashStr := hex.EncodeToString(hash[:])[:12]

	dirName := filepath.Base(absPath)
	if dirName == "" || dirName == "." {
		dirName = "root"
	}

	dirName = strings.ReplaceAll(dirName, " ", "_")
	dirName = strings.ReplaceAll(dirName, "-", "_")

	return fmt.Sprintf("docs_%s_%s", dirName, hashStr)
}
