package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/kadirpekel/hector/pkg/config"
)

type LocalToolSource struct {
	name  string
	tools map[string]Tool
	mu    sync.RWMutex
}

func NewLocalToolSource(name string) *LocalToolSource {
	if name == "" {
		name = "local"
	}

	return &LocalToolSource{
		name:  name,
		tools: make(map[string]Tool),
	}
}

func NewLocalToolSourceWithConfig(toolConfigs map[string]*config.ToolConfig) (*LocalToolSource, error) {
	return NewLocalToolSourceWithConfigAndAgentRegistry(toolConfigs, nil)
}

func NewLocalToolSourceWithConfigAndAgentRegistry(toolConfigs map[string]*config.ToolConfig, agentRegistry interface{}) (*LocalToolSource, error) {
	source := &LocalToolSource{
		name:  "local",
		tools: make(map[string]Tool),
	}

	for toolName, toolConfig := range toolConfigs {
		if toolConfig == nil || toolConfig.Enabled == nil || !*toolConfig.Enabled {
			continue
		}

		var tool Tool
		var err error

		switch toolConfig.Type {
		case "command":
			tool, err = NewCommandToolWithConfig(toolName, toolConfig)
		case "search":
			// Search tool: document stores come from agent assignment, not config
			// When created from global tool config (not agent context), use empty slice = search all stores
			tool, err = NewSearchToolWithConfig(toolName, toolConfig, []string{})
		case "write_file":
			tool, err = NewFileWriterToolWithConfig(toolName, toolConfig)
		case "search_replace":
			tool, err = NewSearchReplaceToolWithConfig(toolName, toolConfig)
		case "read_file":
			tool, err = NewReadFileToolWithConfig(toolName, toolConfig)
		case "apply_patch":
			tool, err = NewApplyPatchToolWithConfig(toolName, toolConfig)
		case "grep_search":
			tool, err = NewGrepSearchToolWithConfig(toolName, toolConfig)
		case "web_request":
			tool, err = NewWebRequestToolWithConfig(toolName, toolConfig)
		case "todo":
			tool = NewTodoTool()
		case "agent_call":

			var registry AgentRegistry
			if agentRegistry != nil {
				if ar, ok := agentRegistry.(AgentRegistry); ok {
					registry = ar
				}
			}

			if registry == nil {
				return nil, fmt.Errorf("agent_call tool requires agent registry but none was provided")
			}
			tool = NewAgentCallTool(registry)
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

func (r *LocalToolSource) GetName() string {
	return r.name
}

func (r *LocalToolSource) GetType() string {
	return "local"
}

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

	return nil
}

func (r *LocalToolSource) DiscoverTools(ctx context.Context) error {

	r.mu.RLock()
	defer r.mu.RUnlock()

	return nil
}

func (r *LocalToolSource) ListTools() []ToolInfo {
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

func (r *LocalToolSource) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

func (r *LocalToolSource) RemoveTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool %s not found in source %s", name, r.name)
	}

	delete(r.tools, name)
	return nil
}
