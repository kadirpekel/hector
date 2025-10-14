---
layout: default
title: API Reference
nav_order: 2
parent: Reference
description: "Complete A2A Protocol API reference"
---

# API Reference

**Complete A2A Protocol API Reference for Hector**

---

## Table of Contents

- [Overview](#overview)
- [Transport Protocols](#transport-protocols)
- [Authentication](#authentication)
- [Discovery Endpoints](#discovery-endpoints)
- [Messaging Endpoints](#messaging-endpoints)
- [Task Management](#task-management)
- [Streaming](#streaming)
- [Error Handling](#error-handling)
- [Client SDKs](#client-sdks)

---

## Overview

Hector provides three transport protocols, all implementing the same [A2A Protocol specification](https://a2a-protocol.org/latest/specification/):

| Transport | Default Port | Use Case | Features |
|-----------|--------------|----------|----------|
| **gRPC** | 8080 | High-performance services | Binary protocol, bidirectional streaming, HTTP/2 |
| **REST** | 8081 | Web applications, browsers | JSON over HTTP, SSE streaming, RESTful |
| **JSON-RPC** | 8082 | Simple integrations | Single endpoint, JSON-RPC 2.0, easy to use |

**Base URLs**:
```
gRPC:     localhost:8080
REST:     http://localhost:8081
JSON-RPC: http://localhost:8082/rpc
```

---

## Transport Protocols

### gRPC

**Protocol**: HTTP/2 with Protocol Buffers

**Service Definition**:
```protobuf
service A2AService {
  rpc SendMessage(SendMessageRequest) returns (SendMessageResponse);
  rpc SendStreamingMessage(SendMessageRequest) returns (stream StreamResponse);
  rpc GetTask(GetTaskRequest) returns (Task);
  rpc CancelTask(CancelTaskRequest) returns (Task);
  rpc TaskSubscription(TaskSubscriptionRequest) returns (stream StreamResponse);
  rpc GetAgentCard(GetAgentCardRequest) returns (AgentCard);
  rpc CreateTaskPushNotificationConfig(CreateTaskPushNotificationConfigRequest) returns (TaskPushNotificationConfig);
  rpc GetTaskPushNotificationConfig(GetTaskPushNotificationConfigRequest) returns (TaskPushNotificationConfig);
  rpc ListTaskPushNotificationConfig(ListTaskPushNotificationConfigRequest) returns (ListTaskPushNotificationConfigResponse);
  rpc DeleteTaskPushNotificationConfig(DeleteTaskPushNotificationConfigRequest) returns (google.protobuf.Empty);
}
```

**Example** (using grpcurl):
```bash
grpcurl -plaintext \
  -d '{"request":{"role":"ROLE_USER","content":[{"text":"Hello"}]}}' \
  -H 'agent-name: assistant' \
  localhost:8080 \
  a2a.v1.A2AService/SendMessage
```

### REST (grpc-gateway)

**Protocol**: HTTP/1.1 with JSON

**Features**:
- Auto-generated from protobuf definitions
- RESTful URL patterns
- Server-Sent Events (SSE) for streaming
- OpenAPI/Swagger compatible

**Agent Routing**: Uses path-based routing with `agent_id` in URL

### JSON-RPC

**Protocol**: HTTP/1.1 with JSON-RPC 2.0

**Endpoint**: `POST /rpc`

**Methods**:
- `SendMessage` - Send non-streaming message
- `GetAgentCard` - Get agent metadata
- `GetTask` - Get task status
- `CancelTask` - Cancel a task

---

## Authentication

All transports support JWT-based authentication (when enabled).

### Bearer Token (JWT)

**HTTP Header**:
```
Authorization: Bearer <jwt_token>
```

**gRPC Metadata**:
```
authorization: Bearer <jwt_token>
```

**Token Structure**:
```json
{
  "sub": "user-123",
  "iss": "https://auth.example.com",
  "aud": "hector-api",
  "roles": ["user", "admin"],
  "tenant_id": "org-456",
  "exp": 1234567890
}
```

### API Key

**HTTP Header**:
```
X-API-Key: <api_key>
```

### Example Authenticated Request

**REST**:
```bash
curl -X POST http://localhost:8081/v1/agents/assistant/message:send \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "Content-Type: application/json" \
  -d '{"message":{"role":"ROLE_USER","content":[{"text":"Hello"}]}}'
```

**gRPC**:
```bash
grpcurl -plaintext \
  -H 'authorization: Bearer eyJhbGc...' \
  -H 'agent-name: assistant' \
  -d '{"request":{"role":"ROLE_USER","content":[{"text":"Hello"}]}}' \
  localhost:8080 \
  a2a.v1.A2AService/SendMessage
```

---

## Discovery Endpoints

### Get Service-Level Agent Card

Get information about the Hector service and list of available agents.

**REST**:
```
GET /.well-known/agent-card.json
```

**Response**:
```json
{
  "name": "Hector A2A Server",
  "description": "Multi-agent AI platform",
  "version": "1.0.0",
  "capabilities": {
    "streaming": true,
    "task_tracking": true,
    "session_support": true,
    "multi_agent": true
  },
  "transports": [
    {"protocol": "grpc", "url": "grpc://localhost:8080"},
    {"protocol": "http", "url": "http://localhost:8081"},
    {"protocol": "jsonrpc", "url": "http://localhost:8082/rpc"}
  ]
}
```

**Example**:
```bash
curl http://localhost:8081/.well-known/agent-card.json
```

### List All Agents

Get a list of all available agents.

**REST**:
```
GET /v1/agents
```

**Response**:
```json
{
  "agents": [
    {
      "id": "assistant",
      "name": "Assistant",
      "description": "Helpful AI assistant",
      "agent_card_url": "/v1/agents/assistant/.well-known/agent-card.json",
      "version": "1.0.0"
    },
    {
      "id": "researcher",
      "name": "Research Assistant",
      "description": "Research and analysis agent",
      "agent_card_url": "/v1/agents/researcher/.well-known/agent-card.json",
      "version": "1.0.0"
    }
  ]
}
```

**Example**:
```bash
curl http://localhost:8081/v1/agents
```

### Get Agent-Specific Card

Get detailed information about a specific agent.

**REST**:
```
GET /v1/agents/{agent_id}/.well-known/agent-card.json
```

**gRPC**:
```protobuf
rpc GetAgentCard(GetAgentCardRequest) returns (AgentCard)
```

**JSON-RPC**:
```json
{
  "jsonrpc": "2.0",
  "method": "GetAgentCard",
  "params": {
    "agentId": "assistant"
  },
  "id": 1
}
```

**Response**:
```json
{
  "name": "Assistant",
  "description": "Helpful AI assistant",
  "version": "1.0.0",
  "capabilities": {
    "streaming": true,
    "tool_calling": true,
    "document_search": false
  },
  "input_modalities": ["text"],
  "output_modalities": ["text"],
  "tools": [
    {
      "name": "command",
      "description": "Execute shell commands",
      "input_schema": {...}
    },
    {
      "name": "file_writer",
      "description": "Write files to disk",
      "input_schema": {...}
    }
  ],
  "security_schemes": {
    "bearer": {
      "type": "http",
      "scheme": "bearer"
    }
  }
}
```

**Examples**:

**REST**:
```bash
curl http://localhost:8081/v1/agents/assistant/.well-known/agent-card.json
```

**gRPC**:
```bash
grpcurl -plaintext \
  -H 'agent-name: assistant' \
  -d '{}' \
  localhost:8080 \
  a2a.v1.A2AService/GetAgentCard
```

**JSON-RPC**:
```bash
curl -X POST http://localhost:8082/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "GetAgentCard",
    "params": {"agentId": "assistant"},
    "id": 1
  }'
```

---

## Messaging Endpoints

### Send Message (Non-Streaming)

Send a message to an agent and receive a complete response.

**REST**:
```
POST /v1/agents/{agent_id}/message:send
```

**gRPC**:
```protobuf
rpc SendMessage(SendMessageRequest) returns (SendMessageResponse)
```

**JSON-RPC**:
```json
{
  "jsonrpc": "2.0",
  "method": "SendMessage",
  "params": {...},
  "id": 1
}
```

**Request Body**:
```json
{
  "message": {
    "role": "ROLE_USER",
    "content": [
      {
        "text": "What is the capital of France?"
      }
    ],
    "metadata": {
      "user_id": "user-123",
      "session_id": "session-456"
    }
  },
  "configuration": {
    "accepted_output_modes": ["text"],
    "blocking": true,
    "history_length": 10
  }
}
```

**Response**:
```json
{
  "message": {
    "messageId": "msg-789",
    "contextId": "ctx-456",
    "role": "ROLE_AGENT",
    "content": [
      {
        "text": "The capital of France is Paris."
      }
    ]
  }
}
```

**Examples**:

**REST**:
```bash
curl -X POST http://localhost:8081/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "role": "ROLE_USER",
      "content": [{"text": "What is 2+2?"}]
    }
  }'
```

**gRPC**:
```bash
grpcurl -plaintext \
  -H 'agent-name: assistant' \
  -d '{
    "request": {
      "role": "ROLE_USER",
      "content": [{"text": "What is 2+2?"}]
    }
  }' \
  localhost:8080 \
  a2a.v1.A2AService/SendMessage
```

**JSON-RPC**:
```bash
curl -X POST http://localhost:8082/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "SendMessage",
    "params": {
      "agentId": "assistant",
      "message": {
        "role": "ROLE_USER",
        "content": [{"text": "What is 2+2?"}]
      }
    },
    "id": 1
  }'
```

### Send Streaming Message

Send a message and receive streaming responses in real-time.

**REST**:
```
POST /v1/agents/{agent_id}/message:stream
Accept: text/event-stream
```

**gRPC**:
```protobuf
rpc SendStreamingMessage(SendMessageRequest) returns (stream StreamResponse)
```

**Request Body**: Same as non-streaming

**Response** (Server-Sent Events):
```
event: message
data: {"result":{"message":{"role":"ROLE_AGENT","content":[{"text":"The"}]}}}

event: message
data: {"result":{"message":{"role":"ROLE_AGENT","content":[{"text":" capital"}]}}}

event: message
data: {"result":{"message":{"role":"ROLE_AGENT","content":[{"text":" is"}]}}}

event: message
data: {"result":{"message":{"role":"ROLE_AGENT","content":[{"text":" Paris."}]}}}

event: status
data: {"result":{"statusUpdate":{"taskId":"task-123","status":{"state":"TASK_STATE_COMPLETED"}}}}
```

**Examples**:

**REST** (curl):
```bash
curl -N -X POST http://localhost:8081/v1/agents/assistant/message:stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "message": {
      "role": "ROLE_USER",
      "content": [{"text": "Tell me a short story"}]
    }
  }'
```

**REST** (JavaScript):
```javascript
const eventSource = new EventSource(
  'http://localhost:8081/v1/agents/assistant/message:stream',
  {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      message: {
        role: 'ROLE_USER',
        content: [{text: 'Tell me a story'}]
      }
    })
  }
);

eventSource.addEventListener('message', (event) => {
  const data = JSON.parse(event.data);
  const msg = data.result.message;
  if (msg && msg.content) {
    console.log(msg.content[0].text);
  }
});

eventSource.addEventListener('status', (event) => {
  const data = JSON.parse(event.data);
  console.log('Status:', data.result.statusUpdate.status.state);
});
```

**gRPC** (Go):
```go
stream, err := client.SendStreamingMessage(ctx, &pb.SendMessageRequest{
    Request: &pb.Message{
        Role: pb.Role_ROLE_USER,
        Content: []*pb.Part{{Part: &pb.Part_Text{Text: "Tell me a story"}}},
    },
})

for {
    resp, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if msg := resp.GetMsg(); msg != nil {
        fmt.Print(msg.Content[0].GetText())
    }
}
```

---

## Task Management

### Get Task Status

Retrieve the current status of a task.

**REST**:
```
GET /v1/tasks/{task_id}
```

**gRPC**:
```protobuf
rpc GetTask(GetTaskRequest) returns (Task)
```

**JSON-RPC**:
```json
{
  "jsonrpc": "2.0",
  "method": "GetTask",
  "params": {
    "name": "tasks/task-123"
  },
  "id": 1
}
```

**Response**:
```json
{
  "id": "task-123",
  "contextId": "ctx-456",
  "status": {
    "state": "TASK_STATE_RUNNING",
    "message": "Processing your request...",
    "progress": 0.5
  },
  "artifacts": [],
  "history": [
    {
      "role": "ROLE_USER",
      "content": [{"text": "User input"}]
    },
    {
      "role": "ROLE_AGENT",
      "content": [{"text": "Agent response"}]
    }
  ]
}
```

**Task States**:
- `TASK_STATE_SUBMITTED` - Task received
- `TASK_STATE_RUNNING` - Task in progress
- `TASK_STATE_COMPLETED` - Task finished successfully
- `TASK_STATE_FAILED` - Task failed with error
- `TASK_STATE_CANCELLED` - Task cancelled by user

**Examples**:

**REST**:
```bash
curl http://localhost:8081/v1/tasks/task-123
```

**JSON-RPC**:
```bash
curl -X POST http://localhost:8082/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "GetTask",
    "params": {"name": "tasks/task-123"},
    "id": 1
  }'
```

### Cancel Task

Cancel a running task.

**REST**:
```
POST /v1/tasks/{task_id}:cancel
```

**gRPC**:
```protobuf
rpc CancelTask(CancelTaskRequest) returns (Task)
```

**JSON-RPC**:
```json
{
  "jsonrpc": "2.0",
  "method": "CancelTask",
  "params": {
    "name": "tasks/task-123"
  },
  "id": 1
}
```

**Response**: Updated task with `TASK_STATE_CANCELLED` status

**Examples**:

**REST**:
```bash
curl -X POST http://localhost:8081/v1/tasks/task-123:cancel \
  -H "Content-Type: application/json" \
  -d '{}'
```

### Subscribe to Task Updates

Subscribe to real-time updates for a task.

**REST**:
```
GET /v1/tasks/{task_id}:subscribe
Accept: text/event-stream
```

**gRPC**:
```protobuf
rpc TaskSubscription(TaskSubscriptionRequest) returns (stream StreamResponse)
```

**Response** (SSE):
```
event: status
data: {"result":{"statusUpdate":{"taskId":"task-123","status":{"state":"TASK_STATE_RUNNING"}}}}

event: message
data: {"result":{"message":{"role":"ROLE_AGENT","content":[{"text":"Progress update"}]}}}

event: status
data: {"result":{"statusUpdate":{"taskId":"task-123","status":{"state":"TASK_STATE_COMPLETED"}}}}
```

**Example**:
```bash
curl -N -H "Accept: text/event-stream" \
  http://localhost:8081/v1/tasks/task-123:subscribe
```

---

## Streaming

### Server-Sent Events (SSE)

REST endpoints use SSE for server-to-client streaming.

**Event Types**:
- `message` - Message chunk from agent
- `status` - Task status update
- `error` - Error message

**Format**:
```
event: <event_type>
data: <json_payload>

```

### gRPC Streaming

gRPC provides native bidirectional streaming.

**Stream Types**:
- **Server Streaming**: `SendStreamingMessage`, `TaskSubscription`
- **Client Streaming**: Not currently used
- **Bidirectional**: Future support for interactive sessions

---

## Error Handling

### REST Errors

**HTTP Status Codes**:
- `200` - Success
- `400` - Bad Request (invalid input)
- `401` - Unauthorized (authentication required)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found (agent or task not found)
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error
- `503` - Service Unavailable

**Error Response**:
```json
{
  "error": {
    "code": 400,
    "message": "Invalid message format",
    "details": [
      {
        "field": "message.content",
        "issue": "content cannot be empty"
      }
    ]
  }
}
```

### gRPC Errors

**Status Codes**:
- `OK (0)` - Success
- `INVALID_ARGUMENT (3)` - Invalid input
- `UNAUTHENTICATED (16)` - Authentication required
- `PERMISSION_DENIED (7)` - Insufficient permissions
- `NOT_FOUND (5)` - Agent or task not found
- `UNAVAILABLE (14)` - Service unavailable

**Error Details**:
```
code: INVALID_ARGUMENT
message: "message text cannot be empty"
details: [...]
```

### JSON-RPC Errors

**Error Codes**:
- `-32700` - Parse error
- `-32600` - Invalid request
- `-32601` - Method not found
- `-32602` - Invalid params
- `-32603` - Internal error

**Error Response**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": {
      "field": "agentId",
      "issue": "Agent not found"
    }
  },
  "id": 1
}
```

---

## Client SDKs

### Hector CLI

Built-in CLI for all operations:

```bash
# List agents
hector list --server http://localhost:8081

# Get agent info
hector info assistant --server http://localhost:8081

# Send message
hector call assistant "What is 2+2?" --server http://localhost:8081

# Interactive chat
hector chat assistant --server http://localhost:8081
```

### Go Client

```go
import "github.com/kadirpekel/hector/pkg/a2a/client"

// Create HTTP client
c := client.NewHTTPClient("http://localhost:8081", "token")

// Send message
resp, err := c.SendMessage(ctx, "assistant", &pb.Message{
    Role: pb.Role_ROLE_USER,
    Content: []*pb.Part{{Part: &pb.Part_Text{Text: "Hello"}}},
})

// Stream messages
stream, err := c.StreamMessage(ctx, "assistant", message)
for chunk := range stream {
    if msg := chunk.GetMsg(); msg != nil {
        fmt.Print(msg.Content[0].GetText())
    }
}
```

### Python Client (grpcio)

```python
import grpc
from a2a.v1 import a2a_pb2, a2a_pb2_grpc

# Create channel
channel = grpc.insecure_channel('localhost:8080')
client = a2a_pb2_grpc.A2AServiceStub(channel)

# Create metadata with agent name
metadata = [('agent-name', 'assistant')]

# Send message
request = a2a_pb2.SendMessageRequest(
    request=a2a_pb2.Message(
        role=a2a_pb2.ROLE_USER,
        content=[a2a_pb2.Part(text="Hello")]
    )
)
response = client.SendMessage(request, metadata=metadata)

# Stream messages
stream = client.SendStreamingMessage(request, metadata=metadata)
for chunk in stream:
    if chunk.HasField('msg'):
        print(chunk.msg.content[0].text, end='', flush=True)
```

### JavaScript/TypeScript (fetch)

```typescript
// Send message
const response = await fetch('http://localhost:8081/v1/agents/assistant/message:send', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer <token>'
  },
  body: JSON.stringify({
    message: {
      role: 'ROLE_USER',
      content: [{text: 'Hello'}]
    }
  })
});

const data = await response.json();
console.log(data.message.content[0].text);

// Stream messages
const eventSource = new EventSource(
  'http://localhost:8081/v1/agents/assistant/message:stream',
  {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      message: {
        role: 'ROLE_USER',
        content: [{text: 'Tell me a story'}]
      }
    })
  }
);

eventSource.addEventListener('message', (event) => {
  const data = JSON.parse(event.data);
  console.log(data.result.message.content[0].text);
});
```

---

## Rate Limiting

Configure rate limiting per agent or globally:

```yaml
global:
  rate_limiting:
    enabled: true
    requests_per_minute: 100
    burst: 20

agents:
  expensive_agent:
    rate_limiting:
      requests_per_minute: 10  # Override global limit
```

**Rate Limit Headers** (REST):
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640000000
```

---

## Webhooks (Push Notifications)

Configure webhooks for task completion:

**Create Webhook**:
```bash
curl -X POST http://localhost:8081/v1/tasks/task-123/pushNotificationConfigs \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "url": "https://your-app.com/webhook",
      "headers": {
        "Authorization": "Bearer webhook-token"
      },
      "events": ["TASK_STATE_COMPLETED", "TASK_STATE_FAILED"]
    }
  }'
```

**Webhook Payload**:
```json
{
  "event": "TASK_STATE_COMPLETED",
  "task": {
    "id": "task-123",
    "status": {
      "state": "TASK_STATE_COMPLETED"
    },
    "artifacts": [...]
  }
}
```

---

## Summary

Hector provides a **complete, production-ready A2A Protocol API** with:

✅ **Three Transports** - gRPC, REST, JSON-RPC  
✅ **Full Spec Compliance** - A2A protocol compliant  
✅ **Authentication** - JWT, API keys, basic auth  
✅ **Streaming** - Real-time responses  
✅ **Discovery** - RFC 8615 well-known endpoints  
✅ **Task Management** - Async processing with status tracking  
✅ **Error Handling** - Comprehensive error responses  
✅ **Client SDKs** - Built-in CLI, Go, Python, JavaScript examples  

**Start building with Hector's API today!** 🚀

For more information:
- [Architecture Guide](ARCHITECTURE.md)
- [CLI Guide](CLI_GUIDE.md)
- [A2A Protocol Specification](https://a2a-protocol.org/latest/specification/)
