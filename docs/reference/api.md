---
title: API Reference
description: Complete API reference for REST, gRPC, and WebSocket
---

# API Reference

Hector provides multiple API transports for interacting with agents. All APIs implement the [A2A Protocol](https://a2a-protocol.org) specification.

---

## Quick Reference

| Transport | Port | Use Case | Protocol |
|-----------|------|----------|----------|
| **gRPC** | 50051 | High performance | Binary/HTTP2 |
| **REST** | 8080 | Web/Browser | JSON/HTTP1 |
| **JSON-RPC** | 8080 | Simple integration | JSON-RPC 2.0 |
| **WebSocket** | 8080/ws | Real-time streaming | WebSocket |
| **Web UI** | 8080 | Interactive browser UI | HTML/JavaScript |

**All HTTP-based APIs consolidated on port 8080 for simplicity.**

---

## Base URLs

```
HTTP (REST/JSON-RPC/WebSocket/UI): http://localhost:8080
gRPC:                               localhost:50051

Specific endpoints:
  REST API:     http://localhost:8080/v1/
  JSON-RPC:     http://localhost:8080/ (POST)
  Web UI:       http://localhost:8080/ (GET)
  WebSocket:    ws://localhost:8080/v1/agents/{agent}/stream
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
    "role": "user",
    "parts": [
      {"text": "What is the capital of France?"}
    ],
    "contextId": "session-123"
  }
}
```

**Response:**
```json
{
  "task": {
    "id": "tasks/task_abc123",
    "contextId": "session-123",
    "status": {
      "state": "completed",
      "message": "Task completed"
    },
    "result": {
      "role": "agent",
      "parts": [
        {"text": "The capital of France is Paris."}
      ],
      "messageId": "msg_xyz789"
    }
  }
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "message": {
      "role": "user",
      "parts": [{"text": "Hello"}],
      "contextId": "my-session"
    }
  }'
```

### Stream Message (SSE)

Stream responses in real-time using Server-Sent Events.

**Endpoint:** `POST /v1/agents/{agent}/message:stream`

**Request:** Same as Send Message

**Response:** Server-Sent Events stream
```
event: message
data: {"message":{"role":"agent","parts":[{"text":"The"}],"messageId":"msg_1"}}

event: message
data: {"message":{"role":"agent","parts":[{"text":" capital"}],"messageId":"msg_1"}}

event: message
data: {"message":{"role":"agent","parts":[{"text":" of France"}],"messageId":"msg_1"}}

event: message
data: {"statusUpdate":{"taskId":"tasks/123","status":{"state":"completed"},"final":true}}
```

**Example:**
```bash
curl -N -X POST http://localhost:8080/v1/agents/assistant/message:stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "message": {
      "role": "user",
      "parts": [{"text": "Explain quantum computing"}],
      "contextId": "my-session"
    }
  }'
```

### Get Agent Card

Get agent metadata and capabilities.

**Endpoint:** `GET /v1/agents/{agent}/.well-known/agent-card.json`

**Per A2A Specification Section 5.3**, agent cards MUST be at `/.well-known/agent-card.json`

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
# Get agent card for specific agent
curl http://localhost:8080/v1/agents/assistant/.well-known/agent-card.json

# With authentication
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/v1/agents/assistant/.well-known/agent-card.json
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
curl http://localhost:8080/v1/agents
```

### Get Task Status

Get status of an async task.

**Endpoint:** `GET /v1/agents/{agent}/tasks/{task_id}`

**Alternative:** `GET /v1/tasks/{task_id}` (agent routing via context)

**Response:**
```json
{
  "id": "tasks/task_abc123",
  "contextId": "session-123",
  "status": {
    "state": "completed",
    "message": "Task completed successfully"
  },
  "result": {
    "role": "agent",
    "parts": [{"text": "Task completed successfully"}],
    "messageId": "msg_xyz789"
  },
  "createdAt": "2025-10-23T10:00:00Z",
  "updatedAt": "2025-10-23T10:00:05Z"
}
```

**Example:**
```bash
# Agent-specific endpoint
curl http://localhost:8080/v1/agents/assistant/tasks/task_abc123

# Generic endpoint (uses context routing)
curl http://localhost:8080/v1/tasks/task_abc123
```

### Cancel Task

Cancel a running task.

**Endpoint:** `POST /v1/agents/{agent}/tasks/{task_id}:cancel`

**Alternative:** `POST /v1/tasks/{task_id}:cancel` (agent routing via context)

**Response:**
```json
{
  "id": "tasks/task_abc123",
  "contextId": "session-123",
  "status": {
    "state": "canceled",
    "message": "Task cancelled by user"
  },
  "createdAt": "2025-10-23T10:00:00Z",
  "updatedAt": "2025-10-23T10:00:10Z"
}
```

**Example:**
```bash
# Agent-specific endpoint
curl -X POST http://localhost:8080/v1/agents/assistant/tasks/task_abc123:cancel

# Generic endpoint (uses context routing)
curl -X POST http://localhost:8080/v1/tasks/task_abc123:cancel
```

**Task States:**

| State | Description |
|-------|-------------|
| `submitted` | Task created and queued |
| `working` | Task is being processed |
| `completed` | Task finished successfully |
| `failed` | Task failed with error |
| `canceled` | Task was cancelled |
| `input-required` | Task needs user input |

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

**Endpoint:** `ws://localhost:8080/v1/agents/{agent}/stream`

### Connection

```javascript
const ws = new WebSocket('ws://localhost:8080/v1/agents/assistant/stream');

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

Simple JSON-RPC 2.0 interface with agent-scoped endpoints.

**Endpoint Pattern:** `POST /v1/agents/{agent_id}/`

All JSON-RPC requests are agent-scoped - the agent is identified by the URL path, not query parameters.

### Agent-Scoped Endpoints

```bash
# Non-streaming JSON-RPC
POST /v1/agents/{agent_id}/

# Streaming JSON-RPC (SSE)
POST /v1/agents/{agent_id}/stream
```

**Examples:**
```bash
# Send message to orchestrator agent
POST http://localhost:8080/v1/agents/orchestrator/

# Streaming to assistant agent
POST http://localhost:8080/v1/agents/assistant/stream
```

### Send Message

**Request:**
```json
{
  "jsonrpc": "2.0",
  "method": "message/send",
  "params": {
    "name": "assistant",
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

**Examples:**

```bash
# Agent-scoped JSON-RPC
curl -X POST http://localhost:8080/v1/agents/assistant/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "request": {
        "role": "user",
        "parts": [{"text": "Hello"}]
      }
    },
    "id": 1
  }'

# Agent-scoped with session context
curl -X POST http://localhost:8080/v1/agents/orchestrator/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "request": {
        "contextId": "session-123",
        "role": "user",
        "parts": [{"text": "Hello"}]
      }
    },
    "id": 1
  }'
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
      "name": "unknown_agent"
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
curl http://localhost:8080/v1/agents/unknown
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
  baseUrl: 'http://localhost:8080',
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
    base_url='http://localhost:8080',
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
    BaseURL: "http://localhost:8080",
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

