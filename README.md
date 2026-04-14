```
██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```

[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![A2A Protocol](https://img.shields.io/badge/A2A%20v0.3.0-100%25%20compliant-brightgreen.svg)](https://a2a-protocol.org/latest/community/#a2a-integrations)
[![Documentation](https://img.shields.io/badge/docs-gohector.dev-blue.svg)](https://gohector.dev)
[![Go Report Card](https://goreportcard.com/badge/github.com/verikod/hector)](https://goreportcard.com/report/github.com/verikod/hector)

**Your agents. Your infrastructure. Your rules.**

Hector is an open-source AI agent runtime built for teams that need full control over their AI infrastructure. One self-contained binary, one YAML config, production-ready defaults. Deploy on-premise, in air-gapped environments, or in any cloud. No external dependencies, no runtime interpreters, no mandatory cloud accounts.

![Hector Studio](docs/assets/ss.png)

**[Documentation](https://gohector.dev)** · **[Studio](https://studio.gohector.dev)** · **[Config Ref](https://gohector.dev/reference/configuration/)** · **[CLI Ref](https://gohector.dev/reference/cli/)** · **[API Ref](https://gohector.dev/reference/api/)**

---

## Why Hector?

- **Single Binary. Zero Dependencies.** A ~30MB Go executable with the visual Studio UI baked in. Copy it to a server and run. No interpreters, no virtualenvs, no package managers. Works out of the box on any Linux, macOS, or Windows host
- **Self-Sovereign by Design**: Deploy on-premise, in air-gapped networks, or behind your firewall. LLM keys stay on your machines, data never transits third-party infrastructure, and there is zero telemetry. MIT licensed, fully auditable
- **Open Standards, Zero Lock-In**: Built entirely on A2A protocol for agent interop and MCP for tool connectivity. Swap LLM providers (Anthropic, OpenAI, Gemini, Ollama) with a one-line config change. Your investment is portable
- **Config-Driven Operations**: Agents, orchestration patterns, RAG pipelines, guardrails, triggers, and notifications. All defined in YAML, all version-controllable, all reviewable in CI. Or use the fluent Go API for full programmatic control
- **SQL-Backed Durability**: Sessions, tasks, and checkpoints persist to SQLite or PostgreSQL. Survives restarts, enables fault recovery, and powers human-in-the-loop workflows with built-in state management
- **Enterprise Security & Multi-Tenancy**: JWKS/OIDC authentication, prompt injection detection, PII redaction, tool sandboxing, rate limiting, agent visibility controls, and full tenant isolation. All declarative, all without writing code

---

## Quick Start

```bash
# Install
go install github.com/verikod/hector/cmd/hector@latest

# Initialize config
hector init --provider anthropic --model claude-sonnet-4

# Set API key
export ANTHROPIC_API_KEY="your-key"

# Run server
hector serve
```

Open **http://localhost:8080/** in your browser for the visual Studio interface, or use the API directly at `/agents/`.

---

## Hector Studio

[Hector Studio](https://studio.gohector.dev) is the web UI for designing, testing, and managing agents.

> **[Try it online →](https://studio.gohector.dev)** — no install required. Connect to any Hector server.

- **Visual Flow Builder**: Drag-and-drop canvas with bi-directional YAML sync
- **Integrated Chat**: Streaming responses with tool-call trace view
- **Resource Management**: Configure LLMs, tools, and guardrails without editing YAML
- **Cloud Integration**: Provision and manage cloud-hosted instances with a license key

Studio is also embedded in release builds and served at `/`. For development builds:

```bash
# Option 1: Build a single binary with UI embedded (requires Node.js)
make build-release

# Option 2: Run Studio dev server (hot-reload for UI development)
cd studio && npm run dev   # → http://localhost:5173
```

See the [Studio Guide](https://gohector.dev/guides/studio/) for details.

---

## Core Architecture

### SQL-Backed Foundation

All runtime state persists to SQL (SQLite or PostgreSQL):

| Component | Persistence |
|-----------|-------------|
| **Sessions** | Event-sourced state with scoped variables (app/user/temp) |
| **Tasks** | Durable queue with retry, exponential backoff, and recovery |
| **Checkpoints** | Incremental progress snapshots for long-running operations |
| **App Configs** | Hot-reloadable YAML with database sync |

```bash
# SQLite (default)
hector serve --database sqlite://.hector/hector.db

# PostgreSQL
hector serve --database postgres://user:pass@localhost/hector
```

### Multi-Tenant Architecture

A single Hector deployment can host multiple isolated apps via the **Admin API** (not file-based configs). Each app has:

- Isolated agent configurations
- Separate session and state storage
- Independent vector store collections
- Per-app JWT authentication

---

## Admin API

Manage apps, sessions, and queue via REST endpoints. Requires `--auth-secret` for authentication.

```bash
hector serve --auth-secret "admin-secret"
```

### App Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/apps` | GET | List all apps |
| `/admin/apps` | POST | Create new app (returns JWT) |
| `/admin/apps/{id}` | GET | Get app details |
| `/admin/apps/{id}` | PUT | Update app config |
| `/admin/apps/{id}` | DELETE | Delete app and all resources |
| `/admin/apps/{id}/token` | POST | Regenerate app JWT |

**Create App:**
```bash
curl -X POST http://localhost:8080/admin/apps \
  -H "Authorization: Bearer admin-secret" \
  -H "Content-Type: application/json" \
  -d '{"name": "customer-support", "config_json": "{...}"}'

# Response includes JWT for app authentication
{
  "app": {"id": "abc123", "name": "customer-support"},
  "access_token": "eyJhbGc...",
  "token_type": "bearer"
}
```

### Session Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/sessions` | GET | List sessions (paginated) |
| `/admin/sessions/{id}` | GET | Get session with events |
| `/admin/sessions/{id}` | DELETE | Delete session |

### Queue Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/queue/stats` | GET | Queue statistics |
| `/admin/queue/dlq` | GET | List dead-letter queue items |
| `/admin/queue/dlq/{id}/requeue` | POST | Requeue failed task |

### JWKS Endpoint

```bash
# Public key for JWT verification
curl http://localhost:8080/admin/jwks
```

---

## Agent Capabilities

### Agent Types

| Type | Description |
|------|-------------|
| `llm` | Standard LLM-backed agent (default) |
| `sequential` | Runs sub-agents in order |
| `parallel` | Runs sub-agents concurrently |
| `loop` | Iterates until condition met |
| `conditional` | Routes based on evaluation |
| `remote` | Proxies to external A2A server |

### Multi-Agent Patterns

```yaml
agents:
  # Agent-as-tool: Call other agents like functions
  orchestrator:
    llm: default
    agent_tools: [researcher, writer]

  # Sub-agent delegation: Transfer control
  manager:
    llm: default
    sub_agents: [analyst, reporter]

  # Conditional routing
  router:
    type: conditional
    condition_agent: classifier
    condition_field: category
    on_true_agent: sales
    on_false_response: Redirecting to support...
```

### Instruction Templating

Dynamic instruction resolution from session state:

```yaml
agents:
  assistant:
    instruction: |
      Hello {user:name}, you're working on {app:project}.
      
      Context: {artifact.context.md}
      
      Previous summary: {temp:last_summary?}
```

| Syntax | Resolution |
|--------|------------|
| `{variable}` | Session state |
| `{app:variable}` | App-scoped state |
| `{user:variable}` | User-scoped state |
| `{temp:variable}` | Temporary state |
| `{artifact.file}` | Artifact content |
| `{variable?}` | Optional (empty if missing) |

---

## Automation & Integration

### Scheduled Triggers

Run agents on cron schedules with timezone support:

```yaml
agents:
  daily_report:
    llm: default
    instruction: Generate the daily summary report.
    trigger:
      type: schedule
      cron: "0 9 * * *"  # Daily at 9am
      timezone: America/New_York
      input: "Generate report for {{.Date}}"
```

### Webhook Triggers

Invoke agents via HTTP with payload transformation:

```yaml
agents:
  github_handler:
    llm: default
    trigger:
      type: webhook
      path: /webhooks/github
      methods: [POST]
      secret: ${WEBHOOK_SECRET}
      signature_header: X-Hub-Signature-256
      webhook_input:
        template: "PR #{{.number}} by {{.user.login}}: {{.title}}"
        session_id: "pr-{{.number}}"
        extract_fields:
          - path: pull_request.title
            as: title
      response:
        mode: async  # sync | async | callback
```

### Outbound Notifications

A2A-compliant push notifications on agent events:

```yaml
agents:
  processor:
    llm: default
    notifications:
      - id: slack-notify
        url: https://hooks.slack.com/services/xxx
        events: [task_completed, task_failed]
        headers:
          Content-Type: application/json
        payload:
          template: |
            {"text": "Agent {{.AgentName}}: {{.Status}}"}
        retry:
          max_attempts: 3
          initial_delay: 1s
          max_delay: 30s
```

---

## Tool Ecosystem

### MCP Protocol Support

Native Model Context Protocol with all transports:

```yaml
tools:
  filesystem:
    type: mcp
    transport: stdio
    command: npx
    args: [-y, "@modelcontextprotocol/server-filesystem", "./data"]
    filter: [read_file, write_file]  # Limit exposed tools

  remote_mcp:
    type: mcp
    transport: sse
    url: http://localhost:8000/mcp
```

### Built-in Tools

| Tool | Description |
|------|-------------|
| `filetool` | File system operations with sandboxing |
| `webtool` | HTTP requests, web scraping |
| `commandtool` | Shell execution with allowlists |
| `memorytool` | Cross-session memory persistence |
| `searchtool` | RAG document search |
| `todotool` | Task/checklist management |
| `approvaltool` | Human-in-the-loop approvals |
| `agenttool` | Agent-as-callable-tool |

### Command Tool with Security

```yaml
tools:
  shell:
    type: command
    allowed_commands: [ls, cat, grep, find]
    denied_commands: [rm, sudo]
    working_directory: /app/workspace
    max_execution_time: 30s
    require_approval: true
    approval_prompt: "Allow command execution?"
```

---

## RAG Pipeline

Enterprise-grade retrieval-augmented generation:

### Document Sources

| Source | Description |
|--------|-------------|
| `directory` | File system with glob patterns |
| `sql` | Database tables with incremental sync |
| `api` | REST endpoints |
| `blob` | Cloud storage (S3, GCS, Azure) |

### Advanced Retrieval

```yaml
document_stores:
  knowledge:
    source:
      type: directory
      include: ["**/*.md", "**/*.pdf"]
      exclude: ["**/node_modules/**"]
    
    chunking:
      strategy: semantic
      size: 1000
      overlap: 200
    
    search:
      top_k: 10
      threshold: 0.7
      enable_hyde: true      # Hypothetical document embeddings
      enable_rerank: true    # LLM reranking
      enable_multi_query: true  # Query expansion
    
    vector_store: default
    embedder: default
    watch: true  # Real-time indexing
    incremental_indexing: true
```

### Vector Stores

| Provider | Type |
|----------|------|
| chromem | Embedded (default) |
| Qdrant | External |
| Pinecone | Cloud |
| Weaviate | External |
| Milvus | External |

---

## Guardrails

Deterministic and LLM-powered content protection:

### Input Guardrails

```yaml
guardrails:
  default:
    input:
      length:
        max_length: 50000
        action: block
      injection:
        enabled: true
        patterns: ["ignore previous", "system:"]
        action: block
        severity: critical
      sanitizer:
        trim_whitespace: true
        strip_html: true
```

### Output Guardrails

```yaml
guardrails:
  default:
    output:
      pii:
        enabled: true
        detect_email: true
        detect_phone: true
        detect_ssn: true
        detect_credit_card: true
        redact_mode: mask  # mask | remove | hash
      content:
        blocked_keywords: [profanity_list]
        blocked_patterns: ["(?i)password.*=.*"]
```

### LLM Moderation

```yaml
guardrails:
  default:
    moderation:
      enabled: true
      strategy: openai  # openai | lakera | prompt
      openai:
        model: omni-moderation-latest
        threshold: 0.8
      action: block
```

---

## Authentication

### JWKS/OIDC Integration

```bash
hector serve \
  --auth-jwks-url "https://auth.example.com/.well-known/jwks.json" \
  --auth-issuer "https://auth.example.com/" \
  --auth-audience "hector-api"
```

### Simple Token Auth

```bash
hector serve --auth-secret "your-admin-secret"
```

---

## Observability

### Prometheus Metrics

```bash
hector serve --metrics
# Metrics available at /metrics
```

### OpenTelemetry Tracing

```bash
hector serve --tracing-endpoint "jaeger:4317"
```

---

## LLM Providers

| Provider | Features |
|----------|----------|
| **Anthropic** | Claude 4, streaming, thinking blocks, multimodal |
| **OpenAI** | GPT-4o, structured output, reasoning |
| **Gemini** | Multimodal (audio, image, video) |
| **Ollama** | Local inference, any open model |

```yaml
llms:
  claude:
    provider: anthropic
    model: claude-sonnet-4
    api_key: ${ANTHROPIC_API_KEY}
    thinking:
      enabled: true
      budget_tokens: 4096
  
  gpt4:
    provider: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}
    
  local:
    provider: ollama
    model: llama3.2
    base_url: http://localhost:11434
```

---

## Deployment

### Single Binary

```bash
# Build
go build -o hector ./cmd/hector

# Run
./hector serve --config config.yaml
```

### Docker

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o hector ./cmd/hector

FROM alpine:latest
COPY --from=builder /app/hector /usr/local/bin/
EXPOSE 8080
CMD ["hector", "serve"]
```

### Configuration Validation

```bash
# Validate before deploy
hector validate --config production.yaml

# JSON Schema available
curl http://localhost:8080/schema
```

---

## Documentation

| Resource | Description |
|----------|-------------|
| [Quick Start](https://gohector.dev/getting-started/quick-start/) | Get running in 5 minutes |
| [Hector Studio](https://gohector.dev/guides/studio/) | Web UI for agent development |
| [Configuration Reference](https://gohector.dev/reference/configuration/) | Complete YAML schema |
| [CLI Reference](https://gohector.dev/reference/cli/) | All commands and flags |
| [API Reference](https://gohector.dev/reference/api/) | HTTP API & Admin endpoints |
| [Go API Reference](https://gohector.dev/reference/go-api/) | Programmatic API |
| [Architecture](https://gohector.dev/reference/architecture/) | System design & internals |

---

## License

MIT (see [LICENSE](LICENSE)).
