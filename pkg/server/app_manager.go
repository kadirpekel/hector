package server

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/verikod/hector/pkg/app"
	"github.com/verikod/hector/pkg/config"
	"github.com/verikod/hector/pkg/notification"
	"github.com/verikod/hector/pkg/observability"
	"github.com/verikod/hector/pkg/rag"
	"github.com/verikod/hector/pkg/runner"
	"github.com/verikod/hector/pkg/task"
)

// Runtime defines the interface for an agent runtime.
// Implemented by pkg/runtime.Runtime.
type Runtime interface {
	// Closer closes the runtime and releases resources.
	Close() error
	// ListAgents returns the names of all configured agents.
	ListAgents() []string
	// RunnerConfig creating a runner config for an agent.
	RunnerConfig(agentName string) (*runner.Config, error)
	// DocumentStores returns the document stores for RAG status.
	DocumentStores() map[string]*rag.DocumentStore
	// Notifier returns the notifier service.
	Notifier() *notification.Notifier
}

// RuntimeFactory functions create a runtime for a given app configuration.
type RuntimeFactory func(ctx context.Context, appID string, cfg *config.AppConfig) (Runtime, error)

// AgentRuntime contains the runtime components for an agent.
type AgentRuntime struct {
	Executor   *Executor
	Card       *a2a.AgentCard // A2A agent card metadata
	Visibility string         // "public", "internal", "private"
}

// AppInstance holds the runtime and cached agent executors for an app.
type AppInstance struct {
	Config  *config.AppConfig
	Runtime Runtime
	Agents  map[string]*AgentRuntime
}

// webhookMatch represents a resolved webhook destination.
type webhookMatch struct {
	appID     string
	agentName string
}

// AppManager manages the lifecycle of loaded apps and their executors.
// It handles on-demand loading of app configurations and caching of executors.
type AppManager struct {
	appStore       app.Store
	runtimeFactory RuntimeFactory
	taskService    task.Service

	// Cache: appID -> AppInstance
	apps map[string]*AppInstance
	// Cache: path -> webhookMatch
	webhookCache map[string]webhookMatch
	mu           sync.RWMutex

	metrics *observability.Metrics
}

// NewAppManager creates a new AppManager.
func NewAppManager(store app.Store, factory RuntimeFactory, taskService task.Service, metrics *observability.Metrics) *AppManager {
	return &AppManager{
		appStore:       store,
		runtimeFactory: factory,
		taskService:    taskService,
		apps:           make(map[string]*AppInstance),
		webhookCache:   make(map[string]webhookMatch),
		metrics:        metrics,
	}
}

// DocumentStores returns the document stores from the default app.
// Implements DocumentStoreProvider for backward compatibility.
func (m *AppManager) DocumentStores() map[string]*rag.DocumentStore {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// For now, we only expose the default app's RAG status to the global /health endpoint
	if instance, ok := m.apps["default"]; ok && instance.Runtime != nil {
		return instance.Runtime.DocumentStores()
	}
	return nil
}

// GetRuntime returns the runtime components for a specific agent in a specific app.
func (m *AppManager) GetRuntime(ctx context.Context, appID, agentName string) (*AgentRuntime, error) {
	// Fast path: check read lock
	m.mu.RLock()
	if instance, ok := m.apps[appID]; ok {
		if runtime, ok := instance.Agents[agentName]; ok {
			m.mu.RUnlock()
			return runtime, nil
		}
	}
	m.mu.RUnlock()

	// Slow path: load app and build runtime
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check locking
	if instance, ok := m.apps[appID]; ok {
		if runtime, ok := instance.Agents[agentName]; ok {
			return runtime, nil
		}
		return nil, fmt.Errorf("agent %q not found in app %q", agentName, appID)
	}

	// Load app
	instance, err := m.loadAppLocked(ctx, appID)
	if err != nil {
		slog.Error("Failed to load app", "app", appID, "agent", agentName, "error", err)
		return nil, err
	}

	// Return requested runtime
	runtime, ok := instance.Agents[agentName]
	if !ok {
		return nil, fmt.Errorf("agent %q not configured for app %q", agentName, appID)
	}

	return runtime, nil
}

// Invalidate clears the cache for an app and closes its runtime.
// Should be called when app config is updated.
func (m *AppManager) Invalidate(appID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if instance, ok := m.apps[appID]; ok {
		// Close asynchronously to not block
		if instance.Runtime != nil {
			go func() {
				if err := instance.Runtime.Close(); err != nil {
					slog.Warn("Failed to close runtime", "app", appID, "error", err)
				}
			}()
		}
		delete(m.apps, appID)
	}

	// Always clear webhook cache for this app, even if not loaded
	for path, match := range m.webhookCache {
		if match.appID == appID {
			delete(m.webhookCache, path)
		}
	}

	slog.Debug("Invalidated app cache", "app", appID)
}

