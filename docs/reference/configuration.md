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

# Providers
llms:             # LLM providers
databases:        # Vector databases
embedders:        # Embedding models
plugins:          # gRPC plugins

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
    rest_gateway_port: 8081      # REST API port (default: 8081)
    jsonrpc_port: 8082           # JSON-RPC port (default: 8082)
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
      # Slot-based (recommended)
      prompt_slots:
        system_role: "Core role"
        reasoning_instructions: "How to think"
        tool_usage: "Tool guidelines"
        output_format: "Response format"
        communication_style: "Tone and style"
        additional: "Extra context"
      
      # OR full override
      system_prompt: "Complete system prompt..."
    
    # Reasoning Configuration
    reasoning:
      engine: "chain-of-thought"  # chain-of-thought|supervisor
      max_iterations: 100
      enable_streaming: true
      show_tool_execution: true
      show_thinking: false
      show_debug_info: false
      enable_structured_reflection: true
      enable_completion_verification: false
    
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
        enabled: true
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
    enabled: true
    # Permissive defaults: all commands allowed (sandboxed)
    # Optional restrictions:
    # allowed_commands: ["npm", "git", "python"]
    # denied_commands: ["rm", "sudo"]
    # max_execution_time: "30s"
  
  write_file:
    type: write_file
    enabled: true
    # Permissive defaults: all file types and paths allowed
    # Optional restrictions:
    # allowed_paths: ["./src/", "./docs/"]
    # denied_paths: ["./secrets/"]
  
  search_replace:
    type: search_replace
    enabled: true
    # Permissive defaults: no restrictions
    # Optional: backup: true
  
  search:
    enabled: true
    default_limit: 10
    max_limit: 50
  
  todo_write:
    enabled: true
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
      enabled: true
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
    port: 6333                    # Default: 6333
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

```yaml
document_stores:
  - name: "codebase"
    paths:
      - "./src/"
      - "./lib/"
    include_patterns:
      - "*.go"
      - "*.py"
      - "*.js"
    exclude_patterns:
      - "*_test.go"
      - "node_modules/*"
      - ".git/*"
    
    chunk_size: 512               # Characters per chunk
    chunk_overlap: 50             # Overlap between chunks
    
    collection: "codebase"        # Qdrant collection name
    batch_size: 100               # Docs per batch
    
    parser: "native"              # native|custom|plugin
    
    # Search configuration
    search_config:
      limit: 5
      score_threshold: 0.7
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
      system_role: "You are ${AGENT_NAME} for ${COMPANY_NAME}"
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
    port: 6333

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
      prompt_slots:
        system_role: "You are an expert programmer."
        reasoning_instructions: "Think step-by-step."
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100
      enable_streaming: true
    
    tools:
      - "search"
      - "write_file"
      - "execute_command"
    
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000
      longterm:
        enabled: true
        storage_scope: "session"
    
    document_stores:
      - name: "codebase"
        paths: ["./src/"]
        include_patterns: ["*.go", "*.py"]
        chunk_size: 512

# Tools
tools:
  execute_command:
    type: command
    enabled: true
    # Permissive defaults (sandboxed)
  
  write_file:
    type: write_file
    enabled: true
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
hector serve --config config.yaml --port 9090
# Result: Uses port 9090
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

