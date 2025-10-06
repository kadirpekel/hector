package httpclient

import (
	"fmt"
	"time"
)

// RetryableError represents an error that can be retried with a specific delay
type RetryableError struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration // How long to wait before retrying
	Err        error
}

// Error implements the error interface
func (e *RetryableError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("HTTP %d: %s (retry after %v)", e.StatusCode, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// Unwrap returns the underlying error
func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if the error is retryable
func (e *RetryableError) IsRetryable() bool {
	return true
}
