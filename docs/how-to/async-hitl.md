---
title: Making HITL Truly Asynchronous
description: Guide to implementing asynchronous human-in-the-loop that survives restarts
---

# Making HITL Truly Asynchronous

This guide explains how to make Hector's Human-in-the-Loop (HITL) feature truly asynchronous, allowing tasks to pause and resume at any time, even after server restarts.

## Current Limitations

The current HITL implementation has these limitations:

1. **Blocking Goroutines**: When a task needs approval, the execution goroutine blocks waiting for input via channels
2. **In-Memory State**: Waiting tasks are stored in memory (`TaskAwaiter.waiting` map), lost on restart
3. **No State Persistence**: Execution state (reasoning state, pending tool calls) is not persisted
4. **No Resume After Restart**: If the server restarts while a task is waiting, it cannot be resumed

## Architecture Overview

**Key Insight**: Hector already has persistent session support! Tasks are linked to sessions via `context_id` (which maps to `session_id`). We can leverage the existing session metadata storage instead of creating separate execution state storage.

```
┌─────────────────────────────────────────────────────────┐
│ 1. SESSION METADATA STORAGE                              │
│    - Use existing sessions.metadata JSON field            │
│    - Store execution state keyed by task_id              │
│    - Leverages existing session persistence ✅            │
├─────────────────────────────────────────────────────────┤
│ 2. NON-BLOCKING EXECUTION                               │
│    - Don't block goroutines on WaitForInput()            │
│    - Save state to session metadata and exit             │
│    - Task remains in INPUT_REQUIRED state                │
├─────────────────────────────────────────────────────────┤
│ 3. RESUME MECHANISM                                     │
│    - Load execution state from session metadata          │
│    - Reconstruct reasoning state                         │
│    - Continue execution from checkpoint                  │
└─────────────────────────────────────────────────────────┘
```

**Why Session Metadata?**

✅ **Already Persisted**: Sessions survive restarts  
✅ **Natural Relationship**: Tasks → Sessions via `context_id`  
✅ **No Schema Changes**: Uses existing `metadata` JSON field  
✅ **Multi-Agent Isolation**: Already handled by `(agent_id, session_id)`  
✅ **Unified Storage**: Same database as tasks (when both use SQL)  

## Implementation Steps

### Step 1: Use Session Metadata (No Schema Changes!)

Instead of adding new columns, use the existing session metadata:

```sql
-- Sessions table already has:
-- metadata TEXT  -- JSON field for arbitrary data

-- Store execution state in metadata like:
{
  "pending_executions": {
    "task-123": {
      "execution_state": {...},
      "pending_tool_call": {...},
      "checkpoint_data": {...}
    }
  }
}
```

**Benefits:**
- ✅ No schema migration needed
- ✅ Leverages existing session persistence
- ✅ Automatic cleanup when session is deleted
- ✅ Works with any session store backend (SQL, in-memory)

### Step 2: Create Execution State Serialization

Create a serializable representation of execution state:

```go
// pkg/agent/execution_state.go

package agent

import (
    "encoding/json"
    "github.com/kadirpekel/hector/pkg/a2a/pb"
    "github.com/kadirpekel/hector/pkg/protocol"
    "github.com/kadirpekel/hector/pkg/reasoning"
)

// ExecutionState represents the state needed to resume task execution
type ExecutionState struct {
    TaskID           string                 `json:"task_id"`
    ContextID        string                 `json:"context_id"`
    Query            string                 `json:"query"`
    ReasoningState   *ReasoningStateSnapshot `json:"reasoning_state"`
    PendingToolCall  *protocol.ToolCall     `json:"pending_tool_call"`
    History          []*pb.Message          `json:"history"`
    CurrentTurn      []*pb.Message          `json:"current_turn"`
    Iteration        int                    `json:"iteration"`
    TotalTokens      int                    `json:"total_tokens"`
    AssistantResponse string                 `json:"assistant_response"`
}

// ReasoningStateSnapshot is a serializable version of reasoning state
type ReasoningStateSnapshot struct {
    Iteration              int                    `json:"iteration"`
    TotalTokens            int                    `json:"total_tokens"`
    History                []*pb.Message          `json:"history"`
    CurrentTurn            []*pb.Message         `json:"current_turn"`
    AssistantResponse      string                 `json:"assistant_response"`
    FirstIterationToolCalls []*protocol.ToolCall `json:"first_iteration_tool_calls"`
    FinalResponseAdded     bool                   `json:"final_response_added"`
    Query                  string                 `json:"query"`
    AgentName              string                 `json:"agent_name"`
    SubAgents              []string               `json:"sub_agents"`
    ShowThinking           bool                   `json:"show_thinking"`
}

// SerializeExecutionState converts execution state to JSON
func SerializeExecutionState(state *ExecutionState) ([]byte, error) {
    return json.Marshal(state)
}

// DeserializeExecutionState reconstructs execution state from JSON
func DeserializeExecutionState(data []byte) (*ExecutionState, error) {
    var state ExecutionState
    if err := json.Unmarshal(data, &state); err != nil {
        return nil, err
    }
    return &state, nil
}

// CaptureExecutionState creates a snapshot of current execution state
func CaptureExecutionState(
    taskID string,
    contextID string,
    query string,
    reasoningState *reasoning.ReasoningState,
    pendingToolCall *protocol.ToolCall,
) *ExecutionState {
    return &ExecutionState{
        TaskID:          taskID,
        ContextID:       contextID,
        Query:           query,
        ReasoningState: &ReasoningStateSnapshot{
            Iteration:              reasoningState.Iteration(),
            TotalTokens:            reasoningState.TotalTokens(),
            History:                reasoningState.GetHistory(),
            CurrentTurn:            reasoningState.GetCurrentTurn(),
            AssistantResponse:      reasoningState.GetAssistantResponse(),
            FirstIterationToolCalls: reasoningState.GetFirstIterationToolCalls(),
            FinalResponseAdded:     reasoningState.IsFinalResponseAdded(),
            Query:                  reasoningState.Query(),
            AgentName:              reasoningState.AgentName(),
            SubAgents:              reasoningState.SubAgents(),
            ShowThinking:          reasoningState.ShowThinking(),
        },
        PendingToolCall:  pendingToolCall,
        History:          reasoningState.GetHistory(),
        CurrentTurn:      reasoningState.GetCurrentTurn(),
        Iteration:        reasoningState.Iteration(),
        TotalTokens:      reasoningState.TotalTokens(),
        AssistantResponse: reasoningState.GetAssistantResponse(),
    }
}

// RestoreReasoningState reconstructs a ReasoningState from snapshot
func (s *ExecutionState) RestoreReasoningState(
    outputCh chan<- *pb.Part,
    services reasoning.AgentServices,
    ctx context.Context,
) (*reasoning.ReasoningState, error) {
    state, err := reasoning.Builder().
        WithQuery(s.Query).
        WithAgentName(s.ReasoningState.AgentName).
        WithSubAgents(s.ReasoningState.SubAgents).
        WithOutputChannel(outputCh).
        WithShowThinking(s.ReasoningState.ShowThinking).
        WithServices(services).
        WithContext(ctx).
        WithHistory(s.History).
        Build()
    
    if err != nil {
        return nil, err
    }
    
    // Restore state fields
    for i := 0; i < s.Iteration; i++ {
        state.NextIteration()
    }
    
    for _, msg := range s.CurrentTurn {
        state.AddCurrentTurnMessage(msg)
    }
    
    state.AppendResponse(s.AssistantResponse)
    
    if s.ReasoningState.FinalResponseAdded {
        state.MarkFinalResponseAdded()
    }
    
    return state, nil
}
```

