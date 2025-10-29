package agent

import (
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
)

type AgentRouter struct {
	pb.UnimplementedA2AServiceServer
	registry *AgentRegistry
}

func NewAgentRouter(registry *AgentRegistry) *AgentRouter {
	return &AgentRouter{
		registry: registry,
	}
}

func (s *AgentRouter) RegisterAgent(agentName string, agentSvc pb.A2AServiceServer) {

	log.Printf("  ✅ Registered agent: %s", agentName)
}

func (s *AgentRouter) GetAgentCardAndVisibility(agentName string) (*pb.AgentCard, string, error) {
	agentSvc, err := s.registry.GetAgent(agentName)
	if err != nil {
		return nil, "", fmt.Errorf("agent not found: %s", agentName)
	}

	agentConfig, err := s.registry.GetAgentConfig(agentName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get agent config: %w", err)
	}

	card, err := agentSvc.GetAgentCard(context.Background(), &pb.GetAgentCardRequest{})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get agent card: %w", err)
	}

	visibility := agentConfig.Visibility
	if visibility == "" {
		visibility = "public"
	}

	return card, visibility, nil
}

func (s *AgentRouter) ListAgents() []string {
	return s.registry.ListAgents()
}

func (s *AgentRouter) GetAgent(agentName string) (pb.A2AServiceServer, bool) {
	agent, err := s.registry.GetAgent(agentName)
	if err != nil {
		return nil, false
	}
	return agent, true
}

func (s *AgentRouter) AgentCount() int {
	return len(s.registry.ListAgents())
}

func (s *AgentRouter) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	agentName, err := s.getAgentNameOrError(ctx, req)
	if err != nil {
		return nil, err
	}

	agentSvc, err := s.registry.GetAgent(agentName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "agent '%s' not found", agentName)
	}

	return agentSvc.SendMessage(ctx, req)
}

func (s *AgentRouter) SendStreamingMessage(req *pb.SendMessageRequest, stream pb.A2AService_SendStreamingMessageServer) error {
	agentName, err := s.getAgentNameOrError(stream.Context(), req)
	if err != nil {
		return err
	}

	agentSvc, err := s.registry.GetAgent(agentName)
	if err != nil {
		return status.Errorf(codes.NotFound, "agent '%s' not found", agentName)
	}

	return agentSvc.SendStreamingMessage(req, stream)
}

func (s *AgentRouter) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error) {

	agentName := s.extractAgentNameFromContext(ctx)

	if agentName == "" {
		agentNames := s.registry.ListAgents()
		if len(agentNames) == 1 {
			agentName = agentNames[0]
		} else {
			return nil, status.Error(codes.InvalidArgument,
				"name required: use ?name=<agent_name> query parameter or specify agent in path")
		}
	}

	agent, err := s.registry.GetAgent(agentName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "agent '%s' not found", agentName)
	}

	return agent.GetAgentCard(ctx, req)
}

func (s *AgentRouter) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	agentSvc, err := s.routeToSingleAgent()
	if err != nil {
		return nil, err
	}
	return agentSvc.GetTask(ctx, req)
}

func (s *AgentRouter) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error) {
	agentSvc, err := s.routeToSingleAgent()
	if err != nil {
		return nil, err
	}
	return agentSvc.CancelTask(ctx, req)
}

func (s *AgentRouter) TaskSubscription(req *pb.TaskSubscriptionRequest, stream pb.A2AService_TaskSubscriptionServer) error {
	agentSvc, err := s.routeToSingleAgent()
	if err != nil {
		return err
	}
	return agentSvc.TaskSubscription(req, stream)
}

func (s *AgentRouter) CreateTaskPushNotificationConfig(ctx context.Context, req *pb.CreateTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

func (s *AgentRouter) GetTaskPushNotificationConfig(ctx context.Context, req *pb.GetTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

func (s *AgentRouter) ListTaskPushNotificationConfig(ctx context.Context, req *pb.ListTaskPushNotificationConfigRequest) (*pb.ListTaskPushNotificationConfigResponse, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

func (s *AgentRouter) DeleteTaskPushNotificationConfig(ctx context.Context, req *pb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

func (s *AgentRouter) routeToSingleAgent() (pb.A2AServiceServer, error) {
	agentNames := s.registry.ListAgents()
	if len(agentNames) == 1 {
		agent, err := s.registry.GetAgent(agentNames[0])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get agent: %v", err)
		}
		return agent, nil
	}
	return nil, status.Error(codes.Unimplemented, "operation requires agent specification in multi-agent mode")
}

func (s *AgentRouter) getAgentNameOrError(ctx context.Context, req *pb.SendMessageRequest) (string, error) {

	agentName := s.extractAgentNameFromContext(ctx)

	if agentName == "" {
		agentName = s.extractAgentName(req)
	}

	if agentName == "" {
		agentNames := s.registry.ListAgents()
		if len(agentNames) == 1 {
			// Single-agent mode: automatically route to the only available agent
			return agentNames[0], nil
		}

		return "", status.Error(codes.InvalidArgument, "name not specified (use context_id format: agent_name:session_id or set agent-name in metadata)")
	}

	return agentName, nil
}

func (s *AgentRouter) extractAgentName(req *pb.SendMessageRequest) string {
	if req.Request == nil {
		return ""
	}

	if req.Request.ContextId != "" {
		parts := strings.SplitN(req.Request.ContextId, ":", 2)

		if len(parts) == 2 && parts[0] != "" {
			return parts[0]
		}

	}

	if req.Request.Metadata != nil {
		if name, ok := req.Request.Metadata.Fields["name"]; ok {
			return name.GetStringValue()
		}

		if agentID, ok := req.Request.Metadata.Fields["agent_id"]; ok {
			return agentID.GetStringValue()
		}
	}

	return ""
}

func (s *AgentRouter) extractAgentNameFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	agentNames := md.Get("agent-name")
	if len(agentNames) == 0 {
		return ""
	}

	return agentNames[0]
}
