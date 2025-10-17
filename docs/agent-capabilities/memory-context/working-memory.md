---
layout: default
title: Working Memory
nav_order: 1
parent: Memory & Context
description: "Session-scoped conversation context management"
---

# Working Memory - Session Context Management 🧠

> **Smart conversation context management within sessions - automatic summarization and token management.**

---

## Overview

Working memory manages conversation history within the current session, providing intelligent context management that prevents token limit issues and maintains conversation flow.

**Key Features:**
- **Token-aware management** - Accurate token counting and budget control
- **Automatic summarization** - LLM condenses old messages for unlimited conversation length
- **Recency-based selection** - Most recent messages prioritized
- **Session-scoped** - Context cleared when session ends

## Why Working Memory Matters

Traditional AI agents lose context in long conversations:
- ❌ Exceed token limits without warning
- ❌ Truncate important messages
- ❌ Use inaccurate character-based estimates
- ❌ No automatic summarization

**Result:** Broken conversations, lost context, frustrated users.

## The Solution

One simple setting that changes everything:

```yaml
memory:
  budget: 2000
```

**That's it.** Your agent now has:
- ✅ **Accurate token counting** - 100% accurate (not estimates)
- ✅ **Recency-based selection** - Most recent messages that fit within budget
- ✅ **Automatic management** - No manual intervention
- ✅ **Optional summarization** - LLM condenses old messages for unlimited conversation length

## Understanding Context Windows

The **context window** is your LLM's maximum input size - the total tokens it can process in one request:

| Model | Context Window |
|-------|----------------|
| GPT-4o | 128K tokens |
| Claude 3.5 Sonnet | 200K tokens |
| Gemini 2.0 Flash | 1M tokens |

Your LLM's context window contains multiple parts:

```
┌─────────────────────────────────────────────┐
│              CONTEXT WINDOW                  │
├─────────────────────────────────────────────┤
│  System Prompt (500 tokens)                 │
│  + Conversation History (2000 tokens)      │
│  + Current Message (100 tokens)             │
│  + Response Space (1000 tokens)             │
│  = Total Used (3600 tokens)                 │
└─────────────────────────────────────────────┘
```

## Working Memory Strategies

Hector provides two strategies for managing conversation history:

### Strategy 1: Summary Buffer (Default - Recommended)

**How it works:**
1. **Recent messages** - Keep most recent messages in full
2. **Older messages** - Summarize when approaching token limit
3. **Automatic management** - No manual intervention needed

**Configuration:**
```yaml
memory:
  budget: 2000
  strategy: "summary_buffer"  # Default
```

**Benefits:**
- ✅ **Unlimited conversation length** - Summarization prevents token overflow
- ✅ **Context preservation** - Important information retained in summaries
- ✅ **Automatic management** - No manual intervention required
- ✅ **Optimal performance** - Balances context vs. token usage

### Strategy 2: Buffer Window

**How it works:**
1. **Fixed window** - Keep only the most recent N messages
2. **Simple truncation** - Older messages completely removed
3. **Predictable behavior** - Always uses same amount of tokens

**Configuration:**
```yaml
memory:
  budget: 2000
  strategy: "buffer_window"
```

**Benefits:**
- ✅ **Predictable token usage** - Always uses exactly the budget
- ✅ **Simple behavior** - Easy to understand and debug
- ✅ **Fast performance** - No LLM calls for summarization

**Limitations:**
- ❌ **Limited conversation length** - Context lost when window fills
- ❌ **No context preservation** - Older messages completely removed

## Strategy Comparison

| Feature | Summary Buffer | Buffer Window |
|---------|----------------|---------------|
| **Conversation Length** | Unlimited | Limited by window |
| **Context Preservation** | High (summaries) | Low (truncation) |
| **Token Usage** | Variable | Fixed |
| **Performance** | Moderate | Fast |
| **Complexity** | High | Low |

## Which Strategy Should I Use?

**Use Summary Buffer (default) when:**
- You need unlimited conversation length
- Context preservation is important
- You're building complex, long-running agents

**Use Buffer Window when:**
- You need predictable token usage
- Performance is critical
- Conversations are typically short

## Configuration Examples

### Basic Working Memory
```yaml
agents:
  support_agent:
    name: "Support Agent"
    llm: "gpt-4o"
    memory:
      budget: 2000
```

### Advanced Configuration
```yaml
agents:
  code_reviewer:
    name: "Code Reviewer"
    llm: "gpt-4o"
    memory:
      budget: 2000
      strategy: "summary_buffer"
      threshold: 0.8
      target: 0.6
```

### High-Performance Setup
```yaml
agents:
  research_agent:
    name: "Research Agent"
    llm: "gpt-4o"
    memory:
      budget: 1000
      strategy: "buffer_window"
```

## Token Counting

Hector uses **accurate token counting** (not character estimates):

- **GPT models** - Uses tiktoken library
- **Claude models** - Uses Anthropic's tokenizer
- **Other models** - Uses appropriate tokenizer

**Why this matters:**
- Character-based estimates can be 2-3x inaccurate
- Token limits are hard limits (requests fail if exceeded)
- Accurate counting prevents unexpected truncation

## Examples

### Example 1: Customer Support Bot
```yaml
agents:
  support_agent:
    name: "Customer Support"
    llm: "gpt-4o"
    memory:
      budget: 1500
      strategy: "summary_buffer"
```

**Result:** Bot maintains context throughout long support conversations while staying within token limits.

### Example 2: Code Review Assistant
```yaml
agents:
  code_reviewer:
    name: "Code Reviewer"
    llm: "gpt-4o"
    memory:
      budget: 3000
      strategy: "summary_buffer"
```

**Result:** Assistant remembers previous feedback and suggestions across long code review sessions.

### Example 3: Research Agent
```yaml
agents:
  research_agent:
    name: "Research Agent"
    llm: "gpt-4o"
    memory:
      budget: 2000
      strategy: "buffer_window"
```

**Result:** Fast, predictable performance for research tasks with limited context needs.

## See Also

- **[Long-term Memory](long-term-memory)** - Persistent knowledge storage
- **[Memory Configuration](memory-configuration)** - Advanced tuning
- **[Memory Management](../memory-context)** - Overview of all memory features