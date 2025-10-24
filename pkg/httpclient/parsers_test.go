package httpclient

import (
	"net/http"
	"testing"
	"time"
)

func TestParseOpenAIRateLimitHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected RateLimitInfo
	}{
		{
			name:     "empty_headers",
			headers:  map[string]string{},
			expected: RateLimitInfo{},
		},
		{
			name: "retry_after_seconds",
			headers: map[string]string{
				"Retry-After": "30",
			},
			expected: RateLimitInfo{
				RetryAfter: 30 * time.Second,
			},
		},
		{
			name: "retry_after_invalid",
			headers: map[string]string{
				"Retry-After": "invalid",
			},
			expected: RateLimitInfo{},
		},
		{
			name: "token_reset_time",
			headers: map[string]string{
				"x-ratelimit-reset-tokens": "1640995200",
			},
			expected: RateLimitInfo{
				ResetTime: 1640995200,
			},
		},
		{
			name: "request_reset_time",
			headers: map[string]string{
				"x-ratelimit-reset-requests": "1640995200",
			},
			expected: RateLimitInfo{
				ResetTime: 1640995200,
			},
		},
		{
			name: "token_reset_priority_over_request",
			headers: map[string]string{
				"x-ratelimit-reset-tokens":   "1640995200",
				"x-ratelimit-reset-requests": "1640995300",
			},
			expected: RateLimitInfo{
				ResetTime: 1640995200,
			},
		},
		{
			name: "reset_time_invalid",
			headers: map[string]string{
				"x-ratelimit-reset-tokens": "invalid",
			},
			expected: RateLimitInfo{},
		},
		{
			name: "remaining_requests",
			headers: map[string]string{
				"x-ratelimit-remaining-requests": "100",
			},
			expected: RateLimitInfo{
				RequestsRemaining: 100,
			},
		},
		{
			name: "remaining_tokens",
			headers: map[string]string{
				"x-ratelimit-remaining-tokens": "50000",
			},
			expected: RateLimitInfo{
				TokensRemaining: 50000,
			},
		},
		{
			name: "remaining_requests_invalid",
			headers: map[string]string{
				"x-ratelimit-remaining-requests": "invalid",
			},
			expected: RateLimitInfo{},
		},
		{
			name: "remaining_tokens_invalid",
			headers: map[string]string{
				"x-ratelimit-remaining-tokens": "invalid",
			},
			expected: RateLimitInfo{},
		},
		{
			name: "complete_openai_headers",
			headers: map[string]string{
				"Retry-After":                    "60",
				"x-ratelimit-reset-tokens":       "1640995200",
				"x-ratelimit-remaining-requests": "50",
				"x-ratelimit-remaining-tokens":   "25000",
			},
			expected: RateLimitInfo{
				RetryAfter:        60 * time.Second,
				ResetTime:         1640995200,
				RequestsRemaining: 50,
				TokensRemaining:   25000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			for key, value := range tt.headers {
				headers.Set(key, value)
			}

			result := ParseOpenAIRateLimitHeaders(headers)

			if result.RetryAfter != tt.expected.RetryAfter {
				t.Errorf("ParseOpenAIRateLimitHeaders() RetryAfter = %v, want %v", result.RetryAfter, tt.expected.RetryAfter)
			}
			if result.ResetTime != tt.expected.ResetTime {
				t.Errorf("ParseOpenAIRateLimitHeaders() ResetTime = %d, want %d", result.ResetTime, tt.expected.ResetTime)
			}
			if result.RequestsRemaining != tt.expected.RequestsRemaining {
				t.Errorf("ParseOpenAIRateLimitHeaders() RequestsRemaining = %d, want %d", result.RequestsRemaining, tt.expected.RequestsRemaining)
			}
			if result.TokensRemaining != tt.expected.TokensRemaining {
				t.Errorf("ParseOpenAIRateLimitHeaders() TokensRemaining = %d, want %d", result.TokensRemaining, tt.expected.TokensRemaining)
			}

			if result.InputTokensRemaining != tt.expected.InputTokensRemaining {
				t.Errorf("ParseOpenAIRateLimitHeaders() InputTokensRemaining = %d, want %d", result.InputTokensRemaining, tt.expected.InputTokensRemaining)
			}
			if result.OutputTokensRemaining != tt.expected.OutputTokensRemaining {
				t.Errorf("ParseOpenAIRateLimitHeaders() OutputTokensRemaining = %d, want %d", result.OutputTokensRemaining, tt.expected.OutputTokensRemaining)
			}
		})
	}
}

func TestParseAnthropicRateLimitHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected RateLimitInfo
	}{
		{
			name:     "empty_headers",
			headers:  map[string]string{},
			expected: RateLimitInfo{},
		},
		{
			name: "retry_after_seconds",
			headers: map[string]string{
				"retry-after": "45",
			},
			expected: RateLimitInfo{
				RetryAfter: 45 * time.Second,
			},
		},
		{
			name: "retry_after_invalid",
			headers: map[string]string{
				"retry-after": "invalid",
			},
			expected: RateLimitInfo{},
		},
		{
			name: "input_tokens_reset_rfc3339",
			headers: map[string]string{
				"anthropic-ratelimit-input-tokens-reset": "2021-12-31T23:59:59Z",
			},
			expected: RateLimitInfo{
				ResetTime: 1640995199,
			},
		},
		{
			name: "output_tokens_reset_rfc3339",
			headers: map[string]string{
				"anthropic-ratelimit-output-tokens-reset": "2021-12-31T23:59:59Z",
			},
			expected: RateLimitInfo{
				ResetTime: 1640995199,
			},
		},
		{
			name: "requests_reset_rfc3339",
			headers: map[string]string{
				"anthropic-ratelimit-requests-reset": "2021-12-31T23:59:59Z",
			},
			expected: RateLimitInfo{
				ResetTime: 1640995199,
			},
		},
		{
			name: "input_tokens_reset_priority",
			headers: map[string]string{
				"anthropic-ratelimit-input-tokens-reset":  "2021-12-31T23:59:59Z",
				"anthropic-ratelimit-output-tokens-reset": "2021-12-31T23:59:58Z",
				"anthropic-ratelimit-requests-reset":      "2021-12-31T23:59:57Z",
			},
			expected: RateLimitInfo{
				ResetTime: 1640995199,
			},
		},
		{
			name: "reset_time_invalid_rfc3339",
			headers: map[string]string{
				"anthropic-ratelimit-input-tokens-reset": "invalid-date",
			},
			expected: RateLimitInfo{},
		},
		{
			name: "remaining_requests",
			headers: map[string]string{
				"anthropic-ratelimit-requests-remaining": "75",
			},
			expected: RateLimitInfo{
				RequestsRemaining: 75,
			},
		},
		{
			name: "remaining_input_tokens",
			headers: map[string]string{
				"anthropic-ratelimit-input-tokens-remaining": "100000",
			},
			expected: RateLimitInfo{
				InputTokensRemaining: 100000,
			},
		},
		{
			name: "remaining_output_tokens",
			headers: map[string]string{
				"anthropic-ratelimit-output-tokens-remaining": "50000",
			},
			expected: RateLimitInfo{
				OutputTokensRemaining: 50000,
			},
		},
		{
			name: "remaining_requests_invalid",
			headers: map[string]string{
				"anthropic-ratelimit-requests-remaining": "invalid",
			},
			expected: RateLimitInfo{},
		},
		{
			name: "remaining_input_tokens_invalid",
			headers: map[string]string{
				"anthropic-ratelimit-input-tokens-remaining": "invalid",
			},
			expected: RateLimitInfo{},
		},
		{
			name: "remaining_output_tokens_invalid",
			headers: map[string]string{
				"anthropic-ratelimit-output-tokens-remaining": "invalid",
			},
			expected: RateLimitInfo{},
		},
		{
			name: "complete_anthropic_headers",
			headers: map[string]string{
				"retry-after":                                 "30",
				"anthropic-ratelimit-input-tokens-reset":      "2021-12-31T23:59:59Z",
				"anthropic-ratelimit-requests-remaining":      "25",
				"anthropic-ratelimit-input-tokens-remaining":  "75000",
				"anthropic-ratelimit-output-tokens-remaining": "25000",
			},
			expected: RateLimitInfo{
				RetryAfter:            30 * time.Second,
				ResetTime:             1640995199,
				RequestsRemaining:     25,
				InputTokensRemaining:  75000,
				OutputTokensRemaining: 25000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			for key, value := range tt.headers {
				headers.Set(key, value)
			}

			result := ParseAnthropicRateLimitHeaders(headers)

			if result.RetryAfter != tt.expected.RetryAfter {
				t.Errorf("ParseAnthropicRateLimitHeaders() RetryAfter = %v, want %v", result.RetryAfter, tt.expected.RetryAfter)
			}
			if result.ResetTime != tt.expected.ResetTime {
				t.Errorf("ParseAnthropicRateLimitHeaders() ResetTime = %d, want %d", result.ResetTime, tt.expected.ResetTime)
			}
			if result.RequestsRemaining != tt.expected.RequestsRemaining {
				t.Errorf("ParseAnthropicRateLimitHeaders() RequestsRemaining = %d, want %d", result.RequestsRemaining, tt.expected.RequestsRemaining)
			}
			if result.InputTokensRemaining != tt.expected.InputTokensRemaining {
				t.Errorf("ParseAnthropicRateLimitHeaders() InputTokensRemaining = %d, want %d", result.InputTokensRemaining, tt.expected.InputTokensRemaining)
			}
			if result.OutputTokensRemaining != tt.expected.OutputTokensRemaining {
				t.Errorf("ParseAnthropicRateLimitHeaders() OutputTokensRemaining = %d, want %d", result.OutputTokensRemaining, tt.expected.OutputTokensRemaining)
			}

			if result.TokensRemaining != tt.expected.TokensRemaining {
				t.Errorf("ParseAnthropicRateLimitHeaders() TokensRemaining = %d, want %d", result.TokensRemaining, tt.expected.TokensRemaining)
			}
		})
	}
}

