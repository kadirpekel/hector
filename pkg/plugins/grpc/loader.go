package grpc

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/kadirpekel/hector/pkg/plugins"
)

type GRPCLoader struct {
	logger hclog.Logger
}

func NewGRPCLoader() *GRPCLoader {
	return &GRPCLoader{
		logger: hclog.New(&hclog.LoggerOptions{
			Name:   "hector-plugin",
			Level:  hclog.Info,
			Output: nil,
		}),
	}
}

func (l *GRPCLoader) Load(ctx context.Context, config *plugins.PluginConfig) (plugins.Plugin, error) {
	if config == nil {
		return nil, fmt.Errorf("plugin config cannot be nil")
	}

	if config.Manifest == nil {
		return nil, fmt.Errorf("plugin manifest is required")
	}

	clientConfig := &plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         l.getPluginMap(config.Manifest.Type),
		Cmd:             exec.Command(config.Path),
		Logger:          l.logger,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
	}

	client := plugin.NewClient(clientConfig)

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to get RPC client: %w", err)
	}

	raw, err := rpcClient.Dispense(string(config.Manifest.Type))
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense plugin: %w", err)
	}

	adapter, err := l.wrapPlugin(raw, config.Manifest, client)
	if err != nil {
		client.Kill()
		return nil, err
	}

	return adapter, nil
}

func (l *GRPCLoader) Unload(ctx context.Context, plugin plugins.Plugin) error {

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
	case *DocumentParserPluginAdapter:
		if adapter.client != nil {
			adapter.client.Kill()
		}
	}
	return nil
}

func (l *GRPCLoader) SupportedProtocol() plugins.PluginProtocol {
	return plugins.ProtocolGRPC
}

func (l *GRPCLoader) Validate(ctx context.Context, path string) error {

	cmd := exec.Command(path)
	if cmd.Path == "" {
		return fmt.Errorf("plugin executable not found: %s", path)
	}
	return nil
}

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
	case plugins.PluginTypeDocumentParser:
		return map[string]plugin.Plugin{
			string(plugins.PluginTypeDocumentParser): &DocumentParserProviderPlugin{},
		}

	default:
		return nil
	}
}

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

	case plugins.PluginTypeDocumentParser:
		if parserPlugin, ok := raw.(DocumentParserProvider); ok {
			return NewDocumentParserPluginAdapter(parserPlugin, manifest, client), nil
		}
		return nil, fmt.Errorf("plugin does not implement Document Parser provider interface")

	default:
		return nil, fmt.Errorf("unsupported plugin type: %s", manifest.Type)
	}
}

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "HECTOR_PLUGIN",
	MagicCookieValue: "hector_plugin_v1",
}

func GetHandshakeConfig() plugin.HandshakeConfig {
	return handshakeConfig
}
