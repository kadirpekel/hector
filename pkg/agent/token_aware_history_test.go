package agent

import (
	"testing"

	"github.com/kadirpekel/hector/pkg/llms"
)

func TestNewTokenAwareHistoryService(t *testing.T) {
	tests := []struct {
		name      string
		config    *TokenAwareHistoryConfig
		wantError bool
	}{
		{
			name:      "Nil config (uses defaults)",
			config:    nil,
			wantError: false,
		},
		{
			name: "Valid config",
			config: &TokenAwareHistoryConfig{
				MaxMessages: 20,
				MaxTokens:   3000,
				Model:       "gpt-4o",
			},
			wantError: false,
		},
		{
			name:      "Empty config (uses defaults)",
			config:    &TokenAwareHistoryConfig{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewTokenAwareHistoryService(tt.config)
			if (err != nil) != tt.wantError {
				t.Errorf("NewTokenAwareHistoryService() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && service == nil {
				t.Error("NewTokenAwareHistoryService() returned nil service")
			}
		})
	}
}

func TestTokenAwareHistoryService_AddAndGetHistory(t *testing.T) {
	config := &TokenAwareHistoryConfig{
		MaxMessages: 10,
		MaxTokens:   1000,
		Model:       "gpt-4o",
	}

	service, err := NewTokenAwareHistoryService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	sessionID := "test-session"

	// Add messages
	messages := []llms.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
		{Role: "assistant", Content: "I'm doing well, thank you!"},
	}

	for _, msg := range messages {
		service.AddToHistory(sessionID, msg)
	}

	// Get history
	history := service.GetRecentHistory(sessionID, 10)

	if len(history) != len(messages) {
		t.Errorf("GetRecentHistory() returned %d messages, want %d", len(history), len(messages))
	}

	// Verify order and content
	for i, msg := range messages {
		if history[i].Role != msg.Role || history[i].Content != msg.Content {
			t.Errorf("Message %d mismatch: got (%s, %s), want (%s, %s)",
				i, history[i].Role, history[i].Content, msg.Role, msg.Content)
		}
	}
}

func TestTokenAwareHistoryService_TokenLimit(t *testing.T) {
	config := &TokenAwareHistoryConfig{
		MaxMessages: 100, // High message count
		MaxTokens:   50,  // Low token limit (should be the constraint)
		Model:       "gpt-4o",
	}

	service, err := NewTokenAwareHistoryService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	sessionID := "test-session"

	// Add many messages
	for i := 0; i < 20; i++ {
		service.AddToHistory(sessionID, llms.Message{
			Role:    "user",
			Content: "This is a test message with some content",
		})
		service.AddToHistory(sessionID, llms.Message{
			Role:    "assistant",
			Content: "This is a response with some content",
		})
	}

	// Get history with token limit
	history := service.GetRecentHistory(sessionID, 100)

	// Should be limited by tokens, not message count
	// With 50 token limit, we should get fewer messages
	if len(history) >= 20 {
		t.Errorf("Expected token limit to constrain history, got %d messages", len(history))
	}

	// Get token service for stats check
	tokenService := service.(*TokenAwareHistoryService)
	_ = tokenService.GetTokenCount(sessionID)

	// The returned history should be much less than all messages
	if len(history) == 40 { // 20 * 2 messages
		t.Error("Token limit not applied, all messages returned")
	}
}

func TestTokenAwareHistoryService_GetRecentHistoryWithTokenLimit(t *testing.T) {
	config := &TokenAwareHistoryConfig{
		MaxMessages: 100,
		MaxTokens:   2000,
		Model:       "gpt-4o",
	}

	service, err := NewTokenAwareHistoryService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	tokenService := service.(*TokenAwareHistoryService)

	sessionID := "test-session"

	// Add messages
	messages := []llms.Message{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
		{Role: "assistant", Content: "Response 2"},
		{Role: "user", Content: "Message 3"},
		{Role: "assistant", Content: "Response 3"},
	}

	for _, msg := range messages {
		service.AddToHistory(sessionID, msg)
	}

	// Get history with custom token limit
	history := tokenService.GetRecentHistoryWithTokenLimit(sessionID, 30)

	// Should return fewer messages due to token limit
	if len(history) > len(messages) {
		t.Errorf("GetRecentHistoryWithTokenLimit() returned more messages than exist")
	}

	// Last message should be the most recent
	if len(history) > 0 {
		lastMsg := history[len(history)-1]
		expectedLast := messages[len(messages)-1]
		if lastMsg.Content != expectedLast.Content {
			t.Error("GetRecentHistoryWithTokenLimit() should preserve most recent messages")
		}
	}
}

func TestTokenAwareHistoryService_ClearHistory(t *testing.T) {
	config := &TokenAwareHistoryConfig{
		MaxMessages: 10,
		MaxTokens:   1000,
		Model:       "gpt-4o",
	}

	service, err := NewTokenAwareHistoryService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	sessionID := "test-session"

	// Add messages
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Hello"})
	service.AddToHistory(sessionID, llms.Message{Role: "assistant", Content: "Hi"})

	// Clear history
	service.ClearHistory(sessionID)

	// Get history should return empty
	history := service.GetRecentHistory(sessionID, 10)
	if len(history) != 0 {
		t.Errorf("ClearHistory() did not clear history, got %d messages", len(history))
	}
}

