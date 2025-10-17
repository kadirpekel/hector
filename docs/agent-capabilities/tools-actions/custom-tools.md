---
layout: default
title: Custom Tools
nav_order: 3
parent: Tools & Actions
description: "Build custom capabilities via gRPC plugins"
---

# Custom Tools

Build **high-performance custom tools** in any language via gRPC.

## Why gRPC Plugins?

**Use cases:**
- **Custom LLMs** - Integrate proprietary models
- **Performance** - Binary protocol, faster than HTTP/JSON
- **Type safety** - Protocol Buffers with strong typing
- **Language flexibility** - Go, Python, Rust, Java, C++, etc.

**vs MCP:**
- MCP: Quick, HTTP-based, ecosystem of providers
- gRPC: Performance-critical, type-safe, custom requirements

## Plugin Types

Hector supports 3 plugin types:
1. **LLM Providers** - Custom language models
2. **Database Providers** - Custom vector databases
3. **Embedder Providers** - Custom embedding models

(Tool plugins are better via MCP for simplicity)

## Quick Example: Custom LLM Plugin

**1. Create plugin (`main.go`):**
```go
package main

import (
    "context"
    "github.com/kadirpekel/hector/plugins/grpc"
    pb "github.com/kadirpekel/hector/plugins/grpc/proto"
)

type MyLLMProvider struct {
    apiKey string
    model  string
}

func (p *MyLLMProvider) Initialize(ctx context.Context, config map[string]string) error {
    p.apiKey = config["api_key"]
    p.model = config["model"]
    // Initialize your LLM client
    return nil
}

func (p *MyLLMProvider) Generate(ctx context.Context, messages []*pb.Message, tools []*pb.ToolDefinition) (*pb.GenerateResponse, error) {
    // Your generation logic
    return &pb.GenerateResponse{
        Text:       "Generated response from custom LLM",
        ToolCalls:  nil,
        TokensUsed: 100,
    }, nil
}

func (p *MyLLMProvider) GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []*pb.ToolDefinition) (<-chan *pb.StreamChunk, error) {
    ch := make(chan *pb.StreamChunk, 10)
    go func() {
        defer close(ch)
        ch <- &pb.StreamChunk{Type: pb.StreamChunk_TEXT, Text: "Streaming..."}
        ch <- &pb.StreamChunk{Type: pb.StreamChunk_DONE}
    }()
    return ch, nil
}

func (p *MyLLMProvider) GetModelInfo(ctx context.Context) (*pb.ModelInfo, error) {
    return &pb.ModelInfo{
        ModelName:   p.model,
        MaxTokens:   4096,
        Temperature: 0.7,
    }, nil
}

func (p *MyLLMProvider) Shutdown(ctx context.Context) error {
    return nil
}

func (p *MyLLMProvider) Health(ctx context.Context) error {
    return nil
}

func main() {
    grpc.ServeLLMPlugin(&MyLLMProvider{})
}
```

**2. Build:**
```bash
go mod init my-llm-plugin
go get github.com/kadirpekel/hector/plugins/grpc
go build -o my-llm-plugin
chmod +x my-llm-plugin
```

**3. Create manifest (`my-llm-plugin.plugin.yaml`):**
```yaml
name: "my-llm-plugin"
version: "1.0.0"
type: "llm"
description: "Custom LLM provider"
author: "Your Name"
```

**4. Configure in Hector:**
```yaml
llms:
  my_custom_llm:
    type: "plugin"
    plugin_path: "./my-llm-plugin"
    config:
      api_key: "${MY_API_KEY}"
      model: "my-model-v1"
```

## Plugin Development

### LLM Provider Interface

```go
type LLMProvider interface {
    Initialize(ctx context.Context, config map[string]string) error
    Generate(ctx context.Context, messages []*pb.Message, tools []*pb.ToolDefinition) (*pb.GenerateResponse, error)
    GenerateStreaming(ctx context.Context, messages []*pb.Message, tools []*pb.ToolDefinition) (<-chan *pb.StreamChunk, error)
    GetModelInfo(ctx context.Context) (*pb.ModelInfo, error)
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error
}
```

### Database Provider Interface

```go
type DatabaseProvider interface {
    Initialize(ctx context.Context, config map[string]string) error
    CreateCollection(ctx context.Context, name string, config *pb.CollectionConfig) error
    Insert(ctx context.Context, collection string, points []*pb.Point) error
    Search(ctx context.Context, collection string, query *pb.SearchQuery) ([]*pb.SearchResult, error)
    Delete(ctx context.Context, collection string, ids []string) error
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error
}
```

## Configuration

```yaml
# LLM Plugin
llms:
  custom_llm:
    type: "plugin"
    plugin_path: "./my-llm-plugin"
    config:
      api_key: "${API_KEY}"
      model: "custom-model"

# Database Plugin
databases:
  custom_db:
    type: "plugin"
    plugin_path: "./my-db-plugin"
    config:
      connection_string: "${DB_URL}"
      collection: "my_collection"
```

## Best Practices

1. **Start Simple**: Begin with basic functionality
2. **Handle Errors**: Implement proper error handling
3. **Test Locally**: Use local development setup
4. **Monitor Performance**: Track plugin performance
5. **Document Configuration**: Clear config options

## See Also

- **[Built-in Tools](built-in-tools)** - Hector's 5 core tools
- **[MCP Integration](mcp-integration)** - Connect to external tools
- **[Plugin Development](../development/PLUGINS)** - Complete plugin guide
