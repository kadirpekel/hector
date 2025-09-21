package hector

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// DynamicReasoningContext holds the evolving context for dynamic reasoning
type DynamicReasoningContext struct {
	Query                string                   `json:"query"`
	OriginalGoal         string                   `json:"original_goal"`
	CurrentGoal          string                   `json:"current_goal"`
	GoalEvolutionHistory []string                 `json:"goal_evolution_history"`
	IterationResults     []DynamicIterationResult `json:"iteration_results"`
	SelfReflections      []SelfReflection         `json:"self_reflections"`
	MetaReasoning        []MetaReasoningStep      `json:"meta_reasoning"`
	AdaptationHistory    []AdaptationDecision     `json:"adaptation_history"`
	QualityMetrics       QualityMetrics           `json:"quality_metrics"`
	CurrentIteration     int                      `json:"current_iteration"`
	AvailableTools       []ToolInfo               `json:"available_tools"`
	DocumentContext      []string                 `json:"document_context"`
	ConversationContext  string                   `json:"conversation_context"`
	ShouldStop           bool                     `json:"should_stop"`
	StopReason           string                   `json:"stop_reason"`
}

// DynamicIterationResult represents the result of a single reasoning iteration
type DynamicIterationResult struct {
	IterationNumber    int                   `json:"iteration_number"`
	StepName           string                `json:"step_name"`
	StepType           string                `json:"step_type"`
	Input              string                `json:"input"`
	Output             string                `json:"output"`
	ToolsUsed          []string              `json:"tools_used"`
	ToolResults        map[string]ToolResult `json:"tool_results"`
	QualityScore       float64               `json:"quality_score"`
	GoalProgress       float64               `json:"goal_progress"`
	Confidence         float64               `json:"confidence"`
	TokensUsed         int                   `json:"tokens_used"`
	Duration           time.Duration         `json:"duration"`
	Success            bool                  `json:"success"`
	Error              string                `json:"error,omitempty"`
	SelfReflection     *SelfReflection       `json:"self_reflection,omitempty"`
	AdaptationNeeded   bool                  `json:"adaptation_needed"`
	NextStepSuggestion string                `json:"next_step_suggestion"`
}

// SelfReflection represents AI's evaluation of its own performance
type SelfReflection struct {
	IterationNumber        int      `json:"iteration_number"`
	PerformanceScore       float64  `json:"performance_score"`
	Strengths              []string `json:"strengths"`
	Weaknesses             []string `json:"weaknesses"`
	ImprovementSuggestions []string `json:"improvement_suggestions"`
	GoalAlignment          float64  `json:"goal_alignment"`
	EfficiencyScore        float64  `json:"efficiency_score"`
	QualityAssessment      string   `json:"quality_assessment"`
	ReflectionPrompt       string   `json:"reflection_prompt"`
	ReflectionResponse     string   `json:"reflection_response"`
}

// MetaReasoningStep represents AI reasoning about its reasoning process
type MetaReasoningStep struct {
	StepNumber         int      `json:"step_number"`
	ReasoningType      string   `json:"reasoning_type"` // "strategy_selection", "step_planning", "quality_evaluation"
	Input              string   `json:"input"`
	Analysis           string   `json:"analysis"`
	Decision           string   `json:"decision"`
	Rationale          string   `json:"rationale"`
	Confidence         float64  `json:"confidence"`
	AlternativeOptions []string `json:"alternative_options"`
}

// AdaptationDecision represents AI's decision to adapt its approach
type AdaptationDecision struct {
	IterationNumber     int     `json:"iteration_number"`
	Trigger             string  `json:"trigger"` // What caused the adaptation
	PreviousApproach    string  `json:"previous_approach"`
	NewApproach         string  `json:"new_approach"`
	Reasoning           string  `json:"reasoning"`
	ExpectedImprovement float64 `json:"expected_improvement"`
	Confidence          float64 `json:"confidence"`
}

// QualityMetrics tracks various quality indicators
type QualityMetrics struct {
	OverallQuality   float64   `json:"overall_quality"`
	GoalAlignment    float64   `json:"goal_alignment"`
	Completeness     float64   `json:"completeness"`
	Accuracy         float64   `json:"accuracy"`
	Efficiency       float64   `json:"efficiency"`
	Innovation       float64   `json:"innovation"`
	Consistency      float64   `json:"consistency"`
	UserSatisfaction float64   `json:"user_satisfaction"`
	Trend            string    `json:"trend"` // "improving", "declining", "stable"
	LastUpdated      time.Time `json:"last_updated"`
}

