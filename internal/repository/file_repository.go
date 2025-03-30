package repository

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/keenoobi/grpc-file-manager/internal/entity"
)

type FileRepository interface {
	Save(ctx context.Context, file *entity.File, data io.Reader) error
	Get(ctx context.Context, filename string) (*entity.File, io.ReadCloser, error)
	List(ctx context.Context) ([]*entity.File, error)
}

type fileRepository struct {
	storagePath string
}

func NewFileRepository(storagePath string) FileRepository {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		panic(err)
	}
	return &fileRepository{storagePath: storagePath}
}

func (r *fileRepository) Save(ctx context.Context, file *entity.File, data io.Reader) error {
	path := filepath.Join(r.storagePath, file.Name)

	// Создаем временный файл
	tempPath := path + ".tmp"
	f, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("create temp file failed: %w", err)
	}

	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	size, err := io.Copy(f, data)
	if err != nil {
		f.Close()
		return fmt.Errorf("write failed: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("close failed: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename failed: %w", err)
	}

	file.Size = size
	file.Path = path
	return nil
}

func (r *fileRepository) Get(ctx context.Context, filename string) (*entity.File, io.ReadCloser, error) {
	path := filepath.Join(r.storagePath, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil, err // Возвращаем оригинальную ошибку
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}

	file := &entity.File{
		Name:      filename,
		Size:      info.Size(),
		CreatedAt: info.ModTime(),
		UpdatedAt: info.ModTime(),
		Path:      path,
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	return file, f, nil
}

func (r *fileRepository) List(ctx context.Context) ([]*entity.File, error) {
	entries, err := os.ReadDir(r.storagePath)
	if err != nil {
		return nil, err
	}

	var files []*entity.File
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, &entity.File{
			Name:      entry.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
			UpdatedAt: info.ModTime(),
			Path:      filepath.Join(r.storagePath, entry.Name()),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedAt.After(files[j].CreatedAt)
	})

	return files, nil
}
