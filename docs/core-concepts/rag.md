---
title: RAG & Semantic Search
description: Give agents semantic search capabilities with vector databases and embeddings
---

# RAG & Semantic Search

RAG (Retrieval-Augmented Generation) gives agents the ability to search through documents semantically—finding information by meaning, not just keywords.

## What is RAG?

Traditional search: "Find files containing 'authentication'"  
Semantic search: "Find code related to user login"

RAG allows agents to:
- Search codebases by meaning
- Find relevant documentation
- Discover similar patterns
- Answer questions from knowledge bases

---

## Prerequisites

RAG requires two components:

1. **Vector Database** - Stores document embeddings (Qdrant)
2. **Embedder** - Converts text to vectors (Ollama)

---

## Quick Setup

### 1. Start Qdrant

```bash
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -p 6334:6334 \
  -v $(pwd)/qdrant_storage:/qdrant/storage \
  qdrant/qdrant
```

Verify: http://localhost:6333/dashboard

### 2. Start Ollama

```bash
# Install Ollama (macOS/Linux)
curl https://ollama.ai/install.sh | sh

# Pull embedding model
ollama pull nomic-embed-text
```

### 3. Configure Hector

```yaml
# Vector database
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6333

# Embedder
embedders:
  embedder:
    type: "ollama"
    host: "http://localhost:11434"
    model: "nomic-embed-text"

# Agent with semantic search
agents:
  coder:
    database: "qdrant"
    embedder: "embedder"
    tools: ["search"]
    document_stores:
      - name: "codebase"
        paths: ["./src/"]
        file_patterns: ["*.go", "*.py", "*.js"]
```

### 4. Test It

```bash
hector call coder "How does authentication work in this codebase?"
```

The agent will semantically search your code and answer!

---

## Document Stores

Document stores define what gets indexed for search.

### Basic Configuration

```yaml
agents:
  assistant:
    database: "qdrant"
    embedder: "embedder"
    document_stores:
      - name: "docs"
        paths: ["./documentation/"]
        file_patterns: ["*.md"]
```

### Multiple Document Stores

```yaml
agents:
  researcher:
    database: "qdrant"
    embedder: "embedder"
    document_stores:
      - name: "codebase"
        paths: ["./src/", "./lib/"]
        file_patterns: ["*.go", "*.py"]
        chunk_size: 512
      
      - name: "documentation"
        paths: ["./docs/"]
        file_patterns: ["*.md"]
        chunk_size: 1024
      
      - name: "configs"
        paths: ["./configs/"]
        file_patterns: ["*.yaml", "*.json"]
        chunk_size: 256
```

### Configuration Options

```yaml
document_stores:
  - name: "my_store"
    paths: ["./path1/", "./path2/"]
    file_patterns: ["*.ext"]
    
    # Chunking
    chunk_size: 512           # Characters per chunk
    chunk_overlap: 50         # Overlap between chunks
    
    # Parsing
    parser: "native"          # native|custom|plugin
    
    # Indexing
    collection: "my_collection"  # Qdrant collection name
    batch_size: 100           # Documents per batch
    
    # Filtering
    exclude_patterns: ["*_test.go", "*.min.js"]
```

---

## How RAG Works

### Indexing Phase

```
1. Hector reads documents from paths
   ├─ ./src/auth.go
   ├─ ./src/user.go
   └─ ./src/db.go

2. Documents split into chunks
   ├─ Chunk 1: "package auth..."
   ├─ Chunk 2: "func Login..."
   └─ Chunk 3: "func Validate..."

3. Each chunk converted to embedding
   ├─ [0.23, -0.45, 0.67, ...] (768 dimensions)
   ├─ [0.12, -0.34, 0.56, ...]
   └─ [-0.45, 0.23, 0.78, ...]

4. Embeddings stored in Qdrant
   ├─ Collection: "codebase"
   └─ Indexed for fast similarity search
```

### Search Phase

```
1. User asks: "How does authentication work?"

2. Query embedded: [0.25, -0.43, 0.69, ...]

3. Qdrant finds similar chunks (cosine similarity)
   ├─ auth.go chunk (similarity: 0.92)
   ├─ user.go chunk (similarity: 0.85)
   └─ db.go chunk (similarity: 0.78)

4. Top chunks injected into agent context

5. Agent answers using retrieved context
```

