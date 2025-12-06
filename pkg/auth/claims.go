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

// Package auth provides authentication and authorization for Hector v2.
//
// This package is ported from legacy Hector (pkg/auth) and adapted to work
// with a2a-go's CallInterceptor system for seamless integration.
//
// # Architecture
//
// The auth package follows a layered approach:
//
//  1. JWTValidator: Validates JWT tokens using JWKS (JSON Web Key Set)
//  2. HTTP Middleware: Extracts and validates tokens from HTTP requests
//  3. CallInterceptor: Bridges to a2a-go's authentication system
//
// # Usage
//
// Configure authentication in your hector.yaml:
//
//	server:
//	  auth:
//	    enabled: true
//	    jwks_url: "https://auth.example.com/.well-known/jwks.json"
//	    issuer: "https://auth.example.com"
//	    audience: "hector-api"
//
// The auth middleware will automatically validate JWT tokens and make
// claims available to agents via the invocation context.
package auth

import (
	"context"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// ClaimsContextKey is the context key for storing validated claims.
	ClaimsContextKey contextKey = "hector_auth_claims"
)

// Claims represents the validated claims from a JWT token.
// This structure is designed to support common identity providers
// (Auth0, Okta, Keycloak, etc.) while being extensible for custom claims.
type Claims struct {
	// Subject is the unique identifier for the user (sub claim).
	Subject string `json:"sub"`

	// Email is the user's email address (if provided).
	Email string `json:"email,omitempty"`

	// Role is the user's role for authorization decisions.
	Role string `json:"role,omitempty"`

	// TenantID supports multi-tenant applications.
	TenantID string `json:"tenant_id,omitempty"`

	// Custom contains any additional claims not mapped to struct fields.
	Custom map[string]any `json:"-"`
}

// GetClaim retrieves a custom claim by key.
func (c *Claims) GetClaim(key string) (any, bool) {
	if c.Custom == nil {
		return nil, false
	}
	val, ok := c.Custom[key]
	return val, ok
}

// GetStringClaim retrieves a custom claim as a string.
func (c *Claims) GetStringClaim(key string) string {
	if val, ok := c.GetClaim(key); ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// HasRole checks if the user has a specific role.
func (c *Claims) HasRole(role string) bool {
	return c.Role == role
}

// HasAnyRole checks if the user has any of the specified roles.
func (c *Claims) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if c.Role == role {
			return true
		}
	}
	return false
}

// ClaimsFromContext extracts claims from a context.
// Returns nil if no claims are present.
func ClaimsFromContext(ctx context.Context) *Claims {
	if claims, ok := ctx.Value(ClaimsContextKey).(*Claims); ok {
		return claims
	}
	return nil
}

// ContextWithClaims returns a new context with the given claims.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ClaimsContextKey, claims)
}
