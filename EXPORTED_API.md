# Hector Public Go API

This document lists exported functions and types intended for use by external Go applications consuming Hector as a library.

**Last Updated**: 2025-11-05

---

## Core Principles

1. **Exported Constructors**: `New*()` functions in public packages are part of the stable API
2. **Registry Pattern**: Most users should use registries (`LLMRegistry`, `ToolRegistry`, etc.) instead of direct constructors
3. **Config-Driven**: Prefer YAML configuration over programmatic setup
4. **Internal Packages**: Code in `pkg/*/internal/` is NOT part of the public API

---

## LLM Providers

### OpenAI
```go
import "github.com/kadirpekel/hector/pkg/llms"

// Public API - Direct provider construction
func NewOpenAIProvider(config llms.LLMConfig) (llms.LLMProvider, error)
```

**Usage**: Direct construction for custom integrations. Most users should use `LLMRegistry.RegisterLLM()` instead.

**Example**:
```go
provider, err := llms.NewOpenAIProvider(llms.LLMConfig{
    Model: "gpt-4",
    APIKey: os.Getenv("OPENAI_API_KEY"),
})
```

---

## Database Providers

### Qdrant
```go
import "github.com/kadirpekel/hector/pkg/databases"

// Public API - Vector database provider
func NewQdrantDatabaseProvider(config config.DatabaseProviderConfig) (databases.DatabaseProvider, error)
```

**Usage**: Create Qdrant vector database client for RAG/semantic search.

**Example**:
```go
db, err := databases.NewQdrantDatabaseProvider(config.DatabaseProviderConfig{
    Type: "qdrant",
    Host: "localhost",
    Port: 6333,
})
```

---

## Embedders

### Ollama Embedder
```go
import "github.com/kadirpekel/hector/pkg/embedders"

// Public API - Local embedding model
func NewOllamaEmbedder(config config.EmbedderProviderConfig) (embedders.EmbedderProvider, error)
```

**Usage**: Create embedder for generating vector embeddings locally via Ollama.

---

## Tools

### Tool Registration
```go
import "github.com/kadirpekel/hector/pkg/tools"

// Public API - Register custom tools
func (r *ToolRegistry) RegisterTool(name string, tool Tool) error
func (r *ToolRegistry) GetTool(name string) (Tool, error)
func (r *ToolRegistry) ListTools() []Tool
```

**Usage**: Register custom tools for agent use.

**Example**:
```go
registry := tools.NewToolRegistry()
registry.RegisterTool("my_tool", &MyCustomTool{})
```

---

## Memory & Context

### Conversation History
```go
import "github.com/kadirpekel/hector/pkg/context"

// Public API - Manage conversation history
func NewConversationHistory(maxTokens int) *ConversationHistory
```

---

## Authentication

### Token Providers
```go
import "github.com/kadirpekel/hector/pkg/auth"

// Public API - Create token providers for custom auth
func NewTokenProviderFromCredentials(credType, token, apiKey, username, password string) (func() (string, error), error)
```

**Usage**: Create token providers for custom authentication schemes.

---

## Plugin Infrastructure (Experimental)

### Plugin Extractors
```go
import "github.com/kadirpekel/hector/pkg/context/extraction"

// Public API - Custom content extractors
func NewPluginExtractor(name, command string, priority int) (*PluginExtractor, error)
```

**Status**: 🚧 Experimental - API may change

**Usage**: Create custom document extractors via external plugins.

---

## Observability

### Metrics & Tracing
```go
import "github.com/kadirpekel/hector/pkg/observability"

// Public API - Access observability components
func (m *Manager) GetTracer() trace.Tracer
func (m *Manager) GetMetrics() *Metrics
```

**Usage**: Access OpenTelemetry tracer and Prometheus metrics for custom instrumentation.

---

## Not Part of Public API

### Internal Implementations
- `pkg/agent/services.go` - Internal service orchestration
- `pkg/transport/*` - Internal transport layer
- `pkg/reasoning/*` - Internal reasoning strategies
- `pkg/cli/*` - CLI-specific code

### Use Config Instead
Rather than calling internal functions, use YAML configuration:

```yaml
# ✅ Recommended: Config-driven
llms:
  gpt4:
    type: openai
    model: gpt-4
    api_key: ${OPENAI_API_KEY}

# ❌ Not recommended: Direct API calls
# Most internal constructors are not designed for external use
```

---

## Stability Guarantees

- ✅ **Stable**: Exported constructors (`New*Provider`, `New*Embedder`, etc.)
- ✅ **Stable**: Registry APIs (`RegisterTool`, `RegisterLLM`, etc.)
- 🚧 **Experimental**: Plugin extractors, custom processors
- ⚠️ **Internal**: Anything in `pkg/*/internal/`, unexported functions

---

## Migration Guide

If you're using Hector programmatically and a function you rely on is not documented here:

1. **Check if it's truly needed** - Most use cases are handled by YAML config
2. **Use registries** - Prefer `XRegistry.Register()` over direct `NewX()`
3. **File an issue** - Request stabilization of the API you need
4. **Fork carefully** - Internal APIs may change between versions

---

## Contributing

To propose additions to the public API:

1. Open an issue describing your use case
2. Explain why YAML configuration is insufficient
3. Propose the function signature and documentation
4. We'll review for inclusion in the stable API

---

**Version**: 1.0.0  
**Compatibility**: Semantic versioning - major version bump for breaking API changes

