# Hector CLI Guide

## Overview

Hector CLI has a clean, simple architecture:
- **Server Mode** (`hector serve`) - Run as an A2A protocol server
- **Client Mode** (all other commands) - Talk to ANY A2A server

### Agent Specification Formats

Hector CLI supports **two convenient ways** to specify agents:

1. **Shorthand Notation** (Recommended for local/frequent use)
   - Format: `agent_id`
   - Example: `hector call my_agent "prompt"`
   - Uses default server (`localhost:8080`) or `HECTOR_SERVER` environment variable
   - Can override with `--server` flag

2. **Full URL** (For external/specific agents)
   - Format: `http://host:port/agents/agent_id`
   - Example: `hector call http://example.com/agents/my_agent "prompt"`
   - Uses URL as-is, ignores `--server` flag and environment variables

**This makes Hector CLI ergonomic for daily use while maintaining full flexibility.**

## Quick Start

### 1. Start a Server

```bash
# Start local A2A server
$ hector serve

# Or with custom config
$ hector serve --config my-agents.yaml

# With debug output
$ hector serve --config hector.yaml --debug
```

### 2. List Available Agents

```bash
# List agents from default server (localhost:8080)
$ hector list

# List agents from specific server
$ hector list --server https://agents.example.com

# Output:
📋 Available agents at http://localhost:8080:

  🤖 Competitor Analysis Agent
     ID: competitor_analyst
     Analyzes market competitors and provides insights
     Capabilities: text_generation, conversation, reasoning
     Endpoint: http://localhost:8080/agents/competitor_analyst/tasks
```

### 3. Get Agent Information

```bash
# Using shorthand (recommended)
$ hector info competitor_analyst

# Using full URL
$ hector info http://localhost:8080/agents/competitor_analyst

# With custom server
$ hector info --server http://localhost:8081 competitor_analyst

# Output shows:
# - Agent description
# - Capabilities
# - Endpoints
# - Input/output types
# - Authentication requirements
```

### 4. Execute a Task (One-shot)

```bash
# Using shorthand (recommended for local)
$ hector call competitor_analyst "Analyze top 5 AI frameworks"

# Using shorthand with custom server
$ hector call --server http://localhost:8081 competitor_analyst "Analyze..."

# Using full URL (for external agents)
$ hector call http://example.com/agents/competitor_analyst "Analyze..."

# With authentication
$ hector call competitor_analyst "prompt" --token "your-token"

# Disable streaming (enabled by default)
$ hector call competitor_analyst "prompt" --stream=false
```

### 5. Interactive Chat

```bash
# Start interactive session (shorthand)
$ hector chat competitor_analyst

# With custom server
$ hector chat --server http://localhost:8081 competitor_analyst

# With full URL
$ hector chat http://example.com/agents/competitor_analyst

💬 Chat with Competitor Analysis Agent
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Type your messages below. Commands:
  /quit or /exit - Exit chat
  /clear - Clear conversation history
  /info - Show agent information

> Analyze LangChain
[Agent responds...]

> What about CrewAI?
[Agent responds...]

> /quit
Goodbye!
```

## Command Reference

### Server Mode

```bash
hector serve [options]

Options:
  --config FILE    Configuration file (default: hector.yaml)
  --debug          Enable debug output

Examples:
  hector serve
  hector serve --config configs/production.yaml
  hector serve --config hector.yaml --debug
```

### List Agents

```bash
hector list [options]

Options:
  --server URL     A2A server URL
  --token TOKEN    Authentication token

Examples:
  hector list
  hector list --server http://localhost:8080
  hector list --server https://agents.company.com --token "abc123"
```

### Get Agent Info

```bash
hector info <agent> [options]

Options:
  --server URL     A2A server URL (for shorthand agent names)
  --token TOKEN    Authentication token

Examples:
  # Shorthand (recommended)
  hector info my_agent
  
  # Shorthand with custom server
  hector info --server http://localhost:8081 my_agent
  
  # Full URL
  hector info http://localhost:8080/agents/my_agent
  
  # External agent with auth
  hector info https://external.com/agents/some_agent --token "abc123"
```

### Execute Task

