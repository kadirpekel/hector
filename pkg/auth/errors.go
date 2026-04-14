package auth

import "errors"

// Common authentication errors.
var (
	// ErrUnauthorized is returned when authentication is required but not provided.
	ErrUnauthorized = errors.New("unauthorized: authentication required")

	// ErrForbidden is returned when the user lacks permission.
	ErrForbidden = errors.New("forbidden: insufficient permissions")

	// ErrInvalidToken is returned when a token cannot be validated.
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired is returned when a token has expired.
	ErrTokenExpired = errors.New("token expired")

	// ErrMissingClaims is returned when required claims are missing.
	ErrMissingClaims = errors.New("missing required claims")
)
