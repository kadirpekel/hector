package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
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

	// Apply defaults to agents
	c.applyDefaults()

	for _, agent := range c.Agents {
		// Expand inline configs to top-level providers
		c.expandInlineConfigs(agent)

		if agent.DocsFolder != "" && len(agent.DocumentStores) == 0 {
			c.expandDocsFolder(agent)
		}

		if agent.EnableTools != nil && *agent.EnableTools && len(agent.Tools) == 0 {
			c.expandEnableTools()
		}
	}
}

func (c *Config) initializeMaps() {
	if c.LLMs == nil {
		c.LLMs = make(map[string]*LLMProviderConfig)
	}
	if c.VectorStores == nil {
		c.VectorStores = make(map[string]*VectorStoreConfig)
	}
	if c.Databases == nil {
		c.Databases = make(map[string]*DatabaseConfig)
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
		Source:                    "directory",
		Path:                      agent.DocsFolder,
		EnableWatchChanges:        BoolPtr(true),
		MaxFileSize:               DefaultMaxDocumentStoreSize,
		EnableIncrementalIndexing: BoolPtr(true),
	}
	c.DocumentStores[storeName] = docStoreConfig

	agent.DocumentStores = []string{storeName}
	agent.DocsFolder = ""

	// Inform about auto-created components
	slog.Info("Auto-created from docs_folder shortcut",
		"document_store", storeName,
		"path", docStoreConfig.Path)

	if agent.VectorStore == "" {
		if _, exists := c.VectorStores["default-vector-store"]; !exists {
			c.VectorStores["default-vector-store"] = &VectorStoreConfig{}
		}
		agent.VectorStore = "default-vector-store"
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

	// Auto-configure MCP parsers ONLY if user explicitly specified a parser tool name
	// Explicit is better than implicit - parser must be explicitly defined to be active
	if agent.MCPParserTool != "" && docStoreConfig.MCPParsers == nil {
		// User explicitly specified parser tool name(s) via --mcp-parser-tool
		// Support comma-separated tool names for fallback chain (e.g., "parse_document,docling_parse,convert_document")
		toolNames := strings.Split(agent.MCPParserTool, ",")
		// Trim whitespace from each tool name
		for i, name := range toolNames {
			toolNames[i] = strings.TrimSpace(name)
		}
		// Runtime will validate that tools exist when actually used
		docStoreConfig.MCPParsers = &DocumentStoreMCPParserConfig{
			ToolNames: toolNames,
			Extensions: []string{
				".pdf",
				".docx",
				".pptx",
				".xlsx",
				".html",
			},
			Priority:     IntPtr(8), // Higher than native parsers (5)
			PreferNative: BoolPtr(false),
		}
		slog.Info("Auto-configured MCP parsers", "tools", toolNames)
	}

	// Clear temporary zero-config fields after expansion
	agent.MCPParserTool = ""
}

func (c *Config) expandEnableTools() {
	if c.Tools == nil {
		c.Tools = make(map[string]*ToolConfig)
	}

	createdTools := []string{}
	for toolName, toolConfig := range GetDefaultToolConfigs() {
		if _, exists := c.Tools[toolName]; !exists {
			c.Tools[toolName] = toolConfig
			createdTools = append(createdTools, toolName)
		}
	}

	if len(createdTools) > 0 {
		slog.Info("Auto-created tools from enable_tools shortcut", "tools", createdTools)
	}
}

func (c *Config) SetDefaults() {
	c.Global.SetDefaults()

	if c.LLMs == nil {
		c.LLMs = make(map[string]*LLMProviderConfig)
	}
	if c.VectorStores == nil {
		c.VectorStores = make(map[string]*VectorStoreConfig)
	}
	if c.Databases == nil {
		c.Databases = make(map[string]*DatabaseConfig)
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
	if len(c.VectorStores) == 0 {
		c.VectorStores["default-vector-store"] = &VectorStoreConfig{}
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

	for name := range c.VectorStores {
		if c.VectorStores[name] != nil {
			c.VectorStores[name].SetDefaults()
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

	// Defaults for agents (new feature)
	Defaults *AgentDefaultsConfig `yaml:"defaults,omitempty"`

	LLMs         map[string]*LLMProviderConfig      `yaml:"llms,omitempty"`
	VectorStores map[string]*VectorStoreConfig      `yaml:"vector_stores,omitempty"`
	Databases    map[string]*DatabaseConfig         `yaml:"databases,omitempty"`
	Embedders    map[string]*EmbedderProviderConfig `yaml:"embedders,omitempty"`

	Agents map[string]*AgentConfig `yaml:"agents,omitempty"`

	Tools map[string]*ToolConfig `yaml:"tools,omitempty"`

	DocumentStores map[string]*DocumentStoreConfig `yaml:"document_stores,omitempty"`

	SessionStores map[string]*SessionStoreConfig `yaml:"session_stores,omitempty"`

	Plugins PluginConfigs `yaml:"plugins,omitempty"`
}

// AgentDefaultsConfig provides default values for agent configurations
type AgentDefaultsConfig struct {
	LLM          string `yaml:"llm,omitempty"`           // Default LLM reference
	VectorStore  string `yaml:"vector_store,omitempty"`  // Default vector store reference
	Embedder     string `yaml:"embedder,omitempty"`      // Default embedder reference
	SessionStore string `yaml:"session_store,omitempty"` // Default session store reference
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

	// Validate all references (moved from separate validateReferences() method)
	if err := c.validateReferences(); err != nil {
		return fmt.Errorf("reference validation failed: %w", err)
	}

	return nil
}

// validateReferences validates all cross-references between configuration sections
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

		if agent.VectorStore != "" {
			if _, exists := c.VectorStores[agent.VectorStore]; !exists {
				return fmt.Errorf("agent '%s': vector_store '%s' not found (available: %v)",
					agentName, agent.VectorStore, mapKeys(c.VectorStores))
			}
		}

		if agent.Embedder != "" {
			if _, exists := c.Embedders[agent.Embedder]; !exists {
				return fmt.Errorf("agent '%s': embedder '%s' not found (available: %v)",
					agentName, agent.Embedder, mapKeys(c.Embedders))
			}
		}

		// Validate conditional database/embedder requirements for agents with document stores
		if len(agent.DocumentStores) > 0 {
			// Check if all document stores have their own vector_store/embedder
			allStoresHaveVectorStore := true
			allStoresHaveEmbedder := true
			for _, dsName := range agent.DocumentStores {
				dsConfig, exists := c.DocumentStores[dsName]
				if !exists {
					return fmt.Errorf("agent '%s': document store '%s' not found (available: %v)",
						agentName, dsName, mapKeys(c.DocumentStores))
				}
				if dsConfig.VectorStore == "" {
					allStoresHaveVectorStore = false
				}
				if dsConfig.Embedder == "" {
					allStoresHaveEmbedder = false
				}
			}

			// Check if IncludeContext is enabled
			includeContextEnabled := agent.Prompt.IncludeContext != nil && *agent.Prompt.IncludeContext

			// Check if long-term memory is enabled
			longTermMemoryEnabled := agent.Memory.LongTerm.IsEnabled()

			// Determine if agent vector_store/embedder is needed
			needsAgentVectorStore := longTermMemoryEnabled || includeContextEnabled || !allStoresHaveVectorStore
			needsAgentEmbedder := longTermMemoryEnabled || includeContextEnabled || !allStoresHaveEmbedder

			// Validate requirements
			if needsAgentVectorStore && agent.VectorStore == "" {
				reasons := []string{}
				if longTermMemoryEnabled {
					reasons = append(reasons, "long-term memory is enabled")
				}
				if includeContextEnabled {
					reasons = append(reasons, "IncludeContext is enabled")
				}
				if !allStoresHaveVectorStore {
					reasons = append(reasons, "at least one document store doesn't specify its own vector_store")
				}
				return fmt.Errorf("agent '%s': vector_store is required when: %s", agentName, strings.Join(reasons, ", "))
			}

			if needsAgentEmbedder && agent.Embedder == "" {
				reasons := []string{}
				if longTermMemoryEnabled {
					reasons = append(reasons, "long-term memory is enabled")
				}
				if includeContextEnabled {
					reasons = append(reasons, "IncludeContext is enabled")
				}
				if !allStoresHaveEmbedder {
					reasons = append(reasons, "at least one document store doesn't specify its own embedder")
				}
				return fmt.Errorf("agent '%s': embedder is required when: %s", agentName, strings.Join(reasons, ", "))
			}
		}

		// Also check for IncludeContext without document stores, and long-term memory
		if len(agent.DocumentStores) == 0 {
			includeContextEnabled := agent.Prompt.IncludeContext != nil && *agent.Prompt.IncludeContext
			longTermMemoryEnabled := agent.Memory.LongTerm.IsEnabled()

			if (includeContextEnabled || longTermMemoryEnabled) && agent.VectorStore == "" {
				reasons := []string{}
				if longTermMemoryEnabled {
					reasons = append(reasons, "long-term memory is enabled")
				}
				if includeContextEnabled {
					reasons = append(reasons, "IncludeContext is enabled")
				}
				return fmt.Errorf("agent '%s': vector_store is required when: %s", agentName, strings.Join(reasons, ", "))
			}

			if (includeContextEnabled || longTermMemoryEnabled) && agent.Embedder == "" {
				reasons := []string{}
				if longTermMemoryEnabled {
					reasons = append(reasons, "long-term memory is enabled")
				}
				if includeContextEnabled {
					reasons = append(reasons, "IncludeContext is enabled")
				}
				return fmt.Errorf("agent '%s': embedder is required when: %s", agentName, strings.Join(reasons, ", "))
			}
		}

		// Validate document store references
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

	// Validate session store references (database references)
	for storeName, store := range c.SessionStores {
		if store == nil {
			continue
		}

		if store.Backend == "sql" && store.SQLDatabase != "" {
			if _, exists := c.Databases[store.SQLDatabase]; !exists {
				return fmt.Errorf("session store '%s': sql_database '%s' not found (available: %v)",
					storeName, store.SQLDatabase, mapKeys(c.Databases))
			}
		}
	}

	// Tool validation: Search tool no longer has document_stores field
	// Document stores come from agent assignment, not tool config
	for _, tool := range c.Tools {
		if tool == nil {
			continue
		}
		// No validation needed for search tool document_stores (removed from config)
	}

	// Validate task database references
	for agentName, agent := range c.Agents {
		if agent == nil || agent.Task == nil {
			continue
		}

		if agent.Task.Backend == "sql" && agent.Task.SQLDatabase != "" {
			if _, exists := c.Databases[agent.Task.SQLDatabase]; !exists {
				return fmt.Errorf("agent '%s': task sql_database '%s' not found (available: %v)",
					agentName, agent.Task.SQLDatabase, mapKeys(c.Databases))
			}
		}
	}

	// Validate document store references (vector_store, database, embedder)
	for storeName, store := range c.DocumentStores {
		if store == nil {
			continue
		}

		// Validate vector_store reference
		if store.VectorStore != "" {
			if _, exists := c.VectorStores[store.VectorStore]; !exists {
				return fmt.Errorf("document store '%s': vector_store '%s' not found (available: %v)",
					storeName, store.VectorStore, mapKeys(c.VectorStores))
			}
		}

		// Validate database reference (for SQL source)
		if store.SQLDatabase != "" {
			if _, exists := c.Databases[store.SQLDatabase]; !exists {
				return fmt.Errorf("document store '%s': sql_database '%s' not found (available: %v)",
					storeName, store.SQLDatabase, mapKeys(c.Databases))
			}
		}

		// Validate embedder reference
		if store.Embedder != "" {
			if _, exists := c.Embedders[store.Embedder]; !exists {
				return fmt.Errorf("document store '%s': embedder '%s' not found (available: %v)",
					storeName, store.Embedder, mapKeys(c.Embedders))
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
	Performance PerformanceConfig `yaml:"performance,omitempty"`

	A2AServer A2AServerConfig `yaml:"a2a_server,omitempty"`

	Auth AuthConfig `yaml:"auth,omitempty"`

	Observability ObservabilityConfig `yaml:"observability,omitempty"`
}

func (c *GlobalSettings) Validate() error {
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
	temperature := extractFloatField(source, "Temperature")
	maxTokens := extractIntField(source, "MaxTokens")
	role := extractStringField(source, "Role")
	instruction := extractStringField(source, "Instruction")
	enableTools := extractBoolField(source, "Tools")
	thinking := extractBoolField(source, "Thinking")
	mcpURL := extractStringField(source, "MCPURL")
	mcpParserTool := extractStringField(source, "MCPParserTool")
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
	// Set temperature and maxTokens
	// The CLI library (Kong) sets defaults from struct tags BEFORE we extract:
	// - If user doesn't provide --temperature: field is 0.7 (from default tag)
	// - If user provides --temperature 0: field is 0.0
	// - If user provides --temperature 0.5: field is 0.5
	// Use pointer to distinguish "explicitly set" from "not set"
	llmConfig.Temperature = &temperature
	if maxTokens == 0 {
		// Explicitly set to 0 - use sentinel value so SetDefaults knows to keep it as 0
		llmConfig.MaxTokens = -1
	} else {
		llmConfig.MaxTokens = maxTokens
	}

	cfg := &Config{
		Name: "Zero Config Mode",
		LLMs: map[string]*LLMProviderConfig{
			provider: llmConfig,
		},
		VectorStores:   make(map[string]*VectorStoreConfig),
		Databases:      make(map[string]*DatabaseConfig),
		Embedders:      make(map[string]*EmbedderProviderConfig),
		DocumentStores: make(map[string]*DocumentStoreConfig),
		SessionStores:  make(map[string]*SessionStoreConfig),
	}

	if observe {
		cfg.Global.Observability = ObservabilityConfig{
			EnableMetrics: BoolPtr(true),
			Tracing: TracingConfig{
				Enabled: BoolPtr(true),
			},
		}
	}

	agentConfig := AgentConfig{
		Name: agentName,
		Type: "native",
		LLM:  provider,
	}

	// Add custom role and/or instruction if provided
	if role != "" || instruction != "" {
		agentConfig.Prompt.PromptSlots = &PromptSlotsConfig{
			SystemRole:   role,
			UserGuidance: instruction,
		}
	}

	if docsFolder != "" {
		agentConfig.DocsFolder = docsFolder
		if mcpParserTool != "" {
			agentConfig.MCPParserTool = mcpParserTool
		}
		if embedderModel != "" {
			cfg.Embedders["default-embedder"] = &EmbedderProviderConfig{
				Model: embedderModel,
			}
		}
	}
	// Note: If mcpParserTool is set without docsFolder, validation in AgentConfig.Validate() will catch it
	// We don't set MCPParserTool here if docsFolder is empty, so it will be caught during validation

	if enableTools {
		agentConfig.EnableTools = BoolPtr(true)
	} else if mcpURL != "" {
		agentConfig.Tools = nil
	} else {
		agentConfig.Tools = []string{}
	}

	// Set thinking flag only if explicitly enabled via --thinking
	// If not set, it will follow the normal default (false) via SetDefaults()
	if thinking {
		agentConfig.Reasoning.EnableThinkingDisplay = BoolPtr(true)
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
			Enabled:   BoolPtr(true),
			ServerURL: mcpURL,
			// Note: Internal flag not set here - by default MCP tools are visible to agents
			// Users can use a config file with internal: true if they want to hide them
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

func extractFloatField(source any, fieldName string) float64 {
	v := reflect.ValueOf(source)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() || !field.CanInterface() {
		return 0
	}

	if field.Kind() != reflect.Float64 && field.Kind() != reflect.Float32 {
		return 0
	}

	return field.Float()
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

	if field.Kind() != reflect.Int && field.Kind() != reflect.Int8 && field.Kind() != reflect.Int16 && field.Kind() != reflect.Int32 && field.Kind() != reflect.Int64 {
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
