package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
)

const (
	// DefaultMCPSSEResponseTimeout is the default timeout for reading SSE responses from MCP servers
	// Set to 5 minutes to accommodate long-running operations like document parsing with OCR
	DefaultMCPSSEResponseTimeout = 5 * time.Minute
)

type MCPToolSource struct {
	name        string
	url         string
	description string
	httpClient  *httpclient.Client
	tools       map[string]Tool
	mu          sync.RWMutex
	sessionID   string        // Session ID for streamable-http transport
	sessionMu   sync.RWMutex  // Separate mutex for sessionID to avoid deadlock
	ssTimeout   time.Duration // Timeout for SSE response reading
	internal    bool          // If true, tools from this source are not visible to agents
}

type MCPTool struct {
	toolInfo ToolInfo
	source   *MCPToolSource
}

type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// MCPToolSourceBuilder provides a fluent API for building MCP tool sources
type MCPToolSourceBuilder struct {
	name               string
	url                string
	description        string
	insecureSkipVerify *bool
	caCertificate      string
	ssTimeout          time.Duration
	internal           bool
}

// NewMCPToolSource creates a new MCP tool source builder
func NewMCPToolSource(name, url, description string) *MCPToolSourceBuilder {
	if name == "" {
		name = "mcp"
	}
	return &MCPToolSourceBuilder{
		name:        name,
		url:         url,
		description: description,
		ssTimeout:   DefaultMCPSSEResponseTimeout,
		internal:    false,
	}
}

// WithInsecureSkipVerify sets TLS certificate verification
func (b *MCPToolSourceBuilder) WithInsecureSkipVerify(skip bool) *MCPToolSourceBuilder {
	b.insecureSkipVerify = &skip
	return b
}

// WithCACertificate sets the CA certificate path
func (b *MCPToolSourceBuilder) WithCACertificate(path string) *MCPToolSourceBuilder {
	b.caCertificate = path
	return b
}

// WithTimeout sets the SSE response timeout
func (b *MCPToolSourceBuilder) WithTimeout(timeout time.Duration) *MCPToolSourceBuilder {
	b.ssTimeout = timeout
	return b
}

// WithInternal marks the source as internal (not visible to agents)
func (b *MCPToolSourceBuilder) WithInternal(internal bool) *MCPToolSourceBuilder {
	b.internal = internal
	return b
}

// Build creates the MCPToolSource
func (b *MCPToolSourceBuilder) Build() *MCPToolSource {
	// Builder sets default timeout in NewMCPToolSource, so ssTimeout should never be 0
	// But we keep this check for safety in case builder is modified in the future
	ssTimeout := b.ssTimeout
	if ssTimeout == 0 {
		ssTimeout = DefaultMCPSSEResponseTimeout
	}

	// Configure TLS using centralized function
	tlsConfig := &httpclient.TLSConfig{}
	if b.insecureSkipVerify != nil {
		tlsConfig.InsecureSkipVerify = *b.insecureSkipVerify
	}
	if b.caCertificate != "" {
		tlsConfig.CACertificate = b.caCertificate
	}

	transport, err := httpclient.ConfigureTLS(tlsConfig)
	if err != nil {
		fmt.Printf("Warning: Failed to configure TLS for MCP server %s: %v\n", b.name, err)
		// Fallback to default transport
		transport = &http.Transport{}
	}

	// Show warning if insecure skip verify is enabled
	if b.insecureSkipVerify != nil && *b.insecureSkipVerify {
		fmt.Printf("Warning: TLS certificate verification disabled for MCP server %s (insecure_skip_verify=true)\n", b.name)
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	return &MCPToolSource{
		name:        b.name,
		url:         b.url,
		description: b.description,
		httpClient: httpclient.New(
			httpclient.WithHTTPClient(httpClient),
			httpclient.WithMaxRetries(3),
			httpclient.WithBaseDelay(2*time.Second),
		),
		tools:     make(map[string]Tool),
		ssTimeout: ssTimeout,
		internal:  b.internal,
	}
}

func NewMCPToolSourceWithConfig(toolConfig *config.ToolConfig) (*MCPToolSource, error) {
	if toolConfig.ServerURL == "" {
		return nil, fmt.Errorf("server_url is required for MCP source")
	}

	// Parse timeout from config if provided
	ssTimeout := DefaultMCPSSEResponseTimeout
	if toolConfig.Timeout != "" {
		parsedTimeout, err := time.ParseDuration(toolConfig.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout for MCP source: %w", err)
		}
		ssTimeout = parsedTimeout
	}

	// Check if source is marked as internal
	internal := false
	if toolConfig.Internal != nil {
		internal = *toolConfig.Internal
	}

	builder := NewMCPToolSource("mcp", toolConfig.ServerURL, toolConfig.Description)
	if toolConfig.InsecureSkipVerify != nil {
		builder = builder.WithInsecureSkipVerify(*toolConfig.InsecureSkipVerify)
	}
	if toolConfig.CACertificate != "" {
		builder = builder.WithCACertificate(toolConfig.CACertificate)
	}
	builder = builder.WithTimeout(ssTimeout)
	builder = builder.WithInternal(internal)
	return builder.Build(), nil
}

func (r *MCPToolSource) GetName() string {
	return r.name
}

func (r *MCPToolSource) GetType() string {
	return "mcp"
}

func (r *MCPToolSource) DiscoverTools(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]Tool)

	if r.url == "" {
		return fmt.Errorf("MCP server URL not configured for source %s", r.name)
	}

	slog.Info("Discovering tools from MCP server", "source", r.name, "url", r.url)

	tools, err := r.discoverToolsFromServer(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover tools from %s: %w", r.name, err)
	}

	for _, toolInfo := range tools {
		tool := &MCPTool{
			toolInfo: toolInfo,
			source:   r,
		}
		r.tools[toolInfo.Name] = tool
	}

	var toolNames []string
	for name := range r.tools {
		toolNames = append(toolNames, name)
	}
	if len(toolNames) > 0 {
		slog.Info("MCP source discovered tools",
			"source", r.name,
			"count", len(r.tools),
			"tools", toolNames)
	} else {
		slog.Warn("MCP source discovered 0 tools", "source", r.name)
	}
	return nil
}

