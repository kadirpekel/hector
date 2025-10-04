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

// ClearHistory clears the conversation history
func (a *Agent) ClearHistory() {
	history := a.services.History()
	if history != nil {
		history.ClearHistory()
	}
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

		// Restore conversation history from HistoryService (interactive mode)
		historyService := a.services.History()
		if historyService != nil {
			// Get recent history and restore to conversation
			recentHistory := historyService.GetRecentHistory(cfg.MaxIterations * 10) // Get plenty of history
			if len(recentHistory) > 0 {
				state.Conversation = append(state.Conversation, recentHistory...)
			}
		}

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
		// Philosophy: Trust the LLM to naturally terminate (like Cursor)
		// Loop continues while there are tool calls to execute
		// maxIterations is a safety valve only, rarely hit
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
			if text != "" {
				state.AssistantResponse.WriteString(text)
			}

			// Store first iteration's tool calls for metadata
			if state.Iteration == 1 && len(toolCalls) > 0 {
				state.FirstIterationToolCalls = toolCalls
			}

			// Execute tools if any
			var results []reasoning.ToolResult
			if len(toolCalls) > 0 {
				results = a.executeTools(ctx, toolCalls, outputCh, cfg)

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

		// Save to history (this now saves full llms.Message objects)
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
		// Streaming mode - tool names shown in real-time during streaming
		toolCalls, tokens, err := llm.GenerateStreaming(messages, toolDefs, outputCh)
		if err != nil {
			return "", nil, 0, err
		}
		return "", toolCalls, tokens, nil
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

	// Show tool names immediately (simulate streaming behavior for consistency)
	if len(toolCalls) > 0 && cfg.ShowToolExecution {
		outputCh <- "\n"
		for _, tc := range toolCalls {
			// Import formatToolLabel from services (or create inline)
			label := formatToolLabelForAgent(tc.Name, tc.Arguments)
			outputCh <- fmt.Sprintf("🔧 %s", label)
		}
	}

	return text, toolCalls, tokens, nil
}

// formatToolLabelForAgent creates a concise label for non-streaming mode
func formatToolLabelForAgent(toolName string, args map[string]interface{}) string {
	switch toolName {
	case "execute_command":
		if cmd, ok := args["command"].(string); ok {
			if len(cmd) > 60 {
				return fmt.Sprintf("%s: %s...", toolName, cmd[:57])
			}
			return fmt.Sprintf("%s: %s", toolName, cmd)
		}
	case "write_file", "search_replace":
		if path, ok := args["path"].(string); ok {
			return fmt.Sprintf("%s: %s", toolName, path)
		}
	case "search":
		if query, ok := args["query"].(string); ok {
			if len(query) > 40 {
				return fmt.Sprintf("%s: %s...", toolName, query[:37])
			}
			return fmt.Sprintf("%s: %s", toolName, query)
		}
	case "todo_write":
		if todos, ok := args["todos"].([]interface{}); ok {
			return fmt.Sprintf("%s: %d tasks", toolName, len(todos))
		}
	}
	return toolName
}

// executeTools executes all tool calls sequentially
func (a *Agent) executeTools(
	ctx context.Context,
	toolCalls []llms.ToolCall,
	outputCh chan<- string,
	cfg config.ReasoningConfig,
) []reasoning.ToolResult {
	tools := a.services.Tools()

	results := make([]reasoning.ToolResult, 0, len(toolCalls))

	for _, toolCall := range toolCalls {
		// Check cancellation before each tool
		select {
		case <-ctx.Done():
			return results
		default:
		}

		// Tool name already shown during streaming (for both OpenAI and Anthropic)
		// We just show the execution status here

		// Execute tool
		result, err := tools.ExecuteToolCall(ctx, toolCall)
		resultContent := result
		if err != nil {
			resultContent = fmt.Sprintf("Error: %v", err)
			if cfg.ShowToolExecution {
				outputCh <- fmt.Sprintf(" ❌\n") // Append status to previously shown tool name
			}
		} else {
			if cfg.ShowToolExecution {
				outputCh <- fmt.Sprintf(" ✅\n") // Append status to previously shown tool name
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

// saveToHistory saves the conversation to history service
func (a *Agent) saveToHistory(
	input string,
	state *reasoning.ReasoningState,
	strategy reasoning.ReasoningStrategy,
	startTime time.Time,
) {
	history := a.services.History()
	if history == nil {
		return
	}

	// Clear previous history and save the entire conversation
	// This ensures HistoryService has the complete conversation state
	history.ClearHistory()

	// Save all messages from the conversation (includes tool calls, tool results, etc.)
	for _, msg := range state.Conversation {
		history.AddToHistory(msg)
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
