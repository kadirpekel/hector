package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// SessionIDKey is re-exported from protocol package for backward compatibility
const SessionIDKey = protocol.SessionIDKey

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	taskIDContextKey       contextKey = "taskID"
	userDecisionContextKey contextKey = "userDecision"
)

// handleInputRequiredResume handles resuming a task that's in INPUT_REQUIRED state.
// Returns (handled, response, error) where handled indicates if this was an INPUT_REQUIRED resume.
func (a *Agent) handleInputRequiredResume(ctx context.Context, userMessage *pb.Message) (bool, *pb.SendMessageResponse, error) {
	if userMessage.TaskId == "" || a.services.Task() == nil {
		return false, nil, nil
	}

	existingTask, err := a.services.Task().GetTask(ctx, userMessage.TaskId)
	if err != nil {
		// Task not found or error - not an INPUT_REQUIRED resume
		return false, nil, nil
	}

	if existingTask.Status.State != pb.TaskState_TASK_STATE_INPUT_REQUIRED {
		return false, nil, nil
	}

	// Validate context ID matches (security: ensure user owns the task)
	if userMessage.ContextId != "" && existingTask.ContextId != "" && userMessage.ContextId != existingTask.ContextId {
		return true, nil, status.Errorf(codes.InvalidArgument, "context ID mismatch: cannot resume task with different context")
	}

	// Check if this is async HITL (has execution state) or blocking HITL (has waiting channel)
	sessionID := existingTask.ContextId
	if a.shouldUseAsyncHITL() {
		// Try to load execution state - if found, use async resume
		execState, err := a.LoadExecutionStateFromSession(ctx, sessionID, userMessage.TaskId)
		if err == nil {
			// Async HITL: Load state and resume execution
			decision := parseUserDecision(userMessage)
			slog.Info("Resuming task from async execution state", "agent", a.id, "task", userMessage.TaskId, "decision", decision)
			go a.resumeTaskExecution(userMessage.TaskId, execState, decision)

			return true, &pb.SendMessageResponse{
				Payload: &pb.SendMessageResponse_Task{
					Task: existingTask,
				},
			}, nil
		}
		// If no execution state found, check if blocking mode is available
		slog.Debug("No execution state found for task, checking blocking mode", "agent", a.id, "task", userMessage.TaskId, "session", sessionID, "error", err)
		if a.taskAwaiter.IsWaiting(userMessage.TaskId) {
			// Blocking HITL: Provide input to waiting goroutine
			if err := a.taskAwaiter.ProvideInput(userMessage.TaskId, userMessage); err != nil {
				return true, nil, status.Errorf(codes.InvalidArgument, "failed to resume task: %v", err)
			}
			return true, &pb.SendMessageResponse{
				Payload: &pb.SendMessageResponse_Task{
					Task: existingTask,
				},
			}, nil
		}
		// Neither async nor blocking mode available - this is an error
		return true, nil, status.Errorf(codes.InvalidArgument, "failed to resume task: task %s is in INPUT_REQUIRED state but no execution state found and task is not waiting for input. This may indicate the execution state was lost or the task was already resumed.", userMessage.TaskId)
	}

	// Blocking HITL: Provide input to waiting goroutine
	if err := a.taskAwaiter.ProvideInput(userMessage.TaskId, userMessage); err != nil {
		return true, nil, status.Errorf(codes.InvalidArgument, "failed to resume task: %v", err)
	}

	// Task will resume execution in background goroutine
	// Return current task state (will be updated as execution continues)
	return true, &pb.SendMessageResponse{
		Payload: &pb.SendMessageResponse_Task{
			Task: existingTask,
		},
	}, nil
}

