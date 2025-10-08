# Hector Configuration Reference

Complete guide to configuring Hector AI Assistant.

> **📖 External A2A Agents:** For integrating external agents via URL, see [EXTERNAL_AGENTS.md](EXTERNAL_AGENTS.md). This document covers native agent configuration only.

---

## Table of Contents

- [Quick Start](#quick-start)
- [Configuration Structure](#configuration-structure)
- [Agent Configuration](#agent-configuration)
- [LLM Providers](#llm-providers)
- [Prompt Configuration](#prompt-configuration)
- [Reasoning Configuration](#reasoning-configuration)
- [Tools Configuration](#tools-configuration)
- [Database & Embedders](#database--embedders)
- [Document Stores](#document-stores)
- [Best Practices](#best-practices)

---

## Quick Start

###

 Minimal Configuration

```yaml
agents:
  assistant:
    name: "My Assistant"
    llm: "main-llm"

llms:
  main-llm:
    type: "anthropic"
    model: "claude-3-7-sonnet-latest"
    api_key: "${ANTHROPIC_API_KEY}"
```

### Recommended Configuration

```yaml
agents:
  assistant:
    name: "Coding Assistant"
    llm: "main-llm"
    
    prompt:
      include_tools: true
      include_history: true
      max_history_messages: 10
    
    reasoning:
      engine: "chain-of-thought"
      # max_iterations: 100  # Safety valve only - LLM naturally terminates
      show_debug_info: true
      enable_streaming: true

llms:
  main-llm:
    type: "anthropic"
    model: "claude-3-7-sonnet-latest"
    api_key: "${ANTHROPIC_API_KEY}"
    temperature: 0.1
    max_tokens: 16000
    timeout: 60
```

---

## Configuration Structure

```yaml
# Agent definitions (required)
agents:
  <agent-name>:
    # Agent configuration

# LLM providers (required)
llms:
  <llm-name>:
    # LLM configuration

# Optional sections
tools:           # Tool configuration
databases:       # Database providers
embedders:       # Embedding providers
document_stores: # Knowledge bases
```

---

## Agent Configuration

**Note:** This section covers **native agents** only. For external A2A agents, see [EXTERNAL_AGENTS.md](EXTERNAL_AGENTS.md).

### Basic Structure

```yaml
agents:
  <agent-name>:
    name: string           # Display name
    description: string    # Agent description (optional)
    visibility: string     # Agent visibility: "public" (default), "internal", or "private"
    llm: string           # Reference to LLM config
    
    prompt:               # Prompt configuration
      system_prompt: string          # Full custom prompt (override)
      prompt_slots: map[string]string # Slot-based customization
      include_tools: bool            # Include tool descriptions
      include_context: bool          # Include semantic search results
      include_history: bool          # Include conversation history
      max_history_messages: int      # History limit (default: 10)
    
    reasoning:            # Reasoning engine configuration
      engine: string                 # "chain-of-thought" (only supported)
      max_iterations: int            # Safety valve only (default: 100, rarely hit)
      show_debug_info: bool          # Show detailed output
      enable_streaming: bool         # Enable streaming mode
    
    document_stores: []string # Document store references
    database: string         # Database provider reference
    embedder: string         # Embedder reference
    search: map              # Search configuration
```

### Prompt Configuration

#### Slot-Based Customization (Recommended)

```yaml
agents:
  assistant:
    prompt:
      prompt_slots:
        system_role: |
          You are an AI coding assistant.
        
        reasoning_instructions: |
          Your goal is to help users with coding tasks.
          Be thorough and accurate.
        
        tool_usage: |
          CRITICAL: ACT IMMEDIATELY, DON'T ANNOUNCE.
          Use tools without preamble.
        
        output_format: |
          Provide clear, accurate responses.
        
        communication_style: |
          Use backticks for code.
        
        additional: |
          <task_management>
          For complex tasks, create todos first.
          </task_management>
```

**Available Slots:**
- `system_role` - Who the assistant is
- `reasoning_instructions` - How to think
- `tool_usage` - How to use tools
- `output_format` - Response formatting
- `communication_style` - How to communicate
- `additional` - Extra instructions (task management, etc.)

#### Full Override

```yaml
agents:
  assistant:
    prompt:
      system_prompt: |
        You are a helpful assistant.
        [Your complete custom prompt here]
```

**Note:** `system_prompt` overrides all slots.

### Reasoning Configuration

```yaml
agents:
  assistant:
    reasoning:
      engine: "chain-of-thought"           # Only supported engine
      # max_iterations: 100                # Optional safety valve (default: 100)
      show_debug_info: true                # Show thinking/reflection
      enable_streaming: true               # Real-time output (default: true)
      enable_structured_reflection: true   # LLM-based reflection (default: true)
      enable_completion_verification: false # Task completion check (default: false)
      enable_goal_extraction: false        # Goal decomposition (default: false)
```

**Philosophy:**
- **Trust the LLM** - Loops naturally terminate when no more tool calls
- `max_iterations` is a safety valve only, not an artificial constraint
- Matches Cursor's approach: continue until work is done

**Engine Options:**
- `chain-of-thought` - Fast, iterative reasoning (Cursor-like)

**Debug Info Includes:**
- Iteration numbers
- Token usage
- Tool execution labels
- Self-reflection (grayed out)
- Timing information

---

### Structured Output Features (Smart Defaults)

Hector enables **structured reflection by default** for better quality output.

#### 1. Structured Reflection (Default: ON)

**What it does:** Uses LLM to analyze tool execution results with structured output, providing confidence scores and recommendations.

**Benefits:**
- +13% quality improvement
- Better error recovery
- More confident decision-making
- Structured insights for debugging

**Cost:** +20% token usage (minimal for most workloads)

**Disable if needed:**
```yaml
reasoning:
  enable_structured_reflection: false  # Falls back to heuristic analysis
```

#### 2. Completion Verification (Default: OFF)

**What it does:** Verifies task completion before stopping, reducing premature exits.

**Benefits:**
- Fewer incomplete responses
- Higher task completion rate
- Quality assurance before final output

**Cost:** +10-15% token usage (only triggers when agent thinks it's done)

**Enable if needed:**
```yaml
reasoning:
  enable_completion_verification: true  # Recommended for critical tasks
```

#### 3. Goal Extraction (Default: OFF, Supervisor only)

**What it does:** Decomposes complex tasks into subtasks with dependencies (supervisor strategy only).

**Benefits:**
- Better multi-agent orchestration
- Structured task planning
- Clear execution order

**Cost:** +5-10% token usage (only on first iteration)

**Enable if needed:**
```yaml
reasoning:
  engine: "supervisor"
  enable_goal_extraction: true  # Only works with supervisor engine
```

---

### Cost Analysis & Optimization

**Default configuration (structured reflection only):**
- Quality: **+13% improvement**
- Cost: **+20% token usage**
- ROI: **Best balance for most use cases**

**All features enabled:**
- Quality: **+25% improvement**
- Cost: **+35-40% token usage**
- ROI: **Recommended for critical/complex tasks**

**Cost optimization strategies:**

1. **Use smart defaults** (structured reflection only) for general tasks
2. **Enable completion verification** for user-facing or critical tasks
3. **Enable goal extraction** only for complex multi-agent workflows
4. **Disable structured reflection** for high-volume, cost-sensitive tasks:
   ```yaml
   reasoning:
     enable_structured_reflection: false  # Heuristic fallback
   ```

5. **Use smaller models** (e.g., `gpt-4o-mini`, `claude-3-haiku`) with structured features for better ROI:
   ```yaml
   llms:
     cost_effective:
       type: "openai"
       model: "gpt-4o-mini"  # Cheaper model
       max_tokens: 2000
   
   agents:
     support_agent:
       llm: "cost_effective"
       reasoning:
         enable_structured_reflection: true  # Small model + structured output = quality + savings
   ```

**Benchmark data:** See `docs/benchmarks/` for detailed performance and cost analysis across providers.

---

## LLM Providers

### Anthropic (Claude)

```yaml
llms:
  main-llm:
    type: "anthropic"
    model: "claude-3-7-sonnet-latest"  # or claude-3-5-haiku-latest
    api_key: "${ANTHROPIC_API_KEY}"    # Environment variable
    host: "https://api.anthropic.com"  # Optional, default shown
    temperature: 0.1                    # 0.0-1.0
    max_tokens: 16000                   # Max response tokens
    timeout: 60                         # Request timeout (seconds)
    max_retries: 5                      # Rate limit retry attempts (default: 5)
    retry_delay: 2                      # Base delay in seconds (default: 2, exponential backoff)
```

**Supported Models:**
- `claude-3-7-sonnet-latest` - Most capable (recommended)
- `claude-3-5-haiku-latest` - Fastest, cheaper
- `claude-sonnet-4.5-20250514` - Specific version

**Rate Limits:**
- Handled automatically with exponential backoff
- Default: 5 retries with 2s, 4s, 8s, 16s, 32s delays (total: ~62s)
- Configurable via `max_retries` and `retry_delay`
- Supports "trust the LLM" philosophy (up to 100 iterations)

### OpenAI

```yaml
llms:
  main-llm:
    type: "openai"
    model: "gpt-4o"                     # or gpt-4o-mini, gpt-3.5-turbo
    api_key: "${OPENAI_API_KEY}"
    host: "https://api.openai.com/v1"  # Optional
    temperature: 0.1
    max_tokens: 16000
    timeout: 60
    max_retries: 5                      # Rate limit retry attempts (default: 5)
    retry_delay: 2                      # Base delay in seconds (default: 2, exponential backoff)
```

### Google Gemini

```yaml
llms:
  main-llm:
    type: "gemini"
    model: "gemini-2.0-flash"                              # or gemini-1.5-pro, gemini-1.5-flash
    api_key: "${GEMINI_API_KEY}"
    host: "https://generativelanguage.googleapis.com"     # Default
    temperature: 0.7
    max_tokens: 2048
    timeout: 60
    max_retries: 3
    retry_delay: 2
```

**Available Models:**
- `gemini-2.0-flash` - Latest, fastest, cost-effective
- `gemini-1.5-pro` - Most powerful, best for complex tasks
- `gemini-1.5-flash` - Balanced speed and capability

**Supported Models:**
- `gpt-4o` - Most capable
- `gpt-4o-mini` - Faster, cheaper
- `gpt-3.5-turbo` - Budget option

---

## Structured Output Configuration

Configure schema-validated JSON/XML/Enum output for reliable data extraction. Works with OpenAI, Anthropic, and Gemini.

**When to use:**
- Data extraction (invoices, forms, documents)
- Classification (sentiment, priority, category)
- Entity extraction (names, dates, locations, amounts)
- API integrations requiring specific JSON formats
- Database record creation
- Form filling from unstructured text

### Basic JSON Schema

```yaml
agents:
  data_extractor:
    llm: "openai-llm"
    structured_output:
      format: "json"
      schema:
        type: "object"
        properties:
          name: {type: "string"}
          age: {type: "number"}
          email: {type: "string", format: "email"}
        required: ["name", "email"]
```

### Provider-Specific Optimizations

#### OpenAI: Strict Mode
```yaml
agents:
  openai_extractor:
    llm: "openai-llm"
    structured_output:
      format: "json"
      schema:  # Schema validated strictly
        type: "object"
        properties:
          sentiment: {type: "string", enum: ["positive", "negative", "neutral"]}
          confidence: {type: "number", minimum: 0, maximum: 1}
        required: ["sentiment", "confidence"]
```

#### Anthropic: Prefill Technique
```yaml
agents:
  anthropic_extractor:
    llm: "claude-llm"
    structured_output:
      format: "json"
      schema: {<your-schema>}
      prefill: '{"sentiment":'  # Forces response to start with JSON
```

#### Gemini: Property Ordering
```yaml
agents:
  gemini_extractor:
    llm: "gemini-llm"
    structured_output:
      format: "json"
      schema: {<your-schema>}
      property_ordering: ["name", "age", "email"]  # Consistent field order
```

### Classification with Enum

```yaml
agents:
  priority_classifier:
    llm: "gemini-llm"
    structured_output:
      format: "enum"
      enum: ["Urgent", "High", "Medium", "Low"]
```

### Real-World Example: Invoice Extraction

```yaml
agents:
  invoice_parser:
    llm: "gemini-llm"
    structured_output:
      format: "json"
      schema:
        type: "object"
        properties:
          vendor: {type: "string"}
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
          total: {type: "number"}
        required: ["vendor", "invoice_number", "total"]
```

**Benefits:**
- ✅ No regex or text parsing
- ✅ Type-safe outputs (strings, numbers, booleans, arrays)
- ✅ Required field validation
- ✅ Direct database/API integration
- ✅ Reduced error rates

**See [Structured Output Guide](STRUCTURED_OUTPUT.md) for complete documentation and examples.**

---

## Tools Configuration

### Zero-Config (Default)

No configuration needed! Default tools are automatically registered:

```yaml
# Just don't specify tools section
# Defaults: execute_command, search, write_file, search_replace, todo_write
```

### Custom Configuration

```yaml
tools:
  execute_command:
    type: command
    allowed_commands: ["ls", "cat", "grep", "find", "git", "go", "npm"]
    working_directory: "./"
    max_execution_time: "30s"
    enable_sandboxing: true
  
  write_file:
    type: write_file
    max_file_size: 1048576
    allowed_extensions: [".go", ".py", ".js", ".ts", ".md"]
    working_directory: "./"
  
  search_replace:
    type: search_replace
    max_replacements: 100
    working_directory: "./"
  
  search:
    type: search
    document_stores: ["codebase"]
    default_limit: 10
    max_limit: 50
    max_results: 100
  
  todo_write:
    type: todo
```

### Available Tools

| Tool | Type | Description |
|------|------|-------------|
| `execute_command` | command | Execute shell commands |
| `write_file` | write_file | Create/overwrite files |
| `search_replace` | search_replace | Precise text replacement |
| `search` | search | Semantic codebase search |
| `todo_write` | todo | Task management |

---

## Database & Embedders

### Qdrant (Vector Database)

```yaml
databases:
  qdrant:
    type: "qdrant"
    host: "localhost"
    port: 6334                # Default Qdrant port
    collection_name: "docs"   # Collection for vectors
```

**Setup:**
```bash
docker run -p 6334:6334 qdrant/qdrant
```

### Ollama Embeddings

```yaml
embedders:
  embedder:
    type: "ollama"
    model: "nomic-embed-text"  # Recommended for code
    host: "http://localhost:11434"
```

**Setup:**
```bash
ollama serve
ollama pull nomic-embed-text
```

---

## Document Stores

### Configuration

```yaml
document_stores:
  hector-code:
    name: "hector-code"
    path: "."                        # Directory to index
    source: "directory"              # Source type
    
    include_patterns:
      - "*.go"
      - "*.md"
      - "*.py"
    
    exclude_patterns:
      - "vendor/**"
      - ".git/**"
      - "**/testdata/**"
      - "node_modules/**"
    
    max_file_size: 1048576          # 1MB
    watch_changes: false             # Auto-reindex on changes
    
    database: "qdrant"               # Database reference
    embedder: "embedder"             # Embedder reference
```

### Linking to Agent

```yaml
agents:
  assistant:
    document_stores:
      - "hector-code"
    database: "qdrant"
    embedder: "embedder"
    
    prompt:
      include_context: true  # Enable semantic injection
```

---

## Best Practices

### Performance

1. **Token Limits**
   - Development: 8000-16000 tokens
   - Production: 4000-8000 tokens (faster, cheaper)

2. **History Management**
   ```yaml
   prompt:
     max_history_messages: 5-10  # Balance context vs cost
   ```

3. **Safety Valve** (Optional)
   ```yaml
   reasoning:
     max_iterations: 100  # Safety only, rarely needed
   ```
   
   **Note:** Default is 100. Lower only if you need strict guarantees.

### Cost Optimization

1. **Use Cheaper Models for Simple Tasks**
   ```yaml
   # For simple queries
   model: "claude-3-5-haiku-latest"  # or gpt-4o-mini
   
   # For complex reasoning
   model: "claude-3-7-sonnet-latest"  # or gpt-4o
   ```

2. **Disable Semantic Search When Not Needed**
   ```yaml
   prompt:
     include_context: false  # Saves embedding costs
   ```

3. **Limit Tool Access**
   ```yaml
   tools:
     execute_command:
       type: command
       allowed_commands: ["ls", "cat", "pwd"]  # Only safe commands
     # Don't include expensive tools like 'search' if not needed
   ```

### Security

1. **Sandbox Commands**
   ```yaml
   tools:
     execute_command:
       type: command
       allowed_commands: ["ls", "cat", "grep"]  # Whitelist only
       enable_sandboxing: true
       max_execution_time: "30s"
   ```

2. **Environment Variables for Secrets**
   ```yaml
   llms:
     main:
       api_key: "${ANTHROPIC_API_KEY}"  # Never hardcode
   ```

3. **File Path Restrictions**
   ```yaml
   document_stores:
     code:
       path: "./src"  # Limit to specific directories
       exclude_patterns:
         - "**/.env"
         - "**/secrets/**"
   ```

### Debugging

1. **Enable Debug Output**
   ```yaml
   reasoning:
     show_debug_info: true  # See iterations, tokens, reflections
   ```

2. **Test with Non-Streaming First**
   ```yaml
   reasoning:
     enable_streaming: false  # Easier to debug
   ```

3. **Trust the LLM to Terminate**
   ```yaml
   reasoning:
     # max_iterations: 100  # Default is fine - LLM stops naturally
   ```
   
   **Philosophy:** Like Cursor, Hector trusts the LLM to complete tasks without artificial limits.

---

## Example Configurations

### Minimal (Quick Start)

```yaml
agents:
  assistant:
    llm: "main"

llms:
  main:
    type: "anthropic"
    model: "claude-3-7-sonnet-latest"
    api_key: "${ANTHROPIC_API_KEY}"
```

### Recommended (Production)

See [hector.yaml](hector.yaml) for the default configuration.

### With Semantic Search

See examples in [configs/](configs/) directory.

---

## Environment Variables

Common environment variables:

```bash
# LLM API Keys
export ANTHROPIC_API_KEY="your-key"
export OPENAI_API_KEY="your-key"

# Optional: Database URLs
export QDRANT_URL="http://localhost:6334"
export OLLAMA_HOST="http://localhost:11434"
```

---

## Troubleshooting

### "No tools available"

**Problem:** `🔧 Available tools: 0`

**Solution:** Remove empty `tools:` section to use defaults, or explicitly define tools.

### "Rate limit exceeded"

**Problem:** API returns 429 errors

**Solution:** Hector automatically retries with exponential backoff (default: 5 attempts, up to 62s). If persistent:

**Increase retry aggressiveness:**
```yaml
llms:
  main-llm:
    max_retries: 7        # More attempts (2s, 4s, 8s, 16s, 32s, 64s, 128s)
    retry_delay: 3        # Longer waits (3s, 6s, 12s, 24s, 48s)
```

**Reduce request frequency:**
- Reduce `max_tokens`
- Decrease `max_history_messages`
- Disable `include_context`

### "Document store not found"

**Problem:** `failed to get database 'qdrant'`

**Solution:**
1. Ensure Qdrant is running: `docker ps | grep qdrant`
2. Check database configuration matches
3. Verify database name in agent config

### "Streaming not working"

**Problem:** No real-time output

**Solution:**
```yaml
reasoning:
  enable_streaming: true
  show_debug_info: true  # See if tools are executing
```

---

## Advanced Topics

### Custom Prompts for Specific Domains

**Legal Assistant:**
```yaml
agents:
  legal-assistant:
    prompt:
      prompt_slots:
        system_role: "You are a legal research assistant."
        reasoning_instructions: |
          Focus on accuracy and cite sources.
          Use formal legal language.
```

**Code Reviewer:**
```yaml
agents:
  reviewer:
    prompt:
      prompt_slots:
        system_role: "You are a code review expert."
        tool_usage: |
          Always search for similar patterns in the codebase.
          Use search tool to find related code.
```

### Multi-Configuration Setup

```bash
# Development
./hector --config config-dev.yaml

# Production
./hector --config config-prod.yaml

# Testing
./hector --config config-test.yaml
```

---

## References

- [README.md](README.md) - Main documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - System design
- [LICENSE.md](LICENSE.md) - Licensing information
- [Examples](configs/) - Sample configurations

---

**Last Updated:** October 4, 2025
