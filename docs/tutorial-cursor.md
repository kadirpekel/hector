---
title: AI Coding Assistant Tutorial
description: Build a Cursor-like AI coding assistant with semantic search and chain-of-thought reasoning
---

# Build Your Own Cursor-Like AI Coding Assistant in Pure YAML

**TL;DR:** Modern AI coding assistants use well-crafted prompts, chain-of-thought reasoning, and semantic search to help developers write code. In this tutorial, you'll build a similar AI coding assistant using **only YAML configuration** - no code required.

---

## Why This Tutorial

With Hector, you can build a powerful AI coding assistant using **pure YAML configuration**—no programming required. This tutorial shows you how to create an intelligent agent with capabilities comparable to commercial coding assistants.

**What you'll build:**

- **Semantic code search** - Find code by meaning, not keywords
- **Chain-of-thought reasoning** - Iterative problem solving
- **Tool execution** - File operations, commands, tests, linters
- **Streaming responses** - Real-time output as the agent works
- **Full customization** - Every prompt, tool, and behavior under your control

**What you'll learn:**

- How AI coding assistants work under the hood
- The power of declarative configuration over imperative code
- How to customize and extend the agent for your workflow
- How to deploy on your own infrastructure

**The result:** A production-ready AI coding assistant you fully own and control, deployable anywhere, integrable with any workflow.

---

## Understanding AI Coding Assistants

AI coding assistants rely on three core components:

### 1. Effective Prompts

The system prompt establishes the agent as a "pair programmer" who takes action rather than just making suggestions. Key elements:

- Defines the agent's role and behavior
- Instructs to implement changes, not just suggest them
- Emphasizes thoroughness and self-sufficiency
- Guides tool usage patterns

### 2. Chain-of-Thought Reasoning

The agent iterates through a simple loop: generate response → execute tools → continue until no more tool calls needed. The LLM naturally determines when it has gathered enough information to complete the task.

### 3. Tool Execution

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

See the complete configuration in the Hector repository's `configs/coding.yaml` file.

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
   # IMPORTANT: Hector uses Qdrant's gRPC interface (port 6334)
   docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
   ```
   
   **Note:** Hector's Qdrant client uses the **gRPC protocol on port 6334**. Make sure both ports are exposed:
   - Port 6333: HTTP/REST API (for debugging)
   - Port 6334: gRPC API (required by Hector)
   
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
              ->
    ┌─────────────────┐
    │  LLM generates  │ ←──┐
    │  response +     │    │
    │  tool calls     │    │
    └─────────────────┘    │
              ->            │
    ┌─────────────────┐    │
    │  Execute tools  │    │
    │  in parallel    │    │
    └─────────────────┘    │
              ->            │
       More tools? ─────Yes─┘
              │
             No
              ->
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
              ->
┌─────────────────────────────────────┐
│ Convert to vector embedding         │
└─────────────────────────────────────┘
              ->
┌─────────────────────────────────────┐
│ Find similar vectors in database    │
└─────────────────────────────────────┘
              ->
┌─────────────────────────────────────┐
│ Return top matches:                 │
│  - auth.go:50-80 (95% match)       │
│  - middleware.go:120-150 (89%)     │
│  - user.go:200-230 (84%)           │
└─────────────────────────────────────┘
```

**Why This Works:**
- Finds code by **meaning**, not just keyword matching
- Discovers related concepts even with different naming
- Fast vector similarity search (sub-second queries)

---

## Customization

### Prompt Customization

You can customize the agent's behavior by modifying the prompt slots:

```yaml
prompt:
  prompt_slots:
    system_role: |
      You are a specialized Python coding assistant.
      Focus on clean, readable code and comprehensive testing.
    
    reasoning_instructions: |
      Always write tests before implementing features.
      Use type hints and follow PEP 8 guidelines.
      Prioritize readability over cleverness.
```

### Tool Configuration

Customize which tools the agent has access to:

```yaml
tools:
  - "search"           # Semantic search
  - "write_file"       # File operations
  - "search_replace"   # Precise edits
  - "execute_command"  # Command execution
  - "todo_write"       # Task management
  # Add custom tools here
```

### Reasoning Parameters

Adjust the reasoning behavior:

```yaml
reasoning:
  engine: "chain-of-thought"
  max_iterations: 50      # Reduce for faster responses
  enable_streaming: true  # Real-time output
```

---

## Troubleshooting

### Common Issues

**Agent doesn't find relevant code:**
- Ensure Qdrant is running on port 6334
- Check that the codebase was indexed successfully
- Verify the search query is specific enough

**Agent makes incorrect changes:**
- Review the system prompt for clarity
- Add more specific instructions in reasoning_instructions
- Consider reducing max_iterations for simpler tasks

**Performance issues:**
- Reduce max_iterations for faster responses
- Disable streaming if not needed
- Use smaller embedding models for faster search

### Debug Mode

Enable debug mode to see what the agent is doing:

```bash
hector serve --config configs/my-cursor.yaml --log-level debug
```

---

## Conclusion

With just ~50 lines of YAML configuration, you've built a capable AI coding assistant using chain-of-thought reasoning, semantic search, and tool execution. The agent can explore codebases, implement features, debug issues, and handle multi-file changes autonomously.

The complete working example is available in the Hector repository's `configs/coding.yaml` file.

---

## Important Note

**What You've Built:** This tutorial demonstrates building the **core AI agent**—the intelligent reasoning engine that powers coding assistants. With Hector's pure YAML configuration, you've created a production-ready agent with:

- **Chain-of-thought reasoning** - Iterative problem solving
- **Semantic code search** - Intelligent codebase exploration  
- **Tool execution** - File operations, commands, and more
- **Streaming responses** - Real-time output
- **Full customization** - Complete control over prompts and behavior

**About Complete Solutions:** Commercial products like Cursor combine a powerful AI agent with a polished IDE experience—native editor integration, inline diffs, visual change previews, and seamless workflows. That complete package offers significant value, especially if you prefer an all-in-one solution.

**Hector's Different Approach:** Instead of an integrated IDE, Hector gives you the intelligence layer as a flexible, standalone service. You can:
- Deploy anywhere (laptop, datacenter, cloud, air-gapped)
- Integrate with any workflow (terminal, API, web, custom IDE plugin)
- Customize every aspect (prompts, tools, reasoning strategies)
- Own and control your infrastructure

**The Choice:** If you want a polished, ready-to-use IDE with AI built in, Cursor and similar products are excellent. If you need flexibility, customization, self-hosting, or want to integrate AI into your own systems and workflows, Hector provides the core intelligence you need—without vendor lock-in.