// DynamicStep represents a step created dynamically by AI
type DynamicStep struct {
	StepID            string                 `json:"step_id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Type              string                 `json:"type"`     // "analysis", "synthesis", "tool_execution", "evaluation"
	Priority          int                    `json:"priority"` // 1-10, higher is more important
	EstimatedDuration time.Duration          `json:"estimated_duration"`
	RequiredTools     []string               `json:"required_tools"`
	SuccessCriteria   []string               `json:"success_criteria"`
	InputDependencies []string               `json:"input_dependencies"` // Other step IDs this depends on
	OutputDescription string                 `json:"output_description"`
	CreatedBy         string                 `json:"created_by"` // "ai_planning", "ai_adaptation", "user_request"
	CreatedAt         time.Time              `json:"created_at"`
	Status            string                 `json:"status"` // "pending", "in_progress", "completed", "failed", "skipped"
	Config            map[string]interface{} `json:"config,omitempty"`
}

// DynamicReasoningEngine handles AI-driven dynamic reasoning
type DynamicReasoningEngine struct {
	agent   *Agent
	config  *DynamicReasoningConfig
	context *DynamicReasoningContext
}

// NewDynamicReasoningEngine creates a new dynamic reasoning engine
func NewDynamicReasoningEngine(agent *Agent, config *DynamicReasoningConfig) *DynamicReasoningEngine {
	config.SetDefaults()
	return &DynamicReasoningEngine{
		agent:  agent,
		config: config,
	}
}

// ExecuteDynamicReasoning performs AI-driven dynamic reasoning with document and conversation context
func (d *DynamicReasoningEngine) ExecuteDynamicReasoning(query string, context []string, modelNames ...string) (*AgentResponse, error) {
	d.agent.verbosePrint("Starting dynamic reasoning for query: %s\n", query)

	// Extract conversation context from enhanced query
	conversationContext := ""
	if strings.Contains(query, "Conversation Context:") {
		parts := strings.Split(query, "Conversation Context:")
		if len(parts) > 1 {
			conversationPart := parts[1]
			if strings.Contains(conversationPart, "\n\nOriginal question:") {
				conversationContext = strings.Split(conversationPart, "\n\nOriginal question:")[0]
			} else if strings.Contains(conversationPart, "\n\nTools used:") {
				conversationContext = strings.Split(conversationPart, "\n\nTools used:")[0]
			} else if strings.Contains(conversationPart, "\n\nTool Results:") {
				conversationContext = strings.Split(conversationPart, "\n\nTool Results:")[0]
			}
		}
	}

	// Initialize dynamic context
	d.context = &DynamicReasoningContext{
		Query:                query,
		OriginalGoal:         query,
		CurrentGoal:          query,
		GoalEvolutionHistory: []string{query},
		IterationResults:     []DynamicIterationResult{},
		SelfReflections:      []SelfReflection{},
		MetaReasoning:        []MetaReasoningStep{},
		AdaptationHistory:    []AdaptationDecision{},
		QualityMetrics:       QualityMetrics{},
		CurrentIteration:     0,
		AvailableTools:       d.agent.mcp.ListTools(),
		DocumentContext:      context,
		ConversationContext:  conversationContext,
		ShouldStop:           false,
	}

	// Main dynamic reasoning loop
	for d.context.CurrentIteration < d.config.MaxIterations && !d.context.ShouldStop {
		d.context.CurrentIteration++

		d.agent.verbosePrint("\n=== Dynamic Iteration %d/%d ===\n",
			d.context.CurrentIteration, d.config.MaxIterations)

		// Step 1: Meta-reasoning about current state
		if d.config.EnableMetaReasoning {
			d.performMetaReasoning()
		}

		// Step 2: Self-reflection on previous iteration
		if d.config.EnableSelfReflection && d.context.CurrentIteration > 1 {
			d.performSelfReflection()
		}

		// Step 3: Dynamic step planning
		nextStep := d.planNextStep()
		if nextStep == nil {
			d.context.ShouldStop = true
			d.context.StopReason = "No more steps planned"
			break
		}

		// Step 4: Execute the planned step
		result := d.executeDynamicStep(nextStep, modelNames...)
		d.context.IterationResults = append(d.context.IterationResults, result)

		// Step 5: Evaluate goal achievement
		goalAchieved, progress := d.evaluateGoalAchievement(result)

		// Step 6: Update quality metrics
		d.updateQualityMetrics(result)

		// Step 7: Check stopping conditions FIRST (before adaptation/evolution)
		if d.shouldStopReasoning(result, goalAchieved, progress) {
			d.context.ShouldStop = true
			d.context.StopReason = d.determineStopReason(result, goalAchieved, progress)
			break
		}

		// Step 8: Decide whether to adapt approach (only if not stopping)
		if d.shouldAdaptApproach(result) {
			d.adaptApproach(result)
		}

		// Step 9: Evolve goal if needed
		if d.config.EnableGoalEvolution {
			d.evolveGoal(result)
		}
	}

	// Generate final response
	return d.generateDynamicFinalResponse()
}

// ExecuteDynamicReasoningStreaming performs AI-driven dynamic reasoning with streaming, document and conversation context
func (d *DynamicReasoningEngine) ExecuteDynamicReasoningStreaming(query string, context []string, modelNames ...string) (<-chan string, error) {
	ch := make(chan string)

	go func() {
		defer close(ch)

		d.agent.verbosePrint("Starting dynamic reasoning streaming for query: %s\n", query)

		// Extract conversation context from enhanced query
		conversationContext := ""
		if strings.Contains(query, "Conversation Context:") {
			parts := strings.Split(query, "Conversation Context:")
			if len(parts) > 1 {
				conversationPart := parts[1]
				if strings.Contains(conversationPart, "\n\nOriginal question:") {
					conversationContext = strings.Split(conversationPart, "\n\nOriginal question:")[0]
				} else if strings.Contains(conversationPart, "\n\nTools used:") {
					conversationContext = strings.Split(conversationPart, "\n\nTools used:")[0]
				} else if strings.Contains(conversationPart, "\n\nTool Results:") {
					conversationContext = strings.Split(conversationPart, "\n\nTool Results:")[0]
				}
			}
		}

		// Initialize dynamic context
		d.context = &DynamicReasoningContext{
			Query:                query,
			OriginalGoal:         query,
			CurrentGoal:          query,
			GoalEvolutionHistory: []string{query},
			IterationResults:     []DynamicIterationResult{},
			SelfReflections:      []SelfReflection{},
			MetaReasoning:        []MetaReasoningStep{},
			AdaptationHistory:    []AdaptationDecision{},
			QualityMetrics:       QualityMetrics{},
			CurrentIteration:     0,
			AvailableTools:       d.agent.mcp.ListTools(),
			DocumentContext:      context,
			ConversationContext:  conversationContext,
			ShouldStop:           false,
		}

		// Main dynamic reasoning loop with streaming
		for d.context.CurrentIteration < d.config.MaxIterations && !d.context.ShouldStop {
			d.context.CurrentIteration++

			ch <- fmt.Sprintf("\n=== Dynamic Iteration %d/%d ===\n",
				d.context.CurrentIteration, d.config.MaxIterations)

			// Step 1: Meta-reasoning about current state
			if d.config.EnableMetaReasoning {
				ch <- "Performing meta-reasoning...\n"
				d.performMetaReasoning()
				if len(d.context.MetaReasoning) > 0 {
					lastMeta := d.context.MetaReasoning[len(d.context.MetaReasoning)-1]
					ch <- fmt.Sprintf("Meta-reasoning: %s - %s (confidence: %.2f)\n",
						lastMeta.ReasoningType, lastMeta.Decision, lastMeta.Confidence)
				}
			}

			// Step 2: Self-reflection on previous iteration
			if d.config.EnableSelfReflection && d.context.CurrentIteration > 1 {
				ch <- "Performing self-reflection...\n"
				d.performSelfReflection()
			}

			// Step 3: Dynamic step planning
			ch <- "Planning next step...\n"
			nextStep := d.planNextStep()
			if nextStep == nil {
				d.context.ShouldStop = true
				d.context.StopReason = "No more steps planned"
				ch <- "No more steps planned, stopping.\n"
				break
			}

			ch <- fmt.Sprintf("Executing step: %s (%s)\n", nextStep.Name, nextStep.Type)

			// Step 4: Execute the planned step
			result := d.executeDynamicStep(nextStep, modelNames...)
			d.context.IterationResults = append(d.context.IterationResults, result)

			ch <- fmt.Sprintf("Step completed: %s (quality: %.2f, confidence: %.2f, duration: %v)\n",
				nextStep.Name, result.QualityScore, result.Confidence, result.Duration)

			// Step 5: Evaluate goal achievement
			goalAchieved, progress := d.evaluateGoalAchievement(result)

			ch <- fmt.Sprintf("Goal evaluation: achieved=%t, progress=%.2f\n",
				goalAchieved, progress)

			// Step 6: Update quality metrics
			d.updateQualityMetrics(result)

			// Step 7: Check stopping conditions FIRST (before adaptation/evolution)
			if d.shouldStopReasoning(result, goalAchieved, progress) {
				d.context.ShouldStop = true
				d.context.StopReason = d.determineStopReason(result, goalAchieved, progress)
				ch <- fmt.Sprintf("Stopping: %s\n", d.context.StopReason)
				break
			}

			// Step 8: Decide whether to adapt approach (only if not stopping)
			if d.shouldAdaptApproach(result) {
				ch <- "Adapting approach based on AI decision...\n"
				d.adaptApproach(result)
			}

			// Step 9: Evolve goal if needed (only if not stopping)
			if d.config.EnableGoalEvolution {
				d.evolveGoal(result)
			}
		}

		ch <- "\n=== Dynamic Reasoning Complete ===\n"
		ch <- fmt.Sprintf("Total iterations: %d\n", len(d.context.IterationResults))
		ch <- fmt.Sprintf("Final quality: %.2f\n", d.context.QualityMetrics.OverallQuality)
		ch <- fmt.Sprintf("Stop reason: %s\n", d.context.StopReason)

		// Generate and stream the final response
		ch <- "\n=== Final Answer ===\n"
		d.generateDynamicFinalResponseStreaming(ch)
	}()

	return ch, nil
}

// performMetaReasoning performs AI reasoning about the reasoning process
func (d *DynamicReasoningEngine) performMetaReasoning() {
	prompt := fmt.Sprintf(`You are an AI reasoning about your own reasoning process. Analyze the current state and decide what to do next.

Current Context:
- Query: %s
- Current Goal: %s
- Iteration: %d/%d
- Previous Results: %d iterations completed
- Quality Metrics: %+v
- Available Tools: %d tools

Conversation Context:
%s

Document Context:
%s

Previous Meta-Reasoning Steps:
%s

Analyze the current situation and provide your reasoning about:
1. What type of reasoning is needed now?
2. What should be the next step?
3. How confident are you in this decision?
4. What alternative approaches exist?
5. How can the conversation and document context inform your reasoning?

Respond with a JSON object containing:
{
  "reasoning_type": "strategy_selection|step_planning|quality_evaluation|adaptation_decision",
  "analysis": "Your analysis of the current state",
  "decision": "Your decision about what to do next",
  "rationale": "Why you made this decision",
  "confidence": 0.0-1.0,
  "alternative_options": ["option1", "option2"]
}`,
		d.context.Query,
		d.context.CurrentGoal,
		d.context.CurrentIteration,
		d.config.MaxIterations,
		len(d.context.IterationResults),
		d.context.QualityMetrics,
		len(d.context.AvailableTools),
		d.formatConversationContext(),
		d.formatDocumentContext(),
		d.formatContextHistory())

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Meta-reasoning failed: %v\n", err)
		return
	}

	// Clean the response first to handle backticks and formatting
	cleanedResponse := d.cleanJSONResponse(response)

	var metaStep MetaReasoningStep
	if err := json.Unmarshal([]byte(cleanedResponse), &metaStep); err != nil {
		d.agent.verbosePrint("Failed to parse meta-reasoning response: %v\n", err)
		d.agent.verbosePrint("Raw response: %s\n", response)
		return
	}

	metaStep.StepNumber = d.context.CurrentIteration
	d.context.MetaReasoning = append(d.context.MetaReasoning, metaStep)

	d.agent.verbosePrint("Meta-reasoning: %s - %s (confidence: %.2f)\n",
		metaStep.ReasoningType, metaStep.Decision, metaStep.Confidence)
}

// planNextStep uses AI to dynamically plan the next reasoning step
func (d *DynamicReasoningEngine) planNextStep() *DynamicStep {
	prompt := fmt.Sprintf(`You are an AI planning the next step in dynamic reasoning. Based on the current context, create a specific step to execute.

Current Context:
- Query: %s
- Current Goal: %s
- Iteration: %d/%d
- Previous Steps: %d completed
- Last Result Quality: %.2f
- Goal Progress: %.2f

Conversation Context:
%s

Document Context:
%s

Previous Steps:
%s

Available Tools:
%s

Based on this context, create the next reasoning step. Consider:
1. What specific action is needed?
2. What tools might be required?
3. What are the success criteria?
4. What dependencies exist?
5. How can the conversation and document context inform this step?

IMPORTANT: Use Go duration format for estimated_duration (e.g., "30s", "5m", "1h0m0s", "2h30m")

Respond with a JSON object:
{
  "step_id": "unique_id",
  "name": "Step name",
  "description": "What this step does",
  "type": "analysis|synthesis|tool_execution|evaluation|research|creative",
  "priority": 1-10,
  "estimated_duration": "30s", // Duration in Go format: "30s", "5m", "1h0m0s", "2h30m", etc.
  "required_tools": ["tool1", "tool2"],
  "success_criteria": ["criterion1", "criterion2"],
  "input_dependencies": ["step_id1"],
  "output_description": "What this step will produce",
  "created_by": "ai_planning"
}

If no more steps are needed, respond with: {"step_id": "none", "name": "stop"}`,
		d.context.Query,
		d.context.CurrentGoal,
		d.context.CurrentIteration,
		d.config.MaxIterations,
		len(d.context.IterationResults),
		d.getLastQualityScore(),
		d.getGoalProgress(),
		d.formatConversationContext(),
		d.formatDocumentContext(),
		d.formatContextHistory(),
		d.formatAvailableTools())

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Step planning failed: %v\n", err)
		return nil
	}

	// Parse LLM response using simple structure first
	cleanedResponse := d.cleanJSONResponse(response)

	// Parse as simple structure first
	var simpleStep struct {
		StepID            string   `json:"step_id"`
		Name              string   `json:"name"`
		Description       string   `json:"description"`
		Type              string   `json:"type"`
		Priority          int      `json:"priority"`
		EstimatedDuration string   `json:"estimated_duration"`
		RequiredTools     []string `json:"required_tools"`
		SuccessCriteria   []string `json:"success_criteria"`
		InputDependencies []string `json:"input_dependencies"`
		OutputDescription string   `json:"output_description"`
		CreatedBy         string   `json:"created_by"`
	}

	if err := json.Unmarshal([]byte(cleanedResponse), &simpleStep); err != nil {
		d.agent.verbosePrint("Failed to parse step planning response: %v\n", err)
		d.agent.verbosePrint("Raw response: %s\n", response)
		return nil
	}

	// Convert to DynamicStep
	var step DynamicStep
	step.StepID = simpleStep.StepID
	step.Name = simpleStep.Name
	step.Description = simpleStep.Description
	step.Type = simpleStep.Type
	step.Priority = simpleStep.Priority
	step.RequiredTools = simpleStep.RequiredTools
	step.SuccessCriteria = simpleStep.SuccessCriteria
	step.InputDependencies = simpleStep.InputDependencies
	step.OutputDescription = simpleStep.OutputDescription
	step.CreatedBy = simpleStep.CreatedBy

	// Parse duration (AI should generate in Go format like "30s", "5m", "1h0m0s")
	if simpleStep.EstimatedDuration != "" {
		duration, err := time.ParseDuration(simpleStep.EstimatedDuration)
		if err != nil {
			d.agent.verbosePrint("Failed to parse duration '%s': %v, using default\n", simpleStep.EstimatedDuration, err)
			step.EstimatedDuration = 30 * time.Second // Default
		} else {
			step.EstimatedDuration = duration
		}
	} else {
		step.EstimatedDuration = 30 * time.Second // Default
	}

	if step.StepID == "none" {
		return nil
	}

	step.CreatedAt = time.Now()
	step.Status = "pending"
	// Store the step for reference
	// (Step is already stored in IterationResults)

	d.agent.verbosePrint("Planned step: %s (%s) - Priority: %d\n",
		step.Name, step.Type, step.Priority)

	return &step
}

// executeDynamicStep executes a dynamically planned step
func (d *DynamicReasoningEngine) executeDynamicStep(step *DynamicStep, modelNames ...string) DynamicIterationResult {
	startTime := time.Now()
	step.Status = "in_progress"

	d.agent.verbosePrint("Executing step: %s\n", step.Name)

	result := DynamicIterationResult{
		IterationNumber: d.context.CurrentIteration,
		StepName:        step.Name,
		StepType:        step.Type,
		Input:           d.context.CurrentGoal,
		ToolsUsed:       []string{},
		ToolResults:     make(map[string]ToolResult),
		Success:         false,
	}

	// Build step-specific prompt
	prompt := d.buildDynamicStepPrompt(step)

	// Execute step based on type
	var output string
	var tokensUsed int
	var err error

	switch step.Type {
	case "analysis":
		output, tokensUsed, err = d.executeAnalysisStep(step, prompt)
	case "synthesis":
		output, tokensUsed, err = d.executeSynthesisStep(step, prompt)
	case "tool_execution":
		output, tokensUsed, err = d.executeToolStep(step, prompt)
	case "evaluation":
		output, tokensUsed, err = d.executeEvaluationStep(step, prompt)
	case "research":
		output, tokensUsed, err = d.executeResearchStep(step, prompt)
	case "creative":
		output, tokensUsed, err = d.executeCreativeStep(step, prompt)
	default:
		err = fmt.Errorf("Unknown step type: %s", step.Type)
	}

	result.Output = output
	result.TokensUsed = tokensUsed
	if err != nil {
		result.Error = err.Error()
	}

	result.Duration = time.Since(startTime)
	result.Success = result.Error == ""

	// Evaluate step quality
	result.QualityScore = d.evaluateStepQuality(result)
	result.Confidence = d.calculateStepConfidence(result)

	step.Status = "completed"
	if !result.Success {
		step.Status = "failed"
	}

	d.agent.verbosePrint("Step completed: %s (quality: %.2f, confidence: %.2f, duration: %v)\n",
		step.Name, result.QualityScore, result.Confidence, result.Duration)

	return result
}

// evaluateGoalAchievement uses AI to evaluate if the goal has been achieved
func (d *DynamicReasoningEngine) evaluateGoalAchievement(result DynamicIterationResult) (bool, float64) {
	prompt := fmt.Sprintf(`You are an AI evaluating goal achievement. Analyze the current state and determine if the goal has been achieved.

Original Goal: %s
Current Goal: %s
Latest Result: %s
All Results: %d iterations

Conversation Context:
%s

Document Context:
%s

Evaluate:
1. Has the original goal been achieved? (true/false)
2. What is the progress percentage? (0.0-1.0)
3. What evidence supports your assessment?
4. What gaps remain?
5. Does the conversation context contain information that satisfies the goal?

IMPORTANT: If the conversation context already contains the information being requested (like location, preferences, etc.), consider the goal achieved.

Respond with JSON:
{
  "goal_achieved": true/false,
  "progress": 0.0-1.0,
  "evidence": "Evidence supporting the assessment",
  "remaining_gaps": ["gap1", "gap2"],
  "confidence": 0.0-1.0
}`,
		d.context.OriginalGoal,
		d.context.CurrentGoal,
		result.Output,
		len(d.context.IterationResults),
		d.formatConversationContext(),
		d.formatDocumentContext())

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Goal evaluation failed: %v\n", err)
		return false, 0.0
	}

	var evaluation struct {
		GoalAchieved  bool     `json:"goal_achieved"`
		Progress      float64  `json:"progress"`
		Evidence      string   `json:"evidence"`
		RemainingGaps []string `json:"remaining_gaps"`
		Confidence    float64  `json:"confidence"`
	}

	// Clean the response first to handle backticks and formatting
	cleanedResponse := d.cleanJSONResponse(response)

	if err := json.Unmarshal([]byte(cleanedResponse), &evaluation); err != nil {
		d.agent.verbosePrint("Failed to parse goal evaluation: %v\n", err)
		d.agent.verbosePrint("Raw response: %s\n", response)
		return false, 0.0
	}

	d.agent.verbosePrint("Goal evaluation: achieved=%t, progress=%.2f, confidence=%.2f\n",
		evaluation.GoalAchieved, evaluation.Progress, evaluation.Confidence)

	return evaluation.GoalAchieved, evaluation.Progress
}

// Helper methods for formatting context information

// formatConversationContext formats conversation context for AI prompts
func (d *DynamicReasoningEngine) formatConversationContext() string {
	if d.context.ConversationContext == "" {
		return "No conversation context available"
	}
	return d.context.ConversationContext
}

// formatDocumentContext formats document context for AI prompts
func (d *DynamicReasoningEngine) formatDocumentContext() string {
	if len(d.context.DocumentContext) == 0 {
		return "No document context available"
	}

	var parts []string
	for i, doc := range d.context.DocumentContext {
		parts = append(parts, fmt.Sprintf("Document %d: %s", i+1, doc))
	}
	return strings.Join(parts, "\n")
}

func (d *DynamicReasoningEngine) formatAvailableTools() string {
	if len(d.context.AvailableTools) == 0 {
		return "None"
	}

	var parts []string
	for _, tool := range d.context.AvailableTools {
		parts = append(parts, fmt.Sprintf("- %s: %s", tool.Name, tool.Description))
	}
	return strings.Join(parts, "\n")
}

func (d *DynamicReasoningEngine) getLastQualityScore() float64 {
	if len(d.context.IterationResults) == 0 {
		return 0.0
	}
	return d.context.IterationResults[len(d.context.IterationResults)-1].QualityScore
}

func (d *DynamicReasoningEngine) getGoalProgress() float64 {
	if len(d.context.IterationResults) == 0 {
		return 0.0
	}
	return d.context.IterationResults[len(d.context.IterationResults)-1].GoalProgress
}

// Placeholder methods for step execution (to be implemented)
func (d *DynamicReasoningEngine) buildDynamicStepPrompt(step *DynamicStep) string {
	return fmt.Sprintf("Execute step: %s\nDescription: %s\nContext: %s",
		step.Name, step.Description, d.context.CurrentGoal)
}

func (d *DynamicReasoningEngine) executeAnalysisStep(step *DynamicStep, prompt string) (string, int, error) {
	// Implementation for analysis steps
	enhancedPrompt := fmt.Sprintf(`You are performing an analysis step in dynamic reasoning.

Step: %s
Description: %s
Context: %s

%s

Provide a detailed analysis that:
1. Breaks down the problem into components
2. Identifies key factors and relationships
3. Examines different perspectives
4. Highlights important patterns or insights
5. Sets up the foundation for next steps

Be thorough, analytical, and insightful.`,
		step.Name, step.Description, d.context.CurrentGoal, prompt)

	return d.agent.llm.Generate(enhancedPrompt)
}

func (d *DynamicReasoningEngine) executeSynthesisStep(step *DynamicStep, prompt string) (string, int, error) {
	// Implementation for synthesis steps
	enhancedPrompt := fmt.Sprintf(`You are performing a synthesis step in dynamic reasoning.

Step: %s
Description: %s
Context: %s

Previous Results:
%s

%s

Synthesize the information by:
1. Combining insights from previous steps
2. Creating coherent conclusions
3. Identifying overarching themes
4. Building toward a comprehensive understanding
5. Preparing for final response or next steps

Be integrative, comprehensive, and forward-looking.`,
		step.Name, step.Description, d.context.CurrentGoal, d.formatContextHistory(), prompt)

	return d.agent.llm.Generate(enhancedPrompt)
}

func (d *DynamicReasoningEngine) executeToolStep(step *DynamicStep, prompt string) (string, int, error) {
	// Implementation for tool execution steps
	enhancedPrompt := fmt.Sprintf(`You are performing a tool execution step in dynamic reasoning.

Step: %s
Description: %s
Context: %s

Available Tools:
%s

Required Tools: %v

%s

Execute the necessary tools and provide:
1. Tool selection rationale
2. Tool execution results
3. Analysis of tool outputs
4. Integration with overall reasoning
5. Next steps based on tool results

Be strategic about tool usage and thorough in analysis.`,
		step.Name, step.Description, d.context.CurrentGoal, d.formatAvailableTools(), step.RequiredTools, prompt)

	return d.agent.llm.Generate(enhancedPrompt)
}

func (d *DynamicReasoningEngine) executeEvaluationStep(step *DynamicStep, prompt string) (string, int, error) {
	// Implementation for evaluation steps
	enhancedPrompt := fmt.Sprintf(`You are performing an evaluation step in dynamic reasoning.

Step: %s
Description: %s
Context: %s

Previous Results:
%s

%s

Evaluate by:
1. Assessing quality and completeness
2. Identifying strengths and weaknesses
3. Measuring progress toward goals
4. Validating conclusions and assumptions
5. Recommending improvements or next steps

Be critical, objective, and constructive.`,
		step.Name, step.Description, d.context.CurrentGoal, d.formatContextHistory(), prompt)

	return d.agent.llm.Generate(enhancedPrompt)
}

func (d *DynamicReasoningEngine) executeResearchStep(step *DynamicStep, prompt string) (string, int, error) {
	// Implementation for research steps
	enhancedPrompt := fmt.Sprintf(`You are performing a research step in dynamic reasoning.

Step: %s
Description: %s
Context: %s

%s

Research by:
1. Gathering relevant information
2. Exploring different sources and perspectives
3. Investigating key questions
4. Analyzing findings and patterns
5. Synthesizing research insights

Be thorough, systematic, and evidence-based.`,
		step.Name, step.Description, d.context.CurrentGoal, prompt)

	return d.agent.llm.Generate(enhancedPrompt)
}

func (d *DynamicReasoningEngine) executeCreativeStep(step *DynamicStep, prompt string) (string, int, error) {
	// Implementation for creative steps
	enhancedPrompt := fmt.Sprintf(`You are performing a creative step in dynamic reasoning.

Step: %s
Description: %s
Context: %s

%s

Be creative by:
1. Generating novel ideas and solutions
2. Exploring unconventional approaches
3. Making unexpected connections
4. Thinking outside conventional boundaries
5. Innovating and imagining possibilities

Be imaginative, original, and innovative.`,
		step.Name, step.Description, d.context.CurrentGoal, prompt)

	return d.agent.llm.Generate(enhancedPrompt)
}

func (d *DynamicReasoningEngine) evaluateStepQuality(result DynamicIterationResult) float64 {
	// AI-based quality evaluation
	prompt := fmt.Sprintf(`Evaluate the quality of this reasoning step result:

Step: %s (%s)
Output: %s
Duration: %v
Tools Used: %v

Rate the quality from 0.0 to 1.0 considering:
- Relevance to the goal
- Completeness of the response
- Accuracy and correctness
- Efficiency
- Innovation

Respond with just a number between 0.0 and 1.0.`,
		result.StepName, result.StepType, result.Output, result.Duration, result.ToolsUsed)

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Quality evaluation failed: %v\n", err)
		return 0.0 // Let AI handle low quality cases
	}

	// Parse the response as a float
	var quality float64
	fmt.Sscanf(response, "%f", &quality)
	return quality
}

func (d *DynamicReasoningEngine) calculateStepConfidence(result DynamicIterationResult) float64 {
	// AI-based confidence calculation
	prompt := fmt.Sprintf(`Calculate confidence in this reasoning step result:

Step: %s (%s)
Output: %s
Quality Score: %.2f

Rate confidence from 0.0 to 1.0 considering:
- Certainty in the result
- Evidence supporting the conclusion
- Potential for error
- Consistency with previous results

Respond with just a number between 0.0 and 1.0.`,
		result.StepName, result.StepType, result.Output, result.QualityScore)

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Confidence calculation failed: %v\n", err)
		return 0.0 // Let AI handle low confidence cases
	}

	var confidence float64
	fmt.Sscanf(response, "%f", &confidence)
	return confidence
}

func (d *DynamicReasoningEngine) updateQualityMetrics(result DynamicIterationResult) {
	// Update quality metrics based on the latest result
	d.context.QualityMetrics.OverallQuality = d.calculateOverallQuality()
	d.context.QualityMetrics.GoalAlignment = d.calculateGoalAlignment()
	d.context.QualityMetrics.Completeness = d.calculateCompleteness()
	d.context.QualityMetrics.Efficiency = d.calculateEfficiency()
	d.context.QualityMetrics.LastUpdated = time.Now()
}

func (d *DynamicReasoningEngine) shouldAdaptApproach(result DynamicIterationResult) bool {
	// Pure AI decision on whether to adapt approach
	prompt := fmt.Sprintf(`You are an AI evaluating whether your current reasoning approach needs adaptation.

Current Context:
- Query: %s
- Current Goal: %s
- Iteration: %d/%d
- Latest Result Quality: %.2f
- Previous Results: %d iterations
- Current Approach: %s

Previous Adaptation History:
%s

Based on your analysis, should you adapt your reasoning approach? Consider:
1. Is the current approach working effectively?
2. Are you making sufficient progress toward the goal?
3. Would a different approach yield better results?
4. What specific changes might help?

Respond with JSON:
{
  "should_adapt": true/false,
  "reasoning": "Your analysis of why adaptation is/isn't needed",
  "confidence": 0.0-1.0,
  "suggested_changes": ["change1", "change2"] (if adapting)
}`,
		d.context.Query,
		d.context.CurrentGoal,
		d.context.CurrentIteration,
		d.config.MaxIterations,
		result.QualityScore,
		len(d.context.IterationResults),
		d.getCurrentApproachDescription(),
		d.formatContextHistory())

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Adaptation decision failed: %v\n", err)
		return false
	}

	var decision struct {
		ShouldAdapt      bool     `json:"should_adapt"`
		Reasoning        string   `json:"reasoning"`
		Confidence       float64  `json:"confidence"`
		SuggestedChanges []string `json:"suggested_changes"`
	}

	cleanedResponse := d.cleanJSONResponse(response)
	if err := json.Unmarshal([]byte(cleanedResponse), &decision); err != nil {
		d.agent.verbosePrint("Failed to parse adaptation decision: %v\n", err)
		return false
	}

	d.agent.verbosePrint("AI Adaptation Decision: %t (confidence: %.2f) - %s\n",
		decision.ShouldAdapt, decision.Confidence, decision.Reasoning)

	return decision.ShouldAdapt
}

func (d *DynamicReasoningEngine) adaptApproach(result DynamicIterationResult) {
	// AI-driven approach adaptation
	d.agent.verbosePrint("Adapting approach based on AI decision\n")

	prompt := fmt.Sprintf(`You are an AI adapting your reasoning approach. The current approach is not working well.

Current Context:
- Query: %s
- Current Goal: %s
- Iteration: %d
- Quality Score: %.2f
- Previous Steps: %d completed
- Last Step: %s (%s)

Previous Approaches Tried:
%s

Analyze why the current approach is failing and suggest a new approach. Consider:
1. What went wrong with the current approach?
2. What alternative strategies could work better?
3. How should the reasoning process change?

Respond with JSON:
{
  "analysis": "Why current approach failed",
  "new_approach": "Description of new approach",
  "reasoning": "Why this new approach should work",
  "expected_improvement": 0.0-1.0,
  "confidence": 0.0-1.0
}`,
		d.context.Query,
		d.context.CurrentGoal,
		d.context.CurrentIteration,
		result.QualityScore,
		len(d.context.IterationResults),
		result.StepName,
		result.StepType,
		d.formatContextHistory())

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Approach adaptation failed: %v\n", err)
		return
	}

	// Clean the response first to handle backticks and formatting
	cleanedResponse := d.cleanJSONResponse(response)

	var adaptation AdaptationDecision
	if err := json.Unmarshal([]byte(cleanedResponse), &adaptation); err != nil {
		d.agent.verbosePrint("Failed to parse adaptation response: %v\n", err)
		d.agent.verbosePrint("Raw response: %s\n", response)
		return
	}

	adaptation.IterationNumber = d.context.CurrentIteration
	adaptation.Trigger = fmt.Sprintf("AI decision to adapt based on quality: %.2f", result.QualityScore)
	adaptation.PreviousApproach = fmt.Sprintf("Step: %s (%s)", result.StepName, result.StepType)

	d.context.AdaptationHistory = append(d.context.AdaptationHistory, adaptation)

	d.agent.verbosePrint("Adapted approach: %s (expected improvement: %.2f)\n",
		adaptation.NewApproach, adaptation.ExpectedImprovement)
}

func (d *DynamicReasoningEngine) shouldStopReasoning(result DynamicIterationResult, goalAchieved bool, progress float64) bool {
	d.agent.verbosePrint("shouldStopReasoning called with: goalAchieved=%t, progress=%.2f\n", goalAchieved, progress)

	// CRITICAL: If goal is achieved with high progress, stop immediately
	if goalAchieved && progress >= 0.9 {
		d.agent.verbosePrint("CRITICAL: Goal achieved with high progress (%.2f) - stopping immediately\n", progress)
		return true
	}

	// If goal is achieved with any progress, also stop
	if goalAchieved {
		d.agent.verbosePrint("Goal achieved (progress=%.2f) - stopping\n", progress)
		return true
	}

	d.agent.verbosePrint("Goal not achieved, asking AI for decision\n")

	// Pure AI decision on whether to stop reasoning
	prompt := fmt.Sprintf(`You are an AI evaluating whether to continue or stop your reasoning process.

Current Context:
- Query: %s
- Current Goal: %s
- Iteration: %d/%d
- Goal Achieved: %t
- Progress: %.2f
- Latest Quality: %.2f
- Total Results: %d iterations

Conversation Context:
%s

Previous Results Summary:
%s

Based on your analysis, should you stop reasoning now? Consider:
1. Have you achieved the original goal?
2. Is the current progress sufficient?
3. Are you making meaningful progress?
4. Would continuing yield diminishing returns?
5. What would be the optimal stopping point?
6. Are you repeating the same approach without progress?
7. Should you try a completely different approach instead of stopping?
8. Do you need different tools or resources?
9. Is the goal already achieved but not recognized?

IMPORTANT: Be decisive about stopping. If you have provided a reasonable response to the user's query, consider stopping rather than continuing indefinitely. Users prefer concise, helpful responses over lengthy reasoning processes.

Also consider stopping if:
- You have provided a comprehensive response to the user's query
- The user's question has been adequately addressed
- Continuing would be repetitive or unhelpful
- You have offered practical support or suggestions

Respond with JSON:
{
  "should_stop": true/false,
  "reasoning": "Your analysis of why to stop/continue",
  "confidence": 0.0-1.0,
  "stop_reason": "Goal achieved|Sufficient progress|Quality threshold|Diminishing returns|Max iterations|Need different approach|Need different tools|Goal already achieved|Comprehensive response provided" (if stopping)
}`,
		d.context.Query,
		d.context.CurrentGoal,
		d.context.CurrentIteration,
		d.config.MaxIterations,
		goalAchieved,
		progress,
		result.QualityScore,
		len(d.context.IterationResults),
		d.formatConversationContext(),
		d.formatContextHistory())

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Stopping decision failed: %v\n", err)
		return false // Let AI continue reasoning
	}

	var decision struct {
		ShouldStop bool    `json:"should_stop"`
		Reasoning  string  `json:"reasoning"`
		Confidence float64 `json:"confidence"`
		StopReason string  `json:"stop_reason"`
	}

	cleanedResponse := d.cleanJSONResponse(response)
	if err := json.Unmarshal([]byte(cleanedResponse), &decision); err != nil {
		d.agent.verbosePrint("Failed to parse stopping decision: %v\n", err)
		return false // Let AI continue reasoning
	}

	d.agent.verbosePrint("AI Stopping Decision: %t (confidence: %.2f) - %s\n",
		decision.ShouldStop, decision.Confidence, decision.Reasoning)

	d.agent.verbosePrint("Final shouldStopReasoning result: %t\n", decision.ShouldStop)
	return decision.ShouldStop
}

func (d *DynamicReasoningEngine) determineStopReason(result DynamicIterationResult, goalAchieved bool, progress float64) string {
	// Pure AI-driven stop reason determination
	prompt := fmt.Sprintf(`You are an AI determining the reason for stopping your reasoning process.

Current Context:
- Query: %s
- Goal Achieved: %t
- Progress: %.2f
- Quality Score: %.2f
- Iteration: %d/%d

Based on your analysis, what is the primary reason for stopping? Consider:
1. Goal achievement status
2. Progress made
3. Quality of results
4. Iteration count
5. Whether you need a different approach
6. Whether you need different tools
7. Whether the goal is already achieved but not recognized

Respond with a single, clear reason:
"Goal achieved" | "Sufficient progress" | "Quality threshold reached" | "Diminishing returns" | "Max iterations" | "Need different approach" | "Need different tools" | "Goal already achieved"`,
		d.context.Query,
		goalAchieved,
		progress,
		result.QualityScore,
		d.context.CurrentIteration,
		d.config.MaxIterations)

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Stop reason determination failed: %v\n", err)
		return "AI decision failed"
	}

	// Clean and return the response
	cleaned := strings.TrimSpace(response)
	if cleaned == "" {
		return "AI decision unclear"
	}
	return cleaned
}

func (d *DynamicReasoningEngine) evolveGoal(result DynamicIterationResult) {
	// AI-driven goal evolution
	d.agent.verbosePrint("Evolving goal based on results\n")

	prompt := fmt.Sprintf(`You are an AI that can evolve its goals based on new understanding. Analyze if the goal should change.

Current Context:
- Original Goal: %s
- Current Goal: %s
- Latest Result: %s
- Quality Score: %.2f
- Goal Progress: %.2f

Goal Evolution History:
%s

Consider:
1. Has new information changed what we should be trying to achieve?
2. Is the current goal still the best way to address the original query?
3. Should we refine, expand, or redirect the goal?

Respond with JSON:
{
  "should_evolve": true/false,
  "new_goal": "Updated goal if evolving",
  "reasoning": "Why the goal should/shouldn't change",
  "confidence": 0.0-1.0
}

If should_evolve is false, new_goal should be the current goal.`,
		d.context.OriginalGoal,
		d.context.CurrentGoal,
		result.Output,
		result.QualityScore,
		result.GoalProgress,
		strings.Join(d.context.GoalEvolutionHistory, " -> "))

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Goal evolution failed: %v\n", err)
		return
	}

	var evolution struct {
		ShouldEvolve bool    `json:"should_evolve"`
		NewGoal      string  `json:"new_goal"`
		Reasoning    string  `json:"reasoning"`
		Confidence   float64 `json:"confidence"`
	}

	// Clean the response first to handle backticks and formatting
	cleanedResponse := d.cleanJSONResponse(response)

	if err := json.Unmarshal([]byte(cleanedResponse), &evolution); err != nil {
		d.agent.verbosePrint("Failed to parse goal evolution response: %v\n", err)
		d.agent.verbosePrint("Raw response: %s\n", response)
		return
	}

	if evolution.ShouldEvolve && evolution.NewGoal != d.context.CurrentGoal {
		d.context.GoalEvolutionHistory = append(d.context.GoalEvolutionHistory, evolution.NewGoal)
		d.context.CurrentGoal = evolution.NewGoal
		d.agent.verbosePrint("Goal evolved: %s -> %s (reasoning: %s)\n",
			d.context.CurrentGoal, evolution.NewGoal, evolution.Reasoning)
	} else {
		d.agent.verbosePrint("Goal unchanged: %s (reasoning: %s)\n",
			d.context.CurrentGoal, evolution.Reasoning)
	}
}

func (d *DynamicReasoningEngine) performSelfReflection() {
	// AI self-reflection implementation
	d.agent.verbosePrint("Performing self-reflection\n")

	if len(d.context.IterationResults) == 0 {
		return
	}

	lastResult := d.context.IterationResults[len(d.context.IterationResults)-1]

	prompt := fmt.Sprintf(`You are an AI reflecting on your own performance. Analyze your recent reasoning step.

Recent Step Analysis:
- Step: %s (%s)
- Output: %s
- Quality Score: %.2f
- Confidence: %.2f
- Duration: %v
- Success: %t

Previous Performance:
%s

Overall Context:
- Query: %s
- Current Goal: %s
- Iteration: %d/%d
- Total Steps: %d

Reflect on:
1. What did you do well in this step?
2. What could have been improved?
3. How does this step contribute to the overall goal?
4. What patterns do you notice in your performance?
5. How can you improve in the next iteration?

Respond with JSON:
{
  "performance_score": 0.0-1.0,
  "strengths": ["strength1", "strength2"],
  "weaknesses": ["weakness1", "weakness2"],
  "improvement_suggestions": ["suggestion1", "suggestion2"],
  "goal_alignment": 0.0-1.0,
  "efficiency_score": 0.0-1.0,
  "quality_assessment": "Detailed assessment of quality",
  "reflection_prompt": "The prompt you used for reflection",
  "reflection_response": "Your reflection response"
}`,
		lastResult.StepName,
		lastResult.StepType,
		lastResult.Output,
		lastResult.QualityScore,
		lastResult.Confidence,
		lastResult.Duration,
		lastResult.Success,
		d.formatContextHistory(),
		d.context.Query,
		d.context.CurrentGoal,
		d.context.CurrentIteration,
		d.config.MaxIterations,
		len(d.context.IterationResults))

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Self-reflection failed: %v\n", err)
		return
	}

	// Clean the response first to handle backticks and formatting
	cleanedResponse := d.cleanJSONResponse(response)

	var reflection SelfReflection
	if err := json.Unmarshal([]byte(cleanedResponse), &reflection); err != nil {
		d.agent.verbosePrint("Failed to parse self-reflection response: %v\n", err)
		d.agent.verbosePrint("Raw response: %s\n", response)
		return
	}

	reflection.IterationNumber = d.context.CurrentIteration
	reflection.ReflectionPrompt = prompt
	reflection.ReflectionResponse = response

	d.context.SelfReflections = append(d.context.SelfReflections, reflection)

	d.agent.verbosePrint("Self-reflection completed: performance=%.2f, goal_alignment=%.2f\n",
		reflection.PerformanceScore, reflection.GoalAlignment)
}

func (d *DynamicReasoningEngine) generateDynamicFinalResponse() (*AgentResponse, error) {
	// Generate final response from dynamic reasoning results
	d.agent.verbosePrint("Generating final dynamic response\n")

	if len(d.context.IterationResults) == 0 {
		return &AgentResponse{
			Answer:     "No reasoning steps completed",
			Confidence: 0.0,
		}, nil
	}

	// Format context once for efficiency
	contextHistory := d.formatContextHistory()

	// Build comprehensive final response using AI
	prompt := fmt.Sprintf(`You are an AI generating a final comprehensive response from dynamic reasoning results.

Original Query: %s
Final Goal: %s
Total Iterations: %d
Final Quality: %.2f
Stop Reason: %s

Conversation Context:
%s

Document Context:
%s

Context History:
%s

Goal Evolution: %s

Generate a comprehensive final response that:
1. Directly answers the original query
2. Synthesizes insights from all reasoning steps
3. Highlights key findings and conclusions
4. Shows the reasoning process and adaptations made
5. Provides confidence in the final answer
6. Incorporates relevant information from conversation and document context

Make the response clear, well-structured, and comprehensive.`,
		d.context.Query,
		d.context.CurrentGoal,
		len(d.context.IterationResults),
		d.context.QualityMetrics.OverallQuality,
		d.context.StopReason,
		d.formatConversationContext(),
		d.formatDocumentContext(),
		contextHistory,
		strings.Join(d.context.GoalEvolutionHistory, " -> "))

	answer, tokensUsed, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Final response generation failed: %v\n", err)
		// Fallback to simple concatenation
		var answerBuilder strings.Builder
		for _, result := range d.context.IterationResults {
			if result.Success {
				answerBuilder.WriteString(fmt.Sprintf("%s: %s\n", result.StepName, result.Output))
			}
		}
		answer = answerBuilder.String()
		tokensUsed = 0
	}

	// Calculate total tokens used
	totalTokens := tokensUsed
	for _, result := range d.context.IterationResults {
		totalTokens += result.TokensUsed
	}

	return &AgentResponse{
		Answer:         answer,
		Confidence:     d.context.QualityMetrics.OverallQuality,
		TokensUsed:     totalTokens,
		ReasoningSteps: d.convertToReasoningStepResults(),
		Sources:        []string{}, // Dynamic mode doesn't use traditional sources
	}, nil
}

// generateDynamicFinalResponseStreaming generates and streams the final response
func (d *DynamicReasoningEngine) generateDynamicFinalResponseStreaming(ch chan<- string) {
	// Generate final response from dynamic reasoning results
	d.agent.verbosePrint("Generating final dynamic response\n")

	if len(d.context.IterationResults) == 0 {
		ch <- "No reasoning steps completed\n"
		return
	}

	// Check for specific stop reasons that need special handling
	if strings.Contains(d.context.StopReason, "Need different approach") ||
		strings.Contains(d.context.StopReason, "Need different tools") ||
		strings.Contains(d.context.StopReason, "Goal already achieved") {
		// Let the AI generate a proper response for these cases
		// Don't assume user input is needed
	}

	// Build comprehensive final response using AI
	// Format context once for efficiency
	contextHistory := d.formatContextHistory()

	prompt := fmt.Sprintf(`You are an AI generating a final comprehensive response from dynamic reasoning results.

Original Query: %s
Final Goal: %s
Total Iterations: %d
Final Quality: %.2f
Stop Reason: %s

Conversation Context:
%s

Document Context:
%s

Context History:
%s

Goal Evolution: %s

Generate a comprehensive final response that:
1. Directly answers the original query
2. Synthesizes insights from all reasoning steps
3. Highlights key findings and conclusions
4. Shows the reasoning process and adaptations made
5. Provides confidence in the final answer
6. Incorporates relevant information from conversation and document context

Make the response clear, well-structured, and comprehensive.`,
		d.context.Query,
		d.context.CurrentGoal,
		len(d.context.IterationResults),
		d.context.QualityMetrics.OverallQuality,
		d.context.StopReason,
		d.formatConversationContext(),
		d.formatDocumentContext(),
		contextHistory,
		strings.Join(d.context.GoalEvolutionHistory, " -> "))

	// Use streaming for the final response
	streamCh, err := d.agent.llm.GenerateStreaming(prompt)
	if err != nil {
		d.agent.verbosePrint("Final response generation failed: %v\n", err)
		// Fallback to simple concatenation
		var answerBuilder strings.Builder
		for _, result := range d.context.IterationResults {
			if result.Success {
				answerBuilder.WriteString(fmt.Sprintf("%s: %s\n", result.StepName, result.Output))
			}
		}
		ch <- answerBuilder.String()
		return
	}

	// Stream the response
	for chunk := range streamCh {
		ch <- chunk
	}

	// Add confidence and token usage info
	ch <- fmt.Sprintf("\n\nConfidence: %.2f\n", d.context.QualityMetrics.OverallQuality)
	ch <- fmt.Sprintf("Tokens used: %d\n", d.calculateTotalTokensUsed())
}

// formatContextHistory formats all context history for AI prompts
func (d *DynamicReasoningEngine) formatContextHistory() string {
	var parts []string

	// Step history
	if len(d.context.IterationResults) > 0 {
		parts = append(parts, "Recent Steps:")
		for _, result := range d.context.IterationResults {
			parts = append(parts, fmt.Sprintf("- %s (%s): %s",
				result.StepName, result.StepType, result.Output))
		}
	}

	// Self-reflections
	if len(d.context.SelfReflections) > 0 {
		parts = append(parts, "Self-Reflections:")
		for _, reflection := range d.context.SelfReflections {
			parts = append(parts, fmt.Sprintf("- Performance: %.2f, Goal Alignment: %.2f",
				reflection.PerformanceScore, reflection.GoalAlignment))
		}
	}

	// Meta-reasoning
	if len(d.context.MetaReasoning) > 0 {
		parts = append(parts, "Meta-Reasoning:")
		for _, meta := range d.context.MetaReasoning {
			parts = append(parts, fmt.Sprintf("- %s: %s",
				meta.ReasoningType, meta.Decision))
		}
	}

	// Adaptation history
	if len(d.context.AdaptationHistory) > 0 {
		parts = append(parts, "Adaptations:")
		for _, adaptation := range d.context.AdaptationHistory {
			parts = append(parts, fmt.Sprintf("- %s -> %s",
				adaptation.PreviousApproach, adaptation.NewApproach))
		}
	}

	if len(parts) == 0 {
		return "No previous context"
	}
	return strings.Join(parts, "\n")
}

// Helper methods for formatting context information

func (d *DynamicReasoningEngine) convertToReasoningStepResults() []ReasoningStepResult {
	var results []ReasoningStepResult
	for _, dynamicResult := range d.context.IterationResults {
		results = append(results, ReasoningStepResult{
			StepName:   dynamicResult.StepName,
			StepType:   dynamicResult.StepType,
			Output:     dynamicResult.Output,
			Success:    dynamicResult.Success,
			Error:      dynamicResult.Error,
			Duration:   dynamicResult.Duration,
			TokensUsed: dynamicResult.TokensUsed,
		})
	}
	return results
}

// cleanJSONResponse extracts JSON from responses that may contain backticks or other formatting
func (d *DynamicReasoningEngine) cleanJSONResponse(response string) string {
	// Remove backticks if present
	cleaned := strings.ReplaceAll(response, "`", "")

	// Find JSON object boundaries
	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")

	if start != -1 && end != -1 && end > start {
		jsonStr := cleaned[start : end+1]

		// Fix common JSON issues with LaTeX expressions
		// Replace problematic LaTeX expressions in string values only
		jsonStr = strings.ReplaceAll(jsonStr, `\( \frac{1}{0} \)`, "1/0")
		jsonStr = strings.ReplaceAll(jsonStr, `\(`, "")
		jsonStr = strings.ReplaceAll(jsonStr, `\)`, "")

		return jsonStr
	}

	// If no JSON object found, try to find JSON array
	start = strings.Index(cleaned, "[")
	end = strings.LastIndex(cleaned, "]")

	if start != -1 && end != -1 && end > start {
		return cleaned[start : end+1]
	}

	return ""
}

