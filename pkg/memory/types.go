package memory

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
	// This is useful with SummaryBufferStrategy to store condensed context
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
	// Note: AutoRecall defaults to true, but we need to handle this carefully
	// because the zero value for bool is false. We'll handle this in the constructor.
}