### Step 3: Add Session Metadata Helpers

Create helpers to manage execution state in session metadata:

```go
// pkg/agent/session_execution_state.go

package agent

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/kadirpekel/hector/pkg/reasoning"
)

const (
    pendingExecutionsKey = "pending_executions"
)

// SaveExecutionStateToSession saves execution state to session metadata
func (a *Agent) SaveExecutionStateToSession(
    ctx context.Context,
    sessionID string,
    taskID string,
    execState *ExecutionState,
) error {
    sessionService := a.services.Session()
    if sessionService == nil {
        return fmt.Errorf("session service not available")
    }

    // Get current session metadata
    metadata, err := sessionService.GetOrCreateSessionMetadata(sessionID)
    if err != nil {
        return fmt.Errorf("failed to get session metadata: %w", err)
    }

    // Initialize pending_executions if needed
    if metadata.Metadata == nil {
        metadata.Metadata = make(map[string]interface{})
    }
    
    pendingExecutions, exists := metadata.Metadata[pendingExecutionsKey]
    if !exists {
        pendingExecutions = make(map[string]interface{})
        metadata.Metadata[pendingExecutionsKey] = pendingExecutions
    }

    // Convert to map for manipulation
    pendingMap, ok := pendingExecutions.(map[string]interface{})
    if !ok {
        pendingMap = make(map[string]interface{})
        metadata.Metadata[pendingExecutionsKey] = pendingMap
    }

    // Serialize execution state
    stateJSON, err := SerializeExecutionState(execState)
    if err != nil {
        return fmt.Errorf("failed to serialize execution state: %w", err)
    }

    // Store as JSON string (metadata values must be JSON-serializable)
    var stateMap map[string]interface{}
    if err := json.Unmarshal(stateJSON, &stateMap); err != nil {
        return fmt.Errorf("failed to unmarshal execution state: %w", err)
    }

    pendingMap[taskID] = stateMap

    // Update session metadata
    metadataJSON, err := json.Marshal(metadata.Metadata)
    if err != nil {
        return fmt.Errorf("failed to marshal metadata: %w", err)
    }

    // Note: SessionService doesn't have UpdateMetadata method yet
    // You'll need to add: UpdateSessionMetadata(sessionID string, metadata map[string]interface{}) error
    // For now, we can work around by updating via AppendMessage with a metadata message
    // Or extend SessionService interface
    
    return nil
}

// LoadExecutionStateFromSession loads execution state from session metadata
func (a *Agent) LoadExecutionStateFromSession(
    ctx context.Context,
    sessionID string,
    taskID string,
) (*ExecutionState, error) {
    sessionService := a.services.Session()
    if sessionService == nil {
        return nil, fmt.Errorf("session service not available")
    }

    metadata, err := sessionService.GetOrCreateSessionMetadata(sessionID)
    if err != nil {
        return nil, fmt.Errorf("failed to get session metadata: %w", err)
    }

    if metadata.Metadata == nil {
        return nil, fmt.Errorf("no execution state found for task %s", taskID)
    }

    pendingExecutions, exists := metadata.Metadata[pendingExecutionsKey]
    if !exists {
        return nil, fmt.Errorf("no pending executions in session")
    }

    pendingMap, ok := pendingExecutions.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("invalid pending_executions format")
    }

    taskState, exists := pendingMap[taskID]
    if !exists {
        return nil, fmt.Errorf("execution state not found for task %s", taskID)
    }

    // Serialize back to JSON for deserialization
    stateJSON, err := json.Marshal(taskState)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal task state: %w", err)
    }

    return DeserializeExecutionState(stateJSON)
}

// ClearExecutionStateFromSession removes execution state after resuming
func (a *Agent) ClearExecutionStateFromSession(
    ctx context.Context,
    sessionID string,
    taskID string,
) error {
    sessionService := a.services.Session()
    if sessionService == nil {
        return fmt.Errorf("session service not available")
    }

    metadata, err := sessionService.GetOrCreateSessionMetadata(sessionID)
    if err != nil {
        return fmt.Errorf("failed to get session metadata: %w", err)
    }

    if metadata.Metadata == nil {
        return nil // Nothing to clear
    }

    pendingExecutions, exists := metadata.Metadata[pendingExecutionsKey]
    if !exists {
        return nil // Nothing to clear
    }

    pendingMap, ok := pendingExecutions.(map[string]interface{})
    if !ok {
        return nil
    }

    delete(pendingMap, taskID)

    // If no more pending executions, remove the key
    if len(pendingMap) == 0 {
        delete(metadata.Metadata, pendingExecutionsKey)
    }

    // Update session metadata (requires UpdateSessionMetadata method)
    // For now, this is a placeholder
    
    return nil
}
```

