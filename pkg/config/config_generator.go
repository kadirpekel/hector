// Hector Configuration Generator
//
// This file implements the unified configuration generation system.
// Every `hector serve` invocation produces a configuration file.
// There is no distinction between "zero-config" and "config" modes.

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/verikod/hector/pkg/utils"
	"gopkg.in/yaml.v3"
)

// SkillFrontmatter represents the YAML frontmatter in SKILL.md.
type SkillFrontmatter struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	AllowedTools []string `yaml:"allowed-tools"`
}

// CLIOptions represents the command-line options provided by the user.
// Only fields that are explicitly set will be included in the generated config.
type CLIOptions struct {
	// LLM options
	Provider    string
	Model       string
	APIKey      string
	BaseURL     string
	Temperature *float64 // Pointer to distinguish "not set" from "set to 0"
	MaxTokens   *int

	// Thinking options
	Thinking       *bool
	ThinkingBudget int

	// Streaming
	Streaming *bool

	// Agent options
	Instruction string
	Role        string

	// Tool options
	MCPURL         string
	Tools          string
	ApproveTools   string
	NoApproveTools string

	// RAG options
	DocsFolder       string
	RAGWatch         *bool
	MCPParserTool    string
	VectorType       string
	VectorHost       string
	VectorAPIKey     string
	EmbedderProvider string
	EmbedderModel    string
	EmbedderURL      string
	IncludeContext   *bool

	// Skill file (auto-detected)
	SkillFile string
}

// EnvVarInfo tracks environment variables that should be documented.
type EnvVarInfo struct {
	Name     string
	Required bool
	Value    string // Current value (for checking if set)
}

// GeneratorResult contains the output of config generation.
type GeneratorResult struct {
	ConfigYAML []byte
	AppConfig  *AppConfig // Structured config for internal use
	EnvVars    []EnvVarInfo
	ConfigPath string
	SkillUsed  bool
	CreatedNew bool
}

