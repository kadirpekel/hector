package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/kadirpekel/hector/pkg/config"
)

// ============================================================================
// AGENT EDGE CASE TESTS
// Additional tests for error paths and edge cases
// ============================================================================

func TestAgent_ExecuteTask_EmptyUserMessage(t *testing.T) {
	services := NewMockAgentServices()
	services.llm.responses = []mockLLMResponse{
		{text: "I received an empty message", toolCalls: nil, tokens: 5},
	}

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "chain-of-thought",
				MaxIterations: 5,
			},
		},
		services: services,
	}

	task := &a2a.Task{
		ID: "empty-msg-test",
		Messages: []a2a.Message{
			{Role: a2a.MessageRoleUser, Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: ""}}},
		},
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	result, err := agent.ExecuteTask(context.Background(), task)

	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	// Should still complete (agent handles empty messages)
	if result.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Expected completed, got %s", result.Status.State)
	}
}

func TestAgent_ExecuteTask_InvalidStrategyEngine(t *testing.T) {
	services := NewMockAgentServices()

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "invalid-strategy-type",
				MaxIterations: 5,
			},
		},
		services: services,
	}

	task := &a2a.Task{
		ID: "invalid-strategy-test",
		Messages: []a2a.Message{
			{Role: a2a.MessageRoleUser, Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: "test"}}},
		},
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	result, err := agent.ExecuteTask(context.Background(), task)

	// Should not error but task should be failed
	if err != nil {
		t.Fatalf("ExecuteTask should not return error: %v", err)
	}

	if result.Status.State != a2a.TaskStateFailed {
		t.Errorf("Expected failed state, got %s", result.Status.State)
	}

	if result.Error == nil {
		t.Error("Expected error to be set in task")
	}

	if result.Error != nil && result.Error.Code != "strategy_error" {
		t.Errorf("Expected strategy_error code, got %s", result.Error.Code)
	}
}

func TestAgent_ExecuteTask_NoMessages(t *testing.T) {
	services := NewMockAgentServices()
	services.llm.responses = []mockLLMResponse{
		{text: "No previous messages", toolCalls: nil, tokens: 3},
	}

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "chain-of-thought",
				MaxIterations: 5,
			},
		},
		services: services,
	}

	task := &a2a.Task{
		ID:       "no-messages-test",
		Messages: []a2a.Message{}, // Empty messages
		Status:   a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	result, err := agent.ExecuteTask(context.Background(), task)

	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	// Should still complete
	if result.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Expected completed, got %s", result.Status.State)
	}
}

func TestAgent_GetAgentCard_Capabilities(t *testing.T) {
	services := NewMockAgentServices()
	agent := &Agent{
		name:        "test-agent",
		description: "Test",
		config:      &config.AgentConfig{Name: "test-agent", LLM: "mock"},
		services:    services,
	}

	card := agent.GetAgentCard()

	if card == nil {
		t.Fatal("GetAgentCard returned nil")
	}

	if card.Name != "test-agent" {
		t.Errorf("Expected name 'test-agent', got '%s'", card.Name)
	}

	// Check capabilities
	if !card.Capabilities.Streaming {
		t.Error("Expected streaming capability")
	}

	if !card.Capabilities.MultiTurn {
		t.Error("Expected multi-turn capability")
	}
}

func TestAgent_GetAgentCard_Description(t *testing.T) {
	services := NewMockAgentServices()
	agent := &Agent{
		name:        "described-agent",
		description: "This is a detailed description",
		config:      &config.AgentConfig{Name: "described-agent", LLM: "mock"},
		services:    services,
	}

	card := agent.GetAgentCard()

	if card.Description != "This is a detailed description" {
		t.Errorf("Expected description to be set, got '%s'", card.Description)
	}
}

func TestAgent_ExecuteTask_MultipleUserMessages(t *testing.T) {
	// Test handling of multiple user messages in history
	services := NewMockAgentServices()
	services.llm.responses = []mockLLMResponse{
		{text: "Response to latest message", toolCalls: nil, tokens: 5},
	}

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "chain-of-thought",
				MaxIterations: 5,
			},
		},
		services: services,
	}

	task := &a2a.Task{
		ID: "multi-user-msg-test",
		Messages: []a2a.Message{
			{Role: a2a.MessageRoleUser, Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: "First message"}}},
			{Role: a2a.MessageRoleAssistant, Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: "First response"}}},
			{Role: a2a.MessageRoleUser, Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: "Second message"}}},
		},
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	result, err := agent.ExecuteTask(context.Background(), task)

	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	if result.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Expected completed, got %s", result.Status.State)
	}

	// Should extract the last user message
	if len(result.Messages) < 4 { // Original 3 + new assistant response
		t.Errorf("Expected at least 4 messages, got %d", len(result.Messages))
	}
}

func TestAgent_ExecuteTask_OnlyAssistantMessages(t *testing.T) {
	// Edge case: task with only assistant messages
	services := NewMockAgentServices()
	services.llm.responses = []mockLLMResponse{
		{text: "Continuing conversation", toolCalls: nil, tokens: 3},
	}

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "chain-of-thought",
				MaxIterations: 5,
			},
		},
		services: services,
	}

	task := &a2a.Task{
		ID: "only-assistant-test",
		Messages: []a2a.Message{
			{Role: a2a.MessageRoleAssistant, Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: "Previous response"}}},
		},
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	result, err := agent.ExecuteTask(context.Background(), task)

	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	// Should still complete even with no user message
	if result.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Expected completed, got %s", result.Status.State)
	}
}

func TestAgent_ExecuteTask_LargeResponseAccumulation(t *testing.T) {
	// Test that large responses are accumulated correctly
	services := NewMockAgentServices()

	// Create a large response by making the LLM return multiple chunks
	largeText := strings.Repeat("This is a long response. ", 100)
	services.llm.responses = []mockLLMResponse{
		{text: largeText, toolCalls: nil, tokens: 500},
	}

	agent := &Agent{
		name:        "test-agent",
		description: "Test agent",
		config: &config.AgentConfig{
			Name: "test-agent",
			LLM:  "mock",
			Reasoning: config.ReasoningConfig{
				Engine:        "chain-of-thought",
				MaxIterations: 5,
			},
		},
		services: services,
	}

	task := &a2a.Task{
		ID: "large-response-test",
		Messages: []a2a.Message{
			{Role: a2a.MessageRoleUser, Parts: []a2a.Part{{Type: a2a.PartTypeText, Text: "Generate a large response"}}},
		},
		Status: a2a.TaskStatus{State: a2a.TaskStateSubmitted},
	}

	result, err := agent.ExecuteTask(context.Background(), task)

	if err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}

	if result.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Expected completed, got %s", result.Status.State)
	}

	// Verify the full response was captured
	if len(result.Messages) < 2 {
		t.Fatalf("Expected at least 2 messages, got %d", len(result.Messages))
	}

	lastMsg := result.Messages[len(result.Messages)-1]
	if len(lastMsg.Parts) == 0 {
		t.Fatal("Last message has no parts")
	}

	responseText := lastMsg.Parts[0].Text
	if len(responseText) < len(largeText) {
		t.Errorf("Response text was truncated: expected %d chars, got %d", len(largeText), len(responseText))
	}
}

// ============================================================================
// COVERAGE SUMMARY
// These edge case tests cover:
// - Empty user messages
// - Invalid strategy configuration
// - No messages in task
// - Agent card generation
// - Multiple user messages in history
// - Only assistant messages
// - Large response accumulation
//
// Combined with agent_execution_test.go and agent_methods_test.go,
// these tests target 50%+ coverage of agent business logic.
// ============================================================================
