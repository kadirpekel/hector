package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/registry"
)

// ============================================================================
// REGISTRY - TOOL SYSTEM CORE
// ============================================================================

// ToolEntry represents a complete tool entry with all metadata
type ToolEntry struct {
	Tool           Tool           `json:"tool"`
	Repository     ToolRepository `json:"repository"`
	RepositoryType string         `json:"repository_type"`
	Name           string         `json:"name"`
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
	mu sync.RWMutex
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		BaseRegistry: registry.NewBaseRegistry[ToolEntry](),
	}
}

// NewToolRegistryWithConfig creates a new tool registry and initializes it with configuration
func NewToolRegistryWithConfig(toolConfig *config.ToolConfigs) (*ToolRegistry, error) {
	registry := &ToolRegistry{
		BaseRegistry: registry.NewBaseRegistry[ToolEntry](),
	}

	// Initialize with configuration if provided
	if toolConfig != nil {
		if err := registry.initializeFromConfig(toolConfig); err != nil {
			return nil, fmt.Errorf("failed to initialize tool registry from config: %w", err)
		}
	}

	return registry, nil
}

// RegisterRepository adds a tool repository to the registry
func (r *ToolRegistry) RegisterRepository(repository ToolRepository) error {
	name := repository.GetName()
	if name == "" {
		return NewToolRegistryError("ToolRegistry", "RegisterRepository", "repository name cannot be empty", nil)
	}

	// Discover tools from the repository
	if err := repository.DiscoverTools(context.Background()); err != nil {
		return NewToolRegistryError("ToolRegistry", "RegisterRepository",
			fmt.Sprintf("failed to discover tools from repository %s", name), err)
	}

	// Register each tool from the repository
	for _, toolInfo := range repository.ListTools() {
		tool, exists := repository.GetTool(toolInfo.Name)
		if !exists {
			continue
		}

		entry := ToolEntry{
			Tool:           tool,
			Repository:     repository,
			RepositoryType: repository.GetType(),
			Name:           toolInfo.Name,
		}

		if err := r.Register(toolInfo.Name, entry); err != nil {
			return NewToolRegistryError("ToolRegistry", "RegisterRepository",
				fmt.Sprintf("failed to register tool %s", toolInfo.Name), err)
		}
	}

	return nil
}

// DiscoverAllTools discovers tools from all registered repositories
func (r *ToolRegistry) DiscoverAllTools(ctx context.Context) error {
	// Get all repositories from entries BEFORE clearing
	repositories := make(map[string]ToolRepository)
	for _, entry := range r.List() {
		repositories[entry.Repository.GetName()] = entry.Repository
	}

	// Clear existing tools after getting repositories
	r.Clear()

	for repoName, repo := range repositories {
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
			if _, exists := r.Get(toolInfo.Name); exists {
				fmt.Printf("Warning: Tool name conflict: %s already exists (skipping)\n", toolInfo.Name)
				continue
			}

			entry := ToolEntry{
				Tool:           tool,
				Repository:     repo,
				RepositoryType: repo.GetType(),
				Name:           toolInfo.Name,
			}

			if err := r.Register(toolInfo.Name, entry); err != nil {
				return NewToolRegistryError("ToolRegistry", "DiscoverAllTools",
					fmt.Sprintf("failed to register tool %s", toolInfo.Name), err)
			}
		}
	}
	return nil
}

// initializeFromConfig initializes the tool registry with configuration
func (r *ToolRegistry) initializeFromConfig(toolConfig *config.ToolConfigs) error {
	// Process each repository configuration
	for _, repoConfig := range toolConfig.Repositories {
		var repo ToolRepository
		var err error

		switch repoConfig.Type {
		case "local":
			repo, err = NewLocalToolRepositoryWithConfig(repoConfig)
		case "mcp":
			repo, err = NewMCPToolRepositoryWithConfig(repoConfig)
		default:
			return fmt.Errorf("unsupported repository type: %s", repoConfig.Type)
		}

		if err != nil {
			return fmt.Errorf("failed to create %s repository '%s': %w", repoConfig.Type, repoConfig.Name, err)
		}

		// Register the repository
		if err := r.RegisterRepository(repo); err != nil {
			return fmt.Errorf("failed to register repository '%s': %w", repoConfig.Name, err)
		}
	}

	// After creating all repositories, discover tools from them
	ctx := context.Background()
	if err := r.DiscoverAllTools(ctx); err != nil {
		return fmt.Errorf("failed to discover tools: %w", err)
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
		// Add repository source to metadata
		info.ServerURL = entry.Repository.GetName()
		tools = append(tools, info)
	}

	// Sort tools by name for consistent output
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	return tools
}

// ListToolsByRepository returns tools grouped by repository
func (r *ToolRegistry) ListToolsByRepository() map[string][]ToolInfo {
	result := make(map[string][]ToolInfo)

	// Group tools by repository
	for _, entry := range r.List() {
		repoName := entry.Repository.GetName()
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

// GetToolSource returns the repository name that provides a specific tool
func (r *ToolRegistry) GetToolSource(toolName string) (string, error) {
	entry, exists := r.Get(toolName)
	if !exists {
		return "", NewToolRegistryError("ToolRegistry", "GetToolSource",
			fmt.Sprintf("tool %s not found", toolName), nil)
	}
	return entry.Repository.GetName(), nil
}

// RemoveRepository removes a repository and all its tools
func (r *ToolRegistry) RemoveRepository(repositoryName string) error {
	// Remove all tools from this repository
	for _, entry := range r.List() {
		if entry.Repository.GetName() == repositoryName {
			if err := r.Remove(entry.Name); err != nil {
				return NewToolRegistryError("ToolRegistry", "RemoveRepository",
					fmt.Sprintf("failed to remove tool %s", entry.Name), err)
			}
		}
	}

	return nil
}
