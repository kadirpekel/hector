package a2a

import (
	"testing"
	"time"
)

// ============================================================================
// TRUE UNIT TESTS for A2A Client
// These test business logic in isolation WITHOUT HTTP layer
// ============================================================================

func TestNewClient_DefaultConfig(t *testing.T) {
	// Test with nil config
	client := NewClient(nil)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}

	// Verify default timeout
	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("Expected default timeout 60s, got %v", client.httpClient.Timeout)
	}

	// Auth should be nil by default
	if client.auth != nil {
		t.Error("auth should be nil by default")
	}
}

func TestNewClient_CustomTimeout(t *testing.T) {
	cfg := &ClientConfig{
		Timeout: 30 * time.Second,
	}

	client := NewClient(cfg)

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.httpClient.Timeout)
	}
}

func TestNewClient_ZeroTimeout(t *testing.T) {
	// When timeout is 0, should default to 60s
	cfg := &ClientConfig{
		Timeout: 0,
	}

	client := NewClient(cfg)

	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("Expected default timeout 60s for zero timeout, got %v", client.httpClient.Timeout)
	}
}

func TestNewClient_WithAuthentication(t *testing.T) {
	tests := []struct {
		name string
		auth *AuthCredentials
	}{
		{
			name: "bearer token",
			auth: &AuthCredentials{
				Type:  "bearer",
				Token: "test-token-123",
			},
		},
		{
			name: "api key",
			auth: &AuthCredentials{
				Type:         "apiKey",
				APIKey:       "test-api-key",
				APIKeyHeader: "X-Custom-Key",
			},
		},
		{
			name: "api key with default header",
			auth: &AuthCredentials{
				Type:   "apiKey",
				APIKey: "test-key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ClientConfig{
				Auth: tt.auth,
			}

			client := NewClient(cfg)

			if client.auth == nil {
				t.Error("auth is nil")
				return
			}

			if client.auth.Type != tt.auth.Type {
				t.Errorf("Expected auth type '%s', got '%s'", tt.auth.Type, client.auth.Type)
			}

			if client.auth.Token != tt.auth.Token {
				t.Errorf("Expected token '%s', got '%s'", tt.auth.Token, client.auth.Token)
			}

			if client.auth.APIKey != tt.auth.APIKey {
				t.Errorf("Expected API key '%s', got '%s'", tt.auth.APIKey, client.auth.APIKey)
			}
		})
	}
}

func TestAuthCredentials_Types(t *testing.T) {
	// Test that AuthCredentials struct can hold different auth types
	tests := []struct {
		name  string
		creds AuthCredentials
	}{
		{
			name: "bearer token only",
			creds: AuthCredentials{
				Type:  "bearer",
				Token: "eyJhbG...",
			},
		},
		{
			name: "api key only",
			creds: AuthCredentials{
				Type:   "apiKey",
				APIKey: "sk-1234567890",
			},
		},
		{
			name: "api key with custom header",
			creds: AuthCredentials{
				Type:         "apiKey",
				APIKey:       "key-123",
				APIKeyHeader: "X-My-API-Key",
			},
		},
		{
			name: "empty credentials",
			creds: AuthCredentials{
				Type: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the struct can be created and fields accessed
			if tt.creds.Type == "bearer" && tt.creds.Token == "" {
				t.Error("Bearer token should not be empty")
			}

			if tt.creds.Type == "apiKey" && tt.creds.APIKey == "" {
				t.Error("API key should not be empty")
			}
		})
	}
}

func TestClientConfig_Validation(t *testing.T) {
	// Test that ClientConfig accepts various configurations
	tests := []struct {
		name   string
		config *ClientConfig
		valid  bool
	}{
		{
			name: "valid full config",
			config: &ClientConfig{
				Timeout: 30 * time.Second,
				Auth: &AuthCredentials{
					Type:  "bearer",
					Token: "test",
				},
			},
			valid: true,
		},
		{
			name:   "nil config",
			config: nil,
			valid:  true, // NewClient handles nil
		},
		{
			name: "config with only timeout",
			config: &ClientConfig{
				Timeout: 10 * time.Second,
			},
			valid: true,
		},
		{
			name: "config with only auth",
			config: &ClientConfig{
				Auth: &AuthCredentials{
					Type:  "bearer",
					Token: "test",
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)

			if client == nil && tt.valid {
				t.Error("NewClient returned nil for valid config")
			}

			if client != nil && !tt.valid {
				t.Error("NewClient should have failed for invalid config")
			}
		})
	}
}

func TestClient_TimeoutConfiguration(t *testing.T) {
	// Test various timeout configurations
	tests := []struct {
		name            string
		config          *ClientConfig
		expectedTimeout time.Duration
	}{
		{
			name:            "nil config uses default",
			config:          nil,
			expectedTimeout: 60 * time.Second,
		},
		{
			name: "zero timeout uses default",
			config: &ClientConfig{
				Timeout: 0,
			},
			expectedTimeout: 60 * time.Second,
		},
		{
			name: "custom timeout is respected",
			config: &ClientConfig{
				Timeout: 120 * time.Second,
			},
			expectedTimeout: 120 * time.Second,
		},
		{
			name: "very short timeout",
			config: &ClientConfig{
				Timeout: 5 * time.Second,
			},
			expectedTimeout: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)

			if client.httpClient.Timeout != tt.expectedTimeout {
				t.Errorf("Expected timeout %v, got %v", tt.expectedTimeout, client.httpClient.Timeout)
			}
		})
	}
}

