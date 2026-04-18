# Go API Reference

Complete reference for Hector's programmatic Go API. Use this when building agents programmatically instead of via YAML configuration.

**Package:** `github.com/verikod/hector/pkg`

---

## Quick Start

```go
import (
    "context"
    pkg "github.com/verikod/hector/pkg"
    "github.com/verikod/hector/pkg/model/openai"
)

func main() {
    // Config-first: Load from YAML
    h, _ := pkg.FromConfig("config.yaml")
    result, _ := h.Generate(context.Background(), "Hello!")
    
    // Programmatic: Build with options
    h, _ := pkg.New(
        pkg.WithOpenAI(openai.Config{APIKey: "sk-..."}),
        pkg.WithInstruction("You are a helpful assistant."),
    )
    result, _ := h.Generate(context.Background(), "Hello!")
}
```

---

## Core Types

### Type Aliases

Convenience aliases so users don't need to import multiple packages:

| Alias | Original |
|-------|----------|
| `Agent` | `agent.Agent` |
| `Tool` | `tool.Tool` |
| `CallableTool` | `tool.CallableTool` |
| `Toolset` | `tool.Toolset` |
| `LLM` | `model.LLM` |
| `AgentConfig` | `config.AgentConfig` |
| `ToolConfig` | `config.ToolConfig` |
| `LLMAgentConfig` | `llmagent.Config` |

---

### Hector

Main entry point for the Hector platform.

```go
type Hector struct {
    // ... internal fields
}
```

**Methods:**

| Method | Description |
|--------|-------------|
| `Generate(ctx, input) (string, error)` | Single-turn generation |
| `GenerateStream(ctx, input) iter.Seq2[*Event, error]` | Streaming generation |
| `Run(ctx, input) iter.Seq2[*Event, error]` | Execute with event stream |
| `RunWithSession(ctx, userID, sessionID, input) iter.Seq2[*Event, error]` | Execute with session |
| `Serve(addr) error` | Start A2A server |
| `Close() error` | Release resources |
| `Runtime() *runtime.Runtime` | Get underlying runtime |
| `Config() *config.AppConfig` | Get configuration |
| `Agent(name) (Agent, bool)` | Get agent by name |
| `DefaultAgent() (Agent, bool)` | Get default agent |
| `SessionService() session.Service` | Get session service |

---

## Creating Hector Instances

### FromConfig

Create from YAML configuration file.

```go
func FromConfig(path string) (*Hector, error)
func FromConfigWithContext(ctx context.Context, path string) (*Hector, error)
```

**Example:**

```go
h, err := pkg.FromConfig("config.yaml")
if err != nil {
    log.Fatal(err)
}
defer h.Close()
```

---

### New

Create programmatically with options.

```go
func New(opts ...Option) (*Hector, error)
```

**Example:**

```go
h, err := pkg.New(
    pkg.WithAnthropic(anthropic.Config{
        APIKey: os.Getenv("ANTHROPIC_API_KEY"),
        Model:  "claude-sonnet-4",
    }),
    pkg.WithInstruction("You are a helpful assistant."),
    pkg.WithMCPCommand("filesystem", "npx", "-y", "@modelcontextprotocol/server-filesystem", "./data"),
)
```

---

## Agent Creation

### NewAgent

Create an LLM agent programmatically.

```go
func NewAgent(cfg llmagent.Config) (Agent, error)
```

**Example:**

```go
model, _ := openai.New(openai.Config{APIKey: key})

researcher, _ := pkg.NewAgent(pkg.LLMAgentConfig{
    Name:        "researcher",
    Description: "Researches topics thoroughly",
    Model:       model,
    Tools:       []tool.Tool{searchTool},
    Instruction: "You are a research assistant.",
})
```

---

### Workflow Agents

#### NewSequentialAgent

Runs sub-agents once, in fixed order.

```go
func NewSequentialAgent(cfg SequentialConfig) (Agent, error)
```

**Example:**

```go
pipeline, _ := pkg.NewSequentialAgent(pkg.SequentialConfig{
    Name:        "pipeline",
    Description: "Data processing pipeline",
    SubAgents:   []pkg.Agent{extractor, transformer, loader},
})
```

---

#### NewParallelAgent

Runs sub-agents simultaneously.