// Placeholder methods for quality calculations
func (d *DynamicReasoningEngine) calculateOverallQuality() float64 {
	if len(d.context.IterationResults) == 0 {
		return 0.0
	}

	var total float64
	for _, result := range d.context.IterationResults {
		total += result.QualityScore
	}
	return total / float64(len(d.context.IterationResults))
}

func (d *DynamicReasoningEngine) calculateGoalAlignment() float64 {
	// AI-driven goal alignment calculation
	if len(d.context.IterationResults) == 0 {
		return 0.0
	}

	prompt := fmt.Sprintf(`You are an AI evaluating how well your reasoning results align with the current goal.

Current Goal: %s
Original Goal: %s
Results to Evaluate: %d iterations

Recent Results:
%s

Rate the overall goal alignment on a scale of 0.0 to 1.0, where:
- 1.0 = Perfect alignment with goal
- 0.5 = Partial alignment
- 0.0 = No alignment

Consider:
1. How directly do the results address the goal?
2. Are the results relevant and useful?
3. Do they provide the information needed?

Respond with JSON:
{
  "alignment_score": 0.0-1.0,
  "reasoning": "Your analysis of the alignment",
  "confidence": 0.0-1.0
}`,
		d.context.CurrentGoal,
		d.context.OriginalGoal,
		len(d.context.IterationResults),
		d.formatContextHistory())

	response, _, err := d.agent.llm.Generate(prompt)
	if err != nil {
		d.agent.verbosePrint("Goal alignment calculation failed: %v\n", err)
		return 0.0 // Let AI handle alignment issues
	}

	var alignment struct {
		AlignmentScore float64 `json:"alignment_score"`
		Reasoning      string  `json:"reasoning"`
		Confidence     float64 `json:"confidence"`
	}

	cleanedResponse := d.cleanJSONResponse(response)
	if err := json.Unmarshal([]byte(cleanedResponse), &alignment); err != nil {
		d.agent.verbosePrint("Failed to parse goal alignment: %v\n", err)
		return 0.0 // Let AI handle alignment issues
	}

	return alignment.AlignmentScore
}

