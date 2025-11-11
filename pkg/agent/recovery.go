package agent

import (
	"context"
	"log"
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
		log.Printf("[Agent:%s] Checkpoint recovery disabled, skipping", a.id)
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
			log.Printf("[Agent:%s] Failed to list tasks in state %v: %v", a.id, state, err)
			continue
		}
		allPendingTasks = append(allPendingTasks, tasks...)
	}

	if len(allPendingTasks) == 0 {
		log.Printf("[Agent:%s] No pending tasks to recover", a.id)
		return nil
	}

	log.Printf("[Agent:%s] Found %d pending tasks, checking for checkpoints...", a.id, len(allPendingTasks))

	recoveredCount := 0
	for _, task := range allPendingTasks {
		// A2A Compliance: Validate state is still recoverable
		if isTerminalState(task.Status.State) {
			log.Printf("[Agent:%s] Task %s in terminal state, skipping recovery", a.id, task.Id)
			continue
		}

		// Check if task has a checkpoint
		sessionID := task.ContextId
		if sessionID == "" {
			log.Printf("[Agent:%s] Task %s has no context ID, skipping recovery", a.id, task.Id)
			continue
		}

		execState, err := a.LoadExecutionStateFromSession(ctx, sessionID, task.Id)
		if err != nil {
			// No checkpoint found - this is OK (task might not have checkpointing enabled)
			// If task is WORKING but has no checkpoint, mark as FAILED (crash scenario)
			if task.Status.State == pb.TaskState_TASK_STATE_WORKING {
				log.Printf("[Agent:%s] Task %s in WORKING state but no checkpoint found, marking as FAILED", a.id, task.Id)
				if updateErr := a.updateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
					log.Printf("[Agent:%s] Failed to update task %s status: %v", a.id, task.Id, updateErr)
				}
			}
			continue
		}

		// Check if checkpoint is still valid (not expired)
		if a.isCheckpointExpired(execState) {
			log.Printf("[Agent:%s] Checkpoint expired for task %s, marking as FAILED", a.id, task.Id)
			// A2A Compliance: Transition to FAILED (valid from WORKING/INPUT_REQUIRED)
			if updateErr := a.updateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				log.Printf("[Agent:%s] Failed to update task %s status: %v", a.id, task.Id, updateErr)
			}
			continue
		}

		// Resume task based on current state and configuration
		if task.Status.State == pb.TaskState_TASK_STATE_INPUT_REQUIRED {
			// INPUT_REQUIRED tasks: Only auto-resume if configured
			if !a.shouldAutoResumeHITL() {
				log.Printf("[Agent:%s] Task %s in INPUT_REQUIRED state, waiting for user input (auto-resume disabled)", a.id, task.Id)
				continue
			}
		}

		// Resume task from checkpoint
		log.Printf("[Agent:%s] Recovering task %s from checkpoint (phase: %s, type: %s)",
			a.id, task.Id, execState.Phase, execState.CheckpointType)

		go a.resumeFromCheckpoint(ctx, execState, "")
		recoveredCount++
	}

	log.Printf("[Agent:%s] Recovered %d tasks from checkpoints", a.id, recoveredCount)
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
		log.Printf("[Agent:%s] Failed to get task %s: %v", a.id, execState.TaskID, err)
		return
	}

	// A2A Compliance: Validate checkpoint state
	if task.Status.State != pb.TaskState_TASK_STATE_WORKING &&
		task.Status.State != pb.TaskState_TASK_STATE_INPUT_REQUIRED {
		log.Printf("[Agent:%s] Cannot resume task %s from terminal state: %v", a.id, execState.TaskID, task.Status.State)
		return
	}

	// Resume execution using the existing resumeTaskExecution mechanism
	// This reuses the logic from agent_a2a_methods.go
	// For generic recovery, we pass empty user decision - task continues from checkpoint
	log.Printf("[Agent:%s] Resuming task %s execution from checkpoint (phase: %s)",
		a.id, execState.TaskID, execState.Phase)

	// Use the existing resume mechanism (from agent_a2a_methods.go)
	// Pass empty string for user decision - task will continue from checkpoint
	// resumeTaskExecution handles state restoration, transition to WORKING, and execution
	a.resumeTaskExecution(execState.TaskID, execState, "")
}
