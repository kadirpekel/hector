---
title: Sessions & Streaming
description: Manage agent sessions and real-time streaming responses
---

# Sessions & Streaming

Sessions provide context continuity across multiple interactions, while streaming delivers responses in real-time as the agent generates them.

## Sessions

### What are Sessions?

A session represents a continuous conversation between a user and an agent. Sessions enable:

- **Context preservation** - Agent remembers previous messages
- **Memory persistence** - Long-term memory scoped to sessions
- **Conversation tracking** - Monitor ongoing interactions
- **Multi-turn conversations** - Natural back-and-forth dialogue
- **Persistent storage** - Conversations survive server restarts (with session stores)

### Session Lifecycle

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

### Using Sessions

#### REST API

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

#### gRPC API

```protobuf
// Create session
rpc CreateSession(CreateSessionRequest) returns (Session)

// Send message
rpc SendMessage(SendMessageRequest) returns (MessageResponse)

// Stream messages
rpc StreamMessage(SendMessageRequest) returns (stream MessageChunk)
```

#### CLI

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

### Session Configuration

#### Basic (In-Memory)

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

#### With Persistent Storage

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

### Session Management

#### List Sessions

```bash
GET /agents/{agent}/sessions
```

#### Get Session Info

```bash
GET /agents/{agent}/sessions/{session_id}
```

#### Delete Session

```bash
DELETE /agents/{agent}/sessions/{session_id}
```

---

## Streaming

### What is Streaming?

Streaming delivers agent responses token-by-token as they're generated, instead of waiting for the complete response.

**Without streaming:**
```
[User waits...]
[User waits...]
[Complete response arrives]
```

**With streaming:**
```
The capital
The capital of
The capital of France
The capital of France is
The capital of France is Paris.
```

### Benefits

- **Real-time feedback** - Users see progress immediately
- **Better UX** - Feels more interactive
- **Early cancellation** - Stop if going wrong direction
- **Perceived speed** - Feels faster even if same total time

### Enabling Streaming

#### Configuration

```yaml
agents:
  assistant:
    reasoning:
      enable_streaming: true  # Enable streaming
```

#### REST API (SSE - Server-Sent Events)

```bash
curl -N http://localhost:8080/agents/assistant/messages/stream \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Explain quantum computing"
  }'
```

Response (SSE format):
```
data: {"chunk": "Quantum"}
data: {"chunk": " computing"}
data: {"chunk": " uses"}
data: {"chunk": " quantum"}
data: {"chunk": " mechanics..."}
data: [DONE]
```

#### WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8080/agents/assistant/stream');

ws.send(JSON.stringify({
  message: "Explain quantum computing"
}));

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.chunk) {
    process.stdout.write(data.chunk);
  }
};
```

#### gRPC Streaming

```go
stream, err := client.StreamMessage(ctx, &pb.SendMessageRequest{
    Agent: "assistant",
    Message: "Explain quantum computing",
})

for {
    chunk, err := stream.Recv()
    if err == io.EOF {
        break
    }
    fmt.Print(chunk.Content)
}
```

#### CLI (Automatic)

```bash
# Streaming enabled by default in CLI
hector call "Explain quantum computing" --agent assistant --config config.yaml
# Response streams as it's generated
```

### Streaming with Tools

When agents use tools, streaming shows progress:

```yaml
agents:
  coder:
    reasoning:
      enable_streaming: true
      show_tool_execution: true
    tools: ["write_file", "execute_command"]
```

**Streamed output:**

```
Let me create that file...
[Tool: write_file("hello.py", "print('Hello')")]
File created successfully.
Now let me test it...
[Tool: execute_command("python hello.py")]
Output: Hello
The program works correctly!
```

---

## Sessions + Streaming

Combine both for best experience:

```yaml
agents:
  assistant:
    reasoning:
      enable_streaming: true
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000
      longterm:
        
        storage_scope: "session"
```

**REST API:**

```bash
# Create session
SESSION_ID=$(curl -X POST http://localhost:8080/agents/assistant/sessions | jq -r '.session_id')

# Stream messages in session
curl -N http://localhost:8080/agents/assistant/sessions/$SESSION_ID/messages/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello"}'
```

Agent streams responses and maintains session context!

---

## Advanced Configuration

### Session Timeout

```yaml
# Coming soon
sessions:
  timeout: "30m"       # Session expires after 30 minutes of inactivity
  max_duration: "24h"  # Maximum session duration
```

### Streaming Options

```yaml
agents:
  assistant:
    reasoning:
      enable_streaming: true
      show_tool_execution: true  # Show tool calls in stream
      show_thinking: false       # Show internal reasoning
      show_debug_info: false     # Show debug details
```

### Streaming Customization

```yaml
agents:
  custom:
    streaming:
      chunk_size: 10      # Characters per chunk
      delay_ms: 50        # Delay between chunks
      buffer_size: 1024   # Buffer size
```

---

## Use Cases

### Chat Applications

```yaml
agents:
  chatbot:
    reasoning:
      enable_streaming: true
    memory:
      working:
        strategy: "buffer_window"
        window_size: 20
      longterm:
        
        storage_scope: "session"
```

**Frontend:**

```javascript
const eventSource = new EventSource(
  `http://localhost:8080/agents/chatbot/sessions/${sessionId}/messages/stream`
);

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  appendToChat(data.chunk);
};
```

### Customer Support

```yaml
agents:
  support:
    reasoning:
      enable_streaming: true
      show_tool_execution: true
    tools: ["search", "agent_call"]
    memory:
      longterm:
        
        storage_scope: "conversational"  # Remember across sessions
```

### Code Assistants

```yaml
agents:
  coder:
    reasoning:
      enable_streaming: true
      show_tool_execution: true
      show_thinking: true  # Show reasoning process
    tools: ["write_file", "execute_command", "search"]
```

---

## Monitoring & Debugging

### Session Tracking

Enable debug logging:

```yaml
agents:
  debug:
    reasoning:
      show_debug_info: true
```

Output shows:
```
[Session: sess_abc123]
[Message: 1]
[Memory: 523 tokens]
[Response: ...]
```

### Streaming Debug

Monitor stream chunks:

```bash
curl -N http://localhost:8080/agents/assistant/messages/stream?debug=true \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello"}'
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
| `/agents/{agent}/stream` | WS | WebSocket streaming |

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

### Session Management

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

### Streaming Performance

```yaml
# ✅ Good: Streaming with reasonable chunk size
agents:
  fast:
    reasoning:
      enable_streaming: true

# ❌ Bad: Streaming disabled for interactive apps
agents:
  slow:
    reasoning:
      enable_streaming: false  # Users wait for complete response
```

### Error Handling

```javascript
// ✅ Good: Handle stream errors
const eventSource = new EventSource(url);

eventSource.onerror = (error) => {
  console.error('Stream error:', error);
  eventSource.close();
};

// ❌ Bad: No error handling
const eventSource = new EventSource(url);
eventSource.onmessage = (event) => { /* ... */ };
```

---

## Next Steps

- **[Setup Session Persistence](../how-to/setup-session-persistence.md)** - Configure persistent session storage
- **[API Reference](../reference/api.md)** - Complete API documentation
- **[Memory](memory.md)** - Session-scoped memory configuration
- **[Security](security.md)** - Session authentication
- **[Architecture](../reference/architecture.md)** - How sessions work internally

---

## Related Topics

- **[Agent Overview](overview.md)** - Understanding agents
- **[Configuration Reference](../reference/configuration.md)** - All session options
- **[Build a Chat Application](../how-to/build-coding-assistant.md)** - Complete tutorial

