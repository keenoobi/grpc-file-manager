package repository

import (
	"context"
	"io"
	"os"
	"path/filepath"

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
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	size, err := io.Copy(f, data)
	if err != nil {
		return err
	}

	file.Size = size
	file.Path = path
	return nil
}

func (r *fileRepository) Get(ctx context.Context, filename string) (*entity.File, io.ReadCloser, error) {
	path := filepath.Join(r.storagePath, filename)
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

	return files, nil
}