```go
func NewParallelAgent(cfg ParallelConfig) (Agent, error)
```

**Example:**

```go
voters, _ := pkg.NewParallelAgent(pkg.ParallelConfig{
    Name:        "voters",
    Description: "Gets multiple perspectives",
    SubAgents:   []pkg.Agent{voter1, voter2, voter3},
})
```

---

#### NewLoopAgent

Runs sub-agents repeatedly until completion.

```go
func NewLoopAgent(cfg LoopConfig) (Agent, error)
```

**Example:**

```go
refiner, _ := pkg.NewLoopAgent(pkg.LoopConfig{
    Name:          "refiner",
    Description:   "Iteratively refines output",
    SubAgents:     []pkg.Agent{reviewer, improver},
    MaxIterations: 3,
})
```

---

### NewRemoteAgent

Create a remote A2A agent.

```go
func NewRemoteAgent(cfg RemoteAgentConfig) (Agent, error)
```

**Example:**

```go
remoteHelper, _ := pkg.NewRemoteAgent(pkg.RemoteAgentConfig{
    Name: "remote_helper",
    URL:  "http://other-server:8080/agents/helper",
})

h, _ := pkg.New(
    pkg.WithOpenAI(openai.Config{APIKey: key}),
    pkg.WithSubAgents(remoteHelper),
)
```

---

## Agent Navigation

### FindAgent

Search for agent by name in tree.

```go
func FindAgent(root Agent, name string) Agent
```

---

### FindAgentPath

Get path to agent in tree.

```go
func FindAgentPath(root Agent, name string) []string
```

---

### WalkAgents

Visit all agents depth-first.

```go
func WalkAgents(root Agent, visitor func(Agent, int) bool)
```

**Example:**

```go
pkg.WalkAgents(root, func(ag pkg.Agent, depth int) bool {
    fmt.Printf("%s%s\n", strings.Repeat("  ", depth), ag.Name())
    return true // continue walking
})
```

---

### ListAgents

Get flat list of all agents.

```go
func ListAgents(root Agent) []Agent
```

---

## Multi-Agent Patterns

### AgentAsTool (Pattern 2: Delegation)

Convert agent to callable tool. Parent maintains control.

```go
func AgentAsTool(ag Agent) Tool
func AgentAsToolWithConfig(ag Agent, cfg *agenttool.Config) Tool
```

**Example:**

```go
searchTool := pkg.AgentAsTool(searchAgent)

parent, _ := pkg.New(
    pkg.WithOpenAI(openai.Config{APIKey: key}),
    pkg.WithTool(searchTool),
)
```

---

### Control Tools

```go
func ExitLoopTool() Tool      // Explicit loop termination
func EscalateTool() Tool      // Escalate to parent
func TransferTool(agentName, description string) Tool  // Transfer to another agent
```

---

## LLM Options

### WithAnthropic

```go
func WithAnthropic(cfg anthropic.Config) Option
```

**Example:**

```go
pkg.WithAnthropic(anthropic.Config{
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
    Model:  "claude-sonnet-4",
})
```

---

### WithOpenAI

```go
func WithOpenAI(cfg openai.Config) Option
```

**Example:**

```go
pkg.WithOpenAI(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4o",
})
```

---

### WithGemini

```go
func WithGemini(cfg gemini.Config) Option
```

**Example:**

```go
pkg.WithGemini(gemini.Config{
    APIKey: os.Getenv("GEMINI_API_KEY"),
    Model:  "gemini-2.0-flash",
})
```

---

### WithOllama

```go
func WithOllama(cfg ollama.Config) Option
```

**Example:**

```go
pkg.WithOllama(ollama.Config{
    Model:   "llama3.2",
    BaseURL: "http://localhost:11434",
})
```

---

### WithLLM

Provide custom LLM instance directly.

```go
func WithLLM(llm model.LLM) Option
```

---

### WithLLMConfig

Add LLM from config struct.

```go
func WithLLMConfig(name string, cfg *config.LLMConfig) Option
```

---

## Tool Options

### WithMCPTool (SSE)

Add MCP toolset via Server-Sent Events transport.

```go
func WithMCPTool(name, url string, filter ...string) Option
```

**Example:**

```go
pkg.WithMCPTool("composio", "https://mcp.composio.dev/sse", "GMAIL_SEND_EMAIL")
```

