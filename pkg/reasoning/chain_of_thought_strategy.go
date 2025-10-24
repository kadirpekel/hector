package reasoning

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/tools"
)

type ChainOfThoughtStrategy struct{}

func NewChainOfThoughtStrategy() *ChainOfThoughtStrategy {
	return &ChainOfThoughtStrategy{}
}

func (s *ChainOfThoughtStrategy) PrepareIteration(iteration int, state *ReasoningState) error {
	return nil
}

func (s *ChainOfThoughtStrategy) ShouldStop(text string, toolCalls []*protocol.ToolCall, state *ReasoningState) bool {

	if len(toolCalls) == 0 {
		return true
	}

	todoTool := s.getTodoTool(state)
	if todoTool != nil {
		todos := todoTool.GetTodos("default")
		allComplete := len(todos) > 0 && s.allTodosComplete(todos)

		todosWereCompleteLast := false
		if val, ok := state.GetToolState()["todos_complete"]; ok {
			todosWereCompleteLast, _ = val.(bool)
		}

		if todosWereCompleteLast && allComplete {
			return true
		}

		state.GetToolState()["todos_complete"] = allComplete
	}

	return false
}

func (s *ChainOfThoughtStrategy) allTodosComplete(todos []tools.TodoItem) bool {
	for _, todo := range todos {
		if todo.Status != "completed" && todo.Status != "canceled" {
			return false
		}
	}
	return true
}

func (s *ChainOfThoughtStrategy) AfterIteration(
	iteration int,
	text string,
	toolCalls []*protocol.ToolCall,
	results []ToolResult,
	state *ReasoningState,
) error {

	if state.ShowThinking() && state.GetOutputChannel() != nil {

		useStructuredReflection := false
		if state.GetServices() != nil {
			cfg := state.GetServices().GetConfig()
			useStructuredReflection = cfg.EnableStructuredReflection != nil && *cfg.EnableStructuredReflection
		}

		if useStructuredReflection && state.GetServices() != nil && state.GetContext() != nil {
			analysis, err := AnalyzeToolResults(state.GetContext(), toolCalls, results, state.GetServices())
			if err == nil {
				s.displayStructuredReflection(iteration, analysis, state)

				state.GetCustomState()["reflection_analysis"] = analysis
			} else {

				s.reflectOnProgress(iteration, text, toolCalls, results, state)
			}
		} else {

			s.reflectOnProgress(iteration, text, toolCalls, results, state)
		}
	}

	return nil
}

func (s *ChainOfThoughtStrategy) displayStructuredReflection(
	iteration int,
	analysis *ReflectionAnalysis,
	state *ReasoningState,
) {
	if len(analysis.SuccessfulTools) == 0 && len(analysis.FailedTools) == 0 {
		return
	}

	output := ThinkingBlock(fmt.Sprintf("Iteration %d: Analyzing results", iteration))

	if len(analysis.SuccessfulTools) > 0 {
		output += ThinkingBlock(fmt.Sprintf("[SUCCESS] Tools: %s", formatStringList(analysis.SuccessfulTools)))
	}
	if len(analysis.FailedTools) > 0 {
		output += ThinkingBlock(fmt.Sprintf("[FAILED] Tools: %s", formatStringList(analysis.FailedTools)))
	}

	var recommendation string
	if analysis.ShouldPivot {
		recommendation = "Pivot approach"
	} else {
		switch analysis.Recommendation {
		case "retry_failed":
			recommendation = "Retry failed tools"
		case "pivot_approach":
			recommendation = "Change approach"
		case "stop":
			recommendation = "Stop (task may be infeasible)"
		default:
			recommendation = "Continue"
		}
	}

	output += ThinkingBlock(fmt.Sprintf("Confidence: %.0f%% - %s", analysis.Confidence*100, recommendation))

	state.GetOutputChannel() <- output
}

func (s *ChainOfThoughtStrategy) reflectOnProgress(
	iteration int,
	text string,
	toolCalls []*protocol.ToolCall,
	results []ToolResult,
	state *ReasoningState,
) {

	successCount := len(results)
	failCount := 0

	for _, result := range results {

		if len(result.Content) > 0 && (contains(result.Content, "Error:") || contains(result.Content, "failed")) {
			failCount++
			successCount--
		}
	}

	if len(toolCalls) > 0 {
		output := ThinkingBlock(fmt.Sprintf("Iteration %d: Evaluating progress", iteration))
		output += ThinkingBlock(fmt.Sprintf("Tools executed: %s", formatToolList(toolCalls)))
		output += ThinkingBlock(fmt.Sprintf("Success/Fail: %s", formatSuccessRatio(successCount, failCount)))

		if failCount > 0 {
			output += ThinkingBlock("Warning: Some tools failed - may need to pivot approach")
		} else if successCount > 0 {
			output += ThinkingBlock("[SUCCESS] All tools succeeded - making progress")
		}

		state.GetOutputChannel() <- output
	}
}

