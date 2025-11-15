---
title: Why Hector?
description: Understanding when and why to choose Hector for your AI agent deployment
---

# Why Hector?

## Built for Production from Day One

Hector is designed for teams who need to **run AI agents in production** with the same operational rigor as their other services. Instead of adding production capabilities as an afterthought, Hector makes them foundational.

### Production-Ready by Design

Hector brings production-grade capabilities that enterprise teams expect:

- **Built-in Observability**: Prometheus metrics and OpenTelemetry tracing out of the box
- **Hot Configuration Reload**: Update agents without downtime or redeployment
- **Security Primitives**: JWT authentication, RBAC, and command sandboxing built in
- **Resource Efficient**: 30MB binary with <100ms startup for dense deployments
- **Standards-Based**: Native A2A protocol support for interoperability

### Advanced Agent Capabilities

Beyond production features, Hector provides sophisticated agentic capabilities through pure YAML configuration:

- **Multi-Agent Orchestration**: Coordinate specialized agents with A2A-native federation
- **Advanced Reasoning**: Chain-of-thought and iterative reasoning strategies
- **Flexible Memory**: Working memory with RAG and vector stores for long-term context
- **Extensive Tooling**: Built-in tools plus MCP protocol for endless extensibility
- **Real-Time Streaming**: Server-sent events for responsive user experiences

## Core Philosophy

### 1. **Operations-First**

**Hot Configuration Reload**
```bash
# Update agent configuration in Consul
consul kv put hector/prod @new-config.json

# Agents reload automatically—no restart, no downtime
```

**Built-in Observability**

- **Prometheus metrics**: latency, token usage, costs, errors
- **OpenTelemetry traces**: distributed tracing across agent calls
- **Structured logging**: context propagation and correlation

**Production Deployment Patterns**

- Kubernetes-ready with health checks
- Distributed config from Consul/Etcd/ZooKeeper
- Zero-downtime rolling updates

### 2. **Security-First**

Not security as an afterthought—security as a design principle:

- **Authentication**: JWT with JWKS, API keys
- **Authorization**: Authentication-based visibility controls (public/internal/private agents)
- **Command Sandboxing**: Allowlist-based tool execution
- **Secret Management**: Environment variable interpolation

```yaml
global:
  auth:
    enabled: true
    jwks_url: https://auth.company.com/.well-known/jwks.json

agents:
  internal-analyst:
    visibility: private  # Not exposed externally
    llm: gpt-4o
    tools:
      - name: command
        config:
          sandboxing: true
          allowed_commands: [ls, cat, grep]
```

### 3. **Standards-Based**

**A2A Protocol Native**

Hector implements the Agent-to-Agent (A2A) protocol as a first-class citizen:

- Standardized discovery, messaging, and streaming
- Federation: agents call agents across boundaries
- Interoperability: integrate external A2A services

```yaml
agents:
  coordinator:
    llm: gpt-4o
    tools: [agent_call]  # Call any A2A agent

  external-specialist:
    type: a2a
    url: https://external-service/v1
    description: "Specialized processing service"
```

### 4. **Zero-Code Configuration**

Define sophisticated agents entirely in YAML:

```yaml
agents:
  analyst:
    llm: gpt-4o
    tools: [search, write_file, agent_call]
    reasoning:
      engine: chain-of-thought
      max_iterations: 100
    memory:
      working:
        strategy: summary_buffer
        max_tokens: 4000
      long_term:
        type: vector
        vector_store: qdrant-cluster
```

No Python. No JavaScript. No code.

## When to Choose Hector

### ✅ Choose Hector When:

- **Running agents in production** with SLAs and uptime requirements
- **Need observability** (Prometheus, OpenTelemetry, distributed tracing)
- **Require security controls** (auth, authorization, sandboxing)
- **Want operational flexibility** (hot reload, distributed config)
- **Building multi-agent systems** with A2A federation
- **Deploying to resource-constrained environments** (edge, Lambda)
- **Platform engineers** managing AI infrastructure for teams

### ⚠️ Consider Alternatives When:

- **Rapid prototyping** with frequent code changes
- **Heavy Python ecosystem integration** required
- **Custom agent behaviors** needing programmatic control
- **Research projects** without production requirements

## Comparison: Hector vs. Traditional Frameworks

| Aspect | Hector | LangChain/LlamaIndex |
|--------|--------|---------------------|
| **Configuration** | Pure YAML | Python/JS code |
| **Hot Reload** | Built-in, zero-downtime | Requires redeployment |
| **Observability** | Prometheus + OTEL native | Manual instrumentation |
| **Security** | JWT, visibility controls, sandboxing | DIY |
| **Binary Size** | 30MB (stripped) | 200-500MB+ runtime |
| **Startup Time** | <100ms | 2-10 seconds |
| **Deployment** | Single 30MB binary | Runtime + dependencies |
| **A2A Protocol** | Native | Not supported |
| **Best For** | Production operations | Development/prototyping |

