package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"time"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/verikod/hector/pkg/agent"
	"github.com/verikod/hector/pkg/agent/llmagent"
	"github.com/verikod/hector/pkg/agent/remoteagent"
	"github.com/verikod/hector/pkg/agent/runneragent"
	"github.com/verikod/hector/pkg/agent/workflowagent"
	"github.com/verikod/hector/pkg/checkpoint"
	"github.com/verikod/hector/pkg/config"
	"github.com/verikod/hector/pkg/embedder"
	"github.com/verikod/hector/pkg/guardrails"
	grInput "github.com/verikod/hector/pkg/guardrails/input"
	"github.com/verikod/hector/pkg/guardrails/moderation"
	grOutput "github.com/verikod/hector/pkg/guardrails/output"
	grTool "github.com/verikod/hector/pkg/guardrails/tool"
	"github.com/verikod/hector/pkg/memory"
	"github.com/verikod/hector/pkg/model"
	"github.com/verikod/hector/pkg/notification"
	"github.com/verikod/hector/pkg/observability"
	"github.com/verikod/hector/pkg/rag"
	"github.com/verikod/hector/pkg/session"
	"github.com/verikod/hector/pkg/tool"
	"github.com/verikod/hector/pkg/tool/agenttool"
	"github.com/verikod/hector/pkg/tool/mcptoolset"
	"github.com/verikod/hector/pkg/tool/searchtool"
	"github.com/verikod/hector/pkg/trigger"
	"github.com/verikod/hector/pkg/vector"
)

// Builder is the foundational API for constructing Runtime instances.
// It uses a fluent interface pattern for programmatic configuration.
// Config-based loading is syntactic sugar that internally uses Builder.
type Builder struct {
	// Optional AppID override (default: cfg.Name)
	appID string
	cfg   *config.AppConfig

	// Core components (user-provided or built from config)
	llms            map[string]model.LLM
	embedders       map[string]embedder.Embedder
	toolsets        map[string]tool.Toolset
	agents          map[string]agent.Agent
	vectorProviders map[string]vector.Provider
	documentStores  map[string]*rag.DocumentStore

	// Services
	sessions      session.Service
	index         memory.IndexService
	checkpoint    *checkpoint.Manager
	observability *observability.Manager

	// Shared resources
	dbPool      *config.DBPool
	databaseDSN string // Database DSN for session/task services

	// Factories for config-based building
	llmFactory      LLMFactory
	embedderFactory EmbedderFactory
	toolsetFactory  ToolsetFactory

	// Multi-agent relationships
	subAgents   map[string][]agent.Agent
	agentTools  map[string][]agent.Agent
	directTools map[string][]tool.Tool
}

// NewBuilder creates a new Builder instance.
func NewBuilder() *Builder {
	return &Builder{
		llms:            make(map[string]model.LLM),
		embedders:       make(map[string]embedder.Embedder),
		toolsets:        make(map[string]tool.Toolset),
		agents:          make(map[string]agent.Agent),
		vectorProviders: make(map[string]vector.Provider),
		documentStores:  make(map[string]*rag.DocumentStore),
		subAgents:       make(map[string][]agent.Agent),
		agentTools:      make(map[string][]agent.Agent),
		directTools:     make(map[string][]tool.Tool),
		llmFactory:      DefaultLLMFactory,
		embedderFactory: DefaultEmbedderFactory,
		toolsetFactory:  DefaultToolsetFactory,
	}
}

// WithAppID sets the application ID.
func (b *Builder) WithAppID(id string) *Builder {
	b.appID = id
	return b
}

// ---------- Programmatic API (Primary Interface) ----------

// WithLLM registers an LLM instance.
func (b *Builder) WithLLM(name string, llm model.LLM) *Builder {
	b.llms[name] = llm
	return b
}

// WithEmbedder registers an embedder instance.
func (b *Builder) WithEmbedder(name string, emb embedder.Embedder) *Builder {
	b.embedders[name] = emb
	return b
}

// WithToolset registers a toolset instance.
func (b *Builder) WithToolset(name string, ts tool.Toolset) *Builder {
	b.toolsets[name] = ts
	return b
}

// WithAgent registers an agent instance.
func (b *Builder) WithAgent(name string, ag agent.Agent) *Builder {
	b.agents[name] = ag
	return b
}

// WithVectorProvider registers a vector provider instance.
func (b *Builder) WithVectorProvider(name string, vp vector.Provider) *Builder {
	b.vectorProviders[name] = vp
	return b
}

// WithDocumentStore registers a document store instance.
func (b *Builder) WithDocumentStore(name string, ds *rag.DocumentStore) *Builder {
	b.documentStores[name] = ds
	return b
}

// WithSessionService sets the session service.
func (b *Builder) WithSessionService(svc session.Service) *Builder {
	b.sessions = svc
	return b
}

// WithIndexService sets the index service.
func (b *Builder) WithIndexService(idx memory.IndexService) *Builder {
	b.index = idx
	return b
}

// WithCheckpointManager sets the checkpoint manager.
func (b *Builder) WithCheckpointManager(mgr *checkpoint.Manager) *Builder {
	b.checkpoint = mgr
	return b
}

// WithObservability sets the observability manager.
func (b *Builder) WithObservability(obs *observability.Manager) *Builder {
	b.observability = obs
	return b
}

// WithDBPool sets the shared database pool.
func (b *Builder) WithDBPool(pool *config.DBPool) *Builder {
	b.dbPool = pool
	return b
}

// WithDatabaseDSN sets the database DSN for session/task services.
func (b *Builder) WithDatabaseDSN(dsn string) *Builder {
	b.databaseDSN = dsn
	return b
}

