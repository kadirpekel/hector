package context

import (
	"fmt"
	"testing"
)

func TestNewConversationHistory(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		wantError bool
	}{
		{
			name:      "valid_session_id",
			sessionID: "test-session-123",
			wantError: false,
		},
		{
			name:      "empty_session_id",
			sessionID: "",
			wantError: true,
		},
		{
			name:      "session_id_with_spaces",
			sessionID: "test session",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history, err := NewConversationHistory(tt.sessionID)

			if tt.wantError {
				if err == nil {
					t.Error("NewConversationHistory() expected error, got nil")
				}
				if history != nil {
					t.Error("NewConversationHistory() expected nil history on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewConversationHistory() error = %v, want nil", err)
				}
				if history == nil {
					t.Error("NewConversationHistory() returned nil history")
				}
				if history != nil {
					if history.SessionID != tt.sessionID {
						t.Errorf("NewConversationHistory() SessionID = %v, want %v", history.SessionID, tt.sessionID)
					}
					if history.MaxMessages != DefaultMaxMessages {
						t.Errorf("NewConversationHistory() MaxMessages = %v, want %v", history.MaxMessages, DefaultMaxMessages)
					}
					if len(history.Messages) != 0 {
						t.Errorf("NewConversationHistory() Messages length = %v, want 0", len(history.Messages))
					}
					if len(history.Context) != 0 {
						t.Errorf("NewConversationHistory() Context length = %v, want 0", len(history.Context))
					}
				}
			}
		})
	}
}

func TestNewConversationHistoryWithMax(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		maxMessages int
		wantError   bool
	}{
		{
			name:        "valid_session_and_max",
			sessionID:   "test-session-123",
			maxMessages: 500,
			wantError:   false,
		},
		{
			name:        "empty_session_id",
			sessionID:   "",
			maxMessages: 500,
			wantError:   true,
		},
		{
			name:        "max_messages_too_low",
			sessionID:   "test-session-123",
			maxMessages: 0,
			wantError:   true,
		},
		{
			name:        "max_messages_too_high",
			sessionID:   "test-session-123",
			maxMessages: 20000,
			wantError:   true,
		},
		{
			name:        "max_messages_minimum",
			sessionID:   "test-session-123",
			maxMessages: MinMaxMessages,
			wantError:   false,
		},
		{
			name:        "max_messages_maximum",
			sessionID:   "test-session-123",
			maxMessages: MaxMaxMessages,
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history, err := NewConversationHistoryWithMax(tt.sessionID, tt.maxMessages)

			if tt.wantError {
				if err == nil {
					t.Error("NewConversationHistoryWithMax() expected error, got nil")
				}
				if history != nil {
					t.Error("NewConversationHistoryWithMax() expected nil history on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewConversationHistoryWithMax() error = %v, want nil", err)
				}
				if history == nil {
					t.Error("NewConversationHistoryWithMax() returned nil history")
				}
				if history != nil {
					if history.SessionID != tt.sessionID {
						t.Errorf("NewConversationHistoryWithMax() SessionID = %v, want %v", history.SessionID, tt.sessionID)
					}
					if history.MaxMessages != tt.maxMessages {
						t.Errorf("NewConversationHistoryWithMax() MaxMessages = %v, want %v", history.MaxMessages, tt.maxMessages)
					}
				}
			}
		})
	}
}

