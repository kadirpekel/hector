package checkpoint

import (
	"log/slog"

	"github.com/a2aproject/a2a-go/a2a"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/model"
	"github.com/verikod/hector/pkg/tool"
)

// Callbacks adapts agent callbacks to checkpoint manager hooks.
// This bridges the agent execution flow (llmagent) to the checkpointing system.
type Callbacks struct {
	manager *Manager
}

// NewCallbacks creates a new Callbacks adapter.
func NewCallbacks(manager *Manager) *Callbacks {
	return &Callbacks{
		manager: manager,
	}
}

// captureState creates a checkpoint State from the context.
func (c *Callbacks) captureState(ctx agent.CallbackContext) *State {
	// Extract query string from user content
	var query string
	if content := ctx.UserContent(); content != nil {
		// Find text part
		for _, part := range content.Parts {
			if txt, ok := part.(a2a.TextPart); ok {
				query += txt.Text
			}
		}
	}

	// Get TaskID from RunConfig (requires InvocationContext)
	var taskID string
	if invCtx, ok := ctx.(agent.InvocationContext); ok {
		if rc := invCtx.RunConfig(); rc != nil && rc.Task != nil {
			taskID = rc.Task.GetID()
		}
	}
	// Note: If taskID is empty, SaveCheckpoint will likely fail or be skipped,
	// which is correct behavior (no checkpointing for untracked invocations).

	state := NewState(
		taskID,
		ctx.SessionID(),
		ctx.UserID(),
		ctx.AppName(),
		query,
		ctx.AgentName(),
		ctx.InvocationID(),
	)

	// Capture architecture (llm) and branch info
	state.AgentState = &AgentStateSnapshot{
		Branch: ctx.Branch(),
	}

	return state
}

// BeforeModel implements agent.BeforeModelCallback.
// Creates a checkpoint before the LLM is called (PhasePreLLM).
func (c *Callbacks) BeforeModel(ctx agent.CallbackContext, req *model.Request) (*model.Response, error) {
	if !c.manager.ShouldCheckpointBeforeLLM() {
		return nil, nil
	}

	state := c.captureState(ctx)
	// If no task ID, skip checkpointing
	if state.TaskID == "" {
		return nil, nil
	}

	state.WithPhase(PhasePreLLM)

	// Best effort: don't fail execution if checkpointing fails
	if err := c.manager.SaveCheckpoint(ctx, state); err != nil {
		slog.Warn("Failed to save pre-LLM checkpoint",
			"task_id", state.TaskID,
			"error", err)
	}

	return nil, nil
}

// AfterModel implements agent.AfterModelCallback.
// Creates a checkpoint after the LLM response is received (PhasePostLLM).
func (c *Callbacks) AfterModel(ctx agent.CallbackContext, resp *model.Response, llmErr error) (*model.Response, error) {
	if !c.manager.IsEnabled() {
		return nil, nil
	}

	// Don't checkpoint if LLM call failed (unless we want to save error state?)
	if llmErr != nil {
		// Optionally save error checkpoint
		state := c.captureState(ctx)
		if state.TaskID != "" {
			state.WithError(llmErr)
			_ = c.manager.SaveCheckpoint(ctx, state)
		}
		return nil, nil
	}

	state := c.captureState(ctx)
	if state.TaskID == "" {
		return nil, nil
	}

	state.WithPhase(PhasePostLLM)

	if err := c.manager.SaveCheckpoint(ctx, state); err != nil {
		slog.Warn("Failed to save post-LLM checkpoint",
			"task_id", state.TaskID,
			"error", err)
	}

	return nil, nil
}

// BeforeTool implements agent.BeforeToolCallback.
// Creates a checkpoint before a tool is executed (PhaseToolExecution).
func (c *Callbacks) BeforeTool(ctx tool.Context, t tool.Tool, args map[string]any) (map[string]any, error) {
	// Skipping tool checkpoints implemented to keep focused on core recovery.
	return nil, nil
}

// AfterTool implements agent.AfterToolCallback.
// Creates a checkpoint after a tool execution if configured.
func (c *Callbacks) AfterTool(ctx tool.Context, t tool.Tool, args, result map[string]any, err error) (map[string]any, error) {
	if !c.manager.ShouldCheckpointAfterTools() {
		return nil, nil
	}

	state := c.captureState(ctx)
	if state.TaskID == "" {
		return nil, nil
	}

	state.WithPhase(PhasePostTool)

	if err := c.manager.SaveCheckpoint(ctx, state); err != nil {
		slog.Warn("Failed to save post-tool checkpoint",
			"task_id", state.TaskID,
			"tool", t.Name(),
			"error", err)
	}

	return nil, nil
}

// OnComplete implements agent.AfterAgentCallback.
// Clears the checkpoint when the agent completes successfully.
func (c *Callbacks) OnComplete(ctx agent.CallbackContext) (*a2a.Message, error) {
	if !c.manager.IsEnabled() {
		return nil, nil
	}

	state := c.captureState(ctx)
	if state.TaskID == "" {
		return nil, nil
	}

	if err := c.manager.ClearCheckpoint(ctx, state.AppName, state.UserID, state.SessionID, state.TaskID); err != nil {
		slog.Warn("Failed to clear checkpoint on completion",
			"task_id", state.TaskID,
			"error", err)
	}

	return nil, nil
}
