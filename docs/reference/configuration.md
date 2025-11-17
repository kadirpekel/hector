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
vector_stores:    # Vector databases (Qdrant, Pinecone, etc.)
databases:        # SQL databases (PostgreSQL, MySQL, SQLite)
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
```

**Note:** Hector uses Go's standard `log` package. For production logging, redirect stdout/stderr to your logging infrastructure.

---

## Global Configuration

### global.a2a_server

A2A server settings (when running `hector serve`):

```yaml
global:
  a2a_server:
    host: "0.0.0.0"                        # Bind host (default: 0.0.0.0)
    port: 8080                             # HTTP/REST/JSON-RPC port (default: 8080)
    grpc_port: 50051                       # Internal gRPC port (default: 50051)
    base_url: "http://localhost:8080"      # External URL for agent cards
    preferred_transport: "json-rpc"        # Preferred A2A transport: "grpc", "json-rpc", or "rest"
```

**Notes:**
- `port`: The main HTTP port for REST API, JSON-RPC, and Web UI
- `grpc_port`: Internal gRPC port for service-to-service communication (optional, defaults to 50051)
- `base_url`: The external URL used in agent cards for client discovery
- `preferred_transport`: The transport protocol advertised in agent cards

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
    
    # TLS configuration (for HTTPS connections with self-signed or custom CA certificates)
    insecure_skip_verify: false  # Skip certificate verification (dev/test only)
    ca_certificate: ""          # Path to custom CA certificate file
    
    # Optional: Structured output
    structured_output:
      format: "json"             # json|xml|enum
      schema:
        type: "object"
        properties:
          field: 
            type: "string"
```

**TLS Configuration for Self-Hosted LLMs:**
- `insecure_skip_verify: true` - Skip TLS certificate verification (⚠️ dev/test only, shows warning)
- `ca_certificate: "/path/to/ca.pem"` - Use custom CA certificate for internal/private certificates

**Example with self-signed certificate:**
```yaml
llms:
  internal_openai:
    type: "openai"
    model: "gpt-4"
    host: "https://internal-llm.company.com"
    api_key: "${INTERNAL_API_KEY}"
    insecure_skip_verify: true  # For self-signed certificates
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
    
    # TLS configuration (for HTTPS connections with self-signed or custom CA certificates)
    insecure_skip_verify: false  # Skip certificate verification (dev/test only)
    ca_certificate: ""          # Path to custom CA certificate file
```

**TLS Configuration:** Same as OpenAI (see above for examples)

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
    
    # TLS configuration (for HTTPS connections with self-signed or custom CA certificates)
    insecure_skip_verify: false  # Skip certificate verification (dev/test only)
    ca_certificate: ""          # Path to custom CA certificate file
```

**TLS Configuration:** Same as OpenAI (see above for examples)

### Ollama (Local Models)

⚠️ **Note**: Currently, only the `qwen3` model is fully supported and tested.

```yaml
llms:
  local-llm:
    type: "ollama"
    model: "qwen3"                    # Currently supported: qwen3
    host: "http://localhost:11434"  # Default: http://localhost:11434 (use https:// for HTTPS)
    temperature: 0.7                 # Default: 0.7
    max_tokens: 8000                  # Default: 8000
    timeout: 600                      # Seconds, default: 600 (10 minutes)
    # Note: Ollama doesn't require an API key for local deployments
    
    # TLS configuration (for HTTPS connections with self-signed or custom CA certificates)
    insecure_skip_verify: false  # Skip certificate verification (dev/test only)
    ca_certificate: ""          # Path to custom CA certificate file
```

**TLS Configuration:** Same as OpenAI (see above for examples)

**Example with HTTPS Ollama:**
```yaml
llms:
  remote-ollama:
    type: "ollama"
    model: "qwen3"
    host: "https://ollama.company.com"  # HTTPS URL
    insecure_skip_verify: true  # For self-signed certificates
```

**Prerequisites:**
1. Install Ollama: [https://ollama.ai](https://ollama.ai)
2. Pull the model: `ollama pull qwen3`
3. Ensure Ollama service is running

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

### Inline Provider Configs (Single-Agent Setups)

For single-agent setups, you can define providers inline to reduce cognitive load:

```yaml
agents:
  my-agent:
    name: "My Agent"
    
    # Inline LLM config (alternative to top-level llms: section)
    llm_config:
      type: "openai"
      model: "gpt-4o"
      api_key: "${OPENAI_API_KEY}"
      temperature: 0.7
    
    # Inline vector store config
    vector_store_config:
      type: "qdrant"
      host: "localhost"
      port: 6334
    
    # Inline embedder config
    embedder_config:
      type: "ollama"
      model: "nomic-embed-text"
      host: "http://localhost:11434"
      dimension: 768
```

**Note:** Inline configs are automatically expanded to top-level providers during processing. You cannot use both a reference (`llm: "name"`) and an inline config (`llm_config: {...}`) for the same provider.

### Defaults for Multi-Agent Setups

Reduce repetition in multi-agent configurations using the `defaults` section:

```yaml
# Define defaults once
defaults:
  llm: "gpt-4o"
  vector_store: "main"
  embedder: "ollama"
  session_store: "postgres"

# Agents inherit defaults
agents:
  agent1:
    name: "Agent 1"
    # Inherits: llm, vector_store, embedder, session_store from defaults
  
  agent2:
    name: "Agent 2"
    llm: "claude"  # Override default LLM
    # Inherits: vector_store, embedder, session_store from defaults
  
  agent3:
    name: "Agent 3"
    # Inherits all defaults
