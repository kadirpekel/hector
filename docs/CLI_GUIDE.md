# Hector CLI Guide

## Overview

Hector CLI has a clean, simple architecture:
- **Server Mode** (`hector serve`) - Run as an A2A protocol server
- **Client Mode** (all other commands) - Talk to ANY A2A server

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
# Get detailed agent card
$ hector info http://localhost:8080/agents/competitor_analyst

# Output shows:
# - Agent description
# - Capabilities
# - Endpoints
# - Input/output types
# - Authentication requirements
```

### 4. Execute a Task (One-shot)

```bash
# Using full URL
$ hector call http://localhost:8080/agents/competitor_analyst "Analyze top 5 AI frameworks"

# Using shorthand (with default server)
$ hector call competitor_analyst "Analyze top 5 AI frameworks"

# With authentication
$ hector call competitor_analyst "prompt" --token "your-token"
```

### 5. Interactive Chat

```bash
# Start interactive session
$ hector chat competitor_analyst

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
hector info <agent-url> [options]

Options:
  --token TOKEN    Authentication token

Examples:
  hector info http://localhost:8080/agents/my-agent
  hector info https://external.com/agents/some-agent --token "abc123"
```

### Execute Task

```bash
hector call <agent> "prompt" [options]

Options:
  --server URL     A2A server URL (for shorthand agent names)
  --token TOKEN    Authentication token
  --stream         Enable streaming (coming soon)

Examples:
  # Full URL
  hector call http://localhost:8080/agents/my-agent "Analyze market"

  # Shorthand (uses default server)
  hector call my-agent "Analyze market"

  # With custom server
  hector call my-agent "prompt" --server https://agents.example.com

  # With authentication
  hector call my-agent "prompt" --token "bearer-token"
```

### Interactive Chat

```bash
hector chat <agent> [options]

Options:
  --server URL     A2A server URL (for shorthand agent names)
  --token TOKEN    Authentication token

Examples:
  hector chat my-agent
  hector chat my-agent --server https://agents.example.com
  hector chat my-agent --token "bearer-token"

Interactive commands:
  /quit, /exit - Exit chat
  /clear       - Clear conversation
  /info        - Show agent info
```

## Environment Variables

Set default values to avoid repeating flags:

```bash
# Set default server
export HECTOR_SERVER="http://localhost:8080"

# Set default token
export HECTOR_TOKEN="your-bearer-token"

# Now you can use shorthand
$ hector list              # Uses HECTOR_SERVER
$ hector call my-agent "..."  # Uses HECTOR_SERVER and HECTOR_TOKEN
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
# List your agents
$ hector list

📋 Available agents at http://localhost:8080:

  🤖 Competitor Analysis Agent
     ID: competitor_analyst
     ...

# Test execution
$ hector call competitor_analyst "Analyze AI agent frameworks"

🤖 Calling Competitor Analysis Agent...

Based on current market research:
1. LangChain - Most popular...
2. CrewAI - Multi-agent focused...
...

📊 Tokens: 450 | Duration: 2500ms
```

## Talking to External A2A Servers

The Hector CLI can talk to ANY A2A-compliant server:

```bash
# Talk to Google's agents (example)
$ hector list --server https://google-a2a.example.com

# Execute on external agent
$ hector call https://external.com/agents/some-agent "prompt"

# Interactive chat with external agent
$ hector chat https://external.com/agents/some-agent
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

# Terminal 2: Rapid testing
$ hector call my-agent "test 1"
$ hector call my-agent "test 2"
$ hector call my-agent "test 3"
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

# Test agents
hector call test-agent "test input 1" > result1.txt
hector call test-agent "test input 2" > result2.txt

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
# Store result in variable
RESULT=$(hector call my-agent "analyze this")

# Process result
echo "$RESULT" | grep "important keyword"

# Pipe input
echo "task description" | hector call my-agent "$(cat)"
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

