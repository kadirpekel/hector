// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kadirpekel/hector/v2/observability"
	"github.com/kadirpekel/hector/v2/utils"
)

// ZeroConfig are CLI options for zero-config mode.
type ZeroConfig struct {
	// Provider type (anthropic, openai, gemini, ollama).
	Provider string

	// Model name.
	Model string

	// APIKey (usually from environment).
	APIKey string

	// BaseURL is a custom API base URL.
	BaseURL string

	// Temperature for generation.
	Temperature float64

	// MaxTokens for generation.
	MaxTokens int

	// Instruction is the system prompt.
	Instruction string

	// Role is the agent's role.
	Role string

	// MCPURL is an MCP server URL to connect to.
	MCPURL string

	// Tools enables built-in local tools.
	// Empty string or "all" enables all tools.
	// Comma-separated list enables specific tools (e.g., "read_file,write_file").
	Tools string

	// ApproveTools enables approval for specific tools (comma-separated).
	// Overrides smart defaults set by SetDefaults().
	ApproveTools string

	// NoApproveTools disables approval for specific tools (comma-separated).
	// Overrides smart defaults set by SetDefaults().
	NoApproveTools string

	// Thinking enables extended thinking for the LLM (like --tools enables tools).
	Thinking *bool

	// ThinkingBudget sets the token budget for thinking (default: 1024 if not specified).
	ThinkingBudget int

	// Streaming enables token-by-token streaming from the LLM (enabled by default).
	Streaming *bool

	// Port for the server.
	Port int

	// AgentName is the name of the agent.
	AgentName string

	// Storage specifies the storage backend: sqlite, postgres, mysql, or inmemory.
	// When set (and not "inmemory"), enables persistent storage for tasks, sessions,
	// and checkpointing.
	Storage string

	// StorageDB overrides the default database DSN/path for storage.
	// For SQLite: file path (default: ./.hector/hector.db)
	// For PostgreSQL/MySQL: connection string or individual params
	StorageDB string

	// Observe enables observability (metrics + OTLP tracing).
	// When enabled, exports traces to localhost:4317 and enables Prometheus metrics.
	Observe bool

	// DocsFolder is a path to documents for RAG.
	// When set, auto-creates a document store with chromem (embedded vector DB).
	DocsFolder string

	// EmbedderModel overrides the auto-detected embedder model.
	// Auto-detection: OpenAI → text-embedding-3-small, otherwise → ollama/nomic-embed-text
	EmbedderModel string

	// RAGWatch enables file watching for auto re-indexing (default: true).
	RAGWatch *bool

	// MCPParserTool is the MCP tool name(s) for document parsing (e.g., Docling).
	// Comma-separated for fallback chain (e.g., "parse_document,docling_parse").
	MCPParserTool string
}

