package executors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kadirpekel/hector"
	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// DAG EXECUTOR - REFACTORED IMPLEMENTATION
// ============================================================================

// DAGExecutor executes workflows as Directed Acyclic Graphs
type DAGExecutor struct {
	*BaseExecutor
	logger *Logger
}

// NewDAGExecutor creates a new DAG executor
func NewDAGExecutor() *DAGExecutor {
	base := NewBaseExecutor("dag", "dag", interfaces.ExecutorCapabilities{
		SupportsParallelExecution: true,
		SupportsDynamicPlanning:   false,
		SupportsStaticWorkflows:   true,
		RequiredFeatures:          []string{"workflow_steps", "dependencies"},
		MaxConcurrency:            DefaultMaxConcurrency,
		SupportsRollback:          false,
	})

	return &DAGExecutor{
		BaseExecutor: base,
		logger:       NewLogger("dag"),
	}
}

// CanHandle returns true if this executor can handle the given workflow
func (de *DAGExecutor) CanHandle(workflow *config.WorkflowConfig) bool {
	return workflow.Mode == config.ExecutionModeDAG
}

// Execute runs the workflow using DAG execution
func (de *DAGExecutor) Execute(ctx context.Context, team interface{}) (interface{}, error) {
	// Cast team to the expected type
	t, ok := team.(*hector.Team)
	if !ok {
		return nil, NewExecutionError(de.GetName(), "execute", "invalid team type", nil)
	}

	// Validate workflow
	if err := ValidateWorkflow(t.GetWorkflow()); err != nil {
		return nil, NewExecutionError(de.GetName(), "execute", "workflow validation failed", err)
	}

	de.logger.Info("Starting DAG execution with %d steps", len(t.GetWorkflow().Execution.DAG.Steps))
	de.StartExecution()

	// Create execution context
	execCtx := NewExecutionContext(t.GetWorkflow(), t)
	execCtx.SetStatus(StatusRunning)

	// Set timeout if configured
	if t.GetWorkflow().Settings.Timeout > 0 {
		execCtx.SetTimeout(t.GetWorkflow().Settings.Timeout)
	}

	// Build workflow graph
	graph, err := de.buildWorkflowGraph(t.GetWorkflow().Execution.DAG)
	if err != nil {
		execCtx.SetStatus(StatusFailed)
		execCtx.AddError(err)
		return nil, NewExecutionError(de.GetName(), "build_graph", "failed to build workflow graph", err)
	}

	// Execute the DAG
	result, err := de.executeDAG(ctx, execCtx, graph)
	if err != nil {
		execCtx.SetStatus(StatusFailed)
		execCtx.AddError(err)
		return nil, err
	}

	execCtx.SetStatus(StatusCompleted)
	de.logger.Info("DAG execution completed successfully in %v", execCtx.GetDuration())
	return result, nil
}

// ============================================================================
// DAG EXECUTION LOGIC
// ============================================================================

// workflowGraph represents the DAG structure
type workflowGraph struct {
	Steps        map[string]*config.WorkflowStep
	Dependencies map[string][]string // step_name -> list of dependencies
	Dependents   map[string][]string // step_name -> list of dependents
	Roots        []string            // steps with no dependencies
	Leaves       []string            // steps with no dependents
}

// buildWorkflowGraph builds a DAG from workflow configuration
func (de *DAGExecutor) buildWorkflowGraph(dagConfig *config.DAGExecution) (*workflowGraph, error) {
	if dagConfig == nil {
		return nil, fmt.Errorf("DAG configuration is nil")
	}

	graph := &workflowGraph{
		Steps:        make(map[string]*config.WorkflowStep),
		Dependencies: make(map[string][]string),
		Dependents:   make(map[string][]string),
		Roots:        make([]string, 0),
		Leaves:       make([]string, 0),
	}

	// Build steps map
	for _, step := range dagConfig.Steps {
		graph.Steps[step.Name] = &step
		graph.Dependencies[step.Name] = make([]string, 0)
		graph.Dependents[step.Name] = make([]string, 0)
	}

	// Build dependency relationships
	for _, step := range dagConfig.Steps {
		for _, dep := range step.DependsOn {
			if _, exists := graph.Steps[dep]; !exists {
				return nil, fmt.Errorf("dependency '%s' not found for step '%s'", dep, step.Name)
			}
			graph.Dependencies[step.Name] = append(graph.Dependencies[step.Name], dep)
			graph.Dependents[dep] = append(graph.Dependents[dep], step.Name)
		}
	}

	// Find roots and leaves
	for stepName := range graph.Steps {
		if len(graph.Dependencies[stepName]) == 0 {
			graph.Roots = append(graph.Roots, stepName)
		}
		if len(graph.Dependents[stepName]) == 0 {
			graph.Leaves = append(graph.Leaves, stepName)
		}
	}

	// Validate no cycles
	if err := de.validateNoCycles(graph); err != nil {
		return nil, fmt.Errorf("workflow graph contains cycles: %w", err)
	}

	de.logger.Debug("Built DAG with %d roots, %d leaves", len(graph.Roots), len(graph.Leaves))
	return graph, nil
}

// validateNoCycles validates that the graph has no cycles
func (de *DAGExecutor) validateNoCycles(graph *workflowGraph) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for stepName := range graph.Steps {
		if !visited[stepName] {
			if de.hasCycleDFS(stepName, graph, visited, recStack) {
				return fmt.Errorf("cycle detected in workflow graph")
			}
		}
	}
	return nil
}