func (a *Agent) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	if req.Request == nil {
		return nil, status.Error(codes.InvalidArgument, "request message cannot be nil")
	}

	userMessage := req.Request

	userText := protocol.ExtractTextFromMessage(userMessage)
	if userText == "" {
		return nil, status.Error(codes.InvalidArgument, "message text cannot be empty")
	}

	contextID := userMessage.ContextId
	if contextID == "" {
		contextID = generateContextID()
		userMessage.ContextId = contextID
	}

	if userMessage.MessageId == "" {
		userMessage.MessageId = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	}

	// A2A Protocol Section 6.3: Check if this is a continuation of an existing task
	// (multi-turn conversation for INPUT_REQUIRED state)
	if handled, resp, err := a.handleInputRequiredResume(ctx, userMessage); handled {
		if err != nil {
			return nil, err
		}
		return resp, nil
	}

	if a.services.Task() != nil {
		blocking := true
		if req.Configuration != nil {
			blocking = req.Configuration.Blocking
		}

		task, err := a.services.Task().CreateTask(ctx, contextID, userMessage)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create task: %v", err)
		}

		if !blocking {
			go a.processTaskAsync(task.Id, userText, contextID)

			return &pb.SendMessageResponse{
				Payload: &pb.SendMessageResponse_Task{
					Task: task,
				},
			}, nil
		}

		if err := a.updateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_WORKING, nil); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update task status: %v", err)
		}

		// Add taskID to context for HITL support (tool approval)
		// Use EnsureAgentContext to maintain consistency
		ctx = EnsureAgentContext(ctx, task.Id, contextID)

		// Non-streaming mode: don't block for HITL, return task in INPUT_REQUIRED state instead
		opts := ExecutionOptions{BlockForHITL: false}
		responseText, err := a.executeReasoningForA2A(ctx, userText, contextID, opts)
		if err != nil {
			// Check if this is an INPUT_REQUIRED signal (HITL needs user input)
			if err == ErrInputRequired {
				// Task is in INPUT_REQUIRED state, return it so client can prompt for approval
				task, getErr := a.services.Task().GetTask(ctx, task.Id)
				if getErr != nil {
					return nil, status.Errorf(codes.Internal, "failed to get task: %v", getErr)
				}
				slog.Info("Returning task in INPUT_REQUIRED state for client-side approval", "agent", a.id, "task", task.Id)
				return &pb.SendMessageResponse{
					Payload: &pb.SendMessageResponse_Task{
						Task: task,
					},
				}, nil
			}
			if updateErr := a.updateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status to FAILED", "agent", a.id, "task", task.Id, "error", updateErr)
			}
			return nil, status.Errorf(codes.Internal, "agent execution failed: %v", err)
		}

		responseMessage := a.createResponseMessage(responseText, contextID, task.Id)

		if err := a.services.Task().AddTaskMessage(ctx, task.Id, responseMessage); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to add response message: %v", err)
		}

		if err := a.updateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_COMPLETED, responseMessage); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update task status: %v", err)
		}

		task, err = a.services.Task().GetTask(ctx, task.Id)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get task: %v", err)
		}

		return &pb.SendMessageResponse{
			Payload: &pb.SendMessageResponse_Task{
				Task: task,
			},
		}, nil
	}

	// No task service: use default execution (blocks for HITL if needed)
	responseText, err := a.executeReasoningForA2A(ctx, userText, contextID, DefaultExecutionOptions())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "agent execution failed: %v", err)
	}

	responseMessage := &pb.Message{
		MessageId: fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		ContextId: contextID,
		Role:      pb.Role_ROLE_AGENT,
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: responseText,
				},
			},
		},
	}

	return &pb.SendMessageResponse{
		Payload: &pb.SendMessageResponse_Msg{
			Msg: responseMessage,
		},
	}, nil
}

