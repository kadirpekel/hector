# Hector A2A API Reference

**A2A Protocol Implementation - HTTP/WebSocket API**

This document provides a complete reference for Hector's A2A (Agent-to-Agent) protocol implementation. Hector follows the [A2A specification](https://a2a-protocol.org) to enable standardized agent interoperability.

---

## Table of Contents

1. [Introduction](#introduction)
2. [Base URL & Authentication](#base-url--authentication)
3. [Agent Discovery](#agent-discovery)
4. [Task Execution](#task-execution)
5. [Session Management](#session-management)
6. [Streaming](#streaming)
7. [Schemas](#schemas)
8. [Error Handling](#error-handling)
9. [Examples](#examples)

---

## Introduction

### A2A Protocol Overview

The A2A (Agent-to-Agent) protocol is an open standard for agent interoperability. It defines:

- **Agent Cards** - Capability discovery and metadata
- **Task Execution** - Standardized request/response model
- **Sessions** - Multi-turn conversation state management
- **Streaming** - Real-time output via WebSocket

### Hector's A2A Implementation

Hector implements the A2A specification with the following features:

✅ **Full A2A Core Compliance** - Discovery, task execution, status  
✅ **Session Management** - Stateful multi-turn conversations  
✅ **WebSocket Streaming** - Real-time chunked output  
✅ **JWT Authentication** - Optional OAuth2/OIDC integration  
✅ **Visibility Control** - Public, internal, and private agents  

### Protocol Version

- **A2A Version**: 1.0 (compatible)
- **Hector API Version**: 1.0

---

## Base URL & Authentication

### Base URL

```
http://localhost:8080
```

For production deployments, use your server's public URL:
```
https://agents.example.com
```

### Authentication

Hector supports **optional JWT-based authentication**:

**Without Authentication** (default):
```bash
curl http://localhost:8080/agents
```

**With Authentication** (when enabled):
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:8080/agents
```

See [Authentication Guide](AUTHENTICATION.md) for setup details.

### Protected Endpoints

When authentication is enabled:
- ✅ `GET /agents` - **Public** (no auth required)
- 🔒 `GET /agents/{agentId}` - **Protected** (auth required)
- 🔒 `POST /agents/{agentId}/tasks` - **Protected**
- 🔒 All `/sessions/*` endpoints - **Protected**

---

## Agent Discovery

### List All Agents

**Endpoint**: `GET /agents`

Returns a directory of all publicly discoverable agents (visibility="public").

**Request**:
```bash
curl http://localhost:8080/agents
```

**Response** (200 OK):
```json
{
  "agents": [
    {
      "agentId": "competitor_analyst",
      "name": "Competitor Analysis Agent",
      "description": "Analyzes market competitors and provides strategic insights",
      "version": "1.0.0",
      "capabilities": ["text_generation", "reasoning", "analysis"],
      "endpoints": {
        "task": "http://localhost:8080/agents/competitor_analyst/tasks",
        "stream": "http://localhost:8080/agents/competitor_analyst/stream",
        "status": "http://localhost:8080/agents/competitor_analyst/tasks/{taskId}"
      },
      "inputTypes": ["text/plain", "application/json"],
      "outputTypes": ["text/plain", "application/json"],
      "metadata": {
        "llm": "gpt-4o",
        "temperature": "0.7"
      }
    }
  ],
  "total": 1
}
```

**Notes**:
- Only agents with `visibility: "public"` are listed
- Internal and private agents are hidden from discovery
- No authentication required for this endpoint

---

### Get Agent Card

**Endpoint**: `GET /agents/{agentId}`

Returns detailed information about a specific agent (the agent's "business card").

**Request**:
```bash
curl http://localhost:8080/agents/competitor_analyst
```

**Response** (200 OK):
```json
{
  "agentId": "competitor_analyst",
  "name": "Competitor Analysis Agent",
  "description": "Analyzes market competitors and provides strategic insights",
  "version": "1.0.0",
  "capabilities": [
    "text_generation",
    "reasoning",
    "analysis",
    "web_search"
  ],
  "endpoints": {
    "task": "http://localhost:8080/agents/competitor_analyst/tasks",
    "stream": "http://localhost:8080/agents/competitor_analyst/stream",
    "status": "http://localhost:8080/agents/competitor_analyst/tasks/{taskId}"
  },
  "inputTypes": ["text/plain", "application/json"],
  "outputTypes": ["text/plain", "application/json"],
  "auth": {
    "type": "bearer",
    "schemes": ["Bearer"]
  },
  "metadata": {
    "llm": "gpt-4o",
    "temperature": "0.7",
    "reasoning": "chain-of-thought"
  }
}
```

**Errors**:
- `404 Not Found` - Agent does not exist or is private
- `401 Unauthorized` - Authentication required but not provided

---

## Task Execution

### Execute Task

**Endpoint**: `POST /agents/{agentId}/tasks`

Executes a task on the specified agent.

**Request**:
```bash
curl -X POST http://localhost:8080/agents/competitor_analyst/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "taskId": "task-123",
    "input": {
      "type": "text/plain",
      "content": "Analyze the top 3 AI agent frameworks"
    },
    "parameters": {
      "temperature": 0.7,
      "max_tokens": 2000
    }
  }'
```

**Request Body** (TaskRequest):
```json
{
  "taskId": "task-123",
  "input": {
    "type": "text/plain",
    "content": "Analyze the top 3 AI agent frameworks"
  },
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 2000
  },
  "context": {
    "sessionId": "session-456",
    "userId": "user-789",
    "metadata": {
      "source": "web-app"
    }
  }
}
```

**Response** (200 OK):
```json
{
  "taskId": "task-123",
  "status": "completed",
  "output": {
    "type": "text/plain",
    "content": "Based on my analysis of the top AI agent frameworks:\n\n1. **LangChain**..."
  },
  "metadata": {
    "tokens_used": 1247,
    "execution_time_ms": 3421
  },
  "startedAt": "2025-10-05T10:30:00Z",
  "endedAt": "2025-10-05T10:30:03Z"
}
```

**For Asynchronous Tasks** (202 Accepted):
```json
{
  "taskId": "task-123",
  "status": "running",
  "startedAt": "2025-10-05T10:30:00Z"
}
```

**Errors**:
- `400 Bad Request` - Invalid request body
- `404 Not Found` - Agent does not exist
- `401 Unauthorized` - Authentication required

**Notes**:
- `taskId` is optional; server generates one if not provided
- If task is long-running, check status using GET endpoint

---

### Get Task Status

**Endpoint**: `GET /agents/{agentId}/tasks/{taskId}`

Retrieves the current status and result of a task.

**Request**:
```bash
curl http://localhost:8080/agents/competitor_analyst/tasks/task-123
```

**Response** (200 OK):

**Completed Task**:
```json
{
  "taskId": "task-123",
  "status": "completed",
  "output": {
    "type": "text/plain",
    "content": "Analysis complete..."
  },
  "startedAt": "2025-10-05T10:30:00Z",
  "endedAt": "2025-10-05T10:30:03Z"
}
```

**Running Task**:
```json
{
  "taskId": "task-123",
  "status": "running",
  "startedAt": "2025-10-05T10:30:00Z"
}
```

**Failed Task**:
```json
{
  "taskId": "task-123",
  "status": "failed",
  "error": {
    "code": "EXECUTION_ERROR",
    "message": "LLM service unavailable",
    "details": "Connection timeout after 30s"
  },
  "startedAt": "2025-10-05T10:30:00Z",
  "endedAt": "2025-10-05T10:30:30Z"
}
```

**Task Status Values**:
- `pending` - Task queued but not started
- `running` - Task currently executing
- `completed` - Task finished successfully
- `failed` - Task encountered an error
- `cancelled` - Task was cancelled

**Errors**:
- `404 Not Found` - Task or agent does not exist

---

## Session Management

Sessions enable multi-turn conversations with agents, maintaining state across multiple tasks.

### Create Session

**Endpoint**: `POST /sessions`

Creates a new session for multi-turn conversations.

**Request**:
```bash
curl -X POST http://localhost:8080/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "agentId": "competitor_analyst",
    "metadata": {
      "user": "john@example.com",
      "purpose": "market-research"
    }
  }'
```

**Response** (201 Created):
```json
{
  "sessionId": "sess-abc123",
  "agentId": "competitor_analyst",
  "createdAt": "2025-10-05T10:30:00Z",
  "lastActivityAt": "2025-10-05T10:30:00Z",
  "state": {},
  "metadata": {
    "user": "john@example.com",
    "purpose": "market-research"
  }
}
```

**Errors**:
- `400 Bad Request` - Invalid request body
- `404 Not Found` - Agent does not exist

---

### Execute Task in Session

**Endpoint**: `POST /sessions/{sessionId}/tasks`

Executes a task within a session context, preserving conversation history.

**Request**:
```bash
curl -X POST http://localhost:8080/sessions/sess-abc123/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "taskId": "task-456",
    "input": {
      "type": "text/plain",
      "content": "What about their pricing models?"
    }
  }'
```

**Response** (200 OK):
```json
{
  "taskId": "task-456",
  "status": "completed",
  "output": {
    "type": "text/plain",
    "content": "Based on our previous discussion about the top frameworks, here's a pricing comparison..."
  },
  "startedAt": "2025-10-05T10:31:00Z",
  "endedAt": "2025-10-05T10:31:02Z"
}
```

**Notes**:
- Agent has access to all previous tasks in the session
- Session state is automatically maintained
- Use this for chatbot-like interactions

---

### Get Session Info

**Endpoint**: `GET /sessions/{sessionId}`

Retrieves session metadata and state.

**Request**:
```bash
curl http://localhost:8080/sessions/sess-abc123
```

**Response** (200 OK):
```json
{
  "sessionId": "sess-abc123",
  "agentId": "competitor_analyst",
  "createdAt": "2025-10-05T10:30:00Z",
  "lastActivityAt": "2025-10-05T10:31:02Z",
  "state": {
    "message_count": 2,
    "topics": ["AI frameworks", "pricing"]
  },
  "metadata": {
    "user": "john@example.com",
    "purpose": "market-research"
  }
}
```

**Errors**:
- `404 Not Found` - Session does not exist

---

### List Sessions

**Endpoint**: `GET /sessions`

Lists all sessions, optionally filtered by agent.

**Request**:
```bash
# All sessions
curl http://localhost:8080/sessions

# Sessions for specific agent
curl http://localhost:8080/sessions?agent_id=competitor_analyst
```

**Response** (200 OK):
```json
{
  "sessions": [
    {
      "sessionId": "sess-abc123",
      "agentId": "competitor_analyst",
      "createdAt": "2025-10-05T10:30:00Z",
      "lastActivityAt": "2025-10-05T10:31:02Z"
    }
  ],
  "total": 1
}
```

---

### Delete Session

**Endpoint**: `DELETE /sessions/{sessionId}`

Ends a session and clears its state.

**Request**:
```bash
curl -X DELETE http://localhost:8080/sessions/sess-abc123
```

**Response** (204 No Content):
```
(empty body)
```

**Errors**:
- `404 Not Found` - Session does not exist

---

## Streaming

Real-time streaming of task output via WebSocket.

### Stream Task Execution

**Endpoint**: `WS /agents/{agentId}/stream`

WebSocket endpoint for streaming task execution.

**Connection**:
```javascript
const ws = new WebSocket('ws://localhost:8080/agents/competitor_analyst/stream');

ws.onopen = () => {
  ws.send(JSON.stringify({
    taskId: 'task-789',
    input: {
      type: 'text/plain',
      content: 'Analyze LangChain vs CrewAI'
    }
  }));
};

ws.onmessage = (event) => {
  const chunk = JSON.parse(event.data);
  console.log(chunk);
};
```

**Stream Chunks**:

**Text Chunk**:
```json
{
  "taskId": "task-789",
  "chunkType": "text",
  "content": "LangChain is a framework...",
  "timestamp": "2025-10-05T10:30:01.234Z",
  "final": false
}
```

**Final Chunk**:
```json
{
  "taskId": "task-789",
  "chunkType": "done",
  "content": null,
  "timestamp": "2025-10-05T10:30:05.678Z",
  "final": true
}
```

**Error Chunk**:
```json
{
  "taskId": "task-789",
  "chunkType": "error",
  "content": "Execution failed: timeout",
  "timestamp": "2025-10-05T10:30:30.000Z",
  "final": true
}
```

**Chunk Types**:
- `text` - Text output chunk
- `data` - Structured data chunk
- `metadata` - Additional metadata
- `error` - Error message
- `done` - Final chunk (end of stream)

**Notes**:
- Chunks arrive as they're generated (real-time)
- `final: true` indicates the last chunk
- Connection closes automatically after final chunk

---

## Schemas

### AgentCard

Agent capability card for discovery.

```typescript
{
  agentId: string;           // Unique agent identifier
  name: string;              // Human-readable name
  description: string;       // Agent description
  version?: string;          // Agent version (optional)
  capabilities: string[];    // List of capabilities
  endpoints: {
    task: string;            // POST endpoint for tasks
    stream?: string;         // WebSocket endpoint (optional)
    status?: string;         // GET endpoint for status (optional)
  };
  inputTypes: string[];      // Accepted MIME types
  outputTypes: string[];     // Produced MIME types
  auth?: {                   // Authentication requirements (optional)
    type: string;            // "bearer", "apiKey", "oauth2", "mtls"
    schemes?: string[];      // Supported schemes
    tokenUrl?: string;       // OAuth2 token URL
  };
  metadata?: Record<string, string>; // Additional metadata
}
```

---

### TaskRequest

Request to execute a task.

```typescript
{
  taskId?: string;           // Optional unique task ID (generated if not provided)
  input: {
    type: string;            // MIME type: "text/plain", "application/json"
    content: any;            // Input content (string, object, etc.)
  };
  parameters?: Record<string, any>; // Optional execution parameters
  context?: {
    sessionId?: string;      // Session ID for multi-turn
    conversationId?: string; // Conversation ID
    userId?: string;         // User identifier
    metadata?: Record<string, string>; // Additional context
  };
}
```

---

### TaskResponse

Response from task execution.

```typescript
{
  taskId: string;            // Task identifier
  status: TaskStatus;        // "pending" | "running" | "completed" | "failed" | "cancelled"
  output?: {
    type: string;            // MIME type of output
    content: any;            // Output content
  };
  error?: {
    code: string;            // Error code
    message: string;         // Error message
    details?: string;        // Additional error details
  };
  metadata?: Record<string, any>; // Execution metadata
  startedAt?: string;        // ISO 8601 timestamp
  endedAt?: string;          // ISO 8601 timestamp
}
```

---

### Session

Multi-turn conversation session.

```typescript
{
  sessionId: string;         // Unique session identifier
  agentId: string;           // Associated agent
  createdAt: string;         // ISO 8601 timestamp
  lastActivityAt: string;    // ISO 8601 timestamp
  state?: Record<string, any>; // Session state
  metadata?: Record<string, string>; // Session metadata
}
```

---

### StreamChunk

Real-time streaming chunk.

```typescript
{
  taskId: string;            // Task identifier
  chunkType: ChunkType;      // "text" | "data" | "error" | "metadata" | "done"
  content: any;              // Chunk content
  timestamp: string;         // ISO 8601 timestamp
  final: boolean;            // True for last chunk
}
```

---

## Error Handling

### HTTP Status Codes

| Code | Meaning | Description |
|------|---------|-------------|
| 200 | OK | Request succeeded |
| 201 | Created | Resource created (session) |
| 202 | Accepted | Task accepted (async) |
| 204 | No Content | Resource deleted |
| 400 | Bad Request | Invalid request body or parameters |
| 401 | Unauthorized | Authentication required |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Agent, task, or session not found |
| 405 | Method Not Allowed | HTTP method not supported |
| 500 | Internal Server Error | Server error |

---

### Error Response Format

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": "Additional error details or stack trace"
  }
}
```

### Common Error Codes

| Code | Description |
|------|-------------|
| `AGENT_NOT_FOUND` | Agent does not exist or is private |
| `TASK_NOT_FOUND` | Task ID not found |
| `SESSION_NOT_FOUND` | Session ID not found |
| `INVALID_INPUT` | Invalid request body or parameters |
| `EXECUTION_ERROR` | Task execution failed |
| `AUTH_REQUIRED` | Authentication token required |
| `INVALID_TOKEN` | JWT token invalid or expired |
| `INSUFFICIENT_PERMISSIONS` | User lacks required permissions |

---

## Examples

### Python Client

```python
import requests
import json

# Base URL
base_url = "http://localhost:8080"

# 1. Discover agents
response = requests.get(f"{base_url}/agents")
agents = response.json()["agents"]
print(f"Found {len(agents)} agents")

# 2. Get agent card
agent_id = "competitor_analyst"
response = requests.get(f"{base_url}/agents/{agent_id}")
card = response.json()
print(f"Agent: {card['name']}")

# 3. Execute task
task_request = {
    "input": {
        "type": "text/plain",
        "content": "Analyze the AI agent market"
    }
}
response = requests.post(
    f"{base_url}/agents/{agent_id}/tasks",
    json=task_request
)
result = response.json()
print(f"Status: {result['status']}")
print(f"Output: {result['output']['content']}")

# 4. Create session
session_request = {
    "agentId": agent_id,
    "metadata": {"user": "john@example.com"}
}
response = requests.post(f"{base_url}/sessions", json=session_request)
session = response.json()
session_id = session["sessionId"]

# 5. Execute in session
task_request = {
    "input": {
        "type": "text/plain",
        "content": "What are the key players?"
    }
}
response = requests.post(
    f"{base_url}/sessions/{session_id}/tasks",
    json=task_request
)
result = response.json()
print(f"Response: {result['output']['content']}")

# 6. Clean up
requests.delete(f"{base_url}/sessions/{session_id}")
```

---

### JavaScript Client

```javascript
// Base URL
const baseUrl = 'http://localhost:8080';

// 1. Discover agents
async function discoverAgents() {
  const response = await fetch(`${baseUrl}/agents`);
  const data = await response.json();
  console.log(`Found ${data.total} agents`);
  return data.agents;
}

// 2. Execute task
async function executeTask(agentId, prompt) {
  const response = await fetch(`${baseUrl}/agents/${agentId}/tasks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      input: {
        type: 'text/plain',
        content: prompt
      }
    })
  });
  
  const result = await response.json();
  return result;
}

// 3. Stream task execution
function streamTask(agentId, prompt) {
  const ws = new WebSocket(`ws://localhost:8080/agents/${agentId}/stream`);
  
  ws.onopen = () => {
    ws.send(JSON.stringify({
      input: {
        type: 'text/plain',
        content: prompt
      }
    }));
  };
  
  ws.onmessage = (event) => {
    const chunk = JSON.parse(event.data);
    
    if (chunk.chunkType === 'text') {
      process.stdout.write(chunk.content);
    } else if (chunk.chunkType === 'done') {
      console.log('\n\nStream complete');
      ws.close();
    }
  };
  
  ws.onerror = (error) => {
    console.error('WebSocket error:', error);
  };
}

// 4. Session management
async function chatSession(agentId) {
  // Create session
  const sessionRes = await fetch(`${baseUrl}/sessions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ agentId })
  });
  const session = await sessionRes.json();
  
  // Execute multiple tasks
  const tasks = [
    "Analyze AI frameworks",
    "What about their pricing?",
    "Which is best for enterprise?"
  ];
  
  for (const prompt of tasks) {
    const response = await fetch(
      `${baseUrl}/sessions/${session.sessionId}/tasks`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          input: { type: 'text/plain', content: prompt }
        })
      }
    );
    const result = await response.json();
    console.log(`Q: ${prompt}`);
    console.log(`A: ${result.output.content}\n`);
  }
  
  // Clean up
  await fetch(`${baseUrl}/sessions/${session.sessionId}`, {
    method: 'DELETE'
  });
}

// Usage
(async () => {
  const agents = await discoverAgents();
  const result = await executeTask('competitor_analyst', 'Analyze market');
  console.log(result.output.content);
})();
```

---

### cURL Examples

**Discover agents**:
```bash
curl http://localhost:8080/agents
```

**Get agent card**:
```bash
curl http://localhost:8080/agents/competitor_analyst
```

**Execute task**:
```bash
curl -X POST http://localhost:8080/agents/competitor_analyst/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "input": {
      "type": "text/plain",
      "content": "Analyze the AI agent market"
    }
  }'
```

**Create session**:
```bash
curl -X POST http://localhost:8080/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "agentId": "competitor_analyst",
    "metadata": {"user": "john@example.com"}
  }'
```

**Execute in session**:
```bash
curl -X POST http://localhost:8080/sessions/sess-abc123/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "input": {
      "type": "text/plain",
      "content": "What are the key players?"
    }
  }'
```

**With authentication**:
```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:8080/agents/competitor_analyst/tasks \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"input": {"type": "text/plain", "content": "Analyze market"}}'
```

---

## Additional Resources

- [A2A Protocol Specification](https://a2a-protocol.org)
- [Hector Architecture](ARCHITECTURE.md)
- [Authentication Guide](AUTHENTICATION.md)
- [Configuration Reference](CONFIGURATION.md)
- [CLI Guide](CLI_GUIDE.md)

---

## Protocol Compliance

Hector implements:
- ✅ **A2A Core**: Agent discovery, task execution, status
- ✅ **Sessions**: Multi-turn conversation state management
- ✅ **Streaming**: Real-time WebSocket output
- ✅ **Authentication**: Optional JWT/OAuth2
- ✅ **Visibility**: Public, internal, private agents

For the complete A2A specification, see [a2a-protocol.org](https://a2a-protocol.org).
