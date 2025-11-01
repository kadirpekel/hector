---
title: Configuration Reference
description: Complete YAML configuration reference for Hector
---

# Configuration Reference

Complete reference for all Hector YAML configuration options.

---

## Configuration File Structure

```yaml
# Global Settings
global:
  a2a_server:     # A2A server configuration
  auth:           # Global authentication
  observability:  # Observability (metrics & tracing)

# Providers
llms:             # LLM providers
databases:        # Vector databases
embedders:        # Embedding models
plugins:          # gRPC plugins

# Session Storage
session_stores:   # Session persistence stores

# Agents
agents:           # Agent definitions

# Tools
tools:            # Tool configurations

# Document Stores
document_stores:  # RAG document sources

# Logging
logging:          # Logging configuration
```

---

## Global Configuration

### global.a2a_server

A2A server settings (when running `hector serve`):

```yaml
global:
  a2a_server:
    host: "0.0.0.0"              # Bind host (default: 0.0.0.0)
    port: 8080                   # gRPC port (default: 8080)
    rest_gateway_port: 8080      # REST API port (default: 8080)
    jsonrpc_port: 8080           # JSON-RPC port (default: 8080)
    base_url: "http://localhost:8080"  # External URL
    
    tls:
      enabled: false
      cert_file: "/path/to/cert.pem"
      key_file: "/path/to/key.pem"
```

### global.auth

Global JWT authentication:

```yaml
global:
  auth:
    jwks_url: "https://provider.com/.well-known/jwks.json"
    issuer: "https://provider.com"
    audience: "hector-api"
    leeway: 60                   # Clock skew tolerance (seconds)
    cache_duration: "15m"        # JWKS cache duration
```

### global.observability

Observability configuration for metrics and distributed tracing:

```yaml
global:
  observability:
    tracing:
      enabled: true              # Enable distributed tracing
      exporter_type: "jaeger"    # Exporter type (jaeger, otlp)
      endpoint_url: "localhost:4317"  # OTLP gRPC endpoint
      sampling_rate: 1.0         # Sampling rate (0.0-1.0)
      service_name: "hector"     # Service name in traces
    
    metrics_enabled: true        # Enable Prometheus metrics
```

**Options:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `tracing.enabled` | boolean | `false` | Enable OpenTelemetry tracing |
| `tracing.exporter_type` | string | `"otlp"` | Trace exporter (jaeger, otlp) |
| `tracing.endpoint_url` | string | `"localhost:4317"` | OTLP gRPC endpoint |
| `tracing.sampling_rate` | float | `1.0` | Trace sampling rate (0.0-1.0) |
| `tracing.service_name` | string | `"hector"` | Service name for traces |
| `metrics_enabled` | boolean | `false` | Enable Prometheus metrics |

**Note:** Metrics are served on the HTTP server at `/metrics` endpoint. All tracing defaults are applied automatically via `SetDefaults()`.

See [Observability](../core-concepts/observability.md) for detailed usage.

---

## LLM Providers

### OpenAI

```yaml
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"              # Default: gpt-4o
    api_key: "${OPENAI_API_KEY}"
    host: "https://api.openai.com/v1"
    temperature: 0.7             # 0.0-2.0
    max_tokens: 8000
    timeout: 60                  # Seconds
    max_retries: 5
    retry_delay: 2               # Seconds
    
    # Optional: Structured output
    structured_output:
      format: "json"             # json|xml|enum
      schema:
        type: "object"
        properties:
          field: 
            type: "string"
```

### Anthropic (Claude)

```yaml
llms:
  claude:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"
    host: "https://api.anthropic.com"
    temperature: 0.7
    max_tokens: 8000
    timeout: 120
    max_retries: 5
    retry_delay: 2
```

### Google Gemini

```yaml
llms:
  gemini:
    type: "gemini"
    model: "gemini-2.0-flash-exp"
    api_key: "${GEMINI_API_KEY}"
    host: "https://generativelanguage.googleapis.com"
    temperature: 0.7
    max_tokens: 4096
    timeout: 60
```

### Custom LLM (Plugin)

```yaml
plugins:
  llms:
    - name: "custom-llm"
      protocol: "grpc"
      path: "/path/to/plugin"

llms:
  custom:
    type: "plugin:custom-llm"
    model: "custom-model"
    # Plugin-specific configuration
```

---

## Agent Configuration

### Configuration Shortcuts

Simplify agent configuration with shortcuts:

