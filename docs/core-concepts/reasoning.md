---
title: Reasoning Strategies
description: How agents think and make decisions with chain-of-thought and supervisor strategies
---

# Reasoning Strategies

Reasoning strategies determine how agents think, make decisions, and accomplish tasks. Hector provides two built-in strategies optimized for different use cases.

## Available Strategies

| Strategy | Best For | Tool Use | Complexity |
|----------|----------|----------|------------|
| **Chain-of-Thought** | Single agents, step-by-step tasks | Sequential | Simple |
| **Supervisor** | Multi-agent orchestration, delegation | Parallel | Advanced |

---

## Chain-of-Thought (Default)

The default reasoning strategy for single-agent tasks. The agent thinks step-by-step, uses tools as needed, and stops naturally when the task is complete.

### Configuration

```yaml
agents:
  assistant:
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100        # Safety limit
      enable_streaming: true
      show_tool_execution: true
      show_thinking: false
      show_debug_info: false
```

### How It Works

```
1. Agent receives user request
2. Agent thinks about the problem
3. Agent decides what to do next:
   - Use a tool
   - Provide an answer
   - Ask for clarification
4. If tool used → go to step 2
5. If answer ready → stop
```

**Example flow:**

```
User: "Create a hello world program"

Agent: I'll create a simple hello world program.
Tool: write_file("hello.py", "print('Hello, World!')")
Agent: Created hello.py with a hello world program.

[Done - agent stops naturally]
```

### Best For

- **Single-agent tasks** - One agent handling the full request
- **Sequential workflows** - Steps that build on each other
- **Tool-enabled agents** - Agents that need to use tools iteratively
- **General purpose** - Default choice for most use cases

### Features

#### Tool Execution Display

Show tool calls to the user:

```yaml
reasoning:
  show_tool_execution: true
```

Output:
```
Agent: Let me check the current directory
[Tool: execute_command("ls -la")]
Agent: Here are the files...
```

#### Thinking Display

Show agent's internal reasoning (Claude-style):

```yaml
reasoning:
  show_thinking: true
```

Output:
```
Agent: [Thinking: I should first check if the file exists, 
        then read its contents, and finally provide a summary]
Agent: Let me read the file...
```

#### Debug Information

Show detailed execution info:

```yaml
reasoning:
  show_debug_info: true
```

Output:
```
[Iteration 1/100]
[Token count: 523]
[Tool calls: 1]
[Strategy: chain-of-thought]
```

#### Structured Reflection

Agent evaluates its own progress:

```yaml
reasoning:
  enable_structured_reflection: true
```

The agent periodically reflects:
- Am I making progress?
- Is my approach working?
- Should I try something different?

#### Max Iterations

Safety valve to prevent infinite loops:

```yaml
reasoning:
  max_iterations: 100  # Stop after 100 steps
```

---

## Supervisor Strategy

Advanced strategy for coordinating multiple agents. One "supervisor" agent delegates tasks to specialist agents and synthesizes their responses.

### Configuration

```yaml
agents:
  # Supervisor agent
  coordinator:
    reasoning:
      engine: "supervisor"
      max_iterations: 20
      enable_goal_extraction: true
    tools: ["agent_call", "todo_write"]  # Required
  
  # Specialist agents
  researcher:
    prompt:
      system_role: "You are a research specialist."
  
  analyst:
    prompt:
      system_role: "You are a data analyst."
  
  writer:
    prompt:
      system_role: "You are a technical writer."
```

### How It Works

```
1. Supervisor receives complex task
2. Supervisor breaks task into sub-tasks
3. Supervisor identifies which agents are needed
4. Supervisor delegates sub-tasks to agents:
   - Can call agents in parallel
   - Can call agents sequentially
   - Can call the same agent multiple times
5. Supervisor collects results
6. Supervisor synthesizes final answer
```

**Example flow:**

