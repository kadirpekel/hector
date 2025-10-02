package agent

import (
	"fmt"

	"github.com/kadirpekel/hector/component"
	"github.com/kadirpekel/hector/config"
	hectorcontext "github.com/kadirpekel/hector/context"
	"github.com/kadirpekel/hector/reasoning"
)

// NewReasoningEngineWithServices creates a reasoning engine with all dependencies wired up
// Returns both the reasoning engine and the agent services for compatibility
func NewReasoningEngineWithServices(agentConfig *config.AgentConfig, componentManager *component.ComponentManager) (reasoning.ReasoningEngine, reasoning.AgentServices, error) {
	if agentConfig == nil {
		return nil, nil, fmt.Errorf("agent config cannot be nil")
	}
	if componentManager == nil {
		return nil, nil, fmt.Errorf("component manager cannot be nil")
	}

	// Initialize LLM
	llm, err := componentManager.GetLLM(agentConfig.LLM)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get LLM '%s': %w", agentConfig.LLM, err)
	}

	// Initialize services
	toolRegistry := componentManager.GetToolRegistry()

	// Create extension service and register tools as native extension
	extensionService := reasoning.NewExtensionService()
	toolExtension := reasoning.NewToolExtension(toolRegistry, extensionService)
	extensionService.RegisterExtension(toolExtension.CreateExtension())

	// Create context service - only if document stores are configured
	var contextService reasoning.ContextService
	if len(agentConfig.DocumentStores) > 0 {
		// Get database and embedder for search engine
		db, err := componentManager.GetDatabase(agentConfig.Database)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get database '%s': %w", agentConfig.Database, err)
		}

		embedder, err := componentManager.GetEmbedder(agentConfig.Embedder)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get embedder '%s': %w", agentConfig.Embedder, err)
		}

		searchEngine, err := hectorcontext.NewSearchEngine(db, embedder, agentConfig.Search)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create search engine: %w", err)
		}
		contextService = NewContextService(searchEngine)
	} else {
		// No document stores configured - create a no-op context service
		contextService = NewNoOpContextService()
	}

	llmService := NewLLMServiceWithExtensions(llm, extensionService)
	promptService := NewPromptService()
	historyService := NewHistoryService(10) // Max 10 history items

	// Create agent services for dependency injection
	agentServices := reasoning.NewAgentServices(
		agentConfig.Reasoning,
		llmService,
		contextService,
		extensionService,
		promptService,
		historyService,
	)

	// Create reasoning engine
	reasoningFactory := reasoning.NewReasoningEngineFactory()
	reasoningEngine, err := reasoningFactory.CreateEngine(agentConfig.Reasoning.Engine, agentServices)
	if err != nil {
		return nil, nil, err
	}

	// Chain-of-thought engine uses behavioral signals (tool calls) for continuation
	// No need for explicit REASONING_CALL extension

	return reasoningEngine, agentServices, nil
}

// NewReasoningEngine creates a reasoning engine with all dependencies wired up
// This replaces the bloated Agent struct with a simple factory function
func NewReasoningEngine(agentConfig *config.AgentConfig, componentManager *component.ComponentManager) (reasoning.ReasoningEngine, error) {
	reasoningEngine, _, err := NewReasoningEngineWithServices(agentConfig, componentManager)
	return reasoningEngine, err
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

	// Create reasoning engine with provided services
	reasoningFactory := reasoning.NewReasoningEngineFactory()
	reasoningEngine, err := reasoningFactory.CreateEngine(agentConfig.Reasoning.Engine, services)
	if err != nil {
		return nil, fmt.Errorf("failed to create reasoning engine: %w", err)
	}

	// Create agent
	return &Agent{
		name:            agentConfig.Name,
		description:     agentConfig.Description,
		config:          agentConfig,
		reasoningEngine: reasoningEngine,
	}, nil
}
