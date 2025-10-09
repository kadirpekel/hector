package memory

import (
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/llms"
)

// BufferWindowStrategy implements simple LIFO memory management
// Keeps last N messages, drops oldest when window is full
type BufferWindowStrategy struct {
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
		windowSize: config.WindowSize,
	}, nil
}

// Name returns the strategy identifier
func (s *BufferWindowStrategy) Name() string {
	return "buffer_window"
}

// SetStatusNotifier sets a callback for status notifications (no-op for buffer window)
func (s *BufferWindowStrategy) SetStatusNotifier(notifier StatusNotifier) {
	// Buffer window strategy doesn't need status notifications
	// This method exists to satisfy the WorkingMemoryStrategy interface
}

// AddMessage adds a message to the session's memory
func (s *BufferWindowStrategy) AddMessage(session *hectorcontext.ConversationHistory, msg llms.Message) error {
	_, err := session.AddMessage(msg.Role, msg.Content, nil)
	return err
}

// GetMessages returns messages from the session within window size
func (s *BufferWindowStrategy) GetMessages(session *hectorcontext.ConversationHistory) ([]llms.Message, error) {
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
