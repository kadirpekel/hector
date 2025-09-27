package reasonings

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// COMMON TYPES (to avoid import cycles)
// ============================================================================

// All common types are now imported from the interfaces package

// AgentResponse is imported from interfaces package

// All interfaces are now imported from the centralized interfaces package
// No duplicate definitions needed

// ============================================================================
// DYNAMIC REASONING TYPES
// ============================================================================

// DynamicReasoningContext holds the evolving context for dynamic reasoning
type DynamicReasoningContext struct {
	Query                string                           `json:"query"`
	OriginalGoal         string                           `json:"original_goal"`
	CurrentGoal          string                           `json:"current_goal"`
	GoalEvolutionHistory []string                         `json:"goal_evolution_history"`
	IterationResults     []DynamicIterationResult         `json:"iteration_results"`
	SelfReflections      []SelfReflection                 `json:"self_reflections"`
	MetaReasoning        []MetaReasoningStep              `json:"meta_reasoning"`
	AdaptationHistory    []AdaptationDecision             `json:"adaptation_history"`
	QualityMetrics       QualityMetrics                   `json:"quality_metrics"`
	CurrentIteration     int                              `json:"current_iteration"`
	AvailableTools       []interfaces.ToolInfo            `json:"available_tools"`
	DocumentContext      []interfaces.SearchResult        `json:"document_context"`
	ConversationContext  []interfaces.ConversationMessage `json:"conversation_context"`
	ShouldStop           bool                             `json:"should_stop"`
	StopReason           string                           `json:"stop_reason"`
}

