package agent

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/auth"
	"github.com/kadirpekel/hector/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ExternalA2AAgent is a client wrapper for external A2A agents
// It implements pb.A2AServiceServer by forwarding calls to a remote agent
type ExternalA2AAgent struct {
	pb.UnimplementedA2AServiceServer

	name        string
	description string
	url         string
	client      pb.A2AServiceClient
	conn        *grpc.ClientConn
	config      *config.AgentConfig
}

// NewExternalA2AAgent creates a client for an external A2A agent
func NewExternalA2AAgent(agentConfig *config.AgentConfig) (*ExternalA2AAgent, error) {
	if agentConfig == nil {
		return nil, fmt.Errorf("agent config cannot be nil")
	}

	if agentConfig.Type != "a2a" {
		return nil, fmt.Errorf("agent type must be 'a2a' for external agents, got: %s", agentConfig.Type)
	}

	if agentConfig.URL == "" {
		return nil, fmt.Errorf("URL is required for external A2A agents")
	}

	// Prepare dial options
	var dialOpts []grpc.DialOption

	// Default to insecure for now (TODO: add TLS support based on URL scheme)
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// Add authentication if credentials are provided
	var conn *grpc.ClientConn
	var err error

	if agentConfig.Credentials != nil {
		// Create token provider from credentials
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

		// Create authenticated connection
		conn, err = auth.NewAuthenticatedClientConn(agentConfig.URL, tokenProvider, dialOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to external agent at %s: %w", agentConfig.URL, err)
		}
	} else {
		// Create connection without authentication
		conn, err = grpc.NewClient(agentConfig.URL, dialOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to external agent at %s: %w", agentConfig.URL, err)
		}
	}

	// Create A2A client
	client := pb.NewA2AServiceClient(conn)

	return &ExternalA2AAgent{
		name:        agentConfig.Name,
		description: agentConfig.Description,
		url:         agentConfig.URL,
		client:      client,
		conn:        conn,
		config:      agentConfig,
	}, nil
}

// Close closes the connection to the external agent
func (e *ExternalA2AAgent) Close() error {
	if e.conn != nil {
		return e.conn.Close()
	}
	return nil
}

// ============================================================================
// A2A PROTOCOL METHODS - Forward to remote agent
// ============================================================================

// GetAgentCard forwards the request to the external agent
func (e *ExternalA2AAgent) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error) {
	return e.client.GetAgentCard(ctx, req)
}

// SendMessage forwards the request to the external agent
func (e *ExternalA2AAgent) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	return e.client.SendMessage(ctx, req)
}

// SendStreamingMessage forwards the streaming request to the external agent
func (e *ExternalA2AAgent) SendStreamingMessage(req *pb.SendMessageRequest, stream pb.A2AService_SendStreamingMessageServer) error {
	// Create client stream
	clientStream, err := e.client.SendStreamingMessage(stream.Context(), req)
	if err != nil {
		return err
	}

	// Forward all responses from external agent to our client
	for {
		resp, err := clientStream.Recv()
		if err != nil {
			return err
		}

		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

// GetTask forwards the request to the external agent
func (e *ExternalA2AAgent) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	return e.client.GetTask(ctx, req)
}

// CancelTask forwards the request to the external agent
func (e *ExternalA2AAgent) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error) {
	return e.client.CancelTask(ctx, req)
}

// TaskSubscription forwards the streaming request to the external agent
func (e *ExternalA2AAgent) TaskSubscription(req *pb.TaskSubscriptionRequest, stream pb.A2AService_TaskSubscriptionServer) error {
	// Create client stream
	clientStream, err := e.client.TaskSubscription(stream.Context(), req)
	if err != nil {
		return err
	}

	// Forward all responses from external agent to our client
	for {
		resp, err := clientStream.Recv()
		if err != nil {
			return err
		}

		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

// CreateTaskPushNotificationConfig forwards the request to the external agent
func (e *ExternalA2AAgent) CreateTaskPushNotificationConfig(ctx context.Context, req *pb.CreateTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return e.client.CreateTaskPushNotificationConfig(ctx, req)
}

// GetTaskPushNotificationConfig forwards the request to the external agent
func (e *ExternalA2AAgent) GetTaskPushNotificationConfig(ctx context.Context, req *pb.GetTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return e.client.GetTaskPushNotificationConfig(ctx, req)
}

// ListTaskPushNotificationConfig forwards the request to the external agent
func (e *ExternalA2AAgent) ListTaskPushNotificationConfig(ctx context.Context, req *pb.ListTaskPushNotificationConfigRequest) (*pb.ListTaskPushNotificationConfigResponse, error) {
	return e.client.ListTaskPushNotificationConfig(ctx, req)
}

// DeleteTaskPushNotificationConfig forwards the request to the external agent
func (e *ExternalA2AAgent) DeleteTaskPushNotificationConfig(ctx context.Context, req *pb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	return e.client.DeleteTaskPushNotificationConfig(ctx, req)
}

// GetName returns the agent name
func (e *ExternalA2AAgent) GetName() string {
	return e.name
}

// GetDescription returns the agent description
func (e *ExternalA2AAgent) GetDescription() string {
	return e.description
}

// GetConfig returns the agent configuration
func (e *ExternalA2AAgent) GetConfig() *config.AgentConfig {
	return e.config
}
