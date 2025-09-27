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
	"github.com/kadirpekel/hector/providers"
)

// ============================================================================
// AUTONOMOUS EXECUTOR - REFACTORED IMPLEMENTATION
// ============================================================================

// AutonomousExecutor executes workflows using dynamic AI tination
type AutonomousExecutor struct {
	*BaseExecutor
	logger  *Logger
	llm     providers.LLMProvider
	context *autonomousContext
	config  *config.AutonomousExecution
}

// NewAutonomousExecutor creates a new autonomous executor
func NewAutonomousExecutor() *AutonomousExecutor {
	base := NewBaseExecutor("autonomous", "autonomous", interfaces.ExecutorCapabilities{
		SupportsParallelExecution: true,
		SupportsDynamicPlanning:   true,
		SupportsStaticWorkflows:   false,
		RequiredFeatures:          []string{"llm_provider", "autonomous_config"},
		MaxConcurrency:            DefaultMaxConcurrency,
		SupportsRollback:          false,
	})

	return &AutonomousExecutor{
		BaseExecutor: base,
		logger:       NewLogger("autonomous"),
	}
}

// CanHandle returns true if this executor can handle the given workflow
func (ae *AutonomousExecutor) CanHandle(workflow *config.WorkflowConfig) bool {
	return workflow.Mode == config.ExecutionModeAutonomous
}

// Execute runs the workflow using autonomous tination
func (ae *AutonomousExecutor) Execute(ctx context.Context, team interface{}) (interface{}, error) {
	// Cast team to the expected type
	t, ok := team.(*hector.Team)
	if !ok {
		return nil, NewExecutionError(ae.GetName(), "execute", "invalid team type", nil)
	}

	// Validate workflow
	if err := ValidateWorkflow(t.GetWorkflow()); err != nil {
		return nil, NewExecutionError(ae.GetName(), "execute", "workflow validation failed", err)
	}

	ae.logger.Info("Starting autonomous execution with goal: %s", t.GetWorkflow().Execution.Autonomous.Goal)
	ae.StartExecution()

	// Create execution context
	execCtx := NewExecutionContext(t.GetWorkflow(), t)
	execCtx.SetStatus(StatusRunning)

	// Set timeout if configured
	if t.GetWorkflow().Execution.Autonomous.TerminationConditions.MaxDuration > 0 {
		execCtx.SetTimeout(t.GetWorkflow().Execution.Autonomous.TerminationConditions.MaxDuration)
	}

	// Initialize autonomous context
	ae.context = &autonomousContext{
		mu:             sync.RWMutex{},
		originalGoal:   t.GetWorkflow().Execution.Autonomous.Goal,
		currentGoal:    t.GetWorkflow().Execution.Autonomous.Goal,
		iterations:     make([]autonomousIteration, 0),
		activeTasks:    make(map[string]agentAssignment),
		completedTasks: make(map[string]agentAssignment),
		failedTasks:    make(map[string]agentAssignment),
		globalContext:  make(map[string]interface{}),
	}

	// Initialize LLM provider
	if err := ae.initializeLLM(t.GetWorkflow().Execution.Autonomous); err != nil {
		execCtx.SetStatus(StatusFailed)
		execCtx.AddError(err)
		return nil, NewExecutionError(ae.GetName(), "initialize_llm", "failed to initialize LLM provider", err)
	}

	// Initialize global context
	ae.context.globalContext["user_input"] = t.GetSharedState().Context["user_input"]
	ae.context.globalContext["workflow_name"] = t.GetWorkflow().Name

	// Execute autonomous workflow
	result, err := ae.executeAutonomousWorkflow(ctx, execCtx)
	if err != nil {
		execCtx.SetStatus(StatusFailed)
		execCtx.AddError(err)
		return nil, err
	}

	execCtx.SetStatus(StatusCompleted)
	ae.logger.Info("Autonomous execution completed successfully in %v", execCtx.GetDuration())
	return result, nil
}

// ============================================================================
// AUTONOMOUS EXECUTION LOGIC
// ============================================================================

