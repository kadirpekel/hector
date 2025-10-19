# Hector Feature & Capability Tree
## Comprehensive Feature-to-Configuration Mapping

**Purpose**: This document serves as the authoritative source-of-truth for all features and capabilities in Hector, with precise mappings to their configuration options. Use this as the foundation for documentation restructuring.

**Last Updated**: 2025-10-19  
**Version**: 1.1 (Verified against codebase)

**Verification Status**: ✅ 100% Code-Verified  
This document has been thoroughly verified against the actual codebase including:
- Configuration types and defaults from `pkg/config/types.go` and test files
- CLI argument parsing from `cmd/hector/main.go`
- Reasoning strategies from `pkg/reasoning/`
- Memory management from `pkg/memory/`
- All example configurations from `configs/`

---

## DOCUMENT CHANGELOG

### Version 1.1 (2025-10-19)
**Major Corrections & Additions**:
- ✅ Fixed Qdrant default port: 6333 (not 6334)
- ✅ Corrected OpenAI default model: gpt-4o (from config defaults)
- ✅ Added zero-config default models for all providers
- ✅ Enhanced Agent-Level Security with full SecurityScheme details
- ✅ Added comprehensive environment variable documentation
- ✅ Added default value annotations throughout
- ✅ Verified all configuration paths against actual code
- ✅ Added external agent authentication details
- ✅ Clarified configuration precedence and defaults

---

## 1. CORE PLATFORM CAPABILITIES

### 1.1 Agent Architecture

#### 1.1.1 Native Agents
**Description**: Local agents executing within Hector runtime with full control

**Configuration**:
```yaml
agents:
  my_agent:
    type: "native"                    # Default type
    name: "Agent Name"
    description: "Agent description"
    llm: "llm-provider-ref"          # Required: LLM provider reference
```

**Config Location**: `agents.<agent_name>`
**Related Files**: 
- `pkg/agent/agent.go` (core implementation)
- `pkg/agent/agent_a2a_methods.go` (A2A protocol methods)
- `pkg/config/types.go` (AgentConfig)

---

#### 1.1.2 External A2A Agents
**Description**: Remote agents accessed via A2A protocol (HTTP/gRPC)

**Configuration**:
```yaml
agents:
  external_agent:
    type: "a2a"                      # External agent type
    name: "External Agent"
    url: "https://remote.com/agents/specialist"  # Required: A2A endpoint
    credentials:                     # Optional authentication
      type: "bearer"                # bearer|api_key|basic
      token: "${EXTERNAL_TOKEN}"
```

**Config Location**: `agents.<agent_name>`
**Related Files**:
- `pkg/agent/a2a_client.go` (HTTP client)
- `docs/external-agents.md` (documentation)

---

#### 1.1.3 Agent Visibility
**Description**: Control agent discoverability and access

**Configuration**:
```yaml
agents:
  my_agent:
    visibility: "public"             # public|internal|private
    # public: Discoverable via /agents and callable
    # internal: Not discoverable, callable if you know agent ID
    # private: Only callable by local orchestrators
```

**Config Location**: `agents.<agent_name>.visibility`
**Related Files**: 
- `pkg/transport/discovery.go`
- `docs/authentication.md`

---

### 1.2 LLM Providers

#### 1.2.1 OpenAI
**Description**: OpenAI GPT models (gpt-4o, gpt-4o-mini, etc.)

**Configuration**:
```yaml
llms:
  my-openai:
    type: "openai"
    model: "gpt-4o"                  # Default if not specified
    api_key: "${OPENAI_API_KEY}"
    host: "https://api.openai.com/v1"  # Default host
    temperature: 0.7                 # Default: 0.7
    max_tokens: 8000                 # Default: 8000
    timeout: 60                      # Default: 60 seconds
    max_retries: 5                   # Default: 5 (rate limit retries)
    retry_delay: 2                   # Default: 2 seconds (exponential backoff)
```

**Config Location**: `llms.<provider_name>`
**Environment Variables**: `OPENAI_API_KEY`
**Related Files**:
- `pkg/llms/openai.go`
- `configs/coding.yaml` (example)

---

#### 1.2.2 Anthropic
**Description**: Anthropic Claude models (claude-sonnet-4, claude-opus-4, etc.)

