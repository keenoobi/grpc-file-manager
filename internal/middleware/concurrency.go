package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ConcurrencyLimiter struct {
	uploadDownloadSem chan struct{}
	listSem           chan struct{}
}

func NewConcurrencyLimiter(uploadDownloadLimit, listLimit int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		uploadDownloadSem: make(chan struct{}, uploadDownloadLimit),
		listSem:           make(chan struct{}, listLimit),
	}
}

func (l *ConcurrencyLimiter) UnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	if info.FullMethod != "/file_service.FileService/ListFiles" {
		return handler(ctx, req) // Ограничения только для ListFiles
	}

	select {
	case l.listSem <- struct{}{}:
		defer func() { <-l.listSem }()
		return handler(ctx, req)
	case <-ctx.Done():
		return nil, status.FromContextError(ctx.Err()).Err()
	default:
		return nil, status.Errorf(codes.ResourceExhausted, "too many concurrent ListFiles requests")
	}
}

func (l *ConcurrencyLimiter) StreamInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	if info.FullMethod != "/file_service.FileService/UploadFile" &&
		info.FullMethod != "/file_service.FileService/DownloadFile" {
		return handler(srv, ss) // Ограничения только для UploadFile/DownloadFile
	}

	select {
	case l.uploadDownloadSem <- struct{}{}:
		defer func() { <-l.uploadDownloadSem }()
		return handler(srv, ss)
	case <-ss.Context().Done():
		return status.FromContextError(ss.Context().Err()).Err()
	default:
		return status.Errorf(codes.ResourceExhausted, "too many concurrent UploadFile/DownloadFile requests")
	}
}
