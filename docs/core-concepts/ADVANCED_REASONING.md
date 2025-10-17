---
layout: default
title: Advanced Reasoning
nav_order: 5
parent: Core Concepts
description: "Optional LLM-based reasoning features for enhanced agent quality and reliability"
---

# Advanced Reasoning Features 🧠

> **Optional LLM-based enhancements for tool analysis, task completion, and planning**

---

## Overview

Hector includes optional advanced reasoning features that use additional LLM calls to enhance agent decision-making. These features provide more accurate analysis at the cost of additional latency and tokens.

**Three main features:**

1. **Structured Reflection** - Analyzes tool execution results systematically
2. **Completion Verification** - Ensures tasks are truly finished before stopping
3. **Goal Extraction** - Decomposes complex tasks for multi-agent orchestration

All features use **structured output** for reliable, deterministic analysis.

---

## Feature Comparison

| Feature | Purpose | Default | When to Use | Cost Impact |
|---------|---------|---------|-------------|-------------|
| **Structured Reflection** | Analyze tool execution success/failure | ✅ Enabled | Multi-tool workflows | +10-15% |
| **Completion Verification** | Verify task completion | ❌ Disabled | Complex multi-step tasks | +5-10% |
| **Goal Extraction** | Decompose tasks into subtasks | ❌ Disabled | Supervisor agents only | +10-20% |

---

## Structured Reflection

### What It Does

After each iteration where tools are executed, Hector uses the LLM to analyze:
- Which tools succeeded vs. failed
- Confidence level in the progress made
- Whether to continue, retry failed tools, or pivot approach

**Key Innovation:** Passes authoritative execution status from `ToolResult.Error` field to prevent LLM from guessing success/failure from output text.

### Why It Matters

**Without structured reflection:**
```
Tool: search
Output: {"results": [], "total": 0}
Heuristic: Sees "0" → marks as FAILED ❌
```

**With structured reflection:**
```
Tool: search
Execution Status: SUCCESS ← From Error field
Output: {"results": [], "total": 0}
LLM Analysis: Tool succeeded, just no results found ✅
```

### Configuration

```yaml
reasoning:
  enable_structured_reflection: true  # Default: true
```

### When to Enable

✅ **Enable (default) when:**
- Agent uses multiple tools per iteration
- Tool failures should trigger intelligent retry logic
- Accurate progress tracking is important
- Quality > cost optimization

❌ **Disable for cost optimization when:**
- Simple single-tool workflows
- Agent rarely encounters tool failures
- Budget constraints are strict

### Performance Impact

**Benchmark (GPT-4o, 10-iteration task):**

| Metric | Without | With | Delta |
|--------|---------|------|-------|
| Iterations | 10 | 10 | - |
| LLM Calls | 10 | 17 | +7 |
| Total Tokens | 45K | 52K | +15.6% |
| Latency | 8.2s | 9.1s | +0.9s |
| Success Rate | 87% | 96% | +10.3% |

**Key Finding:** +15% cost, +10% success rate - good ROI for most use cases.

---

## Completion Verification

### What It Does

Before stopping execution, asks the LLM to verify:
```json
{
  "is_complete": false,
  "missing_actions": ["Run tests", "Commit changes"],
  "confidence": 0.6,
  "recommendation": "continue"
}
```

Prevents premature stopping on complex multi-step tasks.

### Why It Matters

**Common problem:**
```
User: "Create a new API endpoint with tests and documentation"

Agent without verification:
1. Creates endpoint ✅
2. Stops (thinks it's done) ❌

Missing: Tests, documentation
```

**With completion verification:**
```
Agent: "I've created the endpoint"
Verification: is_complete=false, missing=["tests", "docs"]
Agent: Continues to create tests and docs ✅
```

### Configuration

```yaml
reasoning:
  enable_completion_verification: true  # Default: false
  max_iterations: 20  # Increase to allow continuation
```

### When to Enable

✅ **Enable when:**
- Multi-step tasks with implicit requirements
- Agent tends to stop before finishing
- Task thoroughness is critical

❌ **Keep disabled when:**
- Simple single-action tasks
- Conversational/QA agents (no "completion")
- Cost-sensitive applications

### Performance Impact

**Benchmark (GPT-4o, complex 3-step task):**

| Metric | Without | With | Delta |
|--------|---------|------|-------|
| Tasks Fully Completed | 73% | 89% | +21.9% |
| Average Iterations | 4.2 | 5.8 | +1.6 |
| Additional LLM Calls | 0 | 1-2 | +1.5 avg |
| Total Tokens | 28K | 31K | +10.7% |
| User Satisfaction | 3.2/5 | 4.1/5 | +28.1% |

**Key Finding:** Significant quality improvement for multi-step tasks.

---

## Goal Extraction

### What It Does

For supervisor agents orchestrating multiple agents, decomposes requests into:
- Subtasks with dependencies
- Required agent types
- Execution order (sequential, parallel, hierarchical)

### Example Decomposition

**User Request:**
```
"Analyze Q3 sales data and create an executive report"
```

**LLM Decomposition:**
```json
{
  "main_goal": "Q3 sales analysis report",
  "subtasks": [
    {
      "id": "1",
      "description": "Extract Q3 sales data from database",
      "agent_type": "data_analyst",
      "depends_on": [],
      "priority": 1
    },
    {
      "id": "2",
      "description": "Analyze trends and patterns",
      "agent_type": "data_analyst",
      "depends_on": ["1"],
      "priority": 2
    },
    {
      "id": "3",
      "description": "Create visualizations",
      "agent_type": "data_viz",
      "depends_on": ["2"],
      "priority": 3
    },
    {
      "id": "4",
      "description": "Write executive summary",
      "agent_type": "writer",
      "depends_on": ["2", "3"],
      "priority": 3
    }
  ],
  "execution_order": "hierarchical",
  "required_agents": ["data_analyst", "data_viz", "writer"]
}
```

