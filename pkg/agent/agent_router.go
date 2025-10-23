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

// AgentRouter routes A2A protocol requests to individual agents.
// It acts as a multiplexer/gateway, delegating to AgentRegistry for agent storage and lookup.
// The router implements the A2AService interface and handles request routing based on agent names.
// Following LangGraph's A2A pattern: each agent has its own identity (card.Name), the router just routes.
type AgentRouter struct {
	pb.UnimplementedA2AServiceServer
	registry *AgentRegistry
}

// NewAgentRouter creates a new agent router with the given registry.
func NewAgentRouter(registry *AgentRegistry) *AgentRouter {
	return &AgentRouter{
		registry: registry,
	}
}

// RegisterAgent registers an agent with the router.
// Note: Actual registration happens in AgentRegistry - this is just for logging.
func (s *AgentRouter) RegisterAgent(agentName string, agentSvc pb.A2AServiceServer) {
	// AgentRegistry is the single source of truth - registration happens there
	// This method is kept for backward compatibility and logging
	log.Printf("  ✅ Registered agent: %s", agentName)
}

// GetAgentCardAndVisibility returns the A2A agent card and visibility for discovery.
// Returns the official pb.AgentCard (A2A protocol) and the Hector-specific visibility setting.
func (s *AgentRouter) GetAgentCardAndVisibility(agentName string) (*pb.AgentCard, string, error) {
	agentSvc, err := s.registry.GetAgent(agentName)
	if err != nil {
		return nil, "", fmt.Errorf("agent not found: %s", agentName)
	}

	// Get agent config from registry for visibility
	agentConfig, err := s.registry.GetAgentConfig(agentName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get agent config: %w", err)
	}

	// Get official A2A agent card from the agent itself
	card, err := agentSvc.GetAgentCard(context.Background(), &pb.GetAgentCardRequest{})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get agent card: %w", err)
	}

	visibility := agentConfig.Visibility
	if visibility == "" {
		visibility = "public" // Default visibility
	}

	return card, visibility, nil
}

// ListAgents returns all registered agents (for discovery).
// Delegates to AgentRegistry.
func (s *AgentRouter) ListAgents() []string {
	return s.registry.ListAgents()
}

// GetAgent returns a specific agent by name.
// Delegates to AgentRegistry.
func (s *AgentRouter) GetAgent(agentName string) (pb.A2AServiceServer, bool) {
	agent, err := s.registry.GetAgent(agentName)
	if err != nil {
		return nil, false
	}
	return agent, true
}

// AgentCount returns the number of registered agents.
// Delegates to AgentRegistry.
func (s *AgentRouter) AgentCount() int {
	return len(s.registry.ListAgents())
}

// SendMessage routes to the appropriate agent.
// Delegates to AgentRegistry for agent lookup.
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

// SendStreamingMessage routes to the appropriate agent.
// Delegates to AgentRegistry for agent lookup.
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

// GetAgentCard returns the card for a specific agent.
// Following LangGraph's A2A pattern: requires agent name via gRPC metadata or query parameter.
// The REST gateway sets this from: /.well-known/agent-card.json?name={agent_name}
func (s *AgentRouter) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error) {
	// Extract agent name from gRPC metadata (set by REST gateway from query param)
	agentName := s.extractAgentNameFromContext(ctx)

	// If no agent name in metadata, try single agent fallback
	if agentName == "" {
		agentNames := s.registry.ListAgents()
		if len(agentNames) == 1 {
			agentName = agentNames[0]
		} else {
			return nil, status.Error(codes.InvalidArgument,
				"name required: use ?name=<agent_name> query parameter or specify agent in path")
		}
	}

	// Get the specific agent and return its card
	agent, err := s.registry.GetAgent(agentName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "agent '%s' not found", agentName)
	}

	return agent.GetAgentCard(ctx, req)
}

