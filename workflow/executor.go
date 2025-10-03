package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kadirpekel/hector/config"
)

// BaseExecutor provides common functionality for all workflow executors
type BaseExecutor struct {
	name         string
	executorType string
	capabilities ExecutorCapabilities
	mu           sync.RWMutex
	startTime    time.Time
}

// NewBaseExecutor creates a new base executor
func NewBaseExecutor(name, executorType string, capabilities ExecutorCapabilities) *BaseExecutor {
	return &BaseExecutor{
		name:         name,
		executorType: executorType,
		capabilities: capabilities,
	}
}

// GetName returns the executor name
func (be *BaseExecutor) GetName() string {
	return be.name
}

// GetType returns the executor type
func (be *BaseExecutor) GetType() string {
	return be.executorType
}

// GetCapabilities returns the executor's capabilities
func (be *BaseExecutor) GetCapabilities() ExecutorCapabilities {
	be.mu.RLock()
	defer be.mu.RUnlock()
	return be.capabilities
}

// ExecutionContext manages shared execution state - NO INTERFACE{}
type ExecutionContext struct {
	mu          sync.RWMutex
	workflow    *config.WorkflowConfig
	request     *WorkflowRequest
	startTime   time.Time
	timeout     time.Duration
	cancelFunc  context.CancelFunc
	results     map[string]*AgentResult
	sharedState map[string]interface{}
	errors      []error
	status      ExecutionStatus
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(request *WorkflowRequest) *ExecutionContext {
	return &ExecutionContext{
		workflow:    request.Workflow,
		request:     request,
		startTime:   time.Now(),
		results:     make(map[string]*AgentResult),
		sharedState: make(map[string]interface{}),
		errors:      make([]error, 0),
		status:      StatusInitializing,
	}
}

// GetWorkflow returns the workflow configuration
func (ec *ExecutionContext) GetWorkflow() *config.WorkflowConfig {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.workflow
}

// GetRequest returns the workflow request
func (ec *ExecutionContext) GetRequest() *WorkflowRequest {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.request
}

// GetDuration returns the execution duration
func (ec *ExecutionContext) GetDuration() time.Duration {
	return time.Since(ec.startTime)
}

// SetSharedState sets a value in the shared state
func (ec *ExecutionContext) SetSharedState(key string, value interface{}) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.sharedState[key] = value
}

// GetSharedState retrieves a value from the shared state
func (ec *ExecutionContext) GetSharedState(key string) (interface{}, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	value, exists := ec.sharedState[key]
	return value, exists
}

// GetAllSharedState returns all shared state
func (ec *ExecutionContext) GetAllSharedState() map[string]interface{} {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	state := make(map[string]interface{})
	for k, v := range ec.sharedState {
		state[k] = v
	}
	return state
}

// AddError adds an error to the execution context
func (ec *ExecutionContext) AddError(err error) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.errors = append(ec.errors, err)
}

// GetErrors returns all errors
func (ec *ExecutionContext) GetErrors() []error {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	errors := make([]error, len(ec.errors))
	copy(errors, ec.errors)
	return errors
}

// SetStatus sets the execution status
func (ec *ExecutionContext) SetStatus(status ExecutionStatus) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.status = status
}

// GetStatus returns the execution status
func (ec *ExecutionContext) GetStatus() ExecutionStatus {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.status
}

// SetResult sets an agent result
func (ec *ExecutionContext) SetResult(stepName string, result *AgentResult) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.results[stepName] = result
}

// GetResult retrieves an agent result by step name
func (ec *ExecutionContext) GetResult(stepName string) (*AgentResult, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	result, exists := ec.results[stepName]
	return result, exists
}

// GetAllResults returns all agent results
func (ec *ExecutionContext) GetAllResults() map[string]*AgentResult {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	results := make(map[string]*AgentResult)
	for k, v := range ec.results {
		results[k] = v
	}
	return results
}

// CreateWorkflowResult creates a standardized workflow result
func CreateWorkflowResult(
	workflow *config.WorkflowConfig,
	status WorkflowStatus,
	results map[string]*AgentResult,
	sharedState map[string]interface{},
	duration time.Duration,
	errors []error,
) *WorkflowResult {

	totalTokens := 0
	success := status == WorkflowStatusCompleted

	for _, result := range results {
		totalTokens += result.TokensUsed
		if !result.Success && success {
			success = false
		}
	}

	finalOutput := ""
	if success {
		finalOutput = CombineResults(results)
	}

	errorMsg := ""
	if len(errors) > 0 {
		errorMsg = CombineErrors(errors)
	}

	return &WorkflowResult{
		WorkflowName:  workflow.Name,
		Status:        status,
		Success:       success,
		Error:         errorMsg,
		Results:       results,
		FinalOutput:   finalOutput,
		ExecutionTime: duration,
		TotalTokens:   totalTokens,
		StepsExecuted: len(results),
		Metadata:      make(map[string]string),
	}
}

// CombineResults creates final output from individual results
func CombineResults(results map[string]*AgentResult) string {
	if len(results) == 0 {
		return "No results"
	}

	// Results are already streamed in real-time, no need to repeat them
	// Just return a simple completion message
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	return fmt.Sprintf("Workflow completed: %d/%d agents succeeded", successCount, len(results))
}

// CombineErrors combines multiple errors into a single error message
func CombineErrors(errors []error) string {
	if len(errors) == 0 {
		return ""
	}

	errorString := errors[0].Error()
	for _, err := range errors[1:] {
		errorString += "; " + err.Error()
	}
	return errorString
}