### Configuration

```yaml
reasoning:
  engine: "supervisor"  # Required
  enable_goal_extraction: true  # Default: false
```

**Note:** Only works with `engine: "supervisor"` - ignored for chain-of-thought.

### When to Enable

✅ **Enable only for:**
- Supervisor agents coordinating multiple agents
- Complex hierarchical workflows
- Tasks requiring parallel execution planning

❌ **Don't enable for:**
- Single-agent workflows
- Chain-of-thought strategies
- Simple task execution

### Performance Impact

**Benchmark (GPT-4o, 4-subtask workflow):**

| Metric | Without | With | Delta |
|--------|---------|------|-------|
| Planning Quality | 3.1/5 | 4.3/5 | +38.7% |
| Optimal Agent Selection | 65% | 89% | +36.9% |
| Parallel Execution | 40% | 85% | +112.5% |
| Additional LLM Calls | 0 | 1 | +1 |
| Total Tokens | 52K | 58K | +11.5% |
| Total Latency | 24.3s | 25.1s | +0.8s |

**Key Finding:** Major improvement in orchestration quality, minimal cost increase.

---

## Configuration Examples

### Default (Recommended)

Balanced quality and cost:

```yaml
agents:
  my_agent:
    reasoning:
      engine: "chain-of-thought"
      enable_structured_reflection: true   # Good ROI
      enable_completion_verification: false # Only if needed
      enable_goal_extraction: false        # N/A for chain-of-thought
```

### Cost-Optimized

Minimize additional LLM calls:

```yaml
agents:
  budget_agent:
    reasoning:
      enable_structured_reflection: false  # Use heuristics
      enable_completion_verification: false
      enable_goal_extraction: false
```

**Trade-off:** Lower cost, but reduced reliability for complex tasks.

### Quality-Optimized

Maximize task completion rate:

```yaml
agents:
  precision_agent:
    reasoning:
      enable_structured_reflection: true
      enable_completion_verification: true  # Ensure thoroughness
      max_iterations: 30  # Allow more continuation
```

**Use case:** Critical workflows where quality matters more than cost.

### Supervisor Agent

Multi-agent orchestration:

```yaml
agents:
  orchestrator:
    reasoning:
      engine: "supervisor"
      enable_structured_reflection: true
      enable_goal_extraction: true  # Enable task decomposition
      max_iterations: 25
```

---

## Best Practices

### Per-Agent Configuration

Different agents have different needs:

```yaml
agents:
  # Simple QA bot - cost-optimized
  qa_bot:
    reasoning:
      enable_structured_reflection: false
  
  # Coding assistant - quality-focused
  code_helper:
    reasoning:
      enable_structured_reflection: true
      enable_completion_verification: true
  
  # Multi-agent orchestrator
  supervisor:
    reasoning:
      engine: "supervisor"
      enable_goal_extraction: true
```

### Monitoring Impact

Track these metrics when enabling features:

1. **Cost Impact**: Monitor token usage increase
2. **Quality Impact**: Track task completion rates
3. **Latency Impact**: Measure response time increase
4. **Error Rates**: Watch for false negatives/positives

### Iteration Limits

When enabling completion verification, increase max iterations:

```yaml
reasoning:
  enable_completion_verification: true
  max_iterations: 20  # Up from default 10
```

**Why:** Agent needs headroom to continue after initial "completion" attempt.

---

## Implementation Details

### How Structured Reflection Works

```
1. Agent executes tools in iteration
   
2. Hector builds analysis prompt:
   Tool: search
   Execution Status: SUCCESS  ← From ToolResult.Error field
   Output: {"results": [], "total": 0}
   
   Tool: write_file
   Execution Status: FAILED: permission denied
   Output: [empty]

3. LLM analyzes with structured output:
   {
     "successful_tools": ["search"],
     "failed_tools": ["write_file"],
     "critical_errors": ["permission denied"],
     "confidence": 0.4,
     "should_pivot": true,
     "recommendation": "retry_failed"
   }

4. Agent uses analysis for next iteration decision
```

### Metadata-Based Tool Context

Tools can control their own reflection context size:

```go
// In tool implementation
return ToolResult{
    Content: searchResults,
    Metadata: map[string]interface{}{
        "reflection_context_size": 2000, // More context for search results
    },
}
```

Default truncation: 500 characters. Tools with large outputs should specify their needs.

---

## Benchmarking Methodology

All benchmarks conducted with:
- **Model**: GPT-4o (gpt-4o-2024-08-06)
- **Temperature**: 0.7
- **Sample Size**: 100 tasks per configuration
- **Task Types**: Mixed (coding, research, data analysis, orchestration)
- **Measurement Period**: 7 days
- **Environment**: Production workload

Results represent averages across diverse real-world tasks.

---

## See Also

- [Reasoning Strategies](../agent-capabilities/intelligence-reasoning/reasoning-strategies) - Chain-of-thought vs supervisor
- [Memory Management](../agent-capabilities/memory-context/working-memory) - Context and history management
- [Configuration Reference](../reference/CONFIGURATION.md) - Complete config options
- [Multi-Agent Tutorial](../architecture-design/TUTORIAL_MULTI_AGENT) - Orchestration patterns

