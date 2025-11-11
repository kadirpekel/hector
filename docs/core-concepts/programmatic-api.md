---
title: Programmatic API
description: Build agents programmatically using Go code with chained builders
---

# Programmatic API

Hector provides a **pure programmatic API** for building agents directly in Go code. This API is the foundation of Hector—the configuration system is built on top of it, not the other way around.

## When to Use Programmatic API

Use the programmatic API when you need:

- **Custom Logic**: Dynamic agent creation based on runtime conditions
- **Integration**: Embedding agents into existing Go applications
- **Advanced Control**: Fine-grained control over agent construction
- **Testing**: Programmatic agent creation for unit tests
- **Library Development**: Building higher-level abstractions on top of Hector

For simple use cases, [YAML configuration](../reference/configuration.md) is often easier.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│         Pure Programmatic API (Foundation)                  │
│  Chained builders, no config concepts, direct construction │
│                                                              │
│  AgentBuilder → Agent → Runtime                             │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │ Used by
                            │
┌─────────────────────────────────────────────────────────────┐
│         Config System (Convenience Layer)                    │
│  YAML/JSON → Config → Uses Programmatic API → Agents        │
│                                                              │
│  config.Config → AgentBuilder → Agent → Runtime               │
└─────────────────────────────────────────────────────────────┘
```

**Key Principle**: The configuration system **uses** the programmatic API internally. All agent construction flows through the programmatic API.

---

## Quick Start

### Basic Agent

```go
package main

import (
    "github.com/kadirpekel/hector/pkg/hector"
    "github.com/kadirpekel/hector/pkg/tools"
)

func main() {
    // Build LLM provider
    llm, err := hector.NewLLMProvider("openai").
        Model("gpt-4o-mini").
        APIKeyFromEnv("OPENAI_API_KEY").
        Temperature(0.7).
        Build()
    if err != nil {
        panic(err)
    }

    // Build reasoning strategy
    reasoning, err := hector.NewReasoning("chain-of-thought").
        MaxIterations(100).
        EnableStreaming(true).
        Build()
    if err != nil {
        panic(err)
    }

    // Build working memory
    workingMemory, err := hector.NewWorkingMemory("summary_buffer").
        Budget(2000).
        Threshold(0.8).
        WithLLMProvider(llm).
        Build()
    if err != nil {
        panic(err)
    }

    // Build agent
    agent, err := hector.NewAgent("assistant").
        WithName("Assistant").
        WithDescription("A helpful AI assistant").
        WithLLMProvider(llm).
        WithReasoningStrategy(reasoning).
        WithWorkingMemory(workingMemory).
        WithSystemPrompt("You are a helpful assistant.").
        WithTools(
            tools.NewFileWriterTool(nil),
            tools.NewReadFileTool(nil),
        ).
        Build()
    if err != nil {
        panic(err)
    }

    fmt.Printf("Built agent: %s (%s)\n", agent.GetID(), agent.GetName())
}
```

---

## Core Concepts

### 1. Chained Builders

All builders use a fluent, chainable API:

```go
agent, err := hector.NewAgent("my-agent").
    WithName("My Agent").
    WithDescription("Does amazing things").
    WithLLMProvider(llm).
    WithReasoningStrategy(reasoning).
    Build()
```

### 2. Component Builders

Build components independently and reuse them:

```go
// Build once, use many times
llm, _ := hector.NewLLMProvider("openai").
    Model("gpt-4o").
    APIKeyFromEnv("OPENAI_API_KEY").
    Build()

// Use in multiple agents
agent1, _ := hector.NewAgent("agent1").WithLLMProvider(llm).Build()
agent2, _ := hector.NewAgent("agent2").WithLLMProvider(llm).Build()
```

### 3. Direct Construction

No configuration files needed—everything is constructed directly:

```go
// No YAML, no config files—pure Go code
agent, err := hector.NewAgent("custom").
    WithLLMProvider(llm).
    WithReasoningStrategy(reasoning).
    Build()