// GenerateLeanConfig creates a minimal configuration from CLI options.
// Only explicitly provided values are included in the output.
func GenerateLeanConfig(opts CLIOptions, configPath string) (*GeneratorResult, error) {
	result := &GeneratorResult{
		ConfigPath: configPath,
		EnvVars:    []EnvVarInfo{},
	}

	// 1. Initialize AppConfig struct
	cfg := &AppConfig{
		LLMs:           make(map[string]*LLMConfig),
		Agents:         make(map[string]*AgentConfig),
		Tools:          make(map[string]*ToolConfig),
		VectorStores:   make(map[string]*VectorStoreConfig),
		Embedders:      make(map[string]*EmbedderConfig),
		DocumentStores: make(map[string]*DocumentStoreConfig),
	}

	// Parse skill frontmatter early if available
	var skillFM *SkillFrontmatter
	if opts.SkillFile != "" {
		var parseErr error
		skillFM, parseErr = parseSkillFrontmatter(opts.SkillFile)
		if parseErr != nil {
			slog.Warn("Failed to parse SKILL.md frontmatter, tool constraints will not be applied",
				"path", opts.SkillFile, "error", parseErr)
		}

		// Map AllowedTools to CLI opts.Tools if CLI tools not strictly set (defaulting to "all" or empty)
		if skillFM != nil && len(skillFM.AllowedTools) > 0 && (opts.Tools == "" || opts.Tools == "all") {
			mappedTools := make([]string, 0)
			hasTextEditor := false
			hasBash := false

			for _, t := range skillFM.AllowedTools {
				tLower := strings.ToLower(strings.TrimSpace(t))
				if strings.HasPrefix(tLower, "bash") {
					if !hasBash {
						mappedTools = append(mappedTools, "bash")
						hasBash = true
					}
				} else if tLower == "read" || tLower == "editor" || tLower == "edit" || tLower == "write" {
					if !hasTextEditor {
						mappedTools = append(mappedTools, "text_editor")
						hasTextEditor = true
					}
				}
			}
			if len(mappedTools) > 0 {
				opts.Tools = strings.Join(mappedTools, ",")
			}
		}
	}

	// 2. Build LLM Config
	llmCfg := buildLLMConfig(opts, &result.EnvVars)
	if llmCfg != nil {
		cfg.LLMs["default"] = llmCfg
	}

	// 3. Build Agent Config
	agentCfg := buildAgentConfig(opts)
	if agentCfg != nil {
		// Ensure LLM is set
		if agentCfg.LLM == "" {
			agentCfg.LLM = "default"
		}
		cfg.Agents["assistant"] = agentCfg
	} else {
		// Minimal agent
		cfg.Agents["assistant"] = &AgentConfig{
			LLM: "default",
		}
	}

	// 4. Build Tools
	toolsMap := buildToolsConfig(opts, &result.EnvVars)
	for name, tool := range toolsMap {
		cfg.Tools[name] = tool
	}
	// Add tools to agent
	if len(toolsMap) > 0 {
		assistant := cfg.Agents["assistant"]
		for name := range toolsMap {
			assistant.Tools = append(assistant.Tools, name)
		}
	}

	// 5. Handle RAG/Document Stores
	if opts.DocsFolder != "" {
		buildRAGConfig(opts, cfg, &result.EnvVars)
	}

	// 6. Handle SKILL.md
	if opts.SkillFile != "" {
		result.SkillUsed = true
		assistant := cfg.Agents["assistant"]
		// Calculate relative path
		configDir := filepath.Dir(configPath)
		relPath, err := filepath.Rel(configDir, opts.SkillFile)
		if err != nil {
			relPath = opts.SkillFile
		}
		assistant.InstructionFile = relPath

		if skillFM != nil {
			if skillFM.Name != "" {
				assistant.Name = skillFM.Name
			}
			if skillFM.Description != "" {
				assistant.Description = skillFM.Description
			}
		}
	}

	// 7. Store struct in result
	result.AppConfig = cfg

	// 8. Generate YAML from AppConfig
	// Round-trip through JSON to respect struct tags (omitempty, field names)
	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal app config: %w", err)
	}
	var configMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &configMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal app config to map: %w", err)
	}

	// Marshal to YAML
	var buf bytes.Buffer
	buf.WriteString("# Auto-generated by Hector\n")
	buf.WriteString("# Edit this file to customize your agent.\n")
	if result.SkillUsed {
		buf.WriteString("# System prompt is loaded from SKILL.md\n")
	}
	buf.WriteString("\n")

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(configMap); err != nil {
		return nil, fmt.Errorf("failed to marshal config to yaml: %w", err)
	}
	encoder.Close()

	result.ConfigYAML = buf.Bytes()
	return result, nil
}

// buildLLMConfig creates LLM configuration from CLI options.
func buildLLMConfig(opts CLIOptions, envVars *[]EnvVarInfo) *LLMConfig {
	llm := &LLMConfig{}

	// Provider
	provider := opts.Provider
	if provider == "" {
		provider = string(detectProviderFromEnv())
	}
	if provider != "" {
		llm.Provider = LLMProvider(provider)
	}

	// Model
	if opts.Model != "" {
		llm.Model = opts.Model
	}

	// API Key
	if opts.APIKey != "" {
		llm.APIKey = opts.APIKey
	} else {
		envVarName := getAPIKeyEnvVar(LLMProvider(provider))
		if envVarName != "" {
			if os.Getenv(envVarName) != "" {
				llm.APIKey = fmt.Sprintf("${%s}", envVarName)
			}
			*envVars = append(*envVars, EnvVarInfo{
				Name:     envVarName,
				Required: LLMProvider(provider) != LLMProviderOllama,
				Value:    os.Getenv(envVarName),
			})
		}
	}

	// Base URL
	if opts.BaseURL != "" {
		llm.BaseURL = opts.BaseURL
	}

	// Temperature
	if opts.Temperature != nil {
		llm.Temperature = opts.Temperature
	}

	// Max Tokens
	if opts.MaxTokens != nil {
		llm.MaxTokens = *opts.MaxTokens
	}

	// Thinking
	if opts.Thinking != nil && *opts.Thinking {
		llm.Thinking = &ThinkingConfig{
			Enabled: BoolPtr(true),
		}
		if opts.ThinkingBudget > 0 {
			llm.Thinking.BudgetTokens = opts.ThinkingBudget
		}
	}

	// Return nil if empty? No, we likely always want at least basic config if provider is set.
	return llm
}

