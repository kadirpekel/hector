package protocol

import (
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/protobuf/types/known/structpb"
)

func CreateUserMessage(text string) *pb.Message {
	return &pb.Message{
		Role: pb.Role_ROLE_USER,
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: text},
			},
		},
	}
}

func CreateTextMessage(role pb.Role, text string) *pb.Message {
	return &pb.Message{
		Role: role,
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: text},
			},
		},
	}
}

type ToolCall struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"arguments"`
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	Error      string `json:"error,omitempty"`
}

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

func IsToolCallPart(part *pb.Part) bool {
	if part == nil || part.Metadata == nil {
		return false
	}
	if partType, ok := part.Metadata.Fields["part_type"]; ok {
		return partType.GetStringValue() == "tool_call"
	}
	return false
}

func IsToolResultPart(part *pb.Part) bool {
	if part == nil || part.Metadata == nil {
		return false
	}
	if partType, ok := part.Metadata.Fields["part_type"]; ok {
		return partType.GetStringValue() == "tool_result"
	}
	return false
}

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

func GetToolCallsFromMessage(msg *pb.Message) []*ToolCall {
	if msg == nil {
		return nil
	}

	var toolCalls []*ToolCall
	for _, part := range msg.Parts {
		if tc := ExtractToolCall(part); tc != nil {
			toolCalls = append(toolCalls, tc)
		}
	}
	return toolCalls
}

func GetToolResultsFromMessage(msg *pb.Message) []*ToolResult {
	if msg == nil {
		return nil
	}

	var results []*ToolResult
	for _, part := range msg.Parts {
		if result := ExtractToolResult(part); result != nil {
			results = append(results, result)
		}
	}
	return results
}

func ExtractTextFromMessage(msg *pb.Message) string {
	if msg == nil || len(msg.Parts) == 0 {
		return ""
	}
	for _, part := range msg.Parts {
		if text := part.GetText(); text != "" {
			return text
		}
	}
	return ""
}

func ExtractAllTextFromMessage(msg *pb.Message) string {
	if msg == nil || len(msg.Parts) == 0 {
		return ""
	}
	var result string
	for _, part := range msg.Parts {
		if text := part.GetText(); text != "" {
			result += text
		}
	}
	return result
}

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

func HasToolCalls(msg *pb.Message) bool {
	if msg == nil {
		return false
	}
	for _, part := range msg.Parts {
		if IsToolCallPart(part) {
			return true
		}
	}
	return false
}

func HasToolResults(msg *pb.Message) bool {
	if msg == nil {
		return false
	}
	for _, part := range msg.Parts {
		if IsToolResultPart(part) {
			return true
		}
	}
	return false
}
