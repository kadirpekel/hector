package tool_test

import (
	"testing"

	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/tool"
)

// =============================================================================
// Mock Tool for Testing
// =============================================================================

type mockTool struct {
	name             string
	description      string
	isLongRunning    bool
	requiresApproval bool
	schema           map[string]any
}

func (m mockTool) Name() string           { return m.name }
func (m mockTool) Description() string    { return m.description }
func (m mockTool) IsLongRunning() bool    { return m.isLongRunning }
func (m mockTool) RequiresApproval() bool { return m.requiresApproval }
func (m mockTool) Schema() map[string]any { return m.schema }
func (m mockTool) Call(ctx tool.Context, args map[string]any) (map[string]any, error) {
	return nil, nil
}

// Ensure mockTool implements CallableTool
var _ tool.CallableTool = mockTool{}

// =============================================================================
// Predicate Tests
// =============================================================================

func TestStringPredicate(t *testing.T) {
	predicate := tool.StringPredicate([]string{"allowed1", "allowed2"})

	t.Run("allows listed tools", func(t *testing.T) {
		if !predicate(nil, mockTool{name: "allowed1"}) {
			t.Error("Should allow 'allowed1'")
		}
		if !predicate(nil, mockTool{name: "allowed2"}) {
			t.Error("Should allow 'allowed2'")
		}
	})

	t.Run("denies unlisted tools", func(t *testing.T) {
		if predicate(nil, mockTool{name: "denied"}) {
			t.Error("Should deny 'denied'")
		}
	})
}

func TestAllowAll(t *testing.T) {
	predicate := tool.AllowAll()

	if !predicate(nil, mockTool{name: "any"}) {
		t.Error("AllowAll should allow any tool")
	}
	if !predicate(nil, mockTool{name: "another"}) {
		t.Error("AllowAll should allow any tool")
	}
}

func TestDenyAll(t *testing.T) {
	predicate := tool.DenyAll()

	if predicate(nil, mockTool{name: "any"}) {
		t.Error("DenyAll should deny any tool")
	}
	if predicate(nil, mockTool{name: "another"}) {
		t.Error("DenyAll should deny any tool")
	}
}

func TestCombine(t *testing.T) {
	t.Run("all predicates must pass (AND)", func(t *testing.T) {
		p1 := tool.StringPredicate([]string{"tool1", "tool2"})
		p2 := func(ctx agent.ReadonlyContext, t tool.Tool) bool {
			return t.Name() != "tool2" // Deny tool2
		}

		combined := tool.Combine(p1, p2)

		if !combined(nil, mockTool{name: "tool1"}) {
			t.Error("Should allow tool1 (passes both)")
		}
		if combined(nil, mockTool{name: "tool2"}) {
			t.Error("Should deny tool2 (fails p2)")
		}
		if combined(nil, mockTool{name: "tool3"}) {
			t.Error("Should deny tool3 (fails p1)")
		}
	})

	t.Run("empty predicates allows all", func(t *testing.T) {
		combined := tool.Combine()
		if !combined(nil, mockTool{name: "any"}) {
			t.Error("Empty Combine should allow all")
		}
	})
}

func TestOr(t *testing.T) {
	t.Run("any predicate can pass (OR)", func(t *testing.T) {
		p1 := tool.StringPredicate([]string{"tool1"})
		p2 := tool.StringPredicate([]string{"tool2"})

		combined := tool.Or(p1, p2)

		if !combined(nil, mockTool{name: "tool1"}) {
			t.Error("Should allow tool1")
		}
		if !combined(nil, mockTool{name: "tool2"}) {
			t.Error("Should allow tool2")
		}
		if combined(nil, mockTool{name: "tool3"}) {
			t.Error("Should deny tool3")
		}
	})

	t.Run("empty predicates denies all", func(t *testing.T) {
		combined := tool.Or()
		if combined(nil, mockTool{name: "any"}) {
			t.Error("Empty Or should deny all")
		}
	})
}

func TestNot(t *testing.T) {
	predicate := tool.StringPredicate([]string{"blocked"})
	negated := tool.Not(predicate)

	t.Run("negates the predicate", func(t *testing.T) {
		if negated(nil, mockTool{name: "blocked"}) {
			t.Error("Should deny 'blocked' (negated)")
		}
		if !negated(nil, mockTool{name: "allowed"}) {
			t.Error("Should allow 'allowed' (negated)")
		}
	})
}

