---
title: Architecture
description: System architecture and design principles
---

# Architecture

Deep dive into Hector's architecture, components, and design principles.

---

## Overview

Hector is a **declarative AI agent platform** built on three core principles:

1. **A2A Protocol Native** - 100% compliant with A2A Protocol v0.3.0 specification (with enhanced dual-path REST support)
2. **Zero Code Required** - Build agents with YAML configuration
3. **Production Ready** - Enterprise-grade performance and reliability

!!! info "Looking for Performance & Scaling?"
    This document covers **system architecture** (how Hector is built).
    
    For **performance characteristics** (resource usage, scaling, efficiency), see:
    
    - [Performance Overview](../core-concepts/performance.md)
    
    For **technical deep dives** on architecture:
    
    - [Session & Memory Architecture](architecture/session-memory.md)
    - [Agent Lifecycle & Scalability](architecture/agent-lifecycle.md)

---

## System Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      APPLICATION                             │
│                 (Your Agent & Logic)                         │
├──────────────────────────────────────────────────────────────┤
│                     HECTOR RUNTIME                           │
│  • Configuration Loading  • Agent Initialization             │
│  • Component Management   • Lifecycle Management             │
├──────────────────────────────────────────────────────────────┤
│                      CLIENT LAYER                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │         A2AClient Interface (Protocol Native)           │ │
│  ├─────────────────────────────────────────────────────────┤ │
│  │  HTTPClient           │          LocalClient            │ │
│  │  • Remote agents      │          • In-process agents    │ │
│  │  • Uses protojson     │          • No network calls     │ │
│  │  • Multi-transport    │          • Direct protobuf      │ │
│  └─────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────┤
│                      TRANSPORT LAYER                         │
│  ┌──────────────┬──────────────────┬─────────────────────┐   │
│  │  gRPC (Core) │  REST (Gateway)  │  JSON-RPC (Adapter) │   │
│  │  • Native    │  • Auto-gen      │  • Custom HTTP      │   │
│  │  • Binary    │  • JSON          │  • Simple RPC       │   │
│  │  • Streaming │  • SSE           │  • JSON             │   │
│  │  Port: 8080  │  Port: 8080      │  Port: 8080         │   │
│  └──────────────┴──────────────────┴─────────────────────┘   │
├──────────────────────────────────────────────────────────────┤
│                       AGENT LAYER                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  Agent (A2A Interface)                                  │ │
│  │  • SendMessage          • GetAgentCard                  │ │
│  │  • SendStreamingMessage • GetTask/CancelTask            │ │
│  │  • Task subscriptions   • Push notifications            │ │
│  └─────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────┤
│                    REASONING ENGINE                          │
│  Chain-of-Thought Strategy    |    Supervisor Strategy       │
│  • Step-by-step reasoning     |    • Multi-agent coord       │
│  • Natural termination        |    • Task decomposition      │
├──────────────────────────────────────────────────────────────┤
│                        CORE SERVICES                         │
│  ┌───────────┬──────────┬──────────┬──────────┬──────────┐   │
│  │    LLM    │   Tools  │   Memory │    RAG   │   Tasks  │   │
│  │  • OpenAI │ • Local  │ • Buffer │ • Qdrant │ • Async  │   │
│  │• Anthropic│ • MCP    │ • Summary│ • Search │ • Status │   │
│  │  • Gemini │ • Plugin │ • Session│ • Embed  │ • Track  │   │
│  └───────────┴──────────┴──────────┴──────────┴──────────┘   │
└──────────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. Runtime

The runtime manages agent lifecycle and component initialization.

**Responsibilities:**
- Load and validate configuration
- Initialize components (LLMs, databases, embedders)
- Create and register agents
- Handle graceful shutdown

**Key Files:**
- `pkg/runtime/runtime.go`
- `pkg/hector.go`

### 2. Agent

The core agent implementation that handles message processing.

**Responsibilities:**
- Receive and process messages
- Execute reasoning strategy
- Manage conversation context
- Return responses

**Key Files:**
- `pkg/agent/agent.go`
- `pkg/agent/orchestration.go`

### 3. Reasoning Engine

Strategies for agent reasoning and decision-making.

**Chain-of-Thought:**
- Step-by-step reasoning
- Tool execution loop
- Natural termination

**Supervisor:**
- Multi-agent orchestration
- Task decomposition
- Agent selection
- Result synthesis

