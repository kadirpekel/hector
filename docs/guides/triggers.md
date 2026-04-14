# Triggers & Automations

Triggers enable agents to run automatically, on a schedule or in response to external events, without manual invocation.

## Trigger Types

| Type | Use Case |
|------|----------|
| `schedule` | Cron-based recurring tasks (reports, syncs, health checks) |
| `webhook` | External event handling (GitHub, Slack, custom systems) |

## Scheduled Triggers

Run agents on a cron schedule with timezone support.

```yaml
agents:
  daily_reporter:
    llm: claude
    instruction: "Generate a daily summary report."
    trigger:
      type: schedule
      cron: "0 9 * * *"        # 9 AM daily
      timezone: America/New_York
      input: "Generate report for yesterday."
```

### Cron Syntax

Standard 5-field cron expressions:

```
┌───────────── minute (0-59)
│ ┌─────────── hour (0-23)
│ │ ┌───────── day of month (1-31)
│ │ │ ┌─────── month (1-12)
│ │ │ │ ┌───── day of week (0-6, Sunday=0)
│ │ │ │ │
* * * * *
```

**Examples:**

| Expression | Description |
|------------|-------------|
| `0 9 * * 1-5` | 9 AM weekdays |
| `*/15 * * * *` | Every 15 minutes |
| `0 0 1 * *` | First of each month |

### Durable Execution

Scheduled tasks are enqueued to the task queue, not fired in the scheduler goroutine. This means:

- ✅ Automatic retry on failure
- ✅ Persistence across restarts
- ✅ Observable via task APIs

## Webhook Triggers

Invoke agents from external systems via HTTP.

```yaml
agents:
  github_handler:
    llm: claude
    instruction: "Respond to GitHub events."
    trigger:
      type: webhook
      path: /webhooks/github
      methods: [POST]
      secret: ${GITHUB_WEBHOOK_SECRET}
      signature_header: X-Hub-Signature-256
      webhook_input:
        template: "Event: {{.action}} on {{.repository.full_name}}"
```

### Request Authentication

HMAC signature verification protects against spoofed requests:

```yaml
trigger:
  type: webhook
  secret: ${WEBHOOK_SECRET}           # HMAC key
  signature_header: X-Hub-Signature   # Header containing signature
```

### Payload Transformation

Transform incoming JSON to agent input using Go templates:

```yaml
webhook_input:
  template: |
    New issue #{{.issue.number}}: {{.issue.title}}
    Body: {{.issue.body}}
  session_id: "issue-{{.issue.number}}"  # Derive session for continuity
  extract_fields:
    - path: issue.user.login
      as: author
```

### Response Modes

| Mode | Behavior |
|------|----------|
| `sync` | Wait for agent completion, return result |
| `async` | Return task ID immediately for polling |
| `callback` | Return immediately, POST result to callback URL |

```yaml
trigger:
  response:
    mode: async
    # or
    mode: callback
    callback_url: https://my-service.com/results
```

## Configuration Reference

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `type` | string | - | `schedule` or `webhook` |
| `enabled` | boolean | `true` | Toggle trigger |
| `cron` | string | - | Cron expression |
| `timezone` | string | `UTC` | IANA timezone |
| `input` | string | - | Static agent input |
| `path` | string | - | Webhook URL path |
| `methods` | []string | `[POST]` | Allowed HTTP methods |
| `secret` | string | - | HMAC verification secret |
| `signature_header` | string | `X-Webhook-Signature` | Header for signature |

## Next Steps

*   [View Trigger Configuration Reference](../reference/configuration.md#agents)
*   [Learn about Durable Execution](./operations.md)
*   [Configure Notifications](./agents.md#notifications)
