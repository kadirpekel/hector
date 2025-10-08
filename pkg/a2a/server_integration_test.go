//go:build integration

package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ============================================================================
// A2A SERVER TESTS - HTTP+JSON Transport
// Uses MockAgent from protocol_test.go
// ============================================================================

func createTestServer() *Server {
	return NewServer(&ServerConfig{
		Host:    "localhost",
		Port:    8080,
		BaseURL: "http://localhost:8080",
	})
}

func TestNewServer(t *testing.T) {
	server := createTestServer()

	if server.host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", server.host)
	}

	if server.port != 8080 {
		t.Errorf("Expected port 8080, got %d", server.port)
	}

	if server.baseURL != "http://localhost:8080" {
		t.Errorf("Expected baseURL 'http://localhost:8080', got '%s'", server.baseURL)
	}

	if server.agents == nil {
		t.Error("agents map is nil")
	}

	if server.agentCards == nil {
		t.Error("agentCards map is nil")
	}
}

func TestNewServer_DefaultBaseURL(t *testing.T) {
	server := NewServer(&ServerConfig{
		Host: "localhost",
		Port: 9000,
		// BaseURL not provided
	})

	expected := "http://localhost:9000"
	if server.baseURL != expected {
		t.Errorf("Expected default baseURL '%s', got '%s'", expected, server.baseURL)
	}
}

func TestServer_RegisterAgent(t *testing.T) {
	server := createTestServer()

	agent := &MockAgent{
		card: &AgentCard{
			Name:        "test-agent",
			Description: "Test agent",
			Version:     "1.0.0",
			Capabilities: AgentCapabilities{
				Streaming: true,
				MultiTurn: true,
			},
		},
	}

	err := server.RegisterAgent("test-agent", agent, "public")
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	// Verify agent was registered
	server.mu.RLock()
	registeredAgent, exists := server.agents["test-agent"]
	card, cardExists := server.agentCards["test-agent"]
	visibility := server.agentVisibility["test-agent"]
	server.mu.RUnlock()

	if !exists {
		t.Error("Agent not found in registry")
	}

	if registeredAgent != agent {
		t.Error("Registered agent doesn't match")
	}

	if !cardExists {
		t.Error("Agent card not found")
	}

	if card.URL == "" {
		t.Error("Agent card URL not set")
	}

	if card.PreferredTransport != "http+json" {
		t.Errorf("Expected PreferredTransport 'http+json', got '%s'", card.PreferredTransport)
	}

	if visibility != "public" {
		t.Errorf("Expected visibility 'public', got '%s'", visibility)
	}
}

func TestServer_RegisterAgent_InvalidVisibility(t *testing.T) {
	server := createTestServer()

	agent := &MockAgent{
		card: &AgentCard{Name: "test-agent"},
	}

	err := server.RegisterAgent("test-agent", agent, "invalid-visibility")
	if err == nil {
		t.Error("Expected error for invalid visibility, got nil")
	}
}

func TestServer_RegisterAgent_NilCard(t *testing.T) {
	server := createTestServer()

	agent := &MockAgent{

		card: nil, // Nil card
	}

	err := server.RegisterAgent("bad-agent", agent, "public")
	if err == nil {
		t.Error("Expected error for nil agent card, got nil")
	}
}

func TestServer_RegisterAgent_DefaultVisibility(t *testing.T) {
	server := createTestServer()

	agent := &MockAgent{
		card: &AgentCard{Name: "test-agent"},
	}

	err := server.RegisterAgent("test-agent", agent, "") // Empty visibility
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	server.mu.RLock()
	visibility := server.agentVisibility["test-agent"]
	server.mu.RUnlock()

	if visibility != "public" {
		t.Errorf("Expected default visibility 'public', got '%s'", visibility)
	}
}

func TestServer_HandleListAgents(t *testing.T) {
	server := createTestServer()

	// Register agents with different visibilities
	publicAgent := &MockAgent{

		card: &AgentCard{Name: "public-agent", Description: "Public agent"},
	}
	internalAgent := &MockAgent{

		card: &AgentCard{Name: "internal-agent", Description: "Internal agent"},
	}
	privateAgent := &MockAgent{

		card: &AgentCard{Name: "private-agent", Description: "Private agent"},
	}

	_ = server.RegisterAgent("public-agent", publicAgent, "public")
	_ = server.RegisterAgent("internal-agent", internalAgent, "internal")
	_ = server.RegisterAgent("private-agent", privateAgent, "private")

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/agents", nil)
	w := httptest.NewRecorder()

	server.handleListAgents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var directory AgentDirectory
	if err := json.NewDecoder(w.Body).Decode(&directory); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should only include public agents
	if directory.Total != 1 {
		t.Errorf("Expected 1 public agent, got %d", directory.Total)
	}

	if len(directory.Agents) != 1 {
		t.Fatalf("Expected 1 agent in list, got %d", len(directory.Agents))
	}

	if directory.Agents[0].Name != "public-agent" {
		t.Errorf("Expected 'public-agent', got '%s'", directory.Agents[0].Name)
	}
}

