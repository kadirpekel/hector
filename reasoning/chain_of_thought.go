package reasoning

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ============================================================================
// CHAIN OF THOUGHT - ITERATIVE REASONING ENGINE
// Multi-pass reasoning with behavioral signals for continuation
// Uses service-based helper functions for common operations
// ============================================================================

type ChainOfThoughtReasoningEngine struct {
	services AgentServices
}

// NewChainOfThoughtReasoningEngine creates the chain-of-thought reasoning engine
func NewChainOfThoughtReasoningEngine(services AgentServices) *ChainOfThoughtReasoningEngine {
	return &ChainOfThoughtReasoningEngine{
		services: services,
	}
}

// ============================================================================
// ENGINE INTERFACE IMPLEMENTATION
// ============================================================================

func (e *ChainOfThoughtReasoningEngine) GetName() string {
	return "Chain-of-Thought"
}

func (e *ChainOfThoughtReasoningEngine) GetDescription() string {
	return "Iterative reasoning with tool support, self-evaluation, and behavioral stopping signals"
}

// ============================================================================
// MAIN EXECUTION
// ============================================================================

func (e *ChainOfThoughtReasoningEngine) Execute(ctx context.Context, input string) (<-chan string, error) {
	outputCh := make(chan string, 100)

	go func() {
		defer close(outputCh)

		startTime := time.Now()
		config := e.services.GetConfig()
		maxIterations := e.getMaxIterations()
		iteration := 0

		// Track reasoning history for current session (not conversation history)
		var reasoningHistory strings.Builder

		// Collect assistant response for history (only actual content, not debug info)
		var assistantResponse strings.Builder

		// Use helper function for history
		recordUserQuery(e.services, input)

		// Show reasoning metadata only if debug info is enabled
		if config.ShowDebugInfo {
			outputCh <- fmt.Sprintf("\n🔍 **Chain-of-Thought Reasoning**\n")
			outputCh <- fmt.Sprintf("📊 Max iterations: %d\n\n", maxIterations)
		}

		// Main reasoning loop
		for iteration < maxIterations {
			iteration++

			// Show iteration counter only if debug info is enabled
			if config.ShowDebugInfo {
				outputCh <- fmt.Sprintf("🤔 **Iteration %d/%d**\n", iteration, maxIterations)
			}

			// Build prompt using PromptService
			prompt, err := e.buildPrompt(ctx, input, reasoningHistory.String(), iteration, maxIterations)
			if err != nil {
				outputCh <- fmt.Sprintf("Error building prompt: %v\n", err)
				return
			}

			// Use helper function for LLM generation
			response, err := generateResponse(ctx, e.services, prompt, outputCh)
			if err != nil {
				outputCh <- fmt.Sprintf("Error: %v\n", err)
				return
			}

			// Collect for conversation history (cross-query persistence)
			assistantResponse.WriteString(response.String())

			// Store in working memory (intra-query context for next iteration)
			if reasoningHistory.Len() > 0 {
				reasoningHistory.WriteString("\n")
			}
			reasoningHistory.WriteString("Assistant: ")
			reasoningHistory.WriteString(response.String())
			reasoningHistory.WriteString("\n")

			// Use helper function for extension execution
			extensionResults, err := executeDiscoveredExtensions(ctx, e.services, outputCh)
			if err != nil {
				return
			}

			// If extensions were called, store in working memory and continue
			if len(extensionResults) > 0 {
				for _, result := range extensionResults {
					if result.Success {
						// Store tool result in working memory for next iteration
						contentForHistory := result.Content
						if len(contentForHistory) > 2000 {
							contentForHistory = contentForHistory[:2000] + "\n...(truncated)"
						}

						// Show actual tool name (not just "tools") for pattern recognition
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

				outputCh <- "\n"
				continue // Process results in next iteration
			}

			// No extensions called - LLM is signaling it's done
			if config.ShowDebugInfo {
				outputCh <- "\n✅ **Reasoning Complete**\n"
			}
			break
		}

		// Summary - only if debug info is enabled
		if config.ShowDebugInfo {
			duration := time.Since(startTime).Seconds()
			outputCh <- "\n============================================================\n"
			outputCh <- "📝 **REASONING SUMMARY**\n"
			outputCh <- "============================================================\n"
			outputCh <- fmt.Sprintf("**Iterations**: %d/%d\n", iteration, maxIterations)
			outputCh <- fmt.Sprintf("**Duration**: %.1fs\n", duration)
			outputCh <- "**Approach**: Behavioral signals, LLM-driven decisions\n"
			outputCh <- "============================================================\n"
		}

		// Use helper function for history
		recordAssistantResponse(e.services, assistantResponse.String())
	}()

	return outputCh, nil
}

// ============================================================================
// PROMPT BUILDING
// ============================================================================

func (e *ChainOfThoughtReasoningEngine) buildPrompt(ctx context.Context, originalQuery string, reasoningHistory string, currentIter, maxIter int) (string, error) {
	// Get standard prompt data from PromptService
	data, err := e.services.Prompt().BuildDefaultPromptData(
		ctx,
		originalQuery,
		e.services.Context(),
		e.services.History(),
		e.services.Extensions(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to build prompt data: %w", err)
	}

	// Build system prompt - generic reasoning guidance
	// Tool mechanics are handled by tool extension
	systemPrompt := `You are a helpful AI assistant with reasoning capabilities.

Think step-by-step. Before taking any action, review what you've already tried.`

	// Build instructions
	var instructions strings.Builder

	// Include working memory from previous iterations (if any)
	if reasoningHistory != "" {
		// Keep only recent context to avoid prompt explosion
		if len(reasoningHistory) > 3000 {
			instructions.WriteString("[Earlier reasoning]\n")
			instructions.WriteString(reasoningHistory[len(reasoningHistory)-3000:])
		} else {
			instructions.WriteString(reasoningHistory)
		}
		instructions.WriteString("\n")
	}

	// Generic reasoning guidance (tool mechanics handled by tool extension)
	instructions.WriteString("Think carefully about the query. Use available tools if you need more information.\n")

	// Use PromptService to build final prompt
	templateParts := map[string]string{
		"system":       systemPrompt,
		"instructions": instructions.String(),
		"output":       "Response:",
	}

	return e.services.Prompt().BuildPromptFromParts(templateParts, data)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (e *ChainOfThoughtReasoningEngine) getMaxIterations() int {
	config := e.services.GetConfig()
	if config.MaxIterations > 0 {
		return config.MaxIterations
	}
	// Default: 5 iterations (generous but not excessive)
	return 5
}
