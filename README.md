# Hector

```
██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```

**Declarative AI Agent Platform**

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE.md)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)
[![Status](https://img.shields.io/badge/status-alpha-orange.svg)](https://github.com/kadirpekel/hector)
[![Go Report Card](https://goreportcard.com/badge/github.com/kadirpekel/hector)](https://goreportcard.com/report/github.com/kadirpekel/hector)

> ⚠️ **Alpha Stage**: Core features are stable, but APIs may change. Production use at your own discretion.

**📚 Documentation:**
- [Configuration Reference](CONFIGURATION.md) - Complete YAML options and examples
- [Architecture Guide](ARCHITECTURE.md) - System design, patterns, and multi-agent orchestration
- [Example Configs](configs/) - Ready-to-use templates (coding, workflows, etc.)

---

## What is Hector?

Hector is an **AI agent** you configure, not code. It's a complete agentic system that adapts to any task through declarative YAML configuration—from weather queries to complex workflows.

**What makes Hector different:**
- **Hector IS the agent**: You configure what it knows and can do, it handles the rest
- **Declarative configuration**: Define capabilities in YAML, no programming required
- **Domain-agnostic**: Same agent, infinite personas (research, support, development, analysis)
- **Extensible platform**: Connect tools, LLMs, and data sources through configuration
- **Self-hosted intelligence**: Your agent, your data, your control

---

## Quick Start

```bash
# Clone and build
git clone https://github.com/kadirpekel/hector
cd hector
go build -o hector cmd/hector/main.go
./install.sh  # Optional: adds to PATH

# Configure your agent
cp .env.example .env
# Add your OPENAI_API_KEY to .env

# Talk to Hector
hector
```

**See Hector in action:**

```bash
# General knowledge (works immediately)
echo "Explain quantum computing like I'm five" | hector

# Task management (built-in tool)
echo "Create a todo list for planning a weekend trip to Paris" | hector

# Software development
hector coding
> "Create a REST API with JWT authentication"

# Research & analysis
echo "What are the top 3 trends in AI agents for 2024?" | hector
```

**Same agent, different capabilities** through configuration.

**Connect external tools via MCP** (when you have an MCP server running):

```yaml
# configs/weather-agent.yaml (example)
agents:
  weather_assistant:
    llm: "gpt-4o"
    prompt:
      system_role: "Friendly weather assistant with humor"
    
tools:
  weather:
    type: mcp
    server_url: "${MCP_WEATHER_SERVER}"  # Your MCP server
```

```bash
# Then use it (requires running MCP server)
hector --config configs/weather-agent.yaml
> "What's the weather in Tokyo? 🌂"
```

**The power**: Hector adapts through configuration. Built-in tools work immediately, external tools connect via MCP protocol.

**Note**: The weather-agent config demonstrates MCP integration. You'll need a running MCP weather server to use it. See [Model Context Protocol](https://modelcontextprotocol.io/) for server examples.

**Configure Hector per workspace:**

Point Hector at different configs to change its entire personality and capabilities:

```bash
# Individual workspaces (any domain)
research-project/
  ├── hector.yaml          # Research assistant config
  └── data/

writing-workspace/
  ├── hector.yaml          # Content creation config
  └── drafts/

# Multi-tenant shared configs (teams/orgs)
company-workspace/
  ├── configs/
  │   ├── support.yaml     # Customer support agent
  │   ├── research.yaml    # Research analyst
  │   ├── writer.yaml      # Content writer
  │   └── developer.yaml   # Software development
  └── projects/

# Use from anywhere
cd research-project && hector           # Research mode
cd company-workspace && hector support  # Support mode
cd company-workspace && hector developer # Development mode
```

**Use cases:**
- **Domain-Specific**: Configure agents for research, writing, support, development, etc.
- **Team Workflows**: Share persona configs across team (research standards, writing style, support protocols)
- **Multi-Persona**: Switch contexts easily (researcher, writer, analyst, developer)
- **Client/Project Separation**: Different configs per client, project, or environment

---

## Core Capabilities

### 🛠️ Extensive Tool System

**Built-in Tools:**
- `execute_command`: Sandboxed shell execution
- `file_writer`: Create and modify files
- `search_replace`: Precise text replacement
- `search`: Semantic codebase search (requires vector DB)
- `todo_write`: Task management and tracking

**MCP Protocol - Connect External Tools:**

Hector can connect to external tool servers using the [Model Context Protocol](https://modelcontextprotocol.io/). Want weather data? GitHub access? Custom APIs? Point Hector at an MCP server:

```yaml
# configs/weather-agent.yaml (example pattern)
tools:
  weather:
    type: mcp
    server_url: "${MCP_WEATHER_SERVER}"  # Your MCP server
```

```bash
# .env
MCP_WEATHER_SERVER=http://localhost:3000  # Your running MCP server
```

**Note**: MCP integration requires running MCP servers. The weather-agent example shows the pattern—you'll need to set up MCP servers for your specific tools.

**What you can connect:**
- **Data & APIs**: Weather services, news feeds, databases, REST APIs
- **Development**: GitHub, GitLab, Jira, CI/CD systems
- **Cloud**: AWS, GCP, Azure management tools
- **Communication**: Slack, Discord, email integrations
- **Custom**: Your own domain-specific MCP servers

**The power**: Pure declarative extension. Configure once, Hector gains abilities—no code changes, no rebuilds.

**Configuration Presets (Security Profiles):**

Hector provides example configurations with different tool access levels:
- **Safe Mode** (`hector.yaml`): Read-only commands + task management
- **Developer Mode** (`configs/coding.yaml`): File editing + expanded commands
- **Custom**: Define your own tool permissions

*Note: Tool permissions are configured via YAML, not enforced at the framework level. Review and customize tool access for your use case.*

### 🤝 Multi-Agent Orchestration

**Hector can split itself into specialized sub-agents** that coordinate on complex tasks:

```yaml
agents:
  researcher:
    name: "Research Agent"
    llm: "gpt-4o"
    prompt:
      prompt_slots:
        system_role: "Information gathering specialist"
  
  analyst:
    name: "Analysis Agent"
    llm: "claude-3-7-sonnet"
    prompt:
      prompt_slots:
        system_role: "Data analysis expert"
```

**Workflow Orchestration (In Development):**
- **DAG Execution**: Dependency-based coordination
- **Context Sharing**: Pass data between agents
- **Progress Tracking**: Real-time workflow events
- **Error Recovery**: Retries and rollback

[See example →](configs/research-pipeline-workflow.yaml)

### 🧠 Pluggable Reasoning Strategies

**Chain-of-Thought (Production):**
- Iterative problem solving
- Dynamic tool usage
- Self-reflection and replanning

**Future Strategies:**
- Tree-of-Thought
- Reflexion
- Multi-step planning

Define in config:
```yaml
reasoning:
  engine: "chain-of-thought"
  max_iterations: 10
  enable_streaming: true
```

### 🔌 LLM Provider Flexibility

Switch providers via configuration:

```yaml
llms:
  main:
    type: "anthropic"
    model: "claude-3-7-sonnet-latest"
    temperature: 0.7
```

**Supported:**
- OpenAI (GPT-4o, GPT-4, etc.)
- Anthropic (Claude 3.7 Sonnet, etc.)
- Extensible: Add custom providers

### 🔍 Semantic Search

Index codebases for intelligent search:

```yaml
document_stores:
  codebase:
    path: "."
    include_patterns: ["*.go", "*.py", "*.js"]
    database: "qdrant"
    embedder: "ollama"
```

**Requires:** Qdrant (vector DB) + Ollama (embeddings)

### 📝 Prompt Engineering via Slots

Customize agent behavior without full prompt rewrites:

```yaml
prompt:
  prompt_slots:
    system_role: "Expert software architect"
    tool_usage: "Proactively use search and file tools"
    reasoning_instructions: "Break complex tasks into steps"
    communication_style: "Concise, technical, actionable"
```

**Three levels of customization:**
1. Strategy defaults (built-in)
2. Partial override via `prompt_slots`
3. Full override via `system_prompt`

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  YAML Configuration Layer                       │
│   (Agents, Workflows, Tools, LLMs, Prompt Slots, Security)     │
└────────────────────────────┬────────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────────┐
│                      Hector Engine Core                         │
│                                                                 │
│  ┌──────────────┐    ┌─────────────────┐   ┌───────────────┐  │
│  │    Agent     │ ←→ │    Strategy     │   │   Workflow    │  │
│  │ Orchestrator │    │ (Chain-of-      │   │   Executors   │  │
│  │              │    │  Thought, etc)  │   │  (DAG, Auto)  │  │
│  └──────┬───────┘    └─────────────────┘   └───────┬───────┘  │
│         │                                           │          │
│         ├─────► LLM Service (OpenAI, Anthropic)    │          │
│         ├─────► Tool Service (Local, MCP)          │          │
│         ├─────► Context Service (Search, History)  │          │
│         └─────► Prompt Service (Slots, Templates) ◄┘          │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Multi-Agent Orchestration                   │  │
│  │  Team System → Workflow Registry → Executor Selection   │  │
│  │  Context Sharing → Dependency Management → Streaming    │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────────┐
│                  External Integrations                          │
│  LLM APIs │ MCP Servers │ Vector DBs │ Tool Plugins │ Custom   │
└─────────────────────────────────────────────────────────────────┘
```

**Key Patterns:**
- **Strategy Pattern**: Pluggable reasoning engines
- **Service-Oriented**: Clean boundaries for testing and extension
- **Interface-Based**: Extend without modifying core
- **Event-Driven**: Streaming execution with real-time events

**Component Responsibilities:**
- **Agent**: Orchestrates reasoning loop, coordinates services
- **Strategy**: Implements reasoning approach (CoT, ToT, etc.)
- **Workflow**: Multi-agent coordination and dependency management
- **Services**: Isolated concerns (LLM, tools, context, prompts)

---

## Configuration Example

**Basic Single Agent:**
```yaml
agents:
  assistant:
    name: "Development Assistant"
    llm: "main-llm"
    
    prompt:
      prompt_slots:
        system_role: "Expert software development assistant"
        tool_usage: "Use file editing and search proactively"
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 10
      enable_streaming: true

llms:
  main-llm:
    type: "anthropic"
    model: "claude-3-7-sonnet-latest"
    api_key: "${ANTHROPIC_API_KEY}"
    temperature: 0.7

tools:
  execute_command:
    type: command
    allowed_commands: ["ls", "cat", "grep"]
  
  file_writer:
    type: file_writer
```

**Multi-Agent Workflow:**
```yaml
workflows:
  research_pipeline:
    mode: "dag"
    execution:
      dag:
        steps:
          - name: "research"
            agent: "researcher"
            input: "${user_input}"
            output: "research_data"
          
          - name: "analyze"
            agent: "analyst"
            input: "Analyze: ${research_data}"
            depends_on: [research]
            output: "analysis"
          
          - name: "report"
            agent: "writer"
            input: "Report: ${research_data}, ${analysis}"
            depends_on: [research, analyze]
```

[Full configuration reference →](CONFIGURATION.md)

---

## Use Cases

### General Purpose
- **Research Assistants**: Information gathering, fact-checking, literature review
- **Content Creation**: Blog posts, documentation, technical writing, marketing copy
- **Data Analysis**: CSV processing, pattern recognition, insights extraction, reporting
- **Customer Support**: Automated responses, ticket triage, FAQ generation
- **Education**: Tutoring, concept explanation, quiz generation, study guides

### Technical/Development
- **Coding Assistance**: Code generation, refactoring, debugging (via `hector coding`)
- **DevOps Automation**: Infrastructure scripts, deployment automation, monitoring
- **API Integration**: Connect external services via MCP protocol, workflow orchestration

### Enterprise Workflows
- **Multi-Agent Pipelines**: Research → Analysis → Reporting with specialized agents
- **Document Processing**: Extract, transform, summarize large document collections
- **Compliance & Audit**: Policy checking, automated reporting, audit trail generation

---

## Extensibility

### Add Custom Tools

```go
type MyTool struct{}

func (t *MyTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
    // Your implementation
    return ToolResult{Success: true, Content: "Done"}, nil
}

func (t *MyTool) GetDefinition() ToolDefinition {
    return ToolDefinition{
        Name: "my_tool",
        Description: "Does something useful",
        Parameters: []ParameterDefinition{
            {Name: "input", Type: "string", Description: "Input data", Required: true},
        },
    }
}
```

### Add Reasoning Strategies

```go
type CustomStrategy struct{}

func (s *CustomStrategy) PrepareIteration(state *ReasoningState) error {
    // Your reasoning logic
    return nil
}

func (s *CustomStrategy) GetPromptSlots() map[string]string {
    return map[string]string{
        "reasoning_instructions": "Your custom approach...",
    }
}
```

### Add LLM Providers

```go
type CustomProvider struct{}

func (p *CustomProvider) Generate(messages []Message, tools []ToolDefinition) (string, []ToolCall, int, error) {
    // Your provider implementation
}
```

**All extensions are hot-pluggable via configuration.**

---

## What Works Today

**Production-Ready:**
- ✅ Single-agent execution with streaming
- ✅ File creation and modification  
- ✅ Sandboxed command execution
- ✅ LLM provider flexibility (OpenAI, Anthropic)
- ✅ Semantic search (Qdrant + Ollama)
- ✅ Tool system (built-in + MCP protocol foundation)
- ✅ Prompt customization via slots

**Experimental:**
- 🧪 Multi-agent workflow orchestration (DAG executor implemented, needs production validation)
- 🧪 Autonomous workflow mode (research prototype)

**Not Yet Available:**
- ❌ Web UI (CLI only)
- ❌ Visual workflow designer
- ❌ Additional LLM providers (extensible, contributions welcome)

---

## Installation

### From Source
```bash
git clone https://github.com/kadirpekel/hector
cd hector
go build -o hector cmd/hector/main.go
./install.sh  # Optional: adds to PATH
```

### Requirements
- Go 1.21+
- LLM API access (OpenAI or Anthropic)
- Optional: Qdrant + Ollama (semantic search)
- Optional: MCP servers (external tools)

### Configuration Files

Hector looks for configuration in this order:
1. `--config` flag (explicit path)
2. `hector.yaml` in current directory
3. Zero-config mode (safe defaults with `OPENAI_API_KEY` from env)

### Pre-built Configs

```bash
# General-purpose (default)
hector

# Development assistant (Cursor-like experience with full capabilities)
hector coding

# Multi-agent workflow (experimental)
hector --config configs/research-pipeline-workflow.yaml --workflow research_pipeline
```

---

## License

**AGPL-3.0 for Personal Use** | **Commercial License Required**

Hector is dual-licensed:
- **Personal/Non-Commercial**: Free under AGPL-3.0 (hobbyists, education, research, open-source)
- **Commercial Use**: Requires a commercial license (for-profit companies, SaaS, enterprise)

**What's Commercial?**
- Using Hector at a for-profit company
- Building commercial products/services with Hector
- Any use that generates revenue

See [LICENSE.md](LICENSE.md) for full terms and commercial licensing inquiries.

---

## Links

- **GitHub**: [kadirpekel/hector](https://github.com/kadirpekel/hector)
- **Issues**: [Report bugs or request features](https://github.com/kadirpekel/hector/issues)
