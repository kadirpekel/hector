package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kadirpekel/hector/internal/httpclient"
)

// MCPToolSource manages tools from a single MCP server
type MCPToolSource struct {
	name        string
	url         string
	description string
	httpClient  *httpclient.Client
	tools       map[string]Tool
	mu          sync.RWMutex
}

// MCPTool represents an MCP tool that implements the Tool interface
type MCPTool struct {
	toolInfo ToolInfo
	source   *MCPToolSource
}

// Request represents an MCP request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// Response represents an MCP response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents an MCP error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CallParams represents parameters for tools/call
type CallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// NewMCPToolSource creates a new MCP tool source for a single server
func NewMCPToolSource(name, url, description string) *MCPToolSource {
	if name == "" {
		name = "mcp"
	}

	return &MCPToolSource{
		name:        name,
		url:         url,
		description: description,
		httpClient: httpclient.New(
			httpclient.WithHTTPClient(&http.Client{
				Timeout: 30 * time.Second,
			}),
			httpclient.WithMaxRetries(3),
			httpclient.WithBaseDelay(2*time.Second),
		),
		tools: make(map[string]Tool),
	}
}

// NewMCPToolSourceWithConfig creates a new MCP tool source from configuration
// Note: MCP repositories are not used in the new simplified config structure
func NewMCPToolSourceWithConfig(url string) (*MCPToolSource, error) {
	if url == "" {
		return nil, fmt.Errorf("URL is required for MCP source")
	}

	return &MCPToolSource{
		name:        "mcp",
		url:         url,
		description: "",
		httpClient: httpclient.New(
			httpclient.WithHTTPClient(&http.Client{
				Timeout: 30 * time.Second,
			}),
			httpclient.WithMaxRetries(3),
			httpclient.WithBaseDelay(2*time.Second),
		),
		tools: make(map[string]Tool),
	}, nil
}

// GetName returns the source name
func (r *MCPToolSource) GetName() string {
	return r.name
}

// GetType returns the source type
func (r *MCPToolSource) GetType() string {
	return "mcp"
}

// DiscoverTools discovers tools from the configured MCP server
func (r *MCPToolSource) DiscoverTools(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing tools
	r.tools = make(map[string]Tool)

	if r.url == "" {
		return fmt.Errorf("MCP server URL not configured for source %s", r.name)
	}

	fmt.Printf("Discovering tools from MCP server: %s (%s)\n", r.name, r.url)

	// Discover tools from the server
	tools, err := r.discoverToolsFromServer(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover tools from %s: %w", r.name, err)
	}

	// Create tools for discovered tools
	for _, toolInfo := range tools {
		tool := &MCPTool{
			toolInfo: toolInfo,
			source:   r,
		}
		r.tools[toolInfo.Name] = tool
	}

	fmt.Printf("MCP source %s discovered %d tools\n", r.name, len(r.tools))
	return nil
}

// ListTools returns all tools in this source
func (r *MCPToolSource) ListTools() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []ToolInfo
	for _, tool := range r.tools {
		info := tool.GetInfo()
		// Mark with source name as source
		info.ServerURL = r.name
		tools = append(tools, info)
	}

	return tools
}

