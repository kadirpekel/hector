# Search Foundation and Data Flow Architecture

## Overview

Hector has a unified search foundation that supports three distinct search paths, each serving different purposes in the agent reasoning flow. All three paths converge on the same underlying search infrastructure.

## Search Foundation Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Search Foundation                         │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         ParallelSearch (Generic Function)            │  │
│  │  - Handles goroutines, error recovery, cancellation │  │
│  │  - Used by all three search paths                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              SearchEngine                           │  │
│  │  - Query processing (normalization, embedding)     │  │
│  │  - Vector search via DatabaseProvider              │  │
│  │  - Threshold filtering                             │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         DocumentStore                               │  │
│  │  - Wraps SearchEngine per store                      │  │
│  │  - Manages collection names                          │  │
│  │  - Provides store-level search interface            │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         DatabaseProvider (Qdrant, etc.)          │  │
│  │  - Vector similarity search                       │  │
│  │  - Collection management                         │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## The Two Search Paths

Both paths converge on the same foundation: `SearchEngine.SearchWithFilter()`

### Path 1: Search Tool (Explicit Tool Call)

**Purpose**: User-initiated search via LLM tool calling

**Flow**:
```
User Query → LLM → Tool Call: search(query, type, stores, limit)
    ↓
SearchTool.performSearch()
    ↓
Groups stores by collections
    ↓
ParallelSearch across collections
    ↓
SearchEngine.SearchWithFilter() per collection  ← Foundation
    ↓
Returns formatted JSON response to LLM
    ↓
LLM uses results in reasoning
```

**Characteristics**:
- **Trigger**: LLM explicitly calls the `search` tool
- **Control**: LLM decides when and what to search
- **Output**: Structured JSON with results, metadata, suggestions
- **Use Case**: Agent needs to actively search for information during reasoning
- **Limit**: Configurable (default 10, max 50)

**Code Path**: `pkg/tools/search.go` → `ParallelSearch()` → `SearchEngine.SearchWithFilter()`

---

### Path 2: SearchContext / IncludeContext (Automatic Context Injection)

**Purpose**: Automatic RAG context injection during prompt building

**Flow**:
```
Agent.execute() → BuildMessages()
    ↓
PromptService.BuildMessages()
    ↓
If IncludeContext enabled:
    ↓
ContextService.SearchContext(query)
    ↓
SearchAllStores() - searches all registered stores
    ↓
ParallelSearch across all document stores
    ↓
DocumentStore.Search() per store
    ↓
SearchEngine.SearchWithFilter() per store  ← Foundation (same as Path 1)
    ↓
Results formatted as text context
    ↓
Injected into prompt before user query
    ↓
LLM receives context automatically
```

**Characteristics**:
- **Trigger**: Automatic during prompt building if `IncludeContext: true`
- **Control**: System-controlled, happens every iteration
- **Output**: Formatted text context appended to messages
- **Use Case**: Always-on RAG - agent always has relevant context
- **Limit**: Fixed at 5 results (hardcoded in `BuildMessages`)

**Code Path**: `pkg/agent/services.go` → `SearchContext()` → `SearchAllStores()` → `ParallelSearch()` → `SearchEngine.SearchWithFilter()`

---


## Complete Data Flow: Search to Prompt Building

