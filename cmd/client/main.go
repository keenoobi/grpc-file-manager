package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/keenoobi/grpc-file-manager/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Подключение к серверу
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := proto.NewFileServiceClient(conn)

	// Создаем тестовую директорию
	if err := os.MkdirAll("test_data", 0755); err != nil {
		log.Fatalf("Failed to create test directory: %v", err)
	}

	// Тест базовых операций
	runTest("Basic Operations", func() {
		testBasicOperations(client)
	})

	// Тест обработки ошибок
	runTest("Error Handling", func() {
		testErrorHandling(client)
	})

	// Тест конкурентности
	runTest("Concurrency", func() {
		testConcurrency(client)
	})

	// Тест больших файлов
	runTest("Large Files", func() {
		testLargeFiles(client)
	})

	log.Println("All tests completed successfully!")
}

func runTest(name string, testFunc func()) {
	log.Printf("=== Starting test: %s ===", name)
	start := time.Now()
	testFunc()
	log.Printf("=== Test '%s' completed in %v ===\n", name, time.Since(start))
}

// Создает тестовое изображение указанного формата и размера
func createTestImage(filename, format string, width, height int) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Заполняем изображение градиентом
	for y := range height {
		for x := range width {
			c := color.RGBA{
				R: uint8(x * 255 / width),
				G: uint8(y * 255 / height),
				B: uint8((x + y) * 255 / (width + height)),
				A: 255,
			}
			img.Set(x, y, c)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	switch format {
	case "jpeg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	case "png":
		return png.Encode(file, img)
	default:
		return fmt.Errorf("unsupported image format: %s", format)
	}
}

func testBasicOperations(client proto.FileServiceClient) {
	files := []string{
		"test_data/small.jpg",
		"test_data/medium.png",
		"test_data/image with spaces.jpg",
		"test_data/изображение.jpg",
	}

	if err := createTestImage(files[0], "jpeg", 100, 100); err != nil {
		log.Fatalf("Failed to create test image: %v", err)
	}
	if err := createTestImage(files[1], "png", 800, 600); err != nil {
		log.Fatalf("Failed to create test image: %v", err)
	}
	if err := createTestImage(files[2], "jpeg", 200, 200); err != nil {
		log.Fatalf("Failed to create test image: %v", err)
	}
	if err := createTestImage(files[3], "jpeg", 300, 300); err != nil {
		log.Fatalf("Failed to create test image: %v", err)
	}

	// Тестируем загрузку
	for _, filename := range files {
		if err := uploadFile(client, filename); err != nil {
			log.Printf("Upload failed for %s: %v", filename, err)
			continue
		}
		log.Printf("Uploaded: %s", filename)
	}

	// Тестируем список файлов
	listFiles(client)

	// Тестируем скачивание
	for _, filename := range files {
		downloaded := filename + ".downloaded"
		if err := downloadAndVerify(client, filename, downloaded); err != nil {
			log.Printf("Download failed: %v", err)
			continue
		}
		os.Remove(downloaded)
	}
}

func testErrorHandling(client proto.FileServiceClient) {
	// Несуществующий файл при скачивании
	stream, err := client.DownloadFile(context.Background(), &proto.DownloadFileRequest{
		Filename: "non_existent_file_123",
	})
	if err == nil {
		_, err = stream.Recv()
		if err != nil {
			log.Printf("Correctly received error for non-existent file: %v", err)
		} else {
			log.Fatal("Expected error for non-existent file, got nil")
		}
	}

	// Пустое имя файла
	_, err = client.UploadFile(context.Background())
	if err == nil {
		_, err = stream.Recv()
		if err != nil {
			log.Printf("Correctly received error for empty filename: %v", err)
		} else {
			log.Fatal("Expected error for empty filename, got nil")
		}
	}
}

func testConcurrency(client proto.FileServiceClient) {
	var wg sync.WaitGroup
	start := time.Now()

	// Тест лимита ListFiles (100)
	for range 105 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.ListFiles(context.Background(), &proto.ListFilesRequest{})
			if err != nil {
				log.Printf("ListFiles error (expected for some): %v", err)
			}
		}()
	}

	// Тест лимита Upload (10)
	for i := range 15 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			filename := fmt.Sprintf("test_data/concurrent_%d.tmp", n)
			os.WriteFile(filename, []byte(fmt.Sprintf("content %d", n)), 0644)
			if err := uploadFile(client, filename); err != nil {
				log.Printf("Upload %d failed: %v", n, err)
			}
		}(i)
	}

	wg.Wait()
	log.Printf("Concurrency test completed in %v", time.Since(start))
}

