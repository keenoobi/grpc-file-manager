// internal/repository/file_test.go
package repository

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/keenoobi/grpc-file-manager/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestFileRepository_SaveAndGet(t *testing.T) {
	repo := NewFileRepository(t.TempDir())
	ctx := context.Background()

	t.Run("success save and get", func(t *testing.T) {
		data := []byte("test data")
		file := &entity.File{Name: "test.txt", CreatedAt: time.Now()}

		// Сохраняем файл
		err := repo.Save(ctx, file, bytes.NewReader(data))
		require.NoError(t, err)

		// Получаем файл
		savedFile, reader, err := repo.Get(ctx, "test.txt")
		require.NoError(t, err)
		defer reader.Close()

		// Проверяем содержимое
		readData, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, data, readData)
		require.Equal(t, file.Name, savedFile.Name)
	})

	t.Run("file not found", func(t *testing.T) {
		_, _, err := repo.Get(ctx, "non_existent.txt")
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})
}

func TestFileRepository_Concurrency(t *testing.T) {
	repo := NewFileRepository(t.TempDir())
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			filename := fmt.Sprintf("file_%d.txt", i)
			data := []byte(fmt.Sprintf("content %d", i))
			err := repo.Save(ctx, &entity.File{Name: filename}, bytes.NewReader(data))
			require.NoError(t, err)
		}(i)
	}
	wg.Wait()

	files, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, files, 10)
}
