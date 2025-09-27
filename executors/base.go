package executors

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kadirpekel/hector"
	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// BASE EXECUTOR - COMMON FUNCTIONALITY
// ============================================================================

// BaseExecutor provides common functionality for all executors
type BaseExecutor struct {
	name         string
	executorType string
	capabilities interfaces.ExecutorCapabilities
	mu           sync.RWMutex
	startTime    time.Time
	isHealthy    bool
}

// NewBaseExecutor creates a new base executor
func NewBaseExecutor(name, executorType string, capabilities interfaces.ExecutorCapabilities) *BaseExecutor {
	return &BaseExecutor{
		name:         name,
		executorType: executorType,
		capabilities: capabilities,
		isHealthy:    true,
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
func (be *BaseExecutor) GetCapabilities() interfaces.ExecutorCapabilities {
	return be.capabilities
}

// IsHealthy returns true if the executor is ready to execute workflows
func (be *BaseExecutor) IsHealthy(ctx context.Context) bool {
	be.mu.RLock()
	defer be.mu.RUnlock()
	return be.isHealthy
}

// SetHealthy sets the health status of the executor
func (be *BaseExecutor) SetHealthy(healthy bool) {
	be.mu.Lock()
	defer be.mu.Unlock()
	be.isHealthy = healthy
}

// StartExecution initializes execution state
func (be *BaseExecutor) StartExecution() {
	be.mu.Lock()
	defer be.mu.Unlock()
	be.startTime = time.Now()
}

// GetExecutionDuration returns the duration since execution started
func (be *BaseExecutor) GetExecutionDuration() time.Duration {
	be.mu.RLock()
	defer be.mu.RUnlock()
	if be.startTime.IsZero() {
		return 0
	}
	return time.Since(be.startTime)
}

// ============================================================================
// EXECUTION CONTEXT - SHARED STATE MANAGEMENT
// ============================================================================

// ExecutionContext manages shared execution state
type ExecutionContext struct {
	mu          sync.RWMutex
	workflow    *config.WorkflowConfig
	team        *hector.Team
	startTime   time.Time
	timeout     time.Duration
	cancelFunc  context.CancelFunc
	results     map[string]*hector.AgentResult
	sharedState map[string]interface{}
	errors      []error
	status      ExecutionStatus
}

// ExecutionStatus represents the current execution status
type ExecutionStatus string

const (
	StatusInitializing ExecutionStatus = "initializing"
	StatusRunning      ExecutionStatus = "running"
	StatusCompleted    ExecutionStatus = "completed"
	StatusFailed       ExecutionStatus = "failed"
	StatusCancelled    ExecutionStatus = "cancelled"
)

// NewExecutionContext creates a new execution context
func NewExecutionContext(workflow *config.WorkflowConfig, team *hector.Team) *ExecutionContext {
	return &ExecutionContext{
		workflow:    workflow,
		team:        team,
		startTime:   time.Now(),
		results:     make(map[string]*hector.AgentResult),
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

// GetTeam returns the team
func (ec *ExecutionContext) GetTeam() *hector.Team {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.team
}

// SetStatus sets the execution status
func (ec *ExecutionContext) SetStatus(status ExecutionStatus) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.status = status
}

// GetStatus returns the current execution status
func (ec *ExecutionContext) GetStatus() ExecutionStatus {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.status
}

// AddResult adds a result to the execution context
func (ec *ExecutionContext) AddResult(key string, result *hector.AgentResult) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.results[key] = result
}

// GetResult retrieves a result by key
func (ec *ExecutionContext) GetResult(key string) (*hector.AgentResult, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	result, exists := ec.results[key]
	return result, exists
}

// GetAllResults returns all results
func (ec *ExecutionContext) GetAllResults() map[string]*hector.AgentResult {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	results := make(map[string]*hector.AgentResult)
	for k, v := range ec.results {
		results[k] = v
	}
	return results
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

// HasErrors returns true if there are any errors
func (ec *ExecutionContext) HasErrors() bool {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return len(ec.errors) > 0
}

// GetDuration returns the execution duration
func (ec *ExecutionContext) GetDuration() time.Duration {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return time.Since(ec.startTime)
}

// SetTimeout sets the execution timeout
func (ec *ExecutionContext) SetTimeout(timeout time.Duration) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.timeout = timeout
}

// GetTimeout returns the execution timeout
func (ec *ExecutionContext) GetTimeout() time.Duration {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.timeout
}

// ============================================================================
// EXECUTION UTILITIES - SHARED HELPER FUNCTIONS
// ============================================================================

// ExecutionError represents an execution-specific error
type ExecutionError struct {
	ExecutorName string
	StepName     string
	Message      string
	Err          error
	Timestamp    time.Time
}

func (e *ExecutionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.ExecutorName, e.StepName, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.ExecutorName, e.StepName, e.Message)
}

func (e *ExecutionError) Unwrap() error {
	return e.Err
}

// NewExecutionError creates a new execution error
func NewExecutionError(executorName, stepName, message string, err error) *ExecutionError {
	return &ExecutionError{
		ExecutorName: executorName,
		StepName:     stepName,
		Message:      message,
		Err:          err,
		Timestamp:    time.Now(),
	}
}

// ValidateWorkflow validates a workflow configuration
func ValidateWorkflow(workflow *config.WorkflowConfig) error {
	if workflow == nil {
		return NewExecutionError("", "", "workflow configuration is nil", nil)
	}

	if workflow.Name == "" {
		return NewExecutionError("", "", "workflow name is required", nil)
	}

	if len(workflow.Agents) == 0 {
		return NewExecutionError("", "", "workflow must have at least one agent", nil)
	}

	return nil
}

// CreateWorkflowResult creates a standardized workflow result
func CreateWorkflowResult(workflow *config.WorkflowConfig, status hector.WorkflowStatus,
	results map[string]*hector.AgentResult, sharedState map[string]interface{},
	duration time.Duration, errors []error) *hector.WorkflowResult {

	// Calculate total tokens used
	totalTokens := 0
	for _, result := range results {
		totalTokens += result.TokensUsed
	}

	// Create final output from results
	finalOutput := ""
	if status == hector.WorkflowStatusCompleted {
		// Aggregate outputs from all results
		outputs := make([]string, 0, len(results))
		for _, result := range results {
			if result.Output != "" {
				outputs = append(outputs, result.Output)
			}
		}
		finalOutput = strings.Join(outputs, "\n")
	}

	// Get error message if any
	errorMsg := ""
	if len(errors) > 0 {
		errorMsg = errors[0].Error()
	}

	return &hector.WorkflowResult{
		WorkflowName:  workflow.Name,
		Status:        status,
		FinalOutput:   finalOutput,
		AgentResults:  results,
		SharedContext: sharedState,
		ExecutionTime: duration,
		TotalTokens:   totalTokens,
		Success:       status == hector.WorkflowStatusCompleted,
		Error:         errorMsg,
		StepsExecuted: len(results),
		Metadata:      map[string]string{},
	}
}

// ============================================================================
// CONSTANTS - EXECUTION CONFIGURATION
// ============================================================================

const (
	// DefaultTimeout is the default execution timeout
	DefaultTimeout = 30 * time.Minute

	// DefaultMaxConcurrency is the default maximum concurrency
	DefaultMaxConcurrency = 5

	// DefaultRetryAttempts is the default number of retry attempts
	DefaultRetryAttempts = 3

	// DefaultRetryDelay is the default delay between retries
	DefaultRetryDelay = 5 * time.Second
)

// ============================================================================
// LOGGING UTILITIES - STRUCTURED LOGGING
// ============================================================================

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// Logger provides structured logging for executors
type Logger struct {
	executorName string
	level        LogLevel
}

// NewLogger creates a new logger for an executor
func NewLogger(executorName string) *Logger {
	return &Logger{
		executorName: executorName,
		level:        LogLevelInfo,
	}
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= LogLevelDebug {
		fmt.Printf("[DEBUG][%s] %s\n", l.executorName, fmt.Sprintf(format, args...))
	}
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		fmt.Printf("[INFO][%s] %s\n", l.executorName, fmt.Sprintf(format, args...))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		fmt.Printf("[WARN][%s] %s\n", l.executorName, fmt.Sprintf(format, args...))
	}
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= LogLevelError {
		fmt.Printf("[ERROR][%s] %s\n", l.executorName, fmt.Sprintf(format, args...))
	}
}
