```
██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```
[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE.md)
[![A2A Protocol](https://img.shields.io/badge/A2A%20Compliance-100%25-brightgreen.svg)](A2A_COMPLIANCE_SYSTEM.md)
[![Documentation](https://img.shields.io/badge/docs-gohector.dev-blue.svg)](https://gohector.dev)
[![Go Report Card](https://goreportcard.com/badge/github.com/kadirpekel/hector)](https://goreportcard.com/report/github.com/kadirpekel/hector)
[![GoDoc](https://godoc.org/github.com/kadirpekel/hector?status.svg)](https://godoc.org/github.com/kadirpekel/hector)

# Hector
**Declarative AI Agent Platform with Native A2A Protocol Support**

Hector is a **declarative AI agent platform** that eliminates code from agent development. Unlike Python-based frameworks (LangChain, AutoGen, CrewAI), Hector uses **pure YAML configuration** to define complete agent systems:

- **Zero Code Required** - Define agents, tools, prompts, and orchestration entirely in YAML
- **A2A Protocol Native** - Built on the [Agent-to-Agent protocol](https://a2a-protocol.org) for true interoperability
- **Single & Multi-Agent** - From standalone agents to complex orchestration networks
- **Hybrid Architecture** - Mix local agents with remote A2A-compliant services

For complete documentation visit [gohector.dev](https://gohector.dev).

## Quick Start

### Install

```bash
# Clone and build
git clone https://github.com/kadirpekel/hector
cd hector
make build

# Or install as Go package
go install github.com/kadirpekel/hector/cmd/hector@latest
```

**📦 For all installation options (binary releases, Docker, etc.), see [Installation Guide](https://gohector.dev/INSTALLATION.html)**

### Fastest Start - Zero-Config Mode

No configuration file needed!

```bash
# Set API key
export OPENAI_API_KEY="sk-..."

# Start using immediately
./hector call assistant "Explain quantum computing in simple terms"

# Or interactive chat
./hector chat assistant

# With tools enabled
./hector call assistant "List files in current directory" --tools
```

**That's it!** You're up and running with zero configuration.

**📖 See [CLI Guide](https://gohector.dev/CLI_GUIDE.html) for complete command reference and workflows**

### With Config File (For Advanced Features)

Create `my-agent.yaml`:

```yaml
agents:
  assistant:
    name: "My Assistant"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a helpful assistant who explains concepts clearly.

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

Run in **Direct Mode** (in-process, no server):

```bash
# Set API key
export OPENAI_API_KEY="sk-..."

# Call agent directly
./hector call assistant "Explain quantum computing" --config my-agent.yaml

# Interactive chat
./hector chat assistant --config my-agent.yaml
```

**📖 For complete configuration options, see [Configuration Reference](https://gohector.dev/CONFIGURATION.html)**

---

## Why Hector?

Unlike LangChain (500+ lines of Python), Hector uses **pure YAML** (120 lines) for the same functionality.

**Core Capabilities:**
- 🎯 **Zero Code** - Define agents, tools, prompts, and orchestration entirely in YAML
- 🌐 **A2A Protocol Native** - Built on Agent-to-Agent protocol for true interoperability
- 🤖 **Multi-Agent Orchestration** - LLM-driven routing with native & external A2A agents
- 🧠 **Memory Management** - Working memory (token-based) + long-term (vector storage)
- 🛠️ **Tools & MCP** - Built-in tools + MCP protocol for 150+ integrations
- 📚 **RAG & Knowledge** - Vector search (Qdrant), semantic retrieval, document stores
- 🔌 **Plugin System** - gRPC-based extensibility (custom LLMs, databases, tools)
- 🔒 **Production Ready** - JWT auth, streaming (SSE), task persistence (SQL/Redis/Memory)

**📖 Complete documentation:** [gohector.dev](https://gohector.dev)

---

## Architecture

### Agent Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      APPLICATION                             │
│                 (Your Agents & Logic)                        │
├──────────────────────────────────────────────────────────────┤
│                     HECTOR RUNTIME                           │
│  • Configuration Loading  • Agent Initialization             │
│  • Component Management   • Lifecycle Management             │
├──────────────────────────────────────────────────────────────┤
│                      CLIENT LAYER                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │         A2AClient Interface (Protocol Native)           │ │
│  ├─────────────────────────────────────────────────────────┤ │
│  │  HTTPClient           │          DirectClient           │ │
│  │  • Remote agents      │          • In-process agents    │ │
│  │  • Uses protojson     │          • No network calls     │ │
│  │  • Multi-transport    │          • Direct protobuf      │ │
│  └─────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────┤
│                      TRANSPORT LAYER                         │
│  ┌──────────────┬──────────────────┬─────────────────────┐   │
│  │  gRPC (Core) │  REST (Gateway)  │  JSON-RPC (Adapter) │   │
│  │  • Native    │  • Auto-gen      │  • Custom HTTP      │   │
│  │  • Binary    │  • JSON          │  • Simple RPC       │   │
│  │  • Streaming │  • SSE           │  • JSON             │   │
│  │  Port: 8080  │  Port: 8081      │  Port: 8082         │   │
│  └──────────────┴──────────────────┴─────────────────────┘   │
├──────────────────────────────────────────────────────────────┤
│                      SERVER LAYER                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │            RegistryService (Multi-Agent Hub)            │ │
│  │  • Agent registration    • Request routing              │ │
│  │  • Metadata management   • Discovery endpoints          │ │
│  │  • Authentication        • Well-known endpoints         │ │
│  └─────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────┤
│                       AGENT LAYER                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  Agent (pb.A2AServiceServer interface)                  │ │
│  │  • SendMessage          • GetAgentCard                  │ │
│  │  • SendStreamingMessage • GetTask/CancelTask            │ │
│  │  • Task subscriptions   • Push notifications            │ │
│  └─────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────┤
│                    REASONING ENGINE                          │
│  Chain-of-Thought Strategy    |    Supervisor Strategy       │
│  • Step-by-step reasoning     |    • Multi-agent coord       │
│  • Natural termination        |    • Task decomposition      │
├──────────────────────────────────────────────────────────────┤
│                        CORE SERVICES                         │
│  ┌───────────┬──────────┬──────────┬──────────┬──────────┐   │
│  │    LLM    │   Tools  │   Memory │    RAG   │   Tasks  │   │
│  │  • OpenAI │ • Local  │ • Buffer │ • Qdrant │ • Async  │   │
│  │• Anthropic│ • MCP    │ • Summary│ • Search │ • Status │   │
│  │  • Gemini │ • Plugin │ • Session│ • Embed  │ • Track  │   │
│  └───────────┴──────────┴──────────┴──────────┴──────────┘   │
└──────────────────────────────────────────────────────────────┘
```

### Multi-Agent Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        USER / CLIENT                        │
│                  (CLI, HTTP, A2A Protocol)                  │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          │ A2A Protocol (HTTP+JSON/SSE)
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                      A2A SERVER                             │
│         • Discovery (/agents)    • Execution (/tasks)       │
│         • Sessions               • Streaming (SSE)          │
└─────────────────────────┬───────────────────────────────────┘
                          │
      ┌───────────────────┼───────────────────┐
      │                   │                   │
      ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐   ┌──────────────┐
│Orchestrator  │    │   Native     │   │   External   │
│    Agent     │    │   Agents     │   │  A2A Agents  │
│              │    │              │   │              │
│ • Supervisor │    │ • Local      │   │ • Remote URL │
│ • agent_call │    │ • Full Ctrl  │   │ • HTTP Proxy │
│ • Synthesis  │    │              │   │ • Same Iface │
└──────┬───────┘    └──────────────┘   └──────────────┘
       │
       │ LLM-Driven Routing (agent_call tool)
       └──────────────────┐
                          ▼
                  ┌───────────────┐
                  │ Agent Registry│
                  │  (All Agents) │
                  └───────────────┘
```

## Say Hi! to Hector!

![Hector Gopher Logo](gopher.png)

## License

**AGPL-3.0** - See [LICENSE.md](LICENSE.md) for details.

Hector is free and open-source software. You can use, modify, and distribute it under the terms of the AGPL-3.0 license, which requires:
- Source code disclosure for network use
- Same license for derivative works
- Patent grant to users

For commercial licensing options, please contact the maintainers.
