---
title: External Agents
description: Integrate external A2A agents with Hector seamlessly
---

# External A2A Agents Guide

## External Agent Support

Hector **fully supports external A2A agents** through **pure YAML configuration**. Any A2A-compliant agent can be integrated into your multi-agent system without writing any code.

**Key Benefits:**
- **Agent Ecosystem Participation** - Connect to the emerging "agent internet"
- **Enterprise Interoperability** - Integrate partner and vendor agents seamlessly
- **Declarative Integration** - Define external agents in YAML alongside native ones
- **Zero Code Required** - No API integration code, no custom connectors

---

## Quick Start: YAML Configuration (Recommended)

### Define External Agents in Your Config

```yaml
agents:
  # Native agent (runs locally)
  local_researcher:
    name: "Local Research Agent"
    llm: "gpt-4"
    reasoning:
      engine: "chain-of-thought"
  
  # External A2A agent (remote service)
  partner_specialist:
    type: "a2a"  # Marks this as external
    name: "Partner Specialist"
    url: "https://partner-ai.com/agents/specialist"
    # No LLM, reasoning, or tools - external agent has its own!
  
  # Another external agent
  translation_service:
    type: "a2a"
    name: "Translation Service"
    url: "https://translate.ai/agents/translator"
  
  # Orchestrator that uses ALL agents
  orchestrator:
    name: "Hybrid Orchestrator"
    llm: "gpt-4"
    reasoning:
      engine: "supervisor"
    tools:
      - agent_call  # Can call local AND remote agents!
```

**That's it!** Start the server and all agents (native + external) are available:

```bash
hector serve --config your-config.yaml

# All agents are available via same API
hector list  # Shows all: native + external
hector call partner_specialist "Your task"
hector call orchestrator "Complex multi-agent task"
```

**See:** `configs/mixed-agents-example.yaml` for a complete working example.

---

## Use Cases

### 1. Enterprise Integration
```yaml
agents:
  # Your agents
  internal_analyst:
    llm: "gpt-4"
    # ...
  
  # Partner's agents
  vendor_data_service:
    type: "a2a"
    url: "https://vendor.com/agents/data-api"
  
  legal_compliance_checker:
    type: "a2a"
    url: "https://legal-ai.com/agents/compliance"
```

**Benefit:** Integrate vendor services without custom API code. Pure A2A interoperability.

### 2. Agent Internet Ecosystem
```yaml
agents:
  # Community agents
  research_agent:
    type: "a2a"
    url: "https://research-agents.io/agents/scholar"
  
  fact_checker:
    type: "a2a"
    url: "https://factcheck.ai/agents/verifier"
  
  # Your orchestrator coordinates them all
  my_orchestrator:
    llm: "gpt-4"
    tools: [agent_call]
```

**Benefit:** Participate in the broader agent ecosystem. Discover and use community agents.

### 3. Organizational Interoperability
```yaml
agents:
  # Engineering team's agents
  code_reviewer:
    type: "a2a"
    url: "https://eng-agents.company.com/agents/reviewer"
  
  # Data team's agents
  data_analyst:
    type: "a2a"
    url: "https://data-agents.company.com/agents/analyst"
  
  # Your cross-functional orchestrator
  project_coordinator:
    llm: "gpt-4"
    tools: [agent_call]
```

**Benefit:** Different teams can run their own Hector instances, yet orchestrate across them declaratively.

---

## How It Works

### Architecture

```
Hector Orchestrator
    ↓ agent_call tool
AgentRegistry (a2a.Agent interface)
    ↓
    ├─ Native Agents (in-process, fast)
    │  └─ agent.Agent implements a2a.Agent
    │
    └─ External A2A Agents (remote, HTTP)
       └─ agent.A2AAgent wraps a2a.Client
          └─ Calls external A2A server
```

**Key Point:** Both native and external agents implement the **same `a2a.Agent` interface**. The orchestrator doesn't know (or care) about the difference!

---

