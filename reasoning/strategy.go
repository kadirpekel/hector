package reasoning

import (
	"github.com/kadirpekel/hector/llms"
)

// ============================================================================
// REASONING STRATEGY INTERFACE
// This is what varies between different reasoning approaches
// ============================================================================

// ReasoningStrategy defines how different reasoning engines behave
// The core agent handles the function calling protocol (adding messages to conversation)
// Strategies define ADDITIONAL processing (reflection, goal tracking, etc.)
type ReasoningStrategy interface {
	// PrepareIteration is called before each iteration
	// Strategy can initialize state, update prompts, etc.
	PrepareIteration(iteration int, state *ReasoningState) error

	// ShouldStop determines if reasoning should stop
	// Different strategies have different stopping conditions
	ShouldStop(text string, toolCalls []llms.ToolCall, state *ReasoningState) bool

	// AfterIteration is called after each iteration completes (OPTIONAL)
	// Use for meta-cognition, reflection, goal tracking, etc.
	// The core protocol (adding assistant + tool messages) is already handled by Agent
	// This is for ADDITIONAL strategy-specific processing
	AfterIteration(iteration int, text string, toolCalls []llms.ToolCall, results []ToolResult, state *ReasoningState) error

	// GetContextInjection returns additional context to inject into LLM prompt
	// Strategy-specific: ChainOfThought injects todos, StructuredReasoning might inject goals
	// Returns empty string if no additional context needed
	GetContextInjection(state *ReasoningState) string

	// GetPromptSlots returns the strategy's prompt slot values
	// Strategies populate the predefined slots with values appropriate for their reasoning approach
	// Agent merges these with user config, PromptService renders them
	GetPromptSlots() PromptSlots

	// GetName returns the strategy name
	GetName() string

	// GetDescription returns a human-readable description
	GetDescription() string
}

// ToolResult represents the result of executing a tool
type ToolResult struct {
	ToolCall   llms.ToolCall
	Content    string
	Error      error
	ToolCallID string
	ToolName   string
}
