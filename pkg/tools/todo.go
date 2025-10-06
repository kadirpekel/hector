package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// TODO MANAGEMENT TOOL - Matches Cursor's todo_write functionality
// ============================================================================

// TodoTool provides task/todo management capabilities
// This allows the LLM to create, update, and track todos systematically
type TodoTool struct {
	mu    sync.RWMutex
	todos map[string][]TodoItem // Per-session todos (sessionID -> todos)
}

// TodoItem represents a single todo/task
type TodoItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"` // "pending", "in_progress", "completed", "cancelled"
}

// TodoWriteRequest represents the parameters for todo_write
type TodoWriteRequest struct {
	Merge bool       `json:"merge"` // If true, merge with existing; if false, replace
	Todos []TodoItem `json:"todos"` // List of todos
}

// NewTodoTool creates a new todo management tool
func NewTodoTool() *TodoTool {
	return &TodoTool{
		todos: make(map[string][]TodoItem),
	}
}

// GetInfo implements Tool interface
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
				Description: "Array of todo items. Each item has: id (string), content (string), status ('pending'|'in_progress'|'completed'|'cancelled')",
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
							"enum":        []string{"pending", "in_progress", "completed", "cancelled"},
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

// GetName implements Tool interface
func (t *TodoTool) GetName() string {
	return "todo_write"
}

// GetDescription implements Tool interface
func (t *TodoTool) GetDescription() string {
	return "Create and manage todos for complex tasks"
}

// Execute implements Tool interface
func (t *TodoTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	// Extract merge flag
	merge, ok := args["merge"].(bool)
	if !ok {
		return t.errorResult("merge parameter is required (true/false)", start),
			fmt.Errorf("merge parameter is required")
	}

	// Extract todos array
	todosRaw, ok := args["todos"].([]interface{})
	if !ok || len(todosRaw) == 0 {
		return t.errorResult("todos parameter is required and must be a non-empty array", start),
			fmt.Errorf("todos parameter is required")
	}

	// Parse todos
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

		// Validate status
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

	// Session ID (for multi-session support)
	// In single-agent mode, use "default"
	sessionID := "default"
	if sid, ok := ctx.Value("session_id").(string); ok {
		sessionID = sid
	}

	// Update todos
	t.mu.Lock()
	defer t.mu.Unlock()

	if merge {
		// Merge: Update existing todos by ID
		existing := t.todos[sessionID]
		if existing == nil {
			existing = make([]TodoItem, 0)
		}

		// Create a map for quick lookup
		existingMap := make(map[string]*TodoItem)
		for i := range existing {
			existingMap[existing[i].ID] = &existing[i]
		}

		// Update existing todos
		for _, newTodo := range todos {
			if existingTodo, found := existingMap[newTodo.ID]; found {
				// Update existing
				existingTodo.Content = newTodo.Content
				existingTodo.Status = newTodo.Status
			} else {
				// Add new
				existing = append(existing, newTodo)
			}
		}

		t.todos[sessionID] = existing
	} else {
		// Replace: Set new todos
		t.todos[sessionID] = todos
	}

	// Generate summary
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

// GetTodos returns current todos for a session (for system to inject into context)
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

	// Return a copy
	result := make([]TodoItem, len(todos))
	copy(result, todos)
	return result
}

// GetTodosSummary returns a formatted summary of current todos
func (t *TodoTool) GetTodosSummary(sessionID string) string {
	if sessionID == "" {
		sessionID = "default"
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.generateSummary(sessionID)
}

// generateSummary creates a human-readable summary of todos
func (t *TodoTool) generateSummary(sessionID string) string {
	todos := t.todos[sessionID]
	if len(todos) == 0 {
		return "✅ No active todos"
	}

	var pending, inProgress, completed, cancelled int
	for _, todo := range todos {
		switch todo.Status {
		case "pending":
			pending++
		case "in_progress":
			inProgress++
		case "completed":
			completed++
		case "cancelled":
			cancelled++
		}
	}

	summary := fmt.Sprintf("📋 Todo Summary: %d total (%d pending, %d in progress, %d completed, %d cancelled)\n\n",
		len(todos), pending, inProgress, completed, cancelled)

	// List all todos
	for _, todo := range todos {
		icon := getStatusIcon(todo.Status)
		summary += fmt.Sprintf("%s [%s] %s\n", icon, todo.ID, todo.Content)
	}

	return summary
}

// errorResult creates an error ToolResult
func (t *TodoTool) errorResult(message string, start time.Time) ToolResult {
	return ToolResult{
		Success:       false,
		Error:         message,
		ToolName:      "todo_write",
		ExecutionTime: time.Since(start),
	}
}

// isValidStatus checks if a status is valid
func isValidStatus(status string) bool {
	return status == "pending" || status == "in_progress" || status == "completed" || status == "cancelled"
}

// getStatusIcon returns an icon for a status
func getStatusIcon(status string) string {
	switch status {
	case "pending":
		return "⏳"
	case "in_progress":
		return "🔄"
	case "completed":
		return "✅"
	case "cancelled":
		return "❌"
	default:
		return "❓"
	}
}

// FormatTodosForContext formats todos as a string for injection into LLM context
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

// MarshalTodos converts todos to JSON for storage/transmission
func MarshalTodos(todos []TodoItem) (string, error) {
	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
