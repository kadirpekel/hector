# Knowledge Management

Hector provides a powerful, built-in system for Knowledge Management. You can connect your agents to external data sources (your codebase, docs, or database) without writing any glue code.

## Key Concepts

*   **Document Stores**: Sources of data (files, APIs, SQL).
*   **Vector Stores**: Where embeddings are saved (Chroma, Pinecone, etc.).
*   **Embedders**: Models that convert text to vectors (OpenAI, Cohere).
*   **Context Strategy**: How retrieved documents are injected into the agent's prompt.

## Minimal Configuration

To add RAG to an agent, you first define a **Document Store** and then attach it to the agent.

```yaml
# 1. Define where data comes from
document_stores:
  my_files:
    source:
      type: directory
      include: ["./docs/**/*.md", "./src/**/*.go"]

# 2. Attach it to an agent
agents:
  researcher:
    llm: claude
    document_stores: [my_files]
    include_context: true # Auto-inject relevant chunks
```

By default, Hector uses a local vector store (Chroma) and a default embedder, so the above is all you need for a local setup.

## Advanced Configuration

For production, you'll want to configure specific providers.

### 1. Configure Embedder
```yaml
embedders:
  openai_v3:
    provider: openai
    model: text-embedding-3-small
    api_key: ${OPENAI_API_KEY}
```

### 2. Configure Vector Store
```yaml
vector_stores:
  prod_db:
    type: pinecone
    api_key: ${PINECONE_API_KEY}
    environment: us-east-1
    index_name: hector-index
```

### 3. Configure Document Store
Link the store to your specific embedder and vector DB.

```yaml
document_stores:
  company_wiki:
    source:
      type: directory
      include: ["./wiki/**/*"]
    embedder: openai_v3
    vector_store: prod_db
    chunking:
      strategy: recursive
      size: 512
      overlap: 50
    watch: true # Auto-reindex on file variations
```

## Context Strategies

How does the agent use this knowledge?

### Automatic Injection (`include_context: true`)
Hector listens to the user's message, automatically queries the document store for relevant chunks, and injects them into the system prompt *before* the agent thinks.

```yaml
agents:
  helper:
    include_context: true
    include_context_limit: 5 # Max 5 chunks
```

### Tool-Based Search
If you set `document_stores: [...]` but `include_context: false`, the agent receives a `search` tool. It can *decide* when to look up information.

```yaml
agents:
  detective:
    include_context: false
    document_stores: [evidence_db]
    # Agent will see a 'search_evidence_db' tool
```

## MCP Document Parsing

Hector can use MCP servers to parse complex file types (PDFs, PPTs) before indexing.

```yaml
tools:
  docling:
    type: mcp
    command: npx
    args: ["-y", "@verikod/docling-mcp"]

document_stores:
  library:
    source:
      type: directory
      include: ["./books/**/*.pdf"]
    mcp_parsers:
      tool_names: [docling]
      extensions: [pdf]
```

## Incremental Indexing

Hector tracks indexed files via checksums to avoid re-indexing unchanged content.

```yaml
document_stores:
  code:
    source:
      type: directory
      include: ["./src/**/*.go"]
    incremental_indexing: true  # Skip unchanged files
    watch: true                 # Auto-reindex on changes
```

### How It Works

1. On startup, load existing file checksums from checkpoints
2. Compare current files against stored checksums
3. Index only new or modified files
4. Update checkpoints after successful indexing

This significantly reduces startup time for large document sets.

## Advanced Retrieval

### HyDE (Hypothetical Document Embeddings)

Generate a hypothetical answer to improve query relevance:

```yaml
document_stores:
  knowledge:
    search:
      enable_hyde: true
      hyde_llm: claude  # LLM to generate hypothetical doc
```

HyDE works by:
1. Generating a hypothetical answer to the query
2. Embedding the hypothetical answer
3. Searching for similar real documents

### Multi-Query Retrieval

Generate multiple query variations for broader coverage:

```yaml
document_stores:
  research:
    search:
      enable_multi_query: true
      multi_query_llm: claude
      multi_query_count: 3  # Generate 3 query variations
```

### Reranking

Use an LLM to reorder results by relevance:

```yaml
document_stores:
  docs:
    search:
      enable_rerank: true
      rerank_llm: claude
      rerank_max_results: 5  # Return top 5 after reranking
```

### Complete Search Configuration

```yaml
document_stores:
  production:
    search:
      top_k: 20                # Initial retrieval count
      threshold: 0.7           # Minimum similarity score
      enable_hyde: true
      hyde_llm: claude
      enable_rerank: true
      rerank_llm: gpt4
      rerank_max_results: 5
```

## Choosing a Vector Store

| Provider | Type | Best For | Persistence |
|----------|------|----------|-------------|
| **chromem** | Embedded | Development, single-instance, small datasets (<100k docs) | File-based |
| **Qdrant** | External | Production, large datasets, filtering | Server |
| **Pinecone** | Cloud | Managed infrastructure, global scale | Cloud |
| **Weaviate** | External | Hybrid search (vector + keyword) | Server |
| **Milvus** | External | Kubernetes-native, high throughput | Server |
| **Chroma** | External | Python ecosystem integration | Server/file |

**Recommendations:**

- **Starting out?** Use `chromem` (default). Zero setup, works immediately.
- **Production single-instance?** `chromem` with file persistence is sufficient for most apps.
- **Production multi-instance?** Use `Qdrant` or `Pinecone` for shared vector storage across replicas.
- **Need keyword + vector search?** Use `Weaviate` for built-in hybrid search.

## Choosing an Embedder

| Provider | Models | Notes |
|----------|--------|-------|
| **OpenAI** | `text-embedding-3-small`, `text-embedding-3-large` | Best quality/cost ratio. Recommended default. |
| **Ollama** | `nomic-embed-text`, `mxbai-embed-large`, others | Fully local, no API costs. Slower. |
| **Cohere** | `embed-english-v3.0`, `embed-multilingual-v3.0` | Strong multilingual support. |

!!! tip "Mix Providers"
    You can use different providers for LLMs and embeddings. A common pattern: Anthropic Claude for the agent, OpenAI for embeddings.

## Next Steps

- [Configure vector stores](../reference/configuration.md#vector_stores)
- [Explore Embedders](../reference/configuration.md#embedders)
- [View Document Stores Reference](../reference/configuration.md#document_stores)

