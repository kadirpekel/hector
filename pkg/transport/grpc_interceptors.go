package transport

import (
	"context"
	"time"

	"github.com/kadirpekel/hector/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor with observability
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		startTime := time.Now()

		// Create span for gRPC unary call
		tracer := observability.GetTracer("hector.grpc")
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", extractServiceName(info.FullMethod)),
				attribute.String("rpc.method", extractMethodName(info.FullMethod)),
			),
		)
		defer span.End()

		// Call the handler
		resp, err := handler(ctx, req)
		duration := time.Since(startTime)

		// Get gRPC status
		grpcStatus, _ := status.FromError(err)
		statusCode := grpcStatus.Code()

		// Add response attributes to span
		span.SetAttributes(
			attribute.String("rpc.grpc.status_code", statusCode.String()),
			attribute.Int64("rpc.duration_ms", duration.Milliseconds()),
		)

		// Set span status based on gRPC status
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, grpcStatus.Message())
		} else {
			span.SetStatus(codes.Ok, "success")
		}

		// Record Prometheus metrics
		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			service := extractServiceName(info.FullMethod)
			method := extractMethodName(info.FullMethod)
			metrics.RecordGRPCCall(ctx, service, method, statusCode.String(), duration, err)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor with observability
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		startTime := time.Now()

		// Create span for gRPC stream call
		tracer := observability.GetTracer("hector.grpc")
		ctx, span := tracer.Start(ss.Context(), info.FullMethod,
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", extractServiceName(info.FullMethod)),
				attribute.String("rpc.method", extractMethodName(info.FullMethod)),
				attribute.Bool("rpc.is_client_stream", info.IsClientStream),
				attribute.Bool("rpc.is_server_stream", info.IsServerStream),
			),
		)
		defer span.End()

		// Wrap the stream with context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Call the handler
		err := handler(srv, wrappedStream)
		duration := time.Since(startTime)

		// Get gRPC status
		grpcStatus, _ := status.FromError(err)
		statusCode := grpcStatus.Code()

		// Add response attributes to span
		span.SetAttributes(
			attribute.String("rpc.grpc.status_code", statusCode.String()),
			attribute.Int64("rpc.duration_ms", duration.Milliseconds()),
		)

		// Set span status based on gRPC status
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, grpcStatus.Message())
		} else {
			span.SetStatus(codes.Ok, "success")
		}

		// Record Prometheus metrics
		metrics := observability.GetGlobalMetrics()
		if metrics != nil {
			service := extractServiceName(info.FullMethod)
			method := extractMethodName(info.FullMethod)
			metrics.RecordGRPCCall(ctx, service, method, statusCode.String(), duration, err)
		}

		return err
	}
}

// wrappedServerStream wraps grpc.ServerStream to inject context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// extractServiceName extracts service name from full method
// Example: "/hector.A2AService/SendMessage" -> "hector.A2AService"
func extractServiceName(fullMethod string) string {
	if len(fullMethod) == 0 {
		return "unknown"
	}
	// Remove leading slash
	if fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}
	// Split by slash to get service/method
	for i := 0; i < len(fullMethod); i++ {
		if fullMethod[i] == '/' {
			return fullMethod[:i]
		}
	}
	return fullMethod
}

// extractMethodName extracts method name from full method
// Example: "/hector.A2AService/SendMessage" -> "SendMessage"
func extractMethodName(fullMethod string) string {
	if len(fullMethod) == 0 {
		return "unknown"
	}
	// Find last slash
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[i+1:]
		}
	}
	return fullMethod
}
