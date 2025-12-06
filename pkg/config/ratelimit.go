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

package config

import "fmt"

// RateLimitConfig defines rate limiting configuration.
type RateLimitConfig struct {
	// Enabled controls whether rate limiting is active.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// Scope is the rate limiting scope ("session" or "user").
	Scope string `yaml:"scope,omitempty" json:"scope,omitempty"`

	// Backend is the storage backend ("memory" or "sql").
	Backend string `yaml:"backend,omitempty" json:"backend,omitempty"`

	// SQLDatabase is the reference to a SQL database from the databases section.
	// Required when backend is "sql".
	SQLDatabase string `yaml:"sql_database,omitempty" json:"sql_database,omitempty"`

	// Limits defines the rate limit rules.
	Limits []RateLimitRule `yaml:"limits,omitempty" json:"limits,omitempty"`
}

// RateLimitRule defines a single rate limit rule.
type RateLimitRule struct {
	// Type is the limit type ("token" or "count").
	Type string `yaml:"type" json:"type"`

	// Window is the time window ("minute", "hour", "day", "week", "month").
	Window string `yaml:"window" json:"window"`

	// Limit is the maximum allowed in the window.
	Limit int64 `yaml:"limit" json:"limit"`
}

// IsEnabled returns true if rate limiting is enabled.
func (c *RateLimitConfig) IsEnabled() bool {
	return c != nil && c.Enabled != nil && *c.Enabled
}

// SetDefaults sets default values for RateLimitConfig.
func (c *RateLimitConfig) SetDefaults() {
	if c.Enabled == nil {
		c.Enabled = BoolPtr(false)
	}
	if c.IsEnabled() && len(c.Limits) == 0 {
		// Default: 100k tokens per day, 60 requests per minute
		c.Limits = []RateLimitRule{
			{Type: "token", Window: "day", Limit: 100000},
			{Type: "count", Window: "minute", Limit: 60},
		}
	}
	if c.Scope == "" {
		c.Scope = "session"
	}
	if c.Backend == "" {
		c.Backend = "memory"
	}
}

// Validate validates the RateLimitConfig.
func (c *RateLimitConfig) Validate() error {
	if !c.IsEnabled() {
		return nil
	}

	// Validate scope
	if c.Scope != "" && c.Scope != "session" && c.Scope != "user" {
		return fmt.Errorf("invalid rate_limiting.scope '%s', must be 'session' or 'user'", c.Scope)
	}

	// Validate backend
	if c.Backend != "" && c.Backend != "memory" && c.Backend != "sql" {
		return fmt.Errorf("invalid rate_limiting.backend '%s', must be 'memory' or 'sql'", c.Backend)
	}

	// Validate SQL backend requires database reference
	if c.Backend == "sql" && c.SQLDatabase == "" {
		return fmt.Errorf("rate_limiting.backend 'sql' requires 'sql_database' reference")
	}

	// Validate limits
	if len(c.Limits) == 0 {
		return fmt.Errorf("rate_limiting.limits is required when rate limiting is enabled")
	}

	for i, limit := range c.Limits {
		if err := c.validateLimit(i, limit); err != nil {
			return err
		}
	}

	return nil
}

// validateLimit validates a single rate limit rule.
func (c *RateLimitConfig) validateLimit(index int, limit RateLimitRule) error {
	// Validate type
	if limit.Type == "" {
		return fmt.Errorf("rate_limiting.limits[%d].type is required", index)
	}
	if limit.Type != "token" && limit.Type != "count" {
		return fmt.Errorf("invalid rate_limiting.limits[%d].type '%s', must be 'token' or 'count'", index, limit.Type)
	}

	// Validate window
	if limit.Window == "" {
		return fmt.Errorf("rate_limiting.limits[%d].window is required", index)
	}
	validWindows := map[string]bool{
		"minute": true,
		"hour":   true,
		"day":    true,
		"week":   true,
		"month":  true,
	}
	if !validWindows[limit.Window] {
		return fmt.Errorf("invalid rate_limiting.limits[%d].window '%s', must be 'minute', 'hour', 'day', 'week', or 'month'", index, limit.Window)
	}

	// Validate limit
	if limit.Limit <= 0 {
		return fmt.Errorf("rate_limiting.limits[%d].limit must be positive", index)
	}

	return nil
}
