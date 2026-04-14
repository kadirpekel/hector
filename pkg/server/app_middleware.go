package server

import (
	"net/http"

	"github.com/verikod/hector/pkg/app"
	"github.com/verikod/hector/pkg/auth"
)

// AppContextMiddleware extracts tenant_id from auth claims and injects it
// into the request context as app context. This enables downstream handlers
// and stores to access the current app's namespace.
//
// The middleware should be applied after auth middleware so claims are available.
func AppContextMiddleware(appStore app.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get claims from auth middleware
			claims := auth.ClaimsFromContext(ctx)
			if claims != nil {
				// AUTH ENABLED: Strict Tenancy
				// App Token: Set App ID; Admin/Other Token: skip (handler falls back to "default").
				if claims.TenantID != "" {
					ctx = app.WithAppID(ctx, claims.TenantID)
				}
				r = r.WithContext(ctx)
			} else {
				// AUTH DISABLED: Local Dev Convenience
				// No auth configured, default to "default" app for better DX
				ctx = app.WithAppID(ctx, "default")
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// AppContextMiddlewareWithValidation is like AppContextMiddleware but also
// validates that the app exists in the store. Returns 403 if app not found.
func AppContextMiddlewareWithValidation(appStore app.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get claims from auth middleware
			claims := auth.ClaimsFromContext(ctx)
			if claims != nil {
				// AUTH ENABLED: Strict Tenancy
				if claims.TenantID != "" {
					// Validate app exists
					exists, err := appStore.Exists(ctx, claims.TenantID)
					if err != nil || !exists {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusForbidden)
						_, _ = w.Write([]byte(`{"error":"Invalid or unknown app"}`))
						return
					}

					// Set app context
					ctx = app.WithAppID(ctx, claims.TenantID)
				}
				// If claims exist but no TenantID (e.g. Admin Key),
				// we DO NOT default to anything. The request proceeds without App Context.
				r = r.WithContext(ctx)
			} else {
				// AUTH DISABLED: Local Dev Convenience
				ctx = app.WithAppID(ctx, "default")
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}
