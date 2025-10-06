package reasoning

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/tools"
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

	// Display todos to user if thinking is enabled
	if state.ShowThinking && state.OutputChannel != nil {
		s.displayTodos(todos, state.OutputChannel)
	}

	// Format todos for LLM context
	return tools.FormatTodosForContext(todos)
}

// displayTodos shows the current todo list to the user
func (s *ChainOfThoughtStrategy) displayTodos(todos []tools.TodoItem, outputCh chan<- string) {
	outputCh <- "\n\033[90m📋 **Current Tasks:**\n"
	for i, todo := range todos {
		var status string
		switch todo.Status {
		case "pending":
			status = "⏳"
		case "in_progress":
			status = "🔄"
		case "completed":
			status = "✅"
		case "cancelled":
			status = "❌"
		default:
			status = "○"
		}
		outputCh <- fmt.Sprintf("  %d. %s %s\n", i+1, status, todo.Content)
	}
	outputCh <- "\033[0m\n"
}

// getTodoTool retrieves the todo_write tool (guaranteed to exist due to GetRequiredTools)
func (s *ChainOfThoughtStrategy) getTodoTool(state *ReasoningState) *tools.TodoTool {
	if state.Services == nil || state.Services.Tools() == nil {
		return nil
	}

	// Get the todo tool from the tool service
	tool, err := state.Services.Tools().GetTool("todo_write")
	if err != nil {
		return nil
	}

	// Type assert to TodoTool
	todoTool, ok := tool.(*tools.TodoTool)
	if !ok {
		return nil
	}

	return todoTool
}

// GetName implements ReasoningStrategy
func (s *ChainOfThoughtStrategy) GetName() string {
	return "Chain-of-Thought"
}

// GetDescription implements ReasoningStrategy
func (s *ChainOfThoughtStrategy) GetDescription() string {
	return "Iterative reasoning with native LLM function calling (OpenAI, Anthropic)"
}

// GetRequiredTools implements ReasoningStrategy
// ChainOfThought requires todo_write for systematic task management
func (s *ChainOfThoughtStrategy) GetRequiredTools() []RequiredTool {
	return []RequiredTool{
		{
			Name:        "todo_write",
			Type:        "todo",
			Description: "Required for systematic task tracking in complex multi-step workflows",
			AutoCreate:  true, // Always create if not configured
		},
	}
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

Be THOROUGH when gathering information. Make sure you have the FULL picture before replying.
TRACE every symbol back to its definitions and usages so you fully understand it.
Look past the first seemingly relevant result. EXPLORE alternative implementations, edge cases, and varied search terms until you have COMPREHENSIVE coverage of the topic.

If you've performed an action that may partially fulfill the user's query, but you're not confident, gather more information or use more tools before ending your turn.

Bias towards not asking the user for help if you can find the answer yourself.
Break down complex problems into manageable steps.`,

		ToolUsage: `Tool usage guidelines:
- Use tools proactively when they help accomplish the user's goal
- If you can infer the user's intent, proceed immediately without asking for clarification
- Execute tools when appropriate and explain results after execution
- Create todos for complex tasks (3+ steps) to track progress

<maximize_parallel_tool_calls>
If you intend to call multiple tools and there are no dependencies between the tool calls, make all of the independent tool calls in parallel.

Prioritize calling tools simultaneously whenever the actions can be done in parallel rather than sequentially. For example, when reading 3 files, run 3 tool calls in parallel to read all 3 files into context at the same time.

Maximize use of parallel tool calls where possible to increase speed and efficiency.

However, if some tool calls depend on previous calls to inform dependent values like the parameters, do NOT call these tools in parallel and instead call them sequentially. Never use placeholders or guess missing parameters in tool calls.
</maximize_parallel_tool_calls>

<tool_selection>
Use specialized tools instead of terminal commands when possible, as this provides a better user experience.

For file operations, use dedicated tools:
- Don't use cat/head/tail to read files (use file reading tools)
- Don't use sed/awk to edit files (use file editing tools)
- Don't use cat with heredoc or echo redirection to create files (use file writing tools)

Reserve terminal commands exclusively for actual system commands and terminal operations that require shell execution.

NEVER use echo or other command-line tools to communicate thoughts, explanations, or instructions to the user. Output all communication directly in your response text instead.
</tool_selection>

<semantic_search>
When search tools are available, use them as your MAIN exploration tool:
- CRITICAL: Start with a broad, high-level query that captures overall intent (e.g. "authentication flow" or "error-handling policy"), not low-level terms
- Break multi-part questions into focused sub-queries (e.g. "How does authentication work?" or "Where is payment processed?")
- MANDATORY: Run multiple searches with different wording; first-pass results often miss key details
- Keep searching new areas until you're CONFIDENT nothing important remains
</semantic_search>

Note: Your available tools will be provided separately via the function calling API.
Use them naturally to solve the user's problems.`,

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
Step 2: Work on task, call appropriate tools
Step 3: todo_write([{id:1, status:"completed"}], merge=true)
Step 4: Move to next task

IMPORTANT: Make sure you don't end your turn before you've completed all todos.
This helps track progress and ensures nothing is missed.
</task_management>

<parameter_handling>
Check that all the required parameters for each tool call are provided or can reasonably be inferred from context.

IF there are no relevant tools or there are missing values for required parameters, ask the user to supply these values; otherwise proceed with the tool calls.

If the user provides a specific value for a parameter (for example provided in quotes), make sure to use that value EXACTLY.
</parameter_handling>

General guidelines:
- Provide accurate, factual information
- Never generate extremely long hashes or binary code
- Admit when uncertain about something
- Offer to clarify or expand on topics
- Fix any errors you introduce
- Be respectful and inclusive in all communications`,
	}
}
