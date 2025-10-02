package workflow

import (
	"context"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// FOCUSED SERVICE INTERFACES - WHAT TEAM NEEDS FROM WORKFLOW
// ============================================================================

// WorkflowExecutionService defines workflow execution capabilities
type WorkflowExecutionService interface {
	ExecuteWorkflowStreaming(ctx context.Context, request *WorkflowRequest) (<-chan WorkflowEvent, error)
	GetSupportedModes() []string
	CanHandle(workflow *config.WorkflowConfig) bool
}

// WorkflowRegistryService defines workflow executor registry capabilities
type WorkflowRegistryService interface {
	RegisterExecutor(executor WorkflowExecutor) error
	GetExecutor(name string) (WorkflowExecutor, error)
	CreateExecutor(executorType string, workflowConfig config.WorkflowConfig) (WorkflowExecutor, error)
	GetSupportedExecutorsByMode(mode string) []WorkflowExecutor
}

// WorkflowFactoryService defines workflow executor factory capabilities
type WorkflowFactoryService interface {
	CreateExecutor(executorType string, config config.WorkflowConfig) (WorkflowExecutor, error)
	GetSupportedType() string
	GetSupportedModes() []string
	ListAvailableExecutors() []WorkflowExecutorInfo
}

// ============================================================================
// WORKFLOW EXECUTOR CAPABILITIES - WHAT TEAM NEEDS FROM EXECUTORS
// ============================================================================

// WorkflowExecutor represents a workflow execution engine
type WorkflowExecutor interface {
	// GetName returns the executor name (e.g., "dag", "autonomous")
	GetName() string

	// GetType returns the executor type for identification
	GetType() string

	// ExecuteStreaming runs the workflow with real-time event streaming
	ExecuteStreaming(ctx context.Context, request *WorkflowRequest) (<-chan WorkflowEvent, error)

	// CanHandle returns true if this executor can handle the given workflow
	CanHandle(workflow *config.WorkflowConfig) bool

	// GetCapabilities returns the executor's capabilities
	GetCapabilities() ExecutorCapabilities
}

// ExecutorCapabilities describes what an executor can do
type ExecutorCapabilities struct {
	SupportsParallelExecution bool     `json:"supports_parallel_execution"`
	SupportsDynamicPlanning   bool     `json:"supports_dynamic_planning"`
	SupportsStaticWorkflows   bool     `json:"supports_static_workflows"`
	RequiredFeatures          []string `json:"required_features"`
	MaxConcurrency            int      `json:"max_concurrency"`
	SupportsRollback          bool     `json:"supports_rollback"`
}

// ============================================================================
// WORKFLOW EXECUTOR FACTORY - CREATES EXECUTORS WITH DEPENDENCIES
// ============================================================================

// WorkflowExecutorFactory creates workflow executor instances
type WorkflowExecutorFactory interface {
	CreateExecutor(executorType string, config config.WorkflowConfig) (WorkflowExecutor, error)
	GetSupportedType() string
	GetSupportedModes() []string
	ListAvailableExecutors() []WorkflowExecutorInfo
}

// ============================================================================
// WORKFLOW EXECUTOR METADATA
// ============================================================================

// WorkflowExecutorParameter describes a parameter for a workflow executor
type WorkflowExecutorParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// WorkflowExecutorExample provides an example of how to use a workflow executor
type WorkflowExecutorExample struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Config      config.WorkflowConfig `json:"config"`
	Input       string                `json:"input"`
}

// WorkflowExecutorInfo provides information about a workflow executor
type WorkflowExecutorInfo struct {
	Name        string                      `json:"name"`
	Type        string                      `json:"type"`
	Description string                      `json:"description"`
	Features    []string                    `json:"features"`
	Parameters  []WorkflowExecutorParameter `json:"parameters"`
	Examples    []WorkflowExecutorExample   `json:"examples"`
	Modes       []string                    `json:"modes"`
}