**Configuration**:
```yaml
llms:
  my-anthropic:
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

**Config Location**: `llms.<provider_name>`
**Environment Variables**: `ANTHROPIC_API_KEY`
**Related Files**:
- `pkg/llms/anthropic.go`
- `configs/coding.yaml` (example)

---

#### 1.2.3 Google Gemini
**Description**: Google Gemini models (gemini-2.0-flash-exp, etc.)

**Configuration**:
```yaml
llms:
  my-gemini:
    type: "gemini"
    model: "gemini-2.0-flash-exp"
    api_key: "${GEMINI_API_KEY}"
    host: "https://generativelanguage.googleapis.com"
    temperature: 0.7
    max_tokens: 4096
    timeout: 60
```

**Config Location**: `llms.<provider_name>`
**Environment Variables**: `GEMINI_API_KEY`
**Related Files**: `pkg/llms/gemini.go`

---

#### 1.2.4 Structured Output
**Description**: Provider-agnostic structured output (JSON/XML/Enum)

**Configuration**:
```yaml
llms:
  my-llm:
    type: "openai"
    structured_output:
      format: "json"                 # json|xml|enum
      schema:                        # For JSON format
        type: "object"
        properties:
          analysis:
            type: "string"
      enum: ["yes", "no"]            # For enum format
      prefill: "<analysis>"          # Anthropic-specific
      property_ordering: ["field1"]  # Gemini-specific
```

**Config Location**: `llms.<provider_name>.structured_output`
**Related Files**:
- `pkg/llms/types.go` (StructuredOutputConfig)
- `docs/structured-output.md`

---

### 1.3 Reasoning Strategies

#### 1.3.1 Chain-of-Thought
**Description**: Simple step-by-step reasoning with natural termination

**Configuration**:
```yaml
agents:
  my_agent:
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100            # Safety valve (trust LLM to terminate)
      enable_streaming: true
      show_tool_execution: true
      show_thinking: true            # Claude-style grayed-out reflection
      show_debug_info: false
      enable_structured_reflection: true   # LLM-based reflection
      enable_completion_verification: false
```

**Config Location**: `agents.<agent_name>.reasoning`
**Related Files**:
- `pkg/reasoning/chain_of_thought_strategy.go`
- `pkg/reasoning/reflection.go`
- `pkg/reasoning/completion.go`

---

#### 1.3.2 Supervisor Strategy
**Description**: Multi-agent orchestration with LLM-driven delegation

**Configuration**:
```yaml
agents:
  orchestrator:
    reasoning:
      engine: "supervisor"
      max_iterations: 20             # More iterations for orchestration
      enable_goal_extraction: true   # LLM extracts goals from tasks
    sub_agents: ["researcher", "analyst"]  # Optional: restrict agents
    tools:
      - agent_call                   # Required: delegation tool
```

**Config Location**: `agents.<agent_name>.reasoning`
**Related Files**:
- `pkg/reasoning/supervisor_strategy.go`
- `pkg/reasoning/goals.go`
- `configs/orchestrator-example.yaml`

---

### 1.4 Memory Management

#### 1.4.1 Working Memory Strategies

##### Summary Buffer (Default)
**Description**: Token-based memory with threshold-triggered summarization

**Configuration**:
```yaml
agents:
  my_agent:
    memory:
      strategy: "summary_buffer"     # Default
      budget: 2000                   # Token budget
      threshold: 0.8                 # Trigger at 80% capacity
      target: 0.6                    # Compress to 60% capacity
```

**Config Location**: `agents.<agent_name>.memory`
**Related Files**:
- `pkg/memory/summary_buffer.go`
- `configs/memory-strategies-example.yaml`

---

##### Buffer Window
**Description**: Simple LIFO memory keeping last N messages

**Configuration**:
```yaml
agents:
  my_agent:
    memory:
      strategy: "buffer_window"
      window_size: 20                # Keep last 20 messages
```

**Config Location**: `agents.<agent_name>.memory`
**Related Files**:
- `pkg/memory/buffer_window.go`
- `configs/memory-strategies-example.yaml`

---

#### 1.4.2 Long-Term Memory (Semantic Recall)
**Description**: Vector-based persistent memory with semantic search

**Configuration**:
```yaml
agents:
  my_agent:
    database: "qdrant"               # Required for long-term memory
    embedder: "embedder"             # Required for long-term memory
    memory:
      long_term:
        storage_scope: "all"         # all|conversational|summaries_only
        batch_size: 1                # Messages per batch (1=immediate)
        auto_recall: true            # Auto-inject memories
        recall_limit: 5              # Max memories recalled
        collection: "hector_session_memory"
