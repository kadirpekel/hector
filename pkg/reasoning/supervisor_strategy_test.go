package reasoning

import (
	"github.com/kadirpekel/hector/pkg/protocol"
	"testing"
)

func TestNewSupervisorStrategy(t *testing.T) {
	strategy := NewSupervisorStrategy()
	if strategy == nil {
		t.Fatal("NewSupervisorStrategy() returned nil")
	}
}

func TestSupervisorStrategy_PrepareIteration(t *testing.T) {
	strategy := NewSupervisorStrategy()
	state := NewReasoningState()

	// PrepareIteration should not return an error
	err := strategy.PrepareIteration(1, state)
	if err != nil {
		t.Errorf("PrepareIteration() error = %v", err)
	}
}

func TestSupervisorStrategy_ShouldStop(t *testing.T) {
	strategy := NewSupervisorStrategy()
	state := NewReasoningState()

	// ShouldStop should return true when no tool calls
	shouldStop := strategy.ShouldStop("test text", []*protocol.ToolCall{}, state)
	if !shouldStop {
		t.Error("ShouldStop() should return true when no tool calls")
	}

	// ShouldStop should return false when there are tool calls
	toolCalls := []*protocol.ToolCall{
		{Name: "test_tool", Args: map[string]interface{}{}},
	}
	shouldStop = strategy.ShouldStop("test text", toolCalls, state)
	if shouldStop {
		t.Error("ShouldStop() should return false when there are tool calls")
	}
}

func TestSupervisorStrategy_AfterIteration(t *testing.T) {
	strategy := NewSupervisorStrategy()
	state := NewReasoningState()

	// AfterIteration should not return an error
	err := strategy.AfterIteration(1, "test text", []*protocol.ToolCall{}, []ToolResult{}, state)
	if err != nil {
		t.Errorf("AfterIteration() error = %v", err)
	}
}

func TestSupervisorStrategy_GetName(t *testing.T) {
	strategy := NewSupervisorStrategy()
	name := strategy.GetName()

	if name == "" {
		t.Error("GetName() should not return empty string")
	}

	if name != "supervisor" {
		t.Errorf("GetName() = %v, want 'supervisor'", name)
	}
}

func TestSupervisorStrategy_GetDescription(t *testing.T) {
	strategy := NewSupervisorStrategy()
	description := strategy.GetDescription()

	if description == "" {
		t.Error("GetDescription() should not return empty string")
	}
}

func TestSupervisorStrategy_GetRequiredTools(t *testing.T) {
	strategy := NewSupervisorStrategy()
	tools := strategy.GetRequiredTools()

	// Supervisor strategy might not require any specific tools
	// Just check that it doesn't panic and returns a slice
	if tools == nil {
		t.Error("GetRequiredTools() should not return nil")
	}
}

func TestSupervisorStrategy_GetPromptSlots(t *testing.T) {
	strategy := NewSupervisorStrategy()
	slots := strategy.GetPromptSlots()

	if slots.IsEmpty() {
		t.Error("GetPromptSlots() should not return empty slots")
	}
}
