package middleware_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/keenoobi/grpc-file-manager/internal/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockStream) Context() context.Context {
	return m.ctx
}

func TestConcurrencyLimiter(t *testing.T) {
	t.Run("Unary ListFiles limit", func(t *testing.T) {
		const limit = 3
		const requests = 5
		limiter := middleware.NewConcurrencyLimiter(2, limit)

		var wg sync.WaitGroup
		var errCount int32
		var mu sync.Mutex

		for range requests {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := limiter.UnaryInterceptor(
					context.Background(),
					nil,
					&grpc.UnaryServerInfo{FullMethod: "/file_service.FileService/ListFiles"},
					func(ctx context.Context, req any) (any, error) {
						time.Sleep(100 * time.Millisecond)
						return nil, nil
					},
				)

				if err != nil && status.Code(err) == codes.ResourceExhausted {
					mu.Lock()
					errCount++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		expected := requests - limit
		if errCount != int32(expected) {
			t.Errorf("Expected %d rejected requests, got %d", expected, errCount)
		}
	})

	t.Run("Stream UploadFile limit", func(t *testing.T) {
		const limit = 2
		const requests = 4
		limiter := middleware.NewConcurrencyLimiter(limit, 3)

		var wg sync.WaitGroup
		var errCount int32
		var mu sync.Mutex

		for range requests {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := limiter.StreamInterceptor(
					nil,
					&mockStream{ctx: context.Background()},
					&grpc.StreamServerInfo{FullMethod: "/file_service.FileService/UploadFile"},
					func(srv any, stream grpc.ServerStream) error {
						time.Sleep(100 * time.Millisecond)
						return nil
					},
				)

				if err != nil && status.Code(err) == codes.ResourceExhausted {
					mu.Lock()
					errCount++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		expected := requests - limit
		if errCount != int32(expected) {
			t.Errorf("Expected %d rejected requests, got %d", expected, errCount)
		}
	})

	t.Run("Different methods use different limits", func(t *testing.T) {
		limiter := middleware.NewConcurrencyLimiter(1, 1)

		listDone := make(chan struct{})
		go func() {
			limiter.UnaryInterceptor(
				context.Background(),
				nil,
				&grpc.UnaryServerInfo{FullMethod: "/file_service.FileService/ListFiles"},
				func(ctx context.Context, req any) (any, error) {
					<-listDone
					return nil, nil
				},
			)
		}()

		time.Sleep(100 * time.Millisecond)

		uploadDone := make(chan struct{})
		var uploadErr error
		go func() {
			uploadErr = limiter.StreamInterceptor(
				nil,
				&mockStream{ctx: context.Background()},
				&grpc.StreamServerInfo{FullMethod: "/file_service.FileService/UploadFile"},
				func(srv any, stream grpc.ServerStream) error {
					<-uploadDone
					return nil
				},
			)
		}()

		time.Sleep(100 * time.Millisecond)

		if uploadErr != nil {
			t.Errorf("Expected UploadFile to succeed, got error: %v", uploadErr)
		}

		close(listDone)
		close(uploadDone)
	})
}
