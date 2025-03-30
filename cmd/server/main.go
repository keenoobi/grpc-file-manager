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
	"github.com/keenoobi/grpc-file-manager/config"
	"github.com/keenoobi/grpc-file-manager/internal/middleware"
	"github.com/keenoobi/grpc-file-manager/internal/repository"
	grpctransport "github.com/keenoobi/grpc-file-manager/internal/transport/grpc"
	"github.com/keenoobi/grpc-file-manager/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	shutdownTimeout = 5 * time.Second
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	repo := repository.NewFileRepository(cfg.Storage.Path)
	useCase := usecase.NewFileUseCase(repo)
	fileServiceServer := grpctransport.NewFileServiceServer(useCase)

	limiter := middleware.NewConcurrencyLimiter(cfg.Limits.Upload, cfg.Limits.List)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			limiter.UnaryInterceptor,
			middleware.LoggingUnaryInterceptor,
			middleware.RecoveryUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			limiter.StreamInterceptor,
			middleware.LoggingStreamInterceptor,
			middleware.RecoveryStreamInterceptor,
		),
	)

	proto.RegisterFileServiceServer(grpcServer, fileServiceServer)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", cfg.Server.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		log.Printf("Server started on %s, storage path: %s", cfg.Server.Port, cfg.Storage.Path)
		if err := grpcServer.Serve(listener); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	waitForShutdown(grpcServer)
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
