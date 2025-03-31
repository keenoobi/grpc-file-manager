# Stage 1: Build
FROM golang:1.23-alpine AS builder
WORKDIR /app

RUN apk add --no-cache \
    protobuf \
    protobuf-dev \
    make

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd api/proto && \
    protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    file_service.proto && \
    cd ../.. && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /grpc-file-server ./cmd/server/

# Stage 2: Runtime
FROM alpine:3.18
WORKDIR /app

RUN apk add --no-cache ca-certificates && \
    mkdir -p /storage

COPY --from=builder /grpc-file-server /app/
COPY internal/config/config.yaml /app/internal/config/config.yaml

RUN chmod -R 777 /storage

EXPOSE 50051
VOLUME ["/storage"]
CMD ["./grpc-file-server"]