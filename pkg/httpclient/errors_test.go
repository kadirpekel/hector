package httpclient

import (
	"errors"
	"testing"
	"time"
)

func TestRetryableError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *RetryableError
		expected string
	}{
		{
			name: "error_with_retry_after",
			err: &RetryableError{
				StatusCode: 429,
				Message:    "Rate limit exceeded",
				RetryAfter: 30 * time.Second,
				Err:        errors.New("underlying error"),
			},
			expected: "HTTP 429: Rate limit exceeded (retry after 30s)",
		},
		{
			name: "error_without_retry_after",
			err: &RetryableError{
				StatusCode: 500,
				Message:    "Internal server error",
				RetryAfter: 0,
				Err:        errors.New("underlying error"),
			},
			expected: "HTTP 500: Internal server error",
		},
		{
			name: "error_with_zero_retry_after",
			err: &RetryableError{
				StatusCode: 503,
				Message:    "Service unavailable",
				RetryAfter: 0,
				Err:        errors.New("underlying error"),
			},
			expected: "HTTP 503: Service unavailable",
		},
		{
			name: "error_with_millisecond_retry_after",
			err: &RetryableError{
				StatusCode: 429,
				Message:    "Rate limit exceeded",
				RetryAfter: 1500 * time.Millisecond,
				Err:        errors.New("underlying error"),
			},
			expected: "HTTP 429: Rate limit exceeded (retry after 1.5s)",
		},
		{
			name: "error_with_minute_retry_after",
			err: &RetryableError{
				StatusCode: 429,
				Message:    "Rate limit exceeded",
				RetryAfter: 2 * time.Minute,
				Err:        errors.New("underlying error"),
			},
			expected: "HTTP 429: Rate limit exceeded (retry after 2m0s)",
		},
		{
			name: "error_with_empty_message",
			err: &RetryableError{
				StatusCode: 500,
				Message:    "",
				RetryAfter: 10 * time.Second,
				Err:        errors.New("underlying error"),
			},
			expected: "HTTP 500:  (retry after 10s)",
		},
		{
			name: "error_with_zero_status_code",
			err: &RetryableError{
				StatusCode: 0,
				Message:    "Unknown error",
				RetryAfter: 5 * time.Second,
				Err:        errors.New("underlying error"),
			},
			expected: "HTTP 0: Unknown error (retry after 5s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("RetryableError.Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRetryableError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	retryErr := &RetryableError{
		StatusCode: 429,
		Message:    "Rate limit exceeded",
		RetryAfter: 30 * time.Second,
		Err:        underlyingErr,
	}

	result := retryErr.Unwrap()
	if result != underlyingErr {
		t.Errorf("RetryableError.Unwrap() = %v, want %v", result, underlyingErr)
	}
}

func TestRetryableError_Unwrap_Nil(t *testing.T) {
	retryErr := &RetryableError{
		StatusCode: 500,
		Message:    "Internal server error",
		RetryAfter: 0,
		Err:        nil,
	}

	result := retryErr.Unwrap()
	if result != nil {
		t.Errorf("RetryableError.Unwrap() = %v, want nil", result)
	}
}

func TestRetryableError_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      *RetryableError
		expected bool
	}{
		{
			name: "retryable_error_with_retry_after",
			err: &RetryableError{
				StatusCode: 429,
				Message:    "Rate limit exceeded",
				RetryAfter: 30 * time.Second,
				Err:        errors.New("underlying error"),
			},
			expected: true,
		},
		{
			name: "retryable_error_without_retry_after",
			err: &RetryableError{
				StatusCode: 500,
				Message:    "Internal server error",
				RetryAfter: 0,
				Err:        errors.New("underlying error"),
			},
			expected: true,
		},
		{
			name: "retryable_error_with_zero_status_code",
			err: &RetryableError{
				StatusCode: 0,
				Message:    "Unknown error",
				RetryAfter: 5 * time.Second,
				Err:        errors.New("underlying error"),
			},
			expected: true,
		},
		{
			name: "retryable_error_with_empty_message",
			err: &RetryableError{
				StatusCode: 503,
				Message:    "",
				RetryAfter: 10 * time.Second,
				Err:        errors.New("underlying error"),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.IsRetryable()
			if result != tt.expected {
				t.Errorf("RetryableError.IsRetryable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRetryableError_ErrorInterface(t *testing.T) {
	// Test that RetryableError implements the error interface
	var err error = &RetryableError{
		StatusCode: 429,
		Message:    "Rate limit exceeded",
		RetryAfter: 30 * time.Second,
		Err:        errors.New("underlying error"),
	}

	if err == nil {
		t.Error("RetryableError should implement error interface")
	}

	// Test that we can call Error() method
	errorString := err.Error()
	if errorString == "" {
		t.Error("RetryableError.Error() should not return empty string")
	}
}

func TestRetryableError_ErrorWrapping(t *testing.T) {
	// Test that RetryableError properly wraps underlying errors
	underlyingErr := errors.New("network timeout")
	retryErr := &RetryableError{
		StatusCode: 408,
		Message:    "Request timeout",
		RetryAfter: 5 * time.Second,
		Err:        underlyingErr,
	}

	// Test Unwrap
	unwrapped := retryErr.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Test that errors.Is works
	if !errors.Is(retryErr, underlyingErr) {
		t.Error("errors.Is should return true for wrapped error")
	}

	// Test that errors.As works
	var asRetryErr *RetryableError
	if !errors.As(retryErr, &asRetryErr) {
		t.Error("errors.As should work with RetryableError")
	}
	if asRetryErr.StatusCode != 408 {
		t.Errorf("As() StatusCode = %d, want 408", asRetryErr.StatusCode)
	}
}

func TestRetryableError_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name     string
		scenario func() *RetryableError
		validate func(t *testing.T, err *RetryableError)
	}{
		{
			name: "openai_rate_limit",
			scenario: func() *RetryableError {
				return &RetryableError{
					StatusCode: 429,
					Message:    "Rate limit exceeded",
					RetryAfter: 60 * time.Second,
					Err:        errors.New("HTTP 429"),
				}
			},
			validate: func(t *testing.T, err *RetryableError) {
				if err.StatusCode != 429 {
					t.Errorf("Expected StatusCode=429, got %d", err.StatusCode)
				}
				if err.RetryAfter != 60*time.Second {
					t.Errorf("Expected RetryAfter=60s, got %v", err.RetryAfter)
				}
				if !err.IsRetryable() {
					t.Error("Expected IsRetryable()=true")
				}
			},
		},
		{
			name: "anthropic_rate_limit",
			scenario: func() *RetryableError {
				return &RetryableError{
					StatusCode: 429,
					Message:    "Rate limit exceeded",
					RetryAfter: 30 * time.Second,
					Err:        errors.New("HTTP 429"),
				}
			},
			validate: func(t *testing.T, err *RetryableError) {
				if err.StatusCode != 429 {
					t.Errorf("Expected StatusCode=429, got %d", err.StatusCode)
				}
				if err.RetryAfter != 30*time.Second {
					t.Errorf("Expected RetryAfter=30s, got %v", err.RetryAfter)
				}
				if !err.IsRetryable() {
					t.Error("Expected IsRetryable()=true")
				}
			},
		},
		{
			name: "server_error",
			scenario: func() *RetryableError {
				return &RetryableError{
					StatusCode: 500,
					Message:    "Internal server error",
					RetryAfter: 0, // No specific retry time
					Err:        errors.New("HTTP 500"),
				}
			},
			validate: func(t *testing.T, err *RetryableError) {
				if err.StatusCode != 500 {
					t.Errorf("Expected StatusCode=500, got %d", err.StatusCode)
				}
				if err.RetryAfter != 0 {
					t.Errorf("Expected RetryAfter=0, got %v", err.RetryAfter)
				}
				if !err.IsRetryable() {
					t.Error("Expected IsRetryable()=true")
				}
			},
		},
		{
			name: "gateway_timeout",
			scenario: func() *RetryableError {
				return &RetryableError{
					StatusCode: 504,
					Message:    "Gateway timeout",
					RetryAfter: 0,
					Err:        errors.New("HTTP 504"),
				}
			},
			validate: func(t *testing.T, err *RetryableError) {
				if err.StatusCode != 504 {
					t.Errorf("Expected StatusCode=504, got %d", err.StatusCode)
				}
				if err.RetryAfter != 0 {
					t.Errorf("Expected RetryAfter=0, got %v", err.RetryAfter)
				}
				if !err.IsRetryable() {
					t.Error("Expected IsRetryable()=true")
				}
			},
		},
		{
			name: "max_retries_exceeded",
			scenario: func() *RetryableError {
				return &RetryableError{
					StatusCode: 0,
					Message:    "max HTTP retries (5) exceeded",
					RetryAfter: 10 * time.Second,
					Err:        errors.New("max retries exceeded"),
				}
			},
			validate: func(t *testing.T, err *RetryableError) {
				if err.StatusCode != 0 {
					t.Errorf("Expected StatusCode=0, got %d", err.StatusCode)
				}
				if err.RetryAfter != 10*time.Second {
					t.Errorf("Expected RetryAfter=10s, got %v", err.RetryAfter)
				}
				if !err.IsRetryable() {
					t.Error("Expected IsRetryable()=true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scenario()
			tt.validate(t, err)
		})
	}
}

func TestRetryableError_ErrorChain(t *testing.T) {
	// Test error chaining with multiple levels
	rootErr := errors.New("root cause")
	wrappedErr := &RetryableError{
		StatusCode: 429,
		Message:    "Rate limit exceeded",
		RetryAfter: 30 * time.Second,
		Err:        rootErr,
	}

	// Test that we can unwrap to the root error
	unwrapped := wrappedErr.Unwrap()
	if unwrapped != rootErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, rootErr)
	}

	// Test that errors.Is works with the root error
	if !errors.Is(wrappedErr, rootErr) {
		t.Error("errors.Is should return true for root error")
	}

	// Test that errors.Is works with the RetryableError itself
	if !errors.Is(wrappedErr, wrappedErr) {
		t.Error("errors.Is should return true for the error itself")
	}

	// Test that we can extract the RetryableError
	var retryErr *RetryableError
	if !errors.As(wrappedErr, &retryErr) {
		t.Error("errors.As should work with RetryableError")
	}
	if retryErr.StatusCode != 429 {
		t.Errorf("As() StatusCode = %d, want 429", retryErr.StatusCode)
	}
}
