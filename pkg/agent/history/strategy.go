package history

import (
	"github.com/kadirpekel/hector/pkg/llms"
)

// StatusNotifier is a callback for sending status updates during operations
// like summarization (e.g., to inform users via streaming)
type StatusNotifier func(message string)

// HistoryStrategy defines pluggable conversation history management
type HistoryStrategy interface {
	// AddMessage adds a message to the session's history
	// May trigger summarization depending on strategy
	AddMessage(sessionID string, msg llms.Message) error

	// GetHistory returns messages for the session within the strategy's constraints
	GetHistory(sessionID string) ([]llms.Message, error)

	// Clear removes all history for a session
	Clear(sessionID string) error

	// GetSessionCount returns number of active sessions
	GetSessionCount() int

	// Name returns the strategy identifier
	Name() string

	// SetStatusNotifier sets a callback for status notifications
	// Used to inform users about background operations like summarization
	SetStatusNotifier(notifier StatusNotifier)
}
