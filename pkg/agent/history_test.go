package agent

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/llms"
)

// TestNewHistoryService_CountBased tests count-based mode
func TestNewHistoryService_CountBased(t *testing.T) {
	service, err := NewHistoryService(&HistoryConfig{
		MaxMessages: 5,
	})
	if err != nil {
		t.Fatalf("Failed to create count-based history service: %v", err)
	}

	histService := service.(*HistoryService)
	if histService.maxMessages != 5 {
		t.Errorf("Expected maxMessages=5, got %d", histService.maxMessages)
	}
	if histService.tokenBudget != 0 {
		t.Errorf("Expected tokenBudget=0 for count-based mode, got %d", histService.tokenBudget)
	}
	if histService.tokenCounter != nil {
		t.Error("Expected tokenCounter=nil for count-based mode")
	}
}

// TestNewHistoryService_TokenBased tests token-based mode
func TestNewHistoryService_TokenBased(t *testing.T) {
	service, err := NewHistoryService(&HistoryConfig{
		MaxMessages: 10,
		TokenBudget: 2000,
		Model:       "gpt-4o",
	})
	if err != nil {
		t.Fatalf("Failed to create token-based history service: %v", err)
	}

	histService := service.(*HistoryService)
	if histService.tokenBudget != 2000 {
		t.Errorf("Expected tokenBudget=2000, got %d", histService.tokenBudget)
	}
	if histService.tokenCounter == nil {
		t.Error("Expected tokenCounter to be initialized for token-based mode")
	}
}

// TestNewHistoryService_TokenBasedWithoutModel tests validation
func TestNewHistoryService_TokenBasedWithoutModel(t *testing.T) {
	_, err := NewHistoryService(&HistoryConfig{
		TokenBudget: 2000,
		// Model missing
	})
	if err == nil {
		t.Error("Expected error when TokenBudget is set but Model is missing")
	}
}

// TestHistoryService_AddAndGet_CountBased tests basic add/get with count-based mode
func TestHistoryService_AddAndGet_CountBased(t *testing.T) {
	service, err := NewHistoryService(&HistoryConfig{
		MaxMessages: 3,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Add messages
	service.AddToHistory("session1", llms.Message{Role: "user", Content: "Message 1"})
	service.AddToHistory("session1", llms.Message{Role: "assistant", Content: "Response 1"})
	service.AddToHistory("session1", llms.Message{Role: "user", Content: "Message 2"})
	service.AddToHistory("session1", llms.Message{Role: "assistant", Content: "Response 2"})

	// Get recent history (should return last 3 messages)
	messages := service.GetRecentHistory("session1", 3)
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Check that we got the most recent ones
	if messages[0].Content != "Response 1" {
		t.Errorf("Expected first message to be 'Response 1', got '%s'", messages[0].Content)
	}
}

// TestHistoryService_SessionIsolation tests that sessions are isolated
func TestHistoryService_SessionIsolation(t *testing.T) {
	service, err := NewHistoryService(&HistoryConfig{
		MaxMessages: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Add to session1
	service.AddToHistory("session1", llms.Message{Role: "user", Content: "Session 1 message"})

	// Add to session2
	service.AddToHistory("session2", llms.Message{Role: "user", Content: "Session 2 message"})

	// Get session1 history
	messages1 := service.GetRecentHistory("session1", 10)
	if len(messages1) != 1 {
		t.Errorf("Expected 1 message in session1, got %d", len(messages1))
	}
	if messages1[0].Content != "Session 1 message" {
		t.Errorf("Unexpected message in session1: %s", messages1[0].Content)
	}

	// Get session2 history
	messages2 := service.GetRecentHistory("session2", 10)
	if len(messages2) != 1 {
		t.Errorf("Expected 1 message in session2, got %d", len(messages2))
	}
	if messages2[0].Content != "Session 2 message" {
		t.Errorf("Unexpected message in session2: %s", messages2[0].Content)
	}
}

// TestHistoryService_ClearHistory tests clearing a specific session
func TestHistoryService_ClearHistory(t *testing.T) {
	service, err := NewHistoryService(&HistoryConfig{
		MaxMessages: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Add messages
	service.AddToHistory("session1", llms.Message{Role: "user", Content: "Message 1"})
	service.AddToHistory("session1", llms.Message{Role: "assistant", Content: "Response 1"})

	// Verify messages exist
	messages := service.GetRecentHistory("session1", 10)
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages before clear, got %d", len(messages))
	}

	// Clear history
	service.ClearHistory("session1")

	// Verify history is cleared
	messages = service.GetRecentHistory("session1", 10)
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(messages))
	}
}

// TestHistoryService_DefaultSession tests that empty sessionID uses "default"
func TestHistoryService_DefaultSession(t *testing.T) {
	service, err := NewHistoryService(&HistoryConfig{
		MaxMessages: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Add with empty sessionID
	service.AddToHistory("", llms.Message{Role: "user", Content: "Default session message"})

	// Get with "default" sessionID explicitly
	messages := service.GetRecentHistory("default", 10)
	if len(messages) != 1 {
		t.Errorf("Expected 1 message in default session, got %d", len(messages))
	}

	// Get with empty sessionID
	messages = service.GetRecentHistory("", 10)
	if len(messages) != 1 {
		t.Errorf("Expected 1 message when using empty sessionID, got %d", len(messages))
	}
}

// TestHistoryService_GetSessionCount tests session counting
func TestHistoryService_GetSessionCount(t *testing.T) {
	service, err := NewHistoryService(&HistoryConfig{
		MaxMessages: 10,
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	histService := service.(*HistoryService)

	// Initially no sessions
	if count := histService.GetSessionCount(); count != 0 {
		t.Errorf("Expected 0 sessions initially, got %d", count)
	}

	// Add to session1
	service.AddToHistory("session1", llms.Message{Role: "user", Content: "Test"})
	if count := histService.GetSessionCount(); count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
	}

	// Add to session2
	service.AddToHistory("session2", llms.Message{Role: "user", Content: "Test"})
	if count := histService.GetSessionCount(); count != 2 {
		t.Errorf("Expected 2 sessions, got %d", count)
	}

	// Clear all sessions
	histService.ClearAllSessions()
	if count := histService.GetSessionCount(); count != 0 {
		t.Errorf("Expected 0 sessions after clear all, got %d", count)
	}
}

// TestHistoryService_TokenBased_Selection tests token-based message selection
func TestHistoryService_TokenBased_Selection(t *testing.T) {
	service, err := NewHistoryService(&HistoryConfig{
		MaxMessages: 100, // High fallback
		TokenBudget: 50,  // Very small budget to force trimming
		Model:       "gpt-4o",
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Add many messages
	for i := 0; i < 20; i++ {
		service.AddToHistory("session1", llms.Message{
			Role:    "user",
			Content: "This is a test message that takes up some tokens",
		})
	}

	// Get recent history with token budget
	messages := service.GetRecentHistory("session1", 0)

	// Should return fewer than 20 messages due to token budget
	if len(messages) >= 20 {
		t.Errorf("Expected token budget to limit messages, got %d messages", len(messages))
	}
	if len(messages) == 0 {
		t.Error("Expected at least some messages within token budget")
	}
}
