package reasoning

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/protocol"
)

func TestNewReasoningState(t *testing.T) {
	state := NewReasoningState()

	if state == nil {
		t.Fatal("NewReasoningState returned nil")
	}

	if state.Iteration() != 0 {
		t.Errorf("Expected Iteration 0, got %d", state.Iteration())
	}

	if state.TotalTokens() != 0 {
		t.Errorf("Expected TotalTokens 0, got %d", state.TotalTokens())
	}

	history := state.GetHistory()
	if history == nil {
		t.Error("History is nil")
	}

	if len(history) != 0 {
		t.Errorf("Expected empty History, got %d messages", len(history))
	}

	currentTurn := state.GetCurrentTurn()
	if currentTurn == nil {
		t.Error("CurrentTurn is nil")
	}

	if len(currentTurn) != 0 {
		t.Errorf("Expected empty CurrentTurn, got %d messages", len(currentTurn))
	}

	firstToolCalls := state.GetFirstIterationToolCalls()
	if firstToolCalls == nil {
		t.Error("FirstIterationToolCalls is nil")
	}

	if len(firstToolCalls) != 0 {
		t.Errorf("Expected empty FirstIterationToolCalls, got %d", len(firstToolCalls))
	}

	if state.GetCustomState() == nil {
		t.Error("CustomState is nil")
	}

	if len(state.GetCustomState()) != 0 {
		t.Errorf("Expected empty CustomState, got %d entries", len(state.GetCustomState()))
	}

	if state.GetToolState() == nil {
		t.Error("ToolState is nil")
	}

	if len(state.GetToolState()) != 0 {
		t.Errorf("Expected empty ToolState, got %d entries", len(state.GetToolState()))
	}
}

func TestReasoningState_CustomState(t *testing.T) {
	state := NewReasoningState()

	state.GetCustomState()["test_key"] = "test_value"
	state.GetCustomState()["counter"] = 42
	state.GetCustomState()["flag"] = true

	if val, ok := state.GetCustomState()["test_key"].(string); !ok || val != "test_value" {
		t.Error("Failed to store/retrieve string in CustomState")
	}

	if val, ok := state.GetCustomState()["counter"].(int); !ok || val != 42 {
		t.Error("Failed to store/retrieve int in CustomState")
	}

	if val, ok := state.GetCustomState()["flag"].(bool); !ok || !val {
		t.Error("Failed to store/retrieve bool in CustomState")
	}
}

func TestReasoningState_Conversation(t *testing.T) {
	state := NewReasoningState()

	historyMsgs := []*pb.Message{
		protocol.CreateUserMessage("Previous question"),
		{
			Role:    pb.Role_ROLE_AGENT,
			Content: []*pb.Part{{Part: &pb.Part_Text{Text: "Previous answer"}}},
		},
	}
	state.SetHistory(historyMsgs)

	state.AddCurrentTurnMessage(protocol.CreateUserMessage("Hello"))
	state.AddCurrentTurnMessage(&pb.Message{
		Role:    pb.Role_ROLE_AGENT,
		Content: []*pb.Part{{Part: &pb.Part_Text{Text: "Hi there"}}},
	})

	history := state.GetHistory()
	if len(history) != 2 {
		t.Errorf("Expected 2 history messages, got %d", len(history))
	}

	currentTurn := state.GetCurrentTurn()
	if len(currentTurn) != 2 {
		t.Errorf("Expected 2 current turn messages, got %d", len(currentTurn))
	}

	all := state.AllMessages()
	if len(all) != 4 {
		t.Errorf("Expected 4 total messages, got %d", len(all))
	}

	if all[0].Role != pb.Role_ROLE_USER {
		t.Error("First message should be user (history)")
	}
	if all[1].Role != pb.Role_ROLE_AGENT {
		t.Error("Second message should be agent (history)")
	}
	if all[2].Role != pb.Role_ROLE_USER {
		t.Error("Third message should be user (current)")
	}
	if all[3].Role != pb.Role_ROLE_AGENT {
		t.Error("Fourth message should be agent (current)")
	}
}

