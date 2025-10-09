# Future: Vector Database-Backed Memory

**Optional Enhancement - Not Yet Implemented**

This document outlines a proposed future enhancement to Hector's memory system using vector databases for semantic search through conversation history.

---

## Overview

Currently, Hector provides:
- ✅ Accurate token counting
- ✅ Intelligent message selection
- ✅ Optional LLM summarization

This proposal adds:
- 🔮 Semantic search through past conversations
- 🔮 Cross-session learning and memory
- 🔮 Persistent conversation storage
- 🔮 User-specific fact storage

---

## Why Vector DB for Memory?

### 1. Semantic Search Through History ✨

**Current System:**
```
- Messages stored in memory
- Lost on server restart
- No semantic search
```

**With Vector DB:**
```
User: "How do I secure my API?"

System searches ALL past conversations semantically:
- "Setting up JWT authentication" (2 weeks ago)
- "API security best practices" (1 month ago)
- "Implementing OAuth" (yesterday)

Agent: "Based on our previous discussions about JWT..."
```

### 2. Cross-Session Learning 🧠

```go
// Find when you solved similar problems before
pastSolutions := memory.FindSimilarInteractions(
    query: "React performance issues",
    userID: "alice",
    minSimilarity: 0.85
)

// Agent remembers how you solved it last time
```

### 3. Unified Architecture 🏗️

Hector already uses Qdrant for RAG:
```
Documents → Qdrant → Semantic search ✅ (current)
Conversations → Qdrant → Semantic search 🔮 (future)
```

---

## Proposed Configuration

```yaml
agents:
  my_agent:
    llm: gpt-4o
    
    # Current: Token-aware memory (implemented)
    memory:
      budget: 2000
      summarization: true
    
    # Future: Vector-backed memory (proposed)
    memory:
      budget: 2000
      vector_storage:
        enabled: true
        database: qdrant
        collections:
          conversations: "agent_conversations"
          user_facts: "user_knowledge"
          episodes: "solved_problems"
        retention_days: 90
```

---

## Proposed Collections

### 1. Conversations Collection
Stores all conversation messages with embeddings:
```
{
  "id": "msg-uuid",
  "session_id": "session-123",
  "user_id": "alice",
  "role": "assistant",
  "content": "To implement JWT authentication...",
  "timestamp": "2025-01-15T10:30:00Z",
  "vector": [0.123, 0.456, ...],  // embedding
  "metadata": {
    "agent_id": "coding-assistant",
    "topics": ["authentication", "JWT", "security"]
  }
}
```

### 2. User Facts Collection
Stores learned facts about users:
```
{
  "id": "fact-uuid",
  "user_id": "alice",
  "fact": "Uses React for frontend development",
  "confidence": 0.95,
  "last_confirmed": "2025-01-15",
  "vector": [0.789, 0.012, ...],
  "sources": ["session-123", "session-456"]
}
```

### 3. Episodes Collection
Stores problem-solution pairs:
```
{
  "id": "episode-uuid",
  "user_id": "alice",
  "problem": "API rate limiting returning 429 errors",
  "solution": "Implemented exponential backoff with jitter",
  "context": "Using Axios in React app",
  "timestamp": "2025-01-10",
  "vector": [0.234, 0.567, ...],
  "outcome": "success"
}
```

---

## Proposed API

### Search Past Conversations
```go
// Semantic search through conversation history
results := memory.SearchConversations(SearchRequest{
    Query:    "authentication setup",
    UserID:   "alice",
    AgentID:  "coding-assistant",
    Limit:    5,
    MinScore: 0.8,
})
```

### Remember User Facts
```go
// Store learned facts
memory.StoreUserFact(UserFact{
    UserID:     "alice",
    Fact:       "Prefers TypeScript over JavaScript",
    Confidence: 0.9,
    Source:     currentSessionID,
})

// Retrieve relevant facts
facts := memory.GetRelevantFacts(
    userID: "alice",
    context: "Starting new frontend project",
)
```

### Find Similar Past Solutions
```go
// Find how similar problems were solved before
episodes := memory.FindSimilarEpisodes(EpisodeQuery{
    Problem:  "Database connection timeouts",
    UserID:   "alice",
    Limit:    3,
    MinScore: 0.85,
})
```

---

## Benefits

### For Users
- ✅ Agents remember across sessions
- ✅ No need to repeat context
- ✅ Personalized responses based on history
- ✅ Learns your preferences over time

### For Developers
- ✅ Uses existing Qdrant integration
- ✅ No new infrastructure needed
- ✅ Opt-in feature (backward compatible)
- ✅ Standard vector DB queries

### For System
- ✅ Persistent storage (survives restarts)
- ✅ Scalable (Qdrant handles millions of vectors)
- ✅ Fast semantic search (sub-100ms)
- ✅ Built-in clustering and analytics

---

## Implementation Phases

### Phase 1: Conversation Storage
- Store messages in Qdrant
- Basic retrieval by session
- Persistence across restarts

### Phase 2: Semantic Search
- Embed conversation messages
- Search across sessions
- Relevance ranking

### Phase 3: User Facts
- Extract and store facts
- Fact validation and confidence
- Automatic fact retrieval

### Phase 4: Episodic Memory
- Problem-solution storage
- Similar episode finding
- Learning from past interactions

---

## Why Not SQL/Redis?

### SQL (PostgreSQL, MySQL)
- ❌ No semantic search
- ❌ Only keyword/exact match
- ❌ Can't find similar conversations
- ❌ Requires separate embedding infrastructure

### Redis
- ❌ Primarily for caching
- ❌ Limited persistence options
- ❌ No native vector operations (without RedisSearch)
- ❌ Not designed for semantic search

### Qdrant (Proposed)
- ✅ Native semantic search
- ✅ Already integrated with Hector
- ✅ Persistent by default
- ✅ Designed for vector operations
- ✅ Fast and scalable

---

## Example Use Cases

### 1. Customer Support Agent
```
User: "I'm having the same problem again"
Agent searches past conversations:
- Finds the previous issue from 2 months ago
- Remembers the solution that worked
- Applies the same fix immediately
```

### 2. Coding Assistant
```
User: "How do I handle API errors?"
Agent finds:
- Your previous error handling patterns
- Solutions that worked for your stack
- Code snippets from your past projects
```

### 3. Personal Assistant
```
User: "Schedule a meeting"
Agent remembers:
- Your preferred meeting times
- Your timezone
- Your calendar preferences
- Past scheduling conflicts
```

---

## Current Status

- ⚠️ **Not Yet Implemented** - This is a design proposal
- ✅ Infrastructure ready (Qdrant + Embedders already integrated)
- ✅ Current memory system works well for most use cases
- 🔮 This is an optional enhancement for advanced scenarios

For current memory capabilities, see:
- [Memory Configuration Guide](MEMORY_CONFIGURATION.md)
- [Memory Documentation](MEMORY.md)

---

## Feedback Welcome

This is a proposed design for discussion. Feedback is welcome on:
- Use cases and requirements
- API design
- Implementation priorities
- Configuration approach

Open an issue or PR to contribute to the design!

