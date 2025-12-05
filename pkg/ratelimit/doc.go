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

// Package ratelimit provides a comprehensive rate limiting system for Hector v2.
//
// Features:
//   - Multi-layer time windows (minute, hour, day, week, month)
//   - Dual tracking (token count AND request count)
//   - Flexible scopes (per-session or per-user)
//   - Multiple storage backends (in-memory and SQL)
//   - Atomic check-and-record operations
//   - Detailed usage statistics
//
// # Basic Usage
//
//	// Create store (memory or SQL)
//	store := ratelimit.NewMemoryStore()
//
//	// Create limiter with config
//	limiter, err := ratelimit.NewRateLimiter(config, store)
//
//	// Check and record usage
//	result, err := limiter.CheckAndRecord(ctx, ratelimit.ScopeSession, "session-123", 1000, 1)
//	if !result.Allowed {
//	    // Handle rate limit exceeded
//	}
//
// # Configuration
//
//	rate_limiting:
//	  enabled: true
//	  scope: "session"  # or "user"
//	  backend: "memory"  # or "sql"
//	  limits:
//	    - type: token
//	      window: day
//	      limit: 100000
//	    - type: count
//	      window: minute
//	      limit: 60
//
// # Time Windows
//
//   - minute: 60 seconds (burst protection)
//   - hour: 60 minutes (short-term limits)
//   - day: 24 hours (daily quotas)
//   - week: 7 days (weekly budgets)
//   - month: 30 days (monthly billing)
//
// # Limit Types
//
//   - token: Track token usage (LLM API tokens, cost control)
//   - count: Track request count (rate throttling, DDoS protection)
//
// # Scopes
//
//   - session: Each session has independent quotas
//   - user: All sessions for a user share quotas
package ratelimit