### Step 4: Modify handleToolApprovalRequest to Save State and Exit

Instead of blocking, save state to session metadata and exit:

```go
// Modified handleToolApprovalRequest in pkg/agent/agent.go

func (a *Agent) handleToolApprovalRequest(
    ctx context.Context,
    approvalResult *ToolApprovalResult,
    outputCh chan<- *pb.Part,
    reasoningState *reasoning.ReasoningState, // Add this parameter
) (context.Context, bool, error) {
    taskID := getTaskIDFromContext(ctx)
    if taskID == "" {
        // No taskID - can't request approval, deny the tool
        if sendErr := safeSendPart(ctx, outputCh, createTextPart("⚠️  Tool requires approval but task tracking not enabled, denying\n")); sendErr != nil {
            log.Printf("[Agent:%s] Failed to send approval denial message: %v", a.id, sendErr)
        }
        return ctx, false, nil
    }

    sessionID := getSessionIDFromContext(ctx) // context_id maps to session_id
    if sessionID == "" {
        return ctx, false, fmt.Errorf("session ID required for async HITL")
    }

    query := reasoningState.Query()

    // Capture execution state before pausing
    execState := CaptureExecutionState(
        taskID,
        sessionID,
        query,
        reasoningState,
        approvalResult.PendingToolCall,
    )

    // Save to session metadata (not task storage!)
    if err := a.SaveExecutionStateToSession(ctx, sessionID, taskID, execState); err != nil {
        return ctx, false, fmt.Errorf("failed to save execution state: %w", err)
    }

    // Update task to INPUT_REQUIRED state with approval request message
    if err := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_INPUT_REQUIRED, approvalResult.InteractionMsg); err != nil {
        return ctx, false, fmt.Errorf("updating task status: %w", err)
    }

    // Send approval request message parts to stream
    if approvalResult.InteractionMsg != nil && len(approvalResult.InteractionMsg.Parts) > 0 {
        for _, part := range approvalResult.InteractionMsg.Parts {
            if sendErr := safeSendPart(ctx, outputCh, part); sendErr != nil {
                log.Printf("[Agent:%s] Failed to send approval request part: %v", a.id, sendErr)
                return ctx, false, sendErr
            }
        }
    }

    // DON'T BLOCK - Return error to signal that execution should pause
    // The caller should exit the goroutine
    return ctx, false, ErrInputRequired // New error type
}
```

### Step 4: Modify Execution Loop to Handle Pause

Update the execution loop to handle the pause signal:

```go
// In pkg/agent/agent.go execute() or processTaskAsync()

func (a *Agent) processTaskAsync(taskID, userText, contextID string) {
    // ... existing setup code ...

    for {
        // Execute reasoning iteration
        shouldContinue, err := strategy.Execute(ctx, state)
        if err != nil {
            if err == ErrInputRequired {
                // Task paused for input - this is expected, not an error
                // State is already persisted, goroutine can exit
                log.Printf("[Agent:%s] Task %s paused for user input", a.id, taskID)
                return // Exit goroutine - task will resume when user provides input
            }
            // Handle other errors...
        }

        if !shouldContinue {
            break
        }
    }

    // ... completion code ...
}
```

### Step 5: Implement Resume Mechanism

When user provides input, resume execution:

