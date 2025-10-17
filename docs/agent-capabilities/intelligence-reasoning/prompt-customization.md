---
layout: default
title: Prompt Customization
nav_order: 1
parent: Intelligence & Reasoning
description: "Fine-tune agent behavior with slot-based prompts"
---

# Prompt Customization

Hector uses a **slot-based prompt system** that gives you fine-grained control without losing the benefits of built-in reasoning.

## Slot System

**6 predefined slots:**

| Slot | Purpose | Example |
|------|---------|---------|
| `system_role` | Define agent identity | "You are a Python expert" |
| `reasoning_instructions` | How to think | "Use step-by-step reasoning" |
| `tool_usage` | How to use tools | "Use search for documentation" |
| `output_format` | Response structure | "Use markdown formatting" |
| `communication_style` | Tone & style | "Be concise and technical" |
| `additional` | Custom instructions | Domain-specific rules |

## Basic Customization

```yaml
agents:
  support_agent:
    name: "Customer Support Agent"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a customer support agent for TechCorp.
        You are empathetic, patient, and solution-oriented.

      communication_style: |
        - Use friendly, professional language
        - Show empathy for customer frustrations
        - Provide actionable solutions
        - End with "Is there anything else I can help with?"
```

## Advanced Customization

```yaml
agents:
  code_reviewer:
    name: "Senior Code Reviewer"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a senior code reviewer with 10+ years of experience.
        You focus on maintainability, performance, and security.

      reasoning_instructions: |
        For each code review:
        1. Check for security vulnerabilities
        2. Assess code maintainability
        3. Look for performance issues
        4. Suggest specific improvements
        5. Provide code examples for fixes

      tool_usage: |
        - Use search to find similar patterns in the codebase
        - Use execute_command to run linters/tests
        - Use write_file only if explicitly asked to fix code

      output_format: |
        Structure reviews as:
        ## Security
        [findings]

        ## Maintainability
        [findings]

        ## Performance
        [findings]

        ## Recommendations
        [specific changes with code examples]
```

## Domain-Specific Agents

**Research Agent:**
```yaml
agents:
  research_analyst:
    name: "Research Analyst"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a research analyst who synthesizes information
        from multiple sources into clear, actionable insights.

      reasoning_instructions: |
        1. Break down research questions into searchable topics
        2. Use search tool to gather information
        3. Cross-reference multiple sources
        4. Identify patterns and contradictions
        5. Synthesize findings into structured insights
```

**Debugging Agent:**
```yaml
agents:
  debugger:
    name: "Debugging Expert"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You are a debugging expert who finds root causes quickly.

      reasoning_instructions: |
        1. Reproduce the issue (use execute_command)
        2. Form hypotheses about the cause
        3. Test each hypothesis systematically
        4. Find root cause, not just symptoms
        5. Suggest preventive measures
```

## Best Practices

1. **Start Simple**: Begin with `system_role` and `communication_style`
2. **Be Specific**: Give concrete examples and instructions
3. **Test Iteratively**: Refine prompts based on agent performance
4. **Use Examples**: Show the agent what good output looks like
5. **Domain Knowledge**: Include relevant domain-specific instructions

## See Also

- **[Reasoning Strategies](reasoning-strategies)** - How agents think and reason
- **[Advanced Reasoning](../advanced-reasoning)** - Reflection and verification
- **[Agent Basics](../core-concepts/AGENTS)** - Complete agent configuration guide
