# Hector Plugin Architecture

## Overview

Hector supports dynamic plugin discovery and loading, allowing users to extend the system with custom implementations of:
- **LLM Providers** - Custom language model integrations
- **Database Providers** - Custom vector database backends
- **Embedder Providers** - Custom embedding generation
- **Tool Providers** - Custom tool implementations
- **Reasoning Strategies** - Custom reasoning approaches

## Plugin Protocol: gRPC Only

Hector uses **gRPC exclusively** for plugin communication, providing the best balance of flexibility, performance, and production-readiness.

**Architecture:**
```
┌─────────────────┐         gRPC          ┌──────────────────┐
│  Hector Core    │◄─────────────────────►│  Plugin Process  │
│                 │      (Protocol         │  (Any Language)  │
│  - Discovery    │       Buffers)         │                  │
│  - Registry     │                        │  - LLM Provider  │
│  - Management   │                        │  - Custom Logic  │
└─────────────────┘                        └──────────────────┘
```

**Why gRPC Only?**

✅ **Language-Agnostic:** Write plugins in Python, Rust, Go, JavaScript, or any gRPC-supported language

✅ **Process Isolation:** Plugin crashes don't affect Hector; clean resource management

✅ **Production-Ready:** Battle-tested framework used by Terraform, Vault, Packer, and other major tools

✅ **Version-Independent:** No Go version matching required (unlike native Go plugins)

✅ **Cross-Platform:** Works seamlessly on Windows, macOS, and Linux

✅ **Industry Standard:** Using HashiCorp's go-plugin framework with extensive community support

**Implementation:** Using HashiCorp go-plugin framework

---

## Configuration Schema

### Plugin Discovery Configuration

```yaml
# hector.yaml

# Plugin discovery paths
plugin_discovery:
  enabled: true
  paths:
    - "./plugins"
    - "~/.hector/plugins"
    - "/usr/local/hector/plugins"
  scan_subdirectories: true
  
# Individual plugin definitions
plugins:
  # LLM Provider Plugin
  llm_providers:
    my-custom-llm:
      type: grpc                    # Always "grpc"
      path: "./plugins/my-llm"      # executable path
      enabled: true
      config:
        api_key: "${CUSTOM_LLM_KEY}"
        model: "custom-v1"
        temperature: 0.7
      
    another-llm:
      type: grpc
      path: "./plugins/another-llm"
      enabled: true
      config:
        endpoint: "http://localhost:8080"
  
  # Database Provider Plugin
  database_providers:
    my-vector-db:
      type: grpc
      path: "./plugins/my-vector-db"
      enabled: true
      config:
        connection: "${VECTOR_DB_URL}"
        api_key: "${VECTOR_DB_KEY}"
  
  # Embedder Provider Plugin
  embedder_providers:
    custom-embedder:
      type: grpc
      path: "./plugins/custom-embedder"
      enabled: true
      config:
        model: "custom-embedding-v1"
        dimension: 1536
  
  # Tool Provider Plugin
  tool_providers:
    advanced-tools:
      type: grpc
      path: "./plugins/advanced-tools"
      enabled: true
      config:
        tool_config: "custom-settings"
  
  # Reasoning Strategy Plugin
  reasoning_strategies:
    advanced-reasoning:
      type: grpc
      path: "./plugins/advanced-reasoning"
      enabled: true
      config:
        strategy_params: {}

# Use plugins in agent configuration
agents:
  main-agent:
    name: "Main Agent"
    description: "Agent using custom plugins"
    llm: "my-custom-llm"           # Reference plugin
    database: "my-vector-db"        # Reference plugin
    embedder: "custom-embedder"     # Reference plugin
    reasoning_strategy: "advanced-reasoning"
    tools:
      - type: plugin
        provider: "advanced-tools"  # Reference plugin provider
```

---

## Plugin Interface Definitions

### LLM Provider Plugin

```go
type LLMProviderPlugin interface {
    // Generate generates a response with native function calling support
    Generate(ctx context.Context, messages []Message, tools []ToolDefinition) (*LLMResponse, error)
    
    // GenerateStreaming generates a streaming response
    GenerateStreaming(ctx context.Context, messages []Message, tools []ToolDefinition) (StreamReader, error)
    
    // GetModelName returns the model name
    GetModelName() string
    
    // GetMaxTokens returns the maximum tokens for generation
    GetMaxTokens() int
    
    // GetTemperature returns the temperature setting
    GetTemperature() float64
    
    // Initialize initializes the plugin with configuration
    Initialize(config map[string]interface{}) error
    
    // Shutdown cleanly shuts down the plugin
    Shutdown() error
}
```

### Database Provider Plugin

