---
title: Memory
description: Manage conversation context with working and long-term memory strategies
---

# Memory Management

Memory in Hector determines how agents remember and use conversation context. Hector provides two complementary memory systems: **working memory** for active conversations and **long-term memory** for persistent recall.

## Memory Architecture

Hector uses a **three-layer memory system**:

```
┌─────────────────────────────────────────────────┐
│                 AGENT                           │
├─────────────────────────────────────────────────┤
│                                                 │
│  ┌───────────────────────────────────────┐     │
│  │   1. SESSION STORE (Optional)         │     │
│  │   - Session metadata & history        │     │
│  │   - SQL database (SQLite/Postgres)    │     │
│  │   - Survives: Process restarts ✅     │     │
│  └───────────────────────────────────────┘     │
│              ↑ Backs                            │
│  ┌───────────────────────────────────────┐     │
│  │   2. WORKING MEMORY (Session)         │     │
│  │   - Active conversation context       │     │
│  │   - Token-managed strategies          │     │
│  │   - Auto-summarization                │     │
│  └───────────────────────────────────────┘     │
│              ↓ Store      ↑ Recall              │
│  ┌───────────────────────────────────────┐     │
│  │   3. LONG-TERM MEMORY (Optional)      │     │
│  │   - Vector database for search        │     │
│  │   - Semantic recall                   │     │
│  │   - Isolated: agent_id + session_id   │     │
│  └───────────────────────────────────────┘     │
│                                                 │
└─────────────────────────────────────────────────┘
```

**Key Points:**
- **Layer 1 (Session Store)**: Makes working memory persistent across restarts
- **Layer 2 (Working Memory)**: Manages active conversation (always present)
- **Layer 3 (Long-Term Memory)**: Provides semantic search (optional enhancement)

See [Setup Session Persistence](../how-to/setup-session-persistence.md) to enable Layer 1.

---

## Working Memory Strategies

Working memory manages the active conversation context. Choose a strategy based on your needs.

### Summary Buffer (Recommended)

Automatically summarizes old messages when approaching token limits.

**Configuration:**

```yaml
agents:
  assistant:
    memory:
      working:
        strategy: "summary_buffer"  # Default
        budget: 8000                # Token budget (default: 8000)
        threshold: 0.85              # Summarize at 85% capacity (default: 0.85)
        target: 0.7                  # Compress to 70% capacity (default: 0.7)
```

**How it works:**

1. Messages accumulate until reaching 80% of token budget (1600 tokens)
2. Hector asks the LLM to summarize older messages
3. Summary replaces old messages, freeing tokens
4. Conversation continues with summary as context
5. Summary optionally stored in long-term memory

**Best for:**
- Long conversations
- Preserving all information
- Natural conversation flow

**Example:**

```yaml
agents:
  support:
    llm: "gpt-4o"
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000
        threshold: 0.8
        target: 0.6
      longterm:
        
        storage_scope: "session"
```

---

### Buffer Window

Keeps only the most recent N messages.

**Configuration:**

```yaml
agents:
  assistant:
    memory:
      working:
        strategy: "buffer_window"
        window_size: 10  # Keep last 10 messages
```

**How it works:**

1. Maintains a sliding window of recent messages
2. When window is full, oldest message is dropped
3. New messages push out old ones
4. Dropped messages optionally stored in long-term memory

**Best for:**
- Short, focused conversations
- Low token usage
- Predictable memory size
- Fast performance

**Example:**

```yaml
agents:
  chatbot:
    llm: "gpt-4o"
    memory:
      working:
        strategy: "buffer_window"
        window_size: 20
```

---

## Long-Term Memory

Long-term memory provides persistent, semantically searchable storage across sessions.

### Prerequisites

Long-term memory requires:
- **Vector Database** (Qdrant)
- **Embedder** (Ollama with nomic-embed-text)

See [RAG & Semantic Search](rag.md) for setup.

### Configuration

```yaml
# Vector database
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334

# Embedder
embedders:
  embedder:
    type: "ollama"
    host: "http://localhost:11434"
    model: "nomic-embed-text"

# Agent with long-term memory
agents:
  assistant:
    vector_store: "qdrant"
    embedder: "embedder"
    memory:
      working:
        strategy: "summary_buffer"
        budget: 2000
      longterm:
        
        storage_scope: "session"     # all|session|conversational|summaries_only
        batch_size: 1                # Store after each message
        auto_recall: true            # Auto-inject memories
        recall_limit: 5              # Max memories per recall
        collection: "agent_memory"   # Qdrant collection name
```

### Storage Scopes

Control what gets stored in long-term memory:

| Scope | Description | Use Case |
|-------|-------------|----------|
| `all` | Store all messages (user & assistant) | Complete history tracking |
| `session` | Store summaries at end of session | Balanced approach |
| `conversational` | Store only user messages | User preference tracking |
| `summaries_only` | Store only summarized content | Minimal storage |

**Examples:**

```yaml
# Store everything
longterm:
  
  storage_scope: "all"  # Every message vectorized

# Store only summaries (efficient)
longterm:
  
  storage_scope: "summaries_only"

# Store only user messages
longterm:
  
  storage_scope: "conversational"
```

### Auto-Recall

Automatically retrieve relevant memories:

```yaml
longterm:
  
  auto_recall: true       # Automatically inject relevant memories
  recall_limit: 5         # Retrieve top 5 relevant memories
  similarity_threshold: 0.7  # Minimum similarity score
```

**How it works:**

1. User sends a message
2. Hector searches long-term memory for similar past context
3. Top N relevant memories injected into working memory
4. Agent has access to relevant past context automatically

