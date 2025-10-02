package team

import (
	"context"
	"fmt"
	"strings"
	"time"

	hectoragent "github.com/kadirpekel/hector/agent"
	"github.com/kadirpekel/hector/component"
	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/workflow"
)

// ============================================================================
// TEAM SERVICES - FOLLOWING AGENT/REASONING PATTERN
// ============================================================================

// TeamWorkflowService handles workflow execution for teams
type TeamWorkflowService struct {
	workflowRegistry workflow.WorkflowRegistryService
	workflowFactory  workflow.WorkflowFactoryService
}

// NewTeamWorkflowService creates a new team workflow service
func NewTeamWorkflowService(registry workflow.WorkflowRegistryService, factory workflow.WorkflowFactoryService) *TeamWorkflowService {
	return &TeamWorkflowService{
		workflowRegistry: registry,
		workflowFactory:  factory,
	}
}

// ExecuteWorkflowStreaming executes a workflow with streaming using the appropriate executor
func (s *TeamWorkflowService) ExecuteWorkflowStreaming(ctx context.Context, request *workflow.WorkflowRequest) (<-chan workflow.WorkflowEvent, error) {
	if request == nil {
		errCh := make(chan workflow.WorkflowEvent, 1)
		errCh <- workflow.WorkflowEvent{
			Timestamp: time.Now(),
			EventType: workflow.EventAgentError,
			Content:   "Request cannot be nil",
		}
		close(errCh)
		return errCh, NewTeamError("TeamWorkflowService", "ExecuteWorkflowStreaming", "request cannot be nil", nil)
	}

	// Cast to concrete registry type to access ExecuteWorkflowStreaming method
	if registry, ok := s.workflowRegistry.(*workflow.WorkflowExecutorRegistry); ok {
		return registry.ExecuteWorkflowStreaming(ctx, request)
	}

	errCh := make(chan workflow.WorkflowEvent, 1)
	errCh <- workflow.WorkflowEvent{
		Timestamp: time.Now(),
		EventType: workflow.EventAgentError,
		Content:   "Invalid registry type",
	}
	close(errCh)
	return errCh, NewTeamError("TeamWorkflowService", "ExecuteWorkflowStreaming", "invalid registry type", nil)
}

// GetSupportedModes returns supported workflow execution modes
func (s *TeamWorkflowService) GetSupportedModes() []string {
	// Cast to concrete registry type to access GetSupportedModes method
	if registry, ok := s.workflowRegistry.(*workflow.WorkflowExecutorRegistry); ok {
		return registry.GetSupportedModes()
	}
	return []string{}
}

// CanHandle checks if the service can handle the given workflow
func (s *TeamWorkflowService) CanHandle(workflowConfig *config.WorkflowConfig) bool {
	if workflowConfig == nil {
		return false
	}

	// Check if we have an executor that can handle this workflow
	executors := s.workflowRegistry.GetSupportedExecutorsByMode(string(workflowConfig.Mode))
	return len(executors) > 0
}

// ============================================================================
// TEAM AGENT SERVICE - MANAGES AGENT LIFECYCLE AND IMPLEMENTS WORKFLOW.AGENTSERVICES
// ============================================================================

// TeamAgentService handles agent management for teams and provides workflow abstraction
type TeamAgentService struct {
	agentRegistry *hectoragent.AgentRegistry
	agentFactory  *hectoragent.AgentFactory // Use factory for agent creation
}

// NewTeamAgentService creates a new team agent service
func NewTeamAgentService(componentManager *component.ComponentManager) *TeamAgentService {
	return &TeamAgentService{
		agentRegistry: hectoragent.NewAgentRegistry(),
		agentFactory:  hectoragent.NewAgentFactory(componentManager), // Initialize factory
	}
}

// CreateAndRegisterAgent creates and registers an agent using the agent factory
func (s *TeamAgentService) CreateAndRegisterAgent(agentName string, agentConfig *config.AgentConfig) (*hectoragent.Agent, error) {
	if agentName == "" {
		return nil, NewTeamError("TeamAgentService", "CreateAndRegisterAgent", "agent name cannot be empty", nil)
	}
	if agentConfig == nil {
		return nil, NewTeamError("TeamAgentService", "CreateAndRegisterAgent", "agent config cannot be nil", nil)
	}

	// Use factory to create agent - NO DUPLICATION!
	agent, err := s.agentFactory.CreateAgent(agentConfig)
	if err != nil {
		return nil, NewTeamError("TeamAgentService", "CreateAndRegisterAgent",
			fmt.Sprintf("failed to create agent instance for %s", agentName), err)
	}

	// Register the agent
	capabilities := s.getDefaultCapabilities()
	err = s.agentRegistry.RegisterAgent(agentName, agent, agentConfig, capabilities)
	if err != nil {
		return nil, NewTeamError("TeamAgentService", "CreateAndRegisterAgent",
			fmt.Sprintf("failed to register agent %s", agentName), err)
	}

	return agent, nil
}

// GetAgent retrieves an agent by name
func (s *TeamAgentService) GetAgent(name string) (*hectoragent.Agent, error) {
	return s.agentRegistry.GetAgent(name)
}

// GetAllAgents returns all registered agents
func (s *TeamAgentService) GetAllAgents() map[string]*hectoragent.Agent {
	return s.agentRegistry.GetAllAgents()
}