func (a *Agent) SendStreamingMessage(req *pb.SendMessageRequest, stream pb.A2AService_SendStreamingMessageServer) error {
	if req.Request == nil {
		return status.Error(codes.InvalidArgument, "request message cannot be nil")
	}

	userMessage := req.Request
	userText := protocol.ExtractTextFromMessage(userMessage)
	if userText == "" {
		return status.Error(codes.InvalidArgument, "message text cannot be empty")
	}

	contextID := userMessage.ContextId
	if contextID == "" {
		contextID = generateContextID()
		userMessage.ContextId = contextID
	}

	ctx := stream.Context()

	// A2A Protocol Section 6.3: Check if this is a continuation of an existing task
	// (multi-turn conversation for INPUT_REQUIRED state)
	if handled, _, err := a.handleInputRequiredResume(ctx, userMessage); handled {
		if err != nil {
			return err
		}
		// Task will resume execution in background goroutine
		// Just return success - the task is already running
		return nil
	}

	if a.services.Task() != nil {
		task, err := a.services.Task().CreateTask(ctx, contextID, userMessage)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to create task: %v", err)
		}

		if err := stream.Send(&pb.StreamResponse{
			Payload: &pb.StreamResponse_Task{
				Task: task,
			},
		}); err != nil {
			return status.Errorf(codes.Internal, "failed to send task: %v", err)
		}

		if err := a.updateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_WORKING, nil); err != nil {
			return status.Errorf(codes.Internal, "failed to update task status: %v", err)
		}

		updatedTask, _ := a.services.Task().GetTask(ctx, task.Id)
		if err := stream.Send(&pb.StreamResponse{
			Payload: &pb.StreamResponse_StatusUpdate{
				StatusUpdate: &pb.TaskStatusUpdateEvent{
					TaskId:    task.Id,
					ContextId: contextID,
					Status:    updatedTask.Status,
					Final:     false,
				},
			},
		}); err != nil {
			return status.Errorf(codes.Internal, "failed to send status update: %v", err)
		}

		// Get reasoning config from services (works for both config-based and programmatic agents)
		reasoningCfg := a.services.GetConfig()
		strategy, err := reasoning.CreateStrategy(reasoningCfg.Engine, reasoningCfg)
		if err != nil {
			if updateErr := a.updateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status to FAILED", "agent", a.id, "task", task.Id, "error", updateErr)
			}
			return status.Errorf(codes.Internal, "failed to create strategy: %v", err)
		}

		// Create cancellable context for this execution
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Register cancellation function for streaming tasks
		a.executionsMu.Lock()
		a.activeExecutions[task.Id] = cancel
		a.executionsMu.Unlock()

		defer func() {
			a.executionsMu.Lock()
			delete(a.activeExecutions, task.Id)
			a.executionsMu.Unlock()
		}()

		// Add taskID to context for tool approval
		ctx = EnsureAgentContext(ctx, task.Id, contextID)

		// Pass the full userMessage to preserve file parts (images, etc.)
		streamCh, err := a.executeWithMessage(ctx, userText, userMessage, strategy)
		if err != nil {
			if updateErr := a.updateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status to FAILED", "agent", a.id, "task", task.Id, "error", updateErr)
			}
			return status.Errorf(codes.Internal, "reasoning failed: %v", err)
		}

		messageID := fmt.Sprintf("msg-%d", time.Now().UnixNano())
		var fullResponse strings.Builder

		for part := range streamCh {
			// Extract text for building full response (for task storage)
			if textPart, ok := part.Part.(*pb.Part_Text); ok {
				fullResponse.WriteString(textPart.Text)
			}

			// Send each part directly (supports text, tool_call, tool_result)
			chunkMsg := &pb.Message{
				MessageId: messageID,
				ContextId: contextID,
				TaskId:    task.Id,
				Role:      pb.Role_ROLE_AGENT,
				Parts:     []*pb.Part{part},
			}

			if err := stream.Send(&pb.StreamResponse{
				Payload: &pb.StreamResponse_Msg{Msg: chunkMsg},
			}); err != nil {
				// Stream send failed - log error but don't try to update task status
				// as the stream is already broken
				slog.Error("Failed to send chunk to stream", "agent", a.id, "task", task.Id, "error", err)
				return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
			}
		}

		responseMessage := a.createResponseMessage(fullResponse.String(), contextID, task.Id)
		responseMessage.MessageId = messageID

		if err := a.services.Task().AddTaskMessage(ctx, task.Id, responseMessage); err != nil {
			return status.Errorf(codes.Internal, "failed to add response message: %v", err)
		}

		if err := a.updateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_COMPLETED, responseMessage); err != nil {
			return status.Errorf(codes.Internal, "failed to update task status: %v", err)
		}

		finalTask, _ := a.services.Task().GetTask(ctx, task.Id)
		if err := stream.Send(&pb.StreamResponse{
			Payload: &pb.StreamResponse_StatusUpdate{
				StatusUpdate: &pb.TaskStatusUpdateEvent{
					TaskId:    task.Id,
					ContextId: contextID,
					Status:    finalTask.Status,
					Final:     true,
				},
			},
		}); err != nil {
			return status.Errorf(codes.Internal, "failed to send final status: %v", err)
		}

		return nil
	}

	// Get reasoning config from services (works for both config-based and programmatic agents)
	reasoningCfg := a.services.GetConfig()
	strategy, err := reasoning.CreateStrategy(reasoningCfg.Engine, reasoningCfg)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create strategy: %v", err)
	}

	// Pass the full userMessage to preserve file parts (images, etc.)
	streamCh, err := a.executeWithMessage(ctx, userText, userMessage, strategy)
	if err != nil {
		return status.Errorf(codes.Internal, "reasoning failed: %v", err)
	}

	messageID := fmt.Sprintf("msg-%d", time.Now().UnixNano())

	for part := range streamCh {
		// Send each part directly (supports text, tool_call, tool_result)
		chunkMsg := &pb.Message{
			MessageId: messageID,
			ContextId: contextID,
			Role:      pb.Role_ROLE_AGENT,
			Parts:     []*pb.Part{part},
		}

		if err := stream.Send(&pb.StreamResponse{
			Payload: &pb.StreamResponse_Msg{Msg: chunkMsg},
		}); err != nil {
			return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
		}
	}

	return nil
}

