package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/kadirpekel/hector/pkg/a2a/client"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// ApprovalOrchestrator handles the HITL (Human-in-the-Loop) approval flow.
// It centralizes all approval logic to avoid duplication across handlers.
type ApprovalOrchestrator struct {
	client           client.A2AClient
	agentID          string
	handledApprovals map[string]bool
	showThinking     bool
	showTools        bool
}

// NewApprovalOrchestrator creates a new orchestrator for handling approvals.
func NewApprovalOrchestrator(c client.A2AClient, agentID string, showThinking, showTools bool) *ApprovalOrchestrator {
	return &ApprovalOrchestrator{
		client:           c,
		agentID:          agentID,
		handledApprovals: make(map[string]bool),
		showThinking:     showThinking,
		showTools:        showTools,
	}
}

// ApprovalResult represents the outcome of an approval check.
type ApprovalResult struct {
	Handled  bool                    // Whether an approval was handled
	Response *pb.SendMessageResponse // Response from sending approval decision
	Decision string                  // User's decision (approve/deny)
}

// CheckAndHandleApproval checks if a message requires approval and handles the flow.
// Returns ApprovalResult indicating what happened.
func (o *ApprovalOrchestrator) CheckAndHandleApproval(
	ctx context.Context,
	msg *pb.Message,
	taskID, contextID string,
) (*ApprovalResult, error) {
	// Skip if not an approval request
	if !IsApprovalRequest(msg) {
		return &ApprovalResult{Handled: false}, nil
	}

	// Skip if already handled for this task
	if taskID != "" && o.handledApprovals[taskID] {
		return &ApprovalResult{Handled: false}, nil
	}

	// Mark as handled
	if taskID != "" {
		o.handledApprovals[taskID] = true
	}

	// Display and prompt
	o.displayApprovalMessage(msg)
	decision := PromptForApproval()

	// Log decision before sending (audit trail in case of errors)
	slog.Info("User approval decision",
		"decision", decision,
		"task_id", taskID,
		"context_id", contextID)

	// Send approval response
	approvalMsg := CreateApprovalResponse(contextID, taskID, decision)
	resp, err := o.client.SendMessage(ctx, o.agentID, approvalMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to send approval response (decision: %s): %w", decision, err)
	}

	return &ApprovalResult{
		Handled:  true,
		Response: resp,
		Decision: decision,
	}, nil
}

// displayApprovalMessage displays the approval message with proper flushing.
func (o *ApprovalOrchestrator) displayApprovalMessage(msg *pb.Message) {
	DisplayMessageLine(msg, "", o.showThinking, o.showTools)
	o.flush()
	fmt.Println()
	o.flush()
}

// flush ensures output is written to terminal.
func (o *ApprovalOrchestrator) flush() {
	os.Stdout.Sync()
}

// StreamState tracks IDs during stream processing.
type StreamState struct {
	TaskID    string
	ContextID string
}

// UpdateFromStatusUpdate updates state from a status update chunk.
func (s *StreamState) UpdateFromStatusUpdate(update *pb.TaskStatusUpdateEvent) {
	if update.TaskId != "" {
		s.TaskID = update.TaskId
	}
	if update.ContextId != "" {
		s.ContextID = update.ContextId
	}
}

// UpdateFromMessage updates state from a message chunk.
func (s *StreamState) UpdateFromMessage(msg *pb.Message) {
	if msg.TaskId != "" {
		s.TaskID = msg.TaskId
	}
	if msg.ContextId != "" {
		s.ContextID = msg.ContextId
	}
}

// UpdateFromTask updates state from a task chunk.
func (s *StreamState) UpdateFromTask(task *pb.Task) {
	if task.Id != "" {
		s.TaskID = task.Id
	}
	if task.ContextId != "" {
		s.ContextID = task.ContextId
	}
}

