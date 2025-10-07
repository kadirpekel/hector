# A2A Protocol Compliance

Hector is a **100% A2A native agent platform** implementing the [Agent-to-Agent (A2A) Protocol specification](https://a2a-protocol.org/latest/specification/). This document describes how Hector implements each section of the A2A specification.

## Table of Contents

- [Overview](#overview)
- [Implementation Summary](#implementation-summary)
- [Detailed Compliance by Specification Section](#detailed-compliance-by-specification-section)
  - [1. Introduction](#1-introduction)
  - [2. Core Concepts](#2-core-concepts)
  - [3. Agent Discovery](#3-agent-discovery)
  - [4. Task Execution](#4-task-execution)
  - [5. Sessions](#5-sessions)
  - [6. Authentication & Authorization](#6-authentication--authorization)
  - [7. Streaming](#7-streaming)
  - [8. Error Handling](#8-error-handling)
- [A2A Protocol Package](#a2a-protocol-package)
- [Related Documentation](#related-documentation)

---

## Overview

The A2A Protocol is an open standard for agent-to-agent communication, enabling:
- **Interoperability**: Any A2A-compliant agent can communicate with any other A2A agent
- **Discovery**: Standardized agent cards for capability advertisement
- **Task Execution**: Uniform request/response model for agent tasks
- **Streaming**: Real-time output via Server-Sent Events (SSE)
- **Sessions**: Multi-turn conversation support

Hector implements the full A2A specification using **HTTP+JSON/REST** transport.

---

## Implementation Summary

| Feature | Status | Spec Section | Implementation |
|---------|--------|--------------|----------------|
| **Agent Discovery** | ✅ Complete | 3.1, 3.2 | `GET /agents`, `GET /agents/{id}` |
| **Agent Cards** | ✅ Complete | 3.2 | Full AgentCard with capabilities |
| **Task Execution** | ✅ Complete | 4.1, 4.2 | `POST /agents/{id}/message/send` |
| **Task Management** | ✅ Complete | 4.3 | `GET /agents/{id}/tasks/{taskId}`, `DELETE /agents/{id}/tasks/{taskId}` |
| **Streaming (SSE)** | ✅ Complete | 7.1, 7.2 | `POST /agents/{id}/message/stream` |
| **Stream Resubscribe** | ✅ Complete | 7.2 | `POST /agents/{id}/tasks/{taskId}/resubscribe` |
| **Sessions** | ✅ Complete | 5.1, 5.2 | `POST /sessions`, `DELETE /sessions/{id}` |
| **Authentication** | ✅ Complete | 6.1 | Bearer Token, API Key |
| **Error Handling** | ✅ Complete | 8.1 | Standardized error responses |

---

## Detailed Compliance by Specification Section

### 1. Introduction

**Spec Reference**: [Section 1](https://a2a-protocol.org/latest/specification/#introduction)

Hector fully adopts the A2A protocol's vision of creating an open ecosystem for AI agents. Our implementation:
- Uses **HTTP+JSON/REST** as the primary transport (Spec 2.1)
- Provides a standalone, reusable A2A protocol package (`pkg/a2a`)
- Supports both native agents and external A2A agents

---

### 2. Core Concepts

**Spec Reference**: [Section 2](https://a2a-protocol.org/latest/specification/#core-concepts)

#### 2.1 Transport Layer

Hector implements **HTTP+JSON/REST** transport:
- **Implementation**: `pkg/a2a/server.go` and `pkg/a2a/client.go`
- **Server**: HTTP/HTTPS endpoints for agent communication
- **Client**: HTTP client for calling external A2A agents
- **Content-Type**: `application/json`

#### 2.2 Data Structures

All core A2A data structures are implemented in `pkg/a2a/protocol.go`:

| A2A Type | Hector Implementation | Spec Section |
|----------|----------------------|--------------|
| `AgentCard` | `a2a.AgentCard` | 3.2 |
| `Task` | `a2a.Task` | 4.1 |
| `Message` | `a2a.Message` | 4.1.1 |
| `Part` | `a2a.Part` | 4.1.2 |
| `Artifact` | `a2a.Artifact` | 4.1.3 |
| `TaskStatus` | `a2a.TaskStatus` | 4.2 |
| `Session` | `a2a.Session` | 5.1 |

---

### 3. Agent Discovery

**Spec Reference**: [Section 3](https://a2a-protocol.org/latest/specification/#agent-discovery)

#### 3.1 Agent Directory

**Endpoint**: `GET /agents`

**Implementation**: `pkg/a2a/server.go:handleListAgents()`

**Functionality**:
- Lists all discoverable agents
- Returns array of `AgentCard` objects
- Respects agent visibility settings (public/internal/private)

**Configuration**: See [CONFIGURATION.md](./CONFIGURATION.md#agent-visibility)

#### 3.2 Agent Card

**Endpoint**: `GET /agents/{agentId}`

**Implementation**: `pkg/agent/agent.go:GetAgentCard()`

**AgentCard Structure** (Spec 3.2):
```go
type AgentCard struct {
    Name         string              // Agent name
    Description  string              // Human-readable description
    Version      string              // Agent version
    URL          string              // Agent base URL
    Capabilities AgentCapabilities   // Agent capabilities
    Tags         []string            // Tags for categorization
    Metadata     map[string]interface{} // Additional metadata
}

type AgentCapabilities struct {
    Streaming              bool     // SSE streaming support
    MultiTurn              bool     // Session/conversation support
    ToolUse                bool     // Tool calling support
    Orchestration          bool     // Multi-agent orchestration
    SupportedInputFormats  []string // e.g., ["text", "image"]
    SupportedOutputFormats []string // e.g., ["text", "image"]
}
```

**Example Configuration**:
```yaml
agents:
  - name: weather_assistant
    description: "Weather information assistant"
    visibility: public  # Discoverable via /agents
    llm: openai-gpt4
    tools: [search, weather]
```

See [CONFIGURATION.md](./CONFIGURATION.md#agents) for full configuration options.

---

### 4. Task Execution

**Spec Reference**: [Section 4](https://a2a-protocol.org/latest/specification/#task-execution)

#### 4.1 Sending Messages

**Endpoint**: `POST /agents/{agentId}/message/send`

**Implementation**: `pkg/a2a/server.go:handleSendMessage()`

**Request Format** (Spec 4.1):
```json
{
  "message": {
    "role": "user",
    "parts": [
      {
        "type": "text",
        "text": "What's the weather in Paris?"
      }
    ]
  },
  "session_id": "optional-session-id"
}
```

**Response Format** (Spec 4.2):
```json
{
  "task": {
    "id": "task-uuid",
    "status": {
      "state": "completed"
    },
    "messages": [
      {
        "role": "assistant",
        "parts": [
          {
            "type": "text",
            "text": "The weather in Paris is..."
          }
        ]
      }
    ],
    "artifacts": [],
    "created_at": "2025-10-07T12:00:00Z",
    "updated_at": "2025-10-07T12:00:05Z"
  }
}
```

**Task States** (Spec 4.2.1):
- `working` - Task is in progress
- `completed` - Task completed successfully
- `failed` - Task failed with error
- `canceled` - Task was canceled

#### 4.2 Task Status

**Endpoint**: `GET /agents/{agentId}/tasks/{taskId}`

**Implementation**: `pkg/a2a/server.go:handleGetTaskStatus()`

Returns the current state of a task including all messages and artifacts.

#### 4.3 Task Cancellation

**Endpoint**: `DELETE /agents/{agentId}/tasks/{taskId}`

**Implementation**: `pkg/a2a/server.go:handleCancelTask()`

Cancels a running task. Returns `204 No Content` on success.

---

### 5. Sessions

**Spec Reference**: [Section 5](https://a2a-protocol.org/latest/specification/#sessions)

Sessions enable multi-turn conversations with context retention.

#### 5.1 Creating Sessions

**Endpoint**: `POST /sessions`

**Implementation**: `pkg/a2a/server.go:handleCreateSession()`

**Request**:
```json
{
  "agent_name": "weather_assistant",
  "metadata": {
    "user_id": "user123"
  }
}
```

**Response**:
```json
{
  "id": "session-uuid",
  "agent_name": "weather_assistant",
  "created_at": "2025-10-07T12:00:00Z",
  "metadata": {
    "user_id": "user123"
  }
}
```

#### 5.2 Deleting Sessions

**Endpoint**: `DELETE /sessions/{sessionId}`

**Implementation**: `pkg/a2a/server.go:handleDeleteSession()`

Deletes a session and its associated conversation history.

**Session Management**:
- Sessions maintain conversation context across multiple messages
- Pass `session_id` in message requests to use session context
- See [AGENTS.md](./AGENTS.md#sessions) for session configuration

---

### 6. Authentication & Authorization

**Spec Reference**: [Section 6](https://a2a-protocol.org/latest/specification/#authentication-authorization)

#### 6.1 Authentication Schemes

Hector supports standard authentication schemes (Spec 6.1):

**Implementation**: `pkg/auth/middleware.go`

**Supported Schemes**:
1. **Bearer Token Authentication**
   ```yaml
   auth:
     jwt:
       enabled: true
       secret: "your-secret-key"
   ```
   Client sends: `Authorization: Bearer <token>`

2. **API Key Authentication**
   ```yaml
   auth:
     api_key:
       enabled: true
       header: "X-API-Key"
       keys:
         - "key123"
   ```
   Client sends: `X-API-Key: <key>`

**Configuration**: See [AUTHENTICATION.md](./AUTHENTICATION.md) for detailed setup.

#### 6.2 Agent Cards with Auth

Agent cards advertise required authentication (Spec 6.2):
```json
{
  "name": "secure_agent",
  "authentication": {
    "required": true,
    "schemes": ["bearer", "apiKey"]
  }
}
```

---

### 7. Streaming

**Spec Reference**: [Section 7](https://a2a-protocol.org/latest/specification/#streaming)

Hector implements **Server-Sent Events (SSE)** for real-time streaming per A2A spec.

#### 7.1 Streaming Messages

**Endpoint**: `POST /agents/{agentId}/message/stream`

**Implementation**: `pkg/a2a/server.go:handleMessageStream()`

**Request** (same as non-streaming):
```json
{
  "message": {
    "role": "user",
    "parts": [{"type": "text", "text": "Tell me a story"}]
  }
}
```

**Response**: SSE stream with `Content-Type: text/event-stream`

**Event Types** (Spec 7.2):

1. **Status Events**:
   ```
   event: status
   data: {"task_id": "task-123", "status": {"state": "working"}}
   ```

2. **Message Events** (incremental content):
   ```
   event: message
   data: {"task_id": "task-123", "message": {"role": "assistant", "parts": [...]}}
   ```

3. **Artifact Events**:
   ```
   event: artifact
   data: {"task_id": "task-123", "artifact": {"type": "image", "parts": [...]}}
   ```

#### 7.2 Stream Resubscribe

**Endpoint**: `POST /agents/{agentId}/tasks/{taskId}/resubscribe`

**Implementation**: `pkg/a2a/server.go:handleTaskResubscribe()`

**Functionality**:
- Resume streaming for existing tasks
- Useful for reconnection after network interruption
- Continues from last known state

**Request**:
```json
{
  "last_event_id": "optional-event-id"
}
```

**Client Implementation**:
- **Server**: `pkg/a2a/server.go:sendSSEEvent()`
- **Client**: `pkg/a2a/client.go:SendMessageStreaming()`

**Usage**: See [CLI_GUIDE.md](./CLI_GUIDE.md#streaming-mode) for CLI streaming examples.

---

### 8. Error Handling

**Spec Reference**: [Section 8](https://a2a-protocol.org/latest/specification/#error-handling)

#### 8.1 Error Response Format

**Implementation**: `pkg/a2a/protocol.go:TaskError`

All errors follow the A2A standard format (Spec 8.1):

```json
{
  "error": {
    "code": "AGENT_NOT_FOUND",
    "message": "Agent 'unknown_agent' not found",
    "details": {
      "agent_id": "unknown_agent"
    }
  }
}
```

**HTTP Status Codes** (Spec 8.2):
| Status | Meaning | A2A Code |
|--------|---------|----------|
| 400 | Bad Request | `INVALID_REQUEST` |
| 401 | Unauthorized | `AUTH_REQUIRED` |
| 403 | Forbidden | `AUTH_FORBIDDEN` |
| 404 | Not Found | `AGENT_NOT_FOUND`, `TASK_NOT_FOUND` |
| 500 | Internal Error | `INTERNAL_ERROR` |
| 503 | Service Unavailable | `SERVICE_UNAVAILABLE` |

**Error Codes**:
```go
const (
    ErrorCodeInvalidRequest    = "INVALID_REQUEST"
    ErrorCodeAgentNotFound     = "AGENT_NOT_FOUND"
    ErrorCodeTaskNotFound      = "TASK_NOT_FOUND"
    ErrorCodeAuthRequired      = "AUTH_REQUIRED"
    ErrorCodeAuthForbidden     = "AUTH_FORBIDDEN"
    ErrorCodeInternalError     = "INTERNAL_ERROR"
    ErrorCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)
```

---

## A2A Protocol Package

### Standalone Implementation

The `pkg/a2a` package is a **standalone, reusable A2A protocol implementation**:

**Key Features**:
- ✅ **No internal dependencies**: Only uses Go standard library + `github.com/google/uuid`
- ✅ **100% A2A compliant**: Implements full specification
- ✅ **Reusable**: Can be extracted and used in other projects
- ✅ **Well-tested**: Comprehensive unit tests

**Package Structure**:
```
pkg/a2a/
├── protocol.go       # Core A2A data structures
├── server.go         # A2A HTTP server implementation
├── client.go         # A2A HTTP client implementation
└── protocol_test.go  # Protocol tests
```

**Usage Example**:
```go
import "github.com/kadirpekel/hector/pkg/a2a"

// Create A2A client
client := a2a.NewClient(nil)

// Discover agents
agents, err := client.DiscoverAgents(ctx, "http://localhost:8080")

// Send message
message := a2a.CreateTextMessage(a2a.MessageRoleUser, "Hello")
task, err := client.SendTextMessage(ctx, agentURL, message)
```

### Agent Interface

All Hector agents implement the `a2a.Agent` interface:

```go
type Agent interface {
    // GetAgentCard returns the agent's card with capabilities
    GetAgentCard() AgentCard
    
    // ExecuteTask executes a task synchronously
    ExecuteTask(ctx context.Context, task Task) (Task, error)
    
    // ExecuteTaskStreaming executes a task with real-time streaming
    ExecuteTaskStreaming(ctx context.Context, task Task) (<-chan StreamEvent, error)
}
```

**Implementation**: See `pkg/agent/agent.go` for the native agent implementation.

---

## Related Documentation

- **[AGENTS.md](./AGENTS.md)**: Comprehensive guide to agent configuration and capabilities
- **[CONFIGURATION.md](./CONFIGURATION.md)**: Full configuration reference including A2A settings
- **[AUTHENTICATION.md](./AUTHENTICATION.md)**: Authentication and authorization setup
- **[EXTERNAL_AGENTS.md](./EXTERNAL_AGENTS.md)**: Connecting to external A2A agents
- **[CLI_GUIDE.md](./CLI_GUIDE.md)**: Using the CLI to interact with A2A agents
- **[API_REFERENCE.md](./API_REFERENCE.md)**: Complete API endpoint documentation
- **[QUICK_START.md](./QUICK_START.md)**: Getting started with Hector

---

## Compliance Verification

To verify Hector's A2A compliance:

1. **Run tests**:
   ```bash
   make test
   ```

2. **Check A2A endpoints**:
   ```bash
   # Start server
   hector serve configs/weather-agent.yaml
   
   # Test discovery
   curl http://localhost:8080/agents
   
   # Test agent card
   curl http://localhost:8080/agents/weather_assistant
   
   # Test task execution
   curl -X POST http://localhost:8080/agents/weather_assistant/message/send \
     -H "Content-Type: application/json" \
     -d '{"message":{"role":"user","parts":[{"type":"text","text":"Hello"}]}}'
   ```

3. **Test streaming**:
   ```bash
   curl -X POST http://localhost:8080/agents/weather_assistant/message/stream \
     -H "Content-Type: application/json" \
     -H "Accept: text/event-stream" \
     -d '{"message":{"role":"user","parts":[{"type":"text","text":"Tell me a story"}]}}'
   ```

---

## Contributing

When contributing to Hector's A2A implementation:

1. **Maintain compliance**: All changes must preserve A2A spec compliance
2. **Update tests**: Add tests for new A2A features
3. **Keep pkg/a2a standalone**: No internal Hector dependencies
4. **Update documentation**: Document spec section references

See [CONTRIBUTING.md](./CONTRIBUTING.md) for contribution guidelines.

---

## Specification Version

This document describes Hector's implementation of:
- **A2A Protocol Specification**: [Latest](https://a2a-protocol.org/latest/specification/)
- **Last Updated**: October 2025
- **Hector Version**: 1.0.0+

---

## License

Hector is open source under the MIT License. See [LICENSE.md](../LICENSE.md).

The A2A Protocol specification is maintained by the A2A Project and is also open source.