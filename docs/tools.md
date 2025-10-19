---
title: Tools & Extensions
description: Built-in tools, MCP protocol, gRPC plugins, and custom integrations
---

# Tools & Extensions

Complete guide to Hector's extensible tool system

---

## Overview

Hector provides a **powerful, extensible tool system** that gives agents capabilities beyond language generation. Tools can be:

- **Built-in** - 5 ready-to-use tools for common tasks
- **MCP Servers** - Connect to 150+ integrations via Model Context Protocol
- **gRPC Plugins** - Build custom tools in any language

!!! info "Note"
    Individual tools use `enabled: true/false` for easy toggling. This is different from service-level configurations (A2A server, auth, memory) where presence of configuration implies it's enabled.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Global Tool Registry                        │
├─────────────────────────────────────────────────────────────┤
│  Tool Registry  →  Built-in Tools                            │
│  Tool Registry  →  MCP Servers                               │
│  Tool Registry  →  gRPC Plugins                              │
└─────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────┐
│                    Built-in Tools                             │
├─────────────────────────────────────────────────────────────┤
│  command  →  write_file  →  search  →  todo  →  search_replace │
└─────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────┐
│                     MCP Servers                              │
├─────────────────────────────────────────────────────────────┤
│  Composio Server  →  GitHub, Slack, Notion                  │
│  Mem0 Server      →  Vector Memory                          │
│  Custom Servers    →  Custom Integrations                   │
└─────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────┐
│                    gRPC Plugins                              │
├─────────────────────────────────────────────────────────────┤
│  Custom LLMs      →  Custom Language Models                 │
│  Custom Tools     →  Custom Capabilities                    │
└─────────────────────────────────────────────────────────────┘
```
**Key Concepts:**
- **Tool** - A single capability (e.g., "execute_command")
- **Tool Source** - A provider of tools (local, MCP server, plugin)
- **Tool Registry** - Centralized discovery and execution## Built-in Tools

Hector comes with 5 powerful built-in tools ready to use:

### 1. **execute_command** - Shell Command Execution

Run shell commands securely with whitelist-based security.

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
      - "python"
      - "node"
    max_execution_time: "30s"
    working_directory: "./"
```

**Security Features:**
- **Command Whitelist** - Only allowed commands can be executed
- **Execution Timeout** - Prevents hanging processes
- **Working Directory** - Restricts execution to specific directories
- **Sandboxing** - Isolated execution environment

**Agent Usage:**
```
User: What files are in this directory?
Agent: Let me check the directory contents...
Tool: execute_command(command="ls -la")
Agent: Here are the files in the directory: [lists files]
```

### 2. **write_file** - File Creation and Modification

Create and modify files with path restrictions.

```yaml
tools:
  write_file:
    type: "write_file"
    enabled: true
    allowed_paths:
      - "./src/"
      - "./docs/"
      - "./tests/"
    max_file_size: "10MB"
    create_directories: true
```

**Security Features:**
- **Path Restrictions** - Only allowed paths can be written to
- **Size Limits** - Prevents large file creation
- **Directory Creation** - Can create directories if needed
- **Backup Support** - Automatic backups before modification

**Agent Usage:**
```
User: Create a README file for this project
Agent: I'll create a comprehensive README file...
Tool: write_file(path="README.md", content="# Project Title\n\n## Description\n...")
Agent: Created README.md with project documentation
```

### 3. **search_replace** - Find and Replace Operations

Find and replace text in files with pattern matching.

```yaml
tools:
  search_replace:
    type: "search_replace"
    enabled: true
    allowed_paths:
      - "./src/"
      - "./docs/"
    max_replacements: 100
    backup: true
```

**Features:**
- **Pattern Matching** - Support for regex patterns
- **Replacement Limits** - Prevents excessive replacements
- **Automatic Backup** - Creates backups before changes
- **Preview Mode** - Show changes before applying

**Agent Usage:**
```
User: Rename the function getUserData to fetchUserData
Agent: I'll update the function name throughout the codebase...
Tool: search_replace(file="api.js", old="getUserData", new="fetchUserData")
Agent: Renamed getUserData to fetchUserData in api.js
```

### 4. **todo** - Task Management and Tracking

Manage tasks and track progress within conversations.