---

## Vector Databases

### Qdrant (Recommended)

```yaml
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6333           # gRPC port (default: 6333)
    grpc_port: 6334      # REST dashboard port (default: 6334)
    api_key: ""          # Optional for Qdrant Cloud
    use_https: false     # Enable for cloud
```

**Docker:**
```bash
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -p 6334:6334 \
  -v qdrant_data:/qdrant/storage \
  qdrant/qdrant
```

**Qdrant Cloud:**
```yaml
databases:
  qdrant_cloud:
    type: "qdrant"
    host: "your-cluster.qdrant.io"
    port: 6333
    api_key: "${QDRANT_API_KEY}"
    use_https: true
```

### Custom Vector Databases (Plugins)

```yaml
plugins:
  databases:
    - name: "my-vector-db"
      protocol: "grpc"
      path: "/path/to/plugin"

databases:
  custom:
    type: "plugin:my-vector-db"
    # Custom configuration
```

---

## Embedders

### Ollama (Recommended)

```yaml
embedders:
  embedder:
    type: "ollama"
    host: "http://localhost:11434"
    model: "nomic-embed-text"  # Best for code
    timeout: 30
```

**Available Models:**
- `nomic-embed-text` - General purpose, 768 dimensions (recommended)
- `all-minilm` - Lightweight, 384 dimensions
- `mxbai-embed-large` - Large, 1024 dimensions

**Setup:**
```bash
ollama pull nomic-embed-text
```

### Custom Embedders (Plugins)

```yaml
plugins:
  embedders:
    - name: "my-embedder"
      protocol: "grpc"
      path: "/path/to/plugin"

embedders:
  custom:
    type: "plugin:my-embedder"
    # Custom configuration
```

---

## Advanced Configuration

### Chunking Strategy

Balance between context and precision:

```yaml
# Small chunks (precise, less context)
document_stores:
  - name: "precise"
    chunk_size: 256
    chunk_overlap: 25
    # Good for: Code snippets, specific facts

# Medium chunks (balanced)
document_stores:
  - name: "balanced"
    chunk_size: 512
    chunk_overlap: 50
    # Good for: General purpose

# Large chunks (more context, less precise)
document_stores:
  - name: "contextual"
    chunk_size: 2048
    chunk_overlap: 200
    # Good for: Documentation, narratives
```

### Search Configuration

```yaml
agents:
  searcher:
    database: "qdrant"
    embedder: "embedder"
    document_stores:
      - name: "docs"
        paths: ["./"]
        search_config:
          limit: 5              # Top 5 results
          score_threshold: 0.7  # Minimum similarity score
          filter: {}            # Optional metadata filters
```

### Custom Parsers

Parse non-standard formats:

```yaml
plugins:
  parsers:
    - name: "pdf-parser"
      protocol: "grpc"
      path: "/path/to/parser"

document_stores:
  - name: "pdfs"
    paths: ["./documents/"]
    file_patterns: ["*.pdf"]
    parser: "plugin:pdf-parser"
```

---

## Performance Optimization

### Indexing Performance

```yaml
document_stores:
  - name: "large_codebase"
    paths: ["./"]
    batch_size: 100        # Index 100 docs at a time
    parallel: true         # Parallel processing
    cache_embeddings: true # Cache for faster re-indexing
```

### Search Performance

```yaml
agents:
  fast_search:
    document_stores:
      - name: "optimized"
        search_config:
          limit: 3           # Fewer results = faster
          score_threshold: 0.8  # Higher threshold = fewer candidates
          use_cache: true    # Cache frequent queries
```

### Resource Management

```yaml
# Ollama configuration
embedders:
  embedder:
    type: "ollama"
    host: "http://localhost:11434"
    timeout: 30
    batch_size: 32  # Embed 32 chunks at once

# Qdrant configuration
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6333
    connection_pool_size: 10  # Connection pooling
```

---

## Use Cases

### Code Search