// GetTool retrieves a specific tool by name
func (r *MCPToolSource) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// discoverToolsFromServer discovers tools from the MCP server
func (r *MCPToolSource) discoverToolsFromServer(ctx context.Context) ([]ToolInfo, error) {
	response, err := r.makeRequest(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	// Check for MCP error
	if response.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", response.Error.Message)
	}

	// Extract tools from result
	var tools []ToolInfo
	if result, ok := response.Result.(map[string]interface{}); ok {
		if toolsArray, ok := result["tools"].([]interface{}); ok {
			for _, toolItem := range toolsArray {
				if tool, ok := toolItem.(map[string]interface{}); ok {
					toolInfo := ToolInfo{
						Name:        getString(tool, "name"),
						Description: getString(tool, "description"),
						ServerURL:   r.url,
					}

					// Extract parameters from JSON Schema (inputSchema)
					if params, ok := tool["inputSchema"].(map[string]interface{}); ok {
						if properties, ok := params["properties"].(map[string]interface{}); ok {
							for paramName, paramData := range properties {
								if param, ok := paramData.(map[string]interface{}); ok {
									toolParam := ToolParameter{
										Name:        paramName,
										Type:        getString(param, "type"),
										Description: getString(param, "description"),
										Required:    isRequired(params, paramName),
									}

									// Extract enum (possible values)
									if enum, ok := param["enum"].([]interface{}); ok {
										for _, val := range enum {
											if strVal, ok := val.(string); ok {
												toolParam.Enum = append(toolParam.Enum, strVal)
											}
										}
									}

									// Extract default value
									if defaultVal, ok := param["default"]; ok {
										toolParam.Default = defaultVal
									}

									// Extract examples (MCP JSON Schema extension)
									if examples, ok := param["examples"].([]interface{}); ok {
										// Add examples to description if not already there
										if len(examples) > 0 && !strings.Contains(toolParam.Description, "Example") {
											toolParam.Description += "\nExamples:"
											for _, ex := range examples {
												toolParam.Description += fmt.Sprintf("\n  %v", ex)
											}
										}
									}

									// Extract format hint (e.g., "date", "uri", "email")
									if format := getString(param, "format"); format != "" {
										toolParam.Description += fmt.Sprintf(" (format: %s)", format)
									}

									// Extract pattern (regex) if present
									if pattern := getString(param, "pattern"); pattern != "" {
										toolParam.Description += fmt.Sprintf(" (pattern: %s)", pattern)
									}

									toolInfo.Parameters = append(toolInfo.Parameters, toolParam)
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

// makeRequest makes an HTTP request to the MCP server and parses the response
func (r *MCPToolSource) makeRequest(ctx context.Context, method string, params interface{}) (*Response, error) {
	// Build MCP request
	request := Request{
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

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", r.url, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	// Execute request
	httpResp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer httpResp.Body.Close()

	// Check HTTP status
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, httpResp.Status)
	}

	// Read and parse response
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Try direct JSON parsing first
	var mcpResp Response
	if err := json.Unmarshal(responseBody, &mcpResp); err == nil {
		return &mcpResp, nil
	}

	// Try to parse as SSE format
	responseStr := string(responseBody)
	lines := strings.Split(responseStr, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			jsonData := strings.TrimPrefix(line, "data: ")
			if err := json.Unmarshal([]byte(jsonData), &mcpResp); err == nil {
				return &mcpResp, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to parse response as JSON or SSE")
}

// GetInfo returns tool information
func (t *MCPTool) GetInfo() ToolInfo {
	return t.toolInfo
}

// GetName returns the tool name
func (t *MCPTool) GetName() string {
	return t.toolInfo.Name
}

// GetDescription returns the tool description
func (t *MCPTool) GetDescription() string {
	return t.toolInfo.Description
}

// Execute runs the MCP tool
func (t *MCPTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	// Execute MCP call
	params := CallParams{
		Name:      t.toolInfo.Name,
		Arguments: args,
	}

	response, err := t.source.makeRequest(ctx, "tools/call", params)
	if err != nil {
		return ToolResult{
			Content:       "",
			Success:       false,
			Error:         err.Error(),
			ToolName:      t.toolInfo.Name,
			ExecutionTime: time.Since(start),
			Metadata: map[string]interface{}{
				"source":     t.source.name,
				"tool_type":  "remote",
				"server_url": t.source.url,
			},
		}, err
	}

	// Check for MCP error
	if response.Error != nil {
		err := fmt.Errorf("MCP error: %s", response.Error.Message)
		return ToolResult{
			Content:       "",
			Success:       false,
			Error:         err.Error(),
			ToolName:      t.toolInfo.Name,
			ExecutionTime: time.Since(start),
			Metadata: map[string]interface{}{
				"source":     t.source.name,
				"tool_type":  "remote",
				"server_url": t.source.url,
			},
		}, err
	}

	// Extract and format content
	content := t.extractContent(response.Result)

	result := ToolResult{
		Content:       strings.TrimSpace(content),
		Success:       true,
		ToolName:      t.toolInfo.Name,
		ExecutionTime: time.Since(start),
		Metadata: map[string]interface{}{
			"source":     t.source.name,
			"tool_type":  "remote",
			"server_url": t.source.url,
		},
	}

	return result, nil
}

// extractContent extracts text content from MCP response
func (t *MCPTool) extractContent(result interface{}) string {
	var content strings.Builder

	if resultMap, ok := result.(map[string]interface{}); ok {
		if contentArray, ok := resultMap["content"].([]interface{}); ok {
			for _, item := range contentArray {
				if contentItem, ok := item.(map[string]interface{}); ok {
					if text, ok := contentItem["text"].(string); ok {
						content.WriteString(text)
						content.WriteString("\n")
					}
				}
			}
		}
	}

	return content.String()
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
