# Search Foundation: Configuration and Architecture

## Overview

Hector's search foundation provides unified semantic search capabilities across all document stores. The architecture is designed with a single source of truth (`SearchEngine.SearchWithFilter()`) that both search paths converge on, ensuring consistent behavior and performance.

## Core Architecture

### Component Hierarchy

```
┌─────────────────────────────────────────────────────────────┐
│                    Search Foundation                         │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         ParallelSearch (Generic Function)            │  │
│  │  - Handles goroutines, error recovery, cancellation │  │
│  │  - Used by both search paths                        │  │
│  │  - Deduplicates results by document ID              │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              SearchEngine                           │  │
│  │  - Query processing (normalization, embedding)     │  │
│  │  - Vector search via DatabaseProvider              │  │
│  │  - Threshold filtering                             │  │
│  │  - Single instance per DocumentStore               │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         DocumentStore                               │  │
│  │  - Wraps SearchEngine per store                      │  │
│  │  - Manages collection names                          │  │
│  │  - Provides store-level search interface            │  │
│  │  - Handles indexing and file watching                │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         DatabaseProvider (Qdrant, etc.)          │  │
│  │  - Vector similarity search                       │  │
│  │  - Collection management                         │  │
│  │  - Metadata storage                              │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## The Two Search Paths

Both paths converge on the same foundation: `SearchEngine.SearchWithFilter()`

### Path 1: Search Tool (Explicit Tool Call)

**Purpose**: LLM-initiated search via tool calling

**Flow**:
```
User Query → LLM Reasoning → Tool Call: search(query, type, stores, limit)
    ↓
SearchTool.performSearch()
    ↓
Groups stores by collections (for parallel efficiency)
    ↓
ParallelSearch across collections
    ↓
SearchEngine.SearchWithFilter() per collection  ← Foundation
    ↓
Results aggregated, sorted, limited
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
- **Limit**: Uses `SearchConfig.TopK` when limit is 0, max enforced by `SearchToolConfig.MaxLimit`

**Code Path**: `pkg/tools/search.go` → `ParallelSearch()` → `SearchEngine.SearchWithFilter()`

---

### Path 2: IncludeContext (Automatic RAG Context Injection)

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
SearchAllStores() - scoped to agent's assigned stores
    ↓
ParallelSearch across document stores
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
- **Limit**: Configurable via `include_context_limit` (defaults to `SearchConfig.TopK`)

**Code Path**: `pkg/agent/services.go` → `SearchContext()` → `SearchAllStores()` → `ParallelSearch()` → `SearchEngine.SearchWithFilter()`

---

## Configuration

### 1. Search Engine Configuration (`agent.search`)

Controls the core search behavior for semantic search.

```yaml
agents:
  my_agent:
    search:
      top_k: 10              # Default number of results (used when limit is 0)
      threshold: 0.5          # Minimum similarity score (0.0-1.0)
      preserve_case: true    # Don't lowercase queries (default: true for code search)
      search_mode: "vector"   # "vector", "hybrid", "keyword", "multi_query", or "hyde"
      hybrid_alpha: 0.5       # Blending factor for hybrid search (0.0-1.0)
      rerank:                 # Optional: LLM-based re-ranking
        enabled: false
        llm: "gpt-4o-mini"
        max_results: 20
      multi_query:            # Optional: Multi-query expansion
        enabled: false
        llm: "gpt-4o-mini"
        num_variations: 3
      hyde:                   # Optional: HyDE
        enabled: false
        llm: "gpt-4o-mini"
```

**Configuration Details**:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `top_k` | `int` | `10` | Default number of results returned when limit is not specified. Used by both search tool and include context. |
| `threshold` | `float32` | `0.5` | Minimum similarity score (0.0-1.0). Results below this threshold are filtered out. Higher = more precise, lower = more recall. |
| `preserve_case` | `bool` | `true` | If `true`, query text is not lowercased before embedding. Important for code search (e.g., `HTTP`, `API`). Whitespace is always normalized. |
| `search_mode` | `string` | `"vector"` | Search mode: `"vector"` (pure vector search), `"hybrid"` (keyword + vector), `"keyword"` (keyword-focused), `"multi_query"` (query expansion), or `"hyde"` (hypothetical documents). |
| `hybrid_alpha` | `float32` | `0.5` | Blending factor for hybrid search (0.0-1.0). 0.0 = pure keyword, 1.0 = pure vector, 0.5 = balanced. |
| `rerank` | `object` | `null` | Optional LLM-based re-ranking configuration. |
| `multi_query` | `object` | `null` | Optional multi-query expansion configuration. |
| `hyde` | `object` | `null` | Optional HyDE configuration. |

