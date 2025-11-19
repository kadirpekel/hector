---
title: Building Enterprise RAG Systems with Hector
description: Learn how to build a production-grade RAG system step-by-step that works entirely on-premise, searches across multiple data sources, and provides enterprise observability—all configured in YAML.
date: 2025-11-16
authors:
  - name: Hector Team
tags:
  - Enterprise AI
  - RAG
  - On-Premise Deployment
  - Multi-Source Search
  - Production AI
  - Data Sovereignty
hide:
  - navigation
---

# Building Enterprise RAG Systems with Hector: A Step-by-Step Guide

Your enterprise has knowledge scattered across databases, APIs, and file systems. How do you build a RAG system that works entirely on-premise and searches across all these sources?

In this guide, we'll build a production-ready RAG system **step by step** using Hector. You'll start simple with one data source, then learn how to add more.

**Time:** 30-45 minutes  
**Difficulty:** Intermediate  
**Perfect for:** Platform engineers, SREs, DevOps teams

---

## ⚡ Quick Start (2 Minutes)

Want to see it working immediately? Here's the fastest path using **zero-config mode** (no config file needed!):

```bash
# 1. Start services
docker run -d --name qdrant -p 6334:6334 qdrant/qdrant
docker run -d --name ollama -p 11434:11434 ollama/ollama
sleep 30 && docker exec ollama ollama pull nomic-embed-text qwen3

# 2. Create sample file
mkdir -p docs/internal
cat > docs/internal/policy.md << 'EOF'
# Security Policy

## Password Requirements

- Minimum length: 12 characters
- Must include: uppercase, lowercase, numbers, special characters
- Cannot reuse last 5 passwords
- Must change every 90 days
- Password managers are mandatory
- Multi-factor authentication required
EOF

# 3. Test it with zero-config! (no config file needed)
hector call "What are password requirements?" \
  --docs-folder ./docs/internal \
  --provider ollama \
  --role "You are an enterprise knowledge assistant. Always cite your sources when answering questions."
```

**Note:** Defaults used automatically: `--model qwen3`, `--embedder-model nomic-embed-text`, `--vectordb http://localhost:6334`

**🎉 Done!** You have a working RAG system. Hector automatically:

- ✅ Created a document store from your folder
- ✅ Enabled file watching (auto-reindex on changes)
- ✅ Set up vector database and embeddings
- ✅ Created the search tool

**Want more control?** See Step 1.3 below for full YAML configuration.

---

## What You'll Build

By the end of this guide, you'll have a RAG system that searches across multiple data sources (directory, SQL, API) in parallel, runs entirely on-premise, supports advanced search modes (hybrid, multi-query, HyDE), and provides enterprise observability—all configured in YAML with zero code.

---

## Prerequisites

You'll need:

- ✅ **Docker** installed (for Qdrant and Ollama)
- ✅ **Hector** installed ([Installation Guide](../../getting-started/installation.md))
- ✅ **4GB+ free disk space** (for models)
- ✅ **Basic understanding** of YAML

That's it! No coding required.

---

## Step 1: Start with One Data Source (Directory)

Let's start simple with a directory source—it requires no database setup and lets you see results immediately.

### 1.1 Set Up Infrastructure

First, start the services you'll need:

```bash
# Start Qdrant (vector database)
docker run -d --name qdrant -p 6334:6334 -v qdrant-data:/qdrant/storage qdrant/qdrant

# Start Ollama (for embeddings and LLM)
docker run -d --name ollama -p 11434:11434 -v ollama-data:/root/.ollama ollama/ollama

# Wait 30 seconds for Ollama to start, then download models
docker exec ollama ollama pull nomic-embed-text  # Embedding model
docker exec ollama ollama pull qwen3              # LLM model
```

### 1.2 Create Sample Documentation

Create a directory with some sample files:

```bash
mkdir -p docs/internal
cat > docs/internal/security-policy.md << 'EOF'
# Security Policy

## Password Requirements

- Minimum length: 12 characters
- Must include: uppercase, lowercase, numbers, special characters
- Cannot reuse last 5 passwords
- Must change every 90 days
- Password managers are mandatory
- Multi-factor authentication required
EOF
```

### 1.3 Create Your First Configuration

Create `config.yaml`:

```yaml
# Vector Database
vector_stores:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334

# Embedder (creates vector embeddings)
embedders:
  embedder:
    type: "ollama"
    host: "http://localhost:11434"
    model: "nomic-embed-text"

# LLM Provider (100% local, no external APIs)
llms:
  local-llm:
    type: "ollama"
    host: "http://localhost:11434"
    model: "qwen3"
    timeout: 300

# Document Store - Directory Source
document_stores:
  internal_docs:
    source: "directory"
    path: "./docs/internal/"
    include_patterns: ["*.md", "*.txt"]
    chunk_size: 1024
    chunk_overlap: 100
    enable_watch_changes: true    # Auto-reindex when files change (default: true)
    enable_checkpoints: true      # Resume indexing if interrupted (default: true)

# Agent Configuration
agents:
  assistant:
    llm: "local-llm"
    vector_store: "qdrant"
    embedder: "embedder"
    document_stores: ["internal_docs"]
    tools:
      - "search"
      - "evaluate_rag"  # Enable evaluation tool
    prompt:
      system_prompt: |
        You are an enterprise knowledge assistant.
        Always cite your sources when answering questions.
    search:
      top_k: 10
      threshold: 0.5
      preserve_case: true
      
      # Advanced search: Hybrid mode (keyword + vector)
      search_mode: "hybrid"
      hybrid_alpha: 0.6  # 60% vector, 40% keyword
      
      # Optional: Enable re-ranking for better results
      # rerank:
      #   enabled: true
      #   llm: "reranker"
      #   max_results: 20
```

### 1.4 Test It (Local Mode)

You can test your RAG system **without starting a server** using `hector call`:

**Option A: With Config File (Full Control)**

```bash
hector call "What are our password requirements?" \
  --agent assistant \
  --config config.yaml
```

**Expected output (both options produce similar results):**
```
The password requirements according to the security policy are as follows:

- Minimum length: 12 characters
- Must include: uppercase, lowercase, numbers, and special characters
- Cannot reuse: last 5 passwords
- Must change: every 90 days
- Password managers: are mandatory
- Multi-factor authentication: is required
```

*Note: Exact formatting may vary slightly, but all the key information will be present.*

**Option B: Zero-Config (No Config File Needed!)**

For quick testing, you can skip the config file entirely:

```bash
hector call "What are our password requirements?" \
  --docs-folder ./docs/internal \
  --provider ollama \
  --role "You are an enterprise knowledge assistant. Always cite your sources when answering questions."
```

**Note:** Defaults used automatically: `--model qwen3`, `--embedder-model nomic-embed-text`, `--vectordb http://localhost:6334`

Hector automatically creates everything you need! This is perfect for ad-hoc RAG queries. The `--role` flag sets the system prompt (same as `system_prompt` in YAML config). You only need to specify what's different from the defaults.

**🎉 Congratulations!** You've built your first RAG system! Hector indexed your files, created vector embeddings, and can now find relevant content by meaning. Now let's add more sources.

---

## Step 2: Add a Second Source (SQL Database)

Now let's add a SQL database source to search structured data alongside files.

### 2.1 Set Up PostgreSQL

```bash
docker run -d \
  --name postgres \
  -e POSTGRES_DB=knowledge_base \
  -e POSTGRES_USER=hector \
  -e POSTGRES_PASSWORD=secure_password \
  -p 5433:5432 \
  -v postgres-data:/var/lib/postgresql/data \
  postgres:15-alpine
```

### 2.2 Create Sample Data

```bash
# Connect to database
docker exec -it postgres psql -U hector -d knowledge_base

# Create table and insert data
CREATE TABLE knowledge_articles (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255),
    content TEXT,
    category VARCHAR(100),
    status VARCHAR(20) DEFAULT 'published',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO knowledge_articles (title, content, category) VALUES
('Deployment Process', 'Our deployment follows these steps: 1. Code review 2. Automated testing 3. Staging deployment 4. Production deployment', 'Operations'),
('Architecture Overview', 'Our system uses microservices architecture with API gateway pattern', 'Engineering');
```

### 2.3 Update Configuration

Add the SQL source to your `config.yaml`:

```yaml
# ... existing config ...

# SQL Database Connection
databases:
  knowledge_db:
    driver: "postgres"
    host: "localhost"
    port: 5433
    database: "knowledge_base"
    username: "hector"
    password: "secure_password"
    ssl_mode: "disable"

# Add SQL Document Store
document_stores:
  internal_docs:
    # ... existing directory config ...
  
  knowledge_base:  # NEW: SQL source
    source: "sql"
    sql:
      driver: "postgres"
      host: "localhost"
      port: 5433
      database: "knowledge_base"
      username: "hector"
      password: "secure_password"
      ssl_mode: "disable"
    sql_tables:
      - table: "knowledge_articles"
        columns: ["title", "content"]
        id_column: "id"
        updated_column: "updated_at"
        where_clause: "status = 'published'"
        metadata_columns: ["category"]
    chunk_size: 800
    chunk_overlap: 50

# Update agent to search both sources
agents:
  assistant:
    document_stores: ["internal_docs", "knowledge_base"]  # Both sources!
    # ... rest of config ...
```

