package hector

import (
	"github.com/kadirpekel/hector/pkg/config"
)

// SecurityBuilder provides a fluent API for building security config
type SecurityBuilder struct {
	config *config.SecurityConfig
}

// NewSecurityBuilder creates a new security builder
func NewSecurityBuilder(cfg *config.SecurityConfig) *SecurityBuilder {
	if cfg == nil {
		cfg = &config.SecurityConfig{
			Schemes: make(map[string]*config.SecurityScheme),
			Require: make([]map[string][]string, 0),
		}
	}
	if cfg.Schemes == nil {
		cfg.Schemes = make(map[string]*config.SecurityScheme)
	}
	if cfg.Require == nil {
		cfg.Require = make([]map[string][]string, 0)
	}
	return &SecurityBuilder{
		config: cfg,
	}
}

// JWKSURL sets the JWKS URL for JWT validation
func (b *SecurityBuilder) JWKSURL(url string) *SecurityBuilder {
	b.config.JWKSURL = url
	return b
}

// Issuer sets the JWT issuer
func (b *SecurityBuilder) Issuer(issuer string) *SecurityBuilder {
	b.config.Issuer = issuer
	return b
}

// Audience sets the JWT audience
func (b *SecurityBuilder) Audience(audience string) *SecurityBuilder {
	b.config.Audience = audience
	return b
}

// WithScheme adds a security scheme
func (b *SecurityBuilder) WithScheme(name string, scheme *config.SecurityScheme) *SecurityBuilder {
	if b.config.Schemes == nil {
		b.config.Schemes = make(map[string]*config.SecurityScheme)
	}
	b.config.Schemes[name] = scheme
	return b
}

// Scheme creates a security scheme builder
func (b *SecurityBuilder) Scheme(name string) *SecuritySchemeBuilder {
	if b.config.Schemes == nil {
		b.config.Schemes = make(map[string]*config.SecurityScheme)
	}
	if b.config.Schemes[name] == nil {
		b.config.Schemes[name] = &config.SecurityScheme{}
	}
	return NewSecuritySchemeBuilder(b.config.Schemes[name])
}

// Require adds a requirement mapping
func (b *SecurityBuilder) Require(requirement map[string][]string) *SecurityBuilder {
	if b.config.Require == nil {
		b.config.Require = make([]map[string][]string, 0)
	}
	b.config.Require = append(b.config.Require, requirement)
	return b
}

// Build returns the security config
func (b *SecurityBuilder) Build() *config.SecurityConfig {
	return b.config
}

// SecuritySchemeBuilder provides a fluent API for building security schemes
type SecuritySchemeBuilder struct {
	scheme *config.SecurityScheme
}

// NewSecuritySchemeBuilder creates a new security scheme builder
func NewSecuritySchemeBuilder(scheme *config.SecurityScheme) *SecuritySchemeBuilder {
	if scheme == nil {
		scheme = &config.SecurityScheme{}
	}
	return &SecuritySchemeBuilder{
		scheme: scheme,
	}
}

// Type sets the scheme type ("http", "apiKey", "oauth2", "openIdConnect", "mutualTLS")
func (b *SecuritySchemeBuilder) Type(typ string) *SecuritySchemeBuilder {
	b.scheme.Type = typ
	return b
}

// Scheme sets the HTTP scheme ("bearer" or "basic")
func (b *SecuritySchemeBuilder) Scheme(scheme string) *SecuritySchemeBuilder {
	b.scheme.Scheme = scheme
	return b
}

// BearerFormat sets the bearer format
func (b *SecuritySchemeBuilder) BearerFormat(format string) *SecuritySchemeBuilder {
	b.scheme.BearerFormat = format
	return b
}

// Description sets the scheme description
func (b *SecuritySchemeBuilder) Description(desc string) *SecuritySchemeBuilder {
	b.scheme.Description = desc
	return b
}

// In sets where the API key is located ("query", "header", "cookie")
func (b *SecuritySchemeBuilder) In(location string) *SecuritySchemeBuilder {
	b.scheme.In = location
	return b
}

// Name sets the API key name
func (b *SecuritySchemeBuilder) Name(name string) *SecuritySchemeBuilder {
	b.scheme.Name = name
	return b
}

// Build returns the security scheme
func (b *SecuritySchemeBuilder) Build() *config.SecurityScheme {
	return b.scheme
}
