package httpclient

import (
	"fmt"
	"time"
)

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

func (e *RetryableError) IsRetryable() bool {
	return true
}
