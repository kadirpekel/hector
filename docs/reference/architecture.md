# Architecture Reference

Complete reference for Hector's internal architecture, design decisions, and system components.

---

## Design Principles

### Config-First Philosophy

Hector separates **operational** and **functional** configuration:

| Concern | Source | Examples |
|---------|--------|----------|
| **Server (Operational)** | CLI flags + environment variables | Database DSN, port, auth secret, logging, queue, rate limiting |
| **App (Functional)** | YAML file or Admin API | Agents, LLMs, tools, guardrails, RAG pipelines |

This separation enables:

- 12-factor app compliance (no secrets in config files)
- Hot-reloadable app configuration (via `--watch` or Admin API)
- Multi-tenant deployments (multiple app configs in one server)
- One YAML file describes the entire agent application

### Config Lifecycle

```
┌──────────┐     ┌──────────────────┐     ┌────────────────┐     ┌───────────┐
│ CLI args │     │ EnsureConfigExists│     │LoadAppConfigWith│     │  Runtime  │
│ + env    │────►│ (auto-generate   │────►│Lean (YAML→full │────►│  Builder  │
│          │     │  if missing)     │     │ config + lean   │     │           │
└──────────┘     └──────────────────┘     │ JSON)           │     └───────────┘
                                          └───────┬────────┘
                                                  │ lean JSON
                                                  ▼
                                          ┌───────────────┐
                                          │   Database     │
                                          │ (apps table)   │
                                          │ Stores lean    │
                                          │ JSON only      │
                                          └───────┬────────┘
                                                  │ ParseAppConfigJSON
                                                  ▼
                                          ┌───────────────┐
                                          │ Studio / API  │
                                          │ Reads & writes│
                                          │ lean JSON     │
                                          └───────────────┘
```

**Key design decisions:**

- **Lean JSON**: The database stores only user-specified fields (no defaults). This ensures Studio shows exactly what the user configured, not hundreds of default values.
- **Defaults applied at runtime**: `SetDefaults()` runs when loading from YAML *and* when loading from DB, never when storing.
- **Seed, don't sync**: On startup the config file seeds the default app into the DB only if it doesn't exist yet. Once the app is in the DB, the DB owns it — Studio and API changes survive restarts.
- **Explicit sync**: Use `--sync` to force the config file to overwrite the DB on startup. Use `--watch` to live-sync file changes to DB during development.
- **Single file read**: `LoadAppConfigWithLean()` reads the YAML file once, captures lean JSON before defaults, then applies defaults — avoiding double reads.

