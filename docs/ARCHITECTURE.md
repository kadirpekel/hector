---
layout: default
title: Architecture
nav_order: 1
parent: Advanced
description: "System design, multi-agent orchestration, and core components"
---

<style>
.architecture-diagram {
  background: var(--code-background-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 1.5rem;
  margin: 1.5rem 0;
  overflow-x: auto;
}

.architecture-diagram pre {
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.85rem;
  line-height: 1.2;
  color: var(--body-text-color);
  background: transparent;
  border: none;
  padding: 0;
}
</style>

# Hector Architecture

**Design Philosophy:** Clean architecture with Strategy pattern, dependency injection, and single responsibility principle.

---

## System Overview

### Single Agent Architecture

<div class="architecture-diagram">
<pre>
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
│  ┌───────────────────────────────────────────────────┐  │
│  │  MemoryService (pkg/memory/)                      │  │
│  │  ┌─────────────────────────────────────────────┐  │  │
│  │  │ WorkingMemoryStrategy (injected)            │  │  │
│  │  │ • SummaryBufferStrategy (default)           │  │  │
│  │  │ • BufferWindowStrategy                      │  │  │
│  │  └─────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────┘  │
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
</pre>
</div>

### Multi-Agent Orchestration (A2A Protocol)

Hector uses the **A2A (Agent-to-Agent) protocol** for multi-agent orchestration. All agents (native and external) are A2A-compliant peers that can be orchestrated through a supervisor agent. External agents can be integrated **declaratively via YAML configuration** without writing any code.

**📖 Complete Multi-Agent Tutorial:** See our [LangChain vs Hector comparison](tutorials/MULTI_AGENT_RESEARCH_PIPELINE.md) for a detailed walkthrough of building a 3-agent research system, including direct code comparisons showing how Hector's YAML approach compares to traditional Python-based frameworks.

```
┌─────────────────────────────────────────────────────────┐
│                   USER / CLIENT                         │
│            (CLI, API, External A2A Client)              │
└──────────────────────┬──────────────────────────────────┘
                       │ A2A Protocol (HTTP/JSON)
┌──────────────────────▼──────────────────────────────────┐
│                   A2A SERVER                            │
│  • Agent discovery (/agents endpoint)                   │
│  • Task execution (/agents/{id}/tasks)                  │
│  • Session management                                   │
│  • Pure protocol compliance                             │
└──────────────────────┬──────────────────────────────────┘
                       │
         ┌─────────────┼─────────────────────┐
         │             │                     │
         ▼             ▼                     ▼
┌─────────────┐  ┌─────────────┐    ┌─────────────┐
│ Orchestrator│  │  Specialist │    │  Specialist │
│   Agent     │  │   Agent 1   │    │   Agent 2   │
│             │  │             │    │             │
│ Tools:      │  │ Tools:      │    │ Tools:      │
│ • agent_call│  │ • domain    │    │ • domain    │
│             │  │   specific  │    │   specific  │
└──────┬──────┘  └─────────────┘    └─────────────┘
       │
       │ agent_call(agent_id, task)
       │
       └──────────┐
                  ▼
          ┌────────────────┐
          │  AgentRegistry │
          │  (all agents)  │
          └────────────────┘

ORCHESTRATION FLOW:
1. User calls Orchestrator Agent
2. Orchestrator analyzes task, decomposes into subtasks
3. Orchestrator delegates via agent_call tool
4. Target agents execute (native or remote A2A agents)
5. Orchestrator synthesizes results
6. Returns unified response
```

**Key Features:**
- **Pure A2A Protocol**: All agents comply with A2A specification
- **Transparent Delegation**: Orchestrator uses `agent_call` tool
- **Native + Remote**: Supports both in-process and remote A2A agents
- **Declarative External Agents**: Define external agents via URL in YAML config
- **LLM-Driven Routing**: Orchestrator decides delegation dynamically
- **Composable**: Orchestrators can call other orchestrators
- **Agent Ecosystem Ready**: Enables interoperability within organizations and the broader agent internet

---

## Core Components

### 1. A2A Server (`a2a/server.go`)

**Responsibility:** Host agents via A2A protocol

**Key Features:**
- Agent discovery (GET /agents)
- Task execution (POST /agents/{id}/tasks)
- Session management
- Streaming support
- Pure protocol compliance

**Core Methods:**
```go
func (s *Server) RegisterAgent(agentID string, agent Agent) error
func (s *Server) Start() error
func (s *Server) Stop(ctx context.Context) error
```

### 2. Agent (`agent/agent.go`)

**Responsibility:** Execute reasoning tasks via A2A protocol

**A2A Interface Implementation:**
```go
// Pure A2A Agent interface
func (a *Agent) GetAgentCard() *a2a.AgentCard
func (a *Agent) ExecuteTask(ctx context.Context, request *a2a.TaskRequest) (*a2a.TaskResponse, error)
func (a *Agent) ExecuteTaskStreaming(ctx context.Context, request *a2a.TaskRequest) (<-chan *a2a.StreamChunk, error)
```

**Internal Methods:**
- `execute()` - Main reasoning loop
- `callLLM()` - LLM interaction
- `executeTools()` - Tool execution
- `saveToHistory()` - Conversation persistence

**Design Pattern:** Strategy Pattern

```go
type Agent struct {
	name        string
	description string
	config      *config.AgentConfig
	services    reasoning.AgentServices
}
```

**Key Features:**
- Pure A2A compliance (no legacy Query/QueryStreaming)
- Direct protocol implementation
- Transparent to clients (native or remote)
- Tool-based orchestration via `agent_call`

### 3. Orchestration Tools (`agent/agent_call_tool.go`)

**Responsibility:** Enable multi-agent coordination

**`agent_call` Tool:**
```go
type AgentCallTool struct {
	registry *AgentRegistry
}

// Delegates task to another agent
func (t *AgentCallTool) Execute(ctx context.Context, args map[string]interface{}) (tools.ToolResult, error)
```

**Features:**
- Transparent delegation to any registered agent
- Works with both native and remote A2A agents
- Pure A2A protocol communication
- Enables LLM-driven orchestration

**Usage:**
```yaml
agents:
  # Native agent
  researcher:
    name: "Research Agent"
    llm: "gpt-4"
    reasoning:
      engine: "chain-of-thought"
  
  # External A2A agent (pure interoperability!)
  partner_specialist:
    type: "a2a"
    url: "https://partner-ai.com/agents/specialist"
  
  # Orchestrator coordinates both
  orchestrator:
    name: "Hybrid Orchestrator"
    llm: "gpt-4"
    tools:
      - agent_call  # Can call native AND external agents
    reasoning:
      engine: "supervisor"  # Optimized for delegation
```

**Benefits:**
- 🌍 **Agent Internet**: Connect to the emerging agent ecosystem
- 🏢 **Enterprise Interoperability**: Integrate partner/vendor agents without code
- 📝 **Declarative**: External agents defined in YAML like native ones
- 🔌 **Zero Code**: No API integration, no custom connectors

### 4. Reasoning Strategy (`reasoning/chain_of_thought_strategy.go`, `reasoning/supervisor_strategy.go`)

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

### 5. Memory Service (`pkg/memory/`)

**Responsibility:** Manage conversation memory across sessions

**Architecture:**

```go
// MemoryService orchestrates session management and delegates to strategies
type MemoryService struct {
	workingMemory WorkingMemoryStrategy
	sessions      map[string]*ConversationHistory
	mu            sync.RWMutex
}

// WorkingMemoryStrategy defines pluggable memory algorithms
type WorkingMemoryStrategy interface {
	AddMessage(session *ConversationHistory, msg llms.Message) error
	GetMessages(session *ConversationHistory) ([]llms.Message, error)
	Name() string
	SetStatusNotifier(notifier StatusNotifier)
}
```

**Strategies:**
- **`SummaryBufferStrategy`** (default) - Token-based with LLM summarization
- **`BufferWindowStrategy`** - Simple LIFO (last N messages)

**Design Benefits:**
- ✅ Clean separation: Service manages sessions, strategies implement algorithms
- ✅ Strategy Pattern: Easy to add new memory strategies
- ✅ No duplication: Session management centralized
- ✅ Extensible: Foundation for long-term memory

**File Structure:**
```
pkg/memory/
├── memory.go            → MemoryService (orchestrator)
├── working_strategy.go  → WorkingMemoryStrategy interface
├── summary_buffer.go    → Token-based strategy
├── buffer_window.go     → Simple LIFO strategy
└── factory.go           → Strategy factory
```

### 6. Services (`agent/services.go`)

**Service Architecture:**

```go
type AgentServices interface {
	Config() config.ReasoningConfig
	LLM() LLMService
	Tools() ToolService
	Context() ContextService
	Prompt() PromptService
	History() HistoryService  // Returns MemoryService
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
- Prompt services and customization
- Agent factory and services

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
- Local tools (execute_command, write_file, etc.)
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

### `memory/`
- Conversation memory management
- Pluggable working memory strategies
- Session lifecycle management
- SummaryBufferStrategy (token-based with LLM summarization)
- BufferWindowStrategy (simple LIFO)
- Foundation for long-term memory

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

## Sessions and Streaming

### Session Management

**Full A2A Protocol Support:**

```
POST   /sessions              # Create new session
GET    /sessions              # List sessions
GET    /sessions/{id}         # Get session details
DELETE /sessions/{id}         # End session
POST   /sessions/{id}/tasks   # Execute task in session context
```

**Features:**
- ✅ Multi-turn conversations with context
- ✅ Session state management
- ✅ Per-session conversation history
- ✅ Metadata support
- ✅ Activity tracking (lastActivityAt)

**Implementation:**
- **Storage:** In-memory (`map[string]*Session`)
- **Lifecycle:** Sessions survive until explicitly deleted or server restart
- **Future:** Persistent storage (Redis, PostgreSQL)

**Example:**
```bash
# Create session
SESSION=$(curl -s -X POST http://localhost:8080/sessions \
  -d '{"agentId": "assistant"}' | jq -r '.sessionId')

# Chat with context
curl -X POST http://localhost:8080/sessions/$SESSION/tasks \
  -d '{"input":{"type":"text/plain","content":"My name is Alice"}}'

# Agent remembers context
curl -X POST http://localhost:8080/sessions/$SESSION/tasks \
  -d '{"input":{"type":"text/plain","content":"What is my name?"}}'
```

### SSE Streaming (A2A Compliant)

**Real-Time Output:**

```
POST /agents/{agentId}/message/stream
```

**Features:**
- ✅ Real-time output streaming per A2A specification
- ✅ Token-by-token delivery (for LLM streaming)
- ✅ Server-Sent Events (SSE) protocol
- ✅ Multiple event types (status, message, artifact)

**Implementation:**
- **Protocol:** Server-Sent Events (SSE) per A2A spec Section 7
- **Format:** SSE event stream with JSON data payloads
- **Events:** status, message, artifact
- **Backpressure:** Go channels handle it naturally

**Example:**
```bash
curl -N -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"message":{"role":"user","parts":[{"type":"text","text":"Write a poem"}]}}' \
  http://localhost:8080/agents/assistant/message/stream

