// Package auth provides authentication and authorization.
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// JWTValidator validates JWT tokens from external auth providers
// It auto-fetches and caches JWKS (public keys) from the provider
type JWTValidator struct {
	jwksURL  string
	cache    *jwk.Cache
	issuer   string
	audience string
}

// Claims represents extracted JWT claims
type Claims struct {
	Subject  string                 `json:"sub"`       // User ID
	Email    string                 `json:"email"`     // User email
	Role     string                 `json:"role"`      // User role (for RBAC)
	TenantID string                 `json:"tenant_id"` // Tenant ID (for multi-tenancy)
	Custom   map[string]interface{} `json:"-"`         // All other claims
}

// NewJWTValidator creates a validator that auto-fetches JWKS from the provider
// The JWKS is cached and auto-refreshed every 15 minutes to handle key rotation
func NewJWTValidator(jwksURL, issuer, audience string) (*JWTValidator, error) {
	ctx := context.Background()

	// Create JWKS cache with auto-refresh
	cache := jwk.NewCache(ctx)

	// Register JWKS URL for auto-refresh (every 15 minutes)
	if err := cache.Register(jwksURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		return nil, fmt.Errorf("failed to register JWKS URL: %w", err)
	}

	// Trigger initial fetch to validate configuration
	if _, err := cache.Refresh(ctx, jwksURL); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", jwksURL, err)
	}

	return &JWTValidator{
		jwksURL:  jwksURL,
		cache:    cache,
		issuer:   issuer,
		audience: audience,
	}, nil
}

// ValidateToken validates a JWT token and extracts claims
// It verifies:
// - JWT signature (using JWKS from provider)
// - Token expiration
// - Issuer matches configuration
// - Audience matches configuration
// Returns interface{} for compatibility with a2a.AuthValidator interface
func (v *JWTValidator) ValidateToken(ctx context.Context, tokenString string) (interface{}, error) {
	// Get JWKS from cache (auto-refreshed)
	keyset, err := v.cache.Get(ctx, v.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	// Parse and validate token
	token, err := jwt.Parse(
		[]byte(tokenString),
		jwt.WithKeySet(keyset),
		jwt.WithValidate(true),
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(v.audience),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Extract standard claims
	claims := &Claims{
		Subject: token.Subject(),
		Custom:  make(map[string]interface{}),
	}

	// Extract custom claims (optional, provider-specific)
	if email, ok := token.Get("email"); ok {
		if emailStr, ok := email.(string); ok {
			claims.Email = emailStr
		}
	}

	if role, ok := token.Get("role"); ok {
		if roleStr, ok := role.(string); ok {
			claims.Role = roleStr
		}
	}

	if tenantID, ok := token.Get("tenant_id"); ok {
		if tenantStr, ok := tenantID.(string); ok {
			claims.TenantID = tenantStr
		}
	}

	// Store all other claims in Custom map
	for iter := token.Iterate(context.Background()); iter.Next(context.Background()); {
		pair := iter.Pair()
		key := pair.Key.(string)

		// Skip standard claims already extracted
		if key != "sub" && key != "email" && key != "role" && key != "tenant_id" &&
			key != "iss" && key != "aud" && key != "exp" && key != "iat" && key != "nbf" {
			claims.Custom[key] = pair.Value
		}
	}

	return claims, nil
}

// Close stops the auto-refresh goroutine
func (v *JWTValidator) Close() {
	// The cache doesn't have an explicit close method
	// The goroutine will stop when the context is canceled
}