```

**Config Location**: `agents.<agent_name>.memory.long_term`
**Related Files**:
- `pkg/memory/vector_memory.go`
- `pkg/memory/longterm_strategy.go`
- `configs/long-term-memory-example.yaml`

---

### 1.5 Prompt System

#### 1.5.1 Slot-Based Prompts (Recommended)
**Description**: Composable prompt slots for fine-grained customization

**Configuration**:
```yaml
agents:
  my_agent:
    prompt:
      prompt_slots:
        system_role: "You are a helpful assistant..."
        reasoning_instructions: "Think step by step..."
        tool_usage: "Use tools when needed..."
        output_format: "Format responses clearly..."
        communication_style: "Be concise and professional..."
        additional: "Additional context..."
```

**Config Location**: `agents.<agent_name>.prompt.prompt_slots`
**Related Files**:
- `pkg/reasoning/prompt_slots.go`
- `pkg/agent/agent.go` (buildPromptSlots)

---

#### 1.5.2 Full System Prompt Override
**Description**: Complete system prompt replacement (bypasses slots)

**Configuration**:
```yaml
agents:
  my_agent:
    prompt:
      system_prompt: |
        Complete system prompt here...
        This overrides all slot-based prompts.
```

**Config Location**: `agents.<agent_name>.prompt.system_prompt`
**Related Files**: `configs/coding.yaml` (Cursor example)

---

### 1.6 Tool System

#### 1.6.1 Built-in Tools

##### Execute Command
**Description**: Execute system commands with sandboxing

**Configuration**:
```yaml
tools:
  execute_command:
    type: "command"
    enabled: true
    allowed_commands: ["ls", "cat", "git"]
    working_directory: "./"
    max_execution_time: "30s"
    enable_sandboxing: true
```

**Config Location**: `tools.execute_command`
**Related Files**:
- `pkg/tools/command.go`
- `configs/coding.yaml`

---

##### Write File
**Description**: Create or overwrite files

**Configuration**:
```yaml
tools:
  write_file:
    type: "write_file"
    enabled: true
    max_file_size: 1048576           # 1MB
    allowed_extensions: [".go", ".yaml", ".txt"]
    forbidden_paths: ["/etc", "/sys"]
    working_directory: "./"
```

**Config Location**: `tools.write_file`
**Related Files**: `pkg/tools/file_writer.go`

---

##### Search Replace
**Description**: Precise file editing via text replacement

**Configuration**:
```yaml
tools:
  search_replace:
    type: "search_replace"
    enabled: true
    max_replacements: 100
    backup_enabled: true
    working_directory: "./"
```

**Config Location**: `tools.search_replace`
**Related Files**: `pkg/tools/search_replace.go`

---

##### Search Tool
**Description**: Semantic code/document search

**Configuration**:
```yaml
tools:
  search:
    type: "search"
    enabled: true
    document_stores: ["codebase"]
    default_limit: 10
    max_limit: 50
    max_results: 100
```

**Config Location**: `tools.search`
**Related Files**:
- `pkg/tools/search.go`
- `configs/coding.yaml`

---

##### Todo Write
**Description**: Task management for complex workflows

**Configuration**:
```yaml
tools:
  todo_write:
    type: "todo"
    enabled: true
```

**Config Location**: `tools.todo_write`
**Related Files**:
- `pkg/tools/todo.go`
- `configs/coding.yaml`

---

##### Agent Call (Multi-Agent)
**Description**: Delegate tasks to other agents

**Configuration**:
```yaml
agents:
  orchestrator:
    tools:
      - agent_call
    sub_agents: ["researcher", "analyst"]  # Optional filter
```

**Config Location**: Auto-registered for supervisor strategy
**Related Files**:
- `pkg/tools/agent_call.go`
- `configs/orchestrator-example.yaml`

---

#### 1.6.2 MCP (Model Context Protocol) Tools
**Description**: External tools via MCP servers (150+ integrations)

**Configuration**:
```yaml
tools:
  composio:
    type: "mcp"
    enabled: true
    server_url: "https://api.composio.dev/mcp"
    description: "150+ app integrations"
```

**Config Location**: `tools.<mcp_tool_name>`
**Environment Variables**: `MCP_URL` (zero-config mode)
**CLI Flags**: `--mcp-url URL`
**Related Files**:
- `pkg/tools/mcp.go`
- `configs/tools-mcp-example.yaml`
- `docs/mcp-custom-tools.md`

---

### 1.7 Document Stores & RAG

#### 1.7.1 Directory-Based Document Store
**Description**: Index local directories for semantic search

**Configuration**:
```yaml
document_stores:
  codebase:
    name: "codebase"
    source: "directory"
    path: "./src"
    include_patterns:
      - "*.go"
      - "*.py"
      - "*.md"
    exclude_patterns:
      - "**/node_modules/**"
      - "**/.git/**"
    max_file_size: 1048576           # 1MB
    watch_changes: true
    incremental_indexing: true       # Only index changed files
    database: "qdrant"
    embedder: "embedder"
