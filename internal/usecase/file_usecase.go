package usecase

import (
	"context"
	"fmt"
	"io"
	"strings"
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
	if !isValidFilename(filename) {
		return nil, fmt.Errorf("invalid filename")
	}

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

func isValidFilename(filename string) bool {
	if filename == "" || len(filename) > 255 {
		return false
	}
	// Запрещаем: ../, ~/, /, \
	if strings.Contains(filename, "..") || strings.ContainsAny(filename, `/\~`) {
		return false
	}
	// Проверяем на недопустимые символы (например, управляющие символы ASCII)
	for _, r := range filename {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
