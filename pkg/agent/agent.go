package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/tools"
	"go.opentelemetry.io/otel/trace"
)

const (
	outputChannelBuffer        = 100
	historyRetentionMultiplier = 10
	defaultRetryWaitSeconds    = 120
	maxLLMRetries              = 3
)

type Agent struct {
	pb.UnimplementedA2AServiceServer

	id                 string // Agent ID (config key, URL-safe)
	name               string // Display name
	description        string
	config             *config.AgentConfig
	componentManager   *component.ComponentManager // For accessing global config
	services           reasoning.AgentServices
	taskWorkers        chan struct{}
	baseURL            string   // Server base URL for agent card URL construction
	preferredTransport string   // Preferred A2A transport (grpc, json-rpc, rest)
	subAgents          []string // Sub-agents for multi-agent scenarios (set from config or builder)

	// Human-in-the-loop support (A2A Protocol Section 6.3 - INPUT_REQUIRED state)
	taskAwaiter      *TaskAwaiter                  // Handles paused tasks waiting for input
	activeExecutions map[string]context.CancelFunc // Track active task executions for cancellation
	executionsMu     sync.RWMutex                  // Protects activeExecutions
}

func (a *Agent) ClearHistory(sessionID string) error {
	history := a.services.History()
	if history != nil {
		return history.ClearHistory(sessionID)
	}
	return nil
}

func (a *Agent) GetAgentCardSimple() *pb.AgentCard {
	return &pb.AgentCard{
		Name:               a.name,
		Description:        a.description,
		Version:            a.getVersion(),
		PreferredTransport: a.preferredTransport,
		Capabilities: &pb.AgentCapabilities{
			Streaming: true,
		},

		DefaultInputModes:  a.getInputModes(),
		DefaultOutputModes: a.getOutputModes(),
		Skills:             a.getSkills(),
		Provider:           a.getProvider(),
		DocumentationUrl:   a.getDocumentationURL(),
	}
}

func (a *Agent) getVersion() string {
	if a.config != nil && a.config.A2A != nil && a.config.A2A.Version != "" {
		return a.config.A2A.Version
	}
	return "0.3.0"
}

func (a *Agent) getInputModes() []string {
	if a.config != nil && a.config.A2A != nil && len(a.config.A2A.InputModes) > 0 {
		return a.config.A2A.InputModes
	}

	return []string{"text/plain", "application/json"}
}

func (a *Agent) getOutputModes() []string {
	if a.config != nil && a.config.A2A != nil && len(a.config.A2A.OutputModes) > 0 {
		return a.config.A2A.OutputModes
	}

	// Default output modes including AG-UI support
	return []string{"text/plain", "application/json", "application/x-agui-events"}
}

func (a *Agent) getSkills() []*pb.AgentSkill {
	if a.config != nil && a.config.A2A != nil && len(a.config.A2A.Skills) > 0 {

		skills := make([]*pb.AgentSkill, len(a.config.A2A.Skills))
		for i, skill := range a.config.A2A.Skills {
			skills[i] = &pb.AgentSkill{
				Id:          skill.ID,
				Name:        skill.Name,
				Description: skill.Description,
				Tags:        skill.Tags,
				Examples:    skill.Examples,
			}
		}
		return skills
	}

	return []*pb.AgentSkill{
		{
			Id:          "general-assistance",
			Name:        "General Assistance",
			Description: a.description,
			Tags:        []string{"conversation", "assistance"},
		},
	}
}

func (a *Agent) getProvider() *pb.AgentProvider {
	if a.config != nil && a.config.A2A != nil && a.config.A2A.Provider != nil {
		return &pb.AgentProvider{
			Organization: a.config.A2A.Provider.Name,
			Url:          a.config.A2A.Provider.URL,
		}
	}
	return nil
}

func (a *Agent) getDocumentationURL() string {
	if a.config != nil && a.config.A2A != nil {
		return a.config.A2A.DocumentationURL
	}
	return ""
}

// Helper function to create a text part
func createTextPart(text string) *pb.Part {
	return &pb.Part{Part: &pb.Part_Text{Text: text}}
}

// safeSendPart sends a part to the output channel with backpressure handling.
// If the channel is full or the context is cancelled, it returns an error.
// This prevents goroutine leaks when clients disconnect.
func safeSendPart(ctx context.Context, outputCh chan<- *pb.Part, part *pb.Part) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case outputCh <- part:
		return nil
	default:
		// Channel is full - try once more with timeout to avoid blocking indefinitely
		select {
		case <-ctx.Done():
			return ctx.Err()
		case outputCh <- part:
			return nil
		case <-time.After(100 * time.Millisecond):
			// Channel still full after timeout - log warning but don't block
			// This prevents goroutine leaks on client disconnect
			return fmt.Errorf("output channel full, client may have disconnected")
		}
	}
}

// Helper functions for context value extraction
func getSessionIDFromContext(ctx context.Context) string {
	if sessionIDValue := ctx.Value(SessionIDKey); sessionIDValue != nil {
		if sid, ok := sessionIDValue.(string); ok {
			return sid
		}
	}
	return ""
}

func getTaskIDFromContext(ctx context.Context) string {
	if taskIDValue := ctx.Value(taskIDContextKey); taskIDValue != nil {
		if tid, ok := taskIDValue.(string); ok {
			return tid
		}
	}
	return ""
}

func getUserDecisionFromContext(ctx context.Context) string {
	if decisionValue := ctx.Value(userDecisionContextKey); decisionValue != nil {
		if decision, ok := decisionValue.(string); ok {
			return decision
		}
	}
	return ""
}

