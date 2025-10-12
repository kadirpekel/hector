package protocol

import (
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/protobuf/types/known/structpb"
)

// ============================================================================
// MESSAGE CREATION HELPERS
// Pure A2A protocol - no extensions, just native Part types
// ============================================================================

// CreateUserMessage creates a new pb.Message with ROLE_USER and text content.
func CreateUserMessage(text string) *pb.Message {
	return &pb.Message{
		Role: pb.Role_ROLE_USER,
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: text},
			},
		},
	}
}

// CreateTextMessage creates a new pb.Message with a given role and text content.
func CreateTextMessage(role pb.Role, text string) *pb.Message {
	return &pb.Message{
		Role: role,
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: text},
			},
		},
	}
}

// ============================================================================
// TOOL CALLING HELPERS
// Uses native A2A DataPart for structured tool call/result data
// ============================================================================

// ToolCall represents a tool invocation
// This is a helper struct for encoding/decoding tool calls from DataPart
type ToolCall struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	Error      string `json:"error,omitempty"`
}

// CreateToolCallPart creates a Part containing a tool call using DataPart
func CreateToolCallPart(toolCall *ToolCall) *pb.Part {
	data, _ := structpb.NewStruct(map[string]interface{}{
		"id":        toolCall.ID,
		"name":      toolCall.Name,
		"arguments": toolCall.Args,
	})

	metadata, _ := structpb.NewStruct(map[string]interface{}{
		"part_type": "tool_call",
	})

	return &pb.Part{
		Part: &pb.Part_Data{
			Data: &pb.DataPart{Data: data},
		},
		Metadata: metadata,
	}
}

// CreateToolResultPart creates a Part containing a tool result using DataPart
func CreateToolResultPart(result *ToolResult) *pb.Part {
	data, _ := structpb.NewStruct(map[string]interface{}{
		"tool_call_id": result.ToolCallID,
		"content":      result.Content,
		"error":        result.Error,
	})

	metadata, _ := structpb.NewStruct(map[string]interface{}{
		"part_type": "tool_result",
	})

	return &pb.Part{
		Part: &pb.Part_Data{
			Data: &pb.DataPart{Data: data},
		},
		Metadata: metadata,
	}
}

// IsToolCallPart checks if a part represents a tool call
func IsToolCallPart(part *pb.Part) bool {
	if part == nil || part.Metadata == nil {
		return false
	}
	if partType, ok := part.Metadata.Fields["part_type"]; ok {
		return partType.GetStringValue() == "tool_call"
	}
	return false
}

// IsToolResultPart checks if a part represents a tool result
func IsToolResultPart(part *pb.Part) bool {
	if part == nil || part.Metadata == nil {
		return false
	}
	if partType, ok := part.Metadata.Fields["part_type"]; ok {
		return partType.GetStringValue() == "tool_result"
	}
	return false
}

// ExtractToolCall extracts a ToolCall from a DataPart
func ExtractToolCall(part *pb.Part) *ToolCall {
	if !IsToolCallPart(part) {
		return nil
	}

	dataPart := part.GetData()
	if dataPart == nil || dataPart.Data == nil {
		return nil
	}

	fields := dataPart.Data.Fields
	tc := &ToolCall{}

	if id, ok := fields["id"]; ok {
		tc.ID = id.GetStringValue()
	}
	if name, ok := fields["name"]; ok {
		tc.Name = name.GetStringValue()
	}
	if args, ok := fields["arguments"]; ok {
		tc.Args = args.GetStructValue().AsMap()
	}

	return tc
}

// ExtractToolResult extracts a ToolResult from a DataPart
func ExtractToolResult(part *pb.Part) *ToolResult {
	if !IsToolResultPart(part) {
		return nil
	}

	dataPart := part.GetData()
	if dataPart == nil || dataPart.Data == nil {
		return nil
	}

	fields := dataPart.Data.Fields
	result := &ToolResult{}

	if id, ok := fields["tool_call_id"]; ok {
		result.ToolCallID = id.GetStringValue()
	}
	if content, ok := fields["content"]; ok {
		result.Content = content.GetStringValue()
	}
	if err, ok := fields["error"]; ok {
		result.Error = err.GetStringValue()
	}

	return result
}

// GetToolCallsFromMessage extracts all tool calls from a message
func GetToolCallsFromMessage(msg *pb.Message) []*ToolCall {
	if msg == nil {
		return nil
	}

	var toolCalls []*ToolCall
	for _, part := range msg.Content {
		if tc := ExtractToolCall(part); tc != nil {
			toolCalls = append(toolCalls, tc)
		}
	}
	return toolCalls
}

// GetToolResultsFromMessage extracts all tool results from a message
func GetToolResultsFromMessage(msg *pb.Message) []*ToolResult {
	if msg == nil {
		return nil
	}

	var results []*ToolResult
	for _, part := range msg.Content {
		if result := ExtractToolResult(part); result != nil {
			results = append(results, result)
		}
	}
	return results
}

// ============================================================================
// CONTENT EXTRACTION HELPERS
// ============================================================================

// ExtractTextFromMessage extracts the first text content from a pb.Message.
func ExtractTextFromMessage(msg *pb.Message) string {
	if msg == nil || len(msg.Content) == 0 {
		return ""
	}
	for _, part := range msg.Content {
		if text := part.GetText(); text != "" {
			return text
		}
	}
	return ""
}

// ExtractAllTextFromMessage extracts all text parts concatenated
func ExtractAllTextFromMessage(msg *pb.Message) string {
	if msg == nil || len(msg.Content) == 0 {
		return ""
	}
	var result string
	for _, part := range msg.Content {
		if text := part.GetText(); text != "" {
			result += text
		}
	}
	return result
}

// ExtractTextFromTask extracts the text content from the last user message in a pb.Task.
func ExtractTextFromTask(task *pb.Task) string {
	if task == nil || len(task.History) == 0 {
		return ""
	}
	for i := len(task.History) - 1; i >= 0; i-- {
		if task.History[i].Role == pb.Role_ROLE_USER {
			return ExtractTextFromMessage(task.History[i])
		}
	}
	return ""
}

// HasToolCalls checks if a message contains any tool calls
func HasToolCalls(msg *pb.Message) bool {
	if msg == nil {
		return false
	}
	for _, part := range msg.Content {
		if IsToolCallPart(part) {
			return true
		}
	}
	return false
}

// HasToolResults checks if a message contains any tool results
func HasToolResults(msg *pb.Message) bool {
	if msg == nil {
		return false
	}
	for _, part := range msg.Content {
		if IsToolResultPart(part) {
			return true
		}
	}
	return false
}
