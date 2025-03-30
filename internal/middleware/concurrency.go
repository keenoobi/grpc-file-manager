package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ConcurrencyLimiter struct {
	uploadSem chan struct{}
	listSem   chan struct{}
}

func NewConcurrencyLimiter(uploadLimit, listLimit int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		uploadSem: make(chan struct{}, uploadLimit),
		listSem:   make(chan struct{}, listLimit),
	}
}

func (l *ConcurrencyLimiter) UnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Определяем какой лимит применять
	var sem chan struct{}
	if info.FullMethod == "/file_service.FileService/ListFiles" {
		sem = l.listSem
	} else {
		return handler(ctx, req) // Для остальных unary-методов лимит не применяем
	}

	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
		return handler(ctx, req)
	default:
		return nil, status.Errorf(codes.ResourceExhausted, "too many concurrent list requests")
	}
}

func (l *ConcurrencyLimiter) StreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// Применяем только для Upload/Download
	if info.FullMethod != "/file_service.FileService/UploadFile" &&
		info.FullMethod != "/file_service.FileService/DownloadFile" {
		return handler(srv, ss)
	}

	select {
	case l.uploadSem <- struct{}{}:
		defer func() { <-l.uploadSem }()
		return handler(srv, ss)
	default:
		return status.Errorf(codes.ResourceExhausted, "too many concurrent upload/download requests")
	}
}
