---
layout: default
title: Memory Management
nav_order: 4
parent: Core Concepts
description: "Dual-layer memory system for AI agents - working memory and long-term memory"
---

# Memory Management - Never Lose Context 🧠

> **Dual-layer intelligent memory: Working memory for sessions + Long-term memory for persistent knowledge.**

---

## Overview

Hector implements a **cognitive memory architecture** inspired by human memory systems:

```
┌─────────────────────────────────────────────┐
│           HECTOR MEMORY SYSTEM              │
├─────────────────────────────────────────────┤
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │  WORKING MEMORY (Session-Scoped)    │   │
│  │                                     │   │
│  │  • Current conversation context     │   │
│  │  • Token-based management           │   │
│  │  • Automatic summarization          │   │
│  │  • Strategy: summary_buffer/buffer  │   │
│  └─────────────────────────────────────┘   │
│                    ↕                        │
│  ┌─────────────────────────────────────┐   │
│  │  LONG-TERM MEMORY (Persistent)      │   │
│  │                                     │   │
│  │  • Vector-based storage (Qdrant)    │   │
│  │  • Semantic search & recall         │   │
│  │  • Session-scoped persistence       │   │
│  │  • Auto-recall or on-demand         │   │
│  └─────────────────────────────────────┘   │
│                                             │
└─────────────────────────────────────────────┘
```

**Two Memory Types:**

1. **Working Memory** - Manages conversation history within sessions (like human short-term memory)
2. **Long-Term Memory** - Stores and recalls relevant context semantically (like human long-term memory)

Both work together seamlessly to provide optimal context management.

---

## Memory Type 1: Working Memory

**Purpose:** Manage conversation history within the current session

**Lifespan:** Session-scoped (cleared when session ends)

**Implementation:** Token-aware buffer with pluggable strategies

### Why Working Memory Matters

Traditional AI agents lose context in long conversations:
- ❌ Exceed token limits without warning
- ❌ Truncate important messages
- ❌ Use inaccurate character-based estimates
- ❌ No automatic summarization

**Result:** Broken conversations, lost context, frustrated users.

### The Solution

One simple setting that changes everything:

```yaml
memory:
  budget: 2000
```

**That's it.** Your agent now has:
- ✅ **Accurate token counting** - 100% accurate (not estimates)
- ✅ **Recency-based selection** - Most recent messages that fit within budget
- ✅ **Automatic management** - No manual intervention
- ✅ **Optional summarization** - LLM condenses old messages for unlimited conversation length

### Understanding Context Windows

The **context window** is your LLM's maximum input size - the total tokens it can process in one request:

| Model | Context Window |
|-------|----------------|
| GPT-4o | 128K tokens |
| Claude 3.5 Sonnet | 200K tokens |
| Gemini 2.0 Flash | 1M tokens |

Your LLM's context window contains multiple parts:

```
┌─────────────────────────────────────────────┐
│        LLM Context Window (128K tokens)      │
│                                              │
│  System Prompt:         500 tokens    (0.4%) │
│  Tool Definitions:    1,000 tokens    (0.8%) │
│  RAG Context:         2,000 tokens    (1.6%) │
│  Working Memory:      2,000 tokens    (1.6%) ← memory.budget
│  User Input:          1,500 tokens    (1.2%) │
│  Response Buffer:   121,000 tokens   (94.5%) │
└─────────────────────────────────────────────┘
```

**Memory budget** controls how much of your context window is reserved for conversation history.

### Working Memory Strategies

Hector supports **two pluggable working memory strategies**. Choose based on your needs.

#### Strategy 1: Summary Buffer (Default - Recommended)

**Token-based with threshold-triggered summarization.** Best for production and long conversations.

**Configuration:**
```yaml
memory:
  strategy: "summary_buffer"  # This is the DEFAULT
  budget: 2000      # Optional, defaults to 2000
  threshold: 0.8    # Optional, defaults to 0.8 (80%)
  target: 0.6       # Optional, defaults to 0.6 (60%)
```

