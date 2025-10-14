---
layout: default
title: Quick Start
nav_order: 2
parent: Getting Started
description: "Get up and running with Hector in 5 minutes"
---

# Hector Quick Start

Get up and running with Hector in 5 minutes! Build your first AI agent using pure YAML configuration.

## Prerequisites

1. **API Key** - OpenAI API key (get one at [platform.openai.com](https://platform.openai.com))
2. **Installation** - See [Installation Guide](INSTALLATION) for all options
3. **2 minutes** of your time

## Option A: Zero-Config Mode (Fastest!)

**No config file needed - just instant AI assistance!**

### Step 1: Install Hector

See [Installation Guide](INSTALLATION) for all options (binary releases, Go install, Docker, etc.)

### Step 2: Set API Key

```bash
export OPENAI_API_KEY="sk-..."
```

### Step 3: Start Using!

```bash
# Call agent directly (no config needed!)
hector call assistant "Explain quantum computing in simple terms"

# Interactive chat
hector chat assistant

# With tools
hector call assistant "List files in current directory" --tools

# Custom model
hector call assistant "Write a haiku" --model gpt-4o
```

**🎉 That's it!** You're using Hector with zero configuration.

---

## Option B: Full Setup (For Advanced Features)

When you need custom prompts, multiple agents, or RAG, use a config file.

### Step 1: Create Your First Agent

Create `my-agent.yaml`:

```yaml
# LLM Configuration
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
    max_tokens: 2000

# Your First Agent
agents:
  assistant:
    name: "My Assistant"
    description: "A helpful AI assistant"
    llm: "gpt-4o"
    
    # Customize the agent's behavior
    prompt:
      system_role: |
        You are a helpful assistant. Be concise and friendly.
        Always explain your reasoning clearly.
    
    # Enable streaming for real-time responses
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 5
      enable_streaming: true

# Server Configuration
global:
  a2a_server:
    host: "localhost"
    port: 8080
```

### Step 2: Start the Server

Set your API key and start Hector:

```bash
export OPENAI_API_KEY="your-key-here"
./hector serve --config my-agent.yaml
```

You should see:
```
🚀 Starting Hector Server...
✅ My Assistant (assistant) - Ready
🌐 Server running at http://localhost:8080
```

### Step 3: Test Your Agent

Open a new terminal and try these commands:

**List available agents:**
```bash
./hector list
```

**Have a conversation:**
```bash
./hector chat assistant
```

**Single query:**
```bash
./hector call assistant "Explain quantum computing in simple terms"
```

**With streaming (real-time output):**
```bash
./hector call assistant "Write a Python function to calculate fibonacci" --stream
```

## Adding Tools (Advanced Example)

Want your agent to perform actions? Add tools! Create `coding-assistant.yaml`:

```yaml
# LLM Configuration
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7

# Coding Assistant with Tools
agents:
  coder:
    name: "Coding Assistant"
    description: "AI assistant that can write and execute code"
    llm: "gpt-4o"
    
    prompt:
      system_role: |
        You are an expert programmer. You can write files, run commands,
        and help users with coding tasks. Always explain what you're doing.
      
      tool_usage: |
        Use write_file to create/modify files.
        Use execute_command to run code and tests.
        Always test the code you write.
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 10
      enable_streaming: true

# Tool Configuration
tools:
  write_file:
    type: "write_file"
    enabled: true
    allowed_paths:
      - "./workspace/"
    max_file_size: "1MB"
  
  execute_command:
    type: "command"
    enabled: true
    allowed_commands:
      - "python3"
      - "node"
      - "go run"
      - "ls"
      - "cat"
    max_execution_time: "30s"
    working_directory: "./workspace"

# Server Configuration
global:
  a2a_server:
    host: "localhost"
    port: 8080
```

**Create workspace directory:**
```bash
mkdir workspace
```

**Start the server:**
```bash
./hector serve --config coding-assistant.yaml
```

**Test with tools:**
```bash
./hector call coder "Create a Python script that calculates prime numbers and test it"
```

The agent will:
1. 🤖 Write a Python file with prime number logic
2. 🏃 Execute the script to test it
3. 📝 Show you the results
4. 🔧 Fix any issues automatically

## Memory & Sessions

Hector supports persistent conversations and memory:

```yaml
agents:
  smart_assistant:
    name: "Smart Assistant"
    description: "Assistant with memory"
    llm: "gpt-4o"
    
    # Memory configuration
    memory:
      working_memory:
        strategy: "token_based"
        max_tokens: 4000
        summarization_threshold: 0.8
      
      long_term_memory:
        vector_store: "memory_store"
    
    reasoning:
      engine: "chain-of-thought"
      enable_streaming: true

# Vector store for long-term memory
document_stores:
  memory_store:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "agent_memory"
```

**Start a session:**
```bash
./hector chat smart_assistant
```

The agent will remember your conversation across the session and store important information in long-term memory!

**For complete CLI commands and options, see [CLI Guide](CLI_GUIDE)**

## Next Steps

🎉 **Congratulations!** You've built your first AI agent with Hector.

### Ready for More?

**🚀 [Multi-Agent Tutorial](tutorials/MULTI_AGENT_RESEARCH_PIPELINE)** - Build a 3-agent research system and see how Hector compares to LangChain (500+ lines Python → 120 lines YAML!)

**🤖 [Build Cursor-like Assistant](tutorials/BUILD_YOUR_OWN_CURSOR)** - Create a powerful AI coding assistant with semantic search and chain-of-thought reasoning.

### Explore Advanced Features

1. **[Memory Management](MEMORY)** - Working memory, long-term memory, and session persistence
2. **[Tools & Extensions](TOOLS)** - Built-in tools, MCP protocol, custom tools
3. **[Multi-Agent Systems](ARCHITECTURE#orchestrator-pattern)** - Agent orchestration and coordination
4. **[Authentication](AUTHENTICATION)** - JWT, OAuth2, API keys
5. **[Configuration Reference](CONFIGURATION)** - Complete YAML configuration guide

## What You've Learned

✅ **Pure YAML Configuration** - No code required to build AI agents  
✅ **Streaming Responses** - Real-time output for better UX  
✅ **Tool Integration** - Agents that can perform actions  
✅ **Memory Management** - Persistent conversations and context  
✅ **A2A Protocol** - Industry-standard agent communication  

**Ready to build something amazing? Check out the advanced tutorials!** 🚀