// buildAgentConfig creates agent configuration from CLI options.
func buildAgentConfig(opts CLIOptions) *AgentConfig {
	agent := &AgentConfig{
		LLM: "default",
	}
	hasContent := false

	if opts.Instruction != "" {
		agent.Instruction = opts.Instruction
		hasContent = true
	}

	if opts.Streaming != nil {
		agent.Streaming = opts.Streaming
		hasContent = true
	}

	if opts.Role != "" {
		agent.Prompt = &PromptConfig{
			Role: opts.Role,
		}
		hasContent = true
	}

	if !hasContent {
		// Return basic default
		return agent
	}

	return agent
}

// buildToolsConfig creates tools configuration from CLI options.
func buildToolsConfig(opts CLIOptions, envVars *[]EnvVarInfo) map[string]*ToolConfig {
	tools := make(map[string]*ToolConfig)

	// Parse approval override lists
	approveList := parseCommaSeparated(opts.ApproveTools)
	noApproveList := parseCommaSeparated(opts.NoApproveTools)
	approveSet := make(map[string]bool)
	noApproveSet := make(map[string]bool)
	for _, name := range approveList {
		approveSet[name] = true
	}
	for _, name := range noApproveList {
		noApproveSet[name] = true
	}

	// MCP URL
	if opts.MCPURL != "" {
		tools["mcp"] = &ToolConfig{
			Type: "mcp",
			URL:  opts.MCPURL,
		}
	} else if os.Getenv("MCP_URL") != "" {
		tools["mcp"] = &ToolConfig{
			Type: "mcp",
			URL:  "${MCP_URL}",
		}
		*envVars = append(*envVars, EnvVarInfo{
			Name:  "MCP_URL",
			Value: os.Getenv("MCP_URL"),
		})
	}

	// Built-in tools
	if opts.Tools != "" {
		defaultTools := GetDefaultToolConfigs()
		enabledTools := parseToolsList(opts.Tools, defaultTools)
		for _, name := range enabledTools {
			if toolCfg, ok := defaultTools[name]; ok {
				// Clone config
				newTool := &ToolConfig{
					Type:    toolCfg.Type,
					Handler: toolCfg.Handler,
				}

				// Apply approval overrides
				if approveSet[name] {
					newTool.RequireApproval = BoolPtr(true)
				} else if noApproveSet[name] {
					newTool.RequireApproval = BoolPtr(false)
				}

				if name == "web_search" {
					if os.Getenv("TAVILY_API_KEY") != "" {
						*envVars = append(*envVars, EnvVarInfo{
							Name:  "TAVILY_API_KEY",
							Value: os.Getenv("TAVILY_API_KEY"),
						})
					} else {
						*envVars = append(*envVars, EnvVarInfo{
							Name:     "TAVILY_API_KEY",
							Required: false,
						})
					}
				}
				tools[name] = newTool
			}
		}
	}

	return tools
}

// buildRAGConfig creates RAG-related configurations into AppConfig.
func buildRAGConfig(opts CLIOptions, cfg *AppConfig, envVars *[]EnvVarInfo) {
	// Vector store
	vectorType := opts.VectorType
	if vectorType == "" {
		vectorType = "chromem"
	}

	vs := &VectorStoreConfig{
		Type: vectorType,
	}
	if vectorType == "chromem" {
		vs.PersistPath = filepath.Join(".", utils.DefaultHectorDir, "vectors")
	}
	if opts.VectorHost != "" {
		vs.Host = opts.VectorHost
	} // Note: APIKey is not in base VectorStoreConfig? Need to check. Assuming basic fields.

	cfg.VectorStores["_rag_vectors"] = vs

	// Embedder
	embedderProvider := opts.EmbedderProvider
	if embedderProvider == "" {
		if os.Getenv("OPENAI_API_KEY") != "" {
			embedderProvider = "openai"
		} else {
			embedderProvider = "ollama"
		}
	}

	emb := &EmbedderConfig{
		Provider: embedderProvider,
	}
	if opts.EmbedderModel != "" {
		emb.Model = opts.EmbedderModel
	}
	if embedderProvider == "openai" {
		emb.APIKey = "${OPENAI_API_KEY}"
		*envVars = append(*envVars, EnvVarInfo{
			Name:  "OPENAI_API_KEY",
			Value: os.Getenv("OPENAI_API_KEY"),
		})
	}
	cfg.Embedders["_rag_embedder"] = emb

	// Document store
	ds := &DocumentStoreConfig{
		Source: &DocumentSourceConfig{
			Type: "blob",
			Blob: &BlobSourceConfig{
				URL: "file://" + opts.DocsFolder,
			},
		},
		VectorStore: "_rag_vectors",
		Embedder:    "_rag_embedder",
	}
	if opts.RAGWatch != nil {
		ds.Watch = *opts.RAGWatch
	}
	cfg.DocumentStores["_rag_docs"] = ds
}

