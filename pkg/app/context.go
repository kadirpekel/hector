package app

import "context"

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// AppContextKey is the context key for storing the current app.
	AppContextKey contextKey = "hector_app"
)

// FromContext extracts the current app from the context.
// Returns nil if no app is set.
func FromContext(ctx context.Context) *App {
	if app, ok := ctx.Value(AppContextKey).(*App); ok {
		return app
	}
	return nil
}

// IDFromContext extracts the app ID from the context.
// Returns "default" if no app is set.
func IDFromContext(ctx context.Context) string {
	if app := FromContext(ctx); app != nil {
		return app.ID
	}
	return ""
}

// WithApp returns a new context with the given app.
func WithApp(ctx context.Context, app *App) context.Context {
	return context.WithValue(ctx, AppContextKey, app)
}

// WithAppID returns a new context with a minimal app containing just the ID.
// This is useful when you only have the app ID (e.g., from auth claims).
func WithAppID(ctx context.Context, appID string) context.Context {
	return WithApp(ctx, &App{ID: appID})
}
