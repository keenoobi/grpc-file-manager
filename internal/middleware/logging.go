package middleware

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

func LoggingUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	start := time.Now()
	slog.Info("Unary call started", "method", info.FullMethod, "request", req)

	resp, err = handler(ctx, req)

	slog.Info("Unary call completed", "method", info.FullMethod, "duration", time.Since(start), "error", err)
	return
}

func LoggingStreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	slog.Info("Stream call started", "method", info.FullMethod)

	err := handler(srv, ss)

	slog.Info("Stream call completed", "method", info.FullMethod, "duration", time.Since(start), "error", err)
	return err
}