```yaml
tools:
  todo:
    type: "todo"
    enabled: true
    max_tasks: 50
    auto_cleanup: true
```

**Features:**
- **Task Status** - Track pending, in_progress, completed tasks
- **Task Limits** - Prevent excessive task creation
- **Auto Cleanup** - Remove completed tasks automatically
- **Task History** - Keep track of task changes

**Agent Usage:**
```
Agent: Breaking this down into manageable tasks...
Tool: todo_write(tasks=[
  {id: "1", content: "Research API options", status: "in_progress"},
  {id: "2", content: "Compare solutions", status: "pending"},
  {id: "3", content: "Write recommendation", status: "pending"}
])
Agent: I've created a task list to track our progress
```

### 5. **search** - Semantic Search Across Document Stores

Search across document stores for RAG (Retrieval-Augmented Generation).

```yaml
# Automatic if document_stores configured
document_stores:
  codebase_docs:
    type: "qdrant"
    collection: "my_codebase"
    database: "qdrant"
    embedder: "ollama"
```

**Features:**
- **Semantic Search** - Vector-based similarity search
- **Multi-Store** - Search across multiple document stores
- **Relevance Scoring** - Rank results by relevance
- **Source Citation** - Provide source references

**Agent Usage:**
```
User: How do I use the authentication module?
Agent: Let me search the documentation for authentication usage...
Tool: search(query="authentication module usage", stores=["codebase_docs"])
Agent: Based on the documentation, here's how to use the authentication module: [explains with citations]
```

---

## MCP (Model Context Protocol) Integration

Connect to 150+ external services via MCP servers.

### MCP Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Global Tool Registry                        │
├─────────────────────────────────────────────────────────────┤
│  Tool Registry  →  Built-in Tools                            │
│  Tool Registry  →  MCP Servers                               │
│  Tool Registry  →  gRPC Plugins                              │
└─────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────┐
│                    Built-in Tools                             │
├─────────────────────────────────────────────────────────────┤
│  command  →  write_file  →  search  →  todo  →  search_replace │
└─────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────┐
│                     MCP Servers                              │
├─────────────────────────────────────────────────────────────┤
│  Composio Server  →  GitHub, Slack, Notion                  │
│  Mem0 Server      →  Vector Memory                          │
│  Custom Servers    →  Custom Integrations                   │
└─────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────┐
│                    gRPC Plugins                              │
├─────────────────────────────────────────────────────────────┤
│  Custom LLMs      →  Custom Language Models                 │
│  Custom Tools     →  Custom Capabilities                    │
└─────────────────────────────────────────────────────────────┘
```
### Popular MCP Servers

=== "Composio"
    ```yaml
    tools:
      composio:
        type: "mcp"
        enabled: true
        server:
          command: "composio-mcp"
          args: ["--api-key", "${COMPOSIO_API_KEY}"]
        tools:
          - "github_create_issue"
          - "slack_send_message"
          - "notion_create_page"
    ```
    
    **Available Tools:**
    - **GitHub** - Create issues, PRs, manage repos
    - **Slack** - Send messages, manage channels
    - **Notion** - Create pages, databases
    - **Google** - Gmail, Calendar, Drive
    - **Microsoft** - Outlook, Teams, SharePoint

=== "Mem0"
    ```yaml
    tools:
      mem0:
        type: "mcp"
        enabled: true
        server:
          command: "mem0-mcp"
          args: ["--api-key", "${MEM0_API_KEY}"]
        tools:
          - "mem0_store"
          - "mem0_search"
          - "mem0_delete"
    ```
    
    **Features:**
    - **Vector Memory** - Store and retrieve memories
    - **Semantic Search** - Find relevant memories
    - **Memory Management** - Update and delete memories

=== "Custom MCP Server"
    ```yaml
    tools:
      custom_mcp:
        type: "mcp"
        enabled: true
        server:
          command: "python"
          args: ["custom_mcp_server.py"]
        tools:
          - "custom_tool_1"
          - "custom_tool_2"
    ```

### MCP Configuration

```yaml
tools:
  <tool-id>:
    type: "mcp"
    enabled: true
    
    # Server configuration
    server:
      command: string                  # MCP server command
      args: []string                   # Command arguments
      env: {}object                    # Environment variables
      
    # Tool selection
    tools: []string                     # Specific tools to enable (empty = all)
    
    # Connection settings
    timeout: string                     # Connection timeout (default: "30s")
    retries: int                        # Retry attempts (default: 3)
