---
layout: default
title: CLI Guide
nav_order: 3
parent: Getting Started
description: "Complete command-line interface reference for Hector"
---

# Hector CLI Guide

Get up and running with Hector AI Agent Platform in minutes.

---

## Quick Start

### Zero-Config Mode (Fastest)

No configuration file needed! Just set your API key and start:

```bash
# Set API key
export OPENAI_API_KEY="sk-..."

# Start using immediately
hector call assistant "Explain quantum computing"

# Interactive chat
hector chat assistant

# Enable tools
hector call assistant "List files in current directory" --tools

# Custom model
hector call assistant "Write code" --model gpt-4o
```

**Zero-config flags:**
- `--api-key KEY` - API key (or use `OPENAI_API_KEY` env var)
- `--model NAME` - Model name (default: `gpt-4o-mini`)
- `--tools` - Enable local tools (file ops, commands)
- `--mcp URL` - MCP server for tool integration
- `--docs FOLDER` - Document folder for RAG

### With Configuration File

Create `hector.yaml` for advanced features:

```yaml
agents:
  assistant:
    name: "My Assistant"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a helpful assistant who provides clear answers.

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
```

Use your configuration:

```bash
# Direct mode (no server)
hector call assistant "hello" --config hector.yaml
hector chat assistant --config hector.yaml

# Server mode
hector serve --config hector.yaml

# In another terminal, connect to server
hector call assistant "hello" --server http://localhost:8080
```

**📖 For all configuration options, see [Configuration Reference](https://gohector.dev/CONFIGURATION.html)**

---

## Two Modes of Operation

Hector operates in two modes depending on your needs:

### Direct Mode (Default)

Agent runs in-process without a server. Best for:
- Quick tasks and experimentation
- Single-agent workflows
- Local development
- CI/CD scripts

```bash
# No server needed
hector call assistant "hello"
hector chat assistant
hector list
```

### Server Mode

Runs an A2A protocol server for distributed systems. Best for:
- Multi-agent systems
- Production deployments
- Remote agents
- API access

```bash
# Terminal 1: Start server
hector serve --config hector.yaml

# Terminal 2: Connect to server
hector call assistant "hello" --server http://localhost:8080
hector chat assistant --server http://localhost:8080
hector list --server http://localhost:8080
```

**Mode is determined by presence of `--server` flag.**

---

## Essential Commands

### List Agents

```bash
# Direct mode - list from config
hector list --config hector.yaml

# Server mode - list from server
hector list --server http://localhost:8080

# Zero-config mode - list default agent
hector list
```

### Get Agent Info

```bash
# Direct mode
hector info assistant --config hector.yaml

# Server mode
hector info assistant --server http://localhost:8080
```

### Execute Task (One-shot)

```bash
# Direct mode
hector call assistant "Explain machine learning" --config hector.yaml

# Server mode
hector call assistant "Explain machine learning" --server http://localhost:8080

# Zero-config
hector call assistant "Explain machine learning"

# Disable streaming (enabled by default)
hector call assistant "hello" --stream=false
```

### Interactive Chat

```bash
# Direct mode
hector chat assistant --config hector.yaml

# Server mode
hector chat assistant --server http://localhost:8080

# Zero-config
hector chat assistant
```

**Interactive commands:**
- `/quit` or `/exit` - Exit chat
- `/clear` - Clear conversation history
- `/info` - Show agent information

### Start Server

```bash
# With config file
hector serve --config hector.yaml

# Zero-config with flags
hector serve --api-key $OPENAI_API_KEY --tools --model gpt-4o

# Debug mode
hector serve --config hector.yaml --debug
```

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

### Using .env Files

Hector automatically loads `.env` files:

```bash
# Create .env file
cat > .env << EOF
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
EOF

# Now commands work without flags
hector call assistant "hello"
hector serve
```

### Server Defaults

```bash
# Set default server URL
export HECTOR_SERVER="http://localhost:8080"

# Set default token
export HECTOR_TOKEN="your-bearer-token"

# Now you can omit --server flag
hector list
hector call assistant "hello"
```

---

## Common Workflows

### Local Development

```bash
# Terminal 1: Start server
hector serve --config hector.yaml

# Terminal 2: Test agents
hector call assistant "test 1"
hector call assistant "test 2"
hector chat assistant
```

### Zero-Config Scripting

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
# Development
export HECTOR_SERVER="http://localhost:8080"
hector call assistant "test"

# Production
export HECTOR_SERVER="https://prod.example.com"
export HECTOR_TOKEN="prod-token"
hector call assistant "test"
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

## Quick Examples

### Simple Q&A

```bash
hector call assistant "What is machine learning?"
```

### With Tools

```bash
hector call assistant "Count files in this directory" --tools
```

### With Custom Model

```bash
hector call assistant "Write a poem" --model gpt-4o
```

### Interactive Chat

```bash
hector chat assistant
> Tell me about AI
[Agent responds...]
> What are the applications?
[Agent responds...]
> /quit
```

### Multi-Agent Server

```yaml
# config.yaml
agents:
  coder:
    name: "Coding Assistant"
    llm: "gpt-4o"
  
  reviewer:
    name: "Code Reviewer"
    llm: "claude"

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
  
  claude:
    type: "anthropic"
    model: "claude-3-7-sonnet-latest"
    api_key: "${ANTHROPIC_API_KEY}"
```

```bash
# Start server
hector serve --config config.yaml

# Use different agents
hector call coder "Write a function" --server http://localhost:8080
hector call reviewer "Review the code" --server http://localhost:8080
```

---

## Next Steps

- **[Configuration Reference](CONFIGURATION.html)** - Complete configuration options
- **[Building Agents](AGENTS.html)** - Learn advanced agent features
- **[Installation Guide](INSTALLATION.html)** - All installation methods
- **[Examples](https://github.com/kadirpekel/hector/tree/main/configs)** - Sample configurations

**Ready to build?** Start with the [Configuration Reference](CONFIGURATION.html) to unlock Hector's full power.

