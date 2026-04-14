# Rate Limiting

Control API usage with flexible quotas. Hector supports multi-window rate limiting with token and request count tracking.

## Overview

Rate limiting protects:

- **Your budget** - Cap LLM token usage per user/session
- **Your infrastructure** - Prevent request floods
- **Fair usage** - Ensure equitable access in multi-tenant scenarios

## Configuration

Rate limiting is configured via **environment variables** (not YAML):

```bash
export HECTOR_RATE_LIMIT_ENABLED=true
export HECTOR_RATE_LIMIT_SCOPE=user
export HECTOR_RATE_LIMIT_BACKEND=sql
export HECTOR_RATE_LIMIT_LIMITS='[
  {"type": "token", "window": "day", "limit": 100000},
  {"type": "count", "window": "minute", "limit": 60}
]'
```

## Limit Types

| Type | Tracks | Use Case |
|------|--------|----------|
| `token` | LLM tokens consumed | Cost control, billing |
| `count` | Request count | DDoS protection, throttling |

## Time Windows

| Window | Duration | Use Case |
|--------|----------|----------|
| `minute` | 60 seconds | Burst protection |
| `hour` | 60 minutes | Short-term limits |
| `day` | 24 hours | Daily quotas |
| `week` | 7 days | Weekly budgets |
| `month` | 30 days | Billing cycles |

## Scopes

| Scope | Behavior |
|-------|----------|
| `session` | Each session has independent quotas |
| `user` | All sessions for a user share quotas |

## Storage Backends

| Backend | Config | Use Case |
|---------|--------|----------|
| `memory` | Default | Single instance, dev/test |
| `sql` | `HECTOR_RATE_LIMIT_BACKEND=sql` | Production, multi-instance |

> **Note:** Memory backend resets on restart. Use SQL for production.

## Configuration Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `HECTOR_RATE_LIMIT_ENABLED` | `false` | Enable rate limiting |
| `HECTOR_RATE_LIMIT_SCOPE` | `session` | `session` or `user` |
| `HECTOR_RATE_LIMIT_BACKEND` | `memory` | `memory` or `sql` |
| `HECTOR_RATE_LIMIT_LIMITS` | `[]` | JSON array of limit configs |
| `HECTOR_RATE_LIMIT_IP_HEADERS` | - | Headers for client IP (e.g., `CF-Connecting-IP`) |

### Limit Object Schema

```json
{
  "type": "token",      // or "count"
  "window": "day",      // minute, hour, day, week, month
  "limit": 100000       // maximum value
}
```

## Example: Production Setup

```bash
# Enable rate limiting with SQL backend
export HECTOR_RATE_LIMIT_ENABLED=true
export HECTOR_RATE_LIMIT_BACKEND=sql
export HECTOR_RATE_LIMIT_SCOPE=user

# Multi-layer limits
export HECTOR_RATE_LIMIT_LIMITS='[
  {"type": "count", "window": "minute", "limit": 60},
  {"type": "count", "window": "hour", "limit": 500},
  {"type": "token", "window": "day", "limit": 500000},
  {"type": "token", "window": "month", "limit": 5000000}
]'

# Cloudflare IP detection
export HECTOR_RATE_LIMIT_IP_HEADERS="CF-Connecting-IP,X-Forwarded-For"
```

## API Response

When a limit is exceeded, clients receive:

```
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
Retry-After: 45

{
  "error": "rate limit exceeded",
  "limit_type": "token",
  "window": "day",
  "limit": 100000,
  "used": 100000,
  "reset_at": "2026-01-21T00:00:00Z"
}
```

## Next Steps

*   [Security Best Practices](./security.md)
*   [Operations Guide](./operations.md)
*   [CLI Reference](../reference/cli.md)
