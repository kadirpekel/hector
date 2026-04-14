package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/verikod/hector/pkg/app"
	"github.com/verikod/hector/pkg/config"
)

// mockStoreFull implements app.Store for testing
type mockStoreFull struct {
	apps map[string]*app.App
}

func (m *mockStoreFull) Create(ctx context.Context, a *app.App) (*app.App, error) {
	m.apps[a.ID] = a
	return a, nil
}

func (m *mockStoreFull) Get(ctx context.Context, id string) (*app.App, error) {
	if a, ok := m.apps[id]; ok {
		return a, nil
	}
	return nil, app.ErrAppNotFound
}

func (m *mockStoreFull) List(ctx context.Context) ([]*app.App, error) {
	var list []*app.App
	for _, a := range m.apps {
		list = append(list, a)
	}
	return list, nil
}

func (m *mockStoreFull) Update(ctx context.Context, a *app.App) error        { return nil }
func (m *mockStoreFull) Delete(ctx context.Context, id string) error         { return nil }
func (m *mockStoreFull) Exists(ctx context.Context, id string) (bool, error) { return true, nil }

func TestResolveWebhook(t *testing.T) {
	ctx := context.Background()

	// 1. Setup Apps
	app1Cfg := &config.AppConfig{
		LLMs: map[string]*config.LLMConfig{
			"default": {Provider: config.LLMProviderOllama, Model: "qwen3"},
		},
		Agents: map[string]*config.AgentConfig{
			"agent1": {
				LLM: "default",
				Trigger: &config.TriggerConfig{
					Type: config.TriggerTypeWebhook,
					Path: "/custom1",
				},
			},
		},
	}
	app1Bytes, _ := json.Marshal(app1Cfg)

	app2Cfg := &config.AppConfig{
		LLMs: map[string]*config.LLMConfig{
			"default": {Provider: config.LLMProviderOllama, Model: "qwen3"},
		},
		Agents: map[string]*config.AgentConfig{
			"agent2": {
				LLM: "default",
				Trigger: &config.TriggerConfig{
					Type: config.TriggerTypeWebhook,
					// Convention only
				},
			},
		},
	}
	app2Bytes, _ := json.Marshal(app2Cfg)

	store := &mockStoreFull{
		apps: map[string]*app.App{
			"app1": {ID: "app1", ConfigJSON: string(app1Bytes)},
			"app2": {ID: "app2", ConfigJSON: string(app2Bytes)},
		},
	}

	factory := func(ctx context.Context, appID string, cfg *config.AppConfig) (Runtime, error) {
		return &mockRuntime{agents: []string{}}, nil // empty for this test
	}

	am := NewAppManager(store, factory, nil, nil)

	t.Run("ResolveExactMatch", func(t *testing.T) {
		appID, agentName, err := am.ResolveWebhook(ctx, "/custom1")
		if err != nil {
			t.Fatalf("Failed to resolve exact match: %v", err)
		}
		if appID != "app1" || agentName != "agent1" {
			t.Errorf("Expected app1/agent1, got %s/%s", appID, agentName)
		}

		// Verify Cache
		am.mu.RLock()
		if match, ok := am.webhookCache["/custom1"]; !ok || match.appID != "app1" {
			t.Error("Match not cached")
		}
		am.mu.RUnlock()
	})

	t.Run("ResolveConventionMatch", func(t *testing.T) {
		appID, agentName, err := am.ResolveWebhook(ctx, "/webhooks/agent2")
		if err != nil {
			t.Fatalf("Failed to resolve convention match: %v", err)
		}
		if appID != "app2" || agentName != "agent2" {
			t.Errorf("Expected app2/agent2, got %s/%s", appID, agentName)
		}
	})

	t.Run("ResolveActiveAppPriority", func(t *testing.T) {
		// Mock an active app in memory
		app3Cfg := &config.AppConfig{
			LLMs: map[string]*config.LLMConfig{
				"default": {Provider: config.LLMProviderOllama, Model: "qwen3"},
			},
			Agents: map[string]*config.AgentConfig{
				"agent3": {
					LLM: "default",
					Trigger: &config.TriggerConfig{
						Type: config.TriggerTypeWebhook,
						Path: "/active-hook",
					},
				},
			},
		}
		am.mu.Lock()
		am.apps["app3"] = &AppInstance{Config: app3Cfg}
		am.mu.Unlock()

		appID, agentName, err := am.ResolveWebhook(ctx, "/active-hook")
		if err != nil {
			t.Fatalf("Failed to resolve active hook: %v", err)
		}
		if appID != "app3" || agentName != "agent3" {
			t.Errorf("Expected app3/agent3, got %s/%s", appID, agentName)
		}
	})

	t.Run("CacheHit", func(t *testing.T) {
		// Inject into cache manually
		am.mu.Lock()
		am.webhookCache["/cached"] = webhookMatch{appID: "appX", agentName: "agentX"}
		am.mu.Unlock()

		appID, agentName, err := am.ResolveWebhook(ctx, "/cached")
		if err != nil {
			t.Fatalf("Failed to resolve cached hook: %v", err)
		}
		if appID != "appX" || agentName != "agentX" {
			t.Errorf("Expected appX/agentX, got %s/%s", appID, agentName)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, _, err := am.ResolveWebhook(ctx, "/nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent webhook")
		}
	})

	t.Run("InvalidateCache", func(t *testing.T) {
		// Setup cache
		am.updateWebhookCache("/to-delete", "app-to-delete", "agent-to-delete")

		// Invalidate
		am.Invalidate("app-to-delete")

		// Check cache
		am.mu.RLock()
		if _, ok := am.webhookCache["/to-delete"]; ok {
			t.Error("Cache entry not cleared after Invalidate")
		}
		am.mu.RUnlock()
	})
}
