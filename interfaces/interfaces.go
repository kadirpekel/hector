// Package interfaces provides core interface definitions used across the Hector framework.
// This package centralizes all interface definitions to avoid duplication and improve maintainability.
package interfaces

import (
	"context"
	"time"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// REASONING INTERFACES
// ============================================================================

// ReasoningEngine defines the interface for all reasoning implementations
type ReasoningEngine interface {
	// ExecuteReasoning executes the reasoning process for a given query
	ExecuteReasoning(ctx context.Context, query string) (*AgentResponse, error)

	// ExecuteReasoningStreaming executes reasoning with streaming output
	ExecuteReasoningStreaming(ctx context.Context, query string) (<-chan string, error)

	// GetName returns the name of the reasoning engine
	GetName() string

	// GetDescription returns a description of the reasoning engine
	GetDescription() string
}

// ReasoningEngineFactory defines the interface for creating reasoning engines
type ReasoningEngineFactory interface {
	// CreateEngine creates a reasoning engine of the specified type
	CreateEngine(engineType string, agent Agent, config config.ReasoningConfig) (ReasoningEngine, error)

	// ListAvailableEngines returns a list of available reasoning engine types
	ListAvailableEngines() []string

	// GetEngineInfo returns information about a specific engine type
	GetEngineInfo(engineType string) (ReasoningEngineInfo, error)
}

// Agent defines the core contract for AI agents
type Agent interface {
	// Core agent functionality
	GatherContext(ctx context.Context, query string) ([]SearchResult, error)
	ExecuteTools(ctx context.Context, query string) (map[string]ToolResult, error)
	BuildPrompt(query string, context []SearchResult, toolResults map[string]ToolResult) string
	ExtractSources(context []SearchResult) []string

	// Reasoning functionality
	ExecuteQueryWithReasoning(ctx context.Context, query string, reasoningConfig config.ReasoningConfig) (*AgentResponse, error)
	ExecuteQueryWithReasoningStreaming(ctx context.Context, query string, reasoningConfig config.ReasoningConfig) (<-chan string, error)

	// Component access
	GetHistory() ConversationHistoryInterface
	GetToolRegistry() ToolRegistryInterface
	GetLLM() LLMInterface

	// Configuration
	GetConfig() *config.AgentConfig
}

// ConversationHistoryInterface defines what reasoning engines need from history
type ConversationHistoryInterface interface {
	GetRecentConversationMessages(count int) []ConversationMessage
	AddMessage(role, content string, metadata map[string]interface{})
}

// ToolRegistryInterface defines what reasoning engines need from tool registry
type ToolRegistryInterface interface {
	ListTools() []ToolInfo
	ExecuteTool(ctx context.Context, toolName string, parameters map[string]interface{}) (ToolResult, error)
}

// LLMInterface defines what reasoning engines need from LLM providers
type LLMInterface interface {
	Generate(prompt string) (string, int, error)
	GenerateStreaming(prompt string) (<-chan string, error)
}

// ============================================================================
// TOOL SYSTEM INTERFACES
// ============================================================================

// Tool represents a common interface for all tools (local and remote)
type Tool interface {
	// GetInfo returns metadata about the tool
	GetInfo() ToolInfo

	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error)

	// GetName returns the tool name (convenience method)
	GetName() string

	// GetDescription returns the tool description (convenience method)
	GetDescription() string
}

// ToolRepository represents a source of tools (local, MCP server, plugins, etc.)
type ToolRepository interface {
	// GetName returns the repository name
	GetName() string

	// GetType returns the repository type (local, mcp, plugin, etc.)
	GetType() string

	// DiscoverTools discovers and registers tools from this repository
	DiscoverTools(ctx context.Context) error

	// ListTools returns all tools available in this repository
	ListTools() []ToolInfo

	// GetTool retrieves a specific tool by name
	GetTool(name string) (Tool, bool)

	// IsHealthy returns true if the repository is operational
	IsHealthy(ctx context.Context) bool
}

