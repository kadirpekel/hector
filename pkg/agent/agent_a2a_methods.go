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
		Content: []*pb.Part{
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

		streamCh, err := a.execute(ctx, userText, strategy)
		if err != nil {
			_ = a.services.Task().UpdateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil)
			return status.Errorf(codes.Internal, "reasoning failed: %v", err)
		}

		// Generate message ID once for all chunks
		messageID := fmt.Sprintf("msg-%d", time.Now().UnixNano())
		var fullResponse strings.Builder

		// ✅ FIX: Stream each chunk to client in real-time (token-by-token)
		for chunk := range streamCh {
			fullResponse.WriteString(chunk)

			// Send chunk immediately to client (real streaming!)
			if chunk != "" {
				chunkMsg := &pb.Message{
					MessageId: messageID,
					ContextId: contextID,
					TaskId:    task.Id,
					Role:      pb.Role_ROLE_AGENT,
					Content: []*pb.Part{
						{Part: &pb.Part_Text{Text: chunk}},
					},
				}

				if err := stream.Send(&pb.StreamResponse{
					Payload: &pb.StreamResponse_Msg{Msg: chunkMsg},
				}); err != nil {
					_ = a.services.Task().UpdateTaskStatus(ctx, task.Id, pb.TaskState_TASK_STATE_FAILED, nil)
					return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
				}
			}
		}

		// Store complete message in task history
		responseMessage := a.createResponseMessage(fullResponse.String(), contextID, task.Id)
		responseMessage.MessageId = messageID // Use same ID as streamed chunks

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

	// Generate message ID once for all chunks
	messageID := fmt.Sprintf("msg-%d", time.Now().UnixNano())

	// ✅ FIX: Stream each chunk to client in real-time (token-by-token)
	for chunk := range streamCh {
		if chunk != "" {
			chunkMsg := &pb.Message{
				MessageId: messageID,
				ContextId: contextID,
				Role:      pb.Role_ROLE_AGENT,
				Content: []*pb.Part{
					{Part: &pb.Part_Text{Text: chunk}},
				},
			}

			if err := stream.Send(&pb.StreamResponse{
				Payload: &pb.StreamResponse_Msg{Msg: chunkMsg},
			}); err != nil {
				return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
			}
		}
	}

	return nil
}

func (a *Agent) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error) {
	card := &pb.AgentCard{
		Name:        a.name,
		Description: a.description,
		Version:     "1.0.0",
		Capabilities: &pb.AgentCapabilities{
			Streaming: true,
		},
	}

	// Populate security information from config
	if a.config != nil && a.config.Security.IsEnabled() && len(a.config.Security.Schemes) > 0 {
		card.SecuritySchemes = make(map[string]*pb.SecurityScheme)
		for name, scheme := range a.config.Security.Schemes {
			pbScheme := convertConfigSecurityScheme(scheme)
			if pbScheme != nil {
				card.SecuritySchemes[name] = pbScheme
			}
		}

		// Convert security requirements
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
	}

	return card, nil
}

// convertConfigSecurityScheme converts config.SecurityScheme to pb.SecurityScheme
func convertConfigSecurityScheme(scheme config.SecurityScheme) *pb.SecurityScheme {
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
	// Other scheme types can be added here as needed
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
		// Create a copy to avoid copying locks in protobuf message
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
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{
					Text: responseText,
				},
			},
		},
	}
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

	if err := a.services.Task().UpdateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_WORKING, nil); err != nil {
		return
	}

	responseText, err := a.executeReasoningForA2A(ctx, userText, contextID)
	if err != nil {
		_ = a.services.Task().UpdateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil)
		return
	}

	responseMessage := a.createResponseMessage(responseText, contextID, taskID)

	if err := a.services.Task().AddTaskMessage(ctx, taskID, responseMessage); err != nil {
		_ = a.services.Task().UpdateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil)
		return
	}

	_ = a.services.Task().UpdateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_COMPLETED, responseMessage)
}

func (a *Agent) executeReasoningForA2A(ctx context.Context, userText string, contextID string) (string, error) {
	strategy, err := reasoning.CreateStrategy(a.config.Reasoning.Engine, a.config.Reasoning)
	if err != nil {
		return "", fmt.Errorf("failed to create strategy: %w", err)
	}

	streamCh, err := a.execute(ctx, userText, strategy)
	if err != nil {
		return "", fmt.Errorf("reasoning failed: %w", err)
	}

	var fullResponse string
	for chunk := range streamCh {
		fullResponse += chunk
	}

	return fullResponse, nil
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