```

Defaults are only applied if the agent doesn't explicitly set the field.

### Complete Agent Configuration

```yaml
agents:
  advanced:
    name: "Advanced Agent"
    llm: "gpt-4o"
    vector_store: "qdrant"       # For RAG & long-term memory
    embedder: "embedder"         # For RAG & long-term memory
    
    # Prompt Configuration (choose ONE approach)
    prompt:
      # OPTION 1: Complete Override (Full Control)
      # Use this when you want complete control over the prompt
      # WARNING: Disables ALL strategy behavior (tool patterns, workflows, etc.)
      system_prompt: |
        You are an expert assistant. Provide clear, accurate responses.
        
        Guidelines:
        - Think step-by-step
        - Use available tools when helpful
        - Ask clarifying questions
      
      # OPTION 2: Slot-Based Prompts (Recommended - Merges with Strategy)
      # Use this to customize while preserving strategy optimizations
      # Note: Use prompt_slots OR system_prompt, not both
      prompt_slots:
        system_role: |
          WHO you are: Your identity and purpose.
          Example: "You are a Python expert specializing in FastAPI"
        
        instructions: |
          HOW you behave: Behavioral guidance, tool usage patterns, workflow.
          Example: "Always write tests. Prefer async/await over callbacks."
          
          NOTE: Replaces strategy's instructions. Use carefully.
          Most users should use user_guidance instead.
        
        user_guidance: |
          WHAT the user wants: Task-specific guidance (HIGHEST PRIORITY).
          Example: "Focus on performance. Use type hints everywhere."
          
          This is applied LAST and doesn't break strategy behavior.
          RECOMMENDED for most customizations.
      
      # RAG toggle: Include document context in prompts
      include_context: true  # Default: false
      include_context_limit: 10      # Max documents to include (default: uses search.top_k)
      include_context_max_length: 500 # Max content length per document (default: 500)
    
    # Search Configuration (RAG)
    search:
      top_k: 10              # Default number of results (used when limit is 0)
      threshold: 0.5          # Minimum similarity score (0.0-1.0)
      preserve_case: true    # Don't lowercase queries (default: true for code search)
      search_mode: "vector"   # "vector", "hybrid", "keyword", "multi_query", or "hyde" (default: "vector")
      hybrid_alpha: 0.5      # Blending factor for hybrid search (0.0-1.0, default: 0.5)
      
      # Optional: LLM-based re-ranking
      rerank:
        enabled: false        # Enable re-ranking (default: false)
        llm: "gpt-4o-mini"    # LLM provider name for reranking (required if enabled)
        max_results: 20       # Maximum results to rerank (default: 20)
      
      # Optional: Multi-query expansion
      multi_query:
        enabled: false       # Enable multi-query expansion (default: false)
        llm: "gpt-4o-mini"   # LLM provider name for query expansion (required if enabled)
        num_variations: 3    # Number of query variations to generate (default: 3)
      
      # Optional: HyDE (Hypothetical Document Embeddings)
      hyde:
        enabled: false       # Enable HyDE search (default: false)
        llm: "gpt-4o-mini"   # LLM provider name for generating hypothetical documents (required if enabled)
    
    # Reasoning Configuration
    reasoning:
      engine: "chain-of-thought"         # chain-of-thought|supervisor
      max_iterations: 100                 # Safety limit
      
      # LLM Reasoning (improve quality)
      enable_goal_extraction: false       # For supervisor strategy only
      
      # Display Options
      enable_thinking_display: false      # Show [Thinking: ...] meta-reflection blocks
      enable_streaming: true              # Real-time output (default: true)
    
    # Tools
    tools:
      - "read_file"        # Read file contents
      - "apply_patch"      # Apply contextual patches
      - "grep_search"      # Regex pattern search
      - "search"           # Semantic search
      - "write_file"       # Create/modify files
      - "search_replace"   # Find and replace
      - "execute_command"  # Run shell commands
      - "todo_write"       # Task tracking
      - "agent_call"       # Call other agents
    
    # Memory Configuration
    memory:
      working:
        strategy: "summary_buffer"  # summary_buffer|buffer_window
        budget: 8000                # Tokens (default: 8000)
        threshold: 0.85              # Trigger at 85% (default: 0.85)
        target: 0.7                  # Compress to 70% (default: 0.7)
        window_size: 20              # For buffer_window
      
      longterm:
        
        storage_scope: "session"    # all|session|conversational|summaries_only
        batch_size: 1
        auto_recall: true
        recall_limit: 5
        collection: "agent_memory"
    
    # Document Stores (RAG) - Assign stores by name
    # nil/omitted = access all stores, [] = no access, [names...] = scoped access
    document_stores:
      - "codebase"           # Reference to document_stores.codebase (defined globally)
      - "docs"               # Reference to document_stores.docs
    
    # Note: Document stores are defined globally in the document_stores: section
    # This agent assignment controls which stores this agent can access
    
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
    
    # Task Configuration (for async tasks, HITL, and checkpoint recovery)
    task:
      backend: "memory"           # "memory" (default) or "sql"
      worker_pool: 5              # Max concurrent tasks
      input_timeout: 600          # Seconds to wait for user input (HITL, default: 600 = 10 minutes)
      timeout: 3600               # Timeout for async task execution (default: 3600 = 1 hour)
      
      # HITL Mode Configuration (optional)
      hitl:
        mode: "auto"              # "auto" (default), "blocking", or "async"
      
      # Checkpoint Recovery Configuration (flattened structure)
      enable_checkpointing: false      # Enable checkpointing (default: false)
      checkpoint_strategy: "event"     # "event", "interval", or "hybrid" (default: "event")
      checkpoint_interval: 0            # Checkpoint every N iterations (0 = disabled)
      checkpoint_after_tools: false    # Always checkpoint after tool calls (default: false)
      checkpoint_before_llm: false     # Checkpoint before LLM calls (default: false)
      
      # Recovery configuration
      auto_resume: false          # Auto-resume on startup (default: false)
      auto_resume_hitl: false     # Auto-resume INPUT_REQUIRED tasks (default: false)
      resume_timeout: 3600        # Max time to resume after restart (seconds, default: 3600)
      
      # SQL backend (for production persistence)
      sql_database: "tasks-db"    # Reference to SQL database from databases: section
