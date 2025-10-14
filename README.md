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

Run in **Direct Mode** (no server):

```bash
# Set API key
export OPENAI_API_KEY="sk-..."

# Call agent directly
./hector call assistant "Explain quantum computing" --config my-agent.yaml

# Interactive chat
./hector chat assistant --config my-agent.yaml
```

---

## Core Capabilities

Hector provides a comprehensive AI agent platform through pure YAML configuration:

**Declarative Configuration**
- Zero code required - Define complete systems in YAML
- 6-slot prompt system - Modular prompt composition
- Multiple LLM providers - OpenAI, Anthropic, Gemini
- Structured output - JSON schemas, enums, and constraints
- Environment variables - Secure credential management

**Tools & Integrations**
- Built-in tools - Command execution, file operations (read/write/replace), semantic search, todos
- MCP protocol support - Connect to MCP servers for tool discovery
- Multi-agent orchestration - `agent_call` tool for delegation
- Security controls - Command whitelisting, path restrictions, execution timeouts

**Memory & Context Management**
- Working memory strategies - Buffer window (LIFO) or summary buffer (token-based)
- Long-term memory - Semantic recall via vector storage
- Conversation history - Persistent multi-turn sessions
- Automatic summarization - Threshold-triggered conversation compression

**RAG & Knowledge Base**
- Document stores - Directory-based knowledge indexing
- Vector search - Qdrant integration for semantic retrieval
- Embeddings - Ollama embedder support
- Auto-recall - Inject relevant memories into context

**Reasoning Strategies**
- Chain-of-thought - Iterative reasoning with natural termination
- Supervisor - Optimized for multi-agent orchestration
- Configurable limits - Max iterations, quality thresholds
- Streaming support - Real-time output with thinking blocks

**Multi-Agent System**
- Native agents - Full local control and configuration
- External A2A agents - Connect to remote A2A-compliant services
- LLM-driven routing - Automatic delegation and synthesis
- Visibility control - Public, internal, private access levels

**Plugin Architecture (gRPC)**
- Language-agnostic - Write plugins in any language with gRPC
- Extensible providers - Custom LLMs, databases, embedders, tools
- Process isolation - Plugins run independently for stability
- Auto-discovery - Scan directories for available plugins

**Security & Authentication**
- JWT validation - OAuth2/OIDC with JWKS auto-refresh
- Multiple schemes - Bearer tokens, API keys, Basic auth
- Agent visibility - Control discovery and access
- Tool sandboxing - Restrict commands and file access

**A2A Protocol Compliance**
- Agent cards - Standard capability advertisement
- HTTP+JSON transport - RESTful A2A endpoints with discovery
- Server-Sent Events - Streaming responses per A2A spec
- Task management - Async task lifecycle (create, status, cancel)
- Session support - Context-aware multi-turn conversations

**Deployment & Operations**
- Multi-transport - gRPC, REST gateway, JSON-RPC
- Zero-config mode - Get started with just an API key
- Docker support - Production-ready containerization
- Task persistence - In-memory or SQL backend (PostgreSQL, MySQL, SQLite)

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
