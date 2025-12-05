// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package todotool

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/kadirpekel/hector/v2/tool"
	"github.com/kadirpekel/hector/v2/tool/functiontool"
)

// TodoItem represents a single todo item.
type TodoItem struct {
	ID      string `json:"id" jsonschema:"required,description=Unique identifier for the todo"`
	Content string `json:"content" jsonschema:"required,description=Description of the task"`
	Status  string `json:"status" jsonschema:"required,description=Current status of the task,enum=pending|in_progress|completed|canceled"`
}

// TodoWriteArgs defines the parameters for writing todos.
type TodoWriteArgs struct {
	Merge bool       `json:"merge" jsonschema:"required,description=If true merge with existing todos (for updates). If false replace all todos (for new task)."`
	Todos []TodoItem `json:"todos" jsonschema:"required,description=Array of todo items. Must contain at least one item - empty arrays are not allowed. Completed todos remain in the list.,minItems=1"`
}

// TodoManager manages todo state across sessions.
// It provides both the tool and methods to query todos.
type TodoManager struct {
	mu    sync.RWMutex
	todos map[string][]TodoItem
}

// NewTodoManager creates a new TodoManager.
func NewTodoManager() *TodoManager {
	return &TodoManager{
		todos: make(map[string][]TodoItem),
	}
}

// Tool creates a todo_write tool using FunctionTool.
func (m *TodoManager) Tool() (tool.CallableTool, error) {
	return functiontool.NewWithValidation(
		functiontool.Config{
			Name:        "todo_write",
			Description: "Create and manage a structured task list for tracking progress. Use for complex multi-step tasks (3+ steps) to demonstrate thoroughness. IMPORTANT: You cannot clear todos - the todos array must always contain at least one item. Completed todos remain in the list.",
		},
		func(ctx tool.Context, args TodoWriteArgs) (map[string]any, error) {
			return m.writeTodos(ctx, args)
		},
		func(args TodoWriteArgs) error {
			// Validate todos array is not empty
			if len(args.Todos) == 0 {
				return fmt.Errorf("todos array cannot be empty. You cannot clear todos - completed todos remain in the list. To update todos, include at least one todo item with id, content, and status")
			}

			// Validate each todo item
			for i, todo := range args.Todos {
				if todo.ID == "" || todo.Content == "" || todo.Status == "" {
					return fmt.Errorf("todo item %d is missing required fields (id, content, status)", i)
				}
				if !isValidStatus(todo.Status) {
					return fmt.Errorf("todo item %d has invalid status: %s (must be pending, in_progress, completed, or canceled)", i, todo.Status)
				}
			}

			return nil
		},
	)
}

func (m *TodoManager) writeTodos(ctx tool.Context, args TodoWriteArgs) (map[string]any, error) {
	// Extract session ID from context
	sessionID := ctx.SessionID()
	if sessionID == "" {
		sessionID = "default"
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if args.Merge {
		// Merge with existing todos
		existing := m.todos[sessionID]
		if existing == nil {
			existing = make([]TodoItem, 0)
		}

		existingMap := make(map[string]*TodoItem)
		for i := range existing {
			existingMap[existing[i].ID] = &existing[i]
		}

		for _, newTodo := range args.Todos {
			if existingTodo, found := existingMap[newTodo.ID]; found {
				// Update existing
				existingTodo.Content = newTodo.Content
				existingTodo.Status = newTodo.Status
			} else {
				// Add new
				existing = append(existing, newTodo)
			}
		}

		m.todos[sessionID] = existing
	} else {
		// Replace all todos
		m.todos[sessionID] = args.Todos
	}

	summary := m.generateSummary(sessionID)

	return map[string]any{
		"summary":    summary,
		"session_id": sessionID,
		"merge":      args.Merge,
		"count":      len(m.todos[sessionID]),
	}, nil
}

// GetTodos returns todos for a session.
func (m *TodoManager) GetTodos(sessionID string) []TodoItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if sessionID == "" {
		sessionID = "default"
	}

	todos := m.todos[sessionID]
	if todos == nil {
		return []TodoItem{}
	}

	result := make([]TodoItem, len(todos))
	copy(result, todos)
	return result
}

// GetTodosSummary returns a formatted summary of todos.
func (m *TodoManager) GetTodosSummary(sessionID string) string {
	if sessionID == "" {
		sessionID = "default"
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.generateSummary(sessionID)
}

func (m *TodoManager) generateSummary(sessionID string) string {
	todos := m.todos[sessionID]
	if len(todos) == 0 {
		return "No active todos"
	}

	var pending, inProgress, completed, canceled int
	for _, todo := range todos {
		switch todo.Status {
		case "pending":
			pending++
		case "in_progress":
			inProgress++
		case "completed":
			completed++
		case "canceled":
			canceled++
		}
	}

	summary := fmt.Sprintf("Todo Summary: %d total (%d pending, %d in progress, %d completed, %d canceled)\n\n",
		len(todos), pending, inProgress, completed, canceled)

	for _, todo := range todos {
		icon := getStatusIcon(todo.Status)
		summary += fmt.Sprintf("%s [%s] %s\n", icon, todo.ID, todo.Content)
	}

	return summary
}

func isValidStatus(status string) bool {
	return status == "pending" || status == "in_progress" || status == "completed" || status == "canceled"
}

func getStatusIcon(status string) string {
	switch status {
	case "pending":
		return "[PENDING]"
	case "in_progress":
		return "[IN PROGRESS]"
	case "completed":
		return "[DONE]"
	case "canceled":
		return "[CANCELLED]"
	default:
		return "[UNKNOWN]"
	}
}

// FormatTodosForContext formats todos for inclusion in agent context.
func FormatTodosForContext(todos []TodoItem) string {
	if len(todos) == 0 {
		return ""
	}

	var result string
	result += "\n<current_todos>\n"
	result += "Your current task list:\n\n"

	for _, todo := range todos {
		icon := getStatusIcon(todo.Status)
		result += fmt.Sprintf("%s %s - %s\n", icon, todo.Status, todo.Content)
	}

	result += "\nRemember to update todo status using todo_write tool as you make progress.\n"
	result += "</current_todos>\n"

	return result
}

// MarshalTodos marshals todos to JSON.
func MarshalTodos(todos []TodoItem) (string, error) {
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