**How it works:**
1. Accumulates messages until 80% of budget (1600 tokens)
2. Notifies user with status message (appears on new line)
3. Summarizes oldest messages via LLM (blocking, 2-5 seconds)
4. Keeps minimum 3 recent messages for context
5. Compresses to 60% of budget (1200 tokens)
6. Leaves 800 tokens breathing room
7. Repeats when threshold hit again

**User Experience:**
```
> What did I just ask?

💭 Summarizing conversation history...
You just asked about transformers in machine learning...
```
- Status notification appears on its own line
- Brief 2-5 second delay during summarization
- Response continues immediately after
- Recent context is always preserved

**Benefits:**
- Optimal token efficiency
- Hierarchical compression (summary of summaries)
- Unbounded conversation length
- Preserves context intelligently

**Best for:**
- Production applications (90% of users)
- Long conversations (50+ messages)
- When LLM summarization is acceptable
- Optimal memory efficiency

**Example:**
```yaml
agents:
  production-bot:
    llm: gpt4o
    memory:
      strategy: "summary_buffer"
      # Uses all defaults (budget: 2000, threshold: 0.8, target: 0.6)
```

#### Strategy 2: Buffer Window

**Simple LIFO, keeps last N messages.** Best for testing or simple bots.

**Configuration:**
```yaml
memory:
  strategy: "buffer_window"
  window_size: 20   # Optional, defaults to 20
```

**How it works:**
1. Keeps last 20 messages (LIFO)
2. Drops oldest message when new one arrives
3. No LLM calls, no summarization
4. Simple and predictable

**Benefits:**
- No LLM overhead
- Predictable behavior
- Fast and simple
- No blocking

**Best for:**
- Simple chatbots
- Testing/development
- Short conversations (< 20 messages)
- When summarization not needed

**Example:**
```yaml
agents:
  test-bot:
    llm: gpt4o
    memory:
      strategy: "buffer_window"
      window_size: 15  # Keep last 15 messages
```

#### Strategy Comparison

| Feature | Summary Buffer (Default) | Buffer Window |
|---------|-------------------------|---------------|
| **Token Efficiency** | Optimal | Fixed count |
| **Max Conversation** | Unlimited | ~20 messages |
| **LLM Overhead** | Yes (summarization) | No |
| **Blocking** | Yes (2-5s on trigger) | No |
| **Complexity** | Medium | Low |
| **Best For** | Production (90%) | Testing (10%) |

#### Which Strategy Should I Use?

**Use Summary Buffer if:**
- You want production-quality memory (recommended!)
- Conversations may exceed 20 messages
- Token efficiency matters
- Blocking 2-5 seconds for summarization is acceptable

**Use Buffer Window if:**
- You're testing/developing
- Conversations are always short (< 20 messages)
- You don't want LLM summarization overhead
- You need simple, predictable behavior

**Default:** If you don't specify a strategy, Hector uses `summary_buffer` with sensible defaults.

### Working Memory Configuration

#### Tier 1: Most Users (90%)

```yaml
memory:
  budget: 2000
  include_history: true
```

**Use when:**
- Normal conversations (5-50 messages)
- General assistance
- Customer support
- Quick interactions

**You get:**
- 2000 token budget (~50 messages)
- Accurate counting
- Recency-based selection

#### Tier 2: Extended Conversations (9%)

```yaml
memory:
  budget: 3000
  include_history: true
```

**Use when:**
- Longer conversations (50-100 messages)
- Code reviews
- Detailed analysis
- Complex discussions

**You get:**
- 3000 token budget (~75 messages)
- More context retained
- Same accuracy and recency-based selection

#### Tier 3: Very Long Sessions (1%)

```yaml
memory:
  strategy: "summary_buffer"
  budget: 3000
  threshold: 0.8
  target: 0.6
  include_history: true
```

