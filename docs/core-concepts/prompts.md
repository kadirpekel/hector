---
title: Prompts
description: Customize agent behavior through prompts and instructions
---

# Prompts

Prompts define how your agents think, communicate, and behave. Hector offers two approaches: **simple prompts** (recommended for most cases) or **slot-based prompts** for advanced composability.

## Quick Example

```yaml
agents:
  assistant:
    llm: "gpt-4o"
    prompt:
      system_prompt: |
        You are a helpful programming assistant.
        Provide clear, concise code examples.
      instructions: |
        Always test your code before presenting it.
```

---

## Simple Prompts (Recommended)

For most use cases, use `system_prompt` and `instructions`:

```yaml
agents:
  my_agent:
    prompt:
      system_prompt: |
        Define your agent's core role, identity, and capabilities here.
        This is the main system prompt.
      
      instructions: |
        Additional instructions for behavior, guidelines, etc.
```

**Benefits:**
- ✅ Simple and straightforward
- ✅ Works great for most use cases
- ✅ Easy to understand and maintain

---

## Slot-Based Prompts (Advanced)

Slot-based prompts provide granular control over different aspects of agent behavior. Use these when you need:
- Fine-grained control over prompt composition
- To override specific parts of reasoning strategy defaults
- Complex prompt engineering with multiple concerns

### Available Slots

```yaml
agents:
  my_agent:
    prompt:
      prompt_slots:
        system_role: |
          Core identity and role definition
        
        reasoning_instructions: |
          How the agent should think and approach problems
        
        tool_usage: |
          Guidelines for using tools effectively
        
        output_format: |
          How to format responses
        
        communication_style: |
          Tone, verbosity, and interaction style
        
        additional: |
          Any extra context or instructions
```

### Complete Example

```yaml
agents:
  coder:
    llm: "gpt-4o"
    prompt:
      prompt_slots:
        system_role: |
          You are an expert software engineer specializing in
          Python, Go, and JavaScript. You write clean, efficient,
          well-documented code.
        
        reasoning_instructions: |
          - Think through problems step by step
          - Consider edge cases and error handling
          - Explain your reasoning briefly
        
        tool_usage: |
          Use tools proactively:
          - `search` to find relevant code
          - `write_file` to create or modify files
          - `execute_command` to test your changes
        
        output_format: |
          Format code with proper syntax highlighting.
          Include brief explanations above code blocks.
        
        communication_style: |
          Be concise but thorough. Use technical terms
          appropriately. Ask clarifying questions when needed.
```

### Benefits of Slots

- **Composability** - Mix and match different aspects
- **Maintainability** - Update one aspect without touching others
- **Strategy Integration** - Reasoning strategies can inject their own slots
- **Clarity** - Clear separation of concerns

---

## Full System Prompt Override

For complete control, bypass slots and provide the entire system prompt:

```yaml
agents:
  custom:
    llm: "gpt-4o"
    prompt:
      system_prompt: |
        You are a specialized AI agent with the following capabilities:
        
        IDENTITY:
        You are a senior software architect with expertise in distributed systems.
        
        TOOLS AVAILABLE:
        - write_file: Create or modify files
        - execute_command: Run shell commands
        - search: Semantic code search
        
        BEHAVIOR:
        1. Always analyze requirements thoroughly before coding
        2. Write production-ready, tested code
        3. Document all decisions
        4. Consider scalability and maintainability
        
        CONSTRAINTS:
        - Never execute destructive commands without confirmation
        - Always validate input data
        - Follow Python PEP 8 style guide
        
        RESPONSE FORMAT:
        Provide clear, structured responses with code examples when relevant.
```

### When to Use Full Override

- Complete control over prompt structure
- Complex, domain-specific instructions
- Reproducing prompts from other systems
- When slots feel limiting

### Trade-offs

✅ **Pros:**
- Total control
- No hidden prompt composition
- Can optimize exact wording

❌ **Cons:**
- Must handle tool listings manually
- No automatic strategy integration
- More brittle (changes require full rewrite)

---

## Simple System Role (Most Common)

For most use cases, just set `system_role`:

```yaml
agents:
  helper:
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a helpful assistant who provides clear,
        concise answers to user questions.
```

Hector automatically adds:
- Tool descriptions (if tools enabled)
- Reasoning strategy instructions
- Output formatting guidelines

---

## Prompt Engineering Best Practices

### Be Specific

```yaml
# ❌ Vague
system_role: "You are helpful."

# ✅ Specific
system_role: |
  You are a Python expert who writes PEP 8 compliant code
  with comprehensive docstrings and type hints.
```

### Include Examples

```yaml
system_role: |
  You are a data analyst. Format responses like this:
  
  **Analysis:**
  [Your findings here]
  
  **Recommendation:**
  [What to do next]
  
  **Data:**
  ```json
  [Supporting data]
  ```
```

### Set Clear Boundaries

```yaml
system_role: |
  You are a customer support agent.
  
  YOU CAN:
  - Answer questions about our products
  - Help with account issues
  - Escalate to human support
  
  YOU CANNOT:
  - Access user passwords
  - Make refunds (escalate to support)
  - Share confidential business information
```

