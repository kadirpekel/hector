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
| `hector validate` | Validate config | `hector validate config.yaml` |
| `hector serve` | Start server | `hector serve --config config.yaml` |
| `hector info` | Agent details | `hector info assistant` |
| `hector call` | Single message | `hector call "Hello" --agent assistant --config config.yaml` |
| `hector chat` | Interactive chat | `hector chat --agent assistant --config config.yaml` |

---

## Global Flags

Available for all commands:

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--config PATH` | string | Configuration path (file path or backend key) | None (zero-config if omitted) |
| `--config-type TYPE` | string | Configuration backend (`file`, `consul`, `etcd`, `zookeeper`) | `file` |
| `--config-watch` | bool | Watch for configuration changes and auto-reload | `false` |
| `--config-endpoints ENDPOINTS` | string | Comma-separated backend endpoints | Backend-specific defaults |
| `--debug` | bool | Enable debug logging | `false` |
| `--help` | bool | Show help | - |

**Configuration Backends:**
- `file` - Local YAML file (default)
- `consul` - HashiCorp Consul KV store
- `etcd` - Etcd distributed key-value store
- `zookeeper` - Apache ZooKeeper

See [Distributed Configuration](distributed-configuration.md) for detailed usage.

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

### hector validate

Validate configuration file syntax, semantics, and references.

**Usage:**
```bash
hector validate CONFIG [flags]
```

**Arguments:**

| Argument | Type | Description | Required |
|----------|------|-------------|----------|
| `CONFIG` | string | Configuration file path | ✅ Always required |

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `-f, --format` | string | Output format: `compact`, `verbose`, `json` | `compact` |
| `-p, --print-config` | bool | Print expanded configuration | `false` |

**What Gets Validated:**

- ✅ Configuration file syntax (YAML/JSON)
- ✅ LLM provider configurations (API keys, models, endpoints)
- ✅ Database provider settings (connection strings, drivers)
- ✅ Embedder configurations (models, dimensions)
- ✅ Agent configurations (types, LLM references, tools)
- ✅ Tool configurations (command sandboxing, file size limits)
- ✅ Document stores (path existence, embedder references)
- ✅ Session stores (backend types, SQL configurations)
- ✅ Plugin discovery and loading
- ✅ Cross-reference validation (agent → LLM → database links)

**Output Formats:**

**Compact (default)** - One-line format ideal for CI/CD:
```bash
$ hector validate config.yaml
config.yaml: valid
```

**Verbose** - Human-readable detailed output:
```bash
$ hector validate config.yaml --format=verbose
Configuration Validation Successful
===================================

File:   config.yaml
Status: ✓ Valid
```

**JSON** - Machine-parseable for tooling:
```bash
$ hector validate config.yaml --format=json
{
  "valid": true,
  "file": "config.yaml"
}
```

**Examples:**

```bash
# Basic validation
hector validate config.yaml

# Verbose output with details
hector validate config.yaml --format=verbose

# JSON output for CI/CD pipelines
hector validate config.yaml --format=json | jq .valid

# Print expanded configuration (shows defaults, shortcuts, env vars)
hector validate config.yaml --print-config

# Print expanded config as JSON
hector validate config.yaml --print-config --format=json
```

**Exit Codes:**

| Code | Meaning |
|------|---------|
| `0` | Configuration is valid |
| `1` | Configuration has errors |

**Configuration Expansion:**

When using `--print-config`, the command shows the configuration after full processing:

```bash
# Original config with shortcuts
# docs_folder: "."
# enable_tools: true

# Expanded config shows:
# - Auto-created document_stores
# - Auto-created default-database and default-embedder
# - All tool configurations with defaults
# - Environment variables resolved
```

**Use Cases:**

1. **Pre-deployment Validation:**
```bash
#!/bin/bash
if hector validate prod-config.yaml; then
  hector serve --config prod-config.yaml
else
  echo "Invalid configuration - deployment aborted"
  exit 1