```

---

## gRPC Plugin System

Build custom tools in any language that supports gRPC.

### Plugin Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Global Tool Registry                        │
├─────────────────────────────────────────────────────────────┤
│  Tool Registry  →  Built-in Tools                            │
│  Tool Registry  →  MCP Servers                               │
│  Tool Registry  →  gRPC Plugins                              │
└─────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────┐
│                    Built-in Tools                             │
├─────────────────────────────────────────────────────────────┤
│  command  →  write_file  →  search  →  todo  →  search_replace │
└─────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────┐
│                     MCP Servers                              │
├─────────────────────────────────────────────────────────────┤
│  Composio Server  →  GitHub, Slack, Notion                  │
│  Mem0 Server      →  Vector Memory                          │
│  Custom Servers    →  Custom Integrations                   │
└─────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────┐
│                    gRPC Plugins                              │
├─────────────────────────────────────────────────────────────┤
│  Custom LLMs      →  Custom Language Models                 │
│  Custom Tools     →  Custom Capabilities                    │
└─────────────────────────────────────────────────────────────┘
```
### Plugin Types

| Type | Purpose | Interface | Example |
|------|---------|-----------|---------|
| **LLM Provider** | Custom language models | Text generation, streaming | Custom API, local models |
| **Tool Provider** | Custom capabilities | Function execution | File operations, API calls |
| **Database Provider** | Vector databases | Embeddings, search | Custom vector DB |

### Tool Plugin Example

**Configuration:**
```yaml
tools:
  custom_tool:
    type: "plugin:my_custom_tool"
    enabled: true
    config:
      endpoint: "http://localhost:8081"
      api_key: "${CUSTOM_TOOL_API_KEY}"
      timeout: "30s"
```

**Plugin Implementation (Go):**
```go
package main

import (
    "context"
    "log"
    
    "google.golang.org/grpc"
    pb "github.com/kadirpekel/hector/pkg/plugins/grpc/pb"
)

type CustomToolServer struct {
    pb.UnimplementedToolProviderServer
}

func (s *CustomToolServer) ExecuteTool(ctx context.Context, req *pb.ExecuteToolRequest) (*pb.ExecuteToolResponse, error) {
    // Custom tool logic
    result := "Custom tool executed successfully"
    
    return &pb.ExecuteToolResponse{
        Success: true,
        Result:  result,
    }, nil
}

func main() {
    lis, err := net.Listen("tcp", ":8081")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    
    s := grpc.NewServer()
    pb.RegisterToolProviderServer(s, &CustomToolServer{})
    
    log.Println("Custom tool server starting on :8081")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

### Plugin Configuration

```yaml
plugins:
  tool_providers:
    my_custom_tool:
      type: "grpc"
      path: "./plugins/my-tool-plugin"
      enabled: true
      config:
        # Plugin-specific configuration
        endpoint: "http://localhost:8081"
        api_key: "${CUSTOM_API_KEY}"
        timeout: "30s"
      
      # Plugin lifecycle
      auto_start: true
      restart_on_failure: true
      max_restarts: 5
```

---

## Tool Usage Patterns

### Agent Tool Selection

Agents automatically select appropriate tools based on:

- **Task Analysis** - Understanding what needs to be done
- **Tool Capabilities** - Matching tools to requirements
- **Security Context** - Respecting tool permissions
- **Previous Experience** - Learning from past tool usage

### Tool Chaining

Agents can chain multiple tools together:

```
User: Create a new feature branch and implement user authentication
Agent: I'll break this down into steps...

1. Create feature branch
Tool: execute_command(command="git checkout -b feature/user-auth")

2. Create authentication module
Tool: write_file(path="src/auth.js", content="...")

3. Update documentation
Tool: write_file(path="docs/auth.md", content="...")

4. Run tests
Tool: execute_command(command="npm test")

5. Track progress
Tool: todo_write(tasks=[
  {id: "1", content: "Create feature branch", status: "completed"},
  {id: "2", content: "Implement auth module", status: "completed"},
  {id: "3", content: "Update docs", status: "completed"},
  {id: "4", content: "Run tests", status: "completed"}
])
```

### Error Handling

Tools provide comprehensive error handling:

```yaml
# Tool error handling
tools:
  execute_command:
    type: "command"
    enabled: true
    error_handling:
      retry_on_failure: true
      max_retries: 3
      retry_delay: "1s"
      fallback_action: "log_error"
