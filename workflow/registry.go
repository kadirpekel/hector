package workflow

import (
	"context"
	"fmt"
	"sync"

	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/registry"
)

// WorkflowExecutorRegistry manages workflow executors
type WorkflowExecutorRegistry struct {
	*registry.BaseRegistry[WorkflowExecutor]
	mu        sync.RWMutex
	factories map[string]WorkflowExecutorFactory
}

// NewWorkflowExecutorRegistry creates a new workflow executor registry
func NewWorkflowExecutorRegistry() *WorkflowExecutorRegistry {
	return &WorkflowExecutorRegistry{
		BaseRegistry: registry.NewBaseRegistry[WorkflowExecutor](),
		factories:    make(map[string]WorkflowExecutorFactory),
	}
}

// WorkflowExecutionError represents a workflow execution error
type WorkflowExecutionError struct {
	Component string
	Action    string
	Message   string
	Err       error
}

func (e *WorkflowExecutionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Component, e.Action, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Component, e.Action, e.Message)
}

func NewWorkflowExecutionError(component, action, message string, err error) *WorkflowExecutionError {
	return &WorkflowExecutionError{
		Component: component,
		Action:    action,
		Message:   message,
		Err:       err,
	}
}

// RegisterExecutor registers a workflow executor
func (r *WorkflowExecutorRegistry) RegisterExecutor(executor WorkflowExecutor) error {
	if executor == nil {
		return NewWorkflowExecutionError("WorkflowExecutorRegistry", "RegisterExecutor", "executor cannot be nil", nil)
	}

	name := executor.GetName()
	if name == "" {
		return NewWorkflowExecutionError("WorkflowExecutorRegistry", "RegisterExecutor", "executor name cannot be empty", nil)
	}

	return r.Register(name, executor)
}

// RegisterFactory registers an executor factory
func (r *WorkflowExecutorRegistry) RegisterFactory(factory WorkflowExecutorFactory) error {
	if factory == nil {
		return NewWorkflowExecutionError("WorkflowExecutorRegistry", "RegisterFactory", "factory cannot be nil", nil)
	}

	executorType := factory.GetSupportedType()
	if executorType == "" {
		return NewWorkflowExecutionError("WorkflowExecutorRegistry", "RegisterFactory", "factory must specify supported type", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[executorType]; exists {
		return NewWorkflowExecutionError("WorkflowExecutorRegistry", "RegisterFactory",
			fmt.Sprintf("factory already registered for type %s", executorType), nil)
	}
	r.factories[executorType] = factory
	return nil
}

// GetExecutor retrieves an executor by name
func (r *WorkflowExecutorRegistry) GetExecutor(name string) (WorkflowExecutor, error) {
	executor, exists := r.Get(name)
	if !exists {
		return nil, NewWorkflowExecutionError("WorkflowExecutorRegistry", "GetExecutor",
			fmt.Sprintf("executor '%s' not found", name), nil)
	}
	return executor, nil
}

// CreateExecutor creates an executor instance using the registered factory
func (r *WorkflowExecutorRegistry) CreateExecutor(executorType string, workflowConfig config.WorkflowConfig) (WorkflowExecutor, error) {
	r.mu.RLock()
	factory, exists := r.factories[executorType]
	r.mu.RUnlock()

	if !exists {
		return nil, NewWorkflowExecutionError("WorkflowExecutorRegistry", "CreateExecutor",
			fmt.Sprintf("no factory registered for executor type '%s'", executorType), nil)
	}

	executor, err := factory.CreateExecutor(executorType, workflowConfig)
	if err != nil {
		return nil, NewWorkflowExecutionError("WorkflowExecutorRegistry", "CreateExecutor",
			fmt.Sprintf("failed to create executor of type '%s'", executorType), err)
	}
	return executor, nil
}

// ExecuteWorkflow finds a suitable executor and runs the workflow - NO INTERFACE{}!
func (r *WorkflowExecutorRegistry) ExecuteWorkflow(ctx context.Context, request *WorkflowRequest) (*WorkflowResult, error) {
	if request == nil {
		return nil, NewWorkflowExecutionError("WorkflowExecutorRegistry", "ExecuteWorkflow", "request cannot be nil", nil)
	}

	if request.Workflow == nil {
		return nil, NewWorkflowExecutionError("WorkflowExecutorRegistry", "ExecuteWorkflow", "workflow config cannot be nil", nil)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	list := r.List()
	for _, executor := range list {
		if executor.CanHandle(request.Workflow) {
			return executor.Execute(ctx, request)
		}
	}
	return nil, NewWorkflowExecutionError("WorkflowExecutorRegistry", "ExecuteWorkflow",
		fmt.Sprintf("no suitable executor found for workflow mode '%s'", request.Workflow.Mode), nil)
}

// GetSupportedModes returns all supported execution modes
func (r *WorkflowExecutorRegistry) GetSupportedModes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modes := make(map[string]bool)
	list := r.List()
	for _, executor := range list {
		capabilities := executor.GetCapabilities()
		if capabilities.SupportsStaticWorkflows {
			modes["dag"] = true
		}
		if capabilities.SupportsDynamicPlanning {
			modes["autonomous"] = true
		}
	}

	result := make([]string, 0, len(modes))
	for mode := range modes {
		result = append(result, mode)
	}
	return result
}

// CanHandle checks if any registered executor can handle the workflow
func (r *WorkflowExecutorRegistry) CanHandle(workflow *config.WorkflowConfig) bool {
	if workflow == nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	list := r.List()
	for _, executor := range list {
		if executor.CanHandle(workflow) {
			return true
		}
	}
	return false
}

// GetSupportedExecutorsByMode returns executors that can handle a specific execution mode
func (r *WorkflowExecutorRegistry) GetSupportedExecutorsByMode(mode string) []WorkflowExecutor {
	var supported []WorkflowExecutor

	list := r.List()
	for _, executor := range list {
		capabilities := executor.GetCapabilities()
		if mode == "dag" && capabilities.SupportsStaticWorkflows {
			supported = append(supported, executor)
		} else if mode == "autonomous" && capabilities.SupportsDynamicPlanning {
			supported = append(supported, executor)
		}
	}

	return supported
}
