---
title: Using Docling with Hector for Advanced Document Parsing
description: Integrate Docling's powerful document parsing capabilities with Hector using Docker or local setup
date: 2025-01-16
tags:
  - RAG
  - Document Parsing
  - MCP
  - Docling
  - Tutorial
---

# Using Docling with Hector for Advanced Document Parsing

Enhance your RAG system with Docling's advanced document parsing capabilities. Parse PDFs, Word documents, PowerPoint presentations, and more with enterprise-grade accuracy.

**Time:** 10-15 minutes
**Difficulty:** Beginner

---

## What You'll Learn

- Understand Hector's MCP document parsing feature
- Set up Docling using Docker (recommended) or locally
- Configure Hector to use Docling for document parsing
- Parse complex documents (PDFs, DOCX, PPTX, XLSX, HTML)
- Use path mapping for containerized deployments

---

## Hector's MCP Document Parsing

Hector's document stores support **MCP-based document parsing**, allowing you to use any MCP-compliant service to parse documents during indexing. This is configured via the `mcp_parsers` option in your document store configuration.

**Key benefits:**
- **Pluggable architecture** - Use any MCP service that can parse documents
- **Format flexibility** - Support formats beyond Hector's native parsers
- **Quality improvements** - Better parsing for complex layouts, tables, OCR
- **Fallback chains** - Configure multiple parsers with priority ordering

**Common use cases:**
- **Docling** - Advanced PDF/DOCX/PPTX parsing with layout detection
- **Custom parsers** - Domain-specific document processing
- **OCR services** - Scanned document text extraction
- **Audio/Video** - Transcription services via MCP

This tutorial uses **Docling** as an example, but the same pattern applies to any MCP-based parser.

---

## Why Docling?

**Docling** is a popular choice for MCP document parsing because it handles:

- **Complex layouts** - Tables, multi-column layouts, headers/footers
- **Multiple formats** - PDF, DOCX, PPTX, XLSX, HTML, and more
- **Structured extraction** - Preserves document structure and metadata
- **High accuracy** - Better than basic text extraction

**Perfect for RAG systems** where document quality directly impacts search results.

---

## Option 1: Docker Setup

You can run Docling's MCP server in Docker using the `docling-serve` image, which includes the `docling-mcp-server` command.

### Step 1: Pull and Run Docling Container

**Using docling-serve (includes MCP server):**

```bash
# Pull the CPU-optimized image
docker pull ghcr.io/docling-project/docling-serve-cpu:latest

# Run the MCP server with streamable-http transport
# IMPORTANT: Mount your documents directory so Docling can access files
docker run -d \
  --name docling-mcp \
  -p 8000:8000 \
  -v "$(pwd)/test-docs:/docs:ro" \
  ghcr.io/docling-project/docling-serve-cpu:latest \
  /opt/app-root/bin/docling-mcp-server \
  --transport streamable-http \
  --host 0.0.0.0 \
  --port 8000
```

**Important:** The `-v "$(pwd)/test-docs:/docs:ro"` flag mounts your local `test-docs` directory into the container at `/docs` (read-only). When you use path mapping in Hector (`--docs-folder test-docs:/docs`), Hector will remap file paths to match the container mount point.

**Path Mapping:** Hector's path mapping feature (`local:remote` syntax) solves the Docker path mismatch problem:
1. Docker mount: `-v "$(pwd)/test-docs:/docs:ro"` (local `test-docs` → container `/docs`)
2. Hector flag: `--docs-folder test-docs:/docs` (tells Hector to remap paths to `/docs`)
3. Result: Hector sends `/docs/file.pdf` instead of `/Users/you/.../test-docs/file.pdf`

**Note:** If your documents are in a different location, adjust both the volume mount and the path mapping:
- Mount: `-v /path/to/your/documents:/docs:ro`
- Hector: `--docs-folder /path/to/your/documents:/docs`

**For GPU support:**

```bash
# Pull the GPU-enabled image
docker pull ghcr.io/docling-project/docling-serve-cu128:latest

# Run with GPU support
# IMPORTANT: Mount your documents directory
docker run -d \
  --name docling-mcp \
  --gpus all \
  -p 8000:8000 \
  -v "$(pwd)/test-docs:/docs:ro" \
  ghcr.io/docling-project/docling-serve-cu128:latest \
  /opt/app-root/bin/docling-mcp-server \
  --transport streamable-http \
  --host 0.0.0.0 \
  --port 8000
```

### Step 2: Verify Docling MCP Server is Running

