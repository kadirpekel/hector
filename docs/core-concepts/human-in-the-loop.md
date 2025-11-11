---
title: Human-in-the-Loop
description: Tool approval and interactive workflows with A2A Protocol compliance
---

# Human-in-the-Loop (HITL)

Hector implements **100% A2A Protocol-compliant** human-in-the-loop features, allowing agents to pause execution and wait for user approval or input before proceeding with certain actions.

## A2A Protocol Compliance

This implementation follows **A2A Protocol Section 6.3** for the `INPUT_REQUIRED` state:

- Tasks transition to `TASK_STATE_INPUT_REQUIRED` when user input is needed
- Multi-turn conversations use the same `taskId` to resume execution
- Interaction details are provided in `TaskStatus.update` (JSON field name: `message`)
- Standard A2A message structure with `TextPart` and `DataPart`

---

## Use Cases

### 1. Tool Approval

Require user approval before executing potentially dangerous operations:

```yaml
tools:
  execute_command:
    type: command
    enabled: true
    requires_approval: true  # ⭐ Enable approval
    approval_prompt: "Allow command execution: {input}?"
    allowed_commands:
      - "ls"
      - "rm"
      - "git"
  
  delete_file:
    type: delete_file
    enabled: true
    requires_approval: true  # ⭐ Require approval for deletions
```

### 2. Custom Prompts

Customize the approval message for each tool:

```yaml
tools:
  write_file:
    type: write_file
    enabled: true
    requires_approval: true
    approval_prompt: |
      📝 File Write Request
      
      Tool: {tool}
      Input: {input}
      
      Do you approve this operation?
```

---

## Configuration

### Task Configuration

Configure timeout and HITL mode:

```yaml
agents:
  assistant:
    llm: "gpt-4o"
    
    task:
      backend: "memory"  # or "sql"
      worker_pool: 5
      input_timeout: 600  # Seconds to wait for user input (default: 600 = 10 minutes)
      
      # HITL Mode Configuration (optional)
      hitl:
        mode: "auto"  # "auto" (default), "blocking", or "async"
    
    # Session persistence (required for async HITL)
    session_store: "sqlite"  # Required when hitl.mode is "async"
    
    tools:
      - "execute_command"
      - "write_file"
```

**HITL Mode Options:**

| Mode | Description | Requirements |
|------|-------------|--------------|
| `auto` | Automatically detect based on `session_store` presence | None |
| `blocking` | Block execution goroutine while waiting (default behavior) | None |
| `async` | Save state and exit goroutine, resume later | Requires `session_store` configured |

**Mode Detection:**
- If `hitl.mode` is not specified, defaults to `auto`
- `auto` mode uses **async** if `session_store` is configured, otherwise **blocking**
- `async` mode requires `session_store` - validation will fail if missing

### Tool Configuration

Each tool can have approval settings:

```yaml
tools:
  tool_name:
    type: "tool_type"
    enabled: true
    
    # Human-in-the-loop settings
    requires_approval: true           # If true, agent pauses for approval
    approval_prompt: "Custom prompt"  # Optional: Custom approval message
```

**Supported interpolations in `approval_prompt`:**
- `{tool}` - Tool name
- `{input}` - Tool input/arguments

---

## How It Works

### 1. Agent Execution Flow

```
User sends message
   ↓
Agent starts processing (TASK_STATE_WORKING)
   ↓
Agent wants to call tool requiring approval
   ↓
Task transitions to TASK_STATE_INPUT_REQUIRED  ⭐ A2A standard state
   ↓
Approval request sent in TaskStatus.message
   ↓
Agent execution pauses (waits for user response)
   ↓
User sends response with same taskId  ⭐ A2A multi-turn
   ↓
Task resumes to TASK_STATE_WORKING
   ↓
Tool executes (if approved) or skips (if denied)
   ↓
Task completes (TASK_STATE_COMPLETED)
```

### 2. A2A Message Structure

When a task requires input, the `TaskStatus` looks like:

```json
{
  "id": "task-123",
  "status": {
    "state": "TASK_STATE_INPUT_REQUIRED",
    "message": {
      "role": "ROLE_AGENT",
      "parts": [
        {
          "text": "🔐 Tool Approval Required\n\nTool: execute_command\nInput: rm -rf /tmp/old-files\n\nPlease respond with: approve or deny"
        },
        {
          "data": {
            "interaction_type": "tool_approval",
            "tool_name": "execute_command",
            "tool_input": "rm -rf /tmp/old-files",
            "options": ["approve", "deny"]
          }
        }
      ]
    },
    "timestamp": "2025-11-07T10:00:00Z"
  }
}
```

