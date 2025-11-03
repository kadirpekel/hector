package display

import (
	pb "github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// Renderer is the interface for rendering different part types across different media
type Renderer interface {
	// Core content rendering
	RenderText(text string) error
	RenderContentBlock(block *pb.ContentBlock) error

	// Flush ensures all output is written
	Flush() error
}

// RenderMessage renders a complete message using the appropriate renderer
func RenderMessage(msg *pb.Message, renderer Renderer) error {
	for _, part := range msg.Parts {
		if err := RenderPart(part, renderer); err != nil {
			return err
		}
	}
	return renderer.Flush()
}

// RenderPart renders a single part using the appropriate renderer
func RenderPart(part *pb.Part, renderer Renderer) error {
	// Handle plain text parts
	if text := part.GetText(); text != "" {
		return renderer.RenderText(text)
	}

	// Handle content blocks
	if protocol.IsContentBlock(part) {
		block, err := protocol.ExtractContentBlock(part)
		if err != nil {
			return err
		}
		return renderer.RenderContentBlock(block)
	}

	// Handle legacy DataPart (for backward compatibility)
	if data := part.GetData(); data != nil {
		// Check if it's a tool call or result (legacy format) via part metadata
		if part.Metadata != nil {
			if partTypeField, ok := part.Metadata.Fields["part_type"]; ok {
				partType := partTypeField.GetStringValue()
				if partType == "tool_call" || partType == "tool_result" {
					// Legacy tool call/result - could be converted to content block
					// For now, skip or handle specially
					return nil
				}
			}
		}
	}

	return nil
}
