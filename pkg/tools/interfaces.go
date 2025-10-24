package tools

import (
	"context"
	"time"
)

type ToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []ToolParameter `json:"parameters,omitempty"`
	ServerURL   string          `json:"server_url,omitempty"`
}

type ToolParameter struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Required    bool                   `json:"required"`
	Default     interface{}            `json:"default,omitempty"`
	Enum        []string               `json:"enum,omitempty"`
	Items       map[string]interface{} `json:"items,omitempty"`
}

type ToolCall struct {
	Name          string                 `json:"name"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	DisplayDirect bool                   `json:"display_direct,omitempty"`
}

type ToolResult struct {
	Success       bool                   `json:"success"`
	Content       string                 `json:"content,omitempty"`
	Output        interface{}            `json:"output,omitempty"`
	Error         string                 `json:"error,omitempty"`
	ToolName      string                 `json:"tool_name"`
	ExecutionTime time.Duration          `json:"execution_time,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type Tool interface {
	GetInfo() ToolInfo

	Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error)

	GetName() string

	GetDescription() string
}

type ToolSource interface {
	GetName() string

	GetType() string

	DiscoverTools(ctx context.Context) error

	ListTools() []ToolInfo

	GetTool(name string) (Tool, bool)
}