**Use when:**
- 100+ message conversations
- Multi-day projects
- Extended sessions
- Ongoing collaboration

**You get:**
- 3000 token budget with summarization
- Unlimited conversation length
- Context preserved through summaries
- Recent messages intact

---

## Memory Type 2: Long-Term Memory

**Purpose:** Store and recall relevant information semantically across the session

**Lifespan:** Persistent within session (survives working memory summarization)

**Implementation:** Vector database (Qdrant) with semantic search

### Why Long-Term Memory Matters

Even with working memory summarization, important details can be lost:
- ❌ Summaries are lossy - specific facts may disappear
- ❌ Old but relevant context gets compressed away
- ❌ No semantic search - agent can't "remember" specific details from earlier

**Long-term memory solves this** by:
- ✅ Storing all messages persistently in a vector database
- ✅ Enabling semantic recall - find relevant past context
- ✅ Working alongside working memory automatically
- ✅ Surviving working memory summarization

### How It Works

```
User sends message
       ↓
┌──────────────────┐
│ Working Memory   │ ← Manages current conversation (token-aware)
└──────────────────┘
       ↓
┌──────────────────┐
│ Long-Term Memory │ ← Stores messages in vector DB (Qdrant)
└──────────────────┘
       ↓
When agent needs context:
┌──────────────────┐
│ Semantic Recall  │ ← Search vectors for relevant past messages
└──────────────────┘
       ↓
Relevant memories + Working memory → LLM
```

**Key Insight:** Working memory handles *recent* context efficiently, while long-term memory provides *semantic* recall of *any* relevant past context.

### Architecture

```
MemoryService (orchestrator)
├─ Working Memory Strategy ─→ Token-aware session management
└─ Long-Term Memory Strategy ─→ Vector-based persistent storage
   ├─ Store: Embed messages and upsert to Qdrant
   ├─ Recall: Semantic search for relevant context
   └─ Filter: Session-scoped isolation
```

**Design Benefits:**
- ✅ Decoupled strategies - working and long-term are independent
- ✅ MemoryService orchestrates both seamlessly
- ✅ Session isolation - memories don't leak between sessions
- ✅ Configurable storage and recall behavior

### Configuration

```yaml
memory:
  # Working memory (as before)
  strategy: "summary_buffer"
  budget: 2000
  
  # Long-term memory (NEW)
  long_term:
    storage_scope: "all"              # What to store: "all", "conversational", "summaries_only"
    batch_size: 1                     # Store immediately (default), or batch for performance
    auto_recall: true                 # Automatically inject relevant memories before LLM calls
    recall_limit: 5                   # Max memories to recall (default: 5)
    collection: "hector_session_memory"  # Qdrant collection name

# Required: Vector database and embedder
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334  # gRPC port
    use_tls: false

embedders:
  ollama:
    type: "ollama"
    model: "mxbai-embed-large"
    host: "http://localhost:11434"
    dimension: 1024
```

### Storage Scope Options

**`storage_scope`** controls what messages are stored in long-term memory:

| Scope | What Gets Stored | Use Case |
|-------|------------------|----------|
| `all` (default) | All messages (user, assistant, system, tool) | Maximum recall, comprehensive memory |
| `conversational` | Only user and assistant messages | Focus on dialogue, ignore tool internals |
| `summaries_only` | Only summary messages from working memory | Compressed semantic memory, lower storage |

**Example:**
```yaml
long_term:
  storage_scope: "conversational"  # Only store user/assistant dialogue
```

### Auto-Recall vs On-Demand Search

**Two ways to use long-term memory:**

#### 1. Auto-Recall (Recommended)

```yaml
long_term:
  auto_recall: true
  recall_limit: 5
```

**How it works:**
- Before each LLM call, MemoryService automatically searches long-term memory
- Uses the last user message as the search query
- Retrieves top N relevant past messages semantically
- Prepends them to working memory messages
- LLM sees: `[recalled memories] + [working memory] + [current input]`

