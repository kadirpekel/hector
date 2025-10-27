---
title: Set Up RAG & Semantic Search
description: Configure semantic code search for your agents in 20 minutes
---

# How to Set Up RAG & Semantic Search

Give your agents the ability to search through code and documents semantically—finding information by meaning, not just keywords.

**Time:** 20 minutes  
**Difficulty:** Intermediate

---

## What You'll Achieve

Agents that can:
- **Search code semantically** - "Find authentication logic" instead of grep for "auth"
- **Discover patterns** - Find similar code across the codebase
- **Understand context** - Retrieve relevant documentation automatically
- **Answer questions** - Query knowledge bases intelligently

---

## Prerequisites

✅ Hector installed ([Installation Guide](../getting-started/installation.md))  
✅ Docker (for Qdrant)  
✅ Basic understanding of [RAG concepts](../core-concepts/rag.md)

---

## Step 1: Start Qdrant (Vector Database)

Qdrant stores vector embeddings of your documents.

### Using Docker (Recommended)

```bash
docker run -d \
  --name qdrant \
  -p 6334:6334 \
  -p 6334:6334 \
  -v qdrant_data:/qdrant/storage \
  qdrant/qdrant
```

**Ports:**
- `6334` - gRPC API (used by Hector)
- `6334` - REST API + Dashboard

### Verify Installation

```bash
# Check if running
docker ps | grep qdrant

# Access dashboard
open http://localhost:6334/dashboard
```

You should see the Qdrant web interface.

---

## Step 2: Start Ollama (Embeddings)

Ollama generates vector embeddings from text.

### Install Ollama

=== "macOS/Linux"
    ```bash
    curl https://ollama.ai/install.sh | sh
    ```