### Use Persona for Consistency

```yaml
system_role: |
  You are Ada, a friendly but professional coding tutor.
  You explain concepts clearly, use analogies, and always
  encourage learners. You speak in first person and use
  a warm, supportive tone.
```

---

## Advanced Techniques

### Context-Aware Prompts

Use environment variables or configuration to customize prompts:

```yaml
agents:
  support:
    prompt:
      system_role: |
        You are a support agent for ${COMPANY_NAME}.
        Our business hours are ${BUSINESS_HOURS}.
        Escalation email: ${SUPPORT_EMAIL}
```

```bash
export COMPANY_NAME="Acme Corp"
export BUSINESS_HOURS="9am-5pm EST"
export SUPPORT_EMAIL="support@acme.com"
```

### Multi-Language Support

```yaml
agents:
  multilingual:
    prompt:
      system_role: |
        You are a multilingual assistant.
        Respond in the same language the user uses.
        Supported languages: English, Spanish, French, German.
```

### Tool-Specific Instructions

```yaml
agents:
  researcher:
    tools: ["search", "write_file"]
    prompt:
      prompt_slots:
        tool_usage: |
          SEARCH STRATEGY:
          1. Start with broad queries
          2. Refine based on results
          3. Look for recent, authoritative sources
          
          WRITING STRATEGY:
          1. Create outlines first
          2. Write in sections
          3. Cite sources inline
```

### Chain-of-Thought Prompting

```yaml
agents:
  analyst:
    reasoning:
      engine: "chain-of-thought"
    prompt:
      prompt_slots:
        reasoning_instructions: |
          For each problem:
          1. Restate the question in your own words
          2. Break it into sub-problems
          3. Solve each step explicitly
          4. Verify your answer makes sense
          5. State your final conclusion clearly
```

---

## Prompt Debugging

### View Compiled Prompt

Enable debug output to see the final prompt sent to the LLM:

```yaml
agents:
  debug_agent:
    reasoning:
      show_debug_info: true
    prompt:
      system_role: "You are a helpful assistant."
```

### Test Different Prompts

Create multiple agent configurations to A/B test prompts:

```yaml
agents:
  assistant_v1:
    prompt:
      system_role: "You are helpful."
  
  assistant_v2:
    prompt:
      system_role: "You are an expert assistant who provides detailed, well-researched answers."
```

```bash
hector call "Explain recursion" --agent assistant_v1 --config config.yaml
hector call "Explain recursion" --agent assistant_v2 --config config.yaml
```

---

## Examples by Use Case

### Coding Assistant

```yaml
agents:
  coder:
    prompt:
      system_role: |
        You are an expert programmer. Write production-quality code
        with proper error handling, logging, and documentation.
      
      prompt_slots:
        tool_usage: |
          - Use `search` to find existing code patterns
          - Use `write_file` to create or modify files
          - Use `execute_command` to test your code
          - Always run tests after making changes
```

### Research Assistant

```yaml
agents:
  researcher:
    prompt:
      system_role: |
        You are a thorough research assistant. Gather information
        from multiple sources, synthesize findings, and provide
        well-cited, balanced analyses.
      
      prompt_slots:
        output_format: |
          Structure responses as:
          ## Summary
          ## Key Findings
          ## Sources
          ## Recommendations
```

### Customer Support

```yaml
agents:
  support:
    prompt:
      system_role: |
        You are a friendly customer support agent for TechCorp.
        Be empathetic, patient, and solution-oriented.
      
      prompt_slots:
        communication_style: |
          - Acknowledge the customer's frustration
          - Provide step-by-step solutions
          - Offer alternatives when possible
          - End with "Is there anything else I can help with?"
```

### Content Writer

```yaml
agents:
  writer:
    prompt:
      system_role: |
        You are a professional content writer specializing in
        technical blog posts. Write engaging, accurate content
        optimized for SEO.
      
      prompt_slots:
        output_format: |
          Include:
          - Compelling headline
          - Clear introduction
          - Subheadings every 300 words
          - Bullet points for lists
          - Strong conclusion with CTA
```

---

## Prompts vs Configuration

**Prompts:**
- Define behavior, personality, instructions
- Natural language
- Flexible and interpretable

**Configuration:**
- Define capabilities, constraints, connections
- Structured YAML
- Precise and enforced

Example:

```yaml
agents:
  assistant:
    # Configuration (enforced by Hector)
    llm: "gpt-4o"
    tools: ["write_file"]
    memory:
      strategy: "buffer_window"
      window_size: 10
    
    # Prompt (interpreted by LLM)
    prompt:
      system_role: |
        You are a helpful assistant. Be concise.
```

---

## Next Steps

- **[Memory](memory.md)** - Manage conversation context
- **[Tools](tools.md)** - Give agents capabilities
- **[Reasoning Strategies](reasoning.md)** - How agents think
- **[Build a Coding Assistant](../how-to/build-coding-assistant.md)** - Complete tutorial

---

## Related Topics

- **[LLM Providers](llm-providers.md)** - Configure language models
- **[Configuration Reference](../reference/configuration.md)** - All prompt options
- **[Agent Overview](overview.md)** - Understanding agents

