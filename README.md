# Hector

```
 ██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
 ██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
 ███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
 ██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
 ██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
 ╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```

**Declarative Multi-Agent AI Framework**

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Alpha-orange.svg)](https://github.com/kadirpekel/hector)

> **Alpha Release**: Hector is in active development. Core features are stable, but expect API evolution as we refine the framework. Perfect for early adopters exploring declarative AI agent systems.

## What is Hector?

Hector is a **declarative multi-agent AI framework** that lets you build sophisticated AI systems through simple YAML configuration. No code required—just describe what you want, and Hector handles the complexity.

**Key Philosophy**: Every workflow step is a complete AI agent with its own LLM, memory, reasoning capabilities, and tools. This creates naturally composable, powerful multi-agent systems.

### ✨ Core Features

- **🔧 Zero-Config Start**: Works out of the box with just an API key
- **🏗️ Declarative Architecture**: Define complex AI workflows in clean YAML
- **🤖 True Multi-Agent**: Each step is a full agent with independent capabilities
- **🧠 Advanced AI Reasoning**: Meta-reasoning, self-reflection, and goal evolution
- **🔄 Hot-Swappable Providers**: Switch between LLMs, databases, and embedders seamlessly
- **🛠️ MCP Tool Integration**: Native support for Model Context Protocol tools
- **📚 Document Intelligence**: Built-in PDF, Word, and text processing
- **💡 Smart Defaults**: Sensible configurations that just work

## Quick Start

### 1. Install Hector

```bash
# Install the CLI
go install github.com/kadirpekel/hector/cmd/hector@latest

# For local development (optional)
ollama serve
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### 2. Create Your First Agent

**Option A: Zero Configuration**
```bash
# Uses local Ollama + Qdrant with smart defaults
hector
```

**Option B: OpenAI Integration**
```yaml
# config.yaml
llm:
  name: "openai"
  api_key: "sk-your-key-here"
# Everything else uses intelligent defaults
```

```bash
hector --config config.yaml
```

### 3. Start Chatting

```
> What's the weather like in San Francisco?
Processing... 

I'll help you check the weather in San Francisco. Let me use the weather tool to get current conditions.

[Tool: weather_check]
Location: San Francisco, CA
Current: 68°F, Partly Cloudy
Forecast: High 72°F, Low 58°F

The weather in San Francisco is currently 68°F and partly cloudy...
```

## Try the Examples

```bash
# 1. Minimal setup (just API key)
hector --config examples/minimal.yaml

# 2. General purpose with tools  
hector --config examples/basic.yaml

# 3. Multi-agent workflows
hector --config examples/multi-agent.yaml

# 4. AI-driven reasoning
hector --config examples/dynamic-reasoning.yaml

# 5. Document processing
hector --config examples/document-processing.yaml
```

## Configuration Examples

### Minimal Setup
```yaml
llm:
  name: "openai"
  api_key: "sk-..."
# That's it! Everything else is handled automatically
```

### Multi-Agent Workflow
```yaml
workflow:
  max_steps: 3
  steps:
    - name: "researcher"
      type: "analyze"
      agent_config:
        llm:
          model: "gpt-4o"
          temperature: 0.3
        reasoning:
          enable_meta_reasoning: true
    
    - name: "synthesizer"
      type: "execute"
      agent_config:
        llm:
          model: "gpt-4o-mini"
          temperature: 0.7
```

### Advanced AI Reasoning
```yaml
workflow:
  steps:
    - name: "dynamic_agent"
      agent_config:
        reasoning:
          max_iterations: 15
          enable_self_reflection: true
          enable_meta_reasoning: true
          enable_goal_evolution: true
          streaming_mode: "all_steps"
```

### Document Processing
```yaml
models:
  - name: "documents"
    collection: "docs"
    ingestion:
      sources:
        - source: "local_docs"
          pattern: "**/*.pdf"
        - source: "local_docs"
          pattern: "**/*.docx"
```

## Configuration Reference

### Complete Configuration Schema

```yaml
# Agent Information
agent:
  name: "My Agent"                    # Agent display name
  description: "Agent description"    # Agent purpose description

# LLM Provider Configuration
llm:
  name: "openai"                      # Provider: "openai", "ollama", "tgi"
  api_key: "sk-..."                   # API key (OpenAI)
  model: "gpt-4o"                     # Model name
  host: "https://api.openai.com/v1"   # API endpoint
  temperature: 0.7                    # Creativity (0.0-1.0)
  max_tokens: 1000                    # Response length limit
  timeout: 60                         # Request timeout (seconds)

# Memory/Vector Database Configuration  
memory:
  name: "qdrant"                      # Provider: "qdrant"
  host: "localhost"                   # Database host
  port: 6334                          # Database port
  timeout: 30                         # Connection timeout
  use_tls: false                      # Enable TLS
  insecure: false                     # Allow insecure connections

# Embedder Configuration
embedder:
  name: "ollama"                      # Provider: "ollama", "tgi"
  model: "nomic-embed-text"           # Embedding model
  host: "http://localhost:11434"      # Embedder endpoint
  dimension: 768                      # Vector dimensions
  timeout: 30                         # Request timeout
  max_retries: 3                      # Retry attempts

# Search Configuration
search:
  max_context_length: 2000            # Max context chars
  context_strategy: "relevance"       # Strategy: "relevance", "recent"
  enable_reranking: false             # Enable result reranking

# AI Reasoning Configuration
reasoning:
  max_iterations: 15                  # Max reasoning loops
  goal_threshold: 0.85                # Goal achievement threshold (0.0-1.0)
  adaptation_threshold: 0.3           # When to adapt approach (0.0-1.0)
  quality_threshold: 0.7              # Minimum quality threshold (0.0-1.0)
  enable_self_reflection: true        # AI self-evaluation
  enable_meta_reasoning: true         # AI reasons about reasoning
  enable_dynamic_tools: true          # AI selects tools dynamically
  enable_goal_evolution: true         # Goals can evolve during execution
  streaming_mode: "all_steps"         # Streaming: "all_steps", "final_only", "none"

# Workflow Configuration
workflow:
  max_steps: 3                        # Maximum workflow steps
  verbose: true                       # Enable verbose output
  streaming_mode: "all_steps"         # Workflow streaming mode
  
  # Tool Execution Settings
  tool_execution:
    parallel_execution: false         # Execute tools in parallel
    timeout_seconds: 30               # Tool timeout
    retry_delay_ms: 1000             # Retry delay
    max_concurrent: 3                 # Max concurrent tools
  
  # Error Handling
  error_handling:
    strategy: "retry"                 # Strategy: "retry", "skip", "abort"
    max_error_analysis: 1             # Max error analysis attempts
    error_threshold: 0.5              # Error threshold (0.0-1.0)
  
  # Context Management
  context:
    preserve_history: true            # Preserve step history
    max_history_steps: 10             # Max history steps
    enable_context_share: true        # Share context between steps
    context_window: 3                 # Context window size
  
  # Workflow Steps
  steps:
    - name: "step1"                   # Step name
      type: "analyze"                 # Type: "analyze", "execute", "synthesize"
      enabled: true                   # Enable this step
      agent_config:                   # Step-specific agent config
        # Any agent configuration can be overridden per step
        llm:
          model: "gpt-4o"
          temperature: 0.3
        reasoning:
          max_iterations: 5

# Document Models
models:
  - name: "documents"                 # Model name
    collection: "docs"                # Vector collection name
    default_top_k: 10                 # Default search results
    max_top_k: 100                    # Maximum search results
    
    # Document Ingestion
    ingestion:
      auto_sync: false                # Auto-sync documents
      sources:
        - source: "local_docs"        # Source name (from sources)
          pattern: "**/*.pdf"         # File pattern
          exclude_patterns: ["*.tmp"] # Exclusion patterns

# Global Sources
sources:
  local_docs:
    type: "local"                     # Source type: "local", "s3"
    path: "/path/to/documents"        # Local path or S3 bucket

# MCP Tool Servers
mcp_servers:
  - name: "composio"                  # Server name
    url: "https://apollo.composio.dev/v3/mcp/" # MCP endpoint
    description: "Weather and web tools" # Server description
    config:                           # Server-specific config
      api_key: "your-api-key"
```

### Configuration Inheritance

Hector uses a **hierarchical configuration system** where root-level settings are inherited by workflow steps:

```yaml
# Root level - inherited by all steps
llm:
  name: "openai"
  api_key: "sk-..."
  temperature: 0.7

workflow:
  steps:
    - name: "step1"
      # Inherits root llm config
      
    - name: "step2"
      agent_config:
        llm:
          temperature: 0.2  # Override just temperature
          # Inherits name, api_key from root
```

### Provider-Specific Options

#### OpenAI
```yaml
llm:
  name: "openai"
  api_key: "sk-..."                   # Required
  model: "gpt-4o"                     # Default: "gpt-3.5-turbo"
  host: "https://api.openai.com/v1"   # Default endpoint
  temperature: 0.7                    # Default: 0.7
  max_tokens: 1000                    # Default: 1000
  timeout: 60                         # Default: 60
```

#### Ollama
```yaml
llm:
  name: "ollama"
  model: "llama3.2"                   # Default: "llama3.2"
  host: "http://localhost:11434"      # Default: localhost:11434
  temperature: 0.7                    # Default: 0.7
  max_tokens: 1000                    # Default: 1000
  timeout: 60                         # Default: 60

embedder:
  name: "ollama"
  model: "nomic-embed-text"           # Default: "nomic-embed-text"
  host: "http://localhost:11434"      # Default: localhost:11434
  dimension: 768                      # Default: 768
```

#### Qdrant
```yaml
memory:
  name: "qdrant"
  host: "localhost"                   # Default: "localhost"
  port: 6334                          # Default: 6334
  timeout: 30                         # Default: 30
  use_tls: false                      # Default: false
  insecure: false                     # Default: false
```

#### TGI (Text Generation Inference)
```yaml
llm:
  name: "tgi"
  model: "microsoft/DialoGPT-medium"  # Default model
  host: "http://localhost:8080"       # Default: localhost:8080
  temperature: 0.7                    # Default: 0.7
  max_tokens: 1000                    # Default: 1000
  timeout: 60                         # Default: 60

embedder:
  name: "tgi"
  model: "sentence-transformers/all-MiniLM-L6-v2"
  host: "http://localhost:8080"       # Default: localhost:8080
  dimension: 384                      # Default: 384
```

### Reasoning Modes

#### Simple Reasoning
```yaml
reasoning:
  max_iterations: 1                   # Single-pass reasoning
  enable_self_reflection: false
  enable_meta_reasoning: false
```

#### Advanced Reasoning
```yaml
reasoning:
  max_iterations: 10                  # Multi-iteration reasoning
  enable_self_reflection: true        # AI evaluates its work
  enable_meta_reasoning: true         # AI reasons about reasoning
  quality_threshold: 0.8              # High quality threshold
```

#### Dynamic Reasoning
```yaml
reasoning:
  max_iterations: 15                  # Allow complex reasoning
  enable_self_reflection: true        # Self-evaluation
  enable_meta_reasoning: true         # Meta-reasoning
  enable_dynamic_tools: true          # Dynamic tool selection
  enable_goal_evolution: true         # Evolving goals
  streaming_mode: "all_steps"         # Stream all reasoning
```

### Document Processing

#### Local Files
```yaml
sources:
  docs:
    type: "local"
    path: "/path/to/documents"

models:
  - name: "documents"
    collection: "docs"
    ingestion:
      sources:
        - source: "docs"
          pattern: "**/*.{pdf,docx,txt,md}"
          exclude_patterns: ["*.tmp", ".*"]
```

#### S3 Integration
```yaml
sources:
  s3_docs:
    type: "s3"
    bucket: "my-documents"
    region: "us-east-1"
    access_key: "ACCESS_KEY"
    secret_key: "SECRET_KEY"

models:
  - name: "s3_documents"
    collection: "s3_docs"
    ingestion:
      sources:
        - source: "s3_docs"
          pattern: "documents/**/*.pdf"
```

## Supported Providers

### LLM Providers
- **OpenAI**: GPT-3.5, GPT-4, GPT-4o series
- **Ollama**: Local models (Llama, Mistral, etc.)
- **TGI**: Text Generation Inference servers

### Vector Databases
- **Qdrant**: High-performance vector search
- More providers coming soon

### Embedders
- **Ollama**: Local embedding models
- **TGI**: Transformer-based embeddings
- **OpenAI**: text-embedding-ada-002 and newer

### Tools (MCP Protocol)
- **Composio**: 100+ integrated tools (weather, web search, etc.)
- **Custom MCP Servers**: Build your own tool integrations

## Architecture

Hector's **workflow-first architecture** makes every execution naturally multi-agent:

```
Query → Agent Step 1 → Agent Step 2 → Agent Step 3 → Response
         ↓              ↓              ↓
      [LLM+Memory]   [LLM+Memory]   [LLM+Memory]
      [Reasoning]    [Reasoning]    [Reasoning]
      [Tools]        [Tools]        [Tools]
```

Each step is a complete agent that can:
- Use different LLM models and settings
- Maintain independent memory and context
- Apply specialized reasoning strategies
- Access different tool sets
- Make autonomous decisions

## Use Cases

### 🔬 Research & Analysis
```yaml
# Multi-step research pipeline
workflow:
  steps:
    - name: "data_collector"
      # Specialized for gathering information
    - name: "analyzer"
      # Focused on pattern recognition
    - name: "synthesizer"
      # Optimized for creating insights
```

### 📊 Business Intelligence
```yaml
# Document analysis workflow
models:
  - name: "reports"
    ingestion:
      pattern: "**/*.pdf"
reasoning:
  enable_meta_reasoning: true
```

### 🛠️ Task Automation
```yaml
# Tool-heavy automation
mcp_servers:
  - name: "composio"
    url: "https://apollo.composio.dev/v3/mcp/"
reasoning:
  enable_dynamic_tools: true
```

### 🎯 Creative Problem Solving
```yaml
# Dynamic reasoning for open-ended tasks
reasoning:
  max_iterations: 15
  enable_goal_evolution: true
  enable_self_reflection: true
```

## Example Configurations

Explore the [`examples/`](examples/) directory with **5 essential configurations**:

| Example | Use Case | Key Features |
|---------|----------|--------------|
| **[`minimal.yaml`](examples/minimal.yaml)** | Getting started | Just an API key, smart defaults |
| **[`basic.yaml`](examples/basic.yaml)** | General purpose | Tools, intelligent reasoning |
| **[`multi-agent.yaml`](examples/multi-agent.yaml)** | Complex workflows | Specialized agents, inheritance |
| **[`dynamic-reasoning.yaml`](examples/dynamic-reasoning.yaml)** | Creative problems | AI-driven adaptation, goal evolution |
| **[`document-processing.yaml`](examples/document-processing.yaml)** | Knowledge management | PDF/Word/Text ingestion, search |

## CLI Commands

### Interactive Mode
```bash
hector                           # Zero-config mode
hector --config config.yaml     # Custom configuration
```

### Within Chat Session
```bash
/help                           # Show available commands
/tools                          # List available tools
/search "query" documents       # Search document collections
/sync-model documents           # Sync document model
/list-models                    # Show all models
```

## Go API

### Basic Usage
```go
package main

import (
    "github.com/kadirpekel/hector"
    "github.com/kadirpekel/hector/providers"
)

func main() {
    // Register providers
    providers.RegisterDefaultProviders()
    
    // Create agent
    agent, _ := hector.NewAgentWithDefaults()
    
    // Execute query
    response, _ := agent.ExecuteQueryWithReasoning("Hello!")
    fmt.Println(response.Answer)
}
```

### Configuration-Based
```go
// Load from YAML
agent, err := hector.NewAgentFromYAML("config.yaml")

// Streaming responses
stream, err := agent.ExecuteQueryWithReasoningStreaming("Complex query")
for chunk := range stream {
    fmt.Print(chunk)
}
```

## What Makes Hector Different?

### 🎯 **Workflow-First Design**
Unlike other frameworks that bolt multi-agent features onto single-agent architectures, Hector is built from the ground up for multi-agent workflows. Every execution is naturally multi-agent.

### 🧠 **AI-Driven Intelligence**
Agents don't just follow scripts—they reason about their tasks, reflect on their performance, and adapt their strategies in real-time.

### ⚡ **Zero-Config Philosophy**
Start with just an API key. Hector provides intelligent defaults for everything else, but gives you full control when you need it.

### 🔧 **True Composability**
Mix and match LLMs, databases, embedders, and tools. Each agent step can use completely different configurations.

### 📈 **Production Ready**
Built in Go for performance and reliability. Designed for both experimentation and production deployment.

## Contributing

Hector is in active development. We welcome:

- 🐛 Bug reports and feature requests
- 📝 Documentation improvements
- 🔧 Provider implementations
- 💡 Example configurations
- 🧪 Testing and feedback

See our [contribution guidelines](CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Ready to build intelligent AI systems?** Start with `go install github.com/kadirpekel/hector/cmd/hector@latest` and explore the examples!