package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kadirpekel/hector/component"
	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/llms"
	"github.com/kadirpekel/hector/reasoning"
)

// ============================================================================
// AGENT - REASONING ORCHESTRATION + METADATA
// Agent contains the reasoning loop (formerly in Orchestrator)
// ============================================================================

// Agent executes reasoning with strategies
type Agent struct {
	name        string
	description string
	config      *config.AgentConfig
	services    reasoning.AgentServices
}

// NewAgent creates a new agent with services
func NewAgent(agentConfig *config.AgentConfig, componentMgr interface{}) (*Agent, error) {
	// Type assert to get the component manager
	compMgr, ok := componentMgr.(*component.ComponentManager)
	if !ok {
		return nil, fmt.Errorf("invalid component manager type")
	}

	// Create services
	services, err := NewAgentServices(agentConfig, compMgr)
	if err != nil {
		return nil, err
	}

	return &Agent{
		name:        agentConfig.Name,
		description: agentConfig.Description,
		config:      agentConfig,
		services:    services,
	}, nil
}

// ============================================================================
// PUBLIC API - QUERY METHODS
// ============================================================================

// Query executes a query using reasoning strategy (non-streaming interface)
func (a *Agent) Query(ctx context.Context, query string) (*reasoning.ReasoningResponse, error) {
	start := time.Now()

	// Create strategy based on config
	strategy, err := reasoning.CreateStrategy(a.config.Reasoning.Engine, a.config.Reasoning)
	if err != nil {
		return nil, err
	}

	// Execute reasoning loop
	streamCh, err := a.execute(ctx, query, strategy)
	if err != nil {
		return nil, err
	}

	// Collect all streaming output
	var fullResponse strings.Builder
	var tokensUsed int

	for chunk := range streamCh {
		fullResponse.WriteString(chunk)
		// Rough token estimation
		tokensUsed += len(strings.Fields(chunk))
	}

	return &reasoning.ReasoningResponse{
		Answer:     fullResponse.String(),
		TokensUsed: tokensUsed,
		Duration:   time.Since(start),
		Confidence: 0.8, // Good confidence for default approach
	}, nil
}

// QueryStreaming executes a query with streaming output
func (a *Agent) QueryStreaming(ctx context.Context, query string) (<-chan string, error) {
	// Create strategy based on config
	strategy, err := reasoning.CreateStrategy(a.config.Reasoning.Engine, a.config.Reasoning)
	if err != nil {
		return nil, err
	}

	// Execute reasoning loop
	return a.execute(ctx, query, strategy)
}

// ============================================================================
// REASONING LOOP - THE CORE ORCHESTRATION LOGIC
// This is what was in Orchestrator.Execute()
// ============================================================================