func TestConversationHistory_AddMessage(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	tests := []struct {
		name      string
		role      string
		content   string
		metadata  map[string]interface{}
		wantError bool
	}{
		{
			name:      "valid_user_message",
			role:      RoleUser,
			content:   "Hello, how are you?",
			metadata:  map[string]interface{}{"source": "web"},
			wantError: false,
		},
		{
			name:      "valid_assistant_message",
			role:      RoleAssistant,
			content:   "I'm doing well, thank you!",
			metadata:  map[string]interface{}{"model": "gpt-4"},
			wantError: false,
		},
		{
			name:      "valid_system_message",
			role:      RoleSystem,
			content:   "You are a helpful assistant.",
			metadata:  nil,
			wantError: false,
		},
		{
			name:      "empty_role",
			role:      "",
			content:   "Hello",
			metadata:  nil,
			wantError: true,
		},
		{
			name:      "empty_content",
			role:      RoleUser,
			content:   "",
			metadata:  nil,
			wantError: true,
		},
		{
			name:      "content_too_short",
			role:      RoleUser,
			content:   "",
			metadata:  nil,
			wantError: true,
		},
		{
			name:      "content_too_long",
			role:      RoleUser,
			content:   string(make([]byte, MaxMessageLength+1)),
			metadata:  nil,
			wantError: true,
		},
		{
			name:      "invalid_role",
			role:      "invalid",
			content:   "Hello",
			metadata:  nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, err := history.AddMessage(tt.role, tt.content, tt.metadata)

			if tt.wantError {
				if err == nil {
					t.Error("AddMessage() expected error, got nil")
				}
				if message != nil {
					t.Error("AddMessage() expected nil message on error")
				}
			} else {
				if err != nil {
					t.Errorf("AddMessage() error = %v, want nil", err)
				}
				if message == nil {
					t.Error("AddMessage() returned nil message")
				}
				if message != nil {
					if message.Role != tt.role {
						t.Errorf("AddMessage() Role = %v, want %v", message.Role, tt.role)
					}
					if message.Content != tt.content {
						t.Errorf("AddMessage() Content = %v, want %v", message.Content, tt.content)
					}
					if message.ID == "" {
						t.Error("AddMessage() ID should not be empty")
					}
					if message.Timestamp.IsZero() {
						t.Error("AddMessage() Timestamp should not be zero")
					}
					if tt.metadata != nil {
						if len(message.Metadata) != len(tt.metadata) {
							t.Errorf("AddMessage() Metadata length = %v, want %v", len(message.Metadata), len(tt.metadata))
						}
					}
				}
			}
		})
	}
}

func TestConversationHistory_AddUserMessage(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	content := "Hello, I need help with my project."
	metadata := map[string]interface{}{"source": "web"}

	message, err := history.AddUserMessage(content, metadata)
	if err != nil {
		t.Errorf("AddUserMessage() error = %v, want nil", err)
	}
	if message == nil {
		t.Fatal("AddUserMessage() returned nil message")
	}

	if message.Role != RoleUser {
		t.Errorf("AddUserMessage() Role = %v, want %v", message.Role, RoleUser)
	}
	if message.Content != content {
		t.Errorf("AddUserMessage() Content = %v, want %v", message.Content, content)
	}
}

func TestConversationHistory_AddAssistantMessage(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	content := "I'd be happy to help you with your project!"
	metadata := map[string]interface{}{"model": "gpt-4"}

	message, err := history.AddAssistantMessage(content, metadata)
	if err != nil {
		t.Errorf("AddAssistantMessage() error = %v, want nil", err)
	}
	if message == nil {
		t.Fatal("AddAssistantMessage() returned nil message")
	}

	if message.Role != RoleAssistant {
		t.Errorf("AddAssistantMessage() Role = %v, want %v", message.Role, RoleAssistant)
	}
	if message.Content != content {
		t.Errorf("AddAssistantMessage() Content = %v, want %v", message.Content, content)
	}
}

func TestConversationHistory_AddSystemMessage(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	content := "You are a helpful assistant specialized in software development."

	message, err := history.AddSystemMessage(content, nil)
	if err != nil {
		t.Errorf("AddSystemMessage() error = %v, want nil", err)
	}
	if message == nil {
		t.Fatal("AddSystemMessage() returned nil message")
	}

	if message.Role != RoleSystem {
		t.Errorf("AddSystemMessage() Role = %v, want %v", message.Role, RoleSystem)
	}
	if message.Content != content {
		t.Errorf("AddSystemMessage() Content = %v, want %v", message.Content, content)
	}
}

