package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// LOCAL - BUILT-IN TOOL SOURCE
// ============================================================================

// LocalToolSource manages built-in/local tools
type LocalToolSource struct {
	name  string
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewLocalToolSource creates a new local tool source
func NewLocalToolSource(name string) *LocalToolSource {
	if name == "" {
		name = "local"
	}

	return &LocalToolSource{
		name:  name,
		tools: make(map[string]Tool),
	}
}

// NewLocalToolSourceWithConfig creates a new local tool source from configuration
func NewLocalToolSourceWithConfig(toolConfigs map[string]config.ToolConfig) (*LocalToolSource, error) {
	source := &LocalToolSource{
		name:  "local",
		tools: make(map[string]Tool),
	}

	// Register tools defined in the configuration
	for toolName, toolConfig := range toolConfigs {
		if !toolConfig.Enabled {
			continue
		}

		var tool Tool
		var err error

		switch toolConfig.Type {
		case "command":
			tool, err = NewCommandToolWithConfig(toolName, toolConfig)
		case "search":
			tool, err = NewSearchToolWithConfig(toolName, toolConfig)
		case "file_writer":
			tool, err = NewFileWriterToolWithConfig(toolName, toolConfig)
		case "search_replace":
			tool, err = NewSearchReplaceToolWithConfig(toolName, toolConfig)
		case "todo":
			tool = NewTodoTool()
		default:
			fmt.Printf("Warning: Unknown local tool type '%s' for tool '%s', skipping\n", toolConfig.Type, toolName)
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create tool '%s': %w", toolName, err)
		}

		if err := source.RegisterTool(tool); err != nil {
			return nil, fmt.Errorf("failed to register tool '%s': %w", toolName, err)
		}
	}

	return source, nil
}

// GetName returns the source name
func (r *LocalToolSource) GetName() string {
	return r.name
}

// GetType returns the source type
func (r *LocalToolSource) GetType() string {
	return "local"
}

// RegisterTool adds a tool to the local source
func (r *LocalToolSource) RegisterTool(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.GetName()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered in source %s", name, r.name)
	}

	r.tools[name] = tool
	// Quietly register tool (verbose logging removed for cleaner output)
	return nil
}

// DiscoverTools discovers tools (for local source, this is a no-op since tools are pre-registered)
func (r *LocalToolSource) DiscoverTools(ctx context.Context) error {
	// Local tools are registered manually, so discovery is immediate
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Tools are pre-registered, discovery is a no-op for local source
	return nil
}

// ListTools returns all tools in this source
func (r *LocalToolSource) ListTools() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []ToolInfo
	for _, tool := range r.tools {
		info := tool.GetInfo()
		// Mark as local tool
		info.ServerURL = r.name
		tools = append(tools, info)
	}

	return tools
}

// GetTool retrieves a specific tool by name
func (r *LocalToolSource) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// RemoveTool removes a tool from the source
func (r *LocalToolSource) RemoveTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool %s not found in source %s", name, r.name)
	}

	delete(r.tools, name)
	return nil
}