```

**Task Configuration Options:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `backend` | string | `"memory"` | Storage backend: `"memory"` (volatile) or `"sql"` (persistent) |
| `worker_pool` | integer | `5` | Maximum number of concurrent tasks |
| `input_timeout` | integer | `600` | Timeout in seconds for user input (Human-in-the-Loop) |
| `timeout` | integer | `3600` | Timeout in seconds for async task execution |
| `hitl.mode` | string | `"auto"` | HITL mode: `"auto"`, `"blocking"`, or `"async"` |
| `enable_checkpointing` | boolean | `false` | Enable checkpointing |
| `checkpoint_strategy` | string | `"event"` | Checkpoint strategy: `"event"`, `"interval"`, or `"hybrid"` |
| `checkpoint_interval` | integer | `0` | Checkpoint every N iterations (0 = disabled) |
| `checkpoint_after_tools` | boolean | `false` | Always checkpoint after tool calls |
| `checkpoint_before_llm` | boolean | `false` | Checkpoint before LLM calls |
| `auto_resume` | boolean | `false` | Auto-resume tasks on startup |
| `auto_resume_hitl` | boolean | `false` | Auto-resume INPUT_REQUIRED tasks |
| `resume_timeout` | integer | `3600` | Max time to resume after restart (seconds) |
| `sql_database` | string | - | Reference to SQL database from `databases:` section |

**Human-in-the-Loop (HITL):**

- `input_timeout` controls how long tasks wait for user approval
- When a tool with `requires_approval: true` is called, task pauses and waits
- User responds using the same `taskId` (A2A multi-turn)
- `hitl.mode` determines execution behavior:

  - `"auto"` (default): Uses async mode if `session_store` is configured, else blocking
  - `"blocking"`: Execution goroutine blocks while waiting (simple, but not persistent)
  - `"async"`: State saved to session metadata, goroutine exits (requires `session_store`)
- See [Human-in-the-Loop](../core-concepts/human-in-the-loop.md) for details
- See [Checkpoint Recovery](../core-concepts/checkpoint-recovery.md) for async HITL implementation

**Checkpoint Recovery:**

- `checkpoint.enabled` enables generic checkpoint/resume functionality
- Checkpoints are stored in session metadata (requires `session_store`)
- Three strategies:

  - `"event"`: Checkpoint on HITL pauses (uses async HITL foundation)
  - `"interval"`: Background checkpointing every N iterations (task remains in `WORKING`)
  - `"hybrid"`: Combines event-driven and interval-based checkpointing
- Recovery settings:

  - `auto_resume`: Automatically resume tasks on server startup
  - `auto_resume_hitl`: Auto-resume INPUT_REQUIRED tasks (use with caution)
  - `resume_timeout`: Maximum age of checkpoint to resume (prevents stale recovery)
- See [Checkpoint Recovery](../core-concepts/checkpoint-recovery.md) for architecture details
- See [Tasks](../core-concepts/tasks.md#checkpoint-recovery) for usage guide

**See:** [Tasks](../core-concepts/tasks.md) for complete task management guide.

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
      # type: "api_key"
      # api_key: "${API_KEY}"
      # api_key_header: "X-API-Key"
      # OR
      # type: "basic"
      # username: "${USERNAME}"
      # password: "${PASSWORD}"
    
    # TLS configuration (for HTTPS connections)
    insecure_skip_verify: false  # Skip certificate verification (dev/test only)
    ca_certificate: ""          # Path to custom CA certificate file
```

**TLS Configuration:**
- `insecure_skip_verify: true` - Skip TLS certificate verification (⚠️ dev/test only, shows warning)
- `ca_certificate: "/path/to/ca.pem"` - Use custom CA certificate for internal/private certificates

**Example with self-signed certificate:**
```yaml
agents:
  internal_agent:
    type: "a2a"
    url: "https://internal-agent.company.com"
    insecure_skip_verify: true  # For self-signed certificates
    credentials:
      type: "bearer"
      token: "${INTERNAL_TOKEN}"
```

**Example with custom CA:**
```yaml
agents:
  internal_agent:
    type: "a2a"
    url: "https://internal-agent.company.com"
    ca_certificate: "/etc/ssl/certs/company-ca.pem"
    credentials:
      type: "bearer"
      token: "${INTERNAL_TOKEN}"
```

**See:** [TLS/HTTPS Configuration](../core-concepts/tls-https.md) for complete guide.

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
    sub_agents:                   # Optional: agent assignment (consistent with tools/document_stores)
                                  # nil/omitted = all agents, [] = no agents, [agents...] = scoped
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
    
    # Human-in-the-loop: Require approval before execution
    requires_approval: true           # Pause task for user approval
    approval_prompt: "Execute command: {input}?"  # Custom prompt (optional)
  
  write_file:
    type: write_file
    
    # Permissive defaults: all file types and paths allowed
    # Optional restrictions:
    # allowed_paths: ["./src/", "./docs/"]
    # denied_paths: ["./secrets/"]
    
    # Human-in-the-loop: Require approval for file writes
    requires_approval: true
    approval_prompt: |
      📝 File Write Request
      File: {tool}
      Content: {input}
      Approve?
  
  search_replace:
    type: search_replace
    
    # Permissive defaults: no restrictions
    # Optional: backup: true
  
  read_file:
    type: read_file
    max_file_size: 10485760  # 10MB
    working_directory: "./"
    requires_approval: false  # Safe read-only operation
  
  apply_patch:
    type: apply_patch
    max_file_size: 10485760  # 10MB
    context_lines: 3  # Require context lines before/after changes
    working_directory: "./"
  
  grep_search:
    type: grep_search
    max_results: 1000
    max_file_size: 10485760  # 10MB
    context_lines: 2
    working_directory: "./"
  
  search:
    type: search
    max_limit: 50            # Maximum results allowed (default limit comes from agent.search.top_k)
  
  todo_write:
    
