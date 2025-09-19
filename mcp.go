package hector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ============================================================================
// MCP INFRASTRUCTURE - CORE TOOLING MECHANISM
// ============================================================================

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Name        string
	URL         string
	Description string
	Client      *http.Client
}

// ToolInfo represents a tool discovered from an MCP server
type ToolInfo struct {
	Name        string
	Description string
	Parameters  []ToolParameter
	ServerURL   string
}

// ToolParameter defines a tool parameter for MCP tools
type ToolParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // "string", "number", "boolean", "array", "object"
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"` // For enum types
}

// ToolResult represents the result of MCP tool execution
type ToolResult struct {
	Content       string                 `json:"content"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
	ToolName      string                 `json:"tool_name"`
	ExecutionTime int64                  `json:"execution_time_ms,omitempty"`
}

// MCPInfrastructure handles all MCP communication and tool discovery
type MCPInfrastructure struct {
	servers []MCPServer
	tools   map[string]ToolInfo
}

// NewMCPInfrastructure creates a new MCP infrastructure
func NewMCPInfrastructure() *MCPInfrastructure {
	return &MCPInfrastructure{
		servers: make([]MCPServer, 0),
		tools:   make(map[string]ToolInfo),
	}
}

// AddServer adds an MCP server to the infrastructure
func (mcp *MCPInfrastructure) AddServer(name, url, description string) {
	mcp.servers = append(mcp.servers, MCPServer{
		Name:        name,
		URL:         url,
		Description: description,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	})
}

// DiscoverAllTools discovers tools from all configured MCP servers
func (mcp *MCPInfrastructure) DiscoverAllTools(ctx context.Context) error {
	for _, server := range mcp.servers {
		fmt.Printf("Discovering tools from MCP server: %s\n", server.Name)

		tools, err := mcp.discoverToolsFromServer(ctx, server)
		if err != nil {
			fmt.Printf("Failed to discover tools from %s: %v\n", server.Name, err)
			continue
		}

		fmt.Printf("Discovered %d tools from %s\n", len(tools), server.Name)

		// Register discovered tools
		for _, tool := range tools {
			mcp.tools[tool.Name] = tool
			fmt.Printf("  Registered tool: %s - %s\n", tool.Name, tool.Description)
		}
	}

	return nil
}

// GetTool retrieves a tool by name
func (mcp *MCPInfrastructure) GetTool(name string) (ToolInfo, bool) {
	tool, exists := mcp.tools[name]
	return tool, exists
}

// ListTools returns all discovered tools
func (mcp *MCPInfrastructure) ListTools() []ToolInfo {
	var tools []ToolInfo
	for _, tool := range mcp.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ExecuteTool executes a tool via MCP
func (mcp *MCPInfrastructure) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (ToolResult, error) {
	tool, exists := mcp.GetTool(toolName)
	if !exists {
		return mcp.createToolResult(toolName, "", fmt.Errorf("tool %s not found", toolName)), fmt.Errorf("tool %s not found", toolName)
	}

	// Find the server for this tool
	var server MCPServer
	for _, s := range mcp.servers {
		if s.URL == tool.ServerURL {
			server = s
			break
		}
	}

	// Make MCP request
	params := MCPCallParams{
		Name:      toolName,
		Arguments: args,
	}

	mcpResponse, err := mcp.makeMCPRequest(ctx, server, "tools/call", params)
	if err != nil {
		return mcp.createToolResult(toolName, "", err), err
	}

	// Check for MCP error
	if mcpResponse.Error != nil {
		err := fmt.Errorf("MCP error: %s", mcpResponse.Error.Message)
		return mcp.createToolResult(toolName, "", err), err
	}

	// Extract content from result
	content := ""
	if result, ok := mcpResponse.Result.(map[string]interface{}); ok {
		if contentArray, ok := result["content"].([]interface{}); ok {
			for _, item := range contentArray {
				if contentItem, ok := item.(map[string]interface{}); ok {
					if text, ok := contentItem["text"].(string); ok {
						content += text + "\n"
					}
				}
			}
		}
	}

	result := mcp.createToolResult(toolName, content, nil)
	result.Metadata = map[string]interface{}{
		"mcp_server": server.URL,
		"method":     "tools/call",
	}
	return result, nil
}

// discoverToolsFromServer discovers tools from a specific MCP server
func (mcp *MCPInfrastructure) discoverToolsFromServer(ctx context.Context, server MCPServer) ([]ToolInfo, error) {
	mcpResponse, err := mcp.makeMCPRequest(ctx, server, "tools/list", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	// Check for MCP error
	if mcpResponse.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", mcpResponse.Error.Message)
	}

	// Extract tools from result
	var tools []ToolInfo
	if result, ok := mcpResponse.Result.(map[string]interface{}); ok {
		if toolsArray, ok := result["tools"].([]interface{}); ok {
			for _, toolItem := range toolsArray {
				if tool, ok := toolItem.(map[string]interface{}); ok {
					toolInfo := ToolInfo{
						Name:        getString(tool, "name"),
						Description: getString(tool, "description"),
						ServerURL:   server.URL,
					}

					// Extract parameters
					if params, ok := tool["inputSchema"].(map[string]interface{}); ok {
						if properties, ok := params["properties"].(map[string]interface{}); ok {
							for paramName, paramData := range properties {
								if param, ok := paramData.(map[string]interface{}); ok {
									toolInfo.Parameters = append(toolInfo.Parameters, ToolParameter{
										Name:        paramName,
										Type:        getString(param, "type"),
										Description: getString(param, "description"),
										Required:    isRequired(params, paramName),
									})
								}
							}
						}
					}

					tools = append(tools, toolInfo)
				}
			}
		}
	}

	return tools, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// makeMCPRequest makes an HTTP request to an MCP server and parses the SSE response
func (mcp *MCPInfrastructure) makeMCPRequest(ctx context.Context, server MCPServer, method string, params interface{}) (*MCPResponse, error) {
	// Build MCP request
	request := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", server.URL, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	// Execute request
	resp, err := server.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse SSE response
	responseStr := string(responseBody)
	var jsonData string
	lines := strings.Split(responseStr, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			jsonData = strings.TrimPrefix(line, "data: ")
			break
		}
	}

	if jsonData == "" {
		return nil, fmt.Errorf("no JSON data found in SSE response")
	}

	// Parse MCP response
	var mcpResponse MCPResponse
	if err := json.Unmarshal([]byte(jsonData), &mcpResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &mcpResponse, nil
}

// createToolResult creates a ToolResult with consistent error handling
func (mcp *MCPInfrastructure) createToolResult(toolName string, content string, err error) ToolResult {
	if err != nil {
		return ToolResult{
			Content:  "",
			Success:  false,
			Error:    err.Error(),
			ToolName: toolName,
		}
	}
	return ToolResult{
		Content:  content,
		Success:  true,
		ToolName: toolName,
	}
}

// ============================================================================
// MCP PROTOCOL STRUCTURES
// ============================================================================

// MCPRequest represents an MCP request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPCallParams represents parameters for tools/call
type MCPCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func isRequired(schema map[string]interface{}, paramName string) bool {
	if required, ok := schema["required"].([]interface{}); ok {
		for _, req := range required {
			if req == paramName {
				return true
			}
		}
	}
	return false
}
