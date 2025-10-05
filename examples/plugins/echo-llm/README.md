# Echo LLM Plugin - Reference Implementation

This is a simple echo plugin that demonstrates how to create a Hector plugin. It's intended as a reference implementation and starting point for building your own plugins.

## What It Does

The echo plugin:
- Echoes back whatever the user says
- Supports both streaming and non-streaming responses
- Shows basic plugin structure and lifecycle
- Demonstrates configuration handling
- Includes health checks and proper shutdown

## Building

```bash
# From this directory (examples/plugins/echo-llm/)
go mod tidy
go build -o echo-llm
chmod +x echo-llm
```

This creates an executable `echo-llm` in the current directory.

## Testing the Plugin Directly

You can test that the plugin starts correctly:

```bash
# The plugin will wait for gRPC connections
./echo-llm

# You should see:
# 🚀 Starting Echo LLM Plugin...
# (then it blocks waiting for connections)

# Press Ctrl+C to stop
```

## Using with Hector

### Option 1: Explicit Configuration

Create a configuration file `test-echo.yaml`:

```yaml
# LLM configuration
llms:
  # Built-in OpenAI provider (commented out)
  # main-llm:
  #   type: openai
  #   model: gpt-4
  #   api_key: ${OPENAI_API_KEY}

# Plugin configuration
plugins:
  llm_providers:
    echo-llm:
      type: grpc
      path: "./examples/plugins/echo-llm/echo-llm"
      enabled: true
      config:
        prefix: "🔊 Echo says: "
        max_tokens: 1000
        temperature: 0.7

# Agent configuration
agents:
  echo-agent:
    name: "Echo Agent"
    description: "An agent that echoes your messages"
    llm: "echo-llm"  # Use the echo plugin
    tools:
      - type: local
        name: execute_command
        enabled: false
    reasoning:
      max_iterations: 1

# Workflow configuration (optional)
workflows: {}
```

Run Hector with this configuration:

```bash
# From Hector root directory
hector --config examples/plugins/echo-llm/test-echo.yaml --agent echo-agent
```

### Option 2: Auto-Discovery

1. Place the built plugin in a discovery directory:

```bash
mkdir -p ~/.hector/plugins
cp echo-llm ~/.hector/plugins/
cp echo-llm.plugin.yaml ~/.hector/plugins/
```

2. Configure Hector to discover plugins:

```yaml
# In your hector.yaml
plugin_discovery:
  enabled: true
  paths:
    - "~/.hector/plugins"
  scan_subdirectories: true

agents:
  my-agent:
    llm: "echo-llm"  # Automatically discovered!
```

## Example Interaction

```bash
$ hector --config test-echo.yaml --agent echo-agent

> Hello, how are you?
🔊 Echo says: Hello, how are you? (call #1)

> What can you do?
🔊 Echo says: What can you do? (call #2)

> /quit
```

## Code Structure

```go
type EchoLLMProvider struct {
    // Plugin state
}

// Required methods:
func (e *EchoLLMProvider) Initialize(ctx, config) error
func (e *EchoLLMProvider) Generate(ctx, messages, tools) (*GenerateResponse, error)
func (e *EchoLLMProvider) GenerateStreaming(ctx, messages, tools) (<-chan *StreamChunk, error)
func (e *EchoLLMProvider) GetModelInfo(ctx) (*ModelInfo, error)
func (e *EchoLLMProvider) Shutdown(ctx) error
func (e *EchoLLMProvider) Health(ctx) error

func main() {
    grpc.ServeLLMPlugin(&EchoLLMProvider{})
}
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `prefix` | string | "Echo: " | Text to prepend to echoed messages |
| `max_tokens` | int | 1000 | Maximum tokens to generate |
| `temperature` | float | 0.7 | Temperature setting |

## Extending This Plugin

This plugin serves as a template. To create your own plugin:

1. **Copy this directory** to your own project
2. **Rename the plugin** (update names in code and manifest)
3. **Implement the LLM methods** with your actual logic:
   - Replace echo logic with real LLM API calls
   - Handle authentication (use config)
   - Implement proper error handling
   - Add retry logic for network issues
4. **Update the manifest** with your plugin details
5. **Build and test** independently
6. **Deploy** to users

### Example: OpenAI Plugin

```go
type OpenAIPlugin struct {
    client *openai.Client
    model  string
}

func (o *OpenAIPlugin) Initialize(ctx context.Context, config map[string]string) error {
    apiKey := config["api_key"]
    o.model = config["model"]
    o.client = openai.NewClient(apiKey)
    return nil
}

func (o *OpenAIPlugin) Generate(ctx context.Context, messages []*grpc.Message, tools []*grpc.ToolDefinition) (*grpc.GenerateResponse, error) {
    // Convert messages to OpenAI format
    // Call OpenAI API
    // Return response
}
```

## Debugging

If the plugin doesn't work:

1. **Check the executable exists and is executable:**
   ```bash
   ls -l echo-llm
   # Should show: -rwxr-xr-x ... echo-llm
   ```

2. **Check the manifest is in the same directory:**
   ```bash
   ls -l echo-llm.plugin.yaml
   ```

3. **Test the plugin directly:**
   ```bash
   ./echo-llm
   # Should start without errors
   ```

4. **Check Hector logs:**
   ```bash
   hector --debug --config test-echo.yaml --agent echo-agent
   ```

5. **Verify configuration:**
   - Plugin path is correct
   - Plugin is enabled
   - Agent references the correct plugin name

## Learn More

- [Plugin Development Guide](../README.md) - Complete plugin development documentation
- [Plugin Architecture](../../../PLUGIN_ARCHITECTURE.md) - System architecture and design
- [gRPC Plugin API](../../../plugins/grpc/README.md) - gRPC-specific API documentation

## License

MIT - Same as Hector

