---
title: Build a Multi-Agent Research System
description: Create an automated research assistant with specialized agents in 45 minutes
---

# How to Build a Multi-Agent Research System

Build an automated research assistant that coordinates multiple specialized agents to research topics, analyze findings, and write comprehensive reports.

**Time:** 45 minutes  
**Difficulty:** Advanced

---

## What You'll Build

A multi-agent system with:

- **Coordinator** - Orchestrates the research workflow
- **Researcher** - Gathers information from sources
- **Analyst** - Analyzes data and draws insights
- **Writer** - Creates structured reports

**Workflow:**
```
User Query → Coordinator
              ↓
           Researcher (gather info)
              ↓
           Analyst (analyze findings)
              ↓
           Writer (create report)
              ↓
           Coordinator (deliver results)
```

---

## Prerequisites

✅ Hector installed ([Installation Guide](../getting-started/installation.md))  
✅ Understanding of [Multi-Agent Orchestration](../core-concepts/multi-agent.md)  
✅ API keys for LLM providers

---

## Step 1: Understand the Architecture

### Key Components

**Supervisor Agent (Coordinator):**
- Uses `supervisor` reasoning engine
- Has `agent_call` and `todo_write` tools
- Breaks down tasks and delegates to specialists

**Specialist Agents:**
- Each focused on one responsibility
- Use `chain-of-thought` reasoning
- Have tools relevant to their role

### Why Multi-Agent?

Instead of one agent trying to do everything:
```
❌ Single generalist agent → Mediocre at everything
✅ Multiple specialist agents → Expert at their domain
```

---

## Step 2: Create Configuration

**Tip:** While multi-agent systems need explicit agent definitions, you can still use `enable_tools: true` shortcuts for each agent to simplify tool configuration!

Create `research-system.yaml`:

```yaml
# LLM Configurations
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
  
  claude:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"
    temperature: 0.7

# Agents
agents:
  # === COORDINATOR (Supervisor) ===
  research_coordinator:
    name: "Research Coordinator"
    llm: "gpt-4o"
    
    # Supervisor reasoning for orchestration
    reasoning:
      engine: "supervisor"
      max_iterations: 20
      enable_goal_extraction: true
      show_tool_execution: true
    
    # Orchestration tools only
    tools:
      - "agent_call"
      - "todo_write"
    
    prompt:
      system_prompt: |
        You are a research coordinator managing a team of specialists.
        
        TEAM MEMBERS:
        - researcher: Gathers information from sources
        - analyst: Analyzes data and draws conclusions
        - writer: Creates structured reports
        
        WORKFLOW:
        1. Break the research query into clear sub-tasks
        2. Delegate to researcher to gather information
        3. Delegate to analyst to analyze findings
        4. Delegate to writer to create final report
        5. Synthesize and deliver the complete result
        
        Coordinate effectively and ensure high-quality output.
  
  # === RESEARCHER (Specialist) ===
  researcher:
    name: "Research Specialist"
    llm: "gpt-4o"
    
    reasoning:
      engine: "chain-of-thought"
      show_tool_execution: true
    
    tools:
      - "search"          # If RAG enabled
      - "write_file"      # Save research notes
    
    prompt:
      system_prompt: |
        You are a thorough research specialist.
        
        Your role:
        - Gather information from multiple sources
        - Verify facts and credibility
        - Organize findings clearly
        - Cite sources properly
        
        Provide comprehensive, well-researched information.
  
  # === ANALYST (Specialist) ===
  analyst:
    name: "Research Analyst"
    llm: "claude"  # Claude is good at analysis
    
    reasoning:
      engine: "chain-of-thought"
    
    prompt:
      system_prompt: |
        You are a data analyst specializing in research insights.
        
        Your role:
        - Analyze research findings critically
        - Identify patterns and trends
        - Draw meaningful conclusions
        - Provide data-driven insights
        - Highlight limitations and gaps
        
        Be analytical, objective, and insightful.
  
  # === WRITER (Specialist) ===
  writer:
    name: "Technical Writer"
    llm: "claude"  # Claude is good at writing
    
    reasoning:
      engine: "chain-of-thought"
    
    tools:
      - "write_file"  # Save final reports
    
    prompt:
      system_prompt: |
        You are a professional technical writer.
        
        Your role:
        - Create well-structured reports
        - Write clearly and engagingly
        - Use proper formatting (markdown)
        - Include citations and sources
        - Organize information logically
        
        Structure reports with:
        ## Executive Summary
        ## Key Findings
        ## Detailed Analysis
        ## Conclusions
        ## Sources

# Optional: RAG for researcher
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334

embedders:
  embedder:
    type: "ollama"
    model: "nomic-embed-text"
```