**Where It's Used**:

- `SearchEngine` uses `TopK` as default limit when `limit <= 0`
- `SearchEngine` filters results by `Threshold` after vector search
- `SearchEngine` uses `PreserveCase` during query processing

**Example**:
```yaml
agents:
  enterprise_assistant:
    search:
      top_k: 15              # Return 15 results by default
      threshold: 0.6         # Only results with 60%+ similarity
      preserve_case: true    # Preserve case for code/documentation
```

---

### 2. Search Tool Configuration (`tools.search`)

Controls the search tool behavior (Path 1).

```yaml
tools:
  search:
    max_limit: 50            # Maximum results allowed per search (safety limit)
```

**Configuration Details**:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_limit` | `int` | `50` | Maximum number of results allowed per search. This is a tool-level safety limit (must be <= `SearchEngine.MaxTopK` = 100). |

**Important Notes**:

- **No `default_limit`**: The default limit comes from `SearchConfig.TopK`, not tool config
- **No `enabled_search_types`**: Search type filtering was removed (type is now informational only)
- **Document stores**: The search tool automatically uses the agent's assigned document stores (see Document Store Access Rules below)

**Where It's Used**:

- `SearchTool` enforces `MaxLimit` when user/LLM requests more results
- Default limit (when `limit` is 0) comes from `SearchConfig.TopK` via the search engine

**Example**:
```yaml
tools:
  search:
    max_limit: 100           # Allow up to 100 results per search
```

---

### 3. Include Context Configuration (`agent.prompt.include_context`)

Controls automatic RAG context injection (Path 2).

```yaml
agents:
  my_agent:
    prompt:
      include_context: true                    # Enable automatic context injection
      include_context_limit: 10                # Max documents to include (default: uses search.top_k)
      include_context_max_length: 800          # Max content length per document in chars (default: 500)
```

**Configuration Details**:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `include_context` | `bool` | `false` | If `true`, automatically searches and injects relevant context into every prompt |
| `include_context_limit` | `int` | `search.top_k` | Maximum number of documents to include in context. If not set, uses `search.top_k` |
| `include_context_max_length` | `int` | `500` | Maximum content length per document in characters. Longer content is truncated with "..." |

**Where It's Used**:

- `PromptService.BuildMessages()` checks `IncludeContext` flag
- If enabled, calls `ContextService.SearchContext()` automatically
- Results are formatted and injected as text context before user query

**Example**:
```yaml
agents:
  enterprise_assistant:
    prompt:
      include_context: true
      include_context_limit: 15               # Include up to 15 documents
      include_context_max_length: 1000        # Up to 1000 chars per document
    search:
      top_k: 15                               # Search for 15 results
```

---

### 4. Document Store Access Rules

Controls which document stores an agent can access.

```yaml
agents:
  my_agent:
    document_stores:                           # Optional: controls store access
      - knowledge_base
      - internal_docs
