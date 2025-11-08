---
title: Tasks
description: Asynchronous task execution and management with the A2A protocol
---

# Tasks

Tasks are the A2A protocol's mechanism for asynchronous, long-running operations. They provide a way to submit work to an agent and track its progress without maintaining an open connection.

## A2A Protocol Compliance

Hector's task implementation is **fully compliant** with the [A2A (Agent-to-Agent) Protocol specification](https://a2a-protocol.org/latest/specification/). This ensures interoperability with other A2A-compliant systems.

### Core Methods (Section 11.1.2 - MUST implement)

✅ **`message/send`** - Create tasks (blocking or non-blocking)  
✅ **`tasks/get`** - Retrieve task status and results  
✅ **`tasks/cancel`** - Cancel running tasks

### Optional Methods (Section 11.1.3 - MAY implement)

✅ **`message/stream`** - Stream task updates in real-time  
✅ **`tasks/list`** - List tasks with filtering and pagination  
✅ **`tasks/resubscribe`** - Resume streaming for existing tasks

### REST Endpoints (Section 3.5.3 - HTTP+JSON/REST transport)

All task operations are available via REST API:

```bash
# Create task
POST /v1/agents/{agent}/message:send

# Get task details
GET /v1/agents/{agent}/tasks/{taskID}

# Cancel task
POST /v1/agents/{agent}/tasks/{taskID}:cancel

# List tasks
GET /v1/agents/{agent}/tasks
```

### Supported Task States (Section 6.3)

All A2A-specified task states are supported:

| State | Type | Description |
|-------|------|-------------|
| **SUBMITTED** | Initial | Task created, waiting to start |
| **WORKING** | Active | Agent actively processing |
| **COMPLETED** | Terminal | Success |
| **FAILED** | Terminal | Error occurred |
| **CANCELLED** | Terminal | User cancelled |
| **INPUT_REQUIRED** | Interrupted | Needs user input (e.g., tool approval) |
| **REJECTED** | Terminal | Agent rejected task |
| **AUTH_REQUIRED** | Special | Needs authentication |

## Overview

### What is a Task?

A **Task** is a unit of work submitted to an agent that can be:
- **Tracked** - Query status and progress at any time
- **Asynchronous** - Submit and disconnect, check back later
- **Persistent** - Survives server restarts (with SQL backend)
- **Cancellable** - Stop running tasks when needed

### When to Use Tasks

| Use Case | Recommendation |
|----------|---------------|
| Quick queries (<30s) | Use `call` or `chat` commands (blocking) |
| Long operations (minutes/hours) | Use tasks (non-blocking) |
| Background processing | Use tasks |
| Need progress tracking | Use tasks |
| Multi-step workflows | Use tasks |

---

## Task Lifecycle

```
SUBMITTED → WORKING → COMPLETED
                   ↓
                FAILED
                   ↓
             CANCELLED
```

### Task States

| State | Description | Terminal |
|-------|-------------|----------|
| `SUBMITTED` | Task created, waiting to start | No |
| `WORKING` | Task is actively processing | No |
| `COMPLETED` | Task finished successfully | Yes |
| `FAILED` | Task encountered an error | Yes |
| `CANCELLED` | Task was cancelled by user | Yes |
| `INPUT_REQUIRED` | Task paused, waiting for user input (e.g., tool approval) | No |
| `REJECTED` | Task was rejected (e.g., quota exceeded) | Yes |

**Terminal states** mean the task won't change anymore and can be archived.

---

## Configuration

### In-Memory Tasks (Default)

Tasks work out-of-the-box with no configuration. They're stored in memory and lost on server restart.

```yaml
agents:
  assistant:
    llm: gpt-4o
    # No task configuration needed - uses in-memory storage
```

**Characteristics:**
- ✅ Zero configuration
- ✅ Fast
- ❌ Lost on restart
- ❌ Not suitable for production

### SQL-Backed Tasks (Production)

