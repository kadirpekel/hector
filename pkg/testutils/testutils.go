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

// TestTaskRequest returns a valid task request for testing
func TestTaskRequest() *a2a.TaskRequest {
	return &a2a.TaskRequest{
		TaskID: "test-task-1",
		Input: a2a.TaskInput{
			Type:    "text/plain",
			Content: "Test input content",
		},
	}
}

// TestTaskResponse returns a valid task response for testing
func TestTaskResponse() *a2a.TaskResponse {
	return &a2a.TaskResponse{
		TaskID: "test-task-1",
		Status: a2a.TaskStatusCompleted,
		Output: &a2a.TaskOutput{
			Type:    "text/plain",
			Content: "Test response content",
		},
	}
}

// TestAgentCard returns a valid agent card for testing
func TestAgentCard() *a2a.AgentCard {
	return &a2a.AgentCard{
		AgentID:      "test-agent",
		Name:         "Test Agent",
		Description:  "A test agent for unit testing",
		Version:      "1.0.0",
		Capabilities: []string{"testing", "mock-responses"},
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
	ExecuteFunc  func(ctx context.Context, request *a2a.TaskRequest) (*a2a.TaskResponse, error)
	ExecuteDelay time.Duration
	ExecuteError error
}

// NewMockAgent creates a new mock agent with default behavior
func NewMockAgent() *MockAgent {
	return &MockAgent{
		Card: TestAgentCard(),
		ExecuteFunc: func(ctx context.Context, request *a2a.TaskRequest) (*a2a.TaskResponse, error) {
			content := "Mock response"
			if request.Input.Content != nil {
				if contentStr, ok := request.Input.Content.(string); ok {
					content = "Mock response for: " + contentStr
				}
			}
			return &a2a.TaskResponse{
				TaskID: request.TaskID,
				Status: a2a.TaskStatusCompleted,
				Output: &a2a.TaskOutput{
					Type:    "text/plain",
					Content: content,
				},
			}, nil
		},
	}
}

// GetAgentCard returns the agent's capability card
func (m *MockAgent) GetAgentCard() *a2a.AgentCard {
	return m.Card
}

// ExecuteTask executes a task and returns the response
func (m *MockAgent) ExecuteTask(ctx context.Context, request *a2a.TaskRequest) (*a2a.TaskResponse, error) {
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
		return m.ExecuteFunc(ctx, request)
	}

	// Default behavior
	return &a2a.TaskResponse{
		TaskID: request.TaskID,
		Status: a2a.TaskStatusCompleted,
		Output: &a2a.TaskOutput{
			Type:    "text/plain",
			Content: "Mock response",
		},
	}, nil
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
func (m *MockAgent) ExecuteTaskStreaming(ctx context.Context, request *a2a.TaskRequest) (<-chan *a2a.StreamChunk, error) {
	ch := make(chan *a2a.StreamChunk, 1)
	ch <- &a2a.StreamChunk{
		TaskID:    request.TaskID,
		ChunkType: a2a.ChunkTypeText,
		Content:   "Mock streaming response",
		Timestamp: time.Now(),
		Final:     true,
	}
	close(ch)
	return ch, nil
}

// SetExecuteFunc sets a custom function for ExecuteTask
func (m *MockAgent) SetExecuteFunc(fn func(ctx context.Context, request *a2a.TaskRequest) (*a2a.TaskResponse, error)) {
	m.ExecuteFunc = fn
}
