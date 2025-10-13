---
layout: default
title: Tools & Extensions
nav_order: 2
parent: Core Guides
description: "Built-in tools, MCP protocol, gRPC plugins, and custom integrations"
---

# Tools & Extensions

Complete guide to Hector's extensible tool system

---

## Overview

Hector provides a **powerful, extensible tool system** that gives agents capabilities beyond language generation. Tools can be:
- **Built-in** - 5 ready-to-use tools for common tasks
- **MCP Servers** - Connect to 150+ integrations via Model Context Protocol
- **gRPC Plugins** - Build custom tools in any language

> **📝 Note:** Individual tools use `enabled: true/false` for easy toggling. This is different from service-level configurations (A2A server, auth, memory) where presence of configuration implies it's enabled.

### Architecture

```
┌─────────────────────────────────────────────────────┐
│     Global Tool Registry (Loaded at Startup)       │
│  All tools defined once in config.yaml             │
└────────┬────────────────┬──────────────┬───────────┘
         │                │              │
         ▼                ▼              ▼
┌────────────────┐ ┌──────────────┐ ┌─────────────┐
│ Built-in Tools │ │  MCP Servers │ │gRPC Plugins │
│                │ │              │ │             │
│ • command      │ │ • Composio   │ │ • Custom    │
│ • write_file  │ │ • Mem0       │ │   LLMs      │
│ • search       │ │ • Custom     │ │ • Custom    │
│ • todo         │ │   servers    │ │   tools     │
│ • search_replace│ │              │ │             │
└────────────────┘ └──────────────┘ └─────────────┘
```

**Key Concepts:**
- **Tool** - A single capability (e.g., "execute_command")
- **Tool Source** - A provider of tools (local, MCP server, plugin)
- **Tool Registry** - Centralized discovery and execution

---

## Table of Contents

