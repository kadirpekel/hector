package httpclient

import (
	"fmt"
	"net/http"
	"time"
)

// ParseOpenAIRateLimitHeaders extracts OpenAI rate limit information
func ParseOpenAIRateLimitHeaders(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	// Retry-After (seconds)
	if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
		if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
			info.RetryAfter = seconds
		}
	}

	// Parse reset time (Unix timestamp in seconds)
	if resetStr := headers.Get("x-ratelimit-reset-requests"); resetStr != "" {
		var resetTime int64
		fmt.Sscanf(resetStr, "%d", &resetTime)
		info.ResetTime = resetTime
	} else if resetStr := headers.Get("x-ratelimit-reset-tokens"); resetStr != "" {
		var resetTime int64
		fmt.Sscanf(resetStr, "%d", &resetTime)
		info.ResetTime = resetTime
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
func ParseAnthropicRateLimitHeaders(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	// Retry-After (seconds)
	if retryAfter := headers.Get("retry-after"); retryAfter != "" {
		if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
			info.RetryAfter = seconds
		}
	}

	// Parse reset time (RFC 3339 format)
	if resetStr := headers.Get("anthropic-ratelimit-requests-reset"); resetStr != "" {
		if resetTime, err := time.Parse(time.RFC3339, resetStr); err == nil {
			info.ResetTime = resetTime.Unix()
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