// execute runs the reasoning loop with the given strategy
func (a *Agent) execute(
	ctx context.Context,
	input string,
	strategy reasoning.ReasoningStrategy,
) (<-chan string, error) {
	outputCh := make(chan string, 100)

	go func() {
		defer close(outputCh)

		startTime := time.Now()
		cfg := a.services.GetConfig()
		state := reasoning.NewReasoningState()
		state.Query = input // Original user query for strategies
		state.OutputChannel = outputCh
		state.ShowThinking = cfg.ShowThinking
		state.ShowDebugInfo = cfg.ShowDebugInfo
		state.Services = a.services // Give strategies access to services
		state.Context = ctx         // Pass context for LLM calls
		maxIterations := a.getMaxIterations(cfg)

		// Show reasoning metadata if debug enabled
		if cfg.ShowDebugInfo {
			outputCh <- fmt.Sprintf("\n🔍 **%s**\n", strategy.GetName())
			outputCh <- fmt.Sprintf("📊 Max iterations: %d\n\n", maxIterations)
		}

		// Get available tools
		tools := a.services.Tools()
		toolDefs := tools.GetAvailableTools()

		if cfg.ShowDebugInfo {
			outputCh <- fmt.Sprintf("🔧 Available tools: %d\n", len(toolDefs))
			for _, tool := range toolDefs {
				outputCh <- fmt.Sprintf("  - %s: %s\n", tool.Name, tool.Description)
			}
			outputCh <- "\n"
		}

		// Main reasoning loop
		for state.Iteration < maxIterations {
			state.Iteration++

			// Check context cancellation
			select {
			case <-ctx.Done():
				outputCh <- fmt.Sprintf("\n⚠️  Cancelled: %v\n", ctx.Err())
				return
			default:
			}

			if cfg.ShowDebugInfo {
				outputCh <- fmt.Sprintf("🤔 **Iteration %d/%d**\n", state.Iteration, maxIterations)
			}

			// Strategy hook: prepare iteration
			if err := strategy.PrepareIteration(state.Iteration, state); err != nil {
				outputCh <- fmt.Sprintf("Error preparing iteration: %v\n", err)
				return
			}

			// Get prompt slots from strategy and merge with config
			promptSlots := a.buildPromptSlots(strategy)

			// Get strategy-specific context injection (e.g., todos for ChainOfThought)
			additionalContext := strategy.GetContextInjection(state)

			// Build messages using PromptService (with slots and additional context)
			messages, err := a.services.Prompt().BuildMessages(ctx, input, promptSlots, state.Conversation, additionalContext)
			if err != nil {
				outputCh <- fmt.Sprintf("Error building messages: %v\n", err)
				return
			}

			// Call LLM
			text, toolCalls, tokens, err := a.callLLM(ctx, messages, toolDefs, outputCh, cfg)
			if err != nil {
				outputCh <- fmt.Sprintf("Error: %v\n", err)
				return
			}

			state.TotalTokens += tokens

			if cfg.ShowDebugInfo {
				outputCh <- fmt.Sprintf("\033[90m📝 Tokens used: %d (total: %d)\033[0m\n", tokens, state.TotalTokens)
			}

			// Track text for history
			if text != "" && text != "[response was streamed]" {
				state.AssistantResponse.WriteString(text)
			}

			// Store first iteration's tool calls for metadata
			if state.Iteration == 1 && len(toolCalls) > 0 {
				state.FirstIterationToolCalls = toolCalls
			}

			// Execute tools if any
			var results []reasoning.ToolResult
			if len(toolCalls) > 0 {
				// Pass the LLM's text as context for tool labels (it often explains what it's doing)
				results = a.executeTools(ctx, toolCalls, text, outputCh, cfg)

				// Core protocol: Add assistant message + tool results to conversation
				// This is the OpenAI/Anthropic function calling protocol (not strategy-specific)
				assistantMsg := llms.Message{
					Role:      "assistant",
					Content:   text,
					ToolCalls: toolCalls,
				}
				state.Conversation = append(state.Conversation, assistantMsg)

				// Add tool results to conversation
				for _, result := range results {
					toolResultMsg := llms.Message{
						Role:       "tool",
						Content:    result.Content,
						ToolCallID: result.ToolCallID,
						Name:       result.ToolName,
					}
					state.Conversation = append(state.Conversation, toolResultMsg)
				}
			}

			// Strategy hook: Additional processing (reflection, meta-cognition, etc.)
			// Call BEFORE checking ShouldStop so reflection happens even for final iteration
			if err := strategy.AfterIteration(state.Iteration, text, toolCalls, results, state); err != nil {
				outputCh <- fmt.Sprintf("Error in strategy processing: %v\n", err)
				return
			}

			// Strategy hook: should stop?
			if strategy.ShouldStop(text, toolCalls, state) {
				if cfg.ShowDebugInfo {
					outputCh <- "\033[90m\n\n✅ **Reasoning complete**\033[0m\n"
				}
				break
			}
		}

		// Save to history
		a.saveToHistory(input, state, strategy, startTime)

		if cfg.ShowDebugInfo {
			outputCh <- fmt.Sprintf("\033[90m\n⏱️  Total time: %v | Tokens: %d | Iterations: %d\033[0m\n",
				time.Since(startTime), state.TotalTokens, state.Iteration)
		}
	}()

	return outputCh, nil
}