```

### Human-in-the-Loop (Tool Approval)

Configure tools to require user approval before execution:

```yaml
tools:
  dangerous_tool:
    type: command
    enabled: true
    
    # Enable approval workflow
    requires_approval: true           # If true, task pauses for approval
    approval_prompt: "Custom prompt"  # Optional: Custom approval message
    
    # Prompt interpolation variables:
    # {tool} - Tool name
    # {input} - Tool input/arguments (JSON string)
```

**When `requires_approval: true`:**
- Task transitions to `TASK_STATE_INPUT_REQUIRED` when tool is called
- Execution pauses until user responds with approval/denial
- User responds using the same `taskId` (A2A multi-turn)
- Tool executes if approved, skipped if denied

**See:** [Human-in-the-Loop](../core-concepts/human-in-the-loop.md) for complete guide.

**See also:**
- [Tools Guide](../core-concepts/tools.md) - Detailed documentation for tools including read_file, apply_patch, and grep_search
- [Tools Overview](../core-concepts/tools.md) - Complete guide to all built-in tools

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

**MCP Tool with TLS Configuration:**

```yaml
tools:
  docling:
    type: "mcp"
    enabled: true
    server_url: "https://docling.example.com/mcp"  # HTTPS URL
    description: "Docling - Document parsing"
    
    # TLS configuration (for HTTPS connections)
    insecure_skip_verify: false  # Skip certificate verification (dev/test only)
    ca_certificate: ""          # Path to custom CA certificate file
```

**TLS Options:**
- `insecure_skip_verify: true` - Skip TLS certificate verification (⚠️ dev/test only, shows warning)
- `ca_certificate: "/path/to/ca.pem"` - Use custom CA certificate for internal/private certificates

**Example with self-signed certificate:**
```yaml
tools:
  internal_mcp:
    type: "mcp"
    server_url: "https://internal-mcp.company.com/mcp"
    insecure_skip_verify: true  # For self-signed certificates
```

**Example with custom CA:**
```yaml
tools:
  internal_mcp:
    type: "mcp"
    server_url: "https://internal-mcp.company.com/mcp"
    ca_certificate: "/etc/ssl/certs/company-ca.pem"
```

**See:** [TLS/HTTPS Configuration](../core-concepts/tls-https.md) for complete guide.

---

## Vector Stores

Vector stores are used for storing embeddings and performing similarity search. They are required for RAG (document stores) and long-term memory.

Hector supports multiple vector databases:

| Database | Type | Hybrid Search | Best For |
|----------|------|--------------|----------|
| **Qdrant** | Self-hosted | ✅ (RRF) | Production, self-hosted |
| **Pinecone** | Managed | ✅ (RRF) | Managed cloud service |
| **Weaviate** | Self-hosted | ✅ (Native) | Native hybrid search, GraphQL |
| **Milvus** | Self-hosted | ✅ (RRF) | Large-scale, high-performance |
| **Chroma** | Self-hosted | ✅ (RRF) | Simple, lightweight |

### Qdrant

```yaml
vector_stores:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334                    # Default: 6334
    use_tls: false
    api_key: "${QDRANT_API_KEY}"  # Optional
```

### Pinecone

```yaml
vector_stores:
  pinecone:
    type: "pinecone"
    api_key: "${PINECONE_API_KEY}"
    environment: "us-east-1"  # Your Pinecone environment
```

### Weaviate

```yaml
vector_stores:
  weaviate:
    type: "weaviate"
    host: "localhost"
    port: 8080           # Default Weaviate port
    api_key: ""          # Optional API key
    enable_tls: false    # Enable HTTPS (default: false)
    
    # TLS configuration (for HTTPS connections)
    insecure_skip_verify: false  # Skip certificate verification (dev/test only)
    ca_certificate: ""          # Path to custom CA certificate file
```

**TLS Configuration:**
- `enable_tls: true` - Use HTTPS instead of HTTP
- `insecure_skip_verify: true` - Skip TLS certificate verification (⚠️ dev/test only, shows warning)
- `ca_certificate: "/path/to/ca.pem"` - Use custom CA certificate for internal/private certificates

**Example with self-signed certificate:**
```yaml
vector_stores:
  weaviate:
    type: "weaviate"
    host: "internal-weaviate.company.com"
    port: 443
    enable_tls: true
    insecure_skip_verify: true  # For self-signed certificates
```

**Example with custom CA:**
```yaml
vector_stores:
  weaviate:
    type: "weaviate"
    host: "internal-weaviate.company.com"
    port: 443
    enable_tls: true
    ca_certificate: "/etc/ssl/certs/company-ca.pem"
