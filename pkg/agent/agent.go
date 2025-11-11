package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	baseURL            string // Server base URL for agent card URL construction
	preferredTransport string // Preferred A2A transport (grpc, json-rpc, rest)

	// Human-in-the-loop support (A2A Protocol Section 6.3 - INPUT_REQUIRED state)
	taskAwaiter      *TaskAwaiter                  // Handles paused tasks waiting for input
	activeExecutions map[string]context.CancelFunc // Track active task executions for cancellation
	executionsMu     sync.RWMutex                  // Protects activeExecutions
}

func NewAgent(agentID string, agentConfig *config.AgentConfig, componentMgr interface{}, registry *AgentRegistry, baseURL string, preferredTransport string) (*Agent, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}

	compMgr, ok := componentMgr.(*component.ComponentManager)
	if !ok {
		return nil, fmt.Errorf("invalid component manager type")
	}

	services, err := NewAgentServicesWithRegistry(agentID, agentConfig, compMgr, registry)
	if err != nil {
		return nil, err
	}

	var taskWorkers chan struct{}
	if services.Task() != nil && agentConfig.Task != nil && agentConfig.Task.WorkerPool > 0 {

		taskWorkers = make(chan struct{}, agentConfig.Task.WorkerPool)
	}

	// Determine preferred transport: agent-level override > global > default
	transport := preferredTransport
	if agentConfig.A2A != nil && agentConfig.A2A.PreferredTransport != "" {
		transport = agentConfig.A2A.PreferredTransport
	}
	if transport == "" {
		transport = "json-rpc" // Default
	}

	// Initialize task awaiter with default timeout (can be overridden in config)
	awaitTimeout := 10 * time.Minute
	if agentConfig.Task != nil && agentConfig.Task.InputTimeout > 0 {
		awaitTimeout = time.Duration(agentConfig.Task.InputTimeout) * time.Second
	}

	return &Agent{
		id:                 agentID,
		name:               agentConfig.Name,
		description:        agentConfig.Description,
		config:             agentConfig,
		componentManager:   compMgr,
		services:           services,
		taskWorkers:        taskWorkers,
		baseURL:            baseURL,
		preferredTransport: transport,
		taskAwaiter:        NewTaskAwaiter(awaitTimeout),
		activeExecutions:   make(map[string]context.CancelFunc),
	}, nil
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
	if a.config.A2A != nil && a.config.A2A.Version != "" {
		return a.config.A2A.Version
	}
	return "1.0.0"
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
	if a.config.A2A != nil {
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

		spanCtx, span := startAgentSpan(ctx, a.name, a.config.LLM, input)
		defer span.End()

		// Add ShowThinking flag to context for LLM providers
		showThinking := config.BoolValue(cfg.ShowThinking, false)
		spanCtx = context.WithValue(spanCtx, protocol.ShowThinkingKey, showThinking)

		state, err := reasoning.Builder().
			WithQuery(input).
			WithAgentName(a.name).
			WithSubAgents(a.config.SubAgents).
			WithOutputChannel(outputCh).
			WithShowThinking(showThinking).
			WithServices(a.services).
			WithContext(spanCtx).
			Build()

		if err != nil {
			span.RecordError(err)
			recordAgentMetrics(spanCtx, time.Since(startTime), 0, err)
			if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Error: Failed to initialize state: %v\n", err))); sendErr != nil {
				log.Printf("[Agent:%s] Failed to send error to output channel: %v", a.name, sendErr)
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
							log.Printf("[Agent:%s] Failed to send status notification: %v", a.name, sendErr)
						}
					}
				})
			}

			recentHistory, err := historyService.GetRecentHistory(sessionID)
			if err != nil {
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Warning: Failed to load conversation history: %v\n", err))); sendErr != nil {
					log.Printf("[Agent:%s] Failed to send warning: %v", a.name, sendErr)
				}
			} else if len(recentHistory) > 0 {
				state.SetHistory(recentHistory)
			}
		}

		userMsg := protocol.CreateUserMessage(input)
		state.AddCurrentTurnMessage(userMsg)

		maxIterations := a.getMaxIterations(cfg)

		tools := a.services.Tools()
		toolDefs := tools.GetAvailableTools()

		for state.Iteration() < maxIterations {

			currentIteration := state.NextIteration()

			select {
			case <-ctx.Done():
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("\nCanceled: %v\n", ctx.Err()))); sendErr != nil {
					log.Printf("[Agent:%s] Failed to send cancellation message: %v", a.name, sendErr)
				}
				return
			default:
			}

			if err := strategy.PrepareIteration(currentIteration, state); err != nil {
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Error preparing iteration: %v\n", err))); sendErr != nil {
					log.Printf("[Agent:%s] Failed to send error: %v", a.name, sendErr)
				}
				return
			}

			promptSlots := a.buildPromptSlots(strategy)

			additionalContext := strategy.GetContextInjection(state)

			messages, err := a.services.Prompt().BuildMessages(ctx, input, promptSlots, state.AllMessages(), additionalContext)
			if err != nil {
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Error building messages: %v\n", err))); sendErr != nil {
					log.Printf("[Agent:%s] Failed to send error: %v", a.name, sendErr)
				}
				return
			}

			// Call LLM with retry logic for rate limits
			text, toolCalls, tokens, err := a.callLLMWithRetry(spanCtx, messages, toolDefs, outputCh, cfg, span)
			if err != nil {
				span.RecordError(err)
				if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("Error: LLM call failed: %v\n", err))); sendErr != nil {
					log.Printf("[Agent:%s] Failed to send error: %v", a.name, sendErr)
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
						log.Printf("[Agent:%s] Failed to send error: %v", a.name, sendErr)
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
					log.Printf("[Agent:%s] Failed to send error: %v", a.name, sendErr)
				}
				return
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
	// Check if any tools require approval (A2A Protocol Section 6.3 - INPUT_REQUIRED state)
	approvalResult, err := a.filterToolCallsWithApproval(ctx, toolCalls, a.componentManager.GetGlobalConfig().Tools)
	if err != nil {
		return ctx, nil, false, fmt.Errorf("checking tool approval: %w", err)
	}

	// Handle tool approval request if needed
	if approvalResult.NeedsUserInput {
		var shouldContinue bool
		ctx, shouldContinue, err = a.handleToolApprovalRequest(ctx, approvalResult, outputCh)
		if err != nil {
			return ctx, nil, false, err
		}
		if shouldContinue {
			// Re-run approval filter with user's decision
			approvalResult, err = a.filterToolCallsWithApproval(ctx, toolCalls, a.componentManager.GetGlobalConfig().Tools)
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
			Error:      fmt.Errorf("User denied tool execution"),
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
		a2aResult := &protocol.ToolResult{
			ToolCallID: result.ToolCallID,
			Content:    result.Content,
			Error:      errorStr,
		}
		resultMsg := &pb.Message{
			Role: pb.Role_ROLE_AGENT,
			Parts: []*pb.Part{
				protocol.CreateToolResultPart(a2aResult),
			},
		}
		state.AddCurrentTurnMessage(resultMsg)
	}

	return ctx, results, false, nil
}

