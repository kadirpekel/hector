package a2a

import (
	"context"
	"testing"
	"time"
)

func TestTask_Validate(t *testing.T) {
	tests := []struct {
		name    string
		task    Task
		wantErr bool
	}{
		{
			name: "valid task",
			task: Task{
				ID: "test-task-1",
				Messages: []Message{
					{
						Role: MessageRoleUser,
						Parts: []Part{
							{Type: PartTypeText, Text: "Hello"},
						},
					},
				},
				Status: TaskStatus{
					State:     TaskStateSubmitted,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			wantErr: false,
		},
		{
			name: "empty task ID",
			task: Task{
				ID: "",
				Messages: []Message{
					{
						Role: MessageRoleUser,
						Parts: []Part{
							{Type: PartTypeText, Text: "Hello"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty messages",
			task: Task{
				ID:       "test-task-1",
				Messages: []Message{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation
			if tt.task.ID == "" && !tt.wantErr {
				t.Error("Task with empty ID should be invalid")
			}
			if len(tt.task.Messages) == 0 && !tt.wantErr {
				t.Error("Task with empty Messages should be invalid")
			}
		})
	}
}

func TestMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		message Message
		wantErr bool
	}{
		{
			name: "valid user message",
			message: Message{
				Role: MessageRoleUser,
				Parts: []Part{
					{Type: PartTypeText, Text: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid assistant message",
			message: Message{
				Role: MessageRoleAssistant,
				Parts: []Part{
					{Type: PartTypeText, Text: "Hi there!"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty parts",
			message: Message{
				Role:  MessageRoleUser,
				Parts: []Part{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.message.Parts) == 0 && !tt.wantErr {
				t.Error("Message with empty Parts should be invalid")
			}
		})
	}
}

func TestAgentCard_Validate(t *testing.T) {
	tests := []struct {
		name    string
		card    AgentCard
		wantErr bool
	}{
		{
			name: "valid agent card",
			card: AgentCard{
				Name:               "Test Agent",
				Description:        "A test agent",
				Version:            "1.0.0",
				PreferredTransport: "http+json",
				Capabilities: AgentCapabilities{
					Streaming: true,
					MultiTurn: true,
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			card: AgentCard{
				Name:        "",
				Description: "A test agent",
				Version:     "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "empty version",
			card: AgentCard{
				Name:        "Test Agent",
				Description: "A test agent",
				Version:     "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.card.Name == "" && !tt.wantErr {
				t.Error("AgentCard with empty Name should be invalid")
			}
			if tt.card.Version == "" && !tt.wantErr {
				t.Error("AgentCard with empty Version should be invalid")
			}
		})
	}
}

// MockAgent implements the Agent interface for testing
type MockAgent struct {
	card        *AgentCard
	executeFunc func(ctx context.Context, task *Task) (*Task, error)
}

func (m *MockAgent) GetAgentCard() *AgentCard {
	return m.card
}

func (m *MockAgent) ExecuteTask(ctx context.Context, task *Task) (*Task, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, task)
	}

	// Add assistant response
	task.Messages = append(task.Messages, Message{
		Role: MessageRoleAssistant,
		Parts: []Part{
			{Type: PartTypeText, Text: "Mock response"},
		},
	})
	task.Status.State = TaskStateCompleted
	task.Status.UpdatedAt = time.Now()

	return task, nil
}

func (m *MockAgent) ExecuteTaskStreaming(ctx context.Context, task *Task) (<-chan StreamEvent, error) {
	ch := make(chan StreamEvent, 2)

	go func() {
		defer close(ch)

		// Send message event
		ch <- StreamEvent{
			Type:   StreamEventTypeMessage,
			TaskID: task.ID,
			Message: &Message{
				Role: MessageRoleAssistant,
				Parts: []Part{
					{Type: PartTypeText, Text: "Mock streaming response"},
				},
			},
			Timestamp: time.Now(),
		}

		// Send completion status
		ch <- StreamEvent{
			Type:   StreamEventTypeStatus,
			TaskID: task.ID,
			Status: &TaskStatus{
				State:     TaskStateCompleted,
				UpdatedAt: time.Now(),
			},
			Timestamp: time.Now(),
		}
	}()

	return ch, nil
}

func TestMockAgent_ExecuteTask(t *testing.T) {
	agent := &MockAgent{
		card: &AgentCard{
			Name:        "Mock Agent",
			Description: "A mock agent for testing",
			Version:     "1.0.0",
			Capabilities: AgentCapabilities{
				Streaming: true,
			},
		},
	}

	ctx := context.Background()
	task := &Task{
		ID: "test-task",
		Messages: []Message{
			{
				Role: MessageRoleUser,
				Parts: []Part{
					{Type: PartTypeText, Text: "Test input"},
				},
			},
		},
		Status: TaskStatus{
			State:     TaskStateSubmitted,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	result, err := agent.ExecuteTask(ctx, task)
	if err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}

	if result.ID != task.ID {
		t.Errorf("ExecuteTask() ID = %v, want %v", result.ID, task.ID)
	}

	if result.Status.State != TaskStateCompleted {
		t.Errorf("ExecuteTask() State = %v, want %v", result.Status.State, TaskStateCompleted)
	}

	if len(result.Messages) < 2 {
		t.Errorf("ExecuteTask() should have added assistant message, got %d messages", len(result.Messages))
	}
}

func TestMockAgent_GetAgentCard(t *testing.T) {
	expectedCard := &AgentCard{
		Name:        "Mock Agent",
		Description: "A mock agent for testing",
		Version:     "1.0.0",
		Capabilities: AgentCapabilities{
			Streaming: true,
		},
	}

	agent := &MockAgent{card: expectedCard}
	card := agent.GetAgentCard()

	if card.Name != expectedCard.Name {
		t.Errorf("GetAgentCard() Name = %v, want %v", card.Name, expectedCard.Name)
	}

	if card.Version != expectedCard.Version {
		t.Errorf("GetAgentCard() Version = %v, want %v", card.Version, expectedCard.Version)
	}
}

func TestExtractTextFromTask(t *testing.T) {
	tests := []struct {
		name string
		task *Task
		want string
	}{
		{
			name: "extract from assistant messages",
			task: &Task{
				Messages: []Message{
					{
						Role: MessageRoleUser,
						Parts: []Part{
							{Type: PartTypeText, Text: "Hello"},
						},
					},
					{
						Role: MessageRoleAssistant,
						Parts: []Part{
							{Type: PartTypeText, Text: "Response 1"},
						},
					},
					{
						Role: MessageRoleAssistant,
						Parts: []Part{
							{Type: PartTypeText, Text: "Response 2"},
						},
					},
				},
			},
			want: "Response 1\nResponse 2",
		},
		{
			name: "empty task",
			task: &Task{},
			want: "",
		},
		{
			name: "nil task",
			task: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTextFromTask(tt.task)
			if got != tt.want {
				t.Errorf("ExtractTextFromTask() = %v, want %v", got, tt.want)
			}
		})
	}
}