## Quick Example

### Step 1: Discover External Agent

```go
import (
    "context"
    "github.com/kadirpekel/hector/a2a"
    "github.com/kadirpekel/hector/agent"
)

// Create A2A client
client := a2a.NewClient(&a2a.ClientConfig{})

// Discover external agent
ctx := context.Background()
externalAgent, err := agent.NewA2AAgentFromURL(
    ctx,
    "https://external-service.com/agents/expert",
    client,
)
```

### Step 2: Register in Registry

```go
// Create agent registry
registry := agent.NewAgentRegistry()

// Register external agent (same as native!)
err := registry.RegisterAgent(
    "external_expert",
    externalAgent,  // Implements a2a.Agent
    &config.AgentConfig{
        Name: "External Expert",
        Description: "External specialist agent",
    },
    []string{"expert_advice"},
)
```

### Step 3: Use in Orchestration

```yaml
# orchestrator-with-external.yaml
agents:
  orchestrator:
    name: "Orchestrator"
    tools:
      - agent_call
    prompt:
      system_role: |
        Available agents:
        - researcher (native)
        - analyst (native)
        - external_expert (remote A2A agent)
        
        Use agent_call to delegate tasks.
```

**That's it!** The orchestrator can now call all three agents identically.

---

## Advanced: Programmatic Registration

While YAML configuration is recommended, you can also register external agents programmatically:

### Complete Example: Programmatic Registration

```go
package main

import (
    "context"
    "fmt"
    "github.com/kadirpekel/hector/a2a"
    "github.com/kadirpekel/hector/agent"
    "github.com/kadirpekel/hector/config"
)

func main() {
    ctx := context.Background()
    
    // 1. Create A2A client
    client := a2a.NewClient(&a2a.ClientConfig{})
    
    // 2. Discover external translation agent
    translatorAgent, err := agent.NewA2AAgentFromURL(
        ctx,
        "https://translation-service.com/agents/translator",
        client,
    )
    if err != nil {
        panic(err)
    }
    
    // 3. Create agent registry
    registry := agent.NewAgentRegistry()
    
    // 4. Register external agent
    err = registry.RegisterAgent(
        "translator",
        translatorAgent,
        &config.AgentConfig{
            Name:        "Translation Agent",
            Description: "Translates text between languages",
        },
        []string{"translation", "language"},
    )
    if err != nil {
        panic(err)
    }
    
    // 5. Now create orchestrator agent with access to registry
    // The orchestrator can call the translator via agent_call tool!
    
    // 6. Register agent_call tool with registry
    agentCallTool := agent.NewAgentCallTool(registry)
    // ... register tool with component manager ...
    
    fmt.Println("✅ External agent registered and ready!")
}
```

---

## Implementation Details

### A2AAgent Class

**Location:** `agent/a2a_agent.go`

```go
// A2AAgent wraps an external A2A agent
type A2AAgent struct {
    agentCard *a2a.AgentCard
    client    *a2a.Client
}

// Implements a2a.Agent interface
func (a *A2AAgent) GetAgentCard() *a2a.AgentCard
func (a *A2AAgent) ExecuteTask(ctx, request) (*a2a.TaskResponse, error)
func (a *A2AAgent) ExecuteTaskStreaming(ctx, request) (<-chan *a2a.StreamChunk, error)
```

**Key Features:**
- Pure A2A protocol communication
- Same interface as native agents
- Transparent to orchestrator
- Full TaskRequest/TaskResponse support

### AgentRegistry

**Location:** `agent/registry.go`

```go
type AgentRegistry struct {
    instances map[string][]a2a.Agent  // Stores ANY a2a.Agent
}

// Accepts both native and external agents
func (r *AgentRegistry) RegisterAgent(
    name string,
    agent a2a.Agent,  // Interface, not concrete type!
    config *config.AgentConfig,
    capabilities []string,
) error
```

