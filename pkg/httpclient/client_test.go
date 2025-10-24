package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		validate func(t *testing.T, client *Client)
	}{
		{
			name:    "default_configuration",
			options: []Option{},
			validate: func(t *testing.T, client *Client) {
				if client.maxRetries != 5 {
					t.Errorf("Expected maxRetries=5, got %d", client.maxRetries)
				}
				if client.baseDelay != 2*time.Second {
					t.Errorf("Expected baseDelay=2s, got %v", client.baseDelay)
				}
				if client.client.Timeout != 60*time.Second {
					t.Errorf("Expected timeout=60s, got %v", client.client.Timeout)
				}
				if client.strategyFunc == nil {
					t.Error("Expected strategyFunc to be set")
				}
			},
		},
		{
			name: "custom_max_retries",
			options: []Option{
				WithMaxRetries(3),
			},
			validate: func(t *testing.T, client *Client) {
				if client.maxRetries != 3 {
					t.Errorf("Expected maxRetries=3, got %d", client.maxRetries)
				}
			},
		},
		{
			name: "custom_base_delay",
			options: []Option{
				WithBaseDelay(5 * time.Second),
			},
			validate: func(t *testing.T, client *Client) {
				if client.baseDelay != 5*time.Second {
					t.Errorf("Expected baseDelay=5s, got %v", client.baseDelay)
				}
			},
		},
		{
			name: "custom_http_client",
			options: []Option{
				WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
			},
			validate: func(t *testing.T, client *Client) {
				if client.client.Timeout != 30*time.Second {
					t.Errorf("Expected timeout=30s, got %v", client.client.Timeout)
				}
			},
		},
		{
			name: "custom_header_parser",
			options: []Option{
				WithHeaderParser(func(h http.Header) RateLimitInfo {
					return RateLimitInfo{RetryAfter: 10 * time.Second}
				}),
			},
			validate: func(t *testing.T, client *Client) {
				if client.headerParser == nil {
					t.Error("Expected headerParser to be set")
				}

				headers := http.Header{}
				info := client.headerParser(headers)
				if info.RetryAfter != 10*time.Second {
					t.Errorf("Expected RetryAfter=10s, got %v", info.RetryAfter)
				}
			},
		},
		{
			name: "custom_retry_strategy",
			options: []Option{
				WithRetryStrategy(func(statusCode int) RetryStrategy {
					return SmartRetry
				}),
			},
			validate: func(t *testing.T, client *Client) {
				if client.strategyFunc == nil {
					t.Error("Expected strategyFunc to be set")
				}

				strategy := client.strategyFunc(500)
				if strategy != SmartRetry {
					t.Errorf("Expected SmartRetry, got %v", strategy)
				}
			},
		},
		{
			name: "multiple_options",
			options: []Option{
				WithMaxRetries(2),
				WithBaseDelay(1 * time.Second),
				WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
			},
			validate: func(t *testing.T, client *Client) {
				if client.maxRetries != 2 {
					t.Errorf("Expected maxRetries=2, got %d", client.maxRetries)
				}
				if client.baseDelay != 1*time.Second {
					t.Errorf("Expected baseDelay=1s, got %v", client.baseDelay)
				}
				if client.client.Timeout != 10*time.Second {
					t.Errorf("Expected timeout=10s, got %v", client.client.Timeout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(tt.options...)
			tt.validate(t, client)
		})
	}
}

func TestDefaultRetryStrategy(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   RetryStrategy
	}{
		{
			name:       "rate_limit_429",
			statusCode: http.StatusTooManyRequests,
			expected:   SmartRetry,
		},
		{
			name:       "service_unavailable_503",
			statusCode: http.StatusServiceUnavailable,
			expected:   SmartRetry,
		},
		{
			name:       "request_timeout_408",
			statusCode: http.StatusRequestTimeout,
			expected:   ConservativeRetry,
		},
		{
			name:       "internal_server_error_500",
			statusCode: http.StatusInternalServerError,
			expected:   ConservativeRetry,
		},
		{
			name:       "bad_gateway_502",
			statusCode: http.StatusBadGateway,
			expected:   ConservativeRetry,
		},
		{
			name:       "gateway_timeout_504",
			statusCode: http.StatusGatewayTimeout,
			expected:   ConservativeRetry,
		},
		{
			name:       "success_200",
			statusCode: http.StatusOK,
			expected:   NoRetry,
		},
		{
			name:       "not_found_404",
			statusCode: http.StatusNotFound,
			expected:   NoRetry,
		},
		{
			name:       "bad_request_400",
			statusCode: http.StatusBadRequest,
			expected:   NoRetry,
		},
		{
			name:       "unauthorized_401",
			statusCode: http.StatusUnauthorized,
			expected:   NoRetry,
		},
		{
			name:       "forbidden_403",
			statusCode: http.StatusForbidden,
			expected:   NoRetry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultRetryStrategy(tt.statusCode)
			if result != tt.expected {
				t.Errorf("DefaultRetryStrategy(%d) = %v, want %v", tt.statusCode, result, tt.expected)
			}
		})
	}
}

