# Hector Go Package

The `pkg` directory contains the full Hector runtime, both the config-first CLI entrypoint and the programmatic Go API.

## Quick Start (Go API)

The simplest entry point is `pkg.FromConfig` for config-driven usage, or `pkg.New` for programmatic construction:

```go
import "github.com/verikod/hector/pkg"

// Config-first: load from YAML
h, err := pkg.FromConfig("config.yaml")
if err != nil {
    log.Fatal(err)
}
defer h.Close()
h.Serve(":8080")
```

```go
// Programmatic: build with fluent API
h, err := pkg.New(
    pkg.WithOpenAI(openai.Config{APIKey: os.Getenv("OPENAI_API_KEY")}),
    pkg.WithMCPTool("weather", "http://localhost:9000"),
    pkg.WithInstruction("You are a helpful assistant."),
)
if err != nil {
    log.Fatal(err)
}
defer h.Close()

response, err := h.Generate(ctx, "What's the weather in Berlin?")
```

See [`pkg/api.go`](api.go) for the full programmatic API and [`pkg/examples/`](examples/) for runnable examples.

## Architecture

### Context Hierarchy

```
InvocationContext (full mutable access)
    └── CallbackContext  (state modification during tool calls)
            └── ReadonlyContext  (safe read-only access for tools)
```

### Event Flow

```
User Message
    ↓
runner.Run()
    ↓
agent.Run() → yields Event stream
    ↓
session.AppendEvent()
    ↓
A2A protocol translation
    ↓
Client Response (streaming or batch)
```

## Contributing

When adding new features:
1. Define interfaces first in the relevant sub-package
2. Implement concrete types with tests alongside
3. Wire factory registration in `runtime/factories.go` if config-driven

## License

MIT. See [LICENSE](../LICENSE).