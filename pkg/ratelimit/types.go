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
	"time"
)

// Scope represents the scope of rate limiting.
type Scope string

const (
	// ScopeSession applies rate limits per session.
	ScopeSession Scope = "session"

	// ScopeUser applies rate limits per user (across all sessions).
	ScopeUser Scope = "user"
)

// TimeWindow represents a rate limiting time window.
type TimeWindow string

const (
	// WindowMinute represents a 60-second window (burst protection).
	WindowMinute TimeWindow = "minute"

	// WindowHour represents a 60-minute window (short-term limits).
	WindowHour TimeWindow = "hour"

	// WindowDay represents a 24-hour window (daily quotas).
	WindowDay TimeWindow = "day"

	// WindowWeek represents a 7-day window (weekly budgets).
	WindowWeek TimeWindow = "week"

	// WindowMonth represents a 30-day window (monthly billing).
	WindowMonth TimeWindow = "month"
)

// Duration returns the duration for the time window.
func (w TimeWindow) Duration() time.Duration {
	switch w {
	case WindowMinute:
		return time.Minute
	case WindowHour:
		return time.Hour
	case WindowDay:
		return 24 * time.Hour
	case WindowWeek:
		return 7 * 24 * time.Hour
	case WindowMonth:
		return 30 * 24 * time.Hour // Approximate month
	default:
		return time.Hour
	}
}

// String returns the string representation of the time window.
func (w TimeWindow) String() string {
	return string(w)
}

// LimitType represents the type of rate limit.
type LimitType string

const (
	// LimitTypeToken tracks token usage (LLM API tokens).
	LimitTypeToken LimitType = "token"

	// LimitTypeCount tracks request count.
	LimitTypeCount LimitType = "count"
)

// String returns the string representation of the limit type.
func (t LimitType) String() string {
	return string(t)
}

// ParseTimeWindow converts a config string to TimeWindow.
func ParseTimeWindow(s string) TimeWindow {
	switch s {
	case "minute":
		return WindowMinute
	case "hour":
		return WindowHour
	case "day":
		return WindowDay
	case "week":
		return WindowWeek
	case "month":
		return WindowMonth
	default:
		return TimeWindow(s)
	}
}

// ParseLimitType converts a config string to LimitType.
func ParseLimitType(s string) LimitType {
	switch s {
	case "token":
		return LimitTypeToken
	case "count":
		return LimitTypeCount
	default:
		return LimitType(s)
	}
}

// ParseScope converts a config string to Scope.
func ParseScope(s string) Scope {
	switch s {
	case "session":
		return ScopeSession
	case "user":
		return ScopeUser
	default:
		return Scope(s)
	}
}

// Usage represents current usage for a specific limit.
type Usage struct {
	// LimitType is the type of limit (token or count).
	LimitType LimitType `json:"limit_type"`

	// Window is the time window for this usage.
	Window TimeWindow `json:"window"`

	// Current is the current usage in the window.
	Current int64 `json:"current"`

	// Limit is the maximum allowed in the window.
	Limit int64 `json:"limit"`

	// WindowEnd is when the current window ends.
	WindowEnd time.Time `json:"window_end"`

	// Remaining is the remaining quota in the window.
	Remaining int64 `json:"remaining"`

	// Percentage is the usage percentage (0-100+).
	Percentage float64 `json:"percentage"`
}

// CheckResult represents the result of a rate limit check.
type CheckResult struct {
	// Allowed indicates whether the operation is allowed.
	Allowed bool `json:"allowed"`

	// Reason provides a human-readable reason if denied.
	Reason string `json:"reason,omitempty"`

	// Usages contains current usage for all limits.
	Usages []Usage `json:"usages"`

	// RetryAfter indicates how long to wait before retrying (if denied).
	RetryAfter *time.Duration `json:"retry_after,omitempty"`
}

// IsExceeded returns true if any limit is exceeded.
func (r *CheckResult) IsExceeded() bool {
	return !r.Allowed
}

// GetUsage returns usage for a specific limit type and window.
func (r *CheckResult) GetUsage(limitType LimitType, window TimeWindow) *Usage {
	for i := range r.Usages {
		if r.Usages[i].LimitType == limitType && r.Usages[i].Window == window {
			return &r.Usages[i]
		}
	}
	return nil
}

// GetHighestUsagePercentage returns the highest usage percentage across all limits.
func (r *CheckResult) GetHighestUsagePercentage() float64 {
	var highest float64
	for _, u := range r.Usages {
		if u.Percentage > highest {
			highest = u.Percentage
		}
	}
	return highest
}
