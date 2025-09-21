# Installation & Usage Guide

## Quick Install

### Option 1: CLI Binary (Ready to Run)

```bash
# Install Hector CLI executable - ready to use immediately
go install github.com/kadirpekel/hector/cmd/hector@latest

# Run interactive chat
hector

# Run with config
hector --config config.yaml
```

### Option 2: Go Package (For Development)

```bash
# Install Hector as a Go module for use in your projects
go get github.com/kadirpekel/hector@v1.0.1
```

## Installation Methods Explained

### CLI Binary (`go install github.com/kadirpekel/hector/cmd/hector@latest`)
- **What you get**: A `hector` executable command
- **Ready to use**: Immediately after installation
- **Use case**: Interactive chat, CLI usage, quick testing
- **Example**: `hector` → starts interactive chat

### Go Package (`go get github.com/kadirpekel/hector@v1.0.1`)
- **What you get**: Hector library for import in Go code
- **Ready to use**: After importing and writing code
- **Use case**: Building applications, integrating Hector into projects
- **Example**: `import "github.com/kadirpekel/hector"`

## Basic Usage

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/kadirpekel/hector"
    "github.com/kadirpekel/hector/providers"
)

func main() {
    // Register default providers (required)
    if err := providers.RegisterDefaultProviders(); err != nil {
        log.Fatal(err)
    }
    
    // Create agent with zero-config (assumes local Ollama + Qdrant)
    agent, err := hector.NewAgentWithDefaults()
    if err != nil {
        log.Fatal(err)
    }
    
    // Execute query
    response, err := agent.ExecuteQueryWithReasoning("Hello, how are you?")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response)
}
```

## Configuration-Based Usage

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/kadirpekel/hector"
    "github.com/kadirpekel/hector/providers"
)

func main() {
    // Register providers
    if err := providers.RegisterDefaultProviders(); err != nil {
        log.Fatal(err)
    }
    
    // Create agent from YAML config
    agent, err := hector.NewAgentFromYAML("config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Execute with reasoning
    response, err := agent.ExecuteQueryWithReasoning("What's the weather like?")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response)
}
```

## Prerequisites

### Zero-Config Mode
- **Ollama**: `ollama serve` (localhost:11434)
- **Qdrant**: `docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant`

### Custom Configuration
- Any supported LLM provider (Ollama, OpenAI, TGI)
- Any supported vector database (Qdrant, Chroma, Pinecone)
- Any supported embedder (Ollama, OpenAI, TGI)

## Example Config File

```yaml
# config.yaml
agent:
  name: "My Agent"
  description: "Custom AI agent"

llm:
  name: "ollama"
  model: "llama3.2"
  temperature: 0.7

memory:
  name: "qdrant"
  collection: "my_agent"
  default_top_k: 5

embedder:
  name: "ollama"
  model: "nomic-embed-text"

instruction: |
  You are a helpful AI assistant.

reasoning:
  strategy: "simple"
  max_steps: 1
```

## Advanced Features

### Multi-Step Reasoning
```go
// Use stepped reasoning configuration
agent, err := hector.NewAgentFromYAML("configs/stepped.yaml")
```

### Tool Integration
```go
// Add MCP servers for tool access
agent, err := hector.NewAgentFromYAML("configs/tools.yaml")
```

### Nested Reasoning
```go
// Use nested agent hierarchies
agent, err := hector.NewAgentFromYAML("configs/nested.yaml")
```

## API Reference

### Core Functions
- `hector.NewAgentWithDefaults()` - Zero-config agent
- `hector.NewAgentFromYAML(path)` - Config-based agent
- `agent.ExecuteQueryWithReasoning(query)` - Execute with reasoning
- `agent.ExecuteQueryStreaming(query)` - Streaming execution
- `agent.AddDocument(content, metadata)` - Add to memory
- `agent.SearchDocuments(query, model, topK)` - Search memory

### Provider Registration
- `providers.RegisterDefaultProviders()` - Register all providers
- `providers.RegisterLLMProvider(name, provider)` - Register custom LLM
- `providers.RegisterDatabaseProvider(name, provider)` - Register custom DB
- `providers.RegisterEmbedderProvider(name, provider)` - Register custom embedder

## Version Information

- **Current Version**: v1.0.0
- **Go Version**: 1.23+
- **Dependencies**: Qdrant client, gRPC, YAML parser

## Support

- **Documentation**: [README.md](README.md)
- **Examples**: [configs/](configs/) directory
- **Issues**: [GitHub Issues](https://github.com/kadirpekel/hector/issues)