```go
// Modified handleInputRequiredResume in pkg/agent/agent_a2a_methods.go

func (a *Agent) handleInputRequiredResume(ctx context.Context, userMessage *pb.Message) (bool, *pb.SendMessageResponse, error) {
    if userMessage.TaskId == "" || a.services.Task() == nil {
        return false, nil, nil
    }

    existingTask, err := a.services.Task().GetTask(ctx, userMessage.TaskId)
    if err != nil {
        return false, nil, nil
    }

    if existingTask.Status.State != pb.TaskState_TASK_STATE_INPUT_REQUIRED {
        return false, nil, nil
    }

    // Validate context ID matches
    if userMessage.ContextId != "" && existingTask.ContextId != "" && userMessage.ContextId != existingTask.ContextId {
        return true, nil, status.Errorf(codes.InvalidArgument, "context ID mismatch")
    }

    // Load execution state from session metadata
    sessionID := existingTask.ContextId // context_id maps to session_id
    execState, err := a.LoadExecutionStateFromSession(ctx, sessionID, userMessage.TaskId)
    if err != nil {
        return true, nil, status.Errorf(codes.Internal, "failed to load execution state: %v", err)
    }

    // Extract user decision
    decision := parseUserDecision(userMessage)

    // Resume task execution in background
    go a.resumeTaskExecution(userMessage.TaskId, execState, decision)

    return true, &pb.SendMessageResponse{
        Payload: &pb.SendMessageResponse_Task{
            Task: existingTask,
        },
    }, nil
}

// resumeTaskExecution continues execution from saved state
func (a *Agent) resumeTaskExecution(
    taskID string,
    execState *ExecutionState,
    userDecision string,
) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("[Agent:%s] PANIC resuming task %s: %v", a.id, taskID, r)
            ctx := context.Background()
            if updateErr := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil); updateErr != nil {
                log.Printf("[Agent:%s] Failed to update task %s status after panic: %v", a.id, taskID, updateErr)
            }
        }
    }()

    // Create context with taskID and user decision
    ctx := context.Background()
    ctx = EnsureAgentContext(ctx, taskID, execState.ContextID)
    ctx = context.WithValue(ctx, userDecisionContextKey, userDecision)

    // Create output channel
    outputCh := make(chan *pb.Part, outputChannelBuffer)
    defer close(outputCh)

    // Restore reasoning state
    strategy, err := reasoning.CreateStrategy(a.config.Reasoning.Engine, a.config.Reasoning)
    if err != nil {
        a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil)
        return
    }

    reasoningState, err := execState.RestoreReasoningState(
        outputCh,
        a.services,
        ctx,
    )
    if err != nil {
        a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_FAILED, nil)
        return
    }

    // Update task to WORKING
    if err := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_WORKING, nil); err != nil {
        log.Printf("[Agent:%s] Failed to update task %s to WORKING: %v", a.id, taskID, err)
    }

    // Continue execution from where it left off
    // The tool approval check will now see the user decision in context
    // and proceed accordingly
    for {
        shouldContinue, err := strategy.Execute(ctx, reasoningState)
        if err != nil {
            if err == ErrInputRequired {
                // Another approval needed - save state again
                log.Printf("[Agent:%s] Task %s paused again for user input", a.id, taskID)
                return
            }
            // Handle error...
            break
        }

        if !shouldContinue {
            break
        }
    }

    // Task completed
    finalResponse := reasoningState.GetAssistantResponse()
    responseMessage := a.createResponseMessage(finalResponse, execState.ContextID, taskID)
    
    if err := a.services.Task().AddTaskMessage(ctx, taskID, responseMessage); err != nil {
        log.Printf("[Agent:%s] Failed to add response message: %v", a.id, err)
    }

    if err := a.updateTaskStatus(ctx, taskID, pb.TaskState_TASK_STATE_COMPLETED, responseMessage); err != nil {
        log.Printf("[Agent:%s] Failed to update task status: %v", a.id, err)
    }

    // Clear execution state from session metadata
    if err := a.ClearExecutionStateFromSession(ctx, execState.ContextID, taskID); err != nil {
        log.Printf("[Agent:%s] Failed to clear execution state: %v", a.id, err)
    }
}
```

### Step 6: Extend SessionService Interface (Optional)

If you want a cleaner API, add a method to update session metadata:

```go
// In pkg/reasoning/interfaces.go

type SessionService interface {
    // ... existing methods ...
    
    // UpdateSessionMetadata updates session metadata
    UpdateSessionMetadata(sessionID string, metadata map[string]interface{}) error
}
```

Implement for SQL backend:

```go
// In pkg/memory/session_service_sql.go

func (s *SQLSessionService) UpdateSessionMetadata(sessionID string, metadata map[string]interface{}) error {
    metadataJSON, err := json.Marshal(metadata)
    if err != nil {
        return fmt.Errorf("failed to marshal metadata: %w", err)
    }

    query := `UPDATE sessions SET metadata = ?, updated_at = ? WHERE id = ? AND agent_id = ?`
    if s.dialect == "postgres" {
        query = `UPDATE sessions SET metadata = $1, updated_at = $2 WHERE id = $3 AND agent_id = $4`
    }

    _, err = s.db.ExecContext(context.Background(), query, string(metadataJSON), time.Now(), sessionID, s.agentID)
    return err
}
```

**Note**: You can work around this by using the existing `GetOrCreateSessionMetadata` and manually updating, but a dedicated method is cleaner.

### Step 7: Startup Recovery (Optional)

On server startup, detect and optionally resume waiting tasks:

```go
// In pkg/agent/agent.go

func (a *Agent) RecoverWaitingTasks(ctx context.Context) error {
    if a.services.Task() == nil {
        return nil
    }

    // Query all tasks in INPUT_REQUIRED state
    waitingTasks, err := a.services.Task().GetTasksByState(ctx, pb.TaskState_TASK_STATE_INPUT_REQUIRED)
    if err != nil {
        return fmt.Errorf("failed to query waiting tasks: %w", err)
    }

    log.Printf("[Agent:%s] Found %d tasks waiting for input", a.id, len(waitingTasks))

    // Optionally: Send notifications or mark as expired
    // For now, just log - tasks will resume when user provides input
    
    return nil
}
```

