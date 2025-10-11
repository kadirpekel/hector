package memory

import (
	"github.com/kadirpekel/hector/pkg/a2a"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
)

// StatusNotifier is a callback for sending status updates during operations
// like summarization (e.g., to inform users via streaming)
type StatusNotifier func(message string)

// WorkingMemoryStrategy defines pluggable short-term conversation memory management
// Strategies operate on session data without managing session lifecycle
type WorkingMemoryStrategy interface {
	// Name returns the strategy identifier
	Name() string

	// AddMessage adds a message to the session's memory
	// May trigger operations like summarization depending on strategy
	AddMessage(session *hectorcontext.ConversationHistory, msg a2a.Message) error

	// GetMessages returns messages from the session within the strategy's constraints
	GetMessages(session *hectorcontext.ConversationHistory) ([]a2a.Message, error)

	// SetStatusNotifier sets a callback for status notifications
	// Used to inform users about background operations like summarization
	SetStatusNotifier(notifier StatusNotifier)
}
