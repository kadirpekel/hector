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

By the end of this guide, you'll have a RAG system that:

- ✅ Searches across multiple data sources (directory, SQL database, API)
- ✅ Works 100% on-premise (no external APIs)
- ✅ Searches all sources in parallel (fast!)
- ✅ Provides enterprise observability (metrics, health checks)
- ✅ Requires zero code (pure YAML configuration)

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

Let's start simple. We'll build a RAG system that searches through files in a directory.

> **Why start with directory?** It's the simplest source type, requires no database setup, and lets you see results immediately. Once you understand how it works, adding SQL and API sources is straightforward.

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
    prompt:
      system_prompt: |
        You are an enterprise knowledge assistant.
        Always cite your sources when answering questions.
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

**🎉 Congratulations!** You've built your first RAG system! It's searching through your files and answering questions.

> **What just happened?** Hector indexed your files, created vector embeddings, stored them in Qdrant, and can now find relevant content by meaning (not just keywords). This is the foundation - now let's add more sources.

---

## Step 2: Add a Second Source (SQL Database)

> **Why add SQL?** Most enterprises store knowledge in databases (knowledge bases, wikis, CMS). Adding SQL support lets you search structured data alongside files.

Now let's add a second data source - a SQL database with knowledge articles.

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

Now the agent searches **both** your directory files and the database! Hector automatically searches them in parallel for faster results.

> **Key insight:** Notice you didn't need to change your query - the agent automatically searches both sources. This is the power of unified multi-source RAG.

---

## Step 3: Add a Third Source (REST API)

> **Why add API?** Many enterprises have internal APIs (wikis, ticketing systems, documentation platforms). API sources let you index dynamic content that changes frequently.

Let's add one more source - an internal wiki API.

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

The agent now searches **all three sources in parallel** - directory, database, and API! 🚀

> **What's happening under the hood?** Hector groups your sources by collections, launches parallel searches across all of them, then aggregates and ranks the results. Total search time ≈ time of the slowest source (not the sum). This is why parallel search is 3x faster!

---

## Step 4: Deploy as a Server with Web UI

> **Why deploy as a server?** While `hector call` is great for testing, deploying as a server gives you:
> - **Web UI**: ChatGPT-like interface for your team
> - **API access**: Programmatic access for integrations
> - **Persistent sessions**: Conversation history across requests
> - **Multi-user**: Multiple people can use it simultaneously

So far, we've been using `hector call` for local testing. Now let's deploy it as a server with a web UI.

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

Let's add production-ready features.

### 5.1 Enable Observability

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

### 5.2 View Metrics

```bash
curl http://localhost:8080/metrics
```

You'll see metrics like:

- Query latency
- Token usage
- Error rates
- Request counts

---

## Understanding Hector's RAG Foundation

