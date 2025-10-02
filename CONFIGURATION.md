# Hector Configuration Reference

Complete reference for all configuration options in Hector.

## Table of Contents

- [Overview](#overview)
- [Configuration Structure](#configuration-structure)
- [Top-Level Settings](#top-level-settings)
- [Global Settings](#global-settings)
- [LLM Providers](#llm-providers)
- [Database Providers](#database-providers)
- [Embedder Providers](#embedder-providers)
- [Agents](#agents)
- [Reasoning Engines](#reasoning-engines)
- [Prompt Configuration](#prompt-configuration)
- [Search Configuration](#search-configuration)
- [Tools](#tools)
- [Document Stores](#document-stores)
- [Workflows](#workflows)
- [Environment Variables](#environment-variables)
- [Examples](#examples)

---

## Overview

Hector uses a single YAML configuration file (similar to `docker-compose.yml`) that defines:
- **Service Providers**: LLMs, databases, embedders
- **Agents**: AI agents with specific capabilities
- **Workflows**: Multi-agent orchestration
- **Tools**: External integrations and commands
- **Document Stores**: Knowledge bases and vector storage

### Zero-Config Support

Hector provides intelligent defaults for all settings. You can start with minimal configuration:

```yaml
agents:
  my-agent:
    llm: "openai"

llms:
  openai:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

---

## Configuration Structure

```yaml
# Metadata
version: "1.0"
name: "my-config"
description: "Configuration description"

# Global Settings
global:
  logging:
    level: "info"
  performance:
    max_concurrency: 4

# Service Providers
llms:
  my-llm:
    type: "openai"
    model: "gpt-4o"
    # ... provider config

databases:
  my-db:
    type: "qdrant"
    # ... provider config

embedders:
  my-embedder:
    type: "ollama"
    # ... provider config

# Agents
agents:
  my-agent:
    name: "My Agent"
    llm: "my-llm"
    # ... agent config

# Workflows (optional)
workflows:
  my-workflow:
    name: "My Workflow"
    mode: "dag"
    # ... workflow config

# Tools (optional)
tools:
  default_repo: "local"
  repositories:
    - name: "local"
      type: "local"
      # ... tool config

# Document Stores (optional)
document_stores:
  my-store:
    source: "directory"
    path: "./docs"
    # ... store config
```

---

## Top-Level Settings

### Metadata

```yaml
version: "1.0"                    # Config version (optional)
name: "hector-config"             # Config name (optional)
description: "My configuration"   # Description (optional)
metadata:                         # Custom metadata (optional)
  author: "Your Name"
  environment: "production"
```

**Defaults:**
- `version`: "1.0"
- `name`: "hector"
- `description`: "" (empty)

---

## Global Settings

Global settings affect all agents and workflows.

```yaml
global:
  logging:
    level: "info"        # Log level: debug, info, warn, error
    format: "json"       # Format: text, json
    output: "stdout"     # Output: stdout, stderr, file
  
  performance:
    max_concurrency: 4   # Max concurrent operations
    timeout: "15m"       # Global timeout (duration format)
```

**Defaults:**
- `logging.level`: "info"
- `logging.format`: "text"
- `logging.output`: "stdout"
- `performance.max_concurrency`: 4
- `performance.timeout`: "15m"

**Valid Values:**
- `level`: debug, info, warn, error
- `format`: text, json
- `output`: stdout, stderr, file
- `timeout`: Duration format (e.g., "1h", "30m", "90s")

---

## LLM Providers

LLM providers are the language models used by agents.

### OpenAI

```yaml
llms:
  openai:
    type: "openai"
    model: "gpt-4o"                          # Model name
    api_key: "${OPENAI_API_KEY}"             # API key (env var)
    host: "https://api.openai.com/v1"        # API endpoint
    temperature: 0.7                         # 0.0-2.0 (creativity)
    max_tokens: 4000                         # Max response tokens
    timeout: 120                             # Request timeout (seconds)
```

**Supported Models:**
- `gpt-4o` - Latest GPT-4 Omni (best quality)
- `gpt-4o-mini` - Smaller, faster, cost-effective
- `gpt-4` - GPT-4 Turbo
- `gpt-3.5-turbo` - Fast, cost-effective

### Anthropic (Claude)

```yaml
llms:
  claude:
    type: "anthropic"
    model: "claude-sonnet-4.5-20250514"      # Model alias
    api_key: "${ANTHROPIC_API_KEY}"          # API key (env var)
    host: "https://api.anthropic.com"        # API endpoint
    temperature: 0.7                         # 0.0-1.0 (creativity)
    max_tokens: 4000                         # Max response tokens
    timeout: 120                             # Request timeout (seconds)
```

**Supported Models:**
- `claude-sonnet-4.5-20250514` - Claude Sonnet 4.5 (latest)
- `claude-opus-4.1-20250514` - Claude Opus 4.1 (most capable)
- `claude-3-5-sonnet-20241022` - Claude 3.5 Sonnet
- `claude-3-haiku-20240307` - Claude 3 Haiku (fast)

### Ollama (Local)

```yaml
llms:
  ollama:
    type: "ollama"
    model: "llama3.2"                        # Model name
    host: "http://localhost:11434"           # Ollama server
    temperature: 0.7                         # 0.0-2.0 (creativity)
    max_tokens: 2000                         # Max response tokens
    timeout: 60                              # Request timeout (seconds)
```

**Popular Models:**
- `llama3.2` - Meta's Llama 3.2
- `mistral` - Mistral 7B
- `qwen2.5` - Alibaba's Qwen 2.5
- `gemma2` - Google's Gemma 2

**Defaults:**
- `type`: "ollama" (zero-config default)
- `model`: "llama3.2"
- `host`: "http://localhost:11434"
- `temperature`: 0.7
- `max_tokens`: 2000
- `timeout`: 60

---

## Database Providers

Database providers are used for vector storage and semantic search.

### Qdrant

```yaml
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"                        # Qdrant host
    port: 6333                               # Qdrant port
    api_key: "${QDRANT_API_KEY}"             # API key (optional)
    use_tls: false                           # Use TLS connection
    timeout: 30                              # Connection timeout (seconds)
```

**Defaults:**
- `type`: "qdrant"
- `host`: "localhost"
- `port`: 6333
- `timeout`: 30
- `use_tls`: false

---

## Embedder Providers

Embedder providers generate embeddings for semantic search.

### Ollama Embeddings

```yaml
embedders:
  ollama:
    type: "ollama"
    model: "nomic-embed-text"                # Embedding model
    host: "http://localhost:11434"           # Ollama server
    dimension: 768                           # Embedding dimension
    timeout: 30                              # Request timeout (seconds)
    max_retries: 3                           # Max retry attempts
```

**Supported Models:**
- `nomic-embed-text` - 768 dim, high quality
- `mxbai-embed-large` - 1024 dim, best quality
- `all-minilm` - 384 dim, fast

**Defaults:**
- `type`: "ollama"
- `model`: "nomic-embed-text"
- `host`: "http://localhost:11434"
- `dimension`: 768
- `timeout`: 30
- `max_retries`: 3

---

## Agents

Agents are AI assistants with specific capabilities and configurations.

```yaml
agents:
  my-agent:
    name: "My Agent"                         # Agent name
    description: "Agent description"         # Agent description
    llm: "openai"                            # LLM provider reference
    database: "qdrant"                       # Database reference (optional)
    embedder: "ollama"                       # Embedder reference (optional)
    document_stores: ["docs"]                # Document stores (optional)
    
    prompt:
      # ... prompt configuration (see below)
    
    reasoning:
      # ... reasoning configuration (see below)
    
    search:
      # ... search configuration (see below)
    
    tools:
      # ... tool configuration (see below)
```

**Required:**
- `name` - Agent name
- `llm` - LLM provider reference

**Optional:**
- `database` - Required if using document stores or search
- `embedder` - Required if using document stores or search
- `document_stores` - List of document store references
- `prompt`, `reasoning`, `search`, `tools` - All have defaults

---

## Reasoning Engines

Reasoning engines control how agents think and make decisions.

### Chain-of-Thought (Fast)

```yaml
reasoning:
  engine: "chain-of-thought"
  max_iterations: 5                          # Max reasoning iterations
  enable_streaming: true                     # Enable streaming output
  show_debug_info: false                     # Show debug information
```

**Best for:** Simple queries, speed-critical tasks
**Token usage:** Lower (~2-3 LLM calls per query)

### Structured Reasoning (Thorough)

```yaml
reasoning:
  engine: "structured-reasoning"
  max_iterations: 10                         # Max reasoning iterations
  enable_streaming: true                     # Enable streaming output
  show_thinking: true                        # Show internal reasoning (grayed out)
  show_debug_info: false                     # Show full debug information
```

**Best for:** Complex analysis, research tasks
**Token usage:** Higher (~9-14 LLM calls for complex queries)

**Visibility Modes:**
- `show_thinking: true` - Grayed-out thinking blocks (like Claude in Cursor)
- `show_debug_info: true` - Full verbose output with all details
- Both false - Minimal clean output with progress indicators

**Defaults:**
- `engine`: "chain-of-thought"
- `max_iterations`: 5 (chain-of-thought) or 10 (structured)
- `enable_streaming`: true
- `show_thinking`: false
- `show_debug_info`: false

---

## Prompt Configuration

Control how prompts are constructed and what context is included.

```yaml
prompt:
  system_prompt: |
    You are a helpful AI assistant...
  include_context: true                      # Include search context
  include_history: true                      # Include conversation history
  include_tools: true                        # Include tool descriptions
  max_context_length: 4000                   # Max context tokens
```

**Defaults:**
- `system_prompt`: Generic helpful assistant
- `include_context`: true
- `include_history`: true
- `include_tools`: true
- `max_context_length`: 4000

---

## Search Configuration

Configure semantic search for retrieving relevant context.

```yaml
search:
  models:
    - name: "documents"                      # Search model name
      collection: "docs"                     # Vector collection name
      default_top_k: 10                      # Default results
      max_top_k: 50                          # Max results
  
  top_k: 10                                  # Global default results
  threshold: 0.7                             # Similarity threshold (0-1)
  max_context_length: 4000                   # Max context length
```

**Defaults:**
- `top_k`: 5
- `threshold`: 0.7
- `max_context_length`: 4000

---

## Tools

Tools enable agents to interact with external systems.

### Tool Configuration

```yaml
tools:
  default_repo: "local"                      # Default repository
  repositories:
    - name: "mcp"                            # MCP repository
      type: "mcp"
      description: "MCP tool repository"
      url: "${MCP_SERVER_URL}"               # MCP server URL
    
    - name: "local"                          # Local tools
      type: "local"
      description: "Built-in local tools"
      tools:
        - name: "execute_command"
          type: "command"
          enabled: true
          config:
            command_config:
              allowed_commands: ["ls", "cat", "grep"]
              working_directory: "./"
              max_execution_time: "30s"
              enable_sandboxing: true
```

### Command Tool Configuration

```yaml
- name: "execute_command"
  type: "command"
  enabled: true
  config:
    command_config:
      allowed_commands:                      # Whitelist of commands
        - "ls"
        - "cat"
        - "grep"
        - "find"
        - "git"
      working_directory: "./"                # Working directory
      max_execution_time: "30s"              # Max execution time
      enable_sandboxing: true                # Enable sandboxing
```

**Security:**
- Only whitelisted commands can be executed
- Working directory is restricted
- Execution time is limited
- Sandboxing prevents system access

---

## Document Stores

Document stores provide knowledge bases for agents.

```yaml
document_stores:
  codebase:
    name: "Codebase"                         # Store name
    source: "directory"                      # Source type
    path: "./"                               # Directory path
    include_patterns:                        # File patterns to include
      - "*.go"
      - "*.md"
      - "*.yaml"
    exclude_patterns:                        # File patterns to exclude
      - "**/.git/**"
      - "**/node_modules/**"
      - "**/vendor/**"
    watch_changes: true                      # Watch for file changes
    max_file_size: 10485760                  # Max file size (10MB)
```

**Defaults:**
- `source`: "directory"
- `watch_changes`: true
- `max_file_size`: 10485760 (10MB)

---

## Workflows

Workflows enable multi-agent collaboration.

### DAG Workflow (Structured)

```yaml
workflows:
  my-workflow:
    name: "My Workflow"
    description: "Structured workflow"
    mode: "dag"                              # DAG execution mode
    
    agents:                                  # Agent references
      - "research-agent"
      - "analysis-agent"
      - "synthesis-agent"
    
    execution:
      dag:
        steps:
          - name: "research"
            agent: "research-agent"
            input: "${user_input}"
            output: "research_results"
          
          - name: "analysis"
            agent: "analysis-agent"
            input: "${research_results}"
            output: "analysis_results"
            depends_on: ["research"]
          
          - name: "synthesis"
            agent: "synthesis-agent"
            input: "${analysis_results}"
            output: "final_report"
            depends_on: ["analysis"]
    
    settings:
      max_concurrency: 2                     # Max parallel agents
      timeout: "20m"                         # Workflow timeout
      retry_policy:
        max_retries: 3
        backoff: "5s"
```

### Autonomous Workflow (AI-Driven)

```yaml
workflows:
  autonomous-research:
    name: "Autonomous Research"
    description: "AI-driven autonomous workflow"
    mode: "autonomous"                       # Autonomous mode
    
    agents:
      - "research-agent"
      - "analysis-agent"
      - "creative-agent"
    
    execution:
      autonomous:
        goal: "Conduct comprehensive research"
        strategy: "dynamic"                  # Dynamic task planning
        max_iterations: 10
        coordinator_llm: "openai-gpt4"       # Coordinator LLM
        termination_conditions:
          max_duration: "45m"
          quality_threshold: 0.9
          max_iterations: 10
    
    settings:
      max_concurrency: 4
      timeout: "45m"
      retry_policy:
        max_retries: 5
        backoff: "10s"
```

---

## Environment Variables

Hector supports environment variable substitution using `${VAR_NAME}` syntax.

### Common Variables

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# Qdrant
export QDRANT_API_KEY="..."

# MCP Server
export MCP_SERVER_URL="https://..."
```

### Usage in Config

```yaml
llms:
  openai:
    api_key: "${OPENAI_API_KEY}"             # Environment variable
    host: "${OPENAI_HOST:-https://api.openai.com/v1}"  # With default
```

**Syntax:**
- `${VAR}` - Required variable (fails if not set)
- `${VAR:-default}` - Optional with default value
- `${VAR:?error}` - Required with custom error message

---

## Examples

### Minimal Single Agent

```yaml
agents:
  assistant:
    llm: "openai"

llms:
  openai:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

### Chain-of-Thought Agent

```yaml
agents:
  cot-agent:
    name: "Fast Agent"
    llm: "openai"
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 5
      enable_streaming: true

llms:
  openai:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
```

### Structured Reasoning Agent

```yaml
agents:
  research-agent:
    name: "Research Agent"
    llm: "openai"
    reasoning:
      engine: "structured-reasoning"
      max_iterations: 10
      show_thinking: true              # See internal reasoning!

llms:
  openai:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
```

### Agent with Tools

```yaml
agents:
  tool-agent:
    name: "Tool Agent"
    llm: "openai"

tools:
  default_repo: "local"
  repositories:
    - name: "local"
      type: "local"
      tools:
        - name: "execute_command"
          type: "command"
          enabled: true
          config:
            command_config:
              allowed_commands: ["ls", "cat", "grep"]
              working_directory: "./"

llms:
  openai:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

### Agent with Search

```yaml
agents:
  search-agent:
    name: "Search Agent"
    llm: "openai"
    database: "qdrant"
    embedder: "ollama"
    document_stores: ["docs"]
    
    search:
      top_k: 10
      threshold: 0.7

llms:
  openai:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"

databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6333

embedders:
  ollama:
    type: "ollama"
    model: "nomic-embed-text"

document_stores:
  docs:
    source: "directory"
    path: "./docs"
    include_patterns: ["*.md", "*.txt"]
```

### Multi-Agent Workflow

```yaml
workflows:
  research-pipeline:
    name: "Research Pipeline"
    mode: "dag"
    agents:
      - "researcher"
      - "analyst"
    
    execution:
      dag:
        steps:
          - name: "research"
            agent: "researcher"
            input: "${user_input}"
            output: "findings"
          
          - name: "analyze"
            agent: "analyst"
            input: "${findings}"
            output: "report"
            depends_on: ["research"]

agents:
  researcher:
    name: "Researcher"
    llm: "openai"
  
  analyst:
    name: "Analyst"
    llm: "claude"

llms:
  openai:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
  
  claude:
    type: "anthropic"
    model: "claude-sonnet-4.5-20250514"
    api_key: "${ANTHROPIC_API_KEY}"
```

---

## Configuration Best Practices

### 1. Use Environment Variables for Secrets

**❌ Bad:**
```yaml
api_key: "sk-proj-abc123..."  # Hardcoded secret
```

**✅ Good:**
```yaml
api_key: "${OPENAI_API_KEY}"  # Environment variable
```

### 2. Start Simple, Add Complexity

**Phase 1 - Minimal:**
```yaml
agents:
  my-agent:
    llm: "openai"
```

**Phase 2 - Add Reasoning:**
```yaml
agents:
  my-agent:
    llm: "openai"
    reasoning:
      engine: "chain-of-thought"
```

**Phase 3 - Add Tools:**
```yaml
agents:
  my-agent:
    llm: "openai"
    reasoning:
      engine: "chain-of-thought"

tools:
  default_repo: "local"
  # ... tools
```

### 3. Use Descriptive Names

**❌ Bad:**
```yaml
agents:
  a1:
    name: "Agent"
```

**✅ Good:**
```yaml
agents:
  research-agent:
    name: "Research Specialist"
    description: "Gathers and analyzes information"
```

### 4. Organize by Purpose

```yaml
# Group providers
llms:
  # Fast, cost-effective
  mini: { type: "openai", model: "gpt-4o-mini" }
  
  # Best quality
  full: { type: "openai", model: "gpt-4o" }
  
  # Local
  local: { type: "ollama", model: "llama3.2" }

# Group agents by role
agents:
  researcher: # Research tasks
  analyst:    # Analysis tasks
  writer:     # Content creation
```

### 5. Set Appropriate Timeouts

```yaml
llms:
  openai:
    timeout: 120              # 2 minutes for complex queries

global:
  performance:
    timeout: "15m"            # 15 minutes for workflows
```

### 6. Use Comments

```yaml
reasoning:
  engine: "structured-reasoning"
  max_iterations: 10          # Increased for thorough analysis
  show_thinking: true         # Helpful for debugging
```

---

## Troubleshooting

### Configuration Not Loading

**Error:** `failed to load config`

**Solutions:**
1. Check YAML syntax (use `yamllint`)
2. Verify file path is correct
3. Ensure environment variables are set

### Validation Errors

**Error:** `config validation failed: llm provider 'openai' validation failed: api_key is required`

**Solution:**
```bash
export OPENAI_API_KEY="sk-..."
```

### Agent Not Working

**Checklist:**
- [ ] LLM provider configured and reachable
- [ ] API keys set in environment
- [ ] Required services (Qdrant, Ollama) running if used
- [ ] Tool commands whitelisted if using tools
- [ ] Document stores paths exist if using search

---

## See Also

- [README.md](README.md) - Main documentation
- [REASONING_ENGINES.md](REASONING_ENGINES.md) - Reasoning engines guide
- [examples/](examples/) - Example configurations
- [local-configs/](local-configs/) - Production configs

---

**Need Help?**

- 📖 Read the examples in `examples/` directory
- 🔧 Start with `examples/minimal.yaml` for basic setup
- 🚀 Use `examples/advanced.yaml` for enterprise features
- 💬 Check inline comments in example configs

