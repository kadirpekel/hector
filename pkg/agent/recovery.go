package agent

import (
	"context"
	"log/slog"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// RecoverPendingTasks recovers tasks with checkpoints on startup
// This is called during agent initialization to resume tasks that were interrupted
func (a *Agent) RecoverPendingTasks(ctx context.Context) error {
	if a.services.Task() == nil {
		return nil // No task service, nothing to recover
	}

	// Check if recovery is enabled
	if !a.isRecoveryEnabled() {
		slog.Debug("Checkpoint recovery disabled, skipping", "agent", a.id)
		return nil
	}

	// Find all tasks in WORKING or INPUT_REQUIRED state
	// These are the only states that can have checkpoints (A2A compliance)
	recoverableStates := []pb.TaskState{
		pb.TaskState_TASK_STATE_WORKING,
		pb.TaskState_TASK_STATE_INPUT_REQUIRED,
	}

	var allPendingTasks []*pb.Task
	for _, state := range recoverableStates {
		tasks, _, _, err := a.services.Task().ListTasks(ctx, "", state, 100, "")
		if err != nil {
			slog.Error("Failed to list tasks in state", "agent", a.id, "state", state, "error", err)
			continue
		}
		allPendingTasks = append(allPendingTasks, tasks...)
	}

	if len(allPendingTasks) == 0 {
		slog.Debug("No pending tasks to recover", "agent", a.id)
		return nil
	}

	slog.Info("Found pending tasks, checking for checkpoints", "agent", a.id, "count", len(allPendingTasks))

	recoveredCount := 0
	for _, task := range allPendingTasks {
		// A2A Compliance: Validate state is still recoverable
		if isTerminalState(task.Status.State) {
			slog.Debug("Task in terminal state, skipping recovery", "agent", a.id, "task", task.Id)
			continue
		}

		// Check if task has a checkpoint
		sessionID := task.ContextId
		if sessionID == "" {
			slog.Debug("Task has no context ID, skipping recovery", "agent", a.id, "task", task.Id)
			continue
		}

		execState, err := a.LoadExecutionStateFromSession(ctx, sessionID, task.Id)
		if err != nil {
			// No checkpoint found - this is OK (task might not have checkpointing enabled)
			// If task is WORKING but has no checkpoint, mark as FAILED (crash scenario)
			if task.Status.State == pb.TaskState_TASK_STATE_WORKING {
				slog.Warn("Task in WORKING state but no checkpoint found, marking as FAILED", "agent", a.id, "task", task.Id)
				if updateErr := a.updateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
					slog.Error("Failed to update task status", "agent", a.id, "task", task.Id, "error", updateErr)
				}
			}
			continue
		}

		// Check if checkpoint is still valid (not expired)
		if a.isCheckpointExpired(execState) {
			slog.Warn("Checkpoint expired for task, marking as FAILED", "agent", a.id, "task", task.Id)
			// A2A Compliance: Transition to FAILED (valid from WORKING/INPUT_REQUIRED)
			if updateErr := a.updateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status", "agent", a.id, "task", task.Id, "error", updateErr)
			}
			continue
		}

		// Resume task based on current state and configuration
		if task.Status.State == pb.TaskState_TASK_STATE_INPUT_REQUIRED {
			// INPUT_REQUIRED tasks: Only auto-resume if configured
			if !a.shouldAutoResumeHITL() {
				slog.Info("Task in INPUT_REQUIRED state, waiting for user input (auto-resume disabled)", "agent", a.id, "task", task.Id)
				continue
			}
		}

		// Resume task from checkpoint
		slog.Info("Recovering task from checkpoint", "agent", a.id, "task", task.Id, "phase", execState.Phase, "type", execState.CheckpointType)

		go a.resumeFromCheckpoint(ctx, execState, "")
		recoveredCount++
	}

	slog.Info("Recovered tasks from checkpoints", "agent", a.id, "count", recoveredCount)
	return nil
}

// isRecoveryEnabled returns true if checkpoint recovery is enabled
func (a *Agent) isRecoveryEnabled() bool {
	if a.config == nil || a.config.Task == nil {
		return false
	}
	checkpointCfg := a.config.Task.Checkpoint
	if checkpointCfg == nil || !checkpointCfg.Enabled {
		return false
	}
	if checkpointCfg.Recovery == nil {
		return false
	}
	return checkpointCfg.Recovery.AutoResume
}

// shouldAutoResumeHITL returns true if INPUT_REQUIRED tasks should be auto-resumed
func (a *Agent) shouldAutoResumeHITL() bool {
	if a.config == nil || a.config.Task == nil {
		return false
	}
	checkpointCfg := a.config.Task.Checkpoint
	if checkpointCfg == nil || checkpointCfg.Recovery == nil {
		return false
	}
	return checkpointCfg.Recovery.AutoResumeHITL
}

// isCheckpointExpired checks if a checkpoint is expired based on recovery timeout
func (a *Agent) isCheckpointExpired(execState *ExecutionState) bool {
	if execState.CheckpointTime.IsZero() {
		return false // No timestamp, assume valid (backward compatibility)
	}

	timeout := a.getRecoveryTimeout()
	if timeout <= 0 {
		return false // No timeout configured, assume valid
	}

	age := time.Since(execState.CheckpointTime)
	return age > time.Duration(timeout)*time.Second
}

// getRecoveryTimeout returns the configured recovery timeout in seconds
func (a *Agent) getRecoveryTimeout() int {
	if a.config == nil || a.config.Task == nil {
		return 0
	}
	checkpointCfg := a.config.Task.Checkpoint
	if checkpointCfg == nil || checkpointCfg.Recovery == nil {
		return 0
	}
	return checkpointCfg.Recovery.ResumeTimeout
}

// resumeFromCheckpoint resumes task execution from a checkpoint
func (a *Agent) resumeFromCheckpoint(
	ctx context.Context,
	execState *ExecutionState,
	userInput string, // Optional: for INPUT_REQUIRED resumes
) {
	// Get current task state (A2A compliance check)
	task, err := a.services.Task().GetTask(ctx, execState.TaskID)
	if err != nil {
		slog.Error("Failed to get task", "agent", a.id, "task", execState.TaskID, "error", err)
		return
	}

	// A2A Compliance: Validate checkpoint state
	if task.Status.State != pb.TaskState_TASK_STATE_WORKING &&
		task.Status.State != pb.TaskState_TASK_STATE_INPUT_REQUIRED {
		slog.Warn("Cannot resume task from terminal state", "agent", a.id, "task", execState.TaskID, "state", task.Status.State)
		return
	}

	// Resume execution using the existing resumeTaskExecution mechanism
	// This reuses the logic from agent_a2a_methods.go
	// For generic recovery, we pass empty user decision - task continues from checkpoint
	slog.Info("Resuming task execution from checkpoint", "agent", a.id, "task", execState.TaskID, "phase", execState.Phase)

	// Use the existing resume mechanism (from agent_a2a_methods.go)
	// Pass empty string for user decision - task will continue from checkpoint
	// resumeTaskExecution handles state restoration, transition to WORKING, and execution
	a.resumeTaskExecution(execState.TaskID, execState, "")
}