func TestConversationHistory_GetRecentMessages(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Add some messages
	for i := 0; i < 5; i++ {
		_, err := history.AddUserMessage("Message "+string(rune('0'+i)), nil)
		if err != nil {
			t.Fatalf("AddUserMessage() error = %v", err)
		}
	}

	tests := []struct {
		name     string
		n        int
		expected int
	}{
		{
			name:     "get_last_3_messages",
			n:        3,
			expected: 3,
		},
		{
			name:     "get_more_than_available",
			n:        10,
			expected: 5,
		},
		{
			name:     "get_zero_messages",
			n:        0,
			expected: 0,
		},
		{
			name:     "get_negative_messages",
			n:        -1,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := history.GetRecentMessages(tt.n)

			if len(messages) != tt.expected {
				t.Errorf("GetRecentMessages() length = %v, want %v", len(messages), tt.expected)
			}

			// Check that messages are returned in correct order (most recent last)
			if len(messages) > 1 {
				for i := 1; i < len(messages); i++ {
					if messages[i-1].Timestamp.After(messages[i].Timestamp) {
						t.Error("GetRecentMessages() messages not in chronological order")
					}
				}
			}
		})
	}
}

func TestConversationHistory_GetMessagesByRole(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Add messages with different roles
	_, err = history.AddUserMessage("User message 1", nil)
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}
	_, err = history.AddAssistantMessage("Assistant message 1", nil)
	if err != nil {
		t.Fatalf("AddAssistantMessage() error = %v", err)
	}
	_, err = history.AddUserMessage("User message 2", nil)
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}
	_, err = history.AddSystemMessage("System message 1", nil)
	if err != nil {
		t.Fatalf("AddSystemMessage() error = %v", err)
	}

	tests := []struct {
		name     string
		role     string
		limit    int
		expected int
	}{
		{
			name:     "get_user_messages",
			role:     RoleUser,
			limit:    0, // No limit
			expected: 2,
		},
		{
			name:     "get_assistant_messages",
			role:     RoleAssistant,
			limit:    0,
			expected: 1,
		},
		{
			name:     "get_system_messages",
			role:     RoleSystem,
			limit:    0,
			expected: 1,
		},
		{
			name:     "get_nonexistent_role",
			role:     "nonexistent",
			limit:    0,
			expected: 0,
		},
		{
			name:     "get_user_messages_with_limit",
			role:     RoleUser,
			limit:    1,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := history.GetMessagesByRole(tt.role, tt.limit)

			if len(messages) != tt.expected {
				t.Errorf("GetMessagesByRole() length = %v, want %v", len(messages), tt.expected)
			}

			// Check that all returned messages have the correct role
			for _, msg := range messages {
				if msg.Role != tt.role {
					t.Errorf("GetMessagesByRole() message role = %v, want %v", msg.Role, tt.role)
				}
			}
		})
	}
}

func TestConversationHistory_GetMessageByID(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Add a message
	message, err := history.AddUserMessage("Test message", nil)
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}

	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{
			name:     "get_existing_message",
			id:       message.ID,
			expected: true,
		},
		{
			name:     "get_nonexistent_message",
			id:       "nonexistent-id",
			expected: false,
		},
		{
			name:     "get_empty_id",
			id:       "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundMessage, exists := history.GetMessageByID(tt.id)

			if exists != tt.expected {
				t.Errorf("GetMessageByID() exists = %v, want %v", exists, tt.expected)
			}

			if tt.expected {
				if foundMessage == nil {
					t.Error("GetMessageByID() expected message, got nil")
				} else if foundMessage.ID != tt.id {
					t.Errorf("GetMessageByID() ID = %v, want %v", foundMessage.ID, tt.id)
				}
			} else {
				if foundMessage != nil {
					t.Error("GetMessageByID() expected nil message, got message")
				}
			}
		})
	}
}