```bash
hector call <agent> "prompt" [options]

Options:
  --server URL     A2A server URL (for shorthand agent names)
  --token TOKEN    Authentication token
  --stream BOOL    Enable streaming (default: true, use --stream=false to disable)

Examples:
  # Shorthand (recommended for local, uses localhost:8080)
  hector call my_agent "Analyze market trends"

  # Shorthand with custom server
  hector call --server http://localhost:8081 my_agent "Analyze market"

  # Shorthand with environment variable
  export HECTOR_SERVER="https://prod.example.com"
  hector call my_agent "Analyze market"

  # Full URL (ignores --server and env vars)
  hector call http://localhost:8080/agents/my_agent "Analyze market"

  # With authentication
  hector call my_agent "prompt" --token "bearer-token"
  
  # Disable streaming (enabled by default)
  hector call my_agent "prompt" --stream=false
```

### Interactive Chat

```bash
hector chat <agent> [options]

Options:
  --server URL     A2A server URL (for shorthand agent names)
  --token TOKEN    Authentication token

Examples:
  # Shorthand (recommended for local)
  hector chat my_agent
  
  # Shorthand with custom server
  hector chat --server http://localhost:8081 my_agent
  
  # Shorthand with environment variable
  export HECTOR_SERVER="https://prod.example.com"
  hector chat my_agent
  
  # Full URL
  hector chat http://localhost:8080/agents/my_agent
  
  # With authentication
  hector chat my_agent --token "bearer-token"

Interactive commands:
  /quit, /exit - Exit chat
  /clear       - Clear conversation history
  /info        - Show agent information
  
Note: Streaming is always enabled in chat mode for better UX
```

## Environment Variables

Set default values to avoid repeating flags:

```bash
# Set default server (used for shorthand notation)
export HECTOR_SERVER="http://localhost:8080"

# Set default token
export HECTOR_TOKEN="your-bearer-token"

# Now you can use shorthand everywhere
$ hector list                    # Uses HECTOR_SERVER
$ hector call my_agent "..."      # Uses HECTOR_SERVER
$ hector chat my_agent            # Uses HECTOR_SERVER
$ hector info my_agent            # Uses HECTOR_SERVER
```

**How it works:**
- When you use **shorthand** (just agent ID): `my_agent`
  - CLI checks for `--server` flag
  - If not provided, checks `HECTOR_SERVER` environment variable
  - If not set, defaults to `http://localhost:8080`
  
- When you use **full URL**: `http://example.com/agents/my_agent`
  - Uses URL as-is
  - Ignores `--server` flag and `HECTOR_SERVER` variable

**Best Practice:**
```bash
# In development
export HECTOR_SERVER="http://localhost:8080"

# In production
export HECTOR_SERVER="https://prod-agents.company.com"
export HECTOR_TOKEN="your-prod-token"

# Now all commands adapt to your environment
hector call my_agent "prompt"  # Automatically uses right server
```

## Testing Your Own Server

### Terminal 1: Start Server

```bash
$ cd my-project
$ hector serve --config hector.yaml

🚀 Starting Hector A2A Server...
📋 Registering agents...
  ✅ Competitor Analysis Agent (competitor_analyst)
  ✅ Customer Support Agent (customer_support)

🌐 A2A Server ready!
📡 Agent directory: http://localhost:8080/agents

💡 Test with Hector CLI:
   hector list
   hector call <agent-id> "your prompt"

Press Ctrl+C to stop
```

### Terminal 2: Test with Client

```bash
# List your agents (shorthand - uses localhost:8080 by default)
$ hector list

📋 Available agents at http://localhost:8080:

  🤖 Competitor Analysis Agent
     ID: competitor_analyst
     ...

# Test execution with shorthand notation
$ hector call competitor_analyst "Analyze AI agent frameworks"

🤖 Calling Competitor Analysis Agent...

Based on current market research:
1. LangChain - Most popular...
2. CrewAI - Multi-agent focused...
...

📊 Tokens: 450 | Duration: 2500ms

# Interactive chat (also shorthand)
$ hector chat competitor_analyst
```

## Talking to External A2A Servers

The Hector CLI can talk to ANY A2A-compliant server using both methods:

```bash
# Method 1: Use --server flag with shorthand
$ hector list --server https://external-a2a.example.com
$ hector call --server https://external-a2a.example.com some_agent "prompt"
$ hector chat --server https://external-a2a.example.com some_agent

# Method 2: Use full URL directly
$ hector call https://external.com/agents/some_agent "prompt"
$ hector chat https://external.com/agents/some_agent

# Method 3: Set environment variable for convenience
$ export HECTOR_SERVER="https://external-a2a.example.com"
$ hector list                    # Automatically uses external server
$ hector call some_agent "prompt" # Automatically uses external server
$ hector chat some_agent          # Automatically uses external server
```

