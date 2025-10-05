# Hector Architecture

**Design Philosophy:** Clean architecture with Strategy pattern, dependency injection, and single responsibility principle.

---

## System Overview

### Single Agent Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   USER INTERFACE                        │
│                (CLI, Streaming Output)                  │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│                      AGENT                              │
│  • Orchestrates reasoning loop                          │
│  • Manages conversation state                           │
│  • Coordinates services                                 │
│  • Executes tool calls                                  │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│              REASONING STRATEGY                         │
│  • ChainOfThoughtStrategy (production)                  │
│  • Hooks: Prepare, ShouldStop, AfterIteration           │
│  • GetContextInjection (todos, goals)                   │
│  • GetPromptSlots (customizable prompts)                │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│                   SERVICES                              │
│  ┌───────────┐  ┌──────────┐  ┌─────────┐  ┌─────────┐ │
│  │    LLM    │  │  Tools   │  │ Prompt  │  │ Context │ │
│  │ Service   │  │ Service  │  │ Service │  │ Service │ │
│  └───────────┘  └──────────┘  └─────────┘  └─────────┘ │
│  ┌───────────┐                                           │
│  │  History  │                                           │
│  │  Service  │                                           │
│  └───────────┘                                           │
└─────────────────────────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│              PROVIDERS & PLUGINS                        │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Built-in Providers:                            │   │
│  │  • Anthropic (Claude)                           │   │
│  │  • OpenAI (GPT-4)                               │   │
│  │  • Qdrant (Vector DB)                           │   │
│  │  • Ollama (Embeddings)                          │   │
│  └─────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Plugin System (gRPC):                          │   │
│  │  • Custom LLM Providers                         │   │
│  │  • Custom Database Providers                    │   │
│  │  • Custom Embedder Providers                    │   │
│  │  • Process isolation, auto-discovery            │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Multi-Agent Orchestration Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   USER INTERFACE                        │
│          (CLI with --workflow flag)                     │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│                      TEAM                               │
│  • Manages workflow lifecycle                           │
│  • Coordinates multiple agents                          │
│  • Shares context between agents                        │
│  • Streams events from all agents                       │
│                                                         │
│  Services:                                              │
│  ├─ TeamWorkflowService (executor management)          │
│  ├─ TeamAgentService (agent lifecycle)                 │
│  └─ TeamCoordinationService (context sharing)          │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│              WORKFLOW EXECUTOR                          │
│                                                         │
│  DAG Executor:                                          │
│  • Dependency-based execution                           │
│  • Parallel step execution                              │
│  • Context passing (${variables})                       │
│  • Progress tracking                                    │
│                                                         │
│  Autonomous Executor (Experimental):                    │
│  • Self-organizing workflows                            │
│  • Dynamic agent selection                              │
│  • Adaptive coordination                                │
└──────────────────────┬──────────────────────────────────┘
                       │
         ┌─────────────┼─────────────┐
         │             │             │
         ▼             ▼             ▼
    ┌────────┐    ┌────────┐    ┌────────┐
    │ Agent  │    │ Agent  │    │ Agent  │
    │   1    │    │   2    │    │   3    │
    └────┬───┘    └────┬───┘    └────┬───┘
         │             │             │
         └─────────────┼─────────────┘
                       │
                 Shared Context
              (Variables, Artifacts)
