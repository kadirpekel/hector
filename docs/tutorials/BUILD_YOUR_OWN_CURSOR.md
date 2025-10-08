# Build Your Own Cursor-Like AI Coding Assistant in Pure YAML

**TL;DR:** Modern AI coding assistants use well-crafted prompts, chain-of-thought reasoning, and semantic search to help developers write code. In this tutorial, you'll build a similar AI coding assistant using **only YAML configuration** - no code required.

---

## Table of Contents

- [Why This Tutorial](#why-this-tutorial)
- [Understanding AI Coding Assistants](#understanding-ai-coding-assistants)
- [The Configuration](#the-configuration)
- [Getting Started](#getting-started)
- [Example Tasks](#example-tasks)
- [Architecture Deep Dive](#architecture-deep-dive)
- [Customization](#customization)
- [Troubleshooting](#troubleshooting)

---

## Why This Tutorial

This tutorial demonstrates how to build a powerful AI coding assistant using **pure declarative configuration** - no programming required. You'll learn:

1. **How AI coding assistants work** - Understanding the core components (prompts, reasoning loops, tool execution)
2. **Configuration over code** - Building complex behavior through YAML configuration
3. **Flexibility through modularity** - Swapping LLM providers, customizing tools, and adjusting behavior

By the end, you'll have:
- ✅ A working AI coding assistant
- ✅ Deep understanding of how these systems operate
- ✅ Full control to customize for your specific needs
- ✅ Foundation to experiment with different approaches

---

## Understanding AI Coding Assistants

AI coding assistants are built from several key components that work together:

### 1. The Prompts

Modern AI coding assistants use carefully crafted prompts with LLMs like Claude Sonnet. Here's an example of an effective system prompt pattern:

```
You are an AI coding assistant, powered by Claude Sonnet 4.5. You operate in Cursor.

You are pair programming with a USER to solve their coding task. Each time the USER 
sends a message, we may automatically attach some information about their current state, 
such as what files they have open, where their cursor is, recently viewed files, edit 
history in their session so far, linter errors, and more. This information may or may 
not be relevant to the coding task, it is up for you to decide.

Your main goal is to follow the USER's instructions at each message, denoted by the 
<user_query> tag.
```

**Key insight:** The prompt establishes Claude as a "pair programmer" who has context and should take action, not just make suggestions.

### 2. Reasoning Instructions

```
By default, implement changes rather than only suggesting them.
Be THOROUGH when gathering information. Make sure you have the FULL picture before replying.
TRACE every symbol back to its definitions and usages so you fully understand it.
Semantic search is your MAIN exploration tool.
Bias towards not asking the user for help if you can find the answer yourself.
```

**Key insight:** Bias toward **action** (not suggestions), **thoroughness** (not quick answers), and **self-sufficiency** (not asking users).

### 3. Chain-of-Thought Loop

The core reasoning loop is simple - continue until the LLM stops requesting tool calls:

```go
func (s *ChainOfThoughtStrategy) ShouldStop(...) bool {
    return len(toolCalls) == 0  // Stop when no more tool calls
}
```

**Key insight:** Loop until the LLM stops making tool calls. The model determines when it has enough information to provide a complete answer.

### 4. Parallel Tool Calls

```
If you intend to call multiple tools and there are no dependencies between the tool calls, 
make all of the independent tool calls in parallel.
```

**Key insight:** Read 3 files? Make 3 parallel calls. Explore 5 functions? 5 parallel searches. This is **fast**.

### 5. Semantic Search

```
- CRITICAL: Start with a broad, high-level query (e.g. "authentication flow")
- MANDATORY: Run multiple searches with different wording
- Keep searching new areas until you're CONFIDENT nothing important remains
```

**Key insight:** Semantic search is the **main** exploration tool, not a fallback. Start broad, then narrow.

---

## The Configuration

Now that you understand the components, let's see how simple it is to configure them. Here's the **complete configuration** for a capable AI coding assistant:

```yaml
agents:
  coding_assistant:
    name: "Coding Assistant"
    llm: "sonnet-llm"
    
    prompt:
      prompt_slots:
        system_role: |
          You are an AI coding assistant, powered by Claude Sonnet 4.5.
          You operate in Cursor. You are pair programming with a USER.
        
        reasoning_instructions: |
          By default, implement changes rather than only suggesting them.
          Be THOROUGH. TRACE every symbol back to definitions.
          Semantic search is your MAIN exploration tool.
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100
      enable_structured_reflection: true
      enable_streaming: true
    
    tools:
      - "search"          # Semantic code search
      - "write_file"     # Create/update files
      - "search_replace"  # Precise edits
      - "execute_command" # Run commands
      - "todo_write"      # Task management

llms:
  sonnet-llm:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"
    temperature: 0.0   # Deterministic
    max_tokens: 8000
```

**That's it.** ~50 lines of YAML to build a powerful AI coding assistant.

See [`configs/coding.yaml`](../../configs/coding.yaml) for the complete, production-ready configuration with detailed comments.

---

## Getting Started

### Prerequisites

1. **Hector installed:**
   ```bash
   git clone https://github.com/kadirpekel/hector
   cd hector
   make build
   ```

2. **Claude API key:**
   ```bash
   export ANTHROPIC_API_KEY="sk-ant-..."
   ```

3. **For semantic search (strongly recommended for codebase exploration):**
   
   **Qdrant vector database:**
   ```bash
   docker run -p 6334:6333 -p 6333:6333 qdrant/qdrant
   ```
   
   **Ollama for embeddings:**
   ```bash
   # Install from: https://ollama.ai
   ollama pull nomic-embed-text
   ```
   
   **Note:** Semantic search is optional but significantly improves the agent's ability to explore and understand codebases. Without it, the agent can still edit files and run commands but cannot intelligently search through code.

### Quick Start

#### Option A: With Semantic Search (Recommended)

1. **Ensure Qdrant and Ollama are running** (see Prerequisites above)

2. **Copy and customize the configuration:**
   ```bash
   cp configs/coding.yaml configs/my-cursor.yaml
   # Edit configs/my-cursor.yaml if needed (API keys, etc.)
   ```

3. **Start the agent server:**
   ```bash
   hector serve --config configs/my-cursor.yaml
   ```
   
   **Note:** On first run, if you have document stores configured, the codebase will be indexed automatically. This may take a few minutes for large codebases. You'll see indexing progress in the logs.

4. **Chat with your agent:**
   ```bash
   hector chat coding_assistant
   ```

#### Option B: Without Semantic Search (Basic Mode)

If you want to skip the Qdrant/Ollama setup, you can run in basic mode:

1. **Create a simplified configuration:**
   ```yaml
   # In configs/my-cursor.yaml
   agents:
     coding_assistant:
       # Remove document_stores, database, embedder sections
       tools:
         # Remove "search" from tools list
         - "write_file"
         - "search_replace"
         - "execute_command"
         - "todo_write"
   ```

2. **Start the agent:**
   ```bash
   hector serve --config configs/my-cursor.yaml
   hector chat coding_assistant
   ```

**Note:** In basic mode, the agent can edit files and run commands but cannot intelligently explore the codebase through semantic search. This is suitable for small projects or when you provide explicit file paths.

---

## Example Tasks

### 1. Code Exploration

**Prompt:**
```
"How does the agent execute tasks in this codebase?"
```

**What happens:**
1. Agent uses semantic search: "agent task execution flow"
2. Finds `pkg/agent/agent.go` with `ExecuteTask` method
3. Reads the file, traces dependencies
4. Searches for "tool execution" to understand the full flow
5. Synthesizes explanation with code citations

**Output:**
```
The agent executes tasks through the `ExecuteTask` method in `pkg/agent/agent.go`.

Here's the flow:

1. **Task Reception** (lines 245-250):
   ```245:250:pkg/agent/agent.go
   func (a *Agent) ExecuteTask(ctx context.Context, task *a2a.Task) (*a2a.Task, error) {
       // Validates task and prepares for execution
   }
   ```

2. **Reasoning Loop** (lines 300-350):
   The agent uses a reasoning strategy (chain-of-thought or supervisor) to iteratively:
   - Generate responses with tool calls
   - Execute tool calls in parallel
   - Reflect on results
   - Continue until no more tool calls

3. **Tool Execution** (lines 400-450):
   Tools are executed through the tool registry...
```

### 2. Refactoring

**Prompt:**
```
"Refactor the ChainOfThoughtStrategy to use dependency injection for the LLM service"
```

**What happens:**
1. Creates todos:
   ```
   📋 Current Tasks:
     1. 🔄 Read current implementation
     2. ⏳ Design new interface
     3. ⏳ Implement changes
     4. ⏳ Update tests
   ```

2. Reads `pkg/reasoning/chain_of_thought_strategy.go`
3. Searches for "dependency injection patterns in Go"
4. Makes changes using `search_replace`
5. Runs tests with `execute_command`
6. Marks todos complete

### 3. Feature Addition

**Prompt:**
```
"Add support for rate limiting to the HTTP client"
```

**What happens:**
1. Explores existing HTTP client implementation
2. Searches for "rate limiting patterns"
3. Proposes design (token bucket vs. sliding window)
4. Implements with tests
5. Updates documentation

### 4. Debugging

**Prompt:**
```
"The test in pkg/agent/agent_test.go is failing. Fix it."
```

**What happens:**
1. Reads test file
2. Runs test: `go test ./pkg/agent -v`
3. Analyzes error output
4. Searches for related code
5. Proposes fix
6. Applies fix
7. Runs test again to verify

---

## Architecture Deep Dive

### How Chain-of-Thought Works

```
┌─────────────────────────────────────────────────────────────┐
│ USER: "Add tests for the auth module"                      │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ ITERATION 1: Claude generates response                      │
│ • Tool calls: [search("auth module tests")]                │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ Execute tools in parallel                                   │
│ ✅ search → Found auth.go, auth_test.go                    │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ REFLECTION: Analyze tool results                           │
│ • Confidence: 80%                                           │
│ • Recommendation: Continue (need to read files)            │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ ITERATION 2: Claude generates response                      │
│ • Tool calls: [write_file("auth_test.go", "...")]        │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ Execute tools                                               │
│ ✅ write_file → Created auth_test.go                      │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ ITERATION 3: Claude generates response                      │
│ • Tool calls: []                                            │
│ • Text: "I've added comprehensive tests..."               │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ DONE: No more tool calls, return to user                   │
└─────────────────────────────────────────────────────────────┘
```

**Key points:**
1. **Simple loop:** Generate → Execute → Reflect → Repeat
2. **Natural termination:** LLM decides when to stop (no tool calls)
3. **Parallel execution:** Independent tools run simultaneously
4. **Self-reflection:** Agent analyzes its own progress

### How Semantic Search Works

```
┌─────────────────────────────────────────────────────────────┐
│ CODEBASE INDEXING (one-time)                               │
│                                                             │
│ For each file:                                              │
│   1. Split into chunks (~500 tokens)                       │
│   2. Generate embeddings (vector representations)          │
│   3. Store in Qdrant with metadata                         │
│                                                             │
│ Result: Vector database of your entire codebase            │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ SEARCH QUERY: "authentication flow"                        │
│                                                             │
│   1. Generate query embedding                               │
│   2. Find nearest vectors (cosine similarity)              │
│   3. Return top K chunks with scores                       │
│   4. Format results for LLM                                │
│                                                             │
│ Results:                                                    │
│   • auth.go:50-80 (score: 0.92)                           │
│   • middleware.go:120-150 (score: 0.87)                   │
│   • user.go:200-230 (score: 0.81)                         │
└─────────────────────────────────────────────────────────────┘
```

**Why this is powerful:**
- **Semantic** not keyword matching
- **Fast:** O(log n) vector search
- **Context-aware:** Finds related code even with different names

---

## Customization

### Change the LLM

```yaml
llms:
  my_llm:
    # OpenAI
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    
    # Or Anthropic
    type: "anthropic"
    model: "claude-3-5-sonnet-20241022"
    api_key: "${ANTHROPIC_API_KEY}"
    
    # Or Gemini
    type: "gemini"
    model: "gemini-2.0-flash"
    api_key: "${GEMINI_API_KEY}"
```

### Customize Prompts

```yaml
prompt:
  prompt_slots:
    system_role: |
      You are a senior Go engineer at Google.
      You write idiomatic, performant, well-tested code.
    
    reasoning_instructions: |
      1. Always write benchmarks for performance-critical code
      2. Use table-driven tests
      3. Follow Go best practices from Effective Go
```

### Add Custom Tools

```yaml
tools:
  my_linter:
    type: "command"
    allowed_commands:
      - "golangci-lint"
      - "staticcheck"
    working_directory: "./"
```

### Adjust Iterations

```yaml
reasoning:
  max_iterations: 50  # Lower for faster responses
  # or
  max_iterations: 200  # Higher for complex tasks
```

### Optimize Performance

```yaml
llms:
  my_llm:
    temperature: 0.0        # Deterministic (enables caching)
    max_tokens: 4000        # Lower token limit for faster responses
    
    # Use lighter models for simple tasks
    model: "gpt-4o-mini"    # Faster and more efficient
```

---

## Troubleshooting

### Agent is slow

**Symptom:** Takes >1 minute to respond

**Fixes:**
1. **Reduce semantic search results:**
   ```yaml
   tools:
     search:
       default_limit: 5  # Down from 10
   ```

2. **Lower max iterations:**
   ```yaml
   reasoning:
     max_iterations: 50  # Down from 100
   ```

3. **Use faster LLM:**
   ```yaml
   llms:
     my_llm:
       model: "claude-3-haiku"  # 3x faster than Sonnet
   ```

### Agent makes mistakes

**Symptom:** Incorrect code or logic errors

**Fixes:**
1. **Enable structured reflection:**
   ```yaml
   reasoning:
     enable_structured_reflection: true
   ```

2. **Increase iterations:**
   ```yaml
   reasoning:
     max_iterations: 150
   ```

3. **Use better LLM:**
   ```yaml
   model: "claude-sonnet-4-20250514"  # vs. gpt-4o-mini
   ```

### Semantic search not working

**Symptom:** Agent can't find relevant code

**Fixes:**
1. **Restart server to re-index:**
   ```bash
   # Stop the server (Ctrl+C) and restart it
   hector serve --config my-cursor.yaml
   # Indexing happens automatically on startup
   ```

2. **Check Qdrant is running:**
   ```bash
   curl http://localhost:6334/health
   ```

3. **Verify Ollama and embeddings:**
   ```bash
   # Check Ollama is running
   curl http://localhost:11434/api/tags
   
   # Ensure nomic-embed-text is installed
   ollama list | grep nomic
   ```

4. **Check logs for indexing errors:**
   Look for errors during server startup related to document store initialization

---

## Next Steps

1. **Try it out:** Follow the [Getting Started](#getting-started) guide to set up your first agent
2. **Customize:** Experiment with different prompts, tools, and LLM configurations to match your workflow
3. **Test:** Try it on your real projects and evaluate the results
4. **Share:** Contribute your improved configs and learnings back to the community

---

## Community

- **GitHub:** [github.com/kadirpekel/hector](https://github.com/kadirpekel/hector)
- **Discussions:** Share your configs and improvements
- **Issues:** Report bugs or request features

---

## Conclusion

This tutorial demonstrated that AI coding assistants are built on well-understood components: effective prompting, chain-of-thought reasoning loops, and semantic search. By using declarative YAML configuration, you can build sophisticated AI systems without writing code.

**What you've learned:**
- How to configure an AI agent using only YAML
- The core components that make AI coding assistants effective
- How to customize prompts, tools, and reasoning strategies
- How to deploy and troubleshoot your agent

**Next steps to explore:**
- Experiment with different prompt strategies
- Try various LLM providers (OpenAI, Anthropic, Gemini)
- Add custom tools for your specific workflow
- Deploy on your own infrastructure

The complete working example is available at: [`configs/coding.yaml`](../../configs/coding.yaml)

---

## Important Note

**What we've built:** This tutorial demonstrates building a **Cursor-like coding assistant at its core** - the AI agent that reasons, searches code, and executes tasks. On the agent side, this is close to a production-ready system with proper chain-of-thought reasoning, semantic search, tool execution, and reflection capabilities.

**What Cursor offers beyond this:** Cursor is a much more sophisticated **complete IDE solution** that includes:
- **Native editor integration** - Deep integration with VS Code fork
- **Inline diff views** - Visual code change previews directly in the editor
- **Multi-file editing UI** - Seamless interface for reviewing changes across files
- **Checkpoint/rollback system** - Version control for AI-generated changes
- **Inline code completion** - Real-time suggestions as you type
- **Chat panel integration** - Built-in chat interface within the IDE
- **File tree awareness** - Context-aware file navigation and suggestions
- **Terminal integration** - Embedded terminal with AI context
- **Git integration** - Smart commit messages and change tracking
- **Collaborative editing** - Multi-cursor and pair programming features
- **Command palette** - Quick access to AI features via keyboard
- **Settings UI** - User-friendly configuration interface

**The distinction:** Building a powerful AI agent (what this tutorial covers) is different from building a polished IDE experience. The agent's reasoning capabilities, tool use, and code understanding can match professional systems, but the user experience and workflow integration require significant additional IDE engineering.

**This is valuable because:** Understanding how the AI agent works gives you full control over the intelligence layer, letting you customize reasoning strategies, add domain-specific tools, and deploy on your infrastructure - even if you access it through a terminal or API rather than a native IDE.