// getInputTimeout returns the configured input timeout or default
func (a *Agent) getInputTimeout() time.Duration {
	if a.config.Task != nil && a.config.Task.InputTimeout > 0 {
		return time.Duration(a.config.Task.InputTimeout) * time.Second
	}
	return 10 * time.Minute // Default
}

func (a *Agent) execute(
	ctx context.Context,
	input string,
	strategy reasoning.ReasoningStrategy,
) (<-chan *pb.Part, error) {
	outputCh := make(chan *pb.Part, outputChannelBuffer)

	go func() {
		defer close(outputCh)

		startTime := time.Now()
		cfg := a.services.GetConfig()

		// Get LLM name from config if available, otherwise use empty string
		llmName := ""
		if a.config != nil {
			llmName = a.config.LLM
		}
		spanCtx, span := startAgentSpan(ctx, a.name, llmName, input)
		defer span.End()

		// Add ShowThinking flag to context for LLM providers
		showThinking := config.BoolValue(cfg.ShowThinking, false)
		spanCtx = context.WithValue(spanCtx, protocol.ShowThinkingKey, showThinking)

		// Get sub-agents (from config or builder - stored directly on agent)
		state, err := reasoning.Builder().
			WithQuery(input).
			WithAgentName(a.name).
			WithSubAgents(a.subAgents).
			WithOutputChannel(outputCh).
			WithShowThinking(showThinking).
			WithServices(a.services).
			WithContext(spanCtx).
			Build()

		if err != nil {
			span.RecordError(err)
			recordAgentMetrics(spanCtx, time.Since(startTime), 0, err)
			if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Error: Failed to initialize state: %v\n", err))); sendErr != nil {
				slog.Error("Failed to send error to output channel", "agent", a.name, "error", sendErr)
			}
			return
		}

		defer func() {
			a.saveToHistory(spanCtx, input, state, strategy, startTime)

			duration := time.Since(startTime)
			recordAgentMetrics(spanCtx, duration, state.TotalTokens(), nil)
		}()

		historyService := a.services.History()
		if historyService != nil {
			sessionID := getSessionIDFromContext(ctx)

			if notifiable, ok := historyService.(reasoning.StatusNotifiable); ok {
				notifiable.SetStatusNotifier(func(message string) {
					if message != "" {
						if sendErr := safeSendPart(ctx, outputCh, createTextPart("\n"+message+"\n")); sendErr != nil {
							slog.Error("Failed to send status notification", "agent", a.name, "error", sendErr)
						}
					}
				})
			}

			recentHistory, err := historyService.GetRecentHistory(sessionID)
			if err != nil {
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Warning: Failed to load conversation history: %v\n", err))); sendErr != nil {
					slog.Error("Failed to send warning", "agent", a.name, "error", sendErr)
				}
			} else if len(recentHistory) > 0 {
				state.SetHistory(recentHistory)
			}
		}

		userMsg := protocol.CreateUserMessage(input)
		state.AddCurrentTurnMessage(userMsg)

		maxIterations := a.getMaxIterations(cfg)

		toolService := a.services.Tools()
		toolDefs := toolService.GetAvailableTools()

		for state.Iteration() < maxIterations {

			currentIteration := state.NextIteration()

			select {
			case <-ctx.Done():
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("\nCanceled: %v\n", ctx.Err()))); sendErr != nil {
					slog.Error("Failed to send cancellation message", "agent", a.name, "error", sendErr)
				}
				return
			default:
			}

			if err := strategy.PrepareIteration(currentIteration, state); err != nil {
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Error preparing iteration: %v\n", err))); sendErr != nil {
					slog.Error("Failed to send error", "agent", a.name, "error", sendErr)
				}
				return
			}

			promptSlots := a.buildPromptSlots(strategy)

			additionalContext := strategy.GetContextInjection(state)

			messages, err := a.services.Prompt().BuildMessages(ctx, input, promptSlots, state.AllMessages(), additionalContext)
			if err != nil {
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Error building messages: %v\n", err))); sendErr != nil {
					slog.Error("Failed to send error", "agent", a.name, "error", sendErr)
				}
				return
			}

			// Call LLM with retry logic for rate limits
			text, toolCalls, tokens, err := a.callLLMWithRetry(spanCtx, messages, toolDefs, outputCh, cfg, span)
			if err != nil {
				span.RecordError(err)
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Error: LLM call failed: %v\n", err))); sendErr != nil {
					slog.Error("Failed to send error", "agent", a.name, "error", sendErr)
				}
				return
			}

			state.AddTokens(tokens)

			if text != "" {
				state.AppendResponse(text)
			}

			state.RecordFirstToolCalls(toolCalls)

			// Ensure we have some response content
			if text == "" && len(toolCalls) == 0 {
				text = "[Internal: Agent returned empty response]"
				state.AppendResponse(text)
			}

			// Process tool calls if any
			var results []reasoning.ToolResult
			if len(toolCalls) > 0 {
				var shouldContinue bool
				ctx, results, shouldContinue, err = a.processToolCalls(ctx, text, toolCalls, state, outputCh, cfg)
				if err != nil {
					if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Error processing tool calls: %v\n", err))); sendErr != nil {
						slog.Error("Failed to send error", "agent", a.name, "error", sendErr)
					}
					return
				}
				if shouldContinue {
					// Tool approval is pending, wait for next iteration
					continue
				}
			}

			if err := strategy.AfterIteration(currentIteration, text, toolCalls, results, state); err != nil {
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Error in strategy processing: %v\n", err))); sendErr != nil {
					slog.Error("Failed to send error", "agent", a.name, "error", sendErr)
				}
				return
			}

			// Interval-based checkpointing (if enabled)
			// Checkpoint at end of iteration for crash recovery
			if checkpointInterval := a.getCheckpointInterval(); checkpointInterval > 0 {
				if a.shouldCheckpointInterval(currentIteration, checkpointInterval) {
					taskID := getTaskIDFromContext(ctx)
					if taskID != "" {
						// Background checkpoint - don't block execution
						// Errors are logged but don't stop execution
						if err := a.checkpointExecution(
							ctx,
							taskID,
							PhaseIterationEnd,
							CheckpointTypeInterval,
							state,
							nil, // No pending tool call
						); err != nil {
							slog.Warn("Failed to checkpoint at iteration", "agent", a.name, "iteration", currentIteration, "error", err)
							// Continue execution even if checkpoint fails
						}
					}
				}
			}

			if strategy.ShouldStop(text, toolCalls, state) {
				break
			}
		}
	}()

	return outputCh, nil
}