// WithSubAgents adds sub-agents for an agent (Pattern 1: transfer).
func (b *Builder) WithSubAgents(agentName string, subAgents []agent.Agent) *Builder {
	b.subAgents[agentName] = append(b.subAgents[agentName], subAgents...)
	return b
}

// WithAgentTools adds agents as tools for an agent (Pattern 2: delegation).
func (b *Builder) WithAgentTools(agentName string, agentTools []agent.Agent) *Builder {
	b.agentTools[agentName] = append(b.agentTools[agentName], agentTools...)
	return b
}

// WithDirectTools adds tools directly for an agent.
func (b *Builder) WithDirectTools(agentName string, tools []tool.Tool) *Builder {
	b.directTools[agentName] = append(b.directTools[agentName], tools...)
	return b
}

// ---------- Factory Configuration ----------

// WithLLMFactory sets a custom LLM factory for config-based building.
func (b *Builder) WithLLMFactory(f LLMFactory) *Builder {
	b.llmFactory = f
	return b
}

// WithEmbedderFactory sets a custom embedder factory.
func (b *Builder) WithEmbedderFactory(f EmbedderFactory) *Builder {
	b.embedderFactory = f
	return b
}

// WithToolsetFactory sets a custom toolset factory.
func (b *Builder) WithToolsetFactory(f ToolsetFactory) *Builder {
	b.toolsetFactory = f
	return b
}

// ---------- Config-Based Building (Convenience) ----------

// WithConfig sets up the builder from a config file.
// This is syntactic sugar - internally uses the programmatic API.
func (b *Builder) WithConfig(cfg *config.AppConfig) *Builder {
	b.cfg = cfg
	return b
}

// Build creates an immutable Runtime from the builder state.
// This is the ONLY way to create a Runtime.
func (b *Builder) Build() (*Runtime, error) {
	// If config was provided, build components from config first
	if b.cfg != nil {
		if err := b.buildFromConfig(); err != nil {
			return nil, err
		}
	}

	// Validate we have minimum requirements
	if len(b.agents) == 0 && (b.cfg == nil || len(b.cfg.Agents) == 0) {
		return nil, fmt.Errorf("at least one agent is required")
	}

	// Create scheduler for scheduled triggers
	scheduler := b.createScheduler()

	// Create notifier for outbound notifications
	notifier := notification.New(b.cfg, b.observability.Metrics())
	if notifier != nil {
		slog.Info("Notifications enabled")
	}

	// Build the immutable runtime
	rt := &Runtime{
		AppID:           b.appID,
		cfg:             b.cfg,
		llms:            b.llms,
		embedders:       b.embedders,
		toolsets:        b.toolsets,
		agents:          b.agents,
		vectorProviders: b.vectorProviders,
		documentStores:  b.documentStores,
		sessions:        b.sessions,
		index:           b.index,
		checkpoint:      b.checkpoint,
		observability:   b.observability,
		scheduler:       scheduler,
		notifier:        notifier,
		dbPool:          b.dbPool,
	}

	// Fallback for AppID
	if rt.AppID == "" && rt.cfg != nil {
		rt.AppID = rt.cfg.Name
	}

	slog.Info("Runtime built successfully",
		"llms", len(rt.llms),
		"agents", len(rt.agents),
		"toolsets", len(rt.toolsets))
	return rt, nil
}

