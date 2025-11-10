package agent

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// updateTaskStatusWithRetry attempts to update task status with validation and exponential backoff retry.
// This ensures status updates succeed even under transient failures while enforcing A2A protocol rules.
// Validation happens at the Agent level (business logic), not in storage implementations.
func (a *Agent) updateTaskStatusWithRetry(ctx context.Context, taskID string, state pb.TaskState, message *pb.Message) error {
	// Use updateTaskStatus which handles validation
	// Then add retry logic on top
	const maxRetries = 3
	const initialBackoff = 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := a.updateTaskStatus(ctx, taskID, state, message)
		if err == nil {
			return nil
		}

		// Don't retry on validation errors
		if attempt == 0 {
			if strings.Contains(err.Error(), "invalid state transition") || strings.Contains(err.Error(), "cannot transition") {
				return err
			}
		}

		if attempt < maxRetries-1 {
			backoff := initialBackoff * time.Duration(1<<uint(attempt))
			log.Printf("[Agent:%s] Task status update failed (attempt %d/%d), retrying in %v: %v", a.id, attempt+1, maxRetries, backoff, err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	return a.updateTaskStatus(ctx, taskID, state, message)
}