```

**Features:**
- Native hybrid search support (no fallback needed)
- GraphQL API
- Built-in vectorization options

### Milvus

```yaml
vector_stores:
  milvus:
    type: "milvus"
    host: "localhost"
    port: 19530          # Default Milvus port
    api_key: ""          # Optional
    enable_tls: false    # Enable HTTPS (default: false)
    
    # TLS configuration (for HTTPS connections)
    insecure_skip_verify: false  # Skip certificate verification (dev/test only)
    ca_certificate: ""          # Path to custom CA certificate file
```

**TLS Configuration:** Same as Weaviate (see above for examples)

**Features:**
- Optimized for large-scale deployments
- High-performance vector search
- Supports distributed deployments

### Chroma

```yaml
vector_stores:
  chroma:
    type: "chroma"
    host: "localhost"
    port: 8000           # Default Chroma port
    api_key: ""          # Optional
    enable_tls: false    # Enable HTTPS (default: false)
    
    # TLS configuration (for HTTPS connections)
    insecure_skip_verify: false  # Skip certificate verification (dev/test only)
    ca_certificate: ""          # Path to custom CA certificate file
```

**TLS Configuration:** Same as Weaviate (see above for examples)

**Features:**
- Simple and lightweight
- Good for development and small deployments
- Easy to set up

### Custom Vector Store (Plugin)

```yaml
plugins:
  database_providers:
    - name: "custom-vector-store"
      protocol: "grpc"
      path: "/path/to/plugin"

vector_stores:
  custom:
    type: "plugin:custom-vector-store"
    # Plugin-specific configuration
```

---

## SQL Databases

SQL databases are used for relational data storage (session stores, task persistence, document store SQL sources). All SQL database configurations are centralized in the `databases:` section.

### PostgreSQL

```yaml
databases:
  postgres-main:
    driver: "postgres"
    host: "localhost"
    port: 5432
    database: "hector_main"
    username: "user"
    password: "${DB_PASSWORD}"
    ssl_mode: "require"           # disable|require|verify-ca|verify-full
    max_conns: 25                  # Maximum connections
    max_idle: 5                    # Idle connections
    conn_max_lifetime: "1h"        # Connection lifetime (e.g., "1h", "30m")
    conn_max_idle_time: "30m"      # Idle timeout (e.g., "30m", "15m")
```

### MySQL

```yaml
databases:
  mysql-analytics:
    driver: "mysql"
    host: "analytics.example.com"
    port: 3306
    database: "analytics"
    username: "analytics_user"
    password: "${MYSQL_PASSWORD}"
    max_conns: 25
    max_idle: 5
    conn_max_lifetime: "1h"
    conn_max_idle_time: "30m"
```

### SQLite

```yaml
databases:
  sqlite-local:
    driver: "sqlite"
    database: "./data/hector.db"   # File path for SQLite
    max_conns: 1                    # SQLite typically uses 1 connection
    max_idle: 1
    conn_max_lifetime: "1h"
    conn_max_idle_time: "30m"
```

**Note:** For SQLite, `host` and `port` are not required. The `database` field contains the file path.

---

## Session Stores {#session-store}

Session stores provide persistent storage for conversation history and session metadata. When configured, agents can resume conversations after server restarts.

### SQL Session Store (Using Database Reference)

**Recommended:** Reference a SQL database from the `databases:` section:

```yaml
# Define SQL database once
databases:
  postgres-main:
    driver: "postgres"
    host: "localhost"
    port: 5432
    database: "hector_main"
    username: "user"
    password: "${DB_PASSWORD}"
    ssl_mode: "require"
    max_conns: 25
    max_idle: 5
    conn_max_lifetime: "1h"
    conn_max_idle_time: "30m"

# Reference it in session store
session_stores:
  default:
    backend: sql
    database: "postgres-main"  # Reference to databases section
```

**Best for:** Production deployments, shared database connections

### SQL Session Store (Inline Config - Deprecated)

**Note:** Inline SQL configuration is deprecated. Use database references instead.

```yaml
session_stores:
  local-db:
    backend: sql
    sql_database: "sessions-db"  # Reference to SQL database from databases: section
```

**Best for:** Production deployments with shared databases

### Agent Configuration

Reference session stores by name:

```yaml
databases:
  sessions-db:
    driver: sqlite
    database: ./data/sessions.db

session_stores:
  main-db:
    backend: sql
    sql_database: "sessions-db"

agents:
  assistant:
    session_store: "main-db"  # References global store
    memory:
      working:
        strategy: "summary_buffer"
```

**Multi-Agent Isolation:** Multiple agents can share a session store. Sessions are isolated by `agent_id` + `session_id`.

### Rate Limiting

Control API usage with flexible rate limiting per session or user:

```yaml
databases:
  hector-db:
    driver: postgres
    database: hector_db

session_stores:
  default:
    backend: sql
    sql_database: "hector-db"
    rate_limit:
      enabled: true
      scope: user              # "session" or "user"
      backend: sql             # "memory" or "sql"
      limits:
        - type: count          # Request count limit
          window: minute       # minute|hour|day|week|month
          limit: 60
        - type: token          # Token usage limit
          window: day
          limit: 100000
```

**Limit Types:**
- `count`: Number of requests/messages
- `token`: LLM token usage (cost control)

**Scopes:**
- `session`: Each session has independent quota
- `user`: All sessions for a user share quota

**Backends:**
- `memory`: Fast, volatile (for development)
- `sql`: Persistent, production-ready

See [Rate Limiting](../core-concepts/rate-limiting.md) for details.

### Complete Example

```yaml
databases:
  prod-sessions:
    driver: postgres
    host: db.example.com
    port: 5432
    username: hector
    password: "${HECTOR_DB_PASSWORD}"
    database: hector_sessions
    ssl_mode: require
    max_conns: 200
    max_idle: 50
    conn_max_lifetime: 7200

  dev-sessions:
    driver: sqlite
    database: ./dev-sessions.db
    max_conns: 5

