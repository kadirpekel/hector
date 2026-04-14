package config_test

import (
	"testing"

	"github.com/verikod/hector/pkg/config"
)

// =============================================================================
// ServerConfig Tests
// =============================================================================

func TestServerConfig_SetDefaults(t *testing.T) {
	cfg := &config.ServerConfig{}
	cfg.SetDefaults()

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %v, want 0.0.0.0", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %v, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %v, want text", cfg.LogFormat)
	}
	if cfg.Database == "" {
		t.Error("Database should have default value")
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.ServerConfig
		wantErr bool
	}{
		{
			name: "valid minimal config",
			cfg: &config.ServerConfig{
				Port:      8080,
				LogLevel:  "info",
				LogFormat: "text",
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			cfg: &config.ServerConfig{
				Port:      0,
				LogLevel:  "info",
				LogFormat: "text",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			cfg: &config.ServerConfig{
				Port:      70000,
				LogLevel:  "info",
				LogFormat: "text",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			cfg: &config.ServerConfig{
				Port:      8080,
				LogLevel:  "invalid",
				LogFormat: "text",
			},
			wantErr: true,
		},
		{
			name: "invalid log format",
			cfg: &config.ServerConfig{
				Port:      8080,
				LogLevel:  "info",
				LogFormat: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerConfig_Address(t *testing.T) {
	cfg := &config.ServerConfig{
		Host: "127.0.0.1",
		Port: 9000,
	}

	addr := cfg.Address()
	if addr != "127.0.0.1:9000" {
		t.Errorf("Address() = %v, want 127.0.0.1:9000", addr)
	}
}

// =============================================================================
// AuthConfig Tests
// =============================================================================

func TestAuthConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.AuthConfig
		expected bool
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: false,
		},
		{
			name:     "empty config",
			cfg:      &config.AuthConfig{},
			expected: false,
		},
		{
			name: "secret only",
			cfg: &config.AuthConfig{
				Secret: "test-secret",
			},
			expected: true,
		},
		{
			name: "jwks configured",
			cfg: &config.AuthConfig{
				JWKSURL:  "https://example.com/.well-known/jwks.json",
				Issuer:   "https://example.com",
				Audience: "my-api",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.IsEnabled()
			if result != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.AuthConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: false,
		},
		{
			name: "partial jwks - missing issuer",
			cfg: &config.AuthConfig{
				Enabled: true,
				JWKSURL: "https://example.com/.well-known/jwks.json",
			},
			wantErr: true,
		},
		{
			name: "complete jwks config",
			cfg: &config.AuthConfig{
				Enabled:  true,
				JWKSURL:  "https://example.com/.well-known/jwks.json",
				Issuer:   "https://example.com",
				Audience: "my-api",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// AppConfig Tests
// =============================================================================

func TestAppConfig_SetDefaults_InitializesMaps(t *testing.T) {
	cfg := &config.AppConfig{}
	cfg.SetDefaults()

	if cfg.LLMs == nil {
		t.Error("LLMs should be initialized")
	}
	if cfg.Tools == nil {
		t.Error("Tools should be initialized")
	}
	if cfg.Agents == nil {
		t.Error("Agents should be initialized")
	}
	if cfg.Guardrails == nil {
		t.Error("Guardrails should be initialized")
	}
	if cfg.VectorStores == nil {
		t.Error("VectorStores should be initialized")
	}
	if cfg.Embedders == nil {
		t.Error("Embedders should be initialized")
	}
	if cfg.DocumentStores == nil {
		t.Error("DocumentStores should be initialized")
	}
}

func TestAppConfig_GetAgent(t *testing.T) {
	cfg := &config.AppConfig{
		Agents: map[string]*config.AgentConfig{
			"test-agent": {Instruction: "Test instruction"},
		},
	}

	t.Run("existing agent", func(t *testing.T) {
		agent, ok := cfg.GetAgent("test-agent")
		if !ok {
			t.Error("Should find existing agent")
		}
		if agent.Instruction != "Test instruction" {
			t.Error("Should return correct agent config")
		}
	})

	t.Run("nonexistent agent", func(t *testing.T) {
		_, ok := cfg.GetAgent("nonexistent")
		if ok {
			t.Error("Should not find nonexistent agent")
		}
	})
}

func TestAppConfig_ListAgents(t *testing.T) {
	cfg := &config.AppConfig{
		Agents: map[string]*config.AgentConfig{
			"agent-a": {},
			"agent-b": {},
			"agent-c": {},
		},
	}

	names := cfg.ListAgents()
	if len(names) != 3 {
		t.Errorf("Expected 3 agents, got %d", len(names))
	}

	// Check all agents are present (order not guaranteed)
	found := make(map[string]bool)
	for _, name := range names {
		found[name] = true
	}
	for _, expected := range []string{"agent-a", "agent-b", "agent-c"} {
		if !found[expected] {
			t.Errorf("Missing agent: %s", expected)
		}
	}
}

func TestValidateAppConfig_ValidReferences(t *testing.T) {
	cfg := &config.AppConfig{
		LLMs: map[string]*config.LLMConfig{
			"my-llm": {Provider: config.LLMProviderOllama, Model: "qwen3"}, // Ollama doesn't require API key
		},
		Tools: map[string]*config.ToolConfig{
			"my-tool": {Type: config.ToolTypeMCP, URL: "http://localhost:8080"},
		},
		Agents: map[string]*config.AgentConfig{
			"my-agent": {
				LLM:   "my-llm",
				Tools: []string{"my-tool"},
			},
		},
	}

	if err := config.ValidateAppConfig(cfg); err != nil {
		t.Errorf("Config with valid references should pass: %v", err)
	}
}

func TestValidateAppConfig_InvalidLLMReference(t *testing.T) {
	cfg := &config.AppConfig{
		LLMs: map[string]*config.LLMConfig{
			"valid-llm": {Provider: config.LLMProviderOllama, Model: "qwen3"},
		},
		Agents: map[string]*config.AgentConfig{
			"my-agent": {
				LLM: "nonexistent-llm",
			},
		},
	}

	if err := config.ValidateAppConfig(cfg); err == nil {
		t.Error("Reference to undefined LLM should cause validation error")
	}
}

func TestValidateAppConfig_InvalidToolReference(t *testing.T) {
	cfg := &config.AppConfig{
		LLMs: map[string]*config.LLMConfig{
			"default": {Provider: config.LLMProviderOllama, Model: "qwen3"},
		},
		Agents: map[string]*config.AgentConfig{
			"my-agent": {
				LLM:   "default",
				Tools: []string{"undefined-tool"},
			},
		},
	}

	if err := config.ValidateAppConfig(cfg); err == nil {
		t.Error("Reference to undefined tool should cause validation error")
	}
}

func TestValidateAppConfig_InvalidSubAgentReference(t *testing.T) {
	cfg := &config.AppConfig{
		LLMs: map[string]*config.LLMConfig{
			"default": {Provider: config.LLMProviderOllama, Model: "qwen3"},
		},
		Agents: map[string]*config.AgentConfig{
			"parent": {
				LLM:       "default",
				SubAgents: []string{"nonexistent-child"},
			},
		},
	}

	if err := config.ValidateAppConfig(cfg); err == nil {
		t.Error("Reference to undefined sub-agent should cause validation error")
	}
}

func TestValidateAppConfig_InvalidGuardrailsReference(t *testing.T) {
	cfg := &config.AppConfig{
		LLMs: map[string]*config.LLMConfig{
			"default": {Provider: config.LLMProviderOllama, Model: "qwen3"},
		},
		Agents: map[string]*config.AgentConfig{
			"my-agent": {
				LLM:        "default",
				Guardrails: "nonexistent-guardrails",
			},
		},
	}

	if err := config.ValidateAppConfig(cfg); err == nil {
		t.Error("Reference to undefined guardrails should cause validation error")
	}
}
