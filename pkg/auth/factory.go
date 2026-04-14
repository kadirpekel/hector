package auth

import (
	"fmt"
	"time"

	"github.com/verikod/hector/pkg/config"
)

// NewValidatorFromConfig creates a TokenValidator from configuration.
// Returns nil if authentication is not enabled.
func NewValidatorFromConfig(cfg *config.AuthConfig) (TokenValidator, error) {
	if cfg == nil || !cfg.IsEnabled() {
		return nil, nil
	}

	// Ensure defaults are applied
	cfg.SetDefaults()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid auth config: %w", err)
	}

	var validators []TokenValidator

	// 1. Secret Validator (Shared Token)
	if cfg.Secret != "" {
		validators = append(validators, NewSecretValidator(cfg.Secret))
	}

	// 2. JWT Validator (OIDC)
	if cfg.JWKSURL != "" && cfg.Issuer != "" && cfg.Audience != "" {
		jwtVal, err := NewJWTValidator(JWTValidatorConfig{
			JWKSURL:         cfg.JWKSURL,
			Issuer:          cfg.Issuer,
			Audience:        cfg.Audience,
			RefreshInterval: time.Duration(cfg.RefreshInterval) * time.Second,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create JWT validator: %w", err)
		}
		validators = append(validators, jwtVal)
	}

	if len(validators) == 0 {
		return nil, fmt.Errorf("auth enabled but no valid provider configured")
	}

	if len(validators) == 1 {
		return validators[0], nil
	}

	return NewCompositeValidator(validators...), nil
}
