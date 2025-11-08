package agent

import (
	"context"
	"fmt"
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

	// Task is waiting for user input - provide it and resume
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

		if err := a.services.Task().UpdateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_WORKING, nil); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update task status: %v", err)
		}

		// Add taskID to context for HITL support (tool approval)
		ctx = context.WithValue(ctx, taskIDContextKey, task.Id)
		ctx = context.WithValue(ctx, SessionIDKey, contextID)

		responseText, err := a.executeReasoningForA2A(ctx, userText, contextID)
		if err != nil {
			if updateErr := a.services.Task().UpdateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
				return nil, status.Errorf(codes.Internal, "agent execution failed: %v (status update failed: %v)", err, updateErr)
			}
			return nil, status.Errorf(codes.Internal, "agent execution failed: %v", err)
		}

		responseMessage := a.createResponseMessage(responseText, contextID, task.Id)

		if err := a.services.Task().AddTaskMessage(ctx, task.Id, responseMessage); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to add response message: %v", err)
		}

		if err := a.services.Task().UpdateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_COMPLETED, responseMessage); err != nil {
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

	responseText, err := a.executeReasoningForA2A(ctx, userText, contextID)
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

		if err := a.services.Task().UpdateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_WORKING, nil); err != nil {
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

		strategy, err := reasoning.CreateStrategy(a.config.Reasoning.Engine, a.config.Reasoning)
		if err != nil {
			_ = a.services.Task().UpdateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil)
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
		ctx = context.WithValue(ctx, taskIDContextKey, task.Id)

		streamCh, err := a.execute(ctx, userText, strategy)
		if err != nil {
			_ = a.services.Task().UpdateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil)
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
				_ = a.services.Task().UpdateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil)
				return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
			}
		}

		responseMessage := a.createResponseMessage(fullResponse.String(), contextID, task.Id)
		responseMessage.MessageId = messageID

		if err := a.services.Task().AddTaskMessage(ctx, task.Id, responseMessage); err != nil {
			return status.Errorf(codes.Internal, "failed to add response message: %v", err)
		}

		if err := a.services.Task().UpdateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_COMPLETED, responseMessage); err != nil {
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

	strategy, err := reasoning.CreateStrategy(a.config.Reasoning.Engine, a.config.Reasoning)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create strategy: %v", err)
	}

	streamCh, err := a.execute(ctx, userText, strategy)
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
		if globalConfig.Global.Auth.IsEnabled() {
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
	a.executionsMu.RLock()
	if cancelFunc, exists := a.activeExecutions[taskID]; exists {
		a.executionsMu.RUnlock()
		cancelFunc() // Cancel the context to stop execution
	} else {
		a.executionsMu.RUnlock()
	}

	// Cancel any waiting input requests
	a.taskAwaiter.CancelWaiting(taskID)

	task, err := a.services.Task().CancelTask(ctx, taskID)
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
	if a.taskWorkers != nil {
		a.taskWorkers <- struct{}{}
		defer func() { <-a.taskWorkers }()
	}

	ctx := context.Background()

	// Add taskID to context for tool approval logic
	ctx = context.WithValue(ctx, taskIDContextKey, taskID)

	// Create cancellable context for this task execution
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Register cancellation function for this task
	a.executionsMu.Lock()
	a.activeExecutions[taskID] = cancel
	a.executionsMu.Unlock()

	defer func() {
		a.executionsMu.Lock()
		delete(a.activeExecutions, taskID)
		a.executionsMu.Unlock()
	}()

	if err := a.services.Task().UpdateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_WORKING, nil); err != nil {
		return
	}

	// Execute with HITL support
	// The execute() method handles HITL inline - when a tool requires approval,
	// it waits for user input, then continues execution. The channel from execute()
	// remains open until execution fully completes (including resumed execution
	// after approval). So executeReasoningForA2A will block until everything is done.
	// The IsWaiting check below handles edge cases where execution was cancelled
	// or timed out while waiting for input.
	ctx = context.WithValue(ctx, SessionIDKey, contextID)

	responseText, err := a.executeReasoningForA2A(ctx, userText, contextID)
	if err != nil {
		_ = a.services.Task().UpdateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil)
		return
	}

	// Check if task is still waiting for user input (edge case: cancelled/timed out)
	// In normal flow, executeReasoningForA2A blocks until execution completes,
	// so this check will be false. However, if execution was cancelled or timed out
	// while waiting for input, the task may still be in INPUT_REQUIRED state.
	if a.taskAwaiter.IsWaiting(taskID) {
		// Task is still in INPUT_REQUIRED state, don't complete it yet
		// Execution will resume when user provides input via SendMessage
		return
	}

	responseMessage := a.createResponseMessage(responseText, contextID, taskID)

	if err := a.services.Task().AddTaskMessage(ctx, taskID, responseMessage); err != nil {
		_ = a.services.Task().UpdateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil)
		return
	}

	_ = a.services.Task().UpdateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_COMPLETED, responseMessage)
}

// executeReasoningForA2A executes agent reasoning and collects the full response text.
// For tasks with HITL support, this will block until execution fully completes,
// including any resumed execution after user provides input for tool approval.
func (a *Agent) executeReasoningForA2A(ctx context.Context, userText string, contextID string) (string, error) {
	strategy, err := reasoning.CreateStrategy(a.config.Reasoning.Engine, a.config.Reasoning)
	if err != nil {
		return "", fmt.Errorf("failed to create strategy: %w", err)
	}

	// Set SessionIDKey if not already set (may be set by caller)
	if ctx.Value(SessionIDKey) == nil {
		ctx = context.WithValue(ctx, SessionIDKey, contextID)
	}

	streamCh, err := a.execute(ctx, userText, strategy)
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
