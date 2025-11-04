package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	pb "github.com/kadirpekel/hector/pkg/a2a/pb"
	"google.golang.org/protobuf/types/known/structpb"
)

// CreateThinkingBlock creates a thinking content block Part.
// This follows the Anthropic Claude extended thinking pattern.
func CreateThinkingBlock(content string, metadata *pb.ThinkingMetadata) (*pb.Part, error) {
	block := &pb.ContentBlock{
		Id:   uuid.New().String(),
		Type: pb.ContentBlockType_CONTENT_BLOCK_TYPE_THINKING,
		Block: &pb.ContentBlock_ThinkingBlock{
			ThinkingBlock: &pb.ThinkingBlock{
				Thinking: content,
				Metadata: metadata,
			},
		},
	}

	return contentBlockToPart(block)
}

// CreateReasoningBlock creates a reasoning content block Part.
// This is an alias for thinking blocks using Vercel AI SDK terminology.
func CreateReasoningBlock(content string, metadata *pb.ThinkingMetadata) (*pb.Part, error) {
	block := &pb.ContentBlock{
		Id:   uuid.New().String(),
		Type: pb.ContentBlockType_CONTENT_BLOCK_TYPE_REASONING,
		Block: &pb.ContentBlock_ReasoningBlock{
			ReasoningBlock: &pb.ReasoningBlock{
				Reasoning: content,
				Metadata:  metadata,
			},
		},
	}

	return contentBlockToPart(block)
}

// CreateTextBlock creates a text content block Part
func CreateTextBlock(text string) (*pb.Part, error) {
	block := &pb.ContentBlock{
		Id:   uuid.New().String(),
		Type: pb.ContentBlockType_CONTENT_BLOCK_TYPE_TEXT,
		Block: &pb.ContentBlock_TextBlock{
			TextBlock: &pb.TextBlock{
				Text: text,
			},
		},
	}

	return contentBlockToPart(block)
}

// CreateToolCallContentBlock creates a tool call content block Part
func CreateToolCallContentBlock(id, name, input string) (*pb.Part, error) {
	block := &pb.ContentBlock{
		Id:   id,
		Type: pb.ContentBlockType_CONTENT_BLOCK_TYPE_TOOL_CALL,
		Block: &pb.ContentBlock_ToolCallBlock{
			ToolCallBlock: &pb.ToolCallBlock{
				Id:    id,
				Name:  name,
				Input: input,
			},
		},
	}

	return contentBlockToPart(block)
}

// CreateToolResultContentBlock creates a tool result content block Part
func CreateToolResultContentBlock(toolCallID, content string, isError bool) (*pb.Part, error) {
	block := &pb.ContentBlock{
		Id:   uuid.New().String(),
		Type: pb.ContentBlockType_CONTENT_BLOCK_TYPE_TOOL_RESULT,
		Block: &pb.ContentBlock_ToolResultBlock{
			ToolResultBlock: &pb.ToolResultBlock{
				ToolCallId: toolCallID,
				Content:    content,
				IsError:    isError,
			},
		},
	}

	return contentBlockToPart(block)
}

// contentBlockToPart converts a ContentBlock to a Part with proper metadata
func contentBlockToPart(block *pb.ContentBlock) (*pb.Part, error) {
	// Marshal the content block to JSON
	blockJSON, err := json.Marshal(block)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal content block: %w", err)
	}

	// Parse JSON into Struct
	var data map[string]interface{}
	if err := json.Unmarshal(blockJSON, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content block JSON: %w", err)
	}

	structData, err := structpb.NewStruct(data)
	if err != nil {
		return nil, fmt.Errorf("failed to create structpb: %w", err)
	}

	// Create metadata indicating this is a content block
	metadata, err := structpb.NewStruct(map[string]interface{}{
		"content_block": true,
		"block_type":    block.Type.String(),
		"block_id":      block.Id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata: %w", err)
	}

	return &pb.Part{
		Part: &pb.Part_Data{
			Data: &pb.DataPart{
				Data: structData,
			},
		},
		Metadata: metadata,
	}, nil
}

