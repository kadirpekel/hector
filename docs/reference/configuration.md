---
title: Configuration Reference
description: Hector YAML configuration (matches pkg/config)
---

# Configuration Reference

This reflects the current `pkg/config` schema. For the authoritative structure, run `hector schema` (JSON Schema) or `hector validate --print-config` on a sample file.

## Minimal Example
```yaml
version: "2"
name: my-assistant

llms:
  default:
    provider: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}

tools:
  search:
    type: builtin

agents:
  assistant:
    llm: default
    tools: [search]
    instruction: You are a helpful assistant.

server:
  port: 8080
```

## Top-Level Fields
- `version` (string): schema version; defaults to `2`.
- `name`, `description` (string): metadata.
- `databases` (map[string]DatabaseConfig): reusable DB connections.
- `vector_stores` (map[string]VectorStoreConfig): vector DB providers.
- `llms` (map[string]LLMConfig): LLM providers/models.
- `embedders` (map[string]EmbedderConfig): embedding providers.
- `tools` (map[string]ToolConfig): builtin/MCP/etc. tools.
- `agents` (map[string]AgentConfig): agent definitions.
- `document_stores` (map[string]DocumentStoreConfig): RAG sources.
- `server` (ServerConfig): A2A server, tasks, sessions, memory, checkpoint, observability, auth, TLS, CORS.
- `logger` (LoggerConfig): logging settings.
- `rate_limiting` (RateLimitConfig): rate-limit rules/backends.
- `defaults` (DefaultsConfig): defaults applied to agents (e.g., default llm).

## Selected Sections (concise)

### LLMs
```yaml
llms:
  default:
    provider: openai | anthropic | gemini | ollama
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}
    base_url: https://api.openai.com/v1   # optional
    temperature: 0.7                      # optional
    max_tokens: 4096                      # optional
```

### Tools
```yaml
tools:
  search:
    type: builtin
  weather:
    type: mcp
    url: ${MCP_URL}
```

### Agents
```yaml
agents:
  assistant:
    llm: default
    tools: [search, weather]
    instruction: You are a helpful assistant.
    role: helper
    input_modes: [text]
    output_modes: [text]
    reasoning:
      max_iterations: 32
      allow_thinking: true
    memory:
      working:
        strategy: buffer_window
        window_size: 12
```

### Document Stores
```yaml
document_stores:
  docs:
    source:
      type: folder
      path: ./documents
      watch: true
    embedder: default-embedder
    vector_store: default-vs
```

### Server
```yaml
server:
  host: 0.0.0.0
  port: 8080
  transport: json-rpc           # json-rpc (default) or grpc
  grpc_port: 50051              # used when transport=grpc
  tls: {}                       # optional TLSConfig
  cors: {}                      # optional CORSConfig
  auth: {}                      # optional AuthConfig
  tasks:
    backend: inmemory | sql
    database: default-db        # required if backend=sql (refers to databases)
  sessions:
    backend: inmemory | sql
    database: default-db
  memory:
    backend: keyword | vector   # vector requires embedder + vector_provider
    embedder: default-embedder
    vector_provider:
      type: chromem             # default embedded provider; others rejected if not implemented
      chromem:
        persist_path: .hector/chromem
        compress: false
  checkpoint:
    enabled: false
    strategy: event | interval | hybrid
    interval: 0
    after_tools: false
    before_llm: false
    recovery:
      auto_resume: false
      auto_resume_hitl: false
      timeout: 3600
  observability:
    tracing:
      enabled: false
      exporter: otlp
      endpoint: localhost:4317
    metrics:
      enabled: true
```

### Databases
```yaml
databases:
  default-db:
    driver: sqlite | postgres | mysql
    database: .hector/hector.db   # or DSN/URL
```

### Vector Stores & Embedders
```yaml
vector_stores:
  default-vs:
    provider: chroma | qdrant | weaviate | pinecone | milvus | chromem
    url: http://localhost:8000
    collection: docs

embedders:
  default-embedder:
    provider: openai | other
    model: text-embedding-3-small
    api_key: ${OPENAI_API_KEY}
```

### Logger
```yaml
logger:
  level: info
  format: simple
  file: ""    # empty = stderr
```

### Rate Limiting
```yaml
rate_limiting:
  enabled: false
  rules:
    - name: per-user
      scope: user
      window: 1m
      max_requests: 60
```

### Defaults
```yaml
defaults:
  llm: default
```

## Validation and Defaults
- Defaults are applied automatically by `config.SetDefaults()` during `hector serve` and `hector validate`.
- Run `hector validate --print-config` to see the fully expanded config with defaults and env vars resolved.
- The loader rejects unimplemented vector providers (e.g., qdrant/milvus) today; use `chromem` (embedded) unless you add the implementation.