// autonomousContext tracks the state of autonomous execution
type autonomousContext struct {
	mu             sync.RWMutex
	originalGoal   string
	currentGoal    string
	iterations     []autonomousIteration
	activeTasks    map[string]agentAssignment
	completedTasks map[string]agentAssignment
	failedTasks    map[string]agentAssignment
	globalContext  map[string]interface{}
}

// autonomousIteration represents one iteration of autonomous tination
type autonomousIteration struct {
	Number        int               `json:"number"`
	Goal          string            `json:"goal"`
	Plan          *executionPlan    `json:"plan"`
	Assignments   []agentAssignment `json:"assignments"`
	Results       []agentAssignment `json:"results"`
	Consensus     *consensusResult  `json:"consensus,omitempty"`
	GoalEvolution *goalEvolution    `json:"goal_evolution,omitempty"`
	Duration      time.Duration     `json:"duration"`
	Success       bool              `json:"success"`
	Error         string            `json:"error,omitempty"`
}

// executionPlan represents the AI's plan for achieving the goal
type executionPlan struct {
	Strategy   string            `json:"strategy"`
	Tasks      []plannedTask     `json:"tasks"`
	Rationale  string            `json:"rationale"`
	Confidence float64           `json:"confidence"`
	Metadata   map[string]string `json:"metadata"`
}

// plannedTask represents a task in the execution plan
type plannedTask struct {
	ID                   string            `json:"id"`
	Description          string            `json:"description"`
	RequiredCapabilities []string          `json:"required_capabilities"`
	Priority             int               `json:"priority"`
	EstimatedEffort      string            `json:"estimated_effort"`
	Dependencies         []string          `json:"dependencies"`
	Metadata             map[string]string `json:"metadata"`
}

// agentAssignment represents an assignment of a task to an agent
type agentAssignment struct {
	TaskID     string            `json:"task_id"`
	AgentName  string            `json:"agent_name"`
	AgentType  string            `json:"agent_type"`
	Input      string            `json:"input"`
	Output     string            `json:"output,omitempty"`
	Status     string            `json:"status"` // "assigned", "in_progress", "completed", "failed"
	StartTime  time.Time         `json:"start_time,omitempty"`
	EndTime    time.Time         `json:"end_time,omitempty"`
	Duration   time.Duration     `json:"duration,omitempty"`
	TokensUsed int               `json:"tokens_used,omitempty"`
	Confidence float64           `json:"confidence,omitempty"`
	Error      string            `json:"error,omitempty"`
	Metadata   map[string]string `json:"metadata"`
}

// consensusResult represents the result of a consensus decision
type consensusResult struct {
	Decision     string            `json:"decision"`
	Confidence   float64           `json:"confidence"`
	Participants []string          `json:"participants"`
	Rationale    string            `json:"rationale"`
	Metadata     map[string]string `json:"metadata"`
}

// goalEvolution represents how the goal has evolved
type goalEvolution struct {
	OriginalGoal string            `json:"original_goal"`
	CurrentGoal  string            `json:"current_goal"`
	Evolution    []goalChange      `json:"evolution"`
	Rationale    string            `json:"rationale"`
	Metadata     map[string]string `json:"metadata"`
}

// goalChange represents a single change to the goal
type goalChange struct {
	Iteration  int       `json:"iteration"`
	OldGoal    string    `json:"old_goal"`
	NewGoal    string    `json:"new_goal"`
	Rationale  string    `json:"rationale"`
	Timestamp  time.Time `json:"timestamp"`
	Confidence float64   `json:"confidence"`
}

// initializeLLM initializes the LLM provider
func (ae *AutonomousExecutor) initializeLLM(config *config.AutonomousExecution) error {
	// For now, return a stub - provider creation will be implemented later
	ae.logger.Warn("LLM provider initialization not yet implemented")
	return nil
}