func TestClient_Do_Success(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))
	defer server.Close()

	client := New(WithHTTPClient(server.Client()))
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	if resp == nil {
		t.Fatal("Do() response = nil, want non-nil")
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Do() status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestClient_Do_NetworkError(t *testing.T) {

	client := New(WithHTTPClient(&http.Client{Timeout: 1 * time.Millisecond}))
	req, _ := http.NewRequest("GET", "http://invalid-url-that-does-not-exist:9999", nil)

	resp, err := client.Do(req)
	if err == nil {
		t.Error("Do() error = nil, want network error")
	}
	if resp != nil {
		t.Error("Do() response should be nil for network errors")
	}
}

func TestClient_Do_RetryableError(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success after retry"))
		}
	}))
	defer server.Close()

	client := New(
		WithHTTPClient(server.Client()),
		WithMaxRetries(3),
		WithBaseDelay(10*time.Millisecond),
	)
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Do() status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestClient_Do_MaxRetriesExceeded(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(
		WithHTTPClient(server.Client()),
		WithMaxRetries(2),
		WithBaseDelay(10*time.Millisecond),
	)
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)
	if err == nil {
		t.Error("Do() error = nil, want RetryableError")
	}
	if resp == nil {
		t.Error("Do() response = nil, want non-nil")
	} else if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Do() status code = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	retryErr, ok := err.(*RetryableError)
	if !ok {
		t.Errorf("Do() error type = %T, want *RetryableError", err)
	} else {
		if retryErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("RetryableError.StatusCode = %d, want %d", retryErr.StatusCode, http.StatusInternalServerError)
		}

		if retryErr.RetryAfter < 0 {
			t.Error("RetryableError.RetryAfter should be >= 0")
		}
	}

	expectedAttempts := 2 + 1
	if attemptCount != expectedAttempts {
		t.Errorf("Expected %d attempts, got %d", expectedAttempts, attemptCount)
	}
}

func TestClient_Do_RateLimitWithRetryAfter(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success after rate limit"))
		}
	}))
	defer server.Close()

	client := New(
		WithHTTPClient(server.Client()),
		WithMaxRetries(3),
		WithHeaderParser(ParseOpenAIRateLimitHeaders),
	)
	req, _ := http.NewRequest("GET", server.URL, nil)

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Do() error = %v, want nil", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Do() status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if attemptCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", attemptCount)
	}

	if duration < 1*time.Second {
		t.Errorf("Expected to wait at least 1s, waited %v", duration)
	}
}

