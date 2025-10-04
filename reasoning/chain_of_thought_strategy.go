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

	// Rely on LLM to explicitly call todo_write with merge=true for updates

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

	// Tool access through service interface not yet implemented
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
// ChainOfThought provides a comprehensive, production-ready default blueprint
// This is the BEST AVAILABLE general-purpose prompt - users can override via config
func (s *ChainOfThoughtStrategy) GetPromptSlots() PromptSlots {
	return PromptSlots{
		SystemRole: `You are a helpful AI assistant.
You help users with a wide range of tasks including problem-solving, research, planning, and execution.`,

		ReasoningInstructions: `Your main goal is to follow the user's instructions carefully and provide helpful responses.
By default, IMPLEMENT actions rather than only suggesting them.
Use tools to discover information - don't ask the user to run commands.
Be THOROUGH when gathering information. Make sure you have the FULL picture before responding.
Break down complex problems into manageable steps.`,

		ToolUsage: `Available tools (Safe Mode - Tier 1):
- execute_command: Run read-only system commands (ls, cat, grep, find, etc.)
- todo_write: Manage task lists for complex workflows

Note: File editing tools (file_writer, search_replace) are not enabled by default for security.
To enable them, use: hector coding (Developer Mode configuration)

Tool usage guidelines:
- Use tools proactively when they help accomplish the user's goal
- NO asking for clarification if you can infer the intent
- DO execute tools immediately when appropriate
- DO explain results after execution
- Create todos for complex tasks (3+ steps) to track progress
- For file operations, suggest manual steps or guide user to enable Developer Mode`,

		OutputFormat: `Provide clear, well-structured, and informative responses.
Use markdown formatting for better readability (code blocks, lists, headers, etc.).
Include examples when helpful.
Be direct and concise.`,

		CommunicationStyle: `Be friendly, helpful, and professional.
Use backticks to format file, directory, function, and class names.
Use markdown code blocks for code snippets.
Adapt your tone to match the user's needs.
Generally refrain from using emojis unless extremely informative.`,

		Additional: `<task_management>
For multi-step tasks (3+ steps):
1. Use todo_write (merge=false) to create initial task breakdown
2. Update todos as you progress (merge=true)
3. Mark tasks complete (status="completed") after finishing
4. Add new tasks if needed (merge=true)

Example flow for "Create a web server with tests":
Step 1: todo_write([{id:1, content:"Create server file", status:"in_progress"}, ...])
Step 2: Work on task, call file_writer/search_replace
Step 3: todo_write([{id:1, status:"completed"}], merge=true)
Step 4: Move to next task

This helps track progress and ensures nothing is missed.
</task_management>

General guidelines:
- Provide accurate, factual information
- Never generate extremely long hashes or binary code
- Admit when uncertain about something
- Offer to clarify or expand on topics
- Fix any errors you introduce
- Be respectful and inclusive in all communications`,
	}
}
