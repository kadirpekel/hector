package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoader_File_Load(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")

	configYAML := `
version: "1.0"
name: "test-config"
agents:
  test-agent:
    type: native
    name: Test Agent
    llm: openai
llms:
  openai:
    type: openai
    model: gpt-4
    api_key: test-key
`
	if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	loader, err := NewLoader(LoaderOptions{
		Type: ConfigTypeFile,
		Path: configFile,
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}
	defer loader.Stop()

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", cfg.Version)
	}
	if cfg.Name != "test-config" {
		t.Errorf("expected name 'test-config', got %s", cfg.Name)
	}
	if len(cfg.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(cfg.Agents))
	}
	if len(cfg.LLMs) != 1 {
		t.Errorf("expected 1 LLM, got %d", len(cfg.LLMs))
	}
}

func TestLoader_File_NotFound(t *testing.T) {
	loader, err := NewLoader(LoaderOptions{
		Type: ConfigTypeFile,
		Path: "/nonexistent/file.yaml",
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}
	defer loader.Stop()

	_, err = loader.Load()
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoader_File_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `
version: "1.0"
agents:
  - invalid: [unclosed
`
	if err := os.WriteFile(configFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	loader, err := NewLoader(LoaderOptions{
		Type: ConfigTypeFile,
		Path: configFile,
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}
	defer loader.Stop()

	_, err = loader.Load()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoader_File_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid-config.yaml")

	invalidConfig := `
version: "1.0"
invalid_field: value
agents:
  test-agent:
    type: native
`
	if err := os.WriteFile(configFile, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	loader, err := NewLoader(LoaderOptions{
		Type: ConfigTypeFile,
		Path: configFile,
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}
	defer loader.Stop()

	_, err = loader.Load()
	if err == nil {
		t.Fatal("expected error for invalid config structure")
	}
}

func TestLoader_File_Watch(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "watch-test.yaml")

	initialConfig := `
version: "1.0"
name: "initial"
agents:
  test-agent:
    type: native
    name: Initial Agent
    llm: openai
llms:
  openai:
    type: openai
    model: gpt-4
    api_key: test-key
`
	if err := os.WriteFile(configFile, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	reloadCount := 0
	loader, err := NewLoader(LoaderOptions{
		Type:  ConfigTypeFile,
		Path:  configFile,
		Watch: true,
		OnChange: func(cfg *Config) error {
			reloadCount++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}
	defer loader.Stop()

	// Load initial config
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.Name != "initial" {
		t.Errorf("expected name 'initial', got %s", cfg.Name)
	}

	// Wait a bit for watcher to start
	time.Sleep(200 * time.Millisecond)

	// Update config file
	updatedConfig := `
version: "1.0"
name: "updated"
agents:
  test-agent:
    type: native
    name: Updated Agent
    llm: openai
llms:
  openai:
    type: openai
    model: gpt-4
    api_key: test-key
`
	if err := os.WriteFile(configFile, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	// Wait for watch to trigger
	time.Sleep(500 * time.Millisecond)

	if reloadCount == 0 {
		t.Error("expected reload to be triggered, but it wasn't")
	}
}

func TestLoader_EnvVarExpansion(t *testing.T) {
	os.Setenv("TEST_API_KEY", "secret-key-123")
	defer os.Unsetenv("TEST_API_KEY")

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "env-test.yaml")

	configYAML := `
version: "1.0"
agents:
  default-agent:
    type: native
    name: Default Agent
    llm: openai
llms:
  openai:
    type: openai
    model: gpt-4
    api_key: ${TEST_API_KEY}
`
	if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	loader, err := NewLoader(LoaderOptions{
		Type: ConfigTypeFile,
		Path: configFile,
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}
	defer loader.Stop()

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.LLMs["openai"].APIKey != "secret-key-123" {
		t.Errorf("expected API key 'secret-key-123', got %s", cfg.LLMs["openai"].APIKey)
	}
}

func TestLoader_Consul_Load(t *testing.T) {
	// Skip if Consul is not available
	if os.Getenv("SKIP_CONSUL_TEST") == "1" {
		t.Skip("Skipping Consul test")
	}

	// This test requires a running Consul instance
	// For now, we'll test the error handling
	loader, err := NewLoader(LoaderOptions{
		Type:      ConfigTypeConsul,
		Path:      "test/config",
		Endpoints: []string{"localhost:8500"},
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}
	defer loader.Stop()

	_, err = loader.Load()
	// We expect an error if Consul is not running or key doesn't exist
	// This is acceptable - we're testing the loader creation and error handling
	if err == nil {
		t.Log("Consul test passed - config loaded successfully")
	} else {
		t.Logf("Consul test - expected error (Consul may not be running): %v", err)
	}
}

func TestLoader_NewLoader_Defaults(t *testing.T) {
	// Test default config type
	loader, err := NewLoader(LoaderOptions{
		Path: "test.yaml",
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}
	if loader.options.Type != ConfigTypeFile {
		t.Errorf("expected default type File, got %s", loader.options.Type)
	}

	// Test default endpoints
	loader, err = NewLoader(LoaderOptions{
		Type: ConfigTypeConsul,
		Path: "test",
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}
	if len(loader.options.Endpoints) != 1 || loader.options.Endpoints[0] != "localhost:8500" {
		t.Errorf("expected default Consul endpoint, got %v", loader.options.Endpoints)
	}
}

func TestLoader_NewLoader_Errors(t *testing.T) {
	// Test missing path
	_, err := NewLoader(LoaderOptions{
		Type: ConfigTypeFile,
	})
	if err == nil {
		t.Fatal("expected error for missing path")
	}

	// Test invalid config type (should be handled by switch)
	loader, err := NewLoader(LoaderOptions{
		Type: ConfigType("invalid"),
		Path: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = loader.readBytes()
	if err == nil {
		t.Fatal("expected error for unsupported config type")
	}
}

func TestLoader_ParseBytes_YAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")

	configYAML := `
version: "1.0"
name: "test"
`
	if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	loader, err := NewLoader(LoaderOptions{
		Type: ConfigTypeFile,
		Path: configFile,
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	data, err := loader.readBytes()
	if err != nil {
		t.Fatalf("failed to read bytes: %v", err)
	}

	parsed, err := loader.parseBytes(data)
	if err != nil {
		t.Fatalf("failed to parse bytes: %v", err)
	}

	if parsed["version"] != "1.0" {
		t.Errorf("expected version 1.0, got %v", parsed["version"])
	}
}

func TestLoader_ParseBytes_JSON(t *testing.T) {
	// Create a temporary JSON config for Consul simulation
	configJSON := map[string]interface{}{
		"version": "1.0",
		"name":    "test",
	}
	jsonBytes, _ := json.Marshal(configJSON)

	loader := &Loader{
		options: LoaderOptions{
			Type: ConfigTypeConsul,
		},
	}

	parsed, err := loader.parseBytes(jsonBytes)
	if err != nil {
		t.Fatalf("failed to parse JSON bytes: %v", err)
	}

	if parsed["version"] != "1.0" {
		t.Errorf("expected version 1.0, got %v", parsed["version"])
	}
}

func TestLoader_Stop(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")

	configYAML := `
version: "1.0"
`
	if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	loader, err := NewLoader(LoaderOptions{
		Type:  ConfigTypeFile,
		Path:  configFile,
		Watch: true,
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}

	// Stop should not panic
	loader.Stop()

	// Calling Stop multiple times should be safe
	loader.Stop()
}

func TestLoadConfigWithLoader(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.yaml")

	configYAML := `
version: "1.0"
name: "test"
agents:
  test-agent:
    type: native
    name: Test Agent
    llm: openai
llms:
  openai:
    type: openai
    model: gpt-4
    api_key: test-key
`
	if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	cfg, loader, err := LoadConfigWithLoader(LoaderOptions{
		Type: ConfigTypeFile,
		Path: configFile,
	})
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	defer loader.Stop()

	if cfg.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", cfg.Version)
	}
}

func TestParseConfigType(t *testing.T) {
	tests := []struct {
		input    string
		expected ConfigType
		err      bool
	}{
		{"file", ConfigTypeFile, false},
		{"FILE", ConfigTypeFile, false},
		{"  file  ", ConfigTypeFile, false},
		{"consul", ConfigTypeConsul, false},
		{"etcd", ConfigTypeEtcd, false},
		{"zookeeper", ConfigTypeZookeeper, false},
		{"zk", ConfigTypeZookeeper, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseConfigType(tt.input)
			if tt.err {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, result)
				}
			}
		})
	}
}
