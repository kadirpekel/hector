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
	toolExtension := reasoning.NewToolExtension(toolRegistry)
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

	return reasoningEngine, agentServices, nil
}

// NewReasoningEngine creates a reasoning engine with all dependencies wired up
// This replaces the bloated Agent struct with a simple factory function
func NewReasoningEngine(agentConfig *config.AgentConfig, componentManager *component.ComponentManager) (reasoning.ReasoningEngine, error) {
	reasoningEngine, _, err := NewReasoningEngineWithServices(agentConfig, componentManager)
	return reasoningEngine, err
}
