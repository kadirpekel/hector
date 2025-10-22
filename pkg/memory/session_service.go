package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// InMemorySessionService provides an in-memory implementation of reasoning.SessionService
// This is useful for agents that don't need persistent session storage
type InMemorySessionService struct {
	sessions map[string]*SessionData
	mu       sync.RWMutex
}

// SessionData holds messages and metadata for a single session
type SessionData struct {
	Messages []*pb.Message
	Metadata *reasoning.SessionMetadata
}

// NewInMemorySessionService creates a new in-memory session service
func NewInMemorySessionService() *InMemorySessionService {
	return &InMemorySessionService{
		sessions: make(map[string]*SessionData),
	}
}

// AppendMessage appends a message to a session
func (s *InMemorySessionService) AppendMessage(sessionID string, message *pb.Message) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create session
	session, exists := s.sessions[sessionID]
	if !exists {
		session = &SessionData{
			Messages: make([]*pb.Message, 0),
			Metadata: &reasoning.SessionMetadata{
				ID:        sessionID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Metadata:  make(map[string]interface{}),
			},
		}
		s.sessions[sessionID] = session
	}

	// Append message
	session.Messages = append(session.Messages, message)
	session.Metadata.UpdatedAt = time.Now()

	return nil
}

// AppendMessages appends multiple messages atomically
// For in-memory implementation, this is equivalent to multiple AppendMessage calls
// but done atomically under a single lock
func (s *InMemorySessionService) AppendMessages(sessionID string, messages []*pb.Message) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}
	if len(messages) == 0 {
		return nil // Nothing to append
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create session
	session, exists := s.sessions[sessionID]
	if !exists {
		session = &SessionData{
			Messages: make([]*pb.Message, 0),
			Metadata: &reasoning.SessionMetadata{
				ID:        sessionID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Metadata:  make(map[string]interface{}),
			},
		}
		s.sessions[sessionID] = session
	}

	// Append all messages atomically
	session.Messages = append(session.Messages, messages...)
	session.Metadata.UpdatedAt = time.Now()

	return nil
}

// GetMessages returns the most recent messages from a session
func (s *InMemorySessionService) GetMessages(sessionID string, limit int) ([]*pb.Message, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return []*pb.Message{}, nil
	}

	messages := session.Messages
	if limit > 0 && len(messages) > limit {
		// Return last N messages
		return messages[len(messages)-limit:], nil
	}

	return messages, nil
}

// GetMessagesWithOptions returns messages with advanced filtering
// For in-memory implementation, this provides basic limit support
// Role filtering and FromMessageID are not implemented for in-memory
func (s *InMemorySessionService) GetMessagesWithOptions(sessionID string, opts reasoning.LoadOptions) ([]*pb.Message, error) {
	// For in-memory, just use limit
	return s.GetMessages(sessionID, opts.Limit)
}

// GetMessageCount returns the number of messages in a session
func (s *InMemorySessionService) GetMessageCount(sessionID string) (int, error) {
	if sessionID == "" {
		return 0, fmt.Errorf("sessionID cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return 0, nil
	}

	return len(session.Messages), nil
}

// GetOrCreateSessionMetadata returns or creates session metadata
func (s *InMemorySessionService) GetOrCreateSessionMetadata(sessionID string) (*reasoning.SessionMetadata, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		// Create new session
		session = &SessionData{
			Messages: make([]*pb.Message, 0),
			Metadata: &reasoning.SessionMetadata{
				ID:        sessionID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Metadata:  make(map[string]interface{}),
			},
		}
		s.sessions[sessionID] = session
	}

	return session.Metadata, nil
}

// DeleteSession deletes a session
func (s *InMemorySessionService) DeleteSession(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

// SessionCount returns the number of active sessions
func (s *InMemorySessionService) SessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.sessions)
}
