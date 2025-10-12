package memory

import (
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// MockSessionService is a minimal mock for testing SessionService interface
type MockSessionService struct {
	messages map[string][]*pb.Message
}

// NewMockSessionService creates a new mock session service
func NewMockSessionService() *MockSessionService {
	return &MockSessionService{
		messages: make(map[string][]*pb.Message),
	}
}

// AppendMessage appends a message to a session
func (m *MockSessionService) AppendMessage(sessionID string, message *pb.Message) error {
	if m.messages[sessionID] == nil {
		m.messages[sessionID] = make([]*pb.Message, 0)
	}
	m.messages[sessionID] = append(m.messages[sessionID], message)
	return nil
}

// GetMessages returns the most recent messages from a session
func (m *MockSessionService) GetMessages(sessionID string, limit int) ([]*pb.Message, error) {
	msgs, exists := m.messages[sessionID]
	if !exists {
		return []*pb.Message{}, nil
	}
	if limit > 0 && len(msgs) > limit {
		return msgs[len(msgs)-limit:], nil
	}
	return msgs, nil
}

// GetMessageCount returns the number of messages in a session
func (m *MockSessionService) GetMessageCount(sessionID string) (int, error) {
	msgs, exists := m.messages[sessionID]
	if !exists {
		return 0, nil
	}
	return len(msgs), nil
}

// GetOrCreateSessionMetadata returns or creates session metadata
func (m *MockSessionService) GetOrCreateSessionMetadata(sessionID string) (*reasoning.SessionMetadata, error) {
	return &reasoning.SessionMetadata{
		ID:        sessionID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}, nil
}

// DeleteSession deletes a session
func (m *MockSessionService) DeleteSession(sessionID string) error {
	delete(m.messages, sessionID)
	return nil
}

// SessionCount returns the number of active sessions
func (m *MockSessionService) SessionCount() int {
	return len(m.messages)
}
