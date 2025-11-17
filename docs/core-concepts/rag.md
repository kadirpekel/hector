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
  -p 6334:6334 \
  -p 6334:6334 \
  -v $(pwd)/qdrant_storage:/qdrant/storage \
  qdrant/qdrant
```

Verify: http://localhost:6334/dashboard

### 2. Start Ollama

```bash
# Install Ollama (macOS/Linux)
curl https://ollama.ai/install.sh | sh

# Pull embedding model
ollama pull nomic-embed-text
```

### 3. Configure Hector

```yaml
# Vector store
vector_stores:
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

# Agent with semantic search
agents:
  coder:
    vector_store: "qdrant"
    embedder: "embedder"
    tools: ["search"]
    document_stores:
      - name: "codebase"
        source: "directory"
        path: "./src/"
```

### 4. Test It

```bash
hector call coder "How does authentication work in this codebase?"
```

The agent will semantically search your code and answer!

---

## Document Stores

Document stores define what gets indexed for search. Hector supports three source types:

- **Directory** - Index files from local filesystem
- **SQL** - Index data from SQL databases (PostgreSQL, MySQL, SQLite)
- **API** - Index data from REST API endpoints

### Document Parsing

Hector includes **native parsers** for common formats:
- ✅ **PDF** - Basic text extraction
- ✅ **DOCX** - Word document parsing
- ✅ **XLSX** - Excel spreadsheet parsing

For advanced parsing needs, you can use **MCP (Model Context Protocol) tools** (e.g., Docling) for:
- Better PDF parsing (layout detection, table extraction, OCR)
- Additional formats (PPTX, HTML, audio, images)
- Enhanced metadata extraction

### Basic Configuration (Directory Source)

```yaml
vector_stores:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334

agents:
  assistant:
    vector_store: "qdrant"  # Reference to vector_stores section
    embedder: "embedder"
    document_stores:
      - name: "docs"
        source: "directory"  # Optional: defaults to "directory"
        path: "./documentation/"
        # Note: Defaults to parseable file types (text + .pdf/.docx/.xlsx)
        # To restrict further: include_patterns: ["*.md", "*.txt"]
```

### SQL Database Source

Index content from SQL databases:

```yaml
document_stores:
  database_content:
    name: "database_content"
    source: "sql"
    sql:
      driver: "sqlite3"  # "postgres", "mysql", or "sqlite3"
      database: "./data/content.db"
    sql_tables:
      - table: "articles"
        columns: ["title", "content"]
        id_column: "id"
        updated_column: "updated_at"
        metadata_columns: ["author", "category"]
```

### REST API Source

Index content from REST API endpoints:

```yaml
document_stores:
  api_content:
    name: "api_content"
    source: "api"
    api:
      base_url: "https://api.example.com"
      auth:
        type: "bearer"
        token: "${API_TOKEN}"
      endpoints:
        - path: "/articles"
          id_field: "id"
          content_field: "title,content"
          metadata_fields: ["author", "published_at"]
```

### Multiple Document Stores

```yaml
agents:
  researcher:
    vector_store: "qdrant"
    embedder: "embedder"
    document_stores:
      - name: "codebase"
        source: "directory"
        path: "./src/"
        chunk_size: 512
      
      - name: "documentation"
        source: "directory"
        path: "./docs/"
        include_patterns: ["*.md"]
        chunk_size: 1024
      
      - name: "database_content"
        source: "sql"
        sql:
          driver: "sqlite3"
          database: "./data/content.db"
        sql_tables:
          - table: "articles"
            columns: ["title", "content"]
            id_column: "id"
        chunk_size: 800
      
      - name: "api_content"
        source: "api"
        api:
          base_url: "https://api.example.com"
          endpoints:
            - path: "/articles"
              id_field: "id"
              content_field: "title,content"
        chunk_size: 800
```

### Configuration Options

```yaml
document_stores:
  - name: "my_store"
    source: "directory"  # "directory", "sql", or "api"
    path: "./path1/"     # For directory source
    include_patterns: ["*.ext"]  # Optional: defaults to common text files + .pdf/.docx/.xlsx
    
    # Chunking
    chunk_size: 512           # Characters per chunk
    chunk_overlap: 50         # Overlap between chunks
    
    # Parsing
    parser: "native"          # native|custom|plugin
    
    # Indexing
    collection: "my_collection"  # Qdrant collection name
    batch_size: 100           # Documents per batch
    
    # Filtering (directory source only)
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

3. Vector database finds similar chunks (cosine similarity)
   ├─ auth.go chunk (similarity: 0.92)
   ├─ user.go chunk (similarity: 0.85)
   └─ db.go chunk (similarity: 0.78)

