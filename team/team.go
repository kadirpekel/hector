// Package hector provides AI agent team coordination and workflow orchestration.
// Team manages multi-agent workflows with DAG and autonomous execution.
package team

import (
	"context"
	"fmt"
	"sync"
	"time"

	hectoragent "github.com/kadirpekel/hector/agent"
	"github.com/kadirpekel/hector/component"
	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/workflow"
)

// ============================================================================
// TYPE ALIASES - USE WORKFLOW MODULE AS SOURCE OF TRUTH
// ============================================================================

// WorkflowStatus represents the current state of a workflow execution
type WorkflowStatus = workflow.WorkflowStatus

const (
	WorkflowStatusPending   = workflow.WorkflowStatusPending
	WorkflowStatusRunning   = workflow.WorkflowStatusRunning
	WorkflowStatusCompleted = workflow.WorkflowStatusCompleted
	WorkflowStatusFailed    = workflow.WorkflowStatusFailed
	WorkflowStatusCancelled = workflow.WorkflowStatusCancelled
)

// StepStatus represents the execution status of a workflow step
type StepStatus = workflow.StepStatus

const (
	StepStatusPending   = workflow.StepStatusPending
	StepStatusReady     = workflow.StepStatusReady
	StepStatusRunning   = workflow.StepStatusRunning
	StepStatusCompleted = workflow.StepStatusCompleted
	StepStatusFailed    = workflow.StepStatusFailed
	StepStatusSkipped   = workflow.StepStatusSkipped
)

// AgentResult represents the output from an agent execution - USE WORKFLOW TYPES
type AgentResult = workflow.AgentResult

// WorkflowResult represents the final result of a multi-agent workflow - USE WORKFLOW TYPES
type WorkflowResult = workflow.WorkflowResult

// ============================================================================
// CONFIGURATION CONSTANTS
// ============================================================================

const (
	// DefaultAgentLLM is the default LLM provider for agents
	DefaultAgentLLM = "ollama"

	// DefaultAgentDatabase is the default database provider for agents
	DefaultAgentDatabase = "qdrant"

	// DefaultAgentEmbedder is the default embedder provider for agents
	DefaultAgentEmbedder = "ollama"

	// MaxHistorySize is the maximum number of history entries to keep
	MaxHistorySize = 1000
)

// ============================================================================
// ERRORS - STANDARDIZED ERROR TYPES
// ============================================================================

// TeamError represents errors in the team system
type TeamError struct {
	Component string
	Operation string
	Message   string
	Err       error
	Timestamp time.Time
}

func (e *TeamError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Component, e.Operation, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Component, e.Operation, e.Message)
}

func (e *TeamError) Unwrap() error {
	return e.Err
}

