package agent

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// displayToolCall shows the tool call based on the configured display mode
func displayToolCall(outputCh chan<- *pb.Part, toolCall *protocol.ToolCall, cfg config.ReasoningConfig) {
	switch cfg.ToolDisplayMode {
	case "inline":
		// Clean inline display: 🔧 tool_name
		outputCh <- createTextPart(fmt.Sprintf("🔧 %s", toolCall.Name))
	case "detailed":
		// Detailed display with arguments
		label := formatToolLabel(toolCall.Name, toolCall.Args)
		if cfg.ShowToolArgs {
			outputCh <- createTextPart(fmt.Sprintf("[Tool: %s]", label))
		} else {
			outputCh <- createTextPart(fmt.Sprintf("[Tool: %s]", toolCall.Name))
		}
	case "thinking":
		// Just show working indicator
		outputCh <- createTextPart("⏳ Working...")
	case "hidden":
		// Don't show anything
		return
	default:
		// Fallback to inline
		outputCh <- createTextPart(fmt.Sprintf("🔧 %s", toolCall.Name))
	}
}

// displayToolResult shows the tool execution result based on the configured display mode
func displayToolResult(outputCh chan<- *pb.Part, toolCall *protocol.ToolCall, err error, result string, cfg config.ReasoningConfig) {
	switch cfg.ToolDisplayMode {
	case "inline":
		// Show success/failure indicator
		if err != nil {
			outputCh <- createTextPart(" ✗\n")
		} else {
			outputCh <- createTextPart(" ✓\n")
		}
	case "detailed":
		// Show detailed result
		if err != nil {
			outputCh <- createTextPart(" [FAILED]\n")
			if cfg.ShowToolResults {
				outputCh <- createTextPart(fmt.Sprintf("  Error: %v\n", err))
			}
		} else {
			outputCh <- createTextPart(" [SUCCESS]\n")
			if cfg.ShowToolResults {
				// Truncate long results
				if len(result) > 200 {
					outputCh <- createTextPart(fmt.Sprintf("  Result: %s...\n", result[:197]))
				} else {
					outputCh <- createTextPart(fmt.Sprintf("  Result: %s\n", result))
				}
			}
		}
	case "thinking":
		// Clear the "Working..." message
		outputCh <- createTextPart(" Done.\n")
	case "hidden":
		// Don't show anything
		return
	}
}
