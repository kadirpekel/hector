package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/config"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/protocol"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/tools"
)

type NoOpContextService struct{}

func NewNoOpContextService() reasoning.ContextService {
	return &NoOpContextService{}
}

func (s *NoOpContextService) SearchContext(ctx context.Context, query string) ([]databases.SearchResult, error) {

	return []databases.SearchResult{}, nil
}

func (s *NoOpContextService) ExtractSources(context []databases.SearchResult) []string {

	return []string{}
}

type ContextOptions struct {
	MaxResults int
	MinScore   float64
}

type DefaultContextService struct {
	searchEngine *hectorcontext.SearchEngine
}

func NewContextService(searchEngine *hectorcontext.SearchEngine) reasoning.ContextService {
	return &DefaultContextService{
		searchEngine: searchEngine,
	}
}

func (s *DefaultContextService) SearchContext(ctx context.Context, query string) ([]databases.SearchResult, error) {
	if s.searchEngine == nil {
		return []databases.SearchResult{}, nil
	}

	return s.searchEngine.Search(ctx, query, 5)
}

func (s *DefaultContextService) ExtractSources(context []databases.SearchResult) []string {
	sources := make([]string, 0, len(context))
	for _, result := range context {

		if result.Metadata != nil {
			if source, ok := result.Metadata["source"].(string); ok && source != "" {
				sources = append(sources, source)
			} else if result.ID != "" {
				sources = append(sources, result.ID)
			}
		} else if result.ID != "" {
			sources = append(sources, result.ID)
		}
	}
	return sources
}

type DefaultPromptService struct {
	promptConfig   config.PromptConfig
	contextService reasoning.ContextService
	historyService reasoning.HistoryService
}

func NewPromptService(
	promptConfig config.PromptConfig,
	contextService reasoning.ContextService,
	historyService reasoning.HistoryService,
) reasoning.PromptService {
	return &DefaultPromptService{
		promptConfig:   promptConfig,
		contextService: contextService,
		historyService: historyService,
	}
}

func (s *DefaultPromptService) BuildMessages(
	ctx context.Context,
	query string,
	slots reasoning.PromptSlots,
	currentToolConversation []*pb.Message,
	additionalContext string,
) ([]*pb.Message, error) {
	messages := make([]*pb.Message, 0)

	var systemPrompt string
	if !slots.IsEmpty() {
		systemPrompt = s.composeSystemPromptFromSlots(slots)
	} else if s.promptConfig.SystemPrompt != "" {

		systemPrompt = s.promptConfig.SystemPrompt
	}

	if systemPrompt != "" {
		messages = append(messages, protocol.CreateTextMessage(pb.Role_ROLE_UNSPECIFIED, systemPrompt))
	}

	if additionalContext != "" {
		messages = append(messages, protocol.CreateTextMessage(pb.Role_ROLE_UNSPECIFIED, additionalContext))
	}

	if s.promptConfig.IncludeContext && s.contextService != nil {
		contextResults, err := s.contextService.SearchContext(ctx, query)
		if err == nil && len(contextResults) > 0 {
			var contextText strings.Builder
			contextText.WriteString("Relevant context from documents:\n")
			for i, doc := range contextResults {
				if i >= 5 {
					break
				}
				content := doc.Content
				if len(content) > 500 {
					content = content[:500] + "..."
				}
				contextText.WriteString(fmt.Sprintf("- %s\n", content))
			}
			messages = append(messages, protocol.CreateTextMessage(pb.Role_ROLE_UNSPECIFIED, contextText.String()))
		}
	}

	messages = append(messages, currentToolConversation...)

	needsUserQuery := true
	for i := len(currentToolConversation) - 1; i >= 0; i-- {
		msg := currentToolConversation[i]
		if msg.Role == pb.Role_ROLE_USER {
			msgText := protocol.ExtractTextFromMessage(msg)
			if msgText == query {
				needsUserQuery = false
				break
			}
		}
	}

	if needsUserQuery {
		messages = append(messages, protocol.CreateUserMessage(query))
	}

	return messages, nil
}

func (s *DefaultPromptService) composeSystemPromptFromSlots(slots reasoning.PromptSlots) string {
	var prompt strings.Builder

	if slots.SystemRole != "" {
		prompt.WriteString(slots.SystemRole)
		prompt.WriteString("\n\n")
	}

	if slots.ReasoningInstructions != "" {
		prompt.WriteString(slots.ReasoningInstructions)
		prompt.WriteString("\n\n")
	}

	if slots.ToolUsage != "" {
		prompt.WriteString("<tool_usage>\n")
		prompt.WriteString(slots.ToolUsage)
		prompt.WriteString("\n</tool_usage>\n\n")
	}

	if slots.OutputFormat != "" {
		prompt.WriteString("<output_format>\n")
		prompt.WriteString(slots.OutputFormat)
		prompt.WriteString("\n</output_format>\n\n")
	}

	if slots.CommunicationStyle != "" {
		prompt.WriteString("<communication>\n")
		prompt.WriteString(slots.CommunicationStyle)
		prompt.WriteString("\n</communication>\n\n")
	}

	if slots.Additional != "" {
		prompt.WriteString(slots.Additional)
	}

	return strings.TrimSpace(prompt.String())
}

