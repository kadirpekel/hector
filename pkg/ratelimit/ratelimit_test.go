package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
)

func TestRateLimiter_BasicTokenLimit(t *testing.T) {
	enabled := true
	config := config.RateLimitConfig{
		Enabled: &enabled,
		Limits: []config.RateLimitRule{
			{Type: "token", Window: "minute", Limit: 100},
		},
	}

	store := NewMemoryStore()
	limiter, err := NewRateLimiter(&config, store)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// First request: 50 tokens - should be allowed
	result, err := limiter.CheckAndRecord(ctx, ScopeSession, "session1", 50, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected request to be allowed")
	}

	// Check usage
	usage := result.GetUsage(LimitTypeToken, WindowMinute)
	if usage == nil {
		t.Fatal("expected token usage to be present")
	}
	if usage.Current != 50 {
		t.Errorf("expected current usage to be 50, got %d", usage.Current)
	}
	if usage.Remaining != 50 {
		t.Errorf("expected remaining to be 50, got %d", usage.Remaining)
	}

	// Second request: 40 tokens - should be allowed (total 90)
	result, err = limiter.CheckAndRecord(ctx, ScopeSession, "session1", 40, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected request to be allowed")
	}

	usage = result.GetUsage(LimitTypeToken, WindowMinute)
	if usage.Current != 90 {
		t.Errorf("expected current usage to be 90, got %d", usage.Current)
	}

	// Third request: 20 tokens - should be denied (would exceed limit)
	result, err = limiter.CheckAndRecord(ctx, ScopeSession, "session1", 20, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Errorf("expected request to be denied")
	}
	if result.RetryAfter == nil {
		t.Errorf("expected retry_after to be set")
	}
}

func TestRateLimiter_BasicCountLimit(t *testing.T) {
	config := config.RateLimitConfig{
		Enabled: &[]bool{true}[0],
		Limits: []config.RateLimitRule{
			{Type: "count", Window: "minute", Limit: 5},
		},
	}

	store := NewMemoryStore()
	limiter, err := NewRateLimiter(&config, store)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// Make 5 requests - all should be allowed
	for i := 1; i <= 5; i++ {
		result, err := limiter.CheckAndRecord(ctx, ScopeSession, "session1", 0, 1)
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i, err)
		}
		if !result.Allowed {
			t.Errorf("expected request %d to be allowed", i)
		}

		usage := result.GetUsage(LimitTypeCount, WindowMinute)
		if usage.Current != int64(i) {
			t.Errorf("expected current usage to be %d, got %d", i, usage.Current)
		}
	}

	// 6th request should be denied
	result, err := limiter.CheckAndRecord(ctx, ScopeSession, "session1", 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Errorf("expected 6th request to be denied")
	}
}

func TestRateLimiter_MultiLayerLimits(t *testing.T) {
	config := config.RateLimitConfig{
		Enabled: &[]bool{true}[0],
		Limits: []config.RateLimitRule{
			{Type: "token", Window: "minute", Limit: 100},
			{Type: "token", Window: "day", Limit: 1000},
			{Type: "count", Window: "minute", Limit: 10},
		},
	}

	store := NewMemoryStore()
	limiter, err := NewRateLimiter(&config, store)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// Make requests that are within all limits
	result, err := limiter.CheckAndRecord(ctx, ScopeSession, "session1", 50, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected request to be allowed")
	}

	// Check all three limits are tracked
	if len(result.Usages) != 3 {
		t.Errorf("expected 3 usage records, got %d", len(result.Usages))
	}

	tokenMinute := result.GetUsage(LimitTypeToken, WindowMinute)
	if tokenMinute == nil || tokenMinute.Current != 50 {
		t.Errorf("expected token/minute usage to be 50")
	}

	tokenDay := result.GetUsage(LimitTypeToken, WindowDay)
	if tokenDay == nil || tokenDay.Current != 50 {
		t.Errorf("expected token/day usage to be 50")
	}

	countMinute := result.GetUsage(LimitTypeCount, WindowMinute)
	if countMinute == nil || countMinute.Current != 5 {
		t.Errorf("expected count/minute usage to be 5")
	}
}

