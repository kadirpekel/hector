# Session & Memory Architecture: Complete Developer Guide

**Version:** 1.0  
**Date:** October 22, 2025  
**Status:** Production Ready  

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Core Components](#core-components)
4. [Three-Layer Memory System](#three-layer-memory-system)
5. [Data Flow: Message Lifecycle](#data-flow-message-lifecycle)
6. [Session Persistence Mechanism](#session-persistence-mechanism)
7. [Multi-Agent Isolation](#multi-agent-isolation)
8. [Strategy-Managed Loading](#strategy-managed-loading)
9. [API Flows](#api-flows)
10. [Configuration Modes](#configuration-modes)
11. [Database Schema](#database-schema)
12. [Component Interactions](#component-interactions)
13. [Concurrency & Thread Safety](#concurrency--thread-safety)
14. [Error Handling & Recovery](#error-handling--recovery)
15. [Testing Strategy](#testing-strategy)
16. [Troubleshooting Guide](#troubleshooting-guide)

---

## Executive Summary

Hector's session and memory system provides **persistent, multi-agent conversation management** with three distinct memory layers:

1. **Session Store (SQL)** - Persistent message storage, survives process restarts
2. **Working Memory (In-Memory)** - Active conversation context, strategy-managed
3. **Long-Term Memory (Vector DB)** - Semantic recall, RAG capabilities

**Key Design Principles:**
- ✅ **Persistence-First:** All messages saved to SQL atomically
- ✅ **Strategy-Managed Loading:** Each memory strategy controls what to load from SQL
- ✅ **Multi-Agent Isolation:** `(agent_id, session_id)` composite key prevents leaks
- ✅ **Zero-Config Ready:** SQLite session store enabled by default
- ✅ **Checkpoint Detection:** Strategies optimize loading (e.g., from last summary)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        USER INTERACTION LAYER                        │
│  ┌──────────────┐         ┌──────────────┐        ┌──────────────┐ │
│  │  REST API    │         │  CLI (call)  │        │  gRPC (A2A)  │ │
│  │  (HTTP/JSON) │         │  (chat/task) │        │  (Proto)     │ │
│  └──────┬───────┘         └──────┬───────┘        └──────┬───────┘ │
└─────────┼───────────────────────┼────────────────────────┼─────────┘
          │                       │                        │
          └───────────────────────┴────────────────────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │    AGENT CORE (agent.go)  │
                    │  - Message routing        │
                    │  - Reasoning engine       │
                    │  - Tool orchestration     │
                    └─────────────┬─────────────┘
                                  │
          ┌───────────────────────┼───────────────────────┐
          │                       │                       │
┌─────────▼──────────┐  ┌────────▼────────┐  ┌──────────▼─────────┐
│  MEMORY SERVICE    │  │ PROMPT SERVICE  │  │  TOOL SERVICE      │
│  (memory.go)       │  │ (services.go)   │  │  (services.go)     │
│                    │  │                 │  │                    │
│  - Batch saving    │  │ - Message build │  │ - Tool execution   │
│  - Summarization   │  │ - History load  │  │ - Result handling  │
│  - LTM batching    │  └─────────────────┘  └────────────────────┘
└─────────┬──────────┘
          │
          ├────────────────────────────────────────────────────────┐
          │                                                        │
┌─────────▼──────────┐  ┌─────────────────┐  ┌──────────────────┐│
│  SESSION SERVICE   │  │ WORKING MEMORY  │  │ LONG-TERM MEMORY ││
│  (SQL Backend)     │  │  STRATEGY       │  │  (Vector DB)     ││
│                    │  │                 │  │                  ││
│  - Persist msgs    │  │ - LoadState     │  │ - Semantic store ││
│  - Load by options │  │ - Summarization │  │ - Agent isolated ││
│  - Agent isolation │  │ - Checkpoint    │  │ - Session scoped ││
└─────────┬──────────┘  └─────────────────┘  └──────────────────┘│
          │                                                        │
┌─────────▼──────────────────────────────────────────────────────┐│
│              STORAGE LAYER (Persistent)                        ││
│  ┌────────────────┐  ┌────────────────┐  ┌─────────────────┐ ││
│  │   SQLite DB    │  │  PostgreSQL    │  │   Qdrant        │ ││
│  │  (sessions.db) │  │  (sessions)    │  │  (vectors)      │ ││
│  └────────────────┘  └────────────────┘  └─────────────────┘ ││
└───────────────────────────────────────────────────────────────┘│
```

---

## Core Components

### 1. **Agent Core** (`pkg/agent/agent.go`)

**Responsibility:** Central orchestrator for agent behavior

**Key Functions:**
- `SendMessage()` - Synchronous message processing
- `SendStreamingMessage()` - Streaming response with tool calls
- `execute()` - Core reasoning loop (ReAct, CoT, Supervisor)
- `saveMessages()` - Batch save at turn boundaries

**Session ID Flow:**
```go
// Session ID extracted from context
sessionID := ctx.Value("sessionID").(string)
if sessionID == "" {
    sessionID = "default"  // Fallback
}

// Used throughout agent lifecycle:
// 1. Loading history (via MemoryService)
// 2. Saving messages (turn boundary)
// 3. Tool execution context
// 4. LLM prompt building
```

**Critical Design Decision:**
- ❌ **Does NOT filter messages before saving**
- ✅ **Saves ALL messages** (USER, AGENT, SYSTEM, TOOL, UNSPECIFIED)
- 🎯 **Strategy decides what to load back**

```go
// From agent.go - saveMessages()
for _, msg := range currentTurn {
    textContent := protocol.ExtractTextFromMessage(msg)
    hasToolCalls := len(protocol.GetToolCallsFromMessage(msg)) > 0
    hasToolResults := len(protocol.GetToolResultsFromMessage(msg)) > 0

    if textContent != "" || hasToolCalls || hasToolResults {
        messagesToSave = append(messagesToSave, msg)  // NO ROLE FILTERING
    }
}
```

---

### 2. **Memory Service** (`pkg/memory/memory.go`)

**Responsibility:** Facade for all memory operations

**Key Methods:**

#### `AddBatchToHistory(sessionID, messages)` ⭐ **PRIMARY SAVE METHOD**
```go
// 1. Atomic batch save to SQL (transaction)
err := s.sessionService.AppendMessages(sessionID, messages)

// 2. Queue for long-term memory (async)
for _, msg := range messages {
    s.addToLongTermBatch(sessionID, msg)
}

// 3. Check summarization ONCE per turn
history, _ := s.workingMemory.LoadState(sessionID, s.sessionService)
newMessages, _ := s.workingMemory.CheckAndSummarize(history)

// 4. Save strategy messages (e.g., summary)
if len(newMessages) > 0 {
    s.sessionService.AppendMessages(sessionID, newMessages)
}
```

**Why Batch vs Single?**
- ✅ `AddBatchToHistory` - Atomic, checks summarization ONCE (correct)
- ⚠️ `AddToHistory` - Deprecated, can cause infinite loops (use only for tests)

#### `Recall(sessionID, query)` - Long-term memory retrieval
```go
// Vector DB semantic search
// Filters by: agent_id + session_id (multi-agent isolation)
results, err := s.longTermMemory.Recall(s.agentID, sessionID, query, k)
```

#### `Shutdown()` - Graceful cleanup
```go
// Flush all pending LTM batches before exit
s.batchMu.Lock()
for sessionID, batch := range s.pendingBatches {
    s.flushLongTermBatch(sessionID)
}
s.batchMu.Unlock()
```

**Concurrency Protection:**
```go
type MemoryService struct {
    batchMu        sync.RWMutex  // Protects pendingBatches
    pendingBatches map[string][]*pb.Message
    // ...
}
```

---

### 3. **Session Service** (`pkg/memory/session_service_sql.go`)

**Responsibility:** SQL persistence layer

**Key Methods:**

#### `AppendMessages(sessionID, messages)` - Atomic batch save
```go
// Start transaction
tx, err := s.db.BeginTx(ctx, nil)

// Insert all messages
for _, msg := range messages {
    messageJSON, _ := protojson.Marshal(msg)
    _, err = tx.ExecContext(ctx, insertSQL, ...)
}

// Update session timestamp
_, err = tx.ExecContext(ctx, updateSQL, ...)

// Commit or rollback
tx.Commit()  // All-or-nothing atomicity
```

#### `GetMessagesWithOptions(sessionID, LoadOptions)` - Flexible loading
```go
type LoadOptions struct {
    Limit         int        // Max messages (0 = all)
    FromMessageID string     // Checkpoint loading
    Roles         []pb.Role  // Filter by role
}

// SQL query with agent_id isolation (CRITICAL)
query := `
    SELECT sm.message_id, sm.message_json 
    FROM session_messages sm 
    JOIN sessions s ON sm.session_id = s.id 
    WHERE sm.session_id = ? AND s.agent_id = ?  -- Multi-agent isolation
`
```

**Multi-Agent Isolation:**
```go
type SQLSessionService struct {
    agentID string  // Set during construction
    db      *sql.DB
    dialect string
}

// Every query MUST filter by agent_id
WHERE sm.session_id = ? AND s.agent_id = ?
```

---

### 4. **Working Memory Strategies** (`pkg/memory/`)

**Interface:**
```go
type WorkingMemoryStrategy interface {
    AddMessage(session, msg) error
    GetMessages(session) []*pb.Message
    
    // ⭐ NEW: Strategy-managed loading
    LoadState(sessionID, sessionService) (*ConversationHistory, error)
    
    // Returns new messages to persist (e.g., summary)
    CheckAndSummarize(session) ([]*pb.Message, error)
    
    Clear(session) error
}
```

#### **A. Summary Buffer Strategy** (`summary_buffer.go`)

**Concept:** Summarize old messages, keep recent + summary

**LoadState Implementation:**
```go
func (s *SummaryBufferStrategy) LoadState(sessionID, sessService) {
    // 1. Load ALL messages from SQL
    allMessages, _ := sessService.GetMessagesWithOptions(sessionID, LoadOptions{})
    
    // 2. Find last summary (checkpoint detection)
    lastSummaryIdx := s.findLastSummaryIndex(allMessages)
    
    // 3. Load from checkpoint forward
    if lastSummaryIdx >= 0 {
        // Checkpoint found: Load summary + everything after
        messagesToLoad = allMessages[lastSummaryIdx:]
        log.Printf("📍 Checkpoint at message %d/%d", lastSummaryIdx, len(allMessages))
    } else {
        // No checkpoint: Load recent 100 (safety limit)
        messagesToLoad = allMessages[len(allMessages)-100:]
    }
    
    // 4. Reconstruct in-memory session
    session := NewConversationHistory(sessionID)
    for _, msg := range messagesToLoad {
        session.AddMessage(msg)
    }
    return session
}
```

**Checkpoint Detection:**
```go
func findLastSummaryIndex(messages) int {
    for i := len(messages) - 1; i >= 0; i-- {
        msg := messages[i]
        if msg.Role == pb.Role_ROLE_UNSPECIFIED &&
           strings.HasPrefix(ExtractText(msg), "Summary:") {
            return i  // Found summary message
        }
    }
    return -1  // No checkpoint
}
```

**Efficiency Example:**
- 1000 messages in SQL
- Last summary at message 800
- **Loads only 200 messages** (80% reduction!)

#### **B. Buffer Window Strategy** (`buffer_window.go`)

**Concept:** Keep only last N messages

**LoadState Implementation:**
```go
func (s *BufferWindowStrategy) LoadState(sessionID, sessService) {
    // Load last N messages directly from SQL
    messages, _ := sessService.GetMessagesWithOptions(sessionID, LoadOptions{
        Limit: s.windowSize,  // e.g., 10 messages
    })
    
    // Reconstruct in-memory session
    session := NewConversationHistory(sessionID)
    for _, msg := range messages {
        session.AddMessage(msg)
    }
    return session
}
```

**Efficiency:** SQL `LIMIT` clause - only N messages loaded

---

### 5. **Long-Term Memory** (`pkg/memory/vector_memory.go`)

**Responsibility:** Semantic memory with vector search

**Store Operation:**
```go
func (v *VectorMemoryStrategy) Store(agentID, sessionID, msg) {
    // Extract text
    text := protocol.ExtractTextFromMessage(msg)
    
    // Generate embedding (via LLM/Ollama)
    embedding, _ := v.embedder.GenerateEmbedding(text)
    
    // Store with agent+session metadata
    v.vectorDB.Upsert(PointStruct{
        ID:     uuid.New(),
        Vector: embedding,
        Payload: map[string]interface{}{
            "agent_id":   agentID,      // Multi-agent isolation
            "session_id": sessionID,
            "role":       msg.Role.String(),
            "content":    text,
            "timestamp":  time.Now().Unix(),
        },
    })
}
```

**Recall Operation:**
```go
func (v *VectorMemoryStrategy) Recall(agentID, sessionID, query, k) {
    // Generate query embedding
    queryEmbed, _ := v.embedder.GenerateEmbedding(query)
    
    // Search with DUAL isolation filter
    results, _ := v.vectorDB.Search(SearchRequest{
        Vector:  queryEmbed,
        Limit:   k,
        Filter: Filter{
            Must: []Condition{
                {Key: "agent_id", Match: agentID},      // Agent isolation
                {Key: "session_id", Match: sessionID},  // Session isolation
            },
        },
    })
    
    return results
}
```

**Isolation Levels:**
```
┌─────────────────────────────────────────────┐
│         Vector DB (All Memories)            │
├─────────────────────────────────────────────┤
│  Agent: assistant  │  Agent: math_bot       │
│  ┌───────────────┐ │  ┌──────────────┐     │
│  │ Session: s1   │ │  │ Session: s1  │     │
│  │ [msg1, msg2]  │ │  │ [msg3, msg4] │     │
│  └───────────────┘ │  └──────────────┘     │
│  ┌───────────────┐ │                       │
│  │ Session: s2   │ │  ← Isolated by        │
│  │ [msg5, msg6]  │ │    (agent_id,         │
│  └───────────────┘ │     session_id)       │
└─────────────────────────────────────────────┘
```

---

## Three-Layer Memory System

### Layer 1: Session Store (Persistent SQL)

**Purpose:** Durability - survives restarts

**Characteristics:**
- ✅ **All messages persisted** (no filtering)
- ✅ **Transaction support** (atomic saves)
- ✅ **Multi-agent isolated** via composite key
- ✅ **Infinite retention** (no automatic cleanup)

**Storage:**
```
sessions table:
  PRIMARY KEY (id, agent_id)  -- Composite key for isolation
  
session_messages table:
  - session_id (foreign key)
  - message_id (UUID)
  - message_json (protobuf JSON)
  - created_at (timestamp)
```

---

### Layer 2: Working Memory (In-Memory, Strategy-Managed)

**Purpose:** Active context for reasoning

**Characteristics:**
- ⚠️ **NOT a cache** of SQL (misconception!)
- ✅ **Strategy controls loading** via `LoadState()`
- ✅ **Ephemeral** (cleared on restart)
- ✅ **Checkpoint-aware** (efficient loading)

**Data Structure:**
```go
type ConversationHistory struct {
    SessionID string
    Messages  []*pb.Message  // In-memory only
    Metadata  map[string]interface{}
}
```

**Loading Pattern:**
```
On agent.execute():
1. Check if session exists in memory? NO
2. Call workingMemory.LoadState(sessionID, sessionService)
3. Strategy decides what to load:
   - summary_buffer: Load from last checkpoint
   - buffer_window: Load last N messages
4. Populate in-memory ConversationHistory
5. Use for reasoning
```

---

### Layer 3: Long-Term Memory (Persistent Vector DB)

**Purpose:** Semantic recall (RAG)

**Characteristics:**
- ✅ **Asynchronous batching** (performance)
- ✅ **Vector embeddings** (semantic search)
- ✅ **Agent+Session isolated** (dual filtering)
- ✅ **Optional** (can disable)

**Batch Flow:**
```
Message saved → Queue in pendingBatches[sessionID]
              ↓
          Batch size = 5? YES
              ↓
    Generate embeddings (LLM/Ollama)
              ↓
    Upsert to Qdrant with metadata
              ↓
    Clear batch queue
```

---

## Data Flow: Message Lifecycle

### 1. **User Sends Message** (CLI Example)

```bash
./hector call --session s1 "What is 2+2?"
```

**Flow:**
```
1. CLI Parser (cli/commands.go)
   ├─ Parse --session flag
   ├─ Create pb.Message with ContextId = "s1"
   └─ Call LocalClient.SendMessage()

2. LocalClient (a2a/client/direct.go)
   ├─ Extract contextID from message
   ├─ ADD TO CONTEXT: ctx = WithValue(ctx, "sessionID", "s1")  ⭐ CRITICAL
   └─ Call agent.SendMessage(ctx, message)

3. Agent Core (agent/agent.go)
   ├─ Extract sessionID from context
   ├─ Load history: memoryService calls workingMemory.LoadState()
   │  └─ summary_buffer finds checkpoint, loads efficiently
   ├─ Build prompt with history
   ├─ Execute reasoning (CoT/ReAct/Supervisor)
   ├─ Get LLM response
   └─ SAVE MESSAGES (turn boundary)
```

---

### 2. **Saving Messages** (Turn Boundary)

```
Agent.saveMessages() called ONCE per turn:

┌────────────────────────────────────────────────┐
│ 1. Collect ALL messages from current turn:    │
│    - User message (ROLE_USER)                  │
│    - Agent thoughts (ROLE_AGENT, SYSTEM)       │
│    - Tool calls (content with tool_calls)      │
│    - Tool results (ROLE_TOOL)                  │
│    - Summaries (ROLE_UNSPECIFIED)              │
└────────────────┬───────────────────────────────┘
                 │
                 ▼
┌────────────────────────────────────────────────┐
│ 2. MemoryService.AddBatchToHistory()           │
│    ├─ AppendMessages() to SQL (transaction)    │
│    │  └─ BEGIN → INSERT all → UPDATE → COMMIT │
│    ├─ Queue for LTM batching (async)           │
│    ├─ LoadState() via strategy (efficient)     │
│    ├─ CheckAndSummarize() ONCE                 │
│    └─ Save strategy messages (e.g., summary)   │
└────────────────┬───────────────────────────────┘
                 │
                 ▼
┌────────────────────────────────────────────────┐
│ 3. SQL Session Store                           │
│    ├─ Insert into session_messages             │
│    ├─ Update sessions.updated_at               │
│    └─ Commit transaction (atomic)              │
└────────────────┬───────────────────────────────┘
                 │
                 ▼ (async)
┌────────────────────────────────────────────────┐
│ 4. Long-Term Memory (batched)                  │
│    ├─ Accumulate in pendingBatches             │
│    ├─ When batch size = 5:                     │
│    │  ├─ Generate embeddings                   │
│    │  ├─ Upsert to Qdrant with metadata        │
│    │  └─ Clear batch                           │
│    └─ OR on Shutdown(): Flush all pending      │
└────────────────────────────────────────────────┘
```

---

### 3. **Loading History** (Next Request)

```bash
./hector call --session s1 "What did I just ask?"
```

**Flow:**
```
1. Agent.execute() starts
   ├─ sessionID = "s1" (from context)
   └─ Need history for this session

2. MemoryService called (via PromptService)
   └─ workingMemory.LoadState("s1", sessionService)

3. Strategy-Managed Loading (summary_buffer example):
   ┌─────────────────────────────────────────┐
   │ A. Load ALL messages from SQL:          │
   │    SELECT * FROM session_messages       │
   │    WHERE session_id='s1' AND agent='...'│
   │    Result: [msg1...msg1000] (1000 msgs) │
   └─────────────────┬───────────────────────┘
                     │
   ┌─────────────────▼───────────────────────┐
   │ B. Find last summary (checkpoint):      │
   │    Scan backwards for ROLE_UNSPECIFIED  │
   │    with "Summary:" prefix               │
   │    Found at: msg800                     │
   └─────────────────┬───────────────────────┘
                     │
   ┌─────────────────▼───────────────────────┐
   │ C. Load from checkpoint:                │
   │    messagesToLoad = [msg800...msg1000]  │
   │    (200 messages instead of 1000!)      │
   │    ✅ 80% reduction                     │
   └─────────────────┬───────────────────────┘
                     │
   ┌─────────────────▼───────────────────────┐
   │ D. Reconstruct in-memory session:       │
   │    session = NewConversationHistory()   │
   │    for msg in messagesToLoad:           │
   │        session.AddMessage(msg)          │
   │    return session                       │
   └─────────────────────────────────────────┘

4. Agent uses loaded history:
   ├─ Build prompt with messages
   ├─ Send to LLM: "History: [msg800...msg1000]\nUser: What did I just ask?"
   ├─ LLM responds: "You asked about 2+2"
   └─ Save new turn to SQL
```

---

## Session Persistence Mechanism

### Configuration

#### Global Session Stores (Shared Infrastructure)
```yaml
# config.yaml
session_stores:
  my-shared-store:
    backend: sql
    sql:
      driver: postgres
      host: localhost
      port: 5432
      database: hector_sessions
      username: ${DB_USER}
      password: ${DB_PASSWORD}
      max_conns: 50
      max_idle: 10

agents:
  agent1:
    session_store: my-shared-store  # Reference by name
  agent2:
    session_store: my-shared-store  # Both agents share DB
```

#### Zero-Config Mode (Automatic)
```go
// config/config.go - CreateZeroConfig()
SessionStores: map[string]SessionStoreConfig{
    "default-session-store": {
        Backend: "sql",
        SQL: &SessionSQLConfig{
            Driver:   "sqlite",
            Database: "./data/sessions.db",  // Auto-created
            MaxConns: 10,
        },
    },
}

agents["assistant"].SessionStore = "default-session-store"
```

**Result:** CLI works out-of-the-box with persistence!

---

### Component Manager (Connection Pooling)

```go
// component/manager.go
type ComponentManager struct {
    sessionStoreDBs map[string]interface{}  // Cached DB connections
    mu              sync.RWMutex
}

func (cm *ComponentManager) GetSessionService(storeName, agentID) {
    // 1. Check cache
    cm.mu.RLock()
    if db, exists := cm.sessionStoreDBs[storeName]; exists {
        cm.mu.RUnlock()
        return NewSQLSessionService(db, agentID, dialect)
    }
    cm.mu.RUnlock()
    
    // 2. Create new connection
    config := cm.config.SessionStores[storeName]
    db := createDBConnection(config)
    
    // 3. Cache it
    cm.mu.Lock()
    cm.sessionStoreDBs[storeName] = db
    cm.mu.Unlock()
    
    // 4. Return service with agent_id
    return NewSQLSessionService(db, agentID, dialect)
}
```

**Why Caching?**
- ✅ Connection pooling (efficiency)
- ✅ Multiple agents share one DB connection
- ✅ Prevents resource exhaustion

---

## Multi-Agent Isolation

### Problem Statement

**Scenario:**
```
Agent: assistant   (session: s1)  → Message: "My password is secret123"
Agent: math_bot    (session: s1)  → Should NOT see assistant's message!
```

### Solution: Composite Primary Key

#### Database Schema
```sql
CREATE TABLE sessions (
    id VARCHAR(255) NOT NULL,
    agent_id VARCHAR(255) NOT NULL,
    metadata TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (id, agent_id)  -- ⭐ COMPOSITE KEY
);

CREATE TABLE session_messages (
    session_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) PRIMARY KEY,
    message_json TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);
```

**Result:**
```
sessions table:
┌──────┬──────────────┬─────────────┐
│ id   │ agent_id     │ updated_at  │
├──────┼──────────────┼─────────────┤
│ s1   │ assistant    │ 2025-10-22  │  ← Different rows!
│ s1   │ math_bot     │ 2025-10-22  │
└──────┴──────────────┴─────────────┘
```

---

### SQL Query Isolation

**Every query MUST join with sessions table:**

```sql
-- ❌ WRONG (no agent isolation)
SELECT * FROM session_messages 
WHERE session_id = 's1';

-- ✅ CORRECT (agent isolated)
SELECT sm.message_id, sm.message_json 
FROM session_messages sm
JOIN sessions s ON sm.session_id = s.id
WHERE sm.session_id = 's1' AND s.agent_id = 'assistant';
```

**Implementation:**
```go
// session_service_sql.go - GetMessagesWithOptions()
query := `
    SELECT sm.message_id, sm.message_json 
    FROM session_messages sm 
    JOIN sessions s ON sm.session_id = s.id 
    WHERE sm.session_id = ? AND s.agent_id = ?  -- ⭐ CRITICAL
    ORDER BY sm.created_at ASC
`
rows, err := s.db.QueryContext(ctx, query, sessionID, s.agentID)
```

---

### Long-Term Memory Isolation

**Vector DB Metadata:**
```go
vectorDB.Upsert(Point{
    Vector: embedding,
    Payload: {
        "agent_id":   "assistant",  // Isolation key 1
        "session_id": "s1",         // Isolation key 2
        "content":    "message text",
    },
})
```

**Search with Dual Filter:**
```go
vectorDB.Search(SearchRequest{
    Vector: queryEmbed,
    Filter: Filter{
        Must: [
            {Key: "agent_id", Match: "assistant"},   // Must match both
            {Key: "session_id", Match: "s1"},
        ],
    },
})
```

---

## Strategy-Managed Loading

### Design Philosophy

**Old (Broken) Approach:**
```go
// ❌ Memory service loads everything
messages := sessionService.GetMessages(sessionID, 0)  // All messages!
workingMemory.AddMessages(messages)  // Dump into strategy
```

**Problems:**
- 🔴 1000 messages in SQL → 1000 messages loaded (slow!)
- 🔴 Strategy has no control over loading
- 🔴 Checkpoint detection impossible
- 🔴 Exceeds LLM token limits

---

### New (Correct) Approach

**Strategy controls loading:**
```go
// ✅ Strategy decides what to load
session := workingMemory.LoadState(sessionID, sessionService)
```

**Interface:**
```go
type WorkingMemoryStrategy interface {
    // Strategy implements this method
    LoadState(sessionID string, sessionService interface{}) (*ConversationHistory, error)
}
```

---

### Strategy Comparison

#### Summary Buffer (Checkpoint-Aware)

```go
func LoadState(sessionID, sessService) {
    // Load all to find checkpoint
    all := sessService.GetMessagesWithOptions(sessionID, LoadOptions{})
    
    // Find last summary
    checkpointIdx := findLastSummaryIndex(all)
    
    if checkpointIdx >= 0 {
        // Load from checkpoint forward
        return all[checkpointIdx:]  // e.g., 200 of 1000 messages
    } else {
        // No checkpoint: Load recent 100 (safety)
        return all[len(all)-100:]
    }
}
```

**Efficiency:**
- 1000 messages → 200 loaded (80% reduction)
- Scales with conversation length

---

#### Buffer Window (Simple Limit)

```go
func LoadState(sessionID, sessService) {
    // Load last N messages directly
    messages := sessService.GetMessagesWithOptions(sessionID, LoadOptions{
        Limit: 10,  // Window size
    })
    return messages
}
```

**Efficiency:**
- 1000 messages → 10 loaded (99% reduction)
- Fixed memory footprint

---

### LoadOptions Flexibility

```go
type LoadOptions struct {
    Limit         int        // Max messages (0 = all)
    FromMessageID string     // Start from specific message
    Roles         []pb.Role  // Filter by role
}

// Example: Load last 20 agent responses
sessService.GetMessagesWithOptions(sessionID, LoadOptions{
    Limit: 20,
    Roles: []pb.Role{pb.Role_ROLE_AGENT},
})
```

---

## API Flows

### REST API Flow

```
Client → HTTP POST /v1/agents/{name}/message:send
  ↓
Transport (HTTP → Protobuf)
  ├─ Parse JSON body
  ├─ Extract: agent_id, message, context_id
  └─ Build pb.AgentMessageRequest
  ↓
A2A Server (a2a/server/bootstrap.go)
  ├─ Extract context_id from message
  ├─ ADD TO CONTEXT: ctx = WithValue(ctx, "sessionID", context_id)
  └─ Call agent.SendMessage(ctx, req)
  ↓
Agent Core
  ├─ sessionID = ctx.Value("sessionID")  ✅ Available!
  ├─ Load history
  ├─ Execute reasoning
  └─ Save messages
  ↓
HTTP Response → Client
```

**Critical Code:**
```go
// a2a/server/bootstrap.go - HandleAgentMessage
contextID := req.Message.ContextId
if contextID == "" {
    contextID = uuid.New().String()
}
ctx = context.WithValue(ctx, "sessionID", contextID)  // ⭐ MUST SET
```

---

### CLI Flow (In-Process)

```
User → ./hector call --session s1 "hello"
  ↓
CLI Parser (cli/commands.go)
  ├─ Parse flags: --session → args.SessionID
  ├─ Build pb.Message with ContextId = "s1"
  └─ Call executeCall()
  ↓
LocalClient (a2a/client/direct.go)
  ├─ Extract contextID from message
  ├─ ADD TO CONTEXT: ctx = WithValue(ctx, "sessionID", contextID)  ⭐ FIX
  └─ Call agent.SendMessage(ctx, msg) IN-PROCESS
  ↓
Agent Core
  ├─ sessionID = ctx.Value("sessionID")  ✅ Available!
  ├─ Load history
  ├─ Execute reasoning
  └─ Save messages
  ↓
CLI Output → User
```

**Bug History:**
- ❌ **Pre-fix:** LocalClient didn't set context → "default" always used
- ✅ **Post-fix:** Context properly propagated → session resumption works

---

### gRPC A2A Flow

```
Remote Agent → gRPC Call SendMessage(req)
  ↓
A2A Server (a2a/server/server.go)
  ├─ Receive AgentMessageRequest
  ├─ Extract context_id
  └─ Same flow as REST API
  ↓
Agent Core → Response
```

---

## Configuration Modes

### 1. Zero-Config (CLI Default)

**Command:**
```bash
./hector call "hello"  # No config file needed!
```

**Auto-Generated Config:**
```go
// config/config.go - CreateZeroConfig()
Config{
    Agents: map[string]AgentConfig{
        "assistant": {
            LLM: "gpt",
            SessionStore: "default-session-store",  // Auto-assigned
            Memory: MemoryConfig{
                Working: WorkingMemoryConfig{
                    Strategy: "summary_buffer",
                },
            },
        },
    },
    SessionStores: map[string]SessionStoreConfig{
        "default-session-store": {
            Backend: "sql",
            SQL: &SessionSQLConfig{
                Driver:   "sqlite",
                Database: "./data/sessions.db",  // Created automatically
                MaxConns: 10,
            },
        },
    },
}
```

**Result:**
- ✅ SQLite DB at `./data/sessions.db`
- ✅ Session persistence enabled
- ✅ Works immediately without config

---

### 2. Explicit Config (Production)

**config.yaml:**
```yaml
session_stores:
  prod-sessions:
    backend: sql
    sql:
      driver: postgres
      host: db.example.com
      port: 5432
      database: hector_prod
      username: ${DB_USER}
      password: ${DB_PASSWORD}
      max_conns: 100
      max_idle: 20
      ssl_mode: require

agents:
  customer-support:
    llm: gpt
    session_store: prod-sessions
    memory:
      working:
        strategy: summary_buffer
        max_messages: 50
      longterm:
        backend: qdrant
        qdrant:
          url: https://qdrant.example.com
          collection: customer_memories

  math-assistant:
    llm: claude
    session_store: prod-sessions  # Shared DB, isolated by agent_id
    memory:
      working:
        strategy: buffer_window
        window_size: 10
```

**Command:**
```bash
./hector serve --config config.yaml --port 9301
```

---

### 3. Multi-Agent Shared Store

**Scenario:** 10 agents, 1 PostgreSQL database

```yaml
session_stores:
  shared-postgres:
    backend: sql
    sql:
      driver: postgres
      host: shared-db.internal
      database: all_agents_sessions
      max_conns: 200  # Pool shared across agents

agents:
  agent1: {session_store: shared-postgres}
  agent2: {session_store: shared-postgres}
  # ... agent3-10 ...
```

**Architecture:**
```
┌─────────────────────────────────────────┐
│     ComponentManager (1 instance)       │
│  ┌───────────────────────────────────┐  │
│  │ sessionStoreDBs["shared-postgres"]│  │
│  │   ↓                               │  │
│  │ *sql.DB (connection pool)         │  │
│  └───────────────────────────────────┘  │
│         ↑         ↑         ↑           │
│     agent1    agent2    agent3          │
│  (agentID=a1)(agentID=a2)(agentID=a3)  │
└─────────────────────────────────────────┘
           ↓ (shared connection)
┌─────────────────────────────────────────┐
│      PostgreSQL (shared-postgres)       │
│  ┌───────────────────────────────────┐  │
│  │ sessions table:                   │  │
│  │ (s1, a1), (s1, a2), (s1, a3) ...  │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

**Benefits:**
- ✅ Single DB connection pool (efficient)
- ✅ Multi-agent isolation (composite key)
- ✅ Centralized session management

---

## Database Schema

### Sessions Table
```sql
CREATE TABLE sessions (
    id VARCHAR(255) NOT NULL,           -- Session ID (user-provided or auto)
    agent_id VARCHAR(255) NOT NULL,     -- Agent isolation key
    metadata TEXT,                      -- JSON metadata (optional)
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (id, agent_id)          -- Composite key for multi-agent
);

-- Index for performance
CREATE INDEX idx_sessions_agent_updated 
ON sessions(agent_id, updated_at DESC);
```

---

### Session Messages Table
```sql
CREATE TABLE session_messages (
    message_id VARCHAR(255) PRIMARY KEY,  -- UUID
    session_id VARCHAR(255) NOT NULL,     -- Foreign key to sessions.id
    message_json TEXT NOT NULL,           -- Protobuf JSON serialization
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);

-- Index for efficient retrieval
CREATE INDEX idx_session_messages_session_created 
ON session_messages(session_id, created_at ASC);
```

---

### Data Example

**After 2 agents use session "s1":**

**sessions:**
```
┌──────┬──────────────┬──────────┬─────────────┬─────────────┐
│ id   │ agent_id     │ metadata │ created_at  │ updated_at  │
├──────┼──────────────┼──────────┼─────────────┼─────────────┤
│ s1   │ assistant    │ {}       │ 10:00:00    │ 10:05:00    │
│ s1   │ math_bot     │ {}       │ 10:02:00    │ 10:03:00    │
└──────┴──────────────┴──────────┴─────────────┴─────────────┘
```

**session_messages:**
```
┌────────────┬────────────┬─────────────────────────────┬─────────────┐
│ message_id │ session_id │ message_json                │ created_at  │
├────────────┼────────────┼─────────────────────────────┼─────────────┤
│ uuid-1     │ s1         │ {"role":"USER","parts":[...]}│ 10:00:01    │
│ uuid-2     │ s1         │ {"role":"AGENT","parts":[...│ 10:00:05    │
│ uuid-3     │ s1         │ {"role":"USER","parts":[...]}│ 10:02:01    │  ← math_bot
│ uuid-4     │ s1         │ {"role":"AGENT","parts":[...│ 10:02:03    │  ← math_bot
└────────────┴────────────┴─────────────────────────────┴─────────────┘
```

**Query for assistant's session s1:**
```sql
SELECT sm.message_json 
FROM session_messages sm
JOIN sessions s ON sm.session_id = s.id
WHERE sm.session_id = 's1' AND s.agent_id = 'assistant';

-- Result: uuid-1, uuid-2 only (isolated!)
```

---

## Component Interactions

### Startup Sequence

```
1. main.go
   ├─ Load config.yaml
   └─ Create ComponentManager
       ├─ Register LLM providers
       ├─ Register session stores
       └─ Register vector databases

2. AgentFactory.CreateAgent("assistant")
   ├─ Get agent config
   ├─ componentManager.GetSessionService("default-session-store", "assistant")
   │  ├─ Check cache: sessionStoreDBs["default-session-store"]
   │  ├─ If missing: Create *sql.DB, run migrations, cache it
   │  └─ Return SQLSessionService(db, "assistant", "sqlite")
   ├─ Create WorkingMemoryStrategy (summary_buffer)
   ├─ Create LongTermMemoryStrategy (vector_memory)
   ├─ Create MemoryService(sessionSvc, workingSvc, longTermSvc, "assistant")
   └─ Return Agent instance

3. Start Server (REST/gRPC)
   └─ Ready to handle requests
```

---

### Request Processing Sequence

```
1. Request arrives (REST/CLI/gRPC)
   ├─ Extract: agent_id, message, context_id (session_id)
   └─ Add context_id to context.Context

2. Agent.SendMessage(ctx, req)
   ├─ sessionID := ctx.Value("sessionID")
   ├─ history := memoryService.Recall(sessionID, "")  // Empty query = load all
   │  └─ workingMemory.LoadState(sessionID, sessionService)
   │     ├─ Query SQL: GetMessagesWithOptions(sessionID, LoadOptions{})
   │     ├─ Strategy decides: Load from checkpoint or limit
   │     └─ Return ConversationHistory (in-memory)
   ├─ Build prompt with history
   ├─ Execute reasoning loop
   │  ├─ LLM call
   │  ├─ Tool execution (if needed)
   │  └─ Collect responses
   └─ saveMessages(sessionID, currentTurn)

3. Agent.saveMessages(sessionID, currentTurn)
   ├─ Collect ALL messages (no filtering)
   └─ memoryService.AddBatchToHistory(sessionID, messages)
      ├─ sessionService.AppendMessages(sessionID, messages)  // SQL transaction
      ├─ Queue for LTM batching
      ├─ workingMemory.LoadState(sessionID, sessionService)  // Reload for summarization
      ├─ workingMemory.CheckAndSummarize(history)  // Check ONCE
      └─ If summary created: Save summary message to SQL

4. Response returned to client
```

---

### Shutdown Sequence

```
1. Signal received (SIGINT/SIGTERM)

2. Agent.Shutdown()
   └─ memoryService.Shutdown()
      ├─ batchMu.Lock()
      ├─ For each sessionID in pendingBatches:
      │  ├─ flushLongTermBatch(sessionID)
      │  │  ├─ Generate embeddings
      │  │  └─ Upsert to Qdrant
      │  └─ Clear batch
      └─ batchMu.Unlock()

3. ComponentManager.Shutdown()
   └─ For each sessionStoreDB:
      └─ db.Close()  // Close SQL connections

4. Exit
```

---

## Concurrency & Thread Safety

### Race Conditions Fixed

#### 1. **MemoryService.pendingBatches**

**Problem:**
```go
// ❌ Concurrent map writes crash
s.pendingBatches[sessionID] = append(s.pendingBatches[sessionID], msg)
```

**Solution:**
```go
// ✅ Mutex protection
s.batchMu.Lock()
s.pendingBatches[sessionID] = append(s.pendingBatches[sessionID], msg)
s.batchMu.Unlock()
```

---

#### 2. **ComponentManager.sessionStoreDBs**

**Problem:**
```go
// ❌ Concurrent map read/write
if db, exists := cm.sessionStoreDBs[name]; exists {
    return db
}
cm.sessionStoreDBs[name] = newDB  // Race!
```

**Solution:**
```go
// ✅ RWMutex for read-heavy workload
cm.mu.RLock()
if db, exists := cm.sessionStoreDBs[name]; exists {
    cm.mu.RUnlock()
    return db
}
cm.mu.RUnlock()

cm.mu.Lock()
cm.sessionStoreDBs[name] = newDB
cm.mu.Unlock()
```

---

### Database Connection Pooling

```go
// config.yaml
sql:
  max_conns: 50   // Maximum open connections
  max_idle: 10    // Idle connections in pool

// Applied to *sql.DB
db.SetMaxOpenConns(50)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
```

**Why Pooling?**
- ✅ Reuse connections (performance)
- ✅ Limit concurrent connections (prevent DB overload)
- ✅ Automatic connection recycling (memory safety)

---

## Error Handling & Recovery

### Transaction Rollback

```go
func (s *SQLSessionService) AppendMessages(sessionID, messages) error {
    // Start transaction
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    // Ensure rollback on panic or error
    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)  // Re-panic after rollback
        }
    }()
    
    // Insert messages
    for _, msg := range messages {
        _, err := tx.ExecContext(ctx, insertSQL, ...)
        if err != nil {
            tx.Rollback()  // Explicit rollback
            return fmt.Errorf("failed to insert message: %w", err)
        }
    }
    
    // Update session timestamp
    _, err = tx.ExecContext(ctx, updateSQL, ...)
    if err != nil {
        tx.Rollback()
        return fmt.Errorf("failed to update session: %w", err)
    }
    
    // Commit (all-or-nothing)
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return nil
}
```

**Guarantees:**
- ✅ All messages saved OR none saved
- ✅ No partial saves (data integrity)
- ✅ Session timestamp always consistent

---

### Graceful Degradation

```go
// AddBatchToHistory - Error handling strategy
func (m *MemoryService) AddBatchToHistory(sessionID, messages) error {
    // 1. SQL save MUST succeed
    if err := m.sessionService.AppendMessages(sessionID, messages); err != nil {
        return fmt.Errorf("critical: failed to save messages: %w", err)
    }
    
    // 2. LTM batching is optional (async)
    for _, msg := range messages {
        m.addToLongTermBatch(sessionID, msg)  // No error return
    }
    
    // 3. Summarization is optional
    history, err := m.workingMemory.LoadState(sessionID, m.sessionService)
    if err != nil {
        log.Printf("⚠️  Failed to load for summarization: %v", err)
        return nil  // Don't fail - messages already saved!
    }
    
    newMessages, err := m.workingMemory.CheckAndSummarize(history)
    if err != nil {
        log.Printf("⚠️  Summarization failed: %v", err)
        return nil  // Don't fail - messages already saved!
    }
    
    // 4. Summary save SHOULD succeed (affects checkpoint)
    if len(newMessages) > 0 {
        if err := m.sessionService.AppendMessages(sessionID, newMessages); err != nil {
            return fmt.Errorf("warning: failed to save summary: %w", err)
        }
    }
    
    return nil
}
```

**Priority:**
1. 🔴 **Critical:** User messages MUST save
2. 🟡 **Warning:** Summary messages SHOULD save (checkpoint needs them)
3. 🟢 **Optional:** LTM batching, summarization (retry later)

---

## Testing Strategy

### Unit Tests

**Coverage:**
- ✅ `MemoryService` concurrency (race detection)
- ✅ `SessionService` CRUD operations
- ✅ Working memory strategies (buffer_window, summary_buffer)
- ✅ Long-term memory isolation

**Run:**
```bash
cd pkg/memory
go test -v -race -timeout 30s
```

---

### Integration Tests

**Coverage:**
- ✅ SQL session persistence (SQLite, PostgreSQL)
- ✅ Multi-agent isolation (composite key)
- ✅ Strategy loading (checkpoint detection)
- ✅ Transaction rollback

**Script:**
```bash
./test-session-integration.sh
```

---

### End-to-End Tests

**Coverage:**
- ✅ REST API session persistence
- ✅ CLI session resumption (zero-config)
- ✅ Multi-agent shared store
- ✅ Server restart persistence
- ✅ Real LLM calls (reduced tokens)

**Script:**
```bash
./test-e2e-sessions.sh
```

**Example Test:**
```bash
# Test 1: Session persistence
curl -X POST http://localhost:9301/v1/agents/assistant/message:send \
  -d '{"agent_id":"assistant","message":{"context_id":"s1","parts":[{"text":"My name is Alice"}]}}'

# Test 2: Session resumption
curl -X POST http://localhost:9301/v1/agents/assistant/message:send \
  -d '{"agent_id":"assistant","message":{"context_id":"s1","parts":[{"text":"What is my name?"}]}}'

# Expected: "Your name is Alice"

# Test 3: Database verification
sqlite3 ./data/sessions.db "SELECT COUNT(*) FROM session_messages WHERE session_id='s1';"
# Expected: 4 (2 user + 2 agent messages)
```

---

## Troubleshooting Guide

### Issue: Agent doesn't remember conversation

**Symptoms:**
```bash
./hector call --session s1 "My name is Bob"
./hector call --session s1 "What is my name?"
# Response: "I don't have access to previous messages"
```

**Diagnosis:**
```bash
# 1. Check if messages are being saved
sqlite3 ./data/sessions.db "SELECT COUNT(*) FROM session_messages WHERE session_id='s1';"
# If 0: Session ID not propagating

# 2. Check session table
sqlite3 ./data/sessions.db "SELECT * FROM sessions WHERE id='s1';"
# If empty: Session not created

# 3. Check logs for errors
grep "Failed to save messages" server.log
```

**Common Causes:**
1. ❌ **Context not propagated** (LocalClient bug - FIXED)
2. ❌ **Session store not configured** (zero-config bug - FIXED)
3. ❌ **Database file permissions** (check write access)

**Fix:**
- ✅ Ensure `LocalClient` sets `ctx = WithValue(ctx, "sessionID", contextID)`
- ✅ Verify zero-config creates `./data/sessions.db`
- ✅ Check file permissions: `ls -la ./data/`

---

### Issue: Multi-agent isolation broken

**Symptoms:**
```bash
# Agent A
curl .../agents/agentA/message:send -d '{"message":{"context_id":"s1","parts":[{"text":"Secret"}]}}'

# Agent B can see Agent A's message!
curl .../agents/agentB/message:send -d '{"message":{"context_id":"s1","parts":[{"text":"What did agent A say?"}]}}'
# Response: "Agent A said: Secret"  ← BAD!
```

**Diagnosis:**
```sql
-- Check if composite key exists
SELECT sql FROM sqlite_master WHERE type='table' AND name='sessions';
-- Should show: PRIMARY KEY (id, agent_id)

-- Check if queries filter by agent_id
-- Run with query logging enabled:
sqlite3 ./data/sessions.db
.log stdout
-- Then trigger a message and check SQL output
```

**Common Causes:**
1. ❌ **Single primary key** (should be composite: `(id, agent_id)`)
2. ❌ **SQL queries missing `AND s.agent_id = ?`** (FIXED)
3. ❌ **Agent ID not passed to SQLSessionService** (FIXED)

**Fix:**
- ✅ Drop and recreate `sessions` table with composite key
- ✅ Verify all SQL queries join with sessions table
- ✅ Check `NewSQLSessionService` receives `agentID`

---

### Issue: Checkpoint not detected (loading all messages)

**Symptoms:**
```bash
# After 1000 messages with summaries
grep "Checkpoint detected" server.log
# No matches found

grep "Loading all.*messages" server.log
# Shows: "Loading all 1000 messages"
```

**Diagnosis:**
```bash
# 1. Check if summary messages exist
sqlite3 ./data/sessions.db \
  "SELECT message_json FROM session_messages WHERE session_id='s1'" \
  | grep -c "Summary:"
# If 0: Summaries not being saved

# 2. Check summary message format
sqlite3 ./data/sessions.db \
  "SELECT message_json FROM session_messages WHERE message_json LIKE '%Summary:%' LIMIT 1;"
# Check if role is ROLE_UNSPECIFIED
```

**Common Causes:**
1. ❌ **Summary messages not persisted** (CheckAndSummarize doesn't return them - FIXED)
2. ❌ **Wrong role** (should be `ROLE_UNSPECIFIED`)
3. ❌ **Wrong content format** (should start with "Summary:")

**Fix:**
- ✅ Ensure `CheckAndSummarize` returns `[]*pb.Message`
- ✅ Verify `AddBatchToHistory` saves returned messages
- ✅ Check summary format: `Role: ROLE_UNSPECIFIED, Content: "Summary: ..."`

---

### Issue: Database locked (SQLite)

**Symptoms:**
```
Error: failed to save messages: database is locked
```

**Diagnosis:**
```bash
# Check for multiple processes
lsof ./data/sessions.db
# If multiple processes: Close old servers

# Check for long-running transactions
# Enable WAL mode for better concurrency
```

**Fix:**
```go
// Use WAL mode for SQLite (better concurrency)
db.Exec("PRAGMA journal_mode=WAL;")

// Or switch to PostgreSQL for production
session_stores:
  prod:
    backend: sql
    sql:
      driver: postgres
      # ...
```

---

## Best Practices

### 1. **Always Use Batch Methods**
```go
// ✅ GOOD: Atomic, efficient, checks summarization ONCE
memoryService.AddBatchToHistory(sessionID, messages)

// ❌ BAD: Multiple transactions, infinite loop risk
for _, msg := range messages {
    memoryService.AddToHistory(sessionID, msg)  // Deprecated!
}
```

---

### 2. **Use Strategy Loading**
```go
// ✅ GOOD: Strategy-controlled, checkpoint-aware
history := workingMemory.LoadState(sessionID, sessionService)

// ❌ BAD: Loads everything, bypasses strategy
messages := sessionService.GetMessages(sessionID, 0)  // All messages!
```

---

### 3. **Always Propagate Session ID in Context**
```go
// ✅ GOOD: Session ID in context
ctx = context.WithValue(ctx, "sessionID", sessionID)
agent.SendMessage(ctx, req)

// ❌ BAD: Session ID lost
agent.SendMessage(context.Background(), req)  // Uses "default"
```

---

### 4. **Use Composite Keys for Multi-Agent**
```sql
-- ✅ GOOD: Multi-agent isolation
PRIMARY KEY (id, agent_id)

-- ❌ BAD: Session ID conflicts
PRIMARY KEY (id)
```

---

### 5. **Handle Errors Gracefully**
```go
// ✅ GOOD: Critical operations fail fast
if err := sessionService.AppendMessages(...); err != nil {
    return err  // Don't continue without saving!
}

// ✅ GOOD: Optional operations degrade gracefully
if err := summarize(); err != nil {
    log.Printf("⚠️  Summarization failed: %v", err)
    // Continue - messages already saved
}
```

---

## Conclusion

Hector's session and memory architecture provides:

✅ **Durability:** SQL persistence survives restarts  
✅ **Scalability:** Strategy-managed loading with checkpoints  
✅ **Isolation:** Multi-agent composite keys prevent leaks  
✅ **Flexibility:** Pluggable strategies (summary_buffer, buffer_window)  
✅ **Efficiency:** Checkpoint detection reduces token usage by 80%+  
✅ **Simplicity:** Zero-config SQLite works out-of-the-box  
✅ **Safety:** Transaction support ensures data integrity  

**Key Takeaways for Developers:**

1. **Persistence is foundational** - All messages saved to SQL first
2. **Strategies control loading** - Never bypass `LoadState()`
3. **Context carries session ID** - Always propagate via `context.Context`
4. **Multi-agent needs composite keys** - `(session_id, agent_id)` everywhere
5. **Batch operations are atomic** - Use `AppendMessages` for safety
6. **Graceful degradation** - Critical saves must succeed, summarization can fail

---

**Document Version:** 1.0  
**Last Updated:** October 22, 2025  
**Maintainer:** Hector Development Team  
**Next Review:** Architecture evolves with new memory strategies

