# Builder Package Reference (`pkg/builder`)

The `github.com/kadirpekel/hector/pkg/builder` package provides a fluent, chainable API for constructing agents, LLMs, tools, and runners programmatically. It is the recommended way to use Hector in Go code.

## LLM Builder

Constructs LLM instances.

```go
llm := builder.NewLLM("openai"). // "openai", "anthropic", "gemini", "ollama"
    Model("gpt-4").
    APIKey(os.Getenv("OPENAI_API_KEY")).
    Temperature(0.7).
    MaxTokens(2000).
    MustBuild()
```

### Methods
-   `NewLLM(provider string)`: Starts building an LLM.
-   `Model(name string)`: Sets the model name.
-   `APIKey(key string)`: Sets the API key.
-   `BaseURL(url string)`: Sets a custom base URL.
-   `Temperature(temp float64)`: Sets sampling temperature.
-   `MaxTokens(n int)`: Sets max tokens to generate.
-   `Build() (model.LLM, error)`: Finalizes and returns the LLM.
-   `MustBuild() model.LLM`: Panics on error.

## Tool Builders

Helpers for creating tools using closures or typed functions.

### Function Tool (Typed)
Creates a tool from a Go function with a strictly typed argument struct. The struct tags determine the JSON schema.

```go
type WeatherArgs struct {
    City string `json:"city" jsonschema:"required,description=The city"`
}

tool, err := builder.FunctionTool(
    "weather",
    "Get weather for a city",
    func(ctx tool.Context, args WeatherArgs) (map[string]any, error) {
        // Implementation
        return map[string]any{"temp": 22}, nil
    },
)
```

-   `FunctionTool`: Returns `(tool.Tool, error)`.
-   `MustFunctionTool`: Panics on error.

## Agent Builder

Constructs intelligent agents.

```go
agent, err := builder.NewAgent("assistant").
    WithName("My Assistant").
    WithDescription("Helpful agent").
    WithLLM(llm).
    WithInstruction("You are helpful.").
    WithTools(tool1, tool2).
    EnableStreaming(true).
    Build()
```

### Methods
-   `NewAgent(id string)`: Starts building an agent.
-   `WithName(name string)`: Sets the human-readable name.
-   `WithDescription(desc string)`: Sets the description.
-   `WithLLM(llm model.LLM)`: Sets the LLM instance.
-   `WithInstruction(inst string)`: Sets the system instruction.
-   `WithTools(tools ...tool.Tool)`: Adds capabilities.
-   `WithSubAgents(agents ...agent.Agent)`: Adds sub-agents (Transfer pattern).
-   `WithTool(t tool.Tool)`: Adds a single tool (useful for `pkg.AgentAsTool`).
-   `WithReasoning(config *config.ReasoningConfig)`: Configures reasoning loop.
-   `EnableStreaming(enable bool)`: Enables streaming responses.
-   `Build() (agent.Agent, error)`: Finalizes the agent.

## Runner Builder

Constructs a runner to manage sessions and execution.

```go
runner, err := builder.NewRunner("my-app").
    WithAgent(agent).
    Build()
```

### Methods
-   `NewRunner(appName string)`: Starts building a runner.
-   `WithAgent(agent agent.Agent)`: Sets the root agent.
-   `WithSessionService(svc session.Service)`: Sets custom session storage.
-   `Build() (*runner.Runner, error)`: Finalizes the runner.

## Reasoning Builder

Configures the agent's thought process loop.

```go
cfg := builder.NewReasoning().
    MaxIterations(10).
    EnableExitTool(true).
    CompletionInstruction("Call exit_loop when done.").
    Build()
```

## RAG Builders

### Embedder Builder
```go
emb := builder.NewEmbedder("openai").
    Model("text-embedding-3-small").
    MustBuild()
```

### Vector Provider Builder
```go
vec := builder.NewVectorProvider("chromem").
    PersistPath("./data").
    MustBuild()
```

### Document Store Builder
```go
store, err := builder.NewDocumentStore("docs").
    FromDirectory("./files").
    WithEmbedder(emb).
    WithVectorProvider(vec).
    Build()
```
