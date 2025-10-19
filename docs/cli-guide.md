---
title: CLI Guide
description: Complete command-line interface reference for Hector AI Agent Platform
---

# Hector CLI Guide

Complete command-line interface reference for the Hector AI Agent Platform.

---

## Overview

The Hector CLI provides a unified interface for interacting with AI agents in three flexible ways. Whether you need quick local experimentation, persistent server deployment, or remote agent access, the CLI adapts to your workflow with a consistent command structure.

**Key Features:**
- **Zero-config mode** - Start immediately with just an API key
- **A2A protocol native** - Connect to any A2A-compliant server
- **Flexible deployment** - Local, server, or remote
- **Built-in validation** - Fail-fast with clear error messages

---

## Understanding the Three Modes

The CLI automatically selects the appropriate mode based on your command and flags:

### Server Mode
**Trigger:** `hector serve`  
**Purpose:** Host AI agents as a persistent A2A protocol server

Run agents as a service that multiple clients can connect to. Perfect for:
- Production deployments with multiple users
- Team collaboration with shared agent infrastructure
- Long-running services with persistent state
- Exposing agents via REST, gRPC, or JSON-RPC

### Client Mode
**Trigger:** `hector <command> --server <URL>`  
**Purpose:** Connect to remote A2A servers (Hector or 3rd party)

Act as a client connecting to an existing A2A server. Use this to:
- Connect to production Hector servers
- Access 3rd party A2A-compliant agent services
- Use shared team agents without local setup
- Call remote agents from scripts and automation

!!! info "Note"
    You only control client-side options (`--server`, `--token`, `--stream`). The server determines agent configuration.

### Local Mode
**Trigger:** `hector <command>` (without `--server`)  
**Purpose:** Run agents in-process for immediate local execution

Execute agents directly in the CLI process without a server. Ideal for:
- Quick experimentation and testing
- Local development and debugging
- CI/CD pipelines and automation scripts
- One-off tasks and ad-hoc queries

**Supports both:** Configuration files (`--config`) and zero-config flags (`--model`, `--tools`, etc.)

---

## Mode Selection Matrix

| Mode | Trigger | Config Support | Zero-Config Support | Primary Use Case |
|------|---------|---------------|---------------------|------------------|
| **Server** | `serve` command | Yes | Yes | Host agents for multiple clients |
| **Client** | `--server` flag | No | No | Connect to remote A2A servers |
| **Local** | No `--server` flag | Yes | Yes | In-process local execution |

### 1. Server Mode

Start a persistent A2A server to host agents.

**Use cases:** Production deployments, multi-agent systems, multiple concurrent clients

**With configuration file:**
```bash
hector serve --config hector.yaml
```

**With zero-config:**
```bash
export OPENAI_API_KEY="sk-..."

# Start with defaults
hector serve

# With custom options
hector serve --model gpt-4o --tools --docs ./knowledge
```

**Server mode flags:**
- `--config FILE` - Configuration file (default: `hector.yaml`)
- `--port PORT` - Server port (default: 8080, overrides config)
- `--host HOST` - Server host (overrides config)
- `--a2a-base-url URL` - A2A base URL for discovery
- `--debug` - Enable debug output