// processToolCalls handles tool approval, execution, and result processing
// Returns: (updatedContext, results, shouldContinue, error)
// - updatedContext: context with user decision if approval was handled
// - shouldContinue: true if waiting for user approval, false to proceed with iteration
func (a *Agent) processToolCalls(
	ctx context.Context,
	text string,
	toolCalls []*protocol.ToolCall,
	state *reasoning.ReasoningState,
	outputCh chan<- *pb.Part,
	cfg config.ReasoningConfig,
) (context.Context, []reasoning.ToolResult, bool, error) {
	// Get tool configs for approval checking
	// componentManager may be nil for agents built programmatically
	var toolConfigs map[string]*config.ToolConfig
	if a.componentManager != nil {
		globalConfig := a.componentManager.GetGlobalConfig()
		if globalConfig != nil {
			toolConfigs = globalConfig.Tools
		}
	}

	// Check if any tools require approval (A2A Protocol Section 6.3 - INPUT_REQUIRED state)
	approvalResult, err := a.filterToolCallsWithApproval(ctx, toolCalls, toolConfigs)
	if err != nil {
		return ctx, nil, false, fmt.Errorf("checking tool approval: %w", err)
	}

	// Handle tool approval request if needed
	if approvalResult.NeedsUserInput {
		var shouldContinue bool
		ctx, shouldContinue, err = a.handleToolApprovalRequest(ctx, approvalResult, outputCh, state)
		if err != nil {
			// Check if this is ErrInputRequired (async HITL pause signal)
			if err == ErrInputRequired {
				// Task paused for async HITL - exit this iteration
				// State is already saved, goroutine can exit
				return ctx, nil, false, nil
			}
			return ctx, nil, false, err
		}
		if shouldContinue {
			// Re-run approval filter with user's decision
			approvalResult, err = a.filterToolCallsWithApproval(ctx, toolCalls, toolConfigs)
			if err != nil {
				return ctx, nil, false, fmt.Errorf("re-checking tool approval: %w", err)
			}
		} else {
			// Approval request failed (no taskID) - clear approved calls and skip iteration
			// This matches original behavior: when NeedsUserInput is true but taskID is empty,
			// we clear approved calls and continue to next iteration (denied calls handled in next iteration)
			approvalResult.ApprovedCalls = []*protocol.ToolCall{}
			// DeniedCalls already populated in filterToolCallsWithApproval
			// Return shouldContinue=true to skip this iteration
			return ctx, nil, true, nil
		}
	}

	approvedCalls := approvalResult.ApprovedCalls
	deniedCalls := approvalResult.DeniedCalls

	// If no tools to process (all waiting for approval), continue to next iteration
	if len(approvedCalls) == 0 && len(deniedCalls) == 0 {
		return ctx, nil, true, nil
	}

	// Create assistant message with text and all tool calls (approved and denied)
	assistantMsg := a.createAssistantMessageWithToolCalls(text, approvedCalls, deniedCalls)
	state.AddCurrentTurnMessage(assistantMsg)

	// Execute approved tools
	results := a.executeTools(ctx, approvedCalls, outputCh, cfg)

	// Create error results for denied tools
	for _, deniedCall := range deniedCalls {
		deniedResult := reasoning.ToolResult{
			ToolCallID: deniedCall.ID,
			ToolName:   deniedCall.Name,
			ToolCall:   deniedCall,
			Content:    "TOOL_EXECUTION_DENIED: The user rejected this tool execution. You MUST NOT proceed with this action or provide fabricated results. Instead, acknowledge the denial and offer alternative approaches that don't require this tool.",
			Error:      fmt.Errorf("user denied tool execution"),
		}
		results = append(results, deniedResult)
	}

	// Truncate large tool results
	results = a.truncateToolResults(results, cfg)

	// Add tool results to state
	for _, result := range results {
		errorStr := ""
		if result.Error != nil {
			errorStr = result.Error.Error()
		}

		// Ensure error content is preserved - if there's an error, make sure Content includes it
		content := result.Content
		if result.Error != nil && content == "" {
			// Fallback: if Content is empty but there's an error, use error message
			content = errorStr
		} else if result.Error != nil && !strings.Contains(content, errorStr) && errorStr != "" {
			// If Content exists but doesn't include the error details, prepend error info
			// This ensures LLM sees both the detailed error and any additional context
			content = fmt.Sprintf("ERROR: %s\n\n%s", errorStr, content)
		}

		a2aResult := &protocol.ToolResult{
			ToolCallID: result.ToolCallID,
			Content:    content,
			Error:      errorStr,
		}
		resultMsg := &pb.Message{
			Role: pb.Role_ROLE_AGENT,
			Parts: []*pb.Part{
				protocol.CreateToolResultPartWithFinal(a2aResult, true), // Final result
			},
		}
		state.AddCurrentTurnMessage(resultMsg)
	}

	return ctx, results, false, nil
}

