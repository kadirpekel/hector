package ratelimit

import (
	"errors"
	"fmt"
)

// Common errors.
var (
	// ErrRateLimitExceeded is returned when a rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrInvalidIdentifier is returned when an identifier is invalid.
	ErrInvalidIdentifier = errors.New("invalid identifier")

	// ErrStoreUnavailable is returned when the store is unavailable.
	ErrStoreUnavailable = errors.New("store unavailable")
)

// RateLimitError represents a rate limit error with detailed information.
type RateLimitError struct {
	// Message is a human-readable error message.
	Message string

	// Result contains the detailed rate limit check result.
	Result *CheckResult
}

// Error returns the error message.
func (e *RateLimitError) Error() string {
	return e.Message
}

// Unwrap returns the underlying error.
func (e *RateLimitError) Unwrap() error {
	return ErrRateLimitExceeded
}

// NewRateLimitError creates a new RateLimitError from a CheckResult.
func NewRateLimitError(result *CheckResult) *RateLimitError {
	message := "rate limit exceeded"
	if result != nil && result.Reason != "" {
		message = result.Reason
	}
	return &RateLimitError{
		Message: message,
		Result:  result,
	}
}

// IsRateLimitError checks if an error is a rate limit error.
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	var rle *RateLimitError
	if errors.As(err, &rle) {
		return true
	}
	return errors.Is(err, ErrRateLimitExceeded)
}

// GetRateLimitResult extracts the CheckResult from a rate limit error.
// Returns nil if the error is not a RateLimitError.
func GetRateLimitResult(err error) *CheckResult {
	if err == nil {
		return nil
	}
	var rle *RateLimitError
	if errors.As(err, &rle) {
		return rle.Result
	}
	return nil
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Error returns the validation error message.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
