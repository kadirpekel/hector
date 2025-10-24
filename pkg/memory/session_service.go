package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

type InMemorySessionService struct {
	sessions map[string]*SessionData
	mu       sync.RWMutex
}

type SessionData struct {
	Messages []*pb.Message
	Metadata *reasoning.SessionMetadata
}

func NewInMemorySessionService() *InMemorySessionService {
	return &InMemorySessionService{
		sessions: make(map[string]*SessionData),
	}
}

func (s *InMemorySessionService) AppendMessage(sessionID string, message *pb.Message) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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

	session.Messages = append(session.Messages, message)
	session.Metadata.UpdatedAt = time.Now()

	return nil
}

func (s *InMemorySessionService) AppendMessages(sessionID string, messages []*pb.Message) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}
	if len(messages) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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

	session.Messages = append(session.Messages, messages...)
	session.Metadata.UpdatedAt = time.Now()

	return nil
}

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

		return messages[len(messages)-limit:], nil
	}

	return messages, nil
}

func (s *InMemorySessionService) GetMessagesWithOptions(sessionID string, opts reasoning.LoadOptions) ([]*pb.Message, error) {

	return s.GetMessages(sessionID, opts.Limit)
}

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

func (s *InMemorySessionService) GetOrCreateSessionMetadata(sessionID string) (*reasoning.SessionMetadata, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("sessionID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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

	return session.Metadata, nil
}

func (s *InMemorySessionService) DeleteSession(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("sessionID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

func (s *InMemorySessionService) SessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.sessions)
}
