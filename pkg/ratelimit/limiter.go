package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
)

// DefaultRateLimiter implements RateLimiter interface
type DefaultRateLimiter struct {
	config *config.RateLimitConfig
	store  Store
	mu     sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cfg *config.RateLimitConfig, store Store) (*DefaultRateLimiter, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rate limit config: %w", err)
	}

	if store == nil {
		return nil, fmt.Errorf("store is required")
	}

	return &DefaultRateLimiter{
		config: cfg,
		store:  store,
	}, nil
}

// Check verifies if the operation is allowed without recording usage
func (rl *DefaultRateLimiter) Check(ctx context.Context, scope Scope, identifier string) (*CheckResult, error) {
	if !rl.config.Enabled {
		return &CheckResult{Allowed: true}, nil
	}

	if identifier == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	rl.mu.RLock()
	defer rl.mu.RUnlock()

	result := &CheckResult{
		Allowed: true,
		Usages:  make([]Usage, 0, len(rl.config.Limits)),
	}

	now := time.Now()
	var earliestRetry *time.Time

	for _, limit := range rl.config.Limits {
		current, windowEnd, err := rl.store.GetUsage(ctx, scope, identifier, ParseLimitType(limit.Type), ParseTimeWindow(limit.Window))
		if err != nil {
			return nil, fmt.Errorf("failed to get usage for %s/%s: %w", ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), err)
		}

		// If window has expired, reset to 0
		if windowEnd.Before(now) {
			current = 0
			windowEnd = now.Add(ParseTimeWindow(limit.Window).Duration())
		}

		remaining := limit.Limit - current
		if remaining < 0 {
			remaining = 0
		}

		percentage := float64(current) / float64(limit.Limit) * 100

		usage := Usage{
			LimitType:  ParseLimitType(limit.Type),
			Window:     ParseTimeWindow(limit.Window),
			Current:    current,
			Limit:      limit.Limit,
			WindowEnd:  windowEnd,
			Remaining:  remaining,
			Percentage: percentage,
		}

		result.Usages = append(result.Usages, usage)

		// Check if limit is exceeded (strictly greater than)
		if current > limit.Limit {
			result.Allowed = false
			if result.Reason == "" {
				result.Reason = fmt.Sprintf("%s limit exceeded for %s window (%d/%d)",
					ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), current, limit.Limit)
			}
			// Track earliest retry time
			if earliestRetry == nil || windowEnd.Before(*earliestRetry) {
				earliestRetry = &windowEnd
			}
		}
	}

	// Set retry after if any limit was exceeded
	if !result.Allowed && earliestRetry != nil {
		retryDuration := time.Until(*earliestRetry)
		if retryDuration > 0 {
			result.RetryAfter = &retryDuration
		}
	}

	return result, nil
}

// Record records actual usage (tokens and/or count)
func (rl *DefaultRateLimiter) Record(ctx context.Context, scope Scope, identifier string, tokenCount int64, requestCount int64) error {
	if !rl.config.Enabled {
		return nil
	}

	if identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	for _, limit := range rl.config.Limits {
		var amount int64
		switch limit.Type {
		case "token":
			amount = tokenCount
		case "count":
			amount = requestCount
		default:
			continue
		}

		if amount <= 0 {
			continue
		}

		_, windowEnd, err := rl.store.GetUsage(ctx, scope, identifier, ParseLimitType(limit.Type), ParseTimeWindow(limit.Window))
		if err != nil {
			return fmt.Errorf("failed to get usage for %s/%s: %w", ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), err)
		}

		// If window has expired, reset
		if windowEnd.Before(now) {
			windowEnd = now.Add(ParseTimeWindow(limit.Window).Duration())
		}

		_, newWindowEnd, err := rl.store.IncrementUsage(ctx, scope, identifier, ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), amount)
		if err != nil {
			return fmt.Errorf("failed to increment usage for %s/%s: %w", ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), err)
		}

		// Ensure the new usage is set correctly
		if newWindowEnd.Before(now) {
			// Window expired, reset
			windowEnd = now.Add(ParseTimeWindow(limit.Window).Duration())
			if err := rl.store.SetUsage(ctx, scope, identifier, ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), amount, windowEnd); err != nil {
				return fmt.Errorf("failed to reset usage for %s/%s: %w", ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), err)
			}
		}
	}

	return nil
}

