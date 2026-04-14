package httpclient

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ParseAnthropicHeaders extracts rate limit info from Anthropic API headers.
func ParseAnthropicHeaders(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	// Retry-After header
	if retryAfter := headers.Get("retry-after"); retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			info.RetryAfter = time.Duration(seconds) * time.Second
		}
	}

	// Reset time headers (RFC3339 format)
	resetHeaders := []string{
		"anthropic-ratelimit-input-tokens-reset",
		"anthropic-ratelimit-output-tokens-reset",
		"anthropic-ratelimit-requests-reset",
	}
	for _, header := range resetHeaders {
		if resetStr := headers.Get(header); resetStr != "" {
			if resetTime, err := time.Parse(time.RFC3339, resetStr); err == nil {
				info.ResetTime = resetTime.Unix()
				break
			}
		}
	}

	// Remaining counters
	if remaining := headers.Get("anthropic-ratelimit-requests-remaining"); remaining != "" {
		_, _ = fmt.Sscanf(remaining, "%d", &info.RequestsRemaining)
	}
	if remaining := headers.Get("anthropic-ratelimit-input-tokens-remaining"); remaining != "" {
		_, _ = fmt.Sscanf(remaining, "%d", &info.InputTokensRemaining)
	}
	if remaining := headers.Get("anthropic-ratelimit-output-tokens-remaining"); remaining != "" {
		_, _ = fmt.Sscanf(remaining, "%d", &info.OutputTokensRemaining)
	}

	return info
}

// ParseOpenAIHeaders extracts rate limit info from OpenAI API headers.
func ParseOpenAIHeaders(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	// Retry-After header
	if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			info.RetryAfter = time.Duration(seconds) * time.Second
		}
	}

	// Reset time headers
	resetHeaders := []string{
		"x-ratelimit-reset-tokens",
		"x-ratelimit-reset-requests",
	}
	for _, header := range resetHeaders {
		if resetStr := headers.Get(header); resetStr != "" {
			if resetTime, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
				info.ResetTime = resetTime
				break
			}
		}
	}

	// Remaining counters
	if remaining := headers.Get("x-ratelimit-remaining-requests"); remaining != "" {
		_, _ = fmt.Sscanf(remaining, "%d", &info.RequestsRemaining)
	}
	if remaining := headers.Get("x-ratelimit-remaining-tokens"); remaining != "" {
		_, _ = fmt.Sscanf(remaining, "%d", &info.TokensRemaining)
	}

	return info
}

// ParseGeminiHeaders extracts rate limit info from Google Gemini API headers.
func ParseGeminiHeaders(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	// Retry-After header
	if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			info.RetryAfter = time.Duration(seconds) * time.Second
		}
	}

	return info
}