// executeAutonomousWorkflow executes the main autonomous workflow loop
func (ae *AutonomousExecutor) executeAutonomousWorkflow(ctx context.Context, execCtx *ExecutionContext) (*hector.WorkflowResult, error) {
	maxIterations := ae.config.TerminationConditions.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10 // Default maximum iterations
	}

	ae.logger.Info("Starting autonomous workflow with max %d iterations", maxIterations)

	for iteration := 1; iteration <= maxIterations; iteration++ {
		ae.logger.Info("Starting iteration %d/%d", iteration, maxIterations)

		// Execute iteration
		iterationResult, err := ae.executeIteration(ctx, iteration, execCtx)
		if err != nil {
			ae.logger.Error("Iteration %d failed: %v", iteration, err)
			execCtx.AddError(NewExecutionError(ae.GetName(), fmt.Sprintf("iteration_%d", iteration), "iteration execution failed", err))
			continue
		}

		// Add iteration to context
		ae.context.mu.Lock()
		ae.context.iterations = append(ae.context.iterations, *iterationResult)
		ae.context.mu.Unlock()

		// Check if goal is achieved
		if ae.isGoalAchieved(iterationResult) {
			ae.logger.Info("Goal achieved in iteration %d", iteration)
			break
		}

		// Check termination conditions
		if ae.shouldTerminate(iterationResult) {
			ae.logger.Info("Termination conditions met in iteration %d", iteration)
			break
		}

		// Evolve goal if needed
		if ae.shouldEvolveGoal(iterationResult) {
			ae.evolveGoal(iterationResult)
		}
	}

	// Create final result
	return ae.createFinalResult(execCtx)
}

// executeIteration executes a single autonomous iteration
func (ae *AutonomousExecutor) executeIteration(ctx context.Context, iterationNum int, execCtx *ExecutionContext) (*autonomousIteration, error) {
	startTime := time.Now()

	iteration := &autonomousIteration{
		Number: iterationNum,
		Goal:   ae.context.currentGoal,
	}

	// Step 1: Create execution plan
	ae.logger.Debug("Creating execution plan for iteration %d", iterationNum)
	plan, err := ae.createExecutionPlan(ctx, execCtx.GetTeam())
	if err != nil {
		iteration.Error = err.Error()
		iteration.Success = false
		return iteration, fmt.Errorf("failed to create execution plan: %w", err)
	}
	iteration.Plan = plan

	// Step 2: Assign tasks to agents
	ae.logger.Debug("Assigning tasks to agents for iteration %d", iterationNum)
	assignments, err := ae.assignTasks(ctx, plan.Tasks, execCtx.GetTeam())
	if err != nil {
		iteration.Error = err.Error()
		iteration.Success = false
		return iteration, fmt.Errorf("failed to assign tasks: %w", err)
	}
	iteration.Assignments = assignments

	// Step 3: Execute assignments in parallel
	ae.logger.Debug("Executing %d assignments for iteration %d", len(assignments), iterationNum)
	results, err := ae.executeAssignments(ctx, assignments, execCtx.GetTeam())
	if err != nil {
		iteration.Error = err.Error()
		iteration.Success = false
		return iteration, fmt.Errorf("failed to execute assignments: %w", err)
	}
	iteration.Results = results

	// Step 4: Build consensus (if configured)
	if ae.config.Strategy == "democratic" {
		ae.logger.Debug("Building consensus for iteration %d", iterationNum)
		consensus, err := ae.buildConsensus(ctx, results)
		if err != nil {
			ae.logger.Warn("Consensus building failed: %v", err)
		} else {
			iteration.Consensus = consensus
		}
	}

	iteration.Duration = time.Since(startTime)
	iteration.Success = true
	return iteration, nil
}

