package client

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryClientInterceptor returns a grpc.UnaryClientInterceptor that retries requests on transient errors.
func UnaryClientInterceptor(maxRetries int, attemptTimeout time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var err error
		for attempt := 0; attempt < maxRetries; attempt++ {
			if attempt > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(50 * time.Millisecond):
				}
			}

			var attemptCtx context.Context
			var cancel context.CancelFunc
			if attemptTimeout > 0 {
				attemptCtx, cancel = context.WithTimeout(ctx, attemptTimeout)
			} else {
				attemptCtx, cancel = context.WithCancel(ctx)
			}

			// Extract outgoing metadata from parent context and attach to the new context
			if md, ok := metadata.FromOutgoingContext(ctx); ok {
				attemptCtx = metadata.NewOutgoingContext(attemptCtx, md)
			}

			err = invoker(attemptCtx, method, req, reply, cc, opts...)
			cancel()

			if err == nil {
				return nil
			}

			// Only retry on transient errors (e.g., Unavailable)
			st, ok := status.FromError(err)
			if !ok || st.Code() != codes.Unavailable {
				return err
			}
		}
		return err
	}
}