---

## Step 3: Start the System

```bash
# Set API keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Start Hector
hector serve --config research-system.yaml
```

Output:
```
Hector server listening on :8080
Agent registered: research_coordinator
Agent registered: researcher
Agent registered: analyst
Agent registered: writer
```

---

## Step 4: Test Your Research System

### Simple Research Task

```bash
hector call --config research-system.yaml research_coordinator "Research the impact of AI on healthcare"
```

**What happens:**

1. **Coordinator** receives request and creates TODO list:
   ```
   - Gather healthcare AI information (researcher)
   - Analyze impact and trends (analyst)
   - Write comprehensive report (writer)
   ```

2. **Coordinator** calls **researcher**:
   ```
   agent_call("researcher", "Research AI applications in healthcare")
   ```

3. **Researcher** gathers information:
   - Searches for relevant data
   - Compiles findings
   - Returns structured research

4. **Coordinator** calls **analyst**:
   ```
   agent_call("analyst", "Analyze these findings: [research data]")
   ```

5. **Analyst** analyzes:
   - Identifies key trends
   - Draws conclusions
   - Provides insights

6. **Coordinator** calls **writer**:
   ```
   agent_call("writer", "Write report based on: [analysis]")
   ```

7. **Writer** creates report:
   - Structures content
   - Writes clearly
   - Saves to file

8. **Coordinator** delivers final result to user

### Complex Research Task

```bash
hector call --config research-system.yaml research_coordinator \
  "Research quantum computing developments in 2024, analyze market trends, and create an investment report"
```

The coordinator automatically:
- Breaks down into sub-tasks
- Delegates to appropriate specialists
- Ensures quality at each step
- Synthesizes final result

---

## Step 5: Enhance Your System

### Add More Specialists

```yaml
agents:
  # ... existing agents ...
  
  fact_checker:
    name: "Fact Checker"
    llm: "gpt-4o"
    prompt:
      system_prompt: |
        You verify facts and check sources for accuracy.
        Flag any dubious claims or weak sources.
  
  editor:
    name: "Editor"
    llm: "claude"
    prompt:
      system_prompt: |
        You review and improve written content.
        Ensure clarity, grammar, and flow.
```

Update coordinator:
```yaml
research_coordinator:
  prompt:
    system_prompt: |
      TEAM MEMBERS:
      - researcher
      - fact_checker  # NEW
      - analyst
      - editor        # NEW
      - writer
```

### Add External Data Sources

```yaml
researcher:
  tools:
    - "search"
    - "write_file"
    - "execute_command"  # For API calls
  
# Allow calling external APIs
tools:
  execute_command:
    type: command
    
    # Permissive defaults: allows all commands (sandboxed for security)
    # Only restrict if needed:
    # allowed_commands: ["curl", "wget"]
```

### Enable Long-Term Memory

```yaml
agents:
  research_coordinator:
    database: "qdrant"
    embedder: "embedder"
    memory:
      longterm:
        
        storage_scope: "session"
```

Now the coordinator remembers past research!

---

## Advanced Patterns

### Hierarchical Coordination

Add team leads:

```yaml
agents:
  master_coordinator:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  research_lead:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  writing_lead:
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  # Workers
  researcher1:
  researcher2:
  writer1:
  writer2:
```

### Parallel Research

Coordinator can call multiple researchers simultaneously:

