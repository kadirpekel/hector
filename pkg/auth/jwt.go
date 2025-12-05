// SPDX-License-Identifier: AGPL-3.0
// Copyright 2025 Kadir Pekel
//
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.gnu.org/licenses/agpl-3.0.en.html
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// JWTValidator validates JWT tokens using JWKS (JSON Web Key Set).
// It supports automatic key rotation via the jwk.Cache.
//
// This is ported from legacy Hector (pkg/auth/jwt.go) with the same
// production-tested implementation.
type JWTValidator struct {
	jwksURL  string
	cache    *jwk.Cache
	issuer   string
	audience string
}

// JWTValidatorConfig configures the JWT validator.
type JWTValidatorConfig struct {
	// JWKSURL is the URL to fetch JSON Web Key Set from.
	// Example: "https://auth.example.com/.well-known/jwks.json"
	JWKSURL string

	// Issuer is the expected token issuer (iss claim).
	// Example: "https://auth.example.com"
	Issuer string

	// Audience is the expected token audience (aud claim).
	// Example: "hector-api"
	Audience string

	// RefreshInterval is how often to refresh the JWKS.
	// Default: 15 minutes
	RefreshInterval time.Duration
}

// NewJWTValidator creates a new JWT validator.
// It fetches the JWKS from the provided URL and caches the keys.
func NewJWTValidator(cfg JWTValidatorConfig) (*JWTValidator, error) {
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("jwks_url is required")
	}
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("issuer is required")
	}
	if cfg.Audience == "" {
		return nil, fmt.Errorf("audience is required")
	}

	refreshInterval := cfg.RefreshInterval
	if refreshInterval == 0 {
		refreshInterval = 15 * time.Minute
	}

	ctx := context.Background()
	cache := jwk.NewCache(ctx)

	// Register the JWKS URL with automatic refresh
	if err := cache.Register(cfg.JWKSURL, jwk.WithMinRefreshInterval(refreshInterval)); err != nil {
		return nil, fmt.Errorf("failed to register JWKS URL: %w", err)
	}

	// Fetch initial JWKS to verify connectivity
	if _, err := cache.Refresh(ctx, cfg.JWKSURL); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", cfg.JWKSURL, err)
	}

	return &JWTValidator{
		jwksURL:  cfg.JWKSURL,
		cache:    cache,
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
	}, nil
}

// ValidateToken validates a JWT token string and returns the extracted claims.
// It verifies:
//   - Token signature against JWKS
//   - Token expiration (exp claim)
//   - Token not-before time (nbf claim)
//   - Issuer (iss claim)
//   - Audience (aud claim)
func (v *JWTValidator) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	// Get current key set from cache
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

	// Extract claims
	claims := &Claims{
		Subject: token.Subject(),
		Custom:  make(map[string]any),
	}

	// Extract common claims
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

	// Extract custom claims (excluding standard JWT claims)
	standardClaims := map[string]bool{
		"sub": true, "email": true, "role": true, "tenant_id": true,
		"iss": true, "aud": true, "exp": true, "iat": true, "nbf": true, "jti": true,
	}

	for iter := token.Iterate(ctx); iter.Next(ctx); {
		pair := iter.Pair()
		key, ok := pair.Key.(string)
		if !ok {
			continue
		}
		if !standardClaims[key] {
			claims.Custom[key] = pair.Value
		}
	}

	return claims, nil
}

// Close releases resources held by the validator.
// Currently a no-op but provided for future resource cleanup.
func (v *JWTValidator) Close() error {
	// jwk.Cache doesn't require explicit cleanup
	return nil
}

// TokenValidator is the interface for token validation.
// This allows for alternative implementations (e.g., for testing).
type TokenValidator interface {
	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)
	Close() error
}

// Ensure JWTValidator implements TokenValidator
var _ TokenValidator = (*JWTValidator)(nil)
