<div align="center">
  <img src="hector_logo.png" alt="Hector Logo">
</div>

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Alpha-orange.svg)](https://github.com/kadirpekel/hector)

# Hector

> **Declarative AI Agent Framework** - Define once, deploy anywhere. Build sophisticated single agents and multi-agent systems through pure YAML configuration.

## Why Declarative?

Hector eliminates code for agent orchestration. Define your AI system's architecture, reasoning strategies, and workflows in YAML—Hector handles the complexity.

**Traditional Approach:**
```go
// 100+ lines of code
agent := NewAgent(llm)
agent.AddTool(commandTool)
agent.SetReasoning(chainOfThought)
agent.Run()
```

**Hector's Declarative Approach:**
```yaml
# config.yaml - that's it!
agents:
  assistant:
    llm: "openai"
    reasoning:
      engine: "chain-of-thought"
    tools: ["execute_command"]
```

## Table of Contents

- [Why Declarative?](#why-declarative)
- [Core Philosophy](#core-philosophy)
- [Features](#features)
- [Quick Start](#quick-start)
- [Multi-Agent Teams](#multi-agent-teams)
- [Architecture](#architecture)
- [Configuration](#configuration)
- [CLI Reference](#cli-reference)
- [Supported Providers](#supported-providers)
- [License](#license)

## Core Philosophy

### 1. **Configuration Over Code**
Define your AI system's behavior, not its implementation. Focus on what you want, not how to build it.

### 2. **Composable by Design**
Mix and match LLM providers, reasoning engines, tools, and agents like building blocks. Everything is pluggable.

### 3. **Streaming-First**
Real-time output with intelligent buffering, marker detection, and optimized response handling.

### 4. **Complexity Hidden, Power Exposed**
Simple things are simple, complex things are possible. Start with 5 lines of YAML, scale to enterprise workflows.

---

## Features

### Single Agent Capabilities

- **Two Reasoning Engines**: Choose your strategy declaratively
  - `chain-of-thought`: Fast, behavioral-signal-based (simple queries)
  - `structured-reasoning`: Goal-oriented with meta-cognition (complex analysis)
  - **Thinking Mode**: See internal reasoning in grayed-out blocks (like Claude in Cursor)

- **Universal Tool Integration**
  - **Local Tools**: Secure command execution with sandboxing
  - **MCP Protocol**: Connect to external tool servers
  - **Custom Tools**: Extend with your own implementations
  
- **Smart Context Management**
  - Vector search with Qdrant
  - Conversation history with intelligent truncation
  - Document stores with automatic indexing
  - Semantic search across knowledge bases

- **Production-Grade Streaming**
  - Real-time output with intelligent buffering
  - Marker detection for structured responses
  - Zero-allocation optimizations
  - User-friendly masking (no raw JSON)

### Multi-Agent Team Capabilities

- **Real-Time Streaming**: Watch agents work in real-time with progress tracking
- **Two Execution Modes**:
  - **DAG Mode**: Define dependencies, sequential/parallel execution
  - **Autonomous Mode**: AI-driven task planning and dynamic collaboration
- **Specialized Agents**: Each agent has its own LLM, reasoning strategy, and tools
- **Live Progress Tracking**: See which agent is running, completion percentage, real-time output
- **Event-Based Architecture**: Stream workflow events (start, progress, output, complete)

### Universal Features

- **Single YAML Configuration**: Everything from providers to workflows
- **Zero-Config Mode**: Intelligent defaults (Ollama + local tools)
- **Provider Agnostic**: OpenAI, Anthropic (Claude), Ollama - switch declaratively
- **Extension System**: Pluggable tools, memory, custom capabilities
- **Clean Architecture**: SOLID principles, dependency injection, no tight coupling

---

## See It In Action

Here's Hector's **structured reasoning** engine with **thinking mode** enabled:

**Query:** "What is 2+2 and why is it important in mathematics?"

**Output:**
```bash
# Internal reasoning
[Thinking: Goal identified: Answer the question about 2+2 and explain its significance]
[Thinking:   Sub-goal 1: Calculate the sum of 2+2]
[Thinking:   Sub-goal 2: Research and describe the role of basic arithmetic]
[Thinking:   Sub-goal 3: Explain foundational importance]

[Thinking: Iteration 1/10: reasoning]

# Agent response
To address your query, I will first calculate 2+2 and then explain its significance.

The sum of 2+2 is 4.

Role of Basic Arithmetic in Mathematics:
Basic arithmetic operations form the foundation of mathematics. Understanding these
operations is crucial because:

- Foundation for Advanced Topics: Mastery of basic arithmetic is essential for
  learning algebra, geometry, and calculus.
- Everyday Applications: Used in budgeting, shopping, cooking - making it practical.
- Problem Solving: Develops logical thinking valuable in academic and real-world scenarios.

# Quality check
[Thinking: Quality check: 100% confident]
[Thinking:   Decision: Ready to answer - comprehensively addressed main goal and sub-goals]
```

Notice:
- ✅ `[Thinking: ...]` blocks show internal reasoning (appear dimmed in terminal)
- ✅ Goal extraction and sub-goal tracking
- ✅ Confidence-based stopping (100% = done)
- ✅ Meta-cognitive reflection throughout

**Try it yourself:**
```bash
./hector --config local-configs/structured-reasoning.yaml
```

---

## Quick Start

### Prerequisites

```bash
# For local LLMs (optional)
ollama serve

# For vector search (optional)
docker run -p 6333:6333 qdrant/qdrant
```

### Installation

```bash
git clone https://github.com/kadirpekel/hector.git
cd hector
go build ./cmd/hector
```

### Your First Agent

Create `config.yaml`:

```yaml
agents:
  assistant:
    llm: "openai"

llms:
  openai:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

Run:

```bash
export OPENAI_API_KEY="your-key-here"
./hector --config config.yaml
```

**That's it!** Your declarative AI agent is running.

### Add Tools (Declaratively)

```yaml
agents:
  assistant:
    llm: "openai"

llms:
  openai:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"

# Just declare tools - Hector handles the rest
tools:
  repositories:
    - name: "local"
      type: "local"
      tools:
        - name: "execute_command"
          type: "command"
          config:
            allowed_commands: ["ls", "cat", "grep", "git"]
```

### Switch Reasoning Engines (Declaratively)

```yaml
agents:
  # Fast reasoning
  quick-agent:
    llm: "openai"
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 5

  # Thorough reasoning with visible thinking
  deep-agent:
    llm: "openai"
    reasoning:
      engine: "structured-reasoning"
      max_iterations: 10
      show_thinking: true
```

---

## Multi-Agent Teams

### Real-Time Streaming Multi-Agent Workflows

Hector features multi-agent orchestration with real-time streaming and live progress tracking.

#### See It In Action

**DAG Workflow** (Sequential execution with dependencies):

```bash
$ echo "Research and analyze renewable energy trends" | ./hector --workflow examples/workflow.yaml

🚀 Starting workflow: Research Analysis Workflow
------------------------------------------------------------

🤖 Starting agent: research-agent
[Streaming output in real-time...]
Researching renewable energy market trends...
Found 15 relevant sources...
Analyzing growth patterns...
✅ Agent research-agent completed in 12.3s

📊 Progress: 33.3% (1/3 steps)

🤖 Starting agent: analysis-agent
[Streaming output in real-time...]
Analyzing research findings...
Identifying key patterns...
Solar: 23% growth, Wind: 18% growth...
✅ Agent analysis-agent completed in 15.7s

📊 Progress: 66.7% (2/3 steps)

🤖 Starting agent: synthesis-agent
[Streaming output in real-time...]
Synthesizing insights...
Creating comprehensive report...
✅ Agent synthesis-agent completed in 10.2s

------------------------------------------------------------
✅ Workflow completed in 38.2s!
```

Notice:
- ✅ **Real-time output** - See agent responses as they're generated
- ✅ **Progress tracking** - Know exactly where you are in the workflow
- ✅ **Sequential execution** - Each agent completes before the next starts
- ✅ **Live timing** - Duration for each agent and total workflow

#### Two Execution Modes

**1. DAG Mode (Directed Acyclic Graph)**

Define dependencies and execution order:

```yaml
workflows:
  research-workflow:
    name: "Research Analysis"
    mode: "dag"  # Structured execution
    
    agents:
      - "researcher"    # Gathers information
      - "analyzer"      # Analyzes findings
      - "synthesizer"   # Creates report
    
    # Optional: Define explicit dependencies
    execution:
      dag:
        steps:
          - name: "research"
            agent: "researcher"
          - name: "analyze"
            agent: "analyzer"
            depends_on: ["research"]
          - name: "synthesize"
            agent: "synthesizer"
            depends_on: ["analyze"]
```

**Use Cases**:
- Research → Analysis → Report generation
- Data collection → Processing → Visualization
- Code generation → Testing → Documentation

**2. Autonomous Mode (AI-Driven)**

Let AI dynamically coordinate agents:

```yaml
workflows:
  autonomous-workflow:
    name: "Autonomous Problem Solver"
    mode: "autonomous"  # Dynamic coordination
    
    agents:
      - "planner"      # Creates execution plan
      - "executor-1"   # Executes tasks
      - "executor-2"   # Parallel execution
      - "validator"    # Validates results
    
    settings:
      max_iterations: 10
      quality_threshold: 0.8
```

**Use Cases**:
- Complex problem-solving with unknown steps
- Dynamic task decomposition
- Self-organizing agent teams

#### Streaming Architecture

Every workflow event is streamed in real-time:

```go
Event Types:
  • workflow_start   - Workflow begins
  • agent_start      - Agent starts execution
  • agent_output     - Real-time agent output (streamed)
  • agent_complete   - Agent finishes
  • progress         - Progress update (X/Y steps, percentage)
  • workflow_end     - Workflow completes
```

**Benefits**:
- **User Experience**: No waiting in the dark - see progress instantly
- **Debugging**: Watch exactly what each agent does
- **Monitoring**: Track execution in production
- **Cancellation**: Stop workflows mid-execution


#### Quick Start - Multi-Agent Workflow

**1. Create workflow config** (`my-workflow.yaml`):

```yaml
version: "1.0"
name: "my-workflow"

# Define your LLMs
llms:
  main:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"

# Define specialized agents
agents:
  researcher:
    name: "Research Agent"
    llm: "main"
    prompt:
      system_prompt: "You are a research specialist. Gather comprehensive information."
  
  analyzer:
    name: "Analysis Agent"
    llm: "main"
    prompt:
      system_prompt: "You are an analysis specialist. Identify patterns and insights."

# Define workflow
workflows:
  my-workflow:
    name: "Research and Analysis"
    mode: "dag"
    agents:
      - "researcher"
      - "analyzer"
```

**2. Run the workflow**:

```bash
export OPENAI_API_KEY="your-key"
echo "Analyze AI market trends" | ./hector --workflow my-workflow.yaml
```

**3. Watch it stream in real-time!**

---

## Architecture

### Declarative Configuration Flow

```
YAML Config → Validation → Component Registry → Agent Runtime
     ↓              ↓               ↓                  ↓
  Define        Check Rules    Providers Load     Execute
  What You      Defaults       Services Ready     Streaming
  Want          Applied        Extensions Reg     Real-time
```

### System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                   DECLARATIVE LAYER (YAML)                      │
│  User defines: agents, providers, tools, workflows, reasoning   │
└──────────────────────┬──────────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────────┐
│                   CONFIGURATION LAYER                           │
│  • Validation    • Defaults    • Provider Registry             │
└──────────────────────┬──────────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────────┐
│                   COMPONENT MANAGER                             │
│  • LLM Factory   • Tool Registry   • Extension Service          │
└──────────────────────┬──────────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────────┐
│                   AGENT RUNTIME                                 │
│  • Reasoning Engine   • Service Injection   • Streaming         │
└──────────────────────┬──────────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────────┐
│                   EXECUTION                                     │
│  • LLM Calls   • Tool Execution   • Context Management          │
└─────────────────────────────────────────────────────────────────┘
```

### Multi-Agent Workflow Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      TEAM ORCHESTRATION                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Team.ExecuteStreaming()                                        │
│       │                                                         │
│       ▼                                                         │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  WorkflowExecutor (DAG / Autonomous)                     │  │
│  │  • Streams events in real-time                           │  │
│  │  • Tracks progress (steps, percentage)                   │  │
│  │  • Coordinates agent execution                           │  │
│  └────────┬───────────────────────────────────┬──────────────┘  │
│           │                                   │                 │
│           ▼                                   ▼                 │
│  ┌─────────────────┐              ┌─────────────────┐          │
│  │  Agent 1        │              │  Agent 2        │          │
│  │  • Own LLM      │              │  • Own LLM      │          │
│  │  • Own Tools    │              │  • Own Tools    │          │
│  │  • Own Reasoning│              │  • Own Reasoning│          │
│  │  • Streams ──────────────────► • Streams        │          │
│  └─────────────────┘              └─────────────────┘          │
│           │                                   │                 │
│           └────────────┬──────────────────────┘                 │
│                        ▼                                        │
│               Event Stream (chan)                               │
│               • workflow_start                                  │
│               • agent_start                                     │
│               • agent_output  ◄── Real-time!                   │
│               • agent_complete                                  │
│               • progress                                        │
│               • workflow_end                                    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Key Features**:
- ✅ **AgentFactory** - Single source of truth for agent creation (no duplication)
- ✅ **Event-Based** - All workflow events streamed in real-time
- ✅ **Progress Tracking** - Accurate step counting and percentage calculation
- ✅ **Zero Coupling** - Each agent independent, fully isolated
- ✅ **Clean Architecture** - SOLID principles, dependency injection

### Agent Services Architecture

Every agent gets injected with services (dependency injection):

```
┌────────────────────────────────────────────────────────────────┐
│                         AGENT RUNTIME                          │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐   │
│  │   Reasoning  │────►│  Extension   │────►│     LLM      │   │
│  │    Engine    │     │   Service    │     │   Service    │   │
│  │              │     │              │     │              │   │
│  │ • Execute    │     │ • Register   │     │ • Generate   │   │
│  │ • Orchestrate│     │ • Process    │     │ • Stream     │   │
│  │ • Iterate    │     │ • Execute    │     │ • Mask       │   │
│  └──────────────┘     └──────────────┘     └──────────────┘   │
│         │                     │                     │          │
│         └─────────────────────┴─────────────────────┘          │
│                              │                                 │
└──────────────────────────────┼─────────────────────────────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        │                      │                      │
   ┌────▼────┐          ┌─────▼─────┐         ┌─────▼─────┐
   │  Tools  │          │  Context  │         │  History  │
   │Extension│          │  Service  │         │  Service  │
   │         │          │           │         │           │
   │• Execute│          │• Search   │         │• Store    │
   │• Command│          │• Vector   │         │• Retrieve │
   │• MCP    │          │• Index    │         │• Truncate │
   └─────────┘          └───────────┘         └───────────┘
```

### Extension System Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                      LLM STREAMING FLOW                        │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  Raw Chunks → Buffer → Marker Detection → Masking → Output    │
│                                                                │
│  "Let me help"  ┐                                             │
│  " you. TOOL_"  ├─► Accumulate ─► ContainsMarker() ─► Found! │
│  "CALLS: {...}" ┘        │              │              │      │
│                          │              │              │      │
│                    InputBuffer    ExtensionService   Stream   │
│                     (Efficient)     (Delegates)    (Masked)   │
│                                                                │
│  Features:                                                     │
│  ✓ Split-marker detection  ✓ Dynamic buffering               │
│  ✓ Zero allocations        ✓ Real-time masking               │
│  ✓ Generic (any extension) ✓ Production-optimized            │
└────────────────────────────────────────────────────────────────┘
```

---

## Configuration

> 📚 **For complete configuration documentation, see [CONFIGURATION.md](CONFIGURATION.md)**

### Quick Start Configuration

**Minimal Configuration:**
```yaml
agents:
  assistant:
    llm: "openai"

llms:
  openai:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

**With Tools:**
```yaml
agents:
  assistant:
    llm: "openai"

llms:
  openai:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"

tools:
  repositories:
    - name: "local"
      type: "local"
      tools:
        - name: "execute_command"
          type: "command"
          config:
            allowed_commands: ["ls", "cat", "grep", "git"]
```

**Reasoning Engines:**
```yaml
agents:
  # Fast reasoning (chain-of-thought)
  fast-agent:
    llm: "openai"
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 5

  # Thorough reasoning (structured-reasoning)
  thorough-agent:
    llm: "openai"
    reasoning:
      engine: "structured-reasoning"
      max_iterations: 10
      show_thinking: true    # See internal reasoning!
```

### Configuration Sections

All declaratively defined in YAML:

- **Global Settings** - Logging, performance, timeouts
- **LLM Providers** - OpenAI, Anthropic (Claude), Ollama
- **Databases** - Qdrant vector database
- **Embedders** - Ollama embeddings
- **Agents** - AI agent configuration
- **Reasoning Engines** - chain-of-thought vs structured-reasoning
- **Tools** - Command execution, MCP integration
- **Document Stores** - Knowledge base configuration
- **Workflows** - Multi-agent orchestration

See [CONFIGURATION.md](CONFIGURATION.md) for complete reference with all options, defaults, and best practices.

---

## Examples

> 📖 **For detailed examples documentation, see [examples/README.md](examples/README.md)**

Example configurations are available in the `examples/` directory, covering everything from minimal setups to complex multi-agent workflows.

---

## CLI Reference

### Single Agent Mode

```bash
./hector [options]

Options:
  --config string     YAML configuration file path
  --agent string      Agent name to use (defaults to first agent)
  --debug            Show technical details and debug info

Examples:
# Interactive mode
./hector --config config.yaml

# Single query mode (pipe input)
echo "What is AI?" | ./hector --config config.yaml

# Debug mode
./hector --config config.yaml --debug

# Specific agent
./hector --config config.yaml --agent researcher
```

### Multi-Agent Workflow Mode

```bash
./hector --workflow <config.yaml>

Options:
  --workflow string   Workflow configuration file path
  --debug            Show technical details and debug info

Examples:
# Run DAG workflow
echo "Research topic" | ./hector --workflow examples/workflow.yaml

# Run autonomous workflow
echo "Analyze data" | ./hector --workflow examples/advanced.yaml
```

### Interactive Commands

```bash
/help         # Show available commands
/tools        # List available tools
/quit         # Exit the application
```

---

## Supported Providers

All providers are **declaratively configured** - just change the YAML:

### LLM Providers

- **OpenAI**: gpt-4o, gpt-4o-mini, gpt-3.5-turbo
- **Anthropic**: Claude Sonnet 4.5, Claude Opus 4.1, Claude Haiku
- **Ollama**: All supported models (Llama, Mistral, CodeLlama, etc.)

**Switch providers declaratively:**
```yaml
# Use OpenAI
llms:
  main: { type: "openai", model: "gpt-4o" }

# Switch to Claude
llms:
  main: { type: "anthropic", model: "claude-sonnet-4.5-20250514" }

# Use local Ollama
llms:
  main: { type: "ollama", model: "llama3.2" }
```

### Database Providers

- **Qdrant**: Vector similarity search with collections and filtering

### Embedding Providers

- **Ollama**: nomic-embed-text, all-MiniLM-L6-v2, and other models

### Tool Repositories

- **Local**: Built-in command execution with sandboxing
- **MCP**: Model Context Protocol for external tools

---

## Why Hector?

### Declarative = Maintainable

**Change LLM provider?** Edit 1 line in YAML.
**Add reasoning?** Declare it, don't code it.
**Multi-agent workflow?** Define steps, Hector orchestrates.

### Built Right

- ✅ Real-time streaming for single and multi-agent workflows
- ✅ Sandboxed tool execution
- ✅ Clean architecture (SOLID principles)
- ✅ Service-oriented design with dependency injection

### Multi-Agent Support

- ✅ Real-time event streaming
- ✅ Live progress tracking
- ✅ DAG and Autonomous execution modes
- ✅ Efficient linear scaling

### Extensible

Every component is pluggable:
- Custom LLM providers
- Custom tools
- Custom reasoning engines
- Custom workflow executors
- Custom extensions

All through clean interfaces, no fork needed.

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">
  
**Hector** - Declarative AI Agent Framework

*Define once, deploy anywhere. Configuration over code.*

Built with Go • MIT License • Alpha Stage

</div>

