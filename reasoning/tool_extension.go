package reasoning

import (
	"context"
	"fmt"
	"strings"

	"github.com/kadirpekel/hector/tools"
)

// ============================================================================
// NATIVE TOOL EXTENSION - DIRECT INTEGRATION
// ============================================================================

// ToolExtension provides native tool functionality as an extension
type ToolExtension struct {
	toolRegistry     *tools.ToolRegistry
	extensionService ExtensionService
}

// NewToolExtension creates a new native tool extension
func NewToolExtension(toolRegistry *tools.ToolRegistry, extensionService ExtensionService) *ToolExtension {
	return &ToolExtension{
		toolRegistry:     toolRegistry,
		extensionService: extensionService,
	}
}

// CreateExtension creates the tools extension definition
func (te *ToolExtension) CreateExtension() ExtensionDefinition {
	// Get available tools for prompt format
	toolsList := te.toolRegistry.ListTools()
	var toolsInfo strings.Builder
	if len(toolsList) > 0 {
		for _, tool := range toolsList {
			toolsInfo.WriteString(fmt.Sprintf("\n**%s**\n", tool.Name))
			toolsInfo.WriteString(fmt.Sprintf("%s\n", tool.Description))

			if len(tool.Parameters) > 0 {
				toolsInfo.WriteString("Parameters:\n")
				for _, param := range tool.Parameters {
					// Format parameter with type and required status
					required := ""
					if param.Required {
						required = " (required)"
					}
					toolsInfo.WriteString(fmt.Sprintf("  - %s (%s)%s: %s\n",
						param.Name, param.Type, required, param.Description))

					// Show enum values if available
					if len(param.Enum) > 0 {
						toolsInfo.WriteString(fmt.Sprintf("    Possible values: %v\n", param.Enum))
					}

					// Show default value if available
					if param.Default != nil {
						toolsInfo.WriteString(fmt.Sprintf("    Default: %v\n", param.Default))
					}
				}
			}
			toolsInfo.WriteString("\n")
		}
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

Tool call format:
TOOL_CALLS:
{"tool": "TOOL_NAME", "params": {"param1": "value1"}, "label": "📝 Description", "display_direct": true/false}

Format rules:
- NO markdown formatting around tool calls
- Write TOOL_CALLS: on its own line, follow with JSON
- "display_direct" and "label" are TOP LEVEL fields, not inside "params"

display_direct:
- true = User wants raw output (e.g., "show me file")
- false = You need to analyze first (e.g., "summarize", "weather")

Tool patterns:
- Before calling a tool, check if you already called it and got results
- If the same tool returns identical data twice, it likely has limitations
- Example: Weather tool only provides current data, not forecasts
- When you recognize a limitation, explain what you CAN provide

`, toolsInfo.String()),
	}
}

// processToolCall processes tool call content (extracts label for user display)
func (te *ToolExtension) processToolCall(content string) (string, string) {
	if label := te.extensionService.ExtractField(content, "label"); label != "" {
		return "\n" + label, content
	}
	return "\n🔧 Executing tool...", content
}

// executeToolCall executes the tool call directly with tool registry
func (te *ToolExtension) executeToolCall(ctx context.Context, rawData string) (ExtensionResult, error) {
	// Parse tool call JSON using extension service
	var toolCall ParsedToolCall
	if err := te.extensionService.ParseJSON(rawData, &toolCall); err != nil {
		return ExtensionResult{
			Name:    "tools",
			Success: false,
			Error:   fmt.Sprintf("Failed to parse tool call: %v", err),
		}, nil
	}

	// Validate required fields using extension service
	if err := te.extensionService.ValidateRequiredFields(map[string]interface{}{
		"tool": toolCall.Tool,
	}, []string{"tool"}); err != nil {
		return ExtensionResult{
			Name:    "tools",
			Success: false,
			Error:   err.Error(),
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

// ParsedToolCall represents a parsed tool call
type ParsedToolCall struct {
	Tool          string                 `json:"tool"`
	Params        map[string]interface{} `json:"params"`
	Label         string                 `json:"label,omitempty"`
	DisplayDirect bool                   `json:"display_direct,omitempty"`
}
