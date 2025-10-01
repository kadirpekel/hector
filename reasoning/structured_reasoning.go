package reasoning

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ============================================================================
// STRUCTURED REASONING ENGINE - MATCHES CLAUDE'S ACTUAL REASONING PROCESS
// ============================================================================
//
// This engine explicitly models how Claude actually reasons:
// 1. Goal Extraction: "What am I trying to accomplish?"
// 2. Plan Formation: "What steps do I need to take?"
// 3. Iterative Execution: Take action → Evaluate → Decide next step
// 4. Meta-Cognition: "Am I making progress? Do I have enough information?"
// 5. Quality Check: "What's my confidence? Should I continue or synthesize?"
// 6. Synthesis: Provide complete answer
//
// This is NOT just a multi-pass loop - it's structured goal-oriented reasoning
// with explicit self-reflection and quality evaluation.
// ============================================================================

type StructuredReasoningEngine struct {
	services AgentServices
}

// ReasoningGoal represents what needs to be accomplished
type ReasoningGoal struct {
	MainGoal     string   // Primary objective
	SubGoals     []string // Specific sub-tasks
	Accomplished []string // What's been achieved
	Pending      []string // What's still needed
}

// ReflectionResult represents meta-cognitive evaluation
// This is what Claude does internally after each action
type ReflectionResult struct {
	WhatAccomplished  string  // What did I just learn/accomplish?
	WhatsStillMissing string  // What information is still needed?
	Confidence        float64 // 0.0-1.0: How confident in answering?
	ShouldContinue    bool    // Should I keep going or synthesize?
	Reason            string  // Why continue or stop?
}

// NewStructuredReasoningEngine creates the structured reasoning engine
func NewStructuredReasoningEngine(services AgentServices) *StructuredReasoningEngine {
	return &StructuredReasoningEngine{
		services: services,
	}
}

// ============================================================================
// ENGINE INTERFACE IMPLEMENTATION
// ============================================================================

func (e *StructuredReasoningEngine) GetName() string {
	return "Structured-Reasoning"
}

func (e *StructuredReasoningEngine) GetDescription() string {
	return "Goal-oriented reasoning with explicit planning, meta-cognition, and quality evaluation (matches Claude's actual reasoning process)"
}

// ============================================================================
// MAIN EXECUTION - MIRRORS CLAUDE'S REASONING PROCESS
// ============================================================================