```

---

## Core Components

### 1. Team (`team/team.go`)

**Responsibility:** Multi-agent workflow orchestration

**Key Methods:**
- `ExecuteStreaming(input)` - Execute workflow with streaming
- `GetStatus()` - Get workflow status
- `GetSharedState()` - Access shared context
- `GetAgent(name)` - Retrieve specific agent

**Services:**
```go
type Team struct {
	workflowService     *TeamWorkflowService
	agentService        *TeamAgentService
	coordinationService *TeamCoordinationService
}
```

**TeamWorkflowService:**
- Manages workflow executors (DAG, Autonomous)
- Routes to appropriate executor based on mode
- Handles workflow streaming events

**TeamAgentService:**
- Creates and manages agent instances
- Provides agent capabilities query
- Handles agent lifecycle

**TeamCoordinationService:**
- Manages shared state across agents
- Context variable storage
- Inter-agent communication

### 2. Workflow Executors (`workflow/`)

**DAGExecutor (`workflow/executors.go`):**
```go
type DAGExecutor struct {
	name   string
	config *config.DAGExecution
}
```

**Capabilities:**
- Dependency resolution (`depends_on` field)
- Variable substitution (`${variable_name}`)
- Parallel execution of independent steps
- Progress tracking
- Error recovery with retries

**Execution Flow:**
1. Parse workflow steps and dependencies
2. Build dependency graph
3. Execute steps when dependencies satisfied
4. Pass outputs as inputs to dependent steps
5. Stream events for each step

**AutonomousExecutor (Experimental):**
- Dynamic workflow planning
- Self-organizing agent selection
- Adaptive goal pursuit
- Real-time coordination

### 3. Agent (`agent/agent.go`)

**Responsibility:** Orchestrate the reasoning loop

**Key Methods:**
- `Query(input)` - Non-streaming execution
- `QueryStreaming(input)` - Streaming execution
- `execute()` - Main reasoning loop
- `callLLM()` - LLM interaction
- `executeTools()` - Tool execution with dynamic labels
- `saveToHistory()` - Conversation persistence

**Design Pattern:** Orchestrator + Strategy

```go
type Agent struct {
	name        string
	description string
	config      *config.AgentConfig
	services    reasoning.AgentServices
}
```

### 2. Reasoning Strategy (`reasoning/chain_of_thought_strategy.go`)

**Responsibility:** Define reasoning behavior

**Interface:**
```go
type ReasoningStrategy interface {
	PrepareIteration(iteration int, state *ReasoningState) error
	ShouldStop(text string, toolCalls []ToolCall, state *ReasoningState) bool
	AfterIteration(iteration int, text string, toolCalls []ToolCall, results []ToolResult, state *ReasoningState) error
	GetContextInjection(state *ReasoningState) string
	GetPromptSlots() PromptSlots
	GetName() string
	GetDescription() string
}
```

**ChainOfThoughtStrategy:**
- Simple iterative reasoning
- Stops when no tool calls
- Self-reflection after each iteration
- Todo injection into context

### 3. Services (`agent/services.go`)

**Service Architecture:**

```go
type AgentServices interface {
	Config() config.ReasoningConfig
	LLM() LLMService
	Tools() ToolService
	Context() ContextService
	Prompt() PromptService
	History() HistoryService
}
```

**Why Services?**
- ✅ Dependency Injection
- ✅ Easy testing (mock services)
- ✅ Single Responsibility
- ✅ Clean interfaces

---

## Key Design Patterns

### 1. Strategy Pattern

**Problem:** Different reasoning approaches need different logic

**Solution:** `ReasoningStrategy` interface with implementations

```go
// Agent doesn't know HOW to reason
agent := NewAgent(config, services)

// Strategy defines HOW
strategy := NewChainOfThoughtStrategy()

// Agent uses strategy hooks
strategy.PrepareIteration(...)
strategy.ShouldStop(...)
strategy.AfterIteration(...)
```

**Benefits:**
- Easy to add new reasoning strategies
- Agent code stays simple
- Strategies are isolated and testable

### 2. Dependency Injection

**Problem:** Components need access to services

**Solution:** Inject services through constructor

```go
// Bad: Hard-coded dependencies
agent := &Agent{
	llm: openai.New(),  // Tightly coupled!
}

// Good: Injected dependencies
services := NewAgentServices(config, componentManager)
agent := NewAgent(config, services)
```

**Benefits:**
- Easy to test (inject mocks)
- Loose coupling
- Flexible composition

### 3. Service Locator

**Problem:** Need centralized service management

**Solution:** `ComponentManager` + Service Interfaces

```go
type ComponentManager struct {
	llmRegistry      *llms.LLMRegistry
	dbRegistry       *databases.DatabaseRegistry
	embedderRegistry *embedders.EmbedderRegistry
	toolRegistry     *tools.ToolRegistry
	pluginRegistry   *plugins.PluginRegistry
}

