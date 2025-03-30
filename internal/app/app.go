package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/keenoobi/grpc-file-manager/api/proto"
	"github.com/keenoobi/grpc-file-manager/config"
	"github.com/keenoobi/grpc-file-manager/internal/middleware"
	"github.com/keenoobi/grpc-file-manager/internal/repository"
	grpctransport "github.com/keenoobi/grpc-file-manager/internal/transport/grpc"
	"github.com/keenoobi/grpc-file-manager/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type App struct {
	GRPCServer *grpc.Server
	config     *config.Config
}

func New(cfg *config.Config) *App {

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

	return &App{
		GRPCServer: grpcServer,
		config:     cfg,
	}
}

func (a *App) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", a.config.Server.Port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	slog.Info("Server starting",
		"port", a.config.Server.Port,
		"storage_path", a.config.Storage.Path)

	go func() {
		<-ctx.Done()
		a.GRPCServer.GracefulStop()
	}()

	if err := a.GRPCServer.Serve(listener); err != nil && err != grpc.ErrServerStopped {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}
