// internal/usecase/file_test.go
package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/keenoobi/grpc-file-manager/internal/entity"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockFileRepository struct {
	mock.Mock
}

func (m *MockFileRepository) Save(ctx context.Context, file *entity.File, data io.Reader) error {
	args := m.Called(ctx, file, data)
	return args.Error(0)
}

func (m *MockFileRepository) Get(ctx context.Context, filename string) (*entity.File, io.ReadCloser, error) {
	args := m.Called(ctx, filename)
	return args.Get(0).(*entity.File), args.Get(1).(io.ReadCloser), args.Error(2)
}

func (m *MockFileRepository) List(ctx context.Context) ([]*entity.File, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entity.File), args.Error(1)
}

func TestFileUseCase_UploadFile(t *testing.T) {
	mockRepo := new(MockFileRepository)
	uc := NewFileUseCase(mockRepo)
	ctx := context.Background()

	t.Run("valid file", func(t *testing.T) {
		data := bytes.NewReader([]byte("data"))
		mockRepo.On("Save", ctx, mock.Anything, data).Return(nil)

		file, err := uc.UploadFile(ctx, "valid.txt", data)
		require.NoError(t, err)
		require.Equal(t, "valid.txt", file.Name)
	})

	t.Run("invalid filename", func(t *testing.T) {
		_, err := uc.UploadFile(ctx, "../invalid.txt", bytes.NewReader([]byte("data")))
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid filename")
	})
}

func TestFileUseCase_UploadError(t *testing.T) {
	mockRepo := new(MockFileRepository)
	uc := NewFileUseCase(mockRepo)
	ctx := context.Background()

	// Репозиторий возвращает ошибку
	mockRepo.On("Save", ctx, mock.Anything, mock.Anything).Return(fmt.Errorf("disk error"))

	_, err := uc.UploadFile(ctx, "test.txt", bytes.NewReader([]byte("data")))
	require.Error(t, err)
	require.Contains(t, err.Error(), "disk error")
}