func TestConversationHistory_GetRecentConversationMessages(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Add some messages
	_, err = history.AddUserMessage("User message", map[string]interface{}{"source": "web"})
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}
	_, err = history.AddAssistantMessage("Assistant message", map[string]interface{}{"model": "gpt-4"})
	if err != nil {
		t.Fatalf("AddAssistantMessage() error = %v", err)
	}

	conversationMessages := history.GetRecentConversationMessages(2)

	if len(conversationMessages) != 2 {
		t.Errorf("GetRecentConversationMessages() length = %v, want 2", len(conversationMessages))
	}

	// Check that ConversationMessage has the same fields as Message
	for i, convMsg := range conversationMessages {
		if convMsg.Role == "" {
			t.Errorf("GetRecentConversationMessages() message %d has empty role", i)
		}
		if convMsg.Content == "" {
			t.Errorf("GetRecentConversationMessages() message %d has empty content", i)
		}
		if convMsg.Timestamp.IsZero() {
			t.Errorf("GetRecentConversationMessages() message %d has zero timestamp", i)
		}
	}
}

func TestConversationHistory_SetContext(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	tests := []struct {
		name      string
		key       string
		value     interface{}
		wantError bool
	}{
		{
			name:      "set_string_context",
			key:       "user_name",
			value:     "John Doe",
			wantError: false,
		},
		{
			name:      "set_number_context",
			key:       "user_id",
			value:     12345,
			wantError: false,
		},
		{
			name:      "set_map_context",
			key:       "preferences",
			value:     map[string]interface{}{"theme": "dark", "language": "en"},
			wantError: false,
		},
		{
			name:      "empty_key",
			key:       "",
			value:     "value",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := history.SetContext(tt.key, tt.value)

			if tt.wantError {
				if err == nil {
					t.Error("SetContext() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SetContext() error = %v, want nil", err)
				}

				// Verify the context was set
				retrievedValue, exists := history.GetContext(tt.key)
				if !exists {
					t.Errorf("SetContext() context key %v not found", tt.key)
				}
				// For maps, we can't use direct comparison, so we check if the value exists
				if retrievedValue == nil && tt.value != nil {
					t.Errorf("SetContext() retrieved value is nil, want non-nil")
				}
				if retrievedValue != nil && tt.value == nil {
					t.Errorf("SetContext() retrieved value is non-nil, want nil")
				}
				// For non-map values, we can compare directly
				if tt.key != "preferences" && retrievedValue != tt.value {
					t.Errorf("SetContext() retrieved value = %v, want %v", retrievedValue, tt.value)
				}
			}
		})
	}
}

func TestConversationHistory_GetContext(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Set some context
	err = history.SetContext("user_name", "John Doe")
	if err != nil {
		t.Fatalf("SetContext() error = %v", err)
	}
	err = history.SetContext("user_id", 12345)
	if err != nil {
		t.Fatalf("SetContext() error = %v", err)
	}

	tests := []struct {
		name     string
		key      string
		expected interface{}
		exists   bool
	}{
		{
			name:     "get_existing_string_context",
			key:      "user_name",
			expected: "John Doe",
			exists:   true,
		},
		{
			name:     "get_existing_number_context",
			key:      "user_id",
			expected: 12345,
			exists:   true,
		},
		{
			name:     "get_nonexistent_context",
			key:      "nonexistent",
			expected: nil,
			exists:   false,
		},
		{
			name:     "get_empty_key",
			key:      "",
			expected: nil,
			exists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, exists := history.GetContext(tt.key)

			if exists != tt.exists {
				t.Errorf("GetContext() exists = %v, want %v", exists, tt.exists)
			}

			if tt.exists {
				if value != tt.expected {
					t.Errorf("GetContext() value = %v, want %v", value, tt.expected)
				}
			}
		})
	}
}

