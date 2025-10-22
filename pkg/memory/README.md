# Memory System - Ownership and Architecture

## Overview

Hector's memory system manages conversation state across three distinct layers, each with clear ownership and responsibilities.

---

## 🏗️ Three-Layer Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Application                            │
│                      (Agent.execute)                        │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ▼
         ┌───────────────────────────────────────┐
         │   WORKING MEMORY (In-Memory)          │
         │   - ReasoningState.history            │
         │   - ReasoningState.currentTurn        │
         │   - Cached during turn execution      │
         │   - Ephemeral (lost on restart)       │
         └───────────┬───────────────────────────┘
                     │                     ▲
                     │ Load (once)         │ Save (batch)
                     │                     │
                     ▼                     │
         ┌───────────────────────────────────────┐
         │   SESSION STORE (SQL)                 │
         │   - Persistent message storage        │
         │   - Source of truth                   │
         │   - Survives restarts                 │
         │   - Strategy-managed loading          │
         └───────────┬───────────────────────────┘
                     │                     ▲
                     │ Checkpoint          │ Store (batch)
                     │                     │
                     ▼                     │
         ┌───────────────────────────────────────┐
         │   LONG-TERM MEMORY (Vector DB)        │
         │   - Semantic search                   │
         │   - Recall by similarity              │
         │   - Optional (can be disabled)        │
         └───────────────────────────────────────┘
```

---

## 📋 Layer Responsibilities

### 1. Working Memory (In-Memory)

**Purpose:** Active conversation context for current turn

**Components:**
- `ReasoningState.history` - Cached messages from previous turns
- `ReasoningState.currentTurn` - Messages being created in this turn
- `ConversationHistory` - Helper for in-memory message management

**Lifecycle:**
1. Load once at turn start from Session Store
2. Used throughout turn execution
3. Saved as batch at turn end
4. Discarded after turn

**Ownership:** `Agent` (via `ReasoningState`)

**When to use:**
- Building prompts for LLM
- Accumulating assistant responses
- Managing tool call conversation
- Current turn execution

**Key Principle:** 
> Working memory is **ephemeral** - it's a view of the conversation for this turn only

---

### 2. Session Store (SQL Database)

**Purpose:** Persistent source of truth for all messages

**Components:**
- `SessionService` interface
- `SQLSessionService` implementation (default)
- `InMemorySessionService` implementation (testing)

**Storage Schema:**
```sql
sessions (
    id TEXT PRIMARY KEY,
    agent_id TEXT,
    created_at INTEGER,
    last_accessed INTEGER
)

session_messages (
    id INTEGER PRIMARY KEY,
    session_id TEXT,
    message_id TEXT,
    message_json TEXT,
    role TEXT,
    sequence_num INTEGER
)
```

**Lifecycle:**
1. Created on first message
2. Grows with each turn (batch append)
3. Loaded strategically (checkpoint-based or window-based)
4. Persists across restarts

**Ownership:** `MemoryService`

**When to use:**
- Persisting messages permanently
- Loading conversation history
- Session resumption after restart
- Multi-agent isolation (agent_id + session_id)

**Key Principle:**
> Session Store is the **source of truth** - all messages must be saved here

---

### 3. Long-Term Memory (Vector Database)

**Purpose:** Semantic search and recall

**Components:**
- `LongTermMemoryStrategy` interface
- `VectorMemoryStrategy` implementation

**Storage:**
- Vector embeddings of messages
- Metadata: `agent_id`, `session_id`, `role`, `timestamp`
- Indexed for similarity search

**Lifecycle:**
1. Messages batched during turn
2. Flushed at batch size or shutdown
3. Recalled by semantic similarity
4. Optional (can be nil)

**Ownership:** `MemoryService`

**When to use:**
- Semantic search across conversations
- Recall relevant past messages
- Cross-session context
- Knowledge retrieval

**Key Principle:**
> Long-term memory is **optional** - it enhances recall but isn't required

---

## 🔄 Data Flow

### Save Flow (Turn Boundary)

```
Agent.saveToHistory()
    │
    ├─> Collect messages from currentTurn
    │
    ├─> Filter empty messages
    │
    └─> MemoryService.AddBatchToHistory()
            │
            ├─> SessionService.AppendMessage() [FOR EACH MESSAGE]
            │   └─> SQL: INSERT INTO session_messages
            │
            ├─> WorkingMemory.LoadState() [EFFICIENT, CHECKPOINT-AWARE]
            │   └─> SQL: SELECT ... FROM last checkpoint
            │
            ├─> WorkingMemory.CheckAndSummarize()
            │   └─> Returns new messages (e.g., summary)
            │
            ├─> SessionService.AppendMessage() [FOR STRATEGY MESSAGES]
            │   └─> SQL: INSERT (summary for checkpoint)
            │
            └─> addToLongTermBatch() [IF ENABLED]
                └─> Batch messages for vector storage
```

**Key Points:**
- ✅ Batch operation (atomic)
- ✅ Summarization checked ONCE per turn
- ✅ Strategy-generated messages persisted
- ✅ Thread-safe (mutex protected)

---

### Load Flow (Turn Start)

```
Agent.execute()
    │
    ├─> HistoryService.GetRecentHistory(sessionID)
    │   └─> MemoryService.GetRecentHistory()
    │       └─> WorkingMemory.LoadState(sessionID, sessionService)
    │           │
    │           ├─> [summary_buffer strategy]
    │           │   ├─> Load ALL messages
    │           │   ├─> Find last summary (checkpoint)
    │           │   └─> Return summary + messages after it
    │           │
    │           └─> [buffer_window strategy]
    │               └─> Load last N messages
    │
    └─> state.SetHistory(recentHistory)
        └─> Cached for this turn
