# Building Custom MCP Tools in Minutes

**The fastest way to extend Hector with domain-specific capabilities**

---

## Why MCP for Custom Tools?

Hector's built-in tools (`execute_command`, `write_file`, `search`, etc.) cover general-purpose needs. But when you need domain-specific capabilities, **MCP (Model Context Protocol) servers** are the answer.

**Benefits:**
- ⚡ **Fast** - Build in 5-10 minutes
- 🔧 **Simple** - Just Python or TypeScript
- 🔌 **Zero code in Hector** - Pure YAML configuration
- 🌐 **Language-agnostic** - Any language that speaks HTTP/JSON
- 🔄 **Hot-reload ready** - Update tools without restarting Hector
- 📦 **Shareable** - Publish for community use

**When to use:**
- Web scraping/search
- API integrations (weather, finance, CRM)
- Database operations
- Business logic
- Calculations/transformations
- External service calls

---

## Quick Start: 5-Minute Tool

Let's build a web search tool using Python.

### Step 1: Create MCP Server (2 minutes)

```bash
# Create directory
mkdir my-tools-server
cd my-tools-server

# Install MCP SDK
pip install mcp requests
```

**Create `server.py`:**

```python
from mcp.server import Server
import requests
import os

app = Server("my-tools")

@app.tool()
async def web_search(query: str, num_results: int = 5) -> str:
    """Search the web using Brave Search API"""
    api_key = os.getenv("BRAVE_API_KEY")
    url = "https://api.search.brave.com/res/v1/web/search"
    
    response = requests.get(
        url,
        headers={"X-Subscription-Token": api_key},
        params={"q": query, "count": num_results}
    )
    
    results = response.json().get("web", {}).get("results", [])
    
    # Format results
    output = []
    for i, result in enumerate(results, 1):
        output.append(f"{i}. {result['title']}")
        output.append(f"   {result['url']}")
        output.append(f"   {result['description']}\n")
    
    return "\n".join(output)

@app.tool()
async def get_weather(city: str) -> str:
    """Get current weather for a city"""
    api_key = os.getenv("WEATHER_API_KEY")
    url = f"https://api.openweathermap.org/data/2.5/weather"
    
    response = requests.get(url, params={"q": city, "appid": api_key})
    data = response.json()
    
    temp = data["main"]["temp"] - 273.15  # Kelvin to Celsius
    description = data["weather"][0]["description"]
    
    return f"Weather in {city}: {description}, {temp:.1f}°C"

if __name__ == "__main__":
    app.run(port=3000)
```

### Step 2: Start MCP Server (1 minute)

```bash
# Set API keys
export BRAVE_API_KEY="your-key"
export WEATHER_API_KEY="your-key"

# Run server
python server.py
```

Output:
```
🚀 MCP Server listening on http://localhost:3000
📋 Available tools: web_search, get_weather
```

### Step 3: Configure Hector (1 minute)

**Add to your `config.yaml`:**

```yaml
tools:
  my_tools:
    type: "mcp"
    enabled: true
    server_url: "http://localhost:3000"
    description: "Custom web search and weather tools"

agents:
  assistant:
    name: "Assistant with Custom Tools"
    llm: "gpt-4o"
    # Agent automatically has access to web_search and get_weather!
```

### Step 4: Use It! (1 minute)

```bash
# Start Hector
./hector serve --config config.yaml

# Test the custom tools
./hector call assistant "Search the web for 'AI agent frameworks' and tell me the top 3"
./hector call assistant "What's the weather in Tokyo?"
```

**That's it!** You've extended Hector with custom tools in 5 minutes.

---

## Real-World Examples

### Example 1: Database Query Tool