// buildFromConfig populates the builder from config.
// This uses the factories to create components.
func (b *Builder) buildFromConfig() error {
	cfg := b.cfg
	if err := config.ValidateAppConfig(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	cfg.SetDefaults()

	// Observability removed from AppConfig - should be passed via ServerConfig
	// Temporarily disabled during refactoring

	// Create session service from config
	if b.sessions == nil {
		if b.dbPool != nil && b.databaseDSN != "" {
			sessionSvc, err := session.NewSessionService(b.dbPool, b.databaseDSN)
			if err != nil {
				return fmt.Errorf("failed to create session service: %w", err)
			}
			b.sessions = sessionSvc
		} else {
			// Default to in-memory session service suitable for library usage
			b.sessions = session.InMemoryService()
		}
	}

	// Build LLMs
	if err := b.buildLLMsFromConfig(); err != nil {
		return fmt.Errorf("failed to build LLMs: %w", err)
	}

	// Build embedders
	if err := b.buildEmbeddersFromConfig(); err != nil {
		return fmt.Errorf("failed to build embedders: %w", err)
	}

	// Build vector providers
	if err := b.buildVectorProvidersFromConfig(); err != nil {
		return fmt.Errorf("failed to build vector providers: %w", err)
	}

	// Build toolsets (before document stores - MCP extractors need tool access)
	if err := b.buildToolsetsFromConfig(); err != nil {
		return fmt.Errorf("failed to build toolsets: %w", err)
	}

	// Build document stores
	if err := b.buildDocumentStoresFromConfig(); err != nil {
		return fmt.Errorf("failed to build document stores: %w", err)
	}

	// Create index service
	if b.index == nil {
		indexSvc, err := memory.NewIndexServiceFromConfig(cfg, b.embedders)
		if err != nil {
			return fmt.Errorf("failed to create index service: %w", err)
		}
		b.index = indexSvc
	}

	// Create checkpoint manager (zero-config)
	if b.checkpoint == nil && b.dbPool != nil {
		cpMgr, err := b.buildCheckpointManagerZeroConfig()
		if err != nil {
			return fmt.Errorf("failed to create checkpoint manager: %w", err)
		}
		b.checkpoint = cpMgr
	}

	// Build agents
	if err := b.buildAgentsFromConfig(); err != nil {
		return fmt.Errorf("failed to build agents: %w", err)
	}

	return nil
}

// buildLLMsFromConfig creates LLMs from config.
func (b *Builder) buildLLMsFromConfig() error {
	for name, llmCfg := range b.cfg.LLMs {
		if llmCfg == nil {
			continue
		}
		// Skip if already provided programmatically
		if _, exists := b.llms[name]; exists {
			continue
		}
		llm, err := b.llmFactory(llmCfg)
		if err != nil {
			return fmt.Errorf("llm %q: %w", name, err)
		}
		b.llms[name] = llm
	}
	return nil
}

// buildEmbeddersFromConfig creates embedders from config.
func (b *Builder) buildEmbeddersFromConfig() error {
	for name, embCfg := range b.cfg.Embedders {
		if embCfg == nil {
			continue
		}
		if _, exists := b.embedders[name]; exists {
			continue
		}
		emb, err := b.embedderFactory(embCfg)
		if err != nil {
			return fmt.Errorf("embedder %q: %w", name, err)
		}
		b.embedders[name] = emb
	}
	return nil
}

// buildVectorProvidersFromConfig creates vector providers from config.
func (b *Builder) buildVectorProvidersFromConfig() error {
	for name, vpCfg := range b.cfg.VectorStores {
		if vpCfg == nil {
			continue
		}
		if _, exists := b.vectorProviders[name]; exists {
			continue
		}

		// Create a copy to mutate for isolation without affecting original config
		// This is important because b.cfg might be reused or inspected later
		isolatedCfg := *vpCfg

		// Apply isolation if we have a specific AppID (and it's not the default system app)
		// We skip "default" to keep backward compatible paths/collections for the main app
		if b.appID != "" && b.appID != "default" {
			if isolatedCfg.IsEmbedded() {
				// For embedded (chromem), scope the ID persistence path
				// from: .hector/vectors
				// to:   .hector/vectors/app_id
				if isolatedCfg.PersistPath != "" {
					isolatedCfg.PersistPath = filepath.Join(isolatedCfg.PersistPath, b.appID)
				}
			} else {
				// For external stores, scope the collection name
				// from: documents
				// to:   app_id_documents
				if isolatedCfg.Collection != "" {
					isolatedCfg.Collection = fmt.Sprintf("%s_%s", b.appID, isolatedCfg.Collection)
				} else {
					// If no collection specified, assert one based on app ID
					isolatedCfg.Collection = fmt.Sprintf("%s_default", b.appID)
				}
			}
		}

		vp, err := rag.NewVectorProviderFromConfig(&isolatedCfg)
		if err != nil {
			return fmt.Errorf("vector_store %q: %w", name, err)
		}
		b.vectorProviders[name] = vp
	}
	return nil
}

// buildToolsetsFromConfig creates toolsets from config.
func (b *Builder) buildToolsetsFromConfig() error {
	for name, toolCfg := range b.cfg.Tools {
		if toolCfg == nil {
			continue
		}
		if _, exists := b.toolsets[name]; exists {
			continue
		}
		ts, err := b.toolsetFactory(name, toolCfg)
		if err != nil {
			return fmt.Errorf("tool %q: %w", name, err)
		}
		b.toolsets[name] = ts
	}
	return nil
}

// buildDocumentStoresFromConfig creates document stores from config.
func (b *Builder) buildDocumentStoresFromConfig() error {
	if len(b.cfg.DocumentStores) == 0 {
		return nil
	}

	// Create ToolCaller adapter for MCP extractors
	var toolCaller rag.ToolCaller
	if len(b.toolsets) > 0 {
		toolCaller = &toolCallerAdapter{toolsets: b.toolsets}
	}

	deps := &rag.FactoryDeps{
		DBPool:          b.dbPool,
		VectorProviders: b.vectorProviders,
		Embedders:       b.embedders,
		LLMs:            b.llms,
		ToolCaller:      toolCaller,
		// Config field removed during refactoring
	}

	for name, storeCfg := range b.cfg.DocumentStores {
		if storeCfg == nil {
			continue
		}
		if _, exists := b.documentStores[name]; exists {
			continue
		}

		// Create a copy to mutate for isolation without affecting original config
		isolatedCfg := *storeCfg

		// Apply isolation if we have a specific AppID (and it's not the default system app)
		if b.appID != "" && b.appID != "default" {
			// Scope collection override if present
			if isolatedCfg.Collection != "" {
				isolatedCfg.Collection = fmt.Sprintf("%s_%s", b.appID, isolatedCfg.Collection)
			}
		}

		store, err := rag.NewDocumentStoreFromConfig(name, &isolatedCfg, deps)
		if err != nil {
			return fmt.Errorf("document_store %q: %w", name, err)
		}
		b.documentStores[name] = store
	}
	return nil
}

// buildAgentsFromConfig creates agents from config using multi-pass algorithm.
func (b *Builder) buildAgentsFromConfig() error {
	// This reuses the existing multi-pass logic but builds into b.agents
	// The key difference: all dependencies (llms, toolsets) are already in builder

	type pendingAgent struct {
		Name   string
		Config *config.AgentConfig
	}

	var pending []pendingAgent
	for name, cfg := range b.cfg.Agents {
		if cfg == nil {
			continue
		}
		// Skip if already provided programmatically
		if _, exists := b.agents[name]; exists {
			continue
		}
		pending = append(pending, pendingAgent{Name: name, Config: cfg})
	}

	// Multi-pass build loop
	for len(pending) > 0 {
		var nextPending []pendingAgent
		progress := false

		for _, p := range pending {
			name := p.Name
			cfg := p.Config

			// Check dependencies for workflow agents
			ready := true
			if isWorkflowAgentType(cfg.Type) {
				// Check SubAgents dependencies (sequence, parallel, loop)
				for _, subName := range cfg.SubAgents {
					if _, exists := b.agents[subName]; !exists {
						ready = false
						break
					}
				}

				// Check conditional agent dependencies
				if ready && cfg.Type == "conditional" {
					if cfg.ConditionAgent != "" {
						if _, exists := b.agents[cfg.ConditionAgent]; !exists {
							ready = false
						}
					}
					if ready && cfg.OnTrueAgent != "" {
						if _, exists := b.agents[cfg.OnTrueAgent]; !exists {
							ready = false
						}
					}
				}
			}

			if !ready {
				nextPending = append(nextPending, p)
				continue
			}

			// Build the agent
			ag, err := b.buildSingleAgent(name, cfg)
			if err != nil {
				return fmt.Errorf("failed to build agent %q: %w", name, err)
			}

			b.agents[name] = ag
			progress = true
		}

		if !progress && len(nextPending) > 0 {
			var missing []string
			for _, p := range nextPending {
				missing = append(missing, p.Name)
			}
			return fmt.Errorf("dependency cycle or missing dependencies: %v", missing)
		}

		pending = nextPending
	}

	// Pass 2: Wire up multi-agent relationships and rebuild
	for name, cfg := range b.cfg.Agents {
		if cfg == nil || isWorkflowAgentType(cfg.Type) {
			continue
		}

		// Collect sub-agents from config
		for _, subName := range cfg.SubAgents {
			if subAgent, ok := b.agents[subName]; ok {
				b.subAgents[name] = append(b.subAgents[name], subAgent)
			}
		}

		// Collect agent tools from config
		for _, agToolName := range cfg.AgentTools {
			if agTool, ok := b.agents[agToolName]; ok {
				b.agentTools[name] = append(b.agentTools[name], agTool)
			}
		}
	}

	// Pass 3: Rebuild LLM agents with relationships
	for name, cfg := range b.cfg.Agents {
		if cfg == nil || isWorkflowAgentType(cfg.Type) || isRemoteAgentType(cfg.Type) {
			continue
		}

		if len(cfg.SubAgents) == 0 && len(cfg.AgentTools) == 0 {
			continue
		}

		// Rebuild with relationships
		ag, err := b.buildSingleAgent(name, cfg)
		if err != nil {
			return fmt.Errorf("failed to rebuild agent %q with relationships: %w", name, err)
		}
		b.agents[name] = ag
	}

	slog.Info("Agents built successfully", "count", len(b.agents))
	return nil
}

// buildSingleAgent creates a single agent from config.
func (b *Builder) buildSingleAgent(name string, cfg *config.AgentConfig) (agent.Agent, error) {
	if isWorkflowAgentType(cfg.Type) {
		return b.buildWorkflowAgent(name, cfg)
	}
	if isRemoteAgentType(cfg.Type) {
		return b.buildRemoteAgent(name, cfg)
	}
	return b.buildLLMAgent(name, cfg)
}

// buildWorkflowAgent creates a workflow agent.
func (b *Builder) buildWorkflowAgent(name string, cfg *config.AgentConfig) (agent.Agent, error) {
	// Resolve sub-agents
	var subAgents []agent.Agent
	for _, subName := range cfg.SubAgents {
		if sub, ok := b.agents[subName]; ok {
			subAgents = append(subAgents, sub)
		}
	}

	switch cfg.Type {
	case "sequential":
		return workflowagent.NewSequential(workflowagent.SequentialConfig{
			Name:        name,
			Description: cfg.Description,
			SubAgents:   subAgents,
		})
	case "parallel":
		return workflowagent.NewParallel(workflowagent.ParallelConfig{
			Name:        name,
			Description: cfg.Description,
			SubAgents:   subAgents,
		})
	case "loop":
		return workflowagent.NewLoop(workflowagent.LoopConfig{
			Name:          name,
			Description:   cfg.Description,
			SubAgents:     subAgents,
			MaxIterations: cfg.MaxIterations,
		})
	case "runner":
		// Collect tools for runner
		var tools []tool.Tool
		for _, toolName := range cfg.Tools {
			ts, err := b.resolveToolset(toolName)
			if err != nil {
				return nil, fmt.Errorf("runner agent %q: %w", name, err)
			}
			resolvedTools, err := ts.Tools(nil)
			if err != nil {
				return nil, fmt.Errorf("runner agent %q: failed to resolve tools: %w", name, err)
			}
			tools = append(tools, resolvedTools...)
		}
		return runneragent.New(runneragent.Config{
			Name:        name,
			Description: cfg.Description,
			Tools:       tools,
		})
	case "conditional":
		// Resolve condition agent
		var conditionAgent agent.Agent
		if cfg.ConditionAgent != "" {
			if ag, ok := b.agents[cfg.ConditionAgent]; ok {
				conditionAgent = ag
			} else {
				return nil, fmt.Errorf("conditional agent %q: condition_agent %q not found", name, cfg.ConditionAgent)
			}
		}

		// Resolve on_true agent
		var onTrueAgent agent.Agent
		if cfg.OnTrueAgent != "" {
			if ag, ok := b.agents[cfg.OnTrueAgent]; ok {
				onTrueAgent = ag
			} else {
				return nil, fmt.Errorf("conditional agent %q: on_true_agent %q not found", name, cfg.OnTrueAgent)
			}
		}

		// Default on_false_response
		onFalseResponse := cfg.OnFalseResponse
		if onFalseResponse == "" {
			onFalseResponse = "I cannot process this request."
		}

		return workflowagent.NewConditional(workflowagent.ConditionalConfig{
			Name:            name,
			Description:     cfg.Description,
			ConditionAgent:  conditionAgent,
			ConditionField:  cfg.ConditionField,
			OnTrueAgent:     onTrueAgent,
			OnFalseResponse: onFalseResponse,
		})
	default:
		return nil, fmt.Errorf("unknown workflow agent type: %s", cfg.Type)
	}
}

// buildRemoteAgent creates a remote A2A agent.
func (b *Builder) buildRemoteAgent(name string, cfg *config.AgentConfig) (agent.Agent, error) {
	var agentCardSource string
	if cfg.AgentCardFile != "" {
		agentCardSource = cfg.AgentCardFile
	} else if cfg.AgentCardURL != "" {
		agentCardSource = cfg.AgentCardURL
	}

	streaming := config.BoolValue(cfg.Streaming, true)

	return remoteagent.NewA2A(remoteagent.Config{
		Name:            name,
		Description:     cfg.Description,
		URL:             cfg.URL,
		AgentCardSource: agentCardSource,
		Headers:         cfg.Headers,
		Streaming:       streaming,
	})
}

// buildLLMAgent creates an LLM agent.
func (b *Builder) buildLLMAgent(name string, cfg *config.AgentConfig) (agent.Agent, error) {
	llm, ok := b.llms[cfg.LLM]
	if !ok {
		return nil, fmt.Errorf("llm %q not found", cfg.LLM)
	}

	// Collect toolsets
	var agentToolsets []tool.Toolset
	if cfg.Tools == nil {
		// All toolsets
		for toolName, ts := range b.toolsets {
			if toolCfg, ok := b.cfg.Tools[toolName]; ok && toolCfg != nil && !toolCfg.IsEnabled() {
				continue
			}
			agentToolsets = append(agentToolsets, ts)
		}
	} else if len(cfg.Tools) > 0 {
		for _, toolName := range cfg.Tools {
			ts, err := b.resolveToolset(toolName)
			if err != nil {
				return nil, fmt.Errorf("agent %q: %w", name, err)
			}
			agentToolsets = append(agentToolsets, ts)
		}
	}

	// Get direct tools
	var tools []tool.Tool
	if directTools, ok := b.directTools[name]; ok {
		tools = append(tools, directTools...)
	}

	// Get sub-agents
	subAgents := b.subAgents[name]

	// Convert agent tools to direct tools (Pattern 2: delegation)
	for _, agTool := range b.agentTools[name] {
		tools = append(tools, agenttool.New(agTool, nil))
	}

	// RAG: Create search tool if document stores are configured
	var availableStores []string
	shouldCreateSearchTool := false

	if cfg.DocumentStores == nil {
		// Omitted = access ALL stores
		// Only create if there ARE stores
		if len(b.documentStores) > 0 {
			shouldCreateSearchTool = true
			availableStores = nil // Means all in searchtool config
		}
	} else if len(*cfg.DocumentStores) > 0 {
		// Specific stores listed
		shouldCreateSearchTool = true
		availableStores = *cfg.DocumentStores
	}
	// Else: empty list -> no search tool

	var ragTool *searchtool.SearchTool

	if shouldCreateSearchTool {
		ragTool = searchtool.New(searchtool.Config{
			Stores:          b.documentStores,
			AvailableStores: availableStores,
		})
		tools = append(tools, ragTool)
	}

	// Build output schema config
	var outputSchema map[string]any
	var outputSchemaName string
	var outputSchemaStrict *bool
	if cfg.StructuredOutput != nil {
		outputSchema = cfg.StructuredOutput.Schema
		outputSchemaName = cfg.StructuredOutput.Name
		outputSchemaStrict = cfg.StructuredOutput.Strict
	}

	// Build generate config for thinking
	var generateConfig *model.GenerateConfig
	if llmCfg, ok := b.cfg.LLMs[cfg.LLM]; ok && llmCfg != nil {
		if llmCfg.Thinking != nil && config.BoolValue(llmCfg.Thinking.Enabled, false) {
			generateConfig = &model.GenerateConfig{
				EnableThinking: true,
			}
			if llmCfg.Thinking.BudgetTokens > 0 {
				generateConfig.ThinkingBudget = llmCfg.Thinking.BudgetTokens
			}
		}
	}

	// Build reasoning config
	var reasoning *llmagent.ReasoningConfig
	if cfg.Reasoning != nil {
		reasoning = &llmagent.ReasoningConfig{
			MaxIterations:         cfg.Reasoning.MaxIterations,
			EnableExitTool:        config.BoolValue(cfg.Reasoning.EnableExitTool, false),
			EnableEscalateTool:    config.BoolValue(cfg.Reasoning.EnableEscalateTool, false),
			CompletionInstruction: cfg.Reasoning.CompletionInstruction,
		}
	}

	// Build Working Memory Strategy
	// Use agent's LLM model name for token counting
	tokenModel := llm.Name()
	wmStrategy, err := memory.NewWorkingMemoryStrategyFromConfig(
		cfg.Context,
		tokenModel,
		b.llms, // Pass all LLMs to resolve summarizer
		llm,    // Pass agent's LLM as default summarizer
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build working memory: %w", err)
	}

	// Build Context Provider (RAG)
	var ctxProvider llmagent.ContextProvider
	if config.BoolValue(cfg.IncludeContext, false) && ragTool != nil {
		ctxProvider = func(ctx agent.ReadonlyContext, query string) (string, error) {
			// Call search tool directly
			res, err := ragTool.Search(ctx, query, config.IntValue(cfg.IncludeContextLimit, 5))
			if err != nil {
				return "", err
			}

			// Format results into string
			itemBytes, _ := json.Marshal(res)
			return string(itemBytes), nil
		}
	}

	// Auto-recall configuration - zero-config defaults (memory is automatic)
	autoRecall := true
	autoRecallLimit := 10

	// Inject checkpoint callbacks if enabled
	var beforeAgentCallbacks []agent.BeforeAgentCallback
	var beforeModelCallbacks []llmagent.BeforeModelCallback
	var afterModelCallbacks []llmagent.AfterModelCallback
	var beforeToolCallbacks []llmagent.BeforeToolCallback
	var afterToolCallbacks []llmagent.AfterToolCallback
	var afterAgentCallbacks []agent.AfterAgentCallback

	if b.checkpoint != nil && b.checkpoint.IsEnabled() {
		cb := checkpoint.NewCallbacks(b.checkpoint)
		beforeModelCallbacks = append(beforeModelCallbacks, llmagent.BeforeModelCallback(cb.BeforeModel))
		afterModelCallbacks = append(afterModelCallbacks, llmagent.AfterModelCallback(cb.AfterModel))
		beforeToolCallbacks = append(beforeToolCallbacks, llmagent.BeforeToolCallback(cb.BeforeTool))
		afterToolCallbacks = append(afterToolCallbacks, llmagent.AfterToolCallback(cb.AfterTool))
		afterAgentCallbacks = append(afterAgentCallbacks, cb.OnComplete)
	}

	// Inject guardrails callbacks if configured
	if cfg.Guardrails != "" && b.cfg != nil {
		slog.Debug("Looking up guardrails config", "agent", name, "guardrails", cfg.Guardrails)
		if grCfg, ok := b.cfg.Guardrails[cfg.Guardrails]; ok && grCfg.Enabled {
			slog.Debug("Found guardrails config",
				"agent", name,
				"guardrails", cfg.Guardrails,
				"hasModeration", grCfg.Moderation != nil)
			// Convert config.GuardrailsConfig to guardrails.Config
			grConfig := convertGuardrailsConfig(grCfg)

			// Build and inject input guardrails (BeforeAgentCallback)
			inputChain := grConfig.BuildInputChain(grInput.DefaultBuilders())
			if len(inputChain.Guardrails()) > 0 {
				beforeAgentCallbacks = append(beforeAgentCallbacks,
					guardrails.ToBeforeAgentCallback(inputChain))
			}

			// Build and inject output guardrails (AfterModelCallback)
			outputChain := grConfig.BuildOutputChain(grOutput.DefaultBuilders())
			if len(outputChain.Guardrails()) > 0 {
				afterModelCallbacks = append(afterModelCallbacks,
					guardrails.ToAfterModelCallback(outputChain))
			}

			// Build and inject tool guardrails (BeforeToolCallback)
			toolChain := grConfig.BuildToolChain(grTool.DefaultBuilders())
			if len(toolChain.Guardrails()) > 0 {
				beforeToolCallbacks = append(beforeToolCallbacks,
					guardrails.ToBeforeToolCallback(toolChain))
			}

			// Build and inject moderation provider (BeforeAgentCallback)
			if grCfg.Moderation != nil && grCfg.Moderation.Enabled {
				moderationCfg := convertModerationConfig(grCfg.Moderation, llm)
				moderationProvider, err := moderation.Build(moderationCfg, llm)
				if err != nil {
					slog.Warn("Failed to build moderation provider", "error", err)
				} else if moderationProvider != nil {
					beforeAgentCallbacks = append(beforeAgentCallbacks,
						createModerationCallback(moderationProvider, grCfg.Moderation.Action))
					slog.Debug("Moderation enabled for agent",
						"agent", name,
						"strategy", grCfg.Moderation.Strategy)
				}
			}

			slog.Debug("Guardrails enabled for agent", "agent", name, "guardrails", cfg.Guardrails)
		}
	}

	// Create agent
	return llmagent.New(llmagent.Config{
		Name:                   name,
		Description:            cfg.Description,
		Model:                  llm,
		Instruction:            cfg.GetSystemPrompt(),
		Toolsets:               agentToolsets,
		Tools:                  tools,
		SubAgents:              subAgents,
		EnableStreaming:        config.BoolValue(cfg.Streaming, true),
		Reasoning:              reasoning,
		GenerateConfig:         generateConfig,
		WorkingMemory:          wmStrategy,
		AutoRecall:             autoRecall,
		AutoRecallLimit:        autoRecallLimit,
		ContextProvider:        ctxProvider,
		OutputSchema:           outputSchema,
		OutputSchemaName:       outputSchemaName,
		OutputSchemaStrict:     outputSchemaStrict,
		BeforeAgentCallbacks:   beforeAgentCallbacks,
		BeforeModelCallbacks:   beforeModelCallbacks,
		AfterModelCallbacks:    afterModelCallbacks,
		BeforeToolCallbacks:    beforeToolCallbacks,
		AfterToolCallbacks:     afterToolCallbacks,
		AfterAgentCallbacks:    afterAgentCallbacks,
		DisableSafetyProtocols: cfg.DisableSafetyProtocols,
	})
}

// resolveToolset finds a toolset by name or via MCP.
func (b *Builder) resolveToolset(toolName string) (tool.Toolset, error) {
	if ts, ok := b.toolsets[toolName]; ok {
		return ts, nil
	}

	// Try implicit MCP tools
	var toolsetNames []string
	for name, ts := range b.toolsets {
		if _, ok := ts.(*mcptoolset.Toolset); ok {
			toolsetNames = append(toolsetNames, name)
		}
	}
	sort.Strings(toolsetNames)

	for _, name := range toolsetNames {
		ts := b.toolsets[name]
		if mcpTS, ok := ts.(*mcptoolset.Toolset); ok {
			if b.cfg == nil {
				continue
			}
			toolCfg := b.cfg.Tools[mcpTS.Name()]
			if toolCfg == nil || toolCfg.Type != config.ToolTypeMCP {
				continue
			}

			allowed := false
			if len(toolCfg.Filter) > 0 {
				for _, allowedTool := range toolCfg.Filter {
					if allowedTool == toolName {
						allowed = true
						break
					}
				}
			} else {
				allowed = true
			}

			if allowed {
				return mcpTS.WithFilter([]string{toolName}), nil
			}
		}
	}

	return nil, fmt.Errorf("tool %q not found", toolName)
}

// createScheduler creates the trigger scheduler.
func (b *Builder) createScheduler() *trigger.Scheduler {
	if b.cfg == nil {
		return nil
	}

	// Check if any agent has triggers
	hasTriggers := false
	for _, agCfg := range b.cfg.Agents {
		if agCfg != nil && agCfg.Trigger != nil {
			hasTriggers = true
			break
		}
	}
	if !hasTriggers {
		return nil
	}

	// Create invoker that uses builder's agents and sessions
	agents := b.agents
	sessions := b.sessions
	appName := b.cfg.Name

	invoker := func(ctx context.Context, agentName, input string) (string, error) {
		ag, ok := agents[agentName]
		if !ok {
			return "", fmt.Errorf("agent %q not found", agentName)
		}

		slog.Info("Trigger invoking agent", "agent", agentName, "input", input)

		// Create or get session
		userID := "trigger"
		sessionID := fmt.Sprintf("trigger-%s", agentName)

		var sess session.Session
		getResp, err := sessions.Get(ctx, &session.GetRequest{
			AppName:   appName,
			UserID:    userID,
			SessionID: sessionID,
		})
		if err == nil && getResp != nil {
			sess = getResp.Session
		} else {
			createResp, err := sessions.Create(ctx, &session.CreateRequest{
				AppName:   appName,
				UserID:    userID,
				SessionID: sessionID,
			})
			if err != nil {
				return sessionID, fmt.Errorf("failed to create session: %w", err)
			}
			sess = createResp.Session
		}

		// Create invocation context
		userContent := agent.NewTextContent(input, "user")
		invCtx := agent.NewInvocationContext(ctx, agent.InvocationContextParams{
			Agent:       ag,
			Session:     sess,
			Branch:      "main",
			UserContent: userContent,
			RunConfig:   &agent.RunConfig{},
		})

		// Append user message
		userEvent := agent.NewEvent(invCtx.InvocationID())
		userEvent.Author = "user"
		userEvent.Artifact = &a2a.Artifact{Parts: userContent.Parts}
		if err := sessions.AppendEvent(ctx, sess, userEvent); err != nil {
			return sessionID, fmt.Errorf("failed to persist user message: %w", err)
		}

		// Execute agent
		for event, err := range ag.Run(invCtx) {
			if err != nil {
				return sessionID, fmt.Errorf("invocation error: %w", err)
			}
			if event != nil {
				if err := sessions.AppendEvent(ctx, sess, event); err != nil {
					slog.Warn("Failed to persist trigger event", "error", err)
				}
				if event.IsFinalResponse() {
					slog.Info("Trigger invocation completed", "agent", agentName)
				}
			}
		}
		return sessionID, nil
	}

	// Create adapter for direct invocation (no durability in standalone builder)
	// For production usage via Serve(), the serve.go bootstrap provides a durable queue.
	enqueuer := func(ctx context.Context, appName, agentName, input, contextID string) error {
		// Use a reasonable timeout for standalone scheduled tasks
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		_, err := invoker(ctx, agentName, input)
		return err
	}

	scheduler := trigger.NewScheduler(b.appID, enqueuer, b.observability.Metrics())

	// Register triggers
	for name, agCfg := range b.cfg.Agents {
		if agCfg == nil || agCfg.Trigger == nil {
			continue
		}
		ag, ok := b.agents[name]
		if !ok {
			continue
		}
		if err := scheduler.RegisterAgent(name, ag, agCfg.Trigger); err != nil {
			slog.Error("Failed to register agent trigger", "agent", name, "error", err)
		}
	}

	return scheduler
}

// buildCheckpointManagerZeroConfig creates checkpoint manager with zero-config defaults.
func (b *Builder) buildCheckpointManagerZeroConfig() (*checkpoint.Manager, error) {
	if b.dbPool == nil || b.sessions == nil {
		return nil, nil // No database or sessions, no checkpoint
	}

	// Use default checkpoint config with hybrid strategy
	enabled := true
	cpCfg := &checkpoint.Config{
		Enabled:  &enabled,
		Strategy: checkpoint.StrategyHybrid, // Smart automatic checkpointing
	}
	cpCfg.SetDefaults()

	return checkpoint.NewManager(cpCfg, b.sessions), nil
}

// convertGuardrailsConfig converts config.GuardrailsConfig to guardrails.Config.
// This bridges the config package types to the guardrails package types.
func convertGuardrailsConfig(cfg *config.GuardrailsConfig) *guardrails.Config {
	gc := &guardrails.Config{}

	if cfg.Input != nil {
		gc.Input.ChainMode = guardrails.ChainMode(cfg.Input.ChainMode)

		if cfg.Input.Length != nil && cfg.Input.Length.Enabled {
			gc.Input.Length = &guardrails.LengthConfig{
				Enabled:   true,
				MinLength: cfg.Input.Length.MinLength,
				MaxLength: cfg.Input.Length.MaxLength,
				Action:    guardrails.Action(cfg.Input.Length.Action),
				Severity:  guardrails.Severity(cfg.Input.Length.Severity),
			}
		}

		if cfg.Input.Injection != nil && cfg.Input.Injection.Enabled {
			gc.Input.Injection = &guardrails.InjectionConfig{
				Enabled:       true,
				Patterns:      cfg.Input.Injection.Patterns,
				CaseSensitive: cfg.Input.Injection.CaseSensitive,
				Action:        guardrails.Action(cfg.Input.Injection.Action),
				Severity:      guardrails.Severity(cfg.Input.Injection.Severity),
			}
		}

		if cfg.Input.Sanitizer != nil && cfg.Input.Sanitizer.Enabled {
			gc.Input.Sanitizer = &guardrails.SanitizerConfig{
				Enabled:          true,
				TrimWhitespace:   cfg.Input.Sanitizer.TrimWhitespace,
				NormalizeUnicode: cfg.Input.Sanitizer.NormalizeUnicode,
				MaxLength:        cfg.Input.Sanitizer.MaxLength,
				StripHTML:        cfg.Input.Sanitizer.StripHTML,
			}
		}
	}

	if cfg.Output != nil {
		gc.Output.ChainMode = guardrails.ChainMode(cfg.Output.ChainMode)

		if cfg.Output.PII != nil && cfg.Output.PII.Enabled {
			gc.Output.PII = &guardrails.PIIConfig{
				Enabled:          true,
				DetectEmail:      cfg.Output.PII.DetectEmail,
				DetectPhone:      cfg.Output.PII.DetectPhone,
				DetectSSN:        cfg.Output.PII.DetectSSN,
				DetectCreditCard: cfg.Output.PII.DetectCreditCard,
				RedactMode:       guardrails.RedactMode(cfg.Output.PII.RedactMode),
				Action:           guardrails.Action(cfg.Output.PII.Action),
				Severity:         guardrails.Severity(cfg.Output.PII.Severity),
			}
		}

		if cfg.Output.Content != nil && cfg.Output.Content.Enabled {
			gc.Output.Content = &guardrails.ContentConfig{
				Enabled:         true,
				BlockedKeywords: cfg.Output.Content.BlockedKeywords,
				BlockedPatterns: cfg.Output.Content.BlockedPatterns,
				Action:          guardrails.Action(cfg.Output.Content.Action),
				Severity:        guardrails.Severity(cfg.Output.Content.Severity),
			}
		}
	}

	if cfg.Tool != nil {
		gc.Tool.ChainMode = guardrails.ChainMode(cfg.Tool.ChainMode)

		if cfg.Tool.Authorization != nil && cfg.Tool.Authorization.Enabled {
			gc.Tool.Authorization = &guardrails.AuthorizationConfig{
				Enabled:      true,
				AllowedTools: cfg.Tool.Authorization.AllowedTools,
				BlockedTools: cfg.Tool.Authorization.BlockedTools,
				Action:       guardrails.Action(cfg.Tool.Authorization.Action),
				Severity:     guardrails.Severity(cfg.Tool.Authorization.Severity),
			}
		}
	}

	return gc
}

// convertModerationConfig converts config.ModerationGuardrailConfig to moderation.Config.
func convertModerationConfig(cfg *config.ModerationGuardrailConfig, llm model.LLM) *moderation.Config {
	if cfg == nil {
		return nil
	}

	mc := &moderation.Config{
		Strategy: moderation.Strategy(cfg.Strategy),
	}

	if cfg.OpenAI != nil {
		mc.OpenAI = &moderation.OpenAIConfig{
			Model:     cfg.OpenAI.Model,
			Threshold: cfg.OpenAI.Threshold,
		}
	}

	if cfg.Lakera != nil {
		mc.Lakera = &moderation.LakeraConfig{
			ProjectID: cfg.Lakera.ProjectID,
			Breakdown: cfg.Lakera.Breakdown,
			Endpoint:  cfg.Lakera.Endpoint,
		}
	}

	if cfg.Prompt != nil {
		mc.Prompt = &moderation.PromptConfig{
			LLM:       llm, // Use the agent's LLM
			Template:  cfg.Prompt.Template,
			SafeField: cfg.Prompt.SafeField,
		}
	}

	return mc
}

// createModerationCallback creates a BeforeAgentCallback that runs moderation.
func createModerationCallback(provider moderation.Provider, action string) agent.BeforeAgentCallback {
	return func(ctx agent.CallbackContext) (*a2a.Message, error) {
		// Get the user content from the context
		userContent := ctx.UserContent()
		if userContent == nil {
			return nil, nil
		}

		// Extract text from user content
		var text string
		for _, part := range userContent.Parts {
			if tp, ok := part.(a2a.TextPart); ok {
				text += tp.Text
			}
		}

		if text == "" {
			return nil, nil
		}

		// Run moderation - ctx embeds context.Context directly
		result, err := provider.Moderate(ctx, text)
		if err != nil {
			slog.Warn("Moderation check failed", "provider", provider.Name(), "error", err)
			// Don't block on moderation errors - just log and continue
			return nil, nil
		}

		if result != nil && !result.Safe {
			slog.Info("Content blocked by moderation",
				"provider", provider.Name(),
				"category", result.Category,
				"confidence", result.Confidence,
				"reason", result.Reason)

			if action == "" || action == "block" {
				// Build intervention metadata for X-ray traceability
				extras := map[string]any{
					"provider": provider.Name(),
				}
				if result.Category != "" {
					extras["category"] = result.Category
				}
				if result.Confidence > 0 {
					extras["confidence"] = result.Confidence
				}
				ctx.SetMetadata(agent.BuildInterventionMetadata(
					agent.MetadataTypeModeration,
					agent.InterventionActionBlock,
					agent.InterventionSourceModeration,
					result.Reason,
					extras,
				))

				// Block the content - return a message to use instead
				return a2a.NewMessage(a2a.MessageRoleAgent,
					a2a.TextPart{Text: "I cannot process this request. Your message was flagged by our content moderation system."},
				), nil
			}
			// Action is "warn" - log but don't block
		}

		return nil, nil
	}
}
