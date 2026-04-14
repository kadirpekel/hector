package trigger

import "context"

// contextKey is a custom type to avoid collisions in context values.
type contextKey string

const (
	// contextIDKey is the key for storing context/session ID in context.
	contextIDKey contextKey = "trigger:context_id"
)

// WithContextID returns a new context with the given context ID stored.
// This ID will be used as the A2A Message.ContextID for session continuity.
func WithContextID(ctx context.Context, contextID string) context.Context {
	return context.WithValue(ctx, contextIDKey, contextID)
}

// ContextIDFromContext retrieves the context ID from the context.
// Returns empty string if not set.
func ContextIDFromContext(ctx context.Context) string {
	if v := ctx.Value(contextIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
