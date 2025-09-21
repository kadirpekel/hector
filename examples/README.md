# Hector Configuration Examples

This directory contains **5 essential examples** that demonstrate Hector's core capabilities with progressive complexity.

## Quick Reference

| Example | Use Case | Complexity | Key Features |
|---------|----------|------------|--------------|
| [`minimal.yaml`](#1-minimal) | Getting started | ⭐ | Zero-config, just API key |
| [`basic.yaml`](#2-basic) | General purpose | ⭐⭐ | Tools, smart reasoning |
| [`multi-agent.yaml`](#3-multi-agent) | Complex workflows | ⭐⭐⭐ | Specialized agents, inheritance |
| [`dynamic-reasoning.yaml`](#4-dynamic-reasoning) | Creative problems | ⭐⭐⭐⭐ | AI-driven adaptation |
| [`document-processing.yaml`](#5-document-processing) | Knowledge management | ⭐⭐⭐ | PDF/Word/Text ingestion |

## Examples

### 1. `minimal.yaml` - Getting Started
**Perfect for**: First-time users, quick testing, learning basics

```bash
hector --config examples/minimal.yaml
```

**What it demonstrates**:
- Ultra-minimal configuration (just API key)
- Intelligent defaults for all components
- Zero-config alternative using local Ollama

**Key insight**: Hector works with minimal configuration thanks to smart provider defaults.

---

### 2. `basic.yaml` - General Purpose Agent
**Perfect for**: Daily use, tool integration, balanced capabilities

```bash
hector --config examples/basic.yaml
```

**What it demonstrates**:
- MCP tool integration (weather, web search, etc.)
- Intelligent reasoning with meta-cognition
- Document support for knowledge queries
- Configuration inheritance

**Key insight**: A single configuration handles most use cases with tools and smart reasoning.

---

### 3. `multi-agent.yaml` - Specialized Workflows
**Perfect for**: Complex analysis, research projects, enterprise applications

```bash
hector --config examples/multi-agent.yaml
```

**What it demonstrates**:
- Multi-step workflows with specialized agents
- Configuration inheritance and overrides
- Agent specialization (Research → Analysis → Synthesis)
- Different LLM settings per agent role

**Key insight**: Each workflow step is a complete agent with its own capabilities and configuration.

---

### 4. `dynamic-reasoning.yaml` - Creative Problem Solving
**Perfect for**: Open-ended problems, research, strategy, creative tasks

```bash
hector --config examples/dynamic-reasoning.yaml
```

**What it demonstrates**:
- AI-driven adaptive reasoning
- Self-reflection and meta-reasoning
- Goal evolution during execution
- Dynamic tool selection
- Streaming reasoning process

**Key insight**: AI can autonomously adapt its approach and reasoning strategy based on the problem complexity.

**Great for queries like**:
- "Design a sustainable city of the future"
- "Create a comprehensive climate action plan"
- "Develop a novel approach to education reform"

---

### 5. `document-processing.yaml` - Knowledge Management
**Perfect for**: Document analysis, knowledge bases, research assistance

```bash
hector --config examples/document-processing.yaml
```

**What it demonstrates**:
- Multi-format document ingestion (PDF, Word, Text, Markdown)
- Local and cloud (S3) document sources
- Enhanced search and retrieval
- Document-optimized reasoning

**Key insight**: Hector can automatically process and understand various document formats for intelligent search and analysis.

**Usage workflow**:
1. Configure document paths
2. Sync documents: `/sync-model documents`
3. Search: `/search "your query" documents`

## Configuration Patterns

### Progressive Complexity
Each example builds on the previous one:
- **Minimal** → **Basic**: Adds tools and reasoning
- **Basic** → **Multi-Agent**: Adds specialized workflow steps
- **Multi-Agent** → **Dynamic**: Adds AI-driven adaptation
- **Any** → **Document Processing**: Adds document intelligence

### Common Patterns
- **Root-level inheritance**: All examples use configuration inheritance
- **Provider defaults**: Minimal explicit configuration, maximum smart defaults
- **Modular design**: Each component can be independently configured
- **Tool integration**: MCP protocol for external tool access

## Getting Started

1. **Start simple**: Begin with `minimal.yaml`
2. **Add tools**: Move to `basic.yaml` for tool integration
3. **Scale up**: Use `multi-agent.yaml` for complex workflows
4. **Go creative**: Try `dynamic-reasoning.yaml` for open-ended problems
5. **Add documents**: Use `document-processing.yaml` for knowledge work

## Tips

- **API Keys**: Replace `YOUR_*_API_KEY_HERE` with actual keys
- **Local Setup**: Use `hector` (no config) for local Ollama + Qdrant
- **Customization**: Mix and match configurations from different examples
- **Debugging**: Add `verbose: true` to any workflow for detailed output