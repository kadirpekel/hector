---
title: Integrate External A2A Agents
description: Connect to remote A2A agents in your multi-agent workflows
---

# How to Integrate External A2A Agents

Connect your Hector agents to external A2A-compliant agents (v0.3.0 or compatible) running on other servers or services.

**Time:** 15 minutes  
**Difficulty:** Intermediate

---

## What You'll Learn

- Discover external A2A agents
- Configure external agent connections
- Handle authentication (Bearer, API Key, Basic)
- Use external agents in orchestration
- Debug external agent integration

---

## Understanding A2A Integration

**A2A (Agent-to-Agent) Protocol** enables agents to call other agents across networks, regardless of implementation.

**Use cases:**
- **Specialized services** - Call domain-specific agents (legal, medical, financial)
- **Team collaboration** - Connect agents from different teams
- **Commercial services** - Integrate with A2A v0.3.0 compliant SaaS
- **Distributed systems** - Build agent networks across infrastructure
- **Multi-vendor interoperability** - Mix agents from different A2A implementations

---

## Quick Example

### Step 1: Discover External Agent

External A2A agents expose an agent card:

```bash
curl https://external-agent.example.com/.well-known/agent.json
```

Response:
```json
{
  "name": "Research Specialist",
  "description": "Advanced research and analysis agent",
  "version": "1.0.0",
  "capabilities": ["research", "analysis", "reporting"],
  "authentication": {
    "types": ["bearer", "api_key"]
  },
  "endpoints": {
    "grpc": "grpc://external-agent.example.com:8080",
    "rest": "https://external-agent.example.com/api"
  }
}
```

### Step 2: Configure in Hector

Add to `config.yaml`:

```yaml
agents:
  # Local coordinator
  coordinator:
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
    
    prompt:
      system_role: |
        You coordinate work between local and external agents.
        
        TEAM MEMBERS:
        - local_researcher: Internal research agent
        - external_specialist: External research specialist
  
  # Local agent
  local_researcher:
    llm: "gpt-4o"
    tools: ["search"]
  
  # External A2A agent
  # The URL can point to:
  # 1. Service base URL (e.g., http://service.com) - auto-discovers agents
  # 2. Agent card URL (e.g., http://service.com/.well-known/agent.json)
  # 3. Agent-specific URL (e.g., http://service.com/v1/agents/specialist)
  external_specialist:
    type: "a2a"
    url: "https://external-agent.example.com"
    credentials:
      type: "bearer"
      token: "${EXTERNAL_AGENT_TOKEN}"

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

**Note:** Hector's `UniversalA2AClient` automatically:
1. Discovers the agent card from the provided URL
2. Detects supported transports (gRPC, REST, JSON-RPC)
3. Selects the optimal transport
4. Handles authentication across all transports

The config key (`external_specialist`) is used as the remote agent ID by default. If the remote agent has a different ID, use `target_agent_id`:

```yaml
agents:
  my_research_service:  # Local name (flexible)
    type: "a2a"
    url: "https://external-agent.example.com"
    target_agent_id: "research_specialist"  # Remote agent's actual ID
    credentials:
      type: "bearer"
      token: "${EXTERNAL_AGENT_TOKEN}"
```

### Step 3: Use External Agent

```bash
export OPENAI_API_KEY="sk-..."
export EXTERNAL_AGENT_TOKEN="eyJ..."

hector serve --config config.yaml
```

**Call the coordinator:**

```bash
hector call coordinator \
  "Research quantum computing advancements using both local and external research capabilities"
```

**What happens:**

1. Coordinator receives request
2. Coordinator calls `local_researcher` for initial research
3. Coordinator calls `external_specialist` (external agent) for specialized analysis
4. Coordinator synthesizes results from both agents
5. Returns combined findings

---

## Authentication Methods

### Bearer Token (JWT)

Most common for OAuth2/OIDC providers:

```yaml
agents:
  external_agent:
    type: "a2a"
    url: "https://agent.example.com"
    credentials:
      type: "bearer"
      token: "${EXTERNAL_TOKEN}"
```

**Get token from auth provider:**

```bash
# Example with Auth0
curl -X POST https://YOUR-TENANT.auth0.com/oauth/token \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "YOUR_CLIENT_ID",
    "client_secret": "YOUR_CLIENT_SECRET",
    "audience": "external-agent-api",
    "grant_type": "client_credentials"
  }'

