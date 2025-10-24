package httpclient

import (
	"fmt"
	"net/http"
	"time"
)

func ParseOpenAIRateLimitHeaders(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
		if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
			info.RetryAfter = seconds
		}
	}

	resetHeaders := []string{
		"x-ratelimit-reset-tokens",
		"x-ratelimit-reset-requests",
	}

	for _, header := range resetHeaders {
		if resetStr := headers.Get(header); resetStr != "" {
			var resetTime int64
			if _, err := fmt.Sscanf(resetStr, "%d", &resetTime); err == nil {
				info.ResetTime = resetTime
				break
			}
		}
	}

	if remaining := headers.Get("x-ratelimit-remaining-requests"); remaining != "" {
		_, _ = fmt.Sscanf(remaining, "%d", &info.RequestsRemaining)
	}
	if remaining := headers.Get("x-ratelimit-remaining-tokens"); remaining != "" {
		_, _ = fmt.Sscanf(remaining, "%d", &info.TokensRemaining)
	}

	return info
}

func ParseAnthropicRateLimitHeaders(headers http.Header) RateLimitInfo {
	info := RateLimitInfo{}

	if retryAfter := headers.Get("retry-after"); retryAfter != "" {
		if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
			info.RetryAfter = seconds
		}
	}

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