func TestRateLimitHeaderParsers_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		parser  func(http.Header) RateLimitInfo
		headers map[string]string
	}{
		{
			name:   "openai_case_insensitive_headers",
			parser: ParseOpenAIRateLimitHeaders,
			headers: map[string]string{
				"retry-after":                    "30",
				"X-RATELIMIT-RESET-TOKENS":       "1640995200",
				"x-ratelimit-remaining-requests": "100",
			},
		},
		{
			name:   "anthropic_case_insensitive_headers",
			parser: ParseAnthropicRateLimitHeaders,
			headers: map[string]string{
				"RETRY-AFTER":                            "30",
				"anthropic-ratelimit-input-tokens-reset": "2021-12-31T23:59:59Z",
				"ANTHROPIC-RATELIMIT-REQUESTS-REMAINING": "100",
			},
		},
		{
			name:   "openai_multiple_values",
			parser: ParseOpenAIRateLimitHeaders,
			headers: map[string]string{
				"Retry-After": "30, 60",
			},
		},
		{
			name:   "anthropic_multiple_values",
			parser: ParseAnthropicRateLimitHeaders,
			headers: map[string]string{
				"retry-after": "45, 90",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			for key, value := range tt.headers {
				headers.Set(key, value)
			}

			result := tt.parser(headers)

			if result.RetryAfter < 0 {
				t.Errorf("Parser should not return negative RetryAfter: %v", result.RetryAfter)
			}
			if result.ResetTime < 0 {
				t.Errorf("Parser should not return negative ResetTime: %d", result.ResetTime)
			}
			if result.RequestsRemaining < 0 {
				t.Errorf("Parser should not return negative RequestsRemaining: %d", result.RequestsRemaining)
			}
		})
	}
}