## Configuration File for Defaults

Create `~/.hector/config.yaml`:

```yaml
default_server: "http://localhost:8080"
auth:
  type: "bearer"
  token: "${HECTOR_TOKEN}"
```

Then use shorthand everywhere:

```bash
$ hector list                    # Uses default_server
$ hector call my-agent "prompt"  # Uses default_server + auth
```

## Use Cases

### Local Development

```bash
# Terminal 1: Your server
$ hector serve

# Terminal 2: Rapid testing with shorthand (so convenient!)
$ hector call my_agent "test 1"
$ hector call my_agent "test 2"
$ hector call my_agent "test 3"

# Or interactive chat
$ hector chat my_agent
```

### CI/CD Testing

```bash
#!/bin/bash
# test-agents.sh

# Start server in background
hector serve --config test-config.yaml &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Test agents (shorthand notation - so clean!)
hector call test_agent "test input 1" > result1.txt
hector call test_agent "test input 2" > result2.txt

# Verify results
if grep -q "expected output" result1.txt; then
  echo "✅ Test 1 passed"
else
  echo "❌ Test 1 failed"
  kill $SERVER_PID
  exit 1
fi

# Clean up
kill $SERVER_PID
```

### Production Monitoring

```bash
# Check if production agents are healthy
hector list --server https://prod-agents.company.com --token "$PROD_TOKEN"

# Test critical agent
hector call https://prod-agents.company.com/agents/critical-agent \
  "health check" \
  --token "$PROD_TOKEN"
```

### Multi-Environment Workflow

```bash
# Development
export HECTOR_SERVER="http://localhost:8080"
hector call my-agent "test"

# Staging
export HECTOR_SERVER="https://staging-agents.company.com"
export HECTOR_TOKEN="staging-token"
hector call my-agent "test"

# Production
export HECTOR_SERVER="https://prod-agents.company.com"
export HECTOR_TOKEN="prod-token"
hector call my-agent "test"
```

## Troubleshooting

### Connection Issues

```bash
# Check if server is running
$ curl http://localhost:8080/agents

# Test with full URL
$ hector info http://localhost:8080/agents/my-agent

# Enable debug (when implemented)
$ HECTOR_DEBUG=1 hector call my-agent "test"
```

### Authentication Issues

```bash
# Verify token
$ echo $HECTOR_TOKEN

# Try with explicit token
$ hector call my-agent "test" --token "your-token"

# Check agent card for auth requirements
$ hector info http://localhost:8080/agents/my-agent
```

### Agent Not Found

```bash
# List all agents to find correct ID
$ hector list

# Use exact agent ID from list
$ hector call exact-agent-id "prompt"
```

## Tips & Tricks

### 1. Alias for Convenience

```bash
# Add to ~/.bashrc or ~/.zshrc
alias hc='hector call'
alias hl='hector list'
alias hi='hector info'
alias hchat='hector chat'

# Now use shortcuts
$ hl
$ hc my-agent "prompt"
$ hchat my-agent
```

### 2. Scripting with Hector

```bash
# Store result in variable (shorthand notation)
RESULT=$(hector call my_agent "analyze this")

# Process result
echo "$RESULT" | grep "important keyword"

# Loop over multiple inputs
for input in "task1" "task2" "task3"; do
  hector call my_agent "$input"
done

# With different servers
RESULT_DEV=$(hector call --server http://localhost:8080 my_agent "test")
RESULT_PROD=$(hector call --server https://prod.example.com my_agent "test")
```

### 3. Testing Multiple Servers

```bash
# Create env files
# .env.dev
HECTOR_SERVER=http://localhost:8080

# .env.staging
HECTOR_SERVER=https://staging.example.com
HECTOR_TOKEN=staging-token

# Load and test
$ source .env.dev && hector list
$ source .env.staging && hector list
```

## Next Steps

- Read [A2A Protocol Guide](A2A_GUIDE.md) for detailed protocol information
- Check [Configuration Guide](CONFIGURATION.md) for server setup
- See [Examples](configs/) for pre-configured agents

## Support

- GitHub Issues: https://github.com/kadirpekel/hector/issues
- Documentation: https://docs.hector.ai
- Community: https://discord.gg/hector (coming soon)