// CreateZeroConfig creates a Config from CLI options.
// This enables "zero config" mode where users can run Hector
// without a config file by providing CLI flags.
//
// IMPORTANT DESIGN PRINCIPLE: Do NOT duplicate defaults that are already handled by SetDefaults().
// Only set values here if:
//  1. They are explicitly provided via CLI flags (non-zero/non-empty)
//  2. They are zero-config specific overrides (different from config file defaults)
//
// Examples:
//   - ✅ Set streaming=true (zero-config override, different from config file default=false)
//   - ✅ Set thinking.Enabled=true when --thinking flag is used (explicit enablement)
//   - ❌ Do NOT set thinking.BudgetTokens=1024 (SetDefaults() already handles this)
//   - ❌ Do NOT set temperature=0.7 (SetDefaults() already handles this)
//   - ❌ Do NOT set maxTokens=4096 (SetDefaults() already handles this)
//
// After this function, cfg.SetDefaults() is called to apply all standard defaults.
func CreateZeroConfig(opts ZeroConfig) *Config {
	// Detect provider from environment if not specified
	provider := LLMProvider(opts.Provider)
	if provider == "" {
		provider = detectProviderFromEnv()
	}

	// Get API key
	apiKey := opts.APIKey
	if apiKey == "" {
		apiKey = getAPIKeyFromEnv(provider)
	}

	// Check for MCP URL from environment
	mcpURL := opts.MCPURL
	if mcpURL == "" {
		mcpURL = os.Getenv("MCP_URL")
	}

	// Agent name
	// NOTE: This is a zero-config specific default, not handled by SetDefaults()
	// SetDefaults() doesn't set agent names - they come from the map key in config files
	agentName := opts.AgentName
	if agentName == "" {
		agentName = "assistant" // Zero-config default agent name
	}

	// Create LLM config
	llmConfig := &LLMConfig{
		Provider: provider,
		APIKey:   apiKey,
	}

	if opts.Model != "" {
		llmConfig.Model = opts.Model
	}

	if opts.BaseURL != "" {
		llmConfig.BaseURL = opts.BaseURL
	}

	// Set temperature only if explicitly provided (> 0)
	// NOTE: Do NOT set default here - SetDefaults() handles it (defaults to 0.7)
	// Only set if user explicitly provided a value via CLI
	if opts.Temperature > 0 {
		llmConfig.Temperature = &opts.Temperature
	}

	// Set max tokens only if explicitly provided (> 0)
	// NOTE: Do NOT set default here - SetDefaults() handles it (defaults to 4096)
	// Only set if user explicitly provided a value via CLI
	if opts.MaxTokens > 0 {
		llmConfig.MaxTokens = opts.MaxTokens
	}

	// Set thinking configuration if enabled
	// NOTE: Do NOT set default values here that are already handled by SetDefaults().
	// Only set values that are explicitly provided or are zero-config specific overrides.
	// - Thinking.Enabled: Set to true when --thinking flag is used (explicit enablement)
	// - Thinking.BudgetTokens: Only set if explicitly provided (> 0), otherwise let SetDefaults() handle it
	if BoolValue(opts.Thinking, false) {
		llmConfig.Thinking = &ThinkingConfig{
			Enabled: BoolPtr(true), // Explicit enablement when --thinking flag is used
		}
		// Only set budget if explicitly provided via CLI
		// If 0, SetDefaults() will set it to 1024 (see LLMConfig.SetDefaults())
		if opts.ThinkingBudget > 0 {
			llmConfig.Thinking.BudgetTokens = opts.ThinkingBudget
		}
	}

	// Create agent config
	// NOTE: Zero-config specific override: streaming defaults to true (enabled) in zero-config mode.
	// This is different from config file mode where streaming defaults to false.
	// SetDefaults() will NOT override this because it only sets defaults if the field is nil.
	streaming := opts.Streaming
	if streaming == nil {
		streaming = BoolPtr(true) // Zero-config specific: enable streaming by default
	}
	agentConfig := &AgentConfig{
		Name:        agentName,
		LLM:         "default", // SetDefaults() will handle if empty
		Instruction: opts.Instruction,
		Streaming:   streaming, // Zero-config override: true by default
	}

	if opts.Role != "" {
		agentConfig.Prompt = &PromptConfig{
			Role: opts.Role,
		}
	}

	// Create config
	cfg := &Config{
		Name:      "Zero Config Mode",
		Databases: make(map[string]*DatabaseConfig),
		LLMs: map[string]*LLMConfig{
			"default": llmConfig,
		},
		Agents: map[string]*AgentConfig{
			agentName: agentConfig,
		},
		Tools: make(map[string]*ToolConfig),
		Server: ServerConfig{
			// Port: Only set if explicitly provided (> 0)
			// NOTE: Do NOT set default here - SetDefaults() handles it (defaults to 8080)
			Port: opts.Port,
		},
	}

	// Configure persistent storage if specified
	// Supported backends: sqlite, postgres, mysql (inmemory is default, no persistence)
	if opts.Storage != "" && opts.Storage != "inmemory" {
		storageBackend := opts.Storage
		storageDB := opts.StorageDB
		// Get default database config for the specified backend
		dbConfig := DefaultDatabaseConfig(storageBackend)
		if dbConfig != nil {
			// Override with custom DSN/path if provided
			if storageDB != "" {
				if storageBackend == "sqlite" || storageBackend == "sqlite3" {
					dbConfig.Database = storageDB
				} else {
					// For postgres/mysql, StorageDB can be a DSN or just override database name
					dbConfig.Database = storageDB
				}
			}

			// Ensure .hector directory exists for SQLite
			if dbConfig.Driver == "sqlite" {
				dir := filepath.Dir(dbConfig.Database)
				if dir != "" && dir != "." {
					// Use centralized EnsureHectorDir if path contains .hector
					if filepath.Base(dir) == ".hector" {
						basePath := filepath.Dir(dir)
						if basePath == "" || basePath == "." {
							basePath = "."
						}
						_, _ = utils.EnsureHectorDir(basePath)
					} else {
						_ = os.MkdirAll(dir, 0755)
					}
				}
			}

			// Add database config and enable persistence for tasks, sessions, and memory
			cfg.Databases["_default"] = dbConfig
			cfg.Server.Tasks = &TasksConfig{
				Backend:  StorageBackendSQL,
				Database: "_default",
			}
			cfg.Server.Sessions = &SessionsConfig{
				Backend:  StorageBackendSQL,
				Database: "_default",
			}
			// Memory index defaults to keyword (no embedder needed)
			// Users can configure vector index with embedder in config file
			cfg.Server.Memory = &MemoryConfig{
				Backend: "keyword",
			}

			// Auto-enable checkpointing when storage is enabled
			// Checkpoints are stored in session state, so they benefit from persistence
			cfg.Server.Checkpoint = &CheckpointConfig{
				Enabled:    BoolPtr(true),
				Strategy:   "hybrid", // Safe default: event + interval
				AfterTools: BoolPtr(true),
				BeforeLLM:  BoolPtr(true),
				Recovery: &CheckpointRecoveryConfig{
					AutoResume:     BoolPtr(true),  // Auto-resume non-HITL tasks on startup
					AutoResumeHITL: BoolPtr(false), // Require user approval for HITL
					Timeout:        86400,          // 24h expiry for checkpoints (in seconds)
				},
			}
		}
	}

	// Configure observability if enabled
	// Exports traces to OTLP endpoint and enables Prometheus metrics
	if opts.Observe {
		cfg.Server.Observability = &observability.Config{
			Tracing: observability.TracingConfig{
				Enabled:      true,
				Exporter:     "otlp",
				Endpoint:     "localhost:4317",
				SamplingRate: 1.0, // Sample all traces in zero-config mode
			},
			Metrics: observability.MetricsConfig{
				Enabled: true,
			},
		}
	}

	// Configure RAG if docs folder is specified
	// Creates embedded chromem vector store, auto-detected embedder, and document store
	if opts.DocsFolder != "" {
		expandDocsFolder(cfg, agentConfig, opts)
	}

	// Add MCP tool if URL provided
	// NOTE: ToolConfig defaults (Enabled, etc.) are handled by ToolConfig.SetDefaults()
	// Only set values that are explicitly provided here
	if mcpURL != "" {
		cfg.Tools["mcp"] = &ToolConfig{
			Type: ToolTypeMCP,
			URL:  mcpURL,
			// Enabled, Transport, etc. will be set by ToolConfig.SetDefaults()
		}
		agentConfig.Tools = append(agentConfig.Tools, "mcp")
	}

	// Add default local tools if enabled
	// Empty string or "all" enables all tools
	// Comma-separated list enables specific tools
	if opts.Tools != "" {
		defaultTools := GetDefaultToolConfigs()
		enabledTools := parseToolsList(opts.Tools, defaultTools)

		for _, name := range enabledTools {
			if toolCfg, ok := defaultTools[name]; ok {
				cfg.Tools[name] = toolCfg
				agentConfig.Tools = append(agentConfig.Tools, name)
			}
		}
	}

	// Apply defaults
	// IMPORTANT: SetDefaults() is called AFTER setting zero-config specific values.
	// SetDefaults() will:
	// - Set defaults for fields that are nil/empty (not explicitly set)
	// - NOT override fields that are already set (like our streaming=true override)
	// - Handle all standard defaults (temperature, max_tokens, etc.)
	cfg.SetDefaults()

	// Apply tool approval overrides AFTER SetDefaults()
	// This allows CLI flags to override the smart defaults set by SetDefaults()
	ApplyToolApprovalOverrides(cfg, opts.ApproveTools, opts.NoApproveTools)

	return cfg
}

