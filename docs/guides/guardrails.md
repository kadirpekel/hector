# Guardrails & Safety

Guardrails provide a safety layer around your agents, ensuring they behave within defined boundaries. They operate on **inputs** (user messages), **outputs** (agent responses), **tools** (execution authorization), and optionally through **LLM-powered moderation**.

## Quick Start

Enable basic protection with a few lines:

```yaml
guardrails:
  default:
    enabled: true
    input:
      injection: { enabled: true }
      sanitizer: { enabled: true, trim_whitespace: true }
    output:
      pii: { enabled: true, redact_mode: mask }

agents:
  assistant:
    guardrails: default
```

This blocks prompt injection attempts, sanitizes input, and masks PII in output.

---

## How Guardrails Work

Guardrails run as a **chain of responsibility**. Each check runs in sequence and can:

| Action | Behavior |
|--------|----------|
| `allow` | Continue normally |
| `block` | Stop execution, return error |
| `modify` | Continue with transformed content |
| `warn` | Log warning, continue |

Each violation has a **severity level**: `low`, `medium`, `high`, or `critical`.

### Chain Modes

```yaml
guardrails:
  strict:
    input:
      chain_mode: fail_fast    # Stop on first violation (default)
    output:
      chain_mode: collect_all  # Run all checks, return consolidated report
```

- **`fail_fast`** (default): Maximum security. Rejects immediately on first failure
- **`collect_all`**: Runs every check and returns all violations together, useful for debugging or user feedback

---

## Input Guardrails

Input guardrails protect your agent from malicious or malformed user messages.

### Prompt Injection Detection

Detects attempts to override system instructions. Hector includes built-in detection patterns for common attack vectors:

- Override attempts: "ignore previous instructions", "disregard all rules"
- Role manipulation: "you are now", "pretend to be", "act as"
- System impersonation: messages starting with `system:`, `[system]`, `<system>`
- Jailbreak patterns: "jailbreak", "DAN mode", "developer mode"
- Encoded content: `base64:` prefixed payloads

```yaml
input:
  injection:
    enabled: true
    action: block          # block | warn (default: block)
    severity: high         # default: high
    case_sensitive: false  # default: false
    patterns:              # Add custom patterns (regex)
      - "bypass safety"
      - "reveal your prompt"
      - "(?i)ignore.*instructions"
```

!!! warning "Custom Patterns Are Additive"
    Custom patterns are added to the built-in set, not replacing them. The built-in patterns always run when injection detection is enabled.

### Input Sanitization

Cleans inputs before they reach the agent:

```yaml
input:
  sanitizer:
    enabled: true
    trim_whitespace: true      # Remove leading/trailing whitespace (default: true)
    normalize_unicode: false   # Normalize Unicode characters
    max_length: 50000          # Truncate (0 = no limit)
    strip_html: true           # Remove HTML tags
```

### Length Validation

Reject inputs that are too short or too long:

```yaml
input:
  length:
    enabled: true
    min_length: 1         # default: 1
    max_length: 100000    # default: 100000
    action: block
    severity: medium
```

### Pattern Validation

Block or allow inputs matching specific patterns:

```yaml
input:
  pattern:
    enabled: true
    block_patterns:
      - "sudo.*"
      - "rm -rf"
      - "DROP TABLE"
    allow_patterns:
      - ".*"  # Allow everything else
    action: block
    severity: high
```

---

## Output Guardrails

Output guardrails protect your users and organization from harmful agent responses.

### PII Redaction

Detects and redacts personally identifiable information before it reaches the user.

```yaml
output:
  pii:
    enabled: true
    detect_email: true        # Email addresses (default: true)
    detect_phone: true        # Phone numbers (US format, +1 variants)
    detect_ssn: true          # Social Security Numbers (XXX-XX-XXXX)
    detect_credit_card: true  # Visa, Mastercard, Amex, Discover
    redact_mode: mask         # mask | remove | hash
    action: modify            # default: modify
    severity: high
```

**Redaction modes:**

| Mode | Input | Output |
|------|-------|--------|
| `mask` | `john@example.com` | `[EMAIL REDACTED]` |
| `remove` | `john@example.com` | *(removed entirely)* |
| `hash` | `john@example.com` | `[EMAIL:a1b2c3d4]` |

!!! tip "Use `hash` for Debugging"
    Hash mode lets you correlate redacted values across a conversation without exposing the actual data. Useful for debugging PII leaks.

### Content Filtering

Block responses containing specific keywords or patterns:

```yaml
output:
  content:
    enabled: true
    blocked_keywords:           # Case-insensitive exact match
      - "internal_api_key"
      - "confidential"
    blocked_patterns:           # Regex patterns
      - "(?i)password.*=.*"
      - "sk-[a-zA-Z0-9]{20,}"  # Block API key patterns
    action: block
    severity: critical
```

