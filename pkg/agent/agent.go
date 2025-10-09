// Package agent provides the core AI agent implementation for the Hector framework.
//
// This package implements the A2A (Agent-to-Agent) protocol interface and provides
// reasoning orchestration, tool integration, and session management capabilities.
//
// Key components:
//   - Agent: Main agent implementation with reasoning loop
//   - AgentRegistry: Manages agent registration and discovery
//   - A2AAgent: Wrapper for external A2A agents
//   - AgentCallTool: Enables multi-agent orchestration
//
// Example usage:
//
//	agent, err := agent.NewAgent(agentConfig, globalConfig)
//	registry := agent.NewAgentRegistry()
//	registry.RegisterAgent("my_agent", agent, config, capabilities)
package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// ============================================================================
// CONSTANTS
// ============================================================================

const (
	outputChannelBuffer        = 100
	historyRetentionMultiplier = 10
	defaultRetryWaitSeconds    = 120
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
// PUBLIC API - PURE A2A METHODS ONLY
// ============================================================================

// ClearHistory clears the conversation history for a specific session
func (a *Agent) ClearHistory(sessionID string) error {
	history := a.services.History()
	if history != nil {
		return history.ClearHistory(sessionID)
	}
	return nil
}

// ============================================================================
// PURE A2A INTERFACE IMPLEMENTATION
// Agent implements a2a.Agent interface directly - no wrapper needed!
// ============================================================================

// GetAgentCard returns the agent's capability card
func (a *Agent) GetAgentCard() *a2a.AgentCard {
	return &a2a.AgentCard{
		Name:               a.name,
		Description:        a.description,
		Version:            "1.0.0",
		PreferredTransport: "http+json",
		Capabilities: a2a.AgentCapabilities{
			Streaming: true,
			MultiTurn: true,
		},
	}
}

// ExecuteTask implements a2a.Agent.ExecuteTask
func (a *Agent) ExecuteTask(ctx context.Context, task *a2a.Task) (*a2a.Task, error) {
	// Extract user message text
	userText := a2a.ExtractTextFromTask(task)
	if userText == "" && len(task.Messages) > 0 {
		// Get the last user message
		for i := len(task.Messages) - 1; i >= 0; i-- {
			if task.Messages[i].Role == a2a.MessageRoleUser {
				for _, part := range task.Messages[i].Parts {
					if part.Type == a2a.PartTypeText {
						userText = part.Text
						break
					}
				}
				if userText != "" {
					break
				}
			}
		}
	}

	// Create strategy based on config
	strategy, err := reasoning.CreateStrategy(a.config.Reasoning.Engine, a.config.Reasoning)
	if err != nil {
		task.Status.State = a2a.TaskStateFailed
		task.Status.UpdatedAt = time.Now()
		task.Error = &a2a.TaskError{
			Code:    "strategy_error",
			Message: err.Error(),
		}
		return task, nil
	}

	// Execute reasoning loop
	streamCh, err := a.execute(ctx, userText, strategy)
	if err != nil {
		task.Status.State = a2a.TaskStateFailed
		task.Status.UpdatedAt = time.Now()
		task.Error = &a2a.TaskError{
			Code:    "execution_error",
			Message: err.Error(),
		}
		return task, nil
	}

	// Collect all streaming output
	var fullResponse strings.Builder
	for chunk := range streamCh {
		fullResponse.WriteString(chunk)
	}

	// Add assistant response to task
	task.Messages = append(task.Messages, a2a.Message{
		Role: a2a.MessageRoleAssistant,
		Parts: []a2a.Part{
			{
				Type: a2a.PartTypeText,
				Text: fullResponse.String(),
			},
		},
	})
	task.Status.State = a2a.TaskStateCompleted
	task.Status.UpdatedAt = time.Now()

	return task, nil
}

// ExecuteTaskStreaming implements a2a.Agent.ExecuteTaskStreaming
func (a *Agent) ExecuteTaskStreaming(ctx context.Context, task *a2a.Task) (<-chan a2a.StreamEvent, error) {
	// Extract user message text
	userText := a2a.ExtractTextFromTask(task)
	if userText == "" && len(task.Messages) > 0 {
		// Get the last user message
		for i := len(task.Messages) - 1; i >= 0; i-- {
			if task.Messages[i].Role == a2a.MessageRoleUser {
				for _, part := range task.Messages[i].Parts {
					if part.Type == a2a.PartTypeText {
						userText = part.Text
						break
					}
				}
				if userText != "" {
					break
				}
			}
		}
	}

	// Create strategy based on config
	strategy, err := reasoning.CreateStrategy(a.config.Reasoning.Engine, a.config.Reasoning)
	if err != nil {
		return nil, err
	}

	// Execute reasoning loop
	hectorStream, err := a.execute(ctx, userText, strategy)
	if err != nil {
		return nil, err
	}

	// Convert to A2A StreamEvents
	eventCh := make(chan a2a.StreamEvent, 10)

	go func() {
		defer close(eventCh)

		var fullText strings.Builder

		for chunk := range hectorStream {
			fullText.WriteString(chunk)

			// Send text chunk as message event
			eventCh <- a2a.StreamEvent{
				Type:   a2a.StreamEventTypeMessage,
				TaskID: task.ID,
				Message: &a2a.Message{
					Role: a2a.MessageRoleAssistant,
					Parts: []a2a.Part{
						{
							Type: a2a.PartTypeText,
							Text: chunk,
						},
					},
				},
				Timestamp: time.Now(),
			}
		}

		// Send completion status
		eventCh <- a2a.StreamEvent{
			Type:   a2a.StreamEventTypeStatus,
			TaskID: task.ID,
			Status: &a2a.TaskStatus{
				State:     a2a.TaskStateCompleted,
				UpdatedAt: time.Now(),
			},
			Timestamp: time.Now(),
		}
	}()

	return eventCh, nil
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
	outputCh := make(chan string, outputChannelBuffer)

	go func() {
		defer close(outputCh)

		startTime := time.Now()
		cfg := a.services.GetConfig()
		state := reasoning.NewReasoningState()
		state.Query = input // Original user query for strategies

		// Restore conversation history from HistoryService (interactive mode)
		historyService := a.services.History()
		if historyService != nil {
			// Extract sessionID from context
			sessionID := ""
			if sessionIDValue := ctx.Value("sessionID"); sessionIDValue != nil {
				if sid, ok := sessionIDValue.(string); ok {
					sessionID = sid
				}
			}

			// Set up status notifier for summarization feedback
			if memService, ok := historyService.(*memory.MemoryService); ok {
				memService.SetStatusNotifier(func(message string) {
					if message != "" {
						// Newline before, continue on same line after
						outputCh <- "\n" + message + "\n"
					}
				})
			}

			// Get recent history and restore to conversation
			recentHistory, err := historyService.GetRecentHistory(sessionID)
			if err != nil {
				outputCh <- fmt.Sprintf("⚠️  Failed to load conversation history: %v\n", err)
			} else if len(recentHistory) > 0 {
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
				outputCh <- fmt.Sprintf("\n⚠️  Canceled: %v\n", ctx.Err())
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

			// Call LLM with agent-level retry for transient errors
			text, toolCalls, tokens, err := a.callLLM(ctx, messages, toolDefs, outputCh, cfg)
			if err != nil {
				// Check if error is a typed RetryableError
				var retryErr *httpclient.RetryableError
				if errors.As(err, &retryErr) {
					// Use the exact retry time from the error
					waitTime := retryErr.RetryAfter
					if waitTime == 0 {
						waitTime = defaultRetryWaitSeconds * time.Second // Fallback if not specified
					}

					outputCh <- fmt.Sprintf("⏳ Rate limit exceeded (HTTP %d). Waiting %v before retry...\n",
						retryErr.StatusCode, waitTime.Round(time.Second))
					time.Sleep(waitTime)

					// Retry once at agent level
					text, toolCalls, tokens, err = a.callLLM(ctx, messages, toolDefs, outputCh, cfg)
					if err != nil {
						outputCh <- fmt.Sprintf("❌ LLM still unavailable after retry: %v\n", err)
						return
					}
					outputCh <- "✅ Retry successful, continuing...\n"
				} else {
					// Fatal error (auth, invalid request, etc.) - stop immediately
					outputCh <- fmt.Sprintf("❌ Fatal error: %v\n", err)
					return
				}
			}

			state.TotalTokens += tokens

			if cfg.ShowDebugInfo {
				outputCh <- fmt.Sprintf("\033[90m📝 Tokens used: %d (total: %d)\033[0m\n", tokens, state.TotalTokens)
			}

			// Accumulate assistant response text across iterations
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
				// IMPORTANT: Add assistant message WITH tool_calls BEFORE adding tool results
				// This is required by OpenAI/Anthropic for proper tool calling round-trip
				assistantMsg := llms.Message{
					Role:      "assistant",
					Content:   text,
					ToolCalls: toolCalls,
				}
				state.Conversation = append(state.Conversation, assistantMsg)

				results = a.executeTools(ctx, toolCalls, outputCh, cfg)

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
				// Optional: Verify task completion before stopping (if enabled)
				if cfg.EnableCompletionVerification && len(toolCalls) == 0 {
					// Only verify completion when there are no more tool calls
					assessment, err := reasoning.AssessTaskCompletion(ctx, input, state.AssistantResponse.String(), a.services)
					if err == nil && cfg.ShowDebugInfo {
						// Display completion assessment
						outputCh <- "\033[90m\n🎯 **Completion Assessment:**\n"
						outputCh <- fmt.Sprintf("  - Complete: %v (%.0f%% confident)\n", assessment.IsComplete, assessment.Confidence*100)
						outputCh <- fmt.Sprintf("  - Quality: %s\n", assessment.Quality)
						if len(assessment.MissingActions) > 0 {
							outputCh <- fmt.Sprintf("  - Missing: %v\n", assessment.MissingActions)
						}
						outputCh <- fmt.Sprintf("  - Recommendation: %s\n", assessment.Recommendation)
						outputCh <- fmt.Sprintf("  - Reasoning: %s\n", assessment.Reasoning)
						outputCh <- "\033[0m"
					}

					// If not complete, continue for one more iteration
					if err == nil && !assessment.IsComplete && assessment.Recommendation == "continue" {
						if cfg.ShowDebugInfo {
							outputCh <- "\033[90m⚠️  Task not fully complete, continuing...\033[0m\n"
						}
						continue
					}
				}

				if cfg.ShowDebugInfo {
					outputCh <- "\033[90m\n\n✅ **Reasoning complete**\033[0m\n"
				}
				break
			}
		}

		// Save to history (this now saves full llms.Message objects)
		a.saveToHistory(ctx, input, state, strategy, startTime)

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

// isRetryableError checks if an error is transient and worth retrying
// nolint:unused // Reserved for future use
func (a *Agent) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check for HTTP status codes indicating transient issues
	retryablePatterns := []string{
		"429", // Too Many Requests (rate limit)
		"500", // Internal Server Error
		"502", // Bad Gateway
		"503", // Service Unavailable
		"504", // Gateway Timeout
		"rate limit",
		"rate_limit",
		"timeout",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}

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
		// Streaming mode - capture streamed text for history
		var streamedText strings.Builder
		wrappedCh := make(chan string, 100)
		done := make(chan struct{})

		// Goroutine to capture and forward streamed text
		go func() {
			defer close(done)
			for chunk := range wrappedCh {
				streamedText.WriteString(chunk)
				// Safely send to outputCh (might be closed)
				select {
				case outputCh <- chunk:
				case <-ctx.Done():
					return
				}
			}
		}()

		toolCalls, tokens, err := llm.GenerateStreaming(messages, toolDefs, wrappedCh)
		close(wrappedCh)
		<-done // Wait for goroutine to finish

		if err != nil {
			return "", nil, 0, err
		}

		// Return the accumulated streamed text for history
		return streamedText.String(), toolCalls, tokens, nil
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

	// Note: Tool labels are NOT shown here in non-streaming mode
	// They will be shown by executeTools() when tools are actually executed
	// This creates a consistent "tool name + status" display pattern

	return text, toolCalls, tokens, nil
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

	// Add newline before tools section (once, before all tools)
	if len(toolCalls) > 0 && cfg.ShowToolExecution {
		outputCh <- "\n"
	}

	for _, toolCall := range toolCalls {
		// Check cancellation before each tool
		select {
		case <-ctx.Done():
			return results
		default:
		}

		// Show tool label before execution (both streaming and non-streaming)
		// This ensures clean "🔧 tool ✅" pairing for each tool
		if cfg.ShowToolExecution {
			label := formatToolLabel(toolCall.Name, toolCall.Arguments)
			outputCh <- fmt.Sprintf("🔧 %s", label)
		}

		// Execute tool
		result, err := tools.ExecuteToolCall(ctx, toolCall)
		resultContent := result
		if err != nil {
			resultContent = fmt.Sprintf("Error: %v", err)
			if cfg.ShowToolExecution {
				outputCh <- " ❌\n"
			}
		} else {
			if cfg.ShowToolExecution {
				outputCh <- " ✅\n"
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

// formatToolLabel creates a concise label for tool execution
// Shared logic for both streaming and non-streaming modes
func formatToolLabel(toolName string, args map[string]interface{}) string {
	switch toolName {
	case "agent_call":
		if agentName, ok := args["agent"].(string); ok {
			return fmt.Sprintf("%s → %s", toolName, agentName)
		}
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

// saveToHistory saves the conversation to history service
func (a *Agent) saveToHistory(
	ctx context.Context,
	input string,
	state *reasoning.ReasoningState,
	strategy reasoning.ReasoningStrategy,
	startTime time.Time,
) {
	history := a.services.History()
	if history == nil {
		return
	}

	// Extract sessionID from context
	sessionID := ""
	if sessionIDValue := ctx.Value("sessionID"); sessionIDValue != nil {
		if sid, ok := sessionIDValue.(string); ok {
			sessionID = sid
		}
	}

	// Add user's input message to conversation (if not already there)
	if len(state.Conversation) == 0 || state.Conversation[len(state.Conversation)-1].Role != "user" {
		userMsg := llms.Message{
			Role:    "user",
			Content: input,
		}
		state.Conversation = append(state.Conversation, userMsg)
	}

	// Add assistant's final response to conversation (if not already added via tool calls)
	finalResponse := state.AssistantResponse.String()
	if finalResponse != "" {
		// Check if the last message is already an assistant message
		// (it would have been added in-loop if there were tool calls in the final iteration)
		lastIsAssistant := len(state.Conversation) > 0 && state.Conversation[len(state.Conversation)-1].Role == "assistant"

		if !lastIsAssistant {
			// Final iteration had no tool calls - add the response message here
			assistantMsg := llms.Message{
				Role:    "assistant",
				Content: finalResponse,
			}
			state.Conversation = append(state.Conversation, assistantMsg)
		}
	}

	// Save NEW messages from this execution
	// Note: state.Conversation already includes loaded history + new messages
	// We need to save only the NEW messages (after the loaded history)

	// Get current history size to know where new messages start
	existingHistory, err := history.GetRecentHistory(sessionID)
	if err != nil {
		log.Printf("⚠️  Failed to get existing history: %v", err)
		return
	}
	existingCount := len(existingHistory)

	// Save only the new messages (those added during this execution)
	// Filter out tool messages and messages with empty content
	// Session history is for user/assistant conversation only
	for i := existingCount; i < len(state.Conversation); i++ {
		msg := state.Conversation[i]

		// Only save user/assistant messages with content
		if msg.Role == "user" || msg.Role == "assistant" {
			// Skip messages with empty content (e.g., when LLM calls tool without preamble)
			if msg.Content != "" {
				err := history.AddToHistory(sessionID, msg)
				if err != nil {
					log.Printf("⚠️  Failed to add message to history: %v", err)
				}
			}
		}
		// Skip system and tool messages - they're for LLM API, not user history
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

// ============================================================================
// COMPILE-TIME CHECK
// ============================================================================

// Ensure Agent implements a2a.Agent interface directly
var _ a2a.Agent = (*Agent)(nil)
