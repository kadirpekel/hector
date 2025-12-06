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

// Package httpclient provides an HTTP client with retry, backoff, and rate limit handling.
//
// Features:
//   - Automatic retry with exponential backoff
//   - Rate limit header parsing (Anthropic, OpenAI)
//   - Smart retry based on status codes
//   - Configurable retry strategies
package httpclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"
)

// RetryStrategy defines how to handle retries.
type RetryStrategy int

const (
	// NoRetry indicates no retry should be attempted.
	NoRetry RetryStrategy = iota

	// ConservativeRetry attempts up to 2 retries with fixed delays.
	ConservativeRetry

	// SmartRetry uses rate limit headers and exponential backoff.
	SmartRetry
)

// RateLimitInfo contains rate limit information from response headers.
type RateLimitInfo struct {
	RetryAfter            time.Duration
	ResetTime             int64
	RequestsRemaining     int
	InputTokensRemaining  int
	OutputTokensRemaining int
	TokensRemaining       int
}

// HeaderParser extracts rate limit info from response headers.
type HeaderParser func(http.Header) RateLimitInfo

// StrategyFunc determines the retry strategy based on status code.
type StrategyFunc func(int) RetryStrategy

// Client wraps http.Client with retry and backoff capabilities.
type Client struct {
	client       *http.Client
	maxRetries   int
	baseDelay    time.Duration
	maxDelay     time.Duration
	headerParser HeaderParser
	strategyFunc StrategyFunc
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom http.Client.
//
// IMPORTANT: Order matters when using with WithTLSConfig:
//
//   - ✅ CORRECT: Call WithHTTPClient FIRST, then WithTLSConfig
//     Example:
//     httpclient.New(
//     httpclient.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
//     httpclient.WithTLSConfig(&httpclient.TLSConfig{CACertificate: "/path/to/ca.pem"}),
//     )
//
//   - ❌ WRONG: Calling WithTLSConfig before WithHTTPClient will lose TLS configuration
//
//   - ✅ BEST: For custom transport settings, configure TLS on the transport first:
//     Example:
//     tlsTransport, _ := httpclient.ConfigureTLS(&httpclient.TLSConfig{CACertificate: "/path/to/ca.pem"})
//     tlsTransport.MaxIdleConns = 100  // Custom settings
//     httpclient.New(
//     httpclient.WithHTTPClient(&http.Client{Transport: tlsTransport, Timeout: 30 * time.Second}),
//     )
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		// If TLS transport was already configured, try to preserve it
		if c.client != nil && c.client.Transport != nil {
			if existingTransport, ok := c.client.Transport.(*http.Transport); ok {
				if existingTransport.TLSClientConfig != nil {
					// TLS was configured, merge it into the new client's transport
					if client.Transport == nil {
						// New client has no transport, create one with TLS config
						client.Transport = &http.Transport{
							TLSClientConfig: &tls.Config{},
						}
					}
					if newTransport, ok := client.Transport.(*http.Transport); ok {
						// Copy TLS configuration from existing transport
						if newTransport.TLSClientConfig == nil {
							newTransport.TLSClientConfig = &tls.Config{}
						}
						newTransport.TLSClientConfig.RootCAs = existingTransport.TLSClientConfig.RootCAs
						newTransport.TLSClientConfig.InsecureSkipVerify = existingTransport.TLSClientConfig.InsecureSkipVerify
						slog.Debug("Preserved TLS configuration when setting custom HTTP client")
					}
				}
			}
		}
		c.client = client
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(max int) Option {
	return func(c *Client) {
		c.maxRetries = max
	}
}

// WithBaseDelay sets the base delay for exponential backoff.
func WithBaseDelay(delay time.Duration) Option {
	return func(c *Client) {
		c.baseDelay = delay
	}
}

// WithMaxDelay sets the maximum delay between retries.
func WithMaxDelay(delay time.Duration) Option {
	return func(c *Client) {
		c.maxDelay = delay
	}
}

// WithHeaderParser sets a custom rate limit header parser.
func WithHeaderParser(parser HeaderParser) Option {
	return func(c *Client) {
		c.headerParser = parser
	}
}

// WithRetryStrategy sets a custom retry strategy function.
func WithRetryStrategy(strategyFunc StrategyFunc) Option {
	return func(c *Client) {
		c.strategyFunc = strategyFunc
	}
}

// TLSConfig holds TLS configuration options for outbound HTTP requests.
// This is useful for corporate networks with custom CA certificates or
// development environments with self-signed certificates.
type TLSConfig struct {
	// InsecureSkipVerify disables TLS certificate verification.
	// WARNING: Only use for development/testing. Never use in production.
	InsecureSkipVerify bool

	// CACertificate is the path to a custom CA certificate file.
	// Use this for corporate proxies or internal services with custom certificates.
	CACertificate string
}

// ConfigureTLS creates an http.Transport with TLS configuration.
func ConfigureTLS(config *TLSConfig) (*http.Transport, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}

	if config == nil {
		return transport, nil
	}

	// Handle custom CA certificate
	if config.CACertificate != "" {
		caCert, err := os.ReadFile(config.CACertificate)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate from %s: %w", config.CACertificate, err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate from %s", config.CACertificate)
		}

		transport.TLSClientConfig.RootCAs = caCertPool
	}

	// Handle insecure skip verify (dev/test only)
	if config.InsecureSkipVerify {
		transport.TLSClientConfig.InsecureSkipVerify = true
		slog.Warn("TLS certificate verification disabled - NOT for production use")
	}

	return transport, nil
}

