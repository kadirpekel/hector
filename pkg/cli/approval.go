package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/client"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"golang.org/x/term"
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

	// Show session and task info for clarity (helpful for understanding resumption)
	if contextID != "" || taskID != "" {
		fmt.Printf("\n%s[INFO]%s Session: %s | Task: %s\n", colorDim, colorReset, contextID, taskID)
		fmt.Printf("%s[INFO]%s Resumption will happen automatically after approval\n\n", colorDim, colorReset)
	}

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

// isTerminal checks if the file descriptor is a terminal
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// pollTaskUntilComplete polls GetTask until the task completes or fails.
// This is needed for async task resumption where the task runs in a background goroutine.
func (o *ApprovalOrchestrator) pollTaskUntilComplete(ctx context.Context, taskID, contextID string) (*pb.Task, error) {
	const maxPollAttempts = 300 // 5 minutes max (1 second intervals)
	const pollInterval = 1 * time.Second

	for attempt := 0; attempt < maxPollAttempts; attempt++ {
		task, err := o.client.GetTask(ctx, o.agentID, taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task: %w", err)
		}

		if task.Status == nil {
			time.Sleep(pollInterval)
			continue
		}

		state := task.Status.State
		if state == pb.TaskState_TASK_STATE_COMPLETED || state == pb.TaskState_TASK_STATE_FAILED || state == pb.TaskState_TASK_STATE_CANCELLED {
			return task, nil
		}

		// Still working or waiting for input
		if state == pb.TaskState_TASK_STATE_INPUT_REQUIRED {
			// Another approval needed - return task so caller can handle it
			return task, nil
		}

		time.Sleep(pollInterval)
	}

	return nil, fmt.Errorf("task did not complete within %d seconds", maxPollAttempts)
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

		// Check if stdin is a TTY (interactive terminal)
		// If not interactive, return immediately and let user approve in separate call
		// This supports async workflow: create task → approve later
		if !isTerminal(os.Stdin) {
			// Non-interactive: return task info and exit
			// User can approve in separate call: ./hector call "approve" --session <session>
			fmt.Printf("\n%s[INFO]%s Task created and waiting for approval\n", colorDim, colorReset)
			fmt.Printf("%s[INFO]%s Task ID: %s\n", colorDim, colorReset, taskID)
			fmt.Printf("%s[INFO]%s Session: %s\n", colorDim, colorReset, contextID)
			fmt.Printf("%s[INFO]%s To approve, run: ./hector call --no-stream \"approve\" --session %s\n\n", colorDim, colorReset, contextID)
			DisplayTask(task)
			return nil
		}

		// Interactive terminal: prompt for approval
		result, err := o.CheckAndHandleApproval(ctx, msg, taskID, contextID)
		if err != nil {
			return err
		}

		if !result.Handled {
			// Not an approval request, just display it
			DisplayMessageLine(msg, "", o.showThinking, o.showTools)
			return nil
		}

		// Approval was already sent by CheckAndHandleApproval
		// Process the response from approval (same as regular SendMessage response)
		if result.Response == nil {
			return fmt.Errorf("approval sent but no response received")
		}

		// Process approval response the same way as SendMessage response
		// Handle message response
		if respMsg := result.Response.GetMsg(); respMsg != nil {
			DisplayMessageLine(respMsg, "", o.showThinking, o.showTools)
			return nil
		}

		// Handle task response from approval
		approvalTask := result.Response.GetTask()
		if approvalTask == nil {
			return nil
		}

		// Update context ID if provided
		if approvalTask.ContextId != "" {
			contextID = approvalTask.ContextId
		}

		// Check task status and handle accordingly
		if approvalTask.Status == nil {
			DisplayTask(approvalTask)
			return nil
		}

		state := approvalTask.Status.State

		// Task resumed asynchronously - poll for completion
		// After approval, task resumes in background goroutine, so we need to poll GetTask
		// until it completes (per A2A spec Section 6.3)
		// Note: The client (and its database connections) must stay open during polling
		// to allow the background goroutine to access the database
		// IMPORTANT: Check WORKING state BEFORE checking isInputRequired, because
		// WORKING is not INPUT_REQUIRED, so we'd return early and skip polling
		if state == pb.TaskState_TASK_STATE_WORKING {
			slog.Info("Task resumed asynchronously, polling for completion", "task", approvalTask.Id)
			// Use a longer timeout context to ensure we can poll long enough
			// The database connection will stay open as long as the client is not closed
			pollCtx, pollCancel := context.WithTimeout(ctx, 5*time.Minute)
			defer pollCancel()
			completedTask, err := o.pollTaskUntilComplete(pollCtx, approvalTask.Id, contextID)
			if err != nil {
				return fmt.Errorf("failed to poll task completion: %w", err)
			}
			// Display final result
			if completedTask.Status != nil && completedTask.Status.Update != nil {
				DisplayMessageLine(completedTask.Status.Update, "", o.showThinking, o.showTools)
			} else {
				DisplayTask(completedTask)
			}
			return nil
		}

		// Check if task completed (COMPLETED, FAILED, CANCELLED)
		if !o.isInputRequired(approvalTask.Status) {
			// Task completed - display result
			if approvalTask.Status != nil && approvalTask.Status.Update != nil {
				DisplayMessageLine(approvalTask.Status.Update, "", o.showThinking, o.showTools)
			} else {
				DisplayTask(approvalTask)
			}
			return nil
		}

		// Task still requires input (another approval?), continue loop
		// Send empty message to get next update
		currentMsg = &pb.Message{
			ContextId: contextID,
			TaskId:    approvalTask.Id,
			Role:      pb.Role_ROLE_USER,
			Parts:     []*pb.Part{{Part: &pb.Part_Text{Text: ""}}},
		}
	}

	return fmt.Errorf("maximum iterations (%d) exceeded in approval flow", maxIterations)
}
