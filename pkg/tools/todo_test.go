package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewTodoToolForTesting(t *testing.T) {
	tool := NewTodoToolForTesting()
	if tool == nil {
		t.Fatal("NewTodoToolForTesting() returned nil")
	}

	if tool.GetName() != "todo_write" {
		t.Errorf("GetName() = %v, want 'todo_write'", tool.GetName())
	}

	description := tool.GetDescription()
	if description == "" {
		t.Error("GetDescription() should not return empty string")
	}
}

func TestTodoTool_GetInfo(t *testing.T) {
	tool := NewTodoToolForTesting()
	info := tool.GetInfo()

	if info.Name == "" {
		t.Fatal("GetInfo() returned empty name")
	}

	if info.Description == "" {
		t.Error("Expected non-empty description")
	}
	if len(info.Parameters) == 0 {
		t.Error("Expected at least one parameter")
	}

	hasMergeParam := false
	for _, param := range info.Parameters {
		if param.Name == "merge" {
			hasMergeParam = true
			break
		}
	}
	if !hasMergeParam {
		t.Error("Expected 'merge' parameter")
	}
}

func TestTodoTool_Execute_WithCorrectParameters(t *testing.T) {
	tool := NewTodoToolForTesting()

	tests := []struct {
		name        string
		args        map[string]interface{}
		wantSuccess bool
		validate    func(t *testing.T, result ToolResult)
	}{
		{
			name: "add todos with merge=true",
			args: map[string]interface{}{
				"merge": true,
				"todos": []interface{}{
					map[string]interface{}{
						"id":      "1",
						"content": "Test todo 1",
						"status":  "pending",
					},
					map[string]interface{}{
						"id":      "2",
						"content": "Test todo 2",
						"status":  "in_progress",
					},
				},
			},
			wantSuccess: true,
			validate: func(t *testing.T, result ToolResult) {
				if !result.Success {
					t.Error("Expected success=true")
				}
				if result.Content == "" {
					t.Error("Expected non-empty content")
				}
			},
		},
		{
			name: "replace todos with merge=false",
			args: map[string]interface{}{
				"merge": false,
				"todos": []interface{}{
					map[string]interface{}{
						"id":      "1",
						"content": "Replacement todo",
						"status":  "pending",
					},
				},
			},
			wantSuccess: true,
			validate: func(t *testing.T, result ToolResult) {
				if !result.Success {
					t.Error("Expected success=true")
				}
			},
		},
		{
			name: "missing merge parameter",
			args: map[string]interface{}{
				"todos": []interface{}{
					map[string]interface{}{
						"id":      "1",
						"content": "Test todo",
						"status":  "pending",
					},
				},
			},
			wantSuccess: false,
		},
		{
			name: "missing todos parameter",
			args: map[string]interface{}{
				"merge": true,
			},
			wantSuccess: false,
		},
		{
			name: "empty todos array",
			args: map[string]interface{}{
				"merge": true,
				"todos": []interface{}{},
			},
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := tool.Execute(ctx, tt.args)

			if tt.wantSuccess {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
					return
				}
				tt.validate(t, result)
			} else {
				if err == nil {
					t.Error("Execute() expected error, got nil")
				}
			}
		})
	}
}

func TestTodoTool_GetTodos(t *testing.T) {
	tool := NewTodoToolForTesting()

	ctx := context.Background()
	_, err := tool.Execute(ctx, map[string]interface{}{
		"merge": true,
		"todos": []interface{}{
			map[string]interface{}{
				"id":      "1",
				"content": "First todo",
				"status":  "pending",
			},
			map[string]interface{}{
				"id":      "2",
				"content": "Second todo",
				"status":  "in_progress",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add todos: %v", err)
	}

	todos := tool.GetTodos("default")
	if len(todos) != 2 {
		t.Errorf("GetTodos() returned %d todos, want 2", len(todos))
	}

	if todos[0].Content != "First todo" && todos[1].Content != "First todo" {
		t.Error("Expected to find 'First todo'")
	}
	if todos[0].Status != "pending" && todos[1].Status != "pending" {
		t.Error("Expected to find todo with status 'pending'")
	}
}

func TestTodoTool_GetTodosSummary(t *testing.T) {
	tool := NewTodoToolForTesting()

	ctx := context.Background()
	_, err := tool.Execute(ctx, map[string]interface{}{
		"merge": true,
		"todos": []interface{}{
			map[string]interface{}{
				"id":      "1",
				"content": "Pending todo",
				"status":  "pending",
			},
			map[string]interface{}{
				"id":      "2",
				"content": "In progress todo",
				"status":  "in_progress",
			},
			map[string]interface{}{
				"id":      "3",
				"content": "Completed todo",
				"status":  "completed",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add todos: %v", err)
	}

	summary := tool.GetTodosSummary("default")
	if summary == "" {
		t.Fatal("GetTodosSummary() returned empty string")
	}

	if !strings.Contains(summary, "pending") {
		t.Errorf("Expected summary to contain 'pending', got: %s", summary)
	}
	if !strings.Contains(summary, "in progress") {
		t.Errorf("Expected summary to contain 'in progress', got: %s", summary)
	}
	if !strings.Contains(summary, "completed") {
		t.Errorf("Expected summary to contain 'completed', got: %s", summary)
	}
}

func TestTodoTool_GetTodos_JSONSerializable(t *testing.T) {
	tool := NewTodoToolForTesting()

	ctx := context.Background()
	_, err := tool.Execute(ctx, map[string]interface{}{
		"merge": true,
		"todos": []interface{}{
			map[string]interface{}{
				"id":      "1",
				"content": "Test todo",
				"status":  "pending",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add todo: %v", err)
	}

	todos := tool.GetTodos("default")
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}

	data, err := json.Marshal(todos)
	if err != nil {
		t.Fatalf("JSON marshal error = %v", err)
	}

	var unmarshaled []TodoItem
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("JSON unmarshal error = %v", err)
	}

	if len(unmarshaled) != 1 {
		t.Errorf("Expected 1 unmarshaled todo, got %d", len(unmarshaled))
	}
	if unmarshaled[0].Content != "Test todo" {
		t.Errorf("Expected content 'Test todo', got '%s'", unmarshaled[0].Content)
	}
}
