# API Reference

Complete, authoritative reference for Hector's HTTP API. Includes A2A protocol endpoints, Admin API, webhooks, and system endpoints.

**Base URL:** `http://localhost:8080` (configurable via `--host` and `--port`)

---

## Authentication

Hector supports two authentication methods:

### Bearer Token (Admin Secret)

For admin operations using the `--auth-secret` flag:

```http
Authorization: Bearer <admin-secret>
```

### JWT Token (App Authentication)

For multi-tenant app access, JWTs are issued via the Admin API:

```http
Authorization: Bearer <jwt-token>
```

JWT tokens contain:

- `sub`: App ID
- `tenant_id`: App ID (for multi-app routing)

---

## A2A Protocol Endpoints

Hector implements the [A2A (Agent-to-Agent) Protocol](https://a2aproject.org) v0.3.0.

### Agent Card Discovery

#### GET /.well-known/agent-card.json

Returns the default agent's card for the current app.

**Authentication:** Optional (affects visibility filtering)

**Response:**
```json
{
  "name": "Assistant",
  "description": "A helpful AI assistant",
  "url": "http://localhost:8080/agents/assistant",
  "version": "1.0.0",
  "capabilities": {
    "streaming": true,
    "pushNotifications": true
  },
  "skills": [
    {
      "id": "general",
      "name": "General Assistance",
      "description": "Answers general questions"
    }
  ],
  "defaultInputModes": ["text/plain"],
  "defaultOutputModes": ["text/plain"]
}
```

---

### Agent Discovery

#### GET /agents

Returns all visible agents for the current app.

**Authentication:** Optional (affects visibility filtering)  
**Visibility Rules:**

- `public` agents: Always visible
- `internal` agents: Visible only with authentication
- `private` agents: Never visible in discovery

**Response:**
```json
{
  "agents": [
    {
      "name": "Assistant",
      "url": "http://localhost:8080/agents/assistant",
      "description": "A helpful AI assistant",
      "skills": []
    }
  ],
  "total": 1
}
```

---

### Per-Agent Endpoints

#### GET /agents/{name}

Returns the agent card for a specific agent.

**Authentication:** Required for `internal` agents

**Response:** Agent card JSON (same as /.well-known/agent-card.json)

---

#### GET /agents/{name}/.well-known/agent-card.json

Alternative path for agent card discovery.

**Response:** Agent card JSON

---

#### POST /agents/{name}

A2A JSON-RPC 2.0 endpoint for agent interaction.

**Authentication:** Required for `internal` agents  
**Content-Type:** `application/json`

**Supported Methods:**

| Method | Description |
|--------|-------------|
| `message/send` | Send message to agent |
| `message/stream` | Stream message with SSE events |
| `tasks/get` | Get task by ID |
| `tasks/cancel` | Cancel running task |
| `tasks/pushNotification/set` | Configure push notifications |
| `tasks/pushNotification/get` | Get notification config |
| `tasks/resubscribe` | Resubscribe to SSE events |

---

### message/send

Send a message to an agent.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "message/send",
  "params": {
    "message": {
      "role": "user",
      "parts": [
        {"type": "text", "text": "Hello, how can you help me?"}
      ],
      "contextId": "session-123"
    },
    "configuration": {
      "blocking": true
    }
  }
}
```

**Response (blocking=true):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "id": "task-abc123",
    "contextId": "session-123",
    "status": {
      "state": "completed"
    },
    "artifacts": [
      {
        "parts": [
          {"type": "text", "text": "I can help you with..."}
        ]
      }
    ]
  }
}
```

**Response (blocking=false):**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "id": "task-abc123",
    "contextId": "session-123",
    "status": {
      "state": "submitted"
    }
  }
}
```

---

### message/stream

Stream a message with Server-Sent Events.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "message/stream",
  "params": {
    "message": {
      "role": "user",
      "parts": [
        {"type": "text", "text": "Tell me a story"}
      ]
    }
  }
}
```

**Response:** SSE stream with events:

```
event: task
data: {"id": "task-abc123", "status": {"state": "working"}}

event: artifact
data: {"parts": [{"type": "text", "text": "Once upon a time..."}]}

event: task
data: {"id": "task-abc123", "status": {"state": "completed"}}
```

