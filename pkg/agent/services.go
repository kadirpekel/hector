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

// GetAssignedStores returns nil (no stores) for NoOpContextService
func (s *NoOpContextService) GetAssignedStores() []string {
	return []string{} // Empty slice means no stores (explicitly empty)
}

type ContextOptions struct {
	MaxResults int
	MinScore   float64
}

type DefaultContextService struct {
	searchEngine   *hectorcontext.SearchEngine
	assignedStores []string // Document stores assigned to this agent (nil/empty = all stores)
}

func NewContextService(searchEngine *hectorcontext.SearchEngine) reasoning.ContextService {
	return &DefaultContextService{
		searchEngine:   searchEngine,
		assignedStores: nil, // nil means all stores (backward compatible)
	}
}

func NewContextServiceWithStores(searchEngine *hectorcontext.SearchEngine, assignedStores []string) reasoning.ContextService {
	return &DefaultContextService{
		searchEngine:   searchEngine,
		assignedStores: assignedStores,
	}
}

func (s *DefaultContextService) SearchContext(ctx context.Context, query string) ([]databases.SearchResult, error) {
	// Use shared parallel search function, scoped to agent's assigned stores
	// Limit determined by search config TopK (default: 5)
	limit := 5 // Default limit
	if s.searchEngine != nil {
		status := s.searchEngine.GetStatus()
		if cfg, ok := status["config"].(config.SearchConfig); ok && cfg.TopK > 0 {
			limit = cfg.TopK
		}
	}
	// Pass assigned stores to scope the search (nil/empty = all stores)
	return hectorcontext.SearchAllStores(ctx, query, limit, s.assignedStores)
}

// GetSearchConfig returns the search config from the search engine
func (s *DefaultContextService) GetSearchConfig() config.SearchConfig {
	if s.searchEngine == nil {
		return config.SearchConfig{}
	}
	status := s.searchEngine.GetStatus()
	if cfg, ok := status["config"].(config.SearchConfig); ok {
		return cfg
	}
	return config.SearchConfig{}
}

// GetAssignedStores returns the document stores assigned to this agent
// Returns nil if agent has access to all stores, or a list of store names if scoped
// Note: Empty slices are never passed to DefaultContextService (NoOpContextService is used instead)
func (s *DefaultContextService) GetAssignedStores() []string {
	if s.assignedStores == nil {
		return nil // nil means all stores
	}
	// Return a copy to prevent modification
	result := make([]string, len(s.assignedStores))
	copy(result, s.assignedStores)
	return result
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
	searchConfig   config.SearchConfig
	contextService reasoning.ContextService
	historyService reasoning.HistoryService
}

