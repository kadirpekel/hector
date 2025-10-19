---
title: LLM Providers
description: Configure OpenAI, Anthropic, Gemini, and custom LLM providers
---

# LLM Providers

Hector supports multiple LLM providers out of the box. Each agent references an LLM configuration that defines which model to use and how to connect to it.

## Supported Providers

| Provider | Models | Streaming | Structured Output |
|----------|--------|-----------|-------------------|
| **OpenAI** | GPT-4o, GPT-4o-mini, GPT-4 Turbo, etc. | ✅ | ✅ |
| **Anthropic** | Claude Sonnet 4, Claude Opus 4, etc. | ✅ | ✅ |
| **Google Gemini** | Gemini 2.0 Flash, Gemini Pro, etc. | ✅ | ✅ |
| **Custom (Plugin)** | Any model via gRPC plugin | ✅ | ✅ |

---

## Configuration Pattern

LLM providers are configured separately from agents:

```yaml
# Define LLM providers
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
  
  claude:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"

# Agents reference them
agents:
  assistant:
    llm: "gpt-4o"  # References the "gpt-4o" config
  
  researcher:
    llm: "claude"  # References the "claude" config
```

This allows multiple agents to share the same LLM configuration.

---

## OpenAI

### Configuration

```yaml
llms:
  my-openai:
    type: "openai"
    model: "gpt-4o"                    # Default: gpt-4o
    api_key: "${OPENAI_API_KEY}"
    host: "https://api.openai.com/v1" # Default host
    temperature: 0.7                   # Default: 0.7
    max_tokens: 8000                   # Default: 8000
    timeout: 60                        # Seconds, default: 60
    max_retries: 5                     # Rate limit retries, default: 5
    retry_delay: 2                     # Seconds, exponential backoff, default: 2
```

### Popular Models

| Model | Best For | Context Window |
|-------|----------|----------------|
| `gpt-4o` | General purpose, balanced | 128K tokens |
| `gpt-4o-mini` | Fast, cost-effective | 128K tokens |
| `gpt-4-turbo` | Complex reasoning | 128K tokens |
| `gpt-3.5-turbo` | Simple tasks, fast | 16K tokens |

### Environment Variables

```bash
export OPENAI_API_KEY="sk-..."
```

### Example

```yaml
llms:
  gpt-4o:
    type: "openai"
    model: "gpt-4o-mini"
    api_key: "${OPENAI_API_KEY}"
    temperature: 0.7

agents:
  coder:
    name: "Coding Assistant"
    llm: "gpt-4o"
    prompt:
      system_role: "You are an expert programmer."
```

---

## Anthropic (Claude)

### Configuration

```yaml
llms:
  my-anthropic:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"  # Default model
    api_key: "${ANTHROPIC_API_KEY}"
    host: "https://api.anthropic.com"  # Default host
    temperature: 0.7                   # Default: 0.7
    max_tokens: 8000                   # Default: 8000
    timeout: 120                       # Seconds, default: 120
    max_retries: 5                     # Default: 5
    retry_delay: 2                     # Seconds, default: 2
```

### Popular Models

| Model | Best For | Context Window |
|-------|----------|----------------|
| `claude-sonnet-4-20250514` | Balanced speed & capability | 200K tokens |
| `claude-opus-4-20250514` | Maximum capability | 200K tokens |
| `claude-3-5-sonnet-20241022` | Previous generation | 200K tokens |

### Environment Variables

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

### Example

```yaml
llms:
  claude:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"
    temperature: 0.7

agents:
  analyst:
    name: "Research Analyst"
    llm: "claude"
    prompt:
      system_role: "You are a thorough research analyst."
```

---

## Google Gemini

### Configuration

```yaml
llms:
  my-gemini:
    type: "gemini"
    model: "gemini-2.0-flash-exp"         # Default model
    api_key: "${GEMINI_API_KEY}"
    host: "https://generativelanguage.googleapis.com"  # Default
    temperature: 0.7                      # Default: 0.7
    max_tokens: 4096                      # Default: 4096
    timeout: 60                           # Seconds, default: 60
```

### Popular Models

| Model | Best For | Context Window |
|-------|----------|----------------|
| `gemini-2.0-flash-exp` | Fast, efficient (experimental) | 1M tokens |
| `gemini-pro` | General purpose | 1M tokens |
| `gemini-pro-vision` | Image analysis | 16K tokens |

### Environment Variables

```bash
export GEMINI_API_KEY="AI..."
```

### Example

```yaml
llms:
  gemini:
    type: "gemini"
    model: "gemini-2.0-flash-exp"
    api_key: "${GEMINI_API_KEY}"
    temperature: 0.7

agents:
  assistant:
    name: "General Assistant"
    llm: "gemini"
    prompt:
      system_role: "You are a helpful assistant."
```

---

## Custom LLM Providers (Plugins)

Extend Hector with custom LLM providers via gRPC plugins.

### Configuration

