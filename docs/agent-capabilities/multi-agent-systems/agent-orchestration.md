---
layout: default
title: Agent Orchestration
nav_order: 2
parent: Multi-Agent Systems
description: "Coordinate multiple agents"
---

# Agent Orchestration

Coordinate multiple agents to work together on complex tasks and workflows.

## Orchestration Patterns

### Supervisor Pattern

Use a supervisor agent to coordinate multiple specialized agents:

```yaml
agents:
  supervisor:
    name: "Task Supervisor"
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"  # Specialized for orchestration
    tools:
      - agent_call  # Can call other agents
    
  researcher:
    name: "Research Agent"
    llm: "claude-3-5-sonnet"
    document_stores:
      - "research_docs"
    
  writer:
    name: "Content Writer"
    llm: "gpt-4o"
    prompt:
      system_role: "You are an expert technical writer"
```

### Workflow Example

```
User: "Research and write a report on AI trends"

Supervisor: "I'll coordinate this task"
├── Calls researcher: "Find recent AI trend data"
├── Calls writer: "Create report from research findings"
└── Synthesizes final report
```

## Configuration

### Enable Agent Calls

```yaml
tools:
  agent_call:
    type: "agent_call"
    enabled: true
    allowed_agents:
      - "researcher"
      - "writer"
      - "reviewer"
```

### Supervisor Reasoning

```yaml
reasoning:
  engine: "supervisor"
  max_iterations: 20  # More iterations for complex orchestration
  enable_streaming: true
```

## Use Cases

- **Research Pipelines** - Research → Analysis → Writing
- **Code Development** - Planning → Coding → Testing → Review
- **Content Creation** - Research → Writing → Editing → Publishing
- **Customer Support** - Triage → Specialist → Escalation

## Best Practices

1. **Clear Roles** - Define specific responsibilities for each agent
2. **Task Decomposition** - Break complex tasks into manageable steps
3. **Result Synthesis** - Combine outputs from multiple agents
4. **Error Handling** - Handle failures gracefully
5. **Monitoring** - Track orchestration performance

## See Also

- **[External Agents](external-agents)** - Integrate remote agents
- **[Reasoning Strategies](../intelligence-reasoning/reasoning-strategies)** - Supervisor reasoning
- **[LangChain vs Hector Tutorial](../../architecture-design/TUTORIAL_MULTI_AGENT)** - Complete example
