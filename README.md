# Hector

```
 ██╗  ██╗███████╗ ██████╗████████╗ ██████╗ ██████╗ 
 ██║  ██║██╔════╝██╔════╝╚══██╔══╝██╔═══██╗██╔══██╗
 ███████║█████╗  ██║        ██║   ██║   ██║██████╔╝
 ██╔══██║██╔══╝  ██║        ██║   ██║   ██║██╔══██╗
 ██║  ██║███████╗╚██████╗   ██║   ╚██████╔╝██║  ██║
 ╚═╝  ╚═╝╚══════╝ ╚═════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
```

**Declarative AI Agent Framework - Multi-step reasoning with integrated components**

Hector is a declarative AI agent framework that combines LLM providers, vector databases, MCP tools, and embedders into multi-step reasoning workflows. Define agent hierarchies through YAML configuration - from simple single-step agents to nested systems where each reasoning step contains its own specialized sub-agent.

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Alpha-orange.svg)](https://github.com/kadirpekel/hector)

> **Alpha Version**: Hector is currently in alpha. Core features work, but we're actively exploring and experimenting with the framework. Expect API changes as we refine the approach. Perfect for early adopters who want to experiment with declarative multi-step AI agents.

## Why Hector?

Unlike LangChain (Python-focused) or CrewAI (limited configurability), Hector offers:

- **Pure Declarative Approach**: Define entire agent workflows in YAML - no Python/JavaScript required
- **True Multi-Step Reasoning**: Sequential task decomposition with specialized sub-agents, not just tool chaining
- **Integrated Components**: LLM + VectorDB + MCP + Embedders work together seamlessly
- **Nested Agent Hierarchies**: Sub-agents can contain their own reasoning processes (unique capability)
- **Hot-Swappable Providers**: Switch between Ollama, OpenAI, TGI without code changes

Perfect for teams who want sophisticated AI agents without complex programming.

## Key Features

- **Multi-Step Reasoning**: Sequential workflows with specialized agents for each step
- **Nested Agent Hierarchies**: Sub-agents with their own reasoning processes
- **Declarative Configuration**: Single YAML file defines entire agent workflows
- **Integrated Components**: LLM providers, vector databases, MCP tools, embedders
- **Hot-Swappable Providers**: Ollama, OpenAI, TGI, Qdrant with dynamic configuration
- **MCP Tool Integration**: Generic tool access through Model Context Protocol
- **Conversation Memory**: Persistent chat history and context awareness
- **Document Ingestion**: Automated document sync from multiple sources with pattern matching

## Use Cases

### Research & Analysis
- **Market Research Agent**: Multi-step analysis with data collection → analysis → report generation
- **Document Analysis**: Extract insights from large document collections with specialized sub-agents
- **Competitive Intelligence**: Automated research workflows with nested reasoning steps

### Customer Support
- **Tiered Support Agent**: Route → analyze → escalate with specialized agents for each step
- **Knowledge Base Assistant**: Search → synthesize → respond with context-aware memory
- **Technical Troubleshooting**: Diagnose → test → resolve with tool integration

### Content & Operations
- **Content Creation Pipeline**: Research → draft → review → publish workflows
- **Data Processing**: Extract → transform → validate → store with specialized agents
- **Quality Assurance**: Test → analyze → report with nested validation agents

### Business Automation
- **Lead Qualification**: Analyze → score → route with specialized business logic
- **Invoice Processing**: Extract → validate → approve with multi-step verification
- **Compliance Monitoring**: Monitor → analyze → alert with regulatory expertise

## Quick Start

### Prerequisites
```bash
# For local setup
ollama serve
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant

# For S3 support (optional)
# Configure AWS credentials via:
# - AWS CLI: aws configure
# - Environment variables: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
# - AWS credential files: ~/.aws/credentials
```

### Install & Run

**Option 1: CLI Binary (Recommended)**
```bash
# Install Hector CLI
go install github.com/kadirpekel/hector/cmd/hector@latest

# Run with basic configuration (includes tool support)
hector --config examples/basic.yaml

# Run with advanced features
hector --config examples/advanced.yaml

# Run with document ingestion
hector --config examples/document-ingestion.yaml
```

**Option 2: Go Package (For developers)**
```bash
# Add to your Go project
go get github.com/kadirpekel/hector@v1.0.2

# Use in your code
import "github.com/kadirpekel/hector"
```

## Configuration Examples

### Basic Setup with Tools
```yaml
# examples/basic.yaml
agent:
  name: "Hector Basic"
  description: "Basic AI agent with tool support"

llm:
  name: "openai"
  api_key: "YOUR_OPENAI_API_KEY_HERE"
  model: "gpt-4o-mini"
  temperature: 0.3
  max_tokens: 1000

# MCP Tool Integration
mcp_servers:
  - name: "composio"
    url: "https://apollo.composio.dev/v3/mcp/..."
    description: "Weather and web search tools"
    config:
      api_key: "YOUR_COMPOSIO_API_KEY_HERE"

# Basic document model
models:
  - name: "documents"
    collection: "documents"
    default_top_k: 10
    max_top_k: 100

# Enable tool support
reasoning:
  strategy: "single_shot"
  max_iterations: 1
  enable_tools: true
```

### Multi-Step Reasoning
```yaml
reasoning:
  strategy: "state_machine"
  steps:
    - name: "analyzer"
      type: "analyze"
      agent_config:
        llm:
          name: "openai"
          model: "gpt-4"
          temperature: 0.3
    
    - name: "executor"
      type: "execute"
      agent_config:
        llm:
          name: "openai"
          model: "gpt-4"
          temperature: 0.7
```

### Document Ingestion
```yaml
# Global sources (define once, use everywhere)
sources:
  local_docs:
    type: "local"
    path: "/Users/john/documents"
  s3_docs:
    type: "s3"
    path: "my-documents-bucket"  # S3 bucket name
  minio_docs:
    type: "minio"
    path: "minio://localhost:9000/documents"  # MinIO endpoint and bucket
    access_key_id: "minioadmin"
    secret_access_key: "minioadmin"
  gdrive_docs:
    type: "gdrive"
    path: "/My Drive/Documents"  # Google Drive path
    credentials:
      client_id: "your-client-id"
      client_secret: "your-client-secret"
      refresh_token: "your-refresh-token"

# Models with their own ingestion strategies
models:
  - name: "documents"
    collection: "documents"
    
    ingestion:
      auto_sync: true
      sync_interval: "10m"
      sources:
        - source: "local_docs"
          pattern: "**/*.pdf"
          exclude_patterns: ["*.tmp", "drafts/*"]
        - source: "s3_docs"
          pattern: "**/*.txt"
        - source: "minio_docs"
          pattern: "**/*.docx"
        - source: "gdrive_docs"
          pattern: "**/*.md"
```

### Available Configurations

**📁 Configuration Examples** (see `/examples/README.md` for details):

- **`examples/basic.yaml`** - Basic setup with tool support
- **`examples/advanced.yaml`** - Advanced reasoning with tools
- **`examples/document-ingestion.yaml`** - Document automation

## Document Ingestion

Hector supports **automated document ingestion** from multiple sources with flexible pattern matching. Define global sources once and reference them across different models for maximum reusability.

### Key Benefits

- **🔄 Automated Sync**: Keep your knowledge base up-to-date automatically
- **📁 Multiple Sources**: Local directories, S3, MinIO, Google Drive (extensible)
- **🎯 Pattern Matching**: Flexible file filtering (`**/*.pdf`, `**/*.txt`, etc.)
- **🏗️ Model-Level Control**: Each model manages its own ingestion strategy
- **♻️ Source Reuse**: Define sources globally, reference anywhere
- **🚫 Exclusion Patterns**: Skip unwanted files (`*.tmp`, `drafts/*`)

### CLI Commands

```bash
# List all models
/list-models

# Sync a specific model
/sync-model documents

# Sync all models
/sync-all

# Check model status
/model-status documents

# Search with model selection
/search "artificial intelligence" documents
```

### Supported Source Types

| Type | Description | Status |
|------|-------------|--------|
| `local` | Local filesystem directories | ✅ **Ready** |
| `s3` | AWS S3 buckets | ✅ **Ready** |
| `minio` | MinIO object storage (S3-compatible) | ✅ **Ready** |
| `gdrive` | Google Drive | ✅ **Ready** |

### Pattern Matching

- `**/*.pdf` - All PDF files in any subdirectory
- `**/*.txt` - All text files recursively
- `docs/*.md` - Markdown files in docs folder only
- `*.py` - Python files in root directory only

### Metadata Extraction

Each ingested document automatically includes:
- **filename**: Original file name
- **source**: Source name reference
- **path**: Full file path
- **size**: File size in bytes
- **modified**: Last modification time
- **extension**: File extension
- **ingested_at**: Ingestion timestamp

## Programmatic Usage

For developers who want to integrate Hector into their Go applications:

```go
package main

import (
    "fmt"
    "github.com/kadirpekel/hector"
    "github.com/kadirpekel/hector/providers"
)

func main() {
    providers.RegisterDefaultProviders()
    agent, _ := hector.NewAgentWithDefaults()
    response, _ := agent.ExecuteQueryWithReasoning("Hello!")
    fmt.Println(response)
}
```

### Core API

```go
// Agent Creation
func NewAgentWithDefaults() (*Agent, error)
func NewAgentFromYAML(configPath string) (*Agent, error)
func NewAgent() *Agent

// Query Execution
func (a *Agent) ExecuteQueryWithReasoning(query string) (string, error)
func (a *Agent) ExecuteQuery(query string) (string, error)

// Memory Operations
func (a *Agent) AddDocument(content string, metadata map[string]interface{}) error
func (a *Agent) SearchDocuments(query string, topK int) ([]Document, error)
func (a *Agent) GetConversationHistory() []ConversationEntry

// Document Ingestion
func (a *Agent) SyncModel(modelName string) error
func (a *Agent) SyncAllModels() error
func (a *Agent) GetModelStatus(modelName string) (map[string]interface{}, error)
func (a *Agent) ListModels() []string

// Tool Management
func (a *Agent) GetAvailableTools() []Tool
func (a *Agent) RegisterTool(tool Tool) error
func (a *Agent) ExecuteTool(toolName string, params map[string]interface{}) (interface{}, error)

// Provider Management
func (a *Agent) GetLLMProvider() LLMProvider
func (a *Agent) GetMemoryProvider() VectorDB
func (a *Agent) GetEmbedderProvider() EmbeddingProvider
```

### Configuration Structure

```go
type AgentConfig struct {
    Agent      AgentInfo           `yaml:"agent"`
    LLM        YAMLProviderConfig  `yaml:"llm"`
    Memory     YAMLProviderConfig  `yaml:"memory"`
    Embedder   YAMLProviderConfig  `yaml:"embedder"`
    Reasoning  ReasoningConfig     `yaml:"reasoning"`
    Sources    map[string]SourceConfig `yaml:"sources"`  // Global sources
    Models     []ModelConfig       `yaml:"models"`       // Models with ingestion
}

type ModelConfig struct {
    Name            string                 `yaml:"name"`
    Collection      string                 `yaml:"collection"`
    DefaultTopK     int                    `yaml:"default_top_k"`
    MaxTopK         int                    `yaml:"max_top_k"`
    Ingestion       *ModelIngestionConfig  `yaml:"ingestion"`  // Per-model ingestion
}
```

## License

MIT License - see [LICENSE](LICENSE) file for details.