//go:build integration

package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ============================================================================
// A2A CLIENT TESTS - HTTP+JSON Transport
// ============================================================================

func TestNewClient(t *testing.T) {
	cfg := &ClientConfig{
		Timeout: 30 * time.Second,
	}

	client := NewClient(cfg)

	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.httpClient.Timeout)
	}
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	client := NewClient(nil)

	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}

	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("Expected default timeout 60s, got %v", client.httpClient.Timeout)
	}
}

func TestNewClient_WithAuth(t *testing.T) {
	cfg := &ClientConfig{
		Auth: &AuthCredentials{
			Type:  "bearer",
			Token: "test-token-123",
		},
	}

	client := NewClient(cfg)

	if client.auth == nil {
		t.Error("auth is nil")
	}

	if client.auth.Token != "test-token-123" {
		t.Errorf("Expected token 'test-token-123', got '%s'", client.auth.Token)
	}
}

func TestClient_DiscoverAgent(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/agents/test-agent" {
			t.Errorf("Expected path /agents/test-agent, got %s", r.URL.Path)
		}

		card := AgentCard{
			Name:        "test-agent",
			Description: "Test agent for discovery",
			Version:     "1.0.0",
			URL:         "http://localhost:8080/agents/test-agent",
			Capabilities: AgentCapabilities{
				Streaming: true,
				MultiTurn: true,
			},
		}

		_ = json.NewEncoder(w).Encode(card)
	}))
	defer ts.Close()

	client := NewClient(nil)
	card, err := client.DiscoverAgent(context.Background(), ts.URL+"/agents/test-agent")

	if err != nil {
		t.Fatalf("DiscoverAgent failed: %v", err)
	}

	if card.Name != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", card.Name)
	}

	if !card.Capabilities.Streaming {
		t.Error("Expected streaming capability")
	}
}

func TestClient_SendMessage(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/agents/test-agent/message/send" {
			t.Errorf("Expected path /agents/test-agent/message/send, got %s", r.URL.Path)
		}

		// Parse request
		var params MessageSendParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if params.Message.Role != MessageRoleUser {
			t.Errorf("Expected user message, got %s", params.Message.Role)
		}

		// Return task
		task := Task{
			ID: "task-123",
			Messages: []Message{
				params.Message,
				{
					Role: MessageRoleAssistant,
					Parts: []Part{
						{Type: PartTypeText, Text: "Response from agent"},
					},
				},
			},
			Status: TaskStatus{
				State:     TaskStateCompleted,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(task)
	}))
	defer ts.Close()

	client := NewClient(nil)
	message := Message{
		Role: MessageRoleUser,
		Parts: []Part{
			{Type: PartTypeText, Text: "Hello agent"},
		},
	}

	task, err := client.SendMessage(context.Background(), ts.URL+"/agents/test-agent", message, nil)

	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if task.ID != "task-123" {
		t.Errorf("Expected task ID 'task-123', got '%s'", task.ID)
	}

	if len(task.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(task.Messages))
	}

	if task.Status.State != TaskStateCompleted {
		t.Errorf("Expected completed state, got %s", task.Status.State)
	}
}

func TestClient_SendMessage_WithAuth(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", authHeader)
		}

		task := Task{
			ID: "task-456",
			Status: TaskStatus{
				State:     TaskStateCompleted,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(task)
	}))
	defer ts.Close()

	client := NewClient(&ClientConfig{
		Auth: &AuthCredentials{
			Type:  "bearer",
			Token: "test-token",
		},
	})

	message := Message{
		Role: MessageRoleUser,
		Parts: []Part{
			{Type: PartTypeText, Text: "Authenticated request"},
		},
	}

	_, err := client.SendMessage(context.Background(), ts.URL+"/agents/test-agent", message, nil)

	if err != nil {
		t.Fatalf("SendMessage with auth failed: %v", err)
	}
}

func TestClient_SendMessage_WithAPIKey(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check API key header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "test-api-key" {
			t.Errorf("Expected X-API-Key header 'test-api-key', got '%s'", apiKey)
		}

		task := Task{
			ID: "task-789",
			Status: TaskStatus{
				State:     TaskStateCompleted,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(task)
	}))
	defer ts.Close()

	client := NewClient(&ClientConfig{
		Auth: &AuthCredentials{
			Type:   "apiKey",
			APIKey: "test-api-key",
		},
	})

	message := Message{
		Role: MessageRoleUser,
		Parts: []Part{
			{Type: PartTypeText, Text: "API key request"},
		},
	}

	_, err := client.SendMessage(context.Background(), ts.URL+"/agents/test-agent", message, nil)

	if err != nil {
		t.Fatalf("SendMessage with API key failed: %v", err)
	}
}