// CheckAndRecord checks limits and records usage in a single operation (atomic)
func (rl *DefaultRateLimiter) CheckAndRecord(ctx context.Context, scope Scope, identifier string, tokenCount int64, requestCount int64) (*CheckResult, error) {
	if !rl.config.Enabled {
		return &CheckResult{Allowed: true}, nil
	}

	// Lock for atomic check-and-record
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// First check current state
	result, err := rl.checkUnlocked(ctx, scope, identifier)
	if err != nil {
		return nil, err
	}

	// If not allowed, return without recording
	if !result.Allowed {
		return result, nil
	}

	// Record usage
	if err := rl.recordUnlocked(ctx, scope, identifier, tokenCount, requestCount); err != nil {
		return nil, fmt.Errorf("failed to record usage: %w", err)
	}

	// Re-check to update usage stats in result
	result, err = rl.checkUnlocked(ctx, scope, identifier)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetUsage returns current usage statistics for an identifier
func (rl *DefaultRateLimiter) GetUsage(ctx context.Context, scope Scope, identifier string) ([]Usage, error) {
	if !rl.config.Enabled {
		return []Usage{}, nil
	}

	if identifier == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}

	rl.mu.RLock()
	defer rl.mu.RUnlock()

	usages := make([]Usage, 0, len(rl.config.Limits))
	now := time.Now()

	for _, limit := range rl.config.Limits {
		current, windowEnd, err := rl.store.GetUsage(ctx, scope, identifier, ParseLimitType(limit.Type), ParseTimeWindow(limit.Window))
		if err != nil {
			return nil, fmt.Errorf("failed to get usage for %s/%s: %w", ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), err)
		}

		// If window has expired, reset to 0
		if windowEnd.Before(now) {
			current = 0
			windowEnd = now.Add(ParseTimeWindow(limit.Window).Duration())
		}

		remaining := limit.Limit - current
		if remaining < 0 {
			remaining = 0
		}

		percentage := float64(current) / float64(limit.Limit) * 100

		usage := Usage{
			LimitType:  ParseLimitType(limit.Type),
			Window:     ParseTimeWindow(limit.Window),
			Current:    current,
			Limit:      limit.Limit,
			WindowEnd:  windowEnd,
			Remaining:  remaining,
			Percentage: percentage,
		}

		usages = append(usages, usage)
	}

	return usages, nil
}

// Reset resets usage for an identifier
func (rl *DefaultRateLimiter) Reset(ctx context.Context, scope Scope, identifier string) error {
	if identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	return rl.store.DeleteUsage(ctx, scope, identifier)
}

// ResetExpired removes expired usage records
func (rl *DefaultRateLimiter) ResetExpired(ctx context.Context, before time.Time) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return rl.store.DeleteExpired(ctx, before)
}

// checkUnlocked is the unlocked version of Check (for internal use)
func (rl *DefaultRateLimiter) checkUnlocked(ctx context.Context, scope Scope, identifier string) (*CheckResult, error) {
	result := &CheckResult{
		Allowed: true,
		Usages:  make([]Usage, 0, len(rl.config.Limits)),
	}

	now := time.Now()
	var earliestRetry *time.Time

	for _, limit := range rl.config.Limits {
		current, windowEnd, err := rl.store.GetUsage(ctx, scope, identifier, ParseLimitType(limit.Type), ParseTimeWindow(limit.Window))
		if err != nil {
			return nil, fmt.Errorf("failed to get usage for %s/%s: %w", ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), err)
		}

		// If window has expired, reset to 0
		if windowEnd.Before(now) {
			current = 0
			windowEnd = now.Add(ParseTimeWindow(limit.Window).Duration())
		}

		remaining := limit.Limit - current
		if remaining < 0 {
			remaining = 0
		}

		percentage := float64(current) / float64(limit.Limit) * 100

		usage := Usage{
			LimitType:  ParseLimitType(limit.Type),
			Window:     ParseTimeWindow(limit.Window),
			Current:    current,
			Limit:      limit.Limit,
			WindowEnd:  windowEnd,
			Remaining:  remaining,
			Percentage: percentage,
		}

		result.Usages = append(result.Usages, usage)

		// Check if limit is exceeded (strictly greater than)
		if current > limit.Limit {
			result.Allowed = false
			if result.Reason == "" {
				result.Reason = fmt.Sprintf("%s limit exceeded for %s window (%d/%d)",
					ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), current, limit.Limit)
			}
			// Track earliest retry time
			if earliestRetry == nil || windowEnd.Before(*earliestRetry) {
				earliestRetry = &windowEnd
			}
		}
	}

	// Set retry after if any limit was exceeded
	if !result.Allowed && earliestRetry != nil {
		retryDuration := time.Until(*earliestRetry)
		if retryDuration > 0 {
			result.RetryAfter = &retryDuration
		}
	}

	return result, nil
}

// recordUnlocked is the unlocked version of Record (for internal use)
func (rl *DefaultRateLimiter) recordUnlocked(ctx context.Context, scope Scope, identifier string, tokenCount int64, requestCount int64) error {
	now := time.Now()

	for _, limit := range rl.config.Limits {
		var amount int64
		switch limit.Type {
		case "token":
			amount = tokenCount
		case "count":
			amount = requestCount
		default:
			continue
		}

		if amount <= 0 {
			continue
		}

		_, windowEnd, err := rl.store.GetUsage(ctx, scope, identifier, ParseLimitType(limit.Type), ParseTimeWindow(limit.Window))
		if err != nil {
			return fmt.Errorf("failed to get usage for %s/%s: %w", ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), err)
		}

		// If window has expired, reset
		if windowEnd.Before(now) {
			windowEnd = now.Add(ParseTimeWindow(limit.Window).Duration())
			if err := rl.store.SetUsage(ctx, scope, identifier, ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), amount, windowEnd); err != nil {
				return fmt.Errorf("failed to reset usage for %s/%s: %w", ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), err)
			}
			continue
		}

		_, _, err = rl.store.IncrementUsage(ctx, scope, identifier, ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), amount)
		if err != nil {
			return fmt.Errorf("failed to increment usage for %s/%s: %w", ParseLimitType(limit.Type), ParseTimeWindow(limit.Window), err)
		}
	}

	return nil
}
