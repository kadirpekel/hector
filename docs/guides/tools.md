# Tools

Tools are the "hands" of an agent. They allow the LLM to interact with the outside world: read files, execute commands, query databases, or call APIs.

## Overview

| Type | Description | Use Case |
|------|-------------|----------|
| **MCP** | Connect to Model Context Protocol servers | External integrations, third-party tools |
| **Command** | Execute shell commands with security controls | System operations, scripting |
| **Function** | Built-in tool handlers (Go) | File ops, web search, agent-as-tool |

## Built-in Tool Handlers

Before configuring external tools, check if a built-in handler covers your needs:

| Handler | Description | Approval Required |
|---------|-------------|:-----------------:|
| `text_editor` | View, create, and edit files | Yes |
| `apply_patch` | Apply unified diff patches | Yes |
| `web_search` | Search the web | No |
| `web_fetch` | Fetch and extract web page content | No |
| `web_request` | Make arbitrary HTTP requests | Yes |
| `grep_search` | Search files by pattern | No |
| `todo_write` | Manage task checklists | No |

```yaml
tools:
  search:
    type: function
    handler: web_search

  editor:
    type: function
    handler: text_editor

  fetcher:
    type: function
    handler: web_fetch
```

---

## MCP Tools

The [Model Context Protocol](https://modelcontextprotocol.io) is the standard for connecting AI models to external systems. Hector supports all three MCP transports.

### Standard I/O (stdio)

Run an MCP server as a subprocess. Best for local tools.

```yaml
tools:
  filesystem:
    type: mcp
    command: npx
    args: [-y, "@modelcontextprotocol/server-filesystem", "/Users/me/projects"]
```

You can pass environment variables to the subprocess:

```yaml
tools:
  github:
    type: mcp
    transport: stdio
    command: npx
    args: [-y, "@modelcontextprotocol/server-github"]
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: ${GITHUB_TOKEN}
```

### SSE (Server-Sent Events)

Connect to a remote MCP server over HTTP with SSE streaming.

```yaml
tools:
  remote_db:
    type: mcp
    url: "http://localhost:8080/sse"
    transport: sse
```

### Streamable HTTP

The newer HTTP-based transport, preferred for cloud-hosted MCP servers.

```yaml
tools:
  remote_service:
    type: mcp
    url: "http://localhost:8080/mcp"
    transport: streamable-http
```

### Tool Filtering

An MCP server might expose dozens of tools, but you often only need a few. Use `filter` to whitelist specific tools:

```yaml
tools:
  github:
    type: mcp
    command: npx
    args: [-y, "@modelcontextprotocol/server-github"]
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: ${GITHUB_TOKEN}
    filter:
      - get_issue
      - create_issue
      - search_repositories
      # All other GitHub tools are hidden from the agent
```

!!! tip "Lazy Connection"
    MCP toolsets connect lazily. The connection is only established the first time an agent resolves its tools. This means unused MCP tools don't block startup or consume resources.

---

## Command Tools

Command tools allow agents to run shell commands. This is powerful but security-sensitive.

### Basic Usage

```yaml
tools:
  list_files:
    type: command
    command: ls
    args: ["-la"]
    allowed_commands: ["ls", "grep"]
    working_directory: "/tmp"
```

### Security Controls

Command tools have multiple layers of protection:

| Control | Description |
|---------|-------------|
| `allowed_commands` | Whitelist of permitted commands |
| `denied_commands` | Blacklist of blocked commands |
| `deny_by_default` | When `true`, only whitelisted commands work |
| `working_directory` | Constrains filesystem access |
| `max_execution_time` | Timeout (default: `5m`) |
| `require_approval` | Human must approve before execution |

**Default denied commands** (always blocked when sandboxing is active):

`rm`, `rmdir`, `sudo`, `su`, `chmod`, `chown`, `dd`, `mkfs`, `fdisk`, `mount`, `umount`, `kill`, `killall`, `reboot`, `shutdown`, `passwd`, `useradd`, `userdel`

**Default denied patterns** (regex, always blocked):

`rm -rf`, writes to `/dev/`, fork bombs, piped `wget|sh` or `curl|sh`, writes to `/etc/`, `chmod 777`, `--no-preserve-root`

### Recommended: Deny by Default

For production, explicitly whitelist the commands your agent needs:

```yaml
tools:
  safe_shell:
    type: command
    deny_by_default: true
    allowed_commands: ["npm", "docker", "git", "ls", "cat", "grep"]
    denied_commands: ["rm", "sudo"]
    working_directory: "/app/workspace"
    max_execution_time: "30s"
    require_approval: true
    approval_prompt: "Agent wants to run a shell command. Allow?"
```

---

## Function Tools (Programmatic)

When embedding Hector in a Go application, you can register native functions as tools. Hector auto-generates JSON schemas from Go struct tags.

### Defining a Function Tool

```go
package main

import (
    "github.com/verikod/hector/pkg/builder"
    "github.com/verikod/hector/pkg/tool"
)

// Define input schema via struct tags
type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=City name or coordinates"`
    Units    string `json:"units"    jsonschema:"enum=celsius|fahrenheit,default=celsius"`
}

