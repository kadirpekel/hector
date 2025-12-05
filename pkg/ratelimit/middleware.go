// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ratelimit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// IdentifierFunc extracts the rate limit identifier from an HTTP request.
// This function should return the identifier and scope for the request.
type IdentifierFunc func(r *http.Request) (identifier string, scope Scope)

// DefaultIdentifierFunc extracts the identifier from the request.
// It uses the session ID header if present, otherwise falls back to remote address.
func DefaultIdentifierFunc(r *http.Request) (string, Scope) {
	// Try to get session ID from header
	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		return sessionID, ScopeSession
	}

	// Try to get user ID from header (set by auth middleware)
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID, ScopeUser
	}

	// Fall back to remote address
	return r.RemoteAddr, ScopeSession
}

// MiddlewareConfig configures the rate limiting middleware.
type MiddlewareConfig struct {
	// Limiter is the rate limiter to use.
	Limiter RateLimiter

	// IdentifierFunc extracts the identifier and scope from requests.
	// If nil, DefaultIdentifierFunc is used.
	IdentifierFunc IdentifierFunc

	// TokenEstimator estimates token count for a request.
	// If nil, token count is set to 0 (only count-based limiting).
	TokenEstimator func(r *http.Request) int64

	// ExcludedPaths are paths that bypass rate limiting.
	ExcludedPaths []string

	// OnLimited is called when a request is rate limited.
	// If nil, a default JSON error response is sent.
	OnLimited func(w http.ResponseWriter, r *http.Request, result *CheckResult)
}

// Middleware creates an HTTP middleware that enforces rate limits.
func Middleware(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	if cfg.Limiter == nil {
		// No limiter configured, pass through
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	if cfg.IdentifierFunc == nil {
		cfg.IdentifierFunc = DefaultIdentifierFunc
	}

	if cfg.OnLimited == nil {
		cfg.OnLimited = defaultOnLimited
	}

	// Build excluded paths map for fast lookup
	excludedPaths := make(map[string]bool)
	for _, path := range cfg.ExcludedPaths {
		excludedPaths[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is excluded
			if excludedPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Extract identifier and scope
			identifier, scope := cfg.IdentifierFunc(r)
			if identifier == "" {
				// No identifier, pass through
				next.ServeHTTP(w, r)
				return
			}

			// Estimate token count
			var tokenCount int64
			if cfg.TokenEstimator != nil {
				tokenCount = cfg.TokenEstimator(r)
			}

			// Check rate limit
			ctx := r.Context()
			result, err := cfg.Limiter.CheckAndRecord(ctx, scope, identifier, tokenCount, 1)
			if err != nil {
				slog.Error("Rate limit check failed", "error", err, "identifier", identifier)
				// On error, allow the request (fail open)
				next.ServeHTTP(w, r)
				return
			}

			// Store usage in context for downstream handlers
			ctx = context.WithValue(ctx, rateLimitUsageKey{}, result)
			r = r.WithContext(ctx)

			if !result.Allowed {
				cfg.OnLimited(w, r, result)
				return
			}

			// Add rate limit headers to response
			addRateLimitHeaders(w, result)

			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitUsageKey is the context key for rate limit usage.
type rateLimitUsageKey struct{}

// UsageFromContext extracts rate limit usage from the request context.
func UsageFromContext(ctx context.Context) *CheckResult {
	if result, ok := ctx.Value(rateLimitUsageKey{}).(*CheckResult); ok {
		return result
	}
	return nil
}

// defaultOnLimited sends a default 429 response.
func defaultOnLimited(w http.ResponseWriter, r *http.Request, result *CheckResult) {
	w.Header().Set("Content-Type", "application/json")

	// Add retry-after header
	if result.RetryAfter != nil && *result.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.FormatInt(int64(result.RetryAfter.Seconds()), 10))
	}

	// Add rate limit headers
	addRateLimitHeaders(w, result)

	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    "rate_limit_exceeded",
			"message": result.Reason,
		},
	}

	if result.RetryAfter != nil {
		response["retry_after_seconds"] = int64(result.RetryAfter.Seconds())
	}

	// Include usage details
	if len(result.Usages) > 0 {
		usages := make([]map[string]interface{}, len(result.Usages))
		for i, u := range result.Usages {
			usages[i] = map[string]interface{}{
				"type":       u.LimitType,
				"window":     u.Window,
				"current":    u.Current,
				"limit":      u.Limit,
				"remaining":  u.Remaining,
				"percentage": u.Percentage,
				"resets_at":  u.WindowEnd.Format(time.RFC3339),
			}
		}
		response["usage"] = usages
	}

	_ = json.NewEncoder(w).Encode(response)
}

// addRateLimitHeaders adds standard rate limit headers to the response.
func addRateLimitHeaders(w http.ResponseWriter, result *CheckResult) {
	if result == nil || len(result.Usages) == 0 {
		return
	}

	// Use the first usage for standard headers (typically the most restrictive)
	// In practice, you might want to show the most restrictive limit
	var mostRestrictive *Usage
	for i := range result.Usages {
		u := &result.Usages[i]
		if mostRestrictive == nil || u.Percentage > mostRestrictive.Percentage {
			mostRestrictive = u
		}
	}

	if mostRestrictive != nil {
		w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(mostRestrictive.Limit, 10))
		w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(mostRestrictive.Remaining, 10))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(mostRestrictive.WindowEnd.Unix(), 10))
	}
}

// SimpleMiddleware creates a simple rate limiting middleware.
// This is a convenience function for common use cases.
func SimpleMiddleware(limiter RateLimiter, excludedPaths ...string) func(http.Handler) http.Handler {
	return Middleware(MiddlewareConfig{
		Limiter:       limiter,
		ExcludedPaths: excludedPaths,
	})
}