```
User: "Research AI trends and write a blog post"

Supervisor: I'll break this into research and writing tasks.
Supervisor: agent_call("researcher", "Research latest AI trends")
Researcher: [Returns research findings]

Supervisor: agent_call("writer", "Write blog post about: ...")
Writer: [Returns blog post]

Supervisor: Here's the completed blog post based on research.
```

### Best For

- **Multi-agent systems** - Coordinating specialist agents
- **Complex workflows** - Tasks requiring different expertise
- **Task decomposition** - Breaking large tasks into manageable pieces
- **Parallel execution** - Running multiple agents simultaneously

### Required Components

#### 1. agent_call Tool

Supervisor **must** have `agent_call` tool:

```yaml
agents:
  supervisor:
    tools: ["agent_call"]  # Required
```

This allows calling other agents.

#### 2. todo_write Tool (Recommended)

Track task progress:

```yaml
agents:
  supervisor:
    tools: ["agent_call", "todo_write"]  # Recommended
```

### Features

#### Goal Extraction

LLM extracts goals from user requests:

```yaml
reasoning:
  engine: "supervisor"
  enable_goal_extraction: true
```

User: "Build a web app"  
→ Goals: [Setup project, Create backend, Create frontend, Deploy]

#### Agent Selection

Supervisor automatically discovers available agents and selects appropriate ones:

```yaml
# All agents registered with Hector are available
agents:
  supervisor:
    reasoning:
      engine: "supervisor"
  
  frontend_dev:
    # Available to supervisor
  
  backend_dev:
    # Available to supervisor
```

**Or restrict to specific agents:**

```yaml
agents:
  supervisor:
    reasoning:
      engine: "supervisor"
    sub_agents: ["researcher", "analyst"]  # Only these agents
```

#### Task Decomposition

Supervisor breaks complex tasks into manageable sub-tasks automatically.

#### Result Synthesis

Supervisor combines results from multiple agents into a coherent final answer.

---

## Choosing a Strategy

### Use Chain-of-Thought When

```yaml
# ✅ Single agent handling the task
agents:
  coder:
    reasoning:
      engine: "chain-of-thought"
    tools: ["write_file", "execute_command"]

# ✅ Sequential tool use
# ✅ General-purpose agents
# ✅ Simple to moderate complexity
```

### Use Supervisor When

```yaml
# ✅ Multiple specialist agents
agents:
  coordinator:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  researcher:
    # Specialist
  writer:
    # Specialist

# ✅ Complex, multi-step tasks
# ✅ Different expertise needed
# ✅ Parallel execution beneficial
```

---

## Reasoning Configuration Reference

### Chain-of-Thought Options

```yaml
agents:
  agent:
    reasoning:
      engine: "chain-of-thought"
      
      # Safety
      max_iterations: 100              # Stop after N iterations
      
      # Display
      enable_streaming: true           # Stream responses
      show_tool_execution: true        # Show tool calls
      show_thinking: false             # Show internal reasoning
      show_debug_info: false           # Show debug details
      
      # Reflection
      enable_structured_reflection: true   # Self-evaluation
      enable_completion_verification: false  # Verify task completion
```

### Supervisor Options

```yaml
agents:
  supervisor:
    reasoning:
      engine: "supervisor"
      
      # Orchestration
      max_iterations: 20               # Fewer iterations for delegation
      enable_goal_extraction: true     # Extract goals from tasks
      
      # Agent selection
      sub_agents: []                   # Empty = all agents available
      # sub_agents: ["agent1", "agent2"]  # Or specific agents only
      
      # Display (inherited from chain-of-thought)
      enable_streaming: true
      show_tool_execution: true
```

---

## Advanced Patterns

### Hierarchical Orchestration

Supervisors calling other supervisors:

```yaml
agents:
  # Top-level supervisor
  master:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  # Mid-level supervisors
  research_lead:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  dev_lead:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  # Worker agents
  researcher1:
    # Specialist
  researcher2:
    # Specialist
  frontend_dev:
    # Specialist
  backend_dev:
    # Specialist
```

