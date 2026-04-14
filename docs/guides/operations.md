# Operations

This guide covers operating Hector in production: database, task queue, checkpointing, and multi-tenancy.

## Database Configuration

Hector stores sessions, tasks, checkpoints, and app configurations.

### SQLite (Default)

Best for single-instance deployments:

```bash
hector serve --database "sqlite://.hector/hector.db"
```

### PostgreSQL (Production)

Required for multi-instance or HA:

```bash
hector serve --database "postgres://user:pass@localhost:5432/hector?sslmode=require"
```

> Schema migrations run automatically on startup.

---

## Task Queue

Durable background execution with retry and recovery.

### Queue Flow

```
Request → Enqueue → [Worker Pool] → Execute → Complete
                         ↓
                    Retry on failure
                         ↓
                    Dead Letter Queue
```

### Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `--queue-workers` | `4` | Concurrent workers |
| `--queue-max-retries` | `3` | Retry attempts |
| `--queue-initial-delay` | `1s` | First retry delay |
| `--queue-max-delay` | `5m` | Max backoff cap |
| `--queue-stale-threshold` | `5m` | Zombie task recovery |

```bash
hector serve \
  --queue-workers 8 \
  --queue-max-retries 5 \
  --queue-stale-threshold 3m
```

### Task States

| State | Description |
|-------|-------------|
| `pending` | Waiting in queue |
| `running` | Being processed |
| `completed` | Successfully finished |
| `failed` | Exhausted retries |
| `input_required` | Awaiting human approval |

### Stale Task Recovery

If a worker crashes, other workers pick up stale tasks after threshold.

---

## Checkpoint System

Incremental progress snapshots for long-running operations.

### Use Cases

- **RAG Indexing** - Resume after restart via file checksums
- **Workflow Stages** - Track progress through pipelines
- **HITL Tasks** - Preserve state awaiting approval

### Recovery Behavior

On restart, Hector automatically recovers:
- Pending tasks from queue
- In-progress RAG indexing
- HITL-waiting tasks

No manual intervention required.

---

## Human-in-the-Loop (HITL)

When a tool requires approval, the task checkpoints and waits:

```yaml
tools:
  deploy_prod:
    type: command
    command: "./deploy.sh"
    require_approval: true
```

### Flow

1. Agent calls approval-required tool
2. Task enters `input_required` state  
3. Checkpoint saved to database
4. API returns approval request
5. Human approves/rejects via API
6. Task resumes or terminates

### API

```bash
# List pending approvals
curl http://localhost:8080/tasks?status=input_required

# Approve
curl -X POST http://localhost:8080/tasks/{id}/approve \
  -H "Authorization: Bearer ${TOKEN}"
```

---

## Multi-Tenancy

Deploy multiple isolated apps on a single Hector instance.

### Architecture

```
┌─────────────────────────────────────────────┐
│                Hector Server                │
│  ┌───────────┐  ┌───────────┐  ┌─────────┐  │
│  │  App: A   │  │  App: B   │  │ App: C  │  │
│  └───────────┘  └───────────┘  └─────────┘  │
└─────────────────────────────────────────────┘
```

Each app has isolated agents, tools, sessions, and configuration.

### Admin API

```bash
# Create app
curl -X POST http://localhost:8080/admin/apps \
  -H "Authorization: Bearer ${AUTH_SECRET}" \
  -d '{"name": "customer-a", "config": {...}}'

# List apps
curl http://localhost:8080/admin/apps
```

### Tenant Routing via JWT

JWT `tenant_id` claim maps to app name automatically.

---

## Health Checks

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "ok",
  "version": "v1.20.0",
  "database": "connected"
}
```

### Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
```

---

## Graceful Shutdown

On SIGTERM/SIGINT:

1. Stop accepting new requests
2. Wait for in-flight requests (30s timeout)
3. Checkpoint active tasks
4. Close database connections

---

## Next Steps

*   [Security Guide](./security.md)
*   [Observability Guide](./observability.md)
*   [CLI Reference](../reference/cli.md)
