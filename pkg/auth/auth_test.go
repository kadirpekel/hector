package auth_test

import (
	"context"
	"testing"

	"github.com/verikod/hector/pkg/auth"
)

// =============================================================================
// Claims Tests
// =============================================================================

func TestClaims_GetClaim(t *testing.T) {
	t.Run("returns value when exists", func(t *testing.T) {
		claims := &auth.Claims{
			Custom: map[string]any{"key1": "value1"},
		}

		val, ok := claims.GetClaim("key1")
		if !ok {
			t.Error("GetClaim should return true for existing key")
		}
		if val != "value1" {
			t.Errorf("GetClaim = %v, want value1", val)
		}
	})

	t.Run("returns false when not exists", func(t *testing.T) {
		claims := &auth.Claims{
			Custom: map[string]any{"key1": "value1"},
		}

		_, ok := claims.GetClaim("nonexistent")
		if ok {
			t.Error("GetClaim should return false for nonexistent key")
		}
	})

	t.Run("handles nil Custom map", func(t *testing.T) {
		claims := &auth.Claims{}

		_, ok := claims.GetClaim("any")
		if ok {
			t.Error("GetClaim should return false for nil Custom map")
		}
	})
}

func TestClaims_GetStringClaim(t *testing.T) {
	t.Run("returns string value", func(t *testing.T) {
		claims := &auth.Claims{
			Custom: map[string]any{"name": "John Doe"},
		}

		val := claims.GetStringClaim("name")
		if val != "John Doe" {
			t.Errorf("GetStringClaim = %q, want 'John Doe'", val)
		}
	})

	t.Run("returns empty for non-string", func(t *testing.T) {
		claims := &auth.Claims{
			Custom: map[string]any{"count": 42},
		}

		val := claims.GetStringClaim("count")
		if val != "" {
			t.Errorf("GetStringClaim = %q, want empty for non-string", val)
		}
	})

	t.Run("returns empty for missing key", func(t *testing.T) {
		claims := &auth.Claims{}

		val := claims.GetStringClaim("missing")
		if val != "" {
			t.Errorf("GetStringClaim = %q, want empty for missing key", val)
		}
	})
}

func TestClaims_HasRole(t *testing.T) {
	claims := &auth.Claims{Role: "admin"}

	t.Run("returns true for matching role", func(t *testing.T) {
		if !claims.HasRole("admin") {
			t.Error("HasRole should return true for matching role")
		}
	})

	t.Run("returns false for non-matching role", func(t *testing.T) {
		if claims.HasRole("user") {
			t.Error("HasRole should return false for non-matching role")
		}
	})
}

func TestClaims_HasAnyRole(t *testing.T) {
	claims := &auth.Claims{Role: "editor"}

	t.Run("returns true when role is in list", func(t *testing.T) {
		if !claims.HasAnyRole("admin", "editor", "viewer") {
			t.Error("HasAnyRole should return true when role matches")
		}
	})

	t.Run("returns false when role not in list", func(t *testing.T) {
		if claims.HasAnyRole("admin", "superuser") {
			t.Error("HasAnyRole should return false when role doesn't match")
		}
	})

	t.Run("returns false for empty list", func(t *testing.T) {
		if claims.HasAnyRole() {
			t.Error("HasAnyRole should return false for empty list")
		}
	})
}

// =============================================================================
// Context Functions Tests
// =============================================================================

func TestClaimsFromContext(t *testing.T) {
	t.Run("returns claims when present", func(t *testing.T) {
		claims := &auth.Claims{Subject: "user-123", Email: "test@example.com"}
		ctx := auth.ContextWithClaims(context.Background(), claims)

		retrieved := auth.ClaimsFromContext(ctx)
		if retrieved == nil {
			t.Fatal("ClaimsFromContext should return claims")
		}
		if retrieved.Subject != "user-123" {
			t.Errorf("Subject = %q, want user-123", retrieved.Subject)
		}
	})

	t.Run("returns nil when not present", func(t *testing.T) {
		ctx := context.Background()

		if auth.ClaimsFromContext(ctx) != nil {
			t.Error("ClaimsFromContext should return nil for context without claims")
		}
	})
}

