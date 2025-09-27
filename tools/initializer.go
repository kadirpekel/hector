package tools

import (
	"fmt"

	"github.com/kadirpekel/hector"
	"github.com/kadirpekel/hector/config"
	"github.com/kadirpekel/hector/interfaces"
	"github.com/kadirpekel/hector/tools/repositories"
	"github.com/kadirpekel/hector/tools/tools"
)

// InitializeToolsFromConfig initializes the tool system from configuration
func InitializeToolsFromConfig(agent *hector.Agent, agentConfig *config.AgentConfig) error {
	// Check if tools are configured
	if len(agentConfig.Tools.Repositories) == 0 {
		// No tools configured
		return nil
	}

	// Initialize tool registry if not already done
	if agent.GetToolRegistry() == nil {
		agent.SetToolRegistry(NewToolRegistry())
	}

	// Process tool configuration
	for _, repoConfig := range agentConfig.Tools.Repositories {
		repo, err := createRepository(repoConfig)
		if err != nil {
			fmt.Printf("Warning: Failed to create tool repository %s: %v\n", repoConfig.Name, err)
			continue
		}

		if err := agent.GetToolRegistryInstance().RegisterRepository(repo); err != nil {
			fmt.Printf("Warning: Failed to register tool repository %s: %v\n", repoConfig.Name, err)
		}
	}

	return nil
}

// createRepository creates a tool repository from configuration
func createRepository(repoConfig config.ToolRepository) (interfaces.ToolRepository, error) {
	switch repoConfig.Type {
	case "local":
		return createLocalRepository(repoConfig)
	case "mcp":
		return createMCPRepository(repoConfig)
	case "plugin":
		return nil, fmt.Errorf("plugin repositories not yet implemented")
	default:
		return nil, fmt.Errorf("unknown repository type: %s", repoConfig.Type)
	}
}

// createLocalRepository creates a local tool repository from configuration
func createLocalRepository(repoConfig config.ToolRepository) (*repositories.LocalToolRepository, error) {
	repo := repositories.NewLocalToolRepository(repoConfig.Name)

	// Register configured local tools
	for _, toolConfig := range repoConfig.Tools {
		if !toolConfig.Enabled {
			continue
		}

		tool, err := createTool(toolConfig)
		if err != nil {
			fmt.Printf("Warning: Failed to create local tool %s: %v\n", toolConfig.Name, err)
			continue
		}

		if err := repo.RegisterTool(tool); err != nil {
			fmt.Printf("Warning: Failed to register local tool %s: %v\n", toolConfig.Name, err)
		}
	}

	return repo, nil
}

// createMCPRepository creates an MCP tool repository from configuration
func createMCPRepository(repoConfig config.ToolRepository) (*repositories.MCPToolRepository, error) {
	if repoConfig.URL == "" {
		return nil, fmt.Errorf("MCP repository %s requires 'url' field", repoConfig.Name)
	}

	return repositories.NewMCPToolRepository(repoConfig.Name, repoConfig.URL, repoConfig.Description), nil
}

// createTool creates a local tool from configuration
func createTool(toolConfig config.ToolDefinition) (interfaces.Tool, error) {
	switch toolConfig.Type {
	case "search":
		// Create search tool with default config
		return tools.NewSearchTool(nil), nil

	case "command":
		// Create command tool with default config
		return tools.NewCommandTool(nil), nil

	default:
		return nil, fmt.Errorf("unknown local tool type: %s", toolConfig.Type)
	}
}