**Key points:**
- ✅ Uses standard `TASK_STATE_INPUT_REQUIRED` (A2A Protocol Section 6.3)
- ✅ `TextPart` provides human-readable prompt
- ✅ `DataPart` provides structured metadata for programmatic parsing
- ✅ No custom protocol extensions needed

---

## Client Usage

### REST API

#### 1. Send Initial Message

```bash
curl -X POST http://localhost:8080/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "request": {
      "role": "user",
      "parts": [{"text": "Delete all temporary files"}]
    },
    "configuration": {
      "blocking": false
    }
  }'
```

**Response:**
```json
{
  "task": {
    "id": "task-abc123",
    "status": {
      "state": "TASK_STATE_INPUT_REQUIRED",
      "message": {
        "role": "ROLE_AGENT",
        "parts": [
          {"text": "🔐 Tool Approval Required..."},
          {"data": {"interaction_type": "tool_approval", ...}}
        ]
      }
    }
  }
}
```

#### 2. Respond with Approval

**A2A Protocol: Use same `taskId` to resume**

```bash
curl -X POST http://localhost:8080/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "request": {
      "role": "user",
      "taskId": "task-abc123",
      "parts": [
        {"text": "approve"},
        {"data": {"decision": "approve"}}
      ]
    }
  }'
```

**Response:**
```json
{
  "task": {
    "id": "task-abc123",
    "status": {
      "state": "TASK_STATE_COMPLETED",
      "message": {
        "role": "ROLE_AGENT",
        "parts": [{"text": "Files deleted successfully"}]
      }
    }
  }
}
```

---

### TypeScript Client

```typescript
import { A2AClient } from '@hector/sdk';

const client = new A2AClient('http://localhost:8080');

async function executeWithApproval() {
  // Send initial message
  let task = await client.sendMessage('assistant', {
    role: 'user',
    parts: [{ text: 'Delete all temporary files' }]
  });
  
  // Check if approval is required (A2A standard state)
  while (task.status.state === 'TASK_STATE_INPUT_REQUIRED') {
    // Parse approval request from task.status.message
    const message = task.status.message;
    const textPart = message.parts.find(p => 'text' in p);
    const dataPart = message.parts.find(p => 'data' in p);
    
    console.log(textPart.text); // Display to user
    
    // Get user decision
        const decision = await prompt('Approve? (approve/deny)');
    
    // Send response with SAME taskId (A2A multi-turn)
    task = await client.sendMessage('assistant', {
      role: 'user',
      taskId: task.id,  // ⭐ Same task - resume execution
      contextId: task.contextId,
      parts: [
        { text: decision },
        { data: { decision: decision } }
      ]
    });
  }
  
  console.log('Task completed:', task);
}
```

---

### Python Client

```python
from hector_sdk import A2AClient

client = A2AClient('http://localhost:8080')

def execute_with_approval():
    # Send initial message
    task = client.send_message('assistant', {
        'role': 'user',
        'parts': [{'text': 'Delete all temporary files'}]
    })
    
    # Handle approval requests (A2A standard INPUT_REQUIRED state)
    while task['status']['state'] == 'TASK_STATE_INPUT_REQUIRED':
        # Parse approval request
        message = task['status']['message']
        text_part = next(p for p in message['parts'] if 'text' in p)
        data_part = next(p for p in message['parts'] if 'data' in p)
        
        print(text_part['text'])
        
        # Get user decision
        decision = input('Approve? (approve/deny): ')
        
        # Send response with same taskId (A2A multi-turn)
        task = client.send_message('assistant', {
            'role': 'user',
            'taskId': task['id'],  # ⭐ Resume execution
            'contextId': task['contextId'],
            'parts': [
                {'text': decision},
                {'data': {'decision': decision}}
            ]
        })
    
    print(f"Task completed: {task}")
```

---

## Response Options

Users can respond with three options:

### 1. Approve

Execute the tool with original parameters:

```json
{
  "role": "user",
  "taskId": "task-abc123",
  "parts": [
    {"text": "approve"},
    {"data": {"decision": "approve"}}
  ]
}
```

### 2. Deny

Skip the tool execution:

```json
{
  "role": "user",
  "taskId": "task-abc123",
  "parts": [
    {"text": "deny"},
    {"data": {"decision": "deny"}}
  ]
}
```

---

## Task Cancellation

Cancel a running or paused task:

### REST API

