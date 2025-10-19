---
title: Architecture
description: Complete architectural guide for Hector AI Agent Platform
---

# Hector Architecture

**100% A2A Protocol Native - Multi-Transport - Production-Ready**

---


### Key Features

- **100% Protobuf-Based** - All message types use `pb.*` (protobuf) directly
- **Multi-Transport** - Single codebase, three protocols (gRPC, REST, JSON-RPC)
- **Spec-Compliant** - Fully compliant with [A2A Protocol Specification](https://a2a-protocol.org/latest/specification/)
- **Discovery** - RFC 8615 `.well-known` endpoints for agent discovery
- **Authentication** - JWT-based security with configurable schemes
- **Streaming** - Real-time response streaming via gRPC streams and SSE
- **Task Management** - Async task processing with status tracking

---

## Agent Architecture

**How Hector agents work under the hood:**

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
│  │  Port: 8080  │  Port: 8081      │  Port: 8082         │   │
│  └──────────────┴──────────────────┴─────────────────────┘   │
├──────────────────────────────────────────────────────────────┤
│                       AGENT LAYER                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  Agent (pb.A2AServiceServer interface)                  │ │
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

## A2A Protocol Native Architecture

Hector is built **entirely** around the A2A Protocol, with zero abstraction layers between the protocol and implementation.

### Protocol-First Design

```
┌─────────────────────────────────────────────────────────────┐
│                    A2A Protocol Layer                       │
│  ┌─────────────────┬─────────────────┬─────────────────┐    │
│  │ Agent-to-Agent  │ Protobuf        │ A2A             │    │
│  │ Protocol        │ Messages        │ Specification   │    │
│  └─────────┬───────┴─────────┬───────┴─────────┬───────┘    │
│            │                 │                 │            │
│            ▼                 ▼                 ▼            │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                Hector Implementation                   │ │
│  │  ┌─────────────┬─────────────┬─────────────┐           │ │
│  │  │ Transport   │ Runtime     │ Agent       │           │ │
│  │  │ Layer       │ System      │ Management  │           │ │
│  │  └─────────────┴─────────────┴─────────────┘           │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```
### Core A2A Components

| Component | Purpose | Implementation |
|-----------|---------|----------------|
| **Agent Registry** | Service discovery | `.well-known/a2a/agents` |
| **Message Protocol** | Agent communication | Protobuf messages |
| **Task Management** | Async processing | A2A task lifecycle |
| **Streaming** | Real-time responses | gRPC streams + SSE |
| **Authentication** | Security | JWT tokens |

### Protocol Benefits

- **Performance** - Direct protobuf serialization
- **Reliability** - Spec-compliant implementation
- **Interoperability** - Works with any A2A-compliant system
- **Future-Proof** - Protocol evolution support

---

## Distributed Multi-Agent Platform

**Enterprise-grade agent networks and orchestration:**

```
┌─────────────────────────────────────────────────────────────┐
│                        USER / CLIENT                        │
│                  (CLI, HTTP, A2A Protocol)                  │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          │ A2A Protocol (HTTP+JSON/SSE)
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                      A2A SERVER                             │
│         • Discovery (/agents)    • Execution (/tasks)       │
│         • Sessions               • Streaming (SSE)          │
└─────────────────────────┬───────────────────────────────────┘
                          │
      ┌───────────────────┼───────────────────┐
      │                   │                   │
      ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐   ┌──────────────┐
│Orchestrator  │    │   Native     │   │   External   │
│    Agent     │    │   Agents     │   │  A2A Agents  │
│              │    │              │   │              │
│ • Supervisor │    │ • Local      │   │ • Remote URL │
│ • agent_call │    │ • Full Ctrl  │   │ • HTTP Proxy │
│ • Synthesis  │    │              │   │ • Same Iface │
└──────┬───────┘    └──────────────┘   └──────────────┘
       │
       │ LLM-Driven Routing (agent_call tool)
       └──────────────────┐
                          ▼
                  ┌───────────────┐
                  │ Agent Registry│
                  │  (All Agents) │
                  └───────────────┘
```

### Production Examples

**Deploy Multi-Agent Server:**
```bash
hector serve --config agents.yaml --port 8080
```

**Connect from Anywhere:**
```bash
hector call coordinator "Research AI trends" --server https://agents.company.com:8080
```

**API Integration:**
```bash
curl -X POST https://agents.company.com:8080/v1/agents/coordinator/message:send \
  -H "Content-Type: application/json" \
  -d '{"message": "Research AI trends"}'
```

**Hybrid Agent Networks:**
```yaml
agents:
  local_assistant:
    type: "native"
    llm: "gpt-4o"
  remote_analyst:
    type: "a2a"
    url: "https://analytics.partner.com/a2a"
```

---

## Transport Layer

Hector supports multiple transport protocols from a single codebase, as shown in the Agent Architecture diagram above. The transport layer provides three distinct interfaces:

### Transport Options

=== "gRPC (Core)"
    ```bash
    # Direct gRPC calls - Native A2A protocol
    grpcurl -plaintext \
      -d '{"agent_id":"my_agent","message":{"role":"user","parts":[{"type":"text","text":"Hello"}]}}' \
      localhost:8080 \
      a2a.AgentService/SendMessage
    ```
    - **Port**: 8080
    - **Protocol**: Binary protobuf
    - **Features**: Native streaming, high performance

=== "REST (Gateway)"
    ```bash
    # HTTP REST API - Auto-generated from gRPC
    curl -X POST http://localhost:8081/agents/my_agent/message/send \
      -H "Content-Type: application/json" \
      -d '{"message":{"role":"user","parts":[{"type":"text","text":"Hello"}]}}'
    ```
    - **Port**: 8081
    - **Protocol**: JSON over HTTP
    - **Features**: Web compatibility, SSE streaming

=== "JSON-RPC (Adapter)"
    ```bash
    # JSON-RPC calls - Legacy system support
    curl -X POST http://localhost:8082/rpc \
      -H "Content-Type: application/json" \
      -d '{"jsonrpc":"2.0","method":"agent.send_message","params":{"agent_id":"my_agent","message":{"role":"user","parts":[{"type":"text","text":"Hello"}]}},"id":1}'
    ```
    - **Port**: 8082
    - **Protocol**: JSON-RPC 2.0
    - **Features**: Legacy compatibility, simple RPC

---

## Client Architecture

Hector's client architecture supports multiple deployment patterns:

### Client Modes

=== "Local Mode"
    ```yaml
    # Local execution
    agents:
      my_agent:
        llm: "gpt-4o"
        # Runs in-process
    ```
    
    **Use cases:**
- Development and testing
- CI/CD pipelines
- Local automation

=== "Client Mode"
    ```bash
    # Connect to remote server
    hector call my_agent "Hello" --server https://api.hector.dev
    ```
    
    **Use cases:**
- Production deployments
- Team collaboration
- Managed services

=== "Server Mode"
    ```bash
    # Host agents as service
    hector serve --port 8080
    ```
    
    **Use cases:**
- Production hosting
- Multi-user access
- API services

### Client Features

- **Auto-Detection** - Automatically selects mode based on configuration
- **Security** - JWT authentication for remote connections
- **Streaming** - Real-time response streaming
- **Session Management** - Persistent conversation context

---

## Server Architecture

Hector's server provides a robust, scalable platform for hosting AI agents:

### Runtime Features

- **Component Management** - Dynamic loading and configuration
- **Agent Factory** - Creates agents from configuration
- **Service Registry** - Manages LLMs, databases, tools
- **Memory Management** - Session and long-term memory
- **Reasoning Engine** - Chain-of-thought and supervisor strategies

### Server Components

| Component | Purpose | Features |
|-----------|---------|----------|
| **Runtime Manager** | Core orchestration | Component lifecycle, configuration |
| **Agent Factory** | Agent creation | Dynamic loading, validation |
| **Service Registry** | Resource management | LLMs, databases, tools |
| **Memory Service** | Context management | Session, long-term, vector storage |
| **Task Service** | Async processing | Status tracking, streaming |

---

## Multi-Agent Orchestration

Hector supports sophisticated multi-agent workflows through the supervisor reasoning engine:

### Orchestration Patterns

=== "Sequential Processing"
    ```yaml
    agents:
      researcher:
        reasoning:
          engine: "supervisor"
        tools:
          - "agent_call"
    
    # Agent calls another agent
    Tool: agent_call(agent_id="analyzer", message="Analyze this data")
    ```

=== "Parallel Processing"
    ```yaml
    agents:
      coordinator:
        reasoning:
          engine: "supervisor"
        prompt:
          reasoning_instructions: |
            1. Break task into parallel subtasks
            2. Call multiple agents simultaneously
            3. Synthesize results
    ```

=== "Hierarchical Processing"
    ```yaml
    agents:
      manager:
        reasoning:
          engine: "supervisor"
        tools:
          - "agent_call"
      
      worker1:
        # Specialized agent
      
      worker2:
        # Another specialized agent
    ```

### Orchestration Features

- **Agent Communication** - Agents can call other agents via `agent_call` tool
- **Task Decomposition** - Break complex tasks into subtasks
- **Result Synthesis** - Combine results from multiple agents
- **Async Processing** - Non-blocking agent communication
- **LLM-Driven Routing** - Intelligent task delegation based on agent capabilities

---

## Security & Authentication

Hector provides comprehensive security features:

```
### Security Features

- **JWT Authentication** - Industry-standard token-based auth
- **Token Validation** - Secure token verification
- **Access Control** - Agent-level permissions
- **Transport Security** - TLS/SSL support
- **Audit Logging** - Security event tracking

### Configuration

```yaml
# Authentication configuration
auth:
  jwt:
    secret: "${JWT_SECRET}"
    expires_in: "24h"
    issuer: "hector"
  
  # Agent access control
  agents:
    public_agent:
      visibility: "public"
    private_agent:
      visibility: "private"
      required_scopes: ["admin"]
```

---

## Extension Points

Hector's plugin system allows extensive customization:

### Plugin Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Plugin Types                             │
│  ┌─────────────────┬─────────────────┬──────────────────┐   │
│  │ **LLM Provider**│ **Tool Provider**│ **DB Provider** │   │
│  │ Custom models   │ Custom capabilities│ Vector storage│   │
│  │ Text generation │ Function execution│ Search & recall│   │
│  └─────────────────┴─────────────────┴──────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Plugin Features

- **Language Agnostic** - Write in any gRPC-supported language
- **Process Isolation** - Plugins run in separate processes
- **Hot Reload** - Update plugins without restart
- **High Performance** - gRPC-based communication