```yaml
agents:
  assistant:
    name: "My Assistant"
    llm: "claude"
    
    # 🎯 Quick config shortcuts (mutually exclusive with explicit config)
    docs_folder: "./"         # Auto-creates document store + search tool + db + embedder
    enable_tools: true        # Auto-enables all local tools (execute_command, write_file, etc.)
```

**What gets auto-created:**
- `docs_folder: "./"` → Document store + Qdrant database + Ollama embedder + search tool
- `enable_tools: true` → All local tools with safe defaults (sandboxed)

⚠️ **Mutual exclusivity:** Cannot mix shortcuts with explicit config:
- `docs_folder` + `document_stores` = Error
- `enable_tools` + `tools: [...]` = Error

**Comparison:**

| Feature | Shortcuts | Explicit Config |
|---------|-----------|-----------------|
| **Lines of Config** | ~15-40 | ~200-450 |
| **Setup Time** | ~5 minutes | ~30 minutes |
| **Customization** | Limited | Full control |
| **Use Case** | Quick start, demos | Production, fine-tuning |
| **Best For** | 90% of users | Power users |

### Basic Agent

```yaml
agents:
  assistant:
    name: "My Assistant"         # Display name
    llm: "gpt-4o"               # References llms.<name>
    
    prompt:
      system_role: "You are a helpful assistant."
    
    tools:
      - "write_file"
      - "execute_command"
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100
```

### Complete Agent Configuration

```yaml
agents:
  advanced:
    name: "Advanced Agent"
    llm: "gpt-4o"
    database: "qdrant"           # For RAG & long-term memory
    embedder: "embedder"         # For RAG & long-term memory
    
    # Prompt Configuration
    prompt:
      # Simple approach (recommended): Single system prompt
      system_prompt: |
        You are an expert assistant. Provide clear, accurate responses.
        
        Guidelines:
        - Think step-by-step
        - Use available tools when helpful
        - Ask clarifying questions
      
      # Advanced: Slot-based prompts for granular control
      # Note: Use prompt_slots OR system_prompt, not both
      prompt_slots:
        system_role: "Core role and identity"
        reasoning_instructions: "How to think and approach problems"
        tool_usage: "Guidelines for using tools"
        output_format: "Response format preferences"
        communication_style: "Tone and interaction style"
        additional: "Extra context or instructions"
      
      # RAG toggle: Include document context in prompts
      include_context: true  # Default: false
    
    # Reasoning Configuration
    reasoning:
      engine: "chain-of-thought"         # chain-of-thought|supervisor
      max_iterations: 100                 # Safety limit
      
      # LLM Reasoning (improve quality)
      enable_self_reflection: false       # LLM outputs <thinking> tags (Chain of Thought)
      enable_structured_reflection: true  # LLM-based tool analysis (default: true)
      enable_goal_extraction: false       # For supervisor strategy only
      
      # Display Options
      show_thinking: false                # Show [Thinking: ...] meta-reflection blocks
      show_tool_execution: true           # Show tool execution labels
      show_debug_info: false              # Show iteration counts, tokens
      enable_streaming: true              # Real-time output (default: true)
    
    # Tools
    tools:
      - "search"
      - "write_file"
      - "search_replace"
      - "execute_command"
      - "todo_write"
      - "agent_call"
    
    # Memory Configuration
    memory:
      working:
        strategy: "summary_buffer"  # summary_buffer|buffer_window
        budget: 2000                # Tokens
        threshold: 0.8              # Trigger at 80%
        target: 0.6                 # Compress to 60%
        window_size: 20             # For buffer_window
      
      longterm:
        
        storage_scope: "session"    # all|session|conversational|summaries_only
        batch_size: 1
        auto_recall: true
        recall_limit: 5
        collection: "agent_memory"
    
    # Document Stores (RAG)
    document_stores:
      - name: "codebase"
        paths: ["./src/"]
        include_patterns: ["*.go", "*.py"]
        exclude_patterns: ["*_test.go"]
        chunk_size: 512
        chunk_overlap: 50
    
    # Security
    security:
      schemes:
        bearer_auth:
          type: "http"
          scheme: "bearer"
        api_key:
          type: "apiKey"
          name: "X-API-Key"
          in: "header"
      require:
        - bearer_auth
    
    # Visibility
    visibility: "public"          # public|internal|private
```

### External A2A Agent

