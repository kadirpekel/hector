---
title: Build a Cursor-like AI Coding Assistant
description: Create a production-ready coding assistant with semantic search, self-reflection, and tool execution in 30 minutes—using only YAML
---

# Build a Cursor-like AI Coding Assistant

Create a **production-ready AI coding assistant** with semantic code search, intelligent reasoning, and powerful file operations—all through declarative YAML configuration. No code required.

**Time:** 30 minutes  
**Difficulty:** Beginner to Intermediate  
**Perfect for:** Developers wanting AI-powered code assistance

---

## 🎯 What You'll Build

By the end of this guide, you'll have a coding assistant that:

- ✅ **Searches code by meaning** - Semantic search finds relevant code without keywords
- ✅ **Shows its reasoning** - See the LLM's internal `<thinking>` process
- ✅ **Edits files intelligently** - Creates, modifies, and refactors code
- ✅ **Executes commands safely** - Runs tests, linters, build tools (sandboxed)
- ✅ **Streams responses** - Watch it work in real-time
- ✅ **Remembers context** - Maintains conversation history

**Demo:**
```
You: "Add input validation to the login API"

<thinking>
I need to find the login API endpoint first.
Then analyze current implementation.
Add validation using best practices.
Write tests to verify the changes.
</thinking>

I'll implement input validation for the login API...
🔧 search: login API endpoint ✅
🔧 write_file: validators/auth.py ✅
🔧 search_replace: routes/auth.py ✅
🔧 execute_command: pytest tests/test_auth.py ✅

[Thinking: Iteration 1: Analyzing results]
[Thinking: ✅ Succeeded: search, write_file, search_replace, execute_command]
[Thinking: Confidence: 95% - Continue]

✅ Added input validation with email format check, password strength requirements
✅ Updated login route to use new validators
✅ All tests passing
```

---

## 📋 Prerequisites

### Required

- ✅ **Hector installed** - [Installation Guide](../getting-started/installation.md)
- ✅ **API Key** - Anthropic Claude or OpenAI GPT-4

### Optional (for Semantic Search)

- ⭐ **Qdrant** - Vector database for code search
- ⭐ **Ollama** - Local embeddings

> **Note:** Semantic search dramatically improves code understanding, but basic functionality works without it.

---

## 🚀 Quick Start (5 Minutes)

### 1. Set Up Dependencies

**Start Qdrant (Vector Database):**
```bash
docker run -d \
  --name qdrant \
  -p 6334:6334 \
  -p 6333:6333 \
  qdrant/qdrant
```

Verify at: http://localhost:6334/dashboard

**Start Ollama (Embeddings):**
```bash
# Install
curl https://ollama.ai/install.sh | sh

# Pull embedding model
ollama pull nomic-embed-text
```

**Set API Key:**
```bash
# Anthropic (recommended for coding)
export ANTHROPIC_API_KEY="sk-ant-..."

# Or OpenAI
export OPENAI_API_KEY="sk-..."
```

---

### 2. Create Your Configuration

Create `coder.yaml`:

```yaml
# LLM Configuration
llms:
  claude:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"
    temperature: 0.0          # Deterministic for code
    max_tokens: 8000

# Your Coding Assistant
agents:
  coder:
    name: "AI Coding Assistant"
    llm: "claude"
    
    # 🎯 Quick Config Shortcuts (Recommended for Getting Started)
    docs_folder: "."          # Index current directory
    enable_tools: true        # Enable all coding tools
    
    # 🧠 Enhanced Reasoning (Show the AI's thinking!)
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100
      enable_self_reflection: true        # LLM outputs <thinking> tags
      enable_structured_reflection: true  # Analyze tool execution
      show_thinking: true                 # Display reasoning blocks
      enable_streaming: true
      show_tool_execution: true
    
    # 💾 Conversation Memory
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000          # Keep 4000 tokens of context
```

**What the shortcuts auto-configure:**
- ✅ Indexes your entire codebase
- ✅ Connects to Qdrant (localhost:6334)
- ✅ Uses Ollama for embeddings (localhost:11434)
- ✅ Enables semantic search tool
- ✅ Enables all file tools (write_file, search_replace)
- ✅ Enables command execution (sandboxed)

