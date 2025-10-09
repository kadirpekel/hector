package memory

import (
	"sync"
	"time"

	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/llms"
)

// MemoryService manages conversation memory across sessions
// It orchestrates working memory (current conversation) and will later support long-term memory
type MemoryService struct {
	workingMemory WorkingMemoryStrategy
	sessions      map[string]*hectorcontext.ConversationHistory
	mu            sync.RWMutex
}

// NewMemoryService creates a new memory service with the given working memory strategy
func NewMemoryService(strategy WorkingMemoryStrategy) *MemoryService {
	return &MemoryService{
		workingMemory: strategy,
		sessions:      make(map[string]*hectorcontext.ConversationHistory),
	}
}

// AddToHistory adds a message to the session's memory
func (s *MemoryService) AddToHistory(sessionID string, msg llms.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	session := s.getOrCreateSession(sessionID)
	return s.workingMemory.AddMessage(session, msg)
}

// GetRecentHistory returns messages from the session
func (s *MemoryService) GetRecentHistory(sessionID string) ([]llms.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sessionID == "" {
		sessionID = "default"
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		return []llms.Message{}, nil
	}

	return s.workingMemory.GetMessages(session)
}

// ClearHistory removes all memory for a session
func (s *MemoryService) ClearHistory(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	delete(s.sessions, sessionID)
	return nil
}

// GetSessionCount returns the number of active sessions
func (s *MemoryService) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.sessions)
}

// SetStatusNotifier sets a status notifier on the working memory strategy
func (s *MemoryService) SetStatusNotifier(notifier StatusNotifier) {
	s.workingMemory.SetStatusNotifier(notifier)
}

// getOrCreateSession gets an existing session or creates a new one
// Must be called with lock held
func (s *MemoryService) getOrCreateSession(sessionID string) *hectorcontext.ConversationHistory {
	session, exists := s.sessions[sessionID]
	if !exists {
		now := time.Now()
		session = &hectorcontext.ConversationHistory{
			SessionID:   sessionID,
			Messages:    make([]hectorcontext.Message, 0),
			Context:     make(map[string]interface{}),
			LastUpdated: now,
			MaxMessages: 1000, // Max 1000 messages per session
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.sessions[sessionID] = session
	}
	return session
}
