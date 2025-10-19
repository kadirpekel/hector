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

## Transport Layer

Hector supports multiple transport protocols from a single codebase:

### Transport Options

=== "gRPC (Native)"
    ```bash
    # Direct gRPC calls
    grpcurl -plaintext \
      -d '{"agent_id":"my_agent","message":{"role":"user","parts":[{"type":"text","text":"Hello"}]}}' \
      localhost:8080 \
      a2a.AgentService/SendMessage
    ```

=== "REST (Gateway)"
    ```bash
    # HTTP REST API
    curl -X POST http://localhost:8080/agents/my_agent/message/send \
      -H "Content-Type: application/json" \
      -d '{"message":{"role":"user","parts":[{"type":"text","text":"Hello"}]}}'
    ```

=== "JSON-RPC"
    ```bash
    # JSON-RPC calls
    curl -X POST http://localhost:8080/rpc \
      -H "Content-Type: application/json" \
      -d '{"jsonrpc":"2.0","method":"agent.send_message","params":{"agent_id":"my_agent","message":{"role":"user","parts":[{"type":"text","text":"Hello"}]}},"id":1}'
    ```

### Transport Features

- **High Performance** - gRPC for internal communication
- **Web Compatibility** - REST for web applications
- **Legacy Support** - JSON-RPC for existing systems
- **Security** - Authentication across all transports

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

```
### Runtime Features

- **Component Management** - Dynamic loading and configuration
- **Agent Factory** - Creates agents from configuration
- **Service Registry** - Manages LLMs, databases, tools
- **Memory Management** - Session and long-term memory
- **Reasoning Engine** - Chain-of-thought and supervisor strategies

---

## Core Components

Hector's core components provide the foundation for AI agent execution:

### Component Architecture

| Component | Purpose | Features |
|-----------|---------|----------|
| **Agent Factory** | Creates agents from config | Dynamic loading, validation |

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

- **Agent Communication** - Agents can call other agents
- **Task Decomposition** - Break complex tasks into subtasks
- **Result Synthesis** - Combine results from multiple agents
- **Async Processing** - Non-blocking agent communication

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
│  ┌─────────────────┬─────────────────┬─────────────────┐    │
│  │ **LLM Provider**│ **Tool Provider**│ **DB Provider** │    │
│  │ Custom models   │ Custom capabilities│ Vector storage │    │
│  │ Text generation │ Function execution│ Search & recall │    │
│  └─────────────────┴─────────────────┴─────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```### Plugin Features

- **Language Agnostic** - Write in any gRPC-supported language
- **Process Isolation** - Plugins run in separate processes
- **Hot Reload** - Update plugins without restart
- **High Performance** - gRPC-based communication