session_stores:
  # Shared production database
  prod-db:
    backend: sql
    sql_database: "prod-sessions"

  # Local development database
  dev-db:
    backend: sql
    sql_database: "dev-sessions"

agents:
  customer-support:
    session_store: "prod-db"    # Shares DB, isolated by agent_id
    
  sales-assistant:
    session_store: "prod-db"    # Shares DB, isolated by agent_id
    
  dev-agent:
    session_store: "dev-db"     # Separate database
```

**See:** [Sessions](../core-concepts/sessions.md) for full guide.

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

Document stores enable RAG (Retrieval-Augmented Generation) by indexing documents from various sources for semantic search. Hector supports three source types:

- **`directory`** - Index files from local filesystem directories
- **`sql`** - Index data from SQL databases (PostgreSQL, MySQL, SQLite)
- **`api`** - Index data from REST API endpoints

### Agent Assignment

Agents can be assigned to specific document stores to control access:

```yaml
document_stores:
  knowledge_base:
    source: sql
    # ... config ...
  internal_docs:
    source: directory
    # ... config ...

agents:
  # Access all stores (permissive default)
  general_assistant:
    # document_stores: not specified → accesses all stores
  
  # Scoped access (explicit assignment)
  security_agent:
    document_stores:
      - "knowledge_base"      # Only these stores
      - "internal_docs"
  
  # No access (explicit restriction)
  isolated_agent:
    document_stores: []       # Explicitly empty = no access
```

**Access Rules** (consistent pattern across tools, document stores, and sub-agents):

- **`nil`/omitted**: Agent has access to **ALL** registered document stores
- **`[]` (explicitly empty)**: Agent has **NO access** to any document stores
- **`["store1", ...]`**: Agent can **ONLY access** the explicitly listed stores

The search tool is automatically created and scoped to the agent's assigned stores. See [Search Architecture](architecture/search-architecture.md) for details.

!!! note "Consistent Assignment Pattern"
    This same pattern applies to:
    - **Tools** (`agent.tools`) - See [Tools](../core-concepts/tools.md#tool-assignment)
    - **Document Stores** (`agent.document_stores`) - See above
    - **Sub-Agents** (`agent.sub_agents`) - See [Multi-Agent](../core-concepts/multi-agent.md#sub-agent-assignment)
    
    The consistent pattern makes configuration intuitive and predictable.

### MCP Document Parsing

Use MCP (Model Context Protocol) tools for advanced document parsing. This allows using services like Docling for better parsing quality or additional formats.

**Prerequisites:**
1. Configure MCP tools in your `tools` section
2. Start your MCP server (e.g., Docling MCP server)

**Configuration:**

```yaml
tools:
  mcp_tools:
    - server:
        url: "http://localhost:3000"
        protocol: "mcp"
      tools:
        - "parse_document"

document_stores:
  knowledge_base:
    path: "./documents"
    mcp_parsers:
      tool_names: ["parse_document"]  # Required: MCP tool names
      extensions: [".pdf", ".pptx"]    # Optional: Specific formats
      priority: 8                      # Optional: Extractor priority (default: 8)
      prefer_native: false            # Optional: Use native first (default: false)
```

**Options:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `tool_names` | `[]string` | Required | MCP tool names to try (in order) |
| `extensions` | `[]string` | `[]` (all) | File extensions to handle (empty = all binary files) |
| `priority` | `int` | `8` | Extractor priority (higher = preferred, native = 5) |
| `prefer_native` | `bool` | `false` | Use native parsers first, MCP as fallback |

**Use Cases:**

1. **Override native parsers** (better quality):
```yaml
mcp_parsers:
  tool_names: ["parse_document"]
  priority: 10  # Higher than native (5)
```

2. **Use as fallback** (when native fails):
```yaml
mcp_parsers:
  tool_names: ["parse_document"]
  prefer_native: true
  priority: 4  # Lower than native (5)
```

3. **Format-specific** (unsupported formats):
```yaml
mcp_parsers:
  tool_names: ["parse_document"]
  extensions: [".pptx", ".html"]  # Formats not supported natively
```

**Benefits:**
- ✅ Better PDF parsing (layout detection, table extraction, OCR)
- ✅ Additional formats (PPTX, HTML, audio, images)
- ✅ Enhanced metadata extraction
- ✅ Works with any MCP service (not just Docling)

### Directory Source

Index files from local directories:

```yaml
document_stores:
  codebase:
    # Basic configuration
    name: "codebase"              # Required: Store identifier
    source: "directory"           # Required: Source type
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

    # MCP document parsing (optional)
    mcp_parsers:                  # Use MCP tools for document parsing
      tool_names:                 # Required: MCP tool names to use (tried in order)
        - "parse_document"
        - "docling_parse"
      extensions:                 # Optional: File extensions to handle (empty = all binary files)
        - ".pdf"
        - ".pptx"
        - ".html"
      priority: 8                 # Optional: Extractor priority (default: 8, higher = preferred)
      prefer_native: false        # Optional: Use native parsers first, MCP as fallback (default: false)
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

### SQL Source

Index data from SQL databases (PostgreSQL, MySQL, SQLite):