// handleToolApprovalRequest handles the tool approval workflow
// Returns: (updatedContext, shouldContinue, error)
// - updatedContext: context with user decision added
// - shouldContinue: true if approval was received and we should re-check, false if we should skip this iteration
func (a *Agent) handleToolApprovalRequest(
	ctx context.Context,
	approvalResult *ToolApprovalResult,
	outputCh chan<- *pb.Part,
) (context.Context, bool, error) {
	taskID := getTaskIDFromContext(ctx)
	if taskID == "" {
		// No taskID - can't request approval, deny the tool
		if sendErr := safeSendPart(ctx, outputCh, createTextPart("⚠️  Tool requires approval but task tracking not enabled, denying\n")); sendErr != nil {
			log.Printf("[Agent:%s] Failed to send approval denial message: %v", a.id, sendErr)
		}
		return ctx, false, nil
	}

	// Update task to INPUT_REQUIRED state with approval request message
	if err := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_INPUT_REQUIRED, approvalResult.InteractionMsg); err != nil {
		return ctx, false, fmt.Errorf("updating task status: %w", err)
	}

	// Send approval request message parts to stream so UI can display them
	if approvalResult.InteractionMsg != nil && len(approvalResult.InteractionMsg.Parts) > 0 {
		for _, part := range approvalResult.InteractionMsg.Parts {
			if sendErr := safeSendPart(ctx, outputCh, part); sendErr != nil {
				log.Printf("[Agent:%s] Failed to send approval request part: %v", a.id, sendErr)
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
			log.Printf("[Agent:%s] [HITL] Failed to update task %s status after timeout: %v", a.id, taskID, updateErr)
		}
		if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("❌ Approval timeout or cancelled: %v\n", err))); sendErr != nil {
			log.Printf("[Agent:%s] Failed to send timeout message: %v", a.id, sendErr)
		}
		return ctx, false, err
	}

	// Extract user decision and add to context
	decision := parseUserDecision(userMessage)
	ctx = context.WithValue(ctx, userDecisionContextKey, decision)

	// Update task back to WORKING state
	if err := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_WORKING, nil); err != nil {
		log.Printf("[Agent:%s] [HITL] Failed to update task %s status to WORKING: %v", a.id, taskID, err)
	}

	// Send confirmation message to indicate interaction was resolved
	if decision == DecisionApprove {
		confirmMsg := fmt.Sprintf("✅ Approved: %s", pendingToolName)
		if sendErr := safeSendPart(ctx, outputCh, createTextPart(confirmMsg)); sendErr != nil {
			log.Printf("[Agent:%s] Failed to send approval confirmation: %v", a.id, sendErr)
		}
	} else {
		confirmMsg := fmt.Sprintf("🚫 Denied: %s", pendingToolName)
		if sendErr := safeSendPart(ctx, outputCh, createTextPart(confirmMsg)); sendErr != nil {
			log.Printf("[Agent:%s] Failed to send denial confirmation: %v", a.id, sendErr)
		}
	}

	// Return updated context and true to indicate we should re-check approval with user decision
	return ctx, true, nil
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
			log.Printf("[Agent:%s] Failed to send rate limit message: %v", a.name, sendErr)
		}

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return "", nil, 0, ctx.Err()
		case <-time.After(waitTime):
		}

		if sendErr := safeSendPart(ctx, outputCh, createTextPart(fmt.Sprintf("🔄 Retrying LLM call (attempt %d/%d)...\n", attempt+1, maxLLMRetries))); sendErr != nil {
			log.Printf("[Agent:%s] Failed to send retry message: %v", a.name, sendErr)
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

	if a.config.StructuredOutput != nil {

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
				log.Printf("[Agent:%s] Failed to send structured output text: %v", a.name, sendErr)
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
					log.Printf("[Agent:%s] Failed to send streaming text chunk: %v", a.name, sendErr)
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
							log.Printf("[Agent:%s] Failed to send thinking start: %v", a.name, sendErr)
							return "", nil, 0, sendErr
						}
					}
					streamedThinking.WriteString(chunk.Text)
					// Emit thinking delta as text part with thinking metadata
					thinkingPart := protocol.CreateThinkingPart(chunk.Text, currentThinkingBlockID, 0)
					if sendErr := safeSendPart(ctx, outputCh, thinkingPart); sendErr != nil {
						log.Printf("[Agent:%s] Failed to send thinking chunk: %v", a.name, sendErr)
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
			log.Printf("[Agent:%s] Failed to send LLM response text: %v", a.name, sendErr)
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
	tools := a.services.Tools()

	results := make([]reasoning.ToolResult, 0, len(toolCalls))

	// Add newline before tools if showing them
	if len(toolCalls) > 0 && config.BoolValue(cfg.ShowTools, true) {
		if sendErr := safeSendPart(ctx, outputCh, createTextPart("\n")); sendErr != nil {
			log.Printf("[Agent:%s] Failed to send newline: %v", a.name, sendErr)
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
			log.Printf("[Agent:%s] Failed to send tool call part: %v", a.name, sendErr)
			return results
		}

		result, metadata, err := tools.ExecuteToolCall(ctx, toolCall)
		resultContent := result
		if err != nil {
			resultContent = fmt.Sprintf("Error: %v", err)
		}

		// EMIT TOOL RESULT PART (for web UI animations & CLI can extract from this)
		errorStr := ""
		if err != nil {
			errorStr = err.Error()
		}
		a2aResult := &protocol.ToolResult{
			ToolCallID: toolCall.ID,
			Content:    resultContent,
			Error:      errorStr,
		}
		if sendErr := safeSendPart(ctx, outputCh, protocol.CreateToolResultPart(a2aResult)); sendErr != nil {
			log.Printf("[Agent:%s] Failed to send tool result part: %v", a.name, sendErr)
			return results
		}

		// Special handling: if todo_write was called, emit display part immediately
		if toolCall.Name == "todo_write" && config.BoolValue(cfg.ShowThinking, false) {
			a.emitTodoDisplay(ctx, outputCh)
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
		log.Printf("[Agent:%s] Failed to send todo display: %v", a.name, sendErr)
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
			log.Printf("Warning: Failed to save messages to history: %v", err)
		}
	}
}

