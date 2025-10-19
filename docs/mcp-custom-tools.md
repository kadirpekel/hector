---
title: Custom MCP Tools
description: Build custom tools in 5 minutes with Python or TypeScript
---

# Building Custom MCP Tools in Minutes

**The fastest way to extend Hector with domain-specific capabilities**

---

## Why MCP for Custom Tools?

Hector's built-in tools (`execute_command`, `write_file`, `search`, etc.) cover general-purpose needs. But when you need domain-specific capabilities, **MCP (Model Context Protocol) servers** are the answer.

**Benefits:**

- **Fast** - Build in 5-10 minutes
- **Simple** - Just Python or TypeScript
- **Zero code in Hector** - Pure YAML configuration
- **Language-agnostic** - Any language that speaks HTTP/JSON
- **Hot-reload ready** - Update tools without restarting Hector
- **Shareable** - Publish for community use

**When to use:**

- Web scraping/search
- API integrations (weather, finance, CRM)
- Database operations
- File format conversions
- Custom business logic

---

## Quick Start: Python MCP Server

### 1. Create MCP Server (5 minutes)

```python
# weather_server.py
import json
import requests
from mcp import Server, Tool
from mcp.types import TextContent

# Create MCP server
server = Server("weather-server")

@server.tool()
async def get_weather(location: str) -> str:
    """Get current weather for a location."""
    try:
        # Replace with your weather API
        response = requests.get(f"https://api.weather.com/v1/current?location={location}")
        data = response.json()
        
        return f"Weather in {location}: {data['temperature']}°C, {data['condition']}"
    except Exception as e:
        return f"Error getting weather: {str(e)}"

if __name__ == "__main__":
    server.run()
```

### 2. Configure in Hector (30 seconds)

```yaml
# hector-config.yaml
mcp_servers:
  weather:
    command: "python"
    args: ["weather_server.py"]
    env:
      WEATHER_API_KEY: "${WEATHER_API_KEY}"

agents:
  weather_agent:
    name: "Weather Assistant"
    llm: "gpt-4o"
    tools: ["weather"]  # MCP tool automatically available
```

### 3. Use Your Tool

```bash
hector call weather_agent "What's the weather in Paris?"
```

**That's it!** Your custom tool is now available to all agents.

---

## Complete Example: Web Scraper

Let's build a more sophisticated example - a web scraper tool:

### Python Implementation

```python
# scraper_server.py
import json
import requests
from bs4 import BeautifulSoup
from mcp import Server
from typing import List, Dict

server = Server("web-scraper")

@server.tool()
async def scrape_website(url: str, selector: str = None) -> str:
    """Scrape content from a website."""
    try:
        response = requests.get(url, timeout=10)
        response.raise_for_status()
        
        soup = BeautifulSoup(response.content, 'html.parser')
        
        if selector:
            elements = soup.select(selector)
            content = [elem.get_text(strip=True) for elem in elements]
        else:
            # Extract main content
            content = soup.get_text(strip=True)
        
        return json.dumps({
            "url": url,
            "content": content,
            "status": "success"
        }, indent=2)
        
    except Exception as e:
        return json.dumps({
            "url": url,
            "error": str(e),
            "status": "error"
        })

@server.tool()
async def scrape_multiple(urls: List[str]) -> str:
    """Scrape multiple URLs."""
    results = []
    
    for url in urls:
        try:
            response = requests.get(url, timeout=10)
            soup = BeautifulSoup(response.content, 'html.parser')
            
            # Extract title and main content
            title = soup.find('title')
            title_text = title.get_text(strip=True) if title else "No title"
            
            # Get main content (simplified)
            main_content = soup.find('main') or soup.find('article') or soup.find('body')
            content = main_content.get_text(strip=True)[:500] if main_content else ""
            
            results.append({
                "url": url,
                "title": title_text,
                "content": content,
                "status": "success"
            })
            
        except Exception as e:
            results.append({
                "url": url,
                "error": str(e),
                "status": "error"
            })
    
    return json.dumps(results, indent=2)

if __name__ == "__main__":
    server.run()
```

### Hector Configuration

```yaml
mcp_servers:
  scraper:
    command: "python"
    args: ["scraper_server.py"]
    env:
      USER_AGENT: "Hector-Scraper/1.0"

agents:
  research_agent:
    name: "Web Research Agent"
    llm: "gpt-4o"
    tools: ["scrape_website", "scrape_multiple"]
    prompt:
      system_role: |
        You are a web research specialist. Use the scraping tools to gather
        information from websites and provide comprehensive analysis.
```

### Usage

```bash
hector call research_agent "Research the latest AI news from techcrunch.com"
```

---

## TypeScript Implementation

MCP servers can also be built in TypeScript:

```typescript
// database_server.ts
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from '@modelcontextprotocol/sdk/types.js';

const server = new Server(
  {
    name: 'database-server',
    version: '1.0.0',
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

// Database connection (example with SQLite)
import Database from 'better-sqlite3';
const db = new Database('data.db');

server.setRequestHandler(ListToolsRequestSchema, async () => {
  return {
    tools: [
      {
        name: 'query_database',
        description: 'Execute SQL query on the database',
        inputSchema: {
          type: 'object',
          properties: {
            query: {
              type: 'string',
              description: 'SQL query to execute',
            },
          },
          required: ['query'],
        },
      },
    ],
  };
});

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  if (request.params.name === 'query_database') {
    const { query } = request.params.arguments as { query: string };
    
    try {
      const result = db.prepare(query).all();
      return {
        content: [
          {
            type: 'text',
            text: JSON.stringify(result, null, 2),
          },
        ],
      };
    } catch (error) {
      return {
        content: [
          {
            type: 'text',
            text: `Database error: ${error.message}`,
          },
        ],
        isError: true,
      };
    }
  }
  
  throw new Error(`Unknown tool: ${request.params.name}`);
});

async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main().catch(console.error);
```

