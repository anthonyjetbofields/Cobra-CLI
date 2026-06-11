package client

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type mockService struct {
	attempts int
	metadata []metadata.MD
}

func (s *mockService) TestCall(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	s.attempts++
	md, _ := metadata.FromIncomingContext(ctx)
	s.metadata = append(s.metadata, md.Copy())

	if s.attempts < 3 {
		return nil, status.Error(codes.Unavailable, "transient error")
	}
	return &emptypb.Empty{}, nil
}

func TestRetryInterceptorMetadataPropagation(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	mockSvc := &mockService{}

	serviceDesc := grpc.ServiceDesc{
		ServiceName: "test.MockService",
		HandlerType: (*mockService)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "TestCall",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					in := new(emptypb.Empty)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(*mockService).TestCall(ctx, in)
					}
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: "/test.MockService/TestCall",
					}
					return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
						return srv.(*mockService).TestCall(ctx, req.(*emptypb.Empty))
					})
				},
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "mock.proto",
	}
	s.RegisterService(&serviceDesc, mockSvc)

	go func() {
		_ = s.Serve(lis)
	}()
	defer s.Stop()

	retryInterceptor := UnaryClientInterceptor(3, 1*time.Second)
	conn, err := grpc.Dial(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(retryInterceptor),
	)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	ctx := metadata.AppendToOutgoingContext(context.Background(), "custom-header", "test-value")

	var reply emptypb.Empty
	err = conn.Invoke(ctx, "/test.MockService/TestCall", &emptypb.Empty{}, &reply)
	if err != nil {
		t.Fatalf("expected call to succeed, got: %v", err)
	}

	if mockSvc.attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", mockSvc.attempts)
	}

	for i, md := range mockSvc.metadata {
		vals := md.Get("custom-header")
		if len(vals) == 0 || vals[0] != "test-value" {
			t.Errorf("attempt %d: expected custom-header to be 'test-value', got %v", i+1, vals)
		}
	}
}