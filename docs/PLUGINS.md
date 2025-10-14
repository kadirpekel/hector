---
layout: default
title: Plugin Development
nav_order: 6
parent: Advanced
description: "Build custom LLMs, databases, and tools via gRPC plugins"
---

# Plugin Development Guide

## Overview

Hector's plugin system allows you to extend core functionality without modifying Hector's codebase. Write plugins in any language that supports gRPC (Go, Python, Rust, JavaScript, etc.) and integrate them seamlessly.

## Plugin Architecture

### Key Features

- **Language Agnostic**: Write in Go, Python, Rust, JavaScript, or any language with gRPC support
- **Process Isolation**: Plugins run in separate processes for stability and security  
- **gRPC Protocol**: Industry-standard RPC framework for high performance
- **Auto-Discovery**: Plugins can be automatically discovered from configured paths
- **Hot-Reloadable**: Plugins can be updated without restarting Hector (future)

### Plugin Types

| Type | Purpose | Interface |
|------|---------|-----------|
| `llm_provider` | Custom LLM integrations | Generate text, streaming, tool calling |
| `database_provider` | Vector database backends | Store/search embeddings, collections |
| `embedder_provider` | Embedding generation | Convert text to vectors |
| `tool_provider` | Custom tools | Execute domain-specific operations |
| `reasoning_strategy` | Reasoning approaches | Custom agent reasoning patterns |

### Why gRPC Only?

