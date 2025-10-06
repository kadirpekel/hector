package tools

import (
	"context"
	"testing"
	"time"
)

func TestNewMCPToolSource(t *testing.T) {
	source := NewMCPToolSource("test-mcp", "http://localhost:8080", "Test MCP server")
	if source == nil {
		t.Fatal("NewMCPToolSource() returned nil")
	}

	// Test that the source has the expected name
	if source.GetName() != "test-mcp" {
		t.Errorf("GetName() = %v, want 'test-mcp'", source.GetName())
	}

	// Test that the source has the expected type
	if source.GetType() != "mcp" {
		t.Errorf("GetType() = %v, want 'mcp'", source.GetType())
	}
}

func TestNewMCPToolSource_WithEmptyName(t *testing.T) {
	source := NewMCPToolSource("", "http://localhost:8080", "Test MCP server")
	if source == nil {
		t.Fatal("NewMCPToolSource() returned nil")
	}

	// Should default to "mcp"
	if source.GetName() != "mcp" {
		t.Errorf("GetName() = %v, want 'mcp'", source.GetName())
	}
}

func TestNewMCPToolSourceWithConfig(t *testing.T) {
	// Test NewMCPToolSourceWithConfig
	source, err := NewMCPToolSourceWithConfig("http://localhost:8080")
	if err != nil {
		t.Fatalf("NewMCPToolSourceWithConfig() error = %v", err)
	}
	if source == nil {
		t.Fatal("NewMCPToolSourceWithConfig() returned nil")
	}

	// Should have default name
	if source.GetName() != "mcp" {
		t.Errorf("GetName() = %v, want 'mcp'", source.GetName())
	}

	// Should have correct type
	if source.GetType() != "mcp" {
		t.Errorf("GetType() = %v, want 'mcp'", source.GetType())
	}
}

func TestNewMCPToolSourceWithConfig_EmptyURL(t *testing.T) {
	// Test with empty URL
	_, err := NewMCPToolSourceWithConfig("")
	if err == nil {
		t.Error("Expected error when URL is empty")
	}
}

func TestMCPToolSource_GetName(t *testing.T) {
	tests := []struct {
		name       string
		sourceName string
		expected   string
	}{
		{
			name:       "custom name",
			sourceName: "my-mcp-server",
			expected:   "my-mcp-server",
		},
		{
			name:       "empty name defaults to mcp",
			sourceName: "",
			expected:   "mcp",
		},
		{
			name:       "special characters",
			sourceName: "mcp-v1.0",
			expected:   "mcp-v1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewMCPToolSource(tt.sourceName, "http://localhost:8080", "Test")
			result := source.GetName()
			if result != tt.expected {
				t.Errorf("GetName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMCPToolSource_GetType(t *testing.T) {
	source := NewMCPToolSource("test", "http://localhost:8080", "Test")
	result := source.GetType()
	if result != "mcp" {
		t.Errorf("GetType() = %v, want 'mcp'", result)
	}
}

func TestMCPToolSource_ListTools(t *testing.T) {
	source := NewMCPToolSource("test-mcp", "http://localhost:8080", "Test MCP server")

	// Initially should be empty (no tools discovered yet)
	tools := source.ListTools()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools initially, got %d", len(tools))
	}
}

func TestMCPToolSource_GetTool(t *testing.T) {
	source := NewMCPToolSource("test-mcp", "http://localhost:8080", "Test MCP server")

	// Test getting non-existent tool
	_, exists := source.GetTool("non-existent")
	if exists {
		t.Error("Expected false when getting non-existent tool")
	}
}

func TestMCPToolSource_DiscoverTools_WithoutURL(t *testing.T) {
	source := NewMCPToolSource("test-mcp", "", "Test MCP server")

	// Test DiscoverTools without URL
	ctx := context.Background()
	err := source.DiscoverTools(ctx)
	if err == nil {
		t.Error("Expected error when URL is not configured")
	}
}

func TestMCPToolSource_DiscoverTools_WithInvalidURL(t *testing.T) {
	source := NewMCPToolSource("test-mcp", "http://invalid-url-that-does-not-exist:9999", "Test MCP server")

	// Test DiscoverTools with invalid URL
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := source.DiscoverTools(ctx)
	if err == nil {
		t.Error("Expected error when URL is invalid")
	}
}

func TestMCPToolSource_ForTesting(t *testing.T) {
	source := NewMCPToolSourceForTesting("test-mcp", "http://localhost:8080")
	if source == nil {
		t.Fatal("NewMCPToolSourceForTesting() returned nil")
	}

	// Test that the source has the expected name
	if source.GetName() != "test-mcp" {
		t.Errorf("GetName() = %v, want 'test-mcp'", source.GetName())
	}

	// Test that the source has the expected type
	if source.GetType() != "mcp" {
		t.Errorf("GetType() = %v, want 'mcp'", source.GetType())
	}
}

func TestMCPToolSource_RequestResponse(t *testing.T) {
	// Test MCP request/response structures
	request := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
		Params:  nil,
	}

	if request.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got %s", request.JSONRPC)
	}
	if request.ID != 1 {
		t.Errorf("Expected ID 1, got %v", request.ID)
	}
	if request.Method != "tools/list" {
		t.Errorf("Expected Method 'tools/list', got %s", request.Method)
	}

	// Test response structure
	response := Response{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]interface{}{"tools": []interface{}{}},
		Error:   nil,
	}

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got %s", response.JSONRPC)
	}
	if response.ID != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}
	if response.Error != nil {
		t.Error("Expected no error in response")
	}
}