4. Top chunks injected into agent context

5. Agent answers using retrieved context
```

---

## Vector Databases

Hector supports multiple vector databases. Choose based on your needs:

| Database | Hybrid Search | Best For |
|----------|--------------|----------|
| **Qdrant** | ✅ (RRF) | Self-hosted, production-ready |
| **Pinecone** | ✅ (RRF) | Managed cloud service |
| **Weaviate** | ✅ (Native) | Native hybrid search, GraphQL API |
| **Milvus** | ✅ (RRF) | Large-scale, high-performance |
| **Chroma** | ✅ (RRF) | Simple, lightweight, embedded |

### Qdrant (Recommended for Self-Hosted)

```yaml
vector_stores:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334           # gRPC port (default: 6334)
    api_key: ""          # Optional for Qdrant Cloud
    enable_tls: false   # Enable for cloud
```

**Docker:**
```bash
docker run -d \
  --name qdrant \
  -p 6334:6334 \
  -v qdrant_data:/qdrant/storage \
  qdrant/qdrant
```

**Qdrant Cloud:**
```yaml
vector_stores:
  qdrant_cloud:
    type: "qdrant"
    host: "your-cluster.qdrant.io"
    port: 6334
    api_key: "${QDRANT_API_KEY}"
    enable_tls: true
```

### Pinecone (Managed Cloud)

```yaml
vector_stores:
  pinecone:
    type: "pinecone"
    api_key: "${PINECONE_API_KEY}"
    environment: "us-east-1"  # Your Pinecone environment
```

**Setup:**
1. Create account at [pinecone.io](https://pinecone.io)
2. Create an index
3. Get your API key and environment
4. Configure in Hector

### Weaviate (Native Hybrid Search)

```yaml
vector_stores:
  weaviate:
    type: "weaviate"
    host: "localhost"
    port: 8080           # Default Weaviate port
    api_key: ""          # Optional API key
    enable_tls: false
```

**Docker:**
```bash
docker run -d \
  --name weaviate \
  -p 8080:8080 \
  -e PERSISTENCE_DATA_PATH=/var/lib/weaviate \
  -v weaviate_data:/var/lib/weaviate \
  semitechnologies/weaviate:latest
```

**Features:**
- Native hybrid search support (no fallback needed)
- GraphQL API
- Built-in vectorization options

### Milvus (High-Performance)

```yaml
vector_stores:
  milvus:
    type: "milvus"
    host: "localhost"
    port: 19530          # Default Milvus port
    api_key: ""          # Optional
    enable_tls: false
```

**Docker (Standalone):**
```bash
docker run -d \
  --name milvus-standalone \
  -p 19530:19530 \
  -p 9091:9091 \
  milvusdb/milvus:latest
```

**Features:**
- Optimized for large-scale deployments
- High-performance vector search
- Supports distributed deployments

### Chroma (Lightweight)

```yaml
vector_stores:
  chroma:
    type: "chroma"
    host: "localhost"
    port: 8000           # Default Chroma port
    api_key: ""          # Optional
    enable_tls: false
```

**Docker:**
```bash
docker run -d \
  --name chroma \
  -p 8000:8000 \
  chromadb/chroma:latest
```

**Features:**
- Simple and lightweight
- Good for development and small deployments
- Easy to set up

### Custom Vector Databases (Plugins)

```yaml
plugins:
  databases:
    - name: "my-vector-db"
      protocol: "grpc"
      path: "/path/to/plugin"

vector_stores:
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

Configure how documents are searched and ranked:

```yaml
agents:
  searcher:
    vector_store: "qdrant"
    embedder: "embedder"
    document_stores:
      - name: "docs"
        paths: ["./"]
    search:
      top_k: 10              # Default number of results (default: 10)
      threshold: 0.5          # Minimum similarity score 0.0-1.0 (default: 0.5)
      preserve_case: true    # Don't lowercase queries (default: true)
      search_mode: "hybrid"  # "vector", "hybrid", "keyword", "multi_query", or "hyde"
      hybrid_alpha: 0.5      # Blending factor for hybrid search (0.0-1.0)
      
      # Optional: LLM-based re-ranking
      rerank:
        enabled: true
        llm: "gpt-4o-mini"
        max_results: 20
      
      # Optional: Multi-query expansion
      multi_query:
        enabled: false
        llm: "gpt-4o-mini"
        num_variations: 3
```

**Basic Parameters:**
- `top_k`: Maximum number of results to return (default: 10)
- `threshold`: Minimum cosine similarity score (0.0-1.0). Results below this are filtered out.
- `preserve_case`: Whether to preserve query case (useful for code search)