### 2.4 Test Multi-Source Search

```bash
hector call "What is our deployment process?" \
  --agent assistant \
  --config config.yaml
```

Now the agent searches **both** your directory files and the database in parallel for faster results.

---

## Step 3: Add a Third Source (REST API)

Let's add an API source to index dynamic content from internal APIs.

### 3.1 Set Up a Mock API

For this example, we'll use a simple mock API. In production, this would be your actual internal wiki or documentation API.

**Option 1: Simple HTTP Server (Python)**

```bash
# Create a simple Python API
cat > wiki-api/server.py << 'EOF'
from http.server import HTTPServer, BaseHTTPRequestHandler
import json

class APIHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/api/pages':
            pages = [
                {"id": 1, "title": "System Architecture", "content": "Our architecture uses microservices...", "category": "Engineering"},
                {"id": 2, "title": "Monitoring Setup", "content": "We use Prometheus for metrics...", "category": "Operations"},
            ]
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps(pages).encode())
        else:
            self.send_response(404)
            self.end_headers()

if __name__ == '__main__':
    server = HTTPServer(('localhost', 8080), APIHandler)
    server.serve_forever()
EOF

# Run it
python3 wiki-api/server.py &
```

**Option 2: Use the Complete Example**

If you prefer, you can use the complete example which includes a pre-built API:

```bash
cd examples/enterprise-rag
./setup-docker.sh
```

This includes a working wiki API already configured.

### 3.2 Add API Source to Configuration

```yaml
document_stores:
  # ... existing sources ...
  
  wiki_content:  # NEW: API source
    source: "api"
    api:
      base_url: "http://localhost:8080"
      endpoints:
        - path: "/api/pages"
          method: "GET"
          id_field: "id"
          content_field: "title,content"
          metadata_fields: ["category"]
    chunk_size: 800
    chunk_overlap: 50

agents:
  assistant:
    document_stores: ["internal_docs", "knowledge_base", "wiki_content"]  # All three!
```

### 3.3 Test All Three Sources

```bash
hector call "How does our system architecture work?" \
  --agent assistant \
  --config config.yaml
```

The agent now searches **all three sources in parallel** - directory, database, and API! Hector searches all sources simultaneously, so total time equals the slowest source (not the sum), making it 3x faster.

---

## Step 4: Deploy as a Server with Web UI

Deploy as a server to get a web UI, API access, persistent sessions, and multi-user support.

### 4.1 Start the Server

```bash
# Basic server
hector serve --config config.yaml

# With file watching (auto-reload on config changes)
hector serve --config config.yaml --config-watch
```

This starts:

- **A2A API** on port 8080 (for programmatic access)
- **Web UI** on port 8080 (like ChatGPT interface)
- **File watching** (if `--config-watch` is used, reloads config automatically)

### 4.2 Access the Web UI

Open your browser to:
```
http://localhost:8080
```

You'll see a web interface where you can:

- Select any agent from the dropdown
- Chat with the agent in real-time
- See the agent's reasoning process
- View tool calls and results

### 4.3 Use the API

You can also call the agent programmatically:

```bash
# Local call (direct mode)
hector call "What are our password requirements?" \
  --agent assistant \
  --config config.yaml

# Remote call (via server)
hector call "What are our password requirements?" \
  --agent assistant \
  --url http://localhost:8080
```

**Note:** When using `--url`, you're calling a remote agent. When using `--config`, you're calling locally (no server needed).

### 4.4 Automatic File Watching and Checkpoints

Hector includes two powerful features for directory sources:

**File Watching (`enable_watch_changes`):**
- **Automatic re-indexing**: When files are added, modified, or deleted in your directory
- **Incremental updates**: Only changed files are re-indexed (fast!)
- **No restart needed**: Changes are picked up automatically while the server is running

**Checkpoint Recovery (`enable_checkpoints`):**
- **Resume on interruption**: If indexing is interrupted (crash, network issue), Hector can resume
- **Checkpoints saved**: Progress saved every 10 seconds during indexing
- **Automatic recovery**: On restart, resumes from the last checkpoint

**Both are enabled by default**, but you can configure them:

```yaml
document_stores:
  internal_docs:
    source: "directory"
    path: "./docs/internal/"
    enable_watch_changes: true    # Auto-reindex on changes (default: true)
    enable_checkpoints: true      # Resume on interruption (default: true)
```

**Try it:**
1. Start Hector: `hector serve --config config.yaml`
2. Add a new file to `./docs/internal/`
3. Watch the logs - you'll see it automatically indexed!

---

## Step 5: Add Enterprise Features

