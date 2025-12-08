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

**Config-first A2A-Native Agent Platform**

Deploy observable, secure, and scalable AI agents with zero-config or YAML, plus a programmatic API.

**[📚 Documentation](https://gohector.dev)** | [CLI Reference](https://gohector.dev/reference/cli/) | [Config Reference](https://gohector.dev/reference/configuration/)

---

## Quick Start (zero-config)

```bash
go install github.com/kadirpekel/hector/cmd/hector@latest
export OPENAI_API_KEY="sk-..."

hector serve --model gpt-4o --tools
```

RAG in one command (with MCP parsing optional):
```bash
hector serve \
  --model gpt-4o \
  --docs-folder ./documents \
  --mcp-url http://localhost:8000/mcp \
  --mcp-parser-tool convert_document_into_docling_document
```

## Quick Start (config file)

```bash
cat > config.yaml <<'EOF'
version: "2"
llms:
  default:
    provider: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}
agents:
  assistant:
    llm: default
    tools: [search]
server:
  port: 8080
EOF

hector serve --config config.yaml
```

## Highlights
- **Config-first & zero-config**: YAML for repeatability; flags for fast starts. JSON Schema available via `hector schema`.
- **Programmatic API**: Build agents in Go (`pkg/api.go`), including sub-agents and agent-as-tool patterns.
- **RAG & MCP**: Folder-based document stores, embedded vector search (chromem), optional MCP parsing chain.
- **Persistence**: Tasks and sessions can use in-memory or SQL backends (sqlite/postgres/mysql via DSN).
- **Observability**: Metrics endpoint and OTLP tracing options.
- **Checkpointing**: Optional checkpoint/recovery strategies.
- **Auth**: JWT/JWKS support at the server layer.
- **A2A-native**: Uses a2a-go types and JSON-RPC/gRPC endpoints.

## Documentation
- [Getting Started](https://gohector.dev/getting-started/)
- [Configuration Reference](https://gohector.dev/reference/configuration/)
- [CLI Reference](https://gohector.dev/reference/cli/)
- [Core Concepts](https://gohector.dev/core-concepts/)

## License

AGPL-3.0 (see [LICENSE](LICENSE)).
