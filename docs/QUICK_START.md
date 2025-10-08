---
layout: default
title: Quick Start
nav_order: 2
description: "Get up and running with Hector in 5 minutes"
---

# Hector Quick Start - Pure A2A

## Overview

Hector is now a **pure A2A-native AI agent platform**. Everything communicates via the A2A protocol.

## Prerequisites

1. OpenAI API key (set `OPENAI_API_KEY` env var)
2. Go 1.21+ installed
3. Configuration file

## Quick Test (5 Minutes)

### Step 1: Build the CLI

```bash
cd /Users/kadirpekel/hector
go build -o hector ./cmd/hector
```

### Step 2: Create a Simple Config

Create `test-config.yaml`:

```yaml
global:
  a2a_server:
    enabled: true
    host: "0.0.0.0"
    port: 8080

agents:
  hello:
    name: "Hello Agent"
    description: "A simple test agent"
    llm: "test-llm"
    reasoning:
      engine: "react"
      max_iterations: 5
    prompt:
      system_role: |
        You are a friendly assistant. Keep responses concise.

llms:
  test-llm:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
```

### Step 3: Start the A2A Server

Terminal 1:
```bash
export OPENAI_API_KEY="your-key-here"
./hector serve --config test-config.yaml
```

You should see:
```
🚀 Starting Hector A2A Server...

📋 Registering agents...
  ✅ Hello Agent (hello)

🌐 A2A Server ready!
📡 Agent directory: http://0.0.0.0:8080/agents

💡 Test with Hector CLI:
   hector list
   hector call <agent-id> "your prompt"
```

### Step 4: Test with CLI

Terminal 2:

**List agents:**
```bash
./hector list
```

Expected output:
```
📋 Available agents at http://localhost:8080:

  🤖 Hello Agent
     ID: hello
     A simple test agent
     Capabilities: text_generation, conversation, reasoning
     Endpoint: http://0.0.0.0:8080/agents/hello/tasks
```

**Get agent info (shorthand notation):**
```bash
./hector info hello
```

**Call the agent (shorthand notation):**
```bash
./hector call hello "Say hello to the A2A protocol!"
```

Expected output:
```
🤖 Calling Hello Agent...

Hello! It's great to connect via the A2A protocol! How can I assist you today?

📊 Tokens: 42 | Duration: 1234ms
```

**Interactive chat (shorthand notation):**
```bash
./hector chat hello
```

> **Note**: Hector CLI now supports convenient shorthand notation! 
> - Use just the agent ID: `hector call hello "prompt"`
> - Or full URL: `hector call http://localhost:8080/agents/hello "prompt"`
> - Or custom server: `hector call --server http://localhost:8081 hello "prompt"`

Try:
```
You: What is the A2A protocol?
Bot: The A2A (Agent-to-Agent) protocol is...

You: /quit
```

## Full Example with Tools

Create `advanced-config.yaml`:

```yaml
global:
  a2a_server:
    enabled: true
    host: "0.0.0.0"
    port: 8080

agents:
  coder:
    name: "Code Helper"
    description: "Helps with coding tasks"
    llm: "main-llm"
    tools:
      - write_file
      - search_replace
      - execute_command
    reasoning:
      engine: "react"
      max_iterations: 10
    prompt:
      system_role: |
        You are an expert programmer. You can:
        - Write files
        - Edit files
        - Run commands
        Help users with their coding tasks.

llms:
  main-llm:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7

tools:
  write_file:
    type: file_system
    path: "./workspace"
    allowed_operations: ["write", "read"]
  
  execute_command:
    type: command
    allowed_commands: ["ls", "cat", "grep", "find", "python3"]
    working_directory: "./workspace"
```

Start server:
```bash
./hector serve --config advanced-config.yaml
```

Test with tools:
```bash
./hector call coder "Create a hello.py file that prints 'Hello A2A!'"
```

## Testing A2A Compliance

### Test 1: Agent Discovery

```bash
# Get agent directory
curl http://localhost:8080/agents

# Get specific agent card
curl http://localhost:8080/agents/hello
```

### Test 2: Execute Task (Pure A2A)

```bash
curl -X POST http://localhost:8080/agents/hello/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "taskId": "test-123",
    "input": {
      "type": "text/plain",
      "content": "Hello from pure A2A protocol!"
    }
  }'
```