func (r *MCPToolSource) ListTools() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []ToolInfo
	for _, tool := range r.tools {
		info := tool.GetInfo()

		info.ServerURL = r.name
		tools = append(tools, info)
	}

	return tools
}

// ListMCPToolNames returns a list of available MCP tool names
// This is used for debugging when tools are not found
func (r *MCPToolSource) ListMCPToolNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var toolNames []string
	for name := range r.tools {
		toolNames = append(toolNames, name)
	}
	return toolNames
}

func (r *MCPToolSource) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

func (r *MCPToolSource) discoverToolsFromServer(ctx context.Context) ([]ToolInfo, error) {
	// First, try to initialize the session if needed
	// Some MCP servers require initialization before other calls
	initResponse, initErr := r.makeRequest(ctx, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "hector",
			"version": "1.0.0",
		},
	})
	if initErr != nil {
		slog.Debug("MCP initialize failed (non-fatal)", "source", r.name, "error", initErr.Error())
	} else if initResponse != nil && initResponse.Error != nil {
		slog.Debug("MCP initialize returned error (non-fatal)", "source", r.name, "error", initResponse.Error.Message)
	}

	response, err := r.makeRequest(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		slog.Debug("MCP tools/list request failed", "source", r.name, "error", err.Error())
		return nil, err
	}

	if response.Error != nil {
		slog.Debug("MCP tools/list returned error", "source", r.name, "error_code", response.Error.Code, "error_message", response.Error.Message)
		return nil, fmt.Errorf("MCP error: %s", response.Error.Message)
	}

	// Debug: log the response structure
	if resultMap, ok := response.Result.(map[string]interface{}); ok {
		if toolsArray, ok := resultMap["tools"].([]interface{}); ok {
			var toolNames []string
			for _, toolItem := range toolsArray {
				if tool, ok := toolItem.(map[string]interface{}); ok {
					if name, ok := tool["name"].(string); ok {
						toolNames = append(toolNames, name)
					}
				}
			}
			slog.Debug("MCP tools/list response", "source", r.name, "tool_count", len(toolNames), "tools", toolNames)
		} else {
			slog.Debug("MCP tools/list response structure", "source", r.name, "result_keys", getMapKeys(resultMap))
		}
	}

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

					if params, ok := tool["inputSchema"].(map[string]interface{}); ok {
						if properties, ok := params["properties"].(map[string]interface{}); ok {
							for paramName, paramData := range properties {
								if param, ok := paramData.(map[string]interface{}); ok {
									paramType := getString(param, "type")
									// Skip parameters without a valid type
									if paramType == "" {
										continue
									}

									toolParam := ToolParameter{
										Name:        paramName,
										Type:        paramType,
										Description: getString(param, "description"),
										Required:    isRequired(params, paramName),
									}

									if enum, ok := param["enum"].([]interface{}); ok {
										for _, val := range enum {
											if strVal, ok := val.(string); ok && strVal != "" {
												toolParam.Enum = append(toolParam.Enum, strVal)
											}
										}
									}

									if defaultVal, ok := param["default"]; ok {
										toolParam.Default = defaultVal
									}

									if examples, ok := param["examples"].([]interface{}); ok {

										if len(examples) > 0 && !strings.Contains(toolParam.Description, "Example") {
											toolParam.Description += "\nExamples:"
											for _, ex := range examples {
												toolParam.Description += fmt.Sprintf("\n  %v", ex)
											}
										}
									}

									if format := getString(param, "format"); format != "" {
										toolParam.Description += fmt.Sprintf(" (format: %s)", format)
									}

									if pattern := getString(param, "pattern"); pattern != "" {
										toolParam.Description += fmt.Sprintf(" (pattern: %s)", pattern)
									}

									// Extract items schema for array types (required by OpenAI)
									if toolParam.Type == "array" {
										if items, ok := param["items"]; ok && items != nil {
											// items can be a map (object schema) or a simple type string
											if itemsMap, ok := items.(map[string]interface{}); ok {
												// Validate that the schema has a type
												if itemType := getString(itemsMap, "type"); itemType != "" {
													toolParam.Items = itemsMap
												} else {
													// Invalid schema without type, default to string
													toolParam.Items = map[string]interface{}{
														"type": "string",
													}
												}
											} else if itemTypeStr, ok := items.(string); ok && itemTypeStr != "" {
												// If items is a simple type string, wrap it in a schema
												toolParam.Items = map[string]interface{}{
													"type": itemTypeStr,
												}
											} else {
												// Invalid items value, default to string
												toolParam.Items = map[string]interface{}{
													"type": "string",
												}
											}
										} else {
											// Items not specified - OpenAI requires it, so default to string array
											toolParam.Items = map[string]interface{}{
												"type": "string",
											}
										}
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

func (r *MCPToolSource) makeRequest(ctx context.Context, method string, params interface{}) (*Response, error) {

	request := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.url, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	// Add session ID if we have one (for streamable-http transport)
	r.sessionMu.RLock()
	sessionID := r.sessionID
	r.sessionMu.RUnlock()
	if sessionID != "" {
		req.Header.Set("mcp-session-id", sessionID)
	}

	httpResp, err := r.httpClient.Do(req)

	if err != nil {
		slog.Debug("MCP HTTP request failed",
			"source", r.name,
			"url", r.url,
			"method", method,
			"error", err.Error())
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer httpResp.Body.Close()

	slog.Debug("MCP HTTP request completed",
		"source", r.name,
		"url", r.url,
		"method", method,
		"status_code", httpResp.StatusCode,
		"content_type", httpResp.Header.Get("Content-Type"))

	// Extract session ID from response header (for streamable-http transport)
	if sessionID := httpResp.Header.Get("mcp-session-id"); sessionID != "" {
		r.sessionMu.Lock()
		r.sessionID = sessionID
		r.sessionMu.Unlock()
	}

	if httpResp.StatusCode != http.StatusOK {
		// Try to read error response body for better error message
		responseBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s (response: %s)", httpResp.StatusCode, httpResp.Status, string(responseBody))
	}

	// Check if response is SSE (Server-Sent Events)
	contentType := httpResp.Header.Get("Content-Type")

	if strings.Contains(contentType, "text/event-stream") {
		// Read SSE stream until we get first complete message
		// Server may wait up to 30s (batchTimeout) before closing, so we use timeout
		type result struct {
			response *Response
			err      error
		}
		resultChan := make(chan result, 1)

		go func() {
			defer httpResp.Body.Close()

			// Use bufio.NewReader with ReadBytes instead of Scanner for better handling of large lines
			// ReadBytes reads until delimiter (no fixed buffer limit), making it more suitable for
			// large tool results (web search results, etc.) compared to Scanner's default 64KB limit
			reader := bufio.NewReader(httpResp.Body)

			var currentData strings.Builder

			for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					if err == io.EOF {
						break
					}
					slog.Debug("MCP SSE read error", "source", r.name, "error", err)
					break
				}

				lineStr := strings.TrimSpace(string(line))

				// Empty line signals end of event
				if lineStr == "" {
					if currentData.Len() > 0 {
						jsonData := currentData.String()
						dataPreview := jsonData
						if len(dataPreview) > 200 {
							dataPreview = dataPreview[:200] + "..."
						}
						slog.Debug("MCP SSE data received",
							"source", r.name,
							"data_length", len(jsonData),
							"data_preview", dataPreview)

						var mcpResp Response
						if parseErr := json.Unmarshal([]byte(jsonData), &mcpResp); parseErr == nil {
							resultChan <- result{response: &mcpResp}
							return
						} else {
							slog.Debug("MCP SSE JSON parse failed",
								"source", r.name,
								"error", parseErr.Error(),
								"data_preview", dataPreview)
						}

						// Reset for next event
						currentData.Reset()
					}
					continue
				}

				// Parse SSE field - we only care about data lines
				if strings.HasPrefix(lineStr, "data:") {
					data := strings.TrimSpace(strings.TrimPrefix(lineStr, "data:"))
					currentData.WriteString(data)
				}
				// Ignore event type lines and other SSE fields
			}

			// Handle any remaining data when stream ends
			if currentData.Len() > 0 {
				jsonData := currentData.String()
				var mcpResp Response
				if parseErr := json.Unmarshal([]byte(jsonData), &mcpResp); parseErr == nil {
					resultChan <- result{response: &mcpResp}
					return
				}
			}

			// If we exit the loop without finding data, it's an error
			resultChan <- result{err: fmt.Errorf("SSE stream ended without complete message")}
		}()

		// Wait for result with timeout
		// Use configurable timeout (default 5 minutes for document parsing operations)
		// Builder ensures timeout is set, but check for safety if source created directly
		timeout := r.ssTimeout
		if timeout == 0 {
			timeout = DefaultMCPSSEResponseTimeout
		}
		select {
		case res := <-resultChan:
			if res.err != nil {
				return nil, res.err
			}
			return res.response, nil
		case <-time.After(timeout):
			return nil, fmt.Errorf("timeout reading SSE response after %v", timeout)
		}
	}

	// Regular JSON response
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var mcpResp Response
	if err := json.Unmarshal(responseBody, &mcpResp); err == nil {
		return &mcpResp, nil
	}

	return nil, fmt.Errorf("failed to parse response as JSON")
}

func (t *MCPTool) GetInfo() ToolInfo {
	return t.toolInfo
}

func (t *MCPTool) GetName() string {
	return t.toolInfo.Name
}

func (t *MCPTool) GetDescription() string {
	return t.toolInfo.Description
}

func (t *MCPTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	// Log tool execution start
	slog.Debug("MCP tool execution started",
		"tool", t.toolInfo.Name,
		"source", t.source.name,
		"server_url", t.source.url)

	// Validate required parameters
	if err := t.validateParameters(args); err != nil {
		slog.Debug("MCP tool parameter validation failed",
			"tool", t.toolInfo.Name,
			"error", err.Error())
		return buildMCPErrorResult(t.toolInfo.Name, err.Error(), time.Since(start), t.source.name, t.source.url), err
	}

	params := CallParams{
		Name:      t.toolInfo.Name,
		Arguments: args,
	}

	response, err := t.source.makeRequest(ctx, "tools/call", params)
	if err != nil {
		slog.Debug("MCP tool request failed",
			"tool", t.toolInfo.Name,
			"source", t.source.name,
			"error", err.Error())
		return buildMCPErrorResult(t.toolInfo.Name, err.Error(), time.Since(start), t.source.name, t.source.url), err
	}

	if response.Error != nil {
		errorMsg := response.Error.Message
		if errorMsg == "" {
			errorMsg = fmt.Sprintf("MCP protocol error (code: %d)", response.Error.Code)
		}
		err := fmt.Errorf("MCP error: %s", errorMsg)
		slog.Debug("MCP tool protocol error",
			"tool", t.toolInfo.Name,
			"source", t.source.name,
			"error_code", response.Error.Code,
			"error_message", errorMsg)
		return buildMCPErrorResult(t.toolInfo.Name, err.Error(), time.Since(start), t.source.name, t.source.url), err
	}

	// Extract result map once and reuse
	resultMap, isMap := response.Result.(map[string]interface{})
	if isMap {
		slog.Debug("MCP tool response result structure",
			"tool", t.toolInfo.Name,
			"keys", getMapKeys(resultMap))
	}

	content := t.extractContent(response.Result)

	// Extract metadata from response if available
	var responseMetadata map[string]interface{}
	if isMap {
		if metadata, ok := resultMap["metadata"].(map[string]interface{}); ok {
			responseMetadata = metadata
		}
	}

	// Check if result contains error indicators
	hasError := false
	errorMsg := ""

	// Check for errors in response
	if isMap {
		// Check for error in metadata
		if responseMetadata != nil {
			if errStr, ok := responseMetadata["error"].(string); ok && errStr != "" {
				hasError = true
				errorMsg = errStr
			}
		}
		// Check for error field at top level
		if errStr, ok := resultMap["error"].(string); ok && errStr != "" {
			hasError = true
			if errorMsg == "" {
				errorMsg = errStr
			}
		}
		// Check for isError flag
		if isErr, ok := resultMap["isError"].(bool); ok && isErr {
			hasError = true
			if errorMsg == "" {
				errorMsg = "tool reported error"
			}
		}
	}

	// Check if content itself is an error message
	contentTrimmed := strings.TrimSpace(content)
	if isErrorContent(contentTrimmed) {
		hasError = true
		if errorMsg == "" {
			errorMsg = contentTrimmed
		}
	}

	// If error detected, return failure
	if hasError {
		// Ensure we have a non-empty error message
		if errorMsg == "" {
			errorMsg = "tool reported error"
		}
		err := fmt.Errorf("MCP tool error: %s", errorMsg)
		contentPreview := contentTrimmed
		if len(contentPreview) > 100 {
			contentPreview = contentPreview[:100] + "..."
		}
		duration := time.Since(start)
		slog.Debug("MCP tool execution failed",
			"tool", t.toolInfo.Name,
			"source", t.source.name,
			"error", errorMsg,
			"duration_ms", duration.Milliseconds(),
			"content_preview", contentPreview)
		return buildMCPErrorResult(t.toolInfo.Name, err.Error(), duration, t.source.name, t.source.url), err
	}

	// Success - build result with metadata
	duration := time.Since(start)
	contentLength := len(contentTrimmed)
	slog.Debug("MCP tool execution succeeded",
		"tool", t.toolInfo.Name,
		"source", t.source.name,
		"duration_ms", duration.Milliseconds(),
		"content_length", contentLength,
		"has_metadata", len(responseMetadata) > 0)
	return buildMCPSuccessResult(t.toolInfo.Name, contentTrimmed, duration, t.source.name, t.source.url, responseMetadata), nil
}

// validateParameters checks if all required parameters are provided
func (t *MCPTool) validateParameters(args map[string]interface{}) error {
	// Get required parameters from tool info
	var missingParams []string
	for _, param := range t.toolInfo.Parameters {
		if param.Required {
			if _, exists := args[param.Name]; !exists {
				missingParams = append(missingParams, param.Name)
			}
		}
	}

	if len(missingParams) > 0 {
		return fmt.Errorf("missing required parameters: %v", missingParams)
	}

	return nil
}

func (t *MCPTool) extractContent(result interface{}) string {
	var content strings.Builder

	if resultMap, ok := result.(map[string]interface{}); ok {
		// Try multiple content extraction strategies
		// Strategy 1: content array with text items (standard MCP format)
		if contentArray, ok := resultMap["content"].([]interface{}); ok {
			for _, item := range contentArray {
				if contentItem, ok := item.(map[string]interface{}); ok {
					if text, ok := contentItem["text"].(string); ok {
						content.WriteString(text)
						content.WriteString("\n")
					}
				} else if text, ok := item.(string); ok {
					// Handle case where content array contains strings directly
					content.WriteString(text)
					content.WriteString("\n")
				}
			}
		}
		// Strategy 2: direct text/content field
		if content.String() == "" {
			if text, ok := resultMap["text"].(string); ok {
				content.WriteString(text)
			} else if text, ok := resultMap["content"].(string); ok {
				content.WriteString(text)
			}
		}
		// Strategy 3: isError field indicates failure
		if isError, ok := resultMap["isError"].(bool); ok && isError {
			slog.Debug("MCP tool response indicates error via isError field",
				"tool", t.toolInfo.Name)
		}
	}

	extractedContent := content.String()
	if extractedContent == "" {
		slog.Debug("MCP tool extractContent returned empty string",
			"tool", t.toolInfo.Name,
			"result_type", fmt.Sprintf("%T", result))
	}
	return extractedContent
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// isErrorContent checks if content matches common error message patterns
// This is used to detect errors that might be returned as content strings
func isErrorContent(content string) bool {
	if content == "" {
		return false
	}
	contentLower := strings.ToLower(strings.TrimSpace(content))
	return strings.HasPrefix(contentLower, "error executing tool") ||
		strings.HasPrefix(contentLower, "error:") ||
		strings.HasPrefix(contentLower, "tool error:")
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