# Output:
# event: status
# data: {"task_id":"task-123","status":{"state":"working"}}
#
# event: message
# data: {"task_id":"task-123","message":{"role":"assistant","parts":[{"type":"text","text":"Roses are red..."}]}}
#
# event: status
# data: {"task_id":"task-123","status":{"state":"completed"}}
```

**Resume Streaming:**
```
POST /agents/{agentId}/tasks/{taskId}/resubscribe
```

Allows reconnecting to an in-progress task and resuming from a specific event.

---

## Orchestrator Pattern

### Design Decision: Regular Agent + `agent_call` Tool

**Why not a special orchestrator implementation?**

1. **Industry Alignment** - OpenAI, Anthropic use function calling, not special agent types
2. **Pure A2A Philosophy** - All agents implement the same interface
3. **Composability** - Any agent can become an orchestrator with `agent_call` tool

**Configuration:**
```yaml
agents:
  researcher:
    name: "Research Agent"
    llm: "gpt-4o-mini"
  
  analyst:
    name: "Analysis Agent"
    llm: "gpt-4o-mini"
  
  orchestrator:
    name: "Orchestrator"
    llm: "gpt-4o"  # More capable model
    tools:
      - agent_call  # Enable delegation
    reasoning:
      engine: "supervisor"  # Optimized for orchestration
    prompt:
      system_role: |
        You are an orchestrator that coordinates other agents.
        Available agents: researcher, analyst
        Use agent_call to delegate tasks.
```

**Execution Flow:**
```
User Request
    ↓
Orchestrator Agent (supervisor reasoning)
    ↓
Decides: "Need research first"
    ↓
agent_call("researcher", task="Research topic X")
    ↓
Researcher executes via A2A protocol
    ↓
Returns results to Orchestrator
    ↓
Orchestrator decides: "Now analyze"
    ↓
agent_call("analyst", task="Analyze: [research results]")
    ↓
Analyst executes
    ↓
Orchestrator synthesizes final answer
```

**Benefits:**
- ✅ No special orchestrator class needed
- ✅ Native and external agents treated identically
- ✅ LLM makes routing decisions dynamically
- ✅ Same code path for all agents

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

