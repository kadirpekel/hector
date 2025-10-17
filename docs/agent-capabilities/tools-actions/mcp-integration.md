---
layout: default
title: MCP Integration
nav_order: 2
parent: Tools & Actions
description: "Connect to 150+ external tools via Model Context Protocol"
---

# MCP Integration

**Model Context Protocol (MCP)** is an open standard that enables agents to connect to external tools and data sources.

> 🔥 **Want to add custom tools?** See **[Building Custom MCP Tools in 5 Minutes](../how-to/MCP_CUSTOM_TOOLS)** - The fastest way to extend Hector with domain-specific capabilities.

## What is MCP?

MCP is like a **universal adapter** for AI tools:
- **Standard protocol** - JSON-RPC 2.0 over HTTP/SSE
- **Auto-discovery** - Tools self-describe their capabilities
- **Provider ecosystem** - Composio, Mem0, Browserbase, custom servers
- **Language-agnostic** - Python, TypeScript, Go, any language
- **Build your own** - Custom tools in 5-10 minutes

**Why MCP over custom integrations?**
- ✅ **Zero custom code** - Just point to a URL
- ✅ **150+ integrations** - Via providers like Composio
- ✅ **Standardized** - One protocol for everything
- ✅ **Community** - Growing ecosystem of MCP servers
- ✅ **Fast custom tools** - Build in Python/TypeScript in minutes

## Quick Start

**1. Connect to MCP Server:**
```yaml
tools:
  composio_tools:
    type: "mcp"
    enabled: true
    server_url: "https://api.composio.dev/mcp"
    description: "Composio - 150+ app integrations"
```

**2. Hector automatically discovers tools:**
```
🔍 Discovering tools from MCP server: composio_tools (https://api.composio.dev/mcp)
✅ MCP source composio_tools discovered 150 tools
```

**3. Agent can now use them:**
```
Agent: Let me send a Slack message
Tool Call: slack_send_message(
  channel="#general",
  text="Deployment complete!"
)
```

## Popular MCP Providers

### 1. Composio - 150+ App Integrations

**Connect to GitHub, Slack, Gmail, Jira, and more**

```yaml
tools:
  composio:
    type: "mcp"
    enabled: true
    server_url: "https://api.composio.dev/mcp"
    description: "Composio - Enterprise app integrations"
```

**Example tools:**
- `github_create_issue` - Create GitHub issues
- `slack_send_message` - Post to Slack channels
- `gmail_send_email` - Send emails via Gmail
- `jira_create_ticket` - Create Jira tickets
- 146+ more...

**Setup:** Get API key from [Composio](https://composio.dev)

### 2. Mem0 - Memory & Personalization

**Add persistent memory to your agents**

```yaml
tools:
  mem0:
    type: "mcp"
    enabled: true
    server_url: "https://api.mem0.ai/mcp"
    description: "Mem0 - Agent memory layer"
```

**Example tools:**
- `mem0_store` - Store user preferences/facts
- `mem0_recall` - Retrieve relevant memories
- `mem0_search` - Search memory by query

**Use cases:**
- User preferences ("Alice prefers Python")
- Conversation context across sessions
- Long-term agent knowledge

### 3. Browserbase - Browser Automation

**Headless browser capabilities for agents**

```yaml
tools:
  browserbase:
    type: "mcp"
    enabled: true
    server_url: "https://api.browserbase.com/mcp"
```

**Example tools:**
- `browser_navigate` - Go to URL
- `browser_click` - Click elements
- `browser_screenshot` - Capture screenshots
- `browser_extract` - Extract page data

### 4. Your Custom MCP Server

**Build your own tools for specific needs**

```yaml
tools:
  my_custom_tools:
    type: "mcp"
    enabled: true
    server_url: "http://localhost:3000/mcp"
    description: "Custom business logic tools"
```

## Building Custom MCP Servers

Create your own MCP server in **Python** or **TypeScript**.

For a comprehensive guide with real-world examples, see **[Custom MCP Tools](../how-to/MCP_CUSTOM_TOOLS)**.

### Python Example

**1. Install MCP SDK:**
```bash
pip install mcp
```

**2. Create server (`server.py`):**
```python
from mcp.server import Server, Tool
from mcp.types import TextContent

app = Server("my-custom-tools")

@app.tool()
def get_weather(location: str) -> TextContent:
    """Get current weather for a location."""
    # Your custom logic here
    return TextContent(f"Weather in {location}: Sunny, 72°F")

if __name__ == "__main__":
    app.run()
```

**3. Start server:**
```bash
python server.py
```

**4. Connect to Hector:**
```yaml
tools:
  weather_tools:
    type: "mcp"
    enabled: true
    server_url: "http://localhost:8000/mcp"
```

## Configuration Options

```yaml
tools:
  my_mcp_server:
    type: "mcp"
    enabled: true
    server_url: "https://api.example.com/mcp"
    description: "Custom MCP server"
    timeout: "30s"           # Request timeout
    retry_attempts: 3        # Retry failed requests
    api_key: "${API_KEY}"    # Authentication
```

## Best Practices

1. **Start with Providers**: Use Composio, Mem0, etc. before building custom
2. **Test Locally**: Run MCP servers locally during development
3. **Handle Errors**: MCP servers can be unavailable
4. **Monitor Usage**: Track tool calls and performance
5. **Document Tools**: Clear descriptions help agents use tools correctly

## See Also

- **[Built-in Tools](built-in-tools)** - Hector's 5 core tools
- **[Custom Tools](custom-tools)** - Build custom capabilities via gRPC
- **[Custom MCP Tools](../how-to/MCP_CUSTOM_TOOLS)** - Complete tutorial