Hector's RAG system is built on a unified search foundation that handles multiple data sources seamlessly. Here's how it works:

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    User Query                               │
│            "What are our password requirements?"            │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Agent Reasoning                          │
│  ┌────────────────────┐  ┌────────────────────┐             │
│  │  Path 1: Search    │  │  Path 2: Include   │             │
│  │  Tool (Explicit)  │  │  Context (Auto)    │              │
│  └────────┬──────────┘  └────────┬───────────┘              │
│           │                      │                          │
│           └──────────┬───────────┘                          │
│                      ▼                                      │
│         ┌───────────────────────┐                           │
│         │  Parallel Search      │                           │
│         │  (All Sources)         │                          │
│         └───────────┬───────────┘                           │
└─────────────────────┼───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│              Document Stores (Data Sources)                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │  Directory   │  │    SQL       │  │     API      │       │
│  │  (Files)     │  │  (Database)  │  │  (REST API)  │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
│         │                 │                 │               │
│         └─────────┬───────┴─────────┬───────┘               │
│                  ▼                  ▼                       │
│         ┌──────────────────────────────────┐                │
│         │    SearchEngine (Unified)        │                │
│         │  - Query Processing              │                │
│         │  - Embedding Generation          │                │
│         │  - Vector Search                 │                │
│         │  - Result Ranking                │                │
│         └──────────────┬───────────────────┘                │
│                        │                                    │
│                        ▼                                    │
│         ┌──────────────────────────────────┐                │
│         │      Vector Database             │                │
│         │      (Qdrant)                    │                │
│         └──────────────────────────────────┘                │
└─────────────────────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Ranked Results                           │
│  - Aggregated from all sources                              │
│  - Sorted by relevance score                                │
│  - Limited to top K results                                 │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    LLM Response                             │
│  (With cited sources from document stores)                  │
└─────────────────────────────────────────────────────────────┘
```

### Key Features

1. **Unified Search Engine**: Both search paths (tool call and context injection) converge on the same `SearchEngine`, ensuring consistent behavior
2. **Parallel Execution**: All document stores are searched simultaneously, not sequentially
3. **Automatic Aggregation**: Results from all sources are merged, deduplicated, and ranked by relevance
4. **Source Transparency**: Each result includes its source (which document store it came from)

### The Two Search Paths

**Path 1: Search Tool (Explicit)**
- LLM explicitly calls the `search` tool when it needs information
- Full control over what to search and when
- Returns structured JSON with results

**Path 2: Include Context (Automatic)**
- Automatically injects relevant context into the prompt
- Happens before LLM reasoning begins
- Seamless - LLM doesn't need to call tools

Both paths use the same parallel search foundation underneath!

---

## Understanding Parallel Search

Hector automatically searches all your sources **in parallel**, not sequentially. This means:

- **3 sources** = searches all 3 at the same time
- **Total time** ≈ time of the slowest source (not sum of all)
- **3x faster** than searching one after another

**Visual Example:**

```
Sequential Search (slow):
┌─────────┐    ┌─────────┐    ┌─────────┐
│ Source1 │───▶│ Source2 │───▶│ Source3 │  Total: 179+244+387 = 810ms
└─────────┘    └─────────┘    └─────────┘

Parallel Search (fast):
┌─────────┐
│ Source1 │───▶ 179ms
└─────────┘
┌─────────┐
│ Source2 │───▶ 244ms      Total: max(179,244,387) = 388ms ✅
└─────────┘
┌─────────┐
│ Source3 │───▶ 387ms
└─────────┘
```

You can verify this with debug logging:

```bash
hector call "What are our password requirements?" \
  --agent assistant \
  --config config.yaml \
  --debug
```

Look for logs showing parallel execution:
```
DEBUG Launching parallel searches targets=[internal_docs knowledge_base wiki_content] count=3
DEBUG Collection search completed collection=knowledge_base results=3 duration=179ms
DEBUG Collection search completed collection=internal_docs results=2 duration=244ms
DEBUG Parallel search completed total_duration=388ms
```

Notice: Total time (388ms) ≈ max(179ms, 244ms) = parallel execution! 🎯

---

## Troubleshooting

### Common Issues and Solutions

**Issue: "Connection refused" when starting services**

```bash
# Check if ports are already in use
lsof -i :6334  # Qdrant
lsof -i :11434 # Ollama
lsof -i :5433  # PostgreSQL

# Stop conflicting services or use different ports
docker run -d --name qdrant -p 6335:6334 qdrant/qdrant  # Use 6335 instead
```

**Issue: "Model not found" in Ollama**

```bash
# List available models
docker exec ollama ollama list

# Pull missing models
docker exec ollama ollama pull nomic-embed-text
docker exec ollama ollama pull qwen3
```

**Issue: "No results found" when searching**

- **Check indexing**: Look for "Indexing Complete" in logs
- **Verify file paths**: Ensure `path` in config matches actual directory
- **Check file patterns**: Verify `include_patterns` match your file extensions
- **Test with debug**: Use `--debug` flag to see search execution

**Issue: "PostgreSQL connection failed"**

```bash
# Verify PostgreSQL is running
docker ps | grep postgres

# Check connection
docker exec postgres pg_isready -U hector

