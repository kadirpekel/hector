package hector

import (
	"fmt"

	"github.com/kadirpekel/hector/pkg/agent"
	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/llms"
	"github.com/kadirpekel/hector/pkg/memory"
	"github.com/kadirpekel/hector/pkg/reasoning"
	"github.com/kadirpekel/hector/pkg/tools"
)

// AgentBuilder provides a fluent API for building agents programmatically
type AgentBuilder struct {
	id                 string
	agentType          string // "native" or "a2a"
	name               string
	description        string
	visibility         string // "public", "internal", "private"
	url                string // For external A2A agents
	credentials        *config.AgentCredentials
	llmProvider        llms.LLMProvider
	reasoningStrategy  reasoning.ReasoningStrategy
	reasoningConfig    *ReasoningConfig
	workingMemory      memory.WorkingMemoryStrategy
	longTermMemory     memory.LongTermMemoryStrategy
	longTermConfig     memory.LongTermConfig
	toolList           []tools.Tool
	systemPrompt       string
	promptSlots        *reasoning.PromptSlots
	includeContext     *bool
	subAgents          []string
	security           *config.SecurityConfig
	structuredOutput   *config.StructuredOutputConfig
	docsFolder         string
	enableTools        *bool
	a2aCard            *config.A2ACardConfig
	database           string // Internal reference name (for config compatibility)
	embedder           string // Internal reference name (for config compatibility)
	registry           *agent.AgentRegistry
	baseURL            string
	preferredTransport string
	sessionService     reasoning.SessionService
	contextService     reasoning.ContextService
	taskService        reasoning.TaskService
	taskConfig         *config.TaskConfig // Store task config for checkpoint/HITL
}

// NewAgent creates a new agent builder
func NewAgent(id string) *AgentBuilder {
	if id == "" {
		panic("agent ID cannot be empty")
	}
	return &AgentBuilder{
		id:                 id,
		agentType:          "native", // Default to native agent
		visibility:         "public", // Default visibility
		toolList:           make([]tools.Tool, 0),
		preferredTransport: "json-rpc",
	}
}

// WithName sets the agent name
func (b *AgentBuilder) WithName(name string) *AgentBuilder {
	b.name = name
	return b
}

// WithDescription sets the agent description
func (b *AgentBuilder) WithDescription(desc string) *AgentBuilder {
	b.description = desc
	return b
}

// WithLLMProvider sets the LLM provider
func (b *AgentBuilder) WithLLMProvider(provider llms.LLMProvider) *AgentBuilder {
	if provider == nil {
		panic("LLM provider cannot be nil")
	}
	b.llmProvider = provider
	return b
}

// WithSystemPrompt sets the system prompt
func (b *AgentBuilder) WithSystemPrompt(prompt string) *AgentBuilder {
	b.systemPrompt = prompt
	return b
}

// WithReasoningStrategy sets the reasoning strategy
func (b *AgentBuilder) WithReasoningStrategy(strategy reasoning.ReasoningStrategy) *AgentBuilder {
	if strategy == nil {
		panic("reasoning strategy cannot be nil")
	}
	b.reasoningStrategy = strategy
	return b
}

// WithReasoning sets the reasoning strategy and config from a ReasoningBuilder
func (b *AgentBuilder) WithReasoning(builder *ReasoningBuilder) *AgentBuilder {
	if builder == nil {
		panic("reasoning builder cannot be nil")
	}
	strategy, err := builder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build reasoning strategy: %v", err))
	}
	b.reasoningStrategy = strategy
	config := builder.GetConfig()
	b.reasoningConfig = &config
	return b
}

// WithWorkingMemory sets the working memory strategy
func (b *AgentBuilder) WithWorkingMemory(strategy memory.WorkingMemoryStrategy) *AgentBuilder {
	if strategy == nil {
		panic("working memory strategy cannot be nil")
	}
	b.workingMemory = strategy
	return b
}

// WithLongTermMemory sets the long-term memory strategy
func (b *AgentBuilder) WithLongTermMemory(strategy memory.LongTermMemoryStrategy, config memory.LongTermConfig) *AgentBuilder {
	b.longTermMemory = strategy
	b.longTermConfig = config
	return b
}

// WithTool adds a single tool
func (b *AgentBuilder) WithTool(tool tools.Tool) *AgentBuilder {
	if tool == nil {
		panic("tool cannot be nil")
	}
	b.toolList = append(b.toolList, tool)
	return b
}

// WithTools adds multiple tools
func (b *AgentBuilder) WithTools(toolList ...tools.Tool) *AgentBuilder {
	for _, tool := range toolList {
		if tool == nil {
			panic("tool cannot be nil")
		}
		b.toolList = append(b.toolList, tool)
	}
	return b
}