func TestConversationHistory_RemoveContext(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Set some context
	err = history.SetContext("key1", "value1")
	if err != nil {
		t.Fatalf("SetContext() error = %v", err)
	}
	err = history.SetContext("key2", "value2")
	if err != nil {
		t.Fatalf("SetContext() error = %v", err)
	}

	// Remove one context key
	history.RemoveContext("key1")

	// Check that key1 is removed but key2 remains
	_, exists := history.GetContext("key1")
	if exists {
		t.Error("RemoveContext() key1 should not exist after removal")
	}

	value, exists := history.GetContext("key2")
	if !exists {
		t.Error("RemoveContext() key2 should still exist")
	}
	if value != "value2" {
		t.Errorf("RemoveContext() key2 value = %v, want value2", value)
	}
}

func TestConversationHistory_ClearContext(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Set some context
	err = history.SetContext("key1", "value1")
	if err != nil {
		t.Fatalf("SetContext() error = %v", err)
	}
	err = history.SetContext("key2", "value2")
	if err != nil {
		t.Fatalf("SetContext() error = %v", err)
	}

	// Clear all context
	history.ClearContext()

	// Check that all context is cleared
	_, exists := history.GetContext("key1")
	if exists {
		t.Error("ClearContext() key1 should not exist after clearing")
	}

	_, exists = history.GetContext("key2")
	if exists {
		t.Error("ClearContext() key2 should not exist after clearing")
	}
}

func TestConversationHistory_GetContextForLLM(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Add some messages
	_, err = history.AddUserMessage("Hello", nil)
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}
	_, err = history.AddAssistantMessage("Hi there!", nil)
	if err != nil {
		t.Fatalf("AddAssistantMessage() error = %v", err)
	}

	context := history.GetContextForLLM(2)

	if context == "" {
		t.Error("GetContextForLLM() returned empty context")
	}

	// Check that context contains the expected format
	expectedContent := "Conversation History:\nuser: Hello\nassistant: Hi there!\n"
	if context != expectedContent {
		t.Errorf("GetContextForLLM() = %v, want %v", context, expectedContent)
	}
}

func TestConversationHistory_GetContextForLLMWithMetadata(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Add messages with metadata
	_, err = history.AddUserMessage("Hello", map[string]interface{}{"source": "web"})
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}
	_, err = history.AddAssistantMessage("Hi there!", map[string]interface{}{"model": "gpt-4"})
	if err != nil {
		t.Fatalf("AddAssistantMessage() error = %v", err)
	}

	context := history.GetContextForLLMWithMetadata(2)

	if context == "" {
		t.Error("GetContextForLLMWithMetadata() returned empty context")
	}

	// Check that context contains metadata
	if !contains(context, "source=web") {
		t.Error("GetContextForLLMWithMetadata() should contain source=web")
	}
	if !contains(context, "model=gpt-4") {
		t.Error("GetContextForLLMWithMetadata() should contain model=gpt-4")
	}
}

func TestConversationHistory_SetMaxMessages(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	tests := []struct {
		name        string
		maxMessages int
		wantError   bool
	}{
		{
			name:        "valid_max_messages",
			maxMessages: 500,
			wantError:   false,
		},
		{
			name:        "max_messages_too_low",
			maxMessages: 0,
			wantError:   true,
		},
		{
			name:        "max_messages_too_high",
			maxMessages: 20000,
			wantError:   true,
		},
		{
			name:        "minimum_max_messages",
			maxMessages: MinMaxMessages,
			wantError:   false,
		},
		{
			name:        "maximum_max_messages",
			maxMessages: MaxMaxMessages,
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := history.SetMaxMessages(tt.maxMessages)

			if tt.wantError {
				if err == nil {
					t.Error("SetMaxMessages() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SetMaxMessages() error = %v, want nil", err)
				}
				if history.GetMaxMessages() != tt.maxMessages {
					t.Errorf("SetMaxMessages() MaxMessages = %v, want %v", history.GetMaxMessages(), tt.maxMessages)
				}
			}
		})
	}
}