Check the logs to confirm the server started:

```bash
docker logs docling-mcp

# You should see:
# INFO:     Uvicorn running on http://0.0.0.0:8000
# INFO:     StreamableHTTP session manager started
```

### Step 3: Configure Hector

**Quick Start (CLI) with Path Mapping:**

```bash
hector serve \
  --docs-folder test-docs:/docs \
  --mcp-url http://localhost:8000/mcp \
  --mcp-parser-tool convert_document_into_docling_document \
  --tools
```

**Key Points:**
- The `test-docs:/docs` syntax maps your local `test-docs` folder to `/docs` inside the Docker container
- This matches the volume mount: `-v "$(pwd)/test-docs:/docs:ro"`
- Hector remaps paths before sending to Docling (e.g., `/Users/you/.../test-docs/file.pdf` → `/docs/file.pdf`)

**Important:** For streamable-http transport, use the `/mcp` endpoint: `http://localhost:8000/mcp` (not just the base URL).

**Using Configuration File:**

Create `configs/docling-docker.yaml`:

```yaml
global:
  a2a_server:
    host: "0.0.0.0"
    port: 8080

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
    max_tokens: 4000

vector_stores:
  qdrant-db:
    type: "qdrant"
    host: "localhost"
    port: 6334

embedders:
  ollama-embedder:
    type: "ollama"
    model: "nomic-embed-text"
    host: "http://localhost:11434"

tools:
  # Docling MCP tool - provides document parsing capabilities
  docling:
    type: "mcp"
    enabled: true
    internal: true  # Not visible to agents (used only for document parsing)
    server_url: "http://localhost:8000/mcp"  # Include /mcp endpoint for streamable-http transport
    description: "Docling - Advanced document parsing and conversion"

document_stores:
  knowledge_base:
    path: "./test-docs"
    source: "directory"
    # Configure MCP parsers to use Docling for document parsing
    mcp_parsers:
      tool_names:
        - "convert_document_into_docling_document"
      extensions:
        - ".pdf"
        - ".docx"
        - ".pptx"
        - ".xlsx"
        - ".html"
      priority: 8  # Higher than native parsers, so MCP is preferred
      path_prefix: "/docs"  # Remap paths for Docker container (matches -v ./test-docs:/docs)

agents:
  docling_assistant:
    name: "Docling Assistant"
    description: "Assistant with advanced document parsing via Docling"
    llm: "gpt-4o"
    vector_store: "qdrant-db"
    embedder: "ollama-embedder"
    document_stores: ["knowledge_base"]
    prompt:
      system_prompt: |
        You are a helpful assistant with access to documents parsed using Docling.
        Documents are parsed with high accuracy, preserving structure and metadata.
```

Run Hector:

```bash
hector serve --config configs/docling-docker.yaml
```

---

## Option 2: Local Setup (Recommended for Development)

If you prefer running Docling locally without Docker, this is the simplest approach and **avoids path mapping issues** that can occur with Docker volume mounts.

### Step 1: Create Virtual Environment

```bash
# Create virtual environment
python3 -m venv docling-env

# Activate virtual environment
source docling-env/bin/activate  # On macOS/Linux
# or
docling-env\Scripts\activate  # On Windows
```

### Step 2: Install Docling

```bash
# Install docling (includes docling-mcp)
pip install docling
```

### Step 3: Start Docling MCP Server

```bash
# Start the MCP server with streamable-http transport
uvx --from docling-mcp docling-mcp-server --transport streamable-http
```

The server will start on `http://localhost:8000` by default.

### Step 4: Configure Hector

**Using CLI:**

```bash
hector serve \
  --docs-folder test-docs \
  --mcp-url http://localhost:8000/mcp \
  --mcp-parser-tool convert_document_into_docling_document \
  --tools
```

**Using Configuration File:**

Update your config file to use `server_url: "http://localhost:8000/mcp"` (include the `/mcp` path for streamable-http transport).

---

## Testing the Integration

### Step 1: Add Test Documents

Place some documents in your `test-docs` folder:

```bash
mkdir -p test-docs
# Add PDFs, DOCX files, etc.
cp your-document.pdf test-docs/
cp your-presentation.pptx test-docs/
```

### Step 2: Start Hector

```bash
# Using Docker setup
hector serve --config configs/docling-docker.yaml

# Or using CLI
hector serve \
  --docs-folder test-docs \
  --mcp-url http://localhost:8000/mcp \
  --mcp-parser-tool convert_document_into_docling_document \
  --tools
```