```

**Key Points:**
- ✅ Loaded ONCE per turn
- ✅ Strategy decides what to load (checkpoint-aware)
- ✅ Cached in ReasoningState
- ✅ NOT reloaded during prompt building

---

## 🎯 Ownership Rules

### Rule 1: Single Source of Truth
**Session Store is the ONLY source of truth**
- Working memory is a cache
- Long-term memory is a search index
- Always save to Session Store first

### Rule 2: Load Once, Use Many
**History loaded once at turn start**
- Cached in `ReasoningState.history`
- Reused during prompt building
- NOT reloaded from database

### Rule 3: Save Once, At Turn End
**All messages saved as single batch**
- No incremental saves during turn
- Summarization checked once
- Atomic operation (all or nothing with transactions)

### Rule 4: Strategy Decides Loading
**Working memory strategy controls what to load**
- `summary_buffer`: Load from checkpoint
- `buffer_window`: Load last N messages
- Extensible for new strategies

### Rule 5: Agent Isolation
**All storage includes agent_id**
- Session Store: `agent_id + session_id`
- Long-term Memory: metadata includes `agent_id`
- Prevents cross-agent leaks

---

## 🚫 Anti-Patterns (What NOT to Do)

### ❌ Don't Load History in PromptService
```go
// BAD: Redundant loading
func BuildMessages(...) {
    history, _ := historyService.GetRecentHistory(sessionID)
    messages = append(messages, history...)
    messages = append(messages, currentToolConversation...) // Already has history!
}
```

```go
// GOOD: Use cached history
func BuildMessages(...) {
    // currentToolConversation already contains history from state.AllMessages()
    messages = append(messages, currentToolConversation...)
}
```

### ❌ Don't Save Messages One-by-One
```go
// BAD: Multiple saves, multiple summarization checks
for _, msg := range messages {
    memoryService.AddToHistory(sessionID, msg) // ❌ Called N times
}
```

```go
// GOOD: Single batch save
memoryService.AddBatchToHistory(sessionID, messages) // ✅ Called once
```

### ❌ Don't Mix State Sources
```go
// BAD: Inconsistent state
history1 := state.GetHistory()              // From cache
history2 := sessionService.GetMessages(...)  // From database
// Which one is correct? Confusion!
```

```go
// GOOD: Single source during turn
history := state.GetHistory() // Always use cached version during turn
```

---

## 📊 Component Responsibilities Summary

| Component | Owns | Reads From | Writes To | Lifetime |
|-----------|------|------------|-----------|----------|
| **Agent** | Turn execution | ReasoningState | SessionService | Per-request |
| **ReasoningState** | Working memory cache | SessionService (via MemoryService) | Nothing | Per-turn |
| **MemoryService** | Persistence coordination | SessionService | SessionService, LongTermMemory | Application |
| **WorkingMemoryStrategy** | Load decisions | SessionService | Nothing | Application |
| **SessionService** | Message storage | SQL Database | SQL Database | Application |
| **LongTermMemoryStrategy** | Semantic index | Vector Database | Vector Database | Application |

---

## 🧪 Testing Ownership

### Unit Tests
```go
// Test: Load once per turn
func TestLoadOnce(t *testing.T) {
    // 1. Mock SessionService that counts GetMessages calls
    // 2. Execute turn
    // 3. Assert: GetMessages called exactly ONCE
}
```

### Integration Tests
```go
// Test: State consistency
func TestStateConsistency(t *testing.T) {
    // 1. Save messages
    // 2. Restart service
    // 3. Load messages
    // 4. Assert: Messages match exactly
}
```

---

## 🔍 Debugging Guide

### "History not loading"
**Check:** Is history cached in ReasoningState?
```go
// In agent.execute()
if len(recentHistory) > 0 {
    state.SetHistory(recentHistory) // ← Must be called
}
```

### "Duplicate messages in prompt"
**Check:** Is PromptService loading history again?
```go
// PromptService should NOT do this:
historyMsgs, _ := historyService.GetRecentHistory(sessionID) // ❌ Redundant
```

### "Summarization not working"
**Check:** Are strategy messages persisted?
```go
// After CheckAndSummarize()
if len(newMessages) > 0 {
    for _, msg := range newMessages {
        sessionService.AppendMessage(sessionID, msg) // ← Must persist
    }
}
```

---

## 📚 Related Documentation

- **Core Concepts:** [`docs/core-concepts/memory.md`](../../docs/core-concepts/memory.md)
- **Sessions:** [`docs/core-concepts/sessions.md`](../../docs/core-concepts/sessions.md)
- **Configuration:** [`docs/reference/configuration.md`](../../docs/reference/configuration.md)
- **Setup Guide:** [`docs/how-to/setup-session-persistence.md`](../../docs/how-to/setup-session-persistence.md)

---

## ✅ Summary

**Three Layers:**
1. **Working Memory** - Ephemeral cache (ReasoningState)
2. **Session Store** - Persistent truth (SQL)
3. **Long-Term Memory** - Semantic index (Vector DB)

**Five Rules:**
1. Session Store is source of truth
2. Load once per turn
3. Save once at turn end
4. Strategy decides loading
5. Agent isolation everywhere

**Goal:** Clear ownership, no confusion, efficient operations, zero data loss

---

**Last Updated:** October 22, 2025  
**Version:** 1.0.0