```

**Config Location**: `document_stores.<store_name>`
**CLI Flags**: `--docs FOLDER` (zero-config mode)
**Related Files**:
- `pkg/context/document_store.go`
- `configs/coding.yaml`

---

### 1.8 Vector Databases

#### 1.8.1 Qdrant
**Description**: High-performance vector database

**Configuration**:
```yaml
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6333                       # Default port (both gRPC and REST)
    timeout: 300
    use_tls: false
    insecure: false
    api_key: "${QDRANT_API_KEY}"     # Optional
```

**Config Location**: `databases.<db_name>`
**Environment Variables**: `QDRANT_HOST`
**CLI Flags**: `--vectordb URL` (zero-config mode)
**Port Details**: Default port is 6333 (Qdrant uses same port for both gRPC and REST)
**Related Files**:
- `pkg/databases/qdrant.go`
- `configs/coding.yaml`

---

### 1.9 Embedders

#### 1.9.1 Ollama Embeddings
**Description**: Local embeddings via Ollama

**Configuration**:
```yaml
embedders:
  embedder:
    type: "ollama"
    model: "nomic-embed-text"
    host: "http://localhost:11434"
    dimension: 768
    timeout: 30
    max_retries: 3
```

**Config Location**: `embedders.<embedder_name>`
**Environment Variables**: `OLLAMA_HOST`
**CLI Flags**: `--embedder-model MODEL` (zero-config mode)
**Related Files**:
- `pkg/embedders/ollama.go`
- `configs/coding.yaml`

---

## 2. DEPLOYMENT & OPERATIONS

### 2.1 A2A Protocol Server

#### 2.1.1 Server Configuration
**Description**: Host agents via A2A protocol (gRPC + REST + JSON-RPC)

**Configuration**:
```yaml
global:
  a2a_server:
    host: "0.0.0.0"
    port: 8080                       # gRPC port (base)
    # REST: port+1 (8081), JSON-RPC: port+2 (8082)
    base_url: "https://agents.company.com"  # For discovery
```

**Config Location**: `global.a2a_server`
**CLI Flags**: `--port PORT`, `--host HOST`, `--a2a-base-url URL`
**CLI Command**: `hector serve --config config.yaml`
**Related Files**:
- `pkg/transport/server.go` (gRPC)
- `pkg/transport/rest_gateway.go` (REST)
- `pkg/transport/jsonrpc_handler.go` (JSON-RPC)
- `cmd/hector/main.go` (serve command)

---

#### 2.1.2 Agent Discovery
**Description**: A2A-compliant agent discovery endpoints

**Endpoints**:
- `GET /.well-known/agent-card.json` - Service card
- `GET /v1/agents` - List agents
- `GET /v1/agents/{id}/.well-known/agent-card.json` - Agent card

**Config Location**: Automatic (based on `a2a_server` config)
**Related Files**:
- `pkg/transport/discovery.go`
- `docs/a2a-compliance.md`

---

### 2.2 Authentication & Security

#### 2.2.1 JWT Authentication
**Description**: OAuth2/OIDC token validation (Auth0, Keycloak, etc.)

**Configuration**:
```yaml
global:
  auth:
    jwks_url: "https://auth0.com/.well-known/jwks.json"
    issuer: "https://auth0.com/"
    audience: "hector-api"
```

**Config Location**: `global.auth`
**Related Files**:
- `pkg/auth/jwt.go`
- `pkg/auth/middleware.go`
- `configs/auth-example.yaml`
- `docs/authentication.md`

---

#### 2.2.2 Agent-Level Security
**Description**: Per-agent security schemes (OpenAPI-style)

**Configuration**:
```yaml
agents:
  secure_agent:
    security:
      schemes:                       # Define security schemes (OpenAPI-style)
        BearerAuth:
          type: "http"               # http|apiKey|oauth2|openIdConnect|mutualTLS
          scheme: "bearer"           # For HTTP: bearer|basic
          bearer_format: "JWT"       # Optional format hint
          description: "JWT Bearer token authentication"
        ApiKeyAuth:                  # Example API key scheme
          type: "apiKey"
          in: "header"               # header|query|cookie
          name: "X-API-Key"          # Parameter name
      require:                       # List of OR'd AND sets
        - BearerAuth: []             # Require BearerAuth
        # OR
        # - ApiKeyAuth: []           # Alternative requirement
      jwks_url: "https://auth0.com/.well-known/jwks.json"
      issuer: "https://auth0.com/"
      audience: "hector-api"
