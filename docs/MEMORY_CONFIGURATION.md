---
layout: default
title: Memory Configuration
nav_order: 2
parent: Advanced
description: "Advanced memory configuration options and tuning guide"
---

# Memory Configuration Guide

Simple guide for configuring Hector's memory management system.

---

## 🚀 Quick Start (Most Users)

```yaml
agents:
  - name: my-assistant
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

## 📋 History Strategies

Hector supports two pluggable history management strategies:

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
2. Summarizes oldest messages via LLM (blocking, 2-5 seconds)
3. Compresses to 60% of budget (1200 tokens)
4. Leaves 800 tokens breathing room
5. Repeats when threshold hit again

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

## 📊 What Each Setting Does

### Core Options (Balanced for Most Cases)

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `smart_memory` | bool | `false` | **Main switch** - enables all improvements |
| `memory_budget` | int | `2000` | Token budget for conversation history |
| `enable_summarization` | bool | `false` | Auto-summarize when approaching limit |

### Existing Options (Still Work)

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `include_history` | bool | `false` | Include conversation history in prompts |
| `max_history_messages` | int | `10` | Fallback limit (when smart_memory is off) |

### Advanced Options (Power Users Only)

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `use_token_counting` | bool | `false` | Manual token counting (use `smart_memory` instead) |
| `max_tokens` | int | `0` | Manual token limit (use `memory_budget` instead) |
| `token_counting_model` | string | auto | Override token counting model |

---

## 📖 Complete Examples

### 1. Simple Assistant (Recommended)

```yaml
agents:
  - name: assistant
    llm: gpt4o
    prompt:
      smart_memory: true
      include_history: true
```

**Use for:** Most conversations, general assistance

### 2. Extended Conversations

```yaml
agents:
  - name: conversational-assistant
    llm: gpt4o
    prompt:
      smart_memory: true
      memory_budget: 3000
      include_history: true
```

**Use for:** Longer conversations, detailed discussions

### 3. Very Long Sessions

```yaml
agents:
  - name: long-session-assistant
    llm: gpt4o
    prompt:
      smart_memory: true
      memory_budget: 3000
      enable_summarization: true
      include_history: true
```

**Use for:** Extended sessions (100+ messages), ongoing projects

---

## 🎯 Decision Guide

**Choose your settings based on usage:**

### Short Conversations (5-20 messages)
```yaml
prompt:
  smart_memory: true
  memory_budget: 1000
```

### Normal Conversations (20-50 messages) ⭐ **Most Common**
```yaml
prompt:
  smart_memory: true          # Uses default budget: 2000
  # or
  memory_budget: 2000         # Explicit
```

### Long Conversations (50-100 messages)
```yaml
prompt:
  smart_memory: true
  memory_budget: 3000
```

### Very Long Conversations (100+ messages)
```yaml
prompt:
  smart_memory: true
  memory_budget: 3000
  enable_summarization: true
```

---

## 🔍 How It Works

### With `smart_memory: false` (default, backward compatible)
- Uses simple character-based estimation
- Keeps last N messages (max_history_messages)
- May exceed token limits
- Fast but less accurate

### With `smart_memory: true` ✨
1. **Accurate Token Counting**
   - Uses tiktoken for precise counting
   - Never exceeds your LLM's context window
   
2. **Recency-Based Selection**
   - Keeps most recent messages that fit within budget
   - Simple and fast (no complex scoring)
   - Fits within token budget

3. **Optional Summarization**
   - Automatically triggers at 80% capacity
   - Summarizes old messages
   - Keeps recent messages intact

---

## 💡 Best Practices

### ✅ Do

- **Start simple:** Just set `smart_memory: true`
- **Adjust budget:** Only if conversations are too short/long
- **Enable summarization:** For sessions with 100+ messages
- **Monitor logs:** Use `show_debug_info: true` to see token usage

### ❌ Don't

- **Over-configure:** The defaults work well for 90% of cases
- **Set budget too low:** Below 1000 tokens may lose too much context
- **Set budget too high:** Above 4000 may slow down responses
- **Enable summarization unnecessarily:** It adds LLM calls/cost

---

## 🐛 Troubleshooting

### "Context too short"
**Problem:** Not enough history retained  
**Solution:** Increase `memory_budget`
```yaml
memory_budget: 3000  # or 4000
```

### "Responses are slow"
**Problem:** Too much context  
**Solution:** Decrease `memory_budget`
```yaml
memory_budget: 1500
```

### "Old messages being dropped"
**Problem:** Older messages not preserved  
**Solution:** This is expected! Recency-based selection keeps most recent messages. For long conversations, enable summarization to preserve old context in summary form.

### "Running out of tokens"
**Problem:** Very long conversation  
**Solution:** Enable summarization
```yaml
enable_summarization: true
```

---

## 🔧 Advanced Usage

### Override Token Counting Model

```yaml
prompt:
  smart_memory: true
  token_counting_model: gpt-3.5-turbo  # Use different model for counting
```

**When to use:** Rarely needed - auto-detection works for most cases

### Manual Control (Not Recommended)

```yaml
prompt:
  use_token_counting: true
  max_tokens: 2500
  enable_summarization: true
```

**When to use:** You need very specific control (use `smart_memory` instead)

---

## 📊 Comparison

| Feature | Default (No Memory Management) | With Memory Management |
|---------|---------------------------|-------------------|
| Token counting | Character estimate (~25% error) | Accurate (0% error) |
| Context limit | May exceed | Never exceeds |
| Message selection | Last N messages | Recency-based (most recent preserved) |
| Long conversations | Truncates | Summarizes (optional, blocking) |
| Setup complexity | None | One setting: `memory.budget` |
| Performance | Fastest | Fast (minimal overhead) |

---

## 🎓 Summary

**For 90% of users:**
```yaml
prompt:
  smart_memory: true
  include_history: true
```

**For longer conversations:**
```yaml
prompt:
  smart_memory: true
  memory_budget: 3000
  include_history: true
```

**For very long sessions:**
```yaml
prompt:
  smart_memory: true
  memory_budget: 3000
  enable_summarization: true
  include_history: true
```

That's it! Simple, balanced, covers most use cases. 🎉

---

**See also:**
- Complete implementation: `docs/MEMORY_CONFIGURATION.md`
- Example config: `configs/smart-memory-simple.yaml`
- Advanced features: `docs/VECTOR_MEMORY_DESIGN.md` (optional)

