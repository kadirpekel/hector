package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type TodoTool struct {
	mu    sync.RWMutex
	todos map[string][]TodoItem
}

type TodoItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

type TodoWriteRequest struct {
	Merge bool       `json:"merge"`
	Todos []TodoItem `json:"todos"`
}

func NewTodoTool() *TodoTool {
	return &TodoTool{
		todos: make(map[string][]TodoItem),
	}
}

func (t *TodoTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        "todo_write",
		Description: "Create and manage a structured task list for tracking progress. Use for complex multi-step tasks (3+ steps) to demonstrate thoroughness.",
		Parameters: []ToolParameter{
			{
				Name:        "merge",
				Type:        "boolean",
				Description: "If true, merge with existing todos (for updates). If false, replace all todos (for new task).",
				Required:    true,
			},
			{
				Name:        "todos",
				Type:        "array",
				Description: "Array of todo items. Each item has: id (string), content (string), status ('pending'|'in_progress'|'completed'|'canceled')",
				Required:    true,
				Items: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id": map[string]string{
							"type":        "string",
							"description": "Unique identifier for the todo",
						},
						"content": map[string]string{
							"type":        "string",
							"description": "Description of the task",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"pending", "in_progress", "completed", "canceled"},
							"description": "Current status of the task",
						},
					},
					"required": []string{"id", "content", "status"},
				},
			},
		},
		ServerURL: "local",
	}
}

func (t *TodoTool) GetName() string {
	return "todo_write"
}

func (t *TodoTool) GetDescription() string {
	return "Create and manage todos for complex tasks"
}

func (t *TodoTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	merge, ok := args["merge"].(bool)
	if !ok {
		return t.errorResult("merge parameter is required (true/false)", start),
			fmt.Errorf("merge parameter is required")
	}

	todosRaw, ok := args["todos"].([]interface{})
	if !ok || len(todosRaw) == 0 {
		return t.errorResult("todos parameter is required and must be a non-empty array", start),
			fmt.Errorf("todos parameter is required")
	}

	todos := make([]TodoItem, 0, len(todosRaw))
	for i, todoRaw := range todosRaw {
		todoMap, ok := todoRaw.(map[string]interface{})
		if !ok {
			return t.errorResult(fmt.Sprintf("todo item %d is not an object", i), start),
				fmt.Errorf("invalid todo item format")
		}

		id, _ := todoMap["id"].(string)
		content, _ := todoMap["content"].(string)
		status, _ := todoMap["status"].(string)

		if id == "" || content == "" || status == "" {
			return t.errorResult(fmt.Sprintf("todo item %d is missing required fields (id, content, status)", i), start),
				fmt.Errorf("incomplete todo item")
		}

		if !isValidStatus(status) {
			return t.errorResult(fmt.Sprintf("todo item %d has invalid status: %s", i, status), start),
				fmt.Errorf("invalid status")
		}

		todos = append(todos, TodoItem{
			ID:      id,
			Content: content,
			Status:  status,
		})
	}

	// Extract session ID from context
	// Using plain string key to match agent package
	const sessionIDKey = "hector:sessionID"
	sessionID := "default"
	if sid, ok := ctx.Value(sessionIDKey).(string); ok {
		sessionID = sid
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if merge {

		existing := t.todos[sessionID]
		if existing == nil {
			existing = make([]TodoItem, 0)
		}

		existingMap := make(map[string]*TodoItem)
		for i := range existing {
			existingMap[existing[i].ID] = &existing[i]
		}

		for _, newTodo := range todos {
			if existingTodo, found := existingMap[newTodo.ID]; found {

				existingTodo.Content = newTodo.Content
				existingTodo.Status = newTodo.Status
			} else {

				existing = append(existing, newTodo)
			}
		}

		t.todos[sessionID] = existing
	} else {

		t.todos[sessionID] = todos
	}

	summary := t.generateSummary(sessionID)

	return ToolResult{
		Success:       true,
		Content:       summary,
		ToolName:      "todo_write",
		ExecutionTime: time.Since(start),
		Metadata: map[string]interface{}{
			"session_id": sessionID,
			"merge":      merge,
			"count":      len(t.todos[sessionID]),
		},
	}, nil
}

func (t *TodoTool) GetTodos(sessionID string) []TodoItem {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if sessionID == "" {
		sessionID = "default"
	}

	todos := t.todos[sessionID]
	if todos == nil {
		return []TodoItem{}
	}

	result := make([]TodoItem, len(todos))
	copy(result, todos)
	return result
}

func (t *TodoTool) GetTodosSummary(sessionID string) string {
	if sessionID == "" {
		sessionID = "default"
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.generateSummary(sessionID)
}

func (t *TodoTool) generateSummary(sessionID string) string {
	todos := t.todos[sessionID]
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

func (t *TodoTool) errorResult(message string, start time.Time) ToolResult {
	return ToolResult{
		Success:       false,
		Error:         message,
		ToolName:      "todo_write",
		ExecutionTime: time.Since(start),
	}
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

func MarshalTodos(todos []TodoItem) (string, error) {
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
