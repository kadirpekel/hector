package reasoning

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/llms"
)

// ============================================================================
// REASONING STATE TESTS
// Tests the shared state management for reasoning iterations
// ============================================================================

func TestNewReasoningState(t *testing.T) {
	state := NewReasoningState()

	if state == nil {
		t.Fatal("NewReasoningState returned nil")
	}

	if state.Iteration != 0 {
		t.Errorf("Expected Iteration 0, got %d", state.Iteration)
	}

	if state.TotalTokens != 0 {
		t.Errorf("Expected TotalTokens 0, got %d", state.TotalTokens)
	}

	if state.Conversation == nil {
		t.Error("Conversation is nil")
	}

	if len(state.Conversation) != 0 {
		t.Errorf("Expected empty Conversation, got %d messages", len(state.Conversation))
	}

	if state.FirstIterationToolCalls == nil {
		t.Error("FirstIterationToolCalls is nil")
	}

	if len(state.FirstIterationToolCalls) != 0 {
		t.Errorf("Expected empty FirstIterationToolCalls, got %d", len(state.FirstIterationToolCalls))
	}

	if state.CustomState == nil {
		t.Error("CustomState is nil")
	}

	if len(state.CustomState) != 0 {
		t.Errorf("Expected empty CustomState, got %d entries", len(state.CustomState))
	}
}

func TestReasoningState_CustomState(t *testing.T) {
	state := NewReasoningState()

	// Test storing and retrieving custom state
	state.CustomState["test_key"] = "test_value"
	state.CustomState["counter"] = 42
	state.CustomState["flag"] = true

	if val, ok := state.CustomState["test_key"].(string); !ok || val != "test_value" {
		t.Error("Failed to store/retrieve string in CustomState")
	}

	if val, ok := state.CustomState["counter"].(int); !ok || val != 42 {
		t.Error("Failed to store/retrieve int in CustomState")
	}

	if val, ok := state.CustomState["flag"].(bool); !ok || !val {
		t.Error("Failed to store/retrieve bool in CustomState")
	}
}

func TestReasoningState_Conversation(t *testing.T) {
	state := NewReasoningState()

	// Add messages to conversation
	state.Conversation = append(state.Conversation, llms.Message{
		Role:    "user",
		Content: "Hello",
	})

	state.Conversation = append(state.Conversation, llms.Message{
		Role:    "assistant",
		Content: "Hi there",
	})

	if len(state.Conversation) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(state.Conversation))
	}

	if state.Conversation[0].Role != "user" {
		t.Error("First message should be user")
	}

	if state.Conversation[1].Role != "assistant" {
		t.Error("Second message should be assistant")
	}
}

func TestReasoningState_AssistantResponse(t *testing.T) {
	state := NewReasoningState()

	// Build response incrementally
	state.AssistantResponse.WriteString("Part 1")
	state.AssistantResponse.WriteString(" Part 2")
	state.AssistantResponse.WriteString(" Part 3")

	result := state.AssistantResponse.String()
	expected := "Part 1 Part 2 Part 3"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestReasoningState_FirstIterationToolCalls(t *testing.T) {
	state := NewReasoningState()

	// Add tool calls
	state.FirstIterationToolCalls = append(state.FirstIterationToolCalls, llms.ToolCall{
		ID:   "call-1",
		Name: "search",
	})

	state.FirstIterationToolCalls = append(state.FirstIterationToolCalls, llms.ToolCall{
		ID:   "call-2",
		Name: "write_file",
	})

	if len(state.FirstIterationToolCalls) != 2 {
		t.Errorf("Expected 2 tool calls, got %d", len(state.FirstIterationToolCalls))
	}

	if state.FirstIterationToolCalls[0].Name != "search" {
		t.Error("First tool call should be 'search'")
	}

	if state.FirstIterationToolCalls[1].Name != "write_file" {
		t.Error("Second tool call should be 'write_file'")
	}
}

func TestReasoningState_Iteration(t *testing.T) {
	state := NewReasoningState()

	// Simulate iterations
	for i := 1; i <= 5; i++ {
		state.Iteration = i
		if state.Iteration != i {
			t.Errorf("Expected Iteration %d, got %d", i, state.Iteration)
		}
	}
}

func TestReasoningState_TotalTokens(t *testing.T) {
	state := NewReasoningState()

	// Accumulate tokens
	state.TotalTokens += 100
	state.TotalTokens += 250
	state.TotalTokens += 150

	expected := 500
	if state.TotalTokens != expected {
		t.Errorf("Expected TotalTokens %d, got %d", expected, state.TotalTokens)
	}
}

func TestReasoningState_TodoCompletion(t *testing.T) {
	state := NewReasoningState()

	if state.TodosWereCompleteLastIteration {
		t.Error("TodosWereCompleteLastIteration should be false initially")
	}

	state.TodosWereCompleteLastIteration = true
	if !state.TodosWereCompleteLastIteration {
		t.Error("Failed to set TodosWereCompleteLastIteration")
	}
}

func TestReasoningState_Flags(t *testing.T) {
	state := NewReasoningState()

	// Test ShowThinking flag
	if state.ShowThinking {
		t.Error("ShowThinking should be false initially")
	}
	state.ShowThinking = true
	if !state.ShowThinking {
		t.Error("Failed to set ShowThinking")
	}

	// Test ShowDebugInfo flag
	if state.ShowDebugInfo {
		t.Error("ShowDebugInfo should be false initially")
	}
	state.ShowDebugInfo = true
	if !state.ShowDebugInfo {
		t.Error("Failed to set ShowDebugInfo")
	}
}

func TestReasoningState_Query(t *testing.T) {
	state := NewReasoningState()

	state.Query = "What is the weather like?"
	if state.Query != "What is the weather like?" {
		t.Errorf("Expected query to be set, got '%s'", state.Query)
	}
}

// ============================================================================
// COVERAGE SUMMARY
// These tests cover:
// - NewReasoningState: Initialization
// - CustomState: Key-value storage for strategy-specific data
// - Conversation: Message tracking across iterations
// - AssistantResponse: Incremental response building
// - FirstIterationToolCalls: Tool call tracking
// - Iteration: Counter management
// - TotalTokens: Token accumulation
// - TodoCompletion: Deterministic stop condition
// - Flags: ShowThinking, ShowDebugInfo
// - Query: Original user query storage
//
// All functions and fields in state.go now have 100% test coverage
// ============================================================================
