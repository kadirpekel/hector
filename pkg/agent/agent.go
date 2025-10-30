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

const (
	outputChannelBuffer        = 100
	historyRetentionMultiplier = 10
	defaultRetryWaitSeconds    = 120
	maxLLMRetries              = 3
)

type Agent struct {
	pb.UnimplementedA2AServiceServer

	name        string
	description string
	config      *config.AgentConfig
	services    reasoning.AgentServices
	taskWorkers chan struct{}
}

func NewAgent(agentID string, agentConfig *config.AgentConfig, componentMgr interface{}, registry *AgentRegistry) (*Agent, error) {
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

	return &Agent{
		name:        agentConfig.Name,
		description: agentConfig.Description,
		config:      agentConfig,
		services:    services,
		taskWorkers: taskWorkers,
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
		Name:        a.name,
		Description: a.description,
		Version:     a.getVersion(),
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

	return []string{"text/plain", "application/json"}
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

		spanCtx, span := startAgentSpan(ctx, a.name, a.config.LLM, input)
		defer span.End()

		state, err := reasoning.Builder().
			WithQuery(input).
			WithAgentName(a.name).
			WithSubAgents(a.config.SubAgents).
			WithOutputChannel(outputCh).
			WithShowThinking(cfg.ShowThinking).
			WithShowDebugInfo(cfg.ShowDebugInfo).
			WithServices(a.services).
			WithContext(spanCtx).
			Build()

		if err != nil {
			span.RecordError(err)
			recordAgentMetrics(spanCtx, time.Since(startTime), 0, err)
			outputCh <- fmt.Sprintf("Error: Failed to initialize state: %v\n", err)
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

						outputCh <- "\n" + message + "\n"
					}
				})
			}

			recentHistory, err := historyService.GetRecentHistory(sessionID)
			if err != nil {
				outputCh <- fmt.Sprintf("Warning: Failed to load conversation history: %v\n", err)
			} else if len(recentHistory) > 0 {
				state.SetHistory(recentHistory)
			}
		}

		userMsg := protocol.CreateUserMessage(input)
		state.AddCurrentTurnMessage(userMsg)

		maxIterations := a.getMaxIterations(cfg)

		if cfg.ShowDebugInfo {
			outputCh <- fmt.Sprintf("\n[Strategy: %s]\n", strategy.GetName())
			outputCh <- fmt.Sprintf("Max iterations: %d\n\n", maxIterations)
		}

		tools := a.services.Tools()
		toolDefs := tools.GetAvailableTools()

		if cfg.ShowDebugInfo {
			outputCh <- fmt.Sprintf("Available tools: %d\n", len(toolDefs))
			for _, tool := range toolDefs {
				outputCh <- fmt.Sprintf("  - %s: %s\n", tool.Name, tool.Description)
			}
			outputCh <- "\n"
		}

		for state.Iteration() < maxIterations {

			currentIteration := state.NextIteration()

			select {
			case <-ctx.Done():
				outputCh <- fmt.Sprintf("\nCanceled: %v\n", ctx.Err())
				return
			default:
			}

			if cfg.ShowDebugInfo {
				outputCh <- fmt.Sprintf("🤔 **Iteration %d/%d**\n", currentIteration, maxIterations)
			}

			if err := strategy.PrepareIteration(currentIteration, state); err != nil {
				outputCh <- fmt.Sprintf("Error preparing iteration: %v\n", err)
				return
			}

			promptSlots := a.buildPromptSlots(strategy)

			additionalContext := strategy.GetContextInjection(state)

			messages, err := a.services.Prompt().BuildMessages(ctx, input, promptSlots, state.AllMessages(), additionalContext)
			if err != nil {
				outputCh <- fmt.Sprintf("Error building messages: %v\n", err)
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
					outputCh <- fmt.Sprintf("Error: Fatal error: %v\n", err)
					return
				}

				if attempt >= maxLLMRetries {
					span.RecordError(err)
					outputCh <- fmt.Sprintf("Error: Rate limit exceeded after %d retries: %v\n", maxLLMRetries, err)
					return
				}

				waitTime := retryErr.RetryAfter
				if waitTime == 0 {
					waitTime = defaultRetryWaitSeconds * time.Second
				}

				outputCh <- fmt.Sprintf("⏳ Rate limit exceeded (HTTP %d). Waiting %v before retry %d/%d...\n",
					retryErr.StatusCode, waitTime.Round(time.Second), attempt+1, maxLLMRetries)

				time.Sleep(waitTime)

				outputCh <- fmt.Sprintf("🔄 Retrying LLM call (attempt %d/%d)...\n", attempt+1, maxLLMRetries)
			}

			state.AddTokens(tokens)

			if cfg.ShowDebugInfo {
				outputCh <- fmt.Sprintf("\033[90m📝 Tokens used: %d (total: %d)\033[0m\n", tokens, state.TotalTokens())
			}

			if text != "" {
				state.AppendResponse(text)
			}

			state.RecordFirstToolCalls(toolCalls)

			if text == "" && len(toolCalls) == 0 {
				text = "[Internal: Agent returned empty response]"
				state.AppendResponse(text)
				if cfg.ShowDebugInfo {
					outputCh <- "\033[90mWarning: LLM returned empty response\033[0m\n"
				}

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
				outputCh <- fmt.Sprintf("Error in strategy processing: %v\n", err)
				return
			}

			if strategy.ShouldStop(text, toolCalls, state) {
				if cfg.ShowDebugInfo {
					outputCh <- "\033[90m\n\n[Reasoning complete]\033[0m\n"
				}
				break
			}
		}

		if cfg.ShowDebugInfo {
			outputCh <- fmt.Sprintf("\033[90m\nStats: Total time: %v | Tokens: %d | Iterations: %d\033[0m\n",
				time.Since(startTime), state.TotalTokens(), state.Iteration())
		}
	}()

	return outputCh, nil
}