Hector uses gRPC exclusively for plugins (not Go's native plugin system) because:

✅ **Cross-Language**: Write plugins in any language  
✅ **Process Isolation**: Plugin crashes don't affect Hector  
✅ **Production-Ready**: Used by Terraform, Vault, Consul  
✅ **Cross-Platform**: Works on Windows, macOS, Linux  
✅ **Version Independent**: No Go version matching required  
✅ **Network Transparent**: Plugins can run locally or remotely  

---

## Quick Start

### Prerequisites

```bash
# For Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# For Python plugins
pip install grpcio grpcio-tools
```

### 1. Create Your Plugin

**Go Example:**

```go
package main

import (
	"context"
	"fmt"
	"github.com/kadirpekel/hector/plugins/grpc"
)

type MyLLMProvider struct {
	apiKey string
	model  string
}

func (p *MyLLMProvider) Initialize(ctx context.Context, config map[string]string) error {
	p.apiKey = config["api_key"]
	p.model = config["model"]
	fmt.Printf("Initialized with model: %s\n", p.model)
	return nil
}

func (p *MyLLMProvider) Generate(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (*grpc.GenerateResponse, error) {
	// Your LLM logic here
	response := &grpc.GenerateResponse{
		Text:       "Generated response from " + p.model,
		TokensUsed: 50,
	}
	return response, nil
}

func (p *MyLLMProvider) GenerateStreaming(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (<-chan *grpc.StreamChunk, error) {
	chunks := make(chan *grpc.StreamChunk, 10)
	
	go func() {
		defer close(chunks)
		
		chunks <- &grpc.StreamChunk{
			Type: grpc.ChunkTypeText,
			Text: "Streaming ",
		}
		chunks <- &grpc.StreamChunk{
			Type: grpc.ChunkTypeText,
			Text: "response",
		}
		chunks <- &grpc.StreamChunk{
			Type:       grpc.ChunkTypeDone,
			TokensUsed: 25,
		}
	}()
	
	return chunks, nil
}

func (p *MyLLMProvider) GetModelInfo(ctx context.Context) (*grpc.ModelInfo, error) {
	return &grpc.ModelInfo{
		ModelName:   p.model,
		MaxTokens:   4096,
		Temperature: 0.7,
	}, nil
}

func (p *MyLLMProvider) Shutdown(ctx context.Context) error {
	fmt.Println("Shutting down plugin")
	return nil
}

func (p *MyLLMProvider) Health(ctx context.Context) error {
	return nil
}

func main() {
	grpc.ServeLLMPlugin(&MyLLMProvider{})
}
```

### 2. Build Your Plugin

```bash
# Initialize module
go mod init my-llm-plugin
go get github.com/kadirpekel/hector/plugins/grpc
go mod tidy

# Build executable
go build -o my-llm-plugin
chmod +x my-llm-plugin
```

### 3. Create Plugin Manifest

Create `my-llm-plugin.plugin.yaml`:

```yaml
plugin:
  name: "my-llm-plugin"
  version: "1.0.0"
  author: "Your Name <you@example.com>"
  description: "My custom LLM provider"
  homepage: "https://github.com/you/my-llm-plugin"
  license: "MIT"
  
  type: llm_provider
  protocol: grpc
  hector_version: ">=0.1.0"
  
  config_schema:
    required:
      - api_key
      - model
    optional:
      - temperature
      - max_tokens
    defaults:
      temperature: 0.7
      max_tokens: 2000
  
  capabilities:
    streaming: true
    function_calling: false
    vision: false
```

### 4. Configure Hector

Add to `hector.yaml`:

```yaml
plugins:
  llm_providers:
    my-custom-llm:
      name: "my-custom-llm"
      type: grpc
      path: "./plugins/my-llm-plugin"
      enabled: true
      config:
        api_key: "${MY_API_KEY}"
        model: "my-model-v1"
        temperature: 0.7

agents:
  my-agent:
    name: "My Agent"
    llm: "my-custom-llm"
```

### 5. Test Your Plugin

```bash
# Test plugin directly (it will wait for gRPC connections)
./my-llm-plugin

# Run Hector with your plugin
hector serve --config hector.yaml
```

---

## Plugin Types Reference

### LLM Provider Plugin

Custom language model integrations.

**Interface:**
```go
type LLMProvider interface {
    Initialize(ctx context.Context, config map[string]string) error
    Generate(ctx context.Context, messages []*Message, tools []*ToolDefinition) (*GenerateResponse, error)
    GenerateStreaming(ctx context.Context, messages []*Message, tools []*ToolDefinition) (<-chan *StreamChunk, error)
    GetModelInfo(ctx context.Context) (*ModelInfo, error)
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error
}
```

**Serve:** `grpc.ServeLLMPlugin(impl)`

**Use Cases:**
- Integrate proprietary LLMs
- Run local inference servers (Ollama, llama.cpp)
- Implement custom prompt engineering
- Add rate limiting or caching layers

### Database Provider Plugin

Custom vector database backends.

**Interface:**
```go
type DatabaseProvider interface {
    Initialize(ctx context.Context, config map[string]string) error
    Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]string) error
    Search(ctx context.Context, collection string, vector []float32, topK int32) ([]*SearchResult, error)
    Delete(ctx context.Context, collection string, id string) error
    CreateCollection(ctx context.Context, collection string, vectorSize uint64) error
    DeleteCollection(ctx context.Context, collection string) error
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error
}
```

**Serve:** `grpc.ServeDatabasePlugin(impl)`

**Use Cases:**
- Add support for new vector databases (Pinecone, Weaviate, Milvus)
- Implement custom indexing strategies
- Create specialized search algorithms
- Integrate with proprietary storage systems

### Embedder Provider Plugin

Custom embedding generation.

**Interface:**
```go
type EmbedderProvider interface {
    Initialize(ctx context.Context, config map[string]string) error
    Embed(ctx context.Context, text string) ([]float32, error)
    GetEmbedderInfo(ctx context.Context) (*EmbedderInfo, error)
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error
}
```

**Serve:** `grpc.ServeEmbedderPlugin(impl)`

**Use Cases:**
- Use fine-tuned embedding models
- Implement domain-specific embeddings
- Integrate custom embedding services
- Optimize embeddings for specific use cases

---

## Configuration

### Plugin Discovery

Configure automatic plugin discovery:

```yaml
plugins:
  plugin_discovery:
    enabled: true
    paths:
      - "./plugins"
      - "~/.hector/plugins"
      - "/usr/local/hector/plugins"
    scan_subdirectories: true
```

Hector will scan these paths for plugins with `.plugin.yaml` manifests.

### Explicit Plugin Configuration

Define plugins explicitly in your config:

```yaml
plugins:
  llm_providers:
    my-llm:
      name: "my-llm"
      type: grpc
      path: "./plugins/my-llm"
      enabled: true
      config:
        api_key: "${LLM_API_KEY}"
        endpoint: "https://api.example.com"
        timeout: 30
```

### Using Plugins in Agents

Reference plugins by name:

```yaml
agents:
  my-agent:
    name: "My Agent"
    llm: "my-llm"              # Plugin name
    database: "my-db"          # Plugin name
    embedder: "my-embedder"    # Plugin name
```

---

## Examples

### Complete Working Example

See `examples/plugins/echo-llm/` for a complete, working LLM plugin example:

```bash
cd examples/plugins/echo-llm
cat README.md      # Read the guide
cat main.go        # Study the implementation
go build           # Build the plugin
./echo-llm         # Test it directly
```

---

## Protocol Reference

### gRPC Services

Plugin proto definitions are in `plugins/grpc/proto/`:
- `common.proto` - Shared types (Message, ToolDefinition, etc.)
- `llm.proto` - LLM provider service
- `database.proto` - Database provider service
- `embedder.proto` - Embedder provider service

### Generated Code

When you import `github.com/kadirpekel/hector/plugins/grpc`, you get:
- All proto-generated types
- Helper functions for serving plugins
- Constants for chunk types, plugin types, etc.

---

## Current Status

**✅ Implemented:**
- gRPC plugin infrastructure
- Plugin discovery and loading
- Manifest validation
- Process isolation
- Example echo-llm plugin

**🚧 In Progress:**
- Registry integration (plugins load but aren't registered with component registries)
- LLM/Database/Embedder adapter creation
- Plugin health monitoring
- Hot reloading

**📋 Planned:**
- Tool provider plugins
- Reasoning strategy plugins
- Plugin marketplace
- Remote plugin support

