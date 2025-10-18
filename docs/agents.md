---
title: Building Agents
description: Complete guide to building single agents with prompts, tools, RAG, sessions, and streaming
---

# Building AI Agents with Hector

**Declarative, Powerful, Production-Ready**

## Overview

Hector's core strength is making it incredibly easy to build sophisticated AI agents **without writing code**. Define everything in YAML, and Hector handles the complex orchestration, tool integration, context management, and streaming.

**What makes Hector's agents powerful:**

- **100% Declarative** - Pure YAML, zero code
- **Prompt Customization** - Slot-based system for fine control
- **Reasoning Strategies** - Chain-of-thought or supervisor
- **Structured Output** - Provider-aware JSON/XML/Enum for reliable data extraction
- **Built-in Tools** - Search, file ops, commands, todos
- **Plugin Extensibility** - Add custom LLMs, databases, tools
- **Real-Time Streaming** - Token-by-token output
- **Multi-Turn Sessions** - Conversation history & context
- **Document Stores** - Semantic search with RAG
- **Production Ready** - Error handling, logging, monitoring

---

## Quick Example

```yaml
agents:
  coding_assistant:
    name: "Coding Assistant"
    description: "Helps write, debug, and review code"
    
    # LLM Configuration
    llm: "gpt-4o"
    
    # Prompt Customization
    prompt:
      system_role: |
        You are an expert software engineer who writes clean,
        maintainable code and explains your reasoning clearly.
      
      reasoning_instructions: |
        1. Understand the requirement fully
        2. Consider edge cases
        3. Write clean, testable code
        4. Explain your decisions
      
      tool_usage: |
        Use write_file to create/update files
        Use execute_command to run tests
        Use search to find relevant documentation
    
    # Reasoning Strategy
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 15
      enable_streaming: true
    
    # Document Stores (RAG)
    document_stores:
      - "codebase_docs"

# LLM Provider
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
    max_tokens: 4000
```

!!! success "That's it!"
    Start the server and you have a production-ready coding assistant with RAG, tools, and streaming.

---

## Table of Contents

