# Rate Limiting Package

A flexible, multi-layer rate limiting system for Hector that supports both token-based and count-based limits across various time windows.

## Features

- ✅ **Multi-layer time windows**: minute, hour, day, week, month
- ✅ **Dual tracking**: Token count AND request count
- ✅ **Flexible scopes**: Per-session or per-user limiting
- ✅ **Multiple storage backends**: In-memory and SQL (Postgres, MySQL, SQLite)
- ✅ **Atomic check-and-record**: Race-condition safe
- ✅ **Detailed usage stats**: Real-time usage tracking and percentage calculations
- ✅ **Simple integration**: Decorator pattern for existing services

## Quick Start

### Basic Usage

```go
import "github.com/kadirpekel/hector/pkg/ratelimit"

// 1. Configure rate limits
config := ratelimit.RateLimitConfig{
    Enabled: true,
    Limits: []ratelimit.RateLimit{
        // 100k tokens per day
        {Type: ratelimit.LimitTypeToken, Window: ratelimit.WindowDay, Limit: 100000},
        // 60 requests per minute
        {Type: ratelimit.LimitTypeCount, Window: ratelimit.WindowMinute, Limit: 60},
    },
}

// 2. Create store (memory or SQL)
store := ratelimit.NewMemoryStore()
// OR for persistence:
// store, _ := ratelimit.NewSQLStore(db, "postgres")

// 3. Create limiter
limiter, err := ratelimit.NewRateLimiter(config, store)
if err != nil {
    log.Fatal(err)
}

// 4. Check and record usage
result, err := limiter.CheckAndRecord(
    context.Background(),
    ratelimit.ScopeSession, // or ScopeUser
    "session-123",
    1000, // token count
    1,    // request count
)

if !result.Allowed {
    log.Printf("Rate limit exceeded: %s", result.Reason)
    log.Printf("Retry after: %v", result.RetryAfter)
    return
}

// Request allowed - proceed
```

### Integration with SessionService

```go
import (
    "github.com/kadirpekel/hector/pkg/memory"
    "github.com/kadirpekel/hector/pkg/ratelimit"
)

// Create base session service
baseService := memory.NewInMemorySessionService()

// Configure rate limiting
config := ratelimit.RateLimitConfig{
    Enabled: true,
    Limits: []ratelimit.RateLimit{
        {Type: ratelimit.LimitTypeToken, Window: ratelimit.WindowDay, Limit: 50000},
        {Type: ratelimit.LimitTypeCount, Window: ratelimit.WindowMinute, Limit: 30},
    },
}

store := ratelimit.NewMemoryStore()
limiter, _ := ratelimit.NewRateLimiter(config, store)

// Wrap with rate limiting
sessionService := ratelimit.NewRateLimitedSessionService(
    baseService,
    limiter,
    ratelimit.ScopeSession, // Rate limit per session
)

// Use normally - rate limiting is automatic
err := sessionService.AppendMessage(sessionID, message)
if ratelimit.IsRateLimitError(err) {
    result := ratelimit.GetRateLimitResult(err)
    fmt.Printf("Rate limited: %s\n", result.Reason)
    // Handle rate limit...
}
```

## Configuration Examples

### Conservative Daily Limits

```yaml
rate_limiting:
  enabled: true
  limits:
    - type: token
      window: day
      limit: 50000
    - type: count
      window: day
      limit: 1000
```

### Multi-layer Protection

```yaml
rate_limiting:
  enabled: true
  limits:
    # Per-minute burst protection
    - type: count
      window: minute
      limit: 60
    # Hourly token budget
    - type: token
      window: hour
      limit: 10000
    # Daily token quota
    - type: token
      window: day
      limit: 100000
    # Weekly total cap
    - type: token
      window: week
      limit: 500000
```

### High-throughput API

```yaml
rate_limiting:
  enabled: true
  limits:
    - type: count
      window: minute
      limit: 1000
    - type: token
      window: hour
      limit: 500000
```

## Time Windows

| Window | Duration | Use Case |
|--------|----------|----------|
| `minute` | 60 seconds | Burst protection, API throttling |
| `hour` | 60 minutes | Short-term usage control |
| `day` | 24 hours | Daily quotas, typical usage limits |
| `week` | 7 days | Weekly budgets |
| `month` | 30 days | Monthly billing cycles |

## Limit Types

### Token Limits

Track **token usage** (e.g., LLM API tokens). Useful for:
- Cost control
- LLM API quota management
- Proportional resource usage

```go
{Type: ratelimit.LimitTypeToken, Window: ratelimit.WindowDay, Limit: 100000}
```

### Count Limits

Track **request count** regardless of size. Useful for:
- Rate throttling
- DDoS protection
- Fair usage across users

```go
{Type: ratelimit.LimitTypeCount, Window: ratelimit.WindowMinute, Limit: 60}
```

## Scopes

### Session Scope

Each session has its own independent quota.

```go
limiter.CheckAndRecord(ctx, ratelimit.ScopeSession, "session-123", tokens, 1)
```

