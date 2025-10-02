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

// ExecuteStreaming runs the DAG workflow with real-time event streaming
func (e *DAGExecutor) ExecuteStreaming(ctx context.Context, request *WorkflowRequest) (<-chan WorkflowEvent, error) {
	if request == nil {
		return nil, fmt.Errorf("workflow request cannot be nil")
	}

	if request.Workflow == nil {
		return nil, fmt.Errorf("workflow config cannot be nil")
	}

	if request.AgentServices == nil {
		return nil, fmt.Errorf("agent services cannot be nil")
	}

	eventCh := make(chan WorkflowEvent, 100)

	go func() {
		defer close(eventCh)

		// Create execution context
		execCtx := NewExecutionContext(request)
		execCtx.SetStatus(StatusExecuting)

		startTime := time.Now()
		results := make(map[string]*AgentResult)

		// Get available agents
		availableAgents := request.AgentServices.GetAvailableAgents()
		if len(availableAgents) == 0 {
			eventCh <- WorkflowEvent{
				Timestamp: time.Now(),
				EventType: EventAgentError,
				Content:   "No agents available for execution",
			}
			return
		}

		totalSteps := len(request.Workflow.Agents)
		completedSteps := 0

		// Execute agents sequentially (proper DAG would handle dependencies)
		for i, agentName := range request.Workflow.Agents {
			if !request.AgentServices.IsAgentAvailable(agentName) {
				eventCh <- WorkflowEvent{
					Timestamp: time.Now(),
					EventType: EventAgentError,
					AgentName: agentName,
					Content:   fmt.Sprintf("Agent %s is not available", agentName),
				}
				continue
			}

			// Send progress event
			eventCh <- WorkflowEvent{
				Timestamp: time.Now(),
				EventType: EventProgress,
				StepName:  agentName,
				Content:   fmt.Sprintf("Step %d/%d: %s", i+1, totalSteps, agentName),
				Progress: &WorkflowProgress{
					TotalSteps:      totalSteps,
					CompletedSteps:  completedSteps,
					CurrentStep:     agentName,
					PercentComplete: float64(completedSteps) / float64(totalSteps) * 100,
				},
			}

			// Execute agent with streaming
			result, err := request.AgentServices.ExecuteAgentStreaming(ctx, agentName, request.Input, eventCh)
			if err != nil {
				eventCh <- WorkflowEvent{
					Timestamp: time.Now(),
					EventType: EventAgentError,
					AgentName: agentName,
					Content:   fmt.Sprintf("Failed to execute agent: %v", err),
				}
				continue
			}

			results[agentName] = result
			execCtx.SetResult(agentName, result)
			completedSteps++
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

		// Send final completion event
		eventCh <- WorkflowEvent{
			Timestamp: time.Now(),
			EventType: EventWorkflowEnd,
			Content:   workflowResult.FinalOutput,
			Metadata: map[string]string{
				"status":         string(workflowResult.Status),
				"execution_time": workflowResult.ExecutionTime.String(),
				"total_tokens":   fmt.Sprintf("%d", workflowResult.TotalTokens),
				"steps_executed": fmt.Sprintf("%d", workflowResult.StepsExecuted),
			},
		}
	}()

	return eventCh, nil
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

// ExecuteStreaming runs the autonomous workflow with real-time event streaming
func (e *AutonomousExecutor) ExecuteStreaming(ctx context.Context, request *WorkflowRequest) (<-chan WorkflowEvent, error) {
	if request == nil {
		return nil, fmt.Errorf("workflow request cannot be nil")
	}

	if request.Workflow == nil {
		return nil, fmt.Errorf("workflow config cannot be nil")
	}

	if request.AgentServices == nil {
		return nil, fmt.Errorf("agent services cannot be nil")
	}

	eventCh := make(chan WorkflowEvent, 100)

	go func() {
		defer close(eventCh)

		// Create execution context
		execCtx := NewExecutionContext(request)
		execCtx.SetStatus(StatusExecuting)

		startTime := time.Now()
		results := make(map[string]*AgentResult)

		// Get available agents and their capabilities
		availableAgents := request.AgentServices.GetAvailableAgents()
		if len(availableAgents) == 0 {
			eventCh <- WorkflowEvent{
				Timestamp: time.Now(),
				EventType: EventAgentError,
				Content:   "No agents available for autonomous execution",
			}
			return
		}

		// Send planning event
		eventCh <- WorkflowEvent{
			Timestamp: time.Now(),
			EventType: EventProgress,
			Content:   fmt.Sprintf("Autonomous planning: analyzing %d available agents", len(availableAgents)),
		}

		executedCount := 0

		// Autonomous logic: dynamically select and execute agents based on capabilities
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
				// Execute agent with streaming
				result, err := request.AgentServices.ExecuteAgentStreaming(ctx, agentName, request.Input, eventCh)
				if err != nil {
					// In autonomous mode, log but continue with other agents
					eventCh <- WorkflowEvent{
						Timestamp: time.Now(),
						EventType: EventAgentError,
						AgentName: agentName,
						Content:   fmt.Sprintf("Agent failed (continuing): %v", err),
					}
					continue
				}
				results[agentName] = result
				execCtx.SetResult(agentName, result)
				executedCount++
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

		// Send final completion event
		eventCh <- WorkflowEvent{
			Timestamp: time.Now(),
			EventType: EventWorkflowEnd,
			Content:   fmt.Sprintf("Autonomous execution completed: %d agents executed\n%s", executedCount, workflowResult.FinalOutput),
			Metadata: map[string]string{
				"status":          string(workflowResult.Status),
				"execution_time":  workflowResult.ExecutionTime.String(),
				"total_tokens":    fmt.Sprintf("%d", workflowResult.TotalTokens),
				"agents_executed": fmt.Sprintf("%d", executedCount),
			},
		}
	}()

	return eventCh, nil
}

// CanHandle returns true if this executor can handle the given workflow
func (e *AutonomousExecutor) CanHandle(workflow *config.WorkflowConfig) bool {
	return workflow.Mode == config.ExecutionModeAutonomous
}