func TestTokenAwareHistoryService_MultipleSession(t *testing.T) {
	config := &TokenAwareHistoryConfig{
		MaxMessages: 10,
		MaxTokens:   1000,
		Model:       "gpt-4o",
	}

	service, err := NewTokenAwareHistoryService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	tokenService := service.(*TokenAwareHistoryService)

	// Add messages to different sessions
	service.AddToHistory("session1", llms.Message{Role: "user", Content: "Session 1 message"})
	service.AddToHistory("session2", llms.Message{Role: "user", Content: "Session 2 message"})

	// Get history for each session
	history1 := service.GetRecentHistory("session1", 10)
	history2 := service.GetRecentHistory("session2", 10)

	if len(history1) != 1 || len(history2) != 1 {
		t.Error("Multiple sessions not isolated properly")
	}

	if history1[0].Content == history2[0].Content {
		t.Error("Sessions returned same messages")
	}

	// Check session count
	count := tokenService.GetSessionCount()
	if count != 2 {
		t.Errorf("GetSessionCount() = %d, want 2", count)
	}
}

func TestTokenAwareHistoryService_GetTokenCount(t *testing.T) {
	config := &TokenAwareHistoryConfig{
		MaxMessages: 10,
		MaxTokens:   1000,
		Model:       "gpt-4o",
	}

	service, err := NewTokenAwareHistoryService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	tokenService := service.(*TokenAwareHistoryService)

	sessionID := "test-session"

	// Initially should be 0
	count := tokenService.GetTokenCount(sessionID)
	if count != 0 {
		t.Errorf("Initial token count = %d, want 0", count)
	}

	// Add messages
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Hello world"})
	service.AddToHistory(sessionID, llms.Message{Role: "assistant", Content: "Hi there"})

	// Should have tokens now
	count = tokenService.GetTokenCount(sessionID)
	if count <= 0 {
		t.Error("Token count should be > 0 after adding messages")
	}
}

func TestTokenAwareHistoryService_GetSessionStats(t *testing.T) {
	config := &TokenAwareHistoryConfig{
		MaxMessages: 10,
		MaxTokens:   1000,
		Model:       "gpt-4o",
	}

	service, err := NewTokenAwareHistoryService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	tokenService := service.(*TokenAwareHistoryService)

	sessionID := "test-session"

	// Get stats for non-existent session
	stats := tokenService.GetSessionStats(sessionID)
	if stats.MessageCount != 0 || stats.TokenCount != 0 {
		t.Error("Stats for non-existent session should be zero")
	}

	// Add messages
	service.AddToHistory(sessionID, llms.Message{Role: "user", Content: "Test"})
	service.AddToHistory(sessionID, llms.Message{Role: "assistant", Content: "Response"})

	// Get stats
	stats = tokenService.GetSessionStats(sessionID)
	if stats.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2", stats.MessageCount)
	}
	if stats.TokenCount <= 0 {
		t.Error("TokenCount should be > 0")
	}
	if stats.MaxTokens != 1000 {
		t.Errorf("MaxTokens = %d, want 1000", stats.MaxTokens)
	}
	if stats.Utilization <= 0 || stats.Utilization > 100 {
		t.Errorf("Utilization = %f, want between 0 and 100", stats.Utilization)
	}
}

func TestTokenAwareHistoryService_DefaultSessionID(t *testing.T) {
	config := &TokenAwareHistoryConfig{
		MaxMessages: 10,
		MaxTokens:   1000,
		Model:       "gpt-4o",
	}

	service, err := NewTokenAwareHistoryService(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Add message with empty session ID
	service.AddToHistory("", llms.Message{Role: "user", Content: "Test"})

	// Should be stored in "default" session
	history1 := service.GetRecentHistory("", 10)
	history2 := service.GetRecentHistory("default", 10)

	if len(history1) != 1 || len(history2) != 1 {
		t.Error("Empty session ID should map to 'default'")
	}

	if history1[0].Content != history2[0].Content {
		t.Error("Empty and 'default' session IDs should be equivalent")
	}
}

func BenchmarkTokenAwareHistoryService_AddToHistory(b *testing.B) {
	config := &TokenAwareHistoryConfig{
		MaxMessages: 1000,
		MaxTokens:   10000,
		Model:       "gpt-4o",
	}

	service, err := NewTokenAwareHistoryService(config)
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}

	msg := llms.Message{
		Role:    "user",
		Content: "This is a benchmark test message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.AddToHistory("bench-session", msg)
	}
}

func BenchmarkTokenAwareHistoryService_GetRecentHistory(b *testing.B) {
	config := &TokenAwareHistoryConfig{
		MaxMessages: 1000,
		MaxTokens:   10000,
		Model:       "gpt-4o",
	}

	service, err := NewTokenAwareHistoryService(config)
	if err != nil {
		b.Fatalf("Failed to create service: %v", err)
	}

	// Add some messages
	for i := 0; i < 100; i++ {
		service.AddToHistory("bench-session", llms.Message{
			Role:    "user",
			Content: "Test message",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetRecentHistory("bench-session", 50)
	}
}
