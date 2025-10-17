---
layout: default
title: Sessions & Streaming
nav_order: 1
parent: Output & Communication
description: "Multi-turn conversations and real-time output"
---

# Sessions & Streaming

Enable multi-turn conversations and real-time streaming output for better user experience.

## Multi-Turn Sessions

Enable conversation context and history:

**Create session:**
```bash
curl -X POST http://localhost:8080/sessions \
  -d '{"agentId": "support_agent"}'
# Response: {"sessionId": "550e8400-..."}
```

**Chat in session:**
```bash
# Message 1
curl -X POST http://localhost:8080/sessions/550e8400-.../tasks \
  -d '{"input":{"type":"text/plain","content":"My name is Alice"}}'

# Message 2 (agent remembers Alice)
curl -X POST http://localhost:8080/sessions/550e8400-.../tasks \
  -d '{"input":{"type":"text/plain","content":"What is my name?"}}'
# Response: "Your name is Alice"
```

**Benefits:**
- Conversation history maintained
- Context across multiple turns
- Personalized responses
- Follow-up questions work naturally

## Real-Time Streaming

Get token-by-token output via Server-Sent Events (SSE) per A2A specification:

```bash
# Using curl
curl -N -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"message":{"role":"user","parts":[{"type":"text","text":"Explain quantum computing"}]}}' \
  http://localhost:8080/agents/my_agent/message/stream

# Output (SSE format):
# event: message
# data: {"task_id":"task-1","message":{"role":"assistant","parts":[{"type":"text","text":"Quantum"}]}}
#
# event: message
# data: {"task_id":"task-1","message":{"role":"assistant","parts":[{"type":"text","text":" computing"}]}}
#
# event: status
# data: {"task_id":"task-1","status":{"state":"completed"}}
```

**Using Hector CLI:**
```bash
# Streaming is enabled by default in chat and --stream flag in call
hector chat my_agent         # Interactive with streaming
hector call my_agent "prompt" --stream  # Single call with streaming
```

**Benefits:**
- Immediate feedback to users
- Better UX for long responses
- Cancel long-running tasks
- Progress indicators
- A2A-compliant

## Configuration

```yaml
reasoning:
  enable_streaming: true  # Enable streaming output
```

## Session Management

### Session Lifecycle

1. **Create Session**: Initialize with agent ID
2. **Send Messages**: Multiple turns in same session
3. **Maintain Context**: Agent remembers conversation history
4. **End Session**: Explicitly close or timeout

### Session Storage

Sessions are stored in configured session store:
- **Memory**: Default, in-process storage
- **SQL**: Persistent database storage
- **Redis**: Distributed session storage

## Best Practices

1. **Use Sessions for Conversations**: Enable context across multiple turns
2. **Enable Streaming**: Better user experience for long responses
3. **Manage Session Lifecycle**: Clean up old sessions
4. **Handle Disconnections**: Graceful handling of client disconnects

## See Also

- **[Structured Output](structured-output)** - Reliable response formats
- **[Memory Management](../memory-context)** - How agents remember context
- **[API Reference](../../reference/API_REFERENCE)** - Complete API documentation
