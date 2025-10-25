---
title: Quick Start
description: Get your first Hector agent running in 5 minutes
---

# Quick Start

Get your first agent running in under 5 minutes.

## Prerequisites

✅ Hector installed ([Installation Guide](installation.md))  
✅ API key from [OpenAI](https://platform.openai.com/api-keys), [Anthropic](https://console.anthropic.com/), or [Gemini](https://aistudio.google.com/app/apikey)

---

## Your First Agent (Zero-Config Mode)

The fastest way to start—no configuration file needed!

### 1. Set Your API Key

```bash
export OPENAI_API_KEY="sk-..."
```

### 2. Run Your First Query

```bash
hector call "What is the capital of France?"
```

You should see:
```
The capital of France is Paris.
```

**🎉 Congratulations! Your first agent is working.**

### 3. Try Interactive Chat

```bash
hector chat
```

Type your messages and press Enter. Type `exit` or press Ctrl+C to quit.

### 4. Experiment with Options

```bash
# Use a specific model
hector call "Write a haiku about coding" --model gpt-4o

# Enable built-in tools
hector call "List files in the current directory" --tools

# Use a different provider
export ANTHROPIC_API_KEY="sk-ant-..."
hector call "Explain async/await" --provider anthropic
```

---

## Use a Configuration File

For more control, create a YAML configuration file.

### 1. Create `config.yaml`

```yaml
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7

agents:
  coder:
    name: "Coding Assistant"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are an expert software engineer. Provide clear,
        concise code examples with explanations.
    tools: ["execute_command", "write_file", "search_replace"]
```

### 2. Use Your Configured Agent

```bash
hector call --config config.yaml coder "How do I read a CSV file in Python?"
```

---

## Run a Server

Host agents as a persistent service.

### 1. Start the Server

```bash
# With zero-config
export OPENAI_API_KEY="sk-..."
hector serve

# With configuration file
hector serve --config config.yaml
```

You'll see:
```
Hector server listening on :8080
Registered agents: assistant (or your configured agent names)
```

### 2. Connect from Another Terminal

```bash
# List available agents
hector list

# Call an agent
hector call "Explain recursion" assistant

# Interactive chat
hector chat assistant
```

---

## Connect to a Remote Server

Use Hector as a client to connect to any A2A-compliant server.

```bash
# Connect to remote server
hector call "Hello" assistant --server http://remote:8080

# With authentication
hector call "Hello" assistant --server http://remote:8080 --token "your-jwt-token"
```

---

## Common Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `hector version` | Show version | `hector version` |
| `hector call` | Send a single message | `hector call "Hello"` (zero-config) |
| `hector chat` | Interactive conversation | `hector chat` (zero-config) |
| `hector serve` | Start server | `hector serve --config config.yaml` |
| `hector list` | List available agents | `hector list` |

---

## Next Steps

Now that you have Hector running, learn more:

- **[Core Concepts](../core-concepts/overview.md)** - Understand how agents work
- **[Build a Coding Assistant](../how-to/build-coding-assistant.md)** - Complete tutorial with semantic search
- **[Configuration Reference](../reference/configuration.md)** - All configuration options
- **[CLI Reference](../reference/cli.md)** - All command-line options

---

## Troubleshooting

**Agent not responding?**

- Check your API key is set: `echo $OPENAI_API_KEY`
- Verify installation: `hector version`
- Check network connectivity to LLM provider

**"command not found: hector"**

- Ensure `/usr/local/bin` is in your PATH
- Or run from installation directory: `./hector call "Hello"`

**Authentication errors?**

- Verify API key is valid
- Check for typos in environment variable
- Ensure provider matches key (OpenAI, Anthropic, Gemini)

For more help, see [CLI Reference](../reference/cli.md) or [Configuration Reference](../reference/configuration.md).

