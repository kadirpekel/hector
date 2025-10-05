package grpc

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"github.com/kadirpekel/hector/plugins"
)

// ============================================================================
// BASE PLUGIN ADAPTER
// ============================================================================

// BasePluginAdapter provides common functionality for all plugin adapters
type BasePluginAdapter struct {
	manifest *plugins.PluginManifest
	client   *plugin.Client
	status   plugins.PluginStatus
}

// GetManifest returns the plugin manifest
func (a *BasePluginAdapter) GetManifest() *plugins.PluginManifest {
	return a.manifest
}

// GetStatus returns the current plugin status
func (a *BasePluginAdapter) GetStatus() plugins.PluginStatus {
	return a.status
}

// setStatus updates the plugin status
func (a *BasePluginAdapter) setStatus(status plugins.PluginStatus) {
	a.status = status
}

// ============================================================================
// LLM PLUGIN ADAPTER
// ============================================================================

// LLMPluginAdapter adapts an LLM plugin to the Plugin interface
type LLMPluginAdapter struct {
	*BasePluginAdapter
	plugin LLMProvider
}

// NewLLMPluginAdapter creates a new LLM plugin adapter
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

// Initialize initializes the plugin
func (a *LLMPluginAdapter) Initialize(ctx context.Context, config map[string]interface{}) error {
	a.setStatus(plugins.StatusLoading)

	// Convert config to map[string]string for the plugin
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

// Shutdown shuts down the plugin
func (a *LLMPluginAdapter) Shutdown(ctx context.Context) error {
	err := a.plugin.Shutdown(ctx)
	if err != nil {
		return err
	}
	a.setStatus(plugins.StatusShutdown)
	return nil
}

// Health checks plugin health
func (a *LLMPluginAdapter) Health(ctx context.Context) error {
	err := a.plugin.Health(ctx)
	if err != nil {
		a.setStatus(plugins.StatusError)
		return err
	}
	return nil
}

// GetPlugin returns the underlying LLM plugin
func (a *LLMPluginAdapter) GetPlugin() LLMProvider {
	return a.plugin
}

// ============================================================================
// DATABASE PLUGIN ADAPTER
// ============================================================================

// DatabasePluginAdapter adapts a Database plugin to the Plugin interface
type DatabasePluginAdapter struct {
	*BasePluginAdapter
	plugin DatabaseProvider
}

// NewDatabasePluginAdapter creates a new Database plugin adapter
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

// Initialize initializes the plugin
func (a *DatabasePluginAdapter) Initialize(ctx context.Context, config map[string]interface{}) error {
	a.setStatus(plugins.StatusLoading)

	// Convert config to map[string]string for the plugin
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

// Shutdown shuts down the plugin
func (a *DatabasePluginAdapter) Shutdown(ctx context.Context) error {
	err := a.plugin.Shutdown(ctx)
	if err != nil {
		return err
	}
	a.setStatus(plugins.StatusShutdown)
	return nil
}

// Health checks plugin health
func (a *DatabasePluginAdapter) Health(ctx context.Context) error {
	err := a.plugin.Health(ctx)
	if err != nil {
		a.setStatus(plugins.StatusError)
		return err
	}
	return nil
}

// GetPlugin returns the underlying Database plugin
func (a *DatabasePluginAdapter) GetPlugin() DatabaseProvider {
	return a.plugin
}

// ============================================================================
// EMBEDDER PLUGIN ADAPTER
// ============================================================================

// EmbedderPluginAdapter adapts an Embedder plugin to the Plugin interface
type EmbedderPluginAdapter struct {
	*BasePluginAdapter
	plugin EmbedderProvider
}

// NewEmbedderPluginAdapter creates a new Embedder plugin adapter
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

// Initialize initializes the plugin
func (a *EmbedderPluginAdapter) Initialize(ctx context.Context, config map[string]interface{}) error {
	a.setStatus(plugins.StatusLoading)

	// Convert config to map[string]string for the plugin
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

// Shutdown shuts down the plugin
func (a *EmbedderPluginAdapter) Shutdown(ctx context.Context) error {
	err := a.plugin.Shutdown(ctx)
	if err != nil {
		return err
	}
	a.setStatus(plugins.StatusShutdown)
	return nil
}

// Health checks plugin health
func (a *EmbedderPluginAdapter) Health(ctx context.Context) error {
	err := a.plugin.Health(ctx)
	if err != nil {
		a.setStatus(plugins.StatusError)
		return err
	}
	return nil
}

// GetPlugin returns the underlying Embedder plugin
func (a *EmbedderPluginAdapter) GetPlugin() EmbedderProvider {
	return a.plugin
}
