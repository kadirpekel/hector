package config

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/go-zookeeper/zk"
	"github.com/hashicorp/consul/api"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/yaml.v3"
)

// TestAllProviders_Integration tests all config providers end-to-end
// Requires: docker-compose -f docker-compose.config-providers.yaml up -d
func TestAllProviders_Integration(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TEST") == "1" {
		t.Skip("Skipping integration tests")
	}

	tests := []struct {
		name     string
		provider ConfigType
		setup    func(t *testing.T) (string, func()) // returns key and cleanup func
	}{
		{
			name:     "Consul",
			provider: ConfigTypeConsul,
			setup:    setupConsulTest,
		},
		{
			name:     "Etcd",
			provider: ConfigTypeEtcd,
			setup:    setupEtcdTest,
		},
		{
			name:     "ZooKeeper",
			provider: ConfigTypeZookeeper,
			setup:    setupZookeeperTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, cleanup := tt.setup(t)
			defer cleanup()

			// Test loading
			loader, err := NewLoader(LoaderOptions{
				Type:      tt.provider,
				Path:      key,
				Endpoints: getDefaultEndpoints(tt.provider),
			})
			if err != nil {
				t.Fatalf("failed to create loader: %v", err)
			}
			defer loader.Stop()

			cfg, err := loader.Load()
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			if cfg.Version == "" {
				t.Error("config version should be set")
			}
			if len(cfg.Agents) == 0 {
				t.Error("config should have at least one agent")
			}
		})
	}
}

// TestAllProviders_Watch tests watch functionality for all providers
func TestAllProviders_Watch(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TEST") == "1" {
		t.Skip("Skipping integration tests")
	}

	tests := []struct {
		name     string
		provider ConfigType
		setup    func(t *testing.T) (string, func())
		update   func(t *testing.T, key string)
	}{
		{
			name:     "Consul",
			provider: ConfigTypeConsul,
			setup:    setupConsulTest,
			update:   updateConsulConfig,
		},
		{
			name:     "Etcd",
			provider: ConfigTypeEtcd,
			setup:    setupEtcdTest,
			update:   updateEtcdConfig,
		},
		{
			name:     "ZooKeeper",
			provider: ConfigTypeZookeeper,
			setup:    setupZookeeperTest,
			update:   updateZookeeperConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, cleanup := tt.setup(t)
			defer cleanup()

			reloadCount := 0
			loader, err := NewLoader(LoaderOptions{
				Type:      tt.provider,
				Path:      key,
				Endpoints: getDefaultEndpoints(tt.provider),
				Watch:     true,
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
			_, err = loader.Load()
			if err != nil {
				t.Fatalf("failed to load initial config: %v", err)
			}

			// Wait for watcher to start
			time.Sleep(500 * time.Millisecond)

			// Update config
			tt.update(t, key)

			// Wait for reload
			time.Sleep(2 * time.Second)

			if reloadCount == 0 {
				t.Error("expected reload to be triggered, but it wasn't")
			}
		})
	}
}

// Helper functions

func getDefaultEndpoints(provider ConfigType) []string {
	switch provider {
	case ConfigTypeConsul:
		return []string{"localhost:8500"}
	case ConfigTypeEtcd:
		return []string{"localhost:2379"}
	case ConfigTypeZookeeper:
		return []string{"localhost:2181"}
	default:
		return nil
	}
}

func getTestConfig() map[string]interface{} {
	return map[string]interface{}{
		"version": "1.0",
		"name":    "test-config",
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
}

func setupConsulTest(t *testing.T) (string, func()) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		t.Skipf("Skipping Consul test - failed to create client: %v", err)
	}

	// Check if Consul is accessible
	_, _, err = client.KV().Get("test", nil)
	if err != nil {
		t.Skipf("Skipping Consul test - Consul not accessible: %v", err)
	}

	testKey := "hector/test/integration"
	configJSON, _ := json.Marshal(getTestConfig())

	_, err = client.KV().Put(&api.KVPair{
		Key:   testKey,
		Value: configJSON,
	}, nil)
	if err != nil {
		t.Fatalf("failed to upload config: %v", err)
	}

	cleanup := func() {
		_, _ = client.KV().Delete(testKey, nil)
	}

	return testKey, cleanup
}

