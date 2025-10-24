package reasoning

import (
	"github.com/kadirpekel/hector/pkg/protocol"
)

type ReasoningStrategy interface {
	PrepareIteration(iteration int, state *ReasoningState) error

	ShouldStop(text string, toolCalls []*protocol.ToolCall, state *ReasoningState) bool

	AfterIteration(iteration int, text string, toolCalls []*protocol.ToolCall, results []ToolResult, state *ReasoningState) error

	GetContextInjection(state *ReasoningState) string

	GetPromptSlots() PromptSlots

	GetRequiredTools() []RequiredTool

	GetName() string

	GetDescription() string
}

type RequiredTool struct {
	Name        string
	Type        string
	Description string
	AutoCreate  bool
}

type ToolResult struct {
	ToolCall   *protocol.ToolCall
	Content    string
	Error      error
	ToolCallID string
	ToolName   string
	Metadata   map[string]interface{}
}