### Step-by-Step Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. User Query Arrives                                        │
│    "What are our password requirements?"                     │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. Agent.execute() - Reasoning Loop Starts                 │
│    - Creates ReasoningState                                  │
│    - Gets reasoning strategy (ChainOfThought, etc.)         │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. BuildMessages() Called                                    │
│    PromptService.BuildMessages(ctx, query, slots, ...)      │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. System Prompt Assembly                                    │
│    - Compose system prompt from slots                        │
│    - Add additional context from strategy                    │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 5. IncludeContext Check                                      │
│    if IncludeContext && contextService != nil:          │
│        ↓                                                     │
│    ContextService.SearchContext(ctx, query)                 │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 6. SearchAllStores() - Parallel Search                       │
│    - Get all registered document stores                     │
│    - Create search targets (one per store)                   │
│    - Call ParallelSearch()                                   │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 7. ParallelSearch Execution                                  │
│    For each store (in parallel goroutines):                 │
│      - DocumentStore.Search(ctx, query, limit)             │
│        ↓                                                     │
│      - SearchEngine.SearchWithFilter()                      │
│        ↓                                                     │
│      - Query processing (normalize, embed)                  │
│        ↓                                                     │
│      - DatabaseProvider.SearchWithFilter()                   │
│        ↓                                                     │
│      - Vector similarity search                              │
│        ↓                                                     │
│      - Threshold filtering                                  │
│        ↓                                                     │
│      - Return results                                       │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 8. Result Aggregation                                        │
│    - Collect results from all stores                        │
│    - Deduplicate by document ID                            │
│    - Sort by score (highest first)                          │
│    - Limit to top 5 results                                 │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 9. Context Text Formatting                                  │
│    BuildMessages() formats results:                         │
│    "Relevant context from documents:\n"                     │
│    "[Data source: knowledge_base] ...content...\n"          │
│    "[Data source: wiki_content] ...content...\n"            │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 10. Message Assembly                                         │
│     messages = [                                             │
│       SystemPrompt,                                          │
│       AdditionalContext,                                     │
│       ContextText,        ← Search results injected here    │
│       ...conversation history...,                            │
│       UserQuery                                              │
│     ]                                                        │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 11. LLM Call                                                 │
│     callLLMWithRetry(messages, toolDefs, ...)                │
│     - LLM receives context automatically                    │
│     - LLM can also call search tool if needed                │
└─────────────────────────────────────────────────────────────┘
```

## Why Two Paths?

### 1. **Search Tool** - Explicit Control
- **When**: Agent needs to actively search during reasoning
- **Why**: LLM decides what to search and when
- **Example**: "Search for authentication code" → LLM calls search tool
- **Foundation**: `SearchEngine.SearchWithFilter()`

### 2. **IncludeContext** - Automatic RAG
- **When**: Every prompt building cycle (if enabled)
- **Why**: Always provide relevant context without LLM intervention
- **Example**: User asks "What are our password requirements?" → System automatically searches and injects relevant docs
- **Foundation**: `SearchEngine.SearchWithFilter()` (same as Path 1)

## Key Design Decisions

### Unified Parallel Search
Both paths use `ParallelSearch()` for:
- Consistent error handling
- Context cancellation support
- Panic recovery
- Result aggregation

### SearchEngine.SearchWithFilter() as Foundation
**Both paths converge here** - this is the single source of truth for search:
- Single `SearchEngine` instance per `DocumentStore`
- Handles query processing (normalization, embedding)
- Performs vector search via `DatabaseProvider`
- Provides threshold filtering
- Abstracts database provider details

**Key Insight**: Path 1 and Path 2 are just different ways to call the same underlying search function. The only difference is:
- **Path 1**: LLM-initiated, returns structured JSON
- **Path 2**: System-initiated, returns formatted text for prompt injection

### DocumentStore Abstraction
- Each store has its own `SearchEngine`
- Stores can share collections (via `Collection` config)
- `SearchAllStores()` searches all stores in parallel

## Configuration

### Enable IncludeContext
```yaml
agents:
  my_agent:
    prompt:
      include_context: true  # Enables automatic context injection
```

### Configure Search Tool
```yaml
tools:
  search:
    default_limit: 10
    max_limit: 50
    enabled_search_types: [content, file, function, struct]
```

### Search Engine Settings
```yaml
agents:
  my_agent:
    search:
      top_k: 10
      threshold: 0.7
      preserve_case: false
```

## Performance Considerations

1. **Parallel Execution**: Both paths search in parallel across stores/collections
2. **Deduplication**: Results deduplicated by document ID
3. **Limits**: 
   - IncludeContext: 5 results (hardcoded)
   - Search Tool: Configurable (default 10, max 50)
4. **Caching**: Embeddings could be cached (future optimization)

## Future Improvements

1. Make IncludeContext limit configurable (currently hardcoded to 5)
2. Add result caching for repeated queries
3. Support search result ranking/reranking
4. Add search analytics/metrics