// ProcessStreamChunk processes a single stream chunk, handling approval if needed.
// Returns true if stream processing should continue, false if it should stop.
func (o *ApprovalOrchestrator) ProcessStreamChunk(
	ctx context.Context,
	chunk *pb.StreamResponse,
	state *StreamState,
) (continueStream bool, err error) {
	// Handle status updates (primary HITL signal)
	if statusUpdate := chunk.GetStatusUpdate(); statusUpdate != nil {
		state.UpdateFromStatusUpdate(statusUpdate)

		if o.isInputRequired(statusUpdate.Status) {
			if msg := statusUpdate.Status.Update; msg != nil {
				result, err := o.CheckAndHandleApproval(ctx, msg, state.TaskID, state.ContextID)
				if err != nil {
					return false, err
				}
				if result.Handled {
					// Continue streaming - agent will resume
					return true, nil
				}
			}
		}
	}

	// Handle message chunks
	if msgChunk := chunk.GetMsg(); msgChunk != nil {
		state.UpdateFromMessage(msgChunk)

		result, err := o.CheckAndHandleApproval(ctx, msgChunk, state.TaskID, state.ContextID)
		if err != nil {
			return false, err
		}
		if result.Handled {
			return true, nil
		}

		// Regular message - display it
		DisplayMessage(msgChunk, "", o.showThinking, o.showTools)
	}

	// Handle task chunks
	if taskChunk := chunk.GetTask(); taskChunk != nil {
		state.UpdateFromTask(taskChunk)

		if o.isInputRequired(taskChunk.Status) {
			if msg := taskChunk.Status.Update; msg != nil {
				result, err := o.CheckAndHandleApproval(ctx, msg, state.TaskID, state.ContextID)
				if err != nil {
					return false, err
				}
				if result.Handled {
					return true, nil
				}
			}
		}
	}

	return true, nil
}

// isInputRequired checks if a task status indicates input is required.
func (o *ApprovalOrchestrator) isInputRequired(status *pb.TaskStatus) bool {
	return status != nil && status.State == pb.TaskState_TASK_STATE_INPUT_REQUIRED
}

// ProcessNonStreamingResponse handles a non-streaming response with HITL support.
// Uses iteration instead of recursion for the approval loop.
func (o *ApprovalOrchestrator) ProcessNonStreamingResponse(
	ctx context.Context,
	initialMsg *pb.Message,
	sessionID string,
) error {
	const maxIterations = 10
	currentMsg := initialMsg
	contextID := sessionID

	for i := 0; i < maxIterations; i++ {
		resp, err := o.client.SendMessage(ctx, o.agentID, currentMsg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		// Handle message response
		if respMsg := resp.GetMsg(); respMsg != nil {
			DisplayMessageLine(respMsg, "", o.showThinking, o.showTools)
			return nil
		}

		// Handle task response
		task := resp.GetTask()
		if task == nil {
			return nil
		}

		taskID := task.Id
		if task.ContextId != "" {
			contextID = task.ContextId
		}

		// Check if input required
		if !o.isInputRequired(task.Status) {
			// Task completed - display result
			if task.Status != nil && task.Status.Update != nil {
				DisplayMessageLine(task.Status.Update, "", o.showThinking, o.showTools)
			} else {
				DisplayTask(task)
			}
			return nil
		}

		// Handle approval
		msg := task.Status.Update
		if msg == nil {
			DisplayTask(task)
			return nil
		}

		result, err := o.CheckAndHandleApproval(ctx, msg, taskID, contextID)
		if err != nil {
			return err
		}

		if !result.Handled {
			// Not an approval request, just display it
			DisplayMessageLine(msg, "", o.showThinking, o.showTools)
			return nil
		}

		// Prepare next iteration with approval response
		currentMsg = CreateApprovalResponse(contextID, taskID, result.Decision)
	}

	return fmt.Errorf("maximum iterations (%d) exceeded in approval flow", maxIterations)
}
