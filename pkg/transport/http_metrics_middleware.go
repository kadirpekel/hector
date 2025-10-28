package transport

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kadirpekel/hector/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// responseWriter wraps http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Flush implements http.Flusher interface for SSE streaming support
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// metricsMiddleware adds HTTP observability with Prometheus metrics and OpenTelemetry tracing
// Now uses chi router's RouteContext to get the pattern - NO REGEX MATCHING NEEDED!
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Create span for HTTP request
		tracer := observability.GetTracer("hector.http")
		ctx, span := tracer.Start(r.Context(), "http.request",
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.path", r.URL.Path),
				attribute.String("http.scheme", r.URL.Scheme),
				attribute.String("http.host", r.Host),
				attribute.String("http.user_agent", r.UserAgent()),
			),
		)
		defer span.End()

		// Update request context with span context
		r = r.WithContext(ctx)

		// Wrap response writer to capture status and size
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default if WriteHeader not called
			size:           0,
		}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(startTime)

		// Determine if this was an error
		isError := wrapped.statusCode >= 400

		// Add response attributes to span
		span.SetAttributes(
			attribute.Int("http.status_code", wrapped.statusCode),
			attribute.Int("http.response_size", wrapped.size),
			attribute.Int64("http.duration_ms", duration.Milliseconds()),
		)

		// Set span status based on HTTP status
		if isError {
			if wrapped.statusCode >= 500 {
				span.SetStatus(codes.Error, http.StatusText(wrapped.statusCode))
			} else {
				span.SetStatus(codes.Ok, http.StatusText(wrapped.statusCode))
			}
		} else {
			span.SetStatus(codes.Ok, "success")
		}

		// Get route pattern from chi router - THIS IS THE KEY BENEFIT!
		// No regex matching, no cache, just get it from the router
		routePattern := getRoutePattern(r)

		// Record Prometheus metrics
		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			metrics.RecordHTTPRequest(ctx, r.Method, routePattern, wrapped.statusCode, duration, wrapped.size)
		}
	})
}

// getRoutePattern extracts the route pattern from chi's RouteContext
// Returns the actual matched pattern like "/v1/agents/{agent}/message:send"
// Falls back to the raw path if chi context is not available
func getRoutePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}
	// Fallback to raw path (for non-chi routes or during testing)
	return r.URL.Path
}
