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

## Best Practices

### 1. Configuration Validation

Always validate required configuration in `Initialize()`:

```go
func (p *MyPlugin) Initialize(ctx context.Context, config map[string]string) error {
    apiKey, ok := config["api_key"]
    if !ok || apiKey == "" {
        return fmt.Errorf("api_key is required")
    }
    
    endpoint, ok := config["endpoint"]
    if !ok || endpoint == "" {
        return fmt.Errorf("endpoint is required")
    }
    
    p.apiKey = apiKey
    p.endpoint = endpoint
    
    // Initialize your client
    client, err := myapi.NewClient(endpoint, apiKey)
    if err != nil {
        return fmt.Errorf("failed to initialize client: %w", err)
    }
    p.client = client
    
    return nil
}
```

### 2. Context Cancellation

Respect context cancellation in long-running operations:

```go
func (p *MyPlugin) Generate(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (*grpc.GenerateResponse, error) {
    // Check cancellation before expensive operation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Use context in API calls
    response, err := p.client.Generate(ctx, messages)
    if err != nil {
        return nil, fmt.Errorf("generation failed: %w", err)
    }
    
    return response, nil
}
```

### 3. Resource Cleanup

Clean up resources in `Shutdown()`:

```go
func (p *MyPlugin) Shutdown(ctx context.Context) error {
    var errors []error
    
    if p.client != nil {
        if err := p.client.Close(); err != nil {
            errors = append(errors, fmt.Errorf("failed to close client: %w", err))
        }
    }
    
    if p.conn != nil {
        if err := p.conn.Close(); err != nil {
            errors = append(errors, fmt.Errorf("failed to close connection: %w", err))
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("shutdown errors: %v", errors)
    }
    
    return nil
}
```

### 4. Health Checks

Implement meaningful health checks:

```go
func (p *MyPlugin) Health(ctx context.Context) error {
    // Check client initialized
    if p.client == nil {
        return fmt.Errorf("client not initialized")
    }
    
    // Ping the service
    if err := p.client.Ping(ctx); err != nil {
        return fmt.Errorf("service unreachable: %w", err)
    }
    
    return nil
}
```

### 5. Streaming Best Practices

Stream data efficiently with proper buffering and error handling:

```go
func (p *MyPlugin) GenerateStreaming(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (<-chan *grpc.StreamChunk, error) {
    chunks := make(chan *grpc.StreamChunk, 100) // Buffer for smooth streaming
    
    go func() {
        defer close(chunks)
        
        stream, err := p.client.StreamGenerate(ctx, messages)
        if err != nil {
            chunks <- &grpc.StreamChunk{
                Type:  grpc.ChunkTypeError,
                Error: fmt.Sprintf("failed to start stream: %v", err),
            }
            return
        }
        
        var totalTokens int32
        for {
            select {
            case <-ctx.Done():
                chunks <- &grpc.StreamChunk{
                    Type:  grpc.ChunkTypeError,
                    Error: "context cancelled",
                }
                return
            default:
            }
            
            chunk, err := stream.Recv()
            if err == io.EOF {
                // Stream ended successfully
                chunks <- &grpc.StreamChunk{
                    Type:       grpc.ChunkTypeDone,
                    TokensUsed: totalTokens,
                }
                return
            }
            
            if err != nil {
                chunks <- &grpc.StreamChunk{
                    Type:  grpc.ChunkTypeError,
                    Error: fmt.Sprintf("stream error: %v", err),
                }
                return
            }
            
            totalTokens += chunk.Tokens
            chunks <- &grpc.StreamChunk{
                Type: grpc.ChunkTypeText,
                Text: chunk.Text,
            }
        }
    }()
    
    return chunks, nil
}
```

### 6. Error Handling

Return descriptive errors:

```go
func (p *MyPlugin) Generate(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (*grpc.GenerateResponse, error) {
    if len(messages) == 0 {
        return nil, fmt.Errorf("no messages provided")
    }
    
    response, err := p.client.Generate(ctx, convertMessages(messages))
    if err != nil {
        // Include context in errors
        return nil, fmt.Errorf("failed to generate response for %d messages: %w", len(messages), err)
    }
    
    if response.Text == "" {
        return nil, fmt.Errorf("received empty response from API")
    }
    
    return &grpc.GenerateResponse{
        Text:       response.Text,
        TokensUsed: response.Tokens,
    }, nil
}
```

---

## Testing

### Unit Testing

Test your plugin independently:

```go
func TestMyPlugin(t *testing.T) {
    plugin := &MyLLMProvider{}
    
    // Test initialization
    config := map[string]string{
        "api_key": "test-key",
        "model":   "test-model",
    }
    err := plugin.Initialize(context.Background(), config)
    if err != nil {
        t.Fatalf("Initialize failed: %v", err)
    }
    
    // Test generation
    messages := []*grpc.Message{
        {Role: "user", Content: "Hello"},
    }
    response, err := plugin.Generate(context.Background(), messages, nil)
    if err != nil {
        t.Fatalf("Generate failed: %v", err)
    }
    
    if response.Text == "" {
        t.Error("Expected non-empty response")
    }
    
    // Test health
    if err := plugin.Health(context.Background()); err != nil {
        t.Errorf("Health check failed: %v", err)
    }
    
    // Test shutdown
    if err := plugin.Shutdown(context.Background()); err != nil {
        t.Errorf("Shutdown failed: %v", err)
    }
}
```

### Integration Testing

Test with Hector:

```bash
# Create test configuration
cat > test-plugin.yaml << EOF
plugins:
  llm_providers:
    test-plugin:
      name: "test-plugin"
      type: grpc
      path: "./my-llm-plugin"
      enabled: true
      config:
        api_key: "test"

agents:
  test-agent:
    llm: "test-plugin"
EOF

# Run Hector
hector serve --config test-plugin.yaml
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

## Troubleshooting

### Plugin Not Loading

**Check executable permissions:**
```bash
ls -l my-plugin
# Should show: -rwxr-xr-x
chmod +x my-plugin
```

**Check manifest exists:**
```bash
ls my-plugin.plugin.yaml
```

**Enable debug mode:**
```bash
hector serve --config hector.yaml --debug
```

### Plugin Crashes

1. Test plugin independently
2. Check plugin logs
3. Verify configuration
4. Check resource limits

### Configuration Issues

1. Validate YAML syntax: `yamllint hector.yaml`
2. Check environment variables are set
3. Verify required fields are present
4. Review manifest schema

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

---

## Resources

- [gRPC Plugin Infrastructure](../plugins/grpc/README.md)
- [Echo LLM Example](../examples/plugins/echo-llm/README.md)
- [Protocol Buffer Definitions](../plugins/grpc/proto/)
- [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)
- [gRPC Documentation](https://grpc.io/docs/)

---

## Contributing

Have a plugin to share? Open a PR with your plugin in `examples/plugins/`!

- Follow the existing structure
- Include complete documentation
- Add tests
- Provide a working example configuration