```yaml
agents:
  external:
    type: "a2a"
    url: "https://external-agent.com"
    timeout: 60
    max_retries: 3
    
    credentials:
      type: "bearer"              # bearer|api_key|basic
      token: "${EXTERNAL_TOKEN}"
      # OR
      # key: "${API_KEY}"
      # header: "X-API-Key"
      # OR
      # username: "${USERNAME}"
      # password: "${PASSWORD}"
    
    tls:
      verify: true
      ca_cert: "/path/to/ca.crt"
```

### Supervisor Agent (Multi-Agent)

```yaml
agents:
  coordinator:
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"
      enable_goal_extraction: true
    tools:
      - "agent_call"
      - "todo_write"
    sub_agents:                   # Optional: restrict to specific agents
      - "researcher"
      - "analyst"
```

---

## Tool Configuration

### Built-in Tools

```yaml
tools:
  execute_command:
    type: command
    
    # Permissive defaults: all commands allowed (sandboxed)
    # Optional restrictions:
    # allowed_commands: ["npm", "git", "python"]
    # denied_commands: ["rm", "sudo"]
    # max_execution_time: "30s"
  
  write_file:
    type: write_file
    
    # Permissive defaults: all file types and paths allowed
    # Optional restrictions:
    # allowed_paths: ["./src/", "./docs/"]
    # denied_paths: ["./secrets/"]
  
  search_replace:
    type: search_replace
    
    # Permissive defaults: no restrictions
    # Optional: backup: true
  
  search:
    
    default_limit: 10
    max_limit: 50
  
  todo_write:
    
```

### MCP Tools

```yaml
tools:
  mcp_tools:
    - server:
        url: "http://localhost:3000"
        protocol: "mcp"
        timeout: "30s"
      auth:
        type: "bearer"
        token: "${MCP_TOKEN}"
      
      tools:                      # Optional: specific tools only
        - "github_create_issue"
        - "slack_send_message"
```

---

## Databases (Vector Stores)

### Qdrant

```yaml
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334                    # Default: 6334
    timeout: 300
    use_tls: false
    insecure: false
    api_key: "${QDRANT_API_KEY}"  # Optional
```

### Custom Database (Plugin)

```yaml
plugins:
  databases:
    - name: "custom-db"
      protocol: "grpc"
      path: "/path/to/plugin"

databases:
  custom:
    type: "plugin:custom-db"
    # Plugin-specific configuration
```

---

## Session Stores

Session stores provide persistent storage for conversation history and session metadata. When configured, agents can resume conversations after server restarts.

### SQL Session Store (SQLite)

```yaml
session_stores:
  local-db:
    backend: sql
    sql:
      driver: sqlite
      database: ./data/sessions.db
      max_conns: 10              # Maximum connections
      max_idle: 2                # Idle connections
      conn_max_lifetime: 3600    # Seconds (default: 0 = unlimited)
```

**Best for:** Local development, single-instance deployments

### SQL Session Store (PostgreSQL)

```yaml
session_stores:
  postgres-db:
    backend: sql
    sql:
      driver: postgres
      host: localhost
      port: 5432
      user: hector
      password: "${DB_PASSWORD}"  # From environment
      database: hector_sessions
      ssl_mode: require          # disable|require|verify-ca|verify-full
      max_conns: 100
      max_idle: 25
      conn_max_lifetime: 3600
```

**Best for:** Production deployments, distributed systems

### SQL Session Store (MySQL)

```yaml
session_stores:
  mysql-db:
    backend: sql
    sql:
      driver: mysql
      host: localhost
      port: 3306
      user: hector
      password: "${DB_PASSWORD}"
      database: hector_sessions
      max_conns: 100
      max_idle: 25
      conn_max_lifetime: 3600
```

### Agent Configuration

Reference session stores by name:

```yaml
session_stores:
  main-db:
    backend: sql
    sql:
      driver: sqlite
      database: ./data/sessions.db

agents:
  assistant:
    session_store: "main-db"  # References global store
    memory:
      working:
        strategy: "summary_buffer"
```

**Multi-Agent Isolation:** Multiple agents can share a session store. Sessions are isolated by `agent_id` + `session_id`.

### Complete Example

```yaml
session_stores:
  # Shared production database
  prod-db:
    backend: sql
    sql:
      driver: postgres
      host: db.example.com
      port: 5432
      user: hector
      password: "${HECTOR_DB_PASSWORD}"
      database: hector_sessions
      ssl_mode: require
      max_conns: 200
      max_idle: 50
      conn_max_lifetime: 7200

  # Local development database
  dev-db:
    backend: sql
    sql:
      driver: sqlite
      database: ./dev-sessions.db
      max_conns: 5

agents:
  customer-support:
    session_store: "prod-db"    # Shares DB, isolated by agent_id
    
  sales-assistant:
    session_store: "prod-db"    # Shares DB, isolated by agent_id
    
  dev-agent:
    session_store: "dev-db"     # Separate database
```