func (e *StructuredReasoningEngine) Execute(ctx context.Context, input string) (<-chan string, error) {
	outputCh := make(chan string, 100)

	go func() {
		defer close(outputCh)

		startTime := time.Now()
		config := e.services.GetConfig()
		maxIterations := e.getMaxIterations()

		// Record user query in history
		recordUserQuery(e.services, input)

		// ========================================================================
		// PHASE 1: GOAL EXTRACTION (What am I trying to accomplish?)
		// This is the FIRST thing Claude does when receiving a query
		// ========================================================================

		goals, err := e.extractGoals(ctx, input)
		if err != nil {
			outputCh <- fmt.Sprintf("Error extracting goals: %v\n", err)
			// Fallback: create simple goal
			goals = &ReasoningGoal{
				MainGoal: input,
				SubGoals: []string{"Answer the query"},
				Pending:  []string{"Answer the query"},
			}
		}

		// Show goal extraction based on mode
		if config.ShowDebugInfo {
			// Full debug mode
			outputCh <- "\n🎯 **Structured Reasoning Engine**\n"
			outputCh <- "📋 Phase 1: Understanding Goals\n\n"
			outputCh <- fmt.Sprintf("**Main Goal:** %s\n", goals.MainGoal)
			if len(goals.SubGoals) > 0 {
				outputCh <- "**Sub-Goals:**\n"
				for _, sg := range goals.SubGoals {
					outputCh <- fmt.Sprintf("  ☐ %s\n", sg)
				}
			}
			outputCh <- "\n📋 Phase 2: Iterative Reasoning\n\n"
		} else if config.ShowThinking {
			// Thinking mode: grayed-out internal reasoning
			outputCh <- "\n"
			outputCh <- ThinkingGoalExtraction(goals.MainGoal, goals.SubGoals)
			outputCh <- "\n"
		} else {
			// Minimal visibility: just show we're thinking structured
			outputCh <- fmt.Sprintf("\n🎯 Goal: %s\n\n", goals.MainGoal)
		}

		// ========================================================================
		// PHASE 2: ITERATIVE REASONING WITH META-COGNITION
		// Execute → Evaluate → Reflect → Decide next action
		// ========================================================================

		var reasoningHistory strings.Builder
		var assistantResponse strings.Builder
		iteration := 0

		for iteration < maxIterations {
			iteration++

			if config.ShowDebugInfo {
				outputCh <- fmt.Sprintf("🤔 **Iteration %d/%d**\n", iteration, maxIterations)
			} else if config.ShowThinking {
				outputCh <- ThinkingProgress(iteration, maxIterations, "reasoning")
			}

			// Build prompt with goal context
			prompt, err := e.buildGoalOrientedPrompt(ctx, input, goals, reasoningHistory.String(), iteration, maxIterations)
			if err != nil {
				outputCh <- fmt.Sprintf("Error building prompt: %v\n", err)
				return
			}

			// Generate response
			response, err := generateResponse(ctx, e.services, prompt, outputCh)
			if err != nil {
				outputCh <- fmt.Sprintf("Error: %v\n", err)
				return
			}

			// Collect for conversation history
			assistantResponse.WriteString(response.String())

			// Store in working memory
			if reasoningHistory.Len() > 0 {
				reasoningHistory.WriteString("\n")
			}
			reasoningHistory.WriteString(fmt.Sprintf("Iteration %d: %s\n", iteration, response.String()))

			// Execute tools if any were called
			extensionResults, err := executeDiscoveredExtensions(ctx, e.services, outputCh)
			if err != nil {
				return
			}

			// ====================================================================
			// If tools were called: Add results + REFLECT (meta-cognition)
			// ====================================================================
			if len(extensionResults) > 0 {
				// Add tool results to working memory
				for _, result := range extensionResults {
					if result.Success {
						contentForHistory := result.Content
						if len(contentForHistory) > 1500 {
							contentForHistory = contentForHistory[:1500] + "\n...(truncated for brevity)"
						}

						toolName := "unknown"
						if metadata, ok := result.Metadata["tool_name"].(string); ok {
							toolName = metadata
						}

						reasoningHistory.WriteString(fmt.Sprintf("[Tool: %s]\n%s\n\n", toolName, contentForHistory))
					} else {
						outputCh <- fmt.Sprintf("\n❌ Tool failed: %s\n", result.Error)
						reasoningHistory.WriteString(fmt.Sprintf("[Tool: %s - Failed: %s]\n", result.Name, result.Error))
					}
				}

				// ================================================================
				// META-COGNITIVE REFLECTION (This is key!)
				// After getting tool results, Claude asks itself:
				// - What did I just learn?
				// - What's still missing?
				// - Am I making progress toward the goal?
				// - Do I have enough to answer confidently?
				// ================================================================

				reflection, err := e.reflect(ctx, input, goals, reasoningHistory.String())
				if err == nil {
					if config.ShowDebugInfo {
						// Full debug mode
						outputCh <- "\n💭 **Meta-Cognitive Reflection:**\n"
						outputCh <- fmt.Sprintf("Accomplished: %s\n", reflection.WhatAccomplished)
						if reflection.WhatsStillMissing != "" && reflection.WhatsStillMissing != "nothing" && !strings.Contains(strings.ToLower(reflection.WhatsStillMissing), "none") {
							outputCh <- fmt.Sprintf("Still Missing: %s\n", reflection.WhatsStillMissing)
						}
						outputCh <- fmt.Sprintf("Confidence: %.0f%%\n", reflection.Confidence*100)
						outputCh <- fmt.Sprintf("Decision: %s\n", reflection.Reason)
						outputCh <- "\n"
					} else if config.ShowThinking {
						// Thinking mode: grayed-out reflection
						outputCh <- "\n"
						outputCh <- ThinkingReflection(reflection.WhatAccomplished, reflection.WhatsStillMissing, reflection.Confidence)
						outputCh <- "\n"
					} else {
						// Minimal visibility: show key progress
						outputCh <- fmt.Sprintf("📍 Progress: %.0f%% confident", reflection.Confidence*100)
						if reflection.WhatsStillMissing != "" && reflection.WhatsStillMissing != "nothing" && !strings.Contains(strings.ToLower(reflection.WhatsStillMissing), "none") {
							outputCh <- fmt.Sprintf(" (still need: %s)", reflection.WhatsStillMissing)
						}
						outputCh <- "\n\n"
					}

					// Update goals based on reflection
					e.updateGoals(goals, reflection)

					// Quality-based stopping: If confident enough and nothing missing
					if !reflection.ShouldContinue && reflection.Confidence >= 0.75 {
						if config.ShowDebugInfo {
							outputCh <- "✅ **Quality threshold met - synthesizing answer**\n\n"
						} else {
							outputCh <- "✅ Analysis complete\n\n"
						}
						break
					}
				}

				outputCh <- "\n"
				continue
			}

			// ====================================================================
			// No tools called: Check if we should continue or stop
			// ====================================================================

			reflection, err := e.reflect(ctx, input, goals, reasoningHistory.String())
			if err == nil {
				if config.ShowDebugInfo {
					outputCh <- "\n🔍 **Quality Check:**\n"
					outputCh <- fmt.Sprintf("Confidence: %.0f%%\n", reflection.Confidence*100)
					outputCh <- fmt.Sprintf("Assessment: %s\n", reflection.Reason)
				} else if config.ShowThinking {
					outputCh <- "\n"
					outputCh <- ThinkingQualityCheck(reflection.Confidence, reflection.ShouldContinue, reflection.Reason)
					outputCh <- "\n"
				}

				// CRITICAL: Check if we should actually continue
				// Don't stop just because no tools were called!
				if reflection.ShouldContinue || reflection.Confidence < 0.75 {
					if config.ShowDebugInfo {
						outputCh <- "Decision: Continue reasoning to reach higher confidence\n\n"
					} else if !config.ShowThinking {
						outputCh <- fmt.Sprintf("📍 Continuing (%.0f%% confident)...\n\n", reflection.Confidence*100)
					}
					// Update goals and continue
					e.updateGoals(goals, reflection)
					continue
				}

				// Only stop if high confidence AND should not continue
				if !config.ShowDebugInfo && !config.ShowThinking {
					outputCh <- fmt.Sprintf("✅ Analysis complete (%.0f%% confident)\n\n", reflection.Confidence*100)
				} else if config.ShowDebugInfo {
					outputCh <- "Decision: Ready to provide final answer\n\n"
				}
			}

			break
		}

		// ========================================================================
		// PHASE 3: COMPLETION
		// ========================================================================

		// Minimal summary (not intrusive)
		if config.ShowDebugInfo {
			duration := time.Since(startTime).Seconds()
			outputCh <- fmt.Sprintf("\n💡 *Reasoning completed: %d iterations, %.1fs*\n", iteration, duration)
		}

		// Record to conversation history
		recordAssistantResponse(e.services, assistantResponse.String())
	}()

	return outputCh, nil
}

