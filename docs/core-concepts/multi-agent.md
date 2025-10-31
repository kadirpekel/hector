---
title: Multi-Agent Orchestration
description: Coordinate multiple agents to solve complex tasks
---

# Multi-Agent Orchestration

Multi-agent systems allow you to coordinate multiple specialized agents to solve complex tasks. One supervisor agent breaks down tasks and delegates to specialists, each focused on their area of expertise.

## Why Multi-Agent?

**Single Agent:**
```
User: "Research AI trends and write a blog post"
Agent: [Tries to do everything, results may be suboptimal]
```

**Multi-Agent:**
```
User: "Research AI trends and write a blog post"
Coordinator: → Researcher: "Research AI trends"
Coordinator: → Analyst: "Analyze research findings"
Coordinator: → Writer: "Write blog post based on analysis"
Result: [Each specialist does what they do best]
```

### Benefits

- **Specialization** - Each agent focused on one thing
- **Modularity** - Easy to add/remove specialists
- **Parallel Execution** - Multiple agents work simultaneously
- **Better Results** - Experts produce better output than generalists

---

## Quick Example

```yaml
agents:
  # Coordinator with supervisor reasoning
  coordinator:
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"
    tools: ["agent_call", "todo_write"]
    prompt:
      system_prompt: |
        You coordinate a team of specialists. Break tasks
        into sub-tasks and delegate appropriately.
  
  # Specialist agents
  researcher:
    llm: "gpt-4o"
    tools: ["search"]
    prompt:
      system_prompt: "You research topics and gather information."
  
  analyst:
    llm: "claude"
    prompt:
      system_prompt: "You analyze data and draw insights."
  
  writer:
    llm: "claude"
    tools: ["write_file"]
    prompt:
      system_prompt: "You write clear, engaging content."
```

**Usage:**

```bash
hector call coordinator "Research quantum computing and write an article"
```

**What happens:**

1. Coordinator receives request
2. Coordinator: `agent_call("researcher", "Research quantum computing")`
3. Researcher returns findings
4. Coordinator: `agent_call("analyst", "Analyze these findings: ...")`
5. Analyst returns insights
6. Coordinator: `agent_call("writer", "Write article with insights: ...")`
7. Writer returns article
8. Coordinator synthesizes and returns final result

---

## Core Components

### 1. Supervisor Agent

The orchestrator that coordinates work:

```yaml
agents:
  supervisor:
    reasoning:
      engine: "supervisor"  # Required
    tools:
      - "agent_call"        # Required
      - "todo_write"        # Recommended
```

**Requirements:**
- **Supervisor reasoning engine** - Enables delegation
- **agent_call tool** - Calls other agents

**Recommendations:**
- **todo_write tool** - Tracks sub-tasks
- **Strong LLM** - GPT-4o or Claude for good orchestration

### 2. Specialist Agents

Focused agents that do the actual work:

```yaml
agents:
  specialist:
    prompt:
      system_prompt: "Clear, focused role definition"
    tools: ["relevant", "tools"]
```

**Best practices:**
- **Single responsibility** - One clear purpose
- **Clear role** - Precise system prompt
- **Appropriate tools** - Only what's needed for the role

### 3. agent_call Tool

Bridges agents together:

```yaml
supervisor: agent_call("specialist", "Do specific task")
```

Syntax: `agent_call(agent_name, task_description)`

---

## Architecture Patterns

### Pattern 1: Simple Delegation

One coordinator, multiple workers:

```
┌──────────────┐
│ Coordinator  │
└──────┬───────┘
       │
       ├──→ Researcher
       ├──→ Analyst
       └──→ Writer
```

```yaml
agents:
  coordinator:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  researcher:
  analyst:
  writer:
```

### Pattern 2: Pipeline

Sequential processing:

```
User → Coordinator → Agent1 → Agent2 → Agent3 → Result
```

```yaml
agents:
  coordinator:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
    prompt:
      system_prompt: |
        Process tasks through pipeline:
        1. Data collector gathers data
        2. Processor processes it
        3. Formatter formats output
  
  data_collector:
  processor:
  formatter:
```

### Pattern 3: Hierarchical

Supervisors coordinating supervisors:

```
┌─────────────────┐
│ Master          │
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
  Lead1     Lead2
    │         │
  ┌─┴─┐     ┌─┴─┐
  │   │     │   │
  W1  W2    W3  W4
```

```yaml
agents:
  master:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  research_lead:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  dev_lead:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  researcher1:
  researcher2:
  frontend_dev:
  backend_dev:
```

### Pattern 4: Swarm

Coordinator uses same agent type multiple times:

```yaml
agents:
  coordinator:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
    prompt:
      system_prompt: |
        You coordinate multiple researchers working in parallel.
  
  researcher:
    # Can be called multiple times with different queries
    tools: ["search"]
```