// handleToolApprovalRequest handles the tool approval workflow
// Returns: (updatedContext, shouldContinue, error)
// - updatedContext: context with user decision added (if blocking mode)
// - shouldContinue: true if approval was received and we should re-check, false if we should skip this iteration
// - error: ErrInputRequired if async HITL mode (task paused), other errors for failures
func (a *Agent) handleToolApprovalRequest(
	ctx context.Context,
	approvalResult *ToolApprovalResult,
	outputCh chan<- *pb.Part,
	reasoningState *reasoning.ReasoningState,
) (context.Context, bool, error) {
	taskID := getTaskIDFromContext(ctx)
	if taskID == "" {
		// No taskID - can't request approval, deny the tool
		if sendErr := safeSendPart(ctx, outputCh, createTextPart("⚠️  Tool requires approval but task tracking not enabled, denying\n")); sendErr != nil {
			slog.Error("Failed to send approval denial message", "agent", a.id, "error", sendErr)
		}
		return ctx, false, nil
	}

	// Determine which mode to use
	if a.shouldUseAsyncHITL() {
		return a.handleAsyncHITL(ctx, approvalResult, outputCh, reasoningState)
	} else {
		return a.handleBlockingHITL(ctx, approvalResult, outputCh)
	}
}

// handleBlockingHITL handles tool approval in blocking mode (current behavior)
func (a *Agent) handleBlockingHITL(
	ctx context.Context,
	approvalResult *ToolApprovalResult,
	outputCh chan<- *pb.Part,
) (context.Context, bool, error) {
	taskID := getTaskIDFromContext(ctx)

	// Update task to INPUT_REQUIRED state with approval request message
	if err := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_INPUT_REQUIRED, approvalResult.InteractionMsg); err != nil {
		return ctx, false, fmt.Errorf("updating task status: %w", err)
	}

	// Send approval request message parts to stream so UI can display them
	if approvalResult.InteractionMsg != nil && len(approvalResult.InteractionMsg.Parts) > 0 {
		for _, part := range approvalResult.InteractionMsg.Parts {
			if sendErr := safeSendPart(ctx, outputCh, part); sendErr != nil {
				slog.Error("Failed to send approval request part", "agent", a.id, "error", sendErr)
				return ctx, false, sendErr
			}
		}
	}

	// Store tool name before waiting (will be nil after re-running filter)
	pendingToolName := approvalResult.PendingToolCall.Name

	// Wait for user approval inline (don't return - keep goroutine alive)
	timeout := a.getInputTimeout()
	userMessage, err := a.taskAwaiter.WaitForInput(ctx, taskID, timeout)
	if err != nil {
		// Update task state to failed on timeout/cancellation
		if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
			slog.Error("Failed to update task status after timeout", "agent", a.id, "task", taskID, "error", updateErr)
		}
		if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("❌ Approval timeout or cancelled: %v\n", err))); sendErr != nil {
			slog.Error("Failed to send timeout message", "agent", a.id, "error", sendErr)
		}
		return ctx, false, err
	}

	// Extract user decision and add to context
	decision := parseUserDecision(userMessage)
	ctx = context.WithValue(ctx, userDecisionContextKey, decision)

	// Update task back to WORKING state
	if err := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_WORKING, nil); err != nil {
		slog.Error("Failed to update task status to WORKING", "agent", a.id, "task", taskID, "error", err)
	}

	// Send confirmation message to indicate interaction was resolved
	if decision == DecisionApprove {
		confirmMsg := fmt.Sprintf("✅ Approved: %s", pendingToolName)
		if sendErr := safeSendPart(ctx, outputCh, createTextPart(confirmMsg)); sendErr != nil {
			slog.Error("Failed to send approval confirmation", "agent", a.id, "error", sendErr)
		}
	} else {
		confirmMsg := fmt.Sprintf("🚫 Denied: %s", pendingToolName)
		if sendErr := safeSendPart(ctx, outputCh, createTextPart(confirmMsg)); sendErr != nil {
			slog.Error("Failed to send denial confirmation", "agent", a.id, "error", sendErr)
		}
	}

	// Return updated context and true to indicate we should re-check approval with user decision
	return ctx, true, nil
}

