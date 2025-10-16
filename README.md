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
[![A2A Protocol](https://img.shields.io/badge/A2A%20Compliance-100%25-brightgreen.svg)](https://gohector.dev/A2A_COMPLIANCE)
[![Documentation](https://img.shields.io/badge/docs-gohector.dev-blue.svg)](https://gohector.dev)
[![Go Report Card](https://goreportcard.com/badge/github.com/kadirpekel/hector)](https://goreportcard.com/report/github.com/kadirpekel/hector)
[![GoDoc](https://godoc.org/github.com/kadirpekel/hector?status.svg)](https://godoc.org/github.com/kadirpekel/hector)

> ⚠️ Alpha Version Disclaimer: This project is currently in alpha. It is under active development and not yet stable. Features may change, break, or be removed at any time.

# Hector
**Declarative AI Agent Platform with Native A2A Protocol Support**

Hector is a **declarative AI agent platform** that eliminates code from agent development. Built on the [Agent-to-Agent protocol](https://a2a-protocol.org), Hector enables true interoperability between agents across networks, servers, and organizations.

**🚀 From idea to production agent in minutes, not months.**

**Platform Capabilities:**
- **Zero Code Development** - Define agents, tools, prompts, and orchestration entirely in YAML
- **A2A Protocol Native** - 100% compliant with Agent-to-Agent protocol for true interoperability
- **Single & Multi-Agent** - From standalone agents to complex orchestration networks
- **Distributed Deployment** - Deploy agents across multiple servers and organizations
- **Production Security** - JWT authentication, RBAC, and secure agent communication

Visit [gohector.dev](https://gohector.dev) for complete documentation.

### See it in action!
```bash
go install github.com/kadirpekel/hector/cmd/hector@latest

export OPENAI_API_KEY="sk-..."
export MCP_URL="https://apollo.composio.dev/v3/mcp/..."

# Run hector in zero-config mode
hector call "what to wear today in berlin?"

Agent: I'll check the current weather in Berlin to help you decide what to wear today.
🔧 WEATHERMAP_WEATHER ✅

☀️ **15°C (59°F), scattered clouds** - Perfect autumn weather in Berlin! 🌤️ Light jacket and comfortable shoes recommended for a pleasant day outdoors.
```

---

## 🎯 **Why Choose Hector?**

**Transform complex AI agent development from weeks of coding to minutes of configuration.**

### **For Individuals & Developers**
- **🚀 Zero Code Development** - Build powerful AI agents with pure YAML, no programming required
- **⚡ Instant Setup** - Start in seconds with zero-config mode, no complex setup
- **🧠 Advanced Memory Systems** - Working memory, session management, and context window optimization
- **🔍 RAG & Document Stores** - Semantic search across knowledge bases with automatic retrieval
- **🛠️ Rich Tool Ecosystem** - Built-in tools (search, file ops, commands) + MCP protocol integration
- **📚 Task Management** - Async processing with real-time status tracking and streaming
- **🎯 Structured Output** - Provider-aware JSON/XML/Enum for reliable data extraction
- **🔄 Multi-Turn Sessions** - Conversation history and context across interactions

### **For Enterprises & Teams**
- **🌐 True Interoperability** - 100% A2A protocol compliant, works with any A2A agent
- **🤖 Multi-Agent Orchestration** - LLM-driven agent coordination and task delegation
- **🏢 Distributed Architecture** - Deploy agents across servers, teams, and organizations
- **🔒 Production Security** - JWT authentication, RBAC, and secure agent communication
- **📡 Multi-Transport APIs** - gRPC, REST, JSON-RPC - integrate with any system
- **🔌 Plugin System** - gRPC-based extensibility for custom LLMs, databases, and tools
- **⚡ Real-Time Streaming** - Token-by-token output via Server-Sent Events (SSE)
- **🎛️ Fine-Grained Control** - Slot-based prompt system for precise agent behavior

---

## 🏗️ **Agent Architecture**

**How Hector agents work under the hood:**

```
┌──────────────────────────────────────────────────────────────┐
│                      APPLICATION                             │
│                 (Your Agent & Logic)                         │
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

---

## 🌐 **Distributed Multi-Agent Platform**

**Enterprise-grade agent networks and orchestration:**

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

### **Production Examples**

**Deploy Multi-Agent Server:**
```bash
hector serve --config agents.yaml --port 8080
```

**Connect from Anywhere:**
```bash
hector call coordinator "Research AI trends" --server https://agents.company.com:8080
```

**API Integration:**
```bash
curl -X POST https://agents.company.com:8080/v1/agents/coordinator/message:send \
  -H "Content-Type: application/json" \
  -d '{"message": "Research AI trends"}'
```

**Hybrid Agent Networks:**
```yaml
agents:
  local_assistant:
    type: "native"
    llm: "gpt-4o"
  remote_analyst:
    type: "a2a"
    url: "https://analytics.partner.com/a2a"
```

---

## 🚀 **Get Started Now**

```bash
# Install and run in 30 seconds
go install github.com/kadirpekel/hector/cmd/hector@latest
export OPENAI_API_KEY="your-key-here"
hector call "Hello, Hector!"
```

---

## 📚 **Documentation & Resources**

- **[Complete Documentation](https://gohector.dev)** - Full platform guide
- **[CLI Guide](https://gohector.dev/CLI_GUIDE.html)** - Command-line interface
- **[Agent Configuration](https://gohector.dev/AGENTS.html)** - YAML configuration guide
- **[Multi-Agent Tutorial](https://gohector.dev/TUTORIAL_MULTI_AGENT.html)** - Orchestration workflows
- **[A2A Compliance](https://gohector.dev/A2A_COMPLIANCE.html)** - Protocol standards

## Say Hi! to Hector!

![Hector Gopher Logo](gopher.png)

## License

**AGPL-3.0** - See [LICENSE.md](LICENSE.md) for details.

Hector is free and open-source software. You can use, modify, and distribute it under the terms of the AGPL-3.0 license.

For commercial licensing options, please contact the maintainers.
