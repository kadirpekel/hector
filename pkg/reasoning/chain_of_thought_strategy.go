package reasoning

import (
	"context"
	"log"

	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/tools"
)

// extractSessionID gets the session ID from context, returns "default" if not found
func extractSessionID(ctx context.Context) string {
	if ctx == nil {
		return "default"
	}
	if sessionIDValue := ctx.Value(protocol.SessionIDKey); sessionIDValue != nil {
		if sid, ok := sessionIDValue.(string); ok {
			return sid
		}
	}
	return "default"
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
		sessionID := extractSessionID(state.GetContext())
		todos := todoTool.GetTodos(sessionID)
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
	// Log iteration progress for operators/deployers
	if len(toolCalls) > 0 {
		successCount := 0
		failCount := 0

		for _, result := range results {
			if result.Error != nil {
				failCount++
			} else {
				successCount++
			}
		}

		toolNames := make([]string, len(toolCalls))
		for i, tc := range toolCalls {
			toolNames[i] = tc.Name
		}

		log.Printf("[ChainOfThought] Iteration %d: executed %d tool(s) %v (success: %d, failed: %d)",
			iteration, len(toolCalls), toolNames, successCount, failCount)
	}

	return nil
}

func (s *ChainOfThoughtStrategy) GetContextInjection(state *ReasoningState) string {

	if state.GetServices() == nil {
		return ""
	}

	todoTool := s.getTodoTool(state)
	if todoTool == nil {
		return ""
	}

	sessionID := extractSessionID(state.GetContext())
	todos := todoTool.GetTodos(sessionID)

	if len(todos) == 0 {
		return ""
	}

	// Only inject context for LLM, don't display
	// (display is handled by the tool itself when executed)
	return tools.FormatTodosForContext(todos)
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
		SystemRole: `You are an AI assistant. You work through tasks iteratively until complete.

When you have active tasks, each iteration shows a <current_todos> section with your task list status.`,

		Instructions: `## Starting Work

For multi-step requests (2+ distinct actions):
- Write 1-2 sentences acknowledging the request
- Call todo_write tool immediately
- Don't write "## TODO List" or explain what you're doing

For single simple requests: answer directly.

Example flow:
"I'll help you with these tasks."
[call todo_write tool]

## Working Through Tasks

One task per iteration. For each task:
1. Execute tool(s) needed for that task
2. Present results with detail and formatting
3. Update todo (merge=true) with brief summary and mark status
4. Move to next task

You can call multiple tools in parallel within a single task, but don't jump between different tasks in the same iteration.

## Presenting Results

Show the actual data directly:
- Use markdown formatting (lists, bold, structure)
- Include relevant details from tool responses
- Don't add preambles ("The TODO list has been created...", "Here are the results...")
- Brief transitions between tasks are fine ("Moving to the next item.")

When all tasks complete: output only "✓ Done."

## Information Layering

Messages vs TODOs:
- Your messages: full details, rich formatting
- TODO updates: brief summaries for tracking

After executing a task, present full results in your message, then update the TODO with a summary.`,

		UserGuidance: "",
	}
}
