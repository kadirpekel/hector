package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// DAG EXECUTOR IMPLEMENTATION
// ============================================================================

// DAGExecutor implements WorkflowExecutor for DAG-based workflows
type DAGExecutor struct {
	*BaseExecutor
	config config.WorkflowConfig
}

// NewDAGExecutor creates a new DAG executor
func NewDAGExecutor(config config.WorkflowConfig) WorkflowExecutor {
	capabilities := ExecutorCapabilities{
		SupportsParallelExecution: true,
		SupportsDynamicPlanning:   false,
		SupportsStaticWorkflows:   true,
		RequiredFeatures:          []string{"dag", "dependencies"},
		MaxConcurrency:            4,
		SupportsRollback:          true,
	}

	return &DAGExecutor{
		BaseExecutor: NewBaseExecutor("dag", "dag", capabilities),
		config:       config,
	}
}

// Execute runs the DAG workflow using AgentServices abstraction
func (e *DAGExecutor) Execute(ctx context.Context, request *WorkflowRequest) (*WorkflowResult, error) {
	if request == nil {
		return nil, fmt.Errorf("workflow request cannot be nil")
	}

	if request.Workflow == nil {
		return nil, fmt.Errorf("workflow config cannot be nil")
	}

	if request.AgentServices == nil {
		return nil, fmt.Errorf("agent services cannot be nil")
	}

	// Create execution context
	execCtx := NewExecutionContext(request)
	execCtx.SetStatus(StatusExecuting)

	startTime := time.Now()

	// Example DAG execution using AgentServices abstraction
	results := make(map[string]*AgentResult)

	// Get available agents from the abstract service
	availableAgents := request.AgentServices.GetAvailableAgents()
	if len(availableAgents) == 0 {
		return nil, fmt.Errorf("no agents available for execution")
	}

	// Execute agents sequentially for now (proper DAG logic would handle dependencies)
	for _, agentName := range request.Workflow.Agents {
		if !request.AgentServices.IsAgentAvailable(agentName) {
			return nil, fmt.Errorf("agent %s is not available", agentName)
		}

		// Execute agent using the abstract service
		result, err := request.AgentServices.ExecuteAgent(ctx, agentName, request.Input)
		if err != nil {
			return nil, fmt.Errorf("failed to execute agent %s: %w", agentName, err)
		}

		results[agentName] = result
	}

	// Create final workflow result
	workflowResult := CreateWorkflowResult(
		request.Workflow,
		WorkflowStatusCompleted,
		results,
		execCtx.GetAllSharedState(),
		time.Since(startTime),
		execCtx.GetErrors(),
	)

	return workflowResult, nil
}

// CanHandle returns true if this executor can handle the given workflow
func (e *DAGExecutor) CanHandle(workflow *config.WorkflowConfig) bool {
	return workflow.Mode == config.ExecutionModeDAG
}

// ============================================================================
// AUTONOMOUS EXECUTOR IMPLEMENTATION
// ============================================================================

// AutonomousExecutor implements WorkflowExecutor for autonomous workflows
type AutonomousExecutor struct {
	*BaseExecutor
	config config.WorkflowConfig
}

// NewAutonomousExecutor creates a new autonomous executor
func NewAutonomousExecutor(config config.WorkflowConfig) WorkflowExecutor {
	capabilities := ExecutorCapabilities{
		SupportsParallelExecution: true,
		SupportsDynamicPlanning:   true,
		SupportsStaticWorkflows:   false,
		RequiredFeatures:          []string{"autonomous", "planning", "coordination"},
		MaxConcurrency:            8,
		SupportsRollback:          false,
	}

	return &AutonomousExecutor{
		BaseExecutor: NewBaseExecutor("autonomous", "autonomous", capabilities),
		config:       config,
	}
}

// Execute runs the autonomous workflow using AgentServices abstraction
func (e *AutonomousExecutor) Execute(ctx context.Context, request *WorkflowRequest) (*WorkflowResult, error) {
	if request == nil {
		return nil, fmt.Errorf("workflow request cannot be nil")
	}

	if request.Workflow == nil {
		return nil, fmt.Errorf("workflow config cannot be nil")
	}

	if request.AgentServices == nil {
		return nil, fmt.Errorf("agent services cannot be nil")
	}

	// Create execution context
	execCtx := NewExecutionContext(request)
	execCtx.SetStatus(StatusExecuting)

	startTime := time.Now()

	// Example autonomous execution using AgentServices abstraction
	results := make(map[string]*AgentResult)

	// Get available agents and their capabilities
	availableAgents := request.AgentServices.GetAvailableAgents()
	if len(availableAgents) == 0 {
		return nil, fmt.Errorf("no agents available for autonomous execution")
	}

	// Autonomous logic: dynamically select and execute agents based on capabilities
	// For now, execute all available agents (real implementation would use planning)
	for _, agentName := range availableAgents {
		capabilities, err := request.AgentServices.GetAgentCapabilities(agentName)
		if err != nil {
			continue // Skip agents we can't get capabilities for
		}

		// Simple capability check - execute agents with "general" capability
		hasGeneralCapability := false
		for _, cap := range capabilities {
			if cap == "general" {
				hasGeneralCapability = true
				break
			}
		}

		if hasGeneralCapability {
			result, err := request.AgentServices.ExecuteAgent(ctx, agentName, request.Input)
			if err != nil {
				// In autonomous mode, continue with other agents on failure
				continue
			}
			results[agentName] = result
		}
	}

	// Create final workflow result
	workflowResult := CreateWorkflowResult(
		request.Workflow,
		WorkflowStatusCompleted,
		results,
		execCtx.GetAllSharedState(),
		time.Since(startTime),
		execCtx.GetErrors(),
	)

	return workflowResult, nil
}

// CanHandle returns true if this executor can handle the given workflow
func (e *AutonomousExecutor) CanHandle(workflow *config.WorkflowConfig) bool {
	return workflow.Mode == config.ExecutionModeAutonomous
}