func (d *DynamicReasoningEngine) calculateCompleteness() float64 {
	// Implementation for completeness calculation
	if len(d.context.IterationResults) == 0 {
		return 0.0
	}

	// Calculate completeness based on coverage and depth
	var totalCompleteness float64
	for _, result := range d.context.IterationResults {
		if result.Success {
			// Completeness based on output length and quality
			outputLength := float64(len(result.Output))
			completeness := math.Min(outputLength/500.0, 1.0) * result.QualityScore
			totalCompleteness += completeness
		}
	}

	return totalCompleteness / float64(len(d.context.IterationResults))
}

func (d *DynamicReasoningEngine) calculateEfficiency() float64 {
	// Implementation for efficiency calculation
	if len(d.context.IterationResults) == 0 {
		return 0.0
	}

	// Calculate efficiency based on tokens used vs quality achieved
	var totalEfficiency float64
	for _, result := range d.context.IterationResults {
		if result.Success && result.TokensUsed > 0 {
			// Efficiency = quality per token
			efficiency := result.QualityScore / float64(result.TokensUsed) * 1000
			totalEfficiency += efficiency
		}
	}

	return totalEfficiency / float64(len(d.context.IterationResults))
}

// Helper methods for AI-driven decisions
func (d *DynamicReasoningEngine) getCurrentApproachDescription() string {
	if len(d.context.IterationResults) == 0 {
		return "Initial approach - no previous results"
	}

	// Describe the current approach based on recent steps
	var approach []string
	for i, result := range d.context.IterationResults {
		if i >= len(d.context.IterationResults)-3 { // Last 3 steps
			approach = append(approach, fmt.Sprintf("%s (%s, quality: %.2f)",
				result.StepName, result.StepType, result.QualityScore))
		}
	}
	return strings.Join(approach, " -> ")
}

// calculateTotalTokensUsed calculates the total tokens used across all iterations
func (d *DynamicReasoningEngine) calculateTotalTokensUsed() int {
	total := 0
	for _, result := range d.context.IterationResults {
		total += result.TokensUsed
	}
	return total
}