// IsContentBlock checks if a Part contains a content block
func IsContentBlock(part *pb.Part) bool {
	if part.Metadata == nil {
		return false
	}
	if cb, ok := part.Metadata.Fields["content_block"]; ok {
		return cb.GetBoolValue()
	}
	return false
}

// ExtractContentBlock extracts a ContentBlock from a Part
func ExtractContentBlock(part *pb.Part) (*pb.ContentBlock, error) {
	if !IsContentBlock(part) {
		return nil, fmt.Errorf("part is not a content block")
	}

	dataPart := part.GetData()
	if dataPart == nil {
		return nil, fmt.Errorf("part does not contain data")
	}

	// Marshal struct back to JSON
	jsonBytes, err := dataPart.Data.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data part: %w", err)
	}

	// Unmarshal into ContentBlock
	var block pb.ContentBlock
	if err := json.Unmarshal(jsonBytes, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content block: %w", err)
	}

	return &block, nil
}

// IsThinkingBlock checks if a Part is a thinking/reasoning block
func IsThinkingBlock(part *pb.Part) bool {
	if !IsContentBlock(part) {
		return false
	}

	block, err := ExtractContentBlock(part)
	if err != nil {
		return false
	}

	return block.Type == pb.ContentBlockType_CONTENT_BLOCK_TYPE_THINKING ||
		block.Type == pb.ContentBlockType_CONTENT_BLOCK_TYPE_REASONING
}

// GetThinkingContent extracts thinking content from a thinking block
func GetThinkingContent(part *pb.Part) (string, *pb.ThinkingMetadata, error) {
	block, err := ExtractContentBlock(part)
	if err != nil {
		return "", nil, err
	}

	switch block.Type {
	case pb.ContentBlockType_CONTENT_BLOCK_TYPE_THINKING:
		tb := block.GetThinkingBlock()
		if tb == nil {
			return "", nil, fmt.Errorf("thinking block is nil")
		}
		return tb.Thinking, tb.Metadata, nil

	case pb.ContentBlockType_CONTENT_BLOCK_TYPE_REASONING:
		rb := block.GetReasoningBlock()
		if rb == nil {
			return "", nil, fmt.Errorf("reasoning block is nil")
		}
		return rb.Reasoning, rb.Metadata, nil

	default:
		return "", nil, fmt.Errorf("not a thinking/reasoning block: %s", block.Type)
	}
}

// Convenience functions for common thinking types

// CreateReflectionBlock creates a reflection thinking block
func CreateReflectionBlock(content, title string) (*pb.Part, error) {
	return CreateThinkingBlock(content, &pb.ThinkingMetadata{
		Title:       title,
		Collapsible: true,
		Priority:    1,
		Format:      "markdown",
		Tags:        []string{"reflection"},
	})
}

// CreatePlanningBlock creates a planning thinking block (for todos, goals)
func CreatePlanningBlock(content, title string) (*pb.Part, error) {
	return CreateThinkingBlock(content, &pb.ThinkingMetadata{
		Title:       title,
		Collapsible: false,
		Priority:    2,
		Format:      "markdown",
		Tags:        []string{"planning", "goals"},
	})
}

// CreateProgressBlock creates a progress update thinking block
func CreateProgressBlock(content string) (*pb.Part, error) {
	return CreateThinkingBlock(content, &pb.ThinkingMetadata{
		Ephemeral: true,
		Priority:  0,
		Format:    "plain",
		Tags:      []string{"progress"},
	})
}

// CreateDebugBlock creates a debug information thinking block
func CreateDebugBlock(content, title string) (*pb.Part, error) {
	return CreateThinkingBlock(content, &pb.ThinkingMetadata{
		Title:       title,
		Collapsible: true,
		Priority:    0,
		Ephemeral:   true,
		Format:      "plain",
		Tags:        []string{"debug"},
	})
}

// CreateAnalysisBlock creates an analysis thinking block
func CreateAnalysisBlock(content, title string) (*pb.Part, error) {
	return CreateThinkingBlock(content, &pb.ThinkingMetadata{
		Title:       title,
		Collapsible: true,
		Priority:    1,
		Format:      "markdown",
		Tags:        []string{"analysis"},
	})
}
