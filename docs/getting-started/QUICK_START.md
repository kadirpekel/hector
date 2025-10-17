---
layout: default
title: Quick Start
nav_order: 2
parent: Getting Started
description: "Get up and running with Hector in 5 minutes"
---

# Hector Quick Start

Get up and running with Hector in 5 minutes.

## Prerequisites

- **Hector installed** - See [Installation Guide](INSTALLATION)
- **API Key** - From OpenAI, Anthropic, or Gemini

---

## Local Mode: Quick Experimentation

Run agents locally without a server. Perfect for testing and development.

### 1. Set API Key

```bash
export OPENAI_API_KEY="sk-..."
```

### 2. Run Immediately (Zero-Config)

```bash
# Single query (agent name optional!)
hector call "Explain quantum computing"

# Interactive chat (agent name optional!)
hector chat

# Custom model
hector call "Write a haiku" --model gpt-4o

# With tools enabled
hector call "List files" --tools

# Local mode (no agent name needed)
hector call "Explain quantum computing"
hector chat
```

**That's it!** No configuration file needed.

### 3. With Configuration File (Optional)

For custom prompts and behavior, create `config.yaml`:

```yaml
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"

agents:
  assistant:
    name: "My Assistant"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a helpful assistant. Be concise and clear.
```

Run with config:

```bash
hector call assistant "Hello" --config config.yaml
```

---

## Server Mode: Persistent Service

Run a server to host agents for multiple clients.

### 1. Start Server

```bash
# With zero-config
export OPENAI_API_KEY="sk-..."
hector serve

# With configuration file
hector serve --config config.yaml
```

### 2. Use Agents

In another terminal:

```bash
# List agents
hector list

# Call agent
hector call assistant "Explain AI"

# Interactive chat
hector chat assistant
```

---

## Client Mode: Connect to Remote Server

Connect to an existing Hector server (or any A2A-compliant server).

```bash
# Connect to remote server
hector call assistant "Hello" --server http://remote:8080

# With authentication
hector call assistant "Hello" --server http://remote:8080 --token "your-token"

# Interactive chat
hector chat assistant --server http://remote:8080
```

---

## What's Next?

**[CLI Guide](CLI_GUIDE)** - Complete command reference and workflows  
**[Configuration Reference](CONFIGURATION)** - Full YAML configuration options  
**[Building Agents](AGENTS)** - Learn to build sophisticated agents  