```go
type DatabaseProviderPlugin interface {
    // Upsert adds or updates a document
    Upsert(ctx context.Context, collection string, id string, vector []float32, metadata map[string]interface{}) error
    
    // Search performs vector similarity search
    Search(ctx context.Context, collection string, vector []float32, topK int) ([]*SearchResult, error)
    
    // Delete removes a document
    Delete(ctx context.Context, collection string, id string) error
    
    // CreateCollection creates a new collection
    CreateCollection(ctx context.Context, collection string, vectorSize uint64) error
    
    // DeleteCollection removes a collection
    DeleteCollection(ctx context.Context, collection string) error
    
    // Initialize initializes the plugin with configuration
    Initialize(config map[string]interface{}) error
    
    // Shutdown cleanly shuts down the plugin
    Shutdown() error
}
```

### Embedder Provider Plugin

```go
type EmbedderProviderPlugin interface {
    // Embed generates embeddings for text
    Embed(ctx context.Context, text string) ([]float32, error)
    
    // GetDimension returns the embedding dimension
    GetDimension() int
    
    // GetModelName returns the model name
    GetModelName() string
    
    // Initialize initializes the plugin with configuration
    Initialize(config map[string]interface{}) error
    
    // Shutdown cleanly shuts down the plugin
    Shutdown() error
}
```

### Tool Provider Plugin

```go
type ToolProviderPlugin interface {
    // DiscoverTools discovers available tools
    DiscoverTools(ctx context.Context) error
    
    // ListTools returns available tools
    ListTools() []*ToolInfo
    
    // GetTool retrieves a specific tool
    GetTool(name string) (Tool, bool)
    
    // ExecuteTool executes a tool
    ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error)
    
    // Initialize initializes the plugin with configuration
    Initialize(config map[string]interface{}) error
    
    // Shutdown cleanly shuts down the plugin
    Shutdown() error
}
```

### Reasoning Strategy Plugin

```go
type ReasoningStrategyPlugin interface {
    // PrepareIteration prepares for a reasoning iteration
    PrepareIteration(iteration int, state *ReasoningState) error
    
    // ShouldStop determines if reasoning should stop
    ShouldStop(text string, toolCalls []*ToolCall, state *ReasoningState) bool
    
    // AfterIteration runs after an iteration completes
    AfterIteration(iteration int, text string, toolCalls []*ToolCall, results []*ToolResult, state *ReasoningState) error
    
    // GetContextInjection returns context to inject
    GetContextInjection(state *ReasoningState) string
    
    // GetPromptSlots returns prompt configuration
    GetPromptSlots() *PromptSlots
    
    // GetName returns strategy name
    GetName() string
    
    // GetDescription returns strategy description
    GetDescription() string
    
    // Initialize initializes the plugin with configuration
    Initialize(config map[string]interface{}) error
    
    // Shutdown cleanly shuts down the plugin
    Shutdown() error
}
```

---

## Plugin Discovery Flow

```
1. Hector Startup
   │
   ├─► Load Configuration (hector.yaml)
   │   └─► Parse plugin_discovery and plugins sections
   │
   ├─► Plugin Discovery Phase
   │   ├─► Scan configured paths for plugin executables
   │   ├─► Read plugin manifests (.plugin.yaml)
   │   └─► Validate plugin compatibility
   │
   ├─► Plugin Loading Phase
   │   ├─► Initialize plugin processes (gRPC)
   │   ├─► Establish communication channels
   │   └─► Call Initialize() with config
   │
   ├─► Plugin Registration Phase
   │   ├─► Register LLM providers
   │   ├─► Register Database providers
   │   ├─► Register Embedder providers
   │   ├─► Register Tool providers
   │   └─► Register Reasoning strategies
   │
   └─► Component Manager Ready
       └─► Agents can use plugin-provided components
```

---

## Plugin Manifest Format

Each plugin executable should be accompanied by a `.plugin.yaml` manifest:

```yaml
# my-llm.plugin.yaml

plugin:
  name: "my-custom-llm"
  version: "1.0.0"
  author: "Your Name"
  description: "Custom LLM provider for proprietary model"
  
  # Plugin type
  type: llm_provider
  
  # Communication protocol (always grpc)
  protocol: grpc
  
  # Compatibility
  hector_version: ">=0.1.0"
  
  # Configuration schema (for validation)
  config_schema:
    required:
      - api_key
      - model
    optional:
      - temperature
      - max_tokens
      - endpoint
  
  # Capabilities
  capabilities:
    streaming: true
    function_calling: true
    vision: false
```

---

## Directory Structure