// UnloadApp synchronously closes an app's runtime and removes it from cache.
// Checks if the app is currently running before attempting to close.
func (m *AppManager) UnloadApp(ctx context.Context, appID string) error {
	m.mu.Lock()
	instance, ok := m.apps[appID]
	if !ok {
		m.mu.Unlock()
		return nil // Already unloaded
	}
	// Remove from map immediately so no new requests pick it up
	delete(m.apps, appID)

	// Always clear webhook cache for this app, even if not loaded
	for path, match := range m.webhookCache {
		if match.appID == appID {
			delete(m.webhookCache, path)
		}
	}

	m.mu.Unlock()

	if instance.Runtime != nil {
		slog.Info("Unloading app runtime", "app", appID)
		if err := instance.Runtime.Close(); err != nil {
			return fmt.Errorf("failed to close runtime: %w", err)
		}
		m.metrics.RecordAppUnload(appID)
	}
	return nil
}

// Close closes all managed runtimes.
func (m *AppManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for appID, instance := range m.apps {
		if instance.Runtime != nil {
			instance.Runtime.Close()
		}
		delete(m.apps, appID)
	}
	return nil
}

// GetAppConfig returns the configuration for an app.
// Loads the app if not active.
func (m *AppManager) GetAppConfig(ctx context.Context, appID string) (*config.AppConfig, error) {
	m.mu.RLock()
	instance, ok := m.apps[appID]
	m.mu.RUnlock()

	if !ok {
		m.mu.Lock()
		defer m.mu.Unlock()
		// Double check
		if cached, ok := m.apps[appID]; ok {
			return cached.Config, nil
		}
		var err error
		instance, err = m.loadAppLocked(ctx, appID)
		if err != nil {
			return nil, err
		}
	}
	return instance.Config, nil
}

// ListAgents returns all configured agent runtimes for an app.
// Used for discovery and default agent resolution.
func (m *AppManager) ListAgents(ctx context.Context, appID string) ([]*AgentRuntime, error) {
	m.mu.RLock()
	instance, ok := m.apps[appID]
	m.mu.RUnlock()

	if !ok {
		m.mu.Lock()
		// Double check
		if cached, ok := m.apps[appID]; ok {
			instance = cached
		} else {
			var err error
			instance, err = m.loadAppLocked(ctx, appID)
			if err != nil {
				m.mu.Unlock()
				return nil, err
			}
		}
		m.mu.Unlock()
	}

	// Convert map to slice
	list := make([]*AgentRuntime, 0, len(instance.Agents))
	for _, rt := range instance.Agents {
		list = append(list, rt)
	}
	return list, nil
}

// Preload ensures an app is loaded into memory.
// Used for default app bootstrapping.
func (m *AppManager) Preload(ctx context.Context, appID string) error {
	_, err := m.ListAgents(ctx, appID)
	return err
}