**Benefits:**
- Zero agent effort - happens automatically
- Agent always has relevant context
- Transparent to agent - just appears as available context

**Best for:** Most agents (recommended!)

#### 2. On-Demand Search Tool

```yaml
long_term:
  auto_recall: false  # Disable auto-recall
```

The agent still has access to the `search` tool, which can search long-term memory on demand:

```yaml
# Agent can explicitly search long-term memory
Tool: search(query="user's favorite color", type="memory")
```

**Benefits:**
- Agent controls when to recall
- Can use custom queries
- More explicit reasoning

**Best for:** Agents that need fine-grained control over memory recall

### Batching for Performance

```yaml
long_term:
  batch_size: 10  # Store every 10 messages
```

**How it works:**
- Messages accumulate in a pending batch
- When `batch_size` is reached, batch is stored to Qdrant
- Flush also happens on `ClearHistory()` to ensure nothing is lost

**Trade-offs:**
- `batch_size: 1` (default) - Immediate storage, slightly more overhead
- `batch_size: 10+` - Batched storage, better performance, slight recall lag

**Recommendation:** Use default `batch_size: 1` unless you have high-throughput agents.

### Session Isolation

**Important:** Long-term memories are **session-scoped**, not global:

```yaml
# Session 1
User: "My name is Alice"
[Stored in long-term memory with session_id: "session-1"]

# Session 2
User: "What is my name?"
Agent: "I don't know your name"  # Different session, no access to session-1 memories
```

**Benefits:**
- Privacy - sessions don't leak information
- Clean separation - each conversation is independent
- Scalability - no cross-session contamination

**Note:** Cross-session memory (e.g., user profiles) is a future feature.

### Complete Example

```yaml
agents:
  research-agent:
    name: "Research Assistant"
    llm: gpt4o
    
    memory:
      # Working memory for current conversation
      strategy: "summary_buffer"
      budget: 2000
      threshold: 0.8
      target: 0.6
      
      # Long-term memory for semantic recall
      long_term:
        storage_scope: "all"
        batch_size: 1
        auto_recall: true
        recall_limit: 5
        collection: "research_agent_memory"
    
    document_stores:
      - "research_docs"

# Required infrastructure
llms:
  gpt4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"

databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334
    use_tls: false

embedders:
  ollama:
    type: "ollama"
    model: "mxbai-embed-large"
    host: "http://localhost:11434"
    dimension: 1024
```

### User Experience

```
User: My favorite color is blue and I love hiking.
Agent: Got it! I'll remember that.
[Stored in working memory + long-term memory]

... many messages later ...

User: What outdoor activities do I enjoy?
Agent: [Auto-recalls "I love hiking" from long-term memory]
       Based on what you told me earlier, you enjoy hiking!
```

**Seamless:** The agent automatically recalls relevant context without explicit search.

---

## How Working and Long-Term Memory Work Together

### The Orchestration

```
1. User sends message
   ↓
2. MemoryService.AddToHistory()
   ├→ Store to long-term memory (if enabled)
   └→ Add to working memory strategy
   
3. User requests response
   ↓
4. MemoryService.GetRecentHistory()
   ├→ Auto-recall from long-term (if enabled)
   ├→ Get working memory messages
   └→ Return: [recalled] + [working memory]
   
5. LLM receives full context
   ↓
6. Response streamed to user
```

**Key Points:**
- Long-term memory stores *before* working memory processes (no loss)
- Auto-recall happens *before* LLM call (transparent)
- Working memory summarization doesn't affect long-term storage
- Both memories are session-scoped independently

### Example Flow

```yaml
# Configuration
memory:
  budget: 2000
  long_term:
    auto_recall: true
    recall_limit: 3
```

**Conversation:**