- [Agent Anatomy](#agent-anatomy)
- [Prompt Customization](#prompt-customization)
- [Reasoning Strategies](#reasoning-strategies)
- [Built-in Tools](#built-in-tools)
- [Document Stores & RAG](#document-stores-rag)
- [Sessions & Streaming](#sessions-streaming)
- [Plugin System](#plugin-system)
- [Real-World Examples](#real-world-examples)

---

## Agent Anatomy

Every Hector agent consists of:

```yaml
agents:
  my_agent:
    # Identity
    name: "Agent Name"
    description: "What this agent does"
    visibility: "public"  # public, internal, private
    
    # Core Components
    llm: "llm-id"                    # Which LLM to use
    prompt: { }                      # Prompt customization
    reasoning: { }                   # Reasoning strategy
    document_stores: []              # RAG data sources
    
    # Search Configuration (optional)
    search:
      enabled: true
      result_limit: 10
```

### 1. **LLM (Required)**

The language model powering your agent:

```yaml
# Agent definition
agents:
  my_agent:
    llm: "claude-3-5-sonnet"

# LLM provider definition
llms:
  claude-3-5-sonnet:
    type: "anthropic"
    model: "claude-3-5-sonnet-20241022"
    api_key: "${ANTHROPIC_API_KEY}"
    max_tokens: 8000
    temperature: 0.7
```

**Supported Providers:**
- :material-openai: **OpenAI** (GPT-4o, GPT-4, GPT-3.5)
- :material-anthropic: **Anthropic** (Claude 3.5 Sonnet, Claude 3 Opus/Haiku)
- :material-google: **Google Gemini** (Gemini 2.0 Flash, Gemini 1.5 Pro/Flash)
- :material-cog: **Custom** via gRPC plugins

### 2. **Prompt (Optional but Recommended)**

Fine-tune your agent's behavior:

```yaml
prompt:
  system_role: |
    You are a [role description]
  
  reasoning_instructions: |
    How to approach problems
  
  tool_usage: |
    How to use available tools
  
  output_format: |
    How to structure responses
  
  communication_style: |
    Tone and formatting preferences
```

### 3. **Reasoning (Optional)**

Control how your agent thinks:

```yaml
reasoning:
  engine: "chain-of-thought"  # or "supervisor"
  max_iterations: 10          # How many reasoning cycles
  enable_streaming: true      # Real-time output
```

### 4. **Document Stores (Optional)**

Enable RAG (Retrieval-Augmented Generation):

```yaml
document_stores:
  - "company_docs"
  - "product_specs"
```

---

## Prompt Customization

Hector uses a **slot-based prompt system** that gives you fine-grained control without losing the benefits of built-in reasoning.

### Slot System

**6 predefined slots:**

| Slot | Purpose | Example |
|------|---------|---------|
| `system_role` | Define agent identity | "You are a Python expert" |
| `reasoning_instructions` | How to think | "Use step-by-step reasoning" |
| `tool_usage` | How to use tools | "Use search for documentation" |
| `output_format` | Response structure | "Use markdown formatting" |
| `communication_style` | Tone & style | "Be concise and technical" |
| `additional` | Custom instructions | Domain-specific rules |

### Basic Customization

```yaml
prompt:
  system_role: |
    You are a customer support agent for TechCorp.
    You are empathetic, patient, and solution-oriented.
  
  communication_style: |
    - Use friendly, professional language
    - Show empathy for customer frustrations
    - Provide actionable solutions
    - End with "Is there anything else I can help with?"
```

### Advanced Customization

```yaml
prompt:
  system_role: |
    You are a senior code reviewer with 10+ years of experience.
    You focus on maintainability, performance, and security.
  
  reasoning_instructions: |
    For each code review:
    1. Check for security vulnerabilities
    2. Assess code maintainability
    3. Look for performance issues
    4. Suggest specific improvements
    5. Provide code examples for fixes
  
  tool_usage: |
    - Use search to find similar patterns in the codebase
    - Use execute_command to run linters/tests
    - Use write_file only if explicitly asked to fix code
  
  output_format: |
    Structure reviews as:
    ## Security
    [findings]
    
    ## Maintainability
    [findings]
    
    ## Performance
    [findings]
    
    ## Recommendations
    [specific changes with code examples]
```

### Domain-Specific Agents

=== "Research Agent"
    ```yaml
    prompt:
      system_role: |
        You are a research analyst who synthesizes information
        from multiple sources into clear, actionable insights.
      
      reasoning_instructions: |
        1. Break down research questions into searchable topics
        2. Use search tool to gather information
        3. Cross-reference multiple sources
        4. Identify patterns and contradictions
        5. Synthesize findings into structured insights
    ```

=== "Debugging Agent"
    ```yaml
    prompt:
      system_role: |
        You are a debugging expert who finds root causes quickly.
      
      reasoning_instructions: |
        1. Reproduce the issue (use execute_command)
        2. Form hypotheses about the cause
        3. Test each hypothesis systematically
        4. Find root cause, not just symptoms
        5. Suggest preventive measures
    ```

---

## Reasoning Strategies

Hector provides two built-in reasoning strategies:

### 1. Chain-of-Thought (Default)

**Best for:** Single-agent tasks, general problem-solving

**How it works:**
- Agent thinks step-by-step
- Can use tools at any point
- Automatically decides when task is complete
- Fast and cost-effective

**Configuration:**
```yaml
reasoning:
  engine: "chain-of-thought"
  max_iterations: 10
  enable_streaming: true
```

**Characteristics:**
- One LLM call per iteration
- Implicit planning
- Tool execution with automatic continuation
- Natural conversation flow
- Fast response times

**Use cases:**
- Coding assistants
- Research agents
- Customer support
- Content creation
- General Q&A

### 2. Supervisor (For Orchestration)

**Best for:** Multi-agent coordination

**How it works:**
- Specialized prompts for task decomposition
- Guides agent selection and delegation
- Helps synthesize results from multiple agents
- Works with `agent_call` tool

**Configuration:**
```yaml
reasoning:
  engine: "supervisor"
  max_iterations: 20  # More iterations for complex orchestration
  enable_streaming: true
```

**Characteristics:**
- Task decomposition guidance
- Agent delegation patterns
- Result synthesis support
- Based on chain-of-thought with orchestration enhancements

**Use cases:**
- Multi-agent workflows
- Complex research pipelines
- Cross-functional tasks
- Hierarchical processing

!!! info "Multi-Agent Orchestration"
    See [Architecture](architecture.md) for details on multi-agent orchestration patterns.

!!! tip "Want to see a complete multi-agent system in action?"
    Multi-agent workflows that would require 500+ lines of Python code can be implemented with simple YAML configuration in Hector.

---

## Built-in Tools

Hector agents have access to powerful built-in tools:

### 1. **execute_command**

Run shell commands securely:

```yaml
# Default configuration (secure whitelist)
tools:
  execute_command:
    type: "command"
    enabled: true
    allowed_commands:
      - "cat"
      - "ls"
      - "grep"
      - "git"
      - "npm"
      - "go"
    max_execution_time: "30s"
```

**Agent usage:**
```
User: What files are in this directory?
Agent: Let me check...
Tool: execute_command(command="ls -la")
Agent: Here are the files: [lists files]
```

### 2. **search**

Semantic search across document stores:

```yaml
# Automatic if document_stores configured
document_stores:
  codebase_docs:
    type: "qdrant"
    collection: "my_codebase"
```

**Agent usage:**
```
User: How do I use the authentication module?
Agent: Let me search the documentation...
Tool: search(query="authentication module usage", stores=["codebase_docs"])
Agent: Based on the docs, here's how to use it: [explains]
```

### 3. **write_file**

Create and modify files:

```yaml
tools:
  write_file:
    type: "write_file"
    enabled: true
    allowed_paths:
      - "./src/"
      - "./docs/"
    max_file_size: "10MB"
```

**Agent usage:**
```
User: Create a README file
Agent: I'll create that for you...
Tool: write_file(path="README.md", content="# Project Title...")
Agent: Created README.md with [description]
```

### 4. **search_replace**

Find and replace in files:

```yaml
tools:
  search_replace:
    type: "search_replace"
    enabled: true
```

**Agent usage:**
```
User: Rename the function getUserData to fetchUserData
Agent: I'll update that...
Tool: search_replace(file="api.js", old="getUserData", new="fetchUserData")
Agent: Renamed in api.js
```

### 5. **todo_write**

Task management and tracking:

```yaml
tools:
  todo:
    type: "todo"
    enabled: true
```

**Agent usage:**
```
Agent: Breaking this down into steps...
Tool: todo_write(tasks=[
  {id: "1", content: "Research options", status: "in_progress"},
  {id: "2", content: "Compare solutions", status: "pending"},
  {id: "3", content: "Write recommendation", status: "pending"}
])
```

### Tool Security

**Built-in safety features:**
- Command whitelisting
- Path restrictions
- Execution timeouts
- Size limits
- Sandboxing options

**Configure per agent:**
```yaml
agents:
  safe_agent:
    # Only has read-only tools
    # (Hector auto-filters based on tool config)
```

---

## Document Stores & RAG

Enable Retrieval-Augmented Generation (RAG) to give your agents domain knowledge.

### Setup

**1. Define Document Store:**
```yaml
document_stores:
  company_knowledge:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "company_docs"
    
  api_docs:
    type: "qdrant"
    url: "http://localhost:6333"
    collection: "api_reference"
```

**2. Link to Agent:**
```yaml
agents:
  support_agent:
    llm: "gpt-4o"
    document_stores:
      - "company_knowledge"
      - "api_docs"
    
    search:
      enabled: true
      result_limit: 10
```

**3. Agent automatically uses search:**
```
User: How do I reset my password?
Agent: Let me check our documentation...
[Automatic search in company_knowledge and api_docs]
Agent: Here's how to reset your password: [answer with citations]
```

### How It Works

1. **Agent receives question**
2. **Automatically decides if search needed**
3. **Semantic search across document stores**
4. **Retrieves relevant chunks**
5. **Synthesizes answer with context**
6. **Cites sources**

### Best Practices

**Organize by domain:**
```yaml
document_stores:
  product_specs:     # Product features and specs
  api_reference:     # API documentation
  troubleshooting:   # Common issues and solutions
  company_policies:  # HR, legal, compliance
```

**Configure search behavior:**
```yaml
search:
  enabled: true
  result_limit: 10      # Balance between context and cost
  min_similarity: 0.7   # Filter low-relevance results
```

**Prompt guidance:**
```yaml
prompt:
  tool_usage: |
    Use search when:
    - User asks about specific features/docs
    - You need factual information
    - You should cite sources
    
    Don't search for:
    - General coding questions
    - Basic troubleshooting
    - Opinion-based queries
```

---

## Sessions & Streaming

### Multi-Turn Sessions

Enable conversation context and history:

**Create session:**
```bash
curl -X POST http://localhost:8080/sessions \
  -d '{"agentId": "support_agent"}'
# Response: {"sessionId": "550e8400-..."}
```

**Chat in session:**
```bash
# Message 1
curl -X POST http://localhost:8080/sessions/550e8400-.../tasks \
  -d '{"input":{"type":"text/plain","content":"My name is Alice"}}'

# Message 2 (agent remembers Alice)
curl -X POST http://localhost:8080/sessions/550e8400-.../tasks \
  -d '{"input":{"type":"text/plain","content":"What is my name?"}}'
# Response: "Your name is Alice"
```

**Benefits:**
- Conversation history maintained
- Context across multiple turns
- Personalized responses
- Follow-up questions work naturally

### Real-Time Streaming

Get token-by-token output via Server-Sent Events (SSE) per A2A specification:

```bash
# Using curl
curl -N -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"message":{"role":"user","parts":[{"type":"text","text":"Explain quantum computing"}]}}' \
  http://localhost:8080/agents/my_agent/message/stream

# Output (SSE format):
# event: message
# data: {"task_id":"task-1","message":{"role":"assistant","parts":[{"type":"text","text":"Quantum"}]}}
#
# event: message
# data: {"task_id":"task-1","message":{"role":"assistant","parts":[{"type":"text","text":" computing"}]}}
#
# event: status
# data: {"task_id":"task-1","status":{"state":"completed"}}
```

**Using Hector CLI:**
```bash
# Streaming is enabled by default in chat and --stream flag in call
hector chat my_agent         # Interactive with streaming
hector call my_agent "prompt" --stream  # Single call with streaming
```

**Benefits:**
- Immediate feedback to users
- Better UX for long responses
- Cancel long-running tasks
- Progress indicators
- A2A-compliant

**Configure:**
```yaml
reasoning:
  enable_streaming: true  # Enable streaming output
```

---

## Plugin System

Extend Hector with custom LLMs, databases, or tools via gRPC plugins.

### Plugin Types

- :material-brain: **LLM Plugins** - Add custom language models
- :material-database: **Database Plugins** - Add custom vector databases
- :material-wrench: **Tool Plugins** - Add custom tools/capabilities

### Example: Custom LLM Plugin

```yaml
# Configuration
plugins:
  llm_providers:
    my_custom_llm:
      type: "grpc"
      path: "./plugins/my-llm-plugin"
      enabled: true

# Use in agent
llms:
  custom:
    type: "plugin:my_custom_llm"
    model: "custom-model-v1"

agents:
  my_agent:
    llm: "custom"
```

**Plugin interface:**
- gRPC-based for performance
- Language-agnostic (Go, Python, Rust, etc.)
- Hot-reload support
- Sandboxed execution

!!! info "Implementation Details"
    See [Architecture - Plugin System](architecture.md#extension-points) for implementation details.

---

## Real-World Examples

### Example 1: Research Agent

```yaml
agents:
  researcher:
    name: "Research Analyst"
    description: "Conducts comprehensive research and analysis"
    llm: "gpt-4o"
    
    prompt:
      system_role: |
        You are a thorough research analyst who synthesizes
        information from multiple sources into clear insights.
      
      reasoning_instructions: |
        1. Break down research question into sub-topics
        2. Use search to gather information from multiple sources
        3. Cross-reference and verify findings
        4. Identify patterns and contradictions
        5. Synthesize into structured, actionable insights
      
      output_format: |
        Structure research as:
        ## Executive Summary
        ## Key Findings
        ## Detailed Analysis
        ## Recommendations
        ## Sources
    
    document_stores:
      - "company_research"
      - "market_data"
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 15

llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.3  # Lower for factual research
    max_tokens: 8000
```

### Example 2: Coding Assistant

```yaml
agents:
  code_helper:
    name: "Coding Assistant"
    description: "Helps write, debug, and review code"
    llm: "claude-3-5-sonnet"
    
    prompt:
      system_role: |
        You are an expert software engineer who writes clean,
        maintainable, well-tested code.
      
      reasoning_instructions: |
        1. Understand requirements fully before coding
        2. Consider edge cases and error handling
        3. Write clean, self-documenting code
        4. Include comments for complex logic
        5. Suggest tests for new code
      
      tool_usage: |
        - Use search to find similar patterns in codebase
        - Use write_file to create/update files
        - Use execute_command to run tests/linters
        - Use search_replace for refactoring
    
    document_stores:
      - "codebase_index"
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 20
      enable_streaming: true

llms:
  claude-3-5-sonnet:
    type: "anthropic"
    model: "claude-3-5-sonnet-20241022"
    api_key: "${ANTHROPIC_API_KEY}"
    max_tokens: 8000
```

### Example 3: Customer Support

```yaml
agents:
  support:
    name: "Customer Support Agent"
    description: "Helps customers with questions and issues"
    llm: "gpt-4o-mini"  # Cost-effective for support
    
    prompt:
      system_role: |
        You are a friendly, patient customer support agent
        for TechCorp. You prioritize customer satisfaction.
      
      reasoning_instructions: |
        1. Listen carefully to the customer's issue
        2. Ask clarifying questions if needed
        3. Search knowledge base for solutions
        4. Provide clear, step-by-step guidance
        5. Verify the issue is resolved
        6. Document the interaction
      
      communication_style: |
        - Be warm and empathetic
        - Use simple, non-technical language
        - Provide actionable steps
        - Follow up to ensure satisfaction
        - End with: "Is there anything else I can help with?"
    
    document_stores:
      - "faq"
      - "troubleshooting_guides"
      - "product_documentation"
    
    search:
      enabled: true
      result_limit: 5
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 10
      enable_streaming: true  # Better UX

llms:
  gpt-4o-mini:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7
    max_tokens: 2000
```

### Example 4: Data Extraction with Structured Output

**Use case:** Extract structured data from unstructured text for downstream processing.

```yaml
agents:
  invoice_extractor:
    name: "Invoice Data Extractor"
    description: "Extracts structured data from invoices and receipts"
    llm: "gemini-llm"  # Gemini's structured output is excellent
    
    prompt:
      system_role: |
        You are an expert at extracting structured invoice data.
        Extract all relevant fields accurately.
      
      reasoning_instructions: |
        1. Identify the document type (invoice, receipt, bill)
        2. Extract company information
        3. Extract line items with prices
        4. Calculate totals
        5. Identify payment details
      
      output_format: |
        Always output valid JSON matching the schema.
        Use null for missing fields.
    
    # Structured output configuration
    structured_output:
      format: "json"
      schema:
        type: "object"
        properties:
          document_type:
            type: "string"
            enum: ["invoice", "receipt", "bill", "quote"]
          vendor:
            type: "object"
            properties:
              name: {type: "string"}
              address: {type: "string"}
              tax_id: {type: "string"}
          invoice_number: {type: "string"}
          date: {type: "string", format: "date"}
          line_items:
            type: "array"
            items:
              type: "object"
              properties:
                description: {type: "string"}
                quantity: {type: "number"}
                unit_price: {type: "number"}
                total: {type: "number"}
          subtotal: {type: "number"}
          tax: {type: "number"}
          total: {type: "number"}
        required: ["document_type", "vendor", "line_items", "total"]
    
    reasoning:
      engine: "chain-of-thought"
      max_iterations: 5

llms:
  gemini-llm:
    type: "gemini"
    model: "gemini-2.0-flash"
    api_key: "${GEMINI_API_KEY}"
    temperature: 0.1  # Low for accuracy
    max_tokens: 2048
```

**Why structured output is perfect here:**
- **Reliable parsing** - No regex or text parsing needed
- **Type safety** - Guaranteed data types (numbers, dates, enums)
- **Validation** - Required fields enforced by schema
- **Downstream integration** - Direct JSON → database/API
- **Error reduction** - Schema prevents malformed output

**Usage:**
```bash
curl http://localhost:8080/agents/invoice_extractor/message/send \
  -d '{"message":{"role":"user","parts":[{"type":"text","text":"<invoice text>"}]}}'

# Response: Perfect JSON ready for your database
{
  "document_type": "invoice",
  "vendor": {"name": "Acme Corp", "address": "123 Main St"},
  "invoice_number": "INV-2024-001",
  "line_items": [{"description": "Widget", "quantity": 5, "unit_price": 10.00, "total": 50.00}],
  "total": 50.00
}
```

!!! info "More Examples"
    Structured output is perfect for data extraction, API integration, and any scenario requiring reliable, parseable responses.