func (a *Agent) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error) {
	// Build the agent URL - all transports now use agent-scoped endpoints
	// The URL points to the agent's base path: /v1/agents/{agent}
	// This works for REST, gRPC, and JSON-RPC (all agent-scoped)
	agentURL := fmt.Sprintf("%s/v1/agents/%s", a.baseURL, a.id)

	card := &pb.AgentCard{
		Name:               a.name,
		Description:        a.description,
		Version:            a.getVersion(),
		ProtocolVersion:    "0.3.0",
		Url:                agentURL,
		PreferredTransport: a.preferredTransport, // Use configured transport
		Capabilities: &pb.AgentCapabilities{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: false, // Not yet implemented
			Extensions: []*pb.AgentExtension{
				{
					Uri:         "https://ag-ui.org/protocol/v1",
					Description: "AG-UI (Agent User Interaction) Protocol - Standardized streaming event format for agent UIs. Clients can opt-in via Accept header 'application/x-agui-events' or query parameter 'format=agui'.",
					Required:    false,
				},
			},
		},

		DefaultInputModes:  a.getInputModes(),
		DefaultOutputModes: a.getOutputModes(),
		Skills:             a.getSkills(),
		Provider:           a.getProvider(),
		DocumentationUrl:   a.getDocumentationURL(),
	}

	// Add per-agent security configuration if present
	if a.config != nil && a.config.Security != nil && a.config.Security.IsEnabled() && len(a.config.Security.Schemes) > 0 {
		card.SecuritySchemes = make(map[string]*pb.SecurityScheme)
		for name, scheme := range a.config.Security.Schemes {
			pbScheme := convertConfigSecurityScheme(scheme)
			if pbScheme != nil {
				card.SecuritySchemes[name] = pbScheme
			}
		}

		if len(a.config.Security.Require) > 0 {
			card.Security = make([]*pb.Security, 0, len(a.config.Security.Require))
			for _, reqSet := range a.config.Security.Require {
				pbSec := &pb.Security{
					Schemes: make(map[string]*pb.StringList),
				}
				for schemeName, scopes := range reqSet {
					pbSec.Schemes[schemeName] = &pb.StringList{List: scopes}
				}
				card.Security = append(card.Security, pbSec)
			}
		}
	} else if a.componentManager != nil {
		// Add global auth configuration if no per-agent security configured
		// This follows A2A spec Section 5.5 for declaring authentication requirements
		globalConfig := a.componentManager.GetGlobalConfig()
		if globalConfig != nil && globalConfig.Global.Auth.IsEnabled() {
			card.SecuritySchemes = make(map[string]*pb.SecurityScheme)
			card.SecuritySchemes["BearerAuth"] = &pb.SecurityScheme{
				Scheme: &pb.SecurityScheme_HttpAuthSecurityScheme{
					HttpAuthSecurityScheme: &pb.HTTPAuthSecurityScheme{
						Description:  "JWT Bearer token authentication",
						Scheme:       "bearer",
						BearerFormat: "JWT",
					},
				},
			}

			// Require the BearerAuth scheme
			card.Security = []*pb.Security{
				{
					Schemes: map[string]*pb.StringList{
						"BearerAuth": {List: []string{}}, // No specific scopes required
					},
				},
			}
		}
	}

	return card, nil
}

func convertConfigSecurityScheme(scheme *config.SecurityScheme) *pb.SecurityScheme {
	if scheme == nil {
		return nil
	}

	switch scheme.Type {
	case "http":
		return &pb.SecurityScheme{
			Scheme: &pb.SecurityScheme_HttpAuthSecurityScheme{
				HttpAuthSecurityScheme: &pb.HTTPAuthSecurityScheme{
					Description:  scheme.Description,
					Scheme:       scheme.Scheme,
					BearerFormat: scheme.BearerFormat,
				},
			},
		}
	case "apiKey":
		return &pb.SecurityScheme{
			Scheme: &pb.SecurityScheme_ApiKeySecurityScheme{
				ApiKeySecurityScheme: &pb.APIKeySecurityScheme{
					Description: scheme.Description,
					Location:    scheme.In,
					Name:        scheme.Name,
				},
			},
		}

	default:
		return nil
	}
}

