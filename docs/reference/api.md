---
title: API Reference
description: Complete API reference for REST, gRPC, and WebSocket
---

# API Reference

Hector provides multiple API transports for interacting with agents. All implement the A2A protocol.

---

## Quick Reference

| Transport | Port | Use Case | Protocol |
|-----------|------|----------|----------|
| **gRPC** | 8080 | High performance | Binary/HTTP2 |
| **REST** | 8081 | Web/Browser | JSON/HTTP1 |
| **JSON-RPC** | 8082 | Simple integration | JSON-RPC 2.0 |
| **WebSocket** | 8081/ws | Real-time streaming | WebSocket |

---

## Base URLs

```
gRPC:     localhost:8080
REST:     http://localhost:8081
JSON-RPC: http://localhost:8082/rpc
WebSocket: ws://localhost:8081/v1/agents/{agent}/stream
```

---

## Authentication

All transports support JWT authentication (when enabled):

**HTTP Header:**
```
Authorization: Bearer <jwt_token>
```

**gRPC Metadata:**
```
authorization: Bearer <jwt_token>
```

---

## REST API

### Send Message

Send a single message to an agent.

**Endpoint:** `POST /v1/agents/{agent}/message:send`

**Request:**
```json
{
  "message": {
    "role": "ROLE_USER",
    "content": [
      {"text": "What is the capital of France?"}
    ]
  }
}
```

**Response:**
```json
{
  "response": {
    "role": "ROLE_ASSISTANT",
    "content": [
      {"text": "The capital of France is Paris."}
    ]
  },
  "task_id": "task_abc123"
}
```

**Example:**
```bash
curl -X POST http://localhost:8081/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "message": {
      "role": "ROLE_USER",
      "content": [{"text": "Hello"}]
    }
  }'
```

### Stream Message (SSE)

Stream responses in real-time using Server-Sent Events.

**Endpoint:** `POST /v1/agents/{agent}/message:sendStream`

**Request:** Same as Send Message

**Response:** Server-Sent Events stream
```
data: {"response":{"content":[{"text":"The"}]}}
data: {"response":{"content":[{"text":" capital"}]}}
data: {"response":{"content":[{"text":" of France"}]}}
data: [DONE]
```

**Example:**
```bash
curl -N -X POST http://localhost:8081/v1/agents/assistant/message:sendStream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "message": {
      "role": "ROLE_USER",
      "content": [{"text": "Explain quantum computing"}]
    }
  }'
```

### Get Agent Card

Get agent metadata and capabilities.

**Endpoint:** `GET /v1/agents/{agent}`

**Response:**
```json
{
  "agent": {
    "name": "My Assistant",
    "description": "A helpful AI assistant",
    "version": "1.0.0",
    "capabilities": ["chat", "tools", "streaming"],
    "supported_content_types": ["text"]
  }
}
```

**Example:**
```bash
curl http://localhost:8081/v1/agents/assistant
```

### List Agents

List all available agents.

**Endpoint:** `GET /v1/agents`

**Response:**
```json
{
  "agents": [
    {"id": "assistant", "name": "My Assistant"},
    {"id": "coder", "name": "Coding Assistant"}
  ]
}
```

**Example:**
```bash
curl http://localhost:8081/v1/agents
```

### Get Task Status

Get status of an async task.

**Endpoint:** `GET /v1/agents/{agent}/tasks/{task_id}`

**Response:**
```json
{
  "task": {
    "id": "task_abc123",
    "status": "completed",
    "result": {
      "role": "ROLE_ASSISTANT",
      "content": [{"text": "Task completed successfully"}]
    }
  }
}
```

### Cancel Task

Cancel a running task.

**Endpoint:** `POST /v1/agents/{agent}/tasks/{task_id}:cancel`

**Response:**
```json
{
  "task": {
    "id": "task_abc123",
    "status": "cancelled"
  }
}
```

---

## gRPC API

### Service Definition

