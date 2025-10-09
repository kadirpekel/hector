package history

import (
	"sync"
	"time"

	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/llms"
)

// BufferWindowStrategy implements simple LIFO history management
// Keeps last N messages, drops oldest when window is full
type BufferWindowStrategy struct {
	sessions   map[string]*hectorcontext.ConversationHistory
	mu         sync.RWMutex
	windowSize int // Default: 20
}

// BufferWindowConfig configures the buffer window strategy
type BufferWindowConfig struct {
	WindowSize int // Number of messages to keep (default: 20)
}

// NewBufferWindowStrategy creates a new buffer window strategy
func NewBufferWindowStrategy(config BufferWindowConfig) (*BufferWindowStrategy, error) {
	if config.WindowSize <= 0 {
		config.WindowSize = 20 // Sensible default
	}

	return &BufferWindowStrategy{
		sessions:   make(map[string]*hectorcontext.ConversationHistory),
		windowSize: config.WindowSize,
	}, nil
}

// Name returns the strategy identifier
func (s *BufferWindowStrategy) Name() string {
	return "buffer_window"
}

// AddMessage adds a message to the session's history
func (s *BufferWindowStrategy) AddMessage(sessionID string, msg llms.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	session := s.getOrCreateSession(sessionID)
	_, err := session.AddMessage(msg.Role, msg.Content, nil)
	return err
}

// GetHistory returns messages for the session
func (s *BufferWindowStrategy) GetHistory(sessionID string) ([]llms.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sessionID == "" {
		sessionID = "default"
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		return []llms.Message{}, nil
	}

	// Get recent messages within window size
	contextMessages := session.GetRecentMessages(s.windowSize)

	// Convert to llms.Message format
	messages := make([]llms.Message, len(contextMessages))
	for i, msg := range contextMessages {
		messages[i] = llms.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return messages, nil
}

// Clear removes all history for a session
func (s *BufferWindowStrategy) Clear(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	if session, exists := s.sessions[sessionID]; exists {
		session.Clear()
	}

	return nil
}

// GetSessionCount returns the number of active sessions
func (s *BufferWindowStrategy) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.sessions)
}

// getOrCreateSession gets an existing session or creates a new one
// Must be called with lock held
func (s *BufferWindowStrategy) getOrCreateSession(sessionID string) *hectorcontext.ConversationHistory {
	session, exists := s.sessions[sessionID]
	if !exists {
		now := time.Now()
		session = &hectorcontext.ConversationHistory{
			SessionID:   sessionID,
			Messages:    make([]hectorcontext.Message, 0),
			Context:     make(map[string]interface{}),
			LastUpdated: now,
			MaxMessages: s.windowSize,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.sessions[sessionID] = session
	}
	return session
}
