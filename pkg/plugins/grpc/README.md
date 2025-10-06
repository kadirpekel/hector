# gRPC Plugin Development

This package provides the gRPC plugin infrastructure for Hector. Use this to create custom plugins in Go.

## Quick Start

### 1. Create Your Plugin

```go
package main

import (
	"context"
	"github.com/kadirpekel/hector/plugins/grpc"
)

// MyLLMProvider implements the LLMProvider interface
type MyLLMProvider struct {
	apiKey string
	model  string
}

func (p *MyLLMProvider) Initialize(ctx context.Context, config map[string]string) error {
	p.apiKey = config["api_key"]
	p.model = config["model"]
	// Initialize your LLM client here
	return nil
}

func (p *MyLLMProvider) Generate(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (*grpc.GenerateResponse, error) {
	// Your generation logic here
	return &grpc.GenerateResponse{
		Text:       "Generated response",
		ToolCalls:  nil,
		TokensUsed: 100,
	}, nil
}

func (p *MyLLMProvider) GenerateStreaming(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (<-chan *grpc.StreamChunk, error) {
	ch := make(chan *grpc.StreamChunk, 10)
	
	go func() {
		defer close(ch)
		
		// Send text chunks
		ch <- &grpc.StreamChunk{
			Type: grpc.ChunkTypeText,
			Text: "Streaming response...",
		}
		
		// Send done signal
		ch <- &grpc.StreamChunk{
			Type: grpc.ChunkTypeDone,
		}
	}()
	
	return ch, nil
}

func (p *MyLLMProvider) GetModelInfo(ctx context.Context) (*grpc.ModelInfo, error) {
	return &grpc.ModelInfo{
		ModelName:   p.model,
		MaxTokens:   4096,
		Temperature: 0.7,
	}, nil
}

func (p *MyLLMProvider) Shutdown(ctx context.Context) error {
	// Clean up resources
	return nil
}

func (p *MyLLMProvider) Health(ctx context.Context) error {
	// Check if plugin is healthy
	return nil
}

func main() {
	// Serve the plugin - this blocks
	grpc.ServeLLMPlugin(&MyLLMProvider{})
}
```

### 2. Build Your Plugin

```bash
go mod init my-llm-plugin
go mod tidy
go build -o my-llm-plugin
chmod +x my-llm-plugin
```

### 3. Create Manifest

Create `my-llm-plugin.plugin.yaml`:

```yaml
plugin:
  name: "my-llm-plugin"
  version: "1.0.0"
  author: "Your Name"
  description: "My custom LLM provider"
  type: llm_provider
  protocol: grpc
  hector_version: ">=0.1.0"
  
  config_schema:
    required:
      - api_key
      - model
    optional:
      - temperature
  
  capabilities:
    streaming: true
    function_calling: false
```

### 4. Configure Hector

Add to `hector.yaml`:

```yaml
plugins:
  llm_providers:
    my-llm:
      type: grpc
      path: "./plugins/my-llm-plugin"
      enabled: true
      config:
        api_key: "${MY_API_KEY}"
        model: "my-model-v1"
        temperature: 0.7

agents:
  my-agent:
    llm: "my-llm"  # Use your plugin
```

### 5. Run

```bash
hector --agent my-agent
```

## Plugin Types

### LLM Provider

Implement the `LLMProvider` interface:

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

Serve with: `grpc.ServeLLMPlugin(impl)`

### Database Provider

Implement the `DatabaseProvider` interface:

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

Serve with: `grpc.ServeDatabasePlugin(impl)`

### Embedder Provider

Implement the `EmbedderProvider` interface:

```go
type EmbedderProvider interface {
    Initialize(ctx context.Context, config map[string]string) error
    Embed(ctx context.Context, text string) ([]float32, error)
    GetEmbedderInfo(ctx context.Context) (*EmbedderInfo, error)
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error
}
```

Serve with: `grpc.ServeEmbedderPlugin(impl)`

## Available Types

All proto-generated types are exported for convenience:

```go
// Import the package
import "github.com/kadirpekel/hector/plugins/grpc"

// Use the types
var msg *grpc.Message
var tool *grpc.ToolDefinition
var response *grpc.GenerateResponse
var chunk *grpc.StreamChunk
var info *grpc.ModelInfo
var result *grpc.SearchResult
var embedderInfo *grpc.EmbedderInfo
```

## Constants

```go
// Plugin types
grpc.PluginTypeLLM       // "llm_provider"
grpc.PluginTypeDatabase  // "database_provider"
grpc.PluginTypeEmbedder  // "embedder_provider"

// Stream chunk types
grpc.ChunkTypeText      // Text content
grpc.ChunkTypeToolCall  // Tool call
grpc.ChunkTypeDone      // Stream finished
grpc.ChunkTypeError     // Error occurred
```

## Best Practices

### 1. Always Validate Configuration

```go
func (p *MyPlugin) Initialize(ctx context.Context, config map[string]string) error {
    apiKey, ok := config["api_key"]
    if !ok || apiKey == "" {
        return fmt.Errorf("api_key is required")
    }
    p.apiKey = apiKey
    return nil
}
```

### 2. Handle Context Cancellation

```go
func (p *MyPlugin) Generate(ctx context.Context, ...) (*grpc.GenerateResponse, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Continue processing
    }
    // ...
}
```

### 3. Clean Up Resources

```go
func (p *MyPlugin) Shutdown(ctx context.Context) error {
    if p.client != nil {
        p.client.Close()
    }
    if p.conn != nil {
        p.conn.Close()
    }
    return nil
}
```

### 4. Implement Health Checks

```go
func (p *MyPlugin) Health(ctx context.Context) error {
    // Ping your service
    if err := p.client.Ping(ctx); err != nil {
        return fmt.Errorf("service unhealthy: %w", err)
    }
    return nil
}
```

### 5. Stream Data Efficiently

```go
func (p *MyPlugin) GenerateStreaming(ctx context.Context, ...) (<-chan *grpc.StreamChunk, error) {
    ch := make(chan *grpc.StreamChunk, 100) // Buffer for smooth streaming
    
    go func() {
        defer close(ch)
        
        // Check context cancellation
        for {
            select {
            case <-ctx.Done():
                ch <- &grpc.StreamChunk{
                    Type:  grpc.ChunkTypeError,
                    Error: ctx.Err().Error(),
                }
                return
            default:
                // Stream your data
                chunk := produceChunk()
                ch <- chunk
            }
        }
    }()
    
    return ch, nil
}
```

## Error Handling

Return meaningful errors:

```go
if err != nil {
    return nil, fmt.Errorf("failed to generate: %w", err)
}
```

For streaming, send error chunks:

```go
ch <- &grpc.StreamChunk{
    Type:  grpc.ChunkTypeError,
    Error: "connection failed",
}
```

## Testing Your Plugin

Test independently before integrating:

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
}
```

## Examples

See `examples/plugins/` for complete working examples:

- `examples/plugins/custom-llm-go/` - Go LLM provider
- `examples/plugins/custom-db-python/` - Python database provider (uses proto directly)
- `examples/plugins/custom-embedder/` - Embedder provider

## Protocol Buffers

The proto definitions are in `proto/`:
- `common.proto` - Shared types
- `llm.proto` - LLM provider service
- `database.proto` - Database provider service
- `embedder.proto` - Embedder provider service

Generated Go code is automatically included when you import this package.

## Need Help?

- See the [main plugin documentation](../../PLUGIN_ARCHITECTURE.md)
- See the [plugin development guide](../../examples/plugins/README.md)
- Check the example plugins in `examples/plugins/`