```

**Config Location**: `agents.<agent_name>.security`
**Related Files**:
- `pkg/config/types.go` (SecurityConfig, SecurityScheme)
- `configs/security-example.yaml`

---

### 2.3 Task Management

#### 2.3.1 In-Memory Task Store (Default)
**Description**: Task persistence in memory (development)

**Configuration**:
```yaml
agents:
  my_agent:
    task:
      backend: "memory"              # Default
      worker_pool: 100               # Max concurrent tasks
```

**Config Location**: `agents.<agent_name>.task`
**Related Files**: `pkg/agent/task_service.go`

---

#### 2.3.2 SQL Task Store
**Description**: Task persistence in SQL database (production)

**Configuration**:
```yaml
agents:
  my_agent:
    task:
      backend: "sql"
      worker_pool: 100
      sql:
        driver: "postgres"           # postgres|mysql|sqlite
        host: "localhost"
        port: 5432
        database: "hector"
        username: "user"
        password: "${DB_PASSWORD}"
        ssl_mode: "disable"          # For postgres
        max_conns: 25
        max_idle: 5
```

**Config Location**: `agents.<agent_name>.task.sql`
**Related Files**:
- `pkg/agent/task_service_sql.go`
- `configs/task-sql-example.yaml`

---

### 2.4 Plugin System

#### 2.4.1 Plugin Discovery
**Description**: Automatic plugin discovery from directories

**Configuration**:
```yaml
plugins:
  plugin_discovery:
    enabled: true
    paths:
      - "./plugins"
      - "~/.hector/plugins"
    scan_subdirectories: true
```

**Config Location**: `plugins.plugin_discovery`
**Related Files**:
- `pkg/plugins/discovery.go`
- `docs/plugins.md`

---

#### 2.4.2 Plugin Types
**Description**: Extensible plugin architecture

**Plugin Categories**:
- `llm_provider` - Custom LLM providers
- `database_provider` - Custom vector databases
- `embedder_provider` - Custom embedding models
- `tool_provider` - Custom tools
- `reasoning_strategy` - Custom reasoning engines

**Configuration**:
```yaml
plugins:
  llm_providers:
    my_custom_llm:
      name: "my_custom_llm"
      type: "grpc"
      path: "./plugins/my_llm"
      enabled: true
      config:
        api_key: "${CUSTOM_API_KEY}"
```

**Config Location**: `plugins.<category>.<plugin_name>`
**Related Files**:
- `pkg/plugins/types.go`
- `pkg/plugins/grpc/` (gRPC plugin protocol)

---

## 3. CLI & INTERACTION MODES

### 3.1 CLI Modes

#### 3.1.1 Server Mode
**Description**: Host agents for multiple clients

**Command**: `hector serve [options]`
**Flags**:
- `--config FILE` - Configuration file
- `--port PORT` - Server port
- `--host HOST` - Server host
- `--a2a-base-url URL` - Public base URL
- Zero-config flags (see 3.2)

**Related Files**: `cmd/hector/main.go` (executeServeCommand)

---

#### 3.1.2 Client Mode
**Description**: Connect to remote Hector server

**Command**: `hector <command> [options] --server URL`
**Flags**:
- `--server URL` - Remote server URL
- `--token TOKEN` - Authentication token
- `--stream BOOL` - Enable streaming

**Commands**:
- `hector list --server URL`
- `hector info agent --server URL`
- `hector call agent "prompt" --server URL`
- `hector chat agent --server URL`

**Related Files**: `pkg/cli/commands.go`

---

#### 3.1.3 Local Mode
**Description**: In-process agent execution (no server)

**Command**: `hector <command> [options]`
**Supports**:
- Config file mode: `--config FILE`
- Zero-config mode: `--provider`, `--api-key`, etc.

**Commands**:
- `hector call "prompt"`
- `hector chat`
- `hector list`

**Related Files**: `pkg/cli/commands.go`

---

### 3.2 Zero-Config Mode
**Description**: Run agents without configuration file

**CLI Flags**:
```bash
--provider openai|anthropic|gemini  # Auto-detected from API key
--api-key KEY                       # Or use env vars
--base-url URL                      # Optional: custom endpoint
--model MODEL                       # Optional: specific model
--tools                             # Enable all local tools
--mcp-url URL                       # MCP server integration
--docs FOLDER                       # Enable RAG
```

**Default Models** (when --model not specified):
- OpenAI: `gpt-4o-mini` (cost-effective)
- Anthropic: `claude-sonnet-4-20250514` (latest)
- Gemini: `gemini-2.0-flash-exp` (experimental)

**Environment Variables**:
- `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`
- `MCP_URL`

**Example**:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
hector call "What is 2+2?" --model claude-sonnet-4
```

