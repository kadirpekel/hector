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

With Hector, you can build a powerful AI coding assistant using **pure YAML configuration**—no programming required. This tutorial shows you how to create an intelligent agent with capabilities comparable to commercial coding assistants.

**What you'll build:**
- ✅ **Semantic code search** - Find code by meaning, not keywords
- ✅ **Chain-of-thought reasoning** - Iterative problem solving
- ✅ **Tool execution** - File operations, commands, tests, linters
- ✅ **Streaming responses** - Real-time output as the agent works
- ✅ **Full customization** - Every prompt, tool, and behavior under your control

**What you'll learn:**
- How AI coding assistants work under the hood
- The power of declarative configuration over imperative code
- How to customize and extend the agent for your workflow
- How to deploy on your own infrastructure

**The result:** A production-ready AI coding assistant you fully own and control, deployable anywhere, integrable with any workflow.

---

## Understanding AI Coding Assistants

AI coding assistants rely on three core components:

### 1. **Effective Prompts**

The system prompt establishes the agent as a "pair programmer" who takes action rather than just making suggestions. Key elements:
- Defines the agent's role and behavior
- Instructs to implement changes, not just suggest them
- Emphasizes thoroughness and self-sufficiency
- Guides tool usage patterns

### 2. **Chain-of-Thought Reasoning**

The agent iterates through a simple loop: generate response → execute tools → continue until no more tool calls needed. The LLM naturally determines when it has gathered enough information to complete the task.

### 3. **Tool Execution**

Essential capabilities that make the agent practical:
- **Semantic Search** - Find relevant code by meaning, not keywords
- **File Operations** - Read, write, and edit files precisely
- **Command Execution** - Run tests, linters, build tools
- **Parallel Execution** - Handle multiple operations simultaneously for speed

**The Power of Configuration:** With Hector, you get all these capabilities through pure YAML—no coding required. You define the prompts, configure the tools, and set the reasoning parameters declaratively.

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

Once your agent is running, try these example prompts to see it in action:

### Code Exploration
```
"How does the agent execute tasks in this codebase?"
```
The agent uses semantic search to find relevant code, reads files, traces dependencies, and synthesizes an explanation with code citations.

### Refactoring
```
"Refactor the ChainOfThoughtStrategy to use dependency injection"
```
Creates a task plan, reads the implementation, searches for patterns, makes changes using `search_replace`, and runs tests to verify.

### Feature Implementation
```
"Add rate limiting to the HTTP client"
```
Explores existing code, researches patterns, proposes design, implements with tests, and updates documentation.

### Debugging
```
"The test in pkg/agent/agent_test.go is failing. Fix it."
```
Reads the test, runs it to see the error, analyzes the issue, searches for related code, applies a fix, and re-runs to verify.

### Multi-File Changes
```
"Add logging to all HTTP endpoints"
```
Uses semantic search to find all endpoints, creates a task plan, updates files systematically, and ensures consistency.

**What You'll Notice:**
- Agent uses semantic search to understand your codebase
- Works autonomously through multi-step tasks
- Creates todos for complex changes
- Runs tests and linters automatically
- Provides clear explanations of what it's doing

---

## Architecture Deep Dive

### How Chain-of-Thought Works

The reasoning loop follows this pattern:

```
┌──────────────────────────────────────┐
│  User Request                        │
│  "Add tests for auth module"         │
└──────────────────────────────────────┘
              ↓
    ┌─────────────────┐
    │  LLM generates  │ ←──┐
    │  response +     │    │
    │  tool calls     │    │
    └─────────────────┘    │
              ↓            │
    ┌─────────────────┐    │
    │  Execute tools  │    │
    │  in parallel    │    │
    └─────────────────┘    │
              ↓            │
       More tools? ─────Yes─┘
              │
             No
              ↓
    ┌─────────────────┐
    │  Return result  │
    │  to user        │
    └─────────────────┘
```

**Key Points:**
- The LLM naturally determines when it has enough information
- Tools execute in parallel for speed
- The loop continues until the LLM stops requesting tools
- No manual intervention needed—the agent manages its own workflow

### How Semantic Search Works

**One-Time Setup (Indexing):**

```
┌─────────────────────────────────────┐
│  Codebase Indexing                  │
│                                     │
│  For each file:                     │
│    1. Split into chunks             │
│    2. Generate embeddings           │
│    3. Store in Qdrant               │
│                                     │
│  Result: Vector database            │
└─────────────────────────────────────┘
```

**At Query Time:**

```
┌─────────────────────────────────────┐
│ Query: "authentication flow"        │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│ Convert to vector embedding         │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│ Find similar vectors in database    │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│ Return top matches:                 │
│  • auth.go:50-80 (95% match)       │
│  • middleware.go:120-150 (89%)     │
│  • user.go:200-230 (84%)           │
└─────────────────────────────────────┘
```

**Why This Works:**
- Finds code by **meaning**, not just keyword matching
- Discovers related concepts even with different naming
- Fast vector similarity search (sub-second queries)

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

**What You've Built:** This tutorial demonstrates building the **core AI agent**—the intelligent reasoning engine that powers coding assistants. With Hector's pure YAML configuration, you've created a production-ready agent with:

- ✅ **Chain-of-thought reasoning** - Iterative problem solving
- ✅ **Semantic code search** - Intelligent codebase exploration  
- ✅ **Tool execution** - File operations, commands, and more
- ✅ **Streaming responses** - Real-time output
- ✅ **Full customization** - Complete control over prompts and behavior

**About Complete Solutions:** Commercial products like Cursor combine a powerful AI agent with a polished IDE experience—native editor integration, inline diffs, visual change previews, and seamless workflows. That complete package offers significant value, especially if you prefer an all-in-one solution.

**Hector's Different Approach:** Instead of an integrated IDE, Hector gives you the intelligence layer as a flexible, standalone service. You can:
- Deploy anywhere (laptop, datacenter, cloud, air-gapped)
- Integrate with any workflow (terminal, API, web, custom IDE plugin)
- Customize every aspect (prompts, tools, reasoning strategies)
- Own and control your infrastructure

**The Choice:** If you want a polished, ready-to-use IDE with AI built in, Cursor and similar products are excellent. If you need flexibility, customization, self-hosting, or want to integrate AI into your own systems and workflows, Hector provides the core intelligence you need—without vendor lock-in.

