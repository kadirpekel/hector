---
title: A2A Protocol
description: Agent-to-Agent protocol compliance and interoperability
---

# A2A Protocol

Hector is **100% compliant** with the [A2A Protocol](https://a2a-protocol.org) specification, enabling seamless interoperability with any A2A-compliant system.

---

## What is A2A?

The **Agent-to-Agent (A2A) Protocol** is an open standard for agent communication and interoperability.

### Key Benefits

- **Interoperability** - Connect agents from different platforms
- **Standardization** - Consistent API across implementations
- **Future-proof** - Evolving standard with community support
- **Multi-transport** - gRPC, REST, JSON-RPC, WebSocket

---

## Hector's A2A Compliance

### Protocol Native Design

Hector is built **entirely** on A2A protocol types:

```
┌─────────────────────────────────────┐
│     A2A Protocol (Protobuf)         │
├─────────────────────────────────────┤
│     Hector Implementation           │
│  • Direct protobuf usage            │
│  • No abstraction layers            │
│  • Native type system               │
└─────────────────────────────────────┘
```

**Why this matters:**
- Maximum performance (no conversion overhead)
- 100% spec compliance guaranteed
- Native protocol evolution support

### Compliance Summary

| Feature | Status | Details |
|---------|--------|---------|
| **Core Methods** | ✅ 100% | message/send, tasks/get, tasks/cancel |
| **Streaming** | ✅ 100% | gRPC streams, SSE, WebSocket |
| **Task Management** | ✅ 100% | Async execution, status tracking |
| **Agent Discovery** | ✅ 100% | RFC 8615 .well-known endpoints |
| **Authentication** | ✅ 100% | JWT, OpenAPI security schemes |
| **Transport** | ✅ 100% | gRPC, REST, JSON-RPC |

---

## Core Methods

### message/send

Send a message to an agent.

**Request:**
```json
{
  "message": {
    "role": "ROLE_USER",
    "content": [{"text": "Hello"}]
  }
}
```

**Response:**
```json
{
  "response": {
    "role": "ROLE_ASSISTANT",
    "content": [{"text": "Hello! How can I help?"}]
  },
  "task_id": "task_abc123"
}
```

### message/stream

Send a message with streaming response.

**Request:** Same as message/send

**Response:** Stream of chunks
```json
{"response": {"content": [{"text": "Hello"}]}}
{"response": {"content": [{"text": "! How"}]}}
{"response": {"content": [{"text": " can I help?"}]}}
```

### card/get

Get agent metadata.

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

### tasks/get

Get task status.

**Response:**
```json
{
  "task": {
    "id": "task_abc123",
    "status": "completed",
    "result": {
      "role": "ROLE_ASSISTANT",
      "content": [{"text": "Task completed"}]
    }
  }
}
```

### tasks/cancel

Cancel a running task.

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

## Agent Discovery

Hector implements RFC 8615 `.well-known` endpoints for agent discovery.

### Service-Level Discovery

**Endpoint:** `GET /.well-known/a2a`

Lists all available agents:

```json
{
  "agents": [
    {
      "id": "assistant",
      "name": "My Assistant",
      "url": "/v1/agents/assistant"
    },
    {
      "id": "coder",
      "name": "Coding Assistant",
      "url": "/v1/agents/coder"
    }
  ]
}
```

### Agent-Level Discovery

**Endpoint:** `GET /.well-known/a2a/agents/{agent}`

Get specific agent card:

```json
{
  "agent": {
    "name": "My Assistant",
    "description": "A helpful AI assistant",
    "version": "1.0.0",
    "capabilities": ["chat", "tools", "streaming"],
    "supported_content_types": ["text"],
    "security": {
      "schemes": {
        "bearer_auth": {
          "type": "http",
          "scheme": "bearer"
        }
      }
    }
  }
}
```

---

## Message Format

### Message Structure

```protobuf
message Message {
  Role role = 1;  // ROLE_USER | ROLE_ASSISTANT
  repeated Content content = 2;
}

message Content {
  oneof content {
    string text = 1;
    // Future: image, audio, video, file
  }
}
```

### Roles

| Role | Value | Description |
|------|-------|-------------|
| `ROLE_USER` | 0 | User message |
| `ROLE_ASSISTANT` | 1 | Agent response |

### Content Types

| Type | Status | Description |
|------|--------|-------------|
| `text` | ✅ Supported | Plain text content |
| `image` | ⏳ Future | Image content |
| `audio` | ⏳ Future | Audio content |
| `video` | ⏳ Future | Video content |
| `file` | ⏳ Future | File attachments |

---

## Task Lifecycle

### Task States

```
pending → running → completed
                 → failed
                 → cancelled
```

| State | Description |
|-------|-------------|
| `pending` | Task queued, not started |
| `running` | Task in progress |
| `completed` | Task finished successfully |
| `failed` | Task failed with error |
| `cancelled` | Task cancelled by user |

### Task Management

**Create Task:**
```bash
POST /v1/agents/assistant/message:send
```

**Get Status:**
```bash
GET /v1/agents/assistant/tasks/{task_id}
```

**Cancel Task:**
```bash
POST /v1/agents/assistant/tasks/{task_id}:cancel
```

**Subscribe to Updates:**
```bash
POST /v1/agents/assistant/tasks/{task_id}:subscribe
```

---

## Authentication

### JWT Tokens

Hector supports JWT-based authentication:

```
Authorization: Bearer <jwt_token>
```

**Token Claims:**
```json
{
  "sub": "user-123",
  "iss": "https://auth.example.com",
  "aud": "hector-api",
  "roles": ["user", "admin"],
  "exp": 1234567890
}
```

### Security Schemes

Hector supports OpenAPI security schemes:

```yaml
agents:
  assistant:
    security:
      schemes:
        bearer_auth:
          type: "http"
          scheme: "bearer"
        api_key:
          type: "apiKey"
          name: "X-API-Key"
          in: "header"
      require:
        - bearer_auth
```

---

## Streaming

Hector supports multiple streaming protocols.

### gRPC Streaming

```protobuf
rpc SendStreamingMessage(SendMessageRequest) returns (stream StreamResponse);
```

### Server-Sent Events (SSE)

```bash
curl -N -H "Accept: text/event-stream" \
  http://localhost:8081/v1/agents/assistant/message:stream
```

**Response:**
```
data: {"response":{"content":[{"text":"Hello"}]}}
data: {"response":{"content":[{"text":"!"}]}}
data: [DONE]
```

### WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8081/v1/agents/assistant/stream');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(data.response.content[0].text);
};
```

---

## Interoperability

### Connect to External A2A Agents

Hector can call any A2A-compliant agent:

```yaml
agents:
  external:
    type: "a2a"
    url: "https://external-agent.com"
    credentials:
      type: "bearer"
      token: "${EXTERNAL_TOKEN}"
