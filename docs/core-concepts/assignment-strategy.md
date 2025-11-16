# Assignment Strategy: Consistent Access Control

## Overview

Hector uses a **consistent assignment strategy** across all resource types (Tools, Document Stores, and Sub-Agents). This ensures predictable behavior and makes configuration intuitive.

## The Three States

All assignment fields follow the same pattern:

| Configuration | Access | Behavior |
|--------------|--------|----------|
| `nil`/omitted | **All resources** | Permissive default - agent has access to all resources of this type |
| `[]` (explicitly empty) | **No resources** | Explicit restriction - agent has no access to any resources of this type |
| `["item1", ...]` | **Only those resources** | Scoped access - agent can only access the explicitly listed resources |

## Resource Types

### 1. Tools (`agent.tools`)

Controls which tools the agent can use.

```yaml
agents:
  # Access all tools (permissive default)
  general_assistant:
    # tools: not specified → accesses all tools from registry
  
  # No tools (explicit restriction)
  isolated_agent:
    tools: []                    # Explicitly empty = no tools
  
  # Scoped access (explicit assignment)
  file_agent:
    tools:
      - "read_file"
      - "write_file"
      - "search_replace"
```

**Behavior**:
- `nil`/omitted: Agent has access to **ALL** tools from the registry (including MCP tools, default tools, etc.)
- `[]`: Agent has **NO** tools (explicit restriction)
- `[tools...]`: Agent can **ONLY** use the explicitly listed tools

---

### 2. Document Stores (`agent.document_stores`)

Controls which document stores the agent can search.

```yaml
document_stores:
  knowledge_base:
    source: sql
    # ... config ...
  internal_docs:
    source: directory
    # ... config ...

agents:
  # Access all stores (permissive default)
  general_assistant:
    # document_stores: not specified → accesses all stores
  
  # No access (explicit restriction)
  isolated_agent:
    document_stores: []          # Explicitly empty = no access
  
  # Scoped access (explicit assignment)
  security_agent:
    document_stores:
      - "knowledge_base"
      - "internal_docs"
```

**Behavior**:
- `nil`/omitted: Agent has access to **ALL** registered document stores
- `[]`: Agent has **NO access** to any document stores (search tool not created)
- `[stores...]`: Agent can **ONLY access** the explicitly listed stores

**Auto-creation**: The search tool is automatically created if the agent has document store access.

---

### 3. Sub-Agents (`agent.sub_agents`)

Controls which agents this agent can call via the `agent_call` tool.

```yaml
agents:
  # Access all agents (permissive default)
  coordinator:
    # sub_agents: not specified → can call all agents (honoring visibility)
  
  # No agents (explicit restriction)
  isolated_agent:
    sub_agents: []               # Explicitly empty = cannot call any agents
  
  # Scoped access (explicit assignment)
  research_coordinator:
    sub_agents:
      - "researcher"
      - "analyst"
      - "writer"
```

**Behavior**:
- `nil`/omitted: Agent can call **ALL** available agents (honoring visibility - excludes "internal" agents by default)
- `[]`: Agent **CANNOT call** any agents (explicit restriction)
- `[agents...]`: Agent can **ONLY call** the explicitly listed agents

**Auto-creation**: The `agent_call` tool is automatically created if the agent has sub-agent access.

**Visibility**: When `nil` (all agents), agents with `visibility: "internal"` are excluded by default.

---

## Why This Pattern?

### 1. **Intuitive Defaults**
- `nil` = "I don't care, give me everything" (permissive)
- `[]` = "I explicitly want nothing" (restrictive)
- `[items...]` = "I want only these specific items" (scoped)

### 2. **Security by Explicit Restriction**
- To restrict access, you must explicitly set `[]`
- `nil` is permissive (good for development, easy to restrict later)

### 3. **Consistency**
- Same pattern across all resource types
- Easy to remember and reason about
- Predictable behavior

### 4. **Flexibility**
- Start with `nil` (all access) for quick setup
- Gradually restrict with `[]` or `[items...]` as needed
- Easy to audit: explicit lists show exactly what's allowed

---

## Examples

### Example 1: General-Purpose Agent

```yaml
agents:
  assistant:
    # All tools, all document stores, all sub-agents
    # tools: not specified → all tools
    # document_stores: not specified → all stores
    # sub_agents: not specified → all agents
```

**Access**: Everything (permissive default)

---

### Example 2: Isolated Agent

```yaml
agents:
  isolated:
    tools: []                    # No tools
    document_stores: []          # No document stores
    sub_agents: []               # No sub-agents
```

**Access**: Nothing (explicit restrictions)

---

### Example 3: Scoped Specialist Agent

```yaml
agents:
  security_specialist:
    tools:
      - "read_file"              # Only file reading
      - "grep_search"            # Only search
    document_stores:
      - "security_policies"      # Only security docs
      - "compliance_docs"        # Only compliance docs
    sub_agents:
      - "security_analyst"       # Only security analyst
```

**Access**: Only explicitly listed resources (scoped)

---

### Example 4: Mixed Configuration

```yaml
agents:
  coordinator:
    tools: []                    # No tools (orchestration only)
    document_stores: []          # No document stores
    sub_agents:                  # All agents (delegation)
      # not specified → all agents
```

**Access**: No tools/stores, but can call all agents

---

## Migration Guide

### From Old Behavior (Sub-Agents)

**Before** (inconsistent):
- `sub_agents: nil` or `[]` = all agents
- `sub_agents: [agents...]` = scoped

**After** (consistent):
- `sub_agents: nil` = all agents
- `sub_agents: []` = no agents
- `sub_agents: [agents...]` = scoped

**Action Required**: If you had `sub_agents: []` expecting "all agents", change to omit the field or set to `nil`.

---

## Best Practices

### 1. Start Permissive, Restrict Gradually

```yaml
# Development: Start with all access
agents:
  dev_agent:
    # Everything accessible by default

# Production: Restrict explicitly
agents:
  prod_agent:
    tools:
      - "read_file"              # Only safe tools
    document_stores:
      - "public_docs"            # Only public docs
    sub_agents: []               # No sub-agents
```

### 2. Use Explicit Lists for Security

```yaml
# Good: Explicit list shows exactly what's allowed
agents:
  secure_agent:
    tools:
      - "read_file"
      - "search"
    document_stores:
      - "approved_docs"
```

### 3. Document Restrictions

```yaml
agents:
  restricted_agent:
    # Security: No tools, no stores, no sub-agents
    tools: []
    document_stores: []
    sub_agents: []
```

### 4. Use Empty Lists for Isolation

```yaml
# Isolated agent that only uses its LLM
agents:
  llm_only:
    tools: []                    # No tools
    document_stores: []          # No RAG
    sub_agents: []               # No delegation
```

---

## Summary

Hector's assignment strategy is **consistent across all resource types**:

- **`nil`/omitted** = All resources (permissive default)
- **`[]`** = No resources (explicit restriction)
- **`[items...]`** = Only those resources (scoped access)

This pattern applies to:
- ✅ **Tools** (`agent.tools`)
- ✅ **Document Stores** (`agent.document_stores`)
- ✅ **Sub-Agents** (`agent.sub_agents`)

The consistent pattern makes configuration intuitive, predictable, and secure.

