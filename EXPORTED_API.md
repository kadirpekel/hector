# Hector Public Go API

This document lists exported functions and types intended for use by external Go applications consuming Hector as a library.

**Last Updated**: 2025-11-11

---

## Core Principles

1. **Programmatic API First**: Use `pkg/hector` builders for all programmatic agent construction
2. **Low-Level APIs Available**: Direct constructors in `pkg/llms`, `pkg/databases`, etc. are available but prefer builders
3. **Config-Driven Alternative**: YAML configuration is available as a convenience layer
4. **Internal Packages**: Code in `pkg/*/internal/` is NOT part of the public API

---

## Recommended: Programmatic API (`pkg/hector`)

**The programmatic API is the primary way to use Hector programmatically.** All builders are in the `pkg/hector` package.

### Quick Start

```go
import "github.com/kadirpekel/hector/pkg/hector"

// Build agent programmatically
agent, err := hector.NewAgent("my-agent").
    WithName("My Agent").
    WithLLMProvider(
        hector.NewLLMProvider("openai").
            Model("gpt-4o").
            APIKeyFromEnv("OPENAI_API_KEY").
            Build(),
    ).
    WithReasoningStrategy(
        hector.NewReasoning("chain-of-thought").
            MaxIterations(100).
            Build(),
    ).
    Build()
```

### Available Builders

- **`hector.NewAgent()`** - Agent builder
- **`hector.NewLLMProvider()`** - LLM provider builder
- **`hector.NewReasoning()`** - Reasoning strategy builder
- **`hector.NewWorkingMemory()`** - Working memory builder
- **`hector.NewLongTermMemory()`** - Long-term memory builder
- **`hector.NewContextService()`** - RAG/context service builder
- **`hector.NewDocumentStore()`** - Document store builder
- **`hector.NewTaskService()`** - Task service builder
- **`hector.NewSessionService()`** - Session service builder
- **`hector.NewDatabase()`** - Database provider builder
- **`hector.NewEmbedder()`** - Embedder builder
- **`hector.NewObservability()`** - Observability builder
- **`hector.NewSecurityBuilder()`** - Security configuration builder
- **`hector.NewA2ACardBuilder()`** - A2A card builder
- **`hector.NewStructuredOutput()`** - Structured output builder
- **`hector.NewConfigAgentBuilder()`** - Config-to-programmatic bridge

**See**: [Programmatic API Reference](../docs/reference/programmatic-api.md) for complete documentation.

---

## Low-Level APIs (Advanced)

Low-level APIs are available for advanced use cases but should generally be used through the programmatic API builders.

### LLM Providers

```go
import "github.com/kadirpekel/hector/pkg/llms"
import "github.com/kadirpekel/hector/pkg/config"

// Direct provider construction (prefer hector.NewLLMProvider() instead)
func NewOpenAIProviderFromConfig(config *config.LLMProviderConfig) (llms.LLMProvider, error)
func NewAnthropicProviderFromConfig(config *config.LLMProviderConfig) (llms.LLMProvider, error)
func NewGeminiProviderFromConfig(config *config.LLMProviderConfig) (llms.LLMProvider, error)
func NewOllamaProviderFromConfig(config *config.LLMProviderConfig) (llms.LLMProvider, error)
```

**When to Use**: Only if you need direct control or are building custom LLM providers.

**Recommended**: Use `hector.NewLLMProvider("openai").Build()` instead.

---

### Database Providers

```go
import "github.com/kadirpekel/hector/pkg/databases"
import "github.com/kadirpekel/hector/pkg/config"

// Direct database construction (prefer hector.NewDatabase() instead)
func NewQdrantDatabaseProviderFromConfig(config *config.DatabaseProviderConfig) (databases.DatabaseProvider, error)
```

**When to Use**: Only if you need direct control or are building custom database providers.

**Recommended**: Use `hector.NewDatabase("qdrant").Build()` instead.

---

### Embedders