// handleAsyncHITL handles tool approval in async mode (saves state and exits)
func (a *Agent) handleAsyncHITL(
	ctx context.Context,
	approvalResult *ToolApprovalResult,
	outputCh chan<- *pb.Part,
	reasoningState *reasoning.ReasoningState,
) (context.Context, bool, error) {
	taskID := getTaskIDFromContext(ctx)
	sessionID := getSessionIDFromContext(ctx)
	if sessionID == "" {
		return ctx, false, fmt.Errorf("session ID required for async HITL")
	}

	// Use generic checkpoint function with HITL-specific phase
	err := a.checkpointExecution(
		ctx,
		taskID,
		PhaseToolApproval,   // HITL-specific phase
		CheckpointTypeEvent, // Event-driven
		reasoningState,
		approvalResult.PendingToolCall,
	)
	if err != nil {
		return ctx, false, fmt.Errorf("failed to checkpoint execution state: %w", err)
	}

	// Update task to INPUT_REQUIRED state with approval request message
	if err := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_INPUT_REQUIRED, approvalResult.InteractionMsg); err != nil {
		return ctx, false, fmt.Errorf("updating task status: %w", err)
	}

	// Send approval request message parts to stream
	if approvalResult.InteractionMsg != nil && len(approvalResult.InteractionMsg.Parts) > 0 {
		for _, part := range approvalResult.InteractionMsg.Parts {
			if sendErr := safeSendPart(ctx, outputCh, part); sendErr != nil {
				slog.Error("Failed to send approval request part", "agent", a.id, "error", sendErr)
				return ctx, false, sendErr
			}
		}
	}

	// Return ErrInputRequired to signal that execution should pause
	// The caller should exit the goroutine
	slog.Info("Task paused for async user input", "agent", a.id, "task", taskID)
	return ctx, false, ErrInputRequired
}

// createAssistantMessageWithToolCalls creates an assistant message with text and tool calls
func (a *Agent) createAssistantMessageWithToolCalls(
	text string,
	approvedCalls []*protocol.ToolCall,
	deniedCalls []*protocol.ToolCall,
) *pb.Message {
	assistantMsg := &pb.Message{
		Role:  pb.Role_ROLE_AGENT,
		Parts: []*pb.Part{},
	}

	if text != "" {
		assistantMsg.Parts = append(assistantMsg.Parts,
			&pb.Part{Part: &pb.Part_Text{Text: text}})
	}

	// Add approved tool calls
	for _, tc := range approvedCalls {
		assistantMsg.Parts = append(assistantMsg.Parts,
			protocol.CreateToolCallPart(tc))
	}

	// Add denied tool calls (so LLM knows what was attempted)
	for _, tc := range deniedCalls {
		assistantMsg.Parts = append(assistantMsg.Parts,
			protocol.CreateToolCallPart(tc))
	}

	return assistantMsg
}

// callLLMWithRetry calls the LLM with automatic retry logic for rate limits
func (a *Agent) callLLMWithRetry(
	ctx context.Context,
	messages []*pb.Message,
	toolDefs []llms.ToolDefinition,
	outputCh chan<- *pb.Part,
	cfg config.ReasoningConfig,
	span trace.Span,
) (string, []*protocol.ToolCall, int, error) {
	var text string
	var toolCalls []*protocol.ToolCall
	var tokens int
	var err error

	for attempt := 0; attempt <= maxLLMRetries; attempt++ {
		// Check context cancellation before each retry
		select {
		case <-ctx.Done():
			return "", nil, 0, ctx.Err()
		default:
		}

		text, toolCalls, tokens, err = a.callLLM(ctx, messages, toolDefs, outputCh, cfg)

		if err == nil {
			return text, toolCalls, tokens, nil
		}

		var retryErr *httpclient.RetryableError
		if !errors.As(err, &retryErr) {
			// Non-retryable error - return immediately
			return "", nil, 0, err
		}

		if attempt >= maxLLMRetries {
			// Exceeded max retries
			return "", nil, 0, fmt.Errorf("rate limit exceeded after %d retries: %w", maxLLMRetries, err)
		}

		waitTime := retryErr.RetryAfter
		if waitTime == 0 {
			waitTime = defaultRetryWaitSeconds * time.Second
		}

		if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("⏳ Rate limit exceeded (HTTP %d). Waiting %v before retry %d/%d...\n",
			retryErr.StatusCode, waitTime.Round(time.Second), attempt+1, maxLLMRetries))); sendErr != nil {
			slog.Error("Failed to send rate limit message", "agent", a.name, "error", sendErr)
		}

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return "", nil, 0, ctx.Err()
		case <-time.After(waitTime):
		}

		if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("🔄 Retrying LLM call (attempt %d/%d)...\n", attempt+1, maxLLMRetries))); sendErr != nil {
			slog.Error("Failed to send retry message", "agent", a.name, "error", sendErr)
		}
	}

	return "", nil, 0, fmt.Errorf("unexpected retry loop exit")
}

