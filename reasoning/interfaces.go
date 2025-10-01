package reasoning

import (
	"context"
	"time"

	"github.com/kadirpekel/hector/config"
	hectorcontext "github.com/kadirpekel/hector/context"
	"github.com/kadirpekel/hector/databases"
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

// LLMService defines LLM capabilities
type LLMService interface {
	GenerateLLM(prompt string) (string, int, error)
	GenerateLLMStreaming(prompt string) (<-chan string, error)
	GetLastRawResponse() string // Get the raw response from the last streaming call

	// Extension processing methods
	SetExtensionService(service ExtensionService)
	GetExtensionCalls() []ExtensionCall
	GetExtensionResults() map[string]ExtensionResult
}

// ContextService defines context and search capabilities
type ContextService interface {
	SearchContext(ctx context.Context, query string) ([]databases.SearchResult, error)
	ExtractSources(context []databases.SearchResult) []string
}

// ToolCall represents a tool call (shared type)
type ToolCall struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// LLMResponse represents a processed LLM response with tool results
type LLMResponse struct {
	Content     string                      `json:"content"`
	ToolCalls   []ToolCall                  `json:"tool_calls,omitempty"`
	ToolResults map[string]tools.ToolResult `json:"tool_results,omitempty"`
}

// ToolService is replaced by ExtensionService - tools are now extensions
// This interface is kept for compatibility but should not be used in new code

// PromptService defines modern composable prompt building capabilities
type PromptService interface {
	// BuildDefaultPromptData creates standard PromptData with common fields populated (extensionResults optional)
	BuildDefaultPromptData(ctx context.Context, query string, contextService ContextService, historyService HistoryService, extensionService ExtensionService, extensionResults ...map[string]ExtensionResult) (PromptData, error)

	// BuildPromptFromParts builds a prompt using composable parts with map (RECOMMENDED)
	BuildPromptFromParts(templateParts map[string]string, data PromptData) (string, error)

	// BuildPromptWithServices builds a prompt with template parts and auto-populated data (CONVENIENCE)
	BuildPromptWithServices(ctx context.Context, query string, templateParts map[string]string, contextService ContextService, historyService HistoryService, extensionService ExtensionService) (string, error)

	// BuildDefaultPrompt builds a prompt with default template parts and auto-populated data (SHORTCUT)
	BuildDefaultPrompt(ctx context.Context, query string, contextService ContextService, historyService HistoryService, extensionService ExtensionService, extensionResults ...map[string]ExtensionResult) (string, error)
}

// HistoryService defines history management capabilities
type HistoryService interface {
	GetRecentHistory(count int) []hectorcontext.ConversationMessage
	AddToHistory(role, content string, metadata map[string]interface{})
}

// ============================================================================
// PROMPT DATA STRUCTURE
// ============================================================================

// PromptData contains all data needed for prompt building
type PromptData struct {
	Query            string                              `json:"query"`
	Context          []databases.SearchResult            `json:"context,omitempty"`
	History          []hectorcontext.ConversationMessage `json:"history,omitempty"`
	Extensions       []ExtensionDefinition               `json:"extensions,omitempty"`
	ExtensionService ExtensionService                    `json:"-"` // For formatting
	ExtensionResults map[string]ExtensionResult          `json:"extension_results,omitempty"`
	Variables        map[string]interface{}              `json:"variables,omitempty"`
}

// ============================================================================
// REASONING ENGINE CAPABILITIES - WHAT AGENT NEEDS FROM REASONING
// ============================================================================

// ReasoningEngine defines the interface for all reasoning implementations
// All reasoning engines MUST support streaming output - this is the only entry point
type ReasoningEngine interface {
	// Execute executes the reasoning process with streaming output
	// This is the ONLY method for reasoning execution - no non-streaming variant
	Execute(ctx context.Context, query string) (<-chan string, error)

	// GetName returns the name of the reasoning engine
	GetName() string

	// GetDescription returns a description of the reasoning engine
	GetDescription() string
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
	Context() ContextService
	Extensions() ExtensionService
	Prompt() PromptService
	History() HistoryService
}

// DefaultAgentServices provides a concrete implementation of AgentServices
type DefaultAgentServices struct {
	config           config.ReasoningConfig
	llmService       LLMService
	contextService   ContextService
	extensionService ExtensionService
	promptService    PromptService
	historyService   HistoryService
}

// NewAgentServices creates a new AgentServices implementation
func NewAgentServices(
	config config.ReasoningConfig,
	llmService LLMService,
	contextService ContextService,
	extensionService ExtensionService,
	promptService PromptService,
	historyService HistoryService,
) AgentServices {
	return &DefaultAgentServices{
		config:           config,
		llmService:       llmService,
		contextService:   contextService,
		extensionService: extensionService,
		promptService:    promptService,
		historyService:   historyService,
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

// Context returns the context service
func (s *DefaultAgentServices) Context() ContextService {
	return s.contextService
}

// Extensions returns the extension service
func (s *DefaultAgentServices) Extensions() ExtensionService {
	return s.extensionService
}

// Prompt returns the prompt service
func (s *DefaultAgentServices) Prompt() PromptService {
	return s.promptService
}

// History returns the history service
func (s *DefaultAgentServices) History() HistoryService {
	return s.historyService
}

// ============================================================================
// REASONING ENGINE FACTORY - CREATES ENGINES WITH DEPENDENCIES
// ============================================================================

// ReasoningEngineFactory creates reasoning engines with injected services
type ReasoningEngineFactory interface {
	CreateEngine(engineType string, services AgentServices) (ReasoningEngine, error)
	ListAvailableEngines() []ReasoningEngineInfo
}

// ============================================================================
// REASONING ENGINE METADATA
// ============================================================================

// ReasoningParameter describes a parameter for a reasoning engine
type ReasoningParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// ReasoningExample provides an example of how to use a reasoning engine
type ReasoningExample struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      config.ReasoningConfig `json:"config"`
	Query       string                 `json:"query"`
}

// ReasoningEngineInfo provides information about a reasoning engine
type ReasoningEngineInfo struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Features    []string             `json:"features"`
	Parameters  []ReasoningParameter `json:"parameters"`
	Examples    []ReasoningExample   `json:"examples"`
}
