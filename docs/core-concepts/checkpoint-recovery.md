---
title: Checkpoint Recovery
description: Generic checkpoint/resume system for crash recovery and long-running tasks
---

# Checkpoint Recovery

Hector's **checkpoint recovery** system enables tasks to survive server crashes, handle rate limits gracefully, and support long-running workflows by periodically saving execution state and automatically resuming from checkpoints.

## Overview

### What is Checkpoint Recovery?

Checkpoint recovery allows agent execution to be **paused and resumed from any point**, not just when waiting for user input. This provides:

- ✅ **Crash Recovery**: Tasks survive server crashes and restarts
- ✅ **Rate Limit Resilience**: Pause during backoff, resume when limits reset
- ✅ **Long-Running Tasks**: Checkpoint periodically for very long workflows
- ✅ **Resource Management**: Free goroutines during waits, resume later
- ✅ **Debugging**: Inspect task state at any checkpoint

### Relationship to HITL

Checkpoint recovery **extends** the async HITL foundation:

- **Shared Foundation**: Both use `ExecutionState`, session metadata storage, and the same recovery mechanisms
- **HITL is Event-Driven**: Async HITL checkpoints on `INPUT_REQUIRED` transitions (tool approval)
- **Checkpoint is Generic**: Adds interval-based checkpointing and recovery from any execution point
- **Unified Storage**: Both store checkpoints in session metadata (`sessions.metadata` JSON field)

Think of it this way:
- **Async HITL** = Checkpoint on user input events
- **Checkpoint Recovery** = Checkpoint on events + intervals + automatic recovery

---

## How It Works

### Checkpoint Strategies

Hector supports three checkpoint strategies:

#### 1. Event-Driven (Default)

Checkpoints are created on specific events:
- Tool approval requests (HITL pauses)
- Errors
- Rate limit backoffs

**Characteristics:**
- Minimal overhead (only checkpoints when needed)
- Precise recovery points
- Uses async HITL foundation

**Configuration:**
```yaml
task:
  checkpoint:
    enabled: true
    strategy: "event"  # Default
```

#### 2. Interval-Based

Checkpoints are created periodically during execution:
- Every N iterations
- After tool calls (optional)
- Before LLM calls (optional)

**Characteristics:**
- Background checkpointing (task remains in `WORKING` state)
- Configurable frequency
- Trade-off: overhead vs recovery granularity

**Configuration:**
```yaml
task:
  checkpoint:
    enabled: true
    strategy: "interval"
    interval:
      every_n_iterations: 5  # Checkpoint every 5 iterations
      after_tool_calls: true  # Also checkpoint after tool calls
```

#### 3. Hybrid (Recommended)

Combines event-driven and interval-based:
- Event-driven checkpoints on HITL pauses
- Interval-based checkpoints during normal execution
- Best recovery coverage

**Configuration:**
```yaml
task:
  checkpoint:
    enabled: true
    strategy: "hybrid"
    interval:
      every_n_iterations: 5
```

### Checkpoint Storage

Checkpoints are stored in **session metadata** (same foundation as async HITL):

```json
{
  "sessions": {
    "session-123": {
      "metadata": {
        "pending_executions": {
          "task-abc": {
            "task_id": "task-abc",
            "context_id": "session-123",
            "query": "Analyze codebase",
            "reasoning_state": {...},
            "phase": "iteration_end",
            "checkpoint_type": "interval",
            "checkpoint_time": "2025-01-15T10:30:00Z"
          }
        }
      }
    }
  }
}
```

**Why Session Metadata?**
- ✅ Already persisted (survives restarts)
- ✅ Natural relationship (tasks → sessions via `context_id`)
- ✅ No schema changes (uses existing JSON field)
- ✅ Multi-agent isolation (by `agent_id` + `session_id`)

### Recovery Mechanism

On server startup, Hector automatically:

1. **Finds pending tasks**: Queries tasks in `WORKING` or `INPUT_REQUIRED` state
2. **Loads checkpoints**: Retrieves execution state from session metadata
3. **Validates checkpoints**: Checks expiration and validity
4. **Resumes execution**: Restores reasoning state and continues from checkpoint

**Recovery Flow:**
```
Server Startup
  ↓
Find tasks in WORKING or INPUT_REQUIRED state
  ↓
Load checkpoint from session metadata
  ↓
Validate checkpoint (not expired, valid state)
  ↓
Resume execution from checkpoint
  ↓
Task continues from where it left off
```

---

## Configuration

### Basic Setup

Enable checkpoint recovery:

```yaml
agents:
  assistant:
    # Session store (required for checkpoint storage)
    session_store: "sqlite"
    
    task:
      backend: "sql"
      checkpoint:
        enabled: true
        strategy: "hybrid"
        
        interval:
          every_n_iterations: 5
        
        recovery:
          auto_resume: true
          resume_timeout: 3600  # 1 hour
```

### Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `false` | Enable checkpoint recovery |
| `strategy` | string | `"event"` | `"event"`, `"interval"`, or `"hybrid"` |
| `interval.every_n_iterations` | integer | `0` | Checkpoint every N iterations (0 = disabled) |
| `interval.after_tool_calls` | boolean | `false` | Always checkpoint after tool calls |
| `interval.before_llm_calls` | boolean | `false` | Checkpoint before LLM calls |
| `recovery.auto_resume` | boolean | `false` | Auto-resume tasks on startup |
| `recovery.auto_resume_hitl` | boolean | `false` | Auto-resume INPUT_REQUIRED tasks |
| `recovery.resume_timeout` | integer | `3600` | Max checkpoint age to resume (seconds) |

### Programmatic API

Configure checkpoint recovery programmatically:

```go
taskService := hector.NewTaskService().
    Backend("sql").
    Checkpoint().
        Enabled(true).
        Strategy("hybrid").
        Interval().
            EveryNIterations(5).
            Build().
        Recovery().
            AutoResume(true).
            ResumeTimeout(3600).
            Build().
        Build().
    Build()

agent, err := hector.NewAgent("assistant").
    WithLLMProvider(llm).
    WithTask(taskService).
    WithSession(sessionService).
    Build()
```

---

## Use Cases

### 1. Crash Recovery

**Scenario:** Server crashes mid-execution

**Without Checkpoint Recovery:**
- Task lost, must restart from beginning
- User frustration, wasted LLM tokens

**With Checkpoint Recovery:**
- Task automatically resumes from last checkpoint
- Seamless recovery, no user intervention

**Configuration:**
```yaml
task:
  checkpoint:
    enabled: true
    strategy: "interval"
    interval:
      every_n_iterations: 3  # Frequent checkpoints
    recovery:
      auto_resume: true
```

### 2. Rate Limit Resilience

**Scenario:** LLM API rate limit hit during long task

**Without Checkpoint Recovery:**
- Task fails, must restart
- Wasted progress

**With Checkpoint Recovery:**
- Task checkpoints before rate limit error
- Resumes automatically when limits reset
- Continues from checkpoint

**Configuration:**
```yaml
task:
  checkpoint:
    enabled: true
    strategy: "hybrid"
    interval:
      before_llm_calls: true  # Checkpoint before expensive calls
```

### 3. Long-Running Tasks

**Scenario:** Task takes hours to complete (e.g., codebase analysis)

**Without Checkpoint Recovery:**
- Risk of losing progress on restart
- No way to pause/resume

**With Checkpoint Recovery:**
- Periodic checkpoints ensure progress saved
- Can pause for maintenance, resume later
- Survives server restarts

**Configuration:**
```yaml
task:
  checkpoint:
    enabled: true
    strategy: "interval"
    interval:
      every_n_iterations: 10  # Checkpoint every 10 iterations
    recovery:
      auto_resume: true
      resume_timeout: 86400  # 24 hours
```

### 4. Resource Management

**Scenario:** Many concurrent tasks, need to free resources

**Without Checkpoint Recovery:**
- Goroutines blocked waiting
- Resource exhaustion

**With Checkpoint Recovery:**
- Tasks checkpoint and exit goroutines
- Resume later when resources available
- Better resource utilization

---

## A2A Protocol Compliance

Hector's checkpoint recovery is **100% A2A Protocol compliant**:

### State Management ✅

- ✅ **Background Checkpoints**: Remain in `WORKING` state (invisible to client)
- ✅ **Explicit Pauses**: Use `INPUT_REQUIRED` state (standard A2A)
- ✅ **Valid Transitions**: `WORKING` → Checkpoint → Resume as `WORKING`
- ✅ **Terminal States**: Never checkpoints from terminal states (A2A violation)

### Recovery Process ✅

- ✅ **Standard Resume**: Uses A2A `message:send` with `taskId`
- ✅ **Multi-Turn**: Same `taskId` for resume (A2A Section 6.3.1)
- ✅ **Context Alignment**: Uses `contextId` for session continuity
- ✅ **State Persistence**: Task state persisted in task store

### Checkpoint Scenarios

| Scenario | Task State | Checkpoint Type | A2A Compliance |
|----------|------------|-----------------|----------------|
| Tool approval | `INPUT_REQUIRED` | Event-driven | ✅ Standard HITL |
| Interval checkpoint | `WORKING` | Interval-based | ✅ Background only |
| Crash recovery | `WORKING` (resume) | Interval-based | ✅ Invisible to client |
| Rate limit pause | `INPUT_REQUIRED` | Event-driven | ✅ Explicit pause |

**Key Principle:**
- **Client-visible pauses** → Use `INPUT_REQUIRED` state
- **Background checkpoints** → Remain in `WORKING` state
- **Never checkpoint** from terminal states

---

## Examples

