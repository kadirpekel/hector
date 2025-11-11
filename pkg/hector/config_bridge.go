package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/reasoning"
)

// ConfigAgentBuilder builds agents from config using the programmatic API
type ConfigAgentBuilder struct {
	config             *config.Config
	componentManager   *component.ComponentManager
	agentRegistry      *agent.AgentRegistry
	baseURL            string
	preferredTransport string
}

// NewConfigAgentBuilder creates a builder that uses programmatic API
func NewConfigAgentBuilder(cfg *config.Config) (*ConfigAgentBuilder, error) {
	// Create agent registry for multi-agent scenarios
	agentRegistry := agent.NewAgentRegistry()

	// Initialize component manager from config (needed to resolve LLM/Tool references)
	componentMgr, err := component.NewComponentManagerWithAgentRegistry(cfg, agentRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize component manager: %w", err)
	}

	// Resolve base URL
	baseURL := resolveBaseURL(cfg)
	preferredTransport := cfg.Global.A2AServer.PreferredTransport
	if preferredTransport == "" {
		preferredTransport = "json-rpc"
	}

	return &ConfigAgentBuilder{
		config:             cfg,
		componentManager:   componentMgr,
		agentRegistry:      agentRegistry,
		baseURL:            baseURL,
		preferredTransport: preferredTransport,
	}, nil
}

