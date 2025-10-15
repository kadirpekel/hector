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

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/protocol"
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

// Agent executes reasoning with strategies and implements pb.A2AServiceServer
type Agent struct {
	pb.UnimplementedA2AServiceServer

	name        string
	description string
	config      *config.AgentConfig
	services    reasoning.AgentServices
	taskWorkers chan struct{}
}

// NewAgent creates a new agent with services
func NewAgent(agentConfig *config.AgentConfig, componentMgr interface{}, registry *AgentRegistry) (*Agent, error) {
	compMgr, ok := componentMgr.(*component.ComponentManager)
	if !ok {
		return nil, fmt.Errorf("invalid component manager type")
	}

	agentConfig.Task.SetDefaults()

	services, err := NewAgentServicesWithRegistry(agentConfig, compMgr, registry)
	if err != nil {
		return nil, err
	}

	var taskWorkers chan struct{}
	if services.Task() != nil && agentConfig.Task.WorkerPool > 0 {
		// Bounded worker pool: limits concurrent task processing
		// Prevents resource exhaustion when many async tasks are submitted
		taskWorkers = make(chan struct{}, agentConfig.Task.WorkerPool)
	}
	// If WorkerPool is 0 (unlimited), taskWorkers remains nil = no limit

	return &Agent{
		name:        agentConfig.Name,
		description: agentConfig.Description,
		config:      agentConfig,
		services:    services,
		taskWorkers: taskWorkers,
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

// GetAgentCardSimple returns a simple agent card (for registry, not gRPC)
// This is a convenience method that doesn't require context/request
func (a *Agent) GetAgentCardSimple() *pb.AgentCard {
	return &pb.AgentCard{
		Name:        a.name,
		Description: a.description,
		Version:     "1.0.0",
		Capabilities: &pb.AgentCapabilities{
			Streaming: true,
		},
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
	outputCh := make(chan string, outputChannelBuffer)

	go func() {
		defer close(outputCh)

		startTime := time.Now()
		cfg := a.services.GetConfig()
		state := reasoning.NewReasoningState()
		state.Query = input // Original user query for strategies

		// Pass agent context to state
		state.CustomState["agent_name"] = a.name // Current agent name (for visibility filtering)

		// Pass sub_agents config to state for supervisor strategies
		if len(a.config.SubAgents) > 0 {
			state.CustomState["sub_agents"] = a.config.SubAgents
		}

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

		// Add current user input to conversation at the start of new messages
		// This ensures correct message ordering: [history..., USER, ASSISTANT, TOOL_RESULTS, ...]
		// Without this, saveToHistory would append USER at the end (wrong order)
		userMsg := protocol.CreateUserMessage(input)
		state.Conversation = append(state.Conversation, userMsg)

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
				// Create assistant message with text + tool call parts (pure A2A protocol)
				assistantMsg := &pb.Message{
					Role:    pb.Role_ROLE_AGENT,
					Content: []*pb.Part{},
				}

				// Add text part if present
				if text != "" {
					assistantMsg.Content = append(assistantMsg.Content,
						&pb.Part{Part: &pb.Part_Text{Text: text}})
				}

				// Add tool call parts using DataPart (native A2A protocol)
				for _, tc := range toolCalls {
					assistantMsg.Content = append(assistantMsg.Content,
						protocol.CreateToolCallPart(tc))
				}

				state.Conversation = append(state.Conversation, assistantMsg)

				results = a.executeTools(ctx, toolCalls, outputCh, cfg)

				// Add tool result messages using DataPart (native A2A protocol)
				for _, result := range results {
					// Convert reasoning.ToolResult to protocol.ToolResult
					errorStr := ""
					if result.Error != nil {
						errorStr = result.Error.Error()
					}
					a2aResult := &protocol.ToolResult{
						ToolCallID: result.ToolCallID,
						Content:    result.Content,
						Error:      errorStr,
					}
					resultMsg := &pb.Message{
						Role: pb.Role_ROLE_AGENT,
						Content: []*pb.Part{
							protocol.CreateToolResultPart(a2aResult),
						},
					}
					state.Conversation = append(state.Conversation, resultMsg)
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

		// Save to history (this now saves full a2a.Message objects)
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
	messages []*pb.Message,
	toolDefs []llms.ToolDefinition,
	outputCh chan<- string,
	cfg config.ReasoningConfig,
) (string, []*protocol.ToolCall, int, error) {
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
	toolCalls []*protocol.ToolCall,
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
			label := formatToolLabel(toolCall.Name, toolCall.Args)
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

	// Get existing history size to know where new messages start
	// state.Conversation structure: [old history..., USER(current), ...new messages during execution...]
	existingHistory, err := history.GetRecentHistory(sessionID)
	if err != nil {
		log.Printf("⚠️  Failed to get existing history: %v", err)
		return
	}
	existingCount := len(existingHistory)

	// Add assistant's final response if not already in conversation
	// This handles the case where the final iteration had no tool calls
	finalResponse := state.AssistantResponse.String()
	if finalResponse != "" {
		// Determine if we need to add the final response message
		// If conversation has messages after existingCount (beyond the USER message),
		// check if any are assistant messages - that means tool calls happened
		needsFinalResponse := true

		// Check if there are any assistant messages in the new messages
		// (new messages start at existingCount, which is the USER message)
		for i := existingCount + 1; i < len(state.Conversation); i++ {
			if state.Conversation[i].Role == pb.Role_ROLE_AGENT {
				// Found an assistant message - check if it has text or just tool calls/results
				hasText := protocol.ExtractTextFromMessage(state.Conversation[i]) != ""
				hasToolCalls := len(protocol.GetToolCallsFromMessage(state.Conversation[i])) > 0
				hasToolResults := len(protocol.GetToolResultsFromMessage(state.Conversation[i])) > 0

				// If the last assistant message has tool calls/results, tool calling happened
				// The final text is distributed across multiple assistant messages
				if hasToolCalls || hasToolResults || hasText {
					needsFinalResponse = false
					break
				}
			}
		}

		if needsFinalResponse {
			// Final iteration had no tool calls - add the complete response message
			assistantMsg := protocol.CreateTextMessage(pb.Role_ROLE_AGENT, finalResponse)
			state.Conversation = append(state.Conversation, assistantMsg)
		}
	}

	// Save only the NEW messages (those added during this execution)
	// New messages start at index existingCount (USER message + any tool call/result messages)
	// Session history includes user/assistant messages, including tool calls/results
	for i := existingCount; i < len(state.Conversation); i++ {
		msg := state.Conversation[i]

		// Only save user/assistant messages (skip system messages if any)
		if msg.Role == pb.Role_ROLE_USER || msg.Role == pb.Role_ROLE_AGENT {
			// Save message if it has text content OR tool calls/results
			textContent := protocol.ExtractTextFromMessage(msg)
			hasToolCalls := len(protocol.GetToolCallsFromMessage(msg)) > 0
			hasToolResults := len(protocol.GetToolResultsFromMessage(msg)) > 0

			if textContent != "" || hasToolCalls || hasToolResults {
				err := history.AddToHistory(sessionID, msg)
				if err != nil {
					log.Printf("⚠️  Failed to add message to history: %v", err)
				}
			}
		}
		// Skip system messages - they're for prompt construction only
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

// Agent implements pb.A2AServiceServer interface (checked by embedding)
// See agent_a2a_methods.go for the implementation
