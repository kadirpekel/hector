---
title: Build a Coding Assistant
description: Create a Cursor-like AI coding assistant with semantic search in 30 minutes
---

# How to Build a Coding Assistant

Build a powerful AI coding assistant with semantic code search, file operations, and command execution—comparable to Cursor or GitHub Copilot—using only YAML configuration.

**Time:** 30 minutes  
**Difficulty:** Intermediate

---

## What You'll Build

A coding assistant that can:

- **Search code semantically** - Find relevant code by meaning, not keywords
- **Edit files** - Create, modify, and refactor code
- **Run commands** - Execute tests, linters, and build tools
- **Reason step-by-step** - Chain-of-thought problem solving
- **Stream responses** - Real-time output as it works

---

## Prerequisites

### Required

✅ Hector installed ([Installation Guide](../getting-started/installation.md))  
✅ API key from Anthropic or OpenAI

### Optional (for semantic search)

⭐ Qdrant (vector database)  
⭐ Ollama (embeddings)

**Note:** Semantic search significantly improves codebase exploration but isn't required for basic functionality.

---

## Step 1: Set Up Dependencies

### Start Qdrant (Optional but Recommended)

```bash
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -p 6334:6334 \
  qdrant/qdrant
```

Verify: http://localhost:6333/dashboard

### Start Ollama (Optional but Recommended)

```bash
# Install Ollama
curl https://ollama.ai/install.sh | sh

# Pull embedding model
ollama pull nomic-embed-text
```

### Set API Key

```bash
# Anthropic (recommended for coding)
export ANTHROPIC_API_KEY="sk-ant-..."

# Or OpenAI
export OPENAI_API_KEY="sk-..."
```

---

## Step 2: Create Configuration

Create `coding-assistant.yaml`:

```yaml
# LLM Configuration
llms:
  claude:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"
    temperature: 0.0  # Deterministic for code
    max_tokens: 8000

# Vector Database (for semantic search)
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6333

# Embedder (for semantic search)
embedders:
  embedder:
    type: "ollama"
    host: "http://localhost:11434"
    model: "nomic-embed-text"

# Coding Assistant Agent
agents:
  coder:
    name: "AI Coding Assistant"
    llm: "claude"
    database: "qdrant"
    embedder: "embedder"
    
    # Prompt Configuration
    prompt:
      prompt_slots:
        system_role: |
          You are an expert AI coding assistant.
          You operate like a pair programmer in Cursor.
          
          Your capabilities:
          - Semantic code search to understand codebases
          - File creation and modification
          - Command execution for testing and validation
          - Step-by-step reasoning
        
        reasoning_instructions: |
          Always implement changes rather than just suggesting them.
          
          For each task:
          1. Use semantic search to understand the codebase
          2. Analyze existing patterns and conventions
          3. Implement changes following project style
          4. Test your changes when possible
          5. Explain what you did and why
        
        tool_usage: |
          Use tools proactively:
          - search: Find relevant code semantically
          - write_file: Create or modify files
          - search_replace: Make precise edits
          - execute_command: Run tests, linters, builds
          - todo_write: Break down complex tasks
        
        communication_style: |
          Be concise but thorough.
          Show your reasoning process.
          Admit when you're unsure.
          Ask clarifying questions when needed.
    
    # Reasoning Configuration
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100
      enable_structured_reflection: true
      enable_streaming: true
      show_tool_execution: true
    
    # Tools
    tools:
      - "search"
      - "write_file"
      - "search_replace"
      - "execute_command"
      - "todo_write"
    
    # Reference to document stores (defined below)
    document_stores: ["codebase", "docs"]
    
    # Memory Configuration
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000

# Document Stores (semantic search targets)
document_stores:
  codebase:
    name: "codebase"
    paths: ["./src/", "./lib/", "./pkg/"]
    # Defaults: indexes all parseable files (text + code)
  
  docs:
    name: "docs"
    paths: ["./docs/", "./README.md"]
    # Defaults: indexes all parseable files

# Tool Configurations
tools:
  execute_command:
    type: command
    enabled: true
    # Permissive defaults: allows all commands (sandboxed for security)
  
  write_file:
    type: write_file
    enabled: true
    # Permissive defaults: allows all file types and paths
  
  search_replace:
    type: search_replace
    enabled: true
    # Permissive defaults: no restrictions
```

---

## Step 3: Start the Agent

```bash
hector serve --config coding-assistant.yaml
```

**First run:** If semantic search is configured, Hector will index your codebase. This may take a few minutes for large projects.

Output:
```
Hector server listening on :8080
Indexing codebase...
Indexed 1,234 chunks from 156 files
Agent registered: coder
```

---

## Step 4: Test Your Assistant

### Interactive Chat

```bash
hector chat --config coding-assistant.yaml coder
```

**Try these tasks:**

```
> How does authentication work in this codebase?
[Agent uses semantic search to find auth-related code]

> Add input validation to the login function
[Agent searches for login function, analyzes it, adds validation]

> Write tests for the new validation
[Agent creates test file with comprehensive tests]

> Run the tests
[Agent executes: npm test]
```

### Single Command

```bash
hector call --config coding-assistant.yaml coder "Add error handling to the API endpoints"
```

### Via API

```bash
curl -X POST http://localhost:8080/agents/coder/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "task": "Refactor the database connection logic to use connection pooling"
  }'
```

---

## Step 5: Customize for Your Project

### Adjust File Patterns

