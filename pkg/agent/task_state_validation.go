package agent

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// validateStateTransition validates task state transitions according to A2A protocol rules.
// A2A Protocol Section 6.3 defines valid state transitions:
// - Terminal states (COMPLETED, FAILED, CANCELLED, REJECTED) cannot transition to other states
// - INPUT_REQUIRED can transition back to WORKING when input is provided
// - AUTH_REQUIRED can transition to WORKING after authentication
// - WORKING can transition to any non-terminal state or terminal states
// - SUBMITTED can transition to WORKING, REJECTED, or AUTH_REQUIRED
func validateStateTransition(current pb.TaskState, next pb.TaskState) error {
	// Same state is always valid (idempotent updates)
	if current == next {
		return nil
	}

	// Terminal states cannot transition to other states
	if isTerminalState(current) {
		return fmt.Errorf("cannot transition from terminal state %v to %v: terminal states are immutable", current, next)
	}

	// Validate specific transition rules
	switch current {
	case pb.TaskState_TASK_STATE_SUBMITTED:
		// SUBMITTED can transition to: WORKING, REJECTED, AUTH_REQUIRED
		validNext := []pb.TaskState{
			pb.TaskState_TASK_STATE_WORKING,
			pb.TaskState_TASK_STATE_REJECTED,
			pb.TaskState_TASK_STATE_AUTH_REQUIRED,
		}
		if !containsState(validNext, next) {
			return fmt.Errorf("invalid transition from SUBMITTED to %v: valid transitions are WORKING, REJECTED, AUTH_REQUIRED", next)
		}

	case pb.TaskState_TASK_STATE_WORKING:
		// WORKING can transition to: COMPLETED, FAILED, CANCELLED, INPUT_REQUIRED, REJECTED
		validNext := []pb.TaskState{
			pb.TaskState_TASK_STATE_COMPLETED,
			pb.TaskState_TASK_STATE_FAILED,
			pb.TaskState_TASK_STATE_CANCELLED,
			pb.TaskState_TASK_STATE_INPUT_REQUIRED,
			pb.TaskState_TASK_STATE_REJECTED,
		}
		if !containsState(validNext, next) {
			return fmt.Errorf("invalid transition from WORKING to %v: valid transitions are COMPLETED, FAILED, CANCELLED, INPUT_REQUIRED, REJECTED", next)
		}

	case pb.TaskState_TASK_STATE_INPUT_REQUIRED:
		// INPUT_REQUIRED can transition back to: WORKING (when input provided) or CANCELLED (when cancelled)
		validNext := []pb.TaskState{
			pb.TaskState_TASK_STATE_WORKING,
			pb.TaskState_TASK_STATE_CANCELLED,
			pb.TaskState_TASK_STATE_FAILED, // Can fail if input timeout
		}
		if !containsState(validNext, next) {
			return fmt.Errorf("invalid transition from INPUT_REQUIRED to %v: valid transitions are WORKING, CANCELLED, FAILED", next)
		}

	case pb.TaskState_TASK_STATE_AUTH_REQUIRED:
		// AUTH_REQUIRED can transition to: WORKING (after auth) or CANCELLED
		validNext := []pb.TaskState{
			pb.TaskState_TASK_STATE_WORKING,
			pb.TaskState_TASK_STATE_CANCELLED,
		}
		if !containsState(validNext, next) {
			return fmt.Errorf("invalid transition from AUTH_REQUIRED to %v: valid transitions are WORKING, CANCELLED", next)
		}

	case pb.TaskState_TASK_STATE_UNSPECIFIED:
		// UNSPECIFIED can transition to any state (initial state)
		return nil

	default:
		// Unknown current state - allow transition but log warning
		// This allows for future protocol extensions
		return nil
	}

	return nil
}

func containsState(states []pb.TaskState, state pb.TaskState) bool {
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}
