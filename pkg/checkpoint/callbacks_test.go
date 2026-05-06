package checkpoint

import (
	"context"
	"testing"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/model"
	"github.com/verikod/hector/pkg/session"
)

// -- Mocks --

// mockTask implements CancellableTask with GetID
type mockTask struct {
	id string
}

func (m *mockTask) CancelExecution(string) bool             { return false }
func (m *mockTask) RegisterExecution(*agent.ChildExecution) {}
func (m *mockTask) UnregisterExecution(string)              {}
func (m *mockTask) GetID() string                           { return m.id }

// mockTool implements tool.Tool
type mockTool struct{}

func (m *mockTool) Name() string           { return "test-tool" }
func (m *mockTool) Description() string    { return "desc" }
func (m *mockTool) IsLongRunning() bool    { return false }
func (m *mockTool) RequiresApproval() bool { return false }

// mockContext implements agent.InvocationContext AND tool.Context
type mockContext struct {
	context.Context
	taskID      string
	sessionID   string
	userContent *agent.Content
	metadata    map[string]any
}

// InvocationContext methods
func (m *mockContext) InvocationID() string               { return "inv-123" }
func (m *mockContext) AgentName() string                  { return "test-agent" }
func (m *mockContext) UserContent() *agent.Content        { return m.userContent }
func (m *mockContext) ReadonlyState() agent.ReadonlyState { return nil }
func (m *mockContext) UserID() string                     { return "user-456" }
func (m *mockContext) AppName() string                    { return "test-app" }
func (m *mockContext) SessionID() string                  { return m.sessionID }
func (m *mockContext) Branch() string                     { return "root" }
func (m *mockContext) Artifacts() agent.Artifacts         { return nil }
func (m *mockContext) State() agent.State                 { return nil }
func (m *mockContext) Agent() agent.Agent                 { return nil }
func (m *mockContext) Session() agent.Session             { return nil }
func (m *mockContext) Memory() agent.Memory               { return nil }
func (m *mockContext) EndInvocation()                     {}
func (m *mockContext) Ended() bool                        { return false }
func (m *mockContext) RunConfig() *agent.RunConfig {
	return &agent.RunConfig{
		Task: &mockTask{id: m.taskID},
	}
}

// ToolContext methods
func (m *mockContext) FunctionCallID() string       { return "call-1" }
func (m *mockContext) Actions() *agent.EventActions { return nil }
func (m *mockContext) SearchMemory(ctx context.Context, query string) (*agent.MemorySearchResponse, error) {
	return nil, nil
}
func (m *mockContext) Task() agent.CancellableTask {
	return &mockTask{id: m.taskID}
}
func (m *mockContext) SetMetadata(metadata map[string]any) {
	if m.metadata == nil {
		m.metadata = make(map[string]any)
	}
	for k, v := range metadata {
		m.metadata[k] = v
	}
}
func (m *mockContext) Metadata() map[string]any { return m.metadata }

// -- Helpers --

func boolPtr(b bool) *bool { return &b }

// -- Tests --

func TestCallbacks(t *testing.T) {
	// Setup dependencies
	sessService := session.InMemoryService()
	cfg := &Config{
		Enabled:    boolPtr(true),
		Strategy:   "event",
		BeforeLLM:  boolPtr(true),
		AfterTools: boolPtr(true),
	}
	manager := NewManager(cfg, sessService)
	callbacks := NewCallbacks(manager)

	// Setup context
	ctx := context.Background()
	taskID := "task-789"
	sessionID := "session-abc"
	userID := "user-456"
	appName := "test-app"

	// Ensure session exists
	_, err := sessService.Create(ctx, &session.CreateRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
	})
	require.NoError(t, err)

	invCtx := &mockContext{
		Context:   ctx,
		taskID:    taskID,
		sessionID: sessionID,
		userContent: &agent.Content{
			Parts: []a2a.Part{
				a2a.TextPart{Text: "do something"},
			},
		},
	}

	// 1. Test BeforeModel (PhasePreLLM)
	t.Run("BeforeModel", func(t *testing.T) {
		req := &model.Request{}
		_, err := callbacks.BeforeModel(invCtx, req)
		require.NoError(t, err)

		// Verify checkpoint created
		state, err := manager.LoadCheckpoint(ctx, appName, userID, sessionID, taskID)
		require.NoError(t, err)
		assert.Equal(t, PhasePreLLM, state.Phase)
		assert.Equal(t, "do something", state.Query)
		assert.Equal(t, taskID, state.TaskID)
	})

	// 2. Test AfterModel (PhasePostLLM)
	t.Run("AfterModel", func(t *testing.T) {
		resp := &model.Response{
			Content: &model.Content{
				Parts: []a2a.Part{
					a2a.TextPart{Text: "done"},
				},
			},
		}
		_, err := callbacks.AfterModel(invCtx, resp, nil)
		require.NoError(t, err)

		// Verify checkpoint updated
		state, err := manager.LoadCheckpoint(ctx, appName, userID, sessionID, taskID)
		require.NoError(t, err)
		assert.Equal(t, PhasePostLLM, state.Phase)
	})

	// 2.1. Test AfterModel with partial response (no checkpoint write)
	t.Run("AfterModelPartialSkipped", func(t *testing.T) {
		// Seed a known checkpoint phase first
		_, err := callbacks.BeforeModel(invCtx, &model.Request{})
		require.NoError(t, err)

		state, err := manager.LoadCheckpoint(ctx, appName, userID, sessionID, taskID)
		require.NoError(t, err)
		assert.Equal(t, PhasePreLLM, state.Phase)

		resp := &model.Response{Partial: true}
		_, err = callbacks.AfterModel(invCtx, resp, nil)
		require.NoError(t, err)

		// Phase should remain unchanged because partial responses are skipped
		state, err = manager.LoadCheckpoint(ctx, appName, userID, sessionID, taskID)
		require.NoError(t, err)
		assert.Equal(t, PhasePreLLM, state.Phase)
	})

	// 3. Test AfterTool (PhasePostTool)
	t.Run("AfterTool", func(t *testing.T) {
		toolInstance := &mockTool{}
		_, err := callbacks.AfterTool(invCtx, toolInstance, nil, nil, nil)
		require.NoError(t, err)

		state, err := manager.LoadCheckpoint(ctx, appName, userID, sessionID, taskID)
		require.NoError(t, err)
		assert.Equal(t, PhasePostTool, state.Phase)
	})

	// 4. Test OnComplete (Clear Checkpoint)
	t.Run("OnComplete", func(t *testing.T) {
		_, err := callbacks.OnComplete(invCtx)
		require.NoError(t, err)

		// Verify checkpoint gone
		_, err = manager.LoadCheckpoint(ctx, appName, userID, sessionID, taskID)
		assert.Error(t, err) // Should fail to load
	})
}

func TestCallbacks_Disabled(t *testing.T) {
	sessService := session.InMemoryService()
	cfg := &Config{Enabled: boolPtr(false)}
	manager := NewManager(cfg, sessService)
	callbacks := NewCallbacks(manager)

	ctx := context.Background()
	invCtx := &mockContext{Context: ctx, taskID: "task-1", sessionID: "sess-1"}

	// Should not error even if session doesn't exist (because disabled)
	_, err := callbacks.BeforeModel(invCtx, nil)
	assert.NoError(t, err)
}