---

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              HTTP Server                                │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  ┌──────────────┐  │
│  │   A2A API   │  │  Admin API   │  │  Webhooks   │  │   Health     │  │
│  │  /agents/*  │  │  /admin/*    │  │ /webhooks/* │  │  /health     │  │
│  └──────┬──────┘  └──────┬───────┘  └──────┬──────┘  └──────────────┘  │
└─────────┼────────────────┼─────────────────┼───────────────────────────┘
          │                │                 │
          ▼                ▼                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            App Manager                                  │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  ┌──────────────┐  │
│  │  App Store  │  │   Runtime    │  │   Config    │  │    Cache     │  │
│  │   (SQL)     │  │   Factory    │  │   Loader    │  │  (Runtimes)  │  │
│  └─────────────┘  └──────────────┘  └─────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                              Runtime                                    │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  ┌──────────────┐  │
│  │   Agents    │  │    Tools     │  │   Session   │  │  Guardrails  │  │
│  │  Registry   │  │   Factory    │  │   Service   │  │   Chain      │  │
│  └─────────────┘  └──────────────┘  └─────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Persistence Layer                              │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐  ┌──────────────┐  │
│  │  Sessions   │  │    Tasks     │  │ Checkpoints │  │   Vectors    │  │
│  │   (SQL)     │  │   (Queue)    │  │   (SQL)     │  │  (chromem)   │  │
│  └─────────────┘  └──────────────┘  └─────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Core Components

### App Manager

Manages multi-tenant app configurations and runtime caching.

**Responsibilities:**

- Load and cache app configurations from database
- Create and cache agent runtimes per app
- Handle hot-reload on configuration changes
- Provide runtime lookups for request handling

**Location:** `pkg/server/app_manager.go`

---

### Runtime

Orchestrates all components for a single app.

**Responsibilities:**

- Build agent tree from configuration
- Initialize LLM providers and tools
- Manage session and checkpoint services
- Wire guardrails chain

**Location:** `pkg/runtime/`

---

### Builder

Fluent API for constructing agents and runtimes.

**Responsibilities:**

- LLM provider instantiation (Anthropic, OpenAI, Gemini, Ollama)
- Tool registration (MCP, function, agent-as-tool)
- Guardrails chain construction
- Configuration validation

**Location:** `pkg/builder/`

---

## Agent System

### Agent Types

| Type | Implementation | Description |
|------|----------------|-------------|
| `llm` | `pkg/agent/llmagent` | Standard LLM-backed agent |
| `sequential` | `pkg/agent/workflowagent` | Runs sub-agents in order |
| `parallel` | `pkg/agent/workflowagent` | Runs sub-agents concurrently |
| `loop` | `pkg/agent/workflowagent` | Iterates until condition |
| `conditional` | `pkg/agent/workflowagent` | Routes based on evaluation |
| `remote` | `pkg/agent/remoteagent` | Proxies to A2A endpoint |

### Agent Interface

```go
type Agent interface {
    Name() string
    Description() string
    Run(ctx Context, input Content) iter.Seq2[*Event, error]
    SubAgents() []Agent
    Tools() []tool.Tool
}
```

### Multi-Agent Patterns

**Pattern 1: Transfer (Sub-agents)**

- Parent creates `transfer_to_{name}` tools automatically
- Control transfers completely to sub-agent
- Sub-agent runs independently

**Pattern 2: Delegation (Agent-as-tool)**

- Parent calls sub-agent as a tool
- Parent maintains control
- Receives structured result

---

## Session System

### Event-Sourced State

Sessions store events, not snapshots. State is derived by replaying events.

**Event Types:**

| Type | Description |
|------|-------------|
| `user_input` | User message |
| `agent_response` | Agent text output |
| `tool_call` | Tool invocation |
| `tool_result` | Tool response |
| `state_update` | Session state change |

### State Scopes

| Scope | Prefix | Lifetime |
|-------|--------|----------|
| `app` | `app:` | Persists across sessions |
| `user` | `user:` | Persists for user |
| `temp` | `temp:` | Current session only |

**Example:**

```yaml
# In instruction template
instruction: |
  Hello {user:name}, working on {app:project}.
  Last summary: {temp:summary?}
```

---

## Task Queue

Durable background execution with retry and recovery.

### States

```
┌──────────┐    Enqueue    ┌─────────┐    Start    ┌─────────┐
│ Pending  │ ──────────► │ Running │ ──────────► │Complete │
└──────────┘              └─────────┘              └─────────┘
     │                         │
     │                         │ Fail
     │                         ▼
     │                   ┌─────────┐    Retry    ┌─────────┐
     │                   │ Failed  │ ──────────► │   DLQ   │
     │                   └─────────┘              └─────────┘
     │                         ▲
     └─────────────────────────┘
           Stale Recovery
```

### Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `workers` | `4` | Concurrent workers |
| `max_retries` | `3` | Retry attempts |
| `initial_delay` | `1s` | First retry delay |
| `max_delay` | `5m` | Maximum backoff |
| `stale_threshold` | `5m` | Zombie recovery time |

**Location:** `pkg/execution/native/`

---

## Checkpoint System

Incremental progress snapshots for long-running operations (primarily RAG indexing).

### Checkpoint Data

| Field | Description |
|-------|-------------|
| `app_name` | App identifier |
| `store_name` | Document store name |
| `file_path` | Indexed file |
| `file_hash` | Content hash |
| `indexed_at` | Timestamp |

### Recovery Flow

1. On startup, load existing checkpoints
2. Compare file hashes with stored values
3. Skip unchanged files (incremental indexing)
4. Update checkpoints after successful indexing

**Location:** `pkg/checkpoint/`

---

## Tool System

### Tool Interface

```go
type Tool interface {
    Name() string
    Description() string
    Schema() *jsonschema.Schema
}

type CallableTool interface {
    Tool
    Call(ctx Context, args json.RawMessage) (*Result, error)
}
```

### Tool Types

| Type | Package | Description |
|------|---------|-------------|
| Function | `pkg/tool/functiontool` | Custom Go functions |
| MCP | `pkg/tool/mcptoolset` | Model Context Protocol |
| Agent | `pkg/tool/agenttool` | Agent-as-tool wrapper |
| File | `pkg/tool/filetool` | File system operations |
| Web | `pkg/tool/webtool` | HTTP requests |
| Command | `pkg/tool/commandtool` | Shell execution |
| Memory | `pkg/tool/memorytool` | Cross-session memory |
| Search | `pkg/tool/searchtool` | RAG document search |
| Approval | `pkg/tool/approvaltool` | Human-in-the-loop |
| Control | `pkg/tool/controltool` | Flow control (exit, escalate) |

### MCP Transports

| Transport | Config | Use Case |
|-----------|--------|----------|
| `stdio` | `command`, `args` | Local process |
| `sse` | `url` | Server-Sent Events |
| `streamable-http` | `url` | HTTP streaming |

---

## Guardrails System

Chain-of-responsibility pattern for input/output validation.

### Chain Order

```
Input → [Length] → [Injection] → [Sanitizer] → [Pattern] → Agent
                                                            ↓
Output ← [PII] ← [Content] ← [Moderation] ← Agent Response
```

### Guardrail Types

| Category | Type | Description |
|----------|------|-------------|
| **Input** | `length` | Token/character limits |
| | `injection` | Prompt injection detection |
| | `sanitizer` | HTML/whitespace cleanup |
| | `pattern` | Regex pattern matching |
| **Output** | `pii` | Personal data redaction |
| | `content` | Keyword/pattern blocking |
| **Moderation** | `openai` | OpenAI moderation API |
| | `lakera` | Lakera Guard API |
| | `prompt` | LLM-based moderation |
| **Tool** | `authorization` | Tool call restrictions |

**Location:** `pkg/guardrails/`

---

## RAG Pipeline

Enterprise-grade retrieval-augmented generation.

### Pipeline Flow

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Source  │ ─► │ Parser  │ ─► │ Chunker │ ─► │Embedder │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
                                                  │
                                                  ▼
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Context │ ◄─ │ Rerank  │ ◄─ │ Search  │ ◄─ │ Vector  │
└─────────┘    └─────────┘    └─────────┘    │  Store  │
                                              └─────────┘
```

### Components

| Component | Options |
|-----------|---------|
| **Sources** | Directory, SQL, API, Blob |
| **Parsers** | Native (PDF, DOCX, Excel), MCP |
| **Chunkers** | Simple (size), Semantic |
| **Embedders** | OpenAI, Gemini, Ollama |
| **Vector Stores** | chromem (embedded), Qdrant, Pinecone, Weaviate, Milvus |
| **Search** | Similarity, HyDE, Multi-query |
| **Reranking** | LLM-based |

**Location:** `pkg/rag/`

---

## Authentication

### Auth Flow

```
Request → [Extract Token] → [Validate] → [Extract Claims] → [Set Context]
```

### Supported Methods

| Method | Configuration |
|--------|---------------|
| Admin Secret | `--auth-secret` |
| JWT/JWKS | `--auth-jwks-url`, `--auth-issuer` |

### JWT Claims

| Claim | Description |
|-------|-------------|
| `sub` | Subject (app or user ID) |
| `tenant_id` | App ID for multi-tenancy |
| `iss` | Issuer |
| `aud` | Audience |

**Location:** `pkg/auth/`

---

## Observability

### Metrics (Prometheus)

| Metric | Type | Description |
|--------|------|-------------|
| `hector_requests_total` | Counter | Total HTTP requests |
| `hector_request_duration_seconds` | Histogram | Request latency |
| `hector_agent_invocations_total` | Counter | Agent calls |
| `hector_tool_calls_total` | Counter | Tool invocations |
| `hector_llm_tokens_total` | Counter | Token usage |
| `hector_scheduler_triggers_total` | Counter | Scheduled triggers |
| `hector_notifications_total` | Counter | Outbound notifications |

### Tracing (OpenTelemetry)

Spans for:

- HTTP request handling
- Agent execution
- Tool calls
- LLM API calls
- Database operations

**Location:** `pkg/observability/`

---

## Trigger System

### Scheduled Triggers

Cron-based agent invocation with timezone support.

```yaml
trigger:
  type: schedule
  cron: "0 9 * * *"
  timezone: America/New_York
  input: "Generate daily report"
```

**Location:** `pkg/trigger/scheduler.go`

### Webhook Triggers

HTTP-triggered agent invocation with payload transformation.

```yaml
trigger:
  type: webhook
  path: /webhooks/github
  secret: ${WEBHOOK_SECRET}
  webhook_input:
    template: "PR #{{.number}}: {{.title}}"
```

**Location:** `pkg/trigger/webhook.go`

---

## Notification System

Outbound webhooks on agent events.

### Events

| Event | Description |
|-------|-------------|
| `task_started` | Agent began processing |
| `task_completed` | Agent finished successfully |
| `task_failed` | Agent encountered error |

### Features

- Retry with exponential backoff
- Custom payload templates
- Header configuration
- A2A-compliant push notifications

**Location:** `pkg/notification/`

---

## Database Schema

### Core Tables

| Table | Description |
|-------|-------------|
| `apps` | App configurations (multi-tenant) |
| `sessions` | Session metadata |
| `session_events` | Event-sourced session state |
| `tasks` | Task queue |
| `checkpoints` | RAG indexing progress |

### Supported Databases

| Database | DSN Format |
|----------|------------|
| SQLite | `sqlite://.hector/hector.db` |
| PostgreSQL | `postgres://user:pass@host:port/db` |
| MySQL | `mysql://user:pass@tcp(host:port)/db` |

---

## Package Structure

```
pkg/
├── api.go              # Public programmatic API
├── agent/              # Agent implementations
│   ├── llmagent/       # LLM-backed agents
│   ├── workflowagent/  # Sequential, Parallel, Loop
│   └── remoteagent/    # Remote A2A agents
├── auth/               # Authentication
├── builder/            # Fluent construction API
├── checkpoint/         # Progress snapshots
├── config/             # Configuration structs
├── embedder/           # Embedding providers
├── execution/          # Task execution
├── guardrails/         # Input/output validation
├── instruction/        # Template resolution
├── memory/             # Persistent memory
├── model/              # LLM providers
│   ├── anthropic/
│   ├── openai/
│   ├── gemini/
│   └── ollama/
├── notification/       # Outbound webhooks
├── observability/      # Metrics and tracing
├── rag/                # Document retrieval
├── ratelimit/          # Request throttling
├── runtime/            # Orchestration
├── server/             # HTTP server
├── session/            # State management
├── task/               # Task queue
├── tool/               # Tool implementations
├── trigger/            # Scheduled/webhook triggers
└── vector/             # Vector store providers
```
