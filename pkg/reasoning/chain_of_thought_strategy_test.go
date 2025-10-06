package reasoning

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/llms"
)

func TestNewChainOfThoughtStrategy(t *testing.T) {
	strategy := NewChainOfThoughtStrategy()
	if strategy == nil {
		t.Fatal("NewChainOfThoughtStrategy() returned nil")
	}
}

func TestChainOfThoughtStrategy_PrepareIteration(t *testing.T) {
	strategy := NewChainOfThoughtStrategy()
	state := NewReasoningState()

	// PrepareIteration should not return an error
	err := strategy.PrepareIteration(1, state)
	if err != nil {
		t.Errorf("PrepareIteration() error = %v", err)
	}
}

func TestChainOfThoughtStrategy_ShouldStop(t *testing.T) {
	strategy := NewChainOfThoughtStrategy()
	state := NewReasoningState()

	// ShouldStop should return true when no tool calls
	shouldStop := strategy.ShouldStop("test text", []llms.ToolCall{}, state)
	if !shouldStop {
		t.Error("ShouldStop() should return true when no tool calls")
	}

	// ShouldStop should return false when there are tool calls
	toolCalls := []llms.ToolCall{
		{Name: "test_tool", Arguments: map[string]interface{}{}},
	}
	shouldStop = strategy.ShouldStop("test text", toolCalls, state)
	if shouldStop {
		t.Error("ShouldStop() should return false when there are tool calls")
	}
}

func TestChainOfThoughtStrategy_AfterIteration(t *testing.T) {
	strategy := NewChainOfThoughtStrategy()
	state := NewReasoningState()

	// AfterIteration should not return an error
	err := strategy.AfterIteration(1, "test text", []llms.ToolCall{}, []ToolResult{}, state)
	if err != nil {
		t.Errorf("AfterIteration() error = %v", err)
	}
}

func TestChainOfThoughtStrategy_GetName(t *testing.T) {
	strategy := NewChainOfThoughtStrategy()
	name := strategy.GetName()

	if name == "" {
		t.Error("GetName() should not return empty string")
	}

	if name != "Chain-of-Thought" {
		t.Errorf("GetName() = %v, want 'Chain-of-Thought'", name)
	}
}

func TestChainOfThoughtStrategy_GetDescription(t *testing.T) {
	strategy := NewChainOfThoughtStrategy()
	description := strategy.GetDescription()

	if description == "" {
		t.Error("GetDescription() should not return empty string")
	}
}

func TestChainOfThoughtStrategy_GetRequiredTools(t *testing.T) {
	strategy := NewChainOfThoughtStrategy()
	tools := strategy.GetRequiredTools()

	if len(tools) == 0 {
		t.Error("GetRequiredTools() should return at least one tool")
	}

	// Check if todo_write tool is required
	foundTodoTool := false
	for _, tool := range tools {
		if tool.Name == "todo_write" {
			foundTodoTool = true
			break
		}
	}

	if !foundTodoTool {
		t.Error("GetRequiredTools() should include todo_write tool")
	}
}

func TestChainOfThoughtStrategy_GetPromptSlots(t *testing.T) {
	strategy := NewChainOfThoughtStrategy()
	slots := strategy.GetPromptSlots()

	if slots.IsEmpty() {
		t.Error("GetPromptSlots() should not return empty slots")
	}
}