# Response: {"access_token": "eyJ..."}
export EXTERNAL_TOKEN="eyJ..."
```

### API Key

Simple key-based authentication:

```yaml
agents:
  external_agent:
    type: "a2a"
    url: "https://agent.example.com"
    credentials:
      type: "api_key"
      key: "${EXTERNAL_API_KEY}"
      header: "X-API-Key"  # Custom header name (default: "X-API-Key")
```

Environment variable:
```bash
export EXTERNAL_API_KEY="ak_1234567890abcdef"
```

### Basic Authentication

Username and password:

```yaml
agents:
  external_agent:
    type: "a2a"
    url: "https://agent.example.com"
    credentials:
      type: "basic"
      username: "${EXTERNAL_USERNAME}"
      password: "${EXTERNAL_PASSWORD}"
```

Environment variables:
```bash
export EXTERNAL_USERNAME="agent_user"
export EXTERNAL_PASSWORD="secure_password"
```

### No Authentication

Public agents (not recommended for production):

```yaml
agents:
  public_agent:
    type: "a2a"
    url: "https://public-agent.example.com"
    # No credentials block
```

---

## Agent Naming and target_agent_id

### Default Behavior: Config Key as Agent ID

By default, Hector uses the config key as the remote agent ID:

```yaml
agents:
  weather_assistant:  # Config key = Remote agent ID
    type: "a2a"
    url: "https://weather-service.example.com"
```

When calling `agent_call("weather_assistant", "...")`, Hector sends a request to the remote service asking for agent `weather_assistant`.

### Explicit target_agent_id

Use `target_agent_id` when you want a different local name than the remote agent ID:

```yaml
agents:
  my_weather_service:  # Local name (what you call it)
    type: "a2a"
    url: "https://weather-service.example.com"
    target_agent_id: "weather_assistant"  # Remote agent ID
```

Now you call `agent_call("my_weather_service", "...")` locally, but Hector routes to the remote agent named `weather_assistant`.

### Use Cases for target_agent_id

**1. Descriptive Local Names:**
```yaml
agents:
  production_legal_service:
    type: "a2a"
    url: "https://legal.prod.example.com"
    target_agent_id: "legal_v2"  # Remote agent's actual ID
```

**2. Multiple Instances of Same Remote Agent:**
```yaml
agents:
  us_weather:
    type: "a2a"
    url: "https://us.weather.example.com"
    target_agent_id: "weather_assistant"
  
  eu_weather:
    type: "a2a"
    url: "https://eu.weather.example.com"
    target_agent_id: "weather_assistant"
```

**3. Version Migration:**
```yaml
agents:
  research_service:
    type: "a2a"
    url: "https://research.example.com"
    target_agent_id: "researcher_v3"  # Remote upgraded to v3
    # Local code still calls "research_service"
```

**4. Decoupling from Remote Naming:**
```yaml
agents:
  my_assistant:  # Your preferred name
    type: "a2a"
    url: "https://partner.example.com"
    target_agent_id: "partner_assistant_1234"  # Their ID
```

---

## Multi-Agent Orchestration with External Agents

### Example: Hybrid Research Team

```yaml
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"

agents:
  # Supervisor
  research_director:
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"
    tools: ["agent_call", "todo_write"]
    
    prompt:
      system_role: |
        You coordinate a hybrid research team:
        
        LOCAL TEAM:
        - web_researcher: Web research specialist
        - data_analyst: Data analysis specialist
        
        EXTERNAL TEAM:
        - academic_researcher: External academic research service
        - industry_analyst: External industry analysis service
        
        Delegate tasks based on expertise and synthesize results.
  
  # Local agents
  web_researcher:
    llm: "gpt-4o"
    tools: ["search"]
    prompt:
      system_role: "You gather information from web sources."
  
  data_analyst:
    llm: "gpt-4o"
    prompt:
      system_role: "You analyze data and identify trends."
  
  # External agents
  academic_researcher:
    type: "a2a"
    url: "https://academic-research.example.com"
    credentials:
      type: "bearer"
      token: "${ACADEMIC_TOKEN}"
  
  industry_analyst:
    type: "a2a"
    url: "https://industry-analysis.example.com"
    credentials:
      type: "api_key"
      key: "${INDUSTRY_API_KEY}"
      header: "X-API-Key"
```

**Usage:**

```bash
hector call research_director \
  "Research AI impact on healthcare: academic perspective and industry analysis"