### TypeScript Configuration

```yaml
mcp_servers:
  database:
    command: "node"
    args: ["database_server.js"]
    env:
      DATABASE_PATH: "./data.db"

agents:
  data_agent:
    name: "Database Analyst"
    llm: "gpt-4o"
    tools: ["query_database"]
```

---

## Advanced Features

### Environment Variables

Pass environment variables to your MCP servers:

```yaml
mcp_servers:
  api_server:
    command: "python"
    args: ["api_server.py"]
    env:
      API_KEY: "${MY_API_KEY}"
      DEBUG: "true"
      TIMEOUT: "30"
```

### Multiple Tools per Server

One MCP server can provide multiple tools:

```python
@server.tool()
async def get_user(user_id: str) -> str:
    """Get user information by ID."""
    # Implementation

@server.tool()
async def create_user(name: str, email: str) -> str:
    """Create a new user."""
    # Implementation

@server.tool()
async def update_user(user_id: str, data: dict) -> str:
    """Update user information."""
    # Implementation
```

### Error Handling

MCP servers should handle errors gracefully:

```python
@server.tool()
async def safe_api_call(endpoint: str) -> str:
    """Make a safe API call with error handling."""
    try:
        response = requests.get(endpoint, timeout=10)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.Timeout:
        return json.dumps({"error": "Request timeout", "status": "timeout"})
    except requests.exceptions.ConnectionError:
        return json.dumps({"error": "Connection failed", "status": "connection_error"})
    except requests.exceptions.HTTPError as e:
        return json.dumps({"error": f"HTTP {e.response.status_code}", "status": "http_error"})
    except Exception as e:
        return json.dumps({"error": str(e), "status": "unknown_error"})
```

---

## Best Practices

### 1. **Keep Tools Focused**

```python
# Good: Focused tool
@server.tool()
async def get_weather(location: str) -> str:
    """Get current weather for a location."""
    # Single responsibility

# Avoid: Too broad
@server.tool()
async def weather_and_news_and_stocks(location: str, query: str, symbol: str) -> str:
    """Get weather, news, and stock data."""
    # Too many responsibilities
```

### 2. **Use Structured Output**

```python
@server.tool()
async def analyze_text(text: str) -> str:
    """Analyze text and return structured results."""
    # Process text...
    
    result = {
        "word_count": len(text.split()),
        "sentiment": "positive",  # From analysis
        "key_phrases": ["AI", "technology"],
        "language": "en"
    }
    
    return json.dumps(result, indent=2)
```

### 3. **Handle Edge Cases**

```python
@server.tool()
async def safe_operation(input_data: str) -> str:
    """Perform operation with comprehensive error handling."""
    if not input_data:
        return json.dumps({"error": "Input is required", "status": "invalid_input"})
    
    if len(input_data) > 10000:
        return json.dumps({"error": "Input too large", "status": "input_too_large"})
    
    try:
        # Main operation
        result = process_data(input_data)
        return json.dumps({"result": result, "status": "success"})
    except Exception as e:
        return json.dumps({"error": str(e), "status": "processing_error"})
```

### 4. **Document Your Tools**

```python
@server.tool()
async def complex_tool(param1: str, param2: int = 10) -> str:
    """
    Perform complex operation with detailed documentation.
    
    Args:
        param1: Description of param1
        param2: Description of param2 (default: 10)
    
    Returns:
        JSON string with operation results
    
    Raises:
        ValueError: If param1 is invalid
    """
    # Implementation
```

---

## Publishing MCP Servers

### Package for Distribution

```python
# setup.py
from setuptools import setup, find_packages

setup(
    name="hector-weather-tools",
    version="1.0.0",
    packages=find_packages(),
    install_requires=[
        "mcp",
        "requests",
    ],
    entry_points={
        "console_scripts": [
            "weather-server=weather_server:main",
        ],
    },
)
```

### Share with Community

1. **Create GitHub repository**
2. **Add installation instructions**
3. **Include example Hector configuration**
4. **Document tool capabilities**

```markdown
# Hector Weather Tools

MCP server providing weather-related tools for Hector agents.

## Installation

```bash
pip install hector-weather-tools
```

## Configuration

```yaml
mcp_servers:
  weather:
    command: "weather-server"
    env:
      WEATHER_API_KEY: "${WEATHER_API_KEY}"
```

## Tools

- `get_weather(location)` - Get current weather
- `get_forecast(location, days)` - Get weather forecast
```

---

## Troubleshooting

### Common Issues

**1. MCP Server Not Starting**
```bash
# Check if command exists
which python
which node

# Test MCP server directly
python weather_server.py
```

**2. Tools Not Available**
```yaml
# Ensure MCP server is configured
mcp_servers:
  weather:
    command: "python"
    args: ["weather_server.py"]

# Add tools to agent
agents:
  my_agent:
    tools: ["get_weather"]  # Must match tool name
```

**3. Permission Errors**
```bash
# Make scripts executable
chmod +x weather_server.py

# Check file permissions
ls -la weather_server.py
```

### Debug Mode

Enable debug mode in Hector:

```yaml
global:
  debug: true
```

---

## Conclusion

MCP servers are the perfect way to extend Hector with custom capabilities. They're:

- **Fast to build** - 5-10 minutes for simple tools
- **Easy to maintain** - Standard HTTP/JSON interface
- **Language flexible** - Python, TypeScript, Go, Rust, etc.
- **Production ready** - Built-in error handling