```yaml
document_stores:
  frontend:
    name: "frontend"
    paths: ["./frontend/src/"]
    # Defaults: indexes all parseable files
  
  backend:
    name: "backend"
    paths: ["./backend/"]
    # Defaults: indexes all parseable files
```

### Add Project-Specific Commands

```yaml
tools:
  execute_command:
    type: command
    enabled: true
    # All commands allowed by default (sandboxed)
    # Only restrict if needed:
    # allowed_commands: ["npm", "yarn", "pnpm"]
```

### Customize the Prompt

```yaml
agents:
  coder:
    prompt:
      prompt_slots:
        system_role: |
          You are an expert in our tech stack:
          - Frontend: React + TypeScript
          - Backend: Go + PostgreSQL
          - Infrastructure: Docker + Kubernetes
          
          Follow our conventions:
          - Use functional components in React
          - Write tests for all new features
          - Follow Go effective patterns
```

### Adjust Reasoning

```yaml
reasoning:
  max_iterations: 50    # Fewer for simpler tasks
  show_thinking: true   # Show internal reasoning
  show_debug_info: true # Debug mode
```

---

## Advanced Features

### Add Multiple Document Stores

```yaml
document_stores:
  source_code:
    name: "source_code"
    paths: ["./src/"]
    chunk_size: 512  # Smaller for precision
  
  documentation:
    name: "documentation"
    paths: ["./docs/"]
    chunk_size: 2048  # Larger for context
  
  configs:
    name: "configs"
    paths: ["./config/"]
    # Defaults: indexes all parseable files
```

### Use Different LLM for Speed

```yaml
llms:
  fast:
    type: "openai"
    model: "gpt-4o-mini"  # Faster, cheaper
  
  smart:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"  # Smarter, slower

agents:
  quick_coder:
    llm: "fast"  # Use for simple edits
  
  architect:
    llm: "smart"  # Use for complex refactoring
```

### Enable Memory for Context

```yaml
agents:
  coder:
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000
      longterm:
        enabled: true
        storage_scope: "session"
```

Now the agent remembers context across multiple requests!

---

## Production Tips

### Security

```yaml
tools:
  execute_command:
    type: command
    # Optional restrictions (only if needed):
    # allowed_commands: ["npm", "go", "git"]  # Whitelist only
    # denied_commands: ["rm", "dd", "sudo"]   # Blacklist dangerous
  
  write_file:
    type: write_file
    # Optional restrictions (only if needed):
    # allowed_paths: ["./src/", "./tests/"]   # Restrict paths
    # denied_paths: ["./secrets/", "./.env"]  # Protect sensitive
```

### Performance

```yaml
# Index in the background
document_stores:
  codebase:
    name: "codebase"
    batch_size: 100      # Index 100 docs at a time
    parallel: true       # Parallel processing
    cache_embeddings: true  # Cache for re-indexing
```

### Monitoring

```yaml
logging:
  level: "info"
  format: "json"
  
reasoning:
  show_tool_execution: true  # See what the agent does
  show_debug_info: false     # Disable in production
```

---

## Troubleshooting

### "Qdrant connection failed"

```bash
# Check if Qdrant is running
docker ps | grep qdrant

# Check logs
docker logs qdrant

# Verify connectivity
curl http://localhost:6333/
```

### "Ollama not responding"

```bash
# Check if Ollama is running
ollama list

# Pull model if missing
ollama pull nomic-embed-text

# Test embeddings
ollama run nomic-embed-text "test"
```

### "Search returns no results"

- Check documents are indexed in Qdrant dashboard
- Verify file patterns match your project structure
- Lower search threshold if too restrictive

### "Agent not making changes"

Review prompt - emphasize implementation:
```yaml
prompt_slots:
  reasoning_instructions: |
    ALWAYS implement changes directly.
    DO NOT just suggest changes.
    Use write_file and search_replace tools proactively.
```

---

## Example Use Cases

### Code Review

```bash
hector call coder "Review the authentication code for security issues"
```

Agent searches auth code, identifies issues, suggests fixes.

### Feature Implementation

```bash
hector call coder "Add rate limiting to the API endpoints"
```

Agent searches existing patterns, implements rate limiting, adds tests.

### Refactoring

```bash
hector call coder "Refactor the database layer to use repositories pattern"
```

Agent analyzes current structure, implements repository pattern, updates callers.

### Bug Fixing

```bash
hector call coder "Fix the memory leak in the connection pool"
```

Agent searches for connection pool code, identifies leak, implements fix.

---

## Complete Example

See the full configuration in `configs/coding.yaml` in the Hector repository.

**Key files:**
- Configuration: `configs/coding.yaml`
- Documentation: `docs/tutorial-cursor.md`
- Examples: `configs/README.md`

---

## Next Steps

- **[Set Up RAG](setup-rag.md)** - Detailed RAG configuration guide
- **[Build a Research System](build-research-system.md)** - Multi-agent orchestration
- **[Deploy to Production](deploy-production.md)** - Production deployment
- **[Add Custom Tools](add-custom-tools.md)** - Extend with MCP tools

---

## Related Topics

- **[Tools](../core-concepts/tools.md)** - Understanding the tool system
- **[RAG & Semantic Search](../core-concepts/rag.md)** - How semantic search works
- **[Reasoning Strategies](../core-concepts/reasoning.md)** - Chain-of-thought details
- **[Prompts](../core-concepts/prompts.md)** - Prompt engineering