func formatStringList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) <= 3 {
		return fmt.Sprintf("%s", items)
	}
	return fmt.Sprintf("%d items", len(items))
}

func formatToolList(toolCalls []*protocol.ToolCall) string {
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

func (s *ChainOfThoughtStrategy) GetContextInjection(state *ReasoningState) string {

	if state.GetServices() == nil {
		return ""
	}

	todoTool := s.getTodoTool(state)
	if todoTool == nil {
		return ""
	}

	sessionID := "default"
	todos := todoTool.GetTodos(sessionID)

	if len(todos) == 0 {
		return ""
	}

	if state.ShowThinking() && state.GetOutputChannel() != nil {
		s.displayTodos(todos, state.GetOutputChannel())
	}

	return tools.FormatTodosForContext(todos)
}

func (s *ChainOfThoughtStrategy) displayTodos(todos []tools.TodoItem, outputCh chan<- string) {
	outputCh <- "\n\033[90m**Current Tasks:**\n"
	for i, todo := range todos {
		var status string
		switch todo.Status {
		case "pending":
			status = "[PENDING]"
		case "in_progress":
			status = "[IN PROGRESS]"
		case "completed":
			status = "[DONE]"
		case "canceled":
			status = "[CANCELLED]"
		default:
			status = "[UNKNOWN]"
		}
		outputCh <- fmt.Sprintf("  %d. %s %s\n", i+1, status, todo.Content)
	}
	outputCh <- "\033[0m\n"
}

func (s *ChainOfThoughtStrategy) getTodoTool(state *ReasoningState) *tools.TodoTool {
	if state.GetServices() == nil || state.GetServices().Tools() == nil {
		return nil
	}

	tool, err := state.GetServices().Tools().GetTool("todo_write")
	if err != nil {
		return nil
	}

	todoTool, ok := tool.(*tools.TodoTool)
	if !ok {
		return nil
	}

	return todoTool
}

func (s *ChainOfThoughtStrategy) GetName() string {
	return "Chain-of-Thought"
}

func (s *ChainOfThoughtStrategy) GetDescription() string {
	return "Iterative reasoning with native LLM function calling (OpenAI, Anthropic)"
}

func (s *ChainOfThoughtStrategy) GetRequiredTools() []RequiredTool {
	return []RequiredTool{
		{
			Name:        "todo_write",
			Type:        "todo",
			Description: "Required for systematic task tracking in complex multi-step workflows",
			AutoCreate:  true,
		},
	}
}

func (s *ChainOfThoughtStrategy) GetPromptSlots() PromptSlots {
	return PromptSlots{
		SystemRole: `You are an AI assistant helping users solve problems and accomplish tasks.

You are pair programming with a USER to solve their task. Each time the USER sends a message, we may automatically attach information about their current state, files, context, and history. This information may or may not be relevant - it's up to you to decide.

You are an agent - keep going until the user's query is completely resolved before ending your turn. Only terminate when you're sure the problem is solved. Autonomously resolve the query to the best of your ability.

Your main goal is to follow the USER's instructions carefully.`,

		ReasoningInstructions: `Core execution principles:
- By default, IMPLEMENT actions rather than only suggesting them
- State assumptions and continue; don't stop for approval unless blocked
- Use tools to discover information when needed
- Break down complex problems into manageable steps
- When you've completed the user's request, provide a clear summary and stop
- Bias towards not asking the user for help if you can find the answer yourself

<flow>
1. When a new goal is detected: if needed, run a brief discovery pass (read-only scan)
2. For medium-to-large tasks, create a structured plan in the todo list (via todo_write)
3. For simpler tasks or read-only tasks, skip the todo list and execute directly
4. Before logical groups of tool calls, give a brief status update
5. When all tasks done, reconcile todos and give a brief summary
</flow>`,

		ToolUsage: `<tool_calling>
Use only provided tools; follow their schemas exactly.

Parallelize tool calls: batch read-only context reads and independent edits instead of serial calls.

Tools available (use as named):
- search: Semantic code search - your MAIN exploration tool
- execute_command: Run shell commands (grep, cat, git, make, npm, etc.)
- write_file: Create or overwrite files
- search_replace: Edit files by replacing exact text matches
- todo_write: Task management for complex workflows

If actions are dependent or might conflict, sequence them; otherwise, run them in parallel.
Don't mention tool names to the user; describe actions naturally.
If info is discoverable via tools, prefer that over asking the user.
Give a brief progress note before the first tool call each turn.
Whenever you complete tasks, call todo_write to update the todo list before reporting progress.

Gate before new edits: Before starting any new file or code edit, reconcile the TODO list via todo_write (merge=true): mark newly completed tasks as completed and set the next task to in_progress.
Cadence after steps: After each successful step, immediately update the corresponding TODO item's status.
</tool_calling>

<context_understanding>
Semantic search (search tool) is your MAIN exploration tool.

CRITICAL: Start with broad, high-level queries that capture overall intent (e.g. "authentication flow"), not low-level terms.
Break multi-part questions into focused sub-queries.
MANDATORY: Run multiple searches with different wording; first-pass results often miss key details.
Keep searching until you're CONFIDENT nothing important remains.
</context_understanding>

<maximize_parallel_tool_calls>
CRITICAL: For maximum efficiency, invoke all relevant tools concurrently rather than sequentially. Prioritize parallel execution.

Examples that SHOULD use parallel calls:
- Reading 3 files → 3 parallel execute_command calls
- Multiple search patterns → parallel search calls
- Reading multiple files or searching different directories → all at once
- Combining search with grep → parallel execution

Limit to 3-5 tool calls at a time to avoid timeouts.

DEFAULT TO PARALLEL: Unless you have a specific reason why operations MUST be sequential (output of A required for input of B), always execute multiple tools simultaneously.
</maximize_parallel_tool_calls>`,

		OutputFormat: `<communication>
- Always ensure **only relevant sections** are formatted in valid Markdown
- Avoid wrapping entire message in a code block
- Use backticks to format file, directory, function, and class names
- When communicating, optimize for clarity and skimmability
- Ensure code snippets are properly formatted for markdown rendering
- Refer to code changes as "edits" not "patches"
</communication>

<status_updates>
Give brief progress notes (1-3 sentences) about what just happened and what you're about to do.

Critical: If you say you're about to do something, actually do it in the same turn.

Use correct tenses: "I'll" for future actions, past tense for past actions.
Check off completed TODOs before reporting progress.
Before starting any new file/code edit, reconcile the todo list.
Only pause if you truly cannot proceed without the user.
Don't add headings like "Update:".

Your final status update should be a summary.
</status_updates>`,

		CommunicationStyle: `<summary_at_end>
At the end of your turn, provide a summary.

Summarize changes made at high-level and their impact. If user asked for info, summarize the answer but don't explain your search process. If basic query, skip summary.

Use concise bullet points; short paragraphs if needed.
Don't repeat the plan.
Include short code fences only when essential.
Keep summary short, non-repetitive, high-signal.
</summary_at_end>

<markdown>
- Use '###' and '##' headings (never '#')
- Use **bold** to highlight critical information
- Format files/directories/functions with backticks
- When mentioning URLs, use backticks or markdown links (prefer links)
</markdown>`,

		Additional: `<task_management>
Purpose: Use todo_write tool to track and manage tasks.

For multi-step tasks (3+ steps):
1. Create atomic todo items (≤14 words, verb-led, clear outcome)
2. Update todos as you progress (merge=true)
3. Mark tasks complete (status="completed") after finishing
4. When ALL todos complete, provide final summary WITHOUT additional tool calls

This signals task completion.

Todo content should be:
- Simple, clear, short
- Action-oriented (verb-based)
- High-level, meaningful tasks
- NOT include operational actions in service of higher-level tasks
</task_management>

<completion_spec>
When all goal tasks are done:
1. Confirm all tasks checked off (todo_write with merge=true)
2. Reconcile and close the todo list
3. Then give your summary
</completion_spec>

<making_code_changes>
When making code changes, NEVER output code to the USER unless requested. Use code edit tools instead.

Add all necessary imports, dependencies, and endpoints.
NEVER generate extremely long hashes or binary code.
Follow language-specific best practices for naming, formatting, and structure.
</making_code_changes>

<non_compliance>
If you fail to call todo_write to check off tasks before claiming them done, self-correct immediately.
If you used tools without a STATUS UPDATE, self-correct next turn.
If you report work as done without verification, self-correct by verifying first.

If a turn contains any tool call, the message MUST include at least one micro-update before those calls. This is not optional.
</non_compliance>

General guidelines:
- Provide accurate, factual information
- Never generate extremely long hashes or binary code
- Admit when uncertain
- Fix any errors you introduce
- Be respectful and inclusive`,
	}
}