**Disable for manual control:**

```yaml
longterm:
  
  auto_recall: false  # Use tools to explicitly recall
```

---

## Memory Decision Guide

Choose the right memory configuration:

### Simple Tasks (No Persistence Needed)

```yaml
agents:
  simple:
    memory:
      working:
        strategy: "buffer_window"
        window_size: 10
      # No long-term memory
```

### Customer Support (Session Persistence)

```yaml
agents:
  support:
    vector_store: "qdrant"
    embedder: "embedder"
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000
      longterm:
        
        storage_scope: "session"
        auto_recall: true
```

### Personal Assistant (Cross-Session Learning)

```yaml
agents:
  personal:
    vector_store: "qdrant"
    embedder: "embedder"
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000
      longterm:
        
        storage_scope: "all"
        auto_recall: true
        recall_limit: 10
```

### High-Volume Processing (Minimal Memory)

```yaml
agents:
  processor:
    memory:
      working:
        strategy: "buffer_window"
        window_size: 5
      # No long-term memory for performance
```

---

## Memory Best Practices

### Token Budget Sizing

Match budget to your LLM's context window:

```yaml
# GPT-4o (128K context)
memory:
  working:
    budget: 8000  # Leave room for response

# GPT-3.5 Turbo (16K context)
memory:
  working:
    budget: 2000  # Smaller budget
```

### Threshold Tuning

Adjust when summarization triggers:

```yaml
# Aggressive summarization (more frequent, smaller summaries)
memory:
  working:
    threshold: 0.6  # Summarize at 60%
    target: 0.4     # Compress to 40%

# Conservative summarization (less frequent, larger summaries)
memory:
  working:
    threshold: 0.9  # Summarize at 90%
    target: 0.7     # Compress to 70%
```

### Batch Size for Long-Term Storage

Control storage frequency:

```yaml
# Immediate storage (every message)
longterm:
  batch_size: 1  # Store immediately

# Batched storage (every 10 messages)
longterm:
  batch_size: 10  # Better performance

# End of session only
longterm:
  batch_size: 0  # Store only when session ends
```

### Collection Naming

Organize memories by purpose:

```yaml
agents:
  support:
    memory:
      longterm:
        collection: "support_tickets"
  
  personal:
    memory:
      longterm:
        collection: "user_preferences"
```

---

## Advanced Patterns

### Multi-Tier Memory

Combine strategies for optimal performance:

```yaml
agents:
  advanced:
    vector_store: "qdrant"
    embedder: "embedder"
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000
        threshold: 0.8
      longterm:
        
        storage_scope: "summaries_only"  # Only store summaries
        auto_recall: true
        recall_limit: 3
```

### Session-Scoped Memory

Different memory per session/user:

```yaml
agents:
  multi_user:
    vector_store: "qdrant"
    embedder: "embedder"
    memory:
      longterm:
        
        storage_scope: "session"
        collection: "user_sessions"
        # Session ID passed in API calls
```

### Selective Memory

Store only important information:

```yaml
agents:
  selective:
    memory:
      working:
        strategy: "summary_buffer"
      longterm:
        
        storage_scope: "summaries_only"  # Only summaries, not raw messages
        auto_recall: false               # Manual recall only
```

---

## Memory in Action

### Example 1: Customer Support

```yaml
agents:
  support:
    llm: "gpt-4o"
    vector_store: "qdrant"
    embedder: "embedder"
    
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000
        threshold: 0.8
      
      longterm:
        
        storage_scope: "all"
        auto_recall: true
        recall_limit: 5
        collection: "support_history"
    
    prompt:
      system_role: |
        You are a customer support agent. Use past
        interactions to provide personalized support.
```

**Result:** Agent remembers past issues and provides context-aware support.

### Example 2: Research Assistant

```yaml
agents:
  researcher:
    llm: "claude"
    vector_store: "qdrant"
    embedder: "embedder"
    
    memory:
      working:
        strategy: "buffer_window"
        window_size: 15
      
      longterm:
        
        storage_scope: "summaries_only"
        auto_recall: true
        recall_limit: 10
        collection: "research_notes"
    
    prompt:
      system_role: |
        You are a research assistant. Build on previous
        research findings and avoid redundant work.
```

**Result:** Agent builds on past research, avoiding duplication.

---

## Monitoring Memory Usage

### Debug Memory

Enable detailed logging:

```yaml
agents:
  debug:
    reasoning:
    memory:
      working:
        strategy: "summary_buffer"
        budget: 2000
```

Look for log entries like:
```
Working memory: 1500/2000 tokens (75%)
Triggering summarization at threshold 80%
Summarized 5 messages to 200 tokens
Long-term memory: Stored 1 summary
```

### Test Different Strategies

Create multiple agents to compare:

```yaml
agents:
  buffer_test:
    memory:
      working:
        strategy: "buffer_window"
        window_size: 10
  
  summary_test:
    memory:
      working:
        strategy: "summary_buffer"
        budget: 2000
```

---

## Next Steps

- **[RAG & Semantic Search](rag.md)** - Set up vector databases and embedders
- **[Tools](tools.md)** - Give agents capabilities
- **[Reasoning Strategies](reasoning.md)** - How agents think
- **[Building Enterprise RAG Systems](../blog/posts/building-enterprise-rag-systems.md)** - Complete RAG setup guide

---

## Related Topics

- **[LLM Providers](llm-providers.md)** - Configure language models
- **[Agent Overview](overview.md)** - Understanding agents
- **[Configuration Reference](../reference/configuration.md)** - All memory options