// GetTask routes to the appropriate agent.
func (s *AgentRouter) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	agentSvc, err := s.routeToSingleAgent()
	if err != nil {
		return nil, err
	}
	return agentSvc.GetTask(ctx, req)
}

// CancelTask routes to the appropriate agent.
func (s *AgentRouter) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error) {
	agentSvc, err := s.routeToSingleAgent()
	if err != nil {
		return nil, err
	}
	return agentSvc.CancelTask(ctx, req)
}

// TaskSubscription routes to the appropriate agent.
func (s *AgentRouter) TaskSubscription(req *pb.TaskSubscriptionRequest, stream pb.A2AService_TaskSubscriptionServer) error {
	agentSvc, err := s.routeToSingleAgent()
	if err != nil {
		return err
	}
	return agentSvc.TaskSubscription(req, stream)
}

// CreateTaskPushNotificationConfig is not implemented yet.
func (s *AgentRouter) CreateTaskPushNotificationConfig(ctx context.Context, req *pb.CreateTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

// GetTaskPushNotificationConfig is not implemented yet.
func (s *AgentRouter) GetTaskPushNotificationConfig(ctx context.Context, req *pb.GetTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

// ListTaskPushNotificationConfig is not implemented yet.
func (s *AgentRouter) ListTaskPushNotificationConfig(ctx context.Context, req *pb.ListTaskPushNotificationConfigRequest) (*pb.ListTaskPushNotificationConfigResponse, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

// DeleteTaskPushNotificationConfig is not implemented yet.
func (s *AgentRouter) DeleteTaskPushNotificationConfig(ctx context.Context, req *pb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "push notifications not implemented")
}

// ============================================================================
// INTERNAL HELPER METHODS
// ============================================================================

// routeToSingleAgent returns the single agent if only one is registered.
// Returns an error if multiple agents are registered (operation requires agent specification).
// Delegates to AgentRegistry for agent lookup.
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

// getAgentNameOrError extracts and validates agent name from context or request.
// Returns the agent name or an error if not found.
func (s *AgentRouter) getAgentNameOrError(ctx context.Context, req *pb.SendMessageRequest) (string, error) {
	// First try to get agent name from gRPC metadata (set by REST gateway)
	agentName := s.extractAgentNameFromContext(ctx)

	// If not found, try extracting from request payload
	if agentName == "" {
		agentName = s.extractAgentName(req)
	}

	// If still empty and we only have one agent, use it (single-agent mode)
	// This is safe because there's only one agent to route to
	if agentName == "" {
		agentNames := s.registry.ListAgents()
		if len(agentNames) == 1 {
			log.Printf("AgentRouter: No agent name specified, using single agent: %s", agentNames[0])
			return agentNames[0], nil
		}
		// Multiple agents require explicit agent specification
		return "", status.Error(codes.InvalidArgument, "name not specified (use context_id format: agent_name:session_id or set agent-name in metadata)")
	}

	return agentName, nil
}

// extractAgentName extracts agent name from the request payload.
func (s *AgentRouter) extractAgentName(req *pb.SendMessageRequest) string {
	if req.Request == nil {
		return ""
	}

	// Try context_id format: "agent_name:session_id"
	if req.Request.ContextId != "" {
		parts := strings.SplitN(req.Request.ContextId, ":", 2)
		// Only use it if there are actually 2 parts (agent:session format)
		if len(parts) == 2 && parts[0] != "" {
			return parts[0]
		}
		// If no colon, context_id is just a session ID, not an agent name
	}

	// Try metadata - check both "name" (A2A standard) and "agent_id" (legacy)
	if req.Request.Metadata != nil {
		if name, ok := req.Request.Metadata.Fields["name"]; ok {
			return name.GetStringValue()
		}
		// Backward compatibility: also check agent_id
		if agentID, ok := req.Request.Metadata.Fields["agent_id"]; ok {
			return agentID.GetStringValue()
		}
	}

	return ""
}

// extractAgentNameFromContext extracts agent name from gRPC metadata (set by REST gateway).
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
