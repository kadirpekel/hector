package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestJWTValidator_HTTPMiddleware(t *testing.T) {
	validator, privateKey, issuer, audience, _ := setupTestValidator(t)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil {
			http.Error(w, "No claims found", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"subject":   claims.Subject,
			"email":     claims.Email,
			"role":      claims.Role,
			"tenant_id": claims.TenantID,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	middleware := validator.HTTPMiddleware(testHandler)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedBody   string
		setupToken     func() string
	}{
		{
			name:           "valid_token",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"email":"test@example.com","role":"admin","subject":"test-user-123","tenant_id":"tenant-456"}`,
			setupToken: func() string {
				token, err := createTestJWT(privateKey, issuer, audience, "test-user-123", map[string]interface{}{
					"email":     "test@example.com",
					"role":      "admin",
					"tenant_id": "tenant-456",
				})
				if err != nil {
					t.Fatalf("Failed to create test JWT: %v", err)
				}
				return token
			},
		},
		{
			name:           "missing_authorization_header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Missing Authorization header"}`,
			setupToken:     func() string { return "" },
		},
		{
			name:           "invalid_authorization_format",
			authHeader:     "InvalidFormat token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Invalid Authorization format, expected: Bearer <token>"}`,
			setupToken:     func() string { return "" },
		},
		{
			name:           "invalid_token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Unauthorized: `,
			setupToken:     func() string { return "invalid-token" },
		},
		{
			name:           "expired_token",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Unauthorized: `,
			setupToken: func() string {

				token := jwt.New()
				_ = token.Set(jwt.IssuerKey, issuer)
				_ = token.Set(jwt.AudienceKey, audience)
				_ = token.Set(jwt.SubjectKey, "test-user-123")
				_ = token.Set(jwt.IssuedAtKey, time.Now().Add(-2*time.Hour))
				_ = token.Set(jwt.ExpirationKey, time.Now().Add(-1*time.Hour))

				key, err := jwk.FromRaw(privateKey)
				if err != nil {
					t.Fatalf("Failed to create key: %v", err)
				}

				signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, key))
				if err != nil {
					t.Fatalf("Failed to sign token: %v", err)
				}

				return string(signed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupToken()
			authHeader := tt.authHeader + token

			req := httptest.NewRequest("GET", "/test", nil)
			if authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("HTTP status = %v, want %v", rr.Code, tt.expectedStatus)
			}

			body := rr.Body.String()
			if tt.expectedStatus == http.StatusOK {

				body = strings.TrimSpace(body)
				if body != tt.expectedBody {
					t.Errorf("Response body = %v, want %v", body, tt.expectedBody)
				}
			} else {

				if !strings.Contains(body, tt.expectedBody) {
					t.Errorf("Response body = %v, should contain %v", body, tt.expectedBody)
				}
			}
		})
	}
}

func TestGetClaims(t *testing.T) {
	validator, privateKey, issuer, audience, _ := setupTestValidator(t)

	tokenString, err := createTestJWT(privateKey, issuer, audience, "test-user-123", map[string]interface{}{
		"email":     "test@example.com",
		"role":      "admin",
		"tenant_id": "tenant-456",
	})
	if err != nil {
		t.Fatalf("Failed to create test JWT: %v", err)
	}

	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedClaims *Claims
	}{
		{
			name: "request_with_valid_claims",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer "+tokenString)

				claimsInterface, err := validator.ValidateToken(req.Context(), tokenString)
				if err != nil {
					t.Fatalf("Failed to validate token: %v", err)
				}

				claims := claimsInterface.(*Claims)
				ctx := context.WithValue(req.Context(), claimsContextKey, claims)
				return req.WithContext(ctx)
			},
			expectedClaims: &Claims{
				Subject:  "test-user-123",
				Email:    "test@example.com",
				Role:     "admin",
				TenantID: "tenant-456",
				Custom:   make(map[string]interface{}),
			},
		},
		{
			name: "request_without_claims",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/test", nil)
			},
			expectedClaims: nil,
		},
		{
			name: "request_with_invalid_claims_type",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				ctx := context.WithValue(req.Context(), claimsContextKey, "invalid-claims-type")
				return req.WithContext(ctx)
			},
			expectedClaims: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			claims := GetClaims(req)

			if tt.expectedClaims == nil {
				if claims != nil {
					t.Errorf("GetClaims() = %v, want nil", claims)
				}
			} else {
				if claims == nil {
					t.Error("GetClaims() = nil, want claims")
				} else {
					if claims.Subject != tt.expectedClaims.Subject {
						t.Errorf("Claims.Subject = %v, want %v", claims.Subject, tt.expectedClaims.Subject)
					}
					if claims.Email != tt.expectedClaims.Email {
						t.Errorf("Claims.Email = %v, want %v", claims.Email, tt.expectedClaims.Email)
					}
					if claims.Role != tt.expectedClaims.Role {
						t.Errorf("Claims.Role = %v, want %v", claims.Role, tt.expectedClaims.Role)
					}
					if claims.TenantID != tt.expectedClaims.TenantID {
						t.Errorf("Claims.TenantID = %v, want %v", claims.TenantID, tt.expectedClaims.TenantID)
					}
				}
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	validator, privateKey, issuer, audience, _ := setupTestValidator(t)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Access granted"))
	})

	tests := []struct {
		name           string
		allowedRoles   []string
		tokenRole      string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "user_with_allowed_role",
			allowedRoles:   []string{"admin", "user"},
			tokenRole:      "admin",
			expectedStatus: http.StatusOK,
			expectedBody:   "Access granted",
		},
		{
			name:           "user_with_another_allowed_role",
			allowedRoles:   []string{"admin", "user"},
			tokenRole:      "user",
			expectedStatus: http.StatusOK,
			expectedBody:   "Access granted",
		},
		{
			name:           "user_without_allowed_role",
			allowedRoles:   []string{"admin"},
			tokenRole:      "user",
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"Forbidden: insufficient permissions"}`,
		},
		{
			name:           "user_with_empty_role",
			allowedRoles:   []string{"admin"},
			tokenRole:      "",
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"Forbidden: insufficient permissions"}`,
		},
		{
			name:           "no_allowed_roles",
			allowedRoles:   []string{},
			tokenRole:      "admin",
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"Forbidden: insufficient permissions"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tokenString, err := createTestJWT(privateKey, issuer, audience, "test-user-123", map[string]interface{}{
				"email": "test@example.com",
				"role":  tt.tokenRole,
			})
			if err != nil {
				t.Fatalf("Failed to create test JWT: %v", err)
			}

			middleware := RequireRole(validator, tt.allowedRoles...)

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)

			rr := httptest.NewRecorder()
			middleware(testHandler).ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("HTTP status = %v, want %v", rr.Code, tt.expectedStatus)
			}

			body := rr.Body.String()
			if !strings.Contains(body, tt.expectedBody) {
				t.Errorf("Response body = %v, should contain %v", body, tt.expectedBody)
			}
		})
	}
}

func TestRequireTenant(t *testing.T) {
	validator, privateKey, issuer, audience, _ := setupTestValidator(t)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Access granted"))
	})

	tests := []struct {
		name           string
		allowedTenants []string
		tokenTenantID  string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "user_with_allowed_tenant",
			allowedTenants: []string{"tenant-123", "tenant-456"},
			tokenTenantID:  "tenant-123",
			expectedStatus: http.StatusOK,
			expectedBody:   "Access granted",
		},
		{
			name:           "user_with_another_allowed_tenant",
			allowedTenants: []string{"tenant-123", "tenant-456"},
			tokenTenantID:  "tenant-456",
			expectedStatus: http.StatusOK,
			expectedBody:   "Access granted",
		},
		{
			name:           "user_without_allowed_tenant",
			allowedTenants: []string{"tenant-123"},
			tokenTenantID:  "tenant-789",
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"Forbidden: access denied for this tenant"}`,
		},
		{
			name:           "user_with_empty_tenant_id",
			allowedTenants: []string{"tenant-123"},
			tokenTenantID:  "",
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"Forbidden: access denied for this tenant"}`,
		},
		{
			name:           "no_allowed_tenants",
			allowedTenants: []string{},
			tokenTenantID:  "tenant-123",
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"error":"Forbidden: access denied for this tenant"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tokenString, err := createTestJWT(privateKey, issuer, audience, "test-user-123", map[string]interface{}{
				"email":     "test@example.com",
				"role":      "user",
				"tenant_id": tt.tokenTenantID,
			})
			if err != nil {
				t.Fatalf("Failed to create test JWT: %v", err)
			}

			middleware := RequireTenant(validator, tt.allowedTenants...)

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)

			rr := httptest.NewRecorder()
			middleware(testHandler).ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("HTTP status = %v, want %v", rr.Code, tt.expectedStatus)
			}

			body := rr.Body.String()
			if !strings.Contains(body, tt.expectedBody) {
				t.Errorf("Response body = %v, should contain %v", body, tt.expectedBody)
			}
		})
	}
}

func TestRequireRole_WithoutToken(t *testing.T) {
	validator, _, _, _, _ := setupTestValidator(t)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Access granted"))
	})

	middleware := RequireRole(validator, "admin")

	req := httptest.NewRequest("GET", "/test", nil)

	rr := httptest.NewRecorder()
	middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("HTTP status = %v, want %v", rr.Code, http.StatusUnauthorized)
	}

	expectedBody := `{"error":"Missing Authorization header"}`
	body := strings.TrimSpace(rr.Body.String())
	if body != expectedBody {
		t.Errorf("Response body = %v, want %v", body, expectedBody)
	}
}

func TestRequireTenant_WithoutToken(t *testing.T) {
	validator, _, _, _, _ := setupTestValidator(t)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Access granted"))
	})

	middleware := RequireTenant(validator, "tenant-123")

	req := httptest.NewRequest("GET", "/test", nil)

	rr := httptest.NewRecorder()
	middleware(testHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("HTTP status = %v, want %v", rr.Code, http.StatusUnauthorized)
	}

	expectedBody := `{"error":"Missing Authorization header"}`
	body := strings.TrimSpace(rr.Body.String())
	if body != expectedBody {
		t.Errorf("Response body = %v, want %v", body, expectedBody)
	}
}
