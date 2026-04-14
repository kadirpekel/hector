// Package server provides A2A protocol server implementation for Hector v2.
//
// The server package implements the a2asrv.AgentExecutor interface to expose
// Hector agents via the A2A protocol (JSON-RPC, gRPC, HTTP).
//
// # Usage
//
//	executor := server.NewExecutor(server.ExecutorConfig{
//	    RunnerConfig: runner.Config{
//	        AppName:        "my-app",
//	        Agent:          myAgent,
//	        SessionService: session.InMemoryService(),
//	    },
//	})
//
//	handler := a2asrv.NewHandler(executor)
//	http.Handle("/a2a", a2asrv.NewJSONRPCHandler(handler))
package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/checkpoint"
	"github.com/verikod/hector/pkg/notification"
	"github.com/verikod/hector/pkg/runner"
	"github.com/verikod/hector/pkg/session"
	"github.com/verikod/hector/pkg/task"
)

// ExecutorConfig contains the configuration for the A2A executor.
type ExecutorConfig struct {
	// RunnerConfig is used to create a runner for each execution.
	RunnerConfig runner.Config

	// RunConfig contains runtime configuration for agent execution.
	RunConfig agent.RunConfig

	// TaskService provides task management for cascade cancellation.
	// If nil, cascade cancellation will not work.
	TaskService task.Service

	// Notifier dispatches outbound notifications on task events.
	// If nil, notifications are disabled.
	Notifier *notification.Notifier
}

// Executor implements a2asrv.AgentExecutor to bridge Hector agents to A2A.
//
// Event translation follows these rules:
//   - New task: emit TaskStatusUpdateEvent with TaskStateSubmitted
//   - Before runner invocation: emit TaskStatusUpdateEvent with TaskStateWorking
//   - For each agent.Event: emit TaskArtifactUpdateEvent with translated parts
//   - After last event: emit TaskArtifactUpdateEvent with LastChunk=true
//   - On LLM error: emit TaskStatusUpdateEvent with TaskStateFailed
//   - On long-running tool: emit TaskStatusUpdateEvent with TaskStateInputRequired
//   - On success: emit TaskStatusUpdateEvent with TaskStateCompleted
type Executor struct {
	config ExecutorConfig
}

// NewExecutor creates a new A2A executor.
func NewExecutor(config ExecutorConfig) *Executor {
	return &Executor{config: config}
}

// Execute implements a2asrv.AgentExecutor.
func (e *Executor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	msg := reqCtx.Message
	if msg == nil {
		slog.Error("Execute: message not provided")
		return fmt.Errorf("message not provided")
	}

	slog.Debug("Execute: converting message", "parts", len(msg.Parts), "role", msg.Role)

	// Check for approval response (HITL)
	// When a user approves/denies a tool, their response comes as a new message
	approval := ExtractApprovalResponse(msg)
	if approval != nil {
		slog.Debug("Execute: processing approval response", "decision", approval.Decision, "toolCallID", approval.ToolCallID)
		// Store approval decision in context metadata for agent to pick up
		// The agent will read this and either execute or skip the pending tool
		if msg.Metadata == nil {
			msg.Metadata = make(map[string]any)
		}
		msg.Metadata["hector:approval_decision"] = approval.Decision
		msg.Metadata["hector:approval_tool_call_id"] = approval.ToolCallID
	}

	// Convert A2A message to Hector content
	content, err := toHectorContent(msg)
	if err != nil {
		slog.Error("Execute: message conversion failed", "error", err)
		return fmt.Errorf("message conversion failed: %w", err)
	}

	slog.Debug("Execute: creating runner")

	// Create runner
	r, err := runner.New(e.config.RunnerConfig)
	if err != nil {
		slog.Error("Execute: failed to create runner", "error", err)
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Emit TaskStateSubmitted for new tasks
	if reqCtx.StoredTask == nil {
		event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateSubmitted, nil)
		if err := queue.Write(ctx, event); err != nil {
			return fmt.Errorf("failed to write submitted event: %w", err)
		}
	}

	// Extract user/session info from request context
	meta := toInvocationMeta(ctx, reqCtx)

	// Prepare session
	if err := e.prepareSession(ctx, meta); err != nil {
		event := toFailedStatusEvent(reqCtx, err, meta.eventMeta)
		if err := queue.Write(ctx, event); err != nil {
			return err
		}
		return nil
	}

	// Store approval decision in session state if present
	if approval != nil {
		if err := e.storeApprovalDecision(ctx, meta, approval); err != nil {
			slog.Warn("Execute: failed to store approval decision", "error", err)
			// Continue anyway - agent may still work without it
		}
	}

	// Emit TaskStateWorking
	workingEvent := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateWorking, nil)
	workingEvent.Metadata = meta.eventMeta
	if err := queue.Write(ctx, workingEvent); err != nil {
		return err
	}

	// Process agent events
	processor := newEventProcessor(reqCtx, meta)
	return e.process(ctx, r, processor, content, queue)
}

