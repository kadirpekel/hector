// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ratelimit

import (
	"context"
	"time"
)

// RateLimiter is the main interface for rate limiting.
//
// Implementations must be thread-safe and support concurrent access.
type RateLimiter interface {
	// Check verifies if the operation is allowed without recording usage.
	// Use this when you want to check limits before potentially expensive operations.
	Check(ctx context.Context, scope Scope, identifier string) (*CheckResult, error)

	// Record records actual usage (tokens and/or count).
	// Use this after an operation completes to record the actual usage.
	Record(ctx context.Context, scope Scope, identifier string, tokenCount int64, requestCount int64) error

	// CheckAndRecord checks limits and records usage in a single atomic operation.
	// This is the recommended method for most use cases as it prevents race conditions.
	CheckAndRecord(ctx context.Context, scope Scope, identifier string, tokenCount int64, requestCount int64) (*CheckResult, error)

	// GetUsage returns current usage statistics for an identifier.
	// Returns usage for all configured limits.
	GetUsage(ctx context.Context, scope Scope, identifier string) ([]Usage, error)

	// Reset resets usage for an identifier.
	// Useful for testing or manual quota resets.
	Reset(ctx context.Context, scope Scope, identifier string) error

	// ResetExpired removes expired usage records.
	// Should be called periodically for cleanup.
	ResetExpired(ctx context.Context, before time.Time) error
}

// Store is the persistence layer for rate limit data.
//
// Implementations must be thread-safe and support concurrent access.
type Store interface {
	// GetUsage gets current usage for a specific limit.
	// Returns the current amount, window end time, and any error.
	// If no usage exists, returns 0 with a new window end time.
	GetUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow) (int64, time.Time, error)

	// IncrementUsage increments usage for a specific limit.
	// Returns the new amount, window end time, and any error.
	// If the window has expired, it resets and starts a new window.
	IncrementUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow, amount int64) (int64, time.Time, error)

	// SetUsage sets usage for a specific limit.
	// Used for explicit resets or window rollovers.
	SetUsage(ctx context.Context, scope Scope, identifier string, limitType LimitType, window TimeWindow, amount int64, windowEnd time.Time) error

	// DeleteUsage deletes all usage records for an identifier.
	DeleteUsage(ctx context.Context, scope Scope, identifier string) error

	// DeleteExpired deletes all expired usage records.
	// Records with windowEnd before the specified time are deleted.
	DeleteExpired(ctx context.Context, before time.Time) error

	// Close closes the store and releases resources.
	Close() error
}

// Ensure interface compliance at compile time.
var (
	_ RateLimiter = (*DefaultRateLimiter)(nil)
	_ Store       = (*MemoryStore)(nil)
	_ Store       = (*SQLStore)(nil)
)
