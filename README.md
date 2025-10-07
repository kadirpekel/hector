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
┌─────────────────────────────────────────────────────────────┐
│                        USER / CLIENT                        │
│                  (CLI, HTTP, A2A Protocol)                  │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          │ HTTP+JSON / SSE
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                      A2A INTERFACE                          │
│      GetAgentCard() • ExecuteTask() • Streaming (SSE)       │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                    REASONING ENGINE                         │
│  Chain-of-Thought Strategy    |    Supervisor Strategy      │
│  • Step-by-step reasoning     |    • Multi-agent coord      │
│  • Natural termination        |    • Task decomposition     │
└─────────────────────────┬───────────────────────────────────┘
                          │
      ┌───────────────────┼───────────────────┐
      │                   │                   │
      ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐   ┌──────────────┐
│    TOOLS     │    │     LLM      │   │     RAG      │
│              │    │              │   │              │
│ • Command    │    │ • OpenAI     │   │ • Qdrant     │
│ • File Ops   │    │ • Anthropic  │   │ • Semantic   │
│ • Search     │    │ • Ollama     │   │   Search     │
│ • MCP        │    │ • Plugins    │   │ • Documents  │
└──────────────┘    └──────────────┘   └──────────────┘
```

**Single Agent Capabilities:**
- **6-Slot Prompt System** - Fine-tune role, reasoning, tools, output, style, additional
- **Built-in Tools** - Command execution, file operations, search, todos
- **MCP Integration** - 150+ apps (Composio, Mem0, custom servers)
- **RAG Support** - Semantic search with Qdrant vector database
- **Multi-turn Sessions** - Conversation history and context
- **Real-time Streaming** - Server-Sent Events (SSE) per A2A spec
- **gRPC Plugin System** - Extend with custom LLMs, databases, tools (any language)

---

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

**Multi-Agent Capabilities:**
- **LLM-Driven Orchestration** - No hard-coded workflows, intelligent delegation
- **Heterogeneous Agents** - Mix native (local) and external (remote) seamlessly
- **Transparent Interface** - Same `a2a.Agent` interface for all agent types
- **Agent Discovery** - Automatic capability detection via Agent Cards
- **True Interoperability** - Works with any A2A-compliant agent across organizations

**Key Concepts:**
- **A2A Protocol** - Open standard for agent communication ([specification](https://a2a-protocol.org))
- **Agent Card** - JSON document describing capabilities, endpoints, authentication
- **agent_call Tool** - Built-in tool enabling orchestration by delegating to other agents
- **Supervisor Strategy** - Optimized reasoning for multi-agent coordination

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

## Core Capabilities

Hector provides a comprehensive feature set through pure YAML configuration:

**Declarative Configuration**
- Pure YAML - Zero code for complete agent systems
- 6-slot prompt system - Role, reasoning, tools, output, style, additional
- Environment variables - Secure API key management
- Multiple LLM providers - OpenAI, Anthropic, Ollama

**Tools & Integrations**
- Built-in tools - Command execution, file operations, search, todos
- MCP Protocol - 150+ apps (GitHub, Slack, Gmail, Notion via Composio)
- Custom tools - Add domain-specific capabilities
- Security controls - Command whitelisting, path restrictions, timeouts

**RAG & Knowledge**
- Vector databases - Qdrant, Pinecone, or custom via plugins
- Semantic search - Automatic document retrieval
- Document stores - Organize knowledge by domain
- Embeddings - Ollama or custom embedder plugins

**Sessions & Streaming**
- Multi-turn conversations - Persistent conversation history
- Server-Sent Events - Real-time A2A-compliant streaming
- Session management - Create, list, delete sessions via API
- Context retention - Agent remembers conversation across messages

**Multi-Agent Orchestration**
- LLM-driven routing - Agent decides which specialist to delegate to
- Native + External - Mix local and remote A2A agents
- agent_call tool - Automatic orchestration capability
- Supervisor strategy - Optimized for coordination tasks

**Plugin System (gRPC)**
- Language-agnostic - Write in Go, Python, Rust, JavaScript, etc.
- Custom LLMs - Integrate proprietary models or local inference
- Custom databases - Add specialized vector stores
- Custom embedders - Fine-tuned or domain-specific embeddings
- Process isolation - Plugins run in separate processes for stability

**Security & Deployment**
- JWT Authentication - OAuth2/OIDC integration
- Visibility control - Public, internal, private agents
- Tool security - Whitelisting, sandboxing, resource limits
- Docker support - Production-ready containerization

**A2A Protocol Compliance**
- Agent Cards - Standard capability discovery
- HTTP+JSON transport - RESTful A2A endpoints
- SSE streaming - Real-time output per spec
- Task management - Create, get status, cancel tasks
- Session support - Multi-turn conversations

---

## Detailed Examples

### Single Agent with RAG

```yaml
agents:
  coding_assistant:
    name: "Coding Assistant"
    llm: "claude-3-5-sonnet"
    
    prompt:
      system_role: "Expert software engineer"
      reasoning_instructions: |
        1. Understand requirements fully
        2. Search codebase for patterns
        3. Write clean, testable code
    
    document_stores: ["codebase_docs"]
    tools: [execute_command, file_writer, search]
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 15
      enable_streaming: true

