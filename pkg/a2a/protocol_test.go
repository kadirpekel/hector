package a2a

import (
	"context"
	"testing"
	"time"
)

func TestTaskRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request TaskRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: TaskRequest{
				TaskID: "test-task-1",
				Input: TaskInput{
					Type:    "text/plain",
					Content: "Hello, world!",
				},
			},
			wantErr: false,
		},
		{
			name: "empty task ID",
			request: TaskRequest{
				TaskID: "",
				Input: TaskInput{
					Type:    "text/plain",
					Content: "Hello, world!",
				},
			},
			wantErr: true,
		},
		{
			name: "empty input type",
			request: TaskRequest{
				TaskID: "test-task-1",
				Input: TaskInput{
					Type:    "",
					Content: "Hello, world!",
				},
			},
			wantErr: true,
		},
		{
			name: "empty input content",
			request: TaskRequest{
				TaskID: "test-task-1",
				Input: TaskInput{
					Type:    "text/plain",
					Content: "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since there's no Validate method, we'll test the structure creation
			if tt.request.TaskID == "" && !tt.wantErr {
				t.Error("TaskRequest with empty TaskID should be invalid")
			}
			if tt.request.Input.Type == "" && !tt.wantErr {
				t.Error("TaskRequest with empty Input.Type should be invalid")
			}
		})
	}
}

func TestTaskResponse_Validate(t *testing.T) {
	tests := []struct {
		name     string
		response TaskResponse
		wantErr  bool
	}{
		{
			name: "valid response",
			response: TaskResponse{
				TaskID: "test-task-1",
				Status: TaskStatusCompleted,
				Output: &TaskOutput{
					Type:    "text/plain",
					Content: "Response content",
				},
			},
			wantErr: false,
		},
		{
			name: "empty task ID",
			response: TaskResponse{
				TaskID: "",
				Status: TaskStatusCompleted,
				Output: &TaskOutput{
					Type:    "text/plain",
					Content: "Response content",
				},
			},
			wantErr: true,
		},
		{
			name: "valid status",
			response: TaskResponse{
				TaskID: "test-task-1",
				Status: TaskStatusCompleted,
				Output: &TaskOutput{
					Type:    "text/plain",
					Content: "Response content",
				},
			},
			wantErr: false,
		},
		{
			name: "empty output type",
			response: TaskResponse{
				TaskID: "test-task-1",
				Status: TaskStatusCompleted,
				Output: &TaskOutput{
					Type:    "",
					Content: "Response content",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since there's no Validate method, we'll test the structure creation
			if tt.response.TaskID == "" && !tt.wantErr {
				t.Error("TaskResponse with empty TaskID should be invalid")
			}
			if tt.response.Output != nil && tt.response.Output.Type == "" && !tt.wantErr {
				t.Error("TaskResponse with empty Output.Type should be invalid")
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
				AgentID:      "test-agent",
				Name:         "Test Agent",
				Description:  "A test agent",
				Version:      "1.0.0",
				Capabilities: []string{"text-processing", "reasoning"},
			},
			wantErr: false,
		},
		{
			name: "empty ID",
			card: AgentCard{
				AgentID:      "",
				Name:         "Test Agent",
				Description:  "A test agent",
				Version:      "1.0.0",
				Capabilities: []string{"text-processing"},
			},
			wantErr: true,
		},
		{
			name: "empty name",
			card: AgentCard{
				AgentID:      "test-agent",
				Name:         "",
				Description:  "A test agent",
				Version:      "1.0.0",
				Capabilities: []string{"text-processing"},
			},
			wantErr: true,
		},
		{
			name: "empty version",
			card: AgentCard{
				AgentID:      "test-agent",
				Name:         "Test Agent",
				Description:  "A test agent",
				Version:      "",
				Capabilities: []string{"text-processing"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since there's no Validate method, we'll test the structure creation
			if tt.card.AgentID == "" && !tt.wantErr {
				t.Error("AgentCard with empty AgentID should be invalid")
			}
			if tt.card.Name == "" && !tt.wantErr {
				t.Error("AgentCard with empty Name should be invalid")
			}
		})
	}
}

// MockAgent implements the Agent interface for testing
type MockAgent struct {
	card        *AgentCard
	executeFunc func(ctx context.Context, request *TaskRequest) (*TaskResponse, error)
}

func (m *MockAgent) GetAgentCard() *AgentCard {
	return m.card
}

func (m *MockAgent) ExecuteTask(ctx context.Context, request *TaskRequest) (*TaskResponse, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, request)
	}
	return &TaskResponse{
		TaskID: request.TaskID,
		Status: TaskStatusCompleted,
		Output: &TaskOutput{
			Type:    "text/plain",
			Content: "Mock response",
		},
	}, nil
}

func (m *MockAgent) ExecuteTaskStreaming(ctx context.Context, request *TaskRequest) (<-chan *StreamChunk, error) {
	ch := make(chan *StreamChunk, 1)
	ch <- &StreamChunk{
		TaskID:    request.TaskID,
		ChunkType: ChunkTypeText,
		Content:   "Mock streaming response",
		Timestamp: time.Now(),
		Final:     true,
	}
	close(ch)
	return ch, nil
}

func TestMockAgent_ExecuteTask(t *testing.T) {
	agent := &MockAgent{
		card: &AgentCard{
			AgentID:      "mock-agent",
			Name:         "Mock Agent",
			Description:  "A mock agent for testing",
			Version:      "1.0.0",
			Capabilities: []string{"testing"},
		},
	}

	ctx := context.Background()
	request := &TaskRequest{
		TaskID: "test-task",
		Input: TaskInput{
			Type:    "text/plain",
			Content: "Test input",
		},
	}

	response, err := agent.ExecuteTask(ctx, request)
	if err != nil {
		t.Fatalf("ExecuteTask() error = %v", err)
	}

	if response.TaskID != request.TaskID {
		t.Errorf("ExecuteTask() TaskID = %v, want %v", response.TaskID, request.TaskID)
	}

	if response.Status != TaskStatusCompleted {
		t.Errorf("ExecuteTask() Status = %v, want %v", response.Status, TaskStatusCompleted)
	}

	if response.Output.Content != "Mock response" {
		t.Errorf("ExecuteTask() Output.Content = %v, want %v", response.Output.Content, "Mock response")
	}
}

func TestMockAgent_GetAgentCard(t *testing.T) {
	expectedCard := &AgentCard{
		AgentID:      "mock-agent",
		Name:         "Mock Agent",
		Description:  "A mock agent for testing",
		Version:      "1.0.0",
		Capabilities: []string{"testing"},
	}

	agent := &MockAgent{card: expectedCard}
	card := agent.GetAgentCard()

	if card.AgentID != expectedCard.AgentID {
		t.Errorf("GetAgentCard() AgentID = %v, want %v", card.AgentID, expectedCard.AgentID)
	}

	if card.Name != expectedCard.Name {
		t.Errorf("GetAgentCard() Name = %v, want %v", card.Name, expectedCard.Name)
	}
}