// ============================================================================
// GOAL EXTRACTION - PHASE 1
// ============================================================================

func (e *StructuredReasoningEngine) extractGoals(ctx context.Context, input string) (*ReasoningGoal, error) {
	// Use LLM to parse query and identify goals
	// This mirrors how Claude internally breaks down a query
	prompt := fmt.Sprintf(`You are analyzing a user query to extract goals.

USER QUERY: "%s"

Parse this query and identify:
1. The MAIN GOAL (one sentence - what's the primary objective?)
2. SUB-GOALS (specific tasks needed to accomplish the main goal)

Respond EXACTLY in this format:
MAIN_GOAL: [single sentence describing main objective]
SUB_GOALS:
- [first specific sub-task]
- [second specific sub-task]
- [additional sub-tasks as needed]

Be specific and actionable. Each sub-goal should be measurable.`, input)

	response, _, err := e.services.LLM().GenerateLLM(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to extract goals: %w", err)
	}

	// Parse response
	goals := &ReasoningGoal{
		SubGoals:     []string{},
		Accomplished: []string{},
		Pending:      []string{},
	}

	lines := strings.Split(response, "\n")
	inSubGoals := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "MAIN_GOAL:") {
			goals.MainGoal = strings.TrimSpace(strings.TrimPrefix(line, "MAIN_GOAL:"))
		} else if strings.HasPrefix(line, "SUB_GOALS:") || strings.HasPrefix(line, "SUB-GOALS:") {
			inSubGoals = true
		} else if inSubGoals && strings.HasPrefix(line, "- ") {
			subGoal := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if subGoal != "" {
				goals.SubGoals = append(goals.SubGoals, subGoal)
				goals.Pending = append(goals.Pending, subGoal)
			}
		}
	}

	// Fallback if parsing failed
	if goals.MainGoal == "" {
		goals.MainGoal = input
	}
	if len(goals.SubGoals) == 0 {
		goals.SubGoals = []string{"Analyze and answer the query"}
		goals.Pending = []string{"Analyze and answer the query"}
	}

	return goals, nil
}

