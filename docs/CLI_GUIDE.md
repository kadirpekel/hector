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

## Three Modes of Operation

Hector CLI operates in **three distinct modes** depending on your use case:

| Mode | Trigger | Config Support | Zero-Config Support | Use Case |
|------|---------|---------------|---------------------|----------|
| **Server** | `serve` command | ✅ Yes | ✅ Yes | Host agents for multiple clients |
| **Client** | `--server` flag present | ❌ No | ❌ No | Connect to remote Hector server |
| **Direct** | No `--server` flag | ✅ Yes | ✅ Yes | In-process agent execution |

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
- `--api-key KEY` - OpenAI API key (or use `OPENAI_API_KEY` env var)
- `--model NAME` - Model name (default: `gpt-4o-mini`)
- `--tools` - Enable all local tools
- `--mcp URL` - MCP server URL for tool integration
- `--docs FOLDER` - Document folder for RAG
- `--embedder-model MODEL` - Embedder model (default: `nomic-embed-text`)
- `--vectordb URL` - Vector database URL (default: `http://localhost:6333`)

---

### 2. Client Mode

Connect to a **remote Hector server** as a client. No local config or zero-config options supported.

**Use cases:** Connect to remote servers, team collaboration, shared agent infrastructure

**Activated by:** Presence of `--server` flag

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

**⚠️ Important:** In client mode, `--config`, `--model`, `--tools`, and other zero-config flags are **NOT supported** because you're connecting to a remote server with its own configuration.

**Environment variables:**
```bash
# Set defaults to avoid repeating flags
export HECTOR_SERVER="http://remote-server:8080"
export HECTOR_TOKEN="your-auth-token"

# Now omit --server and --token
hector list
hector call assistant "hello"
```

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
- `--api-key KEY` - OpenAI API key (or use `OPENAI_API_KEY`)
- `--base-url URL` - OpenAI API base URL (default: `https://api.openai.com/v1`)
- `--model NAME` - Model name (default: `gpt-4o-mini`)
- `--tools` - Enable local tools (file ops, commands)

**📖 For all configuration options, see [Configuration Reference](https://gohector.dev/CONFIGURATION.html)**

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

# With custom options
hector call assistant "Write code" --model gpt-4o --tools
hector call assistant "Search docs" --docs ./knowledge

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

# With custom model (direct mode)
hector chat assistant --model gpt-4o
```

**Interactive commands:**
- `/quit` or `/exit` - Exit chat
- `/clear` - Clear conversation history
- `/info` - Show agent information

**Modes:** Direct mode, Client mode

---

## Environment Variables

### API Keys

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# Google Gemini
export GEMINI_API_KEY="AI..."
```

### Server Defaults

```bash
# Default server URL for client mode
export HECTOR_SERVER="http://localhost:8080"

# Default authentication token
export HECTOR_TOKEN="your-bearer-token"
```

### Using .env Files

Hector automatically loads `.env` files:

```bash
# Create .env file
cat > .env << EOF
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
HECTOR_SERVER=http://localhost:8080
HECTOR_TOKEN=abc123
EOF

# Now commands work without flags
hector call assistant "hello"
hector serve
```

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

```bash
# Development (client mode)
export HECTOR_SERVER="http://localhost:8080"
hector call assistant "test"

# Production (client mode)
export HECTOR_SERVER="https://prod.example.com"
export HECTOR_TOKEN="prod-token"
hector call assistant "test"
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

---

## CLI Flag Order

**Important:** Flags must come **before** positional arguments (agent name, prompt):

```bash
# ✅ Correct
hector call --server http://localhost:8080 assistant "hello"
hector chat --model gpt-4o assistant

# ❌ Wrong - flags after agent name are ignored
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
