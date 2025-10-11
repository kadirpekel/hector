package reasoning

import (
	"context"
	"time"

	"github.com/kadirpekel/hector/pkg/a2a"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/tools"
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
// Uses A2A protocol Message types for true A2A-native architecture
type LLMService interface {
	// Generate generates a response with tool support (message-based)
	// Returns (text, toolCalls, tokens, error)
	// If toolCalls is non-empty, those should be executed
	// If toolCalls is empty, text contains the final response
	Generate(messages []a2a.Message, tools []llms.ToolDefinition) (string, []a2a.ToolCall, int, error)

	// GenerateStreaming generates a streaming response with tool support (message-based)
	// Text chunks are streamed to outputCh as they arrive
	// Returns (toolCalls, tokens, error)
	GenerateStreaming(messages []a2a.Message, tools []llms.ToolDefinition, outputCh chan<- string) ([]a2a.ToolCall, int, error)

	// GenerateStructured generates a response with structured output (JSON schema)
	// This is used internally for reliable reflection and meta-cognitive analysis
	// Returns (text, toolCalls, tokens, error)
	GenerateStructured(messages []a2a.Message, tools []llms.ToolDefinition, config *llms.StructuredOutputConfig) (string, []a2a.ToolCall, int, error)

	// SupportsStructuredOutput checks if the underlying provider supports structured output
	// Returns false for providers that don't implement StructuredOutputProvider
	SupportsStructuredOutput() bool
}

// ToolService defines tool execution capabilities - only tool responsibilities
type ToolService interface {
	// ExecuteToolCall executes a single tool call and returns the result
	ExecuteToolCall(ctx context.Context, toolCall a2a.ToolCall) (string, error)

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
// Uses A2A protocol Message types for true A2A-native architecture
type PromptService interface {
	// BuildMessages builds a message array for multi-turn conversations with slot-based prompts
	// Parameters:
	//   - ctx: Context
	//   - query: User query
	//   - slots: Prompt slots (from strategy + user config)
	//   - currentToolConversation: Tool loop messages
	//   - additionalContext: Strategy-specific context (e.g., todos, goals)
	BuildMessages(ctx context.Context, query string, slots PromptSlots, currentToolConversation []a2a.Message, additionalContext string) ([]a2a.Message, error)
}

// HistoryService defines the interface for memory management
// Implemented by pkg/memory.MemoryService
// All methods are session-aware (stateless API design)
// Uses A2A protocol Message types for true A2A-native architecture
type HistoryService interface {
	GetRecentHistory(sessionID string) ([]a2a.Message, error)
	AddToHistory(sessionID string, msg a2a.Message) error
	ClearHistory(sessionID string) error
}

// ============================================================================
// AGENT REGISTRY SERVICE INTERFACE
// Uses A2A protocol types (AgentCard) for true A2A-native architecture
// ============================================================================

// AgentRegistryEntry represents an agent entry for orchestration
// Wraps A2A AgentCard with additional orchestration metadata
type AgentRegistryEntry struct {
	ID         string         // Agent ID (key in registry)
	Card       *a2a.AgentCard // A2A protocol agent card
	Visibility string         // "public", "internal", or "private" (for visibility filtering)
}

// AgentRegistryService provides access to registered agents
// Used by reasoning strategies for multi-agent orchestration
// Returns A2A protocol AgentCard types for true protocol compliance
type AgentRegistryService interface {
	// ListAgents returns all available agent entries
	// Each entry contains A2A AgentCard + orchestration metadata
	ListAgents() []AgentRegistryEntry

	// GetAgent returns agent entry for a specific agent
	GetAgent(id string) (AgentRegistryEntry, bool)

	// FilterAgents returns agent entries matching the given IDs
	// If ids is empty, returns all agents
	FilterAgents(ids []string) []AgentRegistryEntry
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
	Registry() AgentRegistryService // Agent registry for orchestration (may be nil)
}

// DefaultAgentServices provides a concrete implementation of AgentServices
type DefaultAgentServices struct {
	config          config.ReasoningConfig
	llmService      LLMService
	toolService     ToolService
	contextService  ContextService
	promptService   PromptService
	historyService  HistoryService
	registryService AgentRegistryService
}

// NewAgentServices creates a new AgentServices implementation
func NewAgentServices(
	config config.ReasoningConfig,
	llmService LLMService,
	toolService ToolService,
	contextService ContextService,
	promptService PromptService,
	historyService HistoryService,
	registryService AgentRegistryService,
) AgentServices {
	return &DefaultAgentServices{
		config:          config,
		llmService:      llmService,
		toolService:     toolService,
		contextService:  contextService,
		promptService:   promptService,
		historyService:  historyService,
		registryService: registryService,
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

// Registry returns the agent registry service
func (s *DefaultAgentServices) Registry() AgentRegistryService {
	return s.registryService
}