For production use, configure SQL persistence:

```yaml
agents:
  production-agent:
    llm: gpt-4o
    task:
      backend: sql
      worker_pool: 100  # Max concurrent tasks
      sql:
        driver: sqlite  # or: postgres, mysql
        database: ./data/tasks.db
        max_conns: 10
        max_idle: 2

llms:
  gpt-4o:
    type: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}
```

**Supported Databases:**
- **SQLite** - Simple file-based storage
- **PostgreSQL** - Production-grade, multi-instance
- **MySQL** - Alternative production option

**PostgreSQL Example:**
```yaml
task:
  backend: sql
  sql:
    driver: postgres
    database: postgres://user:password@localhost:5432/hector_tasks?sslmode=disable
    max_conns: 25
    max_idle: 5
```

---

## Usage

### Server Mode

Start a server with task support:

```bash
# With in-memory tasks (development)
hector serve --config config.yaml

# Server output shows:
# Hector server listening on :8080
# Registered agents: assistant
# Task service: in-memory
```

### Submitting Tasks

Tasks are submitted via the A2A protocol's `blocking` configuration:

**Blocking (default):**
```bash
# Waits for completion, returns result
curl -X POST http://localhost:8080/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {"parts": [{"text": "Long running task"}]},
    "configuration": {"blocking": true}
  }'
```

**Non-blocking (task mode):**
```bash
# Returns task ID immediately
curl -X POST http://localhost:8080/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {"parts": [{"text": "Long running task"}]},
    "configuration": {"blocking": false}
  }'

# Response:
# {
#   "task": {
#     "id": "task-abc123",
#     "context_id": "ctx-xyz",
#     "status": {"state": "TASK_STATE_SUBMITTED"}
#   }
# }
```

### Checking Task Status

Use the task commands to monitor progress:

```bash
# Get task details
hector task get assistant task-abc123 --url http://localhost:8080

# Output:
# Task ID: task-abc123
# Status: TASK_STATE_WORKING
# Context ID: ctx-xyz
# Created: 2024-01-15 10:30:00
```

### Cancelling Tasks

Stop a running task:

```bash
hector task cancel assistant task-abc123 --url http://localhost:8080

# Output:
# ✅ Task cancelled successfully
#
# Task ID: task-abc123
# Status: TASK_STATE_CANCELLED
```

---

## CLI Commands

### `hector task get`

Retrieve task details by ID.

**Usage:**
```bash
hector task get <agent> <task-id> [flags]
```

**Flags:**
- `--url URL` - Agent service URL (required for client mode)
- `--token TOKEN` - Authentication token
- `--config FILE` - Configuration file (for local mode)

**Examples:**

```bash
# Client mode - query remote service
hector task get assistant task-abc123 --url http://localhost:8080

# Local mode - query from config
hector task get assistant task-abc123 --config config.yaml

# With authentication
hector task get assistant task-abc123 \
  --url https://prod:8080 \
  --token "eyJ..."
```

**Output:**
```
Task ID: task-abc123
Status: TASK_STATE_COMPLETED
Context ID: ctx-xyz
Created: 2024-01-15 10:30:00
Updated: 2024-01-15 10:35:00

History:
  [USER] Long running analysis task
  [ASSISTANT] Analysis complete. Found 3 issues...

Artifacts:
  - report.pdf (application/pdf, 2.5 MB)
```

### `hector task cancel`

Cancel a running or pending task.

**Usage:**
```bash
hector task cancel <agent> <task-id> [flags]
```

**Flags:**
- Same as `task get`

**Examples:**

```bash
# Cancel task
hector task cancel assistant task-abc123 --url http://localhost:8080

# Cancel with config
hector task cancel assistant task-abc123 --config config.yaml
```

---

## Modes and Limitations

### Supported Modes

