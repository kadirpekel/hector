package agent

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/protocol"
)

// displayToolCall shows the tool call based on the configured display mode
func displayToolCall(outputCh chan<- string, toolCall *protocol.ToolCall, cfg config.ReasoningConfig) {
	switch cfg.ToolDisplayMode {
	case "inline":
		// Clean inline display: 🔧 tool_name
		outputCh <- fmt.Sprintf("🔧 %s", toolCall.Name)
	case "detailed":
		// Detailed display with arguments
		label := formatToolLabel(toolCall.Name, toolCall.Args)
		if cfg.ShowToolArgs {
			outputCh <- fmt.Sprintf("[Tool: %s]", label)
		} else {
			outputCh <- fmt.Sprintf("[Tool: %s]", toolCall.Name)
		}
	case "thinking":
		// Just show working indicator
		outputCh <- "⏳ Working..."
	case "hidden":
		// Don't show anything
		return
	default:
		// Fallback to inline
		outputCh <- fmt.Sprintf("🔧 %s", toolCall.Name)
	}
}

// displayToolResult shows the tool execution result based on the configured display mode
func displayToolResult(outputCh chan<- string, toolCall *protocol.ToolCall, err error, result string, cfg config.ReasoningConfig) {
	switch cfg.ToolDisplayMode {
	case "inline":
		// Show success/failure indicator
		if err != nil {
			outputCh <- " ✗\n"
		} else {
			outputCh <- " ✓\n"
		}
	case "detailed":
		// Show detailed result
		if err != nil {
			outputCh <- " [FAILED]\n"
			if cfg.ShowToolResults {
				outputCh <- fmt.Sprintf("  Error: %v\n", err)
			}
		} else {
			outputCh <- " [SUCCESS]\n"
			if cfg.ShowToolResults {
				// Truncate long results
				if len(result) > 200 {
					outputCh <- fmt.Sprintf("  Result: %s...\n", result[:197])
				} else {
					outputCh <- fmt.Sprintf("  Result: %s\n", result)
				}
			}
		}
	case "thinking":
		// Clear the "Working..." message
		outputCh <- " Done.\n"
	case "hidden":
		// Don't show anything
		return
	}
}