```

Director automatically:
- Delegates academic research to `academic_researcher` (external)
- Delegates industry analysis to `industry_analyst` (external)
- Uses `web_researcher` and `data_analyst` (local) for supporting data
- Synthesizes all findings into comprehensive report

---

## Advanced Configuration

### Timeout Settings

```yaml
agents:
  slow_external:
    type: "a2a"
    url: "https://agent.example.com"
    timeout: 120  # 120 seconds (default: 60)
    credentials:
      type: "bearer"
      token: "${TOKEN}"
```

### Retry Logic

```yaml
agents:
  reliable_external:
    type: "a2a"
    url: "https://agent.example.com"
    max_retries: 3         # Retry up to 3 times
    retry_delay: 5         # Wait 5 seconds between retries
    credentials:
      type: "bearer"
      token: "${TOKEN}"
```

### Custom Headers

```yaml
agents:
  custom_external:
    type: "a2a"
    url: "https://agent.example.com"
    headers:
      X-Custom-Header: "value"
      X-Tenant-ID: "${TENANT_ID}"
    credentials:
      type: "bearer"
      token: "${TOKEN}"
```

### TLS Configuration

```yaml
agents:
  secure_external:
    type: "a2a"
    url: "https://agent.example.com"
    tls:
      verify: true
      ca_cert: "/path/to/ca.crt"
      client_cert: "/path/to/client.crt"
      client_key: "/path/to/client.key"
    credentials:
      type: "bearer"
      token: "${TOKEN}"
```

---

## Testing External Agents

### Test Connection Directly

```bash
# Test agent discovery
curl https://external-agent.example.com/.well-known/agent.json

# Test with authentication
curl -H "Authorization: Bearer $TOKEN" \
  https://external-agent.example.com/.well-known/agent.json
  https://external-agent.example.com/health
```

### Test via Hector

```bash
# Call external agent directly
hector call external_specialist "Simple test query"

# Check logs for connection details
hector serve --config config.yaml --log-level debug
```

### Debug Mode

```yaml
agents:
  debug_external:
    type: "a2a"
    url: "https://agent.example.com"
    debug: true  # Enable debug logging
    credentials:
      type: "bearer"
      token: "${TOKEN}"
```

---

## Common Patterns

### Pattern 1: Fallback Agents

Use local agent if external fails:

```yaml
agents:
  coordinator:
    prompt:
      system_role: |
        Try to use external_specialist for advanced analysis.
        If it fails or times out, use local_analyst instead.
```

### Pattern 2: Load Balancing

Multiple external agents for same task:

```yaml
agents:
  coordinator:
    prompt:
      system_role: |
        Available researchers:
        - external_researcher_1 (US-based, fast)
        - external_researcher_2 (EU-based, specialized)
        - external_researcher_3 (Asia-based, fallback)
        
        Choose based on availability and requirements.
  
  external_researcher_1:
    type: "a2a"
    url: "https://us.research.example.com"
    credentials:
      type: "bearer"
      token: "${US_TOKEN}"
  
  external_researcher_2:
    type: "a2a"
    url: "https://eu.research.example.com"
    credentials:
      type: "bearer"
      token: "${EU_TOKEN}"
  
  external_researcher_3:
    type: "a2a"
    url: "https://asia.research.example.com"
    credentials:
      type: "bearer"
      token: "${ASIA_TOKEN}"
```

### Pattern 3: Pipeline

Chain local and external agents:

```yaml
agents:
  coordinator:
    prompt:
      system_role: |
        Pipeline:
        1. local_collector: Gather raw data
        2. external_processor: Process with specialized algorithms
        3. local_formatter: Format for presentation
```

---

## Security Best Practices

### 1. Rotate Credentials Regularly

```bash
# Use short-lived tokens
# Rotate API keys every 90 days
# Implement token refresh if possible
```

### 2. Use Environment Variables

```yaml
# ✅ Good
credentials:
  type: "bearer"
  token: "${EXTERNAL_TOKEN}"

# ❌ Bad
credentials:
  type: "bearer"
  token: "hardcoded-token-123"
```

### 3. Verify TLS Certificates

```yaml
agents:
  secure_external:
    type: "a2a"
    url: "https://agent.example.com"
    tls:
      verify: true  # Always verify in production
```

### 4. Principle of Least Privilege

Only give external agents access to what they need:

```yaml
agents:
  coordinator:
    prompt:
      system_role: |
        external_specialist has read-only access.
        Never ask it to modify data or execute commands.
