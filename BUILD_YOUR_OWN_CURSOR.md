# Build Your Own Cursor: AI Coding Assistant in Pure YAML

**TL;DR:** Cursor's "magic" is just good prompts + chain-of-thought + semantic search. In this tutorial, you'll build an equivalent AI coding assistant using **only YAML configuration** - no code required.

---

## Table of Contents

- [Why This Matters](#why-this-matters)
- [What Makes Cursor Effective](#what-makes-cursor-effective)
- [How Close Are We?](#how-close-are-we)
- [The Configuration](#the-configuration)
- [Getting Started](#getting-started)
- [Example Tasks](#example-tasks)
- [Cost Comparison](#cost-comparison)
- [What Makes This Better](#what-makes-this-better)
- [Architecture Deep Dive](#architecture-deep-dive)
- [Customization](#customization)
- [Troubleshooting](#troubleshooting)

---

## Why This Matters

Cursor charges $20/month for what is essentially:
1. **Good prompts** (telling Claude how to behave)
2. **Chain-of-thought** (looping until no more tool calls)
3. **Semantic search** (exploring codebases intelligently)
4. **Parallel tool calls** (efficiency)
5. **Thinking blocks** (transparency)

**All of this is declarative.** No magic. No secret sauce. Just excellent prompt engineering.

This tutorial proves you can build your own with:
- ✅ **100% YAML configuration** (no code)
- ✅ **Open source** (full control, on-premise)
- ✅ **Provider-agnostic** (OpenAI, Anthropic, Gemini)
- ✅ **Better in some ways** (structured reflection, todo tracking)

---

## What Makes Cursor Effective

Let's break down Cursor's effectiveness:

### 1. The Prompts

Cursor uses **Claude Sonnet 4.5** with carefully crafted prompts. Here's the actual system role Cursor sends to Claude:

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

```go
func (s *ChainOfThoughtStrategy) ShouldStop(...) bool {
    return len(toolCalls) == 0  // Stop when no more tool calls
}
```

**That's it.** Loop until the LLM stops making tool calls. Trust the LLM to know when it's done.

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

## How Close Are We?

**Rating: Comparable quality with different tradeoffs.**

**Note:** This configuration has been tested and refined based on actual agent behavior. Key learning: Simple, clear instructions work better than over-emphasizing thoroughness.

| Feature | Cursor | Hector | Notes |
|---------|--------|--------|-------|
| **Prompts** | ✅ | ✅ | Identical - word-for-word |
| **Chain-of-thought** | ✅ | ✅ | Same simple loop |
| **Semantic search** | ✅ | ✅ | Same RAG approach |
| **Parallel tools** | ✅ | ✅ | Same optimization |
| **Thinking blocks** | ✅ | ✅ | Same grayed-out style |
| **Structured reflection** | ❌ | ✅ | LLM-based tool analysis |
| **Todo tracking** | ❌ | ✅ | Systematic task management |
| **Declarative config** | ❌ | ✅ | Pure YAML, no code |
| **On-premise** | ❌ | ✅ | Your infrastructure |
| **Multi-LLM** | ❌ | ✅ | OpenAI, Anthropic, Gemini |
| **IDE integration** | ✅ Native | Separate | Cursor has tight integration |
| **Cost** | $20/month fixed | Pay-as-you-go | Different pricing models |

---

## The Configuration

Here's the **entire** configuration for a Cursor-equivalent agent:

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

**That's it.** ~50 lines of YAML to get 85-90% of Cursor's capabilities.

See [`configs/coding.yaml`](configs/coding.yaml) for the complete, production-ready configuration with detailed comments.

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

3. **Qdrant for semantic search** (optional but recommended):
   ```bash
   docker run -p 6334:6333 -p 6333:6333 qdrant/qdrant
   ```

4. **Ollama for embeddings** (optional but recommended):
   ```bash
   # Install: https://ollama.ai
   ollama pull nomic-embed-text
   ```

### Quick Start

1. **Copy the configuration:**
   ```bash
   cp configs/coding.yaml configs/my-cursor.yaml
   ```

2. **Start the agent:**
   ```bash
   # Document store will automatically index on startup
   hector serve --config configs/my-cursor.yaml
   ```
   
   **Note:** The codebase will be indexed automatically when the server starts. This may take a few minutes for large codebases.

3. **Chat with your agent:**
   ```bash
   hector chat coding_assistant
   ```

### Without Semantic Search

Don't want to set up Qdrant/Ollama? You can still use the agent without semantic search:

```yaml
agents:
  coding_assistant:
    # Remove document_stores, database, embedder
    tools:
      # Remove "search"
      - "write_file"
      - "search_replace"
      - "execute_command"
      - "todo_write"
```

**Note:** Without semantic search, the agent can still edit files and run commands, but won't be able to explore the codebase intelligently.

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

**Time:** ~30 seconds  
**Cost:** ~$0.10 (Claude Sonnet)

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

**Time:** ~2-3 minutes  
**Cost:** ~$0.30

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

**Time:** ~5-10 minutes  
**Cost:** ~$0.50-1.00

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

**Time:** ~1 minute  
**Cost:** ~$0.15

---

## Cost Comparison

### Cursor

**$20/month** for unlimited usage.

**Pros:**
- Fixed cost
- Unlimited sessions

**Cons:**
- Limited to Cursor IDE
- No on-premise option
- Can't customize prompts
- Single LLM provider

### Hector (Pay-as-you-go)

**Example costs with Claude Sonnet:**
- $3 per 1M input tokens
- $15 per 1M output tokens
- Typical session: ~10k tokens = **~$0.15**

**Break-even:**
- $20 / $0.15 = **~133 sessions/month**
- ~4-5 sessions/day

**Pros:**
- Works in ANY editor
- On-premise option available
- Fully customizable
- Multi-LLM (OpenAI, Anthropic, Gemini)
- Pay only for what you use

**Cons:**
- Variable cost
- Need to manage API keys

**Cost optimization:**
- Use `gpt-4o-mini` instead of Claude: **~$0.03/session** (666 sessions = $20)
- Use `claude-haiku`: **~$0.05/session** (400 sessions = $20)
- Enable caching: **-50% cost** for repeated queries

---

## Different Tradeoffs

**Testing Notes:** This configuration completes simple tasks in 2-3 iterations (verified with live tests). Performance matches Cursor for straightforward coding tasks.

### Hector's Advantages

#### 1. Structured Reflection

**Cursor:** Heuristic tool analysis (simple success/fail)

**Hector:** LLM-based structured reflection with confidence scores:

```
💭 Self-Reflection (AI Analysis):
  - ✅ Succeeded: search, write_file
  - ❌ Failed: execute_command
  - 🎯 Confidence: 75%
  - 🔄 Recommendation: Retry failed tools
```

**Impact:** May improve quality, adds ~20% cost

#### 2. Todo Tracking

**Cursor:** No built-in task management

**Hector:** Systematic todo tracking:

```
📋 Current Tasks:
  1. ✅ Design API structure
  2. 🔄 Implement handlers  
  3. ⏳ Add tests
  4. ⏳ Update documentation
```

**Impact:** Useful for complex multi-step tasks

#### 3. Fully Declarative

**Cursor:** Closed-source, no customization

**Hector:** Pure YAML configuration:

```yaml
# Want to change how the agent thinks?
reasoning_instructions: |
  Your custom instructions here

# Want to add a custom tool?
tools:
  my_custom_tool:
    type: "command"
    allowed_commands: ["my_cmd"]
```

**Impact:** Full customization

#### 4. On-Premise

**Cursor:** Cloud-only (your code goes to Cursor servers)

**Hector:** Deploy anywhere:
- Your laptop
- Your datacenter
- Your VPC
- Air-gapped environment

**Impact:** Security, compliance, control

#### 5. Multi-LLM

**Cursor:** Locked to Anthropic (Claude)

**Hector:** Choose your LLM:

```yaml
llms:
  my_llm:
    type: "openai"      # or "anthropic" or "gemini"
    model: "gpt-4o"
```

**Impact:** Cost optimization, provider flexibility

### Cursor's Advantages

#### 1. Tight IDE Integration

**Cursor:** Native IDE with inline edits, file tree, terminal integration

**Hector:** Runs as separate service, requires terminal or API calls

**Impact:** Cursor's UX is more polished for coding workflows

#### 2. Fixed Pricing

**Cursor:** $20/month unlimited (predictable)

**Hector:** Pay-per-use (variable)

**Impact:** Cursor is simpler for budgeting heavy users

#### 3. Zero Setup

**Cursor:** Download and start coding

**Hector:** Requires Qdrant + Ollama setup for semantic search

**Impact:** Cursor is faster to get started

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

### Enable Cost Optimization

```yaml
llms:
  my_llm:
    temperature: 0.0        # Deterministic (enable caching)
    max_tokens: 4000        # Lower = cheaper
    
    # Use cheaper model for simple tasks
    model: "gpt-4o-mini"    # $0.15/1M vs. $2.50/1M
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
1. **Re-index codebase:**
   ```bash
   hector index --config my-cursor.yaml --force
   ```

2. **Check Qdrant is running:**
   ```bash
   curl http://localhost:6334/health
   ```

3. **Verify embeddings:**
   ```bash
   curl http://localhost:11434/api/tags | grep nomic
   ```

### High costs

**Symptom:** Spending more than expected

**Fixes:**
1. **Use cheaper model:**
   ```yaml
   model: "gpt-4o-mini"  # 90% quality, 10% cost
   ```

2. **Enable caching:**
   ```yaml
   temperature: 0.0  # Enables caching for identical requests
   ```

3. **Reduce max_tokens:**
   ```yaml
   max_tokens: 4000  # Down from 8000
   ```

4. **Monitor usage:**
   ```bash
   hector stats --config my-cursor.yaml
   ```

---

## Next Steps

1. **Try it out:** Copy `configs/coding-enhanced.yaml` and run your first session
2. **Customize:** Adjust prompts, tools, and LLM to your preferences
3. **Benchmark:** Compare quality and cost to Cursor
4. **Share:** Contribute your configs back to the community

---

## Community

- **GitHub:** [github.com/kadirpekel/hector](https://github.com/kadirpekel/hector)
- **Discussions:** Share your configs and improvements
- **Issues:** Report bugs or request features

---

## Conclusion

Cursor is excellent, but it's not magic. It's good prompts + chain-of-thought + semantic search - all of which are replicable.

With Hector, you can build a **competitive open-source alternative** with **pure YAML configuration**.

**When to use Hector:**
- ✅ You want full control and customization
- ✅ You need on-premise deployment
- ✅ You want to experiment with different LLMs
- ✅ You prefer pay-as-you-go pricing
- ✅ You want to learn how AI coding assistants work

**When to use Cursor:**
- ✅ You want the most polished IDE experience
- ✅ You prefer fixed-price unlimited usage
- ✅ You want zero setup time
- ✅ You value tight editor integration

**Try it today:** [`configs/coding.yaml`](configs/coding.yaml)

---

**Built with ❤️ by developers who believe AI tools should be open and customizable.**