// expandDocsFolder auto-configures RAG from a docs folder path.
//
// Creates:
// - Chromem vector store (embedded, persisted to .hector/vectors/)
// - Embedder (auto-detected from LLM provider or explicit)
// - Document store with directory source
// - Search tool for the agent
//
// This mirrors legacy pkg/config/config.go:expandDocsFolder()
func expandDocsFolder(cfg *Config, agentConfig *AgentConfig, opts ZeroConfig) {
	// Determine embedder model
	embedderModel := opts.EmbedderModel
	if embedderModel == "" {
		embedderModel = detectEmbedderModel(cfg)
	}

	// Determine embedder provider based on model
	embedderProvider := detectEmbedderProvider(embedderModel)

	// Create embedder config
	if cfg.Embedders == nil {
		cfg.Embedders = make(map[string]*EmbedderConfig)
	}
	cfg.Embedders["_rag_embedder"] = &EmbedderConfig{
		Provider: embedderProvider,
		Model:    embedderModel,
	}

	// Create chromem vector store (embedded, zero external deps)
	if cfg.VectorStores == nil {
		cfg.VectorStores = make(map[string]*VectorStoreConfig)
	}
	cfg.VectorStores["_rag_vectors"] = &VectorStoreConfig{
		Type:        "chromem",
		PersistPath: ".hector/vectors", // Persist for fast restarts
		Compress:    true,              // Save disk space
	}

	// Create document store config
	if cfg.DocumentStores == nil {
		cfg.DocumentStores = make(map[string]*DocumentStoreConfig)
	}

	// Determine if watching is enabled (default: true)
	watchEnabled := BoolValue(opts.RAGWatch, true)

	docStoreConfig := &DocumentStoreConfig{
		Source: &DocumentSourceConfig{
			Type: "directory",
			Path: opts.DocsFolder,
			// Default exclusions for common non-document folders
			Exclude: []string{".git", "node_modules", "__pycache__", ".hector", "vendor"},
		},
		Chunking: &ChunkingConfig{
			Strategy: "simple", // Simple for zero-config (fast, predictable)
			Size:     1000,
			Overlap:  200,
		},
		VectorStore:         "_rag_vectors",
		Embedder:            "_rag_embedder",
		Watch:               watchEnabled,
		IncrementalIndexing: true, // Only re-index changed files
		Indexing:            &IndexingConfig{
			// Use defaults (NumCPU workers, standard retry)
		},
		Search: &DocumentSearchConfig{
			TopK:      10,
			Threshold: 0.5,
		},
	}

	// Configure MCP parser if specified (e.g., Docling)
	if opts.MCPParserTool != "" {
		toolNames := parseCommaSeparatedList(opts.MCPParserTool)
		if len(toolNames) > 0 {
			docStoreConfig.MCPParsers = &MCPParserConfig{
				ToolNames:  toolNames,
				Extensions: []string{".pdf", ".docx", ".pptx", ".xlsx", ".html"},
			}
			docStoreConfig.MCPParsers.SetDefaults()
		}
	}

	cfg.DocumentStores["_rag_docs"] = docStoreConfig

	// Assign document store to agent
	ragDocs := []string{"_rag_docs"}
	agentConfig.DocumentStores = &ragDocs

	// Auto-add search tool if not already present
	if cfg.Tools == nil {
		cfg.Tools = make(map[string]*ToolConfig)
	}
	if _, exists := cfg.Tools["search"]; !exists {
		cfg.Tools["search"] = &ToolConfig{
			Type:            ToolTypeFunction,
			Handler:         "search",
			RequireApproval: BoolPtr(false), // Search is read-only, no approval needed
		}
	}

	// Add search tool to agent if not already present
	hasSearchTool := false
	for _, t := range agentConfig.Tools {
		if t == "search" {
			hasSearchTool = true
			break
		}
	}
	if !hasSearchTool {
		agentConfig.Tools = append(agentConfig.Tools, "search")
	}
}