```

---

## Building Components

### LLM Providers

```go
// OpenAI
llm, err := hector.NewLLMProvider("openai").
    Model("gpt-4o").
    APIKey("sk-...").
    Temperature(0.7).
    MaxTokens(4000).
    Build()

// Anthropic
llm, err := hector.NewLLMProvider("anthropic").
    Model("claude-3-5-sonnet-20241022").
    APIKeyFromEnv("ANTHROPIC_API_KEY").
    Temperature(0.8).
    Build()

// Ollama (local)
llm, err := hector.NewLLMProvider("ollama").
    Model("llama2").
    Host("localhost:11434").
    Build()
```

### Reasoning Strategies

```go
// Chain-of-Thought
reasoning, err := hector.NewReasoning("chain-of-thought").
    MaxIterations(100).
    EnableStreaming(true).
    ShowTools(true).
    ShowThinking(true).
    Build()

// Supervisor (for multi-agent)
reasoning, err := hector.NewReasoning("supervisor").
    MaxIterations(10).
    EnableStreaming(true).
    Build()
```

### Memory Strategies

```go
// Summary Buffer
memory, err := hector.NewWorkingMemory("summary_buffer").
    Budget(2000).
    Threshold(0.8).
    Target(0.6).
    WithLLMProvider(llm).
    Build()

// Buffer Window
memory, err := hector.NewWorkingMemory("buffer_window").
    WindowSize(10).
    Build()

// Sliding Window
memory, err := hector.NewWorkingMemory("sliding_window").
    Budget(4000).
    Build()
```

### Long-Term Memory

```go
// Build database
db, err := hector.NewDatabase("qdrant").
    Host("localhost").
    Port(6333).
    APIKey("...").
    Build()

// Build embedder
embedder, err := hector.NewEmbedder("openai").
    Model("text-embedding-3-small").
    APIKeyFromEnv("OPENAI_API_KEY").
    Build()

// Build long-term memory
longTerm, config, err := hector.NewLongTermMemory().
    Enabled(true).
    Collection("agent_memory").
    StorageScope(memory.StorageScopeAll).
    BatchSize(10).
    AutoRecall(true).
    RecallLimit(5).
    WithDatabase(db).
    WithEmbedder(embedder).
    Build()
```

### Context Service (RAG)

```go
// Build context service for RAG
contextService, err := hector.NewContextService().
    WithDatabase(db).
    WithEmbedder(embedder).
    TopK(5).
    Threshold(0.7).
    IncludeContext(true).
    Build()
```

### Task Service

```go
// In-memory task service
taskService, err := hector.NewTaskService().
    Backend("memory").
    WorkerPool(10).
    InputTimeout(600).
    Build()

// SQL task service
taskService, err := hector.NewTaskService().
    Backend("sql").
    WithSQLConfig(&config.SQLConfig{
        Driver: "postgres",
        DSN:    "postgres://user:pass@localhost/db",
    }).
    WorkerPool(10).
    Build()
```

### Session Service

```go
// In-memory session service
sessionService, err := hector.NewSessionService().
    Backend("memory").
    Build()

// SQL session service
sessionService, err := hector.NewSessionService().
    Backend("sql").
    WithSQLConfig(&config.SQLConfig{
        Driver: "postgres",
        DSN:    "postgres://user:pass@localhost/db",
    }).
    Build()
