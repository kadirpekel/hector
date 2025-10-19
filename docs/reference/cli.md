---
title: CLI Reference
description: Complete command-line interface reference for Hector
---

# CLI Reference

Complete reference for all Hector command-line commands and options.

---

## Quick Reference

| Command | Purpose | Example |
|---------|---------|---------|
| `hector version` | Show version | `hector version` |
| `hector serve` | Start server | `hector serve --config config.yaml` |
| `hector list` | List agents | `hector list` |
| `hector info` | Agent details | `hector info assistant` |
| `hector call` | Single message | `hector call assistant "Hello"` |
| `hector chat` | Interactive chat | `hector chat assistant` |

---

## Global Flags

Available for all commands:

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--config FILE` | string | Configuration file path | None (required for config mode) |
| `--debug` | bool | Enable debug logging | `false` |
| `--log-level LEVEL` | string | Log level (debug/info/warn/error) | `info` |
| `--log-format FORMAT` | string | Log format (text/json) | `text` |
| `--help` | bool | Show help | - |

---

## Commands

### hector version

Show Hector version information.

**Usage:**
```bash
hector version
```

**Output:**
```
Hector version 0.x.x
```

---

### hector serve

Start Hector as an A2A server.

**Usage:**
```bash
hector serve [flags]
```

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--config FILE` | string | Configuration file (required for config mode) | None |
| `--port PORT` | int | Server port | `8080` |
| `--host HOST` | string | Server host | `0.0.0.0` |
| `--a2a-base-url URL` | string | A2A base URL | Auto-detected |

**Zero-Config Flags:**

When using zero-config mode (no `--config` flag), use these to configure quickly:

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--api-key KEY` | string | LLM API key | From env |
| `--model NAME` | string | Model name | `gpt-4o-mini` |
| `--provider NAME` | string | LLM provider (openai/anthropic/gemini) | `openai` |
| `--tools` | bool | Enable all built-in tools | `false` |
| `--mcp-url URL` | string | MCP server URL | - |
| `--docs FOLDER` | string | Documents folder for RAG | - |
| `--vectordb URL` | string | Vector database URL | `http://localhost:6333` |
| `--embedder-model MODEL` | string | Embedding model | `nomic-embed-text` |

**Examples:**

```bash
# With configuration file
hector serve --config config.yaml

# Zero-config mode
export OPENAI_API_KEY="sk-..."
hector serve --model gpt-4o --tools

# With RAG
hector serve --model gpt-4o --tools --docs ./knowledge

# Custom port
hector serve --config config.yaml --port 9090

# With MCP tools
hector serve --mcp-url http://localhost:3000
```

---

### hector list

List all available agents.

**Usage:**
```bash
hector list [flags]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--server URL` | string | Connect to remote server |
| `--token TOKEN` | string | Authentication token |

**Examples:**

```bash
# List local agents
hector list

# List agents on remote server
hector list --server http://remote:8080

# With authentication
hector list --server http://remote:8080 --token "eyJ..."
```

**Output:**
```
Available agents:
- assistant (My Assistant)
- coder (Coding Assistant)
- researcher (Research Specialist)
```

---

### hector info

Get detailed information about an agent.

**Usage:**
```bash
hector info AGENT [flags]
```

**Arguments:**

| Argument | Type | Description | Required |
|----------|------|-------------|----------|
| `AGENT` | string | Agent name | Yes |

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--server URL` | string | Connect to remote server |
| `--token TOKEN` | string | Authentication token |

**Examples:**

```bash
# Get info about local agent
hector info assistant

# Get info about remote agent
hector info assistant --server http://remote:8080
```

**Output:**
```
Agent: assistant
Name: My Assistant
LLM: gpt-4o-mini
Tools: search, write_file, execute_command
Memory: buffer_window
Reasoning: chain-of-thought
```

---

### hector call

Send a single message to an agent.

**Usage:**
```bash
hector call [AGENT] MESSAGE [flags]
```

**Arguments:**

| Argument | Type | Description | Required |
|----------|------|-------------|----------|
| `AGENT` | string | Agent name | ❌ Zero-config / ✅ Config mode / ✅ Client mode |
| `MESSAGE` | string | Message to send | Always required |

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--server URL` | string | Connect to remote server | - |
| `--token TOKEN` | string | Authentication token | - |
| `--stream` | bool | Enable streaming | `true` |
| `--session ID` | string | Session ID for context | - |
| `--timeout DURATION` | duration | Request timeout | `5m` |

**Zero-Config Flags (local mode only):**

| Flag | Type | Description |
|------|------|-------------|
| `--model NAME` | string | Override model |
| `--provider NAME` | string | Override provider |
| `--tools` | bool | Enable tools |

**Examples:**

```bash
# Zero-config mode (NO agent name)
export OPENAI_API_KEY="sk-..."
hector call "What is quantum computing?"

# Config mode (agent name REQUIRED and validated immediately)
hector call --config config.yaml assistant "What is the capital of France?"
hector call --config config.yaml coder "Fix the bug" --session sess_123

# Client mode (agent name REQUIRED)
hector call --server http://remote:8080 assistant "Hello" --token "eyJ..."

# No streaming
hector call --config config.yaml assistant "Hello" --stream=false

# With timeout
hector call --config config.yaml assistant "Complex analysis" --timeout 10m
```

---

### hector chat

Interactive chat with an agent.

**Usage:**
```bash
hector chat [AGENT] [flags]
```

**Arguments:**