// resolveBaseURL constructs the base URL from config
func resolveBaseURL(cfg *config.Config) string {
	if cfg.Global.A2AServer.BaseURL != "" {
		return cfg.Global.A2AServer.BaseURL
	}
	host := cfg.Global.A2AServer.Host
	if host == "" || host == "0.0.0.0" {
		host = "localhost"
	}
	port := cfg.Global.A2AServer.Port
	if port == 0 {
		port = 8080
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

// BuildAgent builds an agent from config using programmatic API
func (b *ConfigAgentBuilder) BuildAgent(agentID string) (*agent.Agent, error) {
	agentCfg, ok := b.config.Agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent %s not found in config", agentID)
	}

	// Use programmatic API to build agent
	builder := NewAgent(agentID).
		WithName(agentCfg.Name).
		WithDescription(agentCfg.Description).
		WithRegistry(b.agentRegistry).
		WithBaseURL(b.baseURL).
		WithPreferredTransport(b.preferredTransport)

	// Get LLM provider from component manager
	llmProvider, err := b.componentManager.GetLLM(agentCfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM %s: %w", agentCfg.LLM, err)
	}
	builder = builder.WithLLMProvider(llmProvider)

	// Convert prompt config
	if agentCfg.Prompt.SystemPrompt != "" {
		builder = builder.WithSystemPrompt(agentCfg.Prompt.SystemPrompt)
	}

	// Convert prompt slots
	if agentCfg.Prompt.PromptSlots != nil {
		promptSlots := &reasoning.PromptSlots{
			SystemRole:   agentCfg.Prompt.PromptSlots.SystemRole,
			Instructions: agentCfg.Prompt.PromptSlots.Instructions,
			UserGuidance: agentCfg.Prompt.PromptSlots.UserGuidance,
		}
		builder = builder.WithPromptSlots(promptSlots)
	}

	// Convert reasoning config using programmatic API
	reasoningBuilder := NewReasoning(agentCfg.Reasoning.Engine)
	if agentCfg.Reasoning.MaxIterations > 0 {
		reasoningBuilder = reasoningBuilder.MaxIterations(agentCfg.Reasoning.MaxIterations)
	}
	if agentCfg.Reasoning.EnableStreaming != nil {
		reasoningBuilder = reasoningBuilder.EnableStreaming(*agentCfg.Reasoning.EnableStreaming)
	}
	if agentCfg.Reasoning.ShowTools != nil {
		reasoningBuilder = reasoningBuilder.ShowTools(*agentCfg.Reasoning.ShowTools)
	}
	if agentCfg.Reasoning.ShowThinking != nil {
		reasoningBuilder = reasoningBuilder.ShowThinking(*agentCfg.Reasoning.ShowThinking)
	}
	builder = builder.WithReasoning(reasoningBuilder)

	// Convert memory config using programmatic API
	workingMemBuilder := NewWorkingMemory(agentCfg.Memory.Strategy)
	if agentCfg.Memory.Strategy == "summary_buffer" || agentCfg.Memory.Strategy == "" {
		workingMemBuilder = workingMemBuilder.
			Budget(agentCfg.Memory.Budget).
			Threshold(agentCfg.Memory.Threshold).
			Target(agentCfg.Memory.Target).
			WithLLMProvider(llmProvider)
	} else if agentCfg.Memory.Strategy == "buffer_window" {
		workingMemBuilder = workingMemBuilder.WindowSize(agentCfg.Memory.WindowSize)
	}

	workingMemory, err := workingMemBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create working memory: %w", err)
	}
	builder = builder.WithWorkingMemory(workingMemory)

	// Long-term memory
	if agentCfg.Memory.LongTerm.IsEnabled() {
		db, err := b.componentManager.GetDatabase(agentCfg.Database)
		if err != nil {
			return nil, fmt.Errorf("failed to get database: %w", err)
		}

		embedder, err := b.componentManager.GetEmbedder(agentCfg.Embedder)
		if err != nil {
			return nil, fmt.Errorf("failed to get embedder: %w", err)
		}

		longTermBuilder := NewLongTermMemory().
			Enabled(true).
			Collection(agentCfg.Memory.LongTerm.Collection).
			StorageScope(memory.StorageScope(agentCfg.Memory.LongTerm.StorageScope)).
			BatchSize(agentCfg.Memory.LongTerm.BatchSize).
			AutoRecall(boolValue(agentCfg.Memory.LongTerm.AutoRecall, false)).
			RecallLimit(agentCfg.Memory.LongTerm.RecallLimit).
			WithDatabase(db).
			WithEmbedder(embedder)

		longTermMemory, longTermConfig, err := longTermBuilder.Build()
		if err != nil {
			return nil, fmt.Errorf("failed to create long-term memory: %w", err)
		}

		builder = builder.WithLongTermMemory(longTermMemory, longTermConfig)
	}

	// Convert tools
	toolRegistry := b.componentManager.GetToolRegistry()
	for _, toolName := range agentCfg.Tools {
		tool, err := toolRegistry.GetTool(toolName)
		if err != nil {
			continue // Skip missing tools
		}
		builder = builder.WithTool(tool)
	}

	// Session service
	if agentCfg.SessionStore != "" {
		storeConfig, ok := b.config.SessionStores[agentCfg.SessionStore]
		if !ok {
			return nil, fmt.Errorf("session store '%s' not found", agentCfg.SessionStore)
		}

		sessionBuilder := NewSessionService(agentID)
		if storeConfig.Backend == "sql" {
			if storeConfig.SQL == nil {
				return nil, fmt.Errorf("SQL configuration is required for SQL session store")
			}
			sessionBuilder = sessionBuilder.Backend("sql").WithSQLConfig(storeConfig.SQL)
		} else {
			sessionBuilder = sessionBuilder.Backend("memory")
		}

		if storeConfig.RateLimit != nil {
			sessionBuilder = sessionBuilder.WithRateLimit(storeConfig.RateLimit)
		}

		builder = builder.WithSession(sessionBuilder)
	}

	// Context service (for RAG/document stores)
	if len(agentCfg.DocumentStores) > 0 {
		if agentCfg.Database == "" {
			return nil, fmt.Errorf("database is required when document stores are configured")
		}
		if agentCfg.Embedder == "" {
			return nil, fmt.Errorf("embedder is required when document stores are configured")
		}

		db, err := b.componentManager.GetDatabase(agentCfg.Database)
		if err != nil {
			return nil, fmt.Errorf("failed to get database: %w", err)
		}

		embedder, err := b.componentManager.GetEmbedder(agentCfg.Embedder)
		if err != nil {
			return nil, fmt.Errorf("failed to get embedder: %w", err)
		}

		contextBuilder := NewContextService().
			WithDatabase(db).
			WithEmbedder(embedder)

		// Set search config
		if agentCfg.Search.TopK > 0 {
			contextBuilder = contextBuilder.TopK(agentCfg.Search.TopK)
		}
		if agentCfg.Search.Threshold > 0 {
			contextBuilder = contextBuilder.Threshold(agentCfg.Search.Threshold)
		}
		if agentCfg.Search.PreserveCase != nil {
			contextBuilder = contextBuilder.PreserveCase(*agentCfg.Search.PreserveCase)
		}
		for _, model := range agentCfg.Search.Models {
			contextBuilder = contextBuilder.WithSearchModel(model)
		}

		// Add document stores
		var documentStoreConfigs []*config.DocumentStoreConfig
		for _, storeName := range agentCfg.DocumentStores {
			storeConfig, exists := b.config.DocumentStores[storeName]
			if !exists {
				return nil, fmt.Errorf("document store '%s' not found", storeName)
			}
			documentStoreConfigs = append(documentStoreConfigs, storeConfig)
		}
		contextBuilder = contextBuilder.WithDocumentStores(documentStoreConfigs)

		// Set IncludeContext from prompt config
		if agentCfg.Prompt.IncludeContext != nil {
			contextBuilder = contextBuilder.IncludeContext(*agentCfg.Prompt.IncludeContext)
		}

		builder = builder.WithContext(contextBuilder)
	} else if agentCfg.Prompt.IncludeContext != nil && *agentCfg.Prompt.IncludeContext {
		// IncludeContext is enabled but no document stores - still set it
		builder = builder.IncludeContext(true)
	}

	// Task service
	if agentCfg.Task != nil && agentCfg.Task.IsEnabled() {
		taskBuilder := NewTaskService().
			Backend(agentCfg.Task.Backend).
			WorkerPool(agentCfg.Task.WorkerPool).
			InputTimeout(agentCfg.Task.InputTimeout).
			Timeout(agentCfg.Task.Timeout)

		if agentCfg.Task.SQL != nil {
			taskBuilder = taskBuilder.WithSQLConfig(agentCfg.Task.SQL)
		}

		builder = builder.WithTask(taskBuilder)
	}

	// Build using programmatic API
	return builder.Build()
}

