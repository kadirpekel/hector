package executors

import (
	"context"
	"fmt"
	"sync"

	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// REGISTRY - EXECUTOR SYSTEM CORE
// ============================================================================

// Executor represents a workflow execution engine
// This is an alias to the interface defined in the interfaces package
type Executor = interfaces.Executor

// ExecutorRegistry manages multiple workflow executors
type ExecutorRegistry struct {
	mu        sync.RWMutex
	executors map[string]Executor
}

// Ensure ExecutorRegistry implements the interface
var _ interfaces.ExecutorRegistry = (*ExecutorRegistry)(nil)

// NewExecutorRegistry creates a new executor registry
func NewExecutorRegistry() *ExecutorRegistry {
	return &ExecutorRegistry{
		executors: make(map[string]Executor),
	}
}

// RegisterExecutor adds an executor to the registry
func (er *ExecutorRegistry) RegisterExecutor(executor Executor) error {
	if executor == nil {
		return fmt.Errorf("executor cannot be nil")
	}

	name := executor.GetName()
	if name == "" {
		return fmt.Errorf("executor name cannot be empty")
	}

	er.mu.Lock()
	defer er.mu.Unlock()

	if _, exists := er.executors[name]; exists {
		return fmt.Errorf("executor with name '%s' already registered", name)
	}

	er.executors[name] = executor
	return nil
}

// GetExecutor retrieves an executor by name
func (er *ExecutorRegistry) GetExecutor(name string) (Executor, bool) {
	er.mu.RLock()
	defer er.mu.RUnlock()

	executor, exists := er.executors[name]
	return executor, exists
}

// ListExecutors returns all registered executors
func (er *ExecutorRegistry) ListExecutors() []Executor {
	er.mu.RLock()
	defer er.mu.RUnlock()

	executors := make([]Executor, 0, len(er.executors))
	for _, executor := range er.executors {
		executors = append(executors, executor)
	}
	return executors
}

// GetExecutorForWorkflow finds the best executor for a given workflow
func (er *ExecutorRegistry) GetExecutorForWorkflow(workflow *config.WorkflowConfig) (Executor, error) {
	er.mu.RLock()
	defer er.mu.RUnlock()

	// Try to find an executor that can handle this workflow
	for _, executor := range er.executors {
		if executor.CanHandle(workflow) {
			return executor, nil
		}
	}

	return nil, fmt.Errorf("no executor found that can handle workflow mode: %s", workflow.Mode)
}

// ExecuteWorkflow executes a workflow using the appropriate executor
func (er *ExecutorRegistry) ExecuteWorkflow(ctx context.Context, workflow *config.WorkflowConfig, team interface{}) (interface{}, error) {
	executor, err := er.GetExecutorForWorkflow(workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to find executor: %w", err)
	}

	// Check executor health
	if !executor.IsHealthy(ctx) {
		return nil, fmt.Errorf("executor '%s' is not healthy", executor.GetName())
	}

	return executor.Execute(ctx, team)
}

// GetExecutorCapabilities returns capabilities for all executors
func (er *ExecutorRegistry) GetExecutorCapabilities() map[string]interfaces.ExecutorCapabilities {
	er.mu.RLock()
	defer er.mu.RUnlock()

	capabilities := make(map[string]interfaces.ExecutorCapabilities)
	for name, executor := range er.executors {
		capabilities[name] = executor.GetCapabilities()
	}
	return capabilities
}

// GetHealthStatus returns the health status of all executors
func (er *ExecutorRegistry) GetHealthStatus(ctx context.Context) map[string]bool {
	er.mu.RLock()
	defer er.mu.RUnlock()

	status := make(map[string]bool)
	for name, executor := range er.executors {
		status[name] = executor.IsHealthy(ctx)
	}
	return status
}

// RemoveExecutor removes an executor from the registry
func (er *ExecutorRegistry) RemoveExecutor(name string) error {
	er.mu.Lock()
	defer er.mu.Unlock()

	if _, exists := er.executors[name]; !exists {
		return fmt.Errorf("executor '%s' not found", name)
	}

	delete(er.executors, name)
	return nil
}

// Clear removes all executors from the registry
func (er *ExecutorRegistry) Clear() {
	er.mu.Lock()
	defer er.mu.Unlock()

	er.executors = make(map[string]Executor)
}

// Count returns the number of registered executors
func (er *ExecutorRegistry) Count() int {
	er.mu.RLock()
	defer er.mu.RUnlock()

	return len(er.executors)
}