// hasCycleDFS performs depth-first search to detect cycles
func (de *DAGExecutor) hasCycleDFS(stepName string, graph *workflowGraph, visited, recStack map[string]bool) bool {
	visited[stepName] = true
	recStack[stepName] = true

	for _, dep := range graph.Dependencies[stepName] {
		if !visited[dep] {
			if de.hasCycleDFS(dep, graph, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[stepName] = false
	return false
}

// executeDAG executes the DAG workflow
func (de *DAGExecutor) executeDAG(ctx context.Context, execCtx *ExecutionContext, graph *workflowGraph) (*hector.WorkflowResult, error) {
	// Create execution context with timeout
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	if execCtx.GetTimeout() > 0 {
		ctxWithCancel, cancel = context.WithTimeout(ctxWithCancel, execCtx.GetTimeout())
		defer cancel()
	}

	// Track execution state
	completed := make(map[string]bool)
	failed := make(map[string]bool)
	running := make(map[string]bool)
	results := make(map[string]*hector.AgentResult)

	// Execute roots first
	readySteps := make([]string, len(graph.Roots))
	copy(readySteps, graph.Roots)

	for len(readySteps) > 0 {
		// Get next ready step
		stepName := readySteps[0]
		readySteps = readySteps[1:]

		if completed[stepName] || failed[stepName] || running[stepName] {
			continue
		}

		// Check if all dependencies are completed
		if !de.allDependenciesCompleted(stepName, graph, completed) {
			continue
		}

		// Execute step
		running[stepName] = true
		result, err := de.executeStep(ctxWithCancel, stepName, graph.Steps[stepName], execCtx.GetTeam())

		if err != nil {
			failed[stepName] = true
			delete(running, stepName)
			execCtx.AddError(NewExecutionError(de.GetName(), stepName, "step execution failed", err))

			// Check error policy
			if de.shouldFailFast(execCtx) {
				return nil, err
			}
			continue
		}

		// Mark as completed
		completed[stepName] = true
		delete(running, stepName)
		results[stepName] = result
		execCtx.AddResult(stepName, result)

		// Add dependents to ready queue
		for _, dependent := range graph.Dependents[stepName] {
			if !completed[dependent] && !failed[dependent] && !running[dependent] {
				readySteps = append(readySteps, dependent)
			}
		}

		de.logger.Debug("Completed step '%s', %d/%d steps done", stepName, len(completed), len(graph.Steps))
	}

	// Check if all steps completed
	if len(completed) == len(graph.Steps) {
		return CreateWorkflowResult(execCtx.GetWorkflow(), hector.WorkflowStatusCompleted,
			results, execCtx.GetAllSharedState(), execCtx.GetDuration(), execCtx.GetErrors()), nil
	}

	// Some steps failed
	return CreateWorkflowResult(execCtx.GetWorkflow(), hector.WorkflowStatusFailed,
		results, execCtx.GetAllSharedState(), execCtx.GetDuration(), execCtx.GetErrors()), nil
}

// allDependenciesCompleted checks if all dependencies for a step are completed
func (de *DAGExecutor) allDependenciesCompleted(stepName string, graph *workflowGraph, completed map[string]bool) bool {
	for _, dep := range graph.Dependencies[stepName] {
		if !completed[dep] {
			return false
		}
	}
	return true
}

// shouldFailFast checks if execution should fail fast on error
func (de *DAGExecutor) shouldFailFast(execCtx *ExecutionContext) bool {
	workflow := execCtx.GetWorkflow()
	return workflow.Settings.ErrorPolicy == "fail_fast"
}

// executeStep executes a single workflow step
func (de *DAGExecutor) executeStep(ctx context.Context, stepName string, step *config.WorkflowStep, team *hector.Team) (*hector.AgentResult, error) {
	de.logger.Debug("Executing step '%s' with agent '%s'", stepName, step.Agent)
	agent := team.GetAgent(step.Agent)
	if agent == nil {
		return nil, fmt.Errorf("agent '%s' not found", step.Agent)
	}

	// Prepare input
	input := de.prepareStepInput(step.Input, team)

	// Execute agent
	startTime := time.Now()
	response, err := agent.Query(ctx, input)
	duration := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	// Create result
	result := &hector.AgentResult{
		AgentName:  step.Agent,
		StepName:   stepName,
		Output:     response.Answer,
		TokensUsed: response.TokensUsed,
		Duration:   duration,
		Success:    true,
		Confidence: response.Confidence,
		Timestamp:  time.Now(),
		Metadata: map[string]string{
			"input":      input,
			"start_time": startTime.Format(time.RFC3339),
		},
	}

	// Store result in shared state
	team.GetSharedState().SetResult(stepName, result)
	team.GetSharedState().SetContext(step.Output, result.Output, step.Agent)

	de.logger.Debug("Step '%s' completed successfully in %v", stepName, duration)
	return result, nil
}

// prepareStepInput prepares the input for a step by replacing placeholders
func (de *DAGExecutor) prepareStepInput(input string, team *hector.Team) string {
	if !strings.Contains(input, "${") {
		return input
	}

	// Replace user_input
	if userInput, exists := team.GetSharedState().GetContext("user_input"); exists {
		if userInputStr, ok := userInput.(string); ok {
			input = strings.ReplaceAll(input, "${user_input}", userInputStr)
		}
	}

	// Replace step outputs
	for resultKey, result := range team.GetSharedState().GetAllResults() {
		placeholder := fmt.Sprintf("${%s}", resultKey)
		if strings.Contains(input, placeholder) {
			input = strings.ReplaceAll(input, placeholder, result.Output)
		}
	}

	// Replace context variables
	for key, value := range team.GetSharedState().Context {
		placeholder := fmt.Sprintf("${%s}", key)
		if strings.Contains(input, placeholder) {
			if valueStr, ok := value.(string); ok {
				input = strings.ReplaceAll(input, placeholder, valueStr)
			}
		}
	}

	return input
}
