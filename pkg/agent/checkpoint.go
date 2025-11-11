package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// checkpointExecution creates a generic checkpoint using the shared foundation
// This function wraps CaptureExecutionState and SaveExecutionStateToSession
// with checkpoint metadata (phase, type, time)
func (a *Agent) checkpointExecution(
	ctx context.Context,
	taskID string,
	phase ExecutionPhase,
	checkpointType CheckpointType,
	reasoningState *reasoning.ReasoningState,
	pendingToolCall *protocol.ToolCall,
) error {
	sessionID := getSessionIDFromContext(ctx)
	if sessionID == "" {
		return fmt.Errorf("session ID required for checkpointing")
	}

	query := reasoningState.Query()

	// Use existing CaptureExecutionState (shared foundation)
	execState := CaptureExecutionState(
		taskID,
		sessionID,
		query,
		reasoningState,
		pendingToolCall,
	)

	// Add checkpoint metadata (NEW: extends ExecutionState)
	execState.Phase = phase
	execState.CheckpointType = checkpointType
	execState.CheckpointTime = time.Now()

	// Use existing SaveExecutionStateToSession (shared foundation)
	return a.SaveExecutionStateToSession(ctx, sessionID, taskID, execState)
}

// shouldCheckpointInterval determines if we should checkpoint at this iteration
// based on interval configuration
func (a *Agent) shouldCheckpointInterval(iteration int, intervalEveryN int) bool {
	if intervalEveryN <= 0 {
		return false // Interval checkpointing disabled
	}
	return iteration > 0 && iteration%intervalEveryN == 0
}

// getCheckpointInterval returns the configured checkpoint interval
func (a *Agent) getCheckpointInterval() int {
	if a.config == nil || a.config.Task == nil {
		return 0 // Disabled by default
	}

	checkpointCfg := a.config.Task.Checkpoint
	if checkpointCfg == nil || !checkpointCfg.Enabled {
		return 0 // Checkpointing disabled
	}

	// Only return interval if strategy is "interval" or "hybrid"
	if checkpointCfg.Strategy != "interval" && checkpointCfg.Strategy != "hybrid" {
		return 0
	}

	if checkpointCfg.Interval == nil {
		return 0
	}

	return checkpointCfg.Interval.EveryNIterations
}

// isCheckpointEnabled returns true if checkpointing is enabled
func (a *Agent) isCheckpointEnabled() bool {
	if a.config == nil || a.config.Task == nil {
		return false
	}
	return a.config.Task.Checkpoint != nil && a.config.Task.Checkpoint.Enabled
}