// storeApprovalDecision stores the approval decision in session state.
// The agent will read this when resuming execution.
func (e *Executor) storeApprovalDecision(ctx context.Context, meta invocationMeta, approval *ApprovalResponse) error {
	service := e.config.RunnerConfig.SessionService

	// Get current session
	resp, err := service.Get(ctx, &session.GetRequest{
		AppName:   e.config.RunnerConfig.AppName,
		UserID:    meta.userID,
		SessionID: meta.sessionID,
	})
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Update state with approval decision
	// Key format: "_approval:<tool_call_id>" = "approve" | "deny"
	key := "_approval"
	if approval.ToolCallID != "" {
		key = "_approval:" + approval.ToolCallID
	}

	err = service.AppendEvent(ctx, resp.Session, &agent.Event{
		ID:     "approval_" + approval.ToolCallID,
		Author: "user",
		Actions: agent.EventActions{
			StateDelta: map[string]any{
				key: approval.Decision,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to store approval: %w", err)
	}

	return nil
}

// Cancel implements a2asrv.AgentExecutor.
func (e *Executor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, queue eventqueue.Queue) error {
	// Cascade cancellation: cancel all child executions if task service available
	if e.config.TaskService != nil && reqCtx.TaskID != "" {
		t, err := e.config.TaskService.Get(ctx, string(reqCtx.TaskID))
		if err == nil && t != nil {
			cancelled, _ := t.CancelAllChildren()
			slog.Info("Cascade cancellation", "task_id", reqCtx.TaskID, "cancelled", cancelled)
		}
	}

	event := a2a.NewStatusUpdateEvent(reqCtx, a2a.TaskStateCanceled, nil)
	event.Final = true
	return queue.Write(ctx, event)
}

func (e *Executor) process(ctx context.Context, r *runner.Runner, processor *eventProcessor, content *agent.Content, q eventqueue.Queue) error {
	meta := processor.meta
	agentName := e.config.RunnerConfig.Agent.Name()
	taskID := string(processor.reqCtx.TaskID)

	// Create a copy of RunConfig to set task for this invocation
	runConfig := e.config.RunConfig

	// Detect if this is a blocking/non-streaming request.
	// For blocking requests, we disable LLM streaming to avoid artifact pollution.
	// Check multiple signals (in order of specificity):
	// 1. RequestContext.Metadata["hector:source"] == "webhook" (programmatic webhook invocation)
	// 2. CallContext.Method() == "message/send" (HTTP JSON-RPC route)
	if processor.reqCtx.Metadata != nil {
		if source, ok := processor.reqCtx.Metadata["hector:source"].(string); ok && source == "webhook" {
			runConfig.StreamingMode = agent.StreamingModeNone
			slog.Debug("Executor: webhook request detected, disabling LLM streaming")
		}
	}
	if runConfig.StreamingMode != agent.StreamingModeNone {
		if callCtx, ok := a2asrv.CallContextFrom(ctx); ok {
			if callCtx.Method() == "message/send" {
				runConfig.StreamingMode = agent.StreamingModeNone
				slog.Debug("Executor: blocking request detected (message/send), disabling LLM streaming")
			}
		}
	}

	// Get or create task for cascade cancellation support
	// Use the a2a taskID as the key so HTTP cancel endpoint can find it
	if e.config.TaskService != nil {
		if taskID != "" {
			t, err := e.config.TaskService.GetOrCreate(ctx, taskID, meta.sessionID)
			if err != nil {
				slog.Warn("Failed to get/create task for cascade cancellation", "error", err)
			} else {
				runConfig.Task = t
			}
		}
	}

	// Notify task started
	if e.config.Notifier != nil {
		e.config.Notifier.NotifyTaskStarted(agentName, taskID)
	}

	var lastResult string
	for event, err := range r.Run(ctx, meta.userID, meta.sessionID, content, runConfig) {
		if err != nil {
			failedEvent := processor.makeFailedEvent(fmt.Errorf("agent run failed: %w", err), nil)
			if writeErr := q.Write(ctx, failedEvent); writeErr != nil {
				return fmt.Errorf("failed to write error event: %w (original: %w)", writeErr, err)
			}
			// Notify task failed
			if e.config.Notifier != nil {
				e.config.Notifier.NotifyTaskFailed(agentName, taskID, err.Error())
			}
			return nil
		}

		// Capture result text for notification
		if event != nil && event.Artifact != nil {
			for _, part := range event.Artifact.Parts {
				if textPart, ok := part.(a2a.TextPart); ok && textPart.Text != "" {
					lastResult = textPart.Text
				}
			}
		}

		a2aEvent, err := processor.process(ctx, event)
		if err != nil {
			failedEvent := processor.makeFailedEvent(fmt.Errorf("event processing failed: %w", err), event)
			if writeErr := q.Write(ctx, failedEvent); writeErr != nil {
				return fmt.Errorf("failed to write processing error: %w (original: %w)", writeErr, err)
			}
			// Notify task failed
			if e.config.Notifier != nil {
				e.config.Notifier.NotifyTaskFailed(agentName, taskID, err.Error())
			}
			return nil
		}

		if a2aEvent != nil {
			if err := q.Write(ctx, a2aEvent); err != nil {
				return fmt.Errorf("failed to write event: %w", err)
			}
		}
	}

	// Write terminal events
	for _, ev := range processor.makeTerminalEvents() {
		if err := q.Write(ctx, ev); err != nil {
			return fmt.Errorf("failed to write terminal event: %w", err)
		}
	}

	// Notify task completed
	if e.config.Notifier != nil {
		e.config.Notifier.NotifyTaskCompleted(agentName, taskID, lastResult)
	}

	return nil
}

func (e *Executor) prepareSession(ctx context.Context, meta invocationMeta) error {
	service := e.config.RunnerConfig.SessionService

	_, err := service.Get(ctx, &session.GetRequest{
		AppName:   e.config.RunnerConfig.AppName,
		UserID:    meta.userID,
		SessionID: meta.sessionID,
	})
	if err == nil {
		return nil
	}

	_, err = service.Create(ctx, &session.CreateRequest{
		AppName:   e.config.RunnerConfig.AppName,
		UserID:    meta.userID,
		SessionID: meta.sessionID,
		State:     make(map[string]any),
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// ResumeFromCheckpoint resumes agent execution from a checkpoint.
// This is called by the checkpoint recovery manager on startup.
//
// The session already contains the conversation history from before the crash/shutdown.
// We re-invoke the agent with the original query, and it will continue from where it left off.
func (e *Executor) ResumeFromCheckpoint(ctx context.Context, state *checkpoint.State) error {
	// Build user content from the original query
	content := &agent.Content{
		Parts: []a2a.Part{a2a.TextPart{Text: state.Query}},
	}

	// Create run config for recovery (no streaming, no task tracking, skip appending user msg)
	runConfig := e.config.RunConfig
	runConfig.StreamingMode = agent.StreamingModeNone
	runConfig.SkipAppendUserMessage = true

	// Create runner
	r, err := runner.New(e.config.RunnerConfig)
	if err != nil {
		return fmt.Errorf("failed to create runner for recovery: %w", err)
	}

	// Run agent and consume events (fire-and-forget for recovery)
	var lastErr error
	for event, err := range r.Run(ctx, state.UserID, state.SessionID, content, runConfig) {
		if err != nil {
			slog.Warn("Error during checkpoint recovery execution",
				"task_id", state.TaskID,
				"error", err)
			lastErr = err
			continue
		}

		// Log progress for non-partial events
		if event != nil && !event.Partial {
			slog.Debug("Checkpoint recovery event",
				"task_id", state.TaskID,
				"author", event.Author)
		}
	}

	if lastErr != nil {
		slog.Error("Checkpoint recovery completed with errors",
			"task_id", state.TaskID,
			"error", lastErr)
		return lastErr
	}

	// Clear checkpoint after successful recovery
	if cpMgr := e.config.RunnerConfig.CheckpointManager; cpMgr != nil {
		if err := cpMgr.ClearCheckpoint(ctx, state.AppName, state.UserID, state.SessionID, state.TaskID); err != nil {
			slog.Warn("Failed to clear checkpoint after recovery",
				"task_id", state.TaskID,
				"error", err)
		}
	}

	slog.Info("Checkpoint recovery completed successfully",
		"task_id", state.TaskID,
		"session_id", state.SessionID)

	return nil
}

// Ensure Executor implements a2asrv.AgentExecutor
var _ a2asrv.AgentExecutor = (*Executor)(nil)
