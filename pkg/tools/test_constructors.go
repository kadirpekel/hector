package tools

import (
	"context"
	"net/http"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
)

func NewCommandToolForTesting() *CommandTool {
	return NewCommandTool(&config.CommandToolsConfig{
		AllowedCommands:  []string{"echo", "pwd", "ls", "cat", "head", "tail"},
		MaxExecutionTime: 1 * time.Second,
		EnableSandboxing: false,
		WorkingDirectory: "./",
	})
}

func NewCommandToolForTestingWithCommands(allowedCommands []string) *CommandTool {
	return NewCommandTool(&config.CommandToolsConfig{
		AllowedCommands:  allowedCommands,
		MaxExecutionTime: 1 * time.Second,
		EnableSandboxing: true,
		WorkingDirectory: "./",
	})
}

func NewTodoToolForTesting() *TodoTool {
	return NewTodoTool()
}

func NewFileWriterToolForTesting() *FileWriterTool {
	return NewFileWriterTool(&config.FileWriterConfig{
		MaxFileSize:       1024,
		AllowedExtensions: []string{".txt", ".md", ".go", ".json"},
		BackupOnOverwrite: false,
		WorkingDirectory:  "./test-temp",
	})
}

func NewSearchReplaceToolForTesting() *SearchReplaceTool {
	return NewSearchReplaceTool(&config.SearchReplaceConfig{
		MaxReplacements:  10,
		ShowDiff:         true,
		CreateBackup:     false,
		WorkingDirectory: "./test-temp",
	})
}

func NewSearchToolForTesting() *SearchTool {
	return NewSearchTool(&config.SearchToolConfig{
		DocumentStores:     []string{"test-store"},
		DefaultLimit:       5,
		MaxLimit:           10,
		EnabledSearchTypes: []string{"content", "file", "function", "struct"},
	})
}

func NewMCPToolSourceForTesting(name, url string) *MCPToolSource {
	return &MCPToolSource{
		name:        name,
		url:         url,
		description: "Test MCP source",
		httpClient: httpclient.New(
			httpclient.WithHTTPClient(&http.Client{
				Timeout: 1 * time.Second,
			}),
			httpclient.WithMaxRetries(1),
		),
		tools: make(map[string]Tool),
	}
}

func NewLocalToolSourceForTesting() *LocalToolSource {
	source := NewLocalToolSource("test-local")

	todoTool := NewTodoToolForTesting()
	_ = source.RegisterTool(todoTool)

	return source
}

func NewToolRegistryForTesting() *ToolRegistry {
	registry := NewToolRegistry()

	todoTool := NewTodoToolForTesting()
	_ = registry.BaseRegistry.Register("todo_write", ToolEntry{
		Tool:       todoTool,
		Source:     &TestToolSource{name: "test-local"},
		SourceType: "local",
		Name:       "todo_write",
	})

	return registry
}

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

func CreateMockMCPResponse(tools []map[string]interface{}) string {

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