// ============================================================================
// HELPER METHODS - ORCHESTRATION UTILITIES
// ============================================================================

// callLLM calls the LLM service (streaming or non-streaming)
func (a *Agent) callLLM(
	ctx context.Context,
	messages []llms.Message,
	toolDefs []llms.ToolDefinition,
	outputCh chan<- string,
	cfg config.ReasoningConfig,
) (string, []llms.ToolCall, int, error) {
	llm := a.services.LLM()

	if cfg.EnableStreaming {
		// Streaming mode
		toolCalls, tokens, err := llm.GenerateStreaming(messages, toolDefs, outputCh)
		if err != nil {
			return "", nil, 0, err
		}
		return "[response was streamed]", toolCalls, tokens, nil
	}

	// Non-streaming mode
	text, toolCalls, tokens, err := llm.Generate(messages, toolDefs)
	if err != nil {
		return "", nil, 0, err
	}

	// Send text to output
	if text != "" {
		outputCh <- text
	}

	return text, toolCalls, tokens, nil
}

// executeTools executes all tool calls
// executeTools executes all tool calls sequentially
func (a *Agent) executeTools(
	ctx context.Context,
	toolCalls []llms.ToolCall,
	llmText string,
	outputCh chan<- string,
	cfg config.ReasoningConfig,
) []reasoning.ToolResult {
	tools := a.services.Tools()

	results := make([]reasoning.ToolResult, 0, len(toolCalls))

	for i, toolCall := range toolCalls {
		// Check cancellation before each tool
		select {
		case <-ctx.Done():
			return results
		default:
		}

		// Show tool execution label (conversational, enabled by default)
		if cfg.ShowToolExecution {
			// Use the LLM's natural text as the label (it often explains what it's doing)
			// If multiple tool calls, or no text, use a simple fallback
			var label string
			// Skip if text is streaming placeholder or empty
			if llmText != "" && llmText != "[response was streamed]" && len(toolCalls) == 1 {
				// Single tool call with accompanying text - use it as the label
				label = a.extractToolLabelFromText(llmText, toolCall)
			} else {
				// Multiple tools, streaming, or no text - generate a simple descriptive label
				label = a.generateSimpleToolLabel(toolCall)
			}

			if i == 0 {
				outputCh <- fmt.Sprintf("\n%s", label) // First tool, add newline before
			} else {
				outputCh <- label // Subsequent tools
			}
		}

		// Execute tool
		result, err := tools.ExecuteToolCall(ctx, toolCall)
		resultContent := result
		if err != nil {
			resultContent = fmt.Sprintf("Error: %v", err)
			if cfg.ShowToolExecution {
				outputCh <- fmt.Sprintf(" ❌\n")
			}
		} else {
			if cfg.ShowToolExecution {
				outputCh <- fmt.Sprintf(" ✅\n")
			}
		}

		results = append(results, reasoning.ToolResult{
			ToolCall:   toolCall,
			Content:    resultContent,
			Error:      err,
			ToolCallID: toolCall.ID,
			ToolName:   toolCall.Name,
		})
	}

	return results
}

// extractToolLabelFromText extracts a conversational label from the LLM's text
// The LLM often says something like "Let me check the weather..." before calling tools
func (a *Agent) extractToolLabelFromText(text string, toolCall llms.ToolCall) string {
	// Clean up the text
	text = strings.TrimSpace(text)

	// If text is too long, truncate it (keep it conversational and concise)
	if len(text) > 100 {
		// Find a good breaking point (sentence end)
		if idx := strings.Index(text[80:], "."); idx != -1 {
			text = text[:80+idx+1]
		} else {
			text = text[:100] + "..."
		}
	}

	// Use generic tool emoji - let the LLM provide context/personality
	// No keyword matching, no deterministic logic
	emoji := "🔧"

	// Format the label
	return fmt.Sprintf("%s %s", emoji, text)
}

