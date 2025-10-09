package agent

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/tools"
)

// NewAgentServices creates agent services with all dependencies wired up
// Returns the configured agent services
func NewAgentServices(agentConfig *config.AgentConfig, componentManager *component.ComponentManager) (reasoning.AgentServices, error) {
	if agentConfig == nil {
		return nil, fmt.Errorf("agent config cannot be nil")
	}
	if componentManager == nil {
		return nil, fmt.Errorf("component manager cannot be nil")
	}

	// Initialize LLM
	llm, err := componentManager.GetLLM(agentConfig.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM '%s': %w", agentConfig.LLM, err)
	}

	// Initialize services
	toolRegistry := componentManager.GetToolRegistry()

	// Create the strategy early to check for required tools
	strategy, err := reasoning.CreateStrategy(agentConfig.Reasoning.Engine, agentConfig.Reasoning)
	if err != nil {
		return nil, fmt.Errorf("failed to create reasoning strategy: %w", err)
	}

	// Auto-register strategy-required tools (e.g., todo_write for ChainOfThought)
	requiredTools := strategy.GetRequiredTools()
	for _, reqTool := range requiredTools {
		// Check if tool already exists
		if _, err := toolRegistry.GetTool(reqTool.Name); err == nil {
			// Tool already exists, skip
			continue
		}

		// Tool doesn't exist - auto-create if requested
		if reqTool.AutoCreate {
			if err := registerRequiredTool(toolRegistry, reqTool); err != nil {
				return nil, fmt.Errorf("failed to register required tool '%s': %w", reqTool.Name, err)
			}
			fmt.Printf("Info: Auto-registered strategy-required tool '%s' (%s)\n", reqTool.Name, reqTool.Description)
		}
	}

	// Create context service - only if document stores are configured
	var contextService reasoning.ContextService
	if len(agentConfig.DocumentStores) > 0 {
		// Document stores require both database and embedder
		if agentConfig.Database == "" {
			return nil, fmt.Errorf("database is required when document stores are configured")
		}
		if agentConfig.Embedder == "" {
			return nil, fmt.Errorf("embedder is required when document stores are configured")
		}

		// Get database and embedder for search engine
		db, err := componentManager.GetDatabase(agentConfig.Database)
		if err != nil {
			return nil, fmt.Errorf("failed to get database '%s': %w", agentConfig.Database, err)
		}

		embedder, err := componentManager.GetEmbedder(agentConfig.Embedder)
		if err != nil {
			return nil, fmt.Errorf("failed to get embedder '%s': %w", agentConfig.Embedder, err)
		}

		searchEngine, err := hectorcontext.NewSearchEngine(db, embedder, agentConfig.Search)
		if err != nil {
			return nil, fmt.Errorf("failed to create search engine: %w", err)
		}
		contextService = NewContextService(searchEngine)
	} else {
		// No document stores configured - create a no-op context service
		contextService = NewNoOpContextService()
	}

	// Create services (order matters due to dependencies)
	llmService := NewLLMService(llm)
	toolService := NewToolService(toolRegistry, agentConfig.Tools)

	// Create history service based on memory configuration
	var historyService reasoning.HistoryService
	var histErr error

	if agentConfig.Memory.Budget > 0 {
		// Token-aware mode enabled
		historyService, histErr = NewHistoryService(&HistoryConfig{
			MaxMessages:         agentConfig.Prompt.MaxHistoryMessages, // Fallback
			TokenBudget:         agentConfig.Memory.Budget,
			Model:               llm.GetModelName(),
			EnableSummarization: agentConfig.Memory.Summarization,
			SummarizeThreshold:  agentConfig.Memory.SummarizationThreshold,
			LLM:                 llm,
		})
		if histErr != nil {
			return nil, fmt.Errorf("failed to create token-aware history service: %w", histErr)
		}
	} else {
		// Count-based mode (fallback)
		maxHistory := 10
		if agentConfig.Prompt.MaxHistoryMessages > 0 {
			maxHistory = agentConfig.Prompt.MaxHistoryMessages
		}
		historyService, histErr = NewHistoryService(&HistoryConfig{
			MaxMessages: maxHistory,
		})
		if histErr != nil {
			return nil, fmt.Errorf("failed to create history service: %w", histErr)
		}
	}

	// contextService already created above based on document store availability
	promptService := NewPromptService(agentConfig.Prompt, contextService, historyService)

	// Create agent services for dependency injection
	// Note: promptService already has contextService and historyService as dependencies
	agentServices := reasoning.NewAgentServices(
		agentConfig.Reasoning,
		llmService,
		toolService,
		contextService,
		promptService,
		historyService,
	)

	return agentServices, nil
}

// registerRequiredTool creates and registers a required tool in the registry
func registerRequiredTool(registry *tools.ToolRegistry, reqTool reasoning.RequiredTool) error {
	// Create a local tool source for the required tool
	localSource := tools.NewLocalToolSource("strategy-required")

	// Create the tool based on type
	var tool tools.Tool
	switch reqTool.Type {
	case "todo":
		tool = tools.NewTodoTool()
	case "command":
		// Create a basic command tool with safe defaults
		cmdTool, err := tools.NewCommandToolWithConfig(reqTool.Name, config.ToolConfig{
			Type:             "command",
			AllowedCommands:  []string{"ls", "cat", "pwd", "echo"},
			WorkingDirectory: "./",
			MaxExecutionTime: "30s",
			EnableSandboxing: true,
		})
		if err != nil {
			return err
		}
		tool = cmdTool
	default:
		return fmt.Errorf("unsupported required tool type: %s", reqTool.Type)
	}

	// Register the tool in the local source
	if err := localSource.RegisterTool(tool); err != nil {
		return err
	}

	// Register the source in the registry
	return registry.RegisterSource(localSource)
}

// ============================================================================
// AGENT FACTORY - SINGLE SOURCE OF TRUTH FOR AGENT CREATION
// ============================================================================

// AgentFactory creates and configures agent instances
type AgentFactory struct {
	componentManager *component.ComponentManager
}

// NewAgentFactory creates a new agent factory
func NewAgentFactory(componentManager *component.ComponentManager) *AgentFactory {
	if componentManager == nil {
		return nil
	}
	return &AgentFactory{
		componentManager: componentManager,
	}
}

// CreateAgent creates a new agent with the given configuration
func (f *AgentFactory) CreateAgent(agentConfig *config.AgentConfig) (*Agent, error) {
	if agentConfig == nil {
		return nil, fmt.Errorf("agent config cannot be nil")
	}

	// Single place for agent creation logic - delegates to NewAgent
	return NewAgent(agentConfig, f.componentManager)
}

// CreateAgentWithServices creates an agent with pre-configured services (for testing)
func (f *AgentFactory) CreateAgentWithServices(agentConfig *config.AgentConfig, services reasoning.AgentServices) (*Agent, error) {
	if agentConfig == nil {
		return nil, fmt.Errorf("agent config cannot be nil")
	}
	if services == nil {
		return nil, fmt.Errorf("agent services cannot be nil")
	}

	// Create agent with provided services
	return &Agent{
		name:        agentConfig.Name,
		description: agentConfig.Description,
		config:      agentConfig,
		services:    services,
	}, nil
}
