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

func NewAgentServicesWithRegistry(agentID string, agentConfig *config.AgentConfig, componentManager *component.ComponentManager, registry *AgentRegistry) (reasoning.AgentServices, error) {

	var registryService reasoning.AgentRegistryService
	if registry != nil {
		registryService = NewRegistryService(registry)
	} else {
		registryService = NewNoOpRegistryService()
	}

	return newAgentServicesInternal(agentID, agentConfig, componentManager, registryService)
}

func newAgentServicesInternal(agentID string, agentConfig *config.AgentConfig, componentManager *component.ComponentManager, registryService reasoning.AgentRegistryService) (reasoning.AgentServices, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}
	if agentConfig == nil {
		return nil, fmt.Errorf("agent config cannot be nil")
	}
	if componentManager == nil {
		return nil, fmt.Errorf("component manager cannot be nil")
	}

	llm, err := componentManager.GetLLM(agentConfig.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM '%s': %w", agentConfig.LLM, err)
	}

	toolRegistry := componentManager.GetToolRegistry()

	strategy, err := reasoning.CreateStrategy(agentConfig.Reasoning.Engine, agentConfig.Reasoning)
	if err != nil {
		return nil, fmt.Errorf("failed to create reasoning strategy: %w", err)
	}

	requiredTools := strategy.GetRequiredTools()
	for _, reqTool := range requiredTools {

		if _, err := toolRegistry.GetTool(reqTool.Name); err == nil {

			continue
		}

		if reqTool.AutoCreate {

			var agentReg interface{}
			if registryService != nil {
				if regSvc, ok := registryService.(*RegistryService); ok {
					agentReg = regSvc.GetRegistry()
				}
			}

			if err := registerRequiredToolWithAgentRegistry(toolRegistry, reqTool, agentReg); err != nil {
				return nil, fmt.Errorf("failed to register required tool '%s': %w", reqTool.Name, err)
			}
			fmt.Printf("Info: Auto-registered strategy-required tool '%s' (%s)\n", reqTool.Name, reqTool.Description)
		}
	}

	var contextService reasoning.ContextService
	if len(agentConfig.DocumentStores) > 0 {

		if agentConfig.Database == "" {
			return nil, fmt.Errorf("database is required when document stores are configured")
		}
		if agentConfig.Embedder == "" {
			return nil, fmt.Errorf("embedder is required when document stores are configured")
		}

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

		globalConfig := componentManager.GetGlobalConfig()
		var documentStoreConfigs []*config.DocumentStoreConfig
		for _, storeName := range agentConfig.DocumentStores {
			storeConfig, exists := globalConfig.DocumentStores[storeName]
			if !exists {
				return nil, fmt.Errorf("document store '%s' not found in global configuration", storeName)
			}
			documentStoreConfigs = append(documentStoreConfigs, storeConfig)
		}

		if err := hectorcontext.InitializeDocumentStoresFromConfig(documentStoreConfigs, searchEngine); err != nil {
			return nil, fmt.Errorf("failed to initialize document stores: %w", err)
		}

		contextService = NewContextService(searchEngine)
	} else {

		contextService = NewNoOpContextService()
	}

	llmService := NewLLMService(llm)
	toolService := NewToolService(toolRegistry, agentConfig.Tools)

	var summarizer *SummarizationService
	if agentConfig.Memory.Strategy == "summary_buffer" || agentConfig.Memory.Strategy == "" {
		var err error
		summarizer, err = NewSummarizationService(llm, &SummarizationConfig{
			Model: llm.GetModelName(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create summarization service: %w", err)
		}
	}

	workingStrategy, err := memory.NewWorkingMemoryStrategy(memory.WorkingMemoryConfig{
		Strategy:   agentConfig.Memory.Strategy,
		WindowSize: agentConfig.Memory.WindowSize,
		Budget:     agentConfig.Memory.Budget,
		Threshold:  agentConfig.Memory.Threshold,
		Target:     agentConfig.Memory.Target,
		Model:      llm.GetModelName(),
		Summarizer: summarizer,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create working memory strategy: %w", err)
	}

	var longTermStrategy memory.LongTermMemoryStrategy
	if agentConfig.Memory.LongTerm.IsEnabled() {

		if agentConfig.Database == "" {
			return nil, fmt.Errorf("long-term memory requires database to be configured")
		}
		if agentConfig.Embedder == "" {
			return nil, fmt.Errorf("long-term memory requires embedder to be configured")
		}

		db, err := componentManager.GetDatabase(agentConfig.Database)
		if err != nil {
			return nil, fmt.Errorf("failed to get database '%s': %w", agentConfig.Database, err)
		}

		embedder, err := componentManager.GetEmbedder(agentConfig.Embedder)
		if err != nil {
			return nil, fmt.Errorf("failed to get embedder '%s': %w", agentConfig.Embedder, err)
		}

		longTermStrategy, err = memory.NewVectorMemoryStrategy(
			db,
			embedder,
			agentConfig.Memory.LongTerm.Collection,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create long-term memory: %w", err)
		}

		fmt.Printf("✅ Long-term memory enabled (collection: %s, batch_size: %d, auto_recall: %t, recall_limit: %d)\n",
			agentConfig.Memory.LongTerm.Collection,
			agentConfig.Memory.LongTerm.BatchSize,
			agentConfig.Memory.LongTerm.AutoRecall,
			agentConfig.Memory.LongTerm.RecallLimit)
	}

	longTermConfig := memory.LongTermConfig{
		Enabled:      agentConfig.Memory.LongTerm.IsEnabled(),
		StorageScope: memory.StorageScope(agentConfig.Memory.LongTerm.StorageScope),
		BatchSize:    agentConfig.Memory.LongTerm.BatchSize,
		AutoRecall:   agentConfig.Memory.LongTerm.AutoRecall,
		RecallLimit:  agentConfig.Memory.LongTerm.RecallLimit,
		Collection:   agentConfig.Memory.LongTerm.Collection,
	}

	sessionService, err := componentManager.GetSessionService(agentConfig.SessionStore, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session service: %w", err)
	}

	historyService := memory.NewMemoryService(
		agentID,
		sessionService,
		workingStrategy,
		longTermStrategy,
		longTermConfig,
	)

	promptService := NewPromptService(agentConfig.Prompt, contextService, historyService)

	var taskService reasoning.TaskService
	if agentConfig.Task != nil && agentConfig.Task.IsEnabled() {
		switch agentConfig.Task.Backend {
		case "memory":
			taskService = NewInMemoryTaskService()
		case "sql":
			if agentConfig.Task.SQL == nil {
				return nil, fmt.Errorf("SQL configuration is required for SQL backend")
			}
			sqlService, err := NewSQLTaskServiceFromConfig(agentConfig.Task.SQL)
			if err != nil {
				return nil, fmt.Errorf("failed to create SQL task service: %w", err)
			}
			taskService = sqlService
		default:
			return nil, fmt.Errorf("unsupported task backend: %s", agentConfig.Task.Backend)
		}
	}

	agentServices := reasoning.NewAgentServices(
		agentConfig.Reasoning,
		llmService,
		toolService,
		contextService,
		promptService,
		sessionService,
		historyService,
		registryService,
		taskService,
	)

	return agentServices, nil
}

func registerRequiredToolWithAgentRegistry(registry *tools.ToolRegistry, reqTool reasoning.RequiredTool, agentRegistry interface{}) error {

	localSource := tools.NewLocalToolSource("strategy-required")

	var tool tools.Tool
	switch reqTool.Type {
	case "todo":
		tool = tools.NewTodoTool()
	case "agent_call":
		var registry tools.AgentRegistry
		if agentRegistry != nil {
			if ar, ok := agentRegistry.(tools.AgentRegistry); ok {
				registry = ar
			}
		}

		if registry == nil {
			return fmt.Errorf("agent_call tool requires agent registry but none was provided")
		}
		tool = tools.NewAgentCallTool(registry)
	case "command":
		toolConfig := &config.ToolConfig{
			Type:             "command",
			AllowedCommands:  []string{"ls", "cat", "pwd", "echo"},
			WorkingDirectory: "./",
			MaxExecutionTime: "30s",
			EnableSandboxing: true,
		}
		cmdTool, err := tools.NewCommandToolWithConfig(reqTool.Name, toolConfig)
		if err != nil {
			return err
		}
		tool = cmdTool
	default:
		return fmt.Errorf("unsupported required tool type: %s", reqTool.Type)
	}

	if err := localSource.RegisterTool(tool); err != nil {
		return err
	}

	return registry.RegisterSource(localSource)
}