```go
import "github.com/kadirpekel/hector/pkg/embedders"
import "github.com/kadirpekel/hector/pkg/config"

// Direct embedder construction (prefer hector.NewEmbedder() instead)
func NewOllamaEmbedderFromConfig(config *config.EmbedderProviderConfig) (embedders.EmbedderProvider, error)
func NewOpenAIEmbedderFromConfig(config *config.EmbedderProviderConfig) (embedders.EmbedderProvider, error)
func NewCohereEmbedderFromConfig(config *config.EmbedderProviderConfig) (embedders.EmbedderProvider, error)
```

**When to Use**: Only if you need direct control or are building custom embedders.

**Recommended**: Use `hector.NewEmbedder("ollama").Build()` instead.

---

### Tools

```go
import "github.com/kadirpekel/hector/pkg/tools"

// Tool registry for custom tools
type ToolRegistry struct { ... }
func NewToolRegistry() *ToolRegistry
func (r *ToolRegistry) RegisterTool(tool Tool) error
func (r *ToolRegistry) GetTool(name string) (Tool, error)
func (r *ToolRegistry) ListTools() []ToolInfo

// Tool interface for custom tools
type Tool interface {
    GetInfo() ToolInfo
    Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error)
    GetName() string
    GetDescription() string
}
```

**Usage**: Create custom tools implementing the `Tool` interface, then register them with agents.

**Example**:
```go
// Create custom tool
customTool := &MyCustomTool{}

// Use with agent builder
agent, err := hector.NewAgent("agent").
    WithTools(customTool).
    Build()
```

**See**: [Extend Hector](../docs/how-to/extend-hector.md) for complete examples.

---

### Reasoning Strategies

```go
import "github.com/kadirpekel/hector/pkg/reasoning"

// ReasoningStrategy interface for custom reasoning engines
type ReasoningStrategy interface {
    PrepareIteration(iteration int, state *ReasoningState) error
    ShouldStop(text string, toolCalls []*protocol.ToolCall, state *ReasoningState) bool
    AfterIteration(iteration int, text string, toolCalls []*protocol.ToolCall, results []ToolResult, state *ReasoningState) error
    GetContextInjection(state *ReasoningState) string
    GetPromptSlots() PromptSlots
    GetRequiredTools() []RequiredTool
    GetName() string
    GetDescription() string
}
```

**Usage**: Implement custom reasoning engines for domain-specific logic.

**See**: [Extend Hector](../docs/how-to/extend-hector.md) for complete examples.

---

### Memory Strategies

```go
import "github.com/kadirpekel/hector/pkg/memory"

// WorkingMemoryStrategy interface for custom working memory
type WorkingMemoryStrategy interface {
    AddMessage(session *hectorcontext.ConversationHistory, message *pb.Message) error
    CheckAndSummarize(session *hectorcontext.ConversationHistory) ([]*pb.Message, error)
    GetMessages(session *hectorcontext.ConversationHistory) ([]*pb.Message, error)
    SetStatusNotifier(notifier StatusNotifier)
    Name() string
    LoadState(sessionID string, sessionService interface{}) (*hectorcontext.ConversationHistory, error)
}

// LongTermMemoryStrategy interface for custom long-term memory
type LongTermMemoryStrategy interface {
    Store(agentID string, sessionID string, messages []*pb.Message) error
    Recall(agentID string, sessionID string, query string, limit int) ([]*pb.Message, error)
    Clear(agentID string, sessionID string) error
    Name() string
}
```

**Usage**: Implement custom memory strategies for specialized memory management.

**See**: [Extend Hector](../docs/how-to/extend-hector.md) for complete examples.

---

### Observability

```go
import "github.com/kadirpekel/hector/pkg/observability"

// Observability manager
type Manager struct { ... }
func NewManager(cfg Config) *Manager
func (m *Manager) Initialize(ctx context.Context) error
func (m *Manager) GetTracer(name string) trace.Tracer
func (m *Manager) GetMetrics() Metrics
func (m *Manager) Shutdown(ctx context.Context) error
```

**Usage**: Configure observability programmatically.

**Recommended**: Use `hector.NewObservability().Build()` instead.