func TestRateLimitHeaderParsers_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name     string
		parser   func(http.Header) RateLimitInfo
		headers  map[string]string
		validate func(t *testing.T, info RateLimitInfo)
	}{
		{
			name:   "openai_rate_limit_429",
			parser: ParseOpenAIRateLimitHeaders,
			headers: map[string]string{
				"Retry-After":                    "60",
				"x-ratelimit-reset-tokens":       "1640995200",
				"x-ratelimit-remaining-requests": "0",
				"x-ratelimit-remaining-tokens":   "0",
			},
			validate: func(t *testing.T, info RateLimitInfo) {
				if info.RetryAfter != 60*time.Second {
					t.Errorf("Expected RetryAfter=60s, got %v", info.RetryAfter)
				}
				if info.ResetTime != 1640995200 {
					t.Errorf("Expected ResetTime=1640995200, got %d", info.ResetTime)
				}
				if info.RequestsRemaining != 0 {
					t.Errorf("Expected RequestsRemaining=0, got %d", info.RequestsRemaining)
				}
				if info.TokensRemaining != 0 {
					t.Errorf("Expected TokensRemaining=0, got %d", info.TokensRemaining)
				}
			},
		},
		{
			name:   "anthropic_rate_limit_429",
			parser: ParseAnthropicRateLimitHeaders,
			headers: map[string]string{
				"retry-after":                                 "30",
				"anthropic-ratelimit-input-tokens-reset":      "2021-12-31T23:59:59Z",
				"anthropic-ratelimit-requests-remaining":      "0",
				"anthropic-ratelimit-input-tokens-remaining":  "0",
				"anthropic-ratelimit-output-tokens-remaining": "0",
			},
			validate: func(t *testing.T, info RateLimitInfo) {
				if info.RetryAfter != 30*time.Second {
					t.Errorf("Expected RetryAfter=30s, got %v", info.RetryAfter)
				}
				if info.ResetTime != 1640995199 {
					t.Errorf("Expected ResetTime=1640995199, got %d", info.ResetTime)
				}
				if info.RequestsRemaining != 0 {
					t.Errorf("Expected RequestsRemaining=0, got %d", info.RequestsRemaining)
				}
				if info.InputTokensRemaining != 0 {
					t.Errorf("Expected InputTokensRemaining=0, got %d", info.InputTokensRemaining)
				}
				if info.OutputTokensRemaining != 0 {
					t.Errorf("Expected OutputTokensRemaining=0, got %d", info.OutputTokensRemaining)
				}
			},
		},
		{
			name:   "openai_normal_operation",
			parser: ParseOpenAIRateLimitHeaders,
			headers: map[string]string{
				"x-ratelimit-reset-tokens":       "1640995200",
				"x-ratelimit-remaining-requests": "50",
				"x-ratelimit-remaining-tokens":   "100000",
			},
			validate: func(t *testing.T, info RateLimitInfo) {
				if info.RetryAfter != 0 {
					t.Errorf("Expected RetryAfter=0, got %v", info.RetryAfter)
				}
				if info.ResetTime != 1640995200 {
					t.Errorf("Expected ResetTime=1640995200, got %d", info.ResetTime)
				}
				if info.RequestsRemaining != 50 {
					t.Errorf("Expected RequestsRemaining=50, got %d", info.RequestsRemaining)
				}
				if info.TokensRemaining != 100000 {
					t.Errorf("Expected TokensRemaining=100000, got %d", info.TokensRemaining)
				}
			},
		},
		{
			name:   "anthropic_normal_operation",
			parser: ParseAnthropicRateLimitHeaders,
			headers: map[string]string{
				"anthropic-ratelimit-input-tokens-reset":      "2021-12-31T23:59:59Z",
				"anthropic-ratelimit-requests-remaining":      "25",
				"anthropic-ratelimit-input-tokens-remaining":  "50000",
				"anthropic-ratelimit-output-tokens-remaining": "25000",
			},
			validate: func(t *testing.T, info RateLimitInfo) {
				if info.RetryAfter != 0 {
					t.Errorf("Expected RetryAfter=0, got %v", info.RetryAfter)
				}
				if info.ResetTime != 1640995199 {
					t.Errorf("Expected ResetTime=1640995199, got %d", info.ResetTime)
				}
				if info.RequestsRemaining != 25 {
					t.Errorf("Expected RequestsRemaining=25, got %d", info.RequestsRemaining)
				}
				if info.InputTokensRemaining != 50000 {
					t.Errorf("Expected InputTokensRemaining=50000, got %d", info.InputTokensRemaining)
				}
				if info.OutputTokensRemaining != 25000 {
					t.Errorf("Expected OutputTokensRemaining=25000, got %d", info.OutputTokensRemaining)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			for key, value := range tt.headers {
				headers.Set(key, value)
			}

			result := tt.parser(headers)
			tt.validate(t, result)
		})
	}
}