**Related Files**:
- `cmd/hector/main.go` (parseArgs)
- `pkg/config/config.go` (CreateZeroConfig)

---

### 3.3 CLI Commands

#### 3.3.1 Serve Command
**Description**: Start A2A server

**Usage**: `hector serve [options]`
**See**: 3.1.1 Server Mode

---

#### 3.3.2 List Command
**Description**: List available agents

**Usage**: `hector list [options]`
**Modes**: Server, Client, Local
**Related Files**: `pkg/cli/commands.go` (ListCommand)

---

#### 3.3.3 Info Command
**Description**: Get agent information (card)

**Usage**: `hector info <agent> [options]`
**Modes**: Server, Client, Local
**Related Files**: `pkg/cli/commands.go` (InfoCommand)

---

#### 3.3.4 Call Command
**Description**: Execute one-shot agent task

**Usage**:
```bash
# Local mode (no agent name)
hector call "prompt" [options]

# Client mode (agent name required)
hector call agent "prompt" --server URL

# Config mode (agent name required)
hector call agent "prompt" --config file.yaml
```

**Flags**: `--stream BOOL` (default: true)
**Related Files**: `pkg/cli/commands.go` (CallCommand)

---

#### 3.3.5 Chat Command
**Description**: Interactive multi-turn conversation

**Usage**:
```bash
# Local mode (no agent name)
hector chat [options]

# Client mode (agent name required)
hector chat agent --server URL

# Config mode (agent name required)
hector chat agent --config file.yaml
```

**Flags**: `--no-stream` (disable streaming)
**Related Files**: `pkg/cli/commands.go` (ChatCommand)

---

#### 3.3.6 Task Commands
**Description**: Manage async tasks

**Usage**:
```bash
hector task get agent task-id [options]
hector task cancel agent task-id [options]
```

**Related Files**: `pkg/cli/task_commands.go`

---

## 4. ADVANCED FEATURES

### 4.1 Multi-Agent Orchestration

#### 4.1.1 Supervisor Strategy
**See**: 1.3.2 Supervisor Strategy

**Key Features**:
- LLM-driven agent selection
- Task decomposition
- Result synthesis
- Sub-agent filtering

**Required Tool**: `agent_call`

---

#### 4.1.2 Agent Registry
**Description**: Central registry for agent discovery

**Visibility Filtering**:
- `public` - Discoverable and callable
- `internal` - Callable but not discoverable
- `private` - Only for local orchestrators

**Related Files**:
- `pkg/agent/registry.go`
- `pkg/agent/registry_service.go`

---

### 4.2 Structured Output

#### 4.2.1 JSON Schema
**See**: 1.2.4 Structured Output

**Use Cases**:
- Data extraction
- Form filling
- API responses

---

#### 4.2.2 XML Output
**Configuration**:
```yaml
llms:
  my-llm:
    structured_output:
      format: "xml"
```

**Use Cases**: Document generation, Claude-optimized output

---

#### 4.2.3 Enum Output
**Configuration**:
```yaml
llms:
  my-llm:
    structured_output:
      format: "enum"
      enum: ["yes", "no", "maybe"]
```

**Use Cases**: Classification, decision making

---

### 4.3 Reflection & Meta-Cognition

#### 4.3.1 Structured Reflection
**Description**: LLM-based tool execution analysis

**Configuration**:
```yaml
agents:
  my_agent:
    reasoning:
      enable_structured_reflection: true  # Default: true
```

**Features**:
- Post-tool execution analysis
- Error detection
- Quality assessment

**Related Files**: `pkg/reasoning/reflection.go`

---

#### 4.3.2 Completion Verification
**Description**: LLM-based task completion assessment

**Configuration**:
```yaml
agents:
  my_agent:
    reasoning:
      enable_completion_verification: true
```

**Features**:
- Assess task completion
- Identify missing actions
- Continue if incomplete

**Related Files**: `pkg/reasoning/completion.go`

---

### 4.4 Streaming

#### 4.4.1 Token-by-Token Streaming
**Description**: Real-time LLM output

