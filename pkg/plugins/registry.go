package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/registry"
)

type PluginRegistry struct {
	*registry.BaseRegistry[Plugin]
	mu                  sync.RWMutex
	loaders             map[PluginProtocol]PluginLoader
	pluginsByType       map[PluginType][]string
	lifecycleHooks      *PluginLifecycleHooks
	autoRestart         bool
	restartAttempts     int
	healthCheckInterval time.Duration
	stopHealthCheck     chan struct{}
}

type PluginRegistryConfig struct {
	AutoRestart         bool
	MaxRestartAttempts  int
	HealthCheckInterval time.Duration
	LifecycleHooks      *PluginLifecycleHooks
}

func NewPluginRegistry(config *PluginRegistryConfig) *PluginRegistry {
	if config == nil {
		config = &PluginRegistryConfig{
			AutoRestart:         true,
			MaxRestartAttempts:  3,
			HealthCheckInterval: 30 * time.Second,
		}
	}

	return &PluginRegistry{
		BaseRegistry:        registry.NewBaseRegistry[Plugin](),
		loaders:             make(map[PluginProtocol]PluginLoader),
		pluginsByType:       make(map[PluginType][]string),
		lifecycleHooks:      config.LifecycleHooks,
		autoRestart:         config.AutoRestart,
		restartAttempts:     config.MaxRestartAttempts,
		healthCheckInterval: config.HealthCheckInterval,
		stopHealthCheck:     make(chan struct{}),
	}
}