func updateConsulConfig(t *testing.T, key string) {
	client, _ := api.NewClient(api.DefaultConfig())
	config := getTestConfig()
	config["name"] = "updated-config"
	configJSON, _ := json.Marshal(config)

	_, err := client.KV().Put(&api.KVPair{
		Key:   key,
		Value: configJSON,
	}, nil)
	if err != nil {
		t.Fatalf("failed to update config: %v", err)
	}
}

func setupEtcdTest(t *testing.T) (string, func()) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Skipf("Skipping Etcd test - failed to create client: %v", err)
	}

	// Check if Etcd is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err = client.Status(ctx, "localhost:2379")
	cancel()
	if err != nil {
		client.Close()
		t.Skipf("Skipping Etcd test - Etcd not accessible: %v", err)
	}

	testKey := "/hector/test/integration"
	configJSON, _ := json.Marshal(getTestConfig())

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	_, err = client.Put(ctx, testKey, string(configJSON))
	cancel()
	if err != nil {
		client.Close()
		t.Fatalf("failed to upload config: %v", err)
	}

	cleanup := func() {
		cleanupClient, _ := clientv3.New(clientv3.Config{
			Endpoints:   []string{"localhost:2379"},
			DialTimeout: 5 * time.Second,
		})
		if cleanupClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, _ = cleanupClient.Delete(ctx, testKey)
			cancel()
			cleanupClient.Close()
		}
		client.Close()
	}

	return testKey, cleanup
}

func updateEtcdConfig(t *testing.T, key string) {
	client, _ := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	defer client.Close()

	config := getTestConfig()
	config["name"] = "updated-config"
	configJSON, _ := json.Marshal(config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.Put(ctx, key, string(configJSON))
	if err != nil {
		t.Fatalf("failed to update config: %v", err)
	}
}

func setupZookeeperTest(t *testing.T) (string, func()) {
	// Check ZooKeeper connection first
	conn, _, err := zk.Connect([]string{"localhost:2181"}, 10*time.Second)
	if err != nil {
		t.Skipf("Skipping ZooKeeper test - failed to connect: %v", err)
	}
	conn.Close()

	testKey := "/hector/test/integration"
	config := getTestConfig()
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	// Create node using helper
	if err := setupZookeeperNode([]string{"localhost:2181"}, testKey, configYAML); err != nil {
		t.Fatalf("failed to create zookeeper node: %v", err)
	}

	// Verify we can read it
	zkProvider, err := NewZookeeperProvider([]string{"localhost:2181"}, testKey)
	if err != nil {
		t.Fatalf("failed to create zookeeper provider: %v", err)
	}

	_, err = zkProvider.ReadBytes()
	if err != nil {
		t.Fatalf("failed to read created zookeeper node: %v", err)
	}

	cleanup := func() {
		zkProvider.Close()
		_ = deleteZookeeperNode([]string{"localhost:2181"}, testKey)
	}

	return testKey, cleanup
}

func updateZookeeperConfig(t *testing.T, key string) {
	// Use direct zk connection to update
	conn, _, err := zk.Connect([]string{"localhost:2181"}, 10*time.Second)
	if err != nil {
		t.Fatalf("failed to connect to zookeeper: %v", err)
	}
	defer conn.Close()

	config := getTestConfig()
	config["name"] = "updated-config"
	configYAML, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	// Update node
	_, err = conn.Set(key, configYAML, -1)
	if err != nil {
		t.Fatalf("failed to update zookeeper node: %v", err)
	}
}
