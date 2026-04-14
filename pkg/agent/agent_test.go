package agent_test

import (
	"testing"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/verikod/hector/pkg/agent"
)

// =============================================================================
// Event Creation Tests
// =============================================================================

func TestNewEvent(t *testing.T) {
	event := agent.NewEvent("invocation-123")

	if event.ID == "" {
		t.Error("Event ID should be generated")
	}
	if event.InvocationID != "invocation-123" {
		t.Errorf("InvocationID = %q, want invocation-123", event.InvocationID)
	}
	if event.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
	if event.Actions.StateDelta == nil {
		t.Error("Actions.StateDelta should be initialized")
	}
}

// =============================================================================
// Event IsFinalResponse Tests
// =============================================================================

func TestEvent_IsFinalResponse(t *testing.T) {
	tests := []struct {
		name     string
		event    *agent.Event
		expected bool
	}{
		{
			name:     "empty event is final",
			event:    &agent.Event{},
			expected: true,
		},
		{
			name:     "partial event is not final",
			event:    &agent.Event{Partial: true},
			expected: false,
		},
		{
			name: "event with tool calls is not final",
			event: func() *agent.Event {
				e := &agent.Event{}
				e.SetToolCalls([]agent.ToolCallState{{ID: "call-1", Name: "test-tool"}})
				return e
			}(),
			expected: false,
		},
		{
			name: "event with tool results is not final",
			event: func() *agent.Event {
				e := &agent.Event{}
				e.SetToolResults([]agent.ToolResultState{{ToolCallID: "call-1", Content: "result"}})
				return e
			}(),
			expected: false,
		},
		{
			name: "SkipSummarization makes event final",
			event: func() *agent.Event {
				e := &agent.Event{Actions: agent.EventActions{SkipSummarization: true}}
				e.SetToolCalls([]agent.ToolCallState{{ID: "call-1", Name: "test-tool"}})
				return e
			}(),
			expected: true,
		},
		{
			name: "long running tool IDs makes event final",
			event: &agent.Event{
				LongRunningToolIDs: []string{"tool-1"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsFinalResponse(); got != tt.expected {
				t.Errorf("IsFinalResponse() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Event HasToolCalls Tests
// =============================================================================

func TestEvent_HasToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		event    *agent.Event
		expected bool
	}{
		{
			name:     "empty event has no tool calls",
			event:    &agent.Event{},
			expected: false,
		},
		{
			name: "event with ToolCalls has tool calls",
			event: func() *agent.Event {
				e := &agent.Event{}
				e.SetToolCalls([]agent.ToolCallState{{ID: "call-1", Name: "test-tool"}})
				return e
			}(),
			expected: true,
		},
		{
			name: "event with tool_use in artifact has tool calls",
			event: &agent.Event{
				Artifact: &a2a.Artifact{
					Parts: []a2a.Part{
						a2a.DataPart{Data: map[string]any{"type": "tool_use"}},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.HasToolCalls(); got != tt.expected {
				t.Errorf("HasToolCalls() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Event HasToolResults Tests
// =============================================================================

func TestEvent_HasToolResults(t *testing.T) {
	tests := []struct {
		name     string
		event    *agent.Event
		expected bool
	}{
		{
			name:     "empty event has no tool results",
			event:    &agent.Event{},
			expected: false,
		},
		{
			name: "event with ToolResults has tool results",
			event: func() *agent.Event {
				e := &agent.Event{}
				e.SetToolResults([]agent.ToolResultState{{ToolCallID: "call-1", Content: "result"}})
				return e
			}(),
			expected: true,
		},
		{
			name: "event with tool_result in artifact has tool results",
			event: &agent.Event{
				Artifact: &a2a.Artifact{
					Parts: []a2a.Part{
						a2a.DataPart{Data: map[string]any{"type": "tool_result"}},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.HasToolResults(); got != tt.expected {
				t.Errorf("HasToolResults() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Event TextContent Tests
// =============================================================================

func TestEvent_TextContent(t *testing.T) {
	tests := []struct {
		name     string
		event    *agent.Event
		expected string
	}{
		{
			name:     "nil artifact returns empty",
			event:    &agent.Event{},
			expected: "",
		},
		{
			name: "extracts text from TextPart",
			event: &agent.Event{
				Artifact: &a2a.Artifact{
					Parts: []a2a.Part{
						a2a.TextPart{Text: "Hello, world!"},
					},
				},
			},
			expected: "Hello, world!",
		},
		{
			name: "concatenates multiple TextParts",
			event: &agent.Event{
				Artifact: &a2a.Artifact{
					Parts: []a2a.Part{
						a2a.TextPart{Text: "Hello, "},
						a2a.TextPart{Text: "world!"},
					},
				},
			},
			expected: "Hello, world!",
		},
		{
			name: "ignores non-text parts",
			event: &agent.Event{
				Artifact: &a2a.Artifact{
					Parts: []a2a.Part{
						a2a.TextPart{Text: "Text"},
						a2a.DataPart{Data: map[string]any{"type": "other"}},
					},
				},
			},
			expected: "Text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.TextContent(); got != tt.expected {
				t.Errorf("TextContent() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Content Tests
// =============================================================================

func TestNewTextContent(t *testing.T) {
	content := agent.NewTextContent("Hello", a2a.MessageRoleUser)

	if len(content.Parts) != 1 {
		t.Fatalf("Expected 1 part, got %d", len(content.Parts))
	}
	if content.Role != a2a.MessageRoleUser {
		t.Errorf("Role = %v, want user", content.Role)
	}

	tp, ok := content.Parts[0].(a2a.TextPart)
	if !ok {
		t.Fatal("First part should be TextPart")
	}
	if tp.Text != "Hello" {
		t.Errorf("Text = %q, want Hello", tp.Text)
	}
}

func TestContent_ToMessage(t *testing.T) {
	t.Run("nil content returns nil", func(t *testing.T) {
		var c *agent.Content
		if c.ToMessage() != nil {
			t.Error("nil Content should return nil Message")
		}
	})

	t.Run("converts to a2a.Message", func(t *testing.T) {
		content := agent.NewTextContent("Test", a2a.MessageRoleAgent)
		msg := content.ToMessage()

		if msg == nil {
			t.Fatal("Message should not be nil")
		}
		if msg.Role != a2a.MessageRoleAgent {
			t.Errorf("Role = %v, want agent", msg.Role)
		}
	})
}

func TestContent_AddPart(t *testing.T) {
	content := &agent.Content{}
	content.AddPart(a2a.TextPart{Text: "added"})

	if len(content.Parts) != 1 {
		t.Errorf("Expected 1 part after AddPart, got %d", len(content.Parts))
	}
}

func TestContent_AddText(t *testing.T) {
	content := &agent.Content{}
	content.AddText("first")
	content.AddText("second")

	if len(content.Parts) != 2 {
		t.Errorf("Expected 2 parts after AddText, got %d", len(content.Parts))
	}
}

// =============================================================================
// AgentType Constants Tests
// =============================================================================

func TestAgentTypeConstants(t *testing.T) {
	// Verify constants exist with expected values
	tests := []struct {
		agentType agent.AgentType
		expected  string
	}{
		{agent.TypeCustomAgent, "custom"},
		{agent.TypeLLMAgent, "llm"},
		{agent.TypeSequentialAgent, "sequential"},
		{agent.TypeParallelAgent, "parallel"},
		{agent.TypeLoopAgent, "loop"},
		{agent.TypeRemoteAgent, "remote"},
		{agent.TypeRunnerAgent, "runner"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.agentType) != tt.expected {
				t.Errorf("AgentType = %q, want %q", tt.agentType, tt.expected)
			}
		})
	}
}

// =============================================================================
// Author Constants Tests
// =============================================================================

func TestAuthorConstants(t *testing.T) {
	if agent.AuthorUser != "user" {
		t.Errorf("AuthorUser = %q, want user", agent.AuthorUser)
	}
	if agent.AuthorSystem != "system" {
		t.Errorf("AuthorSystem = %q, want system", agent.AuthorSystem)
	}
}

// =============================================================================
// Metadata Type Constants Tests
// =============================================================================

func TestMetadataTypeConstants(t *testing.T) {
	if agent.MetadataTypeProgress != "progress" {
		t.Errorf("MetadataTypeProgress = %q", agent.MetadataTypeProgress)
	}
	if agent.MetadataTypeSummary != "summary" {
		t.Errorf("MetadataTypeSummary = %q", agent.MetadataTypeSummary)
	}
}

// =============================================================================
// Progress Status Constants Tests
// =============================================================================

func TestProgressStatusConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{agent.ProgressStatusSummarizing, "summarizing"},
		{agent.ProgressStatusSummarized, "summarized"},
		{agent.ProgressStatusError, "error"},
		{agent.ProgressStatusActive, "active"},
		{agent.ProgressStatusCompleted, "completed"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("%q != %q", tt.constant, tt.expected)
		}
	}
}

// =============================================================================
// ThinkingState Tests
// =============================================================================

func TestThinkingState_Fields(t *testing.T) {
	state := agent.ThinkingState{
		ID:        "thinking-1",
		Status:    "active",
		Content:   "Let me think...",
		Signature: "sig123",
		Type:      "planning",
	}

	if state.ID != "thinking-1" {
		t.Errorf("ID = %q", state.ID)
	}
	if state.Status != "active" {
		t.Errorf("Status = %q", state.Status)
	}
	if state.Content != "Let me think..." {
		t.Errorf("Content = %q", state.Content)
	}
}

// =============================================================================
// ToolCallState and ToolResultState Tests
// =============================================================================

func TestToolCallState_Fields(t *testing.T) {
	state := agent.ToolCallState{
		ID:     "call-1",
		Name:   "test-tool",
		Args:   map[string]any{"arg1": "value1"},
		Status: "working",
	}

	if state.ID != "call-1" {
		t.Errorf("ID = %q", state.ID)
	}
	if state.Name != "test-tool" {
		t.Errorf("Name = %q", state.Name)
	}
}

func TestToolResultState_Fields(t *testing.T) {
	state := agent.ToolResultState{
		ToolCallID: "call-1",
		Content:    "result content",
		Status:     "success",
		IsError:    false,
	}

	if state.ToolCallID != "call-1" {
		t.Errorf("ToolCallID = %q", state.ToolCallID)
	}
	if state.Status != "success" {
		t.Errorf("Status = %q", state.Status)
	}
}

// =============================================================================
// EventActions Tests
// =============================================================================

func TestEventActions_Fields(t *testing.T) {
	actions := agent.EventActions{
		StateDelta:        map[string]any{"key": "value"},
		SkipSummarization: true,
		TransferToAgent:   "other-agent",
		Escalate:          true,
		RequireInput:      true,
		InputPrompt:       "Please approve",
	}

	if actions.StateDelta["key"] != "value" {
		t.Error("StateDelta not set correctly")
	}
	if !actions.SkipSummarization {
		t.Error("SkipSummarization should be true")
	}
	if actions.TransferToAgent != "other-agent" {
		t.Error("TransferToAgent not set")
	}
	if !actions.RequireInput {
		t.Error("RequireInput should be true")
	}
}
