package client

import (
	"context"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

type A2AClient interface {
	SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error)

	StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error)

	ListAgents(ctx context.Context) ([]*pb.AgentCard, error)

	GetAgentCard(ctx context.Context, agentID string) (*pb.AgentCard, error)

	GetTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error)

	CancelTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error)

	Close() error
}