// =============================================================================
// ToDefinition Tests
// =============================================================================

func TestToDefinition(t *testing.T) {
	t.Run("extracts name and description", func(t *testing.T) {
		mock := mockTool{
			name:        "test_tool",
			description: "A test tool",
		}

		def := tool.ToDefinition(mock)

		if def.Name != "test_tool" {
			t.Errorf("Name = %q, want test_tool", def.Name)
		}
		if def.Description != "A test tool" {
			t.Errorf("Description = %q", def.Description)
		}
	})

	t.Run("includes schema from CallableTool", func(t *testing.T) {
		mock := mockTool{
			name:   "tool_with_schema",
			schema: map[string]any{"type": "object"},
		}

		def := tool.ToDefinition(mock)

		if def.Parameters == nil {
			t.Error("Parameters should include schema")
		}
		if def.Parameters["type"] != "object" {
			t.Error("Parameters should contain schema data")
		}
	})
}

// =============================================================================
// Definition Tests
// =============================================================================

func TestDefinition_Fields(t *testing.T) {
	def := tool.Definition{
		Name:        "my_tool",
		Description: "Does something",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"arg1": map[string]any{"type": "string"},
			},
		},
	}

	if def.Name != "my_tool" {
		t.Errorf("Name = %q", def.Name)
	}
	if def.Description != "Does something" {
		t.Errorf("Description = %q", def.Description)
	}
	if def.Parameters == nil {
		t.Error("Parameters should not be nil")
	}
}

// =============================================================================
// ToolCall Tests
// =============================================================================

func TestToolCall_Fields(t *testing.T) {
	call := tool.ToolCall{
		ID:   "call-123",
		Name: "test_tool",
		Args: map[string]any{"key": "value"},
	}

	if call.ID != "call-123" {
		t.Errorf("ID = %q", call.ID)
	}
	if call.Name != "test_tool" {
		t.Errorf("Name = %q", call.Name)
	}
	if call.Args["key"] != "value" {
		t.Error("Args not set correctly")
	}
}

// =============================================================================
// ToolResult Tests
// =============================================================================

func TestToolResult_Fields(t *testing.T) {
	result := tool.ToolResult{
		ToolCallID: "call-123",
		Content:    "Success!",
		Error:      "",
		Metadata:   map[string]any{"timing": 100},
	}

	if result.ToolCallID != "call-123" {
		t.Errorf("ToolCallID = %q", result.ToolCallID)
	}
	if result.Content != "Success!" {
		t.Errorf("Content = %q", result.Content)
	}
	if result.Metadata["timing"] != 100 {
		t.Error("Metadata not set correctly")
	}
}

func TestToolResult_WithError(t *testing.T) {
	result := tool.ToolResult{
		ToolCallID: "call-456",
		Content:    "",
		Error:      "Something went wrong",
	}

	if result.Error != "Something went wrong" {
		t.Errorf("Error = %q", result.Error)
	}
}

// =============================================================================
// Result Tests
// =============================================================================

func TestResult_Fields(t *testing.T) {
	result := tool.Result{
		Content:   "Hello",
		Streaming: true,
		Error:     "",
		Metadata:  map[string]any{"chunk": 1},
	}

	if result.Content != "Hello" {
		t.Errorf("Content = %v", result.Content)
	}
	if !result.Streaming {
		t.Error("Streaming should be true")
	}
}

func TestResult_StreamingChunk(t *testing.T) {
	chunk := tool.Result{Content: "partial...", Streaming: true}
	final := tool.Result{Content: "complete", Streaming: false}

	if !chunk.Streaming {
		t.Error("Chunk should be streaming")
	}
	if final.Streaming {
		t.Error("Final should not be streaming")
	}
}

// =============================================================================
// Request Tests
// =============================================================================

func TestRequest_Fields(t *testing.T) {
	req := tool.Request{
		SystemInstruction: "You are helpful",
		Messages:          []string{"msg1", "msg2"},
		Config:            map[string]any{"temp": 0.7},
		Metadata:          map[string]any{"request_id": "123"},
	}

	if req.SystemInstruction != "You are helpful" {
		t.Errorf("SystemInstruction = %q", req.SystemInstruction)
	}
	if req.Metadata["request_id"] != "123" {
		t.Error("Metadata not set correctly")
	}
}