## Key Design Decisions

### 1. Why Session Metadata Instead of Task Storage?

**Session Metadata Approach** ✅ (Recommended)
- ✅ No schema changes needed
- ✅ Leverages existing session persistence infrastructure
- ✅ Natural relationship: Tasks → Sessions via `context_id`
- ✅ Automatic cleanup when session deleted
- ✅ Works with any session backend (SQL, in-memory)
- ✅ Multi-agent isolation already handled

**Task Storage Approach** ❌ (Alternative)
- ❌ Requires schema migration
- ❌ Duplicates persistence infrastructure
- ❌ Need to manage cleanup separately
- ❌ Less natural relationship

**Recommendation**: Use session metadata - it's cleaner and leverages existing infrastructure.

### 2. State Serialization Format

Use JSON for simplicity and debugging. Consider:
- **JSON**: Human-readable, easy to debug, sufficient for most cases
- **Protocol Buffers**: More compact, type-safe, but requires schema changes
- **MessagePack**: Binary format, smaller than JSON

**Recommendation**: Start with JSON, migrate to Protobuf if size becomes an issue.

### 3. What to Persist

**Must persist:**
- Reasoning state (iteration, tokens, history, current turn)
- Pending tool call
- Query and context
- Agent configuration (for state restoration)

**Don't persist:**
- Channels (can't serialize)
- Context (recreate on resume)
- Output channels (recreate on resume)

### 3. Error Handling

**On resume failure:**
- Log error
- Update task to FAILED state
- Optionally notify user

**On state corruption:**
- Detect invalid state
- Mark task as FAILED
- Provide error message to user

### 4. Timeout Handling

**Current approach**: Timeout cancels task
**Async approach**: Timeout marks task as expired, but state remains

**Options:**
1. Keep current behavior (fail on timeout)
2. Allow resuming expired tasks (with warning)
3. Auto-cancel expired tasks after grace period

## Testing Strategy

### Unit Tests

1. **State Serialization**
   ```go
   func TestSerializeExecutionState(t *testing.T) {
       state := CaptureExecutionState(...)
       json, err := SerializeExecutionState(state)
       assert.NoError(t, err)
       
       restored, err := DeserializeExecutionState(json)
       assert.NoError(t, err)
       assert.Equal(t, state, restored)
   }
   ```

2. **State Restoration**
   ```go
   func TestRestoreReasoningState(t *testing.T) {
       execState := &ExecutionState{...}
       restored, err := execState.RestoreReasoningState(...)
       assert.NoError(t, err)
       assert.Equal(t, originalState.Iteration(), restored.Iteration())
   }
   ```

### Integration Tests

1. **Pause and Resume**
   - Start task requiring approval
   - Verify state persisted
   - Restart server
   - Provide input
   - Verify task resumes correctly

2. **Multiple Pauses**
   - Task requiring multiple approvals
   - Verify each pause/resume cycle works

3. **Timeout Handling**
   - Start task requiring approval
   - Wait for timeout
   - Verify task state

## Migration Path

### Phase 1: Add Session Metadata Support (Non-Breaking)
- Add `SaveExecutionStateToSession` / `LoadExecutionStateFromSession` helpers
- Optionally extend `SessionService` with `UpdateSessionMetadata`
- Keep current blocking behavior (test new code path separately)

### Phase 2: Implement Non-Blocking (Breaking)
- Modify `handleToolApprovalRequest` to save to session metadata and exit
- Update execution loop to handle pause signal
- Implement resume mechanism using session metadata

### Phase 3: Add Recovery (Optional)
- Startup recovery for waiting tasks
- Expiration handling
- Monitoring and alerts
- Cleanup of stale execution states

## Web UI Implications

The web UI already handles HITL correctly, but with truly async HITL, there are some considerations:

### Current UI Behavior (Works with Async HITL)

✅ **TaskId Storage**: UI stores `taskId` in session when `INPUT_REQUIRED` is received  
✅ **Resume Support**: UI sends `taskId` when user approves/denies  
✅ **State Persistence**: UI saves sessions to localStorage, survives refresh  

### What Changes with Async HITL

#### 1. Resume After Refresh

**Current**: If server restarts, taskId is lost (goroutine gone)  
**With Async**: TaskId persists in UI localStorage, can resume after refresh

**UI Enhancement** (Optional):
```javascript
// On page load, check for pending tasks
async checkPendingTasks() {
    const session = this.sessions[this.currentSessionId];
    if (session.taskId) {
        // Check if task is still in INPUT_REQUIRED state
        const taskStatus = await this.getTaskStatus(session.taskId);
        if (taskStatus?.state === 'TASK_STATE_INPUT_REQUIRED') {
            // Show notification: "Task waiting for your input"
            this.showNotification('Task paused - waiting for your input');
        }
    }
}
```

#### 2. Visual State Indicators

**Current**: Shows approval widget, assumes goroutine is waiting  
**With Async**: Should indicate task is "paused" vs "active"

**UI Enhancement** (Optional):
```html
<!-- Show paused indicator -->
<div x-show="taskState === 'TASK_STATE_INPUT_REQUIRED'" 
     class="px-4 py-2 bg-yellow-500/20 border border-yellow-500/50 rounded mb-2">
    <div class="flex items-center gap-2 text-sm">
        <span>⏸️</span>
        <span>Task paused - waiting for your input</span>
    </div>
</div>
```

#### 3. Error Handling

**New Scenario**: Task state might be corrupted or missing after restart

**UI Enhancement**:
```javascript
async handleApproval(widgetId, decision) {
    try {
        // ... existing code ...
    } catch (error) {
        if (error.message.includes('execution state not found')) {
            // Task state lost - offer to restart task
            this.showError('Task state lost. Would you like to restart?');
            // Optionally: Clear taskId and start fresh
        }
    }
}
```

#### 4. Multiple Pauses

**Scenario**: Task could pause multiple times (multiple tool approvals)

**Current UI**: Already handles this - each approval widget is independent  
**No Changes Needed**: UI creates widgets for each approval request

### Required UI Changes

**Minimal Changes Required** ✅

The existing UI code should work with async HITL because:

1. ✅ UI already sends `taskId` when resuming (line 2013 in `handleApproval`)
2. ✅ UI already handles `INPUT_REQUIRED` state correctly
3. ✅ UI already stores `taskId` in session for persistence

**Optional Enhancements** (Better UX):

1. **Task Status Polling** (for refresh scenarios):
```javascript
// Poll task status if taskId exists but no active stream
if (session.taskId && !this.isGenerating) {
    this.pollTaskStatus(session.taskId);
}
```

2. **Visual Paused State**:
```html
<!-- Show when task is paused -->
<div x-show="session.taskId && taskState === 'INPUT_REQUIRED'">
    Task paused - click approve/deny to continue
</div>
```

3. **Resume After Restart**:
```javascript
// On page load
mounted() {
    this.checkPendingTasks();
    // If taskId exists and task is INPUT_REQUIRED, show resume option
}
```

### Backend Changes Required

The backend `handleInputRequiredResume` already works correctly:

```go
// Current: Unblocks waiting goroutine
a.taskAwaiter.ProvideInput(taskID, message)

// With Async: Starts new goroutine
go a.resumeTaskExecution(taskID, execState, decision)
```

**No UI changes needed** - the API contract is the same!

### Testing Checklist

- [ ] UI sends `taskId` when approving/denying ✅ (Already works)
- [ ] UI handles `INPUT_REQUIRED` status updates ✅ (Already works)
- [ ] UI persists `taskId` across refresh ✅ (Already works)
- [ ] UI can resume after server restart ✅ (Works with async HITL)
- [ ] UI handles multiple pauses ✅ (Already works)
- [ ] UI shows error if state corrupted (Optional enhancement)

### Summary

**Good News**: The web UI already supports async HITL! The existing code:
- Stores taskId correctly
- Sends taskId when resuming
- Handles INPUT_REQUIRED state

**Optional Enhancements**:
- Show "paused" indicator after refresh
- Poll task status on page load
- Better error handling for corrupted state

**No Breaking Changes**: The API contract remains the same, so existing UI code continues to work.

---

## Configuration Design: Session Persistence & HITL Dependency

With async HITL storing execution state in session metadata, there's a dependency relationship to design. Here are the design options:

### Design Principles

1. **Backward Compatibility**: Existing blocking HITL should continue to work
2. **Explicit Control**: Users should be able to choose blocking vs async
3. **Sensible Defaults**: Auto-detect best mode when possible
4. **Clear Errors**: Fail fast with helpful error messages

### Recommended Design: Auto-Detect with Explicit Override

**Principle**: Auto-detect async HITL when session persistence is configured, but allow explicit override.

```yaml
agents:
  assistant:
    llm: "gpt-4o"
    
    # Session persistence (optional)
    session_store: "main-db"  # If present, enables async HITL automatically
    
    # Task configuration
    task:
      backend: "memory"
      input_timeout: 600
      
      # Optional: Explicit HITL mode override
      hitl:
        mode: "auto"  # "auto" | "blocking" | "async"
        # "auto" (default): async if session_store exists, blocking otherwise
        # "blocking": always blocking (current behavior)
        # "async": always async (requires session_store)
```

### Behavior Matrix

| `session_store` | `task.hitl.mode` | Result |
|-----------------|------------------|--------|
| Not configured | `auto` (default) | Blocking HITL |
| Not configured | `blocking` | Blocking HITL |
| Not configured | `async` | ❌ Error: requires session_store |
| Configured | `auto` (default) | Async HITL |
| Configured | `blocking` | Blocking HITL (override) |
| Configured | `async` | Async HITL |

### Configuration Examples

#### Example 1: Blocking HITL (Current Behavior)

```yaml
agents:
  assistant:
    llm: "gpt-4o"
    task:
      backend: "memory"
      input_timeout: 600
    tools:
      - execute_command  # requires_approval: true
```

**Behavior**: Goroutine blocks waiting for input, lost on restart.

#### Example 2: Async HITL (Auto-Detected)

```yaml
session_stores:
  main-db:
    backend: sql
    sql:
      driver: sqlite
      database: ./sessions.db

agents:
  assistant:
    llm: "gpt-4o"
    session_store: "main-db"  # ← Enables async HITL automatically
    task:
      backend: "memory"
      input_timeout: 600
    tools:
      - execute_command
```

**Behavior**: Execution state saved to session metadata, survives restart.

#### Example 3: Explicit Async Mode

```yaml
session_stores:
  main-db:
    backend: sql
    sql:
      driver: sqlite
      database: ./sessions.db

agents:
  assistant:
    llm: "gpt-4o"
    session_store: "main-db"
    task:
      backend: "memory"
      input_timeout: 600
      hitl:
        mode: "async"  # Explicit async
    tools:
      - execute_command
```

**Behavior**: Same as Example 2, but explicit.

#### Example 4: Force Blocking (Even with Session Store)

```yaml
session_stores:
  main-db:
    backend: sql
    sql:
      driver: sqlite
      database: ./sessions.db

agents:
  assistant:
    llm: "gpt-4o"
    session_store: "main-db"
    task:
      backend: "memory"
      input_timeout: 600
      hitl:
        mode: "blocking"  # Force blocking mode
    tools:
      - execute_command
```

**Behavior**: Uses blocking HITL even though session_store exists (useful for testing).

#### Example 5: Invalid Configuration

```yaml
agents:
  assistant:
    llm: "gpt-4o"
    # No session_store configured
    task:
      backend: "memory"
      hitl:
        mode: "async"  # ❌ Error: async requires session_store
```

**Error**: `async HITL requires session_store to be configured`

### Implementation

**Configuration Structure**:

```go
type TaskConfig struct {
    Backend      string         `yaml:"backend,omitempty"`
    WorkerPool   int            `yaml:"worker_pool,omitempty"`
    SQL          *TaskSQLConfig `yaml:"sql,omitempty"`
    InputTimeout int            `yaml:"input_timeout,omitempty"`
    Timeout      int            `yaml:"timeout,omitempty"`
    HITL         *HITLConfig    `yaml:"hitl,omitempty"`  // New field
}

type HITLConfig struct {
    Mode string `yaml:"mode,omitempty"` // "auto" (default), "blocking", or "async"
}
```

**Validation Logic**:

```go
func (a *Agent) validateHITLConfig() error {
    taskCfg := a.config.Task
    if taskCfg == nil {
        return nil // No task config, no HITL
    }
    
    hitlMode := "auto" // Default
    if taskCfg.HITL != nil && taskCfg.HITL.Mode != "" {
        hitlMode = taskCfg.HITL.Mode
    }
    
    hasSessionStore := a.services.Session() != nil
    
    switch hitlMode {
    case "async":
        if !hasSessionStore {
            return fmt.Errorf("async HITL requires session_store to be configured")
        }
    case "blocking":
        // Always allowed, even if session_store exists
    case "auto":
        // Auto-detect: no validation needed
    default:
        return fmt.Errorf("invalid hitl.mode: %s (must be 'auto', 'blocking', or 'async')", hitlMode)
    }
    
    return nil
}
```

**Runtime Behavior**:

```go
func (a *Agent) shouldUseAsyncHITL() bool {
    taskCfg := a.config.Task
    mode := "auto" // Default
    if taskCfg != nil && taskCfg.HITL != nil {
        mode = taskCfg.HITL.Mode
    }
    
    hasSessionStore := a.services.Session() != nil
    
    switch mode {
    case "async":
        return true // Explicit async
    case "blocking":
        return false // Explicit blocking
    case "auto":
        return hasSessionStore // Auto-detect
    default:
        return hasSessionStore // Fallback to auto-detect
    }
}
```

### Migration Guide

#### From Blocking to Async HITL

**Before** (Blocking):
```yaml
agents:
  assistant:
    llm: "gpt-4o"
    task:
      backend: "memory"
      input_timeout: 600
```

**After** (Async - just add session_store):
```yaml
session_stores:
  main-db:
    backend: sql
    sql:
      driver: sqlite
      database: ./sessions.db

agents:
  assistant:
    llm: "gpt-4o"
    session_store: "main-db"  # ← Add this line
    task:
      backend: "memory"
      input_timeout: 600
```

**No other changes needed!** Async HITL is enabled automatically.

### Summary

**Recommended Design**:
- ✅ **Auto-detect by default**: If `session_store` exists → async HITL
- ✅ **Explicit override**: `task.hitl.mode` for explicit control
- ✅ **Backward compatible**: No config changes needed for existing setups
- ✅ **Clear errors**: Validation fails fast with helpful messages

**Configuration Priority**:
1. `task.hitl.mode: "async"` → Requires `session_store`
2. `task.hitl.mode: "blocking"` → Always blocking
3. `task.hitl.mode: "auto"` (default) → Auto-detect from `session_store`
4. No `task.hitl` config → Auto-detect (same as "auto")

---

## Should We Maintain Both Modes?

This is an important architectural decision. Here's a balanced analysis:

### Arguments FOR Maintaining Both Modes

#### 1. **Different Use Cases Exist**

**Blocking HITL** is better for:
- ✅ **Development/Testing**: Simpler, faster iteration, easier debugging
- ✅ **Short-lived approvals**: Quick decisions (< 1 minute), no persistence needed
- ✅ **Single-user scenarios**: No need for restart resilience
- ✅ **Performance-critical**: Zero serialization overhead

**Async HITL** is better for:
- ✅ **Production**: Must survive restarts, deployments, crashes
- ✅ **Long-running approvals**: User might respond hours/days later
- ✅ **Multi-user systems**: Shared infrastructure, need isolation
- ✅ **High availability**: Can't lose state on restart

#### 2. **Performance Considerations**

**Blocking Mode Overhead**:
```
Tool approval needed
  ↓ (0ms - no serialization)
Goroutine blocks on channel
  ↓ (wait for user input)
Resume immediately
Total overhead: ~0ms
```

**Async Mode Overhead**:
```
Tool approval needed
  ↓ (1-5ms - serialize execution state)
Save to session metadata
  ↓ (2-10ms - database write)
Goroutine exits
  ↓ (user provides input later)
Load execution state
  ↓ (2-10ms - database read + deserialize)
Resume execution
Total overhead: ~5-25ms per pause/resume cycle
```

**Impact**: For high-frequency approvals, blocking is faster. For production reliability, async is required.

#### 3. **Backward Compatibility**

- ✅ Existing configs continue to work
- ✅ No breaking changes
- ✅ Gradual migration path
- ✅ Users can test async before committing

#### 4. **Testing & Debugging**

**Blocking Mode**:
- ✅ Simpler to test (no state persistence)
- ✅ Easier to debug (state in memory)
- ✅ Faster test execution
- ✅ No database setup needed

**Async Mode**:
- ⚠️ Requires database setup
- ⚠️ More complex state management
- ⚠️ Harder to debug (state in database)

### Arguments AGAINST Maintaining Both Modes

#### 1. **Code Complexity**

**Maintenance Burden**:
- Two code paths to maintain
- Two sets of tests
- Two sets of edge cases
- Potential for bugs in one mode but not the other

**Code Duplication**:
```go
// Need to maintain both:
func handleBlockingHITL(...) { ... }
func handleAsyncHITL(...) { ... }
```

#### 2. **User Confusion**

- Which mode should I use?
- What's the difference?
- Why does it matter?
- When do I need to switch?

#### 3. **Testing Burden**

- Need to test both modes
- Need to test mode switching
- Need to test edge cases in both
- More test code to maintain

### Recommended Approach: Maintain Both, But With Clear Guidance

**Strategy**: Keep both modes, but make async the default for production and provide clear guidance.

#### 1. **Default Behavior**

```yaml
# Development (no session_store) → Blocking (simple, fast)
agents:
  dev-agent:
    task:
      input_timeout: 600

# Production (with session_store) → Async (reliable, persistent)
session_stores:
  main-db: ...

agents:
  prod-agent:
    session_store: "main-db"  # Auto-enables async
    task:
      input_timeout: 600
```

#### 2. **Clear Documentation**

**When to Use Blocking**:
- Development and testing
- Short-lived approvals (< 1 minute)
- Single-user scenarios
- Performance-critical paths

**When to Use Async**:
- Production deployments
- Long-running approvals
- Multi-user systems
- High availability requirements

#### 3. **Migration Path**

**Phase 1** (Current): Both modes supported, blocking default
**Phase 2** (Future): Async default when session_store exists
**Phase 3** (Future): Deprecate blocking for production (but keep for dev)

#### 4. **Implementation Strategy**

**Shared Core Logic**:
```go
// Common approval request logic
func (a *Agent) requestToolApproval(...) (*ToolApprovalResult, error) {
    // Shared logic for creating approval request
    // Used by both blocking and async modes
}

// Mode-specific handlers
func (a *Agent) handleToolApprovalRequest(...) {
    if a.shouldUseAsyncHITL() {
        return a.handleAsyncHITL(...)
    } else {
        return a.handleBlockingHITL(...)
    }
}
```

**Benefits**:
- ✅ Shared logic reduces duplication
- ✅ Mode-specific code is isolated
- ✅ Easy to test both paths
- ✅ Clear separation of concerns

### Long-Term Recommendation

**Keep Both Modes** ✅, but:

1. **Make async the default** when session_store exists (already in design)
2. **Document clearly** when to use each mode
3. **Deprecate blocking for production** (but keep for development)
4. **Provide migration tools** to help users switch

**Rationale**:
- Different use cases justify different modes
- Performance matters for high-frequency scenarios
- Backward compatibility is important
- Code complexity is manageable with shared core logic

### Alternative: Async-Only Approach

**If we only supported async**:

**Pros**:
- ✅ Simpler codebase (one path)
- ✅ No user confusion
- ✅ Consistent behavior
- ✅ Always production-ready

**Cons**:
- ❌ Requires session_store even for development
- ❌ Overhead even when not needed
- ❌ Breaking change for existing users
- ❌ Harder to test (requires database)

**Verdict**: Not recommended - the flexibility is worth the complexity.

### Final Recommendation

**Maintain both modes** with this strategy:

1. ✅ **Auto-detect by default** (async when session_store exists)
2. ✅ **Allow explicit override** (`task.hitl.mode`)
3. ✅ **Clear documentation** on when to use each
4. ✅ **Shared core logic** to minimize duplication
5. ✅ **Deprecate blocking for production** (but keep for dev/testing)

This gives users flexibility while guiding them toward the right choice for their use case.

---

## Limitations

⚠️ **State Size**: Large reasoning states increase session metadata size  
⚠️ **Complexity**: More moving parts to maintain  
⚠️ **Debugging**: Harder to debug paused tasks  
⚠️ **Session Required**: Requires session persistence configured (but this is common)  
⚠️ **Metadata Limits**: Some databases have TEXT field size limits (usually not an issue)

## Summary

Making HITL truly asynchronous requires:

1. **Persist execution state** when entering INPUT_REQUIRED (using session metadata)
2. **Don't block goroutines** - save state and exit
3. **Resume mechanism** - reconstruct state and continue
4. **State serialization** - convert reasoning state to/from storage format

**Key Advantage**: By leveraging existing session persistence infrastructure instead of creating separate storage, you get:
- ✅ No schema migrations
- ✅ Automatic cleanup
- ✅ Natural task-session relationship
- ✅ Reuse of existing persistence code

This enables tasks to pause and resume at any time, even after server restarts, providing a truly asynchronous human-in-the-loop experience.