```bash
POST /v1/agents/{agent}/tasks/{taskID}:cancel
```

### Effect

- Cancels active execution (sends context cancellation signal)
- Clears any waiting input requests
- Transitions task to `TASK_STATE_CANCELLED`

---

## Best Practices

### 1. Enable Approval for Dangerous Operations

```yaml
tools:
  # Safe tools - no approval needed
  read_file:
    type: read_file
    enabled: true
    requires_approval: false  # Safe operation
  
  # Dangerous tools - require approval
  execute_command:
    type: command
    enabled: true
    requires_approval: true  # Potentially dangerous
  
  delete_file:
    type: delete_file
    enabled: true
    requires_approval: true  # Irreversible operation
```

### 2. Set Reasonable Timeouts

```yaml
agents:
  assistant:
    task:
      input_timeout: 300  # 5 minutes for quick decisions
```

### 3. Use Clear Approval Prompts

```yaml
tools:
  execute_command:
    requires_approval: true
    approval_prompt: |
      ⚠️  Command Execution Request
      
      Command: {input}
      
      This will execute on the server. Approve?
```

### 4. Handle Timeout Gracefully

If user doesn't respond within `input_timeout`, the task fails with timeout error.

---

## Advanced: Task Subscription

Subscribe to task updates to receive real-time notifications:

```bash
GET /v1/agents/{agent}/tasks/{taskID}:subscribe
```

**Response (SSE stream):**
```
event: status_update
data: {"taskId":"task-123","status":{"state":"TASK_STATE_INPUT_REQUIRED",...}}

event: status_update
data: {"taskId":"task-123","status":{"state":"TASK_STATE_WORKING",...}}

event: status_update
data: {"taskId":"task-123","status":{"state":"TASK_STATE_COMPLETED",...}}
```

---

## Troubleshooting

### Task Tracking Not Enabled

**Error:** "Tool requires approval but task tracking not enabled"

**Solution:** Enable task tracking in agent config:

```yaml
agents:
  assistant:
    task:
      backend: "memory"  # or "sql"
```

### Timeout Waiting for Input

**Error:** "timeout waiting for user input"

**Solution:** Increase `input_timeout` or respond faster:

```yaml
agents:
  assistant:
    task:
      input_timeout: 900  # 15 minutes
```

### Task Not Resuming

**Issue:** Sending response doesn't resume task

**Check:**
1. Are you using the correct `taskId`?
2. Is task state `TASK_STATE_INPUT_REQUIRED`?
3. Did the task timeout?

---

## Examples

### Complete Example Config

```yaml
llms:
  gpt-4o:
    type: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}

tools:
  execute_command:
    type: command
    enabled: true
    requires_approval: true
    approval_prompt: "Execute command: {input}?"
    allowed_commands: ["ls", "pwd", "git", "npm"]
  
  write_file:
    type: write_file
    enabled: true
    requires_approval: true
    approval_prompt: "Write file with input: {input}?"
  
  read_file:
    type: read_file
    enabled: true
    requires_approval: false  # Safe operation

agents:
  assistant:
    llm: "gpt-4o"
    
    reasoning:
      engine: "chain_of_thought"
      enable_streaming: true
    
    task:
      backend: "memory"
      worker_pool: 5
      input_timeout: 600
    
    tools:
      - "execute_command"
      - "write_file"
      - "read_file"
```

---

## Summary

✅ **100% A2A Protocol Compliant** - Uses standard `INPUT_REQUIRED` state  
✅ **No Custom Extensions** - Pure A2A message structure  
✅ **Multi-turn Support** - Same `taskId` resumes execution  
✅ **Simple Configuration** - Just set `requires_approval: true`  
✅ **Task Cancellation** - Cancel at any time via standard A2A endpoint  

The implementation provides secure human-in-the-loop workflows while maintaining full compatibility with the A2A Protocol specification.

---

## Advanced: Truly Asynchronous HITL

Hector supports two HITL modes: **blocking** and **async**. The mode determines how tasks handle user input requests.

### Blocking Mode (Default)

In blocking mode, the execution goroutine pauses and waits for user input:

- ✅ Simple and straightforward
- ✅ Immediate response when user provides input
- ❌ Goroutine remains blocked while waiting
- ❌ State lost on server restart
- ❌ Limited scalability for many concurrent paused tasks

**When to use:** Development, testing, or when you don't need persistence.

### Async Mode (Production-Ready)

In async mode, execution state is saved to session metadata and the goroutine exits:

