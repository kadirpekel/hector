package memory

import (
	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// LongTermMemoryStrategy defines pluggable long-term memory for semantic storage and recall
// This interface is pure - no knowledge of working memory or summarization
// Uses A2A protocol Message types for true A2A-native architecture
type LongTermMemoryStrategy interface {
	// Store adds messages to long-term memory
	// sessionID: Current session (isolation)
	// messages: Messages to store (batch)
	Store(sessionID string, messages []*pb.Message) error

	// Recall retrieves relevant context from long-term memory
	// sessionID: Current session (filter)
	// query: Semantic query for retrieval
	// limit: Max results to return
	Recall(sessionID string, query string, limit int) ([]*pb.Message, error)

	// Clear removes all long-term memory for a session
	Clear(sessionID string) error

	// Name returns the strategy identifier
	Name() string
}