func TestClient_SendMessageStreaming(t *testing.T) {
	// Create test server that sends SSE events
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Send status event
		statusEvent := TaskStatusUpdateEvent{
			TaskID: "stream-task-123",
			Status: TaskStatus{
				State:     TaskStateWorking,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		statusData, _ := json.Marshal(statusEvent)
		w.Write([]byte("event: status\n"))
		w.Write([]byte("data: " + string(statusData) + "\n\n"))
		flusher.Flush()

		// Send message event
		msgEvent := TaskMessageEvent{
			TaskID: "stream-task-123",
			Message: Message{
				Role: MessageRoleAssistant,
				Parts: []Part{
					{Type: PartTypeText, Text: "Streaming response"},
				},
			},
		}
		msgData, _ := json.Marshal(msgEvent)
		w.Write([]byte("event: message\n"))
		w.Write([]byte("data: " + string(msgData) + "\n\n"))
		flusher.Flush()

		// Send completion
		completeEvent := TaskStatusUpdateEvent{
			TaskID: "stream-task-123",
			Status: TaskStatus{
				State:     TaskStateCompleted,
				UpdatedAt: time.Now(),
			},
		}
		completeData, _ := json.Marshal(completeEvent)
		w.Write([]byte("event: status\n"))
		w.Write([]byte("data: " + string(completeData) + "\n\n"))
		flusher.Flush()
	}))
	defer ts.Close()

	client := NewClient(nil)
	message := Message{
		Role: MessageRoleUser,
		Parts: []Part{
			{Type: PartTypeText, Text: "Stream this"},
		},
	}

	eventCh, err := client.SendMessageStreaming(context.Background(), ts.URL+"/agents/test-agent", message)

	if err != nil {
		t.Fatalf("SendMessageStreaming failed: %v", err)
	}

	eventCount := 0
	hasMessage := false

	for event := range eventCh {
		eventCount++

		if event.Type == StreamEventTypeMessage {
			hasMessage = true
			if len(event.Message.Parts) > 0 && event.Message.Parts[0].Text != "Streaming response" {
				t.Errorf("Expected 'Streaming response', got '%s'", event.Message.Parts[0].Text)
			}
		}

		// StreamEvent doesn't have an Error type in the current protocol
	}

	if eventCount == 0 {
		t.Error("No events received from stream")
	}

	if !hasMessage {
		t.Error("No message event received")
	}
}

func TestClient_GetTask(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		task := Task{
			ID: "task-get-123",
			Messages: []Message{
				{
					Role: MessageRoleUser,
					Parts: []Part{
						{Type: PartTypeText, Text: "Original message"},
					},
				},
			},
			Status: TaskStatus{
				State:     TaskStateCompleted,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		_ = json.NewEncoder(w).Encode(task)
	}))
	defer ts.Close()

	client := NewClient(nil)
	task, err := client.GetTask(context.Background(), ts.URL+"/agents/test-agent", "task-get-123")

	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if task.ID != "task-get-123" {
		t.Errorf("Expected task ID 'task-get-123', got '%s'", task.ID)
	}

	if task.Status.State != TaskStateCompleted {
		t.Errorf("Expected completed state, got %s", task.Status.State)
	}
}

func TestClient_CancelTask(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/agents/test-agent/tasks/task-cancel-123/cancel" {
			t.Errorf("Expected cancel path, got %s", r.URL.Path)
		}

		task := Task{
			ID: "task-cancel-123",
			Status: TaskStatus{
				State:     TaskStateCanceled,
				UpdatedAt: time.Now(),
			},
		}

		_ = json.NewEncoder(w).Encode(task)
	}))
	defer ts.Close()

	client := NewClient(nil)
	task, err := client.CancelTask(context.Background(), ts.URL+"/agents/test-agent", "task-cancel-123", "user requested cancellation")

	if err != nil {
		t.Fatalf("CancelTask failed: %v", err)
	}

	if task.Status.State != TaskStateCanceled {
		t.Errorf("Expected canceled state, got %s", task.Status.State)
	}
}

func TestClient_ListAgents(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/agents" {
			t.Errorf("Expected path /agents, got %s", r.URL.Path)
		}

		directory := AgentDirectory{
			Agents: []AgentCard{
				{
					Name:        "agent-1",
					Description: "First agent",
					Version:     "1.0.0",
				},
				{
					Name:        "agent-2",
					Description: "Second agent",
					Version:     "2.0.0",
				},
			},
			Total: 2,
		}

		_ = json.NewEncoder(w).Encode(directory)
	}))
	defer ts.Close()

	client := NewClient(nil)
	agents, err := client.ListAgents(context.Background(), ts.URL+"/agents")

	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}

	if agents[0].Name != "agent-1" {
		t.Errorf("Expected first agent 'agent-1', got '%s'", agents[0].Name)
	}
}

// ============================================================================
// COVERAGE SUMMARY
// These client tests cover:
// - Client initialization with various configs
// - Agent discovery (DiscoverAgent)
// - Message sending (SendMessage)
// - Authentication (Bearer token, API key)
// - Streaming messages (SendMessageStreaming)
// - Task operations (GetTask, CancelTask)
// - Agent listing (ListAgents)
//
// Combined with server tests, estimated total coverage: 40-50%+ of pkg/a2a/
// ============================================================================