func TestClient_Do_ConservativeRetryLimit(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(
		WithHTTPClient(server.Client()),
		WithMaxRetries(5),
		WithBaseDelay(10*time.Millisecond),
	)
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, err := client.Do(req)
	if err == nil {
		t.Error("Do() error = nil, want error")
	}
	if resp == nil {
		t.Error("Do() response = nil, want non-nil")
	}

	expectedAttempts := 2 + 1
	if attemptCount != expectedAttempts {
		t.Errorf("Expected %d attempts for conservative retry, got %d", expectedAttempts, attemptCount)
	}
}

func TestClient_attemptRequest(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedErr    bool
		expectedCode   int
		expectedStrat  RetryStrategy
	}{
		{
			name: "success_response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectedErr:   false,
			expectedCode:  http.StatusOK,
			expectedStrat: NoRetry,
		},
		{
			name: "rate_limit_response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			},
			expectedErr:   true,
			expectedCode:  http.StatusTooManyRequests,
			expectedStrat: SmartRetry,
		},
		{
			name: "server_error_response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedErr:   true,
			expectedCode:  http.StatusInternalServerError,
			expectedStrat: ConservativeRetry,
		},
		{
			name: "client_error_response",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			expectedErr:   true,
			expectedCode:  http.StatusBadRequest,
			expectedStrat: NoRetry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := New(WithHTTPClient(server.Client()))
			req, _ := http.NewRequest("GET", server.URL, nil)

			resp, strategy, retryInfo, err := client.attemptRequest(req)

			if (err != nil) != tt.expectedErr {
				t.Errorf("attemptRequest() error = %v, wantErr %v", err, tt.expectedErr)
			}
			if resp.StatusCode != tt.expectedCode {
				t.Errorf("attemptRequest() status code = %d, want %d", resp.StatusCode, tt.expectedCode)
			}
			if strategy != tt.expectedStrat {
				t.Errorf("attemptRequest() strategy = %v, want %v", strategy, tt.expectedStrat)
			}

			if retryInfo.RetryAfter != 0 || retryInfo.ResetTime != 0 {
				t.Errorf("attemptRequest() retryInfo should be empty, got %+v", retryInfo)
			}
		})
	}
}

func TestClient_calculateDelay(t *testing.T) {
	client := New(WithBaseDelay(1 * time.Second))

	tests := []struct {
		name      string
		strategy  RetryStrategy
		attempt   int
		retryInfo RateLimitInfo
		expected  time.Duration
	}{
		{
			name:     "no_retry",
			strategy: NoRetry,
			attempt:  0,
			expected: 0,
		},
		{
			name:     "smart_retry_exponential_backoff",
			strategy: SmartRetry,
			attempt:  0,
			expected: 1*time.Second + 100*time.Millisecond,
		},
		{
			name:     "smart_retry_exponential_backoff_attempt_1",
			strategy: SmartRetry,
			attempt:  1,
			expected: 2*time.Second + 200*time.Millisecond,
		},
		{
			name:     "smart_retry_with_retry_after",
			strategy: SmartRetry,
			attempt:  0,
			retryInfo: RateLimitInfo{
				RetryAfter: 5 * time.Second,
			},
			expected: 5 * time.Second,
		},
		{
			name:     "smart_retry_with_reset_time",
			strategy: SmartRetry,
			attempt:  0,
			retryInfo: RateLimitInfo{
				ResetTime: time.Now().Add(3 * time.Second).Unix(),
			},
			expected: 3 * time.Second,
		},
		{
			name:     "conservative_retry_attempt_0",
			strategy: ConservativeRetry,
			attempt:  0,
			expected: 2 * time.Second,
		},
		{
			name:     "conservative_retry_attempt_1",
			strategy: ConservativeRetry,
			attempt:  1,
			expected: 3 * time.Second,
		},
		{
			name:     "conservative_retry_attempt_2",
			strategy: ConservativeRetry,
			attempt:  2,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.calculateDelay(tt.strategy, tt.attempt, tt.retryInfo)

			if tt.name == "smart_retry_with_reset_time" {
				if result < 2*time.Second || result > 4*time.Second {
					t.Errorf("calculateDelay() = %v, want approximately 3s", result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("calculateDelay() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}
