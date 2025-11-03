package tools

import (
	"context"
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
)

func TestNewLocalToolSource(t *testing.T) {
	source := NewLocalToolSource("test-source")
	if source == nil {
		t.Fatal("NewLocalToolSource() returned nil")
	}

	if source.GetName() != "test-source" {
		t.Errorf("GetName() = %v, want 'test-source'", source.GetName())
	}

	if source.GetType() != "local" {
		t.Errorf("GetType() = %v, want 'local'", source.GetType())
	}
}

func TestLocalToolSource_GetName(t *testing.T) {
	tests := []struct {
		name       string
		sourceName string
		expected   string
	}{
		{
			name:       "custom name",
			sourceName: "my-tools",
			expected:   "my-tools",
		},
		{
			name:       "empty name",
			sourceName: "",
			expected:   "local",
		},
		{
			name:       "special characters",
			sourceName: "tools-v1.0",
			expected:   "tools-v1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := NewLocalToolSource(tt.sourceName)
			result := source.GetName()
			if result != tt.expected {
				t.Errorf("GetName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLocalToolSource_GetType(t *testing.T) {
	source := NewLocalToolSource("test")
	result := source.GetType()
	if result != "local" {
		t.Errorf("GetType() = %v, want 'local'", result)
	}
}

func TestLocalToolSource_RegisterTool(t *testing.T) {
	source := NewLocalToolSource("test-source")

	tool := NewTodoToolForTesting()
	err := source.RegisterTool(tool)
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	registeredTool, exists := source.GetTool("todo_write")
	if !exists {
		t.Error("Expected tool to be registered")
	}
	if registeredTool != tool {
		t.Error("Expected registered tool to match")
	}

	commandTool := NewCommandToolForTesting()
	err = source.RegisterTool(commandTool)
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	tools := source.ListTools()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	err = source.RegisterTool(tool)
	if err == nil {
		t.Error("Expected error when registering duplicate tool")
	}
}

func TestLocalToolSource_RegisterTool_WithConfig(t *testing.T) {

	toolConfigs := map[string]*config.ToolConfig{
		"todo_write": {
			Type:    "todo",
			Enabled: config.BoolPtr(true),
		},
		"execute_command": {
			Type:    "command",
			Enabled: config.BoolPtr(true),
		},
	}

	source, err := NewLocalToolSourceWithConfig(toolConfigs)
	if err != nil {
		t.Fatalf("NewLocalToolSourceWithConfig() error = %v", err)
	}
	if source == nil {
		t.Fatal("NewLocalToolSourceWithConfig() returned nil")
	}

	tools := source.ListTools()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	if !toolNames["todo_write"] {
		t.Error("Expected todo_write tool to be registered")
	}
	if !toolNames["execute_command"] {
		t.Error("Expected execute_command tool to be registered")
	}
}

func TestLocalToolSource_ListTools(t *testing.T) {
	source := NewLocalToolSource("test-source")

	tools := source.ListTools()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools initially, got %d", len(tools))
	}

	todoTool := NewTodoToolForTesting()
	commandTool := NewCommandToolForTesting()

	_ = source.RegisterTool(todoTool)
	_ = source.RegisterTool(commandTool)

	tools = source.ListTools()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("Expected non-empty description for tool %s", tool.Name)
		}
		if tool.ServerURL != "test-source" {
			t.Errorf("Expected ServerURL 'test-source' for tool %s, got %s", tool.Name, tool.ServerURL)
		}
	}
	if !toolNames["todo_write"] {
		t.Error("Expected todo_write tool to be listed")
	}
	if !toolNames["execute_command"] {
		t.Error("Expected execute_command tool to be listed")
	}
}

func TestLocalToolSource_GetTool(t *testing.T) {
	source := NewLocalToolSource("test-source")

	_, exists := source.GetTool("non-existent")
	if exists {
		t.Error("Expected false when getting non-existent tool")
	}

	tool := NewTodoToolForTesting()
	_ = source.RegisterTool(tool)

	registeredTool, exists := source.GetTool("todo_write")
	if !exists {
		t.Error("Expected true when getting existing tool")
	}
	if registeredTool != tool {
		t.Error("Expected returned tool to match registered tool")
	}

	_, exists = source.GetTool("TODO_WRITE")
	if exists {
		t.Error("Expected false when getting tool with different case")
	}
}

func TestLocalToolSource_RemoveTool(t *testing.T) {
	source := NewLocalToolSource("test-source")

	err := source.RemoveTool("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent tool")
	}

	tool := NewTodoToolForTesting()
	_ = source.RegisterTool(tool)

	_, exists := source.GetTool("todo_write")
	if !exists {
		t.Fatal("Expected tool to be registered")
	}

	err = source.RemoveTool("todo_write")
	if err != nil {
		t.Fatalf("RemoveTool() error = %v", err)
	}

	_, exists = source.GetTool("todo_write")
	if exists {
		t.Error("Expected tool to be removed")
	}

	tools := source.ListTools()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools after removal, got %d", len(tools))
	}
}

func TestLocalToolSource_DiscoverTools(t *testing.T) {
	source := NewLocalToolSource("test-source")

	ctx := context.Background()
	err := source.DiscoverTools(ctx)
	if err != nil {
		t.Errorf("DiscoverTools() error = %v", err)
	}

	tools := source.ListTools()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools after discovery, got %d", len(tools))
	}
}

func TestLocalToolSource_Concurrency(t *testing.T) {
	source := NewLocalToolSource("test-source")

	done := make(chan bool, 2)

	go func() {
		tool := NewTodoToolForTesting()
		_ = source.RegisterTool(tool)
		done <- true
	}()

	go func() {
		tool := NewCommandToolForTesting()
		_ = source.RegisterTool(tool)
		done <- true
	}()

	<-done
	<-done

	tools := source.ListTools()
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools after concurrent registration, got %d", len(tools))
	}
}

func TestLocalToolSource_WithEmptyConfig(t *testing.T) {

	emptyConfig := map[string]*config.ToolConfig{}

	source, err := NewLocalToolSourceWithConfig(emptyConfig)
	if err != nil {
		t.Fatalf("NewLocalToolSourceWithConfig() with empty config error = %v", err)
	}
	if source == nil {
		t.Fatal("NewLocalToolSourceWithConfig() with empty config returned nil")
	}

	tools := source.ListTools()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools with empty config, got %d", len(tools))
	}
}

func TestLocalToolSource_WithDisabledTools(t *testing.T) {

	toolConfigs := map[string]*config.ToolConfig{
		"todo_write": {
			Type:    "todo",
			Enabled: config.BoolPtr(false),
		},
		"execute_command": {
			Type:    "command",
			Enabled: config.BoolPtr(true),
		},
	}

	source, err := NewLocalToolSourceWithConfig(toolConfigs)
	if err != nil {
		t.Fatalf("NewLocalToolSourceWithConfig() error = %v", err)
	}

	tools := source.ListTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 enabled tool, got %d", len(tools))
	}

	if tools[0].Name != "execute_command" {
		t.Errorf("Expected enabled tool 'execute_command', got %s", tools[0].Name)
	}
}