**See:** [Setup Session Persistence](../how-to/setup-session-persistence.md) for full guide.

---

## Embedders

### Ollama

```yaml
embedders:
  embedder:
    type: "ollama"
    model: "nomic-embed-text"     # nomic-embed-text|all-minilm|mxbai-embed-large
    host: "http://localhost:11434"
    dimension: 768                # Depends on model
    timeout: 30
    max_retries: 3
```

### Custom Embedder (Plugin)

```yaml
plugins:
  embedders:
    - name: "custom-embedder"
      protocol: "grpc"
      path: "/path/to/plugin"

embedders:
  custom:
    type: "plugin:custom-embedder"
    # Plugin-specific configuration
```

---

## Document Stores

Document stores enable RAG (Retrieval-Augmented Generation) by indexing local directories for semantic search.

```yaml
document_stores:
  codebase:
    # Basic configuration
    name: "codebase"              # Required: Store identifier
    source: "directory"           # Required: Source type (only "directory" supported)
    path: "./src"                 # Required: Directory to index

    # File filtering
    include_patterns:             # Optional: Only include matching files
      - "*.go"
      - "*.py"
      - "*.md"

    # Smart exclusion system (choose one approach):
    # Approach 1: Extend defaults (recommended)
    additional_exclude_patterns:  # Extends comprehensive built-in defaults
      - "**/my-custom-dir/**"
      - "**/*.secret"

    # Approach 2: Override defaults (not recommended)
    # exclude_patterns:           # Replaces all defaults - use with caution
    #   - "**/.git/**"
    #   - "**/node_modules/**"

    # Chunking configuration
    chunk_size: 800               # Default: 800 characters per chunk
    chunk_overlap: 0              # Default: 0 characters overlap between chunks
    chunk_strategy: "simple"      # Options: "simple", "overlapping", "semantic"

    # Indexing behavior
    watch_changes: true           # Default: true - Auto-reindex on file changes
    incremental_indexing: true    # Default: true - Only reindex changed files
    max_file_size: 10485760       # Default: 10MB (in bytes)
    max_concurrent_files: 10      # Default: 10 concurrent file processors

    # Progress tracking
    show_progress: true           # Default: true - Show animated progress bar
    verbose_progress: false       # Default: false - Show current file name
    enable_checkpoints: true      # Default: true - Enable resume on interruption
    quiet_mode: true              # Default: true - Suppress per-file warnings

    # Metadata extraction (advanced)
    extract_metadata: false       # Default: false - Extract code metadata
    metadata_languages:           # Languages for metadata extraction
      - "go"
      - "python"
```

**Default Exclusions:**

By default, Hector excludes 115 patterns covering:
- **VCS**: `.git`, `.svn`, `.hg`, `.bzr`
- **Dependencies**: `node_modules`, `venv`, `*-env`, `*_env`, `site-packages`, `dist-packages`, `vendor`
- **Build artifacts**: `dist`, `build`, `target`, `bin`, `obj`, `.gradle`
- **IDE files**: `.vscode`, `.idea`, `.DS_Store`
- **Binary/Media**: `*.exe`, `*.dll`, `*.so`, `*.png`, `*.jpg`, `*.mp4`, `*.mp3`
- **Archives**: `*.zip`, `*.tar`, `*.gz`
- **Logs/Temp**: `*.log`, `*.tmp`, `logs/`, `tmp/`
- **Lock files**: `package-lock.json`, `yarn.lock`, `Cargo.lock`

