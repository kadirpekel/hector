---
title: Sessions
description: Manage agent sessions for context continuity
---

# Sessions

Sessions provide context continuity across multiple interactions, enabling agents to remember previous conversations and maintain state.

## What are Sessions?

A session represents a continuous conversation between a user and an agent. Sessions enable:

- **Context preservation** - Agent remembers previous messages
- **Memory persistence** - Long-term memory scoped to sessions
- **Conversation tracking** - Monitor ongoing interactions
- **Multi-turn conversations** - Natural back-and-forth dialogue
- **Persistent storage** - Conversations survive server restarts (with session stores)

---

## Session Lifecycle

```
1. Session Created
   ├─ Unique session ID generated
   ├─ Memory initialized
   └─ Context store created

2. Conversations Happen
   ├─ Messages exchanged
   ├─ Working memory updated
   └─ Long-term memories stored

3. Session Ends
   ├─ Final memories stored
   ├─ Session marked complete
   └─ Resources cleaned up
```

---

## Using Sessions

### REST API

**Start a session:**

```bash
curl -X POST http://localhost:8080/agents/assistant/sessions \
  -H "Content-Type: application/json"
```

Response:
```json
{
  "session_id": "sess_abc123"
}
```

**Send message in session:**

```bash
curl -X POST http://localhost:8080/agents/assistant/sessions/sess_abc123/messages \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hello, remember me?"
  }'
```

**Continue conversation:**

```bash
curl -X POST http://localhost:8080/agents/assistant/sessions/sess_abc123/messages \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What did we talk about earlier?"
  }'
```

Agent remembers previous context!

### gRPC API

```protobuf
// Create session
rpc CreateSession(CreateSessionRequest) returns (Session)

// Send message
rpc SendMessage(SendMessageRequest) returns (MessageResponse)

// Stream messages
rpc StreamMessage(SendMessageRequest) returns (stream MessageChunk)
```

### CLI

```bash
# Interactive chat (automatic session)
hector chat --agent assistant --config config.yaml

# Specify session ID for resumption
hector chat --agent assistant --config config.yaml --session my-session

# Single call with session
hector call "Hello" --agent assistant --config config.yaml --session my-session

# Resume later (same session ID = same conversation)
hector call "Continue where we left off" --agent assistant --config config.yaml --session my-session
```

See [CLI Reference](../reference/cli.md#session-support) for full details.

---

## Session Configuration

### Basic (In-Memory)

```yaml
agents:
  assistant:
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000

      longterm:

        storage_scope: "session"  # Session-scoped long-term memory
```

### With Persistent Storage

For conversations that survive server restarts:

```yaml
# Global session stores (like llms, databases, tools)
session_stores:
  main-db:
    backend: sql
    sql:
      driver: sqlite  # or postgres, mysql
      database: ./data/sessions.db

agents:
  assistant:
    session_store: "main-db"  # Reference global store
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000
```

**Storage scopes:**

- `session` - Memories per session (most common)
- `conversational` - Memories across all user sessions
- `all` - Global memory across all users
- `summaries_only` - Only summarized content

**Session persistence:** See [Setup Session Persistence](../how-to/setup-session-persistence.md) guide.

---

## Session Management

### List Sessions

```bash
GET /agents/{agent}/sessions
```

### Get Session Info

```bash
GET /agents/{agent}/sessions/{session_id}
```

### Delete Session

```bash
DELETE /agents/{agent}/sessions/{session_id}
```

---

## Use Cases

### Chat Applications

```yaml
agents:
  chatbot:
    memory:
      working:
        strategy: "buffer_window"
        window_size: 20
      longterm:

        storage_scope: "session"
```

### Customer Support

```yaml
agents:
  support:
    tools: ["search", "agent_call"]
    memory:
      longterm:

        storage_scope: "conversational"  # Remember across sessions
```

### Code Assistants

```yaml
agents:
  coder:
    tools: ["write_file", "execute_command", "search"]
    memory:
      working:
        strategy: "summary_buffer"
      longterm:
        storage_scope: "session"
```

---

## Advanced Configuration

### Session Timeout

```yaml
# Coming soon
sessions:
  timeout: "30m"       # Session expires after 30 minutes of inactivity
  max_duration: "24h"  # Maximum session duration
```

---

## Monitoring & Debugging

Enable debug logging:

```yaml
agents:
  debug:
    reasoning:
```

Output shows:
```
[Session: sess_abc123]
[Message: 1]
[Memory: 523 tokens]
[Response: ...]
```

---

## API Reference

### REST Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/agents/{agent}/sessions` | POST | Create session |
| `/agents/{agent}/sessions` | GET | List sessions |
| `/agents/{agent}/sessions/{id}` | GET | Get session info |
| `/agents/{agent}/sessions/{id}` | DELETE | Delete session |
| `/agents/{agent}/sessions/{id}/messages` | POST | Send message |
| `/agents/{agent}/sessions/{id}/messages/stream` | POST | Stream message (SSE) |

### gRPC Methods

```protobuf
service A2AService {
  rpc CreateSession(CreateSessionRequest) returns (Session);
  rpc SendMessage(SendMessageRequest) returns (MessageResponse);
  rpc StreamMessage(SendMessageRequest) returns (stream MessageChunk);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
  rpc DeleteSession(DeleteSessionRequest) returns (DeleteSessionResponse);
}
```

See [API Reference](../reference/api.md) for full details.

---

## Best Practices

### Session-Scoped Memory

```yaml
# ✅ Good: Session-scoped memory
agents:
  support:
    memory:
      longterm:
        storage_scope: "session"

# ⚠️ Caution: Global memory (memory grows indefinitely)
agents:
  risky:
    memory:
      longterm:
        storage_scope: "all"
```

### Error Handling

Always handle session errors gracefully:

```javascript
// ✅ Good: Handle session errors
try {
  const response = await createSession();
  const messages = await sendMessage(response.session_id, "Hello");
} catch (error) {
  console.error('Session error:', error);
  // Fallback logic
}
```

---

## Next Steps

- **[Setup Session Persistence](../how-to/setup-session-persistence.md)** - Configure persistent session storage
- **[Streaming](streaming.md)** - Real-time response delivery
- **[Memory](memory.md)** - Session-scoped memory configuration
- **[API Reference](../reference/api.md)** - Complete API documentation
- **[Security](security.md)** - Session authentication

---

## Related Topics

- **[Agent Overview](overview.md)** - Understanding agents
- **[Configuration Reference](../reference/configuration.md)** - All session options
- **[Architecture](../reference/architecture.md)** - How sessions work internally