```

### 5. Monitor External Calls

Monitor external agent interactions through application logs and observability:

```bash
# Monitor for:
# - Failed authentication
# - Timeouts
# - Unusual patterns
```

---

## Troubleshooting

### Connection Refused

```bash
# Check URL is correct
curl https://external-agent.example.com/.well-known/agent.json

# Check network connectivity
ping external-agent.example.com

# Check firewall rules
```

### Authentication Errors (401)

```bash
# Verify token is valid
echo $EXTERNAL_TOKEN

# Test authentication manually
curl -H "Authorization: Bearer $EXTERNAL_TOKEN" \
  https://external-agent.example.com/health
```

### Timeout Errors

```yaml
agents:
  slow_external:
    type: "a2a"
    url: "https://agent.example.com"
    timeout: 180  # Increase timeout
```

### SSL/TLS Errors

```yaml
agents:
  tls_external:
    type: "a2a"
    url: "https://agent.example.com"
    tls:
      verify: true
      ca_cert: "/path/to/ca-bundle.crt"  # Add CA certificate
```

---

## Monitoring External Agents

### Track Performance

Monitor external agent performance through observability:

```yaml
agents:
  monitored_external:
    type: "a2a"
    url: "https://agent.example.com"
    metrics:
      
      track_latency: true
      track_errors: true
```

### Analyze Logs

```bash
# External agent call latency
cat hector.log | jq 'select(.external_agent) | .latency_ms'

# Success rate
cat hector.log | jq 'select(.external_agent) | .status' | \
  awk '{if($1==200) success++; total++} END {print success/total*100"%"}'
```

---

## Example: Commercial A2A Service

Integrate with a commercial A2A service:

```yaml
agents:
  coordinator:
    llm: "gpt-4o"
    reasoning:
      engine: "supervisor"
    tools: ["agent_call"]
  
  # Commercial legal analysis service
  legal_analyst:
    type: "a2a"
    url: "https://api.legalai.example.com"
    credentials:
      type: "api_key"
      key: "${LEGAL_AI_API_KEY}"
      header: "X-API-Key"
    timeout: 120
  
  # Commercial financial analysis service
  financial_analyst:
    type: "a2a"
    url: "https://api.financeai.example.com"
    credentials:
      type: "bearer"
      token: "${FINANCE_AI_TOKEN}"
    timeout: 90
```

**Usage:**

```bash
hector call coordinator \
  "Analyze the legal and financial implications of the proposed merger"
```

---

## Using CLI Client Mode

Hector's CLI can connect to ANY A2A-compliant service (not just Hector servers) using the `--url` flag:

### Connect to A2A Service

```bash
# List agents from any A2A service (auto-discovers)
hector list --url http://remote-service:8080

# Get agent info
hector info --url http://remote-service:8080 --agent researcher

# Call agent
hector call "task" --url http://remote-service:8080 --agent researcher

# Interactive chat
hector chat --url http://remote-service:8080 --agent researcher
```

### Direct Agent Card URL

If you know the agent card URL, point directly to it:

```bash
# Service root
hector list --url http://service/.well-known/agent.json

# Agent-specific
hector call "task" --url http://service/v1/agents/researcher/.well-known/agent.json

# With authentication
hector call "task" --url http://service/v1/agents/researcher/.well-known/agent.json --token "eyJ..."
```

### Multi-Vendor Interoperability

The CLI works with ANY A2A v0.3.0 compliant service:

```bash
# Connect to Hector service
hector call "task" --url http://hector-service:8080 --agent assistant

# Connect to other A2A implementation
hector call "task" --url http://other-vendor:8080 --agent some-agent

# Connect to commercial A2A SaaS
hector call "analyze contract" --url https://legal-ai.example.com --agent legal_assistant --token "$TOKEN"
```

### Environment Variables

```bash
export HECTOR_URL="https://agents.company.com"
export HECTOR_TOKEN="eyJ..."

# Now --url and --token are optional
hector list
hector call "task" --agent researcher
```

---

## Next Steps

- **[Multi-Agent Orchestration](../core-concepts/multi-agent.md)** - Understand orchestration
- **[Authentication & Security](../core-concepts/security.md)** - Secure external connections
- **[Build a Research System](build-research-system.md)** - Multi-agent example
- **[A2A Protocol](../reference/a2a-protocol.md)** - Protocol details

---

## Related Topics

- **[Agent Overview](../core-concepts/overview.md)** - Understanding agents
- **[Tools](../core-concepts/tools.md)** - agent_call tool
- **[Configuration Reference](../reference/configuration.md)** - External agent options

