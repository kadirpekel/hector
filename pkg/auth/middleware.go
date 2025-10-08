// Package auth provides authentication and authorization.
package auth

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const claimsContextKey contextKey = "claims"

// HTTPMiddleware creates HTTP middleware for JWT authentication
// It extracts the token from Authorization header, validates it,
// and adds claims to the request context
func (v *JWTValidator) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Missing Authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Remove "Bearer " prefix
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, `{"error":"Invalid Authorization format, expected: Bearer <token>"}`, http.StatusUnauthorized)
			return
		}

		// Validate token
		claimsInterface, err := v.ValidateToken(r.Context(), tokenString)
		if err != nil {
			http.Error(w, `{"error":"Unauthorized: `+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}

		// Convert interface{} back to *Claims for type safety
		claims, ok := claimsInterface.(*Claims)
		if !ok {
			http.Error(w, `{"error":"Internal error: invalid claims type"}`, http.StatusInternalServerError)
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClaims extracts claims from request context
// Returns nil if no claims are present (request not authenticated)
func GetClaims(r *http.Request) *Claims {
	if claims, ok := r.Context().Value(claimsContextKey).(*Claims); ok {
		return claims
	}
	return nil
}

// RequireRole creates middleware that checks for specific roles
func RequireRole(validator *JWTValidator, allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return validator.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Check if user has any of the allowed roles
			for _, allowedRole := range allowedRoles {
				if claims.Role == allowedRole {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, `{"error":"Forbidden: insufficient permissions"}`, http.StatusForbidden)
		}))
	}
}

// RequireTenant creates middleware that checks for specific tenants
func RequireTenant(validator *JWTValidator, allowedTenants ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return validator.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Check if user belongs to any of the allowed tenants
			for _, allowedTenant := range allowedTenants {
				if claims.TenantID == allowedTenant {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, `{"error":"Forbidden: access denied for this tenant"}`, http.StatusForbidden)
		}))
	}
}
