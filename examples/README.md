# Hector Configuration Examples

This directory contains working configuration examples demonstrating different features and use cases of Hector.

## Quick Reference

| Example | Description | Best For |
|---------|-------------|----------|
| [zero-config.yaml](#zero-config) | Zero configuration (all defaults) | Getting started quickly |
| [minimal.yaml](#minimal) | Minimal configuration | Simple single-agent use |
| [minimal-config.yaml](#minimal-config) | Minimal with explicit config | Learning configuration structure |
| [basic.yaml](#basic) | General purpose with tools | Production single agent |
| [advanced.yaml](#advanced) | Enterprise multi-agent | Complex workflows |
| [chain-of-thought.yaml](#chain-of-thought) | Fast reasoning engine | Simple queries, speed matters |
| [reasoning-comparison.yaml](#reasoning-comparison) | Compare both reasoning engines | Understanding reasoning differences |
| [workflow.yaml](#workflow) | DAG workflow | Structured multi-agent tasks |
| [progressive-config.yaml](#progressive) | Progressive enhancement | Learning step-by-step |

---

## Zero Config

**File:** `zero-config.yaml`

**Purpose:** Demonstrate Hector's zero-configuration capability with intelligent defaults.

**Features:**
- No explicit LLM configuration (defaults to Ollama)
- No database or embedder needed
- Automatic defaults for all settings

**Configuration:**
```yaml
agents:
  assistant:
    name: "AI Assistant"
```

**Usage:**
```bash
./hector --config examples/zero-config.yaml
```

**Requirements:**
- Ollama running on localhost:11434

**What Gets Created Automatically:**
- LLM provider (Ollama with llama3.2)
- Streaming enabled
- History management
- Basic prompt configuration

---

## Minimal

**File:** `minimal.yaml`

**Purpose:** Minimal configuration with explicit services for production use.

**Features:**
- Explicit LLM, database, embedder configuration
- Document store setup
- Search configuration
- Tool integration

**Configuration:**
```yaml
llms:
  openai:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"

databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6333

embedders:
  ollama:
    type: "ollama"
    model: "nomic-embed-text"

agents:
  MinimalAgent:
    name: "MinimalAgent"
    llm: "openai"
    database: "qdrant"
    embedder: "ollama"
    document_stores: ["codebase"]
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 5

document_stores:
  codebase:
    source: "directory"
    path: "./"
    include_patterns: ["*.go", "*.yaml", "*.md"]
```

**Usage:**
```bash
export OPENAI_API_KEY="your-key"
./hector --config examples/minimal.yaml
```

**Requirements:**
- OpenAI API key
- Qdrant running (docker run -p 6333:6333 qdrant/qdrant)
- Ollama running for embeddings

---

## Minimal Config

**File:** `minimal-config.yaml`

**Purpose:** Learning example showing configuration structure.

**Features:**
- Clear structure demonstration
- Comments explaining each section
- Good starting template

**Configuration:**
```yaml
version: "1.0"
name: "minimal-agent"
description: "A minimal agent configuration for learning"

llms:
  main:
    type: "ollama"
    model: "llama3.2"
    host: "http://localhost:11434"
    temperature: 0.7

agents:
  simple-agent:
    name: "Simple Agent"
    description: "A minimal agent for demonstration"
    llm: "main"
    
    prompt:
      system_prompt: "You are a helpful AI assistant."
      include_history: true
```

**Usage:**
```bash
./hector --config examples/minimal-config.yaml
```

**Requirements:**
- Ollama running

---

## Basic

**File:** `basic.yaml`

**Purpose:** Production-ready single agent with tools and monitoring.

**Features:**
- Multiple LLM providers (Ollama + OpenAI)
- Full tool configuration with sandboxing
- Document stores and search
- Logging and performance settings
- Chain-of-thought reasoning

**Configuration Highlights:**
```yaml
global:
  logging:
    level: "info"
    format: "json"
  performance:
    max_concurrency: 4
    timeout: "15m"

llms:
  ollama-main: { ... }
  openai-gpt4: { ... }

agents:
  main-agent:
    llm: "ollama-main"
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 5

tools:
  repositories:
    - name: "local"
      tools:
        - name: "command_executor"
          config:
            allowed_commands: ["ls", "cat", "grep", "git"]
            enable_sandboxing: true
```

**Usage:**
```bash
./hector --config examples/basic.yaml
```

**Best For:**
- Production single-agent deployments
- General-purpose AI assistant
- File operations and code analysis

---

## Advanced

**File:** `advanced.yaml`

**Purpose:** Enterprise multi-agent system with autonomous workflows.

**Features:**
- 4 specialized agents (coordinator, researcher, analyst, creative)
- Multiple LLM providers with different temperatures
- 2 workflow modes (DAG + autonomous)
- Advanced tool configuration
- Multiple document stores
- Comprehensive error handling and retry policies

**Agents:**
1. **Coordinator** - Task delegation and orchestration (OpenAI GPT-4)
2. **Researcher** - Information gathering (Ollama)
3. **Analyst** - Data analysis (Ollama, temperature 0.3)
4. **Creative** - Innovation and problem-solving (Ollama, temperature 0.9)

**Workflows:**

**1. Autonomous Research:**
```yaml
mode: "autonomous"
goal: "Conduct comprehensive research"
strategy: "dynamic"
max_iterations: 10
termination_conditions:
  quality_threshold: 0.9
```

**2. Structured Analysis (DAG):**
```yaml
mode: "dag"
steps:
  - research → analysis → creative → integration
```

**Usage:**
```bash
# Autonomous workflow
echo "Research AI trends" | ./hector --workflow examples/advanced.yaml

# Specific workflow
./hector --workflow examples/advanced.yaml --workflow-name structured-analysis
```

**Requirements:**
- OpenAI API key (for coordinator)
- Ollama running (for specialized agents)
- Qdrant running
- Multiple document stores configured

**Best For:**
- Enterprise deployments
- Complex research tasks
- Multi-step analysis workflows
- AI-driven task planning

---

## Chain of Thought

**File:** `chain-of-thought.yaml`

**Purpose:** Fast reasoning engine optimized for simple queries.

**Features:**
- Chain-of-thought reasoning engine
- Behavioral stopping (no tools = stop)
- Lower token usage
- Custom system prompt
- Streaming enabled

**Configuration:**
```yaml
agents:
  ChainOfThoughtAgent:
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 10
      enable_streaming: true
      show_debug_info: false

    prompt:
      system_prompt: |
        You are an AI agent with sophisticated reasoning capabilities.
        Use tools when you need information.
        Think step-by-step and be explicit about your reasoning.
```

**Usage:**
```bash
./hector --config examples/chain-of-thought.yaml
```

**Example Query:**
```bash
echo "What's the weather in Berlin?" | ./hector --config examples/chain-of-thought.yaml
```

**Best For:**
- Fast responses needed
- Simple factual queries
- Tool-based lookups
- Cost-sensitive applications

**Performance:**
- ~2-3 LLM calls per simple query
- Stops when no tools are called
- Lower token usage

---

## Reasoning Comparison

**File:** `reasoning-comparison.yaml`

**Purpose:** Compare chain-of-thought vs structured-reasoning engines side-by-side.

**Features:**
- Two agents with different reasoning engines
- Same query, different approaches
- Performance comparison
- Quality comparison

**Agents:**

**1. FastAgent (chain-of-thought):**
```yaml
reasoning:
  engine: "chain-of-thought"
  max_iterations: 5
  show_debug_info: false
```

**2. ThoroughAgent (structured-reasoning):**
```yaml
reasoning:
  engine: "structured-reasoning"
  max_iterations: 10
  show_thinking: true      # See internal reasoning!
  show_debug_info: false
```

**Usage:**
```bash
# Try with fast agent
./hector --config examples/reasoning-comparison.yaml --agent FastAgent

# Try with thorough agent
./hector --config examples/reasoning-comparison.yaml --agent ThoroughAgent
```

**Example Comparison:**

| Aspect | chain-of-thought | structured-reasoning |
|--------|------------------|---------------------|
| Speed | ⚡ Fast | 🐢 Thorough |
| Token Usage | 💰 Lower | 💰💰 Higher |
| Goal Tracking | ❌ No | ✅ Explicit |
| Confidence | ❌ No | ✅ 0-100% |
| Thinking Visible | ❌ No | ✅ Grayed out blocks |

**Best For:**
- Learning the differences
- Choosing the right engine
- Performance testing

---

## Workflow

**File:** `workflow.yaml`

**Purpose:** DAG-based multi-agent workflow with structured execution.

**Features:**
- 3 specialized agents (research, analysis, synthesis)
- DAG execution mode
- Step dependencies
- Shared memory and caching
- Retry policies

**Workflow Steps:**
```yaml
1. research_phase:
   agent: research-agent
   input: user_input
   output: research_results

2. analysis_phase:
   agent: analysis-agent
   input: research_results
   output: analysis_insights
   depends_on: [research_phase]

3. synthesis_phase:
   agent: synthesis-agent
   input: analysis_insights
   output: final_report
   depends_on: [analysis_phase]
```

**Configuration:**
```yaml
workflows:
  research-analysis-workflow:
    mode: "dag"
    agents: ["research-agent", "analysis-agent", "synthesis-agent"]
    
    execution:
      dag:
        steps: [ ... ]
    
    settings:
      max_concurrency: 2
      timeout: "20m"
      retry_policy:
        max_retries: 3
        backoff: "5s"
```

**Usage:**
```bash
echo "Research renewable energy benefits" | ./hector --workflow examples/workflow.yaml
```

**Best For:**
- Structured multi-step tasks
- Research and analysis workflows
- Document generation pipelines
- Predictable execution order

**Flow:**
```
User Query
    ↓
Research Agent (gather info)
    ↓
Analysis Agent (analyze findings)
    ↓
Synthesis Agent (create report)
    ↓
Final Report
```

---

## Progressive Config

**File:** `progressive-config.yaml`

**Purpose:** Learning example showing how to progressively add complexity.

**Features:**
- Starts simple, adds features step by step
- Comments explaining each addition
- Good for tutorials

**Progression:**
1. Basic agent + LLM
2. Add prompt configuration
3. Add reasoning engine
4. Add tools
5. Add search and documents
6. Add workflows

**Usage:**
```bash
./hector --config examples/progressive-config.yaml
```

**Best For:**
- Learning configuration structure
- Understanding feature dependencies
- Building your own config step-by-step

---

## Running Examples

### Prerequisites

**For all examples:**
```bash
# Install Hector
go build ./cmd/hector
```

**For Ollama-based examples:**
```bash
# Start Ollama
ollama serve

# Pull models
ollama pull llama3.2
ollama pull nomic-embed-text
```

**For OpenAI-based examples:**
```bash
export OPENAI_API_KEY="your-openai-api-key"
```

**For examples with search:**
```bash
# Start Qdrant
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### Interactive Mode

```bash
./hector --config examples/basic.yaml
```

### Single Query Mode

```bash
echo "Your question here" | ./hector --config examples/basic.yaml
```

### Debug Mode

```bash
./hector --config examples/basic.yaml --debug
```

### Workflow Mode

```bash
echo "Research topic" | ./hector --workflow examples/workflow.yaml
```

---

## Choosing the Right Example

### I want to...

**Get started quickly:**
→ `zero-config.yaml` or `minimal-config.yaml`

**Build a production agent:**
→ `basic.yaml`

**Use fast reasoning:**
→ `chain-of-thought.yaml`

**Use thorough reasoning with thinking blocks:**
→ `reasoning-comparison.yaml` (ThoroughAgent)

**Build a multi-agent system:**
→ `workflow.yaml` (structured) or `advanced.yaml` (autonomous)

**Learn configuration structure:**
→ `progressive-config.yaml`

**Compare reasoning engines:**
→ `reasoning-comparison.yaml`

**Enterprise deployment:**
→ `advanced.yaml`

---

## Configuration Tips

### Start Simple
Begin with `minimal-config.yaml`, test it, then add features.

### Use Environment Variables
```bash
export OPENAI_API_KEY="sk-..."
export QDRANT_HOST="localhost"
```

### Enable Debug Mode
```bash
./hector --config your-config.yaml --debug
```

### Test Before Production
Test with `gpt-4o-mini` or Ollama before using expensive models.

### Monitor Token Usage
Enable logging to track token consumption:
```yaml
global:
  logging:
    level: "info"
    format: "json"
```

---

## Common Patterns

### Pattern 1: Fast Single Agent
```yaml
# Zero-config + fast reasoning
agents:
  agent: { llm: "openai" }
llms:
  openai: { type: "openai", model: "gpt-4o-mini" }
```

### Pattern 2: Thorough Analysis
```yaml
# Structured reasoning + high-quality model
agents:
  agent:
    llm: "openai"
    reasoning:
      engine: "structured-reasoning"
      show_thinking: true
llms:
  openai: { type: "openai", model: "gpt-4o" }
```

### Pattern 3: Local + Private
```yaml
# Ollama only, no external APIs
agents:
  agent: { llm: "ollama" }
llms:
  ollama: { type: "ollama", model: "llama3.2" }
```

### Pattern 4: Multi-Agent Pipeline
```yaml
# DAG workflow with specialized agents
workflows:
  pipeline:
    mode: "dag"
    agents: ["researcher", "analyst", "writer"]
```

---

## Troubleshooting Examples

### Example Won't Start

**Check:**
1. Required services running (Ollama, Qdrant)
2. Environment variables set
3. Paths in config exist
4. API keys valid

### Tools Not Working

**Check:**
1. Tool repository configured
2. Commands in allowlist
3. Sandboxing settings
4. Working directory exists

### Search Not Working

**Check:**
1. Database running
2. Embedder configured
3. Document store paths exist
4. Collection names match

---

## See Also

- [CONFIGURATION.md](../CONFIGURATION.md) - Complete configuration reference
- [README.md](../README.md) - Main documentation
- [local-configs/](../local-configs/) - Production configurations

---

**Need Help?**

- 📖 Start with `minimal-config.yaml`
- 🔧 Copy and modify examples
- 💬 Check inline comments in configs
- 📚 Read [CONFIGURATION.md](../CONFIGURATION.md) for details
