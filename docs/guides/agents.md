# Agents

Agents are the core building blocks of Hector. An **Agent** is an autonomous entity that combines an LLM, Tools, and Instructions to solve tasks.

## Agent Types

| Type | Description |
|------|-------------|
| `llm` | LLM-backed intelligent agent (default) |
| `sequential` | Runs sub-agents in order |
| `parallel` | Runs sub-agents concurrently |
| `loop` | Iterates until condition met |
| `conditional` | Routes based on evaluation |
| `runner` | Deterministic tool pipeline (no LLM) |
| `remote` | Proxies to external A2A agent |

## LLM Agents

The default agent type uses an LLM for reasoning.

```yaml
agents:
  assistant:
    name: "Assistant"
    description: "A helpful AI assistant"
    llm: claude
    instruction: "You are a helpful assistant."
    tools: [search, calculator]
```

### Key Components

| Component | Description |
|-----------|-------------|
| **llm** | Model powering the agent (e.g., `claude`, `gpt4`) |
| **instruction** | System prompt defining behavior |
| **tools** | Capabilities the agent can call |

### Reasoning Loop

When an agent receives a task:

1. **Observe**: Read conversation history and input
2. **Think**: LLM generates decision
3. **Act**: Emit tool call if needed
4. **Result**: Execute tool, return result
5. **Repeat**: Until final answer

### Instructions

Inline or from file:

```yaml
# Inline
instruction: "You are a concise chatbot."

# From file
instruction_file: "./prompts/researcher.md"
```

Template variables for dynamic context:
- `{user:name}` - User-scoped
- `{app:config}` - App-scoped
- `{artifact.data}` - File content

### Reasoning Configuration

```yaml
agents:
  coder:
    reasoning:
      max_iterations: 50
      enable_exit_tool: true
      enable_escalate_tool: true
```

### Context & Memory