// ToolRegistry manages multiple tool repositories and provides centralized access
type ToolRegistry interface {
	// RegisterRepository adds a tool repository to the registry
	RegisterRepository(repository ToolRepository) error

	// DiscoverAllTools discovers tools from all registered repositories
	DiscoverAllTools(ctx context.Context) error

	// GetTool retrieves a tool by name
	GetTool(name string) (Tool, bool)

	// ListTools returns all available tools
	ListTools() []ToolInfo

	// ExecuteTool executes a tool by name with the given arguments
	ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (ToolResult, error)

	// GetRepositoryStatus returns the health status of all repositories
	GetRepositoryStatus(ctx context.Context) map[string]bool

	// GetToolSource returns the repository name that provides a specific tool
	GetToolSource(toolName string) (string, bool)

	// ListToolsByRepository returns tools grouped by repository
	ListToolsByRepository() map[string][]ToolInfo
}

// ============================================================================
// EXECUTOR INTERFACES
// ============================================================================

// Executor represents a workflow execution engine
type Executor interface {
	// GetName returns the executor name (e.g., "dag", "autonomous")
	GetName() string

	// GetType returns the executor type for identification
	GetType() string

	// Execute runs the workflow with the given context and team
	Execute(ctx context.Context, team interface{}) (interface{}, error)

	// CanHandle returns true if this executor can handle the given workflow
	CanHandle(workflow *config.WorkflowConfig) bool

	// GetCapabilities returns the executor's capabilities
	GetCapabilities() ExecutorCapabilities

	// IsHealthy returns true if the executor is ready to execute workflows
	IsHealthy(ctx context.Context) bool
}

// ExecutorRegistry manages multiple workflow executors
type ExecutorRegistry interface {
	// RegisterExecutor adds an executor to the registry
	RegisterExecutor(executor Executor) error

	// GetExecutor retrieves an executor by name
	GetExecutor(name string) (Executor, bool)

	// ListExecutors returns all registered executors
	ListExecutors() []Executor

	// GetExecutorForWorkflow finds the best executor for a given workflow
	GetExecutorForWorkflow(workflow *config.WorkflowConfig) (Executor, error)

	// ExecuteWorkflow executes a workflow using the appropriate executor
	ExecuteWorkflow(ctx context.Context, workflow *config.WorkflowConfig, team interface{}) (interface{}, error)

	// GetExecutorCapabilities returns capabilities for all executors
	GetExecutorCapabilities() map[string]ExecutorCapabilities

	// GetHealthStatus returns the health status of all executors
	GetHealthStatus(ctx context.Context) map[string]bool
}

// ============================================================================
// SHARED TYPES (to avoid circular dependencies)
// ============================================================================

// AgentResponse represents a response from an agent
type AgentResponse struct {
	Answer      string                `json:"answer"`
	Context     []SearchResult        `json:"context,omitempty"`
	Sources     []string              `json:"sources,omitempty"`
	ToolResults map[string]ToolResult `json:"tool_results,omitempty"`
	TokensUsed  int                   `json:"tokens_used"`
	Duration    time.Duration         `json:"duration"`
	Confidence  float64               `json:"confidence,omitempty"`
}

// ToolInfo represents metadata about a tool
type ToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []ToolParameter `json:"parameters,omitempty"`
	ServerURL   string          `json:"server_url,omitempty"` // Source repository identifier
}

// ToolParameter represents a tool parameter definition
type ToolParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success       bool                   `json:"success"`
	Content       string                 `json:"content,omitempty"`
	Output        interface{}            `json:"output,omitempty"` // For reasonings package compatibility
	Error         string                 `json:"error,omitempty"`
	ToolName      string                 `json:"tool_name"`
	ExecutionTime time.Duration          `json:"execution_time,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SearchResult represents a search result
type SearchResult struct {
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ConversationMessage represents a conversation message
type ConversationMessage struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutorCapabilities describes what an executor can do
type ExecutorCapabilities struct {
	SupportsParallelExecution bool     `json:"supports_parallel_execution"`
	SupportsDynamicPlanning   bool     `json:"supports_dynamic_planning"`
	SupportsStaticWorkflows   bool     `json:"supports_static_workflows"`
	RequiredFeatures          []string `json:"required_features"`
	MaxConcurrency            int      `json:"max_concurrency"`
	SupportsRollback          bool     `json:"supports_rollback"`
}

// ReasoningEngineInfo provides information about a reasoning engine
type ReasoningEngineInfo struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Features    []string             `json:"features"`
	Parameters  []ReasoningParameter `json:"parameters"`
	Examples    []ReasoningExample   `json:"examples"`
}

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

// ExecutorValidationError represents validation errors for executors
type ExecutorValidationError struct {
	ExecutorName string
	WorkflowMode string
	Message      string
}

func (e *ExecutorValidationError) Error() string {
	return e.Message
}