// loadAppLocked loads app from store and builds runtime. Caller must hold lock.
func (m *AppManager) loadAppLocked(ctx context.Context, appID string) (*AppInstance, error) {
	// Load app from store
	slog.Debug("Loading app configuration", "app", appID)
	appEntity, err := m.appStore.Get(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to load app %q: %w", appID, err)
	}

	// Parse config
	cfg, err := config.ParseAppConfigJSON([]byte(appEntity.ConfigJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse config for app %q: %w", appID, err)
	}

	// Build Runtime using factory
	start := time.Now()
	rt, err := m.runtimeFactory(ctx, appID, cfg)
	if err != nil {
		m.metrics.RecordAppLoad(appID, "error", time.Since(start))
		return nil, fmt.Errorf("failed to build runtime for app %q: %w", appID, err)
	}
	slog.Info("Built runtime for app", "app", appID, "duration", time.Since(start))
	m.metrics.RecordAppLoad(appID, "success", time.Since(start))

	// Build Executors and Cards from Runtime
	agents := make(map[string]*AgentRuntime)
	for _, name := range rt.ListAgents() {
		runnerCfg, err := rt.RunnerConfig(name)
		if err != nil {
			slog.Warn("Failed to create runner config", "app", appID, "agent", name, "error", err)
			continue
		}

		// Create Executor
		execConfig := ExecutorConfig{
			RunnerConfig: *runnerCfg,
			TaskService:  m.taskService,
			Notifier:     rt.Notifier(),
		}
		exec := NewExecutor(execConfig)

		// Build Agent Card
		agentCfg, ok := cfg.Agents[name]

		// Minimum required fields + reasonable defaults
		card := &a2a.AgentCard{
			Name:            name,
			Version:         cfg.Version,
			ProtocolVersion: "1.0",
			URL:             fmt.Sprintf("/agents/%s", name),
			Capabilities: a2a.AgentCapabilities{
				Streaming: agentCfg.Streaming == nil || *agentCfg.Streaming,
			},
		}

		visibility := "private" // Default safe visibility
		if ok {
			if agentCfg.Name != "" {
				card.Name = agentCfg.Name
			}
			card.Description = agentCfg.Description

			if len(agentCfg.InputModes) > 0 {
				card.DefaultInputModes = agentCfg.InputModes
			} else {
				card.DefaultInputModes = []string{"text/plain"}
			}
			if len(agentCfg.OutputModes) > 0 {
				card.DefaultOutputModes = agentCfg.OutputModes
			} else {
				card.DefaultOutputModes = []string{"text/plain"}
			}

			// Map Skills
			if len(agentCfg.Skills) > 0 {
				card.Skills = make([]a2a.AgentSkill, len(agentCfg.Skills))
				for i, s := range agentCfg.Skills {
					card.Skills[i] = a2a.AgentSkill{
						ID:          s.ID,
						Name:        s.Name,
						Description: s.Description,
						Tags:        s.Tags,
						Examples:    s.Examples,
						// Use defaults for modes since SkillConfig doesn't have them
						InputModes:  agentCfg.InputModes,
						OutputModes: agentCfg.OutputModes,
					}
				}
			}

			if agentCfg.Visibility != "" {
				visibility = agentCfg.Visibility
			}
		} else {
			// Default logic if config missing (unlikely if ListAgents works)
			card.DefaultInputModes = []string{"text/plain"}
			card.DefaultOutputModes = []string{"text/plain"}
		}

		agents[name] = &AgentRuntime{
			Executor:   exec,
			Card:       card,
			Visibility: visibility,
		}
	}

	instance := &AppInstance{
		Config:  cfg,
		Runtime: rt,
		Agents:  agents,
	}
	m.apps[appID] = instance

	return instance, nil
}

// ResolveWebhook resolves an app and agent from a webhook path.
func (m *AppManager) ResolveWebhook(ctx context.Context, path string) (appID, agentName string, err error) {
	// 1. Check cache
	m.mu.RLock()
	if match, ok := m.webhookCache[path]; ok {
		m.mu.RUnlock()
		return match.appID, match.agentName, nil
	}
	m.mu.RUnlock()

	// 2. Scan active apps first
	m.mu.RLock()
	for id, instance := range m.apps {
		if name, ok := m.LookupWebhookInConfig(path, instance.Config); ok {
			m.mu.RUnlock()
			m.updateWebhookCache(path, id, name)
			return id, name, nil
		}
	}
	m.mu.RUnlock()

	// 3. Scan all apps in store
	apps, err := m.appStore.List(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to list apps for webhook resolution: %w", err)
	}

	for _, a := range apps {
		// Skip already checked active apps
		m.mu.RLock()
		_, active := m.apps[a.ID]
		m.mu.RUnlock()
		if active {
			continue
		}

		cfg, err := config.ParseAppConfigJSON([]byte(a.ConfigJSON))
		if err != nil {
			slog.Warn("Failed to parse app config during webhook resolution", "app", a.ID, "error", err)
			continue
		}

		if name, ok := m.LookupWebhookInConfig(path, cfg); ok {
			m.updateWebhookCache(path, a.ID, name)
			return a.ID, name, nil
		}
	}

	return "", "", fmt.Errorf("no agent found for webhook path: %s", path)
}

// LookupWebhookInConfig searches for a matching webhook trigger in an app config.
func (m *AppManager) LookupWebhookInConfig(path string, cfg *config.AppConfig) (string, bool) {
	if cfg == nil {
		return "", false
	}
	// Logic consistent with handleWebhook's matching convention
	for name, agentCfg := range cfg.Agents {
		if agentCfg.Trigger != nil && agentCfg.Trigger.Type == config.TriggerTypeWebhook {
			// Exact match
			if agentCfg.Trigger.Path != "" && agentCfg.Trigger.Path == path {
				return name, true
			}
			// Convention match: /webhooks/{agentName}
			convention := "/webhooks/" + name
			if convention == path {
				// Only match convention if the agent has NOT defined a custom path
				if agentCfg.Trigger.Path == "" {
					return name, true
				}
			}
		}
	}
	return "", false
}

func (m *AppManager) updateWebhookCache(path, appID, agentName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webhookCache[path] = webhookMatch{appID: appID, agentName: agentName}
}
