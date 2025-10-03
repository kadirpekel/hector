# Hector Architecture

**Design Philosophy:** Clean architecture with Strategy pattern, dependency injection, and single responsibility principle.

---

## System Overview

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
│                 LLM PROVIDERS                           │
│  • Anthropic (Claude)                                   │
│  • OpenAI (GPT-4)                                       │
│  • Native function calling                              │
│  • Streaming support                                    │
│  • Rate limit handling                                  │
└─────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. Agent (`agent/agent.go`)

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
}

// Usage
llm, _ := componentManager.GetLLM("main-llm")
tools := componentManager.GetToolRegistry()
```

**Benefits:**
- Single source of truth
- Lazy initialization
- Easy to swap implementations

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

### `agent/`
- Orchestration
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

### `tools/`
- Tool implementations
- Tool registry
- Local tools (execute_command, file_writer, etc.)

### `config/`
- Configuration types
- Validation
- Defaults

### `context/`
- Semantic search
- Document stores
- Vector operations

---

## Extension Points

### 1. New Reasoning Strategy

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

### 2. New LLM Provider

```go
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

### 3. New Tool

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

## Future Enhancements

### Short Term
- Parallel tool execution
- Auto-todo creation improvements
- Context window smart truncation

### Medium Term
- Linter integration
- Multi-file batch operations
- Advanced error recovery

### Long Term
- Multi-agent workflows
- Custom reasoning strategies
- Plugin system

---

## References

- **SOLID Principles**: https://en.wikipedia.org/wiki/SOLID
- **Strategy Pattern**: https://refactoring.guru/design-patterns/strategy
- **Dependency Injection**: https://martinfowler.com/articles/injection.html
- **Clean Architecture**: https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html

---

**Last Updated:** October 4, 2025