---

## Tool Authorization

Control which tools agents can invoke, using glob patterns for flexible matching:

```yaml
guardrails:
  restricted:
    tool:
      authorization:
        enabled: true
        allowed_tools:
          - "search*"       # All search tools
          - "read_*"        # All read operations
          - "web_fetch"     # Specific tool
        blocked_tools:
          - "delete_*"      # Block all delete operations
          - "*_production"  # Block anything ending in _production
        action: block
        severity: high
```

!!! note "Layered Tool Security"
    Tool authorization works alongside other tool security measures. You can combine: (1) Per-agent tool assignment, (2) MCP tool filtering, (3) Human-in-the-loop approval, (4) Guardrail-based authorization. See the [Security Guide](./security.md) for the full picture.

---

## LLM-Powered Moderation

For semantic safety that pattern matching can't catch, use LLM-based moderation. Hector supports multiple providers:

### OpenAI Moderation

Free to use with any OpenAI API key:

```yaml
guardrails:
  moderated:
    moderation:
      enabled: true
      strategy: openai
      action: block          # block | warn
      openai:
        model: omni-moderation-latest   # or text-moderation-latest
        threshold: 0.8                   # 0-1 confidence threshold
```

### Lakera Guard

Specialized in prompt injection and jailbreak detection:

```yaml
guardrails:
  moderated:
    moderation:
      enabled: true
      strategy: lakera
      action: block
      lakera:
        project_id: "your-project-id"   # Optional, uses default policy if empty
        breakdown: true                   # Return detailed detector breakdown
```

!!! tip "Choosing a Moderation Provider"
    - **OpenAI**: Good general-purpose content moderation (hate, violence, self-harm). Free API.
    - **Lakera**: Best for prompt injection and jailbreak detection. Paid but specialized.
    - **Prompt-based**: Use your own LLM to evaluate content. Most flexible, highest latency.

### Prompt-Based Moderation

Use a custom LLM prompt for domain-specific moderation:

```yaml
guardrails:
  custom_moderation:
    moderation:
      enabled: true
      strategy: prompt
      prompt:
        llm: claude                      # LLM to use (falls back to agent's LLM)
        template: |
          Evaluate if this user message is safe and appropriate
          for a customer support context. Respond with JSON:
          {"safe": true/false, "reason": "..."}
          
          Message: {input}
        safe_field: "safe"               # JSON field to check (default: "safe")
      action: block
```

---

## Complete Example

A production-ready guardrail configuration combining all layers:

```yaml
guardrails:
  production:
    enabled: true
    
    input:
      chain_mode: fail_fast
      length:
        enabled: true
        max_length: 50000
        action: block
        severity: medium
      injection:
        enabled: true
        patterns:
          - "reveal.*system.*prompt"
          - "output.*instructions"
        action: block
        severity: critical
      sanitizer:
        enabled: true
        trim_whitespace: true
        strip_html: true
        max_length: 50000
      pattern:
        enabled: true
        block_patterns:
          - "(?i)DROP TABLE"
          - "(?i)DELETE FROM"
    
    output:
      chain_mode: collect_all
      pii:
        enabled: true
        detect_email: true
        detect_phone: true
        detect_ssn: true
        detect_credit_card: true
        redact_mode: mask
        action: modify
        severity: high
      content:
        enabled: true
        blocked_patterns:
          - "sk-[a-zA-Z0-9]{20,}"    # API keys
          - "(?i)internal use only"
        action: block
        severity: critical
    
    tool:
      authorization:
        enabled: true
        allowed_tools: ["search*", "read_*", "web_fetch"]
        blocked_tools: ["delete_*", "command_*"]
    
    moderation:
      enabled: true
      strategy: openai
      openai:
        model: omni-moderation-latest
        threshold: 0.8
      action: block

agents:
  support_bot:
    guardrails: production
```

---

## Observability

Guardrail interventions are tracked in event metadata, making them visible in logs, traces, and Studio:

- **Intervention source**: `input_guardrail`, `output_guardrail`, `tool_guardrail`, `moderation`
- **Details**: Matched pattern, action taken, severity
- **Metrics**: Prometheus counters for guardrail triggers (by type, action, severity)

This means you can build dashboards and alerts for guardrail activity. See the [Observability Guide](./observability.md).

## Next Steps

- [Security Guide](./security.md): Full security model including auth, sandboxing, and tool permissions
- [Tools Guide](./tools.md): Tool-level security controls
- [Guardrails Reference](../reference/configuration.md#guardrails): Complete YAML schema