---

### 3. Start Your Assistant

**Server Mode:**
```bash
hector serve --config coder.yaml
```

Output:
```
🚀 Hector server listening on :8080
🔍 Indexing codebase from: ./
✅ Indexed 1,234 chunks from 156 files
🤖 Agent registered: coder
```

**Interactive Mode:**
```bash
hector chat --config coder.yaml coder
```

---

### 4. Try It Out!

**Example Tasks:**

```bash
# Understand existing code
> How does authentication work in this codebase?

# Implement features
> Add rate limiting to the API endpoints

# Refactor code
> Refactor the database layer to use the repository pattern

# Fix bugs
> Fix the memory leak in the connection pool

# Run tests
> Write unit tests for the auth module and run them
```

**API Mode:**
```bash
curl -X POST http://localhost:8080/agents/coder/tasks \
  -H "Content-Type: application/json" \
  -d '{"task": "Add input validation to all API routes"}'
```

---

## ⚙️ Fine-Tuning Your Configuration

### Option A: Shortcuts (Recommended for Beginners)

Perfect for quick setup with sensible defaults:

```yaml
agents:
  coder:
    docs_folder: "./src"      # Index source directory
    enable_tools: true        # All tools enabled
```

**Customize the prompt:**
```yaml
agents:
  coder:
    docs_folder: "./src"
    enable_tools: true
    
    prompt:
      prompt_slots:
        system_role: |
          You are an expert in our tech stack:
          - Frontend: React + TypeScript
          - Backend: Python + FastAPI
          - Database: PostgreSQL
          
        reasoning_instructions: |
          Always implement changes, never just suggest.
          Write tests for all new features.
          Use semantic search to understand code first.
```

### Option B: Advanced Control

For production or complex needs, define everything explicitly:

```yaml
# Vector Database
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334

# Embedder
embedders:
  ollama:
    type: "ollama"
    host: "localhost"
    port: 11434
    model: "nomic-embed-text"

# Document Stores
document_stores:
  frontend:
    name: "frontend"
    source: "directory"
    path: "./frontend/src"
    chunk_size: 512           # Smaller for precision
    watch_changes: true
  
  backend:
    name: "backend"
    source: "directory"
    path: "./backend"
    chunk_size: 512

# Tools (explicit configuration)
tools:
  search:
    type: "search"
    document_stores: ["frontend", "backend"]
  
  write_file:
    type: "write_file"
    max_file_size: 1048576    # 1MB limit
    # All file types allowed by default
  
  search_replace:
    type: "search_replace"
    max_replacements: 100
  
  execute_command:
    type: "command"
    enable_sandboxing: true
    max_execution_time: "30s"
    # All commands allowed (sandboxed)
  
  todo_write:
    type: "todo"

# Agent Configuration
agents:
  coder:
    name: "Production Coding Assistant"
    llm: "claude"
    database: "qdrant"
    embedder: "ollama"
    document_stores: ["frontend", "backend"]
    tools:
      - "search"
      - "write_file"
      - "search_replace"
      - "execute_command"
      - "todo_write"
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100
      enable_self_reflection: true
      enable_structured_reflection: true
      show_thinking: true
      enable_streaming: true
      show_tool_execution: true
    
    memory:
      working:
        strategy: "summary_buffer"
        budget: 4000
        threshold: 0.8
        target: 0.6
      longterm:
        storage_scope: "session"
```

**When to use advanced config:**
- ✅ Multiple document stores
- ✅ Custom tool restrictions
- ✅ Production deployments
- ✅ Team-specific workflows

---

## 🎨 Customization Examples

### Multiple Specialized Agents

```yaml
agents:
  # Frontend specialist
  frontend_dev:
    docs_folder: "./frontend"
    enable_tools: true
    prompt:
      prompt_slots:
        system_role: "React + TypeScript expert"
  
  # Backend specialist
  backend_dev:
    docs_folder: "./backend"
    enable_tools: true
    prompt:
      prompt_slots:
        system_role: "Python + FastAPI expert"
  
  # Full-stack architect
  architect:
    docs_folder: "."
    enable_tools: true
    reasoning:
      max_iterations: 150     # More complex tasks
```

