package cli

import (
	"fmt"
	"os"

	aguipb "github.com/kadirpekel/hector/pkg/agui/pb"
)

// AGUIHandler handles AG-UI events for CLI display
type AGUIHandler struct {
	showThinking   bool
	verbose        bool
	useColors      bool
	currentBlockID string
}

// NewAGUIHandler creates a new CLI AG-UI event handler
func NewAGUIHandler(showThinking, verbose, useColors bool) *AGUIHandler {
	return &AGUIHandler{
		showThinking: showThinking,
		verbose:      verbose,
		useColors:    useColors,
	}
}

// HandleEvent processes an AG-UI event and formats it for terminal display
func (h *AGUIHandler) HandleEvent(event *aguipb.AGUIEvent) {
	switch event.Type {
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_MESSAGE_START:
		h.handleMessageStart(event.GetMessageStart())
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_MESSAGE_DELTA:
		h.handleMessageDelta(event.GetMessageDelta())
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_MESSAGE_STOP:
		h.handleMessageStop(event.GetMessageStop())

	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_CONTENT_BLOCK_START:
		h.handleContentBlockStart(event.GetContentBlockStart())
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_CONTENT_BLOCK_DELTA:
		h.handleContentBlockDelta(event.GetContentBlockDelta())
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_CONTENT_BLOCK_STOP:
		h.handleContentBlockStop(event.GetContentBlockStop())

	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TOOL_CALL_START:
		h.handleToolCallStart(event.GetToolCallStart())
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TOOL_CALL_DELTA:
		h.handleToolCallDelta(event.GetToolCallDelta())
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TOOL_CALL_STOP:
		h.handleToolCallStop(event.GetToolCallStop())

	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_THINKING_START:
		h.handleThinkingStart(event.GetThinkingStart())
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_THINKING_DELTA:
		h.handleThinkingDelta(event.GetThinkingDelta())
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_THINKING_STOP:
		h.handleThinkingStop(event.GetThinkingStop())

	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TASK_START:
		h.handleTaskStart(event.GetTaskStart())
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TASK_UPDATE:
		h.handleTaskUpdate(event.GetTaskUpdate())
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TASK_COMPLETE:
		h.handleTaskComplete(event.GetTaskComplete())
	case aguipb.AGUIEventType_AGUI_EVENT_TYPE_TASK_ERROR:
		h.handleTaskError(event.GetTaskError())
	}
}

// ============================================================================
// Message Event Handlers
// ============================================================================

func (h *AGUIHandler) handleMessageStart(payload *aguipb.MessageStartPayload) {
	// Message start is typically silent in CLI
	if h.verbose {
		fmt.Printf("\n[Message %s started]\n", payload.MessageId)
	}
}

func (h *AGUIHandler) handleMessageDelta(payload *aguipb.MessageDeltaPayload) {
	// Print delta directly
	fmt.Print(payload.Delta)
	os.Stdout.Sync()
}

func (h *AGUIHandler) handleMessageStop(payload *aguipb.MessageStopPayload) {
	// Message stop is typically silent in CLI
	if h.verbose {
		fmt.Printf("\n[Message %s stopped]\n", payload.MessageId)
	}
}

// ============================================================================
// Content Block Event Handlers
// ============================================================================

func (h *AGUIHandler) handleContentBlockStart(payload *aguipb.ContentBlockStartPayload) {
	h.currentBlockID = payload.BlockId
	// Content block start is typically silent unless verbose
	if h.verbose {
		fmt.Printf("\n[Content block %s (%s) started]\n", payload.BlockId, payload.BlockType)
	}
}

func (h *AGUIHandler) handleContentBlockDelta(payload *aguipb.ContentBlockDeltaPayload) {
	// Print content delta directly
	fmt.Print(payload.Delta)
	os.Stdout.Sync()
}

func (h *AGUIHandler) handleContentBlockStop(payload *aguipb.ContentBlockStopPayload) {
	h.currentBlockID = ""
	// Content block stop is typically silent
}