## Architecture for Production

Hector's design makes production deployment natural:

```
┌─────────────────────────────────────────┐
│         Load Balancer                    │
└─────────────┬───────────────────────────┘
              │
        ┌─────┴─────┬─────────┐
        ▼           ▼         ▼
   ┏━━━━━━━┓   ┏━━━━━━━┓   ┏━━━━━━━┓
   ┃Hector ┃   ┃Hector ┃   ┃Hector ┃
   ┃ Pod 1 ┃   ┃ Pod 2 ┃   ┃ Pod 3 ┃
   ┗━━━┬━━━┛   ┗━━━┬━━━┛   ┗━━━┬━━━┛
        │           │           │
        └───────────┴───────────┘
                    │
        ┌───────────┼──────────┬──────────┐
        ▼           ▼          ▼          ▼
   ┌────────┐  ┌────────┐ ┌────────┐ ┌────────┐
   │ Consul │  │Postgres│ │ Qdrant │ │ OTEL   │
   │ Config │  │Sessions│ │  RAG   │ │Collector│
   └────────┘  └────────┘ └────────┘ └────────┘
```

**Key Characteristics:**

- **Stateless servers**: Scale horizontally
- **Distributed config**: Centralized, hot-reloadable
- **Session persistence**: SQL-backed continuity
- **Observability**: Metrics and traces to collectors

## Real-World Use Cases

### 1. **Enterprise Agent Infrastructure**

Platform teams deploying multi-agent systems across organizations with A2A v0.3.0 compliance, security, and observability requirements.

**Example:** Central platform team provides agent infrastructure via YAML configs to 50+ product teams. JWT auth ensures proper access control, Prometheus metrics track usage across teams, hot reload enables rapid iteration without downtime.

### 2. **High-Throughput Production Services**

Customer-facing AI services needing low latency, high availability, and efficient resource usage.

**Example:** E-commerce company runs 100+ concurrent agents handling customer inquiries. 128MB footprint per pod enables dense packing on Kubernetes, <100ms startup enables fast autoscaling, OpenTelemetry traces track requests across agent orchestration.

### 3. **Edge/IoT Deployments**

Running agents on resource-constrained devices.

**Example:** Manufacturing company deploys agents on edge devices (10MB binary, 128MB RAM) for real-time quality analysis. Single binary simplifies deployment, Go efficiency enables running on ARM processors.

### 4. **Regulated Industries**

Financial services, healthcare, etc. requiring audit trails, RBAC, and security controls.

**Example:** Financial institution uses Hector for internal analyst agents. JWT integration with corporate IdP, command sandboxing restricts file system access, OpenTelemetry provides audit trails for compliance.

### 5. **Multi-Team Agent Platforms**

Central platform teams providing agent infrastructure to product teams via declarative config.

**Example:** SaaS company platform team maintains Hector infrastructure. Product teams submit YAML configs via GitOps pipeline, platform team reviews and deploys via hot reload, Prometheus provides centralized monitoring dashboard.

## Target Audience

### **Primary: Platform Engineers & SREs**

You manage infrastructure and need to deploy AI agents with the same operational standards as other production services:

- Observable (metrics, traces, logs)
- Secure (auth, authorization, sandboxing)
- Reliable (health checks, graceful shutdown, zero-downtime updates)
- Efficient (low resource usage, fast startup)

### **Secondary: AI Product Teams**

You build AI-powered products and need production infrastructure without managing complexity:

- No code required (pure YAML)
- Fast iteration (hot reload)
- Multi-agent orchestration (A2A native)
- Flexible deployment modes

## Performance Benefits

### Resource Efficiency

| Metric | Hector | Python Frameworks | Difference |
|--------|--------|------------------|------------|
| Binary Size | 30MB (stripped) | 200-500MB+ runtime | **7-15x smaller** |
| Startup Time | <100ms | 2-10s | **20-100x faster** |
| Runtime Footprint | Minimal (Go) | 200-500MB+ (Python) | **10-20x less** |
| Container Image | 50MB (Alpine) | 500MB-2GB+ | **10-40x smaller** |

### Cost Implications

**Example: 100 agent deployment**

- **Python Framework**: 100 pods × 500MB = 50GB RAM baseline
- **Hector**: 100 pods × 50MB = 5GB RAM baseline
- **Savings**: ~90% reduction in baseline resource usage

### Edge Deployment

Hector's efficiency enables edge/IoT deployment impossible with Python frameworks:

- Raspberry Pi 4 (4GB RAM): 10-20 Hector agents vs. 2-5 Python agents
- AWS Lambda: Fast cold starts (<100ms) vs. multi-second Python cold starts
- Edge devices: Single binary, no runtime dependencies

## Developer Experience

### Configuration-as-Code

```yaml
# This is the entire configuration—no Python/JS required
agents:
  analyst:
    llm: gpt-4o
    tools: [search, write_file, agent_call]
    reasoning:
      engine: chain-of-thought
      max_iterations: 100
    memory:
      working:
        strategy: summary_buffer
        max_tokens: 4000
      long_term:
        type: vector
        vector_store: qdrant-cluster

llms:
  gpt-4o:
    type: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}

databases:
  qdrant-cluster:
    type: qdrant
    host: qdrant.internal
    port: 6334
```

### GitOps-Friendly

Store configurations in Git, deploy via CI/CD:

```bash
# Development
git checkout -b add-new-agent
vim agents.yaml  # Add agent config
git commit -m "Add customer support agent"
git push

# Production (via hot reload)
consul kv put hector/prod @agents.yaml
# Agents update automatically, no restart needed
```

### Testing & Validation

```bash
# Validate configuration
hector validate --config agents.yaml

# Test locally
hector call "Test message" --agent analyst --config agents.yaml

# Integration test
hector serve --config agents.yaml &
curl -X POST http://localhost:8080/v1/agents/analyst/message:send \
  -d '{"message": {"parts": [{"text": "Test"}], "role": "user"}}'
```

## Migration Path

### From LangChain/LlamaIndex

1. **Extract LLM configuration** → Hector `llms:` section
2. **Extract agent prompts** → Hector `prompt:` section
3. **Map tools** → Hector built-in tools or MCP servers
4. **Configure memory** → Hector memory strategies
5. **Deploy** → Single binary, no refactoring

**Example Translation:**

=== "LangChain (Python)"
    ```python
    from langchain.agents import initialize_agent
    from langchain.llms import OpenAI
    from langchain.tools import Tool

    llm = OpenAI(temperature=0.7, model="gpt-4")
    tools = [search_tool, write_file_tool]

    agent = initialize_agent(
        tools=tools,
        llm=llm,
        agent="zero-shot-react-description",
        verbose=True
    )
    ```

=== "Hector (YAML)"
    ```yaml
    agents:
      assistant:
        llm: gpt-4o
        tools: [search, write_file]
        reasoning:
          engine: chain-of-thought

    llms:
      gpt-4o:
        type: openai
        model: gpt-4o
        temperature: 0.7
        api_key: ${OPENAI_API_KEY}
    ```

No Python code required. Clearer, more maintainable, production-ready.

## Common Questions

### "Can I still use custom code?"

Yes, via the **plugin system**:

```go
// Custom Go plugin
package main

import "github.com/kadirpekel/hector/pkg/tool"

func MyCustomTool(input string) (string, error) {
    // Your logic here
    return result, nil
}
```

But most use cases are covered by built-in tools + MCP integration.

### "How does it compare to Semantic Kernel?"

| Aspect | Hector | Semantic Kernel |
|--------|--------|-----------------|
| Language | Go (production) | C#/.NET |
| Configuration | Pure YAML | Code + config |
| A2A Protocol | Native | Not supported |
| Hot Reload | Yes | No |
| Observability | Built-in | Manual |
| Deployment | Single binary | .NET runtime required |

### "Is Python better for AI/ML?"

For **research and prototyping**, yes. For **production deployment**, Go offers:

- 10-20x less resource usage
- 20-100x faster startup
- Single binary deployment (30MB vs 200-500MB+ runtime)
- Better concurrency (goroutines vs. GIL)
- Easier operations (no dependency hell)

Hector doesn't replace Python notebooks or research code—it provides production infrastructure for deploying agents.

### "What about performance of LLM calls?"

LLM API latency dominates performance, not the framework. Hector's efficiency matters for:

- **Infrastructure costs** (100x less memory)
- **Startup time** (autoscaling, edge deployment)
- **Concurrent agents** (goroutines vs. threads)

## Getting Started

Ready to deploy production-ready agents?

[Quick Start Guide →](getting-started/quick-start.md){ .md-button .md-button--primary }
[Production Deployment →](how-to/deploy-production.md){ .md-button }

## Next Steps

- **[Core Concepts](core-concepts/overview.md)** - Understand Hector's architecture
- **[Observability](core-concepts/observability.md)** - Setup metrics and tracing
- **[Security](core-concepts/security.md)** - Configure auth and authorization
- **[Multi-Agent Systems](core-concepts/multi-agent.md)** - Build agent orchestration
- **[Configuration Reference](reference/configuration.md)** - Complete configuration guide