func (a *Agent) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	if a.services.Task() == nil {
		return nil, status.Error(codes.Unimplemented, "task tracking not enabled")
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "task name is required")
	}

	taskID := extractTaskID(req.Name)
	if taskID == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid task name format")
	}

	task, err := a.services.Task().GetTask(ctx, taskID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "task not found: %v", err)
	}

	if req.HistoryLength > 0 && len(task.History) > int(req.HistoryLength) {

		taskCopy := &pb.Task{
			Id:        task.Id,
			ContextId: task.ContextId,
			Status:    task.Status,
			Artifacts: task.Artifacts,
			Metadata:  task.Metadata,
		}
		start := len(task.History) - int(req.HistoryLength)
		taskCopy.History = task.History[start:]
		return taskCopy, nil
	}

	return task, nil
}

func (a *Agent) createResponseMessage(responseText, contextID, taskID string) *pb.Message {
	return &pb.Message{
		MessageId: fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		ContextId: contextID,
		TaskId:    taskID,
		Role:      pb.Role_ROLE_AGENT,
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: responseText,
				},
			},
		},
	}
}

func (a *Agent) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	if a.services.Task() == nil {
		return nil, status.Error(codes.Unimplemented, "task tracking not enabled")
	}

	tasks, nextPageToken, totalSize, err := a.services.Task().ListTasks(
		ctx,
		req.ContextId,
		req.Status,
		req.PageSize,
		req.PageToken,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tasks: %v", err)
	}

	return &pb.ListTasksResponse{
		Tasks:         tasks,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}, nil
}

func (a *Agent) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error) {
	if a.services.Task() == nil {
		return nil, status.Error(codes.Unimplemented, "task tracking not enabled")
	}

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "task name is required")
	}

	taskID := extractTaskID(req.Name)
	if taskID == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid task name format")
	}

	// Cancel active execution context if exists
	// Lock properly to prevent race condition: get cancel function and remove from map atomically
	// We need to ensure atomic cancellation: cancel the context AND update task status together
	var cancelFunc context.CancelFunc
	var shouldCancel bool

	a.executionsMu.Lock()
	if cancel, exists := a.activeExecutions[taskID]; exists {
		cancelFunc = cancel
		shouldCancel = true
		// Don't delete yet - we'll delete after ensuring cancellation is complete
	}
	a.executionsMu.Unlock()

	// Cancel any waiting input requests first (before cancelling execution)
	// This ensures HITL tasks are properly cancelled
	a.taskAwaiter.CancelWaiting(taskID)

	// Cancel the execution context if it exists
	// This must happen before CancelTask to ensure execution stops before status update
	if shouldCancel && cancelFunc != nil {
		cancelFunc() // Cancel the context to stop execution
	}

	// Validate cancellation is allowed (get current state first)
	currentTask, err := a.services.Task().GetTask(ctx, taskID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get task: %v", err)
	}

	// Validate transition to CANCELLED (business logic validation)
	if err := validateStateTransition(currentTask.Status.State, pb.TaskState_TASK_STATE_CANCELLED); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot cancel task: %v", err)
	}

	// Now update task status - this is safe because:
	// 1. Execution context is cancelled (goroutine will exit)
	// 2. Waiting input is cancelled (HITL is resolved)
	// 3. State transition is validated
	task, err := a.services.Task().CancelTask(ctx, taskID)

	// Clean up execution tracking after cancellation is complete
	if shouldCancel {
		a.executionsMu.Lock()
		delete(a.activeExecutions, taskID)
		a.executionsMu.Unlock()
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to cancel task: %v", err)
	}

	return task, nil
}

func (a *Agent) TaskSubscription(req *pb.TaskSubscriptionRequest, stream pb.A2AService_TaskSubscriptionServer) error {
	if a.services.Task() == nil {
		return status.Error(codes.Unimplemented, "task tracking not enabled")
	}

	if req.Name == "" {
		return status.Error(codes.InvalidArgument, "task name is required")
	}

	taskID := extractTaskID(req.Name)
	if taskID == "" {
		return status.Error(codes.InvalidArgument, "invalid task name format")
	}

	eventCh, err := a.services.Task().SubscribeToTask(stream.Context(), taskID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to subscribe to task: %v", err)
	}

	for event := range eventCh {
		if err := stream.Send(event); err != nil {
			return status.Errorf(codes.Internal, "failed to send event: %v", err)
		}
	}

	return nil
}