llms:
  claude-3-5-sonnet:
    type: "anthropic"
    model: "claude-3-5-sonnet-20241022"
    api_key: "${ANTHROPIC_API_KEY}"

document_stores:
  codebase_docs:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "codebase"
```

---

### Multi-Agent System

```yaml
agents:
  # Specialized agents
  researcher:
    name: "Research Specialist"
    llm: "gpt-4o"
    document_stores: ["research_db"]
  
  analyst:
    name: "Data Analyst"
    llm: "gpt-4o"
  
  # External A2A agent (just provide URL!)
  translator:
    type: "a2a"
    url: "https://translation-service.com/agents/translator"
  
  # Orchestrator coordinates all
  orchestrator:
    name: "Orchestrator"
    llm: "gpt-4o"
    tools: [agent_call]  # Enable delegation
    reasoning:
      engine: "supervisor"
    prompt:
      system_role: |
        Coordinate specialists: researcher, analyst, translator
        Use agent_call to delegate tasks intelligently
```

**Usage:**
```bash
./hector call orchestrator "Research AI frameworks, analyze top 3, translate summary to Spanish"
```

---

### Custom Plugin Integration

```yaml
# Add custom LLM via gRPC plugin
plugins:
  llm_providers:
    my_custom_llm:
      type: "grpc"
      path: "./plugins/my-llm"
      enabled: true
      config:
        api_key: "${CUSTOM_API_KEY}"

llms:
  custom:
    type: "plugin:my_custom_llm"
    model: "custom-model-v1"

agents:
  my_agent:
    llm: "custom"  # Use plugin
```

---

## CLI Commands

```bash
# Server Commands
hector serve --config FILE [--debug]      # Start A2A server

# Client Commands
hector list [--server URL]                # List available agents
hector call <agent> "prompt" [--stream]   # Call an agent
hector chat <agent>                       # Interactive chat session
hector version                            # Show version

# Environment Variables
export HECTOR_SERVER="http://localhost:8080"
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-..."
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
| **Multi-Agent** | ✅ LLM-driven | ✅ Hard-coded | ✅ Hard-coded | ✅ Hard-coded |
| **Plugins** | ✅ gRPC any language | ⚠️ Python only | ⚠️ Python only | ⚠️ Python only |

**Hector's unique value:**
- **Declarative-first** - Define complete systems in YAML
- **Standards-based** - Built on open A2A protocol
- **True interoperability** - Works with any A2A agent
- **Flexible orchestration** - LLM-driven, not hard-coded workflows
- **Language-agnostic plugins** - Extend in any language via gRPC

---

## Documentation

### Core Guides
- **[Quick Start](docs/QUICK_START.md)** - Get running in 5 minutes
- **[Building Agents](docs/AGENTS.md)** - Complete single-agent guide
- **[Configuration](docs/CONFIGURATION.md)** - Complete config reference
- **[CLI Guide](docs/CLI_GUIDE.md)** - Command-line interface

### Advanced Topics
- **[Multi-Agent Orchestration](docs/ARCHITECTURE.md#orchestrator-pattern)** - Orchestration patterns
- **[External Agents](docs/EXTERNAL_AGENTS.md)** - External agent integration
- **[Tools & MCP](docs/TOOLS.md)** - Built-in tools and MCP protocol
- **[Plugin Development](docs/PLUGINS.md)** - Custom LLMs, databases, tools

### Protocol & Security
- **[A2A Compliance](docs/A2A_COMPLIANCE.md)** - 100% spec compliance details
- **[API Reference](docs/API_REFERENCE.md)** - Complete A2A HTTP/SSE API
- **[Authentication](docs/AUTHENTICATION.md)** - JWT token validation

### Reference
- **[Architecture](docs/ARCHITECTURE.md)** - System design and patterns
- **[Testing Guide](docs/TESTING.md)** - Testing practices

**[📚 Complete Documentation →](docs/)**

---

## Contributing

We welcome contributions! See **[CONTRIBUTING.md](docs/CONTRIBUTING.md)** for:
- Development setup
- Coding standards
- Testing requirements
- Quality checks
- Pull request process

**Quick start:**
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