**Key Files:**
- `pkg/reasoning/chain_of_thought.go`
- `pkg/reasoning/supervisor_strategy.go`

### 4. Memory System

Dual-layer memory for short and long-term context.

**Working Memory:**
- Buffer Window (LIFO)
- Summary Buffer (token-based compression)

**Long-Term Memory:**
- Vector-based storage
- Semantic search
- Session scoping

**Key Files:**
- `pkg/memory/memory.go`
- `pkg/memory/vector_memory.go`

### 5. Tool System

Three-tier tool architecture.

**Built-in Tools:**
- File operations (read, write, search_replace)
- Command execution
- Document search
- Todo management
- Agent communication

**MCP Integration:**
- External tool servers
- 150+ integrations (Composio, Mem0)
- Standard protocol

**gRPC Plugins:**
- Custom high-performance tools
- Native Go interface
- Production-grade

**Key Files:**
- `pkg/tools/*.go`
- `pkg/tools/mcp.go`

### 6. LLM Providers

Unified interface for multiple LLM providers.

**Supported:**
- OpenAI (GPT-4o, GPT-4o-mini)
- Anthropic (Claude)
- Google (Gemini)
- Custom plugins

**Key Files:**
- `pkg/llms/*.go`
- `pkg/llms/registry.go`

### 7. RAG & Document Stores

Semantic search and retrieval augmented generation.

**Components:**
- Document parsers (text, markdown, code)
- Vector databases (Qdrant)
- Embedders (Ollama)
- Incremental indexing

**Key Files:**
- `pkg/context/document_store.go`
- `pkg/context/search.go`

### 8. Transport Layer

Multiple protocol support from single codebase.

**Protocols:**
- gRPC (binary, HTTP/2)
- REST (JSON, HTTP/1.1)
- JSON-RPC (simple integration)
- WebSocket (real-time)

**Key Files:**
- `pkg/transport/*.go`
- `pkg/a2a/server/*.go`

---

## Data Flow

### Message Processing

```
1. Client → Transport Layer
   └─ REST/gRPC/JSON-RPC request

2. Transport → Agent
   └─ Parse and route to agent

3. Agent → Reasoning Engine
   └─ Execute reasoning strategy

4. Reasoning → LLM + Tools
   └─ Generate response, execute tools

5. Agent → Memory
   └─ Store conversation context

6. Agent → Transport
   └─ Return response

7. Transport → Client
   └─ Stream or return complete response
```

### Multi-Agent Orchestration

```
1. Client → Supervisor Agent
   └─ High-level task

2. Supervisor → LLM
   └─ Analyze task, create plan

3. Supervisor → Sub-Agents
   └─ Delegate via agent_call tool

4. Sub-Agents → Execution
   └─ Process individual tasks

5. Sub-Agents → Supervisor
   └─ Return results

6. Supervisor → LLM
   └─ Synthesize final response

7. Supervisor → Client
   └─ Return complete result
```

---

## Plugin System

Hector supports gRPC plugins for extending functionality.

### Plugin Types

| Type | Purpose | Interface |
|------|---------|-----------|
| **LLM** | Custom language models | `LLMProvider` |
| **Database** | Vector databases | `Database` |
| **Embedder** | Text embeddings | `Embedder` |
| **Tool** | Custom tools | `Tool` |
| **Parser** | Document parsing | `DocumentParser` |
| **Reasoning** | Custom strategies | `ReasoningEngine` |

### Plugin Architecture

```
┌─────────────────────────────────────┐
│         Hector Core                 │
├─────────────────────────────────────┤
│   Plugin Interface (gRPC)           │
├─────────────────────────────────────┤
│   ┌──────────┐  ┌──────────┐       │
│   │ Plugin 1 │  │ Plugin 2 │       │
│   │ (Go)     │  │ (Python) │       │
│   └──────────┘  └──────────┘       │
└─────────────────────────────────────┘
```

**Key Files:**
- `pkg/plugins/*.go`
- `pkg/plugins/grpc/*.proto`

---

## Configuration System

YAML-based declarative configuration.

### Configuration Flow

```
1. Load YAML file
2. Parse and validate
3. Substitute environment variables
4. Create components
5. Register agents
6. Start server
```

### Configuration Validation

- Schema validation
- Reference checking (LLM, database, embedder)
- Value range validation
- Default application

**Key Files:**
- `pkg/config/*.go`
- `pkg/config/validation_test.go`

---

## Memory Architecture

### Working Memory