// BuildAllAgents builds all agents from config
func (b *ConfigAgentBuilder) BuildAllAgents() (map[string]*agent.Agent, error) {
	agents := make(map[string]*agent.Agent)

	// Build all agents first
	for agentID := range b.config.Agents {
		agentInstance, err := b.BuildAgent(agentID)
		if err != nil {
			return nil, fmt.Errorf("failed to build agent %s: %w", agentID, err)
		}
		agents[agentID] = agentInstance
	}

	// Register all agents in registry
	for agentID, agentInstance := range agents {
		agentCfg := b.config.Agents[agentID]
		if err := b.agentRegistry.RegisterAgent(agentID, agentInstance, agentCfg, nil); err != nil {
			return nil, fmt.Errorf("failed to register agent %s: %w", agentID, err)
		}
	}

	return agents, nil
}

// AgentRegistry returns the agent registry
func (b *ConfigAgentBuilder) AgentRegistry() *agent.AgentRegistry {
	return b.agentRegistry
}

// ComponentManager returns the component manager (for accessing LLMs, tools, etc.)
func (b *ConfigAgentBuilder) ComponentManager() *component.ComponentManager {
	return b.componentManager
}

// Config returns the underlying config
func (b *ConfigAgentBuilder) Config() *config.Config {
	return b.config
}
