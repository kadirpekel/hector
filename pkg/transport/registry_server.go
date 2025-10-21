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

// RegistryService is a gRPC service that routes requests to multiple agents via a registry
// It implements pb.A2AServiceServer and multiplexes based on agent name from request metadata
type RegistryService struct {
	pb.UnimplementedA2AServiceServer
	registry *agent.AgentRegistry
}

// NewRegistryService creates a new registry-based service
func NewRegistryService(registry *agent.AgentRegistry) *RegistryService {
	return &RegistryService{
		registry: registry,
	}
}

// getAgentFromContext extracts the target agent name from request metadata
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

// getAgent retrieves an agent from the registry by name
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

// SendMessage routes the request to the appropriate agent
func (s *RegistryService) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.SendMessage(ctx, req)
}

// SendStreamingMessage routes the streaming request to the appropriate agent
func (s *RegistryService) SendStreamingMessage(req *pb.SendMessageRequest, stream pb.A2AService_SendStreamingMessageServer) error {
	agent, err := s.getAgent(stream.Context())
	if err != nil {
		return err
	}
	return agent.SendStreamingMessage(req, stream)
}

// GetAgentCard routes the request to the appropriate agent
func (s *RegistryService) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.GetAgentCard(ctx, req)
}

// GetTask routes the request to the appropriate agent
func (s *RegistryService) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.GetTask(ctx, req)
}

// CancelTask routes the request to the appropriate agent
func (s *RegistryService) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.CancelTask(ctx, req)
}

// TaskSubscription routes the streaming request to the appropriate agent
func (s *RegistryService) TaskSubscription(req *pb.TaskSubscriptionRequest, stream pb.A2AService_TaskSubscriptionServer) error {
	agent, err := s.getAgent(stream.Context())
	if err != nil {
		return err
	}
	return agent.TaskSubscription(req, stream)
}

// CreateTaskPushNotificationConfig routes the request to the appropriate agent
func (s *RegistryService) CreateTaskPushNotificationConfig(ctx context.Context, req *pb.CreateTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.CreateTaskPushNotificationConfig(ctx, req)
}

// GetTaskPushNotificationConfig routes the request to the appropriate agent
func (s *RegistryService) GetTaskPushNotificationConfig(ctx context.Context, req *pb.GetTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.GetTaskPushNotificationConfig(ctx, req)
}

// ListTaskPushNotificationConfig routes the request to the appropriate agent
func (s *RegistryService) ListTaskPushNotificationConfig(ctx context.Context, req *pb.ListTaskPushNotificationConfigRequest) (*pb.ListTaskPushNotificationConfigResponse, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.ListTaskPushNotificationConfig(ctx, req)
}

// DeleteTaskPushNotificationConfig routes the request to the appropriate agent
func (s *RegistryService) DeleteTaskPushNotificationConfig(ctx context.Context, req *pb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	agent, err := s.getAgent(ctx)
	if err != nil {
		return nil, err
	}
	return agent.DeleteTaskPushNotificationConfig(ctx, req)
}

// ListAgents returns a list of all registered agent names (custom endpoint for discovery)
func (s *RegistryService) ListAgents() []string {
	entries := s.registry.List()
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name
	}
	return names
}

// GetAgentByName retrieves an agent by name without requiring context metadata
// Used by REST gateway for direct agent access (e.g., agent card endpoint)
func (s *RegistryService) GetAgentByName(name string) (pb.A2AServiceServer, error) {
	agentEntry, ok := s.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("agent '%s' not found", name)
	}
	return agentEntry.Agent, nil
}

// GetAgent retrieves an agent by name (implements DiscoverableService interface)
func (s *RegistryService) GetAgent(agentName string) (pb.A2AServiceServer, bool) {
	agentEntry, ok := s.registry.Get(agentName)
	if !ok {
		return nil, false
	}
	return agentEntry.Agent, true
}

// GetAgentCardAndVisibility returns the A2A agent card and visibility for a specific agent
func (s *RegistryService) GetAgentCardAndVisibility(name string) (*pb.AgentCard, string, error) {
	agentEntry, ok := s.registry.Get(name)
	if !ok {
		return nil, "", fmt.Errorf("agent '%s' not found", name)
	}

	// Get official A2A agent card
	ctx := context.Background()
	card, err := agentEntry.Agent.GetAgentCard(ctx, &pb.GetAgentCardRequest{})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get agent card: %w", err)
	}

	visibility := agentEntry.Config.Visibility
	if visibility == "" {
		visibility = "public" // Default visibility
	}

	return card, visibility, nil
}

// Compile-time check that RegistryService implements pb.A2AServiceServer
var _ pb.A2AServiceServer = (*RegistryService)(nil)

// NewRegistryServer creates a new gRPC server with registry-based routing
func NewRegistryServer(registry *agent.AgentRegistry, config Config) *Server {
	service := NewRegistryService(registry)
	return NewServer(service, config)
}

// ServeAgent creates and starts a server for a single agent (convenience function)
func ServeAgent(agent *agent.Agent, address string) (*Server, error) {
	if agent == nil {
		return nil, fmt.Errorf("agent cannot be nil")
	}

	// Create server directly with the agent as the service
	server := NewServer(agent, Config{Address: address})

	// Start server in background
	go func() {
		if err := server.Start(); err != nil {
			fmt.Printf("❌ Server error: %v\n", err)
		}
	}()

	return server, nil
}
