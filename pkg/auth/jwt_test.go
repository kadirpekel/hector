package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestNewJWTValidator(t *testing.T) {
	// Generate test key pair
	_, publicKey, err := generateRSAKeyPair()
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
	defer server.Close()

	jwksURL := server.URL + "/.well-known/jwks.json"
	issuer := "https://test-issuer.com"
	audience := "test-audience"

	tests := []struct {
		name      string
		jwksURL   string
		issuer    string
		audience  string
		wantError bool
	}{
		{
			name:      "valid_configuration",
			jwksURL:   jwksURL,
			issuer:    issuer,
			audience:  audience,
			wantError: false,
		},
		{
			name:      "invalid_jwks_url",
			jwksURL:   "https://invalid-url.com/jwks.json",
			issuer:    issuer,
			audience:  audience,
			wantError: true,
		},
		{
			name:      "empty_jwks_url",
			jwksURL:   "",
			issuer:    issuer,
			audience:  audience,
			wantError: true,
		},
		{
			name:      "empty_issuer",
			jwksURL:   jwksURL,
			issuer:    "",
			audience:  audience,
			wantError: false, // Issuer can be empty, validation happens during token validation
		},
		{
			name:      "empty_audience",
			jwksURL:   jwksURL,
			issuer:    issuer,
			audience:  "",
			wantError: false, // Audience can be empty, validation happens during token validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewJWTValidator(tt.jwksURL, tt.issuer, tt.audience)

			if tt.wantError {
				if err == nil {
					t.Error("NewJWTValidator() expected error, got nil")
				}
				if validator != nil {
					t.Error("NewJWTValidator() expected nil validator on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewJWTValidator() error = %v, want nil", err)
				}
				if validator == nil {
					t.Error("NewJWTValidator() returned nil validator")
				}
				if validator != nil {
					if validator.jwksURL != tt.jwksURL {
						t.Errorf("NewJWTValidator() jwksURL = %v, want %v", validator.jwksURL, tt.jwksURL)
					}
					if validator.issuer != tt.issuer {
						t.Errorf("NewJWTValidator() issuer = %v, want %v", validator.issuer, tt.issuer)
					}
					if validator.audience != tt.audience {
						t.Errorf("NewJWTValidator() audience = %v, want %v", validator.audience, tt.audience)
					}
				}
			}
		})
	}
}

func TestJWTValidator_ValidateToken(t *testing.T) {
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
	defer server.Close()

	jwksURL := server.URL + "/.well-known/jwks.json"
	issuer := "https://test-issuer.com"
	audience := "test-audience"
	subject := "test-user-123"

	// Create validator
	validator, err := NewJWTValidator(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name        string
		issuer      string
		audience    string
		subject     string
		claims      map[string]interface{}
		wantError   bool
		checkClaims func(*testing.T, *Claims)
	}{
		{
			name:     "valid_token_with_basic_claims",
			issuer:   issuer,
			audience: audience,
			subject:  subject,
			claims: map[string]interface{}{
				"email": "test@example.com",
				"role":  "admin",
			},
			wantError: false,
			checkClaims: func(t *testing.T, claims *Claims) {
				if claims.Subject != subject {
					t.Errorf("Claims.Subject = %v, want %v", claims.Subject, subject)
				}
				if claims.Email != "test@example.com" {
					t.Errorf("Claims.Email = %v, want test@example.com", claims.Email)
				}
				if claims.Role != "admin" {
					t.Errorf("Claims.Role = %v, want admin", claims.Role)
				}
			},
		},
		{
			name:     "valid_token_with_tenant_id",
			issuer:   issuer,
			audience: audience,
			subject:  subject,
			claims: map[string]interface{}{
				"email":     "test@example.com",
				"role":      "user",
				"tenant_id": "tenant-123",
			},
			wantError: false,
			checkClaims: func(t *testing.T, claims *Claims) {
				if claims.TenantID != "tenant-123" {
					t.Errorf("Claims.TenantID = %v, want tenant-123", claims.TenantID)
				}
			},
		},
		{
			name:     "valid_token_with_custom_claims",
			issuer:   issuer,
			audience: audience,
			subject:  subject,
			claims: map[string]interface{}{
				"email":         "test@example.com",
				"role":          "user",
				"tenant_id":     "tenant-123",
				"custom_field":  "custom_value",
				"numeric_field": 42,
			},
			wantError: false,
			checkClaims: func(t *testing.T, claims *Claims) {
				if claims.Custom["custom_field"] != "custom_value" {
					t.Errorf("Claims.Custom[custom_field] = %v, want custom_value", claims.Custom["custom_field"])
				}
				// Note: numeric values might be stored as float64 in JWT
				if claims.Custom["numeric_field"] != 42 && claims.Custom["numeric_field"] != float64(42) {
					t.Errorf("Claims.Custom[numeric_field] = %v, want 42", claims.Custom["numeric_field"])
				}
			},
		},
		{
			name:      "invalid_issuer",
			issuer:    "https://wrong-issuer.com",
			audience:  audience,
			subject:   subject,
			claims:    map[string]interface{}{},
			wantError: true,
		},
		{
			name:      "invalid_audience",
			issuer:    issuer,
			audience:  "wrong-audience",
			subject:   subject,
			claims:    map[string]interface{}{},
			wantError: true,
		},
		{
			name:      "expired_token",
			issuer:    issuer,
			audience:  audience,
			subject:   subject,
			claims:    map[string]interface{}{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create token
			tokenClaims := make(map[string]interface{})
			for k, v := range tt.claims {
				tokenClaims[k] = v
			}

			// Handle expired token case
			if tt.name == "expired_token" {
				// Create expired token manually
				token := jwt.New()
				token.Set(jwt.IssuerKey, tt.issuer)
				token.Set(jwt.AudienceKey, tt.audience)
				token.Set(jwt.SubjectKey, tt.subject)
				token.Set(jwt.IssuedAtKey, time.Now().Add(-2*time.Hour))
				token.Set(jwt.ExpirationKey, time.Now().Add(-1*time.Hour)) // Expired 1 hour ago

				key, err := jwk.FromRaw(privateKey)
				if err != nil {
					t.Fatalf("Failed to create key: %v", err)
				}

				signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, key))
				if err != nil {
					t.Fatalf("Failed to sign token: %v", err)
				}

				_, err = validator.ValidateToken(context.Background(), string(signed))
				if tt.wantError && err == nil {
					t.Error("ValidateToken() expected error for expired token, got nil")
				}
				return
			}

			tokenString, err := createTestJWT(privateKey, tt.issuer, tt.audience, tt.subject, tokenClaims)
			if err != nil {
				t.Fatalf("Failed to create test JWT: %v", err)
			}

			claimsInterface, err := validator.ValidateToken(context.Background(), tokenString)

			if tt.wantError {
				if err == nil {
					t.Error("ValidateToken() expected error, got nil")
				}
				if claimsInterface != nil {
					t.Error("ValidateToken() expected nil claims on error")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateToken() error = %v, want nil", err)
				}
				if claimsInterface == nil {
					t.Error("ValidateToken() returned nil claims")
				}

				claims, ok := claimsInterface.(*Claims)
				if !ok {
					t.Error("ValidateToken() returned claims of wrong type")
				}

				if tt.checkClaims != nil {
					tt.checkClaims(t, claims)
				}
			}
		})
	}
}

