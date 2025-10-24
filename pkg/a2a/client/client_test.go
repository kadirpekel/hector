package client

import (
	"context"
	"testing"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

type MockClient struct {
	SendMessageFunc   func(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error)
	StreamMessageFunc func(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error)
	ListAgentsFunc    func(ctx context.Context) ([]*pb.AgentCard, error)
	GetAgentCardFunc  func(ctx context.Context, agentID string) (*pb.AgentCard, error)
	CloseFunc         func() error
}

func (m *MockClient) SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(ctx, agentID, message)
	}
	return &pb.SendMessageResponse{}, nil
}

func (m *MockClient) StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {
	if m.StreamMessageFunc != nil {
		return m.StreamMessageFunc(ctx, agentID, message)
	}
	ch := make(chan *pb.StreamResponse)
	close(ch)
	return ch, nil
}

func (m *MockClient) ListAgents(ctx context.Context) ([]*pb.AgentCard, error) {
	if m.ListAgentsFunc != nil {
		return m.ListAgentsFunc(ctx)
	}
	return []*pb.AgentCard{}, nil
}

func (m *MockClient) GetAgentCard(ctx context.Context, agentID string) (*pb.AgentCard, error) {
	if m.GetAgentCardFunc != nil {
		return m.GetAgentCardFunc(ctx, agentID)
	}
	return &pb.AgentCard{}, nil
}

func (m *MockClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func TestMockClient_SendMessage(t *testing.T) {
	expectedResp := &pb.SendMessageResponse{
		Payload: &pb.SendMessageResponse_Msg{
			Msg: &pb.Message{
				Role: pb.Role_ROLE_AGENT,
				Content: []*pb.Part{
					{
						Part: &pb.Part_Text{Text: "Hello!"},
					},
				},
			},
		},
	}

	mock := &MockClient{
		SendMessageFunc: func(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
			if agentID != "test-agent" {
				t.Errorf("Expected agentID 'test-agent', got '%s'", agentID)
			}
			return expectedResp, nil
		},
	}

	msg := &pb.Message{
		Role: pb.Role_ROLE_USER,
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: "Test"},
			},
		},
	}

	resp, err := mock.SendMessage(context.Background(), "test-agent", msg)
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if resp != expectedResp {
		t.Error("SendMessage() did not return expected response")
	}
}

func TestMockClient_StreamMessage(t *testing.T) {
	mock := &MockClient{
		StreamMessageFunc: func(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {
			ch := make(chan *pb.StreamResponse, 1)
			ch <- &pb.StreamResponse{
				Payload: &pb.StreamResponse_Msg{
					Msg: &pb.Message{
						Content: []*pb.Part{
							{
								Part: &pb.Part_Text{Text: "Stream"},
							},
						},
					},
				},
			}
			close(ch)
			return ch, nil
		},
	}

	msg := &pb.Message{
		Role: pb.Role_ROLE_USER,
		Content: []*pb.Part{
			{
				Part: &pb.Part_Text{Text: "Test"},
			},
		},
	}

	streamChan, err := mock.StreamMessage(context.Background(), "test-agent", msg)
	if err != nil {
		t.Fatalf("StreamMessage() error = %v", err)
	}

	count := 0
	for range streamChan {
		count++
	}

	if count != 1 {
		t.Errorf("Expected 1 message in stream, got %d", count)
	}
}

func TestMockClient_ListAgents(t *testing.T) {
	expectedAgents := []*pb.AgentCard{
		{Name: "agent1", Description: "Agent 1"},
		{Name: "agent2", Description: "Agent 2"},
	}

	mock := &MockClient{
		ListAgentsFunc: func(ctx context.Context) ([]*pb.AgentCard, error) {
			return expectedAgents, nil
		},
	}

	agents, err := mock.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("ListAgents() error = %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}
}

func TestMockClient_GetAgentCard(t *testing.T) {
	expectedCard := &pb.AgentCard{
		Name:        "Test Agent",
		Description: "Test description",
		Version:     "1.0.0",
	}

	mock := &MockClient{
		GetAgentCardFunc: func(ctx context.Context, agentID string) (*pb.AgentCard, error) {
			if agentID != "test-agent" {
				t.Errorf("Expected agentID 'test-agent', got '%s'", agentID)
			}
			return expectedCard, nil
		},
	}

	card, err := mock.GetAgentCard(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("GetAgentCard() error = %v", err)
	}

	if card.Name != expectedCard.Name {
		t.Errorf("Expected name '%s', got '%s'", expectedCard.Name, card.Name)
	}
}

func TestMockClient_Close(t *testing.T) {
	closeCalled := false
	mock := &MockClient{
		CloseFunc: func() error {
			closeCalled = true
			return nil
		},
	}

	err := mock.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
	if !closeCalled {
		t.Error("Close() was not called")
	}
}

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient("http://localhost:8081", "test-token")
	if client == nil {
		t.Error("NewHTTPClient() returned nil")
	}

	var _ A2AClient = client

	client.Close()
}

func TestHTTPClient_Close(t *testing.T) {
	client := NewHTTPClient("http://localhost:8081", "test-token")
	err := client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
