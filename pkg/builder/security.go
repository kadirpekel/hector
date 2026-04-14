package builder

import (
	"time"

	"github.com/verikod/hector/pkg/config"
)

// AuthBuilder provides a fluent API for building authentication configuration.
//
// Example:
//
//	auth := builder.NewAuth().
//	    JWKSURL("https://auth.example.com/.well-known/jwks.json").
//	    Issuer("https://auth.example.com").
//	    Audience("hector-api").
//	    Build()
type AuthBuilder struct {
	enabled         bool
	jwksURL         string
	issuer          string
	audience        string
	refreshInterval time.Duration
	requireAuth     *bool
	excludedPaths   []string
}

// NewAuth creates a new authentication configuration builder.
//
// Example:
//
//	auth := builder.NewAuth().
//	    JWKSURL("https://auth.example.com/.well-known/jwks.json").
//	    Issuer("https://auth.example.com").
//	    Audience("hector-api").
//	    Build()
func NewAuth() *AuthBuilder {
	return &AuthBuilder{
		enabled:         true,
		refreshInterval: 5 * time.Minute,
		excludedPaths:   []string{"/health", "/ready", "/.well-known/agent.json"},
	}
}

// Enabled enables or disables authentication.
//
// Example:
//
//	builder.NewAuth().Enabled(true)
func (b *AuthBuilder) Enabled(enabled bool) *AuthBuilder {
	b.enabled = enabled
	return b
}

// JWKSURL sets the JSON Web Key Set URL for token validation.
//
// Example:
//
//	builder.NewAuth().JWKSURL("https://auth.example.com/.well-known/jwks.json")
func (b *AuthBuilder) JWKSURL(url string) *AuthBuilder {
	b.jwksURL = url
	return b
}

// Issuer sets the expected JWT issuer (iss claim).
//
// Example:
//
//	builder.NewAuth().Issuer("https://auth.example.com")
func (b *AuthBuilder) Issuer(issuer string) *AuthBuilder {
	b.issuer = issuer
	return b
}

// Audience sets the expected JWT audience (aud claim).
//
// Example:
//
//	builder.NewAuth().Audience("hector-api")
func (b *AuthBuilder) Audience(audience string) *AuthBuilder {
	b.audience = audience
	return b
}

// RefreshInterval sets how often to refresh the JWKS.
//
// Example:
//
//	builder.NewAuth().RefreshInterval(10 * time.Minute)
func (b *AuthBuilder) RefreshInterval(interval time.Duration) *AuthBuilder {
	b.refreshInterval = interval
	return b
}

// RequireAuth sets whether authentication is mandatory.
// When false, unauthenticated requests proceed with nil user.
//
// Example:
//
//	builder.NewAuth().RequireAuth(true)
func (b *AuthBuilder) RequireAuth(require bool) *AuthBuilder {
	b.requireAuth = &require
	return b
}

// ExcludedPaths sets paths excluded from authentication.
//
// Example:
//
//	builder.NewAuth().ExcludedPaths("/health", "/ready")
func (b *AuthBuilder) ExcludedPaths(paths ...string) *AuthBuilder {
	b.excludedPaths = paths
	return b
}

// AddExcludedPath adds a path to the excluded paths list.
//
// Example:
//
//	builder.NewAuth().AddExcludedPath("/public")
func (b *AuthBuilder) AddExcludedPath(path string) *AuthBuilder {
	b.excludedPaths = append(b.excludedPaths, path)
	return b
}

// Build creates the authentication configuration.
func (b *AuthBuilder) Build() *config.AuthConfig {
	return &config.AuthConfig{
		Enabled:         b.enabled,
		JWKSURL:         b.jwksURL,
		Issuer:          b.issuer,
		Audience:        b.audience,
		RefreshInterval: int(b.refreshInterval.Seconds()),
		// TODO: Add RequireAuth and ExcludedPaths when needed
	}
}
