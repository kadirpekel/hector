---
layout: default
title: CLI Guide
nav_order: 3
parent: Getting Started
description: "Complete command-line interface reference for Hector"
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

### 🔧 Server Mode
**Trigger:** `hector serve`  
**Purpose:** Host AI agents as a persistent A2A protocol server

Run agents as a service that multiple clients can connect to. Perfect for:
- Production deployments with multiple users
- Team collaboration with shared agent infrastructure
- Long-running services with persistent state
- Exposing agents via REST, gRPC, or JSON-RPC

### 🌐 Client Mode
**Trigger:** `hector <command> --server <URL>`  
**Purpose:** Connect to remote A2A servers (Hector or 3rd party)

Act as a client connecting to an existing A2A server. Use this to:
- Connect to production Hector servers
- Access 3rd party A2A-compliant agent services
- Use shared team agents without local setup
- Call remote agents from scripts and automation

**Note:** You only control client-side options (`--server`, `--token`, `--stream`). The server determines agent configuration.

### 💻 Direct Mode
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
| **Server** | `serve` command | ✅ Yes | ✅ Yes | Host agents for multiple clients |
| **Client** | `--server` flag | ❌ No | ❌ No | Connect to remote A2A servers |
| **Direct** | No `--server` flag | ✅ Yes | ✅ Yes | In-process local execution |

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

**⚠️ Important:** In client mode, `--config`, `--model`, `--tools`, and other zero-config flags will **FAIL immediately** with a clear error. The remote server has its own configuration—you cannot override it from the client.

**Example error:**
```bash
$ hector call --server URL --config my.yaml agent "task"
❌ Error: --config flag is not supported in Client (Remote) mode

You're connecting to a remote server which has its own configuration.

Solutions:
  • Remove --config flag to use the remote server's configuration
  • Remove --server flag to use Direct mode with local config
```

**Note:** Unlike Direct mode, Client mode does NOT support environment variables for server/token. You must use explicit flags (`--server`, `--token`) for clarity and to avoid ambiguity.

---

### 3. Direct Mode

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
hector call assistant "Explain quantum computing"

# Interactive chat
hector chat assistant

# Enable tools
hector call assistant "List files" --tools

# Custom model
hector call assistant "Write code" --model gpt-4o
```

**Direct mode flags:**
- `--config FILE` - Configuration file (default: `hector.yaml`)

**Zero-config flags (for call and chat commands):**
- `--api-key KEY` - API key (or use env var)
- `--base-url URL` - API base URL (default: `https://api.openai.com/v1`)
- `--model NAME` - Model name (default: `gpt-4o-mini`)
- `--tools` - Enable local tools (file ops, commands)
- `--mcp-url URL` - MCP server URL (supports auth: `https://user:pass@host`)

