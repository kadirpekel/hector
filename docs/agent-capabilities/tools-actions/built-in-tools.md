---
layout: default
title: Built-in Tools
nav_order: 1
parent: Tools & Actions
description: "5 ready-to-use tools for common agent tasks"
---

# Built-in Tools

Hector includes 5 production-ready tools for common agent tasks.

## 1. execute_command

Execute shell commands securely with whitelisting.

**Global Tool Configuration:**
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

  search_replace:
    type: "search_replace"
    enabled: true
    max_replacements: 100
    backup_enabled: true

  todo:
    type: "todo"
    enabled: true

agents:
  file_manager:
    name: "File Manager"
    llm: "gpt-4o"
    tools: ["execute_command", "write_file", "search", "search_replace", "todo"]
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

## 2. write_file

Create and modify files safely.

**Configuration:**
```yaml
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

## 3. search

Semantic search across document stores (RAG).

**Configuration:**
```yaml
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

## 4. search_replace

Find and replace text in files.

**Configuration:**
```yaml
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

## 5. todo_write

Task management and tracking.

**Configuration:**
```yaml
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

## Quick Start

Enable all built-in tools:
```yaml
tools:
  execute_command:
    type: "command"
    enabled: true
    allowed_commands: ["cat", "ls", "grep", "git"]

  write_file:
    type: "write_file"
    enabled: true
    allowed_extensions: [".txt", ".md", ".yaml"]

  search:
    type: "search"
    enabled: true
    document_stores: ["knowledge_base"]

  search_replace:
    type: "search_replace"
    enabled: true

  todo:
    type: "todo"
    enabled: true

agents:
  assistant:
    name: "Assistant"
    llm: "gpt-4o"
    tools: ["execute_command", "write_file", "search", "search_replace", "todo"]
```

## Security Best Practices

1. **Whitelist Commands**: Only allow necessary commands
2. **Restrict File Access**: Use allowed_extensions and forbidden_paths
3. **Set Timeouts**: Prevent runaway processes
4. **Enable Sandboxing**: Isolate tool execution when possible
5. **Monitor Usage**: Log tool calls for security auditing

## See Also

- **[MCP Integration](mcp-integration)** - Connect to external tools
- **[Custom Tools](custom-tools)** - Build custom capabilities
- **[Security & Sandboxing](../production-security/security-sandboxing)** - Production security