- ✅ Non-blocking - goroutines are freed immediately
- ✅ State persists across server restarts
- ✅ Scales to many concurrent paused tasks
- ✅ Production-ready for long-running workflows
- ⚠️ Requires `session_store` configuration

**When to use:** Production deployments, long-running workflows, or when you need state persistence.

### Configuration

```yaml
agents:
  assistant:
    llm: "gpt-4o"
    
    # Session persistence (required for async mode)
    session_store: "sqlite"
    
    task:
      backend: "sql"
      hitl:
        mode: "async"  # Enable async HITL
```

**Mode Selection:**

1. **Explicit async mode:**
   ```yaml
   task:
     hitl:
       mode: "async"  # Requires session_store
   ```

2. **Explicit blocking mode:**
   ```yaml
   task:
     hitl:
       mode: "blocking"  # Always blocking, even with session_store
   ```

3. **Auto-detect (recommended):**
   ```yaml
   task:
     hitl:
       mode: "auto"  # Uses async if session_store exists, else blocking
   ```

### How Async Mode Works

```
User sends message
   ↓
Agent starts processing (TASK_STATE_WORKING)
   ↓
Agent wants to call tool requiring approval
   ↓
Execution state saved to session metadata  ⭐ State persisted
   ↓
Task transitions to TASK_STATE_INPUT_REQUIRED
   ↓
Goroutine exits (non-blocking)  ⭐ Resources freed
   ↓
[Server can restart - state preserved]
   ↓
User sends response with same taskId
   ↓
State loaded from session metadata  ⭐ State restored
   ↓
New goroutine starts, execution resumes
   ↓
Task completes (TASK_STATE_COMPLETED)
```

**Key Benefits:**
- State survives server restarts
- No blocked goroutines
- Better resource utilization
- Production-ready reliability

---

## Checkpoint Recovery (Generic)

Hector extends async HITL with **generic checkpoint/resume** functionality that enables recovery from any point in execution, not just HITL pauses. This provides:

- ✅ **Crash Recovery**: Tasks survive server crashes and restarts
- ✅ **Rate Limit Resilience**: Pause during backoff, resume when limits reset
- ✅ **Long-Running Tasks**: Checkpoint periodically for very long workflows
- ✅ **Resource Management**: Free goroutines during waits, resume later

**Configuration:**
```yaml
agents:
  assistant:
    task:
      checkpoint:
        enabled: true
        strategy: "hybrid"  # "event", "interval", or "hybrid"
        
        interval:
          every_n_iterations: 5  # Checkpoint every 5 iterations
        
        recovery:
          auto_resume: true        # Auto-resume on startup
          auto_resume_hitl: false  # Don't auto-resume INPUT_REQUIRED
          resume_timeout: 3600     # 1 hour timeout
    
    session_store: "sqlite"  # Required for checkpoint storage
```

**How It Works:**
- Uses the same foundation as async HITL (`ExecutionState`, session metadata)
- Interval-based checkpoints happen in background (task remains in `WORKING` state)
- Event-driven checkpoints happen on HITL pauses (`INPUT_REQUIRED` state)
- On server restart, tasks in `WORKING` or `INPUT_REQUIRED` state are automatically recovered if checkpoints exist

**A2A Compliance:**
- ✅ Background checkpoints remain in `WORKING` (invisible to client)
- ✅ Explicit pauses use `INPUT_REQUIRED` (standard A2A state)
- ✅ Recovery uses standard A2A `message:send` with `taskId`
- ✅ Never checkpoints from terminal states

See **[Generic Checkpoint/Resume Design](../design/generic-checkpoint-resume.md)** for complete architecture details.

### Migration Guide

**From Blocking to Async:**

1. Configure session persistence:
   ```yaml
   session_stores:
     sqlite:
       type: sqlite
       database: "./sessions.db"
   
   agents:
     assistant:
       session_store: "sqlite"
   ```

2. Enable async mode:
   ```yaml
   task:
     hitl:
       mode: "async"
   ```

3. Restart server - existing blocking tasks will complete, new tasks use async mode

**See:** [Making HITL Truly Asynchronous](../how-to/async-hitl.md) for complete implementation guide.

---

## Related Topics

- **[Checkpoint Recovery](checkpoint-recovery.md)** - Generic checkpoint/resume system (extends async HITL)
- **[Tasks](tasks.md)** - Task lifecycle and management
- **[Sessions](sessions.md)** - Session persistence (checkpoint storage)
- **[Making HITL Truly Asynchronous](../how-to/async-hitl.md)** - Implementation guide

