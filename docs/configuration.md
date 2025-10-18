---
title: Configuration Reference
description: Complete YAML configuration reference for Hector AI Agent Platform
---

# Hector Configuration Reference

Complete configuration reference for Hector AI Agent Platform.

## 📋 Table of Contents

- [Configuration Structure](#configuration-structure)
- [Agents](#agents)
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

- :material-earth: **`public`** - Discoverable via `/agents` API and callable by anyone
- :material-account-group: **`internal`** - Not listed in discovery but callable if agent ID known
- :material-lock: **`private`** - Only callable by local orchestrators, not via external API

#### Example: Minimal Native Agent

```yaml
agents:
  assistant:
    name: "My Assistant"
    llm: "main-llm"
```

#### Example: Full-Featured Agent

```yaml
agents:
  coding_assistant:
    name: "Coding Assistant"
    description: "Helps with coding tasks and code review"
    visibility: "public"
    
    # Core configuration
    llm: "gpt-4o"
    
    # Prompt customization
    prompt:
      system_role: |
        You are an expert software engineer who writes clean,
        maintainable code and explains your reasoning clearly.
      
      reasoning_instructions: |
        1. Understand the requirement fully
        2. Consider edge cases
        3. Write clean, testable code
        4. Explain your decisions
      
      tool_usage: |
        Use write_file to create/update files
        Use execute_command to run tests
        Use search to find relevant documentation
    
    # Memory configuration
    memory:
      strategy: "summary_buffer"
      max_tokens: 4000
      long_term:
        enabled: true
        strategy: "vector_memory"
    
    # Reasoning configuration
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 15
      enable_streaming: true
    
    # Search configuration
    search:
      enabled: true
      result_limit: 10
      min_similarity: 0.7
    
    # Resource references
    tools: ["execute_command", "write_file", "search_replace", "search"]
    document_stores: ["codebase_docs", "api_reference"]
    database: "qdrant"
    embedder: "ollama"
```

### External A2A Agents

External agents connect to remote A2A-compliant services.

```yaml
agents:
  <agent-id>:
    type: "external"
    name: string                      # Display name
    description: string               # Agent description
    visibility: string                # "public", "internal", or "private"
    
    # External connection
    endpoint: string                  # A2A service endpoint
    auth:
      type: "jwt"                     # Authentication type
      token: string                   # Authentication token
    
    # Optional configuration
    timeout: string                   # Request timeout (default: "30s")
    retries: int                      # Retry attempts (default: 3)
```

#### Example: External Agent

```yaml
agents:
  external_assistant:
    type: "external"
    name: "External Assistant"
    description: "Connects to remote A2A service"
    visibility: "public"
    endpoint: "https://api.external-service.com/a2a"
    auth:
      type: "jwt"
      token: "${EXTERNAL_SERVICE_TOKEN}"
    timeout: "60s"
    retries: 5
```

---

## 🧠 LLM Providers

Configure language model providers for your agents.

### OpenAI

```yaml
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
    max_tokens: 4000
    timeout: "30s"
    
  gpt-4o-mini:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
    max_tokens: 2000
    timeout: "30s"
```

### Anthropic

```yaml
llms:
  claude-3-5-sonnet:
    type: "anthropic"
    model: "claude-3-5-sonnet-20241022"
    api_key: "${ANTHROPIC_API_KEY}"
    temperature: 0.7
    max_tokens: 8000
    timeout: "30s"
    
  claude-3-haiku:
    type: "anthropic"
    model: "claude-3-haiku-20240307"
    api_key: "${ANTHROPIC_API_KEY}"
    temperature: 0.7
    max_tokens: 4000
    timeout: "30s"
```

### Google Gemini

```yaml
llms:
  gemini-2-0-flash:
    type: "gemini"
    model: "gemini-2.0-flash"
    api_key: "${GEMINI_API_KEY}"
    temperature: 0.7
    max_tokens: 8000
    timeout: "30s"
    
  gemini-1-5-pro:
    type: "gemini"
    model: "gemini-1.5-pro"
    api_key: "${GEMINI_API_KEY}"
    temperature: 0.7
    max_tokens: 8000
    timeout: "30s"
```

### Custom LLM Plugin

```yaml
llms:
  custom-llm:
    type: "plugin:my_custom_llm"
    model: "custom-model-v1"
    config:
      endpoint: "http://localhost:8080"
      api_key: "${CUSTOM_API_KEY}"
```

---

## Tools

Configure built-in tools and custom tool plugins.

### Built-in Tools

```yaml
tools:
  # Command execution
  execute_command:
    type: "command"
    enabled: true
    allowed_commands:
      - "cat"
      - "ls"
      - "grep"
      - "git"
      - "npm"
      - "go"
    max_execution_time: "30s"
    working_directory: "./"
  
  # File operations
  write_file:
    type: "write_file"
    enabled: true
    allowed_paths:
      - "./src/"
      - "./docs/"
    max_file_size: "10MB"
  
  # Search and replace
  search_replace:
    type: "search_replace"
    enabled: true
    allowed_paths:
      - "./src/"
      - "./docs/"
  
  # Task management
  todo:
    type: "todo"
    enabled: true
  
  # Agent communication
  agent_call:
    type: "agent_call"
    enabled: true
    allowed_agents: []  # Empty = all agents
```

### Custom Tool Plugins

```yaml
tools:
  custom_tool:
    type: "plugin:my_custom_tool"
    enabled: true
    config:
      endpoint: "http://localhost:8081"
      api_key: "${CUSTOM_TOOL_API_KEY}"
```

---

## 🧠 Memory Configuration

Configure memory strategies for conversation context and long-term knowledge.

### Memory Strategies

=== "Summary Buffer"
    ```yaml
    memory:
      strategy: "summary_buffer"
      max_tokens: 4000
      summary_threshold: 0.8
      long_term:
        enabled: true
        strategy: "vector_memory"
    ```

=== "Buffer Window"
    ```yaml
    memory:
      strategy: "buffer_window"
      max_tokens: 4000
      window_size: 10
      long_term:
        enabled: true
        strategy: "vector_memory"
    ```

=== "Vector Memory"
    ```yaml
    memory:
      strategy: "vector_memory"
      max_tokens: 4000
      collection: "agent_memory"
      similarity_threshold: 0.7
    ```

### Memory Configuration Options

```yaml
memory:
  # Strategy selection
  strategy: string                    # "summary_buffer", "buffer_window", "vector_memory"
  
  # Token management
  max_tokens: int                     # Maximum tokens in working memory
  summary_threshold: float            # When to trigger summarization (0.0-1.0)
  
  # Buffer window specific
  window_size: int                    # Number of messages to keep
  
  # Vector memory specific
  collection: string                  # Vector database collection name
  similarity_threshold: float         # Similarity threshold for retrieval
  
  # Long-term memory
  long_term:
    enabled: bool                     # Enable long-term memory
    strategy: string                  # Long-term memory strategy
    collection: string                # Long-term memory collection
```

---

## 🧠 Reasoning Configuration

Configure reasoning strategies for agent decision-making.

### Chain-of-Thought (Default)

```yaml
reasoning:
  engine: "chain-of-thought"
  max_iterations: 10
  enable_streaming: true
  temperature: 0.7
  max_tokens: 4000
```

### Supervisor (For Orchestration)

```yaml
reasoning:
  engine: "supervisor"
  max_iterations: 20
  enable_streaming: true
  temperature: 0.7
  max_tokens: 4000
```

### Reasoning Configuration Options

```yaml
reasoning:
  # Engine selection
  engine: string                      # "chain-of-thought", "supervisor"
  
  # Iteration control
  max_iterations: int                 # Maximum reasoning iterations
  
  # Output control
  enable_streaming: bool              # Enable real-time streaming
  temperature: float                  # LLM temperature (0.0-2.0)
  max_tokens: int                     # Maximum tokens per iteration
  
  # Custom reasoning (future)
  custom_strategy: string             # Custom reasoning strategy
```

---

## Document Stores

Configure document stores for RAG (Retrieval-Augmented Generation).

### Document Store Configuration

```yaml
document_stores:
  company_docs:
    type: "qdrant"
    collection: "company_documents"
    database: "qdrant"
    embedder: "ollama"
    
  api_reference:
    type: "qdrant"
    collection: "api_docs"
    database: "qdrant"
    embedder: "ollama"
    
  codebase_index:
    type: "qdrant"
    collection: "codebase"
    database: "qdrant"
    embedder: "ollama"
```

### Document Store Options

```yaml
document_stores:
  <store-id>:
    # Store type
    type: string                      # "qdrant", "custom"
    
    # Collection configuration
    collection: string                # Collection name
    
    # Resource references
    database: string                  # Database provider reference
    embedder: string                  # Embedder provider reference
    
    # Optional configuration
    config:
      # Store-specific configuration
      similarity_threshold: float      # Similarity threshold for retrieval
      max_results: int                 # Maximum results per query
```

---

## Database Providers

Configure vector database providers for document storage and retrieval.

### Qdrant

```yaml
databases:
  qdrant:
    type: "qdrant"
    url: "http://localhost:6333"
    api_key: "${QDRANT_API_KEY}"
    timeout: "30s"
    
  qdrant-cloud:
    type: "qdrant"
    url: "https://your-cluster.qdrant.tech"
    api_key: "${QDRANT_CLOUD_API_KEY}"
    timeout: "30s"
```

### Custom Database Plugin

```yaml
databases:
  custom_db:
    type: "plugin:my_custom_db"
    config:
      endpoint: "http://localhost:8082"
      api_key: "${CUSTOM_DB_API_KEY}"
```

---

## 🔤 Embedder Providers

Configure embedding providers for text vectorization.

### Ollama

```yaml
embedders:
  ollama:
    type: "ollama"
    model: "nomic-embed-text"
    url: "http://localhost:11434"
    timeout: "30s"
    
  ollama-large:
    type: "ollama"
    model: "mxbai-embed-large"
    url: "http://localhost:11434"
    timeout: "30s"
```

### Custom Embedder Plugin

```yaml
embedders:
  custom_embedder:
    type: "plugin:my_custom_embedder"
    config:
      endpoint: "http://localhost:8083"
      api_key: "${CUSTOM_EMBEDDER_API_KEY}"
```

---

## 🔐 Security Configuration

Configure authentication and security settings.

### JWT Authentication

```yaml
auth:
  jwt:
    secret: "${JWT_SECRET}"
    expires_in: "24h"
    issuer: "hector"
    audience: "hector-clients"
    
  # Agent access control
  agents:
    public_agent:
      visibility: "public"
    private_agent:
      visibility: "private"
      required_scopes: ["admin"]
```

### Security Configuration Options

```yaml
auth:
  # JWT configuration
  jwt:
    secret: string                    # JWT signing secret
    expires_in: string                # Token expiration time
    issuer: string                    # Token issuer
    audience: string                  # Token audience
    
  # Agent security
  agents:
    <agent-id>:
      visibility: string              # "public", "internal", "private"
      required_scopes: []string       # Required OAuth scopes
      
  # Transport security
  tls:
    enabled: bool                     # Enable TLS/SSL
    cert_file: string                 # Certificate file path
    key_file: string                  # Private key file path
```

---

## 📋 Task Configuration

Configure task management and processing.

### Task Configuration

```yaml
tasks:
  # Task processing
  max_concurrent: 10                  # Maximum concurrent tasks
  timeout: "300s"                     # Default task timeout
  
  # Task storage
  storage:
    type: "memory"                    # "memory", "redis", "database"
    config:
      # Storage-specific configuration
      
  # Task retry
  retry:
    max_attempts: 3                   # Maximum retry attempts
    backoff_multiplier: 2.0           # Backoff multiplier
    max_backoff: "60s"                # Maximum backoff time
```

---

## 🌐 Global Configuration

Configure global server settings.

### Server Configuration

```yaml
global:
  # Server settings
  server:
    host: "0.0.0.0"                   # Server host
    port: 8080                        # Server port
    timeout: "30s"                    # Request timeout
    
  # Logging
  logging:
    level: "info"                     # Log level
    format: "json"                    # Log format
    
  # Monitoring
  monitoring:
    metrics:
      enabled: true                   # Enable metrics
      port: 9090                      # Metrics port
    health:
      enabled: true                   # Enable health checks
      endpoint: "/health"             # Health check endpoint
```

---

## 🔌 Plugins

Configure custom plugins for extending Hector functionality.

### Plugin Configuration

```yaml
plugins:
  # LLM plugins
  llm_providers:
    my_custom_llm:
      type: "grpc"
      path: "./plugins/my-llm-plugin"
      enabled: true
      config:
        # Plugin-specific configuration
  
  # Database plugins
  database_providers:
    my_custom_db:
      type: "grpc"
      path: "./plugins/my-db-plugin"
      enabled: true
      config:
        # Plugin-specific configuration
  
  # Tool plugins
  tool_providers:
    my_custom_tool:
      type: "grpc"
      path: "./plugins/my-tool-plugin"
      enabled: true
      config:
        # Plugin-specific configuration
```

### Plugin Options

```yaml
plugins:
  <plugin-type>:
    <plugin-id>:
      # Plugin type
      type: string                    # "grpc", "http", "custom"
      
      # Plugin path/endpoint
      path: string                    # Plugin binary path
      endpoint: string                # Plugin HTTP endpoint
      
      # Plugin configuration
      enabled: bool                   # Enable/disable plugin
      config: object                  # Plugin-specific configuration
      
      # Plugin lifecycle
      auto_start: bool                # Auto-start plugin
      restart_on_failure: bool       # Restart on failure
```

---

## Configuration Examples

### Minimal Configuration

```yaml
# Minimal working configuration
agents:
  assistant:
    name: "My Assistant"
    llm: "gpt-4o"

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
```

### Production Configuration

```yaml
# Production-ready configuration
agents:
  coding_assistant:
    name: "Coding Assistant"
    description: "Helps with coding tasks"
    visibility: "public"
    llm: "gpt-4o"
    
    prompt:
      system_role: |
        You are an expert software engineer.
    
    memory:
      strategy: "summary_buffer"
      max_tokens: 4000
      long_term:
        enabled: true
        strategy: "vector_memory"
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 15
      enable_streaming: true
    
    tools: ["execute_command", "write_file", "search_replace"]
    document_stores: ["codebase_docs"]
    database: "qdrant"
    embedder: "ollama"

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
    max_tokens: 4000

tools:
  execute_command:
    type: "command"
    enabled: true
    allowed_commands: ["cat", "ls", "grep", "git", "npm", "go"]
    max_execution_time: "30s"
  
  write_file:
    type: "write_file"
    enabled: true
    allowed_paths: ["./src/", "./docs/"]
    max_file_size: "10MB"

databases:
  qdrant:
    type: "qdrant"
    url: "http://localhost:6333"
    timeout: "30s"

embedders:
  ollama:
    type: "ollama"
    model: "nomic-embed-text"
    url: "http://localhost:11434"
    timeout: "30s"

document_stores:
  codebase_docs:
    type: "qdrant"
    collection: "codebase"
    database: "qdrant"
    embedder: "ollama"

auth:
  jwt:
    secret: "${JWT_SECRET}"
    expires_in: "24h"
    issuer: "hector"

global:
  server:
    host: "0.0.0.0"
    port: 8080
    timeout: "30s"
  
  logging:
    level: "info"
    format: "json"
  
  monitoring:
    metrics:
      enabled: true
      port: 9090
    health:
      enabled: true
      endpoint: "/health"
```

---

## Environment Variables

Hector supports environment variable substitution in configuration:

```yaml
# Use environment variables
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"        # From environment
    temperature: 0.7
    max_tokens: 4000

auth:
  jwt:
    secret: "${JWT_SECRET}"             # From environment
    expires_in: "24h"
    issuer: "hector"
```

### Common Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `OPENAI_API_KEY` | OpenAI API key | `sk-...` |
| `ANTHROPIC_API_KEY` | Anthropic API key | `sk-ant-...` |
| `GEMINI_API_KEY` | Google Gemini API key | `AI...` |
| `JWT_SECRET` | JWT signing secret | `your-secret-key` |
| `QDRANT_API_KEY` | Qdrant API key | `your-qdrant-key` |

---

## Configuration Validation

Hector validates configuration on startup:

### Validation Checks

- :material-check: **Required Fields** - All required fields are present
- :material-check: **Type Validation** - Field types match expected types
- :material-check: **Reference Validation** - All references are valid
- :material-check: **Plugin Validation** - Plugin configurations are valid
- :material-check: **Security Validation** - Security settings are valid

### Common Validation Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `missing required field: llm` | Agent missing LLM reference | Add `llm` field to agent |
| `invalid LLM reference: unknown-llm` | LLM provider not found | Define LLM provider or fix reference |
| `invalid tool reference: unknown-tool` | Tool not found | Define tool or fix reference |
| `invalid database reference: unknown-db` | Database provider not found | Define database provider or fix reference |

---

## Related Documentation

- [Building Agents](agents.md) - Learn how to build AI agents
- [Architecture](architecture.md) - Understand Hector's architecture
- [Tools & Extensions](tools.md) - Built-in tools and custom extensions
- [Memory Management](memory.md) - Memory system configuration
- [Plugin Development](plugins.md) - Build custom plugins
