# Hector Configuration Examples

This directory contains **4 essential examples** that demonstrate Hector's capabilities with the new unified configuration structure.

## Quick Reference

| Example | Use Case | Complexity | Key Features |
|---------|----------|------------|--------------|
| [`minimal.yaml`](#1-minimal) | Getting started | ⭐ | Zero-config, basic agent |
| [`basic.yaml`](#2-basic) | General purpose | ⭐⭐ | Tools, smart reasoning |
| [`workflow.yaml`](#3-workflow) | Multi-agent workflows | ⭐⭐⭐ | Specialized agents, DAG execution |
| [`advanced.yaml`](#4-advanced) | Enterprise features | ⭐⭐⭐⭐ | Autonomous workflows, advanced coordination |

## Examples

### 1. `minimal.yaml` - Getting Started
**Perfect for**: First-time users, quick testing, learning basics

```bash
hector --config examples/minimal.yaml
```

**What it demonstrates**:
- Ultra-minimal configuration with essential components
- Single agent with basic capabilities
- Local Ollama and Qdrant setup
- No tools or complex workflows

**Key insight**: Hector works with minimal configuration thanks to smart defaults and the unified config structure.

---

### 2. `basic.yaml` - General Purpose Agent
**Perfect for**: Daily use, tool integration, balanced capabilities

```bash
hector --config examples/basic.yaml
```

**What it demonstrates**:
- Single agent with enhanced capabilities
- Tool integration (command execution, search)
- Smart reasoning with self-reflection
- Document store configuration
- Multiple LLM provider options

**Key insight**: A single well-configured agent can handle most use cases with tools and smart reasoning.

---

### 3. `workflow.yaml` - Multi-Agent Workflows
**Perfect for**: Complex analysis, research projects, specialized tasks

```bash
hector --workflow examples/workflow.yaml
```

**What it demonstrates**:
- Multiple specialized agents (research, analysis, synthesis)
- DAG-based workflow execution
- Agent coordination and shared memory
- Step-by-step task delegation
- Different LLM configurations per agent

**Key insight**: Specialized agents working together can handle complex, multi-step tasks more effectively than a single agent.

---

### 4. `advanced.yaml` - Enterprise Features
**Perfect for**: Enterprise applications, autonomous systems, advanced coordination

```bash
hector --workflow examples/advanced.yaml
```

**What it demonstrates**:
- Autonomous workflow execution
- Advanced agent coordination
- Multiple workflow types (autonomous and DAG)
- Enterprise-grade logging and performance settings
- Comprehensive tool and document store configuration

**Key insight**: Hector can operate autonomously, making decisions about task delegation and workflow execution.

## Configuration Structure

### Unified Configuration
All examples use the new unified `HectorConfig` structure:

```yaml
version: "1.0"
name: "hector-example"
description: "Example configuration"

# Global system settings
global:
  logging: { level: "info", format: "json" }
  performance: { max_concurrency: 4, timeout: "15m" }

# Provider configurations (LLMs, Databases, Embedders)
providers:
  llms: { ollama-main: {...}, openai-gpt4: {...} }
  databases: { qdrant-main: {...} }
  embedders: { ollama-embed: {...} }

# Agent definitions
agents:
  main-agent: { llm: "ollama-main", database: "qdrant-main", ... }

# Workflow definitions
workflows:
  my-workflow: { mode: "dag", agents: [...], execution: {...} }

# Tool configurations
tools:
  repositories: [{ name: "local", type: "local", tools: [...] }]

# Document store configurations
document_stores:
  web-docs: { source: "directory", path: "./docs", ... }
```

### Key Features

#### Provider System
- **LLM Providers**: Ollama, OpenAI, and more
- **Database Providers**: Qdrant for vector storage
- **Embedder Providers**: Ollama-based text embedding

#### Agent Configuration
- **Reference-based**: Agents reference providers by name
- **Specialized**: Different agents for different roles
- **Configurable**: Custom prompts, reasoning, and search settings

#### Workflow Execution
- **DAG Mode**: Step-by-step execution with dependencies
- **Autonomous Mode**: AI-driven dynamic coordination
- **Shared Resources**: Memory, cache, and coordination

#### Tool Integration
- **Local Tools**: Command execution, search capabilities
- **MCP Tools**: External tool integration (future)
- **Configurable**: Security, timeouts, and capabilities

## Getting Started

1. **Start simple**: Begin with `minimal.yaml`
2. **Add capabilities**: Move to `basic.yaml` for tool integration
3. **Scale up**: Use `workflow.yaml` for multi-agent workflows
4. **Go advanced**: Try `advanced.yaml` for autonomous systems

## Environment Setup

### Required Services
- **Ollama**: For local LLM and embedding models
- **Qdrant**: For vector database storage

### Environment Variables
```bash
# Optional: OpenAI API key for advanced examples
export OPENAI_API_KEY="your-openai-api-key"

# Optional: Qdrant API key
export QDRANT_API_KEY="your-qdrant-api-key"
```

### Quick Start
```bash
# Start required services
ollama serve
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant

# Run examples
hector --config examples/minimal.yaml
hector --workflow examples/workflow.yaml
```

## Tips

- **Local Setup**: Examples work with local Ollama and Qdrant
- **Customization**: Mix and match configurations from different examples
- **Debugging**: Set `logging.level: "debug"` for detailed output
- **Performance**: Adjust `max_concurrency` and `timeout` based on your needs
- **Tools**: Enable/disable tools based on your security requirements

## Migration from Old Examples

The new examples use a completely different structure:

### Old Structure (Deprecated)
```yaml
llm:
  name: "openai"
  api_key: "${OPENAI_API_KEY}"

reasoning:
  max_iterations: 5

tools:
  repositories: [...]
```

### New Structure (Current)
```yaml
providers:
  llms:
    openai-main:
      type: "openai"
      api_key: "${OPENAI_API_KEY}"

agents:
  main-agent:
    llm: "openai-main"
    reasoning:
      max_iterations: 5

tools:
  repositories: [...]
```

The new structure provides better organization, reusability, and scalability.