**Key Features:**
- Stores `a2a.Agent` interface (not concrete types)
- Works with native `agent.Agent`
- Works with external `agent.A2AAgent`
- Completely transparent

### Agent_call Tool

**Location:** `agent/agent_call_tool.go`

```go
func (t *AgentCallTool) Execute(ctx context.Context, args map[string]interface{}) {
    // Get agent from registry (could be native OR external)
    targetAgent, _ := t.registry.GetAgent(agentName)
    
    // Call via pure A2A protocol (works for both!)
    taskResponse, _ := targetAgent.ExecuteTask(ctx, taskRequest)
}
```

**Key Features:**
- Uses `a2a.Agent` interface
- Doesn't care if native or external
- Pure protocol communication
- Same delegation logic for all agents

---

## Authentication

### Bearer Token

```go
client := a2a.NewClient(&a2a.ClientConfig{
    AuthToken: "your-bearer-token",
})

externalAgent, _ := agent.NewA2AAgentFromURL(ctx, url, client)
```

### API Key

```go
client := a2a.NewClient(&a2a.ClientConfig{
    APIKey: "your-api-key",
})

externalAgent, _ := agent.NewA2AAgentFromURL(ctx, url, client)
```

### Custom Headers

```go
client := a2a.NewClient(&a2a.ClientConfig{
    Headers: map[string]string{
        "X-Custom-Auth": "value",
    },
})

externalAgent, _ := agent.NewA2AAgentFromURL(ctx, url, client)
```

---

## Configuration-Based Registration (Future)

```yaml
# Future: Configure external agents in YAML
agents:
  # Native agent
  researcher:
    name: "Research Agent"
    llm: "gpt-4o"
    # ... native agent config ...
  
  # External A2A agent
  translator:
    type: "a2a_external"
    url: "https://translation-service.com/agents/translator"
    auth:
      type: "bearer"
      token: "${TRANSLATION_API_KEY}"
  
  # Orchestrator can use both!
  orchestrator:
    name: "Orchestrator"
    tools:
      - agent_call
```

**Status:** Not yet implemented, but architecture supports it!

---

## Verification

### Test External Agent Support

```go
// 1. Create mock A2A agent
mockAgent := &MockA2AAgent{}  // Implements a2a.Agent

// 2. Register in registry
registry.RegisterAgent("mock", mockAgent, config, []string{"test"})

// 3. Get back from registry
retrieved, _ := registry.GetAgent("mock")

// 4. Call via agent_call tool
tool := agent.NewAgentCallTool(registry)
result, _ := tool.Execute(ctx, map[string]interface{}{
    "agent": "mock",
    "task":  "test task",
})

// ✅ Works! External agents fully supported!
```

---

## Key Benefits

### 1. Transparency
- Orchestrator doesn't know if agent is native or external
- Same `agent_call` tool works for both
- Same A2A protocol for all communication

### 2. Flexibility
- Mix native (fast) and external (specialized) agents
- Add external services without code changes
- Replace native with external (or vice versa) easily

### 3. Scalability
- Distribute load across external services
- Use specialized external agents when needed
- Keep frequently-used agents native for speed

### 4. Interoperability
- Any A2A-compliant agent works
- Contribute to A2A ecosystem
- Use agents from other providers

---

## Discovery & Registration Flow

```
1. A2A Discovery
   └─ GET https://external.com/agents/expert
   └─ Returns: AgentCard (capabilities, endpoints)

2. Create A2AAgent
   └─ NewA2AAgentFromURL(url, client)
   └─ Wraps AgentCard + Client

3. Register in Registry
   └─ registry.RegisterAgent(name, agent, config, caps)
   └─ Stored as a2a.Agent interface

4. Orchestrator Uses
   └─ agent_call tool
   └─ registry.GetAgent(name)
   └─ agent.ExecuteTask(request) → Pure A2A protocol
   └─ Returns TaskResponse

5. Response to User
   └─ Orchestrator synthesizes
   └─ User gets unified response
```

**All automatic, all transparent!** ✅
