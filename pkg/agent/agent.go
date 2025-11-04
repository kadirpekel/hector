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
	"github.com/kadirpekel/hector/pkg/tools"
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

		state, err := reasoning.Builder().
			WithQuery(input).
			WithAgentName(a.name).
			WithSubAgents(a.config.SubAgents).
			WithOutputChannel(outputCh).
			WithShowThinking(config.BoolValue(cfg.ShowThinking, true)).
			WithServices(a.services).
			WithContext(spanCtx).
			Build()

		if err != nil {
			span.RecordError(err)
			recordAgentMetrics(spanCtx, time.Since(startTime), 0, err)
			outputCh <- createTextPart(fmt.Sprintf("Error: Failed to initialize state: %v\n", err))
			return
		}

		defer func() {
			a.saveToHistory(spanCtx, input, state, strategy, startTime)

			duration := time.Since(startTime)
			recordAgentMetrics(spanCtx, duration, state.TotalTokens(), nil)
		}()

		historyService := a.services.History()
		if historyService != nil {

			sessionID := ""
			if sessionIDValue := ctx.Value(SessionIDKey); sessionIDValue != nil {
				if sid, ok := sessionIDValue.(string); ok {
					sessionID = sid
				}
			}

			if notifiable, ok := historyService.(reasoning.StatusNotifiable); ok {
				notifiable.SetStatusNotifier(func(message string) {
					if message != "" {

						outputCh <- createTextPart("\n" + message + "\n")
					}
				})
			}

			recentHistory, err := historyService.GetRecentHistory(sessionID)
			if err != nil {
				outputCh <- createTextPart(fmt.Sprintf("Warning: Failed to load conversation history: %v\n", err))
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
				outputCh <- createTextPart(fmt.Sprintf("\nCanceled: %v\n", ctx.Err()))
				return
			default:
			}

			if err := strategy.PrepareIteration(currentIteration, state); err != nil {
				outputCh <- createTextPart(fmt.Sprintf("Error preparing iteration: %v\n", err))
				return
			}

			promptSlots := a.buildPromptSlots(strategy)

			additionalContext := strategy.GetContextInjection(state)

			messages, err := a.services.Prompt().BuildMessages(ctx, input, promptSlots, state.AllMessages(), additionalContext)
			if err != nil {
				outputCh <- createTextPart(fmt.Sprintf("Error building messages: %v\n", err))
				return
			}

			var text string
			var toolCalls []*protocol.ToolCall
			var tokens int

			for attempt := 0; attempt <= maxLLMRetries; attempt++ {
				text, toolCalls, tokens, err = a.callLLM(ctx, messages, toolDefs, outputCh, cfg)

				if err == nil {

					break
				}

				var retryErr *httpclient.RetryableError
				if !errors.As(err, &retryErr) {
					span.RecordError(err)
					outputCh <- createTextPart(fmt.Sprintf("Error: Fatal error: %v\n", err))
					return
				}

				if attempt >= maxLLMRetries {
					span.RecordError(err)
					outputCh <- createTextPart(fmt.Sprintf("Error: Rate limit exceeded after %d retries: %v\n", maxLLMRetries, err))
					return
				}

				waitTime := retryErr.RetryAfter
				if waitTime == 0 {
					waitTime = defaultRetryWaitSeconds * time.Second
				}

				outputCh <- createTextPart(fmt.Sprintf("⏳ Rate limit exceeded (HTTP %d). Waiting %v before retry %d/%d...\n",
					retryErr.StatusCode, waitTime.Round(time.Second), attempt+1, maxLLMRetries))

				time.Sleep(waitTime)

				outputCh <- createTextPart(fmt.Sprintf("🔄 Retrying LLM call (attempt %d/%d)...\n", attempt+1, maxLLMRetries))
			}

			state.AddTokens(tokens)

			if text != "" {
				state.AppendResponse(text)
			}

			state.RecordFirstToolCalls(toolCalls)

			if text == "" && len(toolCalls) == 0 {
				text = "[Internal: Agent returned empty response]"
				state.AppendResponse(text)
			}

			var results []reasoning.ToolResult
			if len(toolCalls) > 0 {

				assistantMsg := &pb.Message{
					Role:  pb.Role_ROLE_AGENT,
					Parts: []*pb.Part{},
				}

				if text != "" {
					assistantMsg.Parts = append(assistantMsg.Parts,
						&pb.Part{Part: &pb.Part_Text{Text: text}})
				}

				for _, tc := range toolCalls {
					assistantMsg.Parts = append(assistantMsg.Parts,
						protocol.CreateToolCallPart(tc))
				}

				state.AddCurrentTurnMessage(assistantMsg)

				results = a.executeTools(ctx, toolCalls, outputCh, cfg)

				results = a.truncateToolResults(results, cfg)

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
			}

			if err := strategy.AfterIteration(currentIteration, text, toolCalls, results, state); err != nil {
				outputCh <- createTextPart(fmt.Sprintf("Error in strategy processing: %v\n", err))
				return
			}

			if strategy.ShouldStop(text, toolCalls, state) {
				break
			}
		}
	}()

	return outputCh, nil
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

		text, toolCalls, tokens, err := llm.GenerateStructured(messages, toolDefs, structConfig)
		if err != nil {
			return "", nil, 0, err
		}

		if text != "" {
			outputCh <- createTextPart(text)
		}

		return text, toolCalls, tokens, nil
	}

	if cfg.EnableStreaming != nil && *cfg.EnableStreaming {

		var streamedText strings.Builder
		wrappedCh := make(chan string, 100)
		done := make(chan struct{})

		go func() {
			defer close(done)
			for chunk := range wrappedCh {
				streamedText.WriteString(chunk)

				select {
				case outputCh <- createTextPart(chunk):
				case <-ctx.Done():
					return
				}
			}
		}()

		toolCalls, tokens, err := llm.GenerateStreaming(messages, toolDefs, wrappedCh)

		close(wrappedCh)
		<-done

		if err != nil {
			return "", nil, 0, err
		}

		return streamedText.String(), toolCalls, tokens, nil
	}

	text, toolCalls, tokens, err := llm.Generate(messages, toolDefs)
	if err != nil {
		return "", nil, 0, err
	}

	if text != "" {
		outputCh <- createTextPart(text)
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
		outputCh <- createTextPart("\n")
	}

	for _, toolCall := range toolCalls {

		select {
		case <-ctx.Done():
			return results
		default:
		}

		// EMIT TOOL CALL PART (for web UI animations & CLI can extract from this)
		outputCh <- protocol.CreateToolCallPart(toolCall)

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
		outputCh <- protocol.CreateToolResultPart(a2aResult)

		// Special handling: if todo_write was called, emit display part immediately
		if toolCall.Name == "todo_write" && config.BoolValue(cfg.ShowThinking, true) {
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
	sessionID := "default"
	if sid, ok := ctx.Value(SessionIDKey).(string); ok {
		sessionID = sid
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
	outputCh <- part
}

//nolint:unused // Reserved for future tool display feature
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

	sessionID := ""
	if sessionIDValue := ctx.Value(SessionIDKey); sessionIDValue != nil {
		if sid, ok := sessionIDValue.(string); ok {
			sessionID = sid
		}
	}

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
