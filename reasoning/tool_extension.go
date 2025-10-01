package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/tools"
)

// ============================================================================
// NATIVE TOOL EXTENSION - DIRECT INTEGRATION
// ============================================================================

// ToolExtension provides native tool functionality as an extension
type ToolExtension struct {
	toolRegistry *tools.ToolRegistry
}

// NewToolExtension creates a new native tool extension
func NewToolExtension(toolRegistry *tools.ToolRegistry) *ToolExtension {
	return &ToolExtension{
		toolRegistry: toolRegistry,
	}
}

// CreateExtension creates the tools extension definition
func (te *ToolExtension) CreateExtension() ExtensionDefinition {
	// Get available tools for prompt format
	toolsList := te.toolRegistry.ListTools()
	var toolsInfo strings.Builder
	if len(toolsList) > 0 {
		toolsInfo.WriteString("Available tools:\n")
		for _, tool := range toolsList {
			toolsInfo.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
		}
		toolsInfo.WriteString("\n")
	}

	return ExtensionDefinition{
		Name:        "tools",
		Description: "Execute system tools for file operations, searches, and system tasks",
		OpenTag:     "TOOL_CALLS:",
		CloseTag:    "",
		Processor:   te.processToolCall,
		Executor:    te.executeToolCall,
		PromptFormat: fmt.Sprintf(`Available tools:
%s
You can use tools when needed to gather information or perform tasks. First provide a natural response, then use tools if additional information is required.

Tool format:
TOOL_CALLS:
{"tool": "TOOL_NAME", "params": {"param1": "value1", "param2": "value2"}, "label": "📝 Description...", "display_direct": true/false}

IMPORTANT: "display_direct" and "label" must be at the TOP LEVEL, not inside "params"!

How to decide display_direct value:
Ask yourself: "Should the user see the raw tool output directly, or do I need to analyze/interpret it first?"

- Set "display_direct": true when:
  * Tool output is already user-friendly and clear (file listings, simple commands, readable text)
  * User asked for raw data or direct output
  * Tool provides structured information that doesn't need interpretation

- Set "display_direct": false when:
  * Tool output is complex data that needs analysis (weather data, search results, API responses)
  * You need to summarize, interpret, or explain the results
  * Tool output is technical and needs human-friendly explanation

Examples:
- File listing: "Let me list files for you." → {"tool": "execute_command", "params": {"command": "ls"}, "label": "📂 Listing files...", "display_direct": true}
- Reading file: "Let me read that file." → {"tool": "execute_command", "params": {"command": "cat file.txt"}, "label": "📄 Reading file...", "display_direct": true}
- Weather check: "Let me check weather." → {"tool": "WEATHERMAP_WEATHER", "params": {"location": "Berlin"}, "label": "🌤️ Getting weather...", "display_direct": false}
- Web search: "Let me search for that." → {"tool": "search_tool", "params": {"query": "topic"}, "label": "🔍 Searching...", "display_direct": false}

`, toolsInfo.String()),
	}
}

// processToolCall processes tool call content (extracts label for user display)
func (te *ToolExtension) processToolCall(content string) (string, string) {
	if label := te.extractToolLabel(content); label != "" {
		return "\n" + label, content
	}
	return "\n🔧 Executing tool...", content
}

// executeToolCall executes the tool call directly with tool registry
func (te *ToolExtension) executeToolCall(ctx context.Context, rawData string) (ExtensionResult, error) {
	// Parse tool call JSON
	toolCall, err := te.parseToolCall(rawData)
	if err != nil {
		return ExtensionResult{
			Name:    "tools",
			Success: false,
			Error:   fmt.Sprintf("Failed to parse tool call: %v", err),
		}, nil
	}

	// Execute tool directly with registry
	toolResult, err := te.toolRegistry.ExecuteTool(ctx, toolCall.Tool, toolCall.Params)
	if err != nil {
		return ExtensionResult{
			Name:    "tools",
			Success: false,
			Error:   fmt.Sprintf("Tool execution failed: %v", err),
		}, nil
	}

	// Use the display_direct value from the tool call
	displayDirect := toolCall.DisplayDirect

	// Convert to extension result
	return ExtensionResult{
		Name:    "tools",
		Success: toolResult.Success,
		Content: toolResult.Content,
		Error:   toolResult.Error,
		Metadata: map[string]interface{}{
			"tool_name":      toolCall.Tool,
			"display_direct": displayDirect,
			"execution_time": toolResult.ExecutionTime,
		},
	}, nil
}

// parseToolCall parses tool call JSON
func (te *ToolExtension) parseToolCall(rawData string) (*ParsedToolCall, error) {
	lines := strings.Split(strings.TrimSpace(rawData), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var parsed ParsedToolCall
		if err := json.Unmarshal([]byte(line), &parsed); err == nil && parsed.Tool != "" {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("no valid tool call JSON found")
}

// extractToolLabel extracts label from tool call JSON
func (te *ToolExtension) extractToolLabel(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var parsed struct {
			Label string `json:"label,omitempty"`
		}
		if err := json.Unmarshal([]byte(line), &parsed); err == nil && parsed.Label != "" {
			return parsed.Label
		}
	}
	return ""
}

// ParsedToolCall represents a parsed tool call
type ParsedToolCall struct {
	Tool          string                 `json:"tool"`
	Params        map[string]interface{} `json:"params"`
	Label         string                 `json:"label,omitempty"`
	DisplayDirect bool                   `json:"display_direct,omitempty"`
}
