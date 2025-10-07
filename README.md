# Hector

```
██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```

**Pure A2A-Native Declarative AI Agent Platform**

[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE.md)
[![A2A Protocol](https://img.shields.io/badge/A2A-compliant-green.svg)](https://a2a-protocol.org)
[![Go Report Card](https://goreportcard.com/badge/github.com/kadirpekel/hector)](https://goreportcard.com/report/github.com/kadirpekel/hector)
[![GoDoc](https://godoc.org/github.com/kadirpekel/hector?status.svg)](https://godoc.org/github.com/kadirpekel/hector)
[![Docker](https://img.shields.io/badge/docker-available-blue.svg)](https://hub.docker.com/r/kadirpekel/hector)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)](https://github.com/kadirpekel/hector/actions)
[![Coverage](https://img.shields.io/badge/coverage-75%25-brightgreen.svg)](https://github.com/kadirpekel/hector/actions)

> **Build powerful AI agents in pure YAML. Compose single agents, orchestrate multi-agent systems, and integrate external A2A agents—all through declarative configuration and industry-standard protocols.**

---

## What is Hector?

Hector is a **declarative AI agent platform** that eliminates code from agent development. Unlike Python-based frameworks (LangChain, AutoGen, CrewAI), Hector uses **pure YAML configuration** to define complete agent systems with:

- **Zero Code Required** - Define agents, tools, prompts, and orchestration in YAML
- **100% A2A Native** - Built on the [Agent-to-Agent protocol](https://a2a-protocol.org) for true interoperability
- **Single & Multi-Agent** - From individual agents to complex orchestration
- **External Integration** - Connect remote A2A agents seamlessly
- **Production Ready** - Authentication, streaming, sessions, monitoring

---

## Architecture

### Single Agent Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Hector Agent                        │
│                                                         │
│  ┌──────────────────────────────────────────────────┐  │
│  │              A2A Interface                       │  │
│  │  GetAgentCard() | ExecuteTask() | Streaming     │  │
│  └─────────────────────┬────────────────────────────┘  │
│                        │                               │
│  ┌─────────────────────▼────────────────────────────┐  │
│  │          Reasoning Engine                       │  │
│  │  • Chain-of-Thought                            │  │
│  │  • Supervisor (Multi-Agent)                    │  │
│  └─────────────────────┬────────────────────────────┘  │
│                        │                               │
│         ┌──────────────┼──────────────┐                │
│         │              │              │                │
│  ┌──────▼─────┐ ┌──────▼─────┐ ┌─────▼──────┐        │
│  │   Tools    │ │    LLM     │ │    RAG     │        │
│  │ • Execute  │ │ • OpenAI   │ │ • Qdrant   │        │
│  │ • File Ops │ │ • Anthropic│ │ • Semantic │        │
│  │ • Search   │ │ • Plugins  │ │   Search   │        │
│  │ • MCP      │ │            │ │            │        │
│  └────────────┘ └────────────┘ └────────────┘        │
└─────────────────────────────────────────────────────────┘
```

**Single Agent Capabilities:**
- ✅ **Custom Prompts** - 6-slot system (role, reasoning, tools, output, style, additional)
- ✅ **Reasoning Strategies** - Chain-of-thought or supervisor modes
- ✅ **Built-in Tools** - Command execution, file operations, search, todos
- ✅ **MCP Integration** - Connect to 150+ apps (Composio, Mem0, custom servers)
- ✅ **RAG Support** - Semantic search with document stores (Qdrant)
- ✅ **Sessions** - Multi-turn conversations with context
- ✅ **Streaming** - Token-by-token output via SSE
- ✅ **Plugin System** - Extend with custom LLMs, databases, tools (gRPC)

---

### Multi-Agent Architecture

```
┌────────────────────────────────────────────────────────────┐
│                    A2A Protocol Layer                      │
│            HTTP+JSON | Sessions | Streaming                │
└───────────────────────┬────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
┌───────▼────────┐ ┌────▼──────┐ ┌─────▼─────────┐
│  Orchestrator  │ │  Native   │ │   External    │
│     Agent      │ │  Agents   │ │  A2A Agents   │
│                │ │           │ │               │
│ • Supervisor   │ │ • Local   │ │ • Remote URL  │
│ • agent_call   │ │ • Full    │ │ • A2A Client  │
│ • Synthesis    │ │   Control │ │ • Transparent │
└───────┬────────┘ └─────┬─────┘ └───────┬───────┘
        │                │                │
        └────────────────┼────────────────┘
                         │
              ┌──────────▼──────────┐
              │  LLM-Driven Routing │
              │  (agent_call tool)  │
              └─────────────────────┘
```

**Multi-Agent Capabilities:**
- ✅ **LLM-Driven Orchestration** - No hard-coded workflows, intelligent delegation
- ✅ **Native + External** - Mix local and remote agents seamlessly
- ✅ **Transparent Interface** - Same `a2a.Agent` interface for all agents
- ✅ **Agent Discovery** - Automatic capability detection via Agent Cards
- ✅ **Ecosystem Ready** - Interoperate across organizations via A2A protocol

**Key Concepts:**
- **A2A Protocol** - Open standard for agent interoperability ([spec](https://a2a-protocol.org))
- **Agent Card** - Describes capabilities, endpoints, authentication
- **Task Model** - Standard request/response with streaming support
- **agent_call Tool** - Enables orchestration by delegating to other agents

---

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

### Create Your First Agent

```yaml
# my-agent.yaml
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

### Run

```bash
# Set API key
export OPENAI_API_KEY="sk-..."

# Start server
./hector serve --config my-agent.yaml

# Chat with agent
./hector chat assistant
```

**That's it!** You have a working AI agent with streaming, sessions, and A2A compliance.

---

## Features

### Declarative Configuration

```yaml
agents:
  coding_assistant:
    name: "Coding Assistant"
    llm: "claude-3-5-sonnet"
    
    # Customize behavior with slot-based prompts
    prompt:
      system_role: "Expert software engineer"
      reasoning_instructions: |
        1. Understand requirements fully
        2. Consider edge cases
        3. Write clean, testable code
    
    # Enable RAG
    document_stores:
      - "codebase_docs"
    
    # Built-in tools
    tools:
      - execute_command
      - file_writer
    
    # Reasoning strategy
    reasoning:
      engine: "chain-of-thought"
      enable_streaming: true
```

---

### Multi-Agent Orchestration

```yaml
agents:
  # Native agents
  researcher:
    llm: "gpt-4o"
    document_stores: ["research_db"]
  
  analyst:
    llm: "gpt-4o"
  
  # External A2A agent (just provide URL!)
  partner_specialist:
    type: "a2a"
    url: "https://partner.com/agents/specialist"
  
  # Orchestrator coordinates them all
  orchestrator:
    llm: "gpt-4o"
    tools:
      - agent_call  # Enable orchestration
    reasoning:
      engine: "supervisor"
```

**Usage:**
```bash
./hector call orchestrator "Research AI frameworks and analyze top 3"
```

The orchestrator automatically delegates to researcher → analyst → synthesis.

---

### A2A Server Mode

Expose your agents via standard A2A protocol:

```bash
# Start server
./hector serve --config agents.yaml

# A2A endpoints available:
# GET  /agents                    → List all agents
# GET  /agents/{id}               → Get agent card
# POST /agents/{id}/message/send  → Execute task
# POST /agents/{id}/message/stream → Streaming execution
# POST /sessions                  → Create session
```

**Any A2A-compliant client can connect:**

```bash
# Using curl
curl http://localhost:8080/agents

# Using Hector CLI
./hector list
./hector call assistant "your prompt"
```

---

## Use Cases

| Scenario | Solution |
|----------|----------|
| **Single Expert Agent** | Define agent with custom prompt, tools, RAG |
| **Multi-Agent Research** | Orchestrator → researchers → analysts → synthesizer |
| **External Integration** | Mix native agents with external A2A services |
| **Agent Marketplace** | Expose agents via A2A for others to consume |
| **CLI Tool** | Use Hector CLI to interact with any A2A server |

---

## CLI Commands

```bash
# Server
hector serve --config FILE [--debug]

# Client
hector list [--server URL]                  # List agents
hector call <agent> "prompt" [--stream]     # Call agent
hector chat <agent>                         # Interactive chat
hector version                              # Show version
```

---

## Why Hector?

| Feature | Hector | LangChain | AutoGen | CrewAI |
|---------|--------|-----------|---------|--------|
| **Configuration** | Pure YAML | Python code | Python code | Python code |
| **A2A Native** | ✅ 100% | ❌ No | ❌ No | ❌ No |
| **External Agents** | ✅ Seamless | ⚠️ Custom | ⚠️ Custom | ❌ No |
| **Zero Code** | ✅ Yes | ❌ No | ❌ No | ❌ No |
| **Interoperability** | ✅ Open protocol | ❌ Proprietary | ❌ Proprietary | ❌ Proprietary |

---

## Examples

### Research Agent

```yaml
agents:
  researcher:
    name: "Research Analyst"
    llm: "gpt-4o"
    prompt:
      system_role: "Thorough research analyst"
      reasoning_instructions: |
        1. Break down research question
        2. Use search to gather information
        3. Cross-reference sources
        4. Synthesize findings
    document_stores:
      - "company_research"
    tools:
      - search
```

### Coding Agent

```yaml
agents:
  coder:
    name: "Coding Assistant"
    llm: "claude-3-5-sonnet"
    prompt:
      system_role: "Expert software engineer"
      tool_usage: |
        - Use search to find patterns in codebase
        - Use file_writer to create/update files
        - Use execute_command to run tests
    document_stores:
      - "codebase_index"
    tools:
      - file_writer
      - execute_command
      - search
```

---

## Documentation

**Core Guides:**
- **[Quick Start](docs/QUICK_START.md)** - Get running in 5 minutes
- **[Building Agents](docs/AGENTS.md)** - Complete single-agent guide
- **[Tools & Extensions](docs/TOOLS.md)** - Built-in tools, MCP, plugins
- **[Configuration](docs/CONFIGURATION.md)** - Complete config reference

**Advanced:**
- **[Multi-Agent Orchestration](docs/ARCHITECTURE.md#orchestrator-pattern)** - Orchestration patterns
- **[External Agents](docs/EXTERNAL_AGENTS.md)** - External agent integration
- **[Authentication](docs/AUTHENTICATION.md)** - JWT token validation
- **[A2A Compliance](docs/A2A_COMPLIANCE.md)** - 100% spec compliance details

**Reference:**
- **[API Reference](docs/API_REFERENCE.md)** - Complete A2A HTTP/SSE API
- **[CLI Guide](docs/CLI_GUIDE.md)** - Command-line interface
- **[Architecture](docs/ARCHITECTURE.md)** - System design
- **[Testing Guide](docs/TESTING.md)** - Testing practices

**[📚 Complete Documentation →](docs/)**

---

## Contributing

We welcome contributions! Please see **[CONTRIBUTING.md](docs/CONTRIBUTING.md)** for:
- Development setup
- Coding standards
- Testing requirements
- Quality checks
- Pull request process

**Quick development workflow:**
```bash
git clone https://github.com/kadirpekel/hector
cd hector
make quality  # Run all checks
```

---

## Project Status

**Current Version**: Alpha

Hector is in active development. While the core functionality is stable and production-ready, APIs may evolve as we refine the platform based on user feedback. We welcome early adopters and contributors!

---

## License

**AGPL-3.0** - See [LICENSE.md](LICENSE.md) for details.

Hector is free and open-source software. You can use, modify, and distribute it under the terms of the AGPL-3.0 license, which requires:
- Source code disclosure for network use
- Same license for derivative works
- Patent grant to users

For commercial licensing options, please contact the maintainers.
