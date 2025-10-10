---
layout: default
title: Memory Configuration
nav_order: 2
parent: Advanced
description: "Memory strategy configuration guide and tuning options"
---

# Memory Configuration Guide

Configure Hector's pluggable memory management system.

---

## 🚀 Quick Start (Most Users)

```yaml
agents:
  my-assistant:
    llm: gpt4o
    memory:
      strategy: "summary_buffer"  # Optional, this is default
      budget: 2000                # Optional, defaults to 2000
    prompt:
      include_history: true
```

**Done!** This enables:
- ✅ Accurate token counting (never exceed limits)
- ✅ Recency-based message selection (most recent messages preserved)
- ✅ ~2000 tokens (~50 messages) of history
- ✅ Automatic summarization when threshold is reached

---

## 📋 Working Memory Strategies

Hector supports two pluggable **working memory strategies** for managing conversation history within a session:

### Summary Buffer (Default - Recommended)

Token-based with threshold-triggered summarization. Best for production.

```yaml
memory:
  strategy: "summary_buffer"  # This is the DEFAULT
  budget: 2000      # Optional, defaults to 2000
  threshold: 0.8    # Optional, defaults to 0.8 (80%)
  target: 0.6       # Optional, defaults to 0.6 (60%)
```

**Parameters:**
- **`budget`** - Maximum tokens for conversation history (default: 2000)
- **`threshold`** - Percentage of budget to trigger summarization (default: 0.8)
- **`target`** - Percentage of budget after summarization (default: 0.6)

**How it works:**
1. Accumulates messages until 80% of budget (1600 tokens)
2. Notifies user: "💭 Summarizing conversation history..."
3. Summarizes oldest messages via LLM (blocking, 2-5 seconds)
4. Keeps minimum 3 recent messages for context
5. Compresses to 60% of budget (1200 tokens)
6. Leaves 800 tokens breathing room
7. Repeats when threshold hit again

**Best for:** Production applications (90% of users), long conversations, optimal token efficiency

### Buffer Window

Simple LIFO, keeps last N messages. Best for testing.

```yaml
memory:
  strategy: "buffer_window"
  window_size: 20   # Optional, defaults to 20
```

**Parameters:**
- **`window_size`** - Number of messages to keep (default: 20)

**How it works:**
1. Keeps last 20 messages (LIFO)
2. Drops oldest when new arrives
3. No LLM calls, no summarization

**Best for:** Simple chatbots, testing/development, short conversations

---

## ⚙️ Common Adjustments

### Adjust Memory Size (Summary Buffer)

```yaml
memory:
  strategy: "summary_buffer"
  budget: 3000          # Increase for longer conversations
```

**Recommendations:**
- `1000` - Quick conversations, faster responses
- `2000` - **Default (recommended)**
- `3000` - Longer conversations
- `4000` - Extended context (may be slower)

### Tune Summarization Thresholds

For **very long conversations** (100+ messages), tune when summarization triggers:

```yaml
memory:
  strategy: "summary_buffer"
  budget: 2000
  threshold: 0.75      # Trigger earlier (at 75% = 1500 tokens)
  target: 0.5          # More aggressive compression (to 50% = 1000 tokens)
```

⚠️ **Note:** Summarization uses additional LLM calls (costs tokens)

---

## 📊 Configuration Reference

### Working Memory Settings

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `strategy` | string | `"summary_buffer"` | Strategy type: `"summary_buffer"` or `"buffer_window"` |
| `budget` | int | `2000` | Token budget for conversation history (summary_buffer) |
| `threshold` | float | `0.8` | Trigger summarization at % of budget (summary_buffer) |
| `target` | float | `0.6` | Compress to % of budget after summarization (summary_buffer) |
| `window_size` | int | `20` | Number of messages to keep (buffer_window) |

### Prompt Settings

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `include_history` | bool | `false` | Include conversation history in prompts |

---

## 📖 Complete Examples

### 1. Simple Assistant (Most Users)

```yaml
agents:
  assistant:
    llm: gpt4o
    memory:
      # Uses all defaults: strategy="summary_buffer", budget=2000
    prompt:
      include_history: true
```

**Use for:** Most conversations, general assistance, production applications

### 2. Extended Conversations

```yaml
agents:
  conversational-assistant:
    llm: gpt4o
    memory:
      strategy: "summary_buffer"
      budget: 3000              # Larger context window
    prompt:
      include_history: true
```

**Use for:** Longer conversations, detailed discussions, complex topics

### 3. Simple Testing Bot

```yaml
agents:
  test-bot:
    llm: gpt4o
    memory:
      strategy: "buffer_window"
      window_size: 15           # Keep last 15 messages
    prompt:
      include_history: true
```

**Use for:** Testing, development, short conversations

### 4. Custom Tuned Assistant

```yaml
agents:
  custom-assistant:
    llm: gpt4o
    memory:
      strategy: "summary_buffer"
      budget: 2500
      threshold: 0.75           # Trigger earlier
      target: 0.5               # More aggressive compression
    prompt:
      include_history: true
```

**Use for:** Fine-tuned balance between context and performance

---