```

---

## Building Agents

### Complete Agent Example

```go
agent, err := hector.NewAgent("research-assistant").
    WithName("Research Assistant").
    WithDescription("Helps with research tasks").
    
    // LLM
    WithLLMProvider(llm).
    
    // Reasoning
    WithReasoningStrategy(reasoning).
    
    // Memory
    WithWorkingMemory(workingMemory).
    WithLongTermMemory(longTermMemory, longTermConfig).
    
    // Context (RAG)
    WithContext(contextService).
    
    // Tasks
    WithTask(taskService).
    
    // Sessions
    WithSession(sessionService).
    
    // Prompts
    WithSystemPrompt("You are a research assistant.").
    WithPromptSlots(&reasoning.PromptSlots{
        SystemRole:   "You are an expert researcher.",
        Instructions: "Always cite sources.",
        UserGuidance: "Be thorough and accurate.",
    }).
    
    // Tools
    WithTools(
        tools.NewFileWriterTool(nil),
        tools.NewReadFileTool(nil),
        tools.NewSearchTool(nil),
    ).
    
    // A2A Configuration
    WithA2ACard(hector.NewA2ACardBuilder(nil).
        Version("1.0.0").
        InputModes([]string{"text/plain", "application/json"}).
        OutputModes([]string{"text/plain", "application/json"}).
        PreferredTransport("json-rpc").
        Build()).
    
    // Security
    WithSecurity(hector.NewSecurityBuilder(nil).
        JWKSURL("https://auth.example.com/.well-known/jwks.json").
        Issuer("https://auth.example.com").
        Audience("hector-api").
        Build()).
    
    Build()
```

---

## Combining Config and Programmatic

You can mix agents built from config and programmatic agents:

```go
// Load config
cfg, _ := config.LoadConfig(config.LoaderOptions{
    Path: "configs/agents.yaml",
})

// Build agents from config (uses programmatic API internally)
configBuilder, _ := hector.NewConfigAgentBuilder(cfg)
configAgents, _ := configBuilder.BuildAllAgents()

// Build agent programmatically
programmaticAgent, _ := hector.NewAgent("custom").
    WithLLMProvider(llm).
    WithReasoningStrategy(reasoning).
    Build()

// Combine in runtime
runtime, _ := runtime.NewRuntimeBuilder().
    WithAgents(configAgents).      // From config
    WithAgent(programmaticAgent).   // Programmatic
    Start()
```

---

## Runtime Builder

Build a runtime programmatically:

```go
runtime, err := runtime.NewRuntimeBuilder().
    WithAgent(agent1).
    WithAgent(agent2).
    WithAgents(map[string]*agent.Agent{
        "agent3": agent3,
        "agent4": agent4,
    }).
    Start()
```

---

## Best Practices

### 1. Reuse Components

Build components once and reuse:

```go
// Build shared components
llm, _ := hector.NewLLMProvider("openai").Model("gpt-4o").APIKeyFromEnv("OPENAI_API_KEY").Build()
reasoning, _ := hector.NewReasoning("chain-of-thought").Build()

// Reuse in multiple agents
agent1, _ := hector.NewAgent("agent1").WithLLMProvider(llm).WithReasoningStrategy(reasoning).Build()
agent2, _ := hector.NewAgent("agent2").WithLLMProvider(llm).WithReasoningStrategy(reasoning).Build()
```

### 2. Error Handling

Always check errors:

```go
llm, err := hector.NewLLMProvider("openai").Build()
if err != nil {
    return fmt.Errorf("failed to build LLM: %w", err)
}
```

### 3. Builder Pattern

Use the fluent builder pattern consistently:

```go
// Good: Chainable
agent, _ := hector.NewAgent("id").
    WithName("Name").
    WithLLMProvider(llm).
    Build()

// Avoid: Multiple assignments
builder := hector.NewAgent("id")
builder.WithName("Name")
builder.WithLLMProvider(llm)
agent, _ := builder.Build()
```

### 4. Component Lifecycle

Components are independent—build them in any order:

```go
// Order doesn't matter
llm, _ := hector.NewLLMProvider("openai").Build()
reasoning, _ := hector.NewReasoning("chain-of-thought").Build()
memory, _ := hector.NewWorkingMemory("summary_buffer").Build()

// Combine when building agent
agent, _ := hector.NewAgent("id").
    WithLLMProvider(llm).
    WithReasoningStrategy(reasoning).
    WithWorkingMemory(memory).
    Build()
```

---

## Next Steps

- [Programmatic API Reference](../reference/programmatic-api.md) - Complete API documentation
- [Configuration Guide](../reference/configuration.md) - YAML configuration alternative
- [Examples](https://github.com/kadirpekel/hector/tree/main/examples) - More code examples