func (a *Agent) callLLM(
	ctx context.Context,
	messages []*pb.Message,
	toolDefs []llms.ToolDefinition,
	outputCh chan<- *pb.Part,
	cfg config.ReasoningConfig,
) (string, []*protocol.ToolCall, int, error) {
	llm := a.services.LLM()

	// Check for structured output configuration
	if a.config != nil && a.config.StructuredOutput != nil {
		structConfig := &llms.StructuredOutputConfig{
			Format:           a.config.StructuredOutput.Format,
			Schema:           a.config.StructuredOutput.Schema,
			Enum:             a.config.StructuredOutput.Enum,
			Prefill:          a.config.StructuredOutput.Prefill,
			PropertyOrdering: a.config.StructuredOutput.PropertyOrdering,
		}

		text, toolCalls, tokens, err := llm.GenerateStructured(ctx, messages, toolDefs, structConfig)
		if err != nil {
			return "", nil, 0, err
		}

		if text != "" {
			if sendErr := safeSendPart(ctx, outputCh, createTextPart(text)); sendErr != nil {
				slog.Error("Failed to send structured output text", "agent", a.name, "error", sendErr)
			}
		}

		return text, toolCalls, tokens, nil
	}

	if cfg.EnableStreaming != nil && *cfg.EnableStreaming {

		var streamedText strings.Builder
		var streamedThinking strings.Builder
		llmService := a.services.LLM()
		showThinking := config.BoolValue(cfg.ShowThinking, false)

		// Use GenerateStreamingChunks to get raw StreamChunks with proper abstraction
		// ctx already has ShowThinking flag from execute() function
		streamCh, err := llmService.GenerateStreamingChunks(ctx, messages, toolDefs)
		if err != nil {
			return "", nil, 0, err
		}

		var accumulatedToolCalls []*protocol.ToolCall
		var tokens int
		var currentThinkingBlockID string

		for chunk := range streamCh {
			switch chunk.Type {
			case "text":
				streamedText.WriteString(chunk.Text)
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(chunk.Text)); sendErr != nil {
					slog.Error("Failed to send streaming text chunk", "agent", a.name, "error", sendErr)
					return "", nil, 0, sendErr
				}
			case "thinking":
				// Only create thinking parts if ShowThinking is enabled
				if showThinking {
					// Create thinking part with AG-UI metadata
					if currentThinkingBlockID == "" {
						currentThinkingBlockID = fmt.Sprintf("think-%d", time.Now().UnixNano())
						// Start of thinking block - create thinking part
						thinkingPart := protocol.CreateThinkingPart("", currentThinkingBlockID, 0)
						if sendErr := safeSendPart(ctx, outputCh, thinkingPart); sendErr != nil {
							slog.Error("Failed to send thinking start", "agent", a.name, "error", sendErr)
							return "", nil, 0, sendErr
						}
					}
					streamedThinking.WriteString(chunk.Text)
					// Emit thinking delta as text part with thinking metadata
					thinkingPart := protocol.CreateThinkingPart(chunk.Text, currentThinkingBlockID, 0)
					if sendErr := safeSendPart(ctx, outputCh, thinkingPart); sendErr != nil {
						slog.Error("Failed to send thinking chunk", "agent", a.name, "error", sendErr)
						return "", nil, 0, sendErr
					}
				} else {
					// Still accumulate thinking for internal tracking, but don't emit parts
					streamedThinking.WriteString(chunk.Text)
				}
			case "tool_call":
				if chunk.ToolCall != nil {
					accumulatedToolCalls = append(accumulatedToolCalls, chunk.ToolCall)
				}
			case "done":
				tokens = chunk.Tokens
				// End of thinking block if any
				if currentThinkingBlockID != "" {
					currentThinkingBlockID = ""
				}
			case "error":
				return "", nil, 0, chunk.Error
			}
		}

		return streamedText.String(), accumulatedToolCalls, tokens, nil
	}

	text, toolCalls, tokens, err := llm.Generate(ctx, messages, toolDefs)
	if err != nil {
		return "", nil, 0, err
	}

	if text != "" {
		if sendErr := safeSendPart(ctx, outputCh, createTextPart(text)); sendErr != nil {
			slog.Error("Failed to send LLM response text", "agent", a.name, "error", sendErr)
		}
	}

	return text, toolCalls, tokens, nil
}

