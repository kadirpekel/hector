package tools

import (
	"context"
	"fmt"
	"sort"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/registry"
)

// ============================================================================
// REGISTRY - TOOL SYSTEM CORE
// ============================================================================

// ToolEntry represents a complete tool entry with all metadata
type ToolEntry struct {
	Tool       Tool       `json:"tool"`
	Source     ToolSource `json:"source"`
	SourceType string     `json:"source_type"`
	Name       string     `json:"name"`
}

// ToolRegistryError represents a tool registry error
type ToolRegistryError struct {
	Component string
	Action    string
	Message   string
	Err       error
}

func (e *ToolRegistryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Component, e.Action, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Component, e.Action, e.Message)
}

func NewToolRegistryError(component, action, message string, err error) *ToolRegistryError {
	return &ToolRegistryError{
		Component: component,
		Action:    action,
		Message:   message,
		Err:       err,
	}
}

// ToolRegistry manages multiple tool repositories and provides centralized access
type ToolRegistry struct {
	*registry.BaseRegistry[ToolEntry]
	// mu sync.RWMutex // Reserved for future use
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		BaseRegistry: registry.NewBaseRegistry[ToolEntry](),
	}
}

// NewToolRegistryWithConfig creates a new tool registry and initializes it with configuration
func NewToolRegistryWithConfig(toolConfig *config.ToolConfigs) (*ToolRegistry, error) {
	return NewToolRegistryWithConfigAndAgentRegistry(toolConfig, nil)
}

// NewToolRegistryWithConfigAndAgentRegistry creates a tool registry with agent registry for agent_call tool
func NewToolRegistryWithConfigAndAgentRegistry(toolConfig *config.ToolConfigs, agentRegistry interface{}) (*ToolRegistry, error) {
	registry := &ToolRegistry{
		BaseRegistry: registry.NewBaseRegistry[ToolEntry](),
	}

	// Initialize with configuration if provided
	if toolConfig != nil {
		if err := registry.initializeFromConfigWithAgentRegistry(toolConfig, agentRegistry); err != nil {
			return nil, fmt.Errorf("failed to initialize tool registry from config: %w", err)
		}
	}

	return registry, nil
}

// RegisterSource adds a tool source to the registry
func (r *ToolRegistry) RegisterSource(source ToolSource) error {
	name := source.GetName()
	if name == "" {
		return NewToolRegistryError("ToolRegistry", "RegisterSource", "source name cannot be empty", nil)
	}

	// Discover tools from the source
	if err := source.DiscoverTools(context.Background()); err != nil {
		return NewToolRegistryError("ToolRegistry", "RegisterSource",
			fmt.Sprintf("failed to discover tools from source %s", name), err)
	}

	// Register each tool from the source
	for _, toolInfo := range source.ListTools() {
		tool, exists := source.GetTool(toolInfo.Name)
		if !exists {
			continue
		}

		entry := ToolEntry{
			Tool:       tool,
			Source:     source,
			SourceType: source.GetType(),
			Name:       toolInfo.Name,
		}

		if err := r.Register(toolInfo.Name, entry); err != nil {
			return NewToolRegistryError("ToolRegistry", "RegisterSource",
				fmt.Sprintf("failed to register tool %s", toolInfo.Name), err)
		}
	}

	return nil
}

// DiscoverAllTools discovers tools from all registered repositories
func (r *ToolRegistry) DiscoverAllTools(ctx context.Context) error {
	// Get all repositories from entries BEFORE clearing
	repositories := make(map[string]ToolSource)
	for _, entry := range r.List() {
		repositories[entry.Source.GetName()] = entry.Source
	}

	// Clear existing tools after getting repositories
	r.Clear()

	for repoName, repo := range repositories {
		if err := repo.DiscoverTools(ctx); err != nil {
			fmt.Printf("Warning: Failed to discover tools from %s: %v\n", repoName, err)
			continue
		}

		// Register tools from this source
		for _, toolInfo := range repo.ListTools() {
			tool, exists := repo.GetTool(toolInfo.Name)
			if !exists {
				fmt.Printf("Warning: Tool %s listed but not available in %s\n", toolInfo.Name, repoName)
				continue
			}

			// Check for name conflicts
			if _, exists := r.Get(toolInfo.Name); exists {
				fmt.Printf("Warning: Tool name conflict: %s already exists (skipping)\n", toolInfo.Name)
				continue
			}

			entry := ToolEntry{
				Tool:       tool,
				Source:     repo,
				SourceType: repo.GetType(),
				Name:       toolInfo.Name,
			}

			if err := r.Register(toolInfo.Name, entry); err != nil {
				return NewToolRegistryError("ToolRegistry", "DiscoverAllTools",
					fmt.Sprintf("failed to register tool %s", toolInfo.Name), err)
			}
		}
	}
	return nil
}

