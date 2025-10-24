package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type JWTValidator struct {
	jwksURL  string
	cache    *jwk.Cache
	issuer   string
	audience string
}

type Claims struct {
	Subject  string                 `json:"sub"`
	Email    string                 `json:"email"`
	Role     string                 `json:"role"`
	TenantID string                 `json:"tenant_id"`
	Custom   map[string]interface{} `json:"-"`
}

func NewJWTValidator(jwksURL, issuer, audience string) (*JWTValidator, error) {
	ctx := context.Background()

	cache := jwk.NewCache(ctx)

	if err := cache.Register(jwksURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		return nil, fmt.Errorf("failed to register JWKS URL: %w", err)
	}

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

func (v *JWTValidator) ValidateToken(ctx context.Context, tokenString string) (interface{}, error) {

	keyset, err := v.cache.Get(ctx, v.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

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

	claims := &Claims{
		Subject: token.Subject(),
		Custom:  make(map[string]interface{}),
	}

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

	for iter := token.Iterate(context.Background()); iter.Next(context.Background()); {
		pair := iter.Pair()
		key := pair.Key.(string)

		if key != "sub" && key != "email" && key != "role" && key != "tenant_id" &&
			key != "iss" && key != "aud" && key != "exp" && key != "iat" && key != "nbf" {
			claims.Custom[key] = pair.Value
		}
	}

	return claims, nil
}

func (v *JWTValidator) Close() {

}