- [Built-in Tools](#built-in-tools)
- [MCP Integration](#mcp-integration)
- [gRPC Plugins](#grpc-plugins)
- [Configuration](#configuration)
- [Security](#security)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Built-in Tools

Hector includes 5 production-ready tools for common agent tasks.

### 1. execute_command

Execute shell commands securely with whitelisting.

**Configuration:**
```yaml
tools:
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
      - "curl"
    working_directory: "./"
    max_execution_time: "30s"
    enable_sandboxing: true
```

**Usage in Agent:**
```
Agent: Let me check what files are in this directory
Tool Call: execute_command(command="ls -la")
Result: total 128
-rw-r--r--  1 user  staff  1234 Oct  5 main.go
drwxr-xr-x  5 user  staff   160 Oct  5 cmd/
```

**Security:**
- **Whitelist only** - Only explicitly allowed commands can run
- **Timeout protection** - Commands killed after max_execution_time
- **Path restrictions** - Optional working_directory constraint
- **Sandboxing** - Optional isolation (platform-dependent)

### 2. write_file

Create and modify files safely.

**Configuration:**
```yaml
tools:
  write_file:
    type: "write_file"
    enabled: true
    allowed_extensions:
      - ".go"
      - ".yaml"
      - ".md"
      - ".txt"
    forbidden_paths:
      - "/etc"
      - "/usr"
      - "/bin"
    max_file_size: 10485760  # 10MB
```

**Usage in Agent:**
```
Agent: I'll create a README file
Tool Call: write_file(
  path="README.md",
  content="# My Project\n\nDescription..."
)
Result: Successfully wrote 125 bytes to README.md
```

**Security:**
- **Extension whitelist** - Only allowed file types can be written
- **Path blacklist** - Forbidden system directories
- **Size limits** - Prevent massive file writes
- **Validation** - Path traversal protection

### 3. search

Semantic search across document stores (RAG).

**Configuration:**
```yaml
tools:
  search:
    type: "search"
    enabled: true
    document_stores:
      - "codebase_docs"
      - "api_reference"
    default_limit: 10
    max_limit: 50
    enabled_search_types:
      - "content"
      - "file"
      - "function"
```

**Usage in Agent:**
```
Agent: Let me search the documentation
Tool Call: search(
  query="authentication setup",
  stores=["codebase_docs"],
  limit=5
)
Result: Found 3 relevant documents:
1. docs/auth.md - "Authentication Configuration"
2. examples/auth.go - "Example: JWT Setup"
3. README.md - "Security Overview"
```

**Features:**
- **Semantic search** - Vector similarity, not just keywords
- **Multiple stores** - Search across different knowledge bases
- **Configurable limits** - Control result counts
- **Type filtering** - Search by content, file, function, etc.

### 4. search_replace

Find and replace text in files.

**Configuration:**
```yaml
tools:
  search_replace:
    type: "search_replace"
    enabled: true
    max_replacements: 100
    backup_enabled: true
```

**Usage in Agent:**
```
Agent: I'll rename that function
Tool Call: search_replace(
  file="api.go",
  old="getUserData",
  new="fetchUserData",
  all=true
)
Result: Replaced 5 occurrences in api.go
```

**Features:**
- **Regex support** - Pattern-based replacements
- **Dry-run mode** - Preview changes before applying
- **Backup files** - Automatic backups before modification
- **Limits** - Prevent accidental mass replacements

### 5. todo_write

Task management and tracking.

**Configuration:**
```yaml
tools:
  todo:
    type: "todo"
    enabled: true
```

**Usage in Agent:**
```
Agent: Breaking this task down into steps
Tool Call: todo_write(
  tasks=[
    {id: "1", content: "Research options", status: "in_progress"},
    {id: "2", content: "Compare solutions", status: "pending"},
    {id: "3", content: "Write report", status: "pending"}
  ]
)
Result: Created 3 tasks
```

**Features:**
- **Status tracking** - pending, in_progress, completed, cancelled
- **Progress updates** - Real-time task status
- **Agent coordination** - Share task lists across agent calls

---

## MCP Integration

**Model Context Protocol (MCP)** is an open standard that enables agents to connect to external tools and data sources.

> 🔥 **Want to add custom tools?** See **[Building Custom MCP Tools in 5 Minutes](MCP_CUSTOM_TOOLS.md)** - The fastest way to extend Hector with domain-specific capabilities.

### What is MCP?

MCP is like a **universal adapter** for AI tools:
- **Standard protocol** - JSON-RPC 2.0 over HTTP/SSE
- **Auto-discovery** - Tools self-describe their capabilities
- **Provider ecosystem** - Composio, Mem0, Browserbase, custom servers
- **Language-agnostic** - Python, TypeScript, Go, any language
- **Build your own** - Custom tools in 5-10 minutes

**Why MCP over custom integrations?**
- ✅ **Zero custom code** - Just point to a URL
- ✅ **150+ integrations** - Via providers like Composio
- ✅ **Standardized** - One protocol for everything
- ✅ **Community** - Growing ecosystem of MCP servers
- ✅ **Fast custom tools** - Build in Python/TypeScript in minutes

### Quick Start

**1. Connect to MCP Server:**
```yaml
tools:
  composio_tools:
    type: "mcp"
    enabled: true
    server_url: "https://api.composio.dev/mcp"
    description: "Composio - 150+ app integrations"
```

**2. Hector automatically discovers tools:**
```
🔍 Discovering tools from MCP server: composio_tools (https://api.composio.dev/mcp)
✅ MCP source composio_tools discovered 150 tools
```

**3. Agent can now use them:**
```
Agent: Let me send a Slack message
Tool Call: slack_send_message(
  channel="#general",
  text="Deployment complete!"
)
```

### Popular MCP Providers

#### 1. Composio - 150+ App Integrations

**Connect to GitHub, Slack, Gmail, Jira, and more**

```yaml
tools:
  composio:
    type: "mcp"
    enabled: true
    server_url: "https://api.composio.dev/mcp"
    description: "Composio - Enterprise app integrations"
```

**Example tools:**
- `github_create_issue` - Create GitHub issues
- `slack_send_message` - Post to Slack channels
- `gmail_send_email` - Send emails via Gmail
- `jira_create_ticket` - Create Jira tickets
- 146+ more...

**Setup:** Get API key from [Composio](https://composio.dev)

#### 2. Mem0 - Memory & Personalization

**Add persistent memory to your agents**

```yaml
tools:
  mem0:
    type: "mcp"
    enabled: true
    server_url: "https://api.mem0.ai/mcp"
    description: "Mem0 - Agent memory layer"
```

**Example tools:**
- `mem0_store` - Store user preferences/facts
- `mem0_recall` - Retrieve relevant memories
- `mem0_search` - Search memory by query

**Use cases:**
- User preferences ("Alice prefers Python")
- Conversation context across sessions
- Long-term agent knowledge

#### 3. Browserbase - Browser Automation

**Headless browser capabilities for agents**

```yaml
tools:
  browserbase:
    type: "mcp"
    enabled: true
    server_url: "https://api.browserbase.com/mcp"
```

**Example tools:**
- `browser_navigate` - Go to URL
- `browser_click` - Click elements
- `browser_screenshot` - Capture screenshots
- `browser_extract` - Extract page data

#### 4. Your Custom MCP Server

**Build your own tools for specific needs**

```yaml
tools:
  my_custom_tools:
    type: "mcp"
    enabled: true
    server_url: "http://localhost:3000/mcp"
    description: "Custom business logic tools"
```

> 📖 **[Complete Guide: Building Custom MCP Tools](MCP_CUSTOM_TOOLS.md)** - Real-world examples, best practices, deployment, and more.

### Building Custom MCP Servers (Quick Overview)

Create your own MCP server in **Python** or **TypeScript**.

For a comprehensive guide with real-world examples, see **[MCP_CUSTOM_TOOLS.md](MCP_CUSTOM_TOOLS.md)**.

#### Python Example

**1. Install MCP SDK:**
```bash
pip install mcp
```

**2. Create server (`server.py`):**
```python
from mcp.server import Server, Tool
from mcp.types import TextContent

app = Server("my-custom-tools")

@app.tool()
async def calculate_discount(
    price: float,
    discount_percent: float
) -> str:
    """Calculate discounted price"""
    discounted = price * (1 - discount_percent / 100)
    return f"${discounted:.2f}"

@app.tool()
async def validate_email(email: str) -> str:
    """Check if email is valid"""
    import re
    pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    is_valid = re.match(pattern, email) is not None
    return f"Email {'valid' if is_valid else 'invalid'}"

if __name__ == "__main__":
    app.run(port=3000)
```

**3. Start server:**
```bash
python server.py
```

**4. Connect Hector:**
```yaml
tools:
  my_tools:
    type: "mcp"
    enabled: true
    server_url: "http://localhost:3000"
```

**5. Agent uses your tools:**
```
Agent: Let me calculate the discount
Tool Call: calculate_discount(price=100, discount_percent=20)
Result: $80.00
```

#### TypeScript Example

**1. Install MCP SDK:**
```bash
npm install @modelcontextprotocol/sdk
```

**2. Create server (`server.ts`):**
```typescript
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";

const server = new Server({
  name: "my-custom-tools",
  version: "1.0.0",
}, {
  capabilities: {
    tools: {},
  },
});

server.setRequestHandler("tools/list", async () => ({
  tools: [
    {
      name: "fetch_weather",
      description: "Get current weather for a city",
      inputSchema: {
        type: "object",
        properties: {
          city: { type: "string", description: "City name" },
        },
        required: ["city"],
      },
    },
  ],
}));

server.setRequestHandler("tools/call", async (request) => {
  const { name, arguments: args } = request.params;
  
  if (name === "fetch_weather") {
    // Your weather fetching logic
    return {
      content: [{
        type: "text",
        text: `Weather in ${args.city}: Sunny, 72°F`,
      }],
    };
  }
});

const transport = new StdioServerTransport();
server.connect(transport);
```

**3. Run server:**
```bash
node server.ts
```

### MCP Protocol Details

**Discovery (automatic):**
```json
// Request
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}

// Response
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "github_create_issue",
        "description": "Create a GitHub issue",
        "inputSchema": {
          "type": "object",
          "properties": {
            "repo": { "type": "string" },
            "title": { "type": "string" },
            "body": { "type": "string" }
          },
          "required": ["repo", "title"]
        }
      }
    ]
  }
}
```

**Execution:**
```json
// Request
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "github_create_issue",
    "arguments": {
      "repo": "owner/repo",
      "title": "Bug report",
      "body": "Description of the bug"
    }
  }
}

// Response
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Issue #123 created successfully"
      }
    ]
  }
}
```

**Features Hector handles:**
- ✅ **Auto-discovery** - Fetches tool list on startup
- ✅ **Schema parsing** - Converts JSON Schema to internal format
- ✅ **Error handling** - Retries with exponential backoff
- ✅ **SSE support** - Streaming responses
- ✅ **Timeout** - 30s default, configurable

---

## gRPC Plugins

Build **high-performance custom tools** in any language via gRPC.

### Why gRPC Plugins?

**Use cases:**
- **Custom LLMs** - Integrate proprietary models
- **Performance** - Binary protocol, faster than HTTP/JSON
- **Type safety** - Protocol Buffers with strong typing
- **Language flexibility** - Go, Python, Rust, Java, C++, etc.

**vs MCP:**
- MCP: Quick, HTTP-based, ecosystem of providers
- gRPC: Performance-critical, type-safe, custom requirements

### Plugin Types

Hector supports 3 plugin types:
1. **LLM Providers** - Custom language models
2. **Database Providers** - Custom vector databases
3. **Embedder Providers** - Custom embedding models

(Tool plugins are better via MCP for simplicity)

### Quick Example: Custom LLM Plugin

**1. Create plugin (`main.go`):**
```go
package main

import (
    "context"
    "github.com/kadirpekel/hector/plugins/grpc"
    pb "github.com/kadirpekel/hector/plugins/grpc/proto"
)

type MyLLMProvider struct {
    apiKey string
    model  string
}

func (p *MyLLMProvider) Initialize(ctx context.Context, config map[string]string) error {
    p.apiKey = config["api_key"]
    p.model = config["model"]
    // Initialize your LLM client
    return nil
}

func (p *MyLLMProvider) Generate(ctx context.Context, messages []*pb.Message, tools []*pb.ToolDefinition) (*pb.GenerateResponse, error) {
    // Your generation logic
    return &pb.GenerateResponse{
        Text:       "Generated response from custom LLM",
        ToolCalls:  nil,
        TokensUsed: 100,
    }, nil
}

func (p *MyLLMProvider) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []*pb.ToolDefinition) (<-chan *pb.StreamChunk, error) {
    ch := make(chan *pb.StreamChunk, 10)
    go func() {
        defer close(ch)
        ch <- &pb.StreamChunk{Type: pb.StreamChunk_TEXT, Text: "Streaming..."}
        ch <- &pb.StreamChunk{Type: pb.StreamChunk_DONE}
    }()
    return ch, nil
}

func (p *MyLLMProvider) GetModelInfo(ctx context.Context) (*pb.ModelInfo, error) {
    return &pb.ModelInfo{
        ModelName:   p.model,
        MaxTokens:   4096,
        Temperature: 0.7,
    }, nil
}

func (p *MyLLMProvider) Shutdown(ctx context.Context) error {
    return nil
}

func (p *MyLLMProvider) Health(ctx context.Context) error {
    return nil
}

func main() {
    grpc.ServeLLMPlugin(&MyLLMProvider{})
}
```

**2. Build:**
```bash
go mod init my-llm-plugin
go get github.com/kadirpekel/hector/plugins/grpc
go build -o my-llm-plugin
chmod +x my-llm-plugin
```

**3. Create manifest (`my-llm-plugin.plugin.yaml`):**
```yaml
name: "my-llm-plugin"
version: "1.0.0"
type: "llm_provider"
description: "Custom LLM provider"
author: "Your Name"
entry_point: "./my-llm-plugin"
```

**4. Configure Hector:**
```yaml
plugins:
  llm_providers:
    my_custom_llm:
      name: "my-llm-plugin"
      type: "grpc"
      path: "./plugins/my-llm-plugin"
      enabled: true
      config:
        api_key: "${CUSTOM_API_KEY}"
        model: "my-model-v1"

llms:
  custom:
    type: "plugin:my_custom_llm"
    model: "my-model-v1"

agents:
  my_agent:
    llm: "custom"  # Uses your plugin!
```

### Protocol Buffers

Plugin interfaces are defined in `.proto` files:

**LLM Provider Interface:**
```protobuf
service LLMProvider {
  rpc Initialize(InitializeRequest) returns (InitializeResponse);
  rpc Generate(GenerateRequest) returns (GenerateResponse);
  rpc GenerateStreaming(GenerateRequest) returns (stream StreamChunk);
  rpc GetModelInfo(Empty) returns (ModelInfo);
  rpc Health(HealthRequest) returns (HealthResponse);
  rpc Shutdown(ShutdownRequest) returns (ShutdownResponse);
}
```

See `plugins/grpc/proto/` for complete definitions.

### Plugin Examples

Check the `examples/plugins/` directory for working examples:
- `echo-llm/` - Simple LLM plugin that echoes input

---

## Configuration

Complete YAML syntax for all tool types.

### Global Tools Configuration

```yaml
tools:
  # Built-in tools
  execute_command:
    type: "command"
    enabled: true
    allowed_commands: ["ls", "cat", "grep"]
    working_directory: "./"
    max_execution_time: "30s"
    enable_sandboxing: true
  
  write_file:
    type: "write_file"
    enabled: true
    allowed_extensions: [".go", ".yaml", ".md"]
    forbidden_paths: ["/etc", "/usr"]
    max_file_size: 10485760  # 10MB
  
  search_replace:
    type: "search_replace"
    enabled: true
    max_replacements: 100
    backup_enabled: true
  
  search:
    type: "search"
    enabled: true
    document_stores: ["codebase_docs"]
    default_limit: 10
    max_limit: 50
    enabled_search_types: ["content", "file", "function"]
  
  todo:
    type: "todo"
    enabled: true
  
  # MCP tools
  composio:
    type: "mcp"
    enabled: true
    server_url: "https://api.composio.dev/mcp"
    description: "Composio integrations"
  
  mem0:
    type: "mcp"
    enabled: true
    server_url: "https://api.mem0.ai/mcp"
    description: "Agent memory"
  
  custom_tools:
    type: "mcp"
    enabled: true
    server_url: "http://localhost:3000/mcp"
    description: "Custom business tools"
```

### Agent-Level Tool Access

**All agents share the global tool registry.** You cannot restrict tools per agent in configuration, but you can guide tool usage via prompts:

```yaml
agents:
  safe_agent:
    prompt:
      tool_usage: |
        You may ONLY use these tools:
        - search (for looking up information)
        - todo (for task tracking)
        
        Never use execute_command or write_file.
```

**Why global?**
- Simplifies configuration
- Reduces duplication
- Centralized security policies
- Consistent tool behavior

### Environment Variables

Use environment variables for sensitive data:

```yaml
tools:
  composio:
    type: "mcp"
    enabled: true
    server_url: "${COMPOSIO_URL}"  # From env
    config:
      api_key: "${COMPOSIO_API_KEY}"  # From env
```

```bash
export COMPOSIO_URL="https://api.composio.dev/mcp"
export COMPOSIO_API_KEY="your-key-here"
```

---

## Security

Protecting your system with tools.

### Command Execution Security

**Whitelist-only:**
```yaml
tools:
  execute_command:
    allowed_commands:
      - "cat"    # Read files
      - "ls"     # List directory
      - "grep"   # Search text
      # Never: rm, sudo, curl (unless needed)
```

**Working directory restriction:**
```yaml
tools:
  execute_command:
    working_directory: "./workspace"  # Confined to this path
```

**Timeout protection:**
```yaml
tools:
  execute_command:
    max_execution_time: "30s"  # Kill after 30 seconds
```

**Sandboxing (advanced):**
```yaml
tools:
  execute_command:
    enable_sandboxing: true  # Platform-specific isolation
```

### File Operation Security

**Extension whitelist:**
```yaml
tools:
  write_file:
    allowed_extensions:
      - ".txt"
      - ".md"
      - ".json"
      # Exclude: .sh, .exe, system files
```

**Path blacklist:**
```yaml
tools:
  write_file:
    forbidden_paths:
      - "/etc"
      - "/usr"
      - "/bin"
      - "/var"
      - "${HOME}/.ssh"  # No SSH keys!
```

**Size limits:**
```yaml
tools:
  write_file:
    max_file_size: 10485760  # 10MB max
```

### MCP Server Security

**HTTPS only in production:**
```yaml
tools:
  production_mcp:
    server_url: "https://api.example.com/mcp"  # ✅ Secure
    # NOT: "http://api.example.com/mcp"  # ❌ Insecure
```

**Authentication:**
MCP servers should handle their own auth. Pass credentials via config:

```yaml
tools:
  composio:
    server_url: "https://api.composio.dev/mcp"
    config:
      api_key: "${COMPOSIO_API_KEY}"  # Server validates
```

**Rate limiting:**
Hector has built-in retry logic (3 attempts, exponential backoff). For additional control, implement rate limiting in your MCP server.

### Plugin Security

**Manifest validation:**
Hector validates plugin manifests before loading.

**Process isolation:**
Plugins run in separate processes via HashiCorp go-plugin.

**Resource limits:**
Set limits in your plugin code to prevent resource exhaustion.

---

## Best Practices

### Tool Selection

**Use built-in tools when possible:**
- Fast (no network overhead)
- Secure by default
- Well-tested

**Use MCP for integrations:**
- 150+ apps via providers
- Standard protocol
- Community support

**Use plugins for performance:**
- Custom LLMs with special requirements
- High-throughput scenarios
- Type-safe interfaces needed

### Prompt Engineering for Tools

**Be explicit:**
```yaml
prompt:
  tool_usage: |
    Use search for:
    - Documentation lookup
    - API reference
    - Code examples
    
    Use write_file for:
    - Creating new files
    - Updating configurations
    
    Use execute_command for:
    - Running tests
    - Building projects
```

**Provide examples:**
```yaml
prompt:
  tool_usage: |
    Example tool usage:
    
    To check files:
    execute_command(command="ls -la")
    
    To search docs:
    search(query="authentication", stores=["docs"])
```

**Set boundaries:**
```yaml
prompt:
  tool_usage: |
    NEVER use execute_command for:
    - Deleting files (rm)
    - System changes (sudo)
    - Network requests (curl, wget)
```

### Error Handling

**Tools return structured errors:**
```json
{
  "success": false,
  "error": "Command 'rm' not in allowed list",
  "tool_name": "execute_command",
  "execution_time": "0.001s"
}
```

**Agents handle errors naturally:**
```
Agent: Let me delete that file
Tool Call: execute_command(command="rm file.txt")
Result: Error - Command 'rm' not in allowed list
Agent: I apologize, but I'm not allowed to delete files for security reasons. 
      I can help you identify files to delete, but you'll need to remove them manually.
```

### Performance Optimization

**Minimize tool calls:**
```yaml
prompt:
  reasoning_instructions: |
    Before calling a tool:
    1. Is this tool call necessary?
    2. Can I get this info from previous results?
    3. Can I batch multiple operations?
```

**Use local tools for speed:**
- Built-in tools: < 1ms
- MCP tools: 10-100ms (network latency)
- Plugins: < 5ms (IPC overhead)

**Cache MCP discoveries:**
Hector caches tool discoveries. Restart only when MCP servers change.

---

## Troubleshooting

### Common Issues

**Tool not found:**
```
Error: tool 'my_tool' not found
```

**Solutions:**
1. Check tool is configured properly
2. Verify tool type is correct
3. Check MCP server URL is reachable
4. Restart Hector after config changes

**MCP discovery fails:**
```
Failed to discover tools from my_mcp: connection refused
```

**Solutions:**
1. Verify MCP server is running: `curl http://localhost:3000`
2. Check server URL in config
3. Ensure server implements `tools/list` endpoint
4. Check server logs for errors

**Command not allowed:**
```
Command 'wget' not in allowed list
```

**Solutions:**
1. Add command to whitelist:
   ```yaml
   allowed_commands: ["cat", "ls", "wget"]
   ```
2. Or guide agent to use different approach:
   ```yaml
   prompt:
     tool_usage: "Don't use wget. Use the built-in HTTP capabilities instead."
   ```

**File operation denied:**
```
Path '/etc/hosts' is forbidden
```

**Solutions:**
1. Use allowed paths only
2. Or remove restriction if safe:
   ```yaml
   forbidden_paths: []  # Use with caution!
   ```

### Debugging

**Enable debug logging:**
```bash
./hector serve --config config.yaml --debug
```

**Check tool registry:**
```bash
./hector list-tools
```

**Test MCP server manually:**
```bash
curl -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list",
    "params": {}
  }'
```

**Validate plugin:**
```bash
./hector validate-plugin ./plugins/my-plugin
```

---

## Examples

### Example 1: Complete Tool Setup

```yaml
# Full tool configuration example
tools:
  # Built-in tools
  execute_command:
    type: "command"
    enabled: true
    allowed_commands: ["cat", "ls", "grep", "git", "npm", "go"]
    max_execution_time: "30s"
  
  write_file:
    type: "write_file"
    enabled: true
    allowed_extensions: [".go", ".yaml", ".md", ".txt"]
    max_file_size: 10485760
  
  search:
    type: "search"
    enabled: true
    document_stores: ["codebase", "docs"]
  
  # MCP integrations
  composio:
    type: "mcp"
    enabled: true
    server_url: "https://api.composio.dev/mcp"
  
  custom_tools:
    type: "mcp"
    enabled: true
    server_url: "http://localhost:3000"

# Agent using all tools
agents:
  developer:
    name: "Developer Agent"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a software development assistant with access to:
        - File operations (read, write, search/replace)
        - Command execution (build, test, git)
        - Documentation search
        - External integrations (GitHub, Slack via Composio)
      
      tool_usage: |
        Use tools appropriately:
        - search: Look up documentation
        - write_file: Create/modify code
        - execute_command: Run builds and tests
        - composio tools: Interact with GitHub, Slack, etc.
```

### Example 2: Custom MCP Server

**Python weather MCP server:**

```python
# weather_server.py
from mcp.server import Server
import requests

app = Server("weather-tools")

@app.tool()
async def get_weather(city: str) -> str:
    """Get current weather for a city"""
    api_key = os.getenv("WEATHER_API_KEY")
    url = f"https://api.openweathermap.org/data/2.5/weather?q={city}&appid={api_key}"
    response = requests.get(url)
    data = response.json()
    
    temp = data["main"]["temp"] - 273.15  # Kelvin to Celsius
    description = data["weather"][0]["description"]
    
    return f"Weather in {city}: {description}, {temp:.1f}°C"

@app.tool()
async def get_forecast(city: str, days: int = 3) -> str:
    """Get weather forecast for a city"""
    # Implementation...
    pass

if __name__ == "__main__":
    app.run(port=3000)
```

**Hector configuration:**
```yaml
tools:
  weather:
    type: "mcp"
    enabled: true
    server_url: "http://localhost:3000"
    description: "Weather information tools"

agents:
  weather_assistant:
    llm: "gpt-4o"
    prompt:
      system_role: |
        You provide weather information using the weather tools.
```

---

## Summary

Hector's tool system provides three powerful extension mechanisms:

1. **Built-in Tools** - 5 production-ready tools for common tasks
2. **MCP Integration** - Connect to 150+ apps via standard protocol
3. **gRPC Plugins** - Build high-performance custom extensions

**Key Takeaways:**
- Tools are managed globally in a centralized registry
- MCP is the easiest way to add integrations (Composio, Mem0, custom)
- Security is built-in with whitelists, timeouts, and path restrictions
- All tools are automatically available to all agents
- Configuration is pure YAML - no code required

**Next Steps:**
- [Configuration Reference](CONFIGURATION.md) - All YAML syntax
- [Building Agents](AGENTS.md) - Use tools in your agents
- [Architecture](ARCHITECTURE.md) - System design
- [External Agents](EXTERNAL_AGENTS.md) - A2A protocol

**Start building with tools now!**