**📖 For all configuration options, see [Configuration Reference](https://gohector.dev/CONFIGURATION.html)**

---

## Mode Validation & Error Handling

Hector performs **strict validation** on flag combinations and **fails immediately** with clear error messages when you use incompatible flags.

### Automatic Mode Detection

The CLI automatically detects which mode you're in based on:
1. If `serve` command → **Server mode**
2. If `--server` flag present → **Client mode**
3. Otherwise → **Direct mode**

**No environment variables affect mode detection.** This keeps behavior explicit and predictable.

### Explicit Mode Selection

Mode selection is **always explicit** - determined only by the command and flags you use:
- `hector serve` → Server mode
- `hector <cmd> --server URL` → Client mode  
- `hector <cmd>` → Direct mode

**No hidden environment variables affect mode selection.** What you type is what you get.

### Validation Examples

**Client mode with invalid flags:**
```bash
# ❌ Fails immediately
$ hector call --server http://remote:8080 --config my.yaml agent "task"
Error: --config flag is not supported in Client (Remote) mode

# ❌ Fails immediately
$ hector call --server http://remote:8080 --model gpt-4o agent "task"
Error: --model flag is not supported in Client (Remote) mode

# ❌ Fails immediately
$ hector call --server http://remote:8080 --tools agent "task"
Error: --tools flag is not supported in Client (Remote) mode

# ❌ Fails immediately
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
# Direct mode (from config)
hector list --config hector.yaml

# Direct mode (zero-config, shows "assistant")
hector list

# Client mode (from remote server)
hector list --server http://localhost:8080 --token abc123
```

**Modes:** Direct mode, Client mode

---

### `info <agent>` - Get Agent Information

Display detailed information about a specific agent.

```bash
# Direct mode
hector info assistant --config hector.yaml

# Client mode
hector info assistant --server http://localhost:8080
```

**Modes:** Direct mode, Client mode

---

### `call <agent> "<prompt>"` - Execute Task

Execute a one-shot task on an agent.

```bash
# Direct mode with config
hector call assistant "Explain AI" --config hector.yaml

# Direct mode with zero-config
hector call assistant "Explain AI"

# Client mode
hector call assistant "Explain AI" --server http://localhost:8080

# With custom options (Direct mode, zero-config)
hector call assistant "Write code" --model gpt-4o --tools
hector call assistant "Use GitHub" --mcp-url https://api.composio.dev/v1/mcp
hector call assistant "Search docs" --docs ./knowledge

# With embedded auth in MCP URL
hector call assistant "Deploy app" --mcp-url https://user:token@composio.dev/mcp

# Disable streaming
hector call assistant "hello" --stream=false
```

**Modes:** Direct mode, Client mode

---

### `chat <agent>` - Interactive Chat

Start an interactive chat session with an agent.

```bash
# Direct mode with config
hector chat assistant --config hector.yaml

# Direct mode with zero-config
hector chat assistant

# Client mode
hector chat assistant --server http://localhost:8080

# With custom options (Direct mode, zero-config)
hector chat assistant --model gpt-4o
hector chat assistant --mcp-url https://api.composio.dev/v1/mcp --tools
```

**Interactive commands:**
- `/quit` or `/exit` - Exit chat
- `/clear` - Clear conversation history
- `/info` - Show agent information

**Modes:** Direct mode, Client mode

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
export MCP_SERVER_URL="https://api-key-here@api.composio.dev/v1/mcp"

# Local MCP server
export MCP_SERVER_URL="http://localhost:3000/mcp"

# With basic auth
export MCP_SERVER_URL="https://user:password@mcp.example.com"
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
MCP_SERVER_URL=https://api-key@api.composio.dev/v1/mcp
EOF

# Now zero-config mode works without flags
hector call assistant "hello"
hector serve
```

**Note:** Server URL and token for Client mode are NOT supported via environment variables. Always use explicit `--server` and `--token` flags for Client mode.

---

## Common Workflows

### Local Development

```bash
# Terminal 1: Start server (server mode)
hector serve --config hector.yaml

# Terminal 2: Test agents (client mode)
hector call assistant "test 1" --server http://localhost:8080
hector call assistant "test 2" --server http://localhost:8080
hector chat assistant --server http://localhost:8080
```

### Quick Scripting (Direct Mode)

```bash
#!/bin/bash
export OPENAI_API_KEY="sk-..."

# Process multiple inputs
for topic in "AI" "ML" "DL"; do
  hector call assistant "Explain $topic in one paragraph" > "${topic}.txt"
done
```

### Multi-Environment

Use shell aliases or scripts for different environments:

```bash
# Define aliases for different environments
alias hector-dev='hector --server http://localhost:8080'
alias hector-prod='hector --server https://prod.example.com --token $PROD_TOKEN'

# Use them
hector-dev call assistant "test"
hector-prod call assistant "test"
```

### Zero-Config Experimentation (Direct Mode)

```bash
export OPENAI_API_KEY="sk-..."

# Try different models
hector call assistant "test" --model gpt-4o-mini
hector call assistant "test" --model gpt-4o

# With tools
hector call assistant "List files" --tools

# Interactive
hector chat assistant
```

### MCP Integration Workflow

```bash
# Set up environment for Composio
export OPENAI_API_KEY="sk-..."
export MCP_SERVER_URL="https://your-api-key@api.composio.dev/v1/mcp"

# Now MCP tools are automatically available
hector call assistant "Create a GitHub issue"
hector serve  # Server also has MCP tools

# Override for testing different MCP server
hector call assistant "test" --mcp-url "http://localhost:3000/mcp"
```

---

## CLI Flag Order

**Important:** Flags must come **before** positional arguments (agent name, prompt):

```bash
# ✅ Correct
hector call --server http://localhost:8080 assistant "hello"
hector chat --model gpt-4o assistant

# ❌ Wrong - flags after agent name won't be parsed
hector call assistant "hello" --server http://localhost:8080
hector chat assistant --model gpt-4o
```

---

## Complete Example: Multi-Agent System

### Configuration File

```yaml
# config.yaml
agents:
  coder:
    name: "Coding Assistant"
    description: "Writes clean code"
    llm: "gpt-4o"
    prompt:
      system_role: "You are an expert programmer"
  
  reviewer:
    name: "Code Reviewer"
    description: "Reviews code quality"
    llm: "claude"
    prompt:
      system_role: "You are a code reviewer focused on best practices"

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
  
  claude:
    type: "anthropic"
    model: "claude-3-5-sonnet-20241022"
    api_key: "${ANTHROPIC_API_KEY}"
```

### Usage

```bash
# Start server (server mode)
hector serve --config config.yaml

# Use different agents (client mode)
hector call coder "Write a function" --server http://localhost:8080
hector call reviewer "Review the code" --server http://localhost:8080

# Interactive sessions
hector chat coder --server http://localhost:8080
hector chat reviewer --server http://localhost:8080
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
              → Use DIRECT MODE: Use commands without `--server`
```

---

## Next Steps

- **[Configuration Reference](CONFIGURATION.html)** - Complete configuration options
- **[Building Agents](AGENTS.html)** - Learn advanced agent features  
- **[Installation Guide](INSTALLATION.html)** - All installation methods
- **[Quick Start](QUICK_START.html)** - Get started in 5 minutes
- **[Examples](https://github.com/kadirpekel/hector/tree/main/configs)** - Sample configurations

**Ready to build?** Start with the [Quick Start Guide](QUICK_START.html) to get your first agent running.
