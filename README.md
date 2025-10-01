<div align="center">
  <img src="hector_logo.png" alt="Hector Logo">
</div>

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Beta-orange.svg)](https://github.com/kadirpekel/hector)

> **Beta Release**: Hector is a mature, production-ready framework for building both single agents and multi-agent systems. Core features are stable with comprehensive tooling, reasoning engines, and workflow orchestration.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Configuration Reference](#configuration-reference)
- [Examples](#examples)
- [CLI Reference](#cli-reference)
- [Supported Providers](#supported-providers)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Features

### Single Agent Capabilities

- **Two Reasoning Engines**: Choose between fast chain-of-thought or thorough structured-reasoning with visible thinking blocks
  - **chain-of-thought**: Fast, behavioral-signal-based (best for simple queries)
  - **structured-reasoning**: Goal-oriented with meta-cognition and quality evaluation (best for complex analysis)
- **Thinking Mode**: See internal reasoning in grayed-out blocks, just like Claude in Cursor
- **Generic Extension System**: Pluggable extension framework supporting tools, memory, and custom extensions with streaming support
- **Optimized Streaming**: Real-time output with intelligent buffering, marker detection, and zero-allocation optimizations
- **Comprehensive Tool System**: Secure command execution, MCP integration, and extensible tool repositories with sandboxing
- **Context Management**: Vector search with Qdrant, conversation history, and document stores with automatic indexing
- **Provider Architecture**: Pluggable LLM (OpenAI, Anthropic/Claude, Ollama), database (Qdrant), and embedder providers with zero-config defaults

### Multi-Agent System Capabilities

- **DAG Workflows**: Deterministic execution with dependencies, parallel processing, and step-by-step coordination
- **Autonomous Coordination**: AI-driven task planning, dynamic agent collaboration, and adaptive workflow execution
- **Team Management**: Coordinator-led task delegation, resource management, and shared state coordination
- **Specialized Agents**: Role-based agents with distinct capabilities, configurations, and reasoning strategies
- **Parallel Execution**: Multiple agents working simultaneously with configurable concurrency limits

### Universal Features

- **Unified Configuration**: Single YAML file defines everything from providers to workflows
- **Zero-Config Support**: Intelligent defaults enable immediate usage with minimal configuration
- **Production Ready**: Robust error handling, optimized streaming, comprehensive logging, and monitoring
- **Extensible Architecture**: Generic extension system for tools, memory, and custom capabilities
- **Clean Separation**: SOLID principles with clear layer boundaries and no tight coupling

## Recent Improvements

### Dual Reasoning Engines (v2.0) - October 2024

**Two Approaches, One Interface** - Choose the right reasoning strategy for your task:

#### 1. Chain-of-Thought (Fast & Simple)
- ✅ **Behavioral Stopping**: Tool call = continue, no tool call = stop
- ✅ **Model Agnostic**: Works with gpt-4o, Claude Sonnet 4.5, gpt-4o-mini, Ollama
- ✅ **Low Token Usage**: Efficient for simple queries
- ✅ **Best for**: Quick queries, simple tool use, speed matters

#### 2. Structured Reasoning (Thorough & Transparent)
- ✅ **Goal Extraction**: Automatically identifies main goal and sub-goals
- ✅ **Meta-Cognition**: Self-reflection after each tool use
- ✅ **Progress Tracking**: Shows accomplished vs pending goals
- ✅ **Quality-Based Stopping**: Confidence scoring (0-100%) and quality gates
- ✅ **Thinking Mode**: See internal reasoning in grayed-out format (like Claude in Cursor)
- ✅ **Best for**: Complex analysis, research tasks, when quality > speed

**Thinking Mode Example:**
```
[Thinking: Goal identified - get weather and analyze mood impact]
[Thinking: Sub-goal 1: Check current weather]

I'll check the weather in Berlin...

[Thinking: Reflection: Got weather data. Confidence 70%]
[Thinking: Still need: Mood analysis]

[Analysis continues...]

[Thinking: Quality check: 95% confident]
```

See [REASONING_ENGINES.md](REASONING_ENGINES.md) for detailed comparison and usage guide.

### Extension System (v1.1)

**Generic Extension Framework** - A production-ready extension system that enables pluggable capabilities:
- ✅ **Tools Extension**: Execute commands and integrate external APIs
- ✅ **Memory Extension**: Coming soon - store and recall facts across conversations
- ✅ **Custom Extensions**: Add domain-specific capabilities with minimal code

**Key Features:**
- Marker-based detection for reliable parsing
- Streaming support with real-time masking
- Clean separation of concerns (no tight coupling)
- Zero-allocation optimizations for production performance
- Generic utilities for parsing, validation, and field extraction
- Service-oriented architecture with dependency injection

### Streaming Optimizations (v1.0)

**Production-Grade Streaming** with intelligent buffering and masking:
- ✅ **Split-marker detection**: Handles markers split across multiple chunks
- ✅ **Dynamic buffer sizing**: Automatically adjusts to longest registered marker
- ✅ **Zero allocations**: Optimized to avoid unnecessary memory allocations
- ✅ **Real-time masking**: User-friendly labels instead of raw JSON

**Performance:**
- Minimal memory overhead with integer-based tracking
- Cached marker lengths to avoid recalculation
- Single `ProcessResponse()` call per streaming session
- Optimized control flow with early returns

### Configuration Simplification (v1.0)

**Zero-Config Support** - Start with sensible defaults, extend as needed:
- ✅ **Automatic defaults**: LLM, database, and embedder created if not specified
- ✅ **Conditional services**: Only initialize services that are actually used
- ✅ **Simplified structure**: Flattened configuration hierarchy
- ✅ **Progressive enhancement**: Add complexity only when required

**Architecture Improvements:**
- Clean separation between branding and internal code
- SOLID principles with dependency injection
- No tight coupling between layers
- Extensible without breaking changes

## Quick Start

### Prerequisites

Hector requires the following services:

```bash
# Start Ollama (for local LLM and embeddings)
ollama serve

# Start Qdrant (for vector database)
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### Installation

```bash
git clone https://github.com/kadirpekel/hector.git
cd hector
go build ./cmd/hector
```

### Basic Usage

Create a simple configuration file (`my-agent.yaml`):

```yaml
version: "1.0"
name: "my-assistant"
description: "A helpful AI assistant"

# LLM Configuration (optional - defaults to Ollama if not specified)
llms:
  main-llm:
    type: "ollama"
    model: "llama3.2"
    host: "http://localhost:11434"
    temperature: 0.7

# Agent Configuration
agents:
  assistant:
    name: "AI Assistant"
    description: "A helpful AI assistant with tools"
    llm: "main-llm"
    
    prompt:
      system_prompt: "You are a helpful AI assistant."
      include_history: true
    
    reasoning:
      engine: "default"
      enable_streaming: true

# Optional: Add tools for command execution
tools:
  repositories:
    - name: "local"
      type: "local"
```

**Even simpler - Zero Config:**

```yaml
version: "1.0"
name: "minimal-agent"

agents:
  assistant:
    name: "AI Assistant"
```

This automatically creates default LLM (Ollama), enables tools, and streaming!

Run your agent:

```bash
# Interactive mode with streaming
./hector --config my-agent.yaml

# Single query mode
echo "What is artificial intelligence?" | ./hector --config my-agent.yaml --no-stream

# Debug mode with detailed output
./hector --config my-agent.yaml --debug
```

### Try the Examples

```bash
# Minimal configuration (zero-config)
./hector --config examples/minimal.yaml

# Basic agent with tools
./hector --config examples/basic.yaml

# Chain-of-thought reasoning engine
echo "What are the implications of AI in healthcare?" | ./hector --config examples/chain-of-thought.yaml

# Multi-agent workflow
echo "Research renewable energy benefits" | ./hector --workflow examples/workflow.yaml

# Advanced autonomous workflow
echo "Analyze market trends" | ./hector --workflow examples/advanced.yaml
```

## Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        HECTOR FRAMEWORK                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐  │
│  │ Configuration   │    │ Component       │    │ Agent       │  │
│  │ System          │───►│ Manager         │───►│ Runtime     │  │
│  │                 │    │                 │    │             │  │
│  │ • YAML Config   │    │ • LLM Registry  │    │ • Reasoning │  │
│  │ • Validation    │    │ • DB Registry   │    │ • Tools     │  │
│  │ • Defaults      │    │ • Tool Registry │    │ • Context   │  │
│  └─────────────────┘    └─────────────────┘    └─────────────┘  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                      MULTI-AGENT LAYER                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐  │
│  │ Team            │    │ Workflow        │    │ Coordination│  │
│  │ Coordinator     │───►│ Executor        │───►│ Engine      │  │
│  │                 │    │                 │    │             │  │
│  │ • Agent Pool    │    │ • DAG Mode      │    │ • Autonomous│  │
│  │ • Task Queue    │    │ • Dependencies  │    │ • Planning  │  │
│  │ • Load Balance  │    │ • Parallel Exec │    │ • Consensus │  │
│  └─────────────────┘    └─────────────────┘    └─────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Agent Architecture

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
   │  Tools  │          │  Memory   │         │  Custom   │
   │Extension│          │ Extension │         │ Extension │
   │         │          │           │         │           │
   │• Execute│          │• Store    │         │• Process  │
   │• Command│          │• Recall   │         │• Execute  │
   │• MCP    │          │• Search   │         │• Custom   │
   └─────────┘          └───────────┘         └───────────┘
        │                      │                      │
        └──────────────────────┴──────────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Service Layer     │
                    ├─────────────────────┤
                    │ • LLM Providers     │
                    │ • Databases         │
                    │ • Embedders         │
                    │ • Tool Registry     │
                    └─────────────────────┘
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

## Configuration Reference

### Complete Configuration Structure

```yaml
version: "1.0"
name: "hector-config"
description: "Complete Hector configuration"

# Global Settings
global:
  logging:
    level: "info"                 # debug, info, warn, error
    format: "json"                # json, text
    output: "stdout"              # stdout, stderr, file
  performance:
    max_concurrency: 4            # Maximum concurrent operations
    timeout: "15m"                # Global timeout

# LLM Providers
llms:
  openai-main:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    host: "https://api.openai.com/v1"
    temperature: 0.7              # 0.0-2.0
    max_tokens: 2000
    timeout: 120                  # seconds
  
  ollama-main:
    type: "ollama"
    model: "llama3.2"
    host: "http://localhost:11434"
    temperature: 0.7
    max_tokens: 2000
    timeout: 60

# Database Providers
databases:
  qdrant-main:
    type: "qdrant"
    host: "localhost"
    port: 6333
    api_key: "${QDRANT_API_KEY:-}"
    timeout: 30
    use_tls: false
    insecure: false

# Embedder Providers
embedders:
  ollama-embed:
    type: "ollama"
    model: "nomic-embed-text"
    host: "http://localhost:11434"
    dimension: 768
    timeout: 60
    max_retries: 3

# Agent Definitions
agents:
  main-agent:
    name: "Main Agent"
    description: "Primary AI agent"
    llm: "openai-main"
    database: "qdrant-main"
    embedder: "ollama-embed"
    document_stores: ["web-docs"]
    
    # Prompt Configuration
    prompt:
      system_prompt: "You are an expert AI assistant."
      include_context: true       # Include search results
      include_history: true       # Include conversation history
      include_tools: true         # Include tool descriptions
      max_context_length: 4000
    
    # Reasoning Configuration
    reasoning:
      engine: "default"           # default reasoning engine with extensions
      max_iterations: 5
      enable_streaming: true
      quality_threshold: 0.8      # 0.0-1.0
    
    # Search Configuration
    search:
      models:
        - name: "documents"
          collection: "docs"
          default_top_k: 10
          max_top_k: 100
        - name: "code"
          collection: "code_docs"
          default_top_k: 10
          max_top_k: 50
      top_k: 10
      threshold: 0.7              # 0.0-1.0
      max_context_length: 4000

# Tool System
tools:
  default_repo: "local"
  repositories:
    - name: "local"
      type: "local"
      description: "Built-in tools"
      tools:
        - name: "command_executor"
          type: "command"
          enabled: true
          config:
            command_config:
              allowed_commands: ["ls", "cat", "head", "tail", "pwd", "git", "curl"]
              working_directory: "./"
              max_execution_time: "30s"
              enable_sandboxing: true
        
        - name: "web_search"
          type: "search"
          enabled: true
          config:
            search_config:
              document_stores: ["web-docs"]
              default_limit: 10
              max_limit: 50
              enabled_search_types: ["content", "file", "function"]

# Document Stores
document_stores:
  web-docs:
    name: "Web Documents"
    source: "directory"
    path: "./docs"
    include_patterns: ["*.md", "*.txt", "*.html", "*.pdf"]
    exclude_patterns: ["**/node_modules/**", "**/.git/**"]
    watch_changes: true
    max_file_size: 10485760      # 10MB

# Multi-Agent Workflows
workflows:
  research-workflow:
    name: "Research Workflow"
    description: "Multi-agent research process"
    mode: "dag"                   # dag, autonomous
    
    agents:
      - "research-agent"
      - "analysis-agent"
      - "synthesis-agent"
    
    shared:
      memory:
        type: "in-memory"
      cache:
        type: "in-memory"
        ttl: "1h"
    
    execution:
      dag:
        steps:
          - name: "research_phase"
            agent: "research-agent"
            input: "${user_input}"
            output: "research_results"
          
          - name: "analysis_phase"
            agent: "analysis-agent"
            input: "${research_results}"
            output: "analysis_insights"
            depends_on: ["research_phase"]
          
          - name: "synthesis_phase"
            agent: "synthesis-agent"
            input: "${analysis_insights}"
            output: "final_report"
            depends_on: ["analysis_phase"]
    
    settings:
      max_concurrency: 3
      timeout: "20m"
      retry_policy:
        max_retries: 3
        backoff: "5s"
```

### Environment Variables

Use environment variables for sensitive configuration:

```bash
# OpenAI API Key
export OPENAI_API_KEY="your-openai-api-key"

# Qdrant API Key (optional)
export QDRANT_API_KEY="your-qdrant-api-key"

# Custom endpoints
export OLLAMA_HOST="http://localhost:11434"
export QDRANT_HOST="localhost"
export QDRANT_PORT="6333"
```

### Extension System

Hector's generic extension system allows you to add custom capabilities beyond tools:

#### How Extensions Work

1. **Registration**: Extensions register with the `ExtensionService`
2. **Prompt Inclusion**: Extension capabilities are automatically included in LLM prompts
3. **Response Processing**: LLM responses are parsed for extension markers
4. **Execution**: Extensions execute and return results
5. **Result Integration**: Results flow back to the LLM for natural responses

#### Built-in Extensions

**Tools Extension** (`TOOL_CALLS:`)
```yaml
# Automatically enabled when tools are configured
tools:
  repositories:
    - name: "local"
      type: "local"
```

#### Creating Custom Extensions

Extensions can be added for any capability:
- **Memory**: Store and recall facts across conversations
- **Knowledge Graph**: Semantic relationship management
- **Code Analysis**: Static analysis and code understanding
- **Web Search**: External API integration
- **Custom**: Any domain-specific capability

See [Extension System Documentation](EXTENSION_SYSTEM.md) for implementation details.

### Reasoning Engines

#### Default Engine
Clean, service-oriented reasoning with extension support:

```yaml
reasoning:
  engine: "default"
  max_iterations: 5
  enable_streaming: true
  quality_threshold: 0.7
```

**Features:**
- ✅ Automatic extension discovery and execution
- ✅ Intelligent result routing (direct vs. analyzed)
- ✅ Real-time streaming with marker masking
- ✅ Conversation history management
- ✅ Context-aware prompt building

#### Chain-of-Thought Engine
Advanced recursive reasoning with LLM-controlled stopping:

```yaml
reasoning:
  engine: "chain-of-thought"
  enable_streaming: true
```

**Features:**
- ✅ Recursive self-calling capability for deep analysis
- ✅ LLM-controlled stopping (non-deterministic)
- ✅ Meta-cognitive reasoning and problem decomposition
- ✅ Alternative approach exploration
- ✅ Reasoning verification and validation
- ✅ Safety mechanism with recursion depth limit
- ✅ Real-time streaming for recursive calls

### Tool Configuration

#### Local Tools (Built-in)

```yaml
tools:
  repositories:
    - name: "local"
      type: "local"
      tools:
        - name: "execute_command"
          type: "command"
          enabled: true
          config:
            command_config:
              allowed_commands: ["ls", "cat", "head", "tail", "pwd", "git", "curl"]
              working_directory: "./"
              max_execution_time: "30s"
              enable_sandboxing: true
```

**Features:**
- ✅ Secure command execution with allowlist
- ✅ Sandboxing for safety
- ✅ Configurable timeout and working directory
- ✅ Automatic tool discovery and registration

#### MCP Tools (Model Context Protocol)

```yaml
# Configure via environment variables
# HECTOR_MCP_SERVERS={"name": "mcp-server", "url": "https://mcp.example.com"}
```

**Features:**
- ✅ External tool integration via MCP protocol
- ✅ Dynamic tool discovery
- ✅ Secure API communication
- ✅ Automatic registration

#### Tool Call Behavior

Tools support intelligent result routing:

```yaml
# In tool calls, the LLM decides:
# - display_direct: true  → Show raw output to user
# - display_direct: false → LLM analyzes first, then responds

# Example: File listing (direct)
TOOL_CALLS:
{"tool": "execute_command", "params": {"command": "ls"}, 
 "label": "📂 Listing files...", "display_direct": true}

# Example: Weather data (analyzed)
TOOL_CALLS:
{"tool": "weather_api", "params": {"location": "Berlin"}, 
 "label": "🌤️ Getting weather...", "display_direct": false}
```

### Multi-Agent Workflows

#### DAG Workflow

```yaml
workflows:
  research-pipeline:
    name: "Research Pipeline"
    mode: "dag"
    
    agents: ["researcher", "analyzer", "writer"]
    
    execution:
      dag:
        steps:
          - name: "research"
            agent: "researcher"
            input: "${user_input}"
            output: "research_data"
          
          - name: "analyze"
            agent: "analyzer"
            input: "${research_data}"
            output: "analysis"
            depends_on: ["research"]
          
          - name: "write_report"
            agent: "writer"
            input: "${analysis}"
            output: "final_report"
            depends_on: ["analyze"]
```

#### Autonomous Workflow

```yaml
workflows:
  autonomous-research:
    name: "Autonomous Research"
    mode: "autonomous"
    
    agents: ["researcher", "analyzer", "writer"]
    
    execution:
      autonomous:
        goal: "Produce comprehensive research report"
        strategy: "dynamic"
        max_iterations: 10
        coordinator_llm: "openai-coordinator"
        termination_conditions:
          max_duration: "30m"
          quality_threshold: 0.9
```

## Examples

### Simple Chat Agent

```yaml
version: "1.0"
name: "chat-bot"

llms:
  main:
    type: "ollama"
    model: "llama3.2"
    host: "http://localhost:11434"

agents:
  chatbot:
    name: "Chat Bot"
    llm: "main"
    prompt:
      system_prompt: "You are a helpful assistant."
      include_history: true
```

### Document Q&A Agent

```yaml
version: "1.0"
name: "document-qa"

llms:
  main:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"

databases:
  vector-db:
    type: "qdrant"
    host: "localhost"
    port: 6333

embedders:
  embedder:
    type: "ollama"
    model: "nomic-embed-text"
    host: "http://localhost:11434"

agents:
  qa-agent:
    name: "Document Expert"
    llm: "main"
    database: "vector-db"
    embedder: "embedder"
    
    search:
      models:
        - name: "docs"
          collection: "documents"
          default_top_k: 5
    
    prompt:
      system_prompt: "Answer questions using the provided context."
      include_context: true

document_stores:
  docs:
    name: "Documents"
    source: "directory"
    path: "./documents"
    include_patterns: ["*.md", "*.txt", "*.pdf"]
```

### Tool-Enabled Agent

```yaml
version: "1.0"
name: "tool-assistant"

llms:
  main:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"

agents:
  assistant:
    name: "Assistant with Tools"
    llm: "main"
    
    prompt:
      system_prompt: "You are an assistant with access to tools."
      include_tools: true
      include_history: true

tools:
  repositories:
    - name: "local"
      type: "local"
      tools:
        - name: "command_executor"
          type: "command"
          config:
            command_config:
              allowed_commands: ["ls", "cat", "head", "pwd", "git"]
              enable_sandboxing: true
```

## CLI Reference

### Single Agent Mode

```bash
./hector [options] [config-name]

Options:
  --config string     YAML configuration file path
  --agent string      Agent name to use (defaults to first agent)
  --debug            Show technical details and debug info
  --no-stream        Disable streaming output

Examples:
# Interactive mode with config file
./hector --config my-agent.yaml

# Interactive mode with named config
./hector my-config

# Debug mode
./hector --config my-agent.yaml --debug

# Non-streaming mode
./hector --config my-agent.yaml --no-stream

# Pipe input for single queries
echo "What is AI?" | ./hector --config my-agent.yaml

# Zero-config mode (uses defaults)
./hector
```

### Multi-Agent Workflow Mode

```bash
./hector --workflow <workflow-config> [options]

Options:
  --workflow string   Workflow configuration file path
  --debug            Show technical details and debug info

Examples:
# Run DAG workflow
echo "Research renewable energy" | ./hector --workflow examples/workflow.yaml

# Run autonomous workflow
echo "Analyze market trends" | ./hector --workflow examples/advanced.yaml

# Interactive workflow
./hector --workflow examples/workflow.yaml
```

### Interactive Commands

```bash
/help         # Show available commands
/tools        # List available tools
/quit         # Exit the application
```

## Supported Providers

### LLM Providers

- **OpenAI**: gpt-4o, gpt-4o-mini, gpt-3.5-turbo
- **Anthropic**: Claude Sonnet 4.5, Claude Opus 4.1, Claude Sonnet 3.7, Claude Haiku
- **Ollama**: All supported models (Llama, Mistral, CodeLlama, etc.)

**Model Recommendations:**
- **Production (best quality)**: gpt-4o or Claude Sonnet 4.5
- **Cost-effective**: gpt-4o-mini (10-30x cheaper)
- **Local/offline**: Ollama models

See [MODEL_RECOMMENDATIONS.md](MODEL_RECOMMENDATIONS.md) for detailed comparison.

### Database Providers

- **Qdrant**: Vector similarity search with collections and filtering

### Embedding Providers

- **Ollama**: nomic-embed-text, all-MiniLM-L6-v2, and other embedding models

### Tool Repositories

- **Local**: Built-in command execution tools with sandboxing
- **MCP**: Model Context Protocol integration for external tools (e.g., Composio)

## Development

### Building from Source

```bash
git clone https://github.com/kadirpekel/hector.git
cd hector
go mod download
go build ./cmd/hector
```

### Running Tests

```bash
go test ./...
```

### Project Structure

```
hector/
├── cmd/hector/          # Main CLI application
├── agent/               # Agent implementation and services
│   ├── services.go      # LLM, Prompt, History, Context services
│   └── factory.go       # Agent factory with dependency injection
├── reasoning/           # Reasoning engines and extensions
│   ├── default.go       # Default reasoning engine
│   ├── chain_of_thought.go  # Chain-of-thought reasoning engine
│   ├── extension_service.go  # Generic extension system
│   ├── reasoning_extension.go  # Chain-of-thought extension
│   ├── tool_extension.go     # Tool extension implementation
│   └── interfaces.go    # Service interfaces
├── config/              # Configuration types and loading
├── context/             # Context management and search
├── databases/           # Database provider implementations
├── embedders/           # Embedding provider implementations
├── llms/                # LLM provider implementations
├── tools/               # Tool system and repositories
│   ├── local.go         # Local command tools
│   ├── mcp.go           # MCP protocol integration
│   └── registry.go      # Tool registry
├── team/                # Multi-agent team coordination
├── workflow/            # Workflow execution engines
├── component/           # Component manager
├── registry/            # Provider registries
├── utils/               # Utility functions
├── examples/            # Configuration examples
└── local-configs/       # Local development configurations
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests if applicable
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">
  **Hector** - Declarative AI Agent Framework
  
  *Built with Go for production environments*
</div>