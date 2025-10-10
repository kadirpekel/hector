package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromString(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		wantErr  bool
		validate func(t *testing.T, config *Config)
	}{
		{
			name: "valid_minimal_config",
			yaml: `
agents:
  test-agent:
    name: "Test Agent"
    llm: "test-llm"

llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "https://api.openai.com/v1"
    api_key: "sk-test-key"
`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if len(config.Agents) != 1 {
					t.Errorf("Expected 1 agent, got %d", len(config.Agents))
				}
				if len(config.LLMs) != 1 {
					t.Errorf("Expected 1 LLM, got %d", len(config.LLMs))
				}

				if agent, exists := config.Agents["test-agent"]; exists {
					if agent.Name != "Test Agent" {
						t.Errorf("Agent name = %v, want %v", agent.Name, "Test Agent")
					}
					if agent.LLM != "test-llm" {
						t.Errorf("Agent LLM = %v, want %v", agent.LLM, "test-llm")
					}
				} else {
					t.Error("Expected agent 'test-agent' to exist")
				}

				if llm, exists := config.LLMs["test-llm"]; exists {
					if llm.Type != "openai" {
						t.Errorf("LLM type = %v, want %v", llm.Type, "openai")
					}
					if llm.Model != "gpt-4o" {
						t.Errorf("LLM model = %v, want %v", llm.Model, "gpt-4o")
					}
				} else {
					t.Error("Expected LLM 'test-llm' to exist")
				}
			},
		},
		{
			name: "valid_complete_config",
			yaml: `
version: "1.0"
name: "Test Config"
description: "Test configuration"

agents:
  test-agent:
    name: "Test Agent"
    description: "A test agent"
    llm: "test-llm"
    tools: ["execute_command", "todo_write"]

llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "https://api.openai.com/v1"
    api_key: "sk-test-key"
    temperature: 0.7
    max_tokens: 4000

global:
  logging:
    level: "info"
    format: "text"
    output: "stdout"
  performance:
    max_concurrency: 8
    timeout: "30m"
  a2a_server:
    enabled: true
    host: "0.0.0.0"
    port: 8080
`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if config.Version != "1.0" {
					t.Errorf("Config version = %v, want %v", config.Version, "1.0")
				}
				if config.Name != "Test Config" {
					t.Errorf("Config name = %v, want %v", config.Name, "Test Config")
				}

				if config.Global.Logging.Level != "info" {
					t.Errorf("Logging level = %v, want %v", config.Global.Logging.Level, "info")
				}
				if config.Global.Performance.MaxConcurrency != 8 {
					t.Errorf("Max concurrency = %v, want %v", config.Global.Performance.MaxConcurrency, 8)
				}
				if config.Global.A2AServer.Port != 8080 {
					t.Errorf("A2A server port = %v, want %v", config.Global.A2AServer.Port, 8080)
				}
			},
		},
		{
			name: "config_with_environment_variables",
			yaml: `
agents:
  test-agent:
    name: "Test Agent"
    llm: "test-llm"

llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "${OPENAI_HOST:-https://api.openai.com/v1}"
    api_key: "${OPENAI_API_KEY:-sk-test-key}"
`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if llm, exists := config.LLMs["test-llm"]; exists {
					// Should expand environment variables
					if llm.Host == "${OPENAI_HOST:-https://api.openai.com/v1}" {
						t.Error("Environment variable should be expanded")
					}
					if llm.APIKey == "${OPENAI_API_KEY}" {
						t.Error("Environment variable should be expanded")
					}
				} else {
					t.Error("Expected LLM 'test-llm' to exist")
				}
			},
		},
		{
			name: "invalid_yaml_syntax",
			yaml: `
agents:
  test-agent:
    name: "Test Agent"
    llm: "test-llm"
    invalid: [unclosed bracket
`,
			wantErr: true,
		},
		{
			name:    "empty_yaml",
			yaml:    "",
			wantErr: true, // Empty YAML will fail validation due to missing API key
		},
		{
			name: "yaml_with_comments",
			yaml: `
# This is a comment
agents:
  test-agent:
    name: "Test Agent"  # Inline comment
    llm: "test-llm"

llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "https://api.openai.com/v1"
    api_key: "sk-test-key"
`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if len(config.Agents) != 1 {
					t.Errorf("Expected 1 agent, got %d", len(config.Agents))
				}
				if len(config.LLMs) != 1 {
					t.Errorf("Expected 1 LLM, got %d", len(config.LLMs))
				}
			},
		},
		{
			name: "yaml_with_multiline_strings",
			yaml: `
agents:
  test-agent:
    name: "Test Agent"
    description: |
      This is a multiline
      description for the
      test agent.
    llm: "test-llm"

llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "https://api.openai.com/v1"
    api_key: "sk-test-key"
`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if agent, exists := config.Agents["test-agent"]; exists {
					if agent.Description == "" {
						t.Error("Multiline description should be preserved")
					}
					if len(agent.Description) < 50 {
						t.Error("Description should contain the full multiline content")
					}
				} else {
					t.Error("Expected agent 'test-agent' to exist")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadConfigFromString(tt.yaml)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfigFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && config == nil {
				t.Error("LoadConfigFromString() returned nil config without error")
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		fileContent string
		wantErr     bool
		validate    func(t *testing.T, config *Config)
	}{
		{
			name: "valid_config_file",
			fileContent: `
agents:
  test-agent:
    name: "Test Agent"
    llm: "test-llm"

llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "https://api.openai.com/v1"
    api_key: "sk-test-key"
`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if len(config.Agents) != 1 {
					t.Errorf("Expected 1 agent, got %d", len(config.Agents))
				}
				if len(config.LLMs) != 1 {
					t.Errorf("Expected 1 LLM, got %d", len(config.LLMs))
				}
			},
		},
		{
			name:        "empty_config_file",
			fileContent: "",
			wantErr:     true, // Empty config will fail validation due to missing API key
		},
		{
			name: "config_file_with_environment_variables",
			fileContent: `
agents:
  test-agent:
    name: "Test Agent"
    llm: "test-llm"

llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "${OPENAI_HOST:-https://api.openai.com/v1}"
    api_key: "${OPENAI_API_KEY:-sk-test-key}"
`,
			wantErr: false,
			validate: func(t *testing.T, config *Config) {
				if llm, exists := config.LLMs["test-llm"]; exists {
					// Should expand environment variables
					if llm.Host == "${OPENAI_HOST:-https://api.openai.com/v1}" {
						t.Error("Environment variable should be expanded")
					}
				} else {
					t.Error("Expected LLM 'test-llm' to exist")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			filePath := filepath.Join(tempDir, "test-config.yaml")
			err := os.WriteFile(filePath, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Load config from file
			config, err := LoadConfig(filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && config == nil {
				t.Error("LoadConfig() returned nil config without error")
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestLoadConfig_FileErrors(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		wantZero    bool
		description string
	}{
		{
			name:        "non_existent_file",
			filePath:    "/non/existent/file.yaml",
			wantZero:    true,
			description: "should return zero-config when file doesn't exist",
		},
		{
			name:        "empty_file_path",
			filePath:    "",
			wantZero:    true,
			description: "should return zero-config when file path is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadConfig(tt.filePath)
			if err != nil {
				t.Errorf("LoadConfig() unexpected error = %v", err)
				return
			}
			if tt.wantZero {
				if cfg.Name != "Zero Config Mode" {
					t.Errorf("LoadConfig() %s: got config name %v, want 'Zero Config Mode'", tt.description, cfg.Name)
				}
				if _, exists := cfg.Agents["assistant"]; !exists {
					t.Errorf("LoadConfig() %s: expected 'assistant' agent to exist in zero-config", tt.description)
				}
			}
		})
	}
}

func TestLoadConfig_EnvironmentVariableExpansion(t *testing.T) {
	// Set up environment variables for testing
	originalOpenAIKey := os.Getenv("OPENAI_API_KEY")
	originalOpenAIHost := os.Getenv("OPENAI_HOST")

	defer func() {
		// Restore original environment variables
		if originalOpenAIKey != "" {
			os.Setenv("OPENAI_API_KEY", originalOpenAIKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
		if originalOpenAIHost != "" {
			os.Setenv("OPENAI_HOST", originalOpenAIHost)
		} else {
			os.Unsetenv("OPENAI_HOST")
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		yaml     string
		validate func(t *testing.T, config *Config)
	}{
		{
			name: "expand_existing_environment_variable",
			envVars: map[string]string{
				"OPENAI_API_KEY": "sk-test-key-123",
			},
			yaml: `
llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
`,
			validate: func(t *testing.T, config *Config) {
				if llm, exists := config.LLMs["test-llm"]; exists {
					if llm.APIKey != "sk-test-key-123" {
						t.Errorf("API key = %v, want %v", llm.APIKey, "sk-test-key-123")
					}
				} else {
					t.Error("Expected LLM 'test-llm' to exist")
				}
			},
		},
		{
			name:    "expand_with_default_value",
			envVars: map[string]string{
				// OPENAI_HOST not set
			},
			yaml: `
llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "${OPENAI_HOST:-https://api.openai.com/v1}"
    api_key: "sk-test-key"
`,
			validate: func(t *testing.T, config *Config) {
				if llm, exists := config.LLMs["test-llm"]; exists {
					if llm.Host != "https://api.openai.com/v1" {
						t.Errorf("Host = %v, want %v", llm.Host, "https://api.openai.com/v1")
					}
				} else {
					t.Error("Expected LLM 'test-llm' to exist")
				}
			},
		},
		{
			name: "expand_with_override_value",
			envVars: map[string]string{
				"OPENAI_HOST": "https://custom.openai.com/v1",
			},
			yaml: `
llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "${OPENAI_HOST:-https://api.openai.com/v1}"
    api_key: "sk-test-key"
`,
			validate: func(t *testing.T, config *Config) {
				if llm, exists := config.LLMs["test-llm"]; exists {
					if llm.Host != "https://custom.openai.com/v1" {
						t.Errorf("Host = %v, want %v", llm.Host, "https://custom.openai.com/v1")
					}
				} else {
					t.Error("Expected LLM 'test-llm' to exist")
				}
			},
		},
		{
			name: "multiple_environment_variables",
			envVars: map[string]string{
				"OPENAI_API_KEY": "sk-test-key-456",
				"OPENAI_HOST":    "https://test.openai.com/v1",
			},
			yaml: `
llms:
  test-llm:
    type: "openai"
    model: "gpt-4o"
    host: "${OPENAI_HOST}"
    api_key: "${OPENAI_API_KEY:-sk-test-key}"
`,
			validate: func(t *testing.T, config *Config) {
				if llm, exists := config.LLMs["test-llm"]; exists {
					if llm.Host != "https://test.openai.com/v1" {
						t.Errorf("Host = %v, want %v", llm.Host, "https://test.openai.com/v1")
					}
					if llm.APIKey != "sk-test-key-456" {
						t.Errorf("API key = %v, want %v", llm.APIKey, "sk-test-key-456")
					}
				} else {
					t.Error("Expected LLM 'test-llm' to exist")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Clean up environment variables after test
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			config, err := LoadConfigFromString(tt.yaml)
			if err != nil {
				t.Fatalf("LoadConfigFromString() error = %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}