```python
from mcp.server import Server
import psycopg2
import os

app = Server("database-tools")

@app.tool()
async def query_customers(email: str) -> str:
    """Look up customer information by email"""
    conn = psycopg2.connect(os.getenv("DATABASE_URL"))
    cursor = conn.cursor()
    
    cursor.execute(
        "SELECT name, email, plan, status FROM customers WHERE email = %s",
        (email,)
    )
    result = cursor.fetchone()
    
    if result:
        return f"Customer: {result[0]}\nEmail: {result[1]}\nPlan: {result[2]}\nStatus: {result[3]}"
    else:
        return "Customer not found"

@app.tool()
async def get_order_history(customer_id: int, limit: int = 10) -> str:
    """Get recent orders for a customer"""
    conn = psycopg2.connect(os.getenv("DATABASE_URL"))
    cursor = conn.cursor()
    
    cursor.execute(
        """
        SELECT order_id, date, total, status 
        FROM orders 
        WHERE customer_id = %s 
        ORDER BY date DESC 
        LIMIT %s
        """,
        (customer_id, limit)
    )
    
    orders = cursor.fetchall()
    
    output = []
    for order in orders:
        output.append(f"Order #{order[0]} - {order[1]} - ${order[2]} - {order[3]}")
    
    return "\n".join(output)

if __name__ == "__main__":
    app.run(port=3001)
```

**Hector config:**
```yaml
tools:
  database:
    type: "mcp"
    enabled: true
    server_url: "http://localhost:3001"

agents:
  support_agent:
    name: "Customer Support"
    llm: "gpt-4o"
    prompt:
      system_role: |
        You help with customer support queries.
        Use query_customers to look up customer info.
        Use get_order_history to check their orders.
```

**Usage:**
```bash
./hector call support_agent "What orders has customer@example.com placed recently?"
```

---

### Example 2: API Integration Tool

```python
from mcp.server import Server
import requests
import os

app = Server("stripe-tools")

@app.tool()
async def create_customer(email: str, name: str) -> str:
    """Create a new Stripe customer"""
    api_key = os.getenv("STRIPE_API_KEY")
    
    response = requests.post(
        "https://api.stripe.com/v1/customers",
        auth=(api_key, ""),
        data={"email": email, "name": name}
    )
    
    customer = response.json()
    return f"Created customer: {customer['id']}"

@app.tool()
async def create_payment_intent(amount: int, currency: str, customer_id: str) -> str:
    """Create a Stripe payment intent"""
    api_key = os.getenv("STRIPE_API_KEY")
    
    response = requests.post(
        "https://api.stripe.com/v1/payment_intents",
        auth=(api_key, ""),
        data={
            "amount": amount,
            "currency": currency,
            "customer": customer_id
        }
    )
    
    intent = response.json()
    return f"Payment intent created: {intent['id']}\nClient secret: {intent['client_secret']}"

if __name__ == "__main__":
    app.run(port=3002)
```

---

### Example 3: Data Processing Tool

```python
from mcp.server import Server
import pandas as pd
import json

app = Server("data-tools")

@app.tool()
async def analyze_csv(file_path: str, operation: str) -> str:
    """Analyze CSV files with various operations
    
    Operations: summary, head, describe, columns
    """
    df = pd.read_csv(file_path)
    
    if operation == "summary":
        return f"Rows: {len(df)}, Columns: {len(df.columns)}\n{df.dtypes.to_string()}"
    elif operation == "head":
        return df.head().to_string()
    elif operation == "describe":
        return df.describe().to_string()
    elif operation == "columns":
        return "\n".join(df.columns.tolist())
    else:
        return "Unknown operation"

@app.tool()
async def transform_data(data: str, transformation: str) -> str:
    """Transform JSON data
    
    Transformations: sort, filter, aggregate
    """
    items = json.loads(data)
    
    if transformation == "sort":
        return json.dumps(sorted(items, key=lambda x: x.get("value", 0)))
    elif transformation == "filter":
        return json.dumps([item for item in items if item.get("active", False)])
    else:
        return data

if __name__ == "__main__":
    app.run(port=3003)
```

---

## TypeScript Alternative

Prefer TypeScript? Here's the equivalent:

### Install SDK

```bash
npm install @modelcontextprotocol/sdk axios
```

### Create Server

**`server.ts`:**

