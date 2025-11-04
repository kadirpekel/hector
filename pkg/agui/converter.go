package agui

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	a2apb "github.com/kadirpekel/hector/pkg/a2a/pb"
	aguipb "github.com/kadirpekel/hector/pkg/agui/pb"
	"google.golang.org/protobuf/types/known/structpb"
)

// Converter converts A2A protocol messages to AG-UI events
type Converter struct {
	messageID      string
	contextID      string
	taskID         string
	currentBlockID string
	blockIndex     int32
}

// NewConverter creates a new AG-UI converter
func NewConverter(messageID, contextID, taskID string) *Converter {
	return &Converter{
		messageID:  messageID,
		contextID:  contextID,
		taskID:     taskID,
		blockIndex: 0,
	}
}

// ConvertPart converts an A2A Part to one or more AG-UI events
func (c *Converter) ConvertPart(part *a2apb.Part) []*aguipb.AGUIEvent {
	var events []*aguipb.AGUIEvent

	// Read AG-UI metadata hints if present
	aguiEventType := ""
	aguiBlockType := ""
	aguiBlockID := ""

	if part.Metadata != nil {
		// Check for AG-UI event type first
		if et, ok := part.Metadata.Fields["event_type"]; ok {
			aguiEventType = et.GetStringValue()
		}
		// Check for AG-UI block type
		if bt, ok := part.Metadata.Fields["block_type"]; ok {
			aguiBlockType = bt.GetStringValue()
		}
		// Check for AG-UI block ID
		if bid, ok := part.Metadata.Fields["block_id"]; ok {
			aguiBlockID = bid.GetStringValue()
		}
	}

	// Check if this is a thinking block (can be Text or Data part)
	if aguiEventType == "thinking" || aguiBlockType == "thinking" {
		// Close any open content block
		if c.currentBlockID != "" {
			events = append(events, NewContentBlockStopEvent(c.currentBlockID))
		}

		// Create new thinking block
		thinkingBlockID := aguiBlockID
		if thinkingBlockID == "" {
			thinkingBlockID = uuid.New().String()
		}

		// Get text content (from Text part or Data part fallback)
		text := part.GetText()
		if text == "" {
			// Try to get text from data part
			if dataPart := part.GetData(); dataPart != nil && dataPart.Data != nil {
				if textField, ok := dataPart.Data.Fields["text"]; ok {
					text = textField.GetStringValue()
				}
			}
		}

		events = append(events, NewThinkingStartEvent(thinkingBlockID, "")) // empty title
		if text != "" {
			events = append(events, NewThinkingDeltaEvent(thinkingBlockID, text))
		}
		events = append(events, NewThinkingStopEvent(thinkingBlockID, "")) // empty signature
		c.currentBlockID = ""
		c.blockIndex++
		return events
	}

	// Handle regular text parts with AG-UI metadata
	if text := part.GetText(); text != "" {
		// Determine block type from AG-UI metadata
		blockType := "text"
		if aguiBlockType != "" {
			blockType = aguiBlockType
		}

		// Regular content block
		if c.currentBlockID == "" {
			c.currentBlockID = aguiBlockID
			if c.currentBlockID == "" {
				c.currentBlockID = uuid.New().String()
			}
			events = append(events, NewContentBlockStartEvent(c.currentBlockID, blockType, c.blockIndex))
			c.blockIndex++
		}

		// Emit content block delta
		events = append(events, NewContentBlockDeltaEvent(c.currentBlockID, text))
		return events
	}

	// Handle tool calls and tool results (AG-UI metadata only)
	if aguiEventType == "tool_call" {
		// Tool calls have is_error field to distinguish call from result
		if part.Metadata != nil {
			_, hasIsError := part.Metadata.Fields["is_error"]

			if hasIsError {
				// This is a tool result
				toolCallID, result, errorMsg, isError := c.extractToolResultInfo(part)
				finalInput := make(map[string]interface{})
				events = append(events, NewToolCallStopEvent(toolCallID, finalInput, result, errorMsg, isError))
			} else {
				// This is a tool call
				// Close any open content block
				if c.currentBlockID != "" {
					events = append(events, NewContentBlockStopEvent(c.currentBlockID))
					c.currentBlockID = ""
				}

				toolCallID, toolName, input := c.extractToolCallInfo(part)
				events = append(events, NewToolCallStartEvent(toolCallID, toolName, input))
			}
		}
	}

	return events
}

// CloseCurrentBlock closes the currently open content block if any
func (c *Converter) CloseCurrentBlock() []*aguipb.AGUIEvent {
	if c.currentBlockID == "" {
		return nil
	}

	event := NewContentBlockStopEvent(c.currentBlockID)
	c.currentBlockID = ""
	return []*aguipb.AGUIEvent{event}
}

