package context

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// ConversationHistory is an IN-MEMORY working buffer for conversation messages
//
// PURPOSE:
//   - Temporary storage during strategy operations (e.g., summarization)
//   - Message manipulation (add, remove, truncate)
//   - In-memory message accumulation
//
// THIS IS NOT:
//
//	❌ A cache (doesn't fetch from persistent storage)
//	❌ A persistent store (lost on restart)
//	❌ A source of truth (SessionService is the source of truth)
//
// OWNERSHIP:
//   - Created by WorkingMemoryStrategy during LoadState
//   - Used temporarily for message management
//   - Discarded after strategy operations complete
//
// LIFECYCLE:
//  1. Strategy calls LoadState()
//  2. ConversationHistory created and populated
//  3. Strategy uses it for operations (e.g., CheckAndSummarize)
//  4. Results returned, ConversationHistory discarded
//
// See pkg/memory/README.md for complete ownership model
type ConversationHistory struct {
	mu          sync.RWMutex
	SessionID   string
	Messages    []*pb.Message
	Context     map[string]interface{}
	LastUpdated time.Time
	MaxMessages int // Safety limit to prevent memory exhaustion
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewConversationHistory(sessionID string) (*ConversationHistory, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}

	now := time.Now()
	return &ConversationHistory{
		SessionID:   sessionID,
		Messages:    make([]*pb.Message, 0),
		Context:     make(map[string]interface{}),
		LastUpdated: now,
		MaxMessages: 1000,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func NewConversationHistoryWithMax(sessionID string, maxMessages int) (*ConversationHistory, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID is required")
	}
	if maxMessages < 1 || maxMessages > 10000 {
		return nil, fmt.Errorf("invalid max messages: %d", maxMessages)
	}

	now := time.Now()
	return &ConversationHistory{
		SessionID:   sessionID,
		Messages:    make([]*pb.Message, 0),
		Context:     make(map[string]interface{}),
		LastUpdated: now,
		MaxMessages: maxMessages,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (ch *ConversationHistory) AddMessage(msg *pb.Message) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	if msg.MessageId == "" {
		msg.MessageId = generateMessageID(ch.SessionID)
	}

	if msg.ContextId == "" {
		msg.ContextId = ch.SessionID
	}

	ch.Messages = append(ch.Messages, msg)
	ch.trimMessagesIfNeeded()
	ch.updateTimestamps()

	return nil
}

func (ch *ConversationHistory) GetRecentMessages(n int) []*pb.Message {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if n <= 0 || len(ch.Messages) == 0 {
		return []*pb.Message{}
	}

	start := len(ch.Messages) - n
	if start < 0 {
		start = 0
	}

	return ch.Messages[start:]
}

func (ch *ConversationHistory) GetAllMessages() []*pb.Message {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	return ch.Messages
}

func (ch *ConversationHistory) GetMessagesByRole(role pb.Role, limit int) []*pb.Message {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	var filtered []*pb.Message
	count := 0

	for i := len(ch.Messages) - 1; i >= 0 && (limit <= 0 || count < limit); i-- {
		if ch.Messages[i].Role == role {
			filtered = append([]*pb.Message{ch.Messages[i]}, filtered...)
			count++
		}
	}

	return filtered
}

func (ch *ConversationHistory) GetMessageByID(id string) (*pb.Message, bool) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	for _, msg := range ch.Messages {
		if msg.MessageId == id {
			return msg, true
		}
	}

	return nil, false
}

func (ch *ConversationHistory) SetContext(key string, value interface{}) error {
	if key == "" {
		return fmt.Errorf("context key cannot be empty")
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.Context[key] = value
	ch.updateTimestamps()

	return nil
}

func (ch *ConversationHistory) GetContext(key string) (interface{}, bool) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	value, exists := ch.Context[key]
	return value, exists
}

func (ch *ConversationHistory) RemoveContext(key string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	delete(ch.Context, key)
	ch.updateTimestamps()
}

func (ch *ConversationHistory) ClearContext() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.Context = make(map[string]interface{})
	ch.updateTimestamps()
}

func (ch *ConversationHistory) SetMaxMessages(maxMessages int) error {
	if maxMessages < 1 || maxMessages > 10000 {
		return fmt.Errorf("invalid max messages: %d", maxMessages)
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.MaxMessages = maxMessages
	ch.trimMessagesIfNeeded()
	ch.updateTimestamps()

	return nil
}

func (ch *ConversationHistory) GetMaxMessages() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.MaxMessages
}

func (ch *ConversationHistory) Clear() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.Messages = make([]*pb.Message, 0)
	ch.Context = make(map[string]interface{})
	ch.updateTimestamps()
}

func (ch *ConversationHistory) GetMessageCount() int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return len(ch.Messages)
}

func (ch *ConversationHistory) IsEmpty() bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return len(ch.Messages) == 0
}

func (ch *ConversationHistory) GetStats() map[string]interface{} {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["session_id"] = ch.SessionID
	stats["message_count"] = len(ch.Messages)
	stats["created_at"] = ch.CreatedAt
	stats["updated_at"] = ch.UpdatedAt
	stats["max_messages"] = ch.MaxMessages

	roleCounts := make(map[string]int)
	for _, msg := range ch.Messages {
		roleCounts[msg.Role.String()]++
	}
	stats["role_counts"] = roleCounts

	return stats
}

func (ch *ConversationHistory) trimMessagesIfNeeded() {
	if len(ch.Messages) > ch.MaxMessages {
		ch.Messages = ch.Messages[len(ch.Messages)-ch.MaxMessages:]
	}
}

func (ch *ConversationHistory) updateTimestamps() {
	now := time.Now()
	ch.LastUpdated = now
	ch.UpdatedAt = now
}

func generateMessageID(sessionID string) string {
	return fmt.Sprintf("%s-%s", sessionID, uuid.New().String()[:8])
}
