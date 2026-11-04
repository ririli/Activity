package grpc

import (
	"context"
	"crypto/subtle"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func isHealthMethod(method string) bool {
	return strings.HasPrefix(method, "/grpc.health.v1.Health/")
}

func internalTokenOK(ctx context.Context) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	values := md.Get("x-internal-token")
	if len(values) != 1 {
		return false
	}
	want := os.Getenv("INTERNAL_TOKEN")
	if len(want) < 32 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(values[0]), []byte(want)) == 1
}

// InternalAuth protects every business unary RPC with the shared internal token.
// Standard gRPC health checks are exempt so deployment probes can use them.
func InternalAuth() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if isHealthMethod(info.FullMethod) {
			return handler(ctx, req)
		}
		if !internalTokenOK(ctx) {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}
		return handler(ctx, req)
	}
}

// InternalAuthStream is the streaming equivalent, needed for health Watch.
func InternalAuthStream() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if isHealthMethod(info.FullMethod) {
			return handler(srv, stream)
		}
		if !internalTokenOK(stream.Context()) {
			return status.Error(codes.Unauthenticated, "unauthorized")
		}
		return handler(srv, stream)
	}
}