```

---

## Security Considerations

### Tool Security Model

Hector implements a comprehensive security model for tools:

- **Permission-Based** - Tools respect agent permissions
- **Path Restrictions** - File operations restricted to allowed paths
- **Execution Limits** - Timeouts prevent hanging processes
- **Resource Limits** - Size and count limits prevent abuse
- **Sandboxing** - Isolated execution environments

### Security Configuration

```yaml
# Security settings
security:
  tools:
    # Global tool security
    max_execution_time: "30s"
    max_file_size: "10MB"
    max_memory_usage: "1GB"
    
    # Path restrictions
    allowed_paths:
      - "./src/"
      - "./docs/"
      - "./tests/"
    
    # Command restrictions
    allowed_commands:
      - "cat"
      - "ls"
      - "grep"
      - "git"
      - "npm"
      - "go"
    
    # Network restrictions
    allowed_hosts:
      - "api.github.com"
      - "api.openai.com"
      - "localhost"
```

---

## Tool Performance

### Performance Optimization

- **Parallel Execution** - Multiple tools can run simultaneously
- **Result Caching** - Cache tool results for repeated operations
- **Timeout Management** - Prevent tools from hanging
- **Performance Monitoring** - Track tool execution metrics

### Performance Configuration

```yaml
# Performance settings
performance:
  tools:
    # Execution settings
    max_concurrent: 10
    timeout: "30s"
    
    # Caching
    cache:
      enabled: true
      ttl: "1h"
      max_size: "100MB"
    
    # Monitoring
    metrics:
      enabled: true
      collect_detailed: true
```

---

## Tool Development

### Building Custom Tools

=== "gRPC Plugin"
    ```bash
    # Create plugin project
    mkdir my-tool-plugin
    cd my-tool-plugin
    
    # Initialize Go module
    go mod init my-tool-plugin
    
    # Add Hector protobuf dependency
    go get github.com/kadirpekel/hector/pkg/plugins/grpc/pb
    ```

=== "MCP Server"
    ```python
    # Create MCP server
    from mcp.server import Server
    from mcp.types import Tool
    
    server = Server("my-tool-server")
    
    @server.tool("my_custom_tool")
    async def my_custom_tool(param: str) -> str:
        # Custom tool logic
        return f"Processed: {param}"
    
    if __name__ == "__main__":
        server.run()
    ```

### Tool Testing

```yaml
# Test configuration
test:
  tools:
    execute_command:
      test_commands:
        - "echo 'Hello World'"
        - "ls -la"
      expected_results:
        - "Hello World"
        - "file listing"
```

---

## Tool Examples

### Complete Tool Configuration

```yaml
# Complete tool configuration example
tools:
  # Built-in tools
  execute_command:
    type: "command"
    enabled: true
    allowed_commands: ["cat", "ls", "grep", "git", "npm", "go"]
    max_execution_time: "30s"
    working_directory: "./"
  
  write_file:
    type: "write_file"
    enabled: true
    allowed_paths: ["./src/", "./docs/", "./tests/"]
    max_file_size: "10MB"
    create_directories: true
  
  search_replace:
    type: "search_replace"
    enabled: true
    allowed_paths: ["./src/", "./docs/"]
    max_replacements: 100
    backup: true
  
  todo:
    type: "todo"
    enabled: true
    max_tasks: 50
    auto_cleanup: true
  
  # MCP tools
  composio:
    type: "mcp"
    enabled: true
    server:
      command: "composio-mcp"
      args: ["--api-key", "${COMPOSIO_API_KEY}"]
    tools: ["github_create_issue", "slack_send_message"]
  
  mem0:
    type: "mcp"
    enabled: true
    server:
      command: "mem0-mcp"
      args: ["--api-key", "${MEM0_API_KEY}"]
    tools: ["mem0_store", "mem0_search"]
  
  # Custom gRPC plugins
  custom_tool:
    type: "plugin:my_custom_tool"
    enabled: true
    config:
      endpoint: "http://localhost:8081"
      api_key: "${CUSTOM_API_KEY}"
      timeout: "30s"
```

---