```yaml
# Define SQL database once
databases:
  postgres-main:
    driver: "postgres"
    host: "localhost"
    port: 5432
    database: "mydb"
    username: "user"
    password: "${DB_PASSWORD}"
    ssl_mode: "disable"
    max_conns: 25
    max_idle: 5

# Reference database in document store
document_stores:
  database_content:
    name: "database_content"
    source: "sql"                 # Required: Source type
    database: "postgres-main"     # Reference to databases section
    
    # Tables to index
    sql_tables:
      - table: "articles"          # Required: Table name
        columns:                   # Required: Columns to concatenate as content
          - "title"
          - "content"
        id_column: "id"            # Required: Primary key or unique identifier
        updated_column: "updated_at"  # Optional: Column for tracking updates (enables incremental indexing)
        where_clause: "status = 'published'"  # Optional: WHERE clause for filtering
        metadata_columns:          # Optional: Columns to include as metadata
          - "author"
          - "category"
          - "created_at"
      
      - table: "products"
        columns:
          - "name"
          - "description"
        id_column: "id"
        updated_column: "updated_at"
        metadata_columns:
          - "price"
          - "status"
    
    sql_max_rows: 10000            # Optional: Maximum rows to index per table (default: 10000)
    
    # Chunking configuration (same as directory source)
    chunk_size: 800
    chunk_overlap: 50
    chunk_strategy: "simple"
    
    # Incremental indexing (requires updated_column)
    incremental_indexing: true
```

**Supported Databases:**

| Driver | Description | Connection String Format |
|--------|-------------|------------------------|
| `postgres` or `pgx` | PostgreSQL | `host=... port=... user=... password=... dbname=... sslmode=...` |
| `mysql` | MySQL/MariaDB | `user:password@tcp(host:port)/database` |
| `sqlite3` | SQLite | File path (database field) |

**Example - SQLite:**
```yaml
databases:
  content-db:
    driver: "sqlite3"
    database: "./data/content.db"

document_stores:
  local_db:
    name: "local_db"
    source: "sql"
    sql_database: "content-db"
    sql_tables:
      - table: "articles"
        columns: ["title", "content"]
        id_column: "id"
        updated_column: "updated_at"
        metadata_columns: ["author", "category"]
```

**Example - PostgreSQL:**
```yaml
databases:
  content-db:
    driver: "postgres"
    host: "db.example.com"
    port: 5432
    database: "content_db"
    username: "${DB_USER}"
    password: "${DB_PASSWORD}"
    ssl_mode: "require"

document_stores:
  production_db:
    name: "production_db"
    source: "sql"
    sql_database: "content-db"
    sql_tables:
      - table: "articles"
        columns: ["title", "body"]
        id_column: "id"
        updated_column: "updated_at"
        where_clause: "published = true"
        metadata_columns: ["author_id", "category_id"]
```

### API Source

Index data from REST API endpoints:

```yaml
document_stores:
  api_content:
    name: "api_content"
    source: "api"                  # Required: Source type
    
    # API configuration
    api:
      base_url: "https://api.example.com"  # Required: Base URL for API
      
      # Global authentication (applied to all endpoints)
      auth:
        type: "bearer"             # "bearer", "basic", or "apikey"
        token: "${API_TOKEN}"      # Token/API key
        # For basic auth:
        # user: "username"
        # pass: "password"
        # For apikey:
        # header: "X-API-Key"      # Header name (default: "X-API-Key")
      
      # Endpoints to index
      endpoints:
        - path: "/articles"        # Required: API path (relative to base_url)
          method: "GET"            # Optional: HTTP method (default: "GET")
          
          # Query parameters
          params:                   # Optional: Query parameters
            status: "published"
            limit: "100"
          
          # Custom headers
          headers:                  # Optional: Additional headers
            X-Custom-Header: "value"
          
          # Request body (for POST/PUT)
          body: '{"filter": "active"}'  # Optional: Request body
          
          # Document extraction
          id_field: "id"           # Required: JSON field to use as document ID
          content_field: "title,content"  # Required: JSON field(s) for content (comma-separated)
          metadata_fields:         # Optional: JSON fields to include as metadata
            - "author"
            - "category"
            - "published_at"
          updated_field: "updated_at"  # Optional: JSON field for last modified (enables incremental indexing)
          
          # Endpoint-specific authentication (overrides global)
          auth:
            type: "bearer"
            token: "${ENDPOINT_TOKEN}"
        
        - path: "/products"
          method: "GET"
          id_field: "id"
          content_field: "name,description"
          metadata_fields: ["price", "status"]
          
          # Pagination support
          pagination:
            type: "page"           # "page", "offset", "cursor", or "link"
            page_param: "page"     # Query parameter name for page number
            size_param: "size"     # Query parameter name for page size
            page_size: 50          # Items per page
            max_pages: 100         # Maximum pages to fetch (0 = unlimited)
            data_field: "items"    # Optional: JSON field containing array (if nested)
            next_field: "next"     # Optional: JSON field containing next page URL/cursor
```

**Authentication Types:**

| Type | Description | Required Fields |
|------|-------------|-----------------|
| `bearer` | Bearer token authentication | `token` |
| `basic` | HTTP Basic authentication | `user`, `pass` |
| `apikey` | API key in header | `token`, `header` (optional, defaults to "X-API-Key") |

**Pagination Types:**

| Type | Description | Use Case |
|------|-------------|----------|
| `page` | Page-based pagination | APIs using `?page=1&size=50` |
| `offset` | Offset-based pagination | APIs using `?offset=0&limit=50` |
| `cursor` | Cursor-based pagination | APIs using cursor tokens |
| `link` | Link-based pagination | APIs returning `next` URL in response |

**Example - Simple API:**
```yaml
document_stores:
  blog_api:
    name: "blog_api"
    source: "api"
    api:
      base_url: "https://api.blog.com"
      auth:
        type: "bearer"
        token: "${BLOG_API_TOKEN}"
      endpoints:
        - path: "/posts"
          method: "GET"
          id_field: "id"
          content_field: "title,body"
          metadata_fields: ["author", "published_at"]
```

