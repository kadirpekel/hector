// Package runtime - A2AClient interface implementation for local mode
package runtime

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc/metadata"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
)

// ============================================================================
// A2AClient Interface Implementation
// ============================================================================
//
// Runtime implements the A2AClient interface directly, eliminating the need
// for a LocalClient wrapper. This allows Runtime to be used polymorphically
// with HTTPClient in CLI commands.
//
// ============================================================================

// SendMessage sends a non-streaming message to an agent
func (r *Runtime) SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
	// Get agent from registry
	agentEntry, ok := r.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	// Create request
	req := &pb.SendMessageRequest{
		Request: message,
	}

	// IMPORTANT: Add sessionID to context so agent can load conversation history
	// Use context_id from message as the session ID (same as A2A server does)
	contextID := message.ContextId
	if contextID == "" {
		contextID = "default" // Fallback to default if not provided
	}
	ctx = context.WithValue(ctx, agent.SessionIDKey, contextID)

	// Call agent directly
	return agentEntry.Agent.SendMessage(ctx, req)
}

// StreamMessage sends a streaming message to an agent
func (r *Runtime) StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {
	// Get agent from registry
	agentEntry, ok := r.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	// Create request
	req := &pb.SendMessageRequest{
		Request: message,
	}

	// IMPORTANT: Add sessionID to context so agent can load conversation history
	// Use context_id from message as the session ID (same as A2A server does)
	contextID := message.ContextId
	if contextID == "" {
		contextID = "default" // Fallback to default if not provided
	}
	ctx = context.WithValue(ctx, agent.SessionIDKey, contextID)

	// Create channel for streaming responses
	streamChan := make(chan *pb.StreamResponse, 10)

	// Create a mock stream that writes to our channel
	stream := &localStream{
		ctx:  ctx,
		send: streamChan,
	}

	// Start streaming in goroutine
	go func() {
		defer close(streamChan)
		if err := agentEntry.Agent.SendStreamingMessage(req, stream); err != nil {
			log.Printf("Warning: streaming error for agent '%s': %v", agentID, err)
		}
	}()

	return streamChan, nil
}

// ListAgents returns a list of all registered agents
func (r *Runtime) ListAgents(ctx context.Context) ([]*pb.AgentCard, error) {
	entries := r.registry.List()
	agents := make([]*pb.AgentCard, 0, len(entries))

	for _, entry := range entries {
		// Get agent card
		card, err := entry.Agent.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
		if err != nil {
			// Fallback: create basic card from config
			card = &pb.AgentCard{
				Name:        entry.Name,
				Description: entry.Config.Description,
				Version:     "1.0.0",
			}
		}

		// Set URL to local endpoint if not already set
		if card.Url == "" {
			card.Url = "local://" + entry.Name
		}

		agents = append(agents, card)
	}

	return agents, nil
}

// GetAgentCard retrieves the agent card for a specific agent
func (r *Runtime) GetAgentCard(ctx context.Context, agentID string) (*pb.AgentCard, error) {
	// Get agent from registry
	agentEntry, ok := r.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	return agentEntry.Agent.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
}

// GetTask retrieves a task by ID
func (r *Runtime) GetTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {
	// Get agent from registry
	agentEntry, ok := r.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	// Call GetTask on the agent
	req := &pb.GetTaskRequest{
		Name: fmt.Sprintf("tasks/%s", taskID),
	}

	return agentEntry.Agent.GetTask(ctx, req)
}

// CancelTask cancels a running task
func (r *Runtime) CancelTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {
	// Get agent from registry
	agentEntry, ok := r.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	// Call CancelTask on the agent
	req := &pb.CancelTaskRequest{
		Name: fmt.Sprintf("tasks/%s", taskID),
	}

	return agentEntry.Agent.CancelTask(ctx, req)
}

// ============================================================================
// localStream - gRPC Stream Adapter
// ============================================================================

// localStream implements pb.A2AService_SendStreamingMessageServer for local mode
type localStream struct {
	ctx  context.Context
	send chan<- *pb.StreamResponse
}

func (s *localStream) Send(resp *pb.StreamResponse) error {
	select {
	case s.send <- resp:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *localStream) Context() context.Context {
	return s.ctx
}

func (s *localStream) SendMsg(m interface{}) error {
	return nil
}

func (s *localStream) RecvMsg(m interface{}) error {
	return nil
}

func (s *localStream) SendHeader(_ metadata.MD) error {
	return nil
}

func (s *localStream) SetHeader(_ metadata.MD) error {
	return nil
}

func (s *localStream) SetTrailer(_ metadata.MD) {
}
