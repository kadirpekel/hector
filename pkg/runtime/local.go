package runtime

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc/metadata"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
)

func (r *Runtime) SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {

	agentEntry, ok := r.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	req := &pb.SendMessageRequest{
		Request: message,
	}

	contextID := message.ContextId
	if contextID == "" {
		contextID = "default"
	}
	ctx = context.WithValue(ctx, agent.SessionIDKey, contextID)

	return agentEntry.Agent.SendMessage(ctx, req)
}

func (r *Runtime) StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {

	agentEntry, ok := r.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	req := &pb.SendMessageRequest{
		Request: message,
	}

	contextID := message.ContextId
	if contextID == "" {
		contextID = "default"
	}
	ctx = context.WithValue(ctx, agent.SessionIDKey, contextID)

	streamChan := make(chan *pb.StreamResponse, 10)

	stream := &localStream{
		ctx:  ctx,
		send: streamChan,
	}

	go func() {
		defer close(streamChan)
		if err := agentEntry.Agent.SendStreamingMessage(req, stream); err != nil {
			log.Printf("Warning: streaming error for agent '%s': %v", agentID, err)
		}
	}()

	return streamChan, nil
}

func (r *Runtime) ListAgents(ctx context.Context) ([]*pb.AgentCard, error) {
	entries := r.registry.List()
	agents := make([]*pb.AgentCard, 0, len(entries))

	for _, entry := range entries {

		card, err := entry.Agent.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
		if err != nil {

			card = &pb.AgentCard{
				Name:        entry.Name,
				Description: entry.Config.Description,
				Version:     "1.0.0",
			}
		}

		if card.Url == "" {
			card.Url = "local://" + entry.Name
		}

		agents = append(agents, card)
	}

	return agents, nil
}

func (r *Runtime) GetAgentCard(ctx context.Context, agentID string) (*pb.AgentCard, error) {

	agentEntry, ok := r.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	return agentEntry.Agent.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
}

func (r *Runtime) GetTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {

	agentEntry, ok := r.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	req := &pb.GetTaskRequest{
		Name: fmt.Sprintf("tasks/%s", taskID),
	}

	return agentEntry.Agent.GetTask(ctx, req)
}

func (r *Runtime) CancelTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {

	agentEntry, ok := r.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	req := &pb.CancelTaskRequest{
		Name: fmt.Sprintf("tasks/%s", taskID),
	}

	return agentEntry.Agent.CancelTask(ctx, req)
}

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
