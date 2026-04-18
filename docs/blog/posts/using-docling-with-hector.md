---
title: Using Docling with Hector for Advanced Document Parsing
description: Integrate Docling's powerful document parsing capabilities with Hector
date: 2026-01-20
tags:
  - RAG
  - Document Parsing
  - MCP
  - Docling
  - Tutorial
hide:
  - navigation
---

# Using Docling with Hector for Advanced Document Parsing

Enhance your RAG system with Docling's advanced document parsing capabilities. Parse PDFs, Word documents, PowerPoint presentations, and more with enterprise-grade accuracy.

**Time:** 10-15 minutes
**Difficulty:** Beginner

---

## What You'll Learn

- Understand Hector's MCP document parsing feature
- Set up Docling using Docker
- Configure Hector to use Docling for document parsing
- Parse complex documents (PDFs, DOCX, PPTX, XLSX, HTML)

---

## Hector's MCP Document Parsing

Hector's document stores support **MCP-based document parsing**, allowing you to use any MCP-compliant service to parse documents during indexing. This is configured via the `mcp_parsers` option in your document store configuration.

**Key benefits:**

- **Pluggable architecture** - Use any MCP service that can parse documents
- **Format flexibility** - Support formats beyond Hector's native parsers
- **Quality improvements** - Better parsing for complex layouts, tables, OCR
- **Fallback chains** - Configure multiple parsers with priority ordering

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

## Docker Setup

You can run Docling's MCP server in Docker using the `docling-serve` image.

### Step 1: Pull and Run Docling Container

**Run with Docling MCP Server:**

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

**Important:** The `-v "$(pwd)/test-docs:/docs:ro"` flag mounts your local `test-docs` directory into the container at `/docs` (read-only). Hector will explicitly remap file paths to use this container path.

---

## Configure Hector

Create a configuration file `configs/docling-docker.yaml`.

**Configuration (`configs/docling-docker.yaml`):**

```yaml
version: "2"

llms:
  gpt-4o:
    provider: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"

vector_stores:
  qdrant-db:
    type: "qdrant"
    host: "localhost"
    port: 6334

embedders:
  nomic:
    provider: "ollama"
    model: "nomic-embed-text"
    base_url: "http://localhost:11434"

tools:
  # Docling MCP tool - provides document parsing capabilities
  docling:
    type: "mcp"
    enabled: true
    url: "http://localhost:8000/mcp"  # MCP server URL
    transport: "streamable-http"
    description: "Docling - Advanced document parsing and conversion"

document_stores:
  knowledge_base:
    source:
      type: "directory"
      max_file_size: 10485760 # 10MB
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
    embedder: "nomic"
    document_stores: ["knowledge_base"]
    instruction: |
        You are a helpful assistant with access to documents parsed using Docling.
        Documents are parsed with high accuracy, preserving structure and metadata.
```

## Running the System

### Step 1: Add Test Documents

Place some documents in your `test-docs` folder:

```bash
mkdir -p test-docs
# Add PDFs, DOCX files, etc.
# cp your-document.pdf test-docs/
```

### Step 2: Start Hector Server

Run Hector with the configuration file.

```bash
export OPENAI_API_KEY="sk-..."
hector serve --config configs/docling-docker.yaml --docs-folder test-docs
```

> **Note:** We map the local `test-docs` folder. The `path_prefix` in config handles the mapping for Docling.

### Step 3: Chat with the Agent

```bash
curl -X POST http://localhost:8080/agents/docling_assistant \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {
        "role": "user",
        "parts": [{"text": "What information is in the documents?"}]
      }
    },
    "id": 1
  }'
```

---

## Configuration Explained

### MCP Parser Configuration

```yaml
document_stores:
  knowledge_base:
    mcp_parsers:
      tool_names: ["convert_document_into_docling_document"]
      extensions: [".pdf", ".docx", ".pptx", ".xlsx", ".html"]
      priority: 8
      path_prefix: "/docs"
```

- **`tool_names`**: The specific tool within the Docling MCP server to call.
- **`path_prefix`**: Crucially important for Docker setups. It tells Hector to rewrite the file path from your local path (e.g. `test-docs/file.pdf`) to the container path (`/docs/file.pdf`) before sending it to Docling.

### Internal Tools

We defined the `docling` tool but **did not** add it to the agent's `tools` list.

```yaml
tools:
  docling: { ... }

agents:
  docling_assistant:
    tools: [] # Docling is NOT listed here
```

This ensures the tool is available for the system (Document Store) to use for parsing, but the Agent itself won't try to call "Docling" as a function during conversation. The agent simply benefits from the *result* (parsed text).

---

## Supported Formats

Docling supports:

- **PDF** - Complex layouts, tables, multi-column
- **DOCX** - Word documents
- **PPTX** - PowerPoint
- **XLSX** - Excel
- **HTML** - Web pages

---