// NewTeamError creates a new team error
func NewTeamError(component, operation, message string, err error) *TeamError {
	return &TeamError{
		Component: component,
		Operation: operation,
		Message:   message,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// ============================================================================
// SHARED STATE AND COMMUNICATION
// ============================================================================

// SharedState manages inter-agent communication and shared context
type SharedState struct {
	mu       sync.RWMutex
	Context  map[string]interface{}  `json:"context"`
	Memory   map[string]interface{}  `json:"memory"`
	Results  map[string]*AgentResult `json:"results"`
	Metadata map[string]string       `json:"metadata"`
	History  []StateChange           `json:"history"`
}

// StateChange represents a change in shared state
type StateChange struct {
	Timestamp time.Time   `json:"timestamp"`
	Agent     string      `json:"agent"`
	Action    string      `json:"action"`
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
}

// NewSharedState creates a new shared state instance
func NewSharedState() *SharedState {
	return &SharedState{
		Context:  make(map[string]interface{}),
		Memory:   make(map[string]interface{}),
		Results:  make(map[string]*AgentResult),
		Metadata: make(map[string]string),
		History:  make([]StateChange, 0),
	}
}

// SetContext sets a context value (thread-safe)
func (s *SharedState) SetContext(key string, value interface{}, agent string) error {
	if key == "" {
		return NewTeamError("SharedState", "SetContext", "key cannot be empty", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.Context[key] = value
	s.addHistoryChange(agent, "set_context", key, value)
	return nil
}

// GetContext gets a context value (thread-safe)
func (s *SharedState) GetContext(key string) (interface{}, bool) {
	if key == "" {
		return nil, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.Context[key]
	return value, exists
}

// SetResult sets an agent result (thread-safe)
func (s *SharedState) SetResult(stepName string, result *AgentResult) error {
	if stepName == "" {
		return NewTeamError("SharedState", "SetResult", "step name cannot be empty", nil)
	}
	if result == nil {
		return NewTeamError("SharedState", "SetResult", "result cannot be nil", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.Results[stepName] = result
	s.addHistoryChange(result.AgentName, "set_result", stepName, result)
	return nil
}

// GetResult gets an agent result (thread-safe)
func (s *SharedState) GetResult(stepName string) (*AgentResult, bool) {
	if stepName == "" {
		return nil, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result, exists := s.Results[stepName]
	return result, exists
}

// GetAllResults returns all results (thread-safe copy)
func (s *SharedState) GetAllResults() map[string]*AgentResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make(map[string]*AgentResult)
	for k, v := range s.Results {
		results[k] = v
	}
	return results
}

// ClearHistory clears old history entries to prevent memory leaks
func (s *SharedState) ClearHistory() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.History) > MaxHistorySize {
		// Keep only the most recent entries
		s.History = s.History[len(s.History)-MaxHistorySize/2:]
	}
}

// addHistoryChange adds a change to the history (assumes lock is held)
func (s *SharedState) addHistoryChange(agent, action, key string, value interface{}) {
	change := StateChange{
		Timestamp: time.Now(),
		Agent:     agent,
		Action:    action,
		Key:       key,
		Value:     value,
	}
	s.History = append(s.History, change)

	// Automatically clear history if it gets too large
	if len(s.History) > MaxHistorySize {
		s.History = s.History[MaxHistorySize/4:]
	}
}

// ============================================================================
// AGENT REGISTRY ALIAS - USE AGENT MODULE'S REGISTRY
// ============================================================================

// ============================================================================
// TEAM - ENHANCED WITH BETTER ERROR HANDLING
// ============================================================================

// Team orchestrates multiple agents in a workflow - CLEAN ARCHITECTURE WITH SERVICES
type Team struct {
	// Core Identity
	name        string
	description string

	// Core Services - Following Agent Pattern
	workflowService     *TeamWorkflowService
	agentService        *TeamAgentService
	coordinationService *TeamCoordinationService

	// Configuration
	workflow         *config.WorkflowConfig
	globalConfig     *config.Config // Global configuration with all agents, LLMs, etc.
	componentManager *component.ComponentManager

	// State Management
	status    WorkflowStatus
	startTime time.Time
	endTime   time.Time
	mu        sync.RWMutex
	errors    []error
}

// NewTeam creates a new team using services - FOLLOWING AGENT PATTERN
func NewTeam(workflowConfig *config.WorkflowConfig, globalConfig *config.Config, componentManager *component.ComponentManager) (*Team, error) {
	if workflowConfig == nil {
		return nil, NewTeamError("Team", "NewTeam", "workflow cannot be nil", nil)
	}
	if globalConfig == nil {
		return nil, NewTeamError("Team", "NewTeam", "global config cannot be nil", nil)
	}
	if componentManager == nil {
		return nil, NewTeamError("Team", "NewTeam", "component manager cannot be nil", nil)
	}

	team := &Team{
		name:             workflowConfig.Name,
		description:      workflowConfig.Description,
		workflow:         workflowConfig,
		globalConfig:     globalConfig,
		componentManager: componentManager,
		status:           WorkflowStatusPending,
		errors:           make([]error, 0),
	}

	// Initialize services - following agent pattern
	team.agentService = NewTeamAgentService(componentManager)
	team.coordinationService = NewTeamCoordinationService()

	// Initialize workflow services
	workflowRegistry := workflow.NewWorkflowExecutorRegistry()
	workflowFactory := workflow.NewWorkflowExecutorFactory()
	team.workflowService = NewTeamWorkflowService(workflowRegistry, workflowFactory)

	// Register default workflow executors
	if err := team.initializeWorkflowExecutors(workflowRegistry, workflowFactory); err != nil {
		return nil, NewTeamError("Team", "NewTeam", "failed to initialize workflow executors", err)
	}

	return team, nil
}

// initializeWorkflowExecutors registers default workflow executors
func (t *Team) initializeWorkflowExecutors(registry *workflow.WorkflowExecutorRegistry, factory workflow.WorkflowExecutorFactory) error {
	// Register DAG executor
	dagExecutor := workflow.NewDAGExecutor(*t.workflow)
	if err := registry.RegisterExecutor(dagExecutor); err != nil {
		return fmt.Errorf("failed to register DAG executor: %w", err)
	}

	// Register Autonomous executor
	autonomousExecutor := workflow.NewAutonomousExecutor(*t.workflow)
	if err := registry.RegisterExecutor(autonomousExecutor); err != nil {
		return fmt.Errorf("failed to register autonomous executor: %w", err)
	}

	return nil
}

// Initialize sets up the team with agents - USING SERVICES
func (t *Team) Initialize(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.workflow.Agents) == 0 {
		return NewTeamError("Team", "Initialize", "no agents defined in workflow", nil)
	}

	// Load and register agents from workflow definition using agent service
	for _, agentName := range t.workflow.Agents {
		if agentName == "" {
			continue // Skip empty agent names
		}

		// Use agent configuration from global config
		var agentConfig *config.AgentConfig
		if cfg, exists := t.globalConfig.Agents[agentName]; exists {
			// Found agent in global config - use it
			agentConfig = &cfg
		} else {
			// Agent not found in config - create default
			return NewTeamError("Team", "Initialize",
				fmt.Sprintf("agent '%s' not found in configuration", agentName), nil)
		}

		// Create and register agent using agent service
		_, err := t.agentService.CreateAndRegisterAgent(agentName, agentConfig)
		if err != nil {
			t.errors = append(t.errors, NewTeamError("Team", "Initialize",
				fmt.Sprintf("failed to create and register agent %s", agentName), err))
			continue
		}
	}

	return nil
}

// createDefaultAgentConfig creates a default agent configuration
func (t *Team) createDefaultAgentConfig(agentName string) *config.AgentConfig {
	return &config.AgentConfig{
		Name:        agentName,
		Description: fmt.Sprintf("Auto-generated agent for %s", agentName),
		LLM:         DefaultAgentLLM,
		Database:    DefaultAgentDatabase,
		Embedder:    DefaultAgentEmbedder,
		Prompt: config.PromptConfig{
			IncludeContext: true,
			IncludeHistory: true,
			IncludeTools:   true,
		},
		Reasoning: config.ReasoningConfig{
			Engine: "dynamic",
		},
		Search: config.SearchConfig{
			TopK: 10,
		},
		Tools: config.ToolConfigs{},
	}
}

// getDefaultCapabilities returns default capabilities for agents
func (t *Team) getDefaultCapabilities() []string {
	return []string{"general", "reasoning", "search", "tools"}
}

// ExecuteStreaming runs the workflow with real-time event streaming
func (t *Team) ExecuteStreaming(ctx context.Context, input string) (<-chan workflow.WorkflowEvent, error) {
	if input == "" {
		errCh := make(chan workflow.WorkflowEvent, 1)
		errCh <- workflow.WorkflowEvent{
			Timestamp: time.Now(),
			EventType: workflow.EventAgentError,
			Content:   "Input cannot be empty",
		}
		close(errCh)
		return errCh, NewTeamError("Team", "ExecuteStreaming", "input cannot be empty", nil)
	}

	eventCh := make(chan workflow.WorkflowEvent, 100)

	go func() {
		defer close(eventCh)

		t.mu.Lock()
		t.status = WorkflowStatusRunning
		t.startTime = time.Now()
		t.mu.Unlock()

		// Send workflow start event
		eventCh <- workflow.WorkflowEvent{
			Timestamp: time.Now(),
			EventType: workflow.EventWorkflowStart,
			Content:   fmt.Sprintf("🚀 Starting workflow: %s", t.workflow.Name),
			Metadata: map[string]string{
				"workflow_name": t.workflow.Name,
				"mode":          string(t.workflow.Mode),
			},
		}

		// Set initial context
		if err := t.coordinationService.SetContext("user_input", input, "system"); err != nil {
			eventCh <- workflow.WorkflowEvent{
				Timestamp: time.Now(),
				EventType: workflow.EventAgentError,
				Content:   fmt.Sprintf("Failed to set context: %v", err),
			}
			return
		}

		// Create workflow request
		request := &workflow.WorkflowRequest{
			Workflow:      t.workflow,
			AgentServices: t.agentService,
			Input:         input,
			Context: workflow.WorkflowContext{
				Variables: make(map[string]string),
				Metadata:  make(map[string]string),
				Artifacts: make(map[string]workflow.Artifact),
			},
		}

		// Execute workflow with streaming
		workflowEventCh, err := t.workflowService.ExecuteWorkflowStreaming(ctx, request)
		if err != nil {
			t.mu.Lock()
			t.status = WorkflowStatusFailed
			t.mu.Unlock()

			eventCh <- workflow.WorkflowEvent{
				Timestamp: time.Now(),
				EventType: workflow.EventWorkflowEnd,
				Content:   fmt.Sprintf("❌ Workflow failed: %v", err),
				Metadata: map[string]string{
					"status": "failed",
					"error":  err.Error(),
				},
			}
			return
		}

		// Forward all workflow events
		for event := range workflowEventCh {
			eventCh <- event
		}

		t.mu.Lock()
		t.endTime = time.Now()
		if t.status == WorkflowStatusRunning {
			t.status = WorkflowStatusCompleted
		}
		t.mu.Unlock()
	}()

	return eventCh, nil
}

// GetStatus returns the current workflow status
func (t *Team) GetStatus() WorkflowStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// GetSharedState returns the shared state (read-only access) - USING COORDINATION SERVICE
func (t *Team) GetSharedState() *SharedState {
	return t.coordinationService.GetSharedState()
}

// GetWorkflow returns the workflow configuration
func (t *Team) GetWorkflow() *config.WorkflowConfig {
	return t.workflow
}

// GetAgent retrieves an agent by name using agent service
func (t *Team) GetAgent(name string) *hectoragent.Agent {
	agent, err := t.agentService.GetAgent(name)
	if err != nil {
		// Log error but don't panic
		t.mu.Lock()
		t.errors = append(t.errors, err)
		t.mu.Unlock()
		return nil
	}
	return agent
}

// GetAgents returns all registered agents using agent service
func (t *Team) GetAgents() map[string]*hectoragent.Agent {
	return t.agentService.GetAllAgents()
}

// GetErrors returns all accumulated errors
func (t *Team) GetErrors() []error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	errors := make([]error, len(t.errors))
	copy(errors, t.errors)
	return errors
}

// Helper methods

// ClearErrors clears all accumulated errors
func (t *Team) ClearErrors() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.errors = make([]error, 0)
}

// ============================================================================
// WORKFLOW LOADING - ENHANCED WITH PROPER ERROR HANDLING
// ============================================================================

// WorkflowLoader handles loading and validation of workflow definitions
type WorkflowLoader struct{}

// NewWorkflowLoader creates a new workflow loader
func NewWorkflowLoader() *WorkflowLoader {
	return &WorkflowLoader{}
}

// LoadWorkflowDefinition loads a workflow definition from a YAML file
func (wl *WorkflowLoader) LoadWorkflowDefinition(filePath string) (*config.WorkflowConfig, error) {
	if filePath == "" {
		return nil, NewTeamError("WorkflowLoader", "LoadWorkflowDefinition", "file path cannot be empty", nil)
	}

	// Load the unified config
	hectorConfig, err := config.LoadConfig(filePath)
	if err != nil {
		return nil, NewTeamError("WorkflowLoader", "LoadWorkflowDefinition",
			"failed to load config", err)
	}

	// Extract the first workflow from the config
	for name, workflowConfig := range hectorConfig.Workflows {
		// Validate the workflow
		if err := wl.validateWorkflow(&workflowConfig); err != nil {
			return nil, NewTeamError("WorkflowLoader", "LoadWorkflowDefinition",
				fmt.Sprintf("workflow validation failed for %s", name), err)
		}
		return &workflowConfig, nil
	}

	return nil, NewTeamError("WorkflowLoader", "LoadWorkflowDefinition",
		"no workflows found in config file", nil)
}

// validateWorkflow validates a workflow configuration
func (wl *WorkflowLoader) validateWorkflow(workflowConfig *config.WorkflowConfig) error {
	if workflowConfig.Name == "" {
		return NewTeamError("WorkflowLoader", "validateWorkflow", "workflow name is required", nil)
	}

	if len(workflowConfig.Agents) == 0 {
		return NewTeamError("WorkflowLoader", "validateWorkflow", "workflow must have at least one agent", nil)
	}

	// Validate agents are not empty
	for i, agent := range workflowConfig.Agents {
		if agent == "" {
			return NewTeamError("WorkflowLoader", "validateWorkflow",
				fmt.Sprintf("agent at index %d cannot be empty", i), nil)
		}
	}

	return nil
}
