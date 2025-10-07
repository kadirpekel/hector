package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Test helper functions
func generateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

func createJWKS(publicKey *rsa.PublicKey) (jwk.Set, error) {
	key, err := jwk.FromRaw(publicKey)
	if err != nil {
		return nil, err
	}

	if err := key.Set(jwk.KeyIDKey, "test-key-id"); err != nil {
		return nil, err
	}

	if err := key.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		return nil, err
	}

	keyset := jwk.NewSet()
	if err := keyset.AddKey(key); err != nil {
		return nil, err
	}

	return keyset, nil
}

func createTestJWT(privateKey *rsa.PrivateKey, issuer, audience, subject string, claims map[string]interface{}) (string, error) {
	token := jwt.New()

	// Set standard claims
	if err := token.Set(jwt.IssuerKey, issuer); err != nil {
		return "", err
	}
	if err := token.Set(jwt.AudienceKey, audience); err != nil {
		return "", err
	}
	if err := token.Set(jwt.SubjectKey, subject); err != nil {
		return "", err
	}
	if err := token.Set(jwt.IssuedAtKey, time.Now()); err != nil {
		return "", err
	}
	if err := token.Set(jwt.ExpirationKey, time.Now().Add(time.Hour)); err != nil {
		return "", err
	}

	// Set custom claims
	for key, value := range claims {
		if err := token.Set(key, value); err != nil {
			return "", err
		}
	}

	// Sign token with key ID
	key, err := jwk.FromRaw(privateKey)
	if err != nil {
		return "", err
	}

	// Set the same key ID as in JWKS
	if err := key.Set(jwk.KeyIDKey, "test-key-id"); err != nil {
		return "", err
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, key))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}

func setupTestValidator(t testing.TB) (*JWTValidator, *rsa.PrivateKey, string, string, string) {
	// Generate test key pair
	privateKey, publicKey, err := generateRSAKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create JWKS
	keyset, err := createJWKS(publicKey)
	if err != nil {
		t.Fatalf("Failed to create JWKS: %v", err)
	}

	// Create test server for JWKS endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}

		// Convert keyset to JSON
		keysetJSON, err := json.Marshal(keyset)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(keysetJSON)
	}))

	jwksURL := server.URL + "/.well-known/jwks.json"
	issuer := "https://test-issuer.com"
	audience := "test-audience"

	// Create validator
	validator, err := NewJWTValidator(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	return validator, privateKey, issuer, audience, jwksURL
}