```
hector/
├── plugins/                    # Plugin infrastructure
│   ├── discovery.go           # Plugin discovery logic
│   ├── registry.go            # Plugin registry
│   ├── types.go               # Common plugin types
│   └── grpc/                  # gRPC plugin implementation
│       ├── proto/             # Protocol buffer definitions
│       │   ├── common.proto
│       │   ├── llm.proto
│       │   ├── database.proto
│       │   └── embedder.proto
│       ├── loader.go          # gRPC plugin loader
│       ├── interfaces.go      # Plugin interfaces
│       ├── adapters.go        # Plugin adapters
│       └── plugin_impl.go     # gRPC client/server implementation
│
├── examples/                   # Example plugins
│   ├── plugins/
│   │   ├── custom-llm/        # Example LLM plugin (Go)
│   │   ├── custom-db/         # Example Database plugin (Python)
│   │   ├── custom-embedder/   # Example Embedder plugin (Rust)
│   │   └── custom-reasoning/  # Example Reasoning strategy plugin
│   └── README.md              # Plugin development guide
│
└── config/
    └── plugin.go              # Plugin configuration types
```

---

## Plugin Development Workflow

### 1. Create Plugin Implementation

**Go Plugin Example:**
```go
// plugins/examples/custom-llm/main.go
package main

import (
    "context"
    "github.com/hashicorp/go-plugin"
    "github.com/kadirpekel/hector/plugins/grpc"
)

type CustomLLMProvider struct {
    config map[string]interface{}
}

func (p *CustomLLMProvider) Initialize(config map[string]interface{}) error {
    p.config = config
    // Initialize your LLM client
    return nil
}

func (p *CustomLLMProvider) Generate(ctx context.Context, messages []grpc.Message, tools []grpc.ToolDefinition) (*grpc.LLMResponse, error) {
    // Your implementation
    return &grpc.LLMResponse{
        Text: "Generated response",
        ToolCalls: nil,
        TokensUsed: 100,
    }, nil
}

// ... implement other methods

func main() {
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: grpc.HandshakeConfig,
        Plugins: map[string]plugin.Plugin{
            "llm_provider": &grpc.LLMProviderPlugin{Impl: &CustomLLMProvider{}},
        },
        GRPCServer: plugin.DefaultGRPCServer,
    })
}
```

### 2. Build Plugin

```bash
go build -o my-llm ./plugins/examples/custom-llm
```

### 3. Create Manifest

```yaml
# my-llm.plugin.yaml
plugin:
  name: "my-custom-llm"
  version: "1.0.0"
  type: llm_provider
  protocol: grpc
```

### 4. Configure in Hector

```yaml
# hector.yaml
plugins:
  llm_providers:
    my-custom-llm:
      type: grpc
      path: "./plugins/my-llm"
      config:
        api_key: "${MY_API_KEY}"
```

### 5. Use in Agent

```yaml
agents:
  my-agent:
    llm: "my-custom-llm"  # References plugin
```

---

## Error Handling

### Plugin Load Failures

```go
// Plugins that fail to load are logged but don't crash Hector
// Built-in providers are always available as fallback

if err := pluginRegistry.LoadPlugin("custom-llm"); err != nil {
    log.Warnf("Failed to load plugin 'custom-llm': %v", err)
    log.Warnf("Falling back to built-in LLM providers")
}
```

### Plugin Runtime Errors

```go
// Plugin crashes are isolated due to process separation
// Hector automatically attempts to restart crashed plugins

if err := llm.Generate(...); err != nil {
    if errors.Is(err, plugin.ErrPluginCrashed) {
        log.Errorf("Plugin crashed, attempting restart...")
        pluginRegistry.RestartPlugin("custom-llm")
    }
}
```

---

## Security Considerations

1. **Plugin Verification:**
   - Optional plugin signature verification
   - Checksum validation
   - Trusted plugin directories

2. **Sandboxing:**
   - Process isolation (gRPC plugins)
   - Resource limits (CPU, memory, file access)
   - Network restrictions

3. **Configuration:**
   ```yaml
   plugin_security:
     verify_signatures: true
     trusted_directories:
       - "/usr/local/hector/plugins"
     max_memory_mb: 512
     max_cpu_percent: 50
   ```

---

## Migration Path

### Phase 1: Infrastructure
- ✅ Plugin discovery system
- ✅ gRPC protocol definitions
- ✅ Plugin registry integration

### Phase 2: Core Providers
- ✅ LLM provider plugins
- ✅ Database provider plugins
- ✅ Embedder provider plugins
- ✅ Tool provider plugins
- ✅ Reasoning strategy plugins

### Phase 3: Ecosystem
- ✅ Example plugins
- ✅ Plugin development SDK
- ✅ Plugin marketplace/registry

---

## Best Practices for Plugin Authors

1. **Always implement graceful shutdown**
2. **Validate configuration in Initialize()**
3. **Use context for cancellation**
4. **Return meaningful errors**
5. **Log using structured logging**
6. **Version your plugin manifests**
7. **Document configuration schema**
8. **Provide example configurations**
9. **Test plugin with Hector integration tests**
10. **Handle network/API failures gracefully**

---

## References

- **HashiCorp go-plugin:** https://github.com/hashicorp/go-plugin
- **gRPC Go:** https://grpc.io/docs/languages/go/
- **Protocol Buffers:** https://protobuf.dev/

---

**Last Updated:** October 4, 2025

