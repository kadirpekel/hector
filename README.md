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

**[📚 Full Documentation →](https://gohector.dev)** | [Quick Start](https://gohector.dev/getting-started/quick-start/) | [API Reference](https://gohector.dev/reference/api/)

---

## Quick Start

```bash
# Install
go install github.com/kadirpekel/hector/cmd/hector@latest

# Create configuration
cat > agents.yaml << EOF
agents:
  assistant:
    llm: gpt-4o
    tools: [search, write_file]
EOF

# Start server
export OPENAI_API_KEY="sk-..."
hector serve --config agents.yaml
```

Visit `http://localhost:8080` for the web UI or use the [CLI](https://gohector.dev/reference/cli/) and [REST API](https://gohector.dev/reference/api/).

### Zero-Config Mode

No YAML file needed—configure common use cases via command-line flags:

```bash
# Complete RAG system with Docling document parsing
hector serve \
  --docs-folder ./documents \
  --mcp-url http://docling:8000/mcp \
  --mcp-parser-tool convert_document_into_docling_document
```

This instantly enables document indexing, semantic search, and RAG capabilities. See [Zero-Config Mode](https://gohector.dev/getting-started/quick-start/#zero-config-mode) for more options.

## Features

### Production & Enterprise

- **Observability** - Prometheus metrics, OpenTelemetry tracing with Jaeger/Datadog/Honeycomb export, Grafana dashboards
- **Security** - JWT authentication with JWKS (Auth0/Keycloak/Okta), agent-level security schemes (Bearer, API key), command sandboxing
- **Distributed Configuration** - Hot reload from Consul/Etcd/ZooKeeper, zero-downtime configuration updates
- **Rate Limiting** - Multi-layer time windows (minute/hour/day/week/month), token & request count tracking, per-session or per-user scoping, SQL or memory backend
- **Session Persistence** - SQL-based storage (SQLite/Postgres), cross-session memory continuity, conversation history retrieval
- **Human-in-the-Loop** - Tool approval workflows, async HITL with state persistence (survives restarts), A2A Protocol compliant (TASK_STATE_INPUT_REQUIRED)
- **Checkpoint Recovery** - Crash recovery, rate limit resilience, long-running task support, event-driven and interval-based strategies
- **TLS/HTTPS** - Built-in TLS support for A2A server and vector stores
- **Health Checks** - Kubernetes-ready liveness/readiness probes

### Core Agent Capabilities

- **Memory Management** - Working memory strategies (buffer window, summary buffer), long-term memory with RAG, vector stores (Qdrant, Pinecone, Weaviate, Milvus, Chroma)
- **Reasoning Engines** - Chain-of-thought (iterative reasoning with tool execution), Supervisor (multi-agent orchestration and task decomposition)
- **Tools** - 10+ built-in tools (execute_command, write_file, read_file, search_replace, apply_patch, grep_search, search, evaluate_rag, todo_write, agent_call, web_request), MCP protocol support (150+ integrations via Composio), gRPC plugins for custom tools
- **Multi-Agent Orchestration** - Supervisor reasoning engine, agent_call tool, A2A-native federation, external A2A agent integration
- **Streaming** - Server-sent events (SSE) for real-time responses
- **RAG & Semantic Search** - Document stores with automatic indexing, advanced search modes (hybrid, multi-query, HyDE), LLM-based re-ranking, multiple embedder support (Ollama, OpenAI, Cohere)
- **LLM Providers** - OpenAI (GPT-4o, GPT-4o-mini), Anthropic (Claude Sonnet 4, Opus 4), Google Gemini (Gemini 2.0 Flash), Ollama (qwen3), custom providers via gRPC plugins
- **A2A Protocol** - 100% v0.3.0 compliant, agent discovery, standardized messaging and streaming, federation support

## Documentation

Complete documentation, guides, and examples available at **[gohector.dev](https://gohector.dev)**:

- [Getting Started](https://gohector.dev/getting-started/)
- [Configuration Reference](https://gohector.dev/reference/configuration/)
- [Production Deployment](https://gohector.dev/how-to/deploy-production/)
- [API Reference](https://gohector.dev/reference/api/)
- [Core Concepts](https://gohector.dev/core-concepts/)

## License

AGPL-3.0 License. See [LICENSE.md](LICENSE.md) for details.