// ============================================================================
// Tool Call Event Handlers
// ============================================================================

func (h *AGUIHandler) handleToolCallStart(payload *aguipb.ToolCallStartPayload) {
	// Format: 🔧 tool_name
	fmt.Printf("🔧 %s ", payload.ToolName)
	os.Stdout.Sync()
}

func (h *AGUIHandler) handleToolCallDelta(payload *aguipb.ToolCallDeltaPayload) {
	// Tool call delta is typically not shown in CLI
	// Could show "..." or progress indicator
	if h.verbose {
		fmt.Print(".")
		os.Stdout.Sync()
	}
}

func (h *AGUIHandler) handleToolCallStop(payload *aguipb.ToolCallStopPayload) {
	// Format: ✓ or ✗
	if payload.IsError {
		if h.useColors {
			fmt.Print("\033[31m✗\033[0m\n") // Red X
		} else {
			fmt.Print("✗\n")
		}
	} else {
		if h.useColors {
			fmt.Print("\033[32m✓\033[0m\n") // Green checkmark
		} else {
			fmt.Print("✓\n")
		}
	}

	// Show error if present and verbose
	if payload.IsError && h.verbose && payload.Error != "" {
		fmt.Printf("  Error: %s\n", payload.Error)
	}

	os.Stdout.Sync()
}

// ============================================================================
// Thinking Event Handlers
// ============================================================================

func (h *AGUIHandler) handleThinkingStart(payload *aguipb.ThinkingStartPayload) {
	if h.showThinking {
		title := payload.Title
		if title == "" {
			title = "Thinking"
		}
		if h.useColors {
			fmt.Printf("\033[90m💭 %s...\033[0m\n", title) // Gray
		} else {
			fmt.Printf("💭 %s...\n", title)
		}
		os.Stdout.Sync()
	}
}

func (h *AGUIHandler) handleThinkingDelta(payload *aguipb.ThinkingDeltaPayload) {
	if h.showThinking {
		if h.useColors {
			fmt.Printf("\033[90m%s\033[0m", payload.Delta) // Gray
		} else {
			fmt.Print(payload.Delta)
		}
		os.Stdout.Sync()
	}
}

func (h *AGUIHandler) handleThinkingStop(payload *aguipb.ThinkingStopPayload) {
	if h.showThinking {
		if h.useColors {
			fmt.Print("\033[90m [Done]\033[0m\n") // Gray
		} else {
			fmt.Print(" [Done]\n")
		}
		os.Stdout.Sync()
	}
}

// ============================================================================
// Task Event Handlers
// ============================================================================

func (h *AGUIHandler) handleTaskStart(payload *aguipb.TaskStartPayload) {
	if h.verbose {
		fmt.Printf("\n[Task %s started]\n", payload.TaskId)
		if payload.Description != "" {
			fmt.Printf("Description: %s\n", payload.Description)
		}
	}
}

func (h *AGUIHandler) handleTaskUpdate(payload *aguipb.TaskUpdatePayload) {
	if h.verbose {
		fmt.Printf("[Task %s: %s]\n", payload.TaskId, payload.Status)
	}
}

func (h *AGUIHandler) handleTaskComplete(payload *aguipb.TaskCompletePayload) {
	if h.verbose {
		if h.useColors {
			fmt.Printf("\033[32m[Task %s completed]\033[0m\n", payload.TaskId) // Green
		} else {
			fmt.Printf("[Task %s completed]\n", payload.TaskId)
		}
	}
}

func (h *AGUIHandler) handleTaskError(payload *aguipb.TaskErrorPayload) {
	if h.useColors {
		fmt.Printf("\033[31m[Task %s error: %s]\033[0m\n", payload.TaskId, payload.ErrorMessage) // Red
	} else {
		fmt.Printf("[Task %s error: %s]\n", payload.TaskId, payload.ErrorMessage)
	}
}
