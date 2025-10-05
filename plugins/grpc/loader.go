package grpc

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/kadirpekel/hector/plugins"
)

// ============================================================================
// GRPC PLUGIN LOADER
// ============================================================================

// GRPCLoader loads gRPC-based plugins using HashiCorp go-plugin
type GRPCLoader struct {
	logger hclog.Logger
}

// NewGRPCLoader creates a new gRPC plugin loader
func NewGRPCLoader() *GRPCLoader {
	return &GRPCLoader{
		logger: hclog.New(&hclog.LoggerOptions{
			Name:   "hector-plugin",
			Level:  hclog.Info,
			Output: nil, // Uses default output
		}),
	}
}

// Load loads a plugin from the given path
func (l *GRPCLoader) Load(ctx context.Context, config *plugins.PluginConfig) (plugins.Plugin, error) {
	if config == nil {
		return nil, fmt.Errorf("plugin config cannot be nil")
	}

	// Need manifest to determine plugin type
	if config.Manifest == nil {
		return nil, fmt.Errorf("plugin manifest is required")
	}

	// Create the plugin client configuration
	clientConfig := &plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         l.getPluginMap(config.Manifest.Type),
		Cmd:             exec.Command(config.Path),
		Logger:          l.logger,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
	}

	// Start the plugin
	client := plugin.NewClient(clientConfig)

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to get RPC client: %w", err)
	}

	// Dispense the plugin based on its type
	raw, err := rpcClient.Dispense(string(config.Manifest.Type))
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense plugin: %w", err)
	}

	// Wrap in appropriate adapter
	adapter, err := l.wrapPlugin(raw, config.Manifest, client)
	if err != nil {
		client.Kill()
		return nil, err
	}

	return adapter, nil
}

// Unload unloads a plugin
func (l *GRPCLoader) Unload(ctx context.Context, plugin plugins.Plugin) error {
	// Try to cast to one of our adapter types
	switch adapter := plugin.(type) {
	case *LLMPluginAdapter:
		if adapter.client != nil {
			adapter.client.Kill()
		}
	case *DatabasePluginAdapter:
		if adapter.client != nil {
			adapter.client.Kill()
		}
	case *EmbedderPluginAdapter:
		if adapter.client != nil {
			adapter.client.Kill()
		}
	}
	return nil
}

// SupportedProtocol returns the protocol this loader supports
func (l *GRPCLoader) SupportedProtocol() plugins.PluginProtocol {
	return plugins.ProtocolGRPC
}

// Validate validates that a plugin can be loaded
func (l *GRPCLoader) Validate(ctx context.Context, path string) error {
	// Check if file exists and is executable
	cmd := exec.Command(path)
	if cmd.Path == "" {
		return fmt.Errorf("plugin executable not found: %s", path)
	}
	return nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// getPluginMap returns the plugin map for a given plugin type
func (l *GRPCLoader) getPluginMap(pluginType plugins.PluginType) map[string]plugin.Plugin {
	switch pluginType {
	case plugins.PluginTypeLLM:
		return map[string]plugin.Plugin{
			string(plugins.PluginTypeLLM): &LLMProviderPlugin{},
		}
	case plugins.PluginTypeDatabase:
		return map[string]plugin.Plugin{
			string(plugins.PluginTypeDatabase): &DatabaseProviderPlugin{},
		}
	case plugins.PluginTypeEmbedder:
		return map[string]plugin.Plugin{
			string(plugins.PluginTypeEmbedder): &EmbedderProviderPlugin{},
		}
	// Add more types as needed
	default:
		return nil
	}
}

// wrapPlugin wraps the raw plugin interface in an adapter
func (l *GRPCLoader) wrapPlugin(raw interface{}, manifest *plugins.PluginManifest, client *plugin.Client) (plugins.Plugin, error) {
	switch manifest.Type {
	case plugins.PluginTypeLLM:
		if llmPlugin, ok := raw.(LLMProvider); ok {
			return NewLLMPluginAdapter(llmPlugin, manifest, client), nil
		}
		return nil, fmt.Errorf("plugin does not implement LLM provider interface")

	case plugins.PluginTypeDatabase:
		if dbPlugin, ok := raw.(DatabaseProvider); ok {
			return NewDatabasePluginAdapter(dbPlugin, manifest, client), nil
		}
		return nil, fmt.Errorf("plugin does not implement Database provider interface")

	case plugins.PluginTypeEmbedder:
		if embedderPlugin, ok := raw.(EmbedderProvider); ok {
			return NewEmbedderPluginAdapter(embedderPlugin, manifest, client), nil
		}
		return nil, fmt.Errorf("plugin does not implement Embedder provider interface")

	default:
		return nil, fmt.Errorf("unsupported plugin type: %s", manifest.Type)
	}
}

// ============================================================================
// HANDSHAKE CONFIGURATION
// ============================================================================

// handshakeConfig is used to verify that the plugin and host are compatible
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "HECTOR_PLUGIN",
	MagicCookieValue: "hector_plugin_v1",
}

// GetHandshakeConfig returns the handshake configuration for plugin authors
func GetHandshakeConfig() plugin.HandshakeConfig {
	return handshakeConfig
}
