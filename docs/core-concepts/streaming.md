---
title: Streaming
description: Real-time response delivery for better user experience
---

# Streaming

Streaming delivers agent responses token-by-token as they're generated, instead of waiting for the complete response.

## What is Streaming?

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

---

## Benefits

- **Real-time feedback** - Users see progress immediately
- **Better UX** - Feels more interactive
- **Early cancellation** - Stop if going wrong direction
- **Perceived speed** - Feels faster even if same total time

---

## Enabling Streaming

### Configuration

```yaml
agents:
  assistant:
    reasoning:
      enable_streaming: true  # Enable streaming
```

### REST API (SSE - Server-Sent Events)

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

### WebSocket

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

### gRPC Streaming

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

### CLI (Automatic)

```bash
# Streaming enabled by default in CLI
hector call "Explain quantum computing" --agent assistant --config config.yaml
# Response streams as it's generated
```

---

## Streaming with Tools

When agents use tools, streaming shows progress:

```yaml
agents:
  coder:
    reasoning:
      enable_streaming: true
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

## Streaming with Sessions

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

### Streaming Options

```yaml
agents:
  assistant:
    reasoning:
      enable_streaming: true
      show_thinking: false       # Show internal reasoning
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
    tools: ["search", "agent_call"]
```

### Code Assistants

```yaml
agents:
  coder:
    reasoning:
      enable_streaming: true
      show_thinking: true  # Show reasoning process
    tools: ["write_file", "execute_command", "search"]
```

---

## Monitoring & Debugging

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
| `/agents/{agent}/messages/stream` | POST | Stream message (SSE) |
| `/agents/{agent}/sessions/{id}/messages/stream` | POST | Stream in session (SSE) |
| `/agents/{agent}/stream` | WS | WebSocket streaming |

### gRPC Methods

```protobuf
service A2AService {
  rpc StreamMessage(SendMessageRequest) returns (stream MessageChunk);
}
```

See [API Reference](../reference/api.md) for full details.

---

## Best Practices

### Enable for Interactive Apps

```yaml
# ✅ Good: Streaming for interactive apps
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

### Buffering for Display

```javascript
// ✅ Good: Buffer and update UI efficiently
let buffer = '';
let updateTimer = null;

eventSource.onmessage = (event) => {
  buffer += event.data;

  clearTimeout(updateTimer);
  updateTimer = setTimeout(() => {
    updateUI(buffer);
    buffer = '';
  }, 50); // Update every 50ms
};
```

---

## Next Steps

- **[Sessions](sessions.md)** - Combine streaming with sessions
- **[API Reference](../reference/api.md)** - Complete API documentation
- **[Memory](memory.md)** - Configure memory strategies
- **[Build a Chat Application](../how-to/build-coding-assistant.md)** - Complete tutorial

---

## Related Topics

- **[Agent Overview](overview.md)** - Understanding agents
- **[Configuration Reference](../reference/configuration.md)** - All streaming options
- **[Reasoning](reasoning.md)** - Reasoning engines