fi
```

2. **CI/CD Integration:**
```yaml
# .github/workflows/validate.yml
- name: Validate Hector configs
  run: |
    for config in configs/*.yaml; do
      hector validate "$config" --format=json || exit 1
    done
```

3. **Configuration Debugging:**
```bash
# See what the config becomes after processing
hector validate config.yaml --print-config > expanded.yaml
diff config.yaml expanded.yaml
```

4. **Documentation Generation:**
```bash
# Extract final config for documentation
hector validate config.yaml --print-config --format=json > config-schema.json
```

**Error Messages:**

When other commands encounter config errors, they provide helpful hints:

```bash
$ hector serve --config bad-config.yaml
Configuration validation failed in bad-config.yaml:
  agent 'assistant' validation failed: llm 'nonexistent' not found

Hint: Run 'hector validate bad-config.yaml' for detailed diagnostics
```

**Best Practices:**

- ✅ Validate configs before committing to version control
- ✅ Add validation to CI/CD pipelines
- ✅ Use `--print-config` to understand shortcut expansions
- ✅ Use JSON output for automated tooling
- ✅ Validate before production deployments

---

### hector serve

Start Hector as an A2A server.

**Usage:**
```bash
hector serve [AGENT] [flags]
```

**Arguments:**

| Argument | Type | Description | Required |
|----------|------|-------------|----------|
| `AGENT` | string | Agent name (zero-config mode only) | ❌ (defaults to `assistant`) |

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
| `--provider NAME` | string | LLM provider (openai/anthropic/gemini/ollama) | `openai` |
| `--role TEXT` | string | Set agent identity (WHO, merges with strategy) | - |
| `--instruction TEXT` | string | Add user guidance (WHAT, highest priority) | - |
| `--tools` | bool | Enable all built-in tools | `false` |
| `--mcp-url URL` | string | MCP server URL | - |
| `--docs FOLDER` | string | Documents folder for RAG | - |
| `--vectordb URL` | string | Vector database URL | `http://localhost:6334` |
| `--embedder-model MODEL` | string | Embedding model | `nomic-embed-text` |

**Examples:**

```bash
# With configuration file (agent names defined in config)
hector serve --config config.yaml

# Zero-config mode (default agent name: "assistant")
export OPENAI_API_KEY="sk-..."
hector serve --model gpt-4o --tools

# Zero-config mode with custom agent name
hector serve --tools gopher
hector serve myagent --model gpt-4o --tools

# With custom role (replaces strategy's system role)
hector serve --role "You are a security expert who focuses on identifying vulnerabilities"

# With additional instruction (adds user guidance, highest priority)
hector serve --instruction "Focus on security and best practices"

# With both for fine-grained control (role + guidance)
hector serve --role "You are a code reviewer" --instruction "Be strict about error handling"

# With RAG
hector serve coder --model gpt-4o --tools --docs-folder ./knowledge

# With RAG and MCP parsing (Docling example)
hector serve coder --model gpt-4o --docs-folder ./knowledge --mcp-parser-tool "convert_document_into_docling_document"

# Custom port
hector serve --config config.yaml --port 9000

# With MCP tools
hector serve --mcp-url http://localhost:3000

# With observability
hector serve --model gpt-4o --tools --observe

# All flags before positional arg (recommended)
hector serve --tools --model gpt-4o coder
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
| `AGENT` | string | Agent name (optional if URL points to specific agent) | Conditional |

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--url URL` | string | Agent card URL or service base URL |
| `--token TOKEN` | string | Authentication token |

**Examples:**

```bash
# Get info about local agent
hector info assistant --config config.yaml

# Get info from A2A service
hector info assistant --url http://remote:8080

# Direct agent card URL (agent name optional)
hector info --url http://remote:8080/v1/agents/assistant/.well-known/agent-card.json
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
hector call MESSAGE [flags]
```

**Arguments:**

| Argument | Type | Description | Required |
|----------|------|-------------|----------|
| `MESSAGE` | string | Message to send | ✅ Always required |

**Flags:**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--agent NAME` | string | Agent name (required with `--config`, optional if `--url` points to specific agent) | - |
| `--url URL` | string | Agent card URL or service base URL (enables client mode) | - |
| `--token TOKEN` | string | Authentication token | - |
| `--[no-]stream` | bool | Enable/disable streaming | `true` (use `--no-stream` to disable) |
| `--session ID` | string | Session ID for context | - |

**Zero-Config Flags (local mode only):**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--provider NAME` | string | LLM provider | `openai` |
| `--model NAME` | string | Model name | - |
| `--api-key KEY` | string | API key | From env |
| `--base-url URL` | string | Custom API base URL | - |
| `--role TEXT` | string | Set agent identity (WHO, merges with strategy) | - |
| `--instruction TEXT` | string | Add user guidance (WHAT, highest priority) | - |
| `--tools` | bool | Enable built-in tools | `false` |
| `--mcp-url URL` | string | MCP server URL | - |
| `--mcp-parser-tool TOOL_NAME` | string | MCP parser tool name(s) for document parsing (comma-separated for fallback chain, e.g., `convert_document_into_docling_document`). Requires `--docs-folder`. Check available tools if you see a warning. | - |
| `--docs-folder PATH` | string | Documents folder for RAG | - |

**Examples:**

```bash
# Zero-config mode (NO --agent flag needed)
export OPENAI_API_KEY="sk-..."
hector call "What is quantum computing?"
hector call "Write a poem about Go" --tools

# Config mode (--agent flag REQUIRED)
hector call "What is the capital of France?" --agent assistant --config config.yaml
hector call "Fix the bug" --agent coder --config config.yaml --session sess_123

# Client mode - service base URL (--agent flag REQUIRED)
hector call "Hello" --agent assistant --url http://remote:8080 --token "eyJ..."

# Client mode - direct agent card URL (--agent flag OPTIONAL)
hector call "Hello" --url http://remote:8080/v1/agents/assistant/.well-known/agent-card.json --token "eyJ..."
hector call "Hello" --url http://remote:8080/.well-known/agent-card.json  # Single-agent service

# No streaming
hector call "Hello" --agent assistant --config config.yaml --no-stream

# Flags can appear anywhere (Kong flexibility)
hector call --config config.yaml --agent assistant "What's 2+2?"
hector call "Help me" --tools --model gpt-4o
```

---

### hector chat

Interactive chat with an agent.

**Usage:**
```bash
hector chat [flags]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--agent NAME` | string | Agent name (required with `--config`, optional if `--url` points to specific agent) |
| `--url URL` | string | Agent card URL or service base URL |
| `--token TOKEN` | string | Authentication token |
| `--session ID` | string | Session ID for context |
| `--[no-]stream` | bool | Enable/disable streaming (default: enabled) |

**Zero-Config Flags (local mode only):**

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--provider NAME` | string | LLM provider | `openai` |
| `--model NAME` | string | Model name | - |
| `--api-key KEY` | string | API key | From env |
| `--base-url URL` | string | Custom API base URL | - |
| `--role TEXT` | string | Set agent identity (WHO, merges with strategy) | - |
| `--instruction TEXT` | string | Add user guidance (WHAT, highest priority) | - |
| `--tools` | bool | Enable built-in tools | `false` |
| `--mcp-url URL` | string | MCP server URL | - |
| `--mcp-parser-tool TOOL_NAME` | string | MCP parser tool name(s) for document parsing (comma-separated for fallback chain, e.g., `convert_document_into_docling_document`). Requires `--docs-folder`. Check available tools if you see a warning. | - |
| `--docs-folder PATH` | string | Documents folder for RAG | - |

**Examples:**

```bash
# Zero-config mode (NO --agent flag needed)
export OPENAI_API_KEY="sk-..."
hector chat
hector chat --tools --model gpt-4o

# Config mode (--agent flag REQUIRED)
hector chat --agent assistant --config config.yaml
hector chat --agent coder --config config.yaml --session sess_123

# Client mode - service base URL (--agent flag REQUIRED)
hector chat --agent assistant --url http://remote:8080
hector chat --agent assistant --url http://remote:8080 --token "eyJ..."

# Client mode - direct agent card URL (--agent flag OPTIONAL)
hector chat --url http://remote:8080/v1/agents/assistant/.well-known/agent-card.json

# Flags flexible positioning (Kong feature)
hector chat --config config.yaml --agent assistant
hector chat --agent assistant --config config.yaml --no-stream
```

**In Chat:**

- Type message and press Enter to send
- Type `exit` or press Ctrl+C to quit
- Type `/help` for chat commands (if available)

---

### hector task get

Get details about a specific task.

**Usage:**
```bash
hector task get <agent> <task-id> [flags]
```

**Arguments:**

| Argument | Type | Description | Required |
|----------|------|-------------|----------|
| `AGENT` | string | Agent name that owns the task | ✅ Always required |
| `TASK_ID` | string | Task ID to retrieve | ✅ Always required |

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--url URL` | string | Agent card URL or service base URL (enables client mode) |
| `--token TOKEN` | string | Authentication token |
| `--config FILE` | string | Configuration file (for local mode) |

**Examples:**

```bash
# Client mode - query task from remote service
hector task get assistant task-abc123 --url http://localhost:8080

# Local mode - query task from config
hector task get assistant task-abc123 --config config.yaml

# With authentication
hector task get assistant task-abc123 --url https://prod:8080 --token "eyJ..."

# Works with ANY A2A service
hector task get researcher task-xyz789 --url http://other-service:8080
```

**Output:**
```
Task ID: task-abc123
Status: TASK_STATE_COMPLETED
Context ID: ctx-xyz
Created: 2024-01-15 10:30:00
Updated: 2024-01-15 10:35:00

History:
  [USER] Analyze the codebase
  [ASSISTANT] Analysis complete...

Artifacts:
  - report.pdf (2.5 MB)
```

**Notes:**
- Task commands require explicit agent configuration
- Not available in zero-config mode (use `call` or `chat` instead)
- Works with any A2A-compliant service

---

### hector task cancel

Cancel a running or pending task.

**Usage:**
```bash
hector task cancel <agent> <task-id> [flags]
```

**Arguments:**

| Argument | Type | Description | Required |
|----------|------|-------------|----------|
| `AGENT` | string | Agent name that owns the task | ✅ Always required |
| `TASK_ID` | string | Task ID to cancel | ✅ Always required |

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--url URL` | string | Agent card URL or service base URL (enables client mode) |
| `--token TOKEN` | string | Authentication token |
| `--config FILE` | string | Configuration file (for local mode) |

**Examples:**

```bash
# Client mode
hector task cancel assistant task-abc123 --url http://localhost:8080

# Local mode
hector task cancel assistant task-abc123 --config config.yaml

# With authentication
hector task cancel assistant task-abc123 --url https://prod:8080 --token "eyJ..."
```

**Output:**
```
✅ Task cancelled successfully

Task ID: task-abc123
Status: TASK_STATE_CANCELLED
Context ID: ctx-xyz
```

**Notes:**
- Can only cancel tasks in non-terminal states (SUBMITTED, WORKING)
- Tasks already COMPLETED, FAILED, or CANCELLED cannot be cancelled again
- The command will succeed silently if task is already in a terminal state

---

## Session Support

The `--session` flag enables conversation resumption across multiple CLI invocations. When you provide the same session ID, the agent remembers the previous conversation context.

### How It Works

```bash
# First conversation
hector call "Remember: meeting at 3pm" --agent assistant --config config.yaml --session work
# Agent: Got it! Meeting at 3pm.

# Later (even after restart)
hector call "When is the meeting?" --agent assistant --config config.yaml --session work
# Agent: The meeting is at 3pm.
```

**Key Points:**
- Same `--session` ID = same conversation context
- Works with `call`, `chat`, and `task` commands
- Requires session persistence configured (see [Configuration Reference](configuration.md#session-store))

### Session IDs

**Format:** Any string (alphanumeric, hyphens, underscores)

**Examples:**
- `work-2024-01-15`
- `customer-support-case-12345`
- `coding-session-abc`
- `$(uuidgen)` (auto-generate UUID)

### CLI Session Examples

**Interactive chat with session:**

```bash
# First session
hector chat --agent assistant --config config.yaml --session my-chat
You: Remember my name is Alice
Agent: Got it, Alice!
You: exit

# Resume later
hector chat --agent assistant --config config.yaml --session my-chat
You: What's my name?
Agent: Your name is Alice.
```

**Single calls with shared session:**

```bash
# Store information
hector call "Project ALPHA started" --agent assistant --config config.yaml --session work

# Query later
hector call "What project did we start?" --agent assistant --config config.yaml --session work
# Agent remembers: Project ALPHA
```

**Auto-generated session IDs:**

```bash
# Chat generates and displays session ID
hector chat --agent assistant --config config.yaml
# Output: 💾 Session ID: cli-chat-1729612345
#         Resume later with: --session=cli-chat-1729612345
```

### Configuration Requirement

Session persistence requires a `session_stores` configuration:

```yaml
session_stores:
  main-db:
    backend: sql
    sql:
      driver: sqlite
      database: ./data/sessions.db

agents:
  assistant:
    session_store: "main-db"  # Enables session persistence
    memory:
      working:
        strategy: "summary_buffer"
```

Without `session_store` configured, sessions work within a single CLI command but don't persist.

See [Configuration Reference](configuration.md#session-store) for full configuration guide.

---

## Operating Modes

Hector operates in three modes based on command and flags:

### Local Mode

Run agents in-process without a server.

**Triggers:**
- Any command without `--url` flag

**Supports:**
- Configuration files (`--config`)
- Zero-config flags (`--model`, `--tools`, etc.)

**Example:**
```bash
hector call "Hello" --agent assistant --config config.yaml
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

Connect to any A2A-compliant agent service.

**Triggers:**
- Any command with `--url` flag

**Supports:**
- Agent card URL (any A2A service)
- Service base URL (auto-discovers agents)
- All A2A transports (gRPC, REST, JSON-RPC)

**Does NOT support:**
- Configuration files (the service defines its own config)
- Zero-config flags (the service controls agent behavior)

**Example:**
```bash
# Service base URL (multi-agent service)
hector call "Hello" --agent assistant --url http://remote:8080

# Direct agent card URL (single agent)
hector call "Hello" --url http://remote:8080/.well-known/agent-card.json

# With authentication
hector call "Hello" --agent assistant --url http://remote:8080 --token "eyJ..."
```

**A2A Interoperability:**

The `--url` flag makes Hector's CLI work with ANY A2A-compliant service, not just Hector servers:

```bash
# Connect to Hector service
hector call "task" --url http://hector-service:8080 --agent assistant

# Connect to ANY other A2A service
hector call "task" --url http://other-a2a-service:8080 --agent some-agent

# Direct agent card discovery
hector info --url http://service/.well-known/agent-card.json
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
| `HECTOR_URL` | Default agent card URL or service base URL (client mode) | `http://localhost:8080` |
| `HECTOR_TOKEN` | Authentication token | `eyJ...` |
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

# Pure strategy defaults (no customization)
hector call "What is recursion?"

# With custom role (WHO - merges with strategy)
hector call "analyze this" --role "You are a security analyst"

# With custom guidance (WHAT - highest priority)
hector call "analyze this" --instruction "Focus on vulnerabilities and rate severity"

# With both (complete customization)
hector call "review code" --role "You are a senior code reviewer" --instruction "Be strict about error handling"
```

### Development with Config

```bash
hector serve --config dev-config.yaml &
hector chat --agent assistant --config dev-config.yaml
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
export HECTOR_URL="https://agents.company.com"
export HECTOR_TOKEN="eyJ..."

hector info assistant --url $HECTOR_URL --token $HECTOR_TOKEN
hector call "task" --agent assistant --url $HECTOR_URL --token $HECTOR_TOKEN

# Or direct agent card URL
hector call "task" --url https://agents.company.com/.well-known/agent-card.json --token $HECTOR_TOKEN
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
hector call "Analyze data" --agent assistant --config config.yaml > results.txt
hector call "Generate report" --agent assistant --config config.yaml >> results.txt

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
hector call "Hello" --agent assistant --config config.yaml

# Check available agents with info
hector info assistant --config config.yaml
```

### "connection refused"

**Solution:**
```bash
# Check server is running
curl http://localhost:8080/agents

# Check if different port is needed
hector serve --port 9000
```

---

## Prompt Customization

The `--role` and `--instruction` flags provide flexible prompt customization in zero-config mode:

**How it works:**
- **Strategy Defaults (BASE)**: Each reasoning strategy has optimized prompts for tool execution, workflow patterns, etc.
- **--role (MERGE)**: Sets the agent's identity (WHO) - replaces strategy's system role, keeps behavior patterns
- **--instruction (HIGHEST)**: Adds your specific guidance (WHAT) - applied last, highest priority

**Examples:**
```bash
# Pure strategy (optimized defaults)
hector call "task"

# Custom role (WHO you are)
hector call "task" --role "You are a Python expert"

# Custom guidance (WHAT you want)
hector call "task" --instruction "Focus on performance, use type hints"

# Both (complete customization)
hector call "task" --role "You are a security analyst" --instruction "Prioritize OWASP Top 10"
```

**See Also:** [Prompts Guide](../core-concepts/prompts.md) for complete documentation on prompt configuration.

---

## Next Steps

- **[Configuration Reference](configuration.md)** - Complete YAML reference
- **[Prompts Guide](../core-concepts/prompts.md)** - Prompt configuration in depth
- **[API Reference](api.md)** - HTTP/gRPC API details
- **[Getting Started](../getting-started/installation.md)** - Installation guide
- **[Quick Start](../getting-started/quick-start.md)** - Get started in 5 minutes

---

## Related Topics

- **[Agent Overview](../core-concepts/overview.md)** - Understanding agents
- **[Configuration Reference](configuration.md)** - Production deployment
- **[Architecture](architecture.md)** - How Hector works