**Configuration**:
```yaml
agents:
  my_agent:
    reasoning:
      enable_streaming: true         # Default: true
```

**Protocols**:
- CLI: Direct console output
- REST: Server-Sent Events (SSE)
- gRPC: Streaming RPC

**Related Files**:
- `pkg/llms/anthropic.go` (GenerateStreaming)
- `pkg/transport/rest_gateway.go` (SSE)

---

### 4.5 Context Management

#### 4.5.1 Smart Context Window
**Description**: Automatic context optimization

**Configuration**:
```yaml
agents:
  my_agent:
    memory:
      budget: 2000                   # Token budget
      threshold: 0.8                 # Trigger summarization
      target: 0.6                    # Target after compression
```

**Features**:
- Token counting
- Automatic summarization
- Context compression

**Related Files**: `pkg/memory/summary_buffer.go`

---

## 5. CONFIGURATION STRUCTURE

### 5.1 Global Configuration

**Structure**:
```yaml
version: "1.0"
name: "My Config"
description: "Description"

global:
  logging:
    level: "info"                    # debug|info|warn|error
    format: "text"                   # text|json
    output: "stdout"                 # stdout|stderr|file
  
  performance:
    max_concurrency: 4
    timeout: "15m"
  
  a2a_server:                        # See 2.1.1
    host: "0.0.0.0"
    port: 8080
    base_url: "https://..."
  
  auth:                              # See 2.2.1
    jwks_url: "..."
    issuer: "..."
    audience: "..."

llms:                                # See 1.2
  <provider_name>: {...}

databases:                           # See 1.8
  <db_name>: {...}

embedders:                           # See 1.9
  <embedder_name>: {...}

agents:                              # See 1.1
  <agent_name>: {...}

tools:                               # See 1.6
  <tool_name>: {...}

document_stores:                     # See 1.7
  <store_name>: {...}

plugins:                             # See 2.4
  plugin_discovery: {...}
  <category>:
    <plugin_name>: {...}
```

**Related Files**:
- `pkg/config/config.go` (Config struct)
- `pkg/config/types.go` (all config types)

---

### 5.2 Agent Configuration Schema

**Full Schema**:
```yaml
agents:
  <agent_name>:
    # Core Identity
    type: "native"                   # native|a2a (default: native)
    name: "Agent Name"               # Required
    description: "Description"       # Optional
    visibility: "public"             # public|internal|private (default: public)
    
    # External agent fields (type=a2a)
    url: "https://..."               # A2A endpoint
    credentials:                     # Optional auth
      type: "bearer"
      token: "..."
    
    # Native agent fields (type=native)
    llm: "llm-ref"
    database: "db-ref"               # For RAG/long-term memory
    embedder: "embedder-ref"         # For RAG/long-term memory
    document_stores: ["store1"]      # For RAG
    
    # Configuration
    prompt:                          # See 1.5
      prompt_slots: {...}
      # OR
      system_prompt: "..."
    
    memory:                          # See 1.4
      strategy: "summary_buffer"
      budget: 2000
      long_term: {...}
    
    reasoning:                       # See 1.3
      engine: "chain-of-thought"
      max_iterations: 100
      enable_streaming: true
      show_tool_execution: true
      show_thinking: true
      enable_structured_reflection: true
      enable_completion_verification: false
    
    search:                          # Legacy (use document_stores)
      models: [...]
      top_k: 5
    
    task:                            # See 2.3
      backend: "memory"
      worker_pool: 100
      sql: {...}
    
    tools: ["tool1", "tool2"]        # Tool references
    sub_agents: ["agent1"]           # For supervisor
    
    security:                        # See 2.2.2
      schemes: {...}
      require: [...]
```

**Related Files**: `pkg/config/types.go` (AgentConfig)

---

## 6. EXAMPLE CONFIGURATIONS

### 6.1 Essential Examples

**Location**: `/configs/`

1. **coding.yaml** (33KB)
   - Cursor-like coding assistant
   - Claude Sonnet 4, semantic search
   - Comprehensive tool set

2. **orchestrator-example.yaml** (6.3KB)
   - Multi-agent orchestration
   - Supervisor strategy
   - Agent delegation

3. **research-assistant.yaml** (6KB)
   - Multi-agent research system
   - Specialized agents

4. **mixed-agents-example.yaml** (5.3KB)
   - Native + external A2A agents
   - Hybrid architecture

5. **auth-example.yaml** (4.9KB)
   - JWT authentication
   - Agent visibility
   - OAuth2/OIDC integration

6. **long-term-memory-example.yaml** (4KB)
   - Vector-based memory
   - Semantic recall

