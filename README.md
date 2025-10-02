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
- [Architecture](#architecture)
- [Configuration](#configuration)
- [Examples](#examples)
- [CLI Reference](#cli-reference)
- [Supported Providers](#supported-providers)
- [License](#license)

## Core Philosophy

### 1. **Configuration Over Code**
Define your AI system's behavior, not its implementation. Focus on what you want, not how to build it.

### 2. **Composable by Design**
Mix and match LLM providers, reasoning engines, tools, and agents like building blocks. Everything is pluggable.

### 3. **Production-First**
Built for real-world deployments with streaming, error handling, monitoring, and zero-downtime updates.

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

### Multi-Agent System Capabilities

- **DAG Workflows**: Define dependencies, Hector executes in order
- **Autonomous Coordination**: AI-driven task planning and collaboration
- **Specialized Agents**: Each agent has its own LLM, reasoning, tools
- **Parallel Execution**: Configurable concurrency limits

### Universal Features

- **Single YAML Configuration**: Everything from providers to workflows
- **Zero-Config Mode**: Intelligent defaults (Ollama + local tools)
- **Provider Agnostic**: OpenAI, Anthropic (Claude), Ollama - switch declaratively
- **Extension System**: Pluggable tools, memory, custom capabilities
- **Clean Architecture**: SOLID principles, dependency injection, no tight coupling

---

## Recent Improvements

### Dual Reasoning Engines

**Two approaches, one interface** - Choose the right reasoning strategy for your task:

#### 1. Chain-of-Thought (Fast & Simple)
- Behavioral stopping: Tool call = continue, no tool call = stop
- Model agnostic: Works with GPT-4, Claude, Ollama
- Low token usage: Efficient for simple queries
- **Best for**: Quick queries, simple tool use, speed matters

#### 2. Structured Reasoning (Thorough & Transparent)
- Goal extraction: Automatically identifies main goal and sub-goals
- Meta-cognition: Self-reflection after each tool use
- Progress tracking: Shows accomplished vs pending goals
- Quality-based stopping: Confidence scoring (0-100%)
- **Thinking Mode**: See internal reasoning in grayed-out format
- **Best for**: Complex analysis, research tasks, quality > speed

**Thinking Mode Example:**
```
[Thinking: Goal identified - analyze weather impact on mood]
[Thinking: Sub-goal 1: Get current weather data]

I'll check the weather in Berlin...

[Thinking: Reflection: Got weather data. Confidence 70%]
[Thinking: Still need: Research mood correlations]

[Analysis continues...]

[Thinking: Quality check: 95% confident - ready to answer]
```

### Generic Extension System

**Pluggable capabilities beyond tools:**
- Tools extension: Execute commands, integrate APIs
- Memory extension: Store and recall facts (coming soon)
- Custom extensions: Add domain-specific capabilities

**Architecture:**
- Marker-based detection for reliable parsing
- Streaming support with real-time masking
- Zero-allocation optimizations
- Clean separation of concerns

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

### Available Examples

| Example | Description | Use Case |
|---------|-------------|----------|
| [zero-config.yaml](examples/zero-config.yaml) | Zero configuration | Quick start |
| [minimal.yaml](examples/minimal.yaml) | Minimal setup | Production single agent |
| [basic.yaml](examples/basic.yaml) | With tools & search | General purpose |
| [chain-of-thought.yaml](examples/chain-of-thought.yaml) | Fast reasoning | Speed matters |
| [reasoning-comparison.yaml](examples/reasoning-comparison.yaml) | Compare engines | Learn differences |
| [workflow.yaml](examples/workflow.yaml) | Multi-agent DAG | Structured workflows |
| [advanced.yaml](examples/advanced.yaml) | Enterprise | Complex systems |

### Quick Example

```bash
# Try the basic example
./hector --config examples/basic.yaml

# Try chain-of-thought reasoning
./hector --config examples/chain-of-thought.yaml

# Try multi-agent workflow
echo "Research renewable energy" | ./hector --workflow examples/workflow.yaml
```

See [examples/README.md](examples/README.md) for detailed descriptions, configuration breakdowns, and usage instructions for each example.

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

### Production-Ready

- ✅ Streaming with zero-allocation optimizations
- ✅ Error handling and retry policies
- ✅ Monitoring and logging
- ✅ Sandboxed tool execution
- ✅ Configurable concurrency limits

### Extensible

Every component is pluggable:
- Custom LLM providers
- Custom tools
- Custom reasoning engines
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

