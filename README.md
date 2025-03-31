# gRPC File Manager Service

![Go](https://img.shields.io/badge/Go-1.20+-blue)
![gRPC](https://img.shields.io/badge/gRPC-1.50+-brightgreen)
![Coverage](https://img.shields.io/badge/Coverage-85%25-green)

Сервис для управления файлами по gRPC с поддержкой потоковой передачи и ограничением конкурентных запросов.

## Требования

- Go 1.20+
- protoc (Protocol Buffer compiler)

## Функциональность

1. Прием и сохранение бинарных файлов (изображений)
2. Просмотр списка файлов с метаданными
3. Скачивание файлов
4. Ограничение конкурентных подключений:
   - 10 одновременных операций Upload/Download
   - 100 одновременных запросов ListFiles

## Архитектура
```
.
├── api/proto                # Protobuf спецификация
├── cmd
│   ├── client               # gRPC клиент для тестирования
│   └── server               # gRPC сервер
├── config                   # Конфигурация
├── internal
│   ├── entity               # Бизнес-сущности
│   ├── middleware           # gRPC middleware
│   ├── repository           # Работа с файловой системой
│   ├── transport/grpc       # gRPC хендлеры
│   └── usecase              # Бизнес-логика
└── storage                  # Директория для хранения файлов
```

## Запуск

1. **Сгенерировать gRPC код**:
   ```bash
   make generate
   ```

2. **Запустить сервер**:
   ```bash
   make run-server
   ```
   *Сервер запустится на `localhost:50051`*

3. **Запустить тестового клиента**:
   ```bash
   make run-client
   ```
   *Выполнит тестовые сценарии:*
   - Базовые операции
   - Обработку ошибок
   - Проверку лимитов (20 конкурентный загрузок)
   - Работу с большими файлами

## Docker-развертывание

Сервер может быть развернут в Docker-контейнере:

1. Сборка образа:
```docker
docker build -t grpc-file-server .
```
2. Запуск контейнера:
```docker
docker run -d \
  -p 50051:50051 \
  -v ./storage:/storage \
  -v ./config:/app/config \
  grpc-file-server
```

## Запуск через Docker Compose

Можно просто использовать команды из Makefile:
1. Для запуска
```bash
make compose-up
```

2. Для остановки остановки и удаления данных:
```bash
make compose-down
```

3. Просмотр логов:
```docker
docker compose logs -f
```

## Конфигурация

`config/config.yaml`:
```yaml
server:
  port: ":50051"
  timeout: "10s"

limits:
  upload: 10    # Макс. одновременных загрузок/скачиваний
  list: 100     # Макс. одновременных запросов списка

storage:
  path: "./storage"  # Директория для файлов
```

## Особенности реализации

1. **Потоковая передача**:
   - Файлы передаются чанками по 1MB
   - Поддержка больших файлов (>50MB)

2. **Безопасность**:
   - Валидация имен файлов
   - Защита от path traversal
   - Обработка битых данных

3. **Надежность**:
   - Graceful shutdown
   - Recovery от паник
   - Атомарная запись файлов (через временный файл)

4. **Мониторинг**:
   - Логирование в JSON

## Тестирование

### Стратегия тестирования

1. **Unit-тесты**:
   - Покрытие всех слоев (transport, usecase, repository)
   - Mock-зависимости
   - Проверка обработки ошибок

2. **Интеграционные тесты**:
   - Фактическое взаимодействие с файловой системой
   - Проверка конкурентных операций

3. **Покрытие кода**:
   ```bash
   make coverage  # Генерирует HTML отчет
   ```
   Текущее покрытие: 85% (цель >90%)

### Запуск тестов

```bash
make test       # Запуск всех тестов
make test-race  # Тесты с проверкой гонок данных
make coverage   # Отчет о покрытии
```

## Примеры использования

### Загрузка файла (клиентская реализация):
```go
stream, _ := client.UploadFile(ctx)
stream.Send(&pb.UploadFileRequest{
    Data: &pb.UploadFileRequest_Metadata{
        Metadata: &pb.FileMetadata{Filename: "test.jpg"},
    },
})
// Отправка чанков...
```

### Получение списка:
```go
resp, _ := client.ListFiles(ctx, &pb.ListFilesRequest{})
for _, file := range resp.Files {
    fmt.Printf("%s (%.2f MB)\n", 
        file.Filename, 
        float64(file.Size)/(1024*1024))
}
```
## Технологический стек
- Go 1.20+
- gRPC 1.50+
- Protocol Buffers v3
- Docker
- Docker Compose