func (a *Agent) CreateTaskPushNotificationConfig(ctx context.Context, req *pb.CreateTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not yet implemented")
}

func (a *Agent) GetTaskPushNotificationConfig(ctx context.Context, req *pb.GetTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not yet implemented")
}

func (a *Agent) ListTaskPushNotificationConfig(ctx context.Context, req *pb.ListTaskPushNotificationConfigRequest) (*pb.ListTaskPushNotificationConfigResponse, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not yet implemented")
}

func (a *Agent) DeleteTaskPushNotificationConfig(ctx context.Context, req *pb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not yet implemented")
}

func (a *Agent) processTaskAsync(taskID, userText, contextID string) {
	// Add panic recovery to ensure task status is always updated
	defer func() {
		if r := recover(); r != nil {
			slog.Error("PANIC in async task", "agent", a.id, "task", taskID, "panic", r)
			// Try to update task status to FAILED
			ctx := context.Background()
			if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status after panic", "agent", a.id, "task", taskID, "error", updateErr)
			}
		}
	}()

	if a.taskWorkers != nil {
		a.taskWorkers <- struct{}{}
		defer func() { <-a.taskWorkers }()
	}

	// Create cancellable context with timeout for async task execution
	// Use context with timeout to prevent indefinite execution
	// Timeout is configurable per agent via TaskConfig.Timeout (default: 1 hour)
	timeout := 1 * time.Hour // Default
	if a.config.Task != nil && a.config.Task.Timeout > 0 {
		timeout = time.Duration(a.config.Task.Timeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Register cancellation function for this task BEFORE setting context values
	a.executionsMu.Lock()
	a.activeExecutions[taskID] = cancel
	a.executionsMu.Unlock()

	defer func() {
		a.executionsMu.Lock()
		delete(a.activeExecutions, taskID)
		a.executionsMu.Unlock()
	}()

	// Add taskID to context for tool approval logic
	ctx = EnsureAgentContext(ctx, taskID, contextID)

	// Retry status update with exponential backoff
	if err := a.updateTaskStatusWithRetry(ctx, taskID, pb.TaskState_TASK_STATE_WORKING, nil); err != nil {
		slog.Error("Failed to update task status to WORKING after retries", "agent", a.id, "task", taskID, "error", err)
		// Don't return - task creation succeeded, just status update failed
		// Task will still be created and can be queried
	}

	// Async mode: block for HITL since we're already in background execution
	responseText, err := a.executeReasoningForA2A(ctx, userText, contextID, DefaultExecutionOptions())
	if err != nil {
		// Check if error is due to context cancellation/timeout
		if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
			slog.Warn("Task execution cancelled or timed out", "agent", a.id, "task", taskID, "error", err)
			// Update task status based on cancellation reason
			if ctx.Err() == context.DeadlineExceeded {
				if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
					slog.Error("Failed to update task status after timeout", "agent", a.id, "task", taskID, "error", updateErr)
				}
			} else {
				// Cancelled - task should be in CANCELLED state (handled by CancelTask)
				// But if it's still active, mark as cancelled
				if a.taskAwaiter.IsWaiting(taskID) {
					// Task was cancelled while waiting for input
					if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_CANCELLED, nil); updateErr != nil {
						slog.Error("Failed to update task status after cancellation", "agent", a.id, "task", taskID, "error", updateErr)
					}
				}
			}
		} else {
			slog.Error("Task execution failed", "agent", a.id, "task", taskID, "error", err)
			if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status to FAILED", "agent", a.id, "task", taskID, "error", updateErr)
			}
		}
		return
	}

	// Check if task is still waiting for user input (edge case: cancelled/timed out)
	// In normal flow, executeReasoningForA2A blocks until execution completes,
	// so this check will be false. However, if execution was cancelled or timed out
	// while waiting for input, the task may still be in INPUT_REQUIRED state.
	if a.taskAwaiter.IsWaiting(taskID) {
		// Task is still in INPUT_REQUIRED state, don't complete it yet
		// Execution will resume when user provides input via SendMessage
		slog.Info("Task still waiting for user input, not completing", "agent", a.id, "task", taskID)
		return
	}

	responseMessage := a.createResponseMessage(responseText, contextID, taskID)

	if err := a.services.Task().AddTaskMessage(ctx, taskID, responseMessage); err != nil {
		slog.Error("Failed to add response message to task", "agent", a.id, "task", taskID, "error", err)
		if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
			slog.Error("Failed to update task status to FAILED", "agent", a.id, "task", taskID, "error", updateErr)
		}
		return
	}

	if err := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_COMPLETED, responseMessage); err != nil {
		slog.Error("Failed to update task status to COMPLETED", "agent", a.id, "task", taskID, "error", err)
		// Don't return - task completed successfully, just status update failed
	}

	// Clear execution state from session metadata
	if err := a.ClearExecutionStateFromSession(ctx, contextID, taskID); err != nil {
		slog.Warn("Failed to clear execution state", "agent", a.id, "error", err)
	}
}

