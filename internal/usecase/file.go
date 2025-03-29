package usecase

import (
	"context"
	"io"
	"time"

	"github.com/keenoobi/grpc-file-manager/internal/entity"
	"github.com/keenoobi/grpc-file-manager/internal/repository"
)

type FileUseCase interface {
	UploadFile(ctx context.Context, filename string, data io.Reader) (*entity.File, error)
	DownloadFile(ctx context.Context, filename string) (*entity.File, io.ReadCloser, error)
	ListFiles(ctx context.Context) ([]*entity.File, error)
}

type fileUseCase struct {
	repo repository.FileRepository
}

func NewFileUseCase(repo repository.FileRepository) FileUseCase {
	return &fileUseCase{repo: repo}
}

func (uc *fileUseCase) UploadFile(ctx context.Context, filename string, data io.Reader) (*entity.File, error) {
	file := &entity.File{
		Name:      filename,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := uc.repo.Save(ctx, file, data); err != nil {
		return nil, err
	}

	return file, nil
}

func (uc *fileUseCase) DownloadFile(ctx context.Context, filename string) (*entity.File, io.ReadCloser, error) {
	return uc.repo.Get(ctx, filename)
}

func (uc *fileUseCase) ListFiles(ctx context.Context) ([]*entity.File, error) {
	return uc.repo.List(ctx)
}
