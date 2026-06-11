package client

import (
    "context"
    "testing"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/test/bufconn"

    "github.com/stretchr/testify/assert"
)

func TestRetryInterceptorMetadataPropagation(t *testing.T) {
    // Create a mock gRPC server
    lis := bufconn.Listen(1024 * 1024)
    s := grpc.NewServer()
    defer s.Stop()

    // Register a mock service
    RegisterMockServiceServer(s, &mockService{})

    // Create a gRPC client with the retry interceptor
    conn, err := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
        return lis.Dial()
    }), grpc.WithUnaryInterceptor(retryInterceptor(2, 10*time.Second)))
    assert.NoError(t, err)
    defer conn.Close()

    // Create a client for the mock service
    client := NewMockServiceClient(conn)

    // Set up the mock server to return a transient error on the first request
    mockServiceError := true

    // Call the mock service with metadata
    ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("key", "value"))
    _, err = client.MockMethod(ctx, &MockRequest{})

    // Verify that the metadata was propagated to the mock server
    assert.NoError(t, err)
    assert.Len(t, mockServiceMetadata, 1)
    assert.Equal(t, "value", mockServiceMetadata[0].Get("key"))
}

type mockService struct{}

func (s *mockService) MockMethod(ctx context.Context, req *MockRequest) (*MockResponse, error) {
    // Record the metadata of the incoming request
    md, ok := metadata.FromIncomingContext(ctx)
    if ok {
        mockServiceMetadata = md
    }

    // Return a transient error on the first request
    if mockServiceError {
        mockServiceError = false
        return nil, grpc.Errorf(codes.Unavailable, "transient error")
    }

    return &MockResponse{}, nil
}

var mockServiceMetadata metadata.MD