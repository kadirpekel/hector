package context

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// CONVERSATION CONSTANTS AND CONFIGURATION
// ============================================================================

const (
	// MinMaxMessages is the minimum allowed max messages
	MinMaxMessages = 1

	// MaxMaxMessages is the maximum allowed max messages
	MaxMaxMessages = 10000

	// MinMessageLength is the minimum message length
	MinMessageLength = 1

	// MaxMessageLength is the maximum message length
	MaxMessageLength = 100000

	// DefaultMaxMessages is the default maximum number of messages
	DefaultMaxMessages = 1000
)

// Message roles
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

// ============================================================================
// CONVERSATION ERRORS - STANDARDIZED ERROR TYPES
// ============================================================================

// ConversationError represents errors in conversation operations
type ConversationError struct {
	SessionID string
	Operation string
	Message   string
	Err       error
	Timestamp time.Time
}

func (e *ConversationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.SessionID, e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.SessionID, e.Operation, e.Message)
}

func (e *ConversationError) Unwrap() error {
	return e.Err
}

// NewConversationError creates a new conversation error
func NewConversationError(sessionID, operation, message string, err error) *ConversationError {
	return &ConversationError{
		SessionID: sessionID,
		Operation: operation,
		Message:   message,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// ============================================================================
// CONVERSATION TYPES AND STRUCTURES
// ============================================================================

// Message represents a single message in the conversation
type Message struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ConversationHistory manages conversation state and history with enhanced features
type ConversationHistory struct {
	mu          sync.RWMutex
	SessionID   string                 `json:"session_id"`
	Messages    []Message              `json:"messages"`
	Context     map[string]interface{} `json:"context"`
	LastUpdated time.Time              `json:"last_updated"`
	MaxMessages int                    `json:"max_messages"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ConversationStats represents statistics about a conversation
type ConversationStats struct {
	SessionID         string    `json:"session_id"`
	MessageCount      int       `json:"message_count"`
	UserMessages      int       `json:"user_messages"`
	AssistantMessages int       `json:"assistant_messages"`
	SystemMessages    int       `json:"system_messages"`
	TotalLength       int       `json:"total_length"`
	CreatedAt         time.Time `json:"created_at"`
	LastUpdated       time.Time `json:"last_updated"`
	MaxMessages       int       `json:"max_messages"`
}

// ============================================================================
// CONVERSATION HISTORY - ENHANCED CONSTRUCTOR
// ============================================================================

// NewConversationHistory creates a new conversation history with validation
func NewConversationHistory(sessionID string) (*ConversationHistory, error) {
	if sessionID == "" {
		return nil, NewConversationError("", "NewConversationHistory", "session ID is required", nil)
	}

	now := time.Now()
	return &ConversationHistory{
		SessionID:   sessionID,
		Messages:    make([]Message, 0),
		Context:     make(map[string]interface{}),
		LastUpdated: now,
		MaxMessages: DefaultMaxMessages,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// NewConversationHistoryWithMax creates a new conversation history with custom max messages
func NewConversationHistoryWithMax(sessionID string, maxMessages int) (*ConversationHistory, error) {
	if sessionID == "" {
		return nil, NewConversationError("", "NewConversationHistoryWithMax", "session ID is required", nil)
	}
	if maxMessages < MinMaxMessages || maxMessages > MaxMaxMessages {
		return nil, NewConversationError(sessionID, "NewConversationHistoryWithMax", "invalid max messages", nil)
	}

	now := time.Now()
	return &ConversationHistory{
		SessionID:   sessionID,
		Messages:    make([]Message, 0),
		Context:     make(map[string]interface{}),
		LastUpdated: now,
		MaxMessages: maxMessages,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// ============================================================================
// MESSAGE MANAGEMENT - ENHANCED WITH VALIDATION
// ============================================================================

// AddMessage adds a message to the conversation history with validation
func (ch *ConversationHistory) AddMessage(role, content string, metadata map[string]interface{}) (*Message, error) {
	// Validate inputs
	if err := ch.validateMessageInputs(role, content); err != nil {
		return nil, err
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Create message
	message := Message{
		ID:        ch.generateMessageID(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  ch.prepareMetadata(metadata),
	}

	// Add message
	ch.Messages = append(ch.Messages, message)
	ch.trimMessagesIfNeeded()
	ch.updateTimestamps()

	return &message, nil
}

// AddUserMessage adds a user message with validation
func (ch *ConversationHistory) AddUserMessage(content string, metadata map[string]interface{}) (*Message, error) {
	return ch.AddMessage(RoleUser, content, metadata)
}

// AddAssistantMessage adds an assistant message with validation
func (ch *ConversationHistory) AddAssistantMessage(content string, metadata map[string]interface{}) (*Message, error) {
	return ch.AddMessage(RoleAssistant, content, metadata)
}

// AddSystemMessage adds a system message with validation
func (ch *ConversationHistory) AddSystemMessage(content string, metadata map[string]interface{}) (*Message, error) {
	return ch.AddMessage(RoleSystem, content, metadata)
}

// ============================================================================
// MESSAGE RETRIEVAL - ENHANCED WITH OPTIONS
// ============================================================================

// GetRecentMessages returns the last N messages with validation
func (ch *ConversationHistory) GetRecentMessages(n int) []Message {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if n <= 0 || len(ch.Messages) == 0 {
		return []Message{}
	}

	start := len(ch.Messages) - n
	if start < 0 {
		start = 0
	}

	// Return a copy to prevent external modification
	messages := make([]Message, len(ch.Messages[start:]))
	copy(messages, ch.Messages[start:])
	return messages
}

// GetMessagesByRole returns messages filtered by role
func (ch *ConversationHistory) GetMessagesByRole(role string, limit int) []Message {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	var filtered []Message
	count := 0

	// Iterate from most recent to oldest
	for i := len(ch.Messages) - 1; i >= 0 && (limit <= 0 || count < limit); i-- {
		if ch.Messages[i].Role == role {
			filtered = append([]Message{ch.Messages[i]}, filtered...)
			count++
		}
	}

	return filtered
}

// GetMessageByID returns a message by its ID
func (ch *ConversationHistory) GetMessageByID(id string) (*Message, bool) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	for _, msg := range ch.Messages {
		if msg.ID == id {
			// Return a copy to prevent external modification
			messageCopy := msg
			return &messageCopy, true
		}
	}

	return nil, false
}

// ConversationMessage represents a conversation message (moved here from interfaces)
type ConversationMessage struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// GetRecentConversationMessages returns the last N messages as ConversationMessage for interface compatibility
func (ch *ConversationHistory) GetRecentConversationMessages(n int) []ConversationMessage {
	messages := ch.GetRecentMessages(n)
	result := make([]ConversationMessage, len(messages))
	for i, msg := range messages {
		result[i] = ConversationMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
			Metadata:  msg.Metadata,
		}
	}
	return result
}

// ============================================================================
// CONTEXT MANAGEMENT - ENHANCED
// ============================================================================

// SetContext sets conversation context with validation
func (ch *ConversationHistory) SetContext(key string, value interface{}) error {
	if key == "" {
		return NewConversationError(ch.SessionID, "SetContext", "context key cannot be empty", nil)
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.Context[key] = value
	ch.updateTimestamps()

	return nil
}

// GetContext gets conversation context
func (ch *ConversationHistory) GetContext(key string) (interface{}, bool) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	value, exists := ch.Context[key]
	return value, exists
}

// RemoveContext removes a context key
func (ch *ConversationHistory) RemoveContext(key string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	delete(ch.Context, key)
	ch.updateTimestamps()
}

// ClearContext clears all context
func (ch *ConversationHistory) ClearContext() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.Context = make(map[string]interface{})
	ch.updateTimestamps()
}

// ============================================================================
// CONTEXT FORMATTING - ENHANCED
// ============================================================================

// GetContextForLLM returns formatted context for LLM with enhanced formatting
func (ch *ConversationHistory) GetContextForLLM(maxMessages int) string {
	recentMessages := ch.GetRecentMessages(maxMessages)

	var context strings.Builder
	context.Grow(len(recentMessages) * 100) // Pre-allocate capacity

	context.WriteString("Conversation History:\n")

	for _, msg := range recentMessages {
		context.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	return context.String()
}

// GetContextForLLMWithMetadata returns formatted context including metadata
func (ch *ConversationHistory) GetContextForLLMWithMetadata(maxMessages int) string {
	recentMessages := ch.GetRecentMessages(maxMessages)

	var context strings.Builder
	context.Grow(len(recentMessages) * 150) // Pre-allocate capacity

	context.WriteString("Conversation History:\n")

	for _, msg := range recentMessages {
		context.WriteString(fmt.Sprintf("%s: %s", msg.Role, msg.Content))

		// Add metadata if present
		if len(msg.Metadata) > 0 {
			context.WriteString(" [")
			first := true
			for k, v := range msg.Metadata {
				if !first {
					context.WriteString(", ")
				}
				context.WriteString(fmt.Sprintf("%s=%v", k, v))
				first = false
			}
			context.WriteString("]")
		}
		context.WriteString("\n")
	}

	return context.String()
}

// ============================================================================
// CONFIGURATION AND MANAGEMENT
// ============================================================================

// SetMaxMessages sets the maximum number of messages to keep
func (ch *ConversationHistory) SetMaxMessages(maxMessages int) error {
	if maxMessages < MinMaxMessages || maxMessages > MaxMaxMessages {
		return NewConversationError(ch.SessionID, "SetMaxMessages", "invalid max messages", nil)
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.MaxMessages = maxMessages
	ch.trimMessagesIfNeeded()
	ch.updateTimestamps()

	return nil
}

// GetMaxMessages returns the current maximum number of messages
func (ch *ConversationHistory) GetMaxMessages() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.MaxMessages
}

// Clear clears all messages and context
func (ch *ConversationHistory) Clear() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.Messages = make([]Message, 0)
	ch.Context = make(map[string]interface{})
	ch.updateTimestamps()
}

// ============================================================================
// STATISTICS AND MONITORING
// ============================================================================

// GetStats returns detailed conversation statistics
func (ch *ConversationHistory) GetStats() *ConversationStats {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	stats := &ConversationStats{
		SessionID:    ch.SessionID,
		MessageCount: len(ch.Messages),
		CreatedAt:    ch.CreatedAt,
		LastUpdated:  ch.UpdatedAt,
		MaxMessages:  ch.MaxMessages,
	}

	// Count messages by role
	for _, msg := range ch.Messages {
		stats.TotalLength += len(msg.Content)
		switch msg.Role {
		case RoleUser:
			stats.UserMessages++
		case RoleAssistant:
			stats.AssistantMessages++
		case RoleSystem:
			stats.SystemMessages++
		}
	}

	return stats
}

// GetMessageCount returns the current number of messages
func (ch *ConversationHistory) GetMessageCount() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return len(ch.Messages)
}

// IsEmpty returns true if the conversation has no messages
func (ch *ConversationHistory) IsEmpty() bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return len(ch.Messages) == 0
}

// ============================================================================
// VALIDATION AND HELPER METHODS
// ============================================================================

// validateMessageInputs validates message inputs
func (ch *ConversationHistory) validateMessageInputs(role, content string) error {
	if role == "" {
		return NewConversationError(ch.SessionID, "validateMessageInputs", "role cannot be empty", nil)
	}
	if content == "" {
		return NewConversationError(ch.SessionID, "validateMessageInputs", "content cannot be empty", nil)
	}
	if len(content) < MinMessageLength {
		return NewConversationError(ch.SessionID, "validateMessageInputs", "content too short", nil)
	}
	if len(content) > MaxMessageLength {
		return NewConversationError(ch.SessionID, "validateMessageInputs", "content too long", nil)
	}

	// Validate role
	switch role {
	case RoleUser, RoleAssistant, RoleSystem:
		// Valid roles
	default:
		return NewConversationError(ch.SessionID, "validateMessageInputs", "invalid role", nil)
	}

	return nil
}

// trimMessagesIfNeeded trims messages if they exceed the limit
func (ch *ConversationHistory) trimMessagesIfNeeded() {
	if len(ch.Messages) > ch.MaxMessages {
		ch.Messages = ch.Messages[len(ch.Messages)-ch.MaxMessages:]
	}
}

// updateTimestamps updates the timestamps
func (ch *ConversationHistory) updateTimestamps() {
	now := time.Now()
	ch.LastUpdated = now
	ch.UpdatedAt = now
}

// prepareMetadata prepares metadata for a message
func (ch *ConversationHistory) prepareMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		return make(map[string]interface{})
	}

	// Create a copy to prevent external modification
	prepared := make(map[string]interface{})
	for k, v := range metadata {
		prepared[k] = v
	}

	return prepared
}

// generateMessageID generates a unique message ID
func (ch *ConversationHistory) generateMessageID() string {
	return fmt.Sprintf("msg_%s_%d", ch.SessionID, time.Now().UnixNano())
}