func (a *Agent) buildPromptSlots(strategy reasoning.ReasoningStrategy) reasoning.PromptSlots {

	if a.config.Prompt.SystemPrompt != "" {
		return reasoning.PromptSlots{}
	}

	strategySlots := strategy.GetPromptSlots()

	// Use typed config slots if provided
	if a.config.Prompt.PromptSlots != nil {
		userSlots := reasoning.PromptSlots{
			SystemRole:   a.config.Prompt.PromptSlots.SystemRole,
			Instructions: a.config.Prompt.PromptSlots.Instructions,
			UserGuidance: a.config.Prompt.PromptSlots.UserGuidance,
		}
		strategySlots = strategySlots.Merge(userSlots)
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
	log.Printf("[Agent:%s] Shutting down...", a.id)

	// Cancel all active executions
	a.executionsMu.Lock()
	activeCount := len(a.activeExecutions)
	cancelFuncs := make([]context.CancelFunc, 0, activeCount)
	for taskID, cancel := range a.activeExecutions {
		cancelFuncs = append(cancelFuncs, cancel)
		log.Printf("[Agent:%s] Cancelling task: %s", a.id, taskID)
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
		log.Printf("[Agent:%s] Cancelled waiting task: %s", a.id, taskID)
	}

	// Wait for active executions to finish (with timeout)
	if activeCount > 0 {
		log.Printf("[Agent:%s] Waiting for %d active tasks to complete...", a.id, activeCount)
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
					log.Printf("[Agent:%s] Warning: %d tasks still active after shutdown timeout", a.id, remaining)
				}
				return waitCtx.Err()
			case <-ticker.C:
				a.executionsMu.RLock()
				remaining := len(a.activeExecutions)
				a.executionsMu.RUnlock()
				if remaining == 0 {
					log.Printf("[Agent:%s] All tasks completed", a.id)
					return nil
				}
			}
		}
	}

	log.Printf("[Agent:%s] Shutdown complete", a.id)
	return nil
}
