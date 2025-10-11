package memory

import (
	"github.com/kadirpekel/hector/pkg/a2a"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
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
func (s *BufferWindowStrategy) AddMessage(session *hectorcontext.ConversationHistory, msg a2a.Message) error {
	textContent := a2a.ExtractTextFromMessage(msg)
	_, err := session.AddMessage(string(msg.Role), textContent, nil)
	return err
}

// GetMessages returns messages from the session within window size
func (s *BufferWindowStrategy) GetMessages(session *hectorcontext.ConversationHistory) ([]a2a.Message, error) {
	// Get recent messages within window size
	contextMessages := session.GetRecentMessages(s.windowSize)

	// Convert to A2A Message format
	messages := make([]a2a.Message, len(contextMessages))
	for i, msg := range contextMessages {
		messages[i] = a2a.CreateTextMessage(a2a.MessageRole(msg.Role), msg.Content)
	}

	return messages, nil
}