**Example - Paginated API:**
```yaml
document_stores:
  products_api:
    name: "products_api"
    source: "api"
    api:
      base_url: "https://api.store.com"
      auth:
        type: "apikey"
        token: "${STORE_API_KEY}"
        header: "X-API-Key"
      endpoints:
        - path: "/products"
          method: "GET"
          id_field: "id"
          content_field: "name,description"
          metadata_fields: ["price", "category"]
          pagination:
            type: "page"
            page_param: "page"
            size_param: "per_page"
            page_size: 100
            max_pages: 50
```

**Example - Multiple Endpoints:**
```yaml
document_stores:
  multi_api:
    name: "multi_api"
    source: "api"
    api:
      base_url: "https://api.example.com"
      auth:
        type: "bearer"
        token: "${API_TOKEN}"
      endpoints:
        - path: "/articles"
          id_field: "id"
          content_field: "title,content"
        - path: "/docs"
          id_field: "id"
          content_field: "title,body"
        - path: "/faq"
          id_field: "id"
          content_field: "question,answer"
```

### Common Configuration (All Sources)

All document store sources support these common options:

```yaml
document_stores:
  my_store:
    # Chunking configuration
    chunk_size: 800               # Default: 800 characters per chunk
    chunk_overlap: 0              # Default: 0 characters overlap
    chunk_strategy: "simple"      # Options: "simple", "overlapping", "semantic"
    
    # MCP document parsing (optional)
    mcp_parsers:                  # Use MCP tools for document parsing
      tool_names:                 # Required: MCP tool names to use (tried in order)
        - "parse_document"
        - "docling_parse"
      extensions:                 # Optional: File extensions to handle (empty = all binary files)
        - ".pdf"
        - ".pptx"
        - ".html"
      priority: 8                 # Optional: Extractor priority (default: 8, higher = preferred)
      prefer_native: false        # Optional: Use native parsers first, MCP as fallback (default: false)
    
    # Incremental indexing (SQL and API sources)
    incremental_indexing: true    # Default: true - Only reindex changed documents
    
    # Metadata extraction (directory source only)
    extract_metadata: false       # Default: false - Extract code metadata
    metadata_languages:           # Languages for metadata extraction
      - "go"
      - "python"
```

**Example - Minimal (Directory):**
```yaml
document_stores:
  docs:
    path: "./documentation"
```

**Example - Advanced (Directory):**
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

**Example - With MCP Parsing:**
```yaml
tools:
  mcp_tools:
    - server:
        url: "http://localhost:3000"
        protocol: "mcp"
      tools:
        - "parse_document"

document_stores:
  knowledge_base:
    path: "./documents"
    mcp_parsers:
      tool_names: ["parse_document"]
      extensions: [".pdf", ".pptx"]  # Only these formats
      priority: 10                     # Override native parsers
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

# Vector Store
vector_stores:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334

# SQL Database
databases:
  postgres-main:
    driver: "postgres"
    host: "localhost"
    port: 5432
    database: "hector_main"
    username: "user"
    password: "${DB_PASSWORD}"

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
    vector_store: "qdrant"
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
```

**Note:** For detailed logging, redirect Hector's stdout/stderr to your logging system (e.g., systemd, Docker logs, etc.).

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

## Prompt Configuration Details

For detailed information about prompts, see [Prompts Guide](../core-concepts/prompts.md).

### Quick Summary

**Priority Hierarchy (highest to lowest):**
1. **system_prompt** - Complete override, disables ALL strategy behavior
2. **prompt_slots** - Merges with strategy, preserves behavior patterns
3. **strategy defaults** - Optimized prompts for each reasoning engine

**Prompt Slots Explained:**

| Slot | Purpose | Use When | Priority |
|------|---------|----------|----------|
| `system_role` | WHO the agent is | Want custom identity | Replaces strategy's role |
| `instructions` | HOW the agent behaves | Need different workflows | Replaces strategy's instructions |
| `user_guidance` | WHAT the user wants | Task-specific needs | Highest (applied last) |

**Recommendations:**
- ✅ **Most users**: Use `user_guidance` only - preserves strategy optimizations
- ⚠️ **Advanced users**: Use `system_role` to change identity, keep `instructions` empty
- 🚫 **Rarely needed**: Use `instructions` only if strategy patterns don't fit
- 🚫 **Special cases**: Use `system_prompt` only for complete custom behavior

**Examples:**

```yaml
# RECOMMENDED: Add user guidance (preserves strategy)
agents:
  assistant:
    prompt:
      prompt_slots:
        user_guidance: "Focus on security. Always validate input."

# ADVANCED: Custom identity + guidance
agents:
  assistant:
    prompt:
      prompt_slots:
        system_role: "You are a Python expert"
        user_guidance: "Use type hints. Prefer async/await."

# SPECIAL: Complete override (loses strategy optimizations)
agents:
  simple_bot:
    prompt:
      system_prompt: "You are a calculator. Only output numbers."
```

---

## Next Steps

- **[Prompts Guide](../core-concepts/prompts.md)** - Complete prompt documentation
- **[CLI Reference](cli.md)** - Command-line options (--role, --instruction flags)
- **[API Reference](api.md)** - HTTP/gRPC APIs
- **[Core Concepts](../core-concepts/overview.md)** - Understanding configuration
- **[Examples](https://github.com/kadirpekel/hector/tree/main/configs)** - Example configurations

---

## Related Topics

- **[Getting Started](../getting-started/installation.md)** - Installation
- **[Agent Overview](../core-concepts/overview.md)** - Agent basics
- **[Architecture](../reference/architecture.md)** - System design and deployment

