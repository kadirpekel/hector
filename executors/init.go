package executors

import (
	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// INIT - EXECUTOR SYSTEM INITIALIZATION
// ============================================================================

// InitializeDefaultExecutors creates and registers the default executors
func InitializeDefaultExecutors() *ExecutorRegistry {
	registry := NewExecutorRegistry()

	// Register DAG executor
	dagExecutor := NewDAGExecutor()
	if err := registry.RegisterExecutor(dagExecutor); err != nil {
		// This should not happen with default executors
		panic("Failed to register DAG executor: " + err.Error())
	}

	// Register Autonomous executor
	autonomousExecutor := NewAutonomousExecutor()
	if err := registry.RegisterExecutor(autonomousExecutor); err != nil {
		// This should not happen with default executors
		panic("Failed to register Autonomous executor: " + err.Error())
	}

	return registry
}

// GetExecutorForMode returns the appropriate executor for a given execution mode
func GetExecutorForMode(mode config.ExecutionMode) string {
	switch mode {
	case config.ExecutionModeDAG:
		return "dag"
	case config.ExecutionModeAutonomous:
		return "autonomous"
	default:
		return ""
	}
}

// ValidateExecutorForWorkflow checks if an executor can handle a workflow
func ValidateExecutorForWorkflow(executor Executor, workflow *config.WorkflowConfig) error {
	if !executor.CanHandle(workflow) {
		return &interfaces.ExecutorValidationError{
			ExecutorName: executor.GetName(),
			WorkflowMode: string(workflow.Mode),
			Message:      "executor cannot handle this workflow type",
		}
	}

	capabilities := executor.GetCapabilities()

	// Check required features based on workflow type
	switch workflow.Mode {
	case config.ExecutionModeDAG:
		if !capabilities.SupportsStaticWorkflows {
			return &interfaces.ExecutorValidationError{
				ExecutorName: executor.GetName(),
				WorkflowMode: string(workflow.Mode),
				Message:      "executor does not support static workflows",
			}
		}
		if len(workflow.Execution.DAG.Steps) == 0 {
			return &interfaces.ExecutorValidationError{
				ExecutorName: executor.GetName(),
				WorkflowMode: string(workflow.Mode),
				Message:      "DAG workflow requires at least one step",
			}
		}
	case config.ExecutionModeAutonomous:
		if !capabilities.SupportsDynamicPlanning {
			return &interfaces.ExecutorValidationError{
				ExecutorName: executor.GetName(),
				WorkflowMode: string(workflow.Mode),
				Message:      "executor does not support dynamic planning",
			}
		}
		if workflow.Execution.Autonomous.Goal == "" {
			return &interfaces.ExecutorValidationError{
				ExecutorName: executor.GetName(),
				WorkflowMode: string(workflow.Mode),
				Message:      "autonomous workflow requires autonomous configuration",
			}
		}
	}

	return nil
}