### Mixed Strategy System

Some agents use chain-of-thought, some use supervisor:

```yaml
agents:
  # Supervisor for orchestration
  coordinator:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  # Chain-of-thought for actual work
  coder:
    reasoning:
      engine: "chain-of-thought"
    tools: ["write_file", "execute_command"]
  
  tester:
    reasoning:
      engine: "chain-of-thought"
    tools: ["execute_command"]
```

---

## Examples by Use Case

### Coding Assistant (Chain-of-Thought)

```yaml
agents:
  coder:
    llm: "gpt-4o"
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 100
      show_tool_execution: true
    tools: ["write_file", "search_replace", "execute_command"]
    
    prompt:
      system_role: |
        You are an expert programmer. Think through problems
        step-by-step and test your changes.
```

### Research System (Supervisor)

```yaml
agents:
  research_coordinator:
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"
      enable_goal_extraction: true
    tools: ["agent_call", "todo_write"]
    
    prompt:
      system_role: |
        You coordinate a research team. Break complex research
        tasks into focused sub-tasks and delegate appropriately.
  
  web_researcher:
    llm: "gpt-4o"
    prompt:
      system_role: "You search the web for information."
  
  analyst:
    llm: "claude"
    prompt:
      system_role: "You analyze data and draw conclusions."
  
  writer:
    llm: "claude"
    prompt:
      system_role: "You write clear, engaging research summaries."
```

---

## Monitoring & Debugging

### See Reasoning Steps

```yaml
agents:
  debug:
    reasoning:
      show_debug_info: true
      show_tool_execution: true
      show_thinking: true
```

Output shows:
```
[Iteration 1/100]
[Thinking: I should first check if the file exists]
[Tool: execute_command("ls README.md")]
[Tool Result: README.md]
[Thinking: File exists, now I'll read it]
...
```

### Track Iterations

Monitor if agents are using too many iterations:

```yaml
reasoning:
  max_iterations: 50  # Lower limit to catch issues sooner
  show_debug_info: true
```

If hitting the limit frequently:
- Task may be too complex for the agent
- Agent may be stuck in a loop
- May need better prompting
- May need supervisor strategy instead

---

## Best Practices

### For Chain-of-Thought

```yaml
# ✅ Good: Clear role and tools
agents:
  assistant:
    reasoning:
      engine: "chain-of-thought"
    tools: ["write_file", "search"]
    prompt:
      system_role: "Clear, specific role"

# ❌ Bad: Too many tools confuses agent
agents:
  confused:
    tools: ["*"]  # All tools - agent unsure what to use
```

### For Supervisor

```yaml
# ✅ Good: Supervisor focuses on delegation
agents:
  supervisor:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call", "todo_write"]  # Only orchestration tools
    prompt:
      system_role: |
        You coordinate specialists. Delegate tasks and synthesize results.

# ❌ Bad: Supervisor doing work itself
agents:
  confused_supervisor:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call", "write_file", "execute_command"]  # Too many tools
    # Supervisor should delegate, not do the work
```

### Prompting for Reasoning

```yaml
agents:
  thoughtful:
    reasoning:
      engine: "chain-of-thought"
    prompt:
      prompt_slots:
        reasoning_instructions: |
          Think step-by-step:
          1. Analyze the problem
          2. Plan your approach
          3. Execute with tools
          4. Verify your solution
          5. Provide clear results
```

---

## Next Steps

- **[Multi-Agent Orchestration](multi-agent.md)** - Build multi-agent systems
- **[Tools](tools.md)** - agent_call and other tools
- **[Prompts](prompts.md)** - Optimize reasoning with better prompts
- **[How to Build a Research System](../how-to/build-research-system.md)** - Complete tutorial

---

## Related Topics

- **[Agent Overview](overview.md)** - Understanding agents
- **[Configuration Reference](../reference/configuration.md)** - All reasoning options
- **[Build a Coding Assistant](../how-to/build-coding-assistant.md)** - Chain-of-thought example