## 🎯 Decision Guide

**Choose your strategy based on usage:**

### Short Conversations (< 20 messages)
```yaml
memory:
  strategy: "buffer_window"
  window_size: 20
```
- Fast, no LLM overhead
- Predictable behavior
- No summarization needed

### Normal Conversations (20-50 messages) ⭐ **Most Common**
```yaml
memory:
  # Uses defaults: summary_buffer with budget=2000
```
- Optimal token efficiency
- Automatic summarization if needed
- Production-ready

### Long Conversations (50-100 messages)
```yaml
memory:
  strategy: "summary_buffer"
  budget: 3000              # Increase budget
```
- More context preserved
- Less frequent summarization
- Better for complex discussions

### Very Long Conversations (100+ messages)
```yaml
memory:
  strategy: "summary_buffer"
  budget: 3000
  threshold: 0.75           # Summarize earlier
  target: 0.5               # More compression
```
- Aggressive memory management
- Handles unlimited length
- Preserves key context

---

## 🏗️ Architecture

### Layered Memory System

```
MemoryService (pkg/memory/)
├─ Manages sessions (lifecycle, isolation)
└─ Delegates to: WorkingMemoryStrategy
    ├─ SummaryBufferStrategy (token-based with summarization)
    └─ BufferWindowStrategy (simple LIFO)
```

**Benefits:**
- ✅ Clean separation: Service manages infrastructure, strategies implement algorithms
- ✅ No duplication: Session management in one place
- ✅ Testable: Each layer tested independently
- ✅ Extensible: Easy to add long-term memory strategies

### How Strategies Work

**Summary Buffer:**
1. Tracks token count for session
2. When threshold hit (80%), triggers summarization
3. LLM condenses oldest messages
4. Keeps recent messages intact
5. Injects summary as system message

**Buffer Window:**
1. Maintains LIFO queue
2. Adds new messages to end
3. Drops oldest when size exceeded
4. Simple, fast, predictable

---

## 💡 Best Practices

### ✅ Do

- **Start with defaults:** `summary_buffer` with `budget: 2000` works for 90% of cases
- **Adjust budget if needed:** Increase for longer conversations, decrease for quick chats
- **Use `buffer_window` for testing:** Simple, fast, predictable for development
- **Monitor logs:** Watch for summarization triggers and token usage

### ❌ Don't

- **Over-configure:** The defaults are carefully balanced
- **Set budget too low:** Below 1000 tokens may lose too much context
- **Set budget too high:** Above 4000 may slow down responses
- **Use `buffer_window` in production:** `summary_buffer` is better for most cases

---

## 🐛 Troubleshooting

### "Context too short"
**Problem:** Not enough history retained  
**Solution:** Increase budget
```yaml
memory:
  budget: 3000  # or 4000
```

### "Responses are slow"
**Problem:** Too much context processing  
**Solution:** Decrease budget
```yaml
memory:
  budget: 1500
```

### "Old messages being dropped"
**Problem:** Older messages not preserved  
**Solution:** This is expected with recency-based selection! For long conversations, `summary_buffer` will automatically summarize old messages to preserve context.

### "Summarization happening too often"
**Problem:** Frequent summarization triggers  
**Solution:** Increase threshold or budget
```yaml
memory:
  budget: 3000        # Larger window
  threshold: 0.85     # Trigger later
```

---

## 🏃 Performance

### Token Counting Overhead
- **Cost:** <1ms per message
- **Caching:** Tiktoken encoding cached
- **Impact:** Negligible

### Summarization Overhead
- **Cost:** 2-5 seconds (blocks user, intentional)
- **Frequency:** Only when threshold exceeded
- **Token cost:** ~200-500 tokens per summarization
- **Savings:** 30-50% overall token reduction

### Memory Usage
- **Token counter:** ~5MB (encoding cache)
- **History buffer:** ~1KB per message
- **Total:** ~5-10MB for typical usage

---

## 📊 Comparison

| Feature | Buffer Window | Summary Buffer (Default) |
|---------|---------------|--------------------------|
| **Token Efficiency** | Fixed count | Optimal |
| **Max Conversation** | ~20 messages | Unlimited |
| **LLM Overhead** | No | Yes (summarization) |
| **Blocking** | No | Yes (2-5s on trigger) |
| **Complexity** | Low | Medium |
| **Best For** | Testing (10%) | Production (90%) |

---

## 🎓 Summary

**Most users (90%):**
```yaml
memory:
  # Uses defaults: summary_buffer, budget=2000
prompt:
  include_history: true
```

**Extended conversations:**
```yaml
memory:
  strategy: "summary_buffer"
  budget: 3000
prompt:
  include_history: true
```

**Testing/Development:**
```yaml
memory:
  strategy: "buffer_window"
  window_size: 20
prompt:
  include_history: true
```

That's it! Simple, balanced, production-ready. 🎉

---

**See also:**
- Main guide: [Memory Management](MEMORY.md)
- Example configs: `configs/memory-strategies-example.yaml`
- [Long-Term Memory Configuration](#long-term-memory-configuration) - Session-scoped persistent memory with vector storage

