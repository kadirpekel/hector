package reasoning

import (
	"context"
	"fmt"
	"strings"
)

// ============================================================================
// CHAIN OF THOUGHT REASONING ENGINE - RECURSIVE AI REASONING
// ============================================================================

// ChainOfThoughtReasoningEngine implements a recursive reasoning approach
// where the AI can call itself to continue reasoning, creating a chain of thoughts
type ChainOfThoughtReasoningEngine struct {
	services      AgentServices
	maxIterations int
	loopDetection bool
}

// NewChainOfThoughtReasoningEngine creates a new chain-of-thought reasoning engine
func NewChainOfThoughtReasoningEngine(services AgentServices, maxIterations int, loopDetection bool) *ChainOfThoughtReasoningEngine {
	if maxIterations <= 0 {
		maxIterations = 5 // Default max iterations
	}

	return &ChainOfThoughtReasoningEngine{
		services:      services,
		maxIterations: maxIterations,
		loopDetection: loopDetection,
	}
}

// ============================================================================
// REASONING ENGINE INTERFACE IMPLEMENTATION
// ============================================================================

func (e *ChainOfThoughtReasoningEngine) Execute(ctx context.Context, query string) (<-chan string, error) {
	outputCh := make(chan string, 100)

	go func() {
		defer close(outputCh)

		// Track reasoning iterations to prevent infinite loops
		iterationCount := 0
		reasoningHistory := make([]string, 0)
		queryHistory := make([]string, 0)

		// Add user query to history
		e.services.History().AddToHistory("user", query, nil)
		queryHistory = append(queryHistory, query)

		// Start the reasoning chain
		e.executeReasoningChain(ctx, query, outputCh, &iterationCount, reasoningHistory, queryHistory)
	}()

	return outputCh, nil
}

// executeReasoningChain performs the recursive reasoning process
func (e *ChainOfThoughtReasoningEngine) executeReasoningChain(
	ctx context.Context,
	query string,
	outputCh chan<- string,
	iterationCount *int,
	reasoningHistory []string,
	queryHistory []string,
) {
	*iterationCount++

	// Check iteration limit
	if *iterationCount > e.maxIterations {
		outputCh <- fmt.Sprintf("\n\n🔄 **Reasoning Complete** (reached max iterations: %d)\n", e.maxIterations)
		return
	}

	// Check for loops if enabled
	if e.loopDetection && e.detectLoop(query, queryHistory) {
		outputCh <- fmt.Sprintf("\n\n🔄 **Reasoning Complete** (loop detected, stopping to prevent infinite recursion)\n")
		return
	}

	// Build reasoning prompt with chain-of-thought instructions
	prompt, err := e.buildChainOfThoughtPrompt(ctx, query, reasoningHistory, *iterationCount)
	if err != nil {
		outputCh <- fmt.Sprintf("Error building reasoning prompt: %v", err)
		return
	}

	// Generate LLM response
	var responseBuilder strings.Builder
	config := e.services.GetConfig()

	if config.EnableStreaming {
		// Stream the response
		streamCh, err := e.services.LLM().GenerateLLMStreaming(prompt)
		if err != nil {
			outputCh <- fmt.Sprintf("Error generating streaming response: %v", err)
			return
		}

		for chunk := range streamCh {
			outputCh <- chunk
			responseBuilder.WriteString(chunk)
		}
	} else {
		// Non-streaming response
		response, _, err := e.services.LLM().GenerateLLM(prompt)
		if err != nil {
			outputCh <- fmt.Sprintf("Error generating response: %v", err)
			return
		}

		outputCh <- response
		responseBuilder.WriteString(response)
	}

	// Check for reasoning extension calls (self-recursive calls)
	llmService := e.services.LLM()
	if llmService != nil {
		extensionCalls := llmService.GetExtensionCalls()

		// Look for reasoning extension calls
		reasoningCalls := e.filterReasoningCalls(extensionCalls)

		if len(reasoningCalls) > 0 {
			// Execute reasoning extensions (self-calls)
			reasoningResults, err := e.services.Extensions().ExecuteExtensions(ctx, reasoningCalls)
			if err != nil {
				outputCh <- fmt.Sprintf("\nError executing reasoning extensions: %v", err)
				return
			}

			// Process reasoning results and continue the chain
			for _, result := range reasoningResults {
				if result.Success {
					// Add to reasoning history
					reasoningHistory = append(reasoningHistory, result.Content)

					// Continue the reasoning chain with the new query
					newQuery := e.extractQueryFromReasoning(result.Content)
					if newQuery != "" {
						queryHistory = append(queryHistory, newQuery)
						outputCh <- fmt.Sprintf("\n\n🔄 **Continuing Reasoning Chain** (iteration %d)\n", *iterationCount)
						e.executeReasoningChain(ctx, newQuery, outputCh, iterationCount, reasoningHistory, queryHistory)
						return
					}
				} else {
					outputCh <- fmt.Sprintf("\n❌ Reasoning extension failed: %s", result.Error)
				}
			}
		}

		// Handle other extensions (tools, etc.)
		otherCalls := e.filterNonReasoningCalls(extensionCalls)
		if len(otherCalls) > 0 {
			extensionResults, err := e.services.Extensions().ExecuteExtensions(ctx, otherCalls)
			if err != nil {
				outputCh <- fmt.Sprintf("\nError executing extensions: %v", err)
				return
			}

			// Process extension results
			e.processExtensionResults(ctx, extensionResults, outputCh, &responseBuilder)
		}
	}

	// Add complete response to history
	e.services.History().AddToHistory("assistant", responseBuilder.String(), nil)

	// If no reasoning extensions were called, the chain ends naturally
	if *iterationCount == 1 {
		outputCh <- fmt.Sprintf("\n\n✅ **Reasoning Complete** (natural conclusion)\n")
	}
}