func TestMCPToolSource_Error(t *testing.T) {
	// Test MCP error structure
	error := Error{
		Code:    -32601,
		Message: "Method not found",
	}

	if error.Code != -32601 {
		t.Errorf("Expected Code -32601, got %d", error.Code)
	}
	if error.Message != "Method not found" {
		t.Errorf("Expected Message 'Method not found', got %s", error.Message)
	}
}

func TestMCPToolSource_CallParams(t *testing.T) {
	// Test CallParams structure
	params := CallParams{
		Name:      "test_tool",
		Arguments: map[string]interface{}{"arg1": "value1"},
	}

	if params.Name != "test_tool" {
		t.Errorf("Expected Name 'test_tool', got %s", params.Name)
	}
	if len(params.Arguments) != 1 {
		t.Errorf("Expected 1 argument, got %d", len(params.Arguments))
	}
	if params.Arguments["arg1"] != "value1" {
		t.Errorf("Expected argument value 'value1', got %v", params.Arguments["arg1"])
	}
}

func TestMCPToolSource_MCPTool(t *testing.T) {
	// Test MCPTool structure
	source := NewMCPToolSource("test-mcp", "http://localhost:8080", "Test MCP server")

	toolInfo := ToolInfo{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  []ToolParameter{},
		ServerURL:   "test-mcp",
	}

	tool := &MCPTool{
		toolInfo: toolInfo,
		source:   source,
	}

	// Test MCPTool methods
	if tool.GetName() != "test_tool" {
		t.Errorf("GetName() = %v, want 'test_tool'", tool.GetName())
	}

	if tool.GetDescription() != "A test tool" {
		t.Errorf("GetDescription() = %v, want 'A test tool'", tool.GetDescription())
	}

	info := tool.GetInfo()
	if info.Name != "test_tool" {
		t.Errorf("GetInfo().Name = %v, want 'test_tool'", info.Name)
	}
	if info.Description != "A test tool" {
		t.Errorf("GetInfo().Description = %v, want 'A test tool'", info.Description)
	}
}

func TestMCPToolSource_MCPTool_Execute(t *testing.T) {
	source := NewMCPToolSource("test-mcp", "http://localhost:8080", "Test MCP server")

	toolInfo := ToolInfo{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  []ToolParameter{},
		ServerURL:   "test-mcp",
	}

	tool := &MCPTool{
		toolInfo: toolInfo,
		source:   source,
	}

	// Test Execute with invalid URL (should fail)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	args := map[string]interface{}{
		"test_arg": "test_value",
	}

	result, err := tool.Execute(ctx, args)
	if err == nil {
		t.Error("Expected error when executing tool with invalid URL")
	}
	if result.Success {
		t.Error("Expected result.Success to be false")
	}
	if result.ToolName != "test_tool" {
		t.Errorf("Expected ToolName 'test_tool', got %s", result.ToolName)
	}
}

func TestMCPToolSource_Concurrency(t *testing.T) {
	source := NewMCPToolSource("test-mcp", "http://localhost:8080", "Test MCP server")

	// Test concurrent access to ListTools and GetTool
	done := make(chan bool, 2)

	go func() {
		source.ListTools()
		done <- true
	}()

	go func() {
		source.GetTool("test")
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Should not panic or cause issues
}

func TestMCPToolSource_WithConfig_InvalidURL(t *testing.T) {
	// Test with invalid URL
	_, err := NewMCPToolSourceWithConfig("not-a-valid-url")
	if err != nil {
		t.Errorf("NewMCPToolSourceWithConfig() should not error on invalid URL format, got: %v", err)
	}
}

func TestMCPToolSource_HTTPClient(t *testing.T) {
	source := NewMCPToolSource("test-mcp", "http://localhost:8080", "Test MCP server")

	// Verify HTTP client is initialized
	if source.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}

func TestMCPToolSource_ForTesting_HTTPClient(t *testing.T) {
	source := NewMCPToolSourceForTesting("test-mcp", "http://localhost:8080")

	// Verify HTTP client is initialized with test-friendly settings
	if source.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}