// Usage
llm, _ := componentManager.GetLLM("main-llm")
tools := componentManager.GetToolRegistry()
```

**Benefits:**
- Single source of truth
- Lazy initialization
- Easy to swap implementations

### 4. Plugin Architecture

**Problem:** Need to extend Hector without modifying core code

**Solution:** gRPC-based plugin system with HashiCorp go-plugin

```go
// Plugin interface
type Plugin interface {
	Initialize(ctx context.Context, config map[string]string) error
	Shutdown(ctx context.Context) error
	Health(ctx context.Context) error
}

// Example: LLM Plugin
type LLMProvider interface {
	Plugin
	Generate(ctx context.Context, messages []*Message, tools []*ToolDefinition) (*GenerateResponse, error)
	GenerateStreaming(ctx context.Context, messages []*Message, tools []*ToolDefinition) (<-chan *StreamChunk, error)
	GetModelInfo(ctx context.Context) (*ModelInfo, error)
}
```

**Architecture:**
```
┌──────────────┐         ┌──────────────┐         ┌──────────────┐
│   Hector     │         │  go-plugin   │         │   Plugin     │
│   Core       │         │   Framework  │         │   Process    │
└──────┬───────┘         └──────┬───────┘         └──────┬───────┘
       │                        │                        │
       │ 1. Load                │                        │
       ├───────────────────────>│                        │
       │                        │ 2. Start Process       │
       │                        ├───────────────────────>│
       │                        │                        │
       │                        │ 3. gRPC Handshake      │
       │                        │<═══════════════════════│
       │                        │                        │
       │ 4. Interface           │                        │
       │<───────────────────────┤                        │
       │                        │                        │
       │ 5. RPC Call            │                        │
       ├───────────────────────>│ 6. gRPC               │
       │                        ├═══════════════════════>│
       │                        │ 7. Response            │
       │ 8. Result              │<═══════════════════════│
       │<───────────────────────┤                        │
```

**Benefits:**
- ✅ **Process Isolation**: Plugin crashes don't affect Hector
- ✅ **Language Agnostic**: Plugins can be in any language (via gRPC)
- ✅ **Auto-Discovery**: Drop plugins in directory, Hector finds them
- ✅ **Hot-Pluggable**: Add providers via configuration only
- ✅ **Production Proven**: Built on HashiCorp go-plugin (used by Terraform, Vault)

---

## Data Flow

### Request Flow

```
User Input
    │
    ▼
┌─────────────────────┐
│  Agent.Query()      │
│  Agent.execute()    │  ← Main Loop
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Strategy.Prepare()  │  ← Hook: Prepare iteration
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ PromptService       │  ← Build messages with slots
│ .BuildMessages()    │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ LLMService          │  ← Call LLM (Anthropic/OpenAI)
│ .Generate()         │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ ToolService         │  ← Execute tools if any
│ .ExecuteToolCall()  │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Strategy            │  ← Hook: Self-reflection
│ .AfterIteration()   │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ Strategy            │  ← Hook: Check if done
│ .ShouldStop()       │
└──────┬──────────────┘
       │
       ▼ (loop or done)
   Response
```

### Streaming Flow

```
User Input
    │
    ▼
Agent creates channel ──────────────┐
    │                               │
    ▼                               │
LLM streams to channel              │
    │                               │
    ├─► Text chunks ────────────────┤
    ├─► Tool call chunks ───────────┤
    └─► Done signal ────────────────┤
                                    │
                                    ▼
                            User sees output
                            (real-time)