```yaml
research_coordinator:
  prompt:
    system_prompt: |
      You can call multiple researchers in parallel
      for faster information gathering.
      
      Example:
      - Call researcher1 for medical journals
      - Call researcher2 for news articles  
      - Call researcher3 for academic papers
      
      All in parallel, then synthesize results.
```

### Quality Control Loop

```yaml
agents:
  qa_reviewer:
    llm: "claude"
    prompt:
      system_prompt: |
        Review the final report for:
        - Accuracy
        - Completeness
        - Clarity
        - Source quality
        
        If issues found, send back for revision.
```

---

## Monitoring & Debugging

### See Agent Coordination

```yaml
agents:
  research_coordinator:
    reasoning:
      show_tool_execution: true
      show_debug_info: true
```

Output shows:
```
[Coordinator] Breaking task into sub-tasks...
[Coordinator] TODO: Research healthcare AI
[Coordinator] TODO: Analyze findings
[Coordinator] TODO: Write report
[Coordinator] Calling researcher...
[Tool] agent_call("researcher", "Research healthcare AI")
[Researcher] Searching for information...
[Researcher] Found 5 relevant sources...
[Tool Result] Research complete
[Coordinator] Calling analyst...
...
```

### Track Performance

```bash
# Enable JSON logging
hector serve --config research-system.yaml --log-format json

# Monitor agent calls
grep "agent_call" logs/hector.log | jq
```

---

## Production Tips

### Use Appropriate LLMs

```yaml
llms:
  fast:
    type: "openai"
    model: "gpt-4o-mini"  # Fast, cheap
  
  smart:
    type: "anthropic"
    model: "claude-opus-4"  # Smart, expensive

agents:
  coordinator:
    llm: "fast"  # Coordination doesn't need best model
  
  analyst:
    llm: "smart"  # Analysis benefits from better model
  
  writer:
    llm: "smart"  # Writing benefits from better model
```

### Set Iteration Limits

```yaml
agents:
  research_coordinator:
    reasoning:
      max_iterations: 20  # Coordinator should finish quickly
  
  researcher:
    reasoning:
      max_iterations: 50  # Researchers may need more iterations
```

### Handle Errors Gracefully

```yaml
agents:
  research_coordinator:
    prompt:
      system_prompt: |
        If a specialist fails:
        1. Try the task a different way
        2. Call a backup specialist if available
        3. Provide partial results if necessary
        4. Always explain what went wrong
```

---

## Example Workflows

### Academic Research

```bash
hector call --config research-system.yaml research_coordinator \
  "Research machine learning interpretability, analyze recent papers, and write a literature review"
```

### Market Analysis

```bash
hector call --config research-system.yaml research_coordinator \
  "Research electric vehicle market, analyze competitors, and create market entry strategy"
```

### Technical Investigation

```bash
hector call --config research-system.yaml research_coordinator \
  "Research microservices architectures, analyze trade-offs, and recommend approach for our system"
```

---

## Comparison with Single Agent

**Single Agent Approach:**
```yaml
agents:
  generalist:
    tools: ["search", "write_file", "execute_command"]
```

- Does everything itself
- Mediocre at all tasks
- No specialization
- Simpler but less capable

**Multi-Agent Approach:**
```yaml
agents:
  coordinator:
    tools: ["agent_call"]
  researcher:
    tools: ["search"]
  analyst:
    tools: []
  writer:
    tools: ["write_file"]
```

- Each agent specialized
- Expert-level results
- Modular and extensible
- More complex but more capable

---

## Next Steps

- **[Multi-Agent Orchestration](../core-concepts/multi-agent.md)** - Understand the concepts
- **[Reasoning Strategies](../core-concepts/reasoning.md)** - Supervisor strategy details
- **[Integrate External Agents](integrate-external-agents.md)** - Add external A2A agents
- **[Deploy to Production](deploy-production.md)** - Production deployment

---

## Related Topics

- **[Agent Overview](../core-concepts/overview.md)** - Understanding agents
- **[Tools](../core-concepts/tools.md)** - agent_call tool
- **[Configuration Reference](../reference/configuration.md)** - All orchestration options

---

## Complete Example

See `configs/orchestrator-example.yaml` in the Hector repository for a complete multi-agent configuration.

