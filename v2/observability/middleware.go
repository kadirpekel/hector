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

package observability

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware creates HTTP middleware that records both traces and metrics.
func HTTPMiddleware(tracer *Tracer, metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Start span
			var span trace.Span
			ctx := r.Context()
			if tracer != nil {
				ctx, span = tracer.Start(ctx, SpanHTTPRequest,
					trace.WithAttributes(
						attribute.String(AttrHTTPMethod, r.Method),
						attribute.String(AttrHTTPPath, r.URL.Path),
					),
				)
				defer span.End()
			}

			// Wrap response writer to capture status code and size
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Get request size
			reqSize := r.ContentLength
			if reqSize < 0 {
				reqSize = 0
			}

			// Process request with updated context
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Calculate duration
			duration := time.Since(start)

			// Record span attributes
			if span != nil {
				span.SetAttributes(
					attribute.Int(AttrHTTPStatusCode, wrapped.statusCode),
					attribute.Int64(AttrHTTPResponseSize, int64(wrapped.bytesWritten)),
				)
				if wrapped.statusCode >= 400 {
					span.SetAttributes(attribute.String(AttrErrorType, fmt.Sprintf("HTTP %d", wrapped.statusCode)))
				}
			}

			// Record metrics
			if metrics != nil {
				metrics.RecordHTTPRequest(
					r.Method,
					r.URL.Path,
					wrapped.statusCode,
					duration,
					reqSize,
					int64(wrapped.bytesWritten),
				)
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and size.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func (w *responseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

// Hijack implements http.Hijacker.
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not implement http.Hijacker")
}

// Flush implements http.Flusher.
func (w *responseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// TracingMiddleware creates HTTP middleware that only records traces.
func TracingMiddleware(tracer *Tracer) func(http.Handler) http.Handler {
	return HTTPMiddleware(tracer, nil)
}

// MetricsMiddleware creates HTTP middleware that only records metrics.
func MetricsMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return HTTPMiddleware(nil, metrics)
}