func NewPromptService(
	promptConfig config.PromptConfig,
	searchConfig config.SearchConfig,
	contextService reasoning.ContextService,
	historyService reasoning.HistoryService,
) reasoning.PromptService {
	return &DefaultPromptService{
		promptConfig:   promptConfig,
		searchConfig:   searchConfig,
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

	if s.promptConfig.IncludeContext != nil && *s.promptConfig.IncludeContext && s.contextService != nil {
		contextResults, err := s.contextService.SearchContext(ctx, query)
		if err == nil && len(contextResults) > 0 {
			var contextText strings.Builder
			contextText.WriteString("Relevant context from documents:\n")

			// Determine max documents: use include_context_limit if set, otherwise use search.top_k
			maxDocs := len(contextResults) // Default: use all retrieved results
			if s.promptConfig.IncludeContextLimit != nil && *s.promptConfig.IncludeContextLimit > 0 {
				maxDocs = *s.promptConfig.IncludeContextLimit
			} else if s.searchConfig.TopK > 0 {
				maxDocs = s.searchConfig.TopK
			}
			if maxDocs > len(contextResults) {
				maxDocs = len(contextResults)
			}

			// Determine max content length: use include_context_max_length if set, otherwise default 500
			maxContentLen := 500
			if s.promptConfig.IncludeContextMaxLength != nil && *s.promptConfig.IncludeContextMaxLength > 0 {
				maxContentLen = *s.promptConfig.IncludeContextMaxLength
			}

			for i, doc := range contextResults {
				if i >= maxDocs {
					break
				}
				content := doc.Content
				if len(content) > maxContentLen {
					content = content[:maxContentLen] + "..."
				}

				// Extract store name (data source) from metadata - required
				storeName := "unknown"
				if doc.Metadata != nil {
					if sn, ok := doc.Metadata["store_name"].(string); ok && sn != "" {
						storeName = sn
					}
				}

				// Build description from store information if available
				description := s.buildStoreDescription(storeName)

				// Format with data source name and description
				if description != "" {
					contextText.WriteString(fmt.Sprintf("[Data source: %s (%s)] %s\n", storeName, description, content))
				} else {
					contextText.WriteString(fmt.Sprintf("[Data source: %s] %s\n", storeName, content))
				}
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

// buildStoreDescription builds a description for a document store from its config and status
func (s *DefaultPromptService) buildStoreDescription(storeName string) string {
	if storeName == "" || storeName == "unknown" {
		return ""
	}

	// Try to get store from registry
	store, exists := hectorcontext.GetDocumentStoreFromRegistry(storeName)
	if !exists || store == nil {
		return ""
	}

	// Use shared store description builder
	return hectorcontext.BuildStoreDescription(store)
}

func (s *DefaultPromptService) composeSystemPromptFromSlots(slots reasoning.PromptSlots) string {
	var prompt strings.Builder

	if slots.SystemRole != "" {
		prompt.WriteString(slots.SystemRole)
		prompt.WriteString("\n\n")
	}

	if slots.Instructions != "" {
		prompt.WriteString(slots.Instructions)
		prompt.WriteString("\n\n")
	}

	if slots.UserGuidance != "" {
		prompt.WriteString("<user_guidance>\n")
		prompt.WriteString(slots.UserGuidance)
		prompt.WriteString("\n</user_guidance>")
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

func (s *DefaultLLMService) GenerateStreamingChunks(ctx context.Context, messages []*pb.Message, tools []llms.ToolDefinition) (<-chan llms.StreamChunk, error) {
	return s.llmProvider.GenerateStreaming(ctx, messages, tools)
}

func (s *DefaultLLMService) Generate(ctx context.Context, messages []*pb.Message, tools []llms.ToolDefinition) (string, []*protocol.ToolCall, int, error) {
	return s.llmProvider.Generate(ctx, messages, tools)
}

func (s *DefaultLLMService) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []llms.ToolDefinition, outputCh chan<- string) ([]*protocol.ToolCall, int, error) {
	streamCh, err := s.llmProvider.GenerateStreaming(ctx, messages, tools)
	if err != nil {
		return nil, 0, err
	}

	var toolCalls []*protocol.ToolCall
	var tokens int

	for chunk := range streamCh {
		switch chunk.Type {
		case "text":
			outputCh <- chunk.Text
		case "thinking":
			// Thinking chunks are handled separately - they need to be converted to parts
			// For now, we'll pass them through as text with a special marker
			// The agent layer will need to handle this differently
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

func (s *DefaultLLMService) GenerateStructured(ctx context.Context, messages []*pb.Message, tools []llms.ToolDefinition, config *llms.StructuredOutputConfig) (string, []*protocol.ToolCall, int, error) {

	structProvider, ok := s.llmProvider.(llms.StructuredOutputProvider)
	if !ok {

		return "", nil, 0, fmt.Errorf("provider does not support structured output")
	}

	if !structProvider.SupportsStructuredOutput() {
		return "", nil, 0, fmt.Errorf("provider does not support structured output")
	}

	return structProvider.GenerateStructured(ctx, messages, tools, config)
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
		// Return error content if available, otherwise use Error field
		// This ensures LLM sees descriptive error messages
		errorContent := result.Content
		if errorContent == "" && result.Error != "" {
			errorContent = result.Error
		}
		if errorContent == "" {
			errorContent = "Tool execution failed"
		}
		return errorContent, result.Metadata, fmt.Errorf("tool failed: %s", result.Error)
	}

	return result.Content, result.Metadata, nil
}

func (s *DefaultToolService) GetAvailableTools() []llms.ToolDefinition {
	if s.toolRegistry == nil {
		return []llms.ToolDefinition{}
	}

	// Always exclude internal tools (they're only for document parsing, not agent use)
	allToolInfos := s.toolRegistry.ListToolsWithFilter(true)
	result := make([]llms.ToolDefinition, 0, len(allToolInfos))

	if s.allowedTools == nil {
		// nil means all non-internal tools
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

// cleanSchema recursively cleans a schema map to remove empty strings and ensure validity
func cleanSchema(schema map[string]interface{}) map[string]interface{} {
	if schema == nil {
		return nil
	}

	cleaned := make(map[string]interface{})
	for key, value := range schema {
		switch v := value.(type) {
		case string:
			// Skip empty strings
			if v != "" {
				cleaned[key] = v
			}
		case map[string]interface{}:
			// Recursively clean nested schemas
			if cleanedNested := cleanSchema(v); len(cleanedNested) > 0 {
				cleaned[key] = cleanedNested
			}
		case []interface{}:
			// Clean arrays by filtering out empty strings
			cleanedArray := make([]interface{}, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					if str != "" {
						cleanedArray = append(cleanedArray, str)
					}
				} else if nestedMap, ok := item.(map[string]interface{}); ok {
					if cleanedNested := cleanSchema(nestedMap); len(cleanedNested) > 0 {
						cleanedArray = append(cleanedArray, cleanedNested)
					}
				} else {
					cleanedArray = append(cleanedArray, item)
				}
			}
			if len(cleanedArray) > 0 {
				cleaned[key] = cleanedArray
			}
		default:
			// Keep other types as-is
			cleaned[key] = value
		}
	}

	// If type is missing or empty, return nil to indicate invalid schema
	typeVal, hasType := schema["type"].(string)
	if !hasType || typeVal == "" {
		return nil
	}
	// Ensure type is in cleaned result
	if _, exists := cleaned["type"]; !exists {
		cleaned["type"] = typeVal
	}

	return cleaned
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
		// Skip parameters with empty type (shouldn't happen, but be defensive)
		if param.Type == "" {
			continue
		}

		propSchema := map[string]interface{}{
			"type": param.Type,
		}

		// Only add description if it's not empty
		if param.Description != "" {
			propSchema["description"] = param.Description
		}

		if param.Type == "array" {
			if param.Items != nil {
				// Clean up items schema to ensure no empty strings
				if itemsSchema := cleanSchema(param.Items); itemsSchema != nil {
					propSchema["items"] = itemsSchema
				} else {
					// Invalid items schema (missing type), default to string
					propSchema["items"] = map[string]interface{}{
						"type": "string",
					}
				}
			} else {
				// Items not provided - OpenAI requires it for arrays, default to string
				propSchema["items"] = map[string]interface{}{
					"type": "string",
				}
			}
		}

		properties[param.Name] = propSchema

		if param.Required {
			required = append(required, param.Name)
		}

		// Filter out empty enum values
		if len(param.Enum) > 0 {
			filteredEnum := make([]string, 0, len(param.Enum))
			for _, val := range param.Enum {
				if val != "" {
					filteredEnum = append(filteredEnum, val)
				}
			}
			if len(filteredEnum) > 0 {
				propSchema["enum"] = filteredEnum
			}
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
