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
	maxLLMRetries              = 3 // Maximum retry attempts for rate limits
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
func NewAgent(agentID string, agentConfig *config.AgentConfig, componentMgr interface{}, registry *AgentRegistry) (*Agent, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}

	compMgr, ok := componentMgr.(*component.ComponentManager)
	if !ok {
		return nil, fmt.Errorf("invalid component manager type")
	}

	agentConfig.Task.SetDefaults()

	services, err := NewAgentServicesWithRegistry(agentID, agentConfig, compMgr, registry)
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

		// Build state using builder pattern for clean, validated initialization
		state, err := reasoning.Builder().
			WithQuery(input).
			WithAgentName(a.name).
			WithSubAgents(a.config.SubAgents).
			WithOutputChannel(outputCh).
			WithShowThinking(cfg.ShowThinking).
			WithShowDebugInfo(cfg.ShowDebugInfo).
			WithServices(a.services).
			WithContext(ctx).
			Build()

		if err != nil {
			outputCh <- fmt.Sprintf("❌ Failed to initialize state: %v\n", err)
			return
		}

		// Ensure history is saved even on early return (cancellation, errors)
		// This captures partial work and maintains conversation continuity
		defer func() {
			a.saveToHistory(ctx, input, state, strategy, startTime)
		}()

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

			// Set up status notifier for summarization feedback (optional interface)
			// This uses Go's optional interface pattern - only implementations that
			// support status notifications need to implement StatusNotifiable
			if notifiable, ok := historyService.(reasoning.StatusNotifiable); ok {
				notifiable.SetStatusNotifier(func(message string) {
					if message != "" {
						// Newline before, continue on same line after
						outputCh <- "\n" + message + "\n"
					}
				})
			}

			// Get recent history and store using setter (immutable during this turn)
			recentHistory, err := historyService.GetRecentHistory(sessionID)
			if err != nil {
				outputCh <- fmt.Sprintf("⚠️  Failed to load conversation history: %v\n", err)
			} else if len(recentHistory) > 0 {
				state.SetHistory(recentHistory)
			}
		}

		// Add current user input to CurrentTurn
		// CurrentTurn will contain: USER message, then ASSISTANT + TOOL messages created during execution
		userMsg := protocol.CreateUserMessage(input)
		state.AddCurrentTurnMessage(userMsg)

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
		for state.Iteration() < maxIterations {
			// Atomically increment iteration counter
			currentIteration := state.NextIteration()

			// Check context cancellation
			select {
			case <-ctx.Done():
				outputCh <- fmt.Sprintf("\n⚠️  Canceled: %v\n", ctx.Err())
				return
			default:
			}

			if cfg.ShowDebugInfo {
				outputCh <- fmt.Sprintf("🤔 **Iteration %d/%d**\n", currentIteration, maxIterations)
			}

			// Strategy hook: prepare iteration
			if err := strategy.PrepareIteration(currentIteration, state); err != nil {
				outputCh <- fmt.Sprintf("Error preparing iteration: %v\n", err)
				return
			}

			// Get prompt slots from strategy and merge with config
			promptSlots := a.buildPromptSlots(strategy)

			// Get strategy-specific context injection (e.g., todos for ChainOfThought)
			additionalContext := strategy.GetContextInjection(state)

			// Build messages using PromptService (with slots and additional context)
			// AllMessages() combines History + CurrentTurn for the full conversation
			messages, err := a.services.Prompt().BuildMessages(ctx, input, promptSlots, state.AllMessages(), additionalContext)
			if err != nil {
				outputCh <- fmt.Sprintf("Error building messages: %v\n", err)
				return
			}

			// Call LLM with intelligent retry for rate limits
			// Respects Retry-After headers from rate limit responses
			var text string
			var toolCalls []*protocol.ToolCall
			var tokens int

			for attempt := 0; attempt <= maxLLMRetries; attempt++ {
				text, toolCalls, tokens, err = a.callLLM(ctx, messages, toolDefs, outputCh, cfg)

				if err == nil {
					// Success!
					break
				}

				// Check if this is a retryable error (rate limit)
				var retryErr *httpclient.RetryableError
				if !errors.As(err, &retryErr) {
					// Fatal error (auth, invalid request, etc.) - stop immediately
					outputCh <- fmt.Sprintf("❌ Fatal error: %v\n", err)
					return
				}

				// Rate limit error - check if we have retries left
				if attempt >= maxLLMRetries {
					outputCh <- fmt.Sprintf("❌ Rate limit exceeded after %d retries: %v\n", maxLLMRetries, err)
					return
				}

				// Use the exact retry time from rate limit headers
				waitTime := retryErr.RetryAfter
				if waitTime == 0 {
					waitTime = defaultRetryWaitSeconds * time.Second // Fallback if not specified
				}

				outputCh <- fmt.Sprintf("⏳ Rate limit exceeded (HTTP %d). Waiting %v before retry %d/%d...\n",
					retryErr.StatusCode, waitTime.Round(time.Second), attempt+1, maxLLMRetries)

				// Respect the rate limit - wait as instructed
				time.Sleep(waitTime)

				outputCh <- fmt.Sprintf("🔄 Retrying LLM call (attempt %d/%d)...\n", attempt+1, maxLLMRetries)
			}

			// Accumulate tokens using mutation method
			state.AddTokens(tokens)

			if cfg.ShowDebugInfo {
				outputCh <- fmt.Sprintf("\033[90m📝 Tokens used: %d (total: %d)\033[0m\n", tokens, state.TotalTokens())
			}

			// Accumulate assistant response text across iterations
			if text != "" {
				state.AppendResponse(text)
			}

			// Store first iteration's tool calls for metadata (atomic method handles the check)
			state.RecordFirstToolCalls(toolCalls)

			// Edge case: If LLM returned neither text nor tool calls, treat as error
			// This ensures we always have a response to save to history
			if text == "" && len(toolCalls) == 0 {
				text = "[Internal: Agent returned empty response]"
				state.AppendResponse(text)
				if cfg.ShowDebugInfo {
					outputCh <- "\033[90m⚠️  LLM returned empty response\033[0m\n"
				}
				// Will naturally stop on next iteration via ShouldStop (no tool calls)
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

				state.AddCurrentTurnMessage(assistantMsg)

				results = a.executeTools(ctx, toolCalls, outputCh, cfg)

				// Validate and truncate tool results to prevent context overflow
				// Large tool outputs (e.g., read_file on huge files) can exceed context limits
				results = a.truncateToolResults(results, cfg)

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
					state.AddCurrentTurnMessage(resultMsg)
				}
			}

			// Strategy hook: Additional processing (reflection, meta-cognition, etc.)
			// Call BEFORE checking ShouldStop so reflection happens even for final iteration
			if err := strategy.AfterIteration(currentIteration, text, toolCalls, results, state); err != nil {
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

		// Note: History saving is handled by defer at function start
		// This ensures partial work is saved even on early return/cancellation

		if cfg.ShowDebugInfo {
			outputCh <- fmt.Sprintf("\033[90m\n⏱️  Total time: %v | Tokens: %d | Iterations: %d\033[0m\n",
				time.Since(startTime), state.TotalTokens(), state.Iteration())
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

	// Check if structured output is configured for this agent
	if a.config.StructuredOutput != nil {
		// Use structured output mode (no streaming support)
		structConfig := &llms.StructuredOutputConfig{
			Format:           a.config.StructuredOutput.Format,
			Schema:           a.config.StructuredOutput.Schema,
			Enum:             a.config.StructuredOutput.Enum,
			Prefill:          a.config.StructuredOutput.Prefill,
			PropertyOrdering: a.config.StructuredOutput.PropertyOrdering,
		}

		text, toolCalls, tokens, err := llm.GenerateStructured(messages, toolDefs, structConfig)
		if err != nil {
			return "", nil, 0, err
		}

		// Send text to output
		if text != "" {
			outputCh <- text
		}

		return text, toolCalls, tokens, nil
	}

	if cfg.EnableStreaming != nil && *cfg.EnableStreaming {
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

		// Call LLM service (blocks until streaming completes)
		// Contract: GenerateStreaming writes to wrappedCh synchronously
		// and only returns when all writes are complete
		toolCalls, tokens, err := llm.GenerateStreaming(messages, toolDefs, wrappedCh)

		// Safe to close now - GenerateStreaming has returned, no more writes
		close(wrappedCh)
		<-done // Wait for forwarding goroutine to finish

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
	if len(toolCalls) > 0 && cfg.ShowToolExecution != nil && *cfg.ShowToolExecution {
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
		if cfg.ShowToolExecution != nil && *cfg.ShowToolExecution {
			label := formatToolLabel(toolCall.Name, toolCall.Args)
			outputCh <- fmt.Sprintf("🔧 %s", label)
		}

		// Execute tool
		result, metadata, err := tools.ExecuteToolCall(ctx, toolCall)
		resultContent := result
		if err != nil {
			resultContent = fmt.Sprintf("Error: %v", err)
			if cfg.ShowToolExecution != nil && *cfg.ShowToolExecution {
				outputCh <- " ❌\n"
			}
		} else {
			if cfg.ShowToolExecution != nil && *cfg.ShowToolExecution {
				outputCh <- " ✅\n"
			}
		}

		results = append(results, reasoning.ToolResult{
			ToolCall:   toolCall,
			Content:    resultContent,
			Error:      err,
			ToolCallID: toolCall.ID,
			ToolName:   toolCall.Name,
			Metadata:   metadata,
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

// truncateToolResults validates and truncates tool results to prevent context overflow
// Large tool outputs (e.g., reading huge files) can exceed LLM context limits
func (a *Agent) truncateToolResults(results []reasoning.ToolResult, cfg config.ReasoningConfig) []reasoning.ToolResult {
	// Maximum size for a single tool result (50KB default)
	// This prevents any single tool from consuming excessive context
	const maxToolResultSize = 50000

	truncated := make([]reasoning.ToolResult, len(results))
	copy(truncated, results)

	for i := range truncated {
		contentSize := len(truncated[i].Content)
		if contentSize > maxToolResultSize {
			// Truncate content and add informative suffix
			originalSize := contentSize
			truncated[i].Content = truncated[i].Content[:maxToolResultSize]

			// Add truncation notice
			suffix := fmt.Sprintf("\n\n⚠️  [Output truncated: showing %d of %d bytes. Use more specific queries or filters to see full content.]",
				maxToolResultSize, originalSize)
			truncated[i].Content += suffix

			// Log truncation if debug enabled
			if cfg.ShowDebugInfo {
				log.Printf("⚠️  Tool result truncated: %s (%d → %d bytes)",
					truncated[i].ToolName, originalSize, maxToolResultSize)
			}
		}
	}

	return truncated
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

	// Add assistant's final response if not already added
	// This handles the case where the final iteration had no tool calls
	// Uses explicit flag instead of complex message inspection
	if !state.IsFinalResponseAdded() {
		finalResponse := state.GetAssistantResponse()
		if finalResponse != "" {
			// Final iteration had no tool calls - add the complete response message
			assistantMsg := protocol.CreateTextMessage(pb.Role_ROLE_AGENT, finalResponse)
			state.AddCurrentTurnMessage(assistantMsg)
			state.MarkFinalResponseAdded()
		}
	}

	// Save all messages from CurrentTurn as a BATCH (much simpler than before!)
	// CurrentTurn contains: USER message + ASSISTANT responses + TOOL calls/results
	// Using batch save ensures summarization is checked ONCE per turn, not per message
	currentTurn := state.GetCurrentTurn()

	// Save ALL messages from current turn (no filtering)
	// Strategy will decide how to load them back
	// This includes: USER, AGENT, SYSTEM, UNSPECIFIED (summaries), TOOL messages
	messagesToSave := make([]*pb.Message, 0, len(currentTurn))
	for _, msg := range currentTurn {
		// Save message if it has ANY content
		textContent := protocol.ExtractTextFromMessage(msg)
		hasToolCalls := len(protocol.GetToolCallsFromMessage(msg)) > 0
		hasToolResults := len(protocol.GetToolResultsFromMessage(msg)) > 0

		if textContent != "" || hasToolCalls || hasToolResults {
			messagesToSave = append(messagesToSave, msg)
		}
		// Note: Empty messages are skipped regardless of role
	}

	// Batch save - summarization is checked ONCE at turn boundary
	if len(messagesToSave) > 0 {
		err := history.AddBatchToHistory(sessionID, messagesToSave)
		if err != nil {
			log.Printf("⚠️  Failed to save messages to history: %v", err)
		}
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

	// INJECT SELF-REFLECTION: If enabled, add thinking tags to reasoning instructions
	if a.config.Reasoning.EnableSelfReflection {
		selfReflectionPrompt := `
<self_reflection>
Before taking actions, output your internal reasoning using <thinking> tags:
<thinking>
- Analyze the user's request and break it down
- Consider what information you need
- Plan your approach step by step
- Reason about potential challenges
</thinking>

After tool execution, reflect on results:
<thinking>
- Evaluate what worked and what didn't
- Assess progress toward the goal
- Decide next steps based on outcomes
</thinking>

Your thinking will be displayed to help users understand your reasoning process.
Make your thinking natural, concise, and focused on the current task.
</self_reflection>
`
		// Append to reasoning instructions
		if strategySlots.ReasoningInstructions != "" {
			strategySlots.ReasoningInstructions += "\n\n" + selfReflectionPrompt
		} else {
			strategySlots.ReasoningInstructions = selfReflectionPrompt
		}
	}

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
