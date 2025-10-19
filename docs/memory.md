---
title: Memory Management
description: Dual-layer memory system for AI agents - working memory and long-term memory
---

# Memory Management - Never Lose Context

!!! quote "Dual-layer intelligent memory: Working memory for sessions + Long-term memory for persistent knowledge."

---

## Overview

Hector implements a **cognitive memory architecture** inspired by human memory systems:

```
┌─────────────────────────────────────────────────────────────┐
│                HECTOR MEMORY SYSTEM                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │            WORKING MEMORY (Session-Scoped)              │ │
│  │  ┌─────────────────────────────────────────────────────┐ │ │
│  │  │ • Current conversation context                       │ │ │
│  │  │ • Token-based management                             │ │ │
│  │  │ • Automatic summarization                           │ │ │
│  │  │ • Strategy: summary_buffer/buffer                    │ │ │
│  │  └─────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │            LONG-TERM MEMORY (Persistent)                 │ │
│  │  ┌─────────────────────────────────────────────────────┐ │ │
│  │  │ • Vector-based storage (Qdrant)                     │ │ │
│  │  │ • Semantic search & recall                          │ │ │
│  │  │ • Session-scoped persistence                        │ │ │
│  │  │ • Auto-recall or on-demand                          │ │ │
│  │  └─────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  Working Memory ←→ Long-Term Memory                          │
└─────────────────────────────────────────────────────────────┘
```
**Two Memory Types:**

1. **Working Memory** - Manages conversation history within sessions (like human short-term memory)
2. **Long-Term Memory** - Stores and recalls relevant context semantically (like human long-term memory)

Both work together seamlessly to provide optimal context management.

---

## Working Memory Strategies

Working memory manages conversation context within sessions using different strategies:

### 1. **Summary Buffer** (Recommended)

Automatically summarizes conversation history when token limits are reached.

```yaml
memory:
  strategy: "summary_buffer"
  max_tokens: 4000
  summary_threshold: 0.8
  long_term:
    enabled: true
    strategy: "vector_memory"
```

**How it works:**
1. :material-chat: **Conversation grows** - Messages accumulate in working memory
2. :material-chart-line: **Threshold reached** - When 80% of max_tokens is used
3. :material-compress: **Auto-summarize** - Older messages are summarized
4. :material-database: **Store in LTM** - Summary stored in long-term memory
5. :material-refresh: **Continue fresh** - Conversation continues with summary context

**Benefits:**
- **Never lose context** - Important information preserved in summaries
- **Efficient token usage** - Stays within token limits
- **Seamless experience** - Users don't notice summarization
- **Long-term recall** - Summaries available for future sessions

### 2. **Buffer Window**

Keeps a fixed number of recent messages, discarding older ones.

```yaml
memory:
  strategy: "buffer_window"
  max_tokens: 4000
  window_size: 10
  long_term:
    enabled: true
    strategy: "vector_memory"
```

**How it works:**
1. :material-chat: **Messages accumulate** - New messages added to buffer
2. :material-counter: **Window limit** - When window_size (10) messages reached
3. :material-delete: **Remove oldest** - Oldest message removed
4. :material-database: **Store in LTM** - Removed message stored in long-term memory
5. :material-refresh: **Continue** - Buffer continues with recent messages

**Benefits:**
- **Simple and predictable** - Always keeps N recent messages
- **Fast performance** - No summarization overhead
- **Exact context** - No information loss from summarization
- **Memory efficient** - Fixed memory usage

### 3. **Vector Memory**

Uses vector embeddings for all memory management.

```yaml
memory:
  strategy: "vector_memory"
  max_tokens: 4000
  collection: "agent_memory"
  similarity_threshold: 0.7
```

**How it works:**
1. :material-chat: **Messages embedded** - Each message converted to vector
2. :material-database: **Vector storage** - Stored in vector database
3. :material-magnify: **Semantic search** - Relevant context retrieved by similarity
4. :material-refresh: **Dynamic context** - Context changes based on current query