// DynamicIterationResult represents the result of a single reasoning iteration
type DynamicIterationResult struct {
	IterationNumber    int                              `json:"iteration_number"`
	StepName           string                           `json:"step_name"`
	StepType           string                           `json:"step_type"`
	Input              string                           `json:"input"`
	Output             string                           `json:"output"`
	ToolsUsed          []string                         `json:"tools_used"`
	ToolResults        map[string]interfaces.ToolResult `json:"tool_results"`
	QualityScore       float64                          `json:"quality_score"`
	GoalProgress       float64                          `json:"goal_progress"`
	Confidence         float64                          `json:"confidence"`
	TokensUsed         int                              `json:"tokens_used"`
	Duration           time.Duration                    `json:"duration"`
	Success            bool                             `json:"success"`
	Error              string                           `json:"error,omitempty"`
	SelfReflection     *SelfReflection                  `json:"self_reflection,omitempty"`
	AdaptationNeeded   bool                             `json:"adaptation_needed"`
	NextStepSuggestion string                           `json:"next_step_suggestion"`
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

// ============================================================================
// DYNAMIC REASONING ENGINE
// ============================================================================

// DynamicReasoningEngine implements the ReasoningEngine interface for dynamic reasoning
type DynamicReasoningEngine struct {
	agent   interfaces.Agent
	config  config.ReasoningConfig
	context *DynamicReasoningContext
}

// NewDynamicReasoningEngine creates a new dynamic reasoning engine
func NewDynamicReasoningEngine(agent interfaces.Agent, config config.ReasoningConfig) *DynamicReasoningEngine {
	return &DynamicReasoningEngine{
		agent:  agent,
		config: config,
	}
}

// ExecuteReasoning executes the main dynamic reasoning loop
func (d *DynamicReasoningEngine) ExecuteReasoning(ctx context.Context, query string) (*interfaces.AgentResponse, error) {
	// 1. Initialize dynamic reasoning context
	if err := d.initializeDynamicContext(ctx, query); err != nil {
		return nil, fmt.Errorf("failed to initialize dynamic context: %w", err)
	}

	// 2. Main dynamic reasoning loop
	for d.context.CurrentIteration < d.config.MaxIterations && !d.context.ShouldStop {
		d.context.CurrentIteration++

		// Step 1: Meta-reasoning about current state
		if d.config.EnableMetaReasoning {
			if err := d.performMetaReasoning(ctx); err != nil {
				// Meta-reasoning failed, continue without it
			}
		}

		// Step 2: Self-reflection on previous iteration
		if d.config.EnableSelfReflection && d.context.CurrentIteration > 1 {
			if err := d.performSelfReflection(ctx); err != nil {
				// Self-reflection failed, continue without it
			}
		}

		// Step 3: Execute reasoning step
		result, err := d.executeDynamicStep(ctx, nil)
		if err != nil {
			result = DynamicIterationResult{
				IterationNumber: d.context.CurrentIteration,
				StepName:        "reasoning_step",
				StepType:        "dynamic",
				Success:         false,
				Error:           err.Error(),
				Duration:        0,
			}
		}
		d.context.IterationResults = append(d.context.IterationResults, result)

		// Step 4: Evaluate goal achievement
		goalAchieved, progress := d.evaluateGoalAchievement(result)
		result.GoalProgress = progress

		// Step 5: Update quality metrics
		d.updateQualityMetrics(result)

		// Step 6: Check stopping conditions
		if d.shouldStopReasoning(result, goalAchieved, progress) {
			d.context.ShouldStop = true
			d.context.StopReason = d.determineStopReason(result, goalAchieved, progress)
			break
		}

		// Step 7: Adapt approach if needed
		if d.shouldAdaptApproach(result) {
			d.adaptApproach(ctx, result)
		}

		// Step 8: Evolve goals if enabled
		if d.config.EnableGoalEvolution {
			d.evolveGoals(ctx, result)
		}
	}

	// 3. Generate final response
	return d.generateFinalResponse(), nil
}

// ExecuteReasoningStreaming executes dynamic reasoning with streaming output
func (d *DynamicReasoningEngine) ExecuteReasoningStreaming(ctx context.Context, query string) (<-chan string, error) {
	ch := make(chan string, 100)

	go func() {
		defer close(ch)

		// 1. Initialize dynamic reasoning context
		if err := d.initializeDynamicContext(ctx, query); err != nil {
			ch <- fmt.Sprintf("Error: Failed to initialize dynamic context: %v\n", err)
			return
		}

		ch <- fmt.Sprintf("🧠 Starting Dynamic Reasoning for: %s\n", query)
		ch <- fmt.Sprintf("📊 Configuration: max_iterations=%d, quality_threshold=%.2f\n",
			d.config.MaxIterations, d.config.QualityThreshold)

		// 2. Main dynamic reasoning loop with streaming
		for d.context.CurrentIteration < d.config.MaxIterations && !d.context.ShouldStop {
			d.context.CurrentIteration++

			ch <- fmt.Sprintf("\n=== 🔄 Dynamic Iteration %d/%d ===\n",
				d.context.CurrentIteration, d.config.MaxIterations)

			// Step 1: Meta-reasoning
			if d.config.EnableMetaReasoning {
				ch <- "🧠 Performing meta-reasoning...\n"
				if err := d.performMetaReasoning(ctx); err == nil {
					if len(d.context.MetaReasoning) > 0 {
						lastMeta := d.context.MetaReasoning[len(d.context.MetaReasoning)-1]
						ch <- fmt.Sprintf("📝 Meta-reasoning: %s - %s (confidence: %.2f)\n",
							lastMeta.ReasoningType, lastMeta.Decision, lastMeta.Confidence)
					}
				}
			}

			// Step 2: Self-reflection
			if d.config.EnableSelfReflection && d.context.CurrentIteration > 1 {
				ch <- "🪞 Performing self-reflection...\n"
				if err := d.performSelfReflection(ctx); err == nil {
					if len(d.context.SelfReflections) > 0 {
						lastReflection := d.context.SelfReflections[len(d.context.SelfReflections)-1]
						ch <- fmt.Sprintf("📊 Performance score: %.2f, Goal alignment: %.2f\n",
							lastReflection.PerformanceScore, lastReflection.GoalAlignment)
					}
				}
			}

			// Step 3: Execute reasoning step
			ch <- "⚡ Executing reasoning step...\n"
			result, err := d.executeDynamicStep(ctx, ch)
			if err != nil {
				result = DynamicIterationResult{
					IterationNumber: d.context.CurrentIteration,
					StepName:        "reasoning_step",
					StepType:        "dynamic",
					Success:         false,
					Error:           err.Error(),
					Duration:        0,
				}
			}
			d.context.IterationResults = append(d.context.IterationResults, result)

			ch <- fmt.Sprintf("✅ Step completed: %s (quality: %.2f, confidence: %.2f, duration: %v)\n",
				result.StepName, result.QualityScore, result.Confidence, result.Duration)

			// Step 4: Evaluate goal achievement
			goalAchieved, progress := d.evaluateGoalAchievement(result)
			result.GoalProgress = progress
			ch <- fmt.Sprintf("🎯 Goal evaluation: achieved=%t, progress=%.2f\n", goalAchieved, progress)

			// Step 5: Update quality metrics
			d.updateQualityMetrics(result)

			// Step 6: Check stopping conditions
			if d.shouldStopReasoning(result, goalAchieved, progress) {
				d.context.ShouldStop = true
				d.context.StopReason = d.determineStopReason(result, goalAchieved, progress)
				ch <- fmt.Sprintf("🛑 Stopping: %s\n", d.context.StopReason)
				break
			}

			// Step 7: Adapt approach if needed
			if d.shouldAdaptApproach(result) {
				ch <- "🔄 Adapting approach...\n"
				d.adaptApproach(ctx, result)
			}

			// Step 8: Evolve goals if enabled
			if d.config.EnableGoalEvolution {
				ch <- "🎯 Evaluating goal evolution...\n"
				d.evolveGoals(ctx, result)
			}
		}

		// 3. Generate final response
		ch <- "\n🎉 Generating final response...\n"
		finalResponse := d.generateFinalResponse()
		ch <- fmt.Sprintf("📝 Final Answer: %s\n", finalResponse.Answer)
		ch <- fmt.Sprintf("📊 Total iterations: %d, Quality: %.2f\n",
			d.context.CurrentIteration, d.context.QualityMetrics.OverallQuality)
	}()

	return ch, nil
}

// GetName returns the name of this reasoning engine
func (d *DynamicReasoningEngine) GetName() string {
	return "dynamic"
}

// GetDescription returns a description of this reasoning engine
func (d *DynamicReasoningEngine) GetDescription() string {
	return "Dynamic reasoning engine with meta-reasoning, self-reflection, and adaptive capabilities"
}

// ============================================================================
// DYNAMIC REASONING ENGINE FACTORY
// ============================================================================

// DynamicReasoningEngineFactory creates dynamic reasoning engines
type DynamicReasoningEngineFactory struct{}

// CreateEngine creates a dynamic reasoning engine
func (f *DynamicReasoningEngineFactory) CreateEngine(engineType string, agent interfaces.Agent, config config.ReasoningConfig) (interfaces.ReasoningEngine, error) {
	if engineType != "dynamic" {
		return nil, fmt.Errorf("unsupported engine type: %s", engineType)
	}

	return NewDynamicReasoningEngine(agent, config), nil
}

// ListAvailableEngines returns available engine types
func (f *DynamicReasoningEngineFactory) ListAvailableEngines() []string {
	return []string{"dynamic"}
}

// GetEngineInfo returns information about the dynamic reasoning engine
func (f *DynamicReasoningEngineFactory) GetEngineInfo(engineType string) (interfaces.ReasoningEngineInfo, error) {
	if engineType != "dynamic" {
		return interfaces.ReasoningEngineInfo{}, fmt.Errorf("unsupported engine type: %s", engineType)
	}

	return interfaces.ReasoningEngineInfo{
		Name:        "dynamic",
		Description: "Dynamic reasoning engine with meta-reasoning, self-reflection, and adaptive capabilities",
		Features:    []string{"meta-reasoning", "self-reflection", "goal-evolution", "adaptive-approach"},
		Parameters: []interfaces.ReasoningParameter{
			{
				Name:        "max_iterations",
				Type:        "int",
				Description: "Maximum number of reasoning iterations",
				Required:    false,
				Default:     5,
			},
			{
				Name:        "quality_threshold",
				Type:        "float64",
				Description: "Quality threshold for stopping reasoning",
				Required:    false,
				Default:     0.8,
			},
			{
				Name:        "enable_meta_reasoning",
				Type:        "bool",
				Description: "Enable meta-reasoning capabilities",
				Required:    false,
				Default:     true,
			},
			{
				Name:        "enable_self_reflection",
				Type:        "bool",
				Description: "Enable self-reflection capabilities",
				Required:    false,
				Default:     true,
			},
		},
		Examples: []interfaces.ReasoningExample{
			{
				Name:        "Basic Dynamic Reasoning",
				Description: "Simple dynamic reasoning with default settings",
				Config: config.ReasoningConfig{
					MaxIterations:        5,
					QualityThreshold:     0.8,
					EnableMetaReasoning:  true,
					EnableSelfReflection: true,
					EnableGoalEvolution:  false,
					EnableDynamicTools:   true,
				},
				Query: "Analyze the current market trends and provide insights",
			},
		},
	}, nil
}

// ============================================================================
// INTERNAL IMPLEMENTATION METHODS
// ============================================================================

// initializeDynamicContext initializes the dynamic reasoning context
func (d *DynamicReasoningEngine) initializeDynamicContext(ctx context.Context, query string) error {
	// Gather document context
	documentContext, err := d.agent.GatherContext(ctx, query)
	if err != nil {
		documentContext = []interfaces.SearchResult{} // Continue with empty context
	}

	// Get conversation context
	conversationContext := []interfaces.ConversationMessage{}
	if d.agent.GetHistory() != nil {
		conversationContext = d.agent.GetHistory().GetRecentConversationMessages(6)
	}

	// Get available tools
	availableTools := []interfaces.ToolInfo{}
	if d.agent.GetToolRegistry() != nil {
		availableTools = d.agent.GetToolRegistry().ListTools()
	}

	// Initialize context
	d.context = &DynamicReasoningContext{
		Query:                query,
		OriginalGoal:         query,
		CurrentGoal:          query,
		GoalEvolutionHistory: []string{query},
		IterationResults:     []DynamicIterationResult{},
		SelfReflections:      []SelfReflection{},
		MetaReasoning:        []MetaReasoningStep{},
		AdaptationHistory:    []AdaptationDecision{},
		QualityMetrics: QualityMetrics{
			OverallQuality: 0.0,
			Trend:          "stable",
			LastUpdated:    time.Now(),
		},
		CurrentIteration:    0,
		AvailableTools:      availableTools,
		DocumentContext:     documentContext,
		ConversationContext: conversationContext,
		ShouldStop:          false,
	}

	return nil
}

// performMetaReasoning performs AI reasoning about the reasoning process
func (d *DynamicReasoningEngine) performMetaReasoning(ctx context.Context) error {
	prompt := d.buildMetaReasoningPrompt()

	response, _, err := d.agent.GetLLM().Generate(prompt)
	if err != nil {
		return fmt.Errorf("meta-reasoning LLM call failed: %w", err)
	}

	// Parse meta-reasoning response
	var metaStep MetaReasoningStep
	if err := json.Unmarshal([]byte(response), &metaStep); err != nil {
		// If JSON parsing fails, create a basic meta-reasoning step
		metaStep = MetaReasoningStep{
			StepNumber:    len(d.context.MetaReasoning) + 1,
			ReasoningType: "strategy_selection",
			Analysis:      response,
			Decision:      "Continue with current approach",
			Confidence:    0.7,
		}
	}

	d.context.MetaReasoning = append(d.context.MetaReasoning, metaStep)
	return nil
}

// performSelfReflection performs AI self-reflection on performance
func (d *DynamicReasoningEngine) performSelfReflection(ctx context.Context) error {
	if len(d.context.IterationResults) == 0 {
		return nil
	}

	prompt := d.buildSelfReflectionPrompt()

	response, _, err := d.agent.GetLLM().Generate(prompt)
	if err != nil {
		return fmt.Errorf("self-reflection LLM call failed: %w", err)
	}

	// Parse self-reflection response
	var reflection SelfReflection
	if err := json.Unmarshal([]byte(response), &reflection); err != nil {
		// If JSON parsing fails, create a basic reflection
		reflection = SelfReflection{
			IterationNumber:   d.context.CurrentIteration,
			PerformanceScore:  0.7,
			GoalAlignment:     0.7,
			QualityAssessment: response,
		}
	}

	d.context.SelfReflections = append(d.context.SelfReflections, reflection)
	return nil
}

// executeDynamicStep executes a single reasoning step
func (d *DynamicReasoningEngine) executeDynamicStep(ctx context.Context, streamCh chan<- string) (DynamicIterationResult, error) {
	startTime := time.Now()

	// Build reasoning prompt
	prompt := d.buildReasoningStepPrompt()

	var response string
	var tokensUsed int
	var err error

	// Use streaming if channel is provided and streaming is enabled
	if streamCh != nil && d.config.EnableStreaming {
		// Generate streaming response
		llmStreamCh, err := d.agent.GetLLM().GenerateStreaming(prompt)
		if err != nil {
			return DynamicIterationResult{}, fmt.Errorf("reasoning step LLM streaming call failed: %w", err)
		}

		// Collect streaming response and forward to output
		var responseBuilder strings.Builder
		for chunk := range llmStreamCh {
			responseBuilder.WriteString(chunk)
			streamCh <- chunk                        // Forward to output channel
			tokensUsed += len(strings.Fields(chunk)) // Rough token estimation
		}
		response = responseBuilder.String()
	} else {
		// Generate non-streaming response
		response, tokensUsed, err = d.agent.GetLLM().Generate(prompt)
		if err != nil {
			return DynamicIterationResult{}, fmt.Errorf("reasoning step LLM call failed: %w", err)
		}
	}

	// Execute tools if dynamic tools are enabled
	var toolResults map[string]interfaces.ToolResult
	if d.config.EnableDynamicTools {
		toolResults, _ = d.agent.ExecuteTools(ctx, d.context.CurrentGoal)
	}

	// Calculate quality metrics
	qualityScore := d.calculateStepQuality(response, toolResults)
	confidence := d.calculateConfidence(response, toolResults)

	result := DynamicIterationResult{
		IterationNumber: d.context.CurrentIteration,
		StepName:        fmt.Sprintf("reasoning_step_%d", d.context.CurrentIteration),
		StepType:        "dynamic_reasoning",
		Input:           d.context.CurrentGoal,
		Output:          response,
		ToolsUsed:       d.extractToolsUsed(toolResults),
		ToolResults:     toolResults,
		QualityScore:    qualityScore,
		Confidence:      confidence,
		TokensUsed:      tokensUsed,
		Duration:        time.Since(startTime),
		Success:         true,
	}

	return result, nil
}

// Rest of the implementation methods...
func (d *DynamicReasoningEngine) evaluateGoalAchievement(result DynamicIterationResult) (bool, float64) {
	progress := (result.QualityScore + result.Confidence) / 2.0
	achieved := progress >= d.config.QualityThreshold
	return achieved, progress
}

func (d *DynamicReasoningEngine) updateQualityMetrics(result DynamicIterationResult) {
	d.context.QualityMetrics.OverallQuality = result.QualityScore
	d.context.QualityMetrics.GoalAlignment = result.GoalProgress
	d.context.QualityMetrics.Completeness = result.GoalProgress
	d.context.QualityMetrics.Accuracy = result.Confidence
	d.context.QualityMetrics.LastUpdated = time.Now()

	if len(d.context.IterationResults) > 1 {
		prevResult := d.context.IterationResults[len(d.context.IterationResults)-2]
		if result.QualityScore > prevResult.QualityScore {
			d.context.QualityMetrics.Trend = "improving"
		} else if result.QualityScore < prevResult.QualityScore {
			d.context.QualityMetrics.Trend = "declining"
		} else {
			d.context.QualityMetrics.Trend = "stable"
		}
	}
}

func (d *DynamicReasoningEngine) shouldStopReasoning(result DynamicIterationResult, goalAchieved bool, progress float64) bool {
	if goalAchieved || result.QualityScore >= d.config.QualityThreshold || d.context.CurrentIteration >= d.config.MaxIterations {
		return true
	}

	if len(d.context.IterationResults) >= 3 {
		recentResults := d.context.IterationResults[len(d.context.IterationResults)-3:]
		declining := true
		for i := 1; i < len(recentResults); i++ {
			if recentResults[i].QualityScore >= recentResults[i-1].QualityScore {
				declining = false
				break
			}
		}
		if declining {
			return true
		}
	}

	return false
}

func (d *DynamicReasoningEngine) determineStopReason(result DynamicIterationResult, goalAchieved bool, progress float64) string {
	if goalAchieved {
		return "Goal achieved"
	}
	if result.QualityScore >= d.config.QualityThreshold {
		return "Quality threshold reached"
	}
	if d.context.CurrentIteration >= d.config.MaxIterations {
		return "Maximum iterations reached"
	}
	if d.context.QualityMetrics.Trend == "declining" {
		return "Quality declining"
	}
	return "Unknown"
}

func (d *DynamicReasoningEngine) shouldAdaptApproach(result DynamicIterationResult) bool {
	if result.QualityScore < 0.5 {
		return true
	}
	if len(d.context.IterationResults) >= 2 {
		prevResult := d.context.IterationResults[len(d.context.IterationResults)-2]
		if result.GoalProgress <= prevResult.GoalProgress {
			return true
		}
	}
	return false
}

func (d *DynamicReasoningEngine) adaptApproach(ctx context.Context, result DynamicIterationResult) {
	adaptation := AdaptationDecision{
		IterationNumber:  d.context.CurrentIteration,
		Trigger:          "Low quality or stagnant progress",
		PreviousApproach: "Current reasoning strategy",
		NewApproach:      "Adjusted reasoning strategy",
		Reasoning:        "Adapting to improve performance",
		Confidence:       0.7,
	}
	d.context.AdaptationHistory = append(d.context.AdaptationHistory, adaptation)
}

func (d *DynamicReasoningEngine) evolveGoals(ctx context.Context, result DynamicIterationResult) {
	if result.QualityScore > 0.8 && len(result.Output) > 100 {
		d.context.GoalEvolutionHistory = append(d.context.GoalEvolutionHistory,
			fmt.Sprintf("Iteration %d: Refined based on insights", d.context.CurrentIteration))
	}
}

func (d *DynamicReasoningEngine) generateFinalResponse() *interfaces.AgentResponse {
	var allToolResults = make(map[string]interfaces.ToolResult)
	var totalTokens int

	// Use only the last successful iteration result as the final answer
	var finalAnswer string
	if len(d.context.IterationResults) > 0 {
		// Get the last iteration result
		lastResult := d.context.IterationResults[len(d.context.IterationResults)-1]
		if lastResult.Success {
			finalAnswer = lastResult.Output
			totalTokens = lastResult.TokensUsed
		}
	}

	// Collect all tool results from all iterations
	for _, result := range d.context.IterationResults {
		if result.Success {
			for name, toolResult := range result.ToolResults {
				allToolResults[name] = toolResult
			}
		}
	}

	return &interfaces.AgentResponse{
		Answer:      strings.TrimSpace(finalAnswer),
		Context:     d.context.DocumentContext,
		Sources:     d.agent.ExtractSources(d.context.DocumentContext),
		ToolResults: allToolResults,
		TokensUsed:  totalTokens,
		Confidence:  d.context.QualityMetrics.OverallQuality,
	}
}

// Helper methods for prompt building
func (d *DynamicReasoningEngine) buildMetaReasoningPrompt() string {
	conversationContextStr := ""
	if len(d.context.ConversationContext) > 0 {
		contextData, _ := json.MarshalIndent(d.context.ConversationContext, "", "  ")
		conversationContextStr = string(contextData)
		if len(conversationContextStr) > 200 {
			conversationContextStr = conversationContextStr[:200]
		}
	}

	return fmt.Sprintf(`You are an AI reasoning about your own reasoning process. Analyze the current state and decide what to do next.

Current Context:
- Query: %s
- Current Goal: %s
- Iteration: %d/%d
- Previous Results: %d iterations completed
- Quality Trend: %s

Available Tools: %d tools
Document Context: %d documents
Conversation Context: %s

Respond with a JSON object:
{
  "reasoning_type": "strategy_selection|step_planning|quality_evaluation",
  "analysis": "Your analysis of the current state",
  "decision": "What should be done next",
  "rationale": "Why this decision makes sense",
  "confidence": 0.8,
  "alternative_options": ["option1", "option2"]
}`,
		d.context.Query,
		d.context.CurrentGoal,
		d.context.CurrentIteration,
		d.config.MaxIterations,
		len(d.context.IterationResults),
		d.context.QualityMetrics.Trend,
		len(d.context.AvailableTools),
		len(d.context.DocumentContext),
		conversationContextStr)
}

func (d *DynamicReasoningEngine) buildSelfReflectionPrompt() string {
	if len(d.context.IterationResults) == 0 {
		return ""
	}

	lastResult := d.context.IterationResults[len(d.context.IterationResults)-1]
	outputPreview := lastResult.Output
	if len(outputPreview) > 200 {
		outputPreview = outputPreview[:200]
	}

	return fmt.Sprintf(`You are an AI reflecting on your own performance. Analyze your recent reasoning step.

Recent Step Analysis:
- Step: %s (%s)
- Output: %s
- Quality Score: %.2f
- Confidence: %.2f
- Duration: %v
- Success: %t

Overall Context:
- Query: %s
- Current Goal: %s
- Iteration: %d/%d

Respond with a JSON object:
{
  "performance_score": 0.8,
  "strengths": ["strength1", "strength2"],
  "weaknesses": ["weakness1"],
  "improvement_suggestions": ["suggestion1"],
  "goal_alignment": 0.9,
  "efficiency_score": 0.7,
  "quality_assessment": "Detailed assessment"
}`,
		lastResult.StepName,
		lastResult.StepType,
		outputPreview,
		lastResult.QualityScore,
		lastResult.Confidence,
		lastResult.Duration,
		lastResult.Success,
		d.context.Query,
		d.context.CurrentGoal,
		d.context.CurrentIteration,
		d.config.MaxIterations)
}

func (d *DynamicReasoningEngine) buildReasoningStepPrompt() string {
	prompt := d.agent.BuildPrompt(d.context.CurrentGoal, d.context.DocumentContext, make(map[string]interfaces.ToolResult))

	prompt += fmt.Sprintf(`

Dynamic Reasoning Context:
- Iteration: %d/%d
- Original Goal: %s
- Current Goal: %s
- Previous Iterations: %d completed

Please provide a thoughtful response that addresses the current goal while considering the context above.`,
		d.context.CurrentIteration,
		d.config.MaxIterations,
		d.context.OriginalGoal,
		d.context.CurrentGoal,
		len(d.context.IterationResults))

	return prompt
}

func (d *DynamicReasoningEngine) calculateStepQuality(response string, toolResults map[string]interfaces.ToolResult) float64 {
	quality := 0.5
	if len(response) > 100 {
		quality += 0.2
	}
	if len(response) > 500 {
		quality += 0.1
	}
	for _, result := range toolResults {
		if result.Success {
			quality += 0.1
		}
	}
	if quality > 1.0 {
		quality = 1.0
	}
	return quality
}

func (d *DynamicReasoningEngine) calculateConfidence(response string, toolResults map[string]interfaces.ToolResult) float64 {
	confidence := 0.7
	successfulTools := 0
	totalTools := len(toolResults)
	for _, result := range toolResults {
		if result.Success {
			successfulTools++
		}
	}
	if totalTools > 0 {
		toolSuccessRate := float64(successfulTools) / float64(totalTools)
		confidence = (confidence + toolSuccessRate) / 2.0
	}
	return confidence
}

func (d *DynamicReasoningEngine) extractToolsUsed(toolResults map[string]interfaces.ToolResult) []string {
	tools := make([]string, 0, len(toolResults))
	for name := range toolResults {
		tools = append(tools, name)
	}
	return tools
}
