package memory

import (
	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

type LongTermMemoryStrategy interface {
	Store(agentID string, sessionID string, messages []*pb.Message) error

	Recall(agentID string, sessionID string, query string, limit int) ([]*pb.Message, error)

	Clear(agentID string, sessionID string) error

	Name() string
}
