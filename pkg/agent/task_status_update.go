package agent

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// updateTaskStatus validates and updates task status.
// This is the primary method for updating task status - it ensures validation happens
// at the Agent level (business logic) before delegating to storage implementations.
// Use this instead of calling services.Task().UpdateTaskStatus() directly.
func (a *Agent) updateTaskStatus(ctx context.Context, taskID string, state pb.TaskState, message *pb.Message) error {
	// Get current task state for validation (business logic layer)
	task, err := a.services.Task().GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task for validation: %w", err)
	}

	// Validate state transition at Agent level (business logic, not storage)
	if err := validateStateTransition(task.Status.State, state); err != nil {
		return fmt.Errorf("invalid state transition for task %s: %w", taskID, err)
	}

	// Delegate to storage implementation (no validation there - it's just persistence)
	return a.services.Task().UpdateTaskStatus(ctx, taskID, state, message)
}

