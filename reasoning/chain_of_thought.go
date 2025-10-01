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
	services       AgentServices
	recursionDepth int
	maxDepth       int
}

// NewChainOfThoughtReasoningEngine creates a new chain-of-thought reasoning engine
func NewChainOfThoughtReasoningEngine(services AgentServices) *ChainOfThoughtReasoningEngine {
	return &ChainOfThoughtReasoningEngine{
		services:       services,
		recursionDepth: 0,
		maxDepth:       3, // Prevent infinite recursion
	}
}

// ============================================================================
// REASONING ENGINE INTERFACE IMPLEMENTATION
// ============================================================================

func (e *ChainOfThoughtReasoningEngine) Execute(ctx context.Context, query string) (<-chan string, error) {
	outputCh := make(chan string, 100)

	go func() {
		defer close(outputCh)

		// Add user query to history
		e.services.History().AddToHistory("user", query, nil)

		// Start the reasoning chain - let LLM decide when to stop
		e.executeReasoningChain(ctx, query, outputCh)
	}()

	return outputCh, nil
}

// executeReasoningChain performs the recursive reasoning process
func (e *ChainOfThoughtReasoningEngine) executeReasoningChain(
	ctx context.Context,
	query string,
	outputCh chan<- string,
) {
	// Check recursion depth to prevent infinite loops
	if e.recursionDepth >= e.maxDepth {
		outputCh <- fmt.Sprintf("\n\n🔄 **Reasoning Complete** (reached max depth: %d)\n", e.maxDepth)
		return
	}

	// Increment recursion depth
	e.recursionDepth++
	defer func() {
		e.recursionDepth-- // Decrement when function returns
	}()
	// Build reasoning prompt with chain-of-thought instructions
	prompt, err := e.buildChainOfThoughtPrompt(ctx, query)
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

	// Check for reasoning calls using the extension system
	llmService := e.services.LLM()
	extensionCalls := llmService.GetExtensionCalls()
	reasoningCalls := e.filterReasoningCalls(extensionCalls)

	if len(reasoningCalls) > 0 {
		// Execute reasoning extensions to get the results
		extensionResults, err := e.services.Extensions().ExecuteExtensions(ctx, reasoningCalls)
		if err != nil {
			outputCh <- fmt.Sprintf("\nError executing reasoning extensions: %v", err)
			return
		}

		// Process reasoning results and continue the chain
		for _, result := range extensionResults {
			if result.Name == "chain-of-thought" && result.Success {
				// Extract the query from the reasoning result metadata
				if query, ok := result.Metadata["original_query"].(string); ok && query != "" {
					// Continue the reasoning chain - let LLM decide when to stop
					outputCh <- fmt.Sprintf("\n\n🤔 **Continuing Reasoning Chain...** (depth: %d)\n", e.recursionDepth)

					// Continue with the next reasoning step
					e.executeReasoningChain(ctx, query, outputCh)
					return
				}
			}
		}
	}

	// Handle other extensions (tools, etc.) if any
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

	// Add complete response to history
	e.services.History().AddToHistory("assistant", responseBuilder.String(), nil)

	// If no reasoning extensions were called, the chain ends naturally
	// The LLM has decided it's done with reasoning
}