### Adjust Reasoning Visibility

```yaml
reasoning:
  # Development mode (see everything)
  enable_self_reflection: true   # See LLM's <thinking>
  show_thinking: true            # See meta-analysis
  show_tool_execution: true      # See tool calls
  show_debug_info: true          # See iterations/tokens
  
  # Production mode (quieter)
  # enable_self_reflection: false
  # show_thinking: false
  # show_debug_info: false
```

### Security & Safety

```yaml
tools:
  execute_command:
    type: "command"
    enable_sandboxing: true     # Always recommended
    
    # Optional: Restrict commands
    # allowed_commands: ["npm", "pytest", "go", "git"]
    # denied_commands: ["rm", "dd", "sudo", "curl"]
  
  write_file:
    type: "write_file"
    
    # Optional: Restrict paths
    # allowed_paths: ["./src/", "./tests/"]
    # denied_paths: ["./secrets/", "./.env", "./config/"]
```

> **💡 Tip:** By default, `execute_command` uses sandboxing, so all commands are safe. Only add restrictions if you need extra control.

---

## 📚 Example Use Cases

### 1. Code Review

```bash
hector call --config coder.yaml coder \
  "Review the authentication module for security issues"
```

The agent will:
1. Search for authentication code
2. Analyze for common vulnerabilities
3. Suggest fixes with code examples

### 2. Feature Implementation

```bash
hector call --config coder.yaml coder \
  "Add rate limiting middleware to all API endpoints"
```

The agent will:
1. Search for existing middleware patterns
2. Implement rate limiting
3. Update route configurations
4. Write tests

### 3. Refactoring

```bash
hector call --config coder.yaml coder \
  "Refactor the user service to follow clean architecture"
```

The agent will:
1. Analyze current structure
2. Create new architecture
3. Migrate code incrementally
4. Update imports and tests

### 4. Testing

```bash
hector call --config coder.yaml coder \
  "Generate comprehensive tests for the payment module"
```

The agent will:
1. Search for payment-related code
2. Identify edge cases
3. Write unit and integration tests
4. Run tests to verify

---

## 🔗 Complete Example

See the full production-ready configuration: [`configs/coding.yaml`](https://github.com/kadirpekel/hector/blob/main/configs/coding.yaml)

For maximum control: [`configs/coding-advanced.yaml`](https://github.com/kadirpekel/hector/blob/main/configs/coding-advanced.yaml)

---

## 🎓 Next Steps

Ready to level up? Check out these guides:

- **[Setup RAG & Semantic Search](setup-rag.md)** - Deep dive into semantic code search
- **[Build a Research System](build-research-system.md)** - Multi-agent orchestration
- **[Deploy to Production](deploy-production.md)** - Docker, Kubernetes, monitoring
- **[Add Custom Tools](add-custom-tools.md)** - Extend with MCP integrations

---

## 📖 Learn More

**Core Concepts:**
- [Tools](../core-concepts/tools.md) - Understanding the tool system
- [RAG & Semantic Search](../core-concepts/rag.md) - How semantic search works
- [Reasoning Strategies](../core-concepts/reasoning.md) - Chain-of-thought & self-reflection
- [Memory](../core-concepts/memory.md) - Context management
- [Prompts](../core-concepts/prompts.md) - Prompt engineering

**Reference:**
- [Configuration Reference](../reference/configuration.md) - Complete config options
- [CLI Reference](../reference/cli.md) - All CLI commands
- [API Reference](../reference/api.md) - HTTP API documentation

---

## 💬 Community & Support

- **GitHub:** [github.com/kadirpekel/hector](https://github.com/kadirpekel/hector)
- **Issues:** [Report bugs or request features](https://github.com/kadirpekel/hector/issues)
- **Discussions:** [Ask questions & share ideas](https://github.com/kadirpekel/hector/discussions)

---

**Built with Hector?** Share your experience! Tag us on social media with **#HectorAI** 🚀