```

**Access Rules (Option B - Permissive Default)**:

| Configuration | Access | Search Tool | Context Service | Prompt Context |
|--------------|--------|-------------|-----------------|----------------|
| `nil`/omitted | **All stores** | Created (empty stores = all) | Created (nil stores = all) | Shows all stores |
| `[]` (explicitly empty) | **No access** | Not created | Not created | Empty |
| `["store1", ...]` | **Only those stores** | Created (scoped) | Created (scoped) | Shows only those stores |

**Detailed Behavior**:

1. **`document_stores` is `nil` or omitted**:

   - Agent has access to **ALL** registered document stores
   - Search tool is auto-created with empty `availableStores` (searches all stores)
   - Context service is created with `nil` assigned stores (searches all stores)
   - Prompt context shows all available stores

2. **`document_stores` is `[]` (explicitly empty)**:

   - Agent has **NO access** to any document stores
   - Search tool is **NOT** created
   - Context service is **NOT** created (returns `NoOpContextService`)
   - Prompt context is empty

3. **`document_stores` is `["store1", "store2", ...]`**:

   - Agent can **ONLY access** the explicitly listed stores
   - Search tool is created and scoped to those stores
   - Context service is created and scoped to those stores
   - Prompt context shows only those stores

**Example**:
```yaml
agents:
  # Access all stores (permissive default)
  general_assistant:
    # document_stores: not specified → accesses all stores

  # No access (explicit restriction)
  isolated_agent:
    document_stores: []                        # Explicitly empty = no access

  # Scoped access (explicit assignment)
  security_agent:
    document_stores:
      - security_policies
      - compliance_docs
```

**Important Notes**:

- The search tool is **automatically created** if the agent has document store access
- The search tool is **implicitly scoped** to the agent's assigned stores (no config needed)
- Document stores come from agent assignment, **not** from tool config

---

## Complete Data Flow

### End-to-End Search Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Configuration Loading                                 │
│    - agent.search (TopK, Threshold, PreserveCase)       │
│    - agent.prompt.include_context                        │
│    - agent.document_stores                               │
│    - tools.search (MaxLimit)                            │
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. Agent Initialization                                    │
│    - Create SearchEngine with SearchConfig                 │
│    - Create ContextService (scoped to document_stores)     │
│    - Auto-create SearchTool (if document_stores accessible)│
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. User Query Arrives                                      │
│    "What are our password requirements?"                    │
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. Agent.execute() - Reasoning Loop                        │
│    - Creates ReasoningState                                 │
│    - Gets reasoning strategy                               │
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 5. BuildMessages() Called                                   │
│    PromptService.BuildMessages(ctx, query, slots, ...)    │
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 6. System Prompt Assembly                                  │
│    - Compose system prompt from slots                       │
│    - Add available document stores context                  │
│    - Add available tools context                            │
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 7. IncludeContext Check (Path 2)                            │
│    if IncludeContext && contextService != nil:          │
│        ↓                                                     │
│    ContextService.SearchContext(ctx, query)                 │
│        ↓                                                     │
│    SearchAllStores(ctx, query, limit, assignedStores)     │
│        ↓                                                     │
│    - If assignedStores is nil: search all stores           │
│    - If assignedStores has values: search only those        │
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 8. ParallelSearch Execution                                 │
│    For each store (in parallel goroutines):                 │
│      - DocumentStore.Search(ctx, query, limit)             │
│        ↓                                                     │
│      - SearchEngine.SearchWithFilter()                      │
│        ↓                                                     │
│      - Query processing:                                    │
│        * Normalize whitespace                               │
│        * Apply PreserveCase                                 │
│        * Generate embedding via EmbedderProvider            │
│        ↓                                                     │
│      - DatabaseProvider.SearchWithFilter()                  │
│        * Vector similarity search                           │
│        * Return top K results                               │
│        ↓                                                     │
│      - Threshold filtering (if Threshold > 0)               │
│        ↓                                                     │
│      - Return results                                       │
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 9. Result Aggregation                                       │
│    - Collect results from all stores                        │
│    - Deduplicate by document ID                            │
│    - Sort by score (highest first)                          │
│    - Limit to configured limit                              │
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 10. Context Text Formatting (Path 2)                        │
│     BuildMessages() formats results:                        │
│     "Relevant context from documents:\n"                    │
│     "[Data source: knowledge_base (SQL database)] ...\n"   │
│     "[Data source: wiki_content (REST API)] ...\n"         │
│     - Truncate to include_context_max_length                │
│     - Limit to include_context_limit documents              │
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 11. Message Assembly                                        │
│     messages = [                                             │
│       SystemPrompt,                                          │
│       AvailableDocumentStores,  ← Shows assigned stores     │
│       AvailableTools,                                       │
│       ContextText,        ← Search results (Path 2)        │
│       ...conversation history...,                            │
│       UserQuery                                              │
│     ]                                                        │
└──────────────────────────┬────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 12. LLM Call                                                │
│     LLM receives:                                            │
│     - System prompt with available stores/tools             │
│     - Automatic context (if IncludeContext enabled)         │
│     - User query                                             │
│     - Tool definitions (including search tool)               │
│                                                              │
│     LLM can:                                                 │
│     - Use injected context directly                          │
│     - Call search tool for additional searches (Path 1)     │
└─────────────────────────────────────────────────────────────┘
```

