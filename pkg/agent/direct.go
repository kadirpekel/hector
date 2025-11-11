package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/tools"
)

// AgentBuilderOptions contains all options for building an agent directly
type AgentBuilderOptions struct {
	ID                 string
	Name               string
	Description        string
	LLMProvider        llms.LLMProvider
	ReasoningStrategy  reasoning.ReasoningStrategy
	ReasoningConfig    *config.ReasoningConfig
	WorkingMemory      memory.WorkingMemoryStrategy
	LongTermMemory     memory.LongTermMemoryStrategy
	LongTermConfig     memory.LongTermConfig
	Tools              []tools.Tool
	SystemPrompt       string
	PromptSlots        *reasoning.PromptSlots
	IncludeContext     *bool
	Registry           *AgentRegistry
	BaseURL            string
	PreferredTransport string
	SessionService     reasoning.SessionService
	ContextService     reasoning.ContextService
	TaskService        reasoning.TaskService
}

// NewAgentDirect creates an agent directly from components without config
func NewAgentDirect(opts AgentBuilderOptions) (*Agent, error) {
	if opts.ID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}
	if opts.LLMProvider == nil {
		return nil, fmt.Errorf("LLM provider is required")
	}
	if opts.ReasoningStrategy == nil {
		return nil, fmt.Errorf("reasoning strategy is required")
	}
	if opts.WorkingMemory == nil {
		return nil, fmt.Errorf("working memory strategy is required")
	}

	// Build services from components
	services, err := buildAgentServicesDirect(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build agent services: %w", err)
	}

	// Determine transport
	transport := opts.PreferredTransport
	if transport == "" {
		transport = "json-rpc"
	}

	// Default timeout
	awaitTimeout := 10 * time.Minute

	// Build agent
	agent := &Agent{
		id:                 opts.ID,
		name:               opts.Name,
		description:        opts.Description,
		services:           services,
		baseURL:            opts.BaseURL,
		preferredTransport: transport,
		taskAwaiter:        NewTaskAwaiter(awaitTimeout),
		activeExecutions:   make(map[string]context.CancelFunc),
		componentManager:   nil, // Not needed for direct construction
		config:             nil, // Not needed for direct construction
	}

	return agent, nil
}

// buildAgentServicesDirect builds agent services from components
func buildAgentServicesDirect(opts AgentBuilderOptions) (reasoning.AgentServices, error) {
	// LLM Service
	llmService := NewLLMService(opts.LLMProvider)

	// Tool Service
	toolRegistry := tools.NewToolRegistry()
	for _, tool := range opts.Tools {
		source := tools.NewLocalToolSource("programmatic")
		if err := source.RegisterTool(tool); err != nil {
			return nil, fmt.Errorf("failed to register tool %s: %w", tool.GetName(), err)
		}
		if err := toolRegistry.RegisterSource(source); err != nil {
			return nil, fmt.Errorf("failed to register tool source: %w", err)
		}
	}

	// Register required tools from reasoning strategy
	requiredTools := opts.ReasoningStrategy.GetRequiredTools()
	for _, reqTool := range requiredTools {
		if _, err := toolRegistry.GetTool(reqTool.Name); err == nil {
			continue
		}

		if reqTool.AutoCreate {
			var agentReg interface{}
			if opts.Registry != nil {
				agentReg = opts.Registry
			}

			if err := registerRequiredToolWithAgentRegistry(toolRegistry, reqTool, agentReg); err != nil {
				return nil, fmt.Errorf("failed to register required tool '%s': %w", reqTool.Name, err)
			}
		}
	}

	toolService := NewToolService(toolRegistry, nil) // nil means all tools allowed

	// Context Service (optional)
	contextService := opts.ContextService
	if contextService == nil {
		contextService = NewNoOpContextService()
	}

	// Session Service (optional)
	sessionService := opts.SessionService
	if sessionService == nil {
		sessionService = memory.NewInMemorySessionService()
	}

	// History Service (Memory Service)
	longTermConfig := opts.LongTermConfig
	longTermConfig.SetDefaults()

	historyService := memory.NewMemoryService(
		opts.ID,
		sessionService,
		opts.WorkingMemory,
		opts.LongTermMemory,
		longTermConfig,
	)

	// Prompt Service
	promptConfig := config.PromptConfig{}
	if opts.SystemPrompt != "" {
		promptConfig.SystemPrompt = opts.SystemPrompt
	}
	if opts.PromptSlots != nil {
		promptConfig.PromptSlots = &config.PromptSlotsConfig{
			SystemRole:   opts.PromptSlots.SystemRole,
			Instructions: opts.PromptSlots.Instructions,
			UserGuidance: opts.PromptSlots.UserGuidance,
		}
	}
	if opts.IncludeContext != nil {
		promptConfig.IncludeContext = opts.IncludeContext
	}

	promptService := NewPromptService(promptConfig, contextService, historyService)

	// Registry Service
	var registryService reasoning.AgentRegistryService
	if opts.Registry != nil {
		registryService = NewRegistryService(opts.Registry)
	} else {
		registryService = NewNoOpRegistryService()
	}

	// Task Service (optional)
	taskService := opts.TaskService

	// Build reasoning config from strategy or use provided config
	reasoningConfig := config.ReasoningConfig{
		Engine: opts.ReasoningStrategy.GetName(),
	}
	if opts.ReasoningConfig != nil {
		reasoningConfig = *opts.ReasoningConfig
		// Ensure engine matches strategy
		reasoningConfig.Engine = opts.ReasoningStrategy.GetName()
	}

	// Create agent services
	agentServices := reasoning.NewAgentServices(
		reasoningConfig,
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

// registerRequiredToolWithAgentRegistry registers a required tool with the tool registry
func registerRequiredToolWithAgentRegistry(registry *tools.ToolRegistry, reqTool reasoning.RequiredTool, agentRegistry interface{}) error {
	localSource := tools.NewLocalToolSource("strategy-required")

	var tool tools.Tool
	switch reqTool.Type {
	case "todo":
		tool = tools.NewTodoTool()
	case "agent_call":
		var reg tools.AgentRegistry
		if agentRegistry != nil {
			if ar, ok := agentRegistry.(tools.AgentRegistry); ok {
				reg = ar
			}
		}

		if reg == nil {
			return fmt.Errorf("agent_call tool requires agent registry but none was provided")
		}
		tool = tools.NewAgentCallTool(reg)
	case "command":
		toolConfig := &config.ToolConfig{
			Type:             "command",
			AllowedCommands:  []string{"ls", "cat", "pwd", "echo"},
			WorkingDirectory: "./",
			MaxExecutionTime: "30s",
			EnableSandboxing: config.BoolPtr(true),
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
