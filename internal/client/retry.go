package client

import (
    "context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
)

// retryInterceptor returns a UnaryClientInterceptor that retries failed requests.
func retryInterceptor(maxRetries int, attemptTimeout time.Duration) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        for attempt := 0; attempt <= maxRetries; attempt++ {
            ctx, cancel := context.WithTimeout(ctx, attemptTimeout)
            defer cancel()

            // Extract outgoing metadata from parent context and attach to the new context
            if md, ok := metadata.FromOutgoingContext(ctx); ok {
                ctx = metadata.NewOutgoingContext(ctx, md)
            }

            err := invoker(ctx, method, req, reply, cc, opts...)
            if err == nil {
                return nil
            }
            if status.Code(err) != codes.Unavailable {
                return err
            }
        }
        return grpc.Errorf(codes.Unavailable, "all retries failed")
    }
}