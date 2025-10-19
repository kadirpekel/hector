---
title: Architecture
description: Complete architectural guide for Hector AI Agent Platform
---

# Hector Architecture

**100% A2A Protocol Native • Multi-Transport • Production-Ready**

---

## Table of Contents

- [Overview](#overview)
- [A2A Protocol Native Architecture](#a2a-protocol-native-architecture)
- [Transport Layer](#transport-layer)
- [Client Architecture](#client-architecture)
- [Server Architecture](#server-architecture)
- [Runtime System](#runtime-system)
- [Core Components](#core-components)
- [Multi-Agent Orchestration](#multi-agent-orchestration)
- [Security & Authentication](#security-authentication)
- [Extension Points](#extension-points)

---

## Overview

Hector is a **100% A2A Protocol Native** AI agent platform built from the ground up with the [Agent-to-Agent (A2A) Protocol](https://a2a-protocol.org/) at its core.

### Design Principles

1. **Protocol Native** - Every component speaks A2A natively using protobuf types - zero abstraction layers
2. **Multi-Transport** - gRPC (native), REST (grpc-gateway), JSON-RPC (custom adapter)
3. **Clean Architecture** - Clear separation of concerns (transport, runtime, client, server, agents)
4. **Interface-Based** - Dependency injection and strategy patterns throughout
5. **Production-Ready** - Authentication, discovery, streaming, task management

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
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Hector Implementation                     │ │
│  │  ┌─────────────┬─────────────┬─────────────┐            │ │
│  │  │ Transport   │ Runtime     │ Agent       │            │ │
│  │  │ Layer       │ System      │ Management  │            │ │
│  │  └─────────────┴─────────────┴─────────────┘            │ │
│  └─────────────────────────────────────────────────────────┘ │
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

### Transport Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    A2A Protocol Layer                       │
│  ┌─────────────────┬─────────────────┬─────────────────┐    │
│  │ Agent-to-Agent  │ Protobuf        │ A2A             │    │
│  │ Protocol        │ Messages        │ Specification   │    │
│  └─────────┬───────┴─────────┬───────┴─────────┬───────┘    │
│            │                 │                 │            │
│            ▼                 ▼                 ▼            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Hector Implementation                     │ │
│  │  ┌─────────────┬─────────────┬─────────────┐            │ │
│  │  │ Transport   │ Runtime     │ Agent       │            │ │
│  │  │ Layer       │ System      │ Management  │            │ │
│  │  └─────────────┴─────────────┴─────────────┘            │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```
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

### Server Components

```
┌─────────────────────────────────────────────────────────────┐
│                    A2A Protocol Layer                       │
│  ┌─────────────────┬─────────────────┬─────────────────┐    │
│  │ Agent-to-Agent  │ Protobuf        │ A2A             │    │
│  │ Protocol        │ Messages        │ Specification   │    │
│  └─────────┬───────┴─────────┬───────┴─────────┬───────┘    │
│            │                 │                 │            │
│            ▼                 ▼                 ▼            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Hector Implementation                     │ │
│  │  ┌─────────────┬─────────────┬─────────────┐            │ │
│  │  │ Transport   │ Runtime     │ Agent       │            │ │
│  │  │ Layer       │ System      │ Management  │            │ │
│  │  └─────────────┴─────────────┴─────────────┘            │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```
### Server Features

- **Multi-Transport** - gRPC, REST, JSON-RPC from single server
- **Authentication** - JWT-based security
- **Agent Registry** - Dynamic agent discovery and management
- **Streaming** - Real-time response streaming
- **Task Management** - Async task processing
- **Hot Reload** - Configuration updates without restart

---

## Runtime System

Hector's runtime system manages the execution of AI agents:

### Runtime Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    A2A Protocol Layer                       │
│  ┌─────────────────┬─────────────────┬─────────────────┐    │
│  │ Agent-to-Agent  │ Protobuf        │ A2A             │    │
│  │ Protocol        │ Messages        │ Specification   │    │
│  └─────────┬───────┴─────────┬───────┴─────────┬───────┘    │
│            │                 │                 │            │
│            ▼                 ▼                 ▼            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Hector Implementation                     │ │
│  │  ┌─────────────┬─────────────┬─────────────┐            │ │
│  │  │ Transport   │ Runtime     │ Agent       │            │ │
│  │  │ Layer       │ System      │ Management  │            │ │
│  │  └─────────────┴─────────────┴─────────────┘            │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
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
| **LLM Registry** | Manages language models | Multi-provider support |
| **Memory Manager** | Handles conversation context | Session + long-term memory |
| **Reasoning Engine** | Controls agent thinking | Chain-of-thought, supervisor |
| **Tool Registry** | Manages agent capabilities | Built-in + custom tools |
| **Document Store** | RAG and knowledge base | Vector search, embeddings |

### Component Features

- **Interface-Based** - Clean abstractions and dependency injection
- **Hot Reload** - Update components without restart
- **Validation** - Configuration validation and error handling
- **Performance** - Optimized for production workloads

---

## Multi-Agent Orchestration

Hector supports sophisticated multi-agent workflows:

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

- **Agent Communication** - Agents can call other agents
- **Task Decomposition** - Break complex tasks into subtasks
- **Result Synthesis** - Combine results from multiple agents
- **Async Processing** - Non-blocking agent communication

---

## Security & Authentication

Hector provides comprehensive security features:

### Authentication Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    A2A Protocol Layer                       │
│  ┌─────────────────┬─────────────────┬─────────────────┐    │
│  │ Agent-to-Agent  │ Protobuf        │ A2A             │    │
│  │ Protocol        │ Messages        │ Specification   │    │
│  └─────────┬───────┴─────────┬───────┴─────────┬───────┘    │
│            │                 │                 │            │
│            ▼                 ▼                 ▼            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Hector Implementation                     │ │
│  │  ┌─────────────┬─────────────┬─────────────┐            │ │
│  │  │ Transport   │ Runtime     │ Agent       │            │ │
│  │  │ Layer       │ System      │ Management  │            │ │
│  │  └─────────────┴─────────────┴─────────────┘            │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
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
│                    A2A Protocol Layer                       │
│  ┌─────────────────┬─────────────────┬─────────────────┐    │
│  │ Agent-to-Agent  │ Protobuf        │ A2A             │    │
│  │ Protocol        │ Messages        │ Specification   │    │
│  └─────────┬───────┴─────────┬───────┴─────────┬───────┘    │
│            │                 │                 │            │
│            ▼                 ▼                 ▼            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                Hector Implementation                     │ │
│  │  ┌─────────────┬─────────────┬─────────────┐            │ │
│  │  │ Transport   │ Runtime     │ Agent       │            │ │
│  │  │ Layer       │ System      │ Management  │            │ │
│  │  └─────────────┴─────────────┴─────────────┘            │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```
### Plugin Types

| Type | Purpose | Interface |
|------|---------|-----------|
| **LLM Provider** | Custom language models | Text generation, streaming |
| **Database Provider** | Vector databases | Embeddings, search |
| **Tool Provider** | Custom capabilities | Function execution |

### Plugin Features

- **Language Agnostic** - Write in any gRPC-supported language
- **Process Isolation** - Plugins run in separate processes
- **Hot Reload** - Update plugins without restart
- **High Performance** - gRPC-based communication

---

## Production Deployment

Hector is designed for production environments:

### Deployment Patterns

=== "Single Server"
    ```bash
    # Simple deployment
    hector serve --port 8080 --config production.yaml
    ```

=== "Load Balanced"
    ```yaml
    # Multiple instances behind load balancer
    servers:
      - host: "hector-1.internal"
        port: 8080
      - host: "hector-2.internal"
        port: 8080
    ```

=== "Kubernetes"
    ```yaml
    # Kubernetes deployment
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: hector
    spec:
      replicas: 3
      template:
        spec:
          containers:
          - name: hector
            image: hector:latest
            ports:
            - containerPort: 8080
    ```

### Production Features

- **Horizontal Scaling** - Multiple server instances
- **Health Checks** - Built-in health monitoring
- **Structured Logging** - JSON logs for monitoring
- **Metrics** - Performance and usage metrics
- **Backup/Restore** - Configuration and state management

---

## Monitoring & Observability

Hector provides comprehensive monitoring capabilities:

### Monitoring Features

- **Metrics** - Performance and usage metrics
- **Structured Logging** - JSON logs with correlation IDs
- **Distributed Tracing** - Request flow tracking
- **Health Checks** - Service health monitoring
- **Dashboards** - Real-time monitoring dashboards

### Configuration

```yaml
# Monitoring configuration
monitoring:
  metrics:
    enabled: true
    port: 9090
  
  logging:
    level: "info"
    format: "json"
  
  health:
    enabled: true
    endpoint: "/health"
```

---

## Migration & Upgrades

Hector supports smooth migration and upgrades:

### Migration Features

- **Configuration Migration** - Automatic config updates
- **Data Migration** - Seamless data migration
- **Rolling Updates** - Zero-downtime updates
- **Version Compatibility** - Backward compatibility

### Upgrade Process

1. **Backup** - Backup current configuration and data
2. **Download** - Download new Hector version
3. **Update** - Update configuration if needed
4. **Deploy** - Deploy new version
5. **Verify** - Verify functionality

---

## Best Practices

### Architecture Best Practices

- **Separation of Concerns** - Keep transport, runtime, and business logic separate
- **Interface-Based Design** - Use interfaces for all major components
- **Security First** - Implement authentication and authorization
- **Scalability** - Design for horizontal scaling
- **Maintainability** - Keep code clean and well-documented

### Deployment Best Practices

- **Security** - Use TLS/SSL, secure secrets management
- **Scaling** - Use load balancers and multiple instances
- **Monitoring** - Implement comprehensive monitoring
- **Backup** - Regular backups of configuration and data
- **Updates** - Regular updates and security patches

---

## Related Documentation

- [Building Agents](agents.md) - Learn how to build AI agents
- [Configuration Reference](configuration.md) - Complete configuration guide
- [Tools & Extensions](tools.md) - Built-in tools and custom extensions
- [Memory Management](memory.md) - Memory system configuration
- [Plugin Development](plugins.md) - Build custom plugins
