// Package hector provides AI agent team coordination and workflow orchestration.
// Team manages multi-agent workflows with DAG and autonomous execution.
package hector

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// CORE TYPES AND ENUMS
// ============================================================================

// WorkflowStatus represents the current state of a workflow execution
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

// StepStatus represents the execution status of a workflow step
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusReady     StepStatus = "ready"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

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

	// AgentHealthCheckTimeout is the timeout for agent health checks
	AgentHealthCheckTimeout = 30 * time.Second
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

// AgentResult represents the output from an agent execution
type AgentResult struct {
	AgentName  string                 `json:"agent_name"`
	StepName   string                 `json:"step_name"`
	Output     string                 `json:"output"`
	Artifacts  map[string]interface{} `json:"artifacts"`
	NextAgents []string               `json:"next_agents,omitempty"`
	Confidence float64                `json:"confidence"`
	Duration   time.Duration          `json:"duration"`
	TokensUsed int                    `json:"tokens_used"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]string      `json:"metadata"`
	Timestamp  time.Time              `json:"timestamp"`
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
// AGENT REGISTRY - ENHANCED WITH PROPER ERROR HANDLING
// ============================================================================

// AgentRegistry manages a pool of agents and their capabilities
type AgentRegistry struct {
	mu              sync.RWMutex
	agents          map[string]*Agent              // agent_name -> agent instance
	definitions     map[string]*config.AgentConfig // agent_name -> agent config
	capabilities    map[string][]string            // agent_name -> capabilities list
	instances       map[string][]*Agent            // agent_type -> instance pool
	health          map[string]bool                // agent_name -> health status
	lastHealthCheck map[string]time.Time           // agent_name -> last health check time
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents:          make(map[string]*Agent),
		definitions:     make(map[string]*config.AgentConfig),
		capabilities:    make(map[string][]string),
		instances:       make(map[string][]*Agent),
		health:          make(map[string]bool),
		lastHealthCheck: make(map[string]time.Time),
	}
}

// RegisterAgent registers an agent with the registry
func (r *AgentRegistry) RegisterAgent(name string, agent *Agent, agentConfig *config.AgentConfig, capabilities []string) error {
	if name == "" {
		return NewTeamError("AgentRegistry", "RegisterAgent", "agent name cannot be empty", nil)
	}
	if agent == nil {
		return NewTeamError("AgentRegistry", "RegisterAgent", "agent cannot be nil", nil)
	}
	if agentConfig == nil {
		return NewTeamError("AgentRegistry", "RegisterAgent", "agent config cannot be nil", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[name]; exists {
		return NewTeamError("AgentRegistry", "RegisterAgent",
			fmt.Sprintf("agent %s already registered", name), nil)
	}

	r.agents[name] = agent
	r.definitions[name] = agentConfig
	r.capabilities[name] = capabilities
	r.health[name] = true
	r.lastHealthCheck[name] = time.Now()

	// Add to instance pool by agent type
	agentType := r.extractAgentType(name)
	if r.instances[agentType] == nil {
		r.instances[agentType] = make([]*Agent, 0)
	}
	r.instances[agentType] = append(r.instances[agentType], agent)

	return nil
}

// extractAgentType extracts the agent type from the agent name
func (r *AgentRegistry) extractAgentType(name string) string {
	// Extract agent type from instance name (e.g., "researcher_0" -> "researcher")
	if underscoreIndex := strings.LastIndex(name, "_"); underscoreIndex > 0 {
		return name[:underscoreIndex]
	}
	return name
}

// GetAgent retrieves an agent by name
func (r *AgentRegistry) GetAgent(name string) (*Agent, error) {
	if name == "" {
		return nil, NewTeamError("AgentRegistry", "GetAgent", "agent name cannot be empty", nil)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[name]
	if !exists {
		return nil, NewTeamError("AgentRegistry", "GetAgent",
			fmt.Sprintf("agent %s not found", name), nil)
	}

	if !r.health[name] {
		return nil, NewTeamError("AgentRegistry", "GetAgent",
			fmt.Sprintf("agent %s is unhealthy", name), nil)
	}

	return agent, nil
}

// GetAgentByType retrieves an available agent instance by type (load balancing)
func (r *AgentRegistry) GetAgentByType(agentType string) (*Agent, error) {
	if agentType == "" {
		return nil, NewTeamError("AgentRegistry", "GetAgentByType", "agent type cannot be empty", nil)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	instances, exists := r.instances[agentType]
	if !exists || len(instances) == 0 {
		return nil, NewTeamError("AgentRegistry", "GetAgentByType",
			fmt.Sprintf("no instances available for agent type %s", agentType), nil)
	}

	// Simple round-robin load balancing - return first healthy instance
	for _, agent := range instances {
		// Find the agent name for this instance
		for name, a := range r.agents {
			if a == agent && r.health[name] {
				return agent, nil
			}
		}
	}

	return nil, NewTeamError("AgentRegistry", "GetAgentByType",
		fmt.Sprintf("no healthy instances available for agent type %s", agentType), nil)
}

// GetAgentsByCapability retrieves agents that have a specific capability
func (r *AgentRegistry) GetAgentsByCapability(capability string) ([]*Agent, error) {
	if capability == "" {
		return nil, NewTeamError("AgentRegistry", "GetAgentsByCapability", "capability cannot be empty", nil)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []*Agent
	for name, caps := range r.capabilities {
		for _, cap := range caps {
			if cap == capability && r.health[name] {
				if agent, exists := r.agents[name]; exists {
					agents = append(agents, agent)
				}
				break
			}
		}
	}

	return agents, nil
}

// ListAgents returns all registered agent names
func (r *AgentRegistry) ListAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}

	return names
}

// SetAgentHealth updates the health status of an agent
func (r *AgentRegistry) SetAgentHealth(name string, healthy bool) error {
	if name == "" {
		return NewTeamError("AgentRegistry", "SetAgentHealth", "agent name cannot be empty", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[name]; !exists {
		return NewTeamError("AgentRegistry", "SetAgentHealth",
			fmt.Sprintf("agent %s not found", name), nil)
	}

	r.health[name] = healthy
	r.lastHealthCheck[name] = time.Now()
	return nil
}

// PerformHealthCheck performs health checks on all agents
func (r *AgentRegistry) PerformHealthCheck(ctx context.Context) map[string]error {
	r.mu.RLock()
	agents := make(map[string]*Agent)
	for name, agent := range r.agents {
		agents[name] = agent
	}
	r.mu.RUnlock()

	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, agent := range agents {
		wg.Add(1)
		go func(agentName string, a *Agent) {
			defer wg.Done()

			// Simple health check - try to get agent config
			healthy := true
			var err error

			// Check if agent is responsive (this is a simple check)
			if a.GetConfig() == nil {
				healthy = false
				err = NewTeamError("AgentRegistry", "PerformHealthCheck",
					"agent config is nil", nil)
			}

			mu.Lock()
			if err != nil {
				results[agentName] = err
			}
			r.SetAgentHealth(agentName, healthy)
			mu.Unlock()
		}(name, agent)
	}

	wg.Wait()
	return results
}

// ============================================================================
// TEAM - ENHANCED WITH BETTER ERROR HANDLING
// ============================================================================

// Team orchestrates multiple agents in a workflow
type Team struct {
	registry         *AgentRegistry
	executorRegistry interfaces.ExecutorRegistry
	sharedState      *SharedState
	workflow         *config.WorkflowConfig
	status           WorkflowStatus
	startTime        time.Time
	endTime          time.Time
	mu               sync.RWMutex
	errors           []error
}

// NewTeam creates a new team
func NewTeam(workflow *config.WorkflowConfig) (*Team, error) {
	if workflow == nil {
		return nil, NewTeamError("Team", "NewTeam", "workflow cannot be nil", nil)
	}

	return &Team{
		registry:         NewAgentRegistry(),
		executorRegistry: nil, // Will be set during initialization
		sharedState:      NewSharedState(),
		workflow:         workflow,
		status:           WorkflowStatusPending,
		errors:           make([]error, 0),
	}, nil
}

// Initialize sets up the team with agents
func (t *Team) Initialize(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.workflow.Agents) == 0 {
		return NewTeamError("Team", "Initialize", "no agents defined in workflow", nil)
	}

	// Load and register agents from workflow definition
	for _, agentName := range t.workflow.Agents {
		if agentName == "" {
			continue // Skip empty agent names
		}

		// Create agent configuration with proper defaults
		agentConfig := t.createDefaultAgentConfig(agentName)

		// Create agent instance
		agent, err := NewAgent(agentConfig, nil)
		if err != nil {
			t.errors = append(t.errors, NewTeamError("Team", "Initialize",
				fmt.Sprintf("failed to create agent instance for %s", agentName), err))
			continue
		}

		// Register the agent
		err = t.registry.RegisterAgent(agentName, agent, agentConfig, t.getDefaultCapabilities())
		if err != nil {
			t.errors = append(t.errors, NewTeamError("Team", "Initialize",
				fmt.Sprintf("failed to register agent %s", agentName), err))
			continue
		}
	}

	// Perform initial health check
	healthResults := t.registry.PerformHealthCheck(ctx)
	for agentName, err := range healthResults {
		if err != nil {
			t.errors = append(t.errors, NewTeamError("Team", "Initialize",
				fmt.Sprintf("health check failed for agent %s", agentName), err))
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

// Execute runs the workflow based on the execution mode
func (t *Team) Execute(ctx context.Context, input string) (*WorkflowResult, error) {
	if input == "" {
		return nil, NewTeamError("Team", "Execute", "input cannot be empty", nil)
	}

	t.mu.Lock()
	t.status = WorkflowStatusRunning
	t.startTime = time.Now()
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		t.endTime = time.Now()
		if t.status == WorkflowStatusRunning {
			t.status = WorkflowStatusCompleted
		}
		t.mu.Unlock()
	}()

	// Set initial context
	if err := t.sharedState.SetContext("user_input", input, "system"); err != nil {
		return nil, NewTeamError("Team", "Execute", "failed to set user input", err)
	}
	if err := t.sharedState.SetContext("workflow_name", t.workflow.Name, "system"); err != nil {
		return nil, NewTeamError("Team", "Execute", "failed to set workflow name", err)
	}

	// Use the executor system (required)
	if t.executorRegistry == nil {
		return nil, NewTeamError("Team", "Execute", "executor registry is required but not initialized", nil)
	}

	result, err := t.executorRegistry.ExecuteWorkflow(ctx, t.workflow, t)
	if err != nil {
		t.mu.Lock()
		t.status = WorkflowStatusFailed
		t.mu.Unlock()
		return nil, NewTeamError("Team", "Execute", "workflow execution failed", err)
	}

	workflowResult, ok := result.(*WorkflowResult)
	if !ok {
		return nil, NewTeamError("Team", "Execute", "invalid result type from executor", nil)
	}

	return workflowResult, nil
}

// GetStatus returns the current workflow status
func (t *Team) GetStatus() WorkflowStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// GetSharedState returns the shared state (read-only access)
func (t *Team) GetSharedState() *SharedState {
	return t.sharedState
}

// GetWorkflow returns the workflow configuration
func (t *Team) GetWorkflow() *config.WorkflowConfig {
	return t.workflow
}

// GetAgent retrieves an agent by name (safe method)
func (t *Team) GetAgent(name string) *Agent {
	agent, err := t.registry.GetAgent(name)
	if err != nil {
		// Log error but don't panic
		t.mu.Lock()
		t.errors = append(t.errors, err)
		t.mu.Unlock()
		return nil
	}
	return agent
}

// GetAgents returns all registered agents
func (t *Team) GetAgents() map[string]*Agent {
	t.registry.mu.RLock()
	defer t.registry.mu.RUnlock()

	agents := make(map[string]*Agent)
	for name, agent := range t.registry.agents {
		if t.registry.health[name] {
			agents[name] = agent
		}
	}

	return agents
}

// GetLLMProvider returns the LLM provider from the first available agent
func (t *Team) GetLLMProvider() (interfaces.LLMInterface, error) {
	agents := t.GetAgents()
	if len(agents) == 0 {
		return nil, NewTeamError("Team", "GetLLMProvider", "no healthy agents available", nil)
	}

	for _, agent := range agents {
		if llm := agent.GetLLM(); llm != nil {
			return llm, nil
		}
	}

	return nil, NewTeamError("Team", "GetLLMProvider", "no LLM provider available from agents", nil)
}

// SetExecutorRegistry sets the executor registry
func (t *Team) SetExecutorRegistry(registry interfaces.ExecutorRegistry) error {
	if registry == nil {
		return NewTeamError("Team", "SetExecutorRegistry", "executor registry cannot be nil", nil)
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.executorRegistry = registry
	return nil
}

// GetExecutorRegistry returns the executor registry
func (t *Team) GetExecutorRegistry() interfaces.ExecutorRegistry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.executorRegistry
}

// GetErrors returns all accumulated errors
func (t *Team) GetErrors() []error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	errors := make([]error, len(t.errors))
	copy(errors, t.errors)
	return errors
}

// ClearErrors clears all accumulated errors
func (t *Team) ClearErrors() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.errors = make([]error, 0)
}

// ============================================================================
// WORKFLOW RESULT
// ============================================================================

// WorkflowResult represents the final result of a multi-agent workflow
type WorkflowResult struct {
	WorkflowName  string                  `json:"workflow_name"`
	Status        WorkflowStatus          `json:"status"`
	FinalOutput   string                  `json:"final_output"`
	AgentResults  map[string]*AgentResult `json:"agent_results"`
	SharedContext map[string]interface{}  `json:"shared_context"`
	ExecutionTime time.Duration           `json:"execution_time"`
	TotalTokens   int                     `json:"total_tokens"`
	StepsExecuted int                     `json:"steps_executed"`
	AgentsUsed    []string                `json:"agents_used"`
	Success       bool                    `json:"success"`
	Error         string                  `json:"error,omitempty"`
	Metadata      map[string]string       `json:"metadata"`
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
	hectorConfig, err := config.LoadHectorConfig(filePath)
	if err != nil {
		return nil, NewTeamError("WorkflowLoader", "LoadWorkflowDefinition",
			"failed to load config", err)
	}

	// Extract the first workflow from the config
	for name, workflow := range hectorConfig.Workflows {
		// Validate the workflow
		if err := wl.validateWorkflow(&workflow); err != nil {
			return nil, NewTeamError("WorkflowLoader", "LoadWorkflowDefinition",
				fmt.Sprintf("workflow validation failed for %s", name), err)
		}
		return &workflow, nil
	}

	return nil, NewTeamError("WorkflowLoader", "LoadWorkflowDefinition",
		"no workflows found in config file", nil)
}

// validateWorkflow validates a workflow configuration
func (wl *WorkflowLoader) validateWorkflow(workflow *config.WorkflowConfig) error {
	if workflow.Name == "" {
		return NewTeamError("WorkflowLoader", "validateWorkflow", "workflow name is required", nil)
	}

	if len(workflow.Agents) == 0 {
		return NewTeamError("WorkflowLoader", "validateWorkflow", "workflow must have at least one agent", nil)
	}

	// Validate agents are not empty
	for i, agent := range workflow.Agents {
		if agent == "" {
			return NewTeamError("WorkflowLoader", "validateWorkflow",
				fmt.Sprintf("agent at index %d cannot be empty", i), nil)
		}
	}

	return nil
}

// LoadWorkflowDefinition is a convenience function for backward compatibility
func LoadWorkflowDefinition(filePath string) (*config.WorkflowConfig, error) {
	loader := NewWorkflowLoader()
	return loader.LoadWorkflowDefinition(filePath)
}