| Argument | Type | Description | Required |
|----------|------|-------------|----------|
| `AGENT` | string | Agent name | ❌ Zero-config / ✅ Config mode / ✅ Client mode |

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--server URL` | string | Connect to remote server |
| `--token TOKEN` | string | Authentication token |
| `--session ID` | string | Session ID for context |

**Examples:**

```bash
# Zero-config mode (NO agent name)
export OPENAI_API_KEY="sk-..."
hector chat

# Config mode (agent name REQUIRED and validated immediately)
hector chat --config config.yaml assistant
hector chat --config config.yaml coder --session sess_123

# Client mode (agent name REQUIRED)
hector chat --server http://remote:8080 assistant
hector chat --server http://remote:8080 assistant --token "eyJ..."
```

**In Chat:**

- Type message and press Enter to send
- Type `exit` or press Ctrl+C to quit
- Type `/help` for chat commands (if available)

---

## Operating Modes

Hector operates in three modes based on command and flags:

### Local Mode

Run agents in-process without a server.

**Triggers:**
- Any command without `--server` flag

**Supports:**
- Configuration files (`--config`)
- Zero-config flags (`--model`, `--tools`, etc.)

**Example:**
```bash
hector call assistant "Hello" --config config.yaml
```

### Server Mode

Run Hector as an A2A server.

**Triggers:**
- `hector serve` command

**Supports:**
- Configuration files
- Zero-config flags

**Example:**
```bash
hector serve --config config.yaml
```

### Client Mode

Connect to a remote A2A server.

**Triggers:**
- Any command with `--server` flag

**Supports:**
- Only client-side flags (`--server`, `--token`, `--stream`)

**Does NOT support:**
- Configuration files
- Zero-config flags
- (Server controls configuration)

**Example:**
```bash
hector call assistant "Hello" --server http://remote:8080
```

---

## Environment Variables

Hector recognizes these environment variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `OPENAI_API_KEY` | OpenAI API key | `sk-...` |
| `ANTHROPIC_API_KEY` | Anthropic API key | `sk-ant-...` |
| `GEMINI_API_KEY` | Google Gemini API key | `AI...` |
| `HECTOR_CONFIG` | Default config file path | `/etc/hector/config.yaml` |
| `QDRANT_HOST` | Qdrant host | `localhost` |
| `OLLAMA_HOST` | Ollama host | `http://localhost:11434` |
| `LOG_LEVEL` | Default log level | `info` |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Configuration error |
| `3` | Authentication error |
| `4` | Network error |
| `5` | Agent not found |

---

## Configuration File

Hector requires an explicit configuration file via the `--config` flag:

```bash
hector serve --config /path/to/config.yaml
hector chat --config myconfig.yaml agent_name
```

**No default location**: Hector does not automatically search for config files. This makes the behavior explicit and predictable.

**Why explicit?**
- Clear distinction between zero-config and config modes
- No "magic" behavior that searches multiple locations
- Easier to understand and debug
- Follows Go's philosophy of explicitness

For quick experimentation without a config file, use zero-config mode (see examples above).

---

## Common Patterns

### Quick Experimentation

```bash
export OPENAI_API_KEY="sk-..."
hector call "What is recursion?"
```

### Development with Config

```bash
hector serve --config dev-config.yaml &
hector chat assistant
```

### Production Deployment

```bash
hector serve \
  --config prod-config.yaml \
  --port 8080 \
  --log-format json \
  --log-level info
```

### Remote Agent Access

```bash
export HECTOR_SERVER="https://agents.company.com"
export HECTOR_TOKEN="eyJ..."

hector list --server $HECTOR_SERVER --token $HECTOR_TOKEN
hector call assistant "task" --server $HECTOR_SERVER --token $HECTOR_TOKEN
```

### Scripting

```bash
#!/bin/bash
set -e

# Start server
hector serve --config config.yaml &
SERVER_PID=$!

# Wait for startup
sleep 5

# Run tasks
hector call assistant "Analyze data" > results.txt
hector call assistant "Generate report" >> results.txt

# Cleanup
kill $SERVER_PID
```

---

## Troubleshooting

### "command not found: hector"

**Solution:**
```bash
# Check installation
which hector

# Add to PATH
export PATH="/usr/local/bin:$PATH"
```

### "configuration file not found"

**Solution:**
```bash
# Specify config explicitly
hector serve --config /path/to/config.yaml

# Or use zero-config
hector serve --model gpt-4o --tools
```

### "API key not found"

**Solution:**
```bash
# Set environment variable
export OPENAI_API_KEY="sk-..."

# Or pass as flag
hector serve --api-key "sk-..."
```

### "agent 'X' not found"

When using `--config`, agent names are validated immediately:

```
Error: agent 'myagent' not found

Available agents in config:
  - assistant
  - coder
```

**Solution:**
```bash
# Use an agent that exists in your config
hector call --config config.yaml assistant "Hello"

# Check available agents
hector list --config config.yaml
```

### "connection refused"

**Solution:**
```bash
# Check server is running
curl http://localhost:8080/agents

# Check port
hector serve --port 9090
```

---

## Next Steps

- **[Configuration Reference](configuration.md)** - Complete YAML reference
- **[API Reference](api.md)** - HTTP/gRPC API details
- **[Getting Started](../getting-started/installation.md)** - Installation guide
- **[Quick Start](../getting-started/quick-start.md)** - Get started in 5 minutes

---

## Related Topics

- **[Agent Overview](../core-concepts/overview.md)** - Understanding agents
- **[Deployment](../how-to/deploy-production.md)** - Production deployment
- **[Architecture](architecture.md)** - How Hector works