```protobuf
service A2AService {
  // Send message and get response
  rpc SendMessage(SendMessageRequest) returns (SendMessageResponse);
  
  // Send message and stream response
  rpc SendStreamingMessage(SendMessageRequest) returns (stream StreamResponse);
  
  // Get agent metadata
  rpc GetAgentCard(GetAgentCardRequest) returns (AgentCard);
  
  // Task management
  rpc GetTask(GetTaskRequest) returns (Task);
  rpc CancelTask(CancelTaskRequest) returns (Task);
  rpc TaskSubscription(TaskSubscriptionRequest) returns (stream StreamResponse);
}
```

### Send Message

**Request:**
```protobuf
message SendMessageRequest {
  Message request = 1;
}

message Message {
  Role role = 1;  // ROLE_USER, ROLE_ASSISTANT
  repeated Content content = 2;
}

message Content {
  oneof content {
    string text = 1;
  }
}
```

**Metadata:**
```
agent-name: assistant
authorization: Bearer <token>
```

**Example:**
```bash
grpcurl -plaintext \
  -H 'agent-name: assistant' \
  -H 'authorization: Bearer $TOKEN' \
  -d '{
    "request": {
      "role": "ROLE_USER",
      "content": [{"text": "Hello"}]
    }
  }' \
  localhost:8080 \
  a2a.v1.A2AService/SendMessage
```

### Stream Message

**Example:**
```bash
grpcurl -plaintext \
  -H 'agent-name: assistant' \
  -d '{
    "request": {
      "role": "ROLE_USER",
      "content": [{"text": "Explain recursion"}]
    }
  }' \
  localhost:8080 \
  a2a.v1.A2AService/SendStreamingMessage
```

---

## WebSocket API

Real-time bidirectional communication.

**Endpoint:** `ws://localhost:8081/v1/agents/{agent}/stream`

### Connection

```javascript
const ws = new WebSocket('ws://localhost:8081/v1/agents/assistant/stream');

ws.onopen = () => {
  console.log('Connected');
  
  // Send message
  ws.send(JSON.stringify({
    message: {
      role: 'ROLE_USER',
      content: [{text: 'Hello'}]
    }
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received:', data);
};

ws.onerror = (error) => {
  console.error('Error:', error);
};

ws.onclose = () => {
  console.log('Disconnected');
};
```

### Message Format

**Send:**
```json
{
  "message": {
    "role": "ROLE_USER",
    "content": [{"text": "Hello"}]
  }
}
```

**Receive:**
```json
{
  "response": {
    "role": "ROLE_ASSISTANT",
    "content": [{"text": "Hello! How can I help?"}]
  }
}
```

---

## JSON-RPC API

Simple JSON-RPC 2.0 interface.

**Endpoint:** `POST /rpc`

### Send Message

**Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "message/send",
  "params": {
    "agent_id": "assistant",
    "message": {
      "role": "ROLE_USER",
      "content": [{"text": "Hello"}]
    }
  },
  "id": 1
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "response": {
      "role": "ROLE_ASSISTANT",
      "content": [{"text": "Hello! How can I help?"}]
    },
    "task_id": "task_abc123"
  },
  "id": 1
}
```

**Example:**
```bash
curl -X POST http://localhost:8082/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "agent_id": "assistant",
      "message": {
        "role": "ROLE_USER",
        "content": [{"text": "Hello"}]
      }
    },
    "id": 1
  }'
```

### Get Agent Card

**Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "card/get",
  "params": {
    "agent_id": "assistant"
  },
  "id": 1
}
```

---

## Sessions

Create and manage conversation sessions.

### Create Session

**REST:** `POST /v1/agents/{agent}/sessions`

**Response:**
```json
{
  "session_id": "sess_abc123"
}
```

### Send Message in Session

**REST:** `POST /v1/agents/{agent}/sessions/{session_id}/messages`

**Request:**
```json
{
  "message": {
    "role": "ROLE_USER",
    "content": [{"text": "Hello"}]
  }
}
```

### List Sessions

**REST:** `GET /v1/agents/{agent}/sessions`

