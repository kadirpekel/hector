package agent

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/client"
	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/auth"
	"github.com/kadirpekel/hector/pkg/config"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ExternalA2AAgent is a proxy to an external A2A-compliant agent.
// It auto-discovers the agent's capabilities and uses the appropriate transport.
type ExternalA2AAgent struct {
	pb.UnimplementedA2AServiceServer

	agentID       string // Local agent ID (config key)
	targetAgentID string // Remote agent ID (for routing to external service)
	name          string
	description   string
	url           string
	client        client.A2AClient
	config        *config.AgentConfig
}

// StreamMessage implements StreamingAgentClient interface for true streaming support
func (e *ExternalA2AAgent) StreamMessage(ctx context.Context, agentID string, message *pb.Message) (<-chan *pb.StreamResponse, error) {
	// Use targetAgentID (remote agent ID) instead of local agentID
	return e.client.StreamMessage(ctx, e.targetAgentID, message)
}

// NewExternalA2AAgent creates a proxy to an external A2A agent.
// The agent is discovered via its agent card, and the appropriate transport is chosen automatically.
func NewExternalA2AAgent(agentID string, agentConfig *config.AgentConfig) (*ExternalA2AAgent, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}
	if agentConfig == nil {
		return nil, fmt.Errorf("agent config cannot be nil")
	}

	if agentConfig.Type != "a2a" {
		return nil, fmt.Errorf("agent type must be 'a2a' for external agents, got: %s", agentConfig.Type)
	}

	if agentConfig.URL == "" {
		return nil, fmt.Errorf("URL is required for external A2A agents")
	}

	// Extract token from credentials
	var token string
	if agentConfig.Credentials != nil {
		tokenProvider, err := auth.NewTokenProviderFromCredentials(
			agentConfig.Credentials.Type,
			agentConfig.Credentials.Token,
			agentConfig.Credentials.APIKey,
			agentConfig.Credentials.Username,
			agentConfig.Credentials.Password,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create token provider: %w", err)
		}
		// Get token for initial connection
		token, err = tokenProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to get authentication token: %w", err)
		}
	}

	// Determine target agent ID (remote agent ID to use when calling the service)
	// Use target_agent_id from config if provided, otherwise use the local agent ID (config key)
	targetAgentID := agentConfig.TargetAgentID
	if targetAgentID == "" {
		targetAgentID = agentID
	}

	// Create universal A2A client (auto-discovers agent and chooses transport)
	// Use targetAgentID for discovery and calls
	a2aClient, err := client.NewUniversalA2AClient(agentConfig.URL, targetAgentID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create A2A client for %s: %w", agentConfig.URL, err)
	}

	return &ExternalA2AAgent{
		agentID:       agentID,
		targetAgentID: targetAgentID,
		name:          agentConfig.Name,
		description:   agentConfig.Description,
		url:           agentConfig.URL,
		client:        a2aClient,
		config:        agentConfig,
	}, nil
}

// Close closes the connection to the external agent
func (e *ExternalA2AAgent) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

// GetAgentCard returns the agent card (from cache or remote)
func (e *ExternalA2AAgent) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error) {
	// Use targetAgentID to get the cached card from discovery (client was initialized with targetAgentID)
	return e.client.GetAgentCard(ctx, e.targetAgentID)
}

// SendMessage sends a message to the external agent
func (e *ExternalA2AAgent) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	if req.Request == nil {
		return nil, fmt.Errorf("request message cannot be nil")
	}
	// Use targetAgentID (remote agent ID) instead of local agentID
	return e.client.SendMessage(ctx, e.targetAgentID, req.Request)
}

// SendStreamingMessage sends a streaming message to the external agent
func (e *ExternalA2AAgent) SendStreamingMessage(req *pb.SendMessageRequest, stream pb.A2AService_SendStreamingMessageServer) error {
	if req.Request == nil {
		return fmt.Errorf("request message cannot be nil")
	}

	// Use targetAgentID (remote agent ID) instead of local agentID
	streamChan, err := e.client.StreamMessage(stream.Context(), e.targetAgentID, req.Request)
	if err != nil {
		return err
	}

	for response := range streamChan {
		if err := stream.Send(response); err != nil {
			return err
		}
	}

	return nil
}

// GetTask gets task status from the external agent
func (e *ExternalA2AAgent) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	// Use extractTaskID from agent_a2a_methods.go
	taskID := extractTaskID(req.GetName())
	// Use targetAgentID (remote agent ID) instead of local agentID
	return e.client.GetTask(ctx, e.targetAgentID, taskID)
}

// CancelTask cancels a task on the external agent
func (e *ExternalA2AAgent) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error) {
	// Use extractTaskID from agent_a2a_methods.go
	taskID := extractTaskID(req.GetName())
	// Use targetAgentID (remote agent ID) instead of local agentID
	return e.client.CancelTask(ctx, e.targetAgentID, taskID)
}

// TaskSubscription subscribes to task updates (not yet implemented for external agents)
func (e *ExternalA2AAgent) TaskSubscription(req *pb.TaskSubscriptionRequest, stream pb.A2AService_TaskSubscriptionServer) error {
	return fmt.Errorf("task subscription not yet implemented for external agents")
}

// Push notification methods (not implemented for external agents)
func (e *ExternalA2AAgent) CreateTaskPushNotificationConfig(ctx context.Context, req *pb.CreateTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return nil, fmt.Errorf("push notifications not supported for external agents")
}

func (e *ExternalA2AAgent) GetTaskPushNotificationConfig(ctx context.Context, req *pb.GetTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return nil, fmt.Errorf("push notifications not supported for external agents")
}

func (e *ExternalA2AAgent) ListTaskPushNotificationConfig(ctx context.Context, req *pb.ListTaskPushNotificationConfigRequest) (*pb.ListTaskPushNotificationConfigResponse, error) {
	return nil, fmt.Errorf("push notifications not supported for external agents")
}

func (e *ExternalA2AAgent) DeleteTaskPushNotificationConfig(ctx context.Context, req *pb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	return nil, fmt.Errorf("push notifications not supported for external agents")
}

// GetName returns the agent's display name
func (e *ExternalA2AAgent) GetName() string {
	return e.name
}

// GetDescription returns the agent's description
func (e *ExternalA2AAgent) GetDescription() string {
	return e.description
}

// GetConfig returns the agent's configuration
func (e *ExternalA2AAgent) GetConfig() *config.AgentConfig {
	return e.config
}