---

### WithMCPToolHTTP (Streamable HTTP)

Add MCP toolset via HTTP transport.

```go
func WithMCPToolHTTP(name, url string, filter ...string) Option
```

---

### WithMCPCommand (stdio)

Add MCP toolset via stdio transport.

```go
func WithMCPCommand(name, command string, args ...string) Option
```

**Example:**

```go
pkg.WithMCPCommand("filesystem", "npx", "-y", "@modelcontextprotocol/server-filesystem", "./data")
```

---

### WithToolset

Add custom toolset.

```go
func WithToolset(ts tool.Toolset) Option
```

---

### WithTool / WithTools

Add tools directly.

```go
func WithTool(t tool.Tool) Option
func WithTools(tools ...tool.Tool) Option
```

---

### WithToolConfig

Add tool from config struct.

```go
func WithToolConfig(name string, cfg *config.ToolConfig) Option
```

---

## Agent Options

### WithInstruction

Set system instruction for default agent.

```go
func WithInstruction(instruction string) Option
```

---

### WithAgentName

Set default agent name.

```go
func WithAgentName(name string) Option
```

---

### WithAgent

Add custom agent configuration.

```go
func WithAgent(name string, cfg *config.AgentConfig) Option
```

---

### WithReasoning

Configure chain-of-thought reasoning loop.

```go
func WithReasoning(cfg *config.ReasoningConfig) Option
```

**Example:**

```go
pkg.WithReasoning(&config.ReasoningConfig{
    MaxIterations:      50,
    EnableExitTool:     true,
    EnableEscalateTool: true,
})
```

---

### WithControlTools

Enable control flow tools.

```go
func WithControlTools(enableExit, enableEscalate bool) Option
```

---

### WithStreaming

Enable token streaming.

```go
func WithStreaming(enabled bool) Option
```

---

## Multi-Agent Options

### WithSubAgents (Pattern 1: Transfer)

Add sub-agents with automatic transfer tools.

```go
func WithSubAgents(agents ...agent.Agent) Option
```

**Example:**

```go
h, _ := pkg.New(
    pkg.WithOpenAI(openai.Config{APIKey: key}),
    pkg.WithSubAgents(researcher, writer),
)
// Creates transfer_to_researcher and transfer_to_writer tools
```

---

### WithAgentTool / WithAgentTools (Pattern 2: Delegation)

Add agents as callable tools.

```go
func WithAgentTool(ag agent.Agent) Option
func WithAgentTools(agents ...agent.Agent) Option
```

**Example:**

```go
h, _ := pkg.New(
    pkg.WithOpenAI(openai.Config{APIKey: key}),
    pkg.WithAgentTools(searchAgent, analysisAgent, writerAgent),
)
```

---

## Session Options

### WithSessionService

Set custom session service.

```go
func WithSessionService(s session.Service) Option
```

---

## Execution Methods

### Generate

Single-turn generation.

```go
func (h *Hector) Generate(ctx context.Context, input string) (string, error)
```

**Example:**

```go
result, err := h.Generate(ctx, "What is the weather today?")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result)
```

---

### GenerateStream

Streaming generation.

```go
func (h *Hector) GenerateStream(ctx context.Context, input string) iter.Seq2[*agent.Event, error]
```

---

### Run

Execute with event stream.

```go
func (h *Hector) Run(ctx context.Context, input string) iter.Seq2[*agent.Event, error]
```

**Example:**

```go
for event, err := range h.Run(ctx, "Research quantum computing") {
    if err != nil {
        log.Fatal(err)
    }
    switch e := event.Data.(type) {
    case *agent.TextDelta:
        fmt.Print(e.Text)
    case *agent.ToolCall:
        fmt.Printf("\nCalling tool: %s\n", e.Name)
    }
}
```

---

### RunWithSession

Execute with specific session.

```go
func (h *Hector) RunWithSession(ctx context.Context, userID, sessionID, input string) iter.Seq2[*agent.Event, error]
```

---

### Serve

Start A2A HTTP server.

```go
func (h *Hector) Serve(addr string) error
```

**Example:**

```go
h, _ := pkg.New(
    pkg.WithOpenAI(openai.Config{APIKey: key}),
)

// Start server on port 8080
if err := h.Serve(":8080"); err != nil {
    log.Fatal(err)
}
```