// initializeFromConfigWithAgentRegistry initializes the tool registry with configuration and agent registry
func (r *ToolRegistry) initializeFromConfigWithAgentRegistry(toolConfig *config.ToolConfigs, agentRegistry interface{}) error {
	// Separate MCP tools from local tools
	localTools := make(map[string]config.ToolConfig)
	mcpTools := make(map[string]config.ToolConfig)

	for name, tool := range toolConfig.Tools {
		if tool.Type == "mcp" {
			mcpTools[name] = tool
		} else {
			localTools[name] = tool
		}
	}

	// Create and register local tool source with non-MCP tools and agent registry
	if len(localTools) > 0 {
		repo, err := NewLocalToolSourceWithConfigAndAgentRegistry(localTools, agentRegistry)
		if err != nil {
			return fmt.Errorf("failed to create local tool source: %w", err)
		}

		if err := r.RegisterSource(repo); err != nil {
			return fmt.Errorf("failed to register local source: %w", err)
		}
	}

	// Create and register MCP tool sources (each MCP server is a separate source)
	for toolName, toolConfig := range mcpTools {
		if !toolConfig.Enabled {
			continue
		}

		// Get server URL from config
		serverURL := toolConfig.ServerURL
		if serverURL == "" {
			fmt.Printf("Warning: MCP tool '%s' missing server_url, skipping\n", toolName)
			continue
		}

		// Create MCP source
		mcpSource := NewMCPToolSource(toolName, serverURL, toolConfig.Description)

		// Register the MCP source (RegisterSource will discover tools automatically)
		if err := r.RegisterSource(mcpSource); err != nil {
			fmt.Printf("Warning: Failed to register MCP source '%s': %v\n", toolName, err)
			continue
		}
	}

	return nil
}

// GetTool retrieves a tool by name
func (r *ToolRegistry) GetTool(name string) (Tool, error) {
	entry, exists := r.Get(name)
	if !exists {
		return nil, NewToolRegistryError("ToolRegistry", "GetTool",
			fmt.Sprintf("tool %s not found", name), nil)
	}
	return entry.Tool, nil
}

// ListTools returns all available tools
func (r *ToolRegistry) ListTools() []ToolInfo {
	var tools []ToolInfo
	for _, entry := range r.List() {
		info := entry.Tool.GetInfo()
		// Add source source to metadata
		info.ServerURL = entry.Source.GetName()
		tools = append(tools, info)
	}

	// Sort tools by name for consistent output
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	return tools
}

// ListToolsBySource returns tools grouped by source
func (r *ToolRegistry) ListToolsBySource() map[string][]ToolInfo {
	result := make(map[string][]ToolInfo)

	// Group tools by source
	for _, entry := range r.List() {
		repoName := entry.Source.GetName()
		if result[repoName] == nil {
			result[repoName] = make([]ToolInfo, 0)
		}
		info := entry.Tool.GetInfo()
		result[repoName] = append(result[repoName], info)
	}

	return result
}

// ExecuteTool executes a tool by name with the given arguments
func (r *ToolRegistry) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (ToolResult, error) {
	tool, err := r.GetTool(toolName)
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    err.Error(),
			ToolName: toolName,
		}, err
	}

	return tool.Execute(ctx, args)
}

// GetToolSource returns the source name that provides a specific tool
func (r *ToolRegistry) GetToolSource(toolName string) (string, error) {
	entry, exists := r.Get(toolName)
	if !exists {
		return "", NewToolRegistryError("ToolRegistry", "GetToolSource",
			fmt.Sprintf("tool %s not found", toolName), nil)
	}
	return entry.Source.GetName(), nil
}

// RemoveSource removes a source and all its tools
func (r *ToolRegistry) RemoveSource(sourceName string) error {
	// Remove all tools from this source
	for _, entry := range r.List() {
		if entry.Source.GetName() == sourceName {
			if err := r.Remove(entry.Name); err != nil {
				return NewToolRegistryError("ToolRegistry", "RemoveSource",
					fmt.Sprintf("failed to remove tool %s", entry.Name), err)
			}
		}
	}

	return nil
}