Let's add production-ready features including advanced search capabilities.

### 5.1 Enable Advanced Search Modes

Hector supports multiple search strategies optimized for different use cases:

**Hybrid Search (Recommended for Enterprise)**
Combines keyword and semantic search for better recall:

```yaml
agents:
  assistant:
    search:
      search_mode: "hybrid"
      hybrid_alpha: 0.6  # 60% vector, 40% keyword
      rerank:
        enabled: true
        llm: "gpt-4o-mini"
        max_results: 20
```

**Multi-Query Expansion**
Generates multiple query variations to improve recall:

```yaml
agents:
  assistant:
    search:
      search_mode: "multi_query"
      multi_query:
        enabled: true
        llm: "gpt-4o-mini"
        num_variations: 3  # Generate 3 query variations
```

**HyDE (Hypothetical Document Embeddings)**
Searches using an ideal answer document:

```yaml
agents:
  assistant:
    search:
      search_mode: "hyde"
      hyde:
        enabled: true
        llm: "gpt-4o-mini"
```

**Complete Advanced Configuration:**

```yaml
agents:
  assistant:
    search:
      top_k: 10
      threshold: 0.5
      preserve_case: true
      
      # Hybrid search (best for enterprise documentation)
      search_mode: "hybrid"
      hybrid_alpha: 0.6
      
      # LLM-based re-ranking for better results
      rerank:
        enabled: true
        llm: "gpt-4o-mini"
        max_results: 20
```

### 5.2 Choose Your Vector Database

Hector supports multiple vector databases. Choose based on your needs:

**Qdrant (Default - Self-Hosted)**
```yaml
vector_stores:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334
```

**Weaviate (Native Hybrid Search)**
```yaml
vector_stores:
  weaviate:
    type: "weaviate"
    host: "localhost"
    port: 8080
```

**Milvus (High-Performance)**
```yaml
vector_stores:
  milvus:
    type: "milvus"
    host: "localhost"
    port: 19530
```

**Chroma (Lightweight)**
```yaml
vector_stores:
  chroma:
    type: "chroma"
    host: "localhost"
    port: 8000
```

### 5.3 Enable RAG Evaluation

Add the evaluation tool to measure search quality:

```yaml
agents:
  assistant:
    tools:
      - "search"
      - "evaluate_rag"  # Enable evaluation tool
```

The agent can now evaluate its own search results:

```bash
hector call "Evaluate the search results for: What are our password requirements?" \
  --agent assistant \
  --config config.yaml
```

### 5.4 Enable Observability

Add to your `config.yaml`:

```yaml
global:
  observability:
    enable_metrics: true
    tracing:
      enabled: true
      endpoint_url: "localhost:4317"
```

**What you get:**
- **Prometheus metrics**: `http://localhost:8080/metrics`
- **Health checks**: `http://localhost:8080/health`
- **OpenTelemetry traces**: Distributed tracing support

### 5.5 View Metrics

```bash
curl http://localhost:8080/metrics
```

You'll see metrics like:

- Query latency
- Token usage
- Error rates
- Request counts
- Search performance (hybrid vs vector)

---

## Understanding Hector's RAG Foundation

Hector's RAG system uses a unified search engine that handles multiple data sources seamlessly. Queries can come through two paths: explicit search tool calls or automatic context injection. Both paths converge on the same parallel search engine, which searches all document stores simultaneously, aggregates results, and ranks them by relevance. Each result includes its source for transparency.

---

## Understanding Parallel Search

Hector automatically searches all your sources **in parallel**, not sequentially. With 3 sources, all are searched simultaneously, so total time equals the slowest source (not the sum), making it 3x faster. You can verify this with `--log-level debug` flag to see parallel execution in the logs.

---

## Conclusion

You've built a complete enterprise RAG system that works 100% on-premise, searches multiple data sources in parallel, supports advanced search modes, and provides enterprise observability—all configured in YAML with zero code.

**Next steps for your enterprise:**
- Connect to your actual databases, APIs, and file systems
- Enable security (JWT authentication, RBAC)
- Deploy to Kubernetes for high availability
- Set up Grafana dashboards for Prometheus metrics

**Resources:**
- [RAG Architecture Guide](../../reference/architecture/search-architecture.md) - Deep dive
- [Configuration Reference](../../reference/configuration.md) - All options
- [Web UI Guide](../../core-concepts/ag-ui.md) - Web interface features

Ready to build your own? Start with one source, then add more. For a complete production deployment example, see `examples/enterprise-rag`.

---

**About Hector**: Hector is a production-grade A2A-native agent platform designed for enterprise deployments. Learn more at [gohector.dev](https://gohector.dev).
