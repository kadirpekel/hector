package tools

import (
	"context"
	"time"
)

// ============================================================================
// TOOL SYSTEM INTERFACES
// ============================================================================
// ToolInfo represents metadata about a tool
type ToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []ToolParameter `json:"parameters,omitempty"`
	ServerURL   string          `json:"server_url,omitempty"` // Source identifier
}

// ToolParameter represents a tool parameter definition
type ToolParameter struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Required    bool                   `json:"required"`
	Default     interface{}            `json:"default,omitempty"`
	Enum        []string               `json:"enum,omitempty"`
	Items       map[string]interface{} `json:"items,omitempty"` // For array types
}

// ToolCall represents a standardized tool call
type ToolCall struct {
	Name          string                 `json:"name"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	DisplayDirect bool                   `json:"display_direct,omitempty"` // Whether to display results directly to user
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success       bool                   `json:"success"`
	Content       string                 `json:"content,omitempty"`
	Output        interface{}            `json:"output,omitempty"` // Generic output field
	Error         string                 `json:"error,omitempty"`
	ToolName      string                 `json:"tool_name"`
	ExecutionTime time.Duration          `json:"execution_time,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

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

// ToolSource represents a source of tools (local, MCP server, plugins, etc.)
type ToolSource interface {
	// GetName returns the source name
	GetName() string

	// GetType returns the source type (local, mcp, plugin, etc.)
	GetType() string

	// DiscoverTools discovers and registers tools from this source
	DiscoverTools(ctx context.Context) error

	// ListTools returns all tools available in this source
	ListTools() []ToolInfo

	// GetTool retrieves a specific tool by name
	GetTool(name string) (Tool, bool)
}