// WithTLSConfig sets TLS configuration for the HTTP client.
// This is useful for:
//   - Corporate networks with custom CA certificates
//   - Internal services with self-signed certificates
//   - Development/testing environments (with InsecureSkipVerify)
//
// NOTE: Call WithTLSConfig AFTER WithHTTPClient if both are used.
// If called before WithHTTPClient, the TLS transport will be overwritten.
func WithTLSConfig(config *TLSConfig) Option {
	return func(c *Client) {
		if config == nil {
			return
		}

		transport, err := ConfigureTLS(config)
		if err != nil {
			// Log warning but don't fail - use default transport
			slog.Warn("Failed to configure TLS", "error", err)
			return
		}

		// Update the HTTP client's transport
		// Preserve existing timeout if client already exists
		if c.client != nil {
			timeout := c.client.Timeout
			c.client.Transport = transport
			c.client.Timeout = timeout // Preserve timeout
		} else {
			// Create new client with transport and default timeout
			c.client = &http.Client{
				Transport: transport,
				Timeout:   120 * time.Second, // Default timeout matches New()
			}
		}
	}
}

// New creates a new Client with the given options.
func New(opts ...Option) *Client {
	c := &Client{
		client:       &http.Client{Timeout: 120 * time.Second},
		maxRetries:   5,
		baseDelay:    2 * time.Second,
		maxDelay:     60 * time.Second,
		strategyFunc: DefaultStrategy,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// DefaultStrategy returns the default retry strategy for a status code.
func DefaultStrategy(statusCode int) RetryStrategy {
	switch statusCode {
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return SmartRetry
	case http.StatusRequestTimeout, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusGatewayTimeout:
		return ConservativeRetry
	default:
		return NoRetry
	}
}

// Do executes the request with retry logic.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// Ensure request body can be replayed
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Reset body for retry
		if attempt > 0 && bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, strategy, retryInfo, err := c.attemptRequest(req)

		// Success or non-retryable error
		if strategy == NoRetry || err == nil {
			return resp, err
		}

		// Max retries exceeded
		if attempt >= c.maxRetries {
			return resp, &RetryableError{
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("max retries (%d) exceeded", c.maxRetries),
				RetryAfter: c.calculateDelay(strategy, attempt, retryInfo),
				Err:        err,
			}
		}

		// Calculate delay
		delay := c.calculateDelay(strategy, attempt, retryInfo)
		if delay <= 0 {
			return resp, err
		}

		// Log and wait
		c.logRetry(strategy, delay, attempt, resp)
		time.Sleep(delay)
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

func (c *Client) calculateDelay(strategy RetryStrategy, attempt int, info RateLimitInfo) time.Duration {
	switch strategy {
	case SmartRetry:
		// Use Retry-After if provided
		if info.RetryAfter > 0 {
			return info.RetryAfter
		}

		// Use reset time if provided
		if info.ResetTime > 0 {
			delay := time.Until(time.Unix(info.ResetTime, 0))
			if delay > 0 {
				return min(delay, c.maxDelay)
			}
		}

		// Exponential backoff with jitter
		delay := time.Duration(math.Pow(2, float64(attempt))) * c.baseDelay
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)
		return min(delay+jitter, c.maxDelay)

	case ConservativeRetry:
		// Limited retries with fixed delays
		if attempt >= 2 {
			return 0 // Stop retrying
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
		errorDetails = extractErrorDetails(resp)
	}

	switch strategy {
	case SmartRetry:
		slog.Info("Rate limited, retrying",
			"status", statusCode,
			"delay", delay,
			"attempt", attempt+1,
			"max", maxAttempts,
			"details", errorDetails)
	case ConservativeRetry:
		if attempt == maxAttempts-1 {
			slog.Warn("Server error, retrying",
				"status", statusCode,
				"delay", delay,
				"attempt", attempt+1,
				"max", maxAttempts,
				"details", errorDetails)
		}
	}
}

func extractErrorDetails(resp *http.Response) string {
	if resp.Body == nil {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		return ""
	}

	// Restore body for later consumption
	resp.Body = io.NopCloser(bytes.NewReader(body))

	// Try to parse as JSON error
	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &errorResp) == nil && errorResp.Error.Message != "" {
		return errorResp.Error.Message
	}

	// Truncate raw body
	s := string(body)
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}

// RetryableError represents an error that may be retried.
type RetryableError struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration
	Err        error
}

func (e *RetryableError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("HTTP %d: %s (retry after %v)", e.StatusCode, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true.
func (e *RetryableError) IsRetryable() bool {
	return true
}