func TestConversationHistory_GetMaxMessages(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	maxMessages := history.GetMaxMessages()
	if maxMessages != DefaultMaxMessages {
		t.Errorf("GetMaxMessages() = %v, want %v", maxMessages, DefaultMaxMessages)
	}
}

func TestConversationHistory_Clear(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Add some messages and context
	_, err = history.AddUserMessage("Hello", nil)
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}
	err = history.SetContext("key", "value")
	if err != nil {
		t.Fatalf("SetContext() error = %v", err)
	}

	// Clear everything
	history.Clear()

	// Check that messages and context are cleared
	if !history.IsEmpty() {
		t.Error("Clear() conversation should be empty after clearing")
	}

	_, exists := history.GetContext("key")
	if exists {
		t.Error("Clear() context should be cleared")
	}
}

func TestConversationHistory_GetStats(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Add messages with different roles
	_, err = history.AddUserMessage("Hello", nil)
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}
	_, err = history.AddAssistantMessage("Hi there!", nil)
	if err != nil {
		t.Fatalf("AddAssistantMessage() error = %v", err)
	}
	_, err = history.AddSystemMessage("You are helpful", nil)
	if err != nil {
		t.Fatalf("AddSystemMessage() error = %v", err)
	}
	_, err = history.AddUserMessage("How are you?", nil)
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}

	stats := history.GetStats()

	if stats.SessionID != "test-session-123" {
		t.Errorf("GetStats() SessionID = %v, want test-session-123", stats.SessionID)
	}
	if stats.MessageCount != 4 {
		t.Errorf("GetStats() MessageCount = %v, want 4", stats.MessageCount)
	}
	if stats.UserMessages != 2 {
		t.Errorf("GetStats() UserMessages = %v, want 2", stats.UserMessages)
	}
	if stats.AssistantMessages != 1 {
		t.Errorf("GetStats() AssistantMessages = %v, want 1", stats.AssistantMessages)
	}
	if stats.SystemMessages != 1 {
		t.Errorf("GetStats() SystemMessages = %v, want 1", stats.SystemMessages)
	}
	expectedLength := len("Hello") + len("Hi there!") + len("You are helpful") + len("How are you?")
	if stats.TotalLength != expectedLength {
		t.Errorf("GetStats() TotalLength = %v, want %v", stats.TotalLength, expectedLength)
	}
	if stats.MaxMessages != DefaultMaxMessages {
		t.Errorf("GetStats() MaxMessages = %v, want %v", stats.MaxMessages, DefaultMaxMessages)
	}
}

func TestConversationHistory_GetMessageCount(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Initially empty
	if history.GetMessageCount() != 0 {
		t.Errorf("GetMessageCount() = %v, want 0", history.GetMessageCount())
	}

	// Add some messages
	_, err = history.AddUserMessage("Message 1", nil)
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}
	_, err = history.AddUserMessage("Message 2", nil)
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}

	if history.GetMessageCount() != 2 {
		t.Errorf("GetMessageCount() = %v, want 2", history.GetMessageCount())
	}
}

func TestConversationHistory_IsEmpty(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Initially empty
	if !history.IsEmpty() {
		t.Error("IsEmpty() should return true for new conversation")
	}

	// Add a message
	_, err = history.AddUserMessage("Hello", nil)
	if err != nil {
		t.Fatalf("AddUserMessage() error = %v", err)
	}

	if history.IsEmpty() {
		t.Error("IsEmpty() should return false after adding message")
	}
}