// WithPromptSlots sets prompt slots
func (b *AgentBuilder) WithPromptSlots(slots *reasoning.PromptSlots) *AgentBuilder {
	b.promptSlots = slots
	return b
}

// WithSessionService sets the session service
func (b *AgentBuilder) WithSessionService(service reasoning.SessionService) *AgentBuilder {
	b.sessionService = service
	return b
}

// WithSession sets the session service from a SessionServiceBuilder
func (b *AgentBuilder) WithSession(builder *SessionServiceBuilder) *AgentBuilder {
	if builder == nil {
		panic("session builder cannot be nil")
	}
	service, err := builder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build session service: %v", err))
	}
	b.sessionService = service
	return b
}

// WithContextService sets the context service (for RAG)
func (b *AgentBuilder) WithContextService(service reasoning.ContextService) *AgentBuilder {
	b.contextService = service
	return b
}

// WithContext sets the context service from a ContextServiceBuilder
func (b *AgentBuilder) WithContext(builder *ContextServiceBuilder) *AgentBuilder {
	if builder == nil {
		panic("context builder cannot be nil")
	}
	service, err := builder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build context service: %v", err))
	}
	b.contextService = service
	// Set IncludeContext from builder
	if includeContext := builder.GetIncludeContext(); includeContext != nil {
		b.includeContext = includeContext
	}
	return b
}

// IncludeContext enables or disables context inclusion in prompts
func (b *AgentBuilder) IncludeContext(include bool) *AgentBuilder {
	b.includeContext = &include
	return b
}

// WithTaskService sets the task service
func (b *AgentBuilder) WithTaskService(service reasoning.TaskService) *AgentBuilder {
	b.taskService = service
	return b
}

// WithTask sets the task service from a TaskServiceBuilder
func (b *AgentBuilder) WithTask(builder *TaskServiceBuilder) *AgentBuilder {
	if builder == nil {
		panic("task builder cannot be nil")
	}
	service, err := builder.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to build task service: %v", err))
	}
	b.taskService = service
	// Store task config for checkpoint/HITL support
	b.taskConfig = builder.GetTaskConfig()
	return b
}

// WithRegistry sets the agent registry (for multi-agent scenarios)
func (b *AgentBuilder) WithRegistry(registry *agent.AgentRegistry) *AgentBuilder {
	b.registry = registry
	return b
}

// WithBaseURL sets the base URL for agent card generation
func (b *AgentBuilder) WithBaseURL(url string) *AgentBuilder {
	b.baseURL = url
	return b
}

// WithPreferredTransport sets the preferred A2A transport
func (b *AgentBuilder) WithPreferredTransport(transport string) *AgentBuilder {
	b.preferredTransport = transport
	return b
}

// WithType sets the agent type ("native" or "a2a")
func (b *AgentBuilder) WithType(agentType string) *AgentBuilder {
	if agentType != "native" && agentType != "a2a" {
		panic(fmt.Sprintf("invalid agent type: %s (must be 'native' or 'a2a')", agentType))
	}
	b.agentType = agentType
	return b
}

// WithVisibility sets the agent visibility ("public", "internal", or "private")
func (b *AgentBuilder) WithVisibility(visibility string) *AgentBuilder {
	if visibility != "public" && visibility != "internal" && visibility != "private" {
		panic(fmt.Sprintf("invalid visibility: %s (must be 'public', 'internal', or 'private')", visibility))
	}
	b.visibility = visibility
	return b
}

// WithURL sets the URL for external A2A agents
func (b *AgentBuilder) WithURL(url string) *AgentBuilder {
	b.url = url
	return b
}

// WithCredentials sets credentials for external A2A agents
func (b *AgentBuilder) WithCredentials(creds *config.AgentCredentials) *AgentBuilder {
	b.credentials = creds
	return b
}

// Credentials creates a credentials builder
func (b *AgentBuilder) Credentials() *AgentCredentialsBuilder {
	if b.credentials == nil {
		b.credentials = &config.AgentCredentials{}
	}
	return NewAgentCredentialsWithConfig(b.credentials)
}

// WithSubAgents sets sub-agent IDs
func (b *AgentBuilder) WithSubAgents(subAgents []string) *AgentBuilder {
	b.subAgents = subAgents
	return b
}

// AddSubAgent adds a sub-agent ID
func (b *AgentBuilder) AddSubAgent(subAgentID string) *AgentBuilder {
	b.subAgents = append(b.subAgents, subAgentID)
	return b
}

// WithSecurity sets security configuration
func (b *AgentBuilder) WithSecurity(cfg *config.SecurityConfig) *AgentBuilder {
	b.security = cfg
	return b
}