func (a *Agent) callLLM(
	ctx context.Context,
	messages []*pb.Message,
	toolDefs []llms.ToolDefinition,
	outputCh chan<- string,
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
			outputCh <- text
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
				case outputCh <- chunk:
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
		outputCh <- text
	}

	return text, toolCalls, tokens, nil
}

func (a *Agent) executeTools(
	ctx context.Context,
	toolCalls []*protocol.ToolCall,
	outputCh chan<- string,
	cfg config.ReasoningConfig,
) []reasoning.ToolResult {
	tools := a.services.Tools()

	results := make([]reasoning.ToolResult, 0, len(toolCalls))

	// Add newline before tools if showing them
	if len(toolCalls) > 0 && cfg.ShowToolExecution != nil && *cfg.ShowToolExecution {
		if cfg.ToolDisplayMode != "hidden" {
			outputCh <- "\n"
		}
	}

	for _, toolCall := range toolCalls {

		select {
		case <-ctx.Done():
			return results
		default:
		}

		// Show tool execution based on display mode
		if cfg.ShowToolExecution != nil && *cfg.ShowToolExecution {
			displayToolCall(outputCh, toolCall, cfg)
		}

		result, metadata, err := tools.ExecuteToolCall(ctx, toolCall)
		resultContent := result
		if err != nil {
			resultContent = fmt.Sprintf("Error: %v", err)
		}

		// Show result based on configuration
		if cfg.ShowToolExecution != nil && *cfg.ShowToolExecution {
			displayToolResult(outputCh, toolCall, err, resultContent, cfg)
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

			if cfg.ShowDebugInfo {
				log.Printf("Warning: Tool result truncated: %s (%d → %d bytes)",
					truncated[i].ToolName, originalSize, maxToolResultSize)
			}
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

		if strategySlots.ReasoningInstructions != "" {
			strategySlots.ReasoningInstructions += "\n\n" + selfReflectionPrompt
		} else {
			strategySlots.ReasoningInstructions = selfReflectionPrompt
		}
	}

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

func (a *Agent) getMaxIterations(cfg config.ReasoningConfig) int {
	if cfg.MaxIterations > 0 {
		return cfg.MaxIterations
	}
	return 5
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
