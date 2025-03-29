package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keenoobi/grpc-file-manager/api/proto"
	"github.com/keenoobi/grpc-file-manager/internal/repository"
	grpctransport "github.com/keenoobi/grpc-file-manager/internal/transport/grpc"
	"github.com/keenoobi/grpc-file-manager/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

const (
	defaultStoragePath = "./storage"
	defaultPort        = ":50051"
	shutdownTimeout    = 5 * time.Second
)

func main() {
	// 1. Configuration setup
	storagePath := getEnv("STORAGE_PATH", defaultStoragePath)
	port := getEnv("PORT", defaultPort)

	// 2. Initialize dependencies
	repo := repository.NewFileRepository(storagePath)
	useCase := usecase.NewFileUseCase(repo)
	fileServiceServer := grpctransport.NewFileServiceServer(useCase)

	// 3. Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			loggingUnaryInterceptor,
			recoveryUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			loggingStreamInterceptor,
			recoveryStreamInterceptor,
		),
	)

	// 4. Register services
	proto.RegisterFileServiceServer(grpcServer, fileServiceServer)
	reflection.Register(grpcServer) // Enable gRPC reflection for testing

	// 5. Start server with graceful shutdown
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		log.Printf("Server started on %s, storage path: %s", port, storagePath)
		if err := grpcServer.Serve(listener); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 6. Graceful shutdown handling
	waitForShutdown(grpcServer)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func waitForShutdown(server *grpc.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		log.Println("Forcing shutdown after timeout...")
		server.Stop()
	case <-stopped:
		log.Println("Server stopped gracefully")
	}
}

// Interceptors for logging and recovery
func loggingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	start := time.Now()
	log.Printf("Unary call: %s, request: %v", info.FullMethod, req)

	resp, err = handler(ctx, req)

	log.Printf("Unary call completed: %s, duration: %v, error: %v",
		info.FullMethod, time.Since(start), err)
	return
}

func loggingStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	log.Printf("Stream call started: %s", info.FullMethod)

	err := handler(srv, ss)

	log.Printf("Stream call completed: %s, duration: %v, error: %v",
		info.FullMethod, time.Since(start), err)
	return err
}

func recoveryUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in unary handler: %v", r)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()

	return handler(ctx, req)
}

func recoveryStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in stream handler: %v", r)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()

	return handler(srv, ss)
}