### Step 3: Test Document Parsing

```bash
# Chat with the agent
hector chat --config configs/docling-docker.yaml --agent docling_assistant

# Or call directly
hector call --config configs/docling-docker.yaml \
  --agent docling_assistant \
  "What information is in the documents?"
```

The agent will use Docling to parse documents and answer questions based on the extracted content.

---

## Understanding the Configuration

### MCP Parser Configuration

```yaml
document_stores:
  knowledge_base:
    mcp_parsers:
      tool_names: 
        - "convert_document_into_docling_document"
      extensions: 
        - ".pdf"
        - ".docx"
        - ".pptx"
        - ".xlsx"
        - ".html"
      priority: 8
```

**Key settings:**

- **`tool_names`**: The MCP tool to use for parsing (Docling's tool name)
- **`extensions`**: File types to parse with Docling
- **`priority`**: Higher priority means Docling is tried before native parsers

### Internal Tools

```yaml
tools:
  docling:
    type: "mcp"
    internal: true  # Hide from agents, available for document stores
```

Setting `internal: true` means:
- ✅ Available for document parsing
- ✅ Not visible to agents (keeps tool list clean)
- ✅ Used automatically by document stores

---

## Supported Document Formats

Docling supports parsing:

- **PDF** - Complex layouts, tables, multi-column
- **DOCX** - Word documents with formatting
- **PPTX** - PowerPoint presentations
- **XLSX** - Excel spreadsheets
- **HTML** - Web pages and HTML documents
- **And more** - Check Docling documentation for full list

---

## Docker Compose Setup

For production deployments, use Docker Compose to run both Hector and Docling in containers. This ensures proper path mapping and eliminates file access issues.

**`docker-compose.docling.yaml`:**

```yaml
services:
  docling:
    image: ghcr.io/docling-project/docling-serve-cpu:latest
    container_name: docling-mcp
    ports:
      - "8000:8000"
    restart: unless-stopped
    command: /opt/app-root/bin/docling-mcp-server --transport streamable-http --host 0.0.0.0 --port 8000
    volumes:
      # Mount documents directory for Docling to access
      - ./test-docs:/docs:ro
    healthcheck:
      test: ["CMD-SHELL", "curl -s http://localhost:8000/ > /dev/null || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 15s
    networks:
      - docling-network

  hector:
    image: kadirpekel/hector:latest
    container_name: hector-docling
    ports:
      - "8080:8080"
    depends_on:
      docling:
        condition: service_healthy
    environment:
      # Use Docker service name for internal communication
      - MCP_URL=http://docling:8000/mcp
      # Add your OpenAI API key (or other LLM provider)
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      # Enable debug logging to see MCP responses
      - LOG_LEVEL=debug
    volumes:
      # Mount documents directory for Hector
      - ./test-docs:/documents:ro
      # Mount config directory if you have custom configs
      - ./configs:/app/configs:ro
    command: >
      /app/hector serve
      --docs-folder /documents:/docs
      --mcp-url http://docling:8000/mcp
      --mcp-parser-tool convert_document_into_docling_document
      --tools
    # The --docs-folder syntax "local:remote" maps Hector's /documents to Docling's /docs
    # This allows containers to have different mount points while still communicating correctly
    networks:
      - docling-network

networks:
  docling-network:
    driver: bridge
```

**Key Points:**

1. **Path Mapping**: Hector mounts at `/documents`, Docling at `/docs`. The `--docs-folder /documents:/docs` syntax tells Hector to remap paths so Docling can find files.

2. **Docker Networking**: Hector uses `http://docling:8000/mcp` (Docker service name with `/mcp` endpoint) for internal communication.

3. **Health Checks**: Docling has a health check, and Hector waits for it to be healthy before starting.

**Start everything:**

```bash
docker-compose -f docker-compose.docling.yaml up -d
```

**Verify it's working:**

```bash
# Check both containers are running
docker-compose -f docker-compose.docling.yaml ps

# Check Hector logs
docker-compose -f docker-compose.docling.yaml logs hector

# Check Docling logs
docker-compose -f docker-compose.docling.yaml logs docling
```

**Using a Configuration File:**

If you prefer using a config file instead of CLI flags, create `configs/docling-docker.yaml`:

```yaml
global:
  a2a_server:
    host: "0.0.0.0"
    port: 8080

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"

vector_stores:
  qdrant-db:
    type: "qdrant"
    host: "localhost"  # Use host.docker.internal if Qdrant runs on host
    port: 6334

document_stores:
  docs:
    type: "directory"
    path: "/documents"  # Use the Docker mount point

mcp:
  sources:
    - name: "docling"
      url: "http://docling:8000/mcp"  # Use Docker service name with /mcp endpoint
      parser_tools:
        - "convert_document_into_docling_document"
```

Then update the docker-compose command:

```yaml
command: /app/hector serve --config /app/configs/docling-docker.yaml
environment:
  - OPENAI_API_KEY=${OPENAI_API_KEY}
```

**Note:** Make sure to set the `OPENAI_API_KEY` environment variable before starting:

```bash
export OPENAI_API_KEY=your-api-key-here
docker-compose -f docker-compose.docling.yaml up -d
```

---

## Troubleshooting

### Docling Not Accessible

**Check if Docling is running:**

```bash
# Check Docker container
docker ps | grep docling

# Check logs
docker logs docling-mcp

# For local setup, check if process is running
ps aux | grep docling-mcp-server
```

### MCP Connection Issues

**Verify the MCP server is running:**

```bash
# Check Docker logs
docker logs docling-mcp | tail -20

# You should see:
# INFO:     Uvicorn running on http://0.0.0.0:8000
# INFO:     StreamableHTTP session manager started
```

**Common issues:**

1. **Wrong URL format**: Use the `/mcp` endpoint (`http://localhost:8000/mcp`) for streamable-http transport
2. **Port mismatch**: Ensure the port in Hector config matches the port Docling is running on
3. **Server not started**: Check logs to confirm the server started successfully
4. **File not found errors**: If you see `FileNotFoundError` in Docling logs like `FileNotFoundError: [Errno 2] No such file or directory: '/Users/.../test-docs/...'`, this means:
   - **Root cause**: Hector is sending absolute host paths that don't exist inside the Docker container
   - **Solution (Recommended)**: Use **path mapping** - the `local:remote` syntax in `--docs-folder`:
     ```bash
     # Docker mount
     docker run -v "$(pwd)/test-docs:/docs:ro" -p 8000:8000 ...

     # Hector with path mapping
     hector serve --docs-folder test-docs:/docs --mcp-url http://localhost:8000/mcp --mcp-parser-tool convert_document_into_docling_document
     ```
   - **Alternative**: Use **local setup** (Option 2) - run both Hector and Docling on the host, avoiding path mapping entirely

   **How path mapping works**: The `test-docs:/docs` syntax tells Hector to remap file paths before sending to Docling. Instead of `/Users/you/.../test-docs/file.pdf`, Hector sends `/docs/file.pdf`, which matches the Docker mount point.

### Tool Not Found

**Check available tools:**

Hector will log available MCP tools on startup. Look for:

```
MCP tools available: convert_document_into_docling_document, ...
```

If the tool isn't listed, verify:
1. Docling server is running and logs show successful startup
2. MCP URL in config uses base URL (not `/mcp` path)
3. Server URL in config matches actual port

### Port Conflicts

**Change ports if needed:**

```bash
# Docker: Map to different host port
docker run -d \
  --name docling-mcp \
  -p 9000:8000 \
  ghcr.io/docling-project/docling-serve-cpu:latest \
  /opt/app-root/bin/docling-mcp-server \
  --transport streamable-http \
  --host 0.0.0.0 \
  --port 8000

# Update Hector config
server_url: "http://localhost:9000"
```

---

## Performance Tips

### GPU Acceleration

For better performance with large documents:

```bash
# Use GPU-enabled image
docker run -d \
  --gpus all \
  -p 5001:5001 \
  ghcr.io/docling-project/docling-serve-cu128:latest
```

### Resource Limits

Set Docker resource limits:

```bash
docker run -d \
  --name docling \
  --memory="4g" \
  --cpus="2" \
  -p 8000:8000 \
  ai/granite-docling
```

### Caching

Hector caches parsed documents. For frequently accessed documents, parsing happens once and results are reused.

---

## Next Steps

- [RAG Documentation](../../core-concepts/rag.md) - Learn about RAG in Hector
- [MCP Document Parsing](../../core-concepts/rag.md#mcp-document-parsing) - MCP parsing details
- [Docling Documentation](https://www.docling.ai/) - Official Docling docs
- [Building Enterprise RAG Systems](building-enterprise-rag-systems.md) - Complete RAG guide

---

**About Hector**: Hector is a production-grade A2A-native agent platform designed for enterprise deployments. Learn more at [gohector.dev](https://gohector.dev).

