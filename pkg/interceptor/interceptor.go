// Package interceptor provides gRPC server interceptors for resilience and
// gateway↔service authentication (defense-in-depth).
package interceptor

import (
	"context"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/malvin/iam-go/pkg/grpcutil"
)

// Recovery converts a panic in a handler into a gRPC Internal error instead of
// crashing the whole server process.
func Recovery() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				debug.PrintStack()
				err = status.Errorf(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

// exemptMethods may be called without the internal service token (health/reflection).
var exemptMethods = map[string]bool{
	"/grpc.health.v1.Health/Check": true,
	"/grpc.health.v1.Health/Watch": true,
}

// ServiceAuth requires that callers present the shared internal token in
// metadata. When token is empty, enforcement is disabled (local dev).
func ServiceAuth(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if token == "" || exemptMethods[info.FullMethod] {
			return handler(ctx, req)
		}
		md, _ := metadata.FromIncomingContext(ctx)
		if vals := md.Get(grpcutil.MDInternalToken); len(vals) == 1 && vals[0] == token {
			return handler(ctx, req)
		}
		return nil, status.Error(codes.Unauthenticated, "missing or invalid internal service token")
	}
}

// Chain composes interceptors into one (recovery outermost).
func Chain(token string) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(Recovery(), ServiceAuth(token))
}
