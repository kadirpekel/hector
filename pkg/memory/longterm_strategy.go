package memory

import (
	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// LongTermMemoryStrategy defines pluggable long-term memory for semantic storage and recall
// This interface is pure - no knowledge of working memory or summarization
// Uses A2A protocol Message types for true A2A-native architecture
// IMPORTANT: Implementations must isolate by BOTH agent_id AND session_id
type LongTermMemoryStrategy interface {
	// Store adds messages to long-term memory
	// agentID: Agent identifier (for multi-agent isolation)
	// sessionID: Current session (isolation)
	// messages: Messages to store (batch)
	Store(agentID string, sessionID string, messages []*pb.Message) error

	// Recall retrieves relevant context from long-term memory
	// agentID: Agent identifier (filter - prevents cross-agent leaks)
	// sessionID: Current session (filter)
	// query: Semantic query for retrieval
	// limit: Max results to return
	Recall(agentID string, sessionID string, query string, limit int) ([]*pb.Message, error)

	// Clear removes all long-term memory for a session
	// agentID: Agent identifier (filter)
	// sessionID: Current session (filter)
	Clear(agentID string, sessionID string) error

	// Name returns the strategy identifier
	Name() string
}
