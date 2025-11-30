package cli

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/logger"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestDisplayAgentList(t *testing.T) {
	agents := []*pb.AgentCard{
		{
			Name:        "Test Agent 1",
			Description: "Test description 1",
			Url:         "http://localhost:8080",
			Version:     "1.0.0",
			Capabilities: &pb.AgentCapabilities{
				Streaming: true,
			},
		},
		{
			Name:        "Test Agent 2",
			Description: "Test description 2",
			Version:     "2.0.0",
		},
	}

	output := captureOutput(func() {
		DisplayAgentList(agents, "Test Mode")
	})

	if !strings.Contains(output, "Test Mode") {
		t.Error("Output should contain mode")
	}
	if !strings.Contains(output, "Test Agent 1") {
		t.Error("Output should contain agent 1 name")
	}
	if !strings.Contains(output, "Test Agent 2") {
		t.Error("Output should contain agent 2 name")
	}
	if !strings.Contains(output, "Test description 1") {
		t.Error("Output should contain agent 1 description")
	}
	if !strings.Contains(output, "2 agent(s)") {
		t.Error("Output should contain agent count")
	}
}

func TestDisplayAgentCard(t *testing.T) {
	card := &pb.AgentCard{
		Name:        "Test Agent",
		Description: "Test description",
		Version:     "1.0.0",
		Capabilities: &pb.AgentCapabilities{
			Streaming: true,
		},
	}

	output := captureOutput(func() {
		DisplayAgentCard("test-id", card)
	})

	if !strings.Contains(output, "test-id") {
		t.Error("Output should contain agent ID")
	}
	if !strings.Contains(output, "Test Agent") {
		t.Error("Output should contain agent name")
	}
	if !strings.Contains(output, "Test description") {
		t.Error("Output should contain description")
	}
	if !strings.Contains(output, "1.0.0") {
		t.Error("Output should contain version")
	}
	if !strings.Contains(output, "true") {
		t.Error("Output should contain streaming capability")
	}
}

func TestDisplayMessage(t *testing.T) {
	msg := &pb.Message{
		Role: pb.Role_ROLE_AGENT,
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: "Hello, world!"},
			},
		},
	}

	output := captureOutput(func() {
		DisplayMessage(msg, "Agent: ", false, false)
	})

	if !strings.Contains(output, "Agent:") {
		t.Error("Output should contain prefix")
	}
	if !strings.Contains(output, "Hello, world!") {
		t.Error("Output should contain message text")
	}
}

func TestDisplayMessage_NilMessage(t *testing.T) {
	output := captureOutput(func() {
		DisplayMessage(nil, "Agent: ", false, false)
	})

	if output != "" {
		t.Error("Output should be empty for nil message")
	}
}

func TestDisplayMessage_NoPrefix(t *testing.T) {
	msg := &pb.Message{
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: "Test"},
			},
		},
	}

	output := captureOutput(func() {
		DisplayMessage(msg, "", false, false)
	})

	if !strings.Contains(output, "Test") {
		t.Error("Output should contain message text")
	}
}

func TestDisplayMessageLine(t *testing.T) {
	msg := &pb.Message{
		Parts: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: "Test message"},
			},
		},
	}

	output := captureOutput(func() {
		DisplayMessageLine(msg, "Prefix: ", false, false)
	})

	if !strings.Contains(output, "Prefix:") {
		t.Error("Output should contain prefix")
	}
	if !strings.Contains(output, "Test message") {
		t.Error("Output should contain message text")
	}
	if !strings.Contains(output, "\n") {
		t.Error("Output should contain newline")
	}
}

func TestDisplayTask(t *testing.T) {
	task := &pb.Task{
		Id: "task-123",
		Status: &pb.TaskStatus{
			State: pb.TaskState_TASK_STATE_SUBMITTED,
		},
	}

	output := captureOutput(func() {
		DisplayTask(task)
	})

	if !strings.Contains(output, "task-123") {
		t.Error("Output should contain task ID")
	}
}

func TestDisplayError(t *testing.T) {
	// Initialize logger for test - use a temporary file to capture output
	tmpFile, err := os.CreateTemp("", "test-log-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	logger.Init(slog.LevelDebug, tmpFile, "simple")

	testErr := &testError{msg: "test error message"}

	DisplayError(testErr)

	// Read the log output
	_, _ = tmpFile.Seek(0, 0)
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, tmpFile)
	output := buf.String()

	if !strings.Contains(output, "test error message") {
		t.Errorf("Output should contain error message, got: %q", output)
	}
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Output should contain ERROR level, got: %q", output)
	}
}

func TestDisplayStreamingStart(t *testing.T) {
	output := captureOutput(func() {
		DisplayStreamingStart("test-agent", "Test Mode")
	})

	if !strings.Contains(output, "test-agent") {
		t.Error("Output should contain agent ID")
	}
	if !strings.Contains(output, "Test Mode") {
		t.Error("Output should contain mode")
	}
	if !strings.Contains(output, "streaming") {
		t.Error("Output should mention streaming")
	}
}

func TestDisplayGoodbye(t *testing.T) {
	output := captureOutput(func() {
		DisplayGoodbye()
	})

	if !strings.Contains(output, "Goodbye") {
		t.Error("Output should contain goodbye message")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// ============================================================================
// Todo Display Tests
// ============================================================================

func TestDisplayTodosInlineCLI_EmptyTodos(t *testing.T) {
	ResetDisplayState()

	output := captureOutput(func() {
		displayTodosInlineCLI()
	})

	if output != "" {
		t.Errorf("Empty todos should produce no output, got: %q", output)
	}
}

func TestDisplayTodosInlineCLI_SingleTodo(t *testing.T) {
	ResetDisplayState()

	// Add a single todo
	accumulatedTodos["1"] = map[string]interface{}{
		"id":      "1",
		"content": "Test task",
		"status":  "pending",
	}
	todoInsertOrder = append(todoInsertOrder, "1")

	output := captureOutput(func() {
		displayTodosInlineCLI()
	})

	if !strings.Contains(output, "Test task") {
		t.Error("Output should contain todo content")
	}
	if !strings.Contains(output, "Tasks") {
		t.Error("Output should contain Tasks header")
	}
}

func TestDisplayTodosInlineCLI_OrderPreservation(t *testing.T) {
	ResetDisplayState()

	// Add todos in specific order
	accumulatedTodos["c"] = map[string]interface{}{"id": "c", "content": "Third", "status": "pending"}
	accumulatedTodos["a"] = map[string]interface{}{"id": "a", "content": "First", "status": "pending"}
	accumulatedTodos["b"] = map[string]interface{}{"id": "b", "content": "Second", "status": "pending"}
	todoInsertOrder = []string{"a", "b", "c"}

	output := captureOutput(func() {
		displayTodosInlineCLI()
	})

	// Check that order is preserved (First should appear before Second, etc.)
	firstIdx := strings.Index(output, "First")
	secondIdx := strings.Index(output, "Second")
	thirdIdx := strings.Index(output, "Third")

	if firstIdx == -1 || secondIdx == -1 || thirdIdx == -1 {
		t.Fatalf("All todos should be in output. Got: %q", output)
	}

	if firstIdx >= secondIdx || secondIdx >= thirdIdx {
		t.Errorf("Todos should appear in insertion order (a, b, c). First=%d, Second=%d, Third=%d",
			firstIdx, secondIdx, thirdIdx)
	}
}

func TestDisplayTodosInlineCLI_StatusIcons(t *testing.T) {
	testCases := []struct {
		status   string
		expected string
	}{
		{"completed", "✓"},
		{"in_progress", "⧗"},
		{"pending", "○"},
		{"canceled", "✗"},
	}

	for _, tc := range testCases {
		t.Run(tc.status, func(t *testing.T) {
			ResetDisplayState()
			accumulatedTodos["1"] = map[string]interface{}{
				"id":      "1",
				"content": "Test",
				"status":  tc.status,
			}
			todoInsertOrder = []string{"1"}

			output := captureOutput(func() {
				displayTodosInlineCLI()
			})

			if !strings.Contains(output, tc.expected) {
				t.Errorf("Status %q should show icon %q, got: %q", tc.status, tc.expected, output)
			}
		})
	}
}

func TestDisplayTodosInlineCLI_MoreIndicator(t *testing.T) {
	ResetDisplayState()

	// Add more than visibleTodosCount (4) todos
	for i := 1; i <= 7; i++ {
		id := string(rune('0' + i))
		accumulatedTodos[id] = map[string]interface{}{
			"id":      id,
			"content": "Task " + id,
			"status":  "pending",
		}
		todoInsertOrder = append(todoInsertOrder, id)
	}

	output := captureOutput(func() {
		displayTodosInlineCLI()
	})

	// Should show "... (3 more)" for 7 - 4 = 3 hidden items
	if !strings.Contains(output, "3 more") {
		t.Errorf("Should show '3 more' indicator for 7 todos. Got: %q", output)
	}

	// Should only show first 4 tasks
	if !strings.Contains(output, "Task 1") {
		t.Error("Should show first task")
	}
	if strings.Contains(output, "Task 5") {
		t.Error("Should NOT show fifth task (beyond visible limit)")
	}
}

func TestDisplayTodosInlineCLI_FallbackSorting(t *testing.T) {
	ResetDisplayState()

	// Add todos but with mismatched todoInsertOrder (simulates corruption)
	accumulatedTodos["b"] = map[string]interface{}{"id": "b", "content": "B task", "status": "pending"}
	accumulatedTodos["a"] = map[string]interface{}{"id": "a", "content": "A task", "status": "pending"}
	accumulatedTodos["c"] = map[string]interface{}{"id": "c", "content": "C task", "status": "pending"}
	// Intentionally wrong order to trigger fallback
	todoInsertOrder = []string{"x", "y"} // IDs that don't exist

	output := captureOutput(func() {
		displayTodosInlineCLI()
	})

	// Should fall back to alphabetical sorting
	aIdx := strings.Index(output, "A task")
	bIdx := strings.Index(output, "B task")
	cIdx := strings.Index(output, "C task")

	if aIdx == -1 || bIdx == -1 || cIdx == -1 {
		t.Fatalf("All todos should be in output. Got: %q", output)
	}

	if aIdx >= bIdx || bIdx >= cIdx {
		t.Errorf("Fallback should sort alphabetically. A=%d, B=%d, C=%d", aIdx, bIdx, cIdx)
	}
}

// ============================================================================
// Concurrency Tests
// ============================================================================

func TestDisplayMessage_ConcurrentAccess(t *testing.T) {
	ResetDisplayState()

	// Redirect stdout once before starting goroutines to avoid race on os.Stdout
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Skip("Cannot open /dev/null")
	}
	defer devNull.Close()

	oldStdout := os.Stdout
	os.Stdout = devNull
	defer func() { os.Stdout = oldStdout }()

	// Run multiple goroutines calling DisplayMessage concurrently
	// This test verifies no race conditions in internal state (run with -race flag)
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			msg := &pb.Message{
				Parts: []*pb.Part{
					{Part: &pb.Part_Text{Text: "Message from goroutine"}},
				},
			}
			DisplayMessage(msg, "", false, false)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestResetDisplayState_ConcurrentAccess(t *testing.T) {
	// Redirect stdout once before starting goroutines
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Skip("Cannot open /dev/null")
	}
	defer devNull.Close()

	oldStdout := os.Stdout
	os.Stdout = devNull
	defer func() { os.Stdout = oldStdout }()

	// Run reset and display concurrently to verify no race conditions
	done := make(chan bool, 20)

	for i := 0; i < 10; i++ {
		go func() {
			ResetDisplayState()
			done <- true
		}()
		go func() {
			msg := &pb.Message{
				Parts: []*pb.Part{
					{Part: &pb.Part_Text{Text: "Test"}},
				},
			}
			DisplayMessage(msg, "", false, false)
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}

// ============================================================================
// Memory Management Tests
// ============================================================================

func TestTodoAccumulation_FIFOEviction(t *testing.T) {
	ResetDisplayState()

	// Manually test FIFO eviction by adding more than maxAccumulatedTodos
	// Note: This tests the eviction logic directly, not through DisplayMessage

	// Add maxAccumulatedTodos + 10 items
	for i := 1; i <= maxAccumulatedTodos+10; i++ {
		id := string(rune('a'-1)+rune(i%26+1)) + string(rune('0'+i/26))

		// Track insertion order for new todos
		if _, exists := accumulatedTodos[id]; !exists {
			todoInsertOrder = append(todoInsertOrder, id)
			// FIFO eviction when over limit
			if len(accumulatedTodos) >= maxAccumulatedTodos {
				if len(todoInsertOrder) > 0 {
					oldestID := todoInsertOrder[0]
					delete(accumulatedTodos, oldestID)
					todoInsertOrder = todoInsertOrder[1:]
				}
			}
		}
		accumulatedTodos[id] = map[string]interface{}{
			"id":      id,
			"content": "Task " + id,
			"status":  "pending",
		}
	}

	// Should be capped at maxAccumulatedTodos
	if len(accumulatedTodos) > maxAccumulatedTodos {
		t.Errorf("Todo count should be capped at %d, got %d", maxAccumulatedTodos, len(accumulatedTodos))
	}

	// todoInsertOrder should match accumulatedTodos length
	if len(todoInsertOrder) != len(accumulatedTodos) {
		t.Errorf("todoInsertOrder length (%d) should match accumulatedTodos (%d)",
			len(todoInsertOrder), len(accumulatedTodos))
	}
}
