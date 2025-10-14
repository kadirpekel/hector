# Hector Configuration Reference

Complete configuration reference for Hector AI Agent Platform.

## Table of Contents

- [Configuration Structure](#configuration-structure)
- [Agents](#agents)
  - [Native Agents](#native-agents)
  - [External A2A Agents](#external-a2a-agents)
- [LLM Providers](#llm-providers)
- [Tools](#tools)
- [Memory Configuration](#memory-configuration)
- [Reasoning Configuration](#reasoning-configuration)
- [Document Stores](#document-stores)
- [Database Providers](#database-providers)
- [Embedder Providers](#embedder-providers)
- [Security Configuration](#security-configuration)
- [Task Configuration](#task-configuration)
- [Global Configuration](#global-configuration)
- [Plugins](#plugins)

---

## Configuration Structure

```yaml
# Top-level configuration sections
agents:           # Agent definitions (required)
llms:             # LLM provider configurations (required)
tools:            # Tool configurations (optional, has defaults)
databases:        # Database provider configurations (optional)
embedders:        # Embedder provider configurations (optional)
document_stores:  # Document store configurations (optional)
plugins:          # Plugin configurations (optional)
global:           # Global server settings (optional)
```

---

## Agents

### Native Agents

Native agents run locally with full configuration control.

```yaml
agents:
  <agent-id>:
    # Required fields
    name: string                      # Display name
    llm: string                       # LLM provider reference
    
    # Optional identity fields
    type: "native"                    # Agent type (default: "native")
    description: string               # Agent description
    visibility: string                # "public" (default), "internal", or "private"
    
    # Configuration sections
    prompt: PromptConfig              # Prompt customization
    memory: MemoryConfig              # Memory/conversation management
    reasoning: ReasoningConfig        # Reasoning strategy
    search: SearchConfig              # Search configuration
    task: TaskConfig                  # Task management
    security: SecurityConfig          # Security settings
    
    # Resource references
    tools: []string                   # Tool IDs to enable
    document_stores: []string         # Document store IDs for RAG
    database: string                  # Database provider reference (required if document_stores set)
    embedder: string                  # Embedder provider reference (required if document_stores set)
    session_store: string             # Session store reference (default: "default")
    
    # Multi-agent orchestration
    sub_agents: []string              # Agent IDs this agent can orchestrate (empty = all)
```

#### Visibility Levels

- `public` - Discoverable via `/agents` API and callable by anyone
- `internal` - Not listed in discovery but callable if agent ID known
- `private` - Only callable by local orchestrators, not via external API

#### Example: Minimal Native Agent

```yaml
agents:
  assistant:
    name: "My Assistant"
    llm: "main-llm"
```

#### Example: Full Native Agent

```yaml
agents:
  coding_assistant:
    name: "Coding Assistant"
    description: "Helps with code writing and debugging"
    type: "native"
    visibility: "public"
    llm: "gpt-4o"
    
    prompt:
      prompt_slots:
        system_role: "You are an expert software engineer"
        reasoning_instructions: "Think step-by-step and write clean code"
    
    memory:
      strategy: "summary_buffer"
      budget: 2000
      threshold: 0.8
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100
      enable_streaming: true
      show_tool_execution: true
    
    tools:
      - "execute_command"
      - "write_file"
      - "search_replace"
      - "search"
    
    document_stores:
      - "codebase"
    database: "qdrant"
    embedder: "ollama"
    
    search:
      top_k: 5
      threshold: 0.7
```

### External A2A Agents

External agents are A2A-compliant services accessed via URL.

```yaml
agents:
  <agent-id>:
    # Required fields
    type: "a2a"                       # Agent type
    name: string                      # Display name
    url: string                       # A2A agent URL
    
    # Optional fields
    description: string               # Agent description
    visibility: string                # "public", "internal", or "private"
    credentials: AgentCredentials     # Authentication for calling this agent
```

#### AgentCredentials

```yaml
credentials:
  type: string                        # "bearer", "api_key", or "basic"
  
  # For bearer auth
  token: string                       # JWT token
  
  # For API key auth
  api_key: string                     # API key
  api_key_header: string              # Header name (default: "X-API-Key")
  
  # For basic auth
  username: string                    # Username
  password: string                    # Password
```

#### Example: External Agent

```yaml
agents:
  weather_service:
    type: "a2a"
    name: "Weather Service"
    url: "https://api.weather.com/agents/forecast"
    visibility: "public"
    credentials:
      type: "bearer"
      token: "${WEATHER_API_TOKEN}"
```

---

## LLM Providers

### OpenAI

```yaml
llms:
  <llm-id>:
    type: "openai"
    model: string                     # e.g. "gpt-4o", "gpt-4o-mini"
    api_key: string                   # API key (use env var: "${OPENAI_API_KEY}")
    host: string                      # Default: "https://api.openai.com/v1"
    temperature: float                # 0.0-2.0, default: 0.7
    max_tokens: int                   # Default: 8000
    timeout: int                      # Request timeout in seconds, default: 60
    max_retries: int                  # Rate limit retry attempts, default: 5
    retry_delay: int                  # Base retry delay in seconds, default: 2
    
    # Structured output (optional)
    structured_output: StructuredOutputConfig
```

**Supported models:** `gpt-4o`, `gpt-4o-mini`, `gpt-4-turbo`, `gpt-3.5-turbo`

### Anthropic

```yaml
llms:
  <llm-id>:
    type: "anthropic"
    model: string                     # e.g. "claude-3-7-sonnet-latest"
    api_key: string                   # API key (use env var: "${ANTHROPIC_API_KEY}")
    host: string                      # Default: "https://api.anthropic.com"
    temperature: float                # 0.0-1.0, default: 0.7
    max_tokens: int                   # Default: 8000
    timeout: int                      # Default: 60
    max_retries: int                  # Default: 5
    retry_delay: int                  # Default: 2
    
    # Structured output (optional)
    structured_output: StructuredOutputConfig
```

**Supported models:** `claude-3-7-sonnet-latest`, `claude-3-5-sonnet-latest`, `claude-3-5-haiku-latest`, `claude-sonnet-4.5-20250514`

### Gemini

```yaml
llms:
  <llm-id>:
    type: "gemini"
    model: string                     # e.g. "gemini-2.0-flash-exp"
    api_key: string                   # API key (use env var: "${GEMINI_API_KEY}")
    host: string                      # Default: "https://generativelanguage.googleapis.com"
    temperature: float                # 0.0-2.0, default: 0.7
    max_tokens: int                   # Default: 8000
    timeout: int                      # Default: 60
    max_retries: int                  # Default: 5
    retry_delay: int                  # Default: 2
    
    # Structured output (optional)
    structured_output: StructuredOutputConfig
```

**Supported models:** `gemini-2.0-flash-exp`, `gemini-1.5-pro`, `gemini-1.5-flash`

### StructuredOutputConfig

```yaml
structured_output:
  format: string                      # "json", "xml", or "enum"
  
  # For JSON format
  schema: object                      # JSON schema definition
  
  # For enum format
  enum: []string                      # List of allowed values
  
  # Provider-specific options
  prefill: string                     # Anthropic: prefill string
  property_ordering: []string         # Gemini: property order
```

---

## Tools

Tools are defined globally and referenced by agents.

### Built-in Tool Types

#### Command Tool

```yaml
tools:
  <tool-id>:
    type: "command"
    enabled: bool                     # Default: true
    allowed_commands: []string        # Whitelist of commands
    working_directory: string         # Default: "./"
    max_execution_time: string        # Duration, default: "30s"
    enable_sandboxing: bool           # Default: true
```

#### File Writer Tool

```yaml
tools:
  <tool-id>:
    type: "write_file"
    enabled: bool                     # Default: true
    max_file_size: int64              # Bytes, default: 1048576 (1MB)
    allowed_extensions: []string      # e.g. [".go", ".py", ".md"]
    forbidden_paths: []string         # Paths to block
```

#### Search Replace Tool

```yaml
tools:
  <tool-id>:
    type: "search_replace"
    enabled: bool                     # Default: true
    max_replacements: int             # Default: 100
    backup_enabled: bool              # Create backup before replace
    working_directory: string         # Default: "./"
```

#### Search Tool

```yaml
tools:
  <tool-id>:
    type: "search"
    enabled: bool                     # Default: true
    document_stores: []string         # Document store IDs to search
    default_limit: int                # Default: 10
    max_limit: int                    # Default: 50
    max_results: int                  # Default: 100
```

#### Todo Tool

```yaml
tools:
  <tool-id>:
    type: "todo"
    enabled: bool                     # Default: true
```

#### Agent Call Tool

```yaml
tools:
  <tool-id>:
    type: "agent_call"
    enabled: bool                     # Default: true
```

**Note:** `agent_call` tool requires agent registry and is automatically configured when needed.

#### MCP Tool

```yaml
tools:
  <tool-id>:
    type: "mcp"
    enabled: bool                     # Default: true
    server_url: string                # MCP server URL
    description: string               # Tool description
```

### Default Tools

If no tools are configured, these defaults are provided:

```yaml
tools:
  execute_command:
    type: "command"
    allowed_commands: ["ls", "cat", "head", "tail", "pwd", "find", "grep", "wc", "date", "echo", "tree", "du", "df"]
  todo_write:
    type: "todo"
```

---

## Memory Configuration

Memory controls conversation history and context management.

```yaml
memory:
  # Strategy selection
  strategy: string                    # "buffer_window" or "summary_buffer" (default)
  
  # Working memory settings
  budget: int                         # Token budget, default: 2000
  
  # Buffer window settings (for buffer_window strategy)
  window_size: int                    # Number of messages to keep, default: 20
  
  # Summary buffer settings (for summary_buffer strategy)
  threshold: float                    # Trigger at % of budget, default: 0.8
  target: float                       # Compress to % of budget, default: 0.6
  
  # Long-term memory (optional)
  long_term: LongTermMemoryConfig
```

### LongTermMemoryConfig

```yaml
long_term:
  storage_scope: string               # "all", "conversational", or "summaries_only"
  batch_size: int                     # Batch size for storage, default: 1 (immediate)
  auto_recall: bool                   # Auto-inject memories, default: true
  recall_limit: int                   # Max memories to recall, default: 5
  collection: string                  # Qdrant collection name, default: "hector_session_memory"
```

---

## Reasoning Configuration

```yaml
reasoning:
  engine: string                      # "chain-of-thought" or "supervisor"
  max_iterations: int                 # Safety valve, default: 100
  
  # Output control
  enable_streaming: bool              # Enable streaming, default: true
  show_debug_info: bool               # Show iteration info, default: false
  show_tool_execution: bool           # Show tool labels, default: true
  show_thinking: bool                 # Show internal reasoning, default: false
  
  # Advanced features
  enable_self_reflection: bool        # Enable self-reflection, default: false
  enable_meta_reasoning: bool         # Enable meta-reasoning, default: false
  enable_goal_evolution: bool         # Enable goal evolution, default: false
  enable_dynamic_tools: bool          # Enable dynamic tools, default: false
  enable_structured_reflection: bool  # LLM-based reflection, default: true
  enable_completion_verification: bool # Task completion check, default: false
  enable_goal_extraction: bool        # Goal decomposition (supervisor), default: false
  
  # Quality threshold
  quality_threshold: float            # 0.0-1.0, default: 0.7
```

### Reasoning Engines

- **chain-of-thought** - Iterative reasoning with natural termination (default)
- **supervisor** - Optimized for multi-agent orchestration

---

## Prompt Configuration

```yaml
prompt:
  # Slot-based customization (recommended)
  prompt_slots:
    system_role: string               # Who the assistant is
    reasoning_instructions: string    # How to think
    tool_usage: string                # How to use tools
    output_format: string             # Response formatting
    communication_style: string       # How to communicate
    additional: string                # Extra instructions
  
  # Full override (bypasses slots)
  system_prompt: string               # Complete custom prompt
  
  # Include flags
  include_tools: bool                 # Include tool descriptions
  include_context: bool               # Include semantic search results
  include_history: bool               # Include conversation history
  
  # Deprecated fields (use memory: section instead)
  max_history_messages: int           # Deprecated: use memory.budget
  max_context_length: int             # Max context length
  enable_summarization: bool          # Deprecated: use memory.strategy
  summarize_threshold: float          # Deprecated: use memory.threshold
  smart_memory: bool                  # Deprecated: use memory
  memory_budget: int                  # Deprecated: use memory.budget
```

---

## Search Configuration

```yaml
search:
  models: []SearchModel               # Search model configurations
  top_k: int                          # Top K results, default: 5
  threshold: float                    # Similarity threshold, default: 0.7
  max_context_length: int             # Max context length, default: 4000
```

### SearchModel

```yaml
models:
  - name: string                      # Model name
    collection: string                # Vector collection name
    default_top_k: int                # Default top K, default: 10
    max_top_k: int                    # Maximum top K, default: 100
```

---

## Document Stores

```yaml
document_stores:
  <store-id>:
    name: string                      # Store name
    source: string                    # Source type, default: "directory"
    path: string                      # Source path
    
    include_patterns: []string        # Include glob patterns
    exclude_patterns: []string        # Exclude glob patterns
    
    watch_changes: bool               # Auto-reindex on changes
    max_file_size: int64              # Max file size in bytes, default: 10485760 (10MB)
```

**Default include patterns:** `["*.md", "*.txt", "*.go", "*.py", "*.js", "*.ts", "*.yaml", "*.yml"]`

**Default exclude patterns:** `["**/node_modules/**", "**/.git/**", "**/vendor/**", "**/__pycache__/**"]`

---

## Database Providers

### Qdrant

```yaml
databases:
  <db-id>:
    type: "qdrant"
    host: string                      # Default: "localhost"
    port: int                         # Default: 6333
    api_key: string                   # Optional API key
    timeout: int                      # Connection timeout in seconds, default: 30
    use_tls: bool                     # Use TLS connection, default: false
    insecure: bool                    # Skip TLS verification, default: false
```

---

## Embedder Providers

### Ollama

```yaml
embedders:
  <embedder-id>:
    type: "ollama"
    model: string                     # e.g. "nomic-embed-text"
    host: string                      # Default: "http://localhost:11434"
    dimension: int                    # Embedding dimension, default: 768
    timeout: int                      # Request timeout in seconds, default: 30
    max_retries: int                  # Max retry attempts, default: 3
```

---

## Security Configuration

```yaml
security:
  schemes: map[string]SecurityScheme  # Security scheme definitions
  require: []map[string][]string      # Security requirements
  jwks_url: string                    # JWKS URL for JWT validation
  issuer: string                      # Expected JWT issuer
  audience: string                    # Expected JWT audience
```

### SecurityScheme

```yaml
schemes:
  <scheme-name>:
    type: string                      # "http", "apiKey", "oauth2", "openIdConnect", "mutualTLS"
    scheme: string                    # For HTTP: "bearer" or "basic"
    bearer_format: string             # For bearer: "JWT"
    description: string               # Human-readable description
    
    # For API Key auth
    in: string                        # "header", "query", or "cookie"
    name: string                      # Parameter name
```

---

## Task Configuration

```yaml
task:
  backend: string                     # "memory" (default) or "sql"
  worker_pool: int                    # Max concurrent tasks, default: 100
  
  # SQL backend configuration
  sql: TaskSQLConfig
```

### TaskSQLConfig

```yaml
sql:
  driver: string                      # "postgres", "mysql", or "sqlite"
  host: string                        # Database host (not for sqlite)
  port: int                           # Database port (not for sqlite)
  database: string                    # Database name or file path (sqlite)
  username: string                    # Username (not for sqlite)
  password: string                    # Password (not for sqlite)
  ssl_mode: string                    # SSL mode for postgres: "disable", "require", "verify-ca", "verify-full"
  max_conns: int                      # Max connections, default: 25
  max_idle: int                       # Max idle connections, default: 5
```

---

## Global Configuration

```yaml
global:
  # A2A server configuration
  a2a_server:
    host: string                      # Server host, default: "0.0.0.0"
    port: int                         # gRPC port, default: 8080
    base_url: string                  # Base URL for discovery
  
  # Authentication
  auth:
    jwks_url: string                  # JWKS URL for JWT validation
    issuer: string                    # Expected JWT issuer
    audience: string                  # Expected JWT audience
  
  # Logging
  logging:
    level: string                     # "debug", "info", "warn", "error"
    format: string                    # "text" or "json"
    output: string                    # "stdout", "stderr", or "file"
  
  # Performance
  performance:
    max_concurrency: int              # Max concurrency, default: 4
    timeout: duration                 # Global timeout, default: 15m
```

---

## Plugins

```yaml
plugins:
  # Plugin discovery
  plugin_discovery:
    enabled: bool                     # Enable auto-discovery, default: true
    paths: []string                   # Discovery paths, default: ["./plugins", "~/.hector/plugins"]
    scan_subdirectories: bool         # Scan subdirectories
  
  # Plugin definitions by category
  llm_providers:
    <plugin-id>: PluginConfig
  
  database_providers:
    <plugin-id>: PluginConfig
  
  embedder_providers:
    <plugin-id>: PluginConfig
  
  tool_providers:
    <plugin-id>: PluginConfig
  
  reasoning_strategies:
    <plugin-id>: PluginConfig
```

### PluginConfig

```yaml
<plugin-id>:
  name: string                        # Plugin name
  type: string                        # Must be "grpc"
  path: string                        # Path to plugin executable
  enabled: bool                       # Whether plugin is enabled
  config: map[string]interface{}     # Plugin-specific configuration
```

---

## Environment Variables

Hector supports environment variable expansion using `${VAR_NAME}` syntax:

```yaml
llms:
  main-llm:
    api_key: "${OPENAI_API_KEY}"      # Expands to value of OPENAI_API_KEY

databases:
  qdrant:
    host: "${QDRANT_HOST:-localhost}" # With default value
```

**Common environment variables:**
- `OPENAI_API_KEY` - OpenAI API key
- `ANTHROPIC_API_KEY` - Anthropic API key
- `GEMINI_API_KEY` - Google Gemini API key
- `QDRANT_HOST` - Qdrant server host
- `OLLAMA_HOST` - Ollama server host

---

## Complete Example

```yaml
# Full configuration example
agents:
  assistant:
    name: "Coding Assistant"
    description: "Expert software engineer"
    llm: "gpt-4o"
    
    prompt:
      prompt_slots:
        system_role: "You are an expert software engineer"
        reasoning_instructions: "Think step-by-step and write clean code"
    
    memory:
      strategy: "summary_buffer"
      budget: 2000
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100
      enable_streaming: true
    
    tools:
      - "execute_command"
      - "write_file"
      - "search"
    
    document_stores:
      - "codebase"
    database: "qdrant"
    embedder: "ollama"

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
    max_tokens: 8000

tools:
  execute_command:
    type: "command"
    allowed_commands: ["ls", "cat", "grep", "git"]
  write_file:
    type: "write_file"
  search:
    type: "search"
    document_stores: ["codebase"]

databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6333

embedders:
  ollama:
    type: "ollama"
    model: "nomic-embed-text"
    host: "http://localhost:11434"

document_stores:
  codebase:
    name: "codebase"
    path: "."
    include_patterns: ["*.go", "*.md"]
    exclude_patterns: ["**/vendor/**", "**/.git/**"]

global:
  a2a_server:
    host: "0.0.0.0"
    port: 8080
  logging:
    level: "info"
    format: "text"
```

---

For more examples, see the [configs/](../configs/) directory.