// ============================================================================
// META-COGNITIVE REFLECTION - THE KEY DIFFERENTIATOR
// ============================================================================

func (e *StructuredReasoningEngine) reflect(ctx context.Context, originalQuery string, goals *ReasoningGoal, history string) (*ReflectionResult, error) {
	// This mirrors Claude's internal self-evaluation process
	// After each tool use, I ask myself these questions:

	// Truncate history if too long
	truncatedHistory := history
	if len(history) > 4000 {
		truncatedHistory = "...(earlier steps)\n" + history[len(history)-4000:]
	}

	prompt := fmt.Sprintf(`You are evaluating your reasoning progress. Be honest and self-aware.

ORIGINAL QUERY: "%s"

MAIN GOAL: %s

SUB-GOALS:
%s

WHAT YOU'VE DONE SO FAR:
%s

META-COGNITIVE SELF-EVALUATION:
Reflect on your progress and answer these questions:

1. What have you accomplished? (Be specific about what you now know)
2. What information is still missing or unclear?
3. How confident are you in providing a complete answer? (0-100%%)
4. Should you continue gathering information or are you ready to synthesize?

Respond EXACTLY in this format:
ACCOMPLISHED: [what you've learned/achieved]
STILL_MISSING: [what's still needed, or "nothing" if ready]
CONFIDENCE: [number 0-100 only]
SHOULD_CONTINUE: [yes or no]
REASON: [brief explanation of your decision]`,
		originalQuery,
		goals.MainGoal,
		e.formatGoalsStatus(goals),
		truncatedHistory)

	response, _, err := e.services.LLM().GenerateLLM(prompt)
	if err != nil {
		return nil, fmt.Errorf("reflection failed: %w", err)
	}

	// Parse reflection
	result := &ReflectionResult{}
	lines := strings.Split(response, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "ACCOMPLISHED:") {
			result.WhatAccomplished = strings.TrimSpace(strings.TrimPrefix(line, "ACCOMPLISHED:"))
		} else if strings.HasPrefix(line, "STILL_MISSING:") {
			result.WhatsStillMissing = strings.TrimSpace(strings.TrimPrefix(line, "STILL_MISSING:"))
		} else if strings.HasPrefix(line, "CONFIDENCE:") {
			confStr := strings.TrimSpace(strings.TrimPrefix(line, "CONFIDENCE:"))
			// Extract just the number
			confStr = strings.TrimSuffix(confStr, "%")
			confStr = strings.TrimSpace(confStr)
			if conf, err := strconv.ParseFloat(confStr, 64); err == nil {
				result.Confidence = conf / 100.0
			}
		} else if strings.HasPrefix(line, "SHOULD_CONTINUE:") {
			continueStr := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "SHOULD_CONTINUE:")))
			result.ShouldContinue = strings.Contains(continueStr, "yes")
		} else if strings.HasPrefix(line, "REASON:") {
			result.Reason = strings.TrimSpace(strings.TrimPrefix(line, "REASON:"))
		}
	}

	// Defaults if parsing failed
	if result.Confidence == 0 {
		result.Confidence = 0.5 // Assume medium confidence
	}

	return result, nil
}