// extractToolCallInfo extracts tool call information from a Part
func (c *Converter) extractToolCallInfo(part *a2apb.Part) (string, string, map[string]interface{}) {
	toolCallID := ""
	toolName := ""
	input := make(map[string]interface{})

	// Read AG-UI metadata
	if part.Metadata != nil {
		if id, ok := part.Metadata.Fields["tool_call_id"]; ok {
			toolCallID = id.GetStringValue()
		}
		if name, ok := part.Metadata.Fields["tool_name"]; ok {
			toolName = name.GetStringValue()
		}
	}

	// Extract from DataPart
	if dataPart := part.GetData(); dataPart != nil && dataPart.Data != nil {
		if name, ok := dataPart.Data.Fields["name"]; ok && toolName == "" {
			toolName = name.GetStringValue()
		}
		if id, ok := dataPart.Data.Fields["id"]; ok && toolCallID == "" {
			toolCallID = id.GetStringValue()
		}

		if inputField, ok := dataPart.Data.Fields["input"]; ok {
			// Try to parse input as JSON
			if inputStr := inputField.GetStringValue(); inputStr != "" {
				json.Unmarshal([]byte(inputStr), &input)
			} else if inputStruct := inputField.GetStructValue(); inputStruct != nil {
				// Convert protobuf Struct to map
				for k, v := range inputStruct.Fields {
					input[k] = convertProtoValue(v)
				}
			}
		}

		// Check for 'arguments' field (standard A2A format)
		if argsField, ok := dataPart.Data.Fields["arguments"]; ok {
			if argsStruct := argsField.GetStructValue(); argsStruct != nil {
				for k, v := range argsStruct.Fields {
					input[k] = convertProtoValue(v)
				}
			}
		}
	}

	if toolCallID == "" {
		toolCallID = uuid.New().String()
	}

	return toolCallID, toolName, input
}

// extractToolResultInfo extracts tool result information from a Part
func (c *Converter) extractToolResultInfo(part *a2apb.Part) (string, string, string, bool) {
	toolCallID := ""
	result := ""
	errorMsg := ""
	isError := false

	// Read AG-UI metadata
	if part.Metadata != nil {
		if id, ok := part.Metadata.Fields["tool_call_id"]; ok {
			toolCallID = id.GetStringValue()
		}
		if isErr, ok := part.Metadata.Fields["is_error"]; ok {
			isError = isErr.GetBoolValue()
		}
	}

	// Extract from DataPart
	if dataPart := part.GetData(); dataPart != nil && dataPart.Data != nil {
		if tcID, ok := dataPart.Data.Fields["tool_call_id"]; ok && toolCallID == "" {
			toolCallID = tcID.GetStringValue()
		}

		if content, ok := dataPart.Data.Fields["content"]; ok {
			result = content.GetStringValue()
		}

		if err, ok := dataPart.Data.Fields["error"]; ok {
			errorMsg = err.GetStringValue()
			if errorMsg != "" {
				isError = true
			}
		}
	}

	if toolCallID == "" {
		toolCallID = uuid.New().String()
	}

	return toolCallID, result, errorMsg, isError
}

// convertProtoValue converts a protobuf Value to a Go value
func convertProtoValue(v *structpb.Value) interface{} {
	if v == nil {
		return nil
	}

	switch kind := v.Kind.(type) {
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_NumberValue:
		return kind.NumberValue
	case *structpb.Value_StringValue:
		return kind.StringValue
	case *structpb.Value_BoolValue:
		return kind.BoolValue
	case *structpb.Value_StructValue:
		m := make(map[string]interface{})
		if kind.StructValue != nil {
			for k, val := range kind.StructValue.Fields {
				m[k] = convertProtoValue(val)
			}
		}
		return m
	case *structpb.Value_ListValue:
		list := []interface{}{}
		if kind.ListValue != nil {
			for _, val := range kind.ListValue.Values {
				list = append(list, convertProtoValue(val))
			}
		}
		return list
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ConvertStreamResponse converts an A2A StreamResponse to AG-UI events
func ConvertStreamResponse(resp *a2apb.StreamResponse, messageID, contextID, taskID string) []*aguipb.AGUIEvent {
	converter := NewConverter(messageID, contextID, taskID)
	var events []*aguipb.AGUIEvent

	switch payload := resp.Payload.(type) {
	case *a2apb.StreamResponse_Task:
		// Task events
		task := payload.Task
		if task.Status != nil {
			status := task.Status.State.String()
			if len(status) > 11 && status[:11] == "TASK_STATE_" {
				status = status[11:] // Remove prefix
			}

			switch task.Status.State {
			case a2apb.TaskState_TASK_STATE_SUBMITTED:
				events = append(events, NewTaskStartEvent(task.Id, task.ContextId, ""))
			case a2apb.TaskState_TASK_STATE_WORKING:
				events = append(events, NewTaskUpdateEvent(task.Id, "working", nil))
			case a2apb.TaskState_TASK_STATE_COMPLETED:
				events = append(events, NewTaskCompleteEvent(task.Id, nil))
			case a2apb.TaskState_TASK_STATE_FAILED:
				events = append(events, NewTaskErrorEvent(task.Id, "Task failed", "FAILED", nil))
			}
		}

	case *a2apb.StreamResponse_Msg:
		// Message with parts - convert each part
		msg := payload.Msg
		for _, part := range msg.Parts {
			partEvents := converter.ConvertPart(part)
			events = append(events, partEvents...)
		}

	case *a2apb.StreamResponse_StatusUpdate:
		// Task status update
		update := payload.StatusUpdate
		status := update.Status.State.String()
		if len(status) > 11 && status[:11] == "TASK_STATE_" {
			status = status[11:]
		}
		events = append(events, NewTaskUpdateEvent(update.TaskId, status, nil))

	case *a2apb.StreamResponse_ArtifactUpdate:
		// Artifact update - treat as content blocks
		artifact := payload.ArtifactUpdate.Artifact
		if artifact != nil {
			for _, part := range artifact.Parts {
				partEvents := converter.ConvertPart(part)
				events = append(events, partEvents...)
			}
		}
	}

	return events
}

// CreateMessageStartEvent creates a message start event
func CreateMessageStartEvent(messageID, contextID, taskID, role string) *aguipb.AGUIEvent {
	return NewMessageStartEvent(messageID, contextID, taskID, role)
}

// CreateMessageStopEvent creates a message stop event
func CreateMessageStopEvent(messageID string) *aguipb.AGUIEvent {
	return NewMessageStopEvent(messageID)
}
