package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AppConfig is the application-level configuration.
// This contains ONLY functional business logic (agents, tools, LLMs).
// Operational settings (database, port, logging) are passed via CLI/env.
//
// Example config.yaml:
//
//	version: "2"
//	name: my-app
//	description: My AI application
//
//	llms:
//	  default:
//	    provider: anthropic
//	    model: claude-sonnet-4
//	    api_key: ${ANTHROPIC_API_KEY}
//
//	agents:
//	  assistant:
//	    llm: default
//	    instruction: You are a helpful assistant.
type AppConfig struct {
	// Metadata
	Version     string `yaml:"version,omitempty" json:"version,omitempty"`
	Name        string `yaml:"name,omitempty" json:"name,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Resources (all app-scoped)
	LLMs           map[string]*LLMConfig           `yaml:"llms,omitempty" json:"llms,omitempty"`
	Tools          map[string]*ToolConfig          `yaml:"tools,omitempty" json:"tools,omitempty"`
	Agents         map[string]*AgentConfig         `yaml:"agents,omitempty" json:"agents,omitempty"`
	Guardrails     map[string]*GuardrailsConfig    `yaml:"guardrails,omitempty" json:"guardrails,omitempty"`
	VectorStores   map[string]*VectorStoreConfig   `yaml:"vector_stores,omitempty" json:"vector_stores,omitempty"`
	Embedders      map[string]*EmbedderConfig      `yaml:"embedders,omitempty" json:"embedders,omitempty"`
	DocumentStores map[string]*DocumentStoreConfig `yaml:"document_stores,omitempty" json:"document_stores,omitempty"`

	// Defaults
	Defaults *DefaultsConfig `yaml:"defaults,omitempty" json:"defaults,omitempty"`

	// NOTE: The following fields are REMOVED (moved to server config):
	// - Database      (use --database CLI flag)
	// - Server        (use --port, --host CLI flags)
	// - Logger        (use --log-level, --log-format CLI flags)
	// - Observability (use --metrics, --tracing-endpoint CLI flags)
	// - RateLimiting  (TBD: might be app-level or server-level)
	// - Memory        (TBD: might be app-level or server-level)
	// - Checkpoint    (TBD: might be app-level or server-level)
}

// DefaultsConfig provides default values for agent configurations.
type DefaultsConfig struct {
	LLM string `yaml:"llm,omitempty" json:"llm,omitempty"`
}

// ParseAppConfigJSON parses an app configuration from JSON bytes.
func ParseAppConfigJSON(data []byte) (*AppConfig, error) {
	// Expand environment variables
	expandedData := os.ExpandEnv(string(data))

	var cfg AppConfig
	if err := json.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Apply defaults and validate
	cfg.SetDefaults()
	if err := ValidateAppConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadAppConfig loads an app configuration from a YAML file.
func LoadAppConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	expandedData := os.ExpandEnv(string(data))

	var cfg AppConfig
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults and validate
	cfg.SetDefaults()
	if err := ValidateAppConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SaveAppConfig saves an app configuration to a YAML file.
func SaveAppConfig(path string, cfg *AppConfig) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SetDefaults applies default values to the config.
func (c *AppConfig) SetDefaults() {
	// Initialize maps if nil
	if c.LLMs == nil {
		c.LLMs = make(map[string]*LLMConfig)
	}
	if c.Tools == nil {
		c.Tools = make(map[string]*ToolConfig)
	}
	if c.Agents == nil {
		c.Agents = make(map[string]*AgentConfig)
	}
	if c.Guardrails == nil {
		c.Guardrails = make(map[string]*GuardrailsConfig)
	}
	if c.VectorStores == nil {
		c.VectorStores = make(map[string]*VectorStoreConfig)
	}
	if c.Embedders == nil {
		c.Embedders = make(map[string]*EmbedderConfig)
	}
	if c.DocumentStores == nil {
		c.DocumentStores = make(map[string]*DocumentStoreConfig)
	}

	// Apply defaults to each resource
	for _, llm := range c.LLMs {
		llm.SetDefaults()
	}
	for _, tool := range c.Tools {
		tool.SetDefaults()
	}
	for _, agent := range c.Agents {
		agent.SetDefaults(c.Defaults)
	}
	for _, guardrail := range c.Guardrails {
		guardrail.SetDefaults()
	}
	for _, vs := range c.VectorStores {
		vs.SetDefaults()
	}
	for _, emb := range c.Embedders {
		emb.SetDefaults()
	}
	for _, ds := range c.DocumentStores {
		ds.SetDefaults()
	}
}

// ValidateAppConfig validates the app configuration comprehensively.
// This is the SINGLE SOURCE OF TRUTH for all config validation.
// It validates both individual config types AND cross-references.
func ValidateAppConfig(cfg *AppConfig) error {
	// =========================================
	// Phase 1: Validate individual config types
	// =========================================

	// Validate all LLM configs
	for name, llm := range cfg.LLMs {
		if llm == nil {
			continue
		}
		if err := llm.Validate(); err != nil {
			return fmt.Errorf("llm %q: %w", name, err)
		}
	}

	// Validate all tool configs
	for name, tool := range cfg.Tools {
		if tool == nil {
			continue
		}
		if err := tool.Validate(); err != nil {
			return fmt.Errorf("tool %q: %w", name, err)
		}
	}

	// Validate all embedder configs
	for name, emb := range cfg.Embedders {
		if emb == nil {
			continue
		}
		if err := emb.Validate(); err != nil {
			return fmt.Errorf("embedder %q: %w", name, err)
		}
	}

	// Validate all vector store configs
	for name, vs := range cfg.VectorStores {
		if vs == nil {
			continue
		}
		if err := vs.Validate(); err != nil {
			return fmt.Errorf("vector_store %q: %w", name, err)
		}
	}

	// Validate all document store configs
	for name, ds := range cfg.DocumentStores {
		if ds == nil {
			continue
		}
		if err := ds.Validate(); err != nil {
			return fmt.Errorf("document_store %q: %w", name, err)
		}
	}

	// Validate all guardrail configs
	for name, gr := range cfg.Guardrails {
		if gr == nil {
			continue
		}
		if err := gr.Validate(); err != nil {
			return fmt.Errorf("guardrails %q: %w", name, err)
		}
	}

	// Validate all agent configs (do this after tools/LLMs since agents may reference them)
	for name, agent := range cfg.Agents {
		if agent == nil {
			continue
		}
		if err := agent.Validate(); err != nil {
			return fmt.Errorf("agent %q: %w", name, err)
		}
	}

	// =========================================
	// Phase 2: Validate cross-references
	// =========================================

	// Validate that agents reference valid LLMs
	for agentName, agent := range cfg.Agents {
		if agent.LLM != "" {
			if _, ok := cfg.LLMs[agent.LLM]; !ok {
				return fmt.Errorf("agent %q references unknown LLM %q", agentName, agent.LLM)
			}
		}

		// Validate that agents reference valid tools
		for _, toolName := range agent.Tools {
			if _, ok := cfg.Tools[toolName]; !ok {
				return fmt.Errorf("agent %q references unknown tool %q", agentName, toolName)
			}
		}

		// Validate that agents reference valid guardrails
		if agent.Guardrails != "" {
			if _, ok := cfg.Guardrails[agent.Guardrails]; !ok {
				return fmt.Errorf("agent %q references unknown guardrail %q", agentName, agent.Guardrails)
			}
		}

		// Validate sub-agents exist
		for _, subAgentName := range agent.SubAgents {
			if _, ok := cfg.Agents[subAgentName]; !ok {
				return fmt.Errorf("agent %q references unknown sub-agent %q", agentName, subAgentName)
			}
		}

		// Validate agent tools exist
		for _, agentToolName := range agent.AgentTools {
			if _, ok := cfg.Agents[agentToolName]; !ok {
				return fmt.Errorf("agent %q references unknown agent tool %q", agentName, agentToolName)
			}
		}
	}

	// Validate document stores reference valid vector stores and embedders
	for dsName, ds := range cfg.DocumentStores {
		if ds.VectorStore != "" {
			if _, ok := cfg.VectorStores[ds.VectorStore]; !ok {
				return fmt.Errorf("document store %q references unknown vector store %q", dsName, ds.VectorStore)
			}
		}
		if ds.Embedder != "" {
			if _, ok := cfg.Embedders[ds.Embedder]; !ok {
				return fmt.Errorf("document store %q references unknown embedder %q", dsName, ds.Embedder)
			}
		}
	}

	return nil
}

// ListAgents returns a sorted list of agent names.
func (c *AppConfig) ListAgents() []string {
	names := make([]string, 0, len(c.Agents))
	for name := range c.Agents {
		names = append(names, name)
	}
	return names
}

// GetAgent retrieves an agent by name.
func (c *AppConfig) GetAgent(name string) (*AgentConfig, bool) {
	agent, ok := c.Agents[name]
	return agent, ok
}