// generateSimpleToolLabel generates a simple descriptive label when no LLM text is available
// Uses generic emoji and tool name - no keyword matching or deterministic logic
func (a *Agent) generateSimpleToolLabel(toolCall llms.ToolCall) string {
	// Generic tool emoji - no keyword matching
	emoji := "🔧"

	// Simple label: just the tool name
	// The LLM is responsible for providing context and personality via its text response
	return fmt.Sprintf("%s Calling %s...", emoji, toolCall.Name)
}

// Helper functions for string manipulation
func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	current := ""
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
			i += len(sep) - 1
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// saveToHistory saves the conversation to history service
func (a *Agent) saveToHistory(
	input string,
	state *reasoning.ReasoningState,
	strategy reasoning.ReasoningStrategy,
	startTime time.Time,
) {
	history := a.services.History()

	// Add user query
	history.AddToHistory("user", input, nil)

	// Add assistant response with metadata
	if state.AssistantResponse.Len() > 0 {
		metadata := map[string]interface{}{
			"reasoning_strategy": strategy.GetName(),
			"iterations":         state.Iteration,
			"total_tokens":       state.TotalTokens,
			"duration":           time.Since(startTime).String(),
		}

		// Include tool calls if any were made
		if len(state.FirstIterationToolCalls) > 0 {
			toolNames := make([]string, len(state.FirstIterationToolCalls))
			for i, tc := range state.FirstIterationToolCalls {
				toolNames[i] = tc.Name
			}
			metadata["tools_used"] = toolNames
		}

		history.AddToHistory("assistant", state.AssistantResponse.String(), metadata)
	}
}

// buildPromptSlots builds prompt slots by merging strategy defaults with user config
// This is the glue layer that prevents PromptService from knowing about Strategy
func (a *Agent) buildPromptSlots(strategy reasoning.ReasoningStrategy) reasoning.PromptSlots {
	// FULL OVERRIDE: If user provides system_prompt, return empty slots
	// This signals PromptService to use system_prompt instead of slots
	if a.config.Prompt.SystemPrompt != "" {
		return reasoning.PromptSlots{} // Empty slots = use system_prompt
	}

	// Get strategy's default slots
	strategySlots := strategy.GetPromptSlots()

	// SLOT-BASED CUSTOMIZATION: Merge with user's slot overrides (if any)
	// Users can override ANY slot individually - complete flexibility
	if len(a.config.Prompt.PromptSlots) > 0 {
		userSlots := reasoning.PromptSlots{
			SystemRole:            a.config.Prompt.PromptSlots["system_role"],
			ReasoningInstructions: a.config.Prompt.PromptSlots["reasoning_instructions"],
			ToolUsage:             a.config.Prompt.PromptSlots["tool_usage"],
			OutputFormat:          a.config.Prompt.PromptSlots["output_format"],
			CommunicationStyle:    a.config.Prompt.PromptSlots["communication_style"],
			Additional:            a.config.Prompt.PromptSlots["additional"],
		}
		strategySlots = strategySlots.Merge(userSlots)
	}

	return strategySlots
}

// getMaxIterations returns the maximum number of iterations
func (a *Agent) getMaxIterations(cfg config.ReasoningConfig) int {
	if cfg.MaxIterations > 0 {
		return cfg.MaxIterations
	}
	return 5 // Default
}

// ============================================================================
// METADATA ACCESSORS
// ============================================================================

// GetName returns the agent's name
func (a *Agent) GetName() string {
	return a.name
}

// GetDescription returns the agent's description
func (a *Agent) GetDescription() string {
	return a.description
}

// GetConfig returns the agent's configuration
func (a *Agent) GetConfig() *config.AgentConfig {
	return a.config
}

// GetServices returns the agent's services (for testing/advanced use)
func (a *Agent) GetServices() reasoning.AgentServices {
	return a.services
}
