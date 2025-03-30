package middleware

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

func LoggingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	start := time.Now()
	log.Printf("Unary call: %s, request: %v", info.FullMethod, req)

	resp, err = handler(ctx, req)

	log.Printf("Unary call completed: %s, duration: %v, error: %v",
		info.FullMethod, time.Since(start), err)
	return
}

func LoggingStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	log.Printf("Stream call started: %s", info.FullMethod)

	err := handler(srv, ss)

	log.Printf("Stream call completed: %s, duration: %v, error: %v",
		info.FullMethod, time.Since(start), err)
	return err
}
