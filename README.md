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
**Pure A2A-Native Declarative AI Agent Platform**

Hector is a **declarative AI agent platform** that eliminates code from agent development. Unlike Python-based frameworks (LangChain, AutoGen, CrewAI), Hector uses **pure YAML configuration** to define complete agent systems with:

- **Zero Code Required** - Define agents, tools, prompts, and orchestration in YAML
- **100% A2A Native** - Built on the [Agent-to-Agent protocol](https://a2a-protocol.org) for true interoperability
- **Single & Multi-Agent** - From individual agents to complex orchestration
- **External Integration** - Connect remote A2A agents seamlessly

For complete documentation please visit [gohector.dev](https://gohector.dev).

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
│  ┌─────────────────────────────────────────────────────────┐│
│  │         A2AClient Interface (Protocol Native)           ││
│  ├─────────────────────────────────────────────────────────┤│
│  │  HTTPClient           │          DirectClient           ││
│  │  • Remote agents      │          • In-process agents    ││
│  │  • Uses protojson     │          • No network calls     ││
│  │  • Multi-transport    │          • Direct protobuf      ││
│  └─────────────────────────────────────────────────────────┘│
├──────────────────────────────────────────────────────────────┤
│                      TRANSPORT LAYER                         │
│  ┌──────────────┬──────────────────┬─────────────────────┐ │
│  │  gRPC (Core) │  REST (Gateway)  │  JSON-RPC (Adapter) │ │
│  │  • Native    │  • Auto-gen      │  • Custom HTTP      │ │
│  │  • Binary    │  • JSON          │  • Simple RPC       │ │
│  │  • Streaming │  • SSE           │  • JSON             │ │
│  │  Port: 8080  │  Port: 8081      │  Port: 8082         │ │
│  └──────────────┴──────────────────┴─────────────────────┘ │
├──────────────────────────────────────────────────────────────┤
│                      SERVER LAYER                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │            RegistryService (Multi-Agent Hub)            ││
│  │  • Agent registration    • Request routing              ││
│  │  • Metadata management   • Discovery endpoints          ││
│  │  • Authentication        • Well-known endpoints         ││
│  └─────────────────────────────────────────────────────────┘│
├──────────────────────────────────────────────────────────────┤
│                       AGENT LAYER                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  Agent (pb.A2AServiceServer interface)                  ││
│  │  • SendMessage          • GetAgentCard                  ││
│  │  • SendStreamingMessage • GetTask/CancelTask            ││
│  │  • Task subscriptions   • Push notifications            ││
│  └─────────────────────────────────────────────────────────┘│
├──────────────────────────────────────────────────────────────┤
│                    REASONING ENGINE                          │
│  Chain-of-Thought Strategy    |    Supervisor Strategy       │
│  • Step-by-step reasoning     |    • Multi-agent coord       │
│  • Natural termination        |    • Task decomposition      │
├──────────────────────────────────────────────────────────────┤
│                        CORE SERVICES                         │
│  ┌───────────┬──────────┬──────────┬──────────┬──────────┐ │
│  │    LLM    │   Tools  │   Memory │    RAG   │   Tasks  │ │
│  │  • OpenAI │ • Local  │ • Buffer │ • Qdrant │ • Async  │ │
│  │• Anthropic│ • MCP    │ • Summary│ • Search │ • Status │ │
│  │  • Gemini │ • Plugin │ • Session│ • Embed  │ • Track  │ │
│  └───────────┴──────────┴──────────┴──────────┴──────────┘ │
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