```
[Message 1]
User: "My project deadline is March 15th"
→ Working memory: [msg1]
→ Long-term memory: [msg1 embedded + stored]

[Messages 2-50]
... conversation continues ...
→ Working memory: summarizes to [summary] + [msg48, msg49, msg50]
→ Long-term memory: [msg1...msg50 all stored]

[Message 51]
User: "When is my deadline again?"
→ Auto-recall searches long-term: finds msg1 "deadline is March 15th"
→ Working memory: [summary] + [msg48, msg49, msg50]
→ LLM receives: [msg1] + [summary] + [msg48, msg49, msg50] + [msg51]
Agent: "Your deadline is March 15th"
```

**Result:** Even though working memory summarized messages 1-47, long-term memory semantically recalled the relevant deadline message!

---

## Architecture

Hector's memory system uses a **clean, layered architecture**:

```
MemoryService (pkg/memory/)
├─ Manages sessions (lifecycle, isolation)
├─ Orchestrates working + long-term strategies
└─ Delegates to:
   ├─ WorkingMemoryStrategy
   │  ├─ SummaryBufferStrategy (token-based with summarization)
   │  └─ BufferWindowStrategy (simple LIFO)
   └─ LongTermMemoryStrategy
      └─ VectorMemoryStrategy (Qdrant + embeddings)
```

**Benefits:**
- ✅ Clean separation: Service manages infrastructure, strategies implement algorithms
- ✅ Decoupled strategies: Working and long-term are independent
- ✅ No duplication: Session management in one place
- ✅ Testable: Each layer tested independently
- ✅ Extensible: Easy to add new strategies

**File Structure:**
```
pkg/memory/
├── memory.go              → MemoryService (orchestrator)
├── working_strategy.go    → WorkingMemoryStrategy interface
├── summary_buffer.go      → Token-based strategy with summarization
├── buffer_window.go       → Simple LIFO strategy
├── longterm_strategy.go   → LongTermMemoryStrategy interface
├── vector_memory.go       → Vector-based long-term memory
├── types.go               → Configuration types
└── factory.go             → Strategy factory
```

---

## Token Counting

Uses `tiktoken-go` for exact token counting:
- **GPT-4o** - o200k_base encoding
- **GPT-4** - cl100k_base encoding
- **GPT-3.5** - cl100k_base encoding
- **Claude** - cl100k_base approximation
- **Gemini** - cl100k_base approximation

**Accuracy:** 100% for OpenAI models, ~95% for others

**Before:**
```
"Hello world" → ~3 tokens (rough estimate, ±25% error)
```

**After:**
```
"Hello world" → 2 tokens (exact, using tiktoken)
```

---

## Examples

### Example 1: Customer Support Bot (Working Memory Only)

```yaml
agents:
  support-bot:
    llm: gpt4o
    memory:
      budget: 2000
      include_history: true
      system_memory: |
        You are a helpful customer support agent.
```

**Result:**
- Remembers customer issues across conversation
- Never loses context mid-conversation
- Handles 50+ message conversations easily

### Example 2: Research Agent (Both Memory Types)

```yaml
agents:
  research-agent:
    llm: gpt4o
    memory:
      strategy: "summary_buffer"
      budget: 3000
      
      long_term:
        storage_scope: "all"
        auto_recall: true
        recall_limit: 5
```

**Result:**
- Working memory handles current research session
- Long-term memory recalls specific facts from earlier
- Can reference details from 100+ messages ago semantically
- Agent: "As you mentioned earlier about X..." (recalls from long-term)

### Example 3: Code Review Assistant (Long Conversations)

```yaml
agents:
  code-reviewer:
    llm: gpt4o
    memory:
      strategy: "summary_buffer"
      budget: 3000
      threshold: 0.8
      target: 0.6
      
      long_term:
        storage_scope: "conversational"  # Only dialogue
        auto_recall: true
        recall_limit: 8  # More context for code
```

**Result:**
- Retains full context of code being reviewed
- Remembers previous suggestions semantically
- Tracks changes across multiple files
- Can recall specific code snippets from earlier