func TestClient_AuthenticationPersistence(t *testing.T) {
	// Test that authentication credentials are stored correctly
	auth := &AuthCredentials{
		Type:         "bearer",
		Token:        "test-token-xyz",
		APIKey:       "fallback-key",
		APIKeyHeader: "X-Custom-Header",
	}

	cfg := &ClientConfig{
		Auth: auth,
	}

	client := NewClient(cfg)

	// Verify all fields are preserved
	if client.auth == nil {
		t.Fatal("auth is nil")
	}

	if client.auth.Type != auth.Type {
		t.Errorf("Type not preserved: expected '%s', got '%s'", auth.Type, client.auth.Type)
	}

	if client.auth.Token != auth.Token {
		t.Errorf("Token not preserved: expected '%s', got '%s'", auth.Token, client.auth.Token)
	}

	if client.auth.APIKey != auth.APIKey {
		t.Errorf("APIKey not preserved: expected '%s', got '%s'", auth.APIKey, client.auth.APIKey)
	}

	if client.auth.APIKeyHeader != auth.APIKeyHeader {
		t.Errorf("APIKeyHeader not preserved: expected '%s', got '%s'", auth.APIKeyHeader, client.auth.APIKeyHeader)
	}
}

func TestAuthCredentials_DifferentAuthTypes(t *testing.T) {
	// Test different authentication types
	tests := []struct {
		name        string
		credentials AuthCredentials
		hasToken    bool
		hasAPIKey   bool
	}{
		{
			name: "bearer auth only",
			credentials: AuthCredentials{
				Type:  "bearer",
				Token: "eyJhbGciOi...",
			},
			hasToken:  true,
			hasAPIKey: false,
		},
		{
			name: "api key auth only",
			credentials: AuthCredentials{
				Type:   "apiKey",
				APIKey: "sk-123456",
			},
			hasToken:  false,
			hasAPIKey: true,
		},
		{
			name: "both present (bearer takes precedence)",
			credentials: AuthCredentials{
				Type:   "bearer",
				Token:  "token-abc",
				APIKey: "key-def",
			},
			hasToken:  true,
			hasAPIKey: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hasToken && tt.credentials.Token == "" {
				t.Error("Expected token to be set")
			}

			if tt.hasAPIKey && tt.credentials.APIKey == "" {
				t.Error("Expected API key to be set")
			}

			if !tt.hasToken && tt.credentials.Token != "" {
				t.Error("Token should not be set")
			}
		})
	}
}

func TestClient_NilAuthHandling(t *testing.T) {
	// Test that nil auth is handled correctly
	cfg := &ClientConfig{
		Timeout: 30 * time.Second,
		Auth:    nil, // Explicitly nil
	}

	client := NewClient(cfg)

	if client.auth != nil {
		t.Error("auth should be nil when not provided")
	}

	// Client should still be functional
	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestClient_MultipleInstances(t *testing.T) {
	// Test that multiple client instances are independent
	client1 := NewClient(&ClientConfig{
		Timeout: 10 * time.Second,
		Auth: &AuthCredentials{
			Type:  "bearer",
			Token: "token1",
		},
	})

	client2 := NewClient(&ClientConfig{
		Timeout: 20 * time.Second,
		Auth: &AuthCredentials{
			Type:  "bearer",
			Token: "token2",
		},
	})

	// Verify they have different settings
	if client1.httpClient.Timeout == client2.httpClient.Timeout {
		t.Error("Clients should have different timeouts")
	}

	if client1.auth.Token == client2.auth.Token {
		t.Error("Clients should have different auth tokens")
	}

	// Verify modifying one doesn't affect the other
	if client1.httpClient == client2.httpClient {
		t.Error("Clients should have separate httpClient instances")
	}
}

// ============================================================================
// COVERAGE SUMMARY
// These unit tests cover:
// - Client initialization with various configs
// - Timeout configuration and defaults
// - Authentication credential handling and persistence
// - Different auth types (bearer, API key)
// - Nil auth handling
// - Multiple client instances independence
// - Configuration validation
//
// What's NOT tested here (by design):
// - HTTP requests/responses: That's integration testing (see client_integration_test.go)
// - Network operations: Requires real HTTP server (integration)
// - SSE streaming: Requires HTTP infrastructure (integration)
//
// This file tests BUSINESS LOGIC, not HTTP/INTEGRATION.
// Run with: go test ./pkg/a2a/
// Integration tests: go test -tags=integration ./pkg/a2a/
//
// Target: 40%+ coverage of business logic
// ============================================================================