```yaml
agents:
  code_assistant:
    database: "qdrant"
    embedder: "embedder"
    tools: ["search", "write_file"]
    document_stores:
      - name: "codebase"
        paths: ["./src/", "./lib/"]
        file_patterns: ["*.go", "*.py", "*.js", "*.ts"]
        chunk_size: 512
    
    prompt:
      system_role: |
        You are a code assistant. Use semantic search to find
        relevant code before answering questions or making changes.
```

### Documentation Assistant

```yaml
agents:
  docs_bot:
    database: "qdrant"
    embedder: "embedder"
    document_stores:
      - name: "documentation"
        paths: ["./docs/"]
        file_patterns: ["*.md", "*.rst"]
        chunk_size: 1024
    
    prompt:
      system_role: |
        Answer questions based on the documentation.
        Always cite your sources.
```

### Research Assistant

```yaml
agents:
  researcher:
    database: "qdrant"
    embedder: "embedder"
    document_stores:
      - name: "papers"
        paths: ["./research/"]
        file_patterns: ["*.pdf", "*.md"]
        chunk_size: 2048
      
      - name: "notes"
        paths: ["./notes/"]
        file_patterns: ["*.md"]
        chunk_size: 512
```

---

## Monitoring & Debugging

### Check Indexing Status

```bash
# View Qdrant dashboard
open http://localhost:6333/dashboard

# Check collection info
curl http://localhost:6333/collections/codebase
```

### Debug Search Results

```yaml
agents:
  debug:
    reasoning:
      show_debug_info: true
    document_stores:
      - name: "test"
        paths: ["./"]
        debug: true  # Log search results
```

### Re-index Documents

```bash
# Delete collection and re-index
curl -X DELETE http://localhost:6333/collections/codebase

# Restart Hector to trigger re-indexing
hector serve --config config.yaml
```

---

## Troubleshooting

### "Qdrant connection failed"

```bash
# Check if Qdrant is running
docker ps | grep qdrant

# Check logs
docker logs qdrant

# Verify port
curl http://localhost:6333/
```

### "Ollama not responding"

```bash
# Check if Ollama is running
ollama list

# Pull model if missing
ollama pull nomic-embed-text

# Check service
curl http://localhost:11434/api/tags
```

### "Search returns no results"

- Check documents are indexed: View Qdrant dashboard
- Verify file patterns match your files
- Lower `score_threshold` in search config
- Check chunk sizes aren't too large

---

## Best Practices

### 1. Choose the Right Chunk Size

```yaml
# Code: Small chunks for precision
chunk_size: 512

# Docs: Medium chunks for balance
chunk_size: 1024

# Narratives: Large chunks for context
chunk_size: 2048
```

### 2. Use Appropriate Overlap

```yaml
# Small chunks: 10-20% overlap
chunk_size: 256
chunk_overlap: 25

# Large chunks: 5-10% overlap
chunk_size: 2048
chunk_overlap: 200
```

### 3. Filter Irrelevant Files

```yaml
document_stores:
  - name: "clean_codebase"
    paths: ["./"]
    file_patterns: ["*.go"]
    exclude_patterns: [
      "*_test.go",     # Test files
      "*.min.js",      # Minified files
      "vendor/*",      # Dependencies
      "node_modules/*"
    ]
```

### 4. Organize by Type

```yaml
document_stores:
  - name: "source_code"
    paths: ["./src/"]
    chunk_size: 512
  
  - name: "documentation"
    paths: ["./docs/"]
    chunk_size: 1024
  
  - name: "configs"
    paths: ["./config/"]
    chunk_size: 256
```

---

## Next Steps

- **[How to Set Up RAG](../how-to/setup-rag.md)** - Complete setup guide
- **[Tools](tools.md)** - Using the search tool
- **[Memory](memory.md)** - Long-term memory with vectors
- **[Build a Coding Assistant](../how-to/build-coding-assistant.md)** - Full tutorial

---

## Related Topics

- **[Agent Overview](overview.md)** - Understanding agents
- **[Configuration Reference](../reference/configuration.md)** - All RAG options
- **[Architecture](../reference/architecture.md)** - How RAG works internally

