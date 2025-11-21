---
title: Tools
description: Give agents capabilities through built-in tools, MCP servers, and plugins
---

# Tools

Tools give agents capabilities beyond language generation—they can execute commands, modify files, search code, call other agents, and integrate with external services.

## Three Ways to Add Tools

| Type | When to Use | Setup Time | Examples |
|------|-------------|------------|----------|
| **Built-in** | Common tasks | Instant | execute_command, write_file, search |
| **MCP Servers** | External services | 5-10 min | GitHub, Slack, Notion via Composio |
| **gRPC Plugins** | Custom tools | Hours/Days | Custom business logic, APIs |

---

## Tool Assignment

Hector uses a **consistent assignment pattern** for tools (same pattern as document stores and sub-agents):

| Configuration | Access | Behavior |
|--------------|--------|----------|
| `nil`/omitted | **All tools** | Permissive default - agent has access to all tools from the registry (excluding `internal: true` tools) |
| `[]` (explicitly empty) | **No tools** | Explicit restriction - agent has no access to any tools |
| `["tool1", ...]` | **Only those tools** | Scoped access - agent can only use the explicitly listed tools (visibility ignored for explicit lists) |

!!! note "Internal Tools"
    Tools marked with `internal: true` are **never** visible to agents, even when `tools` is omitted. They remain available for document stores and other system components. See [Tool Visibility](#tool-visibility) below.

**Example:**
```yaml
agents:
  # Access all tools (permissive default)
  general_assistant:
    # tools: not specified → accesses all tools
  
  # No tools (explicit restriction)
  isolated_agent:
    tools: []
  
  # Scoped access (explicit assignment)
  file_agent:
    tools:
      - "read_file"
      - "write_file"
      - "search_replace"
```

---

## Tool Visibility

Tools can be marked as `internal` to hide them from agents while still allowing document stores and other system components to use them.

### Internal Tools

Mark tools as `internal: true` when they should be available for system use (like document parsing) but not exposed to agents:

```yaml
tools:
  docling:
    type: "mcp"
    enabled: true
    internal: true  # Not visible to agents (used only for document parsing)
    server_url: "http://localhost:3000/mcp"
    description: "Docling - Advanced document parsing"

document_stores:
  knowledge_base:
    path: "./documents"
    mcp_parsers:
      tool_names: ["parse_document", "docling_parse"]  # Uses internal tools
      extensions: [".pdf", ".docx", ".pptx"]
```

**Behavior:**
- ✅ **Available in tool registry** - Tool is registered and discoverable
- ✅ **Document stores can use it** - MCP extractors can access internal tools for parsing
- ❌ **Hidden from agents** - Filtered out when `tools` is `nil` (auto-discovery mode)
- ❌ **Agent cannot call directly** - Even if explicitly listed, internal tools are system-only

**Use cases:**
- **MCP tools for document parsing** - Use Docling, Unstructured, etc. for parsing without exposing to agents
- **System-level tools** - Tools reserved for internal processing
- **Prevent tool pollution** - Keep agent tool lists clean when tools are only for parsing

**Default:** If `internal` is omitted or `false`, the tool is visible to agents (normal behavior).

!!! tip "Explicit Tool Lists Override Visibility"
    When agents explicitly list tools (`tools: ["tool1", "tool2"]`), visibility is ignored. However, internal tools are still filtered out even in explicit lists to prevent accidental exposure.

---

## Built-In Tools

!!! tip "Permissive by Default 🎯"
    All file/command tools are **permissive by default** - they allow maximum flexibility while remaining secure. This makes Hector easy to use in zero-config mode.

    - **write_file**: Allow all file types by default (including extensionless files)
    - **execute_command**: Allow all commands by default (when sandboxing enabled)
    - **document_store**: Index all parseable file types by default (text files + .pdf/.docx/.xlsx)
    - **search_replace**: No file type restrictions

    You can opt-in to whitelist (`allowed_*`) or blacklist (`denied_*`) specific extensions/commands as needed.

Hector includes 12 ready-to-use tools:

### 1. execute_command

Run shell commands securely:

```yaml
agents:
  dev:
    tools: ["execute_command"]

tools:
  execute_command:
    type: command
    
    enable_sandboxing: true  # Default: true (recommended for security)
    # Note: No allowed_commands = allow ALL commands (when sandboxing enabled)
    # To restrict, add: allowed_commands: ["ls", "cat", "grep", "git"]
    max_execution_time: "30s"
    working_directory: "./"
```

!!! warning "Security: Sandboxing Required"
    When `enable_sandboxing: false`, you **must** specify `allowed_commands` explicitly. The config will fail validation without it for security reasons.

**Use cases:** Running tests, building projects, checking git status

**Human-in-the-Loop (Tool Approval):**
Require user approval before executing commands:

```yaml
tools:
  execute_command:
    type: command
    requires_approval: true  # Pause task for approval
    approval_prompt: "Execute command: {input}?"
```

When `requires_approval: true`, the task pauses at `TASK_STATE_INPUT_REQUIRED` and waits for your approval. See [Human-in-the-Loop](human-in-the-loop.md) for details.

**Example:**
```
User: Run the tests
Agent: execute_command("npm test")
Agent: All tests passed!
```

---

### 2. write_file

Create or modify files:

```yaml
agents:
  coder:
    tools: ["write_file"]

tools:
  write_file:
    type: write_file
    
    allowed_paths: ["./src/", "./docs/"]
    max_file_size: 10485760  # 10MB in bytes
    # Note: No allowed_extensions = allow ALL file types by default
    # To whitelist: allowed_extensions: [".py", ".go", ".md"]
    # To blacklist: denied_extensions: [".exe", ".bin"]
```

!!! info "File Extension Management"
    - **Default**: Allow all file types (including extensionless files like `Makefile`)
    - **Whitelist**: Set `allowed_extensions` to only allow specific types
    - **Blacklist**: Set `denied_extensions` to block specific types
    - **Precedence**: Blacklist > Whitelist > Default

**Use cases:** Code generation, documentation creation, config files

**Human-in-the-Loop (Tool Approval):**
Require approval before writing files:

```yaml
tools:
  write_file:
    type: write_file
    requires_approval: true  # Pause task for approval
    approval_prompt: "Write file: {tool} with content: {input}?"
```

**Example:**
```
User: Create a README
Agent: write_file("README.md", "# My Project...")
Agent: Created README.md
```

---

### 3. search_replace

Find and replace in files:

```yaml
agents:
  refactor:
    tools: ["search_replace"]

tools:
  search_replace:
    
    allowed_paths: ["./src/"]
    backup: true
```

**Use cases:** Refactoring, renaming, updating code patterns

**Example:**
```
User: Rename function oldName to newName
Agent: search_replace(pattern="oldName", replacement="newName", files=["src/*.js"])
Agent: Updated 5 files
```

---

### 4. read_file

Read file contents with optional line ranges:

```yaml
agents:
  coder:
    tools: ["read_file"]

tools:
  read_file:
    type: read_file
    max_file_size: 10485760  # 10MB
    working_directory: "./"
```

**Use cases:** Understanding code structure, reviewing files before edits, inspecting specific sections

**Example:**
```
User: Show me the main function
Agent: read_file("src/main.go", start_line=10, end_line=50)
Agent: Here's the main function...
```

---

### 5. apply_patch

Apply contextual patches to files safely:

```yaml
agents:
  refactor:
    tools: ["apply_patch"]

tools:
  apply_patch:
    type: apply_patch
    max_file_size: 10485760  # 10MB
    context_lines: 3  # Require context before/after changes
    working_directory: "./"
```

**Use cases:** Safe code edits, refactoring with context validation, making precise changes

**Example:**
```
User: Update the function to handle errors
Agent: apply_patch("src/handler.go", 
  old_string="func process() { ... }",
  new_string="func process() error { ... }")
Agent: ✅ Patch applied successfully
```

!!! tip "Safer than search_replace"
    `apply_patch` validates surrounding context before applying changes, making it safer for code modifications. Use it when you need confidence that the change location is correct.

---

### 6. grep_search

Search for patterns using regular expressions:

```yaml
agents:
  explorer:
    tools: ["grep_search"]

tools:
  grep_search:
    type: grep_search
    max_results: 1000
    context_lines: 2
    working_directory: "./"
```

**Use cases:** Finding exact strings, searching for function definitions, locating TODO comments

**Example:**
```
User: Find all TODO comments
Agent: grep_search(pattern="TODO:", path=".", recursive=true)
Agent: Found 12 TODO comments across 8 files...
```

---

### 7. search

Semantic code search (requires vector store + embedder):

```yaml
vector_stores:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334

embedders:
  embedder:
    type: "ollama"
    model: "nomic-embed-text"

agents:
  coder:
    vector_store: "qdrant"
    embedder: "embedder"
    tools: ["search"]
    document_stores:
      - name: "codebase"
        paths: ["./src/"]
```

**Use cases:** Finding code by meaning, discovering similar patterns, answering questions from codebase

**Advanced search modes:**
- `search_mode: "hybrid"` - Combines keyword and vector search
- `search_mode: "multi_query"` - Expands query into multiple variations
- `search_mode: "hyde"` - Uses hypothetical document embeddings
- `rerank.enabled: true` - LLM-based re-ranking for better results

See [RAG & Semantic Search](rag.md) and [Search Architecture](../reference/architecture/search-architecture.md) for details.

---

### 8. evaluate_rag

Evaluate RAG system performance by analyzing query, retrieved documents, and generated answer:

```yaml
agents:
  evaluator:
    llm: "gpt-4o-mini"  # LLM for evaluation
    tools: ["evaluate_rag"]
```

**Metrics calculated:**
- **Context Precision**: Proportion of retrieved contexts that are relevant
- **Context Recall**: Proportion of relevant contexts retrieved
- **Answer Relevance**: How relevant the answer is to the query
- **Faithfulness**: How faithful the answer is to retrieved contexts
- **Answer Correctness**: Overall correctness score (average of relevance and faithfulness)

**Use cases:** Measuring RAG quality, benchmarking search improvements, validating system performance

**Example:**
```json
{
  "name": "evaluate_rag",
  "parameters": {
    "query": "How does authentication work?",
    "retrieved_docs": [
      {
        "id": "doc1",
        "content": "Authentication uses JWT tokens...",
        "score": 0.85
      }
    ],
    "generated_answer": "Authentication uses JWT tokens for secure access...",
    "ground_truth": "Optional: Expected answer for comparison"
  }
}
```

**Returns:**
- Metrics scores (0.0-1.0)
- Latency measurement
- Full evaluation result with details

See [Search Architecture](../reference/architecture/search-architecture.md#evaluation-tools) for complete documentation.

---

### 9. todo_write

Task tracking for agents:

```yaml
agents:
  planner:
    tools: ["todo_write"]
    reasoning:
      engine: "chain-of-thought"  # Works best with chain-of-thought
```

**Use cases:** Breaking down complex tasks, tracking progress

**Example:**
```
User: Build a user authentication system
Agent: todo_write([
  {id: "1", content: "Create user model", status: "in_progress"},
  {id: "2", content: "Add password hashing", status: "pending"},
  {id: "3", content: "Create login endpoint", status: "pending"}
])
Agent: I'll start with the user model...
```

---

### 10. agent_call

Call other agents:

```yaml
agents:
  coordinator:
    tools: ["agent_call"]
    reasoning:
      engine: "supervisor"  # Best with supervisor reasoning
  
  researcher:
    # Specialist agent
  
  writer:
    # Specialist agent
```

**Use cases:** Multi-agent orchestration, task delegation

**Example:**
```
User: Research and write about AI
Coordinator: agent_call("researcher", "Research AI trends")
Coordinator: agent_call("writer", "Write article about: ...")
Coordinator: Here's the complete article!
```

See [Multi-Agent Orchestration](multi-agent.md) for details.

---

### 11. web_request

Make HTTP requests to external APIs and web services:

```yaml
agents:
  api_assistant:
    tools: ["web_request"]

tools:
  web_request:
    type: web_request
    timeout: "30s"
    max_retries: 3
    max_request_size: 1048576  # 1MB
    max_response_size: 10485760  # 10MB
    allowed_domains:  # Optional: restrict to specific domains
      - "api.example.com"
      - "*.github.com"
    denied_domains:  # Optional: block specific domains
      - "internal.company.com"
    allowed_methods:  # Optional: restrict HTTP methods
      - "GET"
      - "POST"
    allow_redirects: true
    max_redirects: 5
    user_agent: "Hector-Agent/1.0"
```

**Use cases:** Calling REST APIs, fetching web data, integrating with external services

**Example:**
```
User: Get the weather for San Francisco
Agent: web_request(
  url="https://api.weather.com/forecast",
  method="GET",
  headers={"Authorization": "Bearer token"}
)
Agent: The weather in San Francisco is...
```

!!! tip "Security Best Practices"
    - Use `allowed_domains` to restrict which APIs can be called
    - Set `max_request_size` and `max_response_size` to prevent abuse
    - Consider using `denied_domains` to block internal/private endpoints
    - Configure `allowed_methods` to limit HTTP verbs (e.g., GET only for read-only agents)

---

### 12. generate_image

Generate images from text prompts using DALL-E 3:

```yaml
agents:
  creative:
    tools: ["generate_image"]

tools:
  generate_image:
    type: "generate_image"
    config:
      api_key: "${OPENAI_API_KEY}"  # Required
      model: "dall-e-3"              # Default: dall-e-3
      size: "1024x1024"              # Default: 1024x1024
      quality: "standard"            # standard or hd
      style: "vivid"                 # vivid or natural
      timeout: "60s"                 # Default: 60s
```

**Tool Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `prompt` | string | ✅ | Text description of image to generate |
| `size` | string | ❌ | Image size (e.g., "1024x1024", "1792x1024", "1024x1792") |
| `quality` | string | ❌ | "standard" or "hd" |
| `style` | string | ❌ | "vivid" or "natural" |

**Use cases:** Creative image generation, visual content creation, design mockups

**Example:**
```
User: Generate an image of a sunset over mountains
Agent: generate_image(
  prompt="A beautiful sunset over snow-capped mountains",
  size="1024x1024",
  quality="hd",
  style="vivid"
)
Agent: Image generated successfully: https://oaidalleapiprodscus.blob.core.windows.net/...
```

**Configuration Options:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `api_key` | string | Required | OpenAI API key for DALL-E |
| `model` | string | `dall-e-3` | Model to use (currently only dall-e-3 supported) |
| `size` | string | `1024x1024` | Image dimensions |
| `quality` | string | `standard` | Image quality: `standard` or `hd` |
| `style` | string | `vivid` | Image style: `vivid` or `natural` |
| `timeout` | string | `60s` | Request timeout |

!!! note "API Key Required"
    The `generate_image` tool requires an OpenAI API key configured in the tool's `config.api_key` field or via the `OPENAI_API_KEY` environment variable.

---

### 13. screenshot_page

**Status:** ⚠️ Placeholder (not yet implemented)

Take screenshots of web pages (requires headless browser integration):

```yaml
tools:
  screenshot_page:
    type: "screenshot_page"
    config:
      timeout: "30s"
```

**Tool Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | ✅ | The URL to screenshot |

**Use cases:** Web page capture, visual testing, documentation

**Note:** This tool is currently a placeholder and will return an error when called. Full implementation requires headless browser integration (Chrome DevTools Protocol, Playwright, etc.).

**Future Implementation:**
- Headless browser integration (Chrome DevTools Protocol or Playwright)
- Full page screenshots
- Element-specific screenshots
- PDF generation

---

### 14. delete_file

Delete files from the file system:

```yaml
agents:
  cleanup:
    tools: ["delete_file"]

tools:
  delete_file:
    type: delete_file
    enabled: true
    requires_approval: true  # ⭐ Recommended: require approval for deletions
    approval_prompt: |
      🗑️  File Deletion Request
      
      Delete: {input}
      
      This action cannot be undone. Approve?
    working_directory: "./"
```

**Use cases:** Cleanup operations, removing temporary files, file management

**Example:**
```
User: Delete the old log files
Agent: delete_file("logs/old.log")
Agent: ✅ File deleted successfully
```

!!! warning "Dangerous Operation"
    File deletion is irreversible. Always enable `requires_approval: true` for production deployments. See [Human-in-the-Loop](human-in-the-loop.md) for approval configuration.

!!! note "Implementation Status"
    The `delete_file` tool is referenced in configuration examples but may require implementation. If you encounter issues, consider using `execute_command` with `rm` as an alternative (with proper sandboxing).

---

## Enabling Tools

### Option 1: Enable All (Simple)

```yaml
agents:
  assistant:
    tools: ["execute_command", "write_file", "search_replace"]
```

Uses default configurations for all tools.

### Option 2: Configure Individually (Recommended)

```yaml
agents:
  coder:
    tools: ["execute_command", "write_file"]

tools:
  execute_command:
    
    allowed_commands: ["npm", "git", "python"]
    max_execution_time: "60s"
  
  write_file:
    
    allowed_paths: ["./src/", "./tests/"]
    max_file_size: "5MB"
```

---

## MCP (Model Context Protocol) Servers

MCP allows agents to connect to external services and tools through standardized servers.

### What is MCP?

MCP is an open protocol for connecting AI agents to data sources and tools. Think of it as a universal adapter for agent integrations.

**Popular MCP Servers:**
- **Composio** - 150+ integrations (GitHub, Slack, Notion, Gmail, etc.)
- **Mem0** - Persistent memory management
- **Custom** - Build your own MCP servers

### Quick Setup

**1. Start an MCP Server:**

```bash
# Using Composio (example)
npx @composio/cli server --port 3000
```

**2. Configure in Hector:**

```yaml
tools:
  mcp_tools:
    - server:
        url: "http://localhost:3000"
        protocol: "mcp"
      

agents:
  assistant:
    tools: ["github_*", "slack_*"]  # Glob patterns for MCP tools
```

**3. Use the Tools:**

```yaml
agents:
  developer:
    tools: ["github_create_issue", "slack_send_message"]
```

```
User: Create a GitHub issue for the bug
Agent: github_create_issue(title="Fix bug", body="...")
Agent: Issue created: #123
```

### Composio Integration

Access 150+ apps through Composio:

```yaml
tools:
  mcp_tools:
    - server:
        url: "http://localhost:3000"
        protocol: "mcp"
      auth:
        type: "api_key"
        api_key: "${COMPOSIO_API_KEY}"
      

agents:
  automation:
    tools: [
      "github_*",           # All GitHub tools
      "slack_send_message", # Specific Slack tool
      "notion_create_page"  # Notion integration
    ]
```

### Custom MCP Servers

Build your own MCP server (5-10 minutes):

**Python Example:**

```python
from mcp import Server, Tool

server = Server()

@server.tool()
def fetch_weather(city: str) -> str:
    """Fetch weather for a city"""
    # Your logic here
    return f"Weather in {city}: Sunny, 72°F"

server.run(port=3000)
```

**Configure in Hector:**

```yaml
tools:
  mcp_tools:
    - server:
        url: "http://localhost:3000"
        protocol: "mcp"
      

agents:
  assistant:
    tools: ["fetch_weather"]
```

See [Adding Custom Tools with MCP](../blog/posts/adding-custom-tools-with-mcp.md) for a complete guide.

---

## gRPC Plugins

For advanced use cases, build gRPC plugins in any language.

### When to Use Plugins

- **High Performance** - Millions of operations
- **Complex Logic** - Multi-step tool workflows
- **Custom LLMs** - Proprietary language models
- **Enterprise Integration** - Internal APIs

### Plugin Configuration

```yaml
plugins:
  tools:
    - name: "my-custom-tools"
      protocol: "grpc"
      path: "/path/to/plugin"
      config:
        api_key: "${MY_API_KEY}"

agents:
  assistant:
    tools: ["plugin:my-custom-tools:fetch_data"]
```

### Plugin Development

See [Plugin System](../reference/architecture.md#plugin-system) for implementation details.

---

## Tool Configuration Reference

### Security Options

```yaml
tools:
  execute_command:
    
    allowed_commands: ["ls", "cat"]      # Whitelist
    denied_commands: ["rm", "dd"]        # Blacklist
    max_execution_time: "30s"
    working_directory: "./"
    
  write_file:
    
    allowed_paths: ["./src/"]
    denied_paths: ["./secrets/"]
    max_file_size: "10MB"
    create_directories: true
```

### MCP Server Options

```yaml
tools:
  # Internal MCP tool (for document parsing only)
  docling:
    type: "mcp"
    enabled: true
    internal: true  # Hide from agents, available for document stores
    server_url: "http://localhost:3000/mcp"
    description: "Docling - Document parsing"
    
    # TLS configuration (optional)
    insecure_skip_verify: false  # Skip certificate verification (dev/test only)
    ca_certificate: ""          # Path to custom CA certificate file

  # Regular MCP tools (visible to agents)
  mcp_tools:
    - server:
        url: "http://localhost:3000"
        protocol: "mcp"
        timeout: "30s"
      auth:
        type: "bearer"  # bearer|api_key|basic
        token: "${MCP_TOKEN}"
      
      tools:  # Optional: specific tools only
        - "github_create_issue"
        - "slack_send_message"
```

**Internal MCP Tools for Document Parsing:**

When MCP tools are used exclusively for document parsing (not for agent use), mark them as `internal: true`:

```yaml
tools:
  docling:
    type: "mcp"
    enabled: true
    internal: true  # Prevents agent tool list pollution
    server_url: "http://localhost:3000/mcp"

document_stores:
  knowledge_base:
    path: "./documents"
    mcp_parsers:
      tool_names: ["parse_document", "docling_parse"]
      extensions: [".pdf", ".docx", ".pptx"]
```

This keeps agent tool lists clean while allowing document stores to use MCP tools for advanced parsing.

---

## Tool Selection Best Practices

### Start Small

```yaml
# ✅ Good: Enable only needed tools
agents:
  assistant:
    tools: ["write_file", "search"]

# ❌ Bad: Enable everything
agents:
  assistant:
    tools: ["*"]  # Too many options confuse the agent
```

### Match Tools to Tasks

```yaml
# Coding assistant
agents:
  coder:
    tools: ["execute_command", "write_file", "search_replace", "search"]

# Research assistant
agents:
  researcher:
    tools: ["search", "write_file"]

# Coordinator
agents:
  coordinator:
    tools: ["agent_call", "todo_write"]
    reasoning:
      engine: "supervisor"
```

### Prompt for Tool Usage

```yaml
agents:
  assistant:
    tools: ["write_file", "execute_command"]
    prompt:
      prompt_slots:
        user_guidance: |
          Use tools proactively:
          - write_file: Create and modify files
          - execute_command: Run tests and checks
          
          Always test your changes after making them.
```

---

## Debugging Tools

### See Tool Execution

```yaml
agents:
  debug:
    reasoning:
    tools: ["write_file"]
```

Output shows:
```
[TOOL] write_file(path="test.txt", content="Hello")
[TOOL RESULT] Success: Created test.txt
```

### Test Tools Individually

```bash
# Call agent with specific tool request
hector call coder "Use write_file to create hello.txt with 'Hello World'"
```

---

## Examples by Use Case

### Coding Assistant

```yaml
databases:
  qdrant:
    type: "qdrant"

embedders:
  embedder:
    type: "ollama"
    model: "nomic-embed-text"

agents:
  coder:
    vector_store: "qdrant"
    embedder: "embedder"
    tools: ["execute_command", "write_file", "search_replace", "search"]
    document_stores:
      - name: "codebase"
        paths: ["./src/"]
    
    prompt:
      system_role: |
        You are an expert programmer. Use tools to:
        - Search for relevant code (search)
        - Create/modify files (write_file)
        - Refactor code (search_replace)
        - Run tests (execute_command)
```

### DevOps Automation

```yaml
agents:
  devops:
    tools: ["execute_command"]

tools:
  execute_command:
    
    allowed_commands: ["docker", "kubectl", "git", "terraform"]
    max_execution_time: "120s"
```

### Multi-Agent System

```yaml
agents:
  coordinator:
    tools: ["agent_call", "todo_write"]
    reasoning:
      engine: "supervisor"
  
  researcher:
    tools: ["search"]
  
  writer:
    tools: ["write_file"]
```

---

## Decision Guide

**Choose Built-in Tools when:**
- Need basic capabilities (files, commands)
- Want instant setup
- Prefer security controls

**Choose MCP Servers when:**
- Integrating external services (GitHub, Slack)
- Need 3rd party APIs
- Want rapid prototyping

**Choose gRPC Plugins when:**
- Building custom LLMs
- Need high performance
- Have complex business logic
- Enterprise integrations

---

## Vision Tools

Hector includes vision tools for image generation and processing. See [Multi-Modality Support](multi-modality.md) for complete documentation on sending images to agents and using vision tools.

**Vision Tools:**
- **[generate_image](#12-generate_image)** - Generate images using DALL-E 3
- **[screenshot_page](#13-screenshot_page)** - Take web page screenshots (placeholder)

---

## Next Steps

- **[Multi-Modality Support](multi-modality.md)** - Send images to agents and use vision tools
- **[RAG & Semantic Search](rag.md)** - Set up semantic code search
- **[Reasoning Strategies](reasoning.md)** - How agents use tools
- **[Multi-Agent Orchestration](multi-agent.md)** - Use agent_call
- **[Adding Custom Tools with MCP](../blog/posts/adding-custom-tools-with-mcp.md)** - Complete MCP guide

---

## Related Topics

- **[Agent Overview](overview.md)** - Understanding agents
- **[Configuration Reference](../reference/configuration.md)** - All tool options
- **[Building a Coding Assistant](../blog/posts/building-a-coding-assistant.md)** - Complete tutorial