```

### Expose Hector Agents

Any A2A client can call Hector agents:

```bash
# Start Hector server
hector serve --config config.yaml

# Call from any A2A client
curl -X POST http://localhost:8081/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{"message":{"role":"ROLE_USER","content":[{"text":"Hello"}]}}'
```

---

## Transport Protocols

### gRPC (Port 8080)

**Protocol:** HTTP/2 with Protocol Buffers

**Features:**
- Binary protocol
- Bidirectional streaming
- High performance

**Example:**
```bash
grpcurl -plaintext \
  -H 'agent-name: assistant' \
  -d '{"request":{"role":"ROLE_USER","content":[{"text":"Hello"}]}}' \
  localhost:8080 \
  a2a.v1.A2AService/SendMessage
```

### REST (Port 8081)

**Protocol:** HTTP/1.1 with JSON

**Features:**
- RESTful URLs
- JSON payload
- SSE streaming

**Example:**
```bash
curl -X POST http://localhost:8081/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{"message":{"role":"ROLE_USER","content":[{"text":"Hello"}]}}'
```

### JSON-RPC (Port 8082)

**Protocol:** HTTP/1.1 with JSON-RPC 2.0

**Features:**
- Single endpoint
- Method-based routing
- Simple integration

**Example:**
```bash
curl -X POST http://localhost:8082/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "name": "assistant",
      "message": {"role":"ROLE_USER","content":[{"text":"Hello"}]}
    },
    "id": 1
  }'
```

---

## Error Handling

### Error Codes

| Code | HTTP | Description |
|------|------|-------------|
| `OK` | 200 | Success |
| `INVALID_ARGUMENT` | 400 | Invalid request |
| `UNAUTHENTICATED` | 401 | Missing/invalid auth |
| `PERMISSION_DENIED` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Agent not found |
| `RESOURCE_EXHAUSTED` | 429 | Rate limit exceeded |
| `INTERNAL` | 500 | Internal error |
| `UNAVAILABLE` | 503 | Service unavailable |

### Error Response

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Agent not found: unknown_agent",
    "details": {
      "name": "unknown_agent"
    }
  }
}
```

---

## Protocol Extensions

Hector supports protocol extensions while maintaining compatibility:

### Tool Calling

```json
{
  "response": {
    "role": "ROLE_ASSISTANT",
    "content": [{"text": "I'll search for that."}],
    "tool_calls": [
      {
        "id": "call_123",
        "type": "function",
        "function": {
          "name": "search",
          "arguments": "{\"query\":\"AI trends\"}"
        }
      }
    ]
  }
}
```

### Metadata

```json
{
  "response": {
    "role": "ROLE_ASSISTANT",
    "content": [{"text": "Response"}],
    "metadata": {
      "tokens_used": 150,
      "model": "gpt-4o",
      "execution_time": 1.5
    }
  }
}
```

---

## Compliance Testing

### Test Your Implementation

```bash
# Check agent discovery
curl http://localhost:8081/.well-known/a2a

# Test message/send
curl -X POST http://localhost:8081/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{"message":{"role":"ROLE_USER","content":[{"text":"Hello"}]}}'

# Test streaming
curl -N -H "Accept: text/event-stream" \
  http://localhost:8081/v1/agents/assistant/message:stream

# Test task status
curl http://localhost:8081/v1/agents/assistant/tasks/{task_id}
```

---

## Specification Reference

**A2A Protocol Specification:** https://a2a-protocol.org/latest/specification/

**Key Sections:**
- Section 4: Core Methods
- Section 5: Agent Discovery (RFC 8615)
- Section 6: Authentication
- Section 7: Transport Protocols
- Section 8: Task Management
- Section 9: Streaming

---

## Next Steps

- **[API Reference](api.md)** - Detailed API documentation
- **[Architecture](architecture.md)** - System architecture
- **[External Agents](../how-to/integrate-external-agents.md)** - Connect to A2A agents
- **[Multi-Agent](../core-concepts/multi-agent.md)** - Multi-agent orchestration

---

## Related Topics

- **[Security](../core-concepts/security.md)** - Authentication setup
- **[Sessions](../core-concepts/sessions.md)** - Session management
- **[Deploy to Production](../how-to/deploy-production.md)** - Production deployment

