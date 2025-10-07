package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/pkg/config"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/tools"
)

// ============================================================================
// TOOLS ARE NOW NATIVE EXTENSIONS - NO WRAPPER NEEDED
// ============================================================================

// NoOpContextService provides a no-op implementation when no document stores are configured
type NoOpContextService struct{}

// NewNoOpContextService creates a new no-op context service
func NewNoOpContextService() reasoning.ContextService {
	return &NoOpContextService{}
}

// SearchContext implements reasoning.ContextService
func (s *NoOpContextService) SearchContext(ctx context.Context, query string) ([]databases.SearchResult, error) {
	// Return empty results when no document stores are configured
	return []databases.SearchResult{}, nil
}

// ExtractSources implements reasoning.ContextService
func (s *NoOpContextService) ExtractSources(context []databases.SearchResult) []string {
	// Return empty sources when no document stores are configured
	return []string{}
}

// ============================================================================
// CONTEXT SERVICE
// ============================================================================

// ContextOptions defines options for context gathering
type ContextOptions struct {
	MaxResults int
	MinScore   float64
}

// DefaultContextService implements reasoning.ContextService
type DefaultContextService struct {
	searchEngine *hectorcontext.SearchEngine
}

// NewContextService creates a new context service
func NewContextService(searchEngine *hectorcontext.SearchEngine) reasoning.ContextService {
	return &DefaultContextService{
		searchEngine: searchEngine,
	}
}

// SearchContext implements reasoning.ContextService
func (s *DefaultContextService) SearchContext(ctx context.Context, query string) ([]databases.SearchResult, error) {
	if s.searchEngine == nil {
		return []databases.SearchResult{}, nil // Return empty results if no search engine
	}

	// Use search engine to find relevant context
	return s.searchEngine.Search(ctx, query, 5) // Limit to 5 results
}

