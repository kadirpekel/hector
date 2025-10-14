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

// DirectClient implements A2AClient for in-process agent execution
type DirectClient struct {
	config     *config.Config
	components *component.ComponentManager
	registry   *agent.AgentRegistry
}

// NewDirectClient creates a new direct (in-process) A2A client
func NewDirectClient(cfg *config.Config) (A2AClient, error) {
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

		// Only support native agents in direct mode (not external A2A agents)
		if cfg.Type == "a2a" {
			continue
		}

		// Create agent
		agentInstance, err := agent.NewAgent(&cfg, componentManager, agentRegistry)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent '%s': %w", agentID, err)
		}

		// Register agent
		if err := agentRegistry.RegisterAgent(agentID, agentInstance, &cfg, nil); err != nil {
			return nil, fmt.Errorf("failed to register agent '%s': %w", agentID, err)
		}
	}

	return &DirectClient{
		config:     cfg,
		components: componentManager,
		registry:   agentRegistry,
	}, nil
}

// SendMessage sends a non-streaming message to an agent
func (c *DirectClient) SendMessage(ctx context.Context, agentID string, message *pb.Message) (*pb.SendMessageResponse, error) {
	// Get agent from registry
	agentEntry, ok := c.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	// Create request
	req := &pb.SendMessageRequest{
		Request: message,
	}

	// Call agent directly
	return agentEntry.Agent.SendMessage(ctx, req)
}

// StreamMessage sends a streaming message to an agent
func (c *DirectClient) StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {
	// Get agent from registry
	agentEntry, ok := c.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	// Create request
	req := &pb.SendMessageRequest{
		Request: message,
	}

	// Create channel for streaming responses
	streamChan := make(chan *pb.StreamResponse, 10)

	// Create a mock stream that writes to our channel
	stream := &directStream{
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
func (c *DirectClient) ListAgents(ctx context.Context) ([]AgentInfo, error) {
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
			Endpoint:    "direct://" + entry.Name,
		})
	}

	return agents, nil
}

// GetAgentCard retrieves the agent card for a specific agent
func (c *DirectClient) GetAgentCard(ctx context.Context, agentID string) (*pb.AgentCard, error) {
	// Get agent from registry
	agentEntry, ok := c.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}

	return agentEntry.Agent.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
}

// GetTask retrieves a task by ID
func (c *DirectClient) GetTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {
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
func (c *DirectClient) CancelTask(ctx context.Context, agentID string, taskID string) (*pb.Task, error) {
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
func (c *DirectClient) Close() error {
	// No cleanup needed for direct client
	return nil
}

// directStream implements pb.A2AService_SendStreamingMessageServer for direct mode
type directStream struct {
	ctx  context.Context
	send chan<- *pb.StreamResponse
}

func (s *directStream) Send(resp *pb.StreamResponse) error {
	select {
	case s.send <- resp:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *directStream) Context() context.Context {
	return s.ctx
}

func (s *directStream) SendMsg(m interface{}) error {
	return nil
}

func (s *directStream) RecvMsg(m interface{}) error {
	return nil
}

func (s *directStream) SendHeader(_ metadata.MD) error {
	return nil
}

func (s *directStream) SetHeader(_ metadata.MD) error {
	return nil
}

func (s *directStream) SetTrailer(_ metadata.MD) {
}
