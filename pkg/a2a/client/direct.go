package client

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc/metadata"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const sessionIDKey contextKey = "sessionID"

// LocalClient implements A2AClient for in-process agent execution
type LocalClient struct {
	config     *config.Config
	components *component.ComponentManager
	registry   *agent.AgentRegistry
}

// NewLocalClient creates a new local (in-process) A2A client
func NewLocalClient(cfg *config.Config) (A2AClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Create component manager
	componentManager, err := component.NewComponentManager(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	// Create agent registry
	agentRegistry := agent.NewAgentRegistry()

	// Register all configured agents
	for agentID, agentCfg := range cfg.Agents {
		cfg := agentCfg

		// Only support native agents in local mode (not external A2A agents)
		if cfg.Type == "a2a" {
			continue
		}

		// Create agent
		agentInstance, err := agent.NewAgent(agentID, &cfg, componentManager, agentRegistry)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent '%s': %w", agentID, err)
		}

		// Register agent
		if err := agentRegistry.RegisterAgent(agentID, agentInstance, &cfg, nil); err != nil {
			return nil, fmt.Errorf("failed to register agent '%s': %w", agentID, err)
		}
	}

	return &LocalClient{
		config:     cfg,
		components: componentManager,
		registry:   agentRegistry,
	}, nil
}

// SendMessage sends a non-streaming message to an agent
func (c *LocalClient) SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
	// Get agent from registry
	agentEntry, ok := c.registry.Get(agentID)
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
	ctx = context.WithValue(ctx, sessionIDKey, contextID)

	// Call agent directly
	return agentEntry.Agent.SendMessage(ctx, req)
}

// StreamMessage sends a streaming message to an agent
func (c *LocalClient) StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {
	// Get agent from registry
	agentEntry, ok := c.registry.Get(agentID)
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
	ctx = context.WithValue(ctx, sessionIDKey, contextID)

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
func (c *LocalClient) ListAgents(ctx context.Context) ([]AgentInfo, error) {
	entries := c.registry.List()
	agents := make([]AgentInfo, 0, len(entries))

	for _, entry := range entries {
		// Get agent card for more info
		card, err := entry.Agent.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
		name := entry.Name
		description := entry.Config.Description
		if err == nil && card != nil {
			name = card.Name
			description = card.Description
		}

		agents = append(agents, AgentInfo{
			ID:          entry.Name,
			Name:        name,
			Description: description,
			Endpoint:    "local://" + entry.Name,
		})
	}

	return agents, nil
}

// GetAgentCard retrieves the agent card for a specific agent
func (c *LocalClient) GetAgentCard(ctx context.Context, agentID string) (*pb.AgentCard, error) {
	// Get agent from registry
	agentEntry, ok := c.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	return agentEntry.Agent.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
}

// GetTask retrieves a task by ID
func (c *LocalClient) GetTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {
	// Get agent from registry
	agentEntry, ok := c.registry.Get(agentID)
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
func (c *LocalClient) CancelTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {
	// Get agent from registry
	agentEntry, ok := c.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	// Call CancelTask on the agent
	req := &pb.CancelTaskRequest{
		Name: fmt.Sprintf("tasks/%s", taskID),
	}

	return agentEntry.Agent.CancelTask(ctx, req)
}

// Close releases resources
func (c *LocalClient) Close() error {
	// No cleanup needed for local client
	return nil
}

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
