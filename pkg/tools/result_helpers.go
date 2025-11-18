package tools

import (
	"time"
)

// buildMCPErrorResult creates a standardized error ToolResult for MCP tools
// Includes MCP-specific metadata fields
func buildMCPErrorResult(toolName string, errorMsg string, executionTime time.Duration, sourceName string, serverURL string) ToolResult {
	// Ensure non-empty values for safety
	if toolName == "" {
		toolName = "unknown_mcp_tool"
	}
	if errorMsg == "" {
		errorMsg = "unknown error"
	}
	if sourceName == "" {
		sourceName = "unknown_source"
	}

	metadata := map[string]interface{}{
		"source":     sourceName,
		"tool_type":  "remote",
		"server_url": serverURL,
		"error":      errorMsg,
	}

	return ToolResult{
		Success:       false,
		Content:       "", // Empty content on error
		Error:         errorMsg,
		ToolName:      toolName,
		ExecutionTime: executionTime,
		Metadata:      metadata,
	}
}

// buildMCPSuccessResult creates a standardized success ToolResult for MCP tools
// Includes MCP-specific metadata fields
func buildMCPSuccessResult(toolName string, content string, executionTime time.Duration, sourceName string, serverURL string, responseMetadata map[string]interface{}) ToolResult {
	// Ensure non-empty toolName for safety
	if toolName == "" {
		toolName = "unknown_mcp_tool"
	}
	if sourceName == "" {
		sourceName = "unknown_source"
	}

	metadata := map[string]interface{}{
		"source":     sourceName,
		"tool_type":  "remote",
		"server_url": serverURL,
	}

	// Merge response metadata if provided
	// Note: This will overwrite base metadata keys if they exist in responseMetadata
	for k, v := range responseMetadata {
		metadata[k] = v
	}

	return ToolResult{
		Success:       true,
		Content:       content,
		Error:         "", // No error on success
		ToolName:      toolName,
		ExecutionTime: executionTime,
		Metadata:      metadata,
	}
}
