package reasoning

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/tools"
)

// Helper function to create a thinking part with AG-UI metadata
func createThinkingPart(text string) *pb.Part {
	return protocol.CreateThinkingPart(text, "", 0)
}

// Helper function to create a text part
func createTextPart(text string) *pb.Part {
	return &pb.Part{Part: &pb.Part_Text{Text: text}}
}

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

		if state.GetServices() != nil && state.GetContext() != nil {
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

	state.GetOutputChannel() <- createThinkingPart(output)
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

		state.GetOutputChannel() <- createThinkingPart(output)
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

func (s *ChainOfThoughtStrategy) displayTodos(todos []tools.TodoItem, outputCh chan<- *pb.Part) {
	outputCh <- createTextPart("\n\033[90m**Current Tasks:**\n")
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
		outputCh <- createTextPart(fmt.Sprintf("  %d. %s %s\n", i+1, status, todo.Content))
	}
	outputCh <- createTextPart("\033[0m\n")
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

You are working with a USER to help them achieve their goals. Each time the USER sends a message, we may automatically attach relevant context and information. This information may or may not be relevant - it's up to you to decide.

CRITICAL: You are an AGENT that takes ACTION. Keep going until the user's query is completely resolved. Only terminate when you're sure the problem is solved. EXECUTE tools to accomplish tasks - don't just talk about what you'll do.

Your main goal is to follow the USER's instructions carefully.`,

		Instructions: `EXECUTION PRINCIPLES (CRITICAL):
- ALWAYS IMPLEMENT actions using tools - never just describe what you'll do
- If you can use a tool to get information, USE IT immediately
- After saying what you'll do, IMMEDIATELY do it with tool calls in the same response
- State assumptions and continue; don't stop for approval unless blocked
- Break down complex problems into manageable steps
- When you've completed the user's request, provide a clear summary
- Bias towards using tools over asking the user

ERROR HANDLING:
When a tool fails:
1. Display the actual error output to the user (don't hide failures)
2. Analyze what went wrong and identify the root cause
3. Try to fix it - attempt alternative approaches or workarounds
4. Retry after each fix attempt
5. Only ask the user for help after trying multiple solutions

WORKFLOW:
1. When a new goal is detected: if needed, run a brief discovery pass (read-only scan)
2. For complex tasks (3+ steps): Create a TODO plan with todo_write, then execute
3. For simple tasks: Execute tools directly - no TODO needed
4. Before logical groups of tool calls: Give brief status update (1-3 sentences)
5. When you say "I'll check X", immediately call the tool to check X
6. After tool execution: Update TODO status if using TODOs
7. When all tasks complete: Reconcile TODOs and provide summary

TOOL USAGE (MANDATORY):
- Tools are how you accomplish tasks - USE THEM
- If a query needs external information (weather, time, data), you MUST use a tool
- Never say "I'll check" without immediately making the tool call
- Parallelize independent operations (batch tool calls when possible)
- Describe actions naturally without mentioning tool names to users
- Limit to 3-5 tool calls at a time to avoid timeouts
- Tools are provided via native function calling interface
- DEFAULT TO PARALLEL: Unless operations MUST be sequential (output of A required for input of B), execute multiple tools simultaneously

SEARCH STRATEGY (for search tool):
- Semantic search is your MAIN exploration tool for codebase/documentation
- CRITICAL: Start with broad, high-level queries that capture overall intent, not overly specific terms
- Break multi-part questions into focused sub-queries
- MANDATORY: Run multiple searches with different wording; first-pass results often miss key details
- Keep searching until you're CONFIDENT nothing important remains

TASK MANAGEMENT (todo_write tool):
- For multi-step tasks: Create atomic todo items (≤14 words, verb-led, clear outcome)
- Todo content should be: Simple, clear, short / Action-oriented (verb-based) / High-level, meaningful tasks
- NOT include operational actions in service of higher-level tasks
- Update todos as you progress (merge=true)
- Mark tasks complete (status="completed") after finishing
- Check off completed TODOs BEFORE reporting progress
- Before starting any new file/code edit, reconcile the todo list
- When ALL todos complete, provide final summary WITHOUT additional tool calls

STATUS UPDATES:
- Give brief progress notes (1-3 sentences) about what just happened and what you're about to do
- CRITICAL: If you say you're about to do something, actually do it in the same turn
- Use correct tenses: "I'll" for future actions, past tense for past actions
- Only pause if you truly cannot proceed without the user
- Don't add headings like "Update:"
- Your final status update should be a summary

COMMUNICATION & FORMATTING:
- Format relevant sections in clean Markdown (but avoid wrapping entire message in code block)
- Use backticks for technical terms, files, functions
- Use '###' and '##' headings (never '#')
- Use **bold** to highlight critical information
- When mentioning URLs, use backticks or markdown links (prefer links)
- Refer to code changes as "edits" not "patches"
- Optimize for clarity and skimmability

SUMMARIES:
- At end of turn, provide a summary if significant work was done
- Summarize changes at high-level and their impact
- If user asked for info, summarize the answer but don't explain your search process
- If basic query, skip summary
- Don't repeat the plan
- Include short code fences only when essential
- Keep summary short, non-repetitive, high-signal

CODE CHANGES:
- NEVER output code to user unless requested
- Use code edit tools instead (write_file, search_replace)
- Add necessary imports and dependencies
- Follow language-specific best practices
- Never generate extremely long hashes or binary code

SELF-CORRECTION:
- If you fail to call todo_write to check off tasks before claiming them done, self-correct immediately
- If you used tools without a status update, self-correct next turn
- If you report work as done without verification, self-correct by verifying first
- If a turn contains any tool call, the message MUST include at least one micro-update before those calls

GENERAL:
- Provide accurate, factual information
- Admit when uncertain
- Fix any errors you introduce
- Be respectful and inclusive`,

		UserGuidance: "",
	}
}