func TestRateLimiter_SeparateSessions(t *testing.T) {
	config := config.RateLimitConfig{
		Enabled: &[]bool{true}[0],
		Limits: []config.RateLimitRule{
			{Type: "count", Window: "minute", Limit: 5},
		},
	}

	store := NewMemoryStore()
	limiter, err := NewRateLimiter(&config, store)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// Session 1: use 5 requests
	for i := 0; i < 5; i++ {
		_, err := limiter.CheckAndRecord(ctx, ScopeSession, "session1", 0, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Session 2: should still have full quota
	result, err := limiter.CheckAndRecord(ctx, ScopeSession, "session2", 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected session2 to be allowed (separate quota)")
	}

	// Session 1: should be blocked
	result, err = limiter.CheckAndRecord(ctx, ScopeSession, "session1", 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Errorf("expected session1 to be blocked")
	}
}

func TestRateLimiter_UserScope(t *testing.T) {
	config := config.RateLimitConfig{
		Enabled: &[]bool{true}[0],
		Limits: []config.RateLimitRule{
			{Type: "count", Window: "minute", Limit: 10},
		},
	}

	store := NewMemoryStore()
	limiter, err := NewRateLimiter(&config, store)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// User scope: multiple sessions share the same quota
	// Make 5 requests for user1 via session1
	for i := 0; i < 5; i++ {
		_, err := limiter.CheckAndRecord(ctx, ScopeUser, "user1", 0, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Make 5 more requests for user1 via session2 (should still count toward same user quota)
	for i := 0; i < 5; i++ {
		_, err := limiter.CheckAndRecord(ctx, ScopeUser, "user1", 0, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Next request should be blocked (10 total)
	result, err := limiter.CheckAndRecord(ctx, ScopeUser, "user1", 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Errorf("expected user1 to be blocked after 10 requests")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	config := config.RateLimitConfig{
		Enabled: &[]bool{true}[0],
		Limits: []config.RateLimitRule{
			{Type: "count", Window: "minute", Limit: 5},
		},
	}

	store := NewMemoryStore()
	limiter, err := NewRateLimiter(&config, store)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// Use up quota
	for i := 0; i < 5; i++ {
		_, err := limiter.CheckAndRecord(ctx, ScopeSession, "session1", 0, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Should be blocked
	result, err := limiter.CheckAndRecord(ctx, ScopeSession, "session1", 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Errorf("expected to be blocked")
	}

	// Reset
	err = limiter.Reset(ctx, ScopeSession, "session1")
	if err != nil {
		t.Fatalf("failed to reset: %v", err)
	}

	// Should be allowed again
	result, err = limiter.CheckAndRecord(ctx, ScopeSession, "session1", 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Errorf("expected to be allowed after reset")
	}
}

func TestRateLimiter_DisabledConfig(t *testing.T) {
	config := config.RateLimitConfig{
		Enabled: &[]bool{false}[0],
		Limits:  []config.RateLimitRule{}, // No limits when disabled
	}

	store := NewMemoryStore()
	limiter, err := NewRateLimiter(&config, store)
	if err != nil {
		t.Fatalf("failed to create limiter: %v", err)
	}

	ctx := context.Background()

	// Should always allow when disabled
	for i := 0; i < 1000; i++ {
		result, err := limiter.CheckAndRecord(ctx, ScopeSession, "session1", 1000000, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("expected to be allowed when rate limiting is disabled")
		}
	}
}

func TestMemoryStore_WindowExpiration(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Set usage with a window that expires soon
	windowEnd := time.Now().Add(100 * time.Millisecond)
	err := store.SetUsage(ctx, ScopeSession, "session1", LimitTypeCount, WindowMinute, 100, windowEnd)
	if err != nil {
		t.Fatalf("failed to set usage: %v", err)
	}

	// Get usage immediately - should return 100
	amount, _, err := store.GetUsage(ctx, ScopeSession, "session1", LimitTypeCount, WindowMinute)
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}
	if amount != 100 {
		t.Errorf("expected amount to be 100, got %d", amount)
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Get usage after expiration - should return 0
	amount, newWindowEnd, err := store.GetUsage(ctx, ScopeSession, "session1", LimitTypeCount, WindowMinute)
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}
	if amount != 0 {
		t.Errorf("expected amount to be 0 after expiration, got %d", amount)
	}
	if !newWindowEnd.After(time.Now()) {
		t.Errorf("expected new window end to be in the future")
	}
}

func TestRateLimitConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  config.RateLimitConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: config.RateLimitConfig{
				Enabled: &[]bool{true}[0],
				Limits: []config.RateLimitRule{
					{Type: "token", Window: "day", Limit: 1000},
				},
			},
			wantErr: false,
		},
		{
			name: "disabled config",
			config: config.RateLimitConfig{
				Enabled: &[]bool{false}[0],
				Limits:  []config.RateLimitRule{},
			},
			wantErr: false,
		},
		{
			name: "enabled but no limits",
			config: config.RateLimitConfig{
				Enabled: &[]bool{true}[0],
				Limits:  []config.RateLimitRule{},
			},
			wantErr: true,
		},
		{
			name: "invalid limit type",
			config: config.RateLimitConfig{
				Enabled: &[]bool{true}[0],
				Limits: []config.RateLimitRule{
					{Type: "invalid", Window: "day", Limit: 1000},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid window",
			config: config.RateLimitConfig{
				Enabled: &[]bool{true}[0],
				Limits: []config.RateLimitRule{
					{Type: "token", Window: "invalid", Limit: 1000},
				},
			},
			wantErr: true,
		},
		{
			name: "zero limit",
			config: config.RateLimitConfig{
				Enabled: &[]bool{true}[0],
				Limits: []config.RateLimitRule{
					{Type: "token", Window: "day", Limit: 0},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
