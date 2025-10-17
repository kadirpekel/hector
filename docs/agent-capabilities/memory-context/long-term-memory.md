---
layout: default
title: Long-term Memory
nav_order: 2
parent: Memory & Context
description: "Persistent knowledge with vector storage"
---

# Long-term Memory - Persistent Knowledge Storage 🧠

> **Persistent knowledge storage that survives across sessions with semantic search and retrieval.**

---

## Overview

Long-term memory provides persistent knowledge storage that survives across sessions:

- **Vector Storage** - Semantic similarity search with Qdrant
- **Session-scoped** - Knowledge persists across conversations
- **Auto-recall** - Relevant memories retrieved automatically
- **Manual Search** - On-demand memory retrieval

## Configuration

```yaml
agents:
  researcher:
    name: "Research Assistant"
    llm: "gpt-4o"
    memory:
      long_term:
        enabled: true
        provider: "qdrant"
        config:
          url: "http://localhost:6333"
          collection: "agent_memory"
          api_key: "${QDRANT_API_KEY}"
```

## How It Works

1. **Store Knowledge**: Agent interactions stored as vectors
2. **Semantic Search**: Find relevant memories by meaning
3. **Auto-recall**: Relevant memories retrieved automatically
4. **Context Integration**: Memories added to conversation context

## User Experience

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

## Use Cases

- **User Preferences**: "Alice prefers Python over JavaScript"
- **Conversation History**: Previous discussions and decisions
- **Domain Knowledge**: Learned facts and patterns
- **Personalization**: Tailored responses based on history

## Architecture

Long-term memory uses vector storage for semantic similarity:

```
┌─────────────────────────────────────────────┐
│           LONG-TERM MEMORY                  │
├─────────────────────────────────────────────┤
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │  VECTOR STORAGE (Qdrant)            │   │
│  │                                     │   │
│  │  • Semantic embeddings              │   │
│  │  • Similarity search                │   │
│  │  • Persistent storage               │   │
│  └─────────────────────────────────────┘   │
│                    ↕                        │
│  ┌─────────────────────────────────────┐   │
│  │  AUTO-RECALL                        │   │
│  │                                     │   │
│  │  • Context-aware retrieval          │   │
│  │  • Relevance scoring                │   │
│  │  │  • Integration with working      │   │
│  └─────────────────────────────────────┘   │
│                                             │
└─────────────────────────────────────────────┘
```

## See Also

- **[Working Memory](working-memory)** - Session-scoped context
- **[Memory Configuration](memory-configuration)** - Advanced tuning
- **[Document Stores](../knowledge-rag/document-stores)** - External knowledge
