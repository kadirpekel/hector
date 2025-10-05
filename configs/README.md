# Hector Configuration Examples

This directory contains example configurations demonstrating various use cases for Hector's A2A-native agent platform.

---

## 📋 **Available Configurations**

### 1. **A2A Server** (`a2a-server.yaml`) - 🌐 Production Ready
**Purpose:** Run Hector as an A2A-compliant server exposing agents via HTTP API

**Features:**
- **3 Example Agents**: competitor_analyst, customer_support, research_orchestrator
- 🌐 **A2A Protocol**: Full compliance with A2A specification
- 🔌 **HTTP API**: REST endpoints for agent discovery and task execution
- 📡 **External Access**: Other A2A clients can consume your agents

**Usage:**
```bash
# Start A2A server
hector serve --config configs/a2a-server.yaml

# Test with CLI
hector list
hector info competitor_analyst
hector call competitor_analyst "Analyze AI agent market"

# Test with curl (pure A2A protocol)
curl http://localhost:8080/agents
curl http://localhost:8080/agents/competitor_analyst
```

**Use Cases:**
- Expose agents as services
- API-first deployment
- Integration with external systems
- Multi-tenant agent platforms

---

### 2. **Orchestrator Example** (`orchestrator-example.yaml`) - 🎯 Multi-Agent
**Purpose:** Demonstrates multi-agent orchestration with the `agent_call` tool

**Features:**
- 🤝 **4 Agents**: researcher, analyst, writer, orchestrator
- 🔧 **agent_call Tool**: Orchestrator delegates to specialized agents
- 🧠 **Supervisor Strategy**: Optimized reasoning for coordination
- 📋 **Task Decomposition**: Breaks complex tasks into subtasks

**Usage:**
```bash
# Start server with orchestrator
hector serve --config configs/orchestrator-example.yaml

# Test orchestration (single complex task)
hector call orchestrator "Research AI frameworks, analyze top 3, and write a comparison report"

# Expected behavior:
# 1. Orchestrator breaks down the task
# 2. Calls researcher: "Research AI frameworks"
# 3. Calls analyst: "Analyze these frameworks: [results]"
# 4. Calls writer: "Write comparison report: [analysis]"
# 5. Synthesizes final output
```

**Use Cases:**
- Complex multi-step workflows
- Task decomposition and delegation
- Specialized agent coordination
- Research and analysis pipelines

---

### 3. **Coding Assistant** (`coding.yaml`) - 💻 Developer Mode
**Purpose:** Cursor/Claude-like pair programming experience with full dev tools

**⚠️  Security Notice:** Includes file editing and command execution. Only use in trusted environments.

**Features:**
- 🤖 **LLM**: Claude Sonnet 3.7 (Anthropic)
- 🔧 **Full Dev Tools**: file_writer, search_replace, execute_command, todo_write
- 🔍 **Semantic Search**: Qdrant + Ollama for codebase understanding
- 📝 **Optimized Prompt**: Matches Cursor's pair programming behavior
- 🎯 **High Precision**: Temperature 0.1 for deterministic code

**Usage:**
```bash
# Interactive mode
hector serve --config configs/coding.yaml
hector call assistant "Refactor the auth module to use JWT"

# Prerequisites:
# - ANTHROPIC_API_KEY (required)
# - Qdrant + Ollama (optional, for semantic search)
```

**Use Cases:**
- Professional pair programming
- Multi-file refactoring
- Code generation with context
- Bug fixing and debugging
- Architecture changes

---

### 4. **Weather Agent** (`weather-agent.yaml`) - 🌤️ MCP Integration
**Purpose:** Example showing Model Context Protocol (MCP) tool integration

**⚠️  Requires:** MCP weather server running at `${MCP_SERVER_URL}`

**Features:**
- 🌐 **MCP Integration**: Connects to external MCP servers
- 🌤️ **Weather Tool**: Real-time weather data access
- 😊 **Personality**: Friendly, emoji-rich responses
- 📚 **Integration Example**: Template for your own MCP tools

**Usage:**
```bash
# Setup MCP server first (see: https://modelcontextprotocol.io)
export MCP_SERVER_URL="http://localhost:3000"

# Start agent
hector serve --config configs/weather-agent.yaml
hector call weather_assistant "What's the weather in Tokyo?"
```

**Use Cases:**
- External API integration patterns
- MCP server connectivity
- Tool integration examples
- Custom tool development reference

---

## Quick Start

### Basic A2A Server
```bash
# 1. Set API key
export OPENAI_API_KEY="your-key"

# 2. Start server
hector serve --config configs/a2a-server.yaml

# 3. Test
hector list
hector call competitor_analyst "Your task here"
```

### Multi-Agent Orchestration
```bash
# Start orchestrator server
hector serve --config configs/orchestrator-example.yaml

# Run complex multi-agent task
hector call orchestrator "Research Python vs Rust, analyze performance, write recommendations"
```

---

## Configuration Comparison

| Feature | A2A Server | Orchestrator | Coding | Weather |
|---------|------------|--------------|--------|---------|
| **Agents** | 3 specialists | 4 (3 + orchestrator) | 1 assistant | 1 weather |
| **Reasoning** | chain-of-thought | supervisor | chain-of-thought | chain-of-thought |
| **Tools** | command, file | agent_call | full dev suite | MCP weather |
| **LLM** | GPT-4o | GPT-4o-mini | Claude Sonnet | GPT-4o |
| **Use Case** | Production API | Complex workflows | Development | Integration demo |
| **Security** | 🟢 Safe | 🟢 Safe | 🟡 Dev Mode | 🟢 Safe |

---

## Customization

All configs can be customized by modifying:

1. **LLM Configuration**
   ```yaml
   llms:
     main-llm:
       type: "openai"  # or "anthropic"
       model: "gpt-4o"
       temperature: 0.7
   ```

2. **Agent Prompts**
   ```yaml
   agents:
     my_agent:
       prompt:
         system_role: "Your custom role"
   ```

3. **Reasoning Strategy**
   ```yaml
   reasoning:
     engine: "chain-of-thought"  # or "supervisor"
     max_iterations: 10
   ```

4. **Tools**
   ```yaml
   tools:
     - execute_command
     - agent_call  # For orchestration
   ```

---

## Learn More

- **[Configuration Reference](../docs/CONFIGURATION.md)** - Complete config options
- **[Architecture](../docs/ARCHITECTURE.md)** - System design and A2A protocol
- **[External Agents](../docs/EXTERNAL_AGENTS.md)** - Integrate remote A2A agents
- **[Orchestration Guide](../docs/ORCHESTRATOR_SUMMARY.md)** - Multi-agent patterns

---

## Tips

### For Production
- Start with `a2a-server.yaml`
- Add authentication (API keys, OAuth2)
- Enable monitoring and logging
- Run behind reverse proxy (nginx, Caddy)

### For Development
- Use `coding.yaml` for full dev capabilities
- Enable semantic search for better code understanding
- Set up proper backups before using file editing tools
- Test in isolated environments first

### For Orchestration
- Study `orchestrator-example.yaml` patterns
- Use `supervisor` strategy for coordinators
- Keep specialist agents focused and simple
- Build complex behaviors through composition

---

**Need help?** Check the [docs/](../docs/) directory for comprehensive guides.
