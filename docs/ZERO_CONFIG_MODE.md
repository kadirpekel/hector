---
layout: default
title: Zero-Config Mode
nav_order: 2
parent: Getting Started
description: "Get started instantly with zero-config mode - no configuration file needed"
---

# Zero-Config Mode

Get started with Hector in seconds without creating any configuration file!

## Overview

Zero-config mode lets you use Hector immediately by specifying options via CLI flags. Perfect for:
- **Quick experimentation**  
- **Learning Hector**  
- **Simple use cases**  
- **CI/CD testing**

##  Modes of Operation

Hector has **two modes of operation**:

### 🚀 **Direct Mode** (Default)

Agent runs in-process without starting an A2A server.

```bash
# Call agent directly
hector call assistant "hello"

# Interactive chat
hector chat assistant

# List agents from config
hector list
```

**When to use:**
- Quick experimentation
- Single-agent workflows
- Local development
- CI/CD scripts

###  **Server Mode** (with `--server` flag)

Connects to an A2A protocol server (local or remote).

```bash
# Start server
hector serve

# In another terminal - connect to server
hector call --server http://localhost:8080 assistant "hello"
hector chat --server http://localhost:8080 assistant
hector list --server http://localhost:8080
```

**When to use:**
- Multi-agent systems
- Production deployments
- Remote agents
- Distributed systems

---

## Zero-Config Quick Start

### Direct Mode (Simplest)

```bash
# Set your API key
export OPENAI_API_KEY="sk-..."

# Call agent directly (no config file needed!)
hector call assistant "Explain quantum computing"

# Interactive chat
hector chat assistant

# Customize the model
hector call assistant "Write a poem" --model gpt-4o

# Enable tools (file operations, command execution)
hector call assistant "List files in current directory" --tools
```

**That's it!** No configuration file, no setup, just instant AI assistance.

### Server Mode (For Multi-Agent)

```bash
# Terminal 1: Start server with zero-config
export OPENAI_API_KEY="sk-..."
hector serve --api-key $OPENAI_API_KEY

# Terminal 2: Connect to server
hector call --server http://localhost:8080 assistant "hello"
hector chat --server http://localhost:8080 assistant
```

---

## Quick Flags Reference

### Core Flags (Both Modes)

```bash
--api-key KEY           OpenAI API key (or set OPENAI_API_KEY)
--base-url URL          OpenAI API base URL (default: https://api.openai.com/v1)
--model NAME            Model to use (default: gpt-4o-mini)
--tools                 Enable all local tools (file, command, search)
```

### Advanced Flags

```bash
--mcp URL               MCP server URL for tool integration
--docs FOLDER           Document store folder (enables RAG)
--config FILE           Use config file instead of zero-config
--debug                 Enable debug output
```

### Mode Selection

```bash
--server URL            Connect to A2A server (enables server mode)
                        If not specified, uses direct mode
```

### Environment Variables

Hector automatically loads `.env` files from your current directory for **all commands** (not just `hector serve`):

```bash
# Create .env file
echo "OPENAI_API_KEY=sk-..." > .env

# Now all commands work without --api-key flag
hector call assistant "hello"                    # Direct mode
hector chat assistant                             # Direct mode
hector serve                                      # Server mode
hector call --server http://localhost:8080 assistant "hello"  # Server mode
```

**Supported formats:**
- `.env` - Standard environment file
- `.env.local` - Local overrides (gitignored by default)

**Priority:** CLI flags > `.env` file > environment variables

---

## Examples

### Basic Usage

```bash
# Simple question
hector call assistant "What is machine learning?"

# Custom model
hector call assistant "Explain neural networks" --model gpt-4o

# From environment variable
export OPENAI_API_KEY="sk-..."
hector chat assistant
```

### With Tools

```bash
# Enable local tools
hector call assistant "Count files in this directory" --tools

# The agent can now use:
# - execute_command (run shell commands)
# - write_file (create/edit files)
# - search_replace (find and replace in files)
```

### With MCP Integration

```bash
# Start MCP server (separate terminal)
python my-mcp-server.py --port 3000

# Use with Hector
hector call assistant "Search web for AI news" \
  --mcp http://localhost:3000 \
  --tools
```

### With Document Store (RAG)

```bash
# Create document folder
mkdir docs
echo "Product: Hector\nDescription: AI agent platform" > docs/product.txt

# Use with RAG
hector call assistant "What is our product?" \
  --docs ./docs
```

### Server Mode

```bash
# Terminal 1: Start server
hector serve \
  --api-key $OPENAI_API_KEY \
  --model gpt-4o-mini \
  --tools

# Terminal 2: Connect
hector call assistant "hello" --server http://localhost:8080
hector chat assistant --server http://localhost:8080
```

---

## Configuration Precedence

Zero-config mode follows this precedence (highest to lowest):

1. **CLI flags** - Always take priority
2. **Environment variables** - Used if flag not provided
3. **Defaults** - Sensible defaults for quick start

### API Key Resolution

```bash
# Method 1: CLI flag (highest priority)
hector call assistant "hello" --api-key sk-abc123

# Method 2: Environment variable
export OPENAI_API_KEY="sk-abc123"
hector call assistant "hello"

# Method 3: Error if neither provided
# ❌ Error: OpenAI API key required for zero-config mode
```

### Model Selection