Control conversation history to fit LLM context limits. See the [Context & Memory Strategies](#context-memory-strategies) section below for full details.

```yaml
agents:
  chatbot:
    context:
      strategy: token_window
      budget: 8000
```

---

## Multi-Agent Orchestration

Compose agents into complex systems using workflow types.

### Sequential

Run sub-agents in strict order:

```yaml
agents:
  blog_pipeline:
    type: sequential
    sub_agents: [researcher, writer, editor]

  researcher:
    llm: claude
    instruction: "Find facts about the topic."
  writer:
    llm: claude
    instruction: "Write a draft."
  editor:
    llm: gpt4
    instruction: "Fix grammar and tone."
```

### Parallel

Run sub-agents concurrently:

```yaml
agents:
  consensus:
    type: parallel
    sub_agents: [analyst_a, analyst_b, analyst_c]
```

### Loop

Iterate until condition or max:

```yaml
agents:
  refinement:
    type: loop
    sub_agents: [coder, reviewer]
    max_iterations: 3
```

### Conditional

Route based on evaluation:

```yaml
agents:
  safe_assistant:
    type: conditional
    condition_agent: moderator
    condition_field: "safe"
    on_true_agent: helper
    on_false_response: "I cannot help with that."
```

---

## Runner Agents

Deterministic tool pipelines with **no LLM involvement**.

```yaml
agents:
  etl_job:
    type: runner
    tools: [fetch_api, transform, save_data]
```

### How It Works

1. Input parsed as JSON
2. Tool 1 receives input, returns output
3. Output of Tool N becomes input of Tool N+1
4. Final tool output is the agent response

### Use Cases

- Data fetching pipelines
- ETL workflows
- CI/CD automation
- Format conversion

### Combining with LLM Agents

Use runners as sub-agents for reliable data fetching:

```yaml
agents:
  analyst:
    llm: claude
    sub_agents: [data_fetcher]

  data_fetcher:
    type: runner
    tools: [stock_api, news_api]
```

---

## Agent Composition

### Sub-Agents (Transfer)

Control transfers to sub-agent. The parent hands off the conversation. The sub-agent takes over, including the user interaction.

```yaml
agents:
  manager:
    sub_agents: [researcher, writer]
```

Runtime provides `transfer_to_researcher` and `transfer_to_writer` tools.

### Agent Tools (Delegation)

Parent stays in control. The child agent executes in an isolated session and returns a result. The parent decides what to do with it.

```yaml
agents:
  assistant:
    agent_tools: [fact_checker]
```

### When to Use Which

| | Sub-Agents (Transfer) | Agent Tools (Delegation) |
|-|-----------------------|-------------------------|
| **Control** | Child takes over conversation | Parent stays in control |
| **Session** | Shared session with parent | Isolated session (no state bleed) |
| **Best for** | Routing/triage, specialized handlers | Helper tasks, data enrichment |
| **Example** | "Transfer to billing department" | "Ask the fact-checker about this claim" |

**Rule of thumb:** Use `sub_agents` when the child should talk directly to the user. Use `agent_tools` when the parent needs to process the child's output.

---

## Triggers

Run agents automatically.

### Scheduled

```yaml
trigger:
  type: schedule
  cron: "0 9 * * *"
  timezone: America/New_York
```

### Webhook

```yaml
trigger:
  type: webhook
  path: /webhooks/github
  secret: ${GITHUB_SECRET}
```

See [Triggers Guide](./triggers.md) for details.

---

## Notifications

Outbound webhooks on agent events:

```yaml
notifications:
  - id: slack
    events: [task_completed, task_failed]
    url: https://hooks.slack.com/...
```

---

## Visibility

Control which agents are discoverable and accessible via the A2A protocol:

| Visibility | Discoverable | HTTP Accessible | Use Case |
|------------|:------------:|:---------------:|----------|
| `public` | Yes | Yes (auth if enabled) | Customer-facing agents |
| `internal` | Only when authenticated | Yes | Admin/internal tools |
| `private` | No | No (internal calls only) | Sub-agents, helper agents |

```yaml
agents:
  customer_agent:
    visibility: public      # Listed in agent card, anyone can call
    # ...

  admin_tools:
    visibility: internal    # Only authenticated users can discover & call
    # ...

  classifier:
    visibility: private     # Only callable by other agents, never via HTTP
    # ...
```

The `/.well-known/agent-card.json` and `/agents` endpoints only list agents matching the caller's access level. Private agents are invisible to all external consumers.

---

## Context & Memory Strategies

Control how conversation history is managed to stay within LLM context limits.

### Strategy Overview

| Strategy | Description | Best For |
|----------|-------------|----------|
| `none` | Keep all messages (default) | Short conversations |
| `buffer_window` | Keep last N messages | Simple chat UIs |
| `token_window` | Keep messages within token budget | Cost control |
| `summary_buffer` | Summarize older messages with LLM | Long conversations |

### Buffer Window

Keep a fixed number of recent messages:

```yaml
agents:
  chatbot:
    context:
      strategy: buffer_window
      window_size: 20          # Keep last 20 messages
```

### Token Window

Keep recent messages within a token budget:

```yaml
agents:
  chatbot:
    context:
      strategy: token_window
      budget: 8000             # Max 8000 tokens of history
```

### Summary Buffer (Autonomous Summarization)

When conversation history exceeds the token threshold, Hector automatically summarizes older messages using the agent's LLM. The summary preserves key facts, names, decisions, and context while reducing token count.

```yaml
agents:
  chatbot:
    context:
      strategy: summary_buffer
      budget: 8000             # Summarize when exceeding this
```

How it works:

1. History grows normally until the token budget is reached
2. Older messages are passed to the LLM with a summarization prompt
3. The summary replaces those messages, preserving facts and context
4. New messages continue to accumulate until the next summarization

This happens transparently. Users don't see the summarization, only its effects (the agent maintains long-term context without running out of tokens).

---

## Next Steps

*   [Tools Guide](./tools.md)
*   [Knowledge (RAG) Guide](./knowledge.md)
*   [Agents Reference](../reference/configuration.md#agents)