type DefaultLLMService struct {
	llmProvider llms.LLMProvider
}

func NewLLMService(llmProvider llms.LLMProvider) reasoning.LLMService {
	return &DefaultLLMService{
		llmProvider: llmProvider,
	}
}

func (s *DefaultLLMService) Generate(messages []*pb.Message, tools []llms.ToolDefinition) (string, []*protocol.ToolCall, int, error) {
	return s.llmProvider.Generate(messages, tools)
}

func (s *DefaultLLMService) GenerateStreaming(messages []*pb.Message, tools []llms.ToolDefinition, outputCh chan<- string) ([]*protocol.ToolCall, int, error) {
	streamCh, err := s.llmProvider.GenerateStreaming(messages, tools)
	if err != nil {
		return nil, 0, err
	}

	var toolCalls []*protocol.ToolCall
	var tokens int

	for chunk := range streamCh {
		switch chunk.Type {
		case "text":
			outputCh <- chunk.Text
		case "tool_call":
			if chunk.ToolCall != nil {

				toolCalls = append(toolCalls, chunk.ToolCall)
			}
		case "done":
			tokens = chunk.Tokens
		case "error":
			return toolCalls, tokens, chunk.Error
		}
	}

	return toolCalls, tokens, nil
}

func (s *DefaultLLMService) GenerateStructured(messages []*pb.Message, tools []llms.ToolDefinition, config *llms.StructuredOutputConfig) (string, []*protocol.ToolCall, int, error) {

	structProvider, ok := s.llmProvider.(llms.StructuredOutputProvider)
	if !ok {

		return "", nil, 0, fmt.Errorf("provider does not support structured output")
	}

	if !structProvider.SupportsStructuredOutput() {
		return "", nil, 0, fmt.Errorf("provider does not support structured output")
	}

	return structProvider.GenerateStructured(messages, tools, config)
}

func (s *DefaultLLMService) SupportsStructuredOutput() bool {

	structProvider, ok := s.llmProvider.(llms.StructuredOutputProvider)
	if !ok {
		return false
	}

	return structProvider.SupportsStructuredOutput()
}

type DefaultToolService struct {
	toolRegistry *tools.ToolRegistry
	allowedTools []string
}

func NewToolService(toolRegistry *tools.ToolRegistry, allowedTools []string) reasoning.ToolService {
	return &DefaultToolService{
		toolRegistry: toolRegistry,
		allowedTools: allowedTools,
	}
}

func (s *DefaultToolService) ExecuteToolCall(ctx context.Context, toolCall *protocol.ToolCall) (string, map[string]interface{}, error) {
	if s.toolRegistry == nil {
		return "", nil, fmt.Errorf("tool registry not available")
	}

	result, err := s.toolRegistry.ExecuteTool(ctx, toolCall.Name, toolCall.Args)
	if err != nil {
		return "", nil, fmt.Errorf("tool execution failed: %w", err)
	}

	if !result.Success {
		return "", result.Metadata, fmt.Errorf("tool failed: %s", result.Error)
	}

	return result.Content, result.Metadata, nil
}

func (s *DefaultToolService) GetAvailableTools() []llms.ToolDefinition {
	if s.toolRegistry == nil {
		return []llms.ToolDefinition{}
	}

	allToolInfos := s.toolRegistry.ListTools()
	result := make([]llms.ToolDefinition, 0, len(allToolInfos))

	if s.allowedTools == nil {
		for _, toolInfo := range allToolInfos {
			result = append(result, convertToolInfoToToolDefinition(toolInfo))
		}
		return result
	}

	if len(s.allowedTools) == 0 {
		return []llms.ToolDefinition{}
	}

	allowedSet := make(map[string]bool)
	for _, toolName := range s.allowedTools {
		allowedSet[toolName] = true
	}

	for _, toolInfo := range allToolInfos {
		if allowedSet[toolInfo.Name] {
			result = append(result, convertToolInfoToToolDefinition(toolInfo))
		}
	}

	return result
}

func (s *DefaultToolService) GetTool(name string) (interface{}, error) {
	if s.toolRegistry == nil {
		return nil, fmt.Errorf("tool registry not available")
	}

	return s.toolRegistry.GetTool(name)
}

func convertToolInfoToToolDefinition(info tools.ToolInfo) llms.ToolDefinition {

	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
		"required":   []string{},
	}

	properties := schema["properties"].(map[string]interface{})
	required := []string{}

	for _, param := range info.Parameters {
		propSchema := map[string]interface{}{
			"type":        param.Type,
			"description": param.Description,
		}

		if param.Type == "array" && param.Items != nil {
			propSchema["items"] = param.Items
		}

		properties[param.Name] = propSchema

		if param.Required {
			required = append(required, param.Name)
		}

		if len(param.Enum) > 0 {
			propSchema["enum"] = param.Enum
		}

		if param.Default != nil {
			propSchema["default"] = param.Default
		}
	}

	schema["required"] = required

	return llms.ToolDefinition{
		Name:        info.Name,
		Description: info.Description,
		Parameters:  schema,
	}
}
