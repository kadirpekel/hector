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

## Built-In Tools

!!! tip "Permissive by Default 🎯"
    All file/command tools are **permissive by default** - they allow maximum flexibility while remaining secure. This makes Hector easy to use in zero-config mode.

    - **write_file**: Allow all file types by default (including extensionless files)
    - **execute_command**: Allow all commands by default (when sandboxing enabled)
    - **document_store**: Index all parseable file types by default (text files + .pdf/.docx/.xlsx)
    - **search_replace**: No file type restrictions

    You can opt-in to whitelist (`allowed_*`) or blacklist (`denied_*`) specific extensions/commands as needed.

Hector includes 6 ready-to-use tools:

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

### 4. search

Semantic code search (requires Qdrant + Ollama):

```yaml
databases:
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
    database: "qdrant"
    embedder: "embedder"
    tools: ["search"]
    document_stores:
      - name: "codebase"
        paths: ["./src/"]
```

**Use cases:** Finding relevant code, understanding codebases, discovering patterns

**Example:**
```
User: How does authentication work?
Agent: search("authentication implementation")
Agent: Found authentication in src/auth.go...
```

See [RAG & Semantic Search](rag.md) for setup.

---

### 5. todo_write

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

### 6. agent_call

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

See [How to Add Custom Tools](../how-to/add-custom-tools.md) for a complete guide.

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
        tool_usage: |
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
      show_tool_execution: true
      show_debug_info: true
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
    database: "qdrant"
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

## Next Steps

- **[RAG & Semantic Search](rag.md)** - Set up semantic code search
- **[Reasoning Strategies](reasoning.md)** - How agents use tools
- **[Multi-Agent Orchestration](multi-agent.md)** - Use agent_call
- **[How to Add Custom Tools](../how-to/add-custom-tools.md)** - Complete MCP guide

---

## Related Topics

- **[Agent Overview](overview.md)** - Understanding agents
- **[Configuration Reference](../reference/configuration.md)** - All tool options
- **[Build a Coding Assistant](../how-to/build-coding-assistant.md)** - Complete tutorial

