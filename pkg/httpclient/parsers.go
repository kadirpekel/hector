package httpclient

import (
	"fmt"
	"net/http"
	"time"
)

// ParseOpenAIRateLimitHeaders extracts OpenAI rate limit information
// See: https://platform.openai.com/docs/guides/rate-limits
func ParseOpenAIRateLimitHeaders(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	// Retry-After (seconds) - OpenAI may send this for 429 errors
	if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
		if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
			info.RetryAfter = seconds
		}
	}

	// Parse reset time (Unix timestamp in seconds)
	// OpenAI sends TWO different reset headers depending on which limit was hit.
	// Check tokens first (most common for LLM APIs), then requests
	resetHeaders := []string{
		"x-ratelimit-reset-tokens",   // Token limit (most common for LLM APIs)
		"x-ratelimit-reset-requests", // Request count limit
	}

	for _, header := range resetHeaders {
		if resetStr := headers.Get(header); resetStr != "" {
			var resetTime int64
			if _, err := fmt.Sscanf(resetStr, "%d", &resetTime); err == nil {
				info.ResetTime = resetTime
				break // Use first available reset time
			}
		}
	}

	// Parse remaining counts
	if remaining := headers.Get("x-ratelimit-remaining-requests"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &info.RequestsRemaining)
	}
	if remaining := headers.Get("x-ratelimit-remaining-tokens"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &info.TokensRemaining)
	}

	return info
}

// ParseAnthropicRateLimitHeaders extracts Anthropic rate limit information
// See: https://docs.claude.com/en/api/rate-limits
func ParseAnthropicRateLimitHeaders(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	// Retry-After (seconds) - Anthropic sends this for 429 errors
	if retryAfter := headers.Get("retry-after"); retryAfter != "" {
		if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
			info.RetryAfter = seconds
		}
	}

	// Parse reset time (RFC 3339 format)
	// Anthropic sends THREE different reset headers depending on which limit was hit.
	// Check all three in priority order (input tokens most common for LLM APIs)
	resetHeaders := []string{
		"anthropic-ratelimit-input-tokens-reset",  // Input token limit (most common)
		"anthropic-ratelimit-output-tokens-reset", // Output token limit
		"anthropic-ratelimit-requests-reset",      // Request count limit
	}

	for _, header := range resetHeaders {
		if resetStr := headers.Get(header); resetStr != "" {
			if resetTime, err := time.Parse(time.RFC3339, resetStr); err == nil {
				info.ResetTime = resetTime.Unix()
				break // Use first available reset time
			}
		}
	}

	// Parse remaining counts
	if remaining := headers.Get("anthropic-ratelimit-requests-remaining"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &info.RequestsRemaining)
	}
	if remaining := headers.Get("anthropic-ratelimit-input-tokens-remaining"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &info.InputTokensRemaining)
	}
	if remaining := headers.Get("anthropic-ratelimit-output-tokens-remaining"); remaining != "" {
		fmt.Sscanf(remaining, "%d", &info.OutputTokensRemaining)
	}

	return info
}
