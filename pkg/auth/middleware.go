// Package auth provides authentication and authorization.
package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const claimsContextKey contextKey = "claims"

// HTTPMiddleware creates HTTP middleware for JWT authentication
// It extracts the token from Authorization header, validates it,
// and adds claims to the request context
func (v *JWTValidator) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Missing Authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Remove "Bearer " prefix
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, `{"error":"Invalid Authorization format, expected: Bearer <token>"}`, http.StatusUnauthorized)
			return
		}

		// Validate token
		claimsInterface, err := v.ValidateToken(r.Context(), tokenString)
		if err != nil {
			http.Error(w, `{"error":"Unauthorized: `+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}

		// Convert interface{} back to *Claims for type safety
		claims, ok := claimsInterface.(*Claims)
		if !ok {
			http.Error(w, `{"error":"Internal error: invalid claims type"}`, http.StatusInternalServerError)
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClaims extracts claims from request context
// Returns nil if no claims are present (request not authenticated)
func GetClaims(r *http.Request) *Claims {
	if claims, ok := r.Context().Value(claimsContextKey).(*Claims); ok {
		return claims
	}
	return nil
}

// RequireRole creates middleware that checks for specific roles
func RequireRole(validator *JWTValidator, allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return validator.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Check if user has any of the allowed roles
			for _, allowedRole := range allowedRoles {
				if claims.Role == allowedRole {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, `{"error":"Forbidden: insufficient permissions"}`, http.StatusForbidden)
		}))
	}
}

// RequireTenant creates middleware that checks for specific tenants
func RequireTenant(validator *JWTValidator, allowedTenants ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return validator.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Check if user belongs to any of the allowed tenants
			for _, allowedTenant := range allowedTenants {
				if claims.TenantID == allowedTenant {
					next.ServeHTTP(w, r)
					return
				}
			}

			http.Error(w, `{"error":"Forbidden: access denied for this tenant"}`, http.StatusForbidden)
		}))
	}
}

// ============================================================================
// gRPC INTERCEPTORS
// ============================================================================

// UnaryServerInterceptor creates a gRPC unary interceptor for JWT authentication
func (v *JWTValidator) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		tokenString := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if tokenString == authHeaders[0] {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization format, expected: Bearer <token>")
		}

		// Validate token
		claimsInterface, err := v.ValidateToken(ctx, tokenString)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
		}

		// Add claims to context
		ctx = context.WithValue(ctx, claimsContextKey, claimsInterface)

		// Call handler with authenticated context
		return handler(ctx, req)
	}
}

// StreamServerInterceptor creates a gRPC stream interceptor for JWT authentication
func (v *JWTValidator) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization header")
		}

		tokenString := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if tokenString == authHeaders[0] {
			return status.Error(codes.Unauthenticated, "invalid authorization format, expected: Bearer <token>")
		}

		// Validate token
		claimsInterface, err := v.ValidateToken(ss.Context(), tokenString)
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
		}

		// Create new context with claims
		ctx := context.WithValue(ss.Context(), claimsContextKey, claimsInterface)

		// Create wrapped stream with authenticated context
		wrappedStream := &authenticatedStream{ServerStream: ss, ctx: ctx}

		// Call handler with authenticated stream
		return handler(srv, wrappedStream)
	}
}

// authenticatedStream wraps grpc.ServerStream to use authenticated context
type authenticatedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedStream) Context() context.Context {
	return s.ctx
}

// GetClaimsFromContext extracts claims from gRPC context
func GetClaimsFromContext(ctx context.Context) *Claims {
	if claims, ok := ctx.Value(claimsContextKey).(*Claims); ok {
		return claims
	}
	return nil
}

// ============================================================================
// CLIENT-SIDE AUTH INTERCEPTORS (for agent-to-agent calls)
// ============================================================================

// ClientAuthInterceptor provides JWT authentication for outgoing gRPC calls
type ClientAuthInterceptor struct {
	tokenProvider func() (string, error) // Function to get the current token
}

// NewClientAuthInterceptor creates a new client auth interceptor
func NewClientAuthInterceptor(tokenProvider func() (string, error)) *ClientAuthInterceptor {
	return &ClientAuthInterceptor{
		tokenProvider: tokenProvider,
	}
}

// UnaryClientInterceptor creates a gRPC unary client interceptor for JWT authentication
func (c *ClientAuthInterceptor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Get token from provider
		token, err := c.tokenProvider()
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "failed to get auth token: %v", err)
		}

		// Add token to context metadata
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)

		// Call the remote method
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor creates a gRPC stream client interceptor for JWT authentication
func (c *ClientAuthInterceptor) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Get token from provider
		token, err := c.tokenProvider()
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "failed to get auth token: %v", err)
		}

		// Add token to context metadata
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)

		// Create the stream
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// NewAuthenticatedClientConn creates a gRPC client connection with authentication
// This is a helper function for making authenticated calls to external A2A agents
func NewAuthenticatedClientConn(target string, tokenProvider func() (string, error), opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	interceptor := NewClientAuthInterceptor(tokenProvider)

	// Add auth interceptors to dial options
	opts = append(opts,
		grpc.WithUnaryInterceptor(interceptor.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(interceptor.StreamClientInterceptor()),
	)

	return grpc.NewClient(target, opts...)
}

// NewTokenProviderFromCredentials creates a token provider function from credential values
// This is a helper to simplify creating authenticated connections to external agents
// Parameters match config.AgentCredentials fields to avoid circular dependency
func NewTokenProviderFromCredentials(credType, token, apiKey, username, password string) (func() (string, error), error) {
	switch credType {
	case "bearer":
		if token == "" {
			return nil, fmt.Errorf("bearer token is required")
		}
		// Capture token value
		t := token
		return func() (string, error) {
			return t, nil
		}, nil

	case "api_key":
		if apiKey == "" {
			return nil, fmt.Errorf("api_key is required")
		}
		// For API key auth, we'll use it as a Bearer token
		// The receiving server should handle it appropriately
		k := apiKey
		return func() (string, error) {
			return k, nil
		}, nil

	case "basic":
		if username == "" || password == "" {
			return nil, fmt.Errorf("username and password are required for basic auth")
		}
		// For basic auth, encode as base64(username:password)
		// The interceptor will add it as "Basic <encoded>"
		u, p := username, password
		return func() (string, error) {
			creds := u + ":" + p
			encoded := base64.StdEncoding.EncodeToString([]byte(creds))
			return "Basic " + encoded, nil
		}, nil

	default:
		return nil, fmt.Errorf("unsupported credential type: %s (supported: bearer, api_key, basic)", credType)
	}
}
