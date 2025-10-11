```
██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```
# Hector
**Pure A2A-Native Declarative AI Agent Platform**

[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE.md)
[![A2A Protocol](https://img.shields.io/badge/A2A-compliant-green.svg)](https://a2a-protocol.org)
[![Documentation](https://img.shields.io/badge/docs-gohector.dev-blue.svg)](https://gohector.dev)
[![Go Report Card](https://goreportcard.com/badge/github.com/kadirpekel/hector)](https://goreportcard.com/report/github.com/kadirpekel/hector)
[![GoDoc](https://godoc.org/github.com/kadirpekel/hector?status.svg)](https://godoc.org/github.com/kadirpekel/hector)

> **Build powerful AI agents in pure YAML. Compose single agents, orchestrate multi-agent systems, and integrate external A2A agents—all through declarative configuration and industry-standard protocols.**

📖 **[Complete Documentation & Tutorials → gohector.dev](https://gohector.dev)**

| Quick Links | |
|:------------|:--|
| 🚀 **[Get Started](https://gohector.dev/QUICK_START)** | Install and run your first agent in 5 minutes |
| 🔥 **[LangChain vs Hector](https://gohector.dev/tutorials/MULTI_AGENT_RESEARCH_PIPELINE)** | See 500+ lines of Python become 120 lines of YAML |
| 🤖 **[Build Cursor-like Assistant](https://gohector.dev/tutorials/BUILD_YOUR_OWN_CURSOR)** | Create an AI coding assistant in pure YAML |
| ⚡ **[Custom MCP Tools](https://gohector.dev/MCP_CUSTOM_TOOLS)** | Build custom tools in 5 minutes |

---

## What is Hector?

Hector is a **declarative AI agent platform** that eliminates code from agent development. Unlike Python-based frameworks (LangChain, AutoGen, CrewAI), Hector uses **pure YAML configuration** to define complete agent systems with:

- **Zero Code Required** - Define agents, tools, prompts, and orchestration in YAML
- **100% A2A Native** - Built on the [Agent-to-Agent protocol](https://a2a-protocol.org) for true interoperability
- **Single & Multi-Agent** - From individual agents to complex orchestration
- **External Integration** - Connect remote A2A agents seamlessly
- **Production Ready** - Authentication, streaming, sessions, monitoring

**Want to see what's possible?** Check out our [**LangChain vs Hector Tutorial**](https://gohector.dev/tutorials/MULTI_AGENT_RESEARCH_PIPELINE) to see how Hector transforms complex multi-agent implementations into simple YAML configuration, or try our [**Coding Assistant Tutorial**](https://gohector.dev/tutorials/BUILD_YOUR_OWN_CURSOR) to build a Cursor-like AI coding assistant—all through pure YAML configuration.

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
      ┌───────────────────┼───────────────────┬────────────────┐
      │                   │                   │                │
      ▼                   ▼                   ▼                ▼
┌──────────────┐    ┌──────────────┐   ┌──────────────┐  ┌────────────┐
│    TOOLS     │    │     LLM      │   │     RAG      │  │   MEMORY   │
│              │    │              │   │              │  │            │
│ • Command    │    │ • OpenAI     │   │ • Qdrant     │  │ • Working  │
│ • File Ops   │    │ • Anthropic  │   │ • Semantic   │  │   (Session)│
│ • Search     │    │ • Gemini     │   │   Search     │  │ • Long-term│
│ • MCP        │    │ • Plugins    │   │ • Documents  │  │            │
└──────────────┘    └──────────────┘   └──────────────┘  └────────────┘
```

**Single Agent Capabilities:**
- **6-Slot Prompt System** - Fine-tune role, reasoning, tools, output, style, additional
- **Built-in Tools** - Command execution, file operations, search, todos
- **MCP Integration** - 150+ apps (Composio, Mem0, custom servers)
- **RAG Support** - Semantic search with Qdrant vector database
- **Intelligent Memory** - Pluggable working memory strategies (token-based with summarization, simple LIFO) for conversation history ([docs](docs/MEMORY.md))
- **Multi-turn Sessions** - Conversation context and history management
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

Or run in **Server Mode** (for multi-agent):

```bash
# Terminal 1: Start server
./hector serve --config my-agent.yaml

# Terminal 2: Connect to server
./hector chat --server http://localhost:8080 assistant
./hector call --server http://localhost:8080 assistant "Explain quantum computing"
./hector list --server http://localhost:8080
```

**Now you have:**
- ✅ Streaming output (enabled by default)
- ✅ Multi-turn sessions  
- ✅ A2A protocol compliance
- ✅ Direct or Server mode
- ✅ Zero-config or full configuration

---

## Core Capabilities

Hector provides a comprehensive feature set through pure YAML configuration:

**Declarative Configuration**
- Pure YAML - Zero code for complete agent systems
- 6-slot prompt system - Role, reasoning, tools, output, style, additional
- Environment variables - Secure API key management
- Multiple LLM providers - OpenAI, Anthropic, Gemini

**Tools & Integrations**
- Built-in tools - Command execution, file operations, search, todos
- MCP Protocol - 150+ apps (GitHub, Slack, Gmail, Notion via Composio)
- **Custom MCP tools** - Build your own in 5 minutes (Python/TypeScript) 🔥
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

**🏆 Deep A2A Native Integration**
- **Native Message Types** - Uses `a2a.Message` with rich content (`Parts[]`) throughout
- **Native Tool Calls** - Uses `a2a.ToolCall` for seamless tool integration
- **Native Task Management** - Uses `a2a.Task` for complete protocol compliance
- **Zero Abstraction Layers** - No custom message types or compatibility layers
- **True Protocol Compliance** - Every component uses A2A types directly
- **Future-Ready** - Ready for multi-modal content, files, and rich data

📖 **[Deep A2A Native Architecture →](https://gohector.dev/A2A_NATIVE_ARCHITECTURE)** - Technical deep dive into native A2A implementation

---

## Extend with Custom Tools in 5 Minutes

Need domain-specific capabilities? Build a custom MCP server in minutes:

```python
# my_tools.py
from mcp.server import Server
import requests

app = Server("my-tools")

@app.tool()
async def web_search(query: str, num_results: int = 5) -> str:
    """Search the web"""
    # Your implementation using any API
    response = requests.get(f"https://api.search.com?q={query}")
    return format_results(response.json())

@app.tool()
async def get_weather(city: str) -> str:
    """Get current weather"""
    response = requests.get(f"https://api.weather.com?city={city}")
    data = response.json()
    return f"Weather in {city}: {data['description']}, {data['temp']}°C"

if __name__ == "__main__":
    app.run(port=3000)
```

**Configure Hector:**
```yaml
tools:
  my_tools:
    type: "mcp"
    enabled: true
    server_url: "http://localhost:3000"

agents:
  assistant:
    name: "Assistant with Custom Tools"
    llm: "gpt-4o"
    # Agent automatically gets web_search and get_weather tools!
```

**Use it:**
```bash
./hector call assistant "Search for 'AI frameworks' and tell me the weather in Tokyo"
```

**That's it!** No Hector code changes, just pure Python/TypeScript.

**[📖 Full Guide: Building Custom MCP Tools →](docs/MCP_CUSTOM_TOOLS.md)**

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
    tools: [execute_command, write_file, search]
    
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
# Start the multi-agent server
./hector serve --config multi-agent.yaml

# Call the orchestrator (shorthand notation)
./hector call orchestrator "Research AI frameworks, analyze top 3, translate summary to Spanish"

# Or use interactive chat
./hector chat orchestrator
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

Hector has **two modes**: **Direct Mode** (in-process) and **Server Mode** (A2A protocol).

### Direct Mode (Default)

No server required - agent runs in-process:

```bash
# Zero-Config Mode (no configuration file needed!)
export OPENAI_API_KEY="sk-..."
hector call assistant "Explain quantum computing"
hector chat assistant
hector list

# With Quick Flags
hector call assistant "hello" --model gpt-4o --tools
hector chat assistant --api-key sk-... --model gpt-4o-mini

# With Config File
hector call assistant "hello" --config my-agent.yaml
hector chat assistant --config my-agent.yaml
```

### Server Mode

Start A2A server and connect to it:

```bash
# Terminal 1: Start server
hector serve --config FILE [--debug]

# Or zero-config server
hector serve --api-key $OPENAI_API_KEY --tools

# Terminal 2: Connect to server
hector list --server http://localhost:8080
hector call --server http://localhost:8080 assistant "prompt"
hector chat --server http://localhost:8080 assistant
hector info --server http://localhost:8080 assistant
```

### Zero-Config Quick Flags

```bash
# Core flags
--api-key KEY          OpenAI API key (or set OPENAI_API_KEY)
--base-url URL         OpenAI API base URL
--model NAME           Model (default: gpt-4o-mini)
--tools                Enable all local tools
--mcp URL              MCP server URL
--docs FOLDER          Document store folder (RAG)

# Mode selection
--server URL           Connect to A2A server (enables server mode)
                       Without this flag, uses direct mode

# Other
--config FILE          Use config file
--debug                Enable debug output
--stream               Enable streaming (default: true)
--token TOKEN          Authentication token
```

**Note:** Flags must come **before** positional arguments (agent name, prompt):
```bash
# ✅ Correct
hector call --server http://localhost:8080 assistant "hello"

# ❌ Wrong (flags after agent name are ignored)
hector call assistant "hello" --server http://localhost:8080
```

### Examples

```bash
# Quick start - no config needed
export OPENAI_API_KEY="sk-..."
hector call assistant "hello"

# With tools
hector call assistant "list files" --tools

# Custom model
hector call assistant "write code" --model gpt-4o

# Server mode
hector serve --api-key $OPENAI_API_KEY --tools
hector call assistant "hello" --server http://localhost:8080

# Help & Version
hector help
hector version
```

**📚 See [Zero-Config Mode Guide](https://gohector.dev/ZERO_CONFIG_MODE) for complete documentation.**

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

📖 **[Complete Documentation Site → gohector.dev](https://gohector.dev)**

### 🎯 Featured Tutorials
- **[LangChain vs Hector: Multi-Agent Systems](https://gohector.dev/tutorials/MULTI_AGENT_RESEARCH_PIPELINE)** - See how Hector transforms complex LangChain multi-agent implementations into simple YAML configuration. What takes 500+ lines of Python code becomes 120 lines of YAML
- **[Build Your Own Cursor-Like AI Coding Assistant](https://gohector.dev/tutorials/BUILD_YOUR_OWN_CURSOR)** - Step-by-step tutorial showing how to create a powerful AI coding assistant using pure YAML configuration

### 📚 Core Guides  
- **[Quick Start](https://gohector.dev/QUICK_START)** - Get running in 5 minutes
- **[Building Agents](https://gohector.dev/AGENTS)** - Complete single-agent guide  
- **[Memory Management](https://gohector.dev/MEMORY)** - Intelligent context management
- **[Configuration](https://gohector.dev/CONFIGURATION)** - Complete config reference
- **[CLI Guide](https://gohector.dev/CLI_GUIDE)** - Command-line interface

### 🚀 Advanced Topics
- **[Multi-Agent Orchestration](https://gohector.dev/ARCHITECTURE#multi-agent-orchestration-a2a-protocol)** - Orchestration patterns
- **[External Agents](https://gohector.dev/EXTERNAL_AGENTS)** - External agent integration
- **[Tools & MCP](https://gohector.dev/TOOLS)** - Built-in tools and MCP protocol
- **[Custom MCP Tools](https://gohector.dev/MCP_CUSTOM_TOOLS)** - Build custom tools in 5 minutes 🔥
- **[Memory Configuration](https://gohector.dev/MEMORY_CONFIGURATION)** - Advanced memory options
- **[Structured Output](https://gohector.dev/STRUCTURED_OUTPUT)** - Provider-aware JSON/XML/Enum output
- **[Plugin Development](https://gohector.dev/PLUGINS)** - Custom LLMs, databases, tools

### 🔒 Protocol & Security
- **[A2A Compliance](https://gohector.dev/A2A_COMPLIANCE)** - 100% spec compliance details
- **[API Reference](https://gohector.dev/API_REFERENCE)** - Complete A2A HTTP/SSE API
- **[Authentication](https://gohector.dev/AUTHENTICATION)** - JWT token validation

### 📖 Reference
- **[Architecture](https://gohector.dev/ARCHITECTURE)** - System design and patterns
- **[Testing Guide](https://gohector.dev/TESTING)** - Testing practices

**[📚 Complete Documentation →](https://gohector.dev)**
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

## Say hi to Hector!

![Hector Gopher Logo](gopher.png)

## License

**AGPL-3.0** - See [LICENSE.md](LICENSE.md) for details.

Hector is free and open-source software. You can use, modify, and distribute it under the terms of the AGPL-3.0 license, which requires:
- Source code disclosure for network use
- Same license for derivative works
- Patent grant to users

For commercial licensing options, please contact the maintainers.
