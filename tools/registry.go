package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// REGISTRY - TOOL SYSTEM CORE
// ============================================================================

// Tool and ToolRepository interfaces are imported from the interfaces package

// ToolRegistry manages multiple tool repositories and provides centralized access
type ToolRegistry struct {
	mu           sync.RWMutex
	repositories map[string]interfaces.ToolRepository
	tools        map[string]interfaces.Tool
	toolSources  map[string]string // tool name -> repository name
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		repositories: make(map[string]interfaces.ToolRepository),
		tools:        make(map[string]interfaces.Tool),
		toolSources:  make(map[string]string),
	}
}

// RegisterRepository adds a tool repository to the registry
func (r *ToolRegistry) RegisterRepository(repository interfaces.ToolRepository) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := repository.GetName()
	if name == "" {
		return fmt.Errorf("repository name cannot be empty")
	}

	if _, exists := r.repositories[name]; exists {
		return fmt.Errorf("repository %s already registered", name)
	}

	r.repositories[name] = repository
	return nil
}

// DiscoverAllTools discovers tools from all registered repositories
func (r *ToolRegistry) DiscoverAllTools(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing tools
	r.tools = make(map[string]interfaces.Tool)
	r.toolSources = make(map[string]string)

	for repoName, repo := range r.repositories {
		fmt.Printf("Discovering tools from repository: %s (%s)\n", repoName, repo.GetType())

		if err := repo.DiscoverTools(ctx); err != nil {
			fmt.Printf("Warning: Failed to discover tools from %s: %v\n", repoName, err)
			continue
		}

		// Register tools from this repository
		for _, toolInfo := range repo.ListTools() {
			tool, exists := repo.GetTool(toolInfo.Name)
			if !exists {
				fmt.Printf("Warning: Tool %s listed but not available in %s\n", toolInfo.Name, repoName)
				continue
			}

			// Check for name conflicts
			if existingSource, exists := r.toolSources[toolInfo.Name]; exists {
				fmt.Printf("Warning: Tool name conflict: %s exists in both %s and %s (using %s)\n",
					toolInfo.Name, existingSource, repoName, existingSource)
				continue
			}

			r.tools[toolInfo.Name] = tool
			r.toolSources[toolInfo.Name] = repoName
		}
	}

	fmt.Printf("Discovered %d tools total across %d repositories\n", len(r.tools), len(r.repositories))
	return nil
}

// GetTool retrieves a tool by name
func (r *ToolRegistry) GetTool(name string) (interfaces.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// ListTools returns all available tools
func (r *ToolRegistry) ListTools() []interfaces.ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []interfaces.ToolInfo
	for _, tool := range r.tools {
		info := tool.GetInfo()
		// Add repository source to metadata
		if source, exists := r.toolSources[tool.GetName()]; exists {
			if info.Parameters == nil {
				info.Parameters = make([]interfaces.ToolParameter, 0)
			}
			// Add source as metadata in ServerURL field for compatibility
			info.ServerURL = source
		}
		tools = append(tools, info)
	}

	// Sort tools by name for consistent output
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	return tools
}

// ListToolsByRepository returns tools grouped by repository
func (r *ToolRegistry) ListToolsByRepository() map[string][]interfaces.ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]interfaces.ToolInfo)

	for repoName, repo := range r.repositories {
		result[repoName] = repo.ListTools()
	}

	return result
}

// ExecuteTool executes a tool by name with the given arguments
func (r *ToolRegistry) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (interfaces.ToolResult, error) {
	tool, exists := r.GetTool(toolName)
	if !exists {
		return interfaces.ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("tool %s not found", toolName),
			ToolName: toolName,
		}, fmt.Errorf("tool %s not found", toolName)
	}

	return tool.Execute(ctx, args)
}

// GetRepositoryStatus returns the health status of all repositories
func (r *ToolRegistry) GetRepositoryStatus(ctx context.Context) map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make(map[string]bool)
	for name, repo := range r.repositories {
		status[name] = repo.IsHealthy(ctx)
	}

	return status
}

// GetToolSource returns the repository name that provides a specific tool
func (r *ToolRegistry) GetToolSource(toolName string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	source, exists := r.toolSources[toolName]
	return source, exists
}

// RemoveRepository removes a repository and all its tools
func (r *ToolRegistry) RemoveRepository(repositoryName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.repositories[repositoryName]; !exists {
		return fmt.Errorf("repository %s not found", repositoryName)
	}

	// Remove all tools from this repository
	for toolName, source := range r.toolSources {
		if source == repositoryName {
			delete(r.tools, toolName)
			delete(r.toolSources, toolName)
		}
	}

	delete(r.repositories, repositoryName)
	return nil
}

// ============================================================================
// GLOBAL REGISTRY INSTANCE (DEPRECATED - Use instance methods instead)
// ============================================================================

// GlobalToolRegistry is the default tool registry instance
// Deprecated: Use NewToolRegistry() and instance methods instead
var GlobalToolRegistry = NewToolRegistry()

// RegisterToolRepository registers a repository in the global registry
// Deprecated: Use registry.RegisterRepository() instead
func RegisterToolRepository(repository interfaces.ToolRepository) error {
	return GlobalToolRegistry.RegisterRepository(repository)
}

// DiscoverAllTools discovers tools from all repositories in the global registry
// Deprecated: Use registry.DiscoverAllTools() instead
func DiscoverAllTools(ctx context.Context) error {
	return GlobalToolRegistry.DiscoverAllTools(ctx)
}

// GetTool retrieves a tool from the global registry
// Deprecated: Use registry.GetTool() instead
func GetTool(name string) (interfaces.Tool, bool) {
	return GlobalToolRegistry.GetTool(name)
}

// ExecuteTool executes a tool using the global registry
// Deprecated: Use registry.ExecuteTool() instead
func ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (interfaces.ToolResult, error) {
	return GlobalToolRegistry.ExecuteTool(ctx, toolName, args)
}

// ListAllTools returns all tools from the global registry
// Deprecated: Use registry.ListTools() instead
func ListAllTools() []interfaces.ToolInfo {
	return GlobalToolRegistry.ListTools()
}