func TestConversationHistory_MessageTrimming(t *testing.T) {
	// Create history with small max messages
	history, err := NewConversationHistoryWithMax("test-session-123", 3)
	if err != nil {
		t.Fatalf("NewConversationHistoryWithMax() error = %v", err)
	}

	// Add more messages than the limit
	for i := 0; i < 5; i++ {
		_, err := history.AddUserMessage("Message "+string(rune('0'+i)), nil)
		if err != nil {
			t.Fatalf("AddUserMessage() error = %v", err)
		}
	}

	// Should only keep the last 3 messages
	if history.GetMessageCount() != 3 {
		t.Errorf("GetMessageCount() = %v, want 3", history.GetMessageCount())
	}

	// Check that the oldest messages were trimmed
	messages := history.GetRecentMessages(3)
	if len(messages) != 3 {
		t.Errorf("GetRecentMessages() length = %v, want 3", len(messages))
	}

	// The messages should be "Message 2", "Message 3", "Message 4"
	expectedContents := []string{"Message 2", "Message 3", "Message 4"}
	for i, msg := range messages {
		if msg.Content != expectedContents[i] {
			t.Errorf("GetRecentMessages() message %d content = %v, want %v", i, msg.Content, expectedContents[i])
		}
	}
}

func TestConversationHistory_Concurrency(t *testing.T) {
	history, err := NewConversationHistory("test-session-123")
	if err != nil {
		t.Fatalf("NewConversationHistory() error = %v", err)
	}

	// Test concurrent access
	done := make(chan bool, 2)

	// Goroutine 1: Add messages
	go func() {
		for i := 0; i < 10; i++ {
			_, err := history.AddUserMessage("Message "+string(rune('0'+i)), nil)
			if err != nil {
				t.Errorf("AddUserMessage() error = %v", err)
			}
		}
		done <- true
	}()

	// Goroutine 2: Read messages
	go func() {
		for i := 0; i < 10; i++ {
			_ = history.GetRecentMessages(5)
			_ = history.GetMessageCount()
			_ = history.IsEmpty()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify final state
	if history.GetMessageCount() != 10 {
		t.Errorf("GetMessageCount() = %v, want 10", history.GetMessageCount())
	}
}

func TestConversationError_Error(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		operation string
		message   string
		err       error
		expected  string
	}{
		{
			name:      "error_with_wrapped_error",
			sessionID: "test-session",
			operation: "AddMessage",
			message:   "failed to add message",
			err:       fmt.Errorf("validation failed"),
			expected:  "[test-session:AddMessage] failed to add message: validation failed",
		},
		{
			name:      "error_without_wrapped_error",
			sessionID: "test-session",
			operation: "AddMessage",
			message:   "failed to add message",
			err:       nil,
			expected:  "[test-session:AddMessage] failed to add message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convErr := NewConversationError(tt.sessionID, tt.operation, tt.message, tt.err)
			errorStr := convErr.Error()

			if errorStr != tt.expected {
				t.Errorf("ConversationError.Error() = %v, want %v", errorStr, tt.expected)
			}
		})
	}
}

func TestConversationError_Unwrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	convErr := NewConversationError("test-session", "AddMessage", "failed", originalErr)

	unwrapped := convErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("ConversationError.Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestNewConversationError(t *testing.T) {
	sessionID := "test-session"
	operation := "AddMessage"
	message := "test error"
	err := fmt.Errorf("wrapped error")

	convErr := NewConversationError(sessionID, operation, message, err)

	if convErr.SessionID != sessionID {
		t.Errorf("NewConversationError() SessionID = %v, want %v", convErr.SessionID, sessionID)
	}
	if convErr.Operation != operation {
		t.Errorf("NewConversationError() Operation = %v, want %v", convErr.Operation, operation)
	}
	if convErr.Message != message {
		t.Errorf("NewConversationError() Message = %v, want %v", convErr.Message, message)
	}
	if convErr.Err != err {
		t.Errorf("NewConversationError() Err = %v, want %v", convErr.Err, err)
	}
	if convErr.Timestamp.IsZero() {
		t.Error("NewConversationError() Timestamp should not be zero")
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