| Mode | Task Get | Task Cancel | Notes |
|------|----------|-------------|-------|
| **Server Mode** | ✅ Via API | ✅ Via API | Full task support |
| **Client Mode** | ✅ | ✅ | Connect to any A2A service |
| **Local Config Mode** | ✅ | ✅ | Direct agent access |
| **Zero-Config Mode** | ❌ | ❌ | Not supported* |

**Note:** Task commands require explicit agent configuration. In zero-config mode, use `call` or `chat` commands instead.

### Client Mode

Task commands work with **any A2A-compliant service**:

```bash
# Query task from Hector server
hector task get assistant task-123 --url http://hector:8080

# Query task from ANY other A2A service
hector task get some-agent task-456 --url http://other-service:8080
```

### Local Mode

When using `--config`, tasks are queried directly from the configured agent:

```bash
# Query task from local configuration
hector task get assistant task-abc123 --config agents.yaml
```

The agent must have task persistence configured to retrieve historical tasks.

---

## API Reference

### REST API

**Get Task:**
```bash
GET /v1/tasks/task-abc123
Authorization: Bearer <token>
```

**Cancel Task:**
```bash
POST /v1/tasks/task-abc123:cancel
Authorization: Bearer <token>
```

**List Tasks:**
```bash
GET /v1/tasks?context_id=ctx-xyz&status=TASK_STATE_WORKING
```

### gRPC API

```proto
service A2AService {
  rpc GetTask(GetTaskRequest) returns (Task);
  rpc CancelTask(CancelTaskRequest) returns (Task);
  rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);
  rpc TaskSubscription(TaskSubscriptionRequest) returns (stream StreamResponse);
}
```

---

## Task Subscription (Streaming)

Subscribe to real-time task updates:

**REST (Server-Sent Events):**
```bash
curl -N http://localhost:8080/v1/tasks/task-abc123:subscribe
```

**gRPC:**
```go
stream, err := client.TaskSubscription(ctx, &pb.TaskSubscriptionRequest{
    Name: "tasks/task-abc123",
})
for {
    resp, err := stream.Recv()
    // Handle status updates, artifacts, etc.
}
```

---

## Production Considerations

### Database Selection

| Database | Use Case | Pros | Cons |
|----------|----------|------|------|
| **SQLite** | Single instance, low volume | Simple setup | No multi-instance |
| **PostgreSQL** | Production, multi-instance | Robust, scalable | Requires setup |
| **MySQL** | Production, MySQL shops | Well-known | Less common for tasks |

### Task Retention

Tasks accumulate over time. Implement cleanup:

```sql
-- PostgreSQL: Delete completed tasks older than 30 days
DELETE FROM tasks 
WHERE state IN ('COMPLETED', 'FAILED', 'CANCELLED')
  AND updated_at < NOW() - INTERVAL '30 days';
```

### Worker Pool Sizing

Configure based on your workload:

```yaml
task:
  worker_pool: 100  # Max concurrent tasks
```

**Guidelines:**
- Start with 100 for most workloads
- Increase for high concurrency needs
- Monitor task queue depth

### Monitoring

Track task metrics:
- Task completion rate
- Average task duration
- Failed task percentage
- Queue depth

---

## Examples

### Long-Running Analysis

**config.yaml:**
```yaml
agents:
  analyst:
    llm: gpt-4o
    tools: [search, write_file]
    task:
      backend: sql
      sql:
        driver: sqlite
        database: ./tasks.db

llms:
  gpt-4o:
    type: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}
```

**Submit task:**
```bash
# Start server
hector serve --config config.yaml &

# Submit non-blocking task via API
TASK_ID=$(curl -s -X POST http://localhost:8080/v1/agents/analyst/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {"parts": [{"text": "Analyze the entire codebase for security issues"}]},
    "configuration": {"blocking": false}
  }' | jq -r '.task.id')

echo "Task submitted: $TASK_ID"
```

**Monitor progress:**
```bash
# Check status periodically
watch -n 5 "hector task get analyst $TASK_ID --config config.yaml"
```

