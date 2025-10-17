---
layout: default
title: Reasoning Strategies
nav_order: 2
parent: Intelligence & Reasoning
description: "Chain-of-thought vs supervisor reasoning"
---

# Reasoning Strategies

Hector provides two built-in reasoning strategies optimized for different use cases.

## Chain-of-Thought (Default)

**Best for:** Single-agent tasks, general problem-solving

**How it works:**
- Agent thinks step-by-step
- Can use tools at any point
- Automatically decides when task is complete
- Fast and cost-effective

**Configuration:**
```yaml
agents:
  assistant:
    name: "Assistant"
    llm: "gpt-4o"
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 10
      enable_streaming: true
```

**Characteristics:**
- One LLM call per iteration
- Implicit planning
- Tool execution with automatic continuation
- Natural conversation flow
- Fast response times

**Use cases:**
- Coding assistants
- Research agents
- Customer support
- Content creation
- General Q&A

## Supervisor (For Orchestration)

**Best for:** Multi-agent coordination

**How it works:**
- Specialized prompts for task decomposition
- Guides agent selection and delegation
- Helps synthesize results from multiple agents
- Works with `agent_call` tool

**Configuration:**
```yaml
agents:
  orchestrator:
    name: "Orchestrator"
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"
      max_iterations: 20  # More iterations for complex orchestration
      enable_streaming: true
```

**Characteristics:**
- Task decomposition guidance
- Agent delegation patterns
- Result synthesis support
- Based on chain-of-thought with orchestration enhancements

**Use cases:**
- Multi-agent workflows
- Complex research pipelines
- Cross-functional tasks
- Hierarchical processing

## Choosing the Right Strategy

### Use Chain-of-Thought When:
- Building single agents
- Need fast responses
- Simple to moderate complexity tasks
- General problem-solving

### Use Supervisor When:
- Coordinating multiple agents
- Complex multi-step workflows
- Need task decomposition
- Hierarchical processing

## Configuration Options

```yaml
agents:
  my_agent:
    name: "My Agent"
    llm: "gpt-4o"
    reasoning:
      engine: "chain-of-thought"  # or "supervisor"
      max_iterations: 10          # Maximum reasoning cycles
      enable_streaming: true      # Real-time output
```

## See Also

- **[Prompt Customization](prompt-customization)** - Fine-tune agent behavior
- **[Advanced Reasoning](advanced-reasoning)** - Reflection and verification
- **[Multi-Agent Systems](../multi-agent-systems)** - Agent orchestration patterns
- **[LangChain vs Hector Tutorial](../../architecture-design/TUTORIAL_MULTI_AGENT)** - Complete multi-agent example
