package runtime

import (
	"os"
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
)

func TestNew_WithValidConfig(t *testing.T) {
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "hector-test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Write a minimal valid config
	configContent := `
name: "Test Config"
llms:
  openai:
    type: openai
    model: gpt-4o-mini
    api_key: test-key
    host: https://api.openai.com/v1
agents:
  test:
    name: "Test Agent"
    type: native
    llm: openai
`
	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Test creating runtime from config file
	rt, err := New(Options{
		ConfigFile: tmpfile.Name(),
	})

	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if rt == nil {
		t.Fatal("New() returned nil runtime")
	}

	// Verify runtime has client and config
	if rt.Client() == nil {
		t.Error("Runtime.Client() returned nil")
	}
	if rt.Config() == nil {
		t.Error("Runtime.Config() returned nil")
	}

	// Clean up
	if err := rt.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNew_ZeroConfig_OpenAI(t *testing.T) {
	rt, err := New(Options{
		ConfigFile: "nonexistent.yaml",
		Provider:   "openai",
		APIKey:     "test-key",
		Model:      "gpt-4",
	})

	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if rt == nil {
		t.Fatal("New() returned nil runtime")
	}

	cfg := rt.Config()
	if cfg == nil {
		t.Fatal("Config() returned nil")
	}

	// Verify OpenAI config
	if len(cfg.LLMs) == 0 {
		t.Error("No LLM providers configured")
	}
	if llm, ok := cfg.LLMs["openai"]; ok {
		if llm.APIKey != "test-key" {
			t.Errorf("Expected API key 'test-key', got '%s'", llm.APIKey)
		}
		if llm.Model != "gpt-4" {
			t.Errorf("Expected model 'gpt-4', got '%s'", llm.Model)
		}
	} else {
		t.Error("OpenAI provider not found in config")
	}

	rt.Close()
}

func TestNew_ZeroConfig_Anthropic(t *testing.T) {
	rt, err := New(Options{
		ConfigFile: "nonexistent.yaml",
		Provider:   "anthropic",
		APIKey:     "sk-ant-test",
		Model:      "claude-3-opus",
	})

	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if rt == nil {
		t.Fatal("New() returned nil runtime")
	}

	cfg := rt.Config()
	if cfg == nil {
		t.Fatal("Config() returned nil")
	}

	// Verify Anthropic config
	if llm, ok := cfg.LLMs["anthropic"]; ok {
		if llm.APIKey != "sk-ant-test" {
			t.Errorf("Expected API key 'sk-ant-test', got '%s'", llm.APIKey)
		}
		if llm.Model != "claude-3-opus" {
			t.Errorf("Expected model 'claude-3-opus', got '%s'", llm.Model)
		}
	} else {
		t.Error("Anthropic provider not found in config")
	}

	rt.Close()
}

func TestNew_ZeroConfig_Gemini(t *testing.T) {
	rt, err := New(Options{
		ConfigFile: "nonexistent.yaml",
		Provider:   "gemini",
		APIKey:     "AIza-test",
		Model:      "gemini-pro",
	})

	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	cfg := rt.Config()
	if cfg == nil {
		t.Fatal("Config() returned nil")
	}

	// Verify Gemini config
	if llm, ok := cfg.LLMs["gemini"]; ok {
		if llm.APIKey != "AIza-test" {
			t.Errorf("Expected API key 'AIza-test', got '%s'", llm.APIKey)
		}
		if llm.Model != "gemini-pro" {
			t.Errorf("Expected model 'gemini-pro', got '%s'", llm.Model)
		}
	} else {
		t.Error("Gemini provider not found in config")
	}

	rt.Close()
}

func TestNew_ZeroConfig_WithTools(t *testing.T) {
	rt, err := New(Options{
		ConfigFile: "nonexistent.yaml",
		Provider:   "openai",
		APIKey:     "test-key",
		Tools:      true,
	})

	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	cfg := rt.Config()
	if cfg == nil {
		t.Fatal("Config() returned nil")
	}

	// Verify tools are configured
	if cfg.Tools.Tools == nil {
		t.Error("Tools not configured")
	}
	if _, ok := cfg.Tools.Tools["command"]; !ok {
		t.Error("Command tool not configured")
	}
	if _, ok := cfg.Tools.Tools["file_writer"]; !ok {
		t.Error("File writer tool not configured")
	}

	rt.Close()
}

func TestNew_ZeroConfig_NoAPIKey(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")

	_, err := New(Options{
		ConfigFile: "nonexistent.yaml",
		Provider:   "openai",
		// No API key provided
	})

	if err == nil {
		t.Error("Expected error when no API key provided, got nil")
	}
}

func TestNew_ZeroConfig_EnvironmentVariable(t *testing.T) {
	// Set environment variable
	os.Setenv("OPENAI_API_KEY", "env-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	rt, err := New(Options{
		ConfigFile: "nonexistent.yaml",
		Provider:   "openai",
		// API key should come from env
	})

	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	cfg := rt.Config()
	if llm, ok := cfg.LLMs["openai"]; ok {
		if llm.APIKey != "env-key" {
			t.Errorf("Expected API key from env 'env-key', got '%s'", llm.APIKey)
		}
	}

	rt.Close()
}

func TestNew_InvalidConfigFile(t *testing.T) {
	// Create a temporary file with invalid YAML
	tmpfile, err := os.CreateTemp("", "hector-invalid-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	_, _ = tmpfile.Write([]byte("invalid: yaml: content: ["))
	tmpfile.Close()

	_, err = New(Options{
		ConfigFile: tmpfile.Name(),
	})

	if err == nil {
		t.Error("Expected error for invalid config file, got nil")
	}
}

func TestLoadOrCreateConfig_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		opts     Options
		provider string
	}{
		{
			name: "defaults to openai",
			opts: Options{
				ConfigFile: "nonexistent.yaml",
				APIKey:     "test",
			},
			provider: "openai",
		},
		{
			name: "explicit openai",
			opts: Options{
				ConfigFile: "nonexistent.yaml",
				Provider:   "openai",
				APIKey:     "test",
			},
			provider: "openai",
		},
		{
			name: "explicit anthropic",
			opts: Options{
				ConfigFile: "nonexistent.yaml",
				Provider:   "anthropic",
				APIKey:     "test",
			},
			provider: "anthropic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadOrCreateConfig(tt.opts)
			if err != nil {
				t.Fatalf("loadOrCreateConfig() error = %v", err)
			}
			if _, ok := cfg.LLMs[tt.provider]; !ok {
				t.Errorf("Expected provider '%s' not found", tt.provider)
			}
		})
	}
}

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient("http://localhost:50052", "test-token")
	if client == nil {
		t.Error("NewHTTPClient() returned nil")
	}
	client.Close()
}

func TestNewDirectClient(t *testing.T) {
	cfg := &config.Config{
		Name: "Test",
		LLMs: map[string]config.LLMProviderConfig{
			"openai": {
				Type:   "openai",
				Model:  "gpt-4o-mini",
				APIKey: "test",
				Host:   "https://api.openai.com/v1",
			},
		},
		Agents: map[string]config.AgentConfig{
			"test": {
				Name: "Test",
				Type: "native",
				LLM:  "openai",
			},
		},
	}
	cfg.SetDefaults()

	client, err := NewDirectClient(cfg)
	if err != nil {
		t.Fatalf("NewDirectClient() error = %v", err)
	}
	if client == nil {
		t.Error("NewDirectClient() returned nil")
	}
	client.Close()
}
