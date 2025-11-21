package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"time"
)

type RetryStrategy int

const (
	NoRetry RetryStrategy = iota
	ConservativeRetry
	SmartRetry
)

type RateLimitInfo struct {
	RetryAfter            time.Duration
	ResetTime             int64
	RequestsRemaining     int
	InputTokensRemaining  int
	OutputTokensRemaining int
	TokensRemaining       int
}

type RateLimitHeaderParser func(http.Header) RateLimitInfo

type RetryStrategyFunc func(int) RetryStrategy

type Client struct {
	client       *http.Client
	maxRetries   int
	baseDelay    time.Duration
	headerParser RateLimitHeaderParser
	strategyFunc RetryStrategyFunc
}

type Option func(*Client)

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.client = client
	}
}

func WithMaxRetries(max int) Option {
	return func(c *Client) {
		c.maxRetries = max
	}
}

func WithBaseDelay(delay time.Duration) Option {
	return func(c *Client) {
		c.baseDelay = delay
	}
}

func WithHeaderParser(parser RateLimitHeaderParser) Option {
	return func(c *Client) {
		c.headerParser = parser
	}
}

func WithRetryStrategy(strategyFunc RetryStrategyFunc) Option {
	return func(c *Client) {
		c.strategyFunc = strategyFunc
	}
}

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

func DefaultRetryStrategy(statusCode int) RetryStrategy {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusServiceUnavailable:
		return SmartRetry
	case http.StatusRequestTimeout,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusGatewayTimeout:
		return ConservativeRetry
	default:
		return NoRetry
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	for attempt := 0; attempt <= c.maxRetries; attempt++ {

		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("failed to recreate request body for retry: %w", err)
			}
			req.Body = body
		}

		resp, strategy, retryInfo, err := c.attemptRequest(req)

		if strategy == NoRetry || err == nil {
			return resp, err
		}

		if attempt >= c.maxRetries {

			nextDelay := c.calculateDelay(strategy, attempt, retryInfo)

			retryErr := &RetryableError{
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("max HTTP retries (%d) exceeded", c.maxRetries),
				RetryAfter: nextDelay,
				Err:        err,
			}
			return resp, retryErr
		}

		delay := c.calculateDelay(strategy, attempt, retryInfo)

		if delay > 0 {
			c.logRetry(strategy, delay, attempt, resp)
			time.Sleep(delay)
		} else {

			return resp, err
		}
	}

	return nil, &RetryableError{
		StatusCode: 0,
		Message:    fmt.Sprintf("max retries exceeded after %d attempts", c.maxRetries),
		RetryAfter: c.baseDelay * 2,
		Err:        fmt.Errorf("max retries exceeded"),
	}
}

func (c *Client) attemptRequest(req *http.Request) (*http.Response, RetryStrategy, RateLimitInfo, error) {
	resp, err := c.client.Do(req)
	if err != nil {

		return nil, NoRetry, RateLimitInfo{}, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, NoRetry, RateLimitInfo{}, nil
	}

	var retryInfo RateLimitInfo
	if c.headerParser != nil {
		retryInfo = c.headerParser(resp.Header)
	}

	strategy := c.strategyFunc(resp.StatusCode)

	return resp, strategy, retryInfo, fmt.Errorf("HTTP %d", resp.StatusCode)
}

func (c *Client) calculateDelay(strategy RetryStrategy, attempt int, retryInfo RateLimitInfo) time.Duration {
	switch strategy {
	case SmartRetry:

		if retryInfo.RetryAfter > 0 {
			return retryInfo.RetryAfter
		}

		if retryInfo.ResetTime > 0 {
			delay := time.Until(time.Unix(retryInfo.ResetTime, 0))
			if delay > 0 {
				return delay
			}
		}

		exponentialDelay := time.Duration(math.Pow(2, float64(attempt))) * c.baseDelay
		jitter := time.Duration(float64(exponentialDelay) * 0.1)
		return exponentialDelay + jitter

	case ConservativeRetry:

		if attempt >= 2 {
			return 0
		}
		return time.Duration(2+attempt) * time.Second

	default:
		return 0
	}
}

func (c *Client) logRetry(strategy RetryStrategy, delay time.Duration, attempt int, resp *http.Response) {
	maxAttempts := c.maxRetries
	if strategy == ConservativeRetry {
		maxAttempts = 2
	}

	statusCode := 0
	var errorDetails string
	if resp != nil {
		statusCode = resp.StatusCode
		// Extract error message from response body for better debugging
		if resp.Body != nil {
			body, err := io.ReadAll(resp.Body)
			if err == nil && len(body) > 0 {
				// Restore body for later consumption
				resp.Body = io.NopCloser(bytes.NewReader(body))

				var errorResp struct {
					Error struct {
						Message string `json:"message"`
						Status  string `json:"status"`
						Code    int    `json:"code"`
					} `json:"error"`
				}
				if json.Unmarshal(body, &errorResp) == nil && errorResp.Error.Message != "" {
					errorDetails = fmt.Sprintf(" - %s (status: %s, code: %d)", 
						errorResp.Error.Message, errorResp.Error.Status, errorResp.Error.Code)
				} else {
					bodyStr := string(body)
					if len(bodyStr) > 200 {
						bodyStr = bodyStr[:200] + "..."
					}
					errorDetails = fmt.Sprintf(" - %s", bodyStr)
				}
			}
		}
	}

	switch strategy {
	case SmartRetry:
		slog.Info("Rate limited, retrying", "status_code", statusCode, "delay", delay, "attempt", attempt+1, "max_attempts", maxAttempts, "error_details", errorDetails)
	case ConservativeRetry:
		if attempt == maxAttempts-1 {
			slog.Warn("Server error, retrying", "status_code", statusCode, "delay", delay, "attempt", attempt+1, "max_attempts", maxAttempts, "error_details", errorDetails)
		}
	}
}
