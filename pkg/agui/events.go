package agui

import (
	"github.com/google/uuid"
	pb "github.com/kadirpekel/hector/pkg/agui/pb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ============================================================================
// Message Event Builders
// ============================================================================

// NewMessageStartEvent creates a message_start event
func NewMessageStartEvent(messageID, contextID, taskID, role string) *pb.AGUIEvent {
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_MESSAGE_START,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_MessageStart{
			MessageStart: &pb.MessageStartPayload{
				MessageId: messageID,
				ContextId: contextID,
				TaskId:    taskID,
				Role:      role,
			},
		},
	}
}

// NewMessageStopEvent creates a message_stop event
func NewMessageStopEvent(messageID string) *pb.AGUIEvent {
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_MESSAGE_STOP,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_MessageStop{
			MessageStop: &pb.MessageStopPayload{
				MessageId: messageID,
			},
		},
	}
}

// ============================================================================
// Content Block Event Builders
// ============================================================================

// NewContentBlockStartEvent creates a content_block_start event
func NewContentBlockStartEvent(blockID, blockType string, index int32) *pb.AGUIEvent {
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_CONTENT_BLOCK_START,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_ContentBlockStart{
			ContentBlockStart: &pb.ContentBlockStartPayload{
				BlockId:   blockID,
				BlockType: blockType,
				Index:     index,
			},
		},
	}
}

// NewContentBlockDeltaEvent creates a content_block_delta event
func NewContentBlockDeltaEvent(blockID, delta string) *pb.AGUIEvent {
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_CONTENT_BLOCK_DELTA,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_ContentBlockDelta{
			ContentBlockDelta: &pb.ContentBlockDeltaPayload{
				BlockId: blockID,
				Delta:   delta,
			},
		},
	}
}

// NewContentBlockStopEvent creates a content_block_stop event
func NewContentBlockStopEvent(blockID string) *pb.AGUIEvent {
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_CONTENT_BLOCK_STOP,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_ContentBlockStop{
			ContentBlockStop: &pb.ContentBlockStopPayload{
				BlockId: blockID,
			},
		},
	}
}

// ============================================================================
// Tool Call Event Builders
// ============================================================================

// NewToolCallStartEvent creates a tool_call_start event
func NewToolCallStartEvent(toolCallID, toolName string, input map[string]interface{}) *pb.AGUIEvent {
	inputStruct, _ := structpb.NewStruct(input)
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_TOOL_CALL_START,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_ToolCallStart{
			ToolCallStart: &pb.ToolCallStartPayload{
				ToolCallId: toolCallID,
				ToolName:   toolName,
				Input:      inputStruct,
			},
		},
	}
}

// NewToolCallStopEvent creates a tool_call_stop event
func NewToolCallStopEvent(toolCallID string, finalInput map[string]interface{}, result, errorMsg string, isError bool) *pb.AGUIEvent {
	inputStruct, _ := structpb.NewStruct(finalInput)
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_TOOL_CALL_STOP,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_ToolCallStop{
			ToolCallStop: &pb.ToolCallStopPayload{
				ToolCallId: toolCallID,
				FinalInput: inputStruct,
				Result:     result,
				Error:      errorMsg,
				IsError:    isError,
			},
		},
	}
}

// ============================================================================
// Thinking Event Builders
// ============================================================================

// NewThinkingStartEvent creates a thinking_start event
func NewThinkingStartEvent(thinkingID, title string) *pb.AGUIEvent {
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_THINKING_START,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_ThinkingStart{
			ThinkingStart: &pb.ThinkingStartPayload{
				ThinkingId: thinkingID,
				Title:      title,
			},
		},
	}
}

// NewThinkingDeltaEvent creates a thinking_delta event
func NewThinkingDeltaEvent(thinkingID, delta string) *pb.AGUIEvent {
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_THINKING_DELTA,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_ThinkingDelta{
			ThinkingDelta: &pb.ThinkingDeltaPayload{
				ThinkingId: thinkingID,
				Delta:      delta,
			},
		},
	}
}

// NewThinkingStopEvent creates a thinking_stop event
func NewThinkingStopEvent(thinkingID, signature string) *pb.AGUIEvent {
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_THINKING_STOP,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_ThinkingStop{
			ThinkingStop: &pb.ThinkingStopPayload{
				ThinkingId: thinkingID,
				Signature:  signature,
			},
		},
	}
}

// ============================================================================
// Task Event Builders
// ============================================================================

// NewTaskStartEvent creates a task_start event
func NewTaskStartEvent(taskID, contextID, description string) *pb.AGUIEvent {
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_TASK_START,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_TaskStart{
			TaskStart: &pb.TaskStartPayload{
				TaskId:      taskID,
				ContextId:   contextID,
				Description: description,
			},
		},
	}
}

// NewTaskUpdateEvent creates a task_update event
func NewTaskUpdateEvent(taskID, status string, metadata map[string]interface{}) *pb.AGUIEvent {
	metadataStruct, _ := structpb.NewStruct(metadata)
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_TASK_UPDATE,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_TaskUpdate{
			TaskUpdate: &pb.TaskUpdatePayload{
				TaskId:   taskID,
				Status:   status,
				Metadata: metadataStruct,
			},
		},
	}
}

// NewTaskCompleteEvent creates a task_complete event
func NewTaskCompleteEvent(taskID string, result map[string]interface{}) *pb.AGUIEvent {
	resultStruct, _ := structpb.NewStruct(result)
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_TASK_COMPLETE,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_TaskComplete{
			TaskComplete: &pb.TaskCompletePayload{
				TaskId: taskID,
				Result: resultStruct,
			},
		},
	}
}

// NewTaskErrorEvent creates a task_error event
func NewTaskErrorEvent(taskID, errorMessage, errorCode string, details map[string]interface{}) *pb.AGUIEvent {
	detailsStruct, _ := structpb.NewStruct(details)
	return &pb.AGUIEvent{
		EventId:   uuid.New().String(),
		Type:      pb.AGUIEventType_AGUI_EVENT_TYPE_TASK_ERROR,
		Timestamp: timestamppb.Now(),
		Payload: &pb.AGUIEvent_TaskError{
			TaskError: &pb.TaskErrorPayload{
				TaskId:       taskID,
				ErrorMessage: errorMessage,
				ErrorCode:    errorCode,
				Details:      detailsStruct,
			},
		},
	}
}