```yaml
plugins:
  llms:
    - name: "my-custom-llm"
      protocol: "grpc"
      path: "/path/to/llm-plugin"

llms:
  custom:
    type: "plugin:my-custom-llm"
    model: "my-model"
    # Provider-specific configuration
```

See [Plugin System](../reference/architecture.md#plugin-system) for implementation details.

---

## Common Configuration Options

### Temperature

Controls randomness in responses (0.0 to 2.0):

```yaml
llms:
  creative:
    type: "openai"
    temperature: 1.2  # More creative

  precise:
    type: "openai"
    temperature: 0.3  # More deterministic
```

- **0.0-0.3**: Focused, deterministic (code generation, analysis)
- **0.7-0.9**: Balanced (default for most tasks)
- **1.0-2.0**: Creative (writing, brainstorming)

### Max Tokens

Maximum tokens in the response:

```yaml
llms:
  brief:
    type: "openai"
    max_tokens: 500   # Short responses

  detailed:
    type: "openai"
    max_tokens: 4000  # Long responses
```

### Timeouts and Retries

Configure resilience:

```yaml
llms:
  resilient:
    type: "openai"
    timeout: 120        # Wait up to 2 minutes
    max_retries: 5      # Retry 5 times on rate limits
    retry_delay: 2      # Start with 2s, exponential backoff
```

### Custom API Endpoints

Use compatible APIs (Azure OpenAI, local models, etc.):

```yaml
llms:
  azure-openai:
    type: "openai"
    model: "gpt-4"
    api_key: "${AZURE_API_KEY}"
    host: "https://your-resource.openai.azure.com/openai/deployments/your-deployment"
```

---

## Zero-Config Defaults

When running without configuration, Hector uses these defaults:

| Provider | Model | Trigger |
|----------|-------|---------|
| OpenAI | `gpt-4o-mini` | `OPENAI_API_KEY` set |
| Anthropic | `claude-sonnet-4-20250514` | `ANTHROPIC_API_KEY` set |
| Gemini | `gemini-2.0-flash-exp` | `GEMINI_API_KEY` set |

Priority order: OpenAI → Anthropic → Gemini (first available key wins).

```bash
# Zero-config with OpenAI
export OPENAI_API_KEY="sk-..."
hector call "Hello"  # Uses gpt-4o-mini automatically
```

---

## Structured Output

All providers support structured output (JSON, XML, Enum):

```yaml
llms:
  structured:
    type: "openai"
    model: "gpt-4o"
    structured_output:
      format: "json"
      schema:
        type: "object"
        properties:
          sentiment:
            type: "string"
            enum: ["positive", "negative", "neutral"]
          confidence:
            type: "number"
```

See [Structured Output](structured-output.md) for details.

---

## Choosing a Provider

| Scenario | Recommended Provider | Reason |
|----------|---------------------|--------|
| General purpose | OpenAI GPT-4o | Best balance of speed, cost, capability |
| Complex reasoning | Anthropic Claude Opus 4 | Strongest reasoning capabilities |
| Cost-sensitive | OpenAI GPT-4o-mini | Excellent price/performance |
| Fast responses | Gemini 2.0 Flash | Very fast inference |
| Large context | Gemini Pro | 1M token context window |
| Code generation | OpenAI GPT-4o | Strong code understanding |
| Creative writing | Anthropic Claude Sonnet 4 | Natural, engaging writing |
| Production reliability | OpenAI GPT-4o | Mature API, good availability |

---

## Best Practices

### Use Environment Variables for API Keys

Never hardcode API keys:

```yaml
# ✅ Good
llms:
  gpt-4o:
    api_key: "${OPENAI_API_KEY}"

# ❌ Bad
llms:
  gpt-4o:
    api_key: "sk-hardcoded-key"
```

### Configure Timeouts Appropriately

Match timeouts to task complexity:

```yaml
llms:
  quick-tasks:
    type: "openai"
    timeout: 30  # Simple queries

  complex-analysis:
    type: "anthropic"
    timeout: 180  # Complex reasoning
```

### Use Different Providers for Different Agents

Leverage each provider's strengths:

```yaml
agents:
  coder:
    llm: "gpt-4o"  # Good at code

  writer:
    llm: "claude"  # Good at prose

  analyzer:
    llm: "claude-opus"  # Strong reasoning
```

### Enable Retries for Production

Ensure reliability:

```yaml
llms:
  production:
    type: "openai"
    max_retries: 5
    retry_delay: 2
    timeout: 120
```

---

## Next Steps

- **[Prompts](prompts.md)** - Customize agent behavior and instructions
- **[Memory](memory.md)** - Manage conversation context
- **[Structured Output](structured-output.md)** - Get JSON/XML responses
- **[Configuration Reference](../reference/configuration.md)** - All LLM options

---

## Related Topics

- **[Quick Start](../getting-started/quick-start.md)** - Run your first agent
- **[Build a Coding Assistant](../how-to/build-coding-assistant.md)** - Complete tutorial
- **[API Reference](../reference/api.md)** - API details

