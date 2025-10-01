package workflow

import (
	"fmt"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// CONCRETE FACTORY IMPLEMENTATION
// ============================================================================

// DefaultWorkflowExecutorFactory is the default implementation of WorkflowExecutorFactory
type DefaultWorkflowExecutorFactory struct{}

// NewWorkflowExecutorFactory creates a new workflow executor factory
func NewWorkflowExecutorFactory() WorkflowExecutorFactory {
	return &DefaultWorkflowExecutorFactory{}
}

// CreateExecutor creates a workflow executor of the specified type
func (f *DefaultWorkflowExecutorFactory) CreateExecutor(executorType string, config config.WorkflowConfig) (WorkflowExecutor, error) {
	switch executorType {
	case "dag":
		return NewDAGExecutor(config), nil
	case "autonomous":
		return NewAutonomousExecutor(config), nil
	case "sequential":
		// Future: Sequential execution
		return nil, fmt.Errorf("sequential executor not implemented yet")
	case "parallel":
		// Future: Parallel execution
		return nil, fmt.Errorf("parallel executor not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported workflow executor type: %s", executorType)
	}
}

// GetSupportedType returns the supported executor type
func (f *DefaultWorkflowExecutorFactory) GetSupportedType() string {
	return "default"
}

// GetSupportedModes returns the supported execution modes
func (f *DefaultWorkflowExecutorFactory) GetSupportedModes() []string {
	return []string{"dag", "autonomous"}
}

// ListAvailableExecutors returns information about all available workflow executors
func (f *DefaultWorkflowExecutorFactory) ListAvailableExecutors() []WorkflowExecutorInfo {
	return []WorkflowExecutorInfo{
		{
			Name:        "dag",
			Type:        "dag",
			Description: "Directed Acyclic Graph executor for structured workflows with dependencies",
			Features: []string{
				"Step-by-step execution",
				"Dependency management",
				"Parallel step execution",
				"Error handling and rollback",
				"Progress tracking",
			},
			Parameters: []WorkflowExecutorParameter{
				{
					Name:        "max_concurrency",
					Type:        "int",
					Description: "Maximum number of concurrent steps",
					Required:    false,
					Default:     2,
				},
				{
					Name:        "timeout",
					Type:        "duration",
					Description: "Maximum execution time for the workflow",
					Required:    false,
					Default:     "20m",
				},
			},
			Examples: []WorkflowExecutorExample{
				{
					Name:        "Research Analysis Workflow",
					Description: "Multi-step research and analysis workflow",
					Config: config.WorkflowConfig{
						Name: "research-analysis",
						Mode: "dag",
					},
					Input: "analyze the current market trends",
				},
			},
			Modes: []string{"dag"},
		},
		{
			Name:        "autonomous",
			Type:        "autonomous",
			Description: "Autonomous executor for dynamic, self-organizing workflows",
			Features: []string{
				"Dynamic planning",
				"Self-organizing execution",
				"Adaptive goal pursuit",
				"Real-time coordination",
				"Intelligent agent selection",
			},
			Parameters: []WorkflowExecutorParameter{
				{
					Name:        "max_iterations",
					Type:        "int",
					Description: "Maximum number of planning iterations",
					Required:    false,
					Default:     10,
				},
				{
					Name:        "coordinator_llm",
					Type:        "string",
					Description: "LLM to use for coordination",
					Required:    true,
				},
				{
					Name:        "quality_threshold",
					Type:        "float",
					Description: "Quality threshold for completion",
					Required:    false,
					Default:     0.8,
				},
			},
			Examples: []WorkflowExecutorExample{
				{
					Name:        "Dynamic Problem Solving",
					Description: "Autonomous problem-solving workflow",
					Config: config.WorkflowConfig{
						Name: "autonomous-solver",
						Mode: "autonomous",
					},
					Input: "solve this complex business problem",
				},
			},
			Modes: []string{"autonomous"},
		},
		// Future executors can be added here
	}
}