---

## Complete Examples

### Multi-Agent Research Pipeline

```go
package main

import (
    "context"
    "fmt"
    "os"

    pkg "github.com/verikod/hector/pkg"
    "github.com/verikod/hector/pkg/model/anthropic"
)

func main() {
    ctx := context.Background()
    
    // Create LLM
    model, _ := anthropic.New(anthropic.Config{
        APIKey: os.Getenv("ANTHROPIC_API_KEY"),
        Model:  "claude-sonnet-4",
    })
    
    // Create specialized agents
    researcher, _ := pkg.NewAgent(pkg.LLMAgentConfig{
        Name:        "researcher",
        Description: "Researches topics",
        Model:       model,
        Instruction: "You are a thorough researcher.",
    })
    
    writer, _ := pkg.NewAgent(pkg.LLMAgentConfig{
        Name:        "writer",
        Description: "Writes articles",
        Model:       model,
        Instruction: "You are a skilled writer.",
    })
    
    // Create coordinator with sub-agents
    h, _ := pkg.New(
        pkg.WithAnthropic(anthropic.Config{
            APIKey: os.Getenv("ANTHROPIC_API_KEY"),
            Model:  "claude-sonnet-4",
        }),
        pkg.WithInstruction("You coordinate research and writing."),
        pkg.WithSubAgents(researcher, writer),
    )
    defer h.Close()
    
    // Run
    result, _ := h.Generate(ctx, "Write an article about AI agents")
    fmt.Println(result)
}
```

---

### Agent with MCP Tools

```go
h, _ := pkg.New(
    pkg.WithOpenAI(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o",
    }),
    pkg.WithInstruction("You are a helpful assistant with file access."),
    pkg.WithMCPCommand("filesystem", "npx", "-y", 
        "@modelcontextprotocol/server-filesystem", "./data"),
)
defer h.Close()

result, _ := h.Generate(ctx, "List files in the data directory")
```

---

### Custom Tool Implementation

```go
import (
    "github.com/verikod/hector/pkg/tool"
    "github.com/verikod/hector/pkg/tool/functiontool"
)

type SearchArgs struct {
    Query string `json:"query" jsonschema:"required,description=Search query"`
}

searchTool, _ := functiontool.New("web_search", "Search the web",
    func(ctx tool.Context, args SearchArgs) (map[string]any, error) {
        // Perform search
        results := performSearch(args.Query)
        return map[string]any{"results": results}, nil
    },
)

h, _ := pkg.New(
    pkg.WithOpenAI(openai.Config{APIKey: key}),
    pkg.WithTool(searchTool),
)
```

---

## Custom Server (Bootstrap API)

The `pkg/bootstrap` package simplifies creating custom Hector server binaries. It handles configuration loading, observability setup, and lifecycle management while allowing you to inject custom runtimes or modify server behavior.

### Serve

Main entry point to start the server.

```go
func Serve(ctx context.Context, opts ...ServeOption) error
```

### Options

#### WithServerConfig

Set operational configuration (port, DB, auth).

```go
func WithServerConfig(cfg *config.ServerConfig) ServeOption
```

#### WithConfigPath

Set path to the application YAML config.

```go
func WithConfigPath(path string) ServeOption
```

#### WithRuntimeFactory

Inject a custom runtime factory to replace standard agent execution logic.

```go
type RuntimeFactory func(ctx context.Context, deps BootstrapDependencies, appID string, appCfg *config.AppConfig) (server.Runtime, error)

func WithRuntimeFactory(f RuntimeFactory) ServeOption
```

### Example: Custom Server

```go
package main

import (
    "context"
    "log"

    "github.com/verikod/hector/pkg/bootstrap"
    "github.com/verikod/hector/pkg/config"
)

func main() {
    ctx := context.Background()

    // programmatic server config
    srvCfg := &config.ServerConfig{
        Port: 9090,
        Auth: &config.AuthConfig{Secret: "my-secret"},
    }

    err := bootstrap.Serve(ctx,
        bootstrap.WithServerConfig(srvCfg),
        bootstrap.WithConfigPath(".hector/config.yaml"),
        bootstrap.WithWatch(true),
    )
    if err != nil {
        log.Fatal(err)
    }
}
```