**Benefits:**
- **Semantic understanding** - Finds relevant context by meaning
- **Scalable** - Handles large amounts of historical data
- **Intelligent recall** - Retrieves most relevant information
- **No token limits** - Not constrained by token counts

---

## Long-Term Memory

Long-term memory provides persistent knowledge storage and retrieval:

### Vector-Based Storage

```yaml
memory:
  strategy: "summary_buffer"
  max_tokens: 4000
  long_term:
    enabled: true
    strategy: "vector_memory"
    collection: "agent_long_term_memory"
    similarity_threshold: 0.7
    auto_recall: true
```

**Features:**
- :material-database: **Persistent Storage** - Information survives session restarts
- :material-magnify: **Semantic Search** - Find relevant information by meaning
- :material-chart-line: **Similarity Scoring** - Rank results by relevance
- :material-refresh: **Auto-Recall** - Automatically retrieve relevant context

### Long-Term Memory Types

=== "Session Memory"
    ```yaml
    long_term:
      strategy: "vector_memory"
      collection: "session_memory"
      scope: "session"
      auto_recall: true
    ```
    
    **Purpose:** Store conversation summaries and important context from current session

=== "User Memory"
    ```yaml
    long_term:
      strategy: "vector_memory"
      collection: "user_memory"
      scope: "user"
      auto_recall: true
    ```
    
    **Purpose:** Store user preferences, personal information, and interaction history

=== "Global Memory"
    ```yaml
    long_term:
      strategy: "vector_memory"
      collection: "global_memory"
      scope: "global"
      auto_recall: false
    ```
    
    **Purpose:** Store general knowledge, facts, and shared information

---

## Memory Configuration

### Complete Memory Configuration

```yaml
memory:
  # Working memory strategy
  strategy: "summary_buffer"           # "summary_buffer", "buffer_window", "vector_memory"
  
  # Token management
  max_tokens: 4000                    # Maximum tokens in working memory
  summary_threshold: 0.8              # When to trigger summarization (0.0-1.0)
  
  # Buffer window specific
  window_size: 10                     # Number of messages to keep
  
  # Vector memory specific
  collection: "agent_memory"          # Vector database collection name
  similarity_threshold: 0.7           # Similarity threshold for retrieval
  
  # Long-term memory
  long_term:
    enabled: true                     # Enable long-term memory
    strategy: "vector_memory"          # Long-term memory strategy
    collection: "agent_long_term"     # Long-term memory collection
    similarity_threshold: 0.7         # Similarity threshold for recall
    auto_recall: true                 # Automatically recall relevant context
    max_recall_items: 5               # Maximum items to recall
  
  # Memory persistence
  persistence:
    enabled: true                     # Enable memory persistence
    storage: "qdrant"                 # Storage backend
    backup: true                      # Enable automatic backups
    retention_days: 30                # How long to keep memories
```

### Memory Strategy Comparison

| Strategy | Token Usage | Context Preservation | Performance | Use Case |
|----------|-------------|---------------------|-------------|----------|
| **Summary Buffer** | :material-check: Efficient | :material-check: High | :material-check: Good | General purpose |
| **Buffer Window** | :material-check: Fixed | :material-warning: Limited | :material-check: Excellent | Simple tasks |
| **Vector Memory** | :material-warning: Variable | :material-check: Excellent | :material-warning: Moderate | Complex tasks |

---

## Memory Lifecycle

### Memory Flow

```
┌─────────────────────────────────────────────────────────────┐
│                HECTOR MEMORY SYSTEM                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │            WORKING MEMORY (Session-Scoped)              │ │
│  │  ┌─────────────────────────────────────────────────────┐ │ │
│  │  │ • Current conversation context                       │ │ │
│  │  │ • Token-based management                             │ │ │
│  │  │ • Automatic summarization                           │ │ │
│  │  │ • Strategy: summary_buffer/buffer                    │ │ │
│  │  └─────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │            LONG-TERM MEMORY (Persistent)                 │ │
│  │  ┌─────────────────────────────────────────────────────┐ │ │
│  │  │ • Vector-based storage (Qdrant)                     │ │ │
│  │  │ • Semantic search & recall                          │ │ │
│  │  │ • Session-scoped persistence                        │ │ │
│  │  │ • Auto-recall or on-demand                          │ │ │
│  │  └─────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  Working Memory ←→ Long-Term Memory                          │
└─────────────────────────────────────────────────────────────┘
```
### Memory Operations