// detectEmbedderModel auto-detects the best embedder model based on LLM provider.
func detectEmbedderModel(cfg *Config) string {
	// Check if we have an LLM config to infer from
	if cfg.LLMs != nil {
		if llmCfg, ok := cfg.LLMs["default"]; ok {
			switch llmCfg.Provider {
			case LLMProviderOpenAI:
				return "text-embedding-3-small" // OpenAI's efficient embedder
			case LLMProviderOllama:
				return "nomic-embed-text" // Popular Ollama embedder
			case LLMProviderAnthropic, LLMProviderGemini:
				// These don't have embeddings, check for OpenAI API key
				if os.Getenv("OPENAI_API_KEY") != "" {
					return "text-embedding-3-small"
				}
			}
		}
	}

	// Fallback: check for OpenAI API key
	if os.Getenv("OPENAI_API_KEY") != "" {
		return "text-embedding-3-small"
	}

	// Final fallback: Ollama (works locally without API key)
	return "nomic-embed-text"
}

// detectEmbedderProvider determines the provider based on model name.
func detectEmbedderProvider(model string) string {
	// OpenAI models
	if strings.HasPrefix(model, "text-embedding") {
		return "openai"
	}

	// Common Ollama embedding models
	ollamaModels := []string{
		"nomic-embed-text",
		"mxbai-embed-large",
		"all-minilm",
		"bge-",
	}
	for _, prefix := range ollamaModels {
		if strings.HasPrefix(model, prefix) || strings.Contains(model, prefix) {
			return "ollama"
		}
	}

	// Default to Ollama (most flexible for local use)
	return "ollama"
}