// resumeTaskExecution continues execution from saved state (async HITL resume)
func (a *Agent) resumeTaskExecution(
	taskID string,
	execState *ExecutionState,
	userDecision string,
) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("PANIC resuming task", "agent", a.id, "task", taskID, "panic", r)
			ctx := context.Background()
			if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status after panic", "agent", a.id, "task", taskID, "error", updateErr)
			}
		}
	}()

	if a.taskWorkers != nil {
		a.taskWorkers <- struct{}{}
		defer func() { <-a.taskWorkers }()
	}

	// Create context with taskID and user decision
	ctx := context.Background()
	ctx = EnsureAgentContext(ctx, taskID, execState.ContextID)
	ctx = context.WithValue(ctx, userDecisionContextKey, userDecision)

	// Create timeout context
	timeout := 1 * time.Hour // Default
	if a.config.Task != nil && a.config.Task.Timeout > 0 {
		timeout = time.Duration(a.config.Task.Timeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Register cancellation function
	a.executionsMu.Lock()
	a.activeExecutions[taskID] = cancel
	a.executionsMu.Unlock()

	defer func() {
		a.executionsMu.Lock()
		delete(a.activeExecutions, taskID)
		a.executionsMu.Unlock()
	}()

	// Update task to WORKING
	if err := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_WORKING, nil); err != nil {
		slog.Error("Failed to update task to WORKING", "agent", a.id, "task", taskID, "error", err)
	}

	// Create strategy
	// Get reasoning config from services (works for both config-based and programmatic agents)
	reasoningCfg := a.services.GetConfig()
	strategy, err := reasoning.CreateStrategy(reasoningCfg.Engine, reasoningCfg)
	if err != nil {
		if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
			slog.Error("Failed to update task status", "agent", a.id, "error", updateErr)
		}
		return
	}

	// Create output channel for streaming
	outputCh := make(chan *pb.Part, outputChannelBuffer)
	defer close(outputCh)

	// Restore reasoning state
	reasoningState, err := execState.RestoreReasoningState(
		outputCh,
		a.services,
		ctx,
	)
	if err != nil {
		slog.Error("Failed to restore reasoning state", "agent", a.id, "error", err)
		if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
			slog.Error("Failed to update task status", "agent", a.id, "error", updateErr)
		}
		return
	}

	// Continue execution from where it left off
	// The tool approval check will now see the user decision in context
	// and proceed accordingly
	cfg := a.services.GetConfig()
	maxIterations := a.getMaxIterations(cfg)
	toolDefs := a.services.Tools().GetAvailableTools()

	// Continue from the next iteration after the pause
	for reasoningState.Iteration() < maxIterations {
		currentIteration := reasoningState.NextIteration()

		select {
		case <-ctx.Done():
			if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status", "agent", a.id, "error", updateErr)
			}
			return
		default:
		}

		if err := strategy.PrepareIteration(currentIteration, reasoningState); err != nil {
			slog.Error("Error preparing iteration", "agent", a.id, "error", err)
			if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status", "agent", a.id, "error", updateErr)
			}
			return
		}

		promptSlots := a.buildPromptSlots(strategy)
		additionalContext := strategy.GetContextInjection(reasoningState)

		messages, err := a.services.Prompt().BuildMessages(ctx, execState.Query, promptSlots, reasoningState.AllMessages(), additionalContext)
		if err != nil {
			slog.Error("Error building messages", "agent", a.id, "error", err)
			if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status", "agent", a.id, "error", updateErr)
			}
			return
		}

		// Call LLM
		text, toolCalls, tokens, thinking, err := a.callLLMWithRetry(ctx, messages, toolDefs, outputCh, cfg, nil)
		if err != nil {
			slog.Error("LLM call failed", "agent", a.id, "error", err)
			if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status", "agent", a.id, "error", updateErr)
			}
			return
		}

		reasoningState.AddTokens(tokens)
		if text != "" {
			reasoningState.AppendResponse(text)
		}
		reasoningState.RecordFirstToolCalls(toolCalls)

		// Process tool calls if any
		// In async resume mode, always block for HITL since we're already in background execution
		var results []reasoning.ToolResult
		if len(toolCalls) > 0 {
			var shouldContinue bool
			ctx, results, shouldContinue, err = a.processToolCalls(ctx, text, toolCalls, thinking, reasoningState, outputCh, cfg, DefaultExecutionOptions())
			if err != nil {
				slog.Error("Error processing tool calls", "agent", a.id, "error", err)
				if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
					slog.Error("Failed to update task status", "agent", a.id, "error", updateErr)
				}
				return
			}
			if shouldContinue {
				// Another approval needed - save state again
				slog.Info("Task paused again for user input", "agent", a.id, "task", taskID)
				return
			}
		}

		if err := strategy.AfterIteration(currentIteration, text, toolCalls, results, reasoningState); err != nil {
			slog.Error("Error in strategy processing", "agent", a.id, "error", err)
			if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				slog.Error("Failed to update task status", "agent", a.id, "error", updateErr)
			}
			return
		}

		if strategy.ShouldStop(text, toolCalls, reasoningState) {
			break
		}
	}

	// Task completed
	finalResponse := reasoningState.GetAssistantResponse()
	responseMessage := a.createResponseMessage(finalResponse, execState.ContextID, taskID)

	if err := a.services.Task().AddTaskMessage(ctx, taskID, responseMessage); err != nil {
		slog.Error("Failed to add response message", "agent", a.id, "error", err)
	}

	if err := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_COMPLETED, responseMessage); err != nil {
		slog.Error("Failed to update task status", "agent", a.id, "error", err)
	}

	// Clear execution state from session metadata
	if err := a.ClearExecutionStateFromSession(ctx, execState.ContextID, taskID); err != nil {
		slog.Warn("Failed to clear execution state", "agent", a.id, "error", err)
	}
}