// Security creates a security config builder
func (b *AgentBuilder) Security() *SecurityBuilder {
	if b.security == nil {
		b.security = &config.SecurityConfig{}
	}
	return NewSecurityBuilder(b.security)
}

// WithStructuredOutput sets structured output configuration
func (b *AgentBuilder) WithStructuredOutput(cfg *config.StructuredOutputConfig) *AgentBuilder {
	b.structuredOutput = cfg
	return b
}

// StructuredOutput creates a structured output builder
func (b *AgentBuilder) StructuredOutput() *StructuredOutputBuilder {
	if b.structuredOutput == nil {
		b.structuredOutput = &config.StructuredOutputConfig{}
	}
	return NewStructuredOutputWithConfig(b.structuredOutput)
}

// WithDocsFolder sets the docs folder (shortcut for document stores)
func (b *AgentBuilder) WithDocsFolder(folder string) *AgentBuilder {
	b.docsFolder = folder
	return b
}

// EnableTools enables or disables all tools (shortcut)
func (b *AgentBuilder) EnableTools(enable bool) *AgentBuilder {
	b.enableTools = &enable
	return b
}

// WithA2ACard sets A2A card configuration
func (b *AgentBuilder) WithA2ACard(card *config.A2ACardConfig) *AgentBuilder {
	b.a2aCard = card
	return b
}

// A2ACard creates an A2A card builder
func (b *AgentBuilder) A2ACard() *A2ACardBuilder {
	if b.a2aCard == nil {
		b.a2aCard = &config.A2ACardConfig{}
	}
	return NewA2ACardBuilder(b.a2aCard)
}

// WithDatabase sets the database reference name (internal, used by long-term memory/context)
func (b *AgentBuilder) WithDatabase(dbName string) *AgentBuilder {
	b.database = dbName
	return b
}

// WithEmbedder sets the embedder reference name (internal, used by long-term memory/context)
func (b *AgentBuilder) WithEmbedder(embedderName string) *AgentBuilder {
	b.embedder = embedderName
	return b
}

// Build creates the agent using the programmatic API
func (b *AgentBuilder) Build() (*agent.Agent, error) {
	if b.llmProvider == nil {
		return nil, fmt.Errorf("LLM provider is required")
	}
	if b.reasoningStrategy == nil {
		return nil, fmt.Errorf("reasoning strategy is required")
	}
	if b.workingMemory == nil {
		return nil, fmt.Errorf("working memory strategy is required")
	}

	// Set defaults for long-term config
	longTermConfig := b.longTermConfig
	longTermConfig.SetDefaults()

	// Convert reasoning config to config.ReasoningConfig if available
	var reasoningConfig *config.ReasoningConfig
	if b.reasoningConfig != nil {
		reasoningConfig = &config.ReasoningConfig{
			Engine:          b.reasoningConfig.Engine,
			MaxIterations:   b.reasoningConfig.MaxIterations,
			EnableStreaming: b.reasoningConfig.EnableStreaming,
			ShowTools:       b.reasoningConfig.ShowTools,
			ShowThinking:    b.reasoningConfig.ShowThinking,
		}
	}

	// Build prompt config with IncludeContext
	promptConfig := config.PromptConfig{}
	if b.systemPrompt != "" {
		promptConfig.SystemPrompt = b.systemPrompt
	}
	if b.promptSlots != nil {
		promptConfig.PromptSlots = &config.PromptSlotsConfig{
			SystemRole:   b.promptSlots.SystemRole,
			Instructions: b.promptSlots.Instructions,
			UserGuidance: b.promptSlots.UserGuidance,
		}
	}
	if b.includeContext != nil {
		promptConfig.IncludeContext = b.includeContext
	}

	// Use direct agent constructor
	return agent.NewAgentDirect(agent.AgentBuilderOptions{
		ID:                 b.id,
		Name:               b.name,
		Description:        b.description,
		LLMProvider:        b.llmProvider,
		ReasoningStrategy:  b.reasoningStrategy,
		ReasoningConfig:    reasoningConfig,
		WorkingMemory:      b.workingMemory,
		LongTermMemory:     b.longTermMemory,
		LongTermConfig:     longTermConfig,
		Tools:              b.toolList,
		SystemPrompt:       b.systemPrompt,
		PromptSlots:        b.promptSlots,
		IncludeContext:     b.includeContext,
		Registry:           b.registry,
		BaseURL:            b.baseURL,
		PreferredTransport: b.preferredTransport,
		SessionService:     b.sessionService,
		ContextService:     b.contextService,
		TaskService:        b.taskService,
		TaskConfig:         b.taskConfig,
	})
}