func TestContextWithClaims(t *testing.T) {
	claims := &auth.Claims{
		Subject:  "user-456",
		Email:    "user@test.com",
		Role:     "admin",
		TenantID: "tenant-1",
	}

	ctx := auth.ContextWithClaims(context.Background(), claims)
	retrieved := auth.ClaimsFromContext(ctx)

	if retrieved.Subject != claims.Subject {
		t.Error("Claims Subject should match")
	}
	if retrieved.Email != claims.Email {
		t.Error("Claims Email should match")
	}
	if retrieved.Role != claims.Role {
		t.Error("Claims Role should match")
	}
	if retrieved.TenantID != claims.TenantID {
		t.Error("Claims TenantID should match")
	}
}

// =============================================================================
// Error Constants Tests
// =============================================================================

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		err      error
		contains string
	}{
		{auth.ErrUnauthorized, "unauthorized"},
		{auth.ErrForbidden, "forbidden"},
		{auth.ErrInvalidToken, "invalid token"},
		{auth.ErrTokenExpired, "token expired"},
		{auth.ErrMissingClaims, "missing required claims"},
	}

	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			if tt.err == nil {
				t.Error("Error should not be nil")
			}
			// Just verify the error exists and has an Error() method
			if tt.err.Error() == "" {
				t.Error("Error message should not be empty")
			}
		})
	}
}

// =============================================================================
// Claims Fields Tests
// =============================================================================

func TestClaims_Fields(t *testing.T) {
	claims := &auth.Claims{
		Subject:  "sub-123",
		Email:    "test@example.com",
		Role:     "admin",
		TenantID: "tenant-abc",
		Custom: map[string]any{
			"org_id":     "org-123",
			"department": "engineering",
		},
	}

	if claims.Subject != "sub-123" {
		t.Errorf("Subject = %q", claims.Subject)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email = %q", claims.Email)
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q", claims.Role)
	}
	if claims.TenantID != "tenant-abc" {
		t.Errorf("TenantID = %q", claims.TenantID)
	}
	if len(claims.Custom) != 2 {
		t.Errorf("Custom should have 2 entries, got %d", len(claims.Custom))
	}
}

// =============================================================================
// JWTValidatorConfig Tests
// =============================================================================

func TestJWTValidatorConfig_Fields(t *testing.T) {
	// Test that config struct can be instantiated with all fields
	cfg := auth.JWTValidatorConfig{
		JWKSURL:         "https://auth.example.com/.well-known/jwks.json",
		Issuer:          "https://auth.example.com",
		Audience:        "hector-api",
		RefreshInterval: 0, // Will use default
	}

	if cfg.JWKSURL == "" {
		t.Error("JWKSURL should be set")
	}
	if cfg.Issuer == "" {
		t.Error("Issuer should be set")
	}
	if cfg.Audience == "" {
		t.Error("Audience should be set")
	}
}

func TestNewJWTValidator_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		cfg     auth.JWTValidatorConfig
		wantErr bool
	}{
		{
			name:    "missing JWKSURL",
			cfg:     auth.JWTValidatorConfig{Issuer: "iss", Audience: "aud"},
			wantErr: true,
		},
		{
			name:    "missing Issuer",
			cfg:     auth.JWTValidatorConfig{JWKSURL: "url", Audience: "aud"},
			wantErr: true,
		},
		{
			name:    "missing Audience",
			cfg:     auth.JWTValidatorConfig{JWKSURL: "url", Issuer: "iss"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.NewJWTValidator(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewJWTValidator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// ClaimsContextKey Tests
// =============================================================================

func TestClaimsContextKey(t *testing.T) {
	// Verify the constant exists (type is private but constant value can be used)
	// This is tested implicitly by ContextWithClaims/ClaimsFromContext working
	ctx := context.Background()
	claims := &auth.Claims{Subject: "test"}

	ctx = auth.ContextWithClaims(ctx, claims)
	retrieved := auth.ClaimsFromContext(ctx)

	if retrieved == nil || retrieved.Subject != "test" {
		t.Error("Context key should work correctly")
	}
}