# Verify credentials match config
docker exec -it postgres psql -U hector -d knowledge_base
```

**Issue: "File watching not working"**

- **Check permissions**: Ensure Hector can read the directory
- **Verify source type**: File watching only works for `source: "directory"`
- **Check logs**: Look for file watching errors in debug output

**Issue: Slow indexing**

- **Reduce chunk size**: Try `chunk_size: 512` for faster processing
- **Limit files**: Use `include_patterns` to exclude unnecessary files
- **Check Ollama**: Ensure embedding model is loaded and responsive

---

## Production Deployment

### Option 1: Docker Compose

Create `docker-compose.yaml`:

```yaml
version: '3.8'

services:
  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6334:6334"
    volumes:
      - qdrant-data:/qdrant/storage

  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: knowledge_base
      POSTGRES_USER: hector
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data

  hector:
    image: kadirpekel/hector:1.8.1
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/etc/hector/config.yaml:ro
      - ./docs:/docs:ro
    depends_on:
      - qdrant
      - ollama
      - postgres

volumes:
  qdrant-data:
  ollama-data:
  postgres-data:
```

Deploy:
```bash
docker-compose up -d
```

### Option 2: Use the Complete Example

We have a complete, ready-to-use example:

```bash
cd examples/enterprise-rag
./setup-docker.sh
```

This includes everything pre-configured and ready to go.

---

## Key Takeaways

What you've learned:

1. ✅ **Start simple**: Begin with one data source (directory), get it working, then expand
2. ✅ **Add sources incrementally**: SQL database, then API - each step builds on the previous
3. ✅ **Parallel search**: Hector automatically searches all sources concurrently (3x faster!)
4. ✅ **Local testing**: Use `hector call` with `--config` (no server needed) for quick testing
5. ✅ **Server deployment**: Use `hector serve` for web UI and API access
6. ✅ **Remote calls**: Use `--url` flag when calling remote agents
7. ✅ **File watching**: Automatic re-indexing when files change (`enable_watch_changes`)
8. ✅ **Checkpoint recovery**: Resume indexing if interrupted (`enable_checkpoints`)
9. ✅ **Config watching**: Use `--config-watch` to auto-reload configuration changes
10. ✅ **Zero code**: Everything configured in YAML - no programming required

**The Journey:**
- **Step 1**: One source working ✅
- **Step 2**: Two sources, parallel search ✅
- **Step 3**: Three sources, full multi-source RAG ✅
- **Step 4**: Production deployment with web UI ✅
- **Step 5**: Enterprise features (observability, monitoring) ✅

---

## Next Steps

**Enhance your system:**

- **Add more sources**: Connect to your actual databases and APIs
- **Enable security**: Add JWT authentication, RBAC
- **Scale up**: Deploy to Kubernetes for high availability
- **Monitor**: Set up Grafana dashboards for Prometheus metrics

**Resources:**

- [RAG Architecture Guide](../../reference/architecture/search-architecture.md) - Deep dive
- [Configuration Reference](../../reference/configuration.md) - All options
- [Web UI Guide](../../core-concepts/ag-ui.md) - Web interface features

---

## Conclusion

You've built a complete enterprise RAG system that:

- ✅ Works 100% on-premise (no external APIs)
- ✅ Searches multiple data sources in parallel
- ✅ Provides enterprise observability
- ✅ Requires zero code (pure YAML)

**The best part?** You can test it locally with `hector call`, then deploy it as a server with a web UI using `hector serve`.

### What's Next?

**For Your Enterprise:**

1. **Replace sample data**: Connect to your actual databases, APIs, and file systems
2. **Add security**: Enable JWT authentication, RBAC, and access controls
3. **Scale up**: Deploy to Kubernetes with multiple replicas
4. **Monitor**: Set up Grafana dashboards for Prometheus metrics
5. **Iterate**: Use `--config-watch` for rapid configuration updates

**Ready to build your own?** Start with one source, then add more as you go. The complete example in `examples/enterprise-rag` shows you everything working together.

---

**About Hector**: Hector is a production-grade A2A-native agent platform designed for enterprise deployments. Learn more at [gohector.dev](https://gohector.dev).
