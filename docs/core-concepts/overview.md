---
title: What is an Agent?
description: Understanding Hector agents and how they work
---

# What is an Agent?

An **agent** in Hector is an AI-powered entity that can understand requests, reason about them, use tools to accomplish tasks, and provide intelligent responses. Unlike simple chatbots, Hector agents are declaratively configured, tool-enabled, and designed for interoperability.

## Key Concepts

### Declarative Configuration

Agents are defined entirely in YAML—no code required:

```yaml
agents:
  assistant:
    name: "My Assistant"
    llm: "gpt-4o"
    prompt:
      system_role: "You are a helpful assistant."
    tools: ["write_file", "search"]
```

That's it. Hector handles the rest.

### Agent-to-Agent (A2A) Protocol

Every Hector agent implements the [A2A protocol v0.3.0](https://a2a-protocol.org), enabling:

- **Interoperability** - Agents can call other agents across networks
- **Standards-based** - 100% compatible with any A2A-compliant service
- **Enhanced REST** - Dual-path support for maximum compatibility
- **Distributed systems** - Build agent networks spanning multiple servers

### Agent Lifecycle

When you send a message to an agent:

1. **Request arrives** via REST, gRPC, WebSocket, or CLI
2. **Memory loads** - Agent retrieves conversation history
3. **Reasoning begins** - LLM processes the request
4. **Tools execute** - Agent performs actions if needed
5. **Response returns** - Result sent back to client
6. **History saves** - Conversation stored for context

## Agent Components

Every agent consists of these core components:

### 1. LLM Provider

The language model that powers the agent's intelligence:

```yaml
agents:
  assistant:
    llm: "gpt-4o"  # References an LLM configuration

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

See [LLM Providers](llm-providers.md) for details.

### 2. Prompt

Defines the agent's personality, instructions, and behavior:

```yaml
agents:
  assistant:
    prompt:
      system_role: |
        You are an expert software engineer.
        Provide clear, concise answers.
```

See [Prompts](prompts.md) for details.

### 3. Memory

Manages conversation history and context:

```yaml
agents:
  assistant:
    memory:
      working:
        strategy: "buffer_window"
        window_size: 10
      longterm:
        
        storage_scope: "session"
```

See [Memory](memory.md) for details.

### 4. Tools

Capabilities the agent can use:

```yaml
agents:
  assistant:
    tools:
      - "write_file"
      - "execute_command"
      - "search"
      - "agent_call"
```

See [Tools](tools.md) for details.

### 5. Reasoning Strategy

How the agent thinks and makes decisions:

```yaml
agents:
  coordinator:
    reasoning:
      strategy: "supervisor"  # For multi-agent orchestration
```

See [Reasoning Strategies](reasoning.md) for details.

## Agent Types

### Native Agents

Agents defined in your Hector configuration:

```yaml
agents:
  assistant:
    # Full agent configuration
```

These run locally in your Hector instance.

### External A2A Agents

Remote agents accessed via the A2A protocol:

```yaml
agents:
  external_specialist:
    type: "a2a"
    url: "https://external-agent.example.com"
    credentials:
      type: "bearer"
      token: "${EXTERNAL_TOKEN}"
```

These run on other servers but can be called like local agents.

See [Multi-Agent Orchestration](multi-agent.md) for details.

## Agent Modes

Hector agents can run in different modes:

### Local Mode

Agent runs in-process for a single command:

```bash
hector call "Hello" --agent assistant --config config.yaml
```

- **Use when:** Quick experiments, CI/CD, scripting
- **Pros:** Simple, no server needed
- **Cons:** No persistence between calls

### Server Mode

Agents hosted as a persistent service:

```bash
hector serve --config config.yaml
```

- **Use when:** Production deployments, multiple clients
- **Pros:** Persistent, scalable, multi-user
- **Cons:** Requires server infrastructure

### Client Mode

Connect to a remote Hector server:

```bash
hector call "Hello" --agent assistant --url http://remote:8080
```

- **Use when:** Distributed systems, shared agents
- **Pros:** Centralized management, resource sharing
- **Cons:** Network dependency

See [CLI Reference](../reference/cli.md) for mode details.

## Zero-Config Mode

Hector can create a default agent automatically:

```bash
export OPENAI_API_KEY="sk-..."
hector call "Hello"  # No config file needed!
```

Behind the scenes, Hector creates:
- Default agent named "assistant"
- OpenAI GPT-4o mini LLM
- Basic prompt
- No tools enabled

Perfect for quick experiments and prototyping.

## Configuration Structure

A complete agent configuration:

```yaml
# LLM providers
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7

# Agent definition
agents:
  assistant:
    name: "My Assistant"
    llm: "gpt-4o"
    
    prompt:
      system_role: |
        You are a helpful assistant.
    
    tools:
      - "write_file"
      - "search"
    
    memory:
      working:
        strategy: "buffer_window"
        window_size: 10
    
    reasoning:
      strategy: "chain_of_thought"
```

See [Configuration Reference](../reference/configuration.md) for all options.

## What Makes Hector Agents Different?

| Feature | Hector | Traditional Frameworks |
|---------|--------|----------------------|
| **Configuration** | Pure YAML | Code + config |
| **Interoperability** | A2A native | Custom APIs |
| **Memory** | Built-in strategies | Manual implementation |
| **Multi-agent** | Native supervisor | Custom orchestration |
| **Tools** | Built-in, MCP, plugins | Manual integration |
| **Deployment** | Three modes | Server only |

## Next Steps

Learn about the core components of agents:

- **[LLM Providers](llm-providers.md)** - Connect to OpenAI, Anthropic, or Gemini
- **[Prompts](prompts.md)** - Customize agent behavior and personality
- **[Memory](memory.md)** - Manage conversation history and context
- **[Tools](tools.md)** - Give agents capabilities
- **[RAG & Semantic Search](rag.md)** - Long-term memory with vector stores
- **[Reasoning Strategies](reasoning.md)** - How agents think and decide
- **[Multi-Agent Orchestration](multi-agent.md)** - Coordinate multiple agents
- **[Human-in-the-Loop](human-in-the-loop.md)** - Tool approval and interactive workflows
- **[AG-UI Protocol](ag-ui.md)** - Standardized streaming event format for UIs
- **[A2A Protocol](../reference/a2a-protocol.md)** - Agent-to-Agent communication standard
- **[Sessions](sessions.md)** - Conversation persistence and management
- **[Streaming](streaming.md)** - Real-time response streaming
- **[Tasks](tasks.md)** - Task lifecycle and management
- **[Structured Output](structured-output.md)** - Schema-validated responses
- **[Observability](observability.md)** - Metrics, tracing, and monitoring
- **[Security](security.md)** - Authentication, authorization, and sandboxing
- **[Performance](performance.md)** - Optimization and scaling

## Related Topics

- **[Quick Start](../getting-started/quick-start.md)** - Run your first agent
- **[Build a Coding Assistant](../how-to/build-coding-assistant.md)** - Complete tutorial
- **[Configuration Reference](../reference/configuration.md)** - All configuration options

