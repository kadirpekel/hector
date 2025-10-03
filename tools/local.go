package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/kadirpekel/hector/config"
)

// ============================================================================
// LOCAL - BUILT-IN TOOL REPOSITORY
// ============================================================================

// LocalToolRepository manages built-in/local tools
type LocalToolRepository struct {
	name  string
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewLocalToolRepository creates a new local tool repository
func NewLocalToolRepository(name string) *LocalToolRepository {
	if name == "" {
		name = "local"
	}

	return &LocalToolRepository{
		name:  name,
		tools: make(map[string]Tool),
	}
}

// NewLocalToolRepositoryWithConfig creates a new local tool repository from configuration
func NewLocalToolRepositoryWithConfig(repoConfig config.ToolRepository) (*LocalToolRepository, error) {
	repo := &LocalToolRepository{
		name:  repoConfig.Name,
		tools: make(map[string]Tool),
	}

	// Register tools defined in the configuration
	for _, toolDef := range repoConfig.Tools {
		if !toolDef.Enabled {
			continue
		}

		var tool Tool
		var err error

		switch toolDef.Type {
		case "command":
			tool, err = NewCommandToolWithConfig(toolDef)
		case "search":
			tool, err = NewSearchToolWithConfig(toolDef)
		case "file_writer":
			tool, err = NewFileWriterToolWithConfig(toolDef)
		case "search_replace":
			tool, err = NewSearchReplaceToolWithConfig(toolDef)
		case "todo":
			tool = NewTodoTool()
		default:
			fmt.Printf("Warning: Unknown local tool type '%s' for tool '%s', skipping\n", toolDef.Type, toolDef.Name)
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create tool '%s': %w", toolDef.Name, err)
		}

		if err := repo.RegisterTool(tool); err != nil {
			return nil, fmt.Errorf("failed to register tool '%s': %w", toolDef.Name, err)
		}
	}

	return repo, nil
}

// GetName returns the repository name
func (r *LocalToolRepository) GetName() string {
	return r.name
}

// GetType returns the repository type
func (r *LocalToolRepository) GetType() string {
	return "local"
}

// RegisterTool adds a tool to the local repository
func (r *LocalToolRepository) RegisterTool(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.GetName()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered in repository %s", name, r.name)
	}

	r.tools[name] = tool
	// Quietly register tool (verbose logging removed for cleaner output)
	return nil
}

// DiscoverTools discovers tools (for local repository, this is a no-op since tools are pre-registered)
func (r *LocalToolRepository) DiscoverTools(ctx context.Context) error {
	// Local tools are registered manually, so discovery is immediate
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Tools are pre-registered, discovery is a no-op for local repository
	return nil
}

// ListTools returns all tools in this repository
func (r *LocalToolRepository) ListTools() []ToolInfo {
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
func (r *LocalToolRepository) GetTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// RemoveTool removes a tool from the repository
func (r *LocalToolRepository) RemoveTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool %s not found in repository %s", name, r.name)
	}

	delete(r.tools, name)
	return nil
}