func TestReasoningState_AssistantResponse(t *testing.T) {
	state := NewReasoningState()

	state.AppendResponse("Part 1")
	state.AppendResponse(" Part 2")
	state.AppendResponse(" Part 3")

	result := state.GetAssistantResponse()
	expected := "Part 1 Part 2 Part 3"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestReasoningState_FirstIterationToolCalls(t *testing.T) {
	state := NewReasoningState()

	state.NextIteration()

	calls := []*protocol.ToolCall{
		{ID: "call-1", Name: "search"},
		{ID: "call-2", Name: "write_file"},
	}
	state.RecordFirstToolCalls(calls)

	firstToolCalls := state.GetFirstIterationToolCalls()
	if len(firstToolCalls) != 2 {
		t.Errorf("Expected 2 tool calls, got %d", len(firstToolCalls))
	}

	if firstToolCalls[0].Name != "search" {
		t.Error("First tool call should be 'search'")
	}

	if firstToolCalls[1].Name != "write_file" {
		t.Error("Second tool call should be 'write_file'")
	}
}

func TestReasoningState_Iteration(t *testing.T) {
	state := NewReasoningState()

	for i := 1; i <= 5; i++ {
		currentIter := state.NextIteration()
		if currentIter != i {
			t.Errorf("Expected iteration %d, got %d", i, currentIter)
		}
		if state.Iteration() != i {
			t.Errorf("Expected Iteration() to return %d, got %d", i, state.Iteration())
		}
	}
}

func TestReasoningState_TotalTokens(t *testing.T) {
	state := NewReasoningState()

	state.AddTokens(100)
	state.AddTokens(250)
	state.AddTokens(150)

	expected := 500
	if state.TotalTokens() != expected {
		t.Errorf("Expected TotalTokens %d, got %d", expected, state.TotalTokens())
	}
}

func TestReasoningState_ToolState(t *testing.T) {
	state := NewReasoningState()

	state.GetToolState()["todos_complete"] = true
	state.GetToolState()["file_watcher_active"] = false
	state.GetToolState()["retry_count"] = 3

	if val, ok := state.GetToolState()["todos_complete"].(bool); !ok || !val {
		t.Error("Failed to store/retrieve todos_complete in ToolState")
	}

	if val, ok := state.GetToolState()["file_watcher_active"].(bool); !ok || val {
		t.Error("Failed to store/retrieve file_watcher_active in ToolState")
	}

	if val, ok := state.GetToolState()["retry_count"].(int); !ok || val != 3 {
		t.Error("Failed to store/retrieve retry_count in ToolState")
	}
}

func TestReasoningState_AgentContext(t *testing.T) {
	state := NewReasoningState()

	if state.AgentName() != "" {
		t.Error("AgentName should be empty initially")
	}

	state.SetAgentName("test-agent")
	if state.AgentName() != "test-agent" {
		t.Errorf("Expected 'test-agent', got '%s'", state.AgentName())
	}

	if len(state.SubAgents()) != 0 {
		t.Error("SubAgents should be empty initially")
	}

	subAgents := []string{"agent1", "agent2", "agent3"}
	state.SetSubAgents(subAgents)

	retrieved := state.SubAgents()
	if len(retrieved) != 3 {
		t.Errorf("Expected 3 sub-agents, got %d", len(retrieved))
	}

	if retrieved[0] != "agent1" || retrieved[1] != "agent2" || retrieved[2] != "agent3" {
		t.Error("Sub-agents order/content mismatch")
	}
}

func TestReasoningState_Flags(t *testing.T) {

	state, err := Builder().
		WithQuery("test query").
		WithShowThinking(false).
		Build()

	if err != nil {
		t.Fatalf("Failed to build state: %v", err)
	}

	if state.ShowThinking() {
		t.Error("ShowThinking should be false")
	}

	state2, err := Builder().
		WithQuery("test query").
		WithShowThinking(true).
		Build()

	if err != nil {
		t.Fatalf("Failed to build state: %v", err)
	}

	if !state2.ShowThinking() {
		t.Error("ShowThinking should be true")
	}

	state3, err := Builder().
		WithQuery("test query").
		WithShowDebugInfo(false).
		Build()

	if err != nil {
		t.Fatalf("Failed to build state: %v", err)
	}

	if state3.ShowDebugInfo() {
		t.Error("ShowDebugInfo should be false")
	}

	state4, err := Builder().
		WithQuery("test query").
		WithShowDebugInfo(true).
		Build()

	if err != nil {
		t.Fatalf("Failed to build state: %v", err)
	}

	if !state4.ShowDebugInfo() {
		t.Error("ShowDebugInfo should be true")
	}
}

func TestReasoningState_Query(t *testing.T) {
	query := "What is the weather like?"
	state, err := Builder().
		WithQuery(query).
		Build()

	if err != nil {
		t.Fatalf("Failed to build state: %v", err)
	}

	if state.Query() != query {
		t.Errorf("Expected query to be '%s', got '%s'", query, state.Query())
	}
}