// parseCommaSeparated parses a comma-separated string into a slice.
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// getAPIKeyEnvVar returns the environment variable name for an LLM provider.
func getAPIKeyEnvVar(provider LLMProvider) string {
	switch provider {
	case LLMProviderOpenAI:
		return "OPENAI_API_KEY"
	case LLMProviderAnthropic:
		return "ANTHROPIC_API_KEY"
	case LLMProviderGemini:
		return "GEMINI_API_KEY"
	case LLMProviderDeepSeek:
		return "DEEPSEEK_API_KEY"
	case LLMProviderGroq:
		return "GROQ_API_KEY"
	case LLMProviderMistral:
		return "MISTRAL_API_KEY"
	case LLMProviderCohere:
		return "COHERE_API_KEY"
	default:
		return ""
	}
}

// GenerateEnvExample creates a .env.example file content.
func GenerateEnvExample(envVars []EnvVarInfo) []byte {
	if len(envVars) == 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.WriteString("# Auto-generated by Hector\n")
	buf.WriteString("# Copy this file to .env and fill in the values\n\n")

	seen := make(map[string]bool)
	for _, env := range envVars {
		if seen[env.Name] {
			continue
		}
		seen[env.Name] = true

		if env.Required {
			buf.WriteString(fmt.Sprintf("%s=\n", env.Name))
		} else {
			buf.WriteString(fmt.Sprintf("# %s=\n", env.Name))
		}
	}

	return buf.Bytes()
}

// EnsureConfigExists ensures a configuration file exists, creating one if needed.
// Uses GenerateLeanConfig for smart env detection (provider, SKILL.md, etc.)
// so the file is immediately useful for new users.
func EnsureConfigExists(opts CLIOptions, configPath string) (*GeneratorResult, error) {
	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		// Config exists, don't overwrite
		return &GeneratorResult{
			ConfigPath: configPath,
			CreatedNew: false,
		}, nil
	}

	// Check for SKILL.md in workspace root
	if opts.SkillFile == "" {
		configDir := filepath.Dir(configPath)
		workspaceRoot := configDir
		if filepath.Base(configDir) == ".hector" {
			workspaceRoot = filepath.Dir(configDir)
		}
		skillPath := filepath.Join(workspaceRoot, "SKILL.md")
		if _, err := os.Stat(skillPath); err == nil {
			opts.SkillFile = skillPath
		}
	}

	// Generate lean config with env detection
	result, err := GenerateLeanConfig(opts, configPath)
	if err != nil {
		return nil, err
	}
	result.CreatedNew = true

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, result.ConfigYAML, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	// Generate .env.example if .env doesn't exist
	workspaceRoot := filepath.Dir(configPath)
	if filepath.Base(filepath.Dir(configPath)) == ".hector" {
		workspaceRoot = filepath.Dir(filepath.Dir(configPath))
	}
	envPath := filepath.Join(workspaceRoot, ".env")
	envExamplePath := filepath.Join(workspaceRoot, ".env.example")

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if len(result.EnvVars) > 0 {
			envContent := GenerateEnvExample(result.EnvVars)
			if envContent != nil {
				if _, err := os.Stat(envExamplePath); os.IsNotExist(err) {
					_ = os.WriteFile(envExamplePath, envContent, 0644)
				}
			}
		}
	}

	return result, nil
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

// parseSkillFrontmatter reads SKILL.md and parses the YAML frontmatter.
func parseSkillFrontmatter(path string) (*SkillFrontmatter, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Find the frontmatter block
	s := string(content)
	if !strings.HasPrefix(s, "---") {
		return nil, fmt.Errorf("no frontmatter found")
	}

	parts := strings.SplitN(s, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid frontmatter format")
	}

	var frontmatter SkillFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err != nil {
		return nil, err
	}

	return &frontmatter, nil
}