Expected response:
```json
{
  "taskId": "test-123",
  "status": "completed",
  "output": {
    "type": "text/plain",
    "content": "Hello! Great to hear from you via the A2A protocol!"
  },
  "metadata": {
    "tokens_used": 35,
    "duration_ms": 1123
  },
  "startedAt": "2025-10-05T...",
  "endedAt": "2025-10-05T..."
}
```

### Test 3: External A2A Client

Any A2A-compliant client can now talk to Hector!

```python
# Python example
import requests

# Discover agent
card = requests.get("http://localhost:8080/agents/hello").json()
print(f"Agent: {card['name']}")

# Execute task
task = {
    "taskId": "py-test-1",
    "input": {
        "type": "text/plain",
        "content": "Hello from Python!"
    }
}

response = requests.post(
    "http://localhost:8080/agents/hello/tasks",
    json=task
)

result = response.json()
print(f"Response: {result['output']['content']}")
```

## CLI Commands Reference

### Server Commands

```bash
# Start server
hector serve --config hector.yaml

# Start with debug
hector serve --config hector.yaml --debug
```

### Client Commands

```bash
# List all agents
hector list
hector list --server https://remote-server.com

# Get agent info
hector info http://localhost:8080/agents/hello
hector info hello  # Shorthand (uses default server)

# Execute task
hector call hello "Your prompt here"
hector call http://localhost:8080/agents/hello "Your prompt"
hector call hello "Prompt" --token "bearer-token"

# Interactive chat
hector chat hello
hector chat hello --server https://remote.com
```

## Environment Variables

```bash
# Set default server
export HECTOR_SERVER="http://localhost:8080"

# Set authentication token
export HECTOR_TOKEN="your-bearer-token"

# Now you can omit flags:
hector list
hector call hello "prompt"
```

## Troubleshooting

### Server won't start

**Check config file:**
```bash
cat test-config.yaml
```

**Check port availability:**
```bash
lsof -i :8080
```

### CLI can't connect

**Check server is running:**
```bash
curl http://localhost:8080/agents
```

**Check URL:**
```bash
hector list --server http://localhost:8080
```

### Agent errors

**Check logs:**
Server terminal will show execution details

**Enable debug mode:**
```bash
hector serve --config test-config.yaml --debug
```

## What's Working ✅

1. ✅ **Pure A2A protocol** - 100% compliant
2. ✅ **A2A Server** - Hosts agents via A2A
3. ✅ **A2A Client** - CLI uses A2A to communicate
4. ✅ **Agent discovery** - List and inspect agents
5. ✅ **Task execution** - Call agents via A2A TaskRequest/Response
6. ✅ **Interactive chat** - Multi-turn conversations
7. ✅ **Tool execution** - Agents can use tools
8. ✅ **External clients** - Any A2A client can connect

## What's Next

**Ready for more advanced topics?**

**🚀 [LangChain vs Hector: Multi-Agent Systems](tutorials/MULTI_AGENT_RESEARCH_PIPELINE.md)** - See how Hector transforms complex multi-agent implementations into simple YAML. Perfect next step after this quick start!

**Other topics to explore:**

1. **Streaming** - Server-Sent Events (SSE) streaming per A2A specification
2. **Session management** - Proper A2A session support
3. **Authentication** - Bearer token and API key auth
4. **Multi-agent orchestration** - Using `agent_call` tool
5. **Deployment** - Docker, Kubernetes examples

## Architecture

```
┌─────────────────────────────────────────┐
│  CLI (A2A Client)                       │
│  • list, info, call, chat               │
└────────────┬────────────────────────────┘
             │ A2A Protocol (HTTP/JSON)
┌────────────▼────────────────────────────┐
│  A2A Server                             │
│  • Agent discovery                      │
│  • Task execution                       │
│  • Session management                   │
└────────────┬────────────────────────────┘
             │
┌────────────▼────────────────────────────┐
│  Agents (Pure A2A)                      │
│  • ExecuteTask()                        │
│  • ExecuteTaskStreaming()               │
│  • GetAgentCard()                       │
└─────────────────────────────────────────┘
```

**Everything talks A2A - nothing else!** 🎉

---

## Summary

Hector is now a **pure A2A-native platform**:
- ✅ Server exposes agents via A2A protocol
- ✅ CLI communicates via A2A protocol  
- ✅ Agents implement A2A interface directly
- ✅ External A2A clients can connect
- ✅ 100% protocol compliant

**Ready to test!** Start the server and try the CLI commands above.