=== "Windows"
    Download from [https://ollama.ai](https://ollama.ai)

### Pull Embedding Model

```bash
ollama pull nomic-embed-text
```

This downloads the embedding model (~274MB).

### Verify Installation

```bash
# List models
ollama list

# Should show:
# nomic-embed-text:latest

# Test embeddings
ollama run nomic-embed-text "test"
```

---

## Step 3: Configure Hector

Create `config-with-rag.yaml`:

```yaml
# Vector Database
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

# LLM
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"

# Document Stores (what to index)
document_stores:
  codebase:
    name: "codebase"
    paths: ["./src/", "./lib/"]

# Agent with Semantic Search
agents:
  coder:
    llm: "gpt-4o"
    database: "qdrant"
    embedder: "embedder"
    document_stores: ["codebase"]
    
    tools:
      - "search"  # Enable semantic search tool
```

**Key components:**

1. **database: "qdrant"** - Connect to vector database
2. **embedder: "embedder"** - Use Ollama for embeddings
3. **tools: ["search"]** - Enable search tool
4. **document_stores** - Define what to index

---

## Step 4: Start Hector and Index

```bash
export OPENAI_API_KEY="sk-..."
hector serve --config config-with-rag.yaml
```

**On first run, Hector automatically indexes your codebase:**

```
Hector server listening on :8080
Indexing document store: codebase
  Reading files from ./src/
  Found 156 files
  Creating 1,234 chunks
  Generating embeddings... 
  Storing in Qdrant...
Indexing complete: 1,234 chunks indexed
Agent registered: coder
```

This may take a few minutes for large codebases.

---

## Step 5: Test Semantic Search

### Interactive Chat

```bash
hector chat --config config-with-rag.yaml coder
```

Try these queries:

```
> How does authentication work in this codebase?
[Agent uses semantic search to find auth-related code]

> Where is the database connection configured?
[Agent finds db config files semantically]

> Show me examples of error handling
[Agent finds error handling patterns across codebase]
```

### Single Query

```bash
hector call --config config-with-rag.yaml coder "Explain how the API routes are structured"
```

Agent will:
1. Use semantic search to find routing code
2. Analyze the patterns
3. Provide explanation with examples

---

## Step 6: Verify It Works

### Check Qdrant Dashboard

Visit http://localhost:6334/dashboard

You should see:
- **Collection:** `codebase` (or your document store name)
- **Vectors:** Number of chunks indexed
- **Dimensions:** 768 (for nomic-embed-text)

### Test Search Directly

```bash
# Search via Qdrant API
curl -X POST http://localhost:6334/collections/codebase/points/search \
  -H "Content-Type: application/json" \
  -d '{
    "vector": [0.1, 0.2, ...],  # Would be actual embedding
    "limit": 5
  }'
```

---

## Customizing Your Setup

### Multiple Document Stores

Index different types of content with different settings:

```yaml
document_stores:
  # Source code - small chunks for precision
  source_code:
    name: "source_code"
    paths: ["./src/"]
    chunk_size: 512
  
  # Documentation - large chunks for context
  documentation:
    name: "documentation"
    paths: ["./docs/"]
    chunk_size: 2048
  
  # Configuration files - small chunks
  configs:
    name: "configs"
    paths: ["./config/"]
    chunk_size: 256
```

### Smart Exclusion Patterns

Hector automatically excludes common files you don't want indexed (dependencies, build artifacts, etc.).

**Default exclusions include:**
- VCS: `.git`, `.svn`, `.hg`, `.bzr`
- Python: `site-packages`, `dist-packages`, `venv`, `.venv`, `__pycache__`, `*.pyc`
- Node.js: `node_modules`, `.npm`, `.yarn`, `.pnp`
- Build: `dist`, `build`, `out`, `target`, `bin`, `obj`
- IDE: `.vscode`, `.idea`, `.DS_Store`
- Binary: `*.exe`, `*.dll`, `*.so`, `*.dylib`, `*.class`
- Media: `*.png`, `*.jpg`, `*.mp4`, `*.mp3`
- Archives: `*.zip`, `*.tar`, `*.gz`

**Extend defaults** (recommended - adds to built-in patterns):

```yaml
document_stores:
  codebase:
    name: "codebase"
    paths: ["./"]
    additional_exclude_patterns:
      - "**/legacy/**"
      - "**/deprecated/**"
      - "**/*.test.js"
```

**Override completely** (replaces all defaults):

```yaml
document_stores:
  custom:
    name: "custom"
    paths: ["./"]
    exclude_patterns:
      - "**/.git/**"
      - "**/node_modules/**"
      # Your complete exclusion list
```

### Progress Tracking

Monitor indexing progress with visual feedback:

```yaml
document_stores:
  codebase:
    name: "codebase"
    paths: ["./src/"]

    # Progress options (all optional, defaults shown)
    show_progress: true          # Show progress bar (default: true)
    verbose_progress: false      # Show current file (default: false)
    enable_checkpoints: true     # Enable resume on interrupt (default: true)
    quiet_mode: true             # Suppress per-file warnings (default: true)
```

**Example output:**

```
Indexing document store 'codebase' from: ./src/
📊 [████████████░░░░░░░░░░░░░░░] 42.5% | 156/367 files | 12.3 files/s | ETA: 17s
```

**Checkpoint recovery:**

If indexing is interrupted (Ctrl+C), Hector automatically saves progress and resumes:

```
🔄 Found checkpoint: 156/367 files processed (150 indexed, 6 skipped, 0 failed) - 12s elapsed
   Resuming from checkpoint...
📊 [████████████░░░░░░░░░░░░░░░] 42.5% | 156/367 files | 12.3 files/s | ETA: 17s
```

### Adjust Chunk Sizes

Balance between precision and context:

```yaml
# Precise but less context
chunk_size: 256
chunk_overlap: 25

# Balanced (recommended)
chunk_size: 512
chunk_overlap: 50

# More context but less precise
chunk_size: 2048
chunk_overlap: 200
```

### Performance Tuning

```yaml
document_stores:
  optimized:
    name: "optimized"
    paths: ["./src/"]
    
    # Indexing performance
    batch_size: 100        # Process 100 docs at a time
    parallel: true         # Parallel processing
    cache_embeddings: true # Cache for re-indexing
    
    # Search performance
    search_config:
      limit: 5             # Return top 5 results
      score_threshold: 0.7 # Minimum similarity score
```

---

## Re-Indexing

### Manual Re-Index

```bash
# Delete collection
curl -X DELETE http://localhost:6334/collections/codebase

# Restart Hector to trigger re-indexing
hector serve --config config-with-rag.yaml
```

### Auto Re-Index on Changes

**Coming soon:** File watcher for automatic re-indexing.

**Workaround:** Restart Hector after code changes:

```bash
# In development
while true; do
  hector serve --config config-with-rag.yaml
  sleep 5
done
```

---

## Advanced Configurations

### Qdrant Cloud

Use hosted Qdrant instead of local:

```yaml
databases:
  qdrant_cloud:
    type: "qdrant"
    host: "your-cluster.qdrant.io"
    port: 6334
    api_key: "${QDRANT_API_KEY}"
    use_https: true
```

### Different Embedding Models

```yaml
embedders:
  # Fast, smaller embeddings (384 dimensions)
  fast:
    type: "ollama"
    model: "all-minilm"
  
  # Better quality, larger embeddings (1024 dimensions)
  quality:
    type: "ollama"
    model: "mxbai-embed-large"
  
  # Best for code (768 dimensions, recommended)
  code:
    type: "ollama"
    model: "nomic-embed-text"

agents:
  coder:
    embedder: "code"  # Use code-optimized embeddings
```

### Multiple Collections

```yaml
agents:
  fullstack_dev:
    database: "qdrant"
    embedder: "embedder"
    document_stores: ["frontend", "backend", "docs"]

document_stores:
  frontend:
    name: "frontend"
    paths: ["./frontend/"]
    collection: "frontend_code"
  
  backend:
    name: "backend"
    paths: ["./backend/"]
    collection: "backend_code"
  
  docs:
    name: "docs"
    paths: ["./docs/"]
    collection: "documentation"
```

Each gets its own Qdrant collection.

---

## Troubleshooting

### "Qdrant connection failed"

**Check if running:**
```bash
docker ps | grep qdrant
```

**Check logs:**
```bash
docker logs qdrant
```

**Test connectivity:**
```bash
curl http://localhost:6334/
# Should return Qdrant info
```

**Fix:**
```bash
# Restart Qdrant
docker restart qdrant

# Or start if not running
docker start qdrant
```

### "Ollama not responding"

**Check if running:**
```bash
ollama list
```

**Test service:**
```bash
curl http://localhost:11434/api/tags
```

**Fix:**
```bash
# Restart Ollama service
# macOS/Linux:
sudo systemctl restart ollama

# Or reinstall
curl https://ollama.ai/install.sh | sh
```

### "Search returns no results"

**Verify indexing:**
- Check Qdrant dashboard: http://localhost:6334/dashboard
- Look for your collection
- Check vector count

**Lower threshold:**
```yaml
document_stores:
  codebase:
    name: "codebase"
    paths: ["./src/"]
    search_config:
      score_threshold: 0.5  # Lower from 0.7
```

**Check file patterns:**
```yaml
document_stores:
  codebase:
    name: "codebase"
    paths: ["./src/"]
```

### "Indexing is slow"

**Optimize batch size:**
```yaml
document_stores:
  codebase:
    name: "codebase"
    paths: ["./src/"]
    batch_size: 50  # Increase for better performance
    parallel: true
```

**Use smaller chunks:**
```yaml
chunk_size: 256  # Faster than 512 or 1024
```

---

## Production Considerations

### Persistent Storage

Mount Qdrant data directory:

```bash
docker run -d \
  --name qdrant \
  -p 6334:6334 \
  -v /path/to/qdrant_data:/qdrant/storage \
  qdrant/qdrant
```

### Resource Allocation

```bash
docker run -d \
  --name qdrant \
  -p 6334:6334 \
  --memory="2g" \
  --cpus="2" \
  -v qdrant_data:/qdrant/storage \
  qdrant/qdrant
```

### Backup Strategy

```bash
# Backup Qdrant data
docker exec qdrant tar czf /tmp/qdrant-backup.tar.gz /qdrant/storage
docker cp qdrant:/tmp/qdrant-backup.tar.gz ./backups/

# Restore
docker cp ./backups/qdrant-backup.tar.gz qdrant:/tmp/
docker exec qdrant tar xzf /tmp/qdrant-backup.tar.gz -C /
```

### Monitoring

```yaml
# Enable debug logging
logging:
  level: "info"
  format: "json"

agents:
  coder:
    reasoning:
      show_debug_info: true  # See search performance
```

---

## Verification Checklist

✅ Qdrant running and accessible  
✅ Ollama installed with nomic-embed-text  
✅ Hector configured with database and embedder  
✅ Document stores defined  
✅ Indexing completed successfully  
✅ Search tool enabled in agent  
✅ Agent can find relevant code semantically

---

## Next Steps

- **[Build a Coding Assistant](build-coding-assistant.md)** - Use RAG in practice
- **[RAG & Semantic Search](../core-concepts/rag.md)** - Understand the concepts
- **[Tools](../core-concepts/tools.md)** - Learn about the search tool
- **[Configuration Reference](../reference/configuration.md)** - All RAG options

---

## Related Topics

- **[Memory](../core-concepts/memory.md)** - Long-term memory also uses vectors
- **[Agent Overview](../core-concepts/overview.md)** - Understanding agents
- **[Architecture](../reference/architecture.md)** - How RAG works internally

