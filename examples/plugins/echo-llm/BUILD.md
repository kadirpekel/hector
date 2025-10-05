# Quick Build & Test Guide

## Build the Plugin

```bash
# From the echo-llm directory
go mod tidy
go build -o echo-llm
chmod +x echo-llm
```

## Test Plugin Directly

```bash
# Start the plugin (it will wait for gRPC connections)
./echo-llm

# You should see:
# 🚀 Starting Echo LLM Plugin...
# (blocks waiting for connections)

# Press Ctrl+C to stop
```

## Test with Hector

### Option 1: Using the Test Config

```bash
# From Hector root directory
cd ../../..

# Run Hector with the test config
hector --config examples/plugins/echo-llm/test-echo.yaml --agent echo-agent

# Try it out:
> Hello!
🔊 Echo: Hello! (call #1)

> What is 2 + 2?
🔊 Echo: What is 2 + 2? (call #2)
```

### Option 2: Custom Config

Create your own `my-config.yaml`:

```yaml
plugins:
  llm_providers:
    echo:
      type: grpc
      path: "./examples/plugins/echo-llm/echo-llm"
      enabled: true
      config:
        prefix: "🤖 Bot says: "

agents:
  my-agent:
    name: "My Agent"
    llm: "echo"
    reasoning:
      max_iterations: 1
```

Run it:

```bash
hector --config my-config.yaml --agent my-agent
```

## Build Your Own Plugin

Use this as a template:

```bash
# Copy the echo plugin
cp -r examples/plugins/echo-llm examples/plugins/my-plugin
cd examples/plugins/my-plugin

# Rename and edit
mv echo-llm.plugin.yaml my-plugin.plugin.yaml
# Edit main.go, manifest, README

# Build
go mod init my-plugin
go mod tidy
go build -o my-plugin
chmod +x my-plugin

# Test
./my-plugin
```

## Troubleshooting

### Plugin doesn't start
```bash
# Check if binary is executable
ls -l echo-llm
# Should show: -rwxr-xr-x

# Make it executable if needed
chmod +x echo-llm

# Try running directly
./echo-llm
```

### Hector can't find plugin
```bash
# Check the path in config
# Paths are relative to where you run hector from

# Use absolute path if needed:
plugins:
  llm_providers:
    echo:
      path: "/full/path/to/echo-llm"
```

### Plugin crashes
```bash
# Check Hector logs
hector --debug --config test-echo.yaml --agent echo-agent

# Check plugin health
# Hector will automatically restart failed plugins
```

## Next Steps

1. **Read the plugin**: Open `main.go` and read the implementation
2. **Modify it**: Change the `prefix` config or echo logic
3. **Rebuild**: Run `go build` again
4. **Test changes**: Run with Hector
5. **Create your own**: Use as template for real plugins

## Resources

- [Plugin Development Guide](../README.md)
- [Plugin Architecture](../../../PLUGIN_ARCHITECTURE.md)
- [gRPC API Reference](../../../plugins/grpc/README.md)

