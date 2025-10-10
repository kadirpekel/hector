---
layout: default
title: Getting Started
nav_order: 10
has_children: true
description: "Everything you need to get started with Hector"
---

# Getting Started

New to Hector? Start here! These guides will get you up and running quickly.

## What You'll Learn

- **[Zero-Config Mode](ZERO_CONFIG_MODE)** - Start instantly without any configuration! 🚀
- **[Quick Start](QUICK_START)** - Install and run your first agent in 5 minutes
- **[CLI Guide](CLI_GUIDE)** - Master the command-line interface
- **[Your First Agent](AGENTS#your-first-agent)** - Build a custom agent step-by-step
- **[Basic Configuration](CONFIGURATION#basic-setup)** - Essential configuration options

## Two Ways to Start

### ⚡ Fastest: Zero-Config Mode

No configuration file needed!

```bash
# Set API key
export OPENAI_API_KEY="sk-..."

# Start using immediately
hector call assistant "Explain quantum computing"
hector chat assistant
```

**Perfect for:**
- Quick experimentation
- Learning Hector
- Simple use cases
- CI/CD integration

**[→ See Zero-Config Mode Guide](ZERO_CONFIG_MODE)**

### 📝 Full Configuration

Create a `hector.yaml` for advanced features:

```yaml
agents:
  assistant:
    name: "My Assistant"
    llm: "gpt-4o"
    prompt:
      system_role: "You are a helpful assistant"

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

**Unlocks:**
- Custom prompts
- Multiple agents
- RAG (document stores)
- Advanced memory
- Multi-agent orchestration

**[→ See Quick Start Guide](QUICK_START)**

---

## Dual-Mode Architecture

Hector has two modes of operation:

### 🚀 Direct Mode (Default)

Agent runs in-process - no server needed.

```bash
hector call assistant "hello"
hector chat assistant
```

### 🌐 Server Mode

Runs A2A protocol server for multi-agent systems.

```bash
# Terminal 1: Start server
hector serve

# Terminal 2: Connect
hector call assistant "hello" --server http://localhost:8080
```

**[→ See CLI Guide](CLI_GUIDE) for complete documentation**

---

## Next Steps

Once you're comfortable with the basics:
- Try our **[Tutorials](tutorials/)** for hands-on learning
- Explore **[Core Guides](#)** for deeper understanding
- Check out **[Examples](https://github.com/kadirpekel/hector/tree/main/configs)** in the repository