7. **tools-mcp-example.yaml** (7.7KB)
   - MCP protocol integration
   - 150+ app integrations

8. **memory-strategies-example.yaml** (2.1KB)
   - Summary buffer vs buffer window

9. **security-example.yaml** (1.4KB)
   - Agent visibility
   - Tool sandboxing

10. **external-agent-example.yaml** (1.6KB)
    - Simple external agent

11. **task-sql-example.yaml** (849B)
    - SQL task persistence

**Related Files**: `/configs/README.md`

---

## 7. DOCUMENTATION STRUCTURE

### 7.1 Current Documentation

**Location**: `/docs/`

**Files**:
- `index.md` - Landing page
- `installation.md` - Installation guide
- `quick-start.md` - Quick start
- `agents.md` - Agent configuration
- `configuration.md` - Configuration reference
- `cli-guide.md` - CLI reference
- `tools.md` - Tool reference
- `memory.md` - Memory management
- `authentication.md` - Authentication
- `external-agents.md` - External agents
- `mcp-custom-tools.md` - MCP tools
- `structured-output.md` - Structured output
- `plugins.md` - Plugin system
- `a2a-compliance.md` - A2A protocol
- `api-reference.md` - API reference
- `architecture.md` - Architecture
- `tutorial-cursor.md` - Build Your Own Cursor
- `tutorial-multi-agent.md` - Multi-agent tutorial

---

## 8. KEY CODE MODULES

### 8.1 Package Structure

**Core Packages**:
- `pkg/agent/` - Agent implementation & orchestration
- `pkg/config/` - Configuration types & loading
- `pkg/reasoning/` - Reasoning strategies
- `pkg/memory/` - Memory management
- `pkg/tools/` - Tool system
- `pkg/llms/` - LLM providers
- `pkg/transport/` - A2A protocol servers
- `pkg/auth/` - Authentication
- `pkg/context/` - Document stores & RAG
- `pkg/databases/` - Vector databases
- `pkg/embedders/` - Embedding models
- `pkg/plugins/` - Plugin system
- `pkg/cli/` - CLI commands
- `cmd/hector/` - Main entry point

---

## 9. ENVIRONMENT VARIABLES

### 9.1 LLM Providers
- `OPENAI_API_KEY` - OpenAI API key (auto-loaded if not in config)
- `ANTHROPIC_API_KEY` - Anthropic API key (auto-loaded if not in config)
- `GEMINI_API_KEY` - Google Gemini API key (auto-loaded if not in config)

### 9.2 Infrastructure
- `QDRANT_HOST` - Qdrant host (default: localhost)
- `OLLAMA_HOST` - Ollama host (default: http://localhost:11434)
- `MCP_URL` - MCP server URL (for zero-config mode)

### 9.3 Configuration
- `HECTOR_CONFIG` - Path to config file (alternative to --config flag)

### 9.4 Authentication
- Database passwords, API keys (various, specified in configs with ${VAR} syntax)

### 9.5 External Agents
- `EXTERNAL_AGENT_JWT_TOKEN` - JWT token for external A2A agent authentication
- `EXTERNAL_API_KEY` - API key for external agent authentication
- `AGENT_USERNAME`, `AGENT_PASSWORD` - Basic auth for external agents

---

## 10. A2A PROTOCOL COMPLIANCE

### 10.1 Core Endpoints
- `POST /v1/agents/{id}/message:send` - Send message
- `POST /v1/agents/{id}/message:stream` - Streaming message
- `GET /v1/agents/{id}/task/{task_id}` - Get task
- `POST /v1/agents/{id}/task/{task_id}:cancel` - Cancel task

### 10.2 Discovery Endpoints
- `GET /.well-known/agent-card.json` - Service card
- `GET /v1/agents` - List agents
- `GET /v1/agents/{id}/.well-known/agent-card.json` - Agent card

### 10.3 Compliance Level
**100% A2A Compliant** - See `docs/a2a-compliance.md`

---

## NOTES

### Configuration Precedence
1. CLI flags (highest priority)
2. Configuration file
3. Environment variables
4. Default values (lowest priority)

### Zero-Config Philosophy
Hector supports complete zero-config operation via:
- Automatic provider detection from API keys
- Sensible defaults for all settings
- CLI flags for quick overrides

### Best Practices
- Use `configs/` examples as starting points
- Start with zero-config for testing
- Move to YAML config for production
- Use agent visibility for security
- Enable long-term memory for complex workflows

---

**END OF DOCUMENT**

