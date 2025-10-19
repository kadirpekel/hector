```
██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```

<div align="center">

[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE.md)
[![A2A Protocol](https://img.shields.io/badge/A2A%20Compliance-100%25-brightgreen.svg)](https://gohector.dev/reference/a2a-protocol/)
[![Documentation](https://img.shields.io/badge/docs-gohector.dev-blue.svg)](https://gohector.dev)
[![Go Report Card](https://goreportcard.com/badge/github.com/kadirpekel/hector)](https://goreportcard.com/report/github.com/kadirpekel/hector)

**Build AI agents without code**

A declarative A2A native AI agent platform. Define sophisticated agents through simple YAML configuration.

**Built with Go** for production performance, single-binary deployment, and true portability.

**⚡️ From idea to production agent in minutes, not months.**

[Getting Started](https://gohector.dev/getting-started/installation/) • [Documentation](https://gohector.dev) • [Examples](https://github.com/kadirpekel/hector/tree/main/configs)

</div>

---

> ⚠️ **Alpha Version:** This project is in active development. Features may change as we refine the platform.

---

## Quick Start

```bash
# Install (single binary, no dependencies)
go install github.com/kadirpekel/hector/cmd/hector@latest

# Set API key
export OPENAI_API_KEY="sk-..."

# Run your first agent (zero-config mode)
hector call "What is the capital of France?"
```

> **Why Go?** Unlike Python-based frameworks, Hector compiles to a single binary with no runtime dependencies. Deploy anywhere—from edge devices to Kubernetes—with consistent performance and minimal resource usage.

## Configuration Example

```yaml
agents:
  assistant:
    llm: gpt-4o
    tools: [search, write_file, execute_command]
    reasoning:
      engine: chain-of-thought
      max_iterations: 100
    memory:
      working:
        strategy: summary_buffer
```

That's it. No code required.

---

## Why Hector?

### For Developers

- **Zero-code development** - YAML configuration only
- **Instant setup** - Working agent in 5 minutes
- **Advanced memory** - Working & long-term memory strategies
- **RAG & semantic search** - Built-in vector store integration
- **Rich tool ecosystem** - Built-in tools, MCP, and plugins

### For Enterprises

- **True interoperability** - Native A2A protocol support
- **Multi-agent orchestration** - Coordinate specialized agents
- **Production security** - JWT auth, API keys, agent-level security
- **Distributed architecture** - Local, server, or client modes
- **Multi-transport APIs** - REST, SSE, WebSocket, gRPC

### For Teams

- **Simple configuration** - Human-readable YAML
- **Declarative approach** - No code to maintain
- **Built with Go** - Production performance, single binary, no dependencies
- **Flexible deployment** - Docker, Kubernetes, systemd
- **Extensible platform** - Custom plugins via gRPC
- **Open source** - AGPL-3.0 licensed

---

## Key Features

<table>
<tr>
<td width="50%">

**Agent Development**
- Multiple LLM providers (OpenAI, Anthropic, Gemini)
- Slot-based prompt system
- Structured output (JSON, XML, Enum)
- Working & long-term memory
- Two reasoning strategies
- Session management

</td>
<td width="50%">

**Production Features**
- **Built with Go** - High performance, single binary
- A2A protocol compliant
- Multi-agent orchestration
- Built-in & MCP tools
- RAG with vector databases
- JWT authentication
- Real-time streaming
- Plugin system (gRPC)

</td>
</tr>
</table>

---

## Documentation

📚 **[Complete Documentation at gohector.dev](https://gohector.dev)**

### Quick Links

- **[Getting Started](https://gohector.dev/getting-started/installation/)** - Install and run in 5 minutes
- **[Core Concepts](https://gohector.dev/core-concepts/overview/)** - Understanding agents, LLMs, memory, tools
- **[How-To Guides](https://gohector.dev/how-to/build-coding-assistant/)** - Build real-world agents
- **[CLI Reference](https://gohector.dev/reference/cli/)** - Command-line interface
- **[Configuration](https://gohector.dev/reference/configuration/)** - Complete YAML reference
- **[API Reference](https://gohector.dev/reference/api/)** - REST, gRPC, WebSocket APIs
- **[A2A Protocol](https://gohector.dev/reference/a2a-protocol/)** - Protocol compliance

---

## Architecture

**Hector's layered architecture:**

```
Application (Your Agents)
         ↓
   Hector Runtime
         ↓
    A2A Protocol
         ↓
Multi-Transport Layer (gRPC, REST, JSON-RPC)
         ↓
  Agent Orchestration
         ↓
Core Services (LLM, Memory, Tools, RAG, Tasks)
```

For detailed architecture, see the [Architecture Reference](https://gohector.dev/reference/architecture/).

---

## Deployment Modes

**Three ways to run Hector:**

```bash
# 1. Local Mode - In-process execution
hector call assistant "Hello"

# 2. Server Mode - Host agents for multiple clients
hector serve --config agents.yaml

# 3. Client Mode - Connect to remote agents
hector call assistant "Hello" --server https://remote:8080
```

---

## Multi-Agent Example

**Coordinate specialized agents:**

```yaml
agents:
  coordinator:
    llm: gpt-4o
    reasoning:
      engine: supervisor
    tools:
      - agent_call
    sub_agents:
      - researcher
      - analyst
      - writer
  
  researcher:
    llm: gpt-4o
    tools: [search]
  
  analyst:
    llm: claude-sonnet-4
    tools: [search]
  
  writer:
    llm: gpt-4o
    tools: [write_file]
```

See [Multi-Agent Tutorial](https://gohector.dev/how-to/build-research-system/) for complete example.

---

## A2A Protocol

Hector is **100% compliant** with the [Agent-to-Agent protocol](https://a2a-protocol.org), enabling true interoperability:

- Connect to any A2A-compliant agent
- Expose your agents for others to use
- Federation across organizations
- Standards-based communication

Learn more: [A2A Protocol Reference](https://gohector.dev/reference/a2a-protocol/)

---

## Examples

Check out [`configs/`](https://github.com/kadirpekel/hector/tree/main/configs) for complete examples:

- [Coding Assistant](https://github.com/kadirpekel/hector/blob/main/configs/coding.yaml) - AI pair programmer
- [Multi-Agent Research](https://github.com/kadirpekel/hector/blob/main/configs/orchestrator-example.yaml) - Coordinated research system
- [MCP Tools Integration](https://github.com/kadirpekel/hector/blob/main/configs/tools-mcp-example.yaml) - External tool integration
- [Memory Strategies](https://github.com/kadirpekel/hector/blob/main/configs/memory-strategies-example.yaml) - Memory management
- [Security Setup](https://github.com/kadirpekel/hector/blob/main/configs/security-example.yaml) - JWT authentication

---

## Contributing

We welcome contributions! Hector is in active development and we're building this in the open.

**Ways to contribute:**
- 🐛 Report bugs and issues
- 💡 Suggest features and improvements
- 📖 Improve documentation
- 🔧 Submit pull requests
- ⭐ Star the repo and spread the word

See [Contributing Guidelines](CONTRIBUTING.md) for details.

---

## Community

- **Documentation:** [gohector.dev](https://gohector.dev)
- **Issues:** [GitHub Issues](https://github.com/kadirpekel/hector/issues)
- **Discussions:** [GitHub Discussions](https://github.com/kadirpekel/hector/discussions)

---

## License

**AGPL-3.0** - See [LICENSE.md](LICENSE.md) for details.

Hector is free and open-source software. You can use, modify, and distribute it under the terms of the AGPL-3.0 license.

For commercial licensing options, please contact the maintainers.

---

<div align="center">

**Built with ❤️ by the Hector community**

[![Star on GitHub](https://img.shields.io/github/stars/kadirpekel/hector?style=social)](https://github.com/kadirpekel/hector)

</div>