**Use when**: You want to limit per conversation or session.

### User Scope

All sessions for a user share the same quota.

```go
limiter.CheckAndRecord(ctx, ratelimit.ScopeUser, "user-456", tokens, 1)
```

**Use when**: You want to limit per user across all their sessions.

## Storage Backends

### Memory Store

Fast, in-memory storage. Data lost on restart.

```go
store := ratelimit.NewMemoryStore()
```

**Best for**: Development, testing, single-instance deployments

### SQL Store

Persistent storage with support for Postgres, MySQL, SQLite.

```go
store, err := ratelimit.NewSQLStore(db, "postgres")
```

**Best for**: Production, multi-instance deployments, persistent quotas

## API Reference

### RateLimiter Interface

```go
type RateLimiter interface {
    // Check if operation is allowed (no recording)
    Check(ctx, scope, identifier) (*CheckResult, error)
    
    // Record usage after operation
    Record(ctx, scope, identifier, tokens, count) error
    
    // Atomic check and record
    CheckAndRecord(ctx, scope, identifier, tokens, count) (*CheckResult, error)
    
    // Get current usage statistics
    GetUsage(ctx, scope, identifier) ([]Usage, error)
    
    // Reset usage for identifier
    Reset(ctx, scope, identifier) error
    
    // Clean up expired records
    ResetExpired(ctx, before) error
}
```

### CheckResult

```go
type CheckResult struct {
    Allowed    bool            // Whether request is allowed
    Reason     string          // Reason if denied
    Usages     []Usage         // Current usage for all limits
    RetryAfter *time.Duration  // When to retry if denied
}
```

### Usage

```go
type Usage struct {
    LimitType  LimitType   // "token" or "count"
    Window     TimeWindow  // Time window
    Current    int64       // Current usage
    Limit      int64       // Maximum allowed
    WindowEnd  time.Time   // When window resets
    Remaining  int64       // Remaining quota
    Percentage float64     // Usage percentage
}
```

## Error Handling

```go
err := sessionService.AppendMessage(sessionID, message)

if ratelimit.IsRateLimitError(err) {
    result := ratelimit.GetRateLimitResult(err)
    
    // Log details
    log.Printf("Rate limited: %s", result.Reason)
    
    // Show usage to user
    for _, usage := range result.Usages {
        fmt.Printf("%s/%s: %d/%d (%.1f%%)\n",
            usage.LimitType, usage.Window,
            usage.Current, usage.Limit,
            usage.Percentage)
    }
    
    // Inform when they can retry
    if result.RetryAfter != nil {
        fmt.Printf("Retry after: %v\n", *result.RetryAfter)
    }
    
    return
}
```

## Best Practices

1. **Combine token and count limits** for comprehensive protection
2. **Use multiple time windows** (minute + day) to prevent both bursts and sustained abuse
3. **Choose appropriate scope** (session vs user) based on your use case
4. **Monitor usage** regularly to tune limits
5. **Provide clear feedback** to users when rate limited
6. **Use SQL store** in production for persistence
7. **Clean up expired records** periodically with `ResetExpired()`

## Example Configurations

### Developer Tier

```go
Limits: []ratelimit.RateLimit{
    {Type: LimitTypeToken, Window: WindowDay, Limit: 10000},
    {Type: LimitTypeCount, Window: WindowMinute, Limit: 10},
}
```

### Pro Tier

```go
Limits: []ratelimit.RateLimit{
    {Type: LimitTypeToken, Window: WindowDay, Limit: 100000},
    {Type: LimitTypeCount, Window: WindowMinute, Limit: 60},
}
```

### Enterprise Tier

```go
Limits: []ratelimit.RateLimit{
    {Type: LimitTypeToken, Window: WindowDay, Limit: 1000000},
    {Type: LimitTypeCount, Window: WindowMinute, Limit: 1000},
}
```

## Testing

```bash
cd pkg/ratelimit
go test -v
```

## Architecture

```
┌─────────────────────────────────────────┐
│   RateLimitedSessionService             │
│   (Decorator Pattern)                   │
└────────────┬────────────────────────────┘
             │ uses
             ▼
┌─────────────────────────────────────────┐
│   RateLimiter                           │
│   (Core Logic)                          │
│   - Check limits                        │
│   - Record usage                        │
│   - Calculate windows                   │
└────────────┬────────────────────────────┘
             │ uses
             ▼
┌─────────────────────────────────────────┐
│   Store (Interface)                     │
│   ├─ MemoryStore (dev/testing)         │
│   └─ SQLStore (production)             │
└─────────────────────────────────────────┘
```

## Performance

- **Check operation**: O(n) where n = number of limits (typically 2-5)
- **Record operation**: O(n) where n = number of limits
- **Memory store**: O(1) lookup and update
- **SQL store**: O(1) with proper indexing

## Thread Safety

All components are thread-safe and can be used concurrently across multiple goroutines.

## License

See main Hector license.




