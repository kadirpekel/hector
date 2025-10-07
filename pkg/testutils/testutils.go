// Package testutils provides testing utilities for the Hector framework.
package testutils

import (
	"context"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/kadirpekel/hector/pkg/config"
)

// TestConfig returns a minimal valid configuration for testing
func TestConfig() *config.Config {
	return &config.Config{
		Agents: map[string]config.AgentConfig{
			"test-agent": {
				Name: "Test Agent",
				LLM:  "test-llm",
			},
		},
		LLMs: map[string]config.LLMProviderConfig{
			"test-llm": {
				Type:  "openai",
				Model: "gpt-4o-mini",
			},
		},
	}
}

// TestAgentConfig returns a minimal valid agent configuration for testing
func TestAgentConfig() *config.AgentConfig {
	return &config.AgentConfig{
		Name: "Test Agent",
		LLM:  "test-llm",
	}
}

// TestLLMConfig returns a minimal valid LLM configuration for testing
func TestLLMConfig() *config.LLMProviderConfig {
	return &config.LLMProviderConfig{
		Type:  "openai",
		Model: "gpt-4o-mini",
	}
}

// TestTask returns a valid A2A task for testing
func TestTask() *a2a.Task {
	return &a2a.Task{
		ID: "test-task-1",
		Messages: []a2a.Message{
			{
				Role: a2a.MessageRoleUser,
				Parts: []a2a.Part{
					{
						Type: a2a.PartTypeText,
						Text: "Test input content",
					},
				},
			},
		},
		Status: a2a.TaskStatus{
			State:     a2a.TaskStateSubmitted,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

// TestTaskWithResponse returns a completed A2A task for testing
func TestTaskWithResponse() *a2a.Task {
	task := TestTask()
	task.Messages = append(task.Messages, a2a.Message{
		Role: a2a.MessageRoleAssistant,
		Parts: []a2a.Part{
			{
				Type: a2a.PartTypeText,
				Text: "Test response content",
			},
		},
	})
	task.Status.State = a2a.TaskStateCompleted
	task.Status.UpdatedAt = time.Now()
	return task
}

// TestAgentCard returns a valid agent card for testing
func TestAgentCard() *a2a.AgentCard {
	return &a2a.AgentCard{
		Name:               "Test Agent",
		Description:        "A test agent for unit testing",
		Version:            "1.0.0",
		PreferredTransport: "http+json",
		Capabilities: a2a.AgentCapabilities{
			Streaming: true,
			MultiTurn: true,
		},
	}
}

// TestContext returns a context with timeout for testing
func TestContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// Note: We don't call cancel here because this is a test utility
	// that returns a context for immediate use. The context will be
	// automatically cancelled when the timeout expires.
	_ = cancel // Explicitly ignore to satisfy linter
	return ctx
}

// TestContextWithTimeout returns a context with custom timeout for testing
func TestContextWithTimeout(timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// Note: We don't call cancel here because this is a test utility
	// that returns a context for immediate use. The context will be
	// automatically cancelled when the timeout expires.
	_ = cancel // Explicitly ignore to satisfy linter
	return ctx
}

// MockAgent implements the a2a.Agent interface for testing
type MockAgent struct {
	Card         *a2a.AgentCard
	ExecuteFunc  func(ctx context.Context, task *a2a.Task) (*a2a.Task, error)
	ExecuteDelay time.Duration
	ExecuteError error
}

// NewMockAgent creates a new mock agent with default behavior
func NewMockAgent() *MockAgent {
	return &MockAgent{
		Card: TestAgentCard(),
		ExecuteFunc: func(ctx context.Context, task *a2a.Task) (*a2a.Task, error) {
			// Extract user message text
			userText := ""
			if len(task.Messages) > 0 {
				lastMsg := task.Messages[len(task.Messages)-1]
				if lastMsg.Role == a2a.MessageRoleUser && len(lastMsg.Parts) > 0 {
					if lastMsg.Parts[0].Type == a2a.PartTypeText {
						userText = lastMsg.Parts[0].Text
					}
				}
			}

			// Add assistant response
			task.Messages = append(task.Messages, a2a.Message{
				Role: a2a.MessageRoleAssistant,
				Parts: []a2a.Part{
					{
						Type: a2a.PartTypeText,
						Text: "Mock response for: " + userText,
					},
				},
			})
			task.Status.State = a2a.TaskStateCompleted
			task.Status.UpdatedAt = time.Now()
			return task, nil
		},
	}
}

// GetAgentCard returns the agent's capability card
func (m *MockAgent) GetAgentCard() *a2a.AgentCard {
	return m.Card
}

// ExecuteTask executes a task and returns the response
func (m *MockAgent) ExecuteTask(ctx context.Context, task *a2a.Task) (*a2a.Task, error) {
	// Add delay if specified
	if m.ExecuteDelay > 0 {
		select {
		case <-time.After(m.ExecuteDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Return error if specified
	if m.ExecuteError != nil {
		return nil, m.ExecuteError
	}

	// Use custom function if provided
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, task)
	}

	// Default behavior
	task.Messages = append(task.Messages, a2a.Message{
		Role: a2a.MessageRoleAssistant,
		Parts: []a2a.Part{
			{
				Type: a2a.PartTypeText,
				Text: "Mock response",
			},
		},
	})
	task.Status.State = a2a.TaskStateCompleted
	task.Status.UpdatedAt = time.Now()
	return task, nil
}

// SetExecuteError sets an error to be returned by ExecuteTask
func (m *MockAgent) SetExecuteError(err error) {
	m.ExecuteError = err
}

// SetExecuteDelay sets a delay for ExecuteTask
func (m *MockAgent) SetExecuteDelay(delay time.Duration) {
	m.ExecuteDelay = delay
}

// ExecuteTaskStreaming executes a task with streaming output
func (m *MockAgent) ExecuteTaskStreaming(ctx context.Context, task *a2a.Task) (<-chan a2a.StreamEvent, error) {
	ch := make(chan a2a.StreamEvent, 2)

	go func() {
		defer close(ch)

		// Send message event
		ch <- a2a.StreamEvent{
			Type:   a2a.StreamEventTypeMessage,
			TaskID: task.ID,
			Message: &a2a.Message{
				Role: a2a.MessageRoleAssistant,
				Parts: []a2a.Part{
					{
						Type: a2a.PartTypeText,
						Text: "Mock streaming response",
					},
				},
			},
			Timestamp: time.Now(),
		}

		// Send completion status
		ch <- a2a.StreamEvent{
			Type:   a2a.StreamEventTypeStatus,
			TaskID: task.ID,
			Status: &a2a.TaskStatus{
				State:     a2a.TaskStateCompleted,
				UpdatedAt: time.Now(),
			},
			Timestamp: time.Now(),
		}
	}()

	return ch, nil
}

// SetExecuteFunc sets a custom function for ExecuteTask
func (m *MockAgent) SetExecuteFunc(fn func(ctx context.Context, task *a2a.Task) (*a2a.Task, error)) {
	m.ExecuteFunc = fn
}
