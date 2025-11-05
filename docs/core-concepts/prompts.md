# Prompts

## Overview

Hector uses a flexible prompt system that balances strategy-specific behavior with user customization. The system provides multiple ways to configure agent prompts, from quick CLI flags to detailed YAML configurations.

## Prompt Architecture

### Three-Slot System

Prompts are composed of three slots that serve distinct purposes:

```yaml
prompt_slots:
  system_role: ""     # WHO: Agent identity and core mission
  instructions: ""    # HOW: Behavioral guidance and patterns
  user_guidance: ""   # WHAT: User-specific instructions
```

#### 1. SystemRole (WHO)
Defines the agent's identity, purpose, and core mission.

**Example:**
```yaml
system_role: "You are an AI assistant helping users solve problems and accomplish tasks."
```

#### 2. Instructions (HOW)
Contains all behavioral guidance:
- Execution principles
- Workflow patterns
- Tool usage guidelines
- Communication style
- Task management rules

**Example:**
```yaml
instructions: |
  EXECUTION PRINCIPLES:
  - Use tools to accomplish tasks
  - Provide clear status updates
  - Keep summaries concise
```

#### 3. UserGuidance (WHAT)
User-provided custom instructions that override or augment strategy defaults.

**Example:**
```yaml
user_guidance: "Always respond in French and use technical terminology."
```

---

## Configuration Priority

The prompt system follows a clear priority hierarchy (highest to lowest):

### 1. System Prompt (HIGHEST - Complete Override)

When `system_prompt` is set, it **completely replaces** all prompt slots and strategy defaults.

```yaml
agents:
  calculator:
    prompt:
      system_prompt: "You are a calculator. Only respond with numeric results."
```

**Use cases:**
- Complete control over LLM behavior
- Testing specific prompt variations
- Specialized agents with unique requirements

**⚠️ Warning:** Using `system_prompt` disables all strategy-specific behavior (tool usage patterns, workflow guidance, etc.)

### 2. Prompt Slots (MEDIUM - Merges with Strategy)

When `prompt_slots` is set, it **merges** with strategy defaults. User values override strategy values for the same slot.

```yaml
agents:
  assistant:
    prompt:
      prompt_slots:
        system_role: "You are a helpful coding assistant"
        user_guidance: "Always explain your reasoning"
```

**Merge behavior:**
- If you provide `system_role`, it replaces the strategy's `system_role`
- If you provide `instructions`, it replaces the strategy's `instructions`
- If you provide `user_guidance`, it's added (strategies never set this)
- Empty slots use strategy defaults

### 3. Strategy Defaults (LOWEST - Base)

Each reasoning strategy defines default prompt slots optimized for its purpose.

**Chain of Thought Strategy:**
- Emphasizes tool execution and iterative reasoning
- Includes detailed workflow guidance
- Optimized for coding and problem-solving

**Supervisor Strategy:**
- Emphasizes orchestration and delegation
- Includes multi-agent coordination patterns
- Optimized for complex, multi-step tasks

---

## Usage Examples

### Quick Customization (CLI)

For quick testing or one-off customizations:

```bash
# Override role and add guidance
hector call "query" --role "You are a security expert" --instruction "Focus on vulnerabilities"

# Just add guidance to strategy defaults
hector call "query" --instruction "Be extremely concise"
```

**How it works:**
- `--role` → Sets `system_role` in prompt_slots
- `--instruction` → Sets `user_guidance` in prompt_slots
- Both merge with strategy defaults

### Configuration File (YAML)

For persistent customization:

#### Option 1: Augment Strategy (Recommended)

Merge your customizations with strategy defaults:

```yaml
agents:
  code_reviewer:
    name: "Code Reviewer"
    llm: gpt-4
    prompt:
      prompt_slots:
        system_role: "You are an expert code reviewer specializing in security"
        user_guidance: |
          Focus on:
          - SQL injection vulnerabilities
          - XSS attacks
          - Authentication flaws
    reasoning:
      engine: default  # Uses Chain of Thought strategy
```

**Result:** Your custom role + strategy's execution patterns + your guidance

#### Option 2: Complete Override

Replace everything with custom prompt:

```yaml
agents:
  simple_bot:
    name: "Simple Bot"
    llm: gpt-4
    prompt:
      system_prompt: |
        You are a simple Q&A bot.
        Answer questions directly and briefly.
        Do not use tools.
    reasoning:
      engine: default
```

**Result:** Only your system prompt, no strategy behavior

#### Option 3: Pure Strategy

Use strategy defaults without customization:

```yaml
agents:
  assistant:
    name: "Assistant"
    llm: gpt-4
    reasoning:
      engine: default
```

**Result:** Full strategy defaults, optimized behavior

---

## Zero-Config Mode

In zero-config mode (no YAML file), you can still customize:

```bash
# Use strategy defaults
hector call "query"

# Add custom role
hector call "query" --role "You are a data analyst"

# Add custom role and guidance
hector call "query" \
  --role "You are a friendly assistant" \
  --instruction "Use emojis and be conversational"
```

---

## Prompt Composition Order

The final prompt sent to the LLM is composed in this order:

