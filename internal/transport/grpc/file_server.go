package grpc

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/keenoobi/grpc-file-manager/api/proto"
	"github.com/keenoobi/grpc-file-manager/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fileServiceServer struct {
	proto.UnimplementedFileServiceServer
	fileUseCase usecase.FileUseCase
}

func NewFileServiceServer(fileUseCase usecase.FileUseCase) proto.FileServiceServer {
	return &fileServiceServer{fileUseCase: fileUseCase}
}

func (s *fileServiceServer) UploadFile(stream proto.FileService_UploadFileServer) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "cannot receive file info: %v", err)
	}

	metadata := req.GetMetadata()
	if metadata == nil {
		return status.Errorf(codes.InvalidArgument, "first message must contain metadata")
	}

	filename := metadata.GetFilename()
	if filename == "" {
		return status.Errorf(codes.InvalidArgument, "filename is required")
	}

	// Создаем буфер для накопления данных
	var buf bytes.Buffer

	// Читаем оставшиеся сообщения (чанки данных)
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive chunk: %v", err)
		}

		chunk := req.GetChunk()
		if chunk == nil {
			continue
		}

		if _, err := buf.Write(chunk); err != nil {
			return status.Errorf(codes.Internal, "cannot write chunk: %v", err)
		}
	}

	// Сохраняем файл
	file, err := s.fileUseCase.UploadFile(stream.Context(), filename, &buf)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot save file: %v", err)
	}

	// Отправляем ответ
	return stream.SendAndClose(&proto.UploadFileResponse{
		Filename:  file.Name,
		Size:      uint64(file.Size),
		CreatedAt: timestamppb.New(file.CreatedAt),
	})
}

func (s *fileServiceServer) DownloadFile(req *proto.DownloadFileRequest, stream proto.FileService_DownloadFileServer) error {
	file, reader, err := s.fileUseCase.DownloadFile(stream.Context(), req.GetFilename())
	if err != nil {
		if os.IsNotExist(err) {
			return status.Error(codes.NotFound, "file not found")
		}
		return status.Errorf(codes.Internal, "cannot read file: %v", err)
	}
	defer reader.Close()

	// Отправляем метаданные первым сообщением
	if err := stream.Send(&proto.DownloadFileResponse{
		Content: &proto.DownloadFileResponse_Metadata{
			Metadata: &proto.FileMetadata{
				Filename:  file.Name,
				Size:      uint64(file.Size),
				CreatedAt: timestamppb.New(file.CreatedAt),
			},
		},
	}); err != nil {
		return status.Errorf(codes.Internal, "cannot send metadata: %v", err)
	}

	buffer := make([]byte, 1024)
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "cannot read chunk: %v", err)
		}

		if err := stream.Send(&proto.DownloadFileResponse{
			Content: &proto.DownloadFileResponse_Chunk{
				Chunk: buffer[:n],
			},
		}); err != nil {
			return status.Errorf(codes.Internal, "cannot send chunk: %v", err)
		}
	}
	return nil
}

func (s *fileServiceServer) ListFiles(ctx context.Context, req *proto.ListFilesRequest) (*proto.ListFilesResponse, error) {
	files, err := s.fileUseCase.ListFiles(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot list files: %v", err)
	}

	response := &proto.ListFilesResponse{
		Files: make([]*proto.FileInfo, len(files)),
	}

	for i, file := range files {
		response.Files[i] = &proto.FileInfo{
			Filename:  file.Name,
			CreatedAt: timestamppb.New(file.CreatedAt),
			UpdatedAt: timestamppb.New(file.UpdatedAt),
		}
	}

	return response, nil
}
