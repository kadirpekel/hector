package memory

import (
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
)

type StatusNotifier func(message string)

type WorkingMemoryStrategy interface {
	AddMessage(session *hectorcontext.ConversationHistory, message *pb.Message) error

	CheckAndSummarize(session *hectorcontext.ConversationHistory) ([]*pb.Message, error)

	GetMessages(session *hectorcontext.ConversationHistory) ([]*pb.Message, error)

	SetStatusNotifier(notifier StatusNotifier)

	Name() string

	LoadState(sessionID string, sessionService interface{}) (*hectorcontext.ConversationHistory, error)
}

type LongTermConfig struct {
	Enabled          bool         `yaml:"enabled"`
	StorageScope     StorageScope `yaml:"storage_scope"`
	BatchSize        int          `yaml:"batch_size"`
	EnableAutoRecall bool         `yaml:"enable_auto_recall"`
	RecallLimit      int          `yaml:"recall_limit"`
	Collection       string       `yaml:"collection"`
}

type StorageScope string

const (
	StorageScopeAll StorageScope = "all"

	StorageScopeConversational StorageScope = "conversational"

	StorageScopeSummariesOnly StorageScope = "summaries_only"
)

func (c *LongTermConfig) SetDefaults() {
	if c.BatchSize <= 0 {
		c.BatchSize = 1
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
