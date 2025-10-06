package tools

import (
	"context"
	"testing"

	"github.com/kadirpekel/hector/pkg/config"
)

func TestNewToolRegistryForTesting(t *testing.T) {
	registry := NewToolRegistryForTesting()
	if registry == nil {
		t.Fatal("NewToolRegistryForTesting() returned nil")
	}

	// Test that registry is initialized with test tools
	tools := registry.BaseRegistry.List()
	if len(tools) == 0 {
		t.Error("Expected at least one test tool")
	}
}

func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()

	// Create a test tool
	tool := NewTodoToolForTesting()
	entry := ToolEntry{
		Tool:       tool,
		Source:     &TestToolSource{name: "test-source"},
		SourceType: "test",
		Name:       "test-tool",
	}

	err := registry.BaseRegistry.Register("test-tool", entry)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Verify tool was registered
	registeredEntry, exists := registry.BaseRegistry.Get("test-tool")
	if !exists {
		t.Error("Expected tool to be registered")
	}
	if registeredEntry.Tool != tool {
		t.Error("Expected registered tool to match")
	}
}

func TestToolRegistry_Register_Duplicate(t *testing.T) {
	registry := NewToolRegistry()

	tool := NewTodoToolForTesting()
	entry := ToolEntry{
		Tool:       tool,
		Source:     &TestToolSource{name: "test-source"},
		SourceType: "test",
		Name:       "test-tool",
	}

	// Register first time
	err := registry.BaseRegistry.Register("test-tool", entry)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Try to register again
	err = registry.BaseRegistry.Register("test-tool", entry)
	if err == nil {
		t.Error("Expected error when registering duplicate tool")
	}
}

func TestToolRegistry_Get(t *testing.T) {
	registry := NewToolRegistry()

	tool := NewTodoToolForTesting()
	entry := ToolEntry{
		Tool:       tool,
		Source:     &TestToolSource{name: "test-source"},
		SourceType: "test",
		Name:       "test-tool",
	}

	err := registry.BaseRegistry.Register("test-tool", entry)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Get the tool
	registeredEntry, exists := registry.BaseRegistry.Get("test-tool")
	if !exists {
		t.Fatal("Get() should return true for existing tool")
	}

	if registeredEntry.Tool == nil {
		t.Fatal("Get() returned nil tool")
	}

	if registeredEntry.Tool.GetName() != "todo_write" {
		t.Errorf("Get() tool name = %v, want 'todo_write'", registeredEntry.Tool.GetName())
	}
}

func TestToolRegistry_Get_NotFound(t *testing.T) {
	registry := NewToolRegistry()

	_, exists := registry.BaseRegistry.Get("non-existent-tool")
	if exists {
		t.Error("Expected false when getting non-existent tool")
	}
}

func TestToolRegistry_List(t *testing.T) {
	registry := NewToolRegistry()

	// Initially should be empty
	tools := registry.BaseRegistry.List()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools initially, got %d", len(tools))
	}

	// Register a tool
	tool := NewTodoToolForTesting()
	entry := ToolEntry{
		Tool:       tool,
		Source:     &TestToolSource{name: "test-source"},
		SourceType: "test",
		Name:       "test-tool",
	}

	err := registry.BaseRegistry.Register("test-tool", entry)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Should now have one tool
	tools = registry.BaseRegistry.List()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}
}

func TestToolRegistry_Remove(t *testing.T) {
	registry := NewToolRegistry()

	// Register a tool
	tool := NewTodoToolForTesting()
	entry := ToolEntry{
		Tool:       tool,
		Source:     &TestToolSource{name: "test-source"},
		SourceType: "test",
		Name:       "test-tool",
	}

	err := registry.BaseRegistry.Register("test-tool", entry)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Remove the tool
	err = registry.BaseRegistry.Remove("test-tool")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Verify tool was removed
	_, exists := registry.BaseRegistry.Get("test-tool")
	if exists {
		t.Error("Expected tool to be removed")
	}
}

func TestToolRegistry_Remove_NotFound(t *testing.T) {
	registry := NewToolRegistry()

	err := registry.BaseRegistry.Remove("non-existent-tool")
	if err == nil {
		t.Error("Expected error when removing non-existent tool")
	}
}

func TestToolRegistry_Count(t *testing.T) {
	registry := NewToolRegistry()

	// Initially should be 0
	count := registry.BaseRegistry.Count()
	if count != 0 {
		t.Errorf("Expected count 0 initially, got %d", count)
	}

	// Register tools
	tool1 := NewTodoToolForTesting()
	entry1 := ToolEntry{
		Tool:       tool1,
		Source:     &TestToolSource{name: "test-source"},
		SourceType: "test",
		Name:       "tool1",
	}

	tool2 := NewTodoToolForTesting()
	entry2 := ToolEntry{
		Tool:       tool2,
		Source:     &TestToolSource{name: "test-source"},
		SourceType: "test",
		Name:       "tool2",
	}

	registry.BaseRegistry.Register("tool1", entry1)
	registry.BaseRegistry.Register("tool2", entry2)

	// Should now be 2
	count = registry.BaseRegistry.Count()
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestToolRegistry_Clear(t *testing.T) {
	registry := NewToolRegistry()

	// Register a tool
	tool := NewTodoToolForTesting()
	entry := ToolEntry{
		Tool:       tool,
		Source:     &TestToolSource{name: "test-source"},
		SourceType: "test",
		Name:       "test-tool",
	}

	registry.BaseRegistry.Register("test-tool", entry)

	// Clear the registry
	registry.BaseRegistry.Clear()

	// Should be empty
	count := registry.BaseRegistry.Count()
	if count != 0 {
		t.Errorf("Expected count 0 after clear, got %d", count)
	}
}

func TestNewToolRegistryWithConfig(t *testing.T) {
	config := &config.ToolConfigs{
		Tools: map[string]config.ToolConfig{
			"test-tool": {
				Type:    "todo",
				Enabled: true,
			},
		},
	}

	registry, err := NewToolRegistryWithConfig(config)
	if err != nil {
		t.Fatalf("NewToolRegistryWithConfig() error = %v", err)
	}
	if registry == nil {
		t.Fatal("NewToolRegistryWithConfig() returned nil")
	}
}

// LocalToolSource tests moved to local_test.go for better organization

func TestTestToolSource(t *testing.T) {
	source := NewTestToolSource("test-source")

	if source.GetName() != "test-source" {
		t.Errorf("GetName() = %v, want 'test-source'", source.GetName())
	}

	if source.GetType() != "test" {
		t.Errorf("GetType() = %v, want 'test'", source.GetType())
	}

	// Test DiscoverTools
	ctx := context.Background()
	err := source.DiscoverTools(ctx)
	if err != nil {
		t.Errorf("DiscoverTools() error = %v", err)
	}

	// Test ListTools (should be empty initially)
	tools := source.ListTools()
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools initially, got %d", len(tools))
	}

	// Register a tool
	tool := NewTodoToolForTesting()
	source.RegisterTool(tool)

	// Should now have one tool
	tools = source.ListTools()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	// Test GetTool
	registeredTool, exists := source.GetTool("todo_write")
	if !exists {
		t.Error("Expected tool to be found")
	}
	if registeredTool != tool {
		t.Error("Expected returned tool to match")
	}
}
