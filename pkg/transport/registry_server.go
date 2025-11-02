package transport

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
)

type RegistryService struct {
	pb.UnimplementedA2AServiceServer
	registry *agent.AgentRegistry
}

func NewRegistryService(registry *agent.AgentRegistry) *RegistryService {
	return &RegistryService{
		registry: registry,
	}
}

func (s *RegistryService) getAgentFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.InvalidArgument, "missing metadata")
	}

	agentNames := md.Get("agent-name")
	if len(agentNames) == 0 {
		return "", status.Error(codes.InvalidArgument, "missing agent-name in metadata")
	}

	return agentNames[0], nil
}

func (s *RegistryService) getAgent(ctx context.Context) (pb.A2AServiceServer, error) {
	agentName, err := s.getAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	agentEntry, ok := s.registry.Get(agentName)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "agent '%s' not found", agentName)
	}

	return agentEntry.Agent, nil
}

func (s *RegistryService) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.SendMessage(ctx, req)
}

func (s *RegistryService) SendStreamingMessage(req *pb.SendMessageRequest, stream pb.A2AService_SendStreamingMessageServer) error {
	agent, err := s.getAgent(stream.Context())
	if err != nil {
		return err
	}
	return agent.SendStreamingMessage(req, stream)
}

func (s *RegistryService) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.GetAgentCard(ctx, req)
}

func (s *RegistryService) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.GetTask(ctx, req)
}

func (s *RegistryService) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.ListTasks(ctx, req)
}

func (s *RegistryService) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.CancelTask(ctx, req)
}

func (s *RegistryService) TaskSubscription(req *pb.TaskSubscriptionRequest, stream pb.A2AService_TaskSubscriptionServer) error {
	agent, err := s.getAgent(stream.Context())
	if err != nil {
		return err
	}
	return agent.TaskSubscription(req, stream)
}

func (s *RegistryService) CreateTaskPushNotificationConfig(ctx context.Context, req *pb.CreateTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.CreateTaskPushNotificationConfig(ctx, req)
}

func (s *RegistryService) GetTaskPushNotificationConfig(ctx context.Context, req *pb.GetTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.GetTaskPushNotificationConfig(ctx, req)
}

func (s *RegistryService) ListTaskPushNotificationConfig(ctx context.Context, req *pb.ListTaskPushNotificationConfigRequest) (*pb.ListTaskPushNotificationConfigResponse, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.ListTaskPushNotificationConfig(ctx, req)
}

func (s *RegistryService) DeleteTaskPushNotificationConfig(ctx context.Context, req *pb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.DeleteTaskPushNotificationConfig(ctx, req)
}

func (s *RegistryService) ListAgents() []string {
	entries := s.registry.List()
	ids := make([]string, len(entries))
	for i, entry := range entries {
		ids[i] = entry.ID
	}
	return ids
}

func (s *RegistryService) GetAgentByName(agentID string) (pb.A2AServiceServer, error) {
	agentEntry, ok := s.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", agentID)
	}
	return agentEntry.Agent, nil
}

func (s *RegistryService) GetAgent(agentName string) (pb.A2AServiceServer, bool) {
	agentEntry, ok := s.registry.Get(agentName)
	if !ok {
		return nil, false
	}
	return agentEntry.Agent, true
}

func (s *RegistryService) GetAgentCardAndVisibility(name string) (*pb.AgentCard, string, error) {
	agentEntry, ok := s.registry.Get(name)
	if !ok {
		return nil, "", fmt.Errorf("agent '%s' not found", name)
	}

	ctx := context.Background()
	card, err := agentEntry.Agent.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get agent card: %w", err)
	}

	visibility := agentEntry.Config.Visibility
	if visibility == "" {
		visibility = "public"
	}

	return card, visibility, nil
}

var _ pb.A2AServiceServer = (*RegistryService)(nil)

func NewRegistryServer(registry *agent.AgentRegistry, config Config) *Server {
	service := NewRegistryService(registry)
	return NewServer(service, config)
}

func ServeAgent(agent *agent.Agent, address string) (*Server, error) {
	if agent == nil {
		return nil, fmt.Errorf("agent cannot be nil")
	}

	server := NewServer(agent, Config{Address: address})

	go func() {
		if err := server.Start(); err != nil {
			fmt.Printf("❌ Server error: %v\n", err)
		}
	}()

	return server, nil
}