**Advanced Search Modes:**
- `search_mode: "vector"` - Pure semantic search (default, fastest)
- `search_mode: "hybrid"` - Combines keyword and vector search (better recall)
- `search_mode: "keyword"` - Keyword-focused search
- `search_mode: "multi_query"` - Expands query into multiple variations (improves recall)
- `search_mode: "hyde"` - Uses hypothetical document embeddings

**Re-ranking:**
- `rerank.enabled: true` - Enable LLM-based re-ranking for better result quality
- Requires an LLM provider configured
- Only reranks top N results (configurable via `max_results`)

See [Search Architecture](search-architecture.md) for complete documentation on all search features.

### MCP Document Parsing

Use MCP (Model Context Protocol) tools for advanced document parsing. This allows you to use services like Docling for better parsing quality or additional formats.

**Prerequisites:**
1. Configure MCP tools in your `tools` section
2. Start your MCP server (e.g., Docling MCP server)

**Basic Configuration:**

```yaml
tools:
  mcp_tools:
    - server:
        url: "http://localhost:3000"
        protocol: "mcp"
      tools:
        - "parse_document"

document_stores:
  knowledge_base:
    path: "./documents"
    mcp_parsers:
      tool_names: ["parse_document"]
```

**Advanced Configuration:**

```yaml
document_stores:
  research_papers:
    path: "./papers"
    mcp_parsers:
      tool_names: ["parse_document", "docling_parse"]  # Try tools in order
      extensions: [".pdf", ".pptx", ".html"]            # Only these formats
      priority: 10                                       # Override native parsers
      prefer_native: false                               # Use MCP first
```

**Configuration Options:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `tool_names` | `[]string` | Required | MCP tool names to try (in order) |
| `extensions` | `[]string` | `[]` (all) | File extensions to handle (empty = all binary files) |
| `priority` | `int` | `8` | Extractor priority (higher = preferred) |
| `prefer_native` | `bool` | `false` | Use native parsers first, MCP as fallback |

**Use Cases:**

1. **Override Native Parsers** (better quality):
```yaml
mcp_parsers:
  tool_names: ["parse_document"]
  priority: 10  # Higher than native (5)
```

2. **Use as Fallback** (when native fails):
```yaml
mcp_parsers:
  tool_names: ["parse_document"]
  prefer_native: true
  priority: 4  # Lower than native (5)
```

3. **Format-Specific** (unsupported formats):
```yaml
mcp_parsers:
  tool_names: ["parse_document"]
  extensions: [".pptx", ".html"]  # Formats not supported natively
```

**Benefits:**
- ✅ Better PDF parsing (layout, tables, OCR)
- ✅ Additional formats (PPTX, HTML, audio, images)
- ✅ Enhanced metadata extraction
- ✅ Works with any MCP service (not just Docling)

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
    include_patterns: ["*.pdf"]
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
          top_k: 3             # Fewer results = faster
          threshold: 0.85      # Default: 0.85, higher threshold = fewer candidates
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

# Vector store configuration
vector_stores:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334
```

---

## Use Cases

### Code Search

```yaml
agents:
  code_assistant:
    vector_store: "qdrant"
    embedder: "embedder"
    tools: ["search", "write_file"]
    search:
      search_mode: "hybrid"  # Better for code with specific terms
      hybrid_alpha: 0.6      # Favor vector but include keywords
      preserve_case: true    # Important for code identifiers
    document_stores:
      - name: "codebase"
        paths: ["./src/", "./lib/"]
        include_patterns: ["*.go", "*.py", "*.js", "*.ts"]
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
    vector_store: "qdrant"
    embedder: "embedder"
    document_stores:
      - name: "documentation"
        paths: ["./docs/"]
        include_patterns: ["*.md", "*.rst"]
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
    vector_store: "qdrant"
    embedder: "embedder"
    document_stores:
      - name: "papers"
        paths: ["./research/"]
        include_patterns: ["*.pdf", "*.md"]
        chunk_size: 2048
      
      - name: "notes"
        paths: ["./notes/"]
        include_patterns: ["*.md"]
        chunk_size: 512
```

---

## Monitoring & Debugging

### Check Indexing Status

```bash
# View Qdrant dashboard
open http://localhost:6334/dashboard

# Check collection info
curl http://localhost:6334/collections/codebase
```

### Debug Search Results

```yaml
agents:
  debug:
    reasoning:
    document_stores:
      - name: "test"
        paths: ["./"]
        debug: true  # Log search results
```

### Re-index Documents

```bash
# Delete collection and re-index
curl -X DELETE http://localhost:6334/collections/codebase

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
curl http://localhost:6334/
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
- Lower `threshold` in search config
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

