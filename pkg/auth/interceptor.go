// Copyright 2025 Kadir Pekel
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"context"

	"github.com/a2aproject/a2a-go/a2asrv"
)

// Interceptor bridges Hector's auth system to a2a-go's CallInterceptor.
// It reads Claims from the HTTP context (set by Middleware) and
// sets the a2a-go CallContext.User field.
//
// # Architecture
//
// The auth flow is:
//
//  1. HTTP request arrives
//  2. auth.Middleware validates JWT and stores Claims in context
//  3. a2a-go JSON-RPC handler processes the request
//  4. auth.Interceptor.Before() reads Claims and sets CallContext.User
//  5. Executor/Agent can access User via CallContext
//
// This design keeps JWT validation in standard HTTP middleware while
// bridging to a2a-go's authentication system.
type Interceptor struct {
	// RequireAuth when true rejects unauthenticated requests.
	// When false, unauthenticated requests proceed with a nil User.
	RequireAuth bool
}

// NewInterceptor creates a new auth interceptor.
func NewInterceptor(requireAuth bool) *Interceptor {
	return &Interceptor{
		RequireAuth: requireAuth,
	}
}

// Before is called before each a2a-go request handler method.
// It bridges Claims from HTTP context to a2a-go's User interface.
func (i *Interceptor) Before(ctx context.Context, callCtx *a2asrv.CallContext, req *a2asrv.Request) (context.Context, error) {
	// Get claims from HTTP context (set by auth.Middleware)
	claims := ClaimsFromContext(ctx)

	if claims != nil {
		// Set authenticated user on a2a-go CallContext
		callCtx.User = &AuthenticatedUser{claims: claims}
	} else if i.RequireAuth {
		// This shouldn't happen if HTTP middleware is properly configured,
		// but we check as a safety net.
		return ctx, ErrUnauthorized
	}

	return ctx, nil
}

// After is called after each a2a-go request handler method.
// Currently a no-op but can be extended for audit logging.
func (i *Interceptor) After(ctx context.Context, callCtx *a2asrv.CallContext, resp *a2asrv.Response) error {
	return nil
}

// Ensure Interceptor implements a2asrv.CallInterceptor
var _ a2asrv.CallInterceptor = (*Interceptor)(nil)

// AuthenticatedUser implements a2asrv.User interface.
// It wraps Hector's Claims to provide user information to a2a-go.
type AuthenticatedUser struct {
	claims *Claims
}

// Name returns the user's subject (unique identifier).
func (u *AuthenticatedUser) Name() string {
	if u.claims == nil {
		return ""
	}
	return u.claims.Subject
}

// Authenticated returns true since this represents an authenticated user.
func (u *AuthenticatedUser) Authenticated() bool {
	return true
}

// Claims returns the underlying Hector claims.
// This allows access to full claim data when needed.
func (u *AuthenticatedUser) Claims() *Claims {
	return u.claims
}

// Email returns the user's email address.
func (u *AuthenticatedUser) Email() string {
	if u.claims == nil {
		return ""
	}
	return u.claims.Email
}

// Role returns the user's role.
func (u *AuthenticatedUser) Role() string {
	if u.claims == nil {
		return ""
	}
	return u.claims.Role
}

// TenantID returns the user's tenant ID.
func (u *AuthenticatedUser) TenantID() string {
	if u.claims == nil {
		return ""
	}
	return u.claims.TenantID
}

// Ensure AuthenticatedUser implements a2asrv.User
var _ a2asrv.User = (*AuthenticatedUser)(nil)

// UserFromCallContext extracts the AuthenticatedUser from a2a-go CallContext.
// Returns nil if the user is not authenticated or not an AuthenticatedUser.
func UserFromCallContext(callCtx *a2asrv.CallContext) *AuthenticatedUser {
	if callCtx == nil || callCtx.User == nil {
		return nil
	}
	if user, ok := callCtx.User.(*AuthenticatedUser); ok {
		return user
	}
	return nil
}

// ClaimsFromCallContext extracts Claims from a2a-go CallContext.
// Returns nil if the user is not authenticated.
func ClaimsFromCallContext(callCtx *a2asrv.CallContext) *Claims {
	user := UserFromCallContext(callCtx)
	if user == nil {
		return nil
	}
	return user.Claims()
}
