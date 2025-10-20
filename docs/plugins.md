---
title: Plugin Development
description: Build custom LLMs, databases, and tools via gRPC plugins
---

# Plugin Development Guide

## Overview

Hector's plugin system allows you to extend core functionality without modifying Hector's codebase. Write plugins in any language that supports gRPC (Go, Python, Rust, JavaScript, etc.) and integrate them seamlessly.

## Plugin Architecture

### Key Features

- **Language Agnostic** - Write in Go, Python, Rust, JavaScript, or any language with gRPC support
- **Process Isolation** - Plugins run in separate processes for stability and security  
- **gRPC Protocol** - Industry-standard RPC framework for high performance
- **Auto-Discovery** - Plugins can be automatically discovered from configured paths
- **Hot-Reloadable** - Plugins can be updated without restarting Hector (future)

### Plugin Types

| Type | Purpose | Interface |
|------|---------|-----------|
| **`llm_provider`** | Custom LLM integrations | Generate text, streaming, tool calling |
| **`database_provider`** | Vector database backends | Store/search embeddings, collections |
| **`embedder_provider`** | Embedding generation | Convert text to vectors |
| **`tool_provider`** | Custom tools | Execute domain-specific operations |
| **`reasoning_strategy`** | Reasoning approaches | Custom agent reasoning patterns |

### Why gRPC Only?

