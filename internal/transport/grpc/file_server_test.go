package grpc

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/keenoobi/grpc-file-manager/api/proto"
	"github.com/keenoobi/grpc-file-manager/internal/entity"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockFileUseCase struct {
	mock.Mock
}

func (m *MockFileUseCase) UploadFile(ctx context.Context, filename string, data io.Reader) (*entity.File, error) {
	args := m.Called(ctx, filename, data)
	return args.Get(0).(*entity.File), args.Error(1)
}

func (m *MockFileUseCase) DownloadFile(ctx context.Context, filename string) (*entity.File, io.ReadCloser, error) {
	args := m.Called(ctx, filename)
	return args.Get(0).(*entity.File), args.Get(1).(io.ReadCloser), args.Error(2)
}

func (m *MockFileUseCase) ListFiles(ctx context.Context) ([]*entity.File, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entity.File), args.Error(1)
}

type mockUploadStream struct {
	proto.FileService_UploadFileServer
	ctx          context.Context
	reqs         []*proto.UploadFileRequest
	index        int
	lastResponse *proto.UploadFileResponse
	sendErr      error
}

func (m *mockUploadStream) Context() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func (m *mockUploadStream) Recv() (*proto.UploadFileRequest, error) {
	if m.index >= len(m.reqs) {
		return nil, io.EOF
	}
	req := m.reqs[m.index]
	m.index++
	return req, nil
}

func (m *mockUploadStream) SendAndClose(resp *proto.UploadFileResponse) error {
	m.lastResponse = resp
	return m.sendErr
}

func TestUploadFile_Success(t *testing.T) {
	mockUC := new(MockFileUseCase)
	server := NewFileServiceServer(mockUC)

	mockFile := &entity.File{
		Name:      "test.txt",
		Size:      4,
		CreatedAt: time.Now(),
	}

	mockUC.On("UploadFile", mock.Anything, "test.txt", mock.Anything).Return(mockFile, nil)

	mockStream := &mockUploadStream{
		reqs: []*proto.UploadFileRequest{
			{
				Data: &proto.UploadFileRequest_Metadata{
					Metadata: &proto.FileMetadata{Filename: "test.txt"},
				},
			},
			{
				Data: &proto.UploadFileRequest_Chunk{Chunk: []byte("data")},
			},
		},
	}

	err := server.UploadFile(mockStream)
	require.NoError(t, err)

	require.NotNil(t, mockStream.lastResponse)
	require.Equal(t, "test.txt", mockStream.lastResponse.Filename)
	require.Equal(t, uint64(4), mockStream.lastResponse.Size)

	mockUC.AssertCalled(t, "UploadFile", mock.Anything, "test.txt", mock.AnythingOfType("*bytes.Buffer"))

	args := mockUC.Calls[0].Arguments
	r := args.Get(2).(io.Reader)
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, []byte("data"), data)

	mockUC.AssertExpectations(t)
}
func TestDownloadFile_Success(t *testing.T) {
	mockUC := new(MockFileUseCase)
	server := NewFileServiceServer(mockUC)

	mockFile := &entity.File{
		Name:      "test.txt",
		Size:      4,
		CreatedAt: time.Now(),
	}
	mockReader := io.NopCloser(strings.NewReader("data"))
	mockUC.On("DownloadFile", mock.Anything, "test.txt").Return(mockFile, mockReader, nil)

	mockStream := &mockDownloadStream{}
	err := server.DownloadFile(&proto.DownloadFileRequest{Filename: "test.txt"}, mockStream)
	require.NoError(t, err)
	require.Len(t, mockStream.responses, 2)
	mockUC.AssertExpectations(t)
}

type mockDownloadStream struct {
	proto.FileService_DownloadFileServer
	responses []*proto.DownloadFileResponse
	sendErr   error
}

func (m *mockDownloadStream) Context() context.Context {
	return context.Background()
}

func (m *mockDownloadStream) Send(resp *proto.DownloadFileResponse) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.responses = append(m.responses, resp)
	return nil
}

func TestListFiles_Success(t *testing.T) {
	mockUC := new(MockFileUseCase)
	server := NewFileServiceServer(mockUC)

	mockFiles := []*entity.File{
		{Name: "file1.txt", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Name: "file2.txt", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	mockUC.On("ListFiles", mock.Anything).Return(mockFiles, nil)

	resp, err := server.ListFiles(context.Background(), &proto.ListFilesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Files, 2)
	mockUC.AssertExpectations(t)
}
