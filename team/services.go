package team

import (
	"context"
	"fmt"
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

// ExecuteWorkflow executes a workflow using the appropriate executor
func (s *TeamWorkflowService) ExecuteWorkflow(ctx context.Context, request *workflow.WorkflowRequest) (*workflow.WorkflowResult, error) {
	if request == nil {
		return nil, NewTeamError("TeamWorkflowService", "ExecuteWorkflow", "request cannot be nil", nil)
	}

	// Cast to concrete registry type to access ExecuteWorkflow method
	if registry, ok := s.workflowRegistry.(*workflow.WorkflowExecutorRegistry); ok {
		return registry.ExecuteWorkflow(ctx, request)
	}
	return nil, NewTeamError("TeamWorkflowService", "ExecuteWorkflow", "invalid registry type", nil)
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
	agentRegistry    *hectoragent.AgentRegistry
	componentManager *component.ComponentManager
}

// NewTeamAgentService creates a new team agent service
func NewTeamAgentService(componentManager *component.ComponentManager) *TeamAgentService {
	return &TeamAgentService{
		agentRegistry:    hectoragent.NewAgentRegistry(),
		componentManager: componentManager,
	}
}

// CreateAndRegisterAgent creates and registers an agent
func (s *TeamAgentService) CreateAndRegisterAgent(agentName string, agentConfig *config.AgentConfig) (*hectoragent.Agent, error) {
	if agentName == "" {
		return nil, NewTeamError("TeamAgentService", "CreateAndRegisterAgent", "agent name cannot be empty", nil)
	}
	if agentConfig == nil {
		return nil, NewTeamError("TeamAgentService", "CreateAndRegisterAgent", "agent config cannot be nil", nil)
	}

	// Create agent instance using component manager
	agent, err := hectoragent.NewAgent(agentConfig, s.componentManager)
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

// ExecuteAgent executes an agent with the given input and returns the result
func (s *TeamAgentService) ExecuteAgent(ctx context.Context, agentName string, input string) (*workflow.AgentResult, error) {
	agent, err := s.GetAgent(agentName)
	if err != nil {
		return nil, NewTeamError("TeamAgentService", "ExecuteAgent",
			fmt.Sprintf("failed to get agent %s", agentName), err)
	}

	// Execute the agent query
	response, err := agent.Query(ctx, input)
	if err != nil {
		return nil, NewTeamError("TeamAgentService", "ExecuteAgent",
			fmt.Sprintf("failed to execute agent %s", agentName), err)
	}

	// Convert reasoning response to workflow agent result
	result := &workflow.AgentResult{
		AgentName:  agentName,
		StepName:   agentName, // Use agent name as step name for now
		Result:     response.Answer,
		Success:    true,
		Duration:   response.Duration,
		TokensUsed: response.TokensUsed,
		Artifacts:  make(map[string]workflow.Artifact),
		Metadata:   make(map[string]string),
		Timestamp:  time.Now(),
		Confidence: response.Confidence,
	}

	// Convert context search results to metadata
	if len(response.Sources) > 0 {
		result.Metadata["sources"] = fmt.Sprintf("%v", response.Sources)
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
