PROTO_DIR = api/proto
PROTO_FILE = $(PROTO_DIR)/file_service.proto
PKG = github.com/keenoobi/$(PROJECT_NAME)
BIN_DIR = bin
CONFIG_DIR = config

GO = go
GO_BUILD = $(GO) build
GO_INSTALL = $(GO) install
GO_TEST = $(GO) test -v -race
GO_LINT = golangci-lint run

PROTOC = protoc
PROTOC_FLAGS = --go_out=. --go_opt=paths=source_relative \
               --go-grpc_out=. --go-grpc_opt=paths=source_relative

.PHONY: all generate build-server build-client build server client test clean deps
all: build

generate:
	$(PROTOC) $(PROTOC_FLAGS) $(PROTO_FILE)

build-server:
	$(GO_BUILD) -o $(BIN_DIR)/server ./cmd/server

build-client:
	$(GO_BUILD) -o $(BIN_DIR)/client ./cmd/client

build: generate build-server build-client

server: build-server
	$(BIN_DIR)/server

client: build-client
	$(BIN_DIR)/client

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down

test:
	$(GO_TEST) ./...

test-race:
	$(GO_TEST) -race ./...

coverage:
	$(GO_TEST) ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html && open coverage.html

clean:
	rm -rf $(BIN_DIR)
	rm -rf coverage*

clean-storage:
	rm -rf ./storage/*
	rm -rf ./test_data

deps:
	$(GO_INSTALL) google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO_INSTALL) google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	$(GO_INSTALL) github.com/golangci/golangci-lint/cmd/golangci-lint@latest