// createExecutionPlan creates an execution plan for the current goal
func (ae *AutonomousExecutor) createExecutionPlan(ctx context.Context, team *hector.Team) (*executionPlan, error) {
	// For now, create a simple plan based on available agents
	// This will be enhanced with actual LLM-based planning later
	availableAgents := team.GetWorkflow().Agents

	if len(availableAgents) == 0 {
		return nil, fmt.Errorf("no agents available for planning")
	}

	// Create a simple task for each agent
	tasks := make([]plannedTask, 0, len(availableAgents))
	for i, agentName := range availableAgents {
		task := plannedTask{
			ID:                   fmt.Sprintf("task_%d", i+1),
			Description:          fmt.Sprintf("Work on goal: %s", ae.context.currentGoal),
			RequiredCapabilities: []string{"general"},
			Priority:             i + 1,
			EstimatedEffort:      "medium",
			Dependencies:         []string{},
			Metadata: map[string]string{
				"agent": agentName,
				"goal":  ae.context.currentGoal,
			},
		}
		tasks = append(tasks, task)
	}

	plan := &executionPlan{
		Strategy:   ae.config.Strategy,
		Tasks:      tasks,
		Rationale:  fmt.Sprintf("Simple plan to work on goal: %s", ae.context.currentGoal),
		Confidence: 0.7,
		Metadata: map[string]string{
			"goal":      ae.context.currentGoal,
			"strategy":  ae.config.Strategy,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	return plan, nil
}

// assignTasks assigns tasks to available agents
func (ae *AutonomousExecutor) assignTasks(ctx context.Context, tasks []plannedTask, team *hector.Team) ([]agentAssignment, error) {
	availableAgents := team.GetWorkflow().Agents

	if len(availableAgents) == 0 {
		return nil, fmt.Errorf("no agents available for task assignment")
	}

	assignments := make([]agentAssignment, 0, len(tasks))
	for i, task := range tasks {
		agentName := availableAgents[i%len(availableAgents)] // Round-robin assignment

		assignment := agentAssignment{
			TaskID:    task.ID,
			AgentName: agentName,
			AgentType: "general",
			Input:     task.Description,
			Status:    "assigned",
			Metadata:  task.Metadata,
		}
		assignments = append(assignments, assignment)
	}

	return assignments, nil
}

// executeAssignments executes all assignments in parallel
func (ae *AutonomousExecutor) executeAssignments(ctx context.Context, assignments []agentAssignment, team *hector.Team) ([]agentAssignment, error) {
	results := make([]agentAssignment, len(assignments))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for i, assignment := range assignments {
		wg.Add(1)
		go func(idx int, assign agentAssignment) {
			defer wg.Done()

			result, err := ae.executeAssignment(ctx, assign, team)

			mu.Lock()
			results[idx] = result
			if err != nil {
				errors = append(errors, err)
			}
			mu.Unlock()
		}(i, assignment)
	}

	wg.Wait()

	if len(errors) > 0 {
		return results, fmt.Errorf("some assignments failed: %v", errors)
	}

	return results, nil
}

// executeAssignment executes a single agent assignment
func (ae *AutonomousExecutor) executeAssignment(ctx context.Context, assignment agentAssignment, team *hector.Team) (agentAssignment, error) {

	// Get the agent
	agent := team.GetAgent(assignment.AgentName)
	if agent == nil {
		assignment.Status = "failed"
		assignment.Error = fmt.Sprintf("agent '%s' not found", assignment.AgentName)
		return assignment, fmt.Errorf("agent '%s' not found", assignment.AgentName)
	}

	// Update assignment status
	assignment.Status = "in_progress"
	assignment.StartTime = time.Now()

	// Execute agent
	response, err := agent.Query(ctx, assignment.Input)
	if err != nil {
		assignment.Status = "failed"
		assignment.Error = err.Error()
		assignment.EndTime = time.Now()
		assignment.Duration = assignment.EndTime.Sub(assignment.StartTime)
		return assignment, fmt.Errorf("agent execution failed: %w", err)
	}

	// Update assignment with results
	assignment.Status = "completed"
	assignment.Output = response.Answer
	assignment.EndTime = time.Now()
	assignment.Duration = assignment.EndTime.Sub(assignment.StartTime)
	assignment.TokensUsed = response.TokensUsed
	assignment.Confidence = response.Confidence

	// Store result in shared context
	team.GetSharedState().SetContext(assignment.TaskID, response.Answer, assignment.AgentName)

	return assignment, nil
}

// buildConsensus builds consensus from assignment results
func (ae *AutonomousExecutor) buildConsensus(ctx context.Context, results []agentAssignment) (*consensusResult, error) {
	// Simple consensus: take the first successful result
	for _, result := range results {
		if result.Status == "completed" && result.Output != "" {
			return &consensusResult{
				Decision:     result.Output,
				Confidence:   result.Confidence,
				Participants: []string{result.AgentName},
				Rationale:    "First successful result selected",
				Metadata: map[string]string{
					"method":  "first_successful",
					"task_id": result.TaskID,
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("no successful results for consensus")
}

// isGoalAchieved checks if the current goal has been achieved
func (ae *AutonomousExecutor) isGoalAchieved(iteration *autonomousIteration) bool {
	// Simple check: if we have any completed assignments with output
	for _, result := range iteration.Results {
		if result.Status == "completed" && result.Output != "" {
			return true
		}
	}
	return false
}

// shouldTerminate checks if execution should terminate
func (ae *AutonomousExecutor) shouldTerminate(iteration *autonomousIteration) bool {
	// Check if we've exceeded the maximum duration
	if ae.config.TerminationConditions.MaxDuration > 0 {
		if len(ae.context.iterations) > 0 {
			firstIterationStart := time.Now().Add(-ae.context.iterations[0].Duration)
			if time.Since(firstIterationStart) > ae.config.TerminationConditions.MaxDuration {
				return true
			}
		}
	}

	// Check if we've had too many failed iterations
	failedCount := 0
	for _, iter := range ae.context.iterations {
		if !iter.Success {
			failedCount++
		}
	}

	return failedCount >= 3 // Terminate after 3 failed iterations
}

// shouldEvolveGoal checks if the goal should be evolved
func (ae *AutonomousExecutor) shouldEvolveGoal(iteration *autonomousIteration) bool {
	// Simple heuristic: evolve if we've had 2 failed iterations
	failedCount := 0
	for _, iter := range ae.context.iterations {
		if !iter.Success {
			failedCount++
		}
	}
	return failedCount >= 2
}

// evolveGoal evolves the current goal
func (ae *AutonomousExecutor) evolveGoal(iteration *autonomousIteration) {
	ae.context.mu.Lock()
	defer ae.context.mu.Unlock()

	oldGoal := ae.context.currentGoal
	ae.context.currentGoal = fmt.Sprintf("Refined approach to: %s", oldGoal)

	ae.logger.Info("Goal evolved from '%s' to '%s'", oldGoal, ae.context.currentGoal)
}

// createFinalResult creates the final workflow result
func (ae *AutonomousExecutor) createFinalResult(execCtx *ExecutionContext) (*hector.WorkflowResult, error) {
	// Aggregate all results from iterations
	allResults := make(map[string]*hector.AgentResult)
	totalTokens := 0

	for _, iteration := range ae.context.iterations {
		for _, assignment := range iteration.Results {
			if assignment.Status == "completed" {
				result := &hector.AgentResult{
					AgentName:  assignment.AgentName,
					StepName:   assignment.TaskID,
					Output:     assignment.Output,
					TokensUsed: assignment.TokensUsed,
					Duration:   assignment.Duration,
					Success:    true,
					Confidence: assignment.Confidence,
					Timestamp:  assignment.EndTime,
					Metadata: map[string]string{
						"input":      assignment.Input,
						"start_time": assignment.StartTime.Format(time.RFC3339),
					},
				}
				allResults[assignment.TaskID] = result
				totalTokens += assignment.TokensUsed
			}
		}
	}

	// Determine final status
	status := hector.WorkflowStatusCompleted
	if len(allResults) == 0 {
		status = hector.WorkflowStatusFailed
	}

	// Create final output
	finalOutput := ""
	if status == hector.WorkflowStatusCompleted {
		outputs := make([]string, 0, len(allResults))
		for _, result := range allResults {
			if result.Output != "" {
				outputs = append(outputs, result.Output)
			}
		}
		finalOutput = strings.Join(outputs, "\n")
	}

	// Get error message if any
	errorMsg := ""
	if execCtx.HasErrors() {
		errors := execCtx.GetErrors()
		if len(errors) > 0 {
			errorMsg = errors[0].Error()
		}
	}

	return &hector.WorkflowResult{
		WorkflowName:  execCtx.GetWorkflow().Name,
		Status:        status,
		FinalOutput:   finalOutput,
		AgentResults:  allResults,
		SharedContext: execCtx.GetAllSharedState(),
		ExecutionTime: execCtx.GetDuration(),
		TotalTokens:   totalTokens,
		Success:       status == hector.WorkflowStatusCompleted,
		Error:         errorMsg,
		StepsExecuted: len(allResults),
		Metadata: map[string]string{
			"iterations": fmt.Sprintf("%d", len(ae.context.iterations)),
		},
	}, nil
}