```

---

## Module Responsibilities

### `team/`
- Multi-agent orchestration
- Workflow coordination
- Context sharing between agents
- Team services (workflow, agent, coordination)

### `workflow/`
- Workflow executors (DAG, Autonomous)
- Dependency management
- Variable substitution
- Streaming events
- Executor registry and factory

### `agent/`
- Single-agent orchestration
- Tool execution with dynamic labels
- History management
- Grayed-out debug output

### `reasoning/`
- Strategy interface
- ChainOfThoughtStrategy
- State management
- Prompt slots

### `llms/`
- Provider implementations (Anthropic, OpenAI)
- Native function calling
- Streaming support
- Rate limit handling
- LLM registry

### `databases/`
- Database provider implementations (Qdrant)
- Vector operations
- Database registry

### `embedders/`
- Embedder provider implementations (Ollama)
- Text embedding
- Embedder registry

### `tools/`
- Tool implementations
- Tool registry
- Local tools (execute_command, file_writer, etc.)
- MCP tool integration (foundation)

### `plugins/`
- Plugin system core (registry, discovery, types)
- gRPC plugin implementation
- Protocol Buffer definitions
- Plugin adapters (LLM, Database, Embedder)
- Plugin lifecycle management
- Health monitoring and restart

### `component/`
- ComponentManager (service locator)
- Registry management
- Plugin initialization
- Global configuration

### `config/`
- Configuration types
- Workflow configuration
- Plugin configuration
- Validation
- Defaults

### `context/`
- Semantic search
- Document stores
- Vector operations

---

## Extension Points

### 1. Plugin System (Recommended for Providers)

**For LLM, Database, and Embedder providers, use the plugin system instead of code-level extensions.**

#### Create an LLM Plugin

```go
// Separate executable
package main

import (
	"context"
	"github.com/kadirpekel/hector/plugins/grpc"
)

type MyLLMProvider struct{}

func (p *MyLLMProvider) Initialize(ctx context.Context, config map[string]string) error {
	// Initialize your LLM client
	return nil
}