func TestServer_HandleListAgents_MethodNotAllowed(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest(http.MethodPost, "/agents", nil)
	w := httptest.NewRecorder()

	server.handleListAgents(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestServer_HandleGetAgent(t *testing.T) {
	server := createTestServer()

	agent := &MockAgent{
		card: &AgentCard{
			Name:        "test-agent",
			Description: "Test agent",
			Version:     "1.0.0",
		},
	}
	_ = server.RegisterAgent("test-agent", agent, "public")

	req := httptest.NewRequest(http.MethodGet, "/agents/test-agent", nil)
	w := httptest.NewRecorder()

	server.handleAgentRoutes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var card AgentCard
	if err := json.NewDecoder(w.Body).Decode(&card); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if card.Name != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", card.Name)
	}
}

func TestServer_HandleGetAgent_NotFound(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest(http.MethodGet, "/agents/nonexistent", nil)
	w := httptest.NewRecorder()

	server.handleAgentRoutes(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestServer_HandleSendMessage(t *testing.T) {
	server := createTestServer()

	agent := &MockAgent{
		card: &AgentCard{Name: "test-agent"},
	}
	_ = server.RegisterAgent("test-agent", agent, "public")

	// Create message request
	params := MessageSendParams{
		Message: Message{
			Role: MessageRoleUser,
			Parts: []Part{
				{Type: PartTypeText, Text: "Hello"},
			},
		},
	}

	body, _ := json.Marshal(params)
	req := httptest.NewRequest(http.MethodPost, "/agents/test-agent/message/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAgentRoutes(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202 (Accepted), got %d: %s", w.Code, w.Body.String())
	}

	// Parse response (should be a Task)
	var task Task
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if task.ID == "" {
		t.Error("TaskID is empty")
	}

	if len(task.Messages) == 0 {
		t.Error("No messages in response")
	}
}

func TestServer_HandleSendMessage_InvalidJSON(t *testing.T) {
	server := createTestServer()

	agent := &MockAgent{
		card: &AgentCard{Name: "test-agent"},
	}
	_ = server.RegisterAgent("test-agent", agent, "public")

	req := httptest.NewRequest(http.MethodPost, "/agents/test-agent/message/send", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleAgentRoutes(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestServer_HandleGetSession(t *testing.T) {
	server := createTestServer()

	// Create a session manually
	sessionID := "test-session-123"
	session := &Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	server.mu.Lock()
	server.sessions[sessionID] = session
	server.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/sessions/"+sessionID, nil)
	w := httptest.NewRecorder()

	server.handleSessionRoutes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var retrievedSession Session
	if err := json.NewDecoder(w.Body).Decode(&retrievedSession); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if retrievedSession.ID != sessionID {
		t.Errorf("Expected session ID '%s', got '%s'", sessionID, retrievedSession.ID)
	}
}

func TestServer_HandleDeleteSession(t *testing.T) {
	server := createTestServer()

	// Create a session manually
	sessionID := "test-session-456"
	session := &Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
	}

	server.mu.Lock()
	server.sessions[sessionID] = session
	server.mu.Unlock()

	req := httptest.NewRequest(http.MethodDelete, "/sessions/"+sessionID, nil)
	w := httptest.NewRecorder()

	server.handleSessionRoutes(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify session was deleted
	server.mu.RLock()
	_, exists := server.sessions[sessionID]
	server.mu.RUnlock()

	if exists {
		t.Error("Session should have been deleted")
	}
}

func TestServer_SetAuthValidator(t *testing.T) {
	server := createTestServer()

	// Mock auth validator
	mockValidator := &MockAuthValidator{}

	server.SetAuthValidator(mockValidator)

	if server.authValidator == nil {
		t.Error("AuthValidator not set")
	}
}

// MockAuthValidator for testing
type MockAuthValidator struct{}

func (m *MockAuthValidator) HTTPMiddleware(next http.Handler) http.Handler {
	return next
}

func (m *MockAuthValidator) ValidateToken(ctx context.Context, tokenString string) (interface{}, error) {
	return map[string]string{"user": "test"}, nil
}

// ============================================================================
// COVERAGE SUMMARY
// These tests cover:
// - Server initialization and configuration
// - Agent registration (public, internal, private)
// - Agent discovery (GET /agents)
// - Agent card retrieval (GET /agents/{id})
// - Message sending (POST /agents/{id}/message)
// - Session management (GET/DELETE /sessions/{id})
// - Error handling (invalid JSON, not found, method not allowed)
// - Authentication setup
//
// Estimated coverage: ~40-50% of server.go
// ============================================================================
