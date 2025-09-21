# Hector

```
 ██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
 ██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
 ███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
 ██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
 ██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
 ╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```

**Workflow-First AI Agent Framework**

Hector is a workflow-first AI agent framework that enables you to build sophisticated multi-agent systems through clean, intuitive YAML configuration. Define complete agent workflows declaratively—from basic single-step workflows to complex multi-agent systems with advanced AI-driven reasoning—without any code. Every configuration follows a natural workflow-first structure where each step is a complete agent with its own LLM, memory, reasoning, and tools.

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Alpha-orange.svg)](https://github.com/kadirpekel/hector)

> **Alpha Version**: Hector is currently in alpha. Core features are functional, but we're actively exploring and experimenting with the framework. Expect API changes as we refine the approach. Perfect for early adopters who want to experiment with declarative multi-step AI agents.

## Why Hector?

Hector distinguishes itself through:

- **Workflow-First Architecture**: Clean, intuitive configuration with no root-level confusion
- **Complete Agent Steps**: Each workflow step is a full agent with LLM, memory, reasoning, and tools
- **AI-Driven Reasoning**: Agents automatically choose optimal reasoning complexity
- **Zero Configuration Conflicts**: Only one place to configure each agent - no root vs step confusion
- **Natural Multi-Agent**: Every execution is multi-agent by default with seamless state flow
- **Advanced AI Reasoning**: Sophisticated reasoning with meta-reasoning, self-reflection, and goal evolution
- **Integrated Ecosystem**: Seamless integration of LLM, vector DB, tools, and embedders
- **Hot-Swappable Providers**: Switch between providers without code changes

### Workflow-First Architecture Benefits

- **Zero Confusion**: Only one place to configure each agent - no root vs step conflicts
- **Intuitive Structure**: Configuration matches how users think about agent workflows
- **Complete Agents**: Each step is a full agent with its own LLM, memory, reasoning, and tools
- **Natural Scaling**: Add more steps easily without architectural changes
- **Clean Inheritance**: Global configs (models, MCP servers) merge seamlessly with step configs
- **Production Ready**: Clean, lean codebase with comprehensive testing

### Core Components

- **Agent Core**: Orchestrates reasoning workflows and component coordination
- **Large Language Models**: OpenAI, Ollama, TGI integration with hot-swappable providers for response generation
- **Vector Databases**: Qdrant integration for document storage, retrieval, and semantic search
- **Embedding Providers**: Multiple embedding models for converting text to vector representations
- **MCP Tools**: Model Context Protocol integration for external tool access and services
- **Document Ingestion**: Automated document synchronization from multiple sources
- **Workflow Engine**: Unified execution engine with AI-driven complexity evaluation

## Architecture

Hector follows a modular architecture with clear separation of concerns:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   LLM Provider  │    │  Vector Database │    │ Embedder Provider│
│  (OpenAI/Ollama)│    │    (Qdrant)     │    │ (OpenAI/Ollama)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Workflow Engine  │
                    │ (Multi-Agent)   │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Agent Steps    │
                    │ (Full Agents)   │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Dynamic Reasoning│
                    │ (AI-Driven)     │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │  MCP Tools      │
                    │  (External)     │
                    └─────────────────┘
```

## Agent Workflows

Hector uses a unified multi-agent workflow approach where every execution is naturally multi-agent by default. The AI intelligently decides the complexity of reasoning needed for each agent step.

### Workflow Architecture

**How it works**: 
1. **User defines workflow steps** - Each step represents a specialized agent
2. **Each step = Full agent** - Complete LLM, memory, and reasoning configuration
3. **AI chooses reasoning complexity** - Direct responses for straightforward queries, advanced reasoning for complex problems
4. **Natural state flow** - Context flows seamlessly between agent steps

### Simple Workflow
**Purpose**: Single-step execution with intelligent reasoning

```yaml
workflow:
  max_steps: 1
  verbose: true
  # Single step creates one agent that reasons intelligently
```

### Multi-Agent Workflow
**Purpose**: Structured workflows with specialized agents per step

Each step creates a full agent with its own configuration. Perfect for:
- Structured business processes
- Sequential data processing pipelines
- Multi-step validation workflows
- Specialized agent collaboration

**Configuration**:
```yaml
workflow:
  max_steps: 4
  verbose: true
  steps:
    - name: "research_analyst"
      type: "analyze"
      agent_config:
        llm:
          model: "gpt-4o"
          temperature: 0.3
    - name: "synthesis_expert"
      type: "execute"
      agent_config:
        llm:
          model: "gpt-4o-mini"
          temperature: 0.7
```

### Advanced Reasoning (Internal)
**Purpose**: Sophisticated AI reasoning within each agent

Advanced reasoning represents how sophisticated AI systems naturally think. Each agent can use advanced reasoning internally, which:
- **Meta-reasons** about its own thinking process
- **Self-reflects** and adapts its approach
- **Evolves goals** as understanding deepens
- **Stops intelligently** when quality is optimal

**Example reasoning flow**: When an agent tackles a complex problem, it analyzes what type of challenge it is, creates reasoning steps on-the-fly (research → analyze → synthesize → validate), continuously self-reflects ("Am I on the right track?"), adapts its approach if needed ("I need a different angle"), and stops when it has reached optimal quality.

Ideal for:
- Creative problem solving and innovation
- Complex research requiring adaptive approaches
- Strategic planning with evolving requirements
- Philosophical reasoning and deep analysis
- Multi-domain knowledge synthesis
- Problems requiring flexible, emergent solutions

**Per-Agent Reasoning Configuration**:
```yaml
workflow:
  steps:
    - name: "complex_analyst"
      type: "analyze"
      agent_config:
        reasoning:                            # Per-agent reasoning configuration
          max_iterations: 15
          goal_threshold: 0.85
          adaptation_threshold: 0.3
          quality_threshold: 0.7
          enable_self_reflection: true
          enable_meta_reasoning: true
          enable_dynamic_tools: true
        workflow:
          max_steps: 3
            enable_goal_evolution: true
            streaming_mode: "all_steps"
```

### Intelligent Complexity Selection
**Purpose**: AI automatically chooses reasoning complexity

For each agent step, the AI automatically evaluates whether to use:
- **Direct reasoning**: Immediate LLM calls for straightforward queries
- **Advanced reasoning**: Sophisticated multi-iteration reasoning for complex problems

This happens transparently - users just define workflows, and agents reason intelligently.

## Installation

### Prerequisites

**Local Development Setup**:
```bash
# Start Ollama server
ollama serve

# Start Qdrant vector database
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

**Cloud Provider Setup**:
```bash
# Configure AWS credentials for S3 support (optional)
aws configure
# OR set environment variables
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="us-east-1"
```

### Installation Methods

**Method 1: CLI Binary (Recommended)**
```bash
# Install Hector CLI
go install github.com/kadirpekel/hector/cmd/hector@latest

# Verify installation
hector --version
```

**Method 2: Go Package Integration**
```bash
# Add to your Go project
go get github.com/kadirpekel/hector@latest

# Import in your code
import "github.com/kadirpekel/hector"
```

## Quick Start

### Basic Usage

**Start with Workflow-First Configuration**:
```bash
# Run with basic workflow-first setup (recommended for beginners)
hector --config examples/basic.yaml
```

**Available Example Configurations**:
```bash
# Basic workflow-first agent
hector --config examples/basic.yaml

# Advanced multi-agent workflow
hector --config examples/advanced.yaml

# Advanced reasoning workflow
hector --config examples/dynamic-mode.yaml

# Document ingestion workflow
hector --config examples/document-ingestion.yaml
```

### First Steps

1. **Choose a Configuration**: Start with `basic.yaml` for workflow-first functionality
2. **Configure Your Agent**: Set up LLM, memory, and reasoning in the workflow step
3. **Test Basic Queries**: Try simple questions to verify functionality
4. **Scale Your Workflow**: Add more steps to create multi-agent workflows
5. **Add Documents**: Configure document ingestion for knowledge base functionality

## Configuration Reference

### Workflow-First Configuration Structure

```yaml
# Agent Information (global)
agent:
  name: "Agent Name"
  description: "Agent description"

# Global Models (available to all agents)
models:
  - name: "documents"
    collection: "documents"
    default_top_k: 10
    max_top_k: 100

# Global MCP Servers (available to all agents)
mcp_servers:
  - name: "composio"
    url: "https://apollo.composio.dev/v3/mcp/"
    config:
      api_key: "your-api-key"

# Workflow Configuration (primary structure)
workflow:
  max_steps: 5                               # Maximum workflow steps
  verbose: true                               # Enable verbose output
  
  steps:
    - name: "main_agent"                     # Step name
      type: "execute"                        # Step type
      enabled: true                          # Enable this step
      agent_config:                          # Complete agent configuration
        agent:
          name: "Main Agent"
          description: "Primary agent"
        
        # LLM Provider Configuration (per agent)
        llm:
          name: "openai" | "ollama" | "tgi"
          # OpenAI fields:
          api_key: "your-api-key"            # Required for OpenAI
          model: "gpt-4o-mini"               # OpenAI model name
          temperature: 0.7                    # 0.0 to 2.0
          max_tokens: 2000                   # Maximum response length
          # Ollama fields:
          model: "llama3"                    # Ollama model name
          base_url: "http://localhost:11434" # Ollama server URL
          # TGI fields:
          api_key: "your-api-key"            # Required for TGI
          model: "microsoft/DialoGPT-medium" # TGI model name
          base_url: "https://api.example.com" # TGI server URL
        
        # Memory Provider Configuration (per agent)
        memory:
          name: "qdrant"
          host: "localhost"                  # Qdrant server host
          port: 6334                         # Qdrant server port
          api_key: "optional-api-key"       # For Qdrant Cloud
        
        # Embedder Provider Configuration (per agent)
        embedder:
          name: "ollama" | "openai" | "tgi"
          # OpenAI fields:
          api_key: "your-api-key"            # Required for OpenAI
          model: "text-embedding-3-small"   # OpenAI embedding model
          # Ollama fields:
          model: "nomic-embed-text"         # Ollama embedding model
          base_url: "http://localhost:11434" # Ollama server URL
          # TGI fields:
          api_key: "your-api-key"            # Required for TGI
          model: "sentence-transformers/all-MiniLM-L6-v2" # TGI embedding model
          base_url: "https://api.example.com" # TGI server URL
        
        # AI Reasoning Configuration (per agent)
        reasoning:
          max_iterations: 5                  # Maximum reasoning iterations
          enable_meta_reasoning: true        # Enable meta-reasoning
          enable_self_reflection: true       # Enable self-reflection
          enable_dynamic_tools: true         # Enable dynamic tool selection
          quality_threshold: 0.8             # Quality threshold
          streaming_mode: "all_steps"        # Streaming mode
  streaming_mode: "all_steps" | "final_only" | "none"
  
  # Workflow steps (each creates a full agent)
  steps:
    - name: "research_analyst"
      type: "analyze"
      enabled: true
      agent_config:                           # Full agent configuration per step
        llm:
          model: "gpt-4o"
          temperature: 0.3
        workflow:
          reasoning:                          # Per-agent reasoning configuration
            max_iterations: 10
            enable_meta_reasoning: true
            enable_self_reflection: true
            quality_threshold: 0.8
    - name: "synthesis_expert"
      type: "execute"
      enabled: true
      agent_config:
        llm:
          model: "gpt-4o-mini"
          temperature: 0.7

# MCP Tool Servers
mcp_servers:
  - name: "server-name"
    url: "https://api.example.com/mcp"
    description: "Tool description"
    config:
      api_key: "your-api-key"                # Provider-specific config

# Global Document Sources
sources:
  source_name:
    type: "local" | "s3" | "minio" | "gdrive"
    path: "/path/to/documents" | "bucket-name"
    region: "us-east-1"                     # For S3/MinIO
    access_key_id: "access-key"              # AWS/MinIO access key
    secret_access_key: "secret-key"          # AWS/MinIO secret key
    credentials:                             # For Google Drive
      client_id: "client-id"
      client_secret: "client-secret"
      refresh_token: "refresh-token"
    options:                                 # Additional provider options
      key: "value"

# Document Models
models:
  - name: "model-name"
    collection: "collection-name"
    default_top_k: 10                        # Default search results
    max_top_k: 100                           # Maximum search results
    ingestion:
      auto_sync: true                        # Enable automatic sync
      sync_interval: "10m"                   # Sync interval
      sources:
        - source: "source_name"              # Reference to global source
          pattern: "**/*.pdf"                # File pattern
          exclude_patterns: ["*.tmp", "drafts/*"]  # Exclusion patterns
        - inline_source:                     # Inline source definition
            type: "local"
            path: "/path/to/docs"
```

### Provider-Specific Configurations

#### OpenAI Configuration
```yaml
llm:
  name: "openai"
  api_key: "sk-..."
  model: "gpt-4o-mini"
  temperature: 0.7
  max_tokens: 2000

embedder:
  name: "openai"
  api_key: "sk-..."
  model: "text-embedding-3-small"
```

#### Ollama Configuration
```yaml
llm:
  name: "ollama"
  model: "llama3"
  temperature: 0.7
  max_tokens: 2000
  base_url: "http://localhost:11434"

embedder:
  name: "ollama"
  model: "nomic-embed-text"
  base_url: "http://localhost:11434"
```

#### TGI Configuration
```yaml
llm:
  name: "tgi"
  api_key: "your-api-key"
  model: "microsoft/DialoGPT-medium"
  temperature: 0.7
  max_tokens: 2000
  base_url: "https://api.example.com"

embedder:
  name: "tgi"
  api_key: "your-api-key"
  model: "sentence-transformers/all-MiniLM-L6-v2"
  base_url: "https://api.example.com"
```

#### Qdrant Configuration
```yaml
memory:
  name: "qdrant"
  host: "localhost"
  port: 6334
  api_key: "optional-api-key"  # For Qdrant Cloud
```

### CLI Commands Reference

```bash
# Basic usage with config file
hector --config examples/basic.yaml

# Using named config (if available in examples directory)
hector basic

# Run without config (uses defaults)
hector

# Document ingestion commands (within Hector session)
/list-models                    # List all configured models
/sync-model documents           # Sync specific model
/sync-all                       # Sync all models
/model-status documents         # Check model status
/search "query" documents       # Search with model selection
```

### Configuration Validation

Hector provides comprehensive configuration validation with detailed error messages for:

- **Missing Required Fields**: Identifies missing essential configuration parameters
- **Invalid Provider Names**: Validates provider names against available options
- **Incorrect URL Formats**: Ensures proper URL formatting for external services
- **Invalid Reasoning Configurations**: Validates reasoning configuration settings
- **Malformed YAML Syntax**: Provides clear YAML syntax error reporting
- **Provider-Specific Validation**: Validates provider-specific configuration requirements

### Best Practices

1. **Start Simple**: Begin with `basic.yaml` for basic functionality
2. **Use Workflow-First**: Leverage clean, intuitive configuration structure
3. **Configure Sources Globally**: Define document sources once, reference across multiple models
4. **Implement Pattern Matching**: Use specific patterns to avoid ingesting unwanted files
5. **Secure Credentials**: Store sensitive data like API keys securely in configuration files
6. **Descriptive Naming**: Use descriptive collection and model names for better organization
7. **Appropriate Sync Intervals**: Set sync intervals based on document update frequency
8. **Monitor Performance**: Use verbose logging to monitor agent performance and behavior

## Document Ingestion

Hector provides comprehensive automated document ingestion capabilities with support for multiple sources and flexible pattern matching. The system enables global source definitions that can be referenced across different models for maximum reusability and efficiency.

### Key Benefits

- **Automated Synchronization**: Maintain up-to-date knowledge bases automatically
- **Multiple Source Support**: Local directories, S3, MinIO, Google Drive with extensible architecture
- **Advanced Pattern Matching**: Flexible file filtering with comprehensive wildcard support
- **Model-Level Control**: Individual models manage their own ingestion strategies
- **Source Reusability**: Define sources globally, reference anywhere in the system
- **Exclusion Pattern Support**: Skip unwanted files with sophisticated exclusion patterns

### Supported Source Types

| Type | Description | Pattern Matching | Status |
|------|-------------|------------------|--------|
| `local` | Local filesystem directories | **Supported** | **Ready** |
| `s3` | AWS S3 buckets | **Not Supported** | **Limited** |
| `minio` | MinIO object storage (S3-compatible) | **Not Supported** | **Limited** |
| `gdrive` | Google Drive | **Not Supported** | **Limited** |

**Pattern Matching Limitation**: Pattern matching (wildcards like `*.txt`, `**/*.pdf`) is only supported for local filesystems. For S3, MinIO, and Google Drive, you'll need to use local filesystem or implement custom ingestion logic.

**Credential Management**: Credentials can be configured through the config file (`access_key_id`, `secret_access_key`) or via environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`) or AWS config files (`~/.aws/credentials`).

### Pattern Matching Examples

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
          pattern: "**/*.pdf"           # All PDF files recursively
          exclude_patterns: ["*.tmp"]   # Exclude temporary files
        - source: "docs"
          pattern: "reports/*.md"       # Markdown files in reports folder
        - source: "docs"
          pattern: "*.txt"              # Text files in root directory only
```

### Source Configuration Examples

#### Local Filesystem
```yaml
sources:
  local_docs:
    type: "local"
    path: "/Users/username/Documents"
```

#### AWS S3
```yaml
sources:
  s3_docs:
    type: "s3"
    path: "my-documents-bucket"
    region: "us-east-1"
    access_key_id: "AKIAIOSFODNN7EXAMPLE"
    secret_access_key: "wJalrXUtnFEMI..."
```

#### MinIO (S3-Compatible)
```yaml
sources:
  minio_docs:
    type: "minio"
    path: "minio://localhost:9000/documents"
    access_key_id: "minioadmin"
    secret_access_key: "minioadmin"
```

#### Google Drive
```yaml
sources:
  gdrive_docs:
    type: "gdrive"
    path: "/My Drive/Documents"
    credentials:
      client_id: "your-client-id"
      client_secret: "your-client-secret"
      refresh_token: "your-refresh-token"
```

### Metadata Extraction

Each ingested document automatically includes comprehensive metadata:

- **filename**: Original file name
- **source**: Source name reference
- **path**: Complete file path
- **size**: File size in bytes
- **modified**: Last modification timestamp
- **extension**: File extension
- **ingested_at**: Ingestion timestamp

## Quick Examples

Hector provides comprehensive example configurations in the `/examples/` directory. Each example demonstrates specific capabilities and use cases:

### Available Examples

- **`basic.yaml`** - Basic workflow-first setup (recommended for beginners)
- **`advanced.yaml`** - Multi-agent workflows with specialized agents per step
- **`dynamic-mode.yaml`** - Multi-agent with AI-driven advanced reasoning and meta-cognition
- **`document-ingestion.yaml`** - Automated document synchronization from multiple sources
- **`command-line-tools.yaml`** - Integration with command-line tools via MCP

### Getting Started

```bash
# Start with basic workflow (recommended)
hector --config examples/basic.yaml

# Try multi-agent workflow
hector --config examples/advanced.yaml

# Try advanced reasoning
hector --config examples/dynamic-mode.yaml

# Set up document ingestion
hector --config examples/document-ingestion.yaml
```

For detailed information about each example, see [`/examples/README.md`](examples/README.md).

## License

MIT License - see [LICENSE](LICENSE) file for details.