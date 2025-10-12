package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a/client"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// captureOutput captures stdout during function execution
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
	agents := []client.AgentInfo{
		{
			ID:          "agent1",
			Name:        "Test Agent 1",
			Description: "Test description 1",
			Endpoint:    "http://localhost:50052",
		},
		{
			ID:          "agent2",
			Name:        "Test Agent 2",
			Description: "Test description 2",
		},
	}

	output := captureOutput(func() {
		DisplayAgentList(agents, "Test Mode")
	})

	// Verify output contains expected information
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

	// Verify output contains expected information
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
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: "Hello, world!"},
			},
		},
	}

	output := captureOutput(func() {
		DisplayMessage(msg, "Agent: ")
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
		DisplayMessage(nil, "Agent: ")
	})

	if output != "" {
		t.Error("Output should be empty for nil message")
	}
}

func TestDisplayMessage_NoPrefix(t *testing.T) {
	msg := &pb.Message{
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: "Test"},
			},
		},
	}

	output := captureOutput(func() {
		DisplayMessage(msg, "")
	})

	if !strings.Contains(output, "Test") {
		t.Error("Output should contain message text")
	}
}

func TestDisplayMessageLine(t *testing.T) {
	msg := &pb.Message{
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: "Test message"},
			},
		},
	}

	output := captureOutput(func() {
		DisplayMessageLine(msg, "Prefix: ")
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
	err := &testError{msg: "test error message"}

	output := captureOutput(func() {
		DisplayError(err)
	})

	if !strings.Contains(output, "test error message") {
		t.Error("Output should contain error message")
	}
	if !strings.Contains(output, "❌") {
		t.Error("Output should contain error icon")
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

	if !strings.Contains(output, "Goodbye") || !strings.Contains(output, "👋") {
		t.Error("Output should contain goodbye message")
	}
}

// Test error type
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