// buildChainOfThoughtPrompt builds a prompt that encourages chain-of-thought reasoning
func (e *ChainOfThoughtReasoningEngine) buildChainOfThoughtPrompt(
	ctx context.Context,
	query string,
) (string, error) {
	// Build base prompt data
	promptData, err := e.services.Prompt().BuildDefaultPromptData(
		ctx, query, e.services.Context(), e.services.History(), e.services.Extensions(),
	)
	if err != nil {
		return "", err
	}

	// Add chain-of-thought specific instructions
	chainInstructions := e.buildChainInstructions()

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
func (e *ChainOfThoughtReasoningEngine) buildChainInstructions() string {
	var instructions strings.Builder

	instructions.WriteString("You are an advanced reasoning AI that can think through complex problems step by step.\n\n")

	instructions.WriteString("## CRITICAL: You MUST Use Reasoning Calls\n")
	instructions.WriteString("For complex questions, you MUST use reasoning calls to demonstrate your thinking process.\n")
	instructions.WriteString("This is MANDATORY - not optional.\n\n")

	instructions.WriteString("## When to Use Reasoning Calls\n")
	instructions.WriteString("You MUST use reasoning calls when:\n")
	instructions.WriteString("- The question is complex or multi-faceted\n")
	instructions.WriteString("- You need to explore different aspects or perspectives\n")
	instructions.WriteString("- You want to break down the problem into smaller parts\n")
	instructions.WriteString("- You need to verify your conclusions\n")
	instructions.WriteString("- You want to think about your own thinking process\n\n")

	instructions.WriteString("## Reasoning Call Format\n")
	instructions.WriteString("After your initial response, you MUST include:\n\n")
	instructions.WriteString("REASONING_CALL:\n")
	instructions.WriteString("{\n")
	instructions.WriteString("  \"query\": \"A specific aspect you want to explore further\",\n")
	instructions.WriteString("  \"purpose\": \"Why you need to reason about this\"\n")
	instructions.WriteString("}\n\n")

	instructions.WriteString("## Examples of Good Reasoning Calls\n")
	instructions.WriteString("For a question about AI implications:\n")
	instructions.WriteString("REASONING_CALL:\n")
	instructions.WriteString("{\n")
	instructions.WriteString("  \"query\": \"What are the specific risks and benefits of AI implementation?\",\n")
	instructions.WriteString("  \"purpose\": \"Need to analyze trade-offs before making a recommendation\"\n")
	instructions.WriteString("}\n\n")

	instructions.WriteString("For a mathematical question:\n")
	instructions.WriteString("REASONING_CALL:\n")
	instructions.WriteString("{\n")
	instructions.WriteString("  \"query\": \"What are the real-world applications where this concept is important?\",\n")
	instructions.WriteString("  \"purpose\": \"Want to understand practical implications beyond theory\"\n")
	instructions.WriteString("}\n\n")

	instructions.WriteString("## MANDATORY INSTRUCTIONS\n")
	instructions.WriteString("- You MUST use reasoning calls for complex questions\n")
	instructions.WriteString("- Do NOT just say you will use reasoning calls - ACTUALLY use them\n")
	instructions.WriteString("- The reasoning call must come AFTER your initial analysis\n")
	instructions.WriteString("- This demonstrates your ability to think recursively\n")
	instructions.WriteString("- Use reasoning calls to show your thinking process\n\n")

	instructions.WriteString("## When to Stop\n")
	instructions.WriteString("You can stop reasoning when:\n")
	instructions.WriteString("- You have thoroughly explored the topic\n")
	instructions.WriteString("- You feel confident in your analysis\n")
	instructions.WriteString("- Further reasoning would not add value\n")
	instructions.WriteString("- You have provided a comprehensive answer\n\n")

	return instructions.String()
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

// ReasoningCall represents a reasoning call extracted from LLM response
type ReasoningCall struct {
	Query   string `json:"query"`
	Purpose string `json:"purpose,omitempty"`
	Context string `json:"context,omitempty"`
}

// filterReasoningCalls filters extension calls to only include chain-of-thought calls
func (e *ChainOfThoughtReasoningEngine) filterReasoningCalls(extensionCalls []ExtensionCall) []ExtensionCall {
	var reasoningCalls []ExtensionCall
	for _, call := range extensionCalls {
		if call.Name == "chain-of-thought" {
			reasoningCalls = append(reasoningCalls, call)
		}
	}
	return reasoningCalls
}

// GetName returns the name of the reasoning engine
func (e *ChainOfThoughtReasoningEngine) GetName() string {
	return "chain-of-thought"
}

// GetDescription returns a description of the reasoning engine
func (e *ChainOfThoughtReasoningEngine) GetDescription() string {
	return "Advanced reasoning engine that can recursively call itself to create chains of thought, enabling deep analysis and meta-cognitive reasoning"
}