### Example 1: Basic Crash Recovery

**Configuration:**
```yaml
agents:
  analyst:
    session_store: "sqlite"
    task:
      backend: "sql"
      checkpoint:
        enabled: true
        strategy: "interval"
        interval:
          every_n_iterations: 5
        recovery:
          auto_resume: true
```

**Behavior:**
- Checkpoints created every 5 iterations
- On server restart, tasks automatically resume
- No user intervention required

### Example 2: HITL + Checkpoint

**Configuration:**
```yaml
agents:
  assistant:
    session_store: "sqlite"
    task:
      backend: "sql"
      hitl:
        mode: "async"
      checkpoint:
        enabled: true
        strategy: "hybrid"
        interval:
          every_n_iterations: 10
        recovery:
          auto_resume: true
          auto_resume_hitl: false  # Wait for user input
```

**Behavior:**
- HITL pauses create event-driven checkpoints
- Interval checkpoints every 10 iterations
- On restart: auto-resume WORKING tasks, wait for INPUT_REQUIRED

### Example 3: Production Setup

**Configuration:**
```yaml
session_stores:
  main-db:
    backend: sql
    sql:
      driver: postgres
      host: db.example.com
      database: hector_sessions

agents:
  production-agent:
    session_store: "main-db"
    task:
      backend: "sql"
      sql:
        driver: postgres
        host: db.example.com
        database: hector_tasks
      checkpoint:
        enabled: true
        strategy: "hybrid"
        interval:
          every_n_iterations: 5
          after_tool_calls: true
        recovery:
          auto_resume: true
          resume_timeout: 7200  # 2 hours
```

---

## Performance Considerations

### Checkpoint Overhead

**Costs:**
- State serialization: ~1-5ms per checkpoint
- Storage write: ~5-20ms (depends on backend)
- Total: ~10-25ms per checkpoint

**Mitigation:**
- Use interval-based sparingly (every 5-10 iterations)
- Enable `after_tool_calls` only if needed
- Use `before_llm_calls` for expensive operations

### Storage Considerations

**Session Metadata Size:**
- Typical checkpoint: 10-50KB
- Multiple checkpoints per session: ~100-500KB
- Monitor session metadata size

**Best Practices:**
- Clean up old checkpoints after task completion
- Monitor database size
- Use PostgreSQL for production (better JSON handling)

---

## Troubleshooting

### Checkpoints Not Created

**Symptom:** No checkpoints found after server restart

**Check:**
1. Is `checkpoint.enabled: true`?
2. Is `session_store` configured?
3. Are tasks reaching checkpoint points?

**Solution:**
```yaml
# Verify configuration
task:
  checkpoint:
    enabled: true  # Must be true
    strategy: "interval"
    interval:
      every_n_iterations: 5  # Must be > 0

session_store: "sqlite"  # Required
```

### Tasks Not Resuming

**Symptom:** Tasks remain in `WORKING` state after restart

**Check:**
1. Is `recovery.auto_resume: true`?
2. Are checkpoints expired? (check `resume_timeout`)
3. Are checkpoints valid? (check logs)

**Solution:**
```yaml
recovery:
  auto_resume: true  # Must be true
  resume_timeout: 3600  # Increase if checkpoints too old
```

### Checkpoint Expired

**Symptom:** Checkpoints found but not resumed (expired)

**Solution:**
- Increase `resume_timeout` if tasks legitimately paused longer
- Or manually resume via API if needed

---

## Best Practices

### ✅ Do

- Enable checkpoint recovery for production deployments
- Use `hybrid` strategy for best coverage
- Set appropriate `every_n_iterations` (5-10 recommended)
- Configure `resume_timeout` based on your use case
- Monitor checkpoint creation and recovery in logs

### ❌ Don't

- Don't checkpoint too frequently (overhead)
- Don't use interval-based without `session_store`
- Don't set `auto_resume_hitl: true` unless you want automatic approval
- Don't ignore checkpoint errors (they indicate issues)

---

## Related Topics

- **[Human-in-the-Loop](human-in-the-loop.md)** - Tool approval workflows (uses checkpoint foundation)
- **[Tasks](tasks.md)** - Task lifecycle and management
- **[Sessions](sessions.md)** - Session persistence (checkpoint storage)
- **[Generic Checkpoint/Resume Design](../design/generic-checkpoint-resume.md)** - Complete architecture details
- **[Programmatic API Reference](../reference/programmatic-api.md)** - Configure checkpoints programmatically
- **[Configuration Reference](../reference/configuration.md)** - All checkpoint options

---

## Next Steps

- **[Setup Session Persistence](../how-to/setup-session-persistence.md)** - Configure session storage for checkpoints
- **[Making HITL Truly Asynchronous](../how-to/async-hitl.md)** - Learn about async HITL (checkpoint foundation)
- **[Tasks Guide](tasks.md)** - Complete task management documentation