### Delete Session

**REST:** `DELETE /v1/agents/{agent}/sessions/{session_id}`

---

## Error Handling

### Error Response Format

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "Agent not found: unknown_agent",
    "details": {
      "agent_id": "unknown_agent"
    }
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `OK` | 200 | Success |
| `INVALID_ARGUMENT` | 400 | Invalid request parameters |
| `UNAUTHENTICATED` | 401 | Missing or invalid authentication |
| `PERMISSION_DENIED` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Agent or resource not found |
| `RESOURCE_EXHAUSTED` | 429 | Rate limit exceeded |
| `INTERNAL` | 500 | Internal server error |
| `UNAVAILABLE` | 503 | Service temporarily unavailable |

### Example Error Response

```bash
curl http://localhost:8081/v1/agents/unknown
```

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Agent not found: unknown"
  }
}
```

---

## Rate Limiting

Hector supports rate limiting per agent and per user.

**Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640000000
```

**Rate Limit Exceeded:**
```json
{
  "error": {
    "code": "RESOURCE_EXHAUSTED",
    "message": "Rate limit exceeded. Try again in 60 seconds."
  }
}
```

---

## Content Types

### Supported Content

**Text:**
```json
{
  "content": [{"text": "Hello"}]
}
```

**Multiple Chunks:**
```json
{
  "content": [
    {"text": "First part"},
    {"text": "Second part"}
  ]
}
```

**Future Support:**
- Images
- Audio
- Video
- Files

---

## Client Libraries

### JavaScript/TypeScript

```typescript
import { A2AClient } from '@hector/client';

const client = new A2AClient({
  baseUrl: 'http://localhost:8081',
  token: 'your-jwt-token'
});

// Send message
const response = await client.sendMessage('assistant', {
  role: 'ROLE_USER',
  content: [{text: 'Hello'}]
});

// Stream message
const stream = client.streamMessage('assistant', {
  role: 'ROLE_USER',
  content: [{text: 'Explain AI'}]
});

for await (const chunk of stream) {
  process.stdout.write(chunk.content[0].text);
}
```

### Python

```python
from hector_client import A2AClient

client = A2AClient(
    base_url='http://localhost:8081',
    token='your-jwt-token'
)

# Send message
response = client.send_message('assistant', {
    'role': 'ROLE_USER',
    'content': [{'text': 'Hello'}]
})

# Stream message
for chunk in client.stream_message('assistant', {
    'role': 'ROLE_USER',
    'content': [{'text': 'Explain AI'}]
}):
    print(chunk['content'][0]['text'], end='')
```

### Go

```go
import "github.com/kadirpekel/hector/pkg/client"

client := client.New(client.Config{
    BaseURL: "http://localhost:8081",
    Token:   "your-jwt-token",
})

// Send message
response, err := client.SendMessage(ctx, "assistant", &a2a.Message{
    Role:    a2a.ROLE_USER,
    Content: []*a2a.Content{{Text: "Hello"}},
})

// Stream message
stream, err := client.StreamMessage(ctx, "assistant", &a2a.Message{
    Role:    a2a.ROLE_USER,
    Content: []*a2a.Content{{Text: "Explain AI"}},
})

for {
    chunk, err := stream.Recv()
    if err == io.EOF {
        break
    }
    fmt.Print(chunk.Content[0].Text)
}
```

---

## Health Check

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "healthy",
  "version": "0.x.x",
  "uptime": "2h15m30s"
}
```

---

## Next Steps

- **[A2A Protocol](a2a-protocol.md)** - Protocol details
- **[CLI Reference](cli.md)** - Command-line API usage
- **[Configuration](configuration.md)** - Server configuration
- **[Authentication](../core-concepts/security.md)** - Auth setup

---

## Related Topics

- **[Sessions & Streaming](../core-concepts/sessions.md)** - Session management
- **[Deploy to Production](../how-to/deploy-production.md)** - Production setup
- **[Architecture](architecture.md)** - System architecture

