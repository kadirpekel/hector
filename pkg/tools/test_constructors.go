package tools

import (
	"context"
	"net/http"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
)

// ============================================================================
// TEST-FRIENDLY CONSTRUCTORS
// These provide simple ways to create tools for testing without complex configuration
// ============================================================================

// NewCommandToolForTesting creates a command tool with test-friendly defaults
func NewCommandToolForTesting() *CommandTool {
	return NewCommandTool(&config.CommandToolsConfig{
		AllowedCommands:  []string{"echo", "pwd", "ls", "cat", "head", "tail"},
		MaxExecutionTime: 1 * time.Second, // Short timeout for tests
		EnableSandboxing: false,           // Disable for testing
		WorkingDirectory: "./",
	})
}

// NewCommandToolForTestingWithCommands creates a command tool with custom allowed commands
func NewCommandToolForTestingWithCommands(allowedCommands []string) *CommandTool {
	return NewCommandTool(&config.CommandToolsConfig{
		AllowedCommands:  allowedCommands,
		MaxExecutionTime: 1 * time.Second,
		EnableSandboxing: true, // Enable sandboxing with custom commands
		WorkingDirectory: "./",
	})
}

// NewTodoToolForTesting creates a todo tool (already simple enough)
func NewTodoToolForTesting() *TodoTool {
	return NewTodoTool()
}

// NewFileWriterToolForTesting creates a file writer tool with test-friendly defaults
func NewFileWriterToolForTesting() *FileWriterTool {
	return NewFileWriterTool(&config.FileWriterConfig{
		MaxFileSize:       1024, // Small size for tests
		AllowedExtensions: []string{".txt", ".md", ".go", ".json"},
		BackupOnOverwrite: false, // Disable for testing
		WorkingDirectory:  "./test-temp",
	})
}

// NewSearchReplaceToolForTesting creates a search/replace tool with test-friendly defaults
func NewSearchReplaceToolForTesting() *SearchReplaceTool {
	return NewSearchReplaceTool(&config.SearchReplaceConfig{
		MaxReplacements:  10, // Small limit for tests
		ShowDiff:         true,
		CreateBackup:     false, // Disable for testing
		WorkingDirectory: "./test-temp",
	})
}

// NewSearchToolForTesting creates a search tool with test-friendly defaults
func NewSearchToolForTesting() *SearchTool {
	return NewSearchTool(&config.SearchToolConfig{
		DocumentStores:     []string{"test-store"}, // Mock store for testing
		DefaultLimit:       5,                      // Small limit for tests
		MaxLimit:           10,                     // Small max limit for tests
		EnabledSearchTypes: []string{"content", "file", "function", "struct"},
	})
}

// NewMCPToolSourceForTesting creates an MCP tool source for testing
// Note: This creates a real MCPToolSource but with a test URL
func NewMCPToolSourceForTesting(name, url string) *MCPToolSource {
	return &MCPToolSource{
		name:        name,
		url:         url,
		description: "Test MCP source",
		httpClient: httpclient.New(
			httpclient.WithHTTPClient(&http.Client{
				Timeout: 1 * time.Second, // Short timeout for tests
			}),
			httpclient.WithMaxRetries(1), // Minimal retries for tests
		),
		tools: make(map[string]Tool),
	}
}

// NewLocalToolSourceForTesting creates a local tool source with test tools
func NewLocalToolSourceForTesting() *LocalToolSource {
	source := NewLocalToolSource("test-local")

	// Register some test tools
	todoTool := NewTodoToolForTesting()
	source.RegisterTool(todoTool)

	return source
}

// NewToolRegistryForTesting creates a tool registry with test tools
func NewToolRegistryForTesting() *ToolRegistry {
	registry := NewToolRegistry()

	// Register test tools
	todoTool := NewTodoToolForTesting()
	registry.Register("todo_write", ToolEntry{
		Tool:       todoTool,
		Source:     &TestToolSource{name: "test-local"},
		SourceType: "local",
		Name:       "todo_write",
	})

	return registry
}

// ============================================================================
// TEST UTILITIES AND MOCKS
// ============================================================================

// TestToolSource is a simple tool source for testing
type TestToolSource struct {
	name  string
	tools map[string]Tool
}

func NewTestToolSource(name string) *TestToolSource {
	return &TestToolSource{
		name:  name,
		tools: make(map[string]Tool),
	}
}

func (t *TestToolSource) GetName() string {
	return t.name
}

func (t *TestToolSource) GetType() string {
	return "test"
}

func (t *TestToolSource) DiscoverTools(ctx context.Context) error {
	return nil
}

func (t *TestToolSource) ListTools() []ToolInfo {
	tools := make([]ToolInfo, 0, len(t.tools))
	for _, tool := range t.tools {
		tools = append(tools, tool.GetInfo())
	}
	return tools
}

func (t *TestToolSource) GetTool(name string) (Tool, bool) {
	tool, exists := t.tools[name]
	return tool, exists
}

func (t *TestToolSource) RegisterTool(tool Tool) {
	t.tools[tool.GetName()] = tool
}

// Test utilities for creating mock responses and validating results
func CreateMockMCPResponse(tools []map[string]interface{}) string {
	// Simple JSON marshaling for test responses
	jsonStr := `{"jsonrpc":"2.0","id":1,"result":{"tools":[`
	for i, tool := range tools {
		if i > 0 {
			jsonStr += ","
		}
		jsonStr += `{"name":"` + tool["name"].(string) + `","description":"` + tool["description"].(string) + `"}`
	}
	jsonStr += `]}}`
	return jsonStr
}
