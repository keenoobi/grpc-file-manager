package main

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/keenoobi/grpc-file-manager/api/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := proto.NewFileServiceClient(conn)

	// // Тестируем загрузку файла
	// uploadFile(client, "large.jpg")

	// // Тестируем получение списка файлов
	// listFiles(client)

	// // Тестируем скачивание файла
	// downloadFile(client, "large.jpg", "downloaded_large.jpg")

	// 1. Тест пустого файла
	createEmptyFile("empty.txt")
	uploadFile(client, "empty.txt")
	downloadFile(client, "empty.txt", "downloaded_empty.txt")

	// 2. Тест большого файла (10MB)
	createBigFile("bigfile.bin", 10)
	uploadFile(client, "bigfile.bin")
	downloadFile(client, "bigfile.bin", "downloaded_big.bin")

	// 3. Тест несуществующего файла при скачивании
	downloadNonExistentFile(client)

	// 4. Тест повторной загрузки того же файла
	uploadFile(client, "test.jpg")
	uploadFile(client, "test.jpg")

	// 5. Тест спецсимволов в имени
	uploadFile(client, "file with spaces.txt")
	uploadFile(client, "кириллица.jpg")
}

func createEmptyFile(filename string) {
	os.WriteFile(filename, []byte{}, 0644)
}

func createBigFile(filename string, sizeMB int) {
	file, _ := os.Create(filename)
	defer file.Close()
	file.Seek(int64(sizeMB<<20)-1, 0)
	file.Write([]byte{0})
}

func downloadNonExistentFile(client proto.FileServiceClient) {
	_, err := client.DownloadFile(context.Background(), &proto.DownloadFileRequest{
		Filename: "non_existent.file",
	})
	log.Printf("Download non-existent file error: %v", err) // Должна быть ошибка
}

func uploadFile(client proto.FileServiceClient, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("could not open file: %v", err)
	}
	defer file.Close()

	stream, err := client.UploadFile(context.Background())
	if err != nil {
		log.Fatalf("could not upload file: %v", err)
	}

	// Отправляем метаданные
	err = stream.Send(&proto.UploadFileRequest{
		Data: &proto.UploadFileRequest_Metadata{
			Metadata: &proto.FileMetadata{Filename: filename},
		},
	})
	if err != nil {
		log.Fatalf("could not send metadata: %v", err)
	}

	// Отправляем содержимое файла
	buffer := make([]byte, 1024)
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("could not read chunk: %v", err)
		}

		err = stream.Send(&proto.UploadFileRequest{
			Data: &proto.UploadFileRequest_Chunk{Chunk: buffer[:n]},
		})
		if err != nil {
			log.Fatalf("could not send chunk: %v", err)
		}
	}

	response, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("could not receive response: %v", err)
	}

	log.Printf("File uploaded: %s, size: %d", response.GetFilename(), response.GetSize())
}

func listFiles(client proto.FileServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.ListFiles(ctx, &proto.ListFilesRequest{})
	if err != nil {
		log.Fatalf("could not list files: %v", err)
	}

	log.Println("Files list:")
	for _, file := range response.GetFiles() {
		log.Printf("- %s (created: %v)", file.GetFilename(), file.GetCreatedAt().AsTime())
	}
}

func downloadFile(client proto.FileServiceClient, filename string, saveTo string) {
	stream, err := client.DownloadFile(context.Background(), &proto.DownloadFileRequest{Filename: filename})
	if err != nil {
		log.Fatalf("could not download file: %v", err)
	}

	outputFile, err := os.Create(saveTo)
	if err != nil {
		log.Fatalf("could not create file: %v", err)
	}
	defer outputFile.Close()

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("could not receive chunk: %v", err)
		}

		if chunk := response.GetChunk(); chunk != nil {
			if _, err := outputFile.Write(chunk); err != nil {
				log.Fatalf("could not write chunk: %v", err)
			}
		}
	}

	log.Printf("File downloaded to: %s", saveTo)
}