**Cancel if needed:**
```bash
hector task cancel analyst $TASK_ID --config config.yaml
```

---

## Best Practices

### 1. Use Tasks for Long Operations

```yaml
# ✅ Good: Long-running task
POST /v1/message:send
{"configuration": {"blocking": false}}

# ❌ Bad: Quick query as task (unnecessary overhead)
POST /v1/message:send
{"configuration": {"blocking": false}, "message": {"parts": [{"text": "What is 2+2?"}]}}
```

### 2. Configure SQL for Production

```yaml
# ✅ Good: Production setup
task:
  backend: sql
  sql:
    driver: postgres
    database: postgres://...

# ❌ Bad: In-memory for production
# (tasks lost on restart)
```

### 3. Implement Task Cleanup

Set up periodic cleanup to avoid database bloat:

```bash
# Cron job to clean old tasks
0 2 * * * psql -c "DELETE FROM tasks WHERE state='COMPLETED' AND updated_at < NOW() - INTERVAL '30 days'"
```

### 4. Handle Errors Gracefully

```bash
# Check task status before assuming success
STATUS=$(hector task get assistant $TASK_ID --config config.yaml | grep Status | awk '{print $2}')
if [ "$STATUS" = "TASK_STATE_FAILED" ]; then
  echo "Task failed, check logs"
  exit 1
fi
```

---

## Troubleshooting

### "task not found"

**Cause:** Task ID doesn't exist or expired (in-memory mode with restart).

**Solution:**
- Verify task ID is correct
- Use SQL backend for persistence
- Check if server restarted (in-memory tasks are lost)

### Tasks stuck in WORKING

**Cause:** Agent crashed or task deadlocked.

**Solution:**
- Cancel stuck task
- Check agent logs for errors
- Restart agent if needed

```bash
hector task cancel assistant stuck-task-id --config config.yaml
```

### Task commands not working

**Error:** `unsupported mode for client creation`

**Cause:** Task commands require agent configuration (don't work in pure zero-config).

**Solution:**
Use a configuration file:
```bash
# Create minimal config
cat > config.yaml << EOF
agents:
  assistant:
    llm: gpt
llms:
  gpt:
    type: openai
    model: gpt-4o-mini
    api_key: ${OPENAI_API_KEY}
EOF

# Now task commands work
hector task get assistant task-id --config config.yaml
```

---

## Human-in-the-Loop (HITL)

Tasks can pause execution and wait for user input using the `INPUT_REQUIRED` state. This is commonly used for **tool approval** workflows where agents request permission before executing potentially dangerous operations.

**Example:** An agent wants to delete files. Instead of executing immediately, it transitions to `INPUT_REQUIRED` and waits for your approval.

**Key Features:**
- ✅ A2A Protocol compliant (`TASK_STATE_INPUT_REQUIRED`)
- ✅ Multi-turn conversations using the same `taskId`
- ✅ Configurable timeouts for user responses
- ✅ Simple YAML configuration

**Quick Example:**
```yaml
tools:
  delete_file:
    type: delete_file
    requires_approval: true  # Pause for approval

agents:
  assistant:
    task:
      input_timeout: 600  # Wait up to 10 minutes
```

See **[Human-in-the-Loop](human-in-the-loop.md)** for complete documentation.

---

## Next Steps

- **[Streaming](streaming.md)** - Real-time task updates
- **[Multi-Agent Systems](multi-agent.md)** - Coordinate tasks across agents
- **[Session Persistence](sessions.md)** - Long-term conversation storage
- **[Human-in-the-Loop](human-in-the-loop.md)** - Tool approval and interactive workflows
- **[API Reference](../reference/api.md)** - Complete API documentation

---

## Related Topics

- **[A2A Protocol](../reference/a2a-protocol.md)** - Task specification
- **[CLI Reference](../reference/cli.md)** - Task command details
- **[Configuration](../reference/configuration.md)** - Task configuration options