---

## Calling External A2A Agents

Integrate remote agents via the A2A protocol:

```yaml
agents:
  # Local coordinator
  coordinator:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  # External A2A agent
  external_specialist:
    type: "a2a"
    url: "https://external-agent.example.com"
    credentials:
      type: "bearer"
      token: "${EXTERNAL_TOKEN}"
```

**Usage (same as local agents):**

```yaml
coordinator: agent_call("external_specialist", "Analyze this data")
```

### External Agent Authentication

```yaml
agents:
  external_agent:
    type: "a2a"
    url: "https://agent.example.com"
    credentials:
      # Bearer token
      type: "bearer"
      token: "${TOKEN}"
      
      # OR API key
      type: "api_key"
      key: "${API_KEY}"
      header: "X-API-Key"
      
      # OR Basic auth
      type: "basic"
      username: "${USERNAME}"
      password: "${PASSWORD}"
```

See [How to Integrate External Agents](../how-to/integrate-external-agents.md) for details.

---

## Best Practices

### 1. Clear Role Separation

```yaml
# ✅ Good: Clear, distinct roles
agents:
  coordinator:
    prompt:
      system_prompt: "You coordinate specialists."
  
  researcher:
    prompt:
      system_prompt: "You research topics thoroughly."
  
  writer:
    prompt:
      system_prompt: "You write engaging articles."

# ❌ Bad: Overlapping responsibilities
agents:
  agent1:
    prompt:
      system_prompt: "You research and write."  # Two jobs
```

### 2. Supervisor Doesn't Do Work

```yaml
# ✅ Good: Supervisor only delegates
agents:
  supervisor:
    tools: ["agent_call", "todo_write"]  # Orchestration tools only
    prompt:
      system_prompt: "Delegate to specialists, don't do the work yourself."

# ❌ Bad: Supervisor has worker tools
agents:
  confused:
    tools: ["agent_call", "write_file", "execute_command"]  # Too many tools
    # Supervisor confused about its role
```

### 3. Right Granularity

```yaml
# ✅ Good: Appropriate specialization
agents:
  coordinator:
  frontend_dev:
  backend_dev:
  database_admin:

# ❌ Bad: Over-specialized
agents:
  coordinator:
  button_designer:
  navbar_designer:
  footer_designer:  # Too granular
```

### 4. Task Decomposition Prompting

```yaml
agents:
  coordinator:
    prompt:
      system_prompt: |
        You coordinate specialists. For each task:
        1. Break it into clear sub-tasks
        2. Identify which specialist is best
        3. Delegate with clear instructions
        4. Synthesize results into final answer
        
        Available specialists:
        - researcher: Gathers information
        - analyst: Analyzes data
        - writer: Creates content
```

---

## Complete Examples

### Example 1: Research Team

```yaml
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
  
  claude:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"

agents:
  research_coordinator:
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"
      enable_goal_extraction: true
    tools: ["agent_call", "todo_write"]
    prompt:
      system_prompt: |
        You coordinate a research team. Break research tasks into:
        1. Information gathering (web_researcher)
        2. Data analysis (analyst)
        3. Writing (writer)
        
        Delegate appropriately and synthesize final results.
  
  web_researcher:
    llm: "gpt-4o"
    tools: ["search"]
    prompt:
      system_prompt: |
        You search for information on topics. Provide comprehensive,
        well-cited findings.
  
  analyst:
    llm: "claude"
    prompt:
      system_prompt: |
        You analyze research findings. Draw insights, identify patterns,
        and provide data-driven conclusions.
  
  writer:
    llm: "claude"
    tools: ["write_file"]
    prompt:
      system_prompt: |
        You write engaging, well-structured content based on research
        and analysis. Your writing is clear, accurate, and compelling.
```

**Usage:**

```bash
hector serve --config research-team.yaml

hector call research_coordinator "Research the impact of AI on healthcare and write a comprehensive report"
```

### Example 2: Software Development Team

```yaml
agents:
  dev_lead:
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"
    tools: ["agent_call", "todo_write"]
    prompt:
      system_prompt: |
        You lead a development team. Break tasks into:
        - Architecture (architect)
        - Frontend (frontend_dev)
        - Backend (backend_dev)
        - Testing (tester)
  
  architect:
    llm: "gpt-4o"
    prompt:
      system_prompt: "You design system architecture and make technical decisions."
  
  frontend_dev:
    llm: "gpt-4o"
    tools: ["write_file", "search", "execute_command"]
    prompt:
      system_prompt: "You build React frontends with TypeScript."
  
  backend_dev:
    llm: "gpt-4o"
    tools: ["write_file", "search", "execute_command"]
    prompt:
      system_prompt: "You build Go backends with clean architecture."
  
  tester:
    llm: "gpt-4o"
    tools: ["write_file", "execute_command"]
    prompt:
      system_prompt: "You write and run comprehensive tests."
```

