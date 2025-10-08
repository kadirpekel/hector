// Package httpclient provides HTTP client utilities with retry logic.
package httpclient

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

// RetryStrategy represents the retry approach for different error types
type RetryStrategy int

const (
	NoRetry           RetryStrategy = iota
	ConservativeRetry               // Quick retry for server errors (max 2 attempts)
	SmartRetry                      // Header-driven retry for rate limits
)

// RateLimitInfo contains rate limit information from response headers
type RateLimitInfo struct {
	RetryAfter            time.Duration
	ResetTime             int64
	RequestsRemaining     int
	InputTokensRemaining  int
	OutputTokensRemaining int
	TokensRemaining       int // For OpenAI (combined tokens)
}

// RateLimitHeaderParser extracts rate limit info from HTTP headers
type RateLimitHeaderParser func(http.Header) RateLimitInfo

// RetryStrategyFunc determines retry strategy based on status code
type RetryStrategyFunc func(int) RetryStrategy

// Client is a generic HTTP client with smart retry logic
type Client struct {
	client       *http.Client
	maxRetries   int
	baseDelay    time.Duration
	headerParser RateLimitHeaderParser
	strategyFunc RetryStrategyFunc
}

// Option configures the HTTP client
type Option func(*Client)

// WithHTTPClient sets a custom http.Client
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.client = client
	}
}

// WithMaxRetries sets the maximum number of retry attempts
func WithMaxRetries(max int) Option {
	return func(c *Client) {
		c.maxRetries = max
	}
}

// WithBaseDelay sets the base delay for exponential backoff
func WithBaseDelay(delay time.Duration) Option {
	return func(c *Client) {
		c.baseDelay = delay
	}
}

// WithHeaderParser sets a custom rate limit header parser
func WithHeaderParser(parser RateLimitHeaderParser) Option {
	return func(c *Client) {
		c.headerParser = parser
	}
}

// WithRetryStrategy sets a custom retry strategy function
func WithRetryStrategy(strategyFunc RetryStrategyFunc) Option {
	return func(c *Client) {
		c.strategyFunc = strategyFunc
	}
}

// New creates a new HTTP client with smart retry logic
func New(opts ...Option) *Client {
	client := &Client{
		client:       &http.Client{Timeout: 60 * time.Second},
		maxRetries:   5,
		baseDelay:    2 * time.Second,
		strategyFunc: DefaultRetryStrategy,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// DefaultRetryStrategy is the default three-tier retry strategy
func DefaultRetryStrategy(statusCode int) RetryStrategy {
	switch statusCode {
	case http.StatusTooManyRequests, // 429 - Rate limit with headers
		http.StatusServiceUnavailable: // 503 - May have Retry-After
		return SmartRetry
	case http.StatusRequestTimeout, // 408 - Network timeout
		http.StatusInternalServerError, // 500 - Server error
		http.StatusBadGateway,          // 502 - Gateway issue
		http.StatusGatewayTimeout:      // 504 - Gateway timeout
		return ConservativeRetry
	default:
		return NoRetry
	}
}

// Do executes an HTTP request with smart retry logic
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		resp, strategy, retryInfo, err := c.attemptRequest(req)

		// No retry - return immediately (success or non-retryable error)
		if strategy == NoRetry || err == nil {
			return resp, err
		}

		// Exhausted retries - return typed error with retry information
		if attempt >= c.maxRetries {
			// Calculate next retry delay for agent-level retry
			nextDelay := c.calculateDelay(strategy, attempt, retryInfo)

			retryErr := &RetryableError{
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("max HTTP retries (%d) exceeded", c.maxRetries),
				RetryAfter: nextDelay,
				Err:        err,
			}
			return resp, retryErr
		}

		// Determine retry delay based on strategy
		delay := c.calculateDelay(strategy, attempt, retryInfo)

		// Log retry (if delay > 0)
		if delay > 0 {
			c.logRetry(strategy, delay, attempt, resp)
			time.Sleep(delay)
		} else {
			// Conservative retry limit exceeded
			return resp, err
		}
	}

	// Should not reach here, but if we do, return generic error
	return nil, &RetryableError{
		StatusCode: 0,
		Message:    fmt.Sprintf("max retries exceeded after %d attempts", c.maxRetries),
		RetryAfter: c.baseDelay * 2, // Suggest waiting
		Err:        fmt.Errorf("max retries exceeded"),
	}
}

// attemptRequest makes a single HTTP request attempt
func (c *Client) attemptRequest(req *http.Request) (*http.Response, RetryStrategy, RateLimitInfo, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		// Network error - no retry (connection issues)
		return nil, NoRetry, RateLimitInfo{}, err
	}

	// Success - no retry needed
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, NoRetry, RateLimitInfo{}, nil
	}

	// Extract rate limit info if parser provided
	var retryInfo RateLimitInfo
	if c.headerParser != nil {
		retryInfo = c.headerParser(resp.Header)
	}

	// Determine retry strategy
	strategy := c.strategyFunc(resp.StatusCode)

	return resp, strategy, retryInfo, fmt.Errorf("HTTP %d", resp.StatusCode)
}

// calculateDelay determines the retry delay based on strategy
func (c *Client) calculateDelay(strategy RetryStrategy, attempt int, retryInfo RateLimitInfo) time.Duration {
	switch strategy {
	case SmartRetry:
		// Priority 1: Retry-After header
		if retryInfo.RetryAfter > 0 {
			return retryInfo.RetryAfter
		}

		// Priority 2: Reset timestamp
		if retryInfo.ResetTime > 0 {
			delay := time.Until(time.Unix(retryInfo.ResetTime, 0))
			if delay > 0 {
				return delay
			}
		}

		// Priority 3: Exponential backoff with jitter
		exponentialDelay := time.Duration(math.Pow(2, float64(attempt))) * c.baseDelay
		jitter := time.Duration(float64(exponentialDelay) * 0.1)
		return exponentialDelay + jitter

	case ConservativeRetry:
		// Max 2 attempts with short delays
		if attempt >= 2 {
			return 0 // Signal to stop retrying
		}
		return time.Duration(2+attempt) * time.Second // 2s, 3s

	default:
		return 0
	}
}

// logRetry logs retry information
func (c *Client) logRetry(strategy RetryStrategy, delay time.Duration, attempt int, resp *http.Response) {
	maxAttempts := c.maxRetries
	if strategy == ConservativeRetry {
		maxAttempts = 2
	}

	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}

	switch strategy {
	case SmartRetry:
		fmt.Printf("⏳ Rate limited (HTTP %d). Retrying in %v (attempt %d/%d)\n",
			statusCode, delay, attempt+1, maxAttempts)
	case ConservativeRetry:
		fmt.Printf("⚠️  Server error (HTTP %d). Quick retry in %v (attempt %d/%d)\n",
			statusCode, delay, attempt+1, maxAttempts)
	}
}
