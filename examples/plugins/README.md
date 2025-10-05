# Hector Plugin Development Guide

This directory contains example plugins to help you understand how to extend Hector with custom implementations.

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Plugin Types](#plugin-types)
4. [Creating Your First Plugin](#creating-your-first-plugin)
5. [Plugin Manifest](#plugin-manifest)
6. [Configuration](#configuration)
7. [Example Plugins](#example-plugins)
8. [Testing](#testing)
9. [Best Practices](#best-practices)

---

## Overview

Hector supports dynamic plugin loading, allowing you to:
- Implement custom LLM providers for proprietary models
- Add support for new vector databases
- Create custom embedding providers
- Build specialized tool providers
- Develop advanced reasoning strategies

### Supported Plugin Types

| Type | Purpose | Example Use Cases |
|------|---------|------------------|
| `llm_provider` | Custom LLM integrations | Proprietary models, local inference, custom APIs |
| `database_provider` | Vector database backends | Custom databases, specialized storage |
| `embedder_provider` | Embedding generation | Custom embeddings, fine-tuned models |
| `tool_provider` | Tool implementations | Domain-specific tools, API integrations |
| `reasoning_strategy` | Reasoning approaches | Advanced reasoning, domain-specific logic |

### Plugin Protocol: gRPC Only

Hector uses **gRPC exclusively** for maximum flexibility and production-readiness:

✅ **Language-Agnostic** - Write plugins in Python, Rust, Go, JavaScript, etc.
✅ **Process Isolation** - Plugin crashes don't affect Hector
✅ **Production-Ready** - Used by Terraform, Vault, and other major tools
✅ **Cross-Platform** - Works on Windows, macOS, and Linux
✅ **Version-Independent** - No Go version matching required

---

## Quick Start

### 1. Choose a Plugin Type

Decide what type of plugin you want to create:

```bash
# For a custom LLM provider
cd examples/plugins/custom-llm-go

# For a custom database
cd examples/plugins/custom-db-python

# For custom reasoning
cd examples/plugins/custom-reasoning
```

### 2. Review the Example

Each example directory contains:
- Plugin source code
- `.plugin.yaml` manifest
- Build instructions
- Configuration example

### 3. Build the Plugin

```bash
# For Go plugins
go build -o my-plugin

# For Python plugins
python3 build.py

# Ensure executable permissions
chmod +x my-plugin
```

### 4. Create the Manifest

Create a `.plugin.yaml` file:

```yaml
plugin:
  name: "my-custom-plugin"
  version: "1.0.0"
  author: "Your Name"
  description: "My custom implementation"
  type: llm_provider
  protocol: grpc
  hector_version: ">=0.1.0"
  
  config_schema:
    required:
      - api_key
    optional:
      - temperature
      - max_tokens
```

### 5. Configure Hector

Add to your `hector.yaml`:

```yaml
plugins:
  llm_providers:
    my-custom-llm:
      type: grpc
      path: "./plugins/my-plugin"
      enabled: true
      config:
        api_key: "${MY_API_KEY}"
        temperature: 0.7

agents:
  my-agent:
    llm: "my-custom-llm"  # Use your plugin
```

### 6. Run Hector

```bash
hector --agent my-agent
```

---

## Plugin Types

### LLM Provider Plugin

Implement custom language model integrations.

**Interface:**
```go
type LLMProviderPlugin interface {
    Initialize(ctx context.Context, config map[string]interface{}) error
    Generate(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error)
    GenerateStreaming(ctx context.Context, messages []Message, tools []ToolDefinition) (StreamReader, error)
    GetModelName() string
    GetMaxTokens() int
    GetTemperature() float64
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error
}
```

**Use Cases:**
- Integrate proprietary LLM APIs
- Run local inference servers
- Implement custom prompt engineering
- Add specialized model capabilities

### Database Provider Plugin

Implement custom vector database backends.

**Interface:**
```go
type DatabaseProviderPlugin interface {
    Initialize(ctx context.Context, config map[string]interface{}) error
    Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error
    Search(ctx context.Context, collection string, vector []float32, topK int) ([]*SearchResult, error)
    Delete(ctx context.Context, collection string, id string) error
    CreateCollection(ctx context.Context, collection string, vectorSize uint64) error
    DeleteCollection(ctx context.Context, collection string) error
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error
}
```

**Use Cases:**
- Add support for new vector databases
- Implement custom indexing strategies
- Create specialized search algorithms
- Integrate with proprietary storage systems

### Embedder Provider Plugin

Implement custom embedding generation.

**Interface:**
```go
type EmbedderProviderPlugin interface {
    Initialize(ctx context.Context, config map[string]interface{}) error
    Embed(ctx context.Context, text string) ([]float32, error)
    GetDimension() int
    GetModelName() string
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error
}
```

**Use Cases:**
- Use fine-tuned embedding models
- Implement domain-specific embeddings
- Integrate custom embedding services
- Optimize embeddings for specific use cases

### Tool Provider Plugin

Implement custom tools.

**Interface:**
```go
type ToolProviderPlugin interface {
    Initialize(ctx context.Context, config map[string]interface{}) error
    DiscoverTools(ctx context.Context) error
    ListTools() []*ToolInfo
    GetTool(name string) (Tool, bool)
    ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error)
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error
}
```

**Use Cases:**
- Create domain-specific tools
- Integrate external APIs
- Implement specialized capabilities
- Build tool chains

### Reasoning Strategy Plugin

Implement custom reasoning approaches.

**Interface:**
```go
type ReasoningStrategyPlugin interface {
    Initialize(ctx context.Context, config map[string]interface{}) error
    PrepareIteration(iteration int, state *ReasoningState) error
    ShouldStop(text string, toolCalls []*ToolCall, state *ReasoningState) bool
    AfterIteration(iteration int, text string, toolCalls []*ToolCall, results []*ToolResult, state *ReasoningState) error
    GetContextInjection(state *ReasoningState) string
    GetPromptSlots() *PromptSlots
    GetName() string
    GetDescription() string
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) error
}
```

**Use Cases:**
- Implement advanced reasoning patterns (Tree of Thoughts, etc.)
- Create domain-specific reasoning
- Optimize for specific problem types
- Experiment with novel approaches

---

## Creating Your First Plugin

### Step 1: Set Up Your Plugin Project

```bash
mkdir -p my-first-plugin
cd my-first-plugin
```

### Step 2: Implement Your Plugin

#### Go Plugin Example

```go
// main.go
package main

import (
    "context"
    "fmt"
    "github.com/hashicorp/go-plugin"
    "github.com/kadirpekel/hector/plugins/grpc"
)

type MyLLMProvider struct {
    config map[string]interface{}
}

func (p *MyLLMProvider) Initialize(ctx context.Context, config map[string]interface{}) error {
    p.config = config
    fmt.Println("Plugin initialized with config:", config)
    return nil
}

func (p *MyLLMProvider) Generate(ctx context.Context, messages []grpc.Message, tools []grpc.ToolDefinition) (*grpc.LLMResponse, error) {
    // Your implementation here
    return &grpc.LLMResponse{
        Text: "Hello from my custom LLM!",
        ToolCalls: nil,
        TokensUsed: 10,
    }, nil
}

// Implement other required methods...

func main() {
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: grpc.HandshakeConfig,
        Plugins: map[string]plugin.Plugin{
            "llm_provider": &grpc.LLMProviderPlugin{Impl: &MyLLMProvider{}},
        },
        GRPCServer: plugin.DefaultGRPCServer,
    })
}
```

#### Python Plugin Example

Python plugins work the same way - implement the gRPC service defined in the `.proto` files:

```python
# plugin.py
import grpc
from concurrent import futures
from hector_plugin_pb2 import LLMResponse
from hector_plugin_pb2_grpc import LLMProviderServicer

class MyLLMProvider(LLMProviderServicer):
    def __init__(self):
        self.config = {}
    
    def Initialize(self, request, context):
        self.config = dict(request.config)
        return InitializeResponse(success=True)
    
    def Generate(self, request, context):
        return LLMResponse(
            text="Hello from Python plugin!",
            tool_calls=[],
            tokens_used=10
        )

# See examples/ for complete implementation
```

### Step 3: Create the Manifest

```yaml
# my-first-plugin.plugin.yaml
plugin:
  name: "my-first-plugin"
  version: "1.0.0"
  author: "Your Name <you@example.com>"
  description: "My first Hector plugin"
  homepage: "https://github.com/yourusername/my-first-plugin"
  license: "MIT"
  
  type: llm_provider
  protocol: grpc
  hector_version: ">=0.1.0"
  
  config_schema:
    required:
      - api_key
    optional:
      - model
      - temperature
    defaults:
      temperature: 0.7
      model: "default-model"
  
  capabilities:
    streaming: true
    function_calling: true
    vision: false
```

### Step 4: Build Your Plugin

**For Go:**
```bash
go mod init my-first-plugin
go mod tidy
go build -o my-first-plugin
chmod +x my-first-plugin
```

**For Python:**
```bash
# Create executable wrapper
cat > my-first-plugin << 'EOF'
#!/usr/bin/env python3
from plugin import serve
serve()
EOF
chmod +x my-first-plugin
```

**For Other Languages:**
See language-specific gRPC documentation for building executable gRPC servers.

### Step 5: Test Your Plugin

Create a test configuration:

```yaml
# test-config.yaml
plugins:
  llm_providers:
    my-first-plugin:
      type: grpc
      path: "./my-first-plugin"
      enabled: true
      config:
        api_key: "test-key"
        model: "test-model"

agents:
  test-agent:
    name: "Test Agent"
    llm: "my-first-plugin"
```

Run Hector:

```bash
hector --config test-config.yaml --agent test-agent
```

---

## Plugin Manifest

The `.plugin.yaml` manifest describes your plugin and its requirements.

### Required Fields

```yaml
plugin:
  name: "plugin-name"              # Unique identifier
  version: "1.0.0"                 # Semantic version
  type: llm_provider               # Plugin type
  protocol: grpc                   # Communication protocol
```

### Optional but Recommended Fields

```yaml
plugin:
  author: "Your Name <you@email.com>"
  description: "What your plugin does"
  homepage: "https://github.com/you/plugin"
  repository_url: "https://github.com/you/plugin"
  documentation_url: "https://docs.example.com"
  license: "MIT"
  
  hector_version: ">=0.1.0"        # Compatible Hector versions
  
  config_schema:
    required: [api_key, endpoint]
    optional: [timeout, retries]
    defaults:
      timeout: 30
      retries: 3
  
  capabilities:
    streaming: true
    function_calling: true
    custom_capability: "value"
```

---

## Configuration

### Plugin Discovery

Configure where Hector looks for plugins:

```yaml
plugin_discovery:
  enabled: true
  paths:
    - "./plugins"
    - "~/.hector/plugins"
    - "/usr/local/hector/plugins"
  scan_subdirectories: true
```

### Plugin Registration

#### Explicit Configuration

```yaml
plugins:
  llm_providers:
    my-custom-llm:
      type: grpc
      path: "./plugins/my-llm"
      enabled: true
      config:
        api_key: "${LLM_API_KEY}"
        endpoint: "https://api.example.com"
        temperature: 0.7
```

#### Auto-Discovery

Place plugin in a discovery path with manifest:

```
~/.hector/plugins/
  └── my-llm
      ├── my-llm                # Executable
      └── my-llm.plugin.yaml    # Manifest
```

Hector will automatically discover and load enabled plugins.

### Using Plugins in Agents

```yaml
agents:
  my-agent:
    name: "My Agent"
    llm: "my-custom-llm"          # Reference plugin by name
    database: "my-custom-db"
    embedder: "my-custom-embedder"
    reasoning_strategy: "my-custom-reasoning"
```

---

## Example Plugins

### 1. Custom LLM Provider (Go)

Location: `examples/plugins/custom-llm-go/`

A simple custom LLM provider that demonstrates:
- Plugin initialization
- Message handling
- Tool calling support
- Streaming responses
- Error handling

### 2. Custom Database Provider (Python)

Location: `examples/plugins/custom-db-python/`

A vector database plugin showing:
- Database connection management
- Vector similarity search
- Collection management
- Metadata handling

### 3. Custom Tool Provider (Go)

Location: `examples/plugins/custom-tools/`

A tool provider plugin with:
- Multiple tool implementations
- Dynamic tool discovery
- Tool execution
- Result formatting

### 4. Custom Reasoning Strategy (Go)

Location: `examples/plugins/custom-reasoning/`

An advanced reasoning plugin demonstrating:
- Custom reasoning logic
- State management
- Context injection
- Stopping conditions

---

## Testing

### Unit Testing

Test your plugin independently:

```go
// plugin_test.go
package main

import (
    "context"
    "testing"
)

func TestPluginInitialize(t *testing.T) {
    plugin := &MyLLMProvider{}
    
    config := map[string]interface{}{
        "api_key": "test-key",
    }
    
    err := plugin.Initialize(context.Background(), config)
    if err != nil {
        t.Fatalf("Initialize failed: %v", err)
    }
}

func TestPluginGenerate(t *testing.T) {
    plugin := &MyLLMProvider{}
    plugin.Initialize(context.Background(), map[string]interface{}{})
    
    response, err := plugin.Generate(context.Background(), nil, nil)
    if err != nil {
        t.Fatalf("Generate failed: %v", err)
    }
    
    if response.Text == "" {
        t.Error("Expected non-empty response")
    }
}
```

### Integration Testing

Test with Hector:

```bash
# Create test configuration
cat > test-config.yaml << EOF
plugins:
  llm_providers:
    test-plugin:
      type: grpc
      path: "./my-plugin"
      enabled: true
      config:
        api_key: "test"

agents:
  test-agent:
    llm: "test-plugin"
EOF

# Run Hector with test config
hector --config test-config.yaml --agent test-agent
```

---

## Best Practices

### 1. Error Handling

Always return meaningful errors:

```go
func (p *MyPlugin) Generate(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error) {
    if len(messages) == 0 {
        return nil, fmt.Errorf("no messages provided")
    }
    
    response, err := p.callAPI(messages)
    if err != nil {
        return nil, fmt.Errorf("API call failed: %w", err)
    }
    
    return response, nil
}
```

### 2. Configuration Validation

Validate configuration in `Initialize()`:

```go
func (p *MyPlugin) Initialize(ctx context.Context, config map[string]interface{}) error {
    apiKey, ok := config["api_key"].(string)
    if !ok || apiKey == "" {
        return fmt.Errorf("api_key is required")
    }
    
    p.apiKey = apiKey
    return nil
}
```

### 3. Resource Management

Clean up in `Shutdown()`:

```go
func (p *MyPlugin) Shutdown(ctx context.Context) error {
    if p.client != nil {
        p.client.Close()
    }
    return nil
}
```

### 4. Context Handling

Respect context cancellation:

```go
func (p *MyPlugin) Generate(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Continue processing
    }
    
    // Use context in API calls
    response, err := p.callAPIWithContext(ctx, messages)
    return response, err
}
```

### 5. Logging

Use structured logging:

```go
import "log/slog"

func (p *MyPlugin) Generate(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error) {
    slog.Info("generating response", "message_count", len(messages), "tool_count", len(tools))
    
    response, err := p.callAPI(messages)
    if err != nil {
        slog.Error("generation failed", "error", err)
        return nil, err
    }
    
    slog.Info("generation complete", "tokens", response.TokensUsed)
    return response, nil
}
```

### 6. Health Checks

Implement meaningful health checks:

```go
func (p *MyPlugin) Health(ctx context.Context) error {
    // Check API connectivity
    if err := p.client.Ping(ctx); err != nil {
        return fmt.Errorf("API unreachable: %w", err)
    }
    
    // Check configuration
    if p.apiKey == "" {
        return fmt.Errorf("plugin not properly initialized")
    }
    
    return nil
}
```

### 7. Versioning

Use semantic versioning and document breaking changes:

```yaml
plugin:
  version: "1.2.0"  # MAJOR.MINOR.PATCH
  hector_version: ">=0.1.0,<2.0.0"  # Compatible versions
```

### 8. Documentation

Document your plugin thoroughly:

```markdown
# My Plugin

## Configuration

- `api_key` (required): Your API key
- `endpoint` (optional): Custom API endpoint
- `timeout` (optional, default: 30): Request timeout in seconds

## Example

\`\`\`yaml
plugins:
  llm_providers:
    my-plugin:
      config:
        api_key: "${API_KEY}"
        timeout: 60
\`\`\`
```

---

## Troubleshooting

### Plugin Not Loading

1. Check plugin is executable: `chmod +x my-plugin`
2. Verify manifest exists: `ls my-plugin.plugin.yaml`
3. Check manifest syntax: `yamllint my-plugin.plugin.yaml`
4. Enable debug mode: `hector --debug`

### Plugin Crashes

1. Check plugin logs
2. Verify configuration is correct
3. Test plugin independently
4. Check resource limits

### Configuration Issues

1. Validate YAML syntax
2. Check environment variables are set
3. Verify required fields are present
4. Review default values

---

## Resources

- [Hector Plugin Architecture](../../PLUGIN_ARCHITECTURE.md)
- [Protocol Buffer Definitions](../../plugins/grpc/protocol/)
- [HashiCorp go-plugin Documentation](https://github.com/hashicorp/go-plugin)
- [gRPC Documentation](https://grpc.io/docs/)

---

## Community

- **Questions**: Open an issue on GitHub
- **Discussions**: Join our Discord server
- **Contributions**: Submit a PR with your plugin example

---

## License

These examples are provided under the MIT License. See LICENSE file for details.