// ============================================================================
// GOAL MANAGEMENT
// ============================================================================

func (e *StructuredReasoningEngine) updateGoals(goals *ReasoningGoal, reflection *ReflectionResult) {
	// Update accomplished/pending based on reflection
	// This is a simple heuristic - could be made more sophisticated

	if reflection.WhatAccomplished != "" && !strings.Contains(strings.ToLower(reflection.WhatAccomplished), "nothing") {
		// Try to match accomplished items to pending goals
		accomplishedLower := strings.ToLower(reflection.WhatAccomplished)

		newPending := []string{}
		for _, pending := range goals.Pending {
			// If the pending goal is mentioned in accomplished, move it
			if !strings.Contains(accomplishedLower, strings.ToLower(pending)) {
				newPending = append(newPending, pending)
			} else {
				if !contains(goals.Accomplished, pending) {
					goals.Accomplished = append(goals.Accomplished, pending)
				}
			}
		}
		goals.Pending = newPending
	}
}

func (e *StructuredReasoningEngine) formatGoalsStatus(goals *ReasoningGoal) string {
	var sb strings.Builder

	if len(goals.Accomplished) > 0 {
		sb.WriteString("ACCOMPLISHED:\n")
		for _, item := range goals.Accomplished {
			sb.WriteString(fmt.Sprintf("  ✅ %s\n", item))
		}
	}

	if len(goals.Pending) > 0 {
		sb.WriteString("PENDING:\n")
		for _, item := range goals.Pending {
			sb.WriteString(fmt.Sprintf("  ☐ %s\n", item))
		}
	}

	return sb.String()
}

// ============================================================================
// PROMPT BUILDING WITH GOAL CONTEXT
// ============================================================================

func (e *StructuredReasoningEngine) buildGoalOrientedPrompt(ctx context.Context, query string, goals *ReasoningGoal, history string, iteration, maxIter int) (string, error) {
	// Get standard prompt data from PromptService
	data, err := e.services.Prompt().BuildDefaultPromptData(
		ctx,
		query,
		e.services.Context(),
		e.services.History(),
		e.services.Extensions(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to build prompt data: %w", err)
	}

	// Build system prompt with goal awareness
	systemPrompt := fmt.Sprintf(`You are an AI assistant with structured reasoning capabilities.

MAIN GOAL: %s

You are working toward this goal step-by-step.

IMPORTANT: Before using any tools, briefly explain your reasoning and approach. Then use tools when you need information.`, goals.MainGoal)

	// Build instructions
	var instructions strings.Builder

	// Show goal status
	if len(goals.Accomplished) > 0 || len(goals.Pending) > 0 {
		instructions.WriteString("PROGRESS:\n")
		instructions.WriteString(e.formatGoalsStatus(goals))
		instructions.WriteString("\n")
	}

	// Include working memory
	if history != "" {
		// Truncate if too long
		if len(history) > 3500 {
			instructions.WriteString("[Earlier steps]\n")
			instructions.WriteString(history[len(history)-3500:])
		} else {
			instructions.WriteString("WHAT YOU'VE DONE:\n")
			instructions.WriteString(history)
		}
		instructions.WriteString("\n")
	}

	// Guidance for next action
	instructions.WriteString(fmt.Sprintf("This is iteration %d/%d. ", iteration, maxIter))
	if len(goals.Pending) > 0 {
		instructions.WriteString(fmt.Sprintf("Focus on: %s\n", goals.Pending[0]))
	} else {
		instructions.WriteString("Continue working toward the goal or provide your answer if ready.\n")
	}

	// Use PromptService to build final prompt
	templateParts := map[string]string{
		"system":       systemPrompt,
		"instructions": instructions.String(),
		"output":       "Next action:",
	}

	return e.services.Prompt().BuildPromptFromParts(templateParts, data)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (e *StructuredReasoningEngine) getMaxIterations() int {
	config := e.services.GetConfig()
	if config.MaxIterations > 0 {
		return config.MaxIterations
	}
	return 10 // Default for structured reasoning (may need more iterations)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