func (a *Agent) executeTools(
	ctx context.Context,
	toolCalls []*protocol.ToolCall,
	outputCh chan<- *pb.Part,
	cfg config.ReasoningConfig,
) []reasoning.ToolResult {
	toolService := a.services.Tools()

	results := make([]reasoning.ToolResult, 0, len(toolCalls))

	// Add newline before tools if showing them
	if len(toolCalls) > 0 && config.BoolValue(cfg.ShowTools, true) {
		if sendErr := safeSendPart(ctx, outputCh, createTextPart("\n")); sendErr != nil {
			slog.Error("Failed to send newline", "agent", a.name, "error", sendErr)
		}
	}

	for _, toolCall := range toolCalls {

		select {
		case <-ctx.Done():
			return results
		default:
		}

		// EMIT TOOL CALL PART (for web UI animations & CLI can extract from this)
		if sendErr := safeSendPart(ctx, outputCh, protocol.CreateToolCallPart(toolCall)); sendErr != nil {
			slog.Error("Failed to send tool call part", "agent", a.name, "error", sendErr)
			return results
		}

		// Check if tool supports streaming
		tool, err := toolService.GetTool(toolCall.Name)
		if err != nil {
			slog.Error("Failed to get tool", "agent", a.name, "tool", toolCall.Name, "error", err)
			// Fall back to non-streaming execution
			result, metadata, execErr := toolService.ExecuteToolCall(ctx, toolCall)
			resultContent := result
			errorStr := ""
			if execErr != nil {
				if resultContent == "" {
					resultContent = fmt.Sprintf("Error: %v", execErr)
				}
				errorStr = execErr.Error()
			}
			a2aResult := &protocol.ToolResult{
				ToolCallID: toolCall.ID,
				Content:    resultContent,
				Error:      errorStr,
			}
			// Mark as final since this is a complete (non-streaming) result
			if sendErr := safeSendPart(ctx, outputCh, protocol.CreateToolResultPartWithFinal(a2aResult, true)); sendErr != nil {
				slog.Error("Failed to send tool result part", "agent", a.name, "error", sendErr)
				return results
			}
			results = append(results, reasoning.ToolResult{
				ToolCall:   toolCall,
				Content:    resultContent,
				Error:      execErr,
				ToolCallID: toolCall.ID,
				ToolName:   toolCall.Name,
				Metadata:   metadata,
			})
			continue
		}

		// Check if tool implements StreamingTool interface
		// Directly assert to StreamingTool to avoid double type checking
		if streamingTool, ok := tool.(tools.StreamingTool); ok {
			// Use streaming orchestrator to handle streaming execution
			// This enables incremental tool result streaming, similar to thinking blocks.
			// Each chunk from the tool triggers an immediate tool result part emission.
			const streamingChannelBufferSize = 10
			orchestrator := tools.NewStreamingOrchestrator(streamingChannelBufferSize)

			finalResult, execErr := orchestrator.Execute(ctx, streamingTool, toolCall.Args, func(content string) error {
				// Emit incremental tool result part (streams like thinking blocks)
				// Each chunk triggers a new tool result part with accumulated content
				// Mark as not final (is_final: false) so web UI keeps status as 'working'
				a2aResult := &protocol.ToolResult{
					ToolCallID: toolCall.ID,
					Content:    content, // Accumulated content up to this point
					Error:      "",
				}
				// Use CreateToolResultPartWithFinal to mark as incremental (not final)
				if sendErr := safeSendPart(ctx, outputCh, protocol.CreateToolResultPartWithFinal(a2aResult, false)); sendErr != nil {
					slog.Error("Failed to send streaming tool result chunk", "agent", a.name, "error", sendErr)
					return sendErr
				}
				return nil
			})

			// Emit final tool result part (in case there were no chunks or to ensure final state)
			// Mark as final (is_final: true) so web UI updates status to 'success'/'failed'
			if finalResult.Content != "" || finalResult.Error != "" {
				a2aResult := &protocol.ToolResult{
					ToolCallID: toolCall.ID,
					Content:    finalResult.Content,
					Error:      finalResult.Error,
				}
				// Use CreateToolResultPartWithFinal to mark as final
				if sendErr := safeSendPart(ctx, outputCh, protocol.CreateToolResultPartWithFinal(a2aResult, true)); sendErr != nil {
					slog.Error("Failed to send final tool result part", "agent", a.name, "error", sendErr)
					return results
				}
			}

			results = append(results, reasoning.ToolResult{
				ToolCall:   toolCall,
				Content:    finalResult.Content,
				Error:      execErr,
				ToolCallID: toolCall.ID,
				ToolName:   toolCall.Name,
				Metadata:   finalResult.Metadata,
			})
			continue
		}

		// Non-streaming tool execution (existing path)
		result, metadata, err := toolService.ExecuteToolCall(ctx, toolCall)
		resultContent := result
		errorStr := ""
		if err != nil {
			// If ExecuteToolCall returned an error, it also returned error content in result
			// Use that content (which includes detailed error messages) instead of overwriting
			if resultContent == "" {
				// Fallback if no content was returned
				resultContent = fmt.Sprintf("Error: %v", err)
			}
			errorStr = err.Error()
		}

		// EMIT TOOL RESULT PART (for web UI animations & CLI can extract from this)
		// Non-streaming tools always emit final results
		a2aResult := &protocol.ToolResult{
			ToolCallID: toolCall.ID,
			Content:    resultContent,
			Error:      errorStr,
		}
		// Mark as final since this is a complete (non-streaming) result
		if sendErr := safeSendPart(ctx, outputCh, protocol.CreateToolResultPartWithFinal(a2aResult, true)); sendErr != nil {
			slog.Error("Failed to send tool result part", "agent", a.name, "error", sendErr)
			return results
		}

		results = append(results, reasoning.ToolResult{
			ToolCall:   toolCall,
			Content:    resultContent,
			Error:      err,
			ToolCallID: toolCall.ID,
			ToolName:   toolCall.Name,
			Metadata:   metadata,
		})

		// Special handling: if todo_write was called, emit display part immediately
		if toolCall.Name == "todo_write" && config.BoolValue(cfg.ShowThinking, false) {
			a.emitTodoDisplay(ctx, outputCh)
		}
	}

	return results
}

