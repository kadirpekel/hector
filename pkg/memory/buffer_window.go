package memory

import (
	"fmt"
	"log"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

type BufferWindowStrategy struct {
	windowSize int
}

type BufferWindowConfig struct {
	WindowSize int
}

func NewBufferWindowStrategy(config BufferWindowConfig) (*BufferWindowStrategy, error) {
	if config.WindowSize <= 0 {
		config.WindowSize = 20
	}

	return &BufferWindowStrategy{
		windowSize: config.WindowSize,
	}, nil
}

func (s *BufferWindowStrategy) Name() string {
	return "buffer_window"
}

func (s *BufferWindowStrategy) SetStatusNotifier(notifier StatusNotifier) {
}

func (s *BufferWindowStrategy) AddMessage(session *hectorcontext.ConversationHistory, msg *pb.Message) error {
	return session.AddMessage(msg)
}

func (s *BufferWindowStrategy) CheckAndSummarize(session *hectorcontext.ConversationHistory) ([]*pb.Message, error) {

	return nil, nil
}

func (s *BufferWindowStrategy) GetMessages(session *hectorcontext.ConversationHistory) ([]*pb.Message, error) {
	messages := session.GetRecentMessages(s.windowSize)
	return messages, nil
}

func (s *BufferWindowStrategy) LoadState(sessionID string, sessionService interface{}) (*hectorcontext.ConversationHistory, error) {

	sessService, ok := sessionService.(interface {
		GetMessagesWithOptions(sessionID string, opts reasoning.LoadOptions) ([]*pb.Message, error)
	})
	if !ok {
		return nil, fmt.Errorf("session service does not support GetMessagesWithOptions")
	}

	messages, err := sessService.GetMessagesWithOptions(sessionID, reasoning.LoadOptions{
		Limit: s.windowSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}

	session, err := hectorcontext.NewConversationHistory(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	for _, msg := range messages {
		if err := session.AddMessage(msg); err != nil {
			log.Printf("⚠️  Failed to add message to session: %v", err)
		}
	}

	log.Printf("✅ Loaded %d messages (window size: %d) for session %s", len(messages), s.windowSize, sessionID)
	return session, nil
}