Hector automatically excludes common files that shouldn't be indexed. You can also add custom exclusions.

#### Default Exclusions

Hector automatically skips:

**Version Control:**
```
**/.git/**, **/.svn/**, **/.hg/**, **/.bzr/**
```

**Dependencies:**
```
**/node_modules/**, **/vendor/**, **/venv/**, **/__pycache__/**
**/.npm/**, **/.yarn/**, **/gems/**, **/.bundle/**
```

**Build Artifacts:**
```
**/dist/**, **/build/**, **/target/**, **/.next/**, **/bin/**
**/.cache/**, **/.parcel-cache/**
```

**IDE & Editor:**
```
**/.vscode/**, **/.idea/**, **/.DS_Store
**/*.swp, **/*~
```

**Binary Files:**
```
*.exe, *.dll, *.so, *.pyc, *.o
*.png, *.jpg, *.mp4, *.mp3
*.zip, *.tar, *.gz
```

**Logs & Temp:**
```
*.log, *.tmp, *.cache
**/logs/**, **/tmp/**
```

**Lock Files:**
```
**/package-lock.json, **/yarn.lock
**/Gemfile.lock, **/Cargo.lock
```

**Empty Files:**
```
All files with 0 bytes are automatically skipped
```

See the full list: [112 default exclusions](https://github.com/kadirpekel/hector/blob/main/pkg/config/types.go#L978)

#### Custom Exclusions

Add your own patterns:

```yaml
document_stores:
  - name: "filtered_codebase"
    paths: ["./"]
    include_patterns: ["*.go", "*.py"]
    exclude_patterns: [
      # Custom exclusions (in addition to defaults)
      "*_test.go",        # Test files
      "**/*_mock.go",     # Mock files
      "**/testdata/**",   # Test data directories
      "*.generated.go",   # Generated code
      "**/experiments/**" # Experimental code
    ]
```

#### Pattern Syntax

Hector supports flexible glob patterns:

```yaml
# Directory patterns
"**/node_modules/**"    # Any node_modules directory
"**/.git/**"            # Any .git directory

# File extension patterns
"*.log"                 # All .log files
"*.pyc"                 # All .pyc files

# Specific files
"**/.DS_Store"          # .DS_Store anywhere
"**/package-lock.json"  # Lock files

# Combined patterns
"**/*.min.js"           # Minified JS anywhere
"**/dist/*.map"         # Source maps in dist
```

#### Override Defaults

To use ONLY your patterns (no defaults):

```yaml
document_stores:
  - name: "minimal"
    paths: ["./"]
    # Explicitly set empty to disable defaults
    exclude_patterns: []
    # Now only your include_patterns apply
    include_patterns: ["*.md"]
```

⚠️ **Warning:** Disabling default exclusions may index binary files, node_modules, etc.

#### Performance Tips

**DO:**
- ✅ Exclude large directories (`node_modules`, `vendor`)
- ✅ Exclude binary files (images, videos, archives)
- ✅ Exclude build artifacts (`dist`, `build`)
- ✅ Use specific patterns (`*.test.js` vs `**test**`)

**DON'T:**
- ❌ Index empty files (auto-skipped)
- ❌ Index minified files (`*.min.js`)
- ❌ Index compiled files (`*.pyc`, `*.o`)
- ❌ Use overly broad patterns (`**/*`)

#### Example: Production Setup

```yaml
document_stores:
  - name: "production_codebase"
    paths: ["./src/", "./lib/"]
    
    # Explicit inclusions
    include_patterns: [
      "*.go", "*.py", "*.js", "*.ts",
      "*.md", "*.yaml", "*.json"
    ]
    
    # Additional exclusions (on top of defaults)
    exclude_patterns: [
      # Project-specific
      "*_test.go",
      "**/*_mock.py",
      "**/fixtures/**",
      
      # Generated code
      "*.pb.go",
      "*.generated.*",
      
      # Docs we don't want
      "**/node_modules/**/@types/**",
      "**/vendor/github.com/**/testdata/**"
    ]
    
    # Skip large files
    max_file_size: 10485760  # 10MB
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

- **[Building Enterprise RAG Systems](../blog/posts/building-enterprise-rag-systems.md)** - Complete step-by-step guide
- **[Tools](tools.md)** - Using the search tool
- **[Memory](memory.md)** - Long-term memory with vectors
- **[Building a Coding Assistant](../blog/posts/building-a-coding-assistant.md)** - Full tutorial

---

## Related Topics

- **[Agent Overview](overview.md)** - Understanding agents
- **[Configuration Reference](../reference/configuration.md)** - All RAG options
- **[Architecture](../reference/architecture.md)** - How RAG works internally