```bash
# Default: gpt-4o-mini
hector call assistant "hello"

# Custom via flag
hector call assistant "hello" --model gpt-4o

# Custom via environment (if supported)
export HECTOR_MODEL="gpt-4o"
hector call assistant "hello"
```

---

## Direct Mode vs Server Mode

| Feature | Direct Mode | Server Mode |
|---------|-------------|-------------|
| **Setup** | None | Start server first |
| **Command** | `hector call assistant "..."` | `hector call assistant "..." --server URL` |
| **Use Case** | Quick tasks, single agent | Multi-agent, production |
| **Performance** | Faster (no HTTP) | Slightly slower (HTTP) |
| **Scalability** | Single process | Distributed |
| **Best For** | Experimentation, dev | Production, coordination |

---

## Transitioning to Config Files

When your needs grow, transition to a config file:

### 1. Extract Current Settings

```bash
# Current zero-config usage
hector call assistant "hello" \
  --model gpt-4o \
  --tools

# Equivalent hector.yaml
```

```yaml
agents:
  assistant:
    name: "AI Assistant"
    llm: "gpt-4o"

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"

# Tools enabled by default for all agents
```

### 2. Use Config File

```bash
# Now use config file
hector serve --config hector.yaml

# Or in direct mode
hector call assistant "hello" --config hector.yaml
```

### 3. Advanced Features

Config files unlock advanced features:

```yaml
agents:
  assistant:
    name: "Advanced Assistant"
    llm: "gpt-4o"
    
    # Custom prompts
    prompt:
      system_role: |
        You are a helpful assistant with expertise in...
      reasoning_instructions: |
        1. Think step by step
        2. Verify your reasoning
        3. Provide clear explanations
    
    # Document stores (RAG)
    document_stores:
      - "company_docs"
    
    # Memory configuration
    memory:
      working_memory:
        strategy: "token_based"
        max_tokens: 4000

document_stores:
  company_docs:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "docs"
```

---

## Limitations

Zero-config mode has some limitations:

| Feature | Zero-Config | Config File |
|---------|-------------|-------------|
| **Single Agent** | ✅ Yes | ✅ Yes |
| **Custom Prompts** | ❌ No | ✅ Yes |
| **Multiple Agents** | ❌ No | ✅ Yes |
| **RAG (Basic)** | ⚠️ Limited | ✅ Full |
| **Memory Config** | ⚠️ Default | ✅ Custom |
| **Multi-LLM** | ❌ No | ✅ Yes |
| **Authentication** | ❌ No | ✅ Yes |

**When to use config files:**
- Multiple agents
- Custom prompts
- Advanced RAG
- Production deployments
- Multi-LLM setups

---

## Troubleshooting

### API Key Not Found

```bash
# ❌ Error: OpenAI API key required for zero-config mode

# ✅ Fix: Set environment variable
export OPENAI_API_KEY="sk-..."

# Or use flag
hector call assistant "hello" --api-key sk-...
```

### Invalid Model

```bash
# ❌ Error: Model not found: gpt-5

# ✅ Fix: Use valid model
hector call assistant "hello" --model gpt-4o-mini
```

### Tools Not Working

```bash
# ❌ Agent doesn't execute commands

# ✅ Fix: Enable tools flag
hector call assistant "list files" --tools
```

### Flags After Agent Name Don't Work

```bash
# ❌ Wrong: Flags after agent name are ignored
hector call assistant "hello" --server http://localhost:8080
hector chat assistant --model gpt-4o

# ✅ Correct: Flags must come BEFORE positional arguments
hector call --server http://localhost:8080 assistant "hello"
hector chat --model gpt-4o assistant
```

**Why:** This is standard Go CLI behavior. Flags must precede positional arguments (agent name, prompt).

### Server Connection Failed

```bash
# ❌ Error: Connection refused

# ✅ Fix: Start server first
# Terminal 1:
hector serve --api-key $OPENAI_API_KEY

# Terminal 2:
hector call assistant "hello" --server http://localhost:8080
```

---

## Advanced Usage

### Scripting

```bash
#!/bin/bash
# automated-research.sh

export OPENAI_API_KEY="sk-..."

# Series of queries
TOPICS=("AI" "ML" "DL")

for topic in "${TOPICS[@]}"; do
  echo "Researching: $topic"
  hector call assistant "Explain $topic in one paragraph" --model gpt-4o > "${topic}.txt"
done
```

### CI/CD Integration

```bash
# .github/workflows/ai-review.yml
- name: AI Code Review
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
  run: |
    hector call assistant "Review this PR for security issues" \
      --tools \
      --model gpt-4o > review.txt
```

### Environment Profiles

```bash
# .env.dev
OPENAI_API_KEY=sk-dev-key
HECTOR_MODEL=gpt-4o-mini

# .env.prod
OPENAI_API_KEY=sk-prod-key
HECTOR_MODEL=gpt-4o

# Usage
source .env.dev
hector call assistant "test"
```

---

## Next Steps

- **[CLI Guide](CLI_GUIDE)** - Complete command reference
- **[Quick Start](QUICK_START)** - 5-minute tutorial
- **[Configuration](CONFIGURATION)** - Full config file reference
- **[Building Agents](AGENTS)** - Advanced agent features

**Ready for more?** Try our [tutorials](tutorials/) for hands-on learning!

