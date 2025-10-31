```
██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```
[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE.md)
[![A2A Protocol](https://img.shields.io/badge/A2A%20v0.3.0-100%25%20compliant-brightgreen.svg)](https://gohector.dev/reference/a2a-protocol/)
[![Documentation](https://img.shields.io/badge/docs-gohector.dev-blue.svg)](https://gohector.dev)
[![Go Report Card](https://goreportcard.com/badge/github.com/kadirpekel/hector)](https://goreportcard.com/report/github.com/kadirpekel/hector)

**Production-Grade A2A-Native Agent Platform**

Deploy observable, secure, and scalable AI agents in production—with zero code.

[Documentation](https://gohector.dev) • [Quick Start](https://gohector.dev/getting-started/quick-start/) • [Production Guide](https://gohector.dev/how-to/deploy-production/)

## Overview

Hector is an AI agent platform designed for production deployment, built in Go for performance and operational simplicity. Define sophisticated multi-agent systems through declarative YAML configuration without writing code.

### Key Characteristics

- **Zero-Code Configuration**: Pure YAML agent definition, no Python/Go required
- **Hot Reload**: Update configurations without downtime or restart
- **A2A Protocol v0.3.0 Native**: 100% standards-compliant with enhanced dual-path REST support
- **Production Observability**: Built-in Prometheus metrics and OpenTelemetry tracing
- **Security-First**: JWT authentication, visibility controls, and command sandboxing out of the box
- **Resource Efficient**: Single 30MB binary (stripped), minimal runtime footprint

## Quick Start

### Server Mode (Production)

Start Hector as a multi-agent server exposing REST, gRPC, and WebSocket APIs:

```bash
# Install
go install github.com/kadirpekel/hector/cmd/hector@latest

# Create configuration
cat > agents.yaml << EOF
agents:
  analyst:
    llm: gpt-4o
    tools: [search, write_file, search_replace]
    reasoning:
      engine: chain-of-thought
      max_iterations: 100
EOF

# Export credentials
export OPENAI_API_KEY="sk-..."

# Start server
hector serve --config agents.yaml
```

### Client Access

**Using Hector CLI (Client Mode):**

```bash
# Send message to agent
hector call "Analyze system architecture and suggest improvements" --agent analyst --server http://localhost:8080

# Interactive chat
hector chat --agent analyst --server http://localhost:8080

# List available agents
hector list --server http://localhost:8080
```

**Using REST API (curl):**

```bash
# Send message (A2A Protocol compliant)
curl -X POST http://localhost:8080/v1/agents/analyst/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "parts": [{"text": "Analyze system architecture"}],
      "role": "user"
    }
  }'

# Stream responses (SSE)
curl -N http://localhost:8080/v1/agents/analyst/message:stream \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "parts": [{"text": "Generate report"}],
      "role": "user"
    }
  }'

# List agents
curl http://localhost:8080/v1/agents
```

### Local Mode (Development)

For quick testing without a server:

```bash
# Direct CLI interaction with configuration
hector call "Explain distributed systems" --config agents.yaml

# Zero-config mode (uses default agent)
export OPENAI_API_KEY="sk-..."
hector call "Analyze code quality"
```

## Configuration Example

```yaml
agents:
  analyst:
    llm: gpt-4o
    tools: [search, write_file, search_replace]
    reasoning:
      engine: chain-of-thought
      max_iterations: 100
    memory:
      working:
        strategy: summary_buffer
        max_tokens: 4000
      long_term:
        type: vector
        database: production-qdrant
        
  researcher:
    llm: claude-3-5-sonnet
    tools: [search, agent_call]
    reasoning:
      engine: chain-of-thought
      
llms:
  gpt-4o:
    type: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}
    
  claude-3-5-sonnet:
    type: anthropic
    model: claude-3-5-sonnet-20241022
    api_key: ${ANTHROPIC_API_KEY}

databases:
  production-qdrant:
    type: qdrant
    host: qdrant.internal
    port: 6334
```

[Configuration Reference](https://gohector.dev/reference/configuration/)

## Why Hector?

**For Platform Engineers & SREs:**
- **Operational Excellence**: Built-in Prometheus metrics and OpenTelemetry tracing
- **Zero-Downtime Updates**: Hot reload configurations from Consul/Etcd/ZooKeeper
- **Security Native**: JWT authentication, visibility controls, command sandboxing out of the box
- **Resource Efficient**: Single 30MB binary, minimal dependencies, runs anywhere

**For AI Product Teams:**
- **Zero-Code Configuration**: Pure YAML, no Python/Go required
- **A2A-Native**: Standards-based agent communication and federation
- **Multi-Agent Ready**: Supervisor reasoning + agent_call tool built-in
- **Flexible Deployment**: Local dev, server mode, or distributed

**Built in Go for Production:**
Unlike Python-based frameworks requiring 200-500MB+ runtimes, Hector offers a single 30MB binary with <100ms startup, perfect for Kubernetes, edge devices, or Lambda.

[Learn More: Why Hector?](https://gohector.dev/why-hector/) • [Compare with Alternatives](https://gohector.dev/why-hector/#comparison-hector-vs-traditional-frameworks)

## Production Features

### Distributed Configuration Management

Centralized configuration with automatic reload for production deployments:

**Supported Backends:**
- **File**: Local YAML files with filesystem watching
- **Consul**: HashiCorp Consul KV store (JSON format)
- **Etcd**: Distributed key-value store (JSON format)
- **ZooKeeper**: Apache ZooKeeper coordination service (YAML format)

**Configuration Format by Backend:**
- **File, ZooKeeper**: YAML
- **Consul, Etcd**: JSON (native format for KV stores)

**Server Configuration:**
```bash
# File-based with auto-reload
hector serve --config config.yaml --config-watch

# Consul cluster (JSON configuration)
consul kv put hector/prod @production.json
hector serve --config hector/prod \
  --config-type consul \
  --config-endpoints "consul1:8500,consul2:8500,consul3:8500" \
  --config-watch

# Etcd cluster (JSON configuration)
etcdctl put /hector/production < production.json
hector serve --config /hector/production \
  --config-type etcd \
  --config-endpoints "etcd1:2379,etcd2:2379,etcd3:2379" \
  --config-watch

# ZooKeeper ensemble (YAML configuration)
hector serve --config /hector/config \
  --config-type zookeeper \
  --config-endpoints "zk1:2181,zk2:2181,zk3:2181" \
  --config-watch
```

**Features:**
- Graceful server reload on configuration changes
- Configuration validation before applying updates
- Zero-downtime updates with load balancer
- Centralized management across clusters

[Distributed Configuration Guide](https://gohector.dev/reference/distributed-configuration/)

### Observability

Production-grade observability built-in:

**Metrics (Prometheus):**
- Agent execution latency and throughput
- Token usage and estimated costs
- Tool invocation statistics
- Error rates and types
- Memory consumption

**Tracing (OpenTelemetry):**
- Distributed trace propagation
- A2A protocol call tracing
- LLM request/response spans
- Tool execution traces
- Export to Jaeger, Datadog, Honeycomb, or OTLP collectors

**Configuration:**
```yaml
global:
  observability:
    metrics:
      enabled: true
    tracing:
      enabled: true
      exporter_type: otlp
      endpoint_url: http://otel-collector:4317
      sampling_rate: 1.0
```

**Note:** Metrics are served on the HTTP server at `/metrics` endpoint (default: `http://localhost:8080/metrics`).

**Grafana Integration:**
Pre-built dashboards included for agent performance monitoring.

[Observability Guide](https://gohector.dev/core-concepts/observability/)

### Security

Production-grade security controls:

**Authentication:**
- JWT token validation with JWKS
- API key authentication
- Per-agent access control

**Authorization:**
- Agent-level visibility (public, private, internal)
- Tool execution restrictions
- Command sandboxing with allowlist

**Configuration:**
```yaml
global:
  auth:
    enabled: true
    jwks_url: https://auth.company.com/.well-known/jwks.json
    issuer: https://auth.company.com
    audience: hector-api

agents:
  internal-analyst:
    visibility: private  # Not exposed via A2A discovery
    llm: gpt-4o
    
  public-assistant:
    visibility: public
    llm: gpt-4o
    tools:
      - name: command
        config:
          sandboxing: true
          allowed_commands: [ls, cat, grep]
```

[Security Guide](https://gohector.dev/core-concepts/security/)

### Multi-Agent Orchestration

Coordinate specialized agents for complex workflows using A2A protocol:

**Native Agents:**
```yaml
agents:
  orchestrator:
    llm: gpt-4o
    tools: [agent_call]
    reasoning:
      engine: supervisor  # Optimized for multi-agent coordination
      max_iterations: 10
    description: "Routes tasks to specialist agents based on requirements"
    
  code-analyst:
    llm: gpt-4o
    tools: [search, search_replace, execute_command]
    reasoning:
      engine: chain-of-thought
      max_iterations: 100
    description: "Analyzes codebases and identifies issues"
    
  documentation-writer:
    llm: claude-3-5-sonnet
    tools: [search, write_file]
    reasoning:
      engine: chain-of-thought
      max_iterations: 100
    description: "Creates technical documentation"
```

**External Agents (A2A Federation):**
```yaml
agents:
  coordinator:
    type: native
    llm: gpt-4o
    tools: [agent_call]
    reasoning:
      engine: supervisor
    
  data-processor:
    type: a2a
    url: https://data-service.internal/v1
    description: "Processes large datasets"
    
  ml-predictor:
    type: a2a  
    url: https://ml-service.internal/v1
    description: "Runs ML inference"
```

[Multi-Agent Guide](https://gohector.dev/core-concepts/multi-agent/)

## Advanced Capabilities

### Memory Management

**Working Memory:**
- **Buffer**: Fixed-size conversation history
- **Summary Buffer**: Automatic summarization of older messages
- **Sliding Window**: Token-based context window
- **Summary + Recent**: Combines summary with recent messages

**Long-Term Memory:**
- Vector-based semantic memory with RAG
- Automatic document indexing
- Semantic search across knowledge bases
- Multiple vector database support (Qdrant)

**Session Persistence:**
- SQL-based session storage
- Cross-session memory continuity
- Conversation history retrieval

[Memory Guide](https://gohector.dev/core-concepts/memory/)

### Tool Ecosystem

**Built-in Tools:**
- File operations (write, search/replace)
- Command execution (sandboxed)
- Semantic search (RAG integration)
- Agent-to-agent communication
- Todo management

**MCP Support:**
- Model Context Protocol for extensible tools
- HTTP-based MCP server integration
- Automatic tool discovery

**Custom Plugins:**
- Go plugin system for domain-specific tools
- Compile-time tool registration
- Full access to Go ecosystem

[Tools Guide](https://gohector.dev/core-concepts/tools/)

### Reasoning Engines

**Chain-of-Thought:**
- Single-agent iterative reasoning
- Native function calling
- Implicit planning and completion detection
- Streaming support
- Fast and cost-effective

**Supervisor:**
- Multi-agent orchestration strategy
- Specialized prompts for task decomposition
- Agent selection and delegation guidance
- Result synthesis and integration
- Works with agent_call tool
- Systematic todo tracking for coordination

[Reasoning Guide](https://gohector.dev/core-concepts/reasoning/)

## Performance

Hector is optimized for production deployments:

| Metric | Value | Comparison |
|--------|-------|------------|
| Binary Size | 30MB (stripped) | Single executable, no runtime dependencies |
| Startup Time | <100ms | 20-100x faster than Python |
| Resource Usage | Minimal | 10-20x less than Python frameworks |
| Concurrent Agents | 100+ | Per instance |
| Resource Cost | -90% | vs Python alternatives |

**Deployment Characteristics:**
- No Python runtime required
- No dependency installation
- Cross-platform single binary
- Horizontal scaling ready
- Edge device compatible

[Performance Details](https://gohector.dev/performance/)

## Deployment

### Docker

```dockerfile
FROM golang:1.24-alpine AS builder
RUN go install github.com/kadirpekel/hector/cmd/hector@latest

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/bin/hector /usr/local/bin/
ENTRYPOINT ["hector"]
CMD ["serve", "--config", "/config/agents.yaml"]
```

Build and run:
```bash
docker build -t hector:latest .
docker run -p 8080:8080 -p 50051:50051 \
  -v $(pwd)/config:/config \
  -e OPENAI_API_KEY=$OPENAI_API_KEY \
  hector:latest
```

### Kubernetes

Production deployment with distributed configuration:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hector-server
  namespace: ai-agents
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hector
  template:
    metadata:
      labels:
        app: hector
    spec:
      containers:
      - name: hector
        image: hector:latest
        args:
          - serve
          - --config
          - /hector/production
          - --config-type
          - etcd
          - --config-endpoints
          - etcd-cluster:2379
          - --config-watch
        ports:
        - containerPort: 50051
          name: grpc
        - containerPort: 8080
          name: http
        env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: hector-secrets
              key: openai-api-key
        - name: ANTHROPIC_API_KEY
          valueFrom:
            secretKeyRef:
              name: hector-secrets
              key: anthropic-api-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: hector-service
  namespace: ai-agents
spec:
  selector:
    app: hector
  ports:
  - name: grpc
    port: 50051
    targetPort: 50051
  - name: http
    port: 8080
    targetPort: 8080
  type: LoadBalancer
```

### Configuration Management

**Centralized Configuration:**
Store agent configurations in Consul, Etcd, or ZooKeeper for:
- Single source of truth across clusters
- Automatic agent updates without redeployment
- Version control and audit trails
- Environment-specific configurations

**Example Workflow:**
```bash
# Upload JSON configuration to Consul
consul kv put hector/production @production.json

# Agents automatically detect and reload
# No pod restart required
```

[Deployment Guide](https://gohector.dev/how-to/deploy-production/)

## API Access

Hector exposes multiple transport protocols for maximum flexibility:

### HTTP APIs (Port 8080)

All HTTP-based APIs (REST, JSON-RPC, WebSocket, Web UI) are served on port 8080:

```bash
# REST API: Send message (A2A Protocol compliant)
curl -X POST http://localhost:8080/v1/agents/analyst/message:send \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "message": {
      "parts": [{"text": "Analyze system performance"}],
      "role": "user"
    }
  }'

# REST API: Stream responses (SSE)
curl -N http://localhost:8080/v1/agents/analyst/message:stream \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "parts": [{"text": "Generate report"}],
      "role": "user"
    }
  }'

# REST API: List agents
curl http://localhost:8080/v1/agents

# REST API: Agent discovery
curl http://localhost:8080/v1/agents/analyst

# JSON-RPC: Call agent (single-agent mode)
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"message/send","params":{"request":{"parts":[{"text":"Hello"}],"role":"user"}},"id":"1"}'

# JSON-RPC: Call specific agent (multi-agent mode)
curl -X POST 'http://localhost:8080/?agent=orchestrator' \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"message/send","params":{"request":{"parts":[{"text":"Hello"}],"role":"user"}},"id":"1"}'

# Web UI: Open in browser
open http://localhost:8080/
```

### gRPC (Port 8080)

```go
import (
    "google.golang.org/grpc"
    pb "github.com/kadirpekel/hector/pkg/a2a/pb"
)

conn, _ := grpc.Dial("localhost:8080", grpc.WithInsecure())
client := pb.NewA2AServiceClient(conn)

resp, _ := client.SendMessage(ctx, &pb.MessageRequest{
    AgentId: "analyst",
    Content: "Analyze architecture",
})
```

### WebSocket (Port 8080)

```javascript
const ws = new WebSocket('ws://localhost:8080/v1/agents/analyst/ws');

ws.send(JSON.stringify({
  content: "Real-time analysis"
}));

ws.onmessage = (event) => {
  console.log(JSON.parse(event.data));
};
```

[API Reference](https://gohector.dev/reference/api/)

## A2A Protocol

Hector implements the Agent-to-Agent (A2A) protocol for standardized agent communication:

**Core Capabilities:**
- **Discovery**: Automatic agent capability discovery and registration
- **Streaming**: Real-time bidirectional streaming with SSE and WebSocket
- **Task Management**: Asynchronous task submission, tracking, and retrieval
- **Federation**: Inter-agent communication across network boundaries

**Example:**
```yaml
# Server A
agents:
  coordinator:
    type: native
    llm: gpt-4o
    tools: [agent_call]

# Server B  
agents:
  specialist:
    type: native
    llm: gpt-4o
    
# Coordinator can call specialist via A2A:
# POST /v1/agents/specialist/message:send
```

[A2A Protocol Specification](https://gohector.dev/reference/a2a-protocol/)

## Documentation

Comprehensive documentation available at [gohector.dev](https://gohector.dev):

**Getting Started:**
- [Installation](https://gohector.dev/getting-started/installation/)
- [Quick Start](https://gohector.dev/getting-started/quick-start/)

**Core Concepts:**
- [Overview](https://gohector.dev/core-concepts/overview/)
- [Memory Management](https://gohector.dev/core-concepts/memory/)
- [Tools](https://gohector.dev/core-concepts/tools/)
- [Reasoning](https://gohector.dev/core-concepts/reasoning/)
- [Multi-Agent Systems](https://gohector.dev/core-concepts/multi-agent/)
- [Observability](https://gohector.dev/core-concepts/observability/)
- [Security](https://gohector.dev/core-concepts/security/)

**Configuration:**
- [Configuration Reference](https://gohector.dev/reference/configuration/)
- [Distributed Configuration](https://gohector.dev/reference/distributed-configuration/)
- [CLI Reference](https://gohector.dev/reference/cli/)

**Deployment:**
- [Production Deployment](https://gohector.dev/how-to/deploy-production/)
- [Setup RAG](https://gohector.dev/how-to/setup-rag/)
- [Integrate External Agents](https://gohector.dev/how-to/integrate-external-agents/)

## Examples

Production-ready configuration examples in [`/configs`](./configs):

| Example | Description |
|---------|-------------|
| `coding.yaml` | Code analysis and development assistant |
| `research-assistant.yaml` | Research and documentation agent |
| `orchestrator-example.yaml` | Multi-agent coordination patterns |
| `observability-example.yaml` | Metrics and tracing configuration |
| `security-example.yaml` | Authentication and authorization |
| `multi-agent-sessions-example.yaml` | Session persistence |
| `tools-mcp-example.yaml` | MCP tool integration |

## License

AGPL-3.0 License. See [LICENSE.md](LICENSE.md) for details.

## Status

Alpha version. APIs and configuration format may change as the platform evolves.
