package agent

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	hectorcontext "github.com/kadirpekel/hector/pkg/context"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/tools"
)

// NewAgentServicesWithRegistry creates agent services with registry for orchestration
func NewAgentServicesWithRegistry(agentConfig *config.AgentConfig, componentManager *component.ComponentManager, registry *AgentRegistry) (reasoning.AgentServices, error) {
	// Create registry service (nil-safe)
	var registryService reasoning.AgentRegistryService
	if registry != nil {
		registryService = NewRegistryService(registry)
	} else {
		registryService = NewNoOpRegistryService()
	}

	return newAgentServicesInternal(agentConfig, componentManager, registryService)
}

// NewAgentServices creates agent services with all dependencies wired up
// Returns the configured agent services
// Deprecated: Use NewAgentServicesWithRegistry instead
func NewAgentServices(agentConfig *config.AgentConfig, componentManager *component.ComponentManager) (reasoning.AgentServices, error) {
	return NewAgentServicesWithRegistry(agentConfig, componentManager, nil)
}

// newAgentServicesInternal is the internal implementation
func newAgentServicesInternal(agentConfig *config.AgentConfig, componentManager *component.ComponentManager, registryService reasoning.AgentRegistryService) (reasoning.AgentServices, error) {
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

	// Create memory service with working memory strategy
	var memErr error

	// Create summarization service if needed for summary_buffer strategy
	var summarizer *SummarizationService
	if agentConfig.Memory.Strategy == "summary_buffer" || agentConfig.Memory.Strategy == "" {
		summarizer, memErr = NewSummarizationService(llm, &SummarizationConfig{
			Model: llm.GetModelName(),
		})
		if memErr != nil {
			return nil, fmt.Errorf("failed to create summarization service: %w", memErr)
		}
	}

	// Create working memory strategy using the factory
	workingStrategy, memErr := memory.NewWorkingMemoryStrategy(memory.WorkingMemoryConfig{
		Strategy:   agentConfig.Memory.Strategy,
		WindowSize: agentConfig.Memory.WindowSize,
		Budget:     agentConfig.Memory.Budget,
		Threshold:  agentConfig.Memory.Threshold,
		Target:     agentConfig.Memory.Target,
		Model:      llm.GetModelName(),
		LLM:        llm,
		Summarizer: summarizer,
	})
	if memErr != nil {
		return nil, fmt.Errorf("failed to create working memory strategy: %w", memErr)
	}

	// Create long-term memory strategy (optional)
	var longTermStrategy memory.LongTermMemoryStrategy
	if agentConfig.Memory.LongTerm.Enabled {
		// Long-term memory requires database + embedder (direct access, not SearchEngine)
		if agentConfig.Database == "" {
			return nil, fmt.Errorf("long-term memory requires database to be configured")
		}
		if agentConfig.Embedder == "" {
			return nil, fmt.Errorf("long-term memory requires embedder to be configured")
		}

		// Get database and embedder directly
		db, err := componentManager.GetDatabase(agentConfig.Database)
		if err != nil {
			return nil, fmt.Errorf("failed to get database '%s': %w", agentConfig.Database, err)
		}

		embedder, err := componentManager.GetEmbedder(agentConfig.Embedder)
		if err != nil {
			return nil, fmt.Errorf("failed to get embedder '%s': %w", agentConfig.Embedder, err)
		}

		// Create vector memory strategy with direct database + embedder access
		longTermStrategy, memErr = memory.NewVectorMemoryStrategy(
			db,
			embedder,
			agentConfig.Memory.LongTerm.Collection,
		)
		if memErr != nil {
			return nil, fmt.Errorf("failed to create long-term memory: %w", memErr)
		}

		fmt.Printf("✅ Long-term memory enabled (collection: %s, batch_size: %d, auto_recall: %t, recall_limit: %d)\n",
			agentConfig.Memory.LongTerm.Collection,
			agentConfig.Memory.LongTerm.BatchSize,
			agentConfig.Memory.LongTerm.AutoRecall,
			agentConfig.Memory.LongTerm.RecallLimit)
	}

	// Create memory service (orchestrates working + long-term memory)
	historyService := memory.NewMemoryService(
		workingStrategy,
		longTermStrategy, // May be nil if not enabled
		memory.LongTermConfig{
			Enabled:      agentConfig.Memory.LongTerm.Enabled,
			StorageScope: memory.StorageScope(agentConfig.Memory.LongTerm.StorageScope),
			BatchSize:    agentConfig.Memory.LongTerm.BatchSize,
			AutoRecall:   agentConfig.Memory.LongTerm.AutoRecall,
			RecallLimit:  agentConfig.Memory.LongTerm.RecallLimit,
			Collection:   agentConfig.Memory.LongTerm.Collection,
		},
	)

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
		registryService,
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
// Registry will be nil for agents created through factory (typically tests)
// For production multi-agent scenarios, use NewAgent directly with a registry
func (f *AgentFactory) CreateAgent(agentConfig *config.AgentConfig) (*Agent, error) {
	if agentConfig == nil {
		return nil, fmt.Errorf("agent config cannot be nil")
	}

	// Single place for agent creation logic - delegates to NewAgent
	// Pass nil registry - orchestration won't be available
	return NewAgent(agentConfig, f.componentManager, nil)
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