// emitTodoDisplay retrieves current todos and emits a display part
func (a *Agent) emitTodoDisplay(ctx context.Context, outputCh chan<- *pb.Part) {
	// Get todo tool from registry
	tool, err := a.services.Tools().GetTool("todo_write")
	if err != nil {
		return
	}

	todoTool, ok := tool.(*tools.TodoTool)
	if !ok {
		return
	}

	// Extract session ID from context
	sessionID := getSessionIDFromContext(ctx)
	if sessionID == "" {
		sessionID = "default"
	}

	// Get todos for this session
	todos := todoTool.GetTodos(sessionID)
	if len(todos) == 0 {
		return
	}

	// Build text fallback
	var textBuilder strings.Builder
	textBuilder.WriteString("📋 Current Tasks:\n")
	for i, todo := range todos {
		var checkbox string
		switch todo.Status {
		case "completed":
			checkbox = "☑"
		case "in_progress":
			checkbox = "⧗"
		case "pending":
			checkbox = "☐"
		case "canceled":
			checkbox = "☒"
		default:
			checkbox = "☐"
		}
		textBuilder.WriteString(fmt.Sprintf("  %s %d. %s\n", checkbox, i+1, todo.Content))
	}

	// Build structured data for rich clients
	todoData := make([]map[string]interface{}, len(todos))
	for i, todo := range todos {
		todoData[i] = map[string]interface{}{
			"id":      todo.ID,
			"content": todo.Content,
			"status":  todo.Status,
		}
	}

	data := map[string]interface{}{
		"todos": todoData,
	}

	// Emit as AG-UI thinking part
	part := protocol.CreateThinkingPartWithData(textBuilder.String(), "todo", data)
	if sendErr := safeSendPart(ctx, outputCh, part); sendErr != nil {
		slog.Error("Failed to send todo display", "agent", a.name, "error", sendErr)
	}
}
func (a *Agent) truncateToolResults(results []reasoning.ToolResult, cfg config.ReasoningConfig) []reasoning.ToolResult {

	const maxToolResultSize = 50000

	truncated := make([]reasoning.ToolResult, len(results))
	copy(truncated, results)

	for i := range truncated {
		contentSize := len(truncated[i].Content)
		if contentSize > maxToolResultSize {

			originalSize := contentSize
			truncated[i].Content = truncated[i].Content[:maxToolResultSize]

			suffix := fmt.Sprintf("\n\n[Warning: Output truncated: showing %d of %d bytes. Use more specific queries or filters to see full content.]",
				maxToolResultSize, originalSize)
			truncated[i].Content += suffix
		}
	}

	return truncated
}

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

	sessionID := getSessionIDFromContext(ctx)

	if !state.IsFinalResponseAdded() {
		finalResponse := state.GetAssistantResponse()
		if finalResponse != "" {

			assistantMsg := protocol.CreateTextMessage(pb.Role_ROLE_AGENT, finalResponse)
			state.AddCurrentTurnMessage(assistantMsg)
			state.MarkFinalResponseAdded()
		}
	}

	currentTurn := state.GetCurrentTurn()

	messagesToSave := make([]*pb.Message, 0, len(currentTurn))
	for _, msg := range currentTurn {

		textContent := protocol.ExtractTextFromMessage(msg)
		hasToolCalls := len(protocol.GetToolCallsFromMessage(msg)) > 0
		hasToolResults := len(protocol.GetToolResultsFromMessage(msg)) > 0

		if textContent != "" || hasToolCalls || hasToolResults {
			messagesToSave = append(messagesToSave, msg)
		}

	}

	if len(messagesToSave) > 0 {
		err := history.AddBatchToHistory(sessionID, messagesToSave)
		if err != nil {
			slog.Warn("Failed to save messages to history", "error", err)
		}
	}
}

func (a *Agent) buildPromptSlots(strategy reasoning.ReasoningStrategy) reasoning.PromptSlots {
	strategySlots := strategy.GetPromptSlots()

	// Use config prompt slots if available
	if a.config != nil {
		if a.config.Prompt.SystemPrompt != "" {
			return reasoning.PromptSlots{}
		}

		// Use typed config slots if provided
		if a.config.Prompt.PromptSlots != nil {
			userSlots := reasoning.PromptSlots{
				SystemRole:   a.config.Prompt.PromptSlots.SystemRole,
				Instructions: a.config.Prompt.PromptSlots.Instructions,
				UserGuidance: a.config.Prompt.PromptSlots.UserGuidance,
			}
			strategySlots = strategySlots.Merge(userSlots)
		}
	}

	return strategySlots
}

func (a *Agent) getMaxIterations(cfg config.ReasoningConfig) int {
	if cfg.MaxIterations > 0 {
		return cfg.MaxIterations
	}
	return 5
}

func (a *Agent) GetID() string {
	return a.id
}

func (a *Agent) GetName() string {
	return a.name
}

func (a *Agent) GetDescription() string {
	return a.description
}

func (a *Agent) GetConfig() *config.AgentConfig {
	return a.config
}

func (a *Agent) GetServices() reasoning.AgentServices {
	return a.services
}

// Shutdown gracefully shuts down the agent, cancelling all active tasks
// and waiting for them to complete (with timeout)
func (a *Agent) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down", "agent", a.id)

	// Cancel all active executions
	a.executionsMu.Lock()
	activeCount := len(a.activeExecutions)
	cancelFuncs := make([]context.CancelFunc, 0, activeCount)
	for taskID, cancel := range a.activeExecutions {
		cancelFuncs = append(cancelFuncs, cancel)
		slog.Info("Cancelling task", "agent", a.id, "task", taskID)
	}
	a.executionsMu.Unlock()

	// Cancel all contexts
	for _, cancel := range cancelFuncs {
		cancel()
	}

	// Cancel all waiting tasks
	waitingTasks := a.taskAwaiter.GetWaitingTasks()
	for _, taskID := range waitingTasks {
		a.taskAwaiter.CancelWaiting(taskID)
		slog.Info("Cancelled waiting task", "agent", a.id, "task", taskID)
	}

	// Wait for active executions to finish (with timeout)
	if activeCount > 0 {
		slog.Info("Waiting for active tasks to complete", "agent", a.id, "count", activeCount)
		// Give tasks up to 30 seconds to complete
		waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Poll until all tasks are done or timeout
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-waitCtx.Done():
				// Timeout or parent context cancelled
				a.executionsMu.RLock()
				remaining := len(a.activeExecutions)
				a.executionsMu.RUnlock()
				if remaining > 0 {
					slog.Warn("Tasks still active after shutdown timeout", "agent", a.id, "remaining", remaining)
				}
				return waitCtx.Err()
			case <-ticker.C:
				a.executionsMu.RLock()
				remaining := len(a.activeExecutions)
				a.executionsMu.RUnlock()
				if remaining == 0 {
					slog.Info("All tasks completed", "agent", a.id)
					return nil
				}
			}
		}
	}

	slog.Info("Shutdown complete", "agent", a.id)
	return nil
}