func checkWeather(args WeatherInput) string {
    // Your implementation here
    return fmt.Sprintf("Weather in %s: 22°C, sunny", args.Location)
}

func main() {
    weatherTool := tool.NewFunctionTool(
        "check_weather",
        "Get current weather for a location",
        checkWeather,
    )

    agent, _ := builder.NewAgent("assistant").
        WithTool(weatherTool).
        Build()
    // ...
}
```

### Struct Tag Reference

| Tag | Description | Example |
|-----|-------------|---------|
| `json:"name"` | Field name in JSON | `json:"location"` |
| `jsonschema:"required"` | Mark as required | `jsonschema:"required"` |
| `jsonschema:"description=..."` | Field description | `jsonschema:"description=City name"` |
| `jsonschema:"default=..."` | Default value | `jsonschema:"default=celsius"` |
| `jsonschema:"enum=a\|b\|c"` | Allowed values | `jsonschema:"enum=celsius\|fahrenheit"` |
| `jsonschema:"minimum=N"` | Minimum numeric value | `jsonschema:"minimum=0"` |
| `jsonschema:"maximum=N"` | Maximum numeric value | `jsonschema:"maximum=100"` |

---

## Human-in-the-Loop (Approval)

Any tool can require human approval before execution. This is critical for sensitive actions.

```yaml
tools:
  deploy_prod:
    type: command
    command: "./deploy.sh"
    require_approval: true
    approval_prompt: "Agent wants to deploy to production. Proceed?"
```

### How It Works

1. Agent decides to call the tool
2. Execution **pauses**. The task transitions to `input_required` state
3. The API/UI receives an approval request with the tool name and arguments
4. User approves or denies
5. If approved, the tool executes; if denied, the agent is informed and continues reasoning

### Checking for Pending Approvals

```bash
# List tasks awaiting approval
curl http://localhost:8080/admin/tasks?status=input_required

# Approve a pending task
curl -X POST http://localhost:8080/agents/assistant \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"text": "approved"}]
      },
      "taskId": "task-id-here"
    },
    "id": 1
  }'
```

### Default Approval Settings

Some built-in handlers require approval by default for safety:

- `text_editor`, `apply_patch`, `web_request`: **require approval**
- `web_fetch`, `web_search`, `grep_search`, `todo_write`: **no approval needed**
- All command tools: **require approval** by default

Override with `require_approval: false` when you trust the agent's judgment for a specific tool.

---

## Streaming Tools

Tools can emit incremental output rather than returning a single result. This is useful for long-running operations where you want real-time progress.

Streaming tools implement `iter.Seq2[*Result, error]` and map to A2A `artifact-update` events with `append: true`. Use cases include:

- Docker build/pull output
- Package installation progress
- Sub-agent execution traces
- File processing pipelines

This is a programmatic API feature. See the [Go API Reference](../reference/go-api.md) for implementation details.

---

## Long-Running Tools

Tools that take significant time can declare themselves as long-running. Instead of blocking the LLM, they return a job ID immediately. The agent can poll for completion or continue with other work.

```go
type MyTool struct{}

func (t *MyTool) IsLongRunning() bool { return true }
```

The task stays in `working` state while the tool runs, and the result is injected into the conversation when complete.

---

## Agent-as-Tool

Any agent can be used as a tool by another agent. This enables the delegation pattern:

```yaml
agents:
  coordinator:
    llm: claude
    agent_tools: [researcher, fact_checker]

  researcher:
    llm: claude
    instruction: "Research the given topic thoroughly."

  fact_checker:
    llm: gpt4
    instruction: "Verify the accuracy of claims."
```

The coordinator stays in control. Child agents execute in isolated sessions and return structured results. This is different from `sub_agents` (transfer pattern) where control transfers to the child.

See the [Agents Guide](./agents.md) for more on composition patterns.

---

## Configuration Reference

Full tool configuration schema:

```yaml
tools:
  <tool-name>:
    type: mcp | function | command
    enabled: true                    # Enable/disable without removing

    # MCP options
    url: "..."                       # Server URL (SSE/streamable-http)
    transport: stdio | sse | streamable-http
    command: "..."                   # Executable (stdio transport)
    args: [...]                      # Command arguments
    env: { KEY: value }              # Environment variables
    filter: [tool1, tool2]           # Whitelist exposed tools

    # Function options
    handler: "web_search"            # Built-in handler name
    parameters: { ... }             # JSON schema override

    # Command options
    command: "ls"                    # Shell command to execute
    args: ["-la"]                    # Default arguments
    allowed_commands: [...]          # Whitelist
    denied_commands: [...]           # Blacklist
    deny_by_default: false           # Require explicit whitelist
    working_directory: "./"          # Working directory
    max_execution_time: "5m"         # Execution timeout

    # Security
    require_approval: false          # Human-in-the-loop
    approval_prompt: "..."           # Message shown to approver
```

## Next Steps

- [Secure your tools](./security.md)
- [Use tools in Agents](./agents.md)
- [Guardrails for tools](./guardrails.md)
- [Go API for custom tools](../reference/go-api.md)
- [Full Tools Reference](../reference/configuration.md#tools)
