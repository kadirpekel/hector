package repositories

import (
	"context"
	"fmt"
	"sync"

	"github.com/kadirpekel/hector/interfaces"
)

// ============================================================================
// LOCAL - BUILT-IN TOOL REPOSITORY
// ============================================================================

// LocalToolRepository manages built-in/local tools
type LocalToolRepository struct {
	name  string
	tools map[string]interfaces.Tool
	mu    sync.RWMutex
}

// NewLocalToolRepository creates a new local tool repository
func NewLocalToolRepository(name string) *LocalToolRepository {
	if name == "" {
		name = "local"
	}

	return &LocalToolRepository{
		name:  name,
		tools: make(map[string]interfaces.Tool),
	}
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
func (r *LocalToolRepository) RegisterTool(tool interfaces.Tool) error {
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
	fmt.Printf("Registered local tool: %s - %s\n", name, tool.GetDescription())
	return nil
}

// DiscoverTools discovers tools (for local repository, this is a no-op since tools are pre-registered)
func (r *LocalToolRepository) DiscoverTools(ctx context.Context) error {
	// Local tools are registered manually, so discovery is immediate
	r.mu.RLock()
	defer r.mu.RUnlock()

	fmt.Printf("Local repository %s has %d pre-registered tools\n", r.name, len(r.tools))
	return nil
}

// ListTools returns all tools in this repository
func (r *LocalToolRepository) ListTools() []interfaces.ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []interfaces.ToolInfo
	for _, tool := range r.tools {
		info := tool.GetInfo()
		// Mark as local tool
		info.ServerURL = r.name
		tools = append(tools, info)
	}

	return tools
}

// GetTool retrieves a specific tool by name
func (r *LocalToolRepository) GetTool(name string) (interfaces.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// IsHealthy returns true if the repository is operational (always true for local)
func (r *LocalToolRepository) IsHealthy(ctx context.Context) bool {
	return true
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
