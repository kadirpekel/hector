---
title: Add Custom Tools
description: Extend agents with custom tools using MCP servers in 10 minutes
---

# How to Add Custom Tools

Extend your agents with custom tools using the Model Context Protocol (MCP). Build a custom tool in minutes, not hours.

**Time:** 10-15 minutes  
**Difficulty:** Beginner

---

## What You'll Learn

- Create a simple MCP server with custom tools
- Connect the MCP server to Hector
- Use your custom tools in agents
- Best practices for tool development

---

## Understanding MCP

**MCP (Model Context Protocol)** is an open standard for connecting AI agents to tools and data sources.

**Benefits:**
- **Fast development** - 5-10 minutes to create a tool
- **Language agnostic** - Python, JavaScript, Go, etc.
- **Standardized** - Works with any MCP-compliant agent
- **Ecosystem** - 150+ tools available via Composio

---

## Quick Example: Weather Tool

Let's build a simple weather tool that agents can use.

### Step 1: Create MCP Server

Create `weather_server.py`:

```python
#!/usr/bin/env python3
"""
Simple MCP server providing weather information.
"""

from mcp.server import Server, Tool
from mcp.types import TextContent
import json

# Create MCP server
server = Server("weather-server")

@server.tool()
def get_weather(city: str) -> str:
    """
    Get current weather for a city.
    
    Args:
        city: Name of the city
    
    Returns:
        Weather information as a string
    """
    # In production, call real weather API
    # For demo, return mock data
    weather_data = {
        "San Francisco": "Sunny, 72°F",
        "New York": "Cloudy, 65°F",
        "London": "Rainy, 55°F",
        "Tokyo": "Clear, 68°F"
    }
    
    weather = weather_data.get(city, f"Weather data not available for {city}")
    return f"Weather in {city}: {weather}"

@server.tool()
def get_forecast(city: str, days: int = 3) -> str:
    """
    Get weather forecast for a city.
    
    Args:
        city: Name of the city
        days: Number of days to forecast (default: 3)
    
    Returns:
        Forecast information as a string
    """
    # Mock forecast
    forecast = []
    for i in range(days):
        forecast.append(f"Day {i+1}: Partly cloudy, 70°F")
    
    return f"Forecast for {city}:\n" + "\n".join(forecast)

if __name__ == "__main__":
    # Run server on port 3000
    server.run(port=3000)
```

### Step 2: Install Dependencies

```bash
pip install mcp-server
```

### Step 3: Start MCP Server

```bash
python weather_server.py

# Output:
# MCP server listening on http://localhost:3000
# Tools available: get_weather, get_forecast
```

### Step 4: Configure in Hector

Create `config.yaml`:

```yaml
# MCP Tools Configuration
tools:
  weather_server:
    type: "mcp"
    
    server_url: "http://localhost:3000"
    description: "Weather information tools"

# Agent using weather tools
agents:
  weather_assistant:
    llm: "gpt-4o"
    
    prompt:
      system_prompt: |
        You are a helpful weather assistant.
        Use the weather tools (get_weather, get_forecast)
        to provide accurate weather information.
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 10

# LLM Configuration
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
```

### Step 5: Test It

```bash
hector serve --config config.yaml

# In another terminal
hector call "What's the weather in San Francisco?" --agent weather_assistant --config config.yaml
```

Agent response:
```
Let me check the weather for you.
[Tool: get_weather("San Francisco")]
The weather in San Francisco is currently Sunny with a temperature of 72°F.
```

**That's it!** You've created and integrated a custom tool.

---

## More Complex Example: Database Tool

Create a tool that queries a database:

```python
#!/usr/bin/env python3
from mcp.server import Server
import sqlite3

server = Server("database-server")

@server.tool()
def query_users(name: str = None, limit: int = 10) -> str:
    """
    Query users from database.
    
    Args:
        name: Optional name filter
        limit: Maximum results (default: 10)
    
    Returns:
        JSON string of user records
    """
    conn = sqlite3.connect('users.db')
    cursor = conn.cursor()
    
    if name:
        cursor.execute(
            "SELECT * FROM users WHERE name LIKE ? LIMIT ?",
            (f"%{name}%", limit)
        )
    else:
        cursor.execute("SELECT * FROM users LIMIT ?", (limit,))
    
    results = cursor.fetchall()
    conn.close()
    
    return json.dumps([
        {"id": row[0], "name": row[1], "email": row[2]}
        for row in results
    ])

@server.tool()
def create_user(name: str, email: str) -> str:
    """
    Create a new user.
    
    Args:
        name: User's name
        email: User's email
    
    Returns:
        Success message with user ID
    """
    conn = sqlite3.connect('users.db')
    cursor = conn.cursor()
    
    cursor.execute(
        "INSERT INTO users (name, email) VALUES (?, ?)",
        (name, email)
    )
    user_id = cursor.lastrowid
    
    conn.commit()
    conn.close()
    
    return f"User created successfully with ID: {user_id}"

if __name__ == "__main__":
    server.run(port=3001)
```

---

## Using Composio (150+ Pre-built Tools)

Composio provides MCP servers for popular services:

### Step 1: Start Composio Server

```bash
# Install
npm install -g @composio/cli

# Start server
composio server start --port 3000

# Available tools:
# - github_*: GitHub operations
# - slack_*: Slack messaging
# - notion_*: Notion database
# - gmail_*: Gmail operations
# ... and 150+ more
```

### Step 2: Configure in Hector

```yaml
tools:
  composio:
    type: "mcp"
    
    server_url: "http://localhost:3000"
    description: "Composio - 150+ app integrations"
    # Authentication handled by Composio server

agents:
  automation:
    llm: "gpt-4o"
    
    prompt:
      system_prompt: |
        You are an automation assistant with access to:
        - GitHub (create issues, PRs)
        - Slack (send messages, notifications)
        - Notion (manage pages, databases)
        
        Use the Composio tools to automate workflows.
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 20
```

