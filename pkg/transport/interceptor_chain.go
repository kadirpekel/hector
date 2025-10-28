package transport

import (
	"context"

	"google.golang.org/grpc"
)

// ChainUnaryInterceptors chains multiple unary interceptors into one.
// Interceptors are executed in the order they are provided (first to last).
func ChainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	// Filter out nil interceptors
	filtered := make([]grpc.UnaryServerInterceptor, 0, len(interceptors))
	for _, i := range interceptors {
		if i != nil {
			filtered = append(filtered, i)
		}
	}

	n := len(filtered)

	if n == 0 {
		return nil
	}

	if n == 1 {
		return filtered[0]
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Build the chain from the handler backwards
		chainHandler := handler
		for i := n - 1; i >= 0; i-- {
			// Capture the current interceptor and handler
			currentInterceptor := filtered[i]
			currentHandler := chainHandler

			// Create new handler that calls current interceptor
			chainHandler = func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return currentInterceptor(currentCtx, currentReq, info, currentHandler)
			}
		}

		// Execute the chain
		return chainHandler(ctx, req)
	}
}

// ChainStreamInterceptors chains multiple stream interceptors into one.
// Interceptors are executed in the order they are provided (first to last).
func ChainStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	// Filter out nil interceptors
	filtered := make([]grpc.StreamServerInterceptor, 0, len(interceptors))
	for _, i := range interceptors {
		if i != nil {
			filtered = append(filtered, i)
		}
	}

	n := len(filtered)

	if n == 0 {
		return nil
	}

	if n == 1 {
		return filtered[0]
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Build the chain from the handler backwards
		chainHandler := handler
		for i := n - 1; i >= 0; i-- {
			// Capture the current interceptor and handler
			currentInterceptor := filtered[i]
			currentHandler := chainHandler

			// Create new handler that calls current interceptor
			chainHandler = func(currentSrv interface{}, currentStream grpc.ServerStream) error {
				return currentInterceptor(currentSrv, currentStream, info, currentHandler)
			}
		}

		// Execute the chain
		return chainHandler(srv, ss)
	}
}