```typescript
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import axios from "axios";

const server = new Server({
  name: "my-tools",
  version: "1.0.0",
}, {
  capabilities: {
    tools: {},
  },
});

// Define tools
server.setRequestHandler("tools/list", async () => ({
  tools: [
    {
      name: "web_search",
      description: "Search the web",
      inputSchema: {
        type: "object",
        properties: {
          query: { type: "string", description: "Search query" },
          num_results: { type: "number", description: "Number of results", default: 5 },
        },
        required: ["query"],
      },
    },
    {
      name: "get_weather",
      description: "Get current weather",
      inputSchema: {
        type: "object",
        properties: {
          city: { type: "string", description: "City name" },
        },
        required: ["city"],
      },
    },
  ],
}));

// Handle tool calls
server.setRequestHandler("tools/call", async (request) => {
  const { name, arguments: args } = request.params;
  
  if (name === "web_search") {
    const response = await axios.get("https://api.search.brave.com/res/v1/web/search", {
      headers: { "X-Subscription-Token": process.env.BRAVE_API_KEY },
      params: { q: args.query, count: args.num_results || 5 },
    });
    
    const results = response.data.web?.results || [];
    const formatted = results.map((r: any, i: number) => 
      `${i + 1}. ${r.title}\n   ${r.url}\n   ${r.description}`
    ).join("\n\n");
    
    return {
      content: [{ type: "text", text: formatted }],
    };
  }
  
  if (name === "get_weather") {
    const response = await axios.get("https://api.openweathermap.org/data/2.5/weather", {
      params: { q: args.city, appid: process.env.WEATHER_API_KEY },
    });
    
    const temp = response.data.main.temp - 273.15;
    const description = response.data.weather[0].description;
    
    return {
      content: [{
        type: "text",
        text: `Weather in ${args.city}: ${description}, ${temp.toFixed(1)}°C`,
      }],
    };
  }
  
  throw new Error(`Unknown tool: ${name}`);
});

// Start server
const transport = new StdioServerTransport();
server.connect(transport);
```

**Run:**
```bash
node server.ts
```

---

## Advanced: Parameter Validation

MCP supports rich JSON Schema for parameter validation:

```python
from mcp.server import Server

app = Server("validated-tools")

@app.tool()
async def process_order(
    order_id: int,
    action: str,  # Will be validated via schema
    priority: str = "normal"
) -> str:
    """Process an order with validation
    
    Args:
        order_id: Order ID (must be positive)
        action: Action to take (approve, reject, hold)
        priority: Priority level (low, normal, high, urgent)
    """
    return f"Order {order_id} processed: {action} with priority {priority}"

# MCP automatically generates schema:
# {
#   "type": "object",
#   "properties": {
#     "order_id": {"type": "integer"},
#     "action": {"type": "string"},
#     "priority": {"type": "string", "default": "normal"}
#   },
#   "required": ["order_id", "action"]
# }
```

**Add manual validation for enums:**

```python
@app.tool()
async def process_order(order_id: int, action: str, priority: str = "normal") -> str:
    """Process an order
    
    Parameters:
        order_id: Order ID
        action: One of: approve, reject, hold
        priority: One of: low, normal, high, urgent
    """
    valid_actions = ["approve", "reject", "hold"]
    valid_priorities = ["low", "normal", "high", "urgent"]
    
    if action not in valid_actions:
        return f"Error: action must be one of {valid_actions}"
    if priority not in valid_priorities:
        return f"Error: priority must be one of {valid_priorities}"
    
    # Process order...
    return f"Order {order_id} processed"
```

---

## Production Tips

### 1. Error Handling

```python
@app.tool()
async def safe_operation(param: str) -> str:
    """Operation with proper error handling"""
    try:
        # Your logic here
        result = risky_operation(param)
        return f"Success: {result}"
    except ValueError as e:
        return f"Validation error: {str(e)}"
    except Exception as e:
        return f"Error: {str(e)}"
```

### 2. Logging

```python
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@app.tool()
async def logged_operation(param: str) -> str:
    """Operation with logging"""
    logger.info(f"Starting operation with param: {param}")
    
    try:
        result = do_work(param)
        logger.info(f"Operation completed successfully")
        return result
    except Exception as e:
        logger.error(f"Operation failed: {e}")
        raise
```

### 3. Authentication