// parseToolsList parses a tools string into a list of enabled tool names.
// Empty string or "all" returns all available tools.
// Comma-separated list returns only the specified tools.
func parseToolsList(toolsStr string, availableTools map[string]*ToolConfig) []string {
	toolsStr = strings.TrimSpace(toolsStr)
	if toolsStr == "" || strings.ToLower(toolsStr) == "all" {
		// Return all available tools
		result := make([]string, 0, len(availableTools))
		for name := range availableTools {
			result = append(result, name)
		}
		return result
	}

	// Parse comma-separated list
	parts := strings.Split(toolsStr, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name != "" {
			// Validate tool exists
			if _, ok := availableTools[name]; ok {
				result = append(result, name)
			}
		}
	}
	return result
}

// ApplyToolApprovalOverrides applies CLI-specified approval overrides to tool configs.
// This should be called AFTER SetDefaults() to override the smart defaults set by SetDefaults().
func ApplyToolApprovalOverrides(cfg *Config, approveTools, noApproveTools string) {
	if approveTools == "" && noApproveTools == "" {
		return
	}

	// Parse comma-separated tool lists
	approveList := parseCommaSeparatedList(approveTools)
	noApproveList := parseCommaSeparatedList(noApproveTools)

	// Ensure tools map exists
	if cfg.Tools == nil {
		cfg.Tools = make(map[string]*ToolConfig)
	}

	// Apply approval overrides
	for _, toolName := range approveList {
		applyToolApprovalOverride(cfg, toolName, true)
	}

	// Apply no-approval overrides
	for _, toolName := range noApproveList {
		applyToolApprovalOverride(cfg, toolName, false)
	}
}

// applyToolApprovalOverride applies an approval override to a tool config.
// Creates the tool config if it doesn't exist, then sets RequireApproval.
func applyToolApprovalOverride(cfg *Config, toolName string, enable bool) {
	if cfg.Tools[toolName] == nil {
		// Create tool config if it doesn't exist
		defaultConfigs := GetDefaultToolConfigs()
		if defaultCfg, ok := defaultConfigs[toolName]; ok {
			cfg.Tools[toolName] = &ToolConfig{
				Type:    defaultCfg.Type,
				Handler: defaultCfg.Handler,
			}
			cfg.Tools[toolName].SetDefaults()
		} else {
			// Unknown tool, create minimal config
			cfg.Tools[toolName] = &ToolConfig{
				Type: ToolTypeFunction,
			}
			cfg.Tools[toolName].SetDefaults()
		}
	}
	cfg.Tools[toolName].RequireApproval = BoolPtr(enable)
}

// parseCommaSeparatedList parses a comma-separated string into a list of trimmed strings.
func parseCommaSeparatedList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
