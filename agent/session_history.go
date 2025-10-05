package agent

import (
	"fmt"
	"sync"
	"time"

	hectorcontext "github.com/kadirpekel/hector/context"
	"github.com/kadirpekel/hector/llms"
	"github.com/kadirpekel/hector/reasoning"
)

// ============================================================================
// SESSION-AWARE HISTORY SERVICE
// Manages conversation history across multiple sessions
// ============================================================================

// SessionHistoryService implements reasoning.HistoryService with session awareness
// Each agent instance has ONE SessionHistoryService that manages MULTIPLE sessions
// This is a STATELESS API - sessionID is passed explicitly to all methods
type SessionHistoryService struct {
	sessions map[string]*hectorcontext.ConversationHistory
	maxSize  int // Max messages per session
	mu       sync.RWMutex
}

// NewSessionHistoryService creates a new session-aware history service
func NewSessionHistoryService(maxSize int) reasoning.HistoryService {
	if maxSize <= 0 {
		maxSize = 10 // Default max size
	}
	return &SessionHistoryService{
		sessions: make(map[string]*hectorcontext.ConversationHistory),
		maxSize:  maxSize,
	}
}

// getOrCreateSession gets an existing session or creates a new one
// Must be called with lock held
func (s *SessionHistoryService) getOrCreateSession(sessionID string) *hectorcontext.ConversationHistory {
	if sessionID == "" {
		sessionID = "default"
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		// NewConversationHistoryWithMax returns error, but we handle it gracefully
		var err error
		session, err = hectorcontext.NewConversationHistoryWithMax(sessionID, s.maxSize)
		if err != nil || session == nil {
			// Fallback: create session with default max
			session, err = hectorcontext.NewConversationHistory(sessionID)
			if err != nil || session == nil {
				// Last resort: create minimal session
				fmt.Printf("⚠️  Failed to create session %s: %v, using minimal fallback\n", sessionID, err)
				// Create a minimal working session manually
				now := time.Now()
				session = &hectorcontext.ConversationHistory{
					SessionID:   sessionID,
					Messages:    make([]hectorcontext.Message, 0),
					Context:     make(map[string]interface{}),
					LastUpdated: now,
					MaxMessages: s.maxSize,
					CreatedAt:   now,
					UpdatedAt:   now,
				}
			}
		}
		s.sessions[sessionID] = session
	}
	return session
}

// GetRecentHistory implements reasoning.HistoryService
func (s *SessionHistoryService) GetRecentHistory(sessionID string, count int) []llms.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sessionID == "" {
		sessionID = "default"
	}

	// Get session's conversation history
	session, exists := s.sessions[sessionID]
	if !exists {
		return []llms.Message{}
	}

	// Get recent messages from the session
	messages := session.GetRecentMessages(count)

	// Convert to llms.Message format
	result := make([]llms.Message, 0, len(messages))
	for _, msg := range messages {
		result = append(result, llms.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return result
}

// AddToHistory implements reasoning.HistoryService
func (s *SessionHistoryService) AddToHistory(sessionID string, msg llms.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	// Get or create session
	session := s.getOrCreateSession(sessionID)

	// Add message to session
	_, err := session.AddMessage(msg.Role, msg.Content, nil)
	if err != nil {
		fmt.Printf("⚠️  Failed to add message to session %s: %v\n", sessionID, err)
	}
}

// ClearHistory implements reasoning.HistoryService
// Clears the specified session's history
func (s *SessionHistoryService) ClearHistory(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" {
		sessionID = "default"
	}

	// Clear specified session
	if session, exists := s.sessions[sessionID]; exists {
		session.Clear()
	}
}

// ClearAllSessions clears all sessions (useful for testing/cleanup)
func (s *SessionHistoryService) ClearAllSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = make(map[string]*hectorcontext.ConversationHistory)
}

// GetSessionCount returns the number of active sessions (useful for monitoring)
func (s *SessionHistoryService) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.sessions)
}

// ============================================================================
// COMPILE-TIME CHECK
// ============================================================================

var _ reasoning.HistoryService = (*SessionHistoryService)(nil)
