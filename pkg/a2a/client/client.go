// Package client provides A2A protocol client implementations
package client

import (
	"context"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

// A2AClient is a generic interface for interacting with A2A agents
// It abstracts whether the agent is remote (HTTP) or local (direct)
type A2AClient interface {
	// SendMessage sends a message to an agent and returns the response
	SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error)

	// StreamMessage sends a message and returns a streaming channel
	StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error)

	// ListAgents returns a list of available agents
	ListAgents(ctx context.Context) ([]AgentInfo, error)

	// GetAgentCard retrieves information about a specific agent
	GetAgentCard(ctx context.Context, agentID string) (*pb.AgentCard, error)

	// GetTask retrieves a task by ID
	GetTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error)

	// CancelTask cancels a running task
	CancelTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error)

	// Close releases any resources held by the client
	Close() error
}

// AgentInfo holds basic information about an agent
type AgentInfo struct {
	ID          string
	Name        string
	Description string
	Endpoint    string
}