**Example**:
```go
obsConfig, _ := hector.NewObservability().
    EnableMetrics(true).
    WithTracing(hector.NewTracing().
        Enable(true).
        EndpointURL("http://jaeger:4317").
        SamplingRate(0.1).
        ServiceName("my-app")).
    Build()

obsMgr := observability.NewManager(obsConfig)
obsMgr.Initialize(context.Background())
```

---

### Runtime

```go
import "github.com/kadirpekel/hector/pkg/runtime"

// Runtime builder (programmatic API)
func NewRuntimeBuilder() *RuntimeBuilder
func (b *RuntimeBuilder) WithAgent(agent *agent.Agent) *RuntimeBuilder
func (b *RuntimeBuilder) WithAgents(agents map[string]*agent.Agent) *RuntimeBuilder
func (b *RuntimeBuilder) Start() (*Runtime, error)

// Runtime from config (uses programmatic API internally)
func NewWithConfig(cfg *config.Config) (*Runtime, error)
```

**Usage**: Create runtime with agents built programmatically or from config.

**Example**:
```go
rt, err := runtime.NewRuntimeBuilder().
    WithAgent(agent).
    Start()
```

---

## Extension Points

### Custom Components

You can extend Hector by implementing these interfaces:

- **`reasoning.ReasoningStrategy`** - Custom reasoning engines
- **`tools.Tool`** - Custom tools
- **`memory.WorkingMemoryStrategy`** - Custom working memory
- **`memory.LongTermMemoryStrategy`** - Custom long-term memory
- **`llms.LLMProvider`** - Custom LLM providers (advanced)
- **`databases.DatabaseProvider`** - Custom database providers (advanced)
- **`embedders.EmbedderProvider`** - Custom embedders (advanced)

**See**: [Extend Hector](../docs/how-to/extend-hector.md) for complete examples.

---

## Not Part of Public API

### Internal Implementations

- `pkg/agent/services.go` - Internal service orchestration
- `pkg/transport/*` - Internal transport layer (unless explicitly documented)
- `pkg/cli/*` - CLI-specific code
- `pkg/*/internal/*` - All internal packages

### Use Programmatic API Instead

Rather than calling internal functions directly, use the programmatic API:

```go
// ✅ Recommended: Programmatic API
agent, err := hector.NewAgent("agent").
    WithLLMProvider(llm).
    Build()

// ❌ Not recommended: Direct internal calls
// Most internal constructors are not designed for external use
```

---

## Stability Guarantees

- ✅ **Stable**: Programmatic API builders (`hector.NewAgent()`, `hector.NewLLMProvider()`, etc.)
- ✅ **Stable**: Extension interfaces (`reasoning.ReasoningStrategy`, `tools.Tool`, etc.)
- ✅ **Stable**: Low-level provider constructors (`llms.NewOpenAIProviderFromConfig`, etc.)
- 🚧 **Experimental**: Plugin extractors, custom processors
- ⚠️ **Internal**: Anything in `pkg/*/internal/`, unexported functions

---

## Migration Guide

If you're using Hector programmatically:

1. **Use Programmatic API** - Prefer `hector.NewAgent()` over direct constructors
2. **Use Builders** - Prefer `hector.NewLLMProvider()` over `llms.NewOpenAIProviderFromConfig()`
3. **Extend via Interfaces** - Implement `ReasoningStrategy`, `Tool`, etc. for custom components
4. **See Documentation** - Check [Programmatic API Reference](../docs/reference/programmatic-api.md) for complete API

---

## Documentation

- **[Programmatic API Guide](../docs/core-concepts/programmatic-api.md)** - Core concepts and patterns
- **[Programmatic API Reference](../docs/reference/programmatic-api.md)** - Complete API documentation
- **[Extend Hector](../docs/how-to/extend-hector.md)** - How to create custom components
- **[Configuration Reference](../docs/reference/configuration.md)** - YAML configuration (alternative to programmatic API)

---

**Version**: 2.0.0  
**Compatibility**: Semantic versioning - major version bump for breaking API changes  
**Last Updated**: 2025-11-11
