---
title: A2A Protocol
description: Agent-to-Agent protocol compliance and interoperability
---

# A2A Protocol

Hector implements the [A2A Protocol](https://a2a-protocol.org) specification for seamless interoperability with any A2A-compliant system.

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

### Specification Version

**Hector is 100% compliant with A2A Protocol v0.3.0**

### Protocol Native Design

Hector is built **entirely** on A2A protocol types:

```
┌─────────────────────────────────────┐
│   A2A Protocol v0.3.0 (Protobuf)    │
├─────────────────────────────────────┤
│     Hector Implementation           │
│  • Official proto files (unmodified)│
│  • Direct protobuf usage            │
│  • No abstraction layers            │
│  • Native type system               │
└─────────────────────────────────────┘
```

**Why this matters:**
- Maximum performance (no conversion overhead)
- 100% spec compliance guaranteed
- Native protocol evolution support
- Direct compatibility with A2A ecosystem

### Compliance Summary

| Feature | Status | Details |
|---------|--------|---------|
| **Protocol Version** | ✅ v0.3.0 | Fully compliant with latest spec |
| **Core Methods** | ✅ 100% | message/send, tasks/get, tasks/cancel |
| **Optional Methods** | ✅ 100% | Streaming, resubscribe, agent card |
| **Task Management** | ✅ 100% | All 9 task states, async execution |
| **Agent Discovery** | ✅ 100% | RFC 8615 .well-known endpoints |
| **Authentication** | ✅ 100% | JWT, OpenAPI security schemes |
| **Transport** | ✅ All 3 | gRPC (50051), REST (8080), JSON-RPC (8080) |
| **REST Paths** | ✅✅ **Enhanced** | Dual path support (spec + proto) |

### Enhanced REST Compliance 🌟

Hector **exceeds** the A2A specification by supporting **both** REST path formats:

1. **Spec-preferred format** (Section 3.5.3):
   - `/v1/agents/{agent}/message:send`
   - Agent extracted from URL path

2. **Proto-compatible format**:
   - `/v1/message:send` with `agent-name: {agent}` header
   - Agent extracted from HTTP header or query param

This dual-path support ensures **maximum interoperability** with all A2A implementations.

---

## Core Methods

### message/send

Send a message to an agent.

**Request:**
```json
{
  "message": {
    "role": "ROLE_USER",
    "parts": [{"text": "Hello"}]
  }
}
```

**Response:**
```json
{
  "response": {
    "role": "ROLE_AGENT",
    "parts": [{"text": "Hello! How can I help?"}]
  },
  "task_id": "task_abc123"
}
```

### message/stream

Send a message with streaming response.

**Request:** Same as message/send

**Response:** Stream of chunks
```json
{"response": {"parts": [{"text": "Hello"}]}}
{"response": {"parts": [{"text": "! How"}]}}
{"response": {"parts": [{"text": " can I help?"}]}}
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
      "role": "ROLE_AGENT",
      "parts": [{"text": "Task completed"}]
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

### Agent Card Location

**Per A2A Specification Section 5.3**, agent cards **MUST** be located at:

```
/.well-known/agent.json
```

For multi-agent services, each agent has its own dedicated path:

```
/v1/agents/{agent}/.well-known/agent.json
```

### Multi-Agent Discovery

**Endpoint:** `GET /v1/agents`

List all agents via the discovery endpoint:

```bash
curl http://localhost:8080/v1/agents
```

```json
{
  "agents": [
    {
      "name": "assistant",
      "url": "http://localhost:8080/v1/agents/assistant"
    },
    {
      "name": "coder",
      "url": "http://localhost:8080/v1/agents/coder"
    }
  ]
}
```

### Agent Card Structure

**Endpoint:** `GET /v1/agents/{agent}/.well-known/agent.json`

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
  Role role = 1;           // ROLE_USER | ROLE_AGENT | ROLE_UNSPECIFIED
  repeated Part parts = 2; // Message parts (text, file, data)
  string context_id = 3;   // Optional session/context identifier
}

message Part {
  oneof part {
    string text = 1;       // TextPart (Section 6.5.1)
    File file = 2;         // FilePart (Section 6.5.2)
    google.protobuf.Struct data = 3; // DataPart (Section 6.5.3)
  }
}
```

**Note:** The field is named `parts` (not `content`) per A2A v0.3.0 specification.

### Roles

Per A2A Specification Section 6.4:

| Role | Value | Description |
|------|-------|-------------|
| `ROLE_UNSPECIFIED` | 0 | Default/system messages |
| `ROLE_USER` | 1 | User message |
| `ROLE_AGENT` | 2 | Agent response |

### Part Types

Per A2A Specification Section 6.5:

| Type | Status | Description | Spec Section |
|------|--------|-------------|--------------|
| `TextPart` | ✅ Supported | Plain text messages | 6.5.1 |
| `FilePart` | ✅ Supported | File references with URIs | 6.5.2 |
| `DataPart` | ✅ Supported | Structured JSON data | 6.5.3 |

---

## Task Lifecycle

### Task States

Per A2A Specification Section 6.3, all 9 task states are supported:

```
SUBMITTED → WORKING → COMPLETED
                   → FAILED
                   → CANCELLED
                   → INPUT_REQUIRED
                   → REJECTED
                   → AUTH_REQUIRED
```

| State | Description | Spec Section |
|-------|-------------|--------------|
| `TASK_STATE_UNSPECIFIED` | Default/unknown state | 6.3 |
| `TASK_STATE_SUBMITTED` | Task created and queued | 6.3 |
| `TASK_STATE_WORKING` | Task actively processing | 6.3 |
| `TASK_STATE_COMPLETED` | Task finished successfully | 6.3 |
| `TASK_STATE_FAILED` | Task encountered error | 6.3 |
| `TASK_STATE_CANCELLED` | Task terminated by client | 6.3 |
| `TASK_STATE_INPUT_REQUIRED` | Awaiting user input | 6.3 |
| `TASK_STATE_REJECTED` | Agent refused task | 6.3 |
| `TASK_STATE_AUTH_REQUIRED` | Needs authentication | 6.3 |

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
  http://localhost:8080/v1/agents/assistant/message:stream
```

**Response:**
```
data: {"response":{"content":[{"text":"Hello"}]}}
data: {"response":{"content":[{"text":"!"}]}}
data: [DONE]
```

### WebSocket

```javascript
const ws = new WebSocket('ws://localhost:8080/v1/agents/assistant/stream');

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
curl -X POST http://localhost:8080/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{"message":{"role":"ROLE_USER","content":[{"text":"Hello"}]}}'
```

---

## Transport Protocols

Hector implements all three A2A transport protocols on conventional ports:

| Transport | Port | Endpoint | Protocol |
|-----------|------|----------|----------|
| gRPC | **50051** | `localhost:50051` | HTTP/2 + Protobuf |
| REST | **8080** | `http://localhost:8080/v1/*` | HTTP/1.1 + JSON |
| JSON-RPC | **8080** | `http://localhost:8080/` | HTTP/1.1 + JSON-RPC 2.0 |

### gRPC (Port 50051)

**Protocol:** HTTP/2 with Protocol Buffers

**Features:**
- Binary protocol
- Bidirectional streaming
- High performance
- Native A2A proto support

**Example:**
```bash
grpcurl -plaintext \
  -H 'agent-name: assistant' \
  -d '{"request":{"role":"ROLE_USER","parts":[{"text":"Hello"}]}}' \
  localhost:50051 \
  a2a.v1.A2AService/SendMessage
```

### REST (Port 8080)

**Protocol:** HTTP/1.1 with JSON

**Features:**
- RESTful URLs
- JSON payload
- SSE streaming
- Dual path support (spec + proto formats)

**Examples:**

**Spec-preferred format** (agent in path):
```bash
curl -X POST http://localhost:8080/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{"message":{"role":"ROLE_USER","parts":[{"text":"Hello"}]}}'
```

**Proto-compatible format** (agent in header):
```bash
curl -X POST http://localhost:8080/v1/message:send \
  -H "Content-Type: application/json" \
  -H "agent-name: assistant" \
  -d '{"message":{"role":"ROLE_USER","parts":[{"text":"Hello"}]}}'
```

### JSON-RPC (Port 8080)

**Protocol:** HTTP/1.1 with JSON-RPC 2.0

**Features:**
- Single endpoint: `POST /` (root path)
- Streaming endpoint: `POST /stream` (SSE)
- Method-based routing
- Simple integration
- Method names follow A2A spec (e.g., `message/send`, `tasks/get`)

**Examples:**

**Send message:**
```bash
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {"role":"ROLE_USER","parts":[{"text":"Hello"}]}
    },
    "id": 1
  }'
```

**With agent selection** (multi-agent mode):
```bash
curl -X POST "http://localhost:8080/?agent=assistant" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {"role":"ROLE_USER","parts":[{"text":"Hello"}]}
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
    "role": "ROLE_AGENT",
    "parts": [{"text": "I'll search for that."}],
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
    "role": "ROLE_AGENT",
    "parts": [{"text": "Response"}],
    "metadata": {
      "tokens_used": 150,
      "model": "gpt-4o",
      "execution_time": 1.5
    }
  }
}
```

---

## Testing A2A Endpoints

### Agent Discovery

```bash
# Get agent card (multi-agent service)
curl http://localhost:8080/v1/agents/assistant/.well-known/agent.json

# List all agents
curl http://localhost:8080/v1/agents
```

### Send Message

```bash
curl -X POST http://localhost:8080/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "role": "user",
      "parts": [{"text": "Hello"}],
      "contextId": "session-123"
    }
  }'
```

### Stream Message (SSE)

```bash
curl -N -X POST http://localhost:8080/v1/agents/assistant/message:stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "message": {
      "role": "user",
      "parts": [{"text": "Hello"}],
      "contextId": "session-123"
    }
  }'
```

### JSON-RPC

```bash
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"text": "Hello"}],
        "contextId": "session-123"
      }
    },
    "id": 1
  }'
```

### Task Management

```bash
# Get task status
curl http://localhost:8080/v1/agents/assistant/tasks/task_abc123

# Cancel task
curl -X POST http://localhost:8080/v1/agents/assistant/tasks/task_abc123:cancel
```

---

## Specification Reference

**A2A Protocol Specification:** https://a2a-protocol.org/latest/specification/

**Official Proto Files:** https://github.com/a2aproject/A2A/blob/main/specification/grpc/a2a.proto

**Key Specification Sections:**
- Section 3: Transport Protocols (gRPC, REST, JSON-RPC)
- Section 5: Agent Discovery (RFC 8615)
- Section 6: Message Structure and Task Lifecycle
- Section 11: Compliance Requirements

**Hector Implementation Details:**
- All 3 transport protocols ✅
- All required methods + optional methods ✅
- Dual REST path support (spec + proto formats) ✅✅
- All 9 task states ✅
- All 3 message part types (text, file, data) ✅

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