```
┌────────────────────────────────────┐
│      Conversation History          │
│  ┌──────────────────────────────┐  │
│  │  Recent Messages (Buffer)    │  │
│  │  • Last N messages           │  │
│  │  • Fixed size window         │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │  Compressed Summary          │  │
│  │  • Token budget management   │  │
│  │  • Automatic summarization   │  │
│  └──────────────────────────────┘  │
└────────────────────────────────────┘
```

### Long-Term Memory

```
┌────────────────────────────────────┐
│      Vector Database (Qdrant)      │
│  ┌──────────────────────────────┐  │
│  │  Session Memories            │  │
│  │  • Scoped to conversation    │  │
│  │  • Semantic search           │  │
│  │  • Auto-recall               │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │  User Memories               │  │
│  │  • Across sessions           │  │
│  │  • Persistent knowledge      │  │
│  └──────────────────────────────┘  │
└────────────────────────────────────┘
```

---

## Performance Characteristics

### Throughput

| Operation | Latency | Throughput |
|-----------|---------|------------|
| Message (no tools) | ~200ms | ~50 req/s |
| Message (with tools) | ~1-2s | ~10 req/s |
| Streaming | ~50ms TTFT | Real-time |
| RAG search | ~50-100ms | ~100 req/s |

### Scalability

**Horizontal Scaling:**
- Stateless agent processing
- Shared vector database
- Session storage backends (SQL, Redis)

**Vertical Scaling:**
- Concurrent request handling
- Connection pooling
- Efficient memory management

---

## Security Architecture

### Authentication

```
┌─────────────────────────────────────┐
│         Client Request              │
│    (with JWT token)                 │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│    JWT Validation Middleware        │
│  • Verify signature (JWKS)          │
│  • Check issuer & audience          │
│  • Validate expiration              │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│    Agent-Level Authorization        │
│  • Check required schemes           │
│  • Validate credentials             │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│         Agent Execution             │
└─────────────────────────────────────┘
```

**Key Files:**
- `pkg/auth/*.go`

---

## Extensibility

Hector provides multiple extension points:

### 1. Configuration Extension

```yaml
# Add custom LLM
llms:
  custom:
    type: "plugin:my-llm"
    # Your config
```

### 2. MCP Tools

```yaml
# Add external tools
tools:
  custom_tools:
    type: "mcp"
    server_url: "http://localhost:3000"
```

### 3. gRPC Plugins

```go
// Implement interface
type MyLLM struct {}

func (m *MyLLM) Generate(ctx context.Context, req *pb.GenerateRequest) (*pb.GenerateResponse, error) {
  // Your implementation
}
```

### 4. Custom Reasoning

```go
// Implement ReasoningEngine
type MyStrategy struct {}

func (s *MyStrategy) Execute(ctx context.Context, input *ReasoningInput) (*ReasoningOutput, error) {
  // Your strategy
}
```

---

## Design Principles

### 1. Declarative First

Configuration over code. Agents defined in YAML, no programming required.

### 2. Protocol Native

Built entirely on A2A protocol. No abstraction layers, maximum interoperability.

### 3. Production Ready

Enterprise features built-in: auth, streaming, tasks, monitoring.

### 4. Extensible

Multiple extension points: plugins, MCP, custom components.

### 5. Zero Dependencies

Single binary deployment. No external services required (except for chosen features).

---

## Deployment Modes

### Local Mode

```bash
hector call "Hello" --agent assistant --config config.yaml
```

- In-process execution
- No network overhead
- Fast startup

### Server Mode

```bash
hector serve --config config.yaml
```

- Multi-client support
- Persistent state
- Production ready

### Client Mode

```bash
hector call "Hello" --agent assistant --url http://remote:8080
```

- Connect to remote servers
- Distributed agents
- Team collaboration

---

## Next Steps

### System Reference
- **[Configuration Reference](configuration.md)** - Complete config options
- **[API Reference](api.md)** - REST/gRPC APIs
- **[A2A Protocol](a2a-protocol.md)** - Protocol details

### Performance & Deployment
- **[Performance Overview](../core-concepts/performance.md)** - Resource efficiency & scaling
- **[Configuration Reference](configuration.md)** - Production setup

---

## Related Topics

- **[Core Concepts](../core-concepts/overview.md)** - Understanding components
- **[Multi-Agent](../core-concepts/multi-agent.md)** - Orchestration architecture
- **[Security](../core-concepts/security.md)** - Authentication architecture