// ExtractSources implements reasoning.ContextService
func (s *DefaultContextService) ExtractSources(context []databases.SearchResult) []string {
	sources := make([]string, 0, len(context))
	for _, result := range context {
		// Try to get source from metadata, fallback to ID
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

// ============================================================================
// PROMPT SERVICE
// ============================================================================

// DefaultPromptService implements reasoning.PromptService using composable parts
type DefaultPromptService struct {
	promptConfig   config.PromptConfig
	contextService reasoning.ContextService
	historyService reasoning.HistoryService
}

// NewPromptService creates a new prompt service
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

// BuildMessages builds a message array for multi-turn conversations with slot-based prompts
// Parameters:
//   - ctx: Context
//   - query: The current user query
//   - slots: Prompt slots (merged from strategy + user config)
//   - currentToolConversation: Messages from the current tool-calling loop
func (s *DefaultPromptService) BuildMessages(
	ctx context.Context,
	query string,
	slots reasoning.PromptSlots,
	currentToolConversation []llms.Message,
	additionalContext string,
) ([]llms.Message, error) {
	messages := make([]llms.Message, 0)

	// Compose system prompt from slots (if provided) or use config
	var systemPrompt string
	if !slots.IsEmpty() {
		systemPrompt = s.composeSystemPromptFromSlots(slots)
	} else if s.promptConfig.SystemPrompt != "" {
		// Fallback to config's system_prompt if no slots provided
		systemPrompt = s.promptConfig.SystemPrompt
	}

	if systemPrompt != "" {
		messages = append(messages, llms.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// ⭐ INJECT STRATEGY-SPECIFIC CONTEXT (e.g., todos, goals)
	if additionalContext != "" {
		messages = append(messages, llms.Message{
			Role:    "system",
			Content: additionalContext,
		})
	}

	// Add context if enabled
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
			messages = append(messages, llms.Message{
				Role:    "system",
				Content: contextText.String(),
			})
		}
	}

	// Add conversation history from HistoryService if enabled
	if s.promptConfig.IncludeHistory && s.historyService != nil {
		// Fetch history from HistoryService
		maxHistory := 10 // Default
		if s.promptConfig.MaxHistoryMessages > 0 {
			maxHistory = s.promptConfig.MaxHistoryMessages
		}

		// Extract sessionID from context (if available)
		sessionID := ""
		if sessionIDValue := ctx.Value("sessionID"); sessionIDValue != nil {
			if sid, ok := sessionIDValue.(string); ok {
				sessionID = sid
			}
		}

		historyMsgs := s.historyService.GetRecentHistory(sessionID, maxHistory)

		// Already in llms.Message format - append directly
		messages = append(messages, historyMsgs...)
	}

	// Add current tool conversation (assistant responses + tool results from this query)
	// This includes conversation history loaded in agent.go
	messages = append(messages, currentToolConversation...)

	// Add current user query if not already in history
	// (History might already have it if we're in a follow-up iteration)
	needsUserQuery := true
	if len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		if lastMsg.Role == "user" && lastMsg.Content == query {
			needsUserQuery = false
		}
	}

	if needsUserQuery {
		messages = append(messages, llms.Message{
			Role:    "user",
			Content: query,
		})
	}

	return messages, nil
}

// composeSystemPromptFromSlots composes a system prompt from slot values
// This is the standard template that all strategies use
func (s *DefaultPromptService) composeSystemPromptFromSlots(slots reasoning.PromptSlots) string {
	var prompt strings.Builder

	// System role
	if slots.SystemRole != "" {
		prompt.WriteString(slots.SystemRole)
		prompt.WriteString("\n\n")
	}

	// Reasoning instructions
	if slots.ReasoningInstructions != "" {
		prompt.WriteString(slots.ReasoningInstructions)
		prompt.WriteString("\n\n")
	}

	// Tool usage
	if slots.ToolUsage != "" {
		prompt.WriteString("<tool_usage>\n")
		prompt.WriteString(slots.ToolUsage)
		prompt.WriteString("\n</tool_usage>\n\n")
	}

	// Output format
	if slots.OutputFormat != "" {
		prompt.WriteString("<output_format>\n")
		prompt.WriteString(slots.OutputFormat)
		prompt.WriteString("\n</output_format>\n\n")
	}

	// Communication style
	if slots.CommunicationStyle != "" {
		prompt.WriteString("<communication>\n")
		prompt.WriteString(slots.CommunicationStyle)
		prompt.WriteString("\n</communication>\n\n")
	}

	// Additional instructions
	if slots.Additional != "" {
		prompt.WriteString(slots.Additional)
	}

	return strings.TrimSpace(prompt.String())
}

// ============================================================================
// LLM SERVICE
// ============================================================================

// DefaultLLMService implements reasoning.LLMService
type DefaultLLMService struct {
	llmProvider llms.LLMProvider
}

// NewLLMService creates a new LLM service
func NewLLMService(llmProvider llms.LLMProvider) reasoning.LLMService {
	return &DefaultLLMService{
		llmProvider: llmProvider,
	}
}

// Generate implements reasoning.LLMService
// Note: Non-streaming mode doesn't show tool labels in real-time (they're shown after execution)
func (s *DefaultLLMService) Generate(messages []llms.Message, tools []llms.ToolDefinition) (string, []llms.ToolCall, int, error) {
	return s.llmProvider.Generate(messages, tools)
}

// GenerateStreaming implements reasoning.LLMService
func (s *DefaultLLMService) GenerateStreaming(messages []llms.Message, tools []llms.ToolDefinition, outputCh chan<- string) ([]llms.ToolCall, int, error) {
	streamCh, err := s.llmProvider.GenerateStreaming(messages, tools)
	if err != nil {
		return nil, 0, err
	}

	var toolCalls []llms.ToolCall
	var tokens int

	for chunk := range streamCh {
		switch chunk.Type {
		case "text":
			outputCh <- chunk.Text
		case "tool_call":
			if chunk.ToolCall != nil {
				// Accumulate tool calls silently
				// Tool labels and formatting will be handled entirely by executeTools()
				// to ensure clean, consistent "🔧 label ✅" pairing
				toolCalls = append(toolCalls, *chunk.ToolCall)
			}
		case "done":
			tokens = chunk.Tokens
		case "error":
			return toolCalls, tokens, chunk.Error
		}
	}

	return toolCalls, tokens, nil
}

// GenerateStructured implements reasoning.LLMService
func (s *DefaultLLMService) GenerateStructured(messages []llms.Message, tools []llms.ToolDefinition, config *llms.StructuredOutputConfig) (string, []llms.ToolCall, int, error) {
	// Check if provider supports structured output
	structProvider, ok := s.llmProvider.(llms.StructuredOutputProvider)
	if !ok {
		// Provider doesn't implement structured output interface
		return "", nil, 0, fmt.Errorf("provider does not support structured output")
	}

	if !structProvider.SupportsStructuredOutput() {
		return "", nil, 0, fmt.Errorf("provider does not support structured output")
	}

	return structProvider.GenerateStructured(messages, tools, config)
}

// SupportsStructuredOutput implements reasoning.LLMService
func (s *DefaultLLMService) SupportsStructuredOutput() bool {
	// Check if provider implements StructuredOutputProvider interface
	structProvider, ok := s.llmProvider.(llms.StructuredOutputProvider)
	if !ok {
		return false
	}

	return structProvider.SupportsStructuredOutput()
}

// ============================================================================
// HISTORY SERVICE
// ============================================================================

// DefaultHistoryService implements reasoning.HistoryService
type DefaultHistoryService struct {
	history []llms.Message
	maxSize int
}

// NewHistoryService creates a new history service
func NewHistoryService(maxSize int) reasoning.HistoryService {
	if maxSize <= 0 {
		maxSize = 10 // Default max size
	}
	return &DefaultHistoryService{
		history: make([]llms.Message, 0),
		maxSize: maxSize,
	}
}

// AddToHistory implements reasoning.HistoryService
func (s *DefaultHistoryService) AddToHistory(sessionID string, msg llms.Message) {
	// Ignore sessionID for backward compatibility (non-session-aware)
	s.history = append(s.history, msg)

	// Simple FIFO truncation
	if len(s.history) > s.maxSize {
		s.history = s.history[len(s.history)-s.maxSize:]
	}
}

// GetRecentHistory implements reasoning.HistoryService
func (s *DefaultHistoryService) GetRecentHistory(sessionID string, count int) []llms.Message {
	// Ignore sessionID for backward compatibility (non-session-aware)
	if count <= 0 || len(s.history) == 0 {
		return []llms.Message{}
	}

	start := len(s.history) - count
	if start < 0 {
		start = 0
	}

	// Return a copy to prevent external modification
	result := make([]llms.Message, len(s.history[start:]))
	copy(result, s.history[start:])
	return result
}

// ClearHistory implements reasoning.HistoryService
func (s *DefaultHistoryService) ClearHistory(sessionID string) {
	// Ignore sessionID for backward compatibility (non-session-aware)
	s.history = make([]llms.Message, 0)
}

// ============================================================================
// OLD IMPLEMENTATIONS REMOVED - Using message-based implementations above
// ============================================================================

// ============================================================================
// TOOL SERVICE
// ============================================================================

// DefaultToolService implements reasoning.ToolService
type DefaultToolService struct {
	toolRegistry *tools.ToolRegistry
	allowedTools []string // If nil/empty, all tools are allowed
}

// NewToolService creates a new tool service with optional tool filtering
// If allowedTools is nil or empty, all tools from the registry are available
func NewToolService(toolRegistry *tools.ToolRegistry, allowedTools []string) reasoning.ToolService {
	return &DefaultToolService{
		toolRegistry: toolRegistry,
		allowedTools: allowedTools,
	}
}

// ExecuteToolCall executes a single tool call and returns the result
func (s *DefaultToolService) ExecuteToolCall(ctx context.Context, toolCall llms.ToolCall) (string, error) {
	if s.toolRegistry == nil {
		return "", fmt.Errorf("tool registry not available")
	}

	result, err := s.toolRegistry.ExecuteTool(ctx, toolCall.Name, toolCall.Arguments)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("tool failed: %s", result.Error)
	}

	return result.Content, nil
}

// GetAvailableTools returns tools available to this agent (filtered by allowedTools)
func (s *DefaultToolService) GetAvailableTools() []llms.ToolDefinition {
	if s.toolRegistry == nil {
		return []llms.ToolDefinition{}
	}

	allToolInfos := s.toolRegistry.ListTools()
	result := make([]llms.ToolDefinition, 0, len(allToolInfos))

	// If no tool filter specified, return all tools
	if len(s.allowedTools) == 0 {
		for _, toolInfo := range allToolInfos {
			result = append(result, convertToolInfoToToolDefinition(toolInfo))
		}
		return result
	}

	// Create a set of allowed tools for O(1) lookup
	allowedSet := make(map[string]bool)
	for _, toolName := range s.allowedTools {
		allowedSet[toolName] = true
	}

	// Filter tools based on allowed list
	for _, toolInfo := range allToolInfos {
		if allowedSet[toolInfo.Name] {
			result = append(result, convertToolInfoToToolDefinition(toolInfo))
		}
	}

	return result
}

// GetTool implements reasoning.ToolService
// Allows strategies to access specific tools directly (e.g., TodoTool for task tracking)
func (s *DefaultToolService) GetTool(name string) (interface{}, error) {
	if s.toolRegistry == nil {
		return nil, fmt.Errorf("tool registry not available")
	}

	return s.toolRegistry.GetTool(name)
}

// convertToolInfoToToolDefinition converts from tools.ToolInfo to llms.ToolDefinition
func convertToolInfoToToolDefinition(info tools.ToolInfo) llms.ToolDefinition {
	// Convert parameters to JSON Schema
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

		// Add items for array types (required by OpenAI)
		if param.Type == "array" && param.Items != nil {
			propSchema["items"] = param.Items
		}

		properties[param.Name] = propSchema

		if param.Required {
			required = append(required, param.Name)
		}

		// Add enum if present
		if len(param.Enum) > 0 {
			propSchema["enum"] = param.Enum
		}

		// Add default if present
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
