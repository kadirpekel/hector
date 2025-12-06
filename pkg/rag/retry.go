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

package rag

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"time"
)

// RetryConfig configures retry behavior.
//
// Reuses patterns from v2/httpclient for consistency.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (default: 3).
	MaxRetries int

	// BaseDelay is the initial delay between retries (default: 1s).
	BaseDelay time.Duration

	// MaxDelay is the maximum delay between retries (default: 30s).
	MaxDelay time.Duration

	// JitterFactor adds randomness to delays (0.0-1.0, default: 0.1).
	JitterFactor float64

	// RetryableErrors are error substrings that indicate retryable failures.
	RetryableErrors []string
}

// DefaultRetryConfig returns sensible defaults for RAG operations.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		BaseDelay:    time.Second,
		MaxDelay:     30 * time.Second,
		JitterFactor: 0.1,
		RetryableErrors: []string{
			"connection refused",
			"connection reset",
			"timeout",
			"rate limit",
			"429",
			"500",
			"502",
			"503",
			"504",
			"temporarily unavailable",
			"too many requests",
			"ECONNREFUSED",
			"ETIMEDOUT",
			"ECONNRESET",
		},
	}
}

// Retryer handles retry logic with exponential backoff.
//
// Based on v2/httpclient patterns but generalized for any operation.
type Retryer struct {
	config RetryConfig
}

// NewRetryer creates a new retryer with the given config.
func NewRetryer(cfg RetryConfig) *Retryer {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = time.Second
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	if cfg.JitterFactor <= 0 {
		cfg.JitterFactor = 0.1
	}

	return &Retryer{config: cfg}
}

// Do executes the operation with retry logic.
//
// Returns the first successful result or the last error after all retries.
func (r *Retryer) Do(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check context before each attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute operation
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !r.isRetryable(err) {
			slog.Debug("Non-retryable error",
				"operation", operation,
				"error", err)
			return err
		}

		// Check if we've exhausted retries
		if attempt >= r.config.MaxRetries {
			slog.Warn("Max retries exceeded",
				"operation", operation,
				"attempts", attempt+1,
				"error", err)
			return &RetryError{
				Operation:   operation,
				Attempts:    attempt + 1,
				LastError:   err,
				IsExhausted: true,
			}
		}

		// Calculate delay with exponential backoff and jitter
		delay := r.calculateDelay(attempt)

		slog.Debug("Retrying operation",
			"operation", operation,
			"attempt", attempt+1,
			"max_attempts", r.config.MaxRetries+1,
			"delay", delay,
			"error", err)

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}

// DoWithResult executes an operation that returns a value.
func DoWithResult[T any](ctx context.Context, r *Retryer, operation string, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		var err error
		result, err = fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !r.isRetryable(err) {
			return result, err
		}

		if attempt >= r.config.MaxRetries {
			return result, &RetryError{
				Operation:   operation,
				Attempts:    attempt + 1,
				LastError:   err,
				IsExhausted: true,
			}
		}

		delay := r.calculateDelay(attempt)

		slog.Debug("Retrying operation",
			"operation", operation,
			"attempt", attempt+1,
			"delay", delay,
			"error", err)

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}
	}

	return result, lastErr
}

// isRetryable checks if an error should be retried.
func (r *Retryer) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for context errors (not retryable)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for RetryError that's already exhausted
	var retryErr *RetryError
	if errors.As(err, &retryErr) && retryErr.IsExhausted {
		return false
	}

	// Check error message against retryable patterns
	errStr := strings.ToLower(err.Error())
	for _, pattern := range r.config.RetryableErrors {
		if strings.Contains(errStr, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// calculateDelay computes delay with exponential backoff and jitter.
func (r *Retryer) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	delay := time.Duration(math.Pow(2, float64(attempt))) * r.config.BaseDelay

	// Add jitter (±JitterFactor)
	jitter := time.Duration(rand.Float64() * float64(delay) * r.config.JitterFactor)
	if rand.Float64() < 0.5 {
		delay -= jitter
	} else {
		delay += jitter
	}

	// Clamp to max delay
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}

	return delay
}

// RetryError represents an error after retry attempts.
type RetryError struct {
	Operation   string
	Attempts    int
	LastError   error
	IsExhausted bool
}

func (e *RetryError) Error() string {
	if e.IsExhausted {
		return fmt.Sprintf("%s failed after %d attempts: %v", e.Operation, e.Attempts, e.LastError)
	}
	return fmt.Sprintf("%s failed (attempt %d): %v", e.Operation, e.Attempts, e.LastError)
}

func (e *RetryError) Unwrap() error {
	return e.LastError
}

// IsRetryExhausted checks if an error is a retry exhaustion error.
func IsRetryExhausted(err error) bool {
	var retryErr *RetryError
	return errors.As(err, &retryErr) && retryErr.IsExhausted
}