### Example 3: Customer Support System

```yaml
agents:
  support_router:
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
    prompt:
      system_prompt: |
        You route support requests to specialists:
        - billing_support: Payment issues
        - tech_support: Technical problems
        - account_support: Account issues
  
  billing_support:
    llm: "gpt-4o"
    prompt:
      system_prompt: "You help with billing and payment issues."
  
  tech_support:
    llm: "gpt-4o"
    tools: ["execute_command"]
    prompt:
      system_prompt: "You diagnose and fix technical issues."
  
  account_support:
    llm: "gpt-4o"
    prompt:
      system_prompt: "You help with account management and settings."
```

---

## Debugging Multi-Agent Systems

### Enable Debug Output

```yaml
agents:
  coordinator:
    reasoning:
      show_tool_execution: true
      show_debug_info: true
```

**Output shows:**

```
[Coordinator] Received task: "Research and write about AI"
[Coordinator] Breaking into sub-tasks...
[Coordinator] Calling agent: researcher
[Tool] agent_call("researcher", "Research AI trends")
[Researcher] Searching for AI trends...
[Researcher] Found 10 sources...
[Tool Result] Research complete
[Coordinator] Calling agent: writer
[Tool] agent_call("writer", "Write article about: ...")
[Writer] Creating article structure...
[Writer] Writing content...
[Tool Result] Article complete
[Coordinator] Synthesizing final result...
```

### Track Agent Calls

Look for agent_call tool usage:

```bash
hector call coordinator "Complex task" --debug
```

### Monitor Performance

Check how many calls are made:

```yaml
reasoning:
  max_iterations: 20  # Coordinator should finish in fewer iterations
  show_debug_info: true
```

---

## Comparison with Other Frameworks

### Hector vs. LangChain

| Feature | Hector | LangChain |
|---------|--------|-----------|
| Configuration | Pure YAML | Python code |
| Agent calling | Native `agent_call` | Custom chain setup |
| A2A compatible | ✅ Yes | ❌ No |
| External agents | Native support | Manual integration |
| Setup time | Minutes | Hours |

### Hector vs. AutoGPT

| Feature | Hector | AutoGPT |
|---------|--------|---------|
| Multi-agent | Native supervisor | Plugin-based |
| Configuration | Declarative YAML | Code + config |
| Tool system | Built-in + MCP + plugins | Plugin architecture |
| Production ready | ✅ Yes | ⚠️ Experimental |

---

## When to Use Multi-Agent

**Use Multi-Agent When:**
- Task requires different expertise
- Need parallel processing
- Want modularity and reusability
- Building complex systems

**Use Single Agent When:**
- Simple, focused tasks
- One area of expertise sufficient
- Speed is critical (no delegation overhead)
- Just getting started

---

## API Usage with Multi-Agent

When calling a multi-agent server via API, you **must** specify which agent to use.

### REST API

Agent is part of the URL path:

```bash
# Explicit agent in path
POST /v1/agents/orchestrator/message:send
POST /v1/agents/researcher/message:send
```

### JSON-RPC API

Use one of three methods:

**1. Query String (Recommended):**
```bash
POST /?agent=orchestrator
POST /?agent=researcher
```

**2. Context ID Format:**
```json
{
  "params": {
    "request": {
      "contextId": "orchestrator:session-123"
    }
  }
}
```

**3. Request Metadata:**
```json
{
  "params": {
    "request": {
      "metadata": {"name": "orchestrator"}
    }
  }
}
```

### Single vs Multi-Agent Behavior

| Setup | Agent Specification | Behavior |
|-------|-------------------|----------|
| **Single Agent** | Optional | Auto-routes to only agent |
| **Multi-Agent** | **Required** | Must specify agent name |

**Example Error (Multi-Agent without specification):**
```json
{
  "error": {
    "message": "name not specified"
  }
}
```

See the [API Reference](../reference/api.md#agent-selection) for detailed examples.

---

## Next Steps

- **[How to Build a Research System](../how-to/build-research-system.md)** - Complete tutorial
- **[How to Integrate External Agents](../how-to/integrate-external-agents.md)** - A2A integration
- **[Reasoning Strategies](reasoning.md)** - Supervisor strategy details
- **[Tools](tools.md)** - agent_call tool
- **[API Reference](../reference/api.md#agent-selection)** - Agent selection in API calls

---

## Related Topics

- **[Agent Overview](overview.md)** - Understanding agents
- **[Configuration Reference](../reference/configuration.md)** - All orchestration options
- **[A2A Protocol](../reference/a2a-protocol.md)** - Agent interoperability