func testLargeFiles(client proto.FileServiceClient) {
	largeFile := "test_data/large_file.bin"
	createLargeFile(largeFile, 50) // 50MB

	start := time.Now()
	if err := uploadFile(client, largeFile); err != nil {
		log.Fatalf("Large file upload failed: %v", err)
	}
	log.Printf("Large file (50MB) uploaded in %v", time.Since(start))

	// Скачивание и проверка
	if err := downloadAndVerify(client, largeFile, largeFile+".downloaded"); err != nil {
		log.Fatalf("Large file download failed: %v", err)
	}
	os.Remove(largeFile + ".downloaded")
}

func createLargeFile(path string, sizeMB int) {
	file, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Пишем рандомные данные
	if _, err := io.CopyN(file, rand.Reader, int64(sizeMB)<<20); err != nil {
		log.Fatal(err)
	}
}

func uploadFile(client proto.FileServiceClient, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	stream, err := client.UploadFile(context.Background())
	if err != nil {
		return fmt.Errorf("create upload stream: %w", err)
	}

	// Отправляем метаданные
	if err := stream.Send(&proto.UploadFileRequest{
		Data: &proto.UploadFileRequest_Metadata{
			Metadata: &proto.FileMetadata{Filename: filepath.Base(filename)},
		},
	}); err != nil {
		return fmt.Errorf("send metadata: %w", err)
	}

	// Отправляем содержимое файла
	buf := make([]byte, 1<<20) // 1MB
	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		if err := stream.Send(&proto.UploadFileRequest{
			Data: &proto.UploadFileRequest_Chunk{Chunk: buf[:n]},
		}); err != nil {
			return fmt.Errorf("send chunk: %w", err)
		}
	}

	// Получаем ответ
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("receive response: %w", err)
	}

	log.Printf("Upload completed: %s (size: %d)", resp.Filename, resp.Size)
	return nil
}

func downloadAndVerify(client proto.FileServiceClient, filename, saveTo string) error {
	stream, err := client.DownloadFile(context.Background(), &proto.DownloadFileRequest{
		Filename: filepath.Base(filename),
	})
	if err != nil {
		return fmt.Errorf("create download stream: %w", err)
	}

	output, err := os.Create(saveTo)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer output.Close()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("receive chunk: %w", err)
		}

		if chunk := resp.GetChunk(); chunk != nil {
			if _, err := output.Write(chunk); err != nil {
				return fmt.Errorf("write chunk: %w", err)
			}
		}
	}

	// Проверяем целостность
	if !verifyChecksum(filename, saveTo) {
		return fmt.Errorf("checksum mismatch for %s", filename)
	}

	log.Printf("Download verified: %s", filepath.Base(filename))
	return nil
}

func verifyChecksum(original, downloaded string) bool {
	origHash := fileHash(original)
	downHash := fileHash(downloaded)

	if origHash == "" || downHash == "" {
		return false
	}

	return origHash == downHash
}

func fileHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func listFiles(client proto.FileServiceClient) {
	resp, err := client.ListFiles(context.Background(), &proto.ListFilesRequest{})
	if err != nil {
		log.Printf("ListFiles failed: %v", err)
		return
	}

	log.Println("Files list:")
	for _, file := range resp.Files {
		log.Printf("- %s (created: %v)",
			file.Filename,
			file.CreatedAt.AsTime().Format(time.RFC3339))
	}
}
