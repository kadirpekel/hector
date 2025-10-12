package memory

import (
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
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

func (s *BufferWindowStrategy) GetMessages(session *hectorcontext.ConversationHistory) ([]*pb.Message, error) {
	messages := session.GetRecentMessages(s.windowSize)
	return messages, nil
}