// ListAgents returns all registered agent names
func (s *TeamAgentService) ListAgents() []string {
	return s.agentRegistry.ListAgents()
}

// getDefaultCapabilities returns default capabilities for agents
func (s *TeamAgentService) getDefaultCapabilities() []string {
	return []string{"general", "reasoning", "search", "tools"}
}

// ============================================================================
// WORKFLOW.AGENTSERVICES INTERFACE IMPLEMENTATION
// ============================================================================

// ExecuteAgentStreaming executes an agent with streaming output
func (s *TeamAgentService) ExecuteAgentStreaming(ctx context.Context, agentName string, input string, eventCh chan<- workflow.WorkflowEvent) (*workflow.AgentResult, error) {
	agent, err := s.GetAgent(agentName)
	if err != nil {
		return nil, NewTeamError("TeamAgentService", "ExecuteAgentStreaming",
			fmt.Sprintf("failed to get agent %s", agentName), err)
	}

	// Send agent start event
	eventCh <- workflow.WorkflowEvent{
		Timestamp: time.Now(),
		EventType: workflow.EventAgentStart,
		AgentName: agentName,
		Content:   fmt.Sprintf("🤖 Starting agent: %s", agentName),
	}

	startTime := time.Now()

	// Stream agent's response
	responseCh, err := agent.QueryStreaming(ctx, input)
	if err != nil {
		eventCh <- workflow.WorkflowEvent{
			Timestamp: time.Now(),
			EventType: workflow.EventAgentError,
			AgentName: agentName,
			Content:   fmt.Sprintf("❌ Agent failed: %v", err),
		}
		return nil, NewTeamError("TeamAgentService", "ExecuteAgentStreaming",
			fmt.Sprintf("failed to stream agent %s", agentName), err)
	}

	var fullResponse strings.Builder
	for chunk := range responseCh {
		fullResponse.WriteString(chunk)

		// Forward agent output to workflow event stream
		eventCh <- workflow.WorkflowEvent{
			Timestamp: time.Now(),
			EventType: workflow.EventAgentOutput,
			AgentName: agentName,
			Content:   chunk,
		}
	}

	duration := time.Since(startTime)

	// Send agent complete event
	eventCh <- workflow.WorkflowEvent{
		Timestamp: time.Now(),
		EventType: workflow.EventAgentComplete,
		AgentName: agentName,
		Content:   fmt.Sprintf("✅ Agent %s completed in %.2fs", agentName, duration.Seconds()),
		Metadata: map[string]string{
			"duration": duration.String(),
		},
	}

	// Create result
	result := &workflow.AgentResult{
		AgentName:  agentName,
		StepName:   agentName,
		Result:     fullResponse.String(),
		Success:    true,
		Duration:   duration,
		Timestamp:  time.Now(),
		Artifacts:  make(map[string]workflow.Artifact),
		Metadata:   make(map[string]string),
		Confidence: 1.0,
	}

	return result, nil
}

// GetAvailableAgents returns the list of available agent names
func (s *TeamAgentService) GetAvailableAgents() []string {
	return s.ListAgents()
}

// GetAgentCapabilities returns the capabilities of a specific agent
func (s *TeamAgentService) GetAgentCapabilities(agentName string) ([]string, error) {
	capabilities, err := s.agentRegistry.GetCapabilities(agentName)
	if err != nil {
		return nil, NewTeamError("TeamAgentService", "GetAgentCapabilities",
			fmt.Sprintf("failed to get capabilities for agent %s", agentName), err)
	}
	return capabilities, nil
}

// IsAgentAvailable checks if an agent is available for execution
func (s *TeamAgentService) IsAgentAvailable(agentName string) bool {
	_, err := s.GetAgent(agentName)
	return err == nil
}

// ============================================================================
// TEAM COORDINATION SERVICE - MANAGES TEAM STATE
// ============================================================================

// TeamCoordinationService handles team coordination and shared state
type TeamCoordinationService struct {
	sharedState *SharedState
}

// NewTeamCoordinationService creates a new team coordination service
func NewTeamCoordinationService() *TeamCoordinationService {
	return &TeamCoordinationService{
		sharedState: NewSharedState(),
	}
}

// GetSharedState returns the shared state
func (s *TeamCoordinationService) GetSharedState() *SharedState {
	return s.sharedState
}

// SetContext sets a context value in shared state
func (s *TeamCoordinationService) SetContext(key string, value interface{}, agent string) error {
	return s.sharedState.SetContext(key, value, agent)
}

// GetContext gets a context value from shared state
func (s *TeamCoordinationService) GetContext(key string) (interface{}, bool) {
	return s.sharedState.GetContext(key)
}

// SetResult sets an agent result in shared state
func (s *TeamCoordinationService) SetResult(stepName string, result *workflow.AgentResult) error {
	// Since AgentResult is now an alias to workflow.AgentResult, we can use it directly
	return s.sharedState.SetResult(stepName, result)
}

// GetResult gets an agent result from shared state
func (s *TeamCoordinationService) GetResult(stepName string) (*AgentResult, bool) {
	return s.sharedState.GetResult(stepName)
}

// GetAllResults returns all results from shared state
func (s *TeamCoordinationService) GetAllResults() map[string]*AgentResult {
	return s.sharedState.GetAllResults()
}

// ClearHistory clears old history entries
func (s *TeamCoordinationService) ClearHistory() {
	s.sharedState.ClearHistory()
}
