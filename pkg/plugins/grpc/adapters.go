package grpc

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/kadirpekel/hector/pkg/plugins"
)

type BasePluginAdapter struct {
	manifest *plugins.PluginManifest
	client   *plugin.Client
	status   plugins.PluginStatus
}

func (a *BasePluginAdapter) GetManifest() *plugins.PluginManifest {
	return a.manifest
}

func (a *BasePluginAdapter) GetStatus() plugins.PluginStatus {
	return a.status
}

func (a *BasePluginAdapter) setStatus(status plugins.PluginStatus) {
	a.status = status
}

type LLMPluginAdapter struct {
	*BasePluginAdapter
	plugin LLMProvider
}

func NewLLMPluginAdapter(plugin LLMProvider, manifest *plugins.PluginManifest, client *plugin.Client) *LLMPluginAdapter {
	return &LLMPluginAdapter{
		BasePluginAdapter: &BasePluginAdapter{
			manifest: manifest,
			client:   client,
			status:   plugins.StatusReady,
		},
		plugin: plugin,
	}
}

func (a *LLMPluginAdapter) Initialize(ctx context.Context, config map[string]interface{}) error {
	a.setStatus(plugins.StatusLoading)

	stringConfig := make(map[string]string)
	for k, v := range config {
		if str, ok := v.(string); ok {
			stringConfig[k] = str
		}
	}

	err := a.plugin.Initialize(ctx, stringConfig)
	if err != nil {
		a.setStatus(plugins.StatusError)
		return err
	}

	a.setStatus(plugins.StatusReady)
	return nil
}

func (a *LLMPluginAdapter) Shutdown(ctx context.Context) error {
	err := a.plugin.Shutdown(ctx)
	if err != nil {
		return err
	}
	a.setStatus(plugins.StatusShutdown)
	return nil
}

func (a *LLMPluginAdapter) Health(ctx context.Context) error {
	err := a.plugin.Health(ctx)
	if err != nil {
		a.setStatus(plugins.StatusError)
		return err
	}
	return nil
}

func (a *LLMPluginAdapter) GetPlugin() LLMProvider {
	return a.plugin
}

type DatabasePluginAdapter struct {
	*BasePluginAdapter
	plugin DatabaseProvider
}

func NewDatabasePluginAdapter(plugin DatabaseProvider, manifest *plugins.PluginManifest, client *plugin.Client) *DatabasePluginAdapter {
	return &DatabasePluginAdapter{
		BasePluginAdapter: &BasePluginAdapter{
			manifest: manifest,
			client:   client,
			status:   plugins.StatusReady,
		},
		plugin: plugin,
	}
}

func (a *DatabasePluginAdapter) Initialize(ctx context.Context, config map[string]interface{}) error {
	a.setStatus(plugins.StatusLoading)

	stringConfig := make(map[string]string)
	for k, v := range config {
		if str, ok := v.(string); ok {
			stringConfig[k] = str
		}
	}

	err := a.plugin.Initialize(ctx, stringConfig)
	if err != nil {
		a.setStatus(plugins.StatusError)
		return err
	}

	a.setStatus(plugins.StatusReady)
	return nil
}

func (a *DatabasePluginAdapter) Shutdown(ctx context.Context) error {
	err := a.plugin.Shutdown(ctx)
	if err != nil {
		return err
	}
	a.setStatus(plugins.StatusShutdown)
	return nil
}

func (a *DatabasePluginAdapter) Health(ctx context.Context) error {
	err := a.plugin.Health(ctx)
	if err != nil {
		a.setStatus(plugins.StatusError)
		return err
	}
	return nil
}

func (a *DatabasePluginAdapter) GetPlugin() DatabaseProvider {
	return a.plugin
}

type EmbedderPluginAdapter struct {
	*BasePluginAdapter
	plugin EmbedderProvider
}

func NewEmbedderPluginAdapter(plugin EmbedderProvider, manifest *plugins.PluginManifest, client *plugin.Client) *EmbedderPluginAdapter {
	return &EmbedderPluginAdapter{
		BasePluginAdapter: &BasePluginAdapter{
			manifest: manifest,
			client:   client,
			status:   plugins.StatusReady,
		},
		plugin: plugin,
	}
}

func (a *EmbedderPluginAdapter) Initialize(ctx context.Context, config map[string]interface{}) error {
	a.setStatus(plugins.StatusLoading)

	stringConfig := make(map[string]string)
	for k, v := range config {
		if str, ok := v.(string); ok {
			stringConfig[k] = str
		}
	}

	err := a.plugin.Initialize(ctx, stringConfig)
	if err != nil {
		a.setStatus(plugins.StatusError)
		return err
	}

	a.setStatus(plugins.StatusReady)
	return nil
}

func (a *EmbedderPluginAdapter) Shutdown(ctx context.Context) error {
	err := a.plugin.Shutdown(ctx)
	if err != nil {
		return err
	}
	a.setStatus(plugins.StatusShutdown)
	return nil
}

func (a *EmbedderPluginAdapter) Health(ctx context.Context) error {
	err := a.plugin.Health(ctx)
	if err != nil {
		a.setStatus(plugins.StatusError)
		return err
	}
	return nil
}

func (a *EmbedderPluginAdapter) GetPlugin() EmbedderProvider {
	return a.plugin
}