---

## Configuration Precedence and Defaults

### Search Limit Resolution

When a search is performed, the limit is determined as follows:

1. **Explicit limit provided**: Use that limit (e.g., `search(query, limit=20)`)
2. **Limit is 0 or not provided**: Use `SearchConfig.TopK` (default: 10)
3. **Limit exceeds MaxLimit**: Clamp to `SearchToolConfig.MaxLimit` (default: 50)
4. **Limit exceeds SearchEngine.MaxTopK**: Clamp to 100 (hard limit)

**Example**:
```yaml
agents:
  my_agent:
    search:
      top_k: 15                # Default limit
tools:
  search:
    max_limit: 50              # Max allowed
```

- User calls `search(query)` → limit = 15 (from `top_k`)
- User calls `search(query, limit=100)` → limit = 50 (clamped to `max_limit`)
- User calls `search(query, limit=5)` → limit = 5 (explicit)

### Include Context Limit Resolution

When `IncludeContext` is enabled, the document limit is determined as:

1. **`include_context_limit` set**: Use that value
2. **Not set**: Use `SearchConfig.TopK`
3. **Never exceeds**: Actual number of results returned

**Example**:
```yaml
agents:
  my_agent:
    prompt:
      include_context: true
      include_context_limit: 20    # Use 20 documents
    search:
      top_k: 15                    # Search for 15 results
```

- System searches for 15 results (from `top_k`)
- System includes up to 20 documents in context (from `include_context_limit`)
- If only 10 results found, only 10 are included

---

## Document Store Scoping

### How Scoping Works

Document store scoping ensures agents only search stores they're assigned to:

1. **Agent Assignment** (`agent.document_stores`):

   - `nil` = all stores
   - `[]` = no stores
   - `["store1", ...]` = only those stores

2. **Context Service Scoping**:

   - `ContextService.assignedStores` is set based on agent assignment
   - `SearchContext()` passes `assignedStores` to `SearchAllStores()`
   - `SearchAllStores()` only searches assigned stores

3. **Search Tool Scoping**:

   - `SearchTool.availableStores` is set based on agent assignment
   - `performSearch()` only searches assigned stores
   - Empty `availableStores` means search all stores

4. **Prompt Context Scoping**:

   - `BuildAvailableDocumentStoresContext()` queries `ContextService.GetAssignedStores()`
   - Only shows stores the agent can actually access
   - Prevents misleading instructions

### Scoping Example

```yaml
document_stores:
  knowledge_base:
    source: sql
    # ... config ...
  internal_docs:
    source: directory
    # ... config ...
  public_docs:
    source: directory
    # ... config ...

agents:
  # Agent 1: Access all stores
  general_assistant:
    # document_stores: not specified
    # → Can search: knowledge_base, internal_docs, public_docs
    # → Prompt shows: all three stores

  # Agent 2: Scoped access
  security_agent:
    document_stores:
      - knowledge_base
      - internal_docs
    # → Can search: knowledge_base, internal_docs only
    # → Prompt shows: knowledge_base, internal_docs only

  # Agent 3: No access
  isolated_agent:
    document_stores: []
    # → Cannot search any stores
    # → Prompt shows: nothing
```

---

## Parallel Search Architecture

### Unified Parallel Search Function

Both search paths use the same `ParallelSearch()` generic function:

```go
ParallelSearch[T ParallelSearchTarget, R any](
    ctx context.Context,
    targets []T,
    searchFunc ParallelSearchFunc[T, R],
) ([]ParallelSearchResult[R], error)
```

**Features**:

- **Parallel Execution**: All targets searched concurrently in goroutines
- **Error Recovery**: Panics are caught and reported as errors
- **Context Cancellation**: Respects context cancellation/timeout
- **Result Aggregation**: Collects results from all targets
- **Deduplication**: Results deduplicated by document ID across stores

**Used By**:

- **Search Tool**: Searches collections in parallel
- **IncludeContext**: Searches document stores in parallel

### Collection Grouping (Search Tool)

The search tool groups stores by collection for efficiency:

```
Stores: [knowledge_base, wiki_content, public_docs]
Collections: {
  "kb_collection": [knowledge_base, wiki_content],  // Same collection
  "public_collection": [public_docs]                // Different collection
}

Search Strategy:
  - Search "kb_collection" once → get results for both stores
  - Search "public_collection" once → get results for public_docs
  - Aggregate and deduplicate
```

This reduces redundant searches when multiple stores share a collection.

---

## Configuration Examples

### Example 1: Basic RAG Setup

```yaml
agents:
  assistant:
    # Access all document stores
    # document_stores: not specified → all stores
    
    # Enable automatic context injection
    prompt:
      include_context: true
      include_context_limit: 10
      include_context_max_length: 500
    
    # Configure search behavior
    search:
      top_k: 10              # Default 10 results
      threshold: 0.5          # 50% similarity minimum
      preserve_case: true     # Preserve case for code/docs

document_stores:
  knowledge_base:
    source: sql
    # ... SQL config ...
  wiki_content:
    source: api
    # ... API config ...
```

**Behavior**:

- Agent can search all stores (knowledge_base, wiki_content)
- Every query automatically includes up to 10 relevant documents
- Search tool is auto-created and can search all stores

---

### Example 2: Scoped Access with High Precision

```yaml
agents:
  security_agent:
    # Only access security-related stores
    document_stores:
      - security_policies
      - compliance_docs
    
    prompt:
      include_context: true
      include_context_limit: 5        # Fewer, more focused results
      include_context_max_length: 300 # Shorter snippets
    
    search:
      top_k: 5                # Search for 5 results
      threshold: 0.7          # Higher threshold = more precise
      preserve_case: true

document_stores:
  security_policies:
    source: directory
    path: ./docs/security
  compliance_docs:
    source: sql
    # ... SQL config ...
  public_docs:
    source: directory
    path: ./docs/public
```

**Behavior**:

- Agent can **only** search security_policies and compliance_docs
- Agent **cannot** search public_docs
- Higher threshold (0.7) = more precise, fewer false positives
- Prompt context shows only security_policies and compliance_docs

---

### Example 3: Search Tool Only (No Auto Context)

```yaml
agents:
  research_assistant:
    document_stores:
      - research_papers
      - academic_docs
    
    prompt:
      include_context: false   # Disable automatic context
    
    search:
      top_k: 20               # More results for research
      threshold: 0.4          # Lower threshold = more recall
      preserve_case: false    # Case-insensitive for research

tools:
  search:
    max_limit: 100            # Allow more results for research
```

**Behavior**:

- Agent can search research_papers and academic_docs
- **No automatic context injection** - agent must explicitly call search tool
- Search tool is auto-created and available
- Lower threshold (0.4) = more recall, more results
- Higher max_limit (100) = can request more results

---

### Example 4: Isolated Agent (No Document Access)

```yaml
agents:
  isolated_agent:
    document_stores: []        # Explicitly empty = no access
    
    prompt:
      include_context: false
    
    # No search config needed (no document stores)
```

**Behavior**:

- Agent has **no access** to any document stores
- Search tool is **not** created
- Context service is **not** created
- Prompt context shows no document stores

---

## Performance Considerations

### Parallel Execution

- **Store-Level Parallelism**: All document stores are searched in parallel
- **Collection-Level Parallelism**: Search tool groups by collection for efficiency
- **Timeout Protection**: 30-second timeout per search operation
- **Error Isolation**: One store failure doesn't block others

### Result Limits

| Path | Default Limit | Max Limit | Configurable |
|------|--------------|-----------|--------------|
| **IncludeContext** | `search.top_k` | `include_context_limit` | Yes |
| **Search Tool** | `search.top_k` | `tools.search.max_limit` | Yes |
| **SearchEngine** | `10` (fallback) | `100` (hard limit) | Via `search.top_k` |