// buildChainOfThoughtPrompt builds a prompt that encourages chain-of-thought reasoning
func (e *ChainOfThoughtReasoningEngine) buildChainOfThoughtPrompt(
	ctx context.Context,
	query string,
	reasoningHistory []string,
	iteration int,
) (string, error) {
	// Build base prompt data
	promptData, err := e.services.Prompt().BuildDefaultPromptData(
		ctx, query, e.services.Context(), e.services.History(), e.services.Extensions(),
	)
	if err != nil {
		return "", err
	}

	// Add chain-of-thought specific instructions
	chainInstructions := e.buildChainInstructions(iteration, reasoningHistory)

	// Create custom template parts for chain-of-thought
	templateParts := map[string]string{
		"system":     chainInstructions,
		"context":    "{{.Context}}",
		"history":    "{{.History}}",
		"extensions": "{{.Extensions}}",
		"query":      "{{.Query}}",
	}

	// Build the prompt
	return e.services.Prompt().BuildPromptFromParts(templateParts, promptData)
}

// buildChainInstructions creates the system instructions for chain-of-thought reasoning
func (e *ChainOfThoughtReasoningEngine) buildChainInstructions(iteration int, reasoningHistory []string) string {
	var instructions strings.Builder
	
	instructions.WriteString("You are an advanced reasoning AI that can think through complex problems step by step.\n\n")
	
	instructions.WriteString("## CRITICAL REQUIREMENT: Use Reasoning Calls\n")
	instructions.WriteString("You MUST use the reasoning extension for complex questions. This is mandatory.\n\n")
	
	instructions.WriteString("## Current Reasoning Context\n")
	instructions.WriteString(fmt.Sprintf("**Iteration:** %d of %d\n", iteration, e.maxIterations))
	
	if len(reasoningHistory) > 0 {
		instructions.WriteString("**Previous Reasoning Steps:**\n")
		for i, step := range reasoningHistory {
			instructions.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
		instructions.WriteString("\n")
	}
	
	instructions.WriteString("## MANDATORY: You MUST Use Reasoning Calls\n")
	instructions.WriteString("For complex questions, you MUST follow this exact pattern:\n\n")
	instructions.WriteString("1. Provide initial analysis\n")
	instructions.WriteString("2. Then IMMEDIATELY use a reasoning call in this EXACT format:\n\n")
	instructions.WriteString("REASONING_CALL:\n")
	instructions.WriteString("{\n")
	instructions.WriteString("  \"query\": \"What are the specific risks and benefits of this approach?\",\n")
	instructions.WriteString("  \"purpose\": \"Need to analyze trade-offs before making a recommendation\"\n")
	instructions.WriteString("}\n\n")
	
	instructions.WriteString("## EXACT EXAMPLES - Copy This Format:\n\n")
	
	instructions.WriteString("Example 1 - AI Implementation Question:\n")
	instructions.WriteString("After your initial response, you MUST add:\n")
	instructions.WriteString("REASONING_CALL:\n")
	instructions.WriteString("{\n")
	instructions.WriteString("  \"query\": \"What are the specific risks and benefits of AI implementation?\",\n")
	instructions.WriteString("  \"purpose\": \"Need to analyze trade-offs before making a recommendation\"\n")
	instructions.WriteString("}\n\n")
	
	instructions.WriteString("Example 2 - Business Strategy Question:\n")
	instructions.WriteString("After your initial response, you MUST add:\n")
	instructions.WriteString("REASONING_CALL:\n")
	instructions.WriteString("{\n")
	instructions.WriteString("  \"query\": \"What are the potential challenges and opportunities in this strategy?\",\n")
	instructions.WriteString("  \"purpose\": \"Need to explore both sides before finalizing the analysis\"\n")
	instructions.WriteString("}\n\n")
	
	instructions.WriteString("## CRITICAL INSTRUCTIONS:\n")
	instructions.WriteString("- You MUST use reasoning calls for complex questions\n")
	instructions.WriteString("- Copy the EXACT format shown above\n")
	instructions.WriteString("- Do NOT just say you will use reasoning calls - ACTUALLY use them\n")
	instructions.WriteString("- The reasoning call must come AFTER your initial analysis\n")
	instructions.WriteString("- This is MANDATORY - not optional\n\n")
	
	instructions.WriteString("## FORCE REASONING CALL - DO THIS NOW:\n")
	instructions.WriteString("After your response, you MUST include this EXACT text:\n\n")
	instructions.WriteString("REASONING_CALL:\n")
	instructions.WriteString("{\n")
	instructions.WriteString("  \"query\": \"What are the specific risks and benefits of AI implementation?\",\n")
	instructions.WriteString("  \"purpose\": \"Need to analyze trade-offs before making a recommendation\"\n")
	instructions.WriteString("}\n\n")
	
	instructions.WriteString("Remember: You MUST demonstrate the reasoning loop by actually using REASONING_CALL: format.\n")
	
	return instructions.String()
}

// filterReasoningCalls filters extension calls to find reasoning-related ones
func (e *ChainOfThoughtReasoningEngine) filterReasoningCalls(calls []ExtensionCall) []ExtensionCall {
	var reasoningCalls []ExtensionCall
	for _, call := range calls {
		if call.Name == "reasoning" {
			reasoningCalls = append(reasoningCalls, call)
		}
	}
	return reasoningCalls
}

// filterNonReasoningCalls filters extension calls to exclude reasoning ones
func (e *ChainOfThoughtReasoningEngine) filterNonReasoningCalls(calls []ExtensionCall) []ExtensionCall {
	var otherCalls []ExtensionCall
	for _, call := range calls {
		if call.Name != "reasoning" {
			otherCalls = append(otherCalls, call)
		}
	}
	return otherCalls
}

// processExtensionResults handles non-reasoning extension results
func (e *ChainOfThoughtReasoningEngine) processExtensionResults(
	ctx context.Context,
	results map[string]ExtensionResult,
	outputCh chan<- string,
	responseBuilder *strings.Builder,
) {
	if len(results) == 0 {
		return
	}

	// Separate direct and LLM results
	directResults := make(map[string]ExtensionResult)
	llmResults := make(map[string]ExtensionResult)

	for name, result := range results {
		if result.Metadata != nil {
			if displayDirect, ok := result.Metadata["display_direct"].(bool); ok && displayDirect {
				directResults[name] = result
			} else {
				llmResults[name] = result
			}
		} else {
			llmResults[name] = result
		}
	}

	// Display direct results
	if len(directResults) > 0 {
		outputCh <- "\n\n"
		for name, result := range directResults {
			if result.Success {
				outputCh <- fmt.Sprintf("📋 %s Results:\n%s\n\n", name, result.Content)
			} else {
				outputCh <- fmt.Sprintf("❌ %s failed: %s\n\n", name, result.Error)
			}
		}
	}

	// Process LLM results
	if len(llmResults) > 0 {
		followUpPrompt, err := e.services.Prompt().BuildDefaultPrompt(
			ctx, "Please analyze and summarize the following results:",
			e.services.Context(), e.services.History(), e.services.Extensions(), llmResults,
		)
		if err != nil {
			outputCh <- fmt.Sprintf("\nError building follow-up prompt: %v", err)
			return
		}

		config := e.services.GetConfig()
		if config.EnableStreaming {
			followUpStreamCh, err := e.services.LLM().GenerateLLMStreaming(followUpPrompt)
			if err != nil {
				outputCh <- fmt.Sprintf("\nError generating follow-up response: %v", err)
				return
			}

			if len(directResults) == 0 {
				outputCh <- "\n\n"
			}
			for chunk := range followUpStreamCh {
				outputCh <- chunk
				responseBuilder.WriteString(chunk)
			}
		} else {
			followUpResponse, _, err := e.services.LLM().GenerateLLM(followUpPrompt)
			if err != nil {
				outputCh <- fmt.Sprintf("\nError generating follow-up response: %v", err)
				return
			}

			if len(directResults) == 0 {
				outputCh <- "\n\n"
			}
			outputCh <- followUpResponse
			responseBuilder.WriteString(followUpResponse)
		}
	}
}

// detectLoop checks if the current query is similar to previous queries (simple loop detection)
func (e *ChainOfThoughtReasoningEngine) detectLoop(currentQuery string, queryHistory []string) bool {
	if len(queryHistory) < 2 {
		return false
	}

	// Simple similarity check - if the current query is very similar to recent queries
	currentLower := strings.ToLower(strings.TrimSpace(currentQuery))

	// Check against last few queries
	checkCount := 3
	if len(queryHistory) < checkCount {
		checkCount = len(queryHistory)
	}

	for i := len(queryHistory) - checkCount; i < len(queryHistory); i++ {
		historyLower := strings.ToLower(strings.TrimSpace(queryHistory[i]))

		// Simple similarity: if queries are very similar (80%+ overlap)
		if e.calculateSimilarity(currentLower, historyLower) > 0.8 {
			return true
		}
	}

	return false
}

// calculateSimilarity calculates a simple similarity score between two strings
func (e *ChainOfThoughtReasoningEngine) calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Simple word-based similarity
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Count common words
	common := 0
	for _, word1 := range words1 {
		for _, word2 := range words2 {
			if word1 == word2 {
				common++
				break
			}
		}
	}

	// Return similarity as ratio of common words to total unique words
	total := len(words1) + len(words2) - common
	if total == 0 {
		return 0.0
	}

	return float64(common) / float64(total)
}

// extractQueryFromReasoning extracts a new query from reasoning result
func (e *ChainOfThoughtReasoningEngine) extractQueryFromReasoning(reasoningResult string) string {
	// Look for reasoning call patterns in the result
	// This is a simple extraction - in practice, you might want more sophisticated parsing
	lines := strings.Split(reasoningResult, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "REASONING_CALL:") || strings.Contains(line, "reasoning") {
			// Extract the next meaningful line as the query
			continue
		}
		if len(line) > 10 && !strings.HasPrefix(line, "{") && !strings.HasPrefix(line, "}") {
			return line
		}
	}

	// If no specific query found, return empty string to end the chain
	return ""
}

// GetName returns the name of the reasoning engine
func (e *ChainOfThoughtReasoningEngine) GetName() string {
	return "chain-of-thought"
}

// GetDescription returns a description of the reasoning engine
func (e *ChainOfThoughtReasoningEngine) GetDescription() string {
	return "Advanced reasoning engine that can recursively call itself to create chains of thought, enabling deep analysis and meta-cognitive reasoning"
}
