package config

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
)

func TestLoader_Consul_Integration(t *testing.T) {
	// Skip if Consul is not available
	if os.Getenv("SKIP_CONSUL_TEST") == "1" {
		t.Skip("Skipping Consul integration test")
	}

	// Check if Consul is accessible
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		t.Skipf("Skipping Consul test - failed to create client: %v", err)
	}

	_, _, err = client.KV().Get("test", nil)
	if err != nil {
		t.Skipf("Skipping Consul test - Consul not accessible: %v", err)
	}

	// Prepare test config
	testConfig := map[string]interface{}{
		"version": "1.0",
		"name":    "consul-test",
		"agents": map[string]interface{}{
			"test-agent": map[string]interface{}{
				"type": "native",
				"name": "Test Agent",
				"llm":  "openai",
			},
		},
		"llms": map[string]interface{}{
			"openai": map[string]interface{}{
				"type":    "openai",
				"model":   "gpt-4",
				"api_key": "test-key",
			},
		},
	}

	configJSON, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("failed to marshal test config: %v", err)
	}

	// Upload config to Consul
	testKey := "hector/test/config"
	_, err = client.KV().Put(&api.KVPair{
		Key:   testKey,
		Value: configJSON,
	}, nil)
	if err != nil {
		t.Fatalf("failed to upload config to Consul: %v", err)
	}
	defer func() {
		// Cleanup
		_, _ = client.KV().Delete(testKey, nil)
	}()

	// Test loading
	loader, err := NewLoader(LoaderOptions{
		Type:      ConfigTypeConsul,
		Path:      testKey,
		Endpoints: []string{"localhost:8500"},
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
	if cfg.Name != "consul-test" {
		t.Errorf("expected name 'consul-test', got %s", cfg.Name)
	}
	if len(cfg.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(cfg.Agents))
	}

	// Test watching
	reloadCount := 0
	loader.SetOnChange(func(newCfg *Config) error {
		reloadCount++
		return nil
	})

	// Start watching
	if err := loader.options.Watch; !err {
		// Enable watch
		go loader.watch()
	}

	// Wait for watcher to start
	time.Sleep(500 * time.Millisecond)

	// Update config
	updatedConfig := map[string]interface{}{
		"version": "1.0",
		"name":    "consul-test-updated",
		"agents": map[string]interface{}{
			"test-agent": map[string]interface{}{
				"type": "native",
				"name": "Updated Test Agent",
				"llm":  "openai",
			},
		},
		"llms": map[string]interface{}{
			"openai": map[string]interface{}{
				"type":    "openai",
				"model":   "gpt-4",
				"api_key": "test-key",
			},
		},
	}

	updatedJSON, err := json.Marshal(updatedConfig)
	if err != nil {
		t.Fatalf("failed to marshal updated config: %v", err)
	}

	_, err = client.KV().Put(&api.KVPair{
		Key:   testKey,
		Value: updatedJSON,
	}, nil)
	if err != nil {
		t.Fatalf("failed to update config in Consul: %v", err)
	}

	// Wait for reload
	time.Sleep(2 * time.Second)

	if reloadCount == 0 {
		t.Error("expected reload to be triggered, but it wasn't")
	}
}

func TestLoader_Consul_NotFound(t *testing.T) {
	if os.Getenv("SKIP_CONSUL_TEST") == "1" {
		t.Skip("Skipping Consul test")
	}

	loader, err := NewLoader(LoaderOptions{
		Type:      ConfigTypeConsul,
		Path:      "nonexistent/key/that/does/not/exist",
		Endpoints: []string{"localhost:8500"},
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}
	defer loader.Stop()

	_, err = loader.Load()
	if err == nil {
		t.Fatal("expected error for nonexistent key")
	}
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestLoader_Consul_InvalidJSON(t *testing.T) {
	if os.Getenv("SKIP_CONSUL_TEST") == "1" {
		t.Skip("Skipping Consul test")
	}

	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		t.Skipf("Skipping Consul test - failed to create client: %v", err)
	}

	// Check if Consul is accessible
	_, _, err = client.KV().Get("test", nil)
	if err != nil {
		t.Skipf("Skipping Consul test - Consul not accessible: %v", err)
	}

	testKey := "hector/test/invalid"
	_, err = client.KV().Put(&api.KVPair{
		Key:   testKey,
		Value: []byte("invalid json {"),
	}, nil)
	if err != nil {
		t.Skipf("Skipping Consul test - failed to upload invalid JSON (Consul not accessible): %v", err)
	}
	defer func() {
		_, _ = client.KV().Delete(testKey, nil)
	}()

	loader, err := NewLoader(LoaderOptions{
		Type:      ConfigTypeConsul,
		Path:      testKey,
		Endpoints: []string{"localhost:8500"},
	})
	if err != nil {
		t.Fatalf("failed to create loader: %v", err)
	}
	defer loader.Stop()

	_, err = loader.Load()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