```python
@app.tool()
async def authenticated_operation(api_key: str, data: str) -> str:
    """Operation requiring authentication"""
    if not validate_api_key(api_key):
        return "Error: Invalid API key"
    
    # Proceed with operation
    return process(data)
```

### 4. Rate Limiting

```python
from time import time

rate_limit = {}

@app.tool()
async def rate_limited_operation(user_id: str, query: str) -> str:
    """Operation with rate limiting"""
    now = time()
    
    if user_id in rate_limit:
        if now - rate_limit[user_id] < 60:  # 1 request per minute
            return "Error: Rate limit exceeded. Try again later."
    
    rate_limit[user_id] = now
    return perform_query(query)
```

---

## Deployment

### Development

```bash
# Run locally
python server.py
```

```yaml
tools:
  my_tools:
    server_url: "http://localhost:3000"
```

### Production

**Option 1: Docker**

```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY server.py .

EXPOSE 3000
CMD ["python", "server.py"]
```

```bash
docker build -t my-mcp-server .
docker run -p 3000:3000 -e API_KEY=xxx my-mcp-server
```

```yaml
tools:
  my_tools:
    server_url: "http://my-mcp-server:3000"
```

**Option 2: Cloud Service (Railway, Render, etc.)**

Deploy your MCP server to any cloud platform:

```yaml
tools:
  my_tools:
    server_url: "https://my-mcp-server.railway.app"
```

---

## Testing

### Test MCP Server Directly

```bash
# List tools
curl -X POST http://localhost:3000 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list",
    "params": {}
  }'

# Call a tool
curl -X POST http://localhost:3000 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "web_search",
      "arguments": {
        "query": "AI agents",
        "num_results": 3
      }
    }
  }'
```

### Test with Hector

```bash
# Start Hector
./hector serve --config config.yaml --debug

# Call agent (will use your custom tools)
./hector call assistant "Use web_search to find information about MCP protocol"
```

---

## Common Patterns

### Pattern 1: Multi-Service Integration

```python
app = Server("multi-service")

@app.tool()
async def check_inventory(product_id: str) -> str:
    """Check inventory in multiple warehouses"""
    warehouses = ["US-EAST", "US-WEST", "EU-CENTRAL"]
    results = []
    
    for warehouse in warehouses:
        stock = get_stock(warehouse, product_id)
        results.append(f"{warehouse}: {stock} units")
    
    return "\n".join(results)
```

### Pattern 2: Aggregation

```python
@app.tool()
async def aggregate_metrics(metric_name: str, period: str) -> str:
    """Aggregate metrics from multiple sources"""
    sources = ["analytics", "database", "logs"]
    total = 0
    
    for source in sources:
        value = fetch_metric(source, metric_name, period)
        total += value
    
    return f"Total {metric_name} for {period}: {total}"
```

### Pattern 3: Pipeline

```python
@app.tool()
async def process_pipeline(input_data: str) -> str:
    """Run data through processing pipeline"""
    # Step 1: Validate
    validated = validate(input_data)
    
    # Step 2: Transform
    transformed = transform(validated)
    
    # Step 3: Enrich
    enriched = enrich(transformed)
    
    # Step 4: Store
    result_id = store(enriched)
    
    return f"Pipeline completed. Result ID: {result_id}"
```

---

## Summary

**Building custom MCP tools is the recommended way to extend Hector:**

✅ **Fast** - 5-10 minutes to add new capabilities  
✅ **Simple** - Just Python or TypeScript  
✅ **Clean** - No Hector code changes needed  
✅ **Flexible** - Any HTTP-capable language works  
✅ **Production-ready** - Deploy anywhere  

**Next Steps:**
1. Build your first MCP server (use examples above)
2. Test locally with Hector
3. Deploy to production
4. Share with the community!

**Resources:**
- [MCP Specification](https://modelcontextprotocol.io)
- [MCP Python SDK](https://github.com/modelcontextprotocol/python-sdk)
- [MCP TypeScript SDK](https://github.com/modelcontextprotocol/typescript-sdk)
- [Hector Tools Documentation](TOOLS.md)

---

**Start building your custom tools now! 🚀**

