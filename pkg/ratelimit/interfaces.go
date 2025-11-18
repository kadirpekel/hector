package ratelimit

import (
	"context"
	"time"
)

// Scope represents the scope of rate limiting
type Scope string

const (
	ScopeSession Scope = "session" // Rate limit per session
	ScopeUser    Scope = "user"    // Rate limit per user (across sessions)
)

// RateLimiter is the main interface for rate limiting
type RateLimiter interface {
	// Check verifies if the operation is allowed without recording usage
	Check(ctx context.Context, scope Scope, identifier string) (*CheckResult, error)

	// Record records actual usage (tokens and/or count)
	Record(ctx context.Context, scope Scope, identifier string, tokenCount int64, requestCount int64) error

	// CheckAndRecord checks limits and records usage in a single operation (atomic)
	CheckAndRecord(ctx context.Context, scope Scope, identifier string, tokenCount int64, requestCount int64) (*CheckResult, error)

	// GetUsage returns current usage statistics for an identifier
	GetUsage(ctx context.Context, scope Scope, identifier string) ([]Usage, error)

	// Reset resets usage for an identifier (useful for testing or manual resets)
	Reset(ctx context.Context, scope Scope, identifier string) error

	// ResetExpired removes expired usage records (for cleanup)
	ResetExpired(ctx context.Context, before time.Time) error
}

// Store is the persistence layer for rate limit data
type Store interface {
	// GetUsage gets current usage for a specific limit
	GetUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow) (int64, time.Time, error)

	// IncrementUsage increments usage for a specific limit
	IncrementUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow, amount int64) (int64, time.Time, error)

	// SetUsage sets usage for a specific limit
	SetUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow, amount int64, windowEnd time.Time) error

	// DeleteUsage deletes usage records for an identifier
	DeleteUsage(ctx context.Context, scope Scope, identifier string) error

	// DeleteExpired deletes expired usage records
	DeleteExpired(ctx context.Context, before time.Time) error

	// Close closes the store
	Close() error
}