Hector uses gRPC exclusively for plugins (not Go's native plugin system) because:

-   **Cross-Language** - Write plugins in any language  
-   **Process Isolation** - Plugin crashes don't affect Hector  
-   **Production-Ready** - Used by Terraform, Vault, Consul  
-   **Cross-Platform** - Works on Windows, macOS, Linux  
-   **Version Independent** - No Go version matching required  
-   **Network Transparent** - Plugins can run locally or remotely  

---

## Quick Start

### Prerequisites

- **Go 1.24+** (for building Hector)
- **gRPC** - Any language with gRPC support
- **Protobuf** - Protocol buffer compiler
- **IDE** - Your preferred development environment


```
┌──────────────────────────────────────────────────────────────┐
│                    Hector Core                               │
│  ┌─────────────┬─────────────┬─────────────┐                 │
│  │   Runtime   │   Plugin    │   Service   │                 │
│  │   System    │   Manager   │   Registry  │                 │
│  └──────┬──────┴──────┬──────┴──────┬──────┘                 │
│         │             │             │                        │
│         ▼             ▼             ▼                        │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                   Plugin Types                          │ │
│  │  ┌─────────┬─────────┬─────────┬─────────┬─────────┐    │ │
│  │  │   LLM   │    DB   │Embedder │  Tool   │Reasoning│    │ │
│  │  │Provider │Provider │Provider │Provider │Strategy │    │ │
│  │  └────┬────┴────┬────┴────┬────┴────┬────┴────┬────┘    │ │
│  │       │         │        │         │         │          │ │
│  │       ▼         ▼        ▼         ▼         ▼          │ │
│  │  ┌─────────────────────────────────────────────────────┐│ │
│  │  │              Plugin Interface                       ││ │
│  │  │  ┌─────────────┬─────────────┐                      ││ │
│  │  │  │   gRPC      │ Protobuf    │                      ││ │
│  │  │  │ Interface   │ Messages    │                      ││ │
│  │  │  └─────────────┴─────────────┘                      ││ │
│  │  └─────────────────────────────────────────────────────┘│ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```
---

## LLM Provider Plugin

Create custom language model integrations.

### LLM Provider Interface

```protobuf
service LLMProvider {
  rpc GenerateText(GenerateTextRequest) returns (GenerateTextResponse);
  rpc StreamText(StreamTextRequest) returns (stream StreamTextResponse);
  rpc CallTool(CallToolRequest) returns (CallToolResponse);
  rpc ListModels(ListModelsRequest) returns (ListModelsResponse);
}
```

### Go Implementation Example

```go
package main

import (
    "context"
    "log"
    "net"
    
    "google.golang.org/grpc"
    pb "github.com/kadirpekel/hector/pkg/plugins/grpc/pb"
)

type CustomLLMProvider struct {
    pb.UnimplementedLLMProviderServer
    apiKey string
}

func (s *CustomLLMProvider) GenerateText(ctx context.Context, req *pb.GenerateTextRequest) (*pb.GenerateTextResponse, error) {
    // Custom LLM API call
    response, err := s.callCustomLLM(req.Prompt, req.Model)
    if err != nil {
        return nil, err
    }
    
    return &pb.GenerateTextResponse{
        Text: response.Text,
        Usage: &pb.TokenUsage{
            PromptTokens:     response.PromptTokens,
            CompletionTokens: response.CompletionTokens,
            TotalTokens:      response.TotalTokens,
        },
    }, nil
}

func (s *CustomLLMProvider) StreamText(ctx context.Context, req *pb.StreamTextRequest) (*pb.StreamTextResponse, error) {
    // Streaming implementation
    stream := make(chan *pb.StreamTextResponse)
    
    go func() {
        defer close(stream)
        
        // Stream tokens from custom LLM
        for token := range s.streamCustomLLM(req.Prompt, req.Model) {
            stream <- &pb.StreamTextResponse{
                Text: token,
                Done: false,
            }
        }
        
        stream <- &pb.StreamTextResponse{
            Text: "",
            Done: true,
        }
    }()
    
    return stream, nil
}

func main() {
    lis, err := net.Listen("tcp", ":8081")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    
    s := grpc.NewServer()
    pb.RegisterLLMProviderServer(s, &CustomLLMProvider{
        apiKey: os.Getenv("CUSTOM_LLM_API_KEY"),
    })
    
    log.Println("Custom LLM provider starting on :8081")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

### Python Implementation Example

```python
import grpc
from concurrent import futures
import asyncio
from hector_plugins_pb2 import *
from hector_plugins_pb2_grpc import *

class CustomLLMProvider(LLMProviderServicer):
    def __init__(self, api_key: str):
        self.api_key = api_key
    
    async def GenerateText(self, request, context):
        # Custom LLM API call
        response = await self.call_custom_llm(request.prompt, request.model)
        
        return GenerateTextResponse(
            text=response.text,
            usage=TokenUsage(
                prompt_tokens=response.prompt_tokens,
                completion_tokens=response.completion_tokens,
                total_tokens=response.total_tokens
            )
        )
    
    async def StreamText(self, request, context):
        # Streaming implementation
        async for token in self.stream_custom_llm(request.prompt, request.model):
            yield StreamTextResponse(
                text=token,
                done=False
            )
        
        yield StreamTextResponse(
            text="",
            done=True
        )

async def serve():
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    add_LLMProviderServicer_to_server(CustomLLMProvider(os.getenv("CUSTOM_LLM_API_KEY")), server)
    
    listen_addr = '[::]:8081'
    server.add_insecure_port(listen_addr)
    
    print(f"Custom LLM provider starting on {listen_addr}")
    await server.start()
    await server.wait_for_termination()

if __name__ == '__main__':
    asyncio.run(serve())
```

### Configuration

```yaml
plugins:
  llm_providers:
    custom_llm:
      type: "grpc"
      path: "./plugins/custom-llm-plugin"
      
      config:
        api_key: "${CUSTOM_LLM_API_KEY}"
        endpoint: "http://localhost:8081"
        timeout: "30s"

llms:
  custom:
    type: "plugin:custom_llm"
    model: "custom-model-v1"
    temperature: 0.7
    max_tokens: 4000
```

---

## Database Provider Plugin

Create custom vector database integrations.

### Database Provider Interface

```protobuf
service DatabaseProvider {
  rpc CreateCollection(CreateCollectionRequest) returns (CreateCollectionResponse);
  rpc DeleteCollection(DeleteCollectionRequest) returns (DeleteCollectionResponse);
  rpc UpsertVectors(UpsertVectorsRequest) returns (UpsertVectorsResponse);
  rpc SearchVectors(SearchVectorsRequest) returns (SearchVectorsResponse);
  rpc DeleteVectors(DeleteVectorsRequest) returns (DeleteVectorsResponse);
}
```

### Go Implementation Example

```go
package main

import (
    "context"
    "log"
    "net"
    
    "google.golang.org/grpc"
    pb "github.com/kadirpekel/hector/pkg/plugins/grpc/pb"
)

type CustomDatabaseProvider struct {
    pb.UnimplementedDatabaseProviderServer
    client *CustomDBClient
}

func (s *CustomDatabaseProvider) CreateCollection(ctx context.Context, req *pb.CreateCollectionRequest) (*pb.CreateCollectionResponse, error) {
    err := s.client.CreateCollection(req.Name, req.Dimension)
    if err != nil {
        return &pb.CreateCollectionResponse{
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    
    return &pb.CreateCollectionResponse{
        Success: true,
    }, nil
}

func (s *CustomDatabaseProvider) UpsertVectors(ctx context.Context, req *pb.UpsertVectorsRequest) (*pb.UpsertVectorsResponse, error) {
    vectors := make([]*CustomVector, len(req.Vectors))
    for i, v := range req.Vectors {
        vectors[i] = &CustomVector{
            ID:       v.Id,
            Vector:   v.Vector,
            Metadata: v.Metadata,
        }
    }
    
    err := s.client.UpsertVectors(req.Collection, vectors)
    if err != nil {
        return &pb.UpsertVectorsResponse{
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    
    return &pb.UpsertVectorsResponse{
        Success: true,
    }, nil
}

func (s *CustomDatabaseProvider) SearchVectors(ctx context.Context, req *pb.SearchVectorsRequest) (*pb.SearchVectorsResponse, error) {
    results, err := s.client.SearchVectors(req.Collection, req.Vector, req.Limit, req.Threshold)
    if err != nil {
        return &pb.SearchVectorsResponse{
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    
    pbResults := make([]*pb.VectorResult, len(results))
    for i, r := range results {
        pbResults[i] = &pb.VectorResult{
            Id:       r.ID,
            Score:    r.Score,
            Metadata: r.Metadata,
        }
    }
    
    return &pb.SearchVectorsResponse{
        Success: true,
        Results: pbResults,
    }, nil
}

func main() {
    lis, err := net.Listen("tcp", ":8082")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    
    s := grpc.NewServer()
    pb.RegisterDatabaseProviderServer(s, &CustomDatabaseProvider{
        client: NewCustomDBClient(os.Getenv("CUSTOM_DB_URL")),
    })
    
    log.Println("Custom database provider starting on :8082")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

### Configuration

```yaml
plugins:
  database_providers:
    custom_db:
      type: "grpc"
      path: "./plugins/custom-db-plugin"
      
      config:
        url: "${CUSTOM_DB_URL}"
        api_key: "${CUSTOM_DB_API_KEY}"
        timeout: "30s"

databases:
  custom:
    type: "plugin:custom_db"
    config:
      collection: "my_collection"
      dimension: 768
```

---

## Embedder Provider Plugin

Create custom embedding generation services.

### Embedder Provider Interface

```protobuf
service EmbedderProvider {
  rpc EmbedText(EmbedTextRequest) returns (EmbedTextResponse);
  rpc EmbedBatch(EmbedBatchRequest) returns (EmbedBatchResponse);
  rpc GetDimensions(GetDimensionsRequest) returns (GetDimensionsResponse);
}
```

### Go Implementation Example

```go
package main

import (
    "context"
    "log"
    "net"
    
    "google.golang.org/grpc"
    pb "github.com/kadirpekel/hector/pkg/plugins/grpc/pb"
)

type CustomEmbedderProvider struct {
    pb.UnimplementedEmbedderProviderServer
    client *CustomEmbedderClient
}

func (s *CustomEmbedderProvider) EmbedText(ctx context.Context, req *pb.EmbedTextRequest) (*pb.EmbedTextResponse, error) {
    embedding, err := s.client.EmbedText(req.Text)
    if err != nil {
        return &pb.EmbedTextResponse{
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    
    return &pb.EmbedTextResponse{
        Success: true,
        Vector:  embedding,
    }, nil
}

func (s *CustomEmbedderProvider) EmbedBatch(ctx context.Context, req *pb.EmbedBatchRequest) (*pb.EmbedBatchResponse, error) {
    embeddings, err := s.client.EmbedBatch(req.Texts)
    if err != nil {
        return &pb.EmbedBatchResponse{
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    
    pbEmbeddings := make([]*pb.Embedding, len(embeddings))
    for i, e := range embeddings {
        pbEmbeddings[i] = &pb.Embedding{
            Vector: e.Vector,
        }
    }
    
    return &pb.EmbedBatchResponse{
        Success:    true,
        Embeddings: pbEmbeddings,
    }, nil
}

func main() {
    lis, err := net.Listen("tcp", ":8083")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    
    s := grpc.NewServer()
    pb.RegisterEmbedderProviderServer(s, &CustomEmbedderProvider{
        client: NewCustomEmbedderClient(os.Getenv("CUSTOM_EMBEDDER_URL")),
    })
    
    log.Println("Custom embedder provider starting on :8083")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

### Configuration

```yaml
plugins:
  embedder_providers:
    custom_embedder:
      type: "grpc"
      path: "./plugins/custom-embedder-plugin"
      
      config:
        url: "${CUSTOM_EMBEDDER_URL}"
        api_key: "${CUSTOM_EMBEDDER_API_KEY}"
        timeout: "30s"

embedders:
  custom:
    type: "plugin:custom_embedder"
    config:
      model: "custom-embedding-model"
      dimension: 768
```

---

## Tool Provider Plugin

Create custom tools for domain-specific operations.

### Tool Provider Interface

```protobuf
service ToolProvider {
  rpc ListTools(ListToolsRequest) returns (ListToolsResponse);
  rpc ExecuteTool(ExecuteToolRequest) returns (ExecuteToolResponse);
  rpc GetToolSchema(GetToolSchemaRequest) returns (GetToolSchemaResponse);
}
```

### Go Implementation Example

```go
package main

import (
    "context"
    "log"
    "net"
    
    "google.golang.org/grpc"
    pb "github.com/kadirpekel/hector/pkg/plugins/grpc/pb"
)

type CustomToolProvider struct {
    pb.UnimplementedToolProviderServer
}

func (s *CustomToolProvider) ListTools(ctx context.Context, req *pb.ListToolsRequest) (*pb.ListToolsResponse, error) {
    tools := []*pb.Tool{
        {
            Name:        "custom_api_call",
            Description: "Make API calls to external services",
            Parameters: map[string]interface{}{
                "url":    "string",
                "method": "string",
                "headers": "object",
                "body":   "string",
            },
        },
        {
            Name:        "custom_data_processing",
            Description: "Process data using custom algorithms",
            Parameters: map[string]interface{}{
                "data":      "array",
                "algorithm": "string",
                "options":   "object",
            },
        },
    }
    
    return &pb.ListToolsResponse{
        Tools: tools,
    }, nil
}

func (s *CustomToolProvider) ExecuteTool(ctx context.Context, req *pb.ExecuteToolRequest) (*pb.ExecuteToolResponse, error) {
    switch req.Tool {
    case "custom_api_call":
        result, err := s.executeAPICall(req.Parameters)
        if err != nil {
            return &pb.ExecuteToolResponse{
                Success: false,
                Error:   err.Error(),
            }, nil
        }
        
        return &pb.ExecuteToolResponse{
            Success: true,
            Result:  result,
        }, nil
        
    case "custom_data_processing":
        result, err := s.processData(req.Parameters)
        if err != nil {
            return &pb.ExecuteToolResponse{
                Success: false,
                Error:   err.Error(),
            }, nil
        }
        
        return &pb.ExecuteToolResponse{
            Success: true,
            Result:  result,
        }, nil
        
    default:
        return &pb.ExecuteToolResponse{
            Success: false,
            Error:   "Unknown tool: " + req.Tool,
        }, nil
    }
}

func main() {
    lis, err := net.Listen("tcp", ":8084")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    
    s := grpc.NewServer()
    pb.RegisterToolProviderServer(s, &CustomToolProvider{})
    
    log.Println("Custom tool provider starting on :8084")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

### Configuration

```yaml
plugins:
  tool_providers:
    custom_tools:
      type: "grpc"
      path: "./plugins/custom-tools-plugin"
      
      config:
        api_key: "${CUSTOM_TOOLS_API_KEY}"
        timeout: "30s"

tools:
  custom_api_call:
    type: "plugin:custom_tools"
    
    config:
      tool_name: "custom_api_call"
  
  custom_data_processing:
    type: "plugin:custom_tools"
    
    config:
      tool_name: "custom_data_processing"
```

---

## Reasoning Strategy Plugin

Create custom reasoning strategies for agent decision-making.

### Reasoning Strategy Interface

```protobuf
service ReasoningStrategy {
  rpc Initialize(InitializeRequest) returns (InitializeResponse);
  rpc ProcessIteration(ProcessIterationRequest) returns (ProcessIterationResponse);
  rpc Finalize(FinalizeRequest) returns (FinalizeResponse);
}
```

### Go Implementation Example

```go
package main

import (
    "context"
    "log"
    "net"
    
    "google.golang.org/grpc"
    pb "github.com/kadirpekel/hector/pkg/plugins/grpc/pb"
)

type CustomReasoningStrategy struct {
    pb.UnimplementedReasoningStrategyServer
}

func (s *CustomReasoningStrategy) Initialize(ctx context.Context, req *pb.InitializeRequest) (*pb.InitializeResponse, error) {
    // Initialize custom reasoning strategy
    state := &CustomReasoningState{
        MaxIterations: req.MaxIterations,
        Temperature:   req.Temperature,
        Context:      req.Context,
    }
    
    return &pb.InitializeResponse{
        Success: true,
        State:   state,
    }, nil
}

func (s *CustomReasoningStrategy) ProcessIteration(ctx context.Context, req *pb.ProcessIterationRequest) (*pb.ProcessIterationResponse, error) {
    // Process single reasoning iteration
    result, err := s.processIteration(req.State, req.Input)
    if err != nil {
        return &pb.ProcessIterationResponse{
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    
    return &pb.ProcessIterationResponse{
        Success: true,
        Result:  result,
        Done:    result.Done,
        State:   result.State,
    }, nil
}

func (s *CustomReasoningStrategy) Finalize(ctx context.Context, req *pb.FinalizeRequest) (*pb.FinalizeResponse, error) {
    // Finalize reasoning process
    finalResult, err := s.finalize(req.State)
    if err != nil {
        return &pb.FinalizeResponse{
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    
    return &pb.FinalizeResponse{
        Success: true,
        Result:  finalResult,
    }, nil
}

func main() {
    lis, err := net.Listen("tcp", ":8085")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    
    s := grpc.NewServer()
    pb.RegisterReasoningStrategyServer(s, &CustomReasoningStrategy{})
    
    log.Println("Custom reasoning strategy starting on :8085")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

### Configuration

```yaml
plugins:
  reasoning_strategies:
    custom_reasoning:
      type: "grpc"
      path: "./plugins/custom-reasoning-plugin"
      
      config:
        max_iterations: 20
        temperature: 0.7

agents:
  my_agent:
    name: "My Agent"
    llm: "gpt-4o"
    reasoning:
      engine: "plugin:custom_reasoning"
      max_iterations: 20
      enable_streaming: true
```

---

## Plugin Development Workflow

### 1. **Setup Development Environment**

```bash
# Clone Hector repository
git clone https://github.com/kadirpekel/hector.git
cd hector

# Install dependencies
go mod download

# Generate protobuf files
make generate-proto
```

### 2. **Create Plugin Project**

```bash
# Create plugin directory
mkdir my-plugin
cd my-plugin

# Initialize Go module
go mod init my-plugin

# Add Hector protobuf dependency
go get github.com/kadirpekel/hector/pkg/plugins/grpc/pb
```

### 3. **Implement Plugin Interface**

```go
// main.go
package main

import (
    "context"
    "log"
    "net"
    
    "google.golang.org/grpc"
    pb "github.com/kadirpekel/hector/pkg/plugins/grpc/pb"
)

type MyPlugin struct {
    pb.UnimplementedLLMProviderServer
}

func (s *MyPlugin) GenerateText(ctx context.Context, req *pb.GenerateTextRequest) (*pb.GenerateTextResponse, error) {
    // Implement your custom logic
    return &pb.GenerateTextResponse{
        Text: "Custom response",
    }, nil
}

func main() {
    lis, err := net.Listen("tcp", ":8081")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    
    s := grpc.NewServer()
    pb.RegisterLLMProviderServer(s, &MyPlugin{})
    
    log.Println("My plugin starting on :8081")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

### 4. **Build and Test Plugin**

```bash
# Build plugin
go build -o my-plugin main.go

# Test plugin
./my-plugin
```

### 5. **Configure Hector**

```yaml
plugins:
  llm_providers:
    my_plugin:
      type: "grpc"
      path: "./my-plugin"
      
      config:
        api_key: "${MY_PLUGIN_API_KEY}"

llms:
  custom:
    type: "plugin:my_plugin"
    model: "custom-model"
```

### 6. **Test Integration**

```bash
# Start Hector with plugin
hector serve --config config.yaml

# Test plugin functionality
hector call my_agent "Hello" --llm custom
```

---

## Plugin Testing

### Unit Testing

```go
package main

import (
    "context"
    "testing"
    
    pb "github.com/kadirpekel/hector/pkg/plugins/grpc/pb"
)

func TestMyPlugin(t *testing.T) {
    plugin := &MyPlugin{}
    
    req := &pb.GenerateTextRequest{
        Prompt: "Hello, world!",
        Model:  "custom-model",
    }
    
    resp, err := plugin.GenerateText(context.Background(), req)
    if err != nil {
        t.Fatalf("GenerateText failed: %v", err)
    }
    
    if resp.Text == "" {
        t.Error("Expected non-empty response")
    }
}
```

### Integration Testing

```yaml
# test-config.yaml
plugins:
  llm_providers:
    test_plugin:
      type: "grpc"
      path: "./test-plugin"
      
      config:
        api_key: "test-key"

llms:
  test:
    type: "plugin:test_plugin"
    model: "test-model"

agents:
  test_agent:
    name: "Test Agent"
    llm: "test"
```

```bash
# Run integration tests
hector serve --config test-config.yaml &
sleep 5
hector call test_agent "Test message"
```

---

## Plugin Packaging

### Docker Packaging

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o my-plugin main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/my-plugin .
CMD ["./my-plugin"]
```

### Distribution

```bash
# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o my-plugin-linux main.go
GOOS=windows GOARCH=amd64 go build -o my-plugin-windows.exe main.go
GOOS=darwin GOARCH=amd64 go build -o my-plugin-macos main.go

# Create distribution package
tar -czf my-plugin-v1.0.0.tar.gz my-plugin-*
```

---

## Plugin Security

### Security Best Practices

- **Input Validation** - Validate all inputs
- **Authentication** - Implement proper authentication
- **Timeouts** - Set appropriate timeouts
- **Sandboxing** - Run in isolated environments
- **Logging** - Log security events

### Security Configuration

```yaml
plugins:
  my_plugin:
    type: "grpc"
    path: "./my-plugin"
    
    
    # Security settings
    security:
      sandbox: true
      timeout: "30s"
      max_memory: "1GB"
      allowed_networks: ["localhost"]
    
    config:
      api_key: "${MY_PLUGIN_API_KEY}"
```

---

