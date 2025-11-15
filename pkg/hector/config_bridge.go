package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/a2a/pb"
	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/component"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/databases"
	"github.com/kadirpekel/hector/pkg/embedders"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/tools"
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
	// preferredTransport default is set in config.SetDefaults(), so it should already have a value
	preferredTransport := cfg.Global.A2AServer.PreferredTransport

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
func (b *ConfigAgentBuilder) BuildAgent(agentID string) (pb.A2AServiceServer, error) {
	agentCfg, ok := b.config.Agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent %s not found in config", agentID)
	}

	// Handle external A2A agents separately - they don't need LLM
	if agentCfg.Type == "a2a" {
		externalAgent, err := agent.NewExternalA2AAgent(agentID, agentCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create external A2A agent: %w", err)
		}
		return externalAgent, nil
	}

	// Determine preferred transport: agent-level A2A override > global > default
	preferredTransport := b.preferredTransport
	if agentCfg.A2A != nil && agentCfg.A2A.PreferredTransport != "" {
		preferredTransport = agentCfg.A2A.PreferredTransport
	}

	// Use programmatic API to build agent
	builder := NewAgent(agentID).
		WithName(agentCfg.Name).
		WithDescription(agentCfg.Description).
		WithRegistry(b.agentRegistry).
		WithBaseURL(b.baseURL).
		WithPreferredTransport(preferredTransport).
		WithComponentManager(b.componentManager)

	// Set visibility (default is "public" per config.SetDefaults())
	// Builder also defaults to "public", but we respect config value
	builder = builder.WithVisibility(agentCfg.Visibility)

	// Get LLM provider from component manager (required for native agents)
	if agentCfg.LLM == "" {
		return nil, fmt.Errorf("LLM is required for native agents")
	}
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
	if agentCfg.Reasoning.EnableToolDisplay != nil {
		reasoningBuilder = reasoningBuilder.ShowTools(*agentCfg.Reasoning.EnableToolDisplay)
	}
	if agentCfg.Reasoning.EnableThinkingDisplay != nil {
		reasoningBuilder = reasoningBuilder.ShowThinking(*agentCfg.Reasoning.EnableThinkingDisplay)
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
	// Note: If long-term memory is enabled, validation ensures agent vector_store/embedder are set
	if agentCfg.Memory.LongTerm.IsEnabled() {
		if agentCfg.VectorStore == "" {
			return nil, fmt.Errorf("vector_store is required when long-term memory is enabled")
		}
		if agentCfg.Embedder == "" {
			return nil, fmt.Errorf("embedder is required when long-term memory is enabled")
		}

		db, err := b.componentManager.GetDatabase(agentCfg.VectorStore)
		if err != nil {
			return nil, fmt.Errorf("failed to get vector store: %w", err)
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
			AutoRecall(boolValue(agentCfg.Memory.LongTerm.EnableAutoRecall, false)).
			RecallLimit(agentCfg.Memory.LongTerm.RecallLimit).
			WithDatabase(db).
			WithEmbedder(embedder)

		longTermMemory, longTermConfig, err := longTermBuilder.Build()
		if err != nil {
			return nil, fmt.Errorf("failed to create long-term memory: %w", err)
		}

		builder = builder.WithLongTermMemory(longTermMemory, longTermConfig)
	}

	// Convert sub-agents
	if len(agentCfg.SubAgents) > 0 {
		builder = builder.WithSubAgents(agentCfg.SubAgents)
	}

	// Convert tools
	toolRegistry := b.componentManager.GetToolRegistry()
	defaultToolConfigs := config.GetDefaultToolConfigs()

	// Determine which tools to add to the agent:
	// - If Tools is nil: use all tools from registry (matches old behavior where allowedTools=nil meant "all tools")
	//   This happens when:
	//   * EnableTools=true (expandEnableTools() adds tools to cfg.Tools, but agentCfg.Tools remains nil)
	//   * MCP_URL is set (MCP tools are registered in registry, agentCfg.Tools is set to nil)
	// - If Tools is empty slice: no tools are added (explicit empty list)
	// - If Tools has items: only those specific tools are added
	var toolsToAdd []string
	if agentCfg.Tools == nil {
		// Use all tools from registry (includes MCP tools, default tools from EnableTools, etc.)
		allTools := toolRegistry.ListTools()
		for _, toolInfo := range allTools {
			toolsToAdd = append(toolsToAdd, toolInfo.Name)
		}
	} else {
		// Use explicitly listed tools
		toolsToAdd = agentCfg.Tools
	}

	for _, toolName := range toolsToAdd {
		tool, err := toolRegistry.GetTool(toolName)
		if err != nil {
			// Try to create native tool with default config if it's a known native tool
			if defaultConfig, isNativeTool := defaultToolConfigs[toolName]; isNativeTool {
				// Create tool from default config
				var createdTool tools.Tool
				var createErr error

				switch defaultConfig.Type {
				case "command":
					createdTool, createErr = tools.NewCommandToolWithConfig(toolName, defaultConfig)
				case "write_file":
					createdTool, createErr = tools.NewFileWriterToolWithConfig(toolName, defaultConfig)
				case "search_replace":
					createdTool, createErr = tools.NewSearchReplaceToolWithConfig(toolName, defaultConfig)
				case "read_file":
					createdTool, createErr = tools.NewReadFileToolWithConfig(toolName, defaultConfig)
				case "apply_patch":
					createdTool, createErr = tools.NewApplyPatchToolWithConfig(toolName, defaultConfig)
				case "grep_search":
					createdTool, createErr = tools.NewGrepSearchToolWithConfig(toolName, defaultConfig)
				case "web_request":
					createdTool, createErr = tools.NewWebRequestToolWithConfig(toolName, defaultConfig)
				case "todo":
					createdTool = tools.NewTodoTool()
				case "agent_call":
					// agent_call requires agent registry
					if b.agentRegistry != nil {
						createdTool = tools.NewAgentCallTool(b.agentRegistry)
					} else {
						createErr = fmt.Errorf("agent_call tool requires agent registry")
					}
				default:
					createErr = fmt.Errorf("unknown native tool type: %s", defaultConfig.Type)
				}

				if createErr == nil && createdTool != nil {
					builder = builder.WithTool(createdTool)
					continue
				}
			}

			// Skip missing tools that couldn't be auto-created
			continue
		}
		builder = builder.WithTool(tool)
	}

	// Auto-add agent_call tool if sub-agents are present but tool wasn't explicitly added
	// This ensures sub-agents can be called even if agent_call wasn't in the tools list
	if len(agentCfg.SubAgents) > 0 && b.agentRegistry != nil {
		// Check if agent_call was already added (either in tools list or registry)
		hasAgentCall := false
		for _, toolName := range toolsToAdd {
			if toolName == "agent_call" {
				hasAgentCall = true
				break
			}
		}
		// Also check if it's in the registry
		if !hasAgentCall {
			if _, err := toolRegistry.GetTool("agent_call"); err != nil {
				// Not in registry either, create it using default config (for consistency)
				if defaultConfig, ok := defaultToolConfigs["agent_call"]; ok && defaultConfig.Type == "agent_call" {
					agentCallTool := tools.NewAgentCallTool(b.agentRegistry)
					builder = builder.WithTool(agentCallTool)
				}
			}
		}
	}

	// Session service
	if agentCfg.SessionStore != "" {
		storeConfig, ok := b.config.SessionStores[agentCfg.SessionStore]
		if !ok {
			return nil, fmt.Errorf("session store '%s' not found", agentCfg.SessionStore)
		}

		sessionBuilder := NewSessionService(agentID).WithComponentManager(b.componentManager)
		if storeConfig.Backend == "sql" {
			if storeConfig.SQLDatabase == "" {
				return nil, fmt.Errorf("sql_database reference is required for SQL session store")
			}
			sessionBuilder = sessionBuilder.Backend("sql").Database(storeConfig.SQLDatabase)
		} else {
			sessionBuilder = sessionBuilder.Backend("memory")
		}

		if storeConfig.RateLimit != nil {
			sessionBuilder = sessionBuilder.WithRateLimit(storeConfig.RateLimit)
		}

		builder = builder.WithSession(sessionBuilder)
	}

	// Context service (for RAG/document stores)
	// Check if agent vector_store/embedder is needed
	needsAgentVectorStore := false
	needsAgentEmbedder := false

	// Check if IncludeContext is enabled (requires vector_store/embedder for RAG)
	includeContextEnabled := agentCfg.Prompt.IncludeContext != nil && *agentCfg.Prompt.IncludeContext

	if len(agentCfg.DocumentStores) > 0 {
		// Check if all document stores have their own vector_store/embedder
		allStoresHaveVectorStore := true
		allStoresHaveEmbedder := true

		for _, storeName := range agentCfg.DocumentStores {
			storeConfig, exists := b.config.DocumentStores[storeName]
			if !exists {
				return nil, fmt.Errorf("document store '%s' not found", storeName)
			}
			if storeConfig.VectorStore == "" {
				allStoresHaveVectorStore = false
			}
			if storeConfig.Embedder == "" {
				allStoresHaveEmbedder = false
			}
		}

		// Agent vector_store/embedder needed if:
		// 1. IncludeContext is enabled (for RAG), OR
		// 2. Not all stores have their own vector_store/embedder (as fallback)
		needsAgentVectorStore = includeContextEnabled || !allStoresHaveVectorStore
		needsAgentEmbedder = includeContextEnabled || !allStoresHaveEmbedder
	} else if includeContextEnabled {
		// IncludeContext enabled but no document stores - still needs vector_store/embedder for RAG
		needsAgentVectorStore = true
		needsAgentEmbedder = true
	}

	// Validate requirements
	if needsAgentVectorStore && agentCfg.VectorStore == "" {
		return nil, fmt.Errorf("vector_store is required when: IncludeContext is enabled, or document stores are configured and at least one doesn't specify its own vector_store")
	}
	if needsAgentEmbedder && agentCfg.Embedder == "" {
		return nil, fmt.Errorf("embedder is required when: IncludeContext is enabled, or document stores are configured and at least one doesn't specify its own embedder")
	}

	// Only create context service if document stores are configured
	if len(agentCfg.DocumentStores) > 0 {
		var db databases.DatabaseProvider
		var embedder embedders.EmbedderProvider
		var err error

		// Get vector store (required if needed)
		if needsAgentVectorStore {
			db, err = b.componentManager.GetDatabase(agentCfg.VectorStore)
			if err != nil {
				return nil, fmt.Errorf("failed to get vector store: %w", err)
			}
		}

		// Get embedder (required if needed)
		if needsAgentEmbedder {
			embedder, err = b.componentManager.GetEmbedder(agentCfg.Embedder)
			if err != nil {
				return nil, fmt.Errorf("failed to get embedder: %w", err)
			}
		}

		// If vector_store/embedder not needed (all stores have their own and IncludeContext disabled),
		// we still need them for the context builder (it requires vector_store/embedder instances).
		// Use first store's vector_store/embedder since all stores should have their own at this point.
		if !needsAgentVectorStore {
			// All stores have their own vector_store - use first one for context builder
			firstStoreName := agentCfg.DocumentStores[0]
			firstStoreConfig := b.config.DocumentStores[firstStoreName]
			if firstStoreConfig.VectorStore == "" {
				return nil, fmt.Errorf("internal error: store '%s' should have vector_store specified", firstStoreName)
			}
			db, err = b.componentManager.GetDatabase(firstStoreConfig.VectorStore)
			if err != nil {
				return nil, fmt.Errorf("failed to get vector store from store '%s': %w", firstStoreName, err)
			}
		}

		if !needsAgentEmbedder {
			// All stores have their own embedder - use first one for context builder
			firstStoreName := agentCfg.DocumentStores[0]
			firstStoreConfig := b.config.DocumentStores[firstStoreName]
			if firstStoreConfig.Embedder == "" {
				return nil, fmt.Errorf("internal error: store '%s' should have embedder specified", firstStoreName)
			}
			embedder, err = b.componentManager.GetEmbedder(firstStoreConfig.Embedder)
			if err != nil {
				return nil, fmt.Errorf("failed to get embedder from store '%s': %w", firstStoreName, err)
			}
		}

		contextBuilder := NewContextService().
			WithDatabase(db).
			WithEmbedder(embedder).
			WithComponentManager(b.componentManager)

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
	} else if includeContextEnabled {
		// IncludeContext is enabled but no document stores
		// This requires vector_store/embedder for RAG
		if agentCfg.VectorStore == "" {
			return nil, fmt.Errorf("vector_store is required when IncludeContext is enabled")
		}
		if agentCfg.Embedder == "" {
			return nil, fmt.Errorf("embedder is required when IncludeContext is enabled")
		}

		// Note: Without document stores, IncludeContext won't have anything to search
		// but we still validate the requirement for consistency
		builder = builder.IncludeContext(true)
	}

	// Task service
	if agentCfg.Task != nil && agentCfg.Task.IsEnabled() {
		taskBuilder := NewTaskService().
			Backend(agentCfg.Task.Backend).
			WorkerPool(agentCfg.Task.WorkerPool).
			InputTimeout(agentCfg.Task.InputTimeout).
			Timeout(agentCfg.Task.Timeout).
			WithComponentManager(b.componentManager)

		if agentCfg.Task.SQLDatabase != "" {
			taskBuilder = taskBuilder.Database(agentCfg.Task.SQLDatabase)
		}

		// Add HITL configuration
		if agentCfg.Task.HITL != nil {
			taskBuilder = taskBuilder.WithHITL(agentCfg.Task.HITL)
		}

		// Add checkpoint configuration (using flattened fields)
		if agentCfg.Task.EnableCheckpointing != nil && *agentCfg.Task.EnableCheckpointing {
			// Build CheckpointConfig from flattened fields for builder compatibility
			checkpointCfg := &config.CheckpointConfig{
				Enabled:  agentCfg.Task.EnableCheckpointing,
				Strategy: agentCfg.Task.CheckpointStrategy,
			}
			if agentCfg.Task.CheckpointInterval > 0 || agentCfg.Task.CheckpointAfterTools != nil || agentCfg.Task.CheckpointBeforeLLM != nil {
				checkpointCfg.Interval = &config.CheckpointIntervalConfig{
					EveryNIterations: agentCfg.Task.CheckpointInterval,
					AfterToolCalls:   agentCfg.Task.CheckpointAfterTools,
					BeforeLLMCalls:   agentCfg.Task.CheckpointBeforeLLM,
				}
			}
			if agentCfg.Task.AutoResume != nil || agentCfg.Task.AutoResumeHITL != nil || agentCfg.Task.ResumeTimeout > 0 {
				checkpointCfg.Recovery = &config.CheckpointRecoveryConfig{
					AutoResume:     agentCfg.Task.AutoResume,
					AutoResumeHITL: agentCfg.Task.AutoResumeHITL,
					ResumeTimeout:  agentCfg.Task.ResumeTimeout,
				}
			}
			taskBuilder = taskBuilder.WithCheckpoint(checkpointCfg)
		}

		builder = builder.WithTask(taskBuilder)
	}

	// Set security configuration if present
	if agentCfg.Security != nil {
		builder = builder.WithSecurity(agentCfg.Security)
	}

	// Set A2A card configuration if present
	if agentCfg.A2A != nil {
		builder = builder.WithA2ACard(agentCfg.A2A)
	}

	// Set structured output configuration if present
	if agentCfg.StructuredOutput != nil {
		builder = builder.WithStructuredOutput(agentCfg.StructuredOutput)
	}

	// Build using programmatic API
	agentInstance, err := builder.Build()
	if err != nil {
		return nil, err
	}

	// Note: The config (A2A, Security, StructuredOutput, Task) is now set on the agent
	// via the builder, which passes it to NewAgentDirect. This ensures GetAgentCard()
	// and other methods can access these fields from agent.config.
	// Sub-agents are stored directly on the Agent struct (not in config) via WithSubAgents()

	// Validate HITL configuration
	if agentCfg.Task != nil && agentCfg.Task.HITL != nil {
		hasSessionStore := agentCfg.SessionStore != ""
		hitlMode := agentCfg.Task.HITL.Mode
		if hitlMode == "" {
			hitlMode = "auto"
		}

		if hitlMode == "async" && !hasSessionStore {
			return nil, fmt.Errorf("agent %s: async HITL requires session_store to be configured", agentID)
		}
	}

	// *agent.Agent implements pb.A2AServiceServer, so we can return it directly
	return agentInstance, nil
}

// BuildAllAgents builds all agents from config
func (b *ConfigAgentBuilder) BuildAllAgents() (map[string]pb.A2AServiceServer, error) {
	agents := make(map[string]pb.A2AServiceServer)

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