func TestJWTValidator_ValidateToken_InvalidToken(t *testing.T) {
	// Generate test key pair
	_, publicKey, err := generateRSAKeyPair()
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
	defer server.Close()

	jwksURL := server.URL + "/.well-known/jwks.json"
	issuer := "https://test-issuer.com"
	audience := "test-audience"

	// Create validator
	validator, err := NewJWTValidator(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	tests := []struct {
		name        string
		tokenString string
		wantError   bool
	}{
		{
			name:        "empty_token",
			tokenString: "",
			wantError:   true,
		},
		{
			name:        "invalid_jwt_format",
			tokenString: "invalid.jwt.format",
			wantError:   true,
		},
		{
			name:        "malformed_jwt",
			tokenString: "not-a-jwt-token",
			wantError:   true,
		},
		{
			name:        "token_with_wrong_signature",
			tokenString: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.ValidateToken(context.Background(), tt.tokenString)

			if tt.wantError {
				if err == nil {
					t.Error("ValidateToken() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateToken() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestJWTValidator_Close(t *testing.T) {
	// Use setupTestValidator helper
	validator, privateKey, issuer, audience, _ := setupTestValidator(t)

	// Test Close method - should not panic
	validator.Close()

	// Test that validator still works after Close
	tokenString, err := createTestJWT(privateKey, issuer, audience, "test-user", map[string]interface{}{
		"email": "test@example.com",
	})
	if err != nil {
		t.Fatalf("Failed to create test JWT: %v", err)
	}

	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err != nil {
		t.Errorf("ValidateToken() after Close() error = %v, want nil", err)
	}
}

func TestClaims_Structure(t *testing.T) {
	// Test Claims struct creation and field access
	claims := &Claims{
		Subject:  "test-user-123",
		Email:    "test@example.com",
		Role:     "admin",
		TenantID: "tenant-456",
		Custom: map[string]interface{}{
			"custom_field":  "custom_value",
			"numeric_field": 42,
		},
	}

	if claims.Subject != "test-user-123" {
		t.Errorf("Claims.Subject = %v, want test-user-123", claims.Subject)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Claims.Email = %v, want test@example.com", claims.Email)
	}
	if claims.Role != "admin" {
		t.Errorf("Claims.Role = %v, want admin", claims.Role)
	}
	if claims.TenantID != "tenant-456" {
		t.Errorf("Claims.TenantID = %v, want tenant-456", claims.TenantID)
	}
	if claims.Custom["custom_field"] != "custom_value" {
		t.Errorf("Claims.Custom[custom_field] = %v, want custom_value", claims.Custom["custom_field"])
	}
	if claims.Custom["numeric_field"] != 42 {
		t.Errorf("Claims.Custom[numeric_field] = %v, want 42", claims.Custom["numeric_field"])
	}
}