### Step 3: Use the Tools

```bash
hector call --config config.yaml automation \
  "Create a GitHub issue for the authentication bug and notify the team on Slack"
```

Agent automatically:
1. Creates GitHub issue
2. Sends Slack notification
3. Returns confirmation

---

## Best Practices

### 1. Clear Tool Descriptions

```python
@server.tool()
def analyze_sentiment(text: str) -> str:
    """
    Analyze the sentiment of text.
    
    Args:
        text: The text to analyze (max 1000 characters)
    
    Returns:
        JSON with sentiment (positive/negative/neutral) and confidence score
    
    Example:
        analyze_sentiment("I love this product!")
        -> {"sentiment": "positive", "confidence": 0.95}
    """
    # Implementation
```

**Good descriptions help the LLM use tools correctly.**

### 2. Input Validation

```python
@server.tool()
def send_email(to: str, subject: str, body: str) -> str:
    """Send an email."""
    
    # Validate inputs
    if not to or "@" not in to:
        return "Error: Invalid email address"
    
    if len(body) > 10000:
        return "Error: Email body too long (max 10000 characters)"
    
    # Send email
    ...
```

### 3. Error Handling

```python
@server.tool()
def fetch_data(url: str) -> str:
    """Fetch data from URL."""
    try:
        response = requests.get(url, timeout=10)
        response.raise_for_status()
        return response.text
    except requests.exceptions.Timeout:
        return "Error: Request timed out"
    except requests.exceptions.HTTPError as e:
        return f"Error: HTTP {e.response.status_code}"
    except Exception as e:
        return f"Error: {str(e)}"
```

### 4. Type Hints

```python
from typing import List, Dict, Optional

@server.tool()
def search_products(
    query: str,
    category: Optional[str] = None,
    max_results: int = 10
) -> str:
    """Search products with optional filters."""
    # Type hints help MCP generate correct schemas
```

### 5. Structured Responses

```python
import json

@server.tool()
def get_user_info(user_id: int) -> str:
    """Get user information."""
    user = database.get_user(user_id)
    
    # Return structured data as JSON
    return json.dumps({
        "id": user.id,
        "name": user.name,
        "email": user.email,
        "created_at": user.created_at.isoformat()
    })
```

---

## Advanced: Authentication

For tools that need authentication:

```python
from mcp.server import Server
import os

server = Server("secure-server")

@server.tool()
def secure_operation(api_key: str, resource_id: str) -> str:
    """
    Perform a secure operation.
    
    Args:
        api_key: User's API key
        resource_id: Resource to access
    """
    # Validate API key
    if api_key != os.getenv("VALID_API_KEY"):
        return "Error: Invalid API key"
    
    # Perform operation
    ...
```

In Hector config:

```yaml
tools:
  custom_mcp:
    type: "mcp"
    
    server_url: "http://localhost:3000"
    description: "Custom MCP tools"
    # Add authentication in MCP server if needed
```

---

## MCP vs gRPC Plugins

| Feature | MCP | gRPC Plugins |
|---------|-----|--------------|
| **Setup time** | 5-10 minutes | Hours/Days |
| **Language** | Python, JS, Go, etc. | Any with gRPC |
| **Use case** | Quick tools, integrations | Complex logic, high performance |
| **Ecosystem** | 150+ via Composio | Custom only |
| **Performance** | Good | Excellent |

**Use MCP for:**
- Quick prototypes
- API integrations
- Simple tools
- External services

**Use gRPC plugins for:**
- Custom LLMs
- High-performance tools
- Complex business logic
- Enterprise integrations

---

## Debugging MCP Tools

### Test MCP Server Directly

```bash
# List available tools
curl http://localhost:3000/tools

# Call tool directly
curl -X POST http://localhost:3000/call \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "get_weather",
    "arguments": {"city": "San Francisco"}
  }'
```

### Enable Debug in Hector

```yaml
agents:
  debug_agent:
    reasoning:
      show_tool_execution: true
      show_debug_info: true
```

### Check MCP Server Logs

```bash
# MCP servers log to stdout
python weather_server.py 2>&1 | tee mcp.log
```

---

## Production Deployment

### Docker for MCP Server

```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY weather_server.py .

EXPOSE 3000
CMD ["python", "weather_server.py"]
```

```bash
docker build -t weather-mcp .
docker run -d -p 3000:3000 weather-mcp
```

### Health Checks

```python
@server.health_check()
def health() -> Dict[str, str]:
    """Health check endpoint."""
    return {
        "status": "healthy",
        "tools": len(server.tools),
        "uptime": server.uptime()
    }
```

### Monitoring

```python
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@server.tool()
def my_tool(arg: str) -> str:
    logger.info(f"Tool called with arg: {arg}")
    try:
        result = perform_operation(arg)
        logger.info(f"Tool succeeded")
        return result
    except Exception as e:
        logger.error(f"Tool failed: {e}")
        raise
```

---

## Next Steps

- **[Tools](../core-concepts/tools.md)** - Understand the tool system
- **[Build a Coding Assistant](build-coding-assistant.md)** - Use tools in practice
- **[Architecture](../reference/architecture.md)** - gRPC plugin development
- **[MCP Documentation](https://modelcontextprotocol.org)** - Official MCP docs

---

## Related Topics

- **[Agent Overview](../core-concepts/overview.md)** - Understanding agents
- **[Configuration Reference](../reference/configuration.md)** - Tool configuration
- **[Multi-Agent Orchestration](../core-concepts/multi-agent.md)** - Tools in multi-agent systems

