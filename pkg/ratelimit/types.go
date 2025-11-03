package ratelimit

import (
	"time"
)

// TimeWindow represents a rate limiting time window
type TimeWindow string

const (
	WindowMinute TimeWindow = "minute"
	WindowHour   TimeWindow = "hour"
	WindowDay    TimeWindow = "day"
	WindowWeek   TimeWindow = "week"
	WindowMonth  TimeWindow = "month"
)

// Duration returns the duration for the time window
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

// LimitType represents the type of rate limit
type LimitType string

const (
	LimitTypeToken LimitType = "token" // Token usage limit
	LimitTypeCount LimitType = "count" // Request count limit
)

// ParseTimeWindow converts config string to TimeWindow
func ParseTimeWindow(s string) TimeWindow {
	return TimeWindow(s)
}

// ParseLimitType converts config string to LimitType
func ParseLimitType(s string) LimitType {
	return LimitType(s)
}

// Usage represents current usage for a specific limit
type Usage struct {
	LimitType  LimitType  `json:"limit_type"`
	Window     TimeWindow `json:"window"`
	Current    int64      `json:"current"`    // Current usage in window
	Limit      int64      `json:"limit"`      // Maximum allowed
	WindowEnd  time.Time  `json:"window_end"` // When current window ends
	Remaining  int64      `json:"remaining"`  // Remaining quota
	Percentage float64    `json:"percentage"` // Usage percentage
}

// CheckResult represents the result of a rate limit check
type CheckResult struct {
	Allowed    bool           `json:"allowed"`
	Reason     string         `json:"reason,omitempty"`
	Usages     []Usage        `json:"usages"`
	RetryAfter *time.Duration `json:"retry_after,omitempty"` // How long to wait before retrying
}

// IsExceeded returns true if any limit is exceeded
func (r *CheckResult) IsExceeded() bool {
	return !r.Allowed
}

// GetUsage returns usage for a specific limit type and window
func (r *CheckResult) GetUsage(limitType LimitType, window TimeWindow) *Usage {
	for i := range r.Usages {
		if r.Usages[i].LimitType == limitType && r.Usages[i].Window == window {
			return &r.Usages[i]
		}
	}
	return nil
}
