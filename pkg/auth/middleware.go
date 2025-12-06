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

package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// Middleware creates an HTTP middleware that validates JWT tokens.
// Requests without valid tokens receive 401 Unauthorized.
//
// The middleware extracts the token from the Authorization header:
//   - "Bearer <token>" format (preferred)
//   - Raw token (fallback)
//
// Valid claims are stored in the request context and can be retrieved
// using ClaimsFromContext().
func Middleware(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			// Extract token from header
			tokenString := extractToken(authHeader)
			if tokenString == "" {
				writeAuthError(w, "Invalid Authorization format, expected: Bearer <token>", http.StatusUnauthorized)
				return
			}

			// Validate token
			claims, err := validator.ValidateToken(r.Context(), tokenString)
			if err != nil {
				writeAuthError(w, fmt.Sprintf("Invalid token: %s", err.Error()), http.StatusUnauthorized)
				return
			}

			// Store claims in context and proceed
			ctx := ContextWithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// MiddlewareWithExclusions creates a middleware that skips auth for certain paths.
// This is useful for health checks, public endpoints, etc.
func MiddlewareWithExclusions(validator TokenValidator, excludedPaths []string) func(http.Handler) http.Handler {
	excludeSet := make(map[string]bool, len(excludedPaths))
	for _, path := range excludedPaths {
		excludeSet[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is excluded
			if excludeSet[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Also check with trailing slash variants
			pathWithSlash := r.URL.Path
			if !strings.HasSuffix(pathWithSlash, "/") {
				pathWithSlash += "/"
			}
			pathWithoutSlash := strings.TrimSuffix(r.URL.Path, "/")
			if excludeSet[pathWithSlash] || excludeSet[pathWithoutSlash] {
				next.ServeHTTP(w, r)
				return
			}

			// Apply auth middleware
			Middleware(validator)(next).ServeHTTP(w, r)
		})
	}
}

// RequireRole creates a middleware that requires the user to have a specific role.
// Must be used after Middleware() in the chain.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeAuthError(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !claims.HasAnyRole(roles...) {
				writeAuthError(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireTenant creates a middleware that requires the user to belong to a specific tenant.
// Must be used after Middleware() in the chain.
func RequireTenant(tenants ...string) func(http.Handler) http.Handler {
	tenantSet := make(map[string]bool, len(tenants))
	for _, t := range tenants {
		tenantSet[t] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeAuthError(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !tenantSet[claims.TenantID] {
				writeAuthError(w, "Forbidden: access denied for this tenant", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalMiddleware validates tokens if present but doesn't require them.
// If a valid token is present, claims are added to the context.
// If no token is present, the request proceeds without claims.
// If an invalid token is present, the request is rejected.
func OptionalMiddleware(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No token - proceed without auth
				next.ServeHTTP(w, r)
				return
			}

			// Token present - validate it
			tokenString := extractToken(authHeader)
			if tokenString == "" {
				writeAuthError(w, "Invalid Authorization format", http.StatusUnauthorized)
				return
			}

			claims, err := validator.ValidateToken(r.Context(), tokenString)
			if err != nil {
				writeAuthError(w, fmt.Sprintf("Invalid token: %s", err.Error()), http.StatusUnauthorized)
				return
			}

			ctx := ContextWithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken extracts the token from an Authorization header.
// Supports "Bearer <token>" and raw token formats.
func extractToken(authHeader string) string {
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	// Accept raw token as fallback
	return authHeader
}

// writeAuthError writes a JSON error response.
func writeAuthError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":"%s"}`, message)
}

// CredentialType identifies the type of credential for outbound requests.
type CredentialType string

const (
	CredentialTypeBearer CredentialType = "bearer"
	CredentialTypeAPIKey CredentialType = "api_key"
	CredentialTypeBasic  CredentialType = "basic"
)

// TokenProvider is a function that returns a token for outbound requests.
type TokenProvider func() (string, error)

// NewTokenProvider creates a TokenProvider based on credential configuration.
// This is useful for making authenticated outbound requests (e.g., to remote agents).
func NewTokenProvider(credType CredentialType, token, apiKey, username, password string) (TokenProvider, error) {
	switch credType {
	case CredentialTypeBearer:
		if token == "" {
			return nil, fmt.Errorf("bearer token is required")
		}
		t := token
		return func() (string, error) {
			return "Bearer " + t, nil
		}, nil

	case CredentialTypeAPIKey:
		if apiKey == "" {
			return nil, fmt.Errorf("api_key is required")
		}
		k := apiKey
		return func() (string, error) {
			return k, nil
		}, nil

	case CredentialTypeBasic:
		if username == "" || password == "" {
			return nil, fmt.Errorf("username and password are required for basic auth")
		}
		u, p := username, password
		return func() (string, error) {
			creds := u + ":" + p
			encoded := base64.StdEncoding.EncodeToString([]byte(creds))
			return "Basic " + encoded, nil
		}, nil

	default:
		return nil, fmt.Errorf("unsupported credential type: %s (supported: bearer, api_key, basic)", credType)
	}
}
