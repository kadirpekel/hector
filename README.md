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

Hector is a declarative AI agent framework built in Go that compiles to a single binary for easy deployment and setup. It serves as both a **no-code tool for professionals** and an **agent framework for developers**—define entire agent workflows through YAML configuration alone, from simple single-step agents to advanced AI-driven dynamic reasoning systems that adapt and evolve in real-time, without writing any code. Hector seamlessly integrates large language models, vector databases, MCP tools, **native command-line access**, and embedding providers into cohesive reasoning workflows.

## ⚡ Quick Demo

```bash
# Start Hector
./hector

# Ask a simple question:
> how many files are there in the current directory?
Using tool: execute_command
You have 42 files in the current directory.
```

**That's it!** Natural language + command execution in one step.

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Alpha-orange.svg)](https://github.com/kadirpekel/hector)

> **Alpha Version**: Hector is currently in alpha. Core features are functional, but we're actively exploring and experimenting with the framework. Expect API changes as we refine the approach. Perfect for early adopters who want to experiment with declarative multi-step AI agents.

## Why Hector?

- **Dual-Purpose Design**: No-code tool for professionals and agent framework for developers
- **Single Binary Deployment**: Built in Go and compiles to a single executable for easy distribution and setup
- **Declarative Configuration**: Define entire agent workflows in YAML without programming
- **Multi-Step Reasoning**: Sequential task decomposition with specialized sub-agents
- **Dynamic AI Reasoning**: Pure AI-driven reasoning that adapts and evolves autonomously
- **Integrated Ecosystem**: Seamless integration of LLM, vector DB, tools, and embedders
- **Native Command-Line Tools**: Direct filesystem and shell access for agent workflows
- **Nested Agent Hierarchies**: Sub-agents with independent reasoning processes
- **Context-Aware Memory**: Persistent conversation history and document retrieval
- **Hot-Swappable Providers**: Switch between providers without code changes

### Core Components

- **Agent Core**: Orchestrates reasoning workflows and component coordination
- **Large Language Models**: OpenAI, Ollama, TGI integration with hot-swappable providers for response generation
- **Vector Databases**: Qdrant integration for document storage, retrieval, and semantic search
- **Embedding Providers**: Multiple embedding models for converting text to vector representations
- **MCP Tools**: Model Context Protocol integration for external tool access and services
- **Native Command-Line Tools**: Direct filesystem and shell command execution for agent workflows
- **Document Ingestion**: Automated document synchronization from multiple sources


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
                    │   Agent Core     │
                    │  (Reasoning)    │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │  MCP Tools      │
                    │  (External)     │
                    └─────────────────┘
```

## Reasoning Modes

Hector supports four distinct reasoning modes, each optimized for different task complexities and requirements:

### Simple Mode
**Purpose**: Direct, single-step execution for straightforward queries

Simple Mode provides immediate response generation without multi-step reasoning. Ideal for:
- Direct question answering
- Simple calculations and computations
- Basic information retrieval
- Quick response generation

**Configuration**:
```yaml
reasoning:
  strategy: "simple"
  max_steps: 1
  verbose: true
```

### Multi Mode
**Purpose**: Structured, sequential workflows with predefined steps

Multi Mode executes predetermined step sequences with specialized agents for each phase. Ideal for:
- Structured business processes
- Sequential data processing pipelines
- Multi-step validation workflows
- Predictable task sequences

**Configuration**:
```yaml
reasoning:
  strategy: "multi"
  max_steps: 4
  steps:
    - name: "validate"
      type: "execute"
    - name: "process"
      type: "analyze"
    - name: "approve"
      type: "execute"
```

### Dynamic Mode
**Purpose**: AI reasoning that mirrors how advanced AI systems actually think

Dynamic Mode represents a breakthrough in AI reasoning—it works exactly like how sophisticated AI systems naturally reason. Instead of following predetermined steps, the AI creates its own reasoning process in real-time, adapting and evolving its approach as it gains insights. This mirrors the meta-cognitive processes that advanced AI systems use when tackling complex problems.

**How it works**: The AI meta-reasons about its own thinking, creates steps dynamically based on problem analysis, self-reflects and adapts its approach, evolves goals as understanding deepens, and stops when quality is optimal—just like how advanced AI systems naturally think.


Ideal for:
- Creative problem solving and innovation
- Complex research requiring adaptive approaches
- Strategic planning with evolving requirements
- Philosophical reasoning and deep analysis
- Multi-domain knowledge synthesis
- Problems that require flexible, emergent solutions

**Configuration**:
```yaml
reasoning:
  strategy: "dynamic"
  verbose: true
  dynamic:
    max_iterations: 15
    goal_threshold: 0.85
    adaptation_threshold: 0.3
    quality_threshold: 0.7
    enable_self_reflection: true
    enable_meta_reasoning: true
    enable_dynamic_tools: true
    enable_goal_evolution: true
    streaming_mode: "all_steps"
```

### Auto Mode (Default)
**Purpose**: Intelligent strategy selection based on query analysis

Auto Mode analyzes incoming queries and automatically selects the optimal reasoning strategy. Ideal for:
- Mixed complexity applications
- Unknown query types
- General-purpose agent deployments
- Adaptive workflow requirements

**Configuration**:
```yaml
reasoning:
  strategy: "auto"
  max_steps: 6
  verbose: true
```

**Strategy Selection Logic**:
- **Simple**: Basic queries requiring direct responses
- **Multi**: Workflow queries requiring structured processing
- **Dynamic**: Complex, creative, or exploratory queries requiring adaptive reasoning

## Getting Started

### Prerequisites

**For Local/Default Setup** (optional - only needed if using Ollama/Qdrant):
```bash
# Start Ollama server (for local LLM)
ollama serve

# Start Qdrant vector database (for local vector storage)
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

**Note**: These are only required if you're using the default local setup. You can also use cloud providers (OpenAI, Qdrant Cloud, etc.) without local dependencies.

### Installation

```bash
# Install Hector CLI (single binary)
go install github.com/kadirpekel/hector/cmd/hector@latest

# Verify installation
hector --version
```

### Quick Start

```bash
# Start with minimal configuration (recommended)
hector --config examples/auto-mode-minimal.yaml

# Try different modes
hector --config examples/basic.yaml           # Basic setup with tools
hector --config examples/advanced.yaml       # Multi-step workflows
hector --config examples/dynamic-mode.yaml    # AI-driven dynamic reasoning
hector --config examples/document-ingestion.yaml  # Document synchronization
```

### CLI Commands

```bash
# Basic usage
hector --config examples/basic.yaml
hector basic  # Using named config
hector       # Run with defaults

# Document ingestion commands (within Hector session)
/list-models                    # List all configured models
/sync-model documents           # Sync specific model
/sync-all                       # Sync all models
/model-status documents         # Check model status
/search "query" documents       # Search with model selection
```


## Configuration Reference

### Core Configuration Structure

```yaml
# Agent Information
agent:
  name: "Agent Name"
  description: "Agent description"

# LLM Provider Configuration (dynamic fields based on provider)
llm:
  name: "openai" | "ollama" | "tgi"
  # OpenAI fields:
  api_key: "your-api-key"                    # Required for OpenAI
  model: "gpt-4o-mini"                       # OpenAI model name
  temperature: 0.7                            # 0.0 to 2.0
  max_tokens: 2000                           # Maximum response length
  # Ollama fields:
  model: "llama3"                            # Ollama model name
  base_url: "http://localhost:11434"         # Ollama server URL
  # TGI fields:
  api_key: "your-api-key"                    # Required for TGI
  model: "microsoft/DialoGPT-medium"         # TGI model name
  base_url: "https://api.example.com"        # TGI server URL

# Memory Provider Configuration (Qdrant)
memory:
  name: "qdrant"
  host: "localhost"                          # Qdrant server host
  port: 6334                                 # Qdrant server port
  api_key: "optional-api-key"               # For Qdrant Cloud

# Embedder Provider Configuration (dynamic fields based on provider)
embedder:
  name: "ollama" | "openai" | "tgi"
  # OpenAI fields:
  api_key: "your-api-key"                    # Required for OpenAI
  model: "text-embedding-3-small"           # OpenAI embedding model
  # Ollama fields:
  model: "nomic-embed-text"                 # Ollama embedding model
  base_url: "http://localhost:11434"         # Ollama server URL
  # TGI fields:
  api_key: "your-api-key"                    # Required for TGI
  model: "sentence-transformers/all-MiniLM-L6-v2"  # TGI embedding model
  base_url: "https://api.example.com"        # TGI server URL

# Reasoning Strategy Configuration
reasoning:
  strategy: "simple" | "multi" | "auto" | "dynamic"
  max_steps: 6                               # Maximum reasoning steps
  max_retries: 2                             # Maximum retry attempts
  enable_retry: true                          # Enable retry on failure
  enable_feedback: false                      # Enable feedback collection
  streaming_mode: "all_steps" | "final_only"  # Streaming configuration
  verbose: true                               # Enable detailed logging
  verbose_template: "\033[90m{{.Message}}\033[0m"  # Verbose output template
  
  # Multi-step configuration
  steps:
    - name: "step-name"
      type: "execute" | "analyze" | "synthesize"
      enabled: true
  
  # Dynamic mode configuration
  dynamic:
    max_iterations: 15                        # Maximum reasoning iterations
    goal_threshold: 0.85                     # Goal achievement threshold
    adaptation_threshold: 0.3                # Adaptation trigger threshold
    quality_threshold: 0.7                   # Minimum quality threshold
    enable_self_reflection: true             # Enable self-reflection
    enable_meta_reasoning: true              # Enable meta-reasoning
    enable_dynamic_tools: true               # Enable dynamic tool selection
    enable_goal_evolution: true              # Enable goal evolution
    streaming_mode: "all_steps"             # Streaming configuration

# MCP Tool Servers
mcp_servers:
  - name: "server-name"
    url: "https://api.example.com/mcp"
    description: "Tool description"
    config:
      api_key: "your-api-key"                # Provider-specific config

# Command-Line Tools Configuration
command_tools:
  allowed_commands:                          # Whitelist of allowed commands
    - "ls"                                   # File listing
    - "cat"                                  # File reading
    - "grep"                                 # Text search
    - "find"                                 # File search
    - "git"                                  # Git operations
    - "npm"                                  # Node.js package manager
    - "go"                                   # Go compiler
    - "python"                               # Python interpreter
    # ... add more commands as needed
  working_directory: "./"                    # Default working directory
  max_execution_time_seconds: 30             # Command timeout
  enable_sandboxing: true                    # Enable security restrictions

# MCP Servers Configuration
mcp_servers:
  - name: "server-name"
    url: "http://localhost:8080"
    description: "Server description"

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

### Command-Line Tools Configuration

Hector includes native command-line tool integration that allows agents to execute shell commands directly for filesystem operations, development tasks, and system management.

#### Basic Configuration
```yaml
command_tools:
  allowed_commands:                          # Whitelist of allowed commands
    # File operations
    - "ls"                                   # List directory contents
    - "cat"                                  # Read file contents
    - "head"                                 # Show first lines
    - "tail"                                 # Show last lines
    - "find"                                 # Search for files
    - "grep"                                 # Search in files
    - "cp"                                   # Copy files
    - "mv"                                   # Move files
    - "rm"                                   # Remove files
    - "mkdir"                                # Create directories
    - "touch"                                # Create files
    - "chmod"                                # Change permissions
    - "stat"                                 # File information
    
    # Text processing
    - "awk"                                  # Text processing
    - "sed"                                  # Stream editor
    - "sort"                                 # Sort lines
    - "uniq"                                 # Remove duplicates
    - "wc"                                   # Word count
    
    # System information
    - "pwd"                                  # Current directory
    - "whoami"                               # Current user
    - "uname"                                # System information
    - "ps"                                   # Process list
    - "df"                                   # Disk usage
    - "free"                                 # Memory usage
    
    # Development tools
    - "git"                                  # Git operations
    - "npm"                                  # Node.js package manager
    - "node"                                 # Node.js runtime
    - "python"                               # Python interpreter
    - "go"                                   # Go compiler
    - "gcc"                                  # C compiler
    - "make"                                 # Build tool
    
    # Network tools
    - "curl"                                 # HTTP client
    - "wget"                                 # Download tool
    - "ssh"                                  # SSH client
    - "scp"                                  # Secure copy
  
  working_directory: "./"                    # Default working directory
  max_execution_time_seconds: 30             # Command timeout (0 = no limit)
  enable_sandboxing: true                    # Enable security restrictions
```

#### Security Features

- **Command Whitelist**: Only explicitly allowed commands can be executed
- **Working Directory**: Commands are restricted to specified directory
- **Execution Timeout**: Commands are automatically terminated after timeout
- **Sandboxing**: Additional security restrictions when enabled

#### Example Usage

Command-line tools enable agents to perform filesystem operations, development tasks, and system management as part of their reasoning workflows. The agent automatically selects and executes appropriate commands based on the user's request.

#### Default Configuration

If no `command_tools` section is specified, Hector uses a comprehensive default configuration with common Unix commands enabled and security restrictions active.

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

## License

MIT License - see [LICENSE](LICENSE) file for details.

---

## Contributing 🤝

We welcome contributions! Whether you're fixing bugs, adding features, or improving documentation, your help makes Hector better for everyone.

### Quick Start for Contributors
- 🐛 **Bug Reports**: Use GitHub Issues with detailed reproduction steps
- ✨ **Feature Requests**: Open an issue to discuss new ideas
- 🔧 **Code Contributions**: Fork, branch, and submit a pull request
- 📚 **Documentation**: Help improve examples and guides

### Development Setup
```bash
# Clone and build
git clone https://github.com/kadirpekel/hector.git
cd hector
go build -o hector cmd/hector/main.go

# Run tests
go test ./...
```

For more details, see our [Contributing Guidelines](CONTRIBUTING.md) (coming soon).

---

## Community & Support 💬

- 📖 **Documentation**: Check the `/examples/` directory for configuration examples
- 🐛 **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/kadirpekel/hector/issues)
- 💡 **Discussions**: Join the conversation in [GitHub Discussions](https://github.com/kadirpekel/hector/discussions)
- ⭐ **Star**: If you find Hector useful, give it a star!

---

*Built with ❤️ in Go. Making AI agents accessible to everyone.*