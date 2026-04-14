package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// TokenIssuer issues JWT tokens for apps.
// It uses asymmetric keys (RSA or Ed25519) for signing and provides a JWKS endpoint for verification.
type TokenIssuer struct {
	privateKey jwk.Key
	publicKey  jwk.Key
	issuer     string
	audience   string
	keyID      string
}

// TokenIssuerConfig configures the token issuer.
type TokenIssuerConfig struct {
	// Issuer is the issuer claim (iss) for tokens.
	// Example: "https://hector.example.com"
	Issuer string

	// Audience is the default audience claim (aud) for tokens.
	// Example: "hector-api"
	Audience string

	// PrivateKeyPEM is optional pre-existing signing key (RSA or Ed25519) in PEM format.
	// If not provided, a new RSA key pair will be generated.
	PrivateKeyPEM []byte
}

// NewTokenIssuer creates a new token issuer.
// If no private key is provided, it generates a new RSA-2048 key pair.
func NewTokenIssuer(cfg TokenIssuerConfig) (*TokenIssuer, error) {
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("issuer is required")
	}
	if cfg.Audience == "" {
		return nil, fmt.Errorf("audience is required")
	}

	var privateKey jwk.Key
	var err error

	if len(cfg.PrivateKeyPEM) > 0 {
		// Parse provided key
		privateKey, err = jwk.ParseKey(cfg.PrivateKeyPEM, jwk.WithPEM(true))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
	} else {
		// Generate new RSA key pair
		rawPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("failed to generate RSA key: %w", err)
		}

		privateKey, err = jwk.FromRaw(rawPrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create JWK from private key: %w", err)
		}
	}

	// Generate a deterministic key ID from the public key thumbprint
	thumbprint, err := privateKey.Thumbprint(crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("failed to generate thumbprint: %w", err)
	}
	keyID := base64.RawURLEncoding.EncodeToString(thumbprint)[:12]

	// Set key metadata
	if err := privateKey.Set(jwk.KeyIDKey, keyID); err != nil {
		return nil, fmt.Errorf("failed to set key ID: %w", err)
	}

	// Set algorithm based on key type
	alg := jwa.RS256
	if privateKey.KeyType() == jwa.OKP {
		alg = jwa.EdDSA
	}

	if err := privateKey.Set(jwk.AlgorithmKey, alg); err != nil {
		return nil, fmt.Errorf("failed to set algorithm: %w", err)
	}
	if err := privateKey.Set(jwk.KeyUsageKey, "sig"); err != nil {
		return nil, fmt.Errorf("failed to set key usage: %w", err)
	}

	// Extract public key
	publicKey, err := privateKey.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to extract public key: %w", err)
	}

	return &TokenIssuer{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     cfg.Issuer,
		audience:   cfg.Audience,
		keyID:      keyID,
	}, nil
}

// IssueToken creates a new JWT token for the given app.
func (i *TokenIssuer) IssueToken(appID string, ttl time.Duration) (string, error) {
	if appID == "" {
		return "", fmt.Errorf("appID is required")
	}

	if ttl == 0 {
		ttl = 24 * time.Hour // Default: 24 hours
	}

	now := time.Now()

	// Build token claims
	builder := jwt.NewBuilder().
		Subject(appID).
		Issuer(i.issuer).
		Audience([]string{i.audience}).
		IssuedAt(now).
		Expiration(now.Add(ttl)).
		NotBefore(now).
		JwtID(uuid.NewString())

	// Add tenant_id claim for multi-app support
	builder = builder.Claim("tenant_id", appID)
	builder = builder.Claim("role", "app")

	token, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	return i.signToken(token)
}

func (i *TokenIssuer) signToken(token jwt.Token) (string, error) {
	alg := jwa.RS256
	if i.privateKey.KeyType() == jwa.OKP {
		alg = jwa.EdDSA
	}

	signed, err := jwt.Sign(token, jwt.WithKey(alg, i.privateKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return string(signed), nil
}

// IssueAdminToken creates a JWT token with admin role.
func (i *TokenIssuer) IssueAdminToken(subject string, ttl time.Duration) (string, error) {
	if subject == "" {
		subject = "admin"
	}

	if ttl == 0 {
		ttl = 1 * time.Hour // Default: 1 hour for admin tokens
	}

	now := time.Now()

	token, err := jwt.NewBuilder().
		Subject(subject).
		Issuer(i.issuer).
		Audience([]string{i.audience}).
		IssuedAt(now).
		Expiration(now.Add(ttl)).
		NotBefore(now).
		JwtID(uuid.NewString()).
		Claim("role", "admin").
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build admin token: %w", err)
	}

	signed, err := i.signToken(token)
	if err != nil {
		return "", fmt.Errorf("failed to sign admin token: %w", err)
	}

	return string(signed), nil
}

// PublicJWKS returns the public key set for external verification.
// This should be served at /.well-known/jwks.json
func (i *TokenIssuer) PublicJWKS() jwk.Set {
	set := jwk.NewSet()
	_ = set.AddKey(i.publicKey)
	return set
}

// ValidateToken validates a token issued by this issuer.
// This is used for in-process validation without HTTP round-trip.
func (i *TokenIssuer) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	set := jwk.NewSet()
	_ = set.AddKey(i.publicKey)

	token, err := jwt.Parse(
		[]byte(tokenString),
		jwt.WithKeySet(set),
		jwt.WithValidate(true),
		jwt.WithIssuer(i.issuer),
		jwt.WithAudience(i.audience),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Extract claims
	claims := &Claims{
		Subject: token.Subject(),
		Custom:  make(map[string]any),
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

	return claims, nil
}

// Close releases resources held by the issuer.
func (i *TokenIssuer) Close() error {
	return nil
}

// Issuer returns the issuer URL.
func (i *TokenIssuer) Issuer() string {
	return i.issuer
}

// Ensure TokenIssuer implements TokenValidator
var _ TokenValidator = (*TokenIssuer)(nil)