```
┌─────────────────────────────────────────────────────────────┐
│ SYSTEM MESSAGE                                               │
├─────────────────────────────────────────────────────────────┤
│ 1. SystemRole (from strategy or config)                     │
│ 2. Instructions (from strategy or config)                   │
│ 3. [OPTIONAL] UserGuidance (from config or --instruction)   │
├─────────────────────────────────────────────────────────────┤
│ CONTEXT MESSAGE (if applicable)                             │
├─────────────────────────────────────────────────────────────┤
│ 4. [AUTO-INJECTED] Strategy Context (TODOs, etc)            │
├─────────────────────────────────────────────────────────────┤
│ RAG CONTEXT (if enabled)                                    │
├─────────────────────────────────────────────────────────────┤
│ 5. [AUTO-INJECTED] Document Context (semantic search)       │
├─────────────────────────────────────────────────────────────┤
│ CONVERSATION HISTORY                                         │
├─────────────────────────────────────────────────────────────┤
│ 6. [AUTO-INJECTED] Previous messages                        │
├─────────────────────────────────────────────────────────────┤
│ CURRENT QUERY                                                │
├─────────────────────────────────────────────────────────────┤
│ 7. Current user message                                     │
└─────────────────────────────────────────────────────────────┘
```

**Note:** If `system_prompt` is set, only step 1-3 are replaced with the custom prompt.

---

## Best Practices

### ✅ Do

1. **Use prompt_slots for customization** - Preserves strategy behavior while adding your requirements
2. **Keep system_role focused** - Define identity, not behavior
3. **Put behavior in instructions** - How the agent should work
4. **Use user_guidance for task-specific rules** - Context-specific requirements
5. **Test with strategy defaults first** - Strategies are already optimized

### ❌ Don't

1. **Don't use system_prompt unless necessary** - You lose all strategy optimizations
2. **Don't duplicate strategy behavior** - Instructions will be redundant
3. **Don't put too much in system_role** - It's meant to be brief
4. **Don't override instructions lightly** - Strategies have carefully crafted workflows

---

## Strategy-Specific Guidance

### Chain of Thought Strategy

**Optimized for:** Coding, debugging, problem-solving, research

**Default behavior:**
- Iterative tool execution
- TODO-based task management
- Semantic search for code exploration
- Detailed status updates
- Self-correction patterns

**Good customizations:**
```yaml
prompt_slots:
  user_guidance: |
    Specialize in Python and FastAPI
    Always write comprehensive tests
```

**Bad customizations:**
```yaml
prompt_slots:
  instructions: "Just answer questions briefly"  # Breaks tool execution!
```

### Supervisor Strategy

**Optimized for:** Multi-agent orchestration, complex workflows, delegation

**Default behavior:**
- Task decomposition
- Agent capability matching
- Result synthesis
- Parallel/sequential orchestration

**Good customizations:**
```yaml
prompt_slots:
  user_guidance: |
    Prioritize accuracy over speed
    Always consult the data-analyst agent for statistics
```

**Bad customizations:**
```yaml
prompt_slots:
  instructions: "Do everything yourself"  # Breaks delegation!
```

---

## Debugging Prompts

### View Composed Prompt

Enable debug mode to see the final prompt:

```bash
hector call "query" --debug -c config.yaml --agent myagent
```


```yaml
agents:
  myagent:
    reasoning:
```

### Common Issues

**Issue: Agent ignores my instructions**
- Solution: Check if you're using `system_prompt` (complete override) vs `prompt_slots` (merge)
- Solution: Ensure your guidance doesn't contradict strategy core behavior

**Issue: Agent doesn't use tools**
- Solution: Don't use `system_prompt` without including tool usage instructions
- Solution: Use `prompt_slots.user_guidance` instead of replacing `instructions`

**Issue: Too verbose/too concise**
- Solution: Add communication guidelines to `user_guidance`
```yaml
user_guidance: "Be extremely concise. No explanations unless asked."
```

---

## Advanced: RAG Integration

When document stores are configured, RAG context is automatically injected:

```yaml
agents:
  doc_assistant:
    llm: gpt-4
    document_stores:
      - technical_docs
    prompt:
      include_context: true  # Enable RAG injection
```

The context is injected AFTER the system message but BEFORE conversation history, providing relevant document chunks for the current query.

---

## Migration from Old Format

If you have old configs with deprecated fields:

**Old (deprecated):**
```yaml
prompt_slots:
  reasoning_instructions: "..."
  tool_usage: "..."
  output_format: "..."
  communication_style: "..."
  additional: "..."
```

**New (current):**
```yaml
prompt_slots:
  system_role: "WHO you are"
  instructions: "HOW you behave (merge all old fields here)"
  user_guidance: "WHAT user wants (was 'additional')"
```

**Note:** tool_usage with hardcoded tool lists is no longer needed - tools are provided via native function calling.

---

## See Also

- [Reasoning Strategies](reasoning.md) - Deep dive into strategy-specific behaviors
- [Configuration Reference](../reference/configuration.md) - Complete config options
- [Tools](tools.md) - Tool usage patterns and best practices