// executeReasoningForA2A executes agent reasoning and collects the full response text.
//
// The opts parameter controls HITL behavior:
//   - BlockForHITL=true: Blocks waiting for user input (streaming mode)
//   - BlockForHITL=false: Returns ErrInputRequired immediately when approval needed
//
// When ErrInputRequired is returned, the caller should return the task in
// INPUT_REQUIRED state. The client will send a follow-up message with the
// user's decision, which will be handled by handleInputRequiredResume.
func (a *Agent) executeReasoningForA2A(ctx context.Context, userText string, contextID string, opts ExecutionOptions) (string, error) {
	// Get reasoning config from services (works for both config-based and programmatic agents)
	reasoningCfg := a.services.GetConfig()
	strategy, err := reasoning.CreateStrategy(reasoningCfg.Engine, reasoningCfg)
	if err != nil {
		return "", fmt.Errorf("failed to create strategy: %w", err)
	}

	// Set SessionIDKey if not already set (may be set by caller)
	// Note: This preserves taskID if already set (empty string doesn't overwrite)
	ctx = EnsureAgentContext(ctx, "", contextID)

	streamCh, err := a.executeWithOptions(ctx, userText, nil, strategy, opts)
	if err != nil {
		return "", fmt.Errorf("reasoning failed: %w", err)
	}

	var fullResponse strings.Builder
	for part := range streamCh {
		// Extract text from parts for full response
		if textPart, ok := part.Part.(*pb.Part_Text); ok {
			fullResponse.WriteString(textPart.Text)
		}
	}

	// Check if task is in INPUT_REQUIRED state (non-blocking HITL mode)
	// This happens when BlockForHITL=false and tool approval was needed
	if !opts.BlockForHITL && a.services.Task() != nil {
		taskID := getTaskIDFromContext(ctx)
		if taskID != "" {
			task, err := a.services.Task().GetTask(ctx, taskID)
			if err == nil && task.Status != nil && task.Status.State == pb.TaskState_TASK_STATE_INPUT_REQUIRED {
				return "", ErrInputRequired
			}
		}
	}

	return fullResponse.String(), nil
}

func generateContextID() string {
	return fmt.Sprintf("ctx-%d", time.Now().UnixNano())
}

func extractTaskID(name string) string {
	if len(name) > 6 && name[:6] == "tasks/" {
		return name[6:]
	}
	return name
}
