package protocol

import (
	"github.com/google/uuid"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/protobuf/types/known/structpb"
)

// AG-UI Event Types (from AG-UI specification)
// These are used as metadata hints in A2A parts to make them AG-UI-native
const (
	AGUIEventTypeMessage      = "message"
	AGUIEventTypeContentBlock = "content_block"
	AGUIEventTypeToolCall     = "tool_call"
	AGUIEventTypeThinking     = "thinking"
	AGUIEventTypeTask         = "task"
	AGUIEventTypeError        = "error"
)

// AG-UI Block Types
const (
	AGUIBlockTypeText     = "text"
	AGUIBlockTypeThinking = "thinking"
	AGUIBlockTypeCode     = "code"
)

// CreateTextPartWithAGUI creates a text part with AG-UI metadata
func CreateTextPartWithAGUI(text string, blockID string, blockIndex int) *pb.Part {
	if blockID == "" {
		blockID = uuid.New().String()
	}

	metadata, _ := structpb.NewStruct(map[string]interface{}{
		"event_type":  AGUIEventTypeContentBlock,
		"block_type":  AGUIBlockTypeText,
		"block_id":    blockID,
		"block_index": blockIndex,
	})

	return &pb.Part{
		Part:     &pb.Part_Text{Text: text},
		Metadata: metadata,
	}
}

// CreateThinkingPart creates a thinking/reasoning part with AG-UI metadata
func CreateThinkingPart(text string, blockID string, blockIndex int) *pb.Part {
	if blockID == "" {
		blockID = uuid.New().String()
	}

	metadata, _ := structpb.NewStruct(map[string]interface{}{
		"event_type":  AGUIEventTypeThinking,
		"block_type":  AGUIBlockTypeThinking,
		"block_id":    blockID,
		"block_index": blockIndex,
	})

	return &pb.Part{
		Part:     &pb.Part_Text{Text: text},
		Metadata: metadata,
	}
}

// CreateThinkingPartWithData creates a thinking part with structured data
// Backend emits structured data, client decides how to render
func CreateThinkingPartWithData(text string, thinkingType string, data map[string]interface{}) *pb.Part {
	blockID := uuid.New().String()

	// Metadata: AG-UI event type + optional client rendering hints
	metadata := map[string]interface{}{
		"event_type":  AGUIEventTypeThinking,
		"block_type":  AGUIBlockTypeThinking,
		"block_id":    blockID,
		"block_index": 0,
	}

	// Add thinking_type as rendering hint for client
	if thinkingType != "" {
		metadata["thinking_type"] = thinkingType
	}

	metadataStruct, _ := structpb.NewStruct(metadata)

	// If structured data provided, emit as Data part
	// Text serves as fallback for simple clients
	if len(data) > 0 {
		// Ensure text is in data for clients that prefer it
		data["text"] = text
		dataStruct, err := structpb.NewStruct(data)
		if err != nil {
			// If we can't serialize data, fall back to text-only
			return &pb.Part{
				Part:     &pb.Part_Text{Text: text},
				Metadata: metadataStruct,
			}
		}

		return &pb.Part{
			Part: &pb.Part_Data{
				Data: &pb.DataPart{
					Data: dataStruct,
				},
			},
			Metadata: metadataStruct,
		}
	}

	// Fallback to text-only part
	return &pb.Part{
		Part:     &pb.Part_Text{Text: text},
		Metadata: metadataStruct,
	}
}

// CreateToolCallPartWithAGUI creates an enhanced tool call part with AG-UI metadata
func CreateToolCallPartWithAGUI(toolCall *ToolCall) *pb.Part {
	data, _ := structpb.NewStruct(map[string]interface{}{
		"id":        toolCall.ID,
		"name":      toolCall.Name,
		"arguments": toolCall.Args,
	})

	metadata, _ := structpb.NewStruct(map[string]interface{}{
		"event_type":   AGUIEventTypeToolCall,
		"tool_call_id": toolCall.ID,
		"tool_name":    toolCall.Name,
	})

	return &pb.Part{
		Part: &pb.Part_Data{
			Data: &pb.DataPart{Data: data},
		},
		Metadata: metadata,
	}
}

// CreateToolResultPartWithAGUI creates an enhanced tool result part with AG-UI metadata
func CreateToolResultPartWithAGUI(result *ToolResult) *pb.Part {
	data, _ := structpb.NewStruct(map[string]interface{}{
		"tool_call_id": result.ToolCallID,
		"content":      result.Content,
		"error":        result.Error,
	})

	isError := result.Error != ""
	// For streaming support: check if this is explicitly marked as final
	// We'll add a parameter to indicate if this is a final result
	// Default behavior: if error exists, it's final; otherwise assume it's incremental during streaming
	// The caller should pass isFinal parameter, but for backward compat, we default based on error
	isFinal := isError // If there's an error, it's always final

	metadata, _ := structpb.NewStruct(map[string]interface{}{
		"event_type":   AGUIEventTypeToolCall,
		"tool_call_id": result.ToolCallID,
		"is_error":     isError,
		"is_final":     isFinal, // Will be overridden by CreateToolResultPartWithFinal if needed
	})

	return &pb.Part{
		Part: &pb.Part_Data{
			Data: &pb.DataPart{Data: data},
		},
		Metadata: metadata,
	}
}

// CreateErrorPart creates an error part with AG-UI metadata
func CreateErrorPart(errorText string, errorCode string) *pb.Part {
	metadata, _ := structpb.NewStruct(map[string]interface{}{
		"event_type": AGUIEventTypeError,
		"error_code": errorCode,
	})

	return &pb.Part{
		Part:     &pb.Part_Text{Text: errorText},
		Metadata: metadata,
	}
}

// IsThinkingPart checks if a part is a thinking block
func IsThinkingPart(part *pb.Part) bool {
	if part == nil || part.Metadata == nil {
		return false
	}

	// Check AG-UI metadata
	if eventType, ok := part.Metadata.Fields["event_type"]; ok {
		return eventType.GetStringValue() == AGUIEventTypeThinking
	}

	return false
}

// GetAGUIEventType extracts the AG-UI event type from part metadata
func GetAGUIEventType(part *pb.Part) string {
	if part == nil || part.Metadata == nil {
		return ""
	}

	if eventType, ok := part.Metadata.Fields["event_type"]; ok {
		return eventType.GetStringValue()
	}

	return ""
}

// GetAGUIBlockID extracts the AG-UI block ID from part metadata
func GetAGUIBlockID(part *pb.Part) string {
	if part == nil || part.Metadata == nil {
		return ""
	}

	if blockID, ok := part.Metadata.Fields["block_id"]; ok {
		return blockID.GetStringValue()
	}

	return ""
}

// GetAGUIBlockType extracts the AG-UI block type from part metadata
func GetAGUIBlockType(part *pb.Part) string {
	if part == nil || part.Metadata == nil {
		return ""
	}

	if blockType, ok := part.Metadata.Fields["block_type"]; ok {
		return blockType.GetStringValue()
	}

	return ""
}
