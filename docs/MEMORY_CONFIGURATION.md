# Memory Configuration Guide

Simple guide for configuring Hector's memory management system.

---

## 🚀 Quick Start (Most Users)

```yaml
agents:
  - name: my-assistant
    llm: gpt4o
    prompt:
      smart_memory: true        # That's it!
      include_history: true
```

**Done!** This enables:
- ✅ Accurate token counting (never exceed limits)
- ✅ Intelligent message selection (keeps important context)
- ✅ ~2000 tokens (~50 messages) of history

---

## ⚙️ Common Adjustments

### Adjust Memory Size

```yaml
prompt:
  smart_memory: true
  memory_budget: 3000          # Increase for longer conversations
```

**Recommendations:**
- `1000` - Quick conversations, faster responses
- `2000` - **Default (recommended)**
- `3000` - Longer conversations
- `4000` - Extended context (may be slower)

### Add Automatic Summarization

For **very long conversations** (100+ messages):

```yaml
prompt:
  smart_memory: true
  enable_summarization: true   # Auto-summarize old messages
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
   
2. **Smart Selection**
   - Keeps important messages (system, errors, decisions)
   - Preserves recent context
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

### "Important messages lost"
**Problem:** Smart selection not keeping key context  
**Solution:** Already works! System messages, errors, and tool calls are automatically preserved

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
| Message selection | Last N messages | Smart (important + recent) |
| Long conversations | Truncates | Summarizes (optional) |
| Setup complexity | None | One setting: `smart_memory: true` |
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
- Complete implementation: `docs/IMMEDIATE_IMPROVEMENTS_COMPLETED.md`
- Example config: `configs/smart-memory-simple.yaml`
- Advanced features: `docs/VECTOR_MEMORY_DESIGN.md` (optional)

