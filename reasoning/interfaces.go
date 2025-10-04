package reasoning

import (
	"context"
	"time"

	"github.com/kadirpekel/hector/config"
	hectorcontext "github.com/kadirpekel/hector/context"
	"github.com/kadirpekel/hector/databases"
	"github.com/kadirpekel/hector/llms"
	"github.com/kadirpekel/hector/tools"
)

// ============================================================================
// CORE RESPONSE TYPES
// ============================================================================

// ReasoningResponse represents a response from a reasoning engine
type ReasoningResponse struct {
	Answer      string                      `json:"answer"`
	Context     []databases.SearchResult    `json:"context,omitempty"`
	Sources     []string                    `json:"sources,omitempty"`
	ToolResults map[string]tools.ToolResult `json:"tool_results,omitempty"`
	TokensUsed  int                         `json:"tokens_used"`
	Duration    time.Duration               `json:"duration"`
	Confidence  float64                     `json:"confidence,omitempty"`
}

// ============================================================================
// FOCUSED SERVICE INTERFACES - WHAT REASONING NEEDS FROM AGENT
// ============================================================================

// LLMService defines LLM capabilities - only LLM responsibilities
type LLMService interface {
	// Generate generates a response with tool support (message-based)
	// Returns (text, toolCalls, tokens, error)
	// If toolCalls is non-empty, those should be executed
	// If toolCalls is empty, text contains the final response
	Generate(messages []llms.Message, tools []llms.ToolDefinition) (string, []llms.ToolCall, int, error)

	// GenerateStreaming generates a streaming response with tool support (message-based)
	// Text chunks are streamed to outputCh as they arrive
	// Returns (toolCalls, tokens, error)
	GenerateStreaming(messages []llms.Message, tools []llms.ToolDefinition, outputCh chan<- string) ([]llms.ToolCall, int, error)
}

// ToolService defines tool execution capabilities - only tool responsibilities
type ToolService interface {
	// ExecuteToolCall executes a single tool call and returns the result
	ExecuteToolCall(ctx context.Context, toolCall llms.ToolCall) (string, error)

	// GetAvailableTools returns all available tools as ToolDefinition
	GetAvailableTools() []llms.ToolDefinition

	// GetTool retrieves a specific tool by name (for strategy direct access)
	// Strategies can use this to access tool internals (e.g., TodoTool for task tracking)
	GetTool(name string) (interface{}, error)
}

// ContextService defines context and search capabilities
type ContextService interface {
	SearchContext(ctx context.Context, query string) ([]databases.SearchResult, error)
	ExtractSources(context []databases.SearchResult) []string
}

// PromptService defines modern composable prompt building capabilities
type PromptService interface {
	// BuildMessages builds a message array for multi-turn conversations with slot-based prompts
	// Parameters:
	//   - ctx: Context
	//   - query: User query
	//   - slots: Prompt slots (from strategy + user config)
	//   - currentToolConversation: Tool loop messages
	//   - additionalContext: Strategy-specific context (e.g., todos, goals)
	BuildMessages(ctx context.Context, query string, slots PromptSlots, currentToolConversation []llms.Message, additionalContext string) ([]llms.Message, error)
}

// HistoryService defines history management capabilities
type HistoryService interface {
	GetRecentHistory(count int) []hectorcontext.ConversationMessage
	AddToHistory(role, content string, metadata map[string]interface{})
}

// ============================================================================
// AGENT SERVICES INTERFACE - DEPENDENCY INJECTION
// ============================================================================

// AgentServices defines the interface that reasoning engines depend on
// This enables loose coupling, easy testing, and flexible service implementations
type AgentServices interface {
	// Configuration
	GetConfig() config.ReasoningConfig

	// Core AI Services
	LLM() LLMService
	Tools() ToolService
	Context() ContextService
	Prompt() PromptService
	History() HistoryService
}

// DefaultAgentServices provides a concrete implementation of AgentServices
type DefaultAgentServices struct {
	config         config.ReasoningConfig
	llmService     LLMService
	toolService    ToolService
	contextService ContextService
	promptService  PromptService
	historyService HistoryService
}

// NewAgentServices creates a new AgentServices implementation
func NewAgentServices(
	config config.ReasoningConfig,
	llmService LLMService,
	toolService ToolService,
	contextService ContextService,
	promptService PromptService,
	historyService HistoryService,
) AgentServices {
	return &DefaultAgentServices{
		config:         config,
		llmService:     llmService,
		toolService:    toolService,
		contextService: contextService,
		promptService:  promptService,
		historyService: historyService,
	}
}

// GetConfig returns the reasoning configuration
func (s *DefaultAgentServices) GetConfig() config.ReasoningConfig {
	return s.config
}

// LLM returns the LLM service
func (s *DefaultAgentServices) LLM() LLMService {
	return s.llmService
}

// Tools returns the tool service
func (s *DefaultAgentServices) Tools() ToolService {
	return s.toolService
}

// Context returns the context service
func (s *DefaultAgentServices) Context() ContextService {
	return s.contextService
}

// Prompt returns the prompt service
func (s *DefaultAgentServices) Prompt() PromptService {
	return s.promptService
}

// History returns the history service
func (s *DefaultAgentServices) History() HistoryService {
	return s.historyService
}