func (p *MyLLMProvider) Generate(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (*grpc.GenerateResponse, error) {
	// Your LLM implementation
	return &grpc.GenerateResponse{Text: "response"}, nil
}

func (p *MyLLMProvider) GenerateStreaming(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (<-chan *grpc.StreamChunk, error) {
	// Your streaming implementation
}

func (p *MyLLMProvider) GetModelInfo(ctx context.Context) (*grpc.ModelInfo, error) {
	return &grpc.ModelInfo{ModelName: "my-model"}, nil
}

func (p *MyLLMProvider) Shutdown(ctx context.Context) error {
	return nil
}

func (p *MyLLMProvider) Health(ctx context.Context) error {
	return nil
}

func main() {
	grpc.ServeLLMPlugin(&MyLLMProvider{})
}
```

**Configuration:**
```yaml
plugins:
  llm_providers:
    my-llm:
      type: grpc
      path: "./plugins/my-llm"
      enabled: true
      config:
        api_key: "${MY_API_KEY}"

agents:
  my-agent:
    llm: "my-llm"
```

**Benefits:**
- ✅ Zero changes to Hector core
- ✅ Isolated process (crash-safe)
- ✅ Language agnostic
- ✅ Hot-pluggable via config

**See**: [Plugin Development Guide](examples/plugins/README.md) | [Echo LLM Example](examples/plugins/echo-llm/)

### 2. New Reasoning Strategy (Code-Level)

```go
type MyStrategy struct{}

func (s *MyStrategy) PrepareIteration(...) error {
	// Your logic
}

func (s *MyStrategy) ShouldStop(...) bool {
	// Your stopping condition
}

func (s *MyStrategy) AfterIteration(...) error {
	// Your post-processing
}

func (s *MyStrategy) GetPromptSlots() PromptSlots {
	// Your default prompts
}

// Register in reasoning/factory.go
```

### 3. New Tool (Code-Level)

```go
type MyTool struct{}

func (t *MyTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name: "my_tool",
		Description: "Does something useful",
		Parameters: []ToolParameter{...},
	}
}

func (t *MyTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	// Your tool logic
}

// Register in tools/local.go
```

### 4. Built-in Provider (Advanced, Not Recommended)

Only for providers that need deep integration with Hector internals.

```go
// llms/myprovider.go
type MyProvider struct {
	config config.LLMProviderConfig
	client *http.Client
}

func (p *MyProvider) Generate(messages []Message, tools []ToolDefinition) (string, []ToolCall, int, error) {
	// Your implementation
}

func (p *MyProvider) GenerateStreaming(...) (<-chan StreamChunk, error) {
	// Your streaming implementation
}

// Register in llms/registry.go
```

**Note:** Prefer plugins for extensibility. Built-in providers require modifying Hector core.

---

## Best Practices

### 1. Separation of Concerns

**Good:**
```go
// Agent: Orchestration only
func (a *Agent) execute() {
	strategy.PrepareIteration()
	llm.Generate()
	tools.Execute()
	strategy.AfterIteration()
}

// Strategy: Reasoning logic only
func (s *ChainOfThoughtStrategy) ShouldStop(...) bool {
	return len(toolCalls) == 0
}
```

**Bad:**
```go
// Agent doing everything (❌)
func (a *Agent) execute() {
	// Reasoning logic mixed with orchestration
	if iteration > 5 && confidence > 0.8 {
		return
	}
	// Tool execution logic
	// Reflection logic
	// ...
}
```

### 2. Dependency Direction

```
Low-level (concrete) → High-level (abstract)

llms/anthropic.go  ────►  llms/types.go
                           (interfaces)
                              ▲
                              │
agent/services.go  ───────────┘
(implementations)

reasoning/strategy.go  ────►  reasoning/interfaces.go
                               (interfaces)
                                  ▲
                                  │
agent/agent.go  ───────────────────┘
(uses strategies)
```

**Rule:** High-level modules depend on abstractions, not concretions.

### 3. Interface Segregation

**Good:**
```go
// Small, focused interfaces
type LLMService interface {
	Generate(...) (string, []ToolCall, int, error)
	GenerateStreaming(...) (<-chan StreamChunk, error)
}

type ToolService interface {
	ExecuteToolCall(...) (string, error)
	GetAvailableTools() ([]ToolDefinition, error)
}
```

**Bad:**
```go
// Huge, monolithic interface (❌)
type Service interface {
	Generate(...)
	GenerateStreaming(...)
	ExecuteToolCall(...)
	GetTools(...)
	BuildPrompt(...)
	SaveHistory(...)
	// ... 20 more methods
}
```

---

## Performance Considerations

### 1. Streaming Optimization

```go
// Use buffered channels
outputCh := make(chan string, 100)  // Buffer size matters

// Stream chunks immediately
for chunk := range llmChunks {
	outputCh <- chunk.Text  // Don't accumulate, stream!
}
```

### 2. Tool Execution

```go
// Tools run sequentially (current)
for _, toolCall := range toolCalls {
	result := tools.ExecuteToolCall(toolCall)
	results = append(results, result)
}

// Future: Parallel execution
results := executeToolsInParallel(toolCalls)
```

### 3. Context Management

```go
// Limit history to prevent token overflow
maxHistory := 10
recentMessages := history.GetRecentHistory(maxHistory)

// Use semantic search only when needed
if config.IncludeContext {
	context := contextService.SearchContext(query)
}
```

---

## Testing Strategy

### Unit Tests

```go
// Test services with mocks
func TestAgent_Execute(t *testing.T) {
	mockLLM := &MockLLMService{}
	mockTools := &MockToolService{}
	services := NewMockServices(mockLLM, mockTools)
	
	agent := NewAgent(config, services)
	response := agent.Query("test")
	
	assert.NotEmpty(t, response)
}
```

### Integration Tests

```go
// Test with real providers
func TestRealLLM(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	
	config := loadTestConfig()
	agent := CreateAgent(config)
	response := agent.Query("What is 2+2?")
	
	assert.Contains(t, response, "4")
}
```

---

## Multi-Agent Workflow Execution

### DAG Workflow Example

**Configuration:**
```yaml
workflows:
  research_pipeline:
    mode: "dag"
    execution:
      dag:
        steps:
          - name: "research"
            agent: "researcher"
            input: "${user_input}"
            output: "research_data"
          
          - name: "analyze"
            agent: "analyst"
            input: "Analyze: ${research_data}"
            depends_on: [research]
            output: "analysis"
          
          - name: "report"
            agent: "writer"
            input: "Report: ${research_data}, ${analysis}"
            depends_on: [research, analyze]
```

**Execution Steps:**
1. **Step 1**: Researcher executes with user input
2. **Context Storage**: `research_data = researcher_output`
3. **Step 2**: Analyst waits for researcher, then executes
4. **Context Storage**: `analysis = analyst_output`
5. **Step 3**: Writer waits for both, then executes with both outputs
6. **Final Output**: Writer's response

**Variable Substitution:**
```go
// Template: "Analyze: ${research_data}"
// Resolution: workflowContext.Variables["research_data"]
// Result: "Analyze: [actual researcher output]"
```

### Event Streaming

**Event Types:**
```go
const (
	EventWorkflowStart  WorkflowEventType = "workflow_start"
	EventWorkflowEnd    WorkflowEventType = "workflow_end"
	EventAgentStart     WorkflowEventType = "agent_start"
	EventAgentThinking  WorkflowEventType = "agent_thinking"
	EventAgentOutput    WorkflowEventType = "agent_output"
	EventAgentComplete  WorkflowEventType = "agent_complete"
	EventAgentError     WorkflowEventType = "agent_error"
	EventStepStart      WorkflowEventType = "step_start"
	EventStepComplete   WorkflowEventType = "step_complete"
	EventProgress       WorkflowEventType = "progress"
)
```

**Event Flow:**
```
Team.ExecuteStreaming()
    │
    ├─► EventWorkflowStart
    │
    ├─► EventStepStart (research)
    │   ├─► EventAgentStart (researcher)
    │   ├─► EventAgentThinking
    │   ├─► EventAgentOutput
    │   └─► EventAgentComplete
    │
    ├─► EventStepComplete (research)
    │
    ├─► EventStepStart (analyze)
    │   └─► ... (analyst events)
    │
    ├─► EventStepComplete (analyze)
    │
    ├─► EventStepStart (report)
    │   └─► ... (writer events)
    │
    ├─► EventStepComplete (report)
    │
    └─► EventWorkflowEnd
```

### Context Sharing

**Shared State:**
```go
type SharedState struct {
	Variables map[string]string
	History   []AgentInteraction
	Artifacts map[string]Artifact
}
```

**Usage:**
```go
// Agent 1 stores data
coordinationService.SetContext("research_data", output, "researcher")

// Agent 2 retrieves data
input := coordinationService.GetContext("research_data")
```

**Benefits:**
- Agents can build on each other's work
- No manual data passing required
- Full audit trail of context changes

---

## Future Enhancements

### Short Term
- Production-ready DAG executor testing
- Autonomous mode improvements
- Better error recovery in workflows
- More example plugins (OpenRouter, Cohere, etc.)

### Medium Term
- Conditional workflow steps (if/else)
- Loop constructs (for-each agent)
- Workflow templates and reuse
- MCP tool examples and documentation
- Plugin marketplace/registry
- Plugin sandboxing/permissions

### Long Term
- Visual workflow designer
- Workflow marketplace
- Advanced reasoning strategies (ToT, Reflexion)
- Plugin hot-reload
- Plugin metrics and observability

---

## References

- **SOLID Principles**: https://en.wikipedia.org/wiki/SOLID
- **Strategy Pattern**: https://refactoring.guru/design-patterns/strategy
- **Dependency Injection**: https://martinfowler.com/articles/injection.html
- **Clean Architecture**: https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html
- **Plugin Architecture**: [PLUGIN_ARCHITECTURE.md](PLUGIN_ARCHITECTURE.md)
- **HashiCorp go-plugin**: https://github.com/hashicorp/go-plugin

---

**Last Updated:** October 5, 2025

