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

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE.md)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)
[![A2A Protocol](https://img.shields.io/badge/A2A-compliant-green.svg)](https://a2a-protocol.org)

> **Define agents in YAML, serve via A2A protocol, orchestrate without code**

**📚 Quick Links:**
- [Quick Start](#-quick-start) - Get running in 5 minutes
- [A2A Server](#-a2a-server-mode) - Host agents via A2A protocol
- [Multi-Agent Orchestration](#-multi-agent-orchestration) - Coordinate multiple agents
- [External Agents](docs/EXTERNAL_AGENTS.md) - Use remote A2A agents
- [Architecture](docs/ARCHITECTURE.md) - System design
- [Documentation](docs/) - Complete documentation

---

## 🌟 **What is Hector?**

Hector is a **pure A2A-native declarative AI agent platform** that enables you to:

- **📝 Define agents in YAML** - Native AND external agents, zero code required
- **🌐 Serve via A2A protocol** - Industry-standard agent communication
- **🤖 Deploy single or multi-agent systems** - From simple assistants to complex workflows
- **🔗 Orchestrate with LLM-driven delegation** - Agents coordinate intelligently
- **🔌 Integrate external agents** - Connect to any A2A agent via URL in your config
- **🌍 Agent Ecosystem Ready** - Interoperate within organizations or the broader agent internet
- **🔒 Visibility Control** - Fine-grained control over agent exposure (public/internal/private)

### **How Hector is Different**

Unlike frameworks like **LangChain**, **AutoGen**, or **CrewAI** that require writing Python code to define agents, Hector uses **pure YAML configuration**:

```yaml
# Define your ENTIRE multi-agent system in one YAML file
agents:
  # Native agent (runs locally)
  assistant:
    name: "Customer Support Agent"
    llm: "gpt-4o"
    reasoning:
      engine: "chain-of-thought"
    tools:
      - search
      - database
  
  # External A2A agent (just provide URL!)
  partner_specialist:
    type: "a2a"
    url: "https://partner-ai.com/agents/specialist"
    # That's it! No code, no API integration - pure interoperability
```

**Key Differentiators:**
- ✅ **Declarative** - YAML configuration for native AND external agents, zero code required
- ✅ **A2A-Native** - 100% protocol compliance enables universal interoperability
- ✅ **External Agent Integration** - Connect to any A2A agent via URL in your YAML
- ✅ **Multi-Agent** - Built-in orchestration via `agent_call` tool across all agents
- ✅ **Agent Internet Ready** - Participate in the emerging agent ecosystem
- ✅ **Enterprise Interoperability** - Integrate partner/vendor agents declaratively

---

## 🚀 **Quick Start**

### Install

```bash
# Clone and build
git clone https://github.com/kadirpekel/hector
cd hector
go build -o hector ./cmd/hector

# Optional: Install to PATH
./install.sh
```

### Start A2A Server

```bash
# 1. Set API key
export OPENAI_API_KEY="sk-..."

# 2. Start server hosting multiple agents
./hector serve --config configs/a2a-server.yaml

# Expected output:
# 🚀 Starting Hector A2A Server...
# 📋 Registering agents...
#   ✅ Competitor Analysis Agent (competitor_analyst)
#   ✅ Customer Support Agent (customer_support)
# 🌐 A2A Server ready!
# 📡 Agent directory: http://localhost:8080/agents
```

### Test with CLI

```bash
# List available agents
./hector list

# Get agent details
./hector info competitor_analyst

# Execute a task
./hector call competitor_analyst "Analyze top AI agent frameworks"

# Interactive chat
./hector chat competitor_analyst
```

**That's it!** You now have an A2A server running with multiple agents.

---

## 💡 **Key Concepts**

### 1. **A2A Protocol**

Hector implements the [A2A (Agent-to-Agent) protocol](https://a2a-protocol.org), an open standard for agent interoperability.

**Benefits:**
- ✅ **Interoperability** - Works with any A2A-compliant client or agent
- ✅ **Discovery** - Agents publish capability cards
- ✅ **Standard Communication** - TaskRequest/TaskResponse model
- ✅ **Ecosystem** - Contribute to growing A2A ecosystem

### 2. **Pure A2A Architecture**

```
┌─────────────────────────────────────┐
│  User / External A2A Client         │
└───────────┬─────────────────────────┘
            │ A2A Protocol (HTTP/JSON)
┌───────────▼─────────────────────────┐
│  Hector A2A Server                  │
│  • Agent discovery                  │
│  • Task execution                   │
│  • Session management               │
└───────────┬─────────────────────────┘
            │
    ┌───────┼───────┐
    │       │       │
    ▼       ▼       ▼
┌────────┐ ┌────────┐ ┌────────┐
│Agent 1 │ │Agent 2 │ │Agent 3 │
│(Native)│ │(Native)│ │(Remote)│
└────────┘ └────────┘ └────────┘
```

**All communication via A2A protocol - no proprietary APIs!**

### 3. **Multi-Agent Orchestration**

Instead of hard-coded workflows, Hector uses:
- **LLM-driven delegation** - Orchestrator agent decides routing
- **agent_call tool** - Delegates to other agents via A2A
- **Transparent** - Native and external agents treated identically

```yaml
agents:
  orchestrator:
    tools:
      - agent_call  # Enable orchestration
    reasoning:
      engine: "supervisor"  # Optimized for delegation
    prompt:
      system_role: |
        Coordinate other agents using agent_call.
        Available: researcher, analyst, writer
```

---

## 🌐 **A2A Server Mode**

### Start Server

```bash
./hector serve --config configs/a2a-server.yaml
```

### A2A Endpoints

```
GET  /agents                    → List all agents
GET  /agents/{id}               → Get agent card (capabilities)
POST /agents/{id}/tasks         → Execute task
GET  /agents/{id}/tasks/{taskId} → Get task status
```

### Example: Call via curl

```bash
# Discover agents
curl http://localhost:8080/agents

# Get agent card
curl http://localhost:8080/agents/competitor_analyst

# Execute task
curl -X POST http://localhost:8080/agents/competitor_analyst/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "taskId": "task-1",
    "input": {
      "type": "text/plain",
      "content": "Analyze top 3 AI frameworks"
    }
  }'
```

### Example: Python Client

```python
import requests

# Discover agent
card = requests.get("http://localhost:8080/agents/competitor_analyst").json()
print(f"Agent: {card['name']}")
print(f"Capabilities: {card['capabilities']}")

# Execute task
task = {
    "taskId": "py-task-1",
    "input": {
        "type": "text/plain",
        "content": "Analyze Rust vs Go"
    }
}

response = requests.post(
    "http://localhost:8080/agents/competitor_analyst/tasks",
    json=task
)

result = response.json()
print(f"Status: {result['status']}")
print(f"Output: {result['output']['content']}")
```

**Any A2A-compliant client can interact with Hector agents!**

---

## 🔗 **Multi-Agent Orchestration**

### Simple Example

```yaml
# configs/orchestrator-simple.yaml
agents:
  researcher:
    name: "Research Agent"
    llm: "gpt-4o-mini"
  
  analyst:
    name: "Analysis Agent"
    llm: "gpt-4o-mini"
  
  orchestrator:
    name: "Orchestrator"
    llm: "gpt-4o"
    tools:
      - agent_call  # THE KEY TOOL
    reasoning:
      engine: "supervisor"
    prompt:
      system_role: |
        Coordinate agents:
        - researcher: Gathers information
        - analyst: Analyzes data
        
        Use agent_call to delegate tasks.
```

### Test Orchestration

```bash
# Start server
./hector serve --config configs/orchestrator-simple.yaml

# Call orchestrator (it will delegate to others)
./hector call orchestrator "Research AI frameworks and analyze top 3"
```

**Expected flow:**
1. Orchestrator receives task
2. Calls researcher: "Research AI frameworks"
3. Calls analyst: "Analyze top 3: [research results]"
4. Synthesizes final response

### Advanced Example

See `configs/orchestrator-example.yaml` for a complete multi-agent system with:
- Research Agent
- Analysis Agent
- Content Writer
- Orchestrator (coordinates all)

---

## 🔌 **External A2A Agents**

Hector can orchestrate **external A2A agents** alongside native agents!

### Example: Use External Agent

```go
import (
    "context"
    "github.com/kadirpekel/hector/a2a"
    "github.com/kadirpekel/hector/agent"
)

// 1. Create A2A client
client := a2a.NewClient(&a2a.ClientConfig{})

// 2. Discover external agent
externalAgent, _ := agent.NewA2AAgentFromURL(
    context.Background(),
    "https://external-service.com/agents/translator",
    client,
)

// 3. Register in registry
registry := agent.NewAgentRegistry()
registry.RegisterAgent("translator", externalAgent, config, capabilities)

// 4. Orchestrator can now call it via agent_call!
```

**Key Point:** Native and external agents use the **same interface**. The orchestrator doesn't know (or care) about the difference!

**See [docs/EXTERNAL_AGENTS.md](docs/EXTERNAL_AGENTS.md) for complete guide.**

---

## ⚙️ **Configuration**

### Minimal Agent

```yaml
agents:
  hello:
    name: "Hello Agent"
    llm: "gpt-4o-mini"
    prompt:
      system_role: "You are a friendly assistant"

llms:
  gpt-4o-mini:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

### Agent with Tools

```yaml
agents:
  coder:
    name: "Coding Assistant"
    llm: "gpt-4o"
    tools:
      - write_file
      - execute_command
    prompt:
      system_role: "Expert programmer"

tools:
  write_file:
    type: file_system
    path: "./workspace"
  
  execute_command:
    type: command
    allowed_commands: ["ls", "cat", "python3"]
```

### Orchestrator Agent

```yaml
agents:
  orchestrator:
    name: "Task Orchestrator"
    llm: "gpt-4o"
    tools:
      - agent_call  # Enable orchestration
    reasoning:
      engine: "supervisor"  # Optimized strategy
      max_iterations: 20
    prompt:
      system_role: |
        Coordinate other agents using agent_call tool.
```

**See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for complete reference.**

---

## 🎯 **Use Cases**

### 1. Single Agent Execution

```bash
# Direct agent call
echo "Explain quantum computing" | ./hector
```

### 2. A2A Server

```bash
# Host multiple agents via A2A protocol
./hector serve --config configs/a2a-server.yaml

# Any A2A client can connect
curl http://localhost:8080/agents
```

### 3. Multi-Agent Orchestration

```bash
# Orchestrator coordinates multiple agents
./hector call orchestrator "Research, analyze, and write report on AI"

# Flow: orchestrator → researcher → analyst → writer → synthesize
```

### 4. External Integration

```bash
# Mix native + external A2A agents
# Orchestrator calls both transparently
./hector call orchestrator "Use local researcher and external translator"
```

### 5. CLI Client

```bash
# Use Hector CLI as A2A client
./hector list --server https://external-a2a-server.com
./hector call external_agent "Task" --server https://...
```

---

## 🏗️ **Architecture**

### Core Components

1. **A2A Server** (`a2a/server.go`)
   - Hosts agents via A2A protocol
   - Handles discovery, execution, sessions
   
2. **Agent** (`agent/agent.go`)
   - Implements `a2a.Agent` interface
   - Pure A2A compliance (ExecuteTask, GetAgentCard)
   
3. **A2AAgent** (`agent/a2a_agent.go`)
   - Wraps external A2A agents
   - Same interface as native agents
   
4. **AgentRegistry** (`agent/registry.go`)
   - Stores `a2a.Agent` interface
   - Works with native + external agents
   
5. **agent_call Tool** (`agent/agent_call_tool.go`)
   - Enables orchestration
   - Transparent delegation

### Architecture Diagram

```
User/Client
    ↓ A2A Protocol
A2A Server
    ↓
AgentRegistry (a2a.Agent interface)
    ├─ Native Agents (in-process)
    │  └─ agent.Agent
    │
    └─ External A2A Agents (HTTP)
       └─ agent.A2AAgent → a2a.Client
```

**See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed design.**

---

## 📚 **Documentation**

**[📚 Complete Documentation →](docs/)**

- **[Quick Start](docs/QUICK_START.md)** - Get started in 5 minutes
- **[Architecture](docs/ARCHITECTURE.md)** - System design and A2A protocol
- **[Configuration](docs/CONFIGURATION.md)** - Complete config reference
- **[CLI Guide](docs/CLI_GUIDE.md)** - Command-line interface
- **[External Agents](docs/EXTERNAL_AGENTS.md)** - External agent integration
- **[Orchestrator Guide](docs/ORCHESTRATOR_SUMMARY.md)** - Multi-agent orchestration

---

## 🧪 **Testing**

### Basic Test

```bash
# Test A2A protocol
./test-a2a.sh
```

### Full Integration Test

```bash
# Complete server + client test
./test-a2a-full.sh
```

### Manual Testing

```bash
# Terminal 1: Start server
./hector serve --config configs/orchestrator-example.yaml

# Terminal 2: Test commands
./hector list
./hector info orchestrator
./hector call orchestrator "Research AI and write summary"
```

---

## 🔧 **CLI Reference**

### Server Commands

```bash
hector serve [--config FILE] [--debug]
```

### Client Commands

```bash
hector list [--server URL] [--token TOKEN]
hector info <agent> [--token TOKEN]
hector call <agent> "prompt" [--server URL] [--token TOKEN]
hector chat <agent> [--server URL] [--token TOKEN]
hector help
hector version
```

### Environment Variables

```bash
export HECTOR_SERVER="http://localhost:8080"
export HECTOR_TOKEN="your-bearer-token"
export OPENAI_API_KEY="sk-..."
```

**See [docs/CLI_GUIDE.md](docs/CLI_GUIDE.md) for complete reference.**

---

## 📄 **License**

AGPL-3.0 - See [LICENSE.md](LICENSE.md)
