# Hector

```
 ██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
 ██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
 ███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
 ██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
 ██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
 ╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```

**Declarative AI Agent Framework**

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Beta-orange.svg)](https://github.com/kadirpekel/hector)

> **Beta Release**: Hector is a mature, production-ready framework for building both single agents and multi-agent systems. Core features are stable with comprehensive tooling, reasoning engines, and workflow orchestration.

## What is Hector?

Hector is a **declarative AI agent framework** that enables you to build both single agents and multi-agent systems through simple YAML configuration. It features:

### **🤖 Single Agent Capabilities:**
- 🧠 **Dynamic Reasoning Engine** - Multi-step thinking with self-reflection and meta-reasoning
- 🛠️ **Comprehensive Tool System** - Command execution, web search, and extensible tool repositories
- 📚 **Advanced Context Management** - Vector search with Qdrant, conversation history, and document stores
- ⚡ **Real-time Streaming** - Live reasoning process with conversational output
- 🔒 **Secure Execution** - Sandboxed command execution with configurable permissions
- 🔌 **Provider Architecture** - Pluggable LLM, database, and embedder providers

### **👥 Multi-Agent System Capabilities:**
- 🔄 **DAG Workflows** - Deterministic execution with dependencies and parallel processing
- 🤖 **Autonomous Coordination** - AI-driven task planning and agent collaboration
- 🏛️ **Team Management** - Coordinator-led task delegation and resource management
- ⚡ **Parallel Execution** - Multiple agents working simultaneously on different tasks
- 🎯 **Specialized Agents** - Role-based agents with distinct capabilities and configurations
- 📊 **Shared State Management** - Context sharing and memory persistence across agents

### **🎯 Universal Features:**
- 📋 **YAML Configuration** - Define everything through declarative configuration files
- 🔧 **Extensible Architecture** - Plug-and-play components, providers, and reasoning engines
- 📊 **Rich Monitoring** - Comprehensive logging, metrics, and debugging support
- 🚀 **Production Ready** - Robust error handling, retry policies, and health monitoring

## Quick Start

### Installation

```bash
git clone https://github.com/kadirpekel/hector.git
cd hector
go build ./cmd/hector
```

### Prerequisites

Hector requires the following services to be running:

```bash
# Start Ollama (for local LLM and embeddings)
ollama serve

# Start Qdrant (for vector database)
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### Basic Usage

Create a simple configuration (`my-agent.yaml`):

```yaml
version: "1.0"
name: "my-agent"
description: "A helpful AI agent"

# Provider configurations
providers:
  llms:
    openai-main:
      type: "openai"
      model: "gpt-4o-mini"
      api_key: "${OPENAI_API_KEY}"
      host: "https://api.openai.com/v1"

  databases:
    qdrant-main:
      type: "qdrant"
      host: "localhost"
      port: 6333

  embedders:
    ollama-embed:
      type: "ollama"
      model: "nomic-embed-text"
      host: "http://localhost:11434"

# Agent configuration
agents:
  main-agent:
    name: "My Assistant"
    description: "A helpful AI agent"
    llm: "openai-main"
    database: "qdrant-main"
    embedder: "ollama-embed"
    
    prompt:
      system_prompt: "You are a helpful AI assistant."
      include_history: true
```

Run your agent:

```bash
# Interactive mode
./hector --config my-agent.yaml

# Single query
echo "What is 2+2?" | ./hector --config my-agent.yaml --no-stream

# Debug mode
./hector --config my-agent.yaml --debug
```

### Demo with Tools

Try the included examples:

```bash
# Minimal configuration
./hector --config examples/minimal.yaml

# Basic configuration with tools
./hector --config examples/basic.yaml

# Ask questions that use tools
> What files are in the current directory?
> Show me the README file
> Search for information about AI
```

### Multi-Agent Workflow Demo

Try a multi-agent workflow:

```bash
# Run a DAG workflow
echo "Research renewable energy benefits" | ./hector --workflow examples/workflow.yaml --debug

# Run an advanced autonomous workflow
echo "Analyze market trends" | ./hector --workflow examples/advanced.yaml --debug
```

## Features

### Dynamic Reasoning Engine

Hector features an advanced reasoning system with multiple engines:

- **Multi-Step Reasoning**: Breaks complex problems into manageable steps
- **Self-Reflection**: Agent evaluates its own performance and adjusts approach
- **Meta-Reasoning**: AI reasons about its reasoning process for continuous improvement
- **Adaptive Complexity**: Automatically adjusts approach based on query difficulty
- **Streaming Support**: Real-time reasoning process with live output

```yaml
reasoning:
  engine: "dynamic"
  max_iterations: 5
  enable_self_reflection: true
  enable_meta_reasoning: true
  quality_threshold: 0.8
  show_debug_info: false
  enable_streaming: true
```

### Comprehensive Tool System

#### Command Execution Tools
Execute shell commands securely with sandboxing:

```yaml
tools:
  repositories:
    - name: "local"
      type: "local"
      tools:
        - name: "command_executor"
          type: "command"
          config:
            command_config:
              allowed_commands: ["ls", "cat", "head", "tail", "pwd", "git", "curl"]
              working_directory: "./"
              max_execution_time: "30s"
              enable_sandboxing: true
```

#### Web Search Tools
Semantic search across document stores:

```yaml
tools:
  repositories:
    - name: "local"
      tools:
        - name: "web_search"
          type: "search"
          config:
            search_config:
              document_stores: ["web-documents"]
              default_limit: 10
              max_limit: 50
              enabled_search_types: ["content", "file", "function"]
```

### Advanced Context Management

#### Vector Search with Qdrant
Semantic document search with multiple collections:

```yaml
providers:
  databases:
    qdrant-main:
      type: "qdrant"
      host: "localhost"
      port: 6333
      timeout: 30

  embedders:
    ollama-embed:
      type: "ollama"
      model: "nomic-embed-text"
      host: "http://localhost:11434"
      dimension: 768

search:
  models:
    - name: "documents"
      collection: "docs"
      default_top_k: 10
      max_top_k: 50
    - name: "code"
      collection: "code_docs"
      default_top_k: 10
      max_top_k: 50
  top_k: 10
  threshold: 0.7
  max_context_length: 4000
```

#### Document Stores
Automatic document ingestion and indexing:

```yaml
document_stores:
  web-documents:
    name: "Web Documents"
    source: "directory"
    path: "./docs"
    include_patterns: ["*.md", "*.txt", "*.html", "*.pdf"]
    watch_changes: true
```

#### Conversation History
Automatic conversation tracking and context management:

```yaml
prompt:
  include_history: true
  include_context: true
  include_tools: true
  max_context_length: 4000
```

## Configuration Reference

### Complete Configuration Structure

```yaml
version: "1.0"
name: "hector-config"
description: "Complete Hector configuration example"

# Global settings
global:
  logging:
    level: "info"                 # "debug", "info", "warn", "error"
    format: "json"                # "json", "text"
    output: "stdout"              # "stdout", "stderr", "file"
  performance:
    max_concurrency: 4            # Maximum concurrent operations
    timeout: "15m"                # Global timeout

# Provider configurations
providers:
  llms:
    openai-main:
      type: "openai"              # Provider type
      model: "gpt-4o-mini"        # Model name
      api_key: "${OPENAI_API_KEY}" # API key (use env vars)
      host: "https://api.openai.com/v1" # API endpoint
      temperature: 0.7            # Creativity (0.0-2.0)
      max_tokens: 2000            # Maximum response length
      timeout: 120                # Request timeout in seconds
    
    ollama-main:
      type: "ollama"              # Local LLM provider
      model: "llama3.2"           # Ollama model name
      host: "http://localhost:11434" # Ollama endpoint
      temperature: 0.7
      max_tokens: 2000
      timeout: 60

  databases:
    qdrant-main:
      type: "qdrant"              # Vector database provider
      host: "localhost"           # Database host
      port: 6333                  # Database port
      api_key: "${QDRANT_API_KEY:-}" # Optional API key
      timeout: 30                 # Connection timeout

  embedders:
    ollama-embed:
      type: "ollama"              # Embedding provider
      model: "nomic-embed-text"   # Embedding model
      host: "http://localhost:11434" # Ollama endpoint
      dimension: 768              # Embedding dimension
      timeout: 60                 # Request timeout

# Agent configurations
agents:
  main-agent:
    name: "Main Agent"            # Agent display name
    description: "Primary AI agent" # Agent description
    llm: "openai-main"            # LLM provider reference
    database: "qdrant-main"       # Database provider reference
    embedder: "ollama-embed"      # Embedder provider reference
    
    # Prompt configuration
    prompt:
      system_prompt: "You are an expert AI assistant."
      include_context: true       # Include search results
      include_history: true       # Include conversation history
      include_tools: true         # Include tool execution results
      max_context_length: 4000    # Maximum context size
    
    # Reasoning configuration
    reasoning:
      engine: "dynamic"           # Reasoning engine type
      max_iterations: 5           # Maximum reasoning steps
      enable_self_reflection: true # Enable self-evaluation
      enable_meta_reasoning: true  # Enable meta-reasoning
      quality_threshold: 0.8      # Quality threshold for completion
      show_debug_info: false      # Show technical details
      enable_streaming: true      # Enable real-time streaming
    
    # Search configuration
    search:
      models:
        - name: "documents"       # Search model identifier
          collection: "docs"      # Vector collection name
          default_top_k: 10       # Default search results
          max_top_k: 100          # Maximum search results
        - name: "code"
          collection: "code_docs"
          default_top_k: 10
          max_top_k: 50
      top_k: 10                   # Number of results to retrieve
      threshold: 0.7              # Minimum similarity score
      max_context_length: 4000    # Maximum context characters

# Tool system configuration
tools:
  default_repo: "local"           # Default tool repository
  repositories:
    - name: "local"               # Repository name
      type: "local"               # Repository type
      description: "Built-in tools" # Repository description
      tools:
        - name: "command_executor" # Tool name
          type: "command"         # Tool type
          enabled: true           # Tool enabled status
          config:
            command_config:
              allowed_commands:   # Whitelist of allowed commands
                - "ls"
                - "cat"
                - "head"
                - "tail"
                - "pwd"
                - "find"
                - "grep"
                - "git"
                - "curl"
              working_directory: "./" # Default execution directory
              max_execution_time: "30s" # Command timeout
              enable_sandboxing: true   # Enable security restrictions
        
        - name: "web_search"      # Search tool
          type: "search"
          enabled: true
          config:
            search_config:
              document_stores: ["web-documents"] # Available document stores
              default_limit: 10   # Default search limit
              max_limit: 50       # Maximum search limit
              enabled_search_types: ["content", "file", "function"] # Search types

# Document store configuration
document_stores:
  web-documents:
    name: "Web Documents"         # Store display name
    source: "directory"           # Source type
    path: "./docs"                # Source path
    include_patterns: ["*.md", "*.txt", "*.html", "*.pdf"] # File patterns
    watch_changes: true           # Watch for file changes

# Multi-agent workflow configuration
workflows:
  research-workflow:
    name: "Research Workflow"     # Workflow name
    description: "Multi-agent research process" # Workflow description
    mode: "dag"                   # Execution mode: "dag", "autonomous"
    
    agents:
      - "research-agent"          # Agent references
      - "analysis-agent"
      - "synthesis-agent"
    
    # Shared resources
    shared:
      memory:
        type: "in-memory"         # Memory type
      cache:
        type: "in-memory"         # Cache type
        ttl: "1h"                 # Cache TTL
    
    # DAG execution configuration
    execution:
      dag:
        steps:
          - name: "research_phase" # Step name
            agent: "research-agent" # Agent to execute
            input: "${user_input}" # Input template
            output: "research_results" # Output variable
          
          - name: "analysis_phase"
            agent: "analysis-agent"
            input: "${research_results}"
            output: "analysis_insights"
            depends_on: ["research_phase"] # Dependencies
    
    # Workflow settings
    settings:
      max_concurrency: 3          # Maximum concurrent agents
      timeout: "20m"              # Workflow timeout
      retry_policy:
        max_retries: 3            # Maximum retries
        backoff: "5s"             # Retry backoff
```

### Environment Variables

Use environment variables for sensitive data:

```yaml
providers:
  llms:
    openai-main:
      api_key: "${OPENAI_API_KEY}"
  
  databases:
    qdrant-main:
      api_key: "${QDRANT_API_KEY:-}"
```

## Examples

### Simple Chat Agent

```yaml
version: "1.0"
name: "chat-bot"
description: "Simple chat agent"

providers:
  llms:
    openai-main:
      type: "openai"
      model: "gpt-4o-mini"
      api_key: "${OPENAI_API_KEY}"
      host: "https://api.openai.com/v1"

agents:
  main-agent:
    name: "Chat Bot"
    llm: "openai-main"
    prompt:
      system_prompt: "You are a helpful assistant."
      include_history: true
```

### Document Q&A Agent

```yaml
version: "1.0"
name: "document-expert"
description: "Document Q&A agent"

providers:
  llms:
    openai-main:
      type: "openai"
      model: "gpt-4o-mini"
      api_key: "${OPENAI_API_KEY}"
      host: "https://api.openai.com/v1"
  
  databases:
    qdrant-main:
      type: "qdrant"
      host: "localhost"
      port: 6333
  
  embedders:
    ollama-embed:
      type: "ollama"
      model: "nomic-embed-text"
      host: "http://localhost:11434"

agents:
  main-agent:
    name: "Document Expert"
    llm: "openai-main"
    database: "qdrant-main"
    embedder: "ollama-embed"
    
    search:
      models:
        - name: "docs"
          collection: "documents"
          default_top_k: 5
      top_k: 5
    
    prompt:
      system_prompt: "Answer questions using the provided context."
      include_context: true
```

### Tool-Enabled Agent

```yaml
version: "1.0"
name: "tool-assistant"
description: "Agent with command tools"

providers:
  llms:
    openai-main:
      type: "openai"
      model: "gpt-4o-mini"
      api_key: "${OPENAI_API_KEY}"
      host: "https://api.openai.com/v1"

agents:
  main-agent:
    name: "Assistant with Tools"
    llm: "openai-main"
    
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

## Multi-Agent System Features

### 🔄 DAG Workflows

Execute pre-defined workflows with clear dependencies and parallel processing:

```yaml
workflows:
  research-pipeline:
    name: "Research & Analysis Pipeline"
    description: "Multi-phase research workflow"
    mode: "dag"
    
    agents:
      - "research-agent"
      - "analysis-agent"
      - "synthesis-agent"
    
    execution:
      dag:
        steps:
          - name: "primary_research"
            agent: "research-agent"
            input: "${user_input}"
            output: "research_results"
            parallel: true
            
          - name: "analysis"
            agent: "analysis-agent"
            input: "Analyze: ${research_results}"
            output: "analysis_insights"
            depends_on: ["primary_research"]
            
          - name: "final_report"
            agent: "synthesis-agent"
            input: "Write report: ${analysis_insights}"
            output: "final_report"
            depends_on: ["analysis"]
    
    settings:
      max_concurrency: 3
      timeout: "30m"
      retry_policy:
        max_retries: 3
        backoff: "30s"
```

### 🤖 Autonomous Coordination

AI-driven task planning and agent collaboration:

```yaml
workflows:
  autonomous-research:
    name: "Autonomous Research Team"
    description: "Self-organizing research team"
    mode: "autonomous"
    
    agents:
      - "research-agent"
      - "analysis-agent"
      - "synthesis-agent"
    
    execution:
      autonomous:
        goal: "Produce comprehensive research report with evidence"
        strategy: "adaptive"
        max_iterations: 10
        coordinator_llm: "openai-coordinator"
        termination_conditions:
          max_duration: "45m"
          quality_threshold: 0.9
          max_iterations: 10
    
    settings:
      max_concurrency: 4
      timeout: "45m"
      retry_policy:
        max_retries: 5
        backoff: "10s"
```

### ⚡ Advanced Multi-Agent Features

#### Parallel Execution
```yaml
settings:
  max_concurrency: 4  # Run up to 4 agents simultaneously
  timeout: "30m"      # Global workflow timeout
```

#### Error Recovery and Resilience
```yaml
settings:
  error_policy: "continue"  # Continue despite individual failures
  retry_policy:
    max_retries: 3
    backoff: "30s"
    exponential: true
```

#### Shared Context and Memory
```yaml
shared:
  memory:
    type: "in-memory"
  cache:
    type: "in-memory"
    ttl: "24h"
```

#### Specialized Agent Roles
```yaml
agents:
  research-agent:
    name: "Research Agent"
    description: "Specialized in information gathering"
    capabilities: ["research", "data-gathering", "analysis"]
    
  analysis-agent:
    name: "Analysis Agent"
    description: "Specialized in data analysis"
    capabilities: ["data-analysis", "synthesis", "critical-thinking"]
    
  synthesis-agent:
    name: "Synthesis Agent"
    description: "Specialized in report writing"
    capabilities: ["content-writing", "communication"]
```

## Architecture

### Single Agent Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Agent Core    │    │ Reasoning Engine│    │  Tool Manager   │
│                 │    │                 │    │                 │
│ • Configuration │◄──►│ • Multi-step    │◄──►│ • MCP Tools     │
│ • Lifecycle     │    │ • Self-reflect  │    │ • CLI Commands  │
│ • Context       │    │ • Meta-reason   │    │ • Security      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Context Manager │    │   LLM Provider  │    │ Vector Database │
│                 │    │                 │    │                 │
│ • Conversation  │    │ • OpenAI        │    │ • Qdrant        │
│ • Documents     │    │ • Ollama        │    │ • Embeddings    │
│ • Search        │    │ • Streaming     │    │ • Collections   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Multi-Agent System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    MULTI-AGENT SYSTEM                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐  │
│  │ Team Coordinator│    │ Agent Registry  │    │ Shared State│  │
│  │                 │    │                 │    │             │  │
│  │ • DAG Execution │    │ • Agent Pool    │    │ • Context   │  │
│  │ • Autonomous    │    │ • Capabilities  │    │ • Memory    │  │
│  │ • Load Balance  │    │ • Health Check  │    │ • Results   │  │
│  └─────────────────┘    └─────────────────┘    └─────────────┘  │
│           │                       │                     │       │
│           └───────────────────────┼─────────────────────┘       │
│                                   │                             │
├───────────────────────────────────┼─────────────────────────────┤
│              AGENT LAYER          │                             │
├───────────────────────────────────┼─────────────────────────────┤
│                                   ▼                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────┐ │
│  │ Researcher  │  │  Analyzer   │  │   Writer    │  │   ...   │ │
│  │ Agent       │  │   Agent     │  │   Agent     │  │         │ │
│  │             │  │             │  │             │  │         │ │
│  │ • Research  │  │ • Analysis  │  │ • Writing   │  │ • Tools │ │
│  │ • Tools     │  │ • Reasoning │  │ • Review    │  │ • Spec  │ │
│  │ • Context   │  │ • Memory    │  │ • Format    │  │ • Etc   │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Autonomous Coordination Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                 AUTONOMOUS COORDINATION                         │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐  │
│  │AutonomousCoord  │    │ ExecutionPlan   │    │ Consensus   │  │
│  │ • LLM Planning  │───►│ • Task Breakdown│───►│ • Voting    │  │
│  │ • Agent Matching│    │ • Capability Map│    │ • Synthesis │  │
│  │ • Goal Evolution│    │ • Dependencies  │    │ • Conflicts │  │
│  └─────────────────┘    └─────────────────┘    └─────────────┘  │
│           │                       │                     │       │
│           ▼                       ▼                     ▼       │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐  │
│  │ Task Assignment │    │ Parallel Exec   │    │ Result Sync │  │
│  │ • Capability    │    │ • Load Balance  │    │ • Context   │  │
│  │ • Workload      │    │ • Error Handle  │    │ • Memory    │  │
│  │ • Priority      │    │ • Health Check  │    │ • History   │  │
│  └─────────────────┘    └─────────────────┘    └─────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
┌─────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ User Query  │───►│ Context Manager │───►│ Prompt Builder  │
└─────────────┘    │                 │    │                 │
                   │ • History       │    │ • System Prompt │
                   │ • Documents     │    │ • Instructions  │
                   │ • Search        │    │ • Context       │
                   └─────────────────┘    └─────────────────┘
                            │                       │
                            ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Tool Execution  │◄───│ Reasoning Loop  │◄───│ LLM Generation  │
│                 │    │                 │    │                 │
│ • MCP Tools     │    │ • Multi-step    │    │ • OpenAI/Ollama │
│ • CLI Commands  │    │ • Self-reflect  │    │ • Streaming     │
│ • Results       │    │ • Meta-reason   │    │ • Tool Decision │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 ▼
                   ┌─────────────────┐
                   │ Final Response  │
                   │                 │
                   │ • Answer        │
                   │ • Citations     │
                   │ • Tool Results  │
                   └─────────────────┘
```

## Multi-Agent Workflow Examples

### Research & Analysis Pipeline (DAG)

```yaml
# workflows/research-pipeline.yaml
name: "Research & Analysis Pipeline"
description: "Multi-agent research workflow with parallel processing"

mode: "dag"

agents:
  - name: "researcher"
    config: "agents/researcher.yaml"
    instances: 2  # Parallel research
  - name: "analyzer"
    config: "agents/analyzer.yaml"
  - name: "writer"
    config: "agents/writer.yaml"

workflow:
  steps:
    - name: "primary_research"
      agent: "researcher"
      input: "${user_input}"
      output: "primary_results"
      parallel: true
      timeout: "5m"
      
    - name: "secondary_research"
      agent: "researcher"
      input: "Follow up on: ${primary_results}"
      output: "secondary_results"
      depends_on: ["primary_research"]
      
    - name: "comprehensive_analysis"
      agent: "analyzer"
      input: |
        Analyze the following research:
        Primary: ${primary_results}
        Secondary: ${secondary_results}
      output: "analysis"
      depends_on: ["primary_research", "secondary_research"]
      
    - name: "final_report"
      agent: "writer"
      input: "Create report from: ${analysis}"
      output: "final_report"
      depends_on: ["comprehensive_analysis"]

settings:
  max_concurrency: 3
  timeout: "20m"
  retry_policy:
    max_retries: 2
    backoff: "30s"
```

### Autonomous Research Team

```yaml
# workflows/autonomous-research.yaml
name: "Autonomous Research Team"
description: "Self-organizing research team with democratic coordination"

mode: "autonomous"

agents:
  - name: "researcher"
    config: "agents/researcher.yaml"
    instances: 2
    capabilities: ["research", "data-gathering", "analysis"]
  - name: "analyzer"
    config: "agents/analyzer.yaml"
    capabilities: ["data-analysis", "synthesis", "critical-thinking"]
  - name: "writer"
    config: "agents/writer.yaml"
    capabilities: ["content-writing", "communication"]

autonomous:
  coordinator_llm:
    name: "ollama"
    model: "llama3.2"
    temperature: 0.2
    
  goal: "Produce comprehensive research report with evidence"
  coordination_strategy: "democratic"
  max_iterations: 5
  
  termination_conditions:
    quality_threshold: 0.8
    consensus_reached: true
    max_time: "15m"
    
  coordination:
    communication_style: "broadcast"
    decision_making: "consensus"
    conflict_resolution: "vote"

settings:
  max_concurrency: 3
  error_policy: "continue"
```

### Code Review Workflow

```yaml
# workflows/code-review.yaml
name: "AI Code Review Team"
mode: "dag"

agents:
  - name: "security_reviewer"
    config: "agents/security-expert.yaml"
    capabilities: ["security-analysis", "vulnerability-detection"]
  - name: "quality_reviewer"
    config: "agents/quality-expert.yaml"
    capabilities: ["code-quality", "best-practices"]
  - name: "performance_reviewer"
    config: "agents/performance-expert.yaml"
    capabilities: ["performance-analysis", "optimization"]
  - name: "final_reviewer"
    config: "agents/senior-reviewer.yaml"
    capabilities: ["decision-making", "synthesis"]

workflow:
  steps:
    - name: "security_review"
      agent: "security_reviewer"
      input: "Review code for security issues: ${code_diff}"
      output: "security_report"
      parallel: true
      
    - name: "quality_review"
      agent: "quality_reviewer"
      input: "Review code quality: ${code_diff}"
      output: "quality_report"
      parallel: true
      
    - name: "performance_review"
      agent: "performance_reviewer"
      input: "Review performance: ${code_diff}"
      output: "performance_report"
      parallel: true
      
    - name: "final_decision"
      agent: "final_reviewer"
      input: |
        Make final review decision based on:
        Security: ${security_report}
        Quality: ${quality_report}
        Performance: ${performance_report}
      output: "final_decision"
      depends_on: ["security_review", "quality_review", "performance_review"]
```

## CLI Reference

### Single Agent Mode

```bash
./hector [options] [config-name]

Options:
  --config string     YAML configuration file path
  --debug            Show technical details and debug info  
  --no-stream        Disable streaming output (streaming is default)

Examples:
# Interactive mode with config file
./hector --config my-agent.yaml

# Interactive mode with named config (looks in configs/ directory)
./hector my-config

# Debug mode with technical details
./hector --config my-agent.yaml --debug

# Non-streaming mode for batch processing
./hector --config my-agent.yaml --no-stream

# Pipe input for single queries
echo "What is 2+2?" | ./hector --config my-agent.yaml

# Zero-config mode (uses defaults)
./hector
```

### Multi-Agent Workflow Mode

```bash
./hector --workflow <workflow-file> [options]

Options:
  --workflow string   Multi-agent workflow YAML file path
  --debug            Show technical details and debug info  
  --no-stream        Disable streaming output

Examples:
# Run DAG workflow with piped input
echo "Research renewable energy benefits" | ./hector --workflow examples/workflow.yaml --debug

# Run autonomous workflow
echo "Analyze market trends" | ./hector --workflow examples/advanced.yaml --debug

# Interactive workflow execution
./hector --workflow examples/workflow.yaml
```

### Interactive Commands

When running in interactive mode, you can use these commands:

```bash
/help         # Show available commands
/tools        # List available tools
/quit         # Exit the application
```

### Configuration Discovery

Hector automatically discovers configurations:

1. **Default config**: `hector.yaml` in current directory
2. **Named configs**: `configs/<name>.yaml`
3. **Zero-config**: Uses built-in defaults (requires Ollama and Qdrant)

## Supported Providers

### LLM Providers
- **OpenAI**: GPT-4, GPT-4o, GPT-3.5-turbo, GPT-4o-mini
- **Ollama**: All supported models (Llama, Mistral, CodeLlama, etc.)

### Database Providers
- **Qdrant**: Vector similarity search with collections and filtering

### Embedding Providers
- **Ollama**: nomic-embed-text, all-MiniLM-L6-v2, and other embedding models

### Tool Repositories
- **Local**: Built-in command execution and search tools
- **MCP**: Model Context Protocol integration (planned)

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
├── config/              # Configuration types and loading
├── context/             # Context management and search
├── databases/           # Database provider implementations
├── embedders/           # Embedding provider implementations
├── executors/           # Multi-agent workflow executors
├── interfaces/          # Core interface definitions
├── llms/                # LLM provider implementations
├── providers/           # Provider registry and management
├── reasonings/          # Reasoning engine implementations
├── tools/               # Tool system and repositories
├── examples/            # Configuration examples
└── local-configs/       # Local development configurations
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file.

---

**Hector** - Declarative AI Agent Framework

*Built with ❤️ in Go*