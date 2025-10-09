# Hector Documentation

> **Pure A2A-Native Declarative AI Agent Platform**

Build powerful AI agents in pure YAML. Zero code required.

---

## 🚀 Quick Navigation

### New Users Start Here

1. **[Quick Start](QUICK_START.md)** - Get running in 5 minutes
2. **[🧠 Memory Management](MEMORY.md)** - Never lose context
3. **[Configuration Guide](CONFIGURATION.md)** - Complete YAML reference
4. **[Build a Coding Assistant](tutorials/BUILD_YOUR_OWN_CURSOR.md)** - Real-world tutorial

### Popular Topics

- **[Memory Management](MEMORY.md)** 🔥 - Intelligent context management
- **[Agents Guide](AGENTS.md)** - Build single & multi-agent systems
- **[Tools Integration](TOOLS.md)** - Built-in tools & MCP servers
- **[CLI Guide](CLI_GUIDE.md)** - Command-line interface

---

## 📖 Documentation Index

### Getting Started

| Guide | Description |
|-------|-------------|
| [Quick Start](QUICK_START.md) | Install and run your first agent in 5 minutes |
| [Configuration](CONFIGURATION.md) | Complete YAML configuration reference |
| [CLI Guide](CLI_GUIDE.md) | Command-line interface and usage |
| [API Reference](API_REFERENCE.md) | REST API endpoints and examples |

### Core Features

| Feature | Description |
|---------|-------------|
| **[🧠 Memory Management](MEMORY.md)** | **Intelligent memory with token counting** |
| [Agents](AGENTS.md) | Single & multi-agent systems |
| [Tools](TOOLS.md) | Built-in tools & MCP integration |
| [A2A Protocol](EXTERNAL_AGENTS.md) | External agent integration |
| [Plugins](PLUGINS.md) | gRPC plugin system (any language) |

### Advanced Topics

| Topic | Description |
|-------|-------------|
| [Architecture](ARCHITECTURE.md) | System design and internals |
| [Memory Internals](IMMEDIATE_IMPROVEMENTS_COMPLETED.md) | How Memory Management works |
| [Memory Configuration](MEMORY_CONFIGURATION.md) | Advanced memory options |
| [Testing](TESTING.md) | Testing your agents |
| [Structured Output](STRUCTURED_OUTPUT.md) | JSON/XML/Enum output |
| [Authentication](AUTHENTICATION.md) | Auth0 integration |
| [A2A Compliance](A2A_COMPLIANCE.md) | Protocol implementation details |

### Tutorials

| Tutorial | Description |
|----------|-------------|
| [Build Your Own Cursor](tutorials/BUILD_YOUR_OWN_CURSOR.md) | Complete coding assistant with RAG & reasoning |

---

## 🧠 Featured: Memory Management

**One line of configuration for intelligent context management:**

```yaml
prompt:
  smart_memory: true
```

**What you get:**
- ✅ Accurate token counting (100% accurate, not estimates)
- ✅ Never exceed context limits
- ✅ Intelligent message selection
- ✅ Optional automatic summarization

**Perfect for:**
- Long conversations (50+ messages)
- Code reviews and analysis
- Customer support sessions
- Extended collaborations

👉 **[Read the Complete Guide →](MEMORY.md)**

---

## 🎯 I Want To...

### Build Agents
- **Start simple** → [Quick Start Guide](QUICK_START.md)
- **Enable memory management** → [Memory Management Guide](MEMORY.md) 🔥
- **Add tools** → [Tools Guide](TOOLS.md)
- **Use external agents** → [A2A Integration](EXTERNAL_AGENTS.md)
- **Build coding assistant** → [Tutorial](tutorials/BUILD_YOUR_OWN_CURSOR.md)

### Configure
- **Basic setup** → [Configuration Guide](CONFIGURATION.md)
- **Memory settings** → [Memory Configuration](MEMORY_CONFIGURATION.md)
- **Authentication** → [Auth Guide](AUTHENTICATION.md)
- **Production deployment** → [Architecture](ARCHITECTURE.md)

### Extend
- **Add custom tools** → [Plugins Guide](PLUGINS.md)
- **Build MCP server** → [Tools Guide](TOOLS.md)
- **Custom LLM** → [Plugins Guide](PLUGINS.md)
- **Vector database** → [Architecture](ARCHITECTURE.md)

### Learn
- **How it works** → [Architecture](ARCHITECTURE.md)
- **Memory Management internals** → [Implementation Details](IMMEDIATE_IMPROVEMENTS_COMPLETED.md)
- **A2A Protocol** → [A2A Compliance](A2A_COMPLIANCE.md)
- **Testing** → [Testing Guide](TESTING.md)

---

## 📦 Example Configurations

### Simple Agent with Memory Management

```yaml
agents:
  assistant:
    llm: gpt4o
    prompt:
      smart_memory: true  # One line!
      include_history: true
      system_prompt: You are a helpful assistant.
```

### Multi-Agent System

```yaml
agents:
  orchestrator:
    reasoning:
      engine: supervisor
      orchestrated_agents:
        - researcher
        - writer
        - reviewer
```

### External A2A Agent

```yaml
agents:
  external_specialist:
    type: a2a
    url: https://remote-server.com/agents/specialist
```

---

## 🔗 Quick Links

- **GitHub:** [github.com/kadirpekel/hector](https://github.com/kadirpekel/hector)
- **A2A Protocol:** [a2a-protocol.org](https://a2a-protocol.org)
- **Report Issues:** [GitHub Issues](https://github.com/kadirpekel/hector/issues)
- **Discussions:** [GitHub Discussions](https://github.com/kadirpekel/hector/discussions)

---

## 📝 Recent Updates

### October 2025
- **🧠 Memory Management** - Intelligent context management with accurate token counting
- Conversation summarization for unlimited history
- Smart message selection (preserves important context)
- One-line configuration (`smart_memory: true`)

---

## 🤝 Community

- Ask questions in [GitHub Discussions](https://github.com/kadirpekel/hector/discussions)
- Report bugs in [GitHub Issues](https://github.com/kadirpekel/hector/issues)
- Contribute via [Pull Requests](https://github.com/kadirpekel/hector/pulls)

---

## 📄 License

Hector is licensed under [AGPL-3.0](../LICENSE.md)

---

*Documentation last updated: October 2025*