---

### tasks/get

Get task status and result.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tasks/get",
  "params": {
    "id": "task-abc123"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "id": "task-abc123",
    "status": {
      "state": "completed"
    },
    "artifacts": [...]
  }
}
```

**Task States:**

| State | Description |
|-------|-------------|
| `submitted` | Task queued |
| `working` | Task in progress |
| `input-required` | Awaiting user input |
| `completed` | Task finished successfully |
| `failed` | Task failed |
| `canceled` | Task was canceled |

---

### tasks/cancel

Cancel a running task.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tasks/cancel",
  "params": {
    "id": "task-abc123"
  }
}
```

---

### tasks/pushNotification/set

Configure push notifications for a task.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tasks/pushNotification/set",
  "params": {
    "id": "task-abc123",
    "pushNotificationConfig": {
      "url": "https://example.com/webhook",
      "authentication": {
        "type": "bearer",
        "token": "secret-token"
      }
    }
  }
}
```

---

## Admin API

Manage apps, sessions, and queue. **Requires `--auth-secret`**.

```bash
hector serve --auth-secret "admin-secret"
```

All admin requests require:
```http
Authorization: Bearer <admin-secret>
```

---

### App Management

#### GET /admin/apps

List all apps.

**Response:**
```json
{
  "apps": [
    {
      "id": "abc123",
      "name": "customer-support",
      "config_json": "{...}",
      "created_at": "2025-01-20T10:00:00Z",
      "updated_at": "2025-01-20T10:00:00Z"
    }
  ]
}
```

---

#### POST /admin/apps

Create a new app.

**Request:**
```json
{
  "name": "customer-support",
  "config_json": "{\"agents\": {...}}"
}
```

**Response:**
```json
{
  "app": {
    "id": "abc123",
    "name": "customer-support"
  },
  "access_token": "eyJhbGciOiJFUzI1NiIs...",
  "token_type": "bearer",
  "issuer": "hector"
}
```

---

#### GET /admin/apps/{id}

Get app details.

**Response:**
```json
{
  "id": "abc123",
  "name": "customer-support",
  "config_json": "{...}",
  "created_at": "2025-01-20T10:00:00Z"
}
```

---

#### PUT /admin/apps/{id}

Update app configuration.

**Request:**
```json
{
  "name": "updated-name",
  "config_json": "{...}"
}
```

---

#### DELETE /admin/apps/{id}

Delete app and all associated resources (sessions, tasks, vectors).

**Response:** `204 No Content`

---

#### POST /admin/apps/{id}/token

Regenerate JWT for app.

**Response:**
```json
{
  "access_token": "eyJhbGciOiJFUzI1NiIs...",
  "token_type": "bearer",
  "issuer": "hector"
}
```

---

### Session Management

#### GET /admin/sessions

List sessions.

**Query Parameters:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `app_id` | `default` | Filter by app |
| `user_id` | `default` | Filter by user |
| `page_size` | `50` | Results per page (max 100) |
| `page_token` | - | Pagination token |

**Response:**
```json
{
  "sessions": [
    {
      "id": "session-123",
      "app_name": "default",
      "user_id": "default",
      "last_update": "2025-01-20T10:00:00Z",
      "event_count": 42
    }
  ],
  "next_page_token": "abc123"
}
```

---

#### GET /admin/sessions/{id}

Get session with events.

**Query Parameters:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `app_id` | `default` | App ID |
| `limit` | `100` | Max events to return |

**Response:**
```json
{
  "id": "session-123",
  "app_name": "default",
  "user_id": "default",
  "last_update": "2025-01-20T10:00:00Z",
  "event_count": 42,
  "events": [...]
}
```

---

#### DELETE /admin/sessions/{id}

Delete a session.

**Query Parameters:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `app_id` | `default` | App ID |

**Response:** `204 No Content`

---

### Queue Management

#### GET /admin/queue/stats

Get queue statistics.

**Query Parameters:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `app_id` | `default` | App ID |

**Response:**
```json
{
  "pending": 5,
  "running": 2,
  "completed": 100,
  "failed": 3
}
```

---

#### GET /admin/queue/dlq

List dead-letter queue items.

**Query Parameters:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| `app_id` | `default` | App ID |
| `limit` | `50` | Max items (max 100) |

**Response:**
```json
{
  "items": [
    {
      "id": "dlq-123",
      "task_id": "task-abc",
      "error": "timeout",
      "failed_at": "2025-01-20T10:00:00Z",
      "retry_count": 3
    }
  ],
  "count": 1
}
```

---

#### POST /admin/queue/dlq/{id}/requeue

Move item from DLQ back to pending.

**Response:**
```json
{
  "requeued": true
}
```

---

### JWKS Endpoint

#### GET /admin/jwks

Public JWKS for JWT verification.

**Response:**
```json
{
  "keys": [
    {
      "kty": "EC",
      "crv": "P-256",
      "x": "...",
      "y": "...",
      "kid": "hector-key-1"
    }
  ]
}
```

---

## Webhook Endpoints

Trigger agents via HTTP webhooks.

### POST /webhooks/{agent}

Invoke agent via webhook.

**Routing:**
1. Exact match on agent's `trigger.path` configuration
2. Fallback to `/webhooks/{agentName}` convention

**Headers:**

| Header | Description |
|--------|-------------|
| `X-Webhook-Signature` | HMAC signature (configurable header name) |
| `Content-Type` | `application/json` |

**Request:** Raw JSON payload (transformed via `webhook_input.template`)

**Response Modes:**

| Mode | Description |
|------|-------------|
| `sync` | Wait for completion, return result |
| `async` | Return immediately with task ID |
| `callback` | POST result to callback URL |

**Async Response:**
```json
{
  "task_id": "task-abc123",
  "status": "submitted"
}
```

---

## Task Endpoints

### POST /tasks/{taskId}/toolCalls/{callId}/cancel

Cancel a specific tool execution within a task.

**Response:**
```json
{
  "cancelled": true,
  "task_id": "task-abc123",
  "call_id": "tool_xyz789"
}
```

---

## System Endpoints

### GET /health

Server health check.

**Response:**
```json
{
  "status": "ok",
  "auth": {
    "enabled": true,
    "type": "jwt",
    "issuer": "https://auth.example.com",
    "admin": true
  },
  "admin": {
    "enabled": true
  },
  "rag": {
    "stores": {
      "docs": {
        "indexed": 150,
        "total": 200,
        "indexing": true
      }
    },
    "indexing": true
  }
}
```

---

### GET /schema

JSON Schema for configuration validation.

**Response:** Full JSON Schema for `config.yaml`

---

### GET /metrics

Prometheus metrics (requires `--metrics` flag).

**Response:** Prometheus text format

---

## Error Responses

All errors follow JSON-RPC 2.0 or HTTP conventions:

### JSON-RPC Errors

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32600,
    "message": "Invalid Request",
    "data": "Missing required field: message"
  }
}
```

| Code | Message |
|------|---------|
| `-32700` | Parse error |
| `-32600` | Invalid Request |
| `-32601` | Method not found |
| `-32602` | Invalid params |
| `-32603` | Internal error |

### HTTP Errors

```json
{
  "error": "Unauthorized"
}
```

| Status | Description |
|--------|-------------|
| `400` | Bad Request |
| `401` | Unauthorized |
| `403` | Forbidden |
| `404` | Not Found |
| `405` | Method Not Allowed |
| `429` | Rate Limited |
| `500` | Internal Server Error |
| `503` | Service Unavailable |

---

## Rate Limiting

When enabled via configuration, rate limiting returns:

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1642678800
Retry-After: 60
```

---

## Request Headers

| Header | Description |
|--------|-------------|
| `Authorization` | Bearer token (admin secret or JWT) |
| `Content-Type` | `application/json` |
| `X-Session-ID` | Session identifier for rate limiting |
| `X-Request-ID` | Request tracing ID |

---

## SSE Event Types

For streaming responses (`message/stream`):

| Event | Description |
|-------|-------------|
| `task` | Task status update |
| `artifact` | Content artifact (partial or complete) |
| `message` | Agent message |
| `error` | Error occurred |