**Zero-config flags (when hector.yaml doesn't exist):**
- `--api-key KEY` - API key (or use env var)
- `--model NAME` - Model name (default: `gpt-4o-mini`)
- `--tools` - Enable all local tools
- `--mcp-url URL` - MCP server URL (supports auth: `https://user:pass@host`)
- `--docs FOLDER` - Document folder for RAG
- `--embedder-model MODEL` - Embedder model (default: `nomic-embed-text`)
- `--vectordb URL` - Vector database URL (default: `http://localhost:6333`)

---

### 2. Client Mode

Connect to a **remote Hector server** as a client. No local config or zero-config options supported.

**Use cases:** Connect to remote servers, team collaboration, shared agent infrastructure

**Activated by:** Using the `--server` flag

```bash
# List agents on remote server
hector list --server http://remote-server:8080

# Get agent info
hector info assistant --server http://remote-server:8080

# Call agent
hector call assistant "hello" --server http://remote-server:8080

# Interactive chat
hector chat assistant --server http://remote-server:8080
```

**Client mode flags:**
- `--server URL` - Remote server URL (required for client mode)
- `--token TOKEN` - Authentication token
- `--stream BOOL` - Enable streaming (call only, default: true)

!!! warning "Important"
    In client mode, `--config`, `--model`, `--tools`, and other zero-config flags will **FAIL immediately** with a clear error. The remote server has its own configuration—you cannot override it from the client.

**Example error:**
```bash
$ hector call --server URL --config my.yaml agent "task"
Error: --config flag is not supported in Client (Remote) mode

You're connecting to a remote server which has its own configuration.

Solutions:
  - Remove --config flag to use the remote server's configuration
  - Remove --server flag to use Local mode with local config
```

!!! info "Note"
    Unlike Local mode, Client mode does NOT support environment variables for server/token. You must use explicit flags (`--server`, `--token`) for clarity and to avoid ambiguity.

---

### 3. Local Mode

Run agents **in-process** without a server. Supports both config files AND zero-config options.

**Use cases:** Quick tasks, experimentation, local development, CI/CD scripts, command-line automation

**Activated by:** Using commands (`list`, `info`, `call`, `chat`) **WITHOUT** the `--server` flag

**With configuration file:**
```bash
hector call assistant "hello" --config hector.yaml
hector chat assistant --config hector.yaml
hector list --config hector.yaml
```

**With zero-config (fastest!):**
```bash
export OPENAI_API_KEY="sk-..."

# Start using immediately (no config file needed!)
hector call "Explain quantum computing"        # Agent name optional!
hector call assistant "Explain quantum computing"  # Explicit agent name still works

# Interactive chat
hector chat                                   # Agent name optional!
hector chat assistant                         # Explicit agent name still works

# Enable tools
hector call "List files" --tools              # Agent name optional!
hector call assistant "List files" --tools    # Explicit agent name still works

# Custom model
hector call "Write code" --model gpt-4o       # Agent name optional!
hector call assistant "Write code" --model gpt-4o  # Explicit agent name still works
```

**Local mode flags:**
- `--config FILE` - Configuration file (default: `hector.yaml`)

**Zero-config flags (for call and chat commands):**
- `--api-key KEY` - API key (or use env var)
- `--base-url URL` - API base URL (default: `https://api.openai.com/v1`)
- `--model NAME` - Model name (default: `gpt-4o-mini`)
- `--tools` - Enable local tools (file ops, commands)
- `--mcp-url URL` - MCP server URL (supports auth: `https://user:pass@host`)

!!! info "Configuration Reference"
    For all configuration options, see the Configuration Reference

### Agent Name Behavior

**In Zero-Config Mode (Local mode without config file):**
- Agent name is **optional** for `call` and `chat` commands
- Defaults to `"assistant"` when not specified
- Both `hector call "prompt"` and `hector call assistant "prompt"` work

**In Config File Mode (Local mode with config file):**
- Agent name is **required** for `call` and `chat` commands
- Must specify which agent from your config file to use
- `hector call "prompt"` will fail, `hector call myagent "prompt"` is required

**In Client Mode (with --server flag):**
- Agent name is **required** for `call` and `chat` commands
- Must specify which agent on the remote server to use
- `hector call "prompt" --server URL` will fail, `hector call myagent "prompt" --server URL` is required

**In Server Mode:**
- Agent names are defined in your configuration file
- No local interaction with agents via CLI (agents are hosted for clients)

---

## Mode Validation & Error Handling

Hector performs **strict validation** on flag combinations and **fails immediately** with clear error messages when you use incompatible flags.

### Automatic Mode Detection

The CLI automatically detects which mode you're in based on:
1. If `serve` command → **Server mode**
2. If `--server` flag present → **Client mode**
3. Otherwise → **Local mode**

**No environment variables affect mode detection.** This keeps behavior explicit and predictable.

### Explicit Mode Selection

Mode selection is **always explicit** - determined only by the command and flags you use:
- `hector serve` → Server mode
- `hector <cmd> --server URL` → Client mode  
- `hector <cmd>` → Local mode

**No hidden environment variables affect mode selection.** What you type is what you get.

### Validation Examples

**Client mode with invalid flags:**
```bash
# Fails immediately
$ hector call --server http://remote:8080 --config my.yaml agent "task"
Error: --config flag is not supported in Client (Remote) mode

# Fails immediately
$ hector call --server http://remote:8080 --model gpt-4o agent "task"
Error: --model flag is not supported in Client (Remote) mode

# Fails immediately
$ hector call --server http://remote:8080 --tools agent "task"
Error: --tools flag is not supported in Client (Remote) mode

# Fails immediately
$ hector call --server http://remote:8080 --mcp-url URL agent "task"
Error: --mcp-url flag is not supported in Client (Remote) mode
```

**All errors include:**
- Clear explanation of WHY the flag doesn't work
- Solutions to fix the issue
- Current mode and server information

---

## Commands Reference

### `serve` - Start A2A Server

Start a persistent server to host agents.

```bash
# With config
hector serve --config hector.yaml

# Zero-config
hector serve --api-key $OPENAI_API_KEY --tools

# Custom port and host
hector serve --config hector.yaml --port 9090 --host 0.0.0.0

# Debug mode
hector serve --config hector.yaml --debug
```

**Mode:** Server mode only

---

### `list` - List Available Agents

List all available agents.

```bash
# Local mode (from config)
hector list --config hector.yaml

# Local mode (zero-config, shows "assistant")
hector list

# Client mode (from remote server)
hector list --server http://localhost:8080 --token abc123
```

**Modes:** Local mode, Client mode

---

### `info <agent>` - Get Agent Information

Display detailed information about a specific agent.

```bash
# Local mode
hector info assistant --config hector.yaml

# Client mode
hector info assistant --server http://localhost:8080
```

**Modes:** Local mode, Client mode

---

### `call <agent> "<prompt>"` - Execute Task

Execute a one-shot task on an agent.

```bash
# Local mode with config
hector call assistant "Explain AI" --config hector.yaml

# Local mode with zero-config
hector call assistant "Explain AI"

# Client mode
hector call assistant "Explain AI" --server http://localhost:8080

# With custom options (Local mode, zero-config)
hector call assistant "Write code" --model gpt-4o --tools
hector call assistant "Use GitHub" --mcp-url https://api.composio.dev/v1/mcp
hector call assistant "Search docs" --docs ./knowledge

# With embedded auth in MCP URL
hector call assistant "Deploy app" --mcp-url https://user:token@composio.dev/mcp

# Disable streaming
hector call assistant "hello" --stream=false
```

**Modes:** Local mode, Client mode

---

### `chat <agent>` - Interactive Chat

Start an interactive chat session with an agent.

```bash
# Local mode with config
hector chat assistant --config hector.yaml

# Local mode with zero-config
hector chat assistant

# Client mode
hector chat assistant --server http://localhost:8080

# With custom options (Local mode, zero-config)
hector chat assistant --model gpt-4o
hector chat assistant --mcp-url https://api.composio.dev/v1/mcp --tools
```

**Interactive commands:**
- `/quit` or `/exit` - Exit chat
- `/clear` - Clear conversation history
- `/info` - Show agent information

**Modes:** Local mode, Client mode

---

### `task` - Manage Tasks

Interact with tasks when using the SQL task backend for persistent task storage.

**Available actions:**
- `get` - Retrieve task details and history
- `cancel` - Cancel a running task

```bash
# Get task details (flags before positional args)
hector task --server http://localhost:8081 get assistant task-abc123...

# In local mode (uses local config)
hector task --config configs/task-sql-example.yaml get assistant task-abc123...

# Cancel a task
hector task --server http://localhost:8081 cancel assistant task-abc123...
```

**Output example:**
```
Task Details
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Task ID:     task-abc123...
Context ID:  ctx-xyz789...
Status:      COMPLETED
Updated:     2025-10-14 21:31:09

History (2 messages):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1. [User] Write a haiku about coding
2. [Agent] Code flows like water
          Through silicon valleys deep
          Logic finds its way
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Task states:**
- **SUBMITTED** - Task has been created
- **WORKING** - Task is being processed
- **COMPLETED** - Task completed successfully
- **FAILED** - Task failed
- **CANCELLED** - Task was cancelled

**Prerequisites:**
- Server must be running with task service enabled
- See `configs/task-sql-example.yaml` for configuration example

**Modes:** Local mode, Client mode

!!! warning "Important"
    Flags must come before positional arguments:
```bash
# Correct
hector task --server URL get agent task-id

# Wrong
hector task get agent task-id --server URL
```

---

## Environment Variables

### API Keys (for Zero-Config Mode)

Set the appropriate API key for your LLM provider:

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic (Claude)
export ANTHROPIC_API_KEY="sk-ant-..."

# Google Gemini
export GEMINI_API_KEY="AI..."
```

Hector will automatically detect which provider's key is set and use it for zero-config mode.

### MCP Server URL (for Zero-Config Mode)

Set your MCP server URL to automatically configure tool integration:

```bash
# Composio MCP server with auth
export MCP_URL="https://api-key-here@api.composio.dev/v1/mcp"

# Local MCP server
export MCP_URL="http://localhost:3000/mcp"

# With basic auth
export MCP_URL="https://user:password@mcp.example.com"
```

The `--mcp-url` flag overrides this environment variable if both are provided.

### Using .env Files

Hector automatically loads `.env` files:

```bash
# Create .env file
cat > .env << EOF
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
GEMINI_API_KEY=AI...
MCP_URL=https://api-key@api.composio.dev/v1/mcp
EOF

# Now zero-config mode works without flags
hector call assistant "hello"
hector serve
```

!!! info "Note"
    Server URL and token for Client mode are NOT supported via environment variables. Always use explicit `--server` and `--token` flags for Client mode.

---

## CLI Flag Order

!!! warning "Important"
    Flags must come **before** positional arguments (agent name, prompt):

```bash
# Correct
hector call --server http://localhost:8080 assistant "hello"
hector chat --model gpt-4o assistant

# Wrong - flags after agent name won't be parsed
hector call assistant "hello" --server http://localhost:8080
hector chat assistant --model gpt-4o
```

---

## Mode Selection Decision Tree

```
┌─────────────────────────────────────┐
│  What are you trying to do?        │
└──────────┬──────────────────────────┘
           │
           ├─ Host agents for multiple clients?
           │  → Use SERVER MODE: `hector serve`
           │
           ├─ Connect to existing Hector server?
           │  → Use CLIENT MODE: Add `--server URL` to commands
           │
           └─ Quick local task or script?
              → Use LOCAL MODE: Use commands without `--server`
```