1. :material-plus: **Store** - Save new information to memory
2. :material-magnify: **Search** - Find relevant information
3. :material-refresh: **Recall** - Retrieve context for current task
4. :material-delete: **Forget** - Remove outdated information
5. :material-compress: **Summarize** - Compress information for efficiency

---

## Memory Best Practices

### Configuration Best Practices

=== "Development"
    ```yaml
    memory:
      strategy: "buffer_window"
      max_tokens: 2000
      window_size: 5
      long_term:
        enabled: false
    ```
    
    **Why:** Simple, fast, no external dependencies

=== "Production"
    ```yaml
    memory:
      strategy: "summary_buffer"
      max_tokens: 4000
      summary_threshold: 0.8
      long_term:
        enabled: true
        strategy: "vector_memory"
        collection: "production_memory"
        auto_recall: true
    ```
    
    **Why:** Optimal balance of performance and context preservation

=== "High-Context Tasks"
    ```yaml
    memory:
      strategy: "vector_memory"
      max_tokens: 8000
      collection: "high_context_memory"
      similarity_threshold: 0.6
      long_term:
        enabled: true
        strategy: "vector_memory"
        collection: "high_context_ltm"
        auto_recall: true
    ```
    
    **Why:** Maximum context preservation for complex tasks

### Memory Optimization

- :material-chart-line: **Monitor token usage** - Track memory efficiency
- :material-database: **Regular cleanup** - Remove outdated memories
- :material-compress: **Optimize summaries** - Fine-tune summarization
- :material-magnify: **Tune similarity** - Adjust recall thresholds

---

## Memory Troubleshooting

### Common Issues

=== "High Token Usage"
    **Problem:** Memory using too many tokens
    
    **Solutions:**
    - Reduce `max_tokens`
    - Increase `summary_threshold`
    - Use `buffer_window` strategy
    - Enable long-term memory

=== "Poor Context Recall"
    **Problem:** Agent not recalling relevant information
    
    **Solutions:**
    - Lower `similarity_threshold`
    - Increase `max_recall_items`
    - Enable `auto_recall`
    - Check vector database connection

=== "Memory Performance"
    **Problem:** Slow memory operations
    
    **Solutions:**
    - Use `buffer_window` for simple tasks
    - Optimize vector database
    - Reduce `max_tokens`
    - Enable memory caching

### Debugging Memory

```yaml
# Enable memory debugging
memory:
  strategy: "summary_buffer"
  max_tokens: 4000
  debug: true                        # Enable debug logging
  
  long_term:
    enabled: true
    strategy: "vector_memory"
    debug: true                       # Enable LTM debug logging
```

---

## Memory Monitoring

### Memory Metrics

- :material-chart-line: **Token Usage** - Current and peak token usage
- :material-database: **Memory Size** - Number of stored memories
- :material-magnify: **Recall Accuracy** - Relevance of recalled memories
- :material-timer: **Performance** - Memory operation timing

### Monitoring Configuration

```yaml
# Memory monitoring
monitoring:
  memory:
    enabled: true
    metrics:
      - "token_usage"
      - "memory_size"
      - "recall_accuracy"
      - "performance"
    
    alerts:
      high_token_usage:
        threshold: 0.9
        action: "summarize"
      
      low_recall_accuracy:
        threshold: 0.5
        action: "tune_similarity"
```

---

## Related Documentation

- [Building Agents](agents.md) - Learn how to build AI agents
- [Configuration Reference](configuration.md) - Complete configuration guide
- [Architecture](architecture.md) - Understand Hector's architecture
- [Tools & Extensions](tools.md) - Built-in tools and custom extensions
- [Plugin Development](plugins.md) - Build custom plugins
