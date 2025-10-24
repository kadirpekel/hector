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

type contextKey string

const claimsContextKey contextKey = "claims"

func (v *JWTValidator) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"Missing Authorization header"}`, http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, `{"error":"Invalid Authorization format, expected: Bearer <token>"}`, http.StatusUnauthorized)
			return
		}

		claimsInterface, err := v.ValidateToken(r.Context(), tokenString)
		if err != nil {
			http.Error(w, `{"error":"Unauthorized: `+err.Error()+`"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := claimsInterface.(*Claims)
		if !ok {
			http.Error(w, `{"error":"Internal error: invalid claims type"}`, http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetClaims(r *http.Request) *Claims {
	if claims, ok := r.Context().Value(claimsContextKey).(*Claims); ok {
		return claims
	}
	return nil
}

func RequireRole(validator *JWTValidator, allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return validator.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

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

func RequireTenant(validator *JWTValidator, allowedTenants ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return validator.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r)
			if claims == nil {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

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

func (v *JWTValidator) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

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

		claimsInterface, err := v.ValidateToken(ctx, tokenString)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
		}

		ctx = context.WithValue(ctx, claimsContextKey, claimsInterface)

		return handler(ctx, req)
	}
}

func (v *JWTValidator) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

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

		claimsInterface, err := v.ValidateToken(ss.Context(), tokenString)
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "unauthorized: %v", err)
		}

		ctx := context.WithValue(ss.Context(), claimsContextKey, claimsInterface)

		wrappedStream := &authenticatedStream{ServerStream: ss, ctx: ctx}

		return handler(srv, wrappedStream)
	}
}

type authenticatedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedStream) Context() context.Context {
	return s.ctx
}

func GetClaimsFromContext(ctx context.Context) *Claims {
	if claims, ok := ctx.Value(claimsContextKey).(*Claims); ok {
		return claims
	}
	return nil
}

type ClientAuthInterceptor struct {
	tokenProvider func() (string, error)
}

func NewClientAuthInterceptor(tokenProvider func() (string, error)) *ClientAuthInterceptor {
	return &ClientAuthInterceptor{
		tokenProvider: tokenProvider,
	}
}

func (c *ClientAuthInterceptor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		token, err := c.tokenProvider()
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "failed to get auth token: %v", err)
		}

		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func (c *ClientAuthInterceptor) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

		token, err := c.tokenProvider()
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "failed to get auth token: %v", err)
		}

		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)

		return streamer(ctx, desc, cc, method, opts...)
	}
}

func NewAuthenticatedClientConn(target string, tokenProvider func() (string, error), opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	interceptor := NewClientAuthInterceptor(tokenProvider)

	opts = append(opts,
		grpc.WithUnaryInterceptor(interceptor.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(interceptor.StreamClientInterceptor()),
	)

	return grpc.NewClient(target, opts...)
}

func NewTokenProviderFromCredentials(credType, token, apiKey, username, password string) (func() (string, error), error) {
	switch credType {
	case "bearer":
		if token == "" {
			return nil, fmt.Errorf("bearer token is required")
		}

		t := token
		return func() (string, error) {
			return t, nil
		}, nil

	case "api_key":
		if apiKey == "" {
			return nil, fmt.Errorf("api_key is required")
		}

		k := apiKey
		return func() (string, error) {
			return k, nil
		}, nil

	case "basic":
		if username == "" || password == "" {
			return nil, fmt.Errorf("username and password are required for basic auth")
		}

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
