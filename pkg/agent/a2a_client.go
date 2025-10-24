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

type ExternalA2AAgent struct {
	pb.UnimplementedA2AServiceServer

	name        string
	description string
	url         string
	client      pb.A2AServiceClient
	conn        *grpc.ClientConn
	config      *config.AgentConfig
}

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

	var dialOpts []grpc.DialOption

	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	var conn *grpc.ClientConn
	var err error

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

		conn, err = auth.NewAuthenticatedClientConn(agentConfig.URL, tokenProvider, dialOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to external agent at %s: %w", agentConfig.URL, err)
		}
	} else {

		conn, err = grpc.NewClient(agentConfig.URL, dialOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to external agent at %s: %w", agentConfig.URL, err)
		}
	}

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

func (e *ExternalA2AAgent) Close() error {
	if e.conn != nil {
		return e.conn.Close()
	}
	return nil
}

func (e *ExternalA2AAgent) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error) {
	return e.client.GetAgentCard(ctx, req)
}

func (e *ExternalA2AAgent) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	return e.client.SendMessage(ctx, req)
}

func (e *ExternalA2AAgent) SendStreamingMessage(req *pb.SendMessageRequest, stream pb.A2AService_SendStreamingMessageServer) error {

	clientStream, err := e.client.SendStreamingMessage(stream.Context(), req)
	if err != nil {
		return err
	}

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

func (e *ExternalA2AAgent) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	return e.client.GetTask(ctx, req)
}

func (e *ExternalA2AAgent) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error) {
	return e.client.CancelTask(ctx, req)
}

func (e *ExternalA2AAgent) TaskSubscription(req *pb.TaskSubscriptionRequest, stream pb.A2AService_TaskSubscriptionServer) error {

	clientStream, err := e.client.TaskSubscription(stream.Context(), req)
	if err != nil {
		return err
	}

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

func (e *ExternalA2AAgent) CreateTaskPushNotificationConfig(ctx context.Context, req *pb.CreateTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return e.client.CreateTaskPushNotificationConfig(ctx, req)
}

func (e *ExternalA2AAgent) GetTaskPushNotificationConfig(ctx context.Context, req *pb.GetTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return e.client.GetTaskPushNotificationConfig(ctx, req)
}

func (e *ExternalA2AAgent) ListTaskPushNotificationConfig(ctx context.Context, req *pb.ListTaskPushNotificationConfigRequest) (*pb.ListTaskPushNotificationConfigResponse, error) {
	return e.client.ListTaskPushNotificationConfig(ctx, req)
}

func (e *ExternalA2AAgent) DeleteTaskPushNotificationConfig(ctx context.Context, req *pb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	return e.client.DeleteTaskPushNotificationConfig(ctx, req)
}

func (e *ExternalA2AAgent) GetName() string {
	return e.name
}

func (e *ExternalA2AAgent) GetDescription() string {
	return e.description
}

func (e *ExternalA2AAgent) GetConfig() *config.AgentConfig {
	return e.config
}