### Deduplication

- Results are deduplicated by document ID across all stores
- Prevents duplicate results when stores share collections
- Sorting by score ensures best results first

### Query Processing

- **Whitespace Normalization**: Always applied for consistency
- **Case Preservation**: Controlled by `preserve_case` (default: true)
- **Embedding Generation**: Single embedding per query (cached per request)
- **Threshold Filtering**: Applied after vector search (reduces noise)

---

## Troubleshooting

### Issue: No search results returned

**Possible Causes**:

1. **Threshold too high**: Lower `search.threshold` (try 0.3-0.4)
2. **No matching content**: Check if documents are indexed
3. **Query too specific**: Try broader queries
4. **Document stores not assigned**: Check `agent.document_stores`

**Solution**:
```yaml
agents:
  my_agent:
    search:
      threshold: 0.3          # Lower threshold for more results
```

---

### Issue: Too many irrelevant results

**Possible Causes**:

1. **Threshold too low**: Increase `search.threshold` (try 0.6-0.7)
2. **Query too broad**: Use more specific queries
3. **TopK too high**: Reduce `search.top_k`

**Solution**:
```yaml
agents:
  my_agent:
    search:
      threshold: 0.7          # Higher threshold for precision
      top_k: 5                # Fewer results
```

---

### Issue: Agent can't access document stores

**Possible Causes**:

1. **`document_stores` is `[]`**: Explicitly empty = no access
2. **Stores not registered**: Check if stores are initialized
3. **Store names don't match**: Verify exact store names in config

**Solution**:
```yaml
agents:
  my_agent:
    # Option 1: Access all stores
    # document_stores: not specified
    
    # Option 2: Access specific stores
    document_stores:
      - knowledge_base        # Must match document_stores key
```

---

### Issue: IncludeContext not working

**Possible Causes**:

1. **`include_context: false`**: Check prompt config
2. **No document stores assigned**: Check `document_stores`
3. **Context service not created**: Check vector_store/embedder config

**Solution**:
```yaml
agents:
  my_agent:
    document_stores:
      - knowledge_base        # Must have stores assigned
    prompt:
      include_context: true   # Must be enabled
    vector_store: default     # Required for context service
    embedder: default         # Required for context service
```

---

## Best Practices

### 1. Configure Threshold Based on Use Case

- **High Precision** (security, compliance): `threshold: 0.7-0.8`
- **Balanced** (general RAG): `threshold: 0.5-0.6`
- **High Recall** (research, exploration): `threshold: 0.3-0.4`

### 2. Use Scoped Access for Security

- Assign only necessary stores to each agent
- Use `[]` to explicitly deny access
- Use `nil`/omitted only when agent needs all stores

### 3. Balance IncludeContext and Search Tool

- **IncludeContext**: For always-on RAG, general knowledge
- **Search Tool**: For targeted searches, specific queries
- Can use both: IncludeContext for general context, search tool for deep dives

### 4. Optimize Limits

- **IncludeContext**: Keep `include_context_limit` small (5-10) for focused context
- **Search Tool**: Use `max_limit` to prevent excessive results
- **TopK**: Set based on expected result quality (more results = more noise)

### 5. Monitor Performance

- Higher `top_k` = more embedding/search time
- More stores = more parallel searches (but faster overall)
- Threshold filtering happens after search (doesn't reduce search time)

---

## Summary

Hector's search foundation provides:

1. **Unified Architecture**: Single `SearchEngine.SearchWithFilter()` foundation
2. **Two Search Paths**: Search Tool (explicit) and IncludeContext (automatic)
3. **Parallel Execution**: Efficient parallel search across stores/collections
4. **Flexible Configuration**: Granular control over limits, thresholds, and scoping
5. **Document Store Scoping**: Fine-grained access control per agent
6. **Consistent Behavior**: Same search logic regardless of path

The configuration is designed to be intuitive:

- **Search config** controls core search behavior
- **Tool config** controls tool-level limits
- **Prompt config** controls context injection
- **Document stores** control access scoping

All components work together to provide a powerful, flexible, and performant search foundation for RAG applications.
