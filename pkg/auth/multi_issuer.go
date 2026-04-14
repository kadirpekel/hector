package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// MultiIssuerValidator validates JWTs from multiple issuers.
// It extracts the issuer claim from the token and routes to the correct validator.
// This enables unified auth for both hector-issued tokens and external OIDC tokens.
type MultiIssuerValidator struct {
	// selfIssuer is the issuer URL for hector-issued tokens
	selfIssuer string

	// selfValidator validates hector-issued tokens (in-memory, no HTTP)
	selfValidator TokenValidator

	// externalValidators maps external issuer URLs to their validators
	externalValidators map[string]TokenValidator
}

// MultiIssuerConfig configures the multi-issuer validator.
type MultiIssuerConfig struct {
	// SelfIssuer is the issuer URL for hector-issued tokens.
	SelfIssuer string

	// SelfValidator validates hector-issued tokens.
	SelfValidator TokenValidator

	// ExternalValidators maps external issuer URLs to their JWKS validators.
	ExternalValidators map[string]TokenValidator
}

// NewMultiIssuerValidator creates a new multi-issuer validator.
func NewMultiIssuerValidator(cfg MultiIssuerConfig) (*MultiIssuerValidator, error) {
	if cfg.SelfIssuer == "" {
		return nil, fmt.Errorf("self_issuer is required")
	}
	if cfg.SelfValidator == nil {
		return nil, fmt.Errorf("self_validator is required")
	}

	return &MultiIssuerValidator{
		selfIssuer:         cfg.SelfIssuer,
		selfValidator:      cfg.SelfValidator,
		externalValidators: cfg.ExternalValidators,
	}, nil
}

// ValidateToken validates a JWT token by routing to the correct validator based on issuer.
func (v *MultiIssuerValidator) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	// Extract issuer from token (without full verification)
	issuer, err := extractIssuerFromJWT(tokenString)
	if err != nil {
		// If we can't extract issuer, try self validator first (might be API key)
		return v.selfValidator.ValidateToken(ctx, tokenString)
	}

	// Route to correct validator based on issuer
	if issuer == v.selfIssuer {
		// Hector-issued token - validate in-memory (no HTTP round-trip)
		return v.selfValidator.ValidateToken(ctx, tokenString)
	}

	// External issuer - find and use external validator
	if v.externalValidators != nil {
		if validator, ok := v.externalValidators[issuer]; ok {
			return validator.ValidateToken(ctx, tokenString)
		}
	}

	return nil, fmt.Errorf("unknown issuer: %s", issuer)
}

// Close closes all validators.
func (v *MultiIssuerValidator) Close() error {
	var errs []string

	if err := v.selfValidator.Close(); err != nil {
		errs = append(errs, fmt.Sprintf("self: %s", err))
	}

	for issuer, validator := range v.externalValidators {
		if err := validator.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %s", issuer, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close validators: %s", strings.Join(errs, "; "))
	}
	return nil
}

// extractIssuerFromJWT extracts the issuer claim from a JWT without verification.
// This is safe because we're only using it for routing; the actual validation
// happens with the proper validator.
func extractIssuerFromJWT(tokenString string) (string, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format")
	}

	// Decode payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode payload: %w", err)
	}

	var claims struct {
		Issuer string `json:"iss"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("failed to parse claims: %w", err)
	}

	if claims.Issuer == "" {
		return "", fmt.Errorf("no issuer claim found")
	}

	return claims.Issuer, nil
}

// AddExternalValidator adds an external validator for a given issuer.
func (v *MultiIssuerValidator) AddExternalValidator(issuer string, validator TokenValidator) {
	if v.externalValidators == nil {
		v.externalValidators = make(map[string]TokenValidator)
	}
	v.externalValidators[issuer] = validator
}

// Ensure MultiIssuerValidator implements TokenValidator
var _ TokenValidator = (*MultiIssuerValidator)(nil)
