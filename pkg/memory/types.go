package memory

import (
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
)

// ============================================================================
// A2A-NATIVE MEMORY INTERFACES
// All interfaces work directly with pb.Message (no conversion)
// ============================================================================

// StatusNotifier is a callback for status updates during summarization
type StatusNotifier func(message string)

// WorkingMemoryStrategy defines how recent conversation context is managed
// Uses NATIVE pb.Message storage without any conversion
type WorkingMemoryStrategy interface {
	// AddMessage stores a pb.Message directly (no conversion)
	AddMessage(session *hectorcontext.ConversationHistory, message *pb.Message) error

	// GetMessages retrieves pb.Message directly (no conversion)
	GetMessages(session *hectorcontext.ConversationHistory) ([]*pb.Message, error)

	// SetStatusNotifier sets callback for status updates during operations
	SetStatusNotifier(notifier StatusNotifier)

	// Name returns the strategy identifier
	Name() string
}

// LongTermConfig configures long-term memory behavior
type LongTermConfig struct {
	Enabled      bool         `yaml:"enabled"`       // Enable long-term memory (default: false)
	StorageScope StorageScope `yaml:"storage_scope"` // What messages to store (default: "all")
	BatchSize    int          `yaml:"batch_size"`    // Batch size for storage (default: 1 = immediate)
	AutoRecall   bool         `yaml:"auto_recall"`   // Auto-inject memories before LLM calls (default: true)
	RecallLimit  int          `yaml:"recall_limit"`  // Max memories to recall (default: 5)
	Collection   string       `yaml:"collection"`    // Qdrant collection name (default: "hector_session_memory")
}

// StorageScope defines what messages to store in long-term memory
type StorageScope string

const (
	// StorageScopeAll stores all messages (default)
	StorageScopeAll StorageScope = "all"

	// StorageScopeConversational stores only user and assistant messages
	StorageScopeConversational StorageScope = "conversational"

	// StorageScopeSummariesOnly stores only summary messages
	StorageScopeSummariesOnly StorageScope = "summaries_only"
)

// SetDefaults applies default values to LongTermConfig
func (c *LongTermConfig) SetDefaults() {
	if c.BatchSize <= 0 {
		c.BatchSize = 1 // Default: immediate storage
	}
	if c.StorageScope == "" {
		c.StorageScope = StorageScopeAll
	}
	if c.RecallLimit <= 0 {
		c.RecallLimit = 5
	}
	if c.Collection == "" {
		c.Collection = "hector_session_memory"
	}
}