func (r *PluginRegistry) RegisterLoader(loader PluginLoader) error {
	if loader == nil {
		return fmt.Errorf("loader cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	protocol := loader.SupportedProtocol()
	if _, exists := r.loaders[protocol]; exists {
		return fmt.Errorf("loader for protocol '%s' already registered", protocol)
	}

	r.loaders[protocol] = loader
	return nil
}

func (r *PluginRegistry) GetLoader(protocol PluginProtocol) (PluginLoader, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	loader, exists := r.loaders[protocol]
	if !exists {
		return nil, fmt.Errorf("no loader registered for protocol '%s'", protocol)
	}

	return loader, nil
}

func (r *PluginRegistry) LoadPlugin(ctx context.Context, config *PluginConfig) error {
	if config == nil {
		return NewPluginError("", "LoadPlugin", "config cannot be nil", nil)
	}

	if !config.Enabled {
		return nil
	}

	loader, err := r.GetLoader(config.Type)
	if err != nil {
		return NewPluginError(config.Name, "LoadPlugin", "failed to get loader", err)
	}

	if err := loader.Validate(ctx, config.Path); err != nil {
		return NewPluginError(config.Name, "LoadPlugin", "validation failed", err)
	}

	if r.lifecycleHooks != nil && r.lifecycleHooks.BeforeLoad != nil {
		if err := r.lifecycleHooks.BeforeLoad(ctx, nil); err != nil {
			return NewPluginError(config.Name, "LoadPlugin", "before load hook failed", err)
		}
	}

	plugin, err := loader.Load(ctx, config)
	if err != nil {
		return NewPluginError(config.Name, "LoadPlugin", "failed to load", err)
	}

	if r.lifecycleHooks != nil && r.lifecycleHooks.AfterLoad != nil {
		if err := r.lifecycleHooks.AfterLoad(ctx, plugin); err != nil {

			_ = loader.Unload(ctx, plugin)
			return NewPluginError(config.Name, "LoadPlugin", "after load hook failed", err)
		}
	}

	if r.lifecycleHooks != nil && r.lifecycleHooks.BeforeInit != nil {
		if err := r.lifecycleHooks.BeforeInit(ctx, plugin); err != nil {
			_ = loader.Unload(ctx, plugin)
			return NewPluginError(config.Name, "LoadPlugin", "before init hook failed", err)
		}
	}

	if err := plugin.Initialize(ctx, config.Config); err != nil {
		_ = loader.Unload(ctx, plugin)
		return NewPluginError(config.Name, "LoadPlugin", "initialization failed", err)
	}

	if r.lifecycleHooks != nil && r.lifecycleHooks.AfterInit != nil {
		if err := r.lifecycleHooks.AfterInit(ctx, plugin); err != nil {
			_ = plugin.Shutdown(ctx)
			_ = loader.Unload(ctx, plugin)
			return NewPluginError(config.Name, "LoadPlugin", "after init hook failed", err)
		}
	}

	if err := r.Register(config.Name, plugin); err != nil {
		_ = plugin.Shutdown(ctx)
		_ = loader.Unload(ctx, plugin)
		return NewPluginError(config.Name, "LoadPlugin", "registration failed", err)
	}

	manifest := plugin.GetManifest()
	if manifest != nil {
		r.mu.Lock()
		r.pluginsByType[manifest.Type] = append(r.pluginsByType[manifest.Type], config.Name)
		r.mu.Unlock()
	}

	return nil
}

func (r *PluginRegistry) UnloadPlugin(ctx context.Context, name string) error {
	plugin, exists := r.Get(name)
	if !exists {
		return NewPluginError(name, "UnloadPlugin", "plugin not found", ErrPluginNotFound)
	}

	if r.lifecycleHooks != nil && r.lifecycleHooks.BeforeUnload != nil {
		if err := r.lifecycleHooks.BeforeUnload(ctx, plugin); err != nil {
			return NewPluginError(name, "UnloadPlugin", "before unload hook failed", err)
		}
	}

	if err := plugin.Shutdown(ctx); err != nil {
		return NewPluginError(name, "UnloadPlugin", "shutdown failed", err)
	}

	manifest := plugin.GetManifest()

	if manifest != nil {
		loader, err := r.GetLoader(manifest.Protocol)
		if err != nil {
			return NewPluginError(name, "UnloadPlugin", "failed to get loader", err)
		}

		if err := loader.Unload(ctx, plugin); err != nil {
			return NewPluginError(name, "UnloadPlugin", "unload failed", err)
		}
	}

	if err := r.Remove(name); err != nil {
		return NewPluginError(name, "UnloadPlugin", "removal from registry failed", err)
	}

	if manifest != nil {
		r.mu.Lock()
		if plugins, exists := r.pluginsByType[manifest.Type]; exists {
			for i, pName := range plugins {
				if pName == name {
					r.pluginsByType[manifest.Type] = append(plugins[:i], plugins[i+1:]...)
					break
				}
			}
		}
		r.mu.Unlock()
	}

	if r.lifecycleHooks != nil && r.lifecycleHooks.AfterUnload != nil {
		if err := r.lifecycleHooks.AfterUnload(ctx, plugin); err != nil {
			return NewPluginError(name, "UnloadPlugin", "after unload hook failed", err)
		}
	}

	return nil
}

func (r *PluginRegistry) GetPlugin(name string) (Plugin, error) {
	plugin, exists := r.Get(name)
	if !exists {
		return nil, NewPluginError(name, "GetPlugin", "not found", ErrPluginNotFound)
	}
	return plugin, nil
}

func (r *PluginRegistry) GetPluginsByType(pluginType PluginType) ([]Plugin, error) {
	r.mu.RLock()
	pluginNames, exists := r.pluginsByType[pluginType]
	r.mu.RUnlock()

	if !exists || len(pluginNames) == 0 {
		return []Plugin{}, nil
	}

	plugins := make([]Plugin, 0, len(pluginNames))
	for _, name := range pluginNames {
		if plugin, exists := r.Get(name); exists {
			plugins = append(plugins, plugin)
		}
	}

	return plugins, nil
}

func (r *PluginRegistry) ListPlugins() []string {
	plugins := r.List()
	names := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		manifest := plugin.GetManifest()
		if manifest != nil {
			names = append(names, manifest.Name)
		}
	}
	return names
}

func (r *PluginRegistry) GetPluginStatus(name string) (PluginStatus, error) {
	plugin, err := r.GetPlugin(name)
	if err != nil {
		return StatusUnloaded, err
	}
	return plugin.GetStatus(), nil
}

func (r *PluginRegistry) StartHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(r.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopHealthCheck:
			return
		case <-ticker.C:
			r.performHealthChecks(ctx)
		}
	}
}

func (r *PluginRegistry) StopHealthChecks() {
	close(r.stopHealthCheck)
}

func (r *PluginRegistry) performHealthChecks(ctx context.Context) {
	plugins := r.List()
	for _, plugin := range plugins {
		manifest := plugin.GetManifest()
		if manifest == nil {
			continue
		}

		if err := plugin.Health(ctx); err != nil {

			if r.autoRestart && plugin.GetStatus() == StatusCrashed {

				if r.lifecycleHooks != nil && r.lifecycleHooks.OnCrash != nil {
					_ = r.lifecycleHooks.OnCrash(ctx, plugin)
				}

			}
		}
	}
}

func (r *PluginRegistry) Shutdown(ctx context.Context) error {
	r.StopHealthChecks()

	plugins := r.List()
	var errors []error

	for _, plugin := range plugins {
		manifest := plugin.GetManifest()
		if manifest == nil {
			continue
		}

		if err := r.UnloadPlugin(ctx, manifest.Name); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to shutdown %d plugins: %v", len(errors), errors)
	}

	return nil
}
