package ratelimit

import (
	"fmt"

	"github.com/verikod/hector/pkg/config"
)

// NewRateLimiterFromConfig creates a RateLimiter using the database.
// If rate limiting is disabled, returns nil.
func NewRateLimiterFromConfig(rateLimitCfg *config.RateLimitConfig, pool *config.DBPool, databaseDSN string) (RateLimiter, error) {
	if rateLimitCfg == nil || !rateLimitCfg.IsEnabled() {
		return nil, nil
	}

	// Parse database config
	dbCfg, err := config.ParseDSN(databaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN: %w", err)
	}

	if pool == nil {
		return nil, fmt.Errorf("DBPool is required for rate limiting")
	}

	db, err := pool.Get(dbCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	store, err := NewSQLStore(db, dbCfg.Dialect())
	if err != nil {
		return nil, fmt.Errorf("failed to create SQL store: %w", err)
	}

	// Convert config limits to LimitRules
	limits := make([]LimitRule, len(rateLimitCfg.Limits))
	for i, l := range rateLimitCfg.Limits {
		limits[i] = LimitRule{
			Type:   ParseLimitType(l.Type),
			Window: ParseTimeWindow(l.Window),
			Limit:  l.Limit,
		}
	}

	// Create limiter config
	limiterCfg := &Config{
		Enabled: rateLimitCfg.IsEnabled(),
		Limits:  limits,
	}

	return NewRateLimiter(limiterCfg, store)
}

// NewRateLimiterFromConfigWithStore creates a RateLimiter with a custom store.
// Useful for testing or when you need to share a store across multiple limiters.
func NewRateLimiterFromConfigWithStore(cfg *config.RateLimitConfig, store Store) (RateLimiter, error) {
	if cfg == nil || !cfg.IsEnabled() {
		return nil, nil
	}

	if store == nil {
		return nil, fmt.Errorf("store is required")
	}

	// Convert config limits to LimitRules
	limits := make([]LimitRule, len(cfg.Limits))
	for i, l := range cfg.Limits {
		limits[i] = LimitRule{
			Type:   ParseLimitType(l.Type),
			Window: ParseTimeWindow(l.Window),
			Limit:  l.Limit,
		}
	}

	// Create limiter config
	limiterCfg := &Config{
		Enabled: cfg.IsEnabled(),
		Limits:  limits,
	}

	return NewRateLimiter(limiterCfg, store)
}

// ScopeFromConfig returns the rate limiting scope from configuration.
func ScopeFromConfig(cfg *config.RateLimitConfig) Scope {
	if cfg == nil || cfg.Scope == "" {
		return ScopeSession
	}
	return ParseScope(cfg.Scope)
}
