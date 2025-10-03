package reasoning

import (
	"fmt"

	"github.com/kadirpekel/hector/llms"
	"github.com/kadirpekel/hector/tools"
)

// ============================================================================
// CHAIN-OF-THOUGHT STRATEGY
// Simple iterative reasoning - stops when no tool calls remain
// Includes systematic todo tracking for complex multi-step tasks
// ============================================================================

// ChainOfThoughtStrategy implements the chain-of-thought reasoning strategy
// This is the simplest strategy: iterate until no more tool calls
type ChainOfThoughtStrategy struct{}

// NewChainOfThoughtStrategy creates a new chain-of-thought strategy
func NewChainOfThoughtStrategy() *ChainOfThoughtStrategy {
	return &ChainOfThoughtStrategy{}
}

// PrepareIteration implements ReasoningStrategy
// Chain-of-thought doesn't need any special preparation
func (s *ChainOfThoughtStrategy) PrepareIteration(iteration int, state *ReasoningState) error {
	return nil
}

// ShouldStop implements ReasoningStrategy
// Stop when no tool calls are returned
func (s *ChainOfThoughtStrategy) ShouldStop(text string, toolCalls []llms.ToolCall, state *ReasoningState) bool {
	return len(toolCalls) == 0
}

// AfterIteration implements ReasoningStrategy with self-reflection
// Evaluates progress, updates todos, and assesses approach effectiveness
func (s *ChainOfThoughtStrategy) AfterIteration(
	iteration int,
	text string,
	toolCalls []llms.ToolCall,
	results []ToolResult,
	state *ReasoningState,
) error {
	// Self-reflection: Evaluate progress made in this iteration
	if state.ShowDebugInfo && state.OutputChannel != nil {
		s.reflectOnProgress(iteration, text, toolCalls, results, state)
	}

	// Auto-update todos based on tool results
	// TODO: Implement automatic todo updates based on completion signals
	// For now, rely on LLM to explicitly call todo_write with merge=true

	return nil
}

// reflectOnProgress performs self-reflection and outputs thinking blocks
func (s *ChainOfThoughtStrategy) reflectOnProgress(
	iteration int,
	text string,
	toolCalls []llms.ToolCall,
	results []ToolResult,
	state *ReasoningState,
) {
	// Count tool calls (assuming all executed successfully if no error in results)
	// Note: ToolResult doesn't have a Success field, so we count all executed tools
	successCount := len(results)
	failCount := 0

	// Check if any results indicate errors (would be in the content/error field)
	// For now, assume all executed tools were successful unless we detect error patterns
	for _, result := range results {
		// Simple heuristic: if content contains "Error:" or "failed", count as failure
		if len(result.Content) > 0 && (contains(result.Content, "Error:") || contains(result.Content, "failed")) {
			failCount++
			successCount--
		}
	}

	// Output reflection thinking block (in gray)
	if len(toolCalls) > 0 {
		// ANSI gray color code: \033[90m ... \033[0m
		output := "\033[90m\n💭 **Self-Reflection:**\n"
		output += "  - Tools executed: " + formatToolList(toolCalls) + "\n"
		output += "  - Success/Fail: " + formatSuccessRatio(successCount, failCount) + "\n"

		// Evaluate approach effectiveness
		if failCount > 0 {
			output += "  - ⚠️  Some tools failed - may need to pivot approach\n"
		} else if successCount > 0 {
			output += "  - ✅ All tools succeeded - making progress\n"
		}

		// Check if we're making forward progress
		if iteration > 5 && len(toolCalls) > 0 {
			output += fmt.Sprintf("  - 📊 Iteration %d - consider simplifying approach\n", iteration)
		}

		output += "\033[0m" // Reset color
		state.OutputChannel <- output
	}
}

// Helper functions for formatting reflection output
func formatToolList(toolCalls []llms.ToolCall) string {
	if len(toolCalls) == 0 {
		return "none"
	}
	if len(toolCalls) == 1 {
		return toolCalls[0].Name
	}
	return fmt.Sprintf("%d tools", len(toolCalls))
}

func formatSuccessRatio(success, fail int) string {
	total := success + fail
	if total == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%d/%d", success, total)
}

// contains checks if a string contains a substring (simple helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GetContextInjection implements ReasoningStrategy
// ChainOfThought injects current todos to enable systematic task tracking
func (s *ChainOfThoughtStrategy) GetContextInjection(state *ReasoningState) string {
	// Access todo tool from services
	if state.Services == nil {
		return ""
	}

	// Get the todo tool instance
	todoTool := s.getTodoTool(state)
	if todoTool == nil {
		return ""
	}

	// Get todos for the default session (single-agent mode)
	sessionID := "default"
	todos := todoTool.GetTodos(sessionID)

	if len(todos) == 0 {
		return ""
	}

	// Format todos for LLM context
	return tools.FormatTodosForContext(todos)
}

// getTodoTool retrieves the todo_write tool if available
func (s *ChainOfThoughtStrategy) getTodoTool(state *ReasoningState) *tools.TodoTool {
	if state.Services == nil || state.Services.Tools() == nil {
		return nil
	}

	// Try to execute a dummy call to check if tool exists and is accessible
	// This is a workaround since ToolService doesn't expose tool instances directly
	// In practice, we just need to access the tool's GetTodos method
	// For now, return nil - the actual implementation will need access to the registry
	// TODO: Add GetToolInstance to ToolService interface or pass registry to state
	return nil
}

// GetName implements ReasoningStrategy
func (s *ChainOfThoughtStrategy) GetName() string {
	return "Chain-of-Thought"
}

// GetDescription implements ReasoningStrategy
func (s *ChainOfThoughtStrategy) GetDescription() string {
	return "Iterative reasoning with native LLM function calling (OpenAI, Anthropic)"
}

// GetPromptSlots implements ReasoningStrategy
// ChainOfThought provides flexible, general-purpose slot values
// Users can override any slot for their specific use case
func (s *ChainOfThoughtStrategy) GetPromptSlots() PromptSlots {
	return PromptSlots{
		SystemRole: "You are an AI assistant.",

		ReasoningInstructions: `Your main goal is to follow the user's instructions carefully.
Use available tools to accomplish tasks step by step.
Be thorough in gathering necessary information before responding.`,

		ToolUsage: `You have tools at your disposal to solve tasks:
- Use tools naturally when they help accomplish the task
- Take concrete actions rather than only suggesting them
- Explore all necessary context using available tools
- Clean up any temporary resources after completing tasks`,

		OutputFormat: `Provide clear, accurate, and complete responses.
Be direct and concise in your communication.`,

		CommunicationStyle: `Use professional language appropriate for the task.
Use markdown formatting for better readability where appropriate.`,

		Additional: "", // Users can add domain-specific instructions via config
	}
}