See [types.go:865-923](https://github.com/kadirpekel/hector/blob/main/pkg/config/types.go#L865-L923) for the complete list.

**Chunk Strategies:**

| Strategy | Description | Use Case |
|----------|-------------|----------|
| `simple` | Fixed-size chunks with no overlap | Fast, simple indexing |
| `overlapping` | Fixed-size chunks with configurable overlap | Better context preservation |
| `semantic` | Intelligent chunking at logical boundaries | Highest quality (slower) |

**Progress Tracking:**

When indexing large directories, Hector shows:
- Animated progress bar with percentage
- Files/second processing rate
- ETA (estimated time remaining)
- Failed file count

If interrupted (Ctrl+C), checkpoints enable resuming from where it left off.

**Example - Minimal:**
```yaml
document_stores:
  docs:
    path: "./documentation"
```

**Example - Advanced:**
```yaml
document_stores:
  codebase:
    path: "./src"
    additional_exclude_patterns:
      - "**/generated/**"
      - "**/*.test.js"
    chunk_strategy: "semantic"
    chunk_size: 1200
    chunk_overlap: 200
    watch_changes: true
    verbose_progress: true
```

---

## Plugins

### Plugin Configuration

```yaml
plugins:
  llms:
    - name: "custom-llm"
      protocol: "grpc"
      path: "/path/to/plugin"
      config:
        custom_param: "value"
  
  databases:
    - name: "custom-db"
      protocol: "grpc"
      path: "/path/to/plugin"
  
  embedders:
    - name: "custom-embedder"
      protocol: "grpc"
      path: "/path/to/plugin"
  
  tools:
    - name: "custom-tools"
      protocol: "grpc"
      path: "/path/to/plugin"
  
  parsers:
    - name: "pdf-parser"
      protocol: "grpc"
      path: "/path/to/plugin"
```

---

## Logging

```yaml
logging:
  level: "info"                   # debug|info|warn|error
  format: "text"                  # text|json
  output: "stdout"                # stdout|stderr|file
  file: "/var/log/hector.log"     # If output=file
  max_size: 100                   # MB
  max_backups: 3
  max_age: 28                     # Days
  compress: true
```

---

## Environment Variable Substitution

All configuration values support environment variable substitution:

```yaml
llms:
  gpt-4o:
    api_key: "${OPENAI_API_KEY}"
    
agents:
  assistant:
    prompt:
      system_prompt: |
        You are ${AGENT_NAME} for ${COMPANY_NAME}.
        Provide helpful customer support.
```

**Syntax:**
- `${VAR}` - Required variable (error if not set)
- `${VAR:-default}` - Optional with default value

---

## Complete Example

```yaml
# Global Configuration
global:
  a2a_server:
    host: "0.0.0.0"
    port: 8080
  
  auth:
    jwks_url: "${JWKS_URL}"
    issuer: "${AUTH_ISSUER}"
    audience: "hector-api"

# LLM Providers
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7

# Vector Database
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334

# Embedder
embedders:
  embedder:
    type: "ollama"
    model: "nomic-embed-text"
    host: "http://localhost:11434"

# Agents
agents:
  coder:
    name: "Coding Assistant"
    llm: "gpt-4o"
    database: "qdrant"
    embedder: "embedder"
    
    prompt:
      system_prompt: |
        You are an expert programmer who writes clean,
        efficient, well-tested code. Think step-by-step
        and test your code thoroughly.
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100
      enable_streaming: true
    
    tools:
      - "search"
      - "write_file"
      - "execute_command"
    
    memory:
      strategy: "summary_buffer"
      budget: 4000
    
    document_stores:
      - "codebase"

# Tools
tools:
  execute_command:
    type: command
    
    # Permissive defaults (sandboxed)
  
  write_file:
    type: write_file
    
    # Permissive defaults

# Logging
logging:
  level: "info"
  format: "json"
```

---

## Validation

Hector validates configuration on startup. Common errors:

**Missing required fields:**
```
Error: agents.assistant.llm is required
```

**Invalid references:**
```
Error: agents.assistant.llm references undefined LLM "gpt-5"
```

**Invalid values:**
```
Error: agents.assistant.reasoning.max_iterations must be > 0
```

**Type mismatches:**
```
Error: agents.assistant.temperature must be a number, got string
```

---

## Configuration Precedence

1. **CLI flags** (highest priority)
2. **Environment variables**
3. **Configuration file**
4. **Defaults** (lowest priority)

Example:
```bash
# Config file: port 8080
# Override with flag:
hector serve --config config.yaml --port 9000
# Result: Uses port 9000
```

---

## Next Steps

- **[CLI Reference](cli.md)** - Command-line options
- **[API Reference](api.md)** - HTTP/gRPC APIs
- **[Core Concepts](../core-concepts/overview.md)** - Understanding configuration
- **[Examples](https://github.com/kadirpekel/hector/tree/main/configs)** - Example configurations

---

## Related Topics

- **[Getting Started](../getting-started/installation.md)** - Installation
- **[Agent Overview](../core-concepts/overview.md)** - Agent basics
- **[Deployment](../how-to/deploy-production.md)** - Production configuration

