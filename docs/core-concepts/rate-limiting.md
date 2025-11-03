# Rate Limiting

Control API usage and costs with flexible, multi-layer rate limiting.

---

## Overview

Rate limiting protects your system from:
- **Cost overruns** - Limit token usage for LLM APIs
- **Abuse** - Prevent spam and excessive requests
- **Resource exhaustion** - Control load on your system

Hector's rate limiting supports:
- ✅ Multi-layer time windows (minute → month)
- ✅ Dual tracking (tokens + request counts)
- ✅ Per-session or per-user scoping
- ✅ SQL or memory storage

---

## Quick Start

Add rate limiting to any session store:

```yaml
session_stores:
  default:
    backend: sql
    sql:
      driver: sqlite
      database: ./sessions.db
    rate_limit:
      enabled: true
      scope: session
      backend: memory
      limits:
        - type: count
          window: minute
          limit: 60
        - type: token
          window: day
          limit: 100000
```

---

## Configuration

### Basic Structure

```yaml
rate_limit:
  enabled: true          # Enable rate limiting
  scope: session         # "session" or "user"
  backend: memory        # "memory" or "sql"
  limits:                # List of limits to enforce
    - type: count
      window: minute
      limit: 60
```

### Limit Types

**`count`** - Request/message count
```yaml
- type: count
  window: minute
  limit: 60           # Max 60 requests per minute
```

**`token`** - LLM token usage
```yaml
- type: token
  window: day
  limit: 100000      # Max 100k tokens per day
```

### Time Windows

| Window | Duration | Use Case |
|--------|----------|----------|
| `minute` | 60 sec | Burst protection |
| `hour` | 60 min | Short-term throttling |
| `day` | 24 hours | Daily quotas |
| `week` | 7 days | Weekly budgets |
| `month` | 30 days | Monthly billing |

### Scopes

**Session Scope** - Each session independent
```yaml
scope: session
```
- Separate quota per conversation
- Best for: Per-conversation limits

**User Scope** - All sessions share quota
```yaml
scope: user
```
- Shared quota across all sessions
- Best for: Per-account limits

### Storage Backends

**Memory Backend** - Fast, volatile
```yaml
backend: memory
```
- ✅ Fast (O(1) lookups)
- ❌ Data lost on restart
- Best for: Development, single-instance

**SQL Backend** - Persistent
```yaml
backend: sql
```
- ✅ Survives restarts
- ✅ Works across distributed instances
- Best for: Production

---

## Common Patterns

### Pattern 1: Spam Prevention

```yaml
rate_limit:
  enabled: true
  scope: session
  limits:
    - type: count
      window: minute
      limit: 10
```

### Pattern 2: Cost Control

```yaml
rate_limit:
  enabled: true
  scope: user
  limits:
    - type: token
      window: day
      limit: 50000
    - type: token
      window: month
      limit: 1000000
```

### Pattern 3: Multi-Layer Protection

```yaml
rate_limit:
  enabled: true
  scope: user
  limits:
    - type: count
      window: minute
      limit: 60
    - type: count
      window: hour
      limit: 1000
    - type: token
      window: day
      limit: 100000
```

### Pattern 4: Tiered Limits

```yaml
session_stores:
  free-tier:
    rate_limit:
      enabled: true
      scope: user
      limits:
        - {type: count, window: minute, limit: 10}
        - {type: token, window: day, limit: 10000}
  
  pro-tier:
    rate_limit:
      enabled: true
      scope: user
      limits:
        - {type: count, window: minute, limit: 100}
        - {type: token, window: day, limit: 500000}
```

---

## Error Handling

When a rate limit is exceeded, the API returns:

```json
{
  "error": "rate limit exceeded",
  "details": {
    "limit_type": "count",
    "window": "minute",
    "current": 60,
    "limit": 60,
    "retry_after": "45s"
  }
}
```

---

## How It Works

### Request Flow

```
1. User sends message
   ↓
2. Check rate limits
   ↓
3a. Within limits → Process message
3b. Exceeded → Return 429 error
```

### Token Tracking

Tokens are counted after LLM response:

```
1. Message processed
   ↓
2. LLM returns response (1,234 tokens)
   ↓
3. Record token usage
   ↓
4. Check against limits
```

### Window Management

Windows are **sliding**, not fixed:

```
Limit: 60 requests/minute

12:00:00 → 12:00:59 : First window
12:00:30 → 12:01:29 : Sliding window
12:01:00 → 12:01:59 : New window
```

---

## Performance

### Memory Backend
- Lookup: O(1)
- Storage: In-memory map
- Capacity: ~10k req/sec

### SQL Backend
- Lookup: O(1) with indexes
- Storage: Dedicated `rate_limits` table
- Capacity: ~1k req/sec (depends on database)

---

## Best Practices

1. **Start conservative** - Begin with lower limits
2. **Monitor usage** - Track actual patterns
3. **Use multiple windows** - Combine minute + day limits
4. **Choose right scope**:
   - `session` for spam prevention
   - `user` for cost control
5. **Use SQL in production** - For persistence
6. **Provide clear feedback** - Show retry-after time

---

## Architecture

```
SessionService
    ↓ wraps
RateLimitedSessionService
    ↓ checks
RateLimiter
    ↓ queries
Store (Memory or SQL)
    ↓ persists
rate_limits table
```

### SQL Schema

```sql
CREATE TABLE rate_limits (
    scope VARCHAR(50) NOT NULL,
    identifier VARCHAR(255) NOT NULL,
    limit_type VARCHAR(50) NOT NULL,
    window VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL,
    window_end TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (scope, identifier, limit_type, window)
);
```

---

## See Also

- [Configuration Reference](../reference/configuration.md#rate-limiting)
- [Sessions](sessions.md)
- [Session Persistence](../how-to/setup-session-persistence.